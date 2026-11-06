import { render } from 'preact'
import { act } from 'preact/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { clearDataCache } from '../data'
import { reviewHref, type Route } from '../route'
import { defaultSettings } from '../settings'
import { fixture, fixtureFetch, runEffectsAtOnce } from '../testdata'
import type { Index, Meta } from '../types'
import { ReviewView } from './ReviewView'

const meta = fixture<Meta>('meta.json')
const index = fixture<Index>('index.json')
type ReviewRoute = Extract<Route, { kind: 'review' }>

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

const route = (rest: Partial<ReviewRoute> = {}): ReviewRoute => ({ kind: 'review', tab: 'scope', family: '', sub: '', q: '', all: false, mode: 'kr', jpen: '', ...rest })

async function mount(r: ReviewRoute = route()): Promise<HTMLElement> {
  const container = document.createElement('div')
  document.body.append(container)
  containers.push(container)
  render(<ReviewView route={r} index={index} meta={meta} settings={defaultSettings()} />, container)
  await vi.waitFor(() => expect(container.querySelector('.rv-body, .notice')).not.toBeNull())
  return container
}

const sections = (el: HTMLElement) => [...el.querySelectorAll<HTMLElement>('section.sc-section')]
const cards = (el: HTMLElement) => [...el.querySelectorAll<HTMLElement>('article.rv-card')]

