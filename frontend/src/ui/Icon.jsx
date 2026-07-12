const ICON_PATHS = {
  orders: 'M4 6.5A2.5 2.5 0 0 1 6.5 4h11A2.5 2.5 0 0 1 20 6.5v11a2.5 2.5 0 0 1-2.5 2.5h-11A2.5 2.5 0 0 1 4 17.5v-11Zm4 1.5h8M8 12h8M8 16h5',
  plus: 'M12 5v14M5 12h14',
  users: 'M16 11a4 4 0 1 0-8 0 4 4 0 0 0 8 0Zm-10 9a6 6 0 0 1 12 0M18 8a3 3 0 0 1 0 6M20.5 20a5 5 0 0 0-3.5-4.8',
  user: 'M16 8a4 4 0 1 0-8 0 4 4 0 0 0 8 0ZM5 20a7 7 0 0 1 14 0',
  logout: 'M14 8V6a2 2 0 0 0-2-2H6a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h6a2 2 0 0 0 2-2v-2M10 12h10M17 9l3 3-3 3',
  copy: 'M8 7.5V5.75A1.75 1.75 0 0 1 9.75 4h8.5A1.75 1.75 0 0 1 20 5.75v8.5A1.75 1.75 0 0 1 18.25 16H16.5M5.75 8h8.5A1.75 1.75 0 0 1 16 9.75v8.5A1.75 1.75 0 0 1 14.25 20h-8.5A1.75 1.75 0 0 1 4 18.25v-8.5A1.75 1.75 0 0 1 5.75 8Z',
  eye: 'M2.5 12s3.5-6 9.5-6 9.5 6 9.5 6-3.5 6-9.5 6-9.5-6-9.5-6Zm9.5 3a3 3 0 1 0 0-6 3 3 0 0 0 0 6Z',
  select: 'M4 5.5A1.5 1.5 0 0 1 5.5 4h13A1.5 1.5 0 0 1 20 5.5v13a1.5 1.5 0 0 1-1.5 1.5h-13A1.5 1.5 0 0 1 4 18.5v-13Zm4 6 2.5 2.5L16 8.5',
  calculator: 'M7 3h10a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2Zm2 4h6M9 11h.01M12 11h.01M15 11h.01M9 15h.01M12 15h.01M15 15h.01',
  chevronLeft: 'm15 18-6-6 6-6',
  chevronRight: 'm9 18 6-6-6-6',
  close: 'M6 6l12 12M18 6 6 18',
  star: 'M12 3.5l2.6 5.27 5.82.85-4.21 4.1.99 5.79L12 16.77 6.8 19.5l.99-5.79-4.21-4.1 5.82-.85z',
  table: 'M4 6.5A2.5 2.5 0 0 1 6.5 4h11A2.5 2.5 0 0 1 20 6.5v11a2.5 2.5 0 0 1-2.5 2.5h-11A2.5 2.5 0 0 1 4 17.5v-11ZM4 9.5h16M4 14.5h16M9.5 9.5V20',
  journal: 'M6 3.5h12A1.5 1.5 0 0 1 19.5 5v14a1.5 1.5 0 0 1-1.5 1.5H6A1.5 1.5 0 0 1 4.5 19V5A1.5 1.5 0 0 1 6 3.5ZM8.5 3.5v17M12 8h4M12 12h4',
  menu: 'M4 7h16M4 12h16M4 17h16',
};

// The Figma desktop/mobile icon sets use a light outlined treatment. Individual
// call sites may still opt into a heavier stroke when the action needs it.
export function Icon({ name, size = 18, className = '', strokeWidth = 1.5, filled = false }) {
  const path = ICON_PATHS[name] || ICON_PATHS.orders;

  return (
    <svg
      aria-hidden="true"
      className={className}
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill={filled ? 'currentColor' : 'none'}
      stroke="currentColor"
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={strokeWidth}
    >
      <path d={path} />
    </svg>
  );
}
