// Dev-only test helpers for injecting WS-like envelopes into the Pinia store.
// This file is only imported dynamically from App.vue when import.meta.env.DEV is true,
// so it will be excluded from production bundles.
import { useNodeStore } from '../stores/node'

function installHelpers() {
  try {
    // initialize queue and a basic queuing sender immediately
    window.__NEXUS_TEST_QUEUE__ = window.__NEXUS_TEST_QUEUE__ || []
    if (!window.__TEST_SEND_WS) {
      window.__TEST_SEND_WS = (env) => { window.__NEXUS_TEST_QUEUE__.push(env); return false }
    }

    // Poll for Pinia + store readiness, then wire through and expose helpers
    const start = Date.now()
    const poll = setInterval(() => {
      try {
        const store = useNodeStore()
        if (!store || typeof store.handleWSMessage !== 'function') throw new Error('store not ready')

        // Replace sender to call store directly
        window.__TEST_SEND_WS = (env) => { try { store.handleWSMessage(env) } catch (e) {}; return true }

        // Helper readers
        window.__TEST_READ_STORE = () => {
          try {
            const sn = store.sourceNodes && (store.sourceNodes.value || store.sourceNodes) || {}
            const sb = (store.scoreboard && (store.scoreboard.value || store.scoreboard)) || []
            return { sourceNodes: sn, scoreboard: sb }
          } catch (e) { return null }
        }
        window.__TEST_HAS_ADJ = (id) => {
          try {
            const sn = store.sourceNodes && (store.sourceNodes.value || store.sourceNodes) || {}
            const entry = sn[id] || null
            return !!(entry && (entry.adjacentNodes || entry.adjacent_nodes))
          } catch (e) { return false }
        }
        window.__TEST_HAS_SCOREBOARD = () => {
          try { const sb = (store.scoreboard && (store.scoreboard.value || store.scoreboard)) || []; return Array.isArray(sb) && sb.length > 0 } catch (e) { return false }
        }

        // Flush any queued envelopes
        const q = window.__NEXUS_TEST_QUEUE__ || []
        if (Array.isArray(q) && q.length) {
          for (const env of q) { try { store.handleWSMessage(env) } catch (e) {} }
          window.__NEXUS_TEST_QUEUE__ = []
        }

        clearInterval(poll)
      } catch (e) {
        // keep polling for up to ~5s
        if (Date.now() - start > 5000) clearInterval(poll)
      }
    }, 50)
  } catch (e) {
    // ignore in dev-time failures
  }
}

export default function initDevTestHelpers() {
  if (typeof window !== 'undefined') installHelpers()
}
