import { useEffect, useState } from 'react';
import { MePage } from '../features/account/Me';
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

  if (route.name === 'me') {
    return <MePage />;
  }

  return <OrdersPage route={route} navigate={navigate} />;
}
