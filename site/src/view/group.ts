import type { Settings } from '../settings'
import type { Unit } from '../types'

/** One element of dataList, or one string outside it: what a script shows as a passage. */
export type Entry = {
  /** The index of the dataList element, or -1. */
  i: number
  /** The units that are not the place, the teller or the title. */
  units: Unit[]
  place?: Unit
  teller?: Unit
  title?: Unit
  /** The KR of the place differs from that of the last entry that had a place. */
  placeChanged: boolean
}

const SLOTS = ['place', 'teller', 'title'] as const

/**
 * The units in the order of the file, as entries: neighbours of one dataList
 * element make one entry; each string outside dataList (i == -1) is its own.
 */
export function groupEntries(units: Unit[]): Entry[] {
  const out: Entry[] = []
  let cur: Entry | undefined
  let lastPlace: string | undefined
  for (const u of units) {
    if (cur === undefined || u.i < 0 || u.i !== cur.i) {
      cur = { i: u.i, units: [], placeChanged: false }
      out.push(cur)
    }
    const slot = SLOTS.find((f) => f === u.f)
    // A second place, teller or title of one element is a line like any other: it is not lost.
    if (slot === undefined || cur[slot] !== undefined) {
      cur.units.push(u)
      continue
    }
    cur[slot] = u
    if (slot === 'place') {
      cur.placeChanged = u.kr !== lastPlace
      lastPlace = u.kr
    }
  }
  return out
}

/** Whether the reader wants to see the unit: the origin of its TW text is ticked. Copy units are shown iff origins.copy. */
export function isVisible(u: Unit, s: Settings): boolean {
  return s.origins[u.o]
}

/** The units whose origin is ticked, in their order. */
export function visibleUnits(units: Unit[], s: Settings): Unit[] {
  return units.filter((u) => isVisible(u, s))
}

/** Whether a file reads as a script by its kind: StoryData/, or the dialogue files of RPGSystem/. */
export function isScriptFile(path: string): boolean {
  if (path.startsWith('StoryData/')) return true
  if (!path.startsWith('RPGSystem/')) return false
  return (path.split('/').pop() ?? '').toLowerCase().includes('dialogue')
}

/** What a script shows of an entry once the origin filter has been applied. */
export type Passage = {
  /** Stable for the entry whatever is filtered: the pointer of one of its units. */
  key: string
  /** The place divider to show in front of the passage. */
  divider?: Unit
  teller?: Unit
  title?: Unit
  /** The lines. */
  units: Unit[]
}

/** The pointers of the units a passage shows. */
export function passagePointers(p: Passage): string[] {
  return [p.divider, p.teller, p.title, ...p.units].flatMap((u) => (u ? [u.p] : []))
}

/**
 * The passages of a script under the origin filter. A passage whose lines are
 * all filtered out is hidden whole, speaker and all; its place divider, if it
 * brought one, moves to the next passage that is shown.
 */
export function passages(entries: Entry[], s: Settings): Passage[] {
  const out: Passage[] = []
  const shown = (u: Unit | undefined): Unit | undefined => (u !== undefined && isVisible(u, s) ? u : undefined)
  let pending: Unit | undefined // a place that has begun and has not been shown yet
  let shownPlace: string | undefined // the KR of the last divider shown
  for (const e of entries) {
    const place = shown(e.place)
    if (place !== undefined && e.placeChanged) pending = place
    const units = visibleUnits(e.units, s)
    if (e.units.length > 0 && units.length === 0) continue
    const teller = shown(e.teller)
    const title = shown(e.title)
    const first = e.units[0] ?? e.teller ?? e.title ?? e.place
    if (first === undefined || (units.length === 0 && teller === undefined && title === undefined && pending === undefined)) continue
    const divider = pending !== undefined && pending.kr !== shownPlace ? pending : undefined
    if (pending !== undefined) shownPlace = pending.kr
    pending = undefined
    out.push({ key: first.p, divider, teller, title, units })
  }
  return out
}
