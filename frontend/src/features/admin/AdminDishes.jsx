import { useEffect, useMemo, useState } from 'react';
import { Button } from '../../components/Button';
import { createDish, deleteDish, fetchDishes } from '../../api/dishes';

const fieldClass =
  'h-11 w-full rounded-lg border border-stone-200 bg-white px-3 text-base text-stone-900 outline-none transition focus:border-stone-900 focus:ring-2 focus:ring-stone-900/10';

const emptyForm = { name: '', theme: '' };

// AdminDishes lets an admin manage the dish catalog (add / remove dishes shops
// pick from when creating orders).
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
    return dishes.filter((d) => [d.name, d.theme].some((f) => String(f || '').toLowerCase().includes(q)));
  }, [dishes, query]);

  async function onCreate(event) {
    event.preventDefault();
    setError('');
    setBusy(true);
    try {
      await createDish({ name: form.name.trim(), theme: form.theme.trim() });
      setForm(emptyForm);
      load();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
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
      <div className="mx-auto max-w-3xl">
        <div className="mb-4 flex flex-wrap items-center justify-between gap-2">
          <h1 className="flex items-center gap-2 text-lg font-semibold">
            Блюда
            <span className="rounded-full bg-stone-100 px-2 py-0.5 text-[12px] font-medium tabular-nums text-stone-600">{dishes.length}</span>
          </h1>
          <input
            className="h-9 w-full rounded-md border border-stone-300 bg-white px-3 text-sm outline-none focus:border-stone-900 focus:ring-2 focus:ring-stone-900/10 sm:w-56"
            type="search"
            placeholder="Поиск: название, группа…"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
        </div>

        {error && <div className="mb-4 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-[13px] text-red-800">{error}</div>}

        <form onSubmit={onCreate} className="mb-6 grid gap-2 rounded-xl border border-stone-200 bg-white p-4 shadow-sm sm:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto]">
          <input className={fieldClass} placeholder="Название блюда" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
          <input className={fieldClass} placeholder="Группа (необязательно)" value={form.theme} onChange={(e) => setForm({ ...form, theme: e.target.value })} />
          <Button type="submit" variant="primary" disabled={busy || !form.name.trim()}>Добавить</Button>
        </form>

        <div className="overflow-hidden rounded-xl border border-stone-200 bg-white shadow-sm">
          <div className="divide-y divide-stone-100">
            {filtered.map((dish) => (
              <div className="flex items-center justify-between gap-3 px-3 py-2.5" key={dish.code}>
                <div className="min-w-0">
                  <span className="block truncate text-[14px] text-stone-900">{dish.name}</span>
                  {dish.theme && <span className="block truncate text-[12px] text-stone-400">{dish.theme}</span>}
                </div>
                <Button onClick={() => onDelete(dish)} variant="danger" className="shrink-0">Удалить</Button>
              </div>
            ))}
            {filtered.length === 0 && (
              <div className="px-3 py-6 text-center text-[13px] text-stone-500">{query ? 'Ничего не найдено.' : 'Блюд нет.'}</div>
            )}
          </div>
        </div>
      </div>
    </main>
  );
}
