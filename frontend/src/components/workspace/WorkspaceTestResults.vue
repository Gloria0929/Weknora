<template>
  <section class="check-panel" :aria-label="t('workspace.checksTitle')">
    <header class="check-heading">
      <div>
        <h2>{{ t('workspace.checksTitle') }}</h2>
        <p>{{ t('workspace.checksDescription') }}</p>
      </div>
      <button class="check-request" type="button" @click="emit('ask', t('workspace.testPrompt'))">
        <t-icon name="check-circle" />{{ t('workspace.requestChecks') }}
      </button>
    </header>

    <p v-if="error" class="check-notice" role="alert">{{ error }}</p>
    <div v-if="running" class="check-notice" role="status"><t-icon name="loading" class="spinning" />{{ t('workspace.checkRunning') }}</div>

    <section class="check-overview" :aria-label="t('workspace.checkOverview')">
      <div class="check-overview__heading" role="status" aria-live="polite">
        <t-icon :name="overviewIcon" :class="overviewStatus" size="20px" />
        <strong>{{ t(`workspace.${overviewLabel}`) }}</strong>
        <span>{{ t('workspace.checkCount', { count: scopedEntries.length }) }}</span>
      </div>
      <div class="check-stats">
        <div v-for="status in statuses" :key="status" :class="status">
          <strong>{{ counts[status] }}</strong>
          <span>{{ t(`workspace.${statusKeys[status]}`) }}</span>
        </div>
      </div>
    </section>

    <div class="check-filters">
      <div class="check-scopes" role="group" :aria-label="t('workspace.checkScope')">
        <button type="button" :aria-pressed="scope === 'checks'" @click="scope = 'checks'">{{ t('workspace.validationChecks') }} <span>{{ checks.length }}</span></button>
        <button type="button" :aria-pressed="scope === 'all'" @click="scope = 'all'">{{ t('workspace.allOperations') }} <span>{{ entries.length }}</span></button>
      </div>
      <button class="check-failed-filter" type="button" :aria-pressed="failedOnly" @click="failedOnly = !failedOnly">
        <t-icon name="filter" size="14px" />{{ t('workspace.failuresOnly') }}
      </button>
    </div>

    <div v-if="!visibleEntries.length" class="check-empty">
      <t-icon :name="failedOnly ? 'check-circle' : 'task'" size="28px" />
      <h3>{{ t(failedOnly ? 'workspace.noMatchingFailures' : 'workspace.noCheckRecords') }}</h3>
      <p>{{ t(failedOnly ? 'workspace.failureFilterHint' : 'workspace.checksEmptyHint') }}</p>
      <button v-if="failedOnly" type="button" @click="failedOnly = false">{{ t('workspace.showAllChecks') }}</button>
      <button v-else-if="scope === 'checks' && entries.length" type="button" @click="scope = 'all'">{{ t('workspace.viewOtherOperations') }}</button>
    </div>

    <ol v-else class="check-list">
      <li v-for="entry in visibleEntries" :key="entry.id" class="check-entry">
        <div class="check-entry__head">
          <t-icon :name="statusIcons[entry.status]" :class="[entry.status, { spinning: entry.status === 'running' }]" size="18px" />
          <div class="check-entry__title">
            <strong>{{ t(`workspace.${kindKeys[entry.kind]}`) }}</strong>
            <span :class="entry.status">{{ t(`workspace.${statusKeys[entry.status]}`) }}</span>
          </div>
          <time v-if="entry.durationMs != null">{{ t('workspace.duration', { seconds: (entry.durationMs / 1000).toFixed(2) }) }}</time>
        </div>
        <ul v-if="entry.highlights.length" class="check-evidence" :aria-label="t('workspace.checkSummary')">
          <li v-for="(line, index) in entry.highlights" :key="index">{{ line }}</li>
        </ul>
        <p v-else class="check-fallback">{{ fallbackText(entry) }}</p>
        <div class="check-entry__actions">
          <details class="check-details">
            <summary>{{ t('workspace.executionDetails') }}</summary>
            <div class="check-details__body">
              <code>$ {{ entry.command }}</code>
              <span v-if="entry.exitCode != null" class="check-exit">{{ t('workspace.exitCode', { code: entry.exitCode }) }}</span>
              <p v-if="entry.logTruncated">{{ t('workspace.checkLogTruncated') }}</p>
              <pre>{{ entry.log || t(entry.status === 'running' ? 'workspace.waitingCheckOutput' : 'workspace.noOutput') }}</pre>
            </div>
          </details>
          <button v-if="entry.status === 'failed'" class="check-fix" type="button" @click="requestFix(entry)">{{ t('workspace.analyzeCheckFailure') }}<t-icon name="arrow-up" size="14px" /></button>
        </div>
      </li>
    </ol>
    <p class="check-footnote">{{ t('workspace.checksFootnote') }}</p>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { describeWorkspaceCheck, type WorkspaceCheck } from '@/utils/workspaceChecks'
