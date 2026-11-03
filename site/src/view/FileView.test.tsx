import { render } from 'preact'
import { useState } from 'preact/hooks'
import { act } from 'preact/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { clearDataCache } from '../data'
import { fileHref } from '../route'
import { defaultSettings, type Settings } from '../settings'
import { fixture, fixtureFetch, runEffectsAtOnce } from '../testdata'
import type { FileData, Index, Lang, Origin } from '../types'
import { FileView } from './FileView'

const index = fixture<Index>('index.json')
const STORY = 'StoryData/S9999.json'
const EMPTY = 'Passives-empty.json'
const SKILLS = 'Skills_personality-99.json'

const containers: HTMLElement[] = []
const scrollIntoView = vi.fn()

interface Options {
  ptr?: string
  settings?: Settings
  index?: Index
}

let latest: Settings = defaultSettings()

function Harness({ path, ptr, settings, index: idx }: { path: string } & Options) {
  const [s, setS] = useState(settings ?? defaultSettings())
  latest = s
  return (
    <FileView
      path={path}
      ptr={ptr}
      index={idx ?? index}
      settings={s}
      onSettings={(next) => {
        latest = next
        setS(next)
      }}
    />
  )
}

function mount(path: string, opts: Options = {}): HTMLElement {
  const container = document.createElement('div')
  document.body.append(container)
  containers.push(container)
  render(<Harness path={path} {...opts} />, container)
  return container
}

async function open(path: string, opts: Options = {}): Promise<HTMLElement> {
  const el = mount(path, opts)
  await vi.waitFor(() => expect(el.querySelector('.file-body, .notice, .file-gone')).not.toBeNull())
  return el
}

function row(el: ParentNode, ptr: string): HTMLElement | null {
  return el.querySelector<HTMLElement>(`[data-ptr="${ptr}"]`)
}

function cell(el: ParentNode, ptr: string, lang: Lang): HTMLElement | null {
  return el.querySelector<HTMLElement>(`[data-ptr="${ptr}"] [data-lang="${lang}"]`)
}

function press(key: string, target: EventTarget = document.body): void {
  act(() => {
    target.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true }))
  })
}

function big(n: number): FileData {
  return {
    path: 'Big.json',
    family: 'ally',
    units: Array.from({ length: n }, (_, k) => ({ p: `/dataList/${k}/name`, f: 'name', i: k, kr: `kr${k}`, zh: `zh${k}`, o: 'paratranz' as Origin })),
  }
}

const bigIndex: Index = {
  families: index.families.map((f) => (f.id === 'ally' ? { ...f, files: [...f.files, index.files.length] } : f)),
  files: [...index.files, { path: 'Big.json', family: 'ally', label: '', speakers: [], units: 1500, origins: { paratranz: 1500 }, missing: { en: 1500, jp: 1500 } }],
}

let undoEffects = () => {}

beforeEach(() => {
  undoEffects = runEffectsAtOnce()
  clearDataCache()
  localStorage.clear()
  location.hash = ''
  scrollIntoView.mockReset()
  Object.defineProperty(Element.prototype, 'scrollIntoView', { value: scrollIntoView, configurable: true, writable: true })
  vi.stubGlobal('fetch', vi.fn(fixtureFetch({ 'files/Big.json': big(1500) })))
})

afterEach(() => {
  undoEffects()
  for (const container of containers.splice(0)) {
    render(null, container)
    container.remove()
  }
  Reflect.deleteProperty(Element.prototype, 'scrollIntoView')
  vi.unstubAllGlobals()
})

