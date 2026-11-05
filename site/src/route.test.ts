import { describe, expect, it } from 'vitest'
import { fileHref, parseHash, reviewHref, searchHref, updateHref, updatesHref } from './route'

describe('parseHash', () => {
  it('reads the empty hash and #/ as home', () => {
    expect(parseHash('')).toEqual({ kind: 'home' })
    expect(parseHash('#')).toEqual({ kind: 'home' })
    expect(parseHash('#/')).toEqual({ kind: 'home' })
  })

  it('reads an unknown path as home', () => {
    for (const h of ['#/nope', '#/f', '#/f/', '#/x/y?q=1', '#/search?q=a', '#/files/StoryData/S001A.json']) {
      expect(parseHash(h), h).toEqual({ kind: 'home' })
    }
  })

  it('reads a file with no pointer', () => {
    expect(parseHash('#/f/StoryData/S001A.json')).toEqual({ kind: 'file', path: 'StoryData/S001A.json' })
    expect(parseHash('#/f/StoryData/S001A.json?u=')).toEqual({ kind: 'file', path: 'StoryData/S001A.json' })
  })

  it('reads a search', () => {
    expect(parseHash('#/s?q=%E5%BE%9E&lang=zh&re=1')).toEqual({ kind: 'search', q: '從', lang: 'zh', re: true })
    expect(parseHash('#/s?q=cinq&lang=en')).toEqual({ kind: 'search', q: 'cinq', lang: 'en', re: false })
  })

  it('falls back to zh and substring for what a search link gets wrong', () => {
    expect(parseHash('#/s?q=a&lang=fr&re=yes')).toEqual({ kind: 'search', q: 'a', lang: 'zh', re: false })
    expect(parseHash('#/s')).toEqual({ kind: 'search', q: '', lang: 'zh', re: false })
  })

  it('does not throw on a bad percent escape, and keeps the text as it is', () => {
    expect(parseHash('#/s?q=100%&lang=kr')).toEqual({ kind: 'search', q: '100%', lang: 'kr', re: false })
    expect(parseHash('#/f/a%zz.json')).toEqual({ kind: 'file', path: 'a%zz.json' })
  })

  it('refuses a path that leaves the data directory', () => {
    for (const h of ['#/f/../x.json', '#/f/a/../b.json', '#/f/a/%2E%2E/b.json', '#/f/a//b.json', '#/f/./a.json']) {
      expect(parseHash(h), h).toEqual({ kind: 'home' })
    }
  })
})

describe('fileHref', () => {
  it('keeps the slashes of the path and encodes the pointer', () => {
    expect(fileHref('StoryData/S001A.json')).toBe('#/f/StoryData/S001A.json')
    expect(fileHref('StoryData/S001A.json', '/dataList/14/content')).toBe('#/f/StoryData/S001A.json?u=%2FdataList%2F14%2Fcontent')
  })

  it('goes back through parseHash to the same path and pointer', () => {
    const cases: [string, string | undefined][] = [
      ['StoryData/S001A.json', '/dataList/14/content'],
      ['StoryData/S001A.json', undefined],
      ['Sub Dir/100% real+file #1?.json', '/dataList/3/a b&c=d%'],
      ['技能/人格 01.json', '/dataList/0/名稱'],
    ]
    for (const [path, ptr] of cases) {
      const route = parseHash(fileHref(path, ptr))
      expect(route, path).toEqual(ptr === undefined ? { kind: 'file', path } : { kind: 'file', path, ptr })
    }
  })

  it('encodes every character that would end a segment or the query', () => {
    const href = fileHref('a b/c%d.json', '/x?y#z')
    expect(href).toBe('#/f/a%20b/c%25d.json?u=%2Fx%3Fy%23z')
  })
})

describe('searchHref', () => {
  it('writes q, lang and, for a regular expression, re=1', () => {
    expect(searchHref('從', 'zh', false)).toBe('#/s?q=%E5%BE%9E&lang=zh')
    expect(searchHref('a.+b', 'kr', true)).toBe('#/s?q=a.%2Bb&lang=kr&re=1')
  })

  it('goes back through parseHash', () => {
    for (const q of ['從', 'a b&c=d', '100%', '#?/', ' x ']) {
      for (const re of [false, true]) {
        expect(parseHash(searchHref(q, 'jp', re)), q).toEqual({ kind: 'search', q, lang: 'jp', re })
      }
    }
  })
})

