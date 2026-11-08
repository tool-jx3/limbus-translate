import { describe, expect, it } from 'vitest'
import { countKinds, entryFiles, entryOrigins, filterEntries, type UpdateFilter } from './filter'
import type { UpdateEntry } from '../types'

const none: UpdateFilter = { kinds: [], origin: '', file: '', copy: true, cause: '' }

const entries: UpdateEntry[] = [
  { file: 'A.json', field: 'name', ptr: '/0/name', kind: 'new', new_kr: 'a', new_zh: 'A', new_origin: 'claude' },
  { file: 'B.json', field: 'desc', ptr: '/1/desc', kind: 'kr-changed', old_kr: 'b', new_kr: 'b2', new_origin: 'paratranz', copy: true },
  { file: 'A.json', field: 'name', ptr: '/2/name', kind: 'deleted', old_kr: 'c', old_zh: 'C', old_origin: 'human' },
  { file: 'C.json', field: 'name', ptr: '/3/name', kind: 'zh-changed', old_zh: 'd', new_zh: 'D', new_origin: 'claude', cause: 'late' },
  { file: 'B.json', field: 'name', ptr: '/4/name', kind: 'new', new_origin: 'claude', copy: true },
]

const ptrs = (l: UpdateEntry[]) => l.map((e) => e.ptr)

describe('filterEntries', () => {
  it('keeps everything with no filter', () => {
    expect(filterEntries(entries, none)).toEqual(entries)
  })

  it('filters by kind', () => {
    expect(ptrs(filterEntries(entries, { ...none, kinds: ['new'] }))).toEqual(['/0/name', '/4/name'])
    expect(ptrs(filterEntries(entries, { ...none, kinds: ['new', 'deleted'] }))).toEqual(['/0/name', '/2/name', '/4/name'])
  })

  it('filters by origin: new_origin, old_origin for a deleted entry', () => {
    expect(ptrs(filterEntries(entries, { ...none, origin: 'claude' }))).toEqual(['/0/name', '/3/name', '/4/name'])
    expect(ptrs(filterEntries(entries, { ...none, origin: 'human' }))).toEqual(['/2/name'])
  })

  it('filters by file', () => {
    expect(ptrs(filterEntries(entries, { ...none, file: 'A.json' }))).toEqual(['/0/name', '/2/name'])
  })

  it('hides the entries that need no translation when copy is false', () => {
    expect(ptrs(filterEntries(entries, { ...none, copy: false }))).toEqual(['/0/name', '/2/name', '/3/name'])
  })

  it('combines the filters', () => {
    expect(ptrs(filterEntries(entries, { ...none, kinds: ['new'], origin: 'claude', file: 'B.json' }))).toEqual(['/4/name'])
    expect(filterEntries(entries, { ...none, kinds: ['new'], origin: 'claude', file: 'B.json', copy: false })).toEqual([])
  })
})

describe('countKinds', () => {
  it('counts every entry when copy is true', () => {
    expect(countKinds(entries, true)).toEqual({ new: 2, kr_changed: 1, deleted: 1, zh_changed: 1 })
  })

  it('leaves the copies out when copy is false', () => {
    expect(countKinds(entries, false)).toEqual({ new: 1, kr_changed: 0, deleted: 1, zh_changed: 1 })
  })
})

describe('entryFiles and entryOrigins', () => {
  it('lists the files in order of appearance, once', () => {
    expect(entryFiles(entries)).toEqual(['A.json', 'B.json', 'C.json'])
  })

  it('lists the origins sorted, once, deleted ones by old_origin', () => {
    expect(entryOrigins(entries)).toEqual(['claude', 'human', 'paratranz'])
  })
})

describe('filterEntries by cause', () => {
  const withCauses: UpdateEntry[] = [
    { file: 'A.json', field: 'n', ptr: '/0', kind: 'zh-changed', causes: [{ kind: 'rewrite', id: 'x' }, { kind: 'other' }] },
    { file: 'A.json', field: 'n', ptr: '/1', kind: 'zh-changed', causes: [{ kind: 'other' }] },
    { file: 'A.json', field: 'n', ptr: '/2', kind: 'new', causes: [{ kind: 'new' }] },
  ]
  it('keeps an entry when any of its causes has the kind', () => {
    expect(ptrs(filterEntries(withCauses, { ...none, cause: 'rewrite' }))).toEqual(['/0'])
    expect(ptrs(filterEntries(withCauses, { ...none, cause: 'other' }))).toEqual(['/0', '/1'])
  })
  it('uses the legacy key for an entry with no causes', () => {
    expect(ptrs(filterEntries(entries, { ...none, cause: 'late' }))).toEqual(['/3/name'])
    expect(ptrs(filterEntries(entries, { ...none, cause: 'deleted' }))).toEqual(['/2/name'])
  })
})
