import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { DataError, clearDataCache, loadFile, loadGlossary, loadGlossaryEntry, loadIndex, loadMeta, loadPreviewFile, loadPreviewShared, loadReview, loadUpdate, loadUpdates } from './data'
import { fixture, fixtureFetch } from './testdata'
import type { FileData, Index, Meta, ReviewData } from './types'

function serve(docs: Record<string, unknown> = {}) {
  const fetchMock = vi.fn(fixtureFetch(docs))
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

async function kindOf(p: Promise<unknown>): Promise<string> {
  try {
    await p
  } catch (e) {
    if (e instanceof DataError) return e.kind
    throw e
  }
  return 'resolved'
}

beforeEach(() => clearDataCache())
afterEach(() => vi.unstubAllGlobals())

describe('data loading', () => {
  it('reads meta.json, the index and a file from ./data/', async () => {
    const fetchMock = serve()
    expect(await loadMeta()).toEqual(fixture<Meta>('meta.json'))
    expect(await loadIndex()).toEqual(fixture<Index>('index.json'))
    expect(await loadFile('StoryData/S9999.json')).toEqual(fixture<FileData>('files/StoryData/S9999.json'))
    expect(fetchMock.mock.calls.map((c) => c[0])).toEqual(['./data/meta.json', './data/index.json', './data/files/StoryData/S9999.json'])
  })

  it('reads a file with no strings', async () => {
    serve()
    expect((await loadFile('Passives-empty.json')).units).toEqual([])
  })

  it('encodes the segments of the path it asks for', async () => {
    const fetchMock = serve({ 'files/a b/c%d.json': { path: 'a b/c%d.json', family: 'x', units: [] } })
    expect((await loadFile('a b/c%d.json')).path).toBe('a b/c%d.json')
    expect(fetchMock.mock.calls[0]?.[0]).toBe('./data/files/a%20b/c%25d.json')
  })

  it('format mismatch shows reload notice', async () => {
    serve({ 'meta.json': { ...fixture<Meta>('meta.json'), format: 3 } })
    expect(await kindOf(loadMeta())).toBe('format')
  })

  it('format 1 asks for reload', async () => {
    serve({ 'meta.json': { ...fixture<Meta>('meta.json'), format: 1 } })
    expect(await kindOf(loadMeta())).toBe('format')
  })

  it('reads the four review files from ./data/review/', async () => {
    const fetchMock = serve()
    for (const tab of ['suggestions', 'warnings', 'notes', 'todo'] as const) {
      expect(await loadReview(tab)).toEqual(fixture<ReviewData[typeof tab]>(`review/${tab}.json`))
    }
    expect(fetchMock.mock.calls.map((c) => c[0])).toEqual([
      './data/review/suggestions.json',
      './data/review/warnings.json',
      './data/review/notes.json',
      './data/review/todo.json',
    ])
  })

  it('fetches a review file once, and reports a bad one as a format error', async () => {
    const fetchMock = serve()
    await loadReview('notes')
    await loadReview('notes')
    expect(fetchMock).toHaveBeenCalledTimes(1)
    clearDataCache()
    serve({ 'review/todo.json': { items: [] }, 'review/notes.json': { sections: [] } })
    expect(await kindOf(loadReview('todo'))).toBe('format')
    expect(await kindOf(loadReview('notes'))).toBe('format')
    serve({ 'review/warnings.json': 'x' })
    expect(await kindOf(loadReview('warnings'))).toBe('format')
    vi.stubGlobal('fetch', vi.fn(async () => new Response('', { status: 404 })))
    expect(await kindOf(loadReview('suggestions'))).toBe('missing')
  })

  it('fetch 404 shows reload notice', async () => {
    serve()
    expect(await kindOf(loadFile('StoryData/Gone.json'))).toBe('missing')
    vi.stubGlobal('fetch', vi.fn(async () => new Response('', { status: 404 })))
    expect(await kindOf(loadMeta())).toBe('missing')
    expect(await kindOf(loadIndex())).toBe('missing')
  })

  it('calls any other HTTP failure and a failed fetch a network error', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => new Response('', { status: 503 })))
    expect(await kindOf(loadMeta())).toBe('network')
    vi.stubGlobal('fetch', vi.fn(async () => Promise.reject(new TypeError('Failed to fetch'))))
    expect(await kindOf(loadIndex())).toBe('network')
  })

  it('reads a body that is not the layout of the format as a format error', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => new Response('<html>', { status: 200 })))
    expect(await kindOf(loadMeta())).toBe('format')
    serve({ 'index.json': { families: 1 }, 'files/X.json': { path: 'X.json', family: 'x' } })
    clearDataCache()
    expect(await kindOf(loadIndex())).toBe('format')
    expect(await kindOf(loadFile('X.json'))).toBe('format')
  })

  it('fetches a path once, however often and however concurrently it is asked for', async () => {
    const fetchMock = serve()
    const [a, b] = await Promise.all([loadFile('StoryData/S9999.json'), loadFile('StoryData/S9999.json')])
    const c = await loadFile('StoryData/S9999.json')
    await loadMeta()
    await loadMeta()
    expect(a).toBe(b)
    expect(a).toBe(c)
    expect(fetchMock).toHaveBeenCalledTimes(2)
  })

  it('does not keep a failure: the next call asks again', async () => {
    const fetchMock = vi.fn(async () => new Response('', { status: 503 }))
    vi.stubGlobal('fetch', fetchMock)
    expect(await kindOf(loadMeta())).toBe('network')
    vi.stubGlobal('fetch', vi.fn(fixtureFetch()))
    expect((await loadMeta()).format).toBe(2)
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  it('makes the format error a DataError that says what it is', async () => {
    const e = new DataError('format', 'x')
    expect(e).toBeInstanceOf(Error)
    expect(e.kind).toBe('format')
    expect(e.name).toBe('DataError')
  })
})

