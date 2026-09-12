import {
  useCallback,
  useEffect,
  useRef,
  useState,
} from 'react'

import { usePlayer } from '../../context/PlayerContext'

import {
  cancelPodcastLive,
  endPodcastLive,
  getOwnedPodcastLive,
  getPodcastLiveHostToken,
  publishPodcastLiveRecording,
  schedulePodcastLiveEpisode,
  startPodcastLive,
} from '../../lib/api'

import {
  LiveAudioRoom,
  type LiveRoomState,
} from '../../lib/liveAudio'

import type {
  PodcastEpisode,
} from '../../types/podcast'

import type {
  PodcastLiveDetails,
} from '../../types/podcastLive'

type LiveControlPanelProps = {
  episode: PodcastEpisode

  token: string

  onRefresh: () => Promise<void>
}

const REFRESH_INTERVAL_MS = 30_000

function formatCountdown(
  target: string,
  now: number,
) {
  const date = new Date(target)

  if (
    Number.isNaN(
      date.getTime(),
    )
  ) {
    return ''
  }

  const diff =
    date.getTime() - now

  if (diff <= 0) {
    return 'starting now'
  }

  const totalMinutes = Math.floor(
    diff / 60_000,
  )

  const days = Math.floor(
    totalMinutes / 1440,
  )

  const hours = Math.floor(
    (totalMinutes % 1440) / 60,
  )

  const minutes =
    totalMinutes % 60

  if (days > 0) {
    return `${days}d ${hours}h`
  }

  if (hours > 0) {
    return `${hours}h ${minutes}m`
  }

  return `${minutes}m`
}

