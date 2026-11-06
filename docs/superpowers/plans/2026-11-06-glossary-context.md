# 詞彙表依語境套用與「詞彙範圍」頁 實作計畫

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 讓詞彙表條目能依族群設範圍、列多個正確譯法、長詞條優先、劇情類只警告，並在網站審核頁新增「詞彙範圍」分頁，每次更新都重算各條目的使用情況與建議。

**Architecture:** 比對與範圍判斷集中在 `internal/rules`（`family:<id>`、`also`、長詞條優先都在 `GlossaryHits` 一處生效）；送出時的硬性與否看路由族群的 `glossary: warn`；新套件 `internal/scope` 掃全部已翻譯單位算出每個條目的使用情況與建議，由 `lct site` 匯出 `review/scope.json`、`lct glossary scope --json` 輸出給 skill；網站新增 `#/r/scope` 分頁，決定照審核頁的流程經 `/review-page` 寫回 `glossary/terms.yaml` 或 `rules/declined.yaml`。

**Tech Stack:** Go（`internal/rules`、`internal/validate`、`internal/batch`、`internal/scope`（新）、`internal/site`、`internal/itemid`、`internal/suggest`、`cmd/lct`）、Preact + Vite（`site/`）、`guides/routing.yaml`、`glossary/`、`prompts/translate.md`、`.claude/skills/review-page`。

**Spec:** `docs/superpowers/specs/2026-11-06-glossary-context-design.md`

## Global Constraints

- `CLAUDE.md`：規則只在 `guides/`、`glossary/`、`rules/`；狀態只經由 `lct`；`raw`、`tw`、Release 只由 CI 寫入；所有 `lct` 指令支援 `--json`；skill 與 `prompts/` 不寫模型名稱。
- `files`、`not_files` 的每一項是檔名樣式（`config.MatchAny`）或 `family:<id>`（`rules.Family(file) == id`，路由第一個符合）。
- `also`：譯文（`validate.MatchText` 後）含 `zh` 或任一 `also` 即算有用到；`also` 與 `forbidden` 不得重疊；群組 ID 與逐單位 ID 照舊以 `zh` 計。
- 長詞條優先：只有命中範圍被另一個適用條目的命中範圍嚴格包住時，那一處不算；條目只要還有一處沒被包住就算命中；暫定詞與詞彙表條目一起比較。
- `guides/routing.yaml` 族群欄位 `glossary`：`hard`（預設）或 `warn`；`warn` 族群的檔，送出時 `term-missing` 不升為硬性；`forbidden-term` 照舊硬性。初值 `warn`：`story-main`、`story-interlude`、`identity`、`ego-voice`、`announcer`、`dante-note`。
- 建議 ID：`scope/<sha8 KR>/<sha8 ZH>/<種類>`，種類 `not-files`、`files`、`substring`、`word`、`mixed`、`explained`；條目 ID：`glossary/<sha8 KR>/<sha8 ZH>`（只供附指示，不計待決定、不進 `declined.yaml`）。
- 門檻：群組 n ≥ 3；有用 ≥ 0.8；沒用 ≤ 0.3；詞中詞 n ≥ 3 且 ≥ 0.8 建議 `substring`，條目是 `substring` 且 ≤ 0.3 建議 `word`；有說明 ≥ 3 為 `explained`；失去檢查 ≥ 20 或 ≥ 10% 標記且預設不採用。
- 目錄群組：`StoryData/**`、`PersonalityVoiceDlg/**`、`EGOVoiceDig/**`、`BattleAnnouncerDlg/**`、`RPGSystem/**`、其餘。建議只用 `family:<id>` 或上列目錄樣式。
- 這次就加的 `also`：`대상 → 目標` 加 `對象`；`보스 → BOSS` 加 `老大`、`首領`，`notes` 補上何時用哪個。
- 新的批次檔欄位同時改 `schemas/batch.schema.json`；skill 文字與 `cmd/lct/contract_test.go` 一起改。
- 提交前 `set -o pipefail; go vet ./... && go test ./...`；動到 `site/` 另跑 `cd site && npm test && npm run typecheck && npm run build`；動到 `glossary/`、`guides/` 跑 `bin/lct rules lint`。
- commit 訊息結尾加 harness 給的署名，另加 `Claude-Session: https://claude.ai/code/session_015XrjRvHe6FfxjemSS1js35`。
- 在 `claude/festive-wright-g2k853` 開發；不 push `main`、`tw`、`raw`；真實資料只在 scratchpad 的隔離 clone（`git fetch /home/user/limbus-translate refs/remotes/origin/raw:refs/heads/raw-latest refs/remotes/origin/tw:refs/heads/tw-latest` 後以這兩個分支建 `.work/raw`、`.work/tw`），不碰 `/home/user/limbus-translate/.work/`。

