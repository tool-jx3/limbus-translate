# 翻譯預覽 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 用現行規則模擬「這個單位現在單獨送翻」得到的批次與送出檢查結果，提供 `lct batch preview` 指令與網站預覽頁。

**Architecture:** Go 端在 `internal/batch` 新增 `Previewer`，直接呼叫 `assemble` 組批次，並共用從 `Submit` 抽出的判定函式 `Judge`。`lct site` 以 `Previewer.File` 為每個 KR 檔寫一份精簡預覽檔，另寫一份共用檔（提示詞、指引全文）。瀏覽器只「查表組回」批次 JSON，並由跨語言契約測試保證與 Go 寫出的批次檔逐字相同。

**Tech Stack:** Go 1.26（`cmd/lct`、`internal/`），Preact + Vite + Vitest（`site/`）。

**Spec:** `docs/superpowers/specs/2026-11-06-translation-preview-design.md`

## Global Constraints

- 預覽唯讀：不寫租約、嘗試次數、暫定詞、紀錄、notes；CLI 與 site 測試都要驗證 store 檔案前後一字不變。
- 預覽與實際批次共用程式：組批次用 `assemble`，判定用 `Judge`（`Submit` 也改用它）；不得另寫一套。
- 批次 id 固定 `preview`；上下文範圍是 `contextReach(1)`。
- 待譯中的單位用真實待譯項目；不在待譯、有非 copy 紀錄的用 `requeue.Item`（reason `requeue`、prev_zh 為紀錄譯文、無 hint）；兩者皆無用 reason `new`（`batch.ReasonNew`）。copy 紀錄不預覽：CLI 為使用者錯誤（exit 1），網站顯示「copy：識別碼或各語言相同，不送翻」。
- 檢查：每個非 copy 紀錄都檢查。fallback 標「替代文字（EN 或 KR），不是譯文：結果僅供參考」；human 標「人工譯文」。紀錄 kr_sha 與現在 KR 不同仍檢查，並標「這是舊 KR 的譯文」。沒有紀錄寫「沒有譯文可檢查」。note 是同 key、kr_sha 等於現在 KR 的 notes，依 store 順序以 `\n` 合併。
- 狀態：needs_review 標「等人處理，現在不會發出」；租約中標「已租給批次 X」；凍結檔標「不送翻：凍結檔案」；三者批次照樣組出。
- CLI：`lct batch preview <key> [--json]`；exit 0 成功、1 使用者錯誤（key 不在 KR、copy）、2 系統錯誤；`--json` 輸出 `{batch, state, pending, record, notes, check}`。
- CLI 與 `lct site` 用同一個載入函式 `openPreviewEnv` 取得 `batch.Env`（含 `KeywordNames`、`ForeignNames`、`Provisional`、`RulesCommit`）與 frozen 清單。
- 網站批次 JSON 與 `writeJSONFile` 逐字相同：兩格縮排、不跳脫 `<` `>` `&`、結尾換行、鍵序依 Go 結構、map 鍵排序；Go 會把 U+2028、U+2029 跳脫成 ` `、` `，`JSON.stringify` 不會。
- 網站：路由 `#/pv/<path>?u=<pointer>`（沿用檔案頁的 `u` 參數名，見 Ruling P1）；頁首寫「依規則 `<前 7 碼>` 模擬」；一個單位只載入該檔 `files/<path>`、`preview/<path>` 與 `preview.json`；400px 寬時整頁不得橫向捲動；沿用 `styles.css` 既有變數。
- 預算（回報，不是關卡）：預覽資料總大小與 `lct site` 多花的時間；預估 ≤ 60 MB、≤ 50%。
- 提交前：`go vet ./... && go test ./...`；動到 `site/` 時另跑 `cd site && npm test && npm run typecheck && npm run build`。
- Commit 訊息用繁體中文，結尾兩行：`Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`、`Claude-Session: https://claude.ai/code/session_015XrjRvHe6FfxjemSS1js35`。程式、文件、skill 不寫模型名稱。

## Rulings（計畫層，對規格的細化）

