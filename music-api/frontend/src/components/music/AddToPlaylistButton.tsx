import {
  useEffect,
  useId,
  useState,
} from 'react'

import { useAuth } from '../../context/AuthContext'
import { usePlaylists } from '../../context/PlaylistContext'

import { APIError } from '../../lib/api'

import type { Music } from '../../types/music'

type AddToPlaylistButtonProps = {
  track: Music
}

const PLAYLIST_MENU_OPEN_EVENT =
  'music:add-to-playlist-open'

function AddToPlaylistButton({
  track,
}: AddToPlaylistButtonProps) {
  const instanceID = useId()

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

  const [
    isErrorMessage,
    setIsErrorMessage,
  ] = useState(false)

  const [
    pendingPlaylistID,
    setPendingPlaylistID,
  ] = useState<number | null>(null)

  const [
    isCreating,
    setIsCreating,
  ] = useState(false)

  const [
    newPlaylistName,
    setNewPlaylistName,
  ] = useState('')

  const isVerified =
    Boolean(user?.email_verified)

  const hasPlaylists =
    playlists.length > 0

  function resetMenu() {
    setStatusMessage('')
    setIsErrorMessage(false)
    setNewPlaylistName('')
    setIsCreating(false)
  }

  function closeMenu() {
    setIsOpen(false)
    resetMenu()
  }

  function openMenu() {
    if (!isAuthenticated) {
      return
    }

    resetMenu()

    window.dispatchEvent(
      new CustomEvent(
        PLAYLIST_MENU_OPEN_EVENT,
        {
          detail: instanceID,
        },
      ),
    )

    setIsOpen(true)
  }

  function handleToggle() {
    if (isOpen) {
      closeMenu()
      return
    }

    openMenu()
  }

  useEffect(() => {
    function handleAnotherMenuOpened(
      event: Event,
    ) {
      const customEvent =
        event as CustomEvent<string>

      if (
        customEvent.detail ===
        instanceID
      ) {
        return
      }

      setIsOpen(false)
      resetMenu()
    }

    window.addEventListener(
      PLAYLIST_MENU_OPEN_EVENT,
      handleAnotherMenuOpened,
    )

    return () => {
      window.removeEventListener(
        PLAYLIST_MENU_OPEN_EVENT,
        handleAnotherMenuOpened,
      )
    }
  }, [instanceID])

  async function handleAddToExisting(
    playlistID: number,
  ) {
    if (
      pendingPlaylistID !== null
    ) {
      return
    }

    try {
      setPendingPlaylistID(
        playlistID,
      )

      setStatusMessage('')
      setIsErrorMessage(false)

      await addTrack(
        playlistID,
        track.id,
      )

      closeMenu()
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
    const name =
      newPlaylistName.trim()

    if (
      isCreating ||
      name === ''
    ) {
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

      closeMenu()
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

  return (
    <div className="add-to-playlist">
      <button
        type="button"
        className="text-button add-to-playlist-trigger"
        aria-label={`Add ${track.song_title} to a playlist`}
        aria-expanded={isOpen}
        onClick={handleToggle}
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
                        <li
                          key={playlist.id}
                        >
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
                  value={
                    newPlaylistName
                  }
                  placeholder="New playlist name"
                  maxLength={120}
                  disabled={
                    isCreating
                  }
                  onChange={(event) =>
                    setNewPlaylistName(
                      event.target.value,
                    )
                  }
                />

                <button
                  type="button"
                  className="primary-button"
                  disabled={
                    isCreating ||
                    newPlaylistName
                      .trim() === ''
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