# 第二階段：每週自動更新 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 讓 GitHub Actions 每週四自動把官方語言包的變化翻成 zh-TW 並發布 Release；人經由 `main` 的人工調整、重翻要求與改寫檔介入；處理 tool-jx3/limbus-translate#1 列出的 A、B、C 類事項。

**Architecture:** 新增 `lct plan`（§8.1 新舊原文對齊、§8.2 來源決策，套用人工調整、重翻要求與改寫檔）與 `lct translate`（以 Claude Code CLI 的 headless 模式反覆執行 `/translate-pending`，有花費與時間上限）。`update.yml`（排程）與 `rebuild.yml`（`main` 的規則變更）共用一個 reusable workflow `pipeline.yml`：fetch → snapshot → plan → translate → build → publish → Release → 審核 Issue。報表依 §14 分成「需要你處理」與「資訊」，需要處理的項目做成審核 Issue。

**Tech Stack:** Go 1.26.1（沿用 `gopkg.in/yaml.v3`、`golang.org/x/image/font/sfnt`，其餘標準庫）；git CLI；GitHub Actions；Claude Code CLI（npm 套件 `@anthropic-ai/claude-code`，固定版本 `2.1.316`）；`gh`、`jq`（runner 內建）。

**Spec:** `docs/superpowers/specs/2026-10-31-limbus-zh-tw-translation-design.md`；待辦來源：tool-jx3/limbus-translate#1

**本計畫範圍外（Plan 4）**：`rules-proposal.yml` 與 `anthropics/claude-code-action`（需要安裝 Claude GitHub App）、`rules-review`／`term-decision`／`review` skill、`lct glossary derive`、`rules/declined.yaml` 與建議 id、`ci.yml` 對規則 PR 的影響評估（§11.2、§12 的 `ci.yml` 最後一句）、§14.3「跨語言順序或 id 不一致的檔」清單（本計畫只沿用既有的「未對齊」單位數）、#1 的 D 類小項（本計畫改到同一段程式時順手修的除外，見各任務）。

## Global Constraints

- 單位 key、`kr_sha`、輸出 JSON 格式、`origin` 列舉（`paratranz`、`claude`、`human`、`copy`、`fallback`）、待譯原因（`new`、`kr-changed`、`adjustment-expired`、`paratranz-repair`、`requeue`、`structural-fail`）沿用 Plan 1，不改
- §8.2 來源順序：1 欄位規則判定不翻 → `copy`；2 凍結檔（`lct.yaml` 的 `frozen`，`BgmLyrics/**`）已有譯文 → 沿用，KR 變了也不動、只列入報表；3 人工調整對得上且 `kr_sha` = 目前 KR → `human`；4 既有紀錄 `kr_sha` = 目前 KR → 沿用；5 相同文字重用 → 沿用；6 其他 → 待譯
- 規則改了不自動改舊譯文：只有人寫進 `main` 的重翻要求（`rules/requeue/*.yaml`）與改寫檔（`rules/rewrite/*.yaml`）會改動既有紀錄，而且每個檔只套用一次（已套用的 id 記在 `tw` 的 `.meta/applied.json`）
- 人工調整（`rules/adjustments/*.yaml`）只在它的 `kr_sha` 等於目前 KR 時有效；查找順序：key 相同且 `kr_sha` 相同 → 套用；同檔同欄位 `kr_sha` 相同（多個取 `dataList` 索引最接近的）→ 套用並列為「key 已移動」；都沒有 → 過期，該單位重翻，舊調整當參考
- `update.yml`：cron `0 4 * * 4`（每週四 12:00 UTC+8）；`repository_dispatch` 類型 `limbus_update`；手動輸入 `force`、`requeue`、`max_usd`、`dry_run`。`rebuild.yml`：push 到 `main` 且動到 `rules/**`、`glossary/**`、`guides/**`、`lct.yaml`；手動輸入 `requeue`、`max_usd`、`dry_run`
- 寫入 `raw`／`tw` 的 workflow 共用 `concurrency` 群組 `lct-state`（`cancel-in-progress: false`）
- secrets：`ANTHROPIC_API_KEY`、`LIMBUS_STATUS_URL`；variables：`LIMBUS_CDN_TEMPLATE`、`LCT_MODEL`、`LCT_MAX_USD`（未設定時 30）
- `GITHUB_TOKEN` 權限：`contents: write`、`issues: write`
- 翻譯迴圈：`claude -p "/translate-pending" --output-format json --model "$LCT_MODEL" --max-turns <N>`；停止條件：沒有可領的待譯、累計 `total_cost_usd` ≥ 上限、接近時間上限（單一 job 6 小時）
- 審核 Issue 標題 `審核：YYYY-MM-DD 更新`（日期取 raw tag 的日期）；沒有需要處理的項目時不開 Issue
- 審核 Issue 的內文與留言各有 65,536 字元上限：內容超過時分段，第一段是 Issue 內文，其餘依序留言在同一個 Issue；每段不超過 60,000 字元，不把同一項拆到兩段
- 關鍵字名稱後加半形空格（使用者決定）：KR 用韓文寫狀態名、沒有 `[ID]` 時，中文狀態名後加一個半形空格（「2級破裂 強度」「2層強壯 ，」），因為遊戲以「關鍵字＋空格」配對顯示效果說明；`lct batch submit` 與 `lct adjust add` 在驗證前自動補上
- 需要設定的 secrets 與 variables 寫在 `README.md` 的「GitHub Actions 設定」
- Release tag `zh-TW-YYYYMMDD.N`；只在 `tw` 有新 commit、或該 commit 還沒有 Release 時才建立 Release
- skill、`prompts/`、workflow 不寫模型名稱；模型由 `--model` 或 `LCT_MODEL` 給
- `raw`、`tw` 分支與 Release 只由 CI 寫入；人經由 `main` 介入
- 每個 CLI 指令支援 `--json`；exit code 0 成功、1 有發現或使用者錯誤、2 系統錯誤
- commit 訊息結尾：
  ```
  Co-Authored-By: <your session's attribution line>
  Claude-Session: https://claude.ai/code/session_015XrjRvHe6FfxjemSS1js35
  ```
- 開發分支 `claude/festive-wright-g2k853`；每個任務結束 `git push -u origin claude/festive-wright-g2k853`；`main` 只在 Task 14 以 fast-forward 更新

## Review Focus

1. **整個劇情檔重新編號**（KR 在第 2 行插入一句，後面 200 句的 key 全部錯開）：不變的句子一句都不能進待譯，譯文要跟著搬到新 key — Task 3 `TestTransitionStoryInsertMovesRecords`、`TestTransitionDuplicateLines`
2. **同一次更新跑兩次 plan**（`update.yml` 之後 `rebuild.yml` 又跑一次，或手動重跑）：狀態不變，重翻要求與改寫檔不會套用第二次 — Task 6 `TestPlanTwiceIsIdempotent`
3. **上週沒翻完的待譯遇上這週的新版本**：待譯項目的 key 要跟著搬，租約清掉，`prev_kr` 不被覆寫成錯的版本 — Task 3 `TestTransitionMovesPendingItemsAndClearsLeases`
4. **一次 Claude 執行卡住或花費超過上限**：要能殺掉、停止迴圈，並照常打包已翻好的部分 — Task 9 `TestRunKillsAHungRun`、`TestRunStopsAtCostCap`
5. **審核 Issue 超過 GitHub 的 65,536 字元上限**（例如一次 3,000 筆 needs_review）：分成多段，第一段發 Issue、其餘依序留言在同一個 Issue，每一段都不超過上限、每一項都在 — Task 10 `TestReviewIssueSplits`

## File Structure

| 檔案 | 責任 | 任務 |
|---|---|---|
| `internal/gitx/archive.go` | 把某個 commit 的子目錄匯出成檔案（`git archive`） | 2 |
| `internal/raw/at.go` | 開啟舊 raw tag 的 KR 語言包 | 2 |
| `internal/plan/transition.go` | §8.1 新舊版本轉移（純函式） | 3 |
| `internal/adjust/adjust.go`、`cmd/lct/adjust.go` | 人工調整檔：讀、查找、新增；`lct adjust add` | 4 |
| `internal/requeue/files.go`、`internal/rewrite/rewrite.go` | 重翻要求檔與改寫檔：讀、寫、套用 | 5 |
| `internal/store/applied.go` | `.meta/applied.json` | 5 |
| `internal/plan/plan.go`、`cmd/lct/plan.go` | `lct plan`：§8.2 全流程與 `.meta/plan.json` | 6 |
| `internal/batch/twins.go` | 兄弟句（同 KR）的延後發放與送出時沿用 | 7 |
| `internal/rules/glossary.go`、`internal/rules/rules.go` | 詞彙條目的 `files` 範圍、外來名稱忽略清單 | 7 |
| `internal/validate/normalize.go`、`internal/validate/validate.go` | 關鍵字名稱後的空格、新增的警告 | 8 |
| `internal/translate/translate.go`、`cmd/lct/translate.go` | `lct translate`：CI 翻譯迴圈 | 9 |
| `internal/build/report.go`、`internal/build/issue.go` | §14 報表、審核 Issue | 10 |
| `internal/publish/publish.go` | release notes、Release 判斷 | 1、10 |
| `cmd/lct/search.go`、`.claude/skills/adjust/`、`.claude/skills/weekly-update/` | `lct search` 與兩個 skill | 11 |
| `.github/workflows/pipeline.yml`、`update.yml`、`rebuild.yml`、`ci.yml`、`README.md` | §12；secrets 與 variables 的說明 | 12 |
| `cmd/lct/e2e_phase2_test.go`、`testdata/phase2/` | §17 重播測試 | 13 |

