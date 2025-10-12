import { test, expect } from '@playwright/test'

// Utility to delay
const wait = (ms: number) => new Promise(r => setTimeout(r, ms))

test('SourceNodeCard reacts to WS STATUS_UPDATE and SOURCE_NODE_KEYING', async ({ page }) => {
  // TestSourceNode.vue exposes window.__TEST_SEND_WS which forwards envelopes to the Pinia store.

    page.on('console', msg => console.log('PAGE LOG ›', msg.text()))
    page.on('pageerror', err => console.log('PAGE ERROR ›', err.toString()))
  
     // Ensure TEST_MODE is set so the TestSourceNode exposes test helpers in preview builds
     await page.addInitScript(() => { (globalThis as any).__NEXUS_CONFIG__ = Object.assign((globalThis as any).__NEXUS_CONFIG__ || {}, { TEST_MODE: true }) })

  // Navigate to the Dashboard where SourceNodeCard instances are rendered
  await page.goto('/')

  // Wait for the page to expose the test hook
  const start = Date.now()
  while (true) {
    const hasHook = await page.evaluate(() => !!(globalThis as any).__TEST_SEND_WS)
    if (hasHook) break
    if (Date.now() - start > 8000) break
    await wait(100)
  }
  const hasHook = await page.evaluate(() => !!(globalThis as any).__TEST_SEND_WS)
  expect(hasHook).toBeTruthy()

    // Send STATUS_UPDATE then SOURCE_NODE_KEYING into the first fake websocket instance
    const statusUpdate = {
      messageType: 'STATUS_UPDATE',
      data: {
        node: 0,
        links: [ { node: 2001 } ]
      }
    }

    const sourceKeying = {
      messageType: 'SOURCE_NODE_KEYING',
      data: {
        source_node_id: 1001,
        adjacent_nodes: {
          '2001': { NodeID: 2001, Callsign: 'N1ABC', Description: 'WS Test Node', IsTransmitting: true, ConnectedSince: Date.now() }
        }
      }
    }

  await page.evaluate(({ s, k }: any) => {
    ;(globalThis as any).__TEST_SEND_WS(s)
    ;(globalThis as any).__TEST_SEND_WS(k)
  }, { s: statusUpdate, k: sourceKeying })

  // Allow time for client to process messages and update DOM
    await wait(800)

  // Assert the SourceNodeCard shows at least one adjacent row and a transmitting row
  const rowCount = await page.locator('table tbody tr').count()
  expect(rowCount).toBeGreaterThan(0)
  await expect(page.locator('table tbody tr td').first()).toContainText('2001')
  const txCount = await page.locator('tr.transmitting').count()
  expect(txCount).toBeGreaterThan(0)
})