describe('update loading', () => {
  const tag = 'zh-TW-20261105.1'
  const record = { tag, entries: [] }

  it('reads data/updates/<tag>.json once', async () => {
    const fetchMock = serve({ [`updates/${tag}.json`]: record })
    expect(await loadUpdate(tag)).toEqual(record)
    await loadUpdate(tag)
    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(fetchMock.mock.calls[0]?.[0]).toBe(`./data/updates/${tag}.json`)
  })

  it('reads data/updates/index.json', async () => {
    const fetchMock = serve({ 'updates/index.json': { releases: [] } })
    expect(await loadUpdates()).toEqual({ releases: [] })
    expect(fetchMock.mock.calls[0]?.[0]).toBe('./data/updates/index.json')
  })

  it('rejects documents of another shape, and a missing one', async () => {
    serve({ 'updates/index.json': { nope: 1 }, [`updates/${tag}.json`]: { entries: 1 } })
    expect(await kindOf(loadUpdates())).toBe('format')
    expect(await kindOf(loadUpdate(tag))).toBe('format')
    expect(await kindOf(loadUpdate('zh-TW-20261106.1'))).toBe('missing')
  })
})

describe('glossary data', () => {
  beforeEach(() => {
    clearDataCache()
    vi.stubGlobal('fetch', vi.fn(fixtureFetch()))
  })
  afterEach(() => vi.unstubAllGlobals())

  it('loads the index of the glossary once', async () => {
    const a = await loadGlossary()
    expect(a.entries.length).toBeGreaterThan(0)
    expect(a.thresholds.hard_min_n).toBe(20)
    await loadGlossary()
    expect(fetch).toHaveBeenCalledTimes(1)
  })

  it('loads the file of an entry from its id', async () => {
    const f = await loadGlossaryEntry('glossary/aaaaaaaa/bbbbbbbb')
    expect(f.id).toBe('glossary/aaaaaaaa/bbbbbbbb')
    expect(fetch).toHaveBeenCalledWith('./data/glossary/aaaaaaaa-bbbbbbbb.json', expect.anything())
  })

  it('says missing for an entry with no file, and format for a file with no units', async () => {
    await expect(loadGlossaryEntry('glossary/00000000/00000000')).rejects.toMatchObject({ kind: 'missing' })
    vi.stubGlobal('fetch', vi.fn(fixtureFetch({ 'glossary/index.json': { entries: [] } })))
    clearDataCache()
    await expect(loadGlossary()).rejects.toMatchObject({ kind: 'format' })
  })
})

describe('preview loading', () => {
  it('reads preview.json and preview/<path> with the path encoded like a file', async () => {
    const shared = { rules_commit: 'abc', prompt: 'p', guides: [] }
    const pf = { family: 'x', glossary_check: '', terms: [], keywords: {}, units: [] }
    const fetchMock = serve({ 'preview.json': shared, 'preview/a b/c%d.json': pf })
    expect(await loadPreviewShared()).toEqual(shared)
    expect(await loadPreviewFile('a b/c%d.json')).toEqual(pf)
    expect(fetchMock.mock.calls.map((c) => c[0])).toEqual(['./data/preview.json', './data/preview/a%20b/c%25d.json'])
  })

  it('says format for a document of the wrong shape, missing for a 404', async () => {
    serve({ 'preview.json': { prompt: 'p' }, 'preview/x.json': { units: 1 } })
    expect(await kindOf(loadPreviewShared())).toBe('format')
    expect(await kindOf(loadPreviewFile('x.json'))).toBe('format')
    expect(await kindOf(loadPreviewFile('nope.json'))).toBe('missing')
  })
})