---

### Task 1: Release 與報表的修正（#1 A1、A3、A6）

**Files:**
- Modify: `internal/batch/submit.go`、`internal/build/build.go`、`internal/build/report.go`、`internal/publish/publish.go`、`LICENSE-CONTENT`
- Test: `internal/batch/batch_test.go`、`internal/build/build_test.go`、`internal/publish/publish_test.go`

**Interfaces:**
- Produces:
  - `build.Info`（`.meta/build.json` 的型別，取代 `build` 內的 `buildMeta` 與 `publish.BuildInfo`）：`RawTag, RawToken, RulesCommit, BuiltAt string`、`Upstream map[string]string`、`Zips map[string]string`（zip 檔名 → sha256）、`Report Report`。JSON 鍵：`raw_tag`、`raw_token`、`rules_commit`、`built_at`、`upstream`、`zips`、`report`
  - `build.ReadInfo(twRoot string) (Info, error)`（沒有 build.json 時回傳 `build.ErrNotBuilt`）
  - `publish.Result` 加 `ReleaseNeeded bool` `json:"release_needed"`：`Changed` 為真，或 `TWCommit` 上沒有 `zh-TW-` tag
  - `publish.ErrDirtyRules`：build 的 `rules_commit` 以 `+dirty` 結尾時 `publish.Run` 回傳它，`lct publish` 轉成使用者錯誤（exit 1）

- [ ] **Step 1: 寫失敗的測試**
  - `TestSubmitFlagsAKeptRecordNeedsReview`：送回重譯的單位連續失敗 `maxAttempts` 次、保留原本的好紀錄（P12）時，該紀錄的 `Flags` 含 `needs_review`；之後 `build.Run` 的 `Report.NeedsReview` 為 1
  - `TestBuildDoesNotCountFrozenFallbacksAsNeedsReview`：一筆凍結檔的結構 fallback 只出現在 `FrozenFallback`，`NeedsReview` 為 0
  - `TestBuildRecordsZipChecksums`：`build.json` 的 `zips` 兩個鍵是兩個 zip 的檔名，值是實際檔案的 sha256
  - `TestPublishRefusesDirtyRules`：`rules_commit` 為 `abc+dirty` → `errors.Is(err, publish.ErrDirtyRules)`，`tw` 沒有新 commit
  - `TestPublishRefusesZipsOfAnotherBuild`：`out/` 的 zip 與 `zips` 的 sha256 不同 → 錯誤訊息含 zip 檔名
  - `TestPublishReleaseNeeded`：第一次 `Changed=true, ReleaseNeeded=true`；在 `TWCommit` 打 `zh-TW-20261029.1` 後再跑一次 → `Changed=false, ReleaseNeeded=false`
  - `TestNotesSeparatesImportedAndReferenceSources`：`Origins["paratranz"] > 0` 時 notes 有「匯入：paratranz: <repo@tag>」一行；`ltm`、`llc` 列在「第一階段的參考譯文（未匯入）」下；`Origins["paratranz"] == 0` 時不列任何上游
  - `TestNotesCountsEachProblemOnce`：凍結檔 fallback 3 筆、needs_review 0 筆 → 表格中「待人工審核」那一列是 0、「凍結檔需人工處理」是 3

- [ ] **Step 2: 執行測試確認失敗**

Run: `go test ./internal/batch/ ./internal/build/ ./internal/publish/`
Expected: FAIL（未定義 `build.Info`、`ReleaseNeeded`、`ErrDirtyRules` 等）

- [ ] **Step 3: 實作**
  - `submit.go`：P12 保留紀錄的分支加上 `store.FlagNeedsReview`
  - `build.go`：已列在 `FrozenFallback` 的紀錄不計入 `NeedsReview`；寫 zip 後計算 sha256 填入 `Info.Zips`
  - `publish.go`：改用 `build.ReadInfo`；先檢查 dirty、zip sha256，再 commit；`ReleaseNeeded` 用 `gitx.TagsAt(twRoot, "zh-TW-")`
  - `LICENSE-CONTENT`：來源段落寫成「ParatranzUploader（第一階段匯入）；LimbusTraditionalMandarin、LocalizeLimbusCompany（第一階段的參考譯文，未匯入）」

- [ ] **Step 4: 執行測試確認通過**

Run: `go vet ./... && go test ./...`
Expected: PASS

- [ ] **Step 5: Commit 並 push**

```bash
git add -A internal/ LICENSE-CONTENT
git commit -m "fix(build,publish): flag stuck requeues, count frozen fallbacks once, refuse dirty or stale builds"
git push -u origin claude/festive-wright-g2k853
```

### Task 2: 讀取舊版原文

**Files:**
- Create: `internal/gitx/archive.go`、`internal/raw/at.go`
- Test: `internal/gitx/gitx_test.go`、`internal/raw/raw_test.go`

**Interfaces:**
- Produces:
  - `gitx.Archive(repo, rev string, paths []string, dest string) error`：`git archive --format=tar <rev> -- <paths>` 的內容解到 `dest`；拒絕絕對路徑、含 `..` 與連結的項目
  - `raw.OpenKRAt(worktree, tag, dir string) (*pack.Pack, error)`：把 `tag` 的 `kr/` 匯出到 `dir` 再以 `pack.Open(dir+"/kr", "KR_")` 開啟；tag 不存在時錯誤訊息含 tag

- [ ] **Step 1: 寫失敗的測試**
  - `TestArchiveExtractsTheTreeOfARevision`：在暫存 repo commit `kr/KR_A.json`（內容 1）並打 tag `v1`，再改成內容 2 commit；`Archive(repo, "v1", ["kr"], dest)` 後 `dest/kr/KR_A.json` 是內容 1，位元組相同（含 BOM、CRLF）
  - `TestExtractTarRejectsUnsafeEntries`：對未匯出的 `extractTar(r io.Reader, dest string) error` 餵含 `../x`、`/abs`、symlink 的 tar → 錯誤，`dest` 外沒有新檔
  - `TestOpenKRAtReadsAnOlderSnapshot`：以 `raw.Snapshot` 建兩個 snapshot（tag `raw-20261029`、`raw-20261105`），`OpenKRAt(worktree, "raw-20261029", dir)` 的 `Doc("StoryData/S1.json")` 是第一版內容

- [ ] **Step 2: 執行測試確認失敗**

Run: `go test ./internal/gitx/ ./internal/raw/`
Expected: FAIL（未定義 `Archive`、`OpenKRAt`）

- [ ] **Step 3: 實作 `gitx.Archive` 與 `raw.OpenKRAt`**

`git archive` 的輸出以 `archive/tar` 解開（不呼叫 `tar` 指令）；`dir` 由呼叫者給，呼叫者負責刪除。

- [ ] **Step 4: 執行測試確認通過**

Run: `go vet ./... && go test ./internal/gitx/ ./internal/raw/`
Expected: PASS

- [ ] **Step 5: Commit 並 push**

```bash
git add internal/gitx internal/raw
git commit -m "feat(gitx,raw): read the KR pack of an older snapshot"
git push -u origin claude/festive-wright-g2k853
```

### Task 3: 新舊版本轉移（§8.1）

**Files:**
- Create: `internal/plan/transition.go`
- Test: `internal/plan/transition_test.go`

**Interfaces:**
- Consumes: `align.AlignVersions(file string, old, new *jsonx.Node) []align.VersionMatch`；`config.MatchAny(patterns []string, file string) bool`
- Produces:
  - `plan.Transition(old, new *pack.Pack, recs map[string]store.Record, pending []store.PendingItem, frozen []string, rawTag string) (*plan.Moved, error)`：不改輸入，回傳新的狀態
  - `plan.Moved{Records map[string]store.Record; Pending []store.PendingItem; Deleted []plan.DeletedUnit; Changes plan.Changes; FrozenChanged []string}`
  - `plan.DeletedUnit{File, Field, KR string; Record *store.Record}`（給 Task 6 的相同文字重用）
  - `plan.Changes{Unchanged, Moved, Changed, New, Deleted int; FilesAdded, FilesDeleted, FilesChanged, FilesShifted []string}`（排序；`FilesShifted` = 有不變單位換了 pointer 的檔，即中間插入）

