import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'
import vm from 'node:vm'
import ts from 'typescript'
import { buildUnifiedDiff } from '../../utils/unifiedDiff.ts'
import { formatWorkspaceCode, highlightWorkspaceLines, trimHighlightedIndent } from '../../utils/workspaceCode.ts'
import { computed, effectScope, nextTick, reactive, ref, watch } from 'vue'

// Exercise the component's state transitions without a browser or a sandbox.
// All filesystem and execution calls pass through an explicitly supplied test gateway.
const source = readFileSync(new URL('./ConversationWorkspace.vue', import.meta.url), 'utf8')
const setup = source.split('<script setup lang="ts">')[1].split('</script>')[0].replace(/^import[\s\S]*?from ['"][^'"]+['"];?\n/gm, '')
const compiled = ts.transpile(setup.replace('export interface', 'interface'), { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.None })
const file = path => ({ path, content: path, size: path.length, hash: path })
const deferred = () => { let resolve, reject; const promise = new Promise((a, b) => { resolve = a; reject = b }); return { promise, resolve, reject } }

function mount(gateway = {}, seed = {}) {
  const props = reactive({ sessionId: 'a', tab: 'source', active: false, revision: 0, focusPath: '', agentRuns: [], gateway: { tree: async () => ({ nodes: [], root: '/workspace' }), file: async (_, path) => file(path), run: async () => ({ stdout: '', stderr: '', exit_code: 0, duration_ms: 1, killed: false }), ...gateway } })
  const emitted = [], cleanup = []
  // The chat view provides one change list per conversation; the tests drive it
  // directly so a live edit can be replayed without the SSE handler.
  const changes = seed.changes || ref([]), activeId = seed.activeId || ref(null)
  const scope = effectScope()
  const state = scope.run(() => vm.runInNewContext(`${compiled}; ({ file, nodes, directory, error, busy, command, running, runs, runError, openFile, loadDirectory, refresh, runTests, askToFix, reconcileMissingFile, treeRows, toggleDirectory, childrenByPath, expandedPaths, fileGlyph, fileView, showFilePreview, showFileDiff, showViewSwitch, previewHtml, previewLoading, currentDiff, diffHunks, hasFileDiff, formattedCode, content, highlightedLines, treeWidth, treeStyle, startTreeResize, resizeTree, endTreeResize, TREE_MIN_WIDTH, TREE_MAX_WIDTH })`, {
    computed, ref, watch, defineProps: () => props, withDefaults: value => value,
    defineEmits: () => (...args) => emitted.push(args), useI18n: () => ({ t: (key, values) => key + (values ? JSON.stringify(values) : '') }),
    onBeforeUnmount: fn => cleanup.push(fn), onMounted: fn => cleanup.push(fn), setTimeout, clearTimeout, Blob,
    formatWorkspaceCode: seed.formatWorkspaceCode || (async source => source),
    highlightWorkspaceLines, trimHighlightedIndent,
    canPreviewSource: path => /\.(?:html?|svg|md|markdown)$/i.test(path),
    buildWorkspacePreview: async (path, content) => ({ html: content, warnings: [] }),
    useFileChanges: () => ({ changes, activeId, addChange() {}, clearChanges() {}, hasChanges: () => changes.value.length > 0 }),
    findFileChange: (list, path) => list.find(change => change.path === path || path.endsWith(`/${change.path}`)),
    buildUnifiedDiff,
  }))
  return { props, state, emitted, changes, activeId, unmount: () => { cleanup.forEach(fn => fn()); scope.stop() } }
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
  assert.match(source, /<aside[^>]*class="file-tree"/)
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

const flush = async () => { await nextTick(); await new Promise(resolve => setImmediate(resolve)); await nextTick() }
const liveChange = (id, path) => ({ id, path, type: 'modified', before: 'one\n', after: 'two\n', addedLines: 1, removedLines: 1, timestamp: 1, live: true })

test('a live edit opens the changed file in the diff view and follows A then B', async () => {
  const view = mount()
  view.changes.value = [liveChange('a-1', '/workspace/a.ts')]
  view.activeId.value = 'a-1'
  await flush()
  assert.equal(view.state.file.value.path, 'a.ts')
  assert.equal(view.state.fileView.value, 'diff')
  // A second file retargets the view instead of leaving the reader on A.
  view.changes.value = [...view.changes.value, liveChange('b-2', '/workspace/src/b.ts')]
  view.activeId.value = 'b-2'
  await flush()
  assert.equal(view.state.file.value.path, 'src/b.ts')
  assert.equal(view.state.fileView.value, 'diff')
  view.unmount()
})

test('a panel that mounts after a live edit still lands on the diff, not the restored file', async () => {
  // The edit happened before the panel was mounted: the immediate watcher and
  // the refresh both have to land on the changed file in diff view.
  const changes = ref([liveChange('a-1', '/workspace/a.ts')])
  const view = mount({}, { changes, activeId: ref('a-1') })
  await flush()
  assert.equal(view.state.fileView.value, 'diff')
  assert.equal(view.state.file.value.path, 'a.ts')
  view.unmount()
})

test('restored history alone does not force the diff view open', async () => {
  const changes = ref([{ ...liveChange('h-1', '/workspace/a.ts'), live: false }])
  const view = mount({}, { changes, activeId: ref('h-1') })
  await flush()
  assert.equal(view.state.fileView.value, 'source')
  view.unmount()
})

test('a changed file shows its colored diff inline without a diff toggle', async () => {
  // The user opens the file from the tree instead of waiting for the live
  // watcher: the code pane still has to paint the diff, not plain source.
  const changes = ref([{ ...liveChange('m-1', 'a.ts'), live: false }])
  const view = mount({}, { changes, activeId: ref(null) })
  await view.state.openFile('a.ts')
  await nextTick()
  assert.equal(view.state.hasFileDiff.value, true)
  assert.equal(view.state.showFileDiff.value, true)
  // No previewable file here, so the toolbar carries no view switch at all.
  assert.equal(view.state.showViewSwitch.value, false)
  view.unmount()
})

test('the code pane offers only source and preview, never a diff mode button', () => {
  // The colored diff lives inside the code pane; a dedicated button to reach
  // it would be the separate "changes" view the reader asked us to drop.
  assert.doesNotMatch(source, /fileView = 'diff'/)
  assert.doesNotMatch(source, /v-if="hasFileDiff" type="button"/)
})

test('the code side of the panel is resizable by dragging the tree divider', () => {
  // The sandbox panel already resizes; the code/diff side gets the same grip so
  // a wide diff fits without widening the whole panel.
  assert.match(source, /<PanelResizeHandle[\s\S]*?edge="right"/)
  assert.match(source, /:style="treeStyle"/)
  assert.match(source, /max-width: calc\(100% - 200px\)/)
  const view = mount()
  // Nothing stored yet: the responsive CSS default owns the width.
  assert.equal(view.state.treeWidth.value, 0)
  assert.equal(view.state.treeStyle.value, undefined)
  // Deltas are measured from the width the drag started at, matching the
  // pointer handler that feeds this splitter.
  view.state.startTreeResize()
  view.state.resizeTree(48)
  assert.equal(view.state.treeWidth.value, 180)
  // The style object comes from the vm realm, so compare the field, not the prototype.
  assert.equal(view.state.treeStyle.value.width, '180px')
  // Dragging past the ceiling leaves the code pane its minimum room.
  view.state.resizeTree(100000)
  assert.equal(view.state.treeWidth.value, view.state.TREE_MAX_WIDTH)
  view.state.resizeTree(-100000)
  assert.equal(view.state.treeWidth.value, view.state.TREE_MIN_WIDTH)
  view.unmount()
})

test('the diff renders inside the code view, never as a separate sidebar tab', () => {
  const panel = readFileSync(new URL('../chat/SandboxSidePanel.vue', import.meta.url), 'utf8')
  assert.match(source, /class="diff-table"/)
  assert.match(source, /showFileDiff/)
  // The sidebar keeps Preview/Source/Tests/Artifacts/Terminal — no Changes tab.
  assert.doesNotMatch(panel, /id: 'changes'/)
  assert.doesNotMatch(panel, /tabChanges|changesTab/)
})


test('late formatting cannot replace the current file or conversation', async () => {
  const pending = deferred()
  const view = mount({}, { formatWorkspaceCode: source => source === 'old.ts' ? pending.promise : Promise.resolve('formatted ' + source) })
  await view.state.openFile('old.ts')
  await view.state.openFile('new.ts')
  await new Promise(resolve => setImmediate(resolve))
  assert.equal(view.state.content.value, 'formatted new.ts')
  pending.resolve('stale old file')
  await new Promise(resolve => setImmediate(resolve))
  assert.equal(view.state.content.value, 'formatted new.ts')
  view.props.sessionId = 'b'; await nextTick()
  assert.equal(view.state.content.value, '')
  view.unmount()
})


test('source formatting leaves the original diff and its line numbers intact', async () => {
  const before = '{"count":1}', after = '{"count":2}'
  // Warm the lazy parser so the assertion only waits for the component update.
  await formatWorkspaceCode(before, 'data.json')
  const changes = ref([{ id: 'edit', path: 'data.json', before, after }])
  const view = mount({ file: async () => ({ ...file('data.json'), content: after }) }, { changes, formatWorkspaceCode })
  await view.state.openFile('data.json')
  await new Promise(resolve => setImmediate(resolve))
  assert.match(view.state.content.value, /\n  "count": 2\n/)
  assert.equal(view.state.file.value.content, after)
  const rows = view.state.diffHunks.value.flatMap(hunk => hunk.lines)
  assert.match(rows.find(row => row.type === 'add').highlighted, /hljs-number.*2/)
  assert.match(rows.find(row => row.type === 'del').highlighted, /hljs-number.*1/)
  assert.equal(rows.find(row => row.type === 'add').newNumber, 1)
  assert.equal(rows.find(row => row.type === 'del').content, before)
  assert.equal(rows.find(row => row.type === 'add').content, after)
  view.unmount()
})

test('formatting does not erase whitespace-only diffs', async () => {
  const before = 'const x=1;', after = 'const x = 1;'
  const changes = ref([{ id: 'edit', path: 'app.js', before, after }])
  const view = mount({ file: async () => ({ ...file('app.js'), content: after }) }, { changes, formatWorkspaceCode: async () => after })
  await view.state.openFile('app.js')
  await new Promise(resolve => setImmediate(resolve))
  assert.equal(view.state.hasFileDiff.value, true)
  const rows = view.state.currentDiff.value.hunks.flatMap(hunk => hunk.lines)
  assert.equal(rows.find(row => row.type === 'del').content, before)
  view.unmount()
})
