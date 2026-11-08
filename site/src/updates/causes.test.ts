import { describe, expect, it } from 'vitest'
import { CAUSE_KIND_NAMES, causeKey, entryCauseKinds, groupByCause, groupEntries, type CauseGroup } from './causes'
import type { UpdateEntry, UpdateRecord } from '../types'

function rec(entries: UpdateEntry[], extra: Partial<UpdateRecord> = {}): UpdateRecord {
  return {
    tag: 't', prev_tag: 'p', released_at: '', prev_released_at: '', raw_tag: 'r', prev_raw_tag: 'r',
    rules_commit: '', prev_rules_commit: '',
    counts: { new: 0, kr_changed: 0, deleted: 0, zh_changed: 0 },
    copy_counts: { new: 0, kr_changed: 0, deleted: 0, zh_changed: 0 },
    other_files: [], entries, ...extra,
  }
}
const e = (file: string, ptr: string, kind: UpdateEntry['kind'], extra: Partial<UpdateEntry> = {}): UpdateEntry => ({
  file, field: 'name', ptr, kind, ...extra,
})
const rules = {
  'rewrite:r1': { kind: 'rewrite', id: 'r1', path: 'rules/rewrites/r1.yaml', title: 'R1', commit: 'abc' },
  'requeue:q1': { kind: 'requeue', id: 'q1', path: 'rules/requeues/q1.yaml', title: 'Q1', commit: 'def' },
}

describe('causeKey', () => {
  it('keys rules by kind and id, retranslated by reason, the rest by kind', () => {
    expect(causeKey({ kind: 'rewrite', id: 'r1' })).toBe('rewrite:r1')
    expect(causeKey({ kind: 'adjustment', id: '2026-11-08-x.yaml' })).toBe('adjustment:2026-11-08-x.yaml')
    expect(causeKey({ kind: 'retranslated', reason: 'typo' })).toBe('retranslated:typo')
    expect(causeKey({ kind: 'retranslated' })).toBe('retranslated:')
    expect(causeKey({ kind: 'reuse', batch: 'b' })).toBe('reuse')
    expect(causeKey({ kind: 'unknown', note: 'n' })).toBe('unknown')
  })
})

describe('CAUSE_KIND_NAMES', () => {
  it('names the 12 kinds', () => {
    expect(Object.keys(CAUSE_KIND_NAMES)).toEqual([
      'rewrite', 'adjustment', 'adjustment-expired', 'requeue', 'reuse', 'kr-changed',
      'new', 'deleted', 'late', 'retranslated', 'other', 'unknown',
    ])
    expect(CAUSE_KIND_NAMES['kr-changed']).toBe('KR 改版')
    expect(CAUSE_KIND_NAMES.unknown).toBe('無法判定')
  })
})

