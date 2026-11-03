// The page's side of the search: one worker, one search at a time. A new search
// ends the one before it.
import type { Lang } from '../types'
import type { Hit, Query, Reply } from './engine'

/** What a search comes to: its hits, an error to show, or "a newer search took its place". */
export type SearchResult = { hits: Hit[]; total: number } | { error: string } | { superseded: true }

/** The part of a Worker the client uses, so that a test can stand one in. */
export interface SearchWorker {
  postMessage(message: Query): void
  onmessage: ((event: MessageEvent<Reply>) => void) | null
  onerror: ((event: ErrorEvent) => void) | null
  terminate?(): void
}

export interface SearchClient {
  /** Search language `lang` for q, as text or (re) as a regular expression. Never rejects. */
  search(q: string, lang: Lang, re: boolean): Promise<SearchResult>
  /** Whether the worker holds the corpus of `lang`: a search of it has been answered, so it downloads nothing. */
  has(lang: Lang): boolean
}

/** The worker script. After a build it is a file under assets/, found from this module. */
function defaultWorker(): SearchWorker {
  return new Worker(new URL('./worker.ts', import.meta.url), { type: 'module' })
}

/**
 * Where the data is, as an absolute URL: next to the page. The worker lives
 * under assets/ after a build, so a relative ./data/ would be found from there.
 */
function dataBase(): string {
  return new URL('data/', document.baseURI).href
}

/** A client of the worker that `make` makes; it is made at the first search. */
export function createSearchClient(make: () => SearchWorker = defaultWorker): SearchClient {
  let worker: SearchWorker | undefined
  let nextId = 0
  let current: { id: number; resolve: (result: SearchResult) => void } | undefined
  /** The language of each query that has no answer yet, by id. */
  const asked = new Map<number, Lang>()
  const loaded = new Set<Lang>()

  /** Ends the search in progress, if there is one. */
  function settle(result: SearchResult): void {
    const search = current
    current = undefined
    search?.resolve(result)
  }

  function start(): SearchWorker {
    if (worker) return worker
    const w = make()
    w.onmessage = (event) => {
      const reply = event.data
      const lang = asked.get(reply.id)
      asked.delete(reply.id)
      // A late answer to a search that was replaced still means the corpus came in.
      if (lang !== undefined && 'hits' in reply) loaded.add(lang)
      if (current?.id !== reply.id) return
      settle('error' in reply ? { error: reply.error } : { hits: reply.hits, total: reply.total })
    }
    w.onerror = () => {
      w.terminate?.()
      worker = undefined
      asked.clear()
      loaded.clear()
      settle({ error: '搜尋程式發生錯誤，請重新整理後再試' })
    }
    worker = w
    return w
  }

  return {
    search(q, lang, re) {
      settle({ superseded: true })
      return new Promise((resolve) => {
        let w: SearchWorker
        try {
          w = start()
        } catch {
          resolve({ error: '這個瀏覽器無法執行全文搜尋' })
          return
        }
        const id = ++nextId
        current = { id, resolve }
        asked.set(id, lang)
        w.postMessage({ id, q, lang, re, base: dataBase() })
      })
    },
    has: (lang) => loaded.has(lang),
  }
}

/** The client of the page. */
export const searchClient = createSearchClient()
