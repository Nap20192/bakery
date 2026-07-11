import { useEffect, useMemo, useRef, useState } from 'react';
import { Button } from '../../ui/Button';
import { CategoryBadge } from '../../ui/CategoryBadge';
import { ConfirmDialog } from '../../ui/ConfirmDialog';
import { ErrorBanner } from '../../ui/ErrorBanner';
import { compactControlClass } from '../../ui/Field';
import {
  createCategory,
  createDish,
  deleteCategory,
  deleteDish,
  fetchDishes,
  reorderDishes,
  searchAvailableDishes,
  updateCategory,
  updateDish,
} from '../../api/dishes';
import { fetchCategories } from '../../api/orders';
import { CATEGORY_COLOR_SLUGS, CATEGORY_COLORS } from '../../lib/categories';

const fieldClass =
  'h-11 w-full min-w-0 rounded-lg border border-stone-200 bg-white px-3 text-base text-stone-900 outline-none transition focus:border-stone-900 focus:ring-2 focus:ring-stone-900/10';
const cellInputClass = `${compactControlClass} h-9 w-full min-w-0 px-2`;

// AdminDishes is the full dish-catalog editor. Admins add dishes only from the
// iiko tech cards in the database (looked up on demand, never loaded whole),
// edit name/group, reorder by drag and drop, and delete entries.
export function AdminDishes() {
  const [dishes, setDishes] = useState([]);
  const [categories, setCategories] = useState([]);
  const [group, setGroup] = useState('');
  const [categoryID, setCategoryID] = useState('');
  const [selected, setSelected] = useState(null);
  const [query, setQuery] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [deleting, setDeleting] = useState(null);
  const dragCode = useRef(null);

  function load() {
    fetchDishes()
      .then((rows) => setDishes(Array.isArray(rows) ? rows : []))
      .catch((err) => setError(err instanceof Error ? err.message : String(err)));
    fetchCategories()
      .then((rows) => setCategories(Array.isArray(rows) ? rows : []))
      .catch((err) => setError(err instanceof Error ? err.message : String(err)));
  }

  useEffect(load, []);

  const categoryByID = useMemo(() => {
    const map = new Map();
    for (const category of categories) map.set(String(category.id), category);
    return map;
  }, [categories]);

  const takenCodes = useMemo(() => new Set(dishes.map((d) => d.code)), [dishes]);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return dishes;
    return dishes.filter((d) => [d.code, d.name, d.theme].some((f) => String(f || '').toLowerCase().includes(q)));
  }, [dishes, query]);

  const canReorder = !query.trim();

  async function onAdd() {
    if (!selected) return;
    setError('');
    setBusy(true);
    try {
      await createDish({
        code: selected.code,
        name: selected.name,
        theme: group.trim(),
        category_id: categoryID ? Number(categoryID) : null,
        sort_order: 0,
      });
      setSelected(null);
      setGroup('');
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

  async function confirmDelete() {
    const dish = deleting;
    setDeleting(null);
    if (!dish) return;
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
    <section className="bg-flour p-4 text-stone-900 sm:p-6" aria-labelledby="dishes-title">
      <div className="mx-auto max-w-5xl">
        <div className="mb-4 flex flex-wrap items-center justify-between gap-2">
          <h1 id="dishes-title" className="flex items-center gap-2 text-lg font-semibold">
            Блюда
            <span className="rounded-full bg-stone-100 px-2 py-0.5 text-note font-medium tabular-nums text-stone-600">{dishes.length}</span>
          </h1>
          <input
            className="h-9 w-full rounded-md border border-stone-300 bg-white px-3 text-sm outline-none focus:border-stone-900 focus:ring-2 focus:ring-stone-900/10 sm:w-64"
            type="search"
            placeholder="Поиск в каталоге…"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
        </div>

        <ErrorBanner error={error} className="mb-4" />

        <CategoryManager categories={categories} onChanged={load} onError={setError} />

        <div className="mb-2 grid gap-2 rounded-xl border border-stone-200 bg-white p-4 shadow-sm sm:grid-cols-[minmax(0,1fr)_minmax(0,10rem)_minmax(0,10rem)_auto]">
          <DishPicker selected={selected} onSelect={setSelected} takenCodes={takenCodes} onError={setError} />
          <select className={fieldClass} value={categoryID} onChange={(e) => setCategoryID(e.target.value)}>
            <option value="">Тип заявки…</option>
            {categories.map((category) => (
              <option key={category.id} value={String(category.id)}>{category.name}</option>
            ))}
          </select>
          <input className={fieldClass} placeholder="Группа (необязательно)" value={group} onChange={(e) => setGroup(e.target.value)} />
          <Button type="button" variant="primary" onClick={onAdd} disabled={busy || !selected}>Добавить</Button>
        </div>
        <p className="mb-6 text-note text-stone-500">Начните вводить название или код — блюда подгружаются из техкарт по мере ввода. Порядок меняется перетаскиванием строк.</p>

        <div className="overflow-x-auto rounded-xl border border-stone-200 bg-white shadow-sm">
          <table className="w-full text-sm">
            <thead className="bg-stone-50 text-left text-note text-stone-500">
              <tr>
                <th className="w-8 px-2 py-2"></th>
                <th className="px-3 py-2">Код</th>
                <th className="px-3 py-2">Название</th>
                <th className="px-3 py-2">Тип</th>
                <th className="px-3 py-2">Группа</th>
                <th className="px-3 py-2"></th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((dish) => (
                <DishRow
                  key={dish.code}
                  dish={dish}
                  categories={categories}
                  categoryByID={categoryByID}
                  canReorder={canReorder}
                  onSave={onSave}
                  onDelete={setDeleting}
                  setError={setError}
                  onDragStart={() => { dragCode.current = dish.code; }}
                  onDrop={() => onDrop(dish.code)}
                />
              ))}
              {filtered.length === 0 && (
                <tr><td colSpan={6} className="px-3 py-6 text-center text-body text-stone-500">{query ? 'Ничего не найдено.' : 'Блюд нет.'}</td></tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
      <ConfirmDialog
        open={Boolean(deleting)}
        title={`Удалить «${deleting?.name || ''}»?`}
        description="Блюдо исчезнет из каталога и шаблонов заказа."
        onConfirm={confirmDelete}
        onCancel={() => setDeleting(null)}
      />
    </section>
  );
}

// CategoryManager edits типы заявок: name, letter (goes into order numbers)
// and the highlight color. Deleting a category that still owns dishes is
// rejected by the backend.
function CategoryManager({ categories, onChanged, onError }) {
  const [draft, setDraft] = useState({ name: '', letter: '', color: CATEGORY_COLOR_SLUGS[0] });
  const [busy, setBusy] = useState(false);
  const [removing, setRemoving] = useState(null);

  async function onCreate() {
    setBusy(true);
    onError('');
    try {
      await createCategory({ ...draft, sort_order: categories.length + 1 });
      setDraft({ name: '', letter: '', color: CATEGORY_COLOR_SLUGS[0] });
      onChanged();
    } catch (err) {
      onError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  async function onUpdate(category, patch) {
    onError('');
    try {
      await updateCategory(category.id, { ...category, ...patch });
      onChanged();
    } catch (err) {
      onError(err instanceof Error ? err.message : String(err));
    }
  }

  async function confirmRemove() {
    const category = removing;
    setRemoving(null);
    if (!category) return;
    onError('');
    try {
      await deleteCategory(category.id);
      onChanged();
    } catch (err) {
      onError(err instanceof Error ? err.message : String(err));
    }
  }

  return (
    <section className="mb-6 rounded-xl border border-stone-200 bg-white p-4 shadow-sm">
      <h2 className="m-0 mb-1 text-title font-semibold text-stone-950">Типы заявок</h2>
      <p className="m-0 mb-3 text-note leading-5 text-stone-500">
        Тип выбирается магазином при создании заявки. Буква попадает в номер заказа, цвет выделяет заявки у пекаря.
      </p>
      <div className="space-y-2">
        {categories.map((category) => (
          <CategoryRow key={category.id} category={category} onUpdate={onUpdate} onRemove={setRemoving} />
        ))}
        <div className="grid gap-2 border-t border-stone-100 pt-3 sm:grid-cols-[minmax(0,1fr)_5rem_auto_auto]">
          <input
            className={cellInputClass}
            placeholder="Название нового типа…"
            value={draft.name}
            onChange={(e) => setDraft({ ...draft, name: e.target.value })}
          />
          <input
            className={`${cellInputClass} text-center uppercase`}
            placeholder="Буква"
            maxLength={1}
            value={draft.letter}
            onChange={(e) => setDraft({ ...draft, letter: e.target.value.toUpperCase() })}
          />
          <ColorSwatches value={draft.color} onChange={(color) => setDraft({ ...draft, color })} />
          <Button variant="primary" onClick={onCreate} disabled={busy || !draft.name.trim() || !draft.letter.trim()}>
            Добавить тип
          </Button>
        </div>
      </div>
      <ConfirmDialog
        open={Boolean(removing)}
        title={`Удалить тип «${removing?.name || ''}»?`}
        description="Тип с привязанными блюдами удалить нельзя — сначала перенесите их."
        onConfirm={confirmRemove}
        onCancel={() => setRemoving(null)}
      />
    </section>
  );
}

// parseMonitorCodes splits an admin-typed dough-code list ("17642, 17644 …")
// into clean codes.
function parseMonitorCodes(text) {
  return String(text || '')
    .split(/[\s,;]+/)
    .map((code) => code.trim())
    .filter(Boolean);
}

function CategoryRow({ category, onUpdate, onRemove }) {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(category);
  const [codesText, setCodesText] = useState('');

  if (!editing) {
    return (
      <div className="grid items-center gap-2 sm:grid-cols-[minmax(0,1fr)_auto_auto]">
        <div className="flex min-w-0 flex-wrap items-center gap-2">
          <CategoryBadge category={category} />
          <span className="min-w-0 truncate text-note text-stone-500">
            Тесто: {category.monitor_codes?.length ? category.monitor_codes.join(', ') : <strong className="text-red-700">коды не настроены</strong>}
          </span>
        </div>
        <span className="text-note text-stone-500">Буква в номере: <strong className="text-stone-800">{category.letter || '—'}</strong></span>
        <div className="flex justify-end gap-2">
          <Button
            onClick={() => {
              setDraft(category);
              setCodesText((category.monitor_codes || []).join(', '));
              setEditing(true);
            }}
          >
            Изменить
          </Button>
          <Button variant="danger" onClick={() => onRemove(category)}>Удалить</Button>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-2 rounded-lg bg-stone-50/60 p-2">
      <div className="grid items-center gap-2 sm:grid-cols-[minmax(0,1fr)_5rem_auto]">
        <input className={cellInputClass} value={draft.name} onChange={(e) => setDraft({ ...draft, name: e.target.value })} />
        <input
          className={`${cellInputClass} text-center uppercase`}
          maxLength={1}
          value={draft.letter}
          onChange={(e) => setDraft({ ...draft, letter: e.target.value.toUpperCase() })}
        />
        <ColorSwatches value={draft.color} onChange={(color) => setDraft({ ...draft, color })} />
      </div>
      <label className="block">
        <span className="mb-1 block text-note font-medium text-stone-500">Коды теста для расчёта (через запятую)</span>
        <input
          className={cellInputClass}
          placeholder="Например: 17642, 17644, 17650"
          value={codesText}
          onChange={(e) => setCodesText(e.target.value)}
        />
      </label>
      <div className="flex justify-end gap-2">
        <Button
          variant="primary"
          disabled={!draft.name.trim() || !draft.letter.trim()}
          onClick={async () => {
            await onUpdate(category, {
              name: draft.name,
              letter: draft.letter,
              color: draft.color,
              monitor_codes: parseMonitorCodes(codesText),
            });
            setEditing(false);
          }}
        >
          Сохранить
        </Button>
        <Button onClick={() => setEditing(false)}>Отмена</Button>
      </div>
    </div>
  );
}

// ColorSwatches is the fixed-palette color picker for тип заявки.
function ColorSwatches({ value, onChange }) {
  return (
    <div className="flex items-center gap-1.5" role="radiogroup" aria-label="Цвет типа">
      {CATEGORY_COLOR_SLUGS.map((slug) => (
        <button
          key={slug}
          type="button"
          role="radio"
          aria-checked={value === slug}
          title={slug}
          onClick={() => onChange(slug)}
          className={`h-7 w-7 rounded-full border-2 transition ${CATEGORY_COLORS[slug].dot} ${
            value === slug ? 'border-stone-900 ring-2 ring-stone-900/20' : 'border-white shadow-sm hover:border-stone-300'
          }`}
        />
      ))}
    </div>
  );
}

// DishPicker is a server-backed autocomplete over iiko tech cards: it queries
// the API as the admin types (debounced) so the full product list is never
// loaded. Already-added dishes are filtered out of the suggestions.
function DishPicker({ selected, onSelect, takenCodes, onError }) {
  const [text, setText] = useState('');
  const [results, setResults] = useState([]);
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (selected) return undefined;
    const q = text.trim();
    let alive = true;
    const t = setTimeout(() => {
      if (!alive) return;
      if (q.length < 1) {
        setResults([]);
        return;
      }
      setLoading(true);
      searchAvailableDishes(q)
        .then((rows) => {
          if (!alive) return;
          setResults((Array.isArray(rows) ? rows : []).filter((d) => !takenCodes.has(d.code)));
          setOpen(true);
        })
        .catch((err) => alive && onError(err instanceof Error ? err.message : String(err)))
        .finally(() => alive && setLoading(false));
    }, 250);
    return () => { alive = false; clearTimeout(t); };
  }, [text, selected, takenCodes, onError]);

  function pick(dish) {
    onSelect(dish);
    setText(`${dish.name} · ${dish.code}`);
    setResults([]);
    setOpen(false);
  }

  function onChange(e) {
    setText(e.target.value);
    if (selected) onSelect(null);
  }

  return (
    <div className="relative">
      <input
        className={fieldClass}
        placeholder="Поиск блюда из техкарт…"
        value={text}
        onChange={onChange}
        onFocus={() => results.length && setOpen(true)}
        onBlur={() => setTimeout(() => setOpen(false), 150)}
      />
      {open && (results.length > 0 || !loading) && (
        <ul className="absolute z-10 mt-1 max-h-72 w-full overflow-auto rounded-lg border border-stone-200 bg-white py-1 shadow-lg">
          {results.map((d) => (
            <li key={d.code}>
              <button
                type="button"
                className="block w-full px-3 py-2 text-left text-sm hover:bg-stone-100"
                onMouseDown={(e) => e.preventDefault()}
                onClick={() => pick(d)}
              >
                <span className="text-stone-900">{d.name}</span>
                <span className="ml-2 font-mono text-note text-stone-400">{d.code}</span>
              </button>
            </li>
          ))}
          {results.length === 0 && !loading && (
            <li className="px-3 py-2 text-sm text-stone-500">Ничего не найдено.</li>
          )}
        </ul>
      )}
    </div>
  );
}

// DishRow renders one catalog entry. The whole row is draggable for reordering;
// editing reveals inline inputs for name, тип заявки and group (the code is
// fixed — it ties the entry to its iiko tech card).
function DishRow({ dish, categories, categoryByID, canReorder, onSave, onDelete, setError, onDragStart, onDrop }) {
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
        category_id: draft.category_id ? Number(draft.category_id) : null,
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
        <td className="px-3 py-2 font-mono text-note text-stone-500">{dish.code}</td>
        <td className="px-3 py-2 text-stone-900">{dish.name}</td>
        <td className="px-3 py-2">
          <CategoryBadge category={categoryByID.get(String(dish.category_id || ''))} />
          {!dish.category_id && <span className="text-stone-400">—</span>}
        </td>
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
      <td className="px-3 py-2 font-mono text-note text-stone-400">{dish.code}</td>
      <td className="px-3 py-2"><input className={cellInputClass} value={draft.name} onChange={(e) => setDraft({ ...draft, name: e.target.value })} /></td>
      <td className="px-3 py-2">
        <select
          className={cellInputClass}
          value={draft.category_id ? String(draft.category_id) : ''}
          onChange={(e) => setDraft({ ...draft, category_id: e.target.value ? Number(e.target.value) : null })}
        >
          <option value="">Без типа</option>
          {categories.map((category) => (
            <option key={category.id} value={String(category.id)}>{category.name}</option>
          ))}
        </select>
      </td>
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
