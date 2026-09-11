import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from "react";

import { useAuth } from "./AuthContext";

import {
  ListenedTimeAccumulator,
  PlaybackTrackingEngine,
  type PlaybackPurpose,
} from "../lib/playbackTracking";

import { PodcastPlaybackTrackingEngine } from "../lib/podcastPlaybackTracking";

import type { Music } from "../types/music";
import type { Podcast, PodcastEpisode } from "../types/podcast";

import {
  isMusicPlayable,
  isPodcastPlayable,
  musicToPlayable,
  podcastEpisodeToPlayable,
  type PlayableItem,
} from "../types/player";

type PlayerContextValue = {
  currentTrack: Music | null;
  currentPodcastEpisode: PodcastEpisode | null;
  currentPlayableItem: PlayableItem | null;

  isPlaying: boolean;
  currentTime: number;
  duration: number;
  playbackError: string;

  /*
   * Why the global player is currently being used.
   *
   * 'listener' (default): genuine listening — playback
   * tracking is enabled (listening history, qualified
   * plays).
   *
   * 'editor-preview': the artist Lyrics Editor is using
   * the player to preview/seek/loop its own track — no
   * listening history, no qualified plays, no trending
   * or analytics signals.
   */
  playbackPurpose: PlaybackPurpose;
  setPlaybackPurpose: (purpose: PlaybackPurpose) => void;

  /*
   * Playback queue.
   *
   * currentTrack is always queue[currentQueueIndex]
   * (or null when currentQueueIndex is -1).
   */
  queue: PlayableItem[];
  currentQueueIndex: number;

  playTrack: (track: Music, contextTracks?: Music[]) => void;

  playPodcastEpisode: (
    podcast: Podcast,
    episode: PodcastEpisode,
    contextEpisodes?: PodcastEpisode[],
    /*
     * Optional resume position in seconds. When provided,
     * playback seeks to this point once audio metadata is
     * available (used by Continue Listening).
     */
    startAtSeconds?: number,
  ) => void;

  playFromQueue: (index: number) => void;
  togglePlay: () => void;
  seek: (time: number) => void;
  stopPlayback: () => void;

  enqueue: (track: Music) => boolean;
  enqueueNext: (track: Music) => boolean;
  removeFromQueue: (index: number) => void;
  clearQueue: () => void;
  reorderQueue: (fromIndex: number, toIndex: number) => void;

  playNext: () => void;
  playPrevious: () => void;

  isQueued: (musicID: number) => boolean;
};

const PlayerContext = createContext<PlayerContextValue | undefined>(undefined);

/*
 * Queue is persisted to sessionStorage so a full page
 * refresh can restore it in a paused state. Autoplay
 * after refresh is intentionally avoided (browser
 * autoplay policies would block it).
 */
const QUEUE_STORAGE_KEY = "music_player_queue";

const PREVIOUS_RESTART_SECONDS = 3;

type PersistedPlayerState = {
  queue: PlayableItem[];
  index: number;
  time: number;
};

function normalizePersistedItem(value: unknown): PlayableItem | null {
  if (!value || typeof value !== "object") {
    return null;
  }

  const record = value as Record<string, unknown>;

  /*
   * Current persisted format.
   */
  if (record.kind === "music" || record.kind === "podcast_episode") {
    if (
      typeof record.key === "string" &&
      typeof record.audio_url === "string" &&
      typeof record.title === "string" &&
      typeof record.subtitle === "string"
    ) {
      return value as PlayableItem;
    }

    return null;
  }

  /*
   * Legacy Phase 3 music-only queue.
   * Upgrade it transparently on first read.
   */
  if (
    typeof record.id === "number" &&
    typeof record.song_title === "string" &&
    typeof record.artist_name === "string" &&
    typeof record.audio_url === "string"
  ) {
    return musicToPlayable(value as Music);
  }

  return null;
}

function readPersistedState(): PersistedPlayerState | null {
  try {
    const raw = sessionStorage.getItem(QUEUE_STORAGE_KEY);

    if (!raw) {
      return null;
    }

    const parsed = JSON.parse(raw) as {
      queue?: unknown;
      index?: unknown;
      time?: unknown;
    };

    if (
      !Array.isArray(parsed.queue) ||
      parsed.queue.length === 0 ||
      typeof parsed.index !== "number"
    ) {
      return null;
    }

    const normalizedQueue = parsed.queue
      .map(normalizePersistedItem)
      .filter((item): item is PlayableItem => item !== null);

    if (normalizedQueue.length === 0) {
      return null;
    }

    return {
      queue: normalizedQueue,
      index: parsed.index,
      time: typeof parsed.time === "number" ? parsed.time : 0,
    };
  } catch {
    // Corrupt or unavailable storage is ignored.
    return null;
  }
}

