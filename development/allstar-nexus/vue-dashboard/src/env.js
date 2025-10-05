// Simple runtime config loader.
export const cfg = Object.assign({
  WS_PATH: '/ws',
  DEFAULT_TOKEN: 'MISSING_TOKEN'
}, window.__NEXUS_CONFIG__ || {});

// Exponential backoff websocket connector with jitter.
export function connectWS({ onMessage, onStatus, tokenProvider, maxDelay = 15000 }) {
  let attempt = 0;
  let closedByApp = false;

  function open() {
    const token = encodeURIComponent(tokenProvider ? tokenProvider() : '');
    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
    const url = `${protocol}//${location.host}${cfg.WS_PATH}?token=${token}`;
    const ws = new WebSocket(url);
    onStatus && onStatus('connecting');
    ws.onopen = () => { attempt = 0; onStatus && onStatus('open'); };
    ws.onmessage = onMessage;
    ws.onclose = () => {
      onStatus && onStatus('closed');
      if (!closedByApp) scheduleReconnect();
    };
    ws.onerror = (e) => { onStatus && onStatus('error'); console.error('ws error', e); };
  }

  function scheduleReconnect() {
    attempt++;
    const base = Math.min(maxDelay, 500 * Math.pow(2, attempt));
    const jitter = Math.random() * 0.3 * base;
    const delay = base + jitter;
    onStatus && onStatus(`reconnect_in_${Math.round(delay)}ms`);
    setTimeout(open, delay);
  }

  open();
  return () => { closedByApp = true; };
}