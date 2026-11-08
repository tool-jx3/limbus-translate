# 更新頁變動原因 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 每筆更新紀錄的變動都附上原因（改寫檔、人工調整、重翻申請、KR 改版等），網站「更新」頁可依原因分組並摺疊檢視；過去 10 筆紀錄由 CI 補做一次。

**Architecture:**
- `lct publish` 產生 `.meta/history/<tag>.json` 時，`history` 用前後兩個 Release 的狀態，加上兩個 `rules_commit` 的規則檔，推斷每筆 Entry 的原因，存進 `causes`。
- `causes_version` 比目前版本舊的紀錄，由 `planHistory` 重做。
- 網站讀 `updates/<tag>.json`，新增「依原因」檢視。

**Tech Stack:** Go（`internal/history`、`internal/publish`、`internal/requeue`、`internal/gitx`、`cmd/lct`）；Preact + TypeScript（`site/src/updates`）。

**Spec:** `docs/superpowers/specs/2026-11-08-update-causes-design.md`

## Global Constraints

- tw 只由 CI 寫入；本計畫不推 tw，補做靠 CI 的 publish（spec「補做舊紀錄」）。
- 所有 `lct` 指令支援 `--json`；exit code：0 成功、1 有發現或使用者錯誤、2 系統錯誤。
- 推斷失敗不中斷紀錄：該筆標 `unknown` 並附 `note`，紀錄照常寫出（spec「資料讀不到時」）。
- 改寫只在依序重放的結果**完全等於**新譯文時才記為原因（spec 第 5 步）。
- 舊的 `Entry.cause`（adjustment／late／retranslated）保留不動，新欄位是 `causes`。這樣舊網站與舊資料都不受影響。
- 網站：400px 寬不得整頁橫向捲動，沿用現有樣式變數；舊格式資料（沒有 `causes`）照常顯示。
- 提交前：`go vet ./... && go test ./...`；動到 `site/` 另跑 `cd site && npm test && npm run typecheck && npm run build`。
- Commit 訊息繁體中文，結尾附上工作階段指定的 `Co-Authored-By` 與 `Claude-Session` 兩行。程式、文件、skill 不寫模型名稱。

## Review Focus

1. **重跑 publish 不重做**：重跑 publish 時，`causes_version` 已是目前版本的紀錄不再重做；舊版本的只重做一次（Task 4 測試）。
2. **讀不到規則檔**：commit 不存在或規則檔已刪除時，該筆是 `unknown`，`note` 說明原因；其餘照常（Task 2、Task 3 測試）。
3. **改寫被蓋掉**：改寫套上後又被重翻或調整蓋掉，重放結果不等於新譯文，就不列改寫（Task 3 測試）。
4. **重翻申請跨 Release**：申請在 Release N 生效、單位在 N+1 才翻完，要能從 N 的 `pending.jsonl`（原因 `requeue`）加上所有已套用申請的選擇條件，對回該申請（Task 3 測試）。
5. **網站的舊資料與深連結**：網站讀到沒有 `causes` 的舊資料不出錯；從標籤跳到「依原因」的某一組時，該組要展開（Task 6 測試）。

---

### Task 1: Release 的 applied／pending 與某個 commit 的規則檔

**Files:**
- Modify: `internal/history/release.go`（`Side` 加欄位，`readSide` 載入）
- Create: `internal/history/rules.go`（讀某 commit 的規則檔）
- Modify: `internal/requeue/requeue.go`（匯出選擇判斷）
- Test: `internal/history/release_test.go`、`internal/history/rules_test.go`、`internal/requeue/requeue_test.go`

