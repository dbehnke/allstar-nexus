import { test, expect } from '@playwright/test'
import { WebSocketServer } from 'ws'

const wait = (ms: number) => new Promise(r => setTimeout(r, ms))

async function startWSS(port: number) {
  const wss = new WebSocketServer({ port })
  const clients: any[] = []
  wss.on('connection', (socket) => {
    clients.push(socket)
  })
  return {
    send: (obj: any) => {
      const msg = JSON.stringify(obj)
      for (const c of clients) c.send(msg)
    },
    close: () => new Promise<void>((res) => wss.close(() => res()))
  }
}

const WS_PORT = 6790

test('embedded scoreboard in GAMIFICATION_TALLY_COMPLETED is used by dashboard', async ({ page }) => {
  await page.addInitScript(() => { globalThis.__NEXUS_CONFIG__ = Object.assign(globalThis.__NEXUS_CONFIG__ || {}, { TEST_MODE: true }) })
  await page.goto('/')
  // wait for test hook
  const start = Date.now()
  while (true) {
    const has = await page.evaluate(() => typeof (globalThis as any).__TEST_SEND_WS === 'function')
    if (has) break
    if (Date.now() - start > 5000) break
    await wait(100)
  }
  // The node store expects data.scoreboard to be an array of entries.
  const scoreboard = [ { callsign: 'N1TEST', xp: 123, rank: 1 } ]
  const envelope = { messageType: 'GAMIFICATION_TALLY_COMPLETED', data: { scoreboard } }
  await page.evaluate((env) => { ;(globalThis as any).__TEST_SEND_WS && (globalThis as any).__TEST_SEND_WS(env) }, envelope)

  await wait(300)

  // Wait until the in-page helper reports scoreboard present
  const startHas = Date.now()
  while (true) {
    const ok = await page.evaluate(() => (globalThis as any).__TEST_HAS_SCOREBOARD && (globalThis as any).__TEST_HAS_SCOREBOARD())
    if (ok) break
    if (Date.now() - startHas > 3000) break
    await wait(100)
  }
  const has = await page.evaluate(() => (globalThis as any).__TEST_HAS_SCOREBOARD && (globalThis as any).__TEST_HAS_SCOREBOARD())
  if (!has) throw new Error('store scoreboard missing')

  // Assert scoreboard card contains N1TEST
  await page.waitForSelector('text=N1TEST', { timeout: 2000 })
  expect(await page.locator('text=N1TEST').count()).toBeGreaterThan(0)
})
