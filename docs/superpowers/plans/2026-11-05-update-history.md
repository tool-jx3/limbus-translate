# 網站「更新」頁 實作計畫

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 每個 Release 發布時和前一個 Release 比對，把變動寫進 `tw` 的 `.meta/history/<tag>.json`；網站新增「更新」頁，列出全部 Release，並以卡片或表格顯示單一 Release 的變動。

**Architecture:** 新套件 `internal/history` 負責比對（沿用 `align.AlignVersions` 的對應）與讀寫比對紀錄；`lct publish` 在做 `build:` commit 前寫入這次與缺少的比對紀錄；`lct site` 把紀錄匯出成 `data/updates/`；Preact 網站新增 `#/u` 與 `#/u/<tag>` 兩個頁面。

**Tech Stack:** Go（`internal/history`、`internal/publish`、`internal/site`、`cmd/lct`）；Preact + Vite + Vitest（`site/`）。

**Spec:** `docs/superpowers/specs/2026-11-05-update-history-design.md`

## Global Constraints

- `CLAUDE.md`：狀態只經由 `lct` 修改；`raw`、`tw` 分支與 Release 只由 CI 寫入；規則只在 `guides/`、`glossary/`、`rules/`；所有 `lct` 指令支援 `--json`；exit code 0 成功、1 發現或使用者錯誤、2 系統錯誤。
- 提交前：`go vet ./... && go test ./...`（在 `set -o pipefail` 下）；動到 `site/` 時另跑 `cd site && npm test && npm run typecheck && npm run build`。
- 不用 prettier 重排既有檔案（專案沒有 prettier 設定）；照既有檔案的排版手寫。
- 比對單位的對應沿用 `align.AlignVersions`，不另做一套。
- 比對紀錄寫好後不再改寫，唯一例外是 Release 還沒建成的那個 tag（下一次執行用同一個 tag）。
- 比對紀錄或補齊失敗只發警告，不能讓 `lct publish` 失敗、擋住 Release。
- 網站文字照規格：頂端列「更新」；列表的「比對 <tag>」「遊戲更新 <raw tag>」「重建」「第一版，沒有可比對的上一版」「其他檔案 N」；種類「新增」「KR 修改」「刪除」「只有譯文改變」；原因「人工調整」「補翻」「重翻」；「隱藏不需翻譯的單位」；「卡片」「表格」；「JP／EN（目前的）」；「在檔案中開啟」。
- 網站在手機寬度（約 400px）整頁不左右捲動；深色模式沿用 `styles.css` 的 token。
- commit 訊息結尾附上工作階段指定的 `Co-Authored-By` 與 `Claude-Session` 兩行。
- 在分支 `claude/festive-wright-g2k853` 開發；不要 push 到 `main`、`tw`、`raw`。

## Review Focus

1. **比對紀錄出錯（某個舊 Release 的 raw tag 不在本地、`build.json` 讀不到）**：`lct publish` 照常 commit 並回報 `release_needed`，警告寫在 `history_warnings`，缺的那個 tag 留到下次補。→ Task 3 `TestRunWarnsWhenAHistoryCannotBeMade`。
2. **整個檔案新增或刪除**：新版才有的檔，所有字串是「新增」；舊版才有的，所有字串是「刪除」；打包檔裡沒有該檔時譯文當空字串。→ Task 1 `TestCompareAFileOnlyOnOneSide`。
3. **只是換位置、但譯文也變了的單位**：列為「只有譯文改變」，`ptr` 是新位置、`old_ptr` 是舊位置。→ Task 1 `TestCompareAMovedUnitWhoseTranslationChanged`。
4. **路由裡的 tag 有句點（`zh-TW-20261105.1`），或不是 Release tag（`..`、亂打的字）**：前者正常解析；後者回首頁，不去讀 `data/updates/` 以外的檔。→ Task 5 `route.test.ts` 的兩個案例。
5. **上萬筆的 Release**：一次只畫 300 筆，「再顯示 300 筆」往下加；篩選只在記憶體裡做。→ Task 7 `shows 300 entries and then 300 more`。

---

### Task 1: 比對核心（`internal/history`）

**Files:**
- Create: `internal/history/compare.go`
- Test: `internal/history/compare_test.go`

