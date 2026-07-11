import { useEffect, useMemo, useState } from 'react';
import { Button } from '../../ui/Button';
import { EmptyState } from '../../ui/EmptyState';
import { Icon } from '../../ui/Icon';
import { formatQuantity } from '../../lib/format';

const quantityInputClass =
  'h-9 w-full min-w-0 rounded-md border border-stone-300 bg-white px-1.5 text-center text-sm tabular-nums outline-none focus:border-stone-900 focus:ring-2 focus:ring-stone-900/10';

// buildRows агрегирует позиции выбранных заказов по продукту: сколько
// заказано суммарно и по каждому заказу, и какой факт уже внесён.
function buildRows(orders, sheetItems) {
  const savedByOrder = new Map(
    (sheetItems || []).map((item) => [`${item.order_number}\u0000${String(item.product_name || '').toLowerCase().trim()}`, item]),
  );
  const rows = [];
  const byKey = new Map();
  for (const order of orders) {
    for (const item of order.items || []) {
      const key = String(item.product_name || '').toLowerCase().trim();
      if (!byKey.has(key)) {
        const row = { key, name: item.product_name, ordered: 0, loaded: 0, produced: 0, hasSaved: false, reason: '', perOrder: [] };
        byKey.set(key, row);
        rows.push(row);
      }
      const row = byKey.get(key);
      const ordered = Number(item.production_quantity || 0);
      const saved = savedByOrder.get(`${order.number}\u0000${key}`);
      const loaded = saved?.loaded_quantity != null ? Number(saved.loaded_quantity) : ordered;
      const produced = item.produced_quantity != null ? Number(item.produced_quantity) : ordered;
      row.ordered += ordered;
      row.loaded += loaded;
      row.perOrder.push({
        number: order.number,
        shop: order.from_department?.name || order.location || order.number,
        ordered,
        loaded,
        produced,
      });
      // В журнале хранятся только отклонения: nil = «испечено по заявке».
      row.produced += produced;
      if (saved) row.hasSaved = true;
      if (!row.reason && item.produced_reason) row.reason = item.produced_reason;
    }
  }
  return rows;
}

// prorate раскидывает факт по заказам пропорционально заявке (метод крупных
// остатков — сумма всегда сходится, значения целые).
function prorate(fact, perOrder) {
  const totalOrdered = perOrder.reduce((sum, entry) => sum + entry.ordered, 0);
  if (totalOrdered <= 0) {
    const even = Math.floor(fact / perOrder.length);
    return perOrder.map((_, index) => (index === 0 ? fact - even * (perOrder.length - 1) : even));
  }
  const exact = perOrder.map((entry) => (fact * entry.ordered) / totalOrdered);
  const floors = exact.map(Math.floor);
  let rest = fact - floors.reduce((sum, value) => sum + value, 0);
  const order = exact
    .map((value, index) => ({ index, frac: value - Math.floor(value) }))
    .sort((a, b) => b.frac - a.frac);
  for (const { index } of order) {
    if (rest <= 0) break;
    floors[index] += 1;
    rest -= 1;
  }
  return floors;
}

// ProductionSheet — лист отработки: пекарь вводит агрегаты по продуктам,
// а значения автоматически разносятся по заявкам пропорционально заказу.
// Лист фиксирует партию: сохраняются ВСЕ заказы выбора (и без отклонений
// тоже) плюс отклонения факта. Один и тот же редактор используется при
// создании (со страницы выбранных заказов) и при правке в журнале.
// Быстрые обоснования отклонений — заполняют поле причины одним тапом.
const REASON_PRESETS = ['Подгорело', 'Упало', 'Брак', 'Не хватило теста', 'Испекли про запас'];

