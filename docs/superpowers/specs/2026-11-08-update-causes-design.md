# 更新頁：每筆變動的原因，依原因分組

日期：2026-11-08。狀態：草稿，待使用者核可。

## 目的

網站「更新」頁列出每個 Release 和前一個 Release 之間的變動：新增、KR 改版、刪除、只有譯文改變。現在每筆只顯示前後的來源（paratranz、claude、human），看不出是哪個原因造成的。

使用者要看見每筆變動的原因，尤其是 `rules/*` 造成的：
- 改寫檔 `rules/rewrite/`
- 人工調整 `rules/adjustments/`
- 重翻申請 `rules/requeue/`

並依原因分組，方便確認一條規則改了哪些單位、影響範圍多大。

例：`zh-TW-20261105.3` 有 169 筆「只有譯文改變」，其中 129 筆 paratranz→paratranz、36 筆 →human，目前看不出各來自哪個改寫檔或調整檔。

## 使用者的決定

- **做法 A**：原因在 `lct publish` 產生更新紀錄時推斷，存進 `.meta/history/<tag>.json`。網站只讀取、分組。
- **過去的 Release 也做**：目前 10 筆更新紀錄（`zh-TW-20261029.2`…`zh-TW-20261105.3`）要補上原因。
- **「依原因」檢視包含所有變動**（KR 改版、新增、刪除、其他重翻也算），**`rules/*` 造成的排前面**。

## 做法（已選 A）

| 做法 | 內容 | 取捨 |
|---|---|---|
| **A. 發布時推斷，寫進更新紀錄（選用）** | `lct publish` 做紀錄時算出每筆的原因；缺原因的舊紀錄由 publish 補做一次 | 只算一次，和 Release 綁在一起；網站不變慢。舊紀錄要等 CI 跑一次 publish 才補上 |
| B. 網站產生時推斷 | `lct site` 每次部署重算所有 Release | 不改紀錄格式，但部署變慢、每次重算 |
| C. 套用規則時就記下原因，舊的用 A 補 | plan 時逐單位記錄 | 最精確，但兩套機制；A 的推斷已經夠準 |

## 原因的推斷

**輸入**：前後兩個 Release 的狀態，`history.ReadRev` 已讀到的內容，另外需要：
- 各自 tw 的 `.meta/applied.json`，也就是已套用的改寫檔、重翻申請；
- 紀錄，含 `Batch`；
- 新 Release 的 `rules_commit` 那一版的 `rules/rewrite/*.yaml`、`rules/requeue/*.yaml`、`rules/adjustments/*.yaml`，從 main 的 git 讀取。

**每筆 Entry 可以有多個原因**，依下列順序判斷，全部成立的都列。

1. **KR 改版**：kind 是 `kr-changed`。加上「KR 改版」。若新紀錄是翻譯批次產生的，再加上「翻譯」。
2. **新增／刪除**：kind 是 `new`／`deleted`。
3. **人工調整**：新紀錄 `Origin == human` 且 `Batch == "adjustment:<檔名>"`，原因是該調整檔。
   - 要能顯示該筆調整的 `reason`：同檔中 key 相同的那一筆。
4. **人工調整失效**：舊紀錄是 `adjustment:<檔名>`，新紀錄不是。原因是「人工調整失效」，加上後來的來源。
5. **改寫檔**：
   - 新套用的改寫 = 新 tag 的 `applied.json` 的 `rewrite` 減去舊 tag 的，依檔名順序（同 `rewrite.LoadFiles`）。
   - 從舊譯文開始，依序套用每個檔的 `files`、`origins`、`pattern`、`replace`、`kr_pattern`，規則照 `rewrite.Apply`。KR 用新 Release 的 KR，origin 用舊紀錄的。
   - 改到文字的檔都記為原因。
   - 最後結果等於新譯文時，就完全由改寫解釋。不等時，改寫只解釋一部分，再往下判斷。
6. **重翻申請**：
   - 新套用的申請 = 新 tag `applied.json` 的 `requeue` 減去舊的。
   - 單位符合申請的 `keys`／`files`／`origins`（同 `requeue` 的選擇規則），而且新紀錄是翻譯產生的（`Batch` 和舊紀錄不同，origin 是 claude）。原因是該申請，附上翻譯批次。
