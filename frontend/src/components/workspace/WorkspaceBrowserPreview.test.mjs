import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'
import vm from 'node:vm'
import ts from 'typescript'
import { computed, effectScope, nextTick, reactive, ref, watch } from 'vue'

const source = readFileSync(new URL('./WorkspaceBrowserPreview.vue', import.meta.url), 'utf8')
const setup = source.split('<script setup lang="ts">')[1].split('</script>')[0].replace(/^import .*$/gm, '')
const compiled = ts.transpile(setup, { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.None })
const flush = async () => { for (let i = 0; i < 8; i++) await nextTick() }
function mount(api) {
  const props = reactive({ sessionId: 'a', enabled: true, active: true })
  const scope = effectScope(), cleanup = []
  const state = scope.run(() => vm.runInNewContext(`${compiled}; ({ result, busy, address, open, onDesktopStatus })`, {
    computed, ref, watch, defineProps: () => props, defineEmits: () => () => {}, useI18n: () => ({ t: key => key }),
    onBeforeUnmount: fn => cleanup.push(fn), setTimeout, clearTimeout, openProgrammingPreview: api,
  }))
  return { props, state, stop: () => { cleanup.forEach(fn => fn()); scope.stop() } }
}
test('browser connects before opening a page; project starts only on request', async () => {
  const calls = []
  const view = mount(async (session, data) => { calls.push({ session, ...data }); return { data: { status: 'not_running' } } })
  await view.state.open(true)
  assert.equal(calls.length, 0)
  view.state.onDesktopStatus('connected'); await flush()
  assert.equal(calls[0].start, false)
  await view.state.open(true)
  assert.equal(calls[1].start, true)
  view.stop()
})
test('a previous conversation cannot overwrite a new preview with late output', async () => {
  let resolve
  const view = mount(() => new Promise(done => { resolve = done }))
  view.state.onDesktopStatus('connected')
  view.props.sessionId = 'b'; await nextTick()
  resolve({ data: { status: 'ready', url: 'http://localhost:3000' } }); await flush()
  assert.equal(view.state.result.value, null)
  assert.equal(view.state.address.value, '')
  assert.equal(view.state.busy.value, false)
  view.stop()
})
test('preview failures have a recovery state and release the request lock', async () => {
  const view = mount(async () => { throw Error('offline') })
  view.state.onDesktopStatus('connected'); await flush()
  assert.equal(view.state.result.value.status, 'start_failed')
  assert.equal(view.state.busy.value, false)
  view.stop()
})
