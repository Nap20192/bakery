import { useState } from 'react'
import type { Order, Product } from '../types'
import { StatusBadge } from './StatusBadge'
import { ordersApi, type OrderOverview } from '../api/resources'

interface Props {
  orders: Order[]
  products: Product[]
  actions?: (o: Order) => React.ReactNode
}

export function OrderList({ orders, products, actions }: Props) {
  if (orders.length === 0) {
    return <div className="empty">Заявок нет</div>
  }
  const productName = (id: string) => products.find((p) => p.id === id)?.name ?? id.slice(0, 8)

  return (
    <div className="order-list">
      {orders.map((o) => (
        <article className="order-card" key={o.id}>
          <header className="order-head">
            <div>
              <div className="order-date">{o.order_date}</div>
              <StatusBadge status={o.status} />
            </div>
            {actions && <div className="btn-group">{actions(o)}</div>}
          </header>
          {o.note && <div className="order-note">💬 {o.note}</div>}
          {o.items.length > 0 ? (
            <ul className="items">
              {o.items.map((it) => (
                <li key={it.id}>
                  <span>{productName(it.product_id)}</span>
                  <span>
                    <strong>{it.quantity}</strong> {it.unit}
                  </span>
                </li>
              ))}
            </ul>
          ) : (
            <div className="empty-sm">— без позиций —</div>
          )}
          <OrderOverviewBlock orderId={o.id} />
        </article>
      ))}
    </div>
  )
}

function OrderOverviewBlock({ orderId }: { orderId: string }) {
  const [open, setOpen] = useState(false)
  const [ov, setOv] = useState<OrderOverview | null>(null)
  const [err, setErr] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const toggle = async () => {
    const next = !open
    setOpen(next)
    if (next && !ov) {
      setLoading(true)
      setErr(null)
      try {
        setOv(await ordersApi.overview(orderId))
      } catch (e) {
        setErr((e as Error).message)
      } finally {
        setLoading(false)
      }
    }
  }

  return (
    <div className="overview-block">
      <button className="btn btn-sm btn-ghost" onClick={toggle}>
        {open ? '▾' : '▸'} Состав ингредиентов
      </button>
      {open && (
        <div className="overview-body">
          {loading && <div className="empty-sm">Загрузка…</div>}
          {err && <div className="error">{err}</div>}
          {ov && (
            <>
              {ov.total_ingredients.length === 0 ? (
                <div className="empty-sm">— нет данных по ингредиентам —</div>
              ) : (
                <table className="ing-table">
                  <thead>
                    <tr>
                      <th>Ингредиент</th>
                      <th>Кол-во</th>
                      <th>Ед.</th>
                    </tr>
                  </thead>
                  <tbody>
                    {ov.total_ingredients.map((l) => (
                      <tr key={l.ingredient_id}>
                        <td>{l.ingredient_name}</td>
                        <td>
                          <strong>{l.total_qty.toFixed(3)}</strong>
                        </td>
                        <td>{l.unit}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </>
          )}
        </div>
      )}
    </div>
  )
}
