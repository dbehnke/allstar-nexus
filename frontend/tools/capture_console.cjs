const { chromium } = require('playwright');

(async () => {
  // Usage: node capture_console.cjs <url> <capture_ms>
  const url = process.argv[2] || 'http://127.0.0.1:8080/';
  const timeoutMs = parseInt(process.argv[3] || process.env.CAPTURE_MS || '45000', 10);
  console.log('Opening', url, 'capture_ms=', timeoutMs);

  const browser = await chromium.launch({ args: ['--no-sandbox', '--disable-dev-shm-usage', '--disable-gpu'] });
  const page = await browser.newPage();

  const logs = [];
  page.on('console', (msg) => {
    const text = msg.text();
    const type = msg.type();
    const entry = { type, text, location: msg.location() };
    logs.push(entry);
    // Echo so we can follow progress in the terminal
    console.log('[PAGE_CONSOLE]', type, text);
  });

  // Inject a small script before any page code runs so we can capture raw WebSocket messages
  await page.addInitScript(() => {
    try {
      const OriginalWebSocket = window.WebSocket;
      function WrappedWebSocket(url, protocols) {
        const ws = new OriginalWebSocket(url, protocols);
        ws.addEventListener('message', function (ev) {
          try { console.log('[WS RAW RECV]', typeof ev.data === 'string' ? ev.data : JSON.stringify(ev.data)); } catch (e) { console.error('ws wrap log err', e && e.message); }
        });
        return ws;
      }
      WrappedWebSocket.prototype = OriginalWebSocket.prototype;
      WrappedWebSocket.CONNECTING = OriginalWebSocket.CONNECTING;
      WrappedWebSocket.OPEN = OriginalWebSocket.OPEN;
      WrappedWebSocket.CLOSING = OriginalWebSocket.CLOSING;
      WrappedWebSocket.CLOSED = OriginalWebSocket.CLOSED;
      window.WebSocket = WrappedWebSocket;
    } catch (e) {
      // ignore
    }
  });

  try {
    await page.goto(url, { waitUntil: 'load', timeout: 30000 });
  } catch (e) {
    console.error('goto error', e && e.message);
  }

  // capture for configured milliseconds
  await page.waitForTimeout(timeoutMs);

  await browser.close();

  console.log('===CAPTURED LOGS JSON START===');
  console.log(JSON.stringify(logs, null, 2));
  console.log('===CAPTURED LOGS JSON END===');
  process.exit(0);
})();
