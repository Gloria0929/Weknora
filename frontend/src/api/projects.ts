import { del, get, post, put } from '@/utils/request'

export interface CodingProject {
  id: string
  tenant_id: number
  name: string
  description: string
  default_model: string
  default_editor: string
  git_enabled: boolean
  settings: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface CodingThread {
  id: string
  project_id: string
  folder_id: string
  mode: 'coding'
  title: string
  created_at: string
  updated_at: string
}

export interface CodingProjectFolder {
  id: string
  project_id: string
  name: string
  local_path: string
  created_at: string
  updated_at: string
}

export interface CodingThreadReview {
  available: boolean
  diff: string
  truncated?: boolean
  message: string
}

export interface ProjectInput {
  name: string
  description?: string
  default_model?: string
  default_editor?: string
  git_enabled?: boolean
  settings?: Record<string, unknown>
}

const base = '/api/v1/projects'

export const listCodingProjects = () => get<{ success: boolean; data: CodingProject[] }>(base)
export const getCodingProject = (id: string) => get<{ success: boolean; data: CodingProject }>(`${base}/${encodeURIComponent(id)}`)
export const createCodingProject = (data: ProjectInput) => post<{ success: boolean; data: CodingProject }>(base, data)
export const updateCodingProject = (id: string, data: Partial<ProjectInput>) => put<{ success: boolean; data: CodingProject }>(`${base}/${encodeURIComponent(id)}`, data)
export const deleteCodingProject = (id: string) => del<{ success: boolean }>(`${base}/${encodeURIComponent(id)}`)
export const listCodingFolders = (projectId: string) => get<{ success: boolean; data: CodingProjectFolder[] }>(`${base}/${encodeURIComponent(projectId)}/folders`)
export const createCodingFolder = (projectId: string, data: { name?: string; local_path: string }) => post<{ success: boolean; data: CodingProjectFolder }>(`${base}/${encodeURIComponent(projectId)}/folders`, data)
export const deleteCodingFolder = (projectId: string, folderId: string) => del<{ success: boolean }>(`${base}/${encodeURIComponent(projectId)}/folders/${encodeURIComponent(folderId)}`)
export const listCodingThreads = (projectId: string, folderId: string) => get<{ success: boolean; data: CodingThread[] }>(`${base}/${encodeURIComponent(projectId)}/folders/${encodeURIComponent(folderId)}/threads`)
export const createCodingThread = (projectId: string, folderId: string, title = '') => post<{ success: boolean; data: CodingThread }>(`${base}/${encodeURIComponent(projectId)}/folders/${encodeURIComponent(folderId)}/threads`, { title })
export const getCodingThreadReview = (projectId: string, folderId: string, threadId: string) => get<{ success: boolean; data: CodingThreadReview }>(`${base}/${encodeURIComponent(projectId)}/folders/${encodeURIComponent(folderId)}/threads/${encodeURIComponent(threadId)}/review`)
