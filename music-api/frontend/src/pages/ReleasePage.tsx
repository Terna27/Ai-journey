import {
  useEffect,
  useState,
} from 'react'

import {
  Link,
  useNavigate,
  useParams,
} from 'react-router-dom'

import AddToPlaylistButton from '../components/music/AddToPlaylistButton'
import AddToQueueButton from '../components/music/AddToQueueButton'

import { useAuth } from '../context/AuthContext'
import { useLibrary } from '../context/LibraryContext'
import { usePlayer } from '../context/PlayerContext'

import {
  APIError,
  getRelease,
} from '../lib/api'

import type { Music } from '../types/music'
import type {
  ReleaseDetails,
} from '../types/release'

function formatReleaseDate(
  value: string | null,
) {
  if (!value) {
    return ''
  }

  const date = new Date(value)

  if (Number.isNaN(date.getTime())) {
    return ''
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

function ReleasePage() {
  const { id } = useParams()

  const navigate = useNavigate()

  const {
    isAuthenticated,
  } = useAuth()

  const {
    currentTrack,
    isPlaying,
    playTrack,
  } = usePlayer()

  const {
    isLiked,
    toggleLike,
  } = useLibrary()

  const [
    details,
    setDetails,
  ] = useState<ReleaseDetails | null>(
    null,
  )

  const [
    isLoading,
    setIsLoading,
  ] = useState(true)

  const [error, setError] =
    useState('')

  const [
    notFound,
    setNotFound,
  ] = useState(false)

  const [
    pendingLikeID,
    setPendingLikeID,
  ] = useState<number | null>(null)

  const releaseID = Number(id)

  const isValidReleaseID =
    Number.isInteger(releaseID) &&
    releaseID > 0

  useEffect(() => {
    if (!isValidReleaseID) {
      setIsLoading(false)
      return
    }

    let cancelled = false

    async function loadRelease() {
      try {
        setIsLoading(true)
        setError('')
        setNotFound(false)

        const response =
          await getRelease(
            releaseID,
          )

        if (!cancelled) {
          setDetails(response)
        }
      } catch (err) {
        if (cancelled) {
          return
        }

        if (
          err instanceof APIError &&
          err.status === 404
        ) {
          setNotFound(true)
          return
        }

        setError(
          err instanceof Error
            ? err.message
            : 'Failed to load release',
        )
      } finally {
        if (!cancelled) {
          setIsLoading(false)
        }
      }
    }

    void loadRelease()

    return () => {
      cancelled = true
    }
  }, [
    releaseID,
    isValidReleaseID,
  ])

  function handlePlay(
    track: Music,
  ) {
    if (!isAuthenticated) {
      navigate(
        '/login',
        {
          state: {
            message:
              'Please log in to play music.',
            from:
              `/releases/${releaseID}`,
          },
        },
      )

      return
    }

    if (!details) {
      return
    }

    playTrack(
      track,
      details.tracks,
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
            from:
              `/releases/${releaseID}`,
          },
        },
      )

      return
    }

    if (pendingLikeID !== null) {
      return
    }

    try {
      setPendingLikeID(track.id)

      await toggleLike(track)
    } catch {
      // LibraryContext exposes
      // the library error state.
    } finally {
      setPendingLikeID(null)
    }
  }

  if (!isValidReleaseID) {
    return (
      <section className="content-panel">
        <h3>
          Release not found
        </h3>

        <p>
          That release link does not
          look right.{' '}
          <Link to="/">
            Back to music
          </Link>
          .
        </p>
      </section>
    )
  }

  if (isLoading) {
    return (
      <section className="content-panel">
        <p>
          Loading release...
        </p>
      </section>
    )
  }

  if (notFound) {
    return (
      <section className="content-panel">
        <h3>
          Release not found
        </h3>

        <p>
          This release does not exist
          or is not published.{' '}
          <Link to="/">
            Back to music
          </Link>
          .
        </p>
      </section>
    )
  }

  if (error) {
    return (
      <section className="content-panel">
        <p>{error}</p>
      </section>
    )
  }

  if (!details) {
    return null
  }

  const {
    release,
    tracks,
  } = details

  const artistName =
    tracks[0]?.artist_name ??
    'Artist'

  const releaseDate =
    formatReleaseDate(
      release.release_date,
    )

  return (
    <>
      <header className="release-page-header">
        <div className="release-page-cover">
          {release.cover_image_url ? (
            <img
              src={
                release.cover_image_url
              }
              alt={`${release.title} cover`}
            />
          ) : tracks[0]?.image_url ? (
            <img
              src={
                tracks[0].image_url
              }
              alt={`${release.title} cover`}
            />
          ) : (
            <div className="release-cover-fallback">
              ♪
            </div>
          )}
        </div>

        <div className="release-page-info">
          <p className="eyebrow">
            {release.release_type}
          </p>

          <h2>
            {release.title}
          </h2>

          <Link
            to={`/artists/${release.artist_id}`}
            className="artist-link"
          >
            {artistName}
          </Link>

          <div className="release-meta">
            {releaseDate && (
              <span>
                {releaseDate}
              </span>
            )}

            <span>
              {tracks.length}{' '}
              {tracks.length === 1
                ? 'track'
                : 'tracks'}
            </span>
          </div>

          {release.description && (
            <p className="release-description">
              {release.description}
            </p>
          )}
        </div>
      </header>

      <section className="section">
        <div className="section-heading">
          <div>
            <p className="eyebrow">
              TRACKLIST
            </p>

            <h3>
              {release.title}
            </h3>
          </div>
        </div>

        {tracks.length === 0 ? (
          <section className="content-panel">
            <p>
              This release does not
              contain any tracks yet.
            </p>
          </section>
        ) : (
          <div className="library-list">
            {tracks.map((track) => {
              const isCurrentTrack =
                currentTrack?.id ===
                track.id

              const isThisTrackPlaying =
                isCurrentTrack &&
                isPlaying

              const liked =
                isLiked(track.id)

              const isPendingLike =
                pendingLikeID ===
                track.id

              return (
                <article
                  className="library-track release-track"
                  key={track.id}
                >
                  <span className="release-track-number">
                    {track.track_number ??
                      '—'}
                  </span>

                  <div className="library-track-cover">
                    {track.image_url ? (
                      <img
                        src={
                          track.image_url
                        }
                        alt={`${track.song_title} cover`}
                      />
                    ) : (
                      <span>♪</span>
                    )}
                  </div>

                  <div className="library-track-info">
                    <strong>
                      {
                        track.song_title
                      }
                    </strong>

                    <Link
                      to={`/artists/${release.artist_id}`}
                      className="artist-link"
                    >
                      {
                        track.artist_name
                      }
                    </Link>

                    {isCurrentTrack && (
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

                  <button
                    type="button"
                    className="library-play-button"
                    disabled={
                      !track.audio_url
                    }
                    aria-label={
                      isThisTrackPlaying
                        ? `Pause ${track.song_title}`
                        : `Play ${track.song_title}`
                    }
                    onClick={() =>
                      handlePlay(track)
                    }
                  >
                    {isThisTrackPlaying
                      ? '❚❚'
                      : '▶'}
                  </button>

                  {isAuthenticated && (
                    <AddToQueueButton
                      track={track}
                      variant="full"
                    />
                  )}

                  {isAuthenticated && (
                    <AddToPlaylistButton
                      track={track}
                    />
                  )}

                  <button
                    type="button"
                    className={
                      liked
                        ? 'like-button is-liked'
                        : 'like-button'
                    }
                    aria-label={
                      liked
                        ? `Unlike ${track.song_title}`
                        : `Like ${track.song_title}`
                    }
                    disabled={
                      isPendingLike
                    }
                    onClick={() =>
                      void handleLike(
                        track,
                      )
                    }
                  >
                    {isPendingLike
                      ? '...'
                      : liked
                        ? '♥'
                        : '♡'}
                  </button>
                </article>
              )
            })}
          </div>
        )}
      </section>
    </>
  )
}

export default ReleasePage