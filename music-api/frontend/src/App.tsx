import {
  Navigate,
  Route,
  Routes,
} from 'react-router-dom'

import ProtectedRoute from './components/auth/ProtectedRoute'
import AppLayout from './components/layout/AppLayout'

import ArtistPage from './pages/ArtistPage'
import DiscoverPage from './pages/DiscoverPage'
import HomePage from './pages/HomePage'
import LikedMusicPage from './pages/LikedMusicPage'
import LoginPage from './pages/LoginPage'
import MyMusicPage from './pages/MyMusicPage'
import MyPodcastsPage from './pages/MyPodcastsPage'
import MyReleasesPage from './pages/MyReleasesPage'
import PlaylistDetailsPage from './pages/PlaylistDetailsPage'
import PodcastPage from './pages/PodcastPage'
import PodcastHistoryPage from './pages/PodcastHistoryPage'
import PodcastManagerPage from './pages/PodcastManagerPage'
import PodcastsPage from './pages/PodcastsPage'
import PlaylistsPage from './pages/PlaylistsPage'
import ProfilePage from './pages/ProfilePage'
import RegisterPage from './pages/RegisterPage'
import RecentlyPlayedPage from './pages/RecentlyPlayedPage'
import ReleaseManagerPage from './pages/ReleaseManagerPage'
import ReleasePage from './pages/ReleasePage'
import SearchPage from './pages/SearchPage'
import UploadPage from './pages/UploadPage'
import VerifyEmailPage from './pages/VerifyEmailPage'

import './App.css'

function App() {
  return (
    <Routes>
      <Route element={<AppLayout />}>
        <Route
          path="/"
          element={<HomePage />}
        />

        <Route
          path="/discover"
          element={<DiscoverPage />}
        />

        <Route
          path="/search"
          element={<SearchPage />}
        />

        <Route
          path="/podcasts"
          element={<PodcastsPage />}
        />

        <Route
          path="/podcasts/:slug"
          element={<PodcastPage />}
        />

        <Route
          path="/artists/:id"
          element={<ArtistPage />}
        />

        <Route
          path="/releases/:id"
          element={<ReleasePage />}
        />

        <Route
          path="/login"
          element={<LoginPage />}
        />

        <Route
          path="/register"
          element={<RegisterPage />}
        />

        <Route
          path="/verify-email"
          element={<VerifyEmailPage />}
        />

        <Route element={<ProtectedRoute />}>
          <Route
            path="/profile"
            element={<ProfilePage />}
          />

          <Route
            path="/liked"
            element={<LikedMusicPage />}
          />

          <Route
            path="/playlists"
            element={<PlaylistsPage />}
          />

          <Route
            path="/playlists/:id"
            element={<PlaylistDetailsPage />}
          />

          <Route
            path="/recently-played"
            element={<RecentlyPlayedPage />}
          />

          <Route
            path="/podcast-history"
            element={<PodcastHistoryPage />}
          />

          <Route
            path="/my-podcasts"
            element={<MyPodcastsPage />}
          />

          <Route
            path="/my-podcasts/:id"
            element={<PodcastManagerPage />}
          />
        </Route>

        <Route
          element={
            <ProtectedRoute
              requireArtist
            />
          }
        >
          <Route
            path="/upload"
            element={<UploadPage />}
          />

          <Route
            path="/my-music"
            element={<MyMusicPage />}
          />

          <Route
            path="/my-releases"
            element={<MyReleasesPage />}
          />

          <Route
            path="/my-releases/:id"
            element={<ReleaseManagerPage />}
          />
        </Route>

        <Route
          path="*"
          element={
            <Navigate
              to="/"
              replace
            />
          }
        />
      </Route>
    </Routes>
  )
}

export default App