import {
  useState,
} from 'react'

import {
  Link,
  useLocation,
} from 'react-router-dom'

import { useAuth } from '../context/AuthContext'
import { resendVerification } from '../lib/api'

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

  const [
    isResendingVerification,
    setIsResendingVerification,
  ] = useState(false)

  async function handleBecomeArtist() {
    if (!user?.email_verified) {
      setError(
        'Verify your email address before creating an artist profile.',
      )
      return
    }

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

  async function handleResendVerification() {
    if (!user?.email) {
      return
    }

    setError('')
    setSuccess('')
    setIsResendingVerification(true)

    try {
      const result =
        await resendVerification({
          email: user.email,
        })

      setSuccess(
        result.message,
      )
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : 'Unable to send verification email.',
      )
    } finally {
      setIsResendingVerification(false)
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

        {user?.email_verified ? (
          <div
            className="status-message success-message"
            role="status"
          >
            Email verified
          </div>
        ) : (
          <>
            <div
              className="form-error"
              role="alert"
            >
              Your email address has not
              been verified yet.
            </div>

            <p>
              Verify your email before
              creating an artist profile or
              using protected account
              features.
            </p>

            <button
              type="button"
              className="secondary-button"
              onClick={
                handleResendVerification
              }
              disabled={
                isResendingVerification
              }
            >
              {isResendingVerification
                ? 'Sending verification email...'
                : 'Resend verification email'}
            </button>
          </>
        )}
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

          {user?.email_verified ? (
            <>
              <p>
                Your current account is a
                listener account. Create an
                artist profile when you are
                ready to publish your own
                music.
              </p>

              <button
                type="button"
                className="primary-button"
                onClick={
                  handleBecomeArtist
                }
                disabled={
                  isCreatingArtist
                }
              >
                {isCreatingArtist
                  ? 'Creating artist profile...'
                  : 'Become an Artist'}
              </button>
            </>
          ) : (
            <>
              <p>
                You need to verify your
                email address before you
                can create an artist
                profile.
              </p>

              <button
                type="button"
                className="primary-button"
                disabled
              >
                Verify email first
              </button>
            </>
          )}
        </section>
      )}
    </>
  )
}

export default ProfilePage