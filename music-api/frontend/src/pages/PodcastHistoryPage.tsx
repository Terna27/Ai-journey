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
  getPodcastContinueListening,
  getPodcastListeningHistory,
} from '../lib/api'

import type {
  PodcastContinueListeningItem,
  PodcastListeningHistoryItem,
} from '../types/podcastPlayback'

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

function formatPlaybackTime(
  ms: number,
) {
  const totalSeconds =
    Math.max(0, Math.floor(ms / 1000))

  const hours = Math.floor(
    totalSeconds / 3600,
  )

  const minutes = Math.floor(
    (totalSeconds % 3600) / 60,
  )

  const seconds =
    totalSeconds % 60

  const mm = String(
    minutes,
  ).padStart(2, '0')

  const ss = String(
    seconds,
  ).padStart(2, '0')

  if (hours > 0) {
    return `${hours}:${mm}:${ss}`
  }

  return `${mm}:${ss}`
}

function progressPercent(
  positionMS: number,
  durationMS: number,
) {
  if (
    durationMS <= 0 ||
    positionMS <= 0
  ) {
    return 0
  }

  return Math.min(
    100,
    Math.round(
      (positionMS / durationMS) *
        100,
    ),
  )
}

function PodcastHistoryPage() {
  const navigate = useNavigate()

  const {
    isAuthenticated,
    isLoadingIdentity,
    token,
  } = useAuth()

  const {
    currentPodcastEpisode,
    isPlaying,
    playPodcastEpisode,
  } = usePlayer()

  const [continueItems, setContinueItems] =
    useState<
      PodcastContinueListeningItem[]
    >([])

  const [history, setHistory] =
    useState<
      PodcastListeningHistoryItem[]
    >([])

  const [loading, setLoading] =
    useState(true)

  const [error, setError] =
    useState('')

  const [offset, setOffset] =
    useState(0)

  const [hasMore, setHasMore] =
    useState(false)

  const loadData =
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

          const [continueResult, historyResult] =
            await Promise.all([
              getPodcastContinueListening(
                accessToken,
                PAGE_SIZE,
                0,
              ),

              getPodcastListeningHistory(
                accessToken,
                PAGE_SIZE,
                nextOffset,
              ),
            ])

          setContinueItems(
            continueResult,
          )

          setHistory(
            (current) =>
              append
                ? [
                    ...current,
                    ...historyResult,
                  ]
                : historyResult,
          )

          setHasMore(
            historyResult.length ===
              PAGE_SIZE,
          )

          setOffset(nextOffset)
        } catch (err) {
          setError(
            err instanceof Error
              ? err.message
              : 'Failed to load your podcast history',
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

    void loadData(
      0,
      false,
      token,
    )
  }, [
    isAuthenticated,
    isLoadingIdentity,
    loadData,
    navigate,
    token,
  ])

  /*
   * Resume flows through the existing global player:
   * a NEW playback session is created for the new
   * listening period and the saved position is applied
   * once audio metadata is ready.
   */
  const resumeEpisode = (
    item: PodcastContinueListeningItem,
    contextItems: PodcastContinueListeningItem[],
  ) => {
    playPodcastEpisode(
      item.podcast,
      item.episode,
      contextItems.map(
        (contextItem) =>
          contextItem.episode,
      ),
      item.last_position_ms /
        1000,
    )
  }

  const playHistoryItem = (
    item: PodcastListeningHistoryItem,
  ) => {
    // Completed episodes replay from the beginning in a
    // new lifecycle; unfinished ones resume at the saved
    // position.
    const startAtSeconds =
      item.completed
        ? 0
        : item.last_position_ms /
          1000

    playPodcastEpisode(
      item.podcast,
      item.episode,
      history.map(
        (historyItem) =>
          historyItem.episode,
      ),
      startAtSeconds,
    )
  }

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
            YOUR PODCASTS
          </p>

          <h2>
            Podcast History
          </h2>

          <p>
            Continue where you left
            off and revisit
            episodes you have
            listened to.
          </p>
        </div>
      </header>

      {loading && (
        <section className="content-panel">
          <p>
            Loading your podcast
            activity...
          </p>
        </section>
      )}

      {!loading && error && (
        <section className="content-panel">
          <p>{error}</p>
        </section>
      )}

      {!loading &&
        !error && (
          <>
            <section className="podcast-section">
              <h3>
                Continue Listening
              </h3>

              {continueItems.length ===
                0 && (
                <p className="podcast-section-empty">
                  Episodes you stop
                  partway will show up
                  here.
                </p>
              )}

              {continueItems.length >
                0 && (
                <div className="podcast-continue-list">
                  {continueItems.map(
                    (item) => {
                      const isCurrentEpisode =
                        currentPodcastEpisode?.id ===
                        item.episode.id

                      const isThisEpisodePlaying =
                        isCurrentEpisode &&
                        isPlaying

                      const percent =
                        progressPercent(
                          item.last_position_ms,
                          item.duration_ms,
                        )

                      const remainingMS = Math.max(
                        0,
                        item.duration_ms -
                          item.last_position_ms,
                      )

                      return (
                        <article
                          className="podcast-continue-card"
                          key={
                            item.episode.id
                          }
                        >
                          <div className="my-music-cover podcast-continue-artwork">
                            {(item
                              .episode
                              .artwork_url ||
                              item.podcast
                                .artwork_url) ? (
                              <img
                                src={
                                  item
                                    .episode
                                    .artwork_url ||
                                  item
                                    .podcast
                                    .artwork_url
                                }
                                alt={`${item.episode.title} artwork`}
                              />
                            ) : (
                              <div className="my-music-cover-fallback">
                                No
                                cover
                              </div>
                            )}
                          </div>

                          <div className="podcast-continue-body">
                            <div className="podcast-continue-heading">
                              <div className="podcast-continue-titles">
                                <h4>
                                  {
                                    item
                                      .episode
                                      .title
                                  }
                                </h4>

                                <Link
                                  to={`/podcasts/${item.podcast.slug}`}
                                  className="artist-link"
                                >
                                  {
                                    item
                                      .podcast
                                      .title
                                  }
                                </Link>
                              </div>

                              <span className="podcast-continue-played">
                                {
                                  formatRelativeTime(
                                    item.last_played_at,
                                  )
                                }
                              </span>
                            </div>

                            <div className="podcast-progress-row">
                              <div
                                className="podcast-progress-bar"
                                role="progressbar"
                                aria-valuenow={
                                  percent
                                }
                                aria-valuemin={
                                  0
                                }
                                aria-valuemax={
                                  100
                                }
                                aria-label={`Progress through ${item.episode.title}`}
                              >
                                <div
                                  className="podcast-progress-fill"
                                  style={{
                                    width: `${percent}%`,
                                  }}
                                />
                              </div>

                              <span className="podcast-progress-time">
                                {
                                  formatPlaybackTime(
                                    item.last_position_ms,
                                  )
                                }{' '}
                                /{' '}
                                {
                                  formatPlaybackTime(
                                    item.duration_ms,
                                  )
                                }
                              </span>
                            </div>

                            <div className="podcast-continue-actions">
                              <button
                                type="button"
                                className="secondary-button podcast-resume-button"
                                onClick={() =>
                                  resumeEpisode(
                                    item,
                                    continueItems,
                                  )
                                }
                              >
                                {isThisEpisodePlaying
                                  ? '❚❚ Playing'
                                  : `▶ Resume · ${formatPlaybackTime(
                                      remainingMS,
                                    )} left`}
                              </button>
                            </div>
                          </div>
                        </article>
                      )
                    },
                  )}
                </div>
              )}
            </section>

            <section className="podcast-section">
              <h3>
                All Episodes
              </h3>

              {history.length ===
                0 && (
                <p className="podcast-section-empty">
                  You have not
                  listened to any
                  podcast episodes
                  yet.
                </p>
              )}

              {history.length >
                0 && (
                <div className="podcast-history-list">
                  {history.map(
                    (item) => {
                      const isCurrentEpisode =
                        currentPodcastEpisode?.id ===
                        item.episode.id

                      const isThisEpisodePlaying =
                        isCurrentEpisode &&
                        isPlaying

                      return (
                        <article
                          className="podcast-history-card"
                          key={
                            item.episode.id
                          }
                        >
                          <div className="podcast-history-artwork">
                            {(item
                              .episode
                              .artwork_url ||
                              item.podcast
                                .artwork_url) ? (
                              <img
                                src={
                                  item
                                    .episode
                                    .artwork_url ||
                                  item
                                    .podcast
                                    .artwork_url
                                }
                                alt={`${item.episode.title} artwork`}
                              />
                            ) : (
                              <div className="my-music-cover-fallback">
                                No
                                cover
                              </div>
                            )}
                          </div>

                          <div className="podcast-history-body">
                            <h4>
                              {
                                item
                                  .episode
                                  .title
                              }
                            </h4>

                            <Link
                              to={`/podcasts/${item.podcast.slug}`}
                              className="artist-link"
                            >
                              {
                                item
                                  .podcast
                                  .title
                              }
                            </Link>

                            <p className="podcast-history-meta">
                              Played{' '}
                              {
                                formatRelativeTime(
                                  item.last_played_at,
                                )
                              }

                              {item
                                .qualified_play_count >
                                0 &&
                                ` · ${item.qualified_play_count} ${
                                  item.qualified_play_count ===
                                  1
                                    ? 'play'
                                    : 'plays'
                                }`}
                            </p>
                          </div>

                          <div className="podcast-history-side">
                            {item.completed ? (
                              <span className="podcast-history-badge">
                                Completed
                              </span>
                            ) : (
                              <span className="podcast-history-badge podcast-history-badge-progress">
                                {
                                  progressPercent(
                                    item.last_position_ms,
                                    item.duration_ms,
                                  )
                                }
                                % done
                              </span>
                            )}

                            <button
                              type="button"
                              className="secondary-button"
                              aria-label={
                                isThisEpisodePlaying
                                  ? `Pause ${item.episode.title}`
                                  : item.completed
                                    ? `Replay ${item.episode.title}`
                                    : `Play ${item.episode.title}`
                              }
                              onClick={() =>
                                playHistoryItem(
                                  item,
                                )
                              }
                            >
                              {isThisEpisodePlaying
                                ? '❚❚'
                                : item.completed
                                  ? '▶ Replay'
                                  : '▶ Play'}
                            </button>
                          </div>
                        </article>
                      )
                    },
                  )}
                </div>
              )}
            </section>
          </>
        )}

      {!loading &&
        !error &&
        hasMore && (
          <div className="load-more-row">
            <button
              type="button"
              className="secondary-button"
              onClick={() =>
                void loadData(
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

export default PodcastHistoryPage
