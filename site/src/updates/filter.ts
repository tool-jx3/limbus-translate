import type { UpdateCounts, UpdateEntry, UpdateKind } from '../types'

/** Which changes of an update are shown. */
export interface UpdateFilter {
  /** The kinds shown; empty is all of them. */
  kinds: UpdateKind[]
  /** The origin of the new TW text (of the old one, for a deleted unit); '' is all. */
  origin: string
  /** The file; '' is all. */
  file: string
  /** false hides the entries that need no translation (copy). */
  copy: boolean
  /** The kind of cause (an entry matches if any of its causes has it); '' is all. */
  cause: string
}

/** The origin an entry is filtered by: the new one, or the old one where the unit is deleted. */
function originOf(e: UpdateEntry): string {
  return (e.kind === 'deleted' ? e.old_origin : e.new_origin) ?? ''
}

/** The kinds of the causes of an entry; with none, the key the legacy grouping uses (see groupByCause). */
export function entryCauseKinds(e: UpdateEntry): string[] {
  if (e.causes && e.causes.length > 0) return e.causes.map((c) => c.kind)
  return [e.kind === 'zh-changed' ? (e.cause ?? 'other') : e.kind]
}

export function filterEntries(entries: UpdateEntry[], f: UpdateFilter): UpdateEntry[] {
  return entries.filter(
    (e) =>
      (f.kinds.length === 0 || f.kinds.includes(e.kind)) &&
      (f.origin === '' || originOf(e) === f.origin) &&
      (f.file === '' || e.file === f.file) &&
      (f.cause === '' || entryCauseKinds(e).includes(f.cause)) &&
      (f.copy || !e.copy),
  )
}

/** The number of entries of each kind; with copy false, the ones that need no translation are left out. */
export function countKinds(entries: UpdateEntry[], copy: boolean): UpdateCounts {
  const out: UpdateCounts = { new: 0, kr_changed: 0, deleted: 0, zh_changed: 0 }
  for (const e of entries) {
    if (!copy && e.copy) continue
    if (e.kind === 'new') out.new++
    else if (e.kind === 'kr-changed') out.kr_changed++
    else if (e.kind === 'deleted') out.deleted++
    else out.zh_changed++
  }
  return out
}

/** The files of the entries, in order of appearance, each once. */
export function entryFiles(entries: UpdateEntry[]): string[] {
  return [...new Set(entries.map((e) => e.file))]
}

/** The origins of the entries, sorted, each once. */
export function entryOrigins(entries: UpdateEntry[]): string[] {
  return [...new Set(entries.map(originOf).filter((o) => o !== ''))].sort()
}
