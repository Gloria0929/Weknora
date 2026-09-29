<template>
  <section class="coding-workspace">
    <aside class="coding-nav">
      <div class="coding-brand">
        <span class="coding-brand__mark">&lt;/&gt;</span><span>Coding</span>
      </div>
      <div class="coding-mode-switch" aria-label="工作模式">
        <button type="button" @click="switchWorkMode('chat')">Chat</button
        ><button
          class="is-active"
          type="button"
          @click="switchWorkMode('coding')"
        >
          Coding
        </button>
      </div>
      <button class="primary-action" type="button" @click="openProjectDialog">
        <t-icon name="add" /> 新建项目
      </button>
      <div class="project-list" aria-label="Coding 项目">
        <button
          v-for="project in projects"
          :key="project.id"
          class="project-row"
          :class="{ 'is-active': project.id === selectedProjectId }"
          type="button"
          @click="selectProject(project.id)"
        >
          <t-icon name="folder" /><span>{{ project.name }}</span>
        </button>
      </div>
      <template v-if="selectedProject">
        <div class="section-heading">
          <span>Folders</span
          ><button
            type="button"
            title="添加本地文件夹"
            @click="openFolderDialog"
          >
            <t-icon name="add" size="14px" />
          </button>
        </div>
        <div class="folder-list">
          <button
            v-for="folder in folders"
            :key="folder.id"
            class="folder-row"
            :class="{ 'is-active': folder.id === selectedFolderId }"
            type="button"
            @click="selectFolder(folder.id)"
          >
            <t-icon name="folder-open" size="15px" /><span>{{
              folder.name
            }}</span>
          </button>
          <p v-if="!foldersLoading && folders.length === 0" class="empty-note">
            先为项目添加一个本地文件夹。
          </p>
        </div>
        <template v-if="selectedFolder">
          <div class="section-heading section-heading--threads">
            <span>对话</span
            ><button type="button" title="新建对话" @click="newThread">
              <t-icon name="add" size="14px" />
            </button>
          </div>
          <button
            v-for="thread in threads"
            :key="thread.id"
            class="thread-row"
            :class="{ 'is-active': thread.id === selectedThreadId }"
            type="button"
            @click="selectThread(thread.id)"
          >
            <t-icon name="chat" size="15px" /><span>{{
              thread.title || "未命名对话"
            }}</span>
          </button>
          <p v-if="!threadsLoading && threads.length === 0" class="empty-note">
            在该文件夹下创建第一段对话。
          </p>
        </template>
      </template>
    </aside>
    <main class="coding-main">
      <Chat
        v-if="selectedThreadId"
        :session_id="selectedThreadId"
        agent-id="builtin-smart-reasoning"
        :kb-ids="[]"
        :embedded-mode="true"
        :coding-mode="true"
      />
      <div v-else class="coding-empty">
        <div class="coding-empty__glyph">{ }</div>
        <h1>{{ emptyTitle }}</h1>
        <p>{{ emptyDescription }}</p>
        <button
          v-if="!selectedProject"
          class="primary-action compact"
          type="button"
          @click="openProjectDialog"
        >
          创建项目</button
        ><button
          v-else-if="!selectedFolder"
          class="primary-action compact"
          type="button"
          @click="openFolderDialog"
        >
          添加本地文件夹</button
        ><button
          v-else
          class="primary-action compact"
          type="button"
          @click="newThread"
        >
          新建对话
        </button>
      </div>
    </main>
    <aside class="coding-inspector">
      <template v-if="selectedProject">
        <div class="inspector-heading">Project</div>
        <h2>{{ selectedProject.name }}</h2>
        <p class="muted">项目只用于组织本地文件夹和对话，不关联知识库。</p>
        <div class="inspector-divider"></div>
        <template v-if="selectedFolder"
          ><div class="inspector-heading">Local folder</div>
          <h3>{{ selectedFolder.name }}</h3>
          <p class="project-path">{{ selectedFolder.local_path }}</p>
          <p class="muted">
            创建对话时会验证该目录是否已经在本机 Agent 中授权。
          </p></template
        >
        <template v-else
          ><div class="inspector-heading">Folders</div>
          <p class="muted">
            一个项目可添加多个本地文件夹；每个文件夹各自拥有对话。
          </p></template
        >
        <template v-if="selectedThreadId && selectedFolder"
          ><div class="inspector-divider"></div>
          <div class="review-heading">
            <span class="inspector-heading">Changes</span
            ><button
              type="button"
              :disabled="reviewLoading"
              @click="loadReview"
            >
              刷新 Git Diff
            </button>
          </div>
          <p v-if="reviewMessage" class="muted">{{ reviewMessage }}</p>
          <pre v-if="reviewDiff" class="review-diff">{{ reviewDiff }}</pre>
          <p v-else-if="!reviewLoading" class="muted">
            发送一次 Coding 任务后，可在此检查真实工作树变更。
          </p></template
        >
      </template>
      <template v-else
        ><div class="inspector-heading">Coding mode</div>
        <p class="muted">
          Coding 与知识库空间分开，使用现有 WeKnora Agent
          在已授权的本地目录中工作。
        </p></template
      >
    </aside>
    <t-dialog
      v-model:visible="projectDialogOpen"
      header="新建 Coding 项目"
      :confirm-btn="{ content: '创建', loading: creatingProject }"
      @confirm="createProject"
      ><div class="form-grid">
        <label>项目名称 <t-input v-model="projectDraft.name" /></label
        ><label
          >说明
          <t-textarea
            v-model="projectDraft.description"
            :autosize="{ minRows: 2, maxRows: 4 }"
        /></label>
        <p class="form-note">
          项目是独立组织单元。创建后再向其中添加一个或多个本地文件夹。
        </p>
      </div></t-dialog
    >
    <t-dialog
      v-model:visible="folderDialogOpen"
      header="添加本地文件夹"
      :confirm-btn="{ content: '添加', loading: creatingFolder }"
      @confirm="createFolder"
      ><div class="form-grid">
        <label
          >显示名称（可选）
          <t-input
            v-model="folderDraft.name"
            placeholder="默认使用目录名" /></label
        ><label
          >本地目录
          <t-input
            v-model="folderDraft.local_path"
            placeholder="/Users/you/project/APIForge"
        /></label>
        <p class="form-note">
          添加目录不会授予访问权限。首次创建对话时，系统会校验本机 Agent
          已授权该目录。
        </p>
      </div></t-dialog
    >
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { useRouter } from "vue-router";
import { MessagePlugin } from "tdesign-vue-next";
import Chat from "@/views/chat/index.vue";
import {
  createCodingFolder,
  createCodingProject,
  createCodingThread,
  getCodingThreadReview,
  listCodingFolders,
  listCodingProjects,
  listCodingThreads,
  type CodingProject,
  type CodingProjectFolder,
  type CodingThread,
  type ProjectInput,
} from "@/api/projects";

