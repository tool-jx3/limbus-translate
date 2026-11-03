import { render } from 'preact'
import { h } from 'preact'
import { afterEach, describe, expect, it } from 'vitest'
import { TmpText } from '../tmp/TmpText'
import { WARNING_MARKS } from './marks'

const containers: HTMLElement[] = []

function show(text: string, mark: RegExp | undefined): HTMLElement {
  const container = document.createElement('div')
  document.body.append(container)
  containers.push(container)
  render(h(TmpText, { text, raw: false, lang: 'zh', ...(mark ? { mark } : {}) }), container)
  return container
}

afterEach(() => {
  for (const c of containers.splice(0)) {
    render(null, c)
    c.remove()
  }
})

describe('WARNING_MARKS', () => {
  it('marks the kana of kana', () => {
    const el = show('し協會 希斯克利夫', WARNING_MARKS.kana)
    expect([...el.querySelectorAll('mark')].map((m) => m.textContent)).toEqual(['し'])
    expect(el.textContent).toBe('し協會 希斯克利夫')
  })

  it('does not enter the text of a tag', () => {
    const el = show('<color=#ff0000>達</color>の時', WARNING_MARKS.kana)
    expect([...el.querySelectorAll('mark')].map((m) => m.textContent)).toEqual(['の'])
    expect(el.innerHTML).not.toContain('<mark>#')
    expect(el.querySelector('span[style] mark')).toBeNull()
  })

  it('marks again on every use of the same pattern', () => {
    const mark = WARNING_MARKS['halfwidth-punct']
    expect(show('好,壞', mark).querySelectorAll('mark')).toHaveLength(1)
    expect(show('好,壞', mark).querySelectorAll('mark')).toHaveLength(1)
  })

  it('has a mark for the codes of the spec and none for length-ratio', () => {
    expect(Object.keys(WARNING_MARKS).sort()).toEqual(['ascii-ellipsis', 'curly-quotes', 'halfwidth-bracket', 'halfwidth-punct', 'kana'])
  })
})
