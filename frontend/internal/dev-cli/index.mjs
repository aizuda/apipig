import { spawn } from 'node:child_process'
import { existsSync } from 'node:fs'
import { rm } from 'node:fs/promises'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { intro, log, multiselect, outro, select, spinner } from '@clack/prompts'
import { glob } from 'glob'

const __dirname = dirname(fileURLToPath(import.meta.url))
const rootDir = resolve(__dirname, '../..')

const requiredPackages = [
  { name: '@tabtab/utils', path: 'packages/utils', distCheck: 'dist/index.mjs' },
  { name: '@tabtab/ui', path: 'packages/ui', distCheck: 'dist/vite.mjs' },
]

const cacheTypes = [
  {
    name: 'vite',
    label: 'Vite cache',
    hint: 'node_modules/.vite cache',
    patterns: ['node_modules/.vite', 'packages/*/node_modules/.vite'],
  },
  {
    name: 'vitest',
    label: 'Vitest cache',
    hint: 'test run cache',
    patterns: ['node_modules/.vitest', 'packages/*/node_modules/.vitest'],
  },
  {
    name: 'tsdown',
    label: 'tsdown cache',
    hint: 'library build cache',
    patterns: ['.tsdown', 'packages/*/.tsdown'],
  },
  {
    name: 'tsbuildinfo',
    label: 'TypeScript cache',
    hint: '*.tsbuildinfo files',
    patterns: ['**/*.tsbuildinfo'],
  },
  {
    name: 'dist',
    label: 'Build output',
    hint: 'package dist and web output',
    patterns: ['packages/*/dist', '../web'],
  },
  {
    name: 'node_modules',
    label: 'node_modules',
    hint: 'all dependency folders',
    patterns: ['node_modules', 'packages/*/node_modules', 'internal/*/node_modules'],
    isNodeModules: true,
  },
]

const args = process.argv.slice(2)
const isCleanMode = args.includes('--clean') || args.includes('-c')
const isCleanAll = args.includes('--all') || args.includes('-a')
const isDeepClean = args.includes('--deep') || args.includes('-d')

async function ensurePackagesBuilt() {
  const packagesToBuild = requiredPackages.filter((pkg) => {
    const distPath = resolve(rootDir, pkg.path, pkg.distCheck)
    return !existsSync(distPath)
  })

  if (packagesToBuild.length === 0) return

  log.info('Building required packages first...')

  for (const pkg of packagesToBuild) {
    const s = spinner()
    s.start(`Building ${pkg.name}...`)

    await new Promise((resolvePromise, reject) => {
      const child = spawn('pnpm build', {
        shell: true,
        stdio: 'pipe',
        cwd: resolve(rootDir, pkg.path),
        env: { ...process.env, FORCE_COLOR: '1' },
      })

      let stderr = ''
      child.stderr?.on('data', (data) => {
        stderr += data.toString()
      })

      child.on('close', (code) => {
        if (code === 0) {
          s.stop(`${pkg.name} built`)
          resolvePromise()
          return
        }

        s.stop(`${pkg.name} build failed`)
        if (stderr) log.error(stderr)
        reject(new Error(`${pkg.name} build failed with exit code ${code}`))
      })
    })
  }
}

async function removePath(pattern) {
  if (pattern.includes('*') || pattern.includes('?') || pattern.includes('[')) {
    const files = await glob(pattern, {
      cwd: rootDir,
      absolute: true,
      ignore: ['**/node_modules/**/node_modules/**'],
    })

    for (const file of files) {
      await rm(file, { recursive: true, force: true })
    }
    return files.length
  }

  await rm(resolve(rootDir, pattern), { recursive: true, force: true })
  return 1
}