規則（每個新單位依 `AlignVersions` 的狀態）：
- `unchanged`：紀錄與待譯項目搬到新 key；待譯項目的 `Lease` 清掉
- `changed`：凍結檔 → 紀錄搬到新 key、保留舊 `kr_sha`，key 列入 `FrozenChanged`；其他 → 刪除紀錄，待譯項目 `{Reason: "kr-changed", PrevKR: <舊 KR>, PrevZH: <舊紀錄的 zh，沒有則空>}`，原本在舊 key 的待譯項目保留 `Hint`
- `new`：待譯 `{Reason: "new"}`
- `deleted`：刪除紀錄與待譯項目，加入 `Deleted`
- 只在舊版的檔：全部 `deleted`；只在新版的檔：全部 `new`

- [ ] **Step 1: 寫失敗的測試**（以 `jsonx` 在測試內建小檔）
  - `TestTransitionStoryInsertMovesRecords`：`StoryData/S1.json` 舊版 5 行、新版在第 2 行前插入 1 行 → 5 筆紀錄搬到 `/dataList/i+1/content`，`Pending` 只有新插的那 1 筆，`Changes.Moved == 4`、`FilesShifted == ["StoryData/S1.json"]`
  - `TestTransitionDuplicateLines`：舊版有兩句相同的「……」，新版刪掉第一句 → 剩下的那句沿用紀錄，沒有待譯
  - `TestTransitionChangedQueuesWithPrevious`：一句 KR 被改 → 紀錄刪除，待譯 `kr-changed` 的 `PrevKR`、`PrevZH` 是舊的
  - `TestTransitionFrozenKeepsRecord`：`BgmLyrics/L.json` 的一句 KR 被改 → 紀錄保留（新 key、舊 `kr_sha`），`FrozenChanged` 含該 key，沒有待譯
  - `TestTransitionNewAndDeletedFiles`：新增檔的所有單位是 `new`、`FilesAdded` 含該檔；刪除檔的紀錄不在 `Records`、在 `Deleted`
  - `TestTransitionMovesPendingItemsAndClearsLeases`：舊 key 有一筆有租約、`Reason: "new"` 的待譯，該句不變但位置移動 → 新 key 有同一筆待譯、`Lease == nil`、`Reason` 仍是 `new`
  - `TestTransitionIDFieldAdded`：`Bufs.json` 有唯一 id，某條目新增 `desc` 欄位 → 只有那個欄位是 `new`，其他全部 `unchanged`

- [ ] **Step 2: 執行測試確認失敗**

Run: `go test ./internal/plan/`
Expected: FAIL（套件不存在）

- [ ] **Step 3: 實作 `plan.Transition`**

- [ ] **Step 4: 執行測試確認通過**

Run: `go vet ./... && go test ./internal/plan/`
Expected: PASS

- [ ] **Step 5: Commit 並 push**

```bash
git add internal/plan
git commit -m "feat(plan): carry records and pending items from the old KR to the new"
git push -u origin claude/festive-wright-g2k853
```

### Task 4: 人工調整（§11.1）

**Files:**
- Create: `internal/adjust/adjust.go`、`cmd/lct/adjust.go`、`rules/adjustments/.gitkeep`
- Modify: `internal/rules/lint.go`（檢查調整檔）
- Test: `internal/adjust/adjust_test.go`、`cmd/lct/adjust_test.go`、`internal/rules/rules_test.go`

**Interfaces:**
- Produces:
  - `adjust.Adjustment{Key, Raw, KRSha, ZH, Reason, Rule string; File string /*來源檔，不寫入 YAML*/}`，YAML 鍵 `key`、`raw`、`kr_sha`、`zh`、`reason`、`rule`
  - `adjust.Load(root string) ([]adjust.Adjustment, error)`（讀 `rules/adjustments/*.yaml`，依檔名、檔內順序）
  - `adjust.Resolve(a adjust.Adjustment, kr *pack.Pack) adjust.Resolution`；`Resolution{Status adjust.Status; Key string}`；`Status` 為 `"applied"`、`"moved"`、`"expired"`
  - `adjust.Add(root string, a adjust.Adjustment, date, topic string) (path string, err error)`：寫入或附加到 `rules/adjustments/<date>-<topic>.yaml`；同檔已有同 key → 錯誤
  - `lct adjust add --key K --zh Z --reason R --rule PATH|one-off --topic T [--date YYYY-MM-DD]`：`zh` 先 `validate.Normalize`，再以命中的詞彙條目做 `validate.Check`（新譯文，內容類為硬性）；有硬性問題 → exit 1 並輸出問題；通過 → 寫檔並輸出 `{path, key, zh}`
  - `rules lint`：調整檔的每筆要有全部 6 個鍵、`kr_sha` 為 64 位小寫 hex、`rule` 是存在的檔案路徑或 `one-off`、`zh` 非空、所有調整檔中同一個 key 只出現一次

- [ ] **Step 1: 寫失敗的測試**
  - `TestResolveApplied`：key 所指單位的 `kr_sha` 相同 → `applied`、`Key` 不變
  - `TestResolveMovedPicksTheNearest`：KR 插行後同檔同欄位有兩句 `kr_sha` 相同（`dataList` 索引 3 與 9），調整原本在索引 2 → `moved`、`Key` 是索引 3 那句
  - `TestResolveExpired`：同檔同欄位沒有相同 `kr_sha` → `expired`
  - `TestAddCreatesAndAppends`：第一次建檔、第二次附加；YAML 依序含兩筆；同 key 再加 → 錯誤
  - `TestAdjustAddRejectsHardIssues`：`--zh` 少了 KR 的 `[Burn]` → exit 1，沒有寫檔
  - `TestAdjustAddWritesTheRawTagAndShaOfTheCurrentKR`：寫入的 `raw` 是目前 raw tag、`kr_sha` 是目前 KR 的 sha
  - `TestLintAdjustments`：錯的 sha、不存在的 rule 檔、兩個檔有同 key → 各一個錯誤

- [ ] **Step 2: 執行測試確認失敗**

Run: `go test ./internal/adjust/ ./internal/rules/ ./cmd/lct/ -run 'Resolve|Adjust|LintAdjust'`
Expected: FAIL

- [ ] **Step 3: 實作**

`Resolve` 的「最接近」以 pointer 的第一個 `dataList` 索引差的絕對值比較，相同時取較前面的。

- [ ] **Step 4: 執行測試確認通過**

Run: `go vet ./... && go test ./...`
Expected: PASS

- [ ] **Step 5: Commit 並 push**

```bash
git add internal/adjust cmd/lct/adjust.go cmd/lct/adjust_test.go internal/rules rules/adjustments
git commit -m "feat(adjust): human adjustment files and lct adjust add"
git push -u origin claude/festive-wright-g2k853
```

### Task 5: 重翻要求檔與改寫檔

**Files:**
- Create: `internal/requeue/files.go`、`internal/rewrite/rewrite.go`、`internal/store/applied.go`、`rules/requeue/.gitkeep`、`rules/rewrite/.gitkeep`
- Modify: `cmd/lct/requeue.go`、`internal/rules/lint.go`、spec §7 表格與 §11.1 第 5 步（加入 `rules/rewrite/`）
- Test: `internal/requeue/requeue_test.go`、`internal/rewrite/rewrite_test.go`、`internal/store/store_test.go`、`cmd/lct/requeue_test.go`

**Interfaces:**
- Produces:
  - 重翻要求檔 `rules/requeue/<YYYY-MM-DD>-<topic>.yaml`：`reason`（必填）、`keys`、`files`、`origins`（同 `lct requeue` 的意義，至少一個 key 或 file）；id = 檔名去掉 `.yaml`
  - `requeue.File{ID, Reason string; Selector requeue.Selector}`；`requeue.LoadFiles(root string) ([]requeue.File, error)`；`requeue.WriteFile(root, date, topic string, sel requeue.Selector, reason string) (path string, err error)`
  - `lct requeue --to-file TOPIC [--date YYYY-MM-DD]`：寫要求檔、不碰 `tw`；與 `--needs-review`、`--leased` 互斥
  - 改寫檔 `rules/rewrite/<YYYY-MM-DD>-<topic>.yaml`：`reason`、`files`（glob，必填）、`origins`（預設 `paratranz`、`claude`）、`pattern`（Go regexp，對 zh）、`replace`（`regexp.ReplaceAllString` 的替換字串）、`kr_pattern`（選填；KR 也要符合）
  - `rewrite.File{ID, Reason string; Files, Origins []string; Pattern, Replace, KRPattern string}`；`rewrite.LoadFiles(root string) ([]rewrite.File, error)`
  - `rewrite.Apply(f rewrite.File, recs map[string]store.Record, kr *pack.Pack, ok func(kr, zh string) bool) []string`：改寫符合的紀錄的 `ZH`，改寫後 `ok` 為假的不改；回傳改了的 key（排序）；不改 `Origin`
  - `store.Applied{Requeue []string; Rewrite []string}`（JSON `requeue`、`rewrite`）；`(*Store).LoadApplied() (store.Applied, error)`、`(*Store).SaveApplied(store.Applied) error`（`.meta/applied.json`）
  - `rules lint`：兩種檔的鍵、regexp 可編譯、glob 有效、`reason` 非空

