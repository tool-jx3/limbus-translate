import { render } from 'preact'
import { act } from 'preact/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { clearDataCache } from '../data'
import { fileHref, reviewHref, type Route } from '../route'
import { defaultSettings, type Settings } from '../settings'
import { fixture, fixtureFetch, runEffectsAtOnce } from '../testdata'
import type { Index, Meta, NoteItem } from '../types'
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

function route(tab: ReviewRoute['tab'], rest: Partial<ReviewRoute> = {}): ReviewRoute {
  return { kind: 'review', tab, family: '', sub: '', q: '', all: false, mode: 'kr', jpen: '', ...rest }
}

async function mount(r: ReviewRoute, settings: Settings = defaultSettings()): Promise<HTMLElement> {
  const container = document.createElement('div')
  document.body.append(container)
  containers.push(container)
  render(<ReviewView route={r} index={index} meta={meta} settings={settings} />, container)
  await vi.waitFor(() => expect(container.querySelector('.rv-body, .notice')).not.toBeNull())
  return container
}

const cards = (el: HTMLElement) => el.querySelectorAll('article.rv-card')

describe('ReviewView', () => {
  it('shows the instructions and the unconfirmed count on each of the four tabs', async () => {
    const el = await mount(route('suggestions'))
    expect(el.textContent).toContain('按「複製 ID」，貼到對話中再加上你的決定；Claude 會改規則或把項目記為確認過，下次建置後這頁就跟著更新。')
    const tabs = [...el.querySelectorAll<HTMLAnchorElement>('[role=tab]')]
    expect(tabs.map((t) => t.textContent)).toEqual(['規則建議3', '檢查警告1', '譯者註記1', '待辦清單1'])
    expect(tabs.map((t) => t.getAttribute('href'))).toEqual(['#/r/suggestions', '#/r/warnings', '#/r/notes', '#/r/todo'])
    expect(tabs[0]?.getAttribute('aria-selected')).toBe('true')
  })

  it('hides the confirmed by default and shows them faded, with the reason and the date, on all=1', async () => {
    let el = await mount(route('suggestions'))
    expect(cards(el)).toHaveLength(3)
    expect(el.textContent).not.toContain('沿用遊戲內詞彙，不另立規則')
    const toggle = el.querySelector<HTMLInputElement>('input[type=checkbox]')
    expect(toggle?.parentElement?.textContent).toContain('顯示已確認的 1 項')

    el = await mount(route('suggestions', { all: true }))
    expect(cards(el)).toHaveLength(4)
    const faded = el.querySelector('article.rv-card.is-confirmed')
    expect(faded?.textContent).toContain('沿用遊戲內詞彙，不另立規則')
    expect(faded?.textContent).toContain('2026-11-02')
  })

  it('writes the filters to the address', async () => {
    const el = await mount(route('suggestions'))
    const toggle = el.querySelector<HTMLInputElement>('input[type=checkbox]')!
    act(() => toggle.click())
    expect(location.hash).toBe(reviewHref('suggestions', { all: true }))
    const family = el.querySelector<HTMLSelectElement>('select[name=family]')!
    act(() => {
      family.value = 'ally'
      family.dispatchEvent(new Event('change', { bubbles: true }))
    })
    expect(location.hash).toBe(reviewHref('suggestions', { family: 'ally' }))
    const q = el.querySelector<HTMLInputElement>('input[type=search]')!
    act(() => {
      q.value = '測試'
      q.dispatchEvent(new Event('input', { bubbles: true }))
    })
    expect(location.hash).toBe(reviewHref('suggestions', { q: '測試' }))
  })

  it('shows an inconsistent group with its translations, the units and a copy button', async () => {
    const el = await mount(route('suggestions', { family: 'ally' }))
    const card = el.querySelector('article.rv-card')!
    expect(card.textContent).toContain('inconsistent/ally/name/c3a1192c')
    expect(card.textContent).toContain('시험용 베기')
    expect(card.textContent).toContain('試験用の斬撃')
    expect(card.textContent).toContain('Test Slash')
    expect(card.querySelectorAll('.rv-var')).toHaveLength(2)
    expect(card.textContent).toContain('全部 2 個單位')
    const hrefs = [...card.querySelectorAll('details a')].map((a) => a.getAttribute('href'))
    expect(hrefs).toEqual([
      fileHref('Skills_personality-99.json', '/dataList/0/levelList/0/name'),
      fileHref('Skills_personality-99.json', '/dataList/1/levelList/0/name'),
    ])
    expect(card.querySelector('.rv-copy')?.textContent).toBe('複製 ID')
    expect(card.querySelector('.rv-var.is-top')).not.toBeNull()
    expect(card.querySelectorAll('.rv-var .rv-d').length).toBeGreaterThan(0)
  })

  it('shows evidence rows of the other kinds', async () => {
    const el = await mount(route('suggestions', { sub: 'speaker' }))
    expect(cards(el)).toHaveLength(1)
    expect(el.querySelector('article.rv-card')?.textContent).toContain('新說話者「베아트리체」')
    expect(el.querySelector('article.rv-card')?.textContent).toContain('베아트리체')
  })

  it('marks the items waiting for retranslation', async () => {
    const el = await mount(route('suggestions', { sub: 'note' }))
    expect(el.querySelector('article.rv-card')?.textContent).toContain('等待重翻')
    const some = await mount(route('notes', { all: true }))
    expect(some.querySelectorAll('.rv-pending')).toHaveLength(1)
  })

  it('says how many of the units wait when only some do', async () => {
    const suggestions = fixture<{ items: unknown[] }>('review/suggestions.json').items.map((it) => it)
    const inconsistent = JSON.parse(JSON.stringify(suggestions.find((it) => (it as { kind: string }).kind === 'inconsistent')))
    inconsistent.pending = 1
    vi.stubGlobal('fetch', vi.fn(fixtureFetch({ 'review/suggestions.json': { items: [inconsistent] } })))
    clearDataCache()
    const el = await mount(route('suggestions'))
    expect(el.querySelector('.rv-pending')?.textContent).toBe('待重翻 1')
  })

  it('shows a space the minority translation lacks with a sign', async () => {
    const base = fixture<{ items: { kind: string; variants: { zh: string; count: number; keys: string[] }[] }[] }>('review/suggestions.json').items.find((i) => i.kind === 'inconsistent')!
    const item = { ...base, variants: [{ zh: '獲得提升 ', count: 2, keys: base.variants[0]!.keys }, { zh: '獲得提升', count: 1, keys: base.variants[1]!.keys }] }
    vi.stubGlobal('fetch', vi.fn(fixtureFetch({ 'review/suggestions.json': { items: [item] } })))
    clearDataCache()
    const el = await mount(route('suggestions'))
    const signs = [...el.querySelectorAll('.rv-var .rv-sp')]
    expect(signs.map((s) => s.getAttribute('aria-label'))).toEqual(['少了一個空格'])
    expect(signs[0]?.textContent).toBe('␣')
  })

  it('lists the units of a warning as links, and marks the kana', async () => {
    const el = await mount(route('warnings'))
    expect(cards(el)).toHaveLength(1)
    const card = el.querySelector('article.rv-card')!
    const hrefs = [...card.querySelectorAll('.rv-units a')].map((a) => a.getAttribute('href'))
    expect(hrefs).toEqual([
      fileHref('Skills_personality-99.json', '/dataList/0/levelList/0/coinlist/0/coindescs/1/desc'),
      fileHref('Skills_personality-99.json', '/dataList/2/levelList/0/coinlist/0/coindescs/1/desc'),
    ])
    expect([...card.querySelectorAll('mark')].map((m) => m.textContent)).toEqual(['の'])
    expect(el.querySelector('.rv-section h2')?.textContent).toContain('假名殘留')
    expect(card.textContent).toContain('warning/kana/1315241d/2cc3aa54')
    expect(card.querySelector('.rv-msg')?.textContent).toBe('the translation contains kana: "の"')
  })

  it('shows a term group with its confirmed units only on all=1', async () => {
    const unit = (n: number, accepted?: object) => ({ key: `a.json|/${n}`, id: `warning/term-missing/${n}`, kr: '자판기', zh: '機器', ...(accepted ? { accepted } : {}) })
    const group = {
      id: 'termgroup/aaaaaaaa/bbbbbbbb', code: 'term-missing', kr: '자판기', zh: '自動販賣機', message: 'm', keys: ['a.json|/1'], families: ['story'], origins: [], pending: 0,
      term: { kr: '자판기', zh: '自動販賣機' }, units: [unit(1), unit(2, { reason: '例外', date: '2026-11-03' })],
    }
    vi.stubGlobal('fetch', vi.fn(fixtureFetch({ 'review/warnings.json': { items: [group] } })))
    clearDataCache()
    let el = await mount(route('warnings'))
    expect(el.querySelectorAll('.rv-tunit')).toHaveLength(1)
    el = await mount(route('warnings', { all: true }))
    expect(el.querySelectorAll('.rv-tunit')).toHaveLength(2)
  })

  it('shows the whole note and the unit of a translator note', async () => {
    const el = await mount(route('notes'))
    expect(cards(el)).toHaveLength(1)
    const card = el.querySelector('article.rv-card')!
    expect(card.querySelector('.rv-note-text')?.textContent).toContain('保留 <> 角括號')
    expect(card.querySelector('.rv-units a')?.getAttribute('href')).toBe(fileHref('StoryData/S9999.json', '/dataList/2/content'))
    expect(card.textContent).toContain('2026-11-01')
  })

  it('fills the text filter from a shortcut', async () => {
    const el = await mount(route('notes'))
    const buttons = [...el.querySelectorAll<HTMLButtonElement>('.rv-quick button')]
    expect(buttons.map((b) => b.textContent)).toEqual(['新詞', '諧音', '俗語', '沿用', '標籤'])
    act(() => buttons[4]!.click())
    expect(location.hash).toBe(reviewHref('notes', { q: '標籤' }))
  })

  it('shows the sections of the to-do list, each item linked to its unit', async () => {
    const el = await mount(route('todo'))
    expect(el.textContent).toContain('新說話者、新欄位、新檔案族群與過期的調整列在「規則建議」分頁，可以在那裡確認。')
    expect(el.querySelector('.rv-section h2')?.textContent).toContain('needs_review')
    const a = el.querySelector<HTMLAnchorElement>('.rv-todo a')
    expect(a?.getAttribute('href')).toBe(fileHref('StoryData/S9999.json', '/dataList/2/content'))
  })

  it('says so when the to-do list is empty', async () => {
    vi.stubGlobal('fetch', vi.fn(fixtureFetch({ 'review/todo.json': { sections: [] } })))
    clearDataCache()
    const el = await mount(route('todo'))
    expect(el.textContent).toContain('這次建置沒有待辦項目')
  })

  it('says so when the item of the id is not there', async () => {
    const el = await mount(route('suggestions', { id: 'inconsistent/nothing/1' }))
    expect(el.querySelector('.rv-missing')?.textContent).toContain('inconsistent/nothing/1')
  })

  describe('long lists', () => {
    function manyNotes(n: number): NoteItem[] {
      const base = fixture<{ items: NoteItem[] }>('review/notes.json').items[0]!
      return Array.from({ length: n }, (_, i) => ({ ...base, id: `note/F.json|/dataList/${i}/content/${String(i).padStart(8, '0')}`, key: `F.json|/dataList/${i}/content`, pending: 0 }))
    }

    it('long lists render 100 at a time', async () => {
      vi.stubGlobal('fetch', vi.fn(fixtureFetch({ 'review/notes.json': { items: manyNotes(250) } })))
      clearDataCache()
      const el = await mount(route('notes'))
      expect(cards(el)).toHaveLength(100)
      const more = el.querySelector<HTMLButtonElement>('.rv-more')!
      expect(more.textContent).toBe('再顯示 100 項（共 250）')
      act(() => more.click())
      expect(cards(el)).toHaveLength(200)
      act(() => el.querySelector<HTMLButtonElement>('.rv-more')!.click())
      expect(cards(el)).toHaveLength(250)
      expect(el.querySelector('.rv-more')).toBeNull()
    })

    it('renders and highlights the item of the id even past the first 100', async () => {
      const items = manyNotes(250)
      vi.stubGlobal('fetch', vi.fn(fixtureFetch({ 'review/notes.json': { items } })))
      clearDataCache()
      const id = items[179]!.id
      const el = await mount(route('notes', { id }))
      expect(cards(el).length).toBeGreaterThanOrEqual(180)
      const target = el.querySelector('article.rv-card.is-target')
      expect(target?.getAttribute('data-id')).toBe(id)
    })
  })
})