## Review Focus

1. **同一條目一處被包住、另一處沒有**（「조직의 보스… 보스 전투」）：BOSS 照舊命中這個單位。→ Task 3 `TestLongerEntryCoversOnlyItsOwnSpan`。
2. **部分重疊**（「가나다」與「나다라」在「가나다라」）：兩個都命中。→ Task 3 `TestPartialOverlapKeepsBoth`。
3. **`warn` 族群的檔寫了禁用譯法**：照舊退回；漏用譯名、沒寫 note：收下。→ Task 4 `TestSubmitWarnFamily`。
4. **建議的樣式比對不到檔**：`internal/scope` 只產生 `family:<id>` 與固定目錄樣式，且每個建議樣式在輸入的檔名裡至少符合一個。→ Task 5 `TestProposalPatternsMatchFiles`。
5. **未知的 `family:` id 或 `also` 與 `forbidden` 重疊**：`lct rules lint` 報錯。→ Task 1、Task 2 的 lint 測試。

---

### Task 1: `family:<id>` 範圍

**Files:**
- Modify: `internal/rules/glossary.go`、`internal/rules/lint.go`、`internal/rules/routing.go`（如需）、`glossary/terms.yaml`、`glossary/README.md`、`prompts/translate.md`、`schemas/batch.schema.json`（`files`／`not_files` 說明）
- Test: `internal/rules/rules_test.go`、`internal/batch/batch_test.go`、`cmd/lct/contract_test.go`

**Interfaces:**
- Produces：`func (r *Rules) Applies(t Term, file string) bool`（取代 `Term.AppliesTo`；`files`／`not_files` 每項為樣式或 `family:<id>`）；`GlossaryHits`、`ScriptRenderings`、`ValidateTerms`、`files` 優先規則全改用它。`const FamilyPrefix = "family:"`。

- [ ] **1.1** 測試先行：`family:story-main` 在 `files` 只適用路由判為 story-main 的檔；在 `not_files` 排除它們；與樣式混用；`family:` 也參與「有 files 的條目在它適用處取代通用條目」。
- [ ] **1.2** 實作 `Applies` 並改所有呼叫者（`grep -n "AppliesTo\|scopedTo" -r internal cmd`）。
- [ ] **1.3** lint：`family:` 後的 id 不在路由族群 → 錯誤；`LintScopeWarnings` 對 `family:<id>`（有 raw 時）在沒有任何檔屬於該族群時警告。測試先行。
- [ ] **1.4** 批次與譯者：批次詞彙表原樣帶 `family:<id>`；`prompts/translate.md` 說明「`family:<id>` 指批次 `family` 為 <id> 的單位」（批次的檔同屬一個族群）；schema 說明更新；釘選。
- [ ] **1.5** 資料：`관통`、`별빛`、`흐트러짐 구간`、`키워드`、`이방인`、`전투 패배` 由族群湊成的樣式改寫成 `family:<id>`（`필름` 對不上整個族群，維持樣式）；以真實資料確認每條的適用檔集合只有 `전투 패배` 少了 `Skills_Assist-walpu6.json`；`notes` 的檔名描述同步。
- [ ] **1.6** `glossary/README.md`：`files`、`not_files` 兩節加 `family:<id>` 的寫法與例子。
- [ ] **1.7** lint、完整檢查、commit。

---

### Task 2: `also`

