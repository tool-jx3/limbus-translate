import { render } from 'preact'
import { act } from 'preact/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { clearDataCache } from '../data'
import { glossaryHref, termHref, type Route } from '../route'
import { fixture, fixtureFetch, runEffectsAtOnce } from '../testdata'
import type { GlossaryIndex, ScopeEntry } from '../types'
import { filterEntries, GlossaryView } from './GlossaryView'

type GlossaryRoute = Extract<Route, { kind: 'glossary' }>
const base = fixture<GlossaryIndex>('glossary/index.json')

const containers: HTMLElement[] = []
let undoEffects = () => {}

beforeEach(() => {
  undoEffects = runEffectsAtOnce()
  clearDataCache()
  location.hash = ''
  vi.stubGlobal('fetch', vi.fn(fixtureFetch()))
})

afterEach(() => {
  undoEffects()
  for (const c of containers.splice(0)) {
    render(null, c)
    c.remove()
  }
  vi.unstubAllGlobals()
})

const route = (rest: Partial<GlossaryRoute> = {}): GlossaryRoute => ({ kind: 'glossary', q: '', sort: 'missing', only: '', family: '', ...rest })

async function mount(r: GlossaryRoute = route(), docs: Record<string, unknown> = {}): Promise<HTMLElement> {
  vi.stubGlobal('fetch', vi.fn(fixtureFetch(docs)))
  const container = document.createElement('div')
  document.body.append(container)
  containers.push(container)
  render(<GlossaryView route={r} />, container)
  await vi.waitFor(() => expect(container.querySelector('.gl-table, .notice, .rv-empty')).not.toBeNull())
  return container
}

const rows = (el: HTMLElement) => [...el.querySelectorAll<HTMLElement>('.gl-table tbody tr')]
const krs = (el: HTMLElement) => rows(el).map((r) => r.querySelector('.gl-kr')?.textContent)
const cells = (r: HTMLElement) => [...r.querySelectorAll('td')].map((c) => c.textContent)

