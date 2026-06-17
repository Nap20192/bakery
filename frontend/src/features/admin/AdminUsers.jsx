import { useEffect, useMemo, useState } from 'react';
import { createUser, fetchAdminDepartments, fetchUsers, updateUser } from '../../api/users';
import { logWarn } from '../../lib/logger';

const ROLES = ['admin', 'shop', 'baker', 'user'];

const emptyForm = { username: '', password: '', role: 'shop', department_code: '' };

export function AdminUsers({ onLogout }) {
  const [users, setUsers] = useState([]);
  const [departments, setDepartments] = useState([]);
  const [form, setForm] = useState(emptyForm);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  const deptById = useMemo(() => {
    const map = {};
    for (const d of departments) map[d.id] = d;
    return map;
  }, [departments]);

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

  function deptCodeOf(user) {
    const dept = user.department_id ? deptById[user.department_id] : null;
    return dept ? dept.code : '';
  }

  return (
    <main className="min-h-screen bg-[#fff7df] p-4 text-stone-900 sm:p-6">
      <div className="mx-auto max-w-4xl">
        <div className="mb-4 flex items-center justify-between">
          <h1 className="text-lg font-semibold">Пользователи</h1>
          <button onClick={onLogout} className="rounded-md border border-stone-300 px-3 py-1.5 text-sm">Выйти</button>
        </div>

        {error && <div className="mb-4 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-[13px] text-red-800">{error}</div>}

        <form onSubmit={onCreate} className="mb-6 grid gap-2 rounded-xl border border-amber-200 bg-white p-4 sm:grid-cols-5">
          <input className="rounded-md border border-stone-300 px-2 py-1.5 text-sm" placeholder="логин" value={form.username} onChange={(e) => setForm({ ...form, username: e.target.value })} />
          <input className="rounded-md border border-stone-300 px-2 py-1.5 text-sm" placeholder="пароль" type="password" value={form.password} onChange={(e) => setForm({ ...form, password: e.target.value })} />
          <select className="rounded-md border border-stone-300 px-2 py-1.5 text-sm" value={form.role} onChange={(e) => setForm({ ...form, role: e.target.value })}>
            {ROLES.map((r) => <option key={r} value={r}>{r}</option>)}
          </select>
          <select className="rounded-md border border-stone-300 px-2 py-1.5 text-sm" value={form.department_code} onChange={(e) => setForm({ ...form, department_code: e.target.value })}>
            <option value="">— без отдела —</option>
            {departments.map((d) => <option key={d.id} value={d.code}>{d.name} ({d.type})</option>)}
          </select>
          <button type="submit" disabled={busy || !form.username.trim() || !form.password} className="rounded-md bg-amber-500 px-3 py-1.5 text-sm font-medium text-white disabled:opacity-50">Добавить</button>
        </form>

        <div className="overflow-x-auto rounded-xl border border-amber-200 bg-white">
          <table className="w-full text-sm">
            <thead className="bg-amber-50 text-left text-[12px] text-stone-500">
              <tr><th className="px-3 py-2">ID</th><th className="px-3 py-2">Логин</th><th className="px-3 py-2">Telegram</th><th className="px-3 py-2">Роль</th><th className="px-3 py-2">Отдел</th></tr>
            </thead>
            <tbody>
              {users.map((u) => (
                <tr key={u.id} className="border-t border-stone-100">
                  <td className="px-3 py-2">{u.id}</td>
                  <td className="px-3 py-2">{u.username || '—'}</td>
                  <td className="px-3 py-2">{u.telegram_username || '—'}</td>
                  <td className="px-3 py-2">
                    <select className="rounded-md border border-stone-300 px-2 py-1 text-sm" value={u.role} onChange={(e) => patchUser(u.id, { role: e.target.value })}>
                      {ROLES.map((r) => <option key={r} value={r}>{r}</option>)}
                    </select>
                  </td>
                  <td className="px-3 py-2">
                    <select className="rounded-md border border-stone-300 px-2 py-1 text-sm" value={deptCodeOf(u)} onChange={(e) => patchUser(u.id, { department_code: e.target.value })}>
                      <option value="">—</option>
                      {departments.map((d) => <option key={d.id} value={d.code}>{d.name}</option>)}
                    </select>
                  </td>
                </tr>
              ))}
              {users.length === 0 && <tr><td colSpan={5} className="px-3 py-6 text-center text-stone-500">Нет пользователей.</td></tr>}
            </tbody>
          </table>
        </div>
      </div>
    </main>
  );
}
