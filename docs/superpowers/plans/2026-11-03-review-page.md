# 審核頁 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在譯文對照網站加上「審核」頁（規則建議、檢查警告、譯者註記、待辦清單），每個項目有 ID 與單位連結；確認過的項目記在 `rules/declined.yaml`／`rules/accepted.yaml`，之後網頁、`lct suggest` 與建置報告都略過。

**Architecture:** 做法 A：`lct site` 匯出時以 `suggest.Run`、建置用的 `validate.Check`、`.meta/notes.jsonl` 與 `build.ReviewItems` 算出審核項目（`internal/site/review.go`），扣掉確認清單後寫成 `data/review/*.json`，並在檔案資料的單位上加 `rv`、`nt`、`pq`。新指令 `lct accept` 用同一套程式檢查 ID 後寫確認清單。建置讀確認清單：不計已確認的警告、不讓已確認的不一致譯法佔前 500 組。前端新增 `#/r/<分頁>` 與檔案檢視的標示。新 skill `review-page` 處理使用者在對話中貼上的 ID。

**Tech Stack:** 沿用：Go 1.26.1；`site/` 的 Preact＋Vite＋TypeScript、Vitest（不加任何依賴）。

**Spec:** `docs/superpowers/specs/2026-11-03-review-page-design.md`（以下稱「spec」）；網站 spec `docs/superpowers/specs/2026-11-03-pages-viewer-design.md`；主 spec `docs/superpowers/specs/2026-10-31-limbus-zh-tw-translation-design.md`（§11.2 規則建議與 `declined.yaml`、§15 指令）。草圖：https://claude.ai/artifact/H7FqjGR434sQyRXJA6wbEe

## Global Constraints

- ID（純文字，照抄）：規則建議沿用主 spec §11.2；警告 `warning/<代碼>/<sha8(KR)>/<sha8(紀錄的 zh)>`；註記 `note/<key>/<sha8(註記文字)>`；`sha8` 是 SHA-256 的前 8 個小寫十六進位字元；`key` 是 `檔名|指標`
- 確認清單：規則建議（含 `note/…`）→ `rules/declined.yaml`；`warning/…` → `rules/accepted.yaml`；兩者格式相同：清單，每筆 `id`（雙引號）、`reason`（必填）、`date`（`YYYY-MM-DD`）
- 網站資料 `format: 2`；新鍵照抄：`meta.review.{suggestions,warnings,notes,todo}`；單位 `rv`、`nt`、`pq`；`review/suggestions.json`、`review/warnings.json`、`review/notes.json`、`review/todo.json`（欄位見 spec §7）
- 待辦分頁只有五種節：`needs_review`、`frozen_changed`、`frozen_fallback`、`loop_error`、`not_translated`
- 路由：`#/r/<suggestions|warnings|notes|todo>?family=…&kind=…&q=…&all=1&id=…`；`#/r` 等於 `#/r/suggestions`
- 沿用網站的限制：不用 `innerHTML`／`dangerouslySetInnerHTML`、不用外部 CDN、不加分析工具、各語言欄的 `lang`（`ko`、`ja`、`en`、`zh-Hant-TW`）、網站名稱與頁面不得使用「Claude Code」「Anthropic」
- JSON 以 UTF-8、無 BOM、無縮排輸出；同樣的輸入產生位元組相同的輸出
- 每個 CLI 指令支援 `--json`；exit code 0 成功、1 使用者錯誤、2 系統錯誤
- skill 與 `prompts/` 內不寫模型名稱；skill 只能用 `bin/lct` 指令、`go build -o bin/lct ./cmd/lct`、`Read`、`Edit`
- 提交前 `go vet ./... && go test ./...`；動到 `site/` 的任務另跑 `cd site && npm ci && npm run typecheck && npm test && npm run build`
- commit 訊息結尾：
  ```
  Co-Authored-By: <your session's attribution line>
  Claude-Session: https://claude.ai/code/session_015XrjRvHe6FfxjemSS1js35
  ```
- 開發分支 `claude/festive-wright-g2k853`；每個任務結束 `git push -u origin claude/festive-wright-g2k853`；`main` 只在 Task 11 經使用者同意後以 fast-forward 更新

## Review Focus

1. **網頁版本比本機舊或新**：使用者貼上的 ID 不是本機目前的項目 → `lct accept` 整批拒絕、什麼都不寫、exit 1 並列出這些 ID — Task 6 `TestAcceptRejectsUnknownIDs`
2. **同一個 ID 重複**（一次給兩遍、或已在清單裡）→ 只寫一次，已有的列在 `skipped` — Task 6 `TestAcceptSkipsKnownAndDuplicates`
3. **含特殊字元的 ID 當深連結**（`|`、`/`、韓文、空白、`?`、`&`）→ `#/r/notes?id=…` 來回不變 — Task 7 `review route round-trips odd ids`
4. **很長的清單**（1,000 則註記、500 組各有上百個單位的建議）→ 頁面只渲染前 100 項，按鈕再顯示 100 項；深連結的目標在範圍內 — Task 8 `long lists render 100 at a time`
5. **複製被瀏覽器拒絕** → 改為選取文字並提示，不拋錯 — Task 8 `copy falls back to selecting`

---

### Task 1: 項目 ID 與確認清單

