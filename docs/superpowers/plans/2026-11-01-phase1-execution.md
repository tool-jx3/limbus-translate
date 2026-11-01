# 第一階段執行（Plan 2）Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在這個 Claude Code 工作階段完成 spec §8.3 的第一階段：以 10/29 官方語言包建立 `raw`，匯入上游譯文，翻完待譯清單並改進規則，產生第一版 `tw` 並 push，最後由使用者觸發 `rebuild.yml` 發布第一個 Release。

**Architecture:** Task 1–4 是程式碼，以 SDD 執行（每個任務一個 implementer 加一次審查）：補上第一階段需要、但 Plan 1 沒做的 `lct requeue`、`lct publish`，以及兩個最小 workflow。Task 5–11 是營運步驟，由 controller 在本工作階段直接執行。翻譯交給並行的翻譯子代理，它們照 `.claude/skills/translate-pending/SKILL.md` 跑 `lct batch next／submit`。controller 負責品質抽查、改規則、重翻與存檔點。

**Tech Stack:** Go 1.26.1（既有 `lct`）、git worktree（`raw`／`tw` orphan 分支）、GitHub Actions（`actions/checkout@v4`、`actions/setup-go@v5`、`gh` CLI）

**Spec:** `docs/superpowers/specs/2026-10-31-limbus-zh-tw-translation-design.md`（§8.3 第一階段、§8.4、§9.1、§11、§12、§13）

## Global Constraints

- 原文：token `l20261029_n2rX1Jku6_9sBiu9Np24`，created_at `2026-10-29 02:55:23`（UTC+8），raw tag `raw-20261029`
- 10/29 檔數：KR 2,419、EN 2,376、JP 2,426（KR 為輸出檔集合）
- 上游固定版本與 sha256 以 `lct.yaml` 的 `bootstrap` 為準；第一階段結束後不再讀取上游
- 批次：非劇情 40、劇情 120、同一單位失敗 3 次即 `needs_review`、租約 120 分鐘（`lct.yaml` 的 `batch`）
- `origin` 只能是 `paratranz`、`claude`、`human`、`copy`、`fallback`
- 第一階段紀錄的 `model` 一律記 `phase1-session`（`LCT_MODEL=phase1-session`）。repo、skill、prompt、計畫與 commit 都不寫模型名稱或模型 id
- Release tag `zh-TW-YYYYMMDD.N`，YYYYMMDD 取自 raw tag 的日期，N 從 1 起依序遞增；專案名稱與 Release 名稱不得使用「Claude Code」「Anthropic」；說明文件可以寫「使用 Claude Code 翻譯」
- 授權：程式碼 GPL-3.0；`guides/`、`glossary/`、`rules/` 與 `tw` 譯文為 CC BY-NC-SA 4.0；Release notes 要標示授權與來源
- 狀態（紀錄、待譯清單、租約）只能經由 `lct` 修改；讀檔抽查可以直接讀
- 規則只寫在 `guides/`、`glossary/`、`rules/`；改規則要 commit 到開發分支後，才能再發下一波批次（紀錄的 `rules` 才不會是 `+dirty`）
- 不依賴、不讀寫 `tool-jx3/Assets` 與 `tool-jx3/ParatranzUploader` 的程式碼
- 開發分支 `claude/festive-wright-g2k853`；`main` 由開發分支 fast-forward 更新
- push 到 `raw`、`tw`：只在本計畫標明的關卡進行（第一階段例外，spec §8.3 第 6 步；中途存檔點見 Task 1）

## Review Focus

1. **容器中途被回收**：翻譯要跑很多小時，工作階段可能結束。期望：已 push 的 `raw`／`tw` 加上開發分支就能接著做，不必重做 bootstrap（Task 9 的「接續步驟」；`lct requeue --leased` 可釋放死掉的租約，測試在 Task 2）
2. **翻譯中途改規則**：先前已接受的譯文可能違反新規則。期望：`lct requeue` 依 key、檔名樣式或來源把它們送回，帶上理由；凍結檔不送回，已不在 KR 的 key 只列出不處理（測試在 Task 2）
3. **同一份 raw 重建多次**：期望 Release tag 依序遞增（`.1` → `.2`），`tw` 沒變時不產生空 commit，但 Release 照發（測試在 Task 3）
4. **CI 的全新 clone 沒有 tag**：期望 `rebuild.yml` 抓齊 `raw-*` 與 `zh-TW-*` tag 後才算 N（Plan 1 的 T10 曾出現同類錯誤）。workflow 用 `fetch-depth: 0` 並明確 fetch tag，契約測試檢查（Task 4）
5. **翻譯子代理碰到用量上限、中途失敗**：期望沒送出的單位在租約到期後重新發放，部分寫好的 out 檔不影響狀態。controller 每一波結束都核對 `leased`／`available`（Task 9 步驟）

