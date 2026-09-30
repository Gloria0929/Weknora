import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'
import vm from 'node:vm'
import ts from 'typescript'
import { computed, effectScope, nextTick, reactive, ref, watch } from 'vue'

// Exercise the component's state transitions without a browser or a sandbox.
// All filesystem and execution calls pass through an explicitly supplied test gateway.
const source = readFileSync(new URL('./ConversationWorkspace.vue', import.meta.url), 'utf8')
const setup = source.split('<script setup lang="ts">')[1].split('</script>')[0].replace(/^import[\s\S]*?from ['"][^'"]+['"];?\n/gm, '')
const compiled = ts.transpile(setup.replace('export interface', 'interface'), { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.None })
const file = path => ({ path, content: path, size: path.length, hash: path })
const deferred = () => { let resolve, reject; const promise = new Promise((a, b) => { resolve = a; reject = b }); return { promise, resolve, reject } }

function mount(gateway = {}) {
  const props = reactive({ sessionId: 'a', tab: 'source', active: false, revision: 0, focusPath: '', agentRuns: [], gateway: { tree: async () => ({ nodes: [], root: '/workspace' }), file: async (_, path) => file(path), run: async () => ({ stdout: '', stderr: '', exit_code: 0, duration_ms: 1, killed: false }), ...gateway } })
  const emitted = [], cleanup = []
  const scope = effectScope()
  const state = scope.run(() => vm.runInNewContext(`${compiled}; ({ file, nodes, directory, error, busy, command, running, runs, runError, openFile, loadDirectory, refresh, runTests, askToFix, reconcileMissingFile, treeRows, toggleDirectory, childrenByPath, expandedPaths, fileGlyph, fileView, showFilePreview, previewHtml, previewLoading })`, {
    computed, ref, watch, defineProps: () => props, withDefaults: value => value,
    defineEmits: () => (...args) => emitted.push(args), useI18n: () => ({ t: (key, values) => key + (values ? JSON.stringify(values) : '') }),
    onBeforeUnmount: fn => cleanup.push(fn), setTimeout, clearTimeout, Blob,
    hljs: { getLanguage: () => false },
    canPreviewSource: path => /\.(?:html?|svg|md|markdown)$/i.test(path),
    buildWorkspacePreview: async (path, content) => ({ html: content, warnings: [] }),
  }))
  return { props, state, emitted, unmount: () => { cleanup.forEach(fn => fn()); scope.stop() } }
}

test('rapid file selections cannot show a stale response', async () => {
  const first = deferred(), second = deferred()
  const view = mount({ file: async (_, path) => path === 'first.ts' ? first.promise : second.promise })
  const a = view.state.openFile('first.ts'), b = view.state.openFile('second.ts')
  second.resolve(file('second.ts')); await b
  first.resolve(file('first.ts')); await a
  assert.equal(view.state.file.value.path, 'second.ts')
  view.unmount()
})

test('changing conversation rejects late files and resets workspace state', async () => {
  const pending = deferred(), view = mount({ file: () => pending.promise })
  const request = view.state.openFile('old.ts')
  view.props.sessionId = 'b'; await nextTick()
  pending.resolve(file('old.ts')); await request
  assert.equal(view.state.file.value, null)
  assert.equal(view.state.directory.value, '')
  view.unmount()
})

test('test commands do not run on mount or on blank input; repeated submit is locked', async () => {
  let calls = 0
  const pending = deferred(), view = mount({ run: () => { calls++; return pending.promise } })
  assert.equal(calls, 0)
  view.state.command.value = '  '; await view.state.runTests(); assert.equal(calls, 0)
  view.state.command.value = 'npm test'
  const run = view.state.runTests(); await view.state.runTests()
  assert.equal(calls, 1); assert.equal(view.state.running.value, true)
  pending.resolve({ stdout: 'one failure', stderr: 'assertion failed', exit_code: 1, duration_ms: 42, killed: false }); await run
  assert.equal(view.state.running.value, false)
  assert.equal(view.state.runs.value[0].result.exit_code, 1)
  view.state.askToFix(view.state.runs.value[0])
  assert.equal(view.emitted[0][0], 'ask')
  assert.match(view.emitted[0][1], /assertion failed/)
  assert.match(view.emitted[0][1], /npm test/)
  view.unmount()
})

test('closing and reopening keeps results while conversation switches discard late command output', async () => {
  const pending = deferred(), view = mount({ run: () => pending.promise })
  view.state.command.value = 'go test ./...'
  const run = view.state.runTests()
  view.props.active = false; await nextTick()
  assert.equal(view.state.running.value, true)
  view.props.sessionId = 'b'; await nextTick()
  pending.resolve({ stdout: 'old session output', stderr: '', exit_code: 0, duration_ms: 5, killed: false }); await run
  assert.equal(view.state.runs.value.length, 0)
  assert.equal(view.state.running.value, false)
  view.unmount()
})

test('code tree hides the input and output scratch folders and expands folders lazily', async () => {
  const view = mount({ tree: async (_, path) => path
    ? { root: '/workspace', nodes: [{ path: 'src/main.ts', name: 'main.ts', kind: 'file' }] }
    : { root: '/workspace', nodes: [
      { path: 'input', name: 'input', kind: 'directory' },
      { path: 'output', name: 'output', kind: 'directory' },
      { path: 'src', name: 'src', kind: 'directory' },
      { path: 'index.html', name: 'index.html', kind: 'file' },
    ] } })
  await view.state.loadDirectory()
  assert.deepEqual(Array.from(view.state.treeRows.value, row => row.node.name), ['src', 'index.html'])
  await view.state.toggleDirectory({ path: 'src', name: 'src', kind: 'directory' })
  assert.deepEqual(Array.from(view.state.treeRows.value, row => row.node.name), ['src', 'main.ts', 'index.html'])
  assert.deepEqual(Array.from(view.state.expandedPaths.value), ['src'])
  await view.state.toggleDirectory({ path: 'src', name: 'src', kind: 'directory' })
  assert.deepEqual(Array.from(view.state.treeRows.value, row => row.node.name), ['src', 'index.html'])
  view.unmount()
})