**Files:**
- Modify: `internal/rules/glossary.go`（`Term.Also`，yaml `also`、json `also,omitempty`，`termFields`）、`internal/rules/lint.go`、`internal/validate/validate.go`（`validate.Term.Also`、`checkTerms`）、`schemas/batch.schema.json`、`prompts/translate.md`、`glossary/terms.yaml`、`glossary/README.md`
- Test: `internal/validate/validate_test.go`、`internal/rules/rules_test.go`、`internal/batch/batch_test.go`、`internal/site/review_test.go`、`cmd/lct/contract_test.go`

**Interfaces:**
- Produces：`rules.Term.Also []string`；`validate.Term.Also []string`（`ToValidate` 帶過去）；`func (t Term) Used(zh string) bool`（`MatchText(zh)` 含 `MatchText(ZH)` 或任一 `MatchText(also)`），供 site 與 scope 判「有用」。

- [ ] **2.1** 測試先行：`term-missing` 在譯文用了 `also` 時不出現；`also` 也去標籤後比對；送出時用 `also` 不需 note。
- [ ] **2.2** lint：`also` 含空字串、與 `zh` 相同、與 `forbidden` 任一項相同 → 錯誤。
- [ ] **2.3** 批次詞彙表帶 `also`；schema 加欄位與說明；`prompts/translate.md`：「依 `notes` 在 `zh` 與 `also` 之間選，選 `also` 不需要 note」；釘選。
- [ ] **2.4** 審核頁：詞彙群組不列用了 `also` 的單位；群組 ID 不變（測試）。
- [ ] **2.5** 資料：`대상 → 目標` 加 `also: [對象]`；`보스 → BOSS` 加 `also: [老大, 首領]`；兩條 `notes` 補「2026-11-06 加 also：…」說明何時用哪個。
- [ ] **2.6** 真實資料量這兩條的詞彙群組單位數前後變化，寫進報告。
- [ ] **2.7** lint、完整檢查、commit。

---

### Task 3: 長詞條優先

**Files:**
- Modify: `internal/rules/glossary.go`、`internal/batch/batch.go`（`glossaryFor`、`CheckNewTerms`）、`internal/site/review.go`（term-new 命中）
- Test: `internal/rules/rules_test.go`、`internal/batch/batch_test.go`

**Interfaces:**
- Produces：`func (t Term) Spans(kr string) [][2]int`（在 `MatchText` 後的文字中，依 `Match` 規則的所有命中位置，byte 範圍）；`func DropCovered(kr string, terms []Term) []Term`（去掉每一處命中都被另一個 term 的命中範圍嚴格包住的 term）。`GlossaryHits` 回傳前套用；`glossaryFor`、`CheckNewTerms` 對「詞彙表命中＋暫定詞命中」的聯集套用。

- [ ] **3.1** 測試先行：`TestLongerEntryCoversOnlyItsOwnSpan`（「조직의 보스」與「보스」同句兩處，一處被包、一處沒有 → 兩條都命中；只有被包的那處 → 只剩長的）、`TestPartialOverlapKeepsBoth`、同範圍兩條（不嚴格包住）都留、`substring` 條目的範圍也參與。
- [ ] **3.2** 實作 `Spans`、`DropCovered`，`GlossaryHits` 套用。
- [ ] **3.3** 暫定詞一起比較：批次詞彙表與送出檢查改用聯集後 `DropCovered`；更新原本「暫定詞與詞彙表互相包含時各自生效」的測試為新規則（暫定「와일드헌트」被詞彙表「와일드헌트 히스클리프」包住的那處不算）。
- [ ] **3.4** `glossary/README.md` 加「長詞條優先」一節；`prompts/translate.md` 加一句。
- [ ] **3.5** 真實資料量詞彙群組前後變化、`lct site` 耗時前後，寫進報告。
- [ ] **3.6** 完整檢查、commit。

---

### Task 4: 劇情類只警告

**Files:**
- Modify: `internal/rules/routing.go`（`FamilyRule.Glossary`，yaml `glossary`）、`internal/rules/lint.go`、`internal/batch/submit.go`、`internal/batch/batch.go`（`Batch.GlossaryCheck`，json `glossary_check,omitempty`）、`schemas/batch.schema.json`、`guides/routing.yaml`、`prompts/translate.md`、`.claude/skills/translate-pending/SKILL.md`（若提到詞彙表硬性）、`glossary/README.md`
- Test: `internal/rules/rules_test.go`、`internal/batch/batch_test.go`、`cmd/lct/contract_test.go`

