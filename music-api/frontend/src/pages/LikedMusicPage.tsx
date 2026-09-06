import { useState } from 'react'

import { useLibrary } from '../context/LibraryContext'
import { usePlayer } from '../context/PlayerContext'

import type { Music } from '../types/music'

function LikedMusicPage() {
  const {
    likedMusic,
    isLoadingLibrary,
    libraryError,
    toggleLike,
  } = useLibrary()

  const {
    currentTrack,
    isPlaying,
    playTrack,
  } = usePlayer()

  const [
    pendingTrackID,
    setPendingTrackID,
  ] = useState<number | null>(null)

  async function handleUnlike(
    track: Music,
  ) {
    if (pendingTrackID !== null) {
      return
    }

    try {
      setPendingTrackID(track.id)

      await toggleLike(track)
    } catch {
      // LibraryContext exposes the error.
    } finally {
      setPendingTrackID(null)
    }
  }

  return (
    <>
      <header className="topbar">
        <div>
          <p className="eyebrow">
            YOUR LIBRARY
          </p>

          <h2>Liked Songs</h2>
        </div>

        <span className="text-button">
          {likedMusic.length}{' '}
          {likedMusic.length === 1
            ? 'track'
            : 'tracks'}
        </span>
      </header>

      <section className="section">
        {isLoadingLibrary && (
          <section className="content-panel">
            <p>
              Loading your liked songs...
            </p>
          </section>
        )}

        {!isLoadingLibrary &&
          libraryError && (
            <section className="content-panel">
              <p>{libraryError}</p>
            </section>
          )}

        {!isLoadingLibrary &&
          !libraryError &&
          likedMusic.length === 0 && (
            <section className="content-panel">
              <h3>
                Your library is empty
              </h3>

              <p>
                Like songs you enjoy and
                they will appear here.
              </p>
            </section>
          )}

        {!isLoadingLibrary &&
          likedMusic.length > 0 && (
            <div className="library-list">
              {likedMusic.map(
                (track) => {
                  const isCurrentTrack =
                    currentTrack?.id ===
                    track.id

                  const isThisTrackPlaying =
                    isCurrentTrack &&
                    isPlaying

                  const isPending =
                    pendingTrackID ===
                    track.id

                  return (
                    <article
                      key={track.id}
                      className="library-track"
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
                        onClick={() =>
                          playTrack(track)
                        }
                      >
                        {isThisTrackPlaying
                          ? 'Pause'
                          : 'Play'}
                      </button>

                      <button
                        type="button"
                        className="like-button is-liked"
                        aria-label={`Unlike ${track.song_title}`}
                        disabled={isPending}
                        onClick={() =>
                          void handleUnlike(
                            track,
                          )
                        }
                      >
                        {isPending
                          ? '...'
                          : '♥'}
                      </button>
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

export default LikedMusicPage