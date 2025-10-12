<template>
  <div>
    <h2>Test SourceNodeCard</h2>
    <SourceNodeCard :source-node-id="1001" :data="payload" />
  </div>
</template>

<script setup>
import { computed } from 'vue'
import SourceNodeCard from '../components/SourceNodeCard.vue'
import { useNodeStore } from '../stores/node'

const nodeStore = useNodeStore()
// If the store has an entry for 1001, bind to it so WS updates reflect in the UI.
// Otherwise, provide a static fallback payload so the page renders deterministically
// for simple E2E tests.
const fallback = { source_node_id: 1001, adjacentNodes: { '2001': { NodeID: 2001, Callsign: 'N1ABC', Description: 'Fallback Node', IsTransmitting: false, ConnectedSince: Date.now() } } }
const payload = computed(() => {
  return (nodeStore && nodeStore.sourceNodes && nodeStore.sourceNodes[1001]) ? nodeStore.sourceNodes[1001] : fallback
})

// Expose test-only helpers only when running in dev or when TEST_MODE flag is set.
// This prevents test-only hooks from being present in production builds.
const _isTestDev = (typeof import.meta !== 'undefined' && import.meta.env && import.meta.env.DEV) || (typeof window !== 'undefined' && window.__NEXUS_CONFIG__ && window.__NEXUS_CONFIG__.TEST_MODE)
if (_isTestDev && typeof window !== 'undefined') {
  // For E2E tests: expose a simple global hook that accepts an envelope and forwards it
  // to the store's WebSocket message handler. Tests can call window.__TEST_SEND_WS(envelope)
  // to simulate inbound WS messages without needing a real socket.
  // @ts-ignore
  window.__TEST_SEND_WS = (env) => {
    try { nodeStore.handleWSMessage(env) } catch (e) {}
  }

  // Stub node lookup API used by SourceNodeCard so tests don't log JSON parse errors
  try {
    const origFetch = window.fetch.bind(window)
    window.fetch = async (input, init) => {
      try {
        const url = (typeof input === 'string') ? input : (input && input.url) || ''
        if (url && url.includes('/api/node-lookup')) {
          const q = (url.split('?q=')[1] || '').split('&')[0]
          const id = Number(q) || null
          const body = { results: [] }
          if (id) body.results.push({ callsign: `N${id}ABC`, node: id })
          return new Response(JSON.stringify(body), { status: 200, headers: { 'Content-Type': 'application/json' } })
        }
      } catch (e) {}
      return origFetch(input, init)
    }
  } catch (e) {}
}
</script>
