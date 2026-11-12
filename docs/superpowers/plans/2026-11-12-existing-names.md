# 既有譯名表 實作計畫

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 翻譯前把既有譯文用過的名稱與同句譯文帶給譯者，送出時只警告與既有譯名不同的名稱、把與舊譯不同的暫定詞記下但不生效，翻譯後在建置報告、審核 issue 與建議（`name/<kr>`）列出與既有譯名不同的新譯，由人決定。

**Architecture:** 新套件 `internal/existing` 放名稱表、同句索引、舊譯比對與 `name-differs` 判斷，全是純函式加 `.meta` 讀寫；`lct plan` 每個 raw tag 產生一次 `.meta/names.json` 與 `.meta/same-text.json`；批次（`batch.Next`）、送出（`batch.Submit`）、建置（`build.Run`）、建議（`suggest.Run`）與審核頁（`site.BuildReview`）各自讀表並在使用時套用詞彙表排除。「這次翻譯的單位」以紀錄新增的 `written`（Submit 寫下時的 raw tag）判斷。

**Tech Stack:** Go（既有 `internal/*` 套件）、Preact 網站（`site/`，只改兩個名稱對照）、Markdown skill 與提示詞。

**Spec:** `docs/superpowers/specs/2026-11-12-existing-names-design.md`（使用者 2026-11-12 核可）。執行者要讀 spec 與本計畫。

## Global Constraints

- 提供資訊、不強制：`name-differs` 只警告，不退回；`check: hard` 與 `forbidden` 照舊（spec「非目標」）
- 名稱表「每個 raw tag 只產生一次」：檔案記錄它的 raw tag，同一個 raw tag 的 plan 沿用、不重算；同句索引同此（裁定見 Task 4）
- 「詞彙表的排除在使用時套用（批次、送出、報告時對照當下的詞彙表），不寫進檔案」
- 名稱表來源：名稱欄位的單位，「有紀錄；來源不是 copy；譯文不等於 KR、不含諺文；KR 去掉前後空白後至少 2 個字」。名稱欄位預設 `name`、`abName`、`displayName`、`nickName`，列在 `lct.yaml` 的 `names: { fields: [...] }`，不寫死在程式
- 同一 KR 依官方 JP、EN 分成變體（「JP 相同，或 EN 不分大小寫相同」）；每個變體記錄各種譯法與單位數、主要譯法（最多者）、2～3 個例子的 key
- 文中提及：「只帶至少 3 個字（或含空格）的名稱，以及被引號括起來的名稱（`'…'`、`‘…’`、`「…」`、`"…"`）」，比對方式同詞彙表 `match: word`，長詞優先；每批上限 40 項
- `name-differs` 條件全部成立才警告：單位是名稱欄位；KR 整段等於表中的名稱；JP 或 EN 對得上某個變體；譯文去掉標籤後不是該變體的任何已有譯法；`note` 沒有寫出這個 KR
- 暫定詞與舊譯（1b）：「舊譯至少 5 個單位、而且含暫定譯名的不到一半」視為與舊譯不同；「名稱表有同一 KR 時，直接用名稱表的既有譯名判斷」；送出結果附 3 個舊譯例句（KR、TW）；照樣記下但不生效
- 同句參考：`refs["same_text"]`，KR 至少 `ShortTextRunes`（10）個字，每單位最多 2 種譯法，附檔名與說話者；索引「只收 KR 在兩個以上的組都出現的句子」
- 不自動改譯文、不自動重翻、不自動改規則；不改排程（`update.yml` 的 cron）
- 狀態（待譯清單、翻譯紀錄、批次租約）只經 `lct` 修改；`raw`、`tw` 分支與 Release 只由 CI 寫入；不碰 `.work/`
- skill 與 `prompts/` 不寫模型名稱
- 提交前 `go vet ./... && go test ./...`；動到 `site/` 時 `cd site && npm ci && npm test && npm run build && npm run typecheck`
- commit 訊息結尾加 session 規定的兩行（`Co-Authored-By`、`Claude-Session`）；commit 訊息的其他部分、程式碼、文件與 skill 不寫模型名稱或識別

## Review Focus

- **第一次部署**：tw 沒有 `names.json`，但目前的 raw tag 已經 plan 過、也翻完了。plan 要照常從當下的譯文產生名稱表，不出錯、不改待譯清單；沒有表的批次、送出、建置、審核頁一律當成空表（Task 4 `TestPlanWritesNamesWhenMissingForPlannedRawTag`，Task 5 `TestNextWithoutNameTable`）
- **舊版 lct 寫的檔**：沒有 `written` 的紀錄、沒有 `name_diffs`／`term_diffs`／`new` 的 `build.json`、沒有 `old` 的暫定詞，讀取不出錯，報告與建議照常（Task 1 `TestRecordWithoutWrittenLoads`，Task 8 `TestInfoWithoutNameDiffsDecodes`）
- **輸出位元組要穩定**：同樣的單位以不同順序輸入，`names.json`、`same-text.json` 的位元組相同（不依 map 迭代順序）（Task 2 `TestCollectIsDeterministic`，Task 3 `TestCollectSameTextIsDeterministic`）
- **單位沒有對齊的 JP、EN**：名稱欄位的單位缺 JP、EN 時不警告 `name-differs`，也不會因 nil 當掉（Task 6 `TestDiffersWithoutJPEN`）
- **上限截斷**：故事批次 120 個單位、提到的名稱超過 40 項時，名稱欄位整段相符的項目先保留，再依長度取提及的名稱（Task 5 `TestBatchNamesCapKeepsExactFirst`）

---

## 檔案

- 新增 `internal/existing/`：
  - `names.go`：名稱表的型別、`Collect`、讀寫
  - `index.go`：使用時的索引（詞彙表排除、已不採用）、`Lookup`、`Mentions`
  - `differs.go`：`name-differs`
  - `old.go`：暫定詞與舊譯的比對（1b）
  - `sametext.go`：同句索引
  - 各自的 `_test.go`
- 修改：
  - `internal/store/store.go`、`internal/store/terms.go`：`Record.Written`、`ProvisionalTerm.Old`
  - `internal/config/config.go`、`lct.yaml`：`names.fields`
  - `internal/plan/plan.go`：產生兩個 `.meta` 檔
  - `cmd/lct/plan.go`：傳入名稱欄位
  - `internal/batch/batch.go`、`next.go`、`submit.go`、`twins.go`：`Env` 的新欄位、批次 `names` 與 `same_text`、`name-differs`、1b、`ActiveProvisional`
  - `cmd/lct/batch.go`：載入表、索引與加強名稱
  - `schemas/batch.schema.json`、`prompts/translate.md`
  - `internal/build/build.go`、`report.go`、`issue.go`：報告資料、分節、審核 issue
  - `cmd/lct/build.go`：載入索引
  - `internal/suggest/suggest.go`：種類 `name`
  - `internal/site/review.go`、`cmd/lct/site.go`：審核頁的 `name-differs`
  - `site/src/review/names.ts`
  - `.claude/skills/{translate-pending,review,rules-review,review-page,term-decision}/SKILL.md`
  - `cmd/lct/contract_test.go`、`README.md`

---

### Task 1: 紀錄的寫入時點與暫定詞的舊譯標記

**Files:**
- Modify: `internal/store/store.go`（`Record`）、`internal/store/terms.go`（`ProvisionalTerm`）
- Modify: `internal/batch/submit.go`（`claudeRecord`）、`internal/plan/plan.go`（同句重用，約 1067 行）
- Test: `internal/store/store_test.go`、`internal/store/terms_test.go`、`internal/batch/batch_test.go`、`internal/plan/plan_test.go`

**Interfaces:**
- Produces:
  - `store.Record.Written string` `json:"written,omitempty"`：Submit 寫下這筆 origin claude 譯文時的 raw tag；較舊的 lct 寫的、其他來源的紀錄是 `""`
  - `func (r store.Record) TranslatedIn(rawTag string) bool`：`rawTag != "" && r.Origin == OriginClaude && r.Written == rawTag`
  - `store.OldUse struct { Existing string `json:"existing,omitempty"`; Units int `json:"units"`; With int `json:"with"` }`
  - `store.ProvisionalTerm.Old *OldUse` `json:"old,omitempty"`：有值＝登記時判定與既有譯名或舊譯不同，記下但不生效（Task 7 寫入與讀取）

- [ ] **Step 1: 寫失敗的測試**
  - `TestRecordWrittenRoundTrip`：存一筆 `Written: "raw-20261112"` 的紀錄再讀回，相等；另一筆 `Written` 空的紀錄，檔案位元組裡沒有 `"written"`
  - `TestRecordWithoutWrittenLoads`：手寫一行沒有 `written` 的舊格式 jsonl，`LoadRecords` 成功、`Written == ""`
  - `TestRecordTranslatedIn`：claude＋`Written` 等於 tag → true；tag 空、`Written` 不同、origin paratranz 但 `Written` 相同 → false
  - `TestProvisionalTermOldRoundTrip`：`AppendTerms` 一筆有 `Old{Existing: "挑釁值", Units: 95, With: 0}`、一筆沒有，`LoadTerms` 讀回相等；沒有的那行沒有 `"old"`
  - `TestSubmitRecordsWritten`（batch）：一個 requeue 的單位，原紀錄 `Since: "raw-old"`，Env `RawTag: "raw-new"`；送出收下後紀錄 `Since == "raw-old"`、`Written == "raw-new"`；同批的雙胞胎單位（`Reused`）也是 `Written == "raw-new"`
  - `TestReuseKeepsWritten`（plan）：同組來源紀錄 `Written: "raw-a"`，plan 重用到新單位，新紀錄 `Written == "raw-a"`