**Interfaces:**
- Consumes: `align.AlignVersions(file string, old, new *jsonx.Node) []align.VersionMatch`、`units.Extract(file string, doc *jsonx.Node) []units.Unit`、`pack.Pack`（`Has`、`Doc`）、`store.Record`。
- Produces:
  ```go
  const (
      KindNew       = "new"
      KindKRChanged = "kr-changed"
      KindDeleted   = "deleted"
      KindZHChanged = "zh-changed"
  )
  const (
      CauseAdjustment   = "adjustment"   // 新來源是 human
      CauseLate         = "late"         // 舊來源是 stand-in
      CauseRetranslated = "retranslated" // 其他
  )
  // OriginStandIn 同 site.OriginStandIn 的意思（不 import site，避免循環）。
  const OriginStandIn = "stand-in"

  // State 是一個 Release 的狀態：KR 快照、打包出的語言包（空前綴）、翻譯紀錄。
  type State struct {
      KR, ZH  *pack.Pack
      Records map[string]store.Record
  }

  // Entry 是一個變動。*string 欄位在該側存在時才有（空字串也照存）。
  type Entry struct {
      File       string  `json:"file"`
      Field      string  `json:"field"`
      Pointer    string  `json:"ptr"`               // 新的；刪除的是舊的
      OldPointer string  `json:"old_ptr,omitempty"` // 換了位置時的舊 pointer
      Kind       string  `json:"kind"`
      OldKR      *string `json:"old_kr,omitempty"`  // 只有 kr-changed、deleted
      NewKR      *string `json:"new_kr,omitempty"`  // new、kr-changed、zh-changed
      OldZH      *string `json:"old_zh,omitempty"`  // kr-changed、deleted、zh-changed
      NewZH      *string `json:"new_zh,omitempty"`  // new、kr-changed、zh-changed
      OldOrigin  string  `json:"old_origin,omitempty"`
      NewOrigin  string  `json:"new_origin,omitempty"`
      Copy       bool    `json:"copy,omitempty"`
      Cause      string  `json:"cause,omitempty"`   // 只有 zh-changed
  }

  // Compare 比對 files（KR 檔名，例 "Passives.json"）中每個檔的舊、新狀態，依 files 的順序回傳變動。
  func Compare(old, new State, files []string) ([]Entry, error)
  ```

- [ ] **Step 1: 寫失敗的測試**

用測試輔助函式在暫存目錄寫 KR 檔（`KR_<name>`，以 `pack.Open(dir, "KR_")` 開）與打包檔（以 `pack.Open(dir, "")` 開），組成舊、新兩個 `State`。案例與斷言：

```go
func TestCompareTellsTheFourKinds(t *testing.T)
// 舊 KR {"dataList":[{"id":1,"desc":"a"},{"id":2,"desc":"b"},{"id":3,"desc":"c"}]}
// 新 KR {"dataList":[{"id":1,"desc":"a"},{"id":2,"desc":"B"},{"id":4,"desc":"d"}]}
// 舊譯 a→"甲" b→"乙" c→"丙"；新譯 a→"甲甲" B→"乙乙" d→"丁"
// 想要：/dataList/0/desc zh-changed（NewKR "a"、OldZH "甲"、NewZH "甲甲"、OldKR nil）
//       /dataList/1/desc kr-changed（OldKR "b"、NewKR "B"、OldZH "乙"、NewZH "乙乙"）
//       /dataList/2/desc new（id 4：NewKR "d"、NewZH "丁"、Old* 皆 nil）
//       /dataList/2/desc deleted（id 3：OldKR "c"、OldZH "丙"、New* 皆 nil，排在最後）
func TestCompareLeavesOutWhatDidNotChange(t *testing.T)    // KR 與譯文都相同：沒有 entry
func TestCompareAnEntryInsertedBeforeIsNoChange(t *testing.T) // 無 id 的 dataList 前面插入一筆：只有那筆 new，後面的不列
func TestCompareAMovedUnitWhoseTranslationChanged(t *testing.T) // 同上但被移動的那筆譯文也變：zh-changed，Pointer 新、OldPointer 舊
func TestCompareMatchesByID(t *testing.T)
// LobbyBGM 的情形：id 200342/200343 的 name 換成別的曲名、200344 新增 → 兩筆 kr-changed、一筆 new
func TestCompareAFileOnlyOnOneSide(t *testing.T)
// 新版多一個檔：全部 new；舊版多一個檔：全部 deleted；打包檔缺該檔：譯文為 ""（*string 指向 ""）
func TestCompareOriginsCopyAndCause(t *testing.T)
// 來源規則同網站：紀錄的 ZH 等於打包字串才用紀錄的 Origin，否則 "stand-in"
// zh-changed 的原因：新 human → "adjustment"；舊 stand-in → "late"；舊 paratranz、新 claude → "retranslated"
// 新的（刪除時舊的）來源是 copy：Copy == true
```

- [ ] **Step 2: 跑測試，確認失敗**

Run: `go test ./internal/history/`
Expected: FAIL（`undefined: Compare` 等）

- [ ] **Step 3: 在 `internal/history/compare.go` 實作 `Compare`**

每個檔：兩側不存在的檔當 nil 文件；`align.AlignVersions` 給出配對；`units.Extract` 取欄位與 KR；打包檔的 `Doc(file).Strings()` 取 pointer → 譯文。`StatusNew`／`StatusDeleted`／`StatusChanged` 對應三種；`StatusUnchanged` 只在譯文不同時列 zh-changed。一個檔內的順序就是 `AlignVersions` 回傳的順序（新版文件順序，刪除的在後）。

