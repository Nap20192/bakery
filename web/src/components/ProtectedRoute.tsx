import { Navigate } from 'react-router-dom'
import type { ReactNode } from 'react'
import { useAuth } from '../context/AuthContext'
import type { Role } from '../types'

interface Props {
  roles?: Role[]
  children: ReactNode
}

export function ProtectedRoute({ roles, children }: Props) {
  const { session } = useAuth()
  if (!session) return <Navigate to="/login" replace />
  if (roles && !roles.includes(session.role)) return <Navigate to="/" replace />
  return <>{children}</>
}
