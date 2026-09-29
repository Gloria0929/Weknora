import type { WorkspaceIntent } from './workspaceEvents'

/** Conservative action detection; a question merely mentioning code stays a chat. */
export function workspaceIntentFromPrompt(prompt: string, selected: WorkspaceIntent = 'chat'): WorkspaceIntent {
  if (['build', 'code', 'test'].includes(selected)) return selected
  if (/((创建|生成|开发|制作|搭建).{0,16}(网页|网站|页面)|\b(?:build|create|make)\b.{0,24}\b(?:website|webpage|landing page)\b)/i.test(prompt)) return 'build'
  if (/((运行|执行|补充|编写).{0,8}测试|\b(?:run|write|execute)\b.{0,16}\btests?\b)/i.test(prompt)) return 'test'
  if (/((编写|实现|修复|重构|开发).{0,16}(代码|程序|脚本|应用|组件|接口)|\b(?:write|implement|fix|refactor)\b.{0,24}\b(?:code|script|app|component|api)\b)/i.test(prompt)) return 'code'
  return selected
}

type ResourceSettings = {
  selectedKnowledgeBases: string[]; selectedFiles: string[]; selectedFileKbMap: Record<string, string>;
  selectedTags: unknown[]; selectedMCPServices: string[]; selectedSkills: string[];
}

/** The UI's agent watchers run asynchronously; restore explicit resources after that flush. */
export async function activateWorkspaceAgent<T extends ResourceSettings>(store: {
  selectedAgentId: string; settings: T; selectAgent: (id: string) => void; saveSettings: (settings: T) => void
}, flush: () => Promise<unknown>) {
  if (store.selectedAgentId !== 'builtin-quick-answer') return
  const resources = {
    selectedKnowledgeBases: store.settings.selectedKnowledgeBases,
    selectedFiles: store.settings.selectedFiles,
    selectedFileKbMap: store.settings.selectedFileKbMap,
    selectedTags: store.settings.selectedTags,
    selectedMCPServices: store.settings.selectedMCPServices,
    selectedSkills: store.settings.selectedSkills,
  }
  store.selectAgent('builtin-smart-reasoning')
  await flush()
  // Read again after the asynchronous watcher flush; selectAgent mutates the store.
  if (String(store.selectedAgentId) !== 'builtin-smart-reasoning') return
  Object.assign(store.settings, resources)
  store.saveSettings(store.settings)
}