- [ ] **Step 4: 跑測試，確認通過**

Run: `go vet ./internal/history/ && go test ./internal/history/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/history/compare.go internal/history/compare_test.go
git commit -m "feat: compare the strings of two Releases (internal/history)"
```

---

### Task 2: Release 的狀態與比對紀錄

**Files:**
- Create: `internal/history/release.go`、`internal/history/record.go`
- Modify: `internal/gitx/gitx.go`（加 `RevTime`）
- Test: `internal/history/release_test.go`、`internal/history/record_test.go`、`internal/gitx/gitx_test.go`

**Interfaces:**
- Consumes: Task 1 的 `State`、`Entry`、`Compare`、各 Kind；`gitx.Archive(repo, rev string, paths []string, dest string) error`、`raw.OpenKRAt(worktree, tag, dir string) (*pack.Pack, error)`、`build.ReadInfo(twRoot string) (build.Info, error)`、`store.Open(root).LoadRecords()`、`store.Open(root).WriteMetaJSON(name string, v any) error`、`ReadMetaJSON`。
- Produces:
  ```go
  // gitx
  func RevTime(dir, rev string) (time.Time, error) // rev 的 committer 時間（UTC）

  // history
  type Counts struct {
      New       int `json:"new"`
      KRChanged int `json:"kr_changed"`
      Deleted   int `json:"deleted"`
      ZHChanged int `json:"zh_changed"`
  }
  // Record 是 .meta/history/<Tag>.json。
  type Record struct {
      Tag             string   `json:"tag"`
      PrevTag         string   `json:"prev_tag"`
      ReleasedAt      string   `json:"released_at"`      // RFC 3339，UTC，到秒
      PrevReleasedAt  string   `json:"prev_released_at"`
      RawTag          string   `json:"raw_tag"`
      PrevRawTag      string   `json:"prev_raw_tag"`
      RulesCommit     string   `json:"rules_commit"`
      PrevRulesCommit string   `json:"prev_rules_commit"`
      Counts          Counts   `json:"counts"`      // 不含不需翻譯的
      CopyCounts      Counts   `json:"copy_counts"` // 只有不需翻譯的
      OtherFiles      []string `json:"other_files"` // 從不為 nil
      Entries         []Entry  `json:"entries"`     // 從不為 nil
  }
  // Side 是一個 Release（或還沒 commit 的工作樹）的狀態與它在磁碟上的位置。
  type Side struct {
      Tag, RawTag, RulesCommit string
      Time                     time.Time
      State                    State
      PackDir                  string // 打包檔根目錄（…/LimbusCompany_Data）
  }
  // ReadRev 讀 tw 的 rev（Release tag）：把該 commit 的 LimbusCompany_Data 與 .meta 匯出到 tmp/tw，
  // 依其 .meta/build.json 的 raw tag 用 raw.OpenKRAt 匯出 KR 到 tmp/raw。tmp 必須是空的新目錄。
  func ReadRev(twRoot, rawRoot, rev, langDir, tmp string) (*Side, error)
  // ReadWorktree 讀 tw 工作樹現在的檔案（還沒 commit 的這次建置），KR 同樣依 build.json 的 raw tag 匯出到 tmp/raw。
  func ReadWorktree(twRoot, rawRoot, langDir, tmp string) (*Side, error)
  // Make 比對 prev 與 cur，cur 的 tag 是 tag、發布時間是 at。
  func Make(prev, cur *Side, tag string, at time.Time, langDir string) (*Record, error)
  func (r *Record) Empty() bool            // 沒有 entry 也沒有其他檔案
  func Write(twRoot string, rec *Record) error // store.WriteMetaJSON("history/"+Tag+".json")
  func Has(twRoot, tag string) (bool, error)
  func List(twRoot string) ([]Record, error)   // 全部紀錄，ReleasedAt 新的在前，同時間依 Tag 降冪
  ```

- [ ] **Step 1: 寫失敗的測試**

