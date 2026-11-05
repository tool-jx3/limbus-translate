import { render } from 'preact'
import { act } from 'preact/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { clearDataCache } from '../data'
import { defaultSettings, type Settings } from '../settings'
import { fixtureFetch, runEffectsAtOnce } from '../testdata'
import type { UpdateEntry, UpdateRecord } from '../types'
import { UpdateCard } from './UpdateCard'
import { UPDATE_PAGE, UpdateView } from './UpdateView'
import type { Route } from '../route'

type UpdateRoute = Extract<Route, { kind: 'update' }>

const TAG = 'zh-TW-20261105.1'
const entries: UpdateEntry[] = [
  { file: 'A.json', field: 'name', ptr: '/0/name', kind: 'new', new_kr: '새 이름', new_zh: '新名稱', new_origin: 'claude' },
  {
    file: 'A.json',
    field: 'desc',
    ptr: '/1/desc',
    kind: 'kr-changed',
    old_kr: '옛 설명입니다',
    new_kr: '새 설명입니다',
    old_zh: '舊的說明文字',
    new_zh: '新的說明文字',
    old_origin: 'paratranz',
    new_origin: 'claude',
  },
  { file: 'B.json', field: 'name', ptr: '/2/name', kind: 'deleted', old_kr: '지운 것', old_zh: '刪掉的', old_origin: 'human' },
  {
    file: 'B.json',
    field: 'name',
    ptr: '/3/name',
    kind: 'zh-changed',
    old_kr: '같은 말',
    new_kr: '같은 말',
    old_zh: '舊譯',
    new_zh: '新譯',
    old_origin: 'claude',
    new_origin: 'human',
    cause: 'adjustment',
  },
  { file: 'B.json', field: 'name', ptr: '/4/name', kind: 'zh-changed', new_kr: '늦은', old_zh: 'x', new_zh: 'y', new_origin: 'claude', cause: 'late' },
  { file: 'B.json', field: 'name', ptr: '/5/name', kind: 'zh-changed', new_kr: '다시', old_zh: 'x', new_zh: 'y', new_origin: 'claude', cause: 'retranslated' },
  { file: 'B.json', field: 'name', ptr: '/6/name', kind: 'new', new_kr: 'copy', new_zh: 'copy', new_origin: 'copy', copy: true },
]

function record(over: Partial<UpdateRecord> = {}): UpdateRecord {
  return {
    tag: TAG,
    prev_tag: 'zh-TW-20261029.7',
    released_at: '2026-11-05T08:30:00Z',
    prev_released_at: '2026-10-29T08:30:00Z',
    raw_tag: 'raw-20261105',
    prev_raw_tag: 'raw-20261029',
    rules_commit: 'a',
    prev_rules_commit: 'b',
    counts: { new: 1, kr_changed: 1, deleted: 1, zh_changed: 3 },
    copy_counts: { new: 2, kr_changed: 1, deleted: 1, zh_changed: 3 },
    other_files: ['Other/x.json'],
    entries,
    ...over,
  }
}

const route = (over: Partial<UpdateRoute> = {}): UpdateRoute => ({ kind: 'update', tag: TAG, kinds: [], origin: '', file: '', copy: false, ...over })

const containers: HTMLElement[] = []
let undoEffects = () => {}

beforeEach(() => {
  undoEffects = runEffectsAtOnce()
  clearDataCache()
  location.hash = ''
})

afterEach(() => {
  undoEffects()
  for (const c of containers.splice(0)) {
    render(null, c)
    c.remove()
  }
  vi.unstubAllGlobals()
})

function stub(docs: Record<string, unknown> = {}, rec: UpdateRecord = record()) {
  vi.stubGlobal('fetch', vi.fn(fixtureFetch({ [`updates/${TAG}.json`]: rec, ...docs })))
}

async function mount(
  r: UpdateRoute = route(),
  settings: Settings = defaultSettings(),
  onSettings: (s: Settings) => void = () => {},
): Promise<HTMLElement> {
  const container = document.createElement('div')
  document.body.append(container)
  containers.push(container)
  render(<UpdateView route={r} settings={settings} onSettings={onSettings} />, container)
  await vi.waitFor(() => expect(container.querySelector('.uc-card, .ut-table, .notice, .uv-empty')).not.toBeNull())
  return container
}

const text = (el: Element | null | undefined) => el?.textContent ?? ''

