<template>
  <section ref="root" class="workspace-desktop" :aria-label="t('workspace.remoteDesktop')">
    <header class="desktop-toolbar">
      <span><t-icon name="desktop" size="16px" />{{ t('workspace.remoteDesktop') }}</span>
      <button type="button" :title="t(fullscreen ? 'preview.exitFullscreen' : 'preview.fullscreen')"
        :aria-label="t(fullscreen ? 'preview.exitFullscreen' : 'preview.fullscreen')" @click="toggleFullscreen">
        <t-icon :name="fullscreen ? 'fullscreen-exit' : 'fullscreen'" size="16px" />
      </button>
    </header>
    <p v-if="error" class="desktop-notice" role="alert">{{ error }}</p>
    <div v-if="loading" class="desktop-empty" role="status">
      <t-icon name="loading" size="28px" class="spinning" />
      <p>{{ t('workspace.desktopChecking') }}</p>
    </div>
    <div v-else-if="!enabled" class="desktop-empty" role="status">
      <t-icon name="desktop" size="32px" />
      <h3>{{ t('workspace.desktopUnavailable') }}</h3>
      <p>{{ t('workspace.desktopUnsupportedHint') }}</p>
    </div>
    <SandboxDesktop v-else :key="sessionId" :session-id="sessionId" :agent-id="agentId"
      :agent-source-tenant-id="agentSourceTenantId" appearance="preview" class="desktop-screen"
      @status="emit('status', $event)" />
  </section>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import SandboxDesktop from '@/views/chat/components/SandboxDesktop.vue'

const props = defineProps<{ sessionId: string; agentId?: string; agentSourceTenantId?: string | number | null; enabled: boolean; loading: boolean; active: boolean }>()
const emit = defineEmits<{ status: [value: string] }>()
const { t } = useI18n()
const root = ref<HTMLElement | null>(null)
const fullscreen = ref(false), error = ref('')
function syncFullscreen() { fullscreen.value = Boolean(root.value && document.fullscreenElement === root.value) }
async function leaveFullscreen() {
  if (root.value && document.fullscreenElement === root.value) await document.exitFullscreen().catch(() => {})
}
async function toggleFullscreen() {
  try {
    if (fullscreen.value) await leaveFullscreen()
    else if (root.value?.requestFullscreen) await root.value.requestFullscreen()
    else error.value = t('workspace.previewFullscreenUnavailable')
  } catch { error.value = t('workspace.previewFullscreenUnavailable') }
}
watch(() => props.active, active => { if (!active) void leaveFullscreen() })
watch(() => props.sessionId, () => { void leaveFullscreen(); error.value = '' })
onMounted(() => document.addEventListener('fullscreenchange', syncFullscreen))
onBeforeUnmount(() => {
  void leaveFullscreen()
  document.removeEventListener('fullscreenchange', syncFullscreen)
})
</script>

<style scoped lang="less">
.workspace-desktop { display: flex; flex: 1; flex-direction: column; min-width: 0; min-height: 0; background: var(--td-bg-color-container); color: var(--td-text-color-primary); }
.workspace-desktop:fullscreen { width: 100vw; height: 100dvh; }
.desktop-toolbar { display: flex; align-items: center; justify-content: space-between; gap: 12px; min-height: 48px; padding: 0 12px; border-bottom: 1px solid var(--td-component-stroke); flex-shrink: 0; }
.desktop-toolbar span { display: flex; align-items: center; gap: 8px; font-size: var(--app-text-sm); color: var(--td-text-color-secondary); }
.desktop-toolbar button { display: inline-flex; align-items: center; justify-content: center; width: 28px; height: 28px; padding: 0; border: 0; border-radius: 5px; background: transparent; color: var(--td-text-color-secondary); cursor: pointer; }
.desktop-toolbar button:hover { background: var(--td-bg-color-container-hover); }
.desktop-toolbar button:focus-visible { outline: 2px solid var(--td-brand-color); outline-offset: 2px; }
.desktop-empty { display: flex; flex: 1; flex-direction: column; align-items: center; justify-content: center; gap: 16px; padding: 32px; text-align: center; color: var(--td-text-color-secondary); }
.desktop-empty h3 { margin: 0; font-size: 16px; font-weight: 500; color: var(--td-text-color-primary); }
.desktop-empty p { max-width: 360px; margin: 0; line-height: 1.8; font-size: var(--app-text-sm); }
.desktop-screen { flex: 1; min-height: 0; }
.desktop-notice { padding: 10px 12px; margin: 0; font-size: var(--app-text-sm); color: var(--td-text-color-secondary); }
.spinning { animation: wk-spin 1s linear infinite; }
@media (prefers-reduced-motion: reduce) { .spinning { animation: none; } }
</style>
