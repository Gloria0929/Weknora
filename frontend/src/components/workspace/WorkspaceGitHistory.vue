<template>
  <section class="git-history" :aria-label="t('workspace.commitHistory')">
    <header class="git-history__header">
      <button type="button" class="git-history__toggle" :aria-expanded="!collapsed" @click="collapsed = !collapsed">
        <t-icon :name="collapsed ? 'chevron-right' : 'chevron-down'" size="13px" />
        <t-icon name="history" size="14px" />
        <span>{{ t('workspace.commitHistory') }}</span>
        <span v-if="report?.commits.length" class="git-history__count" :title="t('workspace.historyLoadedCount', { count: report.commits.length })">{{ report.commits.length }}</span>
      </button>
      <button type="button" class="git-history__refresh" :disabled="loading" :title="t('workspace.historyRefresh')"
        :aria-label="t('workspace.historyRefresh')" @click="emit('refresh')">
        <t-icon name="refresh" size="13px" :class="{ 'is-spinning': loading }" />
      </button>
    </header>
    <div v-show="!collapsed" class="git-history__content" :aria-busy="loading">
      <p v-if="error" class="git-history__error" role="alert">{{ error }}</p>
      <div v-if="loading && !report?.commits.length" class="git-history__skeleton" role="status" :aria-label="t('workspace.historyLoading')">
        <div v-for="row in 3" :key="row" class="git-history__skeleton-row"><i /><span /><small /></div>
      </div>
      <p v-else-if="!report?.commits.length && !error" class="git-history__empty">{{ t('workspace.historyEmpty') }}</p>
      <ol v-else-if="report?.commits.length" class="git-history__list">
        <li v-for="commit in report.commits" :key="commit.sha" class="git-history__item" :class="`is-${commit.status}`">
          <details class="git-history__entry">
            <summary class="git-history__row" :title="`${commit.subject}\n${commit.author} · ${formatMessageTimestamp(commit.committed_at)}\n${commit.sha}`">
              <span class="git-history__node" aria-hidden="true">
                <t-icon v-if="commit.parents?.length > 1" name="git-merge" size="14px" />
                <i v-else />
              </span>
              <span class="git-history__info">
                <span class="git-history__subject-line">
                  <span class="git-history__subject">{{ commit.subject }}</span>
                  <span v-if="commit.status === 'outgoing'" class="git-history__state is-outgoing">{{ t('workspace.historyOutgoing') }}</span>
                  <span v-else-if="commit.status === 'incoming'" class="git-history__state is-incoming">{{ t('workspace.historyIncoming') }}</span>
                </span>
                <span class="git-history__meta">
                  <span class="git-history__author">{{ commit.author }}</span>
                  <span aria-hidden="true">·</span>
                  <time :datetime="commit.committed_at">{{ relativeCommitTime(commit.committed_at) }}</time>
                  <code>{{ commit.short_sha }}</code>
                </span>
                <span v-if="commit.refs" class="git-history__refs" :title="commit.refs">
                  <t-icon name="git-branch" size="11px" /><span>{{ commit.refs }}</span>
                </span>
              </span>
              <button type="button" class="git-history__revert" v-if="canRevert(commit, report?.head)" :title="t('workspace.historyRevert')"
                :aria-label="t('workspace.historyRevert')" @click.stop.prevent="emit('revert', commit)"><t-icon name="rollback" size="13px" /></button>
              <button type="button" class="git-history__copy" :title="t('workspace.copyCommitHash')" :aria-label="t('workspace.copyCommitHash')"
                @click.stop.prevent="copyWithToast(commit.sha, 'workspace.commitHashCopied')"><t-icon name="copy" size="13px" /></button>
            </summary>
            <div class="git-history__details">
              <p class="git-history__full-subject">{{ commit.subject }}</p>
              <pre v-if="commit.body">{{ commit.body }}</pre>
              <span>{{ commit.author }} · {{ formatMessageTimestamp(commit.committed_at) }}</span>
              <code>{{ commit.sha }}</code>
            </div>
          </details>
        </li>
      </ol>
      <button v-if="report?.has_more" type="button" class="git-history__more" :disabled="loading" @click="emit('loadMore')">
        <t-icon v-if="loading" name="loading" size="12px" class="is-spinning" />
        {{ t(loading ? 'workspace.historyLoading' : 'workspace.historyLoadMore') }}
      </button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ProgrammingGitHistory, ProgrammingGitHistoryCommit } from '@/api/programming'
