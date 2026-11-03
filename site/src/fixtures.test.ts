// The fixtures are written by hand to the layout of the site spec, section 5,
// and are what the tests of the viewer read. This keeps them true to types.ts
// and to themselves: the index, the files and the search corpus agree.
import { describe, expect, it } from 'vitest'
import type { FileData, Index, Meta, Origin, Unit } from './types'

const modules = import.meta.glob('../fixtures/data/**/*.json', { eager: true, import: 'default' })
const at = (rel: string): unknown => modules[`../fixtures/data/${rel}`]

const meta = at('meta.json') as Meta
const index = at('index.json') as Index
const keys = at('search/keys.json') as [number, string][]
const corpus = Object.fromEntries(
  (['kr', 'jp', 'en', 'zh'] as const).map((lang) => [lang, at(`search/${lang}.json`) as string[]]),
)
const files = index.files.map((f) => at(`files/${f.path}`) as FileData)

const ORIGINS: Origin[] = ['paratranz', 'claude', 'human', 'copy', 'fallback', 'stand-in']

describe('fixtures', () => {
  it('has the keys of the Go types, and no others', () => {
    expect(Object.keys(meta)).toEqual(['format', 'raw_tag', 'rules_commit', 'tw_commit', 'release', 'langs', 'files', 'units'])
    expect(Object.keys(index)).toEqual(['families', 'files'])
    for (const fam of index.families) expect(Object.keys(fam)).toEqual(['id', 'files'])
    for (const f of index.files) {
      expect(Object.keys(f)).toEqual(['path', 'family', 'label', 'speakers', 'units', 'origins', 'missing'])
      expect(Object.keys(f.missing)).toEqual(['en', 'jp'])
    }
    for (const f of files) {
      expect(Object.keys(f)).toEqual(['path', 'family', 'units'])
      for (const u of f.units) {
        const keysOf = Object.keys(u)
        expect(keysOf.filter((k) => !['p', 'f', 'i', 'kr', 'jp', 'en', 'zh', 'o', 'fl'].includes(k))).toEqual([])
        for (const k of ['p', 'f', 'i', 'kr', 'zh', 'o']) expect(keysOf).toContain(k)
        expect(ORIGINS).toContain(u.o)
      }
    }
  })

  it('counts its files and units in the meta', () => {
    expect(meta.format).toBe(1)
    expect(meta.langs).toEqual(['kr', 'jp', 'en', 'zh'])
    expect(meta.files).toBe(index.files.length)
    expect(meta.units).toBe(files.reduce((n, f) => n + f.units.length, 0))
  })

  it('puts every file in one family, in order', () => {
    expect(index.families.map((f) => f.files).flat()).toEqual(index.files.map((_, n) => n))
    expect(index.families.length).toBe(2)
    for (const fam of index.families) {
      for (const n of fam.files) expect(index.files[n]?.family).toBe(fam.id)
    }
  })

  it('has a file for each entry of the index, as the index says', () => {
    index.files.forEach((sum, n) => {
      const file = files[n]
      expect(file?.path).toBe(sum.path)
      expect(file?.family).toBe(sum.family)
      const us = file?.units ?? []
      expect(us).toHaveLength(sum.units)
      const origins: Partial<Record<Origin, number>> = {}
      for (const u of us) origins[u.o] = (origins[u.o] ?? 0) + 1
      expect(origins).toEqual(sum.origins)
      expect(sum.missing).toEqual({
        en: us.filter((u) => u.en === undefined).length,
        jp: us.filter((u) => u.jp === undefined).length,
      })
      const tellers = [...new Set(us.filter((u) => u.f === 'teller' && u.zh !== '').map((u) => u.zh))]
      expect(sum.speakers).toEqual(tellers)
      const place = us.find((u) => u.f === 'place' && u.zh !== '')
      expect(sum.label).toBe(sum.path.startsWith('StoryData/') ? (place?.zh ?? '') : '')
    })
  })

  it('has the searchable units, copy excepted, in the corpus', () => {
    const searchable: [number, Unit][] = files.flatMap((f, n) => f.units.filter((u) => u.o !== 'copy').map((u): [number, Unit] => [n, u]))
    expect(keys).toEqual(searchable.map(([n, u]) => [n, u.p]))
    expect(corpus.kr).toEqual(searchable.map(([, u]) => u.kr))
    expect(corpus.jp).toEqual(searchable.map(([, u]) => u.jp ?? ''))
    expect(corpus.en).toEqual(searchable.map(([, u]) => u.en ?? ''))
    expect(corpus.zh).toEqual(searchable.map(([, u]) => u.zh))
  })

  it('covers what the viewer has to show', () => {
    const all = files.flatMap((f) => f.units)
    expect(files.some((f) => f.units.length === 0)).toBe(true)
    expect(index.files.some((f) => f.path.startsWith('StoryData/'))).toBe(true)
    expect(all.some((u) => u.jp === undefined)).toBe(true)
    expect(all.some((u) => u.en === undefined)).toBe(true)
    expect(all.some((u) => (u.fl ?? []).length > 0)).toBe(true)
    expect(all.some((u) => u.kr.includes('<ruby='))).toBe(true)
    expect(all.some((u) => u.kr.startsWith('<') && u.kr.endsWith('>') && !u.kr.includes('</'))).toBe(true)
    expect(new Set(all.map((u) => u.o))).toEqual(new Set(ORIGINS))
    expect(new Set(all.filter((u) => u.i >= 0).map((u) => u.i)).size).toBeGreaterThanOrEqual(3)
  })
})
