import { apiBase } from '../config/env';
import { apiURL } from '../lib/url';

// login exchanges username + password for a bearer session token.
// No authHeader (no session yet).
export async function login(username, password) {
  const url = apiURL(apiBase, '/login');
  const response = await fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, password }),
  });
  const contentType = response.headers.get('content-type') || '';
  const payload = contentType.includes('application/json') ? await response.json().catch(() => ({})) : {};
  if (!response.ok) {
    throw new Error(payload.error || `HTTP ${response.status}`);
  }
  return payload; // { token, expires_at }
}
