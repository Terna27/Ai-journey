import {
  useEffect,
  useState,
} from 'react'

import { Link } from 'react-router-dom'

import {
  getPublishedPodcasts,
} from '../lib/api'

import type {
  Podcast,
} from '../types/podcast'

function PodcastsPage() {
  const [
    podcasts,
    setPodcasts,
  ] = useState<Podcast[]>([])

  const [
    isLoading,
    setIsLoading,
  ] = useState(true)

  const [error, setError] =
    useState('')

  useEffect(() => {
    let cancelled = false

    async function loadPodcasts() {
      try {
        setIsLoading(true)
        setError('')

        const response =
          await getPublishedPodcasts()

        if (!cancelled) {
          setPodcasts(response)
        }
      } catch (err) {
        if (!cancelled) {
          setError(
            err instanceof Error
              ? err.message
              : 'Failed to load podcasts',
          )
        }
      } finally {
        if (!cancelled) {
          setIsLoading(false)
        }
      }
    }

    void loadPodcasts()

    return () => {
      cancelled = true
    }
  }, [])

  return (
    <>
      <header className="page-header">
        <p className="eyebrow">
          PODCASTS
        </p>

        <h2>
          Listen and discover
        </h2>

        <p>
          Explore conversations,
          stories, interviews, and
          shows from creators.
        </p>
      </header>

      {isLoading ? (
        <section className="content-panel">
          <p>
            Loading podcasts...
          </p>
        </section>
      ) : error ? (
        <section className="content-panel">
          <h3>
            Podcasts unavailable
          </h3>

          <p>{error}</p>
        </section>
      ) : podcasts.length === 0 ? (
        <section className="content-panel">
          <h3>
            No podcasts yet
          </h3>

          <p>
            Published podcasts will
            appear here.
          </p>
        </section>
      ) : (
        <section
          className="podcast-grid"
          aria-label="Published podcasts"
        >
          {podcasts.map(
            (podcast) => (
              <Link
                key={podcast.id}
                to={`/podcasts/${encodeURIComponent(
                  podcast.slug,
                )}`}
                className="podcast-card"
              >
                <div className="podcast-card-artwork">
                  {podcast.artwork_url ? (
                    <img
                      src={
                        podcast.artwork_url
                      }
                      alt={`${podcast.title} artwork`}
                    />
                  ) : (
                    <div className="podcast-artwork-fallback">
                      ◉
                    </div>
                  )}
                </div>

                <div className="podcast-card-body">
                  <h3>
                    {podcast.title}
                  </h3>

                  <div className="podcast-card-meta">
                    {podcast.category && (
                      <span>
                        {podcast.category}
                      </span>
                    )}

                    {podcast.is_explicit && (
                      <span className="podcast-explicit-badge">
                        E
                      </span>
                    )}
                  </div>

                  {podcast.description && (
                    <p>
                      {
                        podcast.description
                      }
                    </p>
                  )}
                </div>
              </Link>
            ),
          )}
        </section>
      )}
    </>
  )
}

export default PodcastsPage