**Interfaces:**
- Produces:
  - `history.Side` 新增 `Applied store.Applied`、`Pending []store.PendingItem`，由 `readSide` 用 `store.Open(root).LoadApplied()`、`LoadPending()` 讀。沒有檔案時是空值，不是錯誤。
  - `type RuleFiles struct { Commit string; Rewrites []rewrite.File; Requeues []requeue.File; Adjustments []adjust.Adjustment }`
  - `func RulesAt(repo, commit, tmp string) (*RuleFiles, error)`：用 `gitx.Archive(repo, commit, []string{"rules"}, dest)` 取出，再呼叫 `rewrite.LoadFiles`、`requeue.LoadFiles`、`adjust.Load`。commit 不存在時回錯誤，訊息含 commit。
  - `type RulesSource func(commit string) (*RuleFiles, error)`：給 Task 3 使用，同一個 commit 只讀一次（快取）。
  - `func (s requeue.Selector) Selects(key string, rec store.Record) bool`：與 `requeue` 現行選擇規則相同，即 `inScope` 加 `originOK`。現有的 `run` 改用它，不得重複邏輯。

- [ ] **Step 1：寫失敗的測試**
  - `TestReadRevLoadsAppliedAndPending`：tw 測試 repo 的 tag 有 `applied.json`（rewrite `a`、requeue `r`）與 `pending.jsonl`（一筆 reason `requeue`）；`Side.Applied`、`Side.Pending` 要讀到。
  - `TestRulesAtReadsRulesOfCommit`：兩個 commit 的 `rules/rewrite/x.yaml` 內容不同，各自讀到該版的內容。
  - `TestRulesAtMissingCommit`：回錯誤，且錯誤含 commit 字串。
  - `TestSelectorSelects`：key 列名、files 樣式、origins 篩選三種情況，與 `ApplyTo` 選到的集合一致。
- [ ] **Step 2：執行，確認失敗**：`go test ./internal/history/ ./internal/requeue/ -run 'ReadRevLoads|RulesAt|SelectorSelects'`，預期 FAIL。
- [ ] **Step 3：實作上列介面**
- [ ] **Step 4：執行，確認通過**：同上加 `go test ./...`，預期 PASS。
- [ ] **Step 5：Commit**

### Task 2: 原因的資料格式

**Files:**
- Modify: `internal/history/compare.go`（`Entry` 加 `Causes`）、`internal/history/record.go`（`Record` 加欄位）
- Create: `internal/history/causes.go`（型別與常數）
- Test: `internal/history/record_test.go`

**Interfaces:**
- Produces:
  - `type Cause struct { Kind string \`json:"kind"\`; ID string \`json:"id,omitempty"\`; Batch string \`json:"batch,omitempty"\`; Reason string \`json:"reason,omitempty"\`; Note string \`json:"note,omitempty"\` }`
    - `Batch`：翻譯批次。
    - `Reason`：調整那一筆的 reason，或 `pending.jsonl` 的原因。
    - `Note`：`unknown` 的說明。
  - `Entry.Causes []Cause \`json:"causes,omitempty"\``
  - 原因種類常數 `CauseKind…`，值為：
    - `rewrite`、`adjustment`、`adjustment-expired`、`requeue`
    - `reuse`、`kr-changed`、`new`、`deleted`
    - `late`（原本是替代文字、首次補翻）、`retranslated`、`other`、`unknown`
    - 前四個是「規則」。另提供 `func IsRuleCause(kind string) bool`。
  - `type RuleInfo struct { Kind, ID, Path, Title, Commit string }`，json 欄位 `kind`、`id`、`path`、`title`、`commit`。
  - `Record.Rules map[string]RuleInfo \`json:"rules,omitempty"\``：key 為 `"<kind>:<id>"`。
  - `Record.CausesVersion int \`json:"causes_version,omitempty"\``
  - `Record.Unknown int \`json:"unknown,omitempty"\``
  - `const CausesVersion = 1`
- [ ] **Step 1：寫失敗的測試**
  - `TestRecordRoundTripsCauses`：寫入再讀回，`Causes`、`Rules`、`CausesVersion` 相同。
  - `TestOldRecordReadsWithoutCauses`：沒有新欄位的舊 json 讀得進來，`CausesVersion == 0`，舊的 `cause` 照舊。
- [ ] **Step 2–4：先看它失敗，實作後看它通過**（`go test ./internal/history/`）
- [ ] **Step 5：Commit**

### Task 3: 推斷原因

