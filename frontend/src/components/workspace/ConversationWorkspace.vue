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
        <aside
          v-show="!showFileDiff"
          ref="treeRef"
          class="file-tree"
          :style="treeStyle"
          :aria-label="t('workspace.files')"
        >
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
          <!-- 拖拽分隔条：像调整整个面板一样，把宽度让给代码/差异视图。 -->
          <PanelResizeHandle
            edge="right"
            :label="t('workspace.resizeFiles')"
            :value="treeWidth || measuredTreeWidth || TREE_MIN_WIDTH"
            :min="TREE_MIN_WIDTH"
            :max="TREE_MAX_WIDTH"
            @start="startTreeResize"
            @resize="resizeTree"
            @end="endTreeResize"
          />
        </aside>
        <div class="workspace-main">
          <div v-if="file || fileChanges.length" class="file-toolbar">
            <span class="file-toolbar__path" :title="file?.path">{{
              showFileDiff ? t("workspace.diff") : file?.path
            }}</span>
            <div
              v-if="showViewSwitch"
              class="file-view-switch"
              role="group"
              :aria-label="t('workspace.fileView')"
            >
              <button
                type="button"
                :aria-pressed="fileView === 'source'"
                @click="fileView = 'source'"
              >
                {{ t("workspace.source") }}
              </button>
              <button
                v-if="fileChanges.length"
                type="button"
                :aria-pressed="fileView === 'diff'"
                @click="fileView = 'diff'"
              >
                {{ t("workspace.diff") }}
              </button>
              <button
                v-if="canPreviewFile"
                type="button"
                :aria-pressed="fileView === 'preview'"
                @click="fileView = 'preview'"
              >
                {{ t("workspace.preview") }}
              </button>
            </div>
            <button
              v-if="showFilePreview"
              type="button"
              :aria-pressed="mobilePreview"
              :title="
                t(mobilePreview ? 'workspace.desktop' : 'workspace.mobile')
              "
              :aria-label="
                t(mobilePreview ? 'workspace.desktop' : 'workspace.mobile')
              "
              @click="mobilePreview = !mobilePreview"
            >
              <t-icon :name="mobilePreview ? 'desktop' : 'mobile'" />
            </button>
            <button
              v-else-if="file && !imageSource && !showFileDiff"
              type="button"
              :title="t(copied ? 'workspace.copied' : 'workspace.copy')"
              :aria-label="t(copied ? 'workspace.copied' : 'workspace.copy')"
              @click="copySource"
            >
              <t-icon :name="copied ? 'check' : 'copy'" />
            </button>
            <button
              type="button"
              :title="t('workspace.refresh')"
              :aria-label="t('workspace.refresh')"
              @click="refresh"
            >
              <t-icon name="refresh" />
            </button>
          </div>
          <div v-if="error" class="workspace-notice" role="alert">
            <t-icon name="info-circle" /><span>{{ error }}</span
            ><button type="button" @click="refresh">
              {{ t("workspace.retry") }}
            </button>
          </div>
          <div
            v-if="showFileDiff"
            class="diff-file-list"
            :aria-label="t('workspace.diff')"
          >
            <WorkspaceFileDiff
              v-for="change in fileChanges"
              :key="`${sessionId}:${change.path}`"
              :change="change"
              @open-source="
                openFile(relativePath(change.path) || change.path, 'source')
              "
            />
          </div>
          <div
            v-else-if="!file && workspaceState !== 'expired'"
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
          <template v-else-if="imageSource && file">
            <p v-if="imageError" class="workspace-notice" role="alert">
              {{ t("workspace.imageError") }}
            </p>
            <div
              v-else
              class="image-preview"
              tabindex="0"
              :aria-label="file.path"
            >
              <img
                :key="imageSource"
                :src="imageSource"
                :alt="file.path"
                @error="imageError = true"
              />
            </div>
          </template>
          <template v-else-if="!showFilePreview && !showFileDiff && file">
            <p v-if="copyError" class="workspace-notice" role="alert">
              {{ copyError }}
            </p>
            <div class="source-scroll" tabindex="0" :aria-label="file.path">
              <div class="source-lines">
                <div
                  v-for="(line, index) in highlightedLines"
                  :key="index"
                  class="source-row"
                >
                  <span class="source-gutter" aria-hidden="true">{{
                    index + 1
                  }}</span>
                  <pre
                    class="source-code"
                  ><code v-html="line || '&#8203;'" /></pre>
                </div>
              </div>
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
          <footer v-if="file && !showFileDiff" class="file-status">
            <span>{{
              t(
                imageSource
                  ? "workspace.imagePreview"
                  : showFilePreview
                    ? "workspace.previewNote"
                    : "workspace.readOnly",
              )
            }}</span>
            <span v-if="!showFilePreview && !imageSource">{{
              t("workspace.lines", { count: lineCount })
            }}</span>
          </footer>
        </div>
      </div>
    </template>
    <WorkspaceTestResults
      v-else
      :key="sessionId"
      :session-id="sessionId"
      :runs="testRuns"
      :running="running"
      :error="runError"
      @ask="emit('ask', $event)"
    />
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import PanelResizeHandle from "@/components/PanelResizeHandle.vue";
import WorkspaceFileDiff from "./WorkspaceFileDiff.vue";
import WorkspaceTestResults from "./WorkspaceTestResults.vue";
import {
  formatWorkspaceCode,
  highlightWorkspaceLines,
} from "@/utils/workspaceCode";
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
import { useFileChanges } from "@/composables/useFileChanges";
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
type FileView = "source" | "diff" | "preview";
const fileView = ref<FileView>("source");
const imageError = ref(false);
const imageSource = computed(() => {
  const current = file.value;
  if (
    current?.encoding !== "base64" ||
    !/^image\/(?:png|jpeg|gif|webp|bmp|x-icon|avif)$/.test(
      current.mime_type || "",
    )
  )
    return "";
  return `data:${current.mime_type};base64,${current.content}`;
});
watch(imageSource, () => {
  imageError.value = false;
});
const { changes: fileChanges, activeId: activeFileChange } = useFileChanges();
const canPreviewFile = computed(() =>
  Boolean(file.value && canPreviewSource(file.value.path)),
);
const formattedCode = ref<{ source: string } | null>(null);
watch(
  () =>
    [
      props.sessionId,
      file.value?.path,
      file.value?.encoding === "base64" ? undefined : file.value?.content,
    ] as const,
  async ([, path, source], _, onCleanup) => {
    let cancelled = false;
    onCleanup(() => {
      cancelled = true;
    });
    formattedCode.value = null;
    if (!path || source === undefined) return;
    const display = await formatWorkspaceCode(source, path);
    if (!cancelled) formattedCode.value = { source: display };
  },
  { immediate: true, flush: "sync" },
);
// Source always contains the complete file. Changes have their own file list.
const showFileDiff = computed(
  () =>
    props.tab === "source" &&
    fileView.value === "diff" &&
    fileChanges.value.length > 0,
);
const showViewSwitch = computed(
  () =>
    props.tab === "source" &&
    (canPreviewFile.value || fileChanges.value.length > 0),
);
const showFilePreview = computed(
  () =>
    !imageSource.value &&
    (props.tab === "preview" ||
      (fileView.value === "preview" && canPreviewFile.value)),
);
const command = ref(""),
  running = ref(false),
  runError = ref("");
