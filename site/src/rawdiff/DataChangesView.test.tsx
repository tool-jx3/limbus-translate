import { render } from 'preact'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { clearDataCache } from '../data'
import { fixtureFetch, runEffectsAtOnce } from '../testdata'
import type { DataSnapshot } from '../types'
import { DataChangesView } from './DataChangesView'
import { ZERO_COUNTS } from './counts'

const zeroCounts = ZERO_COUNTS

const snapshots: DataSnapshot[] = [
  {
    tag: 'raw-20261112',
    prev_tag: 'raw-20261105',
    at: '2026-11-12T03:37:12Z',
    counts: { ...zeroCounts, files_added: 27, new_groups: 2, keys_added: 4, entries_added: 173, tokens_added: 29, files_removed: 1, tokens_removed: 3 },
  },
  { tag: 'raw-20261105', prev_tag: 'raw-20261029', at: '2026-11-05T03:20:00Z', counts: zeroCounts },
]

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

async function mount(): Promise<HTMLElement> {
  const container = document.createElement('div')
  document.body.append(container)
  containers.push(container)
  render(<DataChangesView />, container)
  await vi.waitFor(() => expect(container.querySelector('.uv-row, .dv-empty, .notice')).not.toBeNull())
  return container
}

describe('DataChangesView', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn(fixtureFetch({ 'rawdiff/index.json': { snapshots } })))
  })

  it('lists the snapshots newest first, each linking to its changes, with its day and the snapshot it is compared with', async () => {
    const el = await mount()
    const tags = [...el.querySelectorAll('.uv-row .uv-tag')]
    expect(tags.map((a) => a.textContent)).toEqual(['raw-20261112', 'raw-20261105'])
    expect(tags.map((a) => a.getAttribute('href'))).toEqual(['#/d/raw-20261112', '#/d/raw-20261105'])
    const row = el.querySelectorAll('.uv-row')[0]!
    expect(row.querySelector('time')?.getAttribute('datetime')).toBe('2026-11-12T03:37:12Z')
    expect(row.querySelector('.uv-prev')?.textContent).toBe('比對 raw-20261105')
  })

  it('shows the counts that are not zero: what is new, then what is gone', async () => {
    const row = (await mount()).querySelectorAll('.uv-row')[0]!
    expect([...row.querySelectorAll('.uv-count')].map((c) => `${c.className.replace('uv-count ', '')} ${c.textContent}`)).toEqual([
      'is-new 新種類 2',
      'is-new 新檔案 27',
      'is-new 新欄位 4',
      'is-new 新條目 173',
      'is-new 新標記 29',
      'is-deleted 移除檔案 1',
      'is-deleted 消失的標記 3',
    ])
  })

  it('says so when a snapshot changed nothing of the structure', async () => {
    const row = (await mount()).querySelectorAll('.uv-row')[1]!
    expect(row.querySelector('.uv-count')).toBeNull()
    expect(row.textContent).toContain('結構沒有變化')
  })

  it('says what the page is, and that the text changes are on the updates page', async () => {
    const el = await mount()
    expect(el.querySelector('h1')?.textContent).toBe('資料變化')
    expect(el.querySelector('.dv-intro a')?.getAttribute('href')).toBe('#/u')
  })

  it('says there is nothing to compare when the raw branch has one snapshot', async () => {
    vi.stubGlobal('fetch', vi.fn(fixtureFetch({ 'rawdiff/index.json': { snapshots: [] } })))
    const el = await mount()
    expect(el.querySelector('.dv-empty')?.textContent).toContain('還沒有可比對的快照')
    expect(el.querySelector('.uv-row')).toBeNull()
  })

  it('shows the same notice as the other pages when the load fails', async () => {
    vi.stubGlobal('fetch', vi.fn(fixtureFetch()))
    const el = await mount()
    expect(el.querySelector('.notice')?.textContent).toContain('資料已更新，請重新整理')
  })
})
