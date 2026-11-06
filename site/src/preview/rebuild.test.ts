import { describe, expect, it } from 'vitest'
import type { BatchJSON, FileData, PreviewFile, PreviewShared } from '../types'
import { fullText, goJSON, rebuildBatch } from './rebuild'

// Read as text (?raw), so that the batches are compared byte for byte. The site
// has no node types, so this is how the other tests read fixtures too.
const raw = import.meta.glob('../../fixtures/preview/**/*.json', { eager: true, query: '?raw', import: 'default' }) as Record<string, string>
const read = (rel: string): string => {
  const text = raw[`../../fixtures/preview/${rel}`]
  if (text === undefined) throw new Error(`no fixture ${rel}`)
  return text
}
const json = <T>(rel: string): T => JSON.parse(read(rel)) as T

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
  it('has cases', () => expect(cases.length).toBeGreaterThan(0))
  for (const c of cases) {
    it(`${c.path} #${c.index} is the text of ${c.batch}`, () => {
      const file = json<FileData>(`files/${c.path}`)
      const pf = json<PreviewFile>(`preview/${c.path}`)
      expect(goJSON(rebuildBatch(c.path, file, pf, shared, c.index))).toBe(read(c.batch))
    })
  }

  it('refuses a copy', () => {
    const c = json<PreviewFile>('preview/StoryData/S001A.json')
    const i = c.units.findIndex((u) => u === null)
    expect(i).toBeGreaterThanOrEqual(0)
    expect(() => rebuildBatch('StoryData/S001A.json', json<FileData>('files/StoryData/S001A.json'), c, shared, i)).toThrow()
  })
})

describe('fullText', () => {
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
