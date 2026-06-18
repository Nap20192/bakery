const paths = {
  orders: 'M4 6.5A2.5 2.5 0 0 1 6.5 4h11A2.5 2.5 0 0 1 20 6.5v11a2.5 2.5 0 0 1-2.5 2.5h-11A2.5 2.5 0 0 1 4 17.5v-11Zm4 1.5h8M8 12h8M8 16h5',
  plus: 'M12 5v14M5 12h14',
  users: 'M16 11a4 4 0 1 0-8 0 4 4 0 0 0 8 0Zm-10 9a6 6 0 0 1 12 0M18 8a3 3 0 0 1 0 6M20.5 20a5 5 0 0 0-3.5-4.8',
  user: 'M16 8a4 4 0 1 0-8 0 4 4 0 0 0 8 0ZM5 20a7 7 0 0 1 14 0',
  logout: 'M14 8V6a2 2 0 0 0-2-2H6a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h6a2 2 0 0 0 2-2v-2M10 12h10M17 9l3 3-3 3',
  eye: 'M2.5 12s3.5-6 9.5-6 9.5 6 9.5 6-3.5 6-9.5 6-9.5-6-9.5-6Zm9.5 3a3 3 0 1 0 0-6 3 3 0 0 0 0 6Z',
  select: 'M4 5.5A1.5 1.5 0 0 1 5.5 4h13A1.5 1.5 0 0 1 20 5.5v13a1.5 1.5 0 0 1-1.5 1.5h-13A1.5 1.5 0 0 1 4 18.5v-13Zm4 6 2.5 2.5L16 8.5',
  calculator: 'M7 3h10a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2Zm2 4h6M9 11h.01M12 11h.01M15 11h.01M9 15h.01M12 15h.01M15 15h.01',
  chevronLeft: 'm15 18-6-6 6-6',
  chevronRight: 'm9 18 6-6-6-6',
  close: 'M6 6l12 12M18 6 6 18',
};

export function Icon({ name, size = 18, className = '', strokeWidth = 2 }) {
  return (
    <svg
      aria-hidden="true"
      className={className}
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={strokeWidth}
    >
      <path d={paths[name] || paths.orders} />
    </svg>
  );
}