```go
// gitx_test.go
func TestRevTimeIsTheCommitTimeOfTheRevision(t *testing.T) // 兩個 commit（GIT_COMMITTER_DATE 不同），對 tag 取到較早那個的時間
// record_test.go
func TestWriteThenListNewestFirst(t *testing.T)   // 寫三筆，List 依 ReleasedAt 新→舊；Has 對寫過的是 true
func TestRecordJSONHasNoNilLists(t *testing.T)    // 空的 Record 寫出 "other_files": []、"entries": []
// release_test.go：用暫存 git repo（raw 分支有 raw-A、raw-B 兩個 tag；tw 有兩個 build commit，各打 zh-TW-…​.1、.2）
func TestMakeComparesTwoReleases(t *testing.T)
// 斷言 Counts、CopyCounts、PrevTag、RawTag／PrevRawTag、RulesCommit 取自 build.json、ReleasedAt 是 RevTime
func TestMakeListsTheOtherFilesThatChanged(t *testing.T)
// 只改 Lang/zh-TW/Info/LICENSE-CONTENT 與 Info/version.json：OtherFiles == ["Lang/zh-TW/Info/LICENSE-CONTENT"]（相對於 LimbusCompany_Data），Entries 空，Empty() 為 false
func TestMakeComparesOnlyTheFilesThatDiffer(t *testing.T)
// KR 與打包檔的位元組都相同的檔不進 Compare（以一個內容壞掉、但兩側相同的檔驗證：它不被解析也不出錯）
func TestReadWorktreeReadsWhatIsNotCommitted(t *testing.T)
```

- [ ] **Step 2: 跑測試，確認失敗**

Run: `go test ./internal/history/ ./internal/gitx/`
Expected: FAIL（`undefined: RevTime`、`undefined: Make` 等）

- [ ] **Step 3: 實作**

- `RevTime`：`git log -1 --format=%ct <rev>`，同 `CommitTime` 的寫法。
- `ReadRev`／`ReadWorktree`：打包檔用 `pack.Open(<PackDir>/Lang/<langDir>, "")`；紀錄用 `store.Open(<tw 根>).LoadRecords()`；`build.ReadInfo` 取 raw tag、rules commit。
- `Make`：要比的檔是兩側 KR 檔名的聯集中，KR 位元組或打包檔位元組不同的（`pack.Path` 讀檔比較），依字串排序後交給 `Compare`。其他檔案：`PackDir` 底下不是 `Lang/<langDir>/<KR 檔名>`、也不是 `Lang/<langDir>/Info/version.json` 的檔，兩側位元組不同或只在一側存在的，依字串排序。依 `Copy` 分別加到 `CopyCounts` 或 `Counts`。

- [ ] **Step 4: 跑測試，確認通過**

Run: `go vet ./... && go test ./internal/history/ ./internal/gitx/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/history internal/gitx
git commit -m "feat: make the history record of a Release from git"
```

---

### Task 3: `lct publish` 寫入這次與缺少的比對紀錄

**Files:**
- Modify: `internal/publish/publish.go`（`Run`、`Result`）、`cmd/lct/publish.go`
- Modify: `cmd/lct/e2e_phase2_test.go`、`README.md`
- Test: `internal/publish/publish_test.go`

**Interfaces:**
- Consumes: Task 2 的 `ReadRev`、`ReadWorktree`、`Make`、`Record.Empty`、`Write`、`Has`；`publish.LastRelease`、`gitx.Tags`、`latestRelease` 的排序。
- Produces:
  ```go
  // 新增 rawRoot；其餘不變。
  func Run(root, twRoot, rawRoot, outDir, langDir string) (*Result, error)
  // Result 新增：
  History         []string `json:"history,omitempty"`          // 這次寫入的比對紀錄的 tag
  HistoryWarnings []string `json:"history_warnings,omitempty"` // 沒寫成的原因，一則一個 tag
  ```
  `publish` 套件內的時鐘 `var now = time.Now`，測試可替換。

- [ ] **Step 1: 寫失敗的測試**

```go
func TestRunWritesTheHistoryOfTheReleaseItPrepares(t *testing.T)
// 已有 Release .1；這次的語言包不同 → 這次的 build: commit 含 .meta/history/<下一個 tag>.json，PrevTag == ".1"，Result.History 含該 tag
func TestRunWritesNoHistoryWhenThePackIsTheSame(t *testing.T)
// 語言包和最新的 Release 相同 → 沒有新的比對紀錄
func TestRunFillsInTheMissingHistories(t *testing.T)
// 已有 .1、.2、.3 三個 Release、都沒有比對紀錄 → 寫 .2（比 .1）與 .3（比 .2），不寫 .1；再跑一次 Result.History 為空、tw 沒有新 commit
func TestRunRewritesTheHistoryOfATagNotYetReleased(t *testing.T)
// 跑一次（不建 Release tag）、改一個譯文再跑：同一個 tag 的比對紀錄被更新，含新的變動
func TestRunWarnsWhenAHistoryCannotBeMade(t *testing.T)
// 某個舊 Release 的 raw tag 被刪掉 → Run 不回錯誤、照常 commit、ReleaseNeeded 正確，HistoryWarnings 有一則提到那個 tag，該 tag 沒有比對紀錄
```
`e2e_phase2_test.go`：第二次發 Release 之後，斷言 `.meta/history/<第二個 tag>.json` 在 `tw` 的 HEAD，`prev_tag` 是第一個，`counts` 與 replay 裡 plan 的新增／修改／刪除一致。

- [ ] **Step 2: 跑測試，確認失敗**

