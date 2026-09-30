<template>
  <section
    class="conversation-workspace"
    :aria-label="
      t(
        `workspace.${tab === 'source' ? 'source' : tab === 'preview' ? 'preview' : 'tests'}`,
      )
    "
  >
    <template v-if="tab !== 'tests'">
      <div class="workspace-body">
        <aside class="file-tree" :aria-label="t('workspace.files')">
          <div class="file-tree__scroll">
            <p
              v-if="busy && !treeRows.length"
              class="file-tree__hint"
              role="status"
            >
              {{ t("workspace.loading") }}
            </p>
            <p
              v-else-if="workspaceState === 'expired' && !treeRows.length"
              class="file-tree__hint"
              role="status"
            >
              {{ t("workspace.workspaceExpiredHint") }}
            </p>
            <p v-else-if="!treeRows.length" class="file-tree__hint">
              {{ t("workspace.noFiles") }}
            </p>
            <ul v-else class="file-tree__list">
              <li v-for="row in treeRows" :key="row.node.path">
                <button
                  type="button"
                  class="file-tree__row"
                  :class="{
                    'is-selected': file?.path === row.node.path,
                    'is-dir': row.node.kind === 'directory',
                  }"
                  :style="{ paddingLeft: `${10 + row.depth * 14}px` }"
                  :title="row.node.path"
                  :aria-expanded="
                    row.node.kind === 'directory' ? row.expanded : undefined
                  "
                  @click="activateNode(row.node)"
                >
                  <span class="file-tree__lead">
                    <t-icon
                      v-if="row.node.kind === 'directory'"
                      :name="row.expanded ? 'chevron-down' : 'chevron-right'"
                      size="14px"
                    />
                    <span
                      v-else-if="row.glyph.text"
                      class="file-tree__glyph"
                      aria-hidden="true"
                      >{{ row.glyph.text }}</span
                    >
                    <t-icon
                      v-else
                      :name="row.glyph.icon"
                      size="14px"
                      aria-hidden="true"
                    />
                  </span>
                  <span class="file-tree__name">{{ row.node.name }}</span>
                  <t-icon
                    v-if="row.loading"
                    name="loading"
                    size="13px"
                    class="spinning"
                  />
                </button>
              </li>
            </ul>
          </div>
        </aside>
        <div class="workspace-main">
          <div v-if="file" class="file-toolbar">
            <span class="file-toolbar__path" :title="file.path">{{ file.path }}</span>
            <div v-if="showViewSwitch" class="file-view-switch" role="group" :aria-label="t('workspace.fileView')">
              <button type="button" :aria-pressed="fileView !== 'preview'" @click="fileView = 'source'">{{ t('workspace.source') }}</button>
              <button type="button" :aria-pressed="fileView === 'preview'" @click="fileView = 'preview'">{{ t('workspace.preview') }}</button>
            </div>
            <button v-if="showFilePreview" type="button" :aria-pressed="mobilePreview" :title="t(mobilePreview ? 'workspace.desktop' : 'workspace.mobile')" :aria-label="t(mobilePreview ? 'workspace.desktop' : 'workspace.mobile')" @click="mobilePreview = !mobilePreview"><t-icon :name="mobilePreview ? 'desktop' : 'mobile'" /></button>
            <button v-else type="button" :title="t(copied ? 'workspace.copied' : 'workspace.copy')" :aria-label="t(copied ? 'workspace.copied' : 'workspace.copy')" @click="copySource"><t-icon :name="copied ? 'check' : 'copy'" /></button>
            <button type="button" :title="t('workspace.refresh')" :aria-label="t('workspace.refresh')" @click="refresh"><t-icon name="refresh" /></button>
          </div>
          <div v-if="error" class="workspace-notice" role="alert">
            <t-icon name="info-circle" /><span>{{ error }}</span
            ><button type="button" @click="refresh">
              {{ t("workspace.retry") }}
            </button>
          </div>
          <div
            v-if="!file && workspaceState !== 'expired'"
            class="workspace-empty"
          >
            <div class="empty-art" aria-hidden="true">
              <span>&lt;/&gt;</span>
            </div>
            <h2>{{ t("workspace.emptyTitle") }}</h2>
            <p>{{ t("workspace.emptyDescription") }}</p>
            <div class="empty-steps">
              <span
                >01 <b>{{ t("workspace.code") }}</b></span
              ><i></i
              ><span
                >02 <b>{{ t("workspace.preview") }}</b></span
              ><i></i
              ><span
                >03 <b>{{ t("workspace.test") }}</b></span
              >
            </div>
          </div>
          <template v-else-if="showFileDiff && file">
            <div class="diff-scroll" tabindex="0" :aria-label="file.path">
              <table class="diff-table">
                <tbody>
                  <template
                    v-for="(hunk, hunkIndex) in diffHunks"
                    :key="`hunk-${hunkIndex}`"
                  >
                    <tr class="diff-hunk-row">
                      <td colspan="3" class="diff-hunk">{{ hunk.header }}</td>
                    </tr>
                    <tr
                      v-for="(line, lineIndex) in hunk.lines"
                      :key="`line-${hunkIndex}-${lineIndex}`"
                      class="diff-row"
                      :class="`is-${line.type}`"
                    >
                      <td class="diff-gutter">{{ line.oldNumber ?? "" }}</td>
                      <td class="diff-gutter">{{ line.newNumber ?? "" }}</td>
                      <td class="diff-code">
                        <span class="diff-sign" aria-hidden="true">{{
                          line.type === "add"
                            ? "+"
                            : line.type === "del"
                              ? "-"
                              : " "
                        }}</span>{{ line.content }}
                      </td>
                    </tr>
                  </template>
                </tbody>
              </table>
            </div>
          </template>
          <template v-else-if="!showFilePreview && !showFileDiff && file">
            <p v-if="copyError" class="workspace-notice" role="alert">
              {{ copyError }}
            </p>
            <p
              v-if="file.content.length > SOURCE_LIMIT"
              class="workspace-notice"
            >
              {{ t("workspace.truncated") }}
            </p>
            <div class="source-scroll" tabindex="0" :aria-label="file.path">
              <pre class="source-gutter" aria-hidden="true">{{
                lineNumbers
              }}</pre>
              <pre class="source-code"><code v-html="highlighted" /></pre>
            </div>
          </template>
          <template v-else-if="showFilePreview && file">
            <p v-if="previewError" class="workspace-notice" role="alert">
              {{ previewError }}
            </p>
            <p v-if="previewWarnings.length" class="workspace-notice">
              {{
                t("workspace.previewWarning", {
                  files: previewWarnings.join(", "),
                })
              }}
            </p>
            <div v-if="previewLoading" class="preview-loading" role="status">
              {{ t("workspace.loading") }}
            </div>
            <div
              v-else-if="previewHtml"
              class="preview-stage"
              :class="{ mobile: mobilePreview }"
            >
              <iframe
                :key="previewVersion"
                :srcdoc="previewHtml"
                sandbox="allow-scripts"
                referrerpolicy="no-referrer"
                :title="`${t('workspace.preview')} · ${file.path}`"
              />
            </div>
            <div v-else class="workspace-empty">
              <t-icon name="file-code" size="32px" />
              <p>{{ t("workspace.previewEmpty") }}</p>
            </div>
          </template>
          <footer v-if="file" class="file-status">
            <span>{{ t(showFilePreview ? "workspace.previewNote" : "workspace.readOnly") }}</span>
            <span v-if="!showFilePreview">{{ t("workspace.lines", { count: lineCount }) }}</span>
          </footer>
        </div>
      </div>
    </template>
    <div v-else class="tests-pane">
      <div class="tests-heading">
        <span class="test-glyph"
          ><t-icon name="check-circle" size="24px"
        /></span>
        <h2>{{ t("workspace.testTitle") }}</h2>
        <p>{{ t("workspace.testDescription") }}</p>
      </div>
      <!-- <form class="test-form" @submit.prevent="runTests">
        <label for="workspace-test-command">{{ t('workspace.command') }}</label>
        <div class="command-input"><span aria-hidden="true">$</span><input id="workspace-test-command" v-model="command"
            :placeholder="t('workspace.commandPlaceholder')" :disabled="running" autocomplete="off"
            spellcheck="false" />
        </div>
        <div class="test-form-footer"><span>{{ t('workspace.commandHint') }}</span><button type="submit"
            class="run-button" :disabled="running || !command.trim()"><t-icon :name="running ? 'loading' : 'play'"
              :class="{ spinning: running }" />{{ t(running ? 'workspace.running' : 'workspace.run') }}</button></div>
      </form> -->
      <p v-if="runError" class="workspace-notice" role="alert">
        {{ runError }}
      </p>
      <div v-if="running" class="run-pending" role="status">
        <span class="pulse-dot"></span>{{ t("workspace.running") }}
      </div>
      <article v-for="run in runs" :key="run.id" class="test-result">
        <header>
          <span
            class="result-status"
            :class="
              run.result.killed || run.result.exit_code !== 0
                ? 'failed'
                : 'passed'
            "
            ><t-icon
              :name="
                run.result.killed || run.result.exit_code !== 0
                  ? 'close-circle'
                  : 'check-circle'
              "
            />{{
              t(
                run.result.killed
                  ? "workspace.timedOut"
                  : run.result.exit_code === 0
                    ? "workspace.passed"
                    : "workspace.failed",
              )
            }}</span
          ><span>{{
            t("workspace.duration", {
              seconds: (run.result.duration_ms / 1000).toFixed(2),
            })
          }}</span>
        </header>
        <code class="run-command">$ {{ run.command }}</code>
        <pre>{{
          [run.result.stdout, run.result.stderr].filter(Boolean).join("\n") ||
          t("workspace.noOutput")
        }}</pre>
        <footer>
          <span>{{
            t("workspace.exitCode", { code: run.result.exit_code })
          }}</span
          ><button
            v-if="run.result.exit_code !== 0 || run.result.killed"
            type="button"
            @click="askToFix(run)"
          >
            {{ t("workspace.fix") }}<t-icon name="arrow-up" />
          </button>
        </footer>
      </article>
      <template v-if="agentRuns.length">
        <p class="run-group-label">{{ t("workspace.agentRuns") }}</p>
        <article v-for="run in agentRuns" :key="run.id" class="test-result">
          <header>
            <span class="result-status" :class="run.status">{{
              t(
                run.status === "running"
                  ? "workspace.running"
                  : run.status === "failed"
                    ? "workspace.failed"
                    : run.status === "passed"
                      ? "workspace.passed"
                      : "workspace.unknown",
              )
            }}</span
            ><span v-if="run.durationMs != null">{{
              t("workspace.duration", {
                seconds: (run.durationMs / 1000).toFixed(2),
              })
            }}</span>
          </header>
          <code class="run-command">$ {{ run.command }}</code>
          <pre>{{
            run.output ||
            (run.status === "running" ? "…" : t("workspace.noOutput"))
          }}</pre>
        </article>
      </template>
      <div
        v-if="!runs.length && !agentRuns.length && !running"
        class="tests-idle"
      >
        <t-icon name="terminal" />{{ t("workspace.testingIdle") }}
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import hljs from "highlight.js/lib/common";
import { marked } from "marked";
import { sanitizeMarkdownHTML } from "@/utils/security";
import {
  getProgrammingTree,
  getProgrammingFile,
  runProgrammingCommand,
  type ProgrammingNode,
  type ProgrammingFile,
  type ProgrammingCommandResult,
} from "@/api/programming";
import {
  buildWorkspacePreview,
  canPreviewSource,
} from "@/utils/workspacePreview";
import { buildUnifiedDiff } from "@/utils/unifiedDiff";
import { findFileChange, useFileChanges } from "@/composables/useFileChanges";
import type {
  WorkspaceTab,
  WorkspaceCommandRun,
} from "@/utils/workspaceEvents";

