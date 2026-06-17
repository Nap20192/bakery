import { isTelegramMiniApp, telegramInitData } from './telegram';

const TOKEN_KEY = 'bakery.session.token';

// isWebMode is true when running as a plain website (outside Telegram), where
// the user authenticates via the Telegram Login Widget and a bearer token.
export function isWebMode() {
  return !isTelegramMiniApp();
}

export function getToken() {
  try {
    return localStorage.getItem(TOKEN_KEY) || '';
  } catch {
    return '';
  }
}

export function setToken(token) {
  try {
    localStorage.setItem(TOKEN_KEY, token);
  } catch {
    /* ignore storage errors (private mode, etc.) */
  }
}

export function clearToken() {
  try {
    localStorage.removeItem(TOKEN_KEY);
  } catch {
    /* ignore */
  }
}

// authHeader returns the Authorization header for API requests: Mini App
// initData inside Telegram, otherwise a stored bearer session token.
export function authHeader() {
  const initData = telegramInitData();
  if (initData) {
    return { Authorization: `tma ${initData}` };
  }
  const token = getToken();
  return token ? { Authorization: `Bearer ${token}` } : {};
}
