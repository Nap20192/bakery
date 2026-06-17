import { createRoot } from 'react-dom/client';
import { MePage } from './features/account/Me';
import { OrdersPage } from './features/orders/OrdersPage';
import { initializeTelegramMiniApp } from './lib/telegram';
import './styles.css';

initializeTelegramMiniApp();

const route = window.location.pathname === '/me' ? <MePage /> : <OrdersPage />;

createRoot(document.getElementById('root')).render(route);
