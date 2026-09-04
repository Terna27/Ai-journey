import {
  useState,
  type FormEvent,
} from 'react'

import {
  Link,
  useNavigate,
} from 'react-router-dom'

import { registerUser } from '../lib/api'

function RegisterPage() {
  const navigate = useNavigate()

  const [name, setName] =
    useState('')

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
      await registerUser({
        name: name.trim(),
        email: email.trim(),
        password,
      })

      navigate('/login', {
        replace: true,
        state: {
          message:
            'Account created successfully. Sign in to continue.',
        },
      })
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : 'Unable to create account',
      )
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <section className="form-page">
      <div className="form-card">
        <p className="eyebrow">
          JOIN MUSIC
        </p>

        <h2>Create your account</h2>

        <p className="form-description">
          Create one account to listen to
          music. You can create an artist
          profile later if you want to
          publish your own songs.
        </p>

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
            Name

            <input
              type="text"
              name="name"
              placeholder="Your name"
              autoComplete="name"
              value={name}
              onChange={(event) =>
                setName(
                  event.target.value,
                )
              }
              required
            />
          </label>

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
              placeholder="Create a password"
              autoComplete="new-password"
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
              ? 'Creating account...'
              : 'Create account'}
          </button>
        </form>

        <p className="form-footer">
          Already have an account?{' '}
          <Link to="/login">
            Sign in
          </Link>
        </p>
      </div>
    </section>
  )
}

export default RegisterPage