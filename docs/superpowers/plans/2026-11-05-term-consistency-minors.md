# 術語統一：小問題收尾 實作計畫

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把術語統一分支（`2026-11-05-term-consistency`）各 Task 審查與最終審查記下、沒修的小問題全部處理完，然後合併到 `main`。

**Architecture:** 不加新功能，只修正與補測試。依程式區域分五個 Task：檢查與送出、批次、審核資料與 accept、網站、skill 與文件。後面的 Task 用到前面 Task 定下的介面（Task 3 的 JSON 給 Task 4、Task 3/4 的複製格式給 Task 5）。

**Tech Stack:** Go（`internal/validate`、`internal/batch`、`internal/store`、`internal/site`、`internal/suggest`、`cmd/lct`）、Preact + Vite（`site/`）、skill 與 README。

**Spec:** `docs/superpowers/specs/2026-11-05-term-consistency-design.md`（術語統一的設計；本計畫的每一項都是它的實作缺口或品質問題）

## Global Constraints

- `CLAUDE.md`：規則只在 `guides/`、`glossary/`、`rules/`；狀態只經由 `lct`；`raw`、`tw`、Release 只由 CI 寫入；所有 `lct` 指令支援 `--json`；skill 與 `prompts/` 不寫模型名稱。
- `term-missing` 只在 `batch.Submit` 且該單位 `note` 不含條目 `kr` 時是硬性；其他地方一律是警告。
- 逐單位 ID 是 `warning/term-missing/<sha8>/<sha8>`，由 `itemid.UnitWarning` 產生（第一段雜湊含 KR 原文、條目 KR 與條目譯名）；依詞分組的 ID 是 `termgroup/<sha8 KR>/<sha8 ZH>`，不得符合逐單位格式。其他警告代碼的 ID 不變。
- 新的批次檔與輸出檔欄位要同時加進 `schemas/batch.schema.json`、`schemas/batch-out.schema.json`。
- skill 的 `allowed-tools`、必備句與提示詞的釘選測試在 `cmd/lct/contract_test.go`，改 skill 時一起改。
- 提交前 `set -o pipefail; go vet ./... && go test ./...`；動到 `site/` 另跑 `cd site && npm test && npm run typecheck && npm run build`；不用 prettier 重排既有檔案。
- 不改真正的 `glossary/terms.yaml` 條目與 `rules/` 的紀錄；不碰 `/home/user/limbus-translate/.work/`。
- commit 訊息結尾附上工作階段指定的 `Co-Authored-By` 與 `Claude-Session` 兩行。
- 在 `claude/festive-wright-g2k853` 開發；不 push `main`、`tw`、`raw`（合併到 `main` 由 controller 在最後做）。

## Review Focus

1. **整組 ID 帶單位數（`termgroup/…@N`）與頁面不一致**：本地資料的未確認單位數不是 N 時，`lct accept` 拒絕並說明差多少，一筆也不寫。→ Task 3。
2. **重送批次時才登記的暫定詞**：送出只檢查批次檔詞彙表裡、而且仍生效的暫定詞；中途新登記的不會讓這批被退回。→ Task 1。
3. **已否決的暫定詞**：在審核頁顯示為已否決（像其他規則建議），不是消失。→ Task 3、Task 4。
4. **被標籤切開的 KR 詞**（`<b>와일드</b>헌트`）：仍算命中，`term-missing` 與 `forbidden-term` 都一樣。→ Task 1。
5. **暫定詞卡片在 400px**：`document.documentElement.scrollWidth <= 400`。→ Task 4。

---

### Task 1: 檢查與送出

**Files:**
- Modify: `internal/validate/validate.go`、`internal/rules/glossary.go`、`internal/batch/submit.go`、`internal/batch/batch.go`、`internal/store/terms.go`、`cmd/lct/batch.go`
- Test: `internal/validate/validate_test.go`、`internal/batch/*_test.go`、`cmd/lct/*_test.go`、`cmd/lct/contract_test.go`