import type { WorkspaceCommandRun } from '@/utils/workspaceEvents'

const props = defineProps<{ sessionId: string; runs: WorkspaceCommandRun[]; running?: boolean; error?: string }>()
const emit = defineEmits<{ ask: [prompt: string] }>()
const { t } = useI18n()
const scope = ref<'checks' | 'all'>('checks')
const failedOnly = ref(false)
const statuses = ['passed', 'failed', 'running', 'unknown'] as const
const statusKeys = { passed: 'checkPassed', failed: 'checkFailed', running: 'checkRunning', unknown: 'checkUnknown' } as const
const statusIcons = { passed: 'check-circle', failed: 'close-circle', running: 'loading', unknown: 'help-circle' }
const kindKeys = { tests: 'testChecks', build: 'buildChecks', types: 'typeChecks', lint: 'lintChecks', other: 'otherOperation' } as const
const entries = computed(() => props.runs.map(describeWorkspaceCheck))
const checks = computed(() => entries.value.filter(entry => entry.kind !== 'other'))
const scopedEntries = computed(() => (scope.value === 'checks' ? checks.value : entries.value))
const counts = computed(() => Object.fromEntries(statuses.map(status => [status, scopedEntries.value.filter(entry => entry.status === status).length])) as Record<WorkspaceCommandRun['status'], number>)
const overviewStatus = computed(() => counts.value.failed ? 'failed' : counts.value.running ? 'running' : counts.value.unknown || !scopedEntries.value.length ? 'unknown' : 'passed')
const overviewIcon = computed(() => statusIcons[overviewStatus.value])
const overviewLabel = computed(() => !scopedEntries.value.length ? 'checksNotVerified' : ({ failed: 'checksNeedAttention', running: 'checksInProgress', unknown: 'checksIncomplete', passed: 'checksCompleted' } as const)[overviewStatus.value])
const visibleEntries = computed(() => (scope.value === 'checks' ? checks.value : entries.value).filter(entry => !failedOnly.value || entry.status === 'failed'))
function fallbackText(entry: WorkspaceCheck) {
  if (entry.status === 'running') return t('workspace.waitingCheckOutput')
  if (entry.status === 'unknown') return t('workspace.checkResultUnknown')
  if (entry.status === 'failed') return entry.exitCode != null ? t('workspace.checkExitFailure', { code: entry.exitCode }) : t('workspace.checkFailureDetails')
  return t('workspace.checkCompletedHint')
}
function requestFix(entry: WorkspaceCheck) {
  emit('ask', `${t('workspace.fixPrompt', { command: entry.command })}\n\n${entry.log.slice(-12_000)}`)
}
watch(() => props.sessionId, () => { scope.value = 'checks'; failedOnly.value = false })
</script>

