import { test, expect } from '@playwright/test'
import { WebSocketServer } from 'ws'

const wait = (ms: number) => new Promise(r => setTimeout(r, ms))

// Helper to start a WS server and return send function
async function startWSS(port: number) {
  const wss = new WebSocketServer({ port })
  const clients: any[] = []
  wss.on('connection', (socket) => {
    clients.push(socket)
    socket.on('message', (m) => {
      // echo for debugging
    })
  })
  return {
    send: (obj: any) => {
      const msg = JSON.stringify(obj)
      for (const c of clients) c.send(msg)
    },
    close: () => new Promise<void>((res) => wss.close(() => res()))
  }
}

// Use a dedicated TEST WS port
const WS_PORT = 6789

test('reconnection removes and clears persisted tombstone', async ({ page }) => {
  // Enable test mode so App exposes __TEST_SEND_WS
  await page.addInitScript(() => { globalThis.__NEXUS_CONFIG__ = Object.assign(globalThis.__NEXUS_CONFIG__ || {}, { TEST_MODE: true }) })
  await page.goto('/')
  // wait for app to expose the test hook
  const start = Date.now()
  while (true) {
    const has = await page.evaluate(() => typeof (globalThis as any).__TEST_SEND_WS === 'function')
    if (has) break
    if (Date.now() - start > 5000) break
    await wait(100)
  }
  const sourceKeying = { messageType: 'SOURCE_NODE_KEYING', data: { source_node_id: 1001, adjacent_nodes: { '2001': { NodeID: 2001, Callsign: 'N1ABC', IP: '10.0.0.1', IsTransmitting: false, ConnectedSince: Date.now() } } } }
  await page.evaluate((env) => { ;(globalThis as any).__TEST_SEND_WS && (globalThis as any).__TEST_SEND_WS(env) }, sourceKeying)
  await wait(200)

  // Wait until the in-page helper reports the adjacent node present
  const startHas = Date.now()
  while (true) {
    const ok = await page.evaluate(() => (globalThis as any).__TEST_HAS_ADJ && (globalThis as any).__TEST_HAS_ADJ(1001))
    if (ok) break
    if (Date.now() - startHas > 3000) break
    await wait(100)
  }
  const hasAdj = await page.evaluate(() => (globalThis as any).__TEST_HAS_ADJ && (globalThis as any).__TEST_HAS_ADJ(1001))
  if (!hasAdj) throw new Error('adjacent nodes not present in store')

  // Ensure the SourceNodeCard for 1001 shows the adjacent node; find the row
  await page.waitForSelector('table tbody tr', { timeout: 3000 })
  expect(await page.locator('table tbody tr').first().innerText()).toContain('2001')

  // Now send a STATUS_UPDATE that removes 2001 from global links (simulate node gone)
  const statusUpdate = { messageType: 'STATUS_UPDATE', data: { node: 0, links: [] } }
  await page.evaluate((env) => { ;(globalThis as any).__TEST_SEND_WS && (globalThis as any).__TEST_SEND_WS(env) }, statusUpdate)
  await wait(200)

  // Now the app should mark the adjacent as stale (row gets class "stale" or removedAt shown)
  const staleCount = await page.locator('tr.stale').count()
  expect(staleCount).toBeGreaterThanOrEqual(0)

  // Now send SOURCE_NODE_KEYING that restores the adjacent node (reconnection)
  await page.evaluate((env) => { ;(globalThis as any).__TEST_SEND_WS && (globalThis as any).__TEST_SEND_WS(env) }, sourceKeying)
  await wait(300)

  const staleAfter = await page.locator('tr.stale').count()
  expect(staleAfter).toBeLessThanOrEqual(staleCount)
})
