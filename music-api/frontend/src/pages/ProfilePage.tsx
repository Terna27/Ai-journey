import {
  useEffect,
  useState,
} from 'react'

import {
  Link,
  useLocation,
} from 'react-router-dom'

import { useAuth } from '../context/AuthContext'

import {
  resendVerification,
  updateArtistProfile,
} from '../lib/api'

type ProfileLocationState = {
  message?: string
}

const MAX_IMAGE_SIZE =
  10 * 1024 * 1024

const MAX_VIDEO_SIZE =
  50 * 1024 * 1024

const ALLOWED_IMAGE_TYPES = [
  'image/jpeg',
  'image/png',
  'image/webp',
]

const ALLOWED_VIDEO_TYPES = [
  'video/mp4',
  'video/webm',
  'video/quicktime',
]

function useObjectURL(
  file: File | null,
) {
  const [url, setURL] =
    useState<string | null>(null)

  useEffect(() => {
    if (!file) {
      setURL(null)
      return
    }

    const objectURL =
      URL.createObjectURL(file)

    setURL(objectURL)

    return () => {
      URL.revokeObjectURL(
        objectURL,
      )
    }
  }, [file])

  return url
}

function validateImage(
  file: File,
) {
  if (
    !ALLOWED_IMAGE_TYPES.includes(
      file.type,
    )
  ) {
    return (
      'Image must be JPEG, PNG, or WebP.'
    )
  }

  if (file.size > MAX_IMAGE_SIZE) {
    return (
      'Image must not exceed 10 MB.'
    )
  }

  return ''
}

function validateVideo(
  file: File,
) {
  if (
    !ALLOWED_VIDEO_TYPES.includes(
      file.type,
    )
  ) {
    return (
      'Video must be MP4, WebM, or MOV.'
    )
  }

  if (file.size > MAX_VIDEO_SIZE) {
    return (
      'Video must not exceed 50 MB.'
    )
  }

  return ''
}