**Interfaces:**
- Produces: `batch.Submit` 只用「批次檔詞彙表裡 `provisional: true`、而且目前仍在 `env.Provisional`」的暫定詞檢查；其餘不變。

每一項先寫會失敗的測試，再改：

- [ ] **1.1 一個單位缺兩個詞條**：`validate` 產生兩筆 `term-missing`（各自的 `Term`、`TermZH`）；測試不依賴 issues 的順序（依 `Term` 找）。
- [ ] **1.2 樣式無效的詞條**：`Term` 的比對樣式無效時的行為要有測試（照現有程式的實際行為寫斷言；若現有行為是 panic 或靜默略過，改成略過並在 `lct lint` 或既有的規則檢查報錯——先看 `rules` 的 lint 有沒有涵蓋，有就只補測試）。
- [ ] **1.3 譯名本身含類似標籤的字串**：比對前譯文與條目譯名都去掉標籤（同一個 `tagRe`），條目譯名含 `<…>` 時仍能命中。
- [ ] **1.4 KR 詞被標籤切開**：判斷條目是否命中 KR 時，用去掉標籤的 KR（`term-missing` 與 `forbidden-term` 都用同一個）；測試 KR 側被 `<b>` 切開的情況，也補 KR 側整個被標籤包住的情況。
- [ ] **1.5 note 的例外用同一套比對**：note 是否「寫了這個詞」改用條目自己的比對規則（`Term.OccursIn`：`match: word` 的條目要成詞），不再是純子字串；測試：條目 `속도`（word）時 note 寫「속도감」不算、寫「속도를」算。
- [ ] **1.6 兩個詞都缺、note 只寫一個**：送出時仍因另一個被退回，訊息只列沒寫的那個。
- [ ] **1.7 釘選句**：`cmd/lct/contract_test.go` 對譯者提示中「note 要寫出原詞」的釘選改成釘完整句子（現在太鬆）。
- [ ] **1.8 暫定詞去重與衝突的送出層級測試**：同一 KR、同一譯法由另一個單位再登記 → 不新增；第三種譯法 → 記為衝突；都經由 `Submit`。
- [ ] **1.9 空白 `kr` 的略過理由**：只有空白的 `kr` 以「kr 是空的」略過，不是「太短」；測試理由文字。
- [ ] **1.10 「0 provisional terms kept」**：`submitReport.String` 只在結果檔有登記 `terms` 時才印這行。
- [ ] **1.11 schema 的 minLength**：確認 `lct` 自己以 rune 數檢查 `kr` 至少兩字（送出時），有就在 `schemas/batch-out.schema.json` 的 `kr` 說明補「以字元計」；沒有就在送出時補檢查與測試。
- [ ] **1.12 送出時的暫定詞來源**：照 Interfaces 的規則；測試：批次發出後才登記的暫定詞不讓這批被退回；批次發出後被否決或被詞彙表取代的暫定詞也不再檢查。
- [ ] **1.13 「탐식의 재림」**：補送出層級測試（劇情內的組合名稱、note 寫了原詞 → 收下），與既有「시즌」的測試同一形式。
- [ ] **1.14 跑完整檢查並 commit**（可分多個 commit）

---

### Task 2: 批次

**Files:**
- Modify: `internal/batch/next.go`、`cmd/lct/e2e_test.go`、`cmd/lct/e2e_phase2_test.go`（視需要）
- Test: `internal/batch/*_test.go`、`cmd/lct/e2e*_test.go`