import { copyWithToast } from '@/utils/clipboard'
import { formatMessageTimestamp } from '@/utils/messageTimestamp'

defineProps<{ report?: ProgrammingGitHistory | null; loading?: boolean; error?: string }>()
const emit = defineEmits<{ refresh: []; loadMore: []; revert: [commit: ProgrammingGitHistoryCommit] }>()
const { t, locale } = useI18n()
const collapsed = ref(false)
const relativeTime = computed(() => new Intl.RelativeTimeFormat(locale.value, { numeric: 'auto' }))
const calendarTime = computed(() => new Intl.DateTimeFormat(locale.value, { year: 'numeric', month: 'short', day: 'numeric' }))
function relativeCommitTime(value: string) {
  const date = new Date(value)
  if (!Number.isFinite(date.getTime())) return ''
  const seconds = Math.round((date.getTime() - Date.now()) / 1000)
  if (Math.abs(seconds) >= 7 * 86400) return calendarTime.value.format(date)
  if (Math.abs(seconds) >= 86400) return relativeTime.value.format(Math.round(seconds / 86400), 'day')
  if (Math.abs(seconds) >= 3600) return relativeTime.value.format(Math.round(seconds / 3600), 'hour')
  if (Math.abs(seconds) >= 60) return relativeTime.value.format(Math.round(seconds / 60), 'minute')
  return relativeTime.value.format(seconds, 'second')
}
// Only a commit that is still local can be undone: published history cannot be
// rewritten from here (there is no force push), and incoming commits are not on
// this branch at all. "unknown" means the branch has no upstream yet, so every
// commit on it is still local.
function canRevert(commit: ProgrammingGitHistoryCommit, head?: string) {
  if (!commit.sha || commit.sha === head) return false
  return commit.status === 'outgoing' || commit.status === 'unknown'
}
</script>

