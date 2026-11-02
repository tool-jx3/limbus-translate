# 第二階段：規則改進循環 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 補上 §11.2 的規則改進循環（CI 依報表的建議開規則 PR，人審核後合併），以及 `term-decision`、`review` skill、`lct glossary derive`、規則 PR 的影響評估；語言包內容與上一個 Release 相同時不發 Release；處理 Plan 3 暫緩的行為問題與 #1 的剩餘項目（A 的 `+dirty`、C 的 `[ID]` 檢查、D 類小項）。

**Architecture:** 新增 `lct suggest`（從 `tw` 的建置結果與規則檔產生有穩定 id 的建議，略過 `rules/declined.yaml` 與已關閉的 PR）、`lct impact`（新舊兩份規則對既有譯文的影響）、`lct adjust tidy`（更新已移動的調整 key、刪除已重翻的過期調整）、`lct glossary derive`、`lct review`、`lct proposal start／publish`（規則 PR 的分支與 PR 操作，只動 `guides/`、`glossary/`、`rules/`）。`rules-proposal.yml` 在 `update`／`rebuild` 成功後執行：`lct proposal start` 選分支、`lct suggest` 沒有新建議就結束，否則以 `anthropics/claude-code-action@v1` 跑 `/rules-review`，由 skill 改規則檔、寫 PR 說明，再呼叫 `lct proposal publish`。`lct publish` 比較這次與上一個 Release 的語言包檔案樹（不含 `Info/version.json`），相同就不發 Release。

**Tech Stack:** Go 1.26.1（沿用 `gopkg.in/yaml.v3`、`golang.org/x/image/font/sfnt`，其餘標準庫）；git CLI；`gh`（runner 內建）；GitHub Actions；`anthropics/claude-code-action@v1`（需要在 repo 安裝 Claude GitHub App）；Claude Code CLI `@anthropic-ai/claude-code@2.1.316`（沿用）。

**Spec:** `docs/superpowers/specs/2026-10-31-limbus-zh-tw-translation-design.md`（§7、§9.3、§11、§12、§13、§14、§15、§18）；待辦來源：tool-jx3/limbus-translate#1（內文 A～D 與 Plan 3 完成報告的留言）；前一份計畫：`docs/superpowers/plans/2026-11-02-phase2-automation.md`

**本計畫範圍外**：Plan 3 的 Task 15（內容修正，照常經由 `main` 的規則檔處理）；純重構（拆大檔、合併重複的 `LoadFiles`／`Lint`、YAML 回讀檢查等，列在 #1 的 Plan 3 報告中）；在 raw tag 之後才推上的舊調整重新對齊、GitHub 取消執行時與 `batch submit` 的競態、單字狀態名的防護；第一階段專用的 bootstrap 小項。

## Global Constraints

- 單位 key、`kr_sha`、輸出 JSON 格式、`origin` 列舉（`paratranz`、`claude`、`human`、`copy`、`fallback`）、待譯原因（`new`、`kr-changed`、`adjustment-expired`、`paratranz-repair`、`requeue`、`structural-fail`）沿用，不改；`.meta/plan.json`、`.meta/build.json` 只加欄位、不改既有欄位的型別
- 規則改了不自動改舊譯文：只有 `main` 上的重翻要求檔與改寫檔會改動既有紀錄，每個檔只套用一次
- `raw`、`tw` 分支與 Release 只由 CI 寫入；CI 對 `main` 只能開 PR（`rules-proposal.yml`），不能直接 push；人經由 `main` 介入
- **Release 只在語言包內容改變時建立（使用者決定）**：比較這次的 `tw` commit 與最新一個 `zh-TW-*` tag 的 `LimbusCompany_Data/` 檔案樹（git blob），不含 `LimbusCompany_Data/Lang/<lang_dir>/Info/version.json`；相同就不建立 Release，`tw` 照常 commit 與 push。Release tag 仍是 `zh-TW-YYYYMMDD.N`
- 規則 PR：分支 `rules-proposal/<YYYY-MM-DD>`、label `rules-proposal`、標題 `規則建議：YYYY-MM-DD`、base `main`；同一時間最多一個開著的規則 PR，有的話新建議加進該 PR；`rules-proposal.yml` 以 `anthropics/claude-code-action@v1`、`prompt: "/rules-review"` 執行，Claude 的認證用 `secrets.ANTHROPIC_API_KEY`，模型用 `vars.LCT_MODEL`
- 建議 id 穩定：同一件事每次產生相同的 id；人不採用的記在 `rules/declined.yaml`；關閉而未合併的規則 PR 中的 id 也視為不採用；兩者都不再提出
- PR 說明中每個建議以 `<!-- suggestion:<id> -->` 標示，`lct proposal` 以此讀出 PR 含有哪些建議
- `rules-proposal.yml` 的權限：`contents: write`、`pull-requests: write`、`issues: write`、`id-token: write`；`concurrency: { group: rules-proposal, cancel-in-progress: false }`
- action 版本：`actions/checkout@v5`、`actions/setup-go@v6`、`anthropics/claude-code-action@v1`；`runs-on: ubuntu-24.04`
- skill、`prompts/`、workflow 不寫模型名稱；模型由 `--model` 或 `LCT_MODEL` 給
- 每個 skill 設 `disable-model-invocation: true`、SKILL.md 200 行以內、只寫流程（規則引用 `guides/`、`glossary/`）；`allowed-tools` 只用 `bin/lct …`、`go build -o bin/lct ./cmd/lct`、`Read`、`Edit`
- 每個 CLI 指令支援 `--json`；exit code 0 成功、1 有發現或使用者錯誤、2 系統錯誤（`lct suggest` 有建議不算「發現」，exit 0）
- commit 訊息結尾：
  ```
  Co-Authored-By: <your session's attribution line>
  Claude-Session: https://claude.ai/code/session_015XrjRvHe6FfxjemSS1js35
  ```
- 開發分支 `claude/festive-wright-g2k853`（與 `main` 同在 `7bad47f`）；每個任務結束 `git push -u origin claude/festive-wright-g2k853`；`main` 只在 Task 16 以 fast-forward 更新

## Review Focus

1. **規則 PR 一再提出同一個建議**（人已在 `rules/declined.yaml` 拒絕、關掉了 PR，或已合併採用）：不能再出現 — Task 3 `TestSuggestSkipsDeclinedAndResolved`、Task 7 `TestProposalStartCollectsClosedPRs`
2. **語言包沒變的重建**（例如只改 `guides/` 的措辭、沒有任何譯文變動）：不發 Release；但上次 Release 失敗、或語言包真的變了，下一次一定要發 — Task 1 `TestPublishSkipsAnIdenticalPack`、`TestPublishReleasesAfterAFailedRelease`
3. **`update` 與 `rebuild` 接連完成**，`rules-proposal.yml` 跑兩次：第二次要更新同一個 PR，不能開第二個 — Task 7 `TestProposalStartReusesTheOpenPR`、`TestProposalPublishUpdatesTheOpenPR`；Task 8 的 concurrency 群組
4. **規則 PR 的步驟動到不該動的地方**（push 到 `main`、改到 `tw`、`raw`、程式碼或 workflow）：一律拒絕 — Task 7 `TestProposalPublishRefusesOtherBranches`、`TestProposalPublishRefusesFilesOutsideRules`
5. **影響評估在全量資料上跑太久或數字誤導**（27 萬筆紀錄、只該算「因這次規則變更才不符合」的單位）— Task 5 `TestImpactCountsOnlyNewViolations`，並在 Step 4 以實際的 `tw` 量測時間（目標 60 秒內）

