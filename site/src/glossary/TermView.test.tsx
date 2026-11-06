import { render } from 'preact'
import { act } from 'preact/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { clearDataCache } from '../data'
import { termHref, type Route } from '../route'
import { defaultSettings } from '../settings'
import { fixture, fixtureFetch, runEffectsAtOnce } from '../testdata'
import type { GlossaryFile, GlossaryIndex, GlossaryUnit } from '../types'
import { TermView } from './TermView'

type TermRoute = Extract<Route, { kind: 'term' }>
const ID = 'glossary/aaaaaaaa/bbbbbbbb'
const index = fixture<GlossaryIndex>('glossary/index.json')
const entry = index.entries.find((e) => e.id === ID)!
const file = fixture<GlossaryFile>('glossary/aaaaaaaa-bbbbbbbb.json')

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

const route = (rest: Partial<TermRoute> = {}): TermRoute => ({ kind: 'term', id: ID, family: '', group: '', chapter: '', character: '', event: '', origin: '', q: '', ...rest })

async function mount(r: TermRoute = route(), docs: Record<string, unknown> = {}): Promise<HTMLElement> {
  vi.stubGlobal('fetch', vi.fn(fixtureFetch(docs)))
  const container = document.createElement('div')
  document.body.append(container)
  containers.push(container)
  render(<TermView route={r} settings={defaultSettings()} />, container)
  await vi.waitFor(() => expect(container.querySelector('.tv-missing, .notice, .rv-missing')).not.toBeNull())
  return container
}

const again = (el: HTMLElement, r: TermRoute) => act(() => render(<TermView route={r} settings={defaultSettings()} />, el))

const missingItems = (el: HTMLElement) => [...el.querySelectorAll<HTMLElement>('.tv-missing li.tv-unit')]
const usedItems = (el: HTMLElement) => [...el.querySelectorAll<HTMLElement>('.tv-used li.tv-unit')]
const heading = (el: HTMLElement, section: string) => el.querySelector(`.${section} h2`)?.textContent

const synthetic = (n: number): GlossaryUnit[] =>
  Array.from({ length: n }, (_, i) => ({ key: `StoryData/S9999.json|/dataList/${i}/content`, family: 'story-main', top: 'StoryData/', origin: 'paratranz', kr: `…대상 ${i}…`, zh: `…目標 ${i}…` }))

describe('TermView, the top', () => {
  it('shows the entry: the words, notes, check, scope and rate', async () => {
    const el = await mount()
    const top = el.querySelector('.tv-top')!
    expect(top.querySelector('h1')?.textContent).toContain('대상')
    expect(top.querySelector('h1')?.textContent).toContain('目標')
    expect(top.textContent).toContain('對象')
    expect(top.textContent).toContain(entry.notes)
    expect(top.textContent).toContain('warn')
    expect(top.textContent).toContain('76.9%')
    expect(top.textContent).toContain('40 / 52')
  })

  it('shows the groups: family and top node, each with units and units with the rendering', async () => {
    const el = await mount()
    const rows = [...el.querySelectorAll('.tv-groups tbody tr')].map((r) => [...r.children].map((c) => c.textContent))
    expect(rows).toEqual(entry.groups.map((g) => [g.name, String(g.n), String(g.used), `${Math.round((g.used / g.n) * 100)}%`]))
  })

  it('links to the warnings of the termgroup, to the entry in the scope tab, and back to the list', async () => {
    const el = await mount()
    const hrefs = [...el.querySelectorAll<HTMLAnchorElement>('.tv-top a')].map((a) => a.getAttribute('href'))
    expect(hrefs).toContain('#/r/warnings?q=termgroup%2Faaaaaaaa%2Fbbbbbbbb')
    expect(hrefs).toContain(`#/r/scope?id=${encodeURIComponent(ID)}`)
    expect(hrefs).toContain('#/g')
  })

  it('has the entry id to copy, with the glossary id as it is', async () => {
    const el = await mount()
    expect(el.querySelector('.tv-top .rv-id code')?.textContent).toBe(ID)
    expect(el.querySelector('.tv-top .rv-id button')?.textContent).toBe('複製 ID')
  })
})