- [ ] **Step 2: 跑測試，確認失敗**
  Run: `go test ./internal/store/ ./internal/batch/ ./internal/plan/ -run 'Written|TranslatedIn|TermOld|ReuseKeeps'`
  Expected: FAIL（欄位、方法不存在）
- [ ] **Step 3: 實作**：加欄位與方法；`claudeRecord` 設 `Written: s.env.RawTag`（雙胞胎走同一個函式）；plan 重用的紀錄設 `Written: s.rec.Written`。fallback 紀錄不設
- [ ] **Step 4: 跑測試，確認通過**
  Run: `go vet ./... && go test ./...`
  Expected: PASS（既有測試若比對整筆紀錄而失敗，依新欄位更新期望值，不改行為）
- [ ] **Step 5: Commit**：`store：紀錄記下譯文寫入時的 raw tag；暫定詞可標記與舊譯不同`

---

### Task 2: 名稱表與索引（`internal/existing`）

**Files:**
- Create: `internal/existing/names.go`、`internal/existing/index.go`、`internal/existing/names_test.go`、`internal/existing/index_test.go`
- Modify: `internal/config/config.go`、`internal/config/config_test.go`、`lct.yaml`

**Interfaces:**
- Consumes: `store.Origin*`、`store.Store.ReadMetaJSON/WriteMetaJSON`、`names.HasHangul`、`validate.StripTags`、`validate.MatchText`、`rules.Term{Match: rules.MatchWord}.OccursIn`、`rules.DropCovered`、`rules.NewRuneSet(...).MayStartTerm`
- Produces:
  ```go
  const NamesFile = "names.json" // under .meta
  type Table struct {
      RawTag string   `json:"raw_tag"`
      Fields []string `json:"fields"`
      Names  []Name   `json:"names"` // sorted by KR
  }
  type Name struct {
      KR       string    `json:"kr"`       // trimmed
      Variants []Variant `json:"variants"` // most units first, then JP, then EN
  }
  type Variant struct {
      JP         string      `json:"jp,omitempty"`
      EN         string      `json:"en,omitempty"`
      Renderings []Rendering `json:"renderings"` // most units first, then ZH; [0] is the main one
      Examples   []string    `json:"examples"`   // at most 3 keys
  }
  type Rendering struct {
      ZH    string `json:"zh"`
      Units int    `json:"units"`
  }
  type Source struct{ Key, Field, KR, ZH, Origin, JP, EN string } // JP, EN "" when none aligned
  func Collect(rawTag string, fields []string, units []Source) *Table
  func LoadTable(st *store.Store) (*Table, error) // nil, nil when .meta/names.json is absent
  func SaveTable(st *store.Store, t *Table) error
  func SuggestionID(kr string) string // "name/" + kr

  type Index struct{ /* unexported */ }
  func NewIndex(t *Table, r *rules.Rules, declined map[string]bool) *Index // t nil → empty index; declined holds suggestion ids
  func (ix *Index) RawTag() string
  func (ix *Index) Fields() []string
  func (ix *Index) Lookup(kr string) *Name               // by strings.TrimSpace(kr); nil when none
  func (ix *Index) IsName(field, kr string) bool         // field is a name field and Lookup(kr) != nil
  func (ix *Index) Declined(kr string) bool              // declined[SuggestionID(kr)]
  func (ix *Index) Mentions(kr string, always map[string]bool) []*Name
  func (n *Name) Has(zh string) bool                      // zh (MatchText) contains any rendering of any variant
  ```
  所有 `*Index` 方法在 nil 接收者上回傳零值。
- `config.Config.Names Names` `yaml:"names"`；`config.Names struct { Fields []string `yaml:"fields"` }`

`Collect` 的規則（spec 1）：
- 先依 `Key` 排序輸入；只收 `Field` 在 `fields` 裡、`Origin` 不是 `copy`、`fallback`、`""` 的單位
- KR 取 `strings.TrimSpace`，至少 2 個 rune；譯文取 `strings.TrimSpace(validate.StripTags(ZH))`，不得是空字串、不得等於 KR、不得含諺文
- 變體鍵：`(TrimSpace(JP), strings.ToLower(TrimSpace(EN)))`；`EN` 存第一個（依 key）遇到的寫法
- `Examples`：該變體主要譯法的單位 key（依 key 排序）在前，其他譯法的接著，取前 3 個
- 詞彙表排除不在 `Collect` 做（Global Constraints）

`Mentions` 的規則（spec 2）：
- `plain := validate.MatchText(kr)`；名稱 `n` 入選要同時：`rules.Term{KR: n.KR, Match: rules.MatchWord}.OccursIn(plain)`；而且 `utf8.RuneCountInString(n.KR) >= 3`、或 `n.KR` 含空白、或 `plain` 含 `'`+KR+`'`、`‘`+KR+`’`、`「`+KR+`」`、`"`+KR+`"` 之一、或 `always[n.KR]`
- 結果依 KR 的 rune 數由多到少、再依 KR 排序；用 `rules.DropCovered(plain, …)` 拿掉每一處都被較長名稱蓋住的
- 先用 `rules.NewRuneSet(plain).MayStartTerm` 過濾，避免每個名稱都做字串搜尋

- [ ] **Step 1: 寫失敗的測試**
  - `TestCollectGroupsVariantsByJPEN`：`노을빛 검`，JP `茜さす剣`、EN `Glow Blade` 的 3 個單位（`霞光劍刃`×2、`晚霞之劍`×1），JP `夕焼け`、EN `Sunset` 的 1 個單位（`夕陽`）→ 一個 `Name`、兩個變體；第一個變體 `Renderings == [{霞光劍刃 2} {晚霞之劍 1}]`，`Examples` 是兩個 `霞光劍刃` 的 key（排序）再接 `晚霞之劍` 的
  - `TestCollectENIgnoresCase`：EN `Glow Blade` 與 `glow blade`（JP 相同）是同一個變體，`EN` 是 key 較小者的寫法
  - `TestCollectSkips`：origin copy、fallback、`ZH == KR`、譯文含諺文、KR 去空白後 1 個字、欄位不在 `fields`、譯文去標籤後是空的，都不收
  - `TestCollectStripsTags`：`<b>霞光劍刃</b>` 算作 `霞光劍刃`
  - `TestCollectIsDeterministic`：同一組輸入打亂順序兩次，`json.Marshal` 結果相同
  - `TestTableRoundTrip`：`SaveTable` 後 `LoadTable` 相等；沒有檔案時 `LoadTable` 回 `nil, nil`
  - `TestIndexExcludesGlossary`：詞彙表有 `격돌` 條目（任何狀態）→ `Lookup("격돌") == nil`、`Mentions` 不含它；`Lookup(" 노을빛 검 ")` 找得到
  - `TestIndexDeclined`：`declined{"name/노을빛 검": true}` → `Declined("노을빛 검")` 為真，`Lookup` 仍找得到
  - `TestMentions`：
    - `"노을빛 검으로 벤다"` → `[노을빛 검]`
    - `"붉은노을빛 검"`（前面是韓文音節）→ 空
    - 兩字的 `회피`：`"회피를 한다"` → 空；`"'회피' 스킬"` → `[회피]`；`always{"회피": true}` → `[회피]`
    - 名稱 `가르는 검`（含空白）與 `새벽을 가르는 검` 都在表中，`"새벽을 가르는 검"` → 只有長的
  - `TestMentionsLiteral`：名稱 `E.G.O 개화 (정오)` 以字面比對，不當成正規表示式
  - `TestNameHas`：譯文含任一變體的任一譯法（去標籤後）→ true
  - `TestNilIndex`：`var ix *Index` 的每個方法回零值不 panic
  - `TestLoadNames`（config）：`names: { fields: [name, abName] }` 讀成 `Names.Fields`
- [ ] **Step 2: 跑測試，確認失敗**
  Run: `go test ./internal/existing/ ./internal/config/`
  Expected: FAIL（套件與欄位不存在）
- [ ] **Step 3: 實作** 上列型別與函式；`lct.yaml` 加 `names: { fields: [name, abName, displayName, nickName] }`（一行，放在 `batch:` 之後）
- [ ] **Step 4: 跑測試，確認通過**
  Run: `go vet ./... && go test ./...`
  Expected: PASS
- [ ] **Step 5: Commit**：`existing：既有譯名表與使用時的索引；lct.yaml 列出名稱欄位`

---

### Task 3: 同句索引（`internal/existing`）

**Files:**
- Create: `internal/existing/sametext.go`、`internal/existing/sametext_test.go`

**Interfaces:**
- Produces:
  ```go
  const SameTextFile = "same-text.json" // under .meta
  type SameText struct {
      RawTag string                     `json:"raw_tag"`
      Texts  map[string][]SameRendering `json:"texts"` // KR text → one rendering per group, by group
  }
  type SameRendering struct {
      Group   string `json:"group"`
      ZH      string `json:"zh"`
      File    string `json:"file"`
      Speaker string `json:"speaker,omitempty"`
  }
  type TextUnit struct{ Key, Group, File, Speaker, KR, ZH string } // ZH "" when the unit has no usable translation
  func CollectSameText(rawTag string, units []TextUnit) *SameText
  func (s *SameText) For(kr, group string) []SameRendering // nil receiver → nil
  func SameTextRef(rs []SameRendering) string
  func LoadSameText(st *store.Store) (*SameText, error) // nil, nil when absent
  func SaveSameText(st *store.Store, s *SameText) error
  const MaxSameText = 2
  ```