**Files:**
- Create: `internal/itemid/itemid.go`、`internal/itemid/itemid_test.go`
- Create: `internal/suggest/accepted.go`、`internal/suggest/accepted_test.go`、`rules/accepted.yaml`
- Modify: `internal/suggest/suggest.go`（`inconsistent`、`note` 的 ID 改用 `itemid`）、`internal/suggest/declined.go`（共用的讀寫）、`cmd/lct/rules.go`（`rules lint` 加 `LintAccepted`）

**Interfaces:**
- Produces（`internal/itemid`，不依賴任何專案套件，build 與 suggest 都能用）：
  - `func Sha8(text string) string`
  - `func Inconsistent(family, field, kr string) string` → `"inconsistent/" + family + "/" + field + "/" + Sha8(kr)`
  - `func Warning(code, kr, zh string) string` → `"warning/" + code + "/" + Sha8(kr) + "/" + Sha8(zh)`
  - `func Note(key, note string) string` → `"note/" + key + "/" + Sha8(note)`
- Produces（`internal/suggest`）：
  - `const AcceptedFile = "rules/accepted.yaml"`
  - `func LoadAccepted(root string) ([]Declined, error)`（與 `LoadDeclined` 同一個型別與規則；檔案不存在時為空）
  - `func LintAccepted(root string) []rulefile.Issue`：`id` 符合 `^warning/[a-z0-9-]+/[0-9a-f]{8}/[0-9a-f]{8}$`、`reason` 非空、`date` 格式、`id` 不重複
  - `func AppendEntries(root, file string, entries []Declined) error`：接在檔案最後（保留既有內容與註解），`id` 以雙引號包起來，欄位順序 `id`、`reason`、`date`；檔案不存在時先寫說明註解

- [ ] **Step 1: 寫失敗的測試**
  - `itemid_test.go`：`Sha8("") == "e3b0c442"`；`Warning("kana", "시 히스클리프", "し協會 希斯克利夫")` 以 `"warning/kana/"` 開頭、共 4 段；`Note("A.json|/dataList/0/content", "x")` 是 `"note/A.json|/dataList/0/content/" + Sha8("x")`
  - `suggest` 既有的 ID 測試不變（重構不改 ID）
  - `accepted_test.go`：讀寫來回（`AppendEntries` 後 `LoadAccepted` 得到同樣的條目、原有註解還在）；`LintAccepted` 指出 `id` 格式錯（`warning/kana/xyz`）、`reason` 空、`date` 錯、重複
  - `cmd/lct/rules_test.go`：`rules lint` 對壞的 `accepted.yaml` exit 1

- [ ] **Step 2: 執行測試確認失敗**

Run: `go test ./internal/itemid/ ./internal/suggest/ ./cmd/lct/ -run 'Sha8|Warning|Note|Accepted|RulesLint'`
Expected: FAIL（套件與函式不存在）

- [ ] **Step 3: 實作**（`rules/accepted.yaml` 只有開頭的說明註解，比照 `declined.yaml`：用途、格式、例子、何時失效——譯文改了 ID 就變）

- [ ] **Step 4: 執行測試確認通過**

Run: `go vet ./... && go test ./...`
Expected: PASS

- [ ] **Step 5: Commit 並 push**

```bash
git add internal/itemid internal/suggest cmd/lct/rules.go cmd/lct/rules_test.go rules/accepted.yaml
git commit -m "feat: item ids and rules/accepted.yaml for confirmed check warnings"
git push -u origin claude/festive-wright-g2k853
```

### Task 2: 建置略過確認過的項目；待辦項目帶 key

**Files:**
- Modify: `internal/build/build.go`（`Options`、警告計數、不一致譯法的挑選）、`internal/build/issue.go`（`ReviewSection.Keys`）、`cmd/lct/build.go`（讀兩個清單）
- Test: `internal/build/build_test.go`、`internal/build/issue_test.go`、`cmd/lct/review_test.go`（`lct review --json` 多出 `keys`）

**Interfaces:**
- Consumes: Task 1 `itemid.Warning`、`itemid.Inconsistent`、`suggest.LoadDeclined`、`suggest.LoadAccepted`
- Produces:
  - `build.Options.Declined map[string]bool`、`build.Options.Accepted map[string]bool`（ID 的集合；nil 等於空）
  - `ReviewSection.Keys []string \`json:"keys,omitempty"\``：與 `Items` 等長，第 i 項涉及單位時是它的 key，否則 `""`；`needs_review`、`frozen_changed`、`frozen_fallback` 三種填上

- [ ] **Step 1: 寫失敗的測試**
  - `accepted warning not counted`：fixture 中一個 `kana` 警告的單位，`Options.Accepted` 含它的 `itemid.Warning(code, KR, rec.ZH)` → `Report.Warnings["kana"]` 少 1、`WarningSamples` 不含它的 key；譯文不同（ID 不同）時照常計數
  - `declined group gives up its place`：兩組不一致譯法，`Options.Declined` 含第一組的 `itemid.Inconsistent(...)` → `Report.Inconsistent` 與 `InconsistentAll` 都不含它
  - `review sections carry keys`：`ReviewItems` 的 `needs_review` 節 `Keys[i]` 等於該待譯項目的 key；`loop_error` 節沒有 `keys`

