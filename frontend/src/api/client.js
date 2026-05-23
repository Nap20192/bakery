import { apiBase } from '../config/env';
import { logInfo, logWarn } from '../lib/logger';
import { apiURL } from '../lib/url';

export async function apiRequest(path) {
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
    response = await fetch(url);
  } catch (err) {
    logWarn('api.network_error', {
      path,
      url,
      api_base: apiBase,
      duration_ms: Math.round(performance.now() - started),
      error: err instanceof Error ? err.message : String(err),
    });
    throw err;
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
    throw new Error(payload.error || `HTTP ${response.status}`);
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
    throw new Error('API returned non-JSON response. Check Vite proxy or VITE_API_BASE_URL.');
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