describe('UpdateView header', () => {
  it('shows the tag, date, kind, previous tag, number of files and the other files', async () => {
    stub()
    const el = await mount()
    const head = el.querySelector('.uv-head')!
    expect(text(head)).toContain(TAG)
    expect(text(head)).toContain('2026-11-05')
    expect(text(head)).toContain('遊戲更新 raw-20261105')
    expect(text(head)).toContain('比對 zh-TW-20261029.7')
    expect(text(head)).toContain('2 個檔案')
    expect(text(head)).toContain('Other/x.json')
  })

  it('shows a rebuild as 重建', async () => {
    stub({}, record({ prev_raw_tag: 'raw-20261105', other_files: [] }))
    const head = (await mount()).querySelector('.uv-head')!
    expect(text(head)).toContain('重建')
    expect(text(head)).not.toContain('遊戲更新')
    expect(text(head)).not.toContain('Other/')
  })

  it('shows the notice when the load fails', async () => {
    vi.stubGlobal('fetch', vi.fn(fixtureFetch()))
    const el = await mount()
    expect(el.querySelector('.notice')).not.toBeNull()
  })
})

describe('UpdateView filters', () => {
  it('has four kind chips with counts that leave out copy entries by default', async () => {
    stub()
    const chips = [...(await mount()).querySelectorAll('.uv-kind')]
    expect(chips.map((c) => c.textContent)).toEqual(['新增 1', 'KR 修改 1', '刪除 1', '只有譯文改變 3'])
  })

  it('counts the copy entries when they are shown', async () => {
    stub()
    const chips = [...(await mount(route({ copy: true }))).querySelectorAll('.uv-kind')]
    expect(chips.map((c) => c.textContent)).toEqual(['新增 2', 'KR 修改 1', '刪除 1', '只有譯文改變 3'])
  })

  it('writes the kinds to the URL when a chip is clicked, and takes it off again', async () => {
    stub()
    const el = await mount()
    await act(() => el.querySelectorAll<HTMLButtonElement>('.uv-kind')[1]!.click())
    expect(location.hash).toBe('#/u/zh-TW-20261105.1?k=kr-changed')
    const el2 = await mount(route({ kinds: ['kr-changed'] }))
    const chip = el2.querySelectorAll<HTMLButtonElement>('.uv-kind')[1]!
    expect(chip.getAttribute('aria-pressed')).toBe('true')
    await act(() => chip.click())
    expect(location.hash).toBe('#/u/zh-TW-20261105.1')
  })

  it('writes the origin and the file to the URL', async () => {
    stub()
    const el = await mount()
    const origin = el.querySelector<HTMLSelectElement>('select[name="origin"]')!
    expect([...origin.options].map((o) => o.value)).toEqual(['', 'claude', 'copy', 'human'])
    origin.value = 'claude'
    await act(() => {
      origin.dispatchEvent(new Event('change', { bubbles: true }))
    })
    expect(location.hash).toBe('#/u/zh-TW-20261105.1?o=claude')
    const file = el.querySelector<HTMLSelectElement>('select[name="file"]')!
    expect([...file.options].map((o) => o.value)).toEqual(['', 'A.json', 'B.json'])
    file.value = 'B.json'
    await act(() => {
      file.dispatchEvent(new Event('change', { bubbles: true }))
    })
    expect(location.hash).toBe('#/u/zh-TW-20261105.1?f=B.json')
  })

  it('hides the copy entries by default, with their number, and writes copy=1 when unchecked', async () => {
    stub()
    const el = await mount()
    const box = el.querySelector<HTMLInputElement>('input[name="hide-copy"]')!
    expect(box.checked).toBe(true)
    expect(text(box.closest('label'))).toContain('隱藏不需翻譯的單位（1 個）')
    expect(el.querySelectorAll('.uc-card').length).toBe(6)
    await act(() => box.click())
    expect(location.hash).toBe('#/u/zh-TW-20261105.1?copy=1')
  })

  it('shows the copy entries when the route says copy', async () => {
    stub()
    const el = await mount(route({ copy: true }))
    expect(el.querySelector<HTMLInputElement>('input[name="hide-copy"]')!.checked).toBe(false)
    expect(el.querySelectorAll('.uc-card').length).toBe(7)
  })

  it('shows only what the route filters', async () => {
    stub()
    const el = await mount(route({ kinds: ['zh-changed'], file: 'B.json' }))
    expect(el.querySelectorAll('.uc-card').length).toBe(3)
  })
})