Run: `go test ./internal/publish/ ./cmd/lct/ -run 'History|Phase2'`
Expected: FAIL（`Run` 參數數目不符，或沒有比對紀錄）

- [ ] **Step 3: 實作**

在 `Run` 裡、`CommitAll` 之前，在 store lock 外做（比對很花時間，只有寫檔放進 `CommitAll` 前的同一段）：
1. 所有 Release tag 依 `latestRelease` 的規則由舊到新排序；第 2 個起每個缺紀錄的 tag，和前一個比（`ReadRev` 兩側），`Write`；每個 Release 的狀態只讀一次，暫存目錄用完就刪。
2. 這次：`ReadWorktree` 對 `LastRelease` 的 `ReadRev`，`Make(…, tag, now().UTC().Truncate(time.Second), langDir)`；不是 `Empty()` 才 `Write`。沒有任何 Release 時不寫。
3. 每個失敗都只加一則 `HistoryWarnings`，繼續往下。

`cmd/lct/publish.go` 傳入 raw worktree（`ctx.Config.Worktrees.Raw`），把每則 `HistoryWarnings` 印到 stderr（`lct: warning: …`）。README「GitHub Actions 設定」與 `tw` 的說明加一段：`.meta/history/<tag>.json` 是什麼、何時寫、怎麼補齊。

- [ ] **Step 4: 跑測試，確認通過**

Run: `set -o pipefail; go vet ./... && go test ./...`
Expected: PASS（全部套件）

- [ ] **Step 5: Commit**

```bash
git add internal/publish cmd/lct/publish.go cmd/lct/e2e_phase2_test.go README.md
git commit -m "feat: lct publish keeps the history of every Release"
```

---

### Task 4: `lct site` 匯出 `data/updates/`

**Files:**
- Create: `internal/site/updates.go`
- Modify: `cmd/lct/site.go`
- Test: `internal/site/updates_test.go`、`cmd/lct/site_test.go`

**Interfaces:**
- Consumes: Task 2 的 `history.List`、`Record`、`Counts`；`site.OrderFiles(paths []string, r *rules.Rules) []string`。
- Produces（網站讀的 JSON）：
  ```go
  type UpdateRelease struct {
      Tag        string         `json:"tag"`
      ReleasedAt string         `json:"released_at"`
      PrevTag    string         `json:"prev_tag,omitempty"`
      RawTag     string         `json:"raw_tag"`
      PrevRawTag string         `json:"prev_raw_tag,omitempty"`
      Kind       string         `json:"kind"` // "game"（raw tag 和前一個不同）、"rebuild"、"first"
      Counts     history.Counts `json:"counts"`
      CopyCounts history.Counts `json:"copy_counts"`
      OtherFiles int            `json:"other_files"`
      Files      int            `json:"files"` // 有變動的檔數
  }
  type UpdateIndex struct {
      Releases []UpdateRelease `json:"releases"` // 新的在前；從不為 nil
  }
  // ExportUpdates 寫 <outDir>/updates/index.json 與每個紀錄的 <outDir>/updates/<tag>.json（內容即 history.Record，entries 依 OrderFiles 的檔案順序穩定重排）。
  func ExportUpdates(records []history.Record, r *rules.Rules, outDir string) (UpdateIndex, error)
  ```
  第一版：最早那筆紀錄的 `PrevTag` 若沒有自己的紀錄，就以它的 `PrevTag`、`PrevReleasedAt`、`PrevRawTag` 加一列 `Kind: "first"`，數量皆 0。沒有任何紀錄時 `releases` 為 `[]`。

- [ ] **Step 1: 寫失敗的測試**

```go
func TestExportUpdatesListsEveryReleaseNewestFirst(t *testing.T)
// 兩筆紀錄（.3 比 .2 同 raw tag、.2 比 .1 不同 raw tag）→ releases: .3 rebuild、.2 game、.1 first
func TestExportUpdatesOrdersTheEntriesByFile(t *testing.T) // entries 的檔案順序同 OrderFiles，同檔內順序不變
func TestExportUpdatesWithNoRecords(t *testing.T)          // index.json 為 {"releases":[]}
```
`cmd/lct/site_test.go`：tw 有 `.meta/history/` 時，`lct site --out` 的輸出目錄有 `updates/index.json`。

- [ ] **Step 2: 跑測試，確認失敗**

Run: `go test ./internal/site/ ./cmd/lct/ -run 'Update'`
Expected: FAIL（`undefined: ExportUpdates`）

- [ ] **Step 3: 實作 `ExportUpdates`，並在 `runSite` 於 `site.Export` 之後呼叫**（`history.List(<tw>)`；輸出目錄沿用 `--out`；`siteOutput` 不必多欄位）。

- [ ] **Step 4: 跑測試，確認通過**