規則：
- `CollectSameText` 不看長度（呼叫者只傳 KR 夠長的單位，見 Task 4）。依 `Key` 排序輸入；依 KR 分；同一 KR 的單位分屬至少 2 個 `Group`、而且至少一個有 `ZH` 才收
- 每個有譯文的組一筆：該組最多單位用的 `ZH`（同數取 key 較小者先出現的），`File`、`Speaker` 取用這個譯法、key 最小的單位；依 `Group` 排序
- `For(kr, group)`：該 KR 中 `Group != group` 的譯法，`ZH` 不重複，依順序取前 `MaxSameText` 個
- `SameTextRef`：每種一行，`<file>（<speaker>）：<zh>`；`Speaker` 空時是 `<file>：<zh>`；行之間用 `\n`；沒有譯法時回 `""`

- [ ] **Step 1: 寫失敗的測試**
  - `TestCollectSameTextNeedsTwoGroups`：同一句只在一個組出現 → 不收；在兩個組、只有一個組有譯文 → 收一筆；兩個組都沒有譯文 → 不收
  - `TestCollectSameTextMainRendering`：同組 3 個單位譯法 `A`×2、`B`×1 → 該組的 `ZH == "A"`，`File` 是 `A` 中 key 最小者的檔
  - `TestSameTextForOtherGroups`：三個組各有 `A`、`A`、`C`，`For(kr, 第一組)` → 第二組的 `A` 與 `C`（`A` 只出現一次），`For` 自己的組不回
  - `TestSameTextForAtMostTwo`：四個組四種譯法 → 回 2 個
  - `TestSameTextRef`：`[{File: "StoryData/S1.json", Speaker: "philip", ZH: "為了比誰都悲傷的我"}, {File: "PersonalityVoiceDlg/V.json", ZH: "為了最悲傷的我"}]` → `"StoryData/S1.json（philip）：為了比誰都悲傷的我\nPersonalityVoiceDlg/V.json：為了最悲傷的我"`
  - `TestCollectSameTextIsDeterministic`：打亂輸入，`json.Marshal` 相同
  - `TestSameTextRoundTrip`：存讀相等；沒有檔案回 `nil, nil`；`var s *SameText; s.For(...)` 回 nil
- [ ] **Step 2: 跑測試，確認失敗**
  Run: `go test ./internal/existing/ -run SameText`
  Expected: FAIL
- [ ] **Step 3: 實作**
- [ ] **Step 4: 跑測試，確認通過**
  Run: `go vet ./... && go test ./internal/existing/`
  Expected: PASS
- [ ] **Step 5: Commit**：`existing：同句譯文索引`

---

### Task 4: `lct plan` 每個 raw tag 產生一次名稱表與同句索引

**Files:**
- Modify: `internal/plan/plan.go`（`Inputs`、`Summary`、`Summary.String`、`Run`）、`internal/batch/batch.go`（`SameTextGroup`）、`cmd/lct/plan.go`
- Test: `internal/plan/plan_test.go`、`internal/batch/batch_test.go`、`cmd/lct/plan_test.go`

**Interfaces:**
- Consumes: Task 2 `Collect`、`LoadTable`、`SaveTable`；Task 3 `CollectSameText`、`LoadSameText`、`SaveSameText`；`batch.TextGroupOf`、`batch.SpeakerOf`、`batch.ShortTextRunes`、`align.NewFileAligner`
- Produces:
  - `plan.Inputs.NameFields []string`（lct.yaml 的 `names.fields`；空＝名稱表沒有條目，照樣寫檔）
  - `func batch.SameTextGroup(g TextGroup) string`：`g.Family + "|" + g.Field + "|" + g.Speaker`
  - `plan.Summary.NameTable NameTableSummary` `json:"name_table"`；`type NameTableSummary struct { RawTag string `json:"raw_tag"`; Built bool `json:"built"`; Names int `json:"names"`; SameTexts int `json:"same_texts"` }`。`Summary.String` 加一行：產生時 `name table <raw tag>: built, N names, M same texts`；沿用時 `name table <raw tag>: kept, N names, M same texts`

做法：
- 在 `Run` 的鎖內、`SaveApplied` 之後、`WriteMetaJSON(File, …)` 之前做（文件註解的寫入順序一併更新）。兩個檔各自判斷：已有檔案且 `RawTag == p.sum.RawTag` 就沿用（只讀出來數條目），否則重算並寫入
- 來源是 `p.files` 的每個單位與它在 `p.records` 的紀錄，可用的紀錄要：`KRSha` 等於目前 KR 的 sha、origin 不是 copy 或 fallback、沒有 `needs_review` 旗標
  - 名稱表：`u.Field` 在 `NameFields` 的單位，JP、EN 用該檔的 `align.NewFileAligner(...).Get(u.Pointer)` 取（只為有名稱欄位紀錄的檔建 aligner；`LangAligned` 為 nil 時是 `""`）
  - 同句索引：`utf8.RuneCountInString(u.KR) >= batch.ShortTextRunes` 的所有單位（有沒有紀錄都要傳，沒有可用紀錄的 `ZH` 為 `""`），`Group` 是 `batch.SameTextGroup(batch.TextGroupOf(in.Rules, f.doc, u))`，`Speaker` 是 `batch.SpeakerOf(...)`
- 裁定（記進 ledger）：同句索引與名稱表一樣每個 raw tag 只算一次——spec 只說「在 plan 時建立」，一起算省一次全包掃描，代價是同一次更新內新翻的句子不會成為後面批次的同句參考（同組的仍由雙胞胎與重用處理）

- [ ] **Step 1: 寫失敗的測試**
  - `TestSameTextGroup`（batch）：`TextGroup{Family: "f", Field: "content", Speaker: "philip", KR: "x"}` → `"f|content|philip"`
  - `TestPlanWritesNameTable`：fixture 的 KR 檔有 `name` 欄位與紀錄（paratranz），`NameFields: []string{"name"}`；plan 後 `existing.LoadTable` 的 `RawTag` 是目前的 tag、含該名稱與它的 JP、EN；`Summary.NameTable.Built` 為真
  - `TestPlanKeepsNameTableOfSameRawTag`：plan 一次，改一筆名稱紀錄的 `ZH`，同一個 raw tag 再 plan：`names.json` 位元組不變，`Built` 為假、`Names` 與第一次相同
  - `TestPlanRebuildsNameTableForNewRawTag`：換到新的 raw tag（既有的 transition fixture）後 plan，表的 `RawTag` 是新的
  - `TestPlanWritesNamesWhenMissingForPlannedRawTag`：`plan.json` 已是目前的 raw tag、沒有 `names.json` 與 `same-text.json`；再 plan → 兩個檔都寫出，待譯清單與紀錄不變
  - `TestPlanWritesSameTextIndex`：同一句 ≥10 字的 KR 在兩個組（欄位不同或說話者不同），一個有紀錄 → 索引有這句；9 個字的不收；只在一個組的不收
  - `TestPlanCommandPassesNameFields`（cmd/lct）：測試 repo 的 `lct.yaml` 有 `names.fields`，`lct plan --json` 的 `name_table.names` 大於 0
- [ ] **Step 2: 跑測試，確認失敗**
  Run: `go test ./internal/plan/ ./internal/batch/ ./cmd/lct/ -run 'NameTable|SameText|NameFields|NamesWhenMissing'`
  Expected: FAIL
- [ ] **Step 3: 實作** 上述做法；`cmd/lct/plan.go` 傳 `NameFields: cfg.Names.Fields`
- [ ] **Step 4: 跑測試，確認通過**
  Run: `go vet ./... && go test ./...`
  Expected: PASS
- [ ] **Step 5: Commit**：`plan：每個 raw tag 產生一次既有譯名表與同句索引`

---

### Task 5: 批次帶上 `names` 與 `refs.same_text`，提示詞與 schema

**Files:**
- Modify: `internal/batch/batch.go`（`Env`、`Batch`、新型別）、`internal/batch/next.go`（`assemble`）、`cmd/lct/batch.go`（`baseEnv`）
- Modify: `schemas/batch.schema.json`、`prompts/translate.md`
- Test: `internal/batch/batch_test.go`、`cmd/lct/batch_test.go`、`cmd/lct/contract_test.go`

**Interfaces:**
- Consumes: Task 2 `Index`（`IsName`、`Lookup`、`Mentions`、`Fields`）；Task 3 `SameText.For`、`SameTextRef`；Task 4 `SameTextGroup`；Task 1 `ProvisionalTerm.Old`
- Produces:
  - `batch.Env.Names *existing.Index`、`batch.Env.SameText *existing.SameText`、`batch.Env.NameBoost map[string]bool`（暫定詞 `Old.Existing != ""` 的 KR：即使不滿 3 個字、沒有引號，也帶名稱表的那一項，spec 1b「之後的批次改帶名稱表的那一項」）
  - `batch.Batch.Names []NameRef` `json:"names"`（永遠不是 nil）
  - ```go
    type NameRef struct {
        KR     string               `json:"kr"`
        ZH     string               `json:"zh"`    // the main rendering of the variant
        Count  int                  `json:"count"` // units of ZH
        Others []existing.Rendering `json:"others,omitempty"`
        JP     string               `json:"jp,omitempty"`
        EN     string               `json:"en,omitempty"`
        Units  []string             `json:"units"` // keys of the batch's units it is for, in batch order
    }
    const MaxNames = 40
    const maxNameVariants = 3
    const RefSameText = "same_text"
    ```

