import { describe, expect, it } from 'vitest'
import type { FileData, PreviewFileWire, PreviewShared, PreviewUnitWire } from '../types'
import { DataError } from '../data'
import { expandPreviewFile } from './decode'

const file = (zhs: string[]): FileData => ({
  path: 'A.json',
  family: 'other',
  units: zhs.map((zh, i) => ({ p: `/dataList/${i}/content`, f: 'content', kr: `kr ${i}`, zh })) as unknown as FileData['units'],
})

const unit = (over: Partial<PreviewUnitWire> = {}): PreviewUnitWire => ({ sp: '', g: [], t: [], k: [], n: [], it: { reason: 'new' }, c: [], ...over })

const wire = (units: (PreviewUnitWire | null)[], rd?: PreviewFileWire['rd']): PreviewFileWire => ({
  family: 'other',
  glossary_check: '',
  names: [],
  terms: [],
  keywords: {},
  ...(rd ? { rd } : {}),
  units,
})

const RD = { origin: 'claude', rules: 'abc1234', since: 'raw-1' }
const one = (u: PreviewUnitWire | null, zh = 'ZH', rd?: PreviewFileWire['rd']) => expandPreviewFile(wire([u], rd), file([zh])).units[0]

/** The error `fn` throws must be a format error: the page then says the data was updated, not that the network failed. */
const formatError = (fn: () => unknown, message: RegExp) => {
  let error: unknown
  try {
    fn()
  } catch (e) {
    error = e
  }
  expect(error).toBeInstanceOf(DataError)
  expect((error as DataError).kind).toBe('format')
  expect((error as DataError).message).toMatch(message)
}

describe('expandPreviewFile', () => {
  it('keeps a copy null', () => {
    expect(one(null)).toBeNull()
  })

  it('reads a preview written before the names were added: no names and no n are none', () => {
    const older = wire([unit()])
    delete older.names
    delete older.units[0]!.n
    const pf = expandPreviewFile(older, file(['ZH']))
    expect(pf.names).toEqual([])
    expect(pf.units[0]!.n).toEqual([])
  })

  it('puts back an empty keep', () => {
    expect(one(unit())!.keep).toEqual({ ids: [], placeholders: [], tags: [], dante_wrapper: false })
    expect(Object.keys(one(unit())!.keep)).toEqual(['ids', 'placeholders', 'tags', 'dante_wrapper'])
  })
  it('keeps a keep that is there', () => {
    const keep = { ids: ['1'], placeholders: [], tags: ['<b>'], dante_wrapper: true }
    expect(one(unit({ keep }))!.keep).toEqual(keep)
  })

  it('puts back a state of zeros, and fills what a state leaves out', () => {
    expect(one(unit())!.st).toEqual({ pending: false, needs_review: false, frozen: false })
    expect(one(unit({ st: { pending: true, attempts: 3, last_error: 'x' } }))!.st).toEqual({
      pending: true,
      needs_review: false,
      frozen: false,
      attempts: 3,
      last_error: 'x',
    })
    expect(one(unit({ st: { frozen: true, leased_to: 'b1' } }))!.st).toEqual({ pending: false, needs_review: false, frozen: true, leased_to: 'b1' })
  })

  it('takes prev_zh from the file where pz is 1, and leaves a prev_zh of its own', () => {
    expect(one(unit({ pz: 1 }), '舊譯')!.it.prev_zh).toBe('舊譯')
    expect(one(unit({ it: { reason: 'kr-changed', prev_zh: '另一個' } }), '舊譯')!.it.prev_zh).toBe('另一個')
    expect(one(unit())!.it.prev_zh).toBeUndefined()
  })

  it('reads the methods of EN and JP from m', () => {
    const em = (m?: string) => {
      const u = one(unit(m === undefined ? {} : { m }))!
      return [u.em, u.jm]
    }
    expect(em()).toEqual([undefined, undefined])
    expect(em('p-')).toEqual(['pointer', undefined])
    expect(em('-i')).toEqual([undefined, 'id'])
    expect(em('sp')).toEqual(['sequence', 'pointer'])
    expect(em('ii')).toEqual(['id', 'id'])
  })
  it('refuses a bad m', () => {
    formatError(() => one(unit({ m: 'p' })), /not two characters/)
    formatError(() => one(unit({ m: 'px' })), /no alignment method/)
    formatError(() => one(unit({ m: 'xp' })), /no alignment method/)
    expect(() => one(unit({ m: '--' }))).not.toThrow()
  })

  it('has no record and no check for a unit without a record', () => {
    const u = one(unit(), 'ZH', RD)!
    expect(u.rec).toBeUndefined()
    expect(u.chk).toBeUndefined()
  })
  it('puts the defaults of the file back into a record', () => {
    const u = one(unit({ rec: {} }), 'ZH', RD)!
    expect(u.rec).toEqual({ origin: 'claude', rules: 'abc1234', since: 'raw-1', stale: false })
  })
  it('keeps the fields of a record that are there', () => {
    const u = one(unit({ rec: { origin: 'paratranz', rules: 'zzz', since: 'raw-0', model: 'm', batch: 'b', flags: ['f'], stale: true } }), 'ZH', RD)!
    expect(u.rec).toEqual({ origin: 'paratranz', rules: 'zzz', since: 'raw-0', model: 'm', batch: 'b', flags: ['f'], stale: true })
  })
  it('leaves out rules and since that are empty, as the record has none', () => {
    const u = one(unit({ rec: { rules: '', since: '' } }), 'ZH', RD)!
    expect(u.rec).toEqual({ origin: 'claude', stale: false })
    expect(one(unit({ rec: {} }), 'ZH', { origin: 'human', rules: '', since: '' })!.rec).toEqual({ origin: 'human', stale: false })
  })
  it('refuses a record with no defaults', () => {
    formatError(() => one(unit({ rec: {} })), /no defaults/)
  })

  it('accepts the file text where a record has no check', () => {
    expect(one(unit({ rec: {} }), '譯文', RD)!.chk).toEqual({ verdict: 'accept', zh: '譯文', issues: [] })
  })
  it('puts back the zh and the issues of a check', () => {
    const issue = { code: 'hangul', severity: 'hard', class: 'content', message: 'm' }
    expect(one(unit({ rec: {}, chk: { verdict: 'reject', issues: [issue] } }), '譯文', RD)!.chk).toEqual({ verdict: 'reject', zh: '譯文', issues: [issue] })
    expect(one(unit({ rec: {}, chk: { verdict: 'warn', zh: '檢查的字', issues: [issue] } }), '譯文', RD)!.chk).toEqual({
      verdict: 'warn',
      zh: '檢查的字',
      issues: [issue],
    })
    expect(one(unit({ rec: {}, chk: { verdict: 'accept', zh: '' } }), '譯文', RD)!.chk).toEqual({ verdict: 'accept', zh: '', issues: [] })
  })
  it('refuses a check with no record', () => {
    formatError(() => one(unit({ chk: { verdict: 'accept' } }), 'ZH', RD), /no record/)
  })

  it('refuses a preview of another length than the file', () => {
    formatError(() => expandPreviewFile(wire([null, null]), file(['a'])), /1 units in the file, 2 units in the preview/)
  })
  it('passes the rest through', () => {
    const u = one(unit({ sp: 'S', g: [1], t: [2], k: ['K'], c: [[0], [1, 'x']], nt: 'n', it: { reason: 'new', hint: 'h', refs: { a: 'b' } } }))!
    expect(u).toMatchObject({ sp: 'S', g: [1], t: [2], k: ['K'], c: [[0], [1, 'x']], nt: 'n', it: { reason: 'new', hint: 'h', refs: { a: 'b' } } })
  })
})

