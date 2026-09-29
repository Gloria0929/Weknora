import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const source = readFileSync(new URL('./index.vue', import.meta.url), 'utf8')

function slice(startMarker, endMarker) {
  const start = source.indexOf(startMarker)
  const end = source.indexOf(endMarker, start)
  assert.notEqual(start, -1, `missing ${startMarker}`)
  assert.notEqual(end, -1, `missing ${endMarker}`)
  return source.slice(start, end)
}

// The panel used to pop open on the first tool_call of a turn, while the agent
// was still writing, and landed on the source view. It now waits for the
// finished turn and opens on the artifact that turn produced.
test('the workspace panel opens on the finished turn, not while the agent writes', () => {
  const chunk = slice('onChunk((data) => {', "if (data.response_type === 'session_title')")
  assert.match(chunk, /workspaceTurnSignals\.push\(workspaceSignal\)/)
  assert.doesNotMatch(chunk, /sandboxPanel\.open/)

  const hook = slice('onTurnComplete:', '});')
  assert.match(hook, /openCompletedWorkspacePanel\(\)/)

  const open = slice('function openCompletedWorkspacePanel() {', 'function prefillWorkspacePrompt')
  assert.match(open, /completedWorkspacePanel\(workspaceTurnSignals\)/)
  assert.match(open, /workspaceTurnSignals = \[\]/)
  // The opened panel focuses the file this turn produced, and refreshes so a
  // panel that was already open re-reads it.
  assert.match(open, /if \(decision\.path && decision\.path !== workspacePath\.value\)/)
  assert.match(open, /workspaceRevision\.value\+\+/)
  assert.match(open, /sandboxPanel\.open\(decision\.tab\)/)
  // A panel the user closed by hand stays closed.
  assert.match(open, /!sandboxPanel\.autoOpenAllowed\.value\) return/)
})

// Choosing build / code / test still swaps in the coding agent, but it no
// longer throws an empty workspace panel at the user before anything exists.
test('sending a coding prompt no longer opens an empty panel', () => {
  const send = slice('const sendMsg = async', 'const reasoningEffort')
  assert.match(send, /activateWorkspaceAgent\(useSettingsStoreInstance, nextTick\)/)
  assert.doesNotMatch(send, /sandboxPanel\.open/)
  assert.match(send, /workspaceTurnSignals = \[\]/)

  const sessionWatch = slice('watch(() => session_id.value, () => {', 'function forkAffordanceOf')
  assert.doesNotMatch(sessionWatch, /sandboxPanel\.open/)
})

// History is restored through the same rule, so a reload of a finished build
// lands on its preview instead of an empty source tab.
test('history restore reuses the finished-turn decision', () => {
  const restore = slice('watch(historyLoading', 'const historyLoadingMore')
  assert.match(restore, /lastWorkspaceSignal\(messagesList\)/)
  assert.match(restore, /sandboxPanel\.open\(signal\.tab\)/)
  assert.doesNotMatch(restore, /intentPanelTab/)
})
