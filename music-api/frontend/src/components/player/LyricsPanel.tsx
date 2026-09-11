import {
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react'

import {
  APIError,
  getTrackLyrics,
} from '../../lib/api'

import type { TrackLyrics } from '../../types/lyrics'

type LyricsPanelProps = {
  musicID: number
  songTitle: string
  artistName: string
  currentTime: number
  onSeek: (time: number) => void
  onClose: () => void
}

function LyricsPanel({
  musicID,
  songTitle,
  artistName,
  currentTime,
  onSeek,
  onClose,
}: LyricsPanelProps) {
  const [lyrics, setLyrics] =
    useState<TrackLyrics | null>(null)

  const [isLoading, setIsLoading] =
    useState(true)

  const [isUnavailable, setIsUnavailable] =
    useState(false)

  const [error, setError] =
    useState('')

  const lineRefs =
    useRef<Array<HTMLButtonElement | null>>([])

  const previousActiveIndexRef =
    useRef(-1)

  useEffect(() => {
    let cancelled = false

    setLyrics(null)
    setIsLoading(true)
    setIsUnavailable(false)
    setError('')

    getTrackLyrics(musicID)
      .then((response) => {
        if (cancelled) {
          return
        }

        setLyrics(response)
      })
      .catch((requestError: unknown) => {
        if (cancelled) {
          return
        }

        if (
          requestError instanceof APIError &&
          requestError.status === 404
        ) {
          setIsUnavailable(true)
          return
        }

        if (requestError instanceof Error) {
          setError(requestError.message)
          return
        }

        setError(
          'Unable to load lyrics right now.',
        )
      })
      .finally(() => {
        if (!cancelled) {
          setIsLoading(false)
        }
      })

    return () => {
      cancelled = true
    }
  }, [musicID])

  const currentTimeMS =
    Math.max(0, currentTime * 1000)

  const activeLineIndex =
    useMemo(() => {
      if (
        !lyrics ||
        lyrics.synced_lines.length === 0
      ) {
        return -1
      }

      let activeIndex = -1

      for (
        let index = 0;
        index < lyrics.synced_lines.length;
        index += 1
      ) {
        if (
          lyrics.synced_lines[index].time_ms <=
          currentTimeMS
        ) {
          activeIndex = index
        } else {
          break
        }
      }

      // Lines with an explicit end time (merged in the
      // lyrics editor) stop being active once playback
      // passes that end and no later line has begun.
      if (activeIndex >= 0) {
        const activeLine =
          lyrics.synced_lines[
            activeIndex
          ]

        if (
          activeLine.end_ms !==
            undefined &&
          currentTimeMS >=
            activeLine.end_ms
        ) {
          return -1
        }
      }

      return activeIndex
    }, [
      currentTimeMS,
      lyrics,
    ])

  useEffect(() => {
    if (
      activeLineIndex < 0 ||
      activeLineIndex ===
        previousActiveIndexRef.current
    ) {
      return
    }

    previousActiveIndexRef.current =
      activeLineIndex

    lineRefs.current[
      activeLineIndex
    ]?.scrollIntoView({
      behavior: 'smooth',
      block: 'center',
    })
  }, [activeLineIndex])

  useEffect(() => {
    previousActiveIndexRef.current = -1
    lineRefs.current = []
  }, [musicID])

  const hasSyncedLyrics =
    Boolean(
      lyrics &&
        lyrics.synced_lines.length > 0,
    )

  const hasPlainLyrics =
    Boolean(
      lyrics?.plain_lyrics.trim(),
    )

  return (
    <section
      className="lyrics-panel"
      aria-label={`Lyrics for ${songTitle}`}
    >
      <div className="lyrics-panel-header">
        <div>
          <span className="lyrics-panel-label">
            Lyrics
          </span>

          <strong>
            {songTitle}
          </strong>

          <small>
            {artistName}
          </small>
        </div>

        <button
          type="button"
          className="lyrics-close-button"
          aria-label="Close lyrics"
          onClick={onClose}
        >
          ×
        </button>
      </div>

      <div className="lyrics-panel-body">
        {isLoading && (
          <div className="lyrics-state">
            <span className="lyrics-loading-dot" />

            <p>
              Loading lyrics…
            </p>
          </div>
        )}

        {!isLoading &&
          isUnavailable && (
            <div className="lyrics-state">
              <strong>
                No lyrics yet
              </strong>

              <p>
                Lyrics aren&apos;t available for
                this track yet.
              </p>
            </div>
          )}

        {!isLoading &&
          error && (
            <div className="lyrics-state lyrics-state-error">
              <strong>
                Couldn&apos;t load lyrics
              </strong>

              <p>
                {error}
              </p>
            </div>
          )}

        {!isLoading &&
          !error &&
          !isUnavailable &&
          hasSyncedLyrics &&
          lyrics && (
            <div className="synced-lyrics">
              {lyrics.synced_lines.map(
                (line, index) => {
                  const isActive =
                    index === activeLineIndex

                  const isPast =
                    activeLineIndex >= 0 &&
                    index < activeLineIndex

                  return (
                    <button
                      key={`${line.time_ms}-${index}`}
                      ref={(element) => {
                        lineRefs.current[index] =
                          element
                      }}
                      type="button"
                      className={[
                        'synced-lyric-line',
                        isActive
                          ? 'is-active'
                          : '',
                        isPast
                          ? 'is-past'
                          : '',
                      ]
                        .filter(Boolean)
                        .join(' ')}
                      aria-current={
                        isActive
                          ? 'true'
                          : undefined
                      }
                      onClick={() =>
                        onSeek(
                          line.time_ms / 1000,
                        )
                      }
                    >
                      {line.text}
                    </button>
                  )
                },
              )}
            </div>
          )}

        {!isLoading &&
          !error &&
          !isUnavailable &&
          !hasSyncedLyrics &&
          hasPlainLyrics &&
          lyrics && (
            <div className="plain-lyrics">
              {lyrics.plain_lyrics}
            </div>
          )}

        {!isLoading &&
          !error &&
          !isUnavailable &&
          !hasSyncedLyrics &&
          !hasPlainLyrics && (
            <div className="lyrics-state">
              <strong>
                No lyrics yet
              </strong>

              <p>
                Lyrics aren&apos;t available for
                this track yet.
              </p>
            </div>
          )}
      </div>

      {hasSyncedLyrics && (
        <div className="lyrics-panel-footer">
          Tap any line to jump to that part of the
          song.
        </div>
      )}
    </section>
  )
}

export default LyricsPanel
