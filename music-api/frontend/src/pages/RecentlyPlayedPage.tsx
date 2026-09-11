import {
  useCallback,
  useEffect,
  useState,
} from 'react'

import {
  Link,
  useNavigate,
} from 'react-router-dom'

import { useAuth } from '../context/AuthContext'
import { usePlayer } from '../context/PlayerContext'

import {
  getListeningHistory,
} from '../lib/api'

import type {
  ListeningHistoryItem,
} from '../types/playback'

const PAGE_SIZE = 20

function formatRelativeTime(
  value: string,
) {
  const date = new Date(value)

  if (
    Number.isNaN(
      date.getTime(),
    )
  ) {
    return ''
  }

  const seconds = Math.floor(
    (Date.now() -
      date.getTime()) /
      1000,
  )

  if (seconds < 60) {
    return 'just now'
  }

  const minutes = Math.floor(
    seconds / 60,
  )

  if (minutes < 60) {
    return `${minutes}m ago`
  }

  const hours = Math.floor(
    minutes / 60,
  )

  if (hours < 24) {
    return `${hours}h ago`
  }

  const days = Math.floor(
    hours / 24,
  )

  if (days < 30) {
    return `${days}d ago`
  }

  return new Intl.DateTimeFormat(
    'en',
    {
      day: 'numeric',
      month: 'short',
      year: 'numeric',
    },
  ).format(date)
}

function formatListenCount(
  count: number,
) {
  if (count <= 0) {
    return ''
  }

  return `${count} ${
    count === 1
      ? 'play'
      : 'plays'
  }`
}

function RecentlyPlayedPage() {
  const navigate = useNavigate()

  const {
    isAuthenticated,
    isLoadingIdentity,
    token,
  } = useAuth()

  const {
    currentTrack,
    isPlaying,
    playTrack,
  } = usePlayer()

  const [history, setHistory] =
    useState<
      ListeningHistoryItem[]
    >([])

  const [loading, setLoading] =
    useState(true)

  const [error, setError] =
    useState('')

  const [offset, setOffset] =
    useState(0)

  const [hasMore, setHasMore] =
    useState(false)

  const loadHistory =
    useCallback(
      async (
        nextOffset: number,
        append: boolean,
        accessToken: string,
      ) => {
        try {
          if (!append) {
            setLoading(true)
          }

          setError('')

          const items =
            await getListeningHistory(
              accessToken,
              PAGE_SIZE,
              nextOffset,
            )

          setHistory(
            (current) =>
              append
                ? [
                    ...current,
                    ...items,
                  ]
                : items,
          )

          setHasMore(
            items.length ===
              PAGE_SIZE,
          )

          setOffset(nextOffset)
        } catch (err) {
          setError(
            err instanceof Error
              ? err.message
              : 'Failed to load your listening history',
          )
        } finally {
          setLoading(false)
        }
      },
      [],
    )

  useEffect(() => {
    if (
      isLoadingIdentity
    ) {
      return
    }

    if (
      !isAuthenticated ||
      !token
    ) {
      navigate(
        '/login',
        {
          replace: true,
        },
      )

      return
    }

    void loadHistory(
      0,
      false,
      token,
    )
  }, [
    isAuthenticated,
    isLoadingIdentity,
    loadHistory,
    navigate,
    token,
  ])

  const recentlyPlayedTracks =
    history.map(
      (item) => item.music,
    )

  if (
    !isLoadingIdentity &&
    !isAuthenticated
  ) {
    return null
  }

  return (
    <>
      <header className="page-header">
        <div>
          <p className="eyebrow">
            YOUR ACTIVITY
          </p>

          <h2>
            Recently Played
          </h2>

          <p>
            The music you have
            been listening to,
            newest first.
          </p>
        </div>
      </header>

      {loading && (
        <section className="content-panel">
          <p>
            Loading your
            listening history...
          </p>
        </section>
      )}

      {!loading && error && (
        <section className="content-panel">
          <p>{error}</p>
        </section>
      )}

      {!loading &&
        !error &&
        history.length ===
          0 && (
          <section className="content-panel">
            <p>
              You have not
              played any music
              yet. Start
              listening and
              your history
              will appear here.
            </p>
          </section>
        )}

      {!loading &&
        !error &&
        history.length >
          0 && (
          <section className="my-music-grid">
            {history.map(
              (item) => {
                const track =
                  item.music

                const isCurrentTrack =
                  currentTrack?.id ===
                  track.id

                const isThisTrackPlaying =
                  isCurrentTrack &&
                  isPlaying

                const playCountLabel =
                  formatListenCount(
                    item.qualified_play_count,
                  )

                const playedAtLabel =
                  formatRelativeTime(
                    item.last_played_at,
                  )

                return (
                  <article
                    className={
                      isCurrentTrack
                        ? 'my-music-card playing'
                        : 'my-music-card'
                    }
                    key={
                      track.id
                    }
                  >
                    <div className="my-music-cover">
                      {track.image_url ? (
                        <img
                          src={
                            track.image_url
                          }
                          alt={`${track.song_title} cover`}
                        />
                      ) : (
                        <div className="my-music-cover-fallback">
                          No cover
                        </div>
                      )}

                      {track.audio_url && (
                        <button
                          type="button"
                          className="my-music-play-button"
                          aria-label={
                            isThisTrackPlaying
                              ? `Pause ${track.song_title}`
                              : `Play ${track.song_title}`
                          }
                          onClick={() =>
                            playTrack(
                              track,
                              recentlyPlayedTracks,
                            )
                          }
                        >
                          {isThisTrackPlaying
                            ? '❚❚'
                            : '▶'}
                        </button>
                      )}
                    </div>

                    <div className="my-music-card-body">
                      <div className="my-music-card-heading">
                        <div>
                          <h3>
                            {
                              track.song_title
                            }
                          </h3>

                          {track.artist_id ? (
                            <Link
                              to={`/artists/${track.artist_id}`}
                              className="artist-link"
                            >
                              {
                                track.artist_name
                              }
                            </Link>
                          ) : (
                            <span>
                              {
                                track.artist_name
                              }
                            </span>
                          )}
                        </div>

                        <span className="music-genre">
                          {
                            track.genre
                          }
                        </span>
                      </div>

                      <p className="music-upload-date">
                        Played{' '}
                        {
                          playedAtLabel
                        }
                      </p>

                      {playCountLabel && (
                        <p className="music-now-playing">
                          {
                            playCountLabel
                          }
                        </p>
                      )}

                      {isThisTrackPlaying && (
                        <p className="music-now-playing">
                          Now
                          playing
                        </p>
                      )}

                      {!track.audio_url && (
                        <p className="music-unavailable">
                          Audio
                          unavailable
                        </p>
                      )}
                    </div>
                  </article>
                )
              },
            )}
          </section>
        )}

      {!loading &&
        !error &&
        hasMore && (
          <div className="load-more-row">
            <button
              type="button"
              className="secondary-button"
              onClick={() =>
                void loadHistory(
                  offset +
                    PAGE_SIZE,
                  true,
                  token ??
                    '',
                )
              }
            >
              Load more
            </button>
          </div>
        )}
    </>
  )
}

export default RecentlyPlayedPage