describe('FileView: modes', () => {
  it('shows a StoryData file as a script and any other file as a table', async () => {
    const story = await open(STORY)
    expect(story.querySelector('.script')).not.toBeNull()
    expect(story.querySelector('.tbl')).toBeNull()
    const skills = await open(SKILLS)
    expect(skills.querySelector('.tbl')).not.toBeNull()
    expect(skills.querySelector('.script')).toBeNull()
  })

  it('follows scriptMode on and off over the kind of file', async () => {
    const on = await open(SKILLS, { settings: { ...defaultSettings(), scriptMode: 'on' } })
    expect(on.querySelector('.script')).not.toBeNull()
    const off = await open(STORY, { settings: { ...defaultSettings(), scriptMode: 'off' } })
    expect(off.querySelector('.tbl')).not.toBeNull()
    expect(off.querySelector('.script')).toBeNull()
  })

  it('switches the layout from the select, and writes it to the settings', async () => {
    const el = await open(SKILLS)
    const select = el.querySelector<HTMLSelectElement>('select[data-setting="scriptMode"]')
    expect(select?.value).toBe('auto')
    act(() => {
      if (select) select.value = 'on'
      select?.dispatchEvent(new Event('change', { bubbles: true }))
    })
    expect(latest.scriptMode).toBe('on')
    expect(el.querySelector('.script')).not.toBeNull()
  })

  it('sets the number of language columns for the stylesheet', async () => {
    const el = await open(SKILLS)
    expect(el.querySelector<HTMLElement>('.tbl')?.style.getPropertyValue('--cols')).toBe('4')
  })
})

