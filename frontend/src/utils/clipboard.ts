/**
 * 复制文本，兼容通过 HTTP 或 IP 地址访问时不可用的 Clipboard API。
 */
export async function copyText(text: string): Promise<void> {
  if (!text) throw new Error('没有可复制的内容')

  if (window.isSecureContext && navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(text)
      return
    } catch {
      // 权限策略或浏览器设置可能仍会拒绝，继续使用同步回退方案。
    }
  }

  if (copyWithSelection(text)) return
  throw new Error('当前浏览器不允许访问剪贴板')
}

function copyWithSelection(text: string): boolean {
  if (!document.body || typeof document.execCommand !== 'function') return false

  const activeElement = document.activeElement
  const selection = document.getSelection?.()
  const selectedRanges = selection
    ? Array.from({ length: selection.rangeCount }, (_, index) => selection.getRangeAt(index))
    : []
  const textarea = document.createElement('textarea')
  textarea.value = text
  textarea.setAttribute('aria-hidden', 'true')
  textarea.setAttribute('tabindex', '-1')
  Object.assign(textarea.style, {
    position: 'fixed',
    top: '-9999px',
    left: '-9999px',
    pointerEvents: 'none',
  })

  document.body.appendChild(textarea)
  textarea.focus({ preventScroll: true })
  textarea.select()
  textarea.setSelectionRange(0, textarea.value.length)

  // Explicitly supply text/plain when execCommand dispatches the copy event.
  // This avoids relying solely on hidden-textarea selection behavior, which
  // differs between browsers when the page is served over HTTP.
  const handleCopy = (event: ClipboardEvent) => {
    if (!event.clipboardData) return
    event.clipboardData.setData('text/plain', text)
    event.preventDefault()
  }

  let copied = false
  document.addEventListener('copy', handleCopy, { once: true })
  try {
    copied = document.execCommand('copy')
  } catch {
    copied = false
  } finally {
    document.removeEventListener('copy', handleCopy)
    textarea.remove()
    if (selection) {
      selection.removeAllRanges()
      selectedRanges.forEach((range) => selection.addRange(range))
    }
    if (activeElement instanceof HTMLElement && activeElement.isConnected) {
      activeElement.focus({ preventScroll: true })
    }
  }
  return copied
}
