import { createRoot } from 'react-dom/client';
import { App } from './app/App';
import { initializeTelegramMiniApp } from './lib/telegram';
import './styles.css';
import './ui/ui.css';

initializeTelegramMiniApp();

createRoot(document.getElementById('root')).render(<App />);
