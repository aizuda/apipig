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
  const textarea = document.createElement('textarea')
  textarea.value = text
  textarea.readOnly = true
  textarea.setAttribute('aria-hidden', 'true')
  Object.assign(textarea.style, {
    position: 'fixed',
    top: '0',
    left: '0',
    width: '1px',
    height: '1px',
    opacity: '0',
    pointerEvents: 'none',
  })

  document.body.appendChild(textarea)
  textarea.focus({ preventScroll: true })
  textarea.select()
  textarea.setSelectionRange(0, textarea.value.length)

  let copied = false
  try {
    copied = document.execCommand('copy')
  } catch {
    copied = false
  } finally {
    textarea.remove()
    if (activeElement instanceof HTMLElement) activeElement.focus({ preventScroll: true })
  }
  return copied
}
