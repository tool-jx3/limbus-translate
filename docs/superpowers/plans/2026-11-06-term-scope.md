# 詞彙表條目的比對與適用範圍 實作計畫

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 詞彙表條目預設只比對單獨成詞的用法（A），並能以檔名與路徑樣式排除條目不適用的檔（C），讓 `term-missing` 不再誤判詞中詞與劇情裡的一般用法。

**Architecture:** `rules.Term` 的 `Match` 預設改為整詞，新增 `match: substring` 讓條目回到字串包含；新增 `not_files`（樣式寫法同 `files`）排除檔案。所有命中判斷都經由 `Term.OccursIn` 與 `Rules.GlossaryHits`，所以批次詞彙表、送出檢查、建置與審查頁的警告、禁用詞檢查一起改變。哪些條目要標 `match: substring`、哪些要加 `not_files`，先以真實資料統計列出候選，給使用者在網頁上看過再定，最後才改 `glossary/terms.yaml`。

**Tech Stack:** Go（`internal/rules`、`internal/validate`、`internal/batch`、`internal/site`）、`schemas/`、`glossary/README.md`、`README.md`。

**Spec:** 使用者 2026-11-06 的決定：「A+C ok；用檔名、路徑等等來限制 term 可應用的範圍，像是可以限制哪些可以用在 StoryData/ 哪些不行等等」。A＝預設整詞比對，真的要在複合詞裡命中的條目個別標 `match: substring`；C＝條目可排除檔案，套用在哪些條目由使用者看過清單後決定。

## Global Constraints

- `CLAUDE.md`：規則只在 `guides/`、`glossary/`、`rules/`；狀態只經由 `lct`；`raw`、`tw`、Release 只由 CI 寫入；所有 `lct` 指令支援 `--json`；skill 與 `prompts/` 不寫模型名稱。
- 欄位名稱：`match` 的值只有 `word`（預設，可省略）與 `substring`；排除檔案用 `not_files`，樣式寫法與 `files` 相同（`config.MatchAny`：`path.Match` 的 glob，或以 `/**` 結尾表示目錄）。
- 條目適用於一個檔 ⇔（沒有 `files` 或 `files` 有一個樣式符合）且 `not_files` 沒有樣式符合。`files` 的既有優先規則不變：某檔符合同一個 `kr` 的有 `files` 條目時，沒設 `files` 的那一條在該檔不適用。`not_files` 只讓條目在該檔不適用，不影響這個優先規則。
- 暫定詞（`batch.ActiveProvisional`）照舊是整詞比對，不受 `match: substring` 影響。
- Task 1 只改程式與文件，不改 `glossary/terms.yaml` 的條目；Task 3 才依使用者的決定改條目。
- 提交前 `set -o pipefail; go vet ./... && go test ./...`；動到 `site/` 另跑 `cd site && npm test && npm run typecheck && npm run build`。
- commit 訊息結尾加：
  ```
  Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
  Claude-Session: https://claude.ai/code/session_015XrjRvHe6FfxjemSS1js35
  ```
- 在 `claude/festive-wright-g2k853` 開發；不 push `main`、`tw`、`raw`；真實資料只在 scratchpad 的隔離 clone 用，clone 的 `.work/raw`、`.work/tw` 取自來源 repo 的 `origin/raw`、`origin/tw`（`git fetch /home/user/limbus-translate refs/remotes/origin/raw:refs/heads/raw-latest refs/remotes/origin/tw:refs/heads/tw-latest`），不碰 `/home/user/limbus-translate/.work/`。

## Review Focus

1. **助詞後接的詞仍命中**：「속도로」「속도가」命中 속도；「약속도」「고속도로」「도시락」不命中。→ Task 1。
2. **`not_files` 與 `files` 同時設**：只在 `files` 符合且 `not_files` 不符合的檔適用。→ Task 1。
3. **同一個 `kr` 一條有 `not_files`、一條有 `files`**（例：神父 排除第十章，牧師 只在第十章）：每個檔恰好一條適用。→ Task 1。
4. **舊的 `match: word` 條目**：照舊整詞比對，lint 不報錯。→ Task 1。
5. **批次檔的詞彙表欄位**：`match` 為 `substring` 的條目在批次裡標出來，譯者看得到。→ Task 1。

---

### Task 1: 預設整詞比對、`match: substring`、`not_files`

**Files:**
- Modify: `internal/rules/glossary.go`、`internal/rules/lint.go`、`schemas/batch.schema.json`（batch 詞彙表的 `match` 與新的 `not_files` 說明，視批次是否帶 `files`）、`glossary/README.md`、`README.md`
- Test: `internal/rules/*_test.go`、`internal/batch/*_test.go`、`internal/validate/*_test.go`、`cmd/lct/contract_test.go`（若釘了相關文字）

