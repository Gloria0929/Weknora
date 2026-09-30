import { ref, provide, inject, type InjectionKey, type Ref } from 'vue'

export interface FileChange {
  id: string
  path: string
  type: 'added' | 'modified' | 'deleted'
  /** Pre-write text. Absent when the file was new or too large to ship. */
  before?: string
  /** Post-write text. Absent when the payload was too large to ship. */
  after?: string
  addedLines: number
  removedLines: number
  timestamp: number
  /** True for a change from the live stream, false for history replay. */
  live: boolean
}

export interface FileChangesContext {
  changes: Ref<FileChange[]>
  activeId: Ref<string | null>
  addChange: (change: FileChange) => void
  clearChanges: () => void
  hasChanges: () => boolean
}

const FILE_CHANGES_KEY: InjectionKey<FileChangesContext> = Symbol('fileChanges')

export function provideFileChanges() {
  const changes = ref<FileChange[]>([])
  const activeId = ref<string | null>(null)

  const addChange = (change: FileChange) => {
    // One row per path, newest wins: the code view shows the current state of
    // a file, never a stack of stale revisions of the same file.
    const existing = changes.value.findIndex((c) => c.path === change.path)
    if (existing >= 0) {
      // Replacing in place keeps the original position, so a turn that cycles
      // back to an earlier file does not reshuffle the list under the reader.
      changes.value = changes.value.map((c, index) =>
        index === existing ? change : c,
      )
    } else {
      changes.value = [...changes.value, change]
    }
    activeId.value = change.id
  }

  const clearChanges = () => {
    changes.value = []
    activeId.value = null
  }

  const hasChanges = () => changes.value.length > 0

  const ctx: FileChangesContext = {
    changes,
    activeId,
    addChange,
    clearChanges,
    hasChanges,
  }

  provide(FILE_CHANGES_KEY, ctx)
  return ctx
}

/** The change recorded for `path`, matching on either the full or tree path. */
export function findFileChange(
  changes: FileChange[],
  path: string,
): FileChange | undefined {
  if (!path) return undefined
  const normalized = path.replaceAll('\\', '/')
  const bare = normalized.startsWith('/') ? normalized.slice(1) : normalized
  return changes.find((change) => {
    const target = change.path.replaceAll('\\', '/')
    const targetBare = target.startsWith('/') ? target.slice(1) : target
    return target === normalized || targetBare === bare || targetBare.endsWith(`/${bare}`)
  })
}

export function useFileChanges(): FileChangesContext {
  const ctx = inject(FILE_CHANGES_KEY)
  if (!ctx) {
    // Return a no-op context when not provided (e.g. outside a chat session)
    const changes = ref<FileChange[]>([])
    return {
      changes,
      activeId: ref<string | null>(null),
      addChange: () => {},
      clearChanges: () => {},
      hasChanges: () => false,
    }
  }
  return ctx
}
