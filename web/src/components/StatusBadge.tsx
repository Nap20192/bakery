import type { OrderStatus } from '../types'

const labels: Record<OrderStatus, string> = {
  new: 'новая',
  parsed: 'разобрана',
  confirmed: 'принята',
  processed: 'закрыта',
  cancelled: 'отменена',
}

export function StatusBadge({ status }: { status: OrderStatus }) {
  return <span className={`status-badge status-${status}`}>{labels[status]}</span>
}
