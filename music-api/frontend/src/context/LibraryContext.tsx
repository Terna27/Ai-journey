import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'

import { useAuth } from './AuthContext'

import {
  getLikedMusic,
  likeMusic,
  unlikeMusic,
} from '../lib/api'

import type { Music } from '../types/music'

type LibraryContextValue = {
  likedMusic: Music[]
  likedMusicIDs: Set<number>
  isLoadingLibrary: boolean
  libraryError: string

  isLiked: (musicID: number) => boolean

  toggleLike: (
    track: Music,
  ) => Promise<void>

  refreshLibrary: () => Promise<void>
  clearLibrary: () => void
}

const LibraryContext =
  createContext<
    LibraryContextValue | undefined
  >(undefined)

export function LibraryProvider({
  children,
}: {
  children: ReactNode
}) {
  const {
    token,
    user,
    isAuthenticated,
  } = useAuth()

  const [likedMusic, setLikedMusic] =
    useState<Music[]>([])

  const [
    isLoadingLibrary,
    setIsLoadingLibrary,
  ] = useState(false)

  const [
    libraryError,
    setLibraryError,
  ] = useState('')

  const likedMusicIDs =
    useMemo(() => {
      return new Set(
        likedMusic.map(
          (track) => track.id,
        ),
      )
    }, [likedMusic])

  const clearLibrary =
    useCallback(() => {
      setLikedMusic([])
      setLibraryError('')
    }, [])

  const refreshLibrary =
    useCallback(async () => {
      if (
        !token ||
        !user ||
        !isAuthenticated
      ) {
        clearLibrary()
        return
      }

      setIsLoadingLibrary(true)
      setLibraryError('')

      try {
        const result =
          await getLikedMusic(token)

        setLikedMusic(result)
      } catch (error) {
        setLibraryError(
          error instanceof Error
            ? error.message
            : 'Failed to load liked music',
        )
      } finally {
        setIsLoadingLibrary(false)
      }
    }, [
      clearLibrary,
      isAuthenticated,
      token,
      user,
    ])

  const isLiked =
    useCallback(
      (musicID: number) => {
        return likedMusicIDs.has(
          musicID,
        )
      },
      [likedMusicIDs],
    )

  const toggleLike =
    useCallback(
      async (track: Music) => {
        if (!token || !user) {
          throw new Error(
            'You must be logged in to like music.',
          )
        }

        setLibraryError('')

        const alreadyLiked =
          likedMusicIDs.has(track.id)

        try {
          if (alreadyLiked) {
            await unlikeMusic(
              track.id,
              token,
            )

            setLikedMusic(
              (current) =>
                current.filter(
                  (item) =>
                    item.id !== track.id,
                ),
            )

            return
          }

          const updatedTrack =
            await likeMusic(
              track.id,
              token,
            )

          setLikedMusic(
            (current) => [
              updatedTrack,
              ...current.filter(
                (item) =>
                  item.id !==
                  updatedTrack.id,
              ),
            ],
          )
        } catch (error) {
          const message =
            error instanceof Error
              ? error.message
              : 'Failed to update liked music'

          setLibraryError(message)

          throw error
        }
      },
      [
        likedMusicIDs,
        token,
        user,
      ],
    )

  useEffect(() => {
    if (
      !token ||
      !user ||
      !isAuthenticated
    ) {
      clearLibrary()
      return
    }

    void refreshLibrary()
  }, [
    clearLibrary,
    isAuthenticated,
    refreshLibrary,
    token,
    user,
  ])

  return (
    <LibraryContext.Provider
      value={{
        likedMusic,
        likedMusicIDs,
        isLoadingLibrary,
        libraryError,
        isLiked,
        toggleLike,
        refreshLibrary,
        clearLibrary,
      }}
    >
      {children}
    </LibraryContext.Provider>
  )
}

export function useLibrary() {
  const context =
    useContext(LibraryContext)

  if (!context) {
    throw new Error(
      'useLibrary must be used inside LibraryProvider',
    )
  }

  return context
}