做法：
- 每個單位：`env.Names.IsName(u.Field, u.KR)` 為真時帶 `Lookup(u.KR)`（整段相符）；否則帶 `Mentions(u.KR, env.NameBoost)`。名稱欄位但 KR 不是表中名稱的單位，當一般單位看（裁定：spec 沒說，照「其他單位」處理最寬鬆、不漏）
- 一個名稱展開成前 `maxNameVariants` 個變體，各一個 `NameRef`；同一個 (KR, 變體) 只一項，`Units` 累積
- 超過 `MaxNames` 時保留順序：先整段相符的項目，再提及的項目；各自依 KR 的 rune 數由多到少、KR、變體順序（裁定：spec 只說「長的優先」；整段相符的是送出時會警告的，先保留）
- `same_text`：`rs := env.SameText.For(u.KR, batch.SameTextGroup(TextGroupOf(env.Rules, fd.doc, u)))`，有就放進該單位的 `Refs[RefSameText]`（已有的 `llc`、`ltm`、`adjustment` 不動）
- `baseEnv` 讀 `existing.LoadTable`、`existing.LoadSameText`，以 `existing.NewIndex(table, r, declined)` 建索引（declined 是 `rules/declined.yaml` 的 id 集合，`activeProvisionalRecords` 已讀過，避免重讀），`NameBoost` 從 `st.LoadTerms()` 的結果算（同樣只讀一次）
- schema：`names` 加進 `properties` 與 `required`，項目描述照 `NameRef`；`units.items.properties.refs` 的描述提到 `same_text`
- `prompts/translate.md`，三處，照下列文字：
  1. 「名稱與詞彙」的 `keywords` 那一點之後加一點：
     > - `names` 是本專案既有譯文用過的名稱（技能、人格、E.G.O、效果等名稱欄位的譯法），不是規則：`zh` 是最多單位用的譯法、`count` 是它的單位數，`others` 是其他譯法與單位數，`jp`、`en` 是那些單位的官方 JP、EN，`units` 是本批 KR 用到它的單位。指的是同一個東西（比對單位與該項的 `jp`、`en`，以及單位的 `field`）就用 `zh`；同一個 `kr` 有幾項時，選 `jp`、`en` 對得上的那一項。語境是別的意思（例如劇情裡的一般用語）就照語境翻，並在該單位的 `note` 寫出 KR 原詞與理由。名稱欄位的譯文與對得上的那一項不同、`note` 又沒有寫出 KR 原詞時，送出會警告（`name-differs`），不會退回
  2. 把「人名、地名、組織、技能、狀態等名稱以 `keywords`、`glossary`、指引與 `context` 為準，不取自 `refs`」改成：
     > - 人名、地名、組織、技能、狀態等名稱以 `keywords`、`glossary`、`names`、指引與 `context` 為準，順序表示優先，不取自 `refs`
  3. 「參考譯文 `refs`」最後加一點：
     > - `same_text` 是同一句 KR 在別的檔案（不同族群、欄位或說話者）已有的譯文，每行是「檔案（說話者）：譯文」：同一句話盡量譯得一致，語氣與稱呼依這個單位的說話者與語境調整；它是參考，不是答案

  並在「詞彙表沒有、但應該統一譯法的詞」那一點的第一句後加：「`names` 已有的名稱不必登記。」

- [ ] **Step 1: 寫失敗的測試**
  - `TestBatchNamesExactName`：名稱欄位單位 KR `노을빛 검`，表中兩個變體 → 批次 `Names` 有兩項，`Units` 都是該 key，第一項 `ZH == "霞光劍刃"`、`Count == 2`、`Others == [{晚霞之劍 1}]`
  - `TestBatchNamesMentions`：`desc` 單位提到 `노을빛 검` 與兩字的 `회피` → 只帶 `노을빛 검`；KR 寫 `'회피'` → 兩個都帶
  - `TestBatchNamesBoost`：`NameBoost{"개화": true}`，KR `개화한다` 不含引號 → 帶 `개화`
  - `TestBatchNamesGlossaryExcluded`：詞彙表有 `노을빛 검` → 不帶
  - `TestBatchNamesCapKeepsExactFirst`：一批 2 個整段相符的名稱、45 個提及的名稱 → 共 40 項，兩個整段相符的都在，提及的依長度取
  - `TestBatchSameTextRef`：索引有這句在別組的譯法 → `Refs["same_text"]` 等於 `SameTextRef` 的結果；原有的 `Refs["llc"]` 仍在
  - `TestNextWithoutNameTable`：`Env.Names`、`Env.SameText` 為 nil → `Names` 是 `[]`、沒有 `same_text`
  - `TestBaseEnvLoadsNames`（cmd/lct）：tw 有 `names.json`、`same-text.json`，`lct batch next --json` 的批次檔有 `names`
  - `TestNamesAreExplainedToTheTranslator`（contract）：`prompts/translate.md` 含 `` `names` ``、`name-differs`、`` `same_text` `` 與「順序表示優先」；`schemas/batch.schema.json` 的 `properties.names` 是 array 且在 `required`
- [ ] **Step 2: 跑測試，確認失敗**
  Run: `go test ./internal/batch/ ./cmd/lct/ -run 'BatchNames|SameTextRef|WithoutNameTable|LoadsNames|NamesAreExplained'`
  Expected: FAIL
- [ ] **Step 3: 實作**
- [ ] **Step 4: 跑測試，確認通過**
  Run: `go vet ./... && go test ./...`
  Expected: PASS（比對整個批次 JSON 的既有測試，期望值加上 `"names": []`）
- [ ] **Step 5: Commit**：`batch：批次帶上既有譯名與同句譯文；提示詞與 schema 說明`

---

### Task 6: 送出時的 `name-differs` 警告

**Files:**
- Create: `internal/existing/differs.go`、`internal/existing/differs_test.go`
- Modify: `internal/batch/submit.go`（`unit`）
- Test: `internal/batch/batch_test.go`

**Interfaces:**
- Consumes: Task 2 `Index`、`Name`；`validate.Issue`、`validate.SeverityWarn`、`validate.ClassStyle`
- Produces:
  - `const existing.CodeNameDiffers = "name-differs"`
  - `func (ix *Index) Differs(field, kr, zh, note, jp, en string) *validate.Issue`：spec 3 的條件全部成立才回 issue，否則 nil。`ix.Declined(KR)` 時一律 nil
    - 變體：依順序找第一個 `v.JP != "" && TrimSpace(jp) == v.JP`，或 `v.EN != "" && strings.EqualFold(TrimSpace(en), v.EN)` 的
    - 譯文：`TrimSpace(validate.StripTags(zh))` 不等於該變體任何一個 `Renderings[i].ZH`
    - `note`：`strings.Contains(note, name.KR)` 就不警告
    - issue：`Code: CodeNameDiffers, Severity: warn, Class: style, Term: name.KR, TermZH: v.Renderings[0].ZH`，`Message` 用英文（同其他 issue）：`fmt.Sprintf("the existing translations name %q %q (%d units); this one says %q: use the existing name when it is the same thing, or write the KR word and the reason in the note", name.KR, main.ZH, main.Units, plainZH)`
  - `itemid.UnitWarning(CodeNameDiffers, Term, TermZH, kr, zh)` 是這個警告在審核頁與 `rules/accepted.yaml` 的 id（不改 itemid）

Submit：在收下的路徑、`len(issues) > 0` 判斷之前，`fd.aligned(langJP, pointer)`、`fd.aligned(langEN, pointer)` 取 JP、EN（nil 為 `""`），`s.env.Names.Differs(fd.units[pos].Field, kr, zh, o.Note, jp, en)` 有值就加進 `issues`。不放進 `Judge`（預覽不需要）。

- [ ] **Step 1: 寫失敗的測試**
  - `TestDiffersNeedsMatchingJPOrEN`：表 `노을빛 검` 變體 JP `茜さす剣`、EN `Glow Blade`、譯法 `霞光劍刃`。`Differs("name", "노을빛 검", "晚霞之劍", "", "茜さす剣", "")` → issue，`Term == "노을빛 검"`、`TermZH == "霞光劍刃"`；JP `夕焼け`、EN `Sunset` → nil；JP 空、EN `glow blade` → issue
  - `TestDiffersAcceptsAnyExistingRendering`：譯文是同變體的另一個既有譯法 → nil；`<b>霞光劍刃</b>` → nil
  - `TestDiffersNoteNamesKR`：note `노을빛 검：此處是劇情描寫` → nil
  - `TestDiffersOnlyNameFields`：欄位 `desc` → nil；KR 不是表中名稱 → nil
  - `TestDiffersDeclinedOrGlossary`：declined `name/노을빛 검` → nil；詞彙表有這個 KR → nil
  - `TestDiffersWithoutJPEN`：jp、en 都是 `""` → nil
  - `TestSubmitWarnsNameDiffers`（batch）：Env 有索引，名稱欄位單位的 JP 對得上、譯文不同 → `Accepted` 有它、`Warnings` 有 `code: name-differs`；note 寫出 KR 再送一次另一個單位 → 沒有這個警告