// fixtures/preview/expanded/<path> is what Go makes straight from the Previewer
// (TestPreviewFixturesUpToDate), never by decoding: the decoder must give it back.
const fixtures = import.meta.glob('../../fixtures/preview/**/*.json', { eager: true, query: '?raw', import: 'default' }) as Record<string, string>
const PREFIX = '../../fixtures/preview/'
const fixture = <T>(rel: string): T => {
  const text = fixtures[PREFIX + rel]
  if (text === undefined) throw new Error(`no fixture ${rel}`)
  return JSON.parse(text) as T
}
const expandedPaths = Object.keys(fixtures)
  .filter((k) => k.startsWith(`${PREFIX}expanded/`))
  .map((k) => k.slice(`${PREFIX}expanded/`.length))

describe('expandPreviewFile against the golden files of Go', () => {
  it('has a golden file for every file of the pack, and nothing else', () => {
    const files = Object.keys(fixtures)
      .filter((k) => k.startsWith(`${PREFIX}preview/`) && k !== `${PREFIX}preview.json`)
      .map((k) => k.slice(`${PREFIX}preview/`.length))
    expect(files.length).toBeGreaterThan(0)
    expect([...expandedPaths].sort()).toEqual(files.sort())
  })
  for (const path of expandedPaths) {
    it(`${path} expands to the golden file`, () => {
      const got = expandPreviewFile(fixture<PreviewFileWire>(`preview/${path}`), fixture<FileData>(`files/${path}`))
      // The golden file names what the compact one numbers: the guides by path and the
      // glossary entries and names by content, as the batches of the Previewer have them.
      const guides = fixture<PreviewShared>('preview.json').guides
      const { terms, names, ...rest } = got
      const named = {
        ...rest,
        units: got.units.map((u) =>
          u && { ...u, g: u.g.map((n) => guides[n]?.path), t: u.t.map((n) => terms[n]), n: u.n.map((n) => names[n]) },
        ),
      }
      expect(named).toEqual(fixture<unknown>(`expanded/${path}`))
    })
  }
})