describe('FileView: table', () => {
  it('has one row for each unit that is shown, the field name first', async () => {
    const el = await open(SKILLS)
    const rows = [...el.querySelectorAll<HTMLElement>('.tbl [data-row]')]
    // five units: the copy one is hidden
    expect(rows.map((r) => r.dataset.ptr)).toEqual([
      '/dataList/0/levelList/0/name',
      '/dataList/0/levelList/0/coinlist/0/coindescs/0/desc',
      '/dataList/0/levelList/0/coinlist/0/coindescs/1/desc',
      '/dataList/1/levelList/0/name',
      '/dataList/1/levelList/0/desc',
    ])
    const field = rows[0]?.querySelector<HTMLElement>('.field')
    expect(field?.textContent).toBe('name')
    expect(field?.getAttribute('title')).toBe('/dataList/0/levelList/0/name')
  })

  it('sets the lang attribute on the cell of each language', async () => {
    for (const path of [SKILLS, STORY]) {
      const el = await open(path)
      const expected: Record<Lang, string> = { kr: 'ko', jp: 'ja', en: 'en', zh: 'zh-Hant-TW' }
      for (const lang of ['kr', 'jp', 'en', 'zh'] as const) {
        const cells = [...el.querySelectorAll<HTMLElement>(`.file-body [data-lang="${lang}"]`)]
        expect(cells.length).toBeGreaterThan(0)
        expect(cells.every((c) => c.getAttribute('lang') === expected[lang])).toBe(true)
      }
    }
  })

  it('missing language cell shows dash', async () => {
    const el = await open(SKILLS)
    const ptr = '/dataList/1/levelList/0/desc'
    expect(cell(el, ptr, 'jp')?.textContent).toBe('—')
    expect(cell(el, ptr, 'en')?.textContent).toBe('—')
    expect(cell(el, ptr, 'kr')?.textContent).toContain('방어 시')
    // a unit with every language has no dash
    expect(cell(el, '/dataList/0/levelList/0/name', 'jp')?.textContent).toBe('試験用の斬撃')
  })

  it('marks the TW text with its origin as text', async () => {
    const el = await open(SKILLS)
    const claude = cell(el, '/dataList/0/levelList/0/coinlist/0/coindescs/0/desc', 'zh')
    expect(claude?.textContent).toContain('claude')
    expect(claude?.querySelector('.origin-claude')).not.toBeNull()
    expect(cell(el, '/dataList/0/levelList/0/name', 'zh')?.textContent).toContain('paratranz')
    expect(cell(el, '/dataList/1/levelList/0/desc', 'zh')?.textContent).toContain('stand-in')
    // only the TW cell has it
    expect(cell(el, '/dataList/0/levelList/0/coinlist/0/coindescs/0/desc', 'kr')?.textContent).not.toContain('claude')
  })

  it('shows the flags of a record in the TW cell', async () => {
    const el = await open(STORY, { settings: { ...defaultSettings(), scriptMode: 'off' } })
    expect(cell(el, '/dataList/2/content', 'zh')?.textContent).toContain('needs_review')
  })

  it('has no EN column when EN is off', async () => {
    const s = defaultSettings()
    s.langs.en = false
    const el = await open(SKILLS, { settings: s })
    expect(el.querySelector('[data-lang="en"]')).toBeNull()
    expect(el.querySelector<HTMLElement>('.tbl')?.style.getPropertyValue('--cols')).toBe('3')
    expect([...el.querySelectorAll('.thead .head-lang')].map((h) => h.textContent)).toEqual(['KR', 'JP', 'TW'])
  })

  it('says so when every language is off', async () => {
    const s = defaultSettings()
    s.langs = { kr: false, jp: false, en: false, zh: false }
    const el = await open(SKILLS, { settings: s })
    expect(el.querySelector('.file-note')?.textContent).toContain('語言')
  })

  it('shows tags as the game draws them, and as they are in raw mode', async () => {
    const plain = await open(STORY, { settings: { ...defaultSettings(), scriptMode: 'off' } })
    expect(plain.querySelector('ruby')).not.toBeNull()
    expect(plain.querySelector('.tmp-raw')).toBeNull()
    const raw = await open(STORY, { settings: { ...defaultSettings(), scriptMode: 'off', rawTags: true } })
    expect(raw.querySelector('ruby')).toBeNull()
    expect(raw.querySelector('.tmp-raw')?.textContent).toBeTruthy()
  })

  it('turns raw tags on from the toolbar', async () => {
    const el = await open(STORY)
    const box = el.querySelector<HTMLInputElement>('input[data-setting="rawTags"]')
    expect(box?.checked).toBe(false)
    act(() => box?.click())
    expect(latest.rawTags).toBe(true)
    expect(el.querySelector('ruby')).toBeNull()
  })

  it('renders a file with no strings without an error', async () => {
    const el = await open(EMPTY)
    expect(el.querySelector('.notice')).toBeNull()
    expect(el.querySelector('.file-note')?.textContent).toContain('沒有文字')
    expect(el.querySelector('[data-row]')).toBeNull()
  })

  it('copies the deep link of a unit when its field name is clicked, and says so', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true })
    const el = await open(SKILLS)
    const ptr = '/dataList/0/levelList/0/coinlist/0/coindescs/0/desc'
    act(() => row(el, ptr)?.querySelector<HTMLButtonElement>('.field')?.click())
    expect(writeText).toHaveBeenCalledWith(`${location.origin}${location.pathname}${fileHref(SKILLS, ptr)}`)
    await vi.waitFor(() => expect(el.querySelector('.toast')?.textContent).toContain('已複製'))
    Reflect.deleteProperty(navigator, 'clipboard')
  })

  it('puts the link in the address bar where it cannot copy', async () => {
    Object.defineProperty(navigator, 'clipboard', { value: undefined, configurable: true })
    const el = await open(SKILLS)
    const ptr = '/dataList/1/levelList/0/name'
    act(() => row(el, ptr)?.querySelector<HTMLButtonElement>('.field')?.click())
    expect(location.hash).toBe(fileHref(SKILLS, ptr))
    Reflect.deleteProperty(navigator, 'clipboard')
  })
})

