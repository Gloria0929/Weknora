import { onBeforeUnmount, ref, watch, type Ref } from 'vue'
import { discoverProgrammingWebPreview, type ProgrammingWebPreview } from '@/api/programming'
import { resolveProjectPreviewUrl } from '@/utils/projectPreview'

/** Discovery follows the active session; polling never starts or resumes a sandbox. */
export function useWorkspaceWebPreview(
  session: Ref<string>, active: Ref<boolean>, revision: Ref<number | undefined>,
  discover = discoverProgrammingWebPreview,
) {
  const result = ref<ProgrammingWebPreview | null>(null)
  const currentUrl = ref(''), busy = ref(false), frameKey = ref(0)
  let previewID = '', generation = 0, pendingRefresh = false
  let timer: ReturnType<typeof setTimeout> | undefined
  function schedule(delay: number) {
    clearTimeout(timer)
    if (active.value && session.value) timer = setTimeout(() => { void refresh() }, delay)
  }
  async function refresh(start = false) {
    if (!active.value || !session.value) return
    if (busy.value) { pendingRefresh = true; return }
    clearTimeout(timer)
    const epoch = generation
    busy.value = true
    try {
      const response = await discover(session.value, { start, preview_id: previewID })
      if (epoch !== generation) return
      const data = response.data
      if (data.status === 'ready' && data.url && data.preview_id) {
        currentUrl.value = resolveProjectPreviewUrl(data.url, window.location.href)
        previewID = data.preview_id
        result.value = data
      } else {
        currentUrl.value = ''
        result.value = data.status === 'ready' ? { status: 'error' } : data
      }
    } catch {
      if (epoch === generation) {
        result.value = { status: 'error' }
        // Avoid destroying a working iframe on a transient control-plane error.
      }
    } finally {
      if (epoch === generation) {
        busy.value = false
        schedule(pendingRefresh ? 300 : result.value?.status === 'ready' ? 10_000 : 5_000)
        pendingRefresh = false
      }
    }
  }
  function reload() {
    if (currentUrl.value) frameKey.value++
    void refresh()
  }
  watch(session, () => {
    generation++; clearTimeout(timer)
    previewID = ''; result.value = null; currentUrl.value = ''; busy.value = false; pendingRefresh = false
  }, { flush: 'sync' })
  watch([session, active, revision], () => {
    clearTimeout(timer)
    if (active.value && session.value) schedule(150)
  }, { immediate: true })
  onBeforeUnmount(() => { generation++; clearTimeout(timer) })
  return { result, currentUrl, busy, frameKey, refresh, reload }
}
