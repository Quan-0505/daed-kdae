import { chromium } from 'playwright'
import { readFileSync, writeFileSync } from 'node:fs'
const session = JSON.parse(readFileSync('test-results/live-session.json', 'utf8'))
const browser = await chromium.launch({ headless: true, channel: 'msedge' })
const context = await browser.newContext({ viewport: { width: 1920, height: 1080 }, locale: 'zh-CN' })
await context.addInitScript((token) => {
  localStorage.setItem('token', token)
  localStorage.setItem('endpointURL', `${location.origin}/graphql`)
}, session.token)
const page = await context.newPage()
const errors = [],
  operations = new Set()
page.on('pageerror', (e) => errors.push(e.message))
page.on('response', async (response) => {
  if (response.url().endsWith('/graphql')) {
    try {
      const request = response.request().postDataJSON()
      operations.add(request.query.match(/(?:query|mutation)\s+(\w+)/)?.[1] ?? 'unnamed')
      const body = await response.json()
      if (body.errors) errors.push(...body.errors.map((e) => e.message))
    } catch {}
  }
})
try {
  await page.goto('http://127.0.0.1:12070')
  await page.getByRole('heading', { name: '节点与订阅', exact: true }).waitFor()
  await page.waitForTimeout(2500)
  for (const label of ['配置', 'DNS', '路由', '群组', '节点', '订阅', '日志', '总览']) {
    await page
      .locator('.console-sidebar')
      .getByRole('button', { name: new RegExp(`^${label}`) })
      .click()
    await page.getByRole('heading', { name: label, exact: true }).first().waitFor()
  }
  await page.screenshot({ path: 'test-results/live-runtime.png', fullPage: true })
  await page.goto('http://127.0.0.1:12070/#/?panel=groups')
  await page.getByRole('heading', { name: '群组', exact: true }).first().waitFor()
  await page.waitForTimeout(500)
  await page
    .locator('[data-testid="section"]')
    .first()
    .locator(':scope > div')
    .first()
    .getByRole('button')
    .last()
    .click()
  const dialog = page.getByRole('dialog')
  await dialog.locator('input').first().fill('console-browser-check')
  await dialog.getByRole('button', { name: '提交', exact: true }).click()
  await page.getByText('console-browser-check', { exact: true }).waitFor()
  const removed = await page.evaluate(async () => {
    async function request(query, variables = {}) {
      const result = await (
        await fetch('/graphql', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json', Authorization: 'Bearer ' + localStorage.getItem('token') },
          body: JSON.stringify({ query, variables }),
        })
      ).json()
      if (result.errors) throw Error(JSON.stringify(result.errors))
      return result.data
    }
    const groups = await request('query { groups { id name } }')
    const group = groups.groups.find((g) => g.name === 'console-browser-check')
    if (!group) throw Error('Created group missing from backend')
    return request('mutation RemoveGroup($id:ID!){removeGroup(id:$id)}', { id: group.id })
  })
  if (removed.removeGroup !== 1) throw Error('Group cleanup failed')
  await page.locator('.console-sidebar').getByRole('button', { name: /^总览/ }).click()
  await page.getByRole('switch').click()
  await page.waitForFunction(() =>
    [...document.querySelectorAll('.traffic-legends b')].every((element) => element.textContent.trim() === '—'),
  )
  await page.screenshot({ path: 'test-results/live-stopped.png', fullPage: true })
  console.log('Operations:', [...operations].join(', '))
  console.log('Errors:', JSON.stringify(errors))
  console.log('Group create/read/delete and engine stop state: passed')
  writeFileSync('test-results/live-report.json', JSON.stringify({ operations: [...operations], errors }, null, 2))
  if (errors.length) process.exitCode = 1
} finally {
  await browser.close()
}
