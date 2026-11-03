import type { ComponentChildren } from 'preact'
import { useMemo } from 'preact/hooks'
import { LANG_ATTR, type Lang } from '../types'
import { parseTMP, type TmpNode } from './parse'

// Everything is built with Preact elements, so a string can only become text:
// no tag of it, known or not, ever reaches the DOM as markup.

const HEX_COLOR = /^#(?:[0-9a-f]{3}|[0-9a-f]{6}|[0-9a-f]{8})$/i
const PERCENT = /^(\d+(?:\.\d+)?)%$/
const ABSOLUTE = /^\d+(?:\.\d+)?(?:px|pt|em)?$/i

const MIN_SIZE = 0.75
const MAX_SIZE = 1.5
const ABSOLUTE_SIZE = 1.25

/** The font-size of a <size=…> value, in em, or undefined where it is not one we show. */
function sizeEm(value: string | undefined): string | undefined {
  if (value === undefined) return undefined
  const v = value.trim()
  const percent = PERCENT.exec(v)
  let em: number
  if (percent?.[1] !== undefined) em = Math.min(MAX_SIZE, Math.max(MIN_SIZE, Number(percent[1]) / 100))
  else if (ABSOLUTE.test(v)) em = ABSOLUTE_SIZE
  else return undefined
  return `${Number(em.toFixed(3))}em`
}

function renderNode(n: TmpNode, key: number): ComponentChildren {
  switch (n.t) {
    case 'text':
      return n.s
    case 'token':
      return <span key={key} class="tok">{n.s}</span>
    case 'br':
      return <br key={key} />
    case 'sprite':
      return <span key={key} class="sprite">{n.name === '' ? '[sprite]' : `[sprite:${n.name}]`}</span>
    case 'tag': {
      const kids = renderNodes(n.kids)
      switch (n.name) {
        case 'color':
          return n.value !== undefined && HEX_COLOR.test(n.value) ? <span key={key} style={{ color: n.value }}>{kids}</span> : kids
        case 'size': {
          const fontSize = sizeEm(n.value)
          return fontSize === undefined ? kids : <span key={key} style={{ fontSize }}>{kids}</span>
        }
        case 'b':
          return <b key={key}>{kids}</b>
        case 'i':
          return <i key={key}>{kids}</i>
        case 'u':
          return <u key={key}>{kids}</u>
        case 's':
          return <s key={key}>{kids}</s>
        case 'ruby':
          return n.value ? <ruby key={key}>{kids}<rt>{n.value}</rt></ruby> : kids
        default:
          return kids
      }
    }
  }
}

function renderNodes(nodes: TmpNode[]): ComponentChildren {
  return nodes.map(renderNode)
}

export interface TmpTextProps {
  text: string
  /** Show the string as it is, in a monospace font, tags and all. */
  raw: boolean
  /** The language of the text: it sets the lang attribute. */
  lang: Lang
}

/** A game string with its TextMeshPro tags shown as what they do. */
export function TmpText({ text, raw, lang }: TmpTextProps) {
  const nodes = useMemo(() => (raw ? [] : parseTMP(text)), [raw, text])
  if (raw) {
    return <span class="tmp tmp-raw" lang={LANG_ATTR[lang]}>{text}</span>
  }
  return <span class="tmp" lang={LANG_ATTR[lang]}>{renderNodes(nodes)}</span>
}
