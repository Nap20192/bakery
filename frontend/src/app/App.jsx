import { useEffect, useState } from 'react';
import { OrdersPage } from '../features/orders/OrdersPage';
import { parseRoute, pathFor } from './routes';

export function App() {
  const [route, setRoute] = useState(() => parseRoute());

  useEffect(() => {
    const onPopState = () => setRoute(parseRoute());
    window.addEventListener('popstate', onPopState);
    return () => window.removeEventListener('popstate', onPopState);
  }, []);

  function navigate(nextRoute, replace = false) {
    const path = pathFor(nextRoute);
    if (path === window.location.pathname) {
      setRoute(parseRoute(path));
      return;
    }
    if (replace) {
      window.history.replaceState(null, '', path);
    } else {
      window.history.pushState(null, '', path);
    }
    setRoute(parseRoute(path));
  }

  return <OrdersPage route={route} navigate={navigate} />;
}