function persistState(state: PersistedPlayerState) {
  try {
    sessionStorage.setItem(QUEUE_STORAGE_KEY, JSON.stringify(state));
  } catch {
    // Storage may be unavailable (private mode).
  }
}

function clearPersistedState() {
  try {
    sessionStorage.removeItem(QUEUE_STORAGE_KEY);
  } catch {
    // Ignored.
  }
}

type PlayerProviderProps = {
  children: ReactNode;
};

export function PlayerProvider({ children }: PlayerProviderProps) {
  const { isAuthenticated, isLoadingIdentity, token } = useAuth();

  const audioRef = useRef<HTMLAudioElement | null>(null);

  const [queue, setQueue] = useState<PlayableItem[]>([]);

  const [currentQueueIndex, setCurrentQueueIndex] = useState(-1);

  const [isPlaying, setIsPlaying] = useState(false);

  const [currentTime, setCurrentTime] = useState(0);

  const [duration, setDuration] = useState(0);

  const [playbackError, setPlaybackError] = useState("");

  const [playbackPurpose, setPlaybackPurposeState] =
    useState<PlaybackPurpose>("listener");

  /*
   * Refs mirror the queue state so audio event
   * listeners (bound once at mount) never read
   * stale closures. Every mutation updates the
   * refs synchronously alongside the state.
   */
  const queueRef = useRef<PlayableItem[]>([]);
  const indexRef = useRef(-1);
  const timeRef = useRef(0);
  const pendingSeekRef = useRef(0);

  /*
   * Mirrors isPlaying. A failed load can leave the
   * media element internally un-paused while no
   * audio is flowing, so toggle decisions use this
   * ref (the app's notion of playing) rather than
   * audio.paused, which can disagree in that state.
   */
  const isPlayingRef = useRef(false);

  /*
   * True once this provider has held (or restored)
   * a valid current track. Persisted state must
   * only be deleted after that point: the very
   * first render still shows currentQueueIndex -1
   * while the restore effect has not committed yet,
   * and clearing then would erase a valid persisted
   * queue before restoration can consume it (and
   * would leave the live Audio element empty under
   * React StrictMode's effect re-execution).
   */
  const everHadCurrentRef = useRef(false);

  /*
   * =================================================================
   * Playback tracking (Phase 3.8)
   * =================================================================
   *
   * The engine is a plain mutable object, NOT React state:
   * React StrictMode double-invokes effects, and state-
   * driven tracking could create duplicate playback
   * sessions. The engine is created exactly once (lazily)
   * and driven only by events from the one global audio
   * element.
   *
   * Refs mirror everything the engine needs so its
   * callbacks never read stale closures.
   */
  const tokenRef = useRef<string | null>(null);

  tokenRef.current = token;

  const purposeRef = useRef<PlaybackPurpose>("listener");

  purposeRef.current = playbackPurpose;

  const trackingEngineRef = useRef<PlaybackTrackingEngine | null>(null);

  if (!trackingEngineRef.current) {
    trackingEngineRef.current = new PlaybackTrackingEngine({
      getToken: () => tokenRef.current,

      isTrackingSuspended: () => purposeRef.current !== "listener",
    });
  }

  const trackingEngine = trackingEngineRef.current;

  /*
   * Podcast tracking (Phase 4.1): a dedicated engine with
   * the same event API, calling the podcast playback
   * endpoints. Both engines are driven by the ONE global
   * audio element below — whichever kind the current queue
   * item is, only that engine receives tracking events.
   */
  const podcastTrackingEngineRef = useRef<PodcastPlaybackTrackingEngine | null>(null);

  if (!podcastTrackingEngineRef.current) {
    podcastTrackingEngineRef.current = new PodcastPlaybackTrackingEngine({
      getToken: () => tokenRef.current,

      isTrackingSuspended: () => purposeRef.current !== "listener",
    });
  }

  const podcastTrackingEngine = podcastTrackingEngineRef.current;

  const listenedAccumulatorRef = useRef<ListenedTimeAccumulator | null>(null);

  if (!listenedAccumulatorRef.current) {
    listenedAccumulatorRef.current = new ListenedTimeAccumulator();
  }

  const listenedAccumulator = listenedAccumulatorRef.current;

  const setPlaybackPurpose = useCallback((purpose: PlaybackPurpose) => {
    setPlaybackPurposeState(purpose);
  }, []);

  /*
   * Purpose transitions.
   *
   * Switching to editor-preview must immediately stop
   * tracking the ongoing listener session (the Lyrics
   * Editor hijacks the same global audio element).
   *
   * Returning to listener (editor closed) re-arms
   * tracking when audio is genuinely playing: continued
   * listening after closing the editor is real listening
   * and counts from that moment. Artists are not
   * permanently excluded from their own music.
   */
  useEffect(() => {
    if (playbackPurpose !== "listener") {
      trackingEngine.abandon();

      podcastTrackingEngine.abandon();

      listenedAccumulator.reset();

      return;
    }

    const audio = audioRef.current;

    const item =
      indexRef.current >= 0 ? queueRef.current[indexRef.current] : null;

    if (!audio || audio.paused || !item?.audio_url) {
      return;
    }

    /*
     * Re-arm whichever engine matches the current item:
     * continued listening after closing the editor is real
     * listening and counts from that moment.
     */
    if (isMusicPlayable(item)) {
      trackingEngine.trackChanged(
        item.music.id,
        Number.isFinite(audio.duration) ? audio.duration * 1000 : 0,
        audio.currentTime * 1000,
      );

      trackingEngine.playStarted();
    } else if (isPodcastPlayable(item)) {
      podcastTrackingEngine.trackChanged(
        item.episode.id,
        Number.isFinite(audio.duration) ? audio.duration * 1000 : 0,
        audio.currentTime * 1000,
      );

      podcastTrackingEngine.playStarted();
    }
  }, [playbackPurpose, trackingEngine, podcastTrackingEngine, listenedAccumulator]);

  const applyQueue = useCallback(
    (nextQueue: PlayableItem[], nextIndex: number) => {
      queueRef.current = nextQueue;
      indexRef.current = nextIndex;

      setQueue(nextQueue);
      setCurrentQueueIndex(nextIndex);
    },
    [],
  );

  const saveCurrentState = useCallback(() => {
    const currentQueue = queueRef.current;
    const index = indexRef.current;

    if (index < 0 || index >= currentQueue.length) {
      clearPersistedState();
      return;
    }

    persistState({
      queue: currentQueue,
      index,
      time: timeRef.current,
    });
  }, []);

  /*
   * Load the queued track at `index` into the single
   * global audio element and start playback.
   *
   * seekSeconds (podcast resume) is applied through the
   * pending-seek mechanism once metadata is available —
   * never before — and resetting it here also clears any
   * stale pending seek so it can never be applied to a
   * different item.
   */
  const loadAndPlayIndex = useCallback(
    (index: number, seekSeconds = 0) => {
      const audio = audioRef.current;

      const item = queueRef.current[index];

      if (!audio || !item) {
        return;
      }

      pendingSeekRef.current = Math.max(seekSeconds, 0);

      if (!item.audio_url) {
        audio.pause();

        /*
         * If a session was previously open, selecting an
         * unplayable item ends it.
         */
        trackingEngine.finishSession("track-switch");

        podcastTrackingEngine.finishSession("episode-switch");

        listenedAccumulator.reset();

        indexRef.current = index;
        setCurrentQueueIndex(index);

        isPlayingRef.current = false;
        setIsPlaying(false);

        timeRef.current = 0;
        setCurrentTime(0);
        setDuration(0);

        setPlaybackError("This item has no audio available.");

        return;
      }

      audio.pause();

      audio.src = item.audio_url;
      audio.currentTime = 0;

      /*
       * Tracking is routed by playable kind:
       *
       * - music items use the Phase 3.8 music engine,
       * - podcast episodes use the Phase 4.1 podcast
       *   engine and must never be submitted to the music
       *   playback-session API,
       * - switching kinds completes the other engine's
       *   open session.
       */
      if (isMusicPlayable(item)) {
        podcastTrackingEngine.finishSession("episode-switch");

        trackingEngine.trackChanged(item.music.id, 0, 0);
      } else {
        trackingEngine.finishSession("track-switch");

        trackingEngine.abandon();

        if (isPodcastPlayable(item)) {
          podcastTrackingEngine.trackChanged(item.episode.id, 0, 0);
        } else {
          podcastTrackingEngine.finishSession("episode-switch");

          podcastTrackingEngine.abandon();
        }
      }

      listenedAccumulator.reset();

      timeRef.current = 0;
      setCurrentTime(0);
      setDuration(0);
      setPlaybackError("");

      isPlayingRef.current = false;
      setIsPlaying(false);

      indexRef.current = index;
      setCurrentQueueIndex(index);

      void audio
        .play()
        .then(() => {
          setPlaybackError("");
        })
        .catch(() => {
          isPlayingRef.current = false;
          setIsPlaying(false);
        });
    },
    [trackingEngine, podcastTrackingEngine, listenedAccumulator],
  );

  /*
   * Auto-advance when the current track ends.
   * Kept in a ref so the audio element's `ended`
   * listener (registered once) always runs the
   * latest version without stale queue data.
   */
  const handleEndedRef = useRef<() => void>(() => {});

  useEffect(() => {
    handleEndedRef.current = () => {
      const nextIndex = indexRef.current + 1;

      if (nextIndex < queueRef.current.length) {
        loadAndPlayIndex(nextIndex);
        return;
      }

      // Last track: remain ended cleanly.
      saveCurrentState();
    };
  });

  /*
   * Create one audio element for the entire application.
   *
   * This is what prevents multiple songs from playing
   * at the same time. The queue state is restored from
   * sessionStorage in a paused state (no autoplay).
   */
  useEffect(() => {
    const audio = new Audio();

    audio.preload = "metadata";

    audioRef.current = audio;

    const handleTimeUpdate = () => {
      timeRef.current = audio.currentTime;
      setCurrentTime(audio.currentTime);

      /*
       * Real listening accumulation: credit only the
       * position advance that is consistent with
       * wall-clock time (seeking is never credited). The
       * shared accumulator feeds whichever engine matches
       * the current item.
       */
      const listenedDeltaMS = listenedAccumulator.observe(audio.currentTime);

      const item =
        indexRef.current >= 0 ? queueRef.current[indexRef.current] : null;

      if (isPodcastPlayable(item)) {
        podcastTrackingEngine.accumulate(listenedDeltaMS, audio.currentTime);
      } else {
        trackingEngine.accumulate(listenedDeltaMS, audio.currentTime);
      }
    };

    const handleLoadedMetadata = () => {
      setDuration(Number.isFinite(audio.duration) ? audio.duration : 0);

      trackingEngine.onDurationChange(
        Number.isFinite(audio.duration) ? audio.duration * 1000 : 0,
      );

      podcastTrackingEngine.onDurationChange(
        Number.isFinite(audio.duration) ? audio.duration * 1000 : 0,
      );

      // Restored position after a page refresh or a
      // podcast resume. Never applied before metadata.
      if (pendingSeekRef.current > 0) {
        const restoredTime = Math.min(
          pendingSeekRef.current,
          Number.isFinite(audio.duration)
            ? audio.duration
            : pendingSeekRef.current,
        );

        audio.currentTime = restoredTime;
        timeRef.current = restoredTime;
        setCurrentTime(restoredTime);
        pendingSeekRef.current = 0;

        listenedAccumulator.reset();
      }
    };

    const handleDurationChange = () => {
      setDuration(Number.isFinite(audio.duration) ? audio.duration : 0);

      trackingEngine.onDurationChange(
        Number.isFinite(audio.duration) ? audio.duration * 1000 : 0,
      );

      podcastTrackingEngine.onDurationChange(
        Number.isFinite(audio.duration) ? audio.duration * 1000 : 0,
      );
    };

    const handlePlay = () => {
      isPlayingRef.current = true;
      setIsPlaying(true);

      /*
       * Tracking: genuine playback started (also fires
       * on resume after pause — the engine keeps the
       * existing session in that case). Editor-preview
       * playback never creates a session. The event is
       * routed to the engine matching the current item's
       * kind.
       */
      const item =
        indexRef.current >= 0 ? queueRef.current[indexRef.current] : null;

      if (isPodcastPlayable(item)) {
        trackingEngine.abandon();

        if (item.audio_url) {
          podcastTrackingEngine.playStarted({
            episodeID: item.episode.id,
            durationMS: Number.isFinite(audio.duration)
              ? audio.duration * 1000
              : 0,
            positionMS: audio.currentTime * 1000,
          });
        } else {
          podcastTrackingEngine.abandon();
        }
      } else if (isMusicPlayable(item)) {
        podcastTrackingEngine.abandon();

        trackingEngine.playStarted(
          item.audio_url
            ? {
                musicID: item.music.id,
                durationMS: Number.isFinite(audio.duration)
                  ? audio.duration * 1000
                  : 0,
                positionMS: audio.currentTime * 1000,
              }
            : undefined,
        );
      } else {
        trackingEngine.abandon();

        podcastTrackingEngine.abandon();
      }
    };

    const handlePause = () => {
      isPlayingRef.current = false;
      setIsPlaying(false);
      saveCurrentState();

      /*
       * Tracking: the heartbeat pauses; the session
       * stays open because pausing keeps the track and
       * position.
       */
      trackingEngine.paused();

      podcastTrackingEngine.paused();
    };

    const handleEnded = () => {
      isPlayingRef.current = false;
      setIsPlaying(false);
      setCurrentTime(0);
      timeRef.current = 0;

      // Tracking: the media genuinely finished.
      trackingEngine.finishSession("track-end");

      podcastTrackingEngine.finishSession("episode-end");

      handleEndedRef.current();
    };

    const handleError = () => {
      isPlayingRef.current = false;
      setIsPlaying(false);
      setPlaybackError(
        "This track failed to play. Skip to the next song or try another track.",
      );

      /*
       * Tracking: the track cannot play, so it never
       * produced genuine listening worth keeping open.
       */
      trackingEngine.finishSession("stopped");

      podcastTrackingEngine.finishSession("stopped");
    };

    const handlePageHide = () => {
      saveCurrentState();

      /*
       * Tracking: best-effort completion on unload.
       * fetch keeps running during pagehide in modern
       * browsers.
       */
      trackingEngine.finishSession("unload");

      podcastTrackingEngine.finishSession("unload");
    };

    audio.addEventListener("timeupdate", handleTimeUpdate);

    audio.addEventListener("loadedmetadata", handleLoadedMetadata);

    audio.addEventListener("durationchange", handleDurationChange);

    audio.addEventListener("play", handlePlay);

    audio.addEventListener("pause", handlePause);

    audio.addEventListener("ended", handleEnded);

    audio.addEventListener("error", handleError);

    window.addEventListener("pagehide", handlePageHide);

    /*
     * Restore a persisted queue (paused, not autoplayed).
     *
     * Under React StrictMode (dev only) effects are
     * executed, cleaned up and executed again. The first
     * execution restores the queue into state and the
     * refs below survive into the second execution, while
     * the second Audio instance is the one that stays
     * live. When the storage read misses on that second
     * pass, the refs still hold the restored queue, so the
     * live element is armed from them. In production only
     * the storage path runs; the ref fallback is inert.
     */
    const restoreTrack = (item: PlayableItem | undefined, seekTime: number) => {
      if (!item?.audio_url) {
        return;
      }

      pendingSeekRef.current = Math.max(seekTime, 0);

      /*
       * Arm whichever engine matches the restored item's
       * kind; the other engine drops its state.
       */
      if (isMusicPlayable(item)) {
        podcastTrackingEngine.abandon();

        trackingEngine.trackChanged(
          item.music.id,
          0,
          Math.max(seekTime, 0) * 1000,
        );
      } else if (isPodcastPlayable(item)) {
        trackingEngine.abandon();

        podcastTrackingEngine.trackChanged(
          item.episode.id,
          0,
          Math.max(seekTime, 0) * 1000,
        );
      } else {
        trackingEngine.abandon();

        podcastTrackingEngine.abandon();
      }

      listenedAccumulator.reset();

      audio.src = item.audio_url;
    };

    const saved = readPersistedState();

    if (saved) {
      const index = Math.min(Math.max(saved.index, 0), saved.queue.length - 1);

      applyQueue(saved.queue, index);

      restoreTrack(saved.queue[index], saved.time ?? 0);
    } else if (indexRef.current >= 0) {
      const index = indexRef.current;

      // Prefer a restore position that has not been
      // applied yet over the live (still zero) time.
      const restoreTime =
        pendingSeekRef.current > 0 ? pendingSeekRef.current : timeRef.current;

      restoreTrack(queueRef.current[index], restoreTime);
    }

    return () => {
      audio.pause();

      // Tracking: the provider is unmounting; complete
      // any open session best-effort.
      trackingEngine.finishSession("stopped");

      podcastTrackingEngine.finishSession("stopped");

      // Release the resource so the discarded element
      // (a StrictMode re-execution in dev) cannot keep
      // the media loaded in the background.
      audio.removeAttribute("src");
      audio.load();

      audio.removeEventListener("timeupdate", handleTimeUpdate);

      audio.removeEventListener("loadedmetadata", handleLoadedMetadata);

      audio.removeEventListener("durationchange", handleDurationChange);

      audio.removeEventListener("play", handlePlay);

      audio.removeEventListener("pause", handlePause);

      audio.removeEventListener("ended", handleEnded);

      audio.removeEventListener("error", handleError);

      window.removeEventListener("pagehide", handlePageHide);

      audioRef.current = null;
    };
  }, []);

  const currentPlayableItem =
    currentQueueIndex >= 0 && currentQueueIndex < queue.length
      ? queue[currentQueueIndex]
      : null;

  const currentTrack = isMusicPlayable(currentPlayableItem)
    ? currentPlayableItem.music
    : null;

  const currentPodcastEpisode =
    currentPlayableItem?.kind === "podcast_episode"
      ? currentPlayableItem.episode
      : null;

  /*
   * Keep sessionStorage in sync with the queue
   * whenever it changes structurally.
   *
   * Clearing is intentionally guarded: on the
   * initial render currentQueueIndex is still -1
   * (the restore effect commits afterwards), so an
   * unguarded clear would wipe a valid persisted
   * queue before it is restored.
   */
  useEffect(() => {
    if (currentQueueIndex >= 0) {
      everHadCurrentRef.current = true;

      persistState({
        queue,
        index: currentQueueIndex,
        time: timeRef.current,
      });
    } else if (everHadCurrentRef.current) {
      // An intentional transition to an empty
      // queue (e.g. logout) removes the state.
      clearPersistedState();
    }
  }, [currentQueueIndex, queue]);

  /*
   * Resume the current track.
   *
   * After a failed load the audio element keeps its
   * error state and play() alone would never retry the
   * fetch, so the element is re-armed with the same
   * source first. The pending-seek mechanism restores
   * the last known position once metadata loads.
   */
  const resumeCurrent = useCallback(() => {
    const audio = audioRef.current;

    const item =
      indexRef.current >= 0 ? queueRef.current[indexRef.current] : null;

    if (!audio) {
      return;
    }

    if (audio.error && item?.audio_url) {
      // Re-assigning the same src alone can be a no-op
      // on an errored element, so load() forces the
      // resource selection algorithm to run again.
      pendingSeekRef.current = timeRef.current;
      audio.src = item.audio_url;
      audio.load();
    }

    void audio
      .play()
      .then(() => {
        setPlaybackError("");
      })
      .catch(() => {
        isPlayingRef.current = false;
        setIsPlaying(false);
      });
  }, []);

  /*
   * Play a selected track.
   *
   * - Selecting the current track toggles
   *   play/pause (existing behavior).
   *
   * - When contextTracks (a collection such as
   *   the home grid, liked songs, or a playlist)
   *   is provided, that collection REPLACES the
   *   queue and playback starts at the selected
   *   track. This is how "play Song B of A/B/C"
   *   makes Next/Previous walk the collection.
   *
   * - A track already in the current queue is
   *   jumped to instead of being duplicated.
   *
   * - A track outside any context starts a fresh
   *   single-track queue.
   */
  const playTrack = useCallback(
    (track: Music, contextTracks?: Music[]) => {
      const audio = audioRef.current;

      if (!audio || !track.audio_url) {
        return;
      }

      const playable = musicToPlayable(track);

      const currentIndex = indexRef.current;

      const currentItem =
        currentIndex >= 0 ? queueRef.current[currentIndex] : null;

      /*
       * The key includes the media kind, so music id 7
       * and podcast episode id 7 can never collide.
       */
      if (currentItem?.key === playable.key) {
        if (isPlayingRef.current) {
          audio.pause();
        } else {
          resumeCurrent();
        }

        return;
      }

      let nextQueue: PlayableItem[];

      let nextIndex: number;

      if (contextTracks && contextTracks.some((item) => item.id === track.id)) {
        nextQueue = contextTracks.map(musicToPlayable);

        nextIndex = contextTracks.findIndex((item) => item.id === track.id);
      } else {
        const queuedIndex = queueRef.current.findIndex(
          (item) => item.key === playable.key,
        );

        if (queuedIndex >= 0) {
          nextQueue = queueRef.current;

          nextIndex = queuedIndex;
        } else {
          nextQueue = [playable];
          nextIndex = 0;
        }
      }

      applyQueue(nextQueue, nextIndex);

      loadAndPlayIndex(nextIndex);
    },
    [applyQueue, loadAndPlayIndex, resumeCurrent],
  );

  /*
   * Play a podcast episode through the SAME global
   * audio element used by music.
   *
   * contextEpisodes creates a podcast-episode queue so
   * Next/Previous can move naturally through the show.
   *
   * startAtSeconds (Continue Listening resume) seeks to
   * the saved position once metadata is available and
   * starts a NEW playback session for the new listening
   * period — old session ids are never reused.
   */
  const playPodcastEpisode = useCallback(
    (
      podcast: Podcast,
      episode: PodcastEpisode,
      contextEpisodes?: PodcastEpisode[],
      startAtSeconds?: number,
    ) => {
      const audio = audioRef.current;

      if (!audio || !episode.audio_url) {
        return;
      }

      const playable = podcastEpisodeToPlayable(podcast, episode);

      const currentIndex = indexRef.current;

      const currentItem =
        currentIndex >= 0 ? queueRef.current[currentIndex] : null;

      if (currentItem?.key === playable.key) {
        if (isPlayingRef.current) {
          audio.pause();
        } else {
          resumeCurrent();
        }

        return;
      }

      let nextQueue: PlayableItem[];

      let nextIndex: number;

      if (
        contextEpisodes &&
        contextEpisodes.some((item) => item.id === episode.id)
      ) {
        nextQueue = contextEpisodes
          .filter((item) => Boolean(item.audio_url))
          .map((item) => podcastEpisodeToPlayable(podcast, item));

        nextIndex = nextQueue.findIndex((item) => item.key === playable.key);
      } else {
        const queuedIndex = queueRef.current.findIndex(
          (item) => item.key === playable.key,
        );

        if (queuedIndex >= 0) {
          nextQueue = queueRef.current;

          nextIndex = queuedIndex;
        } else {
          nextQueue = [playable];

          nextIndex = 0;
        }
      }

      if (nextIndex < 0) {
        return;
      }

      applyQueue(nextQueue, nextIndex);

      loadAndPlayIndex(
        nextIndex,
        typeof startAtSeconds === "number" && Number.isFinite(startAtSeconds)
          ? Math.max(0, startAtSeconds)
          : 0,
      );
    },
    [applyQueue, loadAndPlayIndex, resumeCurrent],
  );

  /*
   * Jump to a track in the queue. Clicking the
   * current item toggles play/pause.
   */
  const playFromQueue = useCallback(
    (index: number) => {
      if (index < 0 || index >= queueRef.current.length) {
        return;
      }

      if (index === indexRef.current) {
        const audio = audioRef.current;

        if (!audio) {
          return;
        }

        if (isPlayingRef.current) {
          audio.pause();
        } else {
          resumeCurrent();
        }

        return;
      }

      loadAndPlayIndex(index);
    },
    [loadAndPlayIndex, resumeCurrent],
  );

  /*
   * Play or pause the currently selected track.
   *
   * The decision uses the app's playing state rather
   * than audio.paused: a failed load can leave the
   * media element internally un-paused with no audio
   * flowing, and trusting it would turn the next play
   * button press into a pause.
   */
  const togglePlay = useCallback(() => {
    const audio = audioRef.current;

    if (!audio || indexRef.current < 0) {
      return;
    }

    if (isPlayingRef.current) {
      audio.pause();
    } else {
      resumeCurrent();
    }
  }, [resumeCurrent]);

  /*
   * Move playback to a specific time.
   */
  const seek = useCallback((time: number) => {
    const audio = audioRef.current;

    if (!audio || !Number.isFinite(time)) {
      return;
    }

    const maxDuration = Number.isFinite(audio.duration) ? audio.duration : 0;

    const nextTime = Math.max(0, Math.min(time, maxDuration));

    audio.currentTime = nextTime;
    timeRef.current = nextTime;

    setCurrentTime(nextTime);
  }, []);

  /*
   * Completely stop and clear playback.
   *
   * This is different from pause.
   * Pause keeps the selected track and position.
   *
   * stopPlayback removes the current song
   * completely. We use this on logout.
   */
  const stopPlayback = useCallback(() => {
    const audio = audioRef.current;

    // Tracking: playback intentionally stopped.
    trackingEngine.finishSession("stopped");

    podcastTrackingEngine.finishSession("stopped");

    listenedAccumulator.reset();

    if (audio) {
      audio.pause();
      audio.currentTime = 0;

      audio.removeAttribute("src");
      audio.load();
    }

    timeRef.current = 0;

    applyQueue([], -1);
    isPlayingRef.current = false;
    setIsPlaying(false);
    setCurrentTime(0);
    setDuration(0);
    setPlaybackError("");

    clearPersistedState();
  }, [applyQueue, trackingEngine, podcastTrackingEngine, listenedAccumulator]);

  /*
   * Add a track to the end of the queue.
   *
   * Manual enqueues may contain the same song more
   * than once (explicit user intent). If nothing is
   * playing, the track starts immediately as a
   * single-track queue.
   */
  const enqueue = useCallback(
    (track: Music) => {
      if (!track.audio_url) {
        return false;
      }

      if (indexRef.current < 0) {
        playTrack(track);
        return true;
      }

      applyQueue(
        [...queueRef.current, musicToPlayable(track)],
        indexRef.current,
      );

      return true;
    },
    [applyQueue, playTrack],
  );

  /*
   * Insert a track directly after the current one,
   * so it becomes the next song to play.
   */
  const enqueueNext = useCallback(
    (track: Music) => {
      if (!track.audio_url) {
        return false;
      }

      if (indexRef.current < 0) {
        playTrack(track);
        return true;
      }

      const insertAt = indexRef.current + 1;

      const nextQueue = [...queueRef.current];

      nextQueue.splice(insertAt, 0, musicToPlayable(track));

      applyQueue(nextQueue, indexRef.current);

      return true;
    },
    [applyQueue, playTrack],
  );

  /*
   * Remove a track from the queue.
   *
   * - Removing an upcoming or previous track keeps
   *   the current song playing; the index shifts
   *   to stay pointed at the same track.
   *
   * - Removing the current track starts the track
   *   that follows it (or stops playback when the
   *   queue becomes empty).
   */
  const removeFromQueue = useCallback(
    (index: number) => {
      const currentQueue = queueRef.current;

      if (index < 0 || index >= currentQueue.length) {
        return;
      }

      const currentIndex = indexRef.current;

      const nextQueue = [...currentQueue];

      nextQueue.splice(index, 1);

      if (index === currentIndex) {
        if (nextQueue.length === 0) {
          stopPlayback();
          return;
        }

        const nextIndex = Math.min(index, nextQueue.length - 1);

        applyQueue(nextQueue, nextIndex);
        loadAndPlayIndex(nextIndex);
        return;
      }

      const nextIndex = index < currentIndex ? currentIndex - 1 : currentIndex;

      applyQueue(nextQueue, nextIndex);
    },
    [applyQueue, loadAndPlayIndex, stopPlayback],
  );

  /*
   * Clear the queue.
   *
   * The currently playing song keeps playing and
   * becomes the only remaining queue item. With
   * nothing playing the queue is emptied entirely.
   */
  const clearQueue = useCallback(() => {
    const currentIndex = indexRef.current;

    if (currentIndex < 0) {
      applyQueue([], -1);
      clearPersistedState();
      return;
    }

    const currentTrack = queueRef.current[currentIndex];

    applyQueue([currentTrack], 0);
  }, [applyQueue]);

  /*
   * Move a queue item from one position to another.
   *
   * The current track keeps playing uninterrupted;
   * the index is adjusted so it still points at the
   * same track after the move.
   */
  const reorderQueue = useCallback(
    (fromIndex: number, toIndex: number) => {
      const currentQueue = queueRef.current;

      if (
        fromIndex === toIndex ||
        fromIndex < 0 ||
        toIndex < 0 ||
        fromIndex >= currentQueue.length ||
        toIndex >= currentQueue.length
      ) {
        return;
      }

      const nextQueue = [...currentQueue];

      const [moved] = nextQueue.splice(fromIndex, 1);

      nextQueue.splice(toIndex, 0, moved);

      const currentIndex = indexRef.current;

      let nextIndex = currentIndex;

      if (fromIndex === currentIndex) {
        nextIndex = toIndex;
      } else {
        if (fromIndex < currentIndex) {
          nextIndex -= 1;
        }

        if (toIndex <= nextIndex) {
          nextIndex += 1;
        }
      }

      applyQueue(nextQueue, nextIndex);
    },
    [applyQueue],
  );

  /*
   * Skip to the next queued track.
   * No-op at the end of the queue.
   */
  const playNext = useCallback(() => {
    const nextIndex = indexRef.current + 1;

    if (nextIndex < queueRef.current.length) {
      loadAndPlayIndex(nextIndex);
    }
  }, [loadAndPlayIndex]);

  /*
   * Previous-track behavior:
   *
   * - more than ~3 seconds into the song:
   *   restart the current track
   * - otherwise: go to the previous queued track,
   *   or restart when already at the first track
   */
  const playPrevious = useCallback(() => {
    const audio = audioRef.current;
    const currentIndex = indexRef.current;

    if (!audio || currentIndex < 0) {
      return;
    }

    if (audio.currentTime > PREVIOUS_RESTART_SECONDS || currentIndex === 0) {
      audio.currentTime = 0;
      timeRef.current = 0;
      setCurrentTime(0);
      return;
    }

    loadAndPlayIndex(currentIndex - 1);
  }, [loadAndPlayIndex]);

  const isQueued = useCallback(
    (musicID: number) => {
      return queue.some(
        (item) => isMusicPlayable(item) && item.music.id === musicID,
      );
    },
    [queue],
  );

  /*
   * Playback (including the queue) must never leak
   * between sessions. HomePage already stops
   * playback on logout; this guards every other
   * path where an authenticated session ends.
   */
  const wasAuthenticatedRef = useRef(false);

  useEffect(() => {
    if (wasAuthenticatedRef.current && !isLoadingIdentity && !isAuthenticated) {
      stopPlayback();
    }

    wasAuthenticatedRef.current = isAuthenticated;
  }, [isAuthenticated, isLoadingIdentity, stopPlayback]);

  return (
    <PlayerContext.Provider
      value={{
        currentTrack,
        currentPodcastEpisode,
        currentPlayableItem,
        isPlaying,
        currentTime,
        duration,
        playbackError,
        playbackPurpose,
        setPlaybackPurpose,
        queue,
        currentQueueIndex,
        playTrack,
        playPodcastEpisode,
        playFromQueue,
        togglePlay,
        seek,
        stopPlayback,
        enqueue,
        enqueueNext,
        removeFromQueue,
        clearQueue,
        reorderQueue,
        playNext,
        playPrevious,
        isQueued,
      }}
    >
      {children}
    </PlayerContext.Provider>
  );
}

export function usePlayer() {
  const context = useContext(PlayerContext);

  if (!context) {
    throw new Error("usePlayer must be used inside PlayerProvider");
  }

  return context;
}