- [ ] **Step 2: 執行測試確認失敗**

Run: `go test ./internal/build/ ./cmd/lct/ -run 'Accepted|Declined|Keys|Review'`
Expected: FAIL

- [ ] **Step 3: 實作**（挑選前 500 組時先略過 `Declined` 的 ID，再排序取前 20／前 500；`cmd/lct/build.go` 從 `ctx.Root` 讀兩個清單轉成集合）

- [ ] **Step 4: 執行測試確認通過**

Run: `go vet ./... && go test ./...`
Expected: PASS

- [ ] **Step 5: Commit 並 push**

```bash
git add internal/build cmd/lct/build.go cmd/lct/review_test.go
git commit -m "feat(build): confirmed warnings and declined groups no longer count"
git push -u origin claude/festive-wright-g2k853
```

### Task 3: `internal/site` 審核項目（一）：規則建議與檢查警告

**Files:**
- Create: `internal/site/review.go`、`internal/site/review_test.go`
- Modify: `internal/site/file.go`（`Inputs.Review *ReviewInputs`）

**Interfaces:**
- Consumes: `suggest.Run`、`suggest.Inputs`、`suggest.Evidence`、`suggest.Declined`；`validate.Check`；`fields.Rules`（`Classify`）；Task 1 `itemid`
- Produces（`internal/site`）：

```go
type ReviewInputs struct {
	RawTag      string
	Build       *build.Info   // nil: no build record
	Plan        *plan.Summary // nil: no plan of RawTag
	Adjustments []adjust.Adjustment
	Fields      *fields.Rules
	Frozen      []string // lct.yaml frozen
	Notes       []store.Note
	Pending     []store.PendingItem
	Declined    []suggest.Declined
	Accepted    []suggest.Declined
}
type Confirmed struct{ Reason, Date string }            // JSON: reason, date
type Variant struct{ ZH string; Count int; Keys []string } // JSON: zh, count, keys
type SuggestionItem struct { ID, Kind, Title, Family, Field, KR string; JP, EN *string; Variants []Variant;
	Evidence []suggest.Evidence; Pending int; Declined *Confirmed }
type WarningItem struct { ID, Code, KR, ZH, Message string; Keys, Families, Origins []string; Pending int; Accepted *Confirmed }
type Review struct {
	Suggestions []SuggestionItem
	Warnings    []WarningItem
	Notes       []NoteItem     // Task 4
	Todo        []TodoSection  // Task 4
	Mismatches  []string       // inconsistent ids whose units do not add up to the evidence
	NotesUnplaced int          // Task 4
	Refs     map[string][]string // unit key → unconfirmed item ids, sorted (rv)
	NoteText map[string]string   // unit key → nt (Task 4)
	Queued   map[string]bool     // unit key → in the pending list (pq) (Task 4)
}
func BuildReview(in Inputs, files []string) (Review, error) // in.Review must be set
```

JSON 鍵名照 spec §7.3、§7.4（`id`、`kind`、`title`、`family`、`field`、`kr`、`jp`、`en`、`variants`、`evidence`、`pending`、`declined`；`code`、`zh`、`message`、`keys`、`families`、`origins`、`accepted`；空的可省略鍵照 spec 的例子省略）。

- 規則建議：`suggest.Run`（輸入同 `lct suggest`）的全部結果，順序不變；`declined.yaml` 有的帶 `Declined`
- `inconsistent` 的 `Variants`：走過每個檔案（`BuildFile`）的單位，`O` 是 `paratranz`、`claude`、`human` 的單位依 `(檔案族群, 欄位, KR)` 分組、組內依 `zh` 分組，`itemid.Inconsistent(...)` 對上建議的 ID；`Variants` 依單位數由多到少（相同時依第一個 key），`Keys` 依檔案順序再依單位順序；某種譯法的單位數不等於證據的 `count` 時，ID 加進 `Mismatches`（不是錯誤）
- 檢查警告：與建置相同的選法——單位的欄位依 `Fields.Classify` 要翻譯、有紀錄且可用（`Frozen` 符合，或紀錄的 `kr_sha` 等於 `units.KRSha(KR)`）、紀錄的 `origin` 不是 `copy`；`validate.Check(KR, rec.ZH, validate.Context{Existing: true, File: 檔名})` 中 `SeverityWarn` 的問題；一個單位同一代碼只算一次；依 `itemid.Warning(code, KR, rec.ZH)` 合成項目；`accepted.yaml` 有的帶 `Accepted`；項目依代碼、再依第一個單位排列
- `Refs`：每個單位收它所屬的**未確認**的 `inconsistent` 與 `warning` ID

- [ ] **Step 1: 寫失敗的測試**（以 `internal/site` 既有的 fixture 方式建一組小資料：同一 KR 兩種譯法的三個單位、一個 `kana` 警告出現在兩個單位、一個 `copy` 單位帶假名）
  - `suggestions match lct suggest`：`Suggestions` 的 ID 順序等於同輸入 `suggest.Run` 的順序
  - `inconsistent lists every unit`：該組 `Variants` 為 `[{zh:A,count:2,keys:[k1,k3]},{zh:B,count:1,keys:[k2]}]`，`Mismatches` 為空
  - `routing changed is not fatal`：把其中一個檔改到別的族群 → 不回傳錯誤，ID 在 `Mismatches`
  - `warnings like the build`：兩個單位合成一個 `kana` 項目、`Keys` 兩個；`copy` 單位不列；紀錄的 `kr_sha` 與 KR 不符的單位不列（與建置相同；和建置報告的總數比對在 Task 5 的 e2e）
  - `accepted warning flagged, not in refs`：`Accepted` 含該 ID → 項目帶 `Accepted`、`Refs` 不含它；譯文改了 → 新 ID、沒有 `Accepted`
  - `declined suggestion flagged, not in refs`

