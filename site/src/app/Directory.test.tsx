import { act } from 'preact/test-utils'
import { render } from 'preact'
import { afterEach, describe, expect, it } from 'vitest'
import { filterFiles } from '../data'
import { fixture } from '../testdata'
import type { FileSummary, Index } from '../types'
import { Directory } from './Directory'

const index = fixture<Index>('index.json')

const containers: HTMLElement[] = []

function show(props: { index?: Index; query?: string; current?: string }): HTMLElement {
  const container = document.createElement('div')
  document.body.append(container)
  containers.push(container)
  render(<Directory index={props.index ?? index} query={props.query ?? ''} current={props.current} />, container)
  return container
}

afterEach(() => {
  for (const container of containers.splice(0)) {
    render(null, container)
    container.remove()
  }
})

function sum(path: string, family: string, label: string, speakers: string[]): FileSummary {
  return { path, family, label, speakers, units: 1, origins: {}, missing: { en: 0, jp: 0 } }
}

const story: Index = {
  families: [
    { id: 'story-main', files: [0, 1] },
    { id: 'ally', files: [2] },
  ],
  files: [
    sum('StoryData/S001A.json', 'story-main', '第一階段入口', ['維吉里烏斯']),
    sum('StoryData/S002.json', 'story-main', '', ['但丁', 'Faust']),
    sum('Skills_personality-01.json', 'ally', '', []),
  ],
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

describe('Directory', () => {
  const buttons = (el: HTMLElement) => [...el.querySelectorAll<HTMLButtonElement>('button[aria-expanded]')]
  const families = (el: HTMLElement) =>
    buttons(el).map((b) => `${b.querySelector('.dir-family-id')?.textContent}:${b.querySelector('.dir-count')?.textContent}`)
  const links = (el: HTMLElement) => [...el.querySelectorAll('a')].map((a) => a.textContent)

  it('lists the families of the index, collapsed, with their file counts', () => {
    const el = show({})
    expect(families(el)).toEqual(['story-main:1', 'ally:2'])
    expect(buttons(el).every((b) => b.getAttribute('aria-expanded') === 'false')).toBe(true)
    expect(el.querySelectorAll('a')).toHaveLength(0)
  })

  it('shows the files of a family when it is opened, and hides them when it is closed again', () => {
    const el = show({})
    const ally = buttons(el)[1]
    act(() => ally?.click())
    expect(ally?.getAttribute('aria-expanded')).toBe('true')
    expect(el.querySelectorAll('a')).toHaveLength(2)
    expect([...el.querySelectorAll('a')].map((a) => a.getAttribute('href'))).toEqual([
      '#/f/Passives-empty.json',
      '#/f/Skills_personality-99.json',
    ])
    act(() => ally?.click())
    expect(el.querySelectorAll('a')).toHaveLength(0)
  })

  it('opens the family of the file in view, and marks that file', () => {
    const el = show({ current: 'Skills_personality-99.json' })
    expect(buttons(el).map((b) => b.getAttribute('aria-expanded'))).toEqual(['false', 'true'])
    const current = el.querySelector('a[aria-current="page"]')
    expect(current?.getAttribute('href')).toBe('#/f/Skills_personality-99.json')
  })

  it('shows the label of a file next to its path', () => {
    const el = show({ query: 's9999' })
    expect(el.textContent).toContain('StoryData/S9999.json')
    expect(el.textContent).toContain('分部入口附近')
  })

  it('shows only the families and the files that match a filter, all open', () => {
    const el = show({ index: story, query: 's001' })
    expect(families(el)).toEqual(['story-main:1'])
    expect(links(el)).toEqual(['StoryData/S001A.json第一階段入口'])
  })

  it('finds a file by the name of its speaker, and says which speaker it was', () => {
    const el = show({ index: story, query: '但丁' })
    expect(el.querySelectorAll('a')).toHaveLength(1)
    const a = el.querySelector('a')
    expect(a?.getAttribute('href')).toBe('#/f/StoryData/S002.json')
    expect(a?.textContent).toContain('但丁')
  })

  it('says so when no file matches', () => {
    const el = show({ index: story, query: 'zzz' })
    expect(el.querySelectorAll('a')).toHaveLength(0)
    expect(el.textContent).toContain('沒有符合的檔案')
  })

  it('has the fixture index by family: story-main, then ally', () => {
    const el = show({ query: '.json' })
    const text = el.textContent ?? ''
    expect(text.indexOf('StoryData/S9999.json')).toBeGreaterThan(text.indexOf('story-main'))
    expect(text.indexOf('Passives-empty.json')).toBeGreaterThan(text.indexOf('ally'))
    expect(text.indexOf('ally')).toBeGreaterThan(text.indexOf('StoryData/S9999.json'))
  })
})