- [ ] **Step 1: 寫失敗的測試**
  - `TestWriteRequeueFileAndLoadIt`：寫入後 `LoadFiles` 讀回同樣的 `Selector` 與 `Reason`，`ID` 是 `2026-11-05-honglu`
  - `TestLoadRequeueFilesRejectsUnknownKeys`：多一個鍵 `foo` → 錯誤含檔名
  - `TestRequeueToFileTouchesNoState`：`lct requeue --to-file x --file 'StoryData/S1*' --reason r` 後 `.work/tw/.meta/pending.jsonl` 沒變、檔案存在
  - `TestRewriteApply`：`pattern: '(\d+級[\p{Han}]{1,8}) (強度)'`、`replace: '$1$2'` → 「2級破裂 強度」改成「2級破裂強度」；不在 `files` 的檔不改；`origin` 為 `human` 的不改；`ok` 回傳假的不改
  - `TestRewriteKRPattern`：設定 `kr_pattern` 時 KR 不符合就不改
  - `TestAppliedRoundTrip`：寫入讀回相同；沒有檔時是空的

- [ ] **Step 2: 執行測試確認失敗**

Run: `go test ./internal/requeue/ ./internal/rewrite/ ./internal/store/ ./cmd/lct/ -run 'RequeueFile|ToFile|Rewrite|Applied'`
Expected: FAIL

- [ ] **Step 3: 實作**

- [ ] **Step 4: 執行測試確認通過**

Run: `go vet ./... && go test ./...`
Expected: PASS

- [ ] **Step 5: Commit 並 push**

```bash
git add -A internal/requeue internal/rewrite internal/store cmd/lct rules docs/superpowers/specs
git commit -m "feat(requeue,rewrite): requeue and rewrite files that main carries and CI applies once"
git push -u origin claude/festive-wright-g2k853
```

### Task 6: `lct plan`（§8.2）

**Files:**
- Create: `internal/plan/plan.go`、`cmd/lct/plan.go`
- Test: `internal/plan/plan_test.go`、`cmd/lct/plan_test.go`

**Interfaces:**
- Consumes: Task 2 `raw.OpenKRAt`；Task 3 `plan.Transition`；Task 4 `adjust.Load`、`adjust.Resolve`；Task 5 `requeue.LoadFiles`、`rewrite.LoadFiles`、`rewrite.Apply`、`(*Store).LoadApplied/SaveApplied`；`requeue.Apply`；`fields.Load`、`(*fields.Rules).Classify`、`fields.Learn`、`(*fields.Learned).Diff`；`rules.Load`；`validate.Check`
- Produces:
  - `plan.Inputs{Root, RawWorktree, TWRoot, RulesCommit, BuildRawTag string; Src pack.Set; Frozen []string; Fields *fields.Rules; Rules *rules.Rules; TmpDir string}`；`BuildRawTag` 是 `.meta/build.json` 的 `raw_tag`，由 `cmd/lct/plan.go` 以 `build.ReadInfo` 讀出傳入（`plan` 不 import `build`，因為 Task 10 的 `build` 會 import `plan`）
  - `plan.Run(in plan.Inputs) (*plan.Summary, error)`；同時寫 `.meta/plan.json`（內容就是 `Summary`）
  - `plan.Summary`（JSON 鍵在括號內）：`RawTag`（`raw_tag`）、`PrevRawTag`（`prev_raw_tag`）、`RulesCommit`（`rules_commit`）、`Changes plan.Changes`（`changes`，沒有版本轉移時全為零）、`Copy int`（`copy`，這次新建的 copy 紀錄數）、`Reused int`（`reused`）、`Adjustments plan.AdjustmentSummary`（`adjustments`：`applied int`、`moved []plan.MovedKey{From, To}`、`expired []string`）、`RequeueApplied []plan.Applied`（`requeue_applied`，`{id, units}`）、`RewriteApplied []plan.Applied`（`rewrite_applied`）、`FrozenChanged []string`（`frozen_changed`）、`NewSpeakers []string`（`new_speakers`）、`FieldChanges []fields.Change`（`field_changes`）、`NewFilesDefaultFamily []string`（`new_files_default_family`）、`Pending int`（`pending`，可領的待譯數）
  - `lct plan`（無參數）：輸出 `Summary`

流程（全程持有 store 鎖；requeue 套用另行加鎖）：
1. 上一版 = `.meta/plan.json` 的 `raw_tag`，沒有就用 `in.BuildRawTag`，都沒有就等於目前版本
2. 上一版 ≠ 目前版本 → `raw.OpenKRAt` + `plan.Transition`；`NewSpeakers` = 新版劇情與 RPG 對話中、舊版沒有的 `model`／`speaker` 值
3. 每個目前的 KR 單位：欄位規則判定不翻 → 確保有 `copy` 紀錄（`zh` = KR）並移除其待譯項目
4. 人工調整：`applied`／`moved` → `human` 紀錄（`Batch: "adjustment:<檔名>"`），移除待譯項目；`expired` → 若目前沒有 `kr_sha` 相同的非 `human` 紀錄，加待譯 `{Reason: "adjustment-expired", Refs: {"adjustment": <舊 zh>}}`
5. 相同文字重用：`new`、`kr-changed` 待譯項目的 KR 與某筆紀錄（目前紀錄或轉移的 `Deleted`）的 KR 完全相同、同族群、同欄位名、劇情則同說話者 → 複製其 `zh`、`Origin`、`Model`，`Batch: "reuse:<來源 key>"`、`Since` = 目前 raw tag、`Rules` = 目前 rules commit；KR 少於 10 個字元時只限同檔；複製的 `zh` 對新單位跑 `validate.Check`（新譯文）有硬性問題就不重用
6. 依檔名順序套用 `applied.json` 沒有記錄的重翻要求檔（`requeue.Apply`，hint = `reason`）與改寫檔（`rewrite.Apply`，`ok` = 結構類與內容類都沒有硬性問題），把 id 加進 `applied.json`
7. `fields.Learn` 與 `rules/fields.yaml` 比對 → `FieldChanges`；`FilesAdded` 中族群是預設族群的檔 → `NewFilesDefaultFamily`

- [ ] **Step 1: 寫失敗的測試**（`plan_test.go` 以暫存 repo 與兩個 snapshot 建資料）
  - `TestPlanWithoutTransitionChangesNothing`：上一版等於目前版本、沒有規則檔 → 紀錄與待譯和執行前相同
  - `TestPlanTwiceIsIdempotent`：版本轉移＋一個重翻要求檔＋一個改寫檔，跑兩次 → 第二次的紀錄、待譯、`applied.json` 與第一次相同，第二次的 `RequeueApplied`、`RewriteApplied` 為空
  - `TestPlanCopiesUnitsTheFieldRulesSkip`：新增的 `key` 欄位 → `copy` 紀錄，不進待譯
  - `TestPlanAdjustmentBeatsTheRecord`：`claude` 紀錄與 `kr_sha` 相同的調整 → 結果是 `human`、`zh` 是調整的
  - `TestPlanQueuesAnExpiredAdjustmentOnce`：KR 改了 → 待譯 `adjustment-expired`；把翻好的 `claude` 紀錄寫入後再跑 → 不再加待譯，`Expired` 仍列出
  - `TestPlanReusesTheSameText`：新檔的一句 KR 與既有紀錄相同、同族群同欄位 → 沿用；KR 只有 3 個字元的另一檔 → 不沿用；來源 `zh` 含禁用譯法 → 不沿用
  - `TestPlanReusesADeletedUnit`：id 重新編號（舊條目刪除、新條目新增，KR 相同）→ 沿用被刪除單位的譯文
  - `TestPlanReportsNewSpeakersAndFiles`：新版多一個 `model` 與一個預設族群的新檔 → 分別列在 `NewSpeakers`、`NewFilesDefaultFamily`
  - `cmd/lct`：`TestPlanCommandWritesPlanJSON`：`lct plan --json` 的輸出與 `.meta/plan.json` 相同

- [ ] **Step 2: 執行測試確認失敗**

Run: `go test ./internal/plan/ ./cmd/lct/ -run 'Plan'`
Expected: FAIL

- [ ] **Step 3: 實作 `plan.Run` 與 `lct plan`**

- [ ] **Step 4: 執行測試確認通過**

Run: `go vet ./... && go test ./...`
Expected: PASS

- [ ] **Step 5: Commit 並 push**

```bash
git add internal/plan cmd/lct/plan.go cmd/lct/plan_test.go
git commit -m "feat(plan): decide the source of every unit for a new raw snapshot or new rules"
git push -u origin claude/festive-wright-g2k853
```

### Task 7: 兄弟句、詞彙範圍、外來名稱忽略清單（#1 C）