## File Structure

| 檔案 | 責任 | 任務 |
|---|---|---|
| `internal/publish/publish.go`、`internal/gitx/tree.go` | 與上一個 Release 比較語言包；`lct publish --check` | 1 |
| `.github/workflows/pipeline.yml`、`README.md` | 閒置判斷、Release 略過的說明 | 2 |
| `internal/suggest/suggest.go`、`internal/suggest/declined.go`、`cmd/lct/suggest.go` | 建議與 id、`rules/declined.yaml` | 3 |
| `internal/adjust/tidy.go`、`cmd/lct/adjust.go` | `lct adjust tidy` | 4 |
| `internal/impact/impact.go`、`cmd/lct/impact.go`、`.github/workflows/ci.yml` | 規則變更的影響評估 | 5 |
| `internal/glossary/derive.go`、`cmd/lct/glossary.go` | `lct glossary derive` | 6 |
| `internal/proposal/proposal.go`、`cmd/lct/proposal.go` | 規則 PR 的分支、commit、push、PR | 7 |
| `.claude/skills/rules-review/`、`.github/workflows/rules-proposal.yml` | `rules-review` skill 與 workflow | 8 |
| `cmd/lct/review.go`、`.claude/skills/review/`、`.claude/skills/term-decision/` | `lct review` 與兩個 skill | 9 |
| `internal/build/report.go`、`internal/build/issue.go`、`internal/publish/publish.go` | §14.3 未對齊的檔、過期調整的新 key、來源比例 | 10 |
| `internal/plan/plan.go`、`internal/plan/transition.go` | Plan 3 暫緩的 plan 問題 | 11 |
| `cmd/lct/translate.go`、`cmd/lct/search.go`、`internal/validate/validate.go`… | Plan 3 暫緩的 CLI 問題、#1 A 與 C | 12 |
| 多處 | #1 D 類小項 | 13、14 |
| `cmd/lct/e2e_rules_test.go` | 規則循環的重播測試 | 15 |

---

### Task 1: 語言包沒變就不發 Release（使用者決定）

**Files:**
- Create: `internal/gitx/tree.go`
- Modify: `internal/publish/publish.go`、`cmd/lct/publish.go`、spec §13 與 §18（「每次建置都發 Release」改為「語言包內容與上一個 Release 不同才發」）
- Test: `internal/gitx/gitx_test.go`、`internal/publish/publish_test.go`、`cmd/lct/publish_test.go`

**Interfaces:**
- Produces:
  - `gitx.TreeBlobs(dir, rev, path string) (map[string]string, error)`：`git ls-tree -r <rev> -- <path>` 的「路徑 → blob id」；`rev` 不存在時回傳錯誤
  - `publish.LastRelease(twRoot string) (tag string, err error)`：所有 `zh-TW-*` tag 中最新的一個（先比日期，再比 `.N` 的數字；`.10` 在 `.9` 之後）；沒有時回傳 `""`
  - `publish.SamePack(twRoot, a, b, langDir string) (bool, error)`：兩個 rev 的 `LimbusCompany_Data/` 檔案樹相同（不含 `LimbusCompany_Data/Lang/<langDir>/Info/version.json`）
  - `publish.Result` 加 `SameAs string` `json:"same_as,omitempty"`：語言包與這個 tag 相同時填入
  - `ReleaseNeeded` 改為：`TWCommit` 上沒有 `zh-TW-` tag，而且（沒有任何舊 Release，或與 `LastRelease` 的語言包不同）
  - `lct publish --check`：不 commit、不寫 notes，回報目前 `tw` HEAD 的 `{tw_commit, release_needed, same_as, tag}`

- [ ] **Step 1: 寫失敗的測試**
  - `TestTreeBlobs`：暫存 repo commit `a/x.json`、`a/b/y.json`、`c.txt` → `TreeBlobs(repo, "HEAD", "a")` 恰好兩個鍵，值等於 `git rev-parse HEAD:a/x.json` 等
  - `TestLastReleaseOrdering`：tag `zh-TW-20261029.1`、`.2`、`.10`、`zh-TW-20261105.1`、`raw-20261112` → `zh-TW-20261105.1`；只有前三個 → `zh-TW-20261029.10`
  - `TestPublishSkipsAnIdenticalPack`：建置、publish、打 `zh-TW-20261029.1`；只改 rules commit（`version.json` 不同）再建置、publish → `Changed == true`、`ReleaseNeeded == false`、`SameAs == "zh-TW-20261029.1"`
  - `TestPublishReleasesWhenThePackChanged`：改一筆紀錄的 `zh` 再建置、publish → `ReleaseNeeded == true`、`SameAs == ""`
  - `TestPublishReleasesAfterAFailedRelease`：語言包變了、publish 後沒打 tag（Release 失敗）；再跑一次 publish → `Changed == false`、`ReleaseNeeded == true`
  - `TestPublishFirstRelease`：沒有任何 `zh-TW-` tag → `ReleaseNeeded == true`
  - `cmd/lct`：`TestPublishCheckCommitsNothing`：`lct publish --check --json` 的輸出含 `release_needed`、`same_as`，`tw` 的 HEAD 與狀態不變

- [ ] **Step 2: 執行測試確認失敗**

Run: `go test ./internal/gitx/ ./internal/publish/ ./cmd/lct/ -run 'TreeBlobs|LastRelease|Publish'`
Expected: FAIL（未定義 `TreeBlobs`、`LastRelease`、`SameAs`、`--check`）

- [ ] **Step 3: 實作**（release notes 不變；`SameAs` 只影響 `ReleaseNeeded`）

- [ ] **Step 4: 執行測試確認通過**

Run: `go vet ./... && go test ./...`
Expected: PASS

- [ ] **Step 5: Commit 並 push**

```bash
git add internal/gitx internal/publish cmd/lct docs/superpowers/specs
git commit -m "feat(publish): release only when the pack differs from the last Release"
git push -u origin claude/festive-wright-g2k853
```

### Task 2: pipeline 的閒置判斷與 Release 略過

**Files:**
- Modify: `.github/workflows/pipeline.yml`、`cmd/lct/contract_test.go`、`README.md`
- Test: `cmd/lct/contract_test.go`

**Interfaces:**
- Consumes: Task 1 `lct publish --check --json`（`release_needed`、`same_as`）、`publish.Result.SameAs`
- Produces:
  - 閒置條件（`pipelineIdleTerms`）另加兩項：`publish --check` 的 `release_needed` 為 `false`；`tw` HEAD 的 commit 標題不是以 `translate: keep the progress of` 開頭（保存進度後 build 失敗的情況）
  - publish 之後 `release_needed` 為假且 `same_as` 不是空的 → job summary 加一行「語言包與 `<same_as>` 相同，這次不建立 Release」
  - 建立 Release 的步驟：資產數不是 2 時印出 `::error::publish.json has <n> assets, want 2` 再失敗（#1 D：原本 `jq -e` 直接以 4 結束、沒有訊息）

- [ ] **Step 1: 寫失敗的測試**：`TestWorkflowContract` 加子測試：拿掉任一新閒置條件 → 恰好一個問題；`same_as` 必須在 `lctJSONInputs` 對應的 `publish.Result` 中；資產數檢查沒有 `::error::` → 一個問題

- [ ] **Step 2: 執行測試確認失敗**

Run: `go test ./cmd/lct/ -run WorkflowContract`
Expected: FAIL

- [ ] **Step 3: 修改 workflow 與 README**（README 的「GitHub Actions 設定」與「什麼時候會有新 Release」各加一句：語言包沒變就不發 Release）