describe('review route', () => {
  it('round-trips odd ids', () => {
    const id = 'note/StoryData/S1.json|/dataList/0/content/1a2b3c4d'
    expect(parseHash(reviewHref('notes', { id, q: '協會 & ?' }))).toEqual({
      kind: 'review',
      tab: 'notes',
      family: '',
      sub: '',
      q: '協會 & ?',
      all: false,
      mode: 'kr',
      jpen: '',
      id,
    })
  })

  it('round-trips every filter', () => {
    const route = parseHash(reviewHref('warnings', { family: 'story-main', sub: 'kana', q: 'a=b', all: true }))
    expect(route).toEqual({ kind: 'review', tab: 'warnings', family: 'story-main', sub: 'kana', q: 'a=b', all: true, mode: 'kr', jpen: '' })
  })

  it('reads #/r as the suggestions tab with no filter', () => {
    expect(parseHash('#/r')).toEqual({ kind: 'review', tab: 'suggestions', family: '', sub: '', q: '', all: false, mode: 'kr', jpen: '' })
    expect(parseHash('#/r/todo?all=1')).toEqual({ kind: 'review', tab: 'todo', family: '', sub: '', q: '', all: true, mode: 'kr', jpen: '' })
  })

  it('reads an unknown tab as home', () => {
    expect(parseHash('#/r/bogus')).toEqual({ kind: 'home' })
    expect(parseHash('#/review')).toEqual({ kind: 'home' })
  })

  it('leaves out what is empty', () => {
    expect(reviewHref('suggestions')).toBe('#/r/suggestions')
    expect(reviewHref('todo', { all: true })).toBe('#/r/todo?all=1')
  })
})

describe('review route modes', () => {
  it('review route keeps mode and jpen', () => {
    expect(parseHash(reviewHref('suggestions', { mode: 'jpen', jpen: 'partial', id: 'x' }))).toMatchObject({
      kind: 'review',
      tab: 'suggestions',
      mode: 'jpen',
      jpen: 'partial',
      id: 'x',
    })
    expect(parseHash('#/r/suggestions')).toMatchObject({ mode: 'kr', jpen: '' })
    expect(parseHash('#/r/suggestions?mode=bogus&jpen=bogus')).toMatchObject({ mode: 'kr', jpen: '' })
    expect(reviewHref('suggestions', { mode: 'kr' })).toBe('#/r/suggestions')
  })
})

describe('update routes', () => {
  const tag = 'zh-TW-20261105.1'

  it('reads #/u as the list of updates', () => {
    expect(parseHash('#/u')).toEqual({ kind: 'updates' })
    expect(updatesHref()).toBe('#/u')
  })

  it('reads the filters of an update and writes them back', () => {
    const hash = `#/u/${tag}?k=new,kr-changed&o=claude&f=Passives.json&copy=1`
    const route = parseHash(hash)
    expect(route).toEqual({ kind: 'update', tag, kinds: ['new', 'kr-changed'], origin: 'claude', file: 'Passives.json', copy: true })
    expect(updateHref(tag, { kinds: ['new', 'kr-changed'], origin: 'claude', file: 'Passives.json', copy: true })).toBe(hash)
  })

  it('has no filter by default, and writes none', () => {
    expect(parseHash(`#/u/${tag}`)).toEqual({ kind: 'update', tag, kinds: [], origin: '', file: '', copy: false })
    expect(updateHref(tag)).toBe(`#/u/${tag}`)
    expect(updateHref(tag, { kinds: [], origin: '', file: '', copy: false })).toBe(`#/u/${tag}`)
  })

  it('encodes the values and reads them back', () => {
    const f = { kinds: ['deleted' as const], origin: '', file: 'StoryData/S 1.json', copy: false }
    const hash = updateHref(tag, f)
    expect(hash).toBe(`#/u/${tag}?k=deleted&f=StoryData%2FS%201.json`)
    expect(parseHash(hash)).toEqual({ kind: 'update', tag, ...f })
  })

  it('reads a tag that is not one as home', () => {
    for (const h of ['#/u/..', '#/u/foo', '#/u/zh-TW-2026.1', '#/u/zh-TW-20261105', '#/u/zh-TW-20261105.1/x', '#/u/']) {
      expect(parseHash(h)).toEqual({ kind: 'home' })
    }
  })

  it('drops the kinds it does not know', () => {
    expect(parseHash(`#/u/${tag}?k=new,bogus,,zh-changed`)).toMatchObject({ kinds: ['new', 'zh-changed'] })
    expect(parseHash(`#/u/${tag}?k=bogus`)).toMatchObject({ kinds: [] })
  })
})