**Files:**
- Create: `internal/batch/twins.go`、`rules/foreign-names.ignore.yaml`
- Modify: `internal/batch/next.go`、`internal/batch/submit.go`、`internal/rules/glossary.go`、`internal/rules/rules.go`、`internal/rules/lint.go`、spec §7（`files` 範圍、忽略清單）
- Test: `internal/batch/batch_test.go`、`internal/rules/rules_test.go`

**Interfaces:**
- Produces:
  - 兄弟句：同族群、同欄位名、KR 完全相同的待譯單位（劇情另需同說話者）
  - `batch.Next`：某單位的兄弟句正被另一個未過期的批次租用時，這次不發放它
  - `batch.Submit`：收下一個單位後，把同一份 `zh` 寫給它的兄弟句（沒有租約或租約已過期、沒有 `needs_review`），每個兄弟句各自跑 `validate.Check`，沒有硬性問題才寫；寫入的紀錄 `Batch` 為 `<批次 id>`、`Model` 同送出者；`SubmitResult` 加 `Reused []string` `json:"reused"`
  - `rules.Term` 加 `Files []string` `yaml:"files" json:"files,omitempty"`（glob，空 = 全部檔）；只在檔名符合時注入批次、檢查禁用譯法；`rules lint` 檢查 glob，並允許同一個 `kr` 有多個條目，條件是其中至多一個沒有 `files`
  - `rules.Rules` 加 `ForeignIgnore []string`（`rules/foreign-names.ignore.yaml`，YAML 字串清單）；`batch.Submit` 的外來名稱警告略過清單中的詞
  - `glossary/terms.yaml` 的「신부」改成兩個條目：`zh: 牧師`、`files` 為第十章的檔（`*a1c10*`、`StoryData/S10*`、`RPGSystem/**`、`Enemies-a1c10*`）、`forbidden: [神父]`；以及 `zh: 神父`（其他檔，不設 `files`，`notes` 指向前者）。忽略清單初始內容：`媽媽`、`獨白`

- [ ] **Step 1: 寫失敗的測試**
  - `TestNextDefersTwinsOfLeasedUnits`：A、B 兩檔各有一句相同 KR 的待譯；第一次 `Next` 拿到 A 檔 → 第二次 `Next`（`--family` 限定 B 所在族群）不發放 B 的那句
  - `TestSubmitReusesForPendingTwins`：送出 A 的那句 → B 的那句有紀錄、`zh` 相同、從待譯移除，`Reused` 含 B 的 key
  - `TestSubmitSkipsATwinThatFailsTheCheck`：B 所在檔的詞彙條目禁用那個譯法 → B 不沿用、仍在待譯
  - `TestGlossaryFileScope`：`files: ["StoryData/S10*"]` 的條目只出現在 `StoryData/S1001A.json` 的批次，不出現在 `StoryData/S2001A.json` 的批次；後者用了禁用譯法也通過
  - `TestForeignNameIgnoreList`：清單含「媽媽」→ 送出含「媽媽」的譯文沒有 `foreign-name` 警告
  - `TestLintGlossaryFiles`：`files: ["["]` → lint 錯誤；同一個 `kr` 兩個條目都沒有 `files` → lint 錯誤，其中一個有 `files` → 通過

- [ ] **Step 2: 執行測試確認失敗**

Run: `go test ./internal/batch/ ./internal/rules/`
Expected: FAIL

- [ ] **Step 3: 實作**

- [ ] **Step 4: 執行測試確認通過**

Run: `go vet ./... && go test ./... && go build -o bin/lct ./cmd/lct && bin/lct rules lint`
Expected: PASS；lint 無錯

- [ ] **Step 5: Commit 並 push**

```bash
git add -A internal/batch internal/rules rules glossary docs/superpowers/specs
git commit -m "feat(batch,rules): translate identical lines once, scope glossary terms by file, ignore common words"
git push -u origin claude/festive-wright-g2k853
```

### Task 8: 關鍵字名稱後的空格與新增的驗證警告（使用者決定、#1 C）

**Files:**
- Modify: `internal/validate/normalize.go`、`internal/validate/validate.go`、`internal/build/build.go`、`internal/batch/batch.go`、`internal/batch/submit.go`、`cmd/lct/batch.go`、`cmd/lct/adjust.go`、`guides/base.md`
- Test: `internal/validate/normalize_test.go`、`internal/validate/validate_test.go`、`internal/batch/batch_test.go`

**Interfaces:**
- Produces:
  - `validate.NormalizeKeywordSpace(zh string, names []string) string`：`names` 中的名稱緊接在 `\d+(級|層|回)` 之後、後面不是半形空格時（包括字串結尾），在名稱後插入一個半形空格；名稱依長度由長到短比對（「破裂易損」先於「破裂」）
  - `batch.Env` 加 `KeywordNames []string`：`cmd/lct` 以 `names.KeywordTable` 從目前 `Bufs*`、`BattleKeywords*` 的紀錄取出全部中文狀態名；`batch.Submit` 在 `validate.Normalize` 之後呼叫 `NormalizeKeywordSpace`；`lct adjust add` 同樣處理
  - `guides/base.md` 的「KR 寫韓文狀態名、沒有 `[ID]` 時」改成：寫官方中文名，名稱後加一個半形空格（「1級燒傷 強度」「2層守護 」「2層強壯 ，」）；遊戲以「關鍵字＋空格」配對顯示效果說明；`lct batch submit` 會自動補上。刪掉「新譯與修改時不加空格」與「照抄兄弟檔時空格拿掉」兩句
  - `validate.Context` 加 `File string`
  - 半形標點警告不再對 `E.G.O::`（兩個以上連續的 `:`）報警
  - 新警告 `placeholder-bracket`：zh 有 `[{n}` 而沒有緊接的 `]`，或 `{n}]` 前沒有 `[`，且 KR 的方括號是成對的
  - 新警告 `halfwidth-bracket`：`File` 在 `RPGSystem/` 下、檔名不含 `dialogue`，zh 含半形 `(` 或 `)`

- [ ] **Step 1: 寫失敗的測試**
  - `TestNormalizeKeywordSpace`：`names = [破裂, 破裂易損, 強壯]`；「2級破裂強度」→「2級破裂 強度」；「2層強壯，」→「2層強壯 ，」；「3回破裂易損」→「3回破裂易損 」；「2級破裂 強度」不變；「破裂的力量」（前面沒有數量）不變
  - `TestSubmitAddsKeywordSpace`：out 檔的 zh 是「使目標增加2級破裂強度」→ 收下的紀錄是「使目標增加2級破裂 強度」
  - `TestHalfwidthPunctIgnoresDoubleColon`：`「腦葉公司E.G.O:: 淚鋒之劍 羅佳」` 沒有 `halfwidth-punct`
  - `TestPlaceholderBracket`：KR `[{0}]이(가)…`、zh `<color>[{0}</color>獲得了` → 一個 `placeholder-bracket` 警告；zh `[{0}]獲得了` → 沒有
  - `TestHalfwidthBracketInRPGLabels`：`File: "RPGSystem/rpg-loc-quest-floor-3.json"`、zh `(內部)…` → 警告；`File: "RPGSystem/rpg-loc-dialogue-floor-3.json"` → 沒有

- [ ] **Step 2: 執行測試確認失敗**

Run: `go test ./internal/validate/ ./internal/batch/`
Expected: FAIL

- [ ] **Step 3: 實作**（三個新警告都是 `ClassStyle`）

- [ ] **Step 4: 執行測試確認通過**

Run: `go vet ./... && go test ./...`
Expected: PASS

- [ ] **Step 5: Commit 並 push**

```bash
git add internal/validate internal/build internal/batch cmd/lct guides/base.md
git commit -m "feat(validate): space after keyword names; warn on unbalanced placeholder brackets and half-width brackets in RPG labels"
git push -u origin claude/festive-wright-g2k853
```

### Task 9: `lct translate`（CI 翻譯迴圈，§9.2）

**Files:**
- Create: `internal/translate/translate.go`、`cmd/lct/translate.go`
- Modify: `internal/build/build.go`、`internal/build/report.go`（`build.json`、`version.json` 記模型與花費）
- Test: `internal/translate/translate_test.go`、`internal/build/build_test.go`

