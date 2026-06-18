import { useState } from 'react';
import { login } from '../api/auth';
import { setToken } from '../lib/auth';
import { logWarn } from '../lib/logger';

// Login renders the username/password form for the plain-web build. On success
// it stores the bearer session token and notifies the parent. Mini App clients
// never see this — they authenticate via Telegram initData.
export function Login({ onAuthenticated }) {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  async function onSubmit(event) {
    event.preventDefault();
    setError('');
    setLoading(true);
    try {
      const { token } = await login(username.trim(), password);
      if (!token) {
        throw new Error('Сервер не вернул токен сессии.');
      }
      setToken(token);
      onAuthenticated();
    } catch (err) {
      logWarn('login.failed', { error: err instanceof Error ? err.message : String(err) });
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="flex min-h-screen items-center justify-center bg-[#f7f3ea] p-4 text-stone-900">
      <form onSubmit={onSubmit} className="w-full max-w-sm rounded-xl border border-amber-200 bg-white p-6 shadow-sm">
        <h1 className="mb-1 text-center text-lg font-semibold">Заказы пекарни</h1>
        <p className="mb-5 text-center text-[13px] leading-5 text-stone-600">Войдите, чтобы продолжить.</p>
        <label className="mb-1 block text-[13px] text-stone-600">Логин</label>
        <input
          className="mb-3 w-full rounded-md border border-stone-300 px-3 py-2 text-sm outline-none focus:border-amber-400"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          autoComplete="username"
          autoFocus
        />
        <label className="mb-1 block text-[13px] text-stone-600">Пароль</label>
        <input
          type="password"
          className="mb-4 w-full rounded-md border border-stone-300 px-3 py-2 text-sm outline-none focus:border-amber-400"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          autoComplete="current-password"
        />
        <button
          type="submit"
          disabled={loading || !username.trim() || !password}
          className="w-full rounded-md bg-amber-500 px-3 py-2 text-sm font-medium text-white disabled:opacity-50"
        >
          {loading ? 'Вход…' : 'Войти'}
        </button>
        {error && (
          <div className="mt-4 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-[13px] text-red-800">{error}</div>
        )}
      </form>
    </main>
  );
}