<style scoped lang="less">
.check-panel { flex: 1; min-height: 0; min-width: 0; overflow: auto; padding: 22px 18px; color: var(--td-text-color-primary); }
.check-heading { display: flex; flex-wrap: wrap; align-items: flex-start; gap: 14px; margin-bottom: 22px; }
.check-heading > div { flex: 1; min-width: 170px; }
h2 { margin: 0; font-size: 20px; font-weight: 600; letter-spacing: -.4px; }
.check-heading p { margin: 8px 0 0; line-height: 1.7; color: var(--td-text-color-secondary); font-size: 12px; }
button { display: inline-flex; align-items: center; justify-content: center; gap: 6px; border: 0; background: transparent; color: inherit; cursor: pointer; font: inherit; font-size: 12px; border-radius: 6px; padding: 7px 9px; transition: background var(--app-motion-fast) ease; }
button:hover { background: var(--td-bg-color-container-hover); }
button:active { transform: translateY(1px); }
button:focus-visible, summary:focus-visible { outline: 2px solid var(--td-brand-color); outline-offset: 2px; }
.check-request { border: 1px solid var(--td-component-stroke); white-space: nowrap; }
.check-overview { padding: 15px 0; border-top: 1px solid var(--td-component-stroke); border-bottom: 1px solid var(--td-component-stroke); }
.check-overview__heading { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; font-size: 13px; }
.check-overview__heading > span { margin-left: auto; color: var(--td-text-color-placeholder); font-size: 11px; }
.check-stats { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 10px; padding-top: 18px; }
.check-stats > div { display: flex; flex-direction: column; gap: 5px; }
.check-stats strong { font-size: 24px; font-weight: 500; font-variant-numeric: tabular-nums; }
.check-stats span { color: var(--td-text-color-secondary); font-size: 11px; }
.passed { color: var(--td-success-color); } .failed { color: var(--td-error-color); } .running { color: var(--td-brand-color); } .unknown { color: var(--td-text-color-placeholder); }
.check-filters { display: flex; flex-wrap: wrap; gap: 8px; justify-content: space-between; align-items: center; margin: 18px 0 4px; }
.check-scopes { display: flex; gap: 3px; }
.check-scopes button[aria-pressed="true"], .check-failed-filter[aria-pressed="true"] { background: var(--td-bg-color-secondarycontainer); color: var(--td-text-color-primary); }
.check-scopes button { color: var(--td-text-color-secondary); }
.check-scopes span { font-variant-numeric: tabular-nums; color: var(--td-text-color-placeholder); }
.check-list { list-style: none; padding: 0; margin: 0; }
.check-entry { padding: 18px 0; border-bottom: 1px solid var(--td-component-stroke); }
.check-entry__head { display: flex; align-items: flex-start; gap: 9px; }
.check-entry__head > .t-icon { flex-shrink: 0; margin-top: 2px; }
.check-entry__title { flex: 1; display: flex; flex-wrap: wrap; align-items: center; gap: 8px; min-width: 0; font-size: 13px; }
.check-entry__title strong { font-weight: 500; }
.check-entry__title span, time { font-size: 11px; }
time { color: var(--td-text-color-placeholder); white-space: nowrap; font-variant-numeric: tabular-nums; }
.check-evidence { list-style: none; margin: 10px 0 4px 27px; padding: 0; font-size: 12px; line-height: 1.7; overflow-wrap: anywhere; }
.check-evidence li + li { margin-top: 4px; }
.check-fallback { margin: 10px 0 4px 27px; color: var(--td-text-color-secondary); font-size: 12px; line-height: 1.7; }
.check-entry__actions { display: flex; flex-wrap: wrap; align-items: flex-start; gap: 8px; padding-left: 27px; }
.check-details { flex: 1; min-width: 0; font-size: 12px; }
.check-details[open] { flex-basis: 100%; }
summary { padding: 7px 0; cursor: pointer; color: var(--td-text-color-secondary); width: fit-content; }
.check-details__body { padding: 12px; border-radius: 6px; background: var(--td-bg-color-secondarycontainer); }
.check-details code { display: block; font: 11px/1.7 var(--app-font-family-mono, monospace); white-space: pre-wrap; overflow-wrap: anywhere; }
.check-exit { display: block; margin: 8px 0; color: var(--td-text-color-placeholder); font-size: 11px; }
.check-details pre { margin: 12px 0 0; max-height: 280px; overflow: auto; font: 11px/1.7 var(--app-font-family-mono, monospace); white-space: pre-wrap; overflow-wrap: anywhere; }
.check-details p { font-size: 11px; color: var(--td-text-color-placeholder); }
.check-fix { color: var(--td-text-color-secondary); }
.check-empty { padding: 36px 12px; text-align: center; color: var(--td-text-color-placeholder); }
.check-empty h3 { font-size: 14px; font-weight: 500; color: var(--td-text-color-secondary); margin: 14px 0 8px; }
.check-empty p { font-size: 12px; line-height: 1.8; max-width: 320px; margin: 0 auto 12px; }
.check-footnote { margin: 18px 0 0; color: var(--td-text-color-placeholder); font-size: 11px; line-height: 1.7; }
.check-notice { display: flex; gap: 8px; align-items: center; font-size: 12px; line-height: 1.7; color: var(--td-text-color-secondary); }
.spinning { animation: check-spin 1s linear infinite; }
@keyframes check-spin { to { transform: rotate(360deg); } }
@media (prefers-reduced-motion: reduce) { *, *::before, *::after { animation: none !important; transition: none !important; } }
</style>
