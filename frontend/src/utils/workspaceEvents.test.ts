import test from 'node:test'
import assert from 'node:assert/strict'
import { collectWorkspaceRuns, completedWorkspacePanel, intentPanelTab, workspaceIntent, workspaceToolSignal, lastWorkspaceSignal } from './workspaceEvents'
import { resolveWorkspaceAsset, canPreviewSource, PREVIEW_CSP } from './workspacePreview'

test('intent selects the right workspace surface without treating knowledge as coding', () => {
  assert.equal(intentPanelTab(workspaceIntent('build')), 'preview')
  assert.equal(intentPanelTab(workspaceIntent('test')), 'tests')
  assert.equal(intentPanelTab(workspaceIntent('code')), 'source')
  assert.equal(intentPanelTab(workspaceIntent('knowledge')), null)
  assert.equal(workspaceIntent(['build']), 'chat')
  assert.equal(workspaceIntent('unknown'), 'chat')
})

test('only actual workspace tools open the panel and completed tools trigger refresh', () => {
  assert.equal(workspaceToolSignal({ response_type: 'answer', data: { tool_name: 'shell_exec' } }), null)
  assert.equal(workspaceToolSignal({ response_type: 'tool_call', data: { tool_name: 'knowledge_search' } }), null)
  assert.deepEqual(workspaceToolSignal({ response_type: 'tool_call', data: { tool_name: 'write_sandbox_file', arguments: '{"path":"index.html"}' } }), { tab: 'preview', path: 'index.html', refresh: false, tool: 'write_sandbox_file' })
  assert.deepEqual(workspaceToolSignal({ response_type: 'tool_result', data: { tool_name: 'edit_sandbox_file', tool_data: { path: 'src/main.ts' } } }), { tab: 'source', path: 'src/main.ts', refresh: true, tool: 'edit_sandbox_file' })
  assert.equal(workspaceToolSignal({ response_type: 'tool_call', data: { tool_name: 'shell_exec', arguments: { command: 'npm run test' } } })?.tab, 'tests')
  assert.equal(workspaceToolSignal({ response_type: 'tool_call', data: { tool_name: 'shell_exec', arguments: '{incomplete' } })?.tab, 'source')
  // A listing names a directory: it refreshes the tree but never focuses a row.
  assert.equal(workspaceToolSignal({ response_type: 'tool_result', data: { tool_name: 'list_sandbox_files', arguments: '{"path":"todolist"}' } })?.path, '')
})

test('a finished turn opens the panel on its own result', () => {
  const signal = (tool: string, path = '', tab: 'source' | 'preview' | 'tests' = 'source') => ({ tool, path, tab, refresh: true })
  assert.equal(completedWorkspacePanel([]), null)
  assert.deepEqual(completedWorkspacePanel([signal('shell_exec', '', 'tests')]), { tab: 'tests', path: '' })
  assert.deepEqual(completedWorkspacePanel([signal('shell_exec')]), { tab: 'source', path: '' })
  // The written page wins even when the agent ran tests afterwards.
  assert.deepEqual(completedWorkspacePanel([signal('write_sandbox_file', 'index.html', 'preview'), signal('shell_exec', '', 'tests')]), { tab: 'preview', path: 'index.html' })
  assert.deepEqual(completedWorkspacePanel([signal('edit_sandbox_file', 'src/main.ts')]), { tab: 'source', path: 'src/main.ts' })
  assert.deepEqual(completedWorkspacePanel([signal('write_sandbox_file', 'pages/about.html')]), { tab: 'preview', path: 'pages/about.html' })
  assert.deepEqual(completedWorkspacePanel([signal('write_sandbox_file', 'notes.md')]), { tab: 'preview', path: 'notes.md' })
  // Code that is not previewable shows the run that verified it.
  assert.deepEqual(completedWorkspacePanel([signal('edit_sandbox_file', 'src/main.ts'), signal('shell_exec', '', 'tests')]), { tab: 'tests', path: '' })
  // An artifact outranks a file the same turn only read, previewable or not.
  assert.deepEqual(completedWorkspacePanel([signal('write_sandbox_file', 'app.ts'), signal('read_sandbox_file', 'README.md')]), { tab: 'source', path: 'app.ts' })
})

