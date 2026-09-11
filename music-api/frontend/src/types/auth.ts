export type User = {
  id: number
  name: string
  email: string
  email_verified: boolean
  email_verified_at: string | null
  created_at?: string
  updated_at?: string
}

export type Artist = {
  id: number
  user_id: number
  name: string
  email: string

  bio?: string
  profile_image_url?: string
  hero_video_url?: string
  hero_video_poster_url?: string

  created_at?: string
  updated_at?: string
}

export type RegisterRequest = {
  name: string
  email: string
  password: string
}

export type RegisterResponse = {
  user: User
  verification_email_sent: boolean
  message: string
}

export type LoginRequest = {
  email: string
  password: string
}

export type LoginResponse = {
  token: string
  user: User
}

export type MeResponse = {
  user: User
  artist: Artist | null
}

export type VerifyEmailRequest = {
  token: string
}

export type VerifyEmailResponse = {
  message: string
}

export type ResendVerificationRequest = {
  email: string
}

export type ResendVerificationResponse = {
  message: string
}

export type APIErrorResponse = {
  error?: {
    code?: string
    message?: string
  }
}