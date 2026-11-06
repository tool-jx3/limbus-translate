// The fixtures are written by hand to the layout of the site spec, section 5,
// and are what the tests of the viewer read. This keeps them true to types.ts
// and to themselves: the index, the files and the search corpus agree.
import { describe, expect, it } from 'vitest'
import type { FileData, GlossaryFile, GlossaryIndex, Index, Meta, NoteItem, Origin, ScopeEntry, ScopeProposal, SuggestionItem, TodoSection, Unit, WarningItem } from './types'

const modules = import.meta.glob('../fixtures/data/**/*.json', { eager: true, import: 'default' })
const at = (rel: string): unknown => modules[`../fixtures/data/${rel}`]

const meta = at('meta.json') as Meta
const index = at('index.json') as Index
const keys = at('search/keys.json') as [number, string][]
const corpus = Object.fromEntries(
  (['kr', 'jp', 'en', 'zh'] as const).map((lang) => [lang, at(`search/${lang}.json`) as string[]]),
)
const files = index.files.map((f) => at(`files/${f.path}`) as FileData)

const suggestions = (at('review/suggestions.json') as { items: SuggestionItem[] }).items
const warnings = (at('review/warnings.json') as { items: WarningItem[] }).items
const notes = (at('review/notes.json') as { items: NoteItem[] }).items
const todo = (at('review/todo.json') as { sections: TodoSection[] }).sections
const scope = (at('review/scope.json') as { proposals: ScopeProposal[] }).proposals

const ORIGINS: Origin[] = ['paratranz', 'claude', 'human', 'copy', 'fallback', 'stand-in']