describe('the JP/EN view of the inconsistent translations', () => {
  type V = { zh: string; count: number; keys: string[]; jpen?: unknown[] }
  const base = () => JSON.parse(JSON.stringify(fixture<{ items: { kind: string; variants: V[] }[] }>('review/suggestions.json').items.find((i) => i.kind === 'inconsistent')))
  // 一 is used for (A, a) and (B, —), 壹 for (A, a): the pair (A, a) has two translations.
  function partial() {
    const it = base()
    const [k1, k2] = [it.variants[0].keys[0], it.variants[1].keys[0]]
    it.id = 'inconsistent/ally/name/partial01'
    it.jpen_class = 'partial'
    it.variants = [
      { zh: '一', count: 2, keys: [k1, 'X.json|/dataList/9/name'], jpen: [{ jp: 'A', en: 'a', count: 1, keys: [k1] }, { jp: 'B', count: 1, keys: ['X.json|/dataList/9/name'] }] },
      { zh: '壹', count: 1, keys: [k2], jpen: [{ jp: 'A', en: 'a', count: 1, keys: [k2] }] },
    ]
    return it
  }

  it("JP/EN view shows each variant's JP and EN", async () => {
    let el = await mount(route('suggestions', { mode: 'jpen' }))
    const card = [...cards(el)].find((c) => c.getAttribute('data-id')?.startsWith('inconsistent/'))!
    expect(card.querySelector('.rv-badge')?.textContent).toBe('JP/EN 可解釋')
    const rows = [...card.querySelectorAll('.rv-jpen')].map((r) => r.textContent)
    expect(rows.some((t) => t?.includes('試験用の防御') && t.includes('Test Guard'))).toBe(true)
    expect(card.querySelector('.rv-jpen [lang="ja"]')).not.toBeNull()
    el = await mount(route('suggestions'))
    expect(el.querySelector('.rv-badge')).toBeNull()
    expect(el.querySelector('.rv-jpen')).toBeNull()
  })

  it('jpen row shows a dash', async () => {
    vi.stubGlobal('fetch', vi.fn(fixtureFetch({ 'review/suggestions.json': { items: [partial()] } })))
    clearDataCache()
    const el = await mount(route('suggestions', { mode: 'jpen' }))
    const row = [...el.querySelectorAll('.rv-jpen')].find((r) => r.textContent?.includes('B'))!
    expect(row.querySelector('.rv-jpen-en')?.textContent).toBe('—')
  })

  it('marks the JP/EN pair shared by two translations', async () => {
    vi.stubGlobal('fetch', vi.fn(fixtureFetch({ 'review/suggestions.json': { items: [partial()] } })))
    clearDataCache()
    const el = await mount(route('suggestions', { mode: 'jpen' }))
    expect(el.querySelector('.rv-badge')?.textContent).toBe('部分可解釋')
    const shared = [...el.querySelectorAll('.rv-jpen.is-shared')]
    expect(shared).toHaveLength(2)
    expect(shared.every((r) => r.textContent?.includes('同樣的 JP/EN 有其他譯法'))).toBe(true)
  })

  it('badges and filter', async () => {
    vi.stubGlobal('fetch', vi.fn(fixtureFetch({ 'review/suggestions.json': { items: [base(), partial(), ...fixture<{ items: { kind: string }[] }>('review/suggestions.json').items.filter((i) => i.kind !== 'inconsistent')] } })))
    clearDataCache()
    let el = await mount(route('suggestions', { mode: 'jpen' }))
    const options = [...el.querySelectorAll<HTMLOptionElement>('select[name="jpen"] option')].map((o) => o.textContent)
    expect(options).toEqual(['全部', '可解釋（1）', '部分可解釋（1）', '相同（0）'])
    el = await mount(route('suggestions', { mode: 'jpen', jpen: 'partial' }))
    const shownIds = [...cards(el)].map((c) => c.getAttribute('data-id'))
    expect(shownIds.filter((i) => i?.startsWith('inconsistent/'))).toEqual(['inconsistent/ally/name/partial01'])
    expect(shownIds.some((i) => i?.startsWith('speaker/'))).toBe(true)
    el = await mount(route('suggestions'))
    expect(el.querySelector('select[name="jpen"]')).toBeNull()
  })

  it('deep link keeps working in the JP/EN view', async () => {
    const id = base().id as string
    const el = await mount(route('suggestions', { mode: 'jpen', id }))
    await vi.waitFor(() => expect(el.querySelector(`article.is-target[data-id="${id}"]`)).not.toBeNull())
  })
})

