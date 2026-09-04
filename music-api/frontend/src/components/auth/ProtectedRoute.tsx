import {
  Navigate,
  Outlet,
  useLocation,
} from 'react-router-dom'

import { useAuth } from '../../context/AuthContext'

type ProtectedRouteProps = {
  requireArtist?: boolean
}

function ProtectedRoute({
  requireArtist = false,
}: ProtectedRouteProps) {
  const {
    isAuthenticated,
    isArtist,
    isLoadingIdentity,
  } = useAuth()

  const location = useLocation()

  if (isLoadingIdentity) {
    return (
      <section className="content-panel">
        <p>Loading your account...</p>
      </section>
    )
  }

  if (!isAuthenticated) {
    return (
      <Navigate
        to="/login"
        replace
        state={{
          from: location.pathname,
          message:
            'Please log in to continue.',
        }}
      />
    )
  }

  if (requireArtist && !isArtist) {
    return (
      <Navigate
        to="/profile"
        replace
        state={{
          message:
            'Create your artist profile to use artist features.',
        }}
      />
    )
  }

  return <Outlet />
}

export default ProtectedRoute