describe('UpdateView groups', () => {
  it('groups the cards by file, headed by the path', async () => {
    stub()
    const el = await mount()
    const groups = [...el.querySelectorAll('.uv-group')]
    expect(groups.map((g) => text(g.querySelector('.uv-file')))).toEqual(['A.json', 'B.json'])
    expect(groups.map((g) => g.querySelectorAll('.uc-card').length)).toEqual([2, 4])
  })
})

describe('UpdateCard', () => {
  async function cards(): Promise<Element[]> {
    stub()
    const el = await mount(route({ copy: true }))
    return [...el.querySelectorAll('.uc-card')]
  }

  it('shows old and new KR and TW for a changed KR, with the differences marked', async () => {
    const c = (await cards())[1]!
    expect(text(c.querySelector('.uc-kind'))).toBe('KR 修改')
    expect(text(c.querySelector('.uc-where'))).toContain('desc')
    expect(text(c.querySelector('.uc-where'))).toContain('/1/desc')
    expect(text(c.querySelector('.uc-origins'))).toContain('paratranz')
    expect(text(c.querySelector('.uc-origins'))).toContain('claude')
    const rows = [...c.querySelectorAll('.uc-row')]
    expect(rows.map((r) => text(r.querySelector('dt')))).toEqual(['KR', 'TW'])
    const krOld = rows[0]!.querySelector('.uc-old')!
    const krNew = rows[0]!.querySelector('.uc-new')!
    expect(text(krOld)).toContain('舊')
    expect(text(krOld)).toContain('옛 설명입니다')
    expect(text(krNew)).toContain('新')
    expect(text(krNew)).toContain('새 설명입니다')
    // 옛 / 새 differ; the rest is shared
    expect([...krOld.querySelectorAll('.rv-d')].map((d) => d.textContent)).toEqual(['옛'])
    expect([...krNew.querySelectorAll('.rv-d')].map((d) => d.textContent)).toEqual(['새'])
    const zhOld = rows[1]!.querySelector('.uc-old')!
    const zhNew = rows[1]!.querySelector('.uc-new')!
    expect([...zhOld.querySelectorAll('.rv-d')].map((d) => d.textContent)).toEqual(['舊'])
    expect([...zhNew.querySelectorAll('.rv-d')].map((d) => d.textContent)).toEqual(['新'])
  })

  it('shows only the new texts for a new unit', async () => {
    const c = (await cards())[0]!
    expect(text(c.querySelector('.uc-kind'))).toBe('新增')
    expect(c.querySelector('.uc-old')).toBeNull()
    expect(c.querySelectorAll('.uc-new').length).toBe(2)
    expect(text(c)).toContain('新名稱')
  })

  it('shows only the old texts for a deleted unit, with no link to the file', async () => {
    const c = (await cards())[2]!
    expect(text(c.querySelector('.uc-kind'))).toBe('刪除')
    expect(c.querySelector('.uc-new')).toBeNull()
    expect(c.querySelectorAll('.uc-old').length).toBe(2)
    expect(text(c)).toContain('刪掉的')
    expect(text(c)).not.toContain('在檔案中開啟')
    expect(c.querySelector('a')).toBeNull()
  })

  it('shows the KR once for a zh-changed unit, with the cause', async () => {
    const all = await cards()
    const c = all[3]!
    expect(text(c.querySelector('.uc-kind'))).toBe('只有譯文改變')
    expect(text(c.querySelector('.uc-cause'))).toBe('人工調整')
    const rows = [...c.querySelectorAll('.uc-row')]
    expect(rows[0]!.querySelector('.uc-old')).toBeNull()
    expect(rows[0]!.querySelector('.uc-new')).toBeNull()
    expect(text(rows[0])).toContain('같은 말')
    expect(rows[1]!.querySelector('.uc-old')).not.toBeNull()
    expect(rows[1]!.querySelector('.uc-new')).not.toBeNull()
    expect(text(all[4]!.querySelector('.uc-cause'))).toBe('補翻')
    expect(text(all[5]!.querySelector('.uc-cause'))).toBe('重翻')
    expect(all[0]!.querySelector('.uc-cause')).toBeNull()
  })

  it('links the other cards to the unit in the file view', async () => {
    const c = (await cards())[1]!
    const a = [...c.querySelectorAll('a')].find((x) => text(x).includes('在檔案中開啟'))!
    expect(a.getAttribute('href')).toBe('#/f/A.json?u=%2F1%2Fdesc')
  })

  it('leaves the KR rows out when KR is switched off', async () => {
    stub()
    const s = defaultSettings()
    s.langs.kr = false
    const el = await mount(route({ copy: true }), s)
    const c = el.querySelectorAll('.uc-card')[1]!
    expect([...c.querySelectorAll('.uc-row dt')].map((d) => text(d))).toEqual(['TW'])
  })

  it('draws the diff with raw tags when raw tags are on', async () => {
    const e: UpdateEntry = { file: 'A.json', field: 'x', ptr: '/0/x', kind: 'kr-changed', old_kr: 'a', new_kr: 'a', old_zh: '<b>好</b>', new_zh: '好', new_origin: 'claude' }
    stub({}, record({ entries: [e] }))
    const s = defaultSettings()
    s.rawTags = true
    const el = await mount(route(), s)
    const zhOld = el.querySelectorAll('.uc-row')[1]!.querySelector('.uc-old')!
    expect([...zhOld.querySelectorAll('.rv-d')].map((d) => d.textContent)).toEqual(['<b>', '</b>'])
    expect(zhOld.querySelector('.tmp-raw')).not.toBeNull()
    stub({}, record({ entries: [e] }))
    clearDataCache()
    const el2 = await mount(route(), defaultSettings())
    const zhOld2 = el2.querySelectorAll('.uc-row')[1]!.querySelector('.uc-old')!
    expect(zhOld2.querySelectorAll('.rv-d').length).toBe(0)
  })
})

