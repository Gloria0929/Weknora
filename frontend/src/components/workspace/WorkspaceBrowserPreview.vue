<template>
  <section class="workspace-browser">
    <div class="preview-modes" role="group" :aria-label="t('workspace.previewMode')">
      <button type="button" :aria-pressed="mode === 'url'" @click="mode = 'url'">{{ t('workspace.urlPreview') }}</button>
      <button type="button" :aria-pressed="mode === 'desktop'" :aria-label="desktopUnavailable ? `${t('workspace.remoteDesktop')} · ${t('workspace.unavailableBadge')}` : t('workspace.remoteDesktop')" :title="desktopUnavailable ? t('workspace.desktopUnsupportedHint') : t('workspace.remoteDesktop')" @click="openDesktop">
        {{ t('workspace.remoteDesktop') }}<span v-if="desktopUnavailable" class="desktop-unavailable">{{ t('workspace.unavailableBadge') }}</span>
      </button>
    </div>
    <WorkspaceUrlPreview v-show="mode === 'url'" :key="sessionId" :session-id="sessionId" :revision="revision" :active="active && mode === 'url'" @ask="emit('ask', $event)" @open-terminal="emit('open-terminal')" />
    <WorkspaceDesktopPreview v-if="mode === 'desktop'" :key="desktopGeneration" :session-id="sessionId" :agent-id="agentId" :agent-source-tenant-id="agentSourceTenantId"
      :enabled="enabled && !desktopUnavailable" :loading="loading" :active="active && mode === 'desktop'" @status="desktopStatus = $event" @retry="retryDesktop" />
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import WorkspaceDesktopPreview from './WorkspaceDesktopPreview.vue'
import WorkspaceUrlPreview from './WorkspaceUrlPreview.vue'
const props = defineProps<{ sessionId: string; agentId?: string; agentSourceTenantId?: string | number | null; enabled: boolean; loading: boolean; active: boolean; revision?: number }>()
const emit = defineEmits<{ ask: [prompt: string]; 'open-terminal': []; retry: [] }>()
const { t } = useI18n()
const mode = ref<'url' | 'desktop'>('url')
const desktopStatus = ref('')
const desktopGeneration = ref(0)
const desktopUnavailable = computed(() => !props.loading && (!props.enabled || desktopStatus.value === 'unsupported'))
function resetDesktop() {
  desktopStatus.value = ''
  desktopGeneration.value++
}
function retryDesktop() {
  resetDesktop()
  emit('retry')
}
function openDesktop() {
  mode.value = 'desktop'
  // Unsupported is a result for the previous connection, not a permanent
  // veto on checking the session again after its configuration changes.
  if (desktopUnavailable.value) retryDesktop()
}
watch(() => [props.sessionId, props.agentId, props.agentSourceTenantId], resetDesktop)
watch(() => props.enabled, () => {
  if (desktopStatus.value === 'unsupported') resetDesktop()
  else desktopStatus.value = ''
})
watch(() => props.revision, () => {
  if (desktopStatus.value === 'unsupported') resetDesktop()
})
watch(() => props.sessionId, () => { mode.value = 'url' })
</script>

<style scoped lang="less">
.workspace-browser { display: flex; flex: 1; flex-direction: column; min-width: 0; min-height: 0; height: 100%; background: var(--td-bg-color-container); }
.preview-modes { display: flex; gap: 4px; padding: 8px 12px; border-bottom: 1px solid var(--td-component-stroke); }
.preview-modes button { border: 0; border-radius: 5px; padding: 6px 10px; color: var(--td-text-color-secondary); background: transparent; font: inherit; font-size: var(--app-text-sm); cursor: pointer; }
.preview-modes button[aria-pressed="true"] { color: var(--td-text-color-primary); background: var(--td-bg-color-secondarycontainer); }
.desktop-unavailable { margin-left: 6px; font-size: 11px; color: var(--td-text-color-placeholder); }
.preview-modes button:focus-visible { outline: 2px solid var(--td-brand-color); outline-offset: 2px; }
</style>
