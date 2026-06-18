import { useEffect, useMemo, useState } from 'react';
import { Button } from '../../components/Button';
import { createUser, deleteUser, fetchAdminDepartments, fetchUsers, updateUser } from '../../api/users';
import { logWarn } from '../../lib/logger';

const ROLES = ['admin', 'shop', 'baker', 'user'];

const emptyForm = { username: '', password: '', telegram_username: '', role: 'shop', department_code: '' };
const fieldClass = 'min-h-9 rounded-md border border-stone-300 bg-white px-3 py-1.5 text-sm outline-none focus:border-stone-900 focus:ring-2 focus:ring-stone-900/10';

export function AdminUsers() {
  const [users, setUsers] = useState([]);
  const [departments, setDepartments] = useState([]);
  const [form, setForm] = useState(emptyForm);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [query, setQuery] = useState('');

  const deptById = useMemo(() => {
    const map = {};
    for (const d of departments) map[d.id] = d;
    return map;
  }, [departments]);

  const filteredUsers = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return users;
    return users.filter((u) =>
      [u.username, u.telegram_username, u.role].some((field) => String(field || '').toLowerCase().includes(q)),
    );
  }, [users, query]);

  function load() {
    fetchUsers()
      .then((rows) => setUsers(Array.isArray(rows) ? rows : []))
      .catch((err) => setError(err instanceof Error ? err.message : String(err)));
    fetchAdminDepartments()
      .then((rows) => setDepartments(Array.isArray(rows) ? rows : []))
      .catch((err) => logWarn('admin.departments_failed', { error: String(err) }));
  }

  useEffect(load, []);

  async function onCreate(event) {
    event.preventDefault();
    setError('');
    setBusy(true);
    try {
      await createUser({
        username: form.username.trim(),
        password: form.password,
        telegram_username: form.telegram_username.trim(),
        role: form.role,
        department_code: form.department_code,
      });
      setForm(emptyForm);
      load();
    } catch (err) {
      logWarn('admin.create_user_failed', { error: String(err) });
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  async function patchUser(id, patch) {
    setError('');
    try {
      await updateUser(id, patch);
      load();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function onResetPassword(user) {
    const password = window.prompt(`Новый пароль для ${user.username || user.id}:`);
    if (!password) return;
    await patchUser(user.id, { password });
  }

  async function onDelete(user) {
    if (!window.confirm(`Удалить пользователя ${user.username || user.id}?`)) return;
    setError('');
    try {
      await deleteUser(user.id);
      load();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  function deptCodeOf(user) {
    const dept = user.department_id ? deptById[user.department_id] : null;
    return dept ? dept.code : '';
  }

  return (
    <main className="bg-[#f7f3ea] p-4 text-stone-900 sm:p-6">
      <div className="mx-auto max-w-6xl">
        <div className="mb-4 flex flex-wrap items-center justify-between gap-2">
          <h1 className="flex items-center gap-2 text-lg font-semibold">
            Пользователи
            <span className="rounded-full bg-stone-100 px-2 py-0.5 text-[12px] font-medium tabular-nums text-stone-600">{users.length}</span>
          </h1>
          <input
            className={`${fieldClass} w-full sm:w-64`}
            type="search"
            placeholder="Поиск: логин, telegram, роль…"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
        </div>

        {error && <div className="mb-4 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-[13px] text-red-800">{error}</div>}

        <form onSubmit={onCreate} className="mb-6 grid gap-2 rounded-lg border border-stone-300 bg-white p-4 shadow-sm md:grid-cols-6">
          <input className={fieldClass} placeholder="логин" value={form.username} onChange={(e) => setForm({ ...form, username: e.target.value })} />
          <input className={fieldClass} placeholder="пароль" type="password" value={form.password} onChange={(e) => setForm({ ...form, password: e.target.value })} />
          <input className={fieldClass} placeholder="telegram @username" value={form.telegram_username} onChange={(e) => setForm({ ...form, telegram_username: e.target.value })} />
          <select className={fieldClass} value={form.role} onChange={(e) => setForm({ ...form, role: e.target.value })}>
            {ROLES.map((r) => <option key={r} value={r}>{r}</option>)}
          </select>
          <select className={fieldClass} value={form.department_code} onChange={(e) => setForm({ ...form, department_code: e.target.value })}>
            <option value="">— без отдела —</option>
            {departments.map((d) => <option key={d.id} value={d.code}>{d.name} ({d.type})</option>)}
          </select>
          <Button type="submit" variant="primary" disabled={busy || !form.username.trim() || !form.password}>Добавить</Button>
        </form>

        <div className="overflow-x-auto rounded-lg border border-stone-300 bg-white shadow-sm">
          <table className="w-full text-sm">
            <thead className="bg-stone-50 text-left text-[12px] text-stone-500">
              <tr>
                <th className="px-3 py-2">ID</th>
                <th className="px-3 py-2">Логин</th>
                <th className="px-3 py-2">Telegram</th>
                <th className="px-3 py-2">TG ID</th>
                <th className="px-3 py-2">Пароль</th>
                <th className="px-3 py-2">Роль</th>
                <th className="px-3 py-2">Отдел</th>
                <th className="px-3 py-2">Создан</th>
                <th className="px-3 py-2"></th>
              </tr>
            </thead>
            <tbody>
              {filteredUsers.map((u) => (
                <tr key={u.id} className="border-t border-stone-100">
                  <td className="px-3 py-2">{u.id}</td>
                  <td className="px-3 py-2">{u.username || '—'}</td>
                  <td className="px-3 py-2">{u.telegram_username ? `@${u.telegram_username}` : '—'}</td>
                  <td className="px-3 py-2">{u.telegram_id ?? '—'}</td>
                  <td className="px-3 py-2">
                    <Button onClick={() => onResetPassword(u)}>Сбросить</Button>
                  </td>
                  <td className="px-3 py-2">
                    <select className={fieldClass} value={u.role} onChange={(e) => patchUser(u.id, { role: e.target.value })}>
                      {ROLES.map((r) => <option key={r} value={r}>{r}</option>)}
                    </select>
                  </td>
                  <td className="px-3 py-2">
                    <select className={fieldClass} value={deptCodeOf(u)} onChange={(e) => patchUser(u.id, { department_code: e.target.value })}>
                      <option value="">—</option>
                      {departments.map((d) => <option key={d.id} value={d.code}>{d.name}</option>)}
                    </select>
                  </td>
                  <td className="px-3 py-2 text-[12px] text-stone-500">{u.created_at ? u.created_at.slice(0, 10) : '—'}</td>
                  <td className="px-3 py-2">
                    <Button onClick={() => onDelete(u)} variant="danger">Удалить</Button>
                  </td>
                </tr>
              ))}
              {filteredUsers.length === 0 && (
                <tr><td colSpan={9} className="px-3 py-6 text-center text-stone-500">{query ? 'Ничего не найдено.' : 'Нет пользователей.'}</td></tr>
              )}
            </tbody>
          </table>
        </div>
        <p className="mt-3 text-[12px] text-stone-500">Пароли хранятся в виде необратимого хэша и не отображаются. Используйте «Сбросить» для задания нового.</p>
      </div>
    </main>
  );
}
