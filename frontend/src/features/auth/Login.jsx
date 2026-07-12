import { useState } from 'react';
import { login } from '../../api/auth';
import { setToken } from '../../lib/auth';
import { logWarn } from '../../lib/logger';
import { Button } from '../../ui/Button';
import { ErrorBanner } from '../../ui/ErrorBanner';
import { controlClass } from '../../ui/Field';

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
    <main className="flex min-h-screen items-center justify-center bg-background p-4 text-foreground">
      <form onSubmit={onSubmit} className="ui-panel w-full max-w-sm" data-testid="login-form">
        <h1 className="mb-2 text-center text-title text-foreground">Заказы пекарни</h1>
        <p className="mb-6 text-center text-body leading-6 text-text-secondary">Войдите, чтобы продолжить.</p>
        <label className="ui-field__label" htmlFor="login-username">Логин</label>
        <input
          id="login-username"
          className={`mb-4 ${controlClass}`}
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          autoComplete="username"
          aria-invalid={Boolean(error)}
          aria-describedby={error ? 'login-error' : undefined}
          autoFocus
        />
        <label className="ui-field__label" htmlFor="login-password">Пароль</label>
        <input
          id="login-password"
          type="password"
          className={`mb-6 ${controlClass}`}
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          autoComplete="current-password"
          aria-invalid={Boolean(error)}
          aria-describedby={error ? 'login-error' : undefined}
        />
        <Button type="submit" variant="primary" className="w-full" loading={loading} disabled={!username.trim() || !password}>
          Войти
        </Button>
        <ErrorBanner id="login-error" error={error} className="mt-4" />
      </form>
    </main>
  );
}
