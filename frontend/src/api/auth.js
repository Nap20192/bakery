import { apiBase } from '../config/env';
import { apiURL } from '../lib/url';

// loginWithTelegram exchanges Telegram Login Widget data for a bearer session
// token. It does not use authHeader (there is no session yet).
export async function loginWithTelegram(telegramUser) {
  const url = apiURL(apiBase, '/login');
  const response = await fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(telegramUser),
  });
  const contentType = response.headers.get('content-type') || '';
  const payload = contentType.includes('application/json') ? await response.json().catch(() => ({})) : {};
  if (!response.ok) {
    throw new Error(payload.error || `HTTP ${response.status}`);
  }
  return payload; // { token, expires_at }
}
