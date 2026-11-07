# 專有詞彙一致 實作計畫

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 新譯文必須用詞彙表譯法（例外要在 note 寫明原詞）；譯者登記的新詞成為暫定詞，後面的批次照用；非劇情小檔案合批、關鍵字檔先發放；審核頁依詞列出既有譯文中不合詞彙表的地方。

**Architecture:** 驗證器新增永遠是警告的 `term-missing`，只有 `batch.Submit` 對沒寫 note 說明的升為硬性。暫定詞存在 `tw` 的 `.meta/terms.jsonl`，由 `cmd/lct` 載入放進 `batch.Env`，`Next` 放進批次的 `glossary`，`Submit` 一併檢查。`Next` 的檔案排序先放關鍵字檔，非劇情批次可接同族群的其他檔。`lct suggest` 與網站審核頁新增暫定詞項目與依詞分組的 `term-missing` 警告。

**Tech Stack:** Go（`internal/validate`、`internal/rules`、`internal/batch`、`internal/store`、`internal/suggest`、`internal/site`、`internal/build`、`cmd/lct`）；Preact + Vitest（`site/`）；skill 與提示詞（`.claude/skills/`、`prompts/translate.md`、`schemas/`）。

**Spec:** `docs/superpowers/specs/2026-11-05-term-consistency-design.md`

## Global Constraints

- `CLAUDE.md`：規則只在 `guides/`、`glossary/`、`rules/`；狀態只經由 `lct`；`raw`、`tw`、Release 只由 CI 寫入；所有 `lct` 指令支援 `--json`；skill 與 `prompts/` 不寫模型名稱。
- `term-missing` 只在 `batch.Submit` 且該單位 `note` 不含條目 `kr` 時是硬性；其他所有地方（建置、審核頁、`CheckNew` 的其他呼叫者、Paratranz 匯入、`lct adjust add`、`lct validate`）一律是警告。
- 暫定詞：用 `match: word` 判定、`kr` 至少兩個字、不限檔案；`glossary/terms.yaml` 有同樣 `kr` 的條目或 `rules/declined.yaml` 否決了 `term-new/<kr>` 時不生效；對既有譯文不產生警告；不由 CI 的 `/rules-review` 處理。
- 「保留這些譯法」逐單位記錄（`warning/term-missing/<sha8 KR>/<sha8 TW>`）；依詞分組的項目 ID 不得符合這個逐單位格式。
- 劇情批次（`batch.IsStory`）維持單一檔案與 `context`；非劇情批次合批不超過 `batch.max_units`。
- 新的批次檔與輸出檔欄位要同時加進 `schemas/batch.schema.json`、`schemas/batch-out.schema.json`（兩者都是 `additionalProperties: false`，`lct` 也用嚴格解碼讀）。
- skill 的 `allowed-tools`、內文的必備句與提示詞的釘選測試在 `cmd/lct/contract_test.go`，改 skill 時一起改。
- 提交前 `set -o pipefail; go vet ./... && go test ./...`；動到 `site/` 另跑 `cd site && npm test && npm run typecheck && npm run build`；不用 prettier 重排既有檔案。
- commit 訊息結尾附上工作階段指定的 `Co-Authored-By` 與 `Claude-Session` 兩行。
- 在 `claude/festive-wright-g2k853` 開發；不 push `main`、`tw`、`raw`。

## Review Focus

1. **指定譯法被 TextMeshPro 標籤包住或切開**（`<color=#fff>雷鳴</color>`、`雷<b>鳴</b>`）：仍算有用到，不記 `term-missing`。→ Task 1 `TestTermMissingIgnoresTags`。
2. **note 寫的是別的詞**：note 只提到另一個詞的原文時，缺指定譯法的仍被退回；note 含原詞才收下。→ Task 2 `TestSubmitTermMissingNeedsTheTermInTheNote`。
3. **同一批裡兩個單位登記同一個 `kr` 但譯法不同**：先處理的生效，另一筆記成衝突，不覆蓋。→ Task 3 `TestSubmitRecordsTheFirstRenderingAndAConflict`。
4. **暫定詞與詞彙表的詞互相包含**（暫定「와일드헌트」、詞彙表「와일드헌트 히스클리프」）：兩者各自依命中規則生效；詞彙表有完全相同 `kr` 時暫定詞不生效。→ Task 4 `TestProvisionalTermsYieldToTheGlossary`。
5. **合批後單位屬於不同檔案**：以 `files:` 限定的詞彙表條目只套用在它的檔案的單位，送出時用單位自己的檔案檢查。→ Task 5 `TestMergedBatchScopesTermsByUnitFile`。

