import type { WorkspaceCommandRun } from './workspaceEvents'

export type WorkspaceCheckKind = 'tests' | 'build' | 'types' | 'lint' | 'other'
export type WorkspaceCheck = WorkspaceCommandRun & {
  kind: WorkspaceCheckKind
  highlights: string[]
  log: string
  logTruncated: boolean
}

// Classify the executed command, never words found in its output. Unknown
// commands remain available as other operations and do not count as tests.
export function workspaceCheckKind(command: string): WorkspaceCheckKind {
  const segments = command.split(/&&|\|\||[;\n]/).map(segment => segment.trim())
  for (const segment of segments) {
    const executable = segment.replace(/^(?:(?:[A-Z_][A-Z_\d]*=(?:"[^"]*"|'[^']*'|\S+))\s+)+/, '')
    const script = /^(?:npm|pnpm|yarn|bun)\s+(?:run\s+)?([\w:-]+)/.exec(executable)?.[1] || ''
    if (/^(?:test|test:.+|e2e|coverage)$/.test(script) || /^(?:(?:npx|pnpm exec|bunx|uv run|poetry run)\s+)?(?:vitest|jest|pytest|playwright test|cypress run|go test|cargo test|dotnet test)\b/.test(executable) || /^python[\d.]*\s+-m\s+(?:pytest|unittest)\b/.test(executable)) return 'tests'
    if (/^(?:type-check|typecheck|check-types)(?::.*)?$/.test(script) || /^(?:(?:npx|pnpm exec|bunx)\s+)?(?:tsc|vue-tsc|mypy|pyright)\b/.test(executable)) return 'types'
    if (/^lint(?::.*)?$/.test(script) || /^(?:(?:npx|pnpm exec|bunx|uv run)\s+)?(?:eslint|stylelint|ruff check|golangci-lint run)\b/.test(executable)) return 'lint'
    if (/^build(?::.*)?$/.test(script) || /^(?:(?:npx|pnpm exec|bunx)\s+)?(?:vite build|next build|go build|cargo build|dotnet build)\b/.test(executable)) return 'build'
  }
  return 'other'
}

export function describeWorkspaceCheck(run: WorkspaceCommandRun): WorkspaceCheck {
  const log = run.output.replace(/\x1b\[[0-?]*[ -/]*[@-~]/g, '').replace(/\r(?!\n)/g, '\n')
  const tail = log.slice(-100_000)
  const lines = tail.split('\n').map(line => line.trim()).filter(Boolean)
  const summaries = lines.filter(line => /^(?:Tests?:|Test Suites:|Test Files\s|Tests\s|# (?:tests|pass|fail)|=+.*\d+ (?:passed|failed)|\d+ (?:passing|failing|passed|failed)|ok\s+\S+|test result:|✓.*built in|Successfully compiled)/i.test(line))
  const failures = lines.filter(line => /(?:\b(?:AssertionError|TypeError|ReferenceError|SyntaxError|Error:|error TS\d+|FAIL(?:ED)?\b|Expected:|Received:)|^[×✕❌])/.test(line))
  // Excerpts are evidence, not inferred test-case counts or invented explanations.
  const highlights = [...new Set(run.status === 'failed' ? [...failures, ...summaries] : summaries)].slice(0, 3).map(line => line.slice(0, 400))
  return { ...run, kind: workspaceCheckKind(run.command), highlights, log: tail, logTruncated: log.length > tail.length }
}
