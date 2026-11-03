import type { Meta } from '../types'

const RELEASES = 'https://github.com/tool-jx3/limbus-translate/releases/tag/'

export interface FooterProps {
  /** The version of the data; not there until meta.json is loaded. */
  meta?: Meta
}

export function Footer({ meta }: FooterProps) {
  return (
    <footer class="foot">
      <p>KR／JP／EN 原文著作權屬 Project Moon；本站非商業、供翻譯校對使用。TW 譯文採用 CC BY-NC-SA 4.0，含 Paratranz 社群譯文（第一階段匯入）。</p>
      {meta && (
        <p class="foot-version">
          資料版本：原文 <code>{meta.raw_tag}</code>
          {meta.release !== '' && (
            <>
              ；譯文{' '}
              <a href={RELEASES + encodeURIComponent(meta.release)} target="_blank" rel="noopener noreferrer">
                {meta.release}
              </a>
            </>
          )}
        </p>
      )}
    </footer>
  )
}
