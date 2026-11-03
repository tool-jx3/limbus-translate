import type { NoteItem, SuggestionItem, WarningItem } from '../types'

/** What the address says of the filters of a page of the review. */
export interface ReviewFilter {
  family: string
  /** The kind of a suggestion, the code of a warning. */
  sub: string
  q: string
  /** Show the confirmed items too. */
  all: boolean
}

/** Whether any of the texts has the needle in it (any case); an empty needle is in all. */
function has(needle: string, texts: (string | undefined)[]): boolean {
  if (needle === '') return true
  return texts.some((t) => t !== undefined && t.toLowerCase().includes(needle))
}

const fileOf = (key: string | undefined) => key?.slice(0, key.indexOf('|'))

/** Whether the confirmed item stays out: it does, unless all is on or the address names it. */
function hidden(confirmed: unknown, f: ReviewFilter, id: string, target: string | undefined): boolean {
  return confirmed !== undefined && !f.all && id !== target
}

export function filterSuggestions(items: SuggestionItem[], f: ReviewFilter, target?: string): SuggestionItem[] {
  const needle = f.q.trim().toLowerCase()
  return items.filter(
    (s) =>
      !hidden(s.declined, f, s.id, target) &&
      (f.family === '' || s.family === f.family) &&
      (f.sub === '' || s.kind === f.sub) &&
      has(needle, [
        s.id,
        s.title,
        s.kr,
        s.jp,
        s.en,
        ...(s.variants ?? []).flatMap((v) => [v.zh, ...v.keys]),
        ...s.evidence.flatMap((e) => [e.key, fileOf(e.key), e.kr, e.zh, e.note]),
      ]),
  )
}

export function filterWarnings(items: WarningItem[], f: ReviewFilter, target?: string): WarningItem[] {
  const needle = f.q.trim().toLowerCase()
  return items.filter(
    (w) =>
      !hidden(w.accepted, f, w.id, target) &&
      (f.family === '' || w.families.includes(f.family)) &&
      (f.sub === '' || w.code === f.sub) &&
      has(needle, [w.id, w.kr, w.zh, ...w.keys]),
  )
}

export function filterNotes(items: NoteItem[], f: ReviewFilter, target?: string): NoteItem[] {
  const needle = f.q.trim().toLowerCase()
  return items.filter(
    (n) =>
      !hidden(n.declined, f, n.id, target) &&
      (f.family === '' || n.family === f.family) &&
      has(needle, [n.id, n.key, n.note, n.kr, n.zh]),
  )
}

/** The units of a suggestion: those of its translations, or its rows of evidence. */
export function suggestionUnits(s: SuggestionItem): number {
  return s.variants ? s.variants.reduce((sum, v) => sum + v.count, 0) : s.evidence.length
}

/** The order of the list: as `lct suggest` has it (units), or the ones with more translations first (variants; a stable sort). */
export function sortSuggestions(items: SuggestionItem[], by: 'units' | 'variants'): SuggestionItem[] {
  if (by === 'units') return items
  return [...items].sort((a, b) => (b.variants?.length ?? 0) - (a.variants?.length ?? 0))
}
