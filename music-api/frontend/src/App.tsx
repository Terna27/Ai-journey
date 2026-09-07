import {
  Navigate,
  Route,
  Routes,
} from 'react-router-dom'

import ProtectedRoute from './components/auth/ProtectedRoute'
import AppLayout from './components/layout/AppLayout'

import HomePage from './pages/HomePage'
import LikedMusicPage from './pages/LikedMusicPage'
import LoginPage from './pages/LoginPage'
import MyMusicPage from './pages/MyMusicPage'
import PlaylistDetailsPage from './pages/PlaylistDetailsPage'
import PlaylistsPage from './pages/PlaylistsPage'
import ProfilePage from './pages/ProfilePage'
import RegisterPage from './pages/RegisterPage'
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
            element={
              <PlaylistDetailsPage />
            }
          />
        </Route>

        <Route
          element={
            <ProtectedRoute requireArtist />
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