- [ ] **Step 2: 執行測試確認失敗**

Run: `go test ./internal/site/ -run Review`
Expected: FAIL

- [ ] **Step 3: 實作 `BuildReview`（本任務先填 `Suggestions`、`Warnings`、`Mismatches`、`Refs`；其餘欄位留空，Task 4 補上）**

- [ ] **Step 4: 執行測試確認通過**

Run: `go vet ./... && go test ./...`
Expected: PASS

- [ ] **Step 5: Commit 並 push**

```bash
git add internal/site
git commit -m "feat(site): the review items of rule suggestions and check warnings"
git push -u origin claude/festive-wright-g2k853
```

### Task 4: `internal/site` 審核項目（二）：譯者註記、待辦清單、待譯清單

**Files:**
- Modify: `internal/site/review.go`、`internal/site/review_test.go`

**Interfaces:**
- Consumes: Task 3 `ReviewInputs`、`Review`；Task 2 `ReviewSection.Keys`、`build.ReviewItems`
- Produces:

```go
type NoteItem struct { ID, Key, Family, Batch, Note, KR, ZH, O string; Pending int; Declined *Confirmed }
type TodoItem struct{ Text, Key string }                        // JSON: text, key (omitempty)
type TodoSection struct{ Kind, Title, Hint string; Items []TodoItem } // JSON: kind, title, hint, items
```

- 註記放到單位（spec §5.3）：註記的 key 上的紀錄 `batch` 與 `kr_sha` 都相同 → 這個單位；否則同一檔裡 `batch` 與 `kr_sha` 都相同的紀錄恰好一筆 → 那個單位（`NoteItem.Key` 是這個單位的 key，ID 仍以 `notes.jsonl` 的 key 計算，與 `suggest` 的 `note/…` 相同）；否則 `NotesUnplaced++`
- `NoteItem` 依檔案順序、單位順序、`batch` 排列；`declined.yaml` 有的帶 `Declined`
- `NoteText[key]`：該單位所有註記（含確認過的）以 `"\n"` 連接；`Refs` 加上未確認的註記 ID
- `Todo`：`build.ReviewItems(*Build, Plan, Pending)` 中五種節（Global Constraints），項目文字照抄，`Key` 取 `ReviewSection.Keys`；`Build` 為 nil 時 `Todo` 為空
- `Queued`：待譯清單（`Pending`）中所有的 key；各項目的 `Pending` 是它的單位中在 `Queued` 的數量

- [ ] **Step 1: 寫失敗的測試**
  - `note on its unit`、`note follows a moved record`（key 上已是別的紀錄、同檔另一指標有相同 `batch`＋`kr_sha` → 放到那裡）、`note with no place`（兩筆候選或沒有 → `NotesUnplaced == 1`）、`note of an older translation`（`batch` 不同 → 不列）
  - `note text kept when declined`：確認過的註記仍在 `NoteText`、不在 `Refs`
  - `todo has five kinds`：`Plan` 有新說話者、`Pending` 有一個 `needs_review` → `Todo` 只有 `needs_review` 一節，項目 `Key` 是該 key
  - `pending counted`：`Pending` 含某警告的兩個單位之一 → 該項目 `Pending == 1`、`Queued[k] == true`

- [ ] **Step 2: 執行測試確認失敗**

Run: `go test ./internal/site/ -run Review`
Expected: FAIL

- [ ] **Step 3: 實作**

- [ ] **Step 4: 執行測試確認通過**

Run: `go vet ./... && go test ./...`
Expected: PASS

- [ ] **Step 5: Commit 並 push**

```bash
git add internal/site
git commit -m "feat(site): the review items of translator notes, the to-do list and the pending list"
git push -u origin claude/festive-wright-g2k853
```

### Task 5: 匯出審核資料（`format: 2`）與 `lct site` 接線

**Files:**
- Modify: `internal/site/export.go`、`internal/site/file.go`（`Unit` 加 `RV []string \`json:"rv,omitempty"\``、`NT string \`json:"nt,omitempty"\``、`PQ int \`json:"pq,omitempty"\``）、`internal/site/export_test.go`
- Modify: `cmd/lct/site.go`（讀 `ReviewInputs`；抽出 `loadReviewInputs(ctx *cli.Context, cmd string) (site.ReviewInputs, error)` 給 Task 6 用）、`cmd/lct/site_test.go`
- Test: `cmd/lct/e2e_test.go`（或既有的 e2e 檔）加一個交叉檢查

