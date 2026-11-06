import { act } from 'preact/test-utils'
import { render } from 'preact'
import { afterEach, describe, expect, it } from 'vitest'
import { filterFiles } from '../data'
import { fixture } from '../testdata'
import type { FileSummary, FileTag, Index, IndexGroup } from '../types'
import { Directory } from './Directory'

const index = fixture<Index>('index.json')

const containers: HTMLElement[] = []

function show(props: { index?: Index; query?: string; current?: string }, into?: HTMLElement): HTMLElement {
  const container = into ?? document.createElement('div')
  if (into === undefined) {
    document.body.append(container)
    containers.push(container)
  }
  act(() => render(<Directory index={props.index ?? index} query={props.query ?? ''} current={props.current} />, container))
  return container
}

afterEach(() => {
  for (const container of containers.splice(0)) {
    render(null, container)
    container.remove()
  }
})

function sum(path: string, family: string, label: string, speakers: string[], where: Partial<FileSummary> = {}): FileSummary {
  return { path, family, top: path.split('/')[0] ?? path, label, speakers, units: 1, origins: {}, missing: { en: 0, jp: 0 }, ...where }
}

const story: Index = {
  families: [
    { id: 'story-main', files: [0, 1], glossary: 'hard', nodes: ['StoryData/'] },
    { id: 'ally', files: [2], glossary: 'warn', nodes: [] },
  ],
  files: [
    sum('StoryData/S001A.json', 'story-main', '第一階段入口', ['維吉里烏斯']),
    sum('StoryData/S002.json', 'story-main', '', ['但丁', 'Faust']),
    sum('Skills_personality-01.json', 'ally', '', []),
  ],
  groups: [],
}

describe('filterFiles', () => {
  it('gives every file, in the order of the index, for an empty query', () => {
    expect(filterFiles(story, '')).toEqual([0, 1, 2])
    expect(filterFiles(story, '   ')).toEqual([0, 1, 2])
  })

  it('finds StoryData/S001A.json by a part of its name, in any case', () => {
    expect(filterFiles(story, 's001')).toEqual([0])
    expect(filterFiles(story, 'S001')).toEqual([0])
    expect(story.files[filterFiles(story, 's001')[0] ?? -1]?.path).toBe('StoryData/S001A.json')
  })

  it('finds a file by its label', () => {
    expect(filterFiles(story, '入口')).toEqual([0])
  })

  it('finds a file by the name of a speaker, in any case', () => {
    expect(filterFiles(story, '但丁')).toEqual([1])
    expect(filterFiles(story, 'faust')).toEqual([1])
  })

  it('finds nothing for what no file has', () => {
    expect(filterFiles(story, 'zzz')).toEqual([])
  })

  it('counts a part of the path in the middle of it', () => {
    expect(filterFiles(story, 'personality')).toEqual([2])
    expect(filterFiles(story, 'storydata/')).toEqual([0, 1])
  })
})

const tag = (kind: FileTag['kind'], value: string, label: string): FileTag => ({ kind, value, label })
const g = (kind: IndexGroup['kind'], node: string, label: string, files: number[], children: IndexGroup[] = []): IndexGroup => ({
  kind,
  node,
  label,
  files,
  children,
})

// Three levels: StoryData has two subs, each with chapters; Skills and Misc have the tags under the top.
const tree: Index = {
  families: [{ id: 'all', files: [0, 1, 2, 3, 4, 5, 6, 7, 8], glossary: 'hard', nodes: [] }],
  files: [
    sum('StoryData/S001A.json', 'all', '第一階段入口', ['維吉里烏斯'], { top: 'StoryData', sub: 'Main', tag: tag('chapter', '1', '第 1 章') }),
    sum('StoryData/S002.json', 'all', '', ['但丁', 'Faust'], { top: 'StoryData', sub: 'Main' }),
    sum('StoryData/S101.json', 'all', '', [], { top: 'StoryData', sub: 'Side', tag: tag('chapter', '10', '第 10 章') }),
    sum('StoryData/S102.json', 'all', '', [], { top: 'StoryData', sub: 'Side', tag: tag('chapter', '2', '第 2 章') }),
    sum('Skills_a.json', 'all', '', [], { top: 'Skills', tag: tag('character', 'faust', '浮士德') }),
    sum('Skills_b.json', 'all', '', [], { top: 'Skills', tag: tag('character', 'don', '堂吉訶德') }),
    sum('Skills_c.json', 'all', '', [], { top: 'Skills' }),
    sum('Event_x.json', 'all', '', [], { top: 'Misc', tag: tag('event', 'walpu6', 'walpu6') }),
    sum('Event_y.json', 'all', '', [], { top: 'Misc', tag: tag('event', 'bowl', 'bowl') }),
  ],
  groups: [
    g('top', 'StoryData', 'StoryData', [], [
      g('sub', 'Main', 'Main', [1], [g('chapter', '1', '第 1 章', [0])]),
      g('sub', 'Side', 'Side', [], [g('chapter', '2', '第 2 章', [3]), g('chapter', '10', '第 10 章', [2])]),
    ]),
    g('top', 'Skills', 'Skills', [6], [g('character', 'don', '堂吉訶德', [5]), g('character', 'faust', '浮士德', [4])]),
    g('top', 'Misc', 'Misc', [], [g('event', 'bowl', 'bowl', [8]), g('event', 'walpu6', 'walpu6', [7])]),
  ],
}

