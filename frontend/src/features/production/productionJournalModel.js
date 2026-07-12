const UNCATEGORIZED_KEY = 'uncategorized';

function localDateKey(value) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return String(value || 'unknown');
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

function categoryKey(category) {
  return category?.id ? `category:${category.id}` : UNCATEGORIZED_KEY;
}

// Journal matrix: rows are creation dates, columns are order categories.
// A production batch is category-homogeneous by the selection domain rule.
export function buildProductionJournalMatrix(sheets = [], categories = []) {
  const categoryByKey = new Map();
  categories.forEach((category) => categoryByKey.set(categoryKey(category), category));
  sheets.forEach((sheet) => {
    if (sheet.category) categoryByKey.set(categoryKey(sheet.category), sheet.category);
  });

  const columns = [...categoryByKey.entries()]
    .map(([key, category]) => ({ key, category }))
    .sort((a, b) => (a.category.sort_order || 0) - (b.category.sort_order || 0));
  if (sheets.some((sheet) => !sheet.category)) columns.push({ key: UNCATEGORIZED_KEY, category: null });

  const rowsByDate = new Map();
  sheets.forEach((sheet) => {
    const dateKey = localDateKey(sheet.created_at);
    const row = rowsByDate.get(dateKey) || { dateKey, cells: {} };
    const key = categoryKey(sheet.category);
    row.cells[key] = [...(row.cells[key] || []), sheet];
    rowsByDate.set(dateKey, row);
  });

  const rows = [...rowsByDate.values()].sort((a, b) => b.dateKey.localeCompare(a.dateKey));
  rows.forEach((row) => {
    Object.values(row.cells).forEach((cell) => cell.sort((a, b) => String(b.created_at).localeCompare(String(a.created_at))));
  });
  return { columns, rows };
}

export function formatJournalDate(value) {
  const date = new Date(`${value}T00:00:00`);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat('ru-RU', { day: 'numeric', month: 'short', weekday: 'short' }).format(date);
}
