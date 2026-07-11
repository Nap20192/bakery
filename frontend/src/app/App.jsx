import { OrdersPage } from '../features/orders/OrdersPage';
import { useAppRouter } from './useAppRouter';

export function App() {
  const { route, navigate } = useAppRouter();

  return <OrdersPage route={route} navigate={navigate} />;
}
