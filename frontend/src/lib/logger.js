import { frontendLogsEnabled } from '../config/env';

function log(method, event, payload) {
  if (!frontendLogsEnabled) return;
  method(`[bakery-ui] ${event}`, {
    at: new Date().toISOString(),
    ...payload,
  });
}

export function logInfo(event, payload = {}) {
  log(console.info, event, payload);
}

export function logWarn(event, payload = {}) {
  log(console.warn, event, payload);
}
