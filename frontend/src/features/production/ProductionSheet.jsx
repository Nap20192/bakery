import { useMemo, useState } from 'react';
import { Button } from '../../ui/Button';
import { EmptyState } from '../../ui/EmptyState';
import { Icon } from '../../ui/Icon';
import { formatQuantity } from '../../lib/format';

const factInputClass =
  'h-9 w-20 rounded-md border border-stone-300 bg-white px-2 text-center text-sm tabular-nums outline-none focus:border-stone-900 focus:ring-2 focus:ring-stone-900/10';

// buildRows агрегирует позиции выбранных заказов по продукту: сколько
// заказано суммарно и по каждому заказу, и какой факт уже внесён.
function buildRows(orders) {
  const rows = [];
  const byKey = new Map();
  for (const order of orders) {
    for (const item of order.items || []) {
      const key = String(item.product_name || '').toLowerCase().trim();
      if (!byKey.has(key)) {
        const row = { key, name: item.product_name, ordered: 0, produced: 0, hasProduced: false, perOrder: [] };
        byKey.set(key, row);
        rows.push(row);
      }
      const row = byKey.get(key);
      const ordered = Number(item.production_quantity || 0);
      row.ordered += ordered;
      row.perOrder.push({
        number: order.number,
        shop: order.from_department?.name || order.location || order.number,
        ordered,
        produced: item.produced_quantity ?? null,
      });
      // В журнале хранятся только отклонения: nil = «испечено по заявке».
      row.produced += item.produced_quantity != null ? Number(item.produced_quantity) : ordered;
      if (item.produced_quantity != null) row.hasProduced = true;
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

// ProductionSheet — лист отработки: пекарь вводит факт по продуктам,
// расхождения разносятся по заявкам (пропорционально, с ручной правкой).
// Лист фиксирует партию: сохраняются ВСЕ заказы выбора (и без отклонений
// тоже) плюс отклонения факта. Один и тот же редактор используется при
// создании (со страницы выбранных заказов) и при правке в журнале.
// Быстрые обоснования отклонений — заполняют поле причины одним тапом.
const REASON_PRESETS = ['Подгорело', 'Упало', 'Брак', 'Не хватило теста', 'Испекли про запас'];

export function ProductionSheet({ orders, loading, onSave, onOpenJournal, submitLabel = 'Сохранить отработку' }) {
  const rows = useMemo(() => buildRows(orders), [orders]);
  const [facts, setFacts] = useState({});
  const [allocations, setAllocations] = useState({});
  const [reasons, setReasons] = useState({});

  if (!rows.length) {
    return <EmptyState compact>В выбранных заказах нет позиций.</EmptyState>;
  }

  const hasAnyProduced = rows.some((row) => row.hasProduced);

  function factOf(row) {
    const raw = facts[row.key];
    if (raw !== undefined) return raw === '' ? NaN : Number(raw);
    return row.hasProduced ? row.produced : row.ordered;
  }

  function allocationOf(row) {
    const fact = factOf(row);
    const stored = allocations[row.key];
    if (stored && stored.fact === fact) return stored.values;
    if (row.hasProduced && row.produced === fact) {
      return row.perOrder.map((entry) => (entry.produced != null ? Number(entry.produced) : entry.ordered));
    }
    return prorate(Number.isFinite(fact) ? fact : 0, row.perOrder);
  }

  function setFact(row, value) {
    setFacts((current) => ({ ...current, [row.key]: value.replace(/\D/g, '') }));
  }

  function setAllocation(row, index, value) {
    const fact = factOf(row);
    const values = [...allocationOf(row)];
    values[index] = value === '' ? 0 : Number(value.replace(/\D/g, '') || 0);
    setAllocations((current) => ({ ...current, [row.key]: { fact, values } }));
  }

  function rowState(row) {
    const fact = factOf(row);
    if (!Number.isFinite(fact)) return { valid: false, mismatch: false };
    const mismatch = fact !== row.ordered;
    const values = allocationOf(row);
    const sum = values.reduce((total, value) => total + value, 0);
    return { valid: !mismatch || sum === fact, mismatch, fact, values, sum };
  }

  const states = rows.map((row) => ({ row, ...rowState(row) }));
  const canSave = !loading && states.every((state) => state.valid);

  function save() {
    // Партия сохраняется целиком: каждый выбранный заказ попадает в документ,
    // items — только отклонения (факт, совпавший с заявкой, не хранится).
    const perOrderItems = new Map(orders.map((order) => [order.number, []]));
    for (const { row, mismatch, values } of states) {
      if (!mismatch) continue;
      const reason = (reasons[row.key] || '').trim();
      row.perOrder.forEach((entry, index) => {
        if (values[index] === entry.ordered) return;
        perOrderItems.get(entry.number)?.push({ product_name: row.name, produced_quantity: values[index], reason });
      });
    }
    onSave([...perOrderItems.entries()].map(([number, items]) => ({ number, items })));
  }

  return (
    <div className="space-y-2">
      <div className="grid grid-cols-[minmax(0,1fr)_4.5rem_5rem] items-center gap-2 px-1 text-[11px] font-medium uppercase text-stone-400">
        <span>Продукт</span>
        <span className="text-center">Заказано</span>
        <span className="text-center">Испечено</span>
      </div>
      <div className="divide-y divide-stone-100">
        {states.map(({ row, mismatch, valid, fact, values, sum }) => (
          <div className="py-1.5" key={row.key}>
            <div className="grid grid-cols-[minmax(0,1fr)_4.5rem_5rem] items-center gap-2">
              <span className="min-w-0 break-words text-[13px] leading-5 text-stone-800">{row.name}</span>
              <span className="text-center text-[13px] font-medium tabular-nums text-stone-600">{formatQuantity(row.ordered)}</span>
              <input
                className={`${factInputClass} ${mismatch ? 'border-amber-400 bg-amber-50' : ''}`}
                type="text"
                inputMode="numeric"
                aria-label={`${row.name}: испечено`}
                value={facts[row.key] ?? String(factOf(row))}
                onChange={(event) => setFact(row, event.target.value)}
              />
            </div>
            {mismatch && (
              <div className="mt-1.5 space-y-1 rounded-md bg-stone-50 p-2">
                <p className="m-0 text-[11px] font-medium uppercase text-stone-400">Разнос по заявкам</p>
                {row.perOrder.map((entry, index) => (
                  <div className="grid grid-cols-[minmax(0,1fr)_4.5rem_5rem] items-center gap-2" key={entry.number}>
                    <span className="min-w-0 truncate text-[12px] leading-5 text-stone-600" title={entry.number}>
                      {entry.number}
                    </span>
                    <span className="text-center text-[12px] tabular-nums text-stone-500">{formatQuantity(entry.ordered)}</span>
                    <input
                      className={factInputClass}
                      type="text"
                      inputMode="numeric"
                      aria-label={`${row.name} ${entry.number}: факт`}
                      value={String(values[index] ?? 0)}
                      onChange={(event) => setAllocation(row, index, event.target.value)}
                    />
                  </div>
                ))}
                {!valid && (
                  <p className="m-0 text-[12px] leading-5 text-red-700">
                    Разнесено {formatQuantity(sum || 0)} из {formatQuantity(fact || 0)} — суммы должны совпасть.
                  </p>
                )}
                <div className="border-t border-stone-200 pt-1.5">
                  <p className="m-0 mb-1 text-[11px] font-medium uppercase text-stone-400">Почему? (необязательно)</p>
                  <div className="mb-1 flex flex-wrap gap-1">
                    {REASON_PRESETS.map((preset) => (
                      <button
                        key={preset}
                        type="button"
                        onClick={() =>
                          setReasons((current) => ({
                            ...current,
                            [row.key]: current[row.key] === preset ? '' : preset,
                          }))
                        }
                        className={`rounded-full border px-2 py-0.5 text-[12px] leading-5 transition focus:outline-none focus:ring-2 focus:ring-stone-900/20 ${
                          reasons[row.key] === preset
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
                    value={reasons[row.key] || ''}
                    onChange={(event) => setReasons((current) => ({ ...current, [row.key]: event.target.value }))}
                  />
                </div>
              </div>
            )}
          </div>
        ))}
      </div>
      <div className="flex flex-wrap items-center justify-between gap-2 border-t border-stone-200 pt-2">
        {hasAnyProduced && onOpenJournal ? (
          <button
            type="button"
            onClick={onOpenJournal}
            className="text-[12px] leading-5 text-stone-500 underline decoration-stone-300 underline-offset-2 hover:text-stone-800"
          >
            Изменить или удалить — в журнале отработок
          </button>
        ) : (
          <span className="text-[12px] leading-5 text-stone-400">Лист сохраняет выбранные заказы; отклонения — только там, где факт отличается.</span>
        )}
        <Button variant="primary" onClick={save} disabled={!canSave}>
          <Icon name="select" size={15} />
          {submitLabel}
        </Button>
      </div>
    </div>
  );
}
