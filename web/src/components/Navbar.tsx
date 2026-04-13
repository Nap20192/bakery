import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'

export function Navbar() {
  const { session, logout } = useAuth()
  const nav = useNavigate()

  const handleLogout = () => {
    logout()
    nav('/login')
  }

  return (
    <header className="navbar">
      <Link to="/" className="brand">
        🥖 Пекарня Гагарина
      </Link>
      <nav className="nav-links">
        {session?.role === 'client' && <Link to="/client">Мои заявки</Link>}
        {session?.role === 'kitchen' && (
          <>
            <Link to="/kitchen">Очередь</Link>
            <Link to="/overview">Сводка</Link>
          </>
        )}
        {session?.role === 'admin' && (
          <>
            <Link to="/admin/orders">Все заявки</Link>
            <Link to="/admin/products">Продукты</Link>
            <Link to="/overview">Сводка</Link>
          </>
        )}
      </nav>
      <div className="nav-user">
        {session && (
          <>
            <span className={`role-badge role-${session.role}`}>{session.role}</span>
            <button className="btn btn-ghost" onClick={handleLogout}>
              Выйти
            </button>
          </>
        )}
      </div>
    </header>
  )
}
