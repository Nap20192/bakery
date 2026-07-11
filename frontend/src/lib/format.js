export function formatQuantity(value) {
  if (!Number.isFinite(value)) return '0';
  return new Intl.NumberFormat('ru-RU', { maximumFractionDigits: 3 }).format(value);
}

export function formatDate(value) {
  if (!value) return '';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat('ru-RU', { dateStyle: 'medium', timeStyle: 'short' }).format(date);
}

export function formatFulfillmentDate(value) {
  if (!value) return '';
  const date = new Date(`${value}T00:00:00`);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat('ru-RU', { dateStyle: 'medium' }).format(date);
}

// plural подбирает русскую форму слова под число: plural(3, 'заявка',
// 'заявки', 'заявок') → «заявки». Возвращает только слово, без числа.
export function plural(count, one, few, many) {
  const n = Math.abs(Math.trunc(count)) % 100;
  if (n >= 11 && n <= 14) return many;
  switch (n % 10) {
    case 1:
      return one;
    case 2:
    case 3:
    case 4:
      return few;
    default:
      return many;
  }
}
