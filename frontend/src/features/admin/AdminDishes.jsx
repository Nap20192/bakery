import { useEffect, useMemo, useState } from 'react';
import { Button } from '../../components/Button';
import { createDish, deleteDish, fetchDishes, updateDish } from '../../api/dishes';

const fieldClass =
  'h-11 w-full min-w-0 rounded-lg border border-stone-200 bg-white px-3 text-base text-stone-900 outline-none transition focus:border-stone-900 focus:ring-2 focus:ring-stone-900/10';
const cellInputClass =
  'h-9 w-full min-w-0 rounded-md border border-stone-300 bg-white px-2 text-sm outline-none focus:border-stone-900 focus:ring-2 focus:ring-stone-900/10';

const emptyForm = { code: '', name: '', theme: '', sort_order: '' };

function toPayload(form) {
  return {
    code: form.code.trim(),
    name: form.name.trim(),
    theme: form.theme.trim(),
    sort_order: Number(form.sort_order) || 0,
  };
}

// AdminDishes is the full dish-catalog editor: admins create, edit (code, name,
// group, sort order) and delete the dishes shops pick from when ordering.
export function AdminDishes() {
  const [dishes, setDishes] = useState([]);
  const [form, setForm] = useState(emptyForm);
  const [query, setQuery] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  function load() {
    fetchDishes()
      .then((rows) => setDishes(Array.isArray(rows) ? rows : []))
      .catch((err) => setError(err instanceof Error ? err.message : String(err)));
  }

  useEffect(load, []);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return dishes;
    return dishes.filter((d) => [d.code, d.name, d.theme].some((f) => String(f || '').toLowerCase().includes(q)));
  }, [dishes, query]);

  async function onCreate(event) {
    event.preventDefault();
    setError('');
    setBusy(true);
    try {
      await createDish(toPayload(form));
      setForm(emptyForm);
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

        <form onSubmit={onCreate} className="mb-6 grid gap-2 rounded-xl border border-stone-200 bg-white p-4 shadow-sm sm:grid-cols-[minmax(0,8rem)_minmax(0,1fr)_minmax(0,1fr)_minmax(0,6rem)_auto]">
          <input className={fieldClass} placeholder="Код" value={form.code} onChange={(e) => setForm({ ...form, code: e.target.value })} />
          <input className={fieldClass} placeholder="Название блюда" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
          <input className={fieldClass} placeholder="Группа" value={form.theme} onChange={(e) => setForm({ ...form, theme: e.target.value })} />
          <input className={fieldClass} type="number" inputMode="numeric" placeholder="Порядок" value={form.sort_order} onChange={(e) => setForm({ ...form, sort_order: e.target.value })} />
          <Button type="submit" variant="primary" disabled={busy || !form.name.trim()}>Добавить</Button>
        </form>
        <p className="-mt-4 mb-6 text-[12px] text-stone-500">Код можно не указывать — он будет создан автоматически из названия. Порядок задаёт сортировку (0 — в конце).</p>

        <div className="overflow-x-auto rounded-xl border border-stone-200 bg-white shadow-sm">
          <table className="w-full text-sm">
            <thead className="bg-stone-50 text-left text-[12px] text-stone-500">
              <tr>
                <th className="px-3 py-2">Код</th>
                <th className="px-3 py-2">Название</th>
                <th className="px-3 py-2">Группа</th>
                <th className="px-3 py-2 w-20">Порядок</th>
                <th className="px-3 py-2"></th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((dish) => (
                <DishRow key={dish.code} dish={dish} onSave={onSave} onDelete={onDelete} setError={setError} />
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

// DishRow renders one catalog entry; toggling edit reveals inline inputs for
// every field (code included) and saves the whole row at once.
function DishRow({ dish, onSave, onDelete, setError }) {
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
        code: String(draft.code || '').trim(),
        name: String(draft.name || '').trim(),
        theme: String(draft.theme || '').trim(),
        sort_order: Number(draft.sort_order) || 0,
      });
      setEditing(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  if (!editing) {
    return (
      <tr className="border-t border-stone-100">
        <td className="px-3 py-2 font-mono text-[12px] text-stone-500">{dish.code}</td>
        <td className="px-3 py-2 text-stone-900">{dish.name}</td>
        <td className="px-3 py-2 text-stone-500">{dish.theme || '—'}</td>
        <td className="px-3 py-2 tabular-nums text-stone-500">{dish.sort_order ?? 0}</td>
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
      <td className="px-3 py-2"><input className={`${cellInputClass} font-mono`} value={draft.code} onChange={(e) => setDraft({ ...draft, code: e.target.value })} /></td>
      <td className="px-3 py-2"><input className={cellInputClass} value={draft.name} onChange={(e) => setDraft({ ...draft, name: e.target.value })} /></td>
      <td className="px-3 py-2"><input className={cellInputClass} value={draft.theme} onChange={(e) => setDraft({ ...draft, theme: e.target.value })} /></td>
      <td className="px-3 py-2"><input className={cellInputClass} type="number" inputMode="numeric" value={draft.sort_order} onChange={(e) => setDraft({ ...draft, sort_order: e.target.value })} /></td>
      <td className="px-3 py-2">
        <div className="flex justify-end gap-2">
          <Button onClick={commit} variant="primary" disabled={busy || !String(draft.name).trim()} className="shrink-0">Сохранить</Button>
          <Button onClick={() => setEditing(false)} className="shrink-0">Отмена</Button>
        </div>
      </td>
    </tr>
  );
}