7. **沿用**：新紀錄 `Batch == "reuse:<key>"`。原因是「沿用同文單位」。
8. **其他重翻**：新紀錄是翻譯產生的，但不符合以上任何一項。例如格式錯誤後自動重翻，或來源修補。
9. **其他**：都不符合。照現在的樣子顯示來源。

**資料讀不到時**：例如規則檔在那個 commit 不存在，或 main 的歷史不夠深。該筆不推斷，標記為「無法判定」並附上理由；整份紀錄照常產生，不中斷。

## 資料格式

**`history.Entry` 加 `causes`**：清單，每項 `{kind, id}`。

- `kind` 是 `rewrite`、`adjustment`、`adjustment-expired`、`requeue`、`reuse`、`kr-changed`、`new`、`deleted`、`retranslated`、`other`、`unknown` 其中之一。
- `id` 依 kind 而定：改寫檔 id、調整檔名、申請 id、翻譯批次。

**`history.Record` 加兩個欄位**：
- `rules`：這份紀錄用到的規則索引，id → `{kind, path, title, commit}`。`title` 取規則檔的 `reason`，調整檔則是檔名，以及該檔各筆的 reason（只列被用到的 key）。
- `causes_version`：整數，從 1 起。

**補做舊紀錄**：
- publish 的 `planHistory` 目前只補沒有紀錄的 Release。改成：紀錄存在但 `causes_version` 低於目前版本，也要重做。
- 重做時 Entry 照 `history.Make` 產生（內容不變），再加上原因。
- 每次 publish 都會執行，所以下一次 CI 的 rebuild 或 update 就會補齊 10 筆，寫回 tw。tw 只由 CI 寫入。

## 網站

- `ExportUpdates` 把 `causes` 與 `rules` 帶進 `updates/<tag>.json`。
- **Release 頁**：在「卡片／表格」旁加「依原因」檢視。
  - 每個原因一組。組頭顯示：
    - 種類與 id；
    - `title`；
    - 單位數、檔案數；
    - 規則檔路徑與 commit 前 7 碼。網站知道 repo 網址時，連到該 commit 的檔案；不知道就只顯示文字。
  - **排序**：`rules/*`（rewrite、adjustment、requeue、adjustment-expired）在前，其中依單位數多到少；其餘在後，依 KR 改版、新增、刪除、沿用、其他重翻、其他、無法判定的順序。
  - 一筆有多個原因時，出現在每個原因的組裡，並標示它也屬於哪些組。
  - 組可以展開、收合，展開後沿用現有的前後對照卡片。
- **卡片與表格**：每筆加原因標籤，點了跳到「依原因」檢視中的那一組。
- **篩選**：加一項「原因」。
- 舊格式的資料（沒有 `causes`）照現在的樣子顯示，不出錯。補齊之前的部署會遇到。
- 400px 寬時不得整頁橫向捲動，沿用現有樣式變數。

## 非目標

- 不改規則的套用方式，plan 時不逐單位記錄原因（做法 C）。
- 不改 GitHub Release 的說明文字。
- 不推斷 `glossary/`、`guides/` 的影響：它們不直接改既有譯文。

## 驗收

- **測試**：每種原因各有單元測試。另外要涵蓋：
  - 一筆有多個原因，例如兩個改寫檔接連改，或改寫後又被重翻；
  - 規則檔讀不到時標為無法判定；
  - 舊紀錄的 `causes_version` 補做；
  - 網站讀舊格式不出錯。
- **真實資料**：在 scratch 對 10 個 Release 補做一次，報告每個 Release 各原因的筆數。至少確認：
  - `zh-TW-20261105.3` 的 129 筆 paratranz→paratranz 對到 Task 18 的改寫檔；
  - 36 筆 →human 對到 `2026-11-08-glossary-review.yaml` 等調整檔；
  - `.2` 的括號、模組改寫檔各自的單位數，和模擬時的數字相符；
  - 「其他」與「無法判定」的比例要列出來。
- `lct site` 的時間與資料大小要報告，這是數字，不設關卡。
- `go vet ./... && go test ./...`；`cd site && npm test && npm run typecheck && npm run build`。

## 留給實作者

- 原因推斷放在 `internal/history` 還是另立套件。
- 讀取某 commit 規則檔的方式（`gitx`）。
- 版面細節：
  - 組頭的排版；
  - 標籤的顏色（沿用現有 badge 樣式）；
  - 預設檢視維持現在的卡片。
- 網站是否已知道 repo 網址：查 `pages.yml` 與 `meta.json`。不知道就不做連結，不為此新增設定。
