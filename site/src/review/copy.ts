import { stripTags } from '../search/match'

const EXCERPT = 40

/** What "copy ID" copies: the id, the first 40 characters of the text without its tags, and an arrow to write the decision after; detail, if any, stands between the quote and the arrow, whole (it is not cut). */
export function copyText(id: string, excerptSource: string, detail = ''): string {
  const chars = Array.from(stripTags(excerptSource).replace(/\s+/g, ' ').trim())
  const excerpt = chars.length > EXCERPT ? `${chars.slice(0, EXCERPT).join('')}…` : chars.join('')
  return `${id}「${excerpt}」${detail}→ `
}

/** Puts the text on the clipboard; where the browser says no, selects the text of fallbackEl for the reader to copy. */
export async function copyToClipboard(text: string, fallbackEl: HTMLElement): Promise<'copied' | 'selected'> {
  try {
    await navigator.clipboard.writeText(text)
    return 'copied'
  } catch {
    try {
      const selection = window.getSelection()
      const range = document.createRange()
      range.selectNodeContents(fallbackEl)
      selection?.removeAllRanges()
      selection?.addRange(range)
    } catch {
      // Nothing more to try; the text is still on the page.
    }
    return 'selected'
  }
}
