import {
  useEffect,
  useMemo,
  useState,
} from 'react'

import {
  Link,
  useNavigate,
} from 'react-router-dom'

import { useAuth } from '../context/AuthContext'
import { usePlayer } from '../context/PlayerContext'
import { getMusic } from '../lib/api'
import type { Music } from '../types/music'

function HomePage() {
  const navigate = useNavigate()

  const {
    user,
    isAuthenticated,
    isArtist,
    logout,
  } = useAuth()

  const {
    currentTrack,
    isPlaying,
    playTrack,
    stopPlayback,
  } = usePlayer()

  const [music, setMusic] =
    useState<Music[]>([])

  const [loading, setLoading] =
    useState(true)

  const [error, setError] =
    useState('')

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

  const featuredMusic =
    useMemo(() => {
      return music.slice(0, 8)
    }, [music])

  function handleLogout() {
    stopPlayback()
    logout()
  }

  function handlePlay(track: Music) {
    if (!isAuthenticated) {
      navigate('/login', {
        state: {
          message:
            'Please log in to play music.',
        },
      })

      return
    }

    playTrack(track)
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
                    .toUpperCase() ?? 'U'}
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
                onClick={handleLogout}
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

      <section className="hero-section">
        <div className="hero-copy">
          <span className="hero-badge">
            Featured this week
          </span>

          <h3>
            Music for every moment.
          </h3>

          <p>
            Discover independent artists,
            stream new songs and build a
            collection around the music
            you love.
          </p>

          <div className="hero-actions">
            <a
              href="#discover"
              className="primary-button"
            >
              Explore music
            </a>

            {isAuthenticated ? (
              isArtist ? (
                <Link
                  to="/upload"
                  className="secondary-button"
                >
                  Upload your music
                </Link>
              ) : (
                <Link
                  to="/profile"
                  className="secondary-button"
                >
                  Become an Artist
                </Link>
              )
            ) : (
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

      <section
        className="section"
        id="discover"
      >
        <div className="section-heading">
          <div>
            <p className="eyebrow">
              DISCOVER
            </p>

            <h3>Trending now</h3>
          </div>

          <span className="text-button">
            {music.length} tracks
          </span>
        </div>

        {loading && (
          <section className="content-panel">
            <p>Loading music...</p>
          </section>
        )}

        {!loading && error && (
          <section className="content-panel">
            <p>{error}</p>
          </section>
        )}

        {!loading &&
          !error &&
          featuredMusic.length === 0 && (
            <section className="content-panel">
              <p>
                No music has been
                uploaded yet.
              </p>
            </section>
          )}

        {!loading &&
          !error &&
          featuredMusic.length > 0 && (
            <div className="music-grid">
              {featuredMusic.map(
                (track, index) => {
                  const isCurrentTrack =
                    currentTrack?.id ===
                    track.id

                  const isThisTrackPlaying =
                    isCurrentTrack &&
                    isPlaying

                  return (
                    <article
                      className="music-card"
                      key={track.id}
                    >
                      <div
                        className={`music-cover cover-${
                          (index % 4) + 1
                        }`}
                      >
                        {track.image_url && (
                          <img
                            src={
                              track.image_url
                            }
                            alt={`${track.song_title} cover`}
                          />
                        )}

                        {track.audio_url && (
                          <button
                            type="button"
                            className="play-button"
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
                      </div>

                      <div className="music-card-content">
                        <div>
                          <h4>
                            {
                              track.song_title
                            }
                          </h4>

                          <p>
                            {
                              track.artist_name
                            }
                          </p>

                          {isCurrentTrack &&
                            isAuthenticated && (
                              <p className="music-now-playing">
                                {isPlaying
                                  ? 'Now playing'
                                  : 'Paused'}
                              </p>
                            )}
                        </div>

                        <span className="genre-pill">
                          {track.genre}
                        </span>
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