describe('Directory', () => {
  const buttons = (el: HTMLElement) => [...el.querySelectorAll<HTMLButtonElement>('button[aria-expanded]')]
  const rows = (el: HTMLElement) =>
    buttons(el).map((b) => `${b.querySelector('.dir-family-id')?.textContent}:${b.querySelector('.dir-count')?.textContent}`)
  const expanded = (el: HTMLElement) => buttons(el).filter((b) => b.getAttribute('aria-expanded') === 'true').map((b) => b.querySelector('.dir-family-id')?.textContent)
  const links = (el: HTMLElement) => [...el.querySelectorAll('a')].map((a) => a.textContent)
  const open = (el: HTMLElement, name: string) => {
    const b = buttons(el).find((x) => x.querySelector('.dir-family-id')?.textContent === name)
    if (b === undefined) throw new Error(`no group ${name}`)
    act(() => b.click())
  }
  const select = (el: HTMLElement, label: string): HTMLSelectElement => {
    const found = [...el.querySelectorAll('label')].find((l) => l.textContent?.startsWith(label))?.querySelector('select')
    if (!found) throw new Error(`no select ${label}`)
    return found
  }
  const choose = (el: HTMLElement, label: string, value: string) => {
    const s = select(el, label)
    act(() => {
      s.value = value
      s.dispatchEvent(new Event('change', { bubbles: true }))
    })
  }
  const optionValues = (s: HTMLSelectElement) => [...s.options].map((o) => o.value).filter((v) => v !== '')

  it('draws the groups of the index: the tops collapsed, with the file count of each, children included', () => {
    const el = show({ index: tree })
    expect(rows(el)).toEqual(['StoryData:4', 'Skills:3', 'Misc:2'])
    expect(expanded(el)).toEqual([])
    expect(el.querySelectorAll('a')).toHaveLength(0)
  })

  it('opens a top into its subs, a sub into its chapters, a chapter into its files', () => {
    const el = show({ index: tree })
    open(el, 'StoryData')
    expect(rows(el)).toEqual(['StoryData:4', 'Main:2', 'Side:2', 'Skills:3', 'Misc:2'])
    open(el, 'Main')
    expect(rows(el)).toEqual(['StoryData:4', 'Main:2', '第 1 章:1', 'Side:2', 'Skills:3', 'Misc:2'])
    expect(links(el)).toEqual(['StoryData/S002.json'])
    open(el, '第 1 章')
    expect(links(el)).toEqual(['StoryData/S001A.json第一階段入口', 'StoryData/S002.json'])
    expect(el.querySelector('a')?.getAttribute('href')).toBe('#/f/StoryData/S001A.json')
    open(el, 'StoryData')
    expect(rows(el)).toEqual(['StoryData:4', 'Skills:3', 'Misc:2'])
  })

  it('draws the third level apart from a sub: its own class', () => {
    const el = show({ index: tree })
    open(el, 'Skills')
    expect(el.querySelector('.dir-character')).not.toBeNull()
    expect(el.querySelector('.dir-top')).not.toBeNull()
    expect(el.querySelector('.dir-sub')).toBeNull()
  })

  it('opens every level above a hit, and shows no level without one', () => {
    const el = show({ index: tree, query: 's001a' })
    expect(rows(el)).toEqual(['StoryData:1', 'Main:1', '第 1 章:1'])
    expect(expanded(el)).toEqual(['StoryData', 'Main', '第 1 章'])
    expect(links(el)).toEqual(['StoryData/S001A.json第一階段入口'])
  })

  it('finds a file by the name of its speaker, and says which speaker it was', () => {
    const el = show({ index: tree, query: '但丁' })
    expect(rows(el)).toEqual(['StoryData:1', 'Main:1'])
    const a = el.querySelector('a')
    expect(a?.getAttribute('href')).toBe('#/f/StoryData/S002.json')
    expect(a?.textContent).toContain('但丁')
  })

  it('says so when no file matches', () => {
    const el = show({ index: tree, query: 'zzz' })
    expect(el.querySelectorAll('a')).toHaveLength(0)
    expect(el.textContent).toContain('沒有符合的檔案')
  })

  it('opens the levels of the current file, and marks it', () => {
    const el = show({ index: tree, current: 'StoryData/S001A.json' })
    expect(expanded(el)).toEqual(['StoryData', 'Main', '第 1 章'])
    expect(el.querySelector('a[aria-current="page"]')?.getAttribute('href')).toBe('#/f/StoryData/S001A.json')
  })

  it('opens the levels of another current file, and keeps the ones open that were', () => {
    const el = show({ index: tree, current: 'StoryData/S001A.json' })
    show({ index: tree, current: 'Skills_a.json' }, el)
    expect(expanded(el)).toEqual(['StoryData', 'Main', '第 1 章', 'Skills', '浮士德'])
    expect(el.querySelector('a[aria-current="page"]')?.getAttribute('href')).toBe('#/f/Skills_a.json')
  })

  it('opens the top of a current file that has no tag', () => {
    const el = show({ index: tree, current: 'Skills_c.json' })
    expect(expanded(el)).toEqual(['Skills'])
  })

  it('has the selects 章節, 角色 and 活動, whose options are the tags of the files', () => {
    const el = show({ index: tree })
    expect(optionValues(select(el, '章節'))).toEqual(['1', '2', '10'])
    expect(optionValues(select(el, '角色'))).toEqual(['don', 'faust'])
    expect([...select(el, '角色').options].map((o) => o.textContent)).toEqual(['全部', '堂吉訶德', '浮士德'])
    expect(optionValues(select(el, '活動'))).toEqual(['bowl', 'walpu6'])
  })

  it('shows only the files with the chosen chapter, down to the file', () => {
    const el = show({ index: tree })
    choose(el, '章節', '2')
    expect(rows(el)).toEqual(['StoryData:1', 'Side:1', '第 2 章:1'])
    expect(expanded(el)).toEqual(['StoryData', 'Side', '第 2 章'])
    expect(links(el)).toEqual(['StoryData/S102.json'])
  })

  it('shows only the files of the chosen character or event', () => {
    const el = show({ index: tree })
    choose(el, '角色', 'don')
    expect(links(el)).toEqual(['Skills_b.json'])
    choose(el, '活動', 'walpu6')
    expect(links(el)).toEqual(['Event_x.json'])
  })

  it('uses one select at a time: choosing one clears the other two', () => {
    const el = show({ index: tree })
    choose(el, '章節', '10')
    choose(el, '角色', 'faust')
    expect(select(el, '章節').value).toBe('')
    expect(select(el, '角色').value).toBe('faust')
    choose(el, '活動', 'bowl')
    expect(select(el, '角色').value).toBe('')
    expect(select(el, '活動').value).toBe('bowl')
    expect(links(el)).toEqual(['Event_y.json'])
    choose(el, '活動', '')
    expect(rows(el)).toEqual(['StoryData:4', 'Skills:3', 'Misc:2'])
  })

  it('combines a select with the file filter', () => {
    const el = show({ index: tree, query: 'skills_' })
    choose(el, '角色', 'faust')
    expect(links(el)).toEqual(['Skills_a.json'])
    const none = show({ index: tree, query: 'storydata' }, el)
    expect(none.textContent).toContain('沒有符合的檔案')
    expect(select(none, '角色')).toBeTruthy()
  })

  it('draws the fixture index, tops by file count then name', () => {
    const el = show({})
    expect(rows(el)).toEqual(['Passives:1', 'Skills:1', 'StoryData/:1'])
    open(el, 'Skills')
    expect(rows(el)).toEqual(['Passives:1', 'Skills:1', '浮士德:1', 'StoryData/:1'])
    open(el, 'Passives')
    expect(links(el)).toEqual(['Passives-empty.json'])
    expect(el.textContent).not.toContain('story-main')
  })

  it('shows the label of a file next to its path', () => {
    const el = show({ query: 's9999' })
    expect(el.textContent).toContain('StoryData/S9999.json')
    expect(el.textContent).toContain('分部入口附近')
  })
})
