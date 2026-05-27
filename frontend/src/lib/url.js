export function trimString(value) {
  return String(value || '').trim();
}

export function apiURL(base, path) {
  return `${base.replace(/\/$/, '')}${path}`;
}

export function orderNumberFromLocation() {
  const params = new URLSearchParams(window.location.search);
  return trimString(params.get('order') || params.get('order_number') || params.get('number'));
}

export function miniAppModeFromLocation() {
  const params = new URLSearchParams(window.location.search);
  return trimString(params.get('mode'));
}

export function orderNumbersFromLocation() {
  const params = new URLSearchParams(window.location.search);
  return params.getAll('orders').map(trimString).filter(Boolean);
}

export function syncOrderURL(number, mode = 'view') {
  const orderNumber = trimString(number);
  if (!orderNumber) return;
  const url = new URL(window.location.href);
  url.searchParams.set('order', orderNumber);
  url.searchParams.set('mode', mode);
  url.searchParams.delete('orders');
  window.history.replaceState({}, '', url);
}
