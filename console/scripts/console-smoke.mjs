import assert from 'node:assert/strict'
import { readFileSync, mkdirSync } from 'node:fs'
import { createServer } from 'node:http'
import { resolve, extname } from 'node:path'
import { chromium } from 'playwright'
import ts from 'typescript'
const root = resolve('apps/web/dist')
const server = createServer((req, res) => {
  try {
    const p = new URL(req.url, 'http://localhost').pathname
    const file = resolve(root, '.' + (p === '/' ? '/index.html' : p))
    if (!file.startsWith(root)) throw Error()
    res.setHeader(
      'Content-Type',
      {
        '.html': 'text/html',
        '.js': 'text/javascript',
        '.css': 'text/css',
        '.webp': 'image/webp',
        '.woff2': 'font/woff2',
      }[extname(file)] ?? 'application/octet-stream',
    )
    res.end(readFileSync(file))
  } catch {
    res.writeHead(404)
    res.end()
  }
})
await new Promise((r) => server.listen(0, '127.0.0.1', r))
const origin = `http://127.0.0.1:${server.address().port}`
function load(file, imports) {
  const exports = {}
  const source = readFileSync(file, 'utf8').replace("import.meta.env.VITE_MOCK_MODE === 'true'", 'false')
  new Function(
    'exports',
    'require',
    ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 } })
      .outputText,
  )(exports, imports)
  return exports
}
const schema = load('apps/web/src/schemas/gql/graphql.ts', () => {
  throw Error('Unexpected import')
})
const f = load('apps/web/src/mocks/data.ts', () => schema)
const reads = {
  General: f.mockGeneral,
  Configs: f.mockConfigs,
  Nodes: f.mockNodes,
  Groups: f.mockGroups,
  Subscriptions: f.mockSubscriptions,
  Routings: f.mockRoutings,
  DNSs: f.mockDNSs,
  User: f.mockUser,
  Interfaces: { general: { interfaces: f.mockGeneral.general.interfaces } },
  Defaults: { jsonStorage: Object.values(f.MOCK_DEFAULT_IDS) },
  Mode: { jsonStorage: ['rule'] },
  NodeLatencies: { nodeLatencies: [] },
}
const browser = await chromium.launch({
  headless: true,
  ...(process.env.PLAYWRIGHT_CHANNEL ? { channel: process.env.PLAYWRIGHT_CHANNEL } : {}),
})
mkdirSync('test-results', { recursive: true })
const errors = [],
  operations = []
let trafficMode = 'supported',
  trafficCount = 0
