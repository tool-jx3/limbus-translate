import type { ComponentChildren } from 'preact'
import { useEffect, useMemo, useState } from 'preact/hooks'
import { Notice } from '../app/Notice'
import { DataError, loadDataChange } from '../data'
import { dataChangeHref, dataChangesHref, fileHref } from '../route'
import type { DataDiff, DataFile, DataGroup, DataItem, DataKey, DataLang, DataToken, DataVersus, Index } from '../types'

/** Where the report on the game's code is; the page names its chapters. */
const REPORT = 'https://github.com/tool-jx3/limbus-translate/blob/main/docs/research/2026-11-15-game-text-logic/README.md'

const KIND_NAMES: Record<DataKey['kinds'][number], string> = { bool: '布林', null: '空值', number: '數字', string: '文字' }

const LANGS: DataLang[] = ['kr', 'jp', 'en']

const ITEM_NAMES: Record<DataItem['kind'], string> = { group: '檔案種類', file: '檔案', key: '欄位', entries: '條目', token: '標記' }

type Loaded = { tag: string; lang: DataLang } & ({ status: 'ready'; diff: DataDiff } | { status: 'error'; error: DataError })

function useDataChange(tag: string, lang: DataLang): Loaded | undefined {
  const [state, setState] = useState<Loaded>()
  useEffect(() => {
    let live = true
    loadDataChange(tag, lang).then(
      (diff) => {
        if (live) setState({ tag, lang, status: 'ready', diff })
      },
      (e: unknown) => {
        if (live) setState({ tag, lang, status: 'error', error: e instanceof DataError ? e : new DataError('network', String(e)) })
      },
    )
    return () => {
      live = false
    }
  }, [tag, lang])
  return state?.tag === tag && state.lang === lang ? state : undefined
}

/** What the page knows of the site: which files it has, and the family that takes what no rule routes. */
interface Site {
  files: Set<string>
  fallback: string
}

/** A file name, linked to its page (and to a unit of it) when the site has the file. */
function FileName({ site, file, ptr }: { site: Site; file: string; ptr?: string }) {
  return site.files.has(file) ? <a href={fileHref(file, ptr)}>{file}</a> : <span>{file}</span>
}

function Section({ name, title, added, removed, why, children }: { name: string; title: string; added: number; removed: number; why: ComponentChildren; children: ComponentChildren }) {
  if (added + removed === 0) return null
  return (
    <section class="dv-section" data-section={name}>
      <h2>
        {title}
        {added > 0 && <span class="dv-n is-added"> +{added}</span>}
        {removed > 0 && <span class="dv-n is-removed"> −{removed}</span>}
      </h2>
      <p class="dv-why">{why}</p>
      {children}
    </section>
  )
}

/** A key: its path, the kinds of its values and how many there are. */
function KeyText({ k }: { k: DataKey }) {
  return (
    <>
      <code class="dv-path">{k.path}</code> <span class="dv-kinds">{k.kinds.map((kind) => KIND_NAMES[kind] ?? kind).join('、')}</span>{' '}
      <span class="dv-count">{k.count} 處</span>
    </>
  )
}

function GroupCard({ group: g, files, added }: { group: DataGroup; files: DataFile[]; added: boolean }) {
  const families = [...new Set(files.filter((f) => f.group === g.group && f.family).map((f) => f.family))]
  return (
    <div class={`dv-group ${added ? 'is-added' : 'is-removed'}`}>
      <p class="dv-group-head">
        <span class="dv-sign">{added ? '+' : '−'}</span> <strong class="dv-group-name">{g.group}</strong>{' '}
        <span class="dv-group-meta">
          {g.files} 個檔案{families.length > 0 && ` · 族群 ${families.join('、')}`}
        </span>
      </p>
      <ul class="dv-keys">
        {g.keys.map((k) => (
          <li key={k.path} class="dv-key">
            <KeyText k={k} />
          </li>
        ))}
      </ul>
    </div>
  )
}

