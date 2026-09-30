import { buildShellExecView } from './shellExecResult'
import { canPreviewSource } from './workspacePreview'

export type WorkspaceIntent = 'chat' | 'build' | 'code' | 'test' | 'knowledge'
export type WorkspaceTab = 'source' | 'preview' | 'tests'

export interface WorkspaceSignal {
  tab: WorkspaceTab
  path: string
  refresh: boolean
  tool: string
}

export interface WorkspacePanelDecision {
  tab: WorkspaceTab
  path: string
}

export function workspaceIntent(value: unknown): WorkspaceIntent {
  return typeof value === 'string' && ['build', 'code', 'test', 'knowledge'].includes(value) ? value as WorkspaceIntent : 'chat'
}

export function intentPanelTab(intent: WorkspaceIntent): WorkspaceTab | null {
  return intent === 'build' ? 'preview' : intent === 'code' ? 'source' : intent === 'test' ? 'tests' : null
}

const fileTools = new Set(['write_sandbox_file', 'edit_sandbox_file', 'read_sandbox_file', 'list_sandbox_files'])
const commandTools = new Set(['shell_exec', 'execute_code'])
/** Writes and edits name the artifact a finished turn should show. */
const artifactTools = new Set(['write_sandbox_file', 'edit_sandbox_file'])

/** Writes and edits are the only tools that produce a file diff. */
const mutationTools = new Set(['write_sandbox_file', 'edit_sandbox_file'])

export interface WorkspaceFileChange {
  path: string
  type: 'added' | 'modified'
  /** Pre-write text; empty string for a brand-new file, undefined when the
   *  backend omitted the body because the file was too large to ship. */
  before?: string
  after?: string
  addedLines: number
  removedLines: number
}

function toCount(value: unknown): number {
  const n = typeof value === 'number' ? value : Number(value)
  return Number.isFinite(n) && n > 0 ? Math.trunc(n) : 0
}

/**
 * Every object level a tool result can hide behind: live SSE flattens the
 * result onto `data`, history stores it under `tool_data`, and some shapes wrap
 * it one more time in `data`. Follow the chain so the caller reads the level
 * that actually carries `path` / `diff_*` instead of assuming one shape.
 */
function toolResultLevels(root: any): Array<Record<string, any>> {
  const levels: Array<Record<string, any>> = []
  const seen = new Set<any>()
  let current: any = root
  while (
    current &&
    typeof current === 'object' &&
    !seen.has(current) &&
    levels.length < 6
  ) {
    seen.add(current)
    levels.push(current)
    current = current.tool_data ?? current.data
  }
  return levels
}

/**
 * The file mutation a `tool_result` chunk reports, or null.
 *
 * `before`/`after` are the full-file sides the backend ships for the code view
 * diff; a file past the payload cap reports only the +/- line counts, and the
 * caller still uses the path to move the reader to the file.
 */
export function workspaceFileChange(
  chunk: { response_type?: string; data?: Record<string, any> },
): WorkspaceFileChange | null {
  if (chunk.response_type !== 'tool_result') return null
  const levels = toolResultLevels(chunk.data || {})
  const name = String(
    levels.map((level) => level.tool_name).find((value) => typeof value === 'string' && value) || '',
  )
  if (!mutationTools.has(name)) return null
  const result = levels.find((level) => level.path || level.file_path)
  if (!result) return null
  const path = String(result.path || result.file_path || '')
  if (!path) return null
  const before = typeof result.diff_before === 'string' ? result.diff_before : undefined
  const after = typeof result.diff_after === 'string' ? result.diff_after : undefined
  return {
    path,
    type: name === 'write_sandbox_file' && before === '' ? 'added' : 'modified',
    before,
    after,
    addedLines: toCount(result.added_lines),
    removedLines: toCount(result.removed_lines),
  }
}

/**
 * Every file mutation a loaded conversation recorded, oldest first.
 *
 * Reloading a session replays the persisted tool events, so the diff a turn
 * produced is still there to show after a refresh.
 */
export function collectWorkspaceFileChanges(
  messages: Array<Record<string, any>>,
): WorkspaceFileChange[] {
  const out: WorkspaceFileChange[] = []
  for (const message of messages) {
    for (const event of message.agentEventStream || []) {
      if (event.type !== 'tool_call' || event.pending) continue
      const change = workspaceFileChange({
        response_type: 'tool_result',
        data: { ...(event.tool_data || {}), tool_name: event.tool_name },
      })
      if (change) out.push(change)
    }
  }
  return out
}