- P1：規格寫 `?ptr=`，檔案頁既有路由用 `?u=<pointer>`；為一致改用 `u`。代價：無。
- P2：規格說頁首用 meta 的 `rules_commit`；那是建置包時的規則 commit，可能落後 `lct site` 當下載入的規則。預覽改用 `preview.json` 自帶、由 `openPreviewEnv` 算出的 commit（與 `batch next/submit` 同一個 `rulesCommit`）。代價：兩個 commit 不同時頁面與 meta 顯示不同值，但預覽的才是模擬所用的。
- P3：`lct site` 一次處理一個檔案，`Previewer` 不跨檔快取檔案資料（記憶體）；CLI 的 `Unit` 與 site 的 `File` 共用同一段每單位程式。

## Review Focus

1. 預覽與 `batch next` 的實際批次分岔：同一個單獨待譯單位，`Previewer.Unit` 的批次（除 id）必須等於 `Next` 切出的批次——Task 2 有等價測試。
2. 上下文排除仍在待譯的鄰行：預覽的 `pending` 必須含整個真實待譯清單（不只這個單位），否則待譯鄰行會被誤列為上下文——Task 2 測試。
3. 組回的 JSON 逐字相同的邊界：沒有詞彙條目（Go 輸出 `"glossary": null`）、含 `<` 標籤、含 U+2028、有 refs 與 keywords（map 鍵排序）——Task 4 的 fixture 與 Task 5 的測試涵蓋。
4. 上下文行的 `zh` 與檔案資料 `zh` 不同（例如紀錄帶 needs_review 時建置包放了別的字）——預覽檔要帶覆寫值；Task 4 fixture 涵蓋一行。
5. `lct site` 的時間與記憶體：每檔一個 loader、keyword 表只算一次——Task 4 測試 `Previewer` 不保留前一檔的 fileData；Task 8 量測真實資料。

---

### Task 1: 送出判定抽成 `Judge`

**Files:**
- Modify: `internal/batch/submit.go`（`(*submission).unit`、`provisional`）
- Create: `internal/batch/judge.go`
- Test: `internal/batch/judge_test.go`（新）；既有 `internal/batch/batch_test.go` 全部照過

**Interfaces:**
- Produces:
  - `type Verdict struct { ZH string; Issues []validate.Issue; Rejected bool }`
  - `func Judge(env Env, prov []rules.Term, file, kr, zh, note string) Verdict`：正規化（`validate.Normalize` 再 `NormalizeKeywordSpace(env.KeywordNames)`）→ `CheckNewTerms` → term-missing 升級（`!GlossaryWarnOnly(file) && hardEntry && !noteNames`）→ 有 hard 則 `Rejected=true`、`Issues` 不含 foreign names；否則 `Issues` = 前述 issues 加 `foreignNameIssues(ZH, env.ForeignNames, env.Rules.ForeignIgnore)`。
  - `func provisionalOf(glossary []rules.Term, inForce []rules.Term) []rules.Term`：`(*submission).provisional` 改呼叫它。
- `Submit` 行為不變：被退回時 `s.reject(key, v.Issues...)`；收下時 `Issues` 非空才進 `Warnings`；紀錄寫 `v.ZH`。

- [ ] **Step 1: 寫失敗的測試** `TestJudge`（table）：(a) hard 條目缺譯名、note 沒寫原詞 → `Rejected`、Issues 含 severity hard 的 `term-missing`，且不含 foreign-name；(b) 同上但 note 寫了原詞 → 不退回、Issues 含 warn 的 `term-missing`；(c) 族群 `glossary: warn` 時 hard 條目缺譯名 → 不退回；(d) 收下且譯文含 `ForeignNames` 的字 → Issues 含 foreign-name issue；(e) `ZH` 是正規化後的文字（例如 keyword 名稱後補半形空白）。沿用 `batch_test.go` 既有的 env/rules 測試輔助函式。
- [ ] **Step 2:** `go test ./internal/batch -run TestJudge` → FAIL（`Judge` undefined）
- [ ] **Step 3:** 實作 `judge.go`，`(*submission).unit` 改用 `Judge` 與 `provisionalOf`；刪掉 `unit` 內重複的判定碼。
- [ ] **Step 4:** `go test ./internal/batch` → PASS（含既有 Submit 測試，證明行為不變）
- [ ] **Step 5:** `go vet ./... && go test ./...` → PASS；commit「batch：送出判定抽成 Judge，Submit 與之後的預覽共用」

