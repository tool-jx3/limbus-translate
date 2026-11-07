import { render } from 'preact'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { clearDataCache } from '../data'
import { parseHash } from '../route'
import type { GoldData } from '../types'
import { fixtureFetch, runEffectsAtOnce } from '../testdata'
import { GoldView, goldLines, STORAGE_KEY } from './GoldView'

const row = (family: string, origin: string, sampled: number, judged: number, correct: number, wrong: number, unsure: number, stale = 0) => ({
  family,
  origin,
  sampled,
  judged,
  correct,
  wrong,
  unsure,
  accuracy: judged > 0 ? correct / judged : null,
  stale,
})

const gold: GoldData = {
  per_family: 2,
  missing: 1,
  units: [
    { key: 'a.json|1', path: 'a.json', ptr: '/1', family: 'story', origin: 'claude', kr: '안녕', jp: 'こんにちは', en: 'Hello', zh: '你好', verdict: '', note: '', stale: false },
    { key: 'a.json|2', path: 'a.json', ptr: '/2', family: 'story', origin: 'paratranz', kr: '잘가', zh: '再見', shipped: '掰掰', verdict: 'correct', note: '', stale: true },
    { key: 'b.json|1', path: 'b.json', ptr: '/1', family: 'skill', origin: 'claude', kr: '공격', zh: '攻擊', verdict: '', note: '', stale: false },
  ],
  report: {
    rows: [row('story', 'claude', 1, 0, 0, 0, 0), row('story', 'paratranz', 1, 1, 1, 0, 0, 1), row('skill', 'claude', 1, 0, 0, 0, 0)],
    families: [row('story', '', 2, 1, 1, 0, 0, 1), row('skill', '', 1, 0, 0, 0, 0)].map(({ origin: _o, ...r }) => r),
    total: row('', '', 3, 1, 1, 0, 0, 1),
  },
}

const containers: HTMLElement[] = []
let undoEffects = () => {}

beforeEach(() => {
  undoEffects = runEffectsAtOnce()
  clearDataCache()
  localStorage.clear()
  vi.stubGlobal('fetch', fixtureFetch({ 'gold.json': gold }))
})