---

### Task 1: 驗證器的 `term-missing`

**Files:**
- Modify: `internal/validate/validate.go`（`Term`、`Issue`、`checkTerms`、新碼）
- Modify: `internal/rules/glossary.go`（`ToValidate` 帶 `ZH`）
- Test: `internal/validate/validate_test.go`、`internal/rules/glossary_test.go`

**Interfaces:**
- Produces:
  ```go
  // validate
  const CodeTermMissing = "term-missing"
  type Term struct {
      KR        string
      ZH        string   // 指定譯法；空字串或與 KR 相同時不檢查是否用到
      Forbidden []string
  }
  type Issue struct { // 既有欄位不變，新增：
      Term string `json:"term,omitempty"` // term-missing 與 forbidden-term：條目的 KR
  }
  ```
  `term-missing` 的 `Class` 是 `ClassContent`，`Severity` 一律 `SeverityWarn`（不論 `Existing`）。訊息含原詞、指定譯法，以及「若是合理例外，在 note 寫出原詞『<kr>』與理由」。
  ```go
  // rules
  func (t Term) ToValidate() validate.Term // 帶 KR、ZH、Forbidden
  ```

- [ ] **Step 1: 寫失敗的測試**

```go
func TestTermMissing(t *testing.T)
// Check("우레가 친다", "打雷了", Context{Terms: []Term{{KR: "우레", ZH: "雷鳴"}}})
// → 一則 term-missing，Severity warn，Class content，Term "우레"，Message 含 "雷鳴" 與 "note"
// 譯文 "雷鳴響起" → 沒有 term-missing
// Existing: true 與 false 都是 warn
func TestTermMissingSkipsTermsWithoutARendering(t *testing.T) // ZH "" 或 ZH == KR 的條目不檢查
func TestTermMissingIgnoresTags(t *testing.T)
// "<color=#ffffff>雷鳴</color>響起"、"雷<b>鳴</b>響起" → 沒有 term-missing
func TestForbiddenTermCarriesTheTerm(t *testing.T) // forbidden-term 的 Issue.Term 是條目的 KR
// rules：
func TestToValidateCarriesTheRendering(t *testing.T) // ToValidate().ZH == Term.ZH
```

- [ ] **Step 2: 跑測試，確認失敗**

Run: `go test ./internal/validate/ ./internal/rules/`
Expected: FAIL（`undefined: CodeTermMissing`、`unknown field ZH` 等）

- [ ] **Step 3: 實作**：在 `checkTerms` 的禁用譯法檢查之後加 `term-missing`；比對時用去掉 TextMeshPro 標籤的譯文（沿用驗證器既有的標籤處理函式；沒有就加一個只供此處用的）。

- [ ] **Step 4: 跑測試，確認通過**

Run: `set -o pipefail; go vet ./... && go test ./...`
Expected: PASS（既有測試不因新增的警告失敗；若有測試比對完整 issue 清單而失敗，確認是新增的 `term-missing`/`Term` 所致再更新預期）

- [ ] **Step 5: Commit**

```bash
git add internal/validate internal/rules
git commit -m "feat: the validator warns when a glossary term is translated without its rendering"
```

---

### Task 2: 送出時退回、建置與審核頁帶詞彙表

**Files:**
- Modify: `internal/batch/submit.go`（升級為硬性）
- Modify: `internal/build/build.go:487`（帶 `Terms`）
- Modify: `internal/site/review.go:272`（帶 `Terms`）
- Modify: `prompts/translate.md`、`.claude/skills/translate-pending/SKILL.md`（說明退回與 note 例外）
- Test: `internal/batch/submit_test.go`、`internal/build/build_test.go`、`internal/site/review_test.go`、`cmd/lct/contract_test.go`