**Interfaces:**
- Produces：`rules.MatchSubstring = "substring"`；`rules.Term.NotFiles []string`（yaml `not_files`）；`Term.OccursIn` 預設整詞、`Match == MatchSubstring` 時字串包含；`Term.AppliesTo(file string) bool`（或同等的單一函式）供 `GlossaryHits` 與其他判斷檔案範圍的地方共用。

- [ ] **1.1** 測試先行：`OccursIn` 預設整詞（「빠른 속도로」命中 속도；「약속도」「고속도로」不命中），`match: substring` 時「정사무소」命中 사무소。
- [ ] **1.2** 測試先行：`not_files` 讓條目在符合的檔不適用；與 `files` 同設時兩者都要滿足；Review Focus 3 的兩條互補條目每檔恰好一條適用。
- [ ] **1.3** lint：`match` 只接受 `word`、`substring`；`not_files` 的樣式用 `config.ValidPattern` 檢查；一個條目的 `files` 與 `not_files` 讓它在任何檔都不適用時報錯（例如兩者寫了同一個樣式）。
- [ ] **1.4** 找出所有自己判斷「條目是否適用於這個檔」的地方（`GlossaryHits` 以外，例如 `rules` 的 `files` 優先判斷、`site` 或 `batch` 裡的檢查），全部改用同一個函式。
- [ ] **1.5** schema 與批次檔：批次詞彙表帶 `match` 時要能表示 `substring`；`schemas/batch.schema.json` 的說明更新；`TestSchemasMatchTheTypes` 照舊通過。
- [ ] **1.6** 文件：`glossary/README.md` 的「`match: word`」一節改寫成「比對方式」：預設整詞（後接助詞也算），`match: substring` 才比對長詞裡的用法；新增「`not_files`：排除檔案」一節，附 `StoryData/**` 的例子；`README.md` 相關句子同步。
- [ ] **1.7** 在 scratchpad 的隔離 clone 用真實資料量 `lct site` 的詞彙群組數與單位數（之前 200 組、3098 個單位），寫進報告。
- [ ] **1.8** 跑完整檢查並 commit。

---

### Task 2: 候選清單（只做分析，不改 repo）

在 scratchpad 的隔離 clone 用真實資料，以 Task 1 之前的程式（現行 HEAD）加一支只放在 clone 裡、不 commit 的 Go 程式統計每個 `status: approved` 條目：

- 對條目適用（依現行 `files`）的每個已翻譯單位（origin 是 paratranz、claude、human），KR 以 `validate.MatchText` 處理後：
  - **詞中詞**：字串包含但不單獨成詞（`standsAsWord` 為假）的單位數 `n_c`，其中譯文（`MatchText` 後）含條目譯名的 `u_c`。
  - **單獨成詞**：依 `guides/routing.yaml` 的族群分成敘事類（`story-main`、`story-interlude`、`identity`、`ego-voice`、`announcer`、`dante-note`）與其他；各自的單位數與用了譯名的數目；另列每個族群的數字，以及 `AbEvents*`、`AbDlg_*` 單獨的數字。
- 輸出 `scratchpad/scope/report.json`：
  - `substring_candidates`：`n_c >= 3` 且 `u_c / n_c >= 0.8` 的條目（另列 `n_c >= 1` 的全部），附至多 5 個例子（key、包含它的長詞、KR 片段、TW 片段）。
  - `story_candidates`：敘事類單獨成詞 `>= 5` 個單位、其他檔用譯名比例 `>= 0.8`、敘事類用譯名比例 `<= 0.5` 的條目，附各族群數字與至多 5 個敘事類例子；另列敘事類單位數最多的前 50 個條目（不論比例）。
  - 每個條目附 `kr`、`zh`、`files`、現行 `match`、`notes`。
- 報告寫明門檻、樣本數與方法。

---

### Task 3: 依使用者的決定改詞彙表

- [ ] **3.1** controller 把 Task 2 的候選做成網頁給使用者看；使用者決定哪些條目標 `match: substring`、哪些加 `not_files`（以及樣式）。
- [ ] **3.2** 依決定改 `glossary/terms.yaml`（每條在 `notes` 註明日期與理由），`lct rules lint` 通過。
- [ ] **3.3** 隔離 clone 量真實資料：詞彙群組數、單位數、속도 一組與 `AbEvents-a1c5p3.json|/dataList/4/successDesc/0` 是否還在。
- [ ] **3.4** 跑完整檢查並 commit。
