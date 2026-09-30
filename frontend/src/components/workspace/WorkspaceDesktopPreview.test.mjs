import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const source = readFileSync(new URL('./WorkspaceDesktopPreview.vue', import.meta.url), 'utf8')
test('desktop only renders the sandbox desktop, without web navigation or project execution', () => {
  assert.match(source, /<SandboxDesktop v-else/)
  assert.doesNotMatch(source, /openProgrammingPreview|ProgrammingPreviewRequest|startPreviewPrompt|preview-location|setViewport|<iframe/)
  assert.match(source, /@status="emit\('status', \$event\)"/)
})
test('unknown and unsupported desktop capabilities never mount a connection', () => {
  assert.match(source, /v-if="loading"[\s\S]*desktopChecking/)
  assert.match(source, /v-else-if="!enabled"[\s\S]*desktopUnavailable/)
  assert.match(source, /desktopUnsupportedHint/)
})
