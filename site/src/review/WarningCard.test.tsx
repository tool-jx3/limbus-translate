import { render } from 'preact'
import { act } from 'preact/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { SuggestionItem, WarningItem } from '../types'
import { ShownText, type Shown } from './parts'
import { SuggestionCard } from './SuggestionCard'
import { WarningCard } from './WarningCard'

const containers: HTMLElement[] = []

function mount(node: preact.ComponentChildren, shown?: Shown): HTMLElement {
  const container = document.createElement('div')
  document.body.append(container)
  containers.push(container)
  render(shown ? <ShownText.Provider value={shown}>{node}</ShownText.Provider> : node, container)
  return container
}

afterEach(() => {
  for (const c of containers.splice(0)) {
    render(null, c)
    c.remove()
  }
  vi.unstubAllGlobals()
})

const group: WarningItem = {
  id: 'termgroup/aaaaaaaa/bbbbbbbb',
  code: 'term-missing',
  kr: '자판기',
  zh: '自動販賣機',
  message: 'the translation lacks the glossary rendering',
  keys: ['a.json|/1', 'a.json|/2'],
  families: ['story'],
  origins: ['claude'],
  pending: 0,
  term: { kr: '자판기', zh: '自動販賣機' },
  units: [
    { key: 'a.json|/1', id: 'warning/term-missing/11111111/22222222', kr: '자판기가 고장났다', zh: '販賣機壞了', note: '口語縮略\n第二行' },
    { key: 'a.json|/2', id: 'warning/term-missing/33333333/44444444', kr: '자판기 앞', zh: '販賣機前面' },
    {
      key: 'a.json|/3',
      id: 'warning/term-missing/55555555/66666666',
      kr: '자판기를 쓴다',
      zh: '用機器',
      accepted: { reason: '例外', date: '2026-11-03' },
    },
  ],
}

const unitRows = (el: HTMLElement) => el.querySelectorAll('.rv-tunit')

