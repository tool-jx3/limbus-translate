import { render } from 'preact'
import { act } from 'preact/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { clearDataCache } from '../data'
import { fixture, fixtureFetch, runEffectsAtOnce } from '../testdata'
import { InProcessWorker } from '../search/testworker'
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
    vi.stubGlobal('fetch', vi.fn(fixtureFetch({ 'meta.json': { ...fixture<Meta>('meta.json'), format: 3 } })))
    const el = mount()
    await loaded(el)
    expect(el.querySelector('.notice')?.textContent).toContain('資料已更新，請重新整理')
    expect(el.querySelector('.directory')).toBeNull()
    expect(el.querySelector('footer')?.textContent).toContain('Project Moon')
  })

  it('header shows the review count', async () => {
    const el = mount()
    await loaded(el)
    const { suggestions, warnings, notes, todo } = fixture<Meta>('meta.json').review
    const total = suggestions + warnings + notes + todo
    expect(total).toBeGreaterThan(0)
    const link = el.querySelector<HTMLAnchorElement>('.top a.review-link')
    expect(link?.getAttribute('href')).toBe('#/r/suggestions')
    expect(link?.textContent).toContain('審核')
    expect(link?.querySelector('.review-count')?.textContent).toBe(String(total))
  })

  it('hands the review route to a placeholder until the review page exists', async () => {
    location.hash = '#/r/notes'
    const el = mount()
    await loaded(el)
    expect(el.querySelector('.review-placeholder')?.getAttribute('data-tab')).toBe('notes')
    expect(el.querySelector('.home')).toBeNull()
  })

  it('shows no number beside the review link when nothing is left to review', async () => {
    const meta = fixture<Meta>('meta.json')
    vi.stubGlobal('fetch', vi.fn(fixtureFetch({ 'meta.json': { ...meta, review: { suggestions: 0, warnings: 0, notes: 0, todo: 0 } } })))
    const el = mount()
    await loaded(el)
    expect(el.querySelector('.top a.review-link')?.textContent).toBe('審核')
    expect(el.querySelector('.review-count')).toBeNull()
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

  describe('the full-text search of the address', () => {
    const made: { url: string | URL; options?: WorkerOptions }[] = []

    beforeEach(() => {
      made.length = 0
      // A worker that runs the engine in the page; the page asks for the data at an absolute URL.
      class StandIn extends InProcessWorker {
        constructor(url: string | URL, options?: WorkerOptions) {
          super()
          made.push({ url, options })
        }
      }
      vi.stubGlobal('Worker', StandIn)
      const base = new URL('data/', document.baseURI).href
      const fixtures = fixtureFetch()
      vi.stubGlobal('fetch', vi.fn((url: string) => fixtures(url.startsWith(base) ? `./data/${url.slice(base.length)}` : url)))
    })

    it('shows the hits, found by a module worker made from worker.ts', async () => {
      location.hash = '#/s?q=%E8%AD%A6%E5%91%8A&lang=zh'
      const el = mount()
      await vi.waitFor(() => expect(el.querySelector('.search-view .hit-group')).not.toBeNull())
      expect(el.querySelector('.search-status')?.textContent).toContain('共 1 筆')
      expect(el.querySelector('.search-view mark')?.textContent).toBe('警告')
      expect(el.querySelector('.hit-link')?.getAttribute('href')).toBe('#/f/StoryData/S9999.json?u=%2FdataList%2F3%2Fcontent')
      expect(made).toHaveLength(1)
      expect(made[0]?.options).toEqual({ type: 'module' })
      expect(String(made[0]?.url)).toMatch(/worker\.ts/)
    })

    it('keeps the language and the pattern when the box in the header searches again, and lets the reader turn the pattern off', async () => {
      location.hash = '#/s?q=%5B%5Cw%2B%5D&lang=en&re=1'
      const el = mount()
      await vi.waitFor(() => expect(el.querySelector('.search-view .hit-group')).not.toBeNull())
      type(el.querySelector<HTMLInputElement>('form[role="search"] input') as HTMLInputElement, 'Burn')
      act(() => {
        el.querySelector('form[role="search"]')?.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
      })
      expect(location.hash).toBe('#/s?q=Burn&lang=en&re=1')
      act(() => el.querySelector<HTMLInputElement>('.search-view input[name="re"]')?.click())
      expect(location.hash).toBe('#/s?q=%5B%5Cw%2B%5D&lang=en')
    })

    describe('opening a file from the page of the results', () => {
      const scrollIntoView = vi.fn()

      beforeEach(() => {
        scrollIntoView.mockReset()
        Object.defineProperty(Element.prototype, 'scrollIntoView', { value: scrollIntoView, configurable: true, writable: true })
        document.documentElement.scrollTop = 0
      })

      afterEach(() => {
        Reflect.deleteProperty(Element.prototype, 'scrollIntoView')
        document.documentElement.scrollTop = 0
      })

      /** The page of the results for 警告, scrolled down as a reader leaves it, and the file of its group already loaded. */
      async function scrolledResults(): Promise<HTMLElement> {
        location.hash = '#/s?q=%E8%AD%A6%E5%91%8A&lang=zh'
        const el = mount()
        await vi.waitFor(() => expect(el.querySelector('.search-view .hit-group')).not.toBeNull())
        await vi.waitFor(() => expect(el.querySelector('.hit-line.hit-kr .hit-text')?.textContent).not.toBe('…'))
        document.documentElement.scrollTop = 1170
        return el
      }

      function go(hash: string): void {
        act(() => {
          location.hash = hash
          window.dispatchEvent(new HashChangeEvent('hashchange'))
        })
      }

      it('shows the file from its top, not at the place where the results were scrolled to', async () => {
        const el = await scrolledResults()
        go(el.querySelector('.group-path')?.getAttribute('href') ?? '')
        await vi.waitFor(() => expect(el.querySelector('.file-view .script')).not.toBeNull())
        expect(document.documentElement.scrollTop).toBe(0)
      })

      it('shows a file from the top when the link names no unit in it, as the links of the directory do', async () => {
        const el = await scrolledResults()
        go('#/f/Skills_personality-99.json')
        await vi.waitFor(() => expect(el.querySelector('.file-view .tbl')).not.toBeNull())
        expect(document.documentElement.scrollTop).toBe(0)
      })

      it('goes to the unit of the hit that was clicked, and does not scroll to the top', async () => {
        const el = await scrolledResults()
        const link = el.querySelector('.hit-link')
        expect(link?.getAttribute('href')).toBe('#/f/StoryData/S9999.json?u=%2FdataList%2F3%2Fcontent')
        go(link?.getAttribute('href') ?? '')
        await vi.waitFor(() => expect(el.querySelector('.file-view .is-target')).not.toBeNull())
        await vi.waitFor(() => expect(scrollIntoView).toHaveBeenCalled())
        expect(scrollIntoView.mock.contexts[0]).toBe(el.querySelector('.is-target'))
        expect(document.documentElement.scrollTop).toBe(1170)
      })
    })
  })
})
