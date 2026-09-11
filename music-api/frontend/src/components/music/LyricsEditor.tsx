import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type KeyboardEvent,
  type MouseEvent,
} from 'react'

import { usePlayer } from '../../context/PlayerContext'

import {
  APIError,
  deleteTrackLyrics,
  getTrackLyrics,
  saveTrackLyrics,
} from '../../lib/api'

import type {
  LyricLine,
  TrackLyrics,
} from '../../types/lyrics'

import type { Music } from '../../types/music'

import './LyricsEditorWorkstation.css'

type LyricsEditorProps = {
  track: Music
  token: string
  onClose: () => void
}

type EditableLyricLine = {
  id: number
  timestamp: string
  endTimestamp: string
  text: string
}

const SEGMENT_SIZE_MS = 5000
const MIN_SEGMENT_SIZE_MS = 100

let nextLineID = 1

function createLine(
  timestamp = '',
  text = '',
  endTimestamp = '',
): EditableLyricLine {
  const line = {
    id: nextLineID,
    timestamp,
    endTimestamp,
    text,
  }

  nextLineID += 1

  return line
}

function millisecondsToTimestamp(
  milliseconds: number,
) {
  const safeMilliseconds = Math.max(
    0,
    Math.floor(milliseconds),
  )

  const minutes = Math.floor(
    safeMilliseconds / 60000,
  )

  const seconds = Math.floor(
    (safeMilliseconds % 60000) / 1000,
  )

  const remainder =
    safeMilliseconds % 1000

  return `${minutes}:${seconds
    .toString()
    .padStart(2, '0')}.${remainder
    .toString()
    .padStart(3, '0')}`
}

function timestampToMilliseconds(
  value: string,
): number | null {
  const trimmed = value.trim()

  const match =
    /^(\d+):([0-5]\d)(?:\.(\d{1,3}))?$/.exec(
      trimmed,
    )

  if (!match) {
    return null
  }

  const minutes = Number(match[1])
  const seconds = Number(match[2])

  const fraction =
    match[3] ?? ''

  const milliseconds =
    fraction.length === 0
      ? 0
      : Number(
          fraction.padEnd(3, '0'),
        )

  return (
    minutes * 60000 +
    seconds * 1000 +
    milliseconds
  )
}

function createDefaultBoundaries(
  durationMS: number,
) {
  if (
    !Number.isFinite(durationMS) ||
    durationMS <= 0
  ) {
    return []
  }

  const boundaries = [0]

  for (
    let time = SEGMENT_SIZE_MS;
    time < durationMS;
    time += SEGMENT_SIZE_MS
  ) {
    boundaries.push(time)
  }

  boundaries.push(durationMS)

  return boundaries
}

function getAdjustmentStep(
  shiftKey: boolean,
  controlKey: boolean,
  metaKey: boolean,
) {
  if (controlKey || metaKey) {
    return 1000
  }

  if (shiftKey) {
    return 500
  }

  return 100
}

