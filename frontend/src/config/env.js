export const apiBase = import.meta.env.VITE_API_BASE_URL || '/api';
export const frontendLogsEnabled = import.meta.env.VITE_FRONTEND_LOGS !== 'false';
export const buildMode = import.meta.env.MODE;
// Bot username (without @) used by the Telegram Login Widget on the plain web build.
export const telegramBotUsername = import.meta.env.VITE_TELEGRAM_BOT_USERNAME || '';
