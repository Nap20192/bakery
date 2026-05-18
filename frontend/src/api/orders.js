import { apiRequest } from './client';

export function fetchDepartments(type = '') {
  const params = new URLSearchParams();
  if (type) params.set('type', type);
  const query = params.toString();
  return apiRequest(`/departments${query ? `?${query}` : ''}`);
}

export function fetchOrders(page, limit, filters = {}) {
  const params = new URLSearchParams({
    page: String(page),
    limit: String(limit),
  });
  if (filters.fromDepartmentID) {
    params.set('from_department_id', String(filters.fromDepartmentID));
  }
  if (filters.fulfillmentDate) {
    params.set('fulfillment_date', filters.fulfillmentDate);
  }
  return apiRequest(`/orders?${params.toString()}`);
}

export function fetchOrder(number) {
  return apiRequest(`/orders/${encodeURIComponent(number)}`);
}

export function fetchOrderMonitor(number) {
  return apiRequest(`/monitor/${encodeURIComponent(number)}`);
}

export function fetchBatchOrderMonitor(numbers) {
  const params = new URLSearchParams();
  for (const number of numbers) {
    if (number) params.append('orders', number);
  }
  return apiRequest(`/monitor/batch?${params.toString()}`);
}
