import {
  completePlaybackSession,
  createPlaybackSession,
  updatePlaybackProgress,
} from '../lib/api'

/*
 * Playback tracking engine (Phase 3.8).
 *
 * This module is deliberately NOT a React component or
 * hook: it is a single mutable engine owned by the
 * PlayerContext. React StrictMode double-invokes effects,
 * so tracking that lived inside an effect could create
 * duplicate playback sessions. The engine is created once
 * per provider and driven purely by events from the ONE
 * global Audio() element.
 *
 * Lifecycle:
 *
 * - trackChanged(): the player loaded a different track.
 *   Completes any previous session (user switched
 *   tracks).
 *
 * - playStarted(): the audio element started genuinely
 *   playing. Creates the session lazily (also covers
 *   resuming a restored-from-storage queue) and starts
 *   the progress heartbeat. Idempotent per track.
 *
 * - accumulate(): real listened time, credited only from
 *   timeupdate observations consistent with wall-clock
 *   time (seeking is never credited).
 *
 * - paused(): stops the heartbeat (session stays open;
 *   pause keeps the track and position).
 *
 * - finish(): completes the session (track end, stop,
 *   unload, purpose suspension).
 */

// Progress is sent on this cadence while audio plays.
const PROGRESS_INTERVAL_MS = 8_000

// Minimum listened time before the first progress ping,
// so quick skips do not generate requests.
const MIN_LISTENED_BEFORE_PING_MS = 3_000

export type PlaybackPurpose =
  | 'listener'
  | 'editor-preview'

type EngineOptions = {
  getToken: () => string | null
  /*
   * True when tracking must not run at all (anonymous
   * playback, identity still loading, or the current
   * playback is an editor preview).
   */
  isTrackingSuspended: () => boolean
}

export class PlaybackTrackingEngine {
  private readonly options: EngineOptions

  // -------------------------------------------------------------
  // Session state
  // -------------------------------------------------------------

  private activeSessionID: string | null =
    null

  private activeMusicID: number | null =
    null