test('host workspaces keep project folders named input or output', async () => {
  const view = mount({ tree: async () => ({ root: '/Users/dev/site', origin: 'host', nodes: [
    { path: 'input', name: 'input', kind: 'directory' },
    { path: 'output', name: 'output', kind: 'directory' },
    { path: 'index.html', name: 'index.html', kind: 'file' },
  ] }) })
  await view.state.loadDirectory()
  assert.deepEqual(Array.from(view.state.treeRows.value, row => row.node.name), ['input', 'output', 'index.html'])
  view.unmount()
})

test('a file the backend no longer serves leaves the tree instead of a dead row', async () => {
  let listings = 0
  const view = mount({
    tree: async (_, path) => {
      if (path === 'todolist') {
        listings += 1
        // The first listing is the row the user clicked; the re-listing after
        // the failed read is what a rebuilt sandbox actually returns.
        return { root: '/workspace', nodes: listings === 1 ? [{ path: 'todolist/test_backend.py', name: 'test_backend.py', kind: 'file' }] : [] }
      }
      return { root: '/workspace', nodes: [{ path: 'todolist', name: 'todolist', kind: 'directory' }] }
    },
    file: async () => { throw Error('error code: 1003, error message: file not found') },
  })
  await view.state.loadDirectory()
  await view.state.toggleDirectory({ path: 'todolist', name: 'todolist', kind: 'directory' })
  assert.deepEqual(Array.from(view.state.treeRows.value, row => row.node.name), ['todolist', 'test_backend.py'])

  await view.state.openFile('todolist/test_backend.py')
  await view.state.reconcileMissingFile('todolist/test_backend.py')
  assert.match(view.state.error.value, /workspace\.fileGone/)
  assert.deepEqual(Array.from(view.state.treeRows.value, row => row.node.name), ['todolist'])
  assert.equal(view.state.file.value, null)
  view.unmount()
})

test('the file tree stays visible alongside the per-file toolbar', () => {
  assert.doesNotMatch(source, /workspace-toolbar/)
  assert.doesNotMatch(source, /treeOpen/)
  assert.match(source, /<aside class="file-tree"/)
  assert.doesNotMatch(source, /<aside[^>]*v-if/)
})

test('file markers follow Manus: TS/JS text, braces for data, git glyph for git files', () => {
  const view = mount()
  const glyph = name => view.state.fileGlyph(name)
  assert.equal(glyph('vite.config.ts').text, 'TS')
  assert.equal(glyph('App.tsx').text, 'TS')
  assert.equal(glyph('app.jsx').text, 'JS')
  assert.equal(glyph('package.json').text, '{}')
  assert.equal(glyph('pnpm-lock.yaml').text, '{}')
  assert.equal(glyph('.prettierrc').text, '{}')
  assert.equal(glyph('.gitignore').icon, 'git-branch')
  assert.equal(glyph('.gitkeep').icon, 'file')
  assert.equal(glyph('index.html').icon, 'file')
  assert.equal(glyph('notes.md').icon, 'file')
  view.unmount()
})

test('unavailable workspaces and command errors surface recovery states without fabricated output', async () => {
  const view = mount({ tree: async () => { throw Error('unavailable') }, run: async () => { throw Error('offline') } })
  await view.state.loadDirectory()
  assert.match(view.state.error.value, /workspace.unavailable/)
  assert.equal(view.state.busy.value, false)
  view.state.command.value = 'python -m pytest'; await view.state.runTests()
  assert.match(view.state.runError.value, /workspace.runError/)
  assert.equal(view.state.runs.value.length, 0)
  view.unmount()
})

test('preview frame has an opaque origin and no form, top navigation, or same-origin privilege', () => {
  const sandbox = source.match(/<iframe[^>]*sandbox="([^"]+)"/)?.[1]
  assert.equal(sandbox, 'allow-scripts')
  assert.match(source, /referrerpolicy="no-referrer"/)
})


test('HTML can switch between source and preview inside the code tab', async () => {
  const view = mount()
  await view.state.openFile('index.html')
  assert.equal(view.state.showFilePreview.value, false)
  view.state.fileView.value = 'preview'; await new Promise(resolve => setImmediate(resolve))
  assert.equal(view.state.showFilePreview.value, true)
  assert.equal(view.state.previewHtml.value, 'index.html')
  await view.state.openFile('style.css'); await nextTick()
  assert.equal(view.state.showFilePreview.value, false)
  await view.state.openFile('other.html'); await new Promise(resolve => setImmediate(resolve))
  assert.equal(view.state.showFilePreview.value, true)
  assert.equal(view.state.previewHtml.value, 'other.html')
  view.state.fileView.value = 'source'; await nextTick()
  assert.equal(view.state.showFilePreview.value, false)
  view.unmount()
})

test('file preview selection resets when changing conversations', async () => {
  const view = mount()
  await view.state.openFile('index.html')
  view.state.fileView.value = 'preview'; await nextTick()
  view.props.sessionId = 'b'; await nextTick()
  assert.equal(view.state.fileView.value, 'source')
  assert.equal(view.state.previewHtml.value, '')
  view.unmount()
})
