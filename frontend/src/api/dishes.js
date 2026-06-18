import { apiRequest } from './client';

export function fetchDishes() {
  return apiRequest('/admin/dishes');
}

export function createDish(dish) {
  return apiRequest('/admin/dishes', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(dish),
  });
}

export function deleteDish(code) {
  return apiRequest(`/admin/dishes/${encodeURIComponent(code)}`, { method: 'DELETE' });
}