**Interfaces:**
- Consumes: Task 1 的 `CodeTermMissing`、`Issue.Term`。
- Produces: `Submit` 對 `Code == CodeTermMissing` 且 `!strings.Contains(out.Note, issue.Term)` 的 issue 把 `Severity` 改為 `SeverityHard`（只在 `Submit` 內）；其他 `term-missing` 照常是警告，出現在 `SubmitResult.Warnings`。建置與審核頁呼叫 `validate.Check` 時帶 `Terms: rules.ValidateTerms(file, kr)`，因此也會出現既有譯文的 `forbidden-term` 警告（`Existing: true` 時是警告）。

- [ ] **Step 1: 寫失敗的測試**

```go
func TestSubmitRejectsAMissingTermWithoutANote(t *testing.T)
// 詞彙表 시즌 → 賽季；送出「從兩棵季前起…」無 note → Rejected，errors 含 term-missing（severity hard），Attempts 加 1
func TestSubmitAcceptsAMissingTermTheNoteNames(t *testing.T)
// 同上，note「시즌：劇中季節，非遊戲賽季」→ Accepted，Warnings 含 term-missing（warn）
func TestSubmitTermMissingNeedsTheTermInTheNote(t *testing.T)
// note 只提到別的詞（「탐식：卡包名」）→ 仍 Rejected
func TestCheckNewLeavesTermMissingAWarning(t *testing.T) // CheckNew 對缺譯法的只回 warn，HasHard 為 false
// build：
func TestBuildCountsTermMissingWarnings(t *testing.T) // report.Warnings["term-missing"] 計入；accepted.yaml 有該單位的 warning id 時不計
// site：
func TestReviewWarnsOnTermMissing(t *testing.T) // BuildReview 的 Warnings 有 code term-missing 的項目
```
`contract_test.go`：`prompts/translate.md` 與 translate-pending 的 `SKILL.md` 須提到 `term-missing` 與「note 寫出原詞」的例外（加進既有的必備句清單或新的釘選測試）。

- [ ] **Step 2: 跑測試，確認失敗**

Run: `go test ./internal/batch/ ./internal/build/ ./internal/site/ ./cmd/lct/ -run 'TermMissing|Contract'`
Expected: FAIL

- [ ] **Step 3: 實作**，並改提示詞「名稱與詞彙」一節與 skill 的「退回」一節：缺指定譯法會被退回；合理例外就在該單位 `note` 寫出原詞與理由再送。

- [ ] **Step 4: 跑測試，確認通過**

Run: `set -o pipefail; go vet ./... && go test ./...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/batch internal/build internal/site prompts .claude/skills/translate-pending cmd/lct/contract_test.go
git commit -m "feat: a submission without the glossary rendering is rejected unless its note names the term"
```

---

### Task 3: 譯者登記新詞，送出時記為暫定詞

**Files:**
- Create: `internal/store/terms.go`、`internal/store/terms_test.go`
- Modify: `internal/batch/batch.go`（`OutUnit`）、`internal/batch/submit.go`（記錄）
- Modify: `schemas/batch-out.schema.json`、`prompts/translate.md`、`.claude/skills/translate-pending/SKILL.md`
- Test: `internal/batch/submit_test.go`、`cmd/lct/contract_test.go`

**Interfaces:**
- Produces:
  ```go
  // batch
  type NewTerm struct {
      KR string `json:"kr"`
      ZH string `json:"zh"`
  }
  type OutUnit struct { // 既有欄位不變，新增：
      Terms []NewTerm `json:"terms,omitempty"`
  }
  type SkippedTerm struct {
      Key    string `json:"key"`
      KR     string `json:"kr"`
      ZH     string `json:"zh"`
      Reason string `json:"reason"`
  }
  // SubmitResult 新增：
  Terms        int           `json:"terms"`         // 這次新記下的暫定詞數
  TermsSkipped []SkippedTerm `json:"terms_skipped"` // 從不為 nil
  // store：.meta/terms.jsonl，一行一筆
  type ProvisionalTerm struct {
      KR       string `json:"kr"`
      ZH       string `json:"zh"`
      Key      string `json:"key"`
      Batch    string `json:"batch"`
      RawTag   string `json:"raw_tag"`
      Time     string `json:"time"`               // RFC 3339，UTC
      Conflict bool   `json:"conflict,omitempty"` // 同 KR 已有不同譯法時記下的
  }
  func (s *Store) LoadTerms() ([]ProvisionalTerm, error)       // 沒有檔案時回空
  func (s *Store) AppendTerms(t []ProvisionalTerm) (int, error) // 呼叫者持鎖；KR+ZH 相同的不重記
  ```
  登記的條件（只對收下的單位）：`kr` 在該單位的 KR 原文中、`zh` 在正規化後的譯文中、`kr` 至少兩個字；不合的列進 `TermsSkipped`（理由用英文短句），不影響單位本身。同一個 `kr` 已有暫定詞（不論是否同批）時：譯法相同不記；不同則記一筆 `Conflict: true`。

