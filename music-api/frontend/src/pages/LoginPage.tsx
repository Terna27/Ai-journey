import {
  useState,
  type FormEvent,
} from 'react'

import {
  Link,
  useLocation,
  useNavigate,
} from 'react-router-dom'

import { useAuth } from '../context/AuthContext'
import { loginUser } from '../lib/api'

type LoginLocationState = {
  from?: string
  message?: string
}

function LoginPage() {
  const navigate = useNavigate()
  const location = useLocation()

  const { login } = useAuth()

  const locationState =
    location.state as LoginLocationState | null

  const [email, setEmail] =
    useState('')

  const [password, setPassword] =
    useState('')

  const [error, setError] =
    useState('')

  const [
    isSubmitting,
    setIsSubmitting,
  ] = useState(false)

  async function handleSubmit(
    event: FormEvent<HTMLFormElement>,
  ) {
    event.preventDefault()

    setError('')
    setIsSubmitting(true)

    try {
      const result =
        await loginUser({
          email: email.trim(),
          password,
        })

      await login(result.token)

      navigate(
        locationState?.from ?? '/',
        {
          replace: true,
        },
      )
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : 'Unable to sign in',
      )
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <section className="form-page">
      <div className="form-card">
        <p className="eyebrow">
          YOUR ACCOUNT
        </p>

        <h2>Welcome back</h2>

        <p className="form-description">
          Sign in to listen to music and
          manage your account.
        </p>

        {locationState?.message && (
          <div
            className="status-message"
            role="status"
          >
            {locationState.message}
          </div>
        )}

        {error && (
          <div
            className="form-error"
            role="alert"
          >
            {error}
          </div>
        )}

        <form
          className="auth-form"
          onSubmit={handleSubmit}
        >
          <label>
            Email

            <input
              type="email"
              name="email"
              placeholder="you@example.com"
              autoComplete="email"
              value={email}
              onChange={(event) =>
                setEmail(
                  event.target.value,
                )
              }
              required
            />
          </label>

          <label>
            Password

            <input
              type="password"
              name="password"
              placeholder="Enter your password"
              autoComplete="current-password"
              value={password}
              onChange={(event) =>
                setPassword(
                  event.target.value,
                )
              }
              required
            />
          </label>

          <button
            type="submit"
            className="primary-button form-submit"
            disabled={isSubmitting}
          >
            {isSubmitting
              ? 'Signing in...'
              : 'Sign in'}
          </button>
        </form>

        <p className="form-footer">
          Don't have an account?{' '}
          <Link to="/register">
            Create one
          </Link>
        </p>
      </div>
    </section>
  )
}

export default LoginPage