import { render } from 'preact'
import { afterEach, describe, expect, it } from 'vitest'
import { Header } from './Header'
import { defaultSettings } from '../settings'
import type { Route } from '../route'

const containers: HTMLElement[] = []
afterEach(() => {
  for (const c of containers.splice(0)) {
    render(null, c)
    c.remove()
  }
})

function mount(route: Route): HTMLElement {
  const c = document.createElement('div')
  document.body.append(c)
  containers.push(c)
  render(
    <Header route={route} settings={defaultSettings()} onSettings={() => {}} fileQuery="" onFileQuery={() => {}} drawerOpen={false} onToggleDrawer={() => {}} />,
    c,
  )
  return c
}

describe('Header 基準 link', () => {
  it('points at #/gold and is the current page there', () => {
    const link = mount({ kind: 'home' }).querySelector('a.gold-link') as HTMLAnchorElement
    expect(link.textContent).toBe('基準')
    expect(link.getAttribute('href')).toBe('#/gold')
    expect(link.hasAttribute('aria-current')).toBe(false)
    expect(mount({ kind: 'gold' }).querySelector('a.gold-link')?.getAttribute('aria-current')).toBe('page')
  })
})

describe('Header 詞彙 link', () => {
  it('is the current page on the existing names page', () => {
    expect(mount({ kind: 'names', q: '', only: '', sort: 'units' }).querySelector('a.glossary-link')?.getAttribute('aria-current')).toBe('page')
    expect(mount({ kind: 'home' }).querySelector('a.glossary-link')?.hasAttribute('aria-current')).toBe(false)
  })
})

describe('Header 資料變化 link', () => {
  it('points at #/d and is the current page on the list and on a snapshot', () => {
    const link = mount({ kind: 'home' }).querySelector('a.data-link') as HTMLAnchorElement
    expect(link.textContent).toBe('資料變化')
    expect(link.getAttribute('href')).toBe('#/d')
    expect(link.hasAttribute('aria-current')).toBe(false)
    expect(mount({ kind: 'datas' }).querySelector('a.data-link')?.getAttribute('aria-current')).toBe('page')
    expect(mount({ kind: 'data', tag: 'raw-20261112' }).querySelector('a.data-link')?.getAttribute('aria-current')).toBe('page')
  })
})