---

### Task 1: 文件與規範整理

**Files:**
- Modify: `docs/superpowers/specs/2026-10-31-limbus-zh-tw-translation-design.md`（§8.3、§9.2、§18）
- Modify: `cmd/lct/contract_test.go`（模型名稱的負面測試字串）

**Interfaces:**
- Produces: spec 寫明第一階段的存檔點與 `model` 標記，供 Task 5–11 依循

- [ ] **Step 1: 改 spec**
  - §9.2「模型由 variable `LCT_MODEL` 指定（預設 …）」改為「模型由 repo variable `LCT_MODEL` 指定，預設值設在 variable，不寫在檔案」
  - §18「第二階段預設模型 …」同樣改寫，不留模型 id
  - §8.3 第 4 步後補一段「第一階段的存檔點」：`raw` 在 snapshot 後即 push（供中斷後接續）；每一波翻譯結束、沒有批次在跑時，commit `tw` 的 `.meta` 並 push `tw`；第一階段紀錄的 `model` 記 `phase1-session`
  - §11.1 第 5 步後補一句：第一階段沒有 CI 的 `plan`，`lct requeue` 直接寫入本機待譯清單；寫 `rules/requeue/*.yaml` 由 CI 套用的方式在第二階段加入
- [ ] **Step 2: 改測試字串** `cmd/lct/contract_test.go` 中的負面測試用的是真實模型 id。改成不存在的 `claude-example-9`（仍符合偵測規則 `claude-[a-z0-9.-]+`），兩處斷言一起改
- [ ] **Step 3: 驗證** — `grep -rniE 'claude-(opus|sonnet|haiku|fable)|opus-[0-9]|sonnet-[0-9]' --include='*.md' --include='*.go' --include='*.yaml' --include='*.yml' --include='*.json' . | grep -v '^./.work'` 無輸出；`go test ./cmd/lct/ -run 'Contract|Skill|ModelNames'` PASS
- [ ] **Step 4: Commit & push** — `docs: record phase 1 checkpoints and keep model ids out of the repo`

### Task 2: `lct requeue`（第一階段直接模式）與 `hint`

**Files:**
- Create: `internal/requeue/requeue.go`、`internal/requeue/requeue_test.go`、`cmd/lct/requeue.go`、`cmd/lct/requeue_test.go`
- Modify: `internal/store/store.go`（`PendingItem.Hint`）、`internal/batch/batch.go`、`internal/batch/next.go`（`BatchUnit.Hint`）、`schemas/batch.schema.json`、`prompts/translate.md`（`hint` 一行）

**Interfaces:**
- Consumes: `store.Open`、`(*Store).WithLock`、`LoadRecords`／`SaveRecords`／`LoadPending`／`SavePending`；`pack.OpenRaw`、`units.Extract`、`units.KRSha`、`units.SplitKey`；`config.MatchAny`
- Produces:
  - `store.PendingItem.Hint string`（`json:"hint,omitempty"`）；`batch.BatchUnit.Hint string`（`json:"hint,omitempty"`），由 Next 從 pending item 帶入
  - `requeue.Selector{Keys []string; Files []string; Origins []string; NeedsReview bool; Leased bool}`（`Files` 是 `config.MatchAny` 的樣式，比對 key 的檔名部分）
  - `requeue.Result{Requeued, Reset, Released, AlreadyPending, SkippedFrozen, SkippedCopy, Missing int; MissingKeys []string}`
  - `requeue.Apply(st *store.Store, kr *pack.Pack, frozen []string, sel Selector, hint string) (*Result, error)`：只呼叫一次 `WithLock`，在裡面讀改寫
  - CLI：`lct requeue [--key K]... [--file GLOB]... [--origin O]... [--needs-review] [--leased] --reason TEXT [--json]`。至少要給一個選擇條件，`--reason` 必填，否則是使用者錯誤（exit 1）。輸出 `Result`