/** A token as the text writes it. */
function tokenText(t: DataToken): string {
  switch (t.kind) {
    case 'bracket':
      return `[${t.name}]`
    case 'placeholder':
      return `{${t.name}}`
    case 'style':
      return `<style="${t.name}">`
    case 'tag':
      return `<${t.name}>`
  }
}

function TokenList({ name, title, why, tokens, site, flagged, tail }: { name: string; title: string; why: string; tokens: DataToken[]; site: Site; flagged?: boolean; tail: 'defined' | 'file' | 'before' }) {
  if (tokens.length === 0) return null
  return (
    <div class={`dv-tokens${flagged ? ' is-flagged' : ''}`} data-tokens={name}>
      <h3>
        {title} <span class="dv-n">{tokens.length}</span>
      </h3>
      <p class="dv-why">{why}</p>
      <ul>
        {tokens.map((t) => (
          <li key={`${t.kind} ${t.name}`} class="dv-token">
            <code>{tokenText(t)}</code>{' '}
            {tail === 'before' ? (
              <span class="dv-count">上一版 {t.count} 處</span>
            ) : (
              <>
                <span class="dv-count">{t.count} 處</span>{' '}
                {tail === 'defined' ? <span class="dv-defined">{t.defined_in.join('、')}</span> : <FileName site={site} file={t.file} ptr={t.pointer} />}
              </>
            )}
          </li>
        ))}
      </ul>
    </div>
  )
}

function FileRow({ file: f, added, site }: { file: DataFile; added: boolean; site: Site }) {
  return (
    <li class={`dv-file ${added ? 'is-added' : 'is-removed'}`}>
      <span class="dv-sign">{added ? '+' : '−'}</span> <FileName site={site} file={f.file} /> <span class="dv-family">{f.family ?? '—'}</span>{' '}
      {f.family === site.fallback && <span class="dv-flag">沒有分流規則</span>} <span class="dv-count">{f.entries} 筆</span>
    </li>
  )
}

function ItemList({ name, title, items, site }: { name: string; title: string; items: DataItem[]; site: Site }) {
  if (items.length === 0) return null
  return (
    <div class="dv-vs-list" data-vs={name}>
      <h3>
        {title} <span class="dv-n">{items.length}</span>
      </h3>
      <ul>
        {items.map((it) => (
          <li key={`${it.kind} ${it.change} ${it.name}`} class={`dv-item ${it.change === 'added' ? 'is-added' : 'is-removed'}`}>
            <span class="dv-sign">{it.change === 'added' ? '+' : '−'}</span> <span class="dv-kinds">{ITEM_NAMES[it.kind]}</span>{' '}
            {it.kind === 'file' || it.kind === 'entries' ? <FileName site={site} file={it.name} /> : <code>{it.name}</code>}
            {it.kind === 'entries' && <span class="dv-count"> {it.count ?? 0} 個 id</span>}
          </li>
        ))}
      </ul>
    </div>
  )
}

/** For the pack of another language: the changes that only it has, and those that only KR has. */
function VersusKR({ lang, vs, site }: { lang: DataLang; vs: DataVersus; site: Site }) {
  const n = vs.only_here.length + vs.only_other.length
  return (
    <section class="dv-section" data-section="vs-kr">
      <h2>
        和 KR 不同的變化{n > 0 && <span class="dv-n"> {n}</span>}
      </h2>
      <p class="dv-why">
        同一次更新裡，這個語言的資料和韓文原文變化不一樣的地方。常見的原因是其中一邊比較早或比較晚收到同一批內容：翻譯時參考的 {lang.toUpperCase()}{' '}
        可能因此缺少對應的檔案或條目，也可能已經有韓文還沒有的內容。
      </p>
      {n === 0 ? (
        <p class="dv-same">這個語言這一次的變化和 KR 完全相同。</p>
      ) : (
        <>
          <ItemList name="here" title={`只有 ${lang.toUpperCase()} 有`} items={vs.only_here} site={site} />
          <ItemList name="kr" title="只有 KR 有" items={vs.only_other} site={site} />
        </>
      )}
    </section>
  )
}

