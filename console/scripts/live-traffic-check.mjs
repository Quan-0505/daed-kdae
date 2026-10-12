import assert from 'node:assert/strict'
import { readFileSync, writeFileSync } from 'node:fs'
import { chromium } from 'playwright'

const session = JSON.parse(readFileSync('test-results/live-session.json', 'utf8'))
const browser = await chromium.launch({ headless: true, channel: process.env.PLAYWRIGHT_CHANNEL || 'msedge' })
const context = await browser.newContext({ viewport: { width: 1920, height: 1080 }, locale: 'zh-CN' })
await context.addInitScript((token) => {
  localStorage.setItem('token', token)
  localStorage.setItem('endpointURL', `${location.origin}/graphql`)
}, session.token)
const page = await context.newPage()
const errors = []
const snapshots = []
page.on('pageerror', (error) => errors.push(error.message))
page.on('response', async (response) => {
  if (!response.url().endsWith('/graphql')) return
  try {
    const body = await response.json()
    if (body.errors) errors.push(...body.errors.map((error) => error.message))
    const stats = body.data?.general?.runtimeOverview
    if (stats) snapshots.push(stats)
  } catch {}
})
try {
  await page.goto('http://127.0.0.1:12070')
  await page.getByRole('heading', { name: '运行状态', exact: true }).waitFor()
  await page.waitForFunction(() => {
    const metrics = [...document.querySelectorAll('.runtime-stats > div')]
    const total = metrics.find((metric) => metric.querySelector('small')?.textContent === '累计下载')
    return /MB|GB/.test(total?.querySelector('strong')?.textContent || '')
  })
  await page.waitForFunction(() => document.querySelectorAll('.recharts-line-curve').length === 2)
  await page.waitForTimeout(2200)
  assert(snapshots.some((stats) => Number(stats.downloadTotal) >= 1048576))
  assert(snapshots.some((stats) => stats.samples.some((sample) => sample.downloadRate > 0)))
  assert(snapshots.some((stats) => stats.activeConnections > 0 || stats.udpSessions > 0))
  await page.screenshot({ path: 'test-results/live-traffic.png', fullPage: true })
  const report = { snapshots, errors, realTrafficChart: true }
  await page.setViewportSize({ width: 390, height: 844 })
  await page.waitForTimeout(300)
  assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth))
  await page.screenshot({ path: 'test-results/live-traffic-mobile.png', fullPage: true })
  await page.setViewportSize({ width: 1920, height: 1080 })
  await page.getByRole('switch').click()
  await page.waitForFunction(() =>
    [...document.querySelectorAll('.traffic-legends b')].every((element) => element.textContent.trim() === '—'),
  )
  await page.screenshot({ path: 'test-results/live-traffic-stopped.png', fullPage: true })
  assert.equal(errors.length, 0, JSON.stringify(errors))
  report.engineStop = true
  writeFileSync('test-results/live-traffic-browser-report.json', JSON.stringify(report, null, 2))
  console.log('Real traffic chart, active metrics, mobile layout and UI engine stop: passed')
} finally {
  await browser.close()
}