- [ ] **Step 1: 寫失敗的測試**

```go
func TestSubmitRecordsAProvisionalTerm(t *testing.T)
// out {"key":k,"zh":"雷鳴","terms":[{"kr":"우레","zh":"雷鳴"}]}（KR 含 우레）→ Accepted；Terms == 1；LoadTerms() 有一筆 KR 우레、ZH 雷鳴、Batch、RawTag
func TestSubmitSkipsATermNotInTheUnit(t *testing.T)
// kr 不在 KR、zh 不在譯文、kr 只有一個字 → TermsSkipped 各一筆、LoadTerms() 為空
func TestSubmitRecordsTheFirstRenderingAndAConflict(t *testing.T)
// 同批兩個單位登記 우레→雷鳴、우레→驚雷 → 第一筆生效，第二筆 Conflict: true；再送一次相同的不重記
func TestSubmitRecordsNoTermOfARejectedUnit(t *testing.T)
// store：
func TestAppendTermsAndLoad(t *testing.T)
```
`contract_test.go`：`schemas/batch-out.schema.json` 的 items 有 `terms`（陣列，項目 `kr`、`zh` 字串）；`prompts/translate.md` 說明 `terms`。

- [ ] **Step 2: 跑測試，確認失敗**

Run: `go test ./internal/store/ ./internal/batch/ ./cmd/lct/ -run 'Term|Contract'`
Expected: FAIL

- [ ] **Step 3: 實作**：記錄在 `Submit` 既有的 store 鎖內、存紀錄之前；`cmd/lct` 的 `submitReport` 文字輸出加一行暫定詞數與略過的項目。提示詞「名稱與詞彙」一節：詞彙表沒有、但應統一譯法的名稱，除了 `note`，也在該單位的 `terms` 登記 `{kr, zh}`；kr 用原文中的完整詞。

- [ ] **Step 4: 跑測試，確認通過**

Run: `set -o pipefail; go vet ./... && go test ./...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/store internal/batch cmd/lct schemas prompts .claude/skills/translate-pending
git commit -m "feat: the translator registers new terms, which submit keeps as provisional terms"
```

---

### Task 4: 暫定詞進入批次並在送出時檢查

**Files:**
- Modify: `internal/rules/glossary.go`（匯出命中判定、`Term.Provisional`）
- Modify: `internal/batch/batch.go`（`Env.Provisional`、`CheckNewTerms`）、`internal/batch/next.go`（放進 `glossary`）、`internal/batch/submit.go`（一併檢查）
- Modify: `cmd/lct/batch.go`（`openBatchEnv` 載入）
- Modify: `schemas/batch.schema.json`（glossary 項目加 `match`、`provisional`）、`prompts/translate.md`
- Test: `internal/rules/glossary_test.go`、`internal/batch/next_test.go`、`internal/batch/submit_test.go`、`cmd/lct/batch_test.go`

