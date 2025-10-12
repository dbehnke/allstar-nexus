import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useNodeStore } from '../src/stores/node'
import { useUIStore } from '../src/stores/ui'

// Mock fetch globally
global.fetch = vi.fn()

describe('node store scoreboard behaviors', () => {
  let pinia
  let store

  beforeEach(() => {
    pinia = createPinia()
    setActivePinia(pinia)
    store = useNodeStore()
    // stub UI store toast behavior
    const ui = useUIStore()
    ui.addToast = vi.fn()
    // debug: list store keys to help diagnose missing methods
    // eslint-disable-next-line no-console
    console.debug('STORE KEYS:', Object.keys(store))
    // reset store internals
    store.scoreboard = []
    vi.clearAllMocks()
  })

  afterEach(() => {
    try { store.stopScoreboardPoll() } catch (e) {}
  })

  it('applies embedded scoreboard from GAMIFICATION_TALLY_COMPLETED', () => {
    const msg = { messageType: 'GAMIFICATION_TALLY_COMPLETED', data: { scoreboard: [{ callsign: 'ABC', experience_points: 10 }] } }
    store.handleWSMessage(msg)
    expect(store.scoreboard.length).toBe(1)
    expect(store.scoreboard[0].callsign).toBe('ABC')
  })

  it('handleWSMessage triggers a toast when scoreboard payload present', () => {
    const ui = useUIStore()
    const msg = { messageType: 'GAMIFICATION_TALLY_COMPLETED', data: { scoreboard: [{ callsign: 'TOAST', experience_points: 5 }] } }
    store.handleWSMessage(msg)
    expect(ui.addToast).toHaveBeenCalled()
  })

  it('malformed WS payload falls back to HTTP fetch', async () => {
    // Simulate malformed payload (null data)
    store.scoreboard = []
    global.fetch.mockResolvedValue({ json: async () => ({ scoreboard: [{ callsign: 'FALLBACK', experience_points: 1 }] }) })
    const msg = { messageType: 'GAMIFICATION_TALLY_COMPLETED', data: null }
    store.handleWSMessage(msg)
    // allow async fallback fetch to run
    await new Promise(r => setTimeout(r, 0))
    expect(global.fetch).toHaveBeenCalled()
  })

  it('failed fetch does not throw and leaves scoreboard empty', async () => {
    store.scoreboard = []
    global.fetch.mockRejectedValue(new Error('network'))
    store.queueScoreboardReload(10, 5)
    await new Promise(r => setTimeout(r, 30))
    // no unhandled rejection; scoreboard remains empty
    expect(store.scoreboard.length).toBe(0)
  })

  it('queueScoreboardReload skips fetch when WS populated', async () => {
    // populate scoreboard as if WS did it
    store.scoreboard = [{ callsign: 'X', experience_points: 1 }]
    // stub fetch to fail if called
    global.fetch.mockImplementation(() => { throw new Error('should not be called') })
    store.queueScoreboardReload(50, 10)
    // wait 120ms for timer to elapse
    await new Promise(r => setTimeout(r, 120))
    expect(global.fetch).not.toHaveBeenCalled()
  })

  it('queueScoreboardReload triggers fetch when scoreboard empty', async () => {
    store.scoreboard = []
    global.fetch.mockResolvedValue({ json: async () => ({ scoreboard: [{ callsign: 'Y', experience_points: 2 }] }) })
    store.queueScoreboardReload(50, 10)
    await new Promise(r => setTimeout(r, 120))
    expect(global.fetch).toHaveBeenCalled()
    // since fetchScoreboard sets store.scoreboard asynchronously, allow microtask
    await new Promise(r => setTimeout(r, 0))
    expect(store.scoreboard.length).toBeGreaterThan(0)
  })

  it('startScoreboardPoll does initial fetch when empty and polls', async () => {
    store.scoreboard = []
    // make fetch return different values each call
    let call = 0
    global.fetch.mockImplementation(() => ({ json: async () => ({ scoreboard: [{ callsign: `P${call++}`, experience_points: call }] }) }))
    store.startScoreboardPoll(100, 5) // poll every 100ms for test
    // wait 250ms to allow a couple polls
    await new Promise(r => setTimeout(r, 250))
    expect(global.fetch).toHaveBeenCalled()
    expect(Array.isArray(store.scoreboard)).toBe(true)
    store.stopScoreboardPoll()
  })

  it('STATUS_UPDATE with links (ids) maps to links array', () => {
    const status = { messageType: 'STATUS_UPDATE', data: { links: [3001, 3002, 3003] } }
    store.handleWSMessage(status)
    expect(Array.isArray(store.links)).toBe(true)
    // store.links may be a ref or array; normalize access
    const linksVal = store.links && (store.links.value || store.links)
    expect(linksVal.length).toBe(3)
    expect(linksVal[0].node || linksVal[0].node).toBe(3001)
  })

  it('SOURCE_NODE_KEYING normalizes adjacentNodes and preserves NodeID numeric', () => {
    const msg = { messageType: 'SOURCE_NODE_KEYING', data: { source_node_id: 5001, adjacent_nodes: { '6001': { Callsign: 'CALL', Description: 'Desc' } } } }
    store.handleWSMessage(msg)
    const src = (store.sourceNodes && (store.sourceNodes.value || store.sourceNodes))[5001]
    expect(src).toBeTruthy()
    const adj = src.adjacentNodes || src.adjacent_nodes
    expect(adj['6001']).toBeTruthy()
    expect(Number(adj['6001'].NodeID)).toBe(6001)
  })
})
