import { describe, expect, it } from 'vitest'
const files = import.meta.glob('../styles.css', { eager: true, query: '?raw', import: 'default' }) as Record<string, string>
const css = files['../styles.css'] ?? ''

function rule(selector: string): string {
  const esc = selector.replace(/[.[\]()]/g, '\\$&')
  const m = new RegExp(`(?:^|\\})\\s*${esc}\\s*\\{([^}]*)\\}`, 'm').exec(css)
  if (!m) throw new Error(`no rule ${selector}`)
  return m[1] as string
}

// jsdom has no layout: at 400px the gold page must not scroll sideways.
describe('gold layout CSS', () => {
  it('.gold is a grid whose column can shrink', () => {
    expect(rule('.gold')).toMatch(/grid-template-columns:\s*minmax\(0,\s*1fr\)/)
  })
  it('the report table scrolls inside its own box', () => {
    expect(rule('.gold-table-wrap')).toMatch(/overflow-x:\s*auto/)
    expect(rule('.gold-table-wrap')).toMatch(/max-width:\s*100%/)
  })
  it('texts and the note box wrap within the width', () => {
    expect(rule('.gold-text')).toMatch(/overflow-wrap:\s*anywhere/)
    expect(rule('.gold-note')).toMatch(/max-width:\s*100%/)
  })
})
