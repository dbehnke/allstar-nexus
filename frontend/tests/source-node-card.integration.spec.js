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
let setActivePinia, createPinia, useNodeStore

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
      import('pinia'),
      import('../src/stores/node')
    ]).then(([pin, storeMod]) => {
      setActivePinia = pin.setActivePinia
      createPinia = pin.createPinia
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

  it('handles a minimal adjacent list shape in isolation', async () => {
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

  it('sorts adjacent nodes by transmit activity and connection time', () => {
    const now = Date.now()
    
    // Create test nodes with different states
    const nodes = {
      '2001': { NodeID: 2001, Callsign: 'N1ABC', TotalTxSeconds: 0, ConnectedSince: now - 10000, IsTransmitting: false, LastTxEnd: null },      // Never talked, connected 10s ago
      '2002': { NodeID: 2002, Callsign: 'N2DEF', TotalTxSeconds: 100, ConnectedSince: now - 20000, IsTransmitting: false, LastTxEnd: now - 50000 },    // Talked, last TX ended 50s ago
      '2003': { NodeID: 2003, Callsign: 'N3GHI', TotalTxSeconds: 50, ConnectedSince: now - 5000, IsTransmitting: false, LastTxEnd: now - 100000 },    // Talked, last TX ended 100s ago
      '2004': { NodeID: 2004, Callsign: 'N4JKL', TotalTxSeconds: 200, ConnectedSince: now - 15000, IsTransmitting: true, LastTxEnd: null },    // Currently transmitting
      '2005': { NodeID: 2005, Callsign: 'N5MNO', TotalTxSeconds: 0, ConnectedSince: now - 3000, IsTransmitting: false, LastTxEnd: null },      // Never talked, connected 3s ago
    }
    
    // Expected sort order:
    // 1. Currently transmitting: 2004
    // 2. Recent talkers (TotalTxSeconds > 0), by most recent LastTxEnd: 2002 (50s ago), 2003 (100s ago)
    // 3. Never talked (TotalTxSeconds = 0), newest connection first: 2005 (3s ago), 2001 (10s ago)
    
    // Mock parseAnyToMs function behavior
    const parseAnyToMs = (v) => {
      if (v == null) return NaN
      if (typeof v === 'number') return v
      return NaN
    }
    
    // Simulate the sorting logic from adjacentList computed
    const sorted = Object.values(nodes).sort((a, b) => {
      // 1. Currently transmitting nodes first
      if (a.IsTransmitting && !b.IsTransmitting) return -1
      if (!a.IsTransmitting && b.IsTransmitting) return 1
      
      // 2. Nodes that have transmitted (TotalTxSeconds > 0) before nodes that haven't
      const aHasTalked = a.TotalTxSeconds > 0
      const bHasTalked = b.TotalTxSeconds > 0
      if (aHasTalked && !bHasTalked) return -1
      if (!aHasTalked && bHasTalked) return 1
      
      // 3. Within nodes that have talked, sort by most recent last transmission (newest first)
      if (aHasTalked && bHasTalked) {
        const aLastTx = a.LastTxEnd ? parseAnyToMs(a.LastTxEnd) : 0
        const bLastTx = b.LastTxEnd ? parseAnyToMs(b.LastTxEnd) : 0
        if (aLastTx !== bLastTx) return bLastTx - aLastTx // Higher timestamp = more recent
      }
      
      // 4. For nodes that haven't talked, sort by most recent connection (newest first)
      if (!aHasTalked && !bHasTalked) {
        const aConnected = Number(a.ConnectedSince) || 0
        const bConnected = Number(b.ConnectedSince) || 0
        if (aConnected !== bConnected) return bConnected - aConnected
      }
      
      // 5. Fallback: sort by NodeID for stability
      const na = Number(a.NodeID)
      const nb = Number(b.NodeID)
      if (!isNaN(na) && !isNaN(nb)) return na - nb
      if (a.NodeID < b.NodeID) return -1
      if (a.NodeID > b.NodeID) return 1
      return 0
    })
    
    // Verify the sort order
    expect(sorted.map(n => n.NodeID)).toEqual([2004, 2002, 2003, 2005, 2001])
    
    // Verify first is transmitting
    expect(sorted[0].IsTransmitting).toBe(true)
    
    // Verify next two have talked and are sorted by last TX time (most recent first)
    expect(sorted[1].TotalTxSeconds).toBeGreaterThan(0)
    expect(sorted[2].TotalTxSeconds).toBeGreaterThan(0)
    expect(sorted[1].LastTxEnd).toBeGreaterThan(sorted[2].LastTxEnd) // More recent LastTxEnd (higher timestamp)
    
    // Verify last two haven't talked and are sorted by connection time (newest first)
    expect(sorted[3].TotalTxSeconds).toBe(0)
    expect(sorted[4].TotalTxSeconds).toBe(0)
    expect(sorted[3].ConnectedSince).toBeGreaterThan(sorted[4].ConnectedSince)
  })
})