describe('FileView: origin filter', () => {
  it('hides copy units by default and shows them when copy is ticked', async () => {
    const el = await open(SKILLS)
    expect(row(el, '/dataList/0/code')).toBeNull()
    const box = el.querySelector<HTMLInputElement>('input[data-origin="copy"]')
    expect(box?.checked).toBe(false)
    act(() => box?.click())
    expect(latest.origins.copy).toBe(true)
    expect(row(el, '/dataList/0/code')).not.toBeNull()
  })

  it('hides the units of an origin that is unticked', async () => {
    const el = await open(SKILLS)
    const claude = '/dataList/0/levelList/0/coinlist/0/coindescs/0/desc'
    expect(row(el, claude)).not.toBeNull()
    act(() => el.querySelector<HTMLInputElement>('input[data-origin="claude"]')?.click())
    expect(row(el, claude)).toBeNull()
    expect(row(el, '/dataList/0/levelList/0/name')).not.toBeNull()
  })

  it('offers the origins that the file has, with their counts', async () => {
    const el = await open(SKILLS)
    const boxes = [...el.querySelectorAll<HTMLInputElement>('input[data-origin]')].map((b) => b.dataset.origin)
    expect(boxes).toEqual(['paratranz', 'claude', 'copy', 'fallback', 'stand-in'])
    expect(el.querySelector('input[data-origin="paratranz"]')?.closest('label')?.textContent).toContain('2')
  })

  it('shows the count of the units that are shown', async () => {
    const el = await open(SKILLS)
    expect(el.querySelector('.file-count')?.textContent).toContain('5')
    expect(el.querySelector('.file-count')?.textContent).toContain('6')
  })
})

describe('FileView: script', () => {
  it('makes a passage of each element: the place as a divider, the teller with the title, the lines', async () => {
    const el = await open(STORY)
    const passages = [...el.querySelectorAll<HTMLElement>('.passage')]
    // copy (model) is hidden; the footnote, outside dataList, is a passage of its own
    expect(passages).toHaveLength(5)
    expect(passages[0]?.querySelector('.line-place')?.textContent).toContain('지부 입구 근처')
    expect(passages[1]?.querySelector('.line-place')).toBeNull()
    const speaker = passages[1]?.querySelector('.line-speaker')
    expect(speaker?.querySelector('[data-lang="zh"]')?.textContent).toContain('維吉里烏斯')
    expect(speaker?.querySelector('[data-lang="zh"]')?.textContent).toContain('嚮導')
    expect(passages[3]?.querySelector('.line-place [data-lang="zh"]')?.textContent).toContain('纜車內')
    expect(passages[4]?.querySelector('[data-ptr="/footnote"]')).not.toBeNull()
    expect(el.querySelectorAll('.passage [data-ptr="/footnote"]')).toHaveLength(1)
  })

  it('shows the model lines when copy is on', async () => {
    const s = defaultSettings()
    s.origins.copy = true
    const el = await open(STORY, { settings: s })
    expect(row(el, '/dataList/1/model')).not.toBeNull()
  })

  it('shows a dash in a language the line does not have', async () => {
    const el = await open(STORY)
    expect(cell(el, '/dataList/2/content', 'jp')?.textContent).toBe('—')
    expect(cell(el, '/dataList/2/content', 'en')?.textContent).toContain('What?')
  })

  it('hides a whole passage when its lines are filtered out', async () => {
    const el = await open(STORY)
    expect(row(el, '/dataList/2/teller')).not.toBeNull()
    act(() => el.querySelector<HTMLInputElement>('input[data-origin="claude"]')?.click())
    // passage 2 has a claude line only; the teller goes with it
    expect(row(el, '/dataList/2/content')).toBeNull()
    expect(row(el, '/dataList/2/teller')).toBeNull()
    expect(el.querySelectorAll('.passage')).toHaveLength(4)
    // the place of passage 3 is a claude unit too, so it goes; its line stays
    expect(row(el, '/dataList/3/place')).toBeNull()
    expect(row(el, '/dataList/3/content')).not.toBeNull()
  })

  it('sets the lang attribute on every cell', async () => {
    const el = await open(STORY)
    expect(cell(el, '/dataList/0/content', 'kr')?.getAttribute('lang')).toBe('ko')
    expect(cell(el, '/dataList/0/content', 'jp')?.getAttribute('lang')).toBe('ja')
    expect(cell(el, '/dataList/0/content', 'en')?.getAttribute('lang')).toBe('en')
    expect(cell(el, '/dataList/0/content', 'zh')?.getAttribute('lang')).toBe('zh-Hant-TW')
  })

  it('marks the TW lines with their origin', async () => {
    const el = await open(STORY)
    expect(cell(el, '/dataList/3/content', 'zh')?.textContent).toContain('human')
    expect(cell(el, '/dataList/2/content', 'zh')?.textContent).toContain('claude')
  })
})

