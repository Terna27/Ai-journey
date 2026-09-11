import {
  useCallback,
  useEffect,
  useMemo,
  useState,
} from 'react'

import {
  Link,
  useNavigate,
} from 'react-router-dom'

import { useAuth } from '../context/AuthContext'
import { useLibrary } from '../context/LibraryContext'
import { usePlayer } from '../context/PlayerContext'

import {
  getHeroArtists,
  getMusic,
} from '../lib/api'

import type {
  DiscoveryHeroArtist,
} from '../types/discovery'

import type { Music } from '../types/music'

import AddToPlaylistButton from '../components/music/AddToPlaylistButton'
import AddToQueueButton from '../components/music/AddToQueueButton'

function chooseWeightedHeroIndex(
  artists: DiscoveryHeroArtist[],
  currentIndex: number,
) {
  if (artists.length <= 1) {
    return 0
  }

  const candidates = artists
    .map((artist, index) => ({
      artist,
      index,
    }))
    .filter(
      ({ index }) =>
        index !== currentIndex,
    )

  const weightedCandidates =
    candidates.map(
      ({
        artist,
        index,
      }) => {
        const engagementWeight =
          Math.max(
            artist.engagement_score,
            0,
          ) + 1

        const rankingWeight =
          Math.max(
            artists.length - index,
            1,
          )

        return {
          index,
          weight:
            engagementWeight *
            rankingWeight,
        }
      },
    )

  const totalWeight =
    weightedCandidates.reduce(
      (total, candidate) =>
        total +
        candidate.weight,
      0,
    )

  if (totalWeight <= 0) {
    return candidates[
      Math.floor(
        Math.random() *
          candidates.length,
      )
    ].index
  }

  let randomValue =
    Math.random() * totalWeight

  for (
    const candidate
    of weightedCandidates
  ) {
    randomValue -= candidate.weight

    if (randomValue <= 0) {
      return candidate.index
    }
  }

  return weightedCandidates[
    weightedCandidates.length - 1
  ].index
}

