# 不一致的譯法：連同 JP/EN 的檢視 — 設計（小改動）

> 狀態：使用者已確認（2026-11-04）：保留所有組、在每個譯法旁顯示 JP/EN、加徽章與篩選；審核時（`/review-page`、`/rules-review`）也看得到 JP/EN，但決定一律留給使用者；做完後與尚未合併的詞彙與標點確認一起合併到 `main`。
> 依據：審核頁 spec `docs/superpowers/specs/2026-11-03-review-page-design.md`（§5.1、§7.3、§8.3）。

## 為什麼

不一致的譯法只依 KR 分組。KR 常比 JP、EN 籠統，例如「중지 작은 아우」在 JP／EN 分成「末弟／Brother」與「末妹／Sister」，TW 譯成「幼弟」「幼妹」是對的。實際資料的 500 組中：169 組的 TW 差異完全跟著 JP/EN 走、86 組部分如此、245 組 JP/EN 全同（真正的不一致）。看得到 JP/EN，才能分辨哪些差異是對的。

## 目標、非目標

- 目標：規則建議分頁的不一致譯法卡片多一個檢視「連同 JP/EN」；所有組照常列出，每種 TW 譯法旁顯示它的 JP 與 EN；卡片有徽章（JP/EN 可解釋／部分可解釋／JP/EN 相同），並可依徽章篩選
- 目標：審核時也看得到同樣的 JP/EN 結果：`/review-page` 與 `/rules-review` 查不一致的譯法時，取得每種譯法的 JP/EN 子組與分類
- 非目標：不改 ID、確認清單、`lct suggest` 的輸出、建置；不把 KR+JP+EN 變成新的建議；skill 不因 JP/EN 分類自動確認或不採用任何項目
- 反目標：skill 拿「JP/EN 可解釋」當理由替使用者確認或不採用
- 反目標：切換檢視讓某組消失或改了數字；確認過的組因檢視不同而重新出現

## 做法

只有一個合理的做法：`lct site` 分組時本來就拿著每個單位的 JP 與 EN，所以在 `review/suggestions.json` 的每個譯法下加上 JP/EN 子組，並在組上標出分類。不採用：讓網頁為了 JP/EN 去載入每個單位的檔案（上百個檔，慢）；把 KR+JP+EN 當成新的建議 ID（會改掉 ID，讓既有的確認失效）。

## 資料（`review/suggestions.json`，`format` 不變，只增加鍵）

`inconsistent` 項目：

- 每個 `variants[]` 加 `jpen`：`[{"jp": "…", "en": "…", "count": n, "keys": [...]}, …]`，把該譯法的單位依 (JP, EN) 分組；該語言沒有對應時省略該鍵；依單位數由多到少、相同時依第一個 key（檔案順序、單位順序）；`keys` 照原本的順序。`variants[].keys` 不變
- 組加 `jpen_class`：`"same"`（所有單位的 JP 與 EN 都相同）、`"explained"`（不只一種 (JP, EN)，而且每一種只對到一種 TW）、`"partial"`（其他：至少一種 (JP, EN) 對到兩種以上的 TW）。缺少的語言當作一個值（「沒有」）

## 審核時（skill 與 `lct accept --dry-run`）

- `lct accept --dry-run` 印出的項目就是 `BuildReview` 算出的項目，所以不一致的譯法也帶 `variants[].jpen` 與 `jpen_class`（`--json` 與文字輸出都有；文字輸出在每個譯法下列出 JP/EN 與數量，並印出分類）
- `/review-page`：查不一致的譯法時，把每種譯法的 JP/EN 與分類一起給使用者看；決定一律由使用者做，不因分類自動確認
- `/rules-review`：允許的工具加上 `Bash(bin/lct accept --dry-run *)`（只能查，不能寫；CI 的 `rules-proposal.yml` 給 action 的工具清單同步加上，合約測試兩邊都釘住）；處理不一致的譯法時先用它取得 JP/EN，在說明的證據中列出各譯法的 JP/EN 與分類；不因分類把項目寫進 `rules/declined.yaml`，也不因分類就統一譯法——沒有其他把握的理由時，列進「需要人決定」
- 兩個 skill 的合約測試（`…Says`）釘住「不因 JP/EN 分類替人決定」這一點

## 前端

- 規則建議分頁多一個切換：「只看 KR」（預設，現在的樣子）／「連同 JP/EN」；存在網址的查詢字串 `mode=jpen`
- 「連同 JP/EN」時：
  - 不一致譯法的每一列譯法下列出它的 JP/EN 子組：JP、EN（缺少時顯示「—」）、單位數；一種譯法只有一個子組時，JP/EN 直接接在那一列
  - 卡片顯示徽章：「JP/EN 可解釋」「部分可解釋」「JP/EN 相同」
  - 篩選列多一個「JP/EN」：全部／可解釋／部分可解釋／相同（各附數量），存在 `jpen=explained|partial|same`；只篩不一致譯法，其他種類的建議在篩選時照常顯示（我的決定：其他種類沒有 JP/EN 分類，篩掉會讓人以為它們不見了）
- 「只看 KR」時樣子與現在完全相同；檔案檢視的「不一致」展開也不變
- 各語言文字照舊加 `lang`（JP `ja`、EN `en`）、沿用標籤呈現

## 測試

- Go：`jpen` 子組的分組、排序與省略的鍵；三種 `jpen_class`（含缺少 JP 或 EN 的單位）；輸出位元組穩定；`lct accept --dry-run` 的 JSON 與文字輸出帶 JP/EN 與分類
- 合約：`/rules-review` 的工具清單與 `rules-proposal.yml` 一致並含 dry-run；兩個 skill 的本文要求
- 前端：切換與網址來回；「連同 JP/EN」時譯法旁的 JP/EN 與數量；徽章；篩選與其他種類照常顯示；「只看 KR」與現在相同
- 實際資料：三種分類的數量是 169／86／245（以目前資料）；Playwright 截圖桌面與手機

## 交給實作者決定

- 徽章與子組的版面細節（沿用審核頁的樣式與色彩 token）