- [ ] **Step 4: 執行測試，並在本機以 scratch bare repo 演練改到的 `run:` 區塊（`gh` 以印出取代）**：語言包相同、相同但上次是保存進度的 commit、上次 Release 失敗三種情況

Run: `go vet ./... && go test ./...`
Expected: PASS；三種情況分別：閒置、不閒置、不閒置並建立 Release

- [ ] **Step 5: Commit 並 push**

```bash
git add .github cmd/lct README.md
git commit -m "ci: idle only when nothing is left to release; skip the Release of an unchanged pack"
git push -u origin claude/festive-wright-g2k853
```

### Task 3: 建議與 `rules/declined.yaml`（§11.2）

**Files:**
- Create: `internal/suggest/suggest.go`、`internal/suggest/declined.go`、`cmd/lct/suggest.go`、`rules/declined.yaml`（內容 `[]`）
- Modify: `cmd/lct/rules.go`（lint 加入 `declined.yaml`）、`cmd/lct/main.go`、spec §11.2（id 的格式）
- Test: `internal/suggest/suggest_test.go`、`cmd/lct/suggest_test.go`

**Interfaces:**
- Consumes: `tw` 的 `.meta/build.json`（`build.Info.Report`：`Inconsistent`、`Notes`）、`.meta/plan.json`（`plan.Summary`：`NewSpeakers`、`FieldChanges`、`NewFilesDefaultFamily`、`Adjustments`），只用 `raw_tag` 與目前 raw tag 相同的那份；`rules.Load`；`adjust.Load`
- Produces:
  - `suggest.Suggestion{ID, Kind, Title string; Evidence []suggest.Evidence}`（JSON `id`、`kind`、`title`、`evidence`）；`Evidence{Key, KR, ZH string; Count int}`（`key`、`kr`、`zh`、`count`，空的省略）
  - 種類與 id（`sha8` = KR 或文字的 sha256 前 8 個小寫 hex）：
    - `inconsistent/<family>/<field>/<sha8(KR)>`：`Report.Inconsistent` 的每一組
    - `speaker/<model>`：`NewSpeakers`；`rules.CharacterByModel` 已認得就不提
    - `field/<add|remove>/<group 或 ->/<field>`：`FieldChanges`
    - `family/<file>`：`NewFilesDefaultFamily`；`rules.Family(file)` 已不是預設族群就不提
    - `note/<key>/<sha8(note text)>`：`Report.Notes`
    - `term/<kr>`：`glossary/terms.yaml` 中 `status: candidate` 的條目
    - `adjustment-moved/<舊 key>`：`Adjustments.Moved`；調整檔已改成新 key 就不提
    - `adjustment-expired/<key>`：`Adjustments.Expired`；調整檔已沒有這個 key 就不提
  - §11.2 的「尚未歸納進規則的人工調整」：`adjust` skill 在 §11.1 第 4 步就把調整歸納進規則（`rule` 欄位），這裡只處理 key 移動與過期兩種
  - `suggest.Run(in suggest.Inputs) ([]suggest.Suggestion, error)`；結果依 id 排序
  - `rules/declined.yaml`：清單，每筆 `{id, reason, date}`；`suggest.LoadDeclined(root string) ([]suggest.Declined, error)`；lint：id 的種類是上列之一、`reason` 非空、`date` 是 `YYYY-MM-DD`、id 不重複
  - `lct suggest [--exclude-file F]... [--json]`：略過 `declined.yaml` 與每個 `--exclude-file`（一行一個 id）中的 id；輸出 `{suggestions: [...], excluded: N}`

- [ ] **Step 1: 寫失敗的測試**
  - `TestSuggestKindsAndIDs`：fixture 的 build.json 與 plan.json 各含一項 → 八種各一個建議，id 與上列格式完全相同
  - `TestSuggestIDsAreStable`：同樣的輸入跑兩次 → id 與順序相同；Inconsistent 的 `ZH` 順序不同也一樣
  - `TestSuggestSkipsDeclinedAndResolved`：`declined.yaml` 有其中一個 id、`--exclude-file` 有另一個、`guides/characters/` 已有該 model、`routing.yaml` 已把該檔歸入其他族群、調整檔已改成新 key → 這五個都不出現，`excluded == 2`
  - `TestSuggestIgnoresAnotherRawTag`：plan.json 的 `raw_tag` 不是目前的 → 沒有來自 plan 的建議
  - `TestLintDeclined`：未知種類、空 `reason`、錯的日期、重複 id → 各一個錯誤

- [ ] **Step 2: 執行測試確認失敗**

Run: `go test ./internal/suggest/ ./cmd/lct/ -run 'Suggest|LintDeclined'`
Expected: FAIL

- [ ] **Step 3: 實作**（YAML 讀取沿用 `internal/rulefile`）

- [ ] **Step 4: 執行測試確認通過**

Run: `go vet ./... && go test ./... && go build -o bin/lct ./cmd/lct && bin/lct rules lint`
Expected: PASS；lint 無錯

- [ ] **Step 5: Commit 並 push**

```bash
git add internal/suggest cmd/lct rules/declined.yaml docs/superpowers/specs
git commit -m "feat(suggest): suggestions with stable ids, and the declined list"
git push -u origin claude/festive-wright-g2k853
```

### Task 4: `lct adjust tidy`

**Files:**
- Create: `internal/adjust/tidy.go`
- Modify: `cmd/lct/adjust.go`、`internal/adjust/adjust.go`（`Add` 附加後重新解析整個檔）、`internal/rules/lint.go` 或對應的 lint（`rules/adjustments/`、`rules/requeue/`、`rules/rewrite/` 允許 `README.md`）
- Test: `internal/adjust/tidy_test.go`、`internal/adjust/adjust_test.go`、`cmd/lct/adjust_test.go`

**Interfaces:**
- Consumes: `.meta/plan.json` 的 `Adjustments.Moved`（`{from, to}`）與 `Adjustments.Expired`
- Produces:
  - `adjust.Move{From, To string}`
  - `adjust.Tidy(root string, moves []adjust.Move, expired []string, settled func(key string) bool) (changed []string, err error)`：把 `From` 的 key 改成 `To`（其他欄位不動）；`expired` 中 `settled(key)` 為真（有 `kr_sha` 等於目前 KR 的非 `human` 紀錄、且不在待譯清單）的調整刪掉；檔案刪到沒有條目時刪檔；回傳改到的檔（`rules/adjustments/…`，排序）
  - `lct adjust tidy [--json]`：讀目前的 plan.json 與 `tw` 狀態，輸出 `{changed: [...], moved: N, removed: N}`
  - `adjust.Add`：附加後重新解析整個檔，條目數不是原本加一就不寫（手寫的 flow sequence 或 `...` 結尾）

- [ ] **Step 1: 寫失敗的測試**
  - `TestTidyUpdatesMovedKeys`：一筆移動 → 檔中只有那筆的 `key` 變了，其他位元組相同
  - `TestTidyRemovesRetranslatedExpired`：過期的 key 已重翻（`settled` 為真）→ 刪除；檔只剩它時整個檔刪除
  - `TestTidyKeepsPendingExpired`：`settled` 為假 → 不動
  - `TestAddRefusesAFileItCannotAppendTo`：調整檔是 flow sequence `[{…}]` → `Add` 錯誤，檔案不變
  - `TestLintAllowsAReadme`：三個資料夾各放 `README.md` → lint 無錯；放 `x.yml` 仍有錯

- [ ] **Step 2: 執行測試確認失敗**

