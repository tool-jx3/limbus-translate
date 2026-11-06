import { render } from 'preact'
import { act } from 'preact/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { clearDataCache } from '../data'
import { fileHref } from '../route'
import { fixture, fixtureFetch, runEffectsAtOnce } from '../testdata'
import type { Index, Lang } from '../types'
import { createSearchClient, type SearchClient } from './client'
import { LIMIT, type Reply } from './engine'
import { SearchView } from './SearchView'
import { InProcessWorker } from './testworker'

const index = fixture<Index>('index.json')
const STORY = 'StoryData/S9999.json'
const SKILLS = 'Skills_personality-02.json'

const containers: HTMLElement[] = []
let undoEffects = () => {}
let fetchData: ReturnType<typeof vi.fn<(url: string) => Promise<Response>>>

/** The fetch of the page, answering the absolute URLs the worker asks for from the fixtures. */
function stubFetch(docs: Record<string, unknown> = {}): void {
  const base = new URL('data/', document.baseURI).href
  const fixtures = fixtureFetch(docs)
  fetchData = vi.fn((url: string) => fixtures(url.startsWith(base) ? `./data/${url.slice(base.length)}` : url))
  vi.stubGlobal('fetch', fetchData)
}

function fetched(): string[] {
  const base = new URL('data/', document.baseURI).href
  return fetchData.mock.calls.map((c) => c[0].replace(base, '').replace(/^\.\/data\//, ''))
}

beforeEach(() => {
  undoEffects = runEffectsAtOnce()
  clearDataCache()
  location.hash = ''
  stubFetch()
})

afterEach(() => {
  undoEffects()
  for (const container of containers.splice(0)) {
    render(null, container)
    container.remove()
  }
  vi.unstubAllGlobals()
})

interface Options {
  lang?: Lang
  re?: boolean
  client?: SearchClient
}

function mount(q: string, opts: Options = {}): HTMLElement {
  const container = document.createElement('div')
  document.body.append(container)
  containers.push(container)
  const client = opts.client ?? createSearchClient(() => new InProcessWorker())
  render(<SearchView q={q} lang={opts.lang ?? 'zh'} re={opts.re ?? false} index={index} client={client} />, container)
  return container
}

async function search(q: string, opts: Options = {}): Promise<HTMLElement> {
  const el = mount(q, opts)
  await vi.waitFor(() => expect(el.querySelector('[data-state="done"], [data-state="error"]')).not.toBeNull())
  return el
}

const text = (el: Element | null | undefined) => el?.textContent ?? ''

describe('SearchView', () => {
  it('groups the hits by file, with the total, a link to each hit, and the match in a <mark>', async () => {
    const el = await search('test', { lang: 'en' })
    expect(text(el.querySelector('.search-status'))).toContain('共 3 筆')
    const groups = [...el.querySelectorAll('.hit-group')]
    expect(groups.map((g) => text(g.querySelector('.group-path')))).toEqual([STORY, SKILLS])
    expect(groups.map((g) => text(g.querySelector('.group-count')))).toEqual(['1 筆', '2 筆'])
    expect(text(groups[0]?.querySelector('.group-label'))).toBe('分部入口附近')
    expect(groups[0]?.querySelector('.group-path')?.getAttribute('href')).toBe(fileHref(STORY))
    expect([...el.querySelectorAll('.hit-link')].map((a) => a.getAttribute('href'))).toEqual([
      fileHref(STORY, '/footnote'),
      fileHref(SKILLS, '/dataList/0/levelList/0/name'),
      fileHref(SKILLS, '/dataList/1/levelList/0/name'),
    ])
    // the match is marked, whatever its case, and only in the language that was searched
    expect([...el.querySelectorAll('mark')].map((m) => m.textContent)).toEqual(['test', 'Test', 'Test'])
    expect([...el.querySelectorAll('mark')].every((m) => m.closest('.hit-line')?.classList.contains('is-hit'))).toBe(true)
  })

  it('shows the string that holds the query, marked, in the language of the search, with the KR and the TW of the unit', async () => {
    const el = await search('警告', { lang: 'jp' })
    const hit = el.querySelector('.hit')
    const lines = [...(hit?.querySelectorAll('.hit-line') ?? [])]
    expect(lines.map((l) => l.getAttribute('data-lang'))).toEqual(['kr', 'jp', 'zh'])
    const jp = lines[1]
    expect(jp?.classList.contains('is-hit')).toBe(true)
    expect(jp?.querySelector('.hit-text')?.getAttribute('lang')).toBe('ja')
    expect(jp?.querySelector('mark')?.textContent).toBe('警告')
    expect(text(jp?.querySelector('.hit-text'))).toBe('警告\n[Burn]の数値が{0}以上だと危険です。')
    // KR and TW of the same unit come from its file
    await vi.waitFor(() => expect(text(lines[2]?.querySelector('.hit-text'))).toContain('數值達'))
    expect(text(lines[0]?.querySelector('.hit-text'))).toContain('수치가')
    expect(lines[0]?.querySelector('.hit-text')?.getAttribute('lang')).toBe('ko')
    expect(lines[2]?.querySelector('.hit-text')?.getAttribute('lang')).toBe('zh-Hant-TW')
    expect(lines[0]?.querySelector('mark')).toBeNull()
  })

  it('has only two lines where the search is in KR or in TW', async () => {
    const kr = await search('경고', { lang: 'kr' })
    expect([...kr.querySelectorAll('.hit-line')].map((l) => l.getAttribute('data-lang'))).toEqual(['kr', 'zh'])
    expect(kr.querySelector('.hit-line.is-hit')?.getAttribute('data-lang')).toBe('kr')
    const zh = await search('警告', { lang: 'zh' })
    expect([...zh.querySelectorAll('.hit-line')].map((l) => l.getAttribute('data-lang'))).toEqual(['kr', 'zh'])
    expect(zh.querySelector('.hit-line.is-hit')?.getAttribute('data-lang')).toBe('zh')
  })

  it('shows the hit before its file has come in', async () => {
    let release: () => void = () => {}
    const gate = new Promise<void>((resolve) => (release = resolve))
    const fixtures = fixtureFetch()
    const base = new URL('data/', document.baseURI).href
    vi.stubGlobal('fetch', async (url: string) => {
      if (url.includes('files/')) await gate
      return fixtures(url.startsWith(base) ? `./data/${url.slice(base.length)}` : url)
    })
    const el = await search('警告')
    expect(el.querySelector('.hit .is-hit mark')?.textContent).toBe('警告')
    expect(el.querySelector('.hit-wait')).not.toBeNull()
    release()
    await vi.waitFor(() => expect(el.querySelector('.hit-wait')).toBeNull())
    expect(text(el.querySelector('.hit-kr .hit-text'))).toContain('수치가')
  })

  it('names a hit by the field of its unit once the file is in: the last segment of the pointer of a string in a list is a number', async () => {
    let release: () => void = () => {}
    const gate = new Promise<void>((resolve) => (release = resolve))
    const unit = (n: number) => ({ p: `/dataList/0/tags/${n}`, f: 'tags', i: 0, kr: `태그${n}`, zh: `標籤${n}`, o: 'paratranz' })
    const fixtures = fixtureFetch({
      'search/keys.json': [
        [0, '/dataList/0/tags/0'],
        [0, '/dataList/0/tags/1'],
        [0, '/dataList/1/name'],
      ],
      'search/zh.json': ['標籤0', '標籤1', '標籤名'],
      [`files/${STORY}`]: { path: STORY, family: 'story-main', units: [unit(0), unit(1), { ...unit(2), p: '/dataList/1/name', f: 'name' }] },
    })
    const base = new URL('data/', document.baseURI).href
    vi.stubGlobal('fetch', async (url: string) => {
      if (url.includes('files/')) await gate
      return fixtures(url.startsWith(base) ? `./data/${url.slice(base.length)}` : url)
    })
    const el = await search('標籤')
    const fields = () => [...el.querySelectorAll('.hit-field')].map((f) => text(f))
    // until the file is in, the last segment is all there is
    expect(fields()).toEqual(['0', '1', 'name'])
    release()
    await vi.waitFor(() => expect(fields()).toEqual(['tags', 'tags', 'name']))
    // the pointer is still there, to read
    expect([...el.querySelectorAll('.hit-field')].map((f) => f.getAttribute('title'))).toEqual(['/dataList/0/tags/0', '/dataList/0/tags/1', '/dataList/1/name'])
  })

  it('names a hit by the last segment of its pointer where its file cannot be loaded', async () => {
    const fixtures = fixtureFetch({ 'search/keys.json': [[0, '/dataList/0/tags/0']], 'search/zh.json': ['標籤0'] })
    const base = new URL('data/', document.baseURI).href
    vi.stubGlobal('fetch', async (url: string) =>
      url.includes('files/') ? new Response('gone', { status: 404 }) : fixtures(url.startsWith(base) ? `./data/${url.slice(base.length)}` : url),
    )
    const el = await search('標籤')
    await vi.waitFor(() => expect(el.querySelector('.group-error')).not.toBeNull())
    expect(text(el.querySelector('.hit-field'))).toBe('0')
  })

  it('loads the file of a group, and no file that has no hit', async () => {
    const el = await search('但丁')
    await vi.waitFor(() => expect(el.querySelector('.hit-wait')).toBeNull())
    expect(fetched().filter((f) => f.startsWith('files/'))).toEqual([`files/${STORY}`])
    expect(text(el.querySelector('.hit-kr .hit-text'))).toContain('단테')
  })

  it('does not load the file of a group that is not on screen until it comes into view', async () => {
    const observed: { el: Element; callback: IntersectionObserverCallback }[] = []
    class FakeObserver {
      private readonly callback: IntersectionObserverCallback
      constructor(callback: IntersectionObserverCallback) {
        this.callback = callback
      }
      observe(el: Element) {
        observed.push({ el, callback: this.callback })
      }
      disconnect() {}
      unobserve() {}
    }
    vi.stubGlobal('IntersectionObserver', FakeObserver)
    const el = await search('測')
    await vi.waitFor(() => expect(observed).toHaveLength(2))
    expect(fetched().filter((f) => f.startsWith('files/'))).toEqual([])
    // the second group comes into view: its file is loaded, the first one's is not
    const second = observed[1]
    act(() => second?.callback([{ isIntersecting: true, target: second.el } as unknown as IntersectionObserverEntry], {} as IntersectionObserver))
    await vi.waitFor(() => expect(fetched().filter((f) => f.startsWith('files/'))).toEqual([`files/${SKILLS}`]))
    await vi.waitFor(() => expect(el.querySelectorAll('.hit-group')[1]?.querySelector('.hit-wait')).toBeNull())
    expect(el.querySelectorAll('.hit-group')[0]?.querySelector('.hit-wait')).not.toBeNull()
  })

  it('says 下載中 while the first search of a language downloads, and 搜尋中 when the corpus is there', async () => {
    const client = createSearchClient(() => new InProcessWorker())
    const first = mount('警告', { client })
    expect(text(first.querySelector('.search-status'))).toContain('下載中')
    await vi.waitFor(() => expect(first.querySelector('.hit-group')).not.toBeNull())
    expect(text(first.querySelector('.search-status'))).not.toContain('下載中')
    // the same language again: nothing to download
    const second = mount('測試', { client })
    expect(text(second.querySelector('.search-status'))).toContain('搜尋中')
    expect(text(second.querySelector('.search-status'))).not.toContain('下載中')
    await vi.waitFor(() => expect(second.querySelector('.hit-group')).not.toBeNull())
  })

  it('shows a message for no hit', async () => {
    const el = await search('沒有這個字串')
    expect(text(el.querySelector('.search-status'))).toContain('沒有找到')
    expect(el.querySelector('.hit-group')).toBeNull()
  })

  it('shows the error for a pattern that is not valid, and downloads nothing', async () => {
    const el = await search('(', { re: true })
    expect(text(el.querySelector('.notice[role="alert"]'))).toContain('正規表示式')
    expect(el.querySelector('.hit-group')).toBeNull()
    expect(fetched()).toEqual([])
  })

  it('shows the error where the data cannot be downloaded', async () => {
    vi.stubGlobal('fetch', async () => new Response('no', { status: 404 }))
    const el = await search('警告')
    expect(text(el.querySelector('.notice[role="alert"]'))).toContain('404')
    expect(text(el.querySelector('.notice[role="alert"]'))).toContain('請重新整理後再試')
  })

  it('searches by regular expression only when it is on', async () => {
    const off = await search('\\[\\w+\\]', { lang: 'en' })
    expect(text(off.querySelector('.search-status'))).toContain('沒有找到')
    const on = await search('\\[\\w+\\]', { lang: 'en', re: true })
    expect(text(on.querySelector('.search-status'))).toContain('共 2 筆')
    expect([...on.querySelectorAll('.is-hit mark')].map((m) => m.textContent)).toEqual(['[Burn]', '[OnSucceedAttack]', '[Burn]'])
  })

  it('keeps the offsets of a hit in a string that has tags: it marks the stripped text', async () => {
    const el = await search('數值達', { lang: 'zh' })
    expect(el.querySelector('.is-hit mark')?.textContent).toBe('數值達')
    expect(text(el.querySelector('.is-hit .hit-text'))).toBe('警告\n[Burn] 數值達 {0} 以上時很危險。')
  })

  it('shows what is in a hit as text: no string of the data becomes markup', async () => {
    const evil = '<img src=x onerror=alert(1)>'
    stubFetch({
      'search/keys.json': [[0, '/dataList/0/content']],
      'search/zh.json': [`甲${evil}乙`],
      [`files/${STORY}`]: { path: STORY, family: 'story-main', units: [{ p: '/dataList/0/content', f: 'content', i: 0, kr: evil, zh: `甲${evil}乙`, o: 'paratranz' }] },
    })
    const el = await search('甲')
    await vi.waitFor(() => expect(el.querySelector('.hit-wait')).toBeNull())
    expect(el.querySelector('img')).toBeNull()
    expect(text(el.querySelector('.is-hit .hit-text'))).toBe(`甲${evil}乙`)
  })

  it('shows up to 1000 hits and tells the total', async () => {
    const n = 1500
    const keys = Array.from({ length: n }, (_, k) => [k < 1200 ? 0 : 2, `/dataList/${k}/name`])
    stubFetch({
      'search/keys.json': keys,
      'search/zh.json': keys.map((_, k) => `名稱${k}`),
    })
    const el = await search('名稱')
    expect(text(el.querySelector('.search-status'))).toContain('共 1,500 筆')
    expect(text(el.querySelector('.search-status'))).toContain('前 1,000 筆')
    expect(el.querySelectorAll('.hit')).toHaveLength(LIMIT)
    expect([...el.querySelectorAll('.hit-group')].map((g) => g.querySelectorAll('.hit').length)).toEqual([1000])
  })

  it('asks for nothing when there is nothing to search for', () => {
    const el = mount('')
    expect(text(el.querySelector('.search-status'))).toContain('全文搜尋框')
    expect(fetchData).not.toHaveBeenCalled()
  })

  it('has a language choice and a regular-expression box that go to the search with the other choice kept', async () => {
    const el = await search('警告', { lang: 'zh', re: false })
    const select = el.querySelector<HTMLSelectElement>('select[name="lang"]')
    expect([...(select?.options ?? [])].map((o) => [o.value, o.textContent])).toEqual([
      ['kr', 'KR 韓文'],
      ['jp', 'JP 日文'],
      ['en', 'EN 英文'],
      ['zh', 'TW 繁體中文'],
    ])
    expect(select?.value).toBe('zh')
    act(() => {
      if (select) select.value = 'jp'
      select?.dispatchEvent(new Event('change', { bubbles: true }))
    })
    expect(location.hash).toBe('#/s?q=%E8%AD%A6%E5%91%8A&lang=jp')
    const box = el.querySelector<HTMLInputElement>('input[name="re"]')
    expect(box?.checked).toBe(false)
    act(() => box?.click())
    expect(location.hash).toBe('#/s?q=%E8%AD%A6%E5%91%8A&lang=zh&re=1')
  })

  it('turns regular expressions off again', async () => {
    const el = await search('\\d', { lang: 'en', re: true })
    const box = el.querySelector<HTMLInputElement>('input[name="re"]')
    expect(box?.checked).toBe(true)
    act(() => box?.click())
    expect(location.hash).toBe('#/s?q=%5Cd&lang=en')
  })

  it('shows the answer to the search it has now, not the one before', async () => {
    // the first reply comes after the second
    const replies: ((r: Reply) => void)[] = []
    const queries: { id: number; q: string }[] = []
    const client = createSearchClient(() => {
      const w = new InProcessWorker()
      w.postMessage = (m) => {
        queries.push(m)
        replies.push((r) => w.onmessage?.({ data: r } as MessageEvent<Reply>))
      }
      return w
    })
    const container = document.createElement('div')
    document.body.append(container)
    containers.push(container)
    const view = (q: string) => <SearchView q={q} lang="zh" re={false} index={index} client={client} />
    render(view('警告'), container)
    render(view('測試'), container)
    await vi.waitFor(() => expect(queries).toHaveLength(2))
    act(() => replies[1]?.({ id: queries[1]?.id ?? 0, hits: [{ file: 2, ptr: '/dataList/1/levelList/0/name', text: '測試', ranges: [[0, 2]] }], total: 1 }))
    act(() => replies[0]?.({ id: queries[0]?.id ?? 0, hits: [{ file: 0, ptr: '/dataList/0/place', text: '警告', ranges: [[0, 2]] }], total: 1 }))
    await vi.waitFor(() => expect(container.querySelector('.hit-group')).not.toBeNull())
    expect(container.querySelector('.group-path')?.textContent).toBe(SKILLS)
    expect(container.querySelector('mark')?.textContent).toBe('測試')
  })
})
