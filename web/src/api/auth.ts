import { request } from './client'
import type { AuthPayload, Client, Kitchen, Role, User } from '../types'

export interface RegisterBody {
  login: string
  password: string
  name: string
}

export interface MePayload {
  user: User
  role: Role
  client?: Client
  kitchen?: Kitchen
}

export const authApi = {
  login: (login: string, password: string) =>
    request<AuthPayload>('POST', '/auth/login', { login, password }),
  me: () => request<MePayload>('GET', '/auth/me'),
  registerClient: (body: RegisterBody) =>
    request<{ user: User; client: Client }>('POST', '/auth/register/client', body),
  registerKitchen: (body: RegisterBody) =>
    request<{ user: User; kitchen: Kitchen }>('POST', '/auth/register/kitchen', body),
}
