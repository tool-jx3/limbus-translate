import { render } from 'preact'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { clearDataCache } from '../data'
import { fixtureFetch, runEffectsAtOnce } from '../testdata'
import type { UpdateCounts, UpdateRelease } from '../types'
import { UpdatesView } from './UpdatesView'

const zero: UpdateCounts = { new: 0, kr_changed: 0, deleted: 0, zh_changed: 0 }
const releases: UpdateRelease[] = [
  {
    tag: 'zh-TW-20261105.1',
    released_at: '2026-11-05T08:30:00Z',
    prev_tag: 'zh-TW-20261029.7',
    raw_tag: 'raw-20261105',
    prev_raw_tag: 'raw-20261029',
    kind: 'game',
    counts: { new: 12, kr_changed: 3, deleted: 0, zh_changed: 5 },
    copy_counts: { new: 40, kr_changed: 9, deleted: 1, zh_changed: 9 },
    other_files: 1,
    files: 4,
    recorded: true,
  },
  {
    tag: 'zh-TW-20261029.7',
    released_at: '2026-10-29T08:30:00Z',
    prev_tag: 'zh-TW-20261029.6',
    raw_tag: 'raw-20261029',
    prev_raw_tag: 'raw-20261029',
    kind: 'rebuild',
    counts: { ...zero, zh_changed: 2 },
    copy_counts: zero,
    other_files: 0,
    files: 1,
    recorded: true,
  },
  {
    tag: 'zh-TW-20261001.1',
    released_at: '2026-10-01T08:30:00Z',
    raw_tag: 'raw-20261001',
    kind: 'first',
    counts: zero,
    copy_counts: zero,
    other_files: 0,
    files: 0,
    recorded: false,
  },
]

// A Release whose record is not made yet.
const unrecorded: UpdateRelease = {
  tag: 'zh-TW-20261030.1',
  released_at: '2026-10-30T08:30:00Z',
  raw_tag: '',
  kind: 'rebuild',
  counts: zero,
  copy_counts: zero,
  other_files: 0,
  files: 0,
  recorded: false,
}

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
  render(<UpdatesView />, container)
  await vi.waitFor(() => expect(container.querySelector('.uv-row, .notice')).not.toBeNull())
  return container
}

describe('UpdatesView', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn(fixtureFetch({ 'updates/index.json': { releases } })))
  })

  it('lists the releases newest first, each tag with a record linking to its update', async () => {
    const el = await mount()
    const tags = [...el.querySelectorAll('.uv-row .uv-tag')]
    expect(tags.map((a) => a.textContent)).toEqual(['zh-TW-20261105.1', 'zh-TW-20261029.7', 'zh-TW-20261001.1'])
    expect(tags.map((a) => a.getAttribute('href'))).toEqual(['#/u/zh-TW-20261105.1', '#/u/zh-TW-20261029.7', null])
  })

  it('gives neither the first release nor one without a record a link', async () => {
    vi.stubGlobal('fetch', vi.fn(fixtureFetch({ 'updates/index.json': { releases: [releases[0], unrecorded, releases[2]] } })))
    const rows = (await mount()).querySelectorAll('.uv-row')
    expect([...rows].map((r) => r.querySelector('a') !== null)).toEqual([true, false, false])
    const row = rows[1]!
    expect(row.querySelector('.uv-tag')?.textContent).toBe('zh-TW-20261030.1')
    expect(row.textContent).toContain('還沒有比對紀錄')
    expect(row.querySelector('.uv-count')).toBeNull()
    expect(row.querySelector('.uv-origin')).toBeNull()
  })

  it('shows a game update with its raw tag, the date, the previous tag and the non-zero counts', async () => {
    const row = (await mount()).querySelectorAll('.uv-row')[0]!
    expect(row.textContent).toContain('2026-11-05')
    expect(row.textContent).toContain('比對 zh-TW-20261029.7')
    expect(row.querySelector('.uv-origin')?.textContent).toBe('遊戲更新 raw-20261105')
    expect([...row.querySelectorAll('.uv-count')].map((c) => c.textContent)).toEqual(['新增 12', 'KR 修改 3', '只有譯文改變 5'])
    expect(row.querySelector('.uv-other')?.textContent).toBe('其他檔案 1')
  })

  it('shows a rebuild as 重建 and hides the other-files chip when there are none', async () => {
    const row = (await mount()).querySelectorAll('.uv-row')[1]!
    expect(row.querySelector('.uv-origin')?.textContent).toBe('重建')
    expect([...row.querySelectorAll('.uv-count')].map((c) => c.textContent)).toEqual(['只有譯文改變 2'])
    expect(row.querySelector('.uv-other')).toBeNull()
  })

  it('shows the first release with no previous tag and no counts', async () => {
    const row = (await mount()).querySelectorAll('.uv-row')[2]!
    expect(row.textContent).toContain('第一版，沒有可比對的上一版')
    expect(row.textContent).not.toContain('比對 ')
    expect(row.querySelector('.uv-count')).toBeNull()
    expect(row.querySelector('.uv-origin')).toBeNull()
  })

  it('shows the same notice as the other pages when the load fails', async () => {
    vi.stubGlobal('fetch', vi.fn(fixtureFetch()))
    const el = await mount()
    expect(el.querySelector('.notice')?.textContent).toContain('資料已更新，請重新整理')
  })
})