**Files:**
- Modify: `internal/history/causes.go`
- Modify: `internal/history/release.go`（`Make` 帶 `RulesSource`，產生 `Causes`、`Rules`、`Unknown`、`CausesVersion`）
- Test: `internal/history/causes_test.go`

**Interfaces:**
- Consumes: Task 1 的 `Side.Applied`／`Pending`、`RuleFiles`、`RulesSource`、`Selector.Selects`；Task 2 的型別。
- Produces:
  - `func Make(prev, cur *Side, tag string, at time.Time, langDir string, rules RulesSource) (*Record, error)`：`rules` 為 nil 時不推斷，`CausesVersion` 為 0，讓舊呼叫者與測試可用。
  - `func inferCauses(prev, cur *Side, prevRules, curRules *RuleFiles, e Entry) []Cause`
- 規則照 spec「原因的推斷」1–9。要點：
  - 改寫重放：依 `rewrite.Apply` 的條件，但不含 `ok`。KR 用 `cur` 的，origin 用舊紀錄的。只有最後結果等於新譯文才列。
  - 調整：依 `kr_sha` 加 `zh` 對到調整那一筆。先找 `curRules`，再找 `prevRules`；失效的調整從 `prevRules` 找。
  - 重翻申請：
    - (a) 單位在 `prev.Pending` 裡、原因是 `requeue`，用 `cur.Applied.Requeue` 全部的申請比對 `Selects`，取最後一個符合的；
    - (b) 兩個 tag 之間新套用的申請選到的單位。
  - KR 改版：新紀錄是翻譯批次產生的，`kr-changed` 那一項帶 `Batch`（spec 第 1 步的「再加上翻譯」）。
  - `late`：舊來源是 stand-in 或 fallback，且新紀錄是翻譯產生的。spec 的種類清單沒有這一項；計畫加上它，因為這種單位在 `pending.jsonl` 的原因多半是 `new`，照第 8 步會顯示成「重翻（new）」，其實是第一次翻譯。判斷順序在第 8 步之前。（計畫的決定，交付時告知使用者）
  - `Record.Rules`：調整檔的 `Title` 是檔名；各筆調整的 reason 放在該筆 `Cause.Reason`，因此只會出現被用到的那幾筆。
  - `RulesSource` 回錯誤時，需要規則的那幾筆是 `unknown`，`note` 為錯誤訊息。
- [ ] **Step 1：寫失敗的測試**，每種原因各一。用 `compare_test.go` 的 `state`、`rec` 組小型 `Side`；需要 git 的情況用 `release_test.go` 的 `repoFixture`。
  - `TestCausesRewriteChain`：兩個改寫接連改，`Causes` 依序為兩個 rewrite。
  - `TestCausesRewriteOverwritten`：改寫後又被重翻，不列 rewrite，列 `retranslated`。
  - `TestCausesAdjustmentByKRSha`：調整檔的 key 已被 tidy 搬走，仍對到，`Reason` 為該筆 reason。
  - `TestCausesAdjustmentExpired`：舊紀錄是 `adjustment:x`、新的不是，從 `prevRules` 讀到。
  - `TestCausesRequeueAcrossReleases`：申請不在兩個 tag 的差集裡，但單位在 prev 的 pending 裡、原因是 `requeue`，且符合選擇條件，列 `requeue`，附 `Batch`。
  - `TestCausesRequeueSameRelease`：申請在兩個 tag 之間新套用，單位在同一段期間翻完（prev 的 pending 沒有它），列 `requeue`（spec 第 6 步 (b)）。
  - `TestCausesRetranslatedByPendingReason`：pending 原因 `structural-fail`，列 `retranslated`，`Reason` 為 `structural-fail`。
  - `TestCausesReuseLateKRNewDeleted`：涵蓋 reuse、late、kr-changed（加翻譯）、new、deleted。
  - `TestCausesRulesUnavailable`：`RulesSource` 回錯誤，相關筆為 `unknown` 並有 `note`；`Record.Unknown` 計數正確；紀錄照常產生。
  - `TestMakeRulesIndex`：`Record.Rules` 只列實際用到的規則，`Title` 取規則檔的 `reason`，調整檔為檔名。
