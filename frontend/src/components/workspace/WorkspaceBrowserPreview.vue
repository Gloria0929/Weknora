<template>
  <section ref="root" class="workspace-browser" :class="{ 'is-fullscreen': fullscreen }" :aria-label="t('workspace.preview')">
    <div class="preview-toolbar">
      <div class="preview-devices" role="group" :aria-label="t('workspace.previewDevice')">
        <button v-for="device in devices" :key="device.id" type="button" class="icon-button"
          :class="{ selected: viewport === device.id }" :aria-pressed="viewport === device.id"
          :title="device.label" :aria-label="device.label" :disabled="!connected || busy || result?.status !== 'ready'"
          @click="setViewport(device.id)"><t-icon :name="device.icon" size="16px" /></button>
      </div>
      <form class="preview-location" @submit.prevent="open(false, address)">
        <button type="button" class="icon-button" :title="t('workspace.previewHome')" :aria-label="t('workspace.previewHome')"
          :disabled="!connected || busy || !currentURL" @click="open(false, '/')"><t-icon name="home" size="15px" /></button>
        <input v-model="address" :aria-label="t('workspace.previewAddress')" placeholder="/" :title="currentURL || t('workspace.previewAddress')"
          :disabled="!enabled || loading || !connected || busy" autocomplete="off" spellcheck="false" @keydown.esc.stop="resetAddress" />
        <button type="button" class="icon-button" :title="t('workspace.previewRefresh')" :aria-label="t('workspace.previewRefresh')"
          :disabled="!enabled || loading || !connected || busy" @click="open(false, currentURL, 'reload')">
          <t-icon :name="busy ? 'loading' : 'refresh'" size="15px" :class="{ spinning: busy }" />
        </button>
        <button type="submit" class="location-submit" tabindex="-1" :disabled="!connected || busy">{{ t('workspace.previewNavigate') }}</button>
      </form>
      <button type="button" class="icon-button" :title="t(fullscreen ? 'preview.exitFullscreen' : 'preview.fullscreen')"
        :aria-label="t(fullscreen ? 'preview.exitFullscreen' : 'preview.fullscreen')" @click="toggleFullscreen">
        <t-icon :name="fullscreen ? 'fullscreen-exit' : 'fullscreen'" size="16px" />
      </button>
    </div>

    <div v-if="actionError || (connected && (busy || result?.status !== 'ready' || result?.detail))" class="preview-notice">
      <div class="preview-notice__heading">
        <t-icon :name="busy ? 'loading' : 'info-circle'" size="16px" :class="{ spinning: busy }" />
        <p :role="actionError ? 'alert' : 'status'">{{ actionError || message }}</p>
      </div>
      <details v-if="result?.detail"><summary>{{ t('workspace.previewDetails') }}</summary><pre>{{ result.detail }}</pre></details>
      <div v-if="connected && !busy && result?.status !== 'ready'" class="preview-actions">
        <button v-if="result?.status !== 'browser_unavailable'" type="button" @click="open(true)">{{ t('workspace.startPreview') }}</button>
        <button type="button" @click="open(false)">{{ t('workspace.retry') }}</button>
        <button type="button" @click="emit('ask', t('workspace.startPreviewPrompt'))">{{ t('workspace.fix') }}</button>
      </div>
    </div>

    <div class="preview-stage" :class="{ 'is-mobile': viewport === 'mobile' && connected }">
      <div class="preview-screen" :class="{ 'is-mobile': viewport === 'mobile' && connected }" :style="screenStyle">
        <div v-if="loading" class="preview-empty" role="status">
          <div class="preview-skeleton" aria-hidden="true"><div /><i /><i /><i /></div>
          <p>{{ t('workspace.previewConnecting') }}</p>
        </div>
        <div v-else-if="!enabled" class="preview-empty" role="status">
          <t-icon name="desktop" size="32px" /><p>{{ t('workspace.browserUnsupported') }}</p>
        </div>
        <SandboxDesktop v-else :key="sessionId" :session-id="sessionId" :agent-id="agentId"
          :agent-source-tenant-id="agentSourceTenantId" appearance="preview" @status="onDesktopStatus" />
      </div>
    </div>
    <footer class="preview-status">
      <span><i :class="{ online: connected }" aria-hidden="true" />{{ t(connected ? 'workspace.previewConnected' : 'workspace.previewDisconnected') }}</span>
      <span v-if="connected && result?.width && result?.height">{{ result.width }} × {{ result.height }}</span>
    </footer>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import SandboxDesktop from '@/views/chat/components/SandboxDesktop.vue'
import { openProgrammingPreview, type ProgrammingPreviewResult, type ProgrammingPreviewRequest } from '@/api/programming'

