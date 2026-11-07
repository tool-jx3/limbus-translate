import type { JPENClass, Lang, ReviewTab, UpdateKind } from './types'
import type { UpdateFilter } from './updates/filter'

/**
 * Where the page is, as the hash of its URL says: #/, #/f/<path>?u=<pointer>, #/pv/<path>?u=<pointer> (the translation preview of a unit), #/s?q=…&lang=…&re=1,
 * #/r/<tab>?family=…&kind=…&q=…&all=1&id=… (sub is the kind of the query: of a suggestion, the code of a warning),
 * #/gold (the gold sample to judge),
 * #/u, #/u/<tag>?k=<kinds>&o=<origin>&f=<file>&copy=1,
 * #/g?q=…&sort=…&only=…&family=…, #/g/<a>/<b>?family=…&group=…&chapter=…&character=…&event=…&origin=…&q=… (the entry glossary/<a>/<b>).
 */
export type Route =
  | { kind: 'home' }
  | { kind: 'file'; path: string; ptr?: string }
  | { kind: 'preview'; path: string; ptr: string }
  | { kind: 'search'; q: string; lang: Lang; re: boolean }
  | { kind: 'review'; tab: ReviewTab; family: string; sub: string; q: string; all: boolean; mode: ReviewMode; jpen: JPENFilter; id?: string }
  | { kind: 'updates' }
  | { kind: 'gold' }
  | { kind: 'update'; tag: string; kinds: UpdateKind[]; origin: string; file: string; copy: boolean }
  | { kind: 'glossary'; q: string; sort: GlossarySort; only: GlossaryOnly; family: string }
  | { kind: 'term'; id: string; family: string; group: string; chapter: string; character: string; event: string; origin: string; q: string }

/** How the list of glossary entries is ordered: by units without the rendering (most first), by KR, or by rate. */
/** The columns of the glossary list that sort, each with the direction of its first click: counts go most first, text A to Z, rate lowest first. */
export const GLOSSARY_COLUMNS = [
  { id: 'kr', first: 'asc' },
  { id: 'zh', first: 'asc' },
  { id: 'check', first: 'asc' },
  { id: 'scope', first: 'asc' },
  { id: 'n', first: 'desc' },
  { id: 'used', first: 'desc' },
  { id: 'missing', first: 'desc' },
  { id: 'rate', first: 'asc' },
] as const
export type GlossaryColumn = (typeof GLOSSARY_COLUMNS)[number]['id']
export type SortDir = 'asc' | 'desc'
type SortValueOf<C> = C extends { id: infer I extends string; first: infer D } ? (D extends 'asc' ? I | `${I}-desc` : I | `${I}-asc`) : never
/** A sort of the glossary list: the id of a column for its first direction, the id with -desc or -asc for the other. */
export type GlossarySort = SortValueOf<(typeof GLOSSARY_COLUMNS)[number]>

const other = (d: SortDir): SortDir => (d === 'asc' ? 'desc' : 'asc')

/** The sort value of a column in a direction. */
export function glossarySort(id: GlossaryColumn, dir: SortDir): GlossarySort {
  const first = GLOSSARY_COLUMNS.find((c) => c.id === id)!.first
  return (dir === first ? id : `${id}-${dir}`) as GlossarySort
}

/** The column and direction a sort value stands for. */
export function sortParts(sort: GlossarySort): { id: GlossaryColumn; dir: SortDir } {
  for (const c of GLOSSARY_COLUMNS) {
    if (sort === c.id) return { id: c.id, dir: c.first }
    if (sort === `${c.id}-${other(c.first)}`) return { id: c.id, dir: other(c.first) }
  }
  return { id: 'missing', dir: 'desc' }
}

/** Every sort value, the columns in order, the first direction first. */
export const GLOSSARY_SORTS: readonly GlossarySort[] = GLOSSARY_COLUMNS.flatMap((c) => [glossarySort(c.id, c.first), glossarySort(c.id, other(c.first))])
/** Which entries the list keeps: all, those with a unit without the rendering, or the hard candidates. */
export type GlossaryOnly = '' | 'missing' | 'hard'

/** The filters of one glossary entry (the 'term' route without its id). */
export interface TermFilter {
  family: string
  group: string
  chapter: string
  character: string
  event: string
  origin: string
  q: string
}

/** How the inconsistent translations are shown: by KR alone, or with each variant's JP and EN. */
export type ReviewMode = 'kr' | 'jpen'
/** The JP/EN class the inconsistent translations are filtered by; '' for all. */
export type JPENFilter = '' | JPENClass

