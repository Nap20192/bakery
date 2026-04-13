import { Navigate, Route, Routes } from 'react-router-dom'
import { Navbar } from './components/Navbar'
import { ProtectedRoute } from './components/ProtectedRoute'
import { useAuth } from './context/AuthContext'
import LoginPage from './pages/LoginPage'
import RegisterPage from './pages/RegisterPage'
import ClientPage from './pages/ClientPage'
import KitchenPage from './pages/KitchenPage'
import AdminOrdersPage from './pages/AdminOrdersPage'
import AdminProductsPage from './pages/AdminProductsPage'
import OverviewPage from './pages/OverviewPage'

function Home() {
  const { session } = useAuth()
  if (!session) return <Navigate to="/login" replace />
  switch (session.role) {
    case 'client':
      return <Navigate to="/client" replace />
    case 'kitchen':
      return <Navigate to="/kitchen" replace />
    case 'admin':
      return <Navigate to="/admin/orders" replace />
  }
}

export default function App() {
  const { session } = useAuth()
  return (
    <>
      {session && <Navbar />}
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/register" element={<RegisterPage />} />
        <Route
          path="/client"
          element={
            <ProtectedRoute roles={['client']}>
              <ClientPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/kitchen"
          element={
            <ProtectedRoute roles={['kitchen']}>
              <KitchenPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/admin/orders"
          element={
            <ProtectedRoute roles={['admin']}>
              <AdminOrdersPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/admin/products"
          element={
            <ProtectedRoute roles={['admin']}>
              <AdminProductsPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/overview"
          element={
            <ProtectedRoute roles={['admin', 'kitchen']}>
              <OverviewPage />
            </ProtectedRoute>
          }
        />
        <Route path="/" element={<Home />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </>
  )
}
