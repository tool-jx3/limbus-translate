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
    expect([...card.querySelectorAll('.sc-patterns code')].map((c) => c.textContent)).toEqual(['family:story-main', 'group:StoryData/'])
    expect(card.textContent).toContain('另有混用：group:Passives')
    expect(card.querySelector('.sc-effect')?.textContent).toBe('移除警告 11、少檢查 5')
    // the table is the entry's: group, n, used, rate, class
    const rows = [...card.querySelectorAll('table.sc-table tbody tr')].map((tr) => [...tr.children].map((c) => c.textContent))
    expect(rows).toEqual([
      ['family:skill', '40', '38', '95%', '有用'],
      ['family:story-main', '12', '2', '17%', '沒用'],
      ['group:StoryData/', '14', '3', '21%', '沒用'],
      ['group:Passives', '2', '1', '50%', '太少'],
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
    await vi.waitFor(() =>
      expect(writeText).toHaveBeenCalledWith('scope/aaaaaaaa/bbbbbbbb/not-files「대상 → 目標」（kind: not-files；patterns: family:story-main, group:StoryData/）→ '),
    )
  })

  it('copies the kind, patterns and match of a proposal with its id, and the match alone when it has no patterns', async () => {
    const writeText = vi.fn(async () => {})
    vi.stubGlobal('navigator', { clipboard: { writeText } })
    const el = await mount()
    const card = sections(el)[2]!.querySelector('article[data-id="scope/eeeeeeee/ffffffff/substring"]')!
    act(() => card.querySelector<HTMLButtonElement>('.rv-copy')!.click())
    await vi.waitFor(() => expect(writeText).toHaveBeenCalledWith('scope/eeeeeeee/ffffffff/substring「공격 → 攻擊」（kind: substring；match: substring）→ '))
    // a changed proposal changes the copied text
    const card2 = sections(el)[1]!.querySelector('article')!
    act(() => card2.querySelector<HTMLButtonElement>('.rv-copy')!.click())
    await vi.waitFor(() => expect(writeText).toHaveBeenLastCalledWith('scope/cccccccc/dddddddd/files「보스 → BOSS」（kind: files；patterns: family:skill）→ '))
  })

  it('filters the proposals by the text of the address', async () => {
    const el = await mount(route({ q: '보스' }))
    expect(cards(el)).toHaveLength(2)
  })

  describe('需要人看 collapsed', () => {
    type Doc = { thresholds: object; entries: { id: string; kr: string; zh: string; n: number; used: number; groups: { name: string; kind: string; n: number; used: number }[] }[]; proposals: { kind: string; id: string }[] }
    // Proposal i has i+1 units without the rendering, so the last one ranks lowest.
    function twelve(): Doc {
      const base = fixture<Doc>('review/scope.json')
      const entries = Array.from({ length: 12 }, (_, i) => ({
        ...base.entries[0]!,
        id: `glossary/m${i}/z`,
        kr: `혼합${i}`,
        zh: `混${i}`,
        n: 10,
        used: 10 - (i + 1),
        groups: [{ name: 'family:skill', kind: 'family', n: 10, used: 10 - (i + 1) }],
      }))
      const proposals = entries.map((e) => ({ ...base.proposals[4]!, id: `scope/m${e.id}/mixed`, kr: e.kr, zh: e.zh }))
      return { ...base, entries, proposals }
    }
    const mixed = (el: HTMLElement) => [...sections(el)[3]!.querySelectorAll<HTMLElement>('article')]

    it('shows the top 10 by units without the rendering and a button for the rest', async () => {
      vi.stubGlobal('fetch', vi.fn(fixtureFetch({ 'review/scope.json': twelve() })))
      const el = await mount()
      expect(sections(el)[3]!.querySelector('h2')?.textContent).toBe('需要人看 12 項')
      let shown = mixed(el)
      expect(shown).toHaveLength(10)
      expect(shown[0]!.querySelector('h3')?.textContent).toContain('혼합11')
      expect(shown[9]!.querySelector('h3')?.textContent).toContain('혼합2')
      const button = sections(el)[3]!.querySelector<HTMLButtonElement>('.sc-more-mixed')!
      expect(button.textContent).toBe('顯示更多（共 12）')
      act(() => button.click())
      shown = mixed(el)
      expect(shown).toHaveLength(12)
      expect(sections(el)[3]!.querySelector('.sc-more-mixed')).toBeNull()
    })

    it('ranks by the entry\'s own n - used, not by the sum of its groups (each hit is in two groups)', async () => {
      const doc = twelve()
      // 혼합0 lacks the rendering in 1 unit, but its groups (a family and a node group) add up to 20 such units.
      doc.entries[0]!.groups = [
        { name: 'family:skill', kind: 'family', n: 10, used: 0 },
        { name: 'group:Skills', kind: 'group', n: 10, used: 0 },
      ]
      vi.stubGlobal('fetch', vi.fn(fixtureFetch({ 'review/scope.json': doc })))
      const el = await mount()
      const shown = mixed(el)
      expect(shown[0]!.querySelector('h3')?.textContent).toContain('혼합11')
      expect(shown.map((a) => a.querySelector('h3')?.textContent).some((t) => t?.includes('혼합0'))).toBe(false)
    })

    it('folds back when the filter changes', async () => {
      vi.stubGlobal('fetch', vi.fn(fixtureFetch({ 'review/scope.json': twelve() })))
      const el = await mount()
      act(() => sections(el)[3]!.querySelector<HTMLButtonElement>('.sc-more-mixed')!.click())
      expect(mixed(el)).toHaveLength(12)
      act(() => {
        render(<ReviewView route={route({ q: '혼합' })} index={index} meta={meta} settings={defaultSettings()} />, el)
      })
      expect(mixed(el)).toHaveLength(10)
      expect(sections(el)[3]!.querySelector('.sc-more-mixed')).not.toBeNull()
    })

    it('folds back at once when the 顯示已確認 switch changes', async () => {
      vi.stubGlobal('fetch', vi.fn(fixtureFetch({ 'review/scope.json': twelve() })))
      const el = await mount()
      act(() => sections(el)[3]!.querySelector<HTMLButtonElement>('.sc-more-mixed')!.click())
      expect(mixed(el)).toHaveLength(12)
      act(() => {
        render(<ReviewView route={route({ all: true })} index={index} meta={meta} settings={defaultSettings()} />, el)
      })
      expect(mixed(el)).toHaveLength(10)
    })

    it('shows the one the address names even when it is outside the top 10', async () => {
      vi.stubGlobal('fetch', vi.fn(fixtureFetch({ 'review/scope.json': twelve() })))
      const el = await mount(route({ id: 'scope/mglossary/m0/z/mixed' }))
      expect(mixed(el)).toHaveLength(11)
      expect(el.querySelector('article.is-target')?.querySelector('h3')?.textContent).toContain('혼합0')
      expect(el.querySelector('.rv-missing')).toBeNull()
    })
  })

  it('explains 太少 and 中間 on the cell, each in its own words, with the real min_n', async () => {
    const base = fixture<{ entries: { kr: string; groups: object[] }[] }>('review/scope.json')
    const entries = base.entries.map((e, i) => (i === 0 ? { ...e, groups: [...e.groups, { name: 'family:mirror', kind: 'family', n: 4, used: 3 }] } : e))
    vi.stubGlobal('fetch', vi.fn(fixtureFetch({ 'review/scope.json': { ...base, entries } })))
    const el = await mount()
    const cell = (cls: string) => el.querySelector<HTMLElement>(`.sc-class-${cls}`)!.closest('td')!
    expect(cell('few').title).toBe('出現不到 3 個單位，不計入建議')
    expect(cell('between').title).toBe('未用譯名的不到 3 個單位，不計入建議')
    expect(cell('uses').title).toBe('')
    expect(cell('mixed').title).toBe('')
  })

  it('takes the number in the tooltips from the thresholds', async () => {
    const base = fixture<{ thresholds: object; entries: { groups: object[] }[] }>('review/scope.json')
    const entries = base.entries.map((e, i) => (i === 0 ? { ...e, groups: [...e.groups, { name: 'family:mirror', kind: 'family', n: 4, used: 3 }] } : e))
    vi.stubGlobal('fetch', vi.fn(fixtureFetch({ 'review/scope.json': { ...base, thresholds: { ...base.thresholds, min_n: 4 }, entries } })))
    const el = await mount()
    const cell = (cls: string) => el.querySelector<HTMLElement>(`.sc-class-${cls}`)!.closest('td')!
    expect(cell('few').title).toBe('出現不到 4 個單位，不計入建議')
    expect(cell('between').title).toBe('未用譯名的不到 4 個單位，不計入建議')
  })

  it('explains every threshold in an always-visible block, with the values of the data', async () => {
    let el = await mount()
    const legend = el.querySelector<HTMLElement>('.sc-legend')!
    expect(legend.closest('details')).toBeNull()
    const lines = [...legend.querySelectorAll('li')].map((li) => li.textContent)
    expect(lines).toEqual([
      '群組：族群（family:）與頂層分群（group:）；少於 3 個單位的群組不計入建議。',
      '有用：80% 以上的單位用了譯名；沒用：30% 以下。',
      '混用：介於 30% 到 80% 之間，且沒用譯名的單位至少 3 個；不到 3 個算中間，不計入建議。',
      '詞中詞：複合詞單位至少 3 個且 80% 以上用了譯名，建議 substring；已是 substring 而 30% 以下用了譯名，建議 word。',
      '常被說明偏離：沒用譯名而註記說明的單位至少 3 個。',
      '失去檢查偏多：不再檢查的用了譯名的單位至少 20 個，或佔該條目用了譯名的 10% 以上；這樣的建議預設不採用。',
    ])
    expect(el.querySelector('.sc-flag')?.getAttribute('title')).toBe('不再檢查的用了譯名的單位至少 20 個，或佔 10% 以上；預設不採用')
    const base = fixture<{ thresholds: object }>('review/scope.json')
    const thresholds = { min_n: 5, used: 0.9, not_used: 0.2, explained: 4, lost_units: 30, lost_share: 0.25, hard_rate: 0.95, hard_min_n: 20 }
    vi.stubGlobal('fetch', vi.fn(fixtureFetch({ 'review/scope.json': { ...base, thresholds } })))
    clearDataCache()
    el = await mount()
    const text = el.querySelector('.sc-legend')!.textContent!
    for (const want of ['少於 5 個單位', '90% 以上', '20% 以下', '20% 到 90%', '至少 5 個', '至少 4 個', '至少 30 個', '25% 以上']) {
      expect(text).toContain(want)
    }
    expect(text).not.toMatch(/80%|30% 以下|至少 3 個|至少 20 個|10%/)
  })

  it('links a card to its entry, which opens and is marked when the address names it', async () => {
    let el = await mount()
    const link = sections(el)[0]!.querySelector<HTMLAnchorElement>('article a.sc-entry-link')!
    expect(link.textContent).toBe('看條目')
    expect(link.getAttribute('href')).toBe(reviewHref('scope', { id: 'glossary/aaaaaaaa/bbbbbbbb' }))
    el = await mount(route({ id: 'glossary/aaaaaaaa/bbbbbbbb' }))
    const entry = el.querySelector<HTMLDetailsElement>('.sc-entry.is-target')!
    expect(entry.getAttribute('data-id')).toBe('glossary/aaaaaaaa/bbbbbbbb')
    expect(entry.open).toBe(true)
    expect(el.querySelector('.rv-missing')).toBeNull()
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
      expect(second).toContain('group:StoryData/')
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
      expect(row.textContent).toContain('有說明 4、沒說明 8')
      expect(row.textContent).toContain('story 裡常譯成「對象」。')
      expect(row.querySelector('.rv-id code')?.textContent).toBe('glossary/aaaaaaaa/bbbbbbbb')
      act(() => row.querySelector<HTMLButtonElement>('.rv-copy')!.click())
      await vi.waitFor(() => expect(writeText).toHaveBeenCalledWith('glossary/aaaaaaaa/bbbbbbbb「대상 → 目標」→ '))
    })

    it('shows the examples of the groups that do not use the rendering when the entry is expanded', async () => {
      const el = await mount()
      const row = rows(el)[0]!
      const details = row.querySelector<HTMLElement>('.sc-entry-body details.sc-examples')!
      expect(details.querySelector('summary')?.textContent).toBe('沒用譯名的例子（3）')
      const items = [...details.querySelectorAll('li')]
      expect(items.map((li) => li.querySelector('.rv-where')?.textContent)).toEqual(['family:story-main・', 'family:story-main・', 'group:StoryData/・'])
      expect(items[0]!.textContent).toContain('저 대상은 누구지?')
      expect(items[0]!.textContent).toContain('那個對象是誰？')
      expect(items[0]!.querySelector('a')?.getAttribute('href')).toContain('S9001')
      // an entry without examples has no such block
      expect(rows(el)[1]!.querySelector('.sc-entry-body details.sc-examples')).toBeNull()
    })

    it('keeps the entry the address names in the list even when the search filters it out', async () => {
      const el = await mount(route({ id: 'glossary/cccccccc/dddddddd' }))
      const search = el.querySelector<HTMLInputElement>('input[name=entry-q]')!
      search.value = '攻擊'
      act(() => {
        search.dispatchEvent(new Event('input', { bubbles: true }))
      })
      expect(rows(el).map((r) => r.getAttribute('data-id'))).toEqual(['glossary/cccccccc/dddddddd', 'glossary/eeeeeeee/ffffffff'])
      expect(el.querySelector('.sc-entry.is-target')?.getAttribute('data-id')).toBe('glossary/cccccccc/dddddddd')
    })

    it('shows an entry beyond the first 50 when the address names it', async () => {
      const base = fixture<{ entries: { id: string }[] }>('review/scope.json')
      const many = Array.from({ length: 120 }, (_, i) => ({ ...base.entries[0]!, id: `glossary/x${i}/y`, kr: `단어${i}` }))
      vi.stubGlobal('fetch', vi.fn(fixtureFetch({ 'review/scope.json': { ...base, entries: many } })))
      const el = await mount(route({ id: 'glossary/x77/y' }))
      expect(el.querySelector<HTMLDetailsElement>('.sc-entry.is-target')?.open).toBe(true)
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
