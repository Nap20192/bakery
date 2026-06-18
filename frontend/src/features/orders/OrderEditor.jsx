import { useMemo, useState } from 'react';
import TextField from '@mui/material/TextField';
import { Button } from '../../components/Button';
import { EmptyState } from '../../components/EmptyState';
import { PanelHeader } from '../../components/Panel';
import { formatQuantity } from '../../lib/format';

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
      <div className="flex flex-wrap items-start justify-between gap-2">
        <PanelHeader title={order ? `Изменить ${order.number}` : 'Новый заказ'} />
      </div>

      <div className="grid gap-2 sm:grid-cols-2">
        <TextField
          size="medium"
          label="Дата выполнения"
          type="date"
          value={date}
          onChange={(event) => setDate(event.target.value)}
          slotProps={{ inputLabel: { shrink: true }, htmlInput: { min: todayValue() } }}
          required
          fullWidth
        />
        <TextField
          size="medium"
          label="Поиск блюда"
          type="search"
          value={search}
          onChange={(event) => setSearch(event.target.value)}
          placeholder="начните вводить название…"
          fullWidth
        />
      </div>

      <div className="max-h-[58vh] space-y-3 overflow-y-auto pr-1">
        {visibleGroups.length ? (
          visibleGroups.map((group) => (
            <section className="overflow-hidden rounded-md border border-stone-200" key={group.theme}>
              <h4 className="m-0 bg-stone-50 px-3 py-2 text-[13px] font-semibold text-stone-900">{group.theme}</h4>
              <div className="divide-y divide-stone-200">
                {group.items.map((item) => {
                  const value = quantities[item.name] || {};
                  const filled = Number(value.quantity || 0) + Number(value.reserved_quantity || 0) > 0;
                  return (
                    <div
                      className={`grid gap-2 px-3 py-3 sm:grid-cols-[minmax(0,1fr)_5.4rem_5.4rem] sm:items-center sm:py-2 ${filled ? 'bg-stone-50/60' : ''}`}
                      key={item.name}
                    >
                      <span className="min-w-0 text-[15px] font-medium leading-5 text-stone-800 sm:text-[13px]">{item.name}</span>
                      <div className="grid grid-cols-2 gap-2 sm:contents">
                        <TextField
                          size="medium"
                          type="number"
                          label="Кол-во"
                          value={value.quantity || ''}
                          onChange={(event) => updateQuantity(item.name, 'quantity', event.target.value)}
                          slotProps={{ htmlInput: { min: 0, step: 1, inputMode: 'decimal' } }}
                        />
                        <TextField
                          size="medium"
                          type="number"
                          label="Заказ."
                          value={value.reserved_quantity || ''}
                          onChange={(event) => updateQuantity(item.name, 'reserved_quantity', event.target.value)}
                          slotProps={{ htmlInput: { min: 0, step: 1, inputMode: 'decimal' } }}
                        />
                      </div>
                    </div>
                  );
                })}
              </div>
            </section>
          ))
        ) : (
          <EmptyState compact>{query ? 'Ничего не найдено.' : 'Блюда не найдены.'}</EmptyState>
        )}
      </div>

      <div className="sticky bottom-0 flex flex-wrap items-center justify-between gap-2 border-t border-stone-200 bg-white py-3 shadow-[0_-8px_20px_rgba(28,25,23,0.06)]">
        <span className="text-[13px] text-stone-600">
          Позиций: <strong className="text-stone-900 tabular-nums">{summary.count}</strong>
          <span className="mx-1.5 text-stone-300">·</span>
          Всего: <strong className="text-stone-900 tabular-nums">{formatQuantity(summary.total)}</strong>
        </span>
        <div className="flex gap-2">
          <Button type="button" onClick={onCancel}>
            Отмена
          </Button>
          <Button type="submit" variant="primary" disabled={!canSubmit}>
            {order ? 'Сохранить изменения' : 'Создать заказ'}
          </Button>
        </div>
      </div>
    </form>
  );
}
