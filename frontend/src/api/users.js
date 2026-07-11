// @ts-check
import { api, unwrap } from './client';

/** @typedef {import('./schema').components['schemas']} Schemas */

export function fetchUsers() {
  return unwrap(api.GET('/users'));
}

/** @param {Schemas['UserCreate']} user */
export function createUser(user) {
  return unwrap(api.POST('/users', { body: user }));
}

/**
 * @param {number} id
 * @param {Schemas['UserUpdate']} patch
 */
export function updateUser(id, patch) {
  return unwrap(api.PATCH('/users/{id}', { params: { path: { id } }, body: patch }));
}

/** @param {number} id */
export function deleteUser(id) {
  return unwrap(api.DELETE('/users/{id}', { params: { path: { id } } }));
}
