import { apiRequest } from './client';

export function fetchMe() {
  return apiRequest('/me');
}

export function fetchCatalog() {
  return apiRequest('/catalog');
}

export function fetchCategories() {
  return apiRequest('/categories');
}

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
  if (filters.fulfillmentFrom) {
    params.set('fulfillment_from', filters.fulfillmentFrom);
  }
  if (filters.fulfillmentTo) {
    params.set('fulfillment_to', filters.fulfillmentTo);
  }
  if (filters.categoryID) {
    params.set('category_id', String(filters.categoryID));
  }
  return apiRequest(`/orders?${params.toString()}`);
}

export function fetchOrder(number) {
  return apiRequest(`/orders/${encodeURIComponent(number)}`);
}

export function createOrder(order) {
  return apiRequest('/orders', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(order),
  });
}

export function updateOrder(number, order) {
  return apiRequest(`/orders/${encodeURIComponent(number)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(order),
  });
}

export function setOrderFavorite(number, favorite) {
  return apiRequest(`/orders/${encodeURIComponent(number)}/favorite`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ favorite }),
  });
}

export function cancelOrder(number) {
  return apiRequest(`/orders/${encodeURIComponent(number)}/cancel`, {
    method: 'POST',
  });
}

export function restoreOrder(number) {
  return apiRequest(`/orders/${encodeURIComponent(number)}/restore`, {
    method: 'POST',
  });
}

// Журнал отработок: каждая отработка — отдельный документ, который можно
// открыть, изменить и удалить. Факт в заказах пересчитывается автоматически.
export function createProductionSheet(orders) {
  return apiRequest('/production', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ orders }),
  });
}

export function fetchProductionSheets() {
  return apiRequest('/production');
}

export function fetchProductionSheet(id) {
  return apiRequest(`/production/${id}`);
}

export function updateProductionSheet(id, orders) {
  return apiRequest(`/production/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ orders }),
  });
}

export function deleteProductionSheet(id) {
  return apiRequest(`/production/${id}`, { method: 'DELETE' });
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
