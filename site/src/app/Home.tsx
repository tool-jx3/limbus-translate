import type { Index, Meta } from '../types'

const number = new Intl.NumberFormat('en-US')

export interface HomeProps {
  meta: Meta
  index: Index
  /** Open the directory where it is a drawer. */
  onOpenDirectory: () => void
}

export function Home({ meta, index, onOpenDirectory }: HomeProps) {
  return (
    <section class="home">
      <h1>Limbus 譯文對照</h1>
      <p>KR 韓文、JP 日文、EN 英文與 TW 臺灣繁體中文譯文，一段一段並排對照，供翻譯校對使用。</p>
      <dl class="stats">
        <div>
          <dt>族群</dt>
          <dd>{number.format(index.families.length)}</dd>
        </div>
        <div>
          <dt>檔案</dt>
          <dd>{number.format(meta.files)}</dd>
        </div>
        <div>
          <dt>單位</dt>
          <dd>{number.format(meta.units)}</dd>
        </div>
      </dl>
      <h2>檔案族群</h2>
      <div class="fam-wrap">
        <table class="fam-table">
          <thead>
            <tr>
              <th scope="col">族群</th>
              <th scope="col">glossary</th>
              <th scope="col">節點</th>
              <th scope="col">檔數</th>
            </tr>
          </thead>
          <tbody>
            {index.families.map((fam) => (
              <tr key={fam.id}>
                <th scope="row">{fam.id}</th>
                <td>
                  <span class={`pill pill-${fam.glossary}`}>{fam.glossary}</span>
                </td>
                <td class="fam-nodes">
                  {fam.nodes.map((node) => (
                    <code key={node}>{node}</code>
                  ))}
                </td>
                <td class="fam-n">{number.format(fam.files.length)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <ul class="howto">
        <li>
          目錄依分群列出檔案，點群組名稱展開；章節、角色、活動可由下拉選單篩出。
          <button type="button" class="link only-narrow" onClick={onOpenDirectory}>
            開啟目錄
          </button>
        </li>
        <li>「篩選檔案」比對路徑、地點與說話者。</li>
        <li>「全文搜尋」在選定語言的所有字串中找文字，可用正規表示式。</li>
        <li>上方的語言開關決定顯示哪些語言欄；選擇會記在這個瀏覽器。</li>
      </ul>
    </section>
  )
}
