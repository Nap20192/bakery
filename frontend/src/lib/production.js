// Цветовые метки отработок: заказы одной партии получают общий бейдж
// «№ листа» одного цвета — партия видна в матрице с одного взгляда.
// Цвет детерминирован по id листа (id % палитра), поэтому одинаков в
// матрице, деталях заказа и журнале без какой-либо синхронизации.
// Палитра сознательно не пересекается с семантикой данных: цвета типов
// заявок (amber, sky, violet, emerald, rose, stone) и подсветкой
// сегодня/завтра (emerald/amber). Классы литеральные — Tailwind JIT.
const SHEET_COLORS = [
  { badge: 'border-teal-300 bg-teal-100 text-teal-800', dot: 'bg-teal-500' },
  { badge: 'border-indigo-300 bg-indigo-100 text-indigo-800', dot: 'bg-indigo-500' },
  { badge: 'border-pink-300 bg-pink-100 text-pink-800', dot: 'bg-pink-500' },
  { badge: 'border-lime-300 bg-lime-100 text-lime-800', dot: 'bg-lime-500' },
  { badge: 'border-purple-300 bg-purple-100 text-purple-800', dot: 'bg-purple-500' },
  { badge: 'border-cyan-300 bg-cyan-100 text-cyan-800', dot: 'bg-cyan-500' },
  { badge: 'border-fuchsia-300 bg-fuchsia-100 text-fuchsia-800', dot: 'bg-fuchsia-500' },
  { badge: 'border-yellow-300 bg-yellow-100 text-yellow-800', dot: 'bg-yellow-500' },
];

export function sheetStyle(sheetId) {
  const id = Math.abs(Number(sheetId) || 0);
  return SHEET_COLORS[id % SHEET_COLORS.length];
}

// В листе хранятся только отклонения. Сам production_sheet_id означает,
// что партия зафиксирована, даже когда у всех позиций факт равен заявке.
export function productionDeviations(order) {
  if (!order?.production_sheet_id) return 0;
  return (order.items || []).filter((item) => item.produced_quantity != null).length;
}

export function productionStatus(order) {
  if (!order?.production_sheet_id) return null;
  const deviations = productionDeviations(order);
  return {
    deviations,
    label: deviations > 0 ? `Отработан с отклонениями: ${deviations}` : 'Отработан по заявке',
  };
}