const runs = ref<
  Array<{ id: number; command: string; result: ProgrammingCommandResult }>
>([]);
const testRuns = computed<WorkspaceCommandRun[]>(() => [
  ...runs.value.map((run) => ({
    id: `manual:${run.id}`,
    command: run.command,
    output: [run.result.stdout, run.result.stderr].filter(Boolean).join("\n"),
    status: (run.result.killed || run.result.exit_code !== 0
      ? "failed"
      : "passed") as WorkspaceCommandRun["status"],
    durationMs: run.result.duration_ms,
    exitCode: run.result.exit_code,
  })),
  ...props.agentRuns,
]);
let generation = 0,
  fileRequest = 0,
  previewRequest = 0,
  refreshRequest = 0,
  runId = 0;
const treeRequests = new Map<string, number>();
let refreshTimer: ReturnType<typeof setTimeout> | undefined;
const busy = computed(() => pendingPaths.value.length > 0);
const content = computed(() =>
  file.value?.encoding === "base64"
    ? ""
    : (formattedCode.value?.source ?? file.value?.content ?? ""),
);
const lineCount = computed(() => content.value.split("\n").length);
const highlightedLines = computed(() =>
  highlightWorkspaceLines(content.value, file.value?.path || ""),
);
const fileParts = computed(() => {
  const path =
    relativePath(file.value?.path || "") ||
    (file.value?.path || "").replaceAll("\\", "/");
  return path.split("/").filter(Boolean);
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
  else void openFile(node.path, "source");
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
    imageError.value = false;
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

// --- 文件树 / 代码区之间的可伸缩分隔条 --------------------------------
// The sandbox panel is already resizable; the code side of it is too. Dragging
// the divider trades width between the project tree and the code/diff view so
// a wide diff is readable without widening the whole panel.
const TREE_MIN_WIDTH = 132;
const TREE_MAX_WIDTH = 420;
// Never let the tree eat the code pane: the viewer keeps at least this much.
const TREE_CODE_MIN_WIDTH = 200;
const TREE_WIDTH_STORAGE_KEY = "weknora.workspace.treeWidth";
const treeRef = ref<HTMLElement | null>(null);
const measuredTreeWidth = ref(0);
// 0 keeps the responsive CSS default until the reader drags the divider.
const treeWidth = ref(storedTreeWidth());
let treeDragStart = 0;

function storedTreeWidth(): number {
  try {
    const raw = Number(localStorage.getItem(TREE_WIDTH_STORAGE_KEY));
    if (!Number.isFinite(raw) || raw <= 0) return 0;
    return Math.min(TREE_MAX_WIDTH, Math.max(TREE_MIN_WIDTH, Math.round(raw)));
  } catch {
    return 0;
  }
}

const treeStyle = computed(() =>
  treeWidth.value ? { width: `${treeWidth.value}px` } : undefined,
);

function clampTreeWidth(width: number): number {
  const body = treeRef.value?.parentElement;
  const cap = body
    ? Math.max(TREE_MIN_WIDTH, body.clientWidth - TREE_CODE_MIN_WIDTH)
    : TREE_MAX_WIDTH;
  return Math.min(
    TREE_MAX_WIDTH,
    cap,
    Math.max(TREE_MIN_WIDTH, Math.round(width)),
  );
}

function currentTreeWidth(): number {
  return (
    treeWidth.value ||
    measuredTreeWidth.value ||
    treeRef.value?.getBoundingClientRect().width ||
    TREE_MIN_WIDTH
  );
}

function startTreeResize() {
  treeDragStart = currentTreeWidth();
}

function resizeTree(delta: number) {
  treeWidth.value = clampTreeWidth(treeDragStart + delta);
}

function endTreeResize() {
  if (!treeWidth.value) return;
  try {
    localStorage.setItem(TREE_WIDTH_STORAGE_KEY, String(treeWidth.value));
  } catch {
    // localStorage may be unavailable in private mode; the drag still works.
  }
}

onMounted(() => {
  measuredTreeWidth.value = treeRef.value?.getBoundingClientRect().width || 0;
});

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
    ? relativePath(rawLiveFocus) ||
      (rawLiveFocus.startsWith("/") ? "" : rawLiveFocus)
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
watch(
  activeFileChange,
  async (id) => {
    if (!id || id === lastFollowedChange) return;
    lastFollowedChange = id;
    const change = fileChanges.value.find((item) => item.id === id);
    if (!change?.live) return;
    fileView.value = "diff";
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
  },
  { immediate: true },
);

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
      source = `<html><head><style>body{font:15px/1.8 system-ui;padding:24px;color:var(--td-text-color-primary,#292824);max-width:760px;margin:auto;background:var(--td-bg-color-container,#fff)}pre{white-space:pre-wrap;background:var(--td-bg-color-secondarycontainer,#f5f5f3);padding:16px}img{max-width:100%}::selection{background:#cfe1f7}@media(prefers-color-scheme:dark){body{color:#e5e5e5;background:#141414}pre{background:#2a2a2a}::selection{background:#365073;color:#e8e8e8}}</style></head><body>${sanitizeMarkdownHTML(marked.parse(source, { async: false }) as string)}</body></html>`;
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
    else {
      previewRequest++;
      previewLoading.value = false;
    }
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
  // The divider can widen the tree, but the code view always keeps room.
  max-width: calc(100% - 200px);
  flex-shrink: 0;
  position: relative;
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

.file-status {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  padding: 6px 10px;
  border-top: 1px solid var(--td-component-stroke);
  font-size: 11px;
  color: var(--td-text-color-placeholder);
}
.file-toolbar {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  padding: 8px 10px;
  border-bottom: 1px solid var(--td-component-stroke);
}
.file-toolbar__path {
  flex: 1;
  min-width: 60px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-family: var(--app-font-family-mono, monospace);
  font-size: 12px;
}
.file-toolbar button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 5px;
  padding: 6px;
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;
  font: inherit;
}
.file-toolbar button:hover,
.file-toolbar button[aria-pressed="true"] {
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-primary);
}
.file-toolbar button:focus-visible {
  outline: 2px solid var(--td-brand-color);
  outline-offset: 2px;
}
.file-view-switch {
  display: flex;
  gap: 2px;
  padding: 2px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 7px;
}
.diff-file-list {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 0 10px 16px;
}
.source-scroll {
  overflow: auto;
  flex: 1;
  min-height: 0;
  background: var(--td-bg-color-secondarycontainer);
  padding: 14px 0 24px;

  pre {
    margin: 0;
    font: var(--app-text-md) / 1.4 var(--app-font-family-mono, monospace);
    tab-size: 2;
  }
}

