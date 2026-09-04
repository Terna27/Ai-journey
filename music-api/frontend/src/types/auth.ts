export type User = {
  id: number
  name: string
  email: string
  created_at?: string
  updated_at?: string
}

export type Artist = {
  id: number
  user_id: number
  name: string
  email: string
  created_at?: string
  updated_at?: string
}

export type RegisterRequest = {
  name: string
  email: string
  password: string
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

export type APIErrorResponse = {
  error?: {
    code?: string
    message?: string
  }
}