describe('groupByCause', () => {
  const a = e('A.json', '/1', 'zh-changed', { causes: [{ kind: 'rewrite', id: 'r1' }, { kind: 'reuse', batch: 'b' }] })
  const b = e('B.json', '/2', 'zh-changed', { causes: [{ kind: 'rewrite', id: 'r1' }] })
  const c = e('A.json', '/3', 'zh-changed', { causes: [{ kind: 'requeue', id: 'q1' }] })
  const d = e('C.json', '/4', 'new', { causes: [{ kind: 'new' }] })
  const f = e('C.json', '/5', 'kr-changed', { causes: [{ kind: 'kr-changed' }] })
  const g = e('C.json', '/6', 'zh-changed', { causes: [{ kind: 'unknown', note: 'x' }] })
  const r = rec([d, a, b, c, f, g], { causes_version: 1, rules })

  it('puts a multi-cause entry in each group', () => {
    const groups = groupByCause(r)
    expect(groups.find((x) => x.key === 'rewrite:r1')!.entries).toEqual([a, b])
    expect(groups.find((x) => x.key === 'reuse')!.entries).toEqual([a])
  })

  it('lists once an entry whose two causes share a key', () => {
    const x = e('A.json', '/9', 'zh-changed', { causes: [{ kind: 'reuse', batch: '1' }, { kind: 'reuse', batch: '2' }] })
    expect(groupByCause(rec([x], { causes_version: 1 }))[0]!.entries).toEqual([x])
  })

  it('sorts rules first by size, then by the fixed kind order', () => {
    expect(groupByCause(r).map((x) => x.key)).toEqual(['rewrite:r1', 'requeue:q1', 'kr-changed', 'new', 'reuse', 'unknown'])
  })

  it('fills the rule fields and counts distinct files', () => {
    const [first, second, kr] = groupByCause(r) as [CauseGroup, CauseGroup, CauseGroup]
    expect(first).toMatchObject({ kind: 'rewrite', id: 'r1', title: 'R1', path: 'rules/rewrites/r1.yaml', commit: 'abc', isRule: true, files: 2 })
    expect(first.legacy).toBeUndefined()
    expect(second).toMatchObject({ id: 'q1', files: 1 })
    expect(kr).toMatchObject({ kind: 'kr-changed', isRule: false })
    expect(kr.title).toBeUndefined()
  })

  it('groups retranslated by reason', () => {
    const x1 = e('A.json', '/1', 'zh-changed', { causes: [{ kind: 'retranslated', reason: 'typo' }] })
    const x2 = e('A.json', '/2', 'zh-changed', { causes: [{ kind: 'retranslated', reason: 'tone' }] })
    const x3 = e('A.json', '/3', 'zh-changed', { causes: [{ kind: 'retranslated' }] })
    const x4 = e('B.json', '/4', 'zh-changed', { causes: [{ kind: 'retranslated', reason: 'typo' }] })
    const groups = groupByCause(rec([x1, x2, x3, x4], { causes_version: 1 }))
    expect(groups.map((x) => [x.key, x.entries.length, x.reason])).toEqual([
      ['retranslated:typo', 2, 'typo'],
      ['retranslated:tone', 1, 'tone'],
      ['retranslated:', 1, undefined],
    ])
  })

  it('falls back to the old cause and the kind for a record without causes', () => {
    const l = rec([
      e('A.json', '/1', 'zh-changed', { cause: 'adjustment' }),
      e('A.json', '/2', 'zh-changed', { cause: 'late' }),
      e('A.json', '/3', 'zh-changed', { cause: 'retranslated' }),
      e('B.json', '/4', 'zh-changed'),
      e('B.json', '/5', 'new'),
      e('B.json', '/6', 'kr-changed'),
      e('B.json', '/7', 'deleted'),
    ])
    const groups = groupByCause(l)
    expect(groups.map((x) => x.key)).toEqual(['kr-changed', 'new', 'deleted', 'late', 'retranslated', 'adjustment', 'other'])
    expect(groups.every((x) => x.legacy === true && !x.isRule)).toBe(true)
  })

  it('returns nothing for a record with no entries', () => {
    expect(groupByCause(rec([]))).toEqual([])
  })
})

describe('legacy kinds', () => {
  const entries = [
    e('A', '/1', 'zh-changed', { cause: 'adjustment' }),
    e('A', '/2', 'zh-changed', { cause: 'late' }),
    e('A', '/3', 'zh-changed', { cause: 'retranslated' }),
    e('A', '/4', 'zh-changed'),
    e('A', '/5', 'new'),
    e('A', '/6', 'kr-changed'),
    e('A', '/7', 'deleted'),
  ]
  it('are the keys of the legacy groups, so the cause filter and the groups agree', () => {
    const record = rec(entries)
    for (const g of groupByCause(record)) {
      for (const x of g.entries) expect(entryCauseKinds(x)).toEqual([g.key])
    }
    expect(entries.map((x) => entryCauseKinds(x)[0])).toEqual(['adjustment', 'late', 'retranslated', 'other', 'new', 'kr-changed', 'deleted'])
  })
  it('groupEntries groups only the given entries, in the same order', () => {
    const record = rec(entries)
    const some = entries.filter((x) => x.kind !== 'new')
    expect(groupEntries(record, some).map((g) => g.key)).toEqual(groupByCause(record).map((g) => g.key).filter((k) => k !== 'new'))
  })
})