async function cleanCache(selectedTypes) {
  const s = spinner()
  s.start('Cleaning caches...')

  let cleanedCount = 0
  const errors = []
  const hasNodeModules = selectedTypes.includes('node_modules')

  for (const type of selectedTypes) {
    const cacheType = cacheTypes.find((item) => item.name === type)
    if (!cacheType) continue

    for (const pattern of cacheType.patterns) {
      try {
        cleanedCount += await removePath(pattern)
      } catch (error) {
        if (error?.code !== 'ENOENT') {
          errors.push(`${pattern}: ${error.message}`)
        }
      }
    }
  }

  if (errors.length > 0) {
    s.stop(`Finished with ${errors.length} errors`)
    for (const error of errors) log.error(error)
  } else {
    s.stop(`Removed ${cleanedCount} cache and output items`)
  }

  if (!hasNodeModules) {
    return { shouldExit: false }
  }

  const shouldInstall = await select({
    message: 'Reinstall dependencies now?',
    options: [
      { label: 'Yes', value: true },
      { label: 'No', value: false },
    ],
  })

  if (shouldInstall) {
    log.info('Running pnpm install...')
    const child = spawn('pnpm install', {
      shell: true,
      stdio: 'inherit',
      cwd: rootDir,
      env: { ...process.env, FORCE_COLOR: '1' },
    })

    child.on('close', (code) => process.exit(code ?? 0))
    return { shouldExit: true }
  }

  return { shouldExit: false }
}

async function deepClean() {
  const s = spinner()
  s.start('Removing node_modules...')

  try {
    const nodeModules = await glob('**/node_modules', {
      cwd: rootDir,
      absolute: true,
      onlyDirectories: true,
    })

    nodeModules.push(resolve(rootDir, 'node_modules'))

    for (const dir of nodeModules) {
      await rm(dir, { recursive: true, force: true })
    }

    s.stop('node_modules removed')

    log.info('Running pnpm install...')
    const child = spawn('pnpm install', {
      shell: true,
      stdio: 'inherit',
      cwd: rootDir,
      env: { ...process.env, FORCE_COLOR: '1' },
    })

    child.on('close', (code) => process.exit(code ?? 0))
  } catch (error) {
    s.stop('Deep clean failed')
    log.error(error.message)
    process.exit(1)
  }
}

async function runCleanMode() {
  intro('Cache cleanup')

  if (isDeepClean) {
    const confirmed = await select({
      message: 'Delete all node_modules and reinstall?',
      options: [
        { label: 'Yes', value: true },
        { label: 'No', value: false },
      ],
    })

    if (!confirmed) {
      outro('Cancelled')
      process.exit(0)
    }

    await deepClean()
    return
  }

  if (isCleanAll) {
    const result = await cleanCache(cacheTypes.map((item) => item.name))
    if (!result.shouldExit) outro('Cleanup complete')
    return
  }

  const selectedTypes = await multiselect({
    message: 'Select cache types to clean',
    options: cacheTypes.map((type) => ({
      label: type.label,
      value: type.name,
      hint: type.hint,
    })),
    required: false,
  })

  if (!selectedTypes || selectedTypes.length === 0) {
    outro('No cache type selected')
    process.exit(0)
  }

  if (selectedTypes.includes('node_modules')) {
    const confirmed = await select({
      message: 'Delete node_modules and reinstall dependencies?',
      options: [
        { label: 'Yes', value: true },
        { label: 'No', value: false },
      ],
    })

    if (!confirmed) {
      outro('Cancelled')
      process.exit(0)
    }
  }

  const result = await cleanCache(selectedTypes)
  if (!result.shouldExit) outro('Cleanup complete')
}

async function runDevMode() {
  intro('ApiPig')
  await ensurePackagesBuilt()

  log.info('Starting Vite dev server...')
  const child = spawn('pnpm exec vp dev', {
    shell: true,
    stdio: 'inherit',
    cwd: rootDir,
    env: { ...process.env, FORCE_COLOR: '1' },
  })

  child.on('close', (code) => process.exit(code ?? 0))
}

if (isCleanMode) {
  runCleanMode()
} else {
  runDevMode()
}
