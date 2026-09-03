import { Link } from 'react-router-dom'

const tracks = [
  {
    id: 1,
    title: 'Midnight Drive',
    artist: 'Nova Echo',
    genre: 'Afrobeats',
  },
  {
    id: 2,
    title: 'Golden Hour',
    artist: 'Lena Ray',
    genre: 'R&B',
  },
  {
    id: 3,
    title: 'City Lights',
    artist: 'Juno',
    genre: 'Pop',
  },
  {
    id: 4,
    title: 'Northern Sky',
    artist: 'Ari Stone',
    genre: 'Soul',
  },
]

function HomePage() {
  return (
    <>
      <header className="topbar">
        <div>
          <p className="eyebrow">WELCOME</p>
          <h2>Discover something new</h2>
        </div>

        <div className="topbar-actions">
          <Link to="/login" className="secondary-button">
            Login
          </Link>

          <Link to="/register" className="primary-button">
            Create account
          </Link>
        </div>
      </header>

      <section className="hero-section">
        <div className="hero-copy">
          <span className="hero-badge">Featured this week</span>

          <h3>Music for every moment.</h3>

          <p>
            Discover independent artists, stream new songs and build a
            collection around the music you love.
          </p>

          <div className="hero-actions">
            <button type="button" className="primary-button">
              Explore music
            </button>

            <Link to="/upload" className="secondary-button">
              Upload your music
            </Link>
          </div>
        </div>

        <div className="hero-art">
          <div className="hero-disc">
            <div className="hero-disc-center" />
          </div>
        </div>
      </section>

      <section className="section">
        <div className="section-heading">
          <div>
            <p className="eyebrow">DISCOVER</p>
            <h3>Trending now</h3>
          </div>

          <button type="button" className="text-button">
            View all
          </button>
        </div>

        <div className="music-grid">
          {tracks.map((track, index) => (
            <article className="music-card" key={track.id}>
              <div className={`music-cover cover-${index + 1}`}>
                <button
                  type="button"
                  className="play-button"
                  aria-label={`Play ${track.title}`}
                >
                  ▶
                </button>
              </div>

              <div className="music-card-content">
                <div>
                  <h4>{track.title}</h4>
                  <p>{track.artist}</p>
                </div>

                <span className="genre-pill">{track.genre}</span>
              </div>
            </article>
          ))}
        </div>
      </section>
    </>
  )
}

export default HomePage