import { post } from '@/utils/request'

export function dirFromPickerResponse(body: { data?: { dir?: unknown } } | null | undefined): string {
  const dir = body?.data?.dir
  return typeof dir === 'string' ? dir.trim() : ''
}

export function isHostProjectPickerUnavailable(error: unknown): boolean {
  const candidate = error as { $httpStatus?: number; status?: number; message?: unknown } | null
  const message = typeof candidate?.message === 'string' ? candidate.message : ''
  return candidate?.$httpStatus === 404 || candidate?.status === 404 || /host project picker is not available/i.test(message)
}

// Prefer the Wails binding when the desktop shell is present. Lite's SPA is
// also reverse-proxied in some deployments, so keep the same-process HTTP
// endpoint as the fallback.
export async function pickHostProjectDir(): Promise<string> {
  const wailsPicker = (globalThis as any)?.go?.main?.App?.PickProjectDir
  if (typeof wailsPicker === 'function') {
    try {
      const picked = await wailsPicker()
      return typeof picked === 'string' ? picked.trim() : ''
    } catch {
      // A stale desktop binding can exist while the HTTP bridge is ready.
      // Fall through and try the bridge before showing the manual-path hint.
    }
  }

  try {
    const res = await post<{ data?: { dir?: unknown } }>('/api/v1/system/host-project-dir', {}, { timeout: 0 })
    return dirFromPickerResponse(res)
  } catch (error) {
    if (isHostProjectPickerUnavailable(error)) {
      const unavailable = new Error('当前运行环境不支持原生文件夹选择，请直接输入本地目录路径')
      const typedUnavailable = unavailable as Error & { code?: string }
      typedUnavailable.code = 'HOST_PROJECT_PICKER_UNAVAILABLE'
      throw unavailable
    }
    throw error
  }
}
