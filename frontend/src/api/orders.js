import { apiRequest } from './client';

export function fetchOrders(page, limit) {
  return apiRequest(`/orders?page=${page}&limit=${limit}`);
}

export function fetchOrder(number) {
  return apiRequest(`/orders/${encodeURIComponent(number)}`);
}

export function fetchOrderMonitor(number) {
  return apiRequest(`/monitor/${encodeURIComponent(number)}`);
}
