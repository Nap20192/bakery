import { useState } from 'react';
import { login } from '../../api/auth';
import { setToken } from '../../lib/auth';
import { logWarn } from '../../lib/logger';
import { Button } from '../../ui/Button';

const fieldClass =
  'w-full rounded-md border border-stone-300 bg-white px-3 py-2 text-sm outline-none transition focus:border-stone-900 focus:ring-2 focus:ring-stone-900/10';

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
    <main className="flex min-h-screen items-center justify-center bg-flour p-4 text-stone-900">
      <form onSubmit={onSubmit} className="w-full max-w-sm rounded-xl border border-stone-200 bg-white p-6 shadow-sm">
        <h1 className="mb-1 text-center text-lg font-semibold text-stone-950">Заказы пекарни</h1>
        <p className="mb-5 text-center text-[13px] leading-5 text-stone-600">Войдите, чтобы продолжить.</p>
        <label className="mb-1 block text-[13px] font-medium text-stone-600" htmlFor="login-username">Логин</label>
        <input
          id="login-username"
          className={`mb-3 ${fieldClass}`}
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          autoComplete="username"
          autoFocus
        />
        <label className="mb-1 block text-[13px] font-medium text-stone-600" htmlFor="login-password">Пароль</label>
        <input
          id="login-password"
          type="password"
          className={`mb-5 ${fieldClass}`}
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          autoComplete="current-password"
        />
        <Button type="submit" variant="primary" className="w-full" disabled={loading || !username.trim() || !password}>
          {loading ? 'Вход…' : 'Войти'}
        </Button>
        {error && (
          <div className="mt-4 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-[13px] text-red-800">{error}</div>
        )}
      </form>
    </main>
  );
}
