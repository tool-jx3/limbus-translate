import { render } from 'preact'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { clearDataCache } from '../data'
import { fixture, fixtureFetch, runEffectsAtOnce } from '../testdata'
import type { DataDiff, Index } from '../types'
import { DataChangeView } from './DataChangeView'
import { ZERO_COUNTS } from './counts'

const zeroCounts = ZERO_COUNTS

const tag = 'raw-20261112'

const empty: DataDiff = {
  lang: 'kr',
  from: 'raw-20261105',
  to: tag,
  counts: zeroCounts,
  new_groups: [],
  gone_groups: [],
  files_added: [],
  files_removed: [],
  keys: [],
  entries: [],
  tokens: [],
}

// The index of the fixtures has Skills_personality-02.json and StoryData/S9999.json.
const diff: DataDiff = {
  ...empty,
  counts: { ...zeroCounts, files_added: 3, files_removed: 1, new_groups: 1, gone_groups: 1, keys_added: 2, keys_removed: 1, entries_added: 3, entries_removed: 1, tokens_added: 6, tokens_removed: 1 },
  new_groups: [
    {
      group: 'SynergyKeyword',
      files: 1,
      keys: [
        { path: 'dataList[].id', kinds: ['string'], count: 1, file: 'SynergyKeyword.json', pointer: '/dataList/0/id' },
        { path: 'dataList[].name', kinds: ['string'], count: 1, file: 'SynergyKeyword.json', pointer: '/dataList/0/name' },
      ],
    },
  ],
  gone_groups: [{ group: 'OldThing', files: 2, keys: [{ path: 'dataList[].desc', kinds: ['string'], count: 5, file: 'OldThing-1.json', pointer: '/dataList/0/desc' }] }],
  files_added: [
    { file: 'StoryData/S9999.json', group: 'StoryData', family: 'story-main', entries: 131 },
    { file: 'SynergyKeyword.json', group: 'SynergyKeyword', family: 'keywords', entries: 1 },
    { file: 'Mystery.json', group: 'Mystery', family: 'unsorted', entries: 4 },
  ],
  files_removed: [{ file: 'OldThing-1.json', group: 'OldThing', family: 'ui', entries: 5 }],
  keys: [
    {
      group: 'Bufs',
      added: [{ path: 'dataList[].excludeAutoLink', kinds: ['bool'], count: 14, file: 'Bufs-walpu10.json', pointer: '/dataList/34/excludeAutoLink' }],
      removed: [{ path: 'dataList[].undefined', kinds: ['null', 'string'], count: 2, file: 'Bufs.json', pointer: '/dataList/0/undefined' }],
    },
    {
      group: 'Skills',
      added: [
        {
          path: 'dataList[].levelList[].flavor',
          kinds: ['string'],
          count: 3,
          file: 'Skills_personality-02.json',
          pointer: '/dataList/0/levelList/0/flavor',
        },
      ],
      removed: [],
    },
  ],
  entries: [
    { file: 'Skills_personality-02.json', group: 'Skills', family: 'ally', before: 53, after: 55, added: ['1021101', '1021102', '1021103'], removed: ['1020999'] },
    { file: 'NoID.json', group: 'NoID', before: 1, after: 2, added: [], removed: [] },
  ],
  tokens: [
    { kind: 'bracket', name: 'ComboSkill', change: 'added', count: 41, file: 'Skills_personality-02.json', pointer: '/dataList/1/desc', defined_in: ['BattleKeywords', 'Bufs'] },
    { kind: 'bracket', name: 'OnBattleEnd', change: 'added', count: 6, file: 'Skills_personality-03.json', pointer: '/dataList/57/levelList/0/desc', defined_in: ['SkillTag'] },
    { kind: 'bracket', name: 'Conbustion', change: 'added', count: 2, file: 'EGOgift.json', pointer: '/dataList/3/desc', defined_in: [] },
    { kind: 'bracket', name: 'OldMark', change: 'removed', count: 1, file: 'Bufs.json', pointer: '/dataList/9/desc', defined_in: ['Bufs'] },
    { kind: 'placeholder', name: 'waveNum', change: 'added', count: 1, file: 'BattleUIText.json', pointer: '/dataList/7/content', defined_in: [] },
    { kind: 'style', name: 'number_highlight', change: 'added', count: 1, file: 'MainUIText.json', pointer: '/dataList/2/content', defined_in: [] },
    { kind: 'tag', name: 'font', change: 'added', count: 12, file: 'StoryData/S9999.json', pointer: '/dataList/4/content', defined_in: [] },
  ],
}

