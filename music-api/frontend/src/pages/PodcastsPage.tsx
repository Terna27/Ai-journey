import {
  useEffect,
  useState,
} from 'react'

import { Link } from 'react-router-dom'

import {
  getCurrentPodcastLive,
  getPublishedPodcasts,
  getUpcomingPodcastLive,
} from '../lib/api'

import type {
  Podcast,
} from '../types/podcast'

import type {
  PodcastLiveBroadcast,
} from '../types/podcastLive'

function formatLiveWhen(
  broadcast: PodcastLiveBroadcast,
) {
  const when =
    broadcast.live_session
      .status === 'LIVE'
      ? broadcast.live_session
          .started_at
      : broadcast.live_session
          .scheduled_start_at

  if (!when) {
    return ''
  }

  return new Date(
    when,
  ).toLocaleString()
}

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

  const [
    liveNow,
    setLiveNow,
  ] = useState<
    PodcastLiveBroadcast[]
  >([])

  const [
    upcoming,
    setUpcoming,
  ] = useState<
    PodcastLiveBroadcast[]
  >([])

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

  // Live discovery is public and best-effort: a failure here
  // never hides the podcast catalog.
  useEffect(() => {
    let cancelled = false

    async function loadLive() {
      try {
        const [current, scheduled] =
          await Promise.all([
            getCurrentPodcastLive(),
            getUpcomingPodcastLive(),
          ])

        if (!cancelled) {
          setLiveNow(current)
          setUpcoming(scheduled)
        }
      } catch {
        // Live sections simply stay empty.
      }
    }

    void loadLive()

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

      {liveNow.length > 0 ? (
        <section
          className="podcast-live-section"
          aria-label="Live now"
        >
          <h3>
            <span className="live-status-badge live-status-live">
              LIVE
            </span>{' '}
            On air now
          </h3>

          <div className="podcast-live-grid">
            {liveNow.map(
              (broadcast) => (
                <Link
                  key={
                    broadcast
                      .live_session
                      .id
                  }
                  to={`/live/${broadcast.live_session.id}`}
                  className="podcast-live-card podcast-live-card-active"
                >
                  <div className="podcast-card-artwork">
                    {broadcast
                      .episode
                      .artwork_url ||
                    broadcast
                      .podcast
                      .artwork_url ? (
                      <img
                        src={
                          broadcast
                            .episode
                            .artwork_url ||
                          broadcast
                            .podcast
                            .artwork_url
                        }
                        alt=""
                      />
                    ) : (
                      <div className="podcast-artwork-fallback">
                        ◉
                      </div>
                    )}
                  </div>

                  <div className="podcast-live-card-body">
                    <h4>
                      {
                        broadcast
                          .episode
                          .title
                      }
                    </h4>

                    <p>
                      {
                        broadcast
                          .podcast
                          .title
                      }
                    </p>

                    <span className="podcast-live-when">
                      Live since{' '}
                      {formatLiveWhen(
                        broadcast,
                      )}
                    </span>
                  </div>
                </Link>
              ),
            )}
          </div>
        </section>
      ) : null}

      {upcoming.length > 0 ? (
        <section
          className="podcast-live-section"
          aria-label="Upcoming live"
        >
          <h3>
            Upcoming live
            broadcasts
          </h3>

          <div className="podcast-live-grid">
            {upcoming.map(
              (broadcast) => (
                <Link
                  key={
                    broadcast
                      .live_session
                      .id
                  }
                  to={`/live/${broadcast.live_session.id}`}
                  className="podcast-live-card"
                >
                  <div className="podcast-card-artwork">
                    {broadcast
                      .episode
                      .artwork_url ||
                    broadcast
                      .podcast
                      .artwork_url ? (
                      <img
                        src={
                          broadcast
                            .episode
                            .artwork_url ||
                          broadcast
                            .podcast
                            .artwork_url
                        }
                        alt=""
                      />
                    ) : (
                      <div className="podcast-artwork-fallback">
                        ◉
                      </div>
                    )}
                  </div>

                  <div className="podcast-live-card-body">
                    <h4>
                      {
                        broadcast
                          .episode
                          .title
                      }
                    </h4>

                    <p>
                      {
                        broadcast
                          .podcast
                          .title
                      }
                    </p>

                    <span className="podcast-live-when">
                      {formatLiveWhen(
                        broadcast,
                      )}
                    </span>
                  </div>
                </Link>
              ),
            )}
          </div>
        </section>
      ) : null}

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
