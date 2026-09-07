import { useState } from 'react'

import { Link } from 'react-router-dom'

import { useAuth } from '../context/AuthContext'
import { usePlaylists } from '../context/PlaylistContext'

const NAME_MAX_LENGTH = 120

function PlaylistsPage() {
  const {
    user,
    isLoadingIdentity,
  } = useAuth()

  const {
    playlists,
    isLoadingPlaylists,
    playlistsError,
    createPlaylist,
    refreshPlaylists,
  } = usePlaylists()

  const [name, setName] =
    useState('')

  const [description, setDescription] =
    useState('')

  const [isPublic, setIsPublic] =
    useState(false)

  const [
    validationError,
    setValidationError,
  ] = useState('')

  const [apiError, setApiError] =
    useState('')

  const [successMessage, setSuccessMessage] =
    useState('')

  const [isCreating, setIsCreating] =
    useState(false)

  const isVerified =
    Boolean(user?.email_verified)

  async function handleCreate() {
    if (isCreating) {
      return
    }

    const trimmedName =
      name.trim()

    if (trimmedName === '') {
      setValidationError(
        'Please enter a playlist name.',
      )
      setSuccessMessage('')
      return
    }

    if (
      trimmedName.length >
      NAME_MAX_LENGTH
    ) {
      setValidationError(
        `Playlist names must be ${NAME_MAX_LENGTH} characters or fewer.`,
      )
      setSuccessMessage('')
      return
    }

    try {
      setIsCreating(true)
      setValidationError('')
      setApiError('')
      setSuccessMessage('')

      const created =
        await createPlaylist({
          name: trimmedName,
          description:
            description.trim(),
          is_public: isPublic,
        })

      setSuccessMessage(
        `Created "${created.name}".`,
      )

      setName('')
      setDescription('')
      setIsPublic(false)
    } catch (error) {
      setApiError(
        error instanceof Error
          ? error.message
          : 'Failed to create playlist',
      )
    } finally {
      setIsCreating(false)
    }
  }

  const isLoading =
    isLoadingIdentity ||
    isLoadingPlaylists

  return (
    <>
      <header className="topbar">
        <div>
          <p className="eyebrow">
            YOUR LIBRARY
          </p>

          <h2>Playlists</h2>
        </div>

        <span className="text-button">
          {playlists.length}{' '}
          {playlists.length === 1
            ? 'playlist'
            : 'playlists'}
        </span>
      </header>

      {isLoading && (
        <section className="content-panel">
          <p>
            Loading your playlists...
          </p>
        </section>
      )}

      {!isLoading && !isVerified && (
        <section className="content-panel">
          <h3>
            Verify your email address
          </h3>

          <p>
            Playlists need a verified
            email address. Visit your{' '}
            <Link to="/profile">
              profile
            </Link>{' '}
            to resend the verification
            email.
          </p>
        </section>
      )}

      {!isLoading &&
        isVerified &&
        playlistsError && (
          <section className="content-panel">
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
          </section>
        )}

      {!isLoading && isVerified && (
        <section className="section">
          <div className="playlist-layout">
            <div className="playlist-collection">
              {playlists.length === 0 && (
                <section className="content-panel">
                  <h3>
                    No playlists yet
                  </h3>

                  <p>
                    Create your first
                    playlist and add
                    songs you love from
                    the home page or
                    your liked songs.
                  </p>
                </section>
              )}

              {playlists.length > 0 && (
                <div className="playlist-grid">
                  {playlists.map(
                    (playlist) => (
                      <Link
                        key={playlist.id}
                        to={`/playlists/${playlist.id}`}
                        className="playlist-card"
                      >
                        <div className="playlist-card-cover">
                          <span>
                            ♫
                          </span>
                        </div>

                        <div className="playlist-card-content">
                          <strong>
                            {
                              playlist.name
                            }
                          </strong>

                          <span>
                            {
                              playlist.is_public
                                ? 'Public'
                                : 'Private'
                            }
                          </span>

                          {playlist
                            .description && (
                            <small>
                              {
                                playlist.description
                              }
                            </small>
                          )}
                        </div>
                      </Link>
                    ),
                  )}
                </div>
              )}
            </div>

            <aside className="playlist-create-card">
              <h3>
                Create a playlist
              </h3>

              <form
                className="auth-form"
                onSubmit={(event) => {
                  event.preventDefault()
                  void handleCreate()
                }}
              >
                <label>
                  Name

                  <input
                    type="text"
                    value={name}
                    placeholder="e.g. Weekend Mix"
                    maxLength={
                      NAME_MAX_LENGTH
                    }
                    disabled={isCreating}
                    onChange={(event) =>
                      setName(
                        event.target
                          .value,
                      )
                    }
                  />
                </label>

                <label>
                  Description
                  (optional)

                  <textarea
                    value={description}
                    placeholder="What is this playlist about?"
                    rows={3}
                    disabled={isCreating}
                    onChange={(event) =>
                      setDescription(
                        event.target
                          .value,
                      )
                    }
                  />
                </label>

                <label>
                  Visibility

                  <select
                    value={
                      isPublic
                        ? 'public'
                        : 'private'
                    }
                    disabled={isCreating}
                    onChange={(event) =>
                      setIsPublic(
                        event.target
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
                  disabled={isCreating}
                >
                  {isCreating
                    ? 'Creating...'
                    : 'Create playlist'}
                </button>
              </form>

              {validationError && (
                <p className="form-error">
                  {validationError}
                </p>
              )}

              {apiError && (
                <p className="form-error">
                  {apiError}
                </p>
              )}

              {successMessage && (
                <p className="playlist-success-message">
                  {successMessage}
                </p>
              )}
            </aside>
          </div>
        </section>
      )}
    </>
  )
}

export default PlaylistsPage
