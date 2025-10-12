// Simple runtime config loader.
const hostConfig = (typeof window !== 'undefined' && window.__NEXUS_CONFIG__) ? window.__NEXUS_CONFIG__ : (typeof globalThis !== 'undefined' && globalThis.__NEXUS_CONFIG__ ? globalThis.__NEXUS_CONFIG__ : {})

export const cfg = Object.assign({
  WS_PATH: '/ws',
  DEFAULT_TOKEN: 'MISSING_TOKEN'
  ,
  // How long (ms) to keep removed adjacent links visible in the UI before pruning
  STALE_RETENTION_MS: 60 * 1000,
  // How long (ms) to treat a freshly connected node as "new" (highlight)
  NEW_NODE_HIGHLIGHT_MS: 60 * 1000
}, hostConfig);

import { logger } from './utils/logger'

// Exponential backoff websocket connector with jitter.
export function connectWS({ onMessage, onStatus, tokenProvider, maxDelay = 15000 }) {
  let attempt = 0;
  let closedByApp = false;

  function open() {
    const token = encodeURIComponent(tokenProvider ? tokenProvider() : '');
    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
    // Allow tests or deployments to provide a full ws:// or wss:// URL in cfg.WS_PATH.
    // If WS_PATH starts with ws:// or wss:// treat it as an absolute websocket URL.
    let url = '';
    if (typeof cfg.WS_PATH === 'string' && (cfg.WS_PATH.startsWith('ws://') || cfg.WS_PATH.startsWith('wss://'))) {
      url = `${cfg.WS_PATH}?token=${token}`;
    } else {
      url = `${protocol}//${location.host}${cfg.WS_PATH}?token=${token}`;
    }
    const ws = new WebSocket(url);
    onStatus && onStatus('connecting');
    ws.onopen = () => { attempt = 0; onStatus && onStatus('open'); };
    ws.onmessage = onMessage;
    ws.onclose = () => {
      onStatus && onStatus('closed');
      if (!closedByApp) scheduleReconnect();
    };
    ws.onerror = (e) => { onStatus && onStatus('error'); logger.error('[WS] error', e); };
  }

  function scheduleReconnect() {
    attempt++;
    const base = Math.min(maxDelay, 500 * Math.pow(2, attempt));
    const jitter = Math.random() * 0.3 * base;
    const delay = base + jitter;
    onStatus && onStatus(`reconnect_in_${Math.round(delay)}ms`);
    logger.info('[WS] reconnect scheduled', { delayMs: Math.round(delay), attempt });
    setTimeout(open, delay);
  }

  open();
  return () => { closedByApp = true; };
}