// The fallback family is the one without nodes.
const base = fixture<Index>('index.json')
const index: Index = { ...base, families: [...base.families, { id: 'unsorted', files: [], glossary: 'warn', nodes: [] }] }

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

// The JP pack of the same snapshot: one file of its own, and without two changes of KR.
const jp: DataDiff = {
  ...empty,
  lang: 'jp',
  counts: { ...zeroCounts, files_added: 1 },
  files_added: [{ file: 'StoryData/S9999.json', group: 'StoryData', family: 'story-main', entries: 131 }],
  vs_kr: {
    only_here: [
      { kind: 'group', change: 'added', name: 'MirrorDungeonEnemyBuffDesc' },
      { kind: 'file', change: 'added', name: 'StoryData/S9999.json' },
      { kind: 'entries', change: 'removed', name: 'Skills_personality-02.json', count: 2 },
      { kind: 'token', change: 'added', name: '<pause>' },
    ],
    only_other: [
      { kind: 'key', change: 'added', name: 'Bufs dataList[].excludeAutoLink' },
      { kind: 'entries', change: 'added', name: 'UI.json', count: 6 },
    ],
  },
}

async function mount(doc: DataDiff | null = diff, lang: 'kr' | 'jp' | 'en' = 'kr'): Promise<HTMLElement> {
  const rel = lang === 'kr' ? `rawdiff/${tag}.json` : `rawdiff/${tag}.${lang}.json`
  vi.stubGlobal('fetch', vi.fn(fixtureFetch(doc ? { [rel]: doc } : {})))
  const container = document.createElement('div')
  document.body.append(container)
  containers.push(container)
  render(<DataChangeView tag={tag} lang={lang} index={index} />, container)
  await vi.waitFor(() => expect(container.querySelector('.uv-head, .notice')).not.toBeNull())
  return container
}

const section = (el: HTMLElement, name: string) => el.querySelector(`[data-section="${name}"]`)
const texts = (root: Element | null, sel: string) => [...(root?.querySelectorAll(sel) ?? [])].map((x) => x.textContent?.replace(/\s+/g, ' ').trim())