describe('GlossaryView', () => {
  it('lists every entry with its columns', async () => {
    const el = await mount()
    expect(krs(el)).toEqual(['대상', '공격', '보스', '전투 승리'])
    const heads = [...el.querySelectorAll('.gl-table thead th')].map((h) => h.textContent)
    // the column sorted by carries a direction mark
    expect(heads).toEqual(['原詞', '譯名', '檢查', '範圍', '出現', '用了', '沒用▼', '應用率'])
    const r = rows(el)[0]!
    expect(r.querySelector('.gl-zh')?.textContent).toBe('目標')
    expect(r.querySelector('.gl-also')?.textContent).toContain('對象')
    // n, used, missing, rate with one decimal
    expect(cells(r).slice(-4)).toEqual(['52', '40', '12', '76.9%'])
    expect(el.textContent).toContain('4 條')
  })

  it('shows check as hard, off, or as warn when it is left out', async () => {
    const docs = { 'glossary/index.json': { ...base, entries: base.entries.map((e, i) => (i === 0 ? { ...e, check: 'hard' } : e.kr === '공격' ? { ...e, check: 'off' } : e)) } }
    const el = await mount(route(), docs)
    const checks = Object.fromEntries(rows(el).map((r) => [r.querySelector('.gl-kr')?.textContent, r.querySelector('.gl-check')?.textContent]))
    expect(checks).toEqual({ 대상: 'hard', 공격: 'off', 보스: 'warn', '전투 승리': 'off' })
    const attack = rows(el).find((r) => r.querySelector('.gl-kr')?.textContent === '공격')!
    expect(attack.querySelector('.gl-check')?.classList.contains('gl-check-off')).toBe(true)
    expect(attack.querySelector('.gl-check')?.getAttribute('title')).toContain('不警告')
  })

  it('summarises the scope: files, not_files and match', async () => {
    const el = await mount()
    const boss = rows(el).find((r) => r.querySelector('.gl-kr')?.textContent === '보스')!
    const scope = boss.querySelector('.gl-scope')?.textContent ?? ''
    expect(scope).toContain('files：family:ally')
    expect(scope).toContain('not_files：group:StoryData/')
    expect(scope).not.toContain('match：')
    const first = rows(el)[0]!.querySelector('.gl-scope')?.textContent ?? ''
    expect(first).toBe('全部')
    const attack = rows(el).find((r) => r.querySelector('.gl-kr')?.textContent === '공격')!
    expect(attack.querySelector('.gl-scope')?.textContent).toBe('match：substring')
  })

  it('says 全部 for an entry with no files, no not_files and match word, the way Go writes it', async () => {
    const docs = { 'glossary/index.json': { ...base, entries: base.entries.map((e, i) => (i === 0 ? { ...e, match: 'word' } : e)) } }
    const el = await mount(route(), docs)
    expect(rows(el)[0]!.querySelector('.gl-scope')?.textContent).toBe('全部')
  })

  it('links each row to the entry', async () => {
    const el = await mount()
    const hrefs = rows(el).map((r) => r.querySelector('a')?.getAttribute('href'))
    expect(hrefs).toEqual(['#/g/aaaaaaaa/bbbbbbbb', '#/g/eeeeeeee/ffffffff', '#/g/cccccccc/dddddddd', '#/g/12121212/34343434'])
    expect(hrefs[0]).toBe(termHref('glossary/aaaaaaaa/bbbbbbbb'))
  })

  it('orders by the units without the rendering, most first, by default', async () => {
    const el = await mount()
    // 대상 12, 공격 5, 보스 0; then 전투 승리 (6), which is check: off
    expect(krs(el)).toEqual(['대상', '공격', '보스', '전투 승리'])
    expect(el.querySelector<HTMLSelectElement>('select[name=sort]')?.value).toBe('missing')
  })

  it('orders by KR or by rate when asked', async () => {
    expect(krs(await mount(route({ sort: 'kr' })))).toEqual(['공격', '대상', '보스', '전투 승리'])
    // lowest rate first: 대상 76.9, 공격 83.9, 보스 100; then 전투 승리 (25.0), which is check: off
    expect(krs(await mount(route({ sort: 'rate' })))).toEqual(['대상', '공격', '보스', '전투 승리'])
  })

  it('orders by rate, highest first, with rate-desc', async () => {
    const el = await mount(route({ sort: 'rate-desc' }))
    expect(krs(el)).toEqual(['보스', '공격', '대상', '전투 승리'])
    expect(el.querySelector<HTMLSelectElement>('select[name=sort]')?.value).toBe('rate-desc')
  })

  it('goes to the address of a sort that is picked', async () => {
    const el = await mount()
    const select = el.querySelector<HTMLSelectElement>('select[name=sort]')!
    select.value = 'kr'
    await act(() => {
      select.dispatchEvent(new Event('change', { bubbles: true }))
    })
    expect(location.hash).toBe(glossaryHref({ sort: 'kr' }))
  })

  it('lists only the hard candidates with the thresholds in words', async () => {
    const el = await mount(route({ only: 'hard' }))
    expect(krs(el)).toEqual(['보스'])
    expect(el.querySelector('.gl-hard-note')?.textContent).toContain('應用率高於 95%、出現至少 20 個單位')
  })

  it('takes the thresholds in that sentence from the data', async () => {
    const docs = { 'glossary/index.json': { ...base, thresholds: { ...base.thresholds, hard_rate: 0.875, hard_min_n: 7 } } }
    const el = await mount(route({ only: 'hard' }), docs)
    expect(el.querySelector('.gl-hard-note')?.textContent).toContain('應用率高於 87.5%、出現至少 7 個單位')
  })

  it('does not show the sentence outside the hard filter, and marks the candidates in the list', async () => {
    const el = await mount()
    expect(el.querySelector('.gl-hard-note')).toBeNull()
    const boss = rows(el).find((r) => r.querySelector('.gl-kr')?.textContent === '보스')!
    expect(boss.querySelector('.gl-hard')?.textContent).toBe('可考慮退回')
    expect(rows(el)[0]!.querySelector('.gl-hard')).toBeNull()
  })

  it('lists only the entries with a unit without the rendering, one of check: off after the others', async () => {
    const el = await mount(route({ only: 'missing' }))
    expect(krs(el)).toEqual(['대상', '공격', '전투 승리'])
    const last = rows(el).at(-1)!
    expect(last.querySelector('.gl-check')?.textContent).toBe('off')
  })

  it('filters by a family the entry hits', async () => {
    expect(krs(await mount(route({ family: 'story-main' })))).toEqual(['대상', '공격'])
    expect(krs(await mount(route({ family: 'ally' })))).toEqual(['대상', '공격', '보스', '전투 승리'])
  })

  it('searches the KR, the rendering, also and notes', async () => {
    expect(krs(await mount(route({ q: '보스' })))).toEqual(['보스'])
    expect(krs(await mount(route({ q: 'boss' })))).toEqual(['보스'])
    expect(krs(await mount(route({ q: '老大' })))).toEqual(['보스'])
    expect(krs(await mount(route({ q: '常譯成' })))).toEqual(['대상'])
  })

  it('says so when nothing matches', async () => {
    const el = await mount(route({ q: '沒有這個詞' }))
    expect(rows(el)).toHaveLength(0)
    expect(el.textContent).toContain('沒有符合的詞條')
  })

  it('shows an error when the index cannot be loaded', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => new Response('x', { status: 500 })))
    clearDataCache()
    const container = document.createElement('div')
    document.body.append(container)
    containers.push(container)
    render(<GlossaryView route={route()} />, container)
    await vi.waitFor(() => expect(container.querySelector('.notice')).not.toBeNull())
  })
})

