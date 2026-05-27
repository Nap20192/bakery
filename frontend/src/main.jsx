import { createRoot } from 'react-dom/client';
import { OrdersPage } from './features/orders/OrdersPage';
import { initializeTelegramMiniApp } from './lib/telegram';
import './styles.css';

initializeTelegramMiniApp();

createRoot(document.getElementById('root')).render(<OrdersPage />);
