import { describe, expect, it } from 'vitest'
import { defaultSettings, type Settings } from '../settings'
import type { Origin, Unit } from '../types'
import { groupEntries, isScriptFile, passagePointers, passages, visibleUnits } from './group'

function unit(f: string, i: number, kr: string, o: Origin = 'paratranz'): Unit {
  const p = i < 0 ? `/${f}` : `/dataList/${i}/${f}`
  return { p, f, i, kr, zh: `zh:${kr}`, o }
}

function settings(change: (s: Settings) => void = () => {}): Settings {
  const s = defaultSettings()
  change(s)
  return s
}

describe('groupEntries', () => {
  it('puts teller, title and content of one dataList element in one entry', () => {
    const teller = unit('teller', 1, 'T')
    const title = unit('title', 1, 'Ti')
    const content = unit('content', 1, 'C')
    const entries = groupEntries([teller, title, content])
    expect(entries).toHaveLength(1)
    expect(entries[0]?.i).toBe(1)
    expect(entries[0]?.teller).toBe(teller)
    expect(entries[0]?.title).toBe(title)
    expect(entries[0]?.units).toEqual([content])
    expect(entries[0]?.place).toBeUndefined()
  })

  it('keeps the other units of an element, in order, in units', () => {
    const model = unit('model', 2, 'M', 'copy')
    const content = unit('content', 2, 'C')
    const entries = groupEntries([model, unit('teller', 2, 'T'), content])
    expect(entries[0]?.units).toEqual([model, content])
  })

  it('makes an entry of each unit outside dataList (i == -1), even next to one another', () => {
    const a = unit('footnote', -1, 'a')
    const b = unit('note', -1, 'b')
    const entries = groupEntries([unit('content', 0, 'x'), a, b, unit('content', 1, 'y')])
    expect(entries.map((e) => e.i)).toEqual([0, -1, -1, 1])
    expect(entries[1]?.units).toEqual([a])
    expect(entries[2]?.units).toEqual([b])
  })

  it('starts a new entry where the element changes, not where it was seen before', () => {
    const entries = groupEntries([unit('content', 0, 'a'), unit('content', 1, 'b'), unit('content', 0, 'c')])
    expect(entries.map((e) => e.i)).toEqual([0, 1, 0])
  })

  it('does not lose a second place, teller or title of one element', () => {
    const first = unit('teller', 0, 'one')
    const second = { ...unit('teller', 0, 'two'), p: '/dataList/0/other/teller' }
    const entries = groupEntries([first, second, unit('content', 0, 'c')])
    expect(entries[0]?.teller).toBe(first)
    expect(entries[0]?.units.map((u) => u.p)).toEqual(['/dataList/0/other/teller', '/dataList/0/content'])
  })

  it('is empty for no units', () => {
    expect(groupEntries([])).toEqual([])
  })

  it('marks placeChanged where the KR of the place differs from the last place', () => {
    const entries = groupEntries([
      unit('place', 0, 'A'),
      unit('content', 0, 'a'),
      unit('content', 1, 'b'),
      unit('place', 2, 'B'),
      unit('content', 2, 'c'),
      unit('place', 3, 'B'),
      unit('content', 3, 'd'),
      unit('place', 4, 'A'),
      unit('content', 4, 'e'),
    ])
    expect(entries.map((e) => e.placeChanged)).toEqual([true, false, true, false, true])
  })

  it('compares with the last entry that has a place, not with the one before', () => {
    const entries = groupEntries([unit('place', 0, 'A'), unit('content', 1, 'x'), unit('place', 2, 'A'), unit('content', 3, 'y')])
    expect(entries.map((e) => e.placeChanged)).toEqual([true, false, false, false])
  })
})

describe('visibleUnits', () => {
  const units = [unit('a', 0, '1', 'paratranz'), unit('b', 0, '2', 'claude'), unit('c', 0, '3', 'copy'), unit('d', 0, '4', 'stand-in')]

  it('hides copy by default, and nothing else', () => {
    expect(visibleUnits(units, defaultSettings()).map((u) => u.o)).toEqual(['paratranz', 'claude', 'stand-in'])
  })

  it('removes the claude units when origins.claude is off', () => {
    const s = settings((s) => {
      s.origins.claude = false
    })
    expect(visibleUnits(units, s).map((u) => u.o)).toEqual(['paratranz', 'stand-in'])
  })

  it('removes the copy units when origins.copy is off, and brings them when it is on', () => {
    const off = settings((s) => {
      s.origins.copy = false
    })
    const on = settings((s) => {
      s.origins.copy = true
    })
    expect(visibleUnits(units, off).map((u) => u.o)).not.toContain('copy')
    expect(visibleUnits(units, on).map((u) => u.o)).toEqual(['paratranz', 'claude', 'copy', 'stand-in'])
  })
})

describe('isScriptFile', () => {
  it('is true under StoryData/ and for dialogue files of RPGSystem/', () => {
    expect(isScriptFile('StoryData/S001A.json')).toBe(true)
    expect(isScriptFile('RPGSystem/rpg-loc-dialogue-floor-3.json')).toBe(true)
    expect(isScriptFile('RPGSystem/rpg-loc-dialogue-choice-floor-1.json')).toBe(true)
  })

  it('is false for the others', () => {
    expect(isScriptFile('Skills_personality-01.json')).toBe(false)
    expect(isScriptFile('RPGSystem/rpg-loc-items.json')).toBe(false)
    expect(isScriptFile('Passives-dialogue.json')).toBe(false)
    expect(isScriptFile('StoryDataX/S001A.json')).toBe(false)
  })
})

