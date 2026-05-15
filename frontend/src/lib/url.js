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

export function syncOrderURL(number) {
  const orderNumber = trimString(number);
  if (!orderNumber) return;
  const url = new URL(window.location.href);
  url.searchParams.set('order', orderNumber);
  window.history.replaceState({}, '', url);
}
