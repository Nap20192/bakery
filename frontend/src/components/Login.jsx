import { useEffect, useRef, useState } from 'react';
import { loginWithTelegram } from '../api/auth';
import { telegramBotUsername } from '../config/env';
import { setToken } from '../lib/auth';
import { logWarn } from '../lib/logger';

const WIDGET_SRC = 'https://telegram.org/js/telegram-widget.js?22';
const CALLBACK_NAME = '__bakeryOnTelegramAuth';

// Login renders the Telegram Login Widget for the plain-web build. On a
// successful Telegram callback it exchanges the signed user data for a bearer
// session token and notifies the parent.
export function Login({ onAuthenticated }) {
  const containerRef = useRef(null);
  const [error, setError] = useState('');
  const configError = telegramBotUsername ? '' : 'Не настроен VITE_TELEGRAM_BOT_USERNAME.';

  useEffect(() => {
    if (!telegramBotUsername) {
      return undefined;
    }

    window[CALLBACK_NAME] = async (telegramUser) => {
      try {
        const { token } = await loginWithTelegram(telegramUser);
        if (!token) {
          throw new Error('Сервер не вернул токен сессии.');
        }
        setToken(token);
        onAuthenticated();
      } catch (err) {
        logWarn('login.failed', { error: err instanceof Error ? err.message : String(err) });
        setError(err instanceof Error ? err.message : String(err));
      }
    };

    const script = document.createElement('script');
    script.src = WIDGET_SRC;
    script.async = true;
    script.setAttribute('data-telegram-login', telegramBotUsername);
    script.setAttribute('data-size', 'large');
    script.setAttribute('data-radius', '8');
    script.setAttribute('data-onauth', `${CALLBACK_NAME}(user)`);
    script.setAttribute('data-request-access', 'write');

    const container = containerRef.current;
    container?.appendChild(script);

    return () => {
      if (container) container.innerHTML = '';
      delete window[CALLBACK_NAME];
    };
  }, [onAuthenticated]);

  return (
    <main className="flex min-h-screen items-center justify-center bg-[#fff7df] p-4 text-stone-900">
      <div className="w-full max-w-sm rounded-xl border border-amber-200 bg-white p-6 text-center shadow-sm">
        <h1 className="mb-1 text-lg font-semibold">Заказы пекарни</h1>
        <p className="mb-5 text-[13px] leading-5 text-stone-600">
          Войдите через Telegram, чтобы продолжить.
        </p>
        <div ref={containerRef} className="flex justify-center" />
        {(configError || error) && (
          <div className="mt-4 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-[13px] text-red-800">
            {configError || error}
          </div>
        )}
      </div>
    </main>
  );
}
