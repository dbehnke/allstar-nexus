import { test, expect } from '@playwright/test'

test('SourceNodeCard renders in browser', async ({ page }) => {
  // Forward browser console and uncaught errors into test logs for easier debugging
  page.on('console', msg => console.log('PAGE LOG ›', msg.text()))
  page.on('pageerror', err => console.log('PAGE ERROR ›', err.toString()))

  // Ensure TEST_MODE is set so the TestSourceNode exposes test helpers in preview builds
  await page.addInitScript(() => { globalThis.__NEXUS_CONFIG__ = Object.assign(globalThis.__NEXUS_CONFIG__ || {}, { TEST_MODE: true }) })
  await page.goto('/')
  // Wait for dashboard to render by waiting for the Scoreboard heading or a known dashboard element
  await page.waitForSelector('h3', { timeout: 5000 })
  // The SourceNodeCard will render when a SOURCE_NODE_KEYING envelope is injected, so send one via the test helper
  const start = Date.now()
  while (true) {
    const has = await page.evaluate(() => typeof (globalThis as any).__TEST_SEND_WS === 'function')
    if (has) break
    if (Date.now() - start > 5000) break
    await new Promise(r => setTimeout(r, 100))
  }
  // Send a STATUS_UPDATE and a SOURCE_NODE_KEYING envelope to populate the store
  const statusUpdate = { messageType: 'STATUS_UPDATE', data: { node: 0, links: [2001] } }
  const sourceKeying = { messageType: 'SOURCE_NODE_KEYING', data: { source_node_id: 1001, adjacent_nodes: { '2001': { NodeID: 2001, Callsign: 'N1ABC', Description: 'Test Node', IsTransmitting: true, ConnectedSince: Date.now() } } } }
  await page.evaluate((payload: any) => {
    ;(globalThis as any).__TEST_SEND_WS && (globalThis as any).__TEST_SEND_WS(payload.s)
    ;(globalThis as any).__TEST_SEND_WS && (globalThis as any).__TEST_SEND_WS(payload.k)
  }, { s: statusUpdate, k: sourceKeying })
  // Wait for any SourceNodeCard table to appear
  await page.waitForSelector('table tbody tr', { timeout: 5000 })
})