Run: `go test ./internal/adjust/ ./internal/rules/ ./cmd/lct/ -run 'Tidy|Append|Readme'`
Expected: FAIL

- [ ] **Step 3: 實作**

- [ ] **Step 4: 執行測試確認通過**

Run: `go vet ./... && go test ./...`
Expected: PASS

- [ ] **Step 5: Commit 並 push**

```bash
git add internal/adjust internal/rules cmd/lct
git commit -m "feat(adjust): tidy moved and retranslated adjustments; safer append; README in rule folders"
git push -u origin claude/festive-wright-g2k853
```

### Task 5: 規則變更的影響評估（§11.2、§12 `ci.yml`）

**Files:**
- Create: `internal/impact/impact.go`、`cmd/lct/impact.go`
- Modify: `.github/workflows/ci.yml`、`cmd/lct/contract_test.go`、`cmd/lct/main.go`
- Test: `internal/impact/impact_test.go`、`cmd/lct/impact_test.go`、`cmd/lct/contract_test.go`

**Interfaces:**
- Consumes: `batch.CheckNew(r, file, kr, zh)`（新譯文的嚴格度）；`gitx.Archive`；`rules.Load`
- Produces:
  - `impact.Inputs{Old, New *rules.Rules; Records map[string]store.Record; KR *pack.Pack; Frozen []string}`
  - `impact.Run(in impact.Inputs) (*impact.Result, error)`：對 `origin` 為 `paratranz`、`claude`、`human`，且 `kr_sha` 等於目前 KR 的紀錄，比較兩份規則下的硬性問題；只計「新規則有、舊規則沒有」的問題代碼；凍結檔不算
  - `impact.Result{Checked, Units int; ByCode map[string]int; ByOrigin map[string]int; Samples map[string][]string}`（JSON `checked`、`units`、`by_code`、`by_origin`、`samples`；`Samples` 每個代碼前 10 個 key，排序）
  - `lct impact --base REV [--markdown FILE] [--json]`：舊規則 = `REV` 的 `guides/`、`glossary/`、`rules/`（以 `gitx.Archive` 匯出到暫存資料夾）；新規則 = 工作目錄
  - `ci.yml`：`pull_request` 且與 base 相比動到 `guides/`、`glossary/`、`rules/` 時，fetch `raw`、`tw`，建 worktree，`bin/lct impact --base "origin/$BASE_REF" --markdown out/impact.md`，附到 `$GITHUB_STEP_SUMMARY`；`BASE_REF` 經由 `env:` 傳入；權限維持 `contents: read`

- [ ] **Step 1: 寫失敗的測試**
  - `TestImpactCountsOnlyNewViolations`：新詞彙條目禁用「神父」→ 含「神父」的 `claude` 與 `paratranz` 紀錄各一筆計入，`ByOrigin` 兩者各 1；舊規則就已違反的另一條規則不計
  - `TestImpactIgnoresStaleAndFrozen`：`kr_sha` 不符的紀錄、`BgmLyrics/` 的紀錄、`copy` 與 `fallback` 紀錄 → 不計，`Checked` 不含它們
  - `TestImpactCommand`：暫存 repo 的 base commit 與修改過的工作目錄 → `--json` 的 `units` 與預期相同；`--markdown` 的檔案含 `units` 數與樣本 key
  - `TestWorkflowContract`：`ci.yml` 的影響評估步驟存在、只在 `pull_request` 執行、`base_ref` 不直接寫進 `run:`

- [ ] **Step 2: 執行測試確認失敗**

Run: `go test ./internal/impact/ ./cmd/lct/ -run 'Impact|WorkflowContract'`
Expected: FAIL

- [ ] **Step 3: 實作**

- [ ] **Step 4: 執行測試確認通過，並以實際的 `.work/tw` 量測一次**（新增一個禁用譯法的詞彙條目後跑 `bin/lct impact --base HEAD --json`，記下時間與 `units`）

Run: `go vet ./... && go test ./...`
Expected: PASS；全量 60 秒內

- [ ] **Step 5: Commit 並 push**

```bash
git add internal/impact cmd/lct .github/workflows/ci.yml
git commit -m "feat(impact): how many existing translations a rule change would break; shown on rule PRs"
git push -u origin claude/festive-wright-g2k853
```

### Task 6: `lct glossary derive` 與詞彙 lint

**Files:**
- Create: `internal/glossary/derive.go`、`cmd/lct/glossary.go`
- Modify: `internal/rules/lint.go`、`glossary/README.md`、`cmd/lct/main.go`
- Test: `internal/glossary/derive_test.go`、`internal/rules/rules_test.go`、`cmd/lct/glossary_test.go`

**Interfaces:**
- Consumes: Plan 3 `batch` 的關鍵字名稱疊加（`keywordTable`／`KeywordNames` 用的紀錄疊加函式）；`names` 的說話者名稱表（`ScenarioModelCodes*`）；Task 3 `suggest.LoadDeclined`
- Produces:
  - `glossary.Candidate{KR, ZH, Source string}`（JSON `kr`、`zh`、`source`；`Source` 為 `keyword` 或 `speaker`）
  - `glossary.Derive(in glossary.Inputs) []glossary.Candidate`：本專案目前譯文中的關鍵字名稱與說話者名稱，KR 不在任何詞彙條目、`term/<kr>` 不在 `declined.yaml` 的；ZH 仍含韓文的不列；依 KR 排序
  - `lct glossary derive [--source keyword|speaker] [--json]`：只輸出，不寫檔
  - lint：「同一個 `kr` 至多一個條目沒有 `files`」只計 `status: approved` 的條目；`glossary/README.md` 註明 `files` 重疊的條目會同時生效

- [ ] **Step 1: 寫失敗的測試**
  - `TestGlossaryDeriveKeywordsAndSpeakers`：fixture 有一個關鍵字與一個說話者的譯文 → 兩個候選，`Source` 正確
  - `TestGlossaryDeriveSkipsKnownAndDeclined`：KR 已在詞彙表、`term/<kr>` 已拒絕、ZH 仍含韓文 → 都不列
  - `TestLintGlossaryCountsApprovedOnly`：同一個 `kr` 一個 approved 無 `files`、一個 candidate 無 `files` → lint 通過；兩個 approved 無 `files` → 錯誤

- [ ] **Step 2: 執行測試確認失敗**

Run: `go test ./internal/glossary/ ./internal/rules/ ./cmd/lct/ -run 'GlossaryDerive|LintGlossary'`
Expected: FAIL

- [ ] **Step 3: 實作**

- [ ] **Step 4: 執行測試確認通過**

Run: `go vet ./... && go test ./... && go build -o bin/lct ./cmd/lct && bin/lct rules lint`
Expected: PASS

- [ ] **Step 5: Commit 並 push**

```bash
git add internal/glossary internal/rules cmd/lct glossary/README.md
git commit -m "feat(glossary): derive term candidates from the pack's own names; lint counts approved entries"
git push -u origin claude/festive-wright-g2k853
```

### Task 7: 規則 PR 的分支與 PR 操作（`lct proposal`）

**Files:**
- Create: `internal/proposal/proposal.go`、`cmd/lct/proposal.go`
- Modify: `cmd/lct/main.go`
- Test: `internal/proposal/proposal_test.go`、`cmd/lct/proposal_test.go`