describe('DataChangeView', () => {
  it('names the snapshot and the one it is compared with, and links back to the list', async () => {
    const el = await mount()
    expect(el.querySelector('.uv-head h1')?.textContent).toBe('資料變化 raw-20261112')
    expect(el.querySelector('.uv-meta')?.textContent).toContain('比對 raw-20261105')
    expect(el.querySelector('.uv-meta a')?.getAttribute('href')).toBe('#/d')
  })

  it('has a section for each kind of change, in the order of how much each asks to be looked at', async () => {
    const el = await mount()
    expect([...el.querySelectorAll('[data-section]')].map((s) => s.getAttribute('data-section'))).toEqual(['groups', 'keys', 'tokens', 'files', 'entries'])
    expect(texts(el, '[data-section] > h2')).toEqual(['檔案種類 +1 −1', '欄位 +2 −1', '文字標記 +6 −1', '檔案 +3 −1', '既有檔案的條目 +3 −1'])
    // Each section says what the change means for the translation.
    for (const s of el.querySelectorAll('[data-section]')) expect(s.querySelector('.dv-why')?.textContent?.length).toBeGreaterThan(10)
  })

  it('shows a new kind of file with its files, the family they go to and its keys', async () => {
    const groups = section(await mount(), 'groups')
    const added = groups?.querySelector('.dv-group.is-added')
    expect(added?.querySelector('.dv-group-name')?.textContent).toBe('SynergyKeyword')
    expect(added?.querySelector('.dv-group-meta')?.textContent).toBe('1 個檔案 · 族群 keywords')
    expect(texts(added ?? null, '.dv-key')).toEqual(['dataList[].id 文字 1 處', 'dataList[].name 文字 1 處'])
    const gone = groups?.querySelector('.dv-group.is-removed')
    expect(gone?.querySelector('.dv-group-name')?.textContent).toBe('OldThing')
    expect(gone?.querySelector('.dv-group-meta')?.textContent).toBe('2 個檔案 · 族群 ui')
  })

  it('lists the keys a kind of file gained and lost, with the kinds of value and where the first is', async () => {
    const rows = section(await mount(), 'keys')?.querySelectorAll('.dv-key-row') ?? []
    expect([...rows].map((r) => r.className.replace('dv-key-row ', ''))).toEqual(['is-added', 'is-removed', 'is-added'])
    expect(texts(section(await mount(), 'keys'), '.dv-key-row')).toEqual([
      '+ Bufs dataList[].excludeAutoLink 布林 14 處 Bufs-walpu10.json',
      '− Bufs dataList[].undefined 空值、文字 2 處 Bufs.json',
      '+ Skills dataList[].levelList[].flavor 文字 3 處 Skills_personality-02.json',
    ])
    // A text key of a file the site has links to its first unit; the others are plain.
    expect([...rows].map((r) => r.querySelector('a')?.getAttribute('href') ?? null)).toEqual([
      null,
      null,
      '#/f/Skills_personality-02.json?u=%2FdataList%2F0%2FlevelList%2F0%2Fflavor',
    ])
  })

  it('splits the new bracket names into those an entry defines and those nothing defines, which it flags', async () => {
    const tokens = section(await mount(), 'tokens')
    expect(texts(tokens, '[data-tokens="bracket-defined"] .dv-token')).toEqual(['[ComboSkill] 41 處 BattleKeywords、Bufs', '[OnBattleEnd] 6 處 SkillTag'])
    const undefinedOnes = tokens?.querySelector('[data-tokens="bracket-undefined"]')
    expect(texts(undefinedOnes ?? null, '.dv-token')).toEqual(['[Conbustion] 2 處 EGOgift.json'])
    expect(undefinedOnes?.classList.contains('is-flagged')).toBe(true)
    expect(texts(tokens, '[data-tokens="placeholder"] .dv-token')).toEqual(['{waveNum} 1 處 BattleUIText.json'])
    expect(texts(tokens, '[data-tokens="style"] .dv-token')).toEqual(['<style="number_highlight"> 1 處 MainUIText.json'])
    expect(texts(tokens, '[data-tokens="tag"] .dv-token')).toEqual(['<font> 12 處 StoryData/S9999.json'])
    expect(texts(tokens, '[data-tokens="removed"] .dv-token')).toEqual(['[OldMark] 上一版 1 處'])
    // The first text with a token is linked when the site has its file.
    expect(tokens?.querySelector('[data-tokens="tag"] a')?.getAttribute('href')).toBe('#/f/StoryData/S9999.json?u=%2FdataList%2F4%2Fcontent')
    expect(tokens?.querySelector('[data-tokens="placeholder"] a')).toBeNull()
  })

  it('lists the added and removed files with family and entries, and flags a file that only the fallback family takes', async () => {
    const files = section(await mount(), 'files')
    expect(texts(files, '.dv-file')).toEqual([
      '+ StoryData/S9999.json story-main 131 筆',
      '+ SynergyKeyword.json keywords 1 筆',
      '+ Mystery.json unsorted 沒有分流規則 4 筆',
      '− OldThing-1.json ui 5 筆',
    ])
    expect([...(files?.querySelectorAll('.dv-file') ?? [])].map((f) => f.querySelector('a')?.getAttribute('href') ?? null)).toEqual(['#/f/StoryData/S9999.json', null, null, null])
    expect(files?.querySelectorAll('.dv-file .dv-flag').length).toBe(1)
  })

  it('lists the files whose entries changed, the ids folded away', async () => {
    const entries = section(await mount(), 'entries')
    expect(texts(entries, '.dv-entries > summary')).toEqual(['Skills_personality-02.json 53 → 55 筆 +3 −1', 'NoID.json 1 → 2 筆'])
    const first = entries?.querySelector('.dv-entries')
    expect(first?.hasAttribute('open')).toBe(false)
    expect(texts(first ?? null, '.dv-ids')).toEqual(['新增：1021101、1021102、1021103', '移除：1020999'])
    expect(first?.querySelector('summary a')?.getAttribute('href')).toBe('#/f/Skills_personality-02.json')
    expect(texts(entries?.querySelectorAll('.dv-entries')[1] ?? null, '.dv-ids')).toEqual(['條目沒有 id，只知道筆數變了'])
  })

  it('leaves out the sections with nothing, and says so when nothing changed at all', async () => {
    let el = await mount({ ...empty, counts: { ...zeroCounts, files_added: 1 }, files_added: diff.files_added.slice(0, 1) })
    expect([...el.querySelectorAll('[data-section]')].map((s) => s.getAttribute('data-section'))).toEqual(['files'])
    expect(el.querySelector('.dv-none')).toBeNull()
    clearDataCache()
    el = await mount(empty)
    expect(el.querySelector('[data-section]')).toBeNull()
    expect(el.querySelector('.dv-none')?.textContent).toContain('結構沒有變化')
  })

  it('shows how the lists of the common files changed, and the files no list names', async () => {
    const lists = {
      compared: true,
      changes: [
        { file: 'RemoteLocalizeFileList.json', list: 'ui', added: ['Walpu10EventText', 'UnitKeyword-a1c10p3'], removed: [] },
        { file: 'RemoteLocalizeFileList.json', list: 'skill', added: [], removed: ['Skills_Old'] },
        { file: 'etc/VoiceTable.json', list: 'voiceKeys', added: ['85SV-014'], removed: [] },
      ],
      unlisted: [
        { file: 'SynergyKeyword.json', entries: 1, new: true },
        { file: 'Skills_personality-02.json', entries: 25 },
      ],
    }
    const el = await mount({ ...empty, counts: { ...zeroCounts, lists_added: 3, lists_removed: 1, unlisted_new: 1 }, lists })
    const s = section(el, 'lists')
    expect(s?.querySelector('h2')?.textContent?.replace(/\s+/g, ' ').trim()).toBe('檔案清單 +3 −1')
    expect(s?.querySelector('.dv-why')?.textContent?.length).toBeGreaterThan(10)
    expect(texts(s, '.dv-list-row')).toEqual([
      '+ RemoteLocalizeFileList.json ui Walpu10EventText、UnitKeyword-a1c10p3',
      '− RemoteLocalizeFileList.json skill Skills_Old',
      '+ etc/VoiceTable.json voiceKeys 85SV-014',
    ])
    // The files no list names: the new ones first and flagged; one the site has is a link.
    const un = s?.querySelector('[data-lists="unlisted"]')
    expect(un?.querySelector('h3')?.textContent?.replace(/\s+/g, ' ').trim()).toBe('不在清單裡的檔 2')
    expect(texts(un ?? null, '.dv-unlisted')).toEqual(['SynergyKeyword.json 1 筆 這一版才出現', 'Skills_personality-02.json 25 筆'])
    expect(un?.querySelectorAll('.dv-unlisted .dv-flag').length).toBe(1)
    expect([...(un?.querySelectorAll('.dv-unlisted') ?? [])].map((u) => u.querySelector('a')?.getAttribute('href') ?? null)).toEqual([null, '#/f/Skills_personality-02.json'])
    // The section stands on its own: nothing else of the snapshot changed.
    expect(el.querySelector('.dv-none')).toBeNull()
  })

  it('says so when the snapshot before has no lists to compare with, and has no section without lists', async () => {
    let el = await mount({ ...empty, lists: { compared: false, changes: [], unlisted: [{ file: 'SynergyKeyword.json', entries: 1 }] } })
    const s = section(el, 'lists')
    expect(s?.querySelector('.dv-first-lists')?.textContent).toContain('上一個快照沒有清單')
    expect(s?.querySelector('.dv-list-row')).toBeNull()
    expect(texts(s, '.dv-unlisted')).toEqual(['SynergyKeyword.json 1 筆'])
    clearDataCache()
    el = await mount(empty)
    expect(section(el, 'lists')).toBeNull()
    // The pack of another language shows no lists, even if the data had them.
    clearDataCache()
    el = await mount({ ...jp, lists: { compared: true, changes: [], unlisted: [] } }, 'jp')
    expect(section(el, 'lists')).toBeNull()
  })

  it('has a tab for the pack of each language, the one shown marked', async () => {
    let el = await mount()
    const tabs = (root: HTMLElement) => [...root.querySelectorAll('.dv-langs a')].map((a) => [a.textContent, a.getAttribute('href'), a.getAttribute('aria-current')])
    expect(tabs(el)).toEqual([
      ['KR', '#/d/raw-20261112', 'page'],
      ['JP', '#/d/raw-20261112?l=jp', null],
      ['EN', '#/d/raw-20261112?l=en', null],
    ])
    // The KR pack is what the others are compared with: it has no such section.
    expect(section(el, 'vs-kr')).toBeNull()
    clearDataCache()
    el = await mount(jp, 'jp')
    expect(tabs(el).map((t) => t[2])).toEqual([null, 'page', null])
  })

  it('lists, for the pack of another language, the changes only it has and those only KR has', async () => {
    const el = await mount(jp, 'jp')
    // Before the sections of the pack itself.
    expect([...el.querySelectorAll('[data-section]')].map((s) => s.getAttribute('data-section'))).toEqual(['vs-kr', 'files'])
    const vs = section(el, 'vs-kr')
    expect(vs?.querySelector('h2')?.textContent?.replace(/\s+/g, ' ').trim()).toBe('和 KR 不同的變化 6')
    expect(vs?.querySelector('.dv-why')?.textContent?.length).toBeGreaterThan(10)
    expect(texts(vs, '[data-vs="here"] > h3')).toEqual(['只有 JP 有 4'])
    expect(texts(vs, '[data-vs="here"] .dv-item')).toEqual([
      '+ 檔案種類 MirrorDungeonEnemyBuffDesc',
      '+ 檔案 StoryData/S9999.json',
      '− 條目 Skills_personality-02.json 2 個 id',
      '+ 標記 <pause>',
    ])
    expect(texts(vs, '[data-vs="kr"] > h3')).toEqual(['只有 KR 有 2'])
    expect(texts(vs, '[data-vs="kr"] .dv-item')).toEqual(['+ 欄位 Bufs dataList[].excludeAutoLink', '+ 條目 UI.json 6 個 id'])
    // A file the site has is a link.
    expect([...(vs?.querySelectorAll('[data-vs="here"] .dv-item') ?? [])].map((i) => i.querySelector('a')?.getAttribute('href') ?? null)).toEqual([
      null,
      '#/f/StoryData/S9999.json',
      '#/f/Skills_personality-02.json',
      null,
    ])
  })

  it('says so when the pack of another language changed just as KR did', async () => {
    const el = await mount({ ...jp, vs_kr: { only_here: [], only_other: [] } }, 'jp')
    const vs = section(el, 'vs-kr')
    expect(vs?.querySelector('.dv-same')?.textContent).toContain('和 KR 完全相同')
    expect(vs?.querySelector('[data-vs]')).toBeNull()
    // Nothing changed at all: the note on the pack itself names nothing of KR.
    clearDataCache()
    const none = await mount({ ...empty, lang: 'en', vs_kr: { only_here: [], only_other: [] } }, 'en')
    expect(none.querySelector('.dv-none')).not.toBeNull()
    expect(section(none, 'vs-kr')?.querySelector('.dv-same')).not.toBeNull()
  })

  it('shows the same notice as the other pages when the load fails', async () => {
    const el = await mount(null)
    expect(el.querySelector('.notice')?.textContent).toContain('資料已更新，請重新整理')
  })
})
