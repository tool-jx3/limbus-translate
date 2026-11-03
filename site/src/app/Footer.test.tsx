import { render } from 'preact'
import { afterEach, describe, expect, it } from 'vitest'
import { fixture } from '../testdata'
import type { Meta } from '../types'
import { Footer } from './Footer'

const meta = fixture<Meta>('meta.json')
const containers: HTMLElement[] = []

function show(m?: Meta): HTMLElement {
  const container = document.createElement('div')
  document.body.append(container)
  containers.push(container)
  render(<Footer meta={m} />, container)
  return container
}

afterEach(() => {
  for (const container of containers.splice(0)) {
    render(null, container)
    container.remove()
  }
})

const LICENSE =
  'KR／JP／EN 原文著作權屬 Project Moon；本站非商業、供翻譯校對使用。TW 譯文採用 CC BY-NC-SA 4.0，含 Paratranz 社群譯文（第一階段匯入）。'

describe('Footer', () => {
  it('carries the license text, word for word, before the data is loaded too', () => {
    expect(show().textContent).toContain(LICENSE)
    expect(show(meta).textContent).toContain(LICENSE)
  })

  it('shows the raw tag and links the release to its page', () => {
    const el = show(meta)
    expect(el.textContent).toContain('raw-20261029')
    const a = el.querySelector('a')
    expect(a?.textContent).toBe('zh-TW-20261029.5')
    expect(a?.getAttribute('href')).toBe('https://github.com/tool-jx3/limbus-translate/releases/tag/zh-TW-20261029.5')
    expect(a?.getAttribute('rel')).toContain('noopener')
  })

  it('has no link where the data has no release', () => {
    const el = show({ ...meta, release: '' })
    expect(el.textContent).toContain('raw-20261029')
    expect(el.querySelector('a')).toBeNull()
  })

  it('names neither Claude Code nor Anthropic', () => {
    expect(show(meta).textContent).not.toMatch(/Claude Code|Anthropic/i)
  })
})