**Interfaces:**
- Consumes: Task 3 的 `store.ProvisionalTerm`、`LoadTerms`；`suggest.LoadDeclined`。
- Produces:
  ```go
  // rules
  type Term struct { // 既有欄位不變，新增（不寫進 yaml）：
      Provisional bool `yaml:"-" json:"provisional,omitempty"`
  }
  func (t Term) OccursIn(kr string) bool // 既有 occursIn 匯出
  // batch
  type Env struct { // 既有欄位不變，新增：
      Provisional []rules.Term // 生效中的暫定詞：Match "word"、Status "approved"、Provisional true
  }
  func ActiveProvisional(all []store.ProvisionalTerm, r *rules.Rules, declined map[string]bool) []rules.Term
  // 每個 KR 取第一筆非衝突的；略過：kr 少於兩個字、詞彙表有同 KR 的條目（任何狀態）、declined["term-new/"+kr]
  func CheckNewTerms(r *rules.Rules, provisional []rules.Term, file, kr, zh string) []validate.Issue
  // = CheckNew 加上 provisional 中 OccursIn(kr) 的條目；CheckNew 改為呼叫 CheckNewTerms(r, nil, …)
  ```
  `Next`：批次的 `glossary` 是各單位詞彙表命中的聯集，再加上各單位命中的暫定詞（`provisional: true`），依 KR 去重。`Submit`：用 `CheckNewTerms(env.Rules, env.Provisional, file, kr, zh)`，暫定詞的 `term-missing` 與詞彙表一樣依 note 升為硬性。`cmd/lct`：`openBatchEnv` 讀 `.meta/terms.jsonl` 與 `rules/declined.yaml`，算出 `Env.Provisional`。

- [ ] **Step 1: 寫失敗的測試**

```go
func TestActiveProvisional(t *testing.T)
// 우레→雷鳴（第一筆）、우레→驚雷（Conflict）、하→X（一個字）、詞彙表已有的 KR、declined 的 KR
// → 只回 우레→雷鳴，Match "word"，Provisional true
func TestProvisionalTermsYieldToTheGlossary(t *testing.T)
// 暫定「와일드헌트」與詞彙表「와일드헌트 히스클리프」都命中；詞彙表有完全相同的「와일드헌트」時暫定的不生效
func TestNextCarriesProvisionalTerms(t *testing.T)
// Env.Provisional 有 우레→雷鳴；含「우레가」的單位的批次 glossary 有它且 provisional true；含「우레비」的（不成詞）沒有
func TestSubmitChecksProvisionalTerms(t *testing.T)
// 批次單位含 우레、譯文沒有「雷鳴」、無 note → Rejected（term-missing）；note 含「우레」→ Accepted 附警告
func TestBatchNextLoadsProvisionalTerms(t *testing.T) // cmd：寫 .meta/terms.jsonl 後 lct batch next 的批次帶有暫定詞
```
`cmd/lct/contract_test.go` 或 batch 測試：產生的批次檔照 `schemas/batch.schema.json` 驗證通過（含 `match`、`provisional`）。

- [ ] **Step 2: 跑測試，確認失敗**

Run: `go test ./internal/rules/ ./internal/batch/ ./cmd/lct/ -run 'Provisional|Contract'`
Expected: FAIL

- [ ] **Step 3: 實作**，提示詞補一句：`glossary` 中 `provisional: true` 的是本專案先前登記、尚待確認的暫定詞，和其他條目一樣照用，例外同樣在 `note` 寫出原詞。

- [ ] **Step 4: 跑測試，確認通過**

Run: `set -o pipefail; go vet ./... && go test ./...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/rules internal/batch cmd/lct schemas prompts
git commit -m "feat: provisional terms go into the batches that contain them and are checked on submit"
```

---

### Task 5: 關鍵字檔先發放、非劇情小檔案合批

**Files:**
- Modify: `internal/batch/next.go`（排序與合批）、`internal/batch/batch.go`（`Batch.Files`、`BatchUnit.File`）、`internal/batch/submit.go`（以單位的檔案檢查）
- Modify: `schemas/batch.schema.json`、`prompts/translate.md`、`.claude/skills/translate-pending/SKILL.md`
- Test: `internal/batch/next_test.go`、`internal/batch/submit_test.go`、`cmd/lct/e2e_phase2_test.go`

**Interfaces:**
- Consumes: `names.IsKeywordFile(file string) bool`（`BattleKeywords*`、`Bufs*`）。
- Produces:
  ```go
  type Batch struct { // File 改為第一個檔案；新增：
      Files []string `json:"files"` // 本批的檔案，依發放順序；從不為空
  }
  type BatchUnit struct { // 新增：
      File string `json:"file"`
  }
  ```
  候選檔案排序：`names.IsKeywordFile` 為真的在前，之後依 `Rules.Priority(family)`、再依檔名。取第一個有單位的檔案；若它不是劇情（`!IsStory`）且單位數小於 `MaxUnits`，依同樣順序接著取**同族群**、非劇情的其他檔案，直到 `MaxUnits`。`guides` 是族群指引加上所有單位說話者的角色指引；`keywords` 與 `glossary` 是所有單位的聯集。`Submit` 檢查單位時用該單位 key 的檔案（不用 `Batch.File`）。