<style scoped lang="less">
.git-history { flex: 1 1 auto; min-height: 0; display: flex; flex-direction: column; margin-top: 18px; border-top: 1px solid var(--td-component-stroke); }
.git-history__header { display: flex; align-items: center; gap: 8px; padding: 8px 0; }
.git-history__toggle { display: flex; flex: 1; align-items: center; gap: 6px; min-width: 0; border: 0; padding: 2px 0; background: transparent; color: var(--td-text-color-secondary); font: inherit; font-size: 12px; font-weight: 600; text-align: left; cursor: pointer; }
.git-history__count { margin-left: 2px; padding: 0 5px; border-radius: 4px; background: var(--td-bg-color-secondarycontainer); color: var(--td-text-color-secondary); font-size: 10px; font-variant-numeric: tabular-nums; line-height: 17px; }
.git-history__refresh, .git-history__copy, .git-history__revert { display: inline-flex; align-items: center; justify-content: center; flex: none; width: 24px; height: 24px; padding: 0; border: 0; border-radius: 4px; background: transparent; color: var(--td-text-color-placeholder); cursor: pointer; }
.git-history__refresh:hover:not(:disabled), .git-history__copy:hover { background: var(--td-bg-color-container-hover); color: var(--td-text-color-primary); }
.git-history__revert:hover:not(:disabled) { background: var(--td-bg-color-container-hover); color: var(--td-warning-color); }
.git-history button:disabled { opacity: 0.5; cursor: not-allowed; }
.git-history button:focus-visible, .git-history summary:focus-visible { outline: 2px solid var(--td-brand-color); outline-offset: 2px; }
.git-history__content { flex: 1 1 auto; min-height: 0; overflow: auto; scrollbar-width: thin; }
.git-history__list { margin: 0; padding: 0; list-style: none; }
.git-history__item { position: relative; --history-node-color: var(--td-text-color-placeholder); }
.git-history__item::before { position: absolute; top: 21px; bottom: -21px; left: 13px; width: 1px; background: var(--td-component-stroke); content: ''; pointer-events: none; }
.git-history__item:last-child::before { display: none; }
.git-history__item.is-outgoing { --history-node-color: var(--td-warning-color); }
.git-history__item.is-incoming { --history-node-color: var(--td-brand-color); }
.git-history__row { position: relative; display: flex; align-items: flex-start; gap: 7px; padding: 8px 4px 8px 0; border-radius: 4px; cursor: pointer; list-style: none; transition: background 120ms ease; }
.git-history__row::-webkit-details-marker { display: none; }
.git-history__row:hover, .git-history__entry[open] .git-history__row { background: var(--td-bg-color-container-hover); }
.git-history__node { position: relative; display: flex; align-items: center; justify-content: center; flex: none; width: 27px; height: 24px; color: var(--history-node-color); }
.git-history__node i { width: 8px; height: 8px; border: 1.5px solid var(--history-node-color); border-radius: 50%; background: var(--td-bg-color-container); }
.git-history__info { display: flex; flex: 1; flex-direction: column; gap: 3px; min-width: 0; }
.git-history__subject-line { display: flex; align-items: center; gap: 6px; min-width: 0; }
.git-history__subject { flex: 1; min-width: 0; overflow: hidden; color: var(--td-text-color-primary); font-size: 12px; line-height: 22px; text-overflow: ellipsis; white-space: nowrap; }
.git-history__state { flex: none; padding: 0 4px; border-radius: 3px; font-size: 10px; font-weight: 500; line-height: 17px; }
.git-history__state.is-outgoing { background: var(--td-warning-color-light); color: var(--td-warning-color); }
.git-history__state.is-incoming { background: var(--td-brand-color-light); color: var(--td-brand-color); }
.git-history__meta { display: flex; align-items: center; gap: 5px; min-width: 0; color: var(--td-text-color-placeholder); font-size: 10px; line-height: 16px; }
.git-history__author { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.git-history__meta time { flex: none; }
.git-history__meta code { flex: none; margin-left: auto; color: var(--td-text-color-secondary); font-family: var(--app-font-family-mono, monospace); font-size: 10px; }
.git-history__refs { display: flex; align-items: center; align-self: flex-start; gap: 4px; max-width: 100%; padding: 1px 5px; border-radius: 3px; background: var(--td-bg-color-secondarycontainer); color: var(--td-text-color-secondary); font-family: var(--app-font-family-mono, monospace); font-size: 10px; line-height: 16px; }
.git-history__refs span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.git-history__copy { margin-top: 1px; opacity: 0; }
.git-history__row:hover .git-history__copy, .git-history__row:focus-within .git-history__copy { opacity: 1; }
.git-history__details { display: flex; flex-direction: column; gap: 8px; margin: 0 8px 10px 34px; padding: 10px 0; color: var(--td-text-color-secondary); font-size: 11px; line-height: 1.6; overflow-wrap: anywhere; }
.git-history__full-subject { margin: 0; color: var(--td-text-color-primary); font-weight: 500; }
.git-history__details pre { max-height: 160px; margin: 0; overflow: auto; white-space: pre-wrap; font: inherit; }
.git-history__details code { font-family: var(--app-font-family-mono, monospace); font-size: 10px; }
.git-history__more { display: flex; align-items: center; justify-content: center; gap: 6px; width: 100%; margin: 6px 0; padding: 8px; border: 0; border-radius: 4px; background: transparent; color: var(--td-text-color-secondary); font: inherit; font-size: 11px; cursor: pointer; }
.git-history__more:hover:not(:disabled) { background: var(--td-bg-color-container-hover); color: var(--td-brand-color); }
.git-history__empty, .git-history__error { margin: 4px 0 10px; padding: 8px 12px; color: var(--td-text-color-placeholder); font-size: 12px; line-height: 1.5; }
.git-history__error { color: var(--td-error-color); }
.git-history__skeleton-row { display: flex; align-items: center; gap: 10px; height: 45px; padding: 0 10px; }
.git-history__skeleton-row i, .git-history__skeleton-row span, .git-history__skeleton-row small { border-radius: 3px; background: var(--td-bg-color-secondarycontainer); animation: history-pulse 1.2s ease-in-out infinite alternate; }
.git-history__skeleton-row i { width: 8px; height: 8px; border-radius: 50%; }
.git-history__skeleton-row span { flex: 1; height: 11px; }
.git-history__skeleton-row small { width: 42px; height: 9px; }
.is-spinning { animation: history-spin 1s linear infinite; }
@keyframes history-pulse { to { opacity: 0.4; } }
@keyframes history-spin { to { transform: rotate(360deg); } }
@media (hover: none) { .git-history__copy { opacity: 1; } }
@media (prefers-reduced-motion: reduce) { .git-history__row { transition: none; } .is-spinning, .git-history__skeleton-row i, .git-history__skeleton-row span, .git-history__skeleton-row small { animation: none; } }
</style>
