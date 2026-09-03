import { Link } from 'react-router-dom'

function LoginPage() {
  return (
    <section className="form-page">
      <div className="form-card">
        <p className="eyebrow">ARTIST ACCOUNT</p>

        <h2>Welcome back</h2>

        <p className="form-description">
          Sign in to upload music and manage your releases.
        </p>

        <form className="auth-form">
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
              placeholder="Enter your password"
              autoComplete="current-password"
            />
          </label>

          <button type="submit" className="primary-button form-submit">
            Sign in
          </button>
        </form>

        <p className="form-footer">
          Don't have an account? <Link to="/register">Create one</Link>
        </p>
      </div>
    </section>
  )
}

export default LoginPage