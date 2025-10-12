// Dev-only test helpers for injecting WS-like envelopes into the Pinia store.
// This file is only imported dynamically from App.vue when import.meta.env.DEV is true,
// so it will be excluded from production bundles.
import { useNodeStore } from '../stores/node'

function installHelpers() {
  try {
    const nodeStore = useNodeStore()
    // initialize queue
    window.__NEXUS_TEST_QUEUE__ = window.__NEXUS_TEST_QUEUE__ || []

    if (!window.__TEST_SEND_WS) {
      window.__TEST_SEND_WS = (env) => {
        try {
          if (nodeStore && typeof nodeStore.handleWSMessage === 'function') {
            try { nodeStore.handleWSMessage(env) } catch (e) {}
            return true
          }
        } catch (e) {}
        window.__NEXUS_TEST_QUEUE__.push(env)
        return false
      }
    }

    window.__TEST_READ_STORE = () => {
      try {
        const sn = nodeStore.sourceNodes && (nodeStore.sourceNodes.value || nodeStore.sourceNodes) || {}
        const sb = (nodeStore.scoreboard && (nodeStore.scoreboard.value || nodeStore.scoreboard)) || []
        return { sourceNodes: sn, scoreboard: sb }
      } catch (e) { return null }
    }

    window.__TEST_HAS_ADJ = (id) => {
      try {
        const sn = nodeStore.sourceNodes && (nodeStore.sourceNodes.value || nodeStore.sourceNodes) || {}
        const entry = sn[id] || null
        return !!(entry && (entry.adjacentNodes || entry.adjacent_nodes))
      } catch (e) { return false }
    }

    window.__TEST_HAS_SCOREBOARD = () => {
      try { const sb = (nodeStore.scoreboard && (nodeStore.scoreboard.value || nodeStore.scoreboard)) || []; return Array.isArray(sb) && sb.length > 0 } catch (e) { return false }
    }

    // Flush any queued envelopes after a short delay to allow the store to initialize
    setTimeout(() => {
      try {
        const q = window.__NEXUS_TEST_QUEUE__ || []
        if (Array.isArray(q) && q.length > 0 && nodeStore && typeof nodeStore.handleWSMessage === 'function') {
          for (const e of q) {
            try { nodeStore.handleWSMessage(e) } catch (err) {}
          }
          window.__NEXUS_TEST_QUEUE__ = []
        }
      } catch (e) {}
    }, 200)

  } catch (e) {
    // ignore in dev-time failures
  }
}

export default function initDevTestHelpers() {
  if (typeof window !== 'undefined') installHelpers()
}
