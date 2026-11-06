import { previewHref } from '../route'
import type { Unit } from '../types'

export const COPY_NOT_SENT = 'copy：識別碼或各語言相同，不送翻'

/** The way into the translation preview of a unit; a copy unit is not sent for translation, so it gets a muted note instead. */
export function PreviewLink({ path, unit, named = false }: { path: string; unit: Unit; named?: boolean }) {
  if (unit.o === 'copy') {
    return (
      <span class="no-send" title={COPY_NOT_SENT}>
        不送翻
      </span>
    )
  }
  return (
    <a class="preview-link" href={previewHref(path, unit.p)} title={named ? `預覽「${unit.f}」的翻譯請求` : '預覽這個單位的翻譯請求'}>
      預覽
    </a>
  )
}