describe('JP/EN of the current file', () => {
  const file = {
    path: 'A.json',
    family: 'x',
    units: [
      { p: '/0/name', f: 'name', i: -1, kr: '새 이름', jp: '新しい名前', en: 'New name', zh: '新名稱', o: 'claude' },
      { p: '/1/desc', f: 'desc', i: -1, kr: 'k', zh: 'z', o: 'claude' },
    ],
  }

  it('loads the file only when opened, and shows the JP and EN of the unit', async () => {
    stub({ 'files/A.json': file })
    const fetchMock = vi.mocked(fetch)
    const el = await mount(route({ file: 'A.json' }))
    const calls = () => fetchMock.mock.calls.filter((c) => String(c[0]).includes('files/A.json')).length
    expect(calls()).toBe(0)
    const first = el.querySelector('.uc-card')!
    const d = first.querySelector<HTMLDetailsElement>('details.uc-jpen')!
    expect(text(d.querySelector('summary'))).toBe('JP／EN（目前的）')
    d.open = true
    await act(() => {
      d.dispatchEvent(new Event('toggle'))
    })
    await vi.waitFor(() => expect(text(first)).toContain('新しい名前'))
    expect(text(first)).toContain('New name')
    expect(calls()).toBe(1)
  })

  it('shows a dash for the language the unit has no string for', async () => {
    stub({ 'files/A.json': file })
    const el = await mount(route({ file: 'A.json' }))
    const second = el.querySelectorAll('.uc-card')[1]!
    const d = second.querySelector<HTMLDetailsElement>('details.uc-jpen')!
    d.open = true
    await act(() => {
      d.dispatchEvent(new Event('toggle'))
    })
    await vi.waitFor(() => expect(text(d)).not.toContain('載入中'))
    expect(text(d)).not.toContain('找不到')
    expect(text(d)).toContain('—')
  })

  it('says so where the file has no such unit', async () => {
    stub({ 'files/A.json': { ...file, units: [] } })
    const el = await mount(route({ file: 'A.json' }))
    const first = el.querySelector('.uc-card')!
    const d = first.querySelector<HTMLDetailsElement>('details.uc-jpen')!
    d.open = true
    await act(() => {
      d.dispatchEvent(new Event('toggle'))
    })
    await vi.waitFor(() => expect(text(first)).toContain('目前的檔案裡找不到這個單位'))
  })

  it('has no such disclosure when JP and EN are both switched off, nor for a deleted unit', async () => {
    stub()
    const s = defaultSettings()
    s.langs.jp = false
    s.langs.en = false
    const el = await mount(route(), s)
    expect(el.querySelector('details.uc-jpen')).toBeNull()
    clearDataCache()
    stub()
    const el2 = await mount(route({ kinds: ['deleted'] }))
    expect(el2.querySelector('details.uc-jpen')).toBeNull()
  })
})

