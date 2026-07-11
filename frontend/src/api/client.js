// @ts-check
import createClient from 'openapi-fetch';
import { apiBase } from '../config/env';
import { authHeader } from '../lib/auth';
import { logInfo, logWarn } from '../lib/logger';

export class ApiError extends Error {
  /**
   * @param {string} message
   * @param {number} status
   */
  constructor(message, status) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
  }
}

// statusMessage returns a human-readable fallback when the backend response
// carries no error text (proxy errors, unexpected failures).
/** @param {number} status */
function statusMessage(status) {
  if (status === 401) return 'Требуется вход в систему.';
  if (status === 403) return 'Недостаточно прав для этого действия.';
  if (status === 404) return 'Данные не найдены.';
  if (status === 409) return 'Данные изменились. Обновите страницу и попробуйте снова.';
  if (status >= 500) return 'Ошибка на сервере. Попробуйте позже.';
  return 'Не удалось выполнить запрос. Попробуйте ещё раз.';
}

// Типизированный клиент по контракту docs/api/openapi.yaml: путь, метод,
// параметры и тело каждой операции проверяются против сгенерированного
// schema.d.ts (npm run api-gen). Вызов несуществующей ручки или опечатка в
// поле — ошибка `npm run typecheck`, а не 404 в проде.
const create = /** @type {typeof createClient<import('./schema').paths>} */ (createClient);

export const api = create({ baseUrl: apiBase });

api.use({
  onRequest({ request }) {
    for (const [name, value] of Object.entries(authHeader())) {
      request.headers.set(name, value);
    }
    logInfo('api.request', { url: request.url, method: request.method });
    return request;
  },
  onResponse({ request, response }) {
    const log = response.ok ? logInfo : logWarn;
    log('api.response', { url: request.url, method: request.method, status: response.status });
    return response;
  },
});

/**
 * unwrap приводит результат openapi-fetch к прежнему контракту api-модулей:
 * возвращает данные или бросает ApiError с безопасным русским сообщением.
 * @template T
 * @param {Promise<{ data?: T, error?: unknown, response: Response }>} call
 * @returns {Promise<T>}
 */
export async function unwrap(call) {
  let result;
  try {
    result = await call;
  } catch (err) {
    logWarn('api.network_error', { error: err instanceof Error ? err.message : String(err) });
    throw new ApiError('Нет соединения с сервером. Проверьте подключение к интернету.', 0);
  }
  const { data, error, response } = result;
  if (error !== undefined || !response.ok) {
    const message =
      error && typeof error === 'object' && 'error' in error && typeof error.error === 'string'
        ? error.error
        : statusMessage(response.status);
    throw new ApiError(message, response.status);
  }
  // 204 и пустые ответы: контракту фасадов достаточно пустого объекта.
  return data ?? /** @type {T} */ ({});
}
