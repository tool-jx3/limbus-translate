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
  'name-differs': { name: '與既有譯名不同', desc: '名稱欄位的譯法和既有譯文不同：同一個東西就統一，語境不同就保留' },
  'keyword-link': { name: '關鍵字誤連結', desc: '依遊戲程式推算，TW 的這些字會被連成 KR 沒提到的狀態' },
  'bracket-text': { name: '方括號文字', desc: '技能、被動文字裡不是 ID 的 [文字]，依遊戲程式推算會顯示成 Unknown' },
  'page-count': { name: '頁數不同', desc: '選擇事件以空行分頁；依遊戲程式推算，頁數比 KR 少時會補「...」頁' },
  'level-diff': { name: '同步等級比較', desc: 'KR 兩個同步等級相同的行，TW 不同：依遊戲程式推算，比較畫面會標成有變' },
  'story-comment': { name: '// 截斷', desc: '依遊戲程式推算，劇情文字裡 // 之後的字不會顯示' },
  'stray-brace': { name: '多餘的大括號', desc: 'KR 沒有的 { 或 }，代入數值時可能出錯' },
  'regex-char': { name: '正規表示式字元', desc: '依遊戲程式推算，這個欄位的文字會被當成正規表示式，半形括號等符號不成對會出錯' },
  'literal-newline': { name: '字面的 \\n', desc: '反斜線加 n 不是換行；依遊戲程式推算會原樣顯示（選擇事件裡會被當成分頁）' },
  'reserved-value': { name: '保留字', desc: '依遊戲程式推算，值是 Unknown 或 Error 時會被當成沒有資料' },
  'newline-space': { name: '換行旁的空格', desc: '依遊戲程式推算，稱號的換行在單行顯示時變成空格，旁邊再有空格就是兩個' },
  'qmark-count': { name: '問號個數', desc: '依遊戲程式推算，罪孽型敵人名字的半形 ? 個數決定「N型」，要和 KR 相同' },
  'bullet-count': { name: '條列符號', desc: '但丁筆記裡「換行後接 -」的數量和 KR 不同' },
  'ruby-form': { name: 'ruby 寫法', desc: '依遊戲程式推算，對話框只轉換 <ruby=注音>本文</ruby> 這一種寫法' },
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
  name: '與既有譯名不同',
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
