import {
  useCallback,
  useEffect,
  useState,
} from 'react'

import {
  Link,
  useNavigate,
  useParams,
} from 'react-router-dom'

import { useAuth } from '../context/AuthContext'
import { usePlaylists } from '../context/PlaylistContext'
import { usePlayer } from '../context/PlayerContext'

import AddToQueueButton from '../components/music/AddToQueueButton'

import { getPlaylist } from '../lib/api'

import type { Music } from '../types/music'
import type { Playlist } from '../types/playlist'

const NAME_MAX_LENGTH = 120

function PlaylistDetailsPage() {
  const { id } = useParams()

  const navigate = useNavigate()

  const {
    user,
    token,
    isAuthenticated,
    isLoadingIdentity,
  } = useAuth()

  const {
    updatePlaylist,
    deletePlaylist,
    removeTrack,
    refreshPlaylists,
  } = usePlaylists()

  const {
    currentTrack,
    isPlaying,
    playTrack,
  } = usePlayer()

  const [playlist, setPlaylist] =
    useState<Playlist | null>(null)

  const [tracks, setTracks] =
    useState<Music[]>([])

  const [isLoading, setIsLoading] =
    useState(true)

  const [loadError, setLoadError] =
    useState('')

  const [notFound, setNotFound] =
    useState(false)

  const [
    isEditing,
    setIsEditing,
  ] = useState(false)

  const [editName, setEditName] =
    useState('')

  const [editDescription, setEditDescription] =
    useState('')

  const [editIsPublic, setEditIsPublic] =
    useState(false)

  const [
    editValidationError,
    setEditValidationError,
  ] = useState('')

  const [editApiError, setEditApiError] =
    useState('')

  const [isSaving, setIsSaving] =
    useState(false)

  const [isDeleting, setIsDeleting] =
    useState(false)

  const [
    pendingTrackID,
    setPendingTrackID,
  ] = useState<number | null>(null)

  const [
    trackActionError,
    setTrackActionError,
  ] = useState('')

  const playlistID = Number(id)

  const isValidPlaylistID =
    Number.isInteger(
      playlistID,
    ) && playlistID > 0

  const isOwner =
    Boolean(
      playlist && user &&
      playlist.user_id === user.id,
    )

  const loadPlaylist =
    useCallback(async () => {
      if (
        !token ||
        !isValidPlaylistID
      ) {
        return
      }

      try {
        setIsLoading(true)
        setLoadError('')
        setNotFound(false)

        const details =
          await getPlaylist(
            playlistID,
            token,
          )

        setPlaylist(details.playlist)
        setTracks(details.tracks)
      } catch (error) {
        if (
          error instanceof Error &&
          error.message
            .toLowerCase()
            .includes('not found')
        ) {
          setNotFound(true)
        } else {
          setLoadError(
            error instanceof Error
              ? error.message
              : 'Failed to load playlist',
          )
        }
      } finally {
        setIsLoading(false)
      }
    }, [
      isValidPlaylistID,
      playlistID,
      token,
    ])

  useEffect(() => {
    if (isLoadingIdentity) {
      return
    }

    if (
      !isValidPlaylistID ||
      !isAuthenticated
    ) {
      return
    }

    void loadPlaylist()
  }, [
    isAuthenticated,
    isLoadingIdentity,
    isValidPlaylistID,
    loadPlaylist,
  ])

  function startEditing() {
    if (!playlist) {
      return
    }

    setEditName(playlist.name)
    setEditDescription(
      playlist.description,
    )
    setEditIsPublic(
      playlist.is_public,
    )
    setEditValidationError('')
    setEditApiError('')
    setIsEditing(true)
  }

  async function handleSave() {
    if (!playlist || isSaving) {
      return
    }

    const trimmedName =
      editName.trim()

    if (trimmedName === '') {
      setEditValidationError(
        'Please enter a playlist name.',
      )
      return
    }

    if (
      trimmedName.length >
      NAME_MAX_LENGTH
    ) {
      setEditValidationError(
        `Playlist names must be ${NAME_MAX_LENGTH} characters or fewer.`,
      )
      return
    }

    try {
      setIsSaving(true)
      setEditValidationError('')
      setEditApiError('')

      const updated =
        await updatePlaylist(
          playlist.id,
          {
            name: trimmedName,
            description:
              editDescription.trim(),
            is_public: editIsPublic,
          },
        )

      setPlaylist(updated)
      setIsEditing(false)
    } catch (error) {
      setEditApiError(
        error instanceof Error
          ? error.message
          : 'Failed to update playlist',
      )
    } finally {
      setIsSaving(false)
    }
  }

  async function handleDelete() {
    if (!playlist || isDeleting) {
      return
    }

    try {
      setIsDeleting(true)
      setTrackActionError('')

      await deletePlaylist(
        playlist.id,
      )

      void refreshPlaylists()
      navigate('/playlists')
    } catch (error) {
      setTrackActionError(
        error instanceof Error
          ? error.message
          : 'Failed to delete playlist',
      )
      setIsDeleting(false)
    }
  }

  async function handleRemoveTrack(
    track: Music,
  ) {
    if (
      !playlist ||
      pendingTrackID !== null
    ) {
      return
    }

    try {
      setPendingTrackID(track.id)
      setTrackActionError('')

      await removeTrack(
        playlist.id,
        track.id,
      )

      setTracks((current) =>
        current.filter(
          (item) =>
            item.id !== track.id,
        ),
      )
    } catch (error) {
      setTrackActionError(
        error instanceof Error
          ? error.message
          : 'Failed to remove song',
      )
    } finally {
      setPendingTrackID(null)
    }
  }

  if (isLoadingIdentity) {
    return (
      <section className="content-panel">
        <p>
          Loading your account...
        </p>
      </section>
    )
  }

  if (!isAuthenticated) {
    return (
      <section className="content-panel">
        <h3>
          Please log in
        </h3>

        <p>
          <Link to="/login">
            Log in
          </Link>{' '}
          to view playlists.
        </p>
      </section>
    )
  }

  if (!isValidPlaylistID) {
    return (
      <section className="content-panel">
        <h3>
          Playlist not found
        </h3>

        <p>
          That playlist link does
          not look right.{' '}
          <Link to="/playlists">
            Back to playlists
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
          Loading playlist...
        </p>
      </section>
    )
  }

  if (notFound) {
    return (
      <section className="content-panel">
        <h3>
          Playlist not found
        </h3>

        <p>
          This playlist does not
          exist or is private to
          another listener.{' '}
          <Link to="/playlists">
            Back to playlists
          </Link>
          .
        </p>
      </section>
    )
  }

  if (loadError) {
    return (
      <section className="content-panel">
        <p>{loadError}</p>

        <button
          type="button"
          className="text-button"
          onClick={() =>
            void loadPlaylist()
          }
        >
          Try again
        </button>
      </section>
    )
  }

  if (!playlist) {
    return (
      <section className="content-panel">
        <p>
          Loading playlist...
        </p>
      </section>
    )
  }

  return (
    <>
      <header className="topbar">
        <div>
          <p className="eyebrow">
            PLAYLIST
          </p>

          {isEditing ? (
            <input
              type="text"
              className="playlist-edit-title"
              value={editName}
              maxLength={NAME_MAX_LENGTH}
              disabled={isSaving}
              onChange={(event) =>
                setEditName(
                  event.target
                    .value,
                )
              }
            />
          ) : (
            <h2>
              {playlist.name}
            </h2>
          )}

          <div className="playlist-meta">
            <span className="genre-pill">
              {playlist.is_public
                ? 'Public'
                : 'Private'}
            </span>

            <span className="text-button">
              {tracks.length}{' '}
              {tracks.length === 1
                ? 'track'
                : 'tracks'}
            </span>
          </div>

          {playlist.description &&
            !isEditing && (
              <p className="playlist-description">
                {
                  playlist.description
                }
              </p>
            )}
        </div>

        <div className="topbar-actions">
          {isOwner &&
            !isEditing && (
              <button
                type="button"
                className="secondary-button"
                onClick={startEditing}
              >
                Edit
              </button>
            )}

          {isOwner &&
            isEditing && (
              <>
                <button
                  type="button"
                  className="secondary-button"
                  disabled={isSaving}
                  onClick={() =>
                    setIsEditing(
                      false,
                    )
                  }
                >
                  Cancel
                </button>

                <button
                  type="button"
                  className="primary-button"
                  disabled={isSaving}
                  onClick={() =>
                    void handleSave()
                  }
                >
                  {isSaving
                    ? 'Saving...'
                    : 'Save changes'}
                </button>

                <button
                  type="button"
                  className="logout-button"
                  disabled={
                    isDeleting
                  }
                  onClick={() =>
                    void handleDelete()
                  }
                >
                  {isDeleting
                    ? 'Deleting...'
                    : 'Delete playlist'}
                </button>
              </>
            )}
        </div>
      </header>

      {isEditing && (
        <section className="content-panel playlist-edit-panel">
          <form
            className="auth-form"
            onSubmit={(event) => {
              event.preventDefault()
              void handleSave()
            }}
          >
            <label>
              Description

              <textarea
                value={editDescription}
                placeholder="What is this playlist about?"
                rows={3}
                disabled={isSaving}
                onChange={(event) =>
                  setEditDescription(
                    event
                      .target
                      .value,
                  )
                }
              />
            </label>

            <label>
              Visibility

              <select
                value={
                  editIsPublic
                    ? 'public'
                    : 'private'
                }
                disabled={isSaving}
                onChange={(event) =>
                  setEditIsPublic(
                    event
                      .target
                      .value ===
                    'public',
                  )
                }
              >
                <option value="private">
                  Private
                </option>

                <option value="public">
                  Public
                </option>
              </select>
            </label>

            <button
              type="submit"
              className="form-submit"
              disabled={isSaving}
            >
              {isSaving
                ? 'Saving...'
                : 'Save changes'}
            </button>
          </form>

          {editValidationError && (
            <p className="form-error">
              {editValidationError}
            </p>
          )}

          {editApiError && (
            <p className="form-error">
              {editApiError}
            </p>
          )}
        </section>
      )}

      <section className="section">
        {trackActionError && (
          <section className="content-panel">
            <p>
              {trackActionError}
            </p>
          </section>
        )}

        {tracks.length === 0 && (
          <section className="content-panel">
            <h3>
              This playlist is empty
            </h3>

            <p>
              Add songs from the{' '}
              <Link to="/">
                home page
              </Link>{' '}
              or your{' '}
              <Link to="/liked">
                liked songs
              </Link>
              .
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

                    {track.artist_id ? (
                      <Link
                        to={`/artists/${track.artist_id}`}
                        className="artist-link"
                      >
                        {track.artist_name}
                      </Link>
                    ) : (
                      <span>
                        {track.artist_name}
                      </span>
                    )}

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
                    onClick={() =>
                      // Playing from the playlist queues the
                      // playlist's track order.
                      playTrack(
                        track,
                        tracks,
                      )
                    }
                  >
                    {isThisTrackPlaying ? '❚❚' : '▶'}
                  </button>

                  <AddToQueueButton
                    track={track}
                    variant="full"
                  />

                  {isOwner && (
                    <button
                      type="button"
                      className="logout-button playlist-remove-button"
                      aria-label={`Remove ${track.song_title} from playlist`}
                      disabled={
                        isPending ||
                        pendingTrackID !==
                        null
                      }
                      onClick={() =>
                        void handleRemoveTrack(
                          track,
                        )
                      }
                    >
                      {isPending
                        ? '...'
                        : 'Remove'}
                    </button>
                  )}
                </article>
              )
            })}
          </div>
        )}
      </section>
    </>
  )
}

export default PlaylistDetailsPage