規則：
- `--needs-review`：pending 中帶 `needs_review` 的項目，移除旗標、`Attempts=0`、清租約、`Hint=reason`，計入 `Reset`。它們的 fallback 紀錄保留，重譯送出後會被取代
- `--leased`：清除符合條件的 pending 項目的租約（不論是否過期），計入 `Released`，不改 `Attempts`
- `--key`、`--file`、`--origin` 選紀錄。每一筆符合的紀錄：
  - `copy` → 計入 `SkippedCopy`
  - 檔案凍結 → 計入 `SkippedFrozen`
  - 已在 pending → 補上 `Hint`，計入 `AlreadyPending`
  - KR 中已沒有這個 key → 計入 `Missing` 並列入 `MissingKeys`
  - 其他 → 新增 `PendingItem{Key, Reason:"requeue", PrevZH: rec.ZH, Hint: reason}`，計入 `Requeued`；紀錄保留，重譯前仍照常輸出
- 多個條件同時給：`--key`／`--file` 決定範圍，`--origin` 再篩選；`--needs-review`／`--leased` 只作用在 pending
- 預設 `--origin` 為 `paratranz`、`claude`、`fallback`、`human`（不含 `copy`）

- [ ] **Step 1: 寫失敗測試**
  ```go
  func TestRequeueByFile(t *testing.T)        // 2 筆 claude 紀錄在 StoryData/S001A.json、1 筆 copy、1 筆在 BgmLyrics/X.json（frozen）→ --file "StoryData/S001A.json" → Requeued 2、SkippedCopy 1；pending 有 2 筆 reason "requeue"、PrevZH==紀錄 zh、Hint==reason；紀錄不變
  func TestRequeueSkipsFrozenAndMissing(t *testing.T) // --file "BgmLyrics/**" → SkippedFrozen 1；--key 指向 KR 已不存在的 pointer → Missing 1、MissingKeys 含該 key
  func TestRequeueNeedsReview(t *testing.T)   // pending 一筆 needs_review、Attempts 3 → Reset 1；旗標移除、Attempts 0、Hint 設定；之後 batch.Next 會發出它，BatchUnit.Hint==reason
  func TestRequeueLeased(t *testing.T)        // 一筆有租約 → --leased → Released 1、Lease nil、Attempts 不變
  func TestRequeueAlreadyPending(t *testing.T) // 已在 pending 的 key → AlreadyPending 1、只更新 Hint、不重複
  func TestRequeueOneLock(t *testing.T)       // 與 8 個並行的 batch.Submit 交錯執行，最後 records 與 pending 一致（不遺失更新），-race 通過
  func TestRequeueCLIUsage(t *testing.T)      // 沒有選擇條件或沒有 --reason → exit 1；--json 輸出 Result 的欄位名（snake_case）
  ```
  另在 `internal/batch` 加 `TestNextCarriesHint`，在 `cmd/lct/contract_test.go` 的 schema 測試確認 `hint` 在 `batch.schema.json` 的 unit properties 中
- [ ] **Step 2: 執行確認失敗** — `go test ./internal/requeue/... ./internal/batch/... ./cmd/lct/ -run 'Requeue|Hint|Schema'` FAIL
- [ ] **Step 3: 實作** — 依上方 Interfaces 與規則；`prompts/translate.md` 補一行：「`hint`：人要求重翻的理由，照它修正，其他部分盡量保留 `prev_zh`」
- [ ] **Step 4: 執行確認通過** — 同上指令 PASS；`go test -race ./internal/requeue/... ./internal/batch/...` PASS；`go vet ./... && go test ./...` PASS
- [ ] **Step 5: Commit & push** — `feat(requeue): send records back to the pending list with a reason`

### Task 3: `lct publish`

**Files:**
- Create: `internal/publish/publish.go`、`internal/publish/publish_test.go`、`cmd/lct/publish.go`、`cmd/lct/publish_test.go`

