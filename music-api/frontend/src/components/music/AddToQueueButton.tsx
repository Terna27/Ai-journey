import { useState } from 'react'

import { usePlayer } from '../../context/PlayerContext'

import type { Music } from '../../types/music'

type AddToQueueButtonProps = {
  track: Music
  variant?: 'compact' | 'full'
}

/*
 * Adds a single song to the playback queue.
 * "Play Next" inserts it directly after the
 * current song instead of at the end.
 */
function AddToQueueButton({
  track,
  variant = 'compact',
}: AddToQueueButtonProps) {
  const {
    enqueue,
    enqueueNext,
  } = usePlayer()

  const [statusMessage, setStatusMessage] =
    useState('')

  function reportAdded(
    playedNext: boolean,
  ) {
    setStatusMessage(
      playedNext
        ? 'Playing next.'
        : 'Added to queue.',
    )

    window.setTimeout(
      () => {
        setStatusMessage('')
      },
      1600,
    )
  }

  function handleEnqueue() {
    if (!enqueue(track)) {
      setStatusMessage(
        'This track has no audio to play.',
      )
      return
    }

    reportAdded(false)
  }

  function handleEnqueueNext() {
    if (!enqueueNext(track)) {
      setStatusMessage(
        'This track has no audio to play.',
      )
      return
    }

    reportAdded(true)
  }

  if (variant === 'full') {
    return (
      <div className="add-to-queue">
        <button
          type="button"
          className="secondary-button add-to-queue-trigger"
          aria-label={`Add ${track.song_title} to queue`}
          disabled={!track.audio_url}
          onClick={handleEnqueue}
        >
          Add to Queue
        </button>

        <button
          type="button"
          className="secondary-button add-to-queue-trigger"
          aria-label={`Play ${track.song_title} next`}
          disabled={!track.audio_url}
          onClick={handleEnqueueNext}
        >
          Play Next
        </button>

        {statusMessage && (
          <span className="add-to-queue-status">
            {statusMessage}
          </span>
        )}
      </div>
    )
  }

  return (
    <div className="add-to-queue">
      <button
        type="button"
        className="text-button add-to-queue-compact"
        aria-label={`Add ${track.song_title} to queue`}
        disabled={!track.audio_url}
        onClick={handleEnqueue}
      >
        + Queue
      </button>

      {statusMessage && (
        <span className="add-to-queue-status">
          {statusMessage}
        </span>
      )}
    </div>
  )
}

export default AddToQueueButton