export interface WorkspaceGateway {
  tree: (
    session: string,
    path: string,
  ) => Promise<{
    nodes: ProgrammingNode[];
    root: string;
    origin?: "host" | "sandbox";
    workspace_state?: "live" | "expired";
  }>;
  file: (session: string, path: string) => Promise<ProgrammingFile>;
  run: (session: string, command: string) => Promise<ProgrammingCommandResult>;
}
const props = withDefaults(
  defineProps<{
    sessionId: string;
    tab: WorkspaceTab;
    active: boolean;
    revision?: number;
    focusPath?: string;
    agentRuns?: WorkspaceCommandRun[];
    gateway?: WorkspaceGateway;
  }>(),
  { revision: 0, focusPath: "", agentRuns: () => [] },
);
const emit = defineEmits<{ ask: [prompt: string] }>();
const { t } = useI18n();
const api: WorkspaceGateway = props.gateway || {
  tree: async (session, path) => (await getProgrammingTree(session, path)).data,
  file: async (session, path) => (await getProgrammingFile(session, path)).data,
  run: async (session, command) =>
    (await runProgrammingCommand(session, { command, timeout: 60 })).data,
};
const SOURCE_LIMIT = 200_000;
// Remote sandboxes stage attachments in /workspace/input and artifacts in
// /workspace/output next to the project. The code tree follows Manus and only
// shows the project itself, so both are hidden there. A host workspace has no
// such staging tree, so folders with those names still belong to the project.
const HIDDEN_SANDBOX_ROOT_ENTRIES = new Set(["input", "output"]);
// Manus-style markers: TS/JS text for scripts, braces for data and config files,
// the git glyph for dotfiles that configure the repository, a page for the rest.
const TEXT_GLYPHS: Record<string, string> = {
  ts: "TS",
  tsx: "TS",
  mts: "TS",
  cts: "TS",
  js: "JS",
  jsx: "JS",
  mjs: "JS",
  cjs: "JS",
};
const BRACE_EXTENSIONS = new Set([
  "json",
  "jsonc",
  "json5",
  "yaml",
  "yml",
  "toml",
  "lock",
]);
const GIT_FILES = /^\.git(?:ignore|attributes|modules|config)$/;
type FileGlyph = { text?: string; icon?: string };
type TreeRow = {
  node: ProgrammingNode;
  depth: number;
  expanded: boolean;
  loading: boolean;
  glyph: FileGlyph;
};
function fileGlyph(name: string): FileGlyph {
  const lower = name.toLowerCase();
  if (GIT_FILES.test(lower)) return { icon: "git-branch" };
  const extension = /\.([a-z0-9]+)$/.exec(lower)?.[1] || "";
  if (TEXT_GLYPHS[extension]) return { text: TEXT_GLYPHS[extension] };
  if (BRACE_EXTENSIONS.has(extension)) return { text: "{}" };
  if (/^\.(?!gitkeep$)[a-z0-9]+$/.test(lower)) return { text: "{}" };
  return { icon: "file" };
}
const nodes = ref<ProgrammingNode[]>([]),
  file = ref<ProgrammingFile | null>(null);
