import type {
  APIErrorResponse,
  Artist,
  LoginRequest,
  LoginResponse,
  MeResponse,
  RegisterRequest,
  RegisterResponse,
  ResendVerificationRequest,
  ResendVerificationResponse,
  VerifyEmailRequest,
  VerifyEmailResponse,
} from '../types/auth'

import type {
  Music,
  UploadMusicInput,
} from '../types/music'

import type {
  CreatePlaylistInput,
  Playlist,
  PlaylistDetails,
  UpdatePlaylistInput,
} from '../types/playlist'

import type {
  ArtistProfileResponse,
} from '../types/artist'

const API_BASE_URL =
  import.meta.env.VITE_API_BASE_URL ??
  'http://localhost:8080/api/v1'

/*
 * APIError keeps the backend error code so callers can
 * react to specific cases (e.g. TRACK_ALREADY_IN_PLAYLIST)
 * instead of parsing human-readable messages.
 */
export class APIError extends Error {
  readonly status: number
  readonly code: string

  constructor(
    message: string,
    status: number,
    code: string,
  ) {
    super(message)
    this.name = 'APIError'
    this.status = status
    this.code = code
  }
}

async function request<T>(
  path: string,
  options: RequestInit = {},
): Promise<T> {
  const response = await fetch(
    `${API_BASE_URL}${path}`,
    {
      ...options,
      headers: {
        ...options.headers,
      },
    },
  )

  if (!response.ok) {
    let message = 'Something went wrong'
    let code = ''

    try {
      const body =
        (await response.json()) as APIErrorResponse

      if (body.error?.message) {
        message = body.error.message
      }

      if (body.error?.code) {
        code = body.error.code
      }
    } catch {
      // Response did not contain JSON.
    }

    throw new APIError(
      message,
      response.status,
      code,
    )
  }

  if (response.status === 204) {
    return undefined as T
  }

  return response.json() as Promise<T>
}

export async function registerUser(
  payload: RegisterRequest,
): Promise<RegisterResponse> {
  return request<RegisterResponse>(
    '/auth/register',
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(payload),
    },
  )
}

export async function loginUser(
  payload: LoginRequest,
): Promise<LoginResponse> {
  return request<LoginResponse>(
    '/auth/login',
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(payload),
    },
  )
}

export async function verifyEmail(
  payload: VerifyEmailRequest,
): Promise<VerifyEmailResponse> {
  return request<VerifyEmailResponse>(
    '/auth/verify-email',
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(payload),
    },
  )
}

export async function resendVerification(
  payload: ResendVerificationRequest,
): Promise<ResendVerificationResponse> {
  return request<ResendVerificationResponse>(
    '/auth/resend-verification',
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(payload),
    },
  )
}

export async function getMe(
  token: string,
): Promise<MeResponse> {
  return request<MeResponse>('/me', {
    headers: {
      Authorization: `Bearer ${token}`,
    },
  })
}

export async function createArtistProfile(
  token: string,
): Promise<Artist> {
  return request<Artist>(
    '/artists/profile',
    {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${token}`,
      },
    },
  )
}

export async function getMusic(): Promise<Music[]> {
  return request<Music[]>('/music')
}

export async function uploadMusic(
  payload: UploadMusicInput,
  token: string,
): Promise<Music> {
  const formData = new FormData()

  formData.append(
    'artist_name',
    payload.artistName,
  )

  formData.append(
    'song_title',
    payload.songTitle,
  )

  formData.append(
    'genre',
    payload.genre,
  )

  formData.append(
    'audio_key',
    `${Date.now()}-${payload.audio.name}`,
  )

  formData.append(
    'image',
    payload.image,
  )

  formData.append(
    'audio',
    payload.audio,
  )

  return request<Music>('/music', {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${token}`,
    },
    body: formData,
  })
}


export async function getLikedMusic(
  token: string,
): Promise<Music[]> {  return request<Music[]>(
    '/me/liked-music',
    {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    },
  )
}

export async function likeMusic(
  musicID: number,
  token: string,
): Promise<Music> {
  return request<Music>(
    `/music/${musicID}/like`,
    {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${token}`,
      },
    },
  )
}

export async function unlikeMusic(
  musicID: number,
  token: string,
): Promise<Music> {
  return request<Music>(
    `/music/${musicID}/like`,
    {
      method: 'DELETE',
      headers: {
        Authorization: `Bearer ${token}`,
      },
    },
  )
}

export async function createPlaylist(
  payload: CreatePlaylistInput,
  token: string,
): Promise<Playlist> {
  return request<Playlist>(
    '/playlists',
    {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(payload),
    },
  )
}

export async function getMyPlaylists(
  token: string,
): Promise<Playlist[]> {
  return request<Playlist[]>(
    '/me/playlists',
    {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    },
  )
}

export async function getPlaylist(
  playlistID: number,
  token: string,
): Promise<PlaylistDetails> {
  return request<PlaylistDetails>(
    `/playlists/${playlistID}`,
    {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    },
  )
}

export async function updatePlaylist(
  playlistID: number,
  payload: UpdatePlaylistInput,
  token: string,
): Promise<Playlist> {
  return request<Playlist>(
    `/playlists/${playlistID}`,
    {
      method: 'PUT',
      headers: {
        Authorization: `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(payload),
    },
  )
}

export async function deletePlaylist(
  playlistID: number,
  token: string,
): Promise<void> {
  await request<void>(
    `/playlists/${playlistID}`,
    {
      method: 'DELETE',
      headers: {
        Authorization: `Bearer ${token}`,
      },
    },
  )
}

export async function addTrackToPlaylist(
  playlistID: number,
  musicID: number,
  token: string,
): Promise<{ message: string }> {
  return request<{ message: string }>(
    `/playlists/${playlistID}/tracks/${musicID}`,
    {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${token}`,
      },
    },
  )
}

export async function removeTrackFromPlaylist(
  playlistID: number,
  musicID: number,
  token: string,
): Promise<void> {
  await request<void>(
    `/playlists/${playlistID}/tracks/${musicID}`,
    {
      method: 'DELETE',
      headers: {
        Authorization: `Bearer ${token}`,
      },
    },
  )
}

export async function getArtistProfile(
  artistID: number,
): Promise<ArtistProfileResponse> {
  return request<ArtistProfileResponse>(
    `/artists/${artistID}`,
  )
}

export { API_BASE_URL }