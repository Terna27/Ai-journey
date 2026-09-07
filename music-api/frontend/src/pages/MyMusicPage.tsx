import {
  useEffect,
  useMemo,
  useState,
} from 'react'

import {
  Link,
  useLocation,
} from 'react-router-dom'

import { useAuth } from '../context/AuthContext'
import { usePlayer } from '../context/PlayerContext'
import { getMusic } from '../lib/api'
import type { Music } from '../types/music'

type LocationState = {
  message?: string
}

function formatDate(value: string) {
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

function MyMusicPage() {
  const { artist } = useAuth()

  const {
    currentTrack,
    isPlaying,
    playTrack,
  } = usePlayer()

  const location = useLocation()

  const [music, setMusic] =
    useState<Music[]>([])

  const [loading, setLoading] =
    useState(true)

  const [error, setError] =
    useState('')

  const locationState =
    location.state as LocationState | null

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
              : 'Failed to load your music',
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

  const myMusic = useMemo(() => {
    if (!artist) {
      return []
    }

    return music.filter(
      (track) =>
        track.artist_id ===
        artist.id,
    )
  }, [artist, music])

  return (
    <>
      <header className="page-header">
        <p className="eyebrow">
          YOUR LIBRARY
        </p>

        <h2>My Music</h2>

        <p>
          Manage and listen to the songs
          you have uploaded.
        </p>
      </header>

      {locationState?.message && (
        <div
          className="status-message success-message"
          role="status"
        >
          {locationState.message}
        </div>
      )}

      {loading && (
        <section className="content-panel">
          <p>
            Loading your music...
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
        myMusic.length === 0 && (
          <section className="content-panel">
            <p>
              You have not uploaded any
              music yet.
            </p>
          </section>
        )}

      {!loading &&
        !error &&
        myMusic.length > 0 && (
          <section className="my-music-grid">
            {myMusic.map(
              (track) => {
                const isCurrentTrack =
                  currentTrack?.id ===
                  track.id

                const isThisTrackPlaying =
                  isCurrentTrack &&
                  isPlaying

                return (
                  <article
                    className={
                      isCurrentTrack
                        ? 'my-music-card playing'
                        : 'my-music-card'
                    }
                    key={track.id}
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
                              myMusic,
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
                          {track.genre}
                        </span>
                      </div>

                      <p className="music-upload-date">
                        Uploaded{' '}
                        {formatDate(
                          track.date_posted,
                        )}
                      </p>

                      {isThisTrackPlaying && (
                        <p className="music-now-playing">
                          Now playing
                        </p>
                      )}

                      {!track.audio_url && (
                        <p className="music-unavailable">
                          Audio unavailable
                        </p>
                      )}
                    </div>
                  </article>
                )
              },
            )}
          </section>
        )}
    </>
  )
}

export default MyMusicPage