async function open({ auth = true, width = 1920, height = 1080 } = {}) {
  const context = await browser.newContext({ viewport: { width, height }, locale: 'zh-CN' })
  if (auth)
    await context.addInitScript(() => {
      localStorage.setItem('token', 'fixture-jwt')
      localStorage.setItem('endpointURL', `${location.origin}/graphql`)
    })
  const page = await context.newPage()
  page.on('pageerror', (e) => errors.push(e.message))
  await page.route('**/graphql', async (route) => {
    const request = route.request(),
      body = request.postDataJSON(),
      op = body.query.match(/(?:query|mutation)\s+(\w+)/)?.[1]
    operations.push({ op, authorization: request.headers().authorization })
    if (op === 'TrafficOverview') {
      trafficCount++
      if (trafficMode === 'unsupported')
        return route.fulfill({
          json: { errors: [{ message: 'Cannot query field "runtimeOverview" on type "General".' }] },
        })
      if (trafficMode === 'offline')
        return route.fulfill({ status: 503, json: { errors: [{ message: 'Backend unavailable' }] } })
      const now = Date.now()
      return route.fulfill({
        json: {
          data: {
            general: {
              runtimeOverview: {
                updatedAt: new Date(now).toISOString(),
                uploadRate: 1024,
                downloadRate: 4096,
                uploadTotal: '13631488',
                downloadTotal: '105906176',
                activeConnections: 1,
                udpSessions: 2,
                samples: Array.from({ length: 61 }, (_, i) => ({
                  timestamp: new Date(now - (60 - i) * 1000).toISOString(),
                  uploadRate: 5400 * Math.exp(-((i - 32) ** 2) / 3),
                  downloadRate: 4800 * Math.exp(-((i - 32) ** 2) / 3),
                })),
              },
            },
          },
        },
      })
    }
    if (!reads[op]) {
      errors.push(`Unexpected GraphQL operation: ${op}`)
      return route.fulfill({ json: { errors: [{ message: 'Unexpected operation' }] } })
    }
    await route.fulfill({ json: { data: reads[op] } })
  })
  await page.goto(origin)
  return { context, page }
}
try {
  const { context, page } = await open()
  await page.getByText('101.0 MB', { exact: true }).waitFor()
  assert.equal(await page.getByText('布局预览', { exact: false }).count(), 0)
  assert(operations.every((o) => o.authorization === 'Bearer fixture-jwt'))
  assert(!operations.some((o) => /^(Create|Select|Set)/.test(o.op)))
  await page.screenshot({ path: 'test-results/desktop.png', fullPage: true })
  for (const label of ['配置', 'DNS', '路由', '群组', '节点', '订阅', '日志']) {
    await page
      .locator('.console-sidebar')
      .getByRole('button', { name: new RegExp(`^${label}`) })
      .click()
    await page.getByRole('heading', { name: label, exact: true }).first().waitFor()
    await page.waitForFunction(
      (expected) => document.querySelector('.console-sidebar [aria-current="page"]')?.innerText.split('\n')[0] === expected,
      label,
    )
    assert.equal((await page.locator('.console-sidebar [aria-current="page"]').innerText()).split('\n')[0], label)
  }
  await page.locator('.console-sidebar').getByRole('button', { name: /^群组/ }).click()
  await page.waitForFunction(() => document.querySelectorAll('[data-testid="section"]').length === 3)
  assert.equal(await page.locator('[data-testid="section"]').count(), 3, 'Keep group drag sources')
  await page.reload()
  assert(page.url().includes('panel=groups'))
  await page.locator('.console-sidebar').getByRole('button', { name: /^总览/ }).click()
  await page.getByText('101.0 MB', { exact: true }).waitFor()
  trafficMode = 'offline'
  await page.getByText('连接中断 · 图表保留上次采样', { exact: true }).waitFor({ timeout: 10000 })
  await context.close()
  trafficMode = 'unsupported'
  trafficCount = 0
  const old = await open()
  await old.page
    .getByText('当前后端未提供流量接口，请升级支持 runtimeOverview 的 dae-wing。', { exact: true })
    .waitFor()
  await old.page.waitForTimeout(1600)
  assert.equal(trafficCount, 1, 'Stop polling unsupported endpoint')
  assert.equal(await old.page.getByText('0 B/s', { exact: true }).count(), 0)
  await old.context.close()
  trafficMode = 'supported'
  reads.General.general.dae.running = false
  const stopped = await open()
  await stopped.page.waitForFunction(
    () => document.querySelector('.runtime-badges > span:last-child b')?.textContent === '已停止',
  )
  await stopped.page.waitForTimeout(600)
  assert.deepEqual(await stopped.page.locator('.traffic-legends b').allTextContents(), ['—', '—'])
  await stopped.context.close()
  reads.General.general.dae.running = true
  const mobile = await open({ width: 390, height: 844 })
  await mobile.page.getByRole('heading', { name: '节点与订阅', exact: true }).waitFor()
  assert(await mobile.page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), 'Mobile overflow')
  await mobile.page.getByRole('button', { name: '打开导航', exact: true }).click()
  await mobile.page.locator('.console-sidebar').getByRole('button', { name: /^节点/ }).click()
  await mobile.page.getByRole('heading', { name: '节点', exact: true }).first().waitFor()
  assert.equal(await mobile.page.locator('.console-sidebar.is-open').count(), 0)
  await mobile.page.getByRole('button', { name: '打开导航', exact: true }).click()
  await mobile.page.locator('.console-sidebar').getByRole('button', { name: /^总览/ }).click()
  await mobile.page.waitForFunction(() => document.querySelector('.console-sidebar').getBoundingClientRect().right <= 0)
  await mobile.page.screenshot({ path: 'test-results/mobile.png', fullPage: true })
  await mobile.context.close()
  const anonymous = await open({ auth: false })
  await anonymous.page.waitForURL('**/#/setup')
  assert.equal(await anonymous.page.locator('.console-sidebar').count(), 0)
  await anonymous.context.close()
  assert.deepEqual(errors, [])
  console.log(
    'PASS: production GraphQL/auth, all sections, group drag sources, navigation reload, stale telemetry, unsupported endpoint, mobile drawer/overflow, anonymous login redirect; screenshots saved.',
  )
} finally {
  await browser.close()
  await new Promise((r) => server.close(r))
}
