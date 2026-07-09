import { apiBase } from '../config/env';
import { authHeader } from '../lib/auth';
import { logInfo, logWarn } from '../lib/logger';
import { apiURL } from '../lib/url';

export class ApiError extends Error {
  constructor(message, status) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
  }
}

// statusMessage returns a human-readable fallback when the backend response
// carries no error text (proxy errors, unexpected failures).
function statusMessage(status) {
  if (status === 401) return 'Требуется вход в систему.';
  if (status === 403) return 'Недостаточно прав для этого действия.';
  if (status === 404) return 'Данные не найдены.';
  if (status === 409) return 'Данные изменились. Обновите страницу и попробуйте снова.';
  if (status >= 500) return 'Ошибка на сервере. Попробуйте позже.';
  return 'Не удалось выполнить запрос. Попробуйте ещё раз.';
}

export async function apiRequest(path, options = {}) {
  const url = apiURL(apiBase, path);
  const started = performance.now();
  logInfo('api.request', {
    path,
    url,
    api_base: apiBase,
    page_origin: window.location.origin,
  });

  let response;
  try {
    response = await fetch(url, {
      ...options,
      headers: {
        ...(options.headers || {}),
        ...authHeader(),
      },
    });
  } catch (err) {
    logWarn('api.network_error', {
      path,
      url,
      api_base: apiBase,
      duration_ms: Math.round(performance.now() - started),
      error: err instanceof Error ? err.message : String(err),
    });
    throw new ApiError('Нет соединения с сервером. Проверьте подключение к интернету.', 0);
  }

  const contentType = response.headers.get('content-type') || '';
  const payload = contentType.includes('application/json') ? await response.json().catch(() => ({})) : {};
  if (!response.ok) {
    logWarn('api.error', {
      path,
      url,
      api_base: apiBase,
      status: response.status,
      content_type: contentType,
      duration_ms: Math.round(performance.now() - started),
      error: payload.error || `HTTP ${response.status}`,
    });
    throw new ApiError(payload.error || statusMessage(response.status), response.status);
  }
  if (response.status === 204 || contentType === '') {
    return {};
  }
  if (!contentType.includes('application/json')) {
    logWarn('api.invalid_content_type', {
      path,
      url,
      api_base: apiBase,
      status: response.status,
      content_type: contentType,
      duration_ms: Math.round(performance.now() - started),
    });
    throw new ApiError('Сервер вернул некорректный ответ. Попробуйте позже.', response.status);
  }

  logInfo('api.response', {
    path,
    url,
    status: response.status,
    content_type: contentType,
    duration_ms: Math.round(performance.now() - started),
  });
  return payload;
}
