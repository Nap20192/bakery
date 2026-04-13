import { useState } from 'react'
import type { Product } from '../types'

export interface BuilderItem {
  product_id: string
  quantity: number
  unit: string
}

interface Props {
  products: Product[]
  onSubmit: (items: BuilderItem[], note: string, date: string) => Promise<void>
  submitting?: boolean
}

const todayISO = () => new Date().toISOString().slice(0, 10)

export function OrderBuilder({ products, onSubmit, submitting }: Props) {
  const [rows, setRows] = useState<BuilderItem[]>([
    { product_id: '', quantity: 0, unit: 'шт' },
  ])
  const [note, setNote] = useState('')
  const [date, setDate] = useState(todayISO())

  const updateRow = (i: number, patch: Partial<BuilderItem>) => {
    setRows((rs) => rs.map((r, idx) => (idx === i ? { ...r, ...patch } : r)))
  }

  const addRow = () =>
    setRows((rs) => [...rs, { product_id: '', quantity: 0, unit: 'шт' }])

  const removeRow = (i: number) => setRows((rs) => rs.filter((_, idx) => idx !== i))

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    const valid = rows.filter((r) => r.product_id && r.quantity > 0)
    if (valid.length === 0) return
    await onSubmit(valid, note, date)
    setRows([{ product_id: '', quantity: 0, unit: 'шт' }])
    setNote('')
  }

  return (
    <form className="card" onSubmit={handleSubmit}>
      <h2>Новая заявка</h2>
      <div className="form-row">
        <label className="form-group">
          <span>Дата исполнения</span>
          <input type="date" value={date} onChange={(e) => setDate(e.target.value)} required />
        </label>
        <label className="form-group" style={{ flex: 2 }}>
          <span>Комментарий</span>
          <input
            placeholder="К 6 утра"
            value={note}
            onChange={(e) => setNote(e.target.value)}
          />
        </label>
      </div>

      <h3 style={{ marginTop: 12, color: '#8B4513' }}>Состав</h3>
      <div className="builder">
        {rows.map((row, i) => (
          <div className="builder-row" key={i}>
            <select
              value={row.product_id}
              onChange={(e) => {
                const p = products.find((p) => p.id === e.target.value)
                updateRow(i, { product_id: e.target.value, unit: p?.unit ?? row.unit })
              }}
            >
              <option value="">— продукт —</option>
              {products.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name} ({p.unit})
                </option>
              ))}
            </select>
            <input
              type="number"
              step="0.001"
              min="0"
              placeholder="кол-во"
              value={row.quantity || ''}
              onChange={(e) => updateRow(i, { quantity: parseFloat(e.target.value) || 0 })}
            />
            <input
              value={row.unit}
              onChange={(e) => updateRow(i, { unit: e.target.value })}
            />
            <button type="button" className="btn btn-danger btn-sm" onClick={() => removeRow(i)}>
              ×
            </button>
          </div>
        ))}
      </div>
      <button type="button" className="btn btn-ghost btn-sm" onClick={addRow}>
        + позиция
      </button>

      <div className="btn-group" style={{ marginTop: 16 }}>
        <button className="btn btn-primary" type="submit" disabled={submitting}>
          {submitting ? 'Отправляю…' : 'Оформить заявку'}
        </button>
      </div>
    </form>
  )
}
