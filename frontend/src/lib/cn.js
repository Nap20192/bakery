import { clsx } from 'clsx';
import { twMerge } from 'tailwind-merge';

// cn — стандартная утилита shadcn/ui: склеивает классы и разрешает конфликты
// Tailwind-утилит (поздние побеждают). Используется всеми ui/-компонентами.
export function cn(...inputs) {
  return twMerge(clsx(inputs));
}