describe('TermView, units without the rendering', () => {
  it('lists every one, with KR (the term marked), TW, source, note and links', async () => {
    const el = await mount()
    const items = missingItems(el)
    expect(items).toHaveLength(file.missing.length)
    expect(heading(el, 'tv-missing')).toBe(`沒用譯名 ${file.missing.length} 項`)
    const first = items[0]!
    expect(first.querySelector('dd mark')?.textContent).toBe('대상')
    expect(first.textContent).toContain(file.missing[0]!.zh)
    expect(first.querySelector('.origin')?.textContent).toBe(file.missing[0]!.origin)
    expect(first.querySelector('.rv-note-text')?.textContent).toBe(file.missing[0]!.note)
    const key = file.missing[0]!.key
    expect(first.querySelector('a')?.getAttribute('href')).toBe(`#/f/Skills_personality-99.json?u=${encodeURIComponent(key.split('|')[1]!)}`)
    expect(first.querySelector('.rv-id code')?.textContent).toBe(file.missing[0]!.id)
  })

  it('marks the confirmed units', async () => {
    const el = await mount()
    const marked = missingItems(el).filter((li) => li.classList.contains('is-confirmed'))
    expect(marked).toHaveLength(file.missing.filter((u) => u.accepted).length)
    expect(marked[0]!.textContent).toContain('已確認')
    expect(marked[0]!.textContent).toContain(file.missing.find((u) => u.accepted)!.accepted!.reason)
  })

  describe('copy these units', () => {
    const copied = async (el: HTMLElement) => {
      const write = vi.fn(async () => {})
      vi.stubGlobal('navigator', { ...navigator, clipboard: { writeText: write } })
      const button = [...el.querySelectorAll<HTMLButtonElement>('.tv-missing button')].find((b) => b.textContent === '複製這些單位')!
      await act(() => {
        button.click()
      })
      await vi.waitFor(() => expect(write).toHaveBeenCalled())
      return (write.mock.calls[0] as unknown as [string])[0]
    }

    it('copies the unconfirmed units, one ID with a piece of the TW and an arrow per line', async () => {
      const text = await copied(await mount())
      const open = file.missing.filter((u) => !u.accepted)
      expect(text.split('\n')).toEqual(open.map((u) => `${u.id}「${u.zh}」→ `))
      expect(open.length).toBeLessThan(file.missing.length)
    })

    it('drops the status when a filter changes', async () => {
      const el = await mount()
      await copied(el)
      expect(el.querySelector('.tv-missing .rv-said')?.textContent).toContain('已複製')
      await again(el, route({ family: 'story-main' }))
      expect(el.querySelector('.tv-missing .rv-said')?.textContent).toBe('')
    })

    it('copies the units that the filters keep', async () => {
      const text = await copied(await mount(route({ family: 'story-main' })))
      const open = file.missing.filter((u) => !u.accepted && u.family === 'story-main')
      expect(text.split('\n')).toEqual(open.map((u) => `${u.id}「${u.zh}」→ `))
    })
  })
})

describe('TermView, units with the rendering', () => {
  it('lists them with KR and TW pieces, the terms marked', async () => {
    const el = await mount()
    expect(usedItems(el)).toHaveLength(file.used.length)
    expect(heading(el, 'tv-used')).toBe(`用了譯名 ${file.used.length} 項`)
    const first = usedItems(el)[0]!
    expect(first.querySelectorAll('dd mark')[0]?.textContent).toBe('대상')
    expect(first.querySelectorAll('dd mark')[1]?.textContent).toBe('目標')
    expect(first.querySelector('.origin')).not.toBeNull()
    expect(first.querySelector('.rv-id')).toBeNull()
  })

  const big = { [`glossary/aaaaaaaa-bbbbbbbb.json`]: { id: ID, missing: [], used: synthetic(250) } }

  it('shows 100 first and 100 more at a time', async () => {
    const el = await mount(route(), big)
    expect(usedItems(el)).toHaveLength(100)
    const more = () => el.querySelector<HTMLButtonElement>('.tv-used .rv-more')
    expect(more()?.textContent).toBe('再顯示 100 個（共 250）')
    await act(() => more()!.click())
    expect(usedItems(el)).toHaveLength(200)
    await act(() => more()!.click())
    expect(usedItems(el)).toHaveLength(250)
    expect(more()).toBeNull()
  })

  it('goes back to 100 when a filter changes', async () => {
    const el = await mount(route(), big)
    await act(() => el.querySelector<HTMLButtonElement>('.tv-used .rv-more')!.click())
    expect(usedItems(el)).toHaveLength(200)
    await again(el, route({ origin: 'paratranz' }))
    expect(usedItems(el)).toHaveLength(100)
    await act(() => el.querySelector<HTMLButtonElement>('.tv-used .rv-more')!.click())
    await again(el, route({ q: '目標' }))
    expect(usedItems(el)).toHaveLength(100)
  })
})

