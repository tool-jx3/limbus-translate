import { fileHref } from '../route'
import type { TodoItem, TodoSection } from '../types'
import { TODO_KINDS } from './names'
import { splitKey } from './parts'

/** The text of a line, with the `code` spans of it as code. */
function LineText({ text }: { text: string }) {
  return <>{text.split('`').map((part, k) => (k % 2 === 1 ? <code key={k}>{part}</code> : part))}</>
}

function Line({ item }: { item: TodoItem }) {
  if (item.key === undefined) return <LineText text={item.text} />
  const { path, ptr } = splitKey(item.key)
  return (
    <a href={fileHref(path, ptr || undefined)} title={item.key}>
      <LineText text={item.text} />
    </a>
  )
}

export function TodoView({ sections }: { sections: TodoSection[] }) {
  return (
    <div class="rv-todo">
      <p class="rv-lede">新說話者、新欄位、新檔案族群與過期的調整列在「規則建議」分頁，可以在那裡確認。</p>
      {sections.length === 0 ? (
        <div class="rv-empty">
          <h2>這次建置沒有待辦項目</h2>
          <p>會出現在這裡的種類：</p>
          <ul class="rv-kinds">
            {TODO_KINDS.map((k) => (
              <li key={k.kind}>
                <b>{k.title}</b> <code>{k.kind}</code>
              </li>
            ))}
          </ul>
        </div>
      ) : (
        sections.map((s) => (
          <section key={s.kind} class="rv-section">
            <h2>
              {s.title} <span class="rv-n">{s.items.length}</span>
            </h2>
            <p>
              <LineText text={s.hint} />
            </p>
            <ul>
              {s.items.map((item, k) => (
                <li key={k}>
                  <Line item={item} />
                </li>
              ))}
            </ul>
          </section>
        ))
      )}
    </div>
  )
}