**Interfaces:**
- Consumes: `gitx.CommitAll(dir, msg) (sha string, changed bool, err error)`、`gitx.Tags(dir, prefix) ([]string, error)`、`(*Store).ReadMetaJSON`（讀 `.meta/build.json`）、`build.Report`
- Produces:
  - `publish.ReleaseTag(rawTag string, existing []string) (string, error)`：`raw-20261029` 或 `raw-20261029.2` → `zh-TW-20261029.<1+同日已有的最大 N>`
  - `publish.BuildInfo{RawTag, RawToken, RulesCommit string; Report build.Report}`：`.meta/build.json` 解碼後的內容（JSON 鍵 `raw_tag`、`raw_token`、`rules_commit`、`report`）
  - `publish.Notes(info BuildInfo, tag string) []byte`：release notes markdown，內容依序為：
    1. 標題 `zh-TW-YYYYMMDD.N`
    2. 一句說明：Limbus Company 臺灣繁體中文語言包，使用 Claude Code 翻譯
    3. 安裝方式：把 zip 內的 `LimbusCompany_Data/` 解壓到遊戲資料夾，在遊戲設定選擇 zh-TW
    4. 原文版本：raw tag 與 token
    5. 規則版本：rules commit
    6. 報表摘要：origins、pending、needs_review、structural_fallback、frozen_fallback、missing glyphs 數
    7. 授權與來源：譯文 CC BY-NC-SA 4.0，含 Paratranz 社群譯文（第一階段匯入），原文著作權屬 Project Moon，字型 Sarasa Gothic（SIL OFL）
  - `publish.Run(root, twRoot, outDir string) (*Result, error)`，`Result{TWCommit string; Changed bool; Tag string; NotesPath string; Assets []string}`。流程：讀 `.meta/build.json`；`gitx.CommitAll(twRoot, "build: <raw tag> (rules <rules commit>)")`；以 `gitx.Tags(root, "zh-TW-")` 算 tag；寫 `<outDir>/release-notes.md`；`Assets` 為 `out/zh-TW_text_<raw tag>.zip`、`out/zh-TW_complete_<raw tag>.zip`（兩者都必須存在，否則使用者錯誤）
  - CLI：`lct publish [--json]`。不 push、不建立 Release（由 workflow 執行）

- [ ] **Step 1: 寫失敗測試**
  ```go
  func TestReleaseTag(t *testing.T)     // ("raw-20261029", nil) → "zh-TW-20261029.1"；existing ["zh-TW-20261029.1","zh-TW-20261029.3","zh-TW-20261105.1"] → ".4"；"raw-20261029.2" 也取 20261029；"bogus" → error
  func TestNotes(t *testing.T)          // 含 tag、raw tag、token、rules commit、"CC BY-NC-SA 4.0"、"Project Moon"、"使用 Claude Code 翻譯"；不含 "Anthropic"
  func TestPublishCommitsAndTags(t *testing.T) // 暫存 repo + tw worktree + 假 build.json 與 zip → Changed true、TWCommit==tw HEAD；再跑一次 → Changed false、不產生新 commit、Tag 相同（尚未建立 Release tag 時）
  func TestPublishNeedsBuild(t *testing.T)     // 沒有 build.json 或少一個 zip → exit 1
  ```
- [ ] **Step 2: 執行確認失敗** — `go test ./internal/publish/... ./cmd/lct/ -run Publish` FAIL
- [ ] **Step 3: 實作**
- [ ] **Step 4: 執行確認通過** — 同上 PASS；`go vet ./... && go test ./...` PASS
- [ ] **Step 5: Commit & push** — `feat(publish): commit the tw build and prepare the release`

### Task 4: `ci.yml` 與最小 `rebuild.yml`

**Files:**
- Create: `.github/workflows/ci.yml`、`.github/workflows/rebuild.yml`
- Modify: `cmd/lct/contract_test.go`（加 workflow 檢查）

**Interfaces:**
- Consumes: `lct build`、`lct publish`、`lct rules lint`
- Produces: 使用者可在 Actions 頁面手動執行 `rebuild`，產生 `zh-TW-YYYYMMDD.N` Release。Plan 3 會再加上 push 觸發、`plan` 與翻譯迴圈