test('execution results preserve failures, running state, and output from the conversation', () => {
  const result = collectWorkspaceRuns([{ id: 'm1', agentEventStream: [
    { type: 'tool_call', tool_call_id: 'a', tool_name: 'shell_exec', arguments: { command: 'npm test' }, pending: false, success: false, output: 'FAIL: expected 1', duration_ms: 45 },
    { type: 'tool_call', tool_call_id: 'b', tool_name: 'shell_exec', arguments: { command: 'go test ./...' }, pending: true },
    { type: 'thinking', content: 'run a test' },
  ] }])
  assert.equal(result.length, 2)
  assert.equal(result[0].status, 'running')
  assert.equal(result[1].status, 'failed')
  assert.equal(result[1].output, 'FAIL: expected 1')
  assert.equal(result[1].durationMs, 45)
})

test('asset resolution stays within the workspace and rejects remote URLs and traversal', () => {
  assert.equal(resolveWorkspaceAsset('pages/index.html', '../style.css?v=2'), 'style.css')
  assert.equal(resolveWorkspaceAsset('pages/index.html', '/src/app.js'), 'src/app.js')
  assert.equal(resolveWorkspaceAsset('index.html', './style.css'), 'style.css')
  for (const input of ['../../secrets', '%2e%2e/secrets', 'https://example.com/a.js', '//example.com/a.js', 'data:text/html,test', 'javascript:alert(1)', 'file:///tmp/a', '..\\secret', '%00hidden', '%broken']) {
    assert.equal(resolveWorkspaceAsset('index.html', input), null, input)
  }
})

test('historical coding conversations restore their latest workspace file', () => {
  const messages = [{ id: 'a', agentEventStream: [{ type: 'tool_call', tool_name: 'write_sandbox_file', tool_data: { tool_data: { path: '/workspace/index.html' } } }] }, { id: 'b', agentEventStream: [{ type: 'tool_call', tool_name: 'knowledge_search' }] }]
  assert.equal(lastWorkspaceSignal(messages)?.path, '/workspace/index.html')
  assert.equal(lastWorkspaceSignal(messages)?.tab, 'preview')
  assert.equal(lastWorkspaceSignal([{ content: '```html\n<h1>example</h1>```' }]), null)
})

test('structured command exit codes override tool-level success and absent status stays unknown', () => {
  const runs = collectWorkspaceRuns([{ agentEventStream: [
    { type: 'tool_call', tool_name: 'shell_exec', success: true, arguments: '{"command":"npm test"}', tool_data: { tool_data: { exit_code: 1, stdout: 'FAIL', stderr: 'assertion', duration_ms: 25 } } },
    { type: 'tool_call', tool_name: 'shell_exec', arguments: { command: 'npm test' } },
  ] }])
  assert.equal(runs[0].status, 'unknown')
  assert.equal(runs[1].status, 'failed')
  assert.equal(runs[1].command, 'npm test')
  assert.equal(runs[1].output, 'FAIL\nassertion')
  assert.equal(runs[1].exitCode, 1)
})

test('standalone preview recognizes supported types and denies network, forms, frames, and base changes', () => {
  assert.equal(canPreviewSource('index.html'), true)
  assert.equal(canPreviewSource('README.md'), true)
  assert.equal(canPreviewSource('logo.svg'), true)
  assert.equal(canPreviewSource('src/App.vue'), false)
  assert.equal(canPreviewSource('index.tsx'), false)
  for (const rule of ["default-src 'none'", "connect-src 'none'", "frame-src 'none'", "base-uri 'none'", "form-action 'none'"]) assert.ok(PREVIEW_CSP.includes(rule))
})
