import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const source = readFileSync(new URL('./WorkspaceBrowserPreview.vue', import.meta.url), 'utf8')
test('URL preview works by default without desktop capabilities or connection', () => {
  assert.match(source, /ref<'url' \| 'desktop'>\('url'\)/)
  assert.match(source, /<WorkspaceUrlPreview v-show="mode === 'url'"/)
  assert.match(source, /<WorkspaceDesktopPreview v-if="mode === 'desktop'"/)
  assert.match(source, /watch\(\(\) => props.sessionId/)
})

test('the desktop tab reports unsupported configurations and runtime capability failures', () => {
  assert.match(source, /!props.loading && \(!props.enabled \|\| desktopStatus.value === 'unsupported'\)/)
  assert.match(source, /v-if="desktopUnavailable"/)
  assert.match(source, /workspace.unavailableBadge/)
  assert.match(source, /@status="desktopStatus = \$event"/)
})
