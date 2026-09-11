import {
  useEffect,
  useMemo,
  useState,
  type FormEvent,
} from 'react'

import {
  Link,
  useNavigate,
  useParams,
} from 'react-router-dom'

import { useAuth } from '../context/AuthContext'

import {
  APIError,
  addTrackToRelease,
  deleteRelease,
  getMusic,
  getMyRelease,
  removeTrackFromRelease,
  updateRelease,
} from '../lib/api'

import type { Music } from '../types/music'

import type {
  ReleaseDetails,
  ReleaseType,
} from '../types/release'

function toDateInputValue(
  value: string | null,
) {
  if (!value) {
    return ''
  }

  const date = new Date(value)

  if (Number.isNaN(date.getTime())) {
    return ''
  }

  return date
    .toISOString()
    .slice(0, 10)
}

function ReleaseManagerPage() {
  const { id } = useParams()

  const navigate =
    useNavigate()

  const {
    token,
    artist,
  } = useAuth()

  const releaseID =
    Number(id)

  const validReleaseID =
    Number.isInteger(releaseID) &&
    releaseID > 0

  const [
    details,
    setDetails,
  ] =
    useState<ReleaseDetails | null>(
      null,
    )

  const [
    music,
    setMusic,
  ] = useState<Music[]>([])

  const [
    title,
    setTitle,
  ] = useState('')

  const [
    releaseType,
    setReleaseType,
  ] =
    useState<ReleaseType>('SINGLE')

  const [
    description,
    setDescription,
  ] = useState('')

  const [
    releaseDate,
    setReleaseDate,
  ] = useState('')

  const [
    loading,
    setLoading,
  ] = useState(true)

  const [
    saving,
    setSaving,
  ] = useState(false)

  const [
    workingTrackID,
    setWorkingTrackID,
  ] =
    useState<number | null>(null)

  const [
    deleting,
    setDeleting,
  ] = useState(false)

  const [
    error,
    setError,
  ] = useState('')

  const [
    success,
    setSuccess,
  ] = useState('')

  const [
    notFound,
    setNotFound,
  ] = useState(false)

  async function refreshRelease(
    accessToken: string,
  ) {
    const result =
      await getMyRelease(
        releaseID,
        accessToken,
      )

    setDetails(result)

    setTitle(
      result.release.title,
    )

    setReleaseType(
      result.release.release_type,
    )

    setDescription(
      result.release.description,
    )

    setReleaseDate(
      toDateInputValue(
        result.release.release_date,
      ),
    )
  }

  useEffect(() => {
    if (
      !token ||
      !validReleaseID
    ) {
      setLoading(false)
      return
    }

    const accessToken = token
    let cancelled = false

    async function loadPage() {
      try {
        setLoading(true)
        setError('')
        setNotFound(false)

        const [
          releaseResult,
          musicResult,
        ] = await Promise.all([
          getMyRelease(
            releaseID,
            accessToken,
          ),
          getMusic(),
        ])

        if (cancelled) {
          return
        }

        setDetails(
          releaseResult,
        )

        setMusic(
          musicResult,
        )

        setTitle(
          releaseResult
            .release.title,
        )

        setReleaseType(
          releaseResult
            .release.release_type,
        )

        setDescription(
          releaseResult
            .release.description,
        )

        setReleaseDate(
          toDateInputValue(
            releaseResult
              .release.release_date,
          ),
        )
      } catch (err) {
        if (cancelled) {
          return
        }

        if (
          err instanceof APIError &&
          err.status === 404
        ) {
          setNotFound(true)
          return
        }

        setError(
          err instanceof Error
            ? err.message
            : 'Failed to load release',
        )
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    }

    void loadPage()

    return () => {
      cancelled = true
    }
  }, [
    releaseID,
    token,
    validReleaseID,
  ])

  const availableTracks =
    useMemo(() => {
      if (!artist) {
        return []
      }

      return music.filter(
        (track) =>
          track.artist_id ===
            artist.id &&
          track.release_id === null,
      )
    }, [
      artist,
      music,
    ])

  async function saveRelease(
    isPublished:
      boolean,
  ) {
    if (
      !token ||
      !details ||
      saving
    ) {
      return
    }

    if (!title.trim()) {
      setError(
        'Release title is required.',
      )
      return
    }

    try {
      setSaving(true)
      setError('')
      setSuccess('')

      const updated =
        await updateRelease(
          releaseID,
          {
            title:
              title.trim(),
            release_type:
              releaseType,
            cover_image_url:
              details.release
                .cover_image_url,
            cover_image_public_id:
              details.release
                .cover_image_public_id,
            description:
              description.trim(),
            release_date:
              releaseDate,
            is_published:
              isPublished,
          },
          token,
        )

      setDetails(
        (current) =>
          current
            ? {
                ...current,
                release:
                  updated,
              }
            : current,
      )

      setSuccess(
        isPublished
          ? 'Release published successfully.'
          : 'Release saved as a draft.',
      )
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : 'Failed to save release',
      )
    } finally {
      setSaving(false)
    }
  }

  async function handleSave(
    event: FormEvent<HTMLFormElement>,
  ) {
    event.preventDefault()

    await saveRelease(
      details?.release
        .is_published ?? false,
    )
  }

  async function handleAddTrack(
    track: Music,
  ) {
    if (
      !token ||
      !details ||
      workingTrackID !== null
    ) {
      return
    }

    const suggestedNumber =
      details.tracks.length + 1

    const value =
      window.prompt(
        `Track number for "${track.song_title}"`,
        String(
          suggestedNumber,
        ),
      )

    if (value === null) {
      return
    }

    const trackNumber =
      Number(value)

    if (
      !Number.isInteger(
        trackNumber,
      ) ||
      trackNumber <= 0
    ) {
      setError(
        'Track number must be a positive whole number.',
      )
      return
    }

    try {
      setWorkingTrackID(
        track.id,
      )

      setError('')
      setSuccess('')

      await addTrackToRelease(
        releaseID,
        track.id,
        {
          track_number:
            trackNumber,
        },
        token,
      )

      await refreshRelease(
        token,
      )

      setMusic(
        (current) =>
          current.map(
            (item) =>
              item.id ===
              track.id
                ? {
                    ...item,
                    release_id:
                      releaseID,
                    track_number:
                      trackNumber,
                  }
                : item,
          ),
      )

      setSuccess(
        `"${track.song_title}" added to the release.`,
      )
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : 'Failed to add track',
      )
    } finally {
      setWorkingTrackID(
        null,
      )
    }
  }

  async function handleRemoveTrack(
    track: Music,
  ) {
    if (
      !token ||
      workingTrackID !== null
    ) {
      return
    }

    const confirmed =
      window.confirm(
        `Remove "${track.song_title}" from this release? The song itself will not be deleted.`,
      )

    if (!confirmed) {
      return
    }

    try {
      setWorkingTrackID(
        track.id,
      )

      setError('')
      setSuccess('')

      await removeTrackFromRelease(
        releaseID,
        track.id,
        token,
      )

      await refreshRelease(
        token,
      )

      setMusic(
        (current) =>
          current.map(
            (item) =>
              item.id ===
              track.id
                ? {
                    ...item,
                    release_id:
                      null,
                    track_number:
                      null,
                  }
                : item,
          ),
      )

      setSuccess(
        `"${track.song_title}" removed from the release.`,
      )
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : 'Failed to remove track',
      )
    } finally {
      setWorkingTrackID(
        null,
      )
    }
  }

  async function handleDelete() {
    if (
      !token ||
      !details ||
      deleting
    ) {
      return
    }

    const confirmed =
      window.confirm(
        `Delete "${details.release.title}"? The songs will remain in your music library.`,
      )

    if (!confirmed) {
      return
    }

    try {
      setDeleting(true)
      setError('')

      await deleteRelease(
        releaseID,
        token,
      )

      navigate(
        '/my-releases',
        {
          replace: true,
        },
      )
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : 'Failed to delete release',
      )
    } finally {
      setDeleting(false)
    }
  }

  if (!validReleaseID) {
    return (
      <section className="content-panel">
        <h3>
          Release not found
        </h3>

        <Link to="/my-releases">
          Back to Release Manager
        </Link>
      </section>
    )
  }

  if (loading) {
    return (
      <section className="content-panel">
        <p>
          Loading release...
        </p>
      </section>
    )
  }

  if (notFound) {
    return (
      <section className="content-panel">
        <h3>
          Release not found
        </h3>

        <p>
          This release does not
          exist or does not belong
          to your artist account.
        </p>

        <Link to="/my-releases">
          Back to Release Manager
        </Link>
      </section>
    )
  }

  if (!details) {
    return (
      <section className="content-panel">
        <p>
          {error ||
            'Unable to load release.'}
        </p>
      </section>
    )
  }

  const {
    release,
    tracks,
  } = details

  return (
    <>
      <header className="page-header release-manager-header">
        <div>
          <p className="eyebrow">
            RELEASE MANAGER
          </p>

          <h2>
            {release.title}
          </h2>

          <p>
            {release.is_published
              ? 'Published'
              : 'Draft'}
            {' • '}
            {release.release_type}
          </p>
        </div>

        <div className="release-manager-header-actions">
          {release.is_published && (
            <Link
              to={`/releases/${release.id}`}
              className="release-manager-link"
            >
              View Public Page
            </Link>
          )}

          <Link
            to="/my-releases"
            className="release-manager-link"
          >
            All Releases
          </Link>
        </div>
      </header>

      {error && (
        <div
          className="status-message"
          role="alert"
        >
          {error}
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

      <section className="content-panel">
        <div className="section-heading">
          <div>
            <p className="eyebrow">
              DETAILS
            </p>

            <h3>
              Release information
            </h3>
          </div>
        </div>

        <form
          className="release-manager-form"
          onSubmit={
            handleSave
          }
        >
          <label>
            <span>Title</span>

            <input
              type="text"
              value={title}
              onChange={(event) =>
                setTitle(
                  event.target.value,
                )
              }
              required
            />
          </label>

          <label>
            <span>
              Release type
            </span>

            <select
              value={releaseType}
              onChange={(event) =>
                setReleaseType(
                  event.target
                    .value as ReleaseType,
                )
              }
            >
              <option value="SINGLE">
                Single
              </option>

              <option value="EP">
                EP
              </option>

              <option value="ALBUM">
                Album
              </option>
            </select>
          </label>

          <label>
            <span>
              Release date
            </span>

            <input
              type="date"
              value={releaseDate}
              onChange={(event) =>
                setReleaseDate(
                  event.target.value,
                )
              }
            />
          </label>

          <label className="release-form-wide">
            <span>
              Description
            </span>

            <textarea
              value={description}
              onChange={(event) =>
                setDescription(
                  event.target.value,
                )
              }
              rows={4}
            />
          </label>

          <div className="release-form-wide release-edit-actions">
            <button
              type="submit"
              className="release-primary-button"
              disabled={saving}
            >
              {saving
                ? 'Saving...'
                : 'Save Changes'}
            </button>

            {!release.is_published ? (
              <button
                type="button"
                className="release-publish-button"
                disabled={saving}
                onClick={() =>
                  void saveRelease(
                    true,
                  )
                }
              >
                Publish Release
              </button>
            ) : (
              <button
                type="button"
                className="release-secondary-button"
                disabled={saving}
                onClick={() =>
                  void saveRelease(
                    false,
                  )
                }
              >
                Unpublish
              </button>
            )}
          </div>
        </form>
      </section>

      <section className="section">
        <div className="section-heading">
          <div>
            <p className="eyebrow">
              TRACKLIST
            </p>

            <h3>
              Release tracks
            </h3>
          </div>
        </div>

        {tracks.length === 0 ? (
          <section className="content-panel">
            <p>
              No tracks have been
              added to this release.
            </p>
          </section>
        ) : (
          <div className="release-track-manager-list">
            {tracks.map(
              (track) => (
                <article
                  className="release-track-manager-item"
                  key={track.id}
                >
                  <span className="release-track-manager-number">
                    {track.track_number ??
                      '—'}
                  </span>

                  <div className="release-track-manager-cover">
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

                  <div className="release-track-manager-info">
                    <strong>
                      {
                        track.song_title
                      }
                    </strong>

                    <span>
                      {track.genre}
                    </span>
                  </div>

                  <button
                    type="button"
                    className="release-remove-track-button"
                    disabled={
                      workingTrackID ===
                      track.id
                    }
                    onClick={() =>
                      void handleRemoveTrack(
                        track,
                      )
                    }
                  >
                    {workingTrackID ===
                    track.id
                      ? 'Removing...'
                      : 'Remove'}
                  </button>
                </article>
              ),
            )}
          </div>
        )}
      </section>

      <section className="section">
        <div className="section-heading">
          <div>
            <p className="eyebrow">
              YOUR MUSIC
            </p>

            <h3>
              Add tracks
            </h3>
          </div>
        </div>

        {availableTracks.length ===
        0 ? (
          <section className="content-panel">
            <p>
              You have no standalone
              tracks available to add.
            </p>
          </section>
        ) : (
          <div className="release-track-manager-list">
            {availableTracks.map(
              (track) => (
                <article
                  className="release-track-manager-item"
                  key={track.id}
                >
                  <div className="release-track-manager-cover">
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

                  <div className="release-track-manager-info">
                    <strong>
                      {
                        track.song_title
                      }
                    </strong>

                    <span>
                      {track.genre}
                    </span>
                  </div>

                  <button
                    type="button"
                    className="release-add-track-button"
                    disabled={
                      workingTrackID ===
                      track.id
                    }
                    onClick={() =>
                      void handleAddTrack(
                        track,
                      )
                    }
                  >
                    {workingTrackID ===
                    track.id
                      ? 'Adding...'
                      : 'Add Track'}
                  </button>
                </article>
              ),
            )}
          </div>
        )}
      </section>

      <section className="content-panel release-danger-zone">
        <div>
          <p className="eyebrow">
            DANGER ZONE
          </p>

          <h3>
            Delete release
          </h3>

          <p>
            Deleting the release will
            not delete its songs. The
            songs return to your
            standalone music library.
          </p>
        </div>

        <button
          type="button"
          className="release-delete-button"
          disabled={deleting}
          onClick={() =>
            void handleDelete()
          }
        >
          {deleting
            ? 'Deleting...'
            : 'Delete Release'}
        </button>
      </section>
    </>
  )
}

export default ReleaseManagerPage