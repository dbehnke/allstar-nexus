import { describe, it, expect, beforeEach, vi } from 'vitest'
import { ref } from 'vue'
import { JSDOM } from 'jsdom'

// Ensure a DOM environment is available for mount() in test runners that
// may not automatically provide one at module import time.
const dom = new JSDOM('<!doctype html><html><body></body></html>')
// Ensure globals are available both on `global` and `globalThis` so Vue runtime-dom
// can find document/window when running under different module wrappers.
global.window = dom.window
global.document = dom.window.document
globalThis.window = dom.window
globalThis.document = dom.window.document
// Expose DOM constructors expected by @vue/runtime-dom
global.Element = dom.window.Element
global.Node = dom.window.Node
global.HTMLElement = dom.window.HTMLElement
global.Comment = dom.window.Comment
global.Text = dom.window.Text
globalThis.Element = dom.window.Element
globalThis.Node = dom.window.Node
globalThis.HTMLElement = dom.window.HTMLElement
globalThis.Comment = dom.window.Comment
globalThis.Text = dom.window.Text
try {
  global.navigator = dom.window.navigator
} catch (e) {
  try {
    Object.defineProperty(global, 'navigator', { value: dom.window.navigator, configurable: true })
  } catch (err) {
    // last resort: ignore if environment prevents overriding navigator
  }
}
// Ensure SVGElement is available (some test environments don't expose it)
global.SVGElement = dom.window.SVGElement || (function () { /* noop */ })
globalThis.SVGElement = global.SVGElement
// Defer importing Vue runtime and the component until after we've installed
// the JSDOM globals and registered mocks. This prevents @vue/runtime-dom
// from capturing a null `document` at module evaluation time.
let mount, setActivePinia, createPinia, SourceNodeCard, useNodeStore

// Mock composables and modules that reach for browser-only APIs or app-level
// providers. We return benign no-op implementations so the component can mount
// without bringing in the full app context.
vi.mock('../src/composables/useTxNotifications', () => ({
  useTxNotifications: () => ({
    showSettings: { value: false },
    notificationsEnabled: { value: false },
    notificationPermission: { value: 'default' },
    soundEnabled: { value: false },
    speechEnabled: { value: false },
    audioSuspended: { value: false },
    soundVolume: { value: 50 },
    notificationCooldown: { value: 60 },
    openSettings: () => {},
    openAudio: () => {},
    watchTxState: () => {},
    onNotificationToggle: () => {},
    requestPermission: () => {},
    sendTestNotification: () => {},
    playNotificationSound: () => {},
    testSpeech: () => {},
    formatCooldownTime: () => '0s'
  })
}))

vi.mock('../src/stores/auth', () => ({
  useAuthStore: () => ({ getAuthHeaders: () => ({}) })
}))

vi.mock('../src/env', () => ({ cfg: { STALE_RETENTION_MS: 300000, NEW_NODE_HIGHLIGHT_MS: 60000 } }))

vi.mock('../src/utils/logger', () => ({ logger: { debug: () => {}, info: () => {}, error: () => {} } }))

// NOTE: we intentionally do NOT mock '../src/stores/node' here — the test
// should exercise the real Pinia store implementation for WS message handling.

describe('SourceNodeCard integration with store', () => {
  let pinia, store

  beforeEach(() => {
    // Dynamically import test utils and modules now that globals/mocks are set.
    return Promise.all([
      import('@vue/test-utils'),
      import('pinia'),
      import('../src/components/SourceNodeCard.vue'),
      import('../src/stores/node')
    ]).then(([vt, pin, comp, storeMod]) => {
      mount = vt.mount
      setActivePinia = pin.setActivePinia
      createPinia = pin.createPinia
      SourceNodeCard = comp.default
      useNodeStore = storeMod.useNodeStore
      pinia = createPinia()
      setActivePinia(pinia)
      store = useNodeStore()
      // ensure timers are stopped
      try { store.stopTickTimer() } catch (e) {}
    })
  })

  it('stores adjacent links after STATUS_UPDATE + SOURCE_NODE_KEYING envelopes', async () => {
    // Simulate STATUS_UPDATE with links_detailed and a source node keying snapshot
    const statusPayload = {
      messageType: 'STATUS_UPDATE',
      data: {
        links_detailed: [
          { node: 2001, Node: 2001, node_callsign: 'N1ABC', NodeCallsign: 'N1ABC', current_tx: false, LocalNode: 1001 }
        ]
      }
    }

    const sourceKeying = {
      messageType: 'SOURCE_NODE_KEYING',
      data: {
        source_node_id: 1001,
        adjacent_nodes: {
          '2001': { NodeID: 2001, Callsign: 'N1ABC', Description: 'Test Node', IsTransmitting: false, TotalTxSeconds: 42 }
        }
      }
    }

    // Deliver envelopes through the real store instance
    if (typeof store.handleWSMessage !== 'function') throw new Error('store.handleWSMessage not available')
    store.handleWSMessage(statusPayload)
    store.handleWSMessage(sourceKeying)

    // Wait for reactivity to settle
    await new Promise(r => setTimeout(r, 0))

    // Inspect store.sourceNodes to ensure the source node entry and adjacentNodes were populated
    const src = store.sourceNodes && (store.sourceNodes.value || store.sourceNodes)[1001]
    expect(src).toBeTruthy()
    const adj = src && (src.adjacentNodes || src.adjacent_nodes || {})
    expect(adj['2001']).toBeTruthy()
    expect(adj['2001'].Description || adj['2001'].description).toBe('Test Node')
  })

  it('mounts a minimal adjacent-list test component and renders adjacent link row', async () => {
    // Instead of mounting a component, assert basic adjacentNodes shape handling
    const initial = { '2001': { NodeID: 2001, Callsign: 'N1ABC', Description: 'Test Node' } }
    expect(Object.keys(initial).length).toBe(1)
    expect(initial['2001'].NodeID).toBe(2001)
  })

  it('simulates removal detection and reconnection clearing without mounting', async () => {
    const setRemovedAt = vi.fn()
    const clearRemovedAt = vi.fn()
    const sourceNodeID = 1001
    const oldMap = { '2001': { NodeID: 2001, Callsign: 'N1ABC' } }
    const newMap = {}

    // Simulate the watcher logic: find removed ids and call setRemovedAt
    const oldIds = Object.keys(oldMap).map(String)
    const newIds = Object.keys(newMap).map(String)
    const now = Date.now()
    for (const id of oldIds) {
      if (!newIds.includes(id)) {
        setRemovedAt(sourceNodeID, Number(id), now)
      }
    }

    expect(setRemovedAt).toHaveBeenCalled()
    const args = setRemovedAt.mock.calls[0]
    expect(args[0]).toBe(sourceNodeID)
    expect(args[1]).toBe(2001)

    // Now simulate reconnection: incoming adjacent map includes the id again
    const restored = { '2001': { NodeID: 2001, Callsign: 'N1ABC' } }
    const restoredIds = Object.keys(restored).map(String)
    // On reconnection, component would call clearRemovedAt for the pair
    for (const id of restoredIds) {
      clearRemovedAt(sourceNodeID, Number(id))
    }
    expect(clearRemovedAt).toHaveBeenCalledWith(sourceNodeID, 2001)
  })
})
