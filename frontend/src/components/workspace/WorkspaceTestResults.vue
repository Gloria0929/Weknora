<template>
  <section class="tests-panel" :aria-label="t('workspace.testCasesTitle')">
    <header class="tests-heading">
      <div>
        <h2>{{ t('workspace.testCasesTitle') }}</h2>
        <p>{{ t('workspace.testCasesDescription') }}</p>
      </div>
      <button class="ask-agent" type="button" @click="emit('ask', t('workspace.testPrompt'))">
        <t-icon name="chat" />{{ t('workspace.requestChecks') }}
      </button>
    </header>

    <p v-if="discoveryLoading" class="notice" role="status"><t-icon name="loading" class="spinning" />{{ t('workspace.testDiscoveryLoading') }}</p>
    <p v-else-if="discoveryError" class="notice" role="alert">{{ discoveryError }}</p>
    <div v-else-if="!testCases.length" class="empty-tests">
      <t-icon name="task" size="24px" />
      <span>{{ t('workspace.noDiscoveredTests') }}</span>
    </div>
    <ul v-else class="test-list">
      <li v-for="testCase in testCases" :key="testCase.id" class="test-item">
        <div class="test-info">
          <div class="test-title">
            <strong>{{ testCase.name }}</strong>
          </div>
          <code>{{ testCase.path }}</code>
          <span v-if="testCaseStates[testCase.id]" class="test-state" :class="testCaseStates[testCase.id].status">
            <t-icon :name="stateIcon(testCaseStates[testCase.id].status)" :class="{ spinning: testCaseStates[testCase.id].status === 'running' }" size="14px" />
            {{ t(stateLabel(testCaseStates[testCase.id].status)) }}
            <time v-if="testCaseStates[testCase.id].durationMs">{{ t('workspace.duration', { seconds: (testCaseStates[testCase.id].durationMs! / 1000).toFixed(1) }) }}</time>
          </span>
          <span v-else-if="!testCase.command" class="test-unavailable">{{ t('workspace.manualRunUnavailable') }}</span>
          <details v-if="testCaseStates[testCase.id] && testCaseStates[testCase.id].status !== 'running'" class="test-output">
            <summary>{{ t('workspace.testCaseOutput') }}</summary>
            <code v-if="testCaseStates[testCase.id].command">$ {{ testCaseStates[testCase.id].command }}</code>
            <pre>{{ testCaseStates[testCase.id].output || t('workspace.noOutput') }}</pre>
          </details>
        </div>
        <div class="test-actions">
          <button type="button" :disabled="!testCase.command || running" :title="!testCase.command ? t('workspace.manualRunUnavailable') : undefined" @click="emit('runCase', testCase, 'manual')">
            <t-icon name="play-circle" size="14px" />{{ t('workspace.manualRunCase') }}
          </button>
          <button type="button" :disabled="running" @click="emit('runCase', testCase, 'ai')">
            <t-icon name="chat" size="14px" />{{ t('workspace.aiRunCase') }}
          </button>
        </div>
      </li>
    </ul>
    <p v-if="error" class="notice" role="alert">{{ error }}</p>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { WorkspaceTestCase } from '@/utils/workspaceTestCases'

type TestCaseState = { status: 'running' | 'passed' | 'failed'; command?: string; output: string; durationMs?: number }
const props = defineProps<{ testCases?: WorkspaceTestCase[]; testCaseStates?: Record<string, TestCaseState>; discoveryLoading?: boolean; discoveryError?: string; running?: boolean; error?: string }>()
const emit = defineEmits<{ ask: [prompt: string]; runCase: [testCase: WorkspaceTestCase, mode: 'manual' | 'ai'] }>()
const { t } = useI18n()
const testCases = computed(() => props.testCases || [])
const testCaseStates = computed(() => props.testCaseStates || {})
function stateIcon(status: TestCaseState['status']) {
  return status === 'running' ? 'loading' : status === 'passed' ? 'check-circle-filled' : 'close-circle-filled'
}
function stateLabel(status: TestCaseState['status']) {
  return status === 'running' ? 'workspace.checkRunning' : status === 'passed' ? 'workspace.testCasePassed' : 'workspace.testCaseFailed'
}
</script>