**Interfaces:**
- Produces：`func (r *Rules) GlossaryWarnOnly(file string) bool`；批次欄位 `glossary_check: "warn"`（族群為 warn 時才有）。

- [ ] **4.1** 測試先行 `TestSubmitWarnFamily`：warn 族群的檔漏用譯名、沒 note → 收下（記錄為警告）；同檔用了 `forbidden` → 退回；hard 族群漏用 → 退回。
- [ ] **4.2** lint：`glossary` 只能是空、`hard`、`warn`。
- [ ] **4.3** 實作；批次帶 `glossary_check`；schema；`prompts/translate.md`：warn 時詞彙表譯名是預設、語境不合可不用，不用時仍在 note 寫原因；釘選。
- [ ] **4.4** `guides/routing.yaml` 標六個族群 `glossary: warn`，檔頭說明這個欄位。
- [ ] **4.5** 完整檢查、commit。

---

### Task 5: `internal/scope` 分析、匯出與 `lct glossary scope`

**Files:**
- Create: `internal/scope/scope.go`、`internal/scope/scope_test.go`
- Modify: `internal/itemid/itemid.go`（`Scope`、`GlossaryEntry`）、`internal/suggest/accepted.go` 或 declined 的 lint（接受 `scope/…`）、`internal/site/export.go`（`review/scope.json`、計數）、`cmd/lct/site.go`、`cmd/lct/glossary.go`（`scope` 子指令）、`cmd/lct/accept.go`（`scope/…` 的不採用與 `--dry-run`）
- Test: `internal/scope/scope_test.go`、`internal/site/export_test.go`、`cmd/lct/accept_test.go`、`cmd/lct/glossary_test.go`

**Interfaces:**
- Consumes：Task 1 `Applies`；Task 2 `Term.Used`；Task 3 `GlossaryHits`（已含長詞條優先）。
- Produces（JSON 欄位名稱即下列 snake_case）：
  - `scope.Inputs{Rules *rules.Rules; Units []Unit; Declined map[string]*suggest.Confirmed; Provisional []rules.Term /* 生效中的暫定詞，同審核頁的 inForce */}`；`scope.Unit{Key, File, KR, ZH, Origin, Note string}`。
  - `scope.Report{Thresholds; Entries []Entry; Proposals []Proposal}`。
  - `Entry{ID, KR, ZH string; Also, Files, NotFiles []string; Match, Notes string; Groups []Group; Compound Count; Explained, Unexplained int}`；`Group{Name, Kind string /* "family" | "dir" */; N, Used int}`；`Count{N, Used int}`。
  - `Proposal{ID, Kind, KR, ZH string; Patterns []string; Match string; MixedGroups []string; WarnRemoved, WarnAdded, UsedLost, UsedGained int; Flagged bool; Examples []Example; Declined *suggest.Confirmed}`；`Example{Key, Group, KR, ZH string}`。JSON：`{"id","kind","kr","zh","patterns","match","mixed_groups","warn_removed","warn_added","used_lost","used_gained","flagged","examples","declined"?}`；`mixed_groups` 只在 `not-files`、`files`、`mixed` 有值（混用群組名；`mixed` 為空表示各群組都很少用這個譯名），其餘為 `[]`；效果是改了該條目後對所有條目重算的前後差；`flagged` 看 `used_lost`。
  - 群組名：族群 `family:<id>`、目錄 `StoryData/**` 等、其餘 `rest`；`Example.Group` 與 `Proposal.Patterns` 用同一套名稱。`match` 只有 `substring`、`word` 兩種建議有值，其餘為 `""`。
  - `func Analyze(in Inputs) Report`。
  - `itemid.Scope(kr, zh, kind string) string`、`itemid.GlossaryEntry(kr, zh string) string`。
  - `lct glossary scope --json` 輸出 `Report`；`lct site` 寫 `review/scope.json`（同 `Report`）。