**Interfaces:**
- Produces:
  - `translate.Options{Root string; Claude string /*預設 "claude"*/; Model string; MaxUSD float64; Deadline time.Time; MaxTurns int; RunTimeout time.Duration; Available func() (int, error); Log io.Writer}`
  - `translate.Run(opt translate.Options) (*translate.Result, error)`：每輪執行 `<Claude> -p /translate-pending --output-format json --model <Model> --max-turns <MaxTurns>`，工作目錄 `Root`、環境變數加 `LCT_MODEL=<Model>`；解析 stdout 的 JSON 物件的 `total_cost_usd`、`is_error`、`subtype`、`num_turns`
  - 停止條件（依序檢查，`Result.Stopped` 記原因）：`Available()` 為 0 → `"done"`；累計花費 ≥ `MaxUSD` → `"cost"`；現在 + `RunTimeout` 超過 `Deadline` → `"deadline"`；連續兩輪 `Available()` 沒有減少 → `"no-progress"`；一輪 exit code 不是 0、`is_error` 為真或輸出不是 JSON → `"error"`（訊息記在 `Result.Error`）；一輪超過 `RunTimeout` 就殺掉該行程群組並記為 `"error"`
  - `translate.Result{Model string; Runs int; CostUSD float64; Stopped, Error string; PendingBefore, PendingAfter int; RawTag, RulesCommit string}`；`translate.Run` 不填 `RawTag`、`RulesCommit`，由 `lct translate` 填入目前的 raw tag 與 rules commit 後寫入 `.meta/translate.json`
  - `lct translate --max-usd F --deadline D --max-turns N --run-timeout D [--claude PATH] [--model M]`：`--model` 預設 `$LCT_MODEL`，都沒有 → exit 1；`--deadline` 是從現在起算的時間長度（例如 `5h`）；停止原因是 `"error"` 時 exit 2，其他 exit 0
  - `build.Info` 加 `Model string` `json:"model,omitempty"`、`CostUSD float64` `json:"cost_usd,omitempty"`：`.meta/translate.json` 的 `raw_tag` 與 `rules_commit` 和這次建置相同時才帶入；`Info/version.json` 同樣帶這兩個欄位

- [ ] **Step 1: 寫失敗的測試**（假的 `claude`：測試在 `t.TempDir()` 寫一個 shell script，依環境變數決定行為：把參數與 `LCT_MODEL` 寫進檔案、遞減計數檔、印出 JSON、或 `sleep`）
  - `TestRunStopsWhenNothingIsPending`：計數 3 → 0 用 3 輪，`Stopped == "done"`、`CostUSD` 是三輪總和
  - `TestRunStopsAtCostCap`：每輪 0.4、`MaxUSD` 1.0 → 3 輪後 `Stopped == "cost"`
  - `TestRunStopsWithoutProgress`：計數不變 → 2 輪後 `"no-progress"`
  - `TestRunStopsOnAnError`：script 印 `{"is_error": true}` → 1 輪後 `"error"`，`Error` 非空
  - `TestRunKillsAHungRun`：script `sleep 60`、`RunTimeout` 200ms → 1 秒內回傳，`"error"`，沒有殘留行程
  - `TestRunPassesModelAndArguments`：記錄的參數依序是 `-p /translate-pending --output-format json --model m --max-turns 7`，環境 `LCT_MODEL=m`
  - `TestBuildRecordsModelAndCost`：`.meta/translate.json` 與這次建置同版本 → `build.json` 與 `version.json` 有 `model`、`cost_usd`；不同 raw tag → 沒有

- [ ] **Step 2: 執行測試確認失敗**

Run: `go test ./internal/translate/ ./internal/build/`
Expected: FAIL

- [ ] **Step 3: 實作**（以 `exec.CommandContext` 加 `SysProcAttr{Setpgid: true}`，逾時時殺整個行程群組）

- [ ] **Step 4: 執行測試確認通過**

Run: `go vet ./... && go test ./...`
Expected: PASS

- [ ] **Step 5: Commit 並 push**

```bash
git add internal/translate cmd/lct/translate.go internal/build
git commit -m "feat(translate): run the translate-pending skill headless until done, out of budget or out of time"
git push -u origin claude/festive-wright-g2k853
```

### Task 10: 報表、審核 Issue、release notes（§14、#1 A2）

**Files:**
- Create: `internal/build/issue.go`
- Modify: `internal/build/build.go`、`internal/build/report.go`、`cmd/lct/build.go`、`internal/publish/publish.go`、`internal/batch/submit.go`
- Test: `internal/build/build_test.go`、`internal/build/issue_test.go`、`internal/publish/publish_test.go`、`internal/batch/batch_test.go`

**Interfaces:**
- Consumes: Task 6 `.meta/plan.json`（`plan.Summary`，以 `store.ReadMetaJSON` 讀，`raw_tag` 與這次建置不同就當作沒有）；Task 9 `.meta/translate.json`
- Produces:
  - `build.Report` 加：`WarningSamples map[string][]string`（每種警告前 10 個 key，排序）、`Inconsistent []build.Inconsistency`（同族群、同欄位、同 KR 而 `zh` 不同的組，依組大小遞減取前 20；`Inconsistency{Family, Field, KR string; ZH []string; Keys []string}`）、`Notes []store.Note`（上次建置之後新增的 note）、`NewTranslated int`（`Since` 為這次 raw tag 的 `claude` 紀錄數）
  - `build.Info` 加 `NotesSeen int` `json:"notes_seen"`（建置當時 note 的總數；下次只報告之後的）
  - `build.Options` 加 `RepoURL string`（例如 `https://github.com/tool-jx3/limbus-translate`；空字串時不產生比較頁連結）、`PrevTWCommit string`
  - 報表 `.meta/reports/<raw tag>.md` 依 §14.2、§14.3 分成「需要你處理」與「資訊」兩節；比較頁連結 `<RepoURL>/compare/<prev raw tag>...<raw tag>` 與 `<RepoURL>/compare/<PrevTWCommit>...tw`
  - `build.ReviewIssue(info build.Info, sum *plan.Summary, pending []store.PendingItem) (title string, parts []string, ok bool)`：需要處理的項目 = needs_review 的待譯（key 與最後一次的錯誤）、過期的人工調整、凍結檔 KR 有變、凍結檔 fallback、新說話者、欄位規則變化、預設族群的新檔、翻譯迴圈的 `Stopped` 是 `"error"`；沒有任何一項時 `ok == false`。每項是 `- [ ] ` 勾選清單。`parts[0]` 是 Issue 內文，其餘是依序要留言的內容；每段不超過 60,000 字元，只在兩項之間切開（單一項本身超過上限時截斷該項並註明）；分成多段時每段開頭寫「（第 i／M 段）」，第一段另寫完整報表的位置 `.meta/reports/<raw tag>.md`
  - `lct build` 寫 `out/summary.md`（完整報表，超過 1,000,000 位元組時截斷）；`ReviewIssue` 為真時寫 `out/review-issue/title.txt` 與 `out/review-issue/01.md`、`02.md`…（兩位數，依序），否則刪掉整個 `out/review-issue/`；`buildOutput` 加 `ReviewIssueParts int` `json:"review_issue_parts"`（段數，沒有 Issue 時為 0）。`RepoURL` 取 `$GITHUB_SERVER_URL/$GITHUB_REPOSITORY`，`PrevTWCommit` 取建置前 `tw` 的 HEAD
  - release notes 依 §14.1 精簡：這次的原文版本與變化數（新增／修改／刪除單位）、各來源比例（百分比，一位小數）、仍顯示英文的數量（pending ＋ fallback）、授權與來源（Task 1 的寫法）；不再列完整表格
  - `batch.Submit`：所有 key 驗證完才寫 note（批次檔有格式錯的 key 時什麼都不寫）

- [ ] **Step 1: 寫失敗的測試**
  - `TestReportHasBothSections`：有 needs_review 與 plan 變化時，報表依序有「## 需要你處理」「## 資訊」，資訊節含兩個比較頁連結
  - `TestReportListsNewNotesOnce`：建置一次（2 筆 note）、加 1 筆後再建置 → 第二份報表只列 1 筆，`notes_seen == 3`
  - `TestReportWarningSamples`：12 筆 `kana` 警告 → `WarningSamples["kana"]` 有 10 個 key、排序
  - `TestReportInconsistentRenderings`：同族群同欄位同 KR 兩種 `zh` → 一組，`ZH` 兩個、`Keys` 兩個
  - `TestReviewIssueOnlyWithItems`：沒有任何需要處理的項目 → `ok == false`，`lct build` 刪掉舊的 `out/review-issue/`
  - `TestReviewIssueChecklist`：每類各一項 → body 每項以 `- [ ] ` 開頭，標題 `審核：2026-11-05 更新`
  - `TestReviewIssueSplits`：3,000 筆 needs_review → 多段，每段 ≤ 60,000 字元、開頭有「（第 i／M 段）」；所有段合起來恰好含 3,000 個 `- [ ] ` 項目，沒有一項被拆開；`lct build` 寫出的 `01.md`…檔數等於 `review_issue_parts`
  - `TestNotesAreConcise`：notes 有比例（例如 `paratranz 60.0%`）與「仍顯示英文」一行，沒有完整報表的表格
  - `TestSubmitNoteWithABadKeyWritesNothing`：out 檔一筆合法、一筆 key 格式錯且有 note → 錯誤，`notes.jsonl` 沒變

- [ ] **Step 2: 執行測試確認失敗**

Run: `go test ./internal/build/ ./internal/publish/ ./internal/batch/`
Expected: FAIL

- [ ] **Step 3: 實作**