- [ ] **Step 1: 寫失敗的測試**

```go
func TestNextHandsOutKeywordFilesFirst(t *testing.T)
// 待譯：Passives.json（ally）、Bufs_X.json（other）、BattleKeywords_X.json（other）→ 第一批的檔案是 BattleKeywords_X.json 或 Bufs_X.json
func TestNextMergesSmallFilesOfAFamily(t *testing.T)
// 同族群三個檔各 2 單位、MaxUnits 40 → 一批 6 單位，Files 三個、每個單位有 File；另一族群的檔不併入
func TestNextStopsAtMaxUnits(t *testing.T) // 合批不超過 MaxUnits
func TestNextKeepsStoryBatchesToOneFile(t *testing.T) // StoryData/ 的兩個檔 → 兩批，各有 context
func TestMergedBatchScopesTermsByUnitFile(t *testing.T)
// 詞彙表條目 files: [A.json]；合批 A、B；B 的單位不套用該條目（glossary 只因 A 的單位而有它；送出時 B 單位缺該譯法不被退回）
func TestMergedBatchGuidesCoverEverySpeaker(t *testing.T)
```
`e2e_phase2_test.go`：replay 仍全數翻完；斷言至少有一批的 `Files` 超過一個（replay 資料有同族群的小檔時），否則說明為何不適用。

- [ ] **Step 2: 跑測試，確認失敗**

Run: `go test ./internal/batch/ ./cmd/lct/ -run 'Next|Merged|Phase2'`
Expected: FAIL

- [ ] **Step 3: 實作**；提示詞與 skill：說明單位的 `file`，依檔名或人格區分語氣時以單位的 `file` 判斷；批次可能含多個檔。

- [ ] **Step 4: 跑測試，確認通過**

Run: `set -o pipefail; go vet ./... && go test ./...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/batch cmd/lct schemas prompts .claude/skills/translate-pending
git commit -m "feat: keyword files are handed out first, and small files of a family share a batch"
```

---

### Task 6: 暫定詞成為規則建議，由人決定

**Files:**
- Modify: `internal/suggest/suggest.go`（新種類）、`cmd/lct/suggest.go`（載入 `.meta/terms.jsonl`）
- Modify: `.claude/skills/review-page/SKILL.md`、`.claude/skills/term-decision/SKILL.md`、`.claude/skills/rules-review/SKILL.md`，以及 CI 呼叫 `lct suggest` 的地方（`rules-proposal.yml` 或 rules-review 的指令）
- Test: `internal/suggest/suggest_test.go`、`cmd/lct/suggest_test.go`、`cmd/lct/contract_test.go`

**Interfaces:**
- Consumes: Task 3 的 `store.ProvisionalTerm`；Task 4 的 `ActiveProvisional` 的篩選規則（詞彙表已有、否決的不列）。
- Produces:
  ```go
  const KindNewTerm = "term-new" // 加進 Kinds，排在 KindTerm 之前
  // Inputs 新增：
  Provisional []store.ProvisionalTerm
  ```
  每個 `kr` 一個建議：ID `term-new/<kr>`，標題「暫定詞：<kr> → <zh>」，Evidence：登記的單位（Key、ZH、Note 寫批次 id），以及每筆衝突（Key、ZH、Note「衝突」）。詞彙表已有同 KR 的條目或 `rules/declined.yaml` 有該 ID 時不列。`lct accept term-new/<kr>` 寫入 `rules/declined.yaml`（既有路徑，因為種類在 `Kinds`）。CI 的規則建議流程不處理 `term-new`（在它呼叫 `lct suggest` 處排除這個種類，或 skill 明確跳過；以契約測試釘住）。

- [ ] **Step 1: 寫失敗的測試**

