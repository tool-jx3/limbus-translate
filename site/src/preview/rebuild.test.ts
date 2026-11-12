import { describe, expect, it } from 'vitest'
import type { BatchJSON, FileData, PreviewFile, PreviewFileWire, PreviewShared } from '../types'
import { expandPreviewFile } from './decode'
import { batchJSON, fullText, goJSON, rebuildBatch } from './rebuild'

// Read as text (?raw), so that the batches are compared byte for byte. The site
// has no node types, so this is how the other tests read fixtures too.
const raw = import.meta.glob('../../fixtures/preview/**/*.json', { eager: true, query: '?raw', import: 'default' }) as Record<string, string>
const read = (rel: string): string => {
  const text = raw[`../../fixtures/preview/${rel}`]
  if (text === undefined) throw new Error(`no fixture ${rel}`)
  return text
}
const json = <T>(rel: string): T => JSON.parse(read(rel)) as T
/** preview/<path> as the page works on it: the compact file with its defaults put back. */
const expanded = (path: string): PreviewFile => expandPreviewFile(json<PreviewFileWire>(`preview/${path}`), json<FileData>(`files/${path}`))

const shared = json<PreviewShared>('preview.json')
const cases = json<{ path: string; index: number; batch: string }[]>('cases.json')

describe('goJSON', () => {
  it('indents by two, ends with a line feed, and leaves < > & alone', () => {
    expect(goJSON({ a: '<b>&</b>' })).toBe('{\n  "a": "<b>&</b>"\n}\n')
  })
  it('escapes U+2028 and U+2029 as Go does', () => {
    expect(goJSON('a b c')).toBe('"a\\u2028b\\u2029c"\n')
  })
  it('writes empty arrays and objects compactly', () => {
    expect(goJSON({ a: [], b: {} })).toBe('{\n  "a": [],\n  "b": {}\n}\n')
  })
})

