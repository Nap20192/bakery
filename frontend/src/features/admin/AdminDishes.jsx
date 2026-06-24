import { useEffect, useMemo, useRef, useState } from 'react';
import { Button } from '../../components/Button';
import {
  createDish,
  deleteDish,
  fetchAvailableDishes,
  fetchDishes,
  reorderDishes,
  updateDish,
} from '../../api/dishes';

const fieldClass =
  'h-11 w-full min-w-0 rounded-lg border border-stone-200 bg-white px-3 text-base text-stone-900 outline-none transition focus:border-stone-900 focus:ring-2 focus:ring-stone-900/10';
const cellInputClass =
  'h-9 w-full min-w-0 rounded-md border border-stone-300 bg-white px-2 text-sm outline-none focus:border-stone-900 focus:ring-2 focus:ring-stone-900/10';

const emptyForm = { code: '', theme: '' };

// AdminDishes is the full dish-catalog editor. Admins add dishes only from the
// iiko tech cards present in the database, edit name/group, reorder by drag and
// drop, and delete entries. Shops pick from this catalog when ordering.
export function AdminDishes() {
  const [dishes, setDishes] = useState([]);
  const [available, setAvailable] = useState([]);
  const [form, setForm] = useState(emptyForm);
  const [pickerQuery, setPickerQuery] = useState('');
  const [query, setQuery] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const dragCode = useRef(null);

  function load() {
    fetchDishes()
      .then((rows) => setDishes(Array.isArray(rows) ? rows : []))
      .catch((err) => setError(err instanceof Error ? err.message : String(err)));
  }

  useEffect(() => {
    load();
    fetchAvailableDishes()
      .then((rows) => setAvailable(Array.isArray(rows) ? rows : []))
      .catch((err) => setError(err instanceof Error ? err.message : String(err)));
  }, []);

  // Dishes already in the catalog can't be added again.
  const addable = useMemo(() => {
    const taken = new Set(dishes.map((d) => d.code));
    const q = pickerQuery.trim().toLowerCase();
    return available
      .filter((d) => !taken.has(d.code))
      .filter((d) => !q || [d.code, d.name].some((f) => String(f || '').toLowerCase().includes(q)));
  }, [available, dishes, pickerQuery]);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return dishes;
    return dishes.filter((d) => [d.code, d.name, d.theme].some((f) => String(f || '').toLowerCase().includes(q)));
  }, [dishes, query]);

  const canReorder = !query.trim();

  async function onCreate(event) {
    event.preventDefault();
    const dish = available.find((d) => d.code === form.code);
    if (!dish) return;
    setError('');
    setBusy(true);
    try {
      await createDish({ code: dish.code, name: dish.name, theme: form.theme.trim(), sort_order: 0 });
      setForm(emptyForm);
      setPickerQuery('');
      load();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  async function onSave(code, patch) {
    setError('');
    await updateDish(code, patch);
    load();
  }

  async function onDelete(dish) {
    if (!window.confirm(`Удалить «${dish.name}»?`)) return;
    setError('');
    try {
      await deleteDish(dish.code);
      load();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function onDrop(targetCode) {
    const from = dishes.findIndex((d) => d.code === dragCode.current);
    const to = dishes.findIndex((d) => d.code === targetCode);
    dragCode.current = null;
    if (from < 0 || to < 0 || from === to) return;
    const next = [...dishes];
    const [moved] = next.splice(from, 1);
    next.splice(to, 0, moved);
    setDishes(next);
    try {
      await reorderDishes(next.map((d) => d.code));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      load();
    }
  }

  return (
    <main className="bg-[#f7f3ea] p-4 text-stone-900 sm:p-6">
      <div className="mx-auto max-w-5xl">
        <div className="mb-4 flex flex-wrap items-center justify-between gap-2">
          <h1 className="flex items-center gap-2 text-lg font-semibold">
            Блюда
            <span className="rounded-full bg-stone-100 px-2 py-0.5 text-[12px] font-medium tabular-nums text-stone-600">{dishes.length}</span>
          </h1>
          <input
            className="h-9 w-full rounded-md border border-stone-300 bg-white px-3 text-sm outline-none focus:border-stone-900 focus:ring-2 focus:ring-stone-900/10 sm:w-64"
            type="search"
            placeholder="Поиск: код, название, группа…"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
        </div>

        {error && <div className="mb-4 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-[13px] text-red-800">{error}</div>}

        <form onSubmit={onCreate} className="mb-2 grid gap-2 rounded-xl border border-stone-200 bg-white p-4 shadow-sm sm:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_minmax(0,10rem)_auto]">
          <input
            className={fieldClass}
            placeholder="Поиск блюда из техкарт…"
            value={pickerQuery}
            onChange={(e) => setPickerQuery(e.target.value)}
          />
          <select className={fieldClass} value={form.code} onChange={(e) => setForm({ ...form, code: e.target.value })}>
            <option value="">— выберите блюдо —</option>
            {addable.slice(0, 200).map((d) => (
              <option key={d.code} value={d.code}>{d.name} · {d.code}</option>
            ))}
          </select>
          <input className={fieldClass} placeholder="Группа" value={form.theme} onChange={(e) => setForm({ ...form, theme: e.target.value })} />
          <Button type="submit" variant="primary" disabled={busy || !form.code}>Добавить</Button>
        </form>
        <p className="mb-6 text-[12px] text-stone-500">
          Добавлять можно только блюда с техкартой из базы ({available.length} доступно). Порядок меняется перетаскиванием строк.
        </p>

        <div className="overflow-x-auto rounded-xl border border-stone-200 bg-white shadow-sm">
          <table className="w-full text-sm">
            <thead className="bg-stone-50 text-left text-[12px] text-stone-500">
              <tr>
                <th className="w-8 px-2 py-2"></th>
                <th className="px-3 py-2">Код</th>
                <th className="px-3 py-2">Название</th>
                <th className="px-3 py-2">Группа</th>
                <th className="px-3 py-2"></th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((dish) => (
                <DishRow
                  key={dish.code}
                  dish={dish}
                  canReorder={canReorder}
                  onSave={onSave}
                  onDelete={onDelete}
                  setError={setError}
                  onDragStart={() => { dragCode.current = dish.code; }}
                  onDrop={() => onDrop(dish.code)}
                />
              ))}
              {filtered.length === 0 && (
                <tr><td colSpan={5} className="px-3 py-6 text-center text-[13px] text-stone-500">{query ? 'Ничего не найдено.' : 'Блюд нет.'}</td></tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
    </main>
  );
}

// DishRow renders one catalog entry. The whole row is draggable for reordering;
// editing reveals inline inputs for name and group (the code is fixed — it ties
// the entry to its iiko tech card).
function DishRow({ dish, canReorder, onSave, onDelete, setError, onDragStart, onDrop }) {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(dish);
  const [busy, setBusy] = useState(false);

  function startEdit() {
    setDraft(dish);
    setEditing(true);
  }

  async function commit() {
    setBusy(true);
    try {
      await onSave(dish.code, {
        code: dish.code,
        name: String(draft.name || '').trim(),
        theme: String(draft.theme || '').trim(),
        sort_order: Number(dish.sort_order) || 0,
      });
      setEditing(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  const dragProps = canReorder && !editing
    ? {
        draggable: true,
        onDragStart,
        onDragOver: (e) => e.preventDefault(),
        onDrop,
      }
    : {};

  if (!editing) {
    return (
      <tr className="border-t border-stone-100" {...dragProps}>
        <td className={`px-2 py-2 text-center text-stone-300 ${canReorder ? 'cursor-grab' : ''}`} title={canReorder ? 'Перетащите для сортировки' : 'Сбросьте поиск, чтобы менять порядок'}>⠿</td>
        <td className="px-3 py-2 font-mono text-[12px] text-stone-500">{dish.code}</td>
        <td className="px-3 py-2 text-stone-900">{dish.name}</td>
        <td className="px-3 py-2 text-stone-500">{dish.theme || '—'}</td>
        <td className="px-3 py-2">
          <div className="flex justify-end gap-2">
            <Button onClick={startEdit} className="shrink-0">Изменить</Button>
            <Button onClick={() => onDelete(dish)} variant="danger" className="shrink-0">Удалить</Button>
          </div>
        </td>
      </tr>
    );
  }

  return (
    <tr className="border-t border-stone-100 bg-stone-50/60">
      <td className="px-2 py-2"></td>
      <td className="px-3 py-2 font-mono text-[12px] text-stone-400">{dish.code}</td>
      <td className="px-3 py-2"><input className={cellInputClass} value={draft.name} onChange={(e) => setDraft({ ...draft, name: e.target.value })} /></td>
      <td className="px-3 py-2"><input className={cellInputClass} value={draft.theme} onChange={(e) => setDraft({ ...draft, theme: e.target.value })} /></td>
      <td className="px-3 py-2">
        <div className="flex justify-end gap-2">
          <Button onClick={commit} variant="primary" disabled={busy || !String(draft.name).trim()} className="shrink-0">Сохранить</Button>
          <Button onClick={() => setEditing(false)} className="shrink-0">Отмена</Button>
        </div>
      </td>
    </tr>
  );
}
