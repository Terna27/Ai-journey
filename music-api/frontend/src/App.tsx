import { Navigate, Route, Routes } from 'react-router-dom'
import './App.css'

import AppLayout from './components/layout/AppLayout'
import HomePage from './pages/HomePage'
import LoginPage from './pages/LoginPage'
import RegisterPage from './pages/RegisterPage'
import UploadPage from './pages/UploadPage'
import MyMusicPage from './pages/MyMusicPage'
import ProfilePage from './pages/ProfilePage'

function App() {
  return (
    <Routes>
      <Route element={<AppLayout />}>
        <Route path="/" element={<HomePage />} />
        <Route path="/login" element={<LoginPage />} />
        <Route path="/register" element={<RegisterPage />} />
        <Route path="/upload" element={<UploadPage />} />
        <Route path="/my-music" element={<MyMusicPage />} />
        <Route path="/profile" element={<ProfilePage />} />

        <Route path="*" element={<Navigate to="/" replace />} />
      </Route>
    </Routes>
  )
}

export default App