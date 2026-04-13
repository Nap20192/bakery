import { useCallback, useEffect, useState } from 'react'
import { OrderList } from '../components/OrderList'
import { ordersApi, productsApi, clientsApi } from '../api/resources'
import type { Client, Order, OrderStatus, Product } from '../types'

function todayISO() {
  return new Date().toISOString().slice(0, 10)
}

const STATUSES: (OrderStatus | '')[] = ['', 'new', 'confirmed', 'processed', 'cancelled']

export default function KitchenPage() {
  const [products, setProducts] = useState<Product[]>([])
  const [clients, setClients] = useState<Client[]>([])
  const [orders, setOrders] = useState<Order[]>([])
  const [date, setDate] = useState(todayISO())
  const [statusFilter, setStatusFilter] = useState<OrderStatus | ''>('')
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    setError(null)
    try {
      const [ps, cs, os] = await Promise.all([
        productsApi.list(),
        clientsApi.list(),
        ordersApi.list(date, statusFilter),
      ])
      setProducts(ps)
      setClients(cs)
      setOrders(os)
    } catch (e) {
      setError((e as Error).message)
    }
  }, [date, statusFilter])

  useEffect(() => {
    load()
  }, [load])

  const setStatus = async (id: string, status: OrderStatus) => {
    try {
      await ordersApi.setStatus(id, status)
      await load()
    } catch (e) {
      setError((e as Error).message)
    }
  }

  const clientName = (id: string) => clients.find((c) => c.id === id)?.name ?? id.slice(0, 8)

  return (
    <div className="container">
      <h1>Очередь кухни</h1>
      {error && <div className="error card">{error}</div>}

      <div className="card">
        <div className="form-row" style={{ alignItems: 'flex-end' }}>
          <label className="form-group">
            <span>Дата</span>
            <input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
          </label>
          <label className="form-group">
            <span>Статус</span>
            <select
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value as OrderStatus | '')}
            >
              {STATUSES.map((s) => (
                <option key={s} value={s}>
                  {s || 'все'}
                </option>
              ))}
            </select>
          </label>
          <button className="btn btn-ghost" onClick={load}>
            Обновить
          </button>
        </div>
      </div>

      <OrderList
        orders={orders.map((o) => ({
          ...o,
          note: `${clientName(o.client_id)}${o.note ? ' — ' + o.note : ''}`,
        }))}
        products={products}
        actions={(o) => (
          <>
            {o.status !== 'confirmed' && o.status !== 'processed' && (
              <button
                className="btn btn-sm btn-warning"
                onClick={() => setStatus(o.id, 'confirmed')}
              >
                Принять
              </button>
            )}
            {o.status !== 'processed' && (
              <button
                className="btn btn-sm btn-success"
                onClick={() => setStatus(o.id, 'processed')}
              >
                Закрыть
              </button>
            )}
            {o.status !== 'cancelled' && o.status !== 'processed' && (
              <button
                className="btn btn-sm btn-danger"
                onClick={() => setStatus(o.id, 'cancelled')}
              >
                Отклонить
              </button>
            )}
          </>
        )}
      />
    </div>
  )
}