- [ ] **Step 4: 執行測試確認通過**

Run: `go vet ./... && go test ./...`
Expected: PASS

- [ ] **Step 5: Commit 並 push**

```bash
git add internal/build internal/publish internal/batch cmd/lct
git commit -m "feat(build,publish): report what needs a person, the review issue and concise release notes"
git push -u origin claude/festive-wright-g2k853
```

### Task 11: `lct search`、`adjust` 與 `weekly-update` skill（§9.3、§11.1、#1 A4）

**Files:**
- Create: `cmd/lct/search.go`、`.claude/skills/adjust/SKILL.md`、`.claude/skills/weekly-update/SKILL.md`
- Modify: `internal/batch/batch.go`（匯出說話者）、`cmd/lct/contract_test.go`
- Test: `cmd/lct/search_test.go`、`cmd/lct/contract_test.go`

**Interfaces:**
- Produces:
  - `batch.SpeakerOf(r *rules.Rules, doc *jsonx.Node, u units.Unit) string`（原本未匯出的 `speaker` 的 id）
  - `lct search [--key K]... [--text RE] [--speaker S] [--file GLOB]... [--origin O]... [--limit N]`：條件全部成立的單位（`--text` 對 KR 或 zh 比對 Go regexp）；預設上限 200；輸出 `[{key, field, speaker, kr, zh, origin}]`
  - `adjust` skill（`disable-model-invocation: true`，`allowed-tools`：`Bash(bin/lct search *)`、`Bash(bin/lct adjust add *)`、`Bash(bin/lct requeue --to-file *)`、`Bash(bin/lct rules lint*)`、`Bash(go build -o bin/lct ./cmd/lct)`、`Read(./**)`、`Edit(./guides/**)`、`Edit(./glossary/**)`、`Edit(./rules/**)`）：流程為 §11.1 的 1～6 步；指引與詞彙的修改要先給人看、同意後才寫；最後提醒人 commit 到 `main`（不 push `raw`、`tw`）
  - `weekly-update` skill（`disable-model-invocation: true`）：在本機依序 `bin/lct fetch --out .work/fetch` → `bin/lct snapshot …` → `bin/lct plan` → 照 `translate-pending` 的流程翻譯 → `bin/lct build`，結果只在本機 `out/`；明寫不 push `raw`、`tw`、不 `publish`
  - `TestSettingsContract` 改成比對 `.claude/settings.json` 的允許清單與一份固定清單完全相同（目前的 8 項）

- [ ] **Step 1: 寫失敗的測試**
  - `TestSearchFilters`：`--file 'StoryData/S1*' --origin claude --text '經理'` 只回傳符合三個條件的單位
  - `TestSearchSpeaker`：`--speaker honglu` 只回傳鴻璐的台詞
  - `TestSettingsContractIsExact`：在測試副本的允許清單加 `Bash(bin/lct build *)` → 失敗
  - 既有的 `TestSkillContract` 涵蓋新 skill：引用的檔案與 `bin/lct` 指令都存在、沒有模型名稱、`allowed-tools` 只用 `bin/lct`、`go build`、`Read`、`Edit`

- [ ] **Step 2: 執行測試確認失敗**

Run: `go test ./cmd/lct/ -run 'Search|Contract'`
Expected: FAIL

- [ ] **Step 3: 實作 `lct search`，撰寫兩份 SKILL.md（各 200 行以內，只寫流程，規則引用 `guides/`、`glossary/`）**

- [ ] **Step 4: 執行測試確認通過**

Run: `go vet ./... && go test ./...`
Expected: PASS

- [ ] **Step 5: Commit 並 push**

```bash
git add cmd/lct internal/batch .claude
git commit -m "feat: lct search, the adjust and weekly-update skills, exact settings contract"
git push -u origin claude/festive-wright-g2k853
```

### Task 12: Workflows（§12、#1 A5）

**Files:**
- Create: `.github/workflows/pipeline.yml`、`.github/workflows/update.yml`、`README.md`
- Modify: `.github/workflows/rebuild.yml`、`.github/workflows/ci.yml`、`cmd/lct/fetch.go`、`cmd/lct/contract_test.go`
- Test: `cmd/lct/fetch_test.go`、`cmd/lct/contract_test.go`

**Interfaces:**
- Consumes: `lct fetch`、`lct snapshot`、`lct plan`、`lct requeue`、`lct translate`、`lct build`、`lct publish` 的 JSON 輸出；`publish.Result.ReleaseNeeded`；`out/review-issue/`（`title.txt`、`01.md`…）、`out/summary.md`
- Produces:
  - `lct fetch --check`：只問狀態 API，輸出 `{token, created_at, new}`（`new` = 與 raw 的 `meta/source.json` 的 token 不同），不下載
  - `pipeline.yml`（`on: workflow_call`；inputs：`fetch`（boolean）、`force`（boolean）、`requeue`（string）、`max_usd`（string）、`dry_run`（boolean）；secrets：`ANTHROPIC_API_KEY`、`LIMBUS_STATUS_URL`，都不是必填）：單一 job，`runs-on: ubuntu-24.04`、`timeout-minutes: 350`、`concurrency: { group: lct-state, cancel-in-progress: false }`、`permissions: { contents: write, issues: write }`。步驟依序：
    1. `actions/checkout@v5`（`fetch-depth: 0`）；fetch `raw`、`tw` 與所有 tag；`git worktree add .work/raw raw` 與 `.work/tw tw`
    2. origin 沒有 raw tag 時補推（沿用現有步驟）
    3. `actions/setup-go@v6`；`go build -o bin/lct ./cmd/lct`；設定 git 身分
    4. `fetch` 為真：`bin/lct fetch --check --json`；`new` 或 `force` 為真 → `bin/lct fetch --out .work/fetch`、`bin/lct snapshot --from .work/fetch --token … --created-at …`、push `raw` 與新 tag
    5. `bin/lct plan --json`；`requeue` 不是空的 → `bin/lct requeue --file "$REQUEUE" --reason "手動執行 workflow 要求重翻"`
    6. `fetch` 為真、`new` 與 `force` 都為假、`plan` 的 `pending` 為 0、沒有套用任何重翻要求或改寫檔 → 結束（不 build、不 Release）
    7. 有可領的待譯、`dry_run` 為假、`ANTHROPIC_API_KEY` 不是空的 → `npm install -g @anthropic-ai/claude-code@2.1.316`，`bin/lct translate --max-usd "${MAX_USD}" --deadline 5h --max-turns 300 --run-timeout 90m`（`MAX_USD` = input `max_usd`，否則 `vars.LCT_MAX_USD`，否則 `30`；`LCT_MODEL` = `vars.LCT_MODEL`）；失敗只發 `::warning::`，繼續
    8. `bin/lct build --json`；`out/summary.md` 附到 `$GITHUB_STEP_SUMMARY`
    9. `bin/lct publish --json > out/publish.json`；push `tw`；`release_needed` 為真 → `gh release create`（沿用現有的 asset 檢查）
    10. `out/review-issue/01.md` 存在 → `gh issue create --title "$(cat out/review-issue/title.txt)" --body-file out/review-issue/01.md`，記下 Issue 網址；其餘 `02.md`…依檔名順序 `gh issue comment <網址> --body-file <檔>`
  - `update.yml`：`on: schedule: cron '0 4 * * 4'`、`repository_dispatch: types: [limbus_update]`、`workflow_dispatch`（inputs `force`、`requeue`、`max_usd`、`dry_run`）；job `uses: ./.github/workflows/pipeline.yml`，`with: fetch: true` 加上 inputs，明列 `secrets:` 兩個（不用 `secrets: inherit`）
  - `rebuild.yml`：`on: push: branches: [main], paths: ['rules/**', 'glossary/**', 'guides/**', 'lct.yaml']` 與 `workflow_dispatch`（inputs `requeue`、`max_usd`、`dry_run`）；`with: fetch: false`
  - `ci.yml`：`actions/checkout@v5`、`actions/setup-go@v6`、`runs-on: ubuntu-24.04`
  - `README.md`（中文）：專案一句話說明與授權；安裝語言包（同 release notes）；「GitHub Actions 設定」一節，以表格列出每個 secret 與 variable 的名稱、用途、必填與否、預設值、怎麼取得（`ANTHROPIC_API_KEY`：Claude Console 的 API key，費用記在 Console；`LIMBUS_STATUS_URL`：狀態 API 網址；`LCT_MODEL`：翻譯用的模型；`LCT_MAX_USD`：每次建置花費上限，預設 30；`LIMBUS_CDN_TEMPLATE`：選填，預設值同 `lct.yaml`），設定位置（Settings → Secrets and variables → Actions），Actions 的 Workflow permissions 需要讀寫；三個 workflow 何時執行、手動執行的輸入；人工介入的方式（`rules/adjustments/`、`rules/requeue/`、`rules/rewrite/` 經由 `main`）
  - 契約測試另加：workflow 中出現的每個 `secrets.X` 與 `vars.X` 都列在 `README.md` 的「GitHub Actions 設定」中
  - 契約測試：四個 workflow 都存在；所有 `bin/lct` 指令已註冊；沒有模型名稱；`pipeline.yml` 讀的 `jq` 鍵都在對應的 Go 型別（`publish.Result`、`plan.Summary`、`fetchResult`）中；Claude Code CLI 有固定版本；`update.yml` 的 cron 是 `0 4 * * 4`；兩個呼叫者都不用 `secrets: inherit`；action 版本是 `checkout@v5`、`setup-go@v6`