function LyricsEditor({
  track,
  token,
  onClose,
}: LyricsEditorProps) {
  const {
    currentTrack,
    currentTime,
    duration,
    isPlaying,
    playTrack,
    setPlaybackPurpose,
    togglePlay,
    seek,
  } = usePlayer()

  const syncedSectionRef =
    useRef<HTMLElement | null>(null)

  const lyricInputRefs =
    useRef<
      Map<
        number,
        HTMLInputElement
      >
    >(new Map())

  const [
    pendingFocusLineID,
    setPendingFocusLineID,
  ] = useState<number | null>(
    null,
  )

  const [
    plainLyrics,
    setPlainLyrics,
  ] = useState('')

  const [lines, setLines] =
    useState<
      EditableLyricLine[]
    >([])

  const [
    selectedLineIDs,
    setSelectedLineIDs,
  ] = useState<Set<number>>(
    new Set(),
  )

  const [
    loading,
    setLoading,
  ] = useState(true)

  const [
    saving,
    setSaving,
  ] = useState(false)

  const [
    deleting,
    setDeleting,
  ] = useState(false)

  const [
    hasExistingLyrics,
    setHasExistingLyrics,
  ] = useState(false)

  const [error, setError] =
    useState('')

  const [
    success,
    setSuccess,
  ] = useState('')

  const [
    segmentBoundaries,
    setSegmentBoundaries,
  ] = useState<number[]>([])

  const [
    selectedSegmentIndex,
    setSelectedSegmentIndex,
  ] = useState(0)

  const [
    loopSelectedSegment,
    setLoopSelectedSegment,
  ] = useState(false)

  const [
    pendingSeekSeconds,
    setPendingSeekSeconds,
  ] = useState<
    number | null
  >(null)

  const isEditingTrackLoaded =
    currentTrack?.id === track.id

  const durationMS =
    isEditingTrackLoaded &&
    duration > 0
      ? Math.floor(
          duration * 1000,
        )
      : 0

  const currentTimeMS =
    isEditingTrackLoaded
      ? Math.max(
          0,
          currentTime * 1000,
        )
      : 0

  useEffect(() => {
    let cancelled = false

    async function loadLyrics() {
      setLoading(true)
      setError('')
      setSuccess('')

      try {
        const existing =
          await getTrackLyrics(
            track.id,
          )

        if (cancelled) {
          return
        }

        setPlainLyrics(
          existing.plain_lyrics,
        )

        setLines(
          existing.synced_lines.map(
            (line) =>
              createLine(
                millisecondsToTimestamp(
                  line.time_ms,
                ),
                line.text,
                line.end_ms
                  ? millisecondsToTimestamp(
                      line.end_ms,
                    )
                  : '',
              ),
          ),
        )

        setHasExistingLyrics(
          true,
        )
      } catch (loadError) {
        if (cancelled) {
          return
        }

        if (
          loadError instanceof
            APIError &&
          loadError.status === 404
        ) {
          setPlainLyrics('')
          setLines([])
          setHasExistingLyrics(
            false,
          )

          return
        }

        setError(
          loadError instanceof Error
            ? loadError.message
            : 'Failed to load lyrics.',
        )
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    }

    void loadLyrics()

    return () => {
      cancelled = true
    }
  }, [track.id])

  useEffect(() => {
    setSegmentBoundaries([])
    setSelectedSegmentIndex(0)
    setLoopSelectedSegment(
      false,
    )
    setPendingSeekSeconds(
      null,
    )
    setSelectedLineIDs(
      new Set(),
    )
  }, [track.id])

  /*
   * Editor preview must NEVER generate listening
   * history, qualified plays, or trending/analytics
   * signals, even though it drives the same global
   * audio element. Claiming the 'editor-preview'
   * purpose suspends tracking while this component is
   * mounted; releasing it on unmount restores normal
   * listener tracking. Artists are NOT permanently
   * excluded from their own music — normal playback
   * elsewhere still tracks.
   */
  useEffect(() => {
    setPlaybackPurpose(
      'editor-preview',
    )

    return () => {
      setPlaybackPurpose(
        'listener',
      )
    }
  }, [setPlaybackPurpose])

  useEffect(() => {
    if (durationMS <= 0) {
      return
    }

    setSegmentBoundaries(
      (current) => {
        const currentEnd =
          current[
            current.length - 1
          ]

        if (
          current.length >= 2 &&
          Math.abs(
            currentEnd -
              durationMS,
          ) < 250
        ) {
          return current
        }

        return createDefaultBoundaries(
          durationMS,
        )
      },
    )
  }, [durationMS])

  useEffect(() => {
    if (
      pendingSeekSeconds ===
        null ||
      !isEditingTrackLoaded ||
      duration <= 0
    ) {
      return
    }

    seek(
      pendingSeekSeconds,
    )

    setPendingSeekSeconds(
      null,
    )
  }, [
    duration,
    isEditingTrackLoaded,
    pendingSeekSeconds,
    seek,
  ])

  useEffect(() => {
    if (
      pendingFocusLineID ===
      null
    ) {
      return
    }

    const frame =
      window.requestAnimationFrame(
        () => {
          syncedSectionRef.current?.scrollIntoView(
            {
              behavior: 'smooth',
              block: 'start',
            },
          )

          window.setTimeout(
            () => {
              const input =
                lyricInputRefs.current.get(
                  pendingFocusLineID,
                )

              input?.scrollIntoView(
                {
                  behavior:
                    'smooth',
                  block: 'center',
                },
              )

              input?.focus()

              setPendingFocusLineID(
                null,
              )
            },
            250,
          )
        },
      )

    return () => {
      window.cancelAnimationFrame(
        frame,
      )
    }
  }, [
    lines,
    pendingFocusLineID,
  ])

  const segments =
    useMemo(() => {
      if (
        segmentBoundaries.length <
        2
      ) {
        return []
      }

      return segmentBoundaries
        .slice(0, -1)
        .map(
          (
            startMS,
            index,
          ) => ({
            index,
            startMS,
            endMS:
              segmentBoundaries[
                index + 1
              ],
          }),
        )
    }, [segmentBoundaries])

  const selectedSegment =
    segments[
      selectedSegmentIndex
    ] ?? null

  const currentSegmentIndex =
    useMemo(() => {
      if (
        segments.length === 0 ||
        !isEditingTrackLoaded
      ) {
        return -1
      }

      return segments.findIndex(
        (
          segment,
          index,
        ) => {
          const isLast =
            index ===
            segments.length - 1

          return (
            currentTimeMS >=
              segment.startMS &&
            (
              currentTimeMS <
                segment.endMS ||
              (
                isLast &&
                currentTimeMS <=
                  segment.endMS
              )
            )
          )
        },
      )
    }, [
      currentTimeMS,
      isEditingTrackLoaded,
      segments,
    ])

  useEffect(() => {
    if (
      !loopSelectedSegment ||
      !selectedSegment ||
      !isEditingTrackLoaded ||
      !isPlaying
    ) {
      return
    }

    const endSeconds =
      selectedSegment.endMS /
      1000

    if (
      currentTime >=
      endSeconds
    ) {
      seek(
        selectedSegment.startMS /
          1000,
      )
    }
  }, [
    currentTime,
    isEditingTrackLoaded,
    isPlaying,
    loopSelectedSegment,
    seek,
    selectedSegment,
  ])

  const hasSyncedLines =
    lines.length > 0

  const canDelete =
    hasExistingLyrics &&
    !saving &&
    !deleting

  const isBusy =
    saving || deleting

  const syncedPreview =
    useMemo(() => {
      return lines
        .map((line) => {
          const timeMS =
            timestampToMilliseconds(
              line.timestamp,
            )

          if (
            timeMS === null ||
            !line.text.trim()
          ) {
            return null
          }

          const endMS =
            line.endTimestamp
              ? timestampToMilliseconds(
                  line.endTimestamp,
                )
              : null

          return {
            time_ms: timeMS,
            ...(endMS !== null &&
            endMS > timeMS
              ? {
                  end_ms:
                    endMS,
                }
              : {}),
            text: line.text.trim(),
          }
        })
        .filter(
          (
            line,
          ): line is LyricLine =>
            line !== null,
        )
    }, [lines])

  function updateLine(
    id: number,
    field:
      | 'timestamp'
      | 'endTimestamp'
      | 'text',
    value: string,
  ) {
    setLines((current) =>
      current.map((line) =>
        line.id === id
          ? {
              ...line,
              [field]: value,
            }
          : line,
      ),
    )

    setError('')
    setSuccess('')
  }

  function setLineTime(
    id: number,
    milliseconds: number,
  ) {
    updateLine(
      id,
      'timestamp',
      millisecondsToTimestamp(
        Math.max(
          0,
          milliseconds,
        ),
      ),
    )
  }

  function adjustLineTime(
    id: number,
    direction: -1 | 1,
    step: number,
    field:
      | 'timestamp'
      | 'endTimestamp' =
      'timestamp',
  ) {
    setLines((current) =>
      current.map((line) => {
        if (line.id !== id) {
          return line
        }

        const parsed =
          timestampToMilliseconds(
            line[field],
          )

        const currentMilliseconds =
          parsed ?? 0

        const adjusted =
          Math.max(
            0,
            currentMilliseconds +
              direction * step,
          )

        return {
          ...line,
          [field]:
            millisecondsToTimestamp(
              adjusted,
            ),
        }
      }),
    )

    setError('')
    setSuccess('')
  }

  function handleAdjustmentClick(
    event: MouseEvent<HTMLButtonElement>,
    id: number,
    direction: -1 | 1,
    field:
      | 'timestamp'
      | 'endTimestamp' =
      'timestamp',
  ) {
    adjustLineTime(
      id,
      direction,
      getAdjustmentStep(
        event.shiftKey,
        event.ctrlKey,
        event.metaKey,
      ),
      field,
    )
  }

  function handleTimestampKeyDown(
    event: KeyboardEvent<HTMLInputElement>,
    id: number,
    field:
      | 'timestamp'
      | 'endTimestamp',
  ) {
    if (
      event.key !==
        'ArrowUp' &&
      event.key !==
        'ArrowDown'
    ) {
      return
    }

    event.preventDefault()

    adjustLineTime(
      id,
      event.key ===
        'ArrowUp'
        ? 1
        : -1,
      getAdjustmentStep(
        event.shiftKey,
        event.ctrlKey,
        event.metaKey,
      ),
      field,
    )
  }

  function captureCurrentTime(
    id: number,
  ) {
    if (
      !isEditingTrackLoaded
    ) {
      setError(
        'Play this track first before capturing its current playback time.',
      )

      return
    }

    setLineTime(
      id,
      currentTime * 1000,
    )

    setSuccess(
      `Captured ${millisecondsToTimestamp(
        currentTime * 1000,
      )}.`,
    )
  }

  function addLine() {
    let newLineID = 0

    setLines((current) => {
      const previous =
        current[
          current.length - 1
        ]

      let nextTimestamp =
        ''

      if (previous) {
        const previousMS =
          timestampToMilliseconds(
            previous.timestamp,
          )

        if (
          previousMS !== null
        ) {
          nextTimestamp =
            millisecondsToTimestamp(
              previousMS + 5000,
            )
        }
      }

      const newLine =
        createLine(
          nextTimestamp,
          '',
        )

      newLineID =
        newLine.id

      return [
        ...current,
        newLine,
      ]
    })

    setError('')
    setSuccess('')

    window.setTimeout(
      () => {
        setPendingFocusLineID(
          newLineID,
        )
      },
      0,
    )
  }

  function addLineAtSegment() {
    if (!selectedSegment) {
      addLine()
      return
    }

    const newLine =
      createLine(
        millisecondsToTimestamp(
          selectedSegment.startMS,
        ),
        '',
      )

    setLines((current) => {
      const next = [
        ...current,
        newLine,
      ]

      next.sort(
        (
          first,
          second,
        ) => {
          const firstMS =
            timestampToMilliseconds(
              first.timestamp,
            ) ?? 0

          const secondMS =
            timestampToMilliseconds(
              second.timestamp,
            ) ?? 0

          return (
            firstMS -
            secondMS
          )
        },
      )

      return next
    })

    setPendingFocusLineID(
      newLine.id,
    )

    setError('')
    setSuccess('')
  }

  function removeLine(
    id: number,
  ) {
    lyricInputRefs.current.delete(
      id,
    )

    setSelectedLineIDs(
      (current) => {
        const next =
          new Set(current)

        next.delete(id)

        return next
      },
    )

    setLines((current) =>
      current.filter(
        (line) =>
          line.id !== id,
      ),
    )

    setError('')
    setSuccess('')
  }

  function toggleLineSelection(
    id: number,
    checked: boolean,
  ) {
    setSelectedLineIDs(
      (current) => {
        const next =
          new Set(current)

        if (checked) {
          next.add(id)
        } else {
          next.delete(id)
        }

        return next
      },
    )
  }

  /*
   * The effective end of a line: its explicit end time
   * when set, otherwise the nearest of the next line's
   * start, the next 5-second grid boundary, or the
   * track duration. Merging a run of adjacent
   * 5-second lines therefore produces a range spanning
   * all of them (10s + 15s + 20s -> 10s..25s).
   */
  function resolveEffectiveEndMS(
    index: number,
  ): number | null {
    const line = lines[index]

    const explicitEndMS =
      timestampToMilliseconds(
        line.endTimestamp,
      )

    if (explicitEndMS !== null) {
      return explicitEndMS
    }

    const startMS =
      timestampToMilliseconds(
        line.timestamp,
      )

    if (startMS === null) {
      return null
    }

    const candidates: number[] =
      []

    const nextLine =
      lines[index + 1]

    if (nextLine) {
      const nextStartMS =
        timestampToMilliseconds(
          nextLine.timestamp,
        )

      if (
        nextStartMS !== null &&
        nextStartMS > startMS
      ) {
        candidates.push(
          nextStartMS,
        )
      }
    }

    const gridNextMS =
      segmentBoundaries.find(
        (boundary) =>
          boundary > startMS,
      )

    if (gridNextMS !== undefined) {
      candidates.push(
        gridNextMS,
      )
    }

    if (durationMS > startMS) {
      candidates.push(
        durationMS,
      )
    }

    if (candidates.length === 0) {
      return null
    }

    return Math.min(
      ...candidates,
    )
  }

  function mergeSelectedLines() {
    const indices: number[] =
      []

    lines.forEach(
      (line, index) => {
        if (
          selectedLineIDs.has(
            line.id,
          )
        ) {
          indices.push(index)
        }
      },
    )

    if (indices.length < 2) {
      setError(
        'Select at least two lines to merge.',
      )

      return
    }

    for (
      let i = 1;
      i < indices.length;
      i += 1
    ) {
      if (
        indices[i] !==
        indices[i - 1] + 1
      ) {
        setError(
          'Selected lines must be adjacent in the list to merge. Reorder the timestamps first.',
        )

        return
      }
    }

    const firstIndex =
      indices[0]

    const lastIndex =
      indices[
        indices.length - 1
      ]

    const firstStartMS =
      timestampToMilliseconds(
        lines[firstIndex]
          .timestamp,
      )

    if (firstStartMS === null) {
      setError(
        `Line ${firstIndex + 1} has an invalid start timestamp.`,
      )

      return
    }

    const endMS =
      resolveEffectiveEndMS(
        lastIndex,
      )

    const mergedText = lines
      .slice(
        firstIndex,
        lastIndex + 1,
      )
      .map((line) =>
        line.text.trim(),
      )
      .filter(Boolean)
      .join(' ')

    if (!mergedText) {
      setError(
        'The selected lines have no lyric text to merge.',
      )

      return
    }

    const mergedLine =
      createLine(
        lines[firstIndex]
          .timestamp,
        mergedText,
        endMS !== null
          ? millisecondsToTimestamp(
              endMS,
            )
          : '',
      )

    setLines((current) => {
      const next = [
        ...current,
      ]

      next.splice(
        firstIndex,
        indices.length,
        mergedLine,
      )

      return next
    })

    setSelectedLineIDs(
      new Set(),
    )

    setPendingFocusLineID(
      mergedLine.id,
    )

    setError('')

    setSuccess(
      `Merged ${indices.length} lines into ${millisecondsToTimestamp(firstStartMS)} → ${endMS !== null ? millisecondsToTimestamp(endMS) : 'an open end'}.`,
    )
  }

  function handlePlayTrack() {
    if (
      isEditingTrackLoaded
    ) {
      togglePlay()
      return
    }

    playTrack(
      track,
      [track],
    )
  }

  function seekRelative(
    seconds: number,
  ) {
    if (
      !isEditingTrackLoaded
    ) {
      return
    }

    seek(
      currentTime +
        seconds,
    )
  }

  function restartTrack() {
    setLoopSelectedSegment(
      false,
    )

    if (
      isEditingTrackLoaded
    ) {
      seek(0)

      if (!isPlaying) {
        togglePlay()
      }

      return
    }

    setPendingSeekSeconds(
      0,
    )

    playTrack(
      track,
      [track],
    )
  }

  function playSelectedRegion() {
    if (!selectedSegment) {
      return
    }

    const startSeconds =
      selectedSegment.startMS /
      1000

    if (
      isEditingTrackLoaded
    ) {
      seek(startSeconds)

      if (!isPlaying) {
        togglePlay()
      }

      return
    }

    setPendingSeekSeconds(
      startSeconds,
    )

    playTrack(
      track,
      [track],
    )
  }

  function toggleSegmentLoop() {
    if (!selectedSegment) {
      return
    }

    const nextValue =
      !loopSelectedSegment

    setLoopSelectedSegment(
      nextValue,
    )

    if (nextValue) {
      playSelectedRegion()
    }
  }

  function resetSegmentGrid() {
    if (durationMS <= 0) {
      return
    }

    setSegmentBoundaries(
      createDefaultBoundaries(
        durationMS,
      ),
    )

    setSelectedSegmentIndex(
      0,
    )

    setLoopSelectedSegment(
      false,
    )
  }

  function adjustBoundary(
    boundaryIndex: number,
    deltaMS: number,
  ) {
    setSegmentBoundaries(
      (current) => {
        if (
          boundaryIndex <= 0 ||
          boundaryIndex >=
            current.length - 1
        ) {
          return current
        }

        const previous =
          current[
            boundaryIndex - 1
          ]

        const value =
          current[
            boundaryIndex
          ]

        const next =
          current[
            boundaryIndex + 1
          ]

        const minimum =
          previous +
          MIN_SEGMENT_SIZE_MS

        const maximum =
          next -
          MIN_SEGMENT_SIZE_MS

        const adjusted =
          Math.max(
            minimum,
            Math.min(
              value + deltaMS,
              maximum,
            ),
          )

        const result =
          [...current]

        result[
          boundaryIndex
        ] = adjusted

        return result
      },
    )

    setLoopSelectedSegment(
      false,
    )
  }

  function handleBoundaryClick(
    event: MouseEvent<HTMLButtonElement>,
    boundaryIndex: number,
    direction: -1 | 1,
  ) {
    const step =
      getAdjustmentStep(
        event.shiftKey,
        event.ctrlKey,
        event.metaKey,
      )

    adjustBoundary(
      boundaryIndex,
      step * direction,
    )
  }

  function validateSyncedLines():
    | LyricLine[]
    | null {
    if (!hasSyncedLines) {
      return []
    }

    const parsed: Array<{
      startMS: number
      endMS: number | null
      text: string
    }> = []

    for (
      let index = 0;
      index < lines.length;
      index += 1
    ) {
      const line =
        lines[index]

      const startMS =
        timestampToMilliseconds(
          line.timestamp,
        )

      if (startMS === null) {
        setError(
          `Line ${index + 1} has an invalid timestamp. Use M:SS or M:SS.mmm.`,
        )

        return null
      }

      const text =
        line.text.trim()

      if (!text) {
        setError(
          `Line ${index + 1} needs lyric text.`,
        )

        return null
      }

      let endMS: number | null =
        null

      if (
        line.endTimestamp.trim() !==
        ''
      ) {
        endMS =
          timestampToMilliseconds(
            line.endTimestamp,
          )

        if (endMS === null) {
          setError(
            `Line ${index + 1} has an invalid end timestamp. Use M:SS or M:SS.mmm.`,
          )

          return null
        }

        if (endMS <= startMS) {
          setError(
            `Line ${index + 1} must end later than it starts.`,
          )

          return null
        }
      }

      if (
        index > 0 &&
        startMS <=
          parsed[index - 1]
            .startMS
      ) {
        setError(
          `Line ${index + 1} must have a timestamp later than line ${index}.`,
        )

        return null
      }

      parsed.push({
        startMS,
        endMS,
        text,
      })
    }

    // Explicit ranges must not extend into the next
    // line's timeframe.
    for (
      let index = 0;
      index < parsed.length;
      index += 1
    ) {
      const current =
        parsed[index]

      const next =
        parsed[index + 1]

      if (
        current.endMS !== null &&
        next &&
        current.endMS >
          next.startMS
      ) {
        setError(
          `Line ${index + 1} ends after line ${index + 2} starts. Ranges must not overlap.`,
        )

        return null
      }
    }

    return parsed.map(
      (entry) => ({
        time_ms: entry.startMS,
        ...(entry.endMS !==
        null
          ? {
              end_ms:
                entry.endMS,
            }
          : {}),
        text: entry.text,
      }),
    )
  }

  async function handleSave() {
    if (isBusy) {
      return
    }

    setError('')
    setSuccess('')

    const cleanPlainLyrics =
      plainLyrics.trim()

    const syncedLines =
      validateSyncedLines()

    if (
      syncedLines === null
    ) {
      return
    }

    if (
      !cleanPlainLyrics &&
      syncedLines.length === 0
    ) {
      setError(
        'Add plain lyrics, synchronized lyrics, or both before saving.',
      )

      return
    }

    try {
      setSaving(true)

      const saved =
        await saveTrackLyrics(
          track.id,
          {
            plain_lyrics:
              cleanPlainLyrics,

            synced_lines:
              syncedLines,
          },
          token,
        )

      applySavedLyrics(
        saved,
      )

      setHasExistingLyrics(
        true,
      )

      setSuccess(
        'Lyrics saved successfully.',
      )
    } catch (saveError) {
      setError(
        saveError instanceof Error
          ? saveError.message
          : 'Failed to save lyrics.',
      )
    } finally {
      setSaving(false)
    }
  }

  function applySavedLyrics(
    saved: TrackLyrics,
  ) {
    setPlainLyrics(
      saved.plain_lyrics,
    )

    setLines(
      saved.synced_lines.map(
        (line) =>
          createLine(
            millisecondsToTimestamp(
              line.time_ms,
            ),
            line.text,
            line.end_ms
              ? millisecondsToTimestamp(
                  line.end_ms,
                )
              : '',
          ),
      ),
    )

    setSelectedLineIDs(
      new Set(),
    )
  }

  async function handleDelete() {
    if (!canDelete) {
      return
    }

    const confirmed =
      window.confirm(
        `Delete all lyrics for "${track.song_title}"? This cannot be undone.`,
      )

    if (!confirmed) {
      return
    }

    setError('')
    setSuccess('')

    try {
      setDeleting(true)

      await deleteTrackLyrics(
        track.id,
        token,
      )

      setPlainLyrics('')
      setLines([])

      setHasExistingLyrics(
        false,
      )

      setSuccess(
        'Lyrics deleted successfully.',
      )
    } catch (deleteError) {
      setError(
        deleteError instanceof Error
          ? deleteError.message
          : 'Failed to delete lyrics.',
      )
    } finally {
      setDeleting(false)
    }
  }

  return (
    <div
      className="lyrics-editor-backdrop"
      role="presentation"
      onMouseDown={(event) => {
        if (
          event.target ===
            event.currentTarget &&
          !isBusy
        ) {
          onClose()
        }
      }}
    >
      <section
        className="lyrics-editor lyrics-workstation"
        role="dialog"
        aria-modal="true"
        aria-labelledby="lyrics-editor-title"
      >
        <header className="lyrics-editor-header">
          <div>
            <p className="eyebrow">
              LYRICS WORKSTATION
            </p>

            <h2 id="lyrics-editor-title">
              {track.song_title}
            </h2>

            <p>
              {track.artist_name}
            </p>
          </div>

          <button
            type="button"
            className="lyrics-editor-close"
            aria-label="Close lyrics editor"
            disabled={isBusy}
            onClick={onClose}
          >
            ×
          </button>
        </header>

        {loading ? (
          <div className="lyrics-editor-loading">
            Loading lyrics...
          </div>
        ) : (
          <>
            {error && (
              <div
                className="lyrics-editor-message lyrics-editor-error"
                role="alert"
              >
                {error}
              </div>
            )}

            {success && (
              <div
                className="lyrics-editor-message lyrics-editor-success"
                role="status"
              >
                {success}
              </div>
            )}

            <section className="lyrics-workstation-player">
              <div className="lyrics-workstation-player-top">
                <div>
                  <strong>
                    Synchronization player
                  </strong>

                  <p>
                    Rewind and replay small
                    parts until you identify
                    the exact point where a
                    lyric begins.
                  </p>
                </div>

                <div className="lyrics-workstation-time">
                  <strong>
                    {millisecondsToTimestamp(
                      currentTimeMS,
                    )}
                  </strong>

                  <span>/</span>

                  <span>
                    {durationMS > 0
                      ? millisecondsToTimestamp(
                          durationMS,
                        )
                      : '--:--.---'}
                  </span>
                </div>
              </div>

              <div className="lyrics-transport-controls">
                <button
                  type="button"
                  onClick={
                    restartTrack
                  }
                  disabled={
                    !track.audio_url
                  }
                >
                  ↺ Restart
                </button>

                <button
                  type="button"
                  onClick={() =>
                    seekRelative(-10)
                  }
                  disabled={
                    !isEditingTrackLoaded
                  }
                >
                  −10s
                </button>

                <button
                  type="button"
                  onClick={() =>
                    seekRelative(-5)
                  }
                  disabled={
                    !isEditingTrackLoaded
                  }
                >
                  −5s
                </button>

                <button
                  type="button"
                  onClick={() =>
                    seekRelative(-1)
                  }
                  disabled={
                    !isEditingTrackLoaded
                  }
                >
                  −1s
                </button>

                <button
                  type="button"
                  className="lyrics-transport-play"
                  onClick={
                    handlePlayTrack
                  }
                  disabled={
                    !track.audio_url
                  }
                >
                  {isEditingTrackLoaded &&
                  isPlaying
                    ? '❚❚ Pause'
                    : '▶ Play'}
                </button>

                <button
                  type="button"
                  onClick={() =>
                    seekRelative(1)
                  }
                  disabled={
                    !isEditingTrackLoaded
                  }
                >
                  +1s
                </button>

                <button
                  type="button"
                  onClick={() =>
                    seekRelative(5)
                  }
                  disabled={
                    !isEditingTrackLoaded
                  }
                >
                  +5s
                </button>

                <button
                  type="button"
                  onClick={() =>
                    seekRelative(10)
                  }
                  disabled={
                    !isEditingTrackLoaded
                  }
                >
                  +10s
                </button>
              </div>

              {isEditingTrackLoaded &&
                duration > 0 && (
                  <input
                    className="lyrics-workstation-progress"
                    type="range"
                    min="0"
                    max={duration}
                    step="0.01"
                    value={Math.min(
                      currentTime,
                      duration,
                    )}
                    aria-label="Track playback position"
                    onChange={(
                      event,
                    ) =>
                      seek(
                        Number(
                          event.target
                            .value,
                        ),
                      )
                    }
                  />
                )}
            </section>

            <section className="lyrics-segment-workspace">
              <div className="lyrics-segment-heading">
                <div>
                  <h3>
                    5-second timeline
                  </h3>

                  <p>
                    Select the part of the
                    song you are working on.
                    You can loop it, adjust
                    its boundaries and add a
                    lyric directly from that
                    section.
                  </p>
                </div>

                <button
                  type="button"
                  className="lyrics-segment-reset"
                  onClick={
                    resetSegmentGrid
                  }
                  disabled={
                    durationMS <= 0
                  }
                >
                  Reset 5s grid
                </button>
              </div>

              {durationMS <= 0 ? (
                <div className="lyrics-segment-empty">
                  <strong>
                    Play the track to load
                    its duration
                  </strong>

                  <p>
                    The 5-second timeline
                    will be generated
                    automatically.
                  </p>

                  <button
                    type="button"
                    onClick={
                      handlePlayTrack
                    }
                    disabled={
                      !track.audio_url
                    }
                  >
                    ▶ Play track
                  </button>
                </div>
              ) : (
                <>
                  <div className="lyrics-segment-strip">
                    {segments.map(
                      (segment) => {
                        const selected =
                          segment.index ===
                          selectedSegmentIndex

                        const active =
                          segment.index ===
                          currentSegmentIndex

                        return (
                          <button
                            type="button"
                            key={
                              segment.index
                            }
                            className={[
                              'lyrics-segment-card',
                              selected
                                ? 'is-selected'
                                : '',
                              active
                                ? 'is-current'
                                : '',
                            ]
                              .filter(
                                Boolean,
                              )
                              .join(' ')}
                            onClick={() => {
                              setSelectedSegmentIndex(
                                segment.index,
                              )

                              setLoopSelectedSegment(
                                false,
                              )
                            }}
                          >
                            <span>
                              Segment{' '}
                              {segment.index +
                                1}
                            </span>

                            <strong>
                              {millisecondsToTimestamp(
                                segment.startMS,
                              )}
                            </strong>

                            <small>
                              to{' '}
                              {millisecondsToTimestamp(
                                segment.endMS,
                              )}
                            </small>
                          </button>
                        )
                      },
                    )}
                  </div>

                  {selectedSegment && (
                    <div className="lyrics-selected-segment">
                      <div className="lyrics-selected-segment-main">
                        <div>
                          <p className="eyebrow">
                            SELECTED REGION
                          </p>

                          <h4>
                            Segment{' '}
                            {selectedSegment.index +
                              1}
                          </h4>
                        </div>

                        <div className="lyrics-selected-range">
                          <strong>
                            {millisecondsToTimestamp(
                              selectedSegment.startMS,
                            )}
                          </strong>

                          <span>
                            →
                          </span>

                          <strong>
                            {millisecondsToTimestamp(
                              selectedSegment.endMS,
                            )}
                          </strong>
                        </div>
                      </div>

                      <div className="lyrics-segment-playback-actions">
                        <button
                          type="button"
                          onClick={
                            playSelectedRegion
                          }
                        >
                          ▶ Play segment
                        </button>

                        <button
                          type="button"
                          className={
                            loopSelectedSegment
                              ? 'is-active'
                              : ''
                          }
                          onClick={
                            toggleSegmentLoop
                          }
                        >
                          ↻{' '}
                          {loopSelectedSegment
                            ? 'Stop loop'
                            : 'Loop segment'}
                        </button>

                        <button
                          type="button"
                          className="lyrics-add-at-segment-button"
                          onClick={
                            addLineAtSegment
                          }
                        >
                          + Add lyric here
                        </button>
                      </div>

                      <div className="lyrics-boundary-editor">
                        <div className="lyrics-boundary-control">
                          <span>
                            Start boundary
                          </span>

                          <div>
                            <button
                              type="button"
                              disabled={
                                selectedSegment.index ===
                                0
                              }
                              onClick={(
                                event,
                              ) =>
                                handleBoundaryClick(
                                  event,
                                  selectedSegment.index,
                                  -1,
                                )
                              }
                            >
                              −
                            </button>

                            <strong>
                              {millisecondsToTimestamp(
                                selectedSegment.startMS,
                              )}
                            </strong>

                            <button
                              type="button"
                              disabled={
                                selectedSegment.index ===
                                0
                              }
                              onClick={(
                                event,
                              ) =>
                                handleBoundaryClick(
                                  event,
                                  selectedSegment.index,
                                  1,
                                )
                              }
                            >
                              +
                            </button>
                          </div>

                          <small>
                            Adjusts this
                            segment and the
                            segment before it.
                          </small>
                        </div>

                        <div className="lyrics-boundary-control">
                          <span>
                            End boundary
                          </span>

                          <div>
                            <button
                              type="button"
                              disabled={
                                selectedSegment.index ===
                                segments.length -
                                  1
                              }
                              onClick={(
                                event,
                              ) =>
                                handleBoundaryClick(
                                  event,
                                  selectedSegment.index +
                                    1,
                                  -1,
                                )
                              }
                            >
                              −
                            </button>

                            <strong>
                              {millisecondsToTimestamp(
                                selectedSegment.endMS,
                              )}
                            </strong>

                            <button
                              type="button"
                              disabled={
                                selectedSegment.index ===
                                segments.length -
                                  1
                              }
                              onClick={(
                                event,
                              ) =>
                                handleBoundaryClick(
                                  event,
                                  selectedSegment.index +
                                    1,
                                  1,
                                )
                              }
                            >
                              +
                            </button>
                          </div>

                          <small>
                            Adjusts this
                            segment and the
                            segment after it.
                          </small>
                        </div>
                      </div>

                      <p className="lyrics-boundary-help">
                        Normal adjustment:
                        100 ms. Shift:
                        500 ms. Ctrl:
                        1 second.
                      </p>
                    </div>
                  )}
                </>
              )}
            </section>

            <div className="lyrics-editor-content lyrics-workstation-content">
              <section className="lyrics-editor-section">
                <div className="lyrics-editor-section-heading">
                  <div>
                    <h3>
                      Plain lyrics
                    </h3>

                    <p>
                      The complete lyrics
                      without timing.
                    </p>
                  </div>
                </div>

                <textarea
                  className="lyrics-editor-textarea"
                  value={plainLyrics}
                  disabled={isBusy}
                  placeholder={`Morning light is calling
I can hear the city waking
Another road is waiting`}
                  onChange={(
                    event,
                  ) => {
                    setPlainLyrics(
                      event.target
                        .value,
                    )

                    setError('')
                    setSuccess('')
                  }}
                />

                <div className="lyrics-editor-character-count">
                  {
                    plainLyrics.length
                  }{' '}
                  / 100,000 characters
                </div>
              </section>

              <section
                ref={
                  syncedSectionRef
                }
                className="lyrics-editor-section lyrics-synced-editor-target"
              >
                <div className="lyrics-editor-section-heading lyrics-editor-synced-heading">
                  <div>
                    <h3>
                      Synchronized lyrics
                    </h3>

                    <p>
                      Each line has its
                      own exact
                      timestamp. Select
                      adjacent lines and
                      merge them into
                      one timed range.
                    </p>
                  </div>

                  <div className="lyrics-heading-actions">
                    <button
                      type="button"
                      className="lyrics-editor-merge-button"
                      disabled={
                        isBusy ||
                        selectedLineIDs
                          .size < 2
                      }
                      onClick={
                        mergeSelectedLines
                      }
                    >
                      ⇥ Merge
                      selected (
                      {selectedLineIDs.size}
                      )
                    </button>

                    <button
                      type="button"
                      className="lyrics-editor-add-button"
                      disabled={isBusy}
                      onClick={addLine}
                    >
                      + Add line
                    </button>
                  </div>
                </div>

                <div className="lyrics-sync-help">
                  <span>
                    ± = 100 ms
                  </span>

                  <span>
                    Shift = 500 ms
                  </span>

                  <span>
                    Ctrl = 1 second
                  </span>

                  <span>
                    ↑ / ↓ adjust timestamp
                  </span>
                </div>

                {lines.length ===
                0 ? (
                  <div className="lyrics-editor-empty lyrics-editor-empty-visible">
                    <strong>
                      No synchronized lyrics
                      yet
                    </strong>

                    <p>
                      Select a 5-second
                      section above and press
                      "Add lyric here", or
                      add a line manually.
                    </p>

                    <button
                      type="button"
                      className="lyrics-editor-add-button"
                      disabled={isBusy}
                      onClick={
                        addLineAtSegment
                      }
                    >
                      + Add first lyric
                    </button>
                  </div>
                ) : (
                  <div className="lyrics-editor-lines lyrics-editor-lines-visible">
                    {lines.map(
                      (
                        line,
                        index,
                      ) => (
                        <div
                          className="lyrics-editor-line lyrics-editor-line-expanded lyrics-line-workstation-row"
                          key={line.id}
                        >
                          <label
                            className="lyrics-line-select"
                            title={`Select line ${index + 1} for merging`}
                          >
                            <input
                              type="checkbox"
                              checked={selectedLineIDs.has(
                                line.id,
                              )}
                              disabled={isBusy}
                              onChange={(
                                event,
                              ) =>
                                toggleLineSelection(
                                  line.id,
                                  event
                                    .target
                                    .checked,
                                )
                              }
                            />

                            <span className="lyrics-editor-line-number">
                              {index + 1}
                            </span>
                          </label>

                          <div className="lyrics-time-column">
                            <div className="lyrics-time-controls">
                              <button
                                type="button"
                                className="lyrics-time-adjust"
                                title="Move earlier"
                                disabled={isBusy}
                                onClick={(
                                  event,
                                ) =>
                                  handleAdjustmentClick(
                                    event,
                                    line.id,
                                    -1,
                                    'timestamp',
                                  )
                                }
                              >
                                −
                              </button>

                              <input
                                type="text"
                                className="lyrics-editor-time"
                                value={
                                  line.timestamp
                                }
                                disabled={isBusy}
                                placeholder="0:00.000"
                                onKeyDown={(
                                  event,
                                ) =>
                                  handleTimestampKeyDown(
                                    event,
                                    line.id,
                                    'timestamp',
                                  )
                                }
                                onChange={(
                                  event,
                                ) =>
                                  updateLine(
                                    line.id,
                                    'timestamp',
                                    event
                                      .target
                                      .value,
                                  )
                                }
                              />

                              <button
                                type="button"
                                className="lyrics-time-adjust"
                                title="Move later"
                                disabled={isBusy}
                                onClick={(
                                  event,
                                ) =>
                                  handleAdjustmentClick(
                                    event,
                                    line.id,
                                    1,
                                    'timestamp',
                                  )
                                }
                              >
                                +
                              </button>
                            </div>

                            {line
                              .endTimestamp !==
                              '' && (
                              <div className="lyrics-time-controls lyrics-time-controls-end">
                                <button
                                  type="button"
                                  className="lyrics-time-adjust"
                                  title="End earlier"
                                  disabled={isBusy}
                                  onClick={(
                                    event,
                                  ) =>
                                    handleAdjustmentClick(
                                      event,
                                      line.id,
                                      -1,
                                      'endTimestamp',
                                    )
                                  }
                                >
                                  −
                                </button>

                                <input
                                  type="text"
                                  className="lyrics-editor-time lyrics-editor-time-end"
                                  value={
                                    line.endTimestamp
                                  }
                                  disabled={isBusy}
                                  placeholder="end time"
                                  onKeyDown={(
                                    event,
                                  ) =>
                                    handleTimestampKeyDown(
                                      event,
                                      line.id,
                                      'endTimestamp',
                                    )
                                  }
                                  onChange={(
                                    event,
                                  ) =>
                                    updateLine(
                                      line.id,
                                      'endTimestamp',
                                      event
                                        .target
                                        .value,
                                    )
                                  }
                                />

                                <button
                                  type="button"
                                  className="lyrics-time-adjust"
                                  title="End later"
                                  disabled={isBusy}
                                  onClick={(
                                    event,
                                  ) =>
                                    handleAdjustmentClick(
                                      event,
                                      line.id,
                                      1,
                                      'endTimestamp',
                                    )
                                  }
                                >
                                  +
                                </button>
                              </div>
                            )}
                          </div>

                          <input
                            ref={(
                              element,
                            ) => {
                              if (
                                element
                              ) {
                                lyricInputRefs.current.set(
                                  line.id,
                                  element,
                                )
                              } else {
                                lyricInputRefs.current.delete(
                                  line.id,
                                )
                              }
                            }}
                            type="text"
                            className="lyrics-editor-line-text lyrics-main-line-input"
                            value={
                              line.text
                            }
                            disabled={isBusy}
                            placeholder="Type the lyric for this moment..."
                            onChange={(
                              event,
                            ) =>
                              updateLine(
                                line.id,
                                'text',
                                event
                                  .target
                                  .value,
                              )
                            }
                          />

                          <button
                            type="button"
                            className="lyrics-capture-time-button"
                            disabled={
                              isBusy ||
                              !isEditingTrackLoaded
                            }
                            onClick={() =>
                              captureCurrentTime(
                                line.id,
                              )
                            }
                          >
                            Use current time
                          </button>

                          <button
                            type="button"
                            className="lyrics-editor-remove-line"
                            aria-label={`Remove lyric line ${index + 1}`}
                            disabled={isBusy}
                            onClick={() =>
                              removeLine(
                                line.id,
                              )
                            }
                          >
                            ×
                          </button>
                        </div>
                      ),
                    )}
                  </div>
                )}

                {syncedPreview.length >
                  0 && (
                  <p className="lyrics-editor-summary">
                    {
                      syncedPreview.length
                    }{' '}
                    synchronized{' '}
                    {syncedPreview.length ===
                    1
                      ? 'line'
                      : 'lines'}{' '}
                    ready.
                  </p>
                )}
              </section>
            </div>

            <footer className="lyrics-editor-actions lyrics-workstation-footer">
              <div>
                {hasExistingLyrics && (
                  <button
                    type="button"
                    className="lyrics-editor-delete-button"
                    disabled={!canDelete}
                    onClick={() =>
                      void handleDelete()
                    }
                  >
                    {deleting
                      ? 'Deleting...'
                      : 'Delete lyrics'}
                  </button>
                )}
              </div>

              <div className="lyrics-editor-primary-actions">
                <button
                  type="button"
                  className="lyrics-editor-cancel-button"
                  disabled={isBusy}
                  onClick={onClose}
                >
                  Close
                </button>

                <button
                  type="button"
                  className="release-primary-button"
                  disabled={isBusy}
                  onClick={() =>
                    void handleSave()
                  }
                >
                  {saving
                    ? 'Saving...'
                    : hasExistingLyrics
                      ? 'Update lyrics'
                      : 'Save lyrics'}
                </button>
              </div>
            </footer>
          </>
        )}
      </section>
    </div>
  )
}

export default LyricsEditor