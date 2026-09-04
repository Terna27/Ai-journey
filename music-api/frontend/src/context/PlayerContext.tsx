import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from 'react'

import type { Music } from '../types/music'

type PlayerContextValue = {
  currentTrack: Music | null
  isPlaying: boolean
  currentTime: number
  duration: number
  playTrack: (track: Music) => void
  togglePlay: () => void
  seek: (time: number) => void
  stopPlayback: () => void
}

const PlayerContext =
  createContext<PlayerContextValue | undefined>(undefined)

type PlayerProviderProps = {
  children: ReactNode
}

export function PlayerProvider({
  children,
}: PlayerProviderProps) {
  const audioRef = useRef<HTMLAudioElement | null>(null)

  const [currentTrack, setCurrentTrack] =
    useState<Music | null>(null)

  const [isPlaying, setIsPlaying] = useState(false)

  const [currentTime, setCurrentTime] =
    useState(0)

  const [duration, setDuration] =
    useState(0)

  /*
   * Create one audio element for the entire application.
   *
   * This is what prevents multiple songs from playing
   * at the same time.
   */
  useEffect(() => {
    const audio = new Audio()

    audio.preload = 'metadata'

    audioRef.current = audio

    const handleTimeUpdate = () => {
      setCurrentTime(audio.currentTime)
    }

    const handleLoadedMetadata = () => {
      setDuration(
        Number.isFinite(audio.duration)
          ? audio.duration
          : 0,
      )
    }

    const handleDurationChange = () => {
      setDuration(
        Number.isFinite(audio.duration)
          ? audio.duration
          : 0,
      )
    }

    const handlePlay = () => {
      setIsPlaying(true)
    }

    const handlePause = () => {
      setIsPlaying(false)
    }

    const handleEnded = () => {
      setIsPlaying(false)
      setCurrentTime(0)
    }

    const handleError = () => {
      setIsPlaying(false)
    }

    audio.addEventListener(
      'timeupdate',
      handleTimeUpdate,
    )

    audio.addEventListener(
      'loadedmetadata',
      handleLoadedMetadata,
    )

    audio.addEventListener(
      'durationchange',
      handleDurationChange,
    )

    audio.addEventListener(
      'play',
      handlePlay,
    )

    audio.addEventListener(
      'pause',
      handlePause,
    )

    audio.addEventListener(
      'ended',
      handleEnded,
    )

    audio.addEventListener(
      'error',
      handleError,
    )

    return () => {
      audio.pause()

      audio.removeEventListener(
        'timeupdate',
        handleTimeUpdate,
      )

      audio.removeEventListener(
        'loadedmetadata',
        handleLoadedMetadata,
      )

      audio.removeEventListener(
        'durationchange',
        handleDurationChange,
      )

      audio.removeEventListener(
        'play',
        handlePlay,
      )

      audio.removeEventListener(
        'pause',
        handlePause,
      )

      audio.removeEventListener(
        'ended',
        handleEnded,
      )

      audio.removeEventListener(
        'error',
        handleError,
      )

      audioRef.current = null
    }
  }, [])

  /*
   * Play a selected track.
   *
   * If another track is currently playing,
   * the same global audio element is reused,
   * so the previous track stops automatically.
   *
   * Clicking the currently selected track
   * toggles play/pause.
   */
  const playTrack = useCallback(
    (track: Music) => {
      const audio = audioRef.current

      if (!audio || !track.audio_url) {
        return
      }

      if (currentTrack?.id === track.id) {
        if (audio.paused) {
          void audio.play().catch(() => {
            setIsPlaying(false)
          })
        } else {
          audio.pause()
        }

        return
      }

      audio.pause()

      audio.src = track.audio_url
      audio.currentTime = 0

      setCurrentTrack(track)
      setCurrentTime(0)
      setDuration(0)

      void audio.play().catch(() => {
        setIsPlaying(false)
      })
    },
    [currentTrack],
  )

  /*
   * Play or pause the currently selected track.
   */
  const togglePlay = useCallback(() => {
    const audio = audioRef.current

    if (!audio || !currentTrack) {
      return
    }

    if (audio.paused) {
      void audio.play().catch(() => {
        setIsPlaying(false)
      })
    } else {
      audio.pause()
    }
  }, [currentTrack])

  /*
   * Move playback to a specific time.
   */
  const seek = useCallback(
    (time: number) => {
      const audio = audioRef.current

      if (
        !audio ||
        !Number.isFinite(time)
      ) {
        return
      }

      const maxDuration =
        Number.isFinite(audio.duration)
          ? audio.duration
          : 0

      const nextTime = Math.max(
        0,
        Math.min(time, maxDuration),
      )

      audio.currentTime = nextTime

      setCurrentTime(nextTime)
    },
    [],
  )

  /*
   * Completely stop and clear playback.
   *
   * This is different from pause.
   * Pause keeps the selected track and position.
   *
   * stopPlayback removes the current song
   * completely. We will use this on logout.
   */
  const stopPlayback = useCallback(() => {
    const audio = audioRef.current

    if (audio) {
      audio.pause()
      audio.currentTime = 0

      audio.removeAttribute('src')
      audio.load()
    }

    setCurrentTrack(null)
    setIsPlaying(false)
    setCurrentTime(0)
    setDuration(0)
  }, [])

  return (
    <PlayerContext.Provider
      value={{
        currentTrack,
        isPlaying,
        currentTime,
        duration,
        playTrack,
        togglePlay,
        seek,
        stopPlayback,
      }}
    >
      {children}
    </PlayerContext.Provider>
  )
}

export function usePlayer() {
  const context = useContext(PlayerContext)

  if (!context) {
    throw new Error(
      'usePlayer must be used inside PlayerProvider',
    )
  }

  return context
}