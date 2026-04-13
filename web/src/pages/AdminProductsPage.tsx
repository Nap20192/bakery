import { useEffect, useState } from 'react'
import { productsApi } from '../api/resources'
import type { Product } from '../types'

export default function AdminProductsPage() {
  const [products, setProducts] = useState<Product[]>([])
  const [form, setForm] = useState({ name: '', unit: 'шт', iiko_id: '' })
  const [error, setError] = useState<string | null>(null)

  const load = async () => {
    try {
      setProducts(await productsApi.list())
    } catch (e) {
      setError((e as Error).message)
    }
  }

  useEffect(() => {
    load()
  }, [])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      await productsApi.create({ name: form.name, unit: form.unit, iiko_id: form.iiko_id || undefined })
      setForm({ name: '', unit: 'шт', iiko_id: '' })
      await load()
    } catch (e) {
      setError((e as Error).message)
    }
  }

  return (
    <div className="container">
      <h1>Продукты</h1>
      {error && <div className="error card">{error}</div>}

      <form className="card" onSubmit={handleSubmit}>
        <h2>Добавить продукт</h2>
        <div className="form-row">
          <label className="form-group">
            <span>Название</span>
            <input
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              required
            />
          </label>
          <label className="form-group" style={{ maxWidth: 120 }}>
            <span>Ед. изм.</span>
            <select value={form.unit} onChange={(e) => setForm({ ...form, unit: e.target.value })}>
              <option>шт</option>
              <option>кг</option>
            </select>
          </label>
          <label className="form-group">
            <span>iiko ID</span>
            <input
              value={form.iiko_id}
              onChange={(e) => setForm({ ...form, iiko_id: e.target.value })}
              placeholder="UUID из iiko"
            />
          </label>
        </div>
        <button className="btn btn-primary">Создать</button>
      </form>

      <div className="card">
        <h2>Каталог ({products.length})</h2>
        <table>
          <thead>
            <tr>
              <th>Название</th>
              <th>Ед.</th>
              <th>iiko ID</th>
            </tr>
          </thead>
          <tbody>
            {products.map((p) => (
              <tr key={p.id}>
                <td>{p.name}</td>
                <td>{p.unit}</td>
                <td className="muted">{p.iiko_id ?? '—'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