const router = useRouter();
const projects = ref<CodingProject[]>([]);
const folders = ref<CodingProjectFolder[]>([]);
const threads = ref<CodingThread[]>([]);
const selectedProjectId = ref("");
const selectedFolderId = ref("");
const selectedThreadId = ref("");
const foldersLoading = ref(false);
const threadsLoading = ref(false);
const projectDialogOpen = ref(false);
const folderDialogOpen = ref(false);
const creatingProject = ref(false);
const creatingFolder = ref(false);
const reviewLoading = ref(false);
const reviewDiff = ref("");
const reviewMessage = ref("");
const projectDraft = reactive<ProjectInput>({ name: "", description: "" });
const folderDraft = reactive({ name: "", local_path: "" });
const selectedProject = computed(() =>
  projects.value.find((project) => project.id === selectedProjectId.value),
);
const selectedFolder = computed(() =>
  folders.value.find((folder) => folder.id === selectedFolderId.value),
);
const emptyTitle = computed(() =>
  !selectedProject.value
    ? "选择或创建项目"
    : !selectedFolder.value
      ? "添加一个本地文件夹"
      : "创建一段对话",
);
const emptyDescription = computed(() =>
  !selectedProject.value
    ? "Coding 与知识库空间完全分开。"
    : !selectedFolder.value
      ? "一个项目可以关联多个本地目录。"
      : "对话会绑定到当前文件夹的已授权工作区。",
);

function switchWorkMode(mode: "chat" | "coding") {
  if (mode === "chat") router.push("/platform/creatChat");
}

