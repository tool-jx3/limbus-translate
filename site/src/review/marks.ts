// What to mark in the TW text of a warning card. Each pattern mirrors one check of
// internal/validate (validate.go, checkStyle and its helpers), so the marks show
// the characters the Go check flagged; the card shows the check's own message
// beside them. Tags and placeholders are not text: TmpText marks the text nodes
// only, as the Go checks strip them first.

// isChinese in Go: a Han character or CJK punctuation (U+3000-303F, U+FF00-FFEF).
const CJK = '[\\p{Script=Han}\\u3000-\\u303F\\uFF00-\\uFFEF]'
// halfwidthPunct in Go: , . : except the dots of "..." and the colons of "::".
const PUNCT = ',|(?<!\\.)\\.(?!\\.)|(?<!:):(?!:)'

export const WARNING_MARKS: Partial<Record<string, RegExp>> = {
  // checkStyle, kana: U+3040-U+30FF except the middle dot U+30FB.
  kana: /[぀-ヺー-ヿ]/gu,
  // halfwidthPunct: a half-width , . : touching Chinese text.
  'halfwidth-punct': new RegExp(`(?<=${CJK})(?:${PUNCT})|(?:${PUNCT})(?=${CJK})`, 'gu'),
  // halfwidthBrackets: ( ) (the Go check applies it to the labels of RPGSystem/ only).
  'halfwidth-bracket': /[()]/g,
  // asciiEllRe: three dots or more.
  'ascii-ellipsis': /\.{3,}/g,
  // checkStyle, curly-quotes: “ and ” only.
  'curly-quotes': /[“”]/g,
}

/** Marks every occurrence of a term (a KR or a TW name) in a text; the term is matched as written, not as a pattern. An empty term marks nothing. */
export function termMark(term: string): RegExp | undefined {
  if (term === '') return undefined
  return new RegExp(term.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'gu')
}