afterEach(() => {
  undoEffects()
  for (const c of containers.splice(0)) {
    render(null, c)
    c.remove()
  }
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

async function mount(): Promise<HTMLElement> {
  const container = document.createElement('div')
  document.body.append(container)
  containers.push(container)
  render(<GoldView />, container)
  await vi.waitFor(() => expect(container.querySelector('.gold-unit, .notice')).not.toBeNull())
  return container
}

function choose(c: HTMLElement, key: string, verdict: string) {
  const unit = c.querySelector(`[data-key="${key}"]`) as HTMLElement
  const input = unit.querySelector(`input[value="${verdict}"]`) as HTMLInputElement
  input.click()
}

describe('#/gold route', () => {
  it('is read as the gold page', () => {
    expect(parseHash('#/gold')).toEqual({ kind: 'gold' })
  })
})

describe('GoldView', () => {
  it('lists the units by family with their texts, origin and stale mark', async () => {
    const c = await mount()
    const groups = [...c.querySelectorAll('.gold-family')]
    expect(groups.map((g) => g.getAttribute('data-family'))).toEqual(['story', 'skill'])
    const first = c.querySelector('[data-key="a.json|1"]') as HTMLElement
    for (const t of ['안녕', 'こんにちは', 'Hello', '你好', 'claude']) expect(first.textContent).toContain(t)
    expect((c.querySelector('[data-key="a.json|2"]') as HTMLElement).textContent).toContain('已過時')
    expect(first.textContent).not.toContain('已過時')
  })

  it('shows the report table with the accuracy of each family', async () => {
    const c = await mount()
    const table = c.querySelector('.gold-report') as HTMLElement
    expect(table.textContent).toContain('族群')
    expect(table.textContent).toContain('已判定')
    expect(table.textContent).toContain('正確率')
    expect(table.textContent).toContain('100%')
    expect(table.textContent).toContain('—')
  })

  it('shows the shipped text under TW and the count of missing units', async () => {
    const c = await mount()
    expect((c.querySelector('[data-key="a.json|2"] .gold-shipped') as HTMLElement).textContent).toBe('實際打包：掰掰')
    expect(c.querySelector('[data-key="a.json|1"] .gold-shipped')).toBeNull()
    expect(c.querySelector('.gold-missing')?.textContent).toContain('1')
  })

  it('shows the existing verdict as chosen', async () => {
    const c = await mount()
    const input = c.querySelector('[data-key="a.json|2"] input[value="correct"]') as HTMLInputElement
    expect(input.checked).toBe(true)
  })

  it('copies one line per judged unit: gold <key> <verdict> <note>', async () => {
    const write = vi.fn().mockResolvedValue(undefined)
    vi.stubGlobal('navigator', { clipboard: { writeText: write } })
    const c = await mount()
    choose(c, 'a.json|1', 'wrong')
    const note = c.querySelector('[data-key="a.json|1"] input[type="text"]') as HTMLInputElement
    note.value = '漏譯'
    note.dispatchEvent(new Event('input', { bubbles: true }))
    choose(c, 'b.json|1', 'unsure')
    ;(c.querySelector('.gold-copy') as HTMLButtonElement).click()
    await vi.waitFor(() => expect(write).toHaveBeenCalled())
    expect(write.mock.calls[0]![0]).toBe('gold a.json|1 wrong 漏譯\ngold b.json|1 unsure')
  })

  it('keeps the choices in localStorage', async () => {
    const c = await mount()
    choose(c, 'a.json|1', 'correct')
    await vi.waitFor(() => expect(localStorage.getItem(STORAGE_KEY)).toContain('a.json|1'))
    render(null, c)
    const c2 = await mount()
    expect((c2.querySelector('[data-key="a.json|1"] input[value="correct"]') as HTMLInputElement).checked).toBe(true)
  })

  it('still works when localStorage throws', async () => {
    const boom = () => {
      throw new Error('denied')
    }
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(boom)
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(boom)
    const write = vi.fn().mockResolvedValue(undefined)
    vi.stubGlobal('navigator', { clipboard: { writeText: write } })
    const c = await mount()
    choose(c, 'a.json|1', 'correct')
    ;(c.querySelector('.gold-copy') as HTMLButtonElement).click()
    await vi.waitFor(() => expect(write).toHaveBeenCalledWith('gold a.json|1 correct'))
  })

  it('falls back to a selected box when the clipboard says no', async () => {
    vi.stubGlobal('navigator', { clipboard: { writeText: vi.fn().mockRejectedValue(new Error('no')) } })
    const c = await mount()
    choose(c, 'a.json|1', 'correct')
    ;(c.querySelector('.gold-copy') as HTMLButtonElement).click()
    await vi.waitFor(() => expect(c.querySelector('.tv-copybox:not(.is-hidden)')).not.toBeNull())
    expect((c.querySelector('.tv-copybox') as HTMLTextAreaElement).value).toBe('gold a.json|1 correct')
    expect(c.querySelector('[role="status"]')?.textContent).toContain('Ctrl+C')
  })

  it('shows a notice when gold.json cannot be read', async () => {
    vi.stubGlobal('fetch', async () => new Response('x', { status: 404 }))
    const c = await mount()
    expect(c.querySelector('.notice')).not.toBeNull()
  })
})

describe('GoldView local choices', () => {
  const spaced = { key: 'c.json|/a b', path: 'c.json', ptr: '/a b', family: 'story', origin: 'claude', kr: '공백', zh: '空白', verdict: '' as const, note: '', stale: false }

  it('disables the note until a verdict is chosen, and says so', async () => {
    const c = await mount()
    const note = () => c.querySelector('[data-key="a.json|1"] input[type="text"]') as HTMLInputElement
    expect(note().disabled).toBe(true)
    expect(c.querySelector('[data-key="a.json|1"]')?.textContent).toContain('先選判定')
    choose(c, 'a.json|1', 'wrong')
    await vi.waitFor(() => expect(note().disabled).toBe(false))
  })

  it('does not offer a unit whose key has whitespace, and says it cannot be judged by copy and paste', async () => {
    vi.stubGlobal('fetch', fixtureFetch({ 'gold.json': { ...gold, units: [...gold.units, spaced] } }))
    const c = await mount()
    const unit = c.querySelector('[data-key="c.json|/a b"]') as HTMLElement
    expect(unit.textContent).toContain('空白')
    expect(unit.textContent).toContain('無法用複製貼上判定')
    expect(unit.querySelector('input')).toBeNull()
  })

  it('clears the choices of the page, in memory and in storage', async () => {
    const c = await mount()
    choose(c, 'a.json|1', 'wrong')
    await vi.waitFor(() => expect(localStorage.getItem(STORAGE_KEY)).toContain('a.json|1'))
    ;(c.querySelector('.gold-clear') as HTMLButtonElement).click()
    await vi.waitFor(() => expect((c.querySelector('[data-key="a.json|1"] input[value="wrong"]') as HTMLInputElement).checked).toBe(false))
    expect(localStorage.getItem(STORAGE_KEY) ?? '{}').not.toContain('a.json|1')
  })

  it('drops stored choices that the data already holds, keeps the others', async () => {
    localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({ 'b.json|1': { verdict: '', note: '' }, 'a.json|1': { verdict: 'wrong', note: 'x' } }),
    )
    const c = await mount()
    const stored = JSON.parse(localStorage.getItem(STORAGE_KEY) ?? '{}')
    expect(Object.keys(stored)).toEqual(['a.json|1'])
    expect((c.querySelector('[data-key="a.json|1"] input[value="wrong"]') as HTMLInputElement).checked).toBe(true)
  })

  it('keeps the choice of a stale unit even when it equals the data: sending it again clears the staleness', async () => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify({ 'a.json|2': { verdict: 'correct', note: '' } }))
    const c = await mount()
    expect(Object.keys(JSON.parse(localStorage.getItem(STORAGE_KEY) ?? '{}'))).toEqual(['a.json|2'])
    // and the page says that its line will be copied
    expect(c.querySelector('[data-key="a.json|2"] .gold-tocopy')?.textContent).toContain('待複製')
    expect(c.querySelector('[data-key="a.json|1"] .gold-tocopy')).toBeNull()
    expect(goldLines(gold.units, JSON.parse(localStorage.getItem(STORAGE_KEY) ?? '{}'))).toBe('gold a.json|2 correct')
  })

  it('marks a unit whose line will be copied once it is judged on the page', async () => {
    const c = await mount()
    expect(c.querySelector('[data-key="a.json|1"] .gold-tocopy')).toBeNull()
    choose(c, 'a.json|1', 'wrong')
    await vi.waitFor(() => expect(c.querySelector('[data-key="a.json|1"] .gold-tocopy')).not.toBeNull())
  })

  it('drops the marker when a non-stale unit is set back to its recorded verdict', async () => {
    const units = gold.units.map((u) => (u.key === 'b.json|1' ? { ...u, verdict: 'correct' as const } : u))
    vi.stubGlobal('fetch', fixtureFetch({ 'gold.json': { ...gold, units } }))
    const c = await mount()
    choose(c, 'b.json|1', 'wrong')
    await vi.waitFor(() => expect(c.querySelector('[data-key="b.json|1"] .gold-tocopy')).not.toBeNull())
    choose(c, 'b.json|1', 'correct')
    await vi.waitFor(() => expect(c.querySelector('[data-key="b.json|1"] .gold-tocopy')).toBeNull())
  })

  it('says that the missing units are not in the list below, and that the report still counts them', async () => {
    const c = await mount()
    const text = c.querySelector('.gold-missing')?.textContent ?? ''
    expect(text).toContain('樣本清單')
    expect(text).toContain('報告仍計入')
    expect(text).not.toContain('未列入下表')
  })

  it('counts missing from the report when it carries it', async () => {
    vi.stubGlobal('fetch', fixtureFetch({ 'gold.json': { ...gold, missing: undefined, report: { ...gold.report!, missing: 4 } } }))
    const c = await mount()
    expect(c.querySelector('.gold-missing')?.textContent).toContain('4')
  })
})

describe('goldLines', () => {
  it('leaves out a key with whitespace', () => {
    const u = { ...gold.units[0]!, key: 'x.json|/a b' }
    expect(goldLines([u], { [u.key]: { verdict: 'correct', note: '' } })).toBe('')
  })

  it('leaves out units without a verdict and keeps the sample order; no trailing space for an empty note', () => {
    const units = gold.units
    expect(goldLines(units, { 'b.json|1': { verdict: 'unsure', note: '' }, 'a.json|1': { verdict: '', note: 'x' } })).toBe('gold b.json|1 unsure')
  })
  it('turns line breaks of a note into spaces', () => {
    expect(goldLines(gold.units, { 'a.json|1': { verdict: 'wrong', note: '甲\n乙' } })).toBe('gold a.json|1 wrong 甲 乙')
  })
})