function HomePage() {
  const navigate = useNavigate()

  const {
    user,
    isAuthenticated,
    isArtist,
    logout,
  } = useAuth()

  const {
    isLiked,
    toggleLike,
  } = useLibrary()

  const {
    currentTrack,
    isPlaying,
    playTrack,
    stopPlayback,
  } = usePlayer()

  const [music, setMusic] =
    useState<Music[]>([])

  const [
    heroArtists,
    setHeroArtists,
  ] = useState<
    DiscoveryHeroArtist[]
  >([])

  const [
    activeHeroIndex,
    setActiveHeroIndex,
  ] = useState(0)

  const [
    heroVideoFailed,
    setHeroVideoFailed,
  ] = useState(false)

  const [
    heroLoading,
    setHeroLoading,
  ] = useState(true)

  const [
    heroError,
    setHeroError,
  ] = useState('')

  const [loading, setLoading] =
    useState(true)

  const [error, setError] =
    useState('')

  const [
    pendingLikeTrackID,
    setPendingLikeTrackID,
  ] = useState<number | null>(
    null,
  )

  const [
    likeError,
    setLikeError,
  ] = useState('')

  useEffect(() => {
    let cancelled = false

    async function loadMusic() {
      try {
        setLoading(true)
        setError('')

        const result =
          await getMusic()

        if (!cancelled) {
          setMusic(result)
        }
      } catch (err) {
        if (!cancelled) {
          setError(
            err instanceof Error
              ? err.message
              : 'Failed to load music',
          )
        }
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    }

    void loadMusic()

    return () => {
      cancelled = true
    }
  }, [])

  useEffect(() => {
    let cancelled = false

    async function loadHeroArtists() {
      try {
        setHeroLoading(true)
        setHeroError('')

        const result =
          await getHeroArtists()

        if (cancelled) {
          return
        }

        setHeroArtists(
          result.artists,
        )

        if (
          result.artists.length > 1
        ) {
          setActiveHeroIndex(
            chooseWeightedHeroIndex(
              result.artists,
              -1,
            ),
          )
        } else {
          setActiveHeroIndex(0)
        }
      } catch (err) {
        if (!cancelled) {
          setHeroError(
            err instanceof Error
              ? err.message
              : 'Failed to load featured artists',
          )
        }
      } finally {
        if (!cancelled) {
          setHeroLoading(false)
        }
      }
    }

    void loadHeroArtists()

    return () => {
      cancelled = true
    }
  }, [])

  const featuredMusic =
    useMemo(() => {
      return music.slice(0, 8)
    }, [music])

  const activeHeroArtist =
    heroArtists[
      activeHeroIndex
    ] ?? null

  const selectNextHero =
    useCallback(() => {
      if (
        heroArtists.length <= 1
      ) {
        return
      }

      setActiveHeroIndex(
        (currentIndex) =>
          chooseWeightedHeroIndex(
            heroArtists,
            currentIndex,
          ),
      )
    }, [heroArtists])

  function selectPreviousHero() {
    if (
      heroArtists.length <= 1
    ) {
      return
    }

    setActiveHeroIndex(
      (currentIndex) =>
        currentIndex === 0
          ? heroArtists.length - 1
          : currentIndex - 1,
    )
  }

  useEffect(() => {
    setHeroVideoFailed(false)
  }, [
    activeHeroArtist?.hero_video_url,
  ])

  useEffect(() => {
    if (
      !heroVideoFailed ||
      heroArtists.length <= 1
    ) {
      return
    }

    const timer =
      window.setTimeout(
        selectNextHero,
        12000,
      )

    return () => {
      window.clearTimeout(timer)
    }
  }, [
    heroArtists.length,
    heroVideoFailed,
    selectNextHero,
  ])

  function handleLogout() {
    stopPlayback()
    logout()
  }

  function handlePlay(
    track: Music,
  ) {
    if (!isAuthenticated) {
      navigate('/login', {
        state: {
          message:
            'Please log in to play music.',
        },
      })

      return
    }

    playTrack(
      track,
      featuredMusic,
    )
  }

  async function handleLike(
    track: Music,
  ) {
    if (!isAuthenticated) {
      navigate('/login', {
        state: {
          message:
            'Please log in to like music.',
        },
      })

      return
    }

    if (
      pendingLikeTrackID !== null
    ) {
      return
    }

    try {
      setPendingLikeTrackID(
        track.id,
      )

      setLikeError('')

      await toggleLike(track)
    } catch (err) {
      setLikeError(
        err instanceof Error
          ? err.message
          : 'Failed to update liked songs',
      )
    } finally {
      setPendingLikeTrackID(null)
    }
  }

  return (
    <>
      <header className="topbar">
        <div>
          <p className="eyebrow">
            WELCOME
          </p>

          <h2>
            Discover something new
          </h2>
        </div>

        <div className="topbar-actions">
          {isAuthenticated ? (
            <>
              <Link
                to="/profile"
                className="artist-account-link"
              >
                <span className="artist-avatar">
                  {user?.name
                    ?.charAt(0)
                    .toUpperCase() ??
                    'U'}
                </span>

                <span className="artist-account-info">
                  <strong>
                    {user?.name}
                  </strong>

                  <small>
                    {isArtist
                      ? 'Artist'
                      : 'Listener'}
                  </small>
                </span>
              </Link>

              {isArtist && (
                <Link
                  to="/upload"
                  className="secondary-button"
                >
                  Upload
                </Link>
              )}

              <button
                type="button"
                className="logout-button"
                onClick={
                  handleLogout
                }
              >
                Logout
              </button>
            </>
          ) : (
            <>
              <Link
                to="/login"
                className="secondary-button"
              >
                Login
              </Link>

              <Link
                to="/register"
                className="primary-button"
              >
                Create account
              </Link>
            </>
          )}
        </div>
      </header>

      {heroLoading && (
        <section className="home-video-hero home-video-hero-loading">
          <div className="home-video-hero-content">
            <p className="hero-badge">
              Featured artists
            </p>

            <h3>
              Discovering what is
              moving right now...
            </h3>
          </div>
        </section>
      )}

      {!heroLoading &&
        activeHeroArtist && (
          <section
            className="home-video-hero"
            style={
              activeHeroArtist
                .hero_video_poster_url
                ? {
                    backgroundImage:
                      `url("${activeHeroArtist.hero_video_poster_url}")`,
                  }
                : undefined
            }
          >
            <div className="home-video-hero-media">
              {!heroVideoFailed && (
                <video
                  key={
                    activeHeroArtist.id
                  }
                  className="home-video-hero-video"
                  src={
                    activeHeroArtist.hero_video_url
                  }
                  poster={
                    activeHeroArtist.hero_video_poster_url
                  }
                  autoPlay
                  muted
                  playsInline
                  preload="metadata"
                  loop={
                    heroArtists.length ===
                    1
                  }
                  aria-hidden="true"
                  onEnded={
                    selectNextHero
                  }
                  onError={() =>
                    setHeroVideoFailed(
                      true,
                    )
                  }
                />
              )}

              {heroVideoFailed &&
                activeHeroArtist
                  .hero_video_poster_url && (
                  <img
                    className="home-video-hero-poster"
                    src={
                      activeHeroArtist.hero_video_poster_url
                    }
                    alt=""
                    aria-hidden="true"
                  />
                )}
            </div>

            <div className="home-video-hero-overlay" />

            <div className="home-video-hero-content">
              <div className="home-video-hero-artist">
                <div className="home-video-hero-avatar">
                  {activeHeroArtist
                    .profile_image_url ? (
                    <img
                      src={
                        activeHeroArtist.profile_image_url
                      }
                      alt={`${activeHeroArtist.name} profile`}
                    />
                  ) : (
                    <span>
                      {activeHeroArtist.name
                        .charAt(0)
                        .toUpperCase()}
                    </span>
                  )}
                </div>

                <div className="home-video-hero-copy">
                  <span className="hero-badge">
                    Featured artist
                  </span>

                  <h1>
                    {
                      activeHeroArtist.name
                    }
                  </h1>

                  {activeHeroArtist.bio && (
                    <p className="home-video-hero-bio">
                      {
                        activeHeroArtist.bio
                      }
                    </p>
                  )}

                  <div className="home-video-hero-stats">
                    <span>
                      {
                        activeHeroArtist.track_count
                      }{' '}
                      {activeHeroArtist.track_count ===
                      1
                        ? 'track'
                        : 'tracks'}
                    </span>

                    <span
                      aria-hidden="true"
                    >
                      •
                    </span>

                    <span>
                      Trending score{' '}
                      {
                        activeHeroArtist.engagement_score
                      }
                    </span>
                  </div>

                  <div className="home-video-hero-actions">
                    <Link
                      to={`/artists/${activeHeroArtist.id}`}
                      className="home-video-hero-primary"
                    >
                      View Artist
                    </Link>

                    <a
                      href="#discover"
                      className="home-video-hero-secondary"
                    >
                      Explore music
                    </a>
                  </div>
                </div>
              </div>

              {heroArtists.length >
                1 && (
                <div className="home-video-hero-navigation">
                  <button
                    type="button"
                    className="home-video-nav-button"
                    aria-label="Previous featured artist"
                    onClick={
                      selectPreviousHero
                    }
                  >
                    ‹
                  </button>

                  <div className="home-video-hero-dots">
                    {heroArtists.map(
                      (
                        artist,
                        index,
                      ) => (
                        <button
                          type="button"
                          key={
                            artist.id
                          }
                          className={
                            index ===
                            activeHeroIndex
                              ? 'home-video-hero-dot is-active'
                              : 'home-video-hero-dot'
                          }
                          aria-label={`Show ${artist.name}`}
                          aria-current={
                            index ===
                            activeHeroIndex
                              ? 'true'
                              : undefined
                          }
                          onClick={() =>
                            setActiveHeroIndex(
                              index,
                            )
                          }
                        />
                      ),
                    )}
                  </div>

                  <button
                    type="button"
                    className="home-video-nav-button"
                    aria-label="Next featured artist"
                    onClick={
                      selectNextHero
                    }
                  >
                    ›
                  </button>
                </div>
              )}
            </div>
          </section>
        )}

      {!heroLoading &&
        !activeHeroArtist && (
          <section className="hero-section">
            <div className="hero-copy">
              <span className="hero-badge">
                Discover music
              </span>

              <h3>
                Music for every moment.
              </h3>

              <p>
                Discover independent
                artists, stream new songs
                and build a collection
                around the music you
                love.
              </p>

              {heroError && (
                <p className="form-error">
                  {heroError}
                </p>
              )}

              <div className="hero-actions">
                <a
                  href="#discover"
                  className="primary-button"
                >
                  Explore music
                </a>

                {!isAuthenticated && (
                  <Link
                    to="/register"
                    className="secondary-button"
                  >
                    Create account
                  </Link>
                )}
              </div>
            </div>

            <div className="hero-art">
              <div className="hero-disc">
                <div className="hero-disc-center" />
              </div>
            </div>
          </section>
        )}

      <section
        className="section"
        id="discover"
      >
        <div className="section-heading">
          <div>
            <p className="eyebrow">
              DISCOVER
            </p>

            <h3>
              Trending now
            </h3>
          </div>

          <span className="text-button">
            {music.length} tracks
          </span>
        </div>

        {likeError && (
          <section className="content-panel">
            <p className="form-error">
              {likeError}
            </p>
          </section>
        )}

        {loading && (
          <section className="content-panel">
            <p>
              Loading music...
            </p>
          </section>
        )}

        {!loading &&
          error && (
            <section className="content-panel">
              <p>
                {error}
              </p>
            </section>
          )}

        {!loading &&
          !error &&
          featuredMusic.length ===
            0 && (
            <section className="content-panel">
              <p>
                No music has been
                uploaded yet.
              </p>
            </section>
          )}

        {!loading &&
          !error &&
          featuredMusic.length >
            0 && (
            <div className="music-grid">
              {featuredMusic.map(
                (
                  track,
                  index,
                ) => {
                  const isCurrentTrack =
                    currentTrack?.id ===
                    track.id

                  const isThisTrackPlaying =
                    isCurrentTrack &&
                    isPlaying

                  const trackIsLiked =
                    isLiked(track.id)

                  const isLikePending =
                    pendingLikeTrackID ===
                    track.id

                  return (
                    <article
                      className="music-card"
                      key={track.id}
                    >
                      <div
                        className={`music-cover cover-${(index %
                          4) +
                          1}`}
                      >
                        {track.image_url && (
                          <img
                            src={
                              track.image_url
                            }
                            alt={`${track.song_title} cover`}
                          />
                        )}
                      </div>

                      <div className="music-card-content">
                        <div className="music-card-info">
                          <h4 className="music-card-title">
                            {
                              track.song_title
                            }
                          </h4>

                          <p className="music-card-artist">
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
                              track.artist_name
                            )}
                          </p>
                        </div>

                        <div className="music-card-actions">
                          <span className="genre-pill">
                            {
                              track.genre
                            }
                          </span>

                          {track.audio_url && (
                            <button
                              type="button"
                              className="track-play-button"
                              aria-label={
                                isThisTrackPlaying
                                  ? `Pause ${track.song_title}`
                                  : `Play ${track.song_title}`
                              }
                              onClick={() =>
                                handlePlay(
                                  track,
                                )
                              }
                            >
                              {isThisTrackPlaying
                                ? '❚❚'
                                : '▶'}
                            </button>
                          )}

                          {isAuthenticated && (
                            <AddToQueueButton
                              track={
                                track
                              }
                            />
                          )}

                          {isAuthenticated && (
                            <AddToPlaylistButton
                              track={
                                track
                              }
                            />
                          )}

                          {isAuthenticated && (
                            <button
                              type="button"
                              className={`music-like-button${trackIsLiked
                                ? ' is-liked'
                                : ''
                                }`}
                              aria-label={
                                trackIsLiked
                                  ? `Unlike ${track.song_title}`
                                  : `Like ${track.song_title}`
                              }
                              aria-pressed={
                                trackIsLiked
                              }
                              disabled={
                                isLikePending
                              }
                              onClick={() =>
                                void handleLike(
                                  track,
                                )
                              }
                            >
                              <svg
                                className="music-like-icon"
                                viewBox="0 0 24 24"
                                aria-hidden="true"
                              >
                                <path
                                  d="M12 21s-7.2-4.35-9.5-8.5C.85 9.5 2.15 5.5 5.75 4.5c2.15-.6 4.15.25 5.25 1.85C12.1 4.75 14.1 3.9 16.25 4.5c3.6 1 4.9 5 3.25 8C17.2 16.65 12 21 12 21Z"
                                  fill={
                                    trackIsLiked
                                      ? 'currentColor'
                                      : 'none'
                                  }
                                  stroke="currentColor"
                                  strokeWidth="1.8"
                                  strokeLinecap="round"
                                  strokeLinejoin="round"
                                />
                              </svg>
                            </button>
                          )}
                        </div>
                      </div>
                    </article>
                  )
                },
              )}
            </div>
          )}
      </section>
    </>
  )
}

export default HomePage