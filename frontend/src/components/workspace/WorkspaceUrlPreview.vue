<template>
  <section
    ref="root"
    class="url-preview"
    :aria-label="t('workspace.urlPreview')"
  >
    <div class="url-toolbar">
      <div
        class="url-devices"
        role="group"
        :aria-label="t('workspace.previewDevice')"
      >
        <button
          type="button"
          :aria-pressed="!mobile"
          :title="t('workspace.previewDesktop')"
          :aria-label="t('workspace.previewDesktop')"
          @click="mobile = false"
        >
          <t-icon name="desktop" />
        </button>
        <button
          type="button"
          :aria-pressed="mobile"
          :title="t('workspace.previewMobile')"
          :aria-label="t('workspace.previewMobile')"
          @click="mobile = true"
        >
          <t-icon name="mobile" />
        </button>
      </div>
      <input
        :value="currentUrl ? '/' : ''"
        readonly
        :title="currentUrl"
        :aria-label="t('workspace.autoPreviewAddress')"
        :placeholder="t('workspace.autoPreviewAddress')"
      />
      <button
        type="button"
        :disabled="busy"
        :title="t('workspace.previewRefresh')"
        :aria-label="t('workspace.previewRefresh')"
        @click="reload"
      >
        <t-icon name="refresh" />
      </button>
      <a
        v-if="currentUrl"
        :href="currentUrl"
        target="_blank"
        rel="noopener noreferrer"
        :title="t('workspace.openExternal')"
        :aria-label="t('workspace.openExternal')"
        ><t-icon name="arrow-right-up"
      /></a>
      <button
        type="button"
        :title="t(fullscreen ? 'preview.exitFullscreen' : 'preview.fullscreen')"
        :aria-label="
          t(fullscreen ? 'preview.exitFullscreen' : 'preview.fullscreen')
        "
        @click="toggleFullscreen"
      >
        <t-icon :name="fullscreen ? 'fullscreen-exit' : 'fullscreen'" />
      </button>
    </div>
    <p v-if="error" class="url-notice" role="alert">{{ error }}</p>
    <p
      v-if="currentUrl && result?.status === 'error'"
      class="url-notice"
      role="status"
    >
      {{ t("workspace.autoPreviewRetrying") }}
    </p>
    <div v-if="!currentUrl" class="url-empty">
      <t-icon
        :name="busy ? 'loading' : 'internet'"
        size="32px"
        :class="{ spinning: busy }"
      />
      <h3>
        {{
          t(
            busy
              ? "workspace.autoPreviewDetecting"
              : "workspace.autoPreviewTitle",
          )
        }}
      </h3>
      <p role="status">{{ statusMessage }}</p>
      <details v-if="result?.detail">
        <summary>{{ t("workspace.previewDetails") }}</summary>
        <pre>{{ result.detail }}</pre>
      </details>
      <div v-if="!busy" class="url-actions">
        <button
          v-if="
            result?.status === 'not_running' ||
            result?.status === 'start_failed'
          "
          type="button"
          @click="refresh(true)"
        >
          {{ t("workspace.startPreview") }}
        </button>
        <button type="button" @click="refresh()">
          {{ t("workspace.retry") }}
        </button>
        <button
          v-if="result?.status === 'start_failed' || result?.status === 'error'"
          type="button"
          @click="emit('open-terminal')"
        >
          {{ t('workspace.viewTerminal') }}
        </button>
        <button
          type="button"
          @click="emit('ask', t('workspace.startUrlPreviewPrompt'))"
        >
          {{ t("workspace.fix") }}
        </button>
      </div>
    </div>
    <template v-else>
      <div class="url-stage" :class="{ mobile }">
        <iframe
          :key="frameKey"
          :src="currentUrl"
          sandbox="allow-scripts allow-same-origin allow-forms allow-downloads"
          referrerpolicy="no-referrer"
          :title="t('workspace.urlPreview')"
        />
      </div>
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, toRef, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useWorkspaceWebPreview } from "@/composables/useWorkspaceWebPreview";
const props = defineProps<{
  sessionId: string;
  active: boolean;
  revision?: number;
}>();
const emit = defineEmits<{ ask: [prompt: string]; 'open-terminal': [] }>();
const { t } = useI18n();
const root = ref<HTMLElement | null>(null);
const error = ref("");
const mobile = ref(false),
  fullscreen = ref(false);
const { result, currentUrl, busy, frameKey, refresh, reload } =
  useWorkspaceWebPreview(
    toRef(props, "sessionId"),
    toRef(props, "active"),
    toRef(props, "revision"),
  );