### Task 2: `Previewer`：單獨送翻的批次、狀態與檢查

**Files:**
- Create: `internal/batch/preview.go`、`internal/batch/preview_test.go`
- Modify: `internal/batch/next.go`（`assemble` 取 keyword 表的方式）、`internal/batch/batch.go`（`keywordNames` 改吃表）、`internal/requeue/requeue.go`（抽 `Item`）、`internal/plan/transition.go`（`ReasonNew = batch.ReasonNew`）

**Interfaces:**
- Consumes: Task 1 的 `Judge`、`provisionalOf`。
- Produces（`internal/batch`）：
  - `const ReasonNew = "new"`；`const PreviewID = "preview"`
  - `var ErrNotInKR`、`var ErrCopy`（`errors.New`，可用 `errors.Is` 判斷）
  - `type PreviewState struct { Pending bool \`json:"pending"\`; NeedsReview bool \`json:"needs_review"\`; LeasedTo string \`json:"leased_to,omitempty"\`; Frozen bool \`json:"frozen"\`; Attempts int \`json:"attempts,omitempty"\`; LastError string \`json:"last_error,omitempty"\`; RulesCommit string \`json:"rules_commit"\` }`
  - `type Check struct { Verdict string \`json:"verdict"\` /* "reject" | "warn" | "accept" */; ZH string \`json:"zh"\`; Origin string \`json:"origin"\`; StaleKR bool \`json:"stale_kr"\`; Issues []validate.Issue \`json:"issues"\` }`
  - `type UnitPreview struct { Batch *Batch \`json:"batch"\`; State PreviewState \`json:"state"\`; Pending store.PendingItem \`json:"pending"\` /* 批次所用的項目 */; Record *store.Record \`json:"record"\`; Notes []store.Note \`json:"notes"\`; Check *Check \`json:"check"\` }`（`Notes` 為空時輸出 `[]`）
  - `func NewPreviewer(env Env, frozen []string) (*Previewer, error)`：載入一次 pending、records、notes；keyword 表延後算一次並保留。
  - `func (p *Previewer) Unit(key string) (*UnitPreview, error)`：key 不在 KR → `ErrNotInKR`；copy 紀錄 → `ErrCopy`。
  - `func (p *Previewer) File(path string) ([]*UnitPreview, error)`：依 `units.Extract` 順序回傳每個單位的預覽，copy 單位為 nil；每次呼叫用新的 loader（Ruling P3）。
- `internal/requeue`：`func Item(key string, rec store.Record, hint string) store.PendingItem`，`apply` 改用它。
- `assemble` 新增參數 `keywords func(map[string]store.Record) (map[string]string, error)`（回傳整張 keyword 表）；`Next` 傳 `func(r) { return keywordTable(env.Src.KR, r) }`；`keywordNames(table, krs)` 只做過濾。
- 預覽的 `assemble` 呼叫：`pending` = 真實待譯清單（單位不在其中時附加合成項目於尾端），`byFile[file]` = 該檔所有待譯索引（含合成項），`parts = []part{{fd, []slot{{pos, idx}}}}`，`family = env.Rules.Family(file)`，id `PreviewID`。