describe('FileView: deep link', () => {
  it('marks the row of the unit in the address, and only that row', async () => {
    const ptr = '/dataList/1/levelList/0/name'
    const el = await open(SKILLS, { ptr })
    const marked = [...el.querySelectorAll('.is-target')]
    expect(marked).toHaveLength(1)
    expect(marked[0]?.getAttribute('data-ptr')).toBe(ptr)
  })

  it('marks the line of the unit in a script, and the speaker line for a teller or a title', async () => {
    const content = await open(STORY, { ptr: '/dataList/3/content' })
    expect(content.querySelector('.is-target')?.getAttribute('data-ptr')).toBe('/dataList/3/content')
    const title = await open(STORY, { ptr: '/dataList/1/title' })
    expect(title.querySelectorAll('.is-target')).toHaveLength(1)
    expect(title.querySelector('.is-target')?.classList.contains('line-speaker')).toBe(true)
    const place = await open(STORY, { ptr: '/dataList/3/place' })
    expect(place.querySelector('.is-target')?.classList.contains('line-place')).toBe(true)
  })

  it('scrolls to the target and focuses it', async () => {
    const ptr = '/dataList/1/levelList/0/name'
    const el = await open(SKILLS, { ptr })
    await vi.waitFor(() => expect(document.activeElement).toBe(row(el, ptr)))
    expect(scrollIntoView).toHaveBeenCalled()
    expect(scrollIntoView.mock.contexts[0]).toBe(row(el, ptr))
  })

  it('has the line of the unit in focus in a script, not the whole passage', async () => {
    const el = await open(STORY, { ptr: '/dataList/3/content' })
    await vi.waitFor(() => expect(document.activeElement).toBe(row(el, '/dataList/3/content')))
  })

  it('tells when the unit is not in the file', async () => {
    const el = await open(SKILLS, { ptr: '/dataList/9/nope' })
    expect(el.querySelector('.target-note')?.textContent).toContain('找不到')
    expect(el.querySelector('.is-target')).toBeNull()
  })

  it('tells when the unit is hidden by the origin filter, and shows it on request', async () => {
    const ptr = '/dataList/0/code'
    const el = await open(SKILLS, { ptr })
    expect(row(el, ptr)).toBeNull()
    const note = el.querySelector('.target-note')
    expect(note?.textContent).toContain('copy')
    act(() => note?.querySelector<HTMLButtonElement>('button')?.click())
    expect(latest.origins.copy).toBe(true)
    expect(row(el, ptr)?.classList.contains('is-target')).toBe(true)
    expect(el.querySelector('.target-note')).toBeNull()
    await vi.waitFor(() => expect(scrollIntoView).toHaveBeenCalled())
  })
})

