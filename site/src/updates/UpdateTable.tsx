import { useContext, useState } from 'preact/hooks'
import { ShownText } from '../review/parts'
import type { UpdateEntry } from '../types'
import { CauseChip, KIND_NAMES, Origins, TextPair, entryKr } from './UpdateCard'

const key = (e: UpdateEntry) => `${e.file}|${e.ptr}|${e.kind}`

function Row({ entry: e }: { entry: UpdateEntry }) {
  const { langs } = useContext(ShownText)
  const [open, setOpen] = useState(false)
  const kind = KIND_NAMES[e.kind]
  const kr = entryKr(e)
  return (
    <tr class={open ? 'ut-row is-open' : 'ut-row'} aria-expanded={open ? 'true' : 'false'} onClick={() => setOpen(!open)}>
      <td class="ut-where">
        {/* A click on the button reaches the row, which does the toggling. */}
        <button type="button" class="ut-toggle" aria-label={open ? '收合' : '展開'}>
          {open ? '▾' : '▸'}
        </button>
        <code>
          {e.field} {e.ptr}
        </code>
      </td>
      <td class="ut-kind">
        <span class={`uc-kind ${kind.cls}`}>{kind.name}</span>
        {e.cause && <CauseChip cause={e.cause} />}
      </td>
      <td class="ut-text">{langs.kr && <TextPair lang="kr" old={kr.old} now={kr.now} />}</td>
      <td class="ut-text">{langs.zh && <TextPair lang="zh" old={e.old_zh} now={e.new_zh} />}</td>
      <td class="ut-origin">
        <Origins entry={e} />
      </td>
    </tr>
  )
}

/** The entries as a table, grouped by file; each row is collapsed to a few lines until it is clicked. */
export function UpdateTable({ entries }: { entries: UpdateEntry[] }) {
  const groups = new Map<string, UpdateEntry[]>()
  for (const e of entries) {
    const g = groups.get(e.file)
    if (g) g.push(e)
    else groups.set(e.file, [e])
  }
  return (
    <div class="ut-scroll">
      <table class="ut-table">
        <thead>
          <tr>
            <th scope="col">欄位</th>
            <th scope="col">種類</th>
            <th scope="col">KR</th>
            <th scope="col">TW</th>
            <th scope="col">來源</th>
          </tr>
        </thead>
        {[...groups].map(([file, list]) => (
          <tbody key={file}>
            <tr class="ut-file">
              <th colSpan={5} scope="colgroup">
                {file}
              </th>
            </tr>
            {list.map((e) => (
              <Row key={key(e)} entry={e} />
            ))}
          </tbody>
        ))}
      </table>
    </div>
  )
}
