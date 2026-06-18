import { createRoot } from 'react-dom/client';
import { ThemeProvider } from '@mui/material/styles';
import { App } from './app/App';
import { appTheme } from './app/theme';
import { initializeTelegramMiniApp } from './lib/telegram';
import './styles.css';

initializeTelegramMiniApp();

createRoot(document.getElementById('root')).render(
  <ThemeProvider theme={appTheme}>
    <App />
  </ThemeProvider>,
);
