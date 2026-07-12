import { isWebMode } from '../../lib/auth';
import { Button } from '../../ui/Button';

function Row({ label, value }) {
  return (
    <div className="flex justify-between gap-4 border-t border-stone-100 py-2 text-sm first:border-t-0">
      <span className="text-stone-500">{label}</span>
      <span className="font-medium text-stone-900">{value || '—'}</span>
    </div>
  );
}

// MePanel renders the current user's profile inside the app shell. It reads the
// viewer already loaded by the page and lets web users log out (Mini App users
// stay signed in via Telegram).
export function MePanel({ viewer, onLogout }) {
  const v = viewer || {};
  return (
    <section className="bg-flour p-4 text-stone-900 sm:p-6" aria-labelledby="profile-title">
      <div className="mx-auto max-w-md">
        <div className="rounded-xl border border-stone-200 bg-white p-5 shadow-sm sm:p-6">
          <h1 id="profile-title" className="mb-4 text-page font-semibold text-stone-950">Профиль</h1>
          <Row label="Роль" value={v.role} />
          <Row label="Telegram" value={v.telegram_username ? `@${v.telegram_username}` : ''} />
          <Row label="Отдел" value={v.department_name} />
          <Row label="Тип" value={v.department_type} />
          {isWebMode() ? (
            <Button onClick={onLogout} variant="danger" className="mt-5 w-full">
              Выйти
            </Button>
          ) : (
            <p className="mt-4 text-center text-note text-stone-500">Вход через Telegram — выход не требуется.</p>
          )}
        </div>
      </div>
    </section>
  );
}
