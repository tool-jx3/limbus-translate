import { render } from 'preact'
import { act } from 'preact/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { clearDataCache } from '../data'
import { fileHref, termHref, type Route } from '../route'
import { runEffectsAtOnce } from '../testdata'
import type { BatchJSON, FileData, PreviewFile, PreviewShared } from '../types'
import { PreviewView } from './PreviewView'
import { fullText, rebuildBatch } from './rebuild'
import { sha8 } from './sha8'

const raw = import.meta.glob('../../fixtures/preview/**/*.json', { eager: true, query: '?raw', import: 'default' }) as Record<string, string>
const json = <T,>(rel: string): T => JSON.parse(raw[`../../fixtures/preview/${rel}`] as string) as T

const shared = json<PreviewShared>('preview.json')
const STORY = 'StoryData/S001A.json'
const OTHER = 'Other.json'

type Docs = Record<string, unknown>
const containers: HTMLElement[] = []
let undoEffects = () => {}
let docs: Docs = {}

/** ./data/ as the preview fixtures; `docs` replaces documents by their path under data/. */
function previewFetch() {
  return async (input: string) => {
    const path = input
      .replace(/^\.\/data\//, '')
      .split('/')
      .map(decodeURIComponent)
      .join('/')
    let doc: unknown
    if (path in docs) doc = docs[path]
    else if (path === 'preview.json') doc = shared
    else if (`../../fixtures/preview/${path}` in raw) doc = JSON.parse(raw[`../../fixtures/preview/${path}`] as string)
    else return new Response('not found', { status: 404 })
    return new Response(JSON.stringify(doc), { status: 200 })
  }
}

beforeEach(() => {
  undoEffects = runEffectsAtOnce()
  clearDataCache()
  docs = {}
  vi.stubGlobal('fetch', vi.fn(previewFetch()))
})

afterEach(() => {
  undoEffects()
  for (const c of containers.splice(0)) {
    render(null, c)
    c.remove()
  }
  vi.unstubAllGlobals()
})

const route = (path: string, ptr: string): Extract<Route, { kind: 'preview' }> => ({ kind: 'preview', path, ptr })

async function open(path: string, ptr: string): Promise<HTMLElement> {
  const el = document.createElement('div')
  document.body.append(el)
  containers.push(el)
  render(<PreviewView route={route(path, ptr)} />, el)
  await vi.waitFor(() => expect(el.querySelector('.pv-head, .notice, .pv-gone')).not.toBeNull())
  return el
}

/** The preview file of `path` with the unit at `index` changed by `edit`. */
function edit(path: string, index: number, change: (u: NonNullable<PreviewFile['units'][number]>) => void): void {
  const pf = json<PreviewFile>(`preview/${path}`)
  change(pf.units[index] as NonNullable<PreviewFile['units'][number]>)
  docs[`preview/${path}`] = pf
}

const ptrOf = (path: string, i: number) => json<FileData>(`files/${path}`).units[i]!.p
const text = (el: HTMLElement) => el.textContent ?? ''
const tab = (el: HTMLElement, name: string) => el.querySelector<HTMLElement>(`[role="tab"][data-tab="${name}"]`)!

describe('PreviewView: head and composition', () => {
  it('says which rules the simulation is under, with the first 7 characters of the commit', async () => {
    const el = await open(STORY, ptrOf(STORY, 5))
    expect(el.querySelector('.pv-head')?.textContent).toContain('依規則 0123456 模擬')
    expect(text(el)).not.toContain('0123456789')
  })

  it('links back to the unit in the file view', async () => {
    const ptr = ptrOf(STORY, 5)
    const el = await open(STORY, ptr)
    expect(el.querySelector<HTMLAnchorElement>('a.pv-back')?.getAttribute('href')).toBe(fileHref(STORY, ptr))
  })

  it('lists every guide path, the glossary with links, keywords, context, keep and the reason', async () => {
    const el = await open(STORY, ptrOf(STORY, 5))
    const s = text(el)
    for (const g of [0, 1, 2]) expect(s).toContain(shared.guides[g]!.path)
    // 위력 → 威力 links to its entry; the id is made of the sha8 of KR and of ZH
    const id = `glossary/${await sha8('위력')}/${await sha8('威力')}`
    await vi.waitFor(() => expect(el.querySelector(`a[href="${termHref(id)}"]`)).not.toBeNull())
    expect(el.querySelector(`a[href="${termHref(id)}"]`)?.textContent).toBe('위력 → 威力')
    expect(s).toContain('공격 → 攻擊')
    expect(s).toContain('kr-changed')
    expect(s).toContain('舊的第二行')
    expect(s).toContain('옛 대사 2')
    expect(s).toContain('語氣再正式一點')
    expect(s).toContain('<color=#f8c200>')
    // the context line: KR of unit 3 with the given ZH
    expect(s).toContain('第一行')
  })

  it('lists the keywords of the batch', async () => {
    const el = await open(STORY, ptrOf(STORY, 7))
    const s = text(el)
    expect(s).toContain('Bleed')
    expect(s).toContain('流血')
    expect(s).toContain('[Bleed]')
  })

  it('does not link a provisional term', async () => {
    const pf = json<PreviewFile>(`preview/${STORY}`)
    pf.terms[1] = { ...pf.terms[1]!, provisional: true }
    docs[`preview/${STORY}`] = pf
    const el = await open(STORY, ptrOf(STORY, 5))
    await vi.waitFor(() => expect(el.querySelectorAll('.pv-terms a')).toHaveLength(1))
    expect([...el.querySelectorAll('.pv-terms li')].some((li) => li.textContent?.includes('공격 → 攻擊') && li.querySelector('a') === null)).toBe(true)
  })

  it('keeps long texts in boxes of their own that scroll', async () => {
    const el = await open(STORY, ptrOf(STORY, 5))
    expect(el.querySelectorAll('pre.pv-box').length).toBeGreaterThan(0)
  })
})

describe('PreviewView: state, record and check', () => {
  it('says a needs_review unit is not sent now', async () => {
    const el = await open(OTHER, ptrOf(OTHER, 2))
    expect(text(el)).toContain('等人處理，現在不會發出')
    expect(text(el)).toContain('沒有譯文可檢查')
  })

  it('says to which batch a unit is leased, and still shows its batch', async () => {
    const el = await open(OTHER, ptrOf(OTHER, 3))
    expect(text(el)).toContain('已租給批次 20261101-0002')
    expect(text(el)).toContain('guides/base.md')
  })

  it('says a file is frozen', async () => {
    const el = await open('Frozen.json', ptrOf('Frozen.json', 0))
    expect(text(el)).toContain('不送翻：凍結檔案')
  })

  it('says there is no translation to check when there is no record', async () => {
    const el = await open(OTHER, ptrOf(OTHER, 1))
    expect(text(el)).toContain('沒有譯文可檢查')
  })

  it('shows an accepted record: origin, no model line, and the notes', async () => {
    const el = await open(OTHER, ptrOf(OTHER, 0))
    const s = text(el)
    expect(s).toContain('paratranz')
    expect(s).toContain('這是舊 KR 的譯文')
    expect(s).toContain('收下')
    expect(s).not.toContain('收下但有警告')
    expect(s).toContain('第二則 & 備註')
    expect(s).toContain('<b>粗體</b> & 威力')
  })

  it('shows the model, the batch and whether the rules are the present ones', async () => {
    edit(STORY, 1, (u) => {
      u.rec = { origin: 'claude', model: 'test-model', batch: '20261030-0001', rules: '0123456789abcdef', stale: false }
    })
    let el = await open(STORY, ptrOf(STORY, 1))
    expect(text(el)).toContain('test-model')
    expect(text(el)).toContain('20261030-0001')
    expect(text(el)).toContain('與現行規則相同')
    edit(STORY, 1, (u) => {
      u.rec = { origin: 'claude', rules: 'abc1234', stale: false }
    })
    clearDataCache()
    el = await open(STORY, ptrOf(STORY, 1))
    expect(text(el)).toContain('不是現行規則')
    expect(text(el)).not.toContain('與現行規則相同')
  })

  it('says nothing of the rules when the record has none', async () => {
    const el = await open(STORY, ptrOf(STORY, 9))
    expect(text(el)).not.toContain('現行規則')
  })

  it('shows a rejection and a warning with every issue code and message', async () => {
    edit(STORY, 9, (u) => {
      u.chk = {
        verdict: 'reject',
        zh: '威力',
        issues: [
          { code: 'tags-missing', severity: 'hard', class: 'structure', message: '缺少標籤 <b>' },
          { code: 'term-missing', severity: 'warn', class: 'glossary', message: '沒用譯名 위력' },
        ],
      }
    })
    let el = await open(STORY, ptrOf(STORY, 9))
    expect(text(el)).toContain('退回')
    expect(text(el)).toContain('tags-missing')
    expect(text(el)).toContain('缺少標籤 <b>')
    expect(text(el)).toContain('term-missing')
    expect(text(el)).toContain('沒用譯名 위력')
    edit(STORY, 9, (u) => {
      u.chk = { verdict: 'warn', zh: '威力', issues: [{ code: 'term-missing', severity: 'warn', class: 'glossary', message: '沒用譯名 위력' }] }
    })
    clearDataCache()
    el = await open(STORY, ptrOf(STORY, 9))
    expect(text(el)).toContain('收下但有警告')
    expect(text(el)).not.toContain('退回')
  })

  it('marks a fallback record as not a translation, and a human one as human', async () => {
    edit(STORY, 9, (u) => {
      u.rec = { origin: 'fallback', stale: false }
    })
    let el = await open(STORY, ptrOf(STORY, 9))
    expect(text(el)).toContain('替代文字（EN 或 KR），不是譯文：結果僅供參考')
    edit(STORY, 9, (u) => {
      u.rec = { origin: 'human', stale: false }
    })
    clearDataCache()
    el = await open(STORY, ptrOf(STORY, 9))
    expect(text(el)).toContain('人工譯文')
    expect(text(el)).not.toContain('替代文字')
  })
})

describe('PreviewView: full text', () => {
  const full = () => {
    const file = json<FileData>(`files/${STORY}`)
    const pf = json<PreviewFile>(`preview/${STORY}`)
    const batch: BatchJSON = rebuildBatch(STORY, file, pf, shared, 5)
    return fullText(shared, batch)
  }

  it('shows the full text on its own tab', async () => {
    const el = await open(STORY, ptrOf(STORY, 5))
    expect(el.querySelector('pre.pv-full')).toBeNull()
    act(() => tab(el, 'full').click())
    const pre = el.querySelector<HTMLElement>('pre.pv-full')!
    expect(pre.textContent).toBe(full())
  })

  it('copies the full text', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true })
    const el = await open(STORY, ptrOf(STORY, 5))
    act(() => tab(el, 'full').click())
    const button = [...el.querySelectorAll('button')].find((b) => b.textContent === '複製全文')!
    await act(async () => button.click())
    expect(writeText).toHaveBeenCalledWith(full())
    expect(el.querySelector('.pv-copyhint')).toBeNull()
    Reflect.deleteProperty(navigator, 'clipboard')
  })

  it('shows a selected box and a hint where the clipboard says no', async () => {
    Object.defineProperty(navigator, 'clipboard', { value: { writeText: vi.fn().mockRejectedValue(new Error('no')) }, configurable: true })
    const el = await open(STORY, ptrOf(STORY, 5))
    act(() => tab(el, 'full').click())
    const button = [...el.querySelectorAll('button')].find((b) => b.textContent === '複製全文')!
    await act(async () => button.click())
    await vi.waitFor(() => expect(el.querySelector('.tv-copyhint')).not.toBeNull())
    expect(el.querySelector('.tv-copyhint')?.getAttribute('role')).toBe('status')
    const box = el.querySelector<HTMLTextAreaElement>('textarea.tv-copybox')!
    expect(box.value).toBe(full())
    expect(box.classList.contains('is-hidden')).toBe(false)
    Reflect.deleteProperty(navigator, 'clipboard')
  })
})

describe('PreviewView: units that cannot be previewed', () => {
  it('explains a copy unit instead of failing', async () => {
    const el = await open(STORY, ptrOf(STORY, 13))
    expect(text(el)).toContain('copy：識別碼或各語言相同，不送翻')
    expect(el.querySelector('[role="tab"]')).toBeNull()
    expect(el.querySelector('a.pv-back')).not.toBeNull()
  })

  it('explains a pointer the file does not have', async () => {
    const el = await open(STORY, '/dataList/99/content')
    expect(text(el)).toContain('找不到')
    expect(text(el)).toContain('/dataList/99/content')
    expect(el.querySelector('a.pv-back')).not.toBeNull()
  })

  it('shows the notice of a failed load', async () => {
    const el = await open('Nope.json', '/x')
    expect(el.querySelector('.notice')).not.toBeNull()
  })
})