describe('filterEntries, the sorts of the columns', () => {
  const mk = (kr: string, zh: string, o: Partial<ScopeEntry> = {}): ScopeEntry => ({ ...base.entries[0]!, id: `glossary/${kr}`, kr, zh, also: [], files: [], not_files: [], match: 'word', notes: '', check: 'warn', n: 10, used: 5, rate: 0.5, ...o })
  const f = (sort: Parameters<typeof filterEntries>[1]['sort']) => ({ q: '', sort, only: '' as const, family: '' })
  const order = (es: ScopeEntry[], sort: Parameters<typeof filterEntries>[1]['sort']) => filterEntries([...es], f(sort)).map((e) => e.kr)
  const es = [
    mk('b', 'Y', { check: 'hard', n: 30, used: 10, rate: 0.9, files: ['family:story-main'] }),
    mk('a', 'Z', { n: 10, used: 10, rate: 0.1 }),
    mk('c', 'X', { n: 20, used: 5, rate: 0.5, match: 'substring' }),
  ]

  it('orders by KR and ZH, either way', () => {
    expect(order(es, 'kr')).toEqual(['a', 'b', 'c'])
    expect(order(es, 'kr-desc')).toEqual(['c', 'b', 'a'])
    expect(order(es, 'zh')).toEqual(['c', 'b', 'a'])
    expect(order(es, 'zh-desc')).toEqual(['a', 'b', 'c'])
  })

  it('orders by the check as shown, hard before warn', () => {
    expect(order(es, 'check')).toEqual(['b', 'a', 'c'])
    expect(order(es, 'check-desc')).toEqual(['a', 'c', 'b'])
  })

  it('orders off between hard and warn, as its name sorts', () => {
    const withOff = [...es, mk('d', 'W', { check: 'off' })]
    expect(order(withOff, 'check')).toEqual(['b', 'd', 'a', 'c'])
  })

  it('puts the entries of check: off after the others in the sorts by units without the rendering and by rate, whichever way', () => {
    // d lacks the rendering in the most units and has the lowest rate, e is close behind; both are off.
    const withOff = [...es, mk('d', 'W', { check: 'off', n: 40, used: 0, rate: 0 }), mk('e', 'V', { check: 'off', n: 30, used: 3, rate: 0.1 })]
    expect(order(withOff, 'missing')).toEqual(['b', 'c', 'a', 'd', 'e'])
    expect(order(withOff, 'missing-asc')).toEqual(['a', 'c', 'b', 'e', 'd'])
    expect(order(withOff, 'rate')).toEqual(['a', 'c', 'b', 'd', 'e'])
    expect(order(withOff, 'rate-desc')).toEqual(['b', 'c', 'a', 'e', 'd'])
    // The other columns sort them with the rest.
    expect(order(withOff, 'kr')).toEqual(['a', 'b', 'c', 'd', 'e'])
    expect(order(withOff, 'n')).toEqual(['d', 'b', 'e', 'c', 'a'])
  })

  it('orders by the scope line', () => {
    // files：family:story-main (b), match：substring (c), 全部 (a)
    expect(order(es, 'scope')).toEqual(['b', 'c', 'a'])
    expect(order(es, 'scope-desc')).toEqual(['a', 'c', 'b'])
  })

  it('orders by n, used, missing and rate in the first direction, and back', () => {
    expect(order(es, 'n')).toEqual(['b', 'c', 'a'])
    expect(order(es, 'n-asc')).toEqual(['a', 'c', 'b'])
    expect(order(es, 'used')).toEqual(['a', 'b', 'c'])
    expect(order(es, 'used-asc')).toEqual(['c', 'a', 'b'])
    // missing: b 20, c 15, a 0
    expect(order(es, 'missing')).toEqual(['b', 'c', 'a'])
    expect(order(es, 'missing-asc')).toEqual(['a', 'c', 'b'])
    expect(order(es, 'rate')).toEqual(['a', 'c', 'b'])
    expect(order(es, 'rate-desc')).toEqual(['b', 'c', 'a'])
  })

  it('breaks a tie by KR then ZH, A to Z, whichever way the column goes', () => {
    const tied = [mk('b', 'B', { n: 5 }), mk('a', 'Z', { n: 5 }), mk('a', 'A', { n: 5 }), mk('c', 'A', { n: 9 })]
    const zhs = (sort: Parameters<typeof filterEntries>[1]['sort']) => filterEntries([...tied], f(sort)).map((e) => `${e.kr}${e.zh}`)
    expect(zhs('n')).toEqual(['cA', 'aA', 'aZ', 'bB'])
    expect(zhs('n-asc')).toEqual(['aA', 'aZ', 'bB', 'cA'])
  })
})

