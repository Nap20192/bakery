const WINDOW_DAYS = 8;
const WINDOW_START_OFFSET = -1;
const WEEKDAYS = ['вс', 'пн', 'вт', 'ср', 'чт', 'пт', 'сб'];

function dateKey(date) {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

function addDays(date, days) {
  const next = new Date(date);
  next.setDate(next.getDate() + days);
  return next;
}

export function buildTableColumns(shift) {
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  const start = addDays(today, WINDOW_START_OFFSET + shift * (WINDOW_DAYS - 1));
  const todayKey = dateKey(today);
  const tomorrowKey = dateKey(addDays(today, 1));
  return Array.from({ length: WINDOW_DAYS }, (_, index) => {
    const date = addDays(start, index);
    const key = dateKey(date);
    return {
      key,
      label: `${String(date.getDate()).padStart(2, '0')}.${String(date.getMonth() + 1).padStart(2, '0')}`,
      weekday: WEEKDAYS[date.getDay()],
      tone: key === todayKey ? 'today' : key === tomorrowKey ? 'tomorrow' : '',
    };
  });
}

export function buildTableGroups(orders, columns, catalog, categories) {
  const columnKeys = new Set(columns.map((column) => column.key));
  const catalogIndex = new Map(catalog.map((dish, index) => [String(dish.name || '').toLowerCase().trim(), index]));
  const groups = new Map();

  for (const order of orders) {
    if (order.cancelled) continue;
    const date = String(order.fulfillment_date || '');
    if (!columnKeys.has(date)) continue;
    const categoryID = order.category?.id || 0;
    if (!groups.has(categoryID)) groups.set(categoryID, { category: order.category || null, byDish: new Map() });
    const { byDish } = groups.get(categoryID);
    for (const item of order.items || []) {
      const key = String(item.product_name || '').toLowerCase().trim();
      if (!byDish.has(key)) byDish.set(key, { key, name: item.product_name, cells: {}, total: 0 });
      const row = byDish.get(key);
      const ordered = Number(item.production_quantity || 0);
      // effective — фактический выход (декорированный факт отработки), для
      // незатронутых позиций равен заявке. Ячейка «отработана», только пока
      // КАЖДЫЙ заказ-вкладчик покрыт листом — тогда таблица фиксирует выход,
      // а маленькая дельта показывает ± относительно заказа.
      const effective = item.produced_quantity != null ? Number(item.produced_quantity) : ordered;
      const cell = row.cells[date] || { ordered: 0, effective: 0, produced: true };
      cell.ordered += ordered;
      cell.effective += effective;
      cell.produced = cell.produced && Boolean(order.production_sheet_id);
      row.cells[date] = cell;
      row.total += ordered;
    }
  }

  const categoryOrder = new Map(categories.map((category, index) => [category.id, index]));
  return [...groups.entries()]
    .sort(([aID], [bID]) => (categoryOrder.get(aID) ?? Number.MAX_SAFE_INTEGER) - (categoryOrder.get(bID) ?? Number.MAX_SAFE_INTEGER))
    .map(([, group]) => ({
      category: group.category,
      rows: [...group.byDish.values()].sort((a, b) => {
        const aIndex = catalogIndex.get(a.key) ?? Number.MAX_SAFE_INTEGER;
        const bIndex = catalogIndex.get(b.key) ?? Number.MAX_SAFE_INTEGER;
        return aIndex === bIndex ? a.name.localeCompare(b.name, 'ru') : aIndex - bIndex;
      }),
    }));
}