const statusMessage = computed(() =>
  t(
    busy.value
      ? "workspace.autoPreviewDetectingHint"
      : (
          {
            not_running: "workspace.autoPreviewWaiting",
            paused: "workspace.autoPreviewPaused",
            unsupported: "workspace.autoPreviewUnsupported",
            setup_required: "workspace.autoPreviewSetup",
            start_failed: "workspace.previewStartFailed",
            starting: "workspace.autoPreviewDetectingHint",
            error: "workspace.autoPreviewRetrying",
            ready: "workspace.autoPreviewWaiting",
          } as const
        )[result.value?.status || "not_running"],
  ),
);
function syncFullscreen() {
  fullscreen.value = Boolean(
    root.value && document.fullscreenElement === root.value,
  );
}
async function leaveFullscreen() {
  if (root.value && document.fullscreenElement === root.value)
    await document.exitFullscreen().catch(() => {});
}
async function toggleFullscreen() {
  try {
    if (fullscreen.value) await leaveFullscreen();
    else if (root.value?.requestFullscreen)
      await root.value.requestFullscreen();
    else error.value = t("workspace.previewFullscreenUnavailable");
  } catch {
    error.value = t("workspace.previewFullscreenUnavailable");
  }
}
watch(
  () => props.active,
  (active) => {
    if (!active) void leaveFullscreen();
  },
);
watch(
  () => props.sessionId,
  () => {
    void leaveFullscreen();
    error.value = "";
    mobile.value = false;
  },
);
onMounted(() => document.addEventListener("fullscreenchange", syncFullscreen));
onBeforeUnmount(() => {
  void leaveFullscreen();
  document.removeEventListener("fullscreenchange", syncFullscreen);
});
</script>

<style scoped lang="less">
.url-preview {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
}
.url-preview:fullscreen {
  height: 100dvh;
  width: 100vw;
}
.url-toolbar {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 8px 10px;
  border-bottom: 1px solid var(--td-component-stroke);
}
.url-toolbar input[readonly] {
  cursor: default;
}
.url-toolbar input {
  flex: 1;
  min-width: 0;
  padding: 8px;
  border: 0;
  border-radius: 6px;
  background: var(--td-bg-color-secondarycontainer);
  color: inherit;
  font: inherit;
  font-size: var(--app-text-sm);
}
.url-toolbar button,
.url-toolbar a {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  padding: 6px;
  border: 0;
  border-radius: 5px;
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;
}
.url-toolbar button:hover,
.url-toolbar a:hover,
.url-toolbar button[aria-pressed="true"] {
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-primary);
}
.url-toolbar button:disabled {
  opacity: 0.4;
  cursor: default;
}
button:focus-visible,
a:focus-visible,
input:focus-visible,
summary:focus-visible {
  outline: None;
  outline-offset: 2px;
}
.url-devices {
  display: flex;
  border: 1px solid var(--td-component-stroke);
  border-radius: 7px;
  padding: 2px;
}
.url-empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 32px;
  text-align: center;
  color: var(--td-text-color-secondary);
}
.url-empty h3 {
  font-size: 16px;
  font-weight: 500;
  color: var(--td-text-color-primary);
  margin: 18px 0 0;
}
.url-empty p {
  max-width: 420px;
  font-size: var(--app-text-sm);
  line-height: 1.8;
  margin: 12px 0 20px;
}
.url-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 8px;
}
.url-empty pre {
  max-width: 100%;
  max-height: 160px;
  overflow: auto;
  text-align: left;
  white-space: pre-wrap;
  font-size: 12px;
}
.spinning {
  animation: wk-spin 1s linear infinite;
}
.url-empty button {
  padding: 8px 12px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 6px;
  color: inherit;
  background: transparent;
  cursor: pointer;
  font: inherit;
  font-size: var(--app-text-sm);
}
.url-stage {
  flex: 1;
  min-height: 0;
  display: flex;
  justify-content: center;
  background: var(--td-bg-color-secondarycontainer);
}
.url-stage iframe {
  flex: 1;
  width: 100%;
  min-width: 0;
  height: 100%;
  border: 0;
  background: white;
}
.url-stage.mobile {
  padding: 16px 12px;
}
.url-stage.mobile iframe {
  max-width: 390px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 12px;
}
.url-notice {
  margin: 0;
  padding: 10px 12px;
  font-size: var(--app-text-sm);
  line-height: 1.7;
  color: var(--td-text-color-secondary);
}
</style>
