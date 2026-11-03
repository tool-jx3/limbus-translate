import { render } from 'preact'
import { act } from 'preact/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { clearDataCache } from '../data'
import { fixture, fixtureFetch, runEffectsAtOnce } from '../testdata'
import type { Meta } from '../types'
import { App } from './App'

const containers: HTMLElement[] = []

function mount(): HTMLElement {
  const container = document.createElement('div')
  document.body.append(container)
  containers.push(container)
  render(<App />, container)
  return container
}

async function loaded(el: HTMLElement): Promise<void> {
  await vi.waitFor(() => expect(el.querySelector('.directory, .notice')).not.toBeNull())
}

function type(input: HTMLInputElement, value: string): void {
  act(() => {
    input.value = value
    input.dispatchEvent(new Event('input', { bubbles: true }))
  })
}

let undoEffects = () => {}

beforeEach(() => {
  undoEffects = runEffectsAtOnce()
  clearDataCache()
  localStorage.clear()
  location.hash = ''
  vi.stubGlobal('fetch', vi.fn(fixtureFetch()))
})

afterEach(() => {
  undoEffects()
  for (const container of containers.splice(0)) {
    render(null, container)
    container.remove()
  }
  vi.unstubAllGlobals()
})

describe('App', () => {
  it('shows the site name, the directory by family, the size of the data and its version', async () => {
    const el = mount()
    expect(el.querySelector('.top .brand')?.textContent).toBe('Limbus 譯文對照')
    await loaded(el)
    expect([...el.querySelectorAll('.dir-family-id')].map((e) => e.textContent)).toEqual(['story-main', 'ally'])
    expect(el.querySelector('.home')?.textContent).toContain('單位20')
    expect(el.querySelector('.foot-version')?.textContent).toContain('raw-20261029')
    expect(el.querySelector('.foot-version a')?.getAttribute('href')).toBe(
      'https://github.com/tool-jx3/limbus-translate/releases/tag/zh-TW-20261029.5',
    )
  })

  it('shows the reload notice, and no directory, when the data is of another format', async () => {
    vi.stubGlobal('fetch', vi.fn(fixtureFetch({ 'meta.json': { ...fixture<Meta>('meta.json'), format: 2 } })))
    const el = mount()
    await loaded(el)
    expect(el.querySelector('.notice')?.textContent).toContain('資料已更新，請重新整理')
    expect(el.querySelector('.directory')).toBeNull()
    expect(el.querySelector('footer')?.textContent).toContain('Project Moon')
  })

  it('has a language switch for each of KR, JP, EN and TW, all on, that writes the settings', async () => {
    const el = mount()
    await loaded(el)
    const boxes = [...el.querySelectorAll<HTMLInputElement>('.langs input[type="checkbox"]')]
    expect(boxes.map((b) => b.parentElement?.textContent)).toEqual(['KR', 'JP', 'EN', 'TW'])
    expect(boxes.every((b) => b.checked)).toBe(true)
    act(() => boxes[1]?.click())
    expect(boxes[1]?.checked).toBe(false)
    expect(JSON.parse(localStorage.getItem('lct-viewer-settings') ?? '{}').langs).toEqual({ kr: true, jp: false, en: true, zh: true })
  })

  it('starts from the saved settings', async () => {
    localStorage.setItem('lct-viewer-settings', JSON.stringify({ langs: { kr: true, jp: true, en: false, zh: true } }))
    const el = mount()
    await loaded(el)
    const boxes = [...el.querySelectorAll<HTMLInputElement>('.langs input[type="checkbox"]')]
    expect(boxes.map((b) => b.checked)).toEqual([true, true, false, true])
  })

  it('goes to the search in TW when the search box is submitted', async () => {
    const el = mount()
    await loaded(el)
    const input = el.querySelector<HTMLInputElement>('form[role="search"] input')
    expect(input).not.toBeNull()
    type(input as HTMLInputElement, '從 & more')
    act(() => {
      el.querySelector('form[role="search"]')?.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
    })
    expect(location.hash).toBe('#/s?q=%E5%BE%9E%20%26%20more&lang=zh')
  })

  it('does not search for nothing', async () => {
    const el = mount()
    await loaded(el)
    const before = location.hash
    type(el.querySelector<HTMLInputElement>('form[role="search"] input') as HTMLInputElement, '   ')
    act(() => {
      el.querySelector('form[role="search"]')?.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
    })
    expect(location.hash).toBe(before)
  })

  it('filters the directory from the file filter in the header', async () => {
    const el = mount()
    await loaded(el)
    type(el.querySelector<HTMLInputElement>('.top .file-filter') as HTMLInputElement, '但丁')
    expect([...el.querySelectorAll('.directory a')].map((a) => a.getAttribute('href'))).toEqual(['#/f/StoryData/S9999.json'])
    // the filter of the drawer shows the same text
    expect(el.querySelector<HTMLInputElement>('.dir .file-filter')?.value).toBe('但丁')
  })

  it('opens the drawer from the button of the home page', async () => {
    const el = mount()
    await loaded(el)
    act(() => el.querySelector<HTMLButtonElement>('.home .only-narrow')?.click())
    expect(el.querySelector('.dir')?.classList.contains('is-open')).toBe(true)
  })

  it('opens the family of the file in the address and marks the file', async () => {
    location.hash = '#/f/Skills_personality-99.json'
    const el = mount()
    await loaded(el)
    expect(el.querySelector('.directory a[aria-current="page"]')?.getAttribute('href')).toBe('#/f/Skills_personality-99.json')
    expect(el.querySelector('.home')).toBeNull()
  })

  it('follows the address when it changes', async () => {
    const el = mount()
    await loaded(el)
    expect(el.querySelector('.home')).not.toBeNull()
    act(() => {
      location.hash = '#/f/StoryData/S9999.json'
      window.dispatchEvent(new HashChangeEvent('hashchange'))
    })
    expect(el.querySelector('.home')).toBeNull()
    expect(el.querySelector('.directory a[aria-current="page"]')?.getAttribute('href')).toBe('#/f/StoryData/S9999.json')
  })

  it('shows the file of the address: a story as a script, anything else as a table', async () => {
    location.hash = '#/f/StoryData/S9999.json'
    const el = mount()
    await vi.waitFor(() => expect(el.querySelector('.file-view .script')).not.toBeNull())
    expect(el.querySelector('.file-title')?.textContent).toBe('StoryData/S9999.json')
    act(() => {
      location.hash = '#/f/Skills_personality-99.json'
      window.dispatchEvent(new HashChangeEvent('hashchange'))
    })
    await vi.waitFor(() => expect(el.querySelector('.file-view .tbl')).not.toBeNull())
    expect(el.querySelector('.file-view .script')).toBeNull()
  })

  it('marks the unit of the address in the file', async () => {
    location.hash = '#/f/Skills_personality-99.json?u=%2FdataList%2F1%2FlevelList%2F0%2Fname'
    const el = mount()
    await vi.waitFor(() => expect(el.querySelector('.file-view .is-target')).not.toBeNull())
    expect(el.querySelector('.is-target')?.getAttribute('data-ptr')).toBe('/dataList/1/levelList/0/name')
  })

  it('goes to the next file on ] and keeps the origin filter in the settings', async () => {
    location.hash = '#/f/Passives-empty.json'
    const el = mount()
    await vi.waitFor(() => expect(el.querySelector('.file-view .file-note')).not.toBeNull())
    act(() => {
      document.body.dispatchEvent(new KeyboardEvent('keydown', { key: ']', bubbles: true }))
    })
    await vi.waitFor(() => expect(el.querySelector('.file-title')?.textContent).toBe('Skills_personality-99.json'))
    act(() => el.querySelector<HTMLInputElement>('input[data-origin="claude"]')?.click())
    expect(JSON.parse(localStorage.getItem('lct-viewer-settings') ?? '{}').origins.claude).toBe(false)
    expect(el.querySelector('[data-ptr="/dataList/0/levelList/0/coinlist/0/coindescs/0/desc"]')).toBeNull()
  })

  it('opens the directory drawer from the button and closes it on Escape and on a new address', async () => {
    const el = mount()
    await loaded(el)
    const toggle = el.querySelector<HTMLButtonElement>('.dir-toggle')
    const drawer = el.querySelector('.dir')
    expect(toggle?.getAttribute('aria-expanded')).toBe('false')
    expect(drawer?.classList.contains('is-open')).toBe(false)
    act(() => toggle?.click())
    expect(toggle?.getAttribute('aria-expanded')).toBe('true')
    expect(drawer?.classList.contains('is-open')).toBe(true)
    act(() => {
      document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    })
    expect(drawer?.classList.contains('is-open')).toBe(false)
    act(() => toggle?.click())
    act(() => {
      location.hash = '#/f/StoryData/S9999.json'
      window.dispatchEvent(new HashChangeEvent('hashchange'))
    })
    expect(drawer?.classList.contains('is-open')).toBe(false)
  })
})