function ProfilePage() {
  const location = useLocation()

  const {
    user,
    artist,
    token,
    isArtist,
    becomeArtist,
    refreshIdentity,
  } = useAuth()

  const locationState =
    location.state as
      | ProfileLocationState
      | null

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

  const [
    isSavingArtistProfile,
    setIsSavingArtistProfile,
  ] = useState(false)

  const [bio, setBio] =
    useState('')

  const [
    profileImage,
    setProfileImage,
  ] = useState<File | null>(null)

  const [
    heroVideo,
    setHeroVideo,
  ] = useState<File | null>(null)

  const [
    heroVideoPoster,
    setHeroVideoPoster,
  ] = useState<File | null>(null)

  const profileImagePreview =
    useObjectURL(profileImage)

  const heroVideoPreview =
    useObjectURL(heroVideo)

  const heroPosterPreview =
    useObjectURL(heroVideoPoster)

  useEffect(() => {
    setBio(
      artist?.bio ?? '',
    )
  }, [artist?.bio])

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

  function handleProfileImageChange(
    file: File | null,
  ) {
    setError('')

    if (!file) {
      setProfileImage(null)
      return
    }

    const validationError =
      validateImage(file)

    if (validationError) {
      setProfileImage(null)
      setError(validationError)
      return
    }

    setProfileImage(file)
  }

  function handleHeroVideoChange(
    file: File | null,
  ) {
    setError('')

    if (!file) {
      setHeroVideo(null)
      return
    }

    const validationError =
      validateVideo(file)

    if (validationError) {
      setHeroVideo(null)
      setError(validationError)
      return
    }

    setHeroVideo(file)
  }

  function handleHeroPosterChange(
    file: File | null,
  ) {
    setError('')

    if (!file) {
      setHeroVideoPoster(null)
      return
    }

    const validationError =
      validateImage(file)

    if (validationError) {
      setHeroVideoPoster(null)
      setError(validationError)
      return
    }

    setHeroVideoPoster(file)
  }

  async function handleSaveArtistProfile() {
    if (!token || !artist) {
      setError(
        'Artist authentication is required.',
      )
      return
    }

    if (bio.length > 2000) {
      setError(
        'Artist bio must not exceed 2000 characters.',
      )
      return
    }

    setError('')
    setSuccess('')
    setIsSavingArtistProfile(true)

    try {
      const formData =
        new FormData()

      formData.append(
        'bio',
        bio,
      )

      if (profileImage) {
        formData.append(
          'profile_image',
          profileImage,
        )
      }

      if (heroVideo) {
        formData.append(
          'hero_video',
          heroVideo,
        )
      }

      if (heroVideoPoster) {
        formData.append(
          'hero_video_poster',
          heroVideoPoster,
        )
      }

      await updateArtistProfile(
        formData,
        token,
      )

      await refreshIdentity()

      setProfileImage(null)
      setHeroVideo(null)
      setHeroVideoPoster(null)

      setSuccess(
        'Artist profile updated successfully.',
      )
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : 'Unable to update artist profile.',
      )
    } finally {
      setIsSavingArtistProfile(
        false,
      )
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
        <>
          <section className="content-panel">
            <p className="eyebrow">
              ARTIST PROFILE
            </p>

            <h3>{artist.name}</h3>

            <p>
              Your account has artist
              access. Upload music, manage
              your releases and customize
              your public artist page.
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

              <Link
                to={`/artists/${artist.id}`}
                className="secondary-button"
              >
                View public profile
              </Link>
            </div>
          </section>

          <section className="content-panel artist-settings-panel">
            <div className="artist-settings-header">
              <div>
                <p className="eyebrow">
                  ARTIST SETTINGS
                </p>

                <h3>
                  Edit Artist Profile
                </h3>

                <p>
                  Customize how your
                  artist profile appears
                  to listeners.
                </p>
              </div>
            </div>

            <div className="artist-settings-grid">
              <div className="artist-settings-field artist-settings-field-wide">
                <label
                  htmlFor="artist-bio"
                  className="artist-settings-label"
                >
                  Artist bio
                </label>

                <textarea
                  id="artist-bio"
                  className="artist-settings-textarea"
                  value={bio}
                  maxLength={2000}
                  rows={6}
                  onChange={(event) => {
                    setBio(
                      event.target.value,
                    )
                  }}
                  placeholder="Tell listeners about yourself, your sound and your journey."
                />

                <small className="artist-bio-counter">
                  {bio.length}/2000
                </small>
              </div>

              <div className="artist-media-card">
                <div className="artist-media-card-header">
                  <strong>
                    Profile image
                  </strong>

                  <span>
                    Your main artist photo.
                  </span>
                </div>

                <div className="artist-profile-preview">
                  {profileImagePreview ? (
                    <img
                      src={
                        profileImagePreview
                      }
                      alt="New artist profile preview"
                    />
                  ) : artist.profile_image_url ? (
                    <img
                      src={
                        artist.profile_image_url
                      }
                      alt={`${artist.name} profile`}
                    />
                  ) : (
                    <span className="artist-profile-preview-fallback">
                      {artist.name
                        .charAt(0)
                        .toUpperCase()}
                    </span>
                  )}
                </div>

                <input
                  id="profile-image"
                  className="artist-file-input"
                  type="file"
                  accept="image/jpeg,image/png,image/webp"
                  onChange={(event) => {
                    handleProfileImageChange(
                      event.target
                        .files?.[0] ??
                        null,
                    )
                  }}
                />

                {profileImage && (
                  <span className="artist-selected-file">
                    Selected:{' '}
                    {profileImage.name}
                  </span>
                )}

                <small className="artist-settings-help">
                  JPEG, PNG or WebP.
                  Maximum size 10 MB.
                </small>
              </div>

              <div className="artist-media-card">
                <div className="artist-media-card-header">
                  <strong>
                    Hero video
                  </strong>

                  <span>
                    Promotional video shown
                    on your artist page.
                  </span>
                </div>

                {heroVideoPreview ? (
                  <div className="artist-video-preview">
                    <video
                      src={
                        heroVideoPreview
                      }
                      controls
                      muted
                      playsInline
                    />
                  </div>
                ) : artist.hero_video_url ? (
                  <div className="artist-video-preview">
                    <video
                      src={
                        artist.hero_video_url
                      }
                      poster={
                        artist.hero_video_poster_url
                      }
                      controls
                      muted
                      playsInline
                    />
                  </div>
                ) : (
                  <div className="artist-media-placeholder">
                    <span>
                      Choose a promotional
                      video and preview it
                      here before uploading.
                    </span>
                  </div>
                )}

                <input
                  id="hero-video"
                  className="artist-file-input"
                  type="file"
                  accept="video/mp4,video/webm,video/quicktime"
                  onChange={(event) => {
                    handleHeroVideoChange(
                      event.target
                        .files?.[0] ??
                        null,
                    )
                  }}
                />

                {heroVideo && (
                  <span className="artist-selected-file">
                    Selected:{' '}
                    {heroVideo.name}
                  </span>
                )}

                <small className="artist-settings-help">
                  MP4, WebM or MOV.
                  Maximum size 50 MB.
                  Recommended duration:
                  15–20 seconds.
                </small>
              </div>

              <div className="artist-media-card">
                <div className="artist-media-card-header">
                  <strong>
                    Hero poster
                  </strong>

                  <span>
                    Image displayed while
                    your video loads.
                  </span>
                </div>

                {heroPosterPreview ? (
                  <div className="artist-poster-preview">
                    <img
                      src={
                        heroPosterPreview
                      }
                      alt="New hero poster preview"
                    />
                  </div>
                ) : artist.hero_video_poster_url ? (
                  <div className="artist-poster-preview">
                    <img
                      src={
                        artist.hero_video_poster_url
                      }
                      alt={`${artist.name} hero poster`}
                    />
                  </div>
                ) : (
                  <div className="artist-media-placeholder">
                    <span>
                      Upload a poster image
                      for your hero video.
                    </span>
                  </div>
                )}

                <input
                  id="hero-poster"
                  className="artist-file-input"
                  type="file"
                  accept="image/jpeg,image/png,image/webp"
                  onChange={(event) => {
                    handleHeroPosterChange(
                      event.target
                        .files?.[0] ??
                        null,
                    )
                  }}
                />

                {heroVideoPoster && (
                  <span className="artist-selected-file">
                    Selected:{' '}
                    {heroVideoPoster.name}
                  </span>
                )}

                <small className="artist-settings-help">
                  JPEG, PNG or WebP.
                  Maximum size 10 MB.
                </small>
              </div>
            </div>

            <div className="artist-settings-save-row">
              <button
                type="button"
                className="primary-button"
                onClick={
                  handleSaveArtistProfile
                }
                disabled={
                  isSavingArtistProfile
                }
              >
                {isSavingArtistProfile
                  ? 'Saving artist profile...'
                  : 'Save artist profile'}
              </button>
            </div>
          </section>
        </>
      ) : (
        <section className="content-panel">
          <p className="eyebrow">
            ARTIST ACCESS
          </p>

          <h3>
            Become an Artist
          </h3>

          {user?.email_verified ? (
            <>
              <p>
                Your current account is a
                listener account. Create
                an artist profile when you
                are ready to publish your
                own music.
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