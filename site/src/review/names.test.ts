import { describe, expect, it } from 'vitest'
import { suggestionKindName, warningInfo } from './names'

describe('names', () => {
  it('names the name-differs warning', () => {
    expect(warningInfo('name-differs', '').name).toBe('與既有譯名不同')
  })
  it('names the name suggestion', () => {
    expect(suggestionKindName('name')).toBe('與既有譯名不同')
  })
  it('names every check of the game text', () => {
    const codes = ['keyword-link', 'bracket-text', 'page-count', 'level-diff', 'story-comment', 'stray-brace', 'regex-char',
      'literal-newline', 'reserved-value', 'newline-space', 'qmark-count', 'bullet-count', 'ruby-form', 'shared-line']
    for (const code of codes) {
      const info = warningInfo(code, 'message of the check')
      expect(info.name).not.toBe(code)
      expect(info.desc).not.toBe('message of the check')
      if (code !== 'stray-brace' && code !== 'bullet-count' && code !== 'shared-line') expect(info.desc).toContain('依遊戲程式推算')
    }
  })
})
