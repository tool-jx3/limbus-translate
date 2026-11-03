import { describe, expect, it } from 'vitest'
import { variantDiff } from './diff'

describe('variantDiff', () => {
  it('marks only what differs, and the highlight style counts', () => {
    const pieces = variantDiff('使目標增加<style="highlight">2</style>回 [Laceration]', '使目標增加<style="highlight">2回</style> [Laceration]')
    expect(pieces.filter((p) => !p.same).map((p) => p.text)).toEqual(['回'])
    expect(pieces.find((p) => p.text === '回')?.hl).toBe(true)
  })

  it('shows a space the other has more as different', () => {
    const pieces = variantDiff('使目標增加 [Laceration]', '使目標增加 [Laceration] ')
    const last = pieces[pieces.length - 1]
    expect(last).toEqual({ text: ' ', tok: false, hl: false, same: false })
    expect(pieces.slice(0, -1).every((p) => p.same)).toBe(true)
  })

  it('takes an [ID] as one piece', () => {
    const pieces = variantDiff('a [Laceration]', 'a [Laceration]')
    const tokens = pieces.filter((p) => p.tok)
    expect(tokens).toHaveLength(1)
    expect(tokens[0]?.text).toBe('[Laceration]')
    expect(pieces.every((p) => p.same)).toBe(true)
  })

  it('marks every piece of an unrelated text', () => {
    expect(variantDiff('abc', 'xyz').every((p) => !p.same)).toBe(true)
  })
})
