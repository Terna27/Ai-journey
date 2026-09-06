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

const API_BASE_URL =
  import.meta.env.VITE_API_BASE_URL ??
  'http://localhost:8080/api/v1'

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

    try {
      const body =
        (await response.json()) as APIErrorResponse

      if (body.error?.message) {
        message = body.error.message
      }
    } catch {
      // Response did not contain JSON.
    }

    throw new Error(message)
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
): Promise<Music[]> {
  return request<Music[]>(
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

export { API_BASE_URL }