- [ ] **Step 1: 寫失敗的測試**（`preview_test.go`）：
  - `TestPreviewMatchesNext`：store 只有一個待譯單位（劇情檔，有已譯鄰行、speaker 有角色指引）；`Next(env, NextOptions{MaxUnits: 1})` 的批次與 `Unit(key).Batch` 在把 ID 設成相同後 `reflect.DeepEqual`。
  - `TestPreviewContextSkipsPendingNeighbours`：鄰行也在待譯 → 不在 `Batch.Context`；`contextReach(1)` 之外的已譯行也不在。
  - `TestPreviewItem`：待譯單位帶 draft/hint/refs → 批次單位帶同樣欄位；已譯（claude）單位 → reason `requeue`、`prev_zh` = 紀錄 ZH、無 hint；沒有紀錄也不在待譯 → reason `new`；copy → `errors.Is(err, ErrCopy)`；不存在的 key → `ErrNotInKR`。
  - `TestPreviewState`：needs_review、租約（`LeasedTo` = 批次 id）、凍結檔（`frozen` 含該檔樣式）、attempts/last_error 照實；批次照樣組出。
  - `TestPreviewCheck`：hard 條目缺譯名 → `reject`；只有警告 → `warn`；乾淨 → `accept`；fallback 紀錄 `Origin` = `fallback`；紀錄 kr_sha 不同 → `StaleKR`；notes 只取 kr_sha 相同者、以 `\n` 合併後能讓 hard term-missing 不退回；沒有紀錄 → `Check == nil`。
  - `TestPreviewReadOnly`：`Unit` 與 `File` 前後 store 目錄所有檔案 bytes 相同。
  - `TestPreviewFileFreshLoader`：連續 `File(a)`、`File(b)` 後，`Previewer` 不持有 a 的 fileData（例如以未匯出欄位或計數檢查）。
  - `requeue`：`TestItem`，`apply` 既有測試照過。
- [ ] **Step 2:** `go test ./internal/batch ./internal/requeue` → FAIL（undefined）
- [ ] **Step 3:** 實作上列介面；`plan.ReasonNew` 改為 `batch.ReasonNew` 的別名常數。
- [ ] **Step 4:** `go test ./internal/batch ./internal/requeue ./internal/plan` → PASS
- [ ] **Step 5:** `go vet ./... && go test ./...` → PASS；commit「batch：Previewer 模擬單位單獨送翻的批次與送出檢查」

### Task 3: `lct batch preview` 與共用載入函式

**Files:**
- Create: `cmd/lct/preview.go`、`cmd/lct/preview_test.go`
- Modify: `cmd/lct/batch.go`（`openBatchEnv` 不動，新函式另寫）、`README.md`（指令一節）、`cmd/lct/contract_test.go`（若它列舉指令與 `--json` 欄位）

**Interfaces:**
- Consumes: Task 2 的 `NewPreviewer`、`Unit`、`ErrNotInKR`、`ErrCopy`、`UnitPreview`。
- Produces（`cmd/lct`，供 Task 4 使用）：`func openPreviewEnv(ctx *cli.Context, command string) (batch.Env, []string /* frozen */, error)`：用 `openTWStore`、`raw.LastSource`、`pack.OpenRaw`、`rules.Load`、foreign names（`bootstrap.ForeignNamesFile`）、`activeProvisional`、`keywordNames`、`rulesCommit`；frozen 取 `ctx.Config.Frozen`。

- [ ] **Step 1: 寫失敗的測試**（沿用 `batch_test.go` 的 CLI 測試輔助）：`TestBatchPreviewJSON`（`--json` 的頂層鍵恰為 batch、state、pending、record、notes、check；`batch.id` 為 `preview`）、`TestBatchPreviewText`（文字輸出含族群、每份指引路徑、詞彙「kr → zh」、檢查結果）、`TestBatchPreviewCopyIsUserError`（exit 1，訊息含 copy）、`TestBatchPreviewUnknownKey`（exit 1）、`TestBatchPreviewReadOnly`（tw 的 `.meta/` 前後 bytes 相同）、`TestBatchPreviewKeywordSpace`（紀錄譯文的 keyword 名稱後缺半形空白時，`check.zh` 已補上，證明 `KeywordNames` 有填）。
- [ ] **Step 2:** `go test ./cmd/lct -run BatchPreview` → FAIL
- [ ] **Step 3:** 註冊 `batch preview`（Usage：`show the batch a unit would get if it were sent alone, and how its translation fares under the submit check now: <key> [--json]`），實作 `openPreviewEnv` 與文字輸出（排版自定，含上列內容與狀態、紀錄的 origin/model/batch/rules 及是否等於現行規則）。README 指令表加一行。
- [ ] **Step 4:** `go test ./cmd/lct` → PASS
- [ ] **Step 5:** `go vet ./... && go test ./...` → PASS；commit「lct batch preview：印出單位單獨送翻的批次與送出檢查」