const directory = ref(""),
  root = ref(""),
  error = ref(""),
  workspaceOrigin = ref<"host" | "sandbox">("sandbox");
// A reclaimed sandbox comes back as an empty workspace; the backend marks it
// so the panel explains the state instead of showing "no files yet".
const workspaceState = ref<"live" | "expired">("live");
const childrenByPath = ref<Record<string, ProgrammingNode[]>>({}),
  expandedPaths = ref<string[]>([]),
  pendingPaths = ref<string[]>([]);
const copied = ref(false),
  copyError = ref(""),
  mobilePreview = ref(false),
  previewHtml = ref(""),
  previewWarnings = ref<string[]>([]),
  previewError = ref(""),
  previewLoading = ref(false),
  previewVersion = ref(0);
type FileView = 'source' | 'diff' | 'preview';
const fileView = ref<FileView>('source');
const { changes: fileChanges, activeId: activeFileChange } = useFileChanges();
const canPreviewFile = computed(
  () => Boolean(file.value && canPreviewSource(file.value.path)),
);
/** The write/edit recorded for the file on screen, if any. */
const currentChange = computed(() => {
  const path = file.value?.path;
  return path ? findFileChange(fileChanges.value, path) : undefined;
});
/** The unified diff for the file on screen; null when there is nothing to show. */
const currentDiff = computed(() => {
  const change = currentChange.value;
  if (!change || change.before === undefined || change.after === undefined)
    return null;
  if (change.before === change.after) return null;
  const diff = buildUnifiedDiff(change.before, change.after);
  return diff.empty ? null : diff;
});
const hasFileDiff = computed(() => currentDiff.value !== null);
const diffHunks = computed(() => currentDiff.value?.hunks ?? []);
// The color difference is part of the code pane itself, Codex-style: the moment
// a file has a recorded change we render its diff inline, with no separate
// "diff" mode to switch into. Only the explicit preview view replaces it.
const showFileDiff = computed(
  () =>
    props.tab === 'source' &&
    !(fileView.value === 'preview' && canPreviewFile.value) &&
    hasFileDiff.value,
);
// The switch only needs to offer the standalone preview; the colored diff is
// always drawn in the code pane when one exists.
const showViewSwitch = computed(
  () => props.tab === 'source' && canPreviewFile.value,
);
const showFilePreview = computed(
  () =>
    props.tab === 'preview' ||
    (fileView.value === 'preview' && canPreviewFile.value),
);
const command = ref(""),
  running = ref(false),
  runError = ref("");
const runs = ref<
  Array<{ id: number; command: string; result: ProgrammingCommandResult }>
>([]);
let generation = 0,
  fileRequest = 0,
  previewRequest = 0,
  refreshRequest = 0,
  runId = 0;
