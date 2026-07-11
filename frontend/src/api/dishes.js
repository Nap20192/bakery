// @ts-check
import { api, unwrap } from './client';

/** @typedef {import('./schema').components['schemas']} Schemas */

export function fetchDishes() {
  return unwrap(api.GET('/admin/dishes'));
}

/** @param {string} [query] */
export function searchAvailableDishes(query) {
  return unwrap(api.GET('/admin/dishes/available', { params: { query: { q: query || '' } } }));
}

/** @param {string[]} codes */
export function reorderDishes(codes) {
  return unwrap(api.PUT('/admin/dishes/reorder', { body: { codes } }));
}

/** @param {Schemas['DishWrite']} dish */
export function createDish(dish) {
  return unwrap(api.POST('/admin/dishes', { body: dish }));
}

/**
 * @param {string} code
 * @param {Schemas['DishWrite']} dish
 */
export function updateDish(code, dish) {
  return unwrap(api.PUT('/admin/dishes/{code}', { params: { path: { code } }, body: dish }));
}

/** @param {string} code */
export function deleteDish(code) {
  return unwrap(api.DELETE('/admin/dishes/{code}', { params: { path: { code } } }));
}

/** @param {Schemas['CategoryWrite']} category */
export function createCategory(category) {
  return unwrap(api.POST('/admin/categories', { body: category }));
}

/**
 * @param {number} id
 * @param {Schemas['CategoryWrite']} category
 */
export function updateCategory(id, category) {
  return unwrap(api.PUT('/admin/categories/{id}', { params: { path: { id } }, body: category }));
}

/** @param {number} id */
export function deleteCategory(id) {
  return unwrap(api.DELETE('/admin/categories/{id}', { params: { path: { id } } }));
}
