import { afterEach, describe, expect, it, vi } from 'vitest'
import { copyText, copyToClipboard } from './copy'

afterEach(() => {
  vi.unstubAllGlobals()
  Reflect.deleteProperty(navigator, 'clipboard')
})

describe('copyText', () => {
  it('gives the id, the text without tags, and an arrow', () => {
    expect(copyText('warning/kana/1315241d/2cc3aa54', '<color=#fff>し協會</color> 希斯克利夫')).toBe(
      'warning/kana/1315241d/2cc3aa54「し協會 希斯克利夫」→ ',
    )
  })

  it('folds white space into one space', () => {
    expect(copyText('x/1', 'a\n\n b\t c')).toBe('x/1「a b c」→ ')
  })

  it('cuts at 40 characters with an ellipsis', () => {
    const text = '字'.repeat(41)
    expect(copyText('x/1', text)).toBe(`x/1「${'字'.repeat(40)}…」→ `)
    expect(copyText('x/1', '字'.repeat(40))).toBe(`x/1「${'字'.repeat(40)}」→ `)
  })
})

describe('copyToClipboard', () => {
  it('copies the text', async () => {
    const writeText = vi.fn(async () => {})
    Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true })
    const el = document.createElement('code')
    expect(await copyToClipboard('abc', el)).toBe('copied')
    expect(writeText).toHaveBeenCalledWith('abc')
  })

  it('copy falls back to selecting', async () => {
    Object.defineProperty(navigator, 'clipboard', {
      value: { writeText: () => Promise.reject(new Error('denied')) },
      configurable: true,
    })
    const el = document.createElement('code')
    el.textContent = 'warning/kana/1/2'
    document.body.append(el)
    expect(await copyToClipboard('abc', el)).toBe('selected')
    expect(window.getSelection()?.toString()).toBe('warning/kana/1/2')
    el.remove()
  })

  it('falls back when there is no clipboard', async () => {
    const el = document.createElement('code')
    document.body.append(el)
    expect(await copyToClipboard('abc', el)).toBe('selected')
    el.remove()
  })
})
