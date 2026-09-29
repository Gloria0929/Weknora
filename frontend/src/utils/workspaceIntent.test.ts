import test from 'node:test'
import assert from 'node:assert/strict'
import { activateWorkspaceAgent, workspaceIntentFromPrompt } from './workspaceIntent'

test('natural coding requests activate the workspace without treating explanations as actions', () => {
  assert.equal(workspaceIntentFromPrompt('帮我创建一个产品网站'), 'build')
  assert.equal(workspaceIntentFromPrompt('根据知识库帮我编写一个 Python 脚本'), 'code')
  assert.equal(workspaceIntentFromPrompt('运行这个项目的测试'), 'test')
  assert.equal(workspaceIntentFromPrompt('Please build a website'), 'build')
  assert.equal(workspaceIntentFromPrompt('什么是代码覆盖率？'), 'chat')
  assert.equal(workspaceIntentFromPrompt('解释这份文档', 'knowledge'), 'knowledge')
})

test('agent activation preserves explicit knowledge resources after composer watchers clear defaults', async () => {
  let saved = false
  const store = {
    selectedAgentId: 'builtin-quick-answer',
    settings: { selectedKnowledgeBases: ['kb'], selectedFiles: ['file'], selectedFileKbMap: { file: 'kb' }, selectedTags: ['tag'], selectedMCPServices: ['mcp'], selectedSkills: ['skill'] },
    selectAgent(id: string) { this.selectedAgentId = id; this.settings.selectedKnowledgeBases = []; this.settings.selectedFiles = [] },
    saveSettings() { saved = true },
  }
  await activateWorkspaceAgent(store, async () => { store.settings.selectedKnowledgeBases = []; store.settings.selectedSkills = [] })
  assert.equal(store.selectedAgentId, 'builtin-smart-reasoning')
  assert.deepEqual(store.settings.selectedKnowledgeBases, ['kb'])
  assert.deepEqual(store.settings.selectedFiles, ['file'])
  assert.deepEqual(store.settings.selectedSkills, ['skill'])
  assert.equal(saved, true)
})

test('custom agents and a newer agent choice remain authoritative', async () => {
  let calls = 0
  const store = {
    selectedAgentId: 'custom',
    settings: { selectedKnowledgeBases: [] as string[], selectedFiles: [] as string[], selectedFileKbMap: {}, selectedTags: [], selectedMCPServices: [], selectedSkills: [] },
    selectAgent(id: string) { calls++; this.selectedAgentId = id }, saveSettings() { calls++ },
  }
  await activateWorkspaceAgent(store, async () => {})
  assert.equal(calls, 0)
  store.selectedAgentId = 'builtin-quick-answer'
  await activateWorkspaceAgent(store, async () => { store.selectedAgentId = 'another-custom' })
  assert.equal(store.selectedAgentId, 'another-custom')
  assert.equal(calls, 1)
})
