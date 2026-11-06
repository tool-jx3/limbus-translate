import { describe, expect, it } from 'vitest'
const files = import.meta.glob('../styles.css', { eager: true, query: '?raw', import: 'default' }) as Record<string, string>
const css = files['../styles.css'] ?? ''

/** The declarations of the rule whose selector is exactly `selector`. */
function rule(selector: string): string {
  const esc = selector.replace(/[.[\]()]/g, '\\$&')
  const m = new RegExp(`(?:^|\\})\\s*${esc}\\s*\\{([^}]*)\\}`, 'm').exec(css)
  if (!m) throw new Error(`no rule ${selector}`)
  return m[1] as string
}

// jsdom has no layout: this guards the CSS that keeps a long line of a pre inside its box
// (a grid track of `auto` grows to the longest line, and the page then scrolls sideways).
describe('preview layout CSS', () => {
  for (const sel of ['.preview', '.pv-section', '.pv-panel', '.pv-head']) {
    it(`${sel} is a grid whose column can shrink`, () => {
      expect(rule(sel)).toMatch(/grid-template-columns:\s*minmax\(0,\s*1fr\)/)
    })
  }
  it('.pv-json and .pv-box can shrink and scroll inside themselves', () => {
    expect(rule('.pv-json')).toMatch(/min-width:\s*0/)
    expect(rule('.pv-box')).toMatch(/overflow:\s*auto/)
    expect(rule('.pv-box')).toMatch(/max-width:\s*100%/)
  })
})
