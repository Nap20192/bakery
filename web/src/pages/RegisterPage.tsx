import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { authApi } from '../api/auth'
import { useAuth } from '../context/AuthContext'

type RegMode = 'client' | 'kitchen'

export default function RegisterPage() {
  const { login } = useAuth()
  const nav = useNavigate()
  const [mode, setMode] = useState<RegMode>('client')
  const [form, setForm] = useState({ login: '', password: '', name: '' })
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  const set = (k: keyof typeof form) => (e: React.ChangeEvent<HTMLInputElement>) =>
    setForm((f) => ({ ...f, [k]: e.target.value }))

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)
    setSubmitting(true)
    try {
      if (mode === 'client') await authApi.registerClient(form)
      else await authApi.registerKitchen(form)
      await login(form.login, form.password)
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
        <h1>Регистрация</h1>
        <div className="tabs">
          <button
            type="button"
            className={`tab ${mode === 'client' ? 'active' : ''}`}
            onClick={() => setMode('client')}
          >
            Клиент
          </button>
          <button
            type="button"
            className={`tab ${mode === 'kitchen' ? 'active' : ''}`}
            onClick={() => setMode('kitchen')}
          >
            Кухня
          </button>
        </div>
        <label className="form-group">
          <span>Название {mode === 'client' ? 'клиента' : 'пекарни'}</span>
          <input value={form.name} onChange={set('name')} required />
        </label>
        <label className="form-group">
          <span>Логин</span>
          <input value={form.login} onChange={set('login')} required />
        </label>
        <label className="form-group">
          <span>Пароль</span>
          <input type="password" value={form.password} onChange={set('password')} required />
        </label>
        {error && <div className="error">{error}</div>}
        <button className="btn btn-primary" type="submit" disabled={submitting}>
          {submitting ? '...' : 'Создать аккаунт'}
        </button>
        <div className="auth-switch">
          Уже есть аккаунт? <Link to="/login">Войти</Link>
        </div>
      </form>
    </div>
  )
}