- [ ] **2.1 詞彙表與暫定詞重疊**：`Next` 層級測試：同一批文字同時命中詞彙表「와일드헌트 히스클리프」與暫定「와일드헌트」時，批次詞彙表兩條都有。
- [ ] **2.2 多餘的去重**：`glossaryFor` 對詞彙表 KR 的去重若在 `ActiveProvisional` 之後已不可能觸發，就刪掉並在註解說明；若仍可能觸發，補測試。
- [ ] **2.3 `ActiveProvisional` 的說明文字**：照實際行為改寫（第一筆非衝突、KR 至少兩字、詞彙表有同 KR 或已否決時不生效）。
- [ ] **2.4 schema 測試**：`TestBatchSchemaGlossaryMatchesTheTerm` 併進 `TestSchemasMatchTheTypes`（或讓後者涵蓋 glossary 的欄位），不留兩套。
- [ ] **2.5 合批時雙胞胎已在別批**：合批後面的檔有一個單位的雙胞胎已租給另一批 → 這個單位不進這批、其他照常；補測試。
- [ ] **2.6 `krs` 重複**：`next.go` 收集 KR 文字時不重複加入同一文字。
- [ ] **2.7 `take` 的參數名**：不要遮蔽內建的 `max`。
- [ ] **2.8 e2e 的合批假設**：`merged == mergeable` 的檢查改成直接算出「同族群、非劇情、合計不超過上限的檔」應該合批，而不是假設 fixture 沒有可合批的檔。
- [ ] **2.9 e2e 的 fixture 文字範圍**：`dropFixtureTerms` 掃過 replay 會用到的所有 KR 文字（各快照、各語言包裡的 KR），不只 kr 包與 `newSpeakerKR`。
- [ ] **2.10 跑完整檢查並 commit**

---

### Task 3: 審核資料與 accept

**Files:**
- Modify: `internal/site/review.go`、`internal/site/export.go`、`internal/suggest/suggest.go`、`cmd/lct/accept.go`、`cmd/lct/site.go`（或 `loadReviewInputs` 所在檔）、`internal/requeue/requeue.go`（視 3.11 結果）
- Test: `internal/site/*_test.go`、`internal/suggest/*_test.go`、`cmd/lct/accept_test.go`

**Interfaces:**
- Produces（給 Task 4、5）：
  - `term-new` 的 `SuggestionItem` 加 `term: {kr, zh}`（同 `TermRef`）。
  - 已否決的 `term-new` 照其他種類一樣列出，帶 `declined`。
  - `term-missing` 群組項目的 `message` 是給人看的說明（不是譯者指示），例如「這些譯文沒有用詞彙表的譯名「<zh>」（<kr>）。」
  - `lct accept` 接受 `termgroup/<a>/<b>@<N>`：展開前比對本地未確認單位數，不等於 N 就拒絕（使用者錯誤，exit 1），訊息寫出頁面上的 N 與現在的數目，一筆也不寫；不帶 `@N` 照舊。`--dry-run` 也接受並顯示比對結果。

- [ ] **3.1 暫定詞證據**：證據列總數（登記、衝突、命中）不超過 `suggest.MaxEvidence`；命中列不含照抄（copy）單位與尚未翻譯的單位；登記的單位不在命中列重複出現。
- [ ] **3.2 `term` 欄位**：如 Interfaces。
- [ ] **3.3 已否決的暫定詞**：如 Interfaces；測試否決後仍列出且有 `declined`、不計入待決定的數量（與其他種類一致）。
- [ ] **3.4 `ReviewInputs.Provisional` 的測試**：`internal/site` 測試暫定詞經由 `ReviewInputs.Provisional` 出現在 `BuildReview` 的結果。
- [ ] **3.5 規則只讀一次**：`loadReviewInputs` 不重複載入規則。
- [ ] **3.6 `term-new` 的順序**：與其他規則建議同一套排序規則（看 `suggest.go` 的排序；若其他種類是依某個數量或檔案順序，`term-new` 照辦）。
- [ ] **3.7 空的條目譯名**：分組時條目譯名為空就不分組（保持原本逐句的警告），加防護與測試。
- [ ] **3.8 群組的 `accepted`**：全部單位都確認時，`accepted` 取日期最新的一筆；各單位理由不同時，`reason` 寫「<N> 個單位各自確認」。
- [ ] **3.9 群組的 `message`**：如 Interfaces。
- [ ] **3.10 `termgroup/…@N`**：如 Interfaces；測試相等時寫入、不等時拒絕且不寫、不帶 `@N` 照舊、dry-run 顯示。
- [ ] **3.11 `requeue` 少一個**：在 scratchpad 的隔離 clone（`git clone --shared /home/user/limbus-translate`，`.work/raw`、`.work/tw` 用 `origin/raw`、`origin/tw` 的 worktree）以 `bin/lct requeue --json` 對 raw-20261105 那 71 個 key 重跑，看 `UserTicket-R.json|/dataList/79/name` 算在哪一欄。在某一欄 → 不改程式，在報告寫明；不在任何一欄 → 修正計數並補測試。
- [ ] **3.12 小清理**：`TestAcceptExpandsATermGroup` 的無用變數刪掉；`acceptReport` 的 `empty` 欄位在 `lct accept` 的說明（usage）寫清楚意思。
- [ ] **3.13 跑完整檢查並 commit**