const JPEN_CLASSES: readonly string[] = ['explained', 'partial', 'same']

const LANGS: readonly string[] = ['kr', 'jp', 'en', 'zh']
const REVIEW_TABS: readonly string[] = ['suggestions', 'warnings', 'notes', 'todo', 'scope']
const UPDATE_KINDS: readonly string[] = ['new', 'kr-changed', 'deleted', 'zh-changed']
const GLOSSARY_ONLY: readonly string[] = ['missing', 'hard']
/** The <a>/<b> of glossary/<a>/<b>. */
const ENTRY_PATH = /^[0-9A-Za-z_-]+\/[0-9A-Za-z_-]+$/
const UPDATE_TAG = /^zh-TW-\d{8}\.\d+$/

/** decodeURIComponent that keeps the text as it is where the escape is not valid. */
function decode(s: string): string {
  try {
    return decodeURIComponent(s)
  } catch {
    return s
  }
}

function params(query: string): Map<string, string> {
  const out = new Map<string, string>()
  for (const pair of query.split('&')) {
    if (pair === '') continue
    const eq = pair.indexOf('=')
    const key = decode(eq < 0 ? pair : pair.slice(0, eq))
    if (!out.has(key)) out.set(key, eq < 0 ? '' : decode(pair.slice(eq + 1)))
  }
  return out
}

/** A file path names a file under data/files/: no empty segment, and no . or .. */
function isFilePath(path: string): boolean {
  return path.split('/').every((seg) => seg !== '' && seg !== '.' && seg !== '..')
}

/** The route of a location.hash. What is not a route of ours is home. */
export function parseHash(hash: string): Route {
  const h = hash.startsWith('#') ? hash.slice(1) : hash
  const cut = h.indexOf('?')
  const pathname = cut < 0 ? h : h.slice(0, cut)
  const query = params(cut < 0 ? '' : h.slice(cut + 1))

  if (pathname.startsWith('/f/')) {
    const path = pathname.slice(3).split('/').map(decode).join('/')
    if (!isFilePath(path)) return { kind: 'home' }
    const ptr = query.get('u')
    return ptr ? { kind: 'file', path, ptr } : { kind: 'file', path }
  }
  if (pathname.startsWith('/pv/')) {
    const path = pathname.slice(4).split('/').map(decode).join('/')
    if (!isFilePath(path)) return { kind: 'home' }
    const ptr = query.get('u')
    return ptr ? { kind: 'preview', path, ptr } : { kind: 'file', path }
  }
  if (pathname === '/s') {
    const lang = query.get('lang') ?? ''
    return {
      kind: 'search',
      q: query.get('q') ?? '',
      lang: LANGS.includes(lang) ? (lang as Lang) : 'zh',
      re: query.get('re') === '1',
    }
  }
  if (pathname === '/r' || pathname.startsWith('/r/')) {
    const tab = pathname === '/r' ? 'suggestions' : pathname.slice(3)
    if (!REVIEW_TABS.includes(tab)) return { kind: 'home' }
    const id = query.get('id')
    return {
      kind: 'review',
      tab: tab as ReviewTab,
      family: query.get('family') ?? '',
      sub: query.get('kind') ?? '',
      q: query.get('q') ?? '',
      all: query.get('all') === '1',
      mode: query.get('mode') === 'jpen' ? 'jpen' : 'kr',
      jpen: JPEN_CLASSES.includes(query.get('jpen') ?? '') ? (query.get('jpen') as JPENClass) : '',
      ...(id ? { id } : {}),
    }
  }
  if (pathname === '/g') {
    const sort = query.get('sort') ?? ''
    const only = query.get('only') ?? ''
    return {
      kind: 'glossary',
      q: query.get('q') ?? '',
      sort: (GLOSSARY_SORTS as readonly string[]).includes(sort) ? (sort as GlossarySort) : 'missing',
      only: GLOSSARY_ONLY.includes(only) ? (only as GlossaryOnly) : '',
      family: query.get('family') ?? '',
    }
  }
  if (pathname.startsWith('/g/')) {
    const rest = pathname.slice(3)
    if (!ENTRY_PATH.test(rest)) return { kind: 'home' }
    return {
      kind: 'term',
      id: `glossary/${rest}`,
      family: query.get('family') ?? '',
      group: query.get('group') ?? '',
      chapter: query.get('chapter') ?? '',
      character: query.get('character') ?? '',
      event: query.get('event') ?? '',
      origin: query.get('origin') ?? '',
      q: query.get('q') ?? '',
    }
  }
  if (pathname === '/gold') return { kind: 'gold' }
  if (pathname === '/u') return { kind: 'updates' }
  if (pathname.startsWith('/u/')) {
    const tag = decode(pathname.slice(3))
    if (!UPDATE_TAG.test(tag)) return { kind: 'home' }
    const kinds = (query.get('k') ?? '').split(',').filter((k): k is UpdateKind => UPDATE_KINDS.includes(k))
    return { kind: 'update', tag, kinds, origin: query.get('o') ?? '', file: query.get('f') ?? '', copy: query.get('copy') === '1' }
  }
  return { kind: 'home' }
}

