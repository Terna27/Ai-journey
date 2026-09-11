import {
  NavLink,
  Outlet,
} from 'react-router-dom'

import { useAuth } from '../../context/AuthContext'
import MusicPlayer from '../player/MusicPlayer'

function AppLayout() {
  const {
    isAuthenticated,
    isArtist,
  } = useAuth()

  const navClass = ({
    isActive,
  }: {
    isActive: boolean
  }) =>
    isActive
      ? 'nav-link active'
      : 'nav-link'

  return (
    <div className="app-shell">
      <aside className="sidebar">
        <NavLink
          to="/"
          className="brand"
        >
          <span className="brand-mark">
            M
          </span>

          <span className="brand-name">
            Music
          </span>
        </NavLink>

        <nav className="sidebar-nav">
          <NavLink
            to="/"
            end
            className={navClass}
          >
            Home
          </NavLink>

          <NavLink
            to="/discover"
            className={navClass}
          >
            Discover
          </NavLink>

          <NavLink
            to="/search"
            className={navClass}
          >
            Search
          </NavLink>

          <NavLink
            to="/podcasts"
            className={navClass}
          >
            Podcasts
          </NavLink>

          {isAuthenticated && (
            <>
              <NavLink
                to="/liked"
                className={navClass}
              >
                Liked Songs
              </NavLink>

              <NavLink
                to="/recently-played"
                className={navClass}
              >
                Recently Played
              </NavLink>

              <NavLink
                to="/podcast-history"
                className={navClass}
              >
                Podcast History
              </NavLink>

              <NavLink
                to="/playlists"
                className={navClass}
              >
                Playlists
              </NavLink>

              <NavLink
                to="/my-podcasts"
                className={navClass}
              >
                My Podcasts
              </NavLink>
            </>
          )}

          {isArtist && (
            <>
              <NavLink
                to="/my-music"
                className={navClass}
              >
                My Music
              </NavLink>

              <NavLink
                to="/upload"
                className={navClass}
              >
                Upload
              </NavLink>
            </>
          )}
        </nav>

        {isAuthenticated && (
          <div className="sidebar-footer">
            <NavLink
              to="/profile"
              className={navClass}
            >
              Profile
            </NavLink>
          </div>
        )}
      </aside>

      <main className="main-content">
        <Outlet />
      </main>

      <MusicPlayer />

      <nav className="mobile-nav">
        <NavLink
          to="/"
          end
          className={navClass}
        >
          Home
        </NavLink>

        <NavLink
          to="/discover"
          className={navClass}
        >
          Discover
        </NavLink>

        <NavLink
          to="/search"
          className={navClass}
        >
          Search
        </NavLink>

        <NavLink
          to="/podcasts"
          className={navClass}
        >
          Podcasts
        </NavLink>

        {isAuthenticated ? (
          <>
            <NavLink
              to="/liked"
              className={navClass}
            >
              Liked
            </NavLink>

            <NavLink
              to="/recently-played"
              className={navClass}
            >
              Recent
            </NavLink>

            <NavLink
              to="/playlists"
              className={navClass}
            >
              Playlists
            </NavLink>

            <NavLink
              to="/my-podcasts"
              className={navClass}
            >
              My Podcasts
            </NavLink>

            {isArtist && (
              <>
                <NavLink
                  to="/my-music"
                  className={navClass}
                >
                  My Music
                </NavLink>

                <NavLink
                  to="/upload"
                  className={navClass}
                >
                  Upload
                </NavLink>
              </>
            )}

            <NavLink
              to="/profile"
              className={navClass}
            >
              Profile
            </NavLink>
          </>
        ) : (
          <>
            <NavLink
              to="/login"
              className={navClass}
            >
              Login
            </NavLink>

            <NavLink
              to="/register"
              className={navClass}
            >
              Register
            </NavLink>
          </>
        )}
      </nav>
    </div>
  )
}

export default AppLayout