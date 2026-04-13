import { request } from './client'
import type {
  Client,
  Kitchen,
  Order,
  OrderItem,
  OrderStatus,
  Product,
} from '../types'

export const productsApi = {
  list: () => request<Product[]>('GET', '/products'),
  create: (body: { name: string; unit: string; iiko_id?: string }) =>
    request<Product>('POST', '/products', body),
}

export const clientsApi = {
  list: () => request<Client[]>('GET', '/clients'),
  get: (id: string) => request<Client>('GET', `/clients/${id}`),
}

export const kitchensApi = {
  list: () => request<Kitchen[]>('GET', '/kitchens'),
  create: (body: { name: string }) => request<Kitchen>('POST', '/kitchens', body),
}

export interface CreateOrderItemBody {
  product_id: string
  quantity: number
  unit: string
}

export interface CreateOrderBody {
  client_id: string
  kitchen_id?: string | null
  order_date: string // YYYY-MM-DD
  raw_text?: string
  note?: string
  items?: CreateOrderItemBody[]
}

export const ordersApi = {
  list: (date: string, status?: OrderStatus | '') => {
    const qs = new URLSearchParams({ date })
    if (status) qs.set('status', status)
    return request<Order[]>('GET', `/orders?${qs.toString()}`)
  },
  get: (id: string) => request<Order>('GET', `/orders/${id}`),
  create: (body: CreateOrderBody) => request<Order>('POST', '/orders', body),
  cancel: (id: string) => request<void>('DELETE', `/orders/${id}`),
  setStatus: (id: string, status: OrderStatus) =>
    request<Order>('PATCH', `/orders/${id}/status`, { status }),
  addItem: (id: string, body: { product_id: string; quantity: number; unit: string }) =>
    request<OrderItem>('POST', `/orders/${id}/items`, body),
  listItems: (id: string) => request<OrderItem[]>('GET', `/orders/${id}/items`),
  overview: (id: string) => request<OrderOverview>('GET', `/orders/${id}/overview`),
}

export interface IngredientLine {
  ingredient_id: string
  ingredient_name: string
  unit: string
  total_qty: number
}

export interface OrderItemOverview {
  product_id: string
  product_name: string
  quantity: number
  unit: string
}

export interface OrderOverview {
  order_id: string
  order_date: string
  status: OrderStatus
  client: string
  kitchen: string
  note: string
  items: OrderItemOverview[]
  total_ingredients: IngredientLine[]
}

export interface DayGroup {
  date: string
  orders: OrderOverview[]
  total_ingredients: IngredientLine[]
}

export interface PeriodOverview {
  from: string
  to: string
  days: DayGroup[]
  grand_total: IngredientLine[]
}

export const overviewApi = {
  day: (date: string) => request<DayGroup>('GET', `/overview/day?date=${date}`),
  period: (from: string, to: string) =>
    request<PeriodOverview>('GET', `/overview/period?from=${from}&to=${to}`),
}
