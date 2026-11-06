import { describe, expect, it } from 'vitest'
import type { FileData, PreviewFileWire, PreviewUnitWire } from '../types'
import { expandPreviewFile } from './decode'

const file = (zhs: string[]): FileData => ({
  path: 'A.json',
  family: 'other',
  units: zhs.map((zh, i) => ({ p: `/dataList/${i}/content`, f: 'content', kr: `kr ${i}`, zh })) as unknown as FileData['units'],
})

const unit = (over: Partial<PreviewUnitWire> = {}): PreviewUnitWire => ({ sp: '', g: [], t: [], k: [], it: { reason: 'new' }, c: [], ...over })

const wire = (units: (PreviewUnitWire | null)[], rd?: PreviewFileWire['rd']): PreviewFileWire => ({
  family: 'other',
  glossary_check: '',
  terms: [],
  keywords: {},
  ...(rd ? { rd } : {}),
  units,
})

const RD = { origin: 'claude', rules: 'abc1234', since: 'raw-1' }
const one = (u: PreviewUnitWire | null, zh = 'ZH', rd?: PreviewFileWire['rd']) => expandPreviewFile(wire([u], rd), file([zh])).units[0]

describe('expandPreviewFile', () => {
  it('keeps a copy null', () => {
    expect(one(null)).toBeNull()
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
    expect(() => one(unit({ m: 'p' }))).toThrow(/not two characters/)
    expect(() => one(unit({ m: 'px' }))).toThrow(/no alignment method/)
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
    expect(() => one(unit({ rec: {} }))).toThrow(/no defaults/)
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
    expect(() => one(unit({ chk: { verdict: 'accept' } }), 'ZH', RD)).toThrow(/no record/)
  })

  it('refuses a preview of another length than the file', () => {
    expect(() => expandPreviewFile(wire([null, null]), file(['a']))).toThrow(/1 units in the file, 2 units in the preview/)
  })
  it('passes the rest through', () => {
    const u = one(unit({ sp: 'S', g: [1], t: [2], k: ['K'], c: [[0], [1, 'x']], nt: 'n', it: { reason: 'new', hint: 'h', refs: { a: 'b' } } }))!
    expect(u).toMatchObject({ sp: 'S', g: [1], t: [2], k: ['K'], c: [[0], [1, 'x']], nt: 'n', it: { reason: 'new', hint: 'h', refs: { a: 'b' } } })
  })
})
