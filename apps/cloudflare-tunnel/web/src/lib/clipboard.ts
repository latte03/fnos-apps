import { toast } from './toast'

/** Copy with a clipboard API attempt and a legacy textarea fallback. */
export async function copyText(text: string, label = '已复制'): Promise<boolean> {
  try {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(text)
      toast(label, 'success')
      return true
    }
  } catch {
    /* fall through to the legacy path */
  }

  const area = document.createElement('textarea')
  area.value = text
  area.setAttribute('readonly', '')
  area.style.position = 'fixed'
  area.style.opacity = '0'
  document.body.appendChild(area)
  area.select()
  try {
    const ok = document.execCommand('copy')
    if (ok) toast(label, 'success')
    else toast('复制失败，请手动选择', 'error')
    return ok
  } catch {
    toast('复制失败，请手动选择', 'error')
    return false
  } finally {
    area.remove()
  }
}