describe('paging', () => {
  const many: UpdateEntry[] = Array.from({ length: 650 }, (_, k) => ({
    file: 'A.json',
    field: 'name',
    ptr: `/${k}/name`,
    kind: k % 2 === 0 ? 'new' : 'zh-changed',
    new_kr: 'k',
    old_zh: 'a',
    new_zh: 'b',
    new_origin: 'claude',
  }))

  it('shows 300 entries and then 300 more', async () => {
    expect(UPDATE_PAGE).toBe(300)
    stub({}, record({ entries: many }))
    const el = await mount()
    const more = () => [...el.querySelectorAll<HTMLButtonElement>('button')].find((b) => text(b).startsWith('再顯示'))
    expect(el.querySelectorAll('.uc-card').length).toBe(300)
    expect(text(more())).toBe('再顯示 300 筆')
    await act(() => more()!.click())
    expect(el.querySelectorAll('.uc-card').length).toBe(600)
    expect(text(more())).toBe('再顯示 50 筆')
    await act(() => more()!.click())
    expect(el.querySelectorAll('.uc-card').length).toBe(650)
    expect(more()).toBeUndefined()
  })

  it('goes back to 300 when the filter changes', async () => {
    stub({}, record({ entries: many }))
    const container = document.createElement('div')
    document.body.append(container)
    containers.push(container)
    const draw = (r: UpdateRoute) => render(<UpdateView route={r} settings={defaultSettings()} onSettings={() => {}} />, container)
    await act(() => draw(route()))
    await vi.waitFor(() => expect(container.querySelector('.uc-card')).not.toBeNull())
    await act(() => [...container.querySelectorAll<HTMLButtonElement>('button')].find((b) => text(b).startsWith('再顯示'))!.click())
    expect(container.querySelectorAll('.uc-card').length).toBe(600)
    await act(() => draw(route({ kinds: ['new', 'zh-changed'] })))
    expect(container.querySelectorAll('.uc-card').length).toBe(300)
  })
})

describe('UpdateCard alone', () => {
  it('renders without the provider of the header switches', () => {
    const container = document.createElement('div')
    document.body.append(container)
    containers.push(container)
    render(<UpdateCard entry={entries[0]!} />, container)
    expect(container.querySelector('.uc-card')).not.toBeNull()
  })
})