describe('FileView: reading order', () => {
  it('links to the previous and the next file, in the order of the index', async () => {
    const first = await open(STORY)
    expect(first.querySelector('a.nav-prev')).toBeNull()
    expect(first.querySelector('a.nav-next')?.getAttribute('href')).toBe(fileHref(EMPTY))
    const middle = await open(EMPTY)
    expect(middle.querySelector('a.nav-prev')?.getAttribute('href')).toBe(fileHref(STORY))
    expect(middle.querySelector('a.nav-next')?.getAttribute('href')).toBe(fileHref(SKILLS))
    const last = await open(SKILLS)
    expect(last.querySelector('a.nav-next')).toBeNull()
    expect(last.querySelector('a.nav-prev')?.getAttribute('href')).toBe(fileHref(EMPTY))
  })

  it('goes to the previous and the next file on [ and ]', async () => {
    await open(EMPTY)
    press(']')
    expect(location.hash).toBe(fileHref(SKILLS))
    press('[')
    expect(location.hash).toBe(fileHref(STORY))
  })

  it('does nothing on [ and ] at the ends', async () => {
    await open(STORY)
    press('[')
    expect(location.hash).toBe('')
  })

  it('leaves the keys to a text box that has the focus', async () => {
    await open(EMPTY)
    const input = document.createElement('input')
    document.body.append(input)
    input.focus()
    press(']', input)
    press('j', input)
    input.remove()
    expect(location.hash).toBe('')
  })

  it('moves to the next and the previous passage on j and k', async () => {
    const el = await open(STORY)
    const current = () => document.activeElement?.closest('[data-row]')?.getAttribute('data-row')
    press('j')
    expect(current()).toBe('0')
    press('j')
    expect(current()).toBe('1')
    press('j')
    expect(current()).toBe('2')
    press('k')
    expect(current()).toBe('1')
    expect(scrollIntoView).toHaveBeenCalled()
    expect(document.activeElement).toBe(el.querySelectorAll('.passage')[1])
  })

  it('stops at the first and the last row', async () => {
    const el = await open(SKILLS)
    const rows = el.querySelectorAll('[data-row]').length
    press('k')
    expect(document.activeElement?.getAttribute('data-row')).toBe('0')
    for (let n = 0; n < rows + 3; n++) press('j')
    expect(document.activeElement?.getAttribute('data-row')).toBe(String(rows - 1))
  })

  it('goes on from the unit of the deep link', async () => {
    const el = await open(STORY, { ptr: '/dataList/2/content' })
    await vi.waitFor(() => expect(document.activeElement?.closest('[data-row]')?.getAttribute('data-row')).toBe('2'))
    press('j')
    expect(document.activeElement?.getAttribute('data-row')).toBe('3')
    expect(el.querySelector('.is-target')).not.toBeNull()
  })

  it('shows the file asked for, not the one before, when the path changes', async () => {
    const el = mount(SKILLS)
    await vi.waitFor(() => expect(el.querySelector('.tbl')).not.toBeNull())
    render(<Harness path={STORY} />, el)
    await vi.waitFor(() => expect(el.querySelector('.script')).not.toBeNull())
    expect(el.querySelector('.file-title')?.textContent).toBe(STORY)
  })
})

describe('FileView: states', () => {
  it('says so for a file that the index does not have, without asking the server', async () => {
    const fetchSpy = vi.mocked(fetch)
    const el = await open('Nope.json')
    expect(el.querySelector('.file-note')?.textContent).toContain('找不到')
    expect(fetchSpy).not.toHaveBeenCalled()
  })

  it('shows the reload notice when the file is gone from the server', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => new Response('', { status: 404 })))
    const el = await open(SKILLS)
    expect(el.querySelector('.notice')?.textContent).toContain('資料已更新，請重新整理')
  })

  it('shows the title, the place and the speakers in the head', async () => {
    const el = await open(STORY)
    expect(el.querySelector('.file-title')?.textContent).toBe(STORY)
    expect(el.querySelector('.file-label')?.textContent).toContain('分部入口附近')
  })
})