describe('passages', () => {
  const story = [
    unit('place', 0, 'A'),
    unit('content', 0, 'a'),
    unit('teller', 1, 'T1'),
    unit('title', 1, 'Ti1'),
    unit('content', 1, 'b', 'claude'),
    unit('place', 2, 'B'),
    unit('content', 2, 'c', 'claude'),
    unit('teller', 3, 'T3'),
    unit('content', 3, 'd'),
    unit('footnote', -1, 'f'),
  ]

  it('has a passage for each entry, with the divider where the place changed', () => {
    const ps = passages(groupEntries(story), defaultSettings())
    expect(ps.map((p) => p.units.map((u) => u.kr))).toEqual([['a'], ['b'], ['c'], ['d'], ['f']])
    expect(ps.map((p) => p.divider?.kr)).toEqual(['A', undefined, 'B', undefined, undefined])
    expect(ps[1]?.teller?.kr).toBe('T1')
    expect(ps[1]?.title?.kr).toBe('Ti1')
  })

  it('hides a passage whose lines are all filtered out, speaker and all', () => {
    const s = settings((s) => {
      s.origins.claude = false
    })
    const ps = passages(groupEntries(story), s)
    expect(ps.map((p) => p.units.map((u) => u.kr))).toEqual([['a'], ['d'], ['f']])
    expect(ps.flatMap((p) => [p.teller?.kr, p.title?.kr])).not.toContain('Ti1')
  })

  it('shows the divider of a hidden passage in front of the next passage that is shown', () => {
    const s = settings((s) => {
      s.origins.claude = false
    })
    const ps = passages(groupEntries(story), s)
    // passage 2 (place B) is hidden; passage 3 comes after it
    expect(ps.map((p) => p.divider?.kr)).toEqual(['A', 'B', undefined])
  })

  it('does not show the same place twice in a row', () => {
    const s = settings((s) => {
      s.origins.claude = false
    })
    const units = [unit('place', 0, 'A'), unit('content', 0, 'a'), unit('place', 1, 'B'), unit('content', 1, 'b', 'claude'), unit('place', 2, 'A'), unit('content', 2, 'c')]
    const ps = passages(groupEntries(units), s)
    expect(ps.map((p) => p.divider?.kr)).toEqual(['A', undefined])
  })

  it('keeps a repeated place on its passage, so that a link to it finds the passage', () => {
    const units = [unit('place', 0, 'A'), unit('content', 0, 'a'), unit('place', 1, 'A'), unit('content', 1, 'b')]
    const ps = passages(groupEntries(units), defaultSettings())
    expect(ps.map((p) => p.divider?.kr)).toEqual(['A', undefined])
    expect(ps[1]?.place?.p).toBe('/dataList/1/place')
    expect(passagePointers(ps[1] as (typeof ps)[number])).toContain('/dataList/1/place')
    // the divider is drawn from the unit that is the divider, not twice
    expect(ps[0]?.place).toBeUndefined()
    expect(passagePointers(ps[0] as (typeof ps)[number]).filter((p) => p === '/dataList/0/place')).toHaveLength(1)
  })

  it('keeps the place of a passage whose divider was dropped as a copy of the one shown', () => {
    const s = settings((s) => {
      s.origins.claude = false
    })
    const units = [unit('place', 0, 'A'), unit('content', 0, 'a'), unit('place', 1, 'B'), unit('content', 1, 'b', 'claude'), unit('place', 2, 'A'), unit('content', 2, 'c')]
    const ps = passages(groupEntries(units), s)
    expect(ps.map((p) => p.divider?.kr)).toEqual(['A', undefined])
    expect(ps[1]?.place?.p).toBe('/dataList/2/place')
    expect(passagePointers(ps[1] as (typeof ps)[number])).toContain('/dataList/2/place')
  })

  it('shows a passage that is only a repeated place, since there is nothing else it could show', () => {
    const units = [unit('place', 0, 'A'), unit('content', 0, 'a'), unit('place', 1, 'A'), unit('content', 2, 'c')]
    const ps = passages(groupEntries(units), defaultSettings())
    expect(ps).toHaveLength(3)
    expect(ps[1]?.divider?.p).toBe('/dataList/1/place')
    expect(ps[1]?.place).toBeUndefined()
    expect(ps[1]?.units).toEqual([])
  })

  it('still hides a repeated place together with the passage whose lines are filtered out', () => {
    const s = settings((s) => {
      s.origins.claude = false
    })
    const units = [unit('place', 0, 'A'), unit('content', 0, 'a'), unit('place', 1, 'A'), unit('content', 1, 'b', 'claude')]
    const ps = passages(groupEntries(units), s)
    expect(ps).toHaveLength(1)
    expect(ps.flatMap(passagePointers)).not.toContain('/dataList/1/place')
  })

  it('keeps a passage that has no lines in the file at all, if its speaker is shown', () => {
    const ps = passages(groupEntries([unit('teller', 0, 'T'), unit('title', 0, 'Ti'), unit('content', 1, 'c')]), defaultSettings())
    expect(ps).toHaveLength(2)
    expect(ps[0]?.teller?.kr).toBe('T')
    expect(ps[0]?.units).toEqual([])
  })

  it('drops a passage with nothing left to show', () => {
    const s = settings((s) => {
      s.origins.copy = false
    })
    const ps = passages(groupEntries([unit('model', 0, 'M', 'copy')]), s)
    expect(ps).toEqual([])
  })

  it('names a passage by a unit of the entry, whatever is filtered', () => {
    const all = passages(groupEntries(story), defaultSettings())
    const s = settings((s) => {
      s.origins.paratranz = false
    })
    const some = passages(groupEntries(story), s)
    expect(new Set(all.map((p) => p.key)).size).toBe(all.length)
    expect(some.every((p) => all.some((q) => q.key === p.key))).toBe(true)
  })
})