const treeRequests = new Map<string, number>();
let refreshTimer: ReturnType<typeof setTimeout> | undefined;
const busy = computed(() => pendingPaths.value.length > 0);
const content = computed(
  () => file.value?.content.slice(0, SOURCE_LIMIT) || "",
);
const language = computed(
  () =>
    ({
      js: "javascript",
      ts: "typescript",
      py: "python",
      md: "markdown",
      sh: "bash",
      html: "html",
      svg: "xml",
      yml: "yaml",
    })[file.value?.path.split(".").pop() || ""] ||
    file.value?.path.split(".").pop() ||
    "text",
);
const lineCount = computed(() => content.value.split("\n").length);
const lineNumbers = computed(() =>
  Array.from({ length: lineCount.value }, (_, i) => i + 1).join("\n"),
);
const fileParts = computed(() => {
  const path =
    relativePath(file.value?.path || "") ||
    (file.value?.path || "").replaceAll("\\", "/");
  return path.split("/").filter(Boolean);
});
const highlighted = computed(() => {
  if (content.value.length < 60_000 && hljs.getLanguage(language.value))
    return hljs.highlight(content.value, { language: language.value }).value;
  return content.value
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;");
});

function visibleChildren(path: string): ProgrammingNode[] {
  const list = childrenByPath.value[path] || [];
  if (path || workspaceOrigin.value === "host") return list;
  return list.filter(
    (node) => !HIDDEN_SANDBOX_ROOT_ENTRIES.has(node.name.toLowerCase()),
  );
}
const treeRows = computed<TreeRow[]>(() => {
  const rows: TreeRow[] = [];
  const walk = (path: string, depth: number) => {
    for (const node of visibleChildren(path)) {
      const expanded =
        node.kind === "directory" && expandedPaths.value.includes(node.path);
      rows.push({
        node,
        depth,
        expanded,
        loading: expanded && pendingPaths.value.includes(node.path),
        glyph: node.kind === "file" ? fileGlyph(node.name) : {},
      });
      if (expanded) walk(node.path, depth + 1);
    }
  };
  walk("", 0);
  return rows;
});
function defaultFile(): string {
  const roots = visibleChildren("");
  const isIndex = (path: string) => /(?:^|\/)index\.html?$/i.test(path);
  return (
    roots.find((node) => node.kind === "file" && isIndex(node.path))?.path ||
    roots.find((node) => node.kind === "file")?.path ||
    treeRows.value.find((row) => row.node.kind === "file")?.node.path ||
    ""
  );
}

async function loadDirectory(path = "") {
  const request = (treeRequests.get(path) || 0) + 1,
    epoch = generation;
  treeRequests.set(path, request);
  pendingPaths.value = [
    ...pendingPaths.value.filter((item) => item !== path),
    path,
  ];
  error.value = "";
  try {
    const data = await api.tree(props.sessionId, path);
    if (epoch !== generation || treeRequests.get(path) !== request) return;
    childrenByPath.value = { ...childrenByPath.value, [path]: data.nodes };
    if (data.origin) workspaceOrigin.value = data.origin;
    workspaceState.value =
      data.workspace_state === "expired" ? "expired" : "live";
    nodes.value = visibleChildren(path);
    root.value = data.root;
    directory.value = path;
    if (workspaceState.value === "expired")
      error.value = t("workspace.workspaceExpired");
  } catch {
    if (epoch === generation && treeRequests.get(path) === request)
      error.value = t("workspace.unavailable");
  } finally {
    if (epoch === generation && treeRequests.get(path) === request) {
      pendingPaths.value = pendingPaths.value.filter((item) => item !== path);
    }
  }
}
async function toggleDirectory(node: ProgrammingNode) {
  if (node.kind !== "directory") return;
  if (expandedPaths.value.includes(node.path)) {
    expandedPaths.value = expandedPaths.value.filter(
      (path) => path !== node.path && !path.startsWith(`${node.path}/`),
    );
    return;
  }
  expandedPaths.value = [...expandedPaths.value, node.path];
  if (!(node.path in childrenByPath.value)) await loadDirectory(node.path);
}
function activateNode(node: ProgrammingNode) {
  if (node.kind === "directory") void toggleDirectory(node);
  else void openFile(node.path);
}
/** Expand every ancestor of a file so the tree reveals it after a tool event. */
async function revealPath(target: string) {
  const parts = target.split("/").filter(Boolean),
    epoch = generation;
  if (parts.length < 2) return;
  let current = "";
  for (const part of parts.slice(0, -1)) {
    current = current ? `${current}/${part}` : part;
    if (!(current in childrenByPath.value)) await loadDirectory(current);
    if (epoch !== generation) return;
    if (!expandedPaths.value.includes(current))
      expandedPaths.value = [...expandedPaths.value, current];
  }
}
function relativePath(path: string) {
  const normalized = path.replaceAll("\\", "/");
  const prefix = root.value.replace(/\/$/, "") + "/";
  return normalized.startsWith(prefix)
    ? normalized.slice(prefix.length)
    : normalized.startsWith("/")
      ? ""
      : normalized;
}
async function openFile(path: string, view?: FileView) {
  const request = ++fileRequest,
    epoch = generation;
  error.value = "";
  copied.value = false;
  copyError.value = "";
  previewHtml.value = "";
  previewRequest++;
  try {
    const data = await api.file(props.sessionId, path);
    if (epoch !== generation || request !== fileRequest) return;
    file.value = data;
    if (view) fileView.value = view;
    saveOpenFile(path);
    if (showFilePreview.value) void preparePreview();
  } catch (err) {
    if (epoch !== generation || request !== fileRequest) return;
    // A reclaimed workspace loses every path at once. Say that, instead of
    // blaming one file and sending the reader looking for a path bug.
    if (workspaceExpiredFailure(err)) {
      workspaceState.value = "expired";
      error.value = t("workspace.workspaceExpired");
      void reconcileMissingFile(path);
      return;
    }
    error.value = t("workspace.fileError");
    void reconcileMissingFile(path);
  }
}

function openFileStorageKey(sessionId: string) {
  return `weknora.workspace.openFile.${sessionId}`;
}

// Remember the file the user last had open so a reload restores it instead of
// falling back to the agent's last signal or the first file in the tree.
function savedOpenFile(): string {
  try {
    return localStorage.getItem(openFileStorageKey(props.sessionId)) || "";
  } catch {
    return "";
  }
}

function saveOpenFile(path: string) {
  try {
    localStorage.setItem(openFileStorageKey(props.sessionId), path);
  } catch {
    // localStorage may be unavailable in private mode; the view still works.
  }
}

