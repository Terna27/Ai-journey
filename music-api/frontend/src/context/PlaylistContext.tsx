import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from 'react'

import { useAuth } from './AuthContext'

import {
  addTrackToPlaylist,
  createPlaylist,
  deletePlaylist,
  getMyPlaylists,
  removeTrackFromPlaylist,
  updatePlaylist,
} from '../lib/api'

import type {
  CreatePlaylistInput,
  Playlist,
  UpdatePlaylistInput,
} from '../types/playlist'

type PlaylistContextValue = {
  playlists: Playlist[]
  isLoadingPlaylists: boolean
  playlistsError: string

  refreshPlaylists: () => Promise<void>
  clearPlaylists: () => void

  createPlaylist: (
    payload: CreatePlaylistInput,
  ) => Promise<Playlist>
  updatePlaylist: (
    playlistID: number,
    payload: UpdatePlaylistInput,
  ) => Promise<Playlist>
  deletePlaylist: (
    playlistID: number,
  ) => Promise<void>
  addTrack: (
    playlistID: number,
    musicID: number,
  ) => Promise<void>
  removeTrack: (
    playlistID: number,
    musicID: number,
  ) => Promise<void>
}

const PlaylistContext =
  createContext<
    PlaylistContextValue | undefined
  >(undefined)

export function PlaylistProvider({
  children,
}: {
  children: ReactNode
}) {
  const {
    token,
    user,
    isAuthenticated,
  } = useAuth()

  const [playlists, setPlaylists] =
    useState<Playlist[]>([])

  const [
    isLoadingPlaylists,
    setIsLoadingPlaylists,
  ] = useState(false)

  const [
    playlistsError,
    setPlaylistsError,
  ] = useState('')

  const clearPlaylists =
    useCallback(() => {
      setPlaylists([])
      setPlaylistsError('')
    }, [])

  const refreshPlaylists =
    useCallback(async () => {
      if (
        !token ||
        !user ||
        !isAuthenticated
      ) {
        clearPlaylists()
        return
      }

      setIsLoadingPlaylists(true)
      setPlaylistsError('')

      try {
        const result =
          await getMyPlaylists(token)

        setPlaylists(result)
      } catch (error) {
        setPlaylistsError(
          error instanceof Error
            ? error.message
            : 'Failed to load playlists',
        )
      } finally {
        setIsLoadingPlaylists(false)
      }
    }, [
      clearPlaylists,
      isAuthenticated,
      token,
      user,
    ])

  const createPlaylistForUser =
    useCallback(
      async (
        payload: CreatePlaylistInput,
      ) => {
        if (!token || !user) {
          throw new Error(
            'You must be logged in to create playlists.',
          )
        }

        setPlaylistsError('')

        try {
          const created =
            await createPlaylist(
              payload,
              token,
            )

          setPlaylists((current) => [
            created,
            ...current,
          ])

          return created
        } catch (error) {
          const message =
            error instanceof Error
              ? error.message
              : 'Failed to create playlist'

          setPlaylistsError(message)
          throw error
        }
      },
      [token, user],
    )

  const updatePlaylistForUser =
    useCallback(
      async (
        playlistID: number,
        payload: UpdatePlaylistInput,
      ) => {
        if (!token || !user) {
          throw new Error(
            'You must be logged in to update playlists.',
          )
        }

        setPlaylistsError('')

        try {
          const updated =
            await updatePlaylist(
              playlistID,
              payload,
              token,
            )

          setPlaylists((current) =>
            current.map((playlist) =>
              playlist.id === updated.id
                ? updated
                : playlist,
            ),
          )

          return updated
        } catch (error) {
          const message =
            error instanceof Error
              ? error.message
              : 'Failed to update playlist'

          setPlaylistsError(message)
          throw error
        }
      },
      [token, user],
    )

  const deletePlaylistForUser =
    useCallback(
      async (playlistID: number) => {
        if (!token || !user) {
          throw new Error(
            'You must be logged in to delete playlists.',
          )
        }

        setPlaylistsError('')

        try {
          await deletePlaylist(
            playlistID,
            token,
          )

          setPlaylists((current) =>
            current.filter(
              (playlist) =>
                playlist.id !== playlistID,
            ),
          )
        } catch (error) {
          const message =
            error instanceof Error
              ? error.message
              : 'Failed to delete playlist'

          setPlaylistsError(message)
          throw error
        }
      },
      [token, user],
    )

  const addTrackToPlaylistForUser =
    useCallback(
      async (
        playlistID: number,
        musicID: number,
      ) => {
        if (!token || !user) {
          throw new Error(
            'You must be logged in to add songs to playlists.',
          )
        }

        await addTrackToPlaylist(
          playlistID,
          musicID,
          token,
        )
      },
      [token, user],
    )

  const removeTrackFromPlaylistForUser =
    useCallback(
      async (
        playlistID: number,
        musicID: number,
      ) => {
        if (!token || !user) {
          throw new Error(
            'You must be logged in to remove songs from playlists.',
          )
        }

        await removeTrackFromPlaylist(
          playlistID,
          musicID,
          token,
        )
      },
      [token, user],
    )

  /*
   * Load the user's playlists whenever an authenticated
   * session becomes available, and clear them on logout.
   * Mirrors LibraryContext's lifecycle.
   */
  useEffect(() => {
    if (
      !token ||
      !user ||
      !isAuthenticated
    ) {
      clearPlaylists()
      return
    }

    void refreshPlaylists()
  }, [
    clearPlaylists,
    isAuthenticated,
    refreshPlaylists,
    token,
    user,
  ])

  return (
    <PlaylistContext.Provider
      value={{
        playlists,
        isLoadingPlaylists,
        playlistsError,
        refreshPlaylists,
        clearPlaylists,
        createPlaylist: createPlaylistForUser,
        updatePlaylist: updatePlaylistForUser,
        deletePlaylist: deletePlaylistForUser,
        addTrack: addTrackToPlaylistForUser,
        removeTrack:
          removeTrackFromPlaylistForUser,
      }}
    >
      {children}
    </PlaylistContext.Provider>
  )
}

export function usePlaylists() {
  const context =
    useContext(PlaylistContext)

  if (!context) {
    throw new Error(
      'usePlaylists must be used inside PlaylistProvider',
    )
  }

  return context
}
