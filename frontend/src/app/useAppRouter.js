import { useCallback, useEffect, useState } from 'react';
import { parseRoute, pathFor } from './routes';

// Lightweight router hook for the hand-rolled history API. Keeping browser
// synchronization here leaves App as a composition root rather than a stateful
// navigation component.
export function useAppRouter() {
  const [route, setRoute] = useState(() => parseRoute());

  useEffect(() => {
    const onPopState = () => setRoute(parseRoute());
    window.addEventListener('popstate', onPopState);
    return () => window.removeEventListener('popstate', onPopState);
  }, []);

  const navigate = useCallback((nextRoute, replace = false) => {
    const path = pathFor(nextRoute);
    if (path === window.location.pathname) {
      setRoute(parseRoute(path));
      return;
    }
    window.history[replace ? 'replaceState' : 'pushState'](null, '', path);
    setRoute(parseRoute(path));
  }, []);

  return { route, navigate };
}
