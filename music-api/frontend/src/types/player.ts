import type { Music } from "./music";
import type { Podcast, PodcastEpisode } from "./podcast";

export type MusicPlayableItem = {
  kind: "music";
  key: string;
  audio_url: string;
  title: string;
  subtitle: string;
  artwork_url: string;
  music: Music;
};

export type PodcastPlayableItem = {
  kind: "podcast_episode";
  key: string;
  audio_url: string;
  title: string;
  subtitle: string;
  artwork_url: string;
  podcast: Podcast;
  episode: PodcastEpisode;
};

export type PlayableItem = MusicPlayableItem | PodcastPlayableItem;

export function musicToPlayable(music: Music): MusicPlayableItem {
  return {
    kind: "music",
    key: `music:${music.id}`,
    audio_url: music.audio_url,
    title: music.song_title,
    subtitle: music.artist_name,
    artwork_url: music.image_url,
    music,
  };
}

export function podcastEpisodeToPlayable(
  podcast: Podcast,
  episode: PodcastEpisode,
): PodcastPlayableItem {
  return {
    kind: "podcast_episode",
    key: `podcast_episode:${episode.id}`,
    audio_url: episode.audio_url,
    title: episode.title,
    subtitle: podcast.title,
    artwork_url: episode.artwork_url || podcast.artwork_url,
    podcast,
    episode,
  };
}

export function isMusicPlayable(
  item: PlayableItem | null | undefined,
): item is MusicPlayableItem {
  return item?.kind === "music";
}

export function isPodcastPlayable(
  item: PlayableItem | null | undefined,
): item is PodcastPlayableItem {
  return item?.kind === "podcast_episode";
}
