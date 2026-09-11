import {
  completePodcastPlaybackSession,
  createPodcastPlaybackSession,
  updatePodcastPlaybackProgress,
} from '../lib/api'

import {
  msFromSeconds,
  nonNegativeInt,
} from './playbackTracking'

/*
 * Podcast playback tracking engine (Phase 4.1).
 *
 * A dedicated engine with the SAME event API as the music
 * PlaybackTrackingEngine, calling the dedicated podcast
 * playback endpoints. Podcast episode ids must never be
 * submitted to the music playback session API.
 *
 * Deliberately NOT merged into the music engine: the music
 * engine is proven in production and this file lets podcast
 * tracking evolve (resume, completion thresholds) without
 * any regression risk to music analytics. Both engines are
 * driven by events from the ONE global Audio element owned
 * by the PlayerContext — this module never creates an
 * Audio element.
 *
 * Lifecycle (identical semantics to the music engine):
 *
 * - trackChanged(): the player loaded a different episode.
 *   Completes any previous session.
 *
 * - playStarted(): genuine playback began; creates the
 *   session lazily and starts the progress heartbeat.
 *
 * - accumulate(): real listened time credited from
 *   timeupdate observations (seeking is never credited —
 *   the shared ListenedTimeAccumulator enforces that).
 *
 * - paused(): stops the heartbeat; the session stays open.
 *
 * - finish(): completes the session (ended, switch, stop,
 *   unload, purpose suspension).
 */

const PROGRESS_INTERVAL_MS = 8_000

const MIN_LISTENED_BEFORE_PING_MS = 3_000

type PodcastEngineOptions = {
  getToken: () => string | null

  /*
   * True when tracking must not run at all (anonymous
   * playback, identity still loading, or the current
   * playback is an editor preview).
   */
  isTrackingSuspended: () => boolean
}

export class PodcastPlaybackTrackingEngine {
  private readonly options: PodcastEngineOptions

  // -------------------------------------------------------------
  // Session state
  // -------------------------------------------------------------

  private activeSessionID: string | null =
    null

  private activeEpisodeID: number | null =
    null

  /*
   * The episode the player has loaded, awaiting its first
   * `play` event (session creation is lazy).
   */
  private pendingEpisodeID: number | null =
    null

  private createInFlight = false

  // -------------------------------------------------------------
  // Progress state
  // -------------------------------------------------------------

  private listenedMS = 0

  private durationMS = 0

  private positionMS = 0

  private timer: ReturnType<
    typeof setInterval
  > | null = null

  private progressInFlight = false

  private completedSessions =
    new Set<string>()

  constructor(
    options: PodcastEngineOptions,
  ) {
    this.options = options
  }

  // -------------------------------------------------------------
  // Event API (called from PlayerContext)
  // -------------------------------------------------------------

  /*
   * The player loaded a (possibly different) episode.
   *
   * Any session for a previous episode is completed — the
   * user switched. The new episode's session is created on
   * the first genuine `play` event.
   */
  trackChanged(
    episodeID: number,
    durationMS: number,
    positionMS: number,
  ) {
    if (
      this.activeEpisodeID !== null &&
      this.activeEpisodeID !== episodeID
    ) {
      this.finishSessionInternal(
        'episode-switch',
      )
    }

    this.resetProgress()

    this.pendingEpisodeID = episodeID

    this.durationMS = nonNegativeInt(
      durationMS,
    )

    this.positionMS = nonNegativeInt(
      positionMS,
    )
  }

  /*
   * The audio element started genuinely playing.
   *
   * Resuming after pause keeps the existing session.
   */
  playStarted(
    current?: {
      episodeID: number
      durationMS: number
      positionMS: number
    },
  ) {
    if (
      this.options.isTrackingSuspended()
    ) {
      // Editor preview or anonymous playback: make sure
      // no session is running.
      this.abandon()
      return
    }

    this.startHeartbeat()

    if (this.activeSessionID) {
      return
    }

    if (
      current &&
      this.pendingEpisodeID !==
        current.episodeID
    ) {
      this.pendingEpisodeID =
        current.episodeID

      this.durationMS = nonNegativeInt(
        current.durationMS,
      )

      this.positionMS = nonNegativeInt(
        current.positionMS,
      )
    }

    this.createSessionIfNeeded()
  }

  /*
   * The audio element paused. The session stays open —
   * pause keeps the episode and position.
   */
  paused() {
    this.stopHeartbeat()
  }

  onDurationChange(
    durationMS: number,
  ) {
    if (!Number.isFinite(durationMS)) {
      return
    }

    this.durationMS = nonNegativeInt(
      durationMS,
    )
  }

  /*
   * Credit genuinely listened milliseconds (accumulated
   * by the shared ListenedTimeAccumulator).
   */
  accumulate(
    listenedDeltaMS: number,
    positionSeconds: number,
  ) {
    if (listenedDeltaMS > 0) {
      this.listenedMS += listenedDeltaMS
    }

    if (
      Number.isFinite(positionSeconds)
    ) {
      this.positionMS = msFromSeconds(
        positionSeconds,
      )
    }
  }

