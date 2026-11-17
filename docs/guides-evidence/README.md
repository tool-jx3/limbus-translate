# 指引的依據

`guides/` 裡每條規則的統計、決定來源、經過與出處放在這裡。指引只寫譯者照著做的規則；為什麼這樣寫、有多少證據，查這裡。寫法規範見 `guides/README.md`。

## 路徑對應

路徑與指引相同，只把開頭的 `guides/` 換成 `docs/guides-evidence/`：

| 指引 | 依據檔 |
|---|---|
| `guides/base.md` | `docs/guides-evidence/base.md` |
| `guides/families/keywords.md` | `docs/guides-evidence/families/keywords.md` |
| `guides/shared/ruby.md` | `docs/guides-evidence/shared/ruby.md` |
| `guides/characters/sinclair.md` | `docs/guides-evidence/characters/sinclair.md` |

只有有依據的指引才有對應檔；沒有依據可寫的指引不必建檔。

## 段落格式

依據檔用與指引相同的規則標題分段（規則的粗體標題或節標題）。每段下面依需要寫以下項目，沒有的不寫：

- **統計**：次數、比例、「既有 a／b」、行數、檔數，並註明資料版本，例如 `raw-20261105`
- **決定**：誰在何時決定的，例如「使用者決定 2026-11-07」，或當時的調整檔、改寫檔
- **經過**：舊譯曾經怎麼錯、哪次匯入怎麼處理
- **出處**：例句的 key、研究報告路徑

被刪掉的原指引條文不搬過來，git 歷史可查。

## 不送給譯者

- 依據檔放在 `guides/` 以外；批次的 `guides` 只列 `guides/routing.yaml` 選出的 `guides/` 底下的檔，不會帶到依據檔
- 譯者讀的指引（`base.md`、`families/*.md`、`shared/*.md`、`characters/*.md`）不寫指向這裡的連結，`lct rules lint` 出現 `guides-evidence` 就報錯
- `prompts/translate.md` 與 `translate-pending` 技能不提依據檔

## 維護

- 新增或修改規則時，統計與來源寫進對應的依據檔，不寫進指引。維護規則的技能（`adjust`、`review`、`review-page`、`rules-review`）可以編輯這個目錄
- `docs/research/scripts/family_guides.py` 重算的數字寫進對應的依據檔
- 只改依據檔不觸發 `rebuild.yml`（它只看 `rules/**`、`glossary/**`、`guides/**`、`lct.yaml`），這是預期的：依據不影響翻譯