const props = defineProps<{ sessionId: string; agentId?: string; agentSourceTenantId?: string | number | null; enabled: boolean; loading: boolean; active: boolean; revision?: number }>()
const emit = defineEmits<{ ask: [prompt: string] }>()
const { t } = useI18n()
const root = ref<HTMLElement | null>(null)
const address = ref('/'), currentURL = ref(''), busy = ref(false), connected = ref(false), fullscreen = ref(false)
const actionError = ref(''), viewport = ref<'desktop' | 'mobile'>('desktop')
const result = ref<ProgrammingPreviewResult | null>(null)
const devices = computed(() => [
  { id: 'desktop' as const, icon: 'desktop', label: t('workspace.previewDesktop') },
  { id: 'mobile' as const, icon: 'mobile', label: t('workspace.previewMobile') },
])
const screenStyle = computed(() => viewport.value === 'mobile' && connected.value
  ? { aspectRatio: `${result.value?.width || 390} / ${result.value?.height || 780}` }
  : undefined)
let generation = 0
let refreshTimer: ReturnType<typeof setTimeout> | undefined
const message = computed(() => t(busy.value ? 'workspace.previewOpening' : result.value?.status === 'ready' ? 'workspace.previewControlUnavailable' : result.value?.status === 'browser_unavailable' ? 'workspace.browserMissing' : result.value?.status === 'start_failed' ? 'workspace.previewStartFailed' : result.value?.status === 'starting' ? 'workspace.previewOpening' : 'workspace.previewNotRunning'))

function resolveAddress(raw: string) {
  const value = raw.trim()
  if (!value || value === '/' && !currentURL.value) return ''
  const url = new URL(value, currentURL.value || undefined)
  if (url.protocol !== 'http:' || !['localhost', '127.0.0.1', '[::1]'].includes(url.hostname) || url.username || url.password) throw new Error('invalid preview address')
  return url.href
}
function resetAddress() {
  if (!currentURL.value) { address.value = '/'; return }
  const url = new URL(currentURL.value)
  address.value = url.pathname + url.search + url.hash
}
async function open(start = false, url = '', action?: ProgrammingPreviewRequest['action'], device = viewport.value) {
  if (busy.value || !connected.value || !props.enabled) return
  let resolved: string
  try { resolved = resolveAddress(url) } catch { actionError.value = t('workspace.previewInvalidAddress'); return }
  const epoch = generation
  busy.value = true
  actionError.value = ''
  try {
    const response = await openProgrammingPreview(props.sessionId, { start, url: resolved, action, viewport: device })
    if (epoch !== generation) return
    // Failed controls leave the current framebuffer and selected device intact.
    if (action && response.data.status !== 'ready' && result.value?.status === 'ready') {
      actionError.value = response.data.detail || t('workspace.previewControlUnavailable')
      return
    }
    result.value = response.data
    if (response.data.url) { currentURL.value = response.data.url; resetAddress() }
    if (response.data.viewport) viewport.value = response.data.viewport
    if (action === 'resize' && response.data.viewport !== device) actionError.value = t('workspace.previewControlUnavailable')
  } catch {
    if (epoch !== generation) return
    if (result.value?.status === 'ready') actionError.value = t('workspace.previewError')
    else result.value = { status: 'start_failed', detail: t('workspace.previewError') }
  } finally { if (epoch === generation) busy.value = false }
}
function setViewport(device: 'desktop' | 'mobile') {
  if (device !== viewport.value) void open(false, currentURL.value, 'resize', device)
}
function onDesktopStatus(status: string) {
  connected.value = status === 'connected'
  if (connected.value && props.active) void open(false, currentURL.value)
}
function syncFullscreen() { fullscreen.value = Boolean(root.value && document.fullscreenElement === root.value) }
async function leaveFullscreen() {
  if (root.value && document.fullscreenElement === root.value) await document.exitFullscreen().catch(() => {})
}
async function toggleFullscreen() {
  try {
    if (fullscreen.value) await leaveFullscreen()
    else if (root.value?.requestFullscreen) await root.value.requestFullscreen()
    else actionError.value = t('workspace.previewFullscreenUnavailable')
  } catch { actionError.value = t('workspace.previewFullscreenUnavailable') }
}
watch(() => props.enabled, enabled => { if (!enabled) connected.value = false })
watch(() => [props.active, props.revision] as const, () => {
  clearTimeout(refreshTimer)
  if (!props.active) void leaveFullscreen()
  if (props.active && connected.value && result.value?.status !== 'ready') {
    refreshTimer = setTimeout(() => { void open() }, 500)
  }
})
watch(() => props.sessionId, () => {
  generation++; clearTimeout(refreshTimer)
  void leaveFullscreen()
  address.value = '/'; currentURL.value = ''; result.value = null; busy.value = false; connected.value = false
  viewport.value = 'desktop'; actionError.value = ''
})
onMounted(() => document.addEventListener('fullscreenchange', syncFullscreen))
onBeforeUnmount(() => {
  generation++; clearTimeout(refreshTimer)
  void leaveFullscreen()
  document.removeEventListener('fullscreenchange', syncFullscreen)
})
</script>

