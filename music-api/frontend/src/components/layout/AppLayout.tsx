import { NavLink, Outlet } from 'react-router-dom'

function AppLayout() {
  const navClass = ({ isActive }: { isActive: boolean }) =>
    `nav-item${isActive ? ' active' : ''}`

  return (
    <div className="app-shell">
      <aside className="sidebar">
        <div className="brand">
          <div className="brand-mark">M</div>

          <div>
            <h1>Music</h1>
            <span>Discover your sound</span>
          </div>
        </div>

        <nav className="nav">
          <NavLink to="/" end className={navClass}>
            Home
          </NavLink>

          <NavLink to="/" className="nav-item">
            Discover
          </NavLink>

          <NavLink to="/" className="nav-item">
            Search
          </NavLink>

          <NavLink to="/my-music" className={navClass}>
            My Music
          </NavLink>

          <NavLink to="/upload" className={navClass}>
            Upload
          </NavLink>
        </nav>

        <div className="sidebar-footer">
          <NavLink to="/profile" className={navClass}>
            Profile
          </NavLink>
        </div>
      </aside>

      <main className="main-content">
        <Outlet />
      </main>

      <footer className="player">
        <div className="player-track">
          <div className="player-cover" />

          <div>
            <strong>Nothing playing</strong>
            <span>Choose a song to begin</span>
          </div>
        </div>

        <div className="player-controls">
          <div className="control-buttons">
            <button type="button" aria-label="Previous track">
              ◀
            </button>

            <button type="button" className="main-play" aria-label="Play">
              ▶
            </button>

            <button type="button" aria-label="Next track">
              ▶
            </button>
          </div>

          <div className="progress-row">
            <span>0:00</span>

            <div className="progress">
              <div className="progress-value" />
            </div>

            <span>0:00</span>
          </div>
        </div>

        <div className="volume">
          <span>🔊</span>

          <div className="volume-bar">
            <div className="volume-value" />
          </div>
        </div>
      </footer>
    </div>
  )
}

export default AppLayout