- [ ] **Step 2: 跑測試，確認失敗**
  Run: `go test ./internal/existing/ ./internal/batch/ -run 'Differs'`
  Expected: FAIL
- [ ] **Step 3: 實作**
- [ ] **Step 4: 跑測試，確認通過**
  Run: `go vet ./... && go test ./...`
  Expected: PASS
- [ ] **Step 5: Commit**：`batch：名稱欄位的譯法與既有譯名不同時警告（name-differs）`

---

### Task 7: 暫定詞與舊譯的比對（1b）

**Files:**
- Create: `internal/existing/old.go`、`internal/existing/old_test.go`
- Modify: `internal/batch/submit.go`（`keepTerms`、`SubmitResult`）、`internal/batch/batch.go`（`ActiveProvisional`、新的 `ProvisionalKRs`）、`cmd/lct/batch.go`（`activeProvisionalRecords`）
- Modify: `.claude/skills/translate-pending/SKILL.md`
- Test: `internal/batch/batch_test.go`、`cmd/lct/batch_test.go`、`cmd/lct/contract_test.go`

**Interfaces:**
- Consumes: Task 1 `Record.TranslatedIn`、`store.OldUse`、`ProvisionalTerm.Old`；Task 2 `Index.Lookup`、`Name.Has`
- Produces:
  ```go
  const MinOldUnits = 5
  type OldExample struct {
      Key string `json:"key"`
      KR  string `json:"kr"`
      ZH  string `json:"zh"`
  }
  // IsOld: a record of a translation made before rawTag's update: origin paratranz, claude or human, and not TranslatedIn(rawTag).
  func IsOld(rec store.Record, rawTag string) bool
  type OldTally struct{ /* unexported */ }
  func NewOldTally(terms []rules.Term) *OldTally // KR, ZH of each term; matched as words
  func (t *OldTally) Add(key, kr, zh string)    // an old unit: its KR text and its translation
  func (t *OldTally) Use(ix *Index, kr, zh string) (use store.OldUse, examples []OldExample, differs bool)
  ```
  - `Add`：`plain := validate.MatchText(kr)`；KR 以整詞出現（`rules.MatchWord`）就計入該詞的 `Units`，`validate.MatchText(zh)` 含該詞的 ZH 就計入 `With`
  - `Use`：`ix.Lookup(kr)` 有名稱時：`Existing` 是第一個變體的主要譯法，`differs = !name.Has(zh)`；沒有時：`differs = Units >= MinOldUnits && With*2 < Units`。`examples`：不含 ZH 的舊譯在前、含的在後，各依 key，取 3 個
  - `batch.SubmitResult.TermsDiffer []DifferingTerm` `json:"terms_differ"`（永遠不是 nil）
  - ```go
    type DifferingTerm struct {
        KR       string                `json:"kr"`
        ZH       string                `json:"zh"`
        Key      string                `json:"key"` // the unit that registered it
        Existing string                `json:"existing,omitempty"`
        OldUnits int                   `json:"old_units"`
        OldWith  int                   `json:"old_with"`
        Examples []existing.OldExample `json:"examples"`
    }
    ```
  - `func batch.ProvisionalKRs(all []store.ProvisionalTerm, r *rules.Rules) map[string]bool`：KR 至少 2 個字、詞彙表沒有、有一筆不是衝突的紀錄（不論 `Old`、不論不採用）。`ActiveProvisional` = 這些 KR 中第一筆非衝突紀錄沒有 `Old`、也沒有被不採用的

Submit：`keepTerms` 對這次要新增、而且這個 KR 還沒有任何紀錄的候選，在存之前一次算出比對：只有這時才掃描舊譯——走 `s.env.Src.KR.Files()` 的每個檔、`units.Extract`，紀錄存在、`KRSha` 等於該單位 KR 的 sha、`existing.IsOld(rec, s.env.RawTag)` 的單位才 `Add`。`differs` 的候選照樣存，但設 `Old`，並列進 `TermsDiffer`。已有紀錄的 KR（衝突或重複）不比對。

`cmd/lct/batch.go` 的 `activeProvisionalRecords`：「列出」改用 `batch.ProvisionalKRs`，讓與舊譯不同、沒有生效的暫定詞仍是 `term-new` 建議（spec 1b「翻譯後的報告列出它」）。

translate-pending skill「退回」一節的結果欄位清單之後加一句：
> `terms_differ` 是與既有譯名或舊譯不同的暫定詞（`existing` 是名稱表的既有譯名，`old_units` 是 KR 含這個詞的舊譯單位數，`old_with` 是其中含暫定譯法的，`examples` 是舊譯例句）：已記下但沒有生效，之後的批次不會帶它。已收下的單位不必重送；同一批次還沒送出的單位、之後的批次，依例句與批次的 `names` 用舊譯的寫法。

- [ ] **Step 1: 寫失敗的測試**
  - `TestOldTallyFlagsDifferent`：`도발치 → 嘲諷值`；6 個舊譯單位 KR 含 `도발치`、TW 都寫 `挑釁值` → `differs`，`Units == 6`、`With == 0`，3 個例句依 key
  - `TestOldTallyAgrees`：6 個舊譯中 4 個含暫定譯法 → 不 differs
  - `TestOldTallyTooFew`：只有 4 個舊譯、都不含 → 不 differs
  - `TestOldTallyUsesNameTable`：表中有 `격돌`（`交鋒`），暫定 `衝突` → differs、`Existing == "交鋒"`；暫定 `交鋒` → 不 differs（即使舊譯單位很少）
  - `TestOldTallyWordMatch`：暫定詞 `불주먹 사무소`；KR `큰불주먹 사무소`（前面接韓文音節）不計入，`불주먹 사무소에서` 計入
  - `TestIsOld`：paratranz、human、`Written` 為別的 tag 的 claude → true；`TranslatedIn` 的 claude、copy、fallback → false
  - `TestSubmitMarksDifferingTerm`（batch）：舊譯 5 個單位寫 `挑釁值`，送出登記 `{도발치, 嘲諷值}` → `terms.jsonl` 那筆有 `old.units == 5`、`old.with == 0`；`TermsDiffer` 有它與 3 個例句；`Terms == 1`
  - `TestDifferingTermNotInForce`：上一筆之後 `ActiveProvisional` 不含 `도발치`，`ProvisionalKRs` 含；同 KR 再登記 `挑釁值` 是衝突、不生效
  - `TestSubmitWithoutNewTermsDoesNotScan`：沒有新登記時，`TermsDiffer` 是 `[]`，結果與現在相同（既有的送出測試都要照常通過）
  - `TestDifferingTermIsStillSuggested`（cmd/lct）：上述暫定詞在 `lct suggest --json --kind term-new` 仍列出
  - contract：`TestTranslatePendingExplainsTermsDiffer`：translate-pending skill 含 `terms_differ`
- [ ] **Step 2: 跑測試，確認失敗**
  Run: `go test ./internal/existing/ ./internal/batch/ ./cmd/lct/ -run 'OldTally|IsOld|DifferingTerm|MarksDiffering|DoesNotScan|TermsDiffer'`
  Expected: FAIL
- [ ] **Step 3: 實作**（`ActiveProvisional` 的文件註解改寫成新規則）
- [ ] **Step 4: 跑測試，確認通過**
  Run: `go vet ./... && go test ./...`
  Expected: PASS
- [ ] **Step 5: Commit**：`batch：暫定詞與既有譯名或舊譯不同時記下但不生效，送出結果附舊譯例句`

---

### Task 8: 建置收集「與既有譯名不同的新譯」

**Files:**
- Modify: `internal/build/build.go`（`Options`、`Report`、`Inconsistency`、`builder`、`useRecord`、`addRendering`、`finish`、`newNotes`）、`cmd/lct/build.go`
- Test: `internal/build/build_test.go`、`cmd/lct/build_test.go`

**Interfaces:**
- Consumes: Task 1 `TranslatedIn`、`ProvisionalTerm.Old`；Task 2 `Index`（`IsName`、`Lookup`、`Mentions`、`Declined`、`Name.Has`）；Task 6 `Differs`、`CodeNameDiffers`；Task 7 `IsOld`、`OldTally`、`ProvisionalKRs`
- Produces:
  - `build.Options.Names *existing.Index`（nil＝不做名稱檢查）
  - `build.Report.NameDiffs []NameDiff` `json:"name_diffs"`、`build.Report.TermDiffs []TermDiff` `json:"term_diffs"`（都永遠不是 nil）
  - ```go
    type NameDiff struct {
        KR       string               `json:"kr"`
        Existing []existing.Rendering `json:"existing"` // every variant's renderings, summed by ZH, most units first, at most 5
        Units    []NameDiffUnit       `json:"units"`    // by key
    }
    type NameDiffUnit struct {
        Key  string `json:"key"`
        ZH   string `json:"zh"`
        Note string `json:"note,omitempty"`
        Kind string `json:"kind"` // NameDiffField or NameDiffMention
    }
    const (
        NameDiffField   = "name-differs"
        NameDiffMention = "mention"
    )
    type TermDiff struct {
        KR       string                `json:"kr"`
        ZH       string                `json:"zh"`
        Existing string                `json:"existing,omitempty"`
        OldUnits int                   `json:"old_units"`
        OldWith  int                   `json:"old_with"`
        Examples []existing.OldExample `json:"examples"`
        Keys     []string              `json:"keys"` // this update's units that use ZH for KR, by key, at most 20
    }
    ```
  - `build.Inconsistency.New int` `json:"new,omitempty"`、`build.Inconsistency.NewKeys []string` `json:"new_keys,omitempty"`：組內 `TranslatedIn(opt.RawTag)` 的單位數與 key（排序）

