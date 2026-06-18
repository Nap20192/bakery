import { useMemo, useState } from 'react';
import { Button } from '../../components/Button';
import { EmptyState } from '../../components/EmptyState';
import { PanelHeader } from '../../components/Panel';
import { formatQuantity } from '../../lib/format';

const controlClass =
  'h-11 w-full rounded-lg border border-stone-300 bg-white px-3 text-base text-stone-900 outline-none transition focus:border-stone-900 focus:ring-2 focus:ring-stone-900/10';
const qtyClass =
  'h-10 w-full rounded-lg border border-stone-300 bg-white px-1 text-center text-base text-stone-900 outline-none transition focus:border-stone-900 focus:ring-2 focus:ring-stone-900/10';

function todayValue() {
  const date = new Date();
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

function initialQuantities(order) {
  const quantities = {};
  for (const item of order?.items || []) {
    quantities[item.product_name] = {
      quantity: String(item.quantity || ''),
      reserved_quantity: String(item.reserved_quantity || ''),
    };
  }
  return quantities;
}

export function OrderEditor({ catalog, order, loading, onCancel, onSave }) {
  const [date, setDate] = useState(order?.fulfillment_date || '');
  const [quantities, setQuantities] = useState(() => initialQuantities(order));
  const [search, setSearch] = useState('');

  const groups = useMemo(() => {
    const result = [];
    const byTheme = new Map();
    for (const item of catalog) {
      if (!byTheme.has(item.theme)) {
        const group = { theme: item.theme || 'Без группы', items: [] };
        byTheme.set(item.theme, group);
        result.push(group);
      }
      byTheme.get(item.theme).items.push(item);
    }
    return result;
  }, [catalog]);

  const query = search.trim().toLowerCase();
  const visibleGroups = useMemo(() => {
    if (!query) return groups;
    return groups
      .map((group) => ({ ...group, items: group.items.filter((item) => item.name.toLowerCase().includes(query)) }))
      .filter((group) => group.items.length);
  }, [groups, query]);

  const summary = useMemo(() => {
    let count = 0;
    let total = 0;
    for (const value of Object.values(quantities)) {
      const sum = Number(value.quantity || 0) + Number(value.reserved_quantity || 0);
      if (sum > 0) {
        count += 1;
        total += sum;
      }
    }
    return { count, total };
  }, [quantities]);

  function updateQuantity(name, key, value) {
    setQuantities((current) => ({
      ...current,
      [name]: { ...current[name], [key]: value },
    }));
  }

  function submit(event) {
    event.preventDefault();
    const items = catalog
      .map((item) => {
        const values = quantities[item.name] || {};
        return {
          product_name: item.name,
          quantity: Number(values.quantity || 0),
          reserved_quantity: Number(values.reserved_quantity || 0),
        };
      })
      .filter((item) => item.quantity + item.reserved_quantity > 0);
    if (!items.length) return;
    onSave({ fulfillment_date: date, items });
  }

  const canSubmit = !loading && summary.count > 0 && Boolean(date);

  return (
    <form className="space-y-3" onSubmit={submit}>
      <PanelHeader title={order ? `Изменить ${order.number}` : 'Новый заказ'} />

      <div className="grid gap-2 sm:grid-cols-2">
        <label className="block">
          <span className="mb-1 block text-[12px] font-medium text-stone-500">Дата выполнения</span>
          <input className={controlClass} type="date" value={date} min={todayValue()} onChange={(e) => setDate(e.target.value)} required />
        </label>
        <label className="block">
          <span className="mb-1 block text-[12px] font-medium text-stone-500">Поиск блюда</span>
          <input className={controlClass} type="search" value={search} onChange={(e) => setSearch(e.target.value)} placeholder="название…" />
        </label>
      </div>

      <div className="space-y-2.5">
        {visibleGroups.length ? (
          visibleGroups.map((group) => (
            <section key={group.theme}>
              <div className="grid grid-cols-[minmax(0,1fr)_3.5rem_3.5rem] items-center gap-2 px-1 pb-1">
                <h4 className="m-0 truncate text-[12px] font-semibold uppercase tracking-wide text-stone-500">{group.theme}</h4>
                <span className="text-center text-[10px] font-medium uppercase text-stone-400">Кол-во</span>
                <span className="text-center text-[10px] font-medium uppercase text-stone-400">Заказ</span>
              </div>
              <div className="overflow-hidden rounded-lg border border-stone-200 bg-white">
                <div className="divide-y divide-stone-100">
                  {group.items.map((item) => {
                    const value = quantities[item.name] || {};
                    const filled = Number(value.quantity || 0) + Number(value.reserved_quantity || 0) > 0;
                    return (
                      <div
                        className={`grid grid-cols-[minmax(0,1fr)_3.5rem_3.5rem] items-center gap-2 px-2.5 py-1.5 transition ${filled ? 'bg-stone-50' : ''}`}
                        key={item.name}
                      >
                        <span className="min-w-0 break-words text-[14px] leading-5 text-stone-800">{item.name}</span>
                        <input
                          className={qtyClass}
                          type="number"
                          inputMode="numeric"
                          min={0}
                          step={1}
                          placeholder="0"
                          aria-label={`${item.name}: количество`}
                          value={value.quantity || ''}
                          onChange={(e) => updateQuantity(item.name, 'quantity', e.target.value)}
                        />
                        <input
                          className={qtyClass}
                          type="number"
                          inputMode="numeric"
                          min={0}
                          step={1}
                          placeholder="0"
                          aria-label={`${item.name}: заказано`}
                          value={value.reserved_quantity || ''}
                          onChange={(e) => updateQuantity(item.name, 'reserved_quantity', e.target.value)}
                        />
                      </div>
                    );
                  })}
                </div>
              </div>
            </section>
          ))
        ) : (
          <EmptyState compact>{query ? 'Ничего не найдено.' : 'Блюда не найдены.'}</EmptyState>
        )}
      </div>

      <div className="sticky bottom-16 -mx-3 flex items-center justify-between gap-2 border-t border-stone-200 bg-white/95 px-3 py-2.5 backdrop-blur sm:bottom-0 sm:-mx-4 sm:px-4">
        <span className="text-[13px] text-stone-600">
          <strong className="text-stone-900 tabular-nums">{summary.count}</strong> поз.
          <span className="mx-1.5 text-stone-300">·</span>
          <strong className="text-stone-900 tabular-nums">{formatQuantity(summary.total)}</strong> шт.
        </span>
        <div className="flex gap-2">
          <Button type="button" onClick={onCancel}>Отмена</Button>
          <Button type="submit" variant="primary" disabled={!canSubmit}>
            {order ? 'Сохранить' : 'Создать'}
          </Button>
        </div>
      </div>
    </form>
  );
}