/** 404 bodies from the programming API mark a reaped sandbox with a reason. */
function workspaceExpiredFailure(err: unknown): boolean {
  const details = (err as { error?: { details?: { reason?: string } } } | null)
    ?.error?.details;
  return details?.reason === "workspace_expired";
}
// A reclaimed sandbox comes back with an empty /workspace, so the tree can
// still offer rows the backend no longer reads. Re-listing the parent replaces
// the stale level; when the row really is gone it is dropped instead of
// staying clickable forever, and the notice says why rather than blaming the
// read.
async function reconcileMissingFile(path: string) {
  const parent = path.includes("/") ? path.slice(0, path.lastIndexOf("/")) : "";
  const epoch = generation,
    request = fileRequest,
    readError = error.value;
  await loadDirectory(parent);
  if (epoch !== generation || request !== fileRequest) return;
  if ((childrenByPath.value[parent] || []).some((node) => node.path === path)) {
    error.value = readError || t("workspace.fileError");
    return;
  }
  childrenByPath.value = Object.fromEntries(
    Object.entries(childrenByPath.value).filter(
      ([key]) => key !== path && !key.startsWith(`${path}/`),
    ),
  );
  expandedPaths.value = expandedPaths.value.filter(
    (item) => item !== path && !item.startsWith(`${path}/`),
  );
  // The whole workspace is gone, so the file was not removed from under the
  // reader: keep the last content they opened visible and let the notice
  // explain why it can no longer be refreshed.
  if (workspaceState.value === "expired") {
    error.value = t("workspace.workspaceExpired");
    return;
  }
  if (file.value?.path === path) file.value = null;
  error.value = t("workspace.fileGone");
}
// Set by a live edit so the very next refresh opens that file in the diff view.
// A panel that mounts *after* the change missed the activeFileChange watcher
// firing, so the refresh is what lands the reader on the diff.
let pendingDiffPath = "";
async function refresh() {
  await refreshInternal(false);
}

/** Reload the tree, then reopen either the focus target or the current file. */
async function refreshInternal(preferFocus: boolean) {
  const epoch = generation,
    request = ++refreshRequest;
  const loaded = Object.keys(childrenByPath.value);
  await Promise.all(
    (loaded.length ? loaded : [""]).map((path) => loadDirectory(path)),
  );
  if (epoch !== generation || request !== refreshRequest || error.value) return;
  const focus = relativePath(props.focusPath);
  const restored = savedOpenFile();
  // A live-edit path may still carry the workspace root (the tree is not
  // loaded when the agent reports it); strip it now that root is known.
  const rawLiveFocus = pendingDiffPath;
  pendingDiffPath = "";
  const liveFocus = rawLiveFocus
    ? relativePath(rawLiveFocus) || (rawLiveFocus.startsWith("/") ? "" : rawLiveFocus)
    : "";
  // A manual refresh keeps the file the user opened; an agent-driven refresh
  // follows focusPath (or a live edit) so the view jumps to whichever file is
  // changing, even when the panel only mounted after the change was recorded.
  const follow = preferFocus && focus ? focus : liveFocus;
  const candidate =
    follow || file.value?.path || restored || focus || defaultFile();
  if (!candidate) return;
  await revealPath(candidate);
  if (epoch !== generation || request !== refreshRequest) return;
  // An agent-driven refresh lands on the changed file and shows its diff; a
  // manual refresh keeps whatever view the reader was on.
  await openFile(candidate, follow ? "diff" : undefined);
}

// Live edit: follow the file the agent just wrote. The store keeps one entry
// per path, so A → B → A lands on the right file each time instead of staying
// on whatever was open when the turn started.
let lastFollowedChange = "";
watch(activeFileChange, async (id) => {
  if (!id || id === lastFollowedChange) return;
  lastFollowedChange = id;
  const change = fileChanges.value.find((item) => item.id === id);
  if (!change?.live) return;
  // Stash the raw path before awaiting: a fresh mount runs the refresh watcher
  // in the same tick, and it must find this file rather than a restored one.
  pendingDiffPath = change.path;
  if (!root.value) await loadDirectory("");
  const target = relativePath(change.path) || change.path;
  // An absolute path that does not sit under the tree root is not addressable
  // by the file API; the agent-driven refresh above is the fallback for it.
  if (!target || target.startsWith("/")) return;
  await revealPath(target);
  await openFile(target, "diff");
}, { immediate: true });