/** The page of one snapshot: what the pack of a language has that the snapshot before it has not, and the reverse, as data. */
export function DataChangeView({ tag, lang, index }: { tag: string; lang: DataLang; index: Index }) {
  const state = useDataChange(tag, lang)
  const site = useMemo<Site>(
    () => ({ files: new Set(index.files.map((f) => f.path)), fallback: index.families.find((f) => f.nodes.length === 0)?.id ?? '' }),
    [index],
  )
  if (!state) return <p role="status">資料載入中…</p>
  if (state.status === 'error') return <Notice error={state.error} />
  const d = state.diff
  const c = d.counts
  const added = d.tokens.filter((t) => t.change === 'added')
  const brackets = added.filter((t) => t.kind === 'bracket')
  const nothing = Object.values(c).every((n) => n === 0) && d.entries.length === 0
  return (
    <section class="update data-change">
      <header class="uv-head">
        <h1>資料變化 {d.to}</h1>
        <p class="uv-meta">
          <span>比對 {d.from}</span>
          <a href={dataChangesHref()}>所有快照</a>
          <a href={REPORT}>遊戲程式研究報告</a>
        </p>
        <nav class="dv-langs" aria-label="語言">
          {LANGS.map((l) => (
            <a key={l} href={dataChangeHref(tag, l)} {...(l === lang ? { 'aria-current': 'page' as const } : {})}>
              {l.toUpperCase()}
            </a>
          ))}
        </nav>
      </header>
      {lang !== 'kr' && d.vs_kr && <VersusKR lang={lang} vs={d.vs_kr} site={site} />}
      {nothing && <p class="dv-none">和上一個快照相比，資料的結構沒有變化；文字內容的修改看更新頁。</p>}

      <Section
        name="groups"
        title="檔案種類"
        added={c.new_groups}
        removed={c.gone_groups}
        why="以前沒有出現過（或整個消失）的檔名開頭。新的種類要確認三件事：分到哪個族群（guides/routing.yaml）、哪些欄位不該翻（rules/fields.manual.yaml）、遊戲在哪裡顯示它（研究報告第 1、6、7 章是照當時的檔案整理的，新的種類不在裡面）。"
      >
        {d.new_groups.map((g) => (
          <GroupCard key={g.group} group={g} files={d.files_added} added />
        ))}
        {d.gone_groups.map((g) => (
          <GroupCard key={g.group} group={g} files={d.files_removed} added={false} />
        ))}
      </Section>

      <Section
        name="keys"
        title="欄位"
        added={c.keys_added}
        removed={c.keys_removed}
        why="既有的檔案種類多了或少了 JSON 欄位。新的文字欄位預設會進待譯清單，但遊戲不一定顯示它（研究報告第 7 章）；新的數字或布林欄位不用翻，卻可能是遊戲處理文字的新開關，值得回頭查遊戲程式（研究方法的 fingerprint.py 會指出哪個類別多了欄位）。"
      >
        <ul class="dv-key-rows">
          {d.keys.flatMap((g) =>
            [...g.added.map((k) => ({ k, added: true })), ...g.removed.map((k) => ({ k, added: false }))].map(({ k, added }) => (
              <li key={`${g.group} ${added} ${k.path}`} class={`dv-key-row ${added ? 'is-added' : 'is-removed'}`}>
                <span class="dv-sign">{added ? '+' : '−'}</span> <strong class="dv-group-name">{g.group}</strong> <KeyText k={k} />{' '}
                {added && k.kinds.includes('string') ? <FileName site={site} file={k.file} ptr={k.pointer} /> : <span>{k.file}</span>}
              </li>
            )),
          )}
        </ul>
      </Section>

      <Section
        name="tokens"
        title="文字標記"
        added={c.tokens_added}
        removed={c.tokens_removed}
        why="整包原文第一次出現（或完全消失）的標記。譯文要原樣保留這些標記，檢查規則也要認得它們。"
      >
        <TokenList
          name="bracket-undefined"
          title="方括號名稱：沒有對應條目"
          why="沒有任何條目的 id 是這個名稱：可能是原文筆誤，也可能只是一般文字。遊戲不會把它換成關鍵字，譯文照原文保留。"
          tokens={brackets.filter((t) => t.defined_in.length === 0)}
          site={site}
          flagged
          tail="file"
        />
        <TokenList
          name="placeholder"
          title="具名代入符號"
          why="遊戲執行時會換成別的內容；新的名稱代入的是什麼還沒有查過（研究報告第 8 章）。"
          tokens={added.filter((t) => t.kind === 'placeholder')}
          site={site}
          tail="file"
        />
        <TokenList name="style" title="樣式名稱" why="<style> 用的名稱，遊戲的樣式表有定義才有效果。" tokens={added.filter((t) => t.kind === 'style')} site={site} tail="file" />
        <TokenList name="tag" title="標籤" why="以前沒用過的標籤名稱；要確認檢查規則把它當成標籤，而不是文字。" tokens={added.filter((t) => t.kind === 'tag')} site={site} tail="file" />
        <TokenList
          name="bracket-defined"
          title="方括號名稱：有對應條目"
          why="右邊列的檔案種類裡有同名的 id，是一般的新關鍵字或技能標籤，照既有規則處理。"
          tokens={brackets.filter((t) => t.defined_in.length > 0)}
          site={site}
          tail="defined"
        />
        <TokenList name="removed" title="不再出現" why="上一個快照有，這個快照整包都沒有了。" tokens={d.tokens.filter((t) => t.change === 'removed')} site={site} tail="before" />
      </Section>

      <Section
        name="files"
        title="檔案"
        added={c.files_added}
        removed={c.files_removed}
        why="新增或移除的檔案。族群決定翻譯時讀哪一份指引；標示「沒有分流規則」的檔案只被預設族群接手，要在 guides/routing.yaml 補上。"
      >
        <ul class="dv-files">
          {d.files_added.map((f) => (
            <FileRow key={f.file} file={f} added site={site} />
          ))}
          {d.files_removed.map((f) => (
            <FileRow key={f.file} file={f} added={false} site={site} />
          ))}
        </ul>
      </Section>

      {d.entries.length > 0 && (
        <section class="dv-section" data-section="entries">
          <h2>
            既有檔案的條目
            {c.entries_added > 0 && <span class="dv-n is-added"> +{c.entries_added}</span>}
            {c.entries_removed > 0 && <span class="dv-n is-removed"> −{c.entries_removed}</span>}
          </h2>
          <p class="dv-why">兩個快照都有的檔案，條目（dataList 的元素）多了或少了哪些 id。新檔案的條目算在檔案那一節；條目的文字改了什麼看更新頁。</p>
          {d.entries.map((e) => (
            <details key={e.file} class="dv-entries">
              <summary>
                <FileName site={site} file={e.file} />{' '}
                <span class="dv-count">
                  {e.before} → {e.after} 筆
                </span>
                {e.added.length > 0 && <span class="dv-n is-added"> +{e.added.length}</span>}
                {e.removed.length > 0 && <span class="dv-n is-removed"> −{e.removed.length}</span>}
              </summary>
              {e.added.length > 0 && <p class="dv-ids">新增：{e.added.join('、')}</p>}
              {e.removed.length > 0 && <p class="dv-ids">移除：{e.removed.join('、')}</p>}
              {e.added.length + e.removed.length === 0 && <p class="dv-ids">條目沒有 id，只知道筆數變了</p>}
            </details>
          ))}
        </section>
      )}
    </section>
  )
}
