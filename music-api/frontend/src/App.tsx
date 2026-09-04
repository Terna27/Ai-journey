import {
  Navigate,
  Route,
  Routes,
} from 'react-router-dom'

import ProtectedRoute from './components/auth/ProtectedRoute'
import AppLayout from './components/layout/AppLayout'

import HomePage from './pages/HomePage'
import LoginPage from './pages/LoginPage'
import MyMusicPage from './pages/MyMusicPage'
import ProfilePage from './pages/ProfilePage'
import RegisterPage from './pages/RegisterPage'
import UploadPage from './pages/UploadPage'

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

        <Route element={<ProtectedRoute />}>
          <Route
            path="/profile"
            element={<ProfilePage />}
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