  /*
   * Complete the current session.
   */
  finishSession(
    reason:
      | 'episode-end'
      | 'episode-switch'
      | 'stopped'
      | 'unload',
  ) {
    this.stopHeartbeat()

    this.finishSessionInternal(
      reason,
    )
  }

  /*
   * Drop all state without contacting the server. Used
   * when tracking is suspended (editor preview took over
   * the player, or the user logged out).
   */
  abandon() {
    this.stopHeartbeat()

    this.activeSessionID = null
    this.activeEpisodeID = null
    this.pendingEpisodeID = null

    this.resetProgress()
  }

  // -------------------------------------------------------------
  // Internals
  // -------------------------------------------------------------

  private resetProgress() {
    this.listenedMS = 0
    this.positionMS = 0
    this.durationMS = 0
    this.progressInFlight = false
  }

  private async createSessionIfNeeded() {
    const episodeID =
      this.pendingEpisodeID

    if (
      episodeID === null ||
      this.createInFlight
    ) {
      return
    }

    const token =
      this.options.getToken()

    if (!token) {
      if (import.meta.env.DEV) {
        console.warn(
          '[podcast-tracking] session not created: no auth token',
          {
            episodeID,
          },
        )
      }

      return
    }

    if (import.meta.env.DEV) {
      console.debug(
        '[podcast-tracking] creating session',
        {
          episodeID,
          durationMS: this.durationMS,
          positionMS: this.positionMS,
        },
      )
    }

    this.createInFlight = true

    try {
      const session =
        await createPodcastPlaybackSession(
          {
            episode_id: episodeID,
            duration_ms: this.durationMS,
            position_ms: this.positionMS,
          },
          token,
        )

      // The episode changed while the request was in
      // flight: the session is obsolete, close it.
      //
      // Zeroes are sent deliberately: this.durationMS/
      // positionMS now describe the NEW episode, and
      // reporting them against the old episode's session
      // could corrupt that episode's history (e.g. a
      // near-end resume position marking the old episode
      // completed). The session never accumulated any
      // accepted progress, so there is nothing to report.
      if (
        this.pendingEpisodeID !==
        episodeID
      ) {
        this.completedSessions.add(
          session.id,
        )

        void completePodcastPlaybackSession(
          session.id,
          {
            duration_ms: 0,
            position_ms: 0,
            listened_ms: 0,
          },
          token,
        ).catch(() => {})

        return
      }

      this.activeSessionID =
        session.id

      this.activeEpisodeID = episodeID

      if (import.meta.env.DEV) {
        console.debug(
          '[podcast-tracking] session created',
          {
            episodeID,
            sessionID: session.id,
          },
        )
      }
    } catch (error) {
      // Session creation must never interrupt playback.
      if (import.meta.env.DEV) {
        console.error(
          '[podcast-tracking] session creation failed',
          error,
        )
      }

      this.activeSessionID = null
    } finally {
      this.createInFlight = false
    }
  }

  private startHeartbeat() {
    if (this.timer) {
      return
    }

    this.timer = setInterval(
      () => {
        this.sendProgress()
      },
      PROGRESS_INTERVAL_MS,
    )
  }

  private stopHeartbeat() {
    if (this.timer) {
      clearInterval(this.timer)
      this.timer = null
    }
  }

  private sendProgress() {
    if (
      !this.activeSessionID ||
      this.progressInFlight
    ) {
      return
    }

    if (
      this.listenedMS <
      MIN_LISTENED_BEFORE_PING_MS
    ) {
      return
    }

    const token =
      this.options.getToken()

    if (!token) {
      return
    }

    this.progressInFlight = true

    const {
      durationMS,
      positionMS,
      listenedMS,
    } = this

    updatePodcastPlaybackProgress(
      this.activeSessionID,
      {
        duration_ms: nonNegativeInt(
          durationMS,
        ),
        position_ms: nonNegativeInt(
          positionMS,
        ),
        listened_ms: nonNegativeInt(
          listenedMS,
        ),
      },
      token,
    )
      .catch(() => {
        // Progress pings are best-effort. A completed
        // session (409) simply stops being updated; the
        // engine stops tracking it.
      })
      .finally(() => {
        this.progressInFlight = false
      })
  }

  private finishSessionInternal(
    reason: string,
  ) {
    const sessionID =
      this.activeSessionID

    const token =
      this.options.getToken()

    this.activeSessionID = null
    this.activeEpisodeID = null
    this.pendingEpisodeID = null

    if (!sessionID) {
      return
    }

    if (!token) {
      return
    }

    // Completion is idempotent server-side; the client
    // set also avoids duplicate calls for one session.
    if (
      this.completedSessions.has(
        sessionID,
      )
    ) {
      return
    }

    this.completedSessions.add(
      sessionID,
    )

    const {
      durationMS,
      positionMS,
      listenedMS,
    } = this

    const keepalive =
      reason === 'unload'

    completePodcastPlaybackSession(
      sessionID,
      {
        duration_ms: nonNegativeInt(
          durationMS,
        ),
        position_ms: nonNegativeInt(
          positionMS,
        ),
        listened_ms: nonNegativeInt(
          listenedMS,
        ),
      },
      token,
      keepalive,
    ).catch(() => {
      // Best-effort: the backend treats sessions with
      // no activity as abandoned.
    })
  }
}