Run: `set -o pipefail; go vet ./... && go test ./...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/site/updates.go internal/site/updates_test.go cmd/lct/site.go cmd/lct/site_test.go
git commit -m "feat: lct site exports the history of the Releases"
```

---

### Task 5: 網站資料層：型別、載入、路由、設定、篩選

**Files:**
- Modify: `site/src/types.ts`、`site/src/data.ts`、`site/src/route.ts`、`site/src/settings.ts`
- Create: `site/src/updates/filter.ts`
- Test: `site/src/route.test.ts`、`site/src/settings.test.ts`、`site/src/data.test.ts`、`site/src/updates/filter.test.ts`

**Interfaces:**
- Consumes: Task 4 的 JSON 格式（欄位名稱照抄）。
- Produces:
  ```ts
  // types.ts
  export type UpdateKind = 'new' | 'kr-changed' | 'deleted' | 'zh-changed'
  export type UpdateCause = 'adjustment' | 'late' | 'retranslated'
  export interface UpdateCounts { new: number; kr_changed: number; deleted: number; zh_changed: number }
  export interface UpdateRelease { tag: string; released_at: string; prev_tag?: string; raw_tag: string; prev_raw_tag?: string; kind: 'game' | 'rebuild' | 'first'; counts: UpdateCounts; copy_counts: UpdateCounts; other_files: number; files: number }
  export interface UpdateEntry { file: string; field: string; ptr: string; old_ptr?: string; kind: UpdateKind; old_kr?: string; new_kr?: string; old_zh?: string; new_zh?: string; old_origin?: string; new_origin?: string; copy?: boolean; cause?: UpdateCause }
  export interface UpdateRecord { tag: string; prev_tag: string; released_at: string; prev_released_at: string; raw_tag: string; prev_raw_tag: string; rules_commit: string; prev_rules_commit: string; counts: UpdateCounts; copy_counts: UpdateCounts; other_files: string[]; entries: UpdateEntry[] }
  // data.ts（同既有 loader 的快取與 DataError）
  export function loadUpdates(): Promise<{ releases: UpdateRelease[] }> // data/updates/index.json
  export function loadUpdate(tag: string): Promise<UpdateRecord>       // data/updates/<tag>.json
  // route.ts
  | { kind: 'updates' }
  | { kind: 'update'; tag: string; kinds: UpdateKind[]; origin: string; file: string; copy: boolean }
  export function updatesHref(): string                                  // '#/u'
  export function updateHref(tag: string, f?: Partial<UpdateFilter>): string // '#/u/<tag>?k=new,deleted&o=claude&f=<file>&copy=1'，預設值不寫進網址
  // settings.ts：Settings 新增 updateView: 'cards' | 'table'，預設 'cards'；讀到其他值時用預設
  // updates/filter.ts
  export interface UpdateFilter { kinds: UpdateKind[]; origin: string; file: string; copy: boolean } // kinds 空 = 全部；origin/file '' = 全部；copy=false 隱藏不需翻譯的
  export function filterEntries(entries: UpdateEntry[], f: UpdateFilter): UpdateEntry[] // 來源比對 new_origin，刪除的比對 old_origin
  export function countKinds(entries: UpdateEntry[], copy: boolean): UpdateCounts
  export function entryFiles(entries: UpdateEntry[]): string[]   // 出現順序、不重複
  export function entryOrigins(entries: UpdateEntry[]): string[]  // 排序、不重複
  ```

- [ ] **Step 1: 寫失敗的測試**

- `route.test.ts`：`#/u` → `{kind:'updates'}`；`#/u/zh-TW-20261105.1?k=new,kr-changed&o=claude&f=Passives.json&copy=1` 解析出對應值，`updateHref` 反向得回同一個 hash；`#/u/zh-TW-20261105.1` 的預設值是 `kinds: []`、`origin: ''`、`file: ''`、`copy: false`；`#/u/..`、`#/u/foo`、`#/u/zh-TW-2026.1` 都回 `{kind:'home'}`（tag 必須符合 `/^zh-TW-\d{8}\.\d+$/`）；`k` 裡不認識的種類被忽略。
- `settings.test.ts`：沒存過時 `updateView` 是 `'cards'`；存 `'table'` 後讀回 `'table'`；存了 `'grid'` 讀回 `'cards'`。
- `data.test.ts`：`loadUpdate('zh-TW-20261105.1')` 讀 `data/updates/zh-TW-20261105.1.json`；同一個 tag 只抓一次。
- `filter.test.ts`：種類、來源（刪除的看 old_origin）、檔案、copy 各自與組合的篩選；`countKinds` 在 `copy=false` 時不算不需翻譯的。

- [ ] **Step 2: 跑測試，確認失敗**

Run: `cd site && npm test`
Expected: FAIL（找不到 `updatesHref`、`loadUpdate`、`filterEntries` 等）

- [ ] **Step 3: 實作上述介面**

- [ ] **Step 4: 跑測試，確認通過**