describe('WarningCard, a term-missing group', () => {
  it('titles the group with the entry and the number of units', () => {
    const el = mount(<WarningCard item={group} target={false} showAccepted={false} />)
    expect(el.querySelector('.rv-where')?.textContent).toBe('詞彙表：자판기 → 自動販賣機（2 處，1 處已確認）')
    expect(el.querySelector('.rv-head > .rv-id code')?.textContent).toBe('termgroup/aaaaaaaa/bbbbbbbb@2')
  })

  it('counts the units not confirmed in the title, and says nothing of the confirmed when there are none', () => {
    const open: WarningItem = { ...group, units: group.units!.slice(0, 2) }
    const el = mount(<WarningCard item={open} target={false} showAccepted={false} />)
    expect(el.querySelector('.rv-where')?.textContent).toBe('詞彙表：자판기 → 自動販賣機（2 處）')
    const done: WarningItem = { ...group, keys: [], units: group.units!.slice(2) }
    const el2 = mount(<WarningCard item={done} target={false} showAccepted={false} />)
    expect(el2.querySelector('.rv-where')?.textContent).toBe('詞彙表：자판기 → 自動販賣機（0 處，1 處已確認）')
  })

  it('shows the message of the group', () => {
    const el = mount(<WarningCard item={{ ...group, message: '這些譯文沒有用詞彙表的譯名「自動販賣機」（자판기）。' }} target={false} showAccepted={false} />)
    expect(el.querySelector('.rv-msg')?.textContent).toBe('這些譯文沒有用詞彙表的譯名「自動販賣機」（자판기）。')
  })

  it('copies the id of the whole group with the number of units still open', async () => {
    const writeText = vi.fn(async () => {})
    vi.stubGlobal('navigator', { clipboard: { writeText } })
    const el = mount(<WarningCard item={group} target={false} showAccepted={false} />)
    await act(async () => {
      el.querySelector<HTMLButtonElement>('.rv-head .rv-copy')!.click()
    })
    expect(writeText).toHaveBeenCalledWith('termgroup/aaaaaaaa/bbbbbbbb@2「자판기 → 自動販賣機」→ ')
    expect(el.querySelector('.rv-head .rv-id code')?.textContent).toBe('termgroup/aaaaaaaa/bbbbbbbb@2')
  })

  it('copies @0 and titles 0 處 for a group with every unit confirmed', async () => {
    const writeText = vi.fn(async () => {})
    vi.stubGlobal('navigator', { clipboard: { writeText } })
    const done: WarningItem = { ...group, keys: [], accepted: { reason: '例外', date: '2026-11-03' }, units: group.units!.slice(2) }
    const el = mount(<WarningCard item={done} target={false} showAccepted={false} />)
    expect(el.querySelector('.rv-where')?.textContent).toBe('詞彙表：자판기 → 自動販賣機（0 處，1 處已確認）')
    await act(async () => {
      el.querySelector<HTMLButtonElement>('.rv-head .rv-copy')!.click()
    })
    expect(writeText).toHaveBeenCalledWith('termgroup/aaaaaaaa/bbbbbbbb@0「자판기 → 自動販賣機」→ ')
  })

  it('takes the open and confirmed counts from the units alone, so they cannot disagree', async () => {
    const writeText = vi.fn(async () => {})
    vi.stubGlobal('navigator', { clipboard: { writeText } })
    const stale: WarningItem = { ...group, keys: ['a.json|/1', 'a.json|/2', 'a.json|/9', 'a.json|/8'] }
    const el = mount(<WarningCard item={stale} target={false} showAccepted={false} />)
    expect(el.querySelector('.rv-where')?.textContent).toBe('詞彙表：자판기 → 自動販賣機（2 處，1 處已確認）')
    await act(async () => {
      el.querySelector<HTMLButtonElement>('.rv-head .rv-copy')!.click()
    })
    expect(writeText).toHaveBeenCalledWith('termgroup/aaaaaaaa/bbbbbbbb@2「자판기 → 自動販賣機」→ ')
  })

  it('lists a unit per row with the original term marked, the translation and the note', () => {
    const el = mount(<WarningCard item={group} target={false} showAccepted={false} />)
    const rows = unitRows(el)
    expect(rows).toHaveLength(2)
    expect([...rows[0]!.querySelectorAll('dd mark')].map((m) => m.textContent)).toEqual(['자판기'])
    expect(rows[0]!.textContent).toContain('販賣機壞了')
    expect(rows[0]!.querySelector('.rv-note-text')?.textContent).toBe('口語縮略\n第二行')
    expect(rows[1]!.querySelector('.rv-note-text')).toBeNull()
    expect(rows[0]!.querySelector('a')?.getAttribute('href')).toContain('a.json')
  })

  it('has a copy button for the whole group and one for each unit, with the unit id', () => {
    const el = mount(<WarningCard item={group} target={false} showAccepted={false} />)
    const rows = unitRows(el)
    expect(rows[0]!.querySelector('.rv-id code')?.textContent).toBe('warning/term-missing/11111111/22222222')
    expect(rows[1]!.querySelector('.rv-copy')?.textContent).toBe('複製 ID')
    expect(el.querySelectorAll('.rv-copy')).toHaveLength(3)
  })

  it('copies the id of a unit with a piece of its translation', async () => {
    const writeText = vi.fn(async () => {})
    vi.stubGlobal('navigator', { clipboard: { writeText } })
    const el = mount(<WarningCard item={group} target={false} showAccepted={false} />)
    await act(async () => {
      unitRows(el)[1]!.querySelector<HTMLButtonElement>('.rv-copy')!.click()
    })
    expect(writeText).toHaveBeenCalledWith('warning/term-missing/33333333/44444444「販賣機前面」→ ')
  })

  it('hides the confirmed units until they are asked for, and fades them then', () => {
    const el = mount(<WarningCard item={group} target={false} showAccepted={true} />)
    const rows = unitRows(el)
    expect(rows).toHaveLength(3)
    expect(rows[2]!.classList.contains('is-confirmed')).toBe(true)
    expect(rows[2]!.textContent).toContain('已確認（2026-11-03）：例外')
  })

  it('follows the language switches and the raw tags', () => {
    const shown: Shown = { langs: { kr: false, jp: true, en: true, zh: true }, raw: false }
    let el = mount(<WarningCard item={group} target={false} showAccepted={false} />, shown)
    expect(unitRows(el)[0]!.textContent).not.toContain('자판기가')
    expect(unitRows(el)[0]!.textContent).toContain('販賣機壞了')
    const tagged: WarningItem = { ...group, units: [{ ...group.units![0]!, zh: '<b>販賣機</b>壞了' }] }
    el = mount(<WarningCard item={tagged} target={false} showAccepted={false} />, { langs: { kr: true, jp: true, en: true, zh: true }, raw: true })
    expect(unitRows(el)[0]!.textContent).toContain('<b>販賣機</b>壞了')
    el = mount(<WarningCard item={tagged} target={false} showAccepted={false} />)
    expect(unitRows(el)[0]!.textContent).not.toContain('<b>')
  })

  it('keeps the other warnings as they were', () => {
    const plain: WarningItem = { id: 'warning/kana/1/2', code: 'kana', kr: 'k', zh: 'の', message: 'm', keys: ['a|/1'], families: ['story'], origins: [], pending: 0 }
    const el = mount(<WarningCard item={plain} target={false} showAccepted={false} />)
    expect(el.querySelector('.rv-tunit')).toBeNull()
    expect(el.textContent).toContain('1 個單位用了同一句')
  })
})

