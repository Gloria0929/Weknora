export const PREVIEW_CSP = "default-src 'none'; script-src 'unsafe-inline' blob:; style-src 'unsafe-inline'; img-src data: blob:; font-src data:; media-src data: blob:; connect-src 'none'; frame-src 'none'; object-src 'none'; base-uri 'none'; form-action 'none'"

/** Resolve local references without permitting a URL or a path outside the workspace. */
export function resolveWorkspaceAsset(entry: string, reference: string): string | null {
  let decoded: string
  try { decoded = decodeURIComponent(reference.split(/[?#]/)[0] || '') } catch { return null }
  if (!decoded || /^(?:[a-z][a-z\d+.-]*:|\/\/)/i.test(decoded) || decoded.includes('\\') || /[\u0000-\u001f]/.test(decoded)) return null
  const parts = decoded.startsWith('/') ? [] : entry.split('/').slice(0, -1)
  for (const part of decoded.split('/')) {
    if (part === '..') { if (!parts.length) return null; parts.pop() }
    else if (part && part !== '.') parts.push(part)
  }
  return parts.join('/') || null
}

export function canPreviewSource(path: string) { return /\.(?:html?|svg|md|markdown)$/i.test(path) }

/** The returned document always runs in an opaque-origin iframe with no network access. */
export async function buildWorkspacePreview(path: string, content: string, read: (path: string) => Promise<string>) {
  const doc = new DOMParser().parseFromString(content, 'text/html')
  const warnings: string[] = []
  doc.querySelectorAll('base, meta[http-equiv], iframe, object, embed').forEach(node => node.remove())
  const policy = doc.createElement('meta')
  policy.httpEquiv = 'Content-Security-Policy'
  policy.content = PREVIEW_CSP
  doc.head.prepend(policy)
  let bytes = content.length
  const resources = Array.from(doc.querySelectorAll('link[rel="stylesheet"][href], script[src], img[src]'))
  await Promise.all(resources.map(async (node, index) => {
    const attribute = node.tagName === 'LINK' ? 'href' : 'src'
    const reference = node.getAttribute(attribute) || ''
    if (node.tagName === 'IMG' && reference.startsWith('data:')) return
    const resolved = resolveWorkspaceAsset(path, reference)
    try {
      // The text-file endpoint deliberately rejects binary files. Only inline SVGs here.
      if (!resolved || index >= 24 || (node.tagName === 'IMG' && !/\.svg$/i.test(resolved))) throw new Error('unsupported asset')
      const source = await read(resolved)
      bytes += source.length
      if (bytes > 4 * 1024 * 1024) throw new Error('preview too large')
      if (node.tagName === 'IMG') {
        node.setAttribute('src', `data:image/svg+xml,${encodeURIComponent(source)}`)
      } else if (node.tagName === 'LINK') {
        const style = doc.createElement('style'); style.textContent = source; node.replaceWith(style)
      } else {
        node.removeAttribute('src'); node.removeAttribute('integrity'); node.textContent = source
      }
    } catch {
      warnings.push(reference)
      node.remove()
    }
  }))
  return { html: '<!doctype html>\n' + doc.documentElement.outerHTML, warnings }
}
