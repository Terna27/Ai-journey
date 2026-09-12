import {
  useCallback,
  useEffect,
  useRef,
  useState,
} from 'react'

import { Link, useParams } from 'react-router-dom'

import { useAuth } from '../context/AuthContext'
import { usePlayer } from '../context/PlayerContext'

import {
  getPodcastLive,
  getPodcastLiveListenerToken,
} from '../lib/api'

import {
  LiveAudioRoom,
  type LiveRoomState,
} from '../lib/liveAudio'

import type {
  PodcastLiveBroadcast,
} from '../types/podcastLive'

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

function LiveRoomPage() {
  const { id } = useParams()

  const {
    isAuthenticated,
    isLoadingIdentity,
    token,
  } = useAuth()

  const {
    stopPlayback,
  } = usePlayer()

  const [
    broadcast,
    setBroadcast,
  ] = useState<
    PodcastLiveBroadcast | null
  >(null)

  const [isLoading, setIsLoading] =
    useState(true)

  const [error, setError] =
    useState('')

  const [joinError, setJoinError] =
    useState('')

  const [isJoining, setIsJoining] =
    useState(false)

  const [hasJoined, setHasJoined] =
    useState(false)

  const [roomState, setRoomState] =
    useState<LiveRoomState>('idle')

  const [now, setNow] = useState(
    () => Date.now(),
  )

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

  const sessionID = id ?? ''

  const liveSession =
    broadcast?.live_session

  const isJoinable =
    liveSession?.status === 'LIVE' ||
    liveSession?.status === 'SCHEDULED'

  const loadBroadcast =
    useCallback(
      async () => {
        try {
          setError('')

          const response =
            await getPodcastLive(
              sessionID,
            )

          setBroadcast(response)
        } catch (err) {
          setError(
            err instanceof Error
              ? err.message
              : 'Failed to load live broadcast',
          )
        } finally {
          setIsLoading(false)
        }
      },
      [sessionID],
    )

  useEffect(
    () => {
      if (!sessionID) {
        setIsLoading(false)
        setError(
          'Invalid live session ID',
        )

        return
      }

      setIsLoading(true)

      void loadBroadcast()
    },
    [
      sessionID,
      loadBroadcast,
    ],
  )

  // Poll the broadcast state so listeners see the
  // SCHEDULED -> LIVE -> ENDED transitions without manual
  // refreshes.
  useEffect(
    () => {
      if (!sessionID) {
        return
      }

      const interval = setInterval(
        () => {
          setNow(Date.now())

          if (!isJoining && !hasJoined) {
            void loadBroadcast()
          }
        },
        REFRESH_INTERVAL_MS,
      )

      return () => {
        clearInterval(interval)
      }
    },
    [
      sessionID,
      isJoining,
      hasJoined,
      loadBroadcast,
    ],
  )

  // Countdown ticker.
  useEffect(
    () => {
      if (
        liveSession?.status !==
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
    [liveSession?.status],
  )

  useEffect(
    () => () => {
      unsubscribeStateRef
        .current?.()

      roomRef.current?.disconnect()

      roomRef.current = null
    },
    [],
  )

  const handleJoin = useCallback(
    async () => {
      if (
        !audioRef.current ||
        !sessionID
      ) {
        return
      }

      setIsJoining(true)
      setJoinError('')

      try {
        // The identity and permissions come from the
        // backend token, never from this client. Anonymous
        // listeners are welcome; the token is only sent to
        // keep a signed-in listener's identity stable.
        const accessToken =
          await getPodcastLiveListenerToken(
            sessionID,
            isAuthenticated
              ? token ?? undefined
              : undefined,
          )

        const room =
          new LiveAudioRoom(
            audioRef.current,
        )

        // State updates keep flowing for the whole
        // connection (reconnects, remote disconnects).
        unsubscribeStateRef.current =
          room.onStateChange(
            setRoomState,
          )

        roomRef.current = room

        // One audio surface at a time: live audio replaces
        // any music/podcast replay currently playing.
        stopPlayback()

        // Listeners NEVER publish. The subscribe-only
        // token enforces this server-side too.
        await room.connect(
          accessToken.connect_url,
          accessToken.token,
          false,
        )

        setHasJoined(true)

        await audioRef.current
          .play()
          .catch(() => {
            // Autoplay may be blocked until the user
            // interacts; the element stays ready.
          })

        await loadBroadcast()
      } catch (err) {
        unsubscribeStateRef
          .current?.()

        unsubscribeStateRef.current =
          null

        roomRef.current?.disconnect()

        roomRef.current = null

        setRoomState('idle')
        setHasJoined(false)

        setJoinError(
          err instanceof Error
            ? err.message
            : 'Failed to join the live broadcast',
        )
      } finally {
        setIsJoining(false)
      }
    },
    [
      sessionID,
      isAuthenticated,
      token,
      stopPlayback,
      loadBroadcast,
    ],
  )

  const handleLeave = useCallback(
    () => {
      unsubscribeStateRef
        .current?.()

      unsubscribeStateRef.current =
        null

      roomRef.current?.disconnect()

      roomRef.current = null

      setRoomState('idle')
      setHasJoined(false)
    },
    [],
  )

  const episode =
    broadcast?.episode

  const podcast =
    broadcast?.podcast

  const statusBadge =
    liveSession?.status === 'LIVE'
      ? 'LIVE'
      : liveSession?.status ===
          'ENDED'
        ? 'ENDED'
        : 'UPCOMING'

  return (
    <div className="live-room-page">
      {/* Persistent audio surface: it must exist BEFORE the
          join starts, but stays hidden until connected. */}
      <audio
        ref={audioRef}
        controls
        autoPlay
        hidden={!hasJoined}
      />

      <header className="page-header">
        <p className="eyebrow">
          Live audio
        </p>

        <h1>Live Room</h1>
      </header>

      {isLoading ? (
        <p className="status-text">
          Loading live broadcast…
        </p>
      ) : error ? (
        <section className="content-panel">
          <p className="error-text">
            {error}
          </p>

          <p>
            <Link to="/podcasts">
              Browse podcasts
            </Link>
          </p>
        </section>
      ) : broadcast &&
        episode &&
        podcast ? (
        <section className="content-panel live-room-panel">
          <div className="live-room-header">
            <div className="podcast-card-artwork live-room-artwork">
              {episode.artwork_url ? (
                <img
                  src={
                    episode.artwork_url
                  }
                  alt=""
                />
              ) : (
                <div className="podcast-artwork-fallback">
                  {
                    podcast.title
                  }
                </div>
              )}
            </div>

            <div className="live-room-info">
              <p className="live-room-podcast">
                <Link
                  to={`/podcasts/${podcast.slug}`}
                >
                  {podcast.title}
                </Link>
              </p>

              <h2>
                {episode.title}
              </h2>

              <p className="live-room-status">
                <span
                  className={`live-status-badge live-status-${statusBadge.toLowerCase()}`}
                >
                  {statusBadge}
                </span>

                {episode.is_explicit ? (
                  <span className="podcast-explicit-badge">
                    Explicit
                  </span>
                ) : null}
              </p>

              {liveSession?.status ===
              'SCHEDULED' ? (
                <p className="live-room-schedule">
                  Starts{' '}
                  {new Date(
                    liveSession.scheduled_start_at,
                  ).toLocaleString()}{' '}
                  (
                  {formatCountdown(
                    liveSession.scheduled_start_at,
                    now,
                  )}
                  )
                </p>
              ) : null}

              {liveSession?.status ===
              'ENDED' ? (
                <p className="live-room-schedule">
                  This broadcast has
                  ended.
                </p>
              ) : null}

              {episode.description ? (
                <p className="live-room-description">
                  {
                    episode.description
                  }
                </p>
              ) : null}
            </div>
          </div>

          {isJoinable ? (
            <div className="live-room-controls">
              {hasJoined ? (
                <>
                  <p className="live-room-connection">
                    {roomState ===
                    'connected'
                      ? 'Listening live'
                      : roomState ===
                          'reconnecting'
                        ? 'Reconnecting…'
                        : 'Disconnected'}
                  </p>

                  <button
                    type="button"
                    onClick={
                      handleLeave
                    }
                  >
                    Leave live room
                  </button>
                </>
              ) : (
                <>
                  <button
                    type="button"
                    onClick={() => {
                      void handleJoin()
                    }}
                    disabled={
                      isJoining ||
                      isLoadingIdentity
                    }
                  >
                    {isJoining
                      ? 'Joining…'
                      : liveSession?.status ===
                          'LIVE'
                        ? 'Join live broadcast'
                        : 'Enter live room'}
                  </button>

                  <p className="live-room-hint">
                    Listening is
                    anonymous-friendly.
                    Your microphone
                    is never used.
                  </p>
                </>
              )}

              {joinError ? (
                <p className="error-text">
                  {joinError}
                </p>
              ) : null}
            </div>
          ) : (
            <div className="live-room-controls">
              <p className="live-room-hint">
                This broadcast is
                no longer joinable.
              </p>
            </div>
          )}
        </section>
      ) : null}
    </div>
  )
}

export default LiveRoomPage