describe('TermView, units without the rendering, paged', () => {
  const missingUnits = (n: number): GlossaryUnit[] =>
    Array.from({ length: n }, (_, i) => ({
      key: `StoryData/S9999.json|/dataList/${i}/content`,
      id: `ID${String(i).padStart(3, '0')}`,
      family: 'story-main',
      top: 'StoryData/',
      origin: i % 2 === 0 ? 'paratranz' : 'human',
      kr: `…대상 ${i}…`,
      zh: `…目標 ${i}…`,
    }))
  const docs = { [`glossary/aaaaaaaa-bbbbbbbb.json`]: { id: ID, missing: missingUnits(150), used: [] } }
  const more = (el: HTMLElement) => el.querySelector<HTMLButtonElement>('.tv-missing .rv-more')

  it('shows 100 first and the rest after one click, with the count over all of them', async () => {
    const el = await mount(route(), docs)
    expect(missingItems(el)).toHaveLength(100)
    expect(heading(el, 'tv-missing')).toBe('沒用譯名 150 項')
    expect(more(el)?.textContent).toBe('再顯示 100 個（共 150）')
    await act(() => more(el)!.click())
    expect(missingItems(el)).toHaveLength(150)
    expect(more(el)).toBeNull()
  })

  it('goes back to 100 when a filter changes', async () => {
    const el = await mount(route(), docs)
    await act(() => more(el)!.click())
    expect(missingItems(el)).toHaveLength(150)
    await again(el, route({ q: '目標' }))
    expect(missingItems(el)).toHaveLength(100)
  })

  it('copies every filtered unconfirmed unit, not only the ones on screen', async () => {
    const el2 = await mount(route(), docs)
    const write = vi.fn(async () => {})
    vi.stubGlobal('navigator', { ...navigator, clipboard: { writeText: write } })
    const button = [...el2.querySelectorAll<HTMLButtonElement>('.tv-missing button')].find((b) => b.textContent === '複製這些單位')!
    await act(() => {
      button.click()
    })
    await vi.waitFor(() => expect(write).toHaveBeenCalled())
    const lines = (write.mock.calls[0] as unknown as [string])[0].split('\n')
    expect(lines).toHaveLength(150)
    expect(lines[149]).toBe('ID149「…目標 149…」→ ')
    expect(missingItems(el2)).toHaveLength(100)
  })
})

