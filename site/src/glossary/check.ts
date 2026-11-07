import type { Forms } from '../types'

/** The check of an entry as the pages show it: hard, off, or warn where it is left out. */
export type CheckLevel = 'hard' | 'off' | 'warn'

export const checkLevel = (check: string | undefined): CheckLevel => (check === 'hard' ? 'hard' : check === 'off' ? 'off' : 'warn')

/** What each level means, in a few words (the title of the level on the page). */
export const CHECK_TITLE: Record<CheckLevel, string> = {
  hard: '缺譯名時送出會退回（note 寫出原詞就收下）',
  off: '一般用語：照樣注入批次，缺譯名不警告；forbidden 照樣檢查',
  warn: '缺譯名時只警告',
}

/** The forms of a jp or en, one or several (rules.Forms); none where it is absent. */
export const formsOf = (v: Forms | undefined): string[] => (v === undefined ? [] : Array.isArray(v) ? v : [v])
