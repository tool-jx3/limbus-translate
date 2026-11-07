import type { FileData, GlossaryFile, GlossaryIndex, GoldData, Index, Meta, PreviewFileWire, PreviewShared, ReviewData, ReviewTab, UpdateRecord, UpdateRelease } from './types'

/** The version of the layout and keys of the data that this page reads. */
const FORMAT = 2

/** Where the data is: next to the page, under ./data/. */
const ROOT = './data/'

/**
 * Why data could not be read. `format`: it is a version this page does not
 * know, or not what the layout says. `missing`: the server has no such file,
 * as after a new deployment replaced the data. `network`: no answer, or a
 * server error. For the first two a reload fetches a page that fits the data.
 */
export class DataError extends Error {
  readonly kind: 'format' | 'missing' | 'network'

  constructor(kind: 'format' | 'missing' | 'network', message?: string) {
    super(message ?? `data error: ${kind}`)
    this.name = 'DataError'
    this.kind = kind
  }
}

function isRecord(v: unknown): v is Record<string, unknown> {
  return typeof v === 'object' && v !== null && !Array.isArray(v)
}

const cache = new Map<string, Promise<unknown>>()

/** Forget what was loaded, so that the next load asks the server again. For tests. */
export function clearDataCache(): void {
  cache.clear()
}

/** Options of a load: `fresh` asks the server again and neither reads nor fills the cache. */
export interface LoadOptions {
  fresh?: boolean
}

/** One JSON file under ./data/, fetched once: later calls get the same promise. A failure is not kept. With `fresh` it is fetched anew and kept by no one. */
function load<T>(rel: string, check: (doc: unknown) => T, opts: LoadOptions = {}): Promise<T> {
  const cached = opts.fresh ? undefined : cache.get(rel)
  if (cached) return cached as Promise<T>
  const p = (async () => {
    let res: Response
    try {
      // no-cache: ask the server whether the file is still the same (a 304 costs little)
      // so that a page never mixes files of two deployments for the length of a max-age.
      res = await fetch(ROOT + rel, { cache: 'no-cache' })
    } catch (e) {
      throw new DataError('network', `${rel}: ${String(e)}`)
    }
    if (res.status === 404) throw new DataError('missing', `${rel}: HTTP 404`)
    if (!res.ok) throw new DataError('network', `${rel}: HTTP ${res.status}`)
    let doc: unknown
    try {
      doc = await res.json()
    } catch {
      throw new DataError('format', `${rel}: not JSON`)
    }
    return check(doc)
  })()
  if (opts.fresh) return p
  cache.set(rel, p)
  p.catch(() => {
    if (cache.get(rel) === p) cache.delete(rel)
  })
  return p
}

export function loadMeta(): Promise<Meta> {
  return load('meta.json', (doc) => {
    if (!isRecord(doc)) throw new DataError('format', 'meta.json: not an object')
    if (doc.format !== FORMAT) throw new DataError('format', `meta.json: format ${String(doc.format)}, this page reads ${FORMAT}`)
    return doc as unknown as Meta
  })
}

export function loadIndex(): Promise<Index> {
  return load('index.json', (doc) => {
    if (!isRecord(doc) || !Array.isArray(doc.families) || !Array.isArray(doc.files)) {
      throw new DataError('format', 'index.json: no families and files')
    }
    return doc as unknown as Index
  })
}

/** A file of data/files/, by its path in the index (such as "StoryData/S001A.json"). */
export function loadFile(path: string, opts?: LoadOptions): Promise<FileData> {
  const rel = `files/${path.split('/').map(encodeURIComponent).join('/')}`
  return load(rel, (doc) => {
    if (!isRecord(doc) || !Array.isArray(doc.units)) throw new DataError('format', `${rel}: no units`)
    return doc as unknown as FileData
  }, opts)
}

/** data/preview.json: the prompt, the guides and the rules commit every preview batch shares. */
export function loadPreviewShared(opts?: LoadOptions): Promise<PreviewShared> {
  return load('preview.json', (doc) => {
    if (!isRecord(doc) || !Array.isArray(doc.guides) || typeof doc.prompt !== 'string') {
      throw new DataError('format', 'preview.json: no prompt and guides')
    }
    return doc as unknown as PreviewShared
  }, opts)
}