async function loadProjects() {
  try {
    const response = await listCodingProjects();
    projects.value = response.data || [];
    if (!selectedProjectId.value && projects.value[0])
      await selectProject(projects.value[0].id);
  } catch (error: any) {
    MessagePlugin.error(error?.message || "读取项目失败");
  }
}
async function selectProject(projectId: string) {
  selectedProjectId.value = projectId;
  selectedFolderId.value = "";
  selectedThreadId.value = "";
  folders.value = [];
  threads.value = [];
  clearReview();
  foldersLoading.value = true;
  try {
    const response = await listCodingFolders(projectId);
    folders.value = response.data || [];
    if (folders.value[0]) await selectFolder(folders.value[0].id);
  } catch (error: any) {
    MessagePlugin.error(error?.message || "读取本地文件夹失败");
  } finally {
    foldersLoading.value = false;
  }
}
async function selectFolder(folderId: string) {
  if (!selectedProject.value) return;
  selectedFolderId.value = folderId;
  selectedThreadId.value = "";
  threads.value = [];
  clearReview();
  threadsLoading.value = true;
  try {
    const response = await listCodingThreads(
      selectedProject.value.id,
      folderId,
    );
    threads.value = response.data || [];
    if (threads.value[0]) selectedThreadId.value = threads.value[0].id;
  } catch (error: any) {
    MessagePlugin.error(error?.message || "读取对话失败");
  } finally {
    threadsLoading.value = false;
  }
}
function selectThread(threadId: string) {
  selectedThreadId.value = threadId;
  clearReview();
}
function clearReview() {
  reviewDiff.value = "";
  reviewMessage.value = "";
}
function openProjectDialog() {
  Object.assign(projectDraft, { name: "", description: "" });
  projectDialogOpen.value = true;
}
function openFolderDialog() {
  if (!selectedProject.value) return;
  Object.assign(folderDraft, { name: "", local_path: "" });
  folderDialogOpen.value = true;
}
async function createProject() {
  if (!projectDraft.name.trim()) {
    MessagePlugin.warning("请填写项目名称");
    return;
  }
  creatingProject.value = true;
  try {
    const response = await createCodingProject({
      name: projectDraft.name.trim(),
      description: projectDraft.description?.trim(),
    });
    projects.value.unshift(response.data);
    projectDialogOpen.value = false;
    await selectProject(response.data.id);
  } catch (error: any) {
    MessagePlugin.error(error?.message || "创建项目失败");
  } finally {
    creatingProject.value = false;
  }
}
async function createFolder() {
  if (!selectedProject.value || !folderDraft.local_path.trim()) {
    MessagePlugin.warning("请填写本地目录");
    return;
  }
  creatingFolder.value = true;
  try {
    const response = await createCodingFolder(selectedProject.value.id, {
      name: folderDraft.name.trim(),
      local_path: folderDraft.local_path.trim(),
    });
    folders.value.unshift(response.data);
    folderDialogOpen.value = false;
    await selectFolder(response.data.id);
  } catch (error: any) {
    MessagePlugin.error(error?.message || "添加本地文件夹失败");
  } finally {
    creatingFolder.value = false;
  }
}
async function newThread() {
  if (!selectedProject.value || !selectedFolder.value) return;
  try {
    const response = await createCodingThread(
      selectedProject.value.id,
      selectedFolder.value.id,
    );
    threads.value.unshift(response.data);
    selectThread(response.data.id);
  } catch (error: any) {
    MessagePlugin.error(
      error?.message || "创建对话失败。请先通过本机 Agent 授权该目录。",
    );
  }
}
async function loadReview() {
  if (
    !selectedProject.value ||
    !selectedFolder.value ||
    !selectedThreadId.value
  )
    return;
  reviewLoading.value = true;
  reviewDiff.value = "";
  try {
    const response = await getCodingThreadReview(
      selectedProject.value.id,
      selectedFolder.value.id,
      selectedThreadId.value,
    );
    reviewDiff.value = response.data.diff || "";
    reviewMessage.value = response.data.truncated
      ? `${response.data.message} Diff 已截断。`
      : response.data.message;
  } catch (error: any) {
    reviewMessage.value = error?.message || "暂时无法读取 Git Diff";
  } finally {
    reviewLoading.value = false;
  }
}
onMounted(loadProjects);
</script>

