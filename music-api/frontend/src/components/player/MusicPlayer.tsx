import { useEffect, useState } from "react";

import { usePlayer } from "../../context/PlayerContext";

import LyricsPanel from "./LyricsPanel";
import QueuePanel from "./QueuePanel";

function formatTime(seconds: number) {
  if (!Number.isFinite(seconds) || seconds < 0) {
    return "0:00";
  }

  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = Math.floor(seconds % 60);

  return `${minutes}:${remainingSeconds.toString().padStart(2, "0")}`;
}

function MusicPlayer() {
  const {
    currentTrack,
    currentPlayableItem,
    isPlaying,
    currentTime,
    duration,
    playbackError,
    queue,
    currentQueueIndex,
    togglePlay,
    seek,
    playNext,
    playPrevious,
  } = usePlayer();

  const [isQueueOpen, setIsQueueOpen] = useState(false);

  const [isLyricsOpen, setIsLyricsOpen] = useState(false);

  /*
   * If playback is cleared completely, any auxiliary
   * player panels should close as well.
   */
  useEffect(() => {
    if (!currentPlayableItem) {
      setIsLyricsOpen(false);
      setIsQueueOpen(false);
    }

    if (!currentTrack) {
      setIsLyricsOpen(false);
    }
  }, [currentPlayableItem, currentTrack]);

  const progress =
    duration > 0 ? Math.min((currentTime / duration) * 100, 100) : 0;

  const hasNextTrack =
    currentQueueIndex >= 0 && currentQueueIndex < queue.length - 1;

  const openQueue = () => {
    setIsLyricsOpen(false);

    setIsQueueOpen((open) => !open);
  };

  const openLyrics = () => {
    if (!currentTrack) {
      return;
    }

    setIsQueueOpen(false);

    setIsLyricsOpen((open) => !open);
  };

  return (
    <>
      {isQueueOpen && <QueuePanel onClose={() => setIsQueueOpen(false)} />}

      {isLyricsOpen && currentTrack && (
        <LyricsPanel
          musicID={currentTrack.id}
          songTitle={currentTrack.song_title}
          artistName={currentTrack.artist_name}
          currentTime={currentTime}
          onSeek={seek}
          onClose={() => setIsLyricsOpen(false)}
        />
      )}

      <div className="player">
        <div className="player-track">
          <div className="player-cover">
            {currentPlayableItem?.artwork_url ? (
              <img
                src={currentPlayableItem.artwork_url}
                alt={`${currentPlayableItem.title} artwork`}
              />
            ) : (
              <span>♪</span>
            )}
          </div>

          <div className="player-track-info">
            <strong>
              {currentPlayableItem?.title ?? "Select something to play"}
            </strong>

            <span>{currentPlayableItem?.subtitle ?? "Nothing playing"}</span>

            {playbackError && (
              <small className="player-error">{playbackError}</small>
            )}
          </div>
        </div>

        <div className="player-controls">
          <button
            type="button"
            className="player-control"
            aria-label="Previous track"
            disabled={!currentPlayableItem}
            onClick={playPrevious}
          >
            ◀
          </button>

          <button
            type="button"
            className="player-play"
            aria-label={isPlaying ? "Pause" : "Play"}
            onClick={togglePlay}
            disabled={!currentPlayableItem}
          >
            {isPlaying ? "❚❚" : "▶"}
          </button>

          <button
            type="button"
            className="player-control"
            aria-label="Next track"
            disabled={!hasNextTrack}
            onClick={playNext}
          >
            ▶
          </button>

          <button
            type="button"
            className="player-control player-lyrics-button"
            aria-label={isLyricsOpen ? "Close lyrics" : "Open lyrics"}
            aria-expanded={isLyricsOpen}
            disabled={!currentTrack}
            onClick={openLyrics}
          >
            ♪
          </button>

          <button
            type="button"
            className="player-control player-queue-button"
            aria-label={isQueueOpen ? "Close queue" : "Open queue"}
            aria-expanded={isQueueOpen}
            onClick={openQueue}
          >
            ☰
          </button>
        </div>

        <div className="player-progress">
          <span>{formatTime(currentTime)}</span>

          <button
            type="button"
            className="player-progress-button"
            aria-label="Seek through track"
            disabled={!currentPlayableItem || duration <= 0}
            onClick={(event) => {
              if (!currentPlayableItem || duration <= 0) {
                return;
              }

              const rect = event.currentTarget.getBoundingClientRect();

              const percentage = (event.clientX - rect.left) / rect.width;

              const boundedPercentage = Math.max(0, Math.min(percentage, 1));

              seek(boundedPercentage * duration);
            }}
          >
            <span className="progress-track">
              <span
                className="progress-value"
                style={{
                  width: `${progress}%`,
                }}
              />
            </span>
          </button>

          <span>{formatTime(duration)}</span>
        </div>
      </div>
    </>
  );
}

export default MusicPlayer;
