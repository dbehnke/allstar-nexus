// Simple logger with runtime-toggleable debug mode.
// Debug can be enabled via any of the following:
// - VITE_DEBUG=true or VITE_APP_DEBUG=true at build time
// - window.__NEXUS_CONFIG__.DEBUG = true at runtime
// - localStorage.setItem('nexus_debug', 'true') at runtime

const getInitialDebug = () => {
  try {
    const fromStorage = typeof localStorage !== 'undefined' && localStorage.getItem('nexus_debug');
    if (fromStorage === 'true') return true;
  } catch {}
  try {
    // Vite env flags
    if (typeof import.meta !== 'undefined' && import.meta.env) {
      if (String(import.meta.env.VITE_DEBUG) === 'true') return true;
      if (String(import.meta.env.VITE_APP_DEBUG) === 'true') return true;
    }
  } catch {}
  try {
    if (typeof window !== 'undefined' && window.__NEXUS_CONFIG__ && window.__NEXUS_CONFIG__.DEBUG) return true;
  } catch {}
  return false;
};

let debugEnabled = getInitialDebug();

const setDebug = (on) => {
  debugEnabled = !!on;
  try { localStorage.setItem('nexus_debug', debugEnabled ? 'true' : 'false'); } catch {}
};

const isDebug = () => debugEnabled;

// Always-on levels: warn/error. Gated levels: info/debug.
const logger = {
  debug: (...args) => { if (debugEnabled) { try { console.debug(...args); } catch {} } },
  info:  (...args) => { if (debugEnabled) { try { console.info(...args); } catch {} } },
  warn:  (...args) => { try { console.warn(...args); } catch {} },
  error: (...args) => { try { console.error(...args); } catch {} },
  setDebug,
  isDebug,
};

// Expose toggles for convenience in the browser console
try {
  if (typeof window !== 'undefined') {
    window.NexusLogger = {
      enable: () => setDebug(true),
      disable: () => setDebug(false),
      isDebug,
      logger,
    };
  }
} catch {}

export { logger, setDebug, isDebug };
