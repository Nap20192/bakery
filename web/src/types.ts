export type Role = 'admin' | 'client' | 'kitchen'

export interface AuthPayload {
  token: string
  role: Role
  exp: number
}

export interface User {
  id: string
  login: string
  role: Role
  created_at: string
}

export interface Client {
  id: string
  user_id: string | null
  name: string
  telegram_chat_id: number | null
  note: string
  created_at: string
}

export interface Kitchen {
  id: string
  user_id: string | null
  name: string
  created_at: string
}

export interface Product {
  id: string
  iiko_id: string | null
  name: string
  unit: string
  created_at: string
}

export type OrderStatus = 'new' | 'parsed' | 'confirmed' | 'processed' | 'cancelled'

export interface OrderItem {
  id: string
  order_id: string
  product_id: string
  quantity: number
  unit: string
}

export interface Order {
  id: string
  client_id: string
  kitchen_id: string | null
  order_date: string
  status: OrderStatus
  raw_text: string
  note: string
  items: OrderItem[]
  created_at: string
}
