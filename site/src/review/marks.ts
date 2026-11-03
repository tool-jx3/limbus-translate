/** The characters that make each check fire, for showing them in the TW text; a presentation only, the check itself is the Go one. */
export const WARNING_MARKS: Partial<Record<string, RegExp>> = {
  kana: /[ぁ-ゖァ-ヺ]/g,
  'halfwidth-punct': /[,!?:;]/g,
  'halfwidth-bracket': /[()]/g,
  'ascii-ellipsis': /\.{2,}/g,
  'curly-quotes': /[‘’“”]/g,
}
