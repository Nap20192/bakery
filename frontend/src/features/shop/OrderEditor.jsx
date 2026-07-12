import { useMemo, useState } from 'react';
import { Button } from '../../ui/Button';
import { CategoryBadge } from '../../ui/CategoryBadge';
import { EmptyState } from '../../ui/EmptyState';
import { PanelHeader } from '../../ui/Panel';
import { Icon } from '../../ui/Icon';
import { categoryStyle } from '../../lib/categories';
import { formatQuantity } from '../../lib/format';

const controlClass =
  'block h-11 w-full min-w-0 max-w-full appearance-none rounded-lg border border-stone-200 bg-white px-3 text-base text-stone-900 outline-none transition focus:border-stone-900 focus:ring-2 focus:ring-stone-900/10';
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

export function OrderEditor({ mode, catalog, categories = [], order, shops = [], loading, onCancel, onSave }) {
  // mode: 'create' | 'update'. Дублирование — это mode='create' с исходным
  // order для префилла позиций: форма ведёт себя как новый заказ (POST, выбор
  // типа зафиксирован из источника, дата пустая), но поля предзаполнены.
  const isEdit = mode ? mode === 'update' : Boolean(order);
  const [fromDepartmentID, setFromDepartmentID] = useState(
    order?.from_department?.id ? String(order.from_department.id) : '',
  );
  // Тип заявки выбирается первым шагом при создании; при редактировании он
  // зафиксирован (тип участвует в номере заказа и не меняется).
  const [categoryID, setCategoryID] = useState(order?.category?.id ? String(order.category.id) : '');
  // Дата не переносится в дубликат — новый заказ получает свою дату выполнения.
  const [date, setDate] = useState(isEdit ? order?.fulfillment_date || '' : '');
  const [quantities, setQuantities] = useState(() => initialQuantities(order));
  const [comments, setComments] = useState(() => initialItemComments(order));
  const [openComments, setOpenComments] = useState({});
  const [general, setGeneral] = useState(order?.comments?.general || '');
  const [pasteOpen, setPasteOpen] = useState(false);
  const [pasteText, setPasteText] = useState('');
  const [pasteResult, setPasteResult] = useState(null);

  // Каталог сужен до блюд выбранного типа заявки; блюда без типа видны везде,
  // чтобы новое блюдо не пропало из формы, пока админ не назначил ему тип.
  const categoryCatalog = useMemo(() => {
    if (!categoryID) return catalog;
    return catalog.filter((item) => !item.category_id || String(item.category_id) === String(categoryID));
  }, [catalog, categoryID]);

  const groups = useMemo(() => {
    const result = [];
    const byTheme = new Map();
    for (const item of categoryCatalog) {
      if (!byTheme.has(item.theme)) {
        const group = { theme: item.theme || 'Без группы', items: [] };
        byTheme.set(item.theme, group);
        result.push(group);
      }
      byTheme.get(item.theme).items.push(item);
    }
    return result;
  }, [categoryCatalog]);

  // Index catalog names (lower-cased) for paste matching.
  const nameIndex = useMemo(() => {
    const map = new Map();
    for (const item of categoryCatalog) map.set(item.name.toLowerCase(), item.name);
    return map;
  }, [categoryCatalog]);

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
    const items = categoryCatalog
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
      from_department_id: Number(fromDepartmentID),
      category_id: Number(categoryID),
      items,
      comments: { general: general.trim(), items: itemComments },
    });
  }

  const canSubmit = !loading && summary.count > 0 && Boolean(date) && Boolean(fromDepartmentID) && Boolean(categoryID);
  const selectedCategory = categories.find((category) => String(category.id) === String(categoryID)) || order?.category || null;

  // Шаг 1 при создании: выбор типа заявки. Форма с блюдами появляется после.
  // При дублировании тип уже перенесён из источника — шаг пропускается.
  if (!isEdit && !categoryID) {
    return (
      <div className="space-y-3">
        <PanelHeader title="Новый заказ" />
        <p className="m-0 text-body leading-5 text-stone-500">Выберите тип заявки — от него зависит список блюд.</p>
        <div className="grid gap-2 sm:grid-cols-2">
          {categories.map((category) => {
            const style = categoryStyle(category);
            return (
              <button
                key={category.id}
                type="button"
                onClick={() => setCategoryID(String(category.id))}
                className={`flex min-h-16 items-center gap-3 rounded-xl border px-4 text-left transition focus:outline-none focus:ring-2 focus:ring-stone-900/20 ${style.pick}`}
              >
                <span className={`h-3 w-3 shrink-0 rounded-full ${style.dot}`} aria-hidden="true" />
                <span className="min-w-0">
                  <span className="block text-title font-semibold leading-6 text-stone-950">{category.name}</span>
                  <span className="block text-note leading-5 text-stone-500">Заявка «{category.letter}» в номере</span>
                </span>
              </button>
            );
          })}
        </div>
        {categories.length === 0 && <EmptyState compact>Типы заявок не настроены. Обратитесь к администратору.</EmptyState>}
        <div className="flex justify-end">
          <Button type="button" onClick={onCancel}>Отмена</Button>
        </div>
      </div>
    );
  }

  return (
    <form className="space-y-3" onSubmit={submit}>
      <div className="flex items-center justify-between gap-2">
        <div className="flex min-w-0 flex-wrap items-center gap-2">
          <PanelHeader title={isEdit ? `Изменить ${order.number}` : 'Новый заказ'} />
          <CategoryBadge category={selectedCategory} />
          {!isEdit && (
            <Button type="button" variant="ghost" className="!min-h-7 !px-2 text-note" onClick={() => setCategoryID('')}>
              Сменить тип
            </Button>
          )}
        </div>
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
            <span className="text-note text-stone-500">Строка: «название кол-во» или «название кол-во+заказ».</span>
            <div className="flex gap-2">
              <Button type="button" variant="ghost" onClick={() => { setPasteText(''); setPasteResult(null); }}>Очистить</Button>
              <Button type="button" variant="primary" onClick={applyPaste} disabled={!pasteText.trim()}>Применить</Button>
            </div>
          </div>
          {pasteResult && (
            <div className="text-note leading-5">
              <span className="text-stone-700">Распознано: <strong className="tabular-nums">{pasteResult.matched}</strong>.</span>
              {pasteResult.unmatched.length > 0 && (
                <span className="text-red-700"> Не найдено: {pasteResult.unmatched.join('; ')}</span>
              )}
            </div>
          )}
        </div>
      )}

      <label className="block">
        <span className="mb-1 block text-note font-medium text-stone-500">Магазин (откуда заказ)</span>
        <select
          className={`${controlClass} cursor-pointer bg-[length:1.1rem] bg-[right_0.6rem_center] bg-no-repeat pr-9 bg-[url("data:image/svg+xml,%3Csvg%20xmlns='http://www.w3.org/2000/svg'%20viewBox='0%200%2024%2024'%20fill='none'%20stroke='%2378716c'%20stroke-width='2'%3E%3Cpath%20d='M6%209l6%206%206-6'/%3E%3C/svg%3E")]`}
          value={fromDepartmentID}
          onChange={(e) => setFromDepartmentID(e.target.value)}
          required
        >
          <option value="" disabled>Выберите магазин…</option>
          {shops.map((shop) => (
            <option key={shop.id} value={String(shop.id)}>{shop.name}</option>
          ))}
        </select>
      </label>

      <label className="block">
        <span className="mb-1 block text-note font-medium text-stone-500">Дата выполнения</span>
        <input className={controlClass} type="date" value={date} min={todayValue()} onChange={(e) => setDate(e.target.value)} required />
      </label>

      <div className="space-y-3">
        {groups.length ? (
          groups.map((group) => (
            <section key={group.theme}>
              <div className="grid grid-cols-[minmax(0,1fr)_3.5rem_3.5rem_2rem] items-center gap-2 px-1 pb-1">
                <h4 className="m-0 truncate text-note font-semibold uppercase tracking-wide text-stone-400">{group.theme}</h4>
                <span className="text-center text-caption font-medium uppercase text-stone-300">Кол-во</span>
                <span className="text-center text-caption font-medium uppercase text-stone-300">Заказ</span>
                <span />
              </div>
              <div className="divide-y divide-stone-100">
                {group.items.map((item) => {
                  const value = quantities[item.name] || {};
                  const filled = Number(value.quantity || 0) + Number(value.reserved_quantity || 0) > 0;
                  const hasComment = Boolean(comments[item.name]);
                  const commentOpen = filled && (openComments[item.name] || hasComment);
                  return (
                    <div className="py-1.5" key={item.name}>
                      <div className="grid grid-cols-[minmax(0,1fr)_3.5rem_3.5rem_2rem] items-center gap-2">
                        <span className="min-w-0 break-words text-input leading-5 text-stone-800">{item.name}</span>
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
                        {filled ? (
                          <button
                            type="button"
                            onClick={() => setOpenComments((o) => ({ ...o, [item.name]: !commentOpen }))}
                            title="Комментарий"
                            aria-label={`${item.name}: комментарий`}
                            className={`inline-flex h-8 w-8 items-center justify-center rounded-lg border transition ${
                              hasComment
                                ? 'border-stone-900 bg-stone-900 text-white'
                                : commentOpen
                                  ? 'border-stone-400 text-stone-700'
                                  : 'border-stone-200 text-stone-400 hover:border-stone-400 hover:text-stone-700'
                            }`}
                          >
                            <Icon name="orders" size={15} />
                          </button>
                        ) : (
                          <span />
                        )}
                      </div>
                      {commentOpen && (
                        <input
                          className="fade-in mt-1.5 h-9 w-full rounded-lg border border-stone-200 bg-white px-3 text-base text-stone-700 outline-none transition focus:border-stone-900 focus:ring-2 focus:ring-stone-900/10"
                          type="text"
                          autoFocus={Boolean(openComments[item.name]) && !hasComment}
                          placeholder="Комментарий к позиции…"
                          aria-label={`${item.name}: текст комментария`}
                          value={comments[item.name] || ''}
                          onChange={(e) => updateComment(item.name, e.target.value)}
                        />
                      )}
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

      <label className="block">
        <span className="mb-1 block text-note font-medium text-stone-500">Общий комментарий</span>
        <textarea
          className="block min-h-[3.5rem] w-full resize-y rounded-lg border border-stone-200 bg-white p-2.5 text-base leading-6 text-stone-900 outline-none transition focus:border-stone-900 focus:ring-2 focus:ring-stone-900/10"
          placeholder="Комментарий ко всему заказу…"
          value={general}
          onChange={(e) => setGeneral(e.target.value)}
        />
      </label>

      <div className="-mx-3 mt-1 flex items-center justify-between gap-2 border-t border-stone-200 bg-white px-3 py-3 sm:sticky sm:bottom-0 sm:-mx-4 sm:bg-white/95 sm:px-4 sm:backdrop-blur">
        <span className="text-body text-stone-600">
          <strong className="text-stone-900 tabular-nums">{summary.count}</strong> поз.
          <span className="mx-1.5 text-stone-300">·</span>
          <strong className="text-stone-900 tabular-nums">{formatQuantity(summary.total)}</strong> шт.
        </span>
        <div className="flex gap-2">
          <Button type="button" onClick={onCancel}>Отмена</Button>
          <Button type="submit" variant="primary" loading={loading} disabled={!canSubmit}>
            {isEdit ? 'Сохранить' : 'Создать'}
          </Button>
        </div>
      </div>
    </form>
  );
}
