import type { Lang } from './types'

/** Where the page is, as the hash of its URL says: #/, #/f/<path>?u=<pointer>, #/s?q=…&lang=…&re=1. */
export type Route =
  | { kind: 'home' }
  | { kind: 'file'; path: string; ptr?: string }
  | { kind: 'search'; q: string; lang: Lang; re: boolean }

const LANGS: readonly string[] = ['kr', 'jp', 'en', 'zh']

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
  if (pathname === '/s') {
    const lang = query.get('lang') ?? ''
    return {
      kind: 'search',
      q: query.get('q') ?? '',
      lang: LANGS.includes(lang) ? (lang as Lang) : 'zh',
      re: query.get('re') === '1',
    }
  }
  return { kind: 'home' }
}

/** The hash of a file, and of a unit of it when ptr is given. Each segment of the path is encoded; the slashes stay. */
export function fileHref(path: string, ptr?: string): string {
  const base = `#/f/${path.split('/').map(encodeURIComponent).join('/')}`
  return ptr ? `${base}?u=${encodeURIComponent(ptr)}` : base
}

/** The hash of a full-text search. */
export function searchHref(q: string, lang: Lang, re: boolean): string {
  return `#/s?q=${encodeURIComponent(q)}&lang=${lang}${re ? '&re=1' : ''}`
}
