# 更新頁：每筆變動的原因，依原因分組

日期：2026-11-08。狀態：使用者已核可（2026-11-08）。

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

1. **KR 改版**：kind 是 `kr-changed`，加上「KR 改版」。若新紀錄是翻譯批次產生的，再加上「翻譯」。
2. **新增／刪除**：kind 是 `new`／`deleted`。
3. **人工調整**：新紀錄 `Origin == human` 且 `Batch == "adjustment:<檔名>"`，原因是該調整檔。
   - 調整的 `reason` 依 `kr_sha` 加 `zh` 對到檔中的那一筆，不依 key，因為 `lct adjust tidy` 之後才會搬 key。
   - 先讀新 Release 的 `rules_commit`，找不到再讀 `prev_rules_commit`。（我的決定）
4. **人工調整失效**：舊紀錄是 `adjustment:<檔名>`，新紀錄不是。原因是「人工調整失效」，加上後來的來源。調整檔讀 `prev_rules_commit` 那一版，因為失效的調整在新版可能已被刪掉。
5. **改寫檔**：
   - 新套用的改寫 = 新 tag 的 `applied.json` 的 `rewrite` 減去舊 tag 的，依檔名順序（同 `rewrite.LoadFiles`）。
   - 從舊譯文開始，依序套用每個檔的 `files`、`origins`、`pattern`、`replace`、`kr_pattern`（同 `rewrite.Apply`，但不含 plan 當下的 `ok` 判斷）。KR 用新 Release 的，origin 用舊紀錄的。
   - **只有在最後結果完全等於新譯文時**，改到文字的檔才記為原因。不相等，代表之後又被重翻、調整或沿用蓋掉，這時不列改寫，交給下面的步驟判斷，以免把被蓋掉的改寫算進影響範圍。（我的決定）
6. **重翻申請**：申請生效後，單位要等到之後的建置才翻得完，可能跨好幾個 Release，所以不能只看兩個 tag 之間新套用的申請。判斷方式：
   - **(a)** 單位在舊 tag 的 `.meta/pending.jsonl` 裡，原因是 `requeue`，且新紀錄是翻譯產生的。用新 tag 的 `applied.json` 中**所有**已套用的申請比對 `keys`／`files`／`origins`，符合的最後一個就是原因。
     - 先比對待譯項目的提示（hint）：plan 把申請的 `reason` 寫成提示，之後的申請或 `lct requeue` 會蓋掉，所以提示指出最後碰過它的申請。提示可能由兩個申請以「；」接起，逐段比對。`reason` 等於提示某一段的申請優先，其中選得到該單位的優先；選不到（例如遊戲更新後 key 改了）也算。沒有提示時才用「符合的最後一個」。原因不是 `requeue`、但提示是某申請的 `reason`（例如申請重設了等人處理的單位），也算該申請。（實作時決定）
     - 提示不是任何申請的 `reason`（workflow 的手動 requeue）：顯示為重翻（`requeue`），不是無法判定。只有在有申請檔讀不到時才標無法判定。（實作時決定）
   - **(b)** 兩個 tag 之間新套用的申請，選到的單位在這段期間被翻譯了。這是申請生效與翻譯落在同一個 Release 的情況。
   - 原因附上翻譯批次。（我的決定）
7. **沿用**：新紀錄 `Batch == "reuse:<key>"`，原因是「沿用同文單位」。
8. **其他重翻**：新紀錄是翻譯產生的，但不符合以上任何一項。若舊 tag 的 `pending.jsonl` 有該單位，依它的原因再細分：`adjustment-expired`、`paratranz-repair`、`structural-fail`、`new`、`kr-changed`。否則就是「重翻（原因不明）」。（我的決定）
9. **其他**：都不符合。照現在的樣子顯示來源。

**資料讀不到時**：例如規則檔在那個 commit 不存在，或 main 的歷史不夠深。該筆不推斷，標記為「無法判定」並附上理由；整份紀錄照常產生，不中斷。

## 資料格式

**`history.Entry` 加 `causes`**：清單，每項 `{kind, id}`。

- `kind` 是 `rewrite`、`adjustment`、`adjustment-expired`、`requeue`、`reuse`、`kr-changed`、`new`、`deleted`、`retranslated`、`other`、`unknown` 其中之一。
- `id` 依 kind 而定：改寫檔 id、調整檔名、申請 id、翻譯批次。
- `retranslated` 另帶 `reason`，取自 `pending.jsonl` 的原因。
- `unknown` 另帶 `note`，說明為什麼無法判定。（我的決定）
- 單位無法判定時，紀錄照常寫出；`Record` 另計 `unknown` 的筆數。

**`history.Record` 加兩個欄位**：
- `rules`：這份紀錄用到的規則索引，id → `{kind, path, title, commit}`。`title` 取規則檔的 `reason`，調整檔則是檔名，以及該檔各筆的 reason（只列被用到的 key）。
- `causes_version`：整數，從 1 起。

**補做舊紀錄**：
- publish 的 `planHistory` 目前只補沒有紀錄的 Release。改成：紀錄存在但 `causes_version` 低於目前版本，也要重做。
- 重做時 Entry 照 `history.Make` 產生（內容不變），再加上原因。
- 每次 publish 都會執行，所以下一次 CI 的 rebuild 或 update 就會補齊 10 筆，寫回 tw。tw 只由 CI 寫入。
- 補齊之後，所有紀錄都已有 `causes_version`，之後 CI 只做當次 Release 的紀錄。
- 若不想等下次遊戲更新，可以在合併後手動觸發一次 rebuild，勾選 `dry_run`：只建置、發布，不翻譯，不花費。

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
  - **摺疊**（使用者要求）：
    - 每組預設收合，只顯示組頭。點組頭展開，展開後沿用現有的前後對照卡片。
    - 檢視頂端有「全部展開／全部收合」。
    - 從卡片的原因標籤跳過來時，自動展開那一組並捲到該處。
    - 展開狀態放在網址參數或記憶體；不必跨次保存。
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
- **真實資料**：在 scratch 對 10 個 Release 補做一次，報告每個 Release 各原因的筆數。先 `git fetch --tags` 取得 `.2`、`.3`。
  - **對照標準**：每個改寫檔推斷出的單位數，要和套用它的那次建置的 `.meta/plan.json` 的 `rewrite_applied[].units` 相符。從 tw 的 git 歷史找出套用該檔的 commit。
  - 不相符的列出原因，例如之後被重翻或調整蓋掉。
  - 調整檔：每個 Release 中 `adjustment` 的筆數，要等於該 Release 新紀錄 `Batch` 為 `adjustment:` 而舊紀錄不是的筆數。
  - 「其他」「重翻（原因不明）」「無法判定」的比例要列出來，供人判斷推斷夠不夠準。
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
