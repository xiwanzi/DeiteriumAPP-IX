import React from 'react';
import { createRoot } from 'react-dom/client';
import ConnectedApp from './ConnectedApp.jsx';
import './styles.css';
import './v2.css';
import './admin-v204.css';
import './desktop-admin.css';

// Older preview URLs now follow the same authenticated service path.
if (location.pathname === '/preview' || location.pathname.startsWith('/preview/')) {
  history.replaceState({}, '', location.pathname.slice(8) || '/');
}
try { localStorage.removeItem('deuterium-web-demo-v1'); } catch {}
createRoot(document.getElementById('root')).render(<ConnectedApp />);
