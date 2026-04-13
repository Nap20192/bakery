// Тонкий fetch-wrapper. Достаёт токен из localStorage, подставляет Authorization,
// парсит JSON и бросает Error с сообщением бэкенда ({error: "..."}).

const TOKEN_KEY = 'bakery.token'

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token: string | null) {
  if (token) localStorage.setItem(TOKEN_KEY, token)
  else localStorage.removeItem(TOKEN_KEY)
}

type Method = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'

export async function request<T = unknown>(
  method: Method,
  path: string,
  body?: unknown,
): Promise<T> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  const token = getToken()
  if (token) headers['Authorization'] = `Bearer ${token}`

  const res = await fetch(path, {
    method,
    headers,
    body: body && method !== 'GET' ? JSON.stringify(body) : undefined,
  })

  if (res.status === 204) return undefined as T
  const text = await res.text()
  const data = text ? JSON.parse(text) : null
  if (!res.ok) {
    const msg = (data && data.error) || `HTTP ${res.status}`
    throw new Error(msg)
  }
  return data as T
}
