import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defaultSettings, loadSettings, saveSettings, type Settings } from './settings'

const KEY = 'lct-viewer-settings'

beforeEach(() => localStorage.clear())
afterEach(() => vi.restoreAllMocks())

describe('settings', () => {
  it('has the defaults when nothing is saved', () => {
    expect(loadSettings()).toEqual({
      langs: { kr: true, jp: true, en: true, zh: true },
      origins: { paratranz: true, claude: true, human: true, copy: false, fallback: true, 'stand-in': true },
      rawTags: false,
      scriptMode: 'auto',
      reviewOnly: false,
    })
    expect(loadSettings()).toEqual(defaultSettings())
  })

  it('has the defaults when what is saved is not JSON', () => {
    localStorage.setItem(KEY, '{not json')
    expect(loadSettings()).toEqual(defaultSettings())
  })

  it('has the defaults when what is saved is not an object', () => {
    for (const text of ['null', '42', '"x"', '[1,2]', 'true']) {
      localStorage.setItem(KEY, text)
      expect(loadSettings(), text).toEqual(defaultSettings())
    }
  })

  it('reads back what it saved', () => {
    const s: Settings = {
      langs: { kr: true, jp: false, en: false, zh: true },
      origins: { paratranz: true, claude: false, human: true, copy: true, fallback: false, 'stand-in': true },
      rawTags: true,
      scriptMode: 'on',
      reviewOnly: true,
    }
    saveSettings(s)
    expect(loadSettings()).toEqual(s)
    expect(JSON.parse(localStorage.getItem(KEY) ?? 'null')).toEqual(s)
  })

  it('keeps the valid fields of a partly valid save and defaults the rest', () => {
    localStorage.setItem(
      KEY,
      JSON.stringify({ langs: { kr: false, jp: 'yes', fr: false }, origins: [], rawTags: 1, scriptMode: 'sometimes', reviewOnly: 'yes' }),
    )
    const s = loadSettings()
    expect(s.langs).toEqual({ kr: false, jp: true, en: true, zh: true })
    expect(s.origins).toEqual(defaultSettings().origins)
    expect(s.rawTags).toBe(false)
    expect(s.scriptMode).toBe('auto')
    expect(s.reviewOnly).toBe(false)
  })

  it('does not share its default between calls', () => {
    const a = loadSettings()
    a.langs.kr = false
    expect(loadSettings().langs.kr).toBe(true)
  })

  it('works without a usable localStorage', () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new DOMException('denied', 'SecurityError')
    })
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new DOMException('full', 'QuotaExceededError')
    })
    expect(loadSettings()).toEqual(defaultSettings())
    expect(() => saveSettings(defaultSettings())).not.toThrow()
  })
})
