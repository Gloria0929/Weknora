import type { ProgrammingNode } from '@/api/programming'

export interface WorkspaceTestCase {
  id: string
  path: string
  name: string
  command?: string
}

const TEST_FILE = /(?:^|[-_.])(?:test|tests|spec|specs)(?:$|[-_.])|\.(?:test|spec)\./i
const TEST_DIR = /^(?:test|tests|spec|specs|__tests__)$/i
const IGNORED_DIR = /^(?:\.git|node_modules|vendor|dist|build|coverage|\.next|\.nuxt|\.venv|venv|target|\.tox|__pycache__)$/i
const shellQuote = (value: string) => `'${value.replaceAll("'", "'\\''")}'`

export function isWorkspaceTestFile(node: ProgrammingNode): boolean {
  return node.kind === 'file' && TEST_FILE.test(node.name)
}

export function shouldScanWorkspaceDirectory(node: ProgrammingNode): boolean {
  return node.kind === 'directory' && !IGNORED_DIR.test(node.name)
}

export function parseWorkspaceTestCases(path: string, source: string, packageRoot = ''): WorkspaceTestCase[] {
  const ext = path.split('.').pop()?.toLowerCase() || ''
  const cases: string[] = []
  if (['py', 'pyw'].includes(ext)) {
    for (const match of source.matchAll(/^\s*(?:async\s+)?def\s+(test\w+)\s*\(/gm)) cases.push(match[1])
  } else if (['js', 'jsx', 'ts', 'tsx', 'mjs', 'cjs'].includes(ext)) {
    const pattern = /\b(?:it|test)(?:\.each\s*\([^\n]*?\))?\s*\(\s*(['"`])([^\n]*?)\1/g
    for (const match of source.matchAll(pattern)) cases.push(match[2])
  } else if (ext === 'go') {
    for (const match of source.matchAll(/^\s*func\s+(Test\w+)\s*\(/gm)) cases.push(match[1])
  } else if (ext === 'rs') {
    for (const match of source.matchAll(/^\s*fn\s+(\w+)\s*\(/gm)) {
      const before = source.slice(Math.max(0, match.index! - 160), match.index)
      if (/#\s*\[\s*test\s*\]/.test(before)) cases.push(match[1])
    }
  }
  const unique = [...new Set(cases)]
  const dir = path.includes('/') ? path.slice(0, path.lastIndexOf('/')) : '.'
  const packagePath = packageRoot && path.startsWith(`${packageRoot}/`) ? path.slice(packageRoot.length + 1) : path
  return (unique.length ? unique : [path.split('/').pop() || path]).map((name) => {
    const command = ext === 'py'
      ? `python -m pytest ${shellQuote(unique.length ? `${path}::${name}` : path)}`
      : ext === 'go' && unique.length
        ? `go test ${shellQuote(dir === '.' ? '.' : `./${dir}`)} -run ${shellQuote(`^${name}$`)}`
        : ext === 'rs' && path.startsWith('tests/') && unique.length
          ? `cargo test --test ${shellQuote(path.slice(6, -3))} ${shellQuote(name)}`
          : ['js', 'jsx', 'ts', 'tsx', 'mjs', 'cjs'].includes(ext)
            ? `npm --prefix ${shellQuote(packageRoot || '.')} test -- ${shellQuote(packagePath)}${unique.length ? ` -t ${shellQuote(name)}` : ''}`
            : undefined
    return { id: `${path}:${name}`, path, name, command }
  })
}

export function isLikelyTestDirectory(node: ProgrammingNode): boolean {
  return TEST_DIR.test(node.name)
}