- [ ] **5.1** 測試先行（以小型 fixture）：各種類各一個建議（not-files、files、substring、word、mixed、explained）；效果數字；flagged；已否決的建議帶 `declined`；`TestProposalPatternsMatchFiles`（每個建議樣式在輸入檔名中至少符合一個）；只計 origin `claude`、`human`、`paratranz`。
- [ ] **5.2** 實作 `Analyze`：一次掃過單位；有用以 `Term.Used`；有說明以條目的 `OccursIn` 比對 note；例子每個「沒用」群組至多 3 個、片段至多 60 字。
- [ ] **5.3** ID 與 declined：`rules/declined.yaml` 的 lint 接受 `scope/…`；`lct accept` 能不採用 `scope/…`、`--dry-run` 認得它；`glossary/…` 給 `lct accept` 時回「這是條目 ID，請附指示由 skill 改詞彙表」並 exit 1。
- [ ] **5.4** 匯出：`lct site` 寫 `review/scope.json`，審核頁計數加 `scope`（待決定 = 未否決的建議數）；`lct glossary scope --json`。
- [ ] **5.5** 真實資料：`lct site` 在隔離 clone 跑，記下建議各種類數、條目數、耗時。
- [ ] **5.6** 完整檢查、commit。

---

### Task 6: 網站「詞彙範圍」分頁

**Files:**
- Create: `site/src/review/ScopeView.tsx`、`site/src/review/ScopeView.test.tsx`
- Modify: `site/src/types.ts`、`site/src/data.ts`、`site/src/route.ts`、`site/src/review/ReviewView.tsx`、`site/src/review/copy.ts`（若需）、`site/src/styles.css`
- Test: `site/src/review/ScopeView.test.tsx`、`site/src/route.test.ts`

**Interfaces:**
- Consumes：Task 5 的 `review/scope.json`。

- [ ] **6.1** 測試先行：路由 `#/r/scope`；分區（排除檔案、限定檔案、詞中詞、需要人看、常被說明偏離）各自列出對應建議；卡片顯示樣式、效果、群組表（出現、用了譯名、比例、判斷）、例子；flagged 標示；已否決的建議預設隱藏、「顯示已確認」時顯示；「全部條目」可依 KR／譯名搜尋並展開群組表與 note 例外數；複製文字是 `ID「摘要」→ `，條目列複製 `glossary/…` ID。
- [ ] **6.2** 實作，沿用審核頁的卡片、IdCopy、ConfirmedLine、篩選元件與樣式。
- [ ] **6.3** 完整網站檢查、commit。

---

### Task 7: `/review-page` 與文件

**Files:**
- Modify: `.claude/skills/review-page/SKILL.md`、`README.md`、`glossary/README.md`、`cmd/lct/contract_test.go`

- [ ] **7.1** 釘選先行：skill 說明 `scope/…`（採用 → 讀 `bin/lct glossary scope --json` 找到建議、改 `glossary/terms.yaml` 的欄位並在 `notes` 記日期與數字、`bin/lct rules lint`；不採用或維持現狀 → `bin/lct accept`）與 `glossary/…`（照人附的指示改條目；`also` 只在人明確指示時加）；`allowed-tools` 加 `Bash(bin/lct glossary scope *)`。
- [ ] **7.2** README：網站一節加「詞彙範圍」分頁；詞彙表一節加 `family:`、`also`、長詞條優先、劇情類只警告。
- [ ] **7.3** 完整檢查、commit。

---

### Task 8: 真實資料驗收

- [ ] **8.1** 隔離 clone（見 Global Constraints），`bin/lct build` 若需要，`bin/lct site --out site/public/data`。
- [ ] **8.2** 記錄：詞彙群組數與單位數（前一階段合併時 193 組、2481 單位）、대상 與 보스 兩組的變化、長詞條優先造成的變化、`review/scope.json` 的建議各種類數與條目數、`lct site` 耗時。
- [ ] **8.3** 建網站，Playwright 截 `#/r/scope`（1280px、400px），400px 時 `document.documentElement.scrollWidth <= 400`；截圖放 scratchpad。
- [ ] **8.4** `bin/lct rules lint`（有 raw）沒有警告；完整檢查。