**Interfaces:**
- Produces:
  - `proposal.Options{Repo, Date, Base string; Git, GH string /*可執行檔，預設 "git"、"gh"*/}`（`Base` 預設 `main`）
  - `proposal.Start(o proposal.Options) (*proposal.Started, error)`：
    - `gh pr list --label rules-proposal --state open --json number,url,headRefName,body`：有開著的 → `git fetch origin <head>` 並 checkout 該分支；沒有 → `git checkout -B rules-proposal/<Date> origin/<Base>`
    - `gh pr list --label rules-proposal --state closed --json number,body,mergedAt --limit 200`：`mergedAt` 為空的 PR 中的 id 視為不採用
    - 寫 `work/proposal/exclude.txt`：開著的 PR 已含的 id ＋ 關閉未合併的 PR 中的 id，一行一個、排序
    - `Started{Branch string; PR int; URL string; Existing, Declined []string}`（JSON `branch`、`pr`、`url`、`existing`、`declined`）
  - `proposal.Publish(o proposal.Options, bodyFile string) (*proposal.Published, error)`：
    - 目前分支必須以 `rules-proposal/` 開頭，否則錯誤
    - `git status --porcelain` 中的路徑只能在 `guides/`、`glossary/`、`rules/` 下，否則錯誤並列出
    - 說明檔至少含一個 `<!-- suggestion:<id> -->`，否則錯誤
    - 有變更 → `git add guides glossary rules`、commit `rules: proposal <Date>`、`git push origin <branch>`（不加 `--force`）
    - 這個分支已有開著的 PR → `gh pr edit <n> --body-file <bodyFile>`；否則 `gh label create rules-proposal --force`，再 `gh pr create --base <Base> --head <branch> --title "規則建議：<Date>" --label rules-proposal --body-file <bodyFile>`
    - `Published{Branch, URL, Commit string; Created bool}`
  - `lct proposal start --date YYYY-MM-DD [--json]`、`lct proposal publish --body-file F [--date YYYY-MM-DD] [--json]`；`gh` 的認證用環境中的 `GH_TOKEN` 或 `GITHUB_TOKEN`，兩者都沒有時錯誤訊息說明

- [ ] **Step 1: 寫失敗的測試**（暫存 bare repo 當 origin；假的 `gh` 是 shell script，依環境變數回傳 JSON，並把收到的參數記到檔案）
  - `TestProposalStartNewBranch`：沒有開著的 PR → 分支 `rules-proposal/2026-11-05`，從 `origin/main` 建立
  - `TestProposalStartReusesTheOpenPR`：有開著的 PR（head `rules-proposal/2026-10-29`，body 含兩個 id）→ checkout 該分支，`Existing` 是那兩個 id，`exclude.txt` 含它們
  - `TestProposalStartCollectsClosedPRs`：一個已合併、一個關閉未合併的 PR → 只有未合併那個的 id 進 `Declined` 與 `exclude.txt`
  - `TestProposalPublishCreatesPR`：改 `glossary/terms.yaml` → commit、push 到 origin 的該分支，假 `gh` 收到 `pr create … --title 規則建議：2026-11-05 --label rules-proposal`
  - `TestProposalPublishUpdatesTheOpenPR`：分支已有開著的 PR → 假 `gh` 收到 `pr edit <n> --body-file`，沒有 `pr create`
  - `TestProposalPublishRefusesOtherBranches`：在 `main` 或 `tw` 上 → 錯誤，沒有 push
  - `TestProposalPublishRefusesFilesOutsideRules`：同時改了 `internal/x.go` 或 `.github/workflows/ci.yml` → 錯誤列出該檔，沒有 commit
  - `TestProposalPublishNeedsSuggestionMarkers`：說明檔沒有標記 → 錯誤

- [ ] **Step 2: 執行測試確認失敗**

Run: `go test ./internal/proposal/ ./cmd/lct/ -run Proposal`
Expected: FAIL

- [ ] **Step 3: 實作**

- [ ] **Step 4: 執行測試確認通過**

Run: `go vet ./... && go test ./...`
Expected: PASS

- [ ] **Step 5: Commit 並 push**

```bash
git add internal/proposal cmd/lct
git commit -m "feat(proposal): choose the rules PR branch, then commit only rule files and open or update the PR"
git push -u origin claude/festive-wright-g2k853
```

### Task 8: `rules-review` skill 與 `rules-proposal.yml`（§11.2、§12）

**Files:**
- Create: `.claude/skills/rules-review/SKILL.md`、`.github/workflows/rules-proposal.yml`
- Modify: `cmd/lct/contract_test.go`（`workflowActions` 加 `anthropics/claude-code-action@v1`）、`README.md`、spec §12（`rules-proposal.yml` 的觸發與步驟）
- Test: `cmd/lct/contract_test.go`

**Interfaces:**
- Consumes: Task 3 `lct suggest`、Task 4 `lct adjust tidy`、Task 5 `lct impact`、Task 6 `lct glossary derive`、Task 7 `lct proposal start／publish`；既有 `lct search`、`lct requeue --to-file`、`lct rules lint`
- Produces:
  - `rules-review` skill，`allowed-tools`：`Bash(bin/lct suggest *)`、`Bash(bin/lct search *)`、`Bash(bin/lct glossary derive*)`、`Bash(bin/lct adjust tidy*)`、`Bash(bin/lct requeue --to-file *)`、`Bash(bin/lct rules lint*)`、`Bash(bin/lct impact *)`、`Bash(bin/lct proposal publish *)`、`Bash(go build -o bin/lct ./cmd/lct)`、`Read(./**)`、`Edit(./guides/**)`、`Edit(./glossary/**)`、`Edit(./rules/**)`、`Edit(./work/**)`
  - 流程（CI 與本機相同，最後一步不同）：
    1. `bin/lct suggest --json`（`work/proposal/exclude.txt` 存在時加 `--exclude-file work/proposal/exclude.txt`）
    2. 逐項處理：不一致譯法 → 定一種寫法，寫進指引或詞彙表，並以 `bin/lct requeue --to-file` 附上重翻範圍；新說話者 → `guides/characters/<id>.md`；欄位 → `rules/fields.manual.yaml`；新族群的檔 → `guides/routing.yaml`；note → 詞彙候選或指引；詞彙候選 → 附證據建議採用與否；`adjustment-moved`／`adjustment-expired` → `bin/lct adjust tidy`。沒有把握的項目不改、不列
    3. `bin/lct rules lint`；`bin/lct impact --base origin/main --markdown work/proposal/impact.md`
    4. 寫 `work/proposal/body.md`：每項一節，含 `<!-- suggestion:<id> -->`、建議內容、理由、證據（檔名#key、KR、現行譯文、出現次數），最後附影響評估；開頭說明：不要的項目請移到 `rules/declined.yaml` 再合併
    5. CI（`work/proposal/exclude.txt` 存在）：`bin/lct proposal publish --body-file work/proposal/body.md`；本機：把變更與說明給人看，由人 commit
  - `rules-proposal.yml`：
    - `on: workflow_run: { workflows: [update, rebuild], types: [completed] }`（job 只在 `conclusion == 'success'` 時執行）與 `workflow_dispatch`
    - job：`runs-on: ubuntu-24.04`、`timeout-minutes: 60`、`concurrency: { group: rules-proposal, cancel-in-progress: false }`、權限同 Global Constraints
    - 步驟：checkout（`fetch-depth: 0`）；fetch `raw`、`tw`，建 worktree；setup-go；build `lct`；設定 git 身分；`bin/lct proposal start --date "$(date -u +%F)" --json`（`GH_TOKEN: ${{ github.token }}`）；`bin/lct suggest --json --exclude-file work/proposal/exclude.txt`，建議數為 0 → 結束；`anthropics/claude-code-action@v1`，`with: anthropic_api_key: ${{ secrets.ANTHROPIC_API_KEY }}`、`prompt: "/rules-review"`、`claude_args: --model ${{ vars.LCT_MODEL }} --max-turns 80 --allowedTools "<與 skill 的 allowed-tools 相同>"`；最後以 `gh pr list --label rules-proposal --state open` 確認 PR 存在，印出網址到 job summary
  - 契約測試：workflow 存在；觸發、權限、concurrency、`prompt` 恰好是 `/rules-review`；`claude_args` 的 `--allowedTools` 與 skill 的 `allowed-tools` 完全相同；沒有模型名稱；沒有 `secrets: inherit`；`run:` 中沒有 `${{ }}`
  - README：安裝 Claude GitHub App；`rules-proposal` label；收到規則 PR 時怎麼處理（直接改 PR、把不要的 id 移到 `rules/declined.yaml`、合併或關閉）

