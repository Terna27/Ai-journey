import {
  useMemo,
  useState,
  type ChangeEvent,
  type FormEvent,
} from 'react'

import { useNavigate } from 'react-router-dom'

import { useAuth } from '../context/AuthContext'
import { uploadMusic } from '../lib/api'

const MAX_IMAGE_SIZE = 10 * 1024 * 1024
const MAX_AUDIO_SIZE = 40 * 1024 * 1024

const ALLOWED_IMAGE_TYPES = [
  'image/jpeg',
  'image/png',
  'image/webp',
]

const ALLOWED_AUDIO_TYPES = [
  'audio/mpeg',
  'audio/mp3',
  'audio/wav',
  'audio/x-wav',
  'audio/ogg',
  'audio/mp4',
  'audio/x-m4a',
]

function formatFileSize(bytes: number) {
  if (bytes < 1024 * 1024) {
    return `${(bytes / 1024).toFixed(1)} KB`
  }

  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

function UploadPage() {
  const navigate = useNavigate()
  const { artist, token } = useAuth()

  const [songTitle, setSongTitle] = useState('')
  const [genre, setGenre] = useState('')

  const [image, setImage] = useState<File | null>(null)
  const [audio, setAudio] = useState<File | null>(null)

  const [imagePreview, setImagePreview] = useState<string | null>(null)

  const [error, setError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  const artistName = artist?.name ?? ''

  const canSubmit = useMemo(() => {
    return Boolean(
      songTitle.trim() &&
        genre.trim() &&
        image &&
        audio &&
        token &&
        artistName &&
        !isSubmitting,
    )
  }, [
    songTitle,
    genre,
    image,
    audio,
    token,
    artistName,
    isSubmitting,
  ])

  function handleImageChange(
    event: ChangeEvent<HTMLInputElement>,
  ) {
    setError('')

    const file = event.target.files?.[0]

    if (!file) {
      setImage(null)
      setImagePreview(null)
      return
    }

    if (!ALLOWED_IMAGE_TYPES.includes(file.type)) {
      setError(
        'Cover image must be a JPG, PNG or WebP file.',
      )

      event.target.value = ''
      setImage(null)
      setImagePreview(null)

      return
    }

    if (file.size > MAX_IMAGE_SIZE) {
      setError(
        'Cover image must be smaller than 10 MB.',
      )

      event.target.value = ''
      setImage(null)
      setImagePreview(null)

      return
    }

    if (imagePreview) {
      URL.revokeObjectURL(imagePreview)
    }

    setImage(file)
    setImagePreview(URL.createObjectURL(file))
  }

  function handleAudioChange(
    event: ChangeEvent<HTMLInputElement>,
  ) {
    setError('')

    const file = event.target.files?.[0]

    if (!file) {
      setAudio(null)
      return
    }

    if (!ALLOWED_AUDIO_TYPES.includes(file.type)) {
      setError(
        'Audio must be MP3, WAV, OGG or M4A.',
      )

      event.target.value = ''
      setAudio(null)

      return
    }

    if (file.size > MAX_AUDIO_SIZE) {
      setError(
        'Audio file must be smaller than 40 MB.',
      )

      event.target.value = ''
      setAudio(null)

      return
    }

    setAudio(file)
  }

  async function handleSubmit(
    event: FormEvent<HTMLFormElement>,
  ) {
    event.preventDefault()

    setError('')

    if (!artist || !token) {
      setError(
        'Your login session is missing. Please log in again.',
      )
      return
    }

    if (!songTitle.trim()) {
      setError('Enter the song title.')
      return
    }

    if (!genre.trim()) {
      setError('Enter the song genre.')
      return
    }

    if (!image) {
      setError('Choose a cover image.')
      return
    }

    if (!audio) {
      setError('Choose an audio file.')
      return
    }

    setIsSubmitting(true)

    try {
      await uploadMusic(
        {
          artistName: artist.name,
          songTitle: songTitle.trim(),
          genre: genre.trim(),
          image,
          audio,
        },
        token,
      )

      navigate('/my-music', {
        replace: true,
        state: {
          message: 'Your music was uploaded successfully.',
        },
      })
    } catch (err) {
      if (err instanceof Error) {
        setError(err.message)
      } else {
        setError(
          'Unable to upload your music. Please try again.',
        )
      }
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <>
      <header className="page-header">
        <p className="eyebrow">
          ARTIST STUDIO
        </p>

        <h2>Upload music</h2>

        <p>
          Publish a song to your artist account.
          Add your song information, cover artwork
          and audio file.
        </p>
      </header>

      <section className="upload-layout">
        <form
          className="upload-form"
          onSubmit={handleSubmit}
        >
          <div className="upload-form-section">
            <div className="upload-section-heading">
              <span className="upload-step">
                1
              </span>

              <div>
                <h3>Song information</h3>

                <p>
                  Tell listeners about your track.
                </p>
              </div>
            </div>

            <div className="upload-fields">
              <label>
                Artist

                <input
                  type="text"
                  value={artistName}
                  disabled
                />

                <small>
                  Taken from your authenticated
                  artist account.
                </small>
              </label>

              <label>
                Song title

                <input
                  type="text"
                  value={songTitle}
                  onChange={(event) =>
                    setSongTitle(event.target.value)
                  }
                  placeholder="Enter song title"
                  maxLength={150}
                  disabled={isSubmitting}
                  required
                />
              </label>

              <label>
                Genre

                <input
                  type="text"
                  value={genre}
                  onChange={(event) =>
                    setGenre(event.target.value)
                  }
                  placeholder="e.g. Afrobeats"
                  maxLength={80}
                  disabled={isSubmitting}
                  required
                />
              </label>
            </div>
          </div>

          <div className="upload-form-section">
            <div className="upload-section-heading">
              <span className="upload-step">
                2
              </span>

              <div>
                <h3>Cover artwork</h3>

                <p>
                  Add artwork listeners will see
                  throughout the platform.
                </p>
              </div>
            </div>

            <label className="file-drop">
              <input
                type="file"
                accept="image/jpeg,image/png,image/webp"
                onChange={handleImageChange}
                disabled={isSubmitting}
              />

              {imagePreview ? (
                <div className="cover-preview">
                  <img
                    src={imagePreview}
                    alt="Selected cover preview"
                  />

                  <div>
                    <strong>
                      {image?.name}
                    </strong>

                    <span>
                      {image
                        ? formatFileSize(image.size)
                        : ''}
                    </span>

                    <small>
                      Click to choose another image
                    </small>
                  </div>
                </div>
              ) : (
                <div className="file-drop-content">
                  <span className="file-icon">
                    +
                  </span>

                  <strong>
                    Choose cover artwork
                  </strong>

                  <span>
                    JPG, PNG or WebP · Max 10 MB
                  </span>
                </div>
              )}
            </label>
          </div>

          <div className="upload-form-section">
            <div className="upload-section-heading">
              <span className="upload-step">
                3
              </span>

              <div>
                <h3>Audio file</h3>

                <p>
                  Select the audio file that
                  listeners will stream.
                </p>
              </div>
            </div>

            <label className="file-drop audio-drop">
              <input
                type="file"
                accept=".mp3,.wav,.ogg,.m4a,audio/mpeg,audio/wav,audio/ogg,audio/mp4"
                onChange={handleAudioChange}
                disabled={isSubmitting}
              />

              <div className="file-drop-content">
                <span className="file-icon">
                  ♪
                </span>

                {audio ? (
                  <>
                    <strong>
                      {audio.name}
                    </strong>

                    <span>
                      {formatFileSize(audio.size)}
                    </span>

                    <small>
                      Click to choose another audio file
                    </small>
                  </>
                ) : (
                  <>
                    <strong>
                      Choose audio file
                    </strong>

                    <span>
                      MP3, WAV, OGG or M4A · Max 40 MB
                    </span>
                  </>
                )}
              </div>
            </label>
          </div>

          {error && (
            <div
              className="form-error"
              role="alert"
            >
              {error}
            </div>
          )}

          <div className="upload-submit-row">
            <p>
              Your music will be connected to{' '}
              <strong>{artistName}</strong>.
            </p>

            <button
              type="submit"
              className="primary-button upload-submit"
              disabled={!canSubmit}
            >
              {isSubmitting
                ? 'Uploading...'
                : 'Publish song'}
            </button>
          </div>
        </form>

        <aside className="upload-help">
          <p className="eyebrow">
            BEFORE YOU PUBLISH
          </p>

          <h3>Upload checklist</h3>

          <div className="upload-checklist">
            <div>
              <span>01</span>

              <p>
                Make sure the song title and
                genre are correct.
              </p>
            </div>

            <div>
              <span>02</span>

              <p>
                Use clear square artwork for
                the best presentation.
              </p>
            </div>

            <div>
              <span>03</span>

              <p>
                Upload a good-quality audio
                file that you have the right
                to publish.
              </p>
            </div>

            <div>
              <span>04</span>

              <p>
                Keep this page open until the
                upload finishes.
              </p>
            </div>
          </div>
        </aside>
      </section>
    </>
  )
}

export default UploadPage