describe('fixtures', () => {
  it('has the keys of the Go types, and no others', () => {
    expect(Object.keys(meta)).toEqual(['format', 'raw_tag', 'rules_commit', 'tw_commit', 'release', 'langs', 'files', 'units', 'review'])
    expect(Object.keys(meta.review)).toEqual(['suggestions', 'warnings', 'notes', 'todo', 'scope'])
    expect(Object.keys(index)).toEqual(['families', 'files', 'groups'])
    for (const fam of index.families) expect(Object.keys(fam)).toEqual(['id', 'files', 'glossary', 'nodes'])
    for (const f of index.files) {
      // FileSummary (internal/site/file.go): sub and tag are omitempty; the order is that of the struct.
      const optional = Object.keys(f).filter((k) => !['path', 'family', 'top', 'label', 'speakers', 'units', 'origins', 'missing'].includes(k))
      for (const k of optional) expect(['sub', 'tag']).toContain(k)
      expect(Object.keys(f)).toEqual([
        'path', 'family', 'top', ...(f.sub !== undefined ? ['sub'] : []), ...(f.tag !== undefined ? ['tag'] : []),
        'label', 'speakers', 'units', 'origins', 'missing',
      ])
      if (f.tag !== undefined) expect(Object.keys(f.tag)).toEqual(['kind', 'value', 'label'])
      expect(Object.keys(f.missing)).toEqual(['en', 'jp'])
    }
    const checkGroups = (gs: typeof index.groups) => {
      for (const grp of gs) {
        expect(Object.keys(grp)).toEqual(['kind', 'node', 'label', 'files', 'children'])
        checkGroups(grp.children)
      }
    }
    checkGroups(index.groups)
    for (const f of files) {
      expect(Object.keys(f)).toEqual(['path', 'family', 'units'])
      for (const u of f.units) {
        const keysOf = Object.keys(u)
        expect(keysOf.filter((k) => !['p', 'f', 'i', 'kr', 'jp', 'en', 'zh', 'o', 'fl', 'rv', 'nt', 'pq'].includes(k))).toEqual([])
        for (const k of ['p', 'f', 'i', 'kr', 'zh', 'o']) expect(keysOf).toContain(k)
        expect(ORIGINS).toContain(u.o)
      }
    }
  })

  it('counts its files and units in the meta', () => {
    expect(meta.format).toBe(2)
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

  it('has groups that list every file once, under the top of the file', () => {
    const seen: number[] = []
    const walk = (gs: typeof index.groups, top: string) => {
      for (const grp of gs) {
        const t = grp.kind === 'top' ? grp.node : top
        for (const n of grp.files) {
          seen.push(n)
          expect(index.files[n]?.top).toBe(t)
        }
        walk(grp.children, t)
      }
    }
    walk(index.groups, '')
    expect(seen.sort((x, y) => x - y)).toEqual(index.files.map((_, n) => n))
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
    // a string outside dataList, which has no element to belong to
    expect(all.some((u) => u.i === -1)).toBe(true)
  })
})

describe('review fixtures', () => {
  const unitOf = new Map<string, Unit>(files.flatMap((f) => f.units.map((u): [string, Unit] => [`${f.path}|${u.p}`, u])))
  const ids = (): string[] => [...suggestions.map((s) => s.id), ...warnings.map((w) => w.id), ...notes.map((n) => n.id)]

  it('splits each variant of an inconsistent group by the JP and EN of its units, and classes the group', () => {
    for (const s of suggestions.filter((x) => x.kind === 'inconsistent')) {
      const pairs = new Map<string, Set<string>>()
      for (const v of s.variants ?? []) {
        expect(v.jpen).toBeDefined()
        expect((v.jpen ?? []).flatMap((j) => j.keys).sort()).toEqual([...v.keys].sort())
        for (const j of v.jpen ?? []) {
          expect(j.count).toBe(j.keys.length)
          for (const key of j.keys) {
            const u = unitOf.get(key)!
            expect([u.jp, u.en]).toEqual([j.jp, j.en])
          }
          const p = `${j.jp ?? '-'}\u0000${j.en ?? '-'}`
          pairs.set(p, (pairs.get(p) ?? new Set()).add(v.zh))
        }
      }
      const want = pairs.size <= 1 ? 'same' : [...pairs.values()].some((z) => z.size > 1) ? 'partial' : 'explained'
      expect(s.jpen_class).toBe(want)
    }
  })

  it('has the keys of the Go types, in their order, leaving out what is empty', () => {
    for (const s of suggestions) {
      const keysOf = Object.keys(s)
      expect(keysOf.slice(0, 3)).toEqual(['id', 'kind', 'title'])
      expect(keysOf.slice(-2 - (s.declined ? 1 : 0))).toEqual(s.declined ? ['evidence', 'pending', 'declined'] : ['evidence', 'pending'])
      if (s.kind !== 'inconsistent') expect(keysOf).toEqual(['id', 'kind', 'title', 'evidence', 'pending', ...(s.declined ? ['declined'] : [])])
    }
    for (const w of warnings) {
      expect(Object.keys(w)).toEqual(['id', 'code', 'kr', 'zh', 'message', 'keys', 'families', 'origins', 'pending', ...(w.accepted ? ['accepted'] : [])])
    }
    for (const n of notes) {
      expect(Object.keys(n)).toEqual(['id', 'key', 'family', 'batch', 'note', 'kr', 'zh', 'o', 'pending', ...(n.declined ? ['declined'] : [])])
    }
    expect(Object.keys(at('review/suggestions.json') as object)).toEqual(['items'])
    expect(Object.keys(at('review/todo.json') as object)).toEqual(['sections'])
    for (const sec of todo) {
      expect(Object.keys(sec)).toEqual(['kind', 'title', 'hint', 'items'])
      for (const it of sec.items) expect(Object.keys(it)).toEqual(it.key === undefined ? ['text'] : ['text', 'key'])
    }
  })

  it('counts the unconfirmed items of each file in the meta', () => {
    expect(meta.review).toEqual({
      suggestions: suggestions.filter((s) => !s.declined).length,
      warnings: warnings.filter((w) => !w.accepted).length,
      notes: notes.filter((n) => !n.declined).length,
      todo: todo.reduce((n, sec) => n + sec.items.length, 0),
      scope: scope.filter((p) => !p.declined).length,
    })
  })

  it('has every rv id in a review file, never a confirmed one, and every unconfirmed one on a unit', () => {
    const open = new Set([
      ...suggestions.filter((s) => !s.declined).map((s) => s.id),
      ...warnings.filter((w) => !w.accepted).map((w) => w.id),
      ...notes.filter((n) => !n.declined).map((n) => n.id),
    ])
    const all = new Set(ids())
    const seen = new Set<string>()
    for (const u of unitOf.values()) {
      expect(u.rv === undefined || u.rv.length > 0).toBe(true)
      expect([...(u.rv ?? [])].sort()).toEqual(u.rv ?? [])
      for (const id of u.rv ?? []) {
        expect(all.has(id), id).toBe(true)
        expect(open.has(id), id).toBe(true)
        seen.add(id)
      }
    }
    // the unconfirmed items of units: inconsistent groups, warnings and notes
    const onUnits = [...open].filter((id) => !id.startsWith('speaker/') && !(id.startsWith('note/') && !notes.some((n) => n.id === id)))
    expect([...seen].sort()).toEqual(onUnits.sort())
  })

  it('puts each item on units that exist, with the rv it says', () => {
    for (const s of suggestions) {
      for (const v of s.variants ?? []) {
        expect(v.keys).toHaveLength(v.count)
        for (const k of v.keys) expect(unitOf.get(k)?.rv ?? [], k).toContain(s.id)
      }
      for (const e of s.evidence) if (e.key !== undefined) expect(unitOf.has(e.key), e.key).toBe(true)
      const units = s.variants ? s.variants.flatMap((v) => v.keys) : s.evidence.flatMap((e) => (e.key ? [e.key] : []))
      expect(s.pending).toBe(units.filter((k) => unitOf.get(k)?.pq === 1).length)
    }
    for (const w of warnings) {
      for (const k of w.keys) {
        const u = unitOf.get(k)
        expect(u?.kr, k).toBe(w.kr)
        expect(u?.zh, k).toBe(w.zh)
        if (!w.accepted) expect(u?.rv ?? [], k).toContain(w.id)
        else expect(u?.rv ?? [], k).not.toContain(w.id)
      }
      expect(w.pending).toBe(w.keys.filter((k) => unitOf.get(k)?.pq === 1).length)
    }
  })

  it('has the unit of each note, with its text in nt, confirmed or not', () => {
    for (const n of notes) {
      const u = unitOf.get(n.key)
      expect(u?.nt?.split('\n'), n.key).toContain(n.note)
      expect(u?.kr).toBe(n.kr)
      expect(u?.zh).toBe(n.zh)
      expect(u?.o).toBe(n.o)
      expect(n.id.startsWith(`note/${n.key}/`)).toBe(true)
      expect(n.pending).toBe(u?.pq === 1 ? 1 : 0)
    }
    for (const [key, u] of unitOf) if (u.nt !== undefined) expect(notes.some((n) => n.key === key), key).toBe(true)
  })

  it('has the notes also as rule suggestions, confirmed together', () => {
    for (const n of notes) {
      const s = suggestions.find((x) => x.id === n.id)
      expect(s?.kind).toBe('note')
      expect(Boolean(s?.declined)).toBe(Boolean(n.declined))
    }
  })

  it('covers what the review page has to show', () => {
    const inconsistent = suggestions.filter((s) => s.kind === 'inconsistent')
    expect(inconsistent.some((s) => !s.declined && (s.variants ?? []).length === 2)).toBe(true)
    expect(suggestions.some((s) => s.kind !== 'inconsistent' && !s.declined)).toBe(true)
    expect(suggestions.some((s) => s.declined)).toBe(true)
    expect(warnings.some((w) => w.code === 'kana' && !w.accepted && w.keys.length === 2)).toBe(true)
    expect(warnings.some((w) => w.accepted)).toBe(true)
    expect(notes).toHaveLength(2)
    expect(notes.filter((n) => n.declined)).toHaveLength(1)
    expect(todo.map((s) => s.kind)).toContain('needs_review')
    expect(todo.some((s) => s.items.some((i) => i.key !== undefined && unitOf.has(i.key)))).toBe(true)
    expect([...unitOf.values()].some((u) => u.pq === 1)).toBe(true)
    // the units that the todo section calls needs_review carry the flag
    for (const it of todo.find((s) => s.kind === 'needs_review')?.items ?? []) expect(unitOf.get(it.key ?? '')?.fl).toContain('needs_review')
  })
})

describe('glossary fixtures', () => {
  const g = at('glossary/index.json') as GlossaryIndex
  const entryFile = (id: string) => at(`glossary/${id.replace(/^glossary\//, '').replace('/', '-')}.json`) as GlossaryFile
  const scopeEntries = (at('review/scope.json') as { entries: ScopeEntry[] }).entries

  it('names only families that the site index has (the family:<id> groups)', () => {
    const known = index.families.map((f) => f.id)
    const named = g.entries.flatMap((e) => e.groups.filter((x) => x.kind === 'family').map((x) => x.name.replace(/^family:/, '')))
    expect(named.length).toBeGreaterThan(0)
    for (const id of new Set(named)) expect(known).toContain(id)
    const patterns = g.entries.flatMap((e) => [...e.files, ...e.not_files]).filter((x) => x.startsWith('family:'))
    for (const x of patterns) expect(known).toContain(x.slice('family:'.length))
  })

  it('has the keys of the Go types, in their order (internal/scope, internal/site/glossary.go)', () => {
    expect(Object.keys(g)).toEqual(['thresholds', 'entries'])
    expect(Object.keys(g.thresholds)).toEqual(['min_n', 'used', 'not_used', 'explained', 'lost_units', 'lost_share', 'hard_rate', 'hard_min_n'])
    expect(g.entries.length).toBeGreaterThanOrEqual(2)
    for (const e of g.entries) {
      expect(Object.keys(e)).toEqual([
        'id', 'kr', 'zh', 'also', 'files', 'not_files', 'match', ...(e.check !== undefined ? ['check'] : []), 'notes', 'groups',
        'n', 'used', 'rate', 'hard_candidate', 'compound', 'explained', 'unexplained', 'examples',
      ])
      expect(['word', 'substring']).toContain(e.match)
      for (const grp of e.groups) expect(Object.keys(grp)).toEqual(['name', 'kind', 'n', 'used'])
      expect(Object.keys(e.compound)).toEqual(['n', 'used'])
      for (const ex of e.examples) expect(Object.keys(ex)).toEqual(['key', 'group', 'kr', 'zh'])
    }
    for (const id of g.entries.map((e) => e.id)) {
      const f = entryFile(id)
      expect(Object.keys(f)).toEqual(['id', 'missing', 'used'])
      for (const [which, us] of [['missing', f.missing], ['used', f.used]] as const) {
        for (const u of us) {
          expect(Object.keys(u)).toEqual([
            'key', 'family', 'top', ...(u.sub !== undefined ? ['sub'] : []), ...(u.tag !== undefined ? ['tag'] : []), 'origin', 'kr', 'zh',
            ...(u.note !== undefined ? ['note'] : []), ...(which === 'missing' ? ['id', ...(u.accepted !== undefined ? ['accepted'] : [])] : []),
          ])
          if (u.tag !== undefined) expect(Object.keys(u.tag)).toEqual(['kind', 'value', 'label'])
          expect(ORIGINS).toContain(u.origin)
        }
      }
    }
  })

  it('counts n and used as the units of the entry file, and agrees with the scope entries where they overlap', () => {
    for (const e of g.entries) {
      const f = entryFile(e.id)
      expect(f.id).toBe(e.id)
      expect(f.missing.length + f.used.length, e.id).toBe(e.n)
      expect(f.used.length, e.id).toBe(e.used)
      expect(e.rate).toBeCloseTo(e.n === 0 ? 0 : e.used / e.n, 3)
      expect(e.hard_candidate).toBe(e.check !== 'hard' && e.n >= g.thresholds.hard_min_n && e.rate > g.thresholds.hard_rate)
      const same = scopeEntries.find((s) => s.id === e.id)
      if (same) expect(e).toEqual(same)
      const keys = [...f.missing, ...f.used].map((u) => u.key)
      expect(new Set(keys).size, e.id).toBe(keys.length)
      for (const u of f.missing) expect(u.id?.startsWith('warning/term-missing/')).toBe(true)
    }
  })

  it('covers what the glossary page has to show', () => {
    expect(g.entries.some((e) => e.hard_candidate)).toBe(true)
    expect(g.entries.some((e) => !e.hard_candidate && e.n > e.used)).toBe(true)
    const all = g.entries.flatMap((e) => [...entryFile(e.id).missing, ...entryFile(e.id).used])
    expect(all.some((u) => u.accepted !== undefined)).toBe(true)
    expect(all.some((u) => u.note !== undefined)).toBe(true)
    expect(new Set(all.map((u) => u.origin)).size).toBeGreaterThanOrEqual(3)
    expect(new Set(all.map((u) => u.family)).size).toBeGreaterThanOrEqual(2)
    expect(new Set(all.map((u) => u.tag?.kind)).size).toBeGreaterThanOrEqual(2)
  })
})
