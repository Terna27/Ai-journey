import { Link } from 'react-router-dom'

function RegisterPage() {
  return (
    <section className="form-page">
      <div className="form-card">
        <p className="eyebrow">JOIN AS AN ARTIST</p>

        <h2>Create your account</h2>

        <p className="form-description">
          Create an artist account and start sharing your music.
        </p>

        <form className="auth-form">
          <label>
            Artist name
            <input
              type="text"
              name="name"
              placeholder="Your artist name"
              autoComplete="name"
            />
          </label>

          <label>
            Email
            <input
              type="email"
              name="email"
              placeholder="artist@example.com"
              autoComplete="email"
            />
          </label>

          <label>
            Password
            <input
              type="password"
              name="password"
              placeholder="Create a password"
              autoComplete="new-password"
            />
          </label>

          <button type="submit" className="primary-button form-submit">
            Create account
          </button>
        </form>

        <p className="form-footer">
          Already have an account? <Link to="/login">Sign in</Link>
        </p>
      </div>
    </section>
  )
}

export default RegisterPage