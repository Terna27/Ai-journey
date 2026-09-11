import {
  useEffect,
  useState,
  type FormEvent,
} from 'react'

import {
  Link,
  useNavigate,
} from 'react-router-dom'

import { useAuth } from '../context/AuthContext'

import {
  createRelease,
  getMyReleases,
} from '../lib/api'

import type {
  Release,
  ReleaseType,
} from '../types/release'

function formatDate(
  value: string | null,
) {
  if (!value) {
    return 'No release date'
  }

  const date = new Date(value)

  if (Number.isNaN(date.getTime())) {
    return 'No release date'
  }

  return new Intl.DateTimeFormat(
    'en',
    {
      day: 'numeric',
      month: 'short',
      year: 'numeric',
    },
  ).format(date)
}

function MyReleasesPage() {
  const {
    token,
  } = useAuth()

  const navigate =
    useNavigate()

  const [
    releases,
    setReleases,
  ] = useState<Release[]>([])

  const [
    title,
    setTitle,
  ] = useState('')

  const [
    releaseType,
    setReleaseType,
  ] = useState<ReleaseType>('SINGLE')

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
    creating,
    setCreating,
  ] = useState(false)

  const [
    error,
    setError,
  ] = useState('')

  useEffect(() => {
    if (!token) {
      setLoading(false)
      return
    }

    const accessToken = token
    let cancelled = false

    async function loadReleases() {
      try {
        setLoading(true)
        setError('')

        const result =
          await getMyReleases(
            accessToken,
          )

        if (!cancelled) {
          setReleases(result)
        }
      } catch (err) {
        if (!cancelled) {
          setError(
            err instanceof Error
              ? err.message
              : 'Failed to load releases',
          )
        }
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    }

    void loadReleases()

    return () => {
      cancelled = true
    }
  }, [token])

  async function handleCreate(
    event: FormEvent<HTMLFormElement>,
  ) {
    event.preventDefault()

    if (!token || creating) {
      return
    }

    const trimmedTitle =
      title.trim()

    if (!trimmedTitle) {
      setError(
        'Release title is required.',
      )
      return
    }

    try {
      setCreating(true)
      setError('')

      const release =
        await createRelease(
          {
            title: trimmedTitle,
            release_type:
              releaseType,
            cover_image_url: '',
            cover_image_public_id: '',
            description:
              description.trim(),
            release_date:
              releaseDate,
            is_published: false,
          },
          token,
        )

      navigate(
        `/my-releases/${release.id}`,
      )
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : 'Failed to create release',
      )
    } finally {
      setCreating(false)
    }
  }

  const drafts =
    releases.filter(
      (release) =>
        !release.is_published,
    )

  const published =
    releases.filter(
      (release) =>
        release.is_published,
    )

  return (
    <>
      <header className="page-header release-manager-header">
        <div>
          <p className="eyebrow">
            ARTIST TOOLS
          </p>

          <h2>
            Release Manager
          </h2>

          <p>
            Create and manage your
            singles, EPs and albums.
          </p>
        </div>

        <Link
          to="/my-music"
          className="release-manager-link"
        >
          Back to My Music
        </Link>
      </header>

      {error && (
        <div
          className="status-message"
          role="alert"
        >
          {error}
        </div>
      )}

      <section className="content-panel release-create-panel">
        <div className="section-heading">
          <div>
            <p className="eyebrow">
              NEW RELEASE
            </p>

            <h3>
              Create a draft
            </h3>
          </div>
        </div>

        <form
          className="release-manager-form"
          onSubmit={
            handleCreate
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
              placeholder="Release title"
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
              placeholder="Optional release description"
              rows={4}
            />
          </label>

          <div className="release-form-wide">
            <button
              type="submit"
              className="release-primary-button"
              disabled={creating}
            >
              {creating
                ? 'Creating...'
                : 'Create Draft'}
            </button>
          </div>
        </form>
      </section>

      <section className="section">
        <div className="section-heading">
          <div>
            <p className="eyebrow">
              DRAFTS
            </p>

            <h3>
              Unpublished
            </h3>
          </div>
        </div>

        {loading && (
          <section className="content-panel">
            <p>
              Loading releases...
            </p>
          </section>
        )}

        {!loading &&
          drafts.length === 0 && (
            <section className="content-panel">
              <p>
                You have no draft
                releases.
              </p>
            </section>
          )}

        {!loading &&
          drafts.length > 0 && (
            <div className="release-manager-list">
              {drafts.map(
                (release) => (
                  <Link
                    key={release.id}
                    to={`/my-releases/${release.id}`}
                    className="release-manager-item"
                  >
                    <div>
                      <strong>
                        {
                          release.title
                        }
                      </strong>

                      <span>
                        {
                          release.release_type
                        }
                        {' • '}
                        {formatDate(
                          release.release_date,
                        )}
                      </span>
                    </div>

                    <span className="release-status release-status-draft">
                      Draft
                    </span>
                  </Link>
                ),
              )}
            </div>
          )}
      </section>

      <section className="section">
        <div className="section-heading">
          <div>
            <p className="eyebrow">
              LIVE
            </p>

            <h3>
              Published
            </h3>
          </div>
        </div>

        {!loading &&
          published.length === 0 && (
            <section className="content-panel">
              <p>
                You have no published
                releases yet.
              </p>
            </section>
          )}

        {!loading &&
          published.length > 0 && (
            <div className="release-manager-list">
              {published.map(
                (release) => (
                  <Link
                    key={release.id}
                    to={`/my-releases/${release.id}`}
                    className="release-manager-item"
                  >
                    <div>
                      <strong>
                        {
                          release.title
                        }
                      </strong>

                      <span>
                        {
                          release.release_type
                        }
                        {' • '}
                        {formatDate(
                          release.release_date,
                        )}
                      </span>
                    </div>

                    <span className="release-status release-status-published">
                      Published
                    </span>
                  </Link>
                ),
              )}
            </div>
          )}
      </section>
    </>
  )
}

export default MyReleasesPage