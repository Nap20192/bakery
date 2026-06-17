export function initializeTelegramMiniApp() {
  const webApp = window.Telegram?.WebApp;
  if (!webApp) return;
  webApp.ready();
  webApp.expand();
}

export function telegramInitData() {
  return window.Telegram?.WebApp?.initData || import.meta.env.VITE_TELEGRAM_INIT_DATA || '';
}

// isTelegramMiniApp reports whether the app is running inside Telegram (Mini App
// context). When false, the app runs as a plain website and uses bearer auth.
export function isTelegramMiniApp() {
  return Boolean(telegramInitData());
}
