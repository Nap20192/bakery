export function initializeTelegramMiniApp() {
  const webApp = window.Telegram?.WebApp;
  if (!webApp) return;
  webApp.ready();
  webApp.expand();
}

export function telegramInitData() {
  return window.Telegram?.WebApp?.initData || import.meta.env.VITE_TELEGRAM_INIT_DATA || '';
}