```go
func TestSuggestListsProvisionalTerms(t *testing.T)
// 우레→雷鳴（+ 一筆衝突 驚雷）→ 一個 term-new/우레，Evidence 兩筆；詞彙表有 우레 時沒有；declined 有該 ID 時沒有
func TestSuggestKindsOrderNewTermBeforeTerm(t *testing.T)
func TestAcceptDeclinesANewTerm(t *testing.T) // cmd：lct accept term-new/우레 --reason … 寫進 declined.yaml；之後 lct batch next 不再帶這個暫定詞
```
`contract_test.go`：review-page 與 term-decision 的必備句提到 `term-new`（確認＝寫入 `glossary/terms.yaml`、可改譯法；否決＝`bin/lct accept`）；rules-review 的必備句說明不處理 `term-new`；CI 的建議流程排除 `term-new`。

- [ ] **Step 2: 跑測試，確認失敗**

Run: `go test ./internal/suggest/ ./cmd/lct/ -run 'NewTerm|Provisional|Contract'`
Expected: FAIL

- [ ] **Step 3: 實作**

- [ ] **Step 4: 跑測試，確認通過**

Run: `set -o pipefail; go vet ./... && go test ./...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/suggest cmd/lct .claude/skills .github/workflows
git commit -m "feat: provisional terms are rule suggestions a person approves or declines"
```

---

### Task 7: 審核頁資料：`term-missing` 依詞分組、暫定詞的命中單位

**Files:**
- Modify: `internal/site/review.go`（分組與暫定詞的證據）、`cmd/lct/site.go`（傳入暫定詞）
- Modify: `cmd/lct/accept.go`（群組 ID 展開成逐單位）
- Modify: `.claude/skills/review-page/SKILL.md`
- Test: `internal/site/review_test.go`、`cmd/lct/accept_test.go`、`cmd/lct/contract_test.go`

**Interfaces:**
- Consumes: Task 2 的 `term-missing` 警告（`Issue.Term`）；Task 6 的 `term-new` 建議。
- Produces:
  ```go
  // WarningItem 新增（只有 term-missing 群組有）：
  Term  *TermRef    `json:"term,omitempty"`  // {kr, zh}
  Units []TermUnit  `json:"units,omitempty"`
  type TermRef struct  { KR string `json:"kr"`; ZH string `json:"zh"` }
  type TermUnit struct {
      Key  string `json:"key"`
      ID   string `json:"id"`   // 逐單位的 warning/term-missing/<sha8 KR>/<sha8 TW>
      KR   string `json:"kr"`
      ZH   string `json:"zh"`
      Note string `json:"note,omitempty"`
      Accepted *Confirmed `json:"accepted,omitempty"`
  }
  ```
  `term-missing` 不再逐 KR/TW 產生項目，而是每個詞彙表條目一個項目：ID `termgroup/<sha8 term KR>/<sha8 term ZH>`（不符合 `acceptedIDPattern`），`KR`／`ZH` 是條目的原詞與譯法，`Keys` 是未確認的單位，`Units` 是所有單位（已確認的帶 `Accepted`）。暫定詞建議（`term-new/…`）的 `Evidence` 另加：目前 KR 命中它（`OccursIn`）的所有單位與現在的譯文（`Note` 寫「有用」或「未用」），最多 `suggest.MaxEvidence`。`lct accept termgroup/…`：展開成該群組未確認單位的逐單位 ID，全部寫入 `rules/accepted.yaml`；`--dry-run` 列出它們。

- [ ] **Step 1: 寫失敗的測試**

```go
func TestReviewGroupsTermMissingByTerm(t *testing.T)
// 三個單位缺「自動販賣機」→ 一個 WarningItem，Term {자판기, 自動販賣機}，Units 三筆各有逐單位 ID；ID 不符合 acceptedIDPattern
func TestReviewTermGroupKeepsAcceptedUnits(t *testing.T) // accepted.yaml 有其中一個單位 → 該單位 Accepted 非 nil、不在 Keys
func TestReviewNewTermEvidenceListsTheUnitsItHits(t *testing.T)
func TestAcceptExpandsATermGroup(t *testing.T) // lct accept termgroup/… --reason x → accepted.yaml 新增各單位的 warning/term-missing/… ID
```
`contract_test.go`：review-page 的必備句說明 `termgroup/…`（保留＝accept、改用指定譯法＝`requeue --to-file` 附提示、詞彙表太寬＝改 `glossary/terms.yaml`）。