describe('rebuildBatch', () => {
  it('has a case for every batch file, once', () => {
    const files = Object.keys(raw).filter((k) => /\/batches\/[^/]+\.json$/.test(k)).map((k) => k.replace('../../fixtures/preview/', ''))
    expect(cases.map((c) => c.batch).sort()).toEqual(files.sort())
  })
  for (const c of cases) {
    it(`${c.path} #${c.index} is the text of ${c.batch}`, () => {
      const file = json<FileData>(`files/${c.path}`)
      const pf = expanded(c.path)
      expect(batchJSON(rebuildBatch(c.path, file, pf, shared, c.index))).toBe(read(c.batch))
    })
  }

  it('covers an entry with several jp and en forms, written as lists, and one of check: off', () => {
    const terms = cases.flatMap((c) => (JSON.parse(read(c.batch)) as BatchJSON).glossary)
    const forms = terms.find((t) => Array.isArray(t.jp) && Array.isArray(t.en))
    expect(forms?.jp).toEqual(['一つ目', '一番目'])
    expect(forms?.en).toEqual(['First', 'Firstly'])
    expect(terms.some((t) => typeof t.jp === 'string' || typeof t.en === 'string')).toBe(true)
    expect(terms.find((t) => t.check === 'off')?.kr).toBe('셋째')
  })

  it('rebuilds a batch from a preview written before the names were added, with no names', () => {
    const path = 'Other.json'
    const file = json<FileData>(`files/${path}`)
    const older = json<PreviewFileWire>(`preview/${path}`)
    delete older.names
    for (const u of older.units) if (u) delete u.n
    const pf = expandPreviewFile(older, file)
    const i = pf.units.findIndex((u) => u !== null)
    expect(rebuildBatch(path, file, pf, shared, i).names).toEqual([])
  })

  it('refuses a copy', () => {
    const c = expanded('StoryData/S001A.json')
    const i = c.units.findIndex((u) => u === null)
    expect(i).toBeGreaterThanOrEqual(0)
    expect(() => rebuildBatch('StoryData/S001A.json', json<FileData>('files/StoryData/S001A.json'), c, shared, i)).toThrow(/has no preview/)
  })

  const base = () => {
    const file = json<FileData>('files/StoryData/S001A.json')
    const pf = expanded('StoryData/S001A.json')
    return { file, pf, i: 2 }
  }
  const run = (file: FileData, pf: PreviewFile, i: number) => rebuildBatch('StoryData/S001A.json', file, pf, shared, i)

  it('throws for a context line whose unit has no preview', () => {
    const { file, pf, i } = base()
    const nul = pf.units.findIndex((u) => u === null)
    pf.units[i]!.c = [[nul]]
    expect(() => run(file, pf, i)).toThrow(/context unit \d+ has no preview/)
  })
  it('throws for an en method with no EN string', () => {
    const { file, pf, i } = base()
    delete file.units[i]!.en
    expect(() => run(file, pf, i)).toThrow(/no EN string/)
  })
  it('throws for a jp method with no JP string', () => {
    const { file, pf, i } = base()
    delete file.units[i]!.jp
    expect(() => run(file, pf, i)).toThrow(/no JP string/)
  })
  it('throws for a keyword id missing from the table', () => {
    const { file, pf, i } = base()
    pf.units[i]!.k = ['Nope']
    expect(() => run(file, pf, i)).toThrow(/no keyword Nope/)
  })
  it('throws when the preview and the file differ in length', () => {
    const { file, pf, i } = base()
    pf.units.pop()
    expect(() => run(file, pf, i)).toThrow(/units in the preview/)
  })
  it('writes keywords and refs in byte order', () => {
    const { file, pf, i } = base()
    pf.keywords = { '9': 'x', '10': 'y', a: 'z', B: 'w' }
    pf.units[i]!.k = ['9', '10', 'a', 'B']
    pf.units[i]!.it.refs = { '9': 'x', '10': 'y', a: 'z', B: 'w' }
    const text = batchJSON(run(file, pf, i))
    const order = (from: string) => [...text.slice(text.indexOf(from)).matchAll(/^ {4,8}"([^"]+)": "[xyzw]"/gm)].map((m) => m[1]).slice(0, 4)
    expect(order('"keywords"')).toEqual(['10', '9', 'B', 'a'])
    expect(order('"refs"')).toEqual(['10', '9', 'B', 'a'])
  })
  it('sorts by the path of the map, not by the identity of the object', () => {
    const { file, pf, i } = base()
    pf.keywords = { '9': 'x', '10': 'y', a: 'z', B: 'w' }
    pf.units[i]!.k = ['9', '10', 'a', 'B']
    pf.units[i]!.it.refs = { '9': 'x', '10': 'y', a: 'z', B: 'w' }
    const batch = run(file, pf, i)
    const want = batchJSON(batch)
    // Copies of the maps (spread, structuredClone) are other objects at the same paths.
    const copied = { ...batch, keywords: { ...batch.keywords }, units: batch.units.map((u) => ({ ...u, refs: u.refs && { ...u.refs } })) }
    expect(batchJSON(copied)).toBe(want)
    expect(batchJSON(structuredClone(batch))).toBe(want)
    // goJSON knows no paths: any other object keeps the order it was made in.
    expect(goJSON({ b: 1, 10: 2, a: 3 })).toBe('{\n  "10": 2,\n  "b": 1,\n  "a": 3\n}\n')
    expect(goJSON({ keywords: { b: 1, a: 2 } })).toBe('{\n  "keywords": {\n    "b": 1,\n    "a": 2\n  }\n}\n')
  })
})

describe('fullText', () => {
  it('throws for a guide the shared data lacks', () => {
    const s = { rules_commit: 'x', prompt: 'P', guides: [] }
    expect(() => fullText(s, { guides: ['guides/a.md'] } as unknown as BatchJSON)).toThrow(/no guide guides\/a.md/)
  })
  it('puts the prompt, the guides and the batch in order', () => {
    const batch = { id: 'preview' } as unknown as BatchJSON
    const s = { rules_commit: 'x', prompt: 'P\n', guides: [{ path: 'guides/a.md', text: 'A\n' }] }
    const text = fullText(s, { ...batch, guides: ['guides/a.md'] } as BatchJSON)
    const at = ['# prompts/translate.md', 'P', '# guides/a.md', 'A', '# 批次 JSON', '"id": "preview"'].map((x) => text.indexOf(x))
    expect(at.every((n) => n >= 0)).toBe(true)
    expect([...at].sort((a, b) => a - b)).toEqual(at)
    expect(text).toContain('# prompts/translate.md\n\nP\n')
  })
})