**Interfaces:**
- Consumes: Task 3、4 `BuildReview`、`Review`
- Produces:
  - `FormatVersion = 2`；`Meta.Review ReviewCounts \`json:"review"\``，`ReviewCounts{Suggestions, Warnings, Notes, Todo int}`（JSON 小寫鍵，只算未確認的；`Todo` 是項目數）
  - `Export` 在寫檔案資料前先 `BuildReview`，單位帶 `rv`（`Refs`）、`nt`（`NoteText`）、`pq`（`Queued` 為 1）
  - 寫 `review/suggestions.json`、`review/warnings.json`、`review/notes.json`（`{"items": [...]}`）與 `review/todo.json`（`{"sections": [...]}`）
  - `Result` 加 `Review ReviewCounts \`json:"review"\``、`ReviewMismatches []string \`json:"review_mismatches"\``、`NotesUnplaced int \`json:"notes_unplaced"\``
  - `Inputs.Review == nil` 時照樣寫四個檔（空清單）、`meta.review` 全為 0

- [ ] **Step 1: 寫失敗的測試**
  - `export_test.go`：`format` 為 2；四個 `review/` 檔存在且鍵名照 spec §7；有 `rv` 的單位與 `Refs` 一致；`meta.review` 等於未確認的數量；兩次匯出位元組相同；`Inputs.Review == nil` 時四個檔是空清單
  - `site_test.go`：`lct site --json` 的結果有 `review`、`review_mismatches`、`notes_unplaced`
  - e2e `TestSiteWarningsMatchBuild`：跑完 `lct build` 再跑 `lct site`，`review/warnings.json` 各代碼 `keys` 總數等於 `build.json` 的 `report.warnings`

- [ ] **Step 2: 執行測試確認失敗**

Run: `go test ./internal/site/ ./cmd/lct/ -run 'Export|Site'`
Expected: FAIL

- [ ] **Step 3: 實作**（`BuildFile` 走兩遍可以接受；實際資料上 `lct site` 整體要在 60 秒內，Task 11 量）

- [ ] **Step 4: 執行測試確認通過**

Run: `go vet ./... && go test ./...`
Expected: PASS

- [ ] **Step 5: Commit 並 push**

```bash
git add internal/site cmd/lct
git commit -m "feat(site): lct site writes the review data, format 2"
git push -u origin claude/festive-wright-g2k853
```

### Task 6: `lct accept`

**Files:**
- Create: `cmd/lct/accept.go`、`cmd/lct/accept_test.go`

**Interfaces:**
- Consumes: Task 5 `loadReviewInputs`、`site.BuildReview`；Task 1 `suggest.AppendEntries`、`suggest.DeclinedFile`、`suggest.AcceptedFile`
- Produces: 指令 `accept`，Usage 照 spec §6.3：
  - `lct accept --reason R [--date YYYY-MM-DD] [--file F] [ID...]`：ID 來自參數與 `--file`（一行一個，空行與 `#` 開頭略過）的聯集，依出現順序去重
  - `lct accept --tidy`：刪掉 `accepted.yaml` 中已不是任何目前警告項目的條目
  - `--json`：`{"declined": [...], "accepted": [...], "skipped": [...], "raw_tag": "...", "tw_commit": "..."}`；`--tidy` 為 `{"removed": [...]}`

規則：規則建議的 ID（`Review.Suggestions` 全部，含確認過的）與 `note/…`（`Review.Notes` 全部）→ `declined.yaml`；`warning/…`（`Review.Warnings` 全部）→ `accepted.yaml`；已在清單的 → `skipped`；任何一個 ID 不在以上 → exit 1、什麼都不寫、訊息列出這些 ID；沒有 ID 且沒有 `--tidy`、`--reason` 空、`--date` 格式錯 → exit 1。不 commit。

- [ ] **Step 1: 寫失敗的測試**（沿用 `cmd/lct` 既有的 e2e repo 搭法）
  - `TestAcceptWritesBothLists`：一個 `inconsistent/…`、一個 `note/…`、一個 `warning/…` → 前兩個進 `declined.yaml`、第三個進 `accepted.yaml`，`reason`、`date` 照給的；之後 `lct suggest --json` 不再列前兩個、`lct site` 的 `meta.review` 各少 1
  - `TestAcceptRejectsUnknownIDs`：一個有效、一個 `warning/kana/00000000/00000000` → exit 1，兩個檔的位元組不變，訊息含該 ID
  - `TestAcceptSkipsKnownAndDuplicates`：同一 ID 給兩次＋一個已在清單的 → 寫一筆、`skipped` 列出已有的
  - `TestAcceptFile`：`--file` 的空行、`#` 行
  - `TestAcceptTidy`：`accepted.yaml` 有一筆對不上任何警告 → `--tidy` 刪掉它、`removed` 列出
  - `TestAcceptUsage`：沒有 ID、空 `--reason`、壞 `--date` → exit 1

- [ ] **Step 2: 執行測試確認失敗**

Run: `go test ./cmd/lct/ -run Accept`
Expected: FAIL

- [ ] **Step 3: 實作**

- [ ] **Step 4: 執行測試確認通過**

Run: `go vet ./... && go test ./...`
Expected: PASS

- [ ] **Step 5: Commit 並 push**

```bash
git add cmd/lct/accept.go cmd/lct/accept_test.go
git commit -m "feat: lct accept records the review items a person confirmed"
git push -u origin claude/festive-wright-g2k853
```

### Task 7: 前端資料層：型別、載入、路由、頁首、fixtures

