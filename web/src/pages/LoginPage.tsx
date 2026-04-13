import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'

export default function LoginPage() {
  const { login } = useAuth()
  const nav = useNavigate()
  const [loginVal, setLoginVal] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)
    setSubmitting(true)
    try {
      await login(loginVal, password)
      nav('/')
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="auth-page">
      <form className="card auth-card" onSubmit={handleSubmit}>
        <h1>Вход</h1>
        <label className="form-group">
          <span>Логин</span>
          <input value={loginVal} onChange={(e) => setLoginVal(e.target.value)} required />
        </label>
        <label className="form-group">
          <span>Пароль</span>
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
          />
        </label>
        {error && <div className="error">{error}</div>}
        <button className="btn btn-primary" type="submit" disabled={submitting}>
          {submitting ? '...' : 'Войти'}
        </button>
        <div className="auth-switch">
          Нет аккаунта? <Link to="/register">Регистрация</Link>
        </div>
      </form>
    </div>
  )
}
