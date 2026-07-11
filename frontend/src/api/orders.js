// @ts-check
import { api, unwrap } from './client';

/** @typedef {import('./schema').components['schemas']} Schemas */

export function fetchMe() {
  return unwrap(api.GET('/me'));
}

export function fetchCatalog() {
  return unwrap(api.GET('/catalog'));
}

export function fetchCategories() {
  return unwrap(api.GET('/categories'));
}

/** @param {'shop' | 'workshop' | ''} [type] */
export function fetchDepartments(type = '') {
  return unwrap(api.GET('/departments', { params: { query: type ? { type } : {} } }));
}

/**
 * @param {number} page
 * @param {number} limit
 * @param {{ fromDepartmentID?: number, fulfillmentDate?: string, fulfillmentFrom?: string, fulfillmentTo?: string, categoryID?: number }} [filters]
 */
export function fetchOrders(page, limit, filters = {}) {
  /** @type {NonNullable<import('./schema').operations['listOrders']['parameters']['query']>} */
  const query = { page, limit };
  if (filters.fromDepartmentID) query.from_department_id = filters.fromDepartmentID;
  if (filters.fulfillmentDate) query.fulfillment_date = filters.fulfillmentDate;
  if (filters.fulfillmentFrom) query.fulfillment_from = filters.fulfillmentFrom;
  if (filters.fulfillmentTo) query.fulfillment_to = filters.fulfillmentTo;
  if (filters.categoryID) query.category_id = filters.categoryID;
  return unwrap(api.GET('/orders', { params: { query } }));
}

/** @param {string} number */
export function fetchOrder(number) {
  return unwrap(api.GET('/orders/{id}', { params: { path: { id: number } } }));
}

/** @param {Schemas['OrderWrite']} order */
export function createOrder(order) {
  return unwrap(api.POST('/orders', { body: order }));
}

/**
 * @param {string} number
 * @param {Schemas['OrderWrite']} order
 */
export function updateOrder(number, order) {
  return unwrap(api.PUT('/orders/{id}', { params: { path: { id: number } }, body: order }));
}

/**
 * @param {string} number
 * @param {boolean} favorite
 */
export function setOrderFavorite(number, favorite) {
  return unwrap(api.PATCH('/orders/{id}/favorite', { params: { path: { id: number } }, body: { favorite } }));
}

/** @param {string} number */
export function cancelOrder(number) {
  return unwrap(api.POST('/orders/{id}/cancel', { params: { path: { id: number } } }));
}

/** @param {string} number */
export function restoreOrder(number) {
  return unwrap(api.POST('/orders/{id}/restore', { params: { path: { id: number } } }));
}

// Журнал отработок: каждая отработка — отдельный документ, который можно
// открыть, изменить и удалить. Факт в заказах пересчитывается автоматически.
/** @param {Schemas['ProductionWrite']['orders']} orders */
export function createProductionSheet(orders) {
  return unwrap(api.POST('/production', { body: { orders } }));
}

export function fetchProductionSheets() {
  return unwrap(api.GET('/production'));
}

/** @param {number} id */
export function fetchProductionSheet(id) {
  return unwrap(api.GET('/production/{id}', { params: { path: { id } } }));
}

/**
 * @param {number} id
 * @param {Schemas['ProductionWrite']['orders']} orders
 */
export function updateProductionSheet(id, orders) {
  return unwrap(api.PUT('/production/{id}', { params: { path: { id } }, body: { orders } }));
}

/** @param {number} id */
export function deleteProductionSheet(id) {
  return unwrap(api.DELETE('/production/{id}', { params: { path: { id } } }));
}

/** @param {string} number */
export function fetchOrderMonitor(number) {
  return unwrap(api.GET('/monitor/{id}', { params: { path: { id: number } } }));
}

/**
 * Калькулятор теста: расчёт по введённым вручную позициям, ничего не создаёт.
 * @param {number} categoryId 0 = дефолтные коды теста
 * @param {Schemas['DoughCalcRequest']['items']} items
 */
export function calcDough(categoryId, items) {
  return unwrap(api.POST('/monitor/calc', { body: { category_id: categoryId, items } }));
}

/** @param {string[]} numbers */
export function fetchBatchOrderMonitor(numbers) {
  const orders = (numbers || []).filter(Boolean);
  return unwrap(api.GET('/monitor/batch', { params: { query: { orders } } }));
}
