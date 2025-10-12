import { chromium } from '@playwright/test'

;(async () => {
  const browser = await chromium.launch()
  const ctx = await browser.newContext()
  const page = await ctx.newPage()

  page.on('console', msg => {
    console.log('PAGE LOG ›', msg.type(), msg.text())
  })
  page.on('pageerror', err => {
    console.log('PAGE ERROR ›', err.toString())
  })
  page.on('requestfailed', req => {
    console.log('REQUEST FAILED ›', req.url(), req.failure && req.failure().errorText)
  })

  try {
    await page.goto('http://localhost:5173/_test/source-node', { waitUntil: 'networkidle', timeout: 10000 })
    console.log('Navigated, waiting 2s for console output...')
    await new Promise(r => setTimeout(r, 2000))
  } catch (e) {
    console.error('Navigation failed:', e && e.message)
  }

  await browser.close()
})().catch(e => { console.error(e); process.exit(1) })