// PodcastLiveControlPanel is the host-side live control
// surface for one episode: schedule, start, go live with the
// microphone, mute, end, and cancel.
//
// All permissions come from the backend-minted host token;
// this component never sees provider credentials and can
// never mint a listener's publish rights.
function PodcastLiveControlPanel({
  episode,
  token,
  onRefresh,
}: LiveControlPanelProps) {
  const {
    stopPlayback,
  } = usePlayer()

  const [
    details,
    setDetails,
  ] = useState<
    PodcastLiveDetails | null
  >(null)

  const [
    liveChecked,
    setLiveChecked,
  ] = useState(false)

  const [message, setMessage] =
    useState('')

  const [error, setError] =
    useState('')

  const [busy, setBusy] =
    useState(false)

  const [scheduledAt, setScheduledAt] =
    useState('')

  const [now, setNow] = useState(
    () => Date.now(),
  )

  const [
    roomState,
    setRoomState,
  ] = useState<LiveRoomState>('idle')

  const [micEnabled, setMicEnabled] =
    useState(false)

  const [listenerCount, setListenerCount] =
    useState(0)

  const audioRef =
    useRef<HTMLAudioElement | null>(
      null,
    )

  const roomRef =
    useRef<LiveAudioRoom | null>(
      null,
    )

  const unsubscribeStateRef = useRef<
    (() => void) | null
  >(null)

  const unsubscribeParticipantCountRef = useRef<
    (() => void) | null
  >(null)

  const session =
    details?.live_session

  const loadLive = useCallback(
    async () => {
      try {
        const response =
          await getOwnedPodcastLive(
            episode.id,
            token,
          )

        setDetails(response)
      } catch {
        // No live session for this episode (yet).
        setDetails(null)
      } finally {
        setLiveChecked(true)
      }
    },
    [
      episode.id,
      token,
    ],
  )

  useEffect(
    () => {
      void loadLive()
    },
    [loadLive],
  )

  // Countdown ticker while a broadcast is scheduled.
  useEffect(
    () => {
      if (
        session?.status !==
        'SCHEDULED'
      ) {
        return
      }

      const interval = setInterval(
        () => {
          setNow(Date.now())
        },
        30_000,
      )

      return () => {
        clearInterval(interval)
      }
    },
    [session?.status],
  )

  // Poll while scheduled or live so the host sees remote
  // state changes without refreshing.
  useEffect(
    () => {
      if (
        session?.status !==
          'SCHEDULED' &&
        session?.status !== 'LIVE'
      ) {
        return
      }

      const interval = setInterval(
        () => {
          void loadLive()
        },
        REFRESH_INTERVAL_MS,
      )

      return () => {
        clearInterval(interval)
      }
    },
    [
      session?.status,
      loadLive,
    ],
  )

  // Leave the room when the panel unmounts.
  useEffect(
    () => () => {
      unsubscribeStateRef
        .current?.()

      unsubscribeParticipantCountRef
        .current?.()

      roomRef.current?.disconnect()

      roomRef.current = null
    },
    [],
  )

  const runLiveAction = useCallback(
    async (
      action: () => Promise<PodcastLiveDetails>,
      successMessage: string,
    ) => {
      if (busy) {
        return
      }

      setBusy(true)
      setError('')
      setMessage('')

      try {
        const response =
          await action()

        setDetails(response)

        await onRefresh()

        setMessage(successMessage)
      } catch (err) {
        setError(
          err instanceof Error
            ? err.message
            : 'Live action failed.',
        )
      } finally {
        setBusy(false)
      }
    },
    [
      busy,
      onRefresh,
    ],
  )

  const connectHost = useCallback(
    async (
      sessionID: string,
    ) => {
      if (!audioRef.current) {
        return
      }

      try {
        // Host token: publish + subscribe, short-lived,
        // identity derived server-side.
        const accessToken =
          await getPodcastLiveHostToken(
            sessionID,
            token,
          )

        const room =
          new LiveAudioRoom(
            audioRef.current,
        )

        unsubscribeStateRef.current =
          room.onStateChange(
            setRoomState,
          )

        unsubscribeParticipantCountRef.current =
          room.onParticipantCountChange(
            setListenerCount,
          )

        roomRef.current = room

        // One audio surface at a time: going live stops
        // music/podcast replay.
        stopPlayback()

        await room.connect(
          accessToken.connect_url,
          accessToken.token,
          true,
        )

        setMicEnabled(true)
      } catch (err) {
        unsubscribeStateRef
          .current?.()

        unsubscribeStateRef.current =
          null

        unsubscribeParticipantCountRef
          .current?.()

        unsubscribeParticipantCountRef.current =
          null

        roomRef.current?.disconnect()

        roomRef.current = null

        setRoomState('idle')
        setMicEnabled(false)

        setError(
          err instanceof Error
            ? err.message
            : 'Unable to connect to the live room.',
        )
      }
    },
    [
      token,
      stopPlayback,
    ],
  )

  const handleSchedule = useCallback(
    async () => {
      if (!scheduledAt) {
        setError(
          'Pick a start time first.',
        )

        return
      }

      const scheduledDate =
        new Date(scheduledAt)

      if (
        Number.isNaN(
          scheduledDate.getTime(),
        )
      ) {
        setError(
          'Invalid start time.',
        )

        return
      }

      await runLiveAction(
        () =>
          schedulePodcastLiveEpisode(
            episode.id,
            scheduledDate.toISOString(),
            token,
          ),
        'Live broadcast scheduled.',
      )

      setScheduledAt('')
    },
    [
      scheduledAt,
      runLiveAction,
      episode.id,
      token,
    ],
  )

  const handleStart = useCallback(
    async () => {
      if (!session) {
        return
      }

      await runLiveAction(
        () =>
          startPodcastLive(
            episode.id,
            token,
          ),
        'Broadcast is live.',
      )

      if (session.id) {
        await connectHost(
          session.id,
        )
      }
    },
    [
      session,
      runLiveAction,
      episode.id,
      token,
      connectHost,
    ],
  )

  const handleEnd = useCallback(
    async () => {
      roomRef.current?.disconnect()

      roomRef.current = null

      setRoomState('idle')
      setMicEnabled(false)
      setListenerCount(0)

      await runLiveAction(
        () =>
          endPodcastLive(
            episode.id,
            token,
          ),
        'Broadcast ended.',
      )
    },
    [
      runLiveAction,
      episode.id,
      token,
    ],
  )

  const handleCancel = useCallback(
    async () => {
      await runLiveAction(
        () =>
          cancelPodcastLive(
            episode.id,
            token,
          ),
        'Live broadcast cancelled. The episode is back to draft.',
      )
    },
    [
      runLiveAction,
      episode.id,
      token,
    ],
  )

  const handlePublishRecording =
    useCallback(
      async () => {
        if (!session) {
          return
        }

        // Owner confirmation: publishing is irreversible and
        // makes the recording publicly streamable.
        const confirmed = window.confirm(
          'Publish this recording as the episode audio? It will be publicly streamable like any published episode.',
        )

        if (!confirmed) {
          return
        }

        await runLiveAction(
          () =>
            publishPodcastLiveRecording(
              session.id,
              token,
            ),
          'Recording published. The episode is now on demand.',
        )
      },
      [
        session,
        runLiveAction,
        token,
      ],
    )

  const toggleMic = useCallback(
    async () => {
      const room =
        roomRef.current

      if (!room) {
        return
      }

      try {
        const next =
          !micEnabled

        await room.setMicrophone(
          next,
        )

        setMicEnabled(next)
      } catch (err) {
        setError(
          err instanceof Error
            ? err.message
            : 'Unable to toggle the microphone.',
        )
      }
    },
    [micEnabled],
  )

  // Only draft episodes (no active live session) can be
  // scheduled; live statuses change exclusively through this
  // panel's live endpoints.
  const canSchedule =
    episode.status === 'DRAFT' &&
    (!session ||
      session.status ===
        'CANCELLED')

  const isConnected =
    roomState === 'connected' ||
    roomState === 'reconnecting'

  return (
    <div className="podcast-live-control">
      {/* Persistent audio surface: the host connection may
          start before the LIVE status renders. Hidden unless
          the session is live. */}
      <audio
        ref={audioRef}
        controls
        hidden={
          session?.status !== 'LIVE'
        }
      />

      <p className="podcast-live-control-heading">
        Live broadcast
      </p>

      {session &&
      session.status !==
        'CANCELLED' ? (
        <div className="podcast-live-control-status">
          <span
            className={`live-status-badge live-status-${session.status.toLowerCase()}`}
          >
            {session.status}
          </span>

          {session.status ===
          'SCHEDULED' ? (
            <span>
              Starts{' '}
              {new Date(
                session.scheduled_start_at,
              ).toLocaleString()}{' '}
              (
              {formatCountdown(
                session.scheduled_start_at,
                now,
              )}
              )
            </span>
          ) : null}

          {session.status ===
          'LIVE' ? (
            <span>
              Live since{' '}
              {session.started_at
                ? new Date(
                    session.started_at,
                  ).toLocaleTimeString()
                : 'now'}
            </span>
          ) : null}

          {session.status ===
          'ENDED' ? (
            <span>
              Ended{' '}
              {session.ended_at
                ? new Date(
                    session.ended_at,
                  ).toLocaleString()
                : ''}
            </span>
          ) : null}
        </div>
      ) : null}

      {canSchedule && liveChecked ? (
        <div className="podcast-live-schedule-form">
          <label>
            <span>
              Scheduled start
            </span>

            <input
              type="datetime-local"
              value={
                scheduledAt
              }
              onChange={(
                event,
              ) => {
                setScheduledAt(
                  event
                    .target
                    .value,
                )
              }}
              disabled={
                busy
              }
            />
          </label>

          <button
            type="button"
            className="release-secondary-button"
            disabled={busy}
            onClick={() => {
              void handleSchedule()
            }}
          >
            {busy
              ? 'Scheduling...'
              : 'Schedule Live Broadcast'}
          </button>
        </div>
      ) : null}

      {session?.status ===
      'SCHEDULED' ? (
        <div className="podcast-live-control-actions">
          <button
            type="button"
            className="release-publish-button"
            disabled={busy}
            onClick={() => {
              void handleStart()
            }}
          >
            Start Live
          </button>

          <button
            type="button"
            className="release-secondary-button"
            disabled={busy}
            onClick={() => {
              void handleCancel()
            }}
          >
            Cancel Broadcast
          </button>
        </div>
      ) : null}

      {session?.status ===
      'LIVE' ? (
        <div className="podcast-live-control-actions">
          {isConnected ? (
            <>
              <button
                type="button"
                className="release-secondary-button"
                onClick={() => {
                  void toggleMic()
                }}
              >
                {micEnabled
                  ? 'Mute Microphone'
                  : 'Unmute Microphone'}
              </button>

              <span className="podcast-live-connection">
                {roomState ===
                'connected'
                  ? 'On air'
                  : 'Reconnecting…'}
              </span>

              <span className="podcast-live-connection">
                {listenerCount === 1
                  ? '1 listener'
                  : `${listenerCount} listeners`}
              </span>
            </>
          ) : (
            <button
              type="button"
              className="release-secondary-button"
              disabled={busy}
              onClick={() => {
                void connectHost(
                  session.id,
                )
              }}
            >
              Connect Microphone
            </button>
          )}

          <button
            type="button"
            className="release-delete-button"
            disabled={busy}
            onClick={() => {
              void handleEnd()
            }}
          >
            End Live
          </button>
        </div>
      ) : null}

      {session?.status ===
      'ENDED' ? (
        <div className="podcast-live-control-actions">
          {session.recording_status ===
          'READY' ? (
            episode.status ===
            'ENDED' ? (
              <button
                type="button"
                className="release-publish-button"
                disabled={busy}
                onClick={() => {
                  void handlePublishRecording()
                }}
              >
                Publish Recording
              </button>
            ) : (
              <span className="podcast-live-message">
                Recording published.
              </span>
            )
          ) : (
            <span className="podcast-live-hint">
              Recording:{' '}
              {
                session.recording_status
              }
            </span>
          )}
        </div>
      ) : null}

      {message ? (
        <p className="podcast-live-message">
          {message}
        </p>
      ) : null}

      {error ? (
        <p className="error-text">
          {error}
        </p>
      ) : null}
    </div>
  )
}

export default PodcastLiveControlPanel