async function preparePreview() {
  const current = file.value,
    request = ++previewRequest,
    epoch = generation,
    session = props.sessionId;
  previewHtml.value = "";
  previewWarnings.value = [];
  previewError.value = "";
  previewLoading.value = false;
  if (!current || !canPreviewSource(current.path)) return;
  previewLoading.value = true;
  try {
    let source = current.content;
    if (/\.(?:md|markdown)$/i.test(current.path)) {
      source = `<html><head><style>body{font:15px/1.8 system-ui;padding:24px;color:var(--td-text-color-primary,#292824);max-width:760px;margin:auto;background:var(--td-bg-color-container,#fff)}pre{white-space:pre-wrap;background:var(--td-bg-color-secondarycontainer,#f5f5f3);padding:16px}img{max-width:100%}@media(prefers-color-scheme:dark){body{color:#e5e6dc;background:#252620}pre{background:#2c2d26}}</style></head><body>${sanitizeMarkdownHTML(marked.parse(source, { async: false }) as string)}</body></html>`;
    }
    const result = await buildWorkspacePreview(
      current.path,
      source,
      async (path) => (await api.file(session, path)).content,
    );
    if (epoch !== generation || request !== previewRequest) return;
    previewHtml.value = result.html;
    previewWarnings.value = result.warnings;
    previewVersion.value++;
  } catch {
    if (epoch === generation && request === previewRequest)
      previewError.value = t("workspace.previewError");
  } finally {
    if (epoch === generation && request === previewRequest)
      previewLoading.value = false;
  }
}
async function copySource() {
  try {
    await navigator.clipboard.writeText(file.value?.content || "");
    copied.value = true;
    copyError.value = "";
  } catch {
    copyError.value = t("workspace.copyFailed");
  }
}
function askToFix(run: { command: string; result: ProgrammingCommandResult }) {
  const output = [run.result.stdout, run.result.stderr]
    .filter(Boolean)
    .join("\n")
    .slice(-12_000);
  emit(
    "ask",
    `${t("workspace.fixPrompt", { command: run.command })}\n\n${t("workspace.exitCode", { code: run.result.exit_code })}\n\n${output}`,
  );
}
async function runTests() {
  const value = command.value.trim(),
    epoch = generation;
  if (!value || running.value) return;
  running.value = true;
  runError.value = "";
  try {
    const result = await api.run(props.sessionId, value);
    if (epoch !== generation) return;
    runs.value = [{ id: ++runId, command: value, result }, ...runs.value].slice(
      0,
      10,
    );
  } catch {
    if (epoch === generation) runError.value = t("workspace.runError");
  } finally {
    if (epoch === generation) running.value = false;
  }
}
watch(
  () => props.sessionId,
  () => {
    generation++;
    fileRequest++;
    previewRequest++;
    treeRequests.clear();
    nodes.value = [];
    file.value = null;
    directory.value = "";
    root.value = "";
    error.value = "";
    childrenByPath.value = {};
    expandedPaths.value = [];
    pendingPaths.value = [];
    workspaceOrigin.value = "sandbox";
    workspaceState.value = "live";
    fileView.value = "source";
    previewHtml.value = "";
    previewWarnings.value = [];
    previewError.value = "";
    previewLoading.value = false;
    runs.value = [];
    running.value = false;
    command.value = "";
    runError.value = "";
  },
);
let lastAutoRefresh = { sessionId: "", revision: -1 };
watch(
  () => [props.sessionId, props.active, props.revision] as const,
  ([sessionId, active, revision]) => {
    clearTimeout(refreshTimer);
    if (!active || !sessionId) return;
    // Follow the freshly-written file only when the agent changed something in
    // this same session; a fresh mount, a session switch, or a plain tab switch
    // restores/keeps the file the user last had open.
    const isNewSession = sessionId !== lastAutoRefresh.sessionId;
    const revisionChanged = revision !== lastAutoRefresh.revision;
    lastAutoRefresh = { sessionId, revision };
    const followFocus = revisionChanged && !isNewSession;
    refreshTimer = setTimeout(() => {
      void refreshInternal(followFocus);
    }, 180);
  },
  { immediate: true },
);
watch(
  () => showFilePreview.value,
  (preview) => {
    if (preview) void preparePreview();
    else { previewRequest++; previewLoading.value = false; }
  },
);
onBeforeUnmount(() => {
  generation++;
  clearTimeout(refreshTimer);
  treeRequests.clear();
});
</script>

<style scoped lang="less">
.conversation-workspace {
  flex: 1;
  min-height: 0;
  min-width: 0;
  display: flex;
  flex-direction: column;
  color: var(--td-text-color-primary);
  font-size: var(--app-text-sm);
  --ws-code-keyword: #d73a49;
  --ws-code-entity: #348080;
  --ws-code-string: #0f2f63;
  --ws-code-accent: #245cc5;
  --ws-code-comment: #6a737d;
}

button {
  font: inherit;
  color: inherit;
  cursor: pointer;
  border: 0;
  background: transparent;
  border-radius: var(--app-radius-sm);
  transition: background var(--app-motion-base) ease;

  &:hover {
    background: var(--td-bg-color-container-hover);
  }

  &:focus-visible {
    outline: 2px solid var(--td-text-color-secondary);
    outline-offset: -2px;
  }

  &:disabled {
    opacity: 0.45;
    cursor: not-allowed;
  }
}

.icon-button {
  width: 28px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  border: 1px solid transparent;

  &.active,
  &.is-active {
    background: var(--td-bg-color-secondarycontainer);
  }
}

.workspace-body {
  flex: 1;
  min-height: 0;
  display: flex;
}

.file-tree {
  width: clamp(132px, 28%, 220px);
  flex-shrink: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  border-right: 1px solid var(--td-component-stroke);
}

.file-tree__scroll {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 8px 6px 16px;
}

.file-tree__hint {
  padding: 10px 8px;
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-2xs);
  line-height: 1.8;
}

.file-tree__list {
  list-style: none;
  margin: 0;
  padding: 0;
}

.file-tree__row {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  min-height: 28px;
  padding: 4px 8px;
  text-align: left;
  color: var(--td-text-color-primary);
  font-size: var(--app-text-md);
  border-radius: var(--app-radius-sm);

  &.is-dir {
    font-weight: 500;
  }

  &.is-selected {
    background: var(--td-bg-color-secondarycontainer-active);
  }
}

.file-tree__lead {
  width: 14px;
  height: 14px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  color: var(--td-text-color-secondary);
}

.file-tree__glyph {
  font-family: var(--app-font-family-mono);
  font-size: var(--app-text-2xs);
  font-weight: 600;
  line-height: 1;
  color: var(--td-text-color-secondary);
}

