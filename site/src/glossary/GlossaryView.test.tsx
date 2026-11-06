import { render } from 'preact'
import { act } from 'preact/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { clearDataCache } from '../data'
import { glossaryHref, termHref, type Route } from '../route'
import { fixture, fixtureFetch, runEffectsAtOnce } from '../testdata'
import type { GlossaryIndex } from '../types'
import { GlossaryView } from './GlossaryView'

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
    expect(krs(el)).toEqual(['대상', '공격', '보스'])
    const heads = [...el.querySelectorAll('.gl-table thead th')].map((h) => h.textContent)
    expect(heads).toEqual(['原詞', '譯名', '檢查', '範圍', '出現', '用了', '沒用', '應用率'])
    const r = rows(el)[0]!
    expect(r.querySelector('.gl-zh')?.textContent).toBe('目標')
    expect(r.querySelector('.gl-also')?.textContent).toContain('對象')
    // n, used, missing, rate with one decimal
    expect(cells(r).slice(-4)).toEqual(['52', '40', '12', '76.9%'])
    expect(el.textContent).toContain('3 條')
  })

  it('shows check as hard, or as warn when it is left out', async () => {
    const docs = { 'glossary/index.json': { ...base, entries: base.entries.map((e, i) => (i === 0 ? { ...e, check: 'hard' } : e)) } }
    const el = await mount(route(), docs)
    const checks = Object.fromEntries(rows(el).map((r) => [r.querySelector('.gl-kr')?.textContent, r.querySelector('.gl-check')?.textContent]))
    expect(checks).toEqual({ 대상: 'hard', 공격: 'warn', 보스: 'warn' })
  })

  it('summarises the scope: files, not_files and match', async () => {
    const el = await mount()
    const boss = rows(el).find((r) => r.querySelector('.gl-kr')?.textContent === '보스')!
    const scope = boss.querySelector('.gl-scope')?.textContent ?? ''
    expect(scope).toContain('files：family:skill')
    expect(scope).toContain('not_files：group:StoryData/')
    expect(scope).toContain('match：word')
    const first = rows(el)[0]!.querySelector('.gl-scope')?.textContent ?? ''
    expect(first).toContain('全部')
  })

  it('links each row to the entry', async () => {
    const el = await mount()
    const hrefs = rows(el).map((r) => r.querySelector('a')?.getAttribute('href'))
    expect(hrefs).toEqual(['#/g/aaaaaaaa/bbbbbbbb', '#/g/eeeeeeee/ffffffff', '#/g/cccccccc/dddddddd'])
    expect(hrefs[0]).toBe(termHref('glossary/aaaaaaaa/bbbbbbbb'))
  })

  it('orders by the units without the rendering, most first, by default', async () => {
    const el = await mount()
    // 대상 12, 공격 5, 보스 0
    expect(krs(el)).toEqual(['대상', '공격', '보스'])
    expect(el.querySelector<HTMLSelectElement>('select[name=sort]')?.value).toBe('missing')
  })

  it('orders by KR or by rate when asked', async () => {
    expect(krs(await mount(route({ sort: 'kr' })))).toEqual(['공격', '대상', '보스'])
    // lowest rate first: 대상 76.9, 공격 83.9, 보스 100
    expect(krs(await mount(route({ sort: 'rate' })))).toEqual(['대상', '공격', '보스'])
  })

  it('orders by rate, highest first, with rate-desc', async () => {
    const el = await mount(route({ sort: 'rate-desc' }))
    expect(krs(el)).toEqual(['보스', '공격', '대상'])
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

  it('lists only the entries with a unit without the rendering', async () => {
    expect(krs(await mount(route({ only: 'missing' })))).toEqual(['대상', '공격'])
  })

  it('filters by a family the entry hits', async () => {
    expect(krs(await mount(route({ family: 'story-main' })))).toEqual(['대상', '공격'])
    expect(krs(await mount(route({ family: 'skill' })))).toEqual(['대상', '공격', '보스'])
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