- [ ] **Step 2–4：先看它失敗，實作後看它通過**（`go test ./internal/history/` 加上 `go test ./...`）
- [ ] **Step 5：Commit**

### Task 4: publish 補做舊紀錄，以及唯讀的 `lct history make`

**Files:**
- Modify: `internal/publish/history.go`（`planHistory`）
- Create: `cmd/lct/history.go`（新指令）、`cmd/lct/history_test.go`
- Modify: `cmd/lct/main.go`（註冊）、`README.md`（指令說明）
- Test: `internal/publish/history_test.go`

**Interfaces:**
- Consumes: Task 3 的 `Make(..., rules)`、Task 1 的 `RulesAt`。
- Produces:
  - `planHistory` 對每個已有紀錄的 Release 讀 `CausesVersion`：小於 `history.CausesVersion` 就重做，用前一個 Release 重新產生。`RulesSource` 由 `RulesAt(root, commit, tmp)` 加快取構成。
  - `lct history make <tag> [--json]`：用 tw 的 tags 與 main 的 git，產生該 Release 對前一個的紀錄。
    - 輸出到 stdout，或用 `--out <檔>` 寫到檔案。**不寫 tw**。
    - 文字模式列出各原因的筆數。
    - 用途是驗收與除錯。
- [ ] **Step 1：寫失敗的測試**
  - `TestPlanHistoryRemakesOldCausesVersion`：已有一筆 `causes_version` 為 0 的紀錄，會被列入重做；重做後再跑一次，不再列入（Review Focus 1）。
  - `TestPlanHistoryKeepsCurrentVersion`：`causes_version == CausesVersion` 的紀錄不重做。
  - `TestHistoryMakeIsReadOnly`：執行前後 tw worktree 一字不變；`--json` 輸出含 `causes`。
- [ ] **Step 2–4：先看它失敗，實作後看它通過**（`go test ./internal/publish/ ./cmd/lct/` 加上 `go test ./...`）
- [ ] **Step 5：Commit**

### Task 5: 網站資料層，依原因分組

**Files:**
- Modify: `internal/site/updates.go`（確認 `causes`、`rules` 原樣帶出；若有欄位白名單就加上）
- Modify: `site/src/types.ts`（`UpdateEntry.causes`、`UpdateRecord.rules`／`causes_version`／`unknown`）
- Create: `site/src/updates/causes.ts`、`site/src/updates/causes.test.ts`
- Test: `internal/site/updates_test.go`

**Interfaces:**
- Produces:
  - `export interface CauseGroup { key: string; kind: string; id?: string; title?: string; path?: string; commit?: string; reason?: string; entries: UpdateEntry[]; files: number; isRule: boolean; legacy?: boolean }`
  - `export function groupByCause(rec: UpdateRecord): CauseGroup[]`
    - 一筆有多個原因時，出現在每個原因的組。
    - `retranslated` 依 `reason` 分組，key 為 `retranslated:<reason>`，沒有 reason 時為 `retranslated:`。
  - **排序**：規則組（`isRule`）在前，依 entries 數由多到少；其餘依 `kr-changed`、`new`、`deleted`、`reuse`、`late`、`retranslated`、`other`、`unknown` 的順序。
  - **舊資料**：沒有 `causes` 時，用舊的 `cause` 或 kind 分組，每組標 `legacy: true`。
  - `export const CAUSE_KIND_NAMES: Record<string, string>`，中文名稱：
    - 改寫檔、人工調整、人工調整失效、重翻申請、沿用同文單位、KR 改版、新增、刪除、補翻、重翻、其他、無法判定
- [ ] **Step 1：寫失敗的測試**
  - `causes.test.ts`：
    - 多原因的筆出現在兩組；
    - 排序正確，規則在前；
    - `retranslated` 依 reason 分組；
    - 舊資料 fallback。
  - `internal/site/updates_test.go`：
    - 有 `causes` 的紀錄，匯出的 `updates/<tag>.json` 含 `causes` 與 `rules`。
- [ ] **Step 2–4：先看它失敗，實作後看它通過**（`go test ./internal/site/`；`cd site && npm test -- causes`）
- [ ] **Step 5：Commit**