/** Only real workspace tools drive the panel; ordinary answers/code snippets do not. */
export function workspaceToolSignal(chunk: { response_type?: string; data?: Record<string, any> }): WorkspaceSignal | null {
  if (chunk.response_type !== 'tool_call' && chunk.response_type !== 'tool_result') return null
  const data = chunk.data || {}
  const name = String(data.tool_name || '')
  if (!fileTools.has(name) && !commandTools.has(name)) return null
  let args = data.arguments || {}
  if (typeof args === 'string') {
    try { args = JSON.parse(args) } catch { args = {} }
  }
  const result = data.tool_data || data.data || data
  const command = String(args.command || args.code || result.command || '')
  const isTest = /(?:\b(?:test|pytest|vitest|jest|playwright|unittest)\b|go\s+test)/i.test(command)
  const rawPath = String(result.path || result.file_path || args.path || args.file_path || '')
  // A listing names a directory; the tree can only open files.
  const path = name === 'list_sandbox_files' ? '' : rawPath
  return {
    tab: (isTest ? 'tests' : /\.(?:html?|svg)$/i.test(path) ? 'preview' : 'source') as WorkspaceTab,
    path,
    refresh: chunk.response_type === 'tool_result',
    tool: name,
  }
}

/**
 * A finished turn opens the panel on its own result. Artifacts beat plain
 * reads, a previewable file beats everything, then a test run, then the source
 * view of the file the turn touched.
 */
export function completedWorkspacePanel(signals: WorkspaceSignal[]): WorkspacePanelDecision | null {
  const written = signals.filter(signal => signal.path && artifactTools.has(signal.tool))
  const files = written.length ? written : signals.filter(signal => signal.path)
  const previewable = [...files].reverse().find(signal => canPreviewSource(signal.path))
  if (previewable) return { tab: 'preview', path: previewable.path }
  if (signals.some(signal => signal.tab === 'tests')) return { tab: 'tests', path: '' }
  if (files.length) return { tab: 'source', path: files[files.length - 1].path }
  return signals.length ? { tab: 'source', path: '' } : null
}

export interface WorkspaceCommandRun {
  id: string
  command: string
  output: string
  status: 'running' | 'passed' | 'failed' | 'unknown'
  durationMs?: number
  exitCode?: number | null
}

/** Shared renderer consumes existing SSE/hydrated tool events, including failures. */
export function collectWorkspaceRuns(messages: Array<Record<string, any>>): WorkspaceCommandRun[] {
  return messages.flatMap(message => (message.agentEventStream || [])
    .filter((event: any) => event.type === 'tool_call' && commandTools.has(event.tool_name))
    .map((event: any, index: number): WorkspaceCommandRun => {
      const data = event.tool_data?.tool_data || event.tool_data?.data || event.tool_data || {}
      const view = buildShellExecView(data, event.arguments, event.output || event.error)
      return {
        id: `${message.id}:${event.tool_call_id || event.id || index}`,
        command: view.command || String(event.arguments?.code || event.tool_name),
        output: [view.stdout, view.stderr].filter(Boolean).join('\n'),
        status: event.pending ? 'running'
          : view.killed || (view.exitCode != null && view.exitCode !== 0) || event.success === false ? 'failed'
          : view.exitCode === 0 || event.success === true ? 'passed' : 'unknown',
        durationMs: view.durationMs ?? event.duration_ms ?? event.duration,
        exitCode: view.exitCode,
      }
    })).slice(-20).reverse()
}

/** Restored conversations open preview while retaining the latest workspace path. */
export function lastWorkspaceSignal(messages: Array<Record<string, any>>): WorkspacePanelDecision | null {
  for (const message of [...messages].reverse()) {
    const signals: WorkspaceSignal[] = []
    for (const event of message.agentEventStream || []) {
      if (event.type !== 'tool_call') continue
      const signal = workspaceToolSignal({ response_type: 'tool_result', data: { ...event.tool_data, tool_name: event.tool_name, arguments: event.arguments } })
      if (signal) signals.push(signal)
    }
    const decision = completedWorkspacePanel(signals)
    if (decision) return { ...decision, tab: 'preview' }
  }
  return null
}