內容：
- `ci.yml`：`on: [push, pull_request]`；`actions/checkout@v4`、`actions/setup-go@v5`（`go-version-file: go.mod`）；`go vet ./...`、`go test ./...`、`go build -o bin/lct ./cmd/lct`、`bin/lct rules lint`
- `rebuild.yml`：
  - `on: workflow_dispatch`
  - `permissions: contents: write`
  - `concurrency: { group: lct-state, cancel-in-progress: false }`
  - 步驟：
    1. checkout，`fetch-depth: 0`
    2. `git fetch origin '+refs/heads/raw:refs/remotes/origin/raw' '+refs/heads/tw:refs/remotes/origin/tw' '+refs/tags/*:refs/tags/*'`，接著 `git worktree add .work/raw raw`（`lct build` 只會自行建立 tw worktree，不會建立 raw）
    3. setup-go，`go build -o bin/lct ./cmd/lct`
    4. 設定 git 身分為 `github-actions[bot]`
    5. `bin/lct build --json`
    6. `bin/lct publish --json > out/publish.json`
    7. `git push origin tw`
    8. 以 `gh release create "$tag" <assets> --target "$tw_commit" --title "$tag" --notes-file out/release-notes.md` 建立 Release（`GH_TOKEN: ${{ github.token }}`；`tag`、`tw_commit`、`assets` 以 `jq` 從 `out/publish.json` 取）
  - 不用 Claude、不用任何 secret
- 契約測試新增 `TestWorkflowContract`，四項檢查：
  1. 每個 `.github/workflows/*.yml` 中的 `bin/lct <cmd>` 都是已註冊的指令（沿用既有的 reference 規則）
  2. 不含模型名稱
  3. `rebuild.yml` 有 `concurrency.group: lct-state`、`fetch-depth: 0`、`refs/tags/*` 的 fetch 與 `git worktree add .work/raw raw`
  4. `--title` 不含「Claude Code」「Anthropic」

- [ ] **Step 1: 寫失敗測試** `TestWorkflowContract`（上述四項，各附一個負面子測試）
- [ ] **Step 2: 執行確認失敗** — `go test ./cmd/lct/ -run WorkflowContract` FAIL（檔案不存在）
- [ ] **Step 3: 寫兩個 workflow**
- [ ] **Step 4: 執行確認通過** — `go test ./cmd/lct/ -run WorkflowContract -v` PASS；YAML 可解析
- [ ] **Step 5: Commit & push** — `ci: add tests workflow and a manual rebuild that publishes a release`

---

以下 Task 5–11 由 controller 在本工作階段直接執行，不派 implementer。指令都在 repo 根目錄執行。每一步的結果（數字、決定、commit）都記到 SDD ledger。

### Task 5: 抓取並建立 `raw-20261029`

- [ ] **Step 1:** `go build -o bin/lct ./cmd/lct`
- [ ] **Step 2:** `bin/lct fetch --token l20261029_n2rX1Jku6_9sBiu9Np24 --created-at "2026-10-29 02:55:23" --out work/fetch --json`。預期：exit 0，`work/fetch/{kr,en,jp}`
- [ ] **Step 3:** `bin/lct snapshot --from work/fetch --token l20261029_n2rX1Jku6_9sBiu9Np24 --created-at "2026-10-29 02:55:23" --json`。預期：`tag == "raw-20261029"`、`changed == true`；`.work/raw/meta/source.json` 中 kr／en／jp 的檔數為 2,419／2,376／2,426
- [ ] **Step 4（關卡：push `raw`）:** `git push origin raw refs/tags/raw-20261029`。預期：遠端出現 `raw` 分支與 tag

### Task 6: 以真實原文學習欄位規則

- [ ] **Step 1:** `bin/lct fields learn --json` → 讀 `changes`。預期：出現 `id`、`model`、`usage` 等識別欄位，以及 `BattleSpeechBubbleDlg` 的 `desc`（群組）
- [ ] **Step 2:** 逐項判斷：與 `rules/fields.manual.yaml` 衝突、或疑似把對白欄位判成不翻的，寫進手動規則修正（例如 `content`、`dlg`、`desc` 在一般檔案必須可翻）
- [ ] **Step 3:** `bin/lct fields learn --write`；`git diff --stat rules/`；`bin/lct rules lint` exit 0
- [ ] **Step 4: Commit & push** — `rules: learn field rules from raw-20261029`

### Task 7: 匯入與基準建置