做法：
- `cmd/lct/build.go`：讀 `existing.LoadTable`，以 `existing.NewIndex(table, rules, declined)` 建 `Options.Names`
- 每個單位的 note：`newNotes` 已讀全部備註，另回傳 key → 該 key 所有備註以 `\n` 連接的對照
- `useRecord` 對不是 copy 的紀錄：
  - origin claude、`opt.Names.IsName(u.Field, u.KR)`：取 JP、EN（`al.Get(u.Pointer)`），`Differs` 有值且 `opt.Accepted` 沒有它的 `itemid.UnitWarning` → 和其他警告一樣計入 `Warnings`、`WarningSamples`；紀錄 `TranslatedIn(opt.RawTag)` 時另列進 `NameDiffs`（`Kind: name-differs`）
  - 紀錄 `TranslatedIn` 而不是整段相符的名稱：對 `opt.Names.Mentions(u.KR, nil)` 中沒有 `Declined`、`!n.Has(rec.ZH)` 的名稱，列進 `NameDiffs`（`Kind: mention`，`ZH` 是整個譯文）
  - `existing.IsOld(rec, opt.RawTag)`：`tally.Add(key, u.KR, rec.ZH)`；`TranslatedIn` 而且 KR 含某個不同的暫定詞、譯文含它的暫定譯法：記進那個 `TermDiff.Keys`
- 「不同的暫定詞」：`st.LoadTerms()` 中，每個 KR 第一筆非衝突紀錄有 `Old`、`RawTag == opt.RawTag`、在 `batch.ProvisionalKRs` 裡、`opt.Declined` 沒有 `term-new/<kr>` 的。`finish` 時以 `tally.Use(opt.Names, kr, zh)` 取得例句與重算的 `OldUnits`、`OldWith`；`Existing` 用 `Use` 的結果，沒有時用紀錄的 `Old.Existing`
- `addRendering(u, key, zh, isNew bool)` 累積 `New`、`NewKeys`
- `NameDiffs` 依 KR 排序；同一單位同一名稱只列一次；`TermDiffs` 依 KR

- [ ] **Step 1: 寫失敗的測試**
  - `TestBuildListsNameDiffers`：表有 `노을빛 검`（JP 對得上）；一筆這次（`Written` 等於 tag）的 claude 名稱紀錄譯作 `晚霞之劍` → `Report.Warnings["name-differs"] == 1`、`NameDiffs[0].Units[0].Kind == "name-differs"`，`Existing[0] == {霞光劍刃, n}`
  - `TestBuildNameDifferAccepted`：`opt.Accepted` 有它的 id → 不計警告、不列
  - `TestBuildListsMentions`：這次的 `desc` 單位 KR 含 `노을빛 검`、譯文沒有任何既有譯法 → `Kind: mention`；譯文含 `霞光劍刃` → 不列；舊譯單位（`Written` 空）→ 不列
  - `TestBuildSkipsDeclinedNames`：declined `name/노을빛 검` → 沒有警告、沒有項目
  - `TestBuildListsTermDiffs`：`terms.jsonl` 一筆 `도발치 → 嘲諷值`、`old` 有值、`raw_tag` 是這次；6 個舊譯寫 `挑釁值`、2 個這次的單位寫 `嘲諷值` → `TermDiffs[0]`：`OldUnits == 6`、`OldWith == 0`、3 個例句、`Keys` 是那 2 個
  - `TestBuildMarksNewInconsistency`：同組舊譯 `A`×3、這次 `B`×1 → 該組 `New == 1`、`NewKeys` 是那一個；只有舊譯的組 `New == 0`
  - `TestInfoWithoutNameDiffsDecodes`：沒有 `name_diffs`、`term_diffs`、`new` 的舊 `build.json` 能讀（`ReadInfo`）
  - `TestBuildWithoutNames`：`Options.Names` 為 nil → `NameDiffs`、`TermDiffs` 是空的，其他結果與現在相同
- [ ] **Step 2: 跑測試，確認失敗**
  Run: `go test ./internal/build/ -run 'NameDiff|Mentions|DeclinedNames|TermDiffs|NewInconsistency|WithoutNameDiffs|WithoutNames'`
  Expected: FAIL
- [ ] **Step 3: 實作**
- [ ] **Step 4: 跑測試，確認通過**
  Run: `go vet ./... && go test ./...`
  Expected: PASS
- [ ] **Step 5: Commit**：`build：收集與既有譯名不同的新譯、不同的暫定詞，標出含這次新譯的不一致組`

---

### Task 9: 報告分節與審核 issue

**Files:**
- Modify: `internal/build/report.go`（`markdown`、新的 `infoNameDiffs`、`infoInconsistent`）、`internal/build/issue.go`（`ReviewKinds`、`ReviewItems`、新的 Kind 常數）
- Modify: `.claude/skills/review/SKILL.md`
- Test: `internal/build/issue_test.go`、`internal/build/build_test.go`、`cmd/lct/contract_test.go`（`ReviewKinds` 的既有檢查）

**Interfaces:**
- Consumes: Task 8 `NameDiffs`、`TermDiffs`、`Inconsistency.New/NewKeys`、`Info.InconsistentAll`
- Produces:
  - `KindNameDiffers = "name_differs"`、`KindInconsistentNew = "inconsistent_new"`；`ReviewKinds` 依序為 needs_review、adjustment_expired、frozen_changed、frozen_fallback、new_speaker、field_change、new_family、**name_differs、inconsistent_new**、loop_error、not_translated

報告（`## 資訊` 裡，`驗證警告` 之後、`同一 KR 不同譯法` 之前）：

```
### 與既有譯名不同的新譯

- `노을빛 검`（`name/노을빛 검`）：既有 `霞光劍刃`（12）、`晚霞`（1）
  - `Skills.json|/dataList/3/name`：`晚霞之劍`（名稱欄位）；note：…
  - `StoryData/S1.json|/dataList/9/content`：提及；譯文 `…`
- 暫定詞 `도발치` → `嘲諷值`（`term-new/도발치`，未生效）：舊譯 95 個單位，含暫定譯法 0 個；既有譯名 `挑釁值`
  - 舊譯 `Bufs.json|/dataList/2/desc`：KR `…`／TW `…`
  - 這次用了暫定譯法：`…`、`…`
```
兩者都空時寫 `None.`。`code()` 包 key 與譯文；note 與長譯文照 `oneLine` 收成一行。

`同一 KR 不同譯法` 改成兩小節：`#### 含這次新譯（N 組）` 列 `InconsistentAll`（舊的 `build.json` 沒有時用 `Report.Inconsistent`）中 `New > 0` 的全部組，每組的標題行加「，這次新譯 m 個」；`#### 只有舊譯` 列 `New == 0` 的前 `maxInconsistent` 組。原有每行格式不變。`infoInconsistent` 改收 `Info`。

審核 issue（`ReviewItems`），兩節：
- `KindNameDiffers`：Title `與既有譯名不同的新譯`；Hint：
  > 既有譯文對這些名稱有固定寫法，這次新譯不同（名稱欄位，或說明、劇情裡提到卻沒用既有譯法）。指的是同一個東西：在 `glossary/terms.yaml` 加條目（以既有譯名為準），並寫 `rules/rewrite/` 的改寫檔，只改這次新譯；依語境本來就該不同：把項目開頭的 ID 寫進 `rules/declined.yaml`（審核頁或 `lct accept`）。

  項目：
  - 每個 `NameDiff`：`` `name/<kr>`：既有 `zh1`（n1）、`zh2`（n2）；這次 N 個單位不同：`key1`（`zh`）、`key2`（提及）… ``，單位最多列 5 個，多的寫「等」
  - 每個 `TermDiff`：`` `term-new/<kr>`：暫定詞 → `zh`（未生效），舊譯 N 個單位、含暫定譯法 M 個 ``，有 `Existing` 時加「；既有譯名 `x`」，有 `Keys` 時加「；這次用了暫定譯法：`k1`、…」（最多 5 個）
- `KindInconsistentNew`：Title `同一 KR 不同譯法（含這次新譯）`；Hint：
  > 同一段 KR 在同族群、同欄位有不同譯法，其中有這次新翻的單位。要統一：照 `rules-review` 的 `inconsistent` 處理（詞彙或指引，加重翻要求檔或改寫檔）；依 JP/EN 或語境本來就該不同：把項目開頭的 ID 寫進 `rules/declined.yaml`。

  項目：`New > 0` 的組，跳過 `NewKeys` 全部已列在某個 `NameDiff.Units` 的組（spec 4「只含已列入 name 項目之單位的組就不重複列」）；每項 `` `inconsistent/<family>/<field>/<sha8>`（`itemid.Inconsistent`）：`KR`（族群 `f`、欄位 `x`，n 個單位，這次新譯 m 個）：`zh1`（c1）、`zh2`（c2） ``，譯法最多 5 種