  /*
   * The track the player has loaded, awaiting its first
   * `play` event (session creation is lazy).
   */
  private pendingMusicID: number | null =
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
    options: EngineOptions,
  ) {
    this.options = options
  }

  // -------------------------------------------------------------
  // Event API (called from PlayerContext)
  // -------------------------------------------------------------

  /*
   * The player loaded a (possibly different) track.
   *
   * Any session for a previous track is completed — the
   * user switched tracks. The new track's session is NOT
   * created here; it is created on the first genuine
   * `play` event, which also covers queues restored from
   * sessionStorage (restored paused) and autoplay-blocked
   * loads.
   */
  trackChanged(
    musicID: number,
    durationMS: number,
    positionMS: number,
  ) {
    if (
      this.activeMusicID !== null &&
      this.activeMusicID !== musicID
    ) {
      this.finishSessionInternal(
        'track-switch',
      )
    }

    this.resetProgress()

    this.pendingMusicID = musicID

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
   *
   * `current` describes the track the player holds right
   * now. It re-arms the engine when no session is pending
   * (e.g. tracking was suspended by an editor preview and
   * playback resumes afterwards).
   */
  playStarted(
    current?: {
      musicID: number
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
      this.pendingMusicID !==
        current.musicID
    ) {
      this.pendingMusicID =
        current.musicID

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
   * The audio element paused (user pause, buffering is
   * handled by timeupdate simply not firing).
   *
   * The session stays open — pause keeps the selected
   * track and position, and listening may resume.
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
   * by ListenedTimeAccumulator from timeupdate events).
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
      | 'track-end'
      | 'track-switch'
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
   * when tracking is suspended mid-track (editor preview
   * took over the player, or the user logged out).
   */
  abandon() {
    this.stopHeartbeat()

    this.activeSessionID = null
    this.activeMusicID = null
    this.pendingMusicID = null

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
    const musicID =
      this.pendingMusicID

    if (
      musicID === null ||
      this.createInFlight
    ) {
      return
    }

    const token =
      this.options.getToken()

    if (!token) {
      if (import.meta.env.DEV) {
        console.warn(
          '[playback-tracking] session not created: no auth token',
          {
            musicID,
          },
        )
      }

      return
    }

    if (import.meta.env.DEV) {
      console.debug(
        '[playback-tracking] creating session',
        {
          musicID,
          durationMS: this.durationMS,
          positionMS: this.positionMS,
        },
      )
    }

    this.createInFlight = true

    try {
      const session =
        await createPlaybackSession(
          {
            music_id: musicID,
            duration_ms: this.durationMS,
            position_ms: this.positionMS,
          },
          token,
        )

      // The track changed while the request was in
      // flight: the session is obsolete, close it.
      if (
        this.pendingMusicID !==
        musicID
      ) {
        this.completedSessions.add(
          session.id,
        )

        void completePlaybackSession(
          session.id,
          {
            duration_ms: this
              .durationMS,
            position_ms: this
              .positionMS,
            listened_ms: 0,
          },
          token,
        ).catch(() => {})

        return
      }

      this.activeSessionID =
        session.id

      this.activeMusicID = musicID

      if (import.meta.env.DEV) {
        console.debug(
          '[playback-tracking] session created',
          {
            musicID,
            sessionID: session.id,
          },
        )
      }
    } catch (error) {
      // Creating the session must never interrupt music
      // playback. In development, expose the failure so
      // tracking problems are diagnosable instead of
      // silently disappearing.
      if (import.meta.env.DEV) {
        console.error(
          '[playback-tracking] session creation failed',
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

    updatePlaybackProgress(
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
    this.activeMusicID = null
    this.pendingMusicID = null

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

    // Unload completions use fetch keepalive so the
    // request survives page teardown.
    const keepalive =
      reason === 'unload'

    completePlaybackSession(
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

/*
 * Real listening time accumulator.
 *
 * The browser fires `timeupdate` ~4 times/second while
 * audio genuinely plays. Between two timeupdate events
 * the position advances by roughly the real elapsed time.
 * When the user seeks, the position delta jumps far
 * beyond (or behind) the wall-clock delta, and that delta
 * is NOT credited.
 *
 * Pause, buffering and background tabs are handled
 * naturally: timeupdate stops firing while audio is not
 * advancing, and after a gap the position delta cannot
 * exceed the wall-clock delta.
 */
export class ListenedTimeAccumulator {
  private lastEventAt: number | null =
    null

  private lastPosition: number | null =
    null

  /*
   * Record one timeupdate observation and return the
   * genuinely listened milliseconds it contributes.
   */
  observe(
    positionSeconds: number,
  ): number {
    const now = Date.now()

    if (
      !Number.isFinite(
        positionSeconds,
      )
    ) {
      return 0
    }

    if (
      this.lastEventAt === null ||
      this.lastPosition === null
    ) {
      this.lastEventAt = now
      this.lastPosition =
        positionSeconds

      return 0
    }

    const wallClockDeltaMS =
      now - this.lastEventAt

    const positionDeltaMS =
      (positionSeconds -
        this.lastPosition) *
      1000

    this.lastEventAt = now
    this.lastPosition =
      positionSeconds

    if (positionDeltaMS <= 0) {
      // Seek backwards or duplicate event: no credit.
      return 0
    }

    // Tolerance covers timeupdate cadence jitter
    // (typically ~250ms with small drift).
    const toleranceMS = 400

    if (
      positionDeltaMS <=
      wallClockDeltaMS + toleranceMS
    ) {
      // Position advanced roughly with real time:
      // genuine listening.
      return positionDeltaMS
    }

    // A forward seek: the position jumped beyond wall
    // clock. The seek itself is a discontinuity, so no
    // listening time is credited for this window.
    return 0
  }

  reset() {
    this.lastEventAt = null
    this.lastPosition = null
  }
}

/*
 * Shared normalizer (also used by the podcast tracking
 * engine): non-finite values become 0, everything else is
 * rounded to a non-negative integer.
 */
export function nonNegativeInt(
  value: number,
): number {
  if (!Number.isFinite(value)) {
    return 0
  }

  return Math.max(
    0,
    Math.round(value),
  )
}

export function msFromSeconds(
  seconds: number,
): number {
  if (!Number.isFinite(seconds)) {
    return 0
  }

  return Math.max(
    0,
    Math.round(seconds * 1000),
  )
}
