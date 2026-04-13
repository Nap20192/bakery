import { useCallback, useEffect, useState } from 'react'
import { OrderBuilder, type BuilderItem } from '../components/OrderBuilder'
import { OrderList } from '../components/OrderList'
import { ordersApi, productsApi } from '../api/resources'
import { useAuth } from '../context/AuthContext'
import type { Order, Product } from '../types'

function todayISO() {
  return new Date().toISOString().slice(0, 10)
}

export default function ClientPage() {
  const { client: myClient, loadingMe, refreshMe } = useAuth()
  const [products, setProducts] = useState<Product[]>([])
  const [orders, setOrders] = useState<Order[]>([])
  const [date, setDate] = useState(todayISO())
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  const load = useCallback(async () => {
    try {
      console.debug('[client] load', { date })
      const [ps, os] = await Promise.all([productsApi.list(), ordersApi.list(date)])
      console.debug('[client] loaded', { products: ps.length, orders: os.length })
      setProducts(ps)
      setOrders(os)
    } catch (e) {
      console.error('[client] load failed', e)
      setError((e as Error).message)
    }
  }, [date])

  useEffect(() => {
    load()
  }, [load])

  const handleCreate = async (items: BuilderItem[], note: string, orderDate: string) => {
    console.debug('[client] handleCreate', { items, note, orderDate, myClient })
    if (!myClient) {
      setError(
        loadingMe
          ? 'Загружаю профиль… попробуйте ещё раз через секунду.'
          : 'Не найден привязанный клиентский профиль. Войдите заново после регистрации как клиент.',
      )
      return
    }
    if (items.length === 0) {
      setError('Добавьте хотя бы одну позицию')
      return
    }
    setSubmitting(true)
    setError(null)
    try {
      const body = {
        client_id: myClient.id,
        order_date: orderDate,
        note,
        raw_text: items
          .map(
            (it) =>
              `${it.quantity} ${it.unit} ${
                products.find((p) => p.id === it.product_id)?.name ?? ''
              }`,
          )
          .join(', '),
        items,
      }
      console.debug('[client] POST /orders', body)
      const created = await ordersApi.create(body)
      console.debug('[client] order created', created)
      await load()
    } catch (e) {
      console.error('[client] create failed', e)
      setError((e as Error).message)
    } finally {
      setSubmitting(false)
    }
  }

  useEffect(() => {
    if (!myClient && !loadingMe) {
      console.warn('[client] no myClient profile — triggering refreshMe')
      void refreshMe()
    }
  }, [myClient, loadingMe, refreshMe])

  const handleCancel = async (id: string) => {
    if (!confirm('Отменить заявку?')) return
    try {
      await ordersApi.cancel(id)
      await load()
    } catch (e) {
      setError((e as Error).message)
    }
  }

  return (
    <div className="container">
      <h1>Мои заявки</h1>
      {error && <div className="error card">{error}</div>}

      <OrderBuilder products={products} onSubmit={handleCreate} submitting={submitting} />

      <div className="card">
        <div className="form-row" style={{ alignItems: 'flex-end' }}>
          <label className="form-group">
            <span>Фильтр по дате</span>
            <input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
          </label>
          <button className="btn btn-ghost" onClick={load}>
            Обновить
          </button>
        </div>
      </div>

      <OrderList
        orders={orders}
        products={products}
        actions={(o) =>
          o.status === 'new' ? (
            <button className="btn btn-sm btn-danger" onClick={() => handleCancel(o.id)}>
              Отменить
            </button>
          ) : null
        }
      />
    </div>
  )
}