describe('FileView: a large file', () => {
  it('renders only the rows near the top above 1000 rows', async () => {
    const el = await open('Big.json', { index: bigIndex })
    const rows = [...el.querySelectorAll<HTMLElement>('[data-row]')]
    expect(rows.length).toBeGreaterThan(0)
    expect(rows.length).toBeLessThan(150)
    expect(rows[0]?.dataset.row).toBe('0')
    expect(row(el, '/dataList/1400/name')).toBeNull()
    // the rows that are not there are made up in space
    const after = el.querySelector<HTMLElement>('.vspace-after')
    expect(parseFloat(after?.style.height ?? '0')).toBeGreaterThan(1000)
    expect(el.querySelector('.vspace-before')).toBeNull()
  })

  it('renders every row up to 1000 rows', async () => {
    vi.stubGlobal('fetch', vi.fn(fixtureFetch({ 'files/Big.json': big(1000) })))
    const el = await open('Big.json', { index: bigIndex })
    expect(el.querySelectorAll('[data-row]')).toHaveLength(1000)
    expect(el.querySelector('.vspace-after')).toBeNull()
  })

  it('has the row of the deep link in the window, marked, scrolled to and in focus', async () => {
    const ptr = '/dataList/1400/name'
    const el = await open('Big.json', { index: bigIndex, ptr })
    const target = row(el, ptr)
    expect(target).not.toBeNull()
    expect(target?.classList.contains('is-target')).toBe(true)
    expect(el.querySelectorAll('[data-row]').length).toBeLessThan(150)
    expect(row(el, '/dataList/0/name')).toBeNull()
    expect(parseFloat(el.querySelector<HTMLElement>('.vspace-before')?.style.height ?? '0')).toBeGreaterThan(1000)
    await vi.waitFor(() => expect(document.activeElement).toBe(target))
    expect(scrollIntoView.mock.contexts[0]).toBe(target)
  })

  it('does the same in a script', async () => {
    const ptr = '/dataList/1200/name'
    const el = await open('Big.json', { index: bigIndex, ptr, settings: { ...defaultSettings(), scriptMode: 'on' } })
    expect(row(el, ptr)?.classList.contains('is-target')).toBe(true)
    expect(el.querySelectorAll('.passage').length).toBeLessThan(150)
  })

  it('reaches a row far down with j', async () => {
    const ptr = '/dataList/1400/name'
    const el = await open('Big.json', { index: bigIndex, ptr })
    await vi.waitFor(() => expect(document.activeElement).toBe(row(el, ptr)))
    press('j')
    expect(document.activeElement?.getAttribute('data-ptr')).toBe('/dataList/1401/name')
  })
})

