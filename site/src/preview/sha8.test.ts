import { describe, expect, it } from 'vitest'
import { sha8 } from './sha8'

describe('sha8', () => {
  it('is the first 4 bytes of SHA-256 in hex, as in Go', async () => {
    expect(await sha8('대상')).toBe('8c609deb')
    expect(await sha8('目標')).toBe('c92c0b98')
  })
})
