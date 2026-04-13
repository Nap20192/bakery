import { useCallback, useEffect, useState } from 'react'
import { overviewApi, type DayGroup, type IngredientLine, type PeriodOverview } from '../api/resources'
import { StatusBadge } from '../components/StatusBadge'

function todayISO() {
  return new Date().toISOString().slice(0, 10)
}

type Mode = 'day' | 'period'

export default function OverviewPage() {
  const [mode, setMode] = useState<Mode>('day')
  const [date, setDate] = useState(todayISO())
  const [from, setFrom] = useState(todayISO())
  const [to, setTo] = useState(todayISO())
  const [day, setDay] = useState<DayGroup | null>(null)
  const [period, setPeriod] = useState<PeriodOverview | null>(null)
  const [err, setErr] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    setErr(null)
    try {
      if (mode === 'day') {
        setPeriod(null)
        setDay(await overviewApi.day(date))
      } else {
        setDay(null)
        setPeriod(await overviewApi.period(from, to))
      }
    } catch (e) {
      setErr((e as Error).message)
    } finally {
      setLoading(false)
    }
  }, [mode, date, from, to])

  useEffect(() => {
    load()
  }, [load])

  return (
    <div className="container">
      <h1>Сводка по ингредиентам</h1>

      <div className="card">
        <div className="form-row" style={{ alignItems: 'flex-end' }}>
          <label className="form-group">
            <span>Режим</span>
            <select value={mode} onChange={(e) => setMode(e.target.value as Mode)}>
              <option value="day">День</option>
              <option value="period">Период</option>
            </select>
          </label>
          {mode === 'day' ? (
            <label className="form-group">
              <span>Дата</span>
              <input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
            </label>
          ) : (
            <>
              <label className="form-group">
                <span>С</span>
                <input type="date" value={from} onChange={(e) => setFrom(e.target.value)} />
              </label>
              <label className="form-group">
                <span>По</span>
                <input type="date" value={to} onChange={(e) => setTo(e.target.value)} />
              </label>
            </>
          )}
          <button className="btn btn-ghost" onClick={load}>
            Обновить
          </button>
        </div>
      </div>

      {loading && <div className="card">Загрузка…</div>}
      {err && <div className="error card">{err}</div>}

      {day && <DaySection group={day} />}
      {period && (
        <>
          <IngredientTable title={`Итого за период ${period.from} — ${period.to}`} lines={period.grand_total} />
          {period.days.map((d) => (
            <DaySection key={d.date} group={d} />
          ))}
        </>
      )}
    </div>
  )
}

function DaySection({ group }: { group: DayGroup }) {
  return (
    <section className="card">
      <h2>📅 {group.date}</h2>
      {group.orders.length === 0 ? (
        <div className="empty-sm">Заявок нет</div>
      ) : (
        <ul className="items">
          {group.orders.map((o) => (
            <li key={o.order_id}>
              <span>
                <StatusBadge status={o.status} /> {o.client || '—'}
                {o.kitchen && ` → ${o.kitchen}`}
              </span>
              <span>{o.items.reduce((n, it) => n + it.quantity, 0)} поз.</span>
            </li>
          ))}
        </ul>
      )}
      <IngredientTable title="Итого за день" lines={group.total_ingredients} />
    </section>
  )
}

function IngredientTable({ title, lines }: { title: string; lines: IngredientLine[] }) {
  if (lines.length === 0) {
    return (
      <div>
        <h3>{title}</h3>
        <div className="empty-sm">— нет данных —</div>
      </div>
    )
  }
  return (
    <div>
      <h3>{title}</h3>
      <table className="ing-table">
        <thead>
          <tr>
            <th>Ингредиент</th>
            <th>Кол-во</th>
            <th>Ед.</th>
          </tr>
        </thead>
        <tbody>
          {lines.map((l) => (
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
    </div>
  )
}