describe('FileView: scrolling a large file', () => {
  // jsdom has no layout: say where the rows are. The page is scrolled by `scrollY`;
  // the rows are 100px tall wherever they are.
  let scrollY = 0
  const rect = (top: number, height: number) => ({ top, bottom: top + height, height, left: 0, right: 0, width: 0, x: 0, y: top, toJSON: () => ({}) }) as DOMRect

  beforeEach(() => {
    scrollY = 0
    vi.spyOn(Element.prototype, 'getBoundingClientRect').mockImplementation(function (this: Element) {
      if (this.classList.contains('rows')) return rect(-scrollY, 0)
      if (this.hasAttribute('data-row')) return rect(0, 100)
      return rect(0, 0)
    })
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  /** scrollIntoView as the browser does it here: the page scrolls so that the element is centred. */
  function scrollByCentring(this: HTMLElement): void {
    const list = this.closest('.rows')
    const row = this.closest<HTMLElement>('[data-row]')
    const first = Number(list?.querySelector<HTMLElement>('[data-row]')?.dataset.row)
    const before = parseFloat(list?.querySelector<HTMLElement>('.vspace-before')?.style.height ?? '0')
    scrollY = before + (Number(row?.dataset.row) - first) * 100 - (window.innerHeight - 100) / 2
  }

  const rendered = (el: ParentNode) => [...el.querySelectorAll<HTMLElement>('[data-row]')].map((r) => Number(r.dataset.row))

  async function scrollTo(y: number): Promise<void> {
    scrollY = y
    act(() => {
      window.dispatchEvent(new Event('scroll'))
    })
  }

  it('moves the window with the scroll, and gives the rows that are not there back as space', async () => {
    const el = await open('Big.json', { index: bigIndex })
    expect(Math.min(...rendered(el))).toBe(0)
    await scrollTo(60000)
    // rows not seen yet count 64px, those seen 100px: the row at 60000px is near 926
    await vi.waitFor(() => expect(Math.min(...rendered(el))).toBeGreaterThan(800))
    const rows = rendered(el)
    expect(rows.length).toBeLessThan(60)
    expect(Math.min(...rows)).toBeLessThanOrEqual(926)
    expect(Math.max(...rows)).toBeGreaterThanOrEqual(926)
    // the window is contiguous, and the space before it is what the rows before it take
    expect(rows).toEqual(Array.from({ length: rows.length }, (_, k) => (rows[0] ?? 0) + k))
    const before = parseFloat(el.querySelector<HTMLElement>('.vspace-before')?.style.height ?? '0')
    const first = rows[0] ?? 0
    expect(before).toBeGreaterThan(first * 60)
    expect(before).toBeLessThanOrEqual(first * 100)
  })

  it('brings the first rows back when the page is scrolled to the top again', async () => {
    const el = await open('Big.json', { index: bigIndex })
    await scrollTo(60000)
    await vi.waitFor(() => expect(Math.min(...rendered(el))).toBeGreaterThan(800))
    await scrollTo(0)
    await vi.waitFor(() => expect(Math.min(...rendered(el))).toBe(0))
    expect(el.querySelector('.vspace-before')).toBeNull()
  })

  it('shows rows to the end of the file at the bottom of the page', async () => {
    const el = await open('Big.json', { index: bigIndex })
    await scrollTo(1_000_000)
    await vi.waitFor(() => expect(Math.max(...rendered(el))).toBe(1499))
    expect(el.querySelector('.vspace-after')).toBeNull()
  })

  it('keeps the row of the deep link in view while the rows around it are measured', async () => {
    scrollIntoView.mockImplementation(scrollByCentring)
    const ptr = '/dataList/1400/name'
    const el = await open('Big.json', { index: bigIndex, ptr })
    const target = row(el, ptr)
    expect(target).not.toBeNull()
    expect(target?.classList.contains('is-target')).toBe(true)
    expect(rendered(el).length).toBeLessThan(60)
    expect(scrollY).toBeGreaterThan(1400 * 64 - 1000) // the page went to the row
    // and it stays there when the page then reports its scroll
    await scrollTo(scrollY)
    expect(row(el, ptr)).not.toBeNull()
    expect(document.activeElement).toBe(target)
  })

  it('keeps going from the deep link with j, as far as the window is moved', async () => {
    scrollIntoView.mockImplementation(scrollByCentring)
    const ptr = '/dataList/1400/name'
    const el = await open('Big.json', { index: bigIndex, ptr })
    for (let n = 0; n < 40; n++) press('j')
    expect(document.activeElement?.getAttribute('data-ptr')).toBe('/dataList/1440/name')
    await scrollTo(scrollY)
    expect(row(el, '/dataList/1440/name')).not.toBeNull()
  })

  it('follows the filter: the window stays where the page is', async () => {
    const el = await open('Big.json', { index: bigIndex })
    await scrollTo(30000)
    await vi.waitFor(() => expect(Math.min(...rendered(el))).toBeGreaterThan(200))
    // 1500 -> 0 rows shown (no origin ticked) -> 1500 again
    act(() => el.querySelector<HTMLInputElement>('input[data-origin="paratranz"]')?.click())
    expect(el.querySelector('.file-note')?.textContent).toContain('篩選')
    act(() => el.querySelector<HTMLInputElement>('input[data-origin="paratranz"]')?.click())
    await vi.waitFor(() => expect(Math.min(...rendered(el))).toBeGreaterThan(200))
  })
})
