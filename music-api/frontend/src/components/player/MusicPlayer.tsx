import { useState } from 'react'

import { usePlayer } from '../../context/PlayerContext'

import QueuePanel from '../player/QueuePanel'

function formatTime(seconds: number) {
  if (!Number.isFinite(seconds) || seconds < 0) {
    return '0:00'
  }

  const minutes = Math.floor(seconds / 60)
  const remainingSeconds = Math.floor(
    seconds % 60,
  )

  return `${minutes}:${remainingSeconds
    .toString()
    .padStart(2, '0')}`
}

function MusicPlayer() {
  const {
    currentTrack,
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
  } = usePlayer()

  const [isQueueOpen, setIsQueueOpen] =
    useState(false)

  const progress =
    duration > 0
      ? Math.min(
          (currentTime / duration) * 100,
          100,
        )
      : 0

  const hasNextTrack =
    currentQueueIndex >= 0 &&
    currentQueueIndex < queue.length - 1

  return (
    <>
      {isQueueOpen && (
        <QueuePanel
          onClose={() =>
            setIsQueueOpen(false)
          }
        />
      )}

      <div className="player">
        <div className="player-track">
          <div className="player-cover">
            {currentTrack?.image_url ? (
              <img
                src={currentTrack.image_url}
                alt={`${currentTrack.song_title} cover`}
              />
            ) : (
              <span>♪</span>
            )}
          </div>

          <div className="player-track-info">
            <strong>
              {currentTrack?.song_title ??
                'Select a song'}
            </strong>

            <span>
              {currentTrack?.artist_name ??
                'Nothing playing'}
            </span>

            {playbackError && (
              <small className="player-error">
                {playbackError}
              </small>
            )}
          </div>
        </div>

        <div className="player-controls">
          <button
            type="button"
            className="player-control"
            aria-label="Previous track"
            disabled={!currentTrack}
            onClick={playPrevious}
          >
            ◀
          </button>

          <button
            type="button"
            className="player-play"
            aria-label={
              isPlaying ? 'Pause' : 'Play'
            }
            onClick={togglePlay}
            disabled={!currentTrack}
          >
            {isPlaying ? '❚❚' : '▶'}
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
            className="player-control player-queue-button"
            aria-label={
              isQueueOpen
                ? 'Close queue'
                : 'Open queue'
            }
            aria-expanded={isQueueOpen}
            onClick={() =>
              setIsQueueOpen(
                (open) => !open,
              )
            }
          >
            ☰
          </button>
        </div>

        <div className="player-progress">
          <span>
            {formatTime(currentTime)}
          </span>

          <button
            type="button"
            className="player-progress-button"
            aria-label="Seek through track"
            disabled={!currentTrack || duration <= 0}
            onClick={(event) => {
              if (
                !currentTrack ||
                duration <= 0
              ) {
                return
              }

              const rect =
                event.currentTarget.getBoundingClientRect()

              const percentage =
                (event.clientX - rect.left) /
                rect.width

              const boundedPercentage =
                Math.max(
                  0,
                  Math.min(percentage, 1),
                )

              seek(
                boundedPercentage * duration,
              )
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

          <span>
            {formatTime(duration)}
          </span>
        </div>
      </div>
    </>
  )
}

export default MusicPlayer
