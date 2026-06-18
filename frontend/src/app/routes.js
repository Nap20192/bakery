export function parseRoute(pathname = window.location.pathname) {
  const path = pathname.replace(/\/+$/, '') || '/';
  const parts = path.split('/').filter(Boolean).map(decodeURIComponent);

  if (path === '/') return { name: 'orders' };
  if (path === '/me') return { name: 'me' };
  if (path === '/admin/users') return { name: 'adminUsers' };
  if (parts[0] !== 'orders') return { name: 'orders' };
  if (parts.length === 1) return { name: 'orders' };
  if (parts[1] === 'new') return { name: 'orderNew' };
  if (parts[1] === 'selection') return { name: 'orderSelection' };
  if (parts.length === 2) return { name: 'orderView', number: parts[1] };
  if (parts[2] === 'edit') return { name: 'orderEdit', number: parts[1] };
  if (parts[2] === 'monitor') return { name: 'orderMonitor', number: parts[1] };
  return { name: 'orders' };
}

export function pathFor(route) {
  switch (route.name) {
    case 'me':
      return '/me';
    case 'adminUsers':
      return '/admin/users';
    case 'orderNew':
      return '/orders/new';
    case 'orderSelection':
      return '/orders/selection';
    case 'orderView':
      return `/orders/${encodeURIComponent(route.number)}`;
    case 'orderEdit':
      return `/orders/${encodeURIComponent(route.number)}/edit`;
    case 'orderMonitor':
      return `/orders/${encodeURIComponent(route.number)}/monitor`;
    case 'orders':
    default:
      return '/orders';
  }
}