<style scoped lang="less">
.tests-panel { flex: 1; min-height: 0; min-width: 0; overflow: auto; padding: 22px 18px; color: var(--td-text-color-primary); }
.tests-heading { display: flex; flex-wrap: wrap; align-items: flex-start; gap: 14px; margin-bottom: 22px; }
.tests-heading > div { flex: 1; min-width: 170px; }
h2 { margin: 0; font-size: 20px; font-weight: 600; letter-spacing: -.4px; }
.tests-heading p { margin: 8px 0 0; line-height: 1.7; color: var(--td-text-color-secondary); font-size: 12px; }
button { display: inline-flex; align-items: center; justify-content: center; gap: 6px; border: 0; background: transparent; color: inherit; cursor: pointer; font: inherit; font-size: 12px; border-radius: 6px; padding: 7px 9px; transition: background var(--app-motion-fast) ease; }
button:hover { background: var(--td-bg-color-container-hover); }
button:focus-visible, summary:focus-visible { outline: 2px solid var(--td-brand-color); outline-offset: 2px; }
button:disabled { cursor: not-allowed; opacity: .5; }
.ask-agent, .test-actions button { border: 1px solid var(--td-component-stroke); white-space: nowrap; }
.notice { display: flex; gap: 8px; align-items: center; font-size: 12px; line-height: 1.7; color: var(--td-text-color-secondary); }
.empty-tests { display: flex; align-items: center; gap: 9px; padding: 22px 12px; border: 1px dashed var(--td-component-stroke); border-radius: 6px; color: var(--td-text-color-placeholder); font-size: 12px; }
.test-list { list-style: none; margin: 0; padding: 0; border-top: 1px solid var(--td-component-stroke); }
.test-item { display: flex; align-items: center; gap: 12px; padding: 14px 0; border-bottom: 1px solid var(--td-component-stroke); }
.test-info { flex: 1; min-width: 0; display: flex; flex-direction: column; align-items: flex-start; gap: 5px; }
.test-title { display: flex; align-items: center; flex-wrap: wrap; gap: 8px; }
.test-title strong { font-size: 12px; font-weight: 500; overflow-wrap: anywhere; }
.test-info > code { color: var(--td-text-color-placeholder); font: 10px/1.5 var(--app-font-family-mono, monospace); overflow-wrap: anywhere; }
.test-state { display: inline-flex; align-items: center; gap: 5px; font-size: 11px; }
.test-state time { margin-left: 5px; color: var(--td-text-color-placeholder); font-variant-numeric: tabular-nums; }
.passed { color: var(--td-success-color); } .failed { color: var(--td-error-color); } .running { color: var(--td-brand-color); } .cancelled, .test-unavailable { color: var(--td-text-color-placeholder); }
.test-unavailable { font-size: 11px; }
.test-output { font-size: 11px; width: 100%; }
.test-output summary { padding: 2px 0; cursor: pointer; color: var(--td-text-color-secondary); width: fit-content; }
.test-output > code { display: block; white-space: pre-wrap; overflow-wrap: anywhere; }
.test-output pre { max-height: 200px; overflow: auto; width: 100%; box-sizing: border-box; margin: 6px 0; padding: 8px; border-radius: 4px; background: var(--td-bg-color-secondarycontainer); font: 10px/1.6 var(--app-font-family-mono, monospace); white-space: pre-wrap; overflow-wrap: anywhere; }
.test-actions { display: flex; flex-wrap: wrap; justify-content: flex-end; gap: 4px; }
.spinning { animation: test-spin 1s linear infinite; }
@keyframes test-spin { to { transform: rotate(360deg); } }
@media (prefers-reduced-motion: reduce) { *, *::before, *::after { animation: none !important; transition: none !important; } }
@media (max-width: 560px) { .test-item { align-items: flex-start; flex-direction: column; } .test-actions { justify-content: flex-start; } }
</style>
