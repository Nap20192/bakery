import { isWebMode } from '../../lib/auth';

const navItems = [
  { route: { name: 'orders' }, label: 'Заказы', roles: ['shop', 'baker', 'admin'] },
  { route: { name: 'orderNew' }, label: 'Новый', roles: ['shop'] },
  { route: { name: 'adminUsers' }, label: 'Пользователи', roles: ['admin'] },
  { route: { name: 'me' }, label: 'Профиль', roles: ['shop', 'baker', 'admin'] },
];

export function OrdersLayout({ viewer, active, onNavigate, onLogout, children }) {
  const role = navigationRole(viewer);
  const items = navItems.filter((item) => item.roles.includes(role));

  return (
    <main className="min-h-screen bg-[#fff7df] text-stone-900">
      <header className="sticky top-0 z-20 border-b border-stone-300 bg-[#fff7df]/95 px-3 py-2 backdrop-blur">
        <div className="mx-auto flex max-w-[1440px] items-center gap-2">
          <button
            type="button"
            onClick={() => onNavigate({ name: 'orders' })}
            className="shrink-0 text-left text-[17px] font-semibold leading-6 text-stone-950"
          >
            Bakery
          </button>
          <nav className="flex min-w-0 flex-1 gap-1 overflow-x-auto px-1">
            {items.map((item) => (
              <button
                key={item.label}
                type="button"
                onClick={() => onNavigate(item.route)}
                className={`shrink-0 rounded-md border px-3 py-1.5 text-[13px] font-medium ${
                  active === item.route.name
                    ? 'border-stone-900 bg-stone-900 text-white'
                    : 'border-stone-300 bg-[#fff7df] text-stone-800'
                }`}
              >
                {item.label}
              </button>
            ))}
          </nav>
          {isWebMode() && (
            <button
              type="button"
              onClick={onLogout}
              className="shrink-0 rounded-md border border-red-300 px-3 py-1.5 text-[13px] font-medium text-red-700"
            >
              Выйти
            </button>
          )}
        </div>
      </header>
      {children}
    </main>
  );
}

function navigationRole(viewer) {
  if (viewer?.role === 'admin') return 'admin';
  if (viewer?.department_type === 'shop' || viewer?.role === 'shop') return 'shop';
  if (viewer?.department_type === 'workshop' || viewer?.role === 'baker') return 'baker';
  return viewer?.role || '';
}