**Files:**
- Modify: `site/src/types.ts`、`site/src/data.ts`（`FORMAT = 2`）、`site/src/route.ts`、`site/src/settings.ts`（`reviewOnly`）、`site/src/app/Header.tsx`、`site/src/app/App.tsx`（`review` 路由交給 Task 8 的元件，這裡先放佔位）
- Modify: `site/fixtures/data/meta.json`、`site/fixtures/data/files/**`（加 `rv`、`nt`、`pq`）
- Create: `site/fixtures/data/review/{suggestions,warnings,notes,todo}.json`
- Test: `site/src/route.test.ts`、`site/src/data.test.ts`、`site/src/fixtures.test.ts`、`site/src/settings.test.ts`、`site/src/app/App.test.tsx`

**Interfaces:**
- Produces（TypeScript，鍵名照 spec §7）：
  - `Meta.review: ReviewCounts`（`{suggestions, warnings, notes, todo: number}`）；`Unit.rv?: string[]`、`Unit.nt?: string`、`Unit.pq?: 1`
  - `SuggestionItem`、`Variant`、`Evidence`、`WarningItem`、`NoteItem`、`TodoSection`、`TodoItem`、`Confirmed`
  - `type ReviewTab = 'suggestions' | 'warnings' | 'notes' | 'todo'`
  - `Route` 加 `{ kind: 'review'; tab: ReviewTab; family: string; sub: string; q: string; all: boolean; id?: string }`（`sub` 是查詢字串的 `kind`：建議的種類、警告的代碼）
  - `function reviewHref(tab: ReviewTab, opts?: { family?: string; sub?: string; q?: string; all?: boolean; id?: string }): string`
  - `function loadReview<T extends ReviewTab>(tab: T): Promise<ReviewData[T]>`（`review/<tab>.json`，與 `loadFile` 相同的快取與錯誤處理）
  - `Settings.reviewOnly: boolean`（預設 `false`）
  - 頁首「審核」連結到 `reviewHref('suggestions')`，旁邊顯示 `meta.review` 四項的總和（0 時不顯示數字）

- [ ] **Step 1: 寫失敗的測試**
  - `review route round-trips odd ids`：`parseHash(reviewHref('notes', { id: 'note/StoryData/S1.json|/dataList/0/content/1a2b3c4d', q: '協會 & ?' }))` 得到同樣的 `tab`、`id`、`q`；`#/r` → `tab: 'suggestions'`；`#/r/bogus` → 首頁
  - `format 1 asks for reload`：`meta.json` 的 `format` 為 1 → `DataError('format')`
  - `fixtures are consistent`：每個 `rv` 的 ID 都在 `review/` 某個檔、且沒有 `declined`／`accepted`；`meta.review` 等於各檔未確認的數量；`nt` 的單位在 `notes.json` 有對應項目
  - `header shows the review count`：fixture 的總數出現在「審核」旁

- [ ] **Step 2: 執行測試確認失敗**

Run: `cd site && npm test`
Expected: FAIL

- [ ] **Step 3: 實作**（fixtures 手寫，照 spec §7，涵蓋：一組 `inconsistent` 兩種譯法、一個非 `inconsistent` 建議、一個確認過的建議、兩個單位的 `kana` 警告、一個確認過的警告、兩則註記其中一則確認過、一節 `needs_review` 待辦、一個 `pq` 單位）

- [ ] **Step 4: 執行測試確認通過**

Run: `cd site && npm run typecheck && npm test && npm run build`
Expected: PASS

- [ ] **Step 5: Commit 並 push**

```bash
git add site
git commit -m "feat(site): review data types, loading, route and header link"
git push -u origin claude/festive-wright-g2k853
```

### Task 8: 前端審核頁

**Files:**
- Create: `site/src/review/ReviewView.tsx`、`SuggestionCard.tsx`、`WarningCard.tsx`、`NoteCard.tsx`、`TodoView.tsx`、`diff.ts`、`marks.ts`、`copy.ts`、`names.ts`、`filter.ts`
- Modify: `site/src/tmp/TmpText.tsx`（`mark?: RegExp`：只在文字節點內把符合的字元包成 `<mark>`）、`site/src/app/App.tsx`、`site/src/styles.css`
- Test: `site/src/review/diff.test.ts`、`marks.test.ts`、`copy.test.ts`、`filter.test.ts`、`ReviewView.test.tsx`