describe('ScopeView', () => {
  it('lists the five sections in order, each with its count and its proposals', async () => {
    const el = await mount()
    const heads = sections(el).map((s) => s.querySelector('h2')?.textContent)
    expect(heads).toEqual(['排除檔案 1 項', '限定檔案 1 項', '詞中詞 2 項', '需要人看 1 項', '常被說明偏離 1 項'])
    // the declined one is hidden
    expect(cards(el)).toHaveLength(6)
    expect(sections(el)[0]!.querySelectorAll('article')).toHaveLength(1)
    expect(sections(el)[2]!.querySelectorAll('article')).toHaveLength(2)
  })

  it('shows one muted line for an empty section', async () => {
    vi.stubGlobal('fetch', vi.fn(fixtureFetch({ 'review/scope.json': { ...fixture<object>('review/scope.json'), proposals: [] } })))
    const el = await mount()
    expect(sections(el)).toHaveLength(5)
    expect(sections(el).map((s) => s.querySelector('.rv-empty')?.textContent)).toEqual(Array(5).fill('沒有建議'))
  })

  it('shows on a card the title, patterns, mixed groups, effect, group table and examples', async () => {
    const el = await mount()
    const card = sections(el)[0]!.querySelector('article')!
    expect(card.querySelector('h3')?.textContent).toContain('대상 → 目標')
    expect(card.querySelector('.rv-kind')?.textContent).toBe('排除檔案')
    expect([...card.querySelectorAll('.sc-patterns code')].map((c) => c.textContent)).toEqual(['family:story-main', 'StoryData/**'])
    expect(card.textContent).toContain('另有混用：rest')
    expect(card.querySelector('.sc-effect')?.textContent).toBe('移除警告 11、少檢查 5')
    // the table is the entry's: group, n, used, rate, class
    const rows = [...card.querySelectorAll('table.sc-table tbody tr')].map((tr) => [...tr.children].map((c) => c.textContent))
    expect(rows).toEqual([
      ['family:skill', '40', '38', '95%', '有用'],
      ['family:story-main', '12', '2', '17%', '沒用'],
      ['StoryData/**', '14', '3', '21%', '沒用'],
      ['rest', '2', '1', '50%', '太少'],
    ])
    expect(card.querySelector('details.sc-examples')?.textContent).toContain('確認對象')
  })

  it('shows all four effect numbers when they are not zero, and the flagged badge', async () => {
    const el = await mount()
    const card = sections(el)[1]!.querySelector('article')!
    expect(card.querySelector('.sc-effect')?.textContent).toBe('移除警告 3、新增警告 1、少檢查 25')
    expect(card.querySelector('.sc-flag')?.textContent).toBe('失去檢查偏多')
    expect(sections(el)[0]!.querySelector('.sc-flag')).toBeNull()
  })

  it('labels a group between the limits with enough units lacking the rendering as 混用', async () => {
    const el = await mount()
    const card = sections(el)[3]!.querySelector('article')!
    expect(card.textContent).toContain('另有混用：family:story-main')
    expect(card.querySelector('.sc-effect')).toBeNull()
    const word = sections(el)[2]!.querySelectorAll('article')[1]!
    const rows = [...word.querySelectorAll('table.sc-table tbody tr')].map((tr) => [...tr.children].map((c) => c.textContent))
    expect(rows[1]).toEqual(['family:story-main', '8', '4', '50%', '混用'])
  })

  it('hides a declined proposal until 顯示已確認 is on, then shows it with 已否決', async () => {
    let el = await mount()
    const toggle = el.querySelector<HTMLInputElement>('.rv-all input[type=checkbox]')!
    expect(toggle.parentElement?.textContent).toContain('顯示已確認的 1 項')
    act(() => toggle.click())
    expect(location.hash).toBe(reviewHref('scope', { all: true }))

    el = await mount(route({ all: true }))
    expect(cards(el)).toHaveLength(7)
    const faded = el.querySelector('article.rv-card.is-confirmed')!
    expect(faded.textContent).toContain('已否決（2026-11-05）：劇情裡也要統一')
  })

  it('copies ID「摘要」→ for a proposal', async () => {
    const writeText = vi.fn(async () => {})
    vi.stubGlobal('navigator', { clipboard: { writeText } })
    const el = await mount()
    const card = sections(el)[0]!.querySelector('article')!
    expect(card.querySelector('.rv-id code')?.textContent).toBe('scope/aaaaaaaa/bbbbbbbb/not-files')
    act(() => card.querySelector<HTMLButtonElement>('.rv-copy')!.click())
    await vi.waitFor(() => expect(writeText).toHaveBeenCalledWith('scope/aaaaaaaa/bbbbbbbb/not-files「대상 → 目標」→ '))
  })

  it('filters the proposals by the text of the address', async () => {
    const el = await mount(route({ q: '보스' }))
    expect(cards(el)).toHaveLength(2)
  })

  describe('all entries', () => {
    const rows = (el: HTMLElement) => [...el.querySelectorAll<HTMLElement>('.sc-entry')]

    it('lists the entries with also, files, not_files, match and used/n, and searches KR, ZH and also', async () => {
      const el = await mount()
      expect(rows(el)).toHaveLength(3)
      const first = rows(el)[0]!.querySelector('summary')!
      expect(first.textContent).toContain('대상 → 目標')
      expect(first.textContent).toContain('對象')
      expect(first.textContent).toContain('44/68')
      const second = rows(el)[1]!.querySelector('summary')!.textContent!
      expect(second).toContain('老大')
      expect(second).toContain('family:skill')
      expect(second).toContain('StoryData/**')
      expect(second).toContain('word')

      const search = el.querySelector<HTMLInputElement>('input[name=entry-q]')!
      const type = (v: string) => {
        search.value = v
        act(() => {
          search.dispatchEvent(new Event('input', { bubbles: true }))
        })
      }
      type('老大')
      expect(rows(el)).toHaveLength(1)
      type('攻擊')
      expect(rows(el)).toHaveLength(1)
      type('대')
      expect(rows(el)).toHaveLength(1)
      type('沒有這個')
      expect(rows(el)).toHaveLength(0)
    })

    it('expands to the group table, the compound counts, explained/unexplained and a copy of the glossary ID', async () => {
      const writeText = vi.fn(async () => {})
      vi.stubGlobal('navigator', { clipboard: { writeText } })
      const el = await mount()
      const row = rows(el)[0]!
      expect(row.querySelectorAll('table.sc-table tbody tr')).toHaveLength(4)
      expect(row.textContent).toContain('複合詞 5 個單位，用了譯名 1')
      expect(row.textContent).toContain('有說明 4、沒說明 6')
      expect(row.textContent).toContain('story 裡常譯成「對象」。')
      expect(row.querySelector('.rv-id code')?.textContent).toBe('glossary/aaaaaaaa/bbbbbbbb')
      act(() => row.querySelector<HTMLButtonElement>('.rv-copy')!.click())
      await vi.waitFor(() => expect(writeText).toHaveBeenCalledWith('glossary/aaaaaaaa/bbbbbbbb「대상 → 目標」→ '))
    })

    it('shows the first 50 and more on request', async () => {
      const base = fixture<{ entries: { id: string }[] }>('review/scope.json')
      const many = Array.from({ length: 120 }, (_, i) => ({ ...base.entries[0]!, id: `glossary/x${i}/y`, kr: `단어${i}` }))
      vi.stubGlobal('fetch', vi.fn(fixtureFetch({ 'review/scope.json': { ...base, entries: many } })))
      const el = await mount()
      expect(rows(el)).toHaveLength(50)
      act(() => el.querySelector<HTMLButtonElement>('.sc-more')!.click())
      expect(rows(el)).toHaveLength(100)
    })
  })
})
