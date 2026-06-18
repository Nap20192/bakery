import { useEffect, useState } from 'react';
import { ApiError } from '../../api/client';
import { fetchMe } from '../../api/orders';
import { clearToken, isWebMode } from '../../lib/auth';

function Row({ label, value }) {
  return (
    <div className="flex justify-between gap-4 border-t border-stone-100 py-2 text-sm">
      <span className="text-stone-500">{label}</span>
      <span className="font-medium">{value || '—'}</span>
    </div>
  );
}

// MePage is the standalone profile route (/me): shows the current user and lets
// web users log out. It fetches /me itself so it works as an independent route.
export function MePage() {
  const [me, setMe] = useState(null);
  const [error, setError] = useState('');

  useEffect(() => {
    fetchMe()
      .then(setMe)
      .catch((err) => {
        if (err instanceof ApiError && err.status === 401) {
          goHome();
          return;
        }
        setError(err instanceof Error ? err.message : String(err));
      });
  }, []);

  function goHome() {
    window.location.assign('/');
  }

  function logout() {
    clearToken();
    window.location.assign('/');
  }

  const v = me || {};
  return (
    <main className="flex min-h-screen items-center justify-center bg-[#f7f3ea] p-4 text-stone-900">
      <div className="w-full max-w-sm rounded-xl border border-amber-200 bg-white p-6 shadow-sm">
        <h1 className="mb-4 text-lg font-semibold">Профиль</h1>
        {error && <div className="mb-4 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-[13px] text-red-800">{error}</div>}
        <Row label="Роль" value={v.role} />
        <Row label="Telegram" value={v.telegram_username ? `@${v.telegram_username}` : ''} />
        <Row label="Отдел" value={v.department_name} />
        <Row label="Тип" value={v.department_type} />
        <div className="mt-5 flex gap-2">
          <button onClick={goHome} className="flex-1 rounded-md border border-stone-300 px-3 py-2 text-sm">Назад</button>
          {isWebMode() && (
            <button onClick={logout} className="flex-1 rounded-md bg-red-500 px-3 py-2 text-sm font-medium text-white">Выйти</button>
          )}
        </div>
        {!isWebMode() && <p className="mt-3 text-center text-[12px] text-stone-500">Вход через Telegram — выход не требуется.</p>}
      </div>
    </main>
  );
}