.source-lines {
  width: 100%;
  min-width: 0;
}
.source-row {
  display: grid;
  grid-template-columns: 4.5em minmax(0, 1fr);
}
.source-gutter {
  font: var(--app-text-md) / 1.4 var(--app-font-family-mono, monospace);
  color: var(--td-text-color-disabled);
  text-align: right;
  padding: 0 14px 0 16px;
  user-select: none;
  background: var(--td-bg-color-secondarycontainer);
}

.source-code {
  min-width: 0;
  max-height: none;
  overflow: visible;
  padding: 0 16px 0 8px;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  code {
    font: inherit;
    white-space: inherit;
  }
}

.source-code {
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
  :deep(.hljs-selector-tag),
  :deep(.hljs-selector-id),
  :deep(.hljs-selector-pseudo),
  :deep(.hljs-selector-class),
  :deep(.hljs-variable) {
    color: var(--ws-code-entity);
  }

  :deep(.hljs-attr),
  :deep(.hljs-attribute),
  :deep(.hljs-property),
  :deep(.hljs-tag) {
    color: var(--ws-code-accent);
  }
}

:global(:root[theme-mode="dark"] .conversation-workspace) {
  --ws-code-keyword: #ff7b72;
  --ws-code-entity: #56d4dd;
  --ws-code-string: #a5d6ff;
  --ws-code-accent: #79c0ff;
  --ws-code-comment: #8b949e;
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

:global(:root[theme-mode="dark"]) .conversation-workspace .empty-art {
  background: var(--td-bg-color-secondarycontainer);
}

:global(:root[theme-mode="dark"]) .conversation-workspace .workspace-notice {
  background: var(--td-bg-color-container-active);
  color: var(--td-text-color-secondary);
}

:global(:root[theme-mode="dark"]) .conversation-workspace .run-command {
  color: var(--td-text-color-primary);
}

:global(:root[theme-mode="dark"]) .conversation-workspace .file-tree__hint {
  color: var(--td-text-color-disabled);
}

.image-preview {
  flex: 1;
  min-height: 0;
  min-width: 0;
  overflow: auto;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  background: var(--td-bg-color-secondarycontainer);

  img {
    display: block;
    max-width: 100%;
    max-height: 100%;
    object-fit: contain;
  }
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