### Task 6: 網站「依原因」檢視、摺疊、標籤與篩選

**Files:**
- Modify: `site/src/updates/UpdateView.tsx`（檢視切換加 `causes`）、`site/src/settings.ts`（`updateView` 加 `'causes'`）
- Create: `site/src/updates/CauseView.tsx`
- Modify: `site/src/updates/UpdateCard.tsx`、`UpdateTable.tsx`（原因標籤）、`site/src/updates/filter.ts`（原因篩選）、`site/src/styles.css`
- Test: `site/src/updates/UpdateView.test.tsx`、`site/src/updates/filter.test.ts`

**Interfaces:**
- Consumes: Task 5 的 `groupByCause`、`CAUSE_KIND_NAMES`。
- 行為照 spec「網站」：
  - **組頭**：種類、id、title、單位數、檔案數、路徑與 commit 前 7 碼。
    - 寫計畫時查過：`pages.yml` 與 `meta.json` 都沒有 repo 網址，所以只顯示文字，不做連結，也不為此新增設定。
  - **多原因**：一筆有多個原因時，在每一組的卡片上列出它也屬於的其他組的標籤。
  - **摺疊**：
    - 每組預設收合；
    - 頂端有「全部展開／全部收合」；
    - 展開後沿用 `Cards` 的前後對照。
  - **標籤**：卡片與表格每筆加原因標籤；點了切到「依原因」，並帶網址參數 `g=<group key>`，該組自動展開並捲過去。
  - **篩選**：加一項「原因」，選項為出現的原因種類。
  - **400px**：不得整頁橫向捲動。
- [ ] **Step 1：寫失敗的測試**（`UpdateView.test.tsx`）
  - 切到「依原因」時，各組預設收合，只見組頭；
  - 點組頭會展開；
  - 「全部展開」把每組都展開；
  - 網址參數 `g=rewrite:x` 開啟時，該組已展開；
  - 舊資料沒有 `causes` 時照常顯示，並標「尚未補上原因」；
  - 原因篩選只留下所選種類。
- [ ] **Step 2–4：先看它失敗，實作後看它通過**（`cd site && npm test && npm run typecheck && npm run build`）
- [ ] **Step 5：Commit**

### Task 7: 真實資料驗收與文件

**Files:** Modify `README.md`（更新頁說明）、`docs/` 必要處；報告寫在 scratch，不進 repo。

- [ ] **Step 1：補做 10 個 Release**
  - 在 scratch clone 執行 `git fetch --tags`；
  - 對 `zh-TW-20261029.2`…`zh-TW-20261105.3` 逐一執行 `bin/lct history make <tag> --json`；
  - 列出每個 Release 各原因的筆數。
- [ ] **Step 2：對照標準**（spec「驗收」）
  - 每個改寫檔推斷出的單位數，要和套用它那次建置的 `.meta/plan.json` 中 `rewrite_applied[].units` 相符。從 tw 的 git 歷史找出套用該檔的 commit。不相符的要列出並說明原因。
  - 調整筆數要等於新紀錄 `Batch` 為 `adjustment:`、而舊紀錄不是的筆數。
  - 列出 `other`、「重翻（原因不明）」、`unknown` 的比例。
- [ ] **Step 3：網站**
  - 用補做後的紀錄跑 `lct site`，報告時間與資料大小，和 main 比較。
  - 用 Playwright 檢查 `#/updates/<tag>` 的「依原因」檢視：
    - 400px 與 1100px 都不得整頁橫向捲動；
    - 摺疊與深連結正常。
- [ ] **Step 4：文件**
  - README 的更新頁段落加上「依原因」與原因種類的說明；
  - README 寫明：合併後可手動觸發一次 rebuild（勾 `dry_run`）來補做舊紀錄。
- [ ] **Step 5：Commit 文件**

## 最後

最終整支審查（最強模型）→ 修正 Critical／Important，Minor 也修（使用者要求）→ 回報，附各 Release 的原因統計 → 使用者同意後才合併。合併後提醒使用者可手動觸發一次 dry_run 的 rebuild。
