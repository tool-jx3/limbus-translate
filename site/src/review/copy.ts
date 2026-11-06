import { stripTags } from '../search/match'

const EXCERPT = 40

/** What a page says above the selected box where the browser did not let it write to the clipboard. */
export const COPY_REFUSED_HINT = '瀏覽器不讓網頁寫入剪貼簿：已選取下面的文字，按 Ctrl+C（或 ⌘C）複製'

/** What "copy ID" copies: the id, the first 40 characters of the text without its tags, and an arrow to write the decision after; detail, if any, stands between the quote and the arrow, whole (it is not cut). */
export function copyText(id: string, excerptSource: string, detail = ''): string {
  const chars = Array.from(stripTags(excerptSource).replace(/\s+/g, ' ').trim())
  const excerpt = chars.length > EXCERPT ? `${chars.slice(0, EXCERPT).join('')}…` : chars.join('')
  return `${id}「${excerpt}」${detail}→ `
}

/** Puts the text on the clipboard; where the browser says no, selects the text of fallbackEl (a textarea selects its value) for the reader to copy. */
export async function copyToClipboard(text: string, fallbackEl: HTMLElement): Promise<'copied' | 'selected'> {
  try {
    await navigator.clipboard.writeText(text)
    return 'copied'
  } catch {
    try {
      const selection = window.getSelection()
      if (fallbackEl instanceof HTMLTextAreaElement) {
        fallbackEl.focus()
        fallbackEl.select()
      } else {
        const range = document.createRange()
        range.selectNodeContents(fallbackEl)
        selection?.removeAllRanges()
        selection?.addRange(range)
      }
    } catch {
      // Nothing more to try; the text is still on the page.
    }
    return 'selected'
  }
}