- [ ] **Step 2: 跑測試，確認失敗**

Run: `go test ./internal/site/ ./cmd/lct/ -run 'TermGroup|TermMissing|NewTerm|Accept|Contract'`
Expected: FAIL

- [ ] **Step 3: 實作**

- [ ] **Step 4: 跑測試，確認通過**

Run: `set -o pipefail; go vet ./... && go test ./...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/site cmd/lct .claude/skills/review-page
git commit -m "feat: the review data groups missing glossary renderings by term"
```

---

### Task 8: 審核頁畫面

**Files:**
- Modify: `site/src/types.ts`、`site/src/review/WarningCard.tsx`、`site/src/review/SuggestionCard.tsx`、`site/src/styles.css`
- Test: `site/src/review/ReviewView.test.tsx`（或新的 `WarningCard.test.tsx`）

**Interfaces:**
- Consumes: Task 7 的 JSON（`WarningItem.term`、`WarningItem.units`；`term-new` 建議的 evidence）。
- Produces: TS 型別 `TermRef {kr; zh}`、`TermUnit {key; id; kr; zh; note?; accepted?}`；`WarningItem` 加 `term?`、`units?`。

- [ ] **Step 1: 寫失敗的測試**

- `term-missing` 群組卡片：標題「詞彙表：자판기 → 自動販賣機（3 處）」；每個單位一列，顯示原文（標出原詞）、譯文、譯者 note；每列有複製逐單位 ID 的按鈕，卡片有複製整組 ID 的按鈕；已確認的單位預設隱藏（沿用「顯示已確認」的開關）。
- `term-new` 建議卡片：標題「暫定詞：우레 → 雷鳴」；列出登記的句子、衝突的譯法、命中的句子並標「有用／未用」。
- 語言勾選與「顯示原始標籤」照常影響原文與譯文的顯示。

- [ ] **Step 2: 跑測試，確認失敗**

Run: `cd site && npm test`
Expected: FAIL

- [ ] **Step 3: 實作**（沿用 `WarningCard`、`SuggestionCard` 既有的樣式與複製元件）

- [ ] **Step 4: 跑測試，確認通過**

Run: `cd site && npm test && npm run typecheck && npm run build`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add site/src
git commit -m "feat(site): the review page shows term groups and provisional terms"
```

---

### Task 9: 真實資料驗收與文件

**Files:**
- Modify: `README.md`（新的檢查、暫定詞、合批與發放順序、審核頁的兩種項目）
- 驗收只在 scratchpad 的隔離 clone 進行，不動 `/home/user/limbus-translate/.work/`，不 push

- [ ] **Step 1: 隔離 clone 與真實資料**：照 Task 9 of `docs/superpowers/plans/2026-11-05-update-history.md` 的做法建 clone（`origin/raw`、`origin/tw`、所有 tag），`go build -o bin/lct ./cmd/lct`。

- [ ] **Step 2: 合批與順序**：在 clone 的 `.work/tw` 用 `bin/lct requeue` 把 raw-20261105 這次 Claude 翻的 71 個單位放回待譯（key 從 `.meta/units/*.jsonl` 中 `since == "raw-20261105"` 且 `origin == "claude"` 取得），然後重複 `bin/lct batch next --json` 直到沒有批次（不翻譯）。
Expected：第一批是 `BattleKeywords_Mirror8.json` 或 `Bufs_Mirror8.json`；非劇情批次數明顯少於原本的 15；劇情批次各只有一個檔。

- [ ] **Step 3: 審核頁資料**：`bin/lct site --out <dir>`。
Expected：`review/warnings.json` 的 `term-missing` 依詞分組，「자판기 → 自動販賣機」一組約 19 個單位；沒有逐句數千筆的 `term-missing` 項目；記下群組數與單位總數。建置網站並以 Playwright 截審核頁的 `term-missing` 群組（1280px、400px）。

- [ ] **Step 4: 文件與完整檢查**

Run: `set -o pipefail; go vet ./... && go test ./... && cd site && npm test && npm run typecheck && npm run build`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add README.md
git commit -m "docs: glossary checks, provisional terms and merged batches"
```
