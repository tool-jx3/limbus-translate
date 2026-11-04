import { describe, expect, it } from 'vitest'
import { fixture } from '../testdata'
import type { NoteItem, SuggestionItem, WarningItem } from '../types'
import { filterNotes, filterSuggestions, filterWarnings, jpenCounts, sortSuggestions } from './filter'

const suggestions = fixture<{ items: SuggestionItem[] }>('review/suggestions.json').items
const warnings = fixture<{ items: WarningItem[] }>('review/warnings.json').items
const notes = fixture<{ items: NoteItem[] }>('review/notes.json').items
const none = { family: '', sub: '', q: '', all: false }
const ids = (items: { id: string }[]) => items.map((i) => i.id)

describe('filterSuggestions', () => {
  it('hides the declined unless all', () => {
    expect(ids(filterSuggestions(suggestions, none))).toHaveLength(3)
    expect(ids(filterSuggestions(suggestions, { ...none, all: true }))).toHaveLength(4)
  })

  it('filters by family and kind', () => {
    expect(ids(filterSuggestions(suggestions, { ...none, family: 'ally' }))).toEqual(['inconsistent/ally/name/c3a1192c'])
    expect(ids(filterSuggestions(suggestions, { ...none, sub: 'speaker' }))).toEqual(['speaker/베아트리체'])
  })

  it('filters by text in the id, the file name, KR, TW and the note', () => {
    const q = (s: string) => ids(filterSuggestions(suggestions, { ...none, q: s }))
    expect(q('c3a1192c')).toEqual(['inconsistent/ally/name/c3a1192c'])
    expect(q('skills_personality-99')).toEqual(['inconsistent/ally/name/c3a1192c'])
    expect(q('시험용')).toEqual(['inconsistent/ally/name/c3a1192c'])
    expect(q('試驗斬擊')).toEqual(['inconsistent/ally/name/c3a1192c'])
    expect(q('內心獨白')).toEqual(['note/StoryData/S9999.json|/dataList/2/content/5991de5a'])
    expect(q('no such text')).toEqual([])
  })

  it('keeps the item the id names, confirmed or not', () => {
    const declined = 'note/StoryData/S9999.json|/dataList/3/content/765c0235'
    expect(ids(filterSuggestions(suggestions, none, declined))).toContain(declined)
  })
})

describe('sortSuggestions', () => {
  it('puts the ones with more translations first, and leaves the rest in order', () => {
    const sorted = sortSuggestions(suggestions, 'variants')
    expect(sorted[0]?.id).toBe('inconsistent/ally/name/c3a1192c')
    expect(ids(sortSuggestions(suggestions, 'units'))).toEqual(ids(suggestions))
  })
})

describe('filterWarnings', () => {
  it('hides the accepted unless all', () => {
    expect(ids(filterWarnings(warnings, none))).toEqual(['warning/kana/1315241d/2cc3aa54'])
    expect(ids(filterWarnings(warnings, { ...none, all: true }))).toHaveLength(2)
  })

  it('filters by family, code and text', () => {
    const all = { ...none, all: true }
    expect(ids(filterWarnings(warnings, { ...all, family: 'story-main' }))).toEqual(['warning/halfwidth-punct/145460cb/1a330b10'])
    expect(ids(filterWarnings(warnings, { ...all, sub: 'kana' }))).toEqual(['warning/kana/1315241d/2cc3aa54'])
    expect(ids(filterWarnings(warnings, { ...all, q: 'S9999' }))).toEqual(['warning/halfwidth-punct/145460cb/1a330b10'])
    expect(ids(filterWarnings(warnings, { ...all, q: '以上の時' }))).toEqual(['warning/kana/1315241d/2cc3aa54'])
  })
})

describe('filterNotes', () => {
  it('hides the declined unless all', () => {
    expect(notes).toHaveLength(2)
    expect(filterNotes(notes, none)).toHaveLength(1)
    expect(filterNotes(notes, { ...none, all: true })).toHaveLength(2)
  })

  it('filters by family and text', () => {
    const all = { ...none, all: true }
    expect(filterNotes(notes, { ...all, family: 'ally' })).toHaveLength(0)
    expect(filterNotes(notes, { ...all, q: '角括號' })).toHaveLength(1)
    expect(filterNotes(notes, { ...all, q: '[burn]' })).toHaveLength(1)
  })
})

describe('the JP/EN class filter', () => {
  const inc = suggestions.find((s) => s.kind === 'inconsistent')!
  const mk = (id: string, cls: 'same' | 'explained' | 'partial', declined = false): SuggestionItem => ({
    ...inc,
    id,
    jpen_class: cls,
    ...(declined ? { declined: { reason: 'r', date: '2026-11-04' } } : { declined: undefined }),
  })
  const items: SuggestionItem[] = [mk('a', 'same'), mk('b', 'explained'), mk('c', 'partial'), mk('d', 'same', true), ...suggestions.filter((s) => s.kind !== 'inconsistent')]

  it('keeps only the inconsistent groups of the class, and every other kind', () => {
    const got = ids(filterSuggestions(items, { ...none, jpen: 'same' }))
    expect(got.filter((i) => i.length === 1)).toEqual(['a'])
    expect(got.filter((i) => i.length > 1)).toEqual(ids(suggestions.filter((s) => s.kind !== 'inconsistent' && !s.declined)))
  })

  it('jpen filter counts follow show-confirmed', () => {
    expect(jpenCounts(items, none)).toEqual({ explained: 1, partial: 1, same: 1 })
    expect(jpenCounts(items, { ...none, all: true })).toEqual({ explained: 1, partial: 1, same: 2 })
  })
})