**Interfaces:**
- Consumes: Task 7 的型別、`loadReview`、`reviewHref`、`fileHref`；`stripTags`（`src/search/match.ts`）；`TmpText`
- Produces:
  - `type Piece = { text: string; tok: boolean; hl: boolean; same: boolean }`；`function variantDiff(base: string, other: string): Piece[]`（`[ID]` 一個單位、其他每個字元一個單位；`<style="highlight">` 內外不同算不同；最長共同子序列）
  - `const WARNING_MARKS: Partial<Record<string, RegExp>>`：`kana` `/[ぁ-ゖァ-ヺ]/g`、`halfwidth-punct` `/[,!?:;]/g`、`halfwidth-bracket` `/[()]/g`、`ascii-ellipsis` `/\.{2,}/g`、`curly-quotes` `/[‘’“”]/g`
  - `const WARNING_NAMES: Record<string, { name: string; desc: string }>`：`kana` 假名殘留／TW 裡有日文假名：可能是漏譯，也可能是刻意保留的讀音；`halfwidth-punct` 半形標點／應為全形；`ascii-ellipsis` 英文省略號／應為「……」；`length-ratio` 長度差太多／可能漏譯或多譯；`halfwidth-bracket` 半形括號／應為全形（ ）；`curly-quotes` 彎引號／應為「」『』；`hangul` 韓文殘留；`simplified` 簡體字；`forbidden-term` 禁用譯法；其他代碼顯示代碼本身與 `message`
  - `function copyText(id: string, excerptSource: string): string` → `` `${id}「${excerpt}」→ ` ``，`excerpt` 是 `stripTags` 後空白合一、最多 40 字（超過加「…」）
  - `function copyToClipboard(text: string, fallbackEl: HTMLElement): Promise<'copied' | 'selected'>`
  - `ReviewView({ route, index, meta })`：spec §8.2、§8.3；每頁先顯示 100 項，按鈕「再顯示 100 項（共 N）」；`id` 指定的項目一定在顯示範圍內並捲到它、高亮；「顯示已確認的 N 項」開關（`all=1`）；篩選與開關寫回網址（`reviewHref`）
  - 頁首的使用說明（照抄）：「按「複製 ID」，貼到對話中再加上你的決定；Claude 會改規則或把項目記為確認過，下次建置後這頁就跟著更新。」
  - 項目的 `pending` 大於 0 時標「待重翻 N」，等於單位數時標「等待重翻」
  - 譯者註記分頁的文字篩選旁有快捷鍵：新詞、諧音、俗語、沿用、標籤（按下就把該詞填進文字篩選）
  - 待辦分頁說明（照抄）：「新說話者、新欄位、新檔案族群與過期的調整列在「規則建議」分頁，可以在那裡確認。」沒有項目時顯示「這次建置沒有待辦項目」

- [ ] **Step 1: 寫失敗的測試**
  - `diff.test.ts`：`variantDiff('使目標增加<style="highlight">2</style>回 [Laceration]', '使目標增加<style="highlight">2回</style> [Laceration]')` 中只有「回」的 `same` 為 false；尾端多一個空白 → 該空白 `same: false`；`[Laceration]` 是一個 `tok` 單位
  - `marks.test.ts`：`kana` 標出「し協會」的「し」；標記不進入 `<color=…>` 的標籤文字
  - `copy.test.ts`：`copyText('warning/kana/…', '<color=#fff>し協會</color> 希斯克利夫')` 為 `warning/kana/…「し協會 希斯克利夫」→ `；超過 40 字截斷加「…」；**`copy falls back to selecting`**：`navigator.clipboard.writeText` 拒絕 → 回傳 `'selected'`、不拋錯
  - `filter.test.ts`：族群、種類、文字（比對 ID、檔名、KR、TW、註記）、`all`
  - `ReviewView.test.tsx`（fixtures）：四個分頁籤顯示未確認數；`pq` 單位所屬的項目標「待重翻 1」；確認過的預設不顯示、`all=1` 時以淡色顯示並有理由與日期；`inconsistent` 卡片有兩種譯法與「全部 N 個單位」；警告卡片列出兩個單位連結（`fileHref(path, ptr)`）；待辦分頁的項目連到單位；**`long lists render 100 at a time`**：注入 250 則註記 → 100 張卡片，按一次按鈕 → 200；`id` 指到第 180 則 → 該則已渲染且有高亮

- [ ] **Step 2: 執行測試確認失敗**

Run: `cd site && npm test`
Expected: FAIL

- [ ] **Step 3: 實作**（版面照草圖；顏色沿用 `styles.css` 的 token，深色模式一併處理）

- [ ] **Step 4: 執行測試確認通過**

Run: `cd site && npm run typecheck && npm test && npm run build`
Expected: PASS

- [ ] **Step 5: Commit 並 push**

```bash
git add site
git commit -m "feat(site): the review page, with copyable item ids"
git push -u origin claude/festive-wright-g2k853
```

### Task 9: 檔案檢視的標示

**Files:**
- Modify: `site/src/view/Cell.tsx`（TW 欄的標籤與展開）、`site/src/view/TableView.tsx`、`site/src/view/ScriptView.tsx`、`site/src/view/FileView.tsx`（「只看有審核項目的單位」）、`site/src/view/group.ts`（`visibleUnits` 加 `reviewOnly`）、`site/src/styles.css`
- Test: `site/src/view/FileView.test.tsx`、`site/src/view/group.test.ts`

**Interfaces:**
- Consumes: Task 7 `Unit.rv`、`nt`、`pq`、`Settings.reviewOnly`、`loadReview`、`reviewHref`；Task 8 `WARNING_NAMES`
- Produces: `visibleUnits(units, settings)` 在 `settings.reviewOnly` 時只留有 `rv` 的單位（劇本模式中一段沒有就整段隱藏，沿用既有的段落規則）

標籤：`rv` 的 `inconsistent/…` →「不一致」，`warning/<代碼>/…` →「警告：<名稱>」，`note/…` 或有 `nt` →「註記」，`pq` →「待重翻」；有 `rv` 的單位 TW 欄左側一條色線。點標籤在該列下方展開：不一致（從 `loadReview('suggestions')` 取該項，列出各種譯法與數量，連到 `reviewHref('suggestions', { id })`）、警告（名稱與說明，連到該項）、註記（`nt` 全文）；再點收起。

