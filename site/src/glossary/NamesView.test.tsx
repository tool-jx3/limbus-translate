import { render } from 'preact'
import { act } from 'preact/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { clearDataCache } from '../data'
import { fileHref, type Route } from '../route'
import { fixture, fixtureFetch, runEffectsAtOnce } from '../testdata'
import type { NamesData, NameRow } from '../types'
import { NamesView } from './NamesView'

type NamesRoute = Extract<Route, { kind: 'names' }>
const base = fixture<NamesData>('names.json')

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

const route = (rest: Partial<NamesRoute> = {}): NamesRoute => ({ kind: 'names', q: '', only: '', sort: 'units', ...rest })

async function mount(r: NamesRoute = route(), docs: Record<string, unknown> = {}): Promise<HTMLElement> {
  vi.stubGlobal('fetch', vi.fn(fixtureFetch(docs)))
  const container = document.createElement('div')
  document.body.append(container)
  containers.push(container)
  render(<NamesView route={r} />, container)
  await vi.waitFor(() => expect(container.querySelector('.nm-table, .notice, .rv-empty, .rv-lede')).not.toBeNull())
  return container
}

const rows = (el: HTMLElement) => [...el.querySelectorAll<HTMLElement>('.nm-table tbody tr')]
const manyRows = (n: number): NameRow[] =>
  Array.from({ length: n }, (_, i) => ({ kr: `이름${String(i).padStart(4, '0')}`, units: 1, variants: [{ renderings: [{ zh: `名${i}`, units: 1 }], examples: [] }] }))

describe('NamesView', () => {
  it('shows the lede with the tag and the fields, and the count', async () => {
    const el = await mount()
    expect(el.querySelector('h1')?.textContent).toBe('既有譯名')
    expect(el.querySelector('.rv-lede')?.textContent).toContain('raw-20261029')
    expect(el.querySelector('.rv-lede')?.textContent).toContain('name、title')
    expect(el.querySelector('.rv-count')?.textContent).toBe('3 條（共 3 條）')
  })

  it('lists each name with the JP, EN, main rendering and units of its variants', async () => {
    const el = await mount()
    const first = rows(el)[0]!
    expect(first.textContent).toContain('검')
    expect(first.textContent).toContain('剣')
    expect(first.textContent).toContain('Sword')
    expect(first.textContent).toContain('劍')
    expect(first.textContent).toContain('長劍')
    expect(first.querySelectorAll('.nm-variant')).toHaveLength(2)
    expect(first.querySelector('.nm-variant .nm-main')?.textContent).toBe('劍 3')
    expect(rows(el).map((r) => r.querySelector('.nm-kr')?.textContent)).toEqual(['검', '우레', '방패'])
  })

  it('links each example to its unit in the file view', async () => {
    const el = await mount()
    const link = rows(el)[0]!.querySelector<HTMLAnchorElement>('.nm-example a')!
    expect(link.getAttribute('href')).toBe(fileHref('Skills_personality-02.json', '/dataList/0/name'))
  })

  it('marks the glossary, declined and review names, with the links', async () => {
    const el = await mount()
    const [sword, thunder] = rows(el)
    const g = [...sword!.querySelectorAll<HTMLAnchorElement>('a')].find((a) => a.textContent === '詞彙表已有')!
    expect(g.getAttribute('href')).toBe(`#/g?q=${encodeURIComponent('검')}`)
    const r = [...sword!.querySelectorAll<HTMLAnchorElement>('a')].find((a) => a.textContent === '這次有不同的新譯')!
    expect(r.getAttribute('href')).toBe(`#/r/suggestions?id=${encodeURIComponent('name/검')}`)
    expect(thunder!.textContent).toContain('已不採用')
    expect(thunder!.querySelector('a[href^="#/g"]')).toBeNull()
  })

  it('has no decision or copy button', async () => {
    const el = await mount()
    expect(el.querySelectorAll('.nm-table button')).toHaveLength(0)
  })

  it('says there is no table when raw_tag is empty', async () => {
    const el = await mount(route(), { 'names.json': { raw_tag: '', fields: [], names: [] } })
    expect(el.querySelector('.rv-lede')?.textContent).toContain('還沒有既有譯名表')
    expect(el.querySelector('.nm-table')).toBeNull()
  })

  it('filters by the search and the only select', async () => {
    const el = await mount(route({ q: 'thunder' }))
    expect(rows(el).map((r) => r.querySelector('.nm-kr')?.textContent)).toEqual(['우레'])
    const declined = await mount(route({ only: 'declined' }))
    expect(rows(declined)).toHaveLength(1)
    expect(declined.querySelector('.rv-count')?.textContent).toBe('1 條（共 3 條）')
  })

  it('sorts by KR', async () => {
    const el = await mount(route({ sort: 'kr' }))
    expect(rows(el).map((r) => r.querySelector('.nm-kr')?.textContent)).toEqual(['검', '방패', '우레'])
  })

  it('goes to the address of a changed filter', async () => {
    const el = await mount()
    const select = el.querySelector<HTMLSelectElement>('select[name=only]')!
    select.value = 'multi'
    await act(() => {
      select.dispatchEvent(new Event('change', { bubbles: true }))
    })
    expect(location.hash).toBe('#/n?only=multi')
  })

  it('draws 200 rows first and 200 more at each press', async () => {
    const el = await mount(route(), { 'names.json': { ...base, names: manyRows(450) } })
    expect(rows(el)).toHaveLength(200)
    expect(el.querySelector('.rv-count')?.textContent).toBe('450 條（共 450 條）')
    const more = el.querySelector<HTMLButtonElement>('button.rv-more')!
    expect(more.textContent).toContain('再顯示 200 條')
    await act(() => more.click())
    expect(rows(el)).toHaveLength(400)
    await act(() => el.querySelector<HTMLButtonElement>('button.rv-more')!.click())
    expect(rows(el)).toHaveLength(450)
    expect(el.querySelector('button.rv-more')).toBeNull()
  })

  it('has no more button for 200 rows or fewer', async () => {
    const el = await mount(route(), { 'names.json': { ...base, names: manyRows(200) } })
    expect(rows(el)).toHaveLength(200)
    expect(el.querySelector('button.rv-more')).toBeNull()
  })
})
