import { describe, expect, it } from 'vitest'
import { suggestionKindName, warningInfo } from './names'

describe('names', () => {
  it('names the name-differs warning', () => {
    expect(warningInfo('name-differs', '').name).toBe('與既有譯名不同')
  })
  it('names the name suggestion', () => {
    expect(suggestionKindName('name')).toBe('與既有譯名不同')
  })
})