describe('SuggestionCard, a provisional term', () => {
  const item: SuggestionItem = {
    id: 'term-new/우레',
    kind: 'term-new',
    title: '暫定詞：우레 → 雷鳴',
    pending: 0,
    term: { kr: '우레', zh: '雷鳴' },
    evidence: [
      { key: 'a.json|/1', kr: '우레', zh: '雷鳴', note: 'batch-7' },
      { key: 'b.json|/2', kr: '우레', zh: '雷聲', note: '衝突' },
      { key: 'c.json|/3', kr: '우레가 쳤다', zh: '雷鳴響起', note: '有用' },
      { key: 'd.json|/4', kr: '우레 소리', zh: '打雷聲', note: '未用' },
    ],
  }

  it('titles the card and shows the registration, the conflicts and the hits', () => {
    const el = mount(<SuggestionCard item={item} target={false} />)
    expect(el.querySelector('.rv-where')?.textContent).toBe('暫定詞：우레 → 雷鳴')
    const rows = el.querySelectorAll('.rv-tev')
    expect(rows).toHaveLength(4)
    expect(rows[0]!.textContent).toContain('登記的句子')
    expect(rows[0]!.textContent).toContain('batch-7')
    expect(rows[1]!.textContent).toContain('衝突的譯法')
    expect(rows[1]!.textContent).toContain('雷聲')
    expect(rows[2]!.querySelector('.rv-use')?.textContent).toBe('有用')
    expect(rows[3]!.querySelector('.rv-use')?.textContent).toBe('未用')
    expect(rows[3]!.querySelector('.rv-use')?.classList.contains('is-unused')).toBe(true)
  })

  it('marks the original term in the source and the rendering in the translation of a hit', () => {
    const el = mount(<SuggestionCard item={item} target={false} />)
    const hit = el.querySelectorAll('.rv-tev')[2]!
    expect([...hit.querySelectorAll('mark')].map((m) => m.textContent)).toEqual(['우레', '雷鳴'])
  })

  it('hides the source when KR is switched off', () => {
    const el = mount(<SuggestionCard item={item} target={false} />, { langs: { kr: false, jp: true, en: true, zh: true }, raw: false })
    expect(el.querySelectorAll('.rv-tev')[2]!.textContent).not.toContain('우레가')
    expect(el.querySelectorAll('.rv-tev')[2]!.textContent).toContain('雷鳴響起')
  })

  it('takes its marks from the term, not from the first evidence row', () => {
    const odd: SuggestionItem = { ...item, term: { kr: '소리', zh: '聲' } }
    const el = mount(<SuggestionCard item={odd} target={false} />)
    const hit = el.querySelectorAll('.rv-tev')[3]!
    expect([...hit.querySelectorAll('mark')].map((m) => m.textContent)).toEqual(['소리', '聲'])
  })

  it('marks nothing without a term', () => {
    const { term: _t, ...bare } = item
    const el = mount(<SuggestionCard item={bare} target={false} />)
    expect(el.querySelectorAll('mark')).toHaveLength(0)
  })

  it('shows a declined one like the other declined suggestions', () => {
    const el = mount(<SuggestionCard item={{ ...item, declined: { reason: '譯法還沒定', date: '2026-11-05' } }} target={false} />)
    expect(el.querySelector('article')?.classList.contains('is-confirmed')).toBe(true)
    expect(el.textContent).toContain('已否決（2026-11-05）：譯法還沒定')
    expect(el.textContent).not.toContain('已確認')
  })

  it('keeps 已確認 for a declined suggestion of another kind', () => {
    const other: SuggestionItem = { ...item, id: 'x/1', kind: 'inconsistent', declined: { reason: '可以', date: '2026-11-05' } }
    const el = mount(<SuggestionCard item={other} target={false} />)
    expect(el.textContent).toContain('已確認（2026-11-05）：可以')
  })
})