---

### Task 4: 網站

**Files:**
- Modify: `site/src/types.ts`、`site/src/review/WarningCard.tsx`、`site/src/review/SuggestionCard.tsx`、`site/src/review/marks.ts`、`site/src/review/copy.ts`、`site/src/styles.css`
- Test: `site/src/review/*.test.ts(x)`

**Interfaces:**
- Consumes: Task 3 的 `term`、`declined`、`message`、`@N`。

- [ ] **4.1 `termMark('')`**：空字串不產生標記（回傳 undefined 或不比對任何東西）；測試。
- [ ] **4.2 暫定詞卡片的標記**：改用 `item.term`，不再從 `evidence[0]` 推。
- [ ] **4.3 群組標題的數目**：標題「詞彙表：<kr> → <zh>（<未確認數> 處）」，有已確認的單位時接「，<M> 處已確認」；測試兩種。
- [ ] **4.4 複製整組 ID**：整組的複製文字用 `termgroup/<a>/<b>@<未確認數>`。
- [ ] **4.5 已否決的暫定詞**：與其他已否決的規則建議同樣顯示（沿用「顯示已確認」與既有的已否決樣式）。
- [ ] **4.6 群組說明**：卡片顯示 Task 3 的 `message`。
- [ ] **4.7 400px 驗收**：在 scratchpad 的隔離 clone 用真實資料加一份測試用的 `.meta/terms.jsonl`（至少一筆暫定詞、一筆衝突；只在 clone 裡，不 commit）產生網站資料，Playwright 量 `#/r/suggestions` 與 `#/r/warnings` 在 400px 的 `document.documentElement.scrollWidth <= 400`，各截一張暫定詞卡片與群組卡片（1280、400），截圖放 scratchpad。
- [ ] **4.8 跑網站與 Go 的完整檢查並 commit**

---

### Task 5: skill 與文件

**Files:**
- Modify: `.claude/skills/review-page/SKILL.md`、`.claude/skills/term-decision/SKILL.md`、`README.md`、`cmd/lct/contract_test.go`

- [ ] **5.1 `term-decision` 的權限**：`allowed-tools` 的 `Bash(bin/lct accept *)` 縮到只能 `--json --dry-run` 與 `--json --reason`（不允許 `--tidy`）；內文寫明只用於 `term-new/…`；釘選測試同步。
- [ ] **5.2 單引號**：`review-page` 與 `term-decision` 寫明 ID 或理由含 `'` 時寫成 `'\''`；釘選句。
- [ ] **5.3 `@N`**：`review-page` 說明整組 ID 帶單位數、數目不符時 `lct accept` 會拒絕，該怎麼跟人說（請人重新整理頁面或改貼逐單位 ID）；釘選句。
- [ ] **5.4 README 暫定詞一段**：補上整詞比對、KR 至少兩字、不限檔案、對既有譯文不產生警告、CI 的 `/rules-review` 不處理。
- [ ] **5.5 README 網站一段**：把很長的一段拆成條列，內容不變，另補 `@N` 與已否決暫定詞的顯示。
- [ ] **5.6 跑完整檢查並 commit**
