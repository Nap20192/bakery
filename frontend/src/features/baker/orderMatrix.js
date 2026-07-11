import { formatFulfillmentDate } from '../../lib/format';

// Pure matrix projection: columns are shops, bands are fulfillment dates and
// cells contain that shop's orders. Kept outside React for independent tests.
export function buildOrderMatrix(orders, shops) {
  const columns = shops.map((shop) => ({
    key: String(shop.id),
    name: shop.name.replace(/^Магазин\s+/i, ''),
  }));
  const known = new Set(columns.map((column) => column.key));
  if (orders.some((order) => !known.has(String(order.from_department?.id || '')))) {
    columns.push({ key: 'other', name: 'Прочее' });
  }

  const byDate = new Map();
  for (const order of orders) {
    const date = order.fulfillment_date || '';
    if (!byDate.has(date)) byDate.set(date, new Map());
    const cells = byDate.get(date);
    const departmentID = String(order.from_department?.id || '');
    const columnKey = known.has(departmentID) ? departmentID : 'other';
    if (!cells.has(columnKey)) cells.set(columnKey, []);
    cells.get(columnKey).push(order);
  }

  const dates = [...byDate.keys()].sort((a, b) => (a < b ? 1 : -1));
  const bands = dates.map((date) => {
    const cells = byDate.get(date);
    let total = 0;
    for (const list of cells.values()) total += list.length;
    const relative = relativeDay(date);
    return { key: date || 'none', label: bandLabel(date, relative), kind: relative.kind, cells, total };
  });
  return { columns, bands };
}

function bandLabel(date, relative) {
  if (!date) return 'Без даты';
  const formatted = formatFulfillmentDate(date);
  return relative.label ? `${relative.label} · ${formatted}` : formatted;
}

function relativeDay(date) {
  const today = new Date();
  const value = (offset) => {
    const target = new Date(today);
    target.setDate(target.getDate() + offset);
    const month = String(target.getMonth() + 1).padStart(2, '0');
    const day = String(target.getDate()).padStart(2, '0');
    return `${target.getFullYear()}-${month}-${day}`;
  };
  if (date === value(1)) return { kind: 'tomorrow', label: 'Завтра' };
  if (date === value(0)) return { kind: 'today', label: 'Сегодня' };
  if (date === value(-1)) return { kind: '', label: 'Вчера' };
  return { kind: '', label: '' };
}