- [ ] **Step 1: 寫失敗的測試**
  - `review chips on units`：fixture 中有 `rv` 的單位顯示對應標籤與色線；只有 `nt`（確認過的註記）的單位只有「註記」標籤、沒有色線
  - `chip opens its detail`：點「不一致」→ 出現兩種譯法與連結；點「註記」→ 出現 `nt` 全文
  - `review only filter`：`reviewOnly` 時表格只剩有 `rv` 的單位；劇本模式中沒有 `rv` 的段落整段隱藏；設定寫入 `localStorage`

- [ ] **Step 2: 執行測試確認失敗**

Run: `cd site && npx vitest run src/view`
Expected: FAIL

- [ ] **Step 3: 實作**

- [ ] **Step 4: 執行測試確認通過**

Run: `cd site && npm run typecheck && npm test && npm run build`
Expected: PASS

- [ ] **Step 5: Commit 並 push**

```bash
git add site
git commit -m "feat(site): review marks in the file view"
git push -u origin claude/festive-wright-g2k853
```

### Task 10: `review-page` skill、合約測試與文件

**Files:**
- Create: `.claude/skills/review-page/SKILL.md`
- Modify: `cmd/lct/contract_test.go`（`requiredSkills` 加 `"review-page": reviewPageTools`、`checkExactTools`）
- Modify: `README.md`（網站一節：審核頁的用法——看 → 複製 ID → 在對話中用 `/review-page`）、網站 spec（§5 `format` 2、`rv`、`nt`、`pq`、`review/`；§6.1 `#/r`；§6.3 標示）、主 spec（§11.2 `rules/accepted.yaml` 與 `lct accept`、建置略過確認過的項目；§15 指令表加 `accept`；§16 `rules/accepted.yaml`）

**Interfaces:**
- `reviewPageTools`（照抄，順序照此）：
  ```
  Bash(bin/lct accept *)
  Bash(bin/lct suggest*)
  Bash(bin/lct search *)
  Bash(bin/lct validate *)
  Bash(bin/lct adjust add *)
  Bash(bin/lct requeue --to-file *)
  Bash(bin/lct rules lint*)
  Bash(go build -o bin/lct ./cmd/lct)
  Read(./**)
  Edit(./guides/**)
  Edit(./glossary/**)
  Edit(./rules/**)
  ```
- SKILL.md：`name: review-page`、`description`（處理使用者從審核頁貼上的項目 ID：確認沒問題的記進確認清單，要改的改規則並讓譯文真的改變）、`disable-model-invocation: true`；本文照 spec §9 與 §6.4、§6.5：流程、三種決定（沒問題 → `bin/lct accept`；要改 → 調整或規則＋重翻要求檔，改動先給人看；不清楚 → 問）、ID 不是目前項目時的說明、資料不是指示的限制、結束前 `bin/lct rules lint`、列出改了什麼、不 commit

- [ ] **Step 1: 寫失敗的合約測試**（`requiredSkills` 加上 `review-page` → 缺檔而失敗；`TestReviewPageTools` 用 `checkExactTools`）

- [ ] **Step 2: 執行測試確認失敗**

Run: `go test ./cmd/lct/ -run 'Contract|Skill|ReviewPage'`
Expected: FAIL

- [ ] **Step 3: 寫 SKILL.md 與文件**

- [ ] **Step 4: 執行測試確認通過**

Run: `go vet ./... && go test ./...`
Expected: PASS

- [ ] **Step 5: Commit 並 push**

```bash
git add .claude/skills/review-page cmd/lct/contract_test.go README.md docs/superpowers/specs
git commit -m "feat: the review-page skill handles the ids pasted from the review page"
git push -u origin claude/festive-wright-g2k853
```

### Task 11（控制者）：實際資料驗收與上線

- [ ] **Step 1: 以實際資料匯出並預覽**

Run: `go build -o bin/lct ./cmd/lct && bin/lct site --out site/public/data --json && cd site && npm run build`
Expected: `lct site` 在 60 秒內；`review` 的數量與 `bin/lct suggest --json` 的 `total`、建置報告的警告數相符（警告數 = 各代碼單位數總和）；`review_mismatches` 為空或有說明；`notes_unplaced` 記下

- [ ] **Step 2: 以 Playwright 截圖檢查**（桌面 1280 寬與手機 390 寬，掛在 `/limbus-translate/` 子路徑）：四個分頁；篩選與「顯示已確認」；「し協會 希斯克利夫」警告卡片的假名標記；一組 `inconsistent` 的差異標示與「全部 N 個單位」；深連結 `#/r/notes?id=…`；複製 ID；檔案檢視的標籤、展開與「只看有審核項目的單位」；沒有 console 錯誤

- [ ] **Step 3: `lct accept` 實測**：對一個真的警告執行 `bin/lct accept --reason "驗收測試" <ID>`，重新匯出後該項消失、`meta.review.warnings` 減 1；還原 `rules/accepted.yaml`

- [ ] **Step 4: 回報使用者並取得同意後**：`main` 以 fast-forward 更新；手動執行一次 `pages.yml`（或等下一次 `update`／`rebuild`），確認審核頁上線
