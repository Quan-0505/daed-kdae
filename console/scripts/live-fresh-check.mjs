import assert from 'node:assert/strict'
import { chromium } from 'playwright'
import { readFileSync, writeFileSync } from 'node:fs'
const session = JSON.parse(readFileSync('test-results/live-fresh-session.json', 'utf8'))
const browser = await chromium.launch({ headless: true, channel: 'msedge' })
const context = await browser.newContext({ viewport: { width: 1920, height: 1080 }, locale: 'zh-CN' })
await context.addInitScript((token) => {
  localStorage.setItem('token', token)
  localStorage.setItem('endpointURL', `${location.origin}/graphql`)
}, session.token)
const page = await context.newPage()
const errors = []
page.on('pageerror', (error) => errors.push(error.message))
page.on('response', async (response) => {
  if (response.url().endsWith('/graphql')) {
    const body = await response.json()
    if (body.errors) errors.push(...body.errors.map((e) => e.message))
  }
})
try {
  await page.goto('http://127.0.0.1:12071')
  await page.getByText('global', { exact: true }).waitFor()
  await page.getByText('proxy', { exact: true }).waitFor()
  await page.waitForTimeout(1500)
  assert.deepEqual(errors, [], 'Fresh database initialization must not report missing config')
  assert.deepEqual(
    await page.locator('.traffic-legends b').allTextContents(),
    ['—', '—'],
    'Stopped engine must not appear as live zero traffic',
  )
  await page.screenshot({ path: 'test-results/live-fresh-stopped.png', fullPage: true })
  writeFileSync(
    'test-results/live-fresh-report.json',
    JSON.stringify({ freshDatabaseInitialized: true, zeroTimestampHidden: true, errors }, null, 2),
  )
  console.log(
    'PASS: fresh database initialization, real GraphQL management defaults, stopped engine/zero timestamp, no browser or API errors',
  )
} finally {
  await browser.close()
}
