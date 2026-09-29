import { get, post, put } from '@/utils/request'

export interface ProgrammingNode {
  path: string
  name: string
  kind: 'file' | 'directory'
  size?: number
  modified?: string
}

export interface ProgrammingTree {
  root: string
  origin: 'host' | 'sandbox'
  /** "expired" means the session sandbox was reclaimed and its files are gone. */
  workspace_state?: 'live' | 'expired'
  nodes: ProgrammingNode[]
}

export interface ProgrammingFile {
  path: string
  content: string
  hash: string
  size: number
  modified?: string
}

export interface ProgrammingCommandResult {
  stdout: string
  stderr: string
  exit_code: number
  duration_ms: number
  killed: boolean
}

export interface ProgrammingPreviewCapabilities {
  pinned: boolean
  desktop_enabled: boolean
}
export interface ProgrammingPreviewResult {
  status: 'ready' | 'not_running' | 'starting' | 'start_failed' | 'browser_unavailable' | 'unsupported'
  url?: string
  detail?: string
  viewport?: 'desktop' | 'mobile'
  width?: number
  height?: number
}

export interface ProgrammingPreviewRequest {
  url?: string
  start?: boolean
  action?: 'resize' | 'reload'
  viewport?: 'desktop' | 'mobile'
}

export function getProgrammingPreviewCapabilities(sessionId: string) {
  return get<{ success: boolean; data: ProgrammingPreviewCapabilities }>(`${base(sessionId)}/preview`)
}

export function openProgrammingPreview(sessionId: string, data: ProgrammingPreviewRequest = {}) {
  return post<{ success: boolean; data: ProgrammingPreviewResult }>(`${base(sessionId)}/preview`, data, { timeout: 50_000 })
}

const base = (sessionId: string) => `/api/v1/sessions/${encodeURIComponent(sessionId)}/programming`

export function getProgrammingTree(sessionId: string, path = '') {
  const query = path ? `?path=${encodeURIComponent(path)}` : ''
  return get<{ success: boolean; data: ProgrammingTree }>(`${base(sessionId)}/tree${query}`)
}

export function getProgrammingFile(sessionId: string, path: string) {
  return get<{ success: boolean; data: ProgrammingFile }>(`${base(sessionId)}/file?path=${encodeURIComponent(path)}`)
}

export function saveProgrammingFile(sessionId: string, data: { path: string; content: string; base_hash?: string }) {
  return put<{ success: boolean; data: ProgrammingFile }>(`${base(sessionId)}/file`, data)
}

export function createProgrammingFile(sessionId: string, path: string) {
  return post<{ success: boolean; data: ProgrammingFile }>(`${base(sessionId)}/file`, { path })
}

export function moveProgrammingFile(sessionId: string, source: string, target: string) {
  return post<{ success: boolean; data: ProgrammingFile }>(`${base(sessionId)}/file/move`, { source, target })
}

export function deleteProgrammingFile(sessionId: string, path: string) {
  return post<{ success: boolean; data: { path: string } }>(`${base(sessionId)}/file/delete`, { path })
}

export function runProgrammingCommand(sessionId: string, data: { command: string; path?: string; timeout?: number }) {
  return post<{ success: boolean; data: ProgrammingCommandResult }>(`${base(sessionId)}/command`, data, { timeout: 70_000 })
}
