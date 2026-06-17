import { apiRequest } from './client';

export function fetchUsers() {
  return apiRequest('/users');
}

export function fetchAdminDepartments() {
  return apiRequest('/admin/departments');
}

export function createUser(user) {
  return apiRequest('/users', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(user),
  });
}

export function updateUser(id, patch) {
  return apiRequest(`/users/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(patch),
  });
}

export function deleteUser(id) {
  return apiRequest(`/users/${encodeURIComponent(id)}`, { method: 'DELETE' });
}
