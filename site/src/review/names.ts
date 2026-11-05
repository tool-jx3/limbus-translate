/** The name and the one-sentence description of the check codes the page knows. */
export const WARNING_NAMES: Record<string, { name: string; desc: string }> = {
  kana: { name: '假名殘留', desc: 'TW 裡有日文假名：可能是漏譯，也可能是刻意保留的讀音' },
  'halfwidth-punct': { name: '半形標點', desc: '應為全形' },
  'ascii-ellipsis': { name: '英文省略號', desc: '應為「……」' },
  'length-ratio': { name: '長度差太多', desc: '可能漏譯或多譯' },
  'halfwidth-bracket': { name: '半形括號', desc: '應為全形（ ）' },
  'curly-quotes': { name: '彎引號', desc: '應為「」『』' },
  hangul: { name: '韓文殘留', desc: 'TW 裡有韓文字：可能是漏譯' },
  simplified: { name: '簡體字', desc: 'TW 裡有簡體字，應為繁體' },
  'forbidden-term': { name: '禁用譯法', desc: 'TW 用了詞彙表禁用的譯法' },
}

/** What to call a check code: the known ones by name, any other by its code and the message of its first unit. */
export function warningInfo(code: string, message: string): { name: string; desc: string } {
  return WARNING_NAMES[code] ?? { name: code, desc: message }
}

const SUGGESTION_KINDS: Record<string, string> = {
  inconsistent: '不一致的譯法',
  speaker: '新說話者',
  field: '欄位規則變化',
  family: '新檔案族群',
  note: '譯者備註',
  'term-new': '暫定詞',
  term: '詞彙候選',
  'adjustment-moved': '調整已移位',
  'adjustment-expired': '調整已過期',
}

/** The name of a kind of rule suggestion; an unknown kind is its own name. */
export function suggestionKindName(kind: string): string {
  return SUGGESTION_KINDS[kind] ?? kind
}

/** The to-do sections that can be on the page, for the page to say what shows up there. */
export const TODO_KINDS: { kind: string; title: string }[] = [
  { kind: 'needs_review', title: '翻譯失敗、暫用英文的單位' },
  { kind: 'frozen_changed', title: '凍結檔的 KR 有變' },
  { kind: 'frozen_fallback', title: '凍結檔的譯文破壞原文結構' },
  { kind: 'loop_error', title: '翻譯迴圈因錯誤中斷' },
  { kind: 'not_translated', title: '翻譯沒有執行' },
]
