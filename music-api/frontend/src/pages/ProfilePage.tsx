import {
  useState,
} from 'react'

import {
  Link,
  useLocation,
} from 'react-router-dom'

import { useAuth } from '../context/AuthContext'

type ProfileLocationState = {
  message?: string
}

function ProfilePage() {
  const location = useLocation()

  const {
    user,
    artist,
    isArtist,
    becomeArtist,
  } = useAuth()

  const locationState =
    location.state as ProfileLocationState | null

  const [error, setError] =
    useState('')

  const [success, setSuccess] =
    useState('')

  const [
    isCreatingArtist,
    setIsCreatingArtist,
  ] = useState(false)

  async function handleBecomeArtist() {
    setError('')
    setSuccess('')
    setIsCreatingArtist(true)

    try {
      await becomeArtist()

      setSuccess(
        'Your artist profile has been created successfully.',
      )
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : 'Unable to create artist profile.',
      )
    } finally {
      setIsCreatingArtist(false)
    }
  }

  return (
    <>
      <header className="page-header">
        <p className="eyebrow">
          ACCOUNT
        </p>

        <h2>Profile</h2>

        <p>
          Manage your Music account and
          artist access.
        </p>
      </header>

      {locationState?.message && (
        <div
          className="status-message"
          role="status"
        >
          {locationState.message}
        </div>
      )}

      {success && (
        <div
          className="status-message success-message"
          role="status"
        >
          {success}
        </div>
      )}

      {error && (
        <div
          className="form-error"
          role="alert"
        >
          {error}
        </div>
      )}

      <section className="content-panel">
        <p className="eyebrow">
          ACCOUNT INFORMATION
        </p>

        <h3>{user?.name}</h3>

        <p>{user?.email}</p>
      </section>

      {isArtist && artist ? (
        <section className="content-panel">
          <p className="eyebrow">
            ARTIST PROFILE
          </p>

          <h3>{artist.name}</h3>

          <p>
            Your account has artist access.
            You can upload and manage your
            music.
          </p>

          <div className="hero-actions">
            <Link
              to="/upload"
              className="primary-button"
            >
              Upload music
            </Link>

            <Link
              to="/my-music"
              className="secondary-button"
            >
              My Music
            </Link>
          </div>
        </section>
      ) : (
        <section className="content-panel">
          <p className="eyebrow">
            ARTIST ACCESS
          </p>

          <h3>Become an Artist</h3>

          <p>
            Your current account is a
            listener account. Create an
            artist profile when you are
            ready to publish your own music.
          </p>

          <button
            type="button"
            className="primary-button"
            onClick={handleBecomeArtist}
            disabled={isCreatingArtist}
          >
            {isCreatingArtist
              ? 'Creating artist profile...'
              : 'Become an Artist'}
          </button>
        </section>
      )}
    </>
  )
}

export default ProfilePage