describe('TermView, the filters', () => {
  const both = (el: HTMLElement) => [missingItems(el).length, usedItems(el).length]
  const count = (pred: (u: GlossaryUnit) => boolean) => [file.missing.filter(pred).length, file.used.filter(pred).length]

  it('lists the options the units have', async () => {
    const el = await mount()
    const opts = (name: string) => [...el.querySelectorAll<HTMLOptionElement>(`select[name=${name}] option`)].map((o) => o.value)
    expect(opts('family')).toEqual(['', 'skill', 'story-main'])
    expect(opts('group')).toEqual(['', 'Skills', 'StoryData/'])
    expect(opts('chapter')).toEqual(['', '1'])
    expect(opts('character')).toEqual(['', 'faust'])
    expect(opts('event')).toEqual([''])
    expect(opts('origin')).toEqual(['', 'claude', 'human', 'paratranz'])
  })

  it('applies a family to both sections', async () => {
    const el = await mount(route({ family: 'story-main' }))
    expect(both(el)).toEqual(count((u) => u.family === 'story-main'))
    expect(both(el)).toEqual([10, 2])
    expect(el.querySelector<HTMLSelectElement>('select[name=family]')?.value).toBe('story-main')
  })

  it('applies a top node, a chapter, a character and an origin', async () => {
    expect(both(await mount(route({ group: 'Skills' })))).toEqual(count((u) => u.top === 'Skills'))
    expect(both(await mount(route({ chapter: '1' })))).toEqual(count((u) => u.tag?.kind === 'chapter' && u.tag.value === '1'))
    expect(both(await mount(route({ character: 'faust' })))).toEqual(count((u) => u.tag?.kind === 'character' && u.tag.value === 'faust'))
    expect(both(await mount(route({ origin: 'human' })))).toEqual(count((u) => u.origin === 'human'))
  })

  it('applies an event of the tag, and none matches an event the units lack', async () => {
    expect(both(await mount(route({ event: 'walpu6' })))).toEqual([0, 0])
  })

  it('applies the text to KR, TW, note and key', async () => {
    expect(both(await mount(route({ q: '(41)' })))).toEqual([1, 0])
    expect(both(await mount(route({ q: '(5)' })))).toEqual([0, 1])
    expect(both(await mount(route({ q: '跟丟了' })))).toEqual(count((u) => u.zh.includes('跟丟了')))
    expect(both(await mount(route({ q: '此處不用' })))).toEqual([1, 0])
    expect(both(await mount(route({ q: 'S9999.json' })))).toEqual(count((u) => u.key.includes('S9999')))
  })

  it('adds the filters together', async () => {
    expect(both(await mount(route({ family: 'story-main', origin: 'human' })))).toEqual(count((u) => u.family === 'story-main' && u.origin === 'human'))
  })

  it('counts what the filters keep in the headings', async () => {
    const el = await mount(route({ family: 'story-main' }))
    expect(heading(el, 'tv-missing')).toBe('沒用譯名 10 項')
    expect(heading(el, 'tv-used')).toBe('用了譯名 2 項')
  })

  it('goes to the address of a filter that is picked, keeping the others', async () => {
    const el = await mount(route({ origin: 'claude' }))
    const select = el.querySelector<HTMLSelectElement>('select[name=family]')!
    select.value = 'story-main'
    await act(() => {
      select.dispatchEvent(new Event('change', { bubbles: true }))
    })
    expect(location.hash).toBe(termHref(ID, { family: 'story-main', origin: 'claude' }))
  })

  it('says so when no unit is left', async () => {
    const el = await mount(route({ q: '沒有這一串字' }))
    expect(el.textContent).toContain('沒有符合的單位')
  })
})

describe('TermView, what goes wrong', () => {
  it('says 找不到詞條 for an id that the index lacks, and loads no entry file', async () => {
    const el = await mount(route({ id: 'glossary/00000000/11111111' }))
    expect(el.querySelector('.rv-missing')?.textContent).toContain('找不到詞條 glossary/00000000/11111111')
    expect(el.querySelector('.tv-top')).toBeNull()
    const calls = (fetch as unknown as ReturnType<typeof vi.fn>).mock.calls.map((c) => String(c[0]))
    expect(calls.some((u) => u.includes('00000000-11111111'))).toBe(false)
  })

  it('shows the entry and an error when its file cannot be loaded', async () => {
    vi.stubGlobal('fetch', vi.fn(async (u: string) => (u.includes('glossary/index.json') ? fixtureFetch()(u) : new Response('x', { status: 500 }))))
    clearDataCache()
    const container = document.createElement('div')
    document.body.append(container)
    containers.push(container)
    render(<TermView route={route()} settings={defaultSettings()} />, container)
    await vi.waitFor(() => expect(container.querySelector('.notice')).not.toBeNull())
    expect(container.querySelector('.tv-top')).not.toBeNull()
    expect(container.querySelector('.tv-missing')).toBeNull()
  })
})
