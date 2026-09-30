import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'
import vm from 'node:vm'
import ts from 'typescript'
import { computed, effectScope, nextTick, reactive, ref, watch } from 'vue'

const panel = readFileSync(new URL('./SandboxSidePanel.vue', import.meta.url), 'utf8')

test('browser preview has one entry before code, artifacts and terminal', () => {
  const preview = panel.indexOf("id: 'preview'")
  const source = panel.indexOf("id: 'source'")
  const artifacts = panel.indexOf("id: 'artifacts'")
  const terminal = panel.indexOf("id: 'terminal'")
  assert.notEqual(preview, -1)
  assert.ok(preview < source)
  assert.ok(source < artifacts)
  assert.notEqual(artifacts, -1)
  assert.ok(artifacts < terminal)
  assert.doesNotMatch(panel, /id: 'desktop'/)
  assert.match(panel, /<ChatArtifactsPanel/)
  assert.doesNotMatch(panel, /chat-sandbox-panel__title/)
  assert.doesNotMatch(panel, /<h3/)
})

test('closing the panel drops the terminal mount flag so Files reopen does not reconnect', () => {
  assert.match(panel, /if\s*\(!visible\)/)
  assert.match(panel, /terminalMounted\.value = false/)
  assert.match(panel, /tab === 'terminal'/)
})

test('the unified browser uses one desktop relay with capability gating', () => {
  assert.match(panel, /ensureSandboxConfigs/)
  assert.match(panel, /desktopTabVisible/)
  assert.match(panel, /desktop_enabled/)
  assert.match(panel, /id: 'preview', icon: 'desktop', label: t\('workspace.preview'\)/)
  assert.match(panel, /:enabled="desktopTabVisible"/)
  assert.equal((panel.match(/<WorkspaceBrowserPreview\b/g) || []).length, 1)
  assert.doesNotMatch(panel, /<SandboxDesktop\b/)
})

const setup = panel.split('<script setup lang="ts">')[1].split('</script>')[0].replace(/^import[\s\S]*?from ['"][^'"]+['"]\n/gm, '')
const compiled = ts.transpile(setup, { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.None })
function mount(capabilities, options = {}) {
  const props = reactive({ sessionId: 'a', workspaceRevision: 0, ...options })
  const ui = { visible: ref(true), activeTab: ref('preview'), width: ref(560), clearArtifactFocus() {} }
  const resources = reactive({ agents: [{ id: 'agent', config: { sandbox_config_id: 'cfg' } }], sandboxConfigs: [{ id: 'cfg', config: { desktop_enabled: true } }], ensureSandboxConfigs: async () => {} })
  const scope = effectScope()
  const state = scope.run(() => vm.runInNewContext(`${compiled}; ({ desktopTabVisible, isWorkspaceTab, desktopMounted, tabs })`, {
    computed, nextTick, ref, watch, defineAsyncComponent: () => ({}), defineProps: () => props, withDefaults: value => value, defineEmits: () => () => {},
    useI18n: () => ({ t: key => key }), useChatSandboxPanel: () => ui, useChatResourcesStore: () => resources,
    getProgrammingPreviewCapabilities: capabilities,
  }))
  return { props, ui, state, stop: () => scope.stop() }
}
const flush = async () => { for (let i = 0; i < 6; i++) await nextTick() }

test('pinned desktop is visible even when the selected agent is missing', async () => {
  const view = mount(async () => ({ data: { pinned: true, desktop_enabled: true } }))
  await flush()
  assert.equal(view.state.desktopTabVisible.value, true)
  assert.equal(view.state.tabs.value[0].label, 'workspace.preview')
  assert.equal(view.state.tabs.value.filter(tab => ['preview', 'desktop'].includes(tab.id)).length, 1)
  assert.equal(view.state.desktopMounted.value, true)
  assert.equal(view.state.isWorkspaceTab.value, false, 'real previews do not mount the source/file preview')
  view.ui.activeTab.value = 'source'; await flush()
  assert.equal(view.state.isWorkspaceTab.value, true)
  view.stop()
})

test('pinned CLI config overrides a newly selected desktop agent', async () => {
  const view = mount(async () => ({ data: { pinned: true, desktop_enabled: false } }), { agentId: 'agent' })
  await flush()
  assert.equal(view.state.desktopTabVisible.value, false)
  assert.equal(view.state.tabs.value[0].label, 'workspace.preview')
  view.stop()
})

test('late capabilities from the previous session cannot enable its browser', async () => {
  let resolveOld
  const old = new Promise(resolve => { resolveOld = resolve })
  const view = mount(id => id === 'a' ? old : Promise.resolve({ data: { pinned: true, desktop_enabled: false } }))
  view.props.sessionId = 'b'; await flush()
  resolveOld({ data: { pinned: true, desktop_enabled: true } }); await flush()
  assert.equal(view.state.desktopTabVisible.value, false)
  view.stop()
})

test('panel slide-in is clipped to the viewport so it cannot create a document scrollbar', () => {
  assert.match(panel, /chat-sandbox-panel-clip/)
  assert.match(panel, /sandbox-panel-enter-from \.chat-sandbox-panel/)
  assert.match(panel, /translateX\(100%\)/)
  assert.match(panel, /:duration/)
})
