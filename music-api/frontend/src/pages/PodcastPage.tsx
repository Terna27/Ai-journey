import { useEffect, useState } from "react";

import { Link, useParams } from "react-router-dom";

import { usePlayer } from "../context/PlayerContext";

import { APIError, getPublicPodcastBySlug } from "../lib/api";

import type { PodcastDetails, PodcastEpisode } from "../types/podcast";

function formatDate(value: string | null) {
  if (!value) {
    return "";
  }

  const date = new Date(value);

  if (Number.isNaN(date.getTime())) {
    return "";
  }

  return new Intl.DateTimeFormat("en", {
    day: "numeric",
    month: "short",
    year: "numeric",
  }).format(date);
}

function formatEpisodeLabel(episode: PodcastEpisode) {
  const parts: string[] = [];

  if (episode.season_number > 0) {
    parts.push(`S${episode.season_number}`);
  }

  if (episode.episode_number) {
    parts.push(`E${episode.episode_number}`);
  }

  return parts.join(" · ");
}

function PodcastPage() {
  const { slug } = useParams();

  const { currentPlayableItem, isPlaying, playPodcastEpisode } = usePlayer();

  const [details, setDetails] = useState<PodcastDetails | null>(null);

  const [isLoading, setIsLoading] = useState(true);

  const [error, setError] = useState("");

  const [notFound, setNotFound] = useState(false);

  useEffect(() => {
    if (!slug) {
      setIsLoading(false);
      setNotFound(true);
      return;
    }

    let cancelled = false;

    async function loadPodcast() {
      try {
        setIsLoading(true);
        setError("");
        setNotFound(false);

        const response = await getPublicPodcastBySlug(slug!);

        if (!cancelled) {
          setDetails(response);
        }
      } catch (err) {
        if (cancelled) {
          return;
        }

        if (err instanceof APIError && err.status === 404) {
          setNotFound(true);
          return;
        }

        setError(err instanceof Error ? err.message : "Failed to load podcast");
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    }

    void loadPodcast();

    return () => {
      cancelled = true;
    };
  }, [slug]);

  if (isLoading) {
    return (
      <section className="content-panel">
        <p>Loading podcast...</p>
      </section>
    );
  }

  if (notFound) {
    return (
      <section className="content-panel">
        <h3>Podcast not found</h3>

        <p>
          This podcast does not exist or is not published.{" "}
          <Link to="/podcasts">Browse podcasts</Link>.
        </p>
      </section>
    );
  }

  if (error) {
    return (
      <section className="content-panel">
        <h3>Podcast unavailable</h3>

        <p>{error}</p>
      </section>
    );
  }

  if (!details) {
    return null;
  }

  const { podcast, episodes } = details;

  return (
    <>
      <header className="podcast-page-header">
        <div className="podcast-page-artwork">
          {podcast.artwork_url ? (
            <img src={podcast.artwork_url} alt={`${podcast.title} artwork`} />
          ) : (
            <div className="podcast-artwork-fallback">◉</div>
          )}
        </div>

        <div className="podcast-page-info">
          <p className="eyebrow">PODCAST</p>

          <h2>{podcast.title}</h2>

          <div className="podcast-page-meta">
            {podcast.category && <span>{podcast.category}</span>}

            <span>
              {episodes.length} {episodes.length === 1 ? "episode" : "episodes"}
            </span>

            {podcast.is_explicit && (
              <span className="podcast-explicit-badge">E</span>
            )}
          </div>

          {podcast.description && (
            <p className="podcast-description">{podcast.description}</p>
          )}
        </div>
      </header>

      <section className="podcast-episodes-section">
        <div className="podcast-section-heading">
          <div>
            <p className="eyebrow">EPISODES</p>

            <h3>Latest episodes</h3>
          </div>
        </div>

        {episodes.length === 0 ? (
          <div className="content-panel">
            <p>No published episodes yet.</p>
          </div>
        ) : (
          <div className="podcast-episode-list">
            {episodes.map((episode) => {
              const episodeLabel = formatEpisodeLabel(episode);

              const publishedDate = formatDate(episode.published_at);

              return (
                <article key={episode.id} className="podcast-episode-card">
                  <div className="podcast-episode-artwork">
                    {episode.artwork_url || podcast.artwork_url ? (
                      <img
                        src={episode.artwork_url || podcast.artwork_url}
                        alt={`${episode.title} artwork`}
                      />
                    ) : (
                      <span>◉</span>
                    )}
                  </div>

                  <div className="podcast-episode-content">
                    <div className="podcast-episode-heading">
                      <div>
                        <div className="podcast-episode-meta">
                          {episodeLabel && <span>{episodeLabel}</span>}

                          <span>{episode.episode_type}</span>

                          {publishedDate && <span>{publishedDate}</span>}

                          {episode.is_explicit && (
                            <span className="podcast-explicit-badge">E</span>
                          )}
                        </div>

                        <h4>{episode.title}</h4>
                      </div>

                      <button
                        type="button"
                        className="podcast-play-button"
                        disabled={!episode.audio_url}
                        aria-label={
                          currentPlayableItem?.kind === "podcast_episode" &&
                          currentPlayableItem.episode.id === episode.id &&
                          isPlaying
                            ? `Pause ${episode.title}`
                            : `Play ${episode.title}`
                        }
                        onClick={() =>
                          playPodcastEpisode(podcast, episode, episodes)
                        }
                      >
                        {currentPlayableItem?.kind === "podcast_episode" &&
                        currentPlayableItem.episode.id === episode.id &&
                        isPlaying
                          ? "❚❚ Pause"
                          : "▶ Play"}
                      </button>
                    </div>

                    {episode.description && (
                      <p className="podcast-episode-description">
                        {episode.description}
                      </p>
                    )}
                  </div>
                </article>
              );
            })}
          </div>
        )}
      </section>
    </>
  );
}

export default PodcastPage;