- [ ] **Step 1:** `bin/lct bootstrap --fetch --json`。記下 Stats，預期與 Plan 1 試跑同一數量級：paratranz 約 16.2 萬、pending 約 1.9 萬、`refs-ltm` 與 `refs-llc` 大於 0、`unaligned-paratranz` 數百以內。學到欄位規則後 copy 應增加、pending 應減少；若 pending 反而大增，停下檢查 Task 6
- [ ] **Step 2:** `bin/lct pending --count --json`，記下 `by_reason`（`new`／`paratranz-repair`）
- [ ] **Step 3:** `/usr/bin/time -v bin/lct build --json`，記下耗時與 Maximum resident set size。讀 `.work/tw/.meta/reports/raw-20261029.md`：
  - `missing_glyphs`：有缺字就列出，判斷是否需要換字型
  - `structural_fallback`、`frozen_fallback`：應接近 0；不是的話逐條看原因
  - `warnings`：前幾名的 code
- [ ] **Step 4（關卡：push `tw`）:** `git -C .work/tw add -A && git -C .work/tw commit -m "phase 1: bootstrap from raw-20261029"`，再 `git push origin tw`

### Task 8: 試譯與校準

- [ ] **Step 1:** 從 `bin/lct pending --json` 的 key 統計待譯量最多的族群。每個有待譯的族群各派一個翻譯子代理（同時最多 6 個，背景執行），只做 1 批：`bin/lct batch next --family <F> --json`。派發內容見下方「翻譯子代理的派發內容」，加上「只做這一批」
- [ ] **Step 2:** controller 逐批對照 `work/batches/<id>.json` 與 `<id>.out.json` 檢查：
  - 您／你、你／妳
  - 角色語氣
  - `[ID]` 與標籤
  - 名稱是否與 `keywords`／詞彙表一致
  - 標點
  - 劇情上下文是否連貫
- [ ] **Step 3:** 問題歸類：
  - 規則缺漏 → 改 `guides/`／`glossary/`
  - 提示不清 → 改 `prompts/translate.md`，需通過契約測試
  - 個別錯譯 → `bin/lct requeue --key <K> --reason "<理由>"`
- [ ] **Step 4:** `bin/lct rules lint`；`go test ./cmd/lct/ -run Contract`；commit `rules: calibrate from phase 1 pilot`；確認 `git status --porcelain -- guides glossary rules prompts` 是空的

**翻譯子代理的派發內容**：
- 模型：最強可用的模型
- 必須寫進派發提示：
  1. 讀並照做 `.claude/skills/translate-pending/SKILL.md`；跳過第 0 步，`bin/lct` 由 controller 建好
  2. 每次送出都用 `LCT_MODEL=phase1-session bin/lct batch submit <id> --json`
  3. 送出 N 批後，或 `batch` 為 null 就停（N 由該波決定，預設 6）
  4. 結束時回報：每批的 id、接受數、退回後修好的數、needs_review 的 key、`released` 是否非空，以及遇到的規則問題（同時寫進 `note`）
  5. 只能寫 `work/`，不得改 `guides/`、`glossary/`、`rules/`、`.meta/`

### Task 9: 翻譯波次（重複直到沒有可發放的單位）

每一波：

- [ ] **Step 1: 開始前檢查**
  - `git status --porcelain -- guides glossary rules prompts` 是空的
  - `go build -o bin/lct ./cmd/lct`
  - `bin/lct pending --count --json` 記下 `total`、`available`、`leased`、`needs_review`
- [ ] **Step 2:** 派 6 個翻譯子代理並行（背景），各做最多 6 批（內容同 Task 8，不加 `--family`）
- [ ] **Step 3:** 全部回報後，再跑一次 `bin/lct pending --count --json`
  - 若 `leased > 0` 且沒有子代理在跑，表示有子代理中途失敗。等租約到期，或 `bin/lct requeue --leased --reason "worker stopped"`
- [ ] **Step 4: 抽查**
  - controller 用 scratchpad 的腳本，從本波的 `work/batches/*.out.json` 隨機抽 40 個已接受的單位，每批至少 1 個，連同 KR／EN 與所屬族群寫成檔案
  - 派一個審查子代理（最強可用的模型），依 `guides/` 與 `glossary/` 逐條判定 OK、錯譯或規則缺漏，附理由