- [ ] **Step 1: 寫失敗的測試**：`TestSkillContract` 涵蓋 `rules-review`（引用的檔案與指令存在、工具清單完全相同）；`TestWorkflowContract` 加 `rules-proposal.yml` 的各項，每項一個以 `mutate` 改壞的子測試（例如 `--allowedTools` 少一個工具 → 恰好一個問題）

- [ ] **Step 2: 執行測試確認失敗**

Run: `go test ./cmd/lct/ -run 'SkillContract|WorkflowContract'`
Expected: FAIL

- [ ] **Step 3: 撰寫 SKILL.md 與 workflow**

- [ ] **Step 4: 執行測試確認通過，並在本機以 scratch bare repo 演練 `rules-proposal.yml` 中 action 之前與之後的 `run:` 區塊（`gh` 以假 script 取代）**：沒有建議（結束）、有建議且沒有開著的 PR、有開著的 PR 三種情況

Run: `go vet ./... && go test ./...`
Expected: PASS

- [ ] **Step 5: Commit 並 push**

```bash
git add .claude .github cmd/lct README.md docs/superpowers/specs
git commit -m "ci: rules-proposal opens or updates the rules PR from new suggestions"
git push -u origin claude/festive-wright-g2k853
```

### Task 9: `lct review`、`review` 與 `term-decision` skill（§9.3）

**Files:**
- Create: `cmd/lct/review.go`、`.claude/skills/review/SKILL.md`、`.claude/skills/term-decision/SKILL.md`
- Modify: `internal/build/issue.go`（匯出項目清單）、`cmd/lct/contract_test.go`、`cmd/lct/main.go`
- Test: `cmd/lct/review_test.go`、`internal/build/issue_test.go`、`cmd/lct/contract_test.go`

**Interfaces:**
- Consumes: Plan 3 `build.ReviewIssue` 內部的項目分節（`reviewSections`）
- Produces:
  - `build.ReviewSection{Kind, Title, Hint string; Items []string}`（JSON `kind`、`title`、`hint`、`items`）；`build.ReviewItems(info build.Info, sum *plan.Summary, pending []store.PendingItem) []build.ReviewSection`；`ReviewIssue` 改用它
  - `lct review [--json]`：目前 `tw` 狀態的 `{title, sections}`，內容與這次建置的審核 Issue 相同
  - `review` skill（`allowed-tools`：`Bash(bin/lct review*)`、`Bash(bin/lct search *)`、`Bash(bin/lct adjust add *)`、`Bash(bin/lct adjust tidy*)`、`Bash(bin/lct requeue --to-file *)`、`Bash(bin/lct fields learn*)`、`Bash(bin/lct rules lint*)`、`Bash(go build -o bin/lct ./cmd/lct)`、`Read(./**)`、`Edit(./guides/**)`、`Edit(./glossary/**)`、`Edit(./rules/**)`、`Edit(./lct.yaml)`）：依 `lct review` 的各節帶人處理（needs_review → 調整或重翻；過期調整 → 確認新譯文後 `adjust tidy`；凍結檔 → 決定是否解除凍結；新說話者、欄位、族群 → 對應的規則檔；翻譯迴圈錯誤 → 說明原因）；最後提醒 commit 到 `main` 並勾選 Issue
  - `term-decision` skill（`allowed-tools`：`Bash(bin/lct suggest *)`、`Bash(bin/lct glossary derive*)`、`Bash(bin/lct search *)`、`Bash(bin/lct requeue --to-file *)`、`Bash(bin/lct rules lint*)`、`Bash(go build -o bin/lct ./cmd/lct)`、`Read(./**)`、`Edit(./glossary/**)`、`Edit(./rules/**)`）：候選來自 `lct suggest` 的 `term/…` 與 `lct glossary derive`；每個詞由人決定採用（寫成 `status: approved`）、改譯法、不採用（寫進 `rules/declined.yaml`）或略過；採用的譯法與現行譯文不同時，以 `lct search` 找出並提議重翻要求檔

- [ ] **Step 1: 寫失敗的測試**
  - `TestReviewItemsMatchTheIssue`：同一組輸入，`ReviewItems` 的每一項都出現在 `ReviewIssue` 的內文，順序相同
  - `TestReviewCommand`：`lct review --json` 與 `out/review-issue/01.md` 列出的項目一致；沒有需要處理的項目時 `sections` 是 `[]`
  - `TestSkillContract` 涵蓋兩個新 skill，工具清單完全相同

- [ ] **Step 2: 執行測試確認失敗**

Run: `go test ./internal/build/ ./cmd/lct/ -run 'ReviewItems|ReviewCommand|SkillContract'`
Expected: FAIL

- [ ] **Step 3: 實作 `lct review`，撰寫兩份 SKILL.md**

- [ ] **Step 4: 執行測試確認通過**

Run: `go vet ./... && go test ./...`
Expected: PASS

- [ ] **Step 5: Commit 並 push**

```bash
git add internal/build cmd/lct .claude
git commit -m "feat: lct review, the review and term-decision skills"
git push -u origin claude/festive-wright-g2k853
```

### Task 10: 報表補強（§14.3、Plan 3 暫緩）

**Files:**
- Modify: `internal/build/build.go`、`internal/build/report.go`、`internal/build/issue.go`、`internal/plan/plan.go`（過期調整對應的單位）、`internal/publish/publish.go`
- Test: `internal/build/build_test.go`、`internal/build/issue_test.go`、`internal/plan/plan_test.go`、`internal/publish/publish_test.go`

**Interfaces:**
- Produces:
  - `build.Report` 加 `Misaligned []build.FileMisaligned`（JSON `misaligned`；`{file, en, jp}`：這個檔有多少 KR 單位找不到 EN／JP 的對應；依 `en+jp` 遞減取前 20）；報表「資訊」節列出（§14.3「跨語言順序或 id 不一致的檔」）
  - `plan.AdjustmentSummary` 加 `ExpiredUnits map[string]string`（JSON `expired_units`；調整的 key → 這一版重翻的單位 key，兩者相同時不列）；審核 Issue 的「人工調整已過期」寫成「調整 `<key>`（這一版的單位：`<unit>`）」
  - `build.Report` 加 `Shown map[string]int`（JSON `shown`）：每個單位最後顯示的內容只算一次（`paratranz`、`claude`、`human`、`copy`，以及顯示英文的 `stand-in`）；release notes 的比例改用它，分母是全部單位

- [ ] **Step 1: 寫失敗的測試**
  - `TestReportListsMisalignedFiles`：fixture 中一個檔的 EN 少一行 → `Misaligned` 含該檔、`en == 1`
  - `TestReviewIssueNamesTheUnitOfAnExpiredAdjustment`：調整的 key 因插行移到別處 → Issue 項目同時有兩個 key
  - `TestNotesSharesCountEachUnitOnce`：一筆 `claude` 紀錄因破壞結構而顯示英文 → 只算在 `stand-in`；比例加總為 100.0%

