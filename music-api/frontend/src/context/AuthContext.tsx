import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from 'react'

import {
  createArtistProfile,
  getMe,
} from '../lib/api'

import type {
  Artist,
  User,
} from '../types/auth'

type AuthContextValue = {
  user: User | null
  artist: Artist | null
  token: string | null

  isAuthenticated: boolean
  isArtist: boolean
  isLoadingIdentity: boolean

  login: (token: string) => Promise<void>
  logout: () => void
  refreshIdentity: () => Promise<void>
  becomeArtist: () => Promise<Artist>
}

const AuthContext =
  createContext<AuthContextValue | undefined>(
    undefined,
  )

const TOKEN_KEY = 'music_access_token'

export function AuthProvider({
  children,
}: {
  children: ReactNode
}) {
  const [token, setToken] =
    useState<string | null>(() =>
      sessionStorage.getItem(TOKEN_KEY),
    )

  const [user, setUser] =
    useState<User | null>(null)

  const [artist, setArtist] =
    useState<Artist | null>(null)

  const [
    isLoadingIdentity,
    setIsLoadingIdentity,
  ] = useState(Boolean(token))

  const clearAuth = useCallback(() => {
    setToken(null)
    setUser(null)
    setArtist(null)

    sessionStorage.removeItem(TOKEN_KEY)
  }, [])

  const loadIdentity = useCallback(
    async (accessToken: string) => {
      const identity =
        await getMe(accessToken)

      setUser(identity.user)
      setArtist(identity.artist)
    },
    [],
  )

  const refreshIdentity =
    useCallback(async () => {
      if (!token) {
        setUser(null)
        setArtist(null)
        return
      }

      await loadIdentity(token)
    }, [loadIdentity, token])

  const login = useCallback(
    async (newToken: string) => {
      setIsLoadingIdentity(true)

      try {
        await loadIdentity(newToken)

        setToken(newToken)

        sessionStorage.setItem(
          TOKEN_KEY,
          newToken,
        )
      } catch (error) {
        clearAuth()
        throw error
      } finally {
        setIsLoadingIdentity(false)
      }
    },
    [clearAuth, loadIdentity],
  )

  const logout = useCallback(() => {
    clearAuth()
  }, [clearAuth])

  const becomeArtist =
    useCallback(async () => {
      if (!token) {
        throw new Error(
          'You must be logged in first.',
        )
      }

      const createdArtist =
        await createArtistProfile(token)

      await loadIdentity(token)

      return createdArtist
    }, [loadIdentity, token])

  useEffect(() => {
    if (!token) {
      setIsLoadingIdentity(false)
      return
    }

    const accessToken = token
    let cancelled = false

    async function restoreSession() {
      setIsLoadingIdentity(true)

      try {
        const identity =
          await getMe(accessToken)

        if (cancelled) {
          return
        }

        setUser(identity.user)
        setArtist(identity.artist)
      } catch {
        if (!cancelled) {
          clearAuth()
        }
      } finally {
        if (!cancelled) {
          setIsLoadingIdentity(false)
        }
      }
    }

    void restoreSession()

    return () => {
      cancelled = true
    }
  }, [clearAuth, token])

  return (
    <AuthContext.Provider
      value={{
        user,
        artist,
        token,

        isAuthenticated:
          Boolean(token && user),

        isArtist:
          Boolean(token && user && artist),

        isLoadingIdentity,

        login,
        logout,
        refreshIdentity,
        becomeArtist,
      }}
    >
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const context =
    useContext(AuthContext)

  if (!context) {
    throw new Error(
      'useAuth must be used inside AuthProvider',
    )
  }

  return context
}