### Task 4: `lct site` 輸出預覽資料與 Go 端契約 fixture

**Files:**
- Create: `internal/site/preview.go`、`internal/site/preview_test.go`
- Modify: `internal/site/export.go`（`Inputs`、`Result`、`Export`）、`cmd/lct/site.go`（接 `openPreviewEnv`、讀提示詞與指引）
- Create（產生，進版控）：`site/fixtures/preview/`（`files/…`、`preview.json`、`preview/…`、`batches/<n>.json`、`cases.json`）

**Interfaces:**
- Consumes: Task 2 的 `Previewer.File`、`UnitPreview`；Task 3 的 `openPreviewEnv`。
- Produces：
  - `site.Inputs.Preview *PreviewInputs`；`type PreviewInputs struct { Previewer *batch.Previewer; RulesCommit string; Prompt string; Guides map[string]string /* 路徑 → 全文 */ }`；nil 時不輸出預覽（既有測試不變）。
  - `Result.PreviewBytes int64`（`data/preview.json` 與 `data/preview/` 的大小，含在 `Bytes` 內）。
  - 資料格式（Task 5 依此組回）：

```
data/preview.json  { "rules_commit": string, "prompt": string, "guides": [{"path": string, "text": string}] }
                   guides 只列被引用的，依第一次被引用的順序編號。
data/preview/<path>  {
  "family": string, "glossary_check": "" | "warn",
  "terms": [Term],          // rules.Term 的 JSON（Go 欄位序），不含 units；組回時尾端加 "units": [key]
  "keywords": {id: name},   // 本檔用到的 keyword 名稱
  "units": [Unit | null]    // 與 files/<path> 的 units 同序同長；copy 為 null
}
Unit {
  "sp": string,                         // BatchUnit.speaker
  "g": [int], "t": [int], "k": [string] // Batch.guides / glossary / keywords 的順序與內容
  "it": {"reason": string, "prev_kr"?, "prev_zh"?, "draft"?, "hint"?, "refs"?: {..}},
  "em"?: string, "jm"?: string,         // 有對齊時的 method；value 取 files/<path> 的 en/jp
  "keep": Constraints 的 JSON,
  "c": [[int] | [int, string]],         // Batch.context：單位編號，必要時帶 zh 覆寫
  "st": PreviewState（不含 rules_commit）, "rec"?: {origin, model?, batch?, since?, rules?, flags?, stale: bool},
  "chk"?: {"verdict", "zh", "issues": [Issue]}, "nt"?: string
}
```
  - 上下文行的 speaker 取該行單位的 `sp`，kr 取檔案資料的 `kr`。

- [ ] **Step 1: 寫失敗的測試**：
  - `TestExportPreviewRoundTrip`：用測試 pack 呼叫 `Export`，對每個非 null 單位，以 Go 寫一個「照上列格式組回批次」的測試輔助函式，組回的結果與 `Previewer.Unit(key).Batch` 經 `writeJSONFile` 同格式編碼（`json.Encoder`、`SetEscapeHTML(false)`、`SetIndent("", "  ")`）後逐字相同。
  - `TestExportPreviewCopyNull`、`TestExportPreviewSharedGuides`（只列被引用的指引、順序穩定、文字取自 `Guides`；被引用的路徑不在 `Guides` → error）、`TestExportWithoutPreview`（`Inputs.Preview == nil` 時不寫 `preview*`）。
  - `TestPreviewFixturesUpToDate`：以固定測試 pack 產生 `site/fixtures/preview/`，與版控中的檔案逐位元組比較；加 `-update` 旗標時改寫它們。fixture 的單位必須涵蓋：劇情單位（有上下文且有 speaker 指引）、非劇情單位、含 `[ID]`（有 keywords）、含 `<` 標籤、待譯且有 draft 與 hint 與 refs、沒有任何詞彙條目（`"glossary": null`）、KR 含 U+2028、一行上下文需要 zh 覆寫。`cases.json` 列出每個案例的 `{path, index, batch: "batches/<n>.json"}`；`batches/<n>.json` 是該單位 `Previewer.Unit` 的批次以 `writeJSONFile` 格式寫出的文字。