- [ ] **Step 2: 執行測試確認失敗**

Run: `go test ./internal/build/ ./internal/plan/ ./internal/publish/ -run 'Misaligned|ExpiredAdjustment|SharesCount'`
Expected: FAIL

- [ ] **Step 3: 實作**

- [ ] **Step 4: 執行測試確認通過**

Run: `go vet ./... && go test ./...`
Expected: PASS

- [ ] **Step 5: Commit 並 push**

```bash
git add internal/build internal/plan internal/publish
git commit -m "feat(build): list misaligned files, name the unit of an expired adjustment, count each unit once in the notes"
git push -u origin claude/festive-wright-g2k853
```

### Task 11: `lct plan` 的暫緩問題（Plan 3 最終審查）

**Files:**
- Modify: `internal/plan/plan.go`、`internal/plan/transition.go`、`internal/rewrite/rewrite.go`
- Test: `internal/plan/plan_test.go`、`internal/plan/transition_test.go`

**Interfaces:**
- Produces:
  - 改寫檔改了某筆紀錄時，同一個 key 的待譯項目若 `PrevZH` 等於改寫前的 `zh`，也改成改寫後的 `zh`
  - `plan.DeletedUnit` 加 `Pending *store.PendingItem`（被刪除單位原本的待譯項目）；相同文字重用時，來源是帶有待譯項目（原因 `requeue` 或有 `Hint`）的刪除單位 → 不重用譯文，改把該待譯項目搬到新單位（保留 `Reason`、`Hint`、`PrevKR`、`PrevZH`）
  - 凍結檔的步驟：只有 `copy` 紀錄不算「已有譯文」；欄位規則改成要翻時照一般流程排入待譯
  - 沒有版本轉移時，`kr_sha` 與目前 KR 不同、又沒有待譯項目的紀錄 → 待譯 `{Reason: "kr-changed", PrevZH: <紀錄的 zh>}`

- [ ] **Step 1: 寫失敗的測試**
  - `TestPlanRewriteUpdatesPendingPrevZH`
  - `TestPlanCarriesARequestAcrossRenumbering`：`Passives.json` 的條目 id 重新編號，舊單位帶有 `requeue` 待譯與 hint → 新單位有同一筆待譯、沒有 `reuse:` 紀錄
  - `TestPlanFrozenCopyIsNotATranslation`
  - `TestPlanStaleRecordQueuesKRChanged`
  - 既有的 `TestPlanTwiceIsIdempotent` 加上這四種情況，第二次仍不變

- [ ] **Step 2: 執行測試確認失敗**

Run: `go test ./internal/plan/`
Expected: FAIL

- [ ] **Step 3: 實作**

- [ ] **Step 4: 執行測試確認通過**

Run: `go vet ./... && go test ./...`
Expected: PASS

- [ ] **Step 5: Commit 並 push**

```bash
git add internal/plan internal/rewrite
git commit -m "fix(plan): rewrites reach pending items, requests survive renumbering, frozen copies and stale records are queued"
git push -u origin claude/festive-wright-g2k853
```

### Task 12: CLI 的暫緩問題與 #1 的 A、C 剩餘項目

**Files:**
- Modify: `cmd/lct/translate.go`、`internal/build/issue.go`、`.github/workflows/pipeline.yml`、`cmd/lct/search.go`、`cmd/lct/requeue.go`、`cmd/lct/adjust.go`、`cmd/lct/batch.go`、`internal/validate/validate.go`
- Test: `cmd/lct/translate_test.go`、`internal/build/issue_test.go`、`cmd/lct/search_test.go`、`cmd/lct/requeue_test.go`、`cmd/lct/adjust_test.go`、`cmd/lct/batch_test.go`、`internal/validate/validate_test.go`、`cmd/lct/contract_test.go`

**Interfaces:**
- Produces:
  - `lct translate --not-run REASON`：不執行 Claude，寫 `.meta/translate.json`（`Stopped: "not-run"`、`Error: REASON`、`Runs: 0`）；pipeline 在有可領的待譯、但缺 `ANTHROPIC_API_KEY` 或 `LCT_MODEL`（不是 `dry_run`）時呼叫它；審核 Issue 在 `Stopped` 是 `"not-run"` 且待譯不是 0 時列一項「翻譯沒有執行」
  - `lct search`、`lct requeue`、`lct adjust add`、`lct review`、`lct impact`、`lct suggest`：`tw` worktree 不存在（沒有 `.meta/`）時 exit 1，訊息說明如何建立
  - `lct search --key K`：指名的 key 不受預設略過 `copy` 的限制
  - `lct batch submit`：規則有未 commit 的修改（rules commit 以 `+dirty` 結尾）時，輸出的 `warnings` 加一句並印到 stderr，仍照常送出（#1 A）
  - 新警告 `extra-keyword-id`（`ClassStyle`）：zh 有 KR 沒有的 `[ID]`（KR 寫韓文狀態名、譯文卻寫成 `[ID]`，#1 C）

- [ ] **Step 1: 寫失敗的測試**
  - `TestTranslateNotRun`、`TestReviewIssueListsATranslationThatDidNotRun`、`TestWorkflowContract`（缺 key 或模型時呼叫 `--not-run`）
  - `TestCommandsNeedTheTWWorktree`：上列六個指令在沒有 `.work/tw` 時 exit 1
  - `TestSearchKeyIncludesCopies`
  - `TestSubmitWarnsOnDirtyRules`
  - `TestExtraKeywordID`：KR `상대에게 화상 2 부여`、zh `給予對方2層[Burn]` → 一個 `extra-keyword-id`；KR 有 `[Burn]` → 沒有

- [ ] **Step 2: 執行測試確認失敗**

Run: `go test ./cmd/lct/ ./internal/build/ ./internal/validate/`
Expected: FAIL

- [ ] **Step 3: 實作**

- [ ] **Step 4: 執行測試確認通過**

Run: `go vet ./... && go test ./...`
Expected: PASS

- [ ] **Step 5: Commit 並 push**

```bash
git add cmd/lct internal/build internal/validate .github/workflows/pipeline.yml
git commit -m "fix: report a translation that did not run, need the tw worktree, warn on dirty rules and stray keyword ids"
git push -u origin claude/festive-wright-g2k853
```

### Task 13: #1 D 類小項（一）：資料與儲存

**Files:**
- Modify: `internal/jsonx/`、`internal/pack/`、`internal/store/store.go`、`internal/cli/`（`FindRoot`）、`internal/fields/learn.go`、`cmd/lct/fields.go`
- Test: 各套件的 `*_test.go`

**Interfaces:**
- Produces（每項一個測試，名稱如下）：
  - `TestEncodeManualNullAndBool`：手動建立的空 `Null`、`Bool` 節點輸出合法 JSON（`null`、`false`）
  - `TestOpenFollowsASymlinkedRoot`：pack 的根目錄是符號連結時照常讀到檔案
  - `TestFileOfKeyRejectsNonCanonical`：`./a.json|/x`、`a/../b.json|/x`、`a//b.json|/x`、`a\b.json|/x` → 錯誤
  - `TestSaveRemovesStaleTemps`：`.meta/units/` 下當機留下的 `.*.tmp*` 在下次 `SaveRecords` 時刪除
  - `TestFindRootStopsOnOtherErrors`：往上找 `lct.yaml` 時只在 `os.IsNotExist` 時繼續，權限錯誤直接回傳
  - `TestLearnWriteKeepsRulesWhenNothingPasses`：`Counted > 0` 但沒有一層過門檻時，`--write` 不清空 `rules/fields.yaml`
  - `TestFieldsTypoIsAUserError`：規則檔有錯字時 `lct fields` exit 1

