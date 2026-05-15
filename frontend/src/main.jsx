import React from 'react';
import { createRoot } from 'react-dom/client';
import { OrdersPage } from './features/orders/OrdersPage';
import './styles.css';

createRoot(document.getElementById('root')).render(<OrdersPage />);
