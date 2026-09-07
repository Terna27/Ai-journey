import { useState } from 'react'

import { useAuth } from '../../context/AuthContext'
import { usePlaylists } from '../../context/PlaylistContext'

import { APIError } from '../../lib/api'

import type { Music } from '../../types/music'

type AddToPlaylistButtonProps = {
  track: Music
}

/*
 * Opens a small menu with the user's playlists so a song
 * can be added with one click, plus inline creation of a
 * new playlist that immediately receives the song.
 */
function AddToPlaylistButton({
  track,
}: AddToPlaylistButtonProps) {
  const {
    user,
    isAuthenticated,
  } = useAuth()

  const {
    playlists,
    isLoadingPlaylists,
    playlistsError,
    createPlaylist,
    addTrack,
    refreshPlaylists,
  } = usePlaylists()

  const [isOpen, setIsOpen] =
    useState(false)

  const [
    statusMessage,
    setStatusMessage,
  ] = useState('')

  const [isErrorMessage, setIsErrorMessage] =
    useState(false)

  const [pendingPlaylistID, setPendingPlaylistID] =
    useState<number | null>(null)

  const [isCreating, setIsCreating] =
    useState(false)

  const [newPlaylistName, setNewPlaylistName] =
    useState('')

  const [isVerified, setIsVerified] =
    useState(false)

  function resetMenu() {
    setStatusMessage('')
    setIsErrorMessage(false)
    setNewPlaylistName('')
    setIsCreating(false)
  }

  function openMenu() {
    if (!isAuthenticated) {
      return
    }

    resetMenu()
    setIsOpen(true)

    // Playlist endpoints require a verified email address.
    setIsVerified(
      Boolean(user?.email_verified),
    )
  }

  function closeMenu() {
    setIsOpen(false)
    resetMenu()
  }

  async function handleAddToExisting(
    playlistID: number,
  ) {
    if (pendingPlaylistID !== null) {
      return
    }

    try {
      setPendingPlaylistID(playlistID)
      setStatusMessage('')
      setIsErrorMessage(false)

      await addTrack(playlistID, track.id)

      setStatusMessage(
        `Added to playlist.`,
      )
      setIsErrorMessage(false)
    } catch (error) {
      if (
        error instanceof APIError &&
        error.code ===
          'TRACK_ALREADY_IN_PLAYLIST'
      ) {
        setStatusMessage(
          'This song is already in that playlist.',
        )
        setIsErrorMessage(false)
        return
      }

      setStatusMessage(
        error instanceof Error
          ? error.message
          : 'Failed to add song to playlist',
      )
      setIsErrorMessage(true)
    } finally {
      setPendingPlaylistID(null)
    }
  }

  async function handleCreateAndAdd() {
    const name = newPlaylistName.trim()

    if (isCreating || name === '') {
      return
    }

    try {
      setIsCreating(true)
      setStatusMessage('')
      setIsErrorMessage(false)

      const created =
        await createPlaylist({
          name,
          description: '',
          is_public: false,
        })

      await addTrack(
        created.id,
        track.id,
      )

      setStatusMessage(
        `Added to "${created.name}".`,
      )
      setIsErrorMessage(false)
      setNewPlaylistName('')
    } catch (error) {
      setStatusMessage(
        error instanceof Error
          ? error.message
          : 'Failed to create playlist',
      )
      setIsErrorMessage(true)
    } finally {
      setIsCreating(false)
    }
  }

  const hasPlaylists =
    playlists.length > 0

  return (
    <div className="add-to-playlist">
      <button
        type="button"
        className="secondary-button add-to-playlist-trigger"
        aria-label={`Add ${track.song_title} to a playlist`}
        onClick={openMenu}
      >
        + Playlist
      </button>

      {isOpen && (
        <div
          className="add-to-playlist-menu"
          role="menu"
          aria-label={`Add ${track.song_title} to playlist`}
        >
          <div className="add-to-playlist-header">
            <strong>
              Add to playlist
            </strong>

            <button
              type="button"
              className="text-button"
              onClick={closeMenu}
            >
              Close
            </button>
          </div>

          {!isVerified && (
            <p className="add-to-playlist-notice">
              Verify your email address
              before adding songs to
              playlists. Visit your
              profile to resend the
              verification email.
            </p>
          )}

          {isVerified && (
            <>
              {isLoadingPlaylists && (
                <p className="add-to-playlist-notice">
                  Loading your playlists...
                </p>
              )}

              {!isLoadingPlaylists &&
                playlistsError && (
                  <div className="add-to-playlist-notice">
                    <p>
                      {playlistsError}
                    </p>

                    <button
                      type="button"
                      className="text-button"
                      onClick={() =>
                        void refreshPlaylists()
                      }
                    >
                      Try again
                    </button>
                  </div>
                )}

              {!isLoadingPlaylists &&
                !playlistsError &&
                !hasPlaylists && (
                  <p className="add-to-playlist-notice">
                    You have no playlists
                    yet. Create one below.
                  </p>
                )}

              {hasPlaylists && (
                <ul className="add-to-playlist-list">
                  {playlists.map(
                    (playlist) => {
                      const isPending =
                        pendingPlaylistID ===
                        playlist.id

                      return (
                        <li key={playlist.id}>
                          <button
                            type="button"
                            disabled={
                              isPending ||
                              pendingPlaylistID !==
                                null
                            }
                            onClick={() =>
                              void handleAddToExisting(
                                playlist.id,
                              )
                            }
                          >
                            <span>
                              {
                                playlist.name
                              }
                            </span>

                            <small>
                              {
                                playlist.is_public
                                  ? 'Public'
                                  : 'Private'
                              }
                            </small>

                            {isPending && (
                              <em>
                                Adding...
                              </em>
                            )}
                          </button>
                        </li>
                      )
                    },
                  )}
                </ul>
              )}

              <div className="add-to-playlist-create">
                <input
                  type="text"
                  value={newPlaylistName}
                  placeholder="New playlist name"
                  maxLength={120}
                  disabled={isCreating}
                  onChange={(event) =>
                    setNewPlaylistName(
                      event.target
                        .value,
                    )
                  }
                />

                <button
                  type="button"
                  className="primary-button"
                  disabled={
                    isCreating ||
                    newPlaylistName.trim() ===
                      ''
                  }
                  onClick={() =>
                    void handleCreateAndAdd()
                  }
                >
                  {isCreating
                    ? 'Creating...'
                    : 'Create & add'}
                </button>
              </div>
            </>
          )}

          {statusMessage && (
            <p
              className={
                isErrorMessage
                  ? 'add-to-playlist-error'
                  : 'add-to-playlist-success'
              }
            >
              {statusMessage}
            </p>
          )}
        </div>
      )}
    </div>
  )
}

export default AddToPlaylistButton