- [ ] **Step 1: 寫失敗的測試**
  - `TestFetchCheck`：以 `file://` 的狀態檔與有 snapshot 的 raw worktree → token 相同時 `new == false`、不同時 `new == true`，兩者都沒有下載任何 zip
  - `TestWorkflowContract` 依上列各點各加一個以 `mutate` 改壞 workflow 的子測試（例如把 cron 改成 `0 5 * * 4` → 恰好一個問題；workflow 多用一個 `vars.FOO` 而 README 沒有 → 恰好一個問題）

- [ ] **Step 2: 執行測試確認失敗**

Run: `go test ./cmd/lct/ -run 'FetchCheck|WorkflowContract'`
Expected: FAIL

- [ ] **Step 3: 實作 `--check`，撰寫 workflows**

shell 步驟沿用現有 `rebuild.yml` 的寫法：先把指令輸出存到變數再交給 `jq -e`，`jq` 失敗就讓步驟失敗。

- [ ] **Step 4: 執行測試確認通過，並在本機以 scratch bare repo 演練 `pipeline.yml` 的每個 `run:` 區塊（`gh` 與 `npm` 以印出取代），`fetch` 為假與為真各一次**

Run: `go vet ./... && go test ./...`
Expected: PASS；演練時兩種情況都走到預期的步驟

- [ ] **Step 5: Commit 並 push**

```bash
git add .github cmd/lct README.md
git commit -m "ci: weekly update and rebuild share one pipeline: fetch, plan, translate, build, release, review issue"
git push -u origin claude/festive-wright-g2k853
```

### Task 13: 第二階段重播測試（§17）

**Files:**
- Create: `cmd/lct/e2e_phase2_test.go`、`testdata/phase2/v1/{kr,en,jp}/…`、`testdata/phase2/v2/{kr,en,jp}/…`
- Test: 同上

**Interfaces:**
- Consumes: Task 2～12 的指令；Plan 1 的 `e2eRepo`、`e2eCDN`、`fakeTranslate`、`lct`、`mustLct`
- Produces: `TestPhase2Replay`

fixture（全部小檔）：v1 → v2 有 `StoryData/KR_S1.json` 在第 2 行前插入一行並改一行、`KR_Bufs.json` 某條目新增欄位、`BgmLyrics/KR_L.json` 改一行、新增 `StoryData/KR_S2.json`（含一句與 S1 相同的 KR）、刪除 `StoryData/KR_S3.json`、一個 id 重新編號的條目。假的 `claude` 是測試執行檔本身（`TestHelperProcess` 模式）：以 `fakeTranslate` 反覆 `bin/lct batch next／submit`，最後印出 `{"total_cost_usd":0.01,"is_error":false,"subtype":"success","num_turns":3}`。

- [ ] **Step 1: 寫失敗的測試**，依序：
  1. snapshot v1 → `lct plan` → `lct translate --claude <假的>` → `lct build` → `lct publish`：`release_needed == true`
  2. 在 repo commit 一個人工調整（S1 不變的一句）、一個人工調整（S1 被改的那句，會過期）、一個重翻要求檔（`files: [Bufs.json]`）、一個改寫檔
  3. snapshot v2 → `lct plan`：S1 不變的句子都沒有進待譯、紀錄都在新 key；改的那句是 `kr-changed` 帶 `prev_zh`；第一個調整 `moved`、第二個 `expired`；S2 那句與 S1 相同的 KR 被重用；凍結檔那句在 `frozen_changed`；重翻要求套用一次
  4. 再跑一次 `lct plan`：狀態不變
  5. `lct translate` → `lct build`：`out/review-issue/01.md` 存在，含過期調整與凍結檔兩項；`build.json` 有 `model`、`cost_usd`
  6. `lct publish`：`release_needed == true`；不做任何變更再 build、publish → `release_needed == false`

- [ ] **Step 2: 執行測試確認失敗（尚未有 fixture 時）**

Run: `go test ./cmd/lct/ -run TestPhase2Replay`
Expected: FAIL

- [ ] **Step 3: 建立 fixture，補齊測試發現的程式問題（每個問題各自 commit，訊息寫明是哪個任務的介面）**

- [ ] **Step 4: 執行測試確認通過**

Run: `go vet ./... && go test ./...`
Expected: PASS

- [ ] **Step 5: Commit 並 push**

```bash
git add cmd/lct testdata/phase2
git commit -m "test: phase 2 replay from one snapshot to the next"
git push -u origin claude/festive-wright-g2k853
```

### Task 14（控制者）：上線

- [ ] **Step 1（使用者）:** 依 `README.md` 的「GitHub Actions 設定」，到 repo Settings → Secrets and variables → Actions：
  - secrets：`ANTHROPIC_API_KEY`（Claude Console 的 key）、`LIMBUS_STATUS_URL`
  - variables：`LCT_MODEL`（要用的模型）、`LCT_MAX_USD`（例如 `30`）
- [ ] **Step 2:** 最終全分支審查（SDD 的 final review），修正後 `git checkout main && git merge --ff-only claude/festive-wright-g2k853 && go test ./... && git push origin main`，再 `git checkout claude/festive-wright-g2k853`。這次 push 動到 `rules/`、`glossary/`，會觸發一次 `rebuild.yml`；預期：`plan` 沒有版本轉移，套用 Task 7 的詞彙修改（只改規則、不改舊譯文），build、publish，`release_needed` 依 `tw` 是否有新 commit 決定
- [ ] **Step 3（使用者）:** Actions → `update` → Run workflow，`dry_run: true`。預期：`fetch --check` 得到目前的 token；沒有新 token、沒有待譯 → 在第 6 步結束，沒有 Release、沒有 Issue
- [ ] **Step 4:** 確認 Step 2 的 run：job summary 有完整報表；`tw` 有 `.meta/plan.json`、`.meta/applied.json`；Release（若有）的 notes 是精簡版
- [ ] **Step 5:** 等第一個週四的排程（或使用者手動觸發 `update`）：確認新 token 的 snapshot、plan 的轉移數字、翻譯迴圈的花費與 `Stopped`、Release、審核 Issue

### Task 15（控制者）：#1 的內容事項

在 Task 14 之後，全部經由 `main` 的檔案讓 CI 處理（每個檔都會觸發 `rebuild.yml`；可以一次 commit 多個檔）。

- [ ] **Step 1:** 補回關鍵字名稱後的空格（使用者決定：遊戲以「關鍵字＋空格」配對顯示效果說明）。改寫檔 `rules/rewrite/<date>-keyword-space.yaml`：`files: ["**"]`、`origins: [claude, paratranz]`、`pattern` 為 `(\d+(?:級|層|回)(?:<名稱 1>|<名稱 2>|…))([^ ]|$)`（名稱取自 `names.KeywordTable`，由長到短，以腳本產生）、`replace: '$1 $2'`。2026-11-02 的資料中約 314 筆 claude、572 筆 paratranz 紀錄缺這個空格。先在本機以暫存的 tw 副本跑 `lct plan` 確認改寫筆數與此相近、全部通過驗證
- [ ] **Step 2:** 掃描並寫重翻要求檔，每個檔附具體的 `reason`（翻譯者看得到）：
  - 技能文字漏掉「本回合」限定（KR 含 `해당 턴|이번 턴|그 턴`、zh 不含 `本回合|該回合|當回合|這回合|此回合`，排除 `해당 턴 종료 시`）
  - 社群草稿修補後與草稿 95% 以上相同的 199 筆
  - 「N級 [ID]」漏了「強度」（zh 符合 `\d+級 \[[A-Za-z0-9_]+\](?! ?強度)`、KR 是強度）
  - `{n}` 外方括號與 KR 不同的 24 筆
  - 瓦爾普吉斯之夜紀念禮包名稱（KR 符合 `특정 추출」 출시 기념`）
  - 唐吉訶德對女性罪人的「~군」（4 筆）
- [ ] **Step 3:** 固定句型：在 `guides/families.md` 的鏡像迷宮段落寫明 EGO 飾品啟動條件句與事件結尾「어떻게 할까?」的固定譯法，並寫重翻要求檔處理現有的不同寫法
- [ ] **Step 4:** #1「需要人工決定」的項目：依使用者在 Issue 上的回覆，以 `adjust` skill 寫人工調整或改規則
- [ ] **Step 5:** 確認 CI 的結果（花費、Release、審核 Issue），在 #1 回報並勾選完成的項目