describe('the review page follows the language and raw-tag switches of the header', () => {
  const shown = (langs: Partial<Settings['langs']>, rest: Partial<Settings> = {}): Settings => ({
    ...defaultSettings(),
    ...rest,
    langs: { ...defaultSettings().langs, ...langs },
  })
  const labels = (el: HTMLElement) => [...el.querySelectorAll('article.rv-card dt')].map((d) => d.textContent)

  it('hides KR, JP and EN when they are switched off, and keeps TW', async () => {
    let el = await mount(route('suggestions'))
    expect(labels(el)).toEqual(expect.arrayContaining(['KR', 'JP', 'EN']))
    el = await mount(route('suggestions'), shown({ kr: false, jp: false, en: false }))
    expect(labels(el).filter((l) => l === 'KR' || l === 'JP' || l === 'EN')).toEqual([])
    el = await mount(route('warnings'), shown({ kr: false }))
    expect(labels(el)).not.toContain('KR')
    expect(labels(el)).toContain('TW')
  })

  it('hides the JP or the EN of the JP/EN view when switched off, and keeps the class', async () => {
    let el = await mount(route('suggestions', { mode: 'jpen' }), shown({ jp: false }))
    expect(el.querySelector('.rv-jpen-jp')).toBeNull()
    expect(el.querySelector('.rv-jpen-en')).not.toBeNull()
    el = await mount(route('suggestions', { mode: 'jpen' }), shown({ jp: false, en: false }))
    expect(el.querySelector('.rv-jpen')).toBeNull()
    expect(el.querySelector('.rv-badge')).not.toBeNull()
  })

  it('shows the tags of the translations of an inconsistent group as they are when raw tags are on', async () => {
    const it = JSON.parse(JSON.stringify(fixture<{ items: { kind: string; variants: { zh: string }[] }[] }>('review/suggestions.json').items.find((i) => i.kind === 'inconsistent')))
    it.variants[1].zh = '<color=#ff6000>試驗</color>斬擊'
    vi.stubGlobal('fetch', vi.fn(fixtureFetch({ 'review/suggestions.json': { items: [it] } })))
    clearDataCache()
    let el = await mount(route('suggestions'))
    const zh = (e: HTMLElement) => [...e.querySelectorAll('.rv-var .rv-zh')].map((z) => z.textContent)
    expect(zh(el)).not.toContain('<color=#ff6000>試驗</color>斬擊')
    el = await mount(route('suggestions'), shown({}, { rawTags: true }))
    expect(zh(el)).toContain('<color=#ff6000>試驗</color>斬擊')
    expect(el.querySelector('.rv-var .rv-zh .tmp-raw')).not.toBeNull()
  })

  it('redraws every translation of an inconsistent group when raw tags are switched on and off', async () => {
    const it = JSON.parse(JSON.stringify(fixture<{ items: { kind: string; variants: { zh: string }[] }[] }>('review/suggestions.json').items.find((i) => i.kind === 'inconsistent')))
    it.variants[0].zh = '[OnSucceedAttack] 使目標增加<style="highlight">2</style>回 [Laceration]'
    it.variants[1].zh = '[OnSucceedAttack] 使目標增加2級 [Laceration] 強度'
    vi.stubGlobal('fetch', vi.fn(fixtureFetch({ 'review/suggestions.json': { items: [it] } })))
    clearDataCache()
    const el = await mount(route('suggestions'))
    const zh = () => [...el.querySelectorAll('.rv-var .rv-zh')].map((z) => z.textContent)
    const drawn = zh()
    render(<ReviewView route={route('suggestions')} index={index} meta={meta} settings={shown({}, { rawTags: true })} />, el)
    // A space only one translation has shows as ␣.
    expect(zh()).toEqual(['[OnSucceedAttack] 使目標增加<style="highlight">2</style>回 [Laceration]', '[OnSucceedAttack] 使目標增加2級 [Laceration]␣強度'])
    render(<ReviewView route={route('suggestions')} index={index} meta={meta} settings={shown({})} />, el)
    expect(zh()).toEqual(drawn)
  })

  it('shows the tags as they are when raw tags are on, still marking the problem', async () => {
    let el = await mount(route('warnings'))
    expect(el.querySelector('article.rv-card dd .tmp-raw')).toBeNull()
    el = await mount(route('warnings'), shown({}, { rawTags: true }))
    const raw = [...el.querySelectorAll('article.rv-card dd .tmp-raw')].map((r) => r.textContent)
    expect(raw).toContain('<sprite name="Burn"> 達 5 次以上の時，攻擊威力 +1')
    expect([...el.querySelectorAll('article.rv-card dd mark')].map((m) => m.textContent)).toContain('の')
  })
})