<style scoped lang="less">
.coding-workspace {
  height: 100%;
  min-height: 0;
  display: grid;
  grid-template-columns: 250px minmax(0, 1fr) 270px;
  background: var(--td-bg-color-page);
  color: var(--td-text-color-primary);
}
.coding-nav,
.coding-inspector {
  min-width: 0;
  padding: 16px 12px;
  background: var(--td-bg-color-container);
}
.coding-nav {
  border-right: 1px solid var(--td-component-border);
}
.coding-inspector {
  border-left: 1px solid var(--td-component-border);
}
.coding-brand {
  display: flex;
  align-items: center;
  gap: 9px;
  padding: 6px 8px 18px;
  font-weight: 650;
}
.coding-brand__mark {
  display: grid;
  place-items: center;
  width: 26px;
  height: 26px;
  border-radius: var(--app-radius-md);
  background: var(--td-brand-color);
  color: var(--td-text-color-anti);
  font:
    700 11px/1 ui-monospace,
    monospace;
}
.primary-action {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-height: 34px;
  width: 100%;
  border: 0;
  border-radius: var(--app-radius-md);
  background: var(--td-brand-color);
  color: var(--td-text-color-anti);
  cursor: pointer;
  font-weight: 600;
}
.primary-action:hover {
  background: var(--td-brand-color-hover);
}
.primary-action.compact {
  width: auto;
  padding: 0 16px;
}
.project-list,
.folder-list {
  display: grid;
  gap: 3px;
  margin-top: 18px;
}
.project-row,
.folder-row,
.thread-row {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 8px;
  border: 0;
  border-radius: var(--app-radius-sm);
  background: transparent;
  color: inherit;
  text-align: left;
  cursor: pointer;
  overflow: hidden;
}
.project-row span,
.folder-row span,
.thread-row span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.project-row:hover,
.folder-row:hover,
.thread-row:hover {
  background: var(--td-bg-color-secondarycontainer);
}
.project-row.is-active,
.folder-row.is-active,
.thread-row.is-active {
  background: var(--td-brand-color-light);
  color: var(--td-brand-color);
}
.section-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 22px 8px 7px;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-sm);
}
.section-heading--threads {
  padding-top: 18px;
}
.section-heading button,
.review-heading button {
  border: 0;
  background: transparent;
  color: var(--td-brand-color);
  cursor: pointer;
}
.empty-note,
.muted {
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-md);
  line-height: 1.6;
}
.empty-note {
  padding: 0 8px;
}
.coding-main {
  min-width: 0;
  min-height: 0;
  overflow: hidden;
}
.coding-main :deep(.chat),
.coding-main :deep(.chat_thread) {
  height: 100%;
}
.coding-empty {
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px;
  text-align: center;
}
.coding-empty__glyph {
  margin-bottom: 20px;
  color: var(--td-brand-color);
  font:
    700 48px/1 ui-monospace,
    monospace;
}
.coding-empty h1 {
  margin: 0;
  font-size: var(--app-text-4xl);
}
.coding-empty p {
  max-width: 430px;
  color: var(--td-text-color-secondary);
  line-height: 1.7;
}
.inspector-heading {
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-sm);
  font-weight: 650;
  text-transform: uppercase;
  letter-spacing: 0.08em;
}
.coding-inspector h2,
.coding-inspector h3 {
  margin: 10px 0 6px;
}
.coding-inspector h2 {
  font-size: var(--app-text-2xl);
}
.coding-inspector h3 {
  font-size: var(--app-text-lg);
}
.project-path {
  margin: 0;
  overflow-wrap: anywhere;
  color: var(--td-text-color-secondary);
  font:
    12px/1.55 ui-monospace,
    monospace;
}
.inspector-divider {
  height: 1px;
  margin: 24px 0;
  background: var(--td-component-border);
}
.review-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.review-heading button:disabled {
  cursor: wait;
  opacity: 0.6;
}
.review-diff {
  max-height: 330px;
  margin: 10px 0 0;
  padding: 10px;
  overflow: auto;
  border: 1px solid var(--td-component-border);
  border-radius: var(--app-radius-sm);
  background: #0d1117;
  color: #d0d7de;
  font:
    11px/1.5 ui-monospace,
    monospace;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}
.form-grid {
  display: grid;
  gap: 14px;
}
.form-grid label {
  display: grid;
  gap: 6px;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-md);
}
.form-note {
  margin: 0;
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-sm);
  line-height: 1.6;
}
@media (max-width: 1050px) {
  .coding-workspace {
    grid-template-columns: 220px minmax(0, 1fr);
  }
  .coding-inspector {
    display: none;
  }
}
@media (max-width: 720px) {
  .coding-workspace {
    grid-template-columns: 1fr;
  }
  .coding-nav {
    display: none;
  }
}
.coding-mode-switch {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 3px;
  margin: 0 0 12px;
  padding: 3px;
  border-radius: var(--app-radius-md);
  background: var(--td-bg-color-container-hover);
}
.coding-mode-switch button {
  min-height: 28px;
  border: 0;
  border-radius: var(--app-radius-sm);
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;
  font-size: var(--app-text-sm);
  font-weight: 600;
}
.coding-mode-switch button.is-active {
  background: var(--td-bg-color-container);
  color: var(--td-brand-color);
  box-shadow: var(--td-shadow-1);
}
</style>