- [ ] **Step 2:** `go test ./internal/site -run 'Preview|WithoutPreview'` → FAIL
- [ ] **Step 3:** 實作 `preview.go` 與 `Export` 的接線；`go test ./internal/site -run TestPreviewFixturesUpToDate -update` 產生 fixture。`cmd/lct/site.go`：`openPreviewEnv`、`batch.NewPreviewer`、讀 `prompts/translate.md` 與 `guides/` 下所有 `.md`（路徑用 repo 相對、`/` 分隔），傳入 `Inputs.Preview`；`siteOutput.String()` 加上預覽大小。
- [ ] **Step 4:** `go test ./internal/site ./cmd/lct` → PASS
- [ ] **Step 5:** `go vet ./... && go test ./...` → PASS；commit「lct site：輸出每檔預覽資料與共用的提示詞、指引；契約 fixture」

### Task 5: 網站資料層與批次組回

**Files:**
- Create: `site/src/preview/rebuild.ts`、`site/src/preview/rebuild.test.ts`
- Modify: `site/src/types.ts`（`PreviewShared`、`PreviewFile`、`PreviewUnit` 等，對應 Task 4 格式）、`site/src/data.ts`（`loadPreviewShared(): Promise<PreviewShared>`、`loadPreviewFile(path: string): Promise<PreviewFile>`，路徑編碼同 `loadFile`）、`site/src/data.test.ts`

**Interfaces:**
- Consumes: Task 4 的資料格式與 `site/fixtures/preview/`。
- Produces：
  - `goJSON(value: unknown): string`：`JSON.stringify(value, null, 2)` 再把 U+2028、U+2029 換成 ` `、` `，結尾加 `\n`。
  - `rebuildBatch(path: string, file: FileData, pf: PreviewFile, shared: PreviewShared, index: number): BatchJSON`：物件鍵依 Go 欄位序——id、family、glossary_check（空字串時省略）、file、files、guides、glossary（空時 `null`）、keywords（鍵排序）、units、context；單位鍵序 key、file、field、speaker、kr、reason、prev_kr、prev_zh、draft、hint、en、jp、refs（鍵排序）、keep；context 行鍵序 key、speaker、kr、zh。
  - `fullText(shared: PreviewShared, batch: BatchJSON): string`：依序 `# prompts/translate.md` 標題與提示詞、每份指引以 `# <path>` 標題與全文、最後 `# 批次 JSON` 與 `goJSON(batch)`，段落間空一行。
  - 單位 key：`${path}|${pointer}`。

- [ ] **Step 1: 寫失敗的測試**：`rebuild.test.ts` 讀 `site/fixtures/preview/cases.json`，每個案例 `goJSON(rebuildBatch(...))` 必須 `===` `batches/<n>.json` 的文字（用 `fs.readFileSync`）；`goJSON` 單元測試（U+2028、`<`、`&` 不跳脫、空陣列 `[]`、空物件 `{}`）；`fullText` 的段落順序。`data.test.ts`：兩個 loader 的路徑與格式錯誤處理。
- [ ] **Step 2:** `cd site && npx vitest run src/preview src/data.test.ts` → FAIL
- [ ] **Step 3:** 實作。
- [ ] **Step 4:** 同指令 → PASS
- [ ] **Step 5:** `cd site && npm test && npm run typecheck && npm run build` → PASS；commit「網站：預覽資料載入與批次 JSON 組回（與 Go 逐字相同）」

### Task 6: 網站預覽頁與檔案頁入口

**Files:**
- Create: `site/src/preview/PreviewView.tsx`、`site/src/preview/PreviewView.test.tsx`
- Modify: `site/src/route.ts`（`{ kind: 'preview'; path: string; ptr: string }`、`previewHref(path, ptr)`）、`site/src/route.test.ts`、`site/src/app/App.tsx`、`site/src/view/TableView.tsx`、`site/src/view/ScriptView.tsx`、`site/src/view/FileView.test.tsx`、`site/src/styles.css`、`README.md`（網站一節）、`.claude/skills/review-page/SKILL.md`（提到可用預覽頁或 `lct batch preview` 查單位的批次組成）

