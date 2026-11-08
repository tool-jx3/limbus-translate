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

const route = (over: Partial<UpdateRoute> = {}): UpdateRoute => ({ kind: 'update', tag: TAG, kinds: [], origin: '', file: '', copy: false, cause: '', group: '', ...over })

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
  await vi.waitFor(() => expect(container.querySelector('.uc-card, .ut-table, .notice, .uv-empty, .cg-head')).not.toBeNull())
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
    vi.stubGlobal('fetch', vi.fn(fixtureFetch({ [`updates/${TAG}.json`]: { tag: TAG } })))
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
      { p: '/1/desc', f: 'desc', i: -1, kr: '새 설명입니다', zh: 'z', o: 'claude' },
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

  it('does not show the JP and EN of another unit that now lies at the pointer, and still offers the link', async () => {
    const moved = { ...file, units: [{ ...file.units[0]!, kr: '다른 문장' }, file.units[1]!] }
    stub({ 'files/A.json': moved })
    const el = await mount(route({ file: 'A.json' }))
    const first = el.querySelector('.uc-card')!
    const d = first.querySelector<HTMLDetailsElement>('details.uc-jpen')!
    d.open = true
    await act(() => {
      d.dispatchEvent(new Event('toggle'))
    })
    await vi.waitFor(() => expect(text(d)).toContain('目前檔案的這個位置已是另一句'))
    expect(text(d)).not.toContain('新しい名前')
    expect(text(d)).not.toContain('New name')
    expect(first.querySelector('a.uc-open')).not.toBeNull()
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

describe('a Release without a record', () => {
  it('says there is no comparison for the tag, with a link back to the list', async () => {
    vi.stubGlobal('fetch', vi.fn(fixtureFetch()))
    const el = await mount(route())
    expect(text(el)).toContain(`沒有 ${TAG} 的比對紀錄`)
    expect(el.querySelector('.notice')).toBeNull()
    expect(el.querySelector('a')?.getAttribute('href')).toBe('#/u')
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
    expect(buttons.map((b) => text(b))).toEqual(['卡片', '表格', '依原因'])
    expect(buttons.map((b) => b.getAttribute('aria-pressed'))).toEqual(['true', 'false', 'false'])
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
    expect(buttons.map((b) => b.getAttribute('aria-pressed'))).toEqual(['false', 'true', 'false'])
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

const E = (ptr: string, file: string, kind: UpdateEntry['kind'], causes: UpdateEntry['causes']): UpdateEntry => ({
  file,
  field: 'name',
  ptr,
  kind,
  new_kr: 'kr',
  new_zh: 'zh',
  new_origin: 'claude',
  causes,
})
const causeEntries: UpdateEntry[] = [
  E('/0', 'A.json', 'new', [{ kind: 'new' }]),
  E('/1', 'A.json', 'kr-changed', [{ kind: 'kr-changed' }]),
  E('/2', 'B.json', 'zh-changed', [{ kind: 'rewrite', id: 'x' }, { kind: 'other' }]),
  E('/3', 'A.json', 'zh-changed', [{ kind: 'rewrite', id: 'x' }]),
  E('/4', 'B.json', 'zh-changed', [{ kind: 'unknown', note: 'rule file unreadable' }]),
]
const withCauses = () =>
  record({
    entries: causeEntries,
    causes_version: 1,
    rules: { 'rewrite:x': { kind: 'rewrite', id: 'x', path: 'rules/rewrites/x.yaml', title: 'Fix the name', commit: 'abcdef0123456789' } },
  })
const causesView: Settings = { ...defaultSettings(), updateView: 'causes' }
const adjustmentEntries: UpdateEntry[] = [
  E('/a1', 'A.json', 'zh-changed', [{ kind: 'adjustment', id: 'fix.yaml', reason: '改正' }]),
  E('/a2', 'B.json', 'zh-changed', [{ kind: 'adjustment', id: 'fix.yaml', reason: '改正' }]),
  E('/a3', 'A.json', 'zh-changed', [{ kind: 'adjustment', id: 'fix.yaml', reason: '語氣' }]),
  E('/a4', 'A.json', 'kr-changed', [{ kind: 'kr-changed' }, { kind: 'adjustment-expired', id: 'old.yaml', reason: '舊理由' }]),
]
const withAdjustments = () =>
  record({
    entries: adjustmentEntries,
    causes_version: 1,
    rules: {
      'adjustment:fix.yaml': { kind: 'adjustment', id: 'fix.yaml', path: 'rules/adjustments/fix.yaml', title: 'fix.yaml', commit: 'abcdef0123456789' },
      'adjustment-expired:old.yaml': { kind: 'adjustment-expired', id: 'old.yaml', path: 'rules/adjustments/old.yaml', title: 'old.yaml', commit: '0123456789abcdef' },
    },
  })
const heads = (el: HTMLElement) => [...el.querySelectorAll<HTMLButtonElement>('.cg-head')]

describe('UpdateView by cause', () => {
  it('shows every group collapsed at first, with only the headers', async () => {
    stub({}, withCauses())
    const el = await mount(route(), causesView)
    const hs = heads(el)
    expect(hs.length).toBe(5)
    expect(hs.every((h) => h.getAttribute('aria-expanded') === 'false')).toBe(true)
    expect(el.querySelectorAll('.uc-card').length).toBe(0)
    // Rule groups first, then the spec's order.
    expect(hs.map((h) => text(h.querySelector('.cg-name')))).toEqual(['改寫檔 x', 'KR 改版', '新增', '其他', '無法判定'])
  })

  it('shows the kind, id, title, counts, path and the first 7 chars of the commit in the header', async () => {
    stub({}, withCauses())
    const h = heads(await mount(route(), causesView))[0]!
    expect(text(h)).toContain('改寫檔 x')
    expect(text(h)).toContain('Fix the name')
    expect(text(h)).toContain('2 筆、2 個檔案')
    expect(text(h)).toContain('rules/rewrites/x.yaml')
    expect(text(h)).toContain('abcdef0')
    expect(text(h)).not.toContain('abcdef01')
  })

  it('expands a group when its header is clicked, and collapses it again', async () => {
    stub({}, withCauses())
    const el = await mount(route(), causesView)
    const h = heads(el)[0]!
    await act(() => h.click())
    expect(h.getAttribute('aria-expanded')).toBe('true')
    expect(el.querySelector(`#${CSS.escape(h.getAttribute('aria-controls')!)}`)).not.toBeNull()
    expect(el.querySelectorAll('.uc-card').length).toBe(2)
    await act(() => h.click())
    expect(el.querySelectorAll('.uc-card').length).toBe(0)
  })

  it('expands and collapses all groups from the top', async () => {
    stub({}, withCauses())
    const el = await mount(route(), causesView)
    const button = (name: string) => [...el.querySelectorAll<HTMLButtonElement>('.cg-all button')].find((b) => text(b) === name)!
    await act(() => button('全部展開').click())
    expect(heads(el).every((h) => h.getAttribute('aria-expanded') === 'true')).toBe(true)
    // 2 + 1 + 1 + 1 + 1 entries, the one with two causes drawn in both groups.
    expect(el.querySelectorAll('.uc-card').length).toBe(6)
    await act(() => button('全部收合').click())
    expect(el.querySelectorAll('.uc-card').length).toBe(0)
  })

  it('opens the group of the g parameter, whatever the saved view', async () => {
    stub({}, withCauses())
    const el = await mount(route({ group: 'rewrite:x' }))
    const hs = heads(el)
    expect(hs.map((h) => h.getAttribute('aria-expanded'))).toEqual(['true', 'false', 'false', 'false', 'false'])
    expect(el.querySelectorAll('.uc-card').length).toBe(2)
  })

  it('marks, on a card in a group, the other groups it also belongs to', async () => {
    stub({}, withCauses())
    const el = await mount(route({ group: 'rewrite:x' }))
    const also = [...el.querySelectorAll('.uc-also')]
    expect(also.length).toBe(1)
    expect(text(also[0])).toContain('也屬於')
    expect(text(also[0])).toContain('其他')
    expect(text(also[0])).not.toContain('改寫檔')
  })

  it('lists the notes of the unknown group, framed in Chinese', async () => {
    stub({}, withCauses())
    const el = await mount(route({ group: 'unknown' }))
    expect(text(el.querySelector('.cg-notes'))).toContain('說明：rule file unreadable')
  })

  it('shows a record without causes as before, in the legacy groups, with a notice', async () => {
    stub()
    const el = await mount(route(), causesView)
    expect(text(el)).toContain('這個 Release 的紀錄尚未補上原因，暫以舊的分類顯示。')
    expect(heads(el).length).toBeGreaterThan(0)
    const cards = await mount()
    expect(cards.querySelector('.cg-head')).toBeNull()
    expect(text(cards)).not.toContain('尚未補上原因')
    expect(cards.querySelector('.uc-cause')).not.toBeNull()
  })

  it('filters by the kind of cause: options are the kinds present, and the choice goes to the URL', async () => {
    stub({}, withCauses())
    const el = await mount()
    const sel = el.querySelector<HTMLSelectElement>('select[name="cause"]')!
    expect([...sel.options].map((o) => o.textContent)).toEqual(['全部', '改寫檔', 'KR 改版', '新增', '其他', '無法判定'])
    sel.value = 'rewrite'
    await act(() => {
      sel.dispatchEvent(new Event('change', { bubbles: true }))
    })
    expect(location.hash).toBe('#/u/zh-TW-20261105.1?c=rewrite')
  })

  it('keeps only the entries of the chosen cause kind, in the cards and in the group counts', async () => {
    stub({}, withCauses())
    expect((await mount(route({ cause: 'other' }))).querySelectorAll('.uc-card').length).toBe(1)
    const el = await mount(route({ cause: 'rewrite' }), causesView)
    expect(heads(el).map((h) => text(h.querySelector('.cg-name')))).toEqual(['改寫檔 x', '其他'])
    expect(text(heads(el)[1])).toContain('1 筆、1 個檔案')
  })

  it('draws a chip per cause on a card, linking to its group with the other filters kept', async () => {
    stub({}, withCauses())
    const el = await mount(route({ file: 'B.json' }))
    const chips = [...el.querySelectorAll<HTMLAnchorElement>('a.uc-cause')]
    const rewrite = chips.find((c) => text(c) === '改寫檔 x')!
    expect(rewrite.getAttribute('href')).toBe('#/u/zh-TW-20261105.1?f=B.json&g=rewrite%3Ax')
    expect(rewrite.getAttribute('title')).toContain('x')
  })

  it('draws the chips in the table rows too', async () => {
    stub({}, withCauses())
    const el = await mount(route(), { ...defaultSettings(), updateView: 'table' })
    expect(el.querySelectorAll('.ut-row a.uc-cause').length).toBe(6)
  })

  it('lists the reasons of an adjustment group with their counts, and shows its own on each card', async () => {
    stub({}, withAdjustments())
    const el = await mount(route({ group: 'adjustment:fix.yaml' }))
    expect([...el.querySelectorAll('.cg-reasons li')].map(text)).toEqual(['理由：改正（2 筆）', '理由：語氣（1 筆）'])
    // A.json (/a1, /a3), then B.json (/a2).
    expect([...el.querySelectorAll('.cg-panel .uc-card .uc-reason')].map(text)).toEqual(['理由：改正', '理由：語氣', '理由：改正'])
  })

  it('does the same for an expired adjustment, next to the chips of the other groups', async () => {
    stub({}, withAdjustments())
    const el = await mount(route({ group: 'adjustment-expired:old.yaml' }))
    expect([...el.querySelectorAll('.cg-reasons li')].map(text)).toEqual(['理由：舊理由（1 筆）'])
    const card = el.querySelector('.cg-panel .uc-card')!
    expect(text(card.querySelector('.uc-reason'))).toBe('理由：舊理由')
    expect(text(card.querySelector('.uc-also'))).toContain('KR 改版')
  })

  it('shows the reason of an adjustment next to its chip on a card, and keeps it in the title', async () => {
    stub({}, withAdjustments())
    const el = await mount()
    const card = el.querySelector('.uc-card')!
    const chip = card.querySelector('a.uc-cause')!
    expect(text(chip)).toBe('人工調整 fix.yaml')
    expect(chip.getAttribute('title')).toBe('人工調整 fix.yaml（改正）')
    expect(text(chip.nextElementSibling)).toBe('理由：改正')
    expect(chip.nextElementSibling!.classList.contains('uc-reason')).toBe(true)
  })

  it('names an adjustment file once in its header, and keeps the mark with the name', async () => {
    stub({}, withAdjustments())
    const adj = heads(await mount(route(), causesView))[0]!
    expect(text(adj.querySelector('.cg-name'))).toBe('人工調整 fix.yaml')
    expect(adj.querySelector('.cg-subtitle')).toBeNull()
    expect(adj.querySelector('.cg-lead > .cg-mark + .cg-name')).not.toBeNull()
    clearDataCache()
    stub({}, withCauses())
    const rewrite = heads(await mount(route(), causesView))[0]!
    expect(text(rewrite.querySelector('.cg-subtitle'))).toBe('Fix the name')
  })

  it('drops g from the URL when another view is chosen', async () => {
    stub({}, withCauses())
    const seen: Settings[] = []
    const el = await mount(route({ group: 'rewrite:x' }), defaultSettings(), (s) => seen.push(s))
    const buttons = [...el.querySelectorAll<HTMLButtonElement>('.uv-views button')]
    expect(buttons.map((b) => text(b))).toEqual(['卡片', '表格', '依原因'])
    expect(buttons[2]!.getAttribute('aria-pressed')).toBe('true')
    await act(() => buttons[1]!.click())
    expect(location.hash).toBe('#/u/zh-TW-20261105.1')
    expect(seen[0]!.updateView).toBe('table')
  })
})

describe('UpdateView by cause: jumping, switching and paging', () => {
  const scrollIntoView = vi.fn()
  beforeEach(() => {
    scrollIntoView.mockReset()
    Object.defineProperty(Element.prototype, 'scrollIntoView', { value: scrollIntoView, configurable: true, writable: true })
  })
  afterEach(() => {
    Reflect.deleteProperty(Element.prototype, 'scrollIntoView')
  })

  const draw = (container: HTMLElement, r: UpdateRoute) =>
    render(<UpdateView route={r} settings={causesView} onSettings={() => {}} />, container)
  const newContainer = () => {
    const c = document.createElement('div')
    document.body.append(c)
    containers.push(c)
    return c
  }

  it('opens another group and scrolls to its header when g changes while shown', async () => {
    stub({}, withCauses())
    const c = newContainer()
    await act(() => draw(c, route({ group: 'rewrite:x' })))
    await vi.waitFor(() => expect(c.querySelector('.cg-head')).not.toBeNull())
    expect(scrollIntoView).toHaveBeenCalledTimes(1)
    expect(scrollIntoView.mock.contexts[0]).toBe(c.querySelector('#cg-group-rewrite\\:x'))
    await act(() => draw(c, route({ group: 'new' })))
    const head = heads(c).find((h) => text(h.querySelector('.cg-name')) === '新增')!
    expect(head.getAttribute('aria-expanded')).toBe('true')
    expect(scrollIntoView).toHaveBeenCalledTimes(2)
    const target = scrollIntoView.mock.contexts[1] as HTMLElement
    expect(target.contains(head)).toBe(true)
    expect(target.hidden).toBe(false)
  })

  it('opens, scrolls to and focuses the group of a chip whose g the URL has already', async () => {
    stub({}, withCauses())
    const c = newContainer()
    await act(() => draw(c, route({ group: 'rewrite:x' })))
    await vi.waitFor(() => expect(c.querySelector('.cg-head')).not.toBeNull())
    const button = (name: string) => [...c.querySelectorAll<HTMLButtonElement>('.cg-all button')].find((b) => text(b) === name)!
    await act(() => button('全部收合').click())
    const other = heads(c).find((h) => text(h.querySelector('.cg-name')) === '其他')!
    await act(() => other.click())
    other.focus()
    scrollIntoView.mockReset()
    const chip = [...c.querySelectorAll<HTMLAnchorElement>('.cg-panel a.uc-cause')].find((a) => text(a) === '改寫檔 x')!
    expect(chip.getAttribute('href')).toBe('#/u/zh-TW-20261105.1?g=rewrite%3Ax')
    await act(() => chip.click())
    const rewrite = heads(c)[0]!
    expect(rewrite.getAttribute('aria-expanded')).toBe('true')
    expect(scrollIntoView).toHaveBeenCalledTimes(1)
    expect(scrollIntoView.mock.contexts[0]).toBe(c.querySelector('#cg-group-rewrite\\:x'))
    expect(document.activeElement).toBe(rewrite)
  })

  it('moves the focus to the header of the group a chip of the cards leads to', async () => {
    stub({}, withCauses())
    const c = newContainer()
    await act(() => render(<UpdateView route={route()} settings={defaultSettings()} onSettings={() => {}} />, c))
    await vi.waitFor(() => expect(c.querySelector('.uc-card')).not.toBeNull())
    const chip = [...c.querySelectorAll<HTMLAnchorElement>('a.uc-cause')].find((a) => text(a) === '改寫檔 x')!
    chip.focus()
    await act(() => chip.click())
    // The app draws the route of the new hash: the chip is gone.
    await act(() => render(<UpdateView route={route({ group: 'rewrite:x' })} settings={defaultSettings()} onSettings={() => {}} />, c))
    const rewrite = heads(c)[0]!
    expect(rewrite.getAttribute('aria-expanded')).toBe('true')
    expect(document.activeElement).toBe(rewrite)
  })

  it('starts a Release collapsed again after another one was open', async () => {
    const TAG_B = 'zh-TW-20261106.1'
    vi.stubGlobal(
      'fetch',
      vi.fn(fixtureFetch({ [`updates/${TAG}.json`]: withCauses(), [`updates/${TAG_B}.json`]: { ...withCauses(), tag: TAG_B } })),
    )
    const c = newContainer()
    await act(() => draw(c, route()))
    await vi.waitFor(() => expect(c.querySelector('.cg-head')).not.toBeNull())
    await act(() => heads(c)[0]!.click())
    expect(heads(c)[0]!.getAttribute('aria-expanded')).toBe('true')
    await act(() => draw(c, route({ tag: TAG_B })))
    await vi.waitFor(() => expect(c.querySelector('.cg-head')).not.toBeNull())
    expect(heads(c).every((h) => h.getAttribute('aria-expanded') === 'false')).toBe(true)
    await act(() => draw(c, route({ tag: TAG_B, group: 'rewrite:x' })))
    expect(heads(c)[0]!.getAttribute('aria-expanded')).toBe('true')
  })

  it('puts the files of an opened group one heading level below the group', async () => {
    stub({}, withCauses())
    const el = await mount(route({ group: 'rewrite:x' }))
    expect(el.querySelector('.cg-title')!.tagName).toBe('H2')
    expect(el.querySelectorAll('.cg-panel h3.uv-file').length).toBe(2)
    expect(el.querySelectorAll('.cg-panel h2').length).toBe(0)
  })

  it('pages each group on its own', async () => {
    const big = (kind: 'new' | 'kr-changed') =>
      Array.from({ length: UPDATE_PAGE + 2 }, (_, k) => E(`/${kind}${k}`, 'A.json', kind, [{ kind }]))
    stub({}, record({ entries: [...big('new'), ...big('kr-changed')], causes_version: 1 }))
    const el = await mount(route(), causesView)
    await act(() => heads(el).forEach((h) => h.click()))
    const panels = [...el.querySelectorAll<HTMLElement>('.cg-panel')]
    const more = (p: HTMLElement) => [...p.querySelectorAll<HTMLButtonElement>('button')].find((b) => text(b).startsWith('再顯示'))
    expect(panels.map((p) => p.querySelectorAll('.uc-card').length)).toEqual([UPDATE_PAGE, UPDATE_PAGE])
    expect(text(more(panels[0]!))).toBe('再顯示 2 筆')
    await act(() => more(panels[0]!)!.click())
    expect(panels.map((p) => p.querySelectorAll('.uc-card').length)).toEqual([UPDATE_PAGE + 2, UPDATE_PAGE])
    expect(more(panels[0]!)).toBeUndefined()
    expect(more(panels[1]!)).toBeDefined()
  })

  it('filters a legacy record by the legacy kind, in the cards and in the groups', async () => {
    stub()
    const sel = (await mount()).querySelector<HTMLSelectElement>('select[name="cause"]')!
    expect([...sel.options].map((o) => o.value)).toEqual(['', 'kr-changed', 'new', 'deleted', 'late', 'retranslated', 'adjustment'])
    expect((await mount(route({ cause: 'late' }))).querySelectorAll('.uc-card').length).toBe(1)
    const el = await mount(route({ cause: 'adjustment' }), causesView)
    expect(heads(el).map((h) => text(h.querySelector('.cg-name')))).toEqual(['人工調整'])
  })
})