review skill：「流程」第 1 步的種類清單中、`new_family` 之後加兩點：
> - `name_differs`：這次新譯的名稱與既有譯名不同。item 開頭是建議的 ID（`name/<kr>` 或 `term-new/<kr>`）。用 `bin/lct search --json --key K` 給人看 KR、新譯文與既有譯名。人決定統一：在 `glossary/terms.yaml` 加條目，`zh` 用既有譯名（格式見 `glossary/README.md`），再手寫 `rules/rewrite/<日期>-<主題>.yaml`（`reason`、`files`、`origins: [claude]`、`pattern`、`replace`、`kr_pattern`），只改 item 列出、寫法不同的這次新譯單位，寫好後給人看會改到哪些字。人決定保留（依語境該不同）：在 `rules/declined.yaml` 加一筆 `id`（item 開頭的 ID，用雙引號包起來）、`reason`、`date`。詞彙與改寫檔都先給人看，人同意才寫
> - `inconsistent_new`：同一段 KR 有不同譯法，其中有這次新翻的單位。item 開頭是 `inconsistent/…` 的 ID。統一或保留的判斷與 `/rules-review` 的 `inconsistent` 相同（KR 常比 JP、EN 籠統，譯法不同可能是對的）：統一就寫進詞彙或指引，再用 `bin/lct requeue --to-file TOPIC --json --reason R --key K …` 寫重翻要求檔或手寫改寫檔；保留就在 `rules/declined.yaml` 加一筆 `id`、`reason`、`date`。都先給人看，人同意才寫

- [ ] **Step 1: 寫失敗的測試**
  - `TestReportListsNameDiffs`：`Info` 有一個 `NameDiff`（名稱欄位一個、提及一個）與一個 `TermDiff` → 報告有 `### 與既有譯名不同的新譯`，含 `name/노을빛 검`、`（名稱欄位）`、`提及`、`term-new/도발치`、`未生效`、`既有譯名 `挑釁值``
  - `TestReportNameDiffsNone`：都空 → 該節是 `None.`
  - `TestReportSplitsInconsistent`：`InconsistentAll` 有 `New` 為 2 與 0 的組 → `#### 含這次新譯（1 組）` 下有前者且有「這次新譯 2 個」；`#### 只有舊譯` 下有後者
  - `TestReviewIssueNameDiffers`：issue 有 `name_differs` 節，項目各以 `` `name/<kr>` ``、`` `term-new/<kr>` `` 開頭；單位超過 5 個時寫「等」
  - `TestReviewIssueInconsistentNew`：兩個 `New > 0` 的組，一組的 `NewKeys` 全在 `NameDiffs` → issue 只列另一組；只有舊譯的組不列
  - `TestReviewKindsOrder`：`ReviewKinds` 照上面順序
  - 既有的 review skill contract 測試（每個 `ReviewKinds` 都要在 skill 裡）因新種類先失敗
- [ ] **Step 2: 跑測試，確認失敗**
  Run: `go test ./internal/build/ ./cmd/lct/ -run 'NameDiffs|SplitsInconsistent|IssueNameDiffers|InconsistentNew|ReviewKinds|SkillContract'`
  Expected: FAIL
- [ ] **Step 3: 實作** 報告、issue 與 skill 文字
- [ ] **Step 4: 跑測試，確認通過**
  Run: `go vet ./... && go test ./...`
  Expected: PASS
- [ ] **Step 5: Commit**：`build：報告與審核 issue 列出與既有譯名不同的新譯；不一致的組分新舊`

---

### Task 10: 建議種類 `name`

**Files:**
- Modify: `internal/suggest/suggest.go`（種類、`Kinds`、`fromBuild`、`fromProvisional`、`Run` 的合併）
- Test: `internal/suggest/suggest_test.go`、`cmd/lct/suggest_test.go`、`cmd/lct/accept_test.go`

**Interfaces:**
- Consumes: Task 8 `Info.Report.NameDiffs`、`TermDiffs`；Task 1 `ProvisionalTerm.Old`；Task 2 `existing.SuggestionID`
- Produces:
  - `suggest.KindName = "name"`，id `name/<kr>`（`existing.SuggestionID`）；doc 的 id 清單加一行 `name/<kr>   a name this update translated otherwise than the existing translations`
  - `Kinds`：note、speaker、family、field、adjustment-moved、adjustment-expired、**name**、term-new、term、inconsistent

做法：
- `fromBuild`：每個 `NameDiff` 一個建議。Title `與既有譯名不同：<kr>（既有 <主要譯法>，這次 <n> 個單位不同）`（KR 照 `clip(…, maxTitleText)`）。證據：先是每個既有譯法 `{KR: kr, ZH: zh, Count: units}`，再是每個單位 `{Key, KR: kr, ZH: unit.ZH, Note: unit.Note}`，合計最多 `MaxEvidence`
- `fromProvisional`：KR 第一筆非衝突紀錄有 `Old` 時，Title 改成 `暫定詞（與舊譯不同，未生效）：<kr> → <zh>`，登記那一列之後加一列 `{KR: kr, ZH: Old.Existing, Count: Old.Units, Note: "與舊譯不同：舊譯 <Units> 個單位，含暫定譯法 <With> 個"}`
- `fromBuild` 的 `TermDiffs`：在 `Run` 收集完之後，把每個 `TermDiff.Examples` 以 `{Key, KR, ZH, Note: "舊譯例句"}` 加到 `term-new/<kr>` 的證據（有這個建議才加，上限照舊）
- `Run` 最後：同一個 KR 同時有 `name/<kr>` 與 `term-new/<kr>` 時，把 `term-new` 的證據接到 `name/<kr>` 後面（上限照舊），刪掉 `term-new/<kr>`（spec 4「只提出一個：`name/<kr>`」）
- `rules/declined.yaml` 有 `name/<kr>` 時照現有 `Exclude` 略過；`lct accept` 對 `name/<kr>` 寫進 `rules/declined.yaml`（現有規則建議的處理，確認測試即可）

- [ ] **Step 1: 寫失敗的測試**
  - `TestSuggestName`：`Build.Report.NameDiffs` 一項 → 一個 `name/노을빛 검`，`Kind == "name"`，證據先是既有譯法（有 `Count`）、再是單位（有 `Key`、`Note`）
  - `TestSuggestNameOrder`：`name` 排在 `adjustment-expired` 之後、`term-new` 之前
  - `TestSuggestTermNewDiffering`：暫定詞紀錄有 `Old` → 標題含「與舊譯不同，未生效」，證據第二列 `Count == Old.Units`；`TermDiffs` 的例句以 `Note: "舊譯例句"` 加在後面
  - `TestSuggestMergesTermNewIntoName`：同一 KR 兩者都有 → 只有 `name/<kr>`，證據含暫定詞的登記列
  - `TestSuggestNameDeclined`（cmd/lct）：`rules/declined.yaml` 有 `name/<kr>` → `lct suggest --json` 不列
  - `TestAcceptNameSuggestion`（cmd/lct）：`lct accept --reason R 'name/<kr>'` 寫進 `rules/declined.yaml`
- [ ] **Step 2: 跑測試，確認失敗**
  Run: `go test ./internal/suggest/ ./cmd/lct/ -run 'SuggestName|TermNewDiffering|MergesTermNew|AcceptNameSuggestion'`
  Expected: FAIL
- [ ] **Step 3: 實作**
- [ ] **Step 4: 跑測試，確認通過**
  Run: `go vet ./... && go test ./...`
  Expected: PASS
- [ ] **Step 5: Commit**：`suggest：與既有譯名不同的新譯成為 name 建議；與舊譯不同的暫定詞附證據`

---

### Task 11: 審核頁與 skill

**Files:**
- Modify: `internal/site/review.go`（`ReviewInputs.Names`、`BuildReview`）、`cmd/lct/site.go`（`loadReviewInputs`）
- Modify: `site/src/review/names.ts`
- Modify: `.claude/skills/rules-review/SKILL.md`、`.claude/skills/review-page/SKILL.md`、`.claude/skills/term-decision/SKILL.md`
- Test: `internal/site/review_test.go`、`site/src/review/*.test.ts(x)`（有對照表的測試就加一筆）、`cmd/lct/contract_test.go`

**Interfaces:**
- Consumes: Task 2 `NewIndex`、`LoadTable`；Task 6 `Differs`、`CodeNameDiffers`；Task 10 `KindName`
- Produces: `site.ReviewInputs.Names *existing.Index`

做法：
- `loadReviewInputs` 讀表並以 `rules/declined.yaml` 的 id 建索引
- `BuildReview` 的 `visit`：origin claude、`Names.IsName(ku.Field, ku.KR)` 的單位，用該單位全部備註（`rv.Notes` 依 key 連成一段）與 `en`、`jp` 呼叫 `Differs`；有值就照其他單一單位的警告建 `WarningItem`（id `itemid.UnitWarning`，`Keys`、`families`、`origins` 照舊累積）。`lct accept`、`--dry-run`、`--tidy` 因此認得這個警告，不另改
- `names.ts`：`WARNING_NAMES['name-differs'] = { name: '與既有譯名不同', desc: '名稱欄位的譯法和既有譯文不同：同一個東西就統一，語境不同就保留' }`；`SUGGESTION_KINDS.name = '與既有譯名不同'`
- rules-review skill：
  - 開頭的建議種類清單加「與既有譯名不同的名稱」
  - 流程中處理各種類的地方加 `name` 一點：
    > - `name`：這次新譯的名稱與既有譯名不同。`evidence` 先列既有譯法（`count` 是單位數），再列這次的單位（`key`、`zh`、`note`）；有暫定詞的列（`note` 是批次 id、「衝突」或「與舊譯不同…」「舊譯例句」）時，這也是翻譯者登記的暫定詞。以既有譯名為準：單位與既有譯名指同一個東西（用 `bin/lct search --json --key K` 看全文與 JP/EN），就在 `glossary/terms.yaml` 加條目（`zh` 用既有譯名，格式見 `glossary/README.md`），並手寫 `rules/rewrite/<日期>-<主題>.yaml`（`reason`、`files`、`origins: [claude]`、`pattern`、`replace`、`kr_pattern`），只改證據中寫法不同的這次新譯單位。單位的 `note` 說明了語境不同、或證據不足以判斷是不是同一個東西時，不改檔，列進 `## 需要人決定`
  - 「限制」中 `term-new` 那一點加一句：「與 `name/<kr>` 同一個 KR 的暫定詞已併進 `name/<kr>`，照 `name` 處理」