describe('GlossaryView, the sort by header', () => {
  const heads = (el: HTMLElement) => [...el.querySelectorAll<HTMLElement>('.gl-table thead th')]
  const click = (el: HTMLElement, name: string) => act(() => heads(el).find((h) => h.querySelector('button')?.textContent?.startsWith(name))!.querySelector('button')!.click())
  const sortOf = () => new URLSearchParams(location.hash.split('?')[1] ?? '').get('sort')

  it('puts a button in every header', async () => {
    const el = await mount()
    expect(heads(el).map((h) => h.querySelector('button')?.getAttribute('type'))).toEqual(Array(8).fill('button'))
  })

  it('sorts by another column in the direction of its first click', async () => {
    const el = await mount()
    await click(el, '出現')
    expect(sortOf()).toBe('n')
    await click(el, '原詞')
    expect(sortOf()).toBe('kr')
  })

  it('goes the other way on the column it is sorted by', async () => {
    const el = await mount(route({ sort: 'n' }))
    await click(el, '出現')
    expect(sortOf()).toBe('n-asc')
    const back = await mount(route({ sort: 'n-asc' }))
    await click(back, '出現')
    expect(sortOf()).toBe('n')
    const kr = await mount(route({ sort: 'kr' }))
    await click(kr, '原詞')
    expect(sortOf()).toBe('kr-desc')
  })

  it('goes back to the default address when missing is the sort picked again by its first click', async () => {
    const el = await mount(route({ sort: 'missing-asc' }))
    await click(el, '沒用')
    expect(location.hash).toBe(glossaryHref())
  })

  it('marks the sorted column with aria-sort and a hidden mark, and no other', async () => {
    const asc = await mount(route({ sort: 'rate' }))
    expect(heads(asc).map((h) => h.getAttribute('aria-sort'))).toEqual([null, null, null, null, null, null, null, 'ascending'])
    expect(heads(asc)[7]!.querySelector('span[aria-hidden=true]')?.textContent).toBe('▲')
    const desc = await mount(route({ sort: 'n' }))
    expect(heads(desc).map((h) => h.getAttribute('aria-sort'))).toEqual([null, null, null, null, 'descending', null, null, null])
    expect(heads(desc)[4]!.querySelector('span[aria-hidden=true]')?.textContent).toBe('▼')
    expect(desc.querySelectorAll('.gl-table thead [aria-hidden=true]')).toHaveLength(1)
  })

  it('lists two directions of each column in the select, the same values as the headers', async () => {
    const el = await mount()
    const opts = [...el.querySelectorAll<HTMLOptionElement>('select[name=sort] option')]
    expect(opts.map((o) => o.value)).toEqual(['kr', 'kr-desc', 'zh', 'zh-desc', 'check', 'check-desc', 'scope', 'scope-desc', 'n', 'n-asc', 'used', 'used-asc', 'missing', 'missing-asc', 'rate', 'rate-desc'])
    expect(opts.map((o) => o.textContent)).toEqual([
      '原詞（A→Z）', '原詞（Z→A）', '譯名（A→Z）', '譯名（Z→A）', '檢查（A→Z）', '檢查（Z→A）', '範圍（A→Z）', '範圍（Z→A）',
      '出現（多到少）', '出現（少到多）', '用了（多到少）', '用了（少到多）', '沒用（多到少）', '沒用（少到多）', '應用率（低到高）', '應用率（高到低）',
    ])
    for (const o of opts) {
      const el2 = await mount(route({ sort: o.value as GlossaryRoute['sort'] }))
      expect(el2.querySelectorAll('.gl-table thead [aria-sort]')).toHaveLength(1)
    }
  })
})
