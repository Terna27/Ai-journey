import {
  type FormEvent,
  useCallback,
  useEffect,
  useState,
} from 'react'

import {
  Link,
  useNavigate,
} from 'react-router-dom'

import { useAuth } from '../context/AuthContext'

import {
  createPodcast,
  getMyPodcasts,
} from '../lib/api'

import type { Podcast } from '../types/podcast'

function formatDate(value: string) {
  const date = new Date(value)

  if (Number.isNaN(date.getTime())) {
    return ''
  }

  return date.toLocaleDateString()
}

function MyPodcastsPage() {
  const navigate = useNavigate()

  const { token } = useAuth()

  const [podcasts, setPodcasts] =
    useState<Podcast[]>([])

  const [loading, setLoading] =
    useState(true)

  const [creating, setCreating] =
    useState(false)

  const [error, setError] =
    useState('')

  const [title, setTitle] =
    useState('')

  const [description, setDescription] =
    useState('')

  const [category, setCategory] =
    useState('')

  const [isExplicit, setIsExplicit] =
    useState(false)

  const loadPodcasts =
    useCallback(async () => {
      if (!token) {
        return
      }

      setLoading(true)
      setError('')

      try {
        const result =
          await getMyPodcasts(token)

        setPodcasts(result)
      } catch (err) {
        setError(
          err instanceof Error
            ? err.message
            : 'Unable to load your podcasts.',
        )
      } finally {
        setLoading(false)
      }
    }, [token])

  useEffect(() => {
    void loadPodcasts()
  }, [loadPodcasts])

  async function handleCreate(
    event: FormEvent<HTMLFormElement>,
  ) {
    event.preventDefault()

    if (!token || creating) {
      return
    }

    const cleanTitle = title.trim()

    if (!cleanTitle) {
      setError(
        'Podcast title is required.',
      )
      return
    }

    setCreating(true)
    setError('')

    try {
      const podcast =
        await createPodcast(
          {
            title: cleanTitle,
            description:
              description.trim(),
            category:
              category.trim(),
            is_explicit:
              isExplicit,
          },
          token,
        )

      setTitle('')
      setDescription('')
      setCategory('')
      setIsExplicit(false)

      navigate(
        `/my-podcasts/${podcast.id}`,
      )
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : 'Unable to create podcast.',
      )
    } finally {
      setCreating(false)
    }
  }

  return (
    <>
      <header className="page-header podcast-owner-header">
        <div>
          <p className="eyebrow">
            PODCAST STUDIO
          </p>

          <h2>My Podcasts</h2>

          <p>
            Create and manage your
            podcast shows and episodes.
          </p>
        </div>

        <Link
          to="/podcasts"
          className="release-manager-link"
        >
          Browse Podcasts
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

      <section className="content-panel">
        <div className="section-heading">
          <div>
            <p className="eyebrow">
              NEW SHOW
            </p>

            <h3>
              Create a podcast
            </h3>
          </div>
        </div>

        <form
          className="release-manager-form"
          onSubmit={handleCreate}
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
              placeholder="Podcast title"
              required
            />
          </label>

          <label>
            <span>Category</span>

            <input
              type="text"
              value={category}
              onChange={(event) =>
                setCategory(
                  event.target.value,
                )
              }
              placeholder="Music, Culture, Technology..."
            />
          </label>

          <label className="release-form-wide">
            <span>Description</span>

            <textarea
              value={description}
              onChange={(event) =>
                setDescription(
                  event.target.value,
                )
              }
              rows={4}
              placeholder="Tell listeners what this podcast is about."
            />
          </label>

          <label className="podcast-checkbox-field release-form-wide">
            <input
              type="checkbox"
              checked={isExplicit}
              onChange={(event) =>
                setIsExplicit(
                  event.target.checked,
                )
              }
            />

            <span>
              This podcast contains
              explicit content
            </span>
          </label>

          <div className="release-form-wide release-edit-actions">
            <button
              type="submit"
              className="release-primary-button"
              disabled={creating}
            >
              {creating
                ? 'Creating...'
                : 'Create Podcast'}
            </button>
          </div>
        </form>
      </section>

      <section className="section">
        <div className="section-heading">
          <div>
            <p className="eyebrow">
              YOUR SHOWS
            </p>

            <h3>
              Podcast library
            </h3>
          </div>

          {!loading && (
            <span>
              {podcasts.length}{' '}
              {podcasts.length === 1
                ? 'show'
                : 'shows'}
            </span>
          )}
        </div>

        {loading ? (
          <section className="content-panel">
            <p>
              Loading your podcasts...
            </p>
          </section>
        ) : podcasts.length === 0 ? (
          <section className="content-panel">
            <p>
              You have not created a
              podcast yet. Create your
              first show above.
            </p>
          </section>
        ) : (
          <div className="podcast-owner-grid">
            {podcasts.map(
              (podcast) => (
                <article
                  className="podcast-owner-card"
                  key={podcast.id}
                >
                  <div className="podcast-owner-artwork">
                    {podcast.artwork_url ? (
                      <img
                        src={
                          podcast.artwork_url
                        }
                        alt={`${podcast.title} artwork`}
                      />
                    ) : (
                      <span>◉</span>
                    )}
                  </div>

                  <div className="podcast-owner-card-body">
                    <div className="podcast-owner-card-heading">
                      <div>
                        <span className={`podcast-status podcast-status-${podcast.status.toLowerCase()}`}>
                          {podcast.status}
                        </span>

                        <h3>
                          {podcast.title}
                        </h3>
                      </div>

                      {podcast.is_explicit && (
                        <span className="podcast-explicit-badge">
                          E
                        </span>
                      )}
                    </div>

                    {podcast.category && (
                      <p className="podcast-owner-category">
                        {podcast.category}
                      </p>
                    )}

                    <p className="podcast-owner-description">
                      {podcast.description ||
                        'No description yet.'}
                    </p>

                    <div className="podcast-owner-meta">
                      <span>
                        Updated{' '}
                        {formatDate(
                          podcast.updated_at,
                        )}
                      </span>
                    </div>

                    <div className="podcast-owner-actions">
                      <Link
                        to={`/my-podcasts/${podcast.id}`}
                        className="release-primary-button"
                      >
                        Manage
                      </Link>

                      {podcast.status ===
                        'PUBLISHED' && (
                        <Link
                          to={`/podcasts/${encodeURIComponent(
                            podcast.slug,
                          )}`}
                          className="release-manager-link"
                        >
                          Public Page
                        </Link>
                      )}
                    </div>
                  </div>
                </article>
              ),
            )}
          </div>
        )}
      </section>
    </>
  )
}

export default MyPodcastsPage