export function ProductionSheet({ orders, sheetItems = [], loading, onSave, onOpenJournal, onDirtyChange, submitLabel = 'Сохранить отработку' }) {
  const rows = useMemo(() => buildRows(orders, sheetItems), [orders, sheetItems]);
  const [loads, setLoads] = useState({});
  const [outputs, setOutputs] = useState({});
  const [reasons, setReasons] = useState({});
  const [openComments, setOpenComments] = useState({});

  // Несохранённые правки листа: пока они есть, расчёт теста заблокирован
  // (docs/constraints.md) — расчёт всегда идёт по сохранённому факту.
  const dirty = Object.keys(loads).length > 0 || Object.keys(outputs).length > 0 || Object.keys(reasons).length > 0;
  useEffect(() => {
    onDirtyChange?.(dirty);
  }, [dirty, onDirtyChange]);

  if (!rows.length) {
    return <EmptyState compact>В выбранных заказах нет позиций.</EmptyState>;
  }

  // «Отменить» возвращает редактор к последнему сохранённому состоянию.
  function revert() {
    setLoads({});
    setOutputs({});
    setReasons({});
    setOpenComments({});
  }

  const hasSavedSheet = rows.some((row) => row.hasSaved);

  function quantityOf(row, values, fallback) {
    const raw = values[row.key];
    if (raw !== undefined) return raw === '' ? NaN : Number(raw);
    return fallback;
  }

  function allocationOf(row, total, field) {
    if (row.hasSaved && row[field] === total) {
      return row.perOrder.map((entry) => Number(entry[field]));
    }
    return prorate(Number.isFinite(total) ? total : 0, row.perOrder);
  }

  function setQuantity(setter, row, value) {
    setter((current) => ({ ...current, [row.key]: value.replace(/\D/g, '') }));
  }

  function rowState(row) {
    const loaded = quantityOf(row, loads, row.loaded);
    // На новом листе выход следует за закладкой. В сохранённом листе связь
    // остаётся только когда все исходные значения равны; любое сохранённое
    // отклонение считается осознанным и никогда автоматически не затирается.
    const savedValuesAreDefault = row.loaded === row.ordered && row.produced === row.loaded;
    const outputFollowsLoad = outputs[row.key] === undefined && (!row.hasSaved || savedValuesAreDefault);
    const output = outputFollowsLoad ? loaded : quantityOf(row, outputs, row.produced);
    if (!Number.isFinite(loaded) || !Number.isFinite(output)) return { valid: false, expanded: false };
    const loadValues = allocationOf(row, loaded, 'loaded');
    const outputValues = outputFollowsLoad
      ? loadValues
      : allocationOf(row, output, 'produced');
    return {
      valid: true,
      loaded,
      output,
      outputFollowsLoad,
      loadValues,
      outputValues,
    };
  }

  const states = rows.map((row) => ({ row, ...rowState(row) }));
  const canSave = !loading && states.every((state) => state.valid);

  function save() {
    // Партия сохраняется целиком: каждый выбранный заказ попадает в документ,
    // items — только отклонения (факт, совпавший с заявкой, не хранится).
    const perOrderItems = new Map(orders.map((order) => [order.number, []]));
    for (const { row, loadValues, outputValues } of states) {
      const reason = (reasons[row.key] ?? row.reason).trim();
      row.perOrder.forEach((entry, index) => {
        perOrderItems.get(entry.number)?.push({
          product_name: row.name,
          loaded_quantity: loadValues[index],
          produced_quantity: outputValues[index],
          reason: outputValues[index] === entry.ordered ? '' : reason,
        });
      });
    }
    onSave([...perOrderItems.entries()].map(([number, items]) => ({ number, items })));
  }

  return (
    <div className="space-y-2">
      <div className="grid grid-cols-[minmax(5.5rem,1fr)_3.5rem_4.25rem_4.25rem] items-center gap-1.5 px-1 text-caption font-medium uppercase text-stone-500 sm:grid-cols-[minmax(0,1fr)_4.5rem_5rem_5rem] sm:gap-2 sm:text-caption">
        <span>Продукт</span>
        <span className="text-center">Заказ</span>
        <span className="text-center">Закладка</span>
        <span className="text-center">Выход</span>
      </div>
      <div className="divide-y divide-stone-100">
        {states.map(({ row, loaded, output, outputFollowsLoad }) => {
          const reason = reasons[row.key] ?? row.reason;
          const commentOpen = Boolean(openComments[row.key] || reason);
          return (
          <div className="py-1.5" key={row.key}>
            <div className="grid grid-cols-[minmax(5.5rem,1fr)_3.5rem_4.25rem_4.25rem] items-center gap-1.5 sm:grid-cols-[minmax(0,1fr)_4.5rem_5rem_5rem] sm:gap-2">
              <span className="flex min-w-0 items-center gap-1.5">
                <span className="min-w-0 flex-1 break-words text-body leading-5 text-stone-800">{row.name}</span>
                <button
                  type="button"
                  onClick={() => setOpenComments((current) => ({ ...current, [row.key]: !commentOpen }))}
                  title="Комментарий"
                  aria-label={`${row.name}: комментарий`}
                  className={`inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-md border transition ${
                    reason
                      ? 'border-stone-900 bg-stone-900 text-white'
                      : commentOpen
                        ? 'border-stone-400 text-stone-700'
                        : 'border-stone-200 text-stone-400 hover:border-stone-400 hover:text-stone-700'
                  }`}
                >
                  <Icon name="orders" size={14} />
                </button>
              </span>
              <span className="text-center text-body font-medium tabular-nums text-stone-600">{formatQuantity(row.ordered)}</span>
              <input
                className={`${quantityInputClass} ${loaded !== row.ordered ? 'border-sky-400 bg-sky-50' : ''}`}
                type="text"
                inputMode="numeric"
                aria-label={`${row.name}: закладка`}
                value={loads[row.key] ?? String(row.loaded)}
                onChange={(event) => setQuantity(setLoads, row, event.target.value)}
              />
              <input
                className={`${quantityInputClass} ${output !== row.ordered ? 'border-amber-400 bg-amber-50' : ''}`}
                type="text"
                inputMode="numeric"
                aria-label={`${row.name}: выход`}
                value={outputs[row.key] ?? String(outputFollowsLoad ? loaded : row.produced)}
                onChange={(event) => setQuantity(setOutputs, row, event.target.value)}
              />
            </div>
            {commentOpen && (
              <div className="fade-in mt-1.5 rounded-md border border-stone-200 bg-stone-50 p-2">
                  <p className="m-0 mb-1 text-caption font-medium uppercase text-stone-400">Комментарий к выходу (необязательно)</p>
                  <div className="mb-1 flex flex-wrap gap-1">
                    {REASON_PRESETS.map((preset) => (
                      <button
                        key={preset}
                        type="button"
                        onClick={() =>
                          setReasons((current) => ({
                            ...current,
                            [row.key]: (current[row.key] ?? row.reason) === preset ? '' : preset,
                          }))
                        }
                        className={`rounded-full border px-2 py-0.5 text-note leading-5 transition focus:outline-none focus:ring-2 focus:ring-stone-900/20 ${
                          reason === preset
                            ? 'border-stone-900 bg-stone-900 text-white'
                            : 'border-stone-300 bg-white text-stone-600 hover:border-stone-400'
                        }`}
                      >
                        {preset}
                      </button>
                    ))}
                  </div>
                  <input
                    className="h-9 w-full rounded-md border border-stone-300 bg-white px-2 text-sm outline-none focus:border-stone-900 focus:ring-2 focus:ring-stone-900/10"
                    type="text"
                    maxLength={200}
                    placeholder="Своя причина…"
                    aria-label={`${row.name}: причина отклонения`}
                    value={reason}
                    onChange={(event) => setReasons((current) => ({ ...current, [row.key]: event.target.value }))}
                  />
              </div>
            )}
          </div>
          );
        })}
      </div>
      <div className="flex flex-wrap items-center justify-between gap-2 border-t border-stone-200 pt-2">
        {hasSavedSheet && onOpenJournal ? (
          <button
            type="button"
            onClick={onOpenJournal}
            className="text-note leading-5 text-stone-500 underline decoration-stone-300 underline-offset-2 hover:text-stone-800"
          >
            Изменить или удалить — в журнале отработок
          </button>
        ) : (
          <span className="text-note leading-5 text-stone-400">Лист сохраняет выбранные заказы; отклонения — только там, где факт отличается.</span>
        )}
        <div className="flex shrink-0 gap-2">
          {dirty && (
            <Button onClick={revert} disabled={loading} aria-label="Отменить несохранённые правки">
              <Icon name="close" size={15} />
              Отменить
            </Button>
          )}
          <Button variant="primary" onClick={save} loading={loading} disabled={!canSave}>
            <Icon name="select" size={15} />
            {submitLabel}
          </Button>
        </div>
      </div>
    </div>
  );
}
