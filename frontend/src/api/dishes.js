import { apiRequest } from './client';

export function fetchDishes() {
  return apiRequest('/admin/dishes');
}

export function searchAvailableDishes(query) {
  return apiRequest(`/admin/dishes/available?q=${encodeURIComponent(query || '')}`);
}

export function reorderDishes(codes) {
  return apiRequest('/admin/dishes/reorder', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ codes }),
  });
}

export function createDish(dish) {
  return apiRequest('/admin/dishes', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(dish),
  });
}

export function updateDish(code, dish) {
  return apiRequest(`/admin/dishes/${encodeURIComponent(code)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(dish),
  });
}

export function deleteDish(code) {
  return apiRequest(`/admin/dishes/${encodeURIComponent(code)}`, { method: 'DELETE' });
}

export function createCategory(category) {
  return apiRequest('/admin/categories', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(category),
  });
}

export function updateCategory(id, category) {
  return apiRequest(`/admin/categories/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(category),
  });
}

export function deleteCategory(id) {
  return apiRequest(`/admin/categories/${id}`, { method: 'DELETE' });
}
