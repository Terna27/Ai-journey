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
  getArtistProfile,
} from '../lib/api'

import type {
  PublicArtist,
} from '../types/artist'

import type { Music } from '../types/music'

function ArtistPage() {
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

  const [artist, setArtist] =
    useState<PublicArtist | null>(null)

  const [tracks, setTracks] =
    useState<Music[]>([])

  const [isLoading, setIsLoading] =
    useState(true)

  const [error, setError] =
    useState('')

  const [notFound, setNotFound] =
    useState(false)

  const [
    pendingLikeID,
    setPendingLikeID,
  ] = useState<number | null>(null)

  const artistID = Number(id)

  const isValidArtistID =
    Number.isInteger(artistID) &&
    artistID > 0

  useEffect(() => {
    if (!isValidArtistID) {
      setIsLoading(false)
      return
    }

    let cancelled = false

    async function loadArtist() {
      try {
        setIsLoading(true)
        setError('')
        setNotFound(false)

        const response =
          await getArtistProfile(
            artistID,
          )

        if (cancelled) {
          return
        }

        setArtist(response.artist)
        setTracks(response.tracks)
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
            : 'Failed to load artist',
        )
      } finally {
        if (!cancelled) {
          setIsLoading(false)
        }
      }
    }

    void loadArtist()

    return () => {
      cancelled = true
    }
  }, [
    artistID,
    isValidArtistID,
  ])

  function handlePlay(track: Music) {
    if (!isAuthenticated) {
      navigate('/login', {
        state: {
          message:
            'Please log in to play music.',
          from: `/artists/${artistID}`,
        },
      })

      return
    }

    playTrack(
      track,
      tracks,
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
          from: `/artists/${artistID}`,
        },
      })

      return
    }

    if (pendingLikeID !== null) {
      return
    }

    try {
      setPendingLikeID(track.id)

      await toggleLike(track)
    } catch {
      // LibraryContext already exposes
      // the library error state.
    } finally {
      setPendingLikeID(null)
    }
  }

  if (!isValidArtistID) {
    return (
      <section className="content-panel">
        <h3>Artist not found</h3>

        <p>
          That artist link does not
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
          Loading artist...
        </p>
      </section>
    )
  }

  if (notFound) {
    return (
      <section className="content-panel">
        <h3>Artist not found</h3>

        <p>
          This artist does not exist.{' '}
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

  if (!artist) {
    return null
  }

  return (
    <>
      <header className="artist-page-header">
        <div className="artist-page-avatar">
          {artist.name
            .charAt(0)
            .toUpperCase()}
        </div>

        <div>
          <p className="eyebrow">
            ARTIST
          </p>

          <h2>{artist.name}</h2>

          <p>
            {tracks.length}{' '}
            {tracks.length === 1
              ? 'release'
              : 'releases'}
          </p>
        </div>
      </header>

      <section className="section">
        <div className="section-heading">
          <div>
            <p className="eyebrow">
              MUSIC
            </p>

            <h3>
              Releases
            </h3>
          </div>
        </div>

        {tracks.length === 0 && (
          <section className="content-panel">
            <p>
              This artist has not
              released any music yet.
            </p>
          </section>
        )}

        {tracks.length > 0 && (
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
                  className="library-track"
                  key={track.id}
                >
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

                    <span>
                      {
                        track.artist_name
                      }
                    </span>

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

export default ArtistPage