- review-page skill：規則建議的 ID 清單加 `name/…`；處理說明加一點：`name/<kr>`（與既有譯名不同）人決定統一：同 rules-review 的 `name`（詞彙條目以既有譯名為準，加改寫檔只改這次新譯）；人決定保留：照第 3 步「沒問題」的 `bin/lct accept --json --reason '<理由>' 'name/<kr>'`，之後不再提出、不再警告，批次仍帶它當參考。`warning/name-differs/…` 是單一單位的警告，`lct accept` 只代表那個單位保留
- term-decision skill：暫定詞那一點加：「標題有『與舊譯不同，未生效』的暫定詞沒有生效，`evidence` 有舊譯的單位數與例句；以舊譯為主，人選的譯法與舊譯不同時照第 4 步處理。與 `name/<kr>` 同一個 KR 的暫定詞已併進 `name/<kr>`，不在這裡列出，由審核頁或 `/rules-review` 處理」

- [ ] **Step 1: 寫失敗的測試**
  - `TestReviewWarnsNameDiffers`（site）：`ReviewInputs.Names` 有表，claude 名稱紀錄與既有不同 → `Warnings` 有 `code: name-differs` 的項目，id 是 `itemid.UnitWarning(...)`；`Accepted` 有它時 `Confirmed` 有值（照現有警告的行為）；paratranz 紀錄不警告；`Names` 為 nil 時沒有
  - `TestAcceptDryRunNameDiffers`（cmd/lct）：`lct accept --dry-run --json 'warning/name-differs/…'` 認得這個 id
  - 網站：`warningInfo('name-differs', '')` 回 `與既有譯名不同`；`suggestionKindName('name')` 回 `與既有譯名不同`
  - contract：`TestNameSuggestionIsHandledBySkills`：rules-review、review-page、term-decision 三個 skill 都含 `name/`
- [ ] **Step 2: 跑測試，確認失敗**
  Run: `go test ./internal/site/ ./cmd/lct/ -run 'NameDiffers|NameSuggestion'` 與 `cd site && npm test`
  Expected: FAIL
- [ ] **Step 3: 實作**
- [ ] **Step 4: 跑測試，確認通過**
  Run: `go vet ./... && go test ./...`，再 `cd site && npm ci && npm test && npm run build && npm run typecheck`
  Expected: PASS
- [ ] **Step 5: Commit**：`site、skills：審核頁列出 name-differs；name 建議的處理`

---

### Task 12: 以 raw-20261112 的資料重演驗收，補文件

**Files:**
- Modify: `README.md`（`lct batch submit` 詞彙檢查那一點之後加「既有譯名表」一點）
- Create（不進版控）：`work/replay/`（重演用的 Go 程式與腳本；`/work/` 已在 `.gitignore`）、結果寫到 scratchpad 的 `existing-names-replay.md`
- 不改：`.work/`、tw、raw

**Interfaces:**
- Consumes: 全部前面的 task

重演環境用 scratchpad 的 `adjclone`（本機 clone，`tw`、`raw` 是本地分支；先 `git -C .work/tw checkout -- . && git -C .work/tw clean -fd`），把功能分支 fetch 進去、`go build -o bin/lct ./cmd/lct`。規則（`guides/`、`glossary/`、`rules/`）用分支起點 `git merge-base origin/main <功能分支>` 的版本，也就是這次加詞彙與改寫檔之前：否則這次加進詞彙表的 KR 會被名稱表排除、改寫檔會被 plan 套用，量不到「當時」的效果。做法是在 adjclone 的主工作樹 `git checkout <merge-base> -- guides glossary rules`（只在 scratch clone），程式仍用功能分支建的 `bin/lct`。手動找出的 23 個 KR：
- 暫定詞 9 個：`불주먹 사무소`、`스티그마 공방`、`기본 공격 스킬`、`파괴 불가 코인`、`도발치`、`연기전쟁`、`눈물로 벼려낸 검`、`타오르는 일격`、`8시의 서커스`
- 技能名 10 個：`새벽녘`、`노을빛 검`、`새벽을 가르는 검`、`아지랑이`、`여명의 섬광`、`사출`、`맹렬한 돌진`、`격돌`、`쐐기 대형 전개`、`연계`
- 其他 4 個：`개화`、`회피`、`기본 스킬`、`피해량 증가`

- [ ] **Step 1: 名稱表與批次**：本地 `tw` 分支重設到 `5deb6c9`（更新前），`raw` worktree 在 `raw-20261112`，`bin/lct plan --json`。記下 `name_table` 的名稱數、同句數與兩個檔的大小。接著重複 `bin/lct batch next --json` 直到沒有批次（不翻譯），讀 `work/batches/*.json` 統計：
  - 23 個 KR 中，各有幾個「KR 含它的待譯單位」，其中幾個的批次 `names` 帶到它（列出沒帶到的與原因，例如兩個字的詞在說明裡只當參考、或名稱表沒有它）
  - 每批平均帶幾項、最多幾項、達到上限 40 的批次數；抽 3 批人工看帶到的名稱是否相關，估雜訊比例
  - 有 `same_text` 的單位數，抽看菲利普「그 누구보다 슬퍼하고 있는 나를 위해」那幾個單位是否帶到
  Expected: 指令都 exit 0；數字寫進結果檔
- [ ] **Step 2: 1b**：`work/replay/` 寫一支 Go 程式（`go run ./work/replay …`），以 `5deb6c9` 的紀錄（全部都是舊譯）、`raw-20261112` 的 KR 包、Step 1 的名稱表與上述分支起點的詞彙表，對 `522cc08` 的 `.meta/terms.jsonl` 裡 26 個暫定詞跑 `OldTally.Use`
  Expected: `changed: true` 的 8 個（`불주먹 사무소`、`스티그마 공방`、`기본 공격 스킬`、`파괴 불가 코인`、`도발치`、`연기전쟁`、`눈물로 벼려낸 검`、`타오르는 일격`）被標出，其餘 18 個不被標出（含 `8시의 서커스`：暫定 `八點馬戲團` 與舊譯相同，它的問題是新譯裡出現一處 `8點馬戲團`，不是暫定詞與舊譯不同）。spec 寫「9 個／17 個」把 `8시의 서커스` 算在 9 個裡；這是 spec 的計數，記為裁定。結果與預期不符的逐一說明原因
- [ ] **Step 3: 報告**：同一支程式另做一份 store：`522cc08` 的紀錄，其中「與 `5deb6c9` 相比新增或改過的 origin claude 紀錄」設 `Written: "raw-20261112"`；放進 Step 1 的 `names.json`；`terms.jsonl` 依 Step 2 的結果補上 `old`。在一個重設到 `522cc08` 的本地 tw 分支上寫入這些狀態（只在 scratch clone），`bin/lct build`，讀 `.meta/reports/raw-20261112.md` 與 `bin/lct review --json`
  Expected: 報告有「與既有譯名不同的新譯」，23 個 KR 中列出幾個、漏了幾個，漏的逐一說明原因；「同一 KR 不同譯法」的「含這次新譯」組數（對照 spec 說的 78 項混在一起）；`name_differs`、`inconsistent_new` 兩節的項數
- [ ] **Step 4: README**：在第 111 行（詞彙檢查與暫定詞）之後加一點：
  > - 既有譯名表：`lct plan` 在每次更新（每個 raw tag）第一次執行時，從現行譯文的名稱欄位（`lct.yaml` 的 `names.fields`）整理既有譯名，寫進 `tw` 的 `.meta/names.json`，並把在不同族群、欄位或說話者出現的同一句譯文整理到 `.meta/same-text.json`；同一個 raw tag 之後的 plan 沿用，不重算。批次把單位用到的名稱放進 `names`、同句的其他譯文放進 `refs.same_text`，只是參考。送出時名稱欄位的譯法與既有譯名不同（JP 或 EN 對得上、`note` 沒寫出 KR）只警告 `name-differs`；暫定詞與名稱表或舊譯不同時照樣記下但不生效，送出結果附舊譯例句。建置報告與審核 issue 列出「與既有譯名不同的新譯」，`lct suggest` 提出 `name/<kr>`：人決定統一就加詞彙條目（以既有譯名為準）與改寫檔，決定保留就寫進 `rules/declined.yaml`。詞彙表的條目永遠優先於名稱表
- [ ] **Step 5: 跑全部檢查**
  Run: `go vet ./... && go test ./...`，`cd site && npm ci && npm test && npm run build && npm run typecheck`
  Expected: PASS
- [ ] **Step 6: Commit**：`README：既有譯名表`（重演的程式與結果不進版控；結果檔路徑與摘要寫進 ledger 與最後的回報）
