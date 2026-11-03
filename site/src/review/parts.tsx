import { Fragment, type ComponentChildren } from 'preact'
import { useRef, useState } from 'preact/hooks'
import { fileHref } from '../route'
import { LANG_ATTR, type Lang, type Confirmed } from '../types'
import { TmpText } from '../tmp/TmpText'
import { copyText, copyToClipboard } from './copy'
import type { Piece } from './diff'

/** The path and the pointer of a unit key, "<file path>|<pointer>". */
export function splitKey(key: string): { path: string; ptr: string } {
  const bar = key.indexOf('|')
  return bar < 0 ? { path: key, ptr: '' } : { path: key.slice(0, bar), ptr: key.slice(bar + 1) }
}

/** A link to a unit in the file view. */
export function UnitLink({ unitKey, text }: { unitKey: string; text?: string }) {
  const { path, ptr } = splitKey(unitKey)
  return (
    <a href={fileHref(path, ptr || undefined)} title={unitKey}>
      {text ?? `${path} ${ptr}`}
    </a>
  )
}

export function UnitLinks({ keys }: { keys: string[] }) {
  return (
    <div class="rv-units">
      {keys.map((k) => (
        <UnitLink key={k} unitKey={k} />
      ))}
    </div>
  )
}

/** The mark of the units of an item that wait for retranslation: all of them, or some. */
export function Pending({ pending, units }: { pending: number; units: number }) {
  if (pending <= 0) return null
  return <span class="rv-pending">{pending >= units ? '等待重翻' : `待重翻 ${pending}`}</span>
}

/** The ID of an item, selectable, and the button that copies it with a piece of the text. */
export function IdCopy({ id, source }: { id: string; source: string }) {
  const code = useRef<HTMLElement>(null)
  const [said, setSaid] = useState('')
  const copy = async () => {
    if (!code.current) return
    const result = await copyToClipboard(copyText(id, source), code.current)
    setSaid(result === 'copied' ? '已複製，貼到對話中再加上你的決定' : '瀏覽器不讓複製，已選取 ID，請按 Ctrl/⌘+C')
  }
  return (
    <span class="rv-id">
      <code ref={code}>{id}</code>
      <button type="button" class="rv-copy" onClick={copy}>
        複製 ID
      </button>
      <span class="rv-said" role="status">
        {said}
      </span>
    </span>
  )
}

/** The game text of one language, with its tags shown as what they do. */
export function Row({ label, lang, text, mark }: { label: string; lang: Lang; text: string; mark?: RegExp }) {
  return (
    <>
      <dt>{label}</dt>
      <dd lang={LANG_ATTR[lang]}>
        <TmpText text={text} raw={false} lang={lang} {...(mark ? { mark } : {})} />
      </dd>
    </>
  )
}

/** Why and when a person confirmed an item. */
export function ConfirmedLine({ confirmed }: { confirmed: Confirmed | undefined }) {
  if (!confirmed) return null
  return (
    <p class="rv-confirmed">
      已確認（{confirmed.date}）：{confirmed.reason}
    </p>
  )
}

/** The pieces of a translation: [ID]s as tokens, the highlight style as colour, what differs on a background, an extra space as a sign. */
export function DiffText({ pieces }: { pieces: Piece[] }) {
  const out: ComponentChildren[] = []
  let run: ComponentChildren[] | undefined
  const flush = (k: number) => {
    if (run) out.push(<span key={k} class="rv-d">{run}</span>)
    run = undefined
  }
  pieces.forEach((p, k) => {
    const node = piece(p, k)
    if (p.same) {
      flush(k)
      out.push(node)
    } else (run ??= []).push(node)
  })
  flush(pieces.length)
  return <span class="tmp" lang="zh-Hant-TW">{out}</span>
}

function piece(p: Piece, k: number): ComponentChildren {
  let node: ComponentChildren
  if (p.tok) node = <span class="tok">{p.text}</span>
  else if (p.text === ' ' && !p.same) node = <span class="rv-sp" title="多了一個空格">␣</span>
  else node = p.text
  return p.hl ? <span key={k} class="rv-hl">{node}</span> : <Fragment key={k}>{node}</Fragment>
}
