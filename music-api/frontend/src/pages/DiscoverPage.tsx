import {
  useEffect,
  useState,
} from 'react'

import {
  Link,
  useNavigate,
} from 'react-router-dom'

import AddToPlaylistButton from '../components/music/AddToPlaylistButton'
import AddToQueueButton from '../components/music/AddToQueueButton'

import { useAuth } from '../context/AuthContext'
import { useLibrary } from '../context/LibraryContext'
import { usePlayer } from '../context/PlayerContext'

import {
  getDiscovery,
} from '../lib/api'

import type {
  DiscoveryResponse,
} from '../types/discovery'

import type {
  Music,
} from '../types/music'

function DiscoverPage() {
  const navigate = useNavigate()

  const {
    isAuthenticated,
  } = useAuth()

  const {
    isLiked,
    toggleLike,
  } = useLibrary()

  const {
    currentTrack,
    isPlaying,
    playTrack,
  } = usePlayer()

  const [
    discovery,
    setDiscovery,
  ] = useState<DiscoveryResponse | null>(
    null,
  )

  const [
    loading,
    setLoading,
  ] = useState(true)

  const [
    error,
    setError,
  ] = useState('')

  const [
    likeError,
    setLikeError,
  ] = useState('')

  const [
    pendingLikeTrackID,
    setPendingLikeTrackID,
  ] = useState<number | null>(null)

  useEffect(() => {
    let cancelled = false

    async function loadDiscovery() {
      try {
        setLoading(true)
        setError('')

        const result =
          await getDiscovery()

        if (!cancelled) {
          setDiscovery(result)
        }
      } catch (err) {
        if (!cancelled) {
          setError(
            err instanceof Error
              ? err.message
              : 'Failed to load discovery',
          )
        }
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    }

    void loadDiscovery()

    return () => {
      cancelled = true
    }
  }, [])

  function handlePlay(
    track: Music,
    queue: Music[],
  ) {
    if (!isAuthenticated) {
      navigate(
        '/login',
        {
          state: {
            message:
              'Please log in to play music.',
          },
        },
      )

      return
    }

    playTrack(
      track,
      queue,
    )
  }

  async function handleLike(
    track: Music,
  ) {
    if (!isAuthenticated) {
      navigate(
        '/login',
        {
          state: {
            message:
              'Please log in to like music.',
          },
        },
      )

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

  function renderTracks(
    tracks: Music[],
  ) {
    if (tracks.length === 0) {
      return (
        <section className="content-panel">
          <p>
            No tracks available yet.
          </p>
        </section>
      )
    }

    return (
      <div className="music-grid">
        {tracks.map(
          (track, index) => {
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
                  className={`music-cover cover-${(index % 4) + 1}`}
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
                    <Link
                      to={`/search?q=${encodeURIComponent(
                        track.genre,
                      )}&type=track&genre=${encodeURIComponent(
                        track.genre,
                      )}`}
                      className="genre-pill"
                    >
                      {track.genre}
                    </Link>

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
                          tracks,
                        )
                      }
                    >
                      {isThisTrackPlaying
                        ? '❚❚'
                        : '▶'}
                    </button>

                    {isAuthenticated && (
                      <AddToQueueButton
                        track={track}
                      />
                    )}

                    {isAuthenticated && (
                      <AddToPlaylistButton
                        track={track}
                      />
                    )}

                    {isAuthenticated && (
                      <button
                        type="button"
                        className={`music-like-button${
                          trackIsLiked
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
    )
  }

  if (loading) {
    return (
      <>
        <header className="topbar">
          <div>
            <p className="eyebrow">
              DISCOVER
            </p>

            <h2>
              Explore music
            </h2>
          </div>
        </header>

        <section className="content-panel">
          <p>
            Loading discovery...
          </p>
        </section>
      </>
    )
  }

  if (
    error ||
    !discovery
  ) {
    return (
      <>
        <header className="topbar">
          <div>
            <p className="eyebrow">
              DISCOVER
            </p>

            <h2>
              Explore music
            </h2>
          </div>
        </header>

        <section className="content-panel">
          <p className="form-error">
            {error ||
              'Discovery is unavailable.'}
          </p>
        </section>
      </>
    )
  }

  return (
    <>
      <header className="topbar">
        <div>
          <p className="eyebrow">
            DISCOVER
          </p>

          <h2>
            Find your next favorite sound
          </h2>
        </div>

        <Link
          to="/search"
          className="secondary-button"
        >
          Search music
        </Link>
      </header>

      <section className="discover-intro">
        <div>
          <span className="hero-badge">
            Explore
          </span>

          <h3>
            Music picked from what is
            happening across the platform.
          </h3>

          <p>
            Explore trending tracks,
            new music, releases, artists
            and genres.
          </p>
        </div>
      </section>

      {likeError && (
        <section className="content-panel">
          <p className="form-error">
            {likeError}
          </p>
        </section>
      )}

      <section className="section">
        <div className="section-heading">
          <div>
            <p className="eyebrow">
              TRENDING
            </p>

            <h3>
              Trending now
            </h3>
          </div>

          <span className="text-button">
            {
              discovery
                .trending_tracks
                .length
            } tracks
          </span>
        </div>

        {renderTracks(
          discovery.trending_tracks,
        )}
      </section>

      <section className="section">
        <div className="section-heading">
          <div>
            <p className="eyebrow">
              FRESH MUSIC
            </p>

            <h3>
              New tracks
            </h3>
          </div>
        </div>

        {renderTracks(
          discovery.new_tracks,
        )}
      </section>

      <section className="section">
        <div className="section-heading">
          <div>
            <p className="eyebrow">
              RELEASES
            </p>

            <h3>
              New releases
            </h3>
          </div>
        </div>

        {discovery.new_releases.length ===
        0 ? (
          <section className="content-panel">
            <p>
              No published releases yet.
            </p>
          </section>
        ) : (
          <div className="discover-release-grid">
            {discovery.new_releases.map(
              (release) => (
                <Link
                  key={release.id}
                  to={`/releases/${release.id}`}
                  className="discover-release-card"
                >
                  <div className="discover-release-cover">
                    {release.cover_image_url ? (
                      <img
                        src={
                          release.cover_image_url
                        }
                        alt={`${release.title} cover`}
                      />
                    ) : (
                      <span>
                        {
                          release.release_type
                        }
                      </span>
                    )}
                  </div>

                  <div>
                    <span className="genre-pill">
                      {
                        release.release_type
                      }
                    </span>

                    <h4>
                      {release.title}
                    </h4>

                    <p>
                      {
                        release.artist_name
                      }
                    </p>
                  </div>
                </Link>
              ),
            )}
          </div>
        )}
      </section>

      <section className="section">
        <div className="section-heading">
          <div>
            <p className="eyebrow">
              ARTISTS
            </p>

            <h3>
              Popular artists
            </h3>
          </div>
        </div>

        {discovery.popular_artists.length ===
        0 ? (
          <section className="content-panel">
            <p>
              No artists available yet.
            </p>
          </section>
        ) : (
          <div className="discover-artist-grid">
            {discovery.popular_artists.map(
              (artist) => (
                <Link
                  key={artist.id}
                  to={`/artists/${artist.id}`}
                  className="discover-artist-card"
                >
                  <span className="discover-artist-avatar">
                    {artist.name
                      .charAt(0)
                      .toUpperCase()}
                  </span>

                  <strong>
                    {artist.name}
                  </strong>

                  <small>
                    View artist
                  </small>
                </Link>
              ),
            )}
          </div>
        )}
      </section>

      <section className="section">
        <div className="section-heading">
          <div>
            <p className="eyebrow">
              GENRES
            </p>

            <h3>
              Browse by genre
            </h3>
          </div>
        </div>

        {discovery.genres.length ===
        0 ? (
          <section className="content-panel">
            <p>
              No genres available yet.
            </p>
          </section>
        ) : (
          <div className="discover-genre-grid">
            {discovery.genres.map(
              (genre) => (
                <Link
                  key={genre.name}
                  to={`/search?q=${encodeURIComponent(
                    genre.name,
                  )}&type=track&genre=${encodeURIComponent(
                    genre.name,
                  )}`}
                  className="discover-genre-card"
                >
                  <strong>
                    {genre.name}
                  </strong>

                  <span>
                    {genre.track_count}{' '}
                    {genre.track_count ===
                    1
                      ? 'track'
                      : 'tracks'}
                  </span>
                </Link>
              ),
            )}
          </div>
        )}
      </section>
    </>
  )
}

export default DiscoverPage