- [ ] **Step 1: 寫失敗的測試**（上列七個）
- [ ] **Step 2: 執行測試確認失敗**

Run: `go test ./internal/jsonx/ ./internal/pack/ ./internal/store/ ./internal/cli/ ./internal/fields/ ./cmd/lct/ -run 'ManualNull|SymlinkedRoot|NonCanonical|StaleTemps|FindRoot|NothingPasses|FieldsTypo'`
Expected: FAIL

- [ ] **Step 3: 實作**
- [ ] **Step 4: 執行測試確認通過**

Run: `go vet ./... && go test ./...`
Expected: PASS

- [ ] **Step 5: Commit 並 push**

```bash
git add internal cmd/lct
git commit -m "fix: robustness of json, pack, store, root lookup and field learning (#1 D)"
git push -u origin claude/festive-wright-g2k853
```

### Task 14: #1 D 類小項（二）：驗證、批次與發布

**Files:**
- Modify: `internal/validate/`（標籤解析、結束標籤、正規化）、`internal/batch/`、`internal/requeue/`、`internal/publish/publish.go`
- Test: 各套件的 `*_test.go`

**Interfaces:**
- Produces（每項一個測試）：
  - `TestTagNamesAndNonTags`：白名單標籤名稱一律不分大小寫（含 `ruby`）；`<B 사무소…>` 這種名稱後接空格與非屬性文字的不是標籤
  - `TestExtraClosingTagOnlyAtTheEnd`：KR 沒關的標籤，zh 多出的結束標籤只允許在句尾
  - `TestNormalizeSpacesOnlyAfterKeywordIDs`：只在 `[ID]` 前後依指引補空格；`名][Burn]` 中的 `]` 是另一對括號的結尾時不補（先讀 `guides/base.md` 的空格規則，測試案例照它寫）
  - `TestNextFailsOnUnreadableWork`：`work/` 的 `Stat` 錯誤不是「不存在」時回傳錯誤，不無限迴圈
  - `TestStoryBatchWithANeedsReviewLine`：劇情檔中有 `needs_review` 的待譯行時，批次照常產生（該行不發放，當作上下文）
  - `TestStoryContextOfASmallBatch`：只有 1 個單位的劇情批次，前後上下文各不超過 1 個元素
  - `TestRequeueNeedsReviewWithoutSnapshot`：`lct requeue --needs-review` 不需要 raw 快照
  - `TestNotesUseTheLangDir`：release notes 的安裝說明用 `lct.yaml` 的 `lang_dir`

- [ ] **Step 1: 寫失敗的測試**（上列八個）
- [ ] **Step 2: 執行測試確認失敗**

Run: `go test ./internal/validate/ ./internal/batch/ ./internal/requeue/ ./internal/publish/ ./cmd/lct/`
Expected: FAIL

- [ ] **Step 3: 實作**
- [ ] **Step 4: 執行測試確認通過**

Run: `go vet ./... && go test ./...`
Expected: PASS

- [ ] **Step 5: Commit 並 push**

```bash
git add internal cmd/lct
git commit -m "fix: tags, closing tags, keyword spacing, story batches, requeue and notes (#1 D)"
git push -u origin claude/festive-wright-g2k853
```

### Task 15: 規則循環的重播測試

**Files:**
- Create: `cmd/lct/e2e_rules_test.go`
- Test: 同上

**Interfaces:**
- Consumes: Task 1～12 的指令；Plan 3 的 `e2eRepo`、`lct`、`mustLct`、`TestPhase2Replay` 的 fixture 與假 `claude`
- Produces: `TestRulesLoopReplay`

步驟（假的 `gh` 為 shell script，以檔案保存 PR 狀態；origin 為暫存 bare repo）：
1. 沿用 `testdata/phase2` 建置到 v2，報表有一組不一致譯法、一個新說話者、一筆 key 已移動的調整
2. `lct proposal start` → 新分支；`lct suggest --exclude-file work/proposal/exclude.txt` → 三個建議，id 符合 Task 3 的格式
3. 以程式模擬 `rules-review` 的結果：寫一個角色指引、一個重翻要求檔、執行 `lct adjust tidy`、寫 `work/proposal/body.md`；`lct proposal publish` → 假 `gh` 有一個開著的 PR，origin 有該分支、只動了規則檔
4. 再跑一次 `proposal start`＋`suggest` → 已在 PR 中的 id 不再出現
5. 假 `gh` 把 PR 改成「關閉未合併」→ 再跑 → 這三個 id 都視為不採用
6. 把分支合併到 main、`lct plan`、`lct build`、`lct publish`：重翻要求套用一次；只改規則、沒改譯文的那次建置 `release_needed == false`、`same_as` 是上一個 tag

- [ ] **Step 1: 寫失敗的測試**
- [ ] **Step 2: 執行測試確認失敗**

Run: `go test ./cmd/lct/ -run TestRulesLoopReplay`
Expected: FAIL

- [ ] **Step 3: 補齊測試發現的程式問題（每個問題各自 commit，訊息寫明是哪個任務的介面）**
- [ ] **Step 4: 執行測試確認通過**

Run: `go vet ./... && go test ./...`
Expected: PASS

- [ ] **Step 5: Commit 並 push**

```bash
git add cmd/lct
git commit -m "test: rules loop replay from suggestions to a merged rules PR"
git push -u origin claude/festive-wright-g2k853
```

### Task 16（控制者）：上線

- [ ] **Step 1（使用者）:** 在 repo 安裝 Claude GitHub App（https://github.com/apps/claude），允許 `tool-jx3/limbus-translate`
- [ ] **Step 2:** 最終全分支審查（SDD 的 final review），修正後 `git checkout main && git merge --ff-only claude/festive-wright-g2k853 && go test ./... && git push origin main`，再 `git checkout claude/festive-wright-g2k853`。這次 push 動到 `rules/`、`glossary/` 與 `guides/`，會觸發 `rebuild.yml`；預期 plan 套用改寫檔 `2026-11-02-closing-tags`（`rewrite_applied` 2 個單位），build 沒有結構性退回（`structural_fallback` 0、`stand_in` 3，與 `zh-TW-20261029.2` 相同；沒有這個改寫檔是 2 與 5，兩句退回英文）。語言包只差這兩句 `</color>` 的位置，所以會建立 Release `zh-TW-20261029.3`；之後的執行在沒有新 token 時結束（`publish --check` 的 `same_as` 是 `zh-TW-20261029.3`）。這個結果已在暫存複本上以 origin 的 `tw`（`zh-TW-20261029.2`）演練過（最終審查的 F5）
- [ ] **Step 3:** `rebuild` 成功後 `rules-proposal.yml` 會被觸發：確認 `proposal start`、`suggest` 的輸出；有建議時確認 action 執行、PR 以 Claude GitHub App 的身分開出、`ci.yml` 在 PR 上執行並附影響評估；沒有建議時確認在 suggest 之後結束
- [ ] **Step 4:** 確認 action 中 `lct proposal publish` 拿得到 GitHub token（`GH_TOKEN` 或 `GITHUB_TOKEN`）；拿不到時改由 workflow 傳入 action 的 `github_token` 輸出並記錄在 README
- [ ] **Step 5:** 在 #1 回報並勾選完成的項目