/** data/preview/<path>, compact as written: what the batches of the units of a file have in common (see expandPreviewFile). */
export function loadPreviewFile(path: string, opts?: LoadOptions): Promise<PreviewFileWire> {
  const rel = `preview/${path.split('/').map(encodeURIComponent).join('/')}`
  return load(rel, (doc) => {
    if (!isRecord(doc) || !Array.isArray(doc.units)) throw new DataError('format', `${rel}: no units`)
    return doc as unknown as PreviewFileWire
  }, opts)
}

/** The key under which each review document holds its list. */
const REVIEW_LIST: Record<ReviewTab, string> = {
  suggestions: 'items',
  warnings: 'items',
  notes: 'items',
  todo: 'sections',
  scope: 'proposals',
}

/** A page of data/review/: the items of one tab. */
export function loadReview<T extends ReviewTab>(tab: T): Promise<ReviewData[T]> {
  const rel = `review/${tab}.json`
  return load(rel, (doc) => {
    if (!isRecord(doc) || !Array.isArray(doc[REVIEW_LIST[tab]])) throw new DataError('format', `${rel}: no ${REVIEW_LIST[tab]}`)
    return doc as unknown as ReviewData[T]
  })
}

/** data/updates/index.json: the releases, newest first. */
export function loadUpdates(): Promise<{ releases: UpdateRelease[] }> {
  return load('updates/index.json', (doc) => {
    if (!isRecord(doc) || !Array.isArray(doc.releases)) throw new DataError('format', 'updates/index.json: no releases')
    return doc as unknown as { releases: UpdateRelease[] }
  })
}

/** data/updates/<tag>.json: what one release changed. */
export function loadUpdate(tag: string): Promise<UpdateRecord> {
  const rel = `updates/${encodeURIComponent(tag)}.json`
  return load(rel, (doc) => {
    if (!isRecord(doc) || !Array.isArray(doc.entries)) throw new DataError('format', `${rel}: no entries`)
    return doc as unknown as UpdateRecord
  })
}

/** data/glossary/index.json: the entries of the glossary and the limits of the hard candidates. */
export function loadGlossary(): Promise<GlossaryIndex> {
  return load('glossary/index.json', (doc) => {
    if (!isRecord(doc) || !isRecord(doc.thresholds) || !Array.isArray(doc.entries)) {
      throw new DataError('format', 'glossary/index.json: no thresholds and entries')
    }
    return doc as unknown as GlossaryIndex
  })
}

/** data/glossary/<a>-<b>.json: the units of the entry glossary/<a>/<b>. */
export function loadGlossaryEntry(id: string): Promise<GlossaryFile> {
  const rel = `glossary/${id.replace(/^glossary\//, '').replaceAll('/', '-')}.json`
  return load(rel, (doc) => {
    if (!isRecord(doc) || !Array.isArray(doc.missing) || !Array.isArray(doc.used)) {
      throw new DataError('format', `${rel}: no missing and used`)
    }
    return doc as unknown as GlossaryFile
  })
}

/** Where the substring q is in the path, the label or a speaker of each file (any case): positions in index.files. An empty q is every file. */
export function filterFiles(index: Index, q: string): number[] {
  const needle = q.trim().toLowerCase()
  const all = index.files.map((_, n) => n)
  if (needle === '') return all
  return all.filter((n) => {
    const f = index.files[n]
    if (!f) return false
    return [f.path, f.label, ...f.speakers].some((s) => s.toLowerCase().includes(needle))
  })
}

/** data/gold.json: the gold sample with its texts, and the report of what is judged. */
export function loadGold(): Promise<GoldData> {
  return load('gold.json', (doc) => {
    if (!isRecord(doc) || !Array.isArray(doc.units)) throw new DataError('format', 'gold.json: no units')
    return doc as unknown as GoldData
  })
}