<style scoped lang="less">
.workspace-browser { display: flex; flex: 1; flex-direction: column; min-width: 0; min-height: 0; height: 100%; background: var(--td-bg-color-container); color: var(--td-text-color-primary); }
.workspace-browser:fullscreen { width: 100vw; height: 100dvh; }
.preview-toolbar { display: flex; align-items: center; gap: 10px; min-height: 48px; padding: 0 12px; border-bottom: 1px solid var(--td-component-stroke); flex-shrink: 0; }
button { cursor: pointer; font: inherit; color: inherit; transition: background-color .15s, color .15s; }
button:disabled { opacity: .4; cursor: default; }
button:focus-visible, input:focus-visible, summary:focus-visible { outline: 2px solid var(--td-brand-color); outline-offset: 2px; }
.icon-button { display: inline-flex; align-items: center; justify-content: center; width: 28px; height: 28px; padding: 0; border: 0; border-radius: 5px; background: transparent; color: var(--td-text-color-secondary); flex-shrink: 0; }
.icon-button:hover:not(:disabled) { background: var(--td-bg-color-container-hover); color: var(--td-text-color-primary); }
.preview-devices { display: flex; padding: 2px; border: 1px solid var(--td-component-stroke); border-radius: 7px; gap: 2px; }
.preview-devices .selected { background: var(--td-bg-color-secondarycontainer); color: var(--td-text-color-primary); }
.preview-location { display: flex; align-items: center; flex: 1; min-width: 0; gap: 2px; padding: 2px; border-radius: 7px; background: var(--td-bg-color-secondarycontainer); }
.preview-location input { width: 0; flex: 1; min-width: 0; border: 0; outline-offset: 0; background: transparent; color: inherit; padding: 6px 4px; font: inherit; font-size: var(--app-text-sm); }
.location-submit { position: absolute; width: 1px; height: 1px; padding: 0; clip-path: inset(50%); overflow: hidden; border: 0; }
.preview-stage { flex: 1; min-height: 0; min-width: 0; display: flex; justify-content: center; overflow: hidden; background: var(--td-bg-color-container); }
.preview-stage.is-mobile { padding: 20px 16px; background: var(--td-bg-color-secondarycontainer); }
.preview-screen { flex: 1; min-width: 0; min-height: 0; position: relative; }
.preview-screen.is-mobile { flex: none; height: 100%; width: auto; max-width: 390px; overflow: hidden; border-radius: 14px; outline: 1px solid var(--td-component-stroke); box-shadow: 0 4px 16px rgb(0 0 0 / 5%); background: var(--td-bg-color-container); }
.preview-empty { display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 16px; height: 100%; padding: 32px; box-sizing: border-box; text-align: center; color: var(--td-text-color-placeholder); }
.preview-empty p { max-width: 320px; margin: 0; line-height: 1.7; font-size: var(--app-text-sm); }
.preview-skeleton { width: 144px; height: 100px; border: 1px solid var(--td-component-stroke); border-radius: 10px; padding: 12px; box-sizing: border-box; display: grid; grid-template-columns: 2fr 1fr; gap: 8px; animation: preview-pulse 1.8s ease-in-out infinite; }
.preview-skeleton div { grid-column: 1 / -1; height: 6px; width: 38%; border-radius: 3px; background: var(--td-bg-color-secondarycontainer); }
.preview-skeleton i { border-radius: 4px; background: var(--td-bg-color-secondarycontainer); }
.preview-skeleton i:last-child { height: 6px; width: 75%; }
.preview-notice { flex-shrink: 0; max-height: 35%; overflow: auto; padding: 10px 14px; border-bottom: 1px solid var(--td-component-stroke); color: var(--td-text-color-secondary); font-size: var(--app-text-sm); }
.preview-notice__heading { display: flex; align-items: flex-start; gap: 8px; }
.preview-notice__heading .t-icon { flex-shrink: 0; margin-top: 2px; }
.preview-notice p { margin: 0; line-height: 1.6; }
.preview-notice details { margin-top: 8px; }
.preview-notice summary { cursor: pointer; }
.preview-notice pre { max-width: 100%; max-height: 120px; overflow: auto; white-space: pre-wrap; overflow-wrap: anywhere; font-size: 12px; }
.preview-actions { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 10px; }
.preview-actions button { padding: 5px 9px; border: 1px solid var(--td-component-stroke); border-radius: 6px; background: var(--td-bg-color-container); }
.preview-actions button:hover { background: var(--td-bg-color-container-hover); }
.preview-status { display: flex; align-items: center; justify-content: space-between; padding: 0 12px; height: 28px; flex-shrink: 0; border-top: 1px solid var(--td-component-stroke); color: var(--td-text-color-placeholder); font-size: 11px; font-variant-numeric: tabular-nums; }
.preview-status span:first-child { display: flex; align-items: center; gap: 6px; }
.preview-status i { width: 5px; height: 5px; border-radius: 50%; background: var(--td-text-color-placeholder); }
.preview-status i.online { background: var(--td-success-color); }
.spinning { animation: wk-spin 1s linear infinite; }
@keyframes preview-pulse { 50% { opacity: .45; } }
@media (prefers-reduced-motion: reduce) { .preview-skeleton, .spinning { animation: none; } }
@media (max-width: 500px) { .preview-toolbar { gap: 6px; padding: 0 8px; } }
</style>
