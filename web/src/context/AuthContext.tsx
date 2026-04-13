import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from 'react'
import type { ReactNode } from 'react'
import { authApi, type MePayload } from '../api/auth'
import { getToken, setToken } from '../api/client'
import type { Client, Kitchen, Role } from '../types'

interface Session {
  token: string
  role: Role
  exp: number
}

interface AuthContextValue {
  session: Session | null
  me: MePayload | null
  client: Client | null
  kitchen: Kitchen | null
  loadingMe: boolean
  login: (login: string, password: string) => Promise<void>
  logout: () => void
  refreshMe: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue | null>(null)

const SESSION_KEY = 'bakery.session'

function readSession(): Session | null {
  const token = getToken()
  if (!token) return null
  try {
    const raw = localStorage.getItem(SESSION_KEY)
    if (!raw) return null
    const s = JSON.parse(raw) as Session
    if (s.exp * 1000 < Date.now()) return null
    return s
  } catch {
    return null
  }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [session, setSession] = useState<Session | null>(() => readSession())
  const [me, setMe] = useState<MePayload | null>(null)
  const [loadingMe, setLoadingMe] = useState(false)

  useEffect(() => {
    if (session) {
      localStorage.setItem(SESSION_KEY, JSON.stringify(session))
      setToken(session.token)
    } else {
      localStorage.removeItem(SESSION_KEY)
      setToken(null)
      setMe(null)
    }
  }, [session])

  const refreshMe = useCallback(async () => {
    if (!session) {
      console.debug('[auth] refreshMe skipped: no session')
      return
    }
    setLoadingMe(true)
    try {
      console.debug('[auth] GET /auth/me')
      const payload = await authApi.me()
      console.debug('[auth] /auth/me payload', payload)
      setMe(payload)
    } catch (err) {
      console.error('[auth] /auth/me failed', err)
      setMe(null)
    } finally {
      setLoadingMe(false)
    }
  }, [session])

  useEffect(() => {
    if (session && !me) {
      void refreshMe()
    }
  }, [session, me, refreshMe])

  const login = useCallback(async (loginVal: string, password: string) => {
    console.debug('[auth] login', loginVal)
    const payload = await authApi.login(loginVal, password)
    setSession({ token: payload.token, role: payload.role, exp: payload.exp })
  }, [])

  const logout = useCallback(() => {
    console.debug('[auth] logout')
    setSession(null)
  }, [])

  const value = useMemo<AuthContextValue>(
    () => ({
      session,
      me,
      client: me?.client ?? null,
      kitchen: me?.kitchen ?? null,
      loadingMe,
      login,
      logout,
      refreshMe,
    }),
    [session, me, loadingMe, login, logout, refreshMe],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used inside AuthProvider')
  return ctx
}