Run: `cd site && npm test && npm run typecheck`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add site/src/types.ts site/src/data.ts site/src/route.ts site/src/settings.ts site/src/updates site/src/*.test.ts
git commit -m "feat(site): data, routes and filters of the update pages"
```

---

### Task 6: 「更新」列表頁與頂端列入口

**Files:**
- Create: `site/src/updates/UpdatesView.tsx`
- Modify: `site/src/app/Header.tsx`、`site/src/app/App.tsx`、`site/src/styles.css`
- Test: `site/src/updates/UpdatesView.test.tsx`、`site/src/app/App.test.tsx`

**Interfaces:**
- Consumes: Task 5 的 `loadUpdates`、`updatesHref`、`updateHref`、`UpdateRelease`。
- Produces: `export function UpdatesView(): JSX.Element`；`App` 在 `route.kind === 'updates'` 時顯示它，`'update'` 時顯示 Task 7 的 `UpdateView`（本 Task 先放一個顯示 tag 的佔位元件，Task 7 取代）。

- [ ] **Step 1: 寫失敗的測試**（照 `ReviewView.test.tsx` 模擬資料的方式）

- 頂端列在「審核」左邊有連到 `#/u` 的「更新」；在 `#/u` 與 `#/u/<tag>` 時帶 `aria-current="page"`。
- 列表新的在上；`kind: 'game'` 的列顯示「遊戲更新 raw-20261105」，`'rebuild'` 顯示「重建」，`'first'` 顯示「第一版，沒有可比對的上一版」且沒有數量。
- 每列有「比對 zh-TW-20261029.7」與四種數量（為 0 的不顯示）；`other_files > 0` 時顯示「其他檔案 1」；tag 連到 `updateHref(tag)`。
- 載入失敗時顯示和其他頁相同的錯誤訊息。

- [ ] **Step 2: 跑測試，確認失敗**

Run: `cd site && npm test`
Expected: FAIL

- [ ] **Step 3: 實作**（樣式沿用 `styles.css` 的 token，種類標籤配色加四個新 token，深色模式也要定義；版面參考樣式草圖畫面 1）

- [ ] **Step 4: 跑測試，確認通過**

Run: `cd site && npm test && npm run typecheck && npm run build`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add site/src
git commit -m "feat(site): the list of Releases on the update page"
```

---

### Task 7: 單一 Release 頁與卡片顯示

**Files:**
- Create: `site/src/updates/UpdateView.tsx`、`site/src/updates/UpdateCard.tsx`
- Modify: `site/src/app/App.tsx`（換掉 Task 6 的佔位）、`site/src/styles.css`
- Test: `site/src/updates/UpdateView.test.tsx`

**Interfaces:**
- Consumes: Task 5 的 `loadUpdate`、`loadFile`、`filterEntries`、`countKinds`、`entryFiles`、`entryOrigins`、`updateHref`、`fileHref`、`Settings.updateView`；審核頁的 `variantDiff(base, other, {raw})` 與 `DiffText`、`ShownText`。
- Produces:
  ```ts
  export function UpdateView(props: { route: Extract<Route, { kind: 'update' }>; settings: Settings; onSettings: (s: Settings) => void }): JSX.Element
  export function UpdateCard(props: { entry: UpdateEntry }): JSX.Element
  export const UPDATE_PAGE = 300
  ```
  篩選的變更寫回網址（`location.hash = updateHref(...)`）；顯示方式寫進設定（`onSettings`）。Task 8 在 `settings.updateView === 'table'` 時改畫表格。

- [ ] **Step 1: 寫失敗的測試**

- 表頭：tag、日期、「遊戲更新 raw-20261105」或「重建」、「比對 <prev_tag>」、變動檔數；有其他檔案時列出路徑。
- 種類標籤四個，附 `countKinds` 的數量；點一個會改網址的 `k`。來源與檔案下拉改網址的 `o`、`f`。「隱藏不需翻譯的單位」預設勾選、旁邊寫數量；取消勾選改網址 `copy=1`。
- 依檔案分組，組名是檔案路徑。
- 卡片：kr-changed 顯示舊、新 KR 與舊、新譯文，差異以 `DiffText` 標色；new 只有新的；deleted 只有舊的且沒有「在檔案中開啟」；zh-changed 的 KR 只一份，並有「人工調整」／「補翻」／「重翻」標籤；其他卡片的「在檔案中開啟」連到 `fileHref(file, ptr)`。
- 「JP／EN（目前的）」點開才呼叫 `loadFile(file)`，顯示 pointer 相同那個單位的 jp／en；找不到時寫「目前的檔案裡找不到這個單位」；語言勾選沒有 JP 也沒有 EN 時不出現這個按鈕；KR 未勾時不顯示 KR 的前後。
- 「顯示原始標籤」開啟時，譯文與差異改用原始標籤（`variantDiff` 的 `raw`）。
- `shows 300 entries and then 300 more`：650 筆時先畫 300 張卡片，按「再顯示 300 筆」後 600，再按 650，按鈕消失；改篩選時回到 300。

- [ ] **Step 2: 跑測試，確認失敗**

Run: `cd site && npm test`
Expected: FAIL

- [ ] **Step 3: 實作**（舊的在上、新的在下；版面參考樣式草圖畫面 2 的 A）

- [ ] **Step 4: 跑測試，確認通過**

Run: `cd site && npm test && npm run typecheck && npm run build`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add site/src
git commit -m "feat(site): the changes of one Release, as cards"
```

---

### Task 8: 表格顯示與「卡片｜表格」切換

**Files:**
- Create: `site/src/updates/UpdateTable.tsx`
- Modify: `site/src/updates/UpdateView.tsx`、`site/src/styles.css`
- Test: `site/src/updates/UpdateView.test.tsx`

**Interfaces:**
- Consumes: Task 7 的 `UpdateView`、`UPDATE_PAGE`、差異標色的用法；`Settings.updateView`。
- Produces: `export function UpdateTable(props: { entries: UpdateEntry[] }): JSX.Element`。

- [ ] **Step 1: 寫失敗的測試**

- 篩選列有「卡片｜表格」切換；按「表格」呼叫 `onSettings` 帶 `updateView: 'table'`，畫面改成表格；以 `updateView: 'table'` 打開時直接是表格。
- 表格欄位：欄位、種類、KR、TW、來源；格子裡舊的在上、新的在下，有差異標色；檔案名稱是表格裡的分組列。
- 一列預設收合（CSS 只顯示前 3 行），點該列展開、再點收合；展開的狀態不寫進網址。
- 表格包在 `overflow-x: auto` 的容器裡。
- 分頁（300 筆、「再顯示 300 筆」）與卡片相同。

- [ ] **Step 2: 跑測試，確認失敗**

Run: `cd site && npm test`
Expected: FAIL

- [ ] **Step 3: 實作**（版面參考樣式草圖畫面 2 的 B）

- [ ] **Step 4: 跑測試，確認通過**

Run: `cd site && npm test && npm run typecheck && npm run build`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add site/src
git commit -m "feat(site): the changes of one Release, as a table, and the switch"
```

---

### Task 9: 真實資料驗收與文件

**Files:**
- Modify: `README.md`（網站一節：「更新」頁怎麼用；`.meta/history/` 已在 Task 3 寫過，這裡補網站的部分）
- 不改其他檔；驗收只在本機 `.work/` 與 scratchpad 裡做，不 push `tw`

**Interfaces:**
- Consumes: Task 1–8 的全部。

- [ ] **Step 1: 在本機真實資料上產生比對紀錄**

用最新的 `origin/raw`、`origin/tw` 與全部 Release tag 建 `.work/raw`、`.work/tw` 的工作樹（`git worktree`），`go build -o bin/lct ./cmd/lct`，在暫存的 tw 分支副本上跑一次會呼叫補齊的路徑（例：`bin/lct --json publish`，或寫一個只呼叫 `history.ReadRev`／`Make` 的一次性測試程式放在 scratchpad），不要 push。
Expected:
- `zh-TW-20261105.1.json` 的 `counts.new + copy_counts.new == 104`、`kr_changed` 合計 14、`deleted` 合計 12。
- `LobbyBGM-29.json` 的 `/dataList/39/name`、`/40/name`、`/41/name` 是 `kr-changed`。
- `EGOgift_MirrorDungeon.json` 的 `/dataList/277/desc` 是 `kr-changed`，舊譯文含「拚點威力+1」、新譯文含「拚點威力+2」，`old_origin` 是 `paratranz`、`new_origin` 是 `claude`。
- `.2` 的 entries 為空、`other_files` 只有 `Lang/zh-TW/Info/LICENSE-CONTENT`；`.3`–`.7` 的 entries 數依序為 2、132、596、53、6（全是 `zh-changed`）。
- 再跑一次，沒有任何檔案改變。

- [ ] **Step 2: 用這些紀錄跑網站**

`bin/lct site --out site/public/data`，`cd site && npm run build && npx vite preview`，以 Playwright（Chromium 已安裝）在 1280px 與 400px 寬、淺色與深色各截一張 `#/u` 與 `#/u/zh-TW-20261105.1`（卡片與表格各一），確認：列表有 8 個 Release；切換顯示方式後重新整理仍保持；400px 時 `document.documentElement.scrollWidth <= 400`。截圖放 scratchpad。

- [ ] **Step 3: README 補上「更新」頁的說明，跑完整檢查**

Run: `set -o pipefail; go vet ./... && go test ./... && cd site && npm ci && npm test && npm run typecheck && npm run build`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add README.md
git commit -m "docs: the update page of the site"
```