.file-tree__name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.workspace-main {
  flex: 1;
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.workspace-notice {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 12px 16px;
  margin: 0;
  color: var(--td-text-color-secondary);
  background: var(--td-bg-color-secondarycontainer);
  font-size: var(--app-text-xs);
  line-height: 1.8;
  overflow-wrap: anywhere;
  flex-shrink: 0;

  .t-icon {
    margin-top: 4px;
    flex-shrink: 0;
  }

  button {
    margin-left: auto;
    white-space: nowrap;
    text-decoration: underline;
  }
}

.workspace-empty {
  flex: 1;
  min-height: 260px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  padding: 32px;
  text-align: center;

  h2 {
    font-size: var(--app-text-2xl);
    font-weight: 500;
    margin: 24px 0 8px;
  }

  p {
    max-width: 290px;
    font-size: var(--app-text-sm);
    line-height: 1.9;
    color: var(--td-text-color-secondary);
  }
}

.empty-art {
  width: 80px;
  height: 68px;
  border: 1px solid var(--td-component-border);
  border-radius: var(--app-radius-lg);
  transform: rotate(-6deg);
  display: grid;
  place-items: center;
  position: relative;
  background: var(--td-bg-color-container);

  &:before {
    content: "";
    position: absolute;
    inset: -9px 8px 8px -8px;
    border: 1px solid var(--td-component-stroke);
    border-radius: var(--app-radius-lg);
    z-index: -1;
    transform: rotate(12deg);
  }

  span {
    font: 24px monospace;
    color: var(--td-text-color-secondary);
  }
}

.empty-steps {
  display: flex;
  gap: 12px;
  align-items: center;
  margin-top: 28px;
  font-size: var(--app-text-2xs);
  color: var(--td-text-color-placeholder);

  span {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  b {
    font-weight: 400;
    font-size: var(--app-text-xs);
  }

  i {
    width: 18px;
    border-top: 1px solid var(--td-component-border);
  }
}

.file-status { display: flex; justify-content: space-between; gap: 8px; padding: 6px 10px; border-top: 1px solid var(--td-component-stroke); font-size: 11px; color: var(--td-text-color-placeholder); }
.file-toolbar { display: flex; align-items: center; flex-wrap: wrap; gap: 6px; padding: 8px 10px; border-bottom: 1px solid var(--td-component-stroke); }
.file-toolbar__path { flex: 1; min-width: 60px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-family: var(--app-font-family-mono, monospace); font-size: 12px; }
.file-toolbar button { display: inline-flex; align-items: center; justify-content: center; border: 0; border-radius: 5px; padding: 6px; background: transparent; color: var(--td-text-color-secondary); cursor: pointer; font: inherit; }
.file-toolbar button:hover, .file-toolbar button[aria-pressed="true"] { background: var(--td-bg-color-secondarycontainer); color: var(--td-text-color-primary); }
.file-toolbar button:focus-visible { outline: 2px solid var(--td-brand-color); outline-offset: 2px; }
.file-view-switch { display: flex; gap: 2px; padding: 2px; border: 1px solid var(--td-component-stroke); border-radius: 7px; }
.diff-scroll {
  overflow: auto;
  flex: 1;
  min-height: 0;
  background: var(--td-bg-color-secondarycontainer);
}
.diff-table {
  width: 100%;
  border-collapse: collapse;
  font: var(--app-text-md)/1.5 var(--app-font-family-mono, monospace);
  tab-size: 4;
}
.diff-hunk-row td { position: sticky; top: 0; z-index: 1; }
.diff-hunk {
  padding: 4px 14px;
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-xs, 12px);
  background: color-mix(in srgb, var(--td-text-color-primary) 4%, var(--td-bg-color-secondarycontainer));
  user-select: none;
}
.diff-row {
  &.is-add { background: color-mix(in srgb, #2da44e 22%, transparent); }
  &.is-del { background: color-mix(in srgb, #cf222e 22%, transparent); }
}
.diff-gutter {
  width: 1%;
  min-width: 40px;
  padding: 0 8px;
  text-align: right;
  vertical-align: top;
  white-space: nowrap;
  user-select: none;
  color: var(--td-text-color-disabled);
  border-right: 1px solid var(--td-component-stroke);
}
.diff-row.is-add .diff-gutter:first-child { box-shadow: inset 2px 0 0 #2da44e; }
.diff-row.is-del .diff-gutter:first-child { box-shadow: inset 2px 0 0 #cf222e; }
.diff-code {
  padding: 0 16px 0 10px;
  white-space: pre;
  vertical-align: top;
  word-break: normal;
  overflow-wrap: normal;
}
.diff-sign {
  display: inline-block;
  width: 1.2em;
  color: var(--td-text-color-disabled);
  user-select: none;
}
.diff-row.is-add .diff-sign { color: #2da44e; }
.diff-row.is-del .diff-sign { color: #cf222e; }
.source-scroll {
  overflow: auto;
  display: flex;
  align-items: flex-start;
  flex: 1;
  min-height: 0;
  background: var(--td-bg-color-secondarycontainer);
  padding: 14px 0 24px;

  pre {
    margin: 0;
    font: var(--app-text-md)/1.4 var(--app-font-family-mono, monospace);
    tab-size: 2;
  }
}

.source-gutter {
  color: var(--td-text-color-disabled);
  text-align: right;
  padding: 0 14px 0 16px;
  user-select: none;
  position: sticky;
  left: 0;
  background: var(--td-bg-color-secondarycontainer);
}

.source-code {
  padding: 0 24px 0 8px;

  :deep(.hljs-keyword),
  :deep(.hljs-doctag),
  :deep(.hljs-meta) {
    color: var(--ws-code-keyword);
  }

  :deep(.hljs-string),
  :deep(.hljs-regexp) {
    color: var(--ws-code-string);
  }

  :deep(.hljs-comment),
  :deep(.hljs-quote) {
    color: var(--ws-code-comment);
  }

  :deep(.hljs-number),
  :deep(.hljs-literal),
  :deep(.hljs-operator),
  :deep(.hljs-punctuation),
  :deep(.hljs-params) {
    color: var(--ws-code-accent);
  }

  :deep(.hljs-title),
  :deep(.hljs-name),
  :deep(.hljs-built_in),
  :deep(.hljs-type),
  :deep(.hljs-section),
  :deep(.hljs-selector-class),
  :deep(.hljs-variable) {
    color: var(--ws-code-entity);
  }

  :deep(.hljs-attr),
  :deep(.hljs-attribute),
  :deep(.hljs-property),
  :deep(.hljs-tag) {
    color: inherit;
  }
}

:global(:root[theme-mode="dark"]) .conversation-workspace {
  --ws-code-keyword: #ff7b72;
  --ws-code-entity: #56d4dd;
  --ws-code-string: #a5d6ff;
  --ws-code-accent: #79c0ff;
  --ws-code-comment: #8b949e;
}

:global(:root[theme-mode="dark"])
  .conversation-workspace
  .diff-row.is-add {
  background: color-mix(in srgb, #3fb950 20%, transparent);
}

:global(:root[theme-mode="dark"])
  .conversation-workspace
  .diff-row.is-del {
  background: color-mix(in srgb, #f85149 20%, transparent);
}

:global(:root[theme-mode="dark"])
  .conversation-workspace
  .diff-row.is-add
  .diff-gutter:first-child {
  box-shadow: inset 2px 0 0 #3fb950;
}

:global(:root[theme-mode="dark"])
  .conversation-workspace
  .diff-row.is-del
  .diff-gutter:first-child {
  box-shadow: inset 2px 0 0 #f85149;
}

:global(:root[theme-mode="dark"])
  .conversation-workspace
  .diff-row.is-add
  .diff-sign {
  color: #3fb950;
}

:global(:root[theme-mode="dark"])
  .conversation-workspace
  .diff-row.is-del
  .diff-sign {
  color: #f85149;
}

:global(:root[theme-mode="dark"])
  .conversation-workspace
  .file-tree__row.is-selected {
  background: var(--td-bg-color-container-active);
}

:global(:root[theme-mode="dark"])
  .conversation-workspace
  .preview-stage
  iframe {
  background: var(--td-bg-color-page);
}

:global(:root[theme-mode="dark"]) .conversation-workspace .test-result pre {
  background: var(--td-bg-color-page);
}

:global(:root[theme-mode="dark"]) .conversation-workspace .test-glyph {
  background: var(--td-bg-color-container-active);
}

:global(:root[theme-mode="dark"]) .conversation-workspace .empty-art {
  background: var(--td-bg-color-secondarycontainer);
}

:global(:root[theme-mode="dark"]) .conversation-workspace .workspace-notice {
  background: var(--td-bg-color-container-active);
  color: var(--td-text-color-secondary);
}

:global(:root[theme-mode="dark"]) .conversation-workspace .test-result {
  border-color: var(--td-component-stroke);
}

:global(:root[theme-mode="dark"]) .conversation-workspace .run-command {
  color: var(--td-text-color-primary);
}

:global(:root[theme-mode="dark"]) .conversation-workspace .file-tree__hint {
  color: var(--td-text-color-disabled);
}

.preview-stage {
  flex: 1;
  min-height: 0;
  padding: 0;
  display: flex;
  justify-content: center;
  background: var(--td-bg-color-secondarycontainer);

  iframe {
    flex: 1;
    min-width: 0;
    width: 100%;
    height: 100%;
    border: 0;
    background: var(--td-bg-color-container, #fff);
  }

  &.mobile {
    padding: 16px 12px;

    iframe {
      max-width: 375px;
      border-radius: var(--app-radius-2xl);
      border: 1px solid var(--td-component-border);
    }
  }
}

.preview-loading {
  padding: 24px;
  color: var(--td-text-color-placeholder);
}

.tests-pane {
  overflow: auto;
  flex: 1;
  padding: 24px 20px;
}

.tests-heading {
  margin-bottom: 26px;

  h2 {
    font-size: var(--app-text-3xl);
    font-weight: 500;
    margin: 16px 0 10px;
    letter-spacing: -0.4px;
  }

  p {
    color: var(--td-text-color-secondary);
    font-size: var(--app-text-sm);
    line-height: 1.9;
    max-width: 400px;
  }
}

.test-glyph {
  display: inline-flex;
  padding: 10px;
  background: var(--td-bg-color-secondarycontainer);
  border-radius: var(--app-radius-xl);
}

.test-form label {
  display: block;
  font-size: var(--app-text-xs);
  margin-bottom: 10px;
}

.command-input {
  display: flex;
  gap: 10px;
  border: 1px solid var(--td-component-border);
  border-radius: var(--app-radius-md);
  padding: 12px;
  align-items: center;

  &:focus-within {
    outline: 1px solid var(--td-text-color-secondary);
  }

  span {
    color: var(--td-text-color-placeholder);
  }

  input {
    flex: 1;
    min-width: 0;
    font: 12px var(--app-font-family-mono, monospace);
    background: transparent;
    color: inherit;
    border: 0;
    outline: none;

    &::placeholder {
      font-size: var(--app-text-2xs);
    }
  }
}

.test-form-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-top: 12px;

  > span {
    font-size: var(--app-text-2xs);
    color: var(--td-text-color-placeholder);
  }

  .run-button {
    display: flex;
    align-items: center;
    gap: 6px;
    background: var(--td-text-color-primary);
    color: var(--td-bg-color-container);
    padding: 9px 12px;
    border-radius: var(--app-radius-md);
    font-size: var(--app-text-xs);
  }
}

.test-result {
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-lg);
  margin-top: 20px;
  overflow: hidden;

  header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px;
    font-size: var(--app-text-2xs);
    color: var(--td-text-color-placeholder);
  }

  pre {
    margin: 0;
    padding: 14px;
    font: 11px/1.8 var(--app-font-family-mono, monospace);
    background: var(--td-bg-color-secondarycontainer);
    max-height: 320px;
    overflow: auto;
    white-space: pre-wrap;
    overflow-wrap: anywhere;
  }

  footer {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 10px 12px;
    color: var(--td-text-color-secondary);
    font-size: var(--app-text-2xs);

    button {
      display: flex;
      align-items: center;
      gap: 6px;
    }
  }
}

.result-status {
  display: flex;
  gap: 6px;
  align-items: center;

  &.passed {
    color: var(--td-success-color);
  }

  &.failed {
    color: var(--td-error-color);
  }
}

.run-command {
  display: block;
  padding: 0 12px 12px;
  overflow-wrap: anywhere;
  font-size: var(--app-text-xs);
}

.tests-idle {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 8px;
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-xs);
  padding: 70px 0;
}

.run-pending {
  padding: 24px 0;
  display: flex;
  gap: 8px;
  align-items: center;
  color: var(--td-text-color-secondary);
}

.pulse-dot {
  width: 6px;
  height: 6px;
  background: currentColor;
  border-radius: 50%;
  animation: pulse 1s infinite alternate;
}

.run-group-label {
  margin-top: 28px;
  font-size: var(--app-text-2xs);
  color: var(--td-text-color-placeholder);
}

.spinning {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@keyframes pulse {
  to {
    opacity: 0.3;
  }
}

@media (prefers-reduced-motion: reduce) {
  *,
  *::before {
    animation: none !important;
    transition: none !important;
  }
}
</style>
