import { useMemo, useState } from 'react';
import { Button } from '../../components/Button';
import { EmptyState } from '../../components/EmptyState';
import { PanelHeader } from '../../components/Panel';
import { Icon } from '../../components/Icon';
import { formatQuantity } from '../../lib/format';

const controlClass =
  'h-11 w-full rounded-lg border border-stone-200 bg-white px-3 text-base text-stone-900 outline-none transition focus:border-stone-900 focus:ring-2 focus:ring-stone-900/10';
const qtyClass =
  'h-10 w-full rounded-lg border border-stone-200 bg-stone-50 px-1 text-center text-base text-stone-900 outline-none transition focus:border-stone-900 focus:bg-white focus:ring-2 focus:ring-stone-900/10';

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

function initialItemComments(order) {
  const map = {};
  for (const entry of order?.comments?.items || []) {
    if (entry.product_name) map[entry.product_name] = entry.comment || '';
  }
  return map;
}

// parsePasteLine extracts "<name> <qty>[+<reserved>]" from a single line.
function parsePasteLine(line) {
  const match = line.match(/^(.*?)[\s:–—-]*?(\d+(?:[.,]\d+)?)\s*(?:\+\s*(\d+(?:[.,]\d+)?))?\s*(?:шт\.?)?$/i);
  if (!match) return null;
  const name = match[1].trim();
  if (!name) return null;
  const quantity = match[2].replace(',', '.');
  const reserved = match[3] ? match[3].replace(',', '.') : '';
  return { name, quantity, reserved };
}