- [ ] **Step 5: 改進規則**
  - 讀本波新增的 `.work/tw/.meta/notes.jsonl`、子代理回報的規則問題，以及抽查結果
  - 改 `guides/`／`glossary/`，跑 `bin/lct rules lint`，commit `rules: <主題>`
  - 改了會影響既有譯文的規則，就用 `bin/lct requeue --file|--key ... --reason "<新規則摘要>"` 送回受影響的單位
- [ ] **Step 6（關卡：push `tw` 存檔點）:**
  - 確認沒有子代理在跑
  - `git -C .work/tw add -A && git -C .work/tw commit -m "phase 1: wave <n>"`
  - `git push origin tw`
  - 開發分支有規則 commit 時一併 push
- [ ] **Step 7:** ledger 記一行：`wave <n>: batches <b>, accepted <a>, needs_review <r>, available <v>, rules <commits>`
- [ ] **結束條件:** `available == 0` 且 `leased == 0`
- [ ] **用量上限:** 子代理因用量上限失敗時，暫停派發，稍後再開下一波

**接續步驟**（新的工作階段或容器）：
1. clone repo，checkout 開發分支
2. `git fetch origin raw tw '+refs/tags/raw-*:refs/tags/raw-*'`
3. `git worktree add .work/raw raw`
4. `git worktree add .work/tw tw`
5. `go build -o bin/lct ./cmd/lct`
6. `bin/lct requeue --leased --reason "resume"`
7. 從 Task 9 Step 1 繼續，不要重跑 bootstrap

### Task 10: needs_review 與最終品質檢查

- [ ] **Step 1:** `bin/lct pending --json` 篩出帶 `needs_review` 的項目，依族群與退回原因分類。看原因時直接讀 `.work/tw/.meta/units/` 中的 fallback 紀錄與 `work/batches/`
- [ ] **Step 2:** 原因是規則或提示 → 依 Task 9 Step 5 修；原因是驗證程式誤判 → 停下，開一個程式碼修正任務（走 SDD：implementer 加審查）
- [ ] **Step 3:** `bin/lct requeue --needs-review --reason "<修了什麼>"`，再跑一波（Task 9）
- [ ] **Step 4:** 跑 `bin/lct build --json`，讀報表：
  - `needs_review`、`structural_fallback`、`frozen_fallback`、`missing_glyphs`
  - `warnings` 前幾名
  - `field_rule_override`，應為 0
  
  殘留的 needs_review 列入最終回報，在第二階段的審核 Issue 處理
- [ ] **Step 5:** 最終抽查：每個族群 20 個 claude 譯文、10 個 paratranz 譯文，同 Task 9 Step 4。修正後再跑一波；沒有新問題就結束

### Task 11: 收尾：push、合併到 `main`、第一個 Release

- [ ] **Step 1:** `bin/lct build --json`，接著 `bin/lct publish --json`。預期：`changed == true`、`tag == "zh-TW-20261029.1"`；兩個 zip 都在 `out/`
- [ ] **Step 2（關卡：push `tw`）:** `git push origin tw`
- [ ] **Step 3:** 合併到 `main`：`git checkout main && git merge --ff-only claude/festive-wright-g2k853 && go test ./... && git push origin main`，再 `git checkout claude/festive-wright-g2k853`
- [ ] **Step 4（使用者）:** 到 GitHub 的 repo Settings → General → Default branch，改成 `main`
- [ ] **Step 5（使用者）:** Actions → `rebuild` → Run workflow（branch `main`）
- [ ] **Step 6:** 確認 Release `zh-TW-20261029.1`：
  - 兩個 zip 都在
  - release notes 有授權、來源與報表摘要
  - complete zip 內有 `LimbusCompany_Data/Lang/config.json` 與 `zh-TW/Font/Context/SarasaGothicTC-Bold.ttf`
  
  `tw` 在 CI 重建後應沒有變動（`publish.json` 的 `changed == false`），或只有 `version.json`
- [ ] **Step 7:** 最終回報給使用者：
  - 各來源的數量
  - 殘留的 needs_review 與 fallback
  - 本階段改了哪些規則（commit 清單）
  - 缺字
  - 第二階段（Plan 3）要接手的事項