describe('the table view', () => {
  const table = (): Settings => ({ ...defaultSettings(), updateView: 'table' })
  const rowsOf = (el: Element) => [...el.querySelectorAll<HTMLElement>('tr.ut-row')]

  it('has a switch 卡片｜表格 that asks for the table, and the cards stay by default', async () => {
    stub()
    const onSettings = vi.fn()
    const el = await mount(route(), defaultSettings(), onSettings)
    const buttons = [...el.querySelectorAll<HTMLButtonElement>('.uv-views button')]
    expect(buttons.map((b) => text(b))).toEqual(['卡片', '表格'])
    expect(buttons.map((b) => b.getAttribute('aria-pressed'))).toEqual(['true', 'false'])
    expect(el.querySelector('table')).toBeNull()
    await act(() => buttons[1]!.click())
    expect(onSettings).toHaveBeenCalledWith({ ...defaultSettings(), updateView: 'table' })
  })

  it('opens as a table when the setting says table, and the switch can go back to cards', async () => {
    stub()
    const onSettings = vi.fn()
    const el = await mount(route(), table(), onSettings)
    expect(el.querySelector('.uc-card')).toBeNull()
    const buttons = [...el.querySelectorAll<HTMLButtonElement>('.uv-views button')]
    expect(buttons.map((b) => b.getAttribute('aria-pressed'))).toEqual(['false', 'true'])
    await act(() => buttons[0]!.click())
    expect(onSettings).toHaveBeenCalledWith({ ...defaultSettings(), updateView: 'cards' })
  })

  it('has the columns, and a group row for each file', async () => {
    stub()
    const el = await mount(route(), table())
    expect([...el.querySelectorAll('thead th')].map((h) => text(h))).toEqual(['欄位', '種類', 'KR', 'TW', '來源'])
    const groups = [...el.querySelectorAll('tr.ut-file')]
    expect(groups.map((g) => text(g))).toEqual(['A.json', 'B.json'])
    expect(groups[0]!.querySelector('th')!.getAttribute('colspan')).toBe('5')
    expect(rowsOf(el).length).toBe(6)
  })

  it('shows the field and pointer, the kind, the cause and the origins of a row', async () => {
    stub()
    const el = await mount(route(), table())
    const rows = rowsOf(el)
    expect(text(rows[1]!.querySelector('code'))).toContain('desc')
    expect(text(rows[1]!.querySelector('code'))).toContain('/1/desc')
    expect(text(rows[1]!.querySelector('.uc-kind'))).toBe('KR 修改')
    expect(text(rows[1]!.querySelector('.uc-origins'))).toContain('paratranz')
    expect(text(rows[1]!.querySelector('.uc-origins'))).toContain('claude')
    expect(text(rows[3]!.querySelector('.uc-cause'))).toBe('人工調整')
  })

  it('puts the old above the new in the KR and TW cells, with the differences marked', async () => {
    stub()
    const row = rowsOf(await mount(route(), table()))[1]!
    const cells = [...row.querySelectorAll('td')]
    const kr = cells[2]!
    const zh = cells[3]!
    expect(kr.querySelector('.uc-old')!.compareDocumentPosition(kr.querySelector('.uc-new')!) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    expect([...kr.querySelectorAll('.uc-old .rv-d')].map((d) => d.textContent)).toEqual(['옛'])
    expect([...kr.querySelectorAll('.uc-new .rv-d')].map((d) => d.textContent)).toEqual(['새'])
    expect([...zh.querySelectorAll('.uc-old .rv-d')].map((d) => d.textContent)).toEqual(['舊'])
    expect([...zh.querySelectorAll('.uc-new .rv-d')].map((d) => d.textContent)).toEqual(['新'])
  })

  it('leaves the KR cell empty when KR is switched off', async () => {
    stub()
    const s = table()
    s.langs.kr = false
    const row = rowsOf(await mount(route(), s))[1]!
    const cells = [...row.querySelectorAll('td')]
    expect(text(cells[2])).toBe('')
    expect(text(cells[3])).toContain('新的說明文字')
  })

  it('is collapsed at first, toggles on a click on the row, and keeps it out of the URL', async () => {
    stub()
    const el = await mount(route(), table())
    const row = rowsOf(el)[0]!
    expect(row.classList.contains('is-open')).toBe(false)
    expect(row.getAttribute('aria-expanded')).toBe('false')
    await act(() => row.click())
    expect(rowsOf(el)[0]!.classList.contains('is-open')).toBe(true)
    expect(rowsOf(el)[0]!.getAttribute('aria-expanded')).toBe('true')
    expect(rowsOf(el)[1]!.classList.contains('is-open')).toBe(false)
    expect(location.hash).toBe('')
    await act(() => rowsOf(el)[0]!.click())
    expect(rowsOf(el)[0]!.classList.contains('is-open')).toBe(false)
  })

  it('can be toggled from the keyboard through a button in the row', async () => {
    stub()
    const el = await mount(route(), table())
    const button = rowsOf(el)[0]!.querySelector<HTMLButtonElement>('button.ut-toggle')!
    expect(button).not.toBeNull()
    await act(() => button.click())
    expect(rowsOf(el)[0]!.classList.contains('is-open')).toBe(true)
  })

  it('is wrapped in a container that scrolls sideways', async () => {
    stub()
    const el = await mount(route(), table())
    const wrap = el.querySelector('table')!.parentElement!
    expect(wrap.classList.contains('ut-scroll')).toBe(true)
    expect(wrap.style.overflowX).toBe('auto')
  })

  it('pages like the cards do', async () => {
    const many: UpdateEntry[] = Array.from({ length: 650 }, (_, k) => ({
      file: 'A.json',
      field: 'name',
      ptr: `/${k}/name`,
      kind: 'new',
      new_kr: 'k',
      new_zh: 'b',
      new_origin: 'claude',
    }))
    stub({}, record({ entries: many }))
    const el = await mount(route(), table())
    const more = () => [...el.querySelectorAll<HTMLButtonElement>('button')].find((b) => text(b).startsWith('再顯示'))
    expect(rowsOf(el).length).toBe(300)
    expect(text(more())).toBe('再顯示 300 筆')
    await act(() => more()!.click())
    expect(rowsOf(el).length).toBe(600)
    expect(text(more())).toBe('再顯示 50 筆')
  })
})