export function OrderEditor({ catalog, order, loading, onCancel, onSave }) {
  const [date, setDate] = useState(order?.fulfillment_date || '');
  const [quantities, setQuantities] = useState(() => initialQuantities(order));
  const [comments, setComments] = useState(() => initialItemComments(order));
  const [openComments, setOpenComments] = useState({});
  const [general, setGeneral] = useState(order?.comments?.general || '');
  const [search, setSearch] = useState('');
  const [pasteOpen, setPasteOpen] = useState(false);
  const [pasteText, setPasteText] = useState('');
  const [pasteResult, setPasteResult] = useState(null);

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

  // Index catalog names (lower-cased) for paste matching.
  const nameIndex = useMemo(() => {
    const map = new Map();
    for (const item of catalog) map.set(item.name.toLowerCase(), item.name);
    return map;
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

  function updateComment(name, value) {
    setComments((current) => ({ ...current, [name]: value }));
  }

  function resolveName(rawName) {
    const lower = rawName.toLowerCase();
    if (nameIndex.has(lower)) return nameIndex.get(lower);
    // Fallback: unique partial match.
    const hits = [];
    for (const [key, name] of nameIndex) {
      if (key.includes(lower) || lower.includes(key)) hits.push(name);
    }
    return hits.length === 1 ? hits[0] : null;
  }

  function applyPaste() {
    const lines = pasteText.split('\n').map((line) => line.trim()).filter(Boolean);
    const next = {};
    const unmatched = [];
    let matched = 0;
    for (const line of lines) {
      const parsed = parsePasteLine(line);
      if (!parsed) {
        unmatched.push(line);
        continue;
      }
      const name = resolveName(parsed.name);
      if (!name) {
        unmatched.push(line);
        continue;
      }
      next[name] = { quantity: parsed.quantity, reserved_quantity: parsed.reserved };
      matched += 1;
    }
    setQuantities((current) => ({ ...current, ...next }));
    setPasteResult({ matched, unmatched });
    if (matched > 0 && unmatched.length === 0) {
      setPasteOpen(false);
      setPasteText('');
    }
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
    const itemComments = items
      .map((item) => ({ product_name: item.product_name, comment: (comments[item.product_name] || '').trim() }))
      .filter((entry) => entry.comment);
    onSave({
      fulfillment_date: date,
      items,
      comments: { general: general.trim(), items: itemComments },
    });
  }

  const canSubmit = !loading && summary.count > 0 && Boolean(date);

  return (
    <form className="space-y-3" onSubmit={submit}>
      <div className="flex items-center justify-between gap-2">
        <PanelHeader title={order ? `Изменить ${order.number}` : 'Новый заказ'} />
        <Button type="button" variant="ghost" className="shrink-0" onClick={() => setPasteOpen((v) => !v)}>
          <Icon name="orders" size={15} />
          Вставить списком
        </Button>
      </div>

      {pasteOpen && (
        <div className="space-y-2 rounded-lg border border-stone-200 bg-stone-50 p-2.5">
          <textarea
            className="block min-h-[5.5rem] w-full resize-y rounded-lg border border-stone-200 bg-white p-2.5 text-base leading-6 text-stone-900 outline-none focus:border-stone-900 focus:ring-2 focus:ring-stone-900/10"
            placeholder={'Вставьте список, например:\nКокрок 5\nСосиска в тесте 3+2'}
            value={pasteText}
            onChange={(e) => setPasteText(e.target.value)}
          />
          <div className="flex items-center justify-between gap-2">
            <span className="text-[12px] text-stone-500">Строка: «название кол-во» или «название кол-во+заказ».</span>
            <div className="flex gap-2">
              <Button type="button" variant="ghost" onClick={() => { setPasteText(''); setPasteResult(null); }}>Очистить</Button>
              <Button type="button" variant="primary" onClick={applyPaste} disabled={!pasteText.trim()}>Применить</Button>
            </div>
          </div>
          {pasteResult && (
            <div className="text-[12px] leading-5">
              <span className="text-stone-700">Распознано: <strong className="tabular-nums">{pasteResult.matched}</strong>.</span>
              {pasteResult.unmatched.length > 0 && (
                <span className="text-red-700"> Не найдено: {pasteResult.unmatched.join('; ')}</span>
              )}
            </div>
          )}
        </div>
      )}

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

      <div className="space-y-3">
        {visibleGroups.length ? (
          visibleGroups.map((group) => (
            <section key={group.theme}>
              <div className="grid grid-cols-[minmax(0,1fr)_3.5rem_3.5rem] items-center gap-2 px-1 pb-1">
                <h4 className="m-0 truncate text-[12px] font-semibold uppercase tracking-wide text-stone-400">{group.theme}</h4>
                <span className="text-center text-[10px] font-medium uppercase text-stone-300">Кол-во</span>
                <span className="text-center text-[10px] font-medium uppercase text-stone-300">Заказ</span>
              </div>
              <div className="divide-y divide-stone-100">
                {group.items.map((item) => {
                  const value = quantities[item.name] || {};
                  const filled = Number(value.quantity || 0) + Number(value.reserved_quantity || 0) > 0;
                  return (
                    <div className="py-1.5" key={item.name}>
                      <div className="grid grid-cols-[minmax(0,1fr)_3.5rem_3.5rem] items-center gap-2">
                        <span className="min-w-0 break-words text-[14px] leading-5 text-stone-800">{item.name}</span>
                        <input
                          className={qtyClass}
                          type="text"
                          inputMode="numeric"
                          pattern="[0-9]*"
                          placeholder="0"
                          aria-label={`${item.name}: количество`}
                          value={value.quantity || ''}
                          onChange={(e) => updateQuantity(item.name, 'quantity', e.target.value.replace(/\D/g, ''))}
                        />
                        <input
                          className={qtyClass}
                          type="text"
                          inputMode="numeric"
                          pattern="[0-9]*"
                          placeholder="0"
                          aria-label={`${item.name}: заказано`}
                          value={value.reserved_quantity || ''}
                          onChange={(e) => updateQuantity(item.name, 'reserved_quantity', e.target.value.replace(/\D/g, ''))}
                        />
                      </div>
                      {filled && (openComments[item.name] || comments[item.name] ? (
                        <input
                          className="fade-in mt-1.5 h-9 w-full rounded-lg border border-stone-200 bg-white px-3 text-base text-stone-700 outline-none transition focus:border-stone-900 focus:ring-2 focus:ring-stone-900/10"
                          type="text"
                          autoFocus={Boolean(openComments[item.name]) && !comments[item.name]}
                          placeholder="Комментарий к позиции…"
                          aria-label={`${item.name}: комментарий`}
                          value={comments[item.name] || ''}
                          onChange={(e) => updateComment(item.name, e.target.value)}
                        />
                      ) : (
                        <button
                          type="button"
                          onClick={() => setOpenComments((o) => ({ ...o, [item.name]: true }))}
                          className="mt-1.5 inline-flex items-center gap-1 text-[12px] font-medium text-stone-400 transition hover:text-stone-700"
                        >
                          <Icon name="plus" size={13} />
                          Комментарий
                        </button>
                      ))}
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

      <label className="block">
        <span className="mb-1 block text-[12px] font-medium text-stone-500">Общий комментарий</span>
        <textarea
          className="block min-h-[3.5rem] w-full resize-y rounded-lg border border-stone-200 bg-white p-2.5 text-base leading-6 text-stone-900 outline-none transition focus:border-stone-900 focus:ring-2 focus:ring-stone-900/10"
          placeholder="Комментарий ко всему заказу…"
          value={general}
          onChange={(e) => setGeneral(e.target.value)}
        />
      </label>

      <div className="-mx-3 mt-1 flex items-center justify-between gap-2 border-t border-stone-200 bg-white px-3 py-3 sm:sticky sm:bottom-0 sm:-mx-4 sm:bg-white/95 sm:px-4 sm:backdrop-blur">
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
