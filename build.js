import { spawnSync } from 'node:child_process'
import { cpSync, mkdirSync, rmSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join, resolve } from 'node:path'
const root = dirname(fileURLToPath(import.meta.url))
const windows = process.platform === 'win32'
const result = spawnSync(windows ? 'cmd.exe' : 'pnpm', windows ? ['/d', '/s', '/c', 'pnpm --dir console --filter daed... build'] : ['--dir', 'console', '--filter', 'daed...', 'build'], { cwd: root, stdio: 'inherit', env: { ...process.env, VITE_MOCK_MODE: 'false' } })
if (result.error) throw result.error
if (result.status !== 0) process.exit(result.status ?? 1)
const destination = resolve(root, 'dist')
if (dirname(destination) !== root) throw new Error('Invalid build destination')
rmSync(destination, { recursive: true, force: true })
mkdirSync(destination, { recursive: true })
cpSync(join(root, 'console/apps/web/dist'), destination, { recursive: true })
console.log('GraphQL console built to dist/ (production, no mock backend).')