/** The hash of a file, and of a unit of it when ptr is given. Each segment of the path is encoded; the slashes stay. */
export function fileHref(path: string, ptr?: string): string {
  const base = `#/f/${path.split('/').map(encodeURIComponent).join('/')}`
  return ptr ? `${base}?u=${encodeURIComponent(ptr)}` : base
}

/** The hash of the translation preview of a unit. */
export function previewHref(path: string, ptr: string): string {
  return `#/pv/${path.split('/').map(encodeURIComponent).join('/')}?u=${encodeURIComponent(ptr)}`
}

/** The hash of a full-text search. */
export function searchHref(q: string, lang: Lang, re: boolean): string {
  return `#/s?q=${encodeURIComponent(q)}&lang=${lang}${re ? '&re=1' : ''}`
}

/** The hash of a page of the review, with the filters and the item to scroll to that are given. */
export function reviewHref(
  tab: ReviewTab,
  opts: { family?: string; sub?: string; q?: string; all?: boolean; mode?: ReviewMode; jpen?: JPENFilter; id?: string } = {},
): string {
  const pairs: [string, string][] = []
  if (opts.family) pairs.push(['family', opts.family])
  if (opts.sub) pairs.push(['kind', opts.sub])
  if (opts.q) pairs.push(['q', opts.q])
  if (opts.all) pairs.push(['all', '1'])
  if (opts.mode === 'jpen') pairs.push(['mode', 'jpen'])
  if (opts.jpen) pairs.push(['jpen', opts.jpen])
  if (opts.id) pairs.push(['id', opts.id])
  const query = pairs.map(([k, v]) => `${k}=${encodeURIComponent(v)}`).join('&')
  return `#/r/${tab}${query ? `?${query}` : ''}`
}

/** The hash of the gold page. */
export function goldHref(): string {
  return '#/gold'
}

/** The hash of the list of updates. */
export function updatesHref(): string {
  return '#/u'
}

/** The hash of an update, with the filters that are not the default. */
export function updateHref(tag: string, f: Partial<UpdateFilter> = {}): string {
  const pairs: [string, string][] = []
  if (f.kinds && f.kinds.length > 0) pairs.push(['k', f.kinds.join(',')])
  if (f.origin) pairs.push(['o', encodeURIComponent(f.origin)])
  if (f.file) pairs.push(['f', encodeURIComponent(f.file)])
  if (f.copy) pairs.push(['copy', '1'])
  const query = pairs.map(([k, v]) => `${k}=${v}`).join('&')
  return `#/u/${encodeURIComponent(tag)}${query ? `?${query}` : ''}`
}

/** The hash of the list of glossary entries, with the filters that are not the default. */
export function glossaryHref(f: Partial<Pick<Extract<Route, { kind: 'glossary' }>, 'q' | 'sort' | 'only' | 'family'>> = {}): string {
  const pairs: [string, string][] = []
  if (f.q) pairs.push(['q', f.q])
  if (f.sort && f.sort !== 'missing') pairs.push(['sort', f.sort])
  if (f.only) pairs.push(['only', f.only])
  if (f.family) pairs.push(['family', f.family])
  const query = pairs.map(([k, v]) => `${k}=${encodeURIComponent(v)}`).join('&')
  return `#/g${query ? `?${query}` : ''}`
}

/** The hash of one glossary entry (its id is glossary/<a>/<b>), with the filters that are given. */
export function termHref(id: string, f: Partial<TermFilter> = {}): string {
  const pairs: [string, string][] = []
  for (const k of ['family', 'group', 'chapter', 'character', 'event', 'origin', 'q'] as const) {
    const v = f[k]
    if (v) pairs.push([k, v])
  }
  const query = pairs.map(([k, v]) => `${k}=${encodeURIComponent(v)}`).join('&')
  return `#/g/${id.replace(/^glossary\//, '')}${query ? `?${query}` : ''}`
}
