import { cp, mkdir, readFile, readdir, rm, stat } from 'node:fs/promises'
import { dirname, isAbsolute, relative, resolve, sep } from 'node:path'
import { fileURLToPath } from 'node:url'

const repositoryRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const frontendRoot = resolve(repositoryRoot, 'frontend')
const sourceRoot = resolve(frontendRoot, 'dist-mobile-viewer')
const destinationRoot = resolve(
  repositoryRoot,
  'mobile',
  'ios',
  'HayaTab',
  'Resources',
  'Viewer'
)
const alphaTabRoot = resolve(
  frontendRoot,
  'node_modules',
  '@coderline',
  'alphatab',
  'dist'
)

function assertChild(parent, child) {
  const childRelative = relative(parent, child)
  if (
    childRelative === ''
    || childRelative === '..'
    || childRelative.startsWith(`..${sep}`)
    || isAbsolute(childRelative)
  ) {
    throw new Error(`Unsafe package path: ${child}`)
  }
}

async function assertBuildManifest() {
  const manifestURL = resolve(sourceRoot, '.vite', 'manifest.json')
  const manifest = JSON.parse(await readFile(manifestURL, 'utf8'))
  for (const entry of Object.values(manifest)) {
    const paths = [
      entry.file,
      ...(entry.css ?? []),
      ...(entry.assets ?? []),
      ...(entry.imports ?? [])
    ]
    for (const candidate of paths) {
      if (
        typeof candidate !== 'string'
        || candidate.length === 0
        || isAbsolute(candidate)
        || candidate.split(/[\\/]/u).includes('..')
      ) {
        throw new Error(`Unsafe manifest entry: ${String(candidate)}`)
      }
      assertChild(sourceRoot, resolve(sourceRoot, candidate))
    }
  }
}

async function collectFiles(root, directory = root) {
  const entries = await readdir(directory, { withFileTypes: true })
  const files = []
  for (const entry of entries) {
    const entryPath = resolve(directory, entry.name)
    assertChild(root, entryPath)
    if (entry.isDirectory()) {
      files.push(...await collectFiles(root, entryPath))
    } else if (entry.isFile()) {
      files.push(entryPath)
    } else {
      throw new Error(`Unsupported package entry: ${entryPath}`)
    }
  }
  return files
}

async function assertNoDevelopmentRuntime() {
  const files = await collectFiles(destinationRoot)
  for (const file of files) {
    if (!/\.(?:html|css|js|json)$/iu.test(file)) continue
    const contents = await readFile(file, 'utf8')
    if (/\b(?:localhost|127\.0\.0\.1|wails)\b/iu.test(contents)) {
      throw new Error(`Development runtime reference in ${file}`)
    }
    if (/\.html$/iu.test(file) && /<script[^>]+src=["']https?:\/\//iu.test(contents)) {
      throw new Error(`External script URL in ${file}`)
    }
  }
}

if (!(await stat(sourceRoot)).isDirectory()) {
  throw new Error('Build frontend/dist-mobile-viewer before packaging')
}
assertChild(repositoryRoot, destinationRoot)
await assertBuildManifest()

await rm(destinationRoot, { recursive: true, force: true })
await mkdir(destinationRoot, { recursive: true })
await cp(sourceRoot, destinationRoot, { recursive: true })
await cp(resolve(alphaTabRoot, 'font'), resolve(destinationRoot, 'font'), {
  recursive: true
})
await cp(
  resolve(alphaTabRoot, 'soundfont'),
  resolve(destinationRoot, 'soundfont'),
  { recursive: true }
)
await assertNoDevelopmentRuntime()

console.log(`Packaged local viewer at ${destinationRoot}`)
