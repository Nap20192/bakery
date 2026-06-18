import { useMemo, useState } from 'react';
import TextField from '@mui/material/TextField';
import { Button } from '../../components/Button';
import { EmptyState } from '../../components/EmptyState';
import { PanelHeader } from '../../components/Panel';

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
    onSave({ fulfillment_date: date, items });
  }

  return (
    <form className="space-y-3" onSubmit={submit}>
      <div className="flex flex-wrap items-start justify-between gap-2">
        <PanelHeader title={order ? `Изменить ${order.number}` : 'Новый заказ'} />
      </div>

      <div className="grid gap-2">
        <TextField
          size="medium"
          label="Дата выполнения"
          type="date"
          value={date}
          onChange={(event) => setDate(event.target.value)}
          slotProps={{ inputLabel: { shrink: true } }}
          required
          fullWidth
        />
      </div>

      <div className="max-h-[62vh] space-y-3 overflow-y-auto pr-1">
        {groups.length ? (
          groups.map((group) => (
            <section className="overflow-hidden rounded-md border border-stone-300" key={group.theme}>
              <h4 className="m-0 bg-stone-50 px-3 py-2 text-[13px] font-semibold text-stone-900">{group.theme}</h4>
              <div className="divide-y divide-stone-200">
                {group.items.map((item) => {
                  const value = quantities[item.name] || {};
                  return (
                    <div className="grid gap-2 px-3 py-3 sm:grid-cols-[minmax(0,1fr)_5.4rem_5.4rem] sm:items-center sm:py-2" key={item.name}>
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
          <EmptyState compact>Блюда не найдены.</EmptyState>
        )}
      </div>

      <div className="sticky bottom-0 flex justify-end gap-2 border-t border-stone-300 bg-white py-3">
        <Button type="button" onClick={onCancel}>
          Отмена
        </Button>
        <Button type="submit" variant="primary" disabled={loading}>
          {order ? 'Сохранить изменения' : 'Создать заказ'}
        </Button>
      </div>
    </form>
  );
}