**Interfaces:**
- Consumes: Task 5 的 loader、`rebuildBatch`、`goJSON`、`fullText`；既有 `loadFile`、`loadMeta`、`OriginBadge`、`Cell`。

- [ ] **Step 1: 寫失敗的測試**：
  - route：`#/pv/StoryData/S001A.json?u=%2FdataList%2F3%2Fcontent` ↔ `{kind:'preview', path, ptr}` 雙向；缺 `u` 時退回檔案頁路由。
  - 檔案頁：表格每列（非 copy）有連結文字「預覽」、href 為 `previewHref`；copy 列沒有連結，改為淡色文字「不送翻」，title 為「copy：識別碼或各語言相同，不送翻」；劇本檢視每個單位一個「預覽」連結（同行多單位時每個都有，title 含欄位名），copy 單位同上。
  - 預覽頁（用 `site/fixtures/preview/` 的資料）：頁首「依規則 `<前 7 碼>` 模擬」；「組合」分頁列出每份指引路徑、詞彙「kr → zh」並連到 `#/g/<a>/<b>`（暫定詞不連）、keywords、上下文行、keep、待譯原因與舊譯、紀錄（origin、model、batch、規則 commit 是否等於現行）、檢查結果文字「退回」「收下但有警告」「收下」與每個 issue 的 code 與 message；needs_review、租約、凍結、fallback、human、舊 KR、沒有譯文的提示文字照 Global Constraints；「全文」分頁顯示 `fullText` 的文字且有「複製全文」按鈕（沿用詞彙頁的剪貼簿退路：被拒時顯示選取好的 textarea 與提示）；有回到 `#/f/<path>?u=<ptr>` 的連結。
  - 指定的 `u` 在檔案中不存在或是 copy → 顯示說明而非崩潰。
- [ ] **Step 2:** `cd site && npx vitest run src/preview src/route.test.ts src/view` → FAIL
- [ ] **Step 3:** 實作；版面自定，長文字（JSON、指引全文）在自己的捲動框內，400px 時整頁不橫向捲動。
- [ ] **Step 4:** 同指令 → PASS
- [ ] **Step 5:** `cd site && npm test && npm run typecheck && npm run build` 與 `go vet ./... && go test ./...` → PASS；commit「網站：翻譯預覽頁與檔案頁的預覽連結」

### Task 7: 真實資料驗收

**Files:** 無程式改動（若驗收發現問題，回到所屬 Task 修正並加測試）。

- [ ] **Step 1:** 在 scratch clone（`git fetch /home/user/limbus-translate refs/remotes/origin/raw:refs/heads/raw-latest refs/remotes/origin/tw:refs/heads/tw-latest`，不碰 `/home/user/limbus-translate/.work/`）以 `main`（4b68ada）與本分支 HEAD 各建一次 `bin/lct`，各跑一次 `bin/lct site --out site/public/data --json`，記錄時間、`bytes`、`preview_bytes`。
- [ ] **Step 2:** 抽 6 個單位（劇情有上下文、劇情待譯、技能檔含 `[ID]`、UI、fallback、needs_review 或租約中各一，找不到就註明）：`bin/lct batch preview <key> --json` 的 `batch` 以 `writeJSONFile` 格式寫出，與網站組回（用建好的 dist 加真實資料、Playwright 開 `#/pv/...`，取「全文」分頁的批次 JSON 段）逐字比較。
- [ ] **Step 3:** Playwright 400px 與 1100px 截圖預覽頁（組合、全文），確認無整頁橫向捲動。
- [ ] **Step 4:** 確認 store（tw `.meta/`）在上述指令前後 `git status` 乾淨。
- [ ] **Step 5:** 結果寫進最終回報（大小、時間、與預算比較、比對結果）。

## 最後

全部完成後做整支分支的最終審查（最強模型），修正 Critical/Important，Minor 一併列給使用者；再請使用者同意後才 fast-forward `main`（使用者指示：做完預覽再合併，含先前的小問題與表頭排序）。
