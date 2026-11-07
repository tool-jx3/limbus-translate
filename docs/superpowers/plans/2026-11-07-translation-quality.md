# 翻譯品質 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 指引依分群分檔組合、詞彙補 JP/EN/詞義（附出處）、依分群回顧應用率未滿 100% 的詞條，並提供舊譯複查（`lct audit`）與人工基準樣本（`lct gold`、網站 `#/gold`）。

**Architecture:** 程式（Task 1–7）只提供組合、檢查、證據與工具；規則內容（Task 8–11）由代理人依真實資料與出處寫進 `guides/`、`glossary/`、`rules/`，使用者合併前審閱。不提交重翻申請、不花翻譯費用。

**Tech Stack:** Go 1.26（`cmd/lct`、`internal/`），Preact + Vite + Vitest（`site/`）。

**Spec:** `docs/superpowers/specs/2026-11-06-translation-quality-design.md`

## Global Constraints

- 規則只寫在 `guides/`、`glossary/`、`rules/`；程式只組合與檢查（CLAUDE.md）。狀態只經 `lct` 改。
- 不編造：每條新寫的指引規則、`meaning`、`jp`、`en` 都要有出處——遊戲檔案的 key（`game:<檔名>#<pointer>`）、`https://limbuscompany.wiki.gg/…` 網址，或現有譯文的統計（附單位數）；找不到就留空。namu.wiki、wikiwiki.jp 不用（會被擋）。
- 不花翻譯費用：不提交 `rules/requeue/` 的重翻申請；`lct audit --to-file` 只在測試中使用。
- 詞彙條目 `rules.Term` 的新欄位 `Meaning`、`Sources` 必須排在 `Provisional`、`Units` 之前（網站組回批次時把 `units` 接在條目最後；欄位順序錯會破壞與 `batch next` 逐字相同的契約）。
- 族群指引每份 ≤ 4096 bytes；每批指引組合（不含說話者角色）總量 ≤ 34 KB（35,000 bytes）。
- 對齊驗證門檻：候選須出現在 ≥60% 含此詞且有對齊的單位，且 ≥10 個單位。
- 範圍縮小門檻：某族群或 top 節點的缺漏 ≥5 單位且他義 ≥80%。
- 基準樣本每族群 20 個（使用者核可），固定種子。
- 提交前：`go vet ./... && go test ./...`；動到 `site/` 另跑 `cd site && npm test && npm run typecheck && npm run build`。
- Commit 訊息繁體中文，結尾附上工作階段指定的 `Co-Authored-By` 與 `Claude-Session` 兩行。程式、文件、skill 不寫模型名稱。
- 內容任務需要真實資料：用 scratch clone `/tmp/claude-0/-home-user/f7dbac2a-9a09-566e-ac65-036521a48b24/scratchpad/acc`（已有 `.work/raw`、`.work/tw`），在那裡 fetch 本分支、建 `bin/lct` 後執行分析；檔案修改寫在 `/home/user/limbus-translate`。不碰 `/home/user/limbus-translate/.work/`。wiki.gg 用 `curl -A "limbus-translate-glossary/1.0 (research)"` 讀 `https://limbuscompany.wiki.gg/api.php`。

## Review Focus

1. `Term` 欄位順序：`meaning`/`sources` 若排在 `units` 之後，預覽契約 fixture 會壞——Task 2 加測試：含 meaning 的條目經 `batch next` 寫出與網站組回逐字相同（在 `internal/site` 的 round-trip 測試世界加一個有 meaning 的條目）。
2. 指引組合的重複與順序：節點層與標記層同時符合、或與族群指引同檔時只列一次；角色指引仍在最後——Task 1 測試。
3. 同義／他義：沒有可比對的對齊（例如條目只有 en、單位沒有 EN 對齊），或 EN／JP 判斷不一致時是「未知」，不是他義——Task 4 測試（2026-11-07 Task 9c 改為 EN、JP 一致才判同義或他義）。
4. `lct gold sample` 不覆蓋已有判定；同種子同資料結果相同——Task 5 測試。
5. `lct audit` 不帶 `--to-file` 時不寫任何檔；排除待譯中、copy、凍結的單位——Task 7 測試。

---

### Task 1: 指引的節點層與標記層

**Files:** Modify `internal/rules/routing.go`、`internal/rules/rules.go`（`GuidesFor`）、`internal/rules/lint.go`；Test `internal/rules/*_test.go`；Modify `prompts/translate.md`、`guides/routing.yaml`（只加註解說明 `guides:` 語法，不加條目）。

**Interfaces — Produces:**
- `type GuideRule struct { When string \`yaml:"when" json:"when"\`; Guide string \`yaml:"guide" json:"guide"\` }`；`Routing.Guides []GuideRule \`yaml:"guides,omitempty" json:"guides,omitempty"\``。
- `When`：節點名稱或 `path.Match` 樣式（比對 `FileNode(file).Top` 或 `.Sub`，與 `nodes` 相同語法），或標記 `chapter:<n>`、`character:<id>`、`event:<id>`（比對 `FileTag(file)`）。
- `GuidesFor(file, speakers)` 順序：base → 族群指引 → 依 `Guides` 清單順序符合的節點層 → 符合的標記層 → `CharacterByFile` → 說話者角色；重複的只列第一次。
- Lint（錯誤）：`Guides` 的檔案不存在、`When` 語法錯誤；`guides/families/*.md` 任何一份 > 4096 bytes。Lint（警告，`LintRoutingWarnings` 有 raw 時）：任一 KR 檔 `GuidesFor(file, nil)` 的檔案大小總和 > 35,000 bytes。

- [ ] Step 1：失敗測試——`TestGuidesForLayers`（族群＋節點＋標記＋角色的順序、重複只列一次）、`TestLintGuideRules`（不存在的檔、壞樣式、超過 4096 bytes）、`TestLintGuideTotal`（總量超過時有警告）。
- [ ] Step 2：`go test ./internal/rules -run 'GuidesFor|LintGuide'` → FAIL。
- [ ] Step 3：實作；`prompts/translate.md` 在說明 `guides` 的段落加一句疊加順序。
- [ ] Step 4：同指令 → PASS；`go vet ./... && go test ./...` → PASS。
- [ ] Step 5：commit「規則：指引可依節點與標記疊加，族群指引大小與每批總量檢查」。

### Task 2: 詞彙條目的 meaning 與 sources

**Files:** Modify `internal/rules/glossary.go`（`Term`）、`internal/rules/lint.go`、`schemas/batch.schema.json`、`prompts/translate.md`、`glossary/README.md`、`internal/scope/scope.go`（`Entry` 帶 `JP`、`EN`、`Meaning`、`Sources`）、`site/src/types.ts`、`site/src/glossary/TermView.tsx`（顯示 JP、EN、詞義與出處連結）、`internal/site/preview_test.go` 的測試世界。

**Interfaces — Produces:**
- `Term.Meaning string \`yaml:"meaning" json:"meaning,omitempty"\``、`Term.Sources []string \`yaml:"sources" json:"sources,omitempty"\``，放在 `Check` 之後、`Provisional` 之前。
- Lint 錯誤：`meaning` 非空而 `sources` 空；`sources` 每項須符合 `^game:[^#\s]+#/\S*$` 或 `^https://limbuscompany\.wiki\.gg/\S+$`。
- `scope.Entry` 加 `JP, EN, Meaning string`、`Sources []string`（json `jp`、`en`、`meaning`、`sources`，omitempty）。

- [ ] Step 1：失敗測試——`TestLintMeaningNeedsSources`、`TestLintSourceForms`；`TestTermJSONOrder`（`json.Marshal` 含 meaning 與 units 的條目，`units` 是最後一個鍵）；round-trip 世界加一條有 meaning、sources 的條目，`TestExportPreviewRoundTrip` 與 `TestPreviewFixturesUpToDate`（`-update-preview` 重產生後 `batches/*` 舊案例不變、新內容只在含該條目的案例）照過；TS：`TermView` 顯示詞義與出處（wiki 網址為連結、`game:` 為文字）。
- [ ] Step 2：`go test ./internal/rules ./internal/site` 與 `cd site && npx vitest run src/glossary` → FAIL。
- [ ] Step 3：實作；`prompts/translate.md` 的詞彙段落加：條目的 `jp`、`en`、`meaning` 用來判斷單位裡的 KR 是否同一個意思（比對單位的 `jp`、`en`）；意思不同就不用 `zh`，並在 note 寫原詞與理由。`glossary/README.md` 說明兩個新欄位與出處格式。
- [ ] Step 4：Go 全套與 site 全套 → PASS。
- [ ] Step 5：commit（Go、site 各一）。

### Task 3: `lct glossary evidence`

**Files:** Create `internal/evidence/evidence.go`、`internal/evidence/evidence_test.go`、`cmd/lct/evidence.go`（註冊於 `glossary evidence`）、`cmd/lct/evidence_test.go`；Modify `README.md`（指令一行）。

**Interfaces — Produces:**
- `type Name struct { JP, EN string; Keys []string; Count int }`（KR 與單位整句相同的名稱欄位，對齊的 JP/EN 聚合）。
- `type Desc struct { Key, KR, JP, EN string }`（名稱命中位於 keyword 檔（`names.IsKeywordFile`）且欄位為 `name` 時，同一 dataList 元素的 `desc`）。
- `type Check struct { Lang, Candidate string; N, Hits int; Rate float64; OK bool }`（含此詞（`Term.OccursIn` 於 `validate.MatchText(kr)`）且該語言有對齊的單位數 N、對齊文字含候選的 Hits；EN 比對不分大小寫且以字詞邊界、JP 為子字串；OK = N ≥ 10 且 Rate ≥ 0.6）。
- `type Item struct { KR, ZH string; JP, EN string /* 條目現有 */; Names []Name; Descs []Desc; Checks []Check }`。
- `func Collect(r *rules.Rules, set pack.Set, krs []string, candidates map[string]Candidates) ([]Item, error)`；`type Candidates struct { JP, EN []string }`。`krs` 空表示全部 approved 條目；檢查的候選 = 條目現有 jp/en ＋ Names 中出現最多的 JP/EN ＋ `candidates` 給的。
- CLI：`lct glossary evidence [--kr K]... [--candidates FILE.yaml] [--json]`；只需 raw（`raw.LastSource`、`pack.OpenRaw`、`rules.Load`）。文字輸出每條一段：名稱表、說明、每個候選的 N/Hits/Rate/OK。

- [ ] Step 1：失敗測試——名稱表聚合（兩個檔同名不同 JP 時兩組）、keyword desc 對應、驗證門檻（9 個單位 100% → 不 OK；10 個 60% → OK；EN 大小寫與字詞邊界）、CLI `--json` 形狀與 `--kr` 篩選。
- [ ] Step 2：`go test ./internal/evidence ./cmd/lct -run Evidence` → FAIL。
- [ ] Step 3：實作。
- [ ] Step 4：PASS；全套 PASS。
- [ ] Step 5：commit「lct glossary evidence：詞條的官方名稱、說明與 JP/EN 對齊驗證」。

### Task 4: 範圍分析加入同義／他義

**Files:** Modify `internal/scope/scope.go`（`Unit` 加 `EN, JP string`；`Entry`、`Group` 加 `Same, Other, Unknown int`）、`internal/site/review.go`（建 `scope.Unit` 時帶入對齊的 EN/JP）、`cmd/lct/glossary.go`（`glossary scope` 文字輸出顯示三數）、`site/src/types.ts`、`site/src/glossary/TermView.tsx`（群組表加「同義／他義」兩欄）、`site/src/review/ScopeView.tsx`（條目列顯示）；Tests。

**Interfaces — Produces:** 對沒用譯名的單位（`!Used`），可比對的語言是條目有該值、且單位該語言的對齊非空白者（EN 不分大小寫、字詞邊界、最後一字容許單複數；JP 子字串）：可比對的都含 → Same；都不含 → Other；沒有可比對的，或 EN／JP 判斷不一致 → Unknown（2026-11-07 Task 9c 起；原本有 EN 就只看 EN）。json：`same`、`other`、`unknown`。現有提案邏輯不變。

- [ ] Step 1：失敗測試——三種分類（含「條目有 en、單位無 EN 對齊 → Unknown」）、群組加總、site 欄位顯示。
- [ ] Step 2：FAIL。Step 3：實作。Step 4：Go 與 site 全套 PASS。Step 5：commit。

### Task 5: 基準樣本 `lct gold`

**Files:** Create `internal/gold/gold.go`、`internal/gold/gold_test.go`、`cmd/lct/gold.go`、`cmd/lct/gold_test.go`；Modify `internal/site/export.go`（寫 `data/gold.json`）、`README.md`。

**Interfaces — Produces:**
- `rules/gold.yaml`：`{seed int, per_family int, units: [{key, family, origin, verdict, note, judged_zh, judged_at}]}`；`verdict` 為 `""`、`correct`、`wrong`、`unsure`。
- `lct gold sample [--per-family 20] [--seed 20261107] [--force]`：每族群從非 copy、有紀錄的單位中，依 `sha256(seed|key)` 排序取前 N；檔案已有任何判定時拒絕（使用者錯誤），`--force` 才覆蓋。
- `lct gold set <key> <correct|wrong|unsure> [--note TEXT]`：記判定、`judged_zh`（現有紀錄譯文）、`judged_at`（UTC 日期）；key 不在樣本 → 使用者錯誤。
- `lct gold report [--json]`：依族群與來源列 judged、correct、wrong、unsure、accuracy（correct/(correct+wrong)，無判定為 null）、stale（現有譯文 ≠ judged_zh）。
- `data/gold.json`：`{per_family, units:[{key, path, ptr, family, origin, kr, jp?, en?, zh, verdict, note, stale}], report}`；`rules/gold.yaml` 不存在時寫 `{units: []}`。

- [ ] Step 1：失敗測試——取樣決定性、每族群數量（不足 N 時全取）、拒絕覆蓋、set 寫入與錯誤、report 計算（含 stale、無判定）、export 形狀。
- [ ] Step 2–4：FAIL → 實作 → 全套 PASS。Step 5：commit。

### Task 6: 網站基準頁與 gold skill

**Files:** Create `site/src/gold/GoldView.tsx`、`site/src/gold/GoldView.test.tsx`、`.claude/skills/gold-review/SKILL.md`；Modify `site/src/route.ts`（`#/gold`）、`site/src/app/App.tsx`、`site/src/app/Header.tsx`（「基準」連結）、`site/src/data.ts`（`loadGold()`）、`site/src/types.ts`、`site/src/styles.css`、`cmd/lct/contract_test.go`（新 skill 的 allowed-tools）、`README.md`。

**Interfaces:** 頁面依族群列出樣本（KR/JP/EN/TW、來源、現有判定、stale 標示），每個單位選「正確／有誤／不確定」與說明（存瀏覽器 localStorage，以 key 為索引，try/catch 包住），「複製判定」產生每行 `gold <key> <correct|wrong|unsure> <說明>`（沿用剪貼簿退路）；頁首顯示 report 表（族群、已判定、正確率）。skill `gold-review`：使用者貼上判定行，逐行執行 `bin/lct gold set`，最後 `bin/lct gold report`；allowed-tools：`Bash(bin/lct gold set *)`、`Bash(bin/lct gold report*)`、`Bash(go build -o bin/lct ./cmd/lct)`、`Read(./**)`；修改只留在 `main` 工作樹由人 commit。

- [ ] Step 1：失敗測試——route、頁面渲染與分組、選擇後複製的行格式、localStorage 失敗時仍可用、400px 樣式規則（同 `layout.test.ts` 的做法）、contract test 列出新 skill 的工具。
- [ ] Step 2–4：FAIL → 實作 → 全套 PASS。Step 5：commit。

### Task 7: `lct audit`

**Files:** Create `internal/audit/audit.go`、`internal/audit/audit_test.go`、`cmd/lct/audit.go`、`cmd/lct/audit_test.go`；Modify `README.md`。

**Interfaces — Produces:**
- 訊號與權重：`inconsistent`（不一致群組成員）3、`term-missing-same`（Task 4 的 Same）3、`term-missing`（其他未說明的缺譯名）1、`length-ratio` 2、其他格式警告 1。來源：`site.BuildReview` 的 suggestions／warnings 與 `scope.Analyze` 的條目單位。
- 候選：非 copy、不在待譯清單、不在凍結檔；分數 = 權重和；排序：分數高、族群優先序（`routing.yaml` priority）、key。
- `lct audit [--wave-size 500] [--wave N] [--usd-per-unit X] [--to-file TOPIC] [--json]`：輸出每波的單位數、各族群與各訊號數、估計費用（`--usd-per-unit`，預設由 tw `.meta/translate.json` 的 `cost_usd /(pending_before − pending_after)`，無法算時為 null）；`--to-file` 寫 `rules/requeue/<date>-audit-<topic>.yaml`（`reason`：「複查：<訊號>；保留原措辭，只修正問題」，`keys`：該波），沒有 `--to-file` 不寫任何檔。

- [ ] Step 1：失敗測試——排除規則、排序、分波、費用估算（含無法估算）、`--to-file` 才寫檔且格式與 `lct requeue --to-file` 相同。
- [ ] Step 2–4：FAIL → 實作 → 全套 PASS。Step 5：commit。

### Task 8: 族群指引內容（內容任務）

**Files:** Create `guides/families/<id>.md`（16 份）；Modify `guides/routing.yaml`（各族群 `guide:` 指向自己的檔）、`guides/families.md`（開頭註明：給人看的參考，不送給譯者）、`guides/README.md`、必要時 `guides/base.md`（搬出只屬單一族群的內容，不重複）。

- [ ] Step 1：在 scratch clone 建本分支的 `bin/lct`，逐族群收集證據：該族群的檔案與欄位、`base.md`／`rules/adjustments`／詞彙 notes 中只適用該族群的規則、已核可譯文中的慣例（只寫出現率 ≥80% 且 ≥30 單位者，以 `lct search` 或腳本統計並附數字與例子 key）。
- [ ] Step 2：每份用四節（文本類型；語氣與人稱；格式慣例；常見錯誤），每條規則句末以括號附出處；≤ 4096 bytes。沒有證據的節寫「（目前無足夠證據）」。
- [ ] Step 3：`bin/lct rules lint` 無錯、無總量警告；每族群挑一個單位 `bin/lct batch preview <key>`，確認 guides 清單為 base → 該族群檔 →（角色）。
- [ ] Step 4：`go vet ./... && go test ./...`（contract／lint 測試）→ PASS。
- [ ] Step 5：commit「指引：各族群專屬指引，族群規則附出處」；報告列出每份的規則數與出處類型。

### Task 9: 詞彙補 JP、EN、詞義（內容任務）

**Files:** Modify `glossary/terms.yaml`（只加 `jp`、`en`、`meaning`、`sources`，不改既有譯名與範圍）。

- [ ] Step 1：`bin/lct glossary evidence --json` 全部條目；名稱表一致（所有命中的 JP/EN 相同、≥1 個單位）的條目：直接填 `jp`、`en`，`sources` 記一個代表 key（`game:<檔名>#<pointer>`）；不一致的列入報告不填。
- [ ] Step 2：應用率未滿 100% 的 192 條與一詞多義條目（`also` 非空、notes 提到語境、或 Task 4 的 Other > 0）：無名稱表時，由三語對照提出 JP/EN 候選寫進 candidates 檔，`--candidates` 驗證，只填 OK 的。
- [ ] Step 3：`meaning`：有 keyword `desc` 的用官方說明改寫成 zh-TW 一兩句（`sources` 記 desc 的 key）；否則查 wiki.gg（API `action=query&list=search`、`action=parse&prop=wikitext`），有明確定義的才寫，`sources` 記頁面網址；都沒有就不寫。meaning 只描述出處說的內容。
- [ ] Step 4：`bin/lct rules lint` 無錯；`go test ./...` PASS；用 `bin/lct batch preview` 看一個含已補條目的單位，批次 glossary 帶出 jp/en/meaning。
- [ ] Step 5：commit；報告：補了幾條 jp/en、幾條 meaning、出處類型分布、名稱表不一致而未填的清單。

### Task 10: 應用率回顧前 20 條（內容任務）

**Files:** Modify `glossary/terms.yaml`（範圍與 `also`）；Create 報告於 scratch（不進 repo）。

- [ ] Step 1：重建 `bin/lct`，`bin/lct glossary scope --json`（Task 4 的同義／他義）；取缺漏單位最多的 20 條。
- [ ] Step 2：每條每組：缺漏 ≥5 且他義 ≥80% → 加 `not_files: [family:<id> 或 group:<top>]`；他義譯法集中於一種（≥80%）→ 改為加 `also`（若語意屬同一用法）或在報告提議另立條目（不自行新增條目）；同義缺漏不改範圍。條目無 jp/en 而無法分類的，只列報告。
- [ ] Step 3：重跑 scope，記錄每條應用率改前改後；`bin/lct rules lint` 無錯；`go test ./...` PASS。
- [ ] Step 4：commit；報告：每條的處置、證據數字、留為提案的與理由。

### Task 11: 基準樣本與第一波複查候選（內容任務）

**Files:** Create `rules/gold.yaml`（`bin/lct gold sample` 產生，判定空白）；Create 報告於 scratch。

- [ ] Step 1：重建 `bin/lct`（含 Task 8–10 的規則），`bin/lct gold sample`；檢查每族群 20 個（不足者全取）。
- [ ] Step 2：`bin/lct audit --json`（不帶 `--to-file`），記下第一波（500）的族群與訊號分布、估計費用；全部波數與總單位數。
- [ ] Step 3：`go test ./...` PASS；commit `rules/gold.yaml`（不提交任何 `rules/requeue/`）。

### Task 12: 真實資料驗收

- [ ] `lct site` 跑真實資料：時間、大小與上一版比較；`data/gold.json` 存在、`#/gold` 在 400px 與 1100px 無整頁橫向捲動（Playwright）。
- [ ] 抽 5 個單位（每個不同族群）：`lct batch preview` 的 guides 清單與總大小，與 `main` 版比較。
- [ ] 抽 20 條補了 meaning/jp/en 的條目核對出處可開啟／key 存在。
- [ ] 前 20 條的應用率改前改後表。
- [ ] `lct audit` 第一波估計。結果進最終回報；之後最終整支審查。

### Task 13: 族群引入共用指引（`include:`）

**Files:** Modify `internal/rules/routing.go`（`FamilyRule.Include []string`，yaml `include`）、`internal/rules/rules.go`（`GuidesFor`）、`internal/rules/lint.go`；Test `internal/rules/*_test.go`；Modify `guides/README.md`、`prompts/translate.md`（疊加順序）、`guides/routing.yaml`（註解說明 `include:`）。

- [ ] 測試先寫：`GuidesFor` 順序為 base → 族群 → include（依序、去重）→ 節點層 → 標記層 → 角色；include 與族群指引或層相同時只列一次。
- [ ] lint：include 每項須為存在的 `guides/**/*.md`；`guides/shared/*.md` 與族群指引同樣 ≤ 4096 bytes；每批總量檢查計入 include。
- [ ] `go vet ./... && go test ./...`；網站預覽靠 `GuidesFor`，不需改，跑 `cd site && npm test` 確認。

### Task 14: 括號慣例與共用指引（內容任務）

**Files:** Modify `guides/base.md`（第 7 條與其證據行）、`guides/families/*.md`、`guides/routing.yaml`（`include:`）；Create `guides/shared/*.md`、`rules/rewrite/2026-11-07-*.yaml`；Modify `docs/research/scripts/family_guides.py`。

- [ ] base.md 第 7 條改為全形，例外照舊；戰鬥說明類的半形寫進它們共用的指引；rpg.md 不再稱為例外。
- [ ] 跨族群重複的規則搬進 `guides/shared/`，用到的族群 `include:`；統計以引入族群合計（腳本可重算，≥80% 且 ≥30）。
- [ ] 改寫檔：非戰鬥族群的半形括號 → 全形；戰鬥說明類的全形 → 半形。只動 KR 有對應括號、括號內含中文的；排除 ruby「Y(X)」與亂碼符號。用真實資料模擬，報告各族群會改幾個單位並附樣本。
- [ ] `rules lint` 無錯；`go vet ./... && go test ./...`。

### Task 15: 詞彙證據排除不翻譯欄位（程式，已完成）

### Task 16: jp／en 多值、證據依條目範圍、`check: off`（程式）

**Files:** `internal/rules`（Term 的 jp／en 可為字串或清單、`check: off`、lint）、`internal/evidence`（多值聯集、只算條目適用的單位）、`internal/scope`（多值的 sense 判定、off 條目不產生提案）、`internal/validate` 與 `internal/batch`（off 不發 term-missing）、`internal/audit`（off 不產生訊號）、`internal/site` 與 `site/`（顯示）、`schemas/batch.schema.json`、`prompts/translate.md`。

- [ ] 測試先寫。單值條目的批次 JSON 與網站組回的結果必須與改前逐字相同。
- [ ] 多值條目在批次中帶全部值。網站契約測試涵蓋多值與 off。
- [ ] `go vet ./... && go test ./...`；`cd site && npm test && npm run typecheck && npm run build`。

### Task 17: B、C 的內容（內容任務）

- [ ] 補單一值 10 條；神父條補 jp、en；另立「EX CLEAR 전투 승리」；多值 4 條。
- [ ] 一般用語 7 條改 `check: off`；base.md 加一般用語與省略的規則；소모 的兩種慣例寫進指引。
- [ ] P1–P8；依詞義限制 9 條的範圍；also 4 條、「조작 슬롯」長條目、정신력 加 also 理智。
- [ ] H1 以人工調整統一成「本回合內」；H2 的 11 個與 KR 誤命中的 20 個單位標成已確認。
- [ ] `rules lint` 無錯；`go vet ./... && go test ./...`。

### Task 18: 其餘詞條的應用率回顧（內容任務，合併後追加）

- [ ] 用合併後的建置（tw `ff9b2c6`）重算 `glossary scope`，涵蓋所有應用率未滿 100%、且前面還沒處理過的條目。前 20 條與 Task 17 動過的不再重做。
- [ ] 依前面的處理方式分類：
  - 他義集中在某些檔或組：用 `not_files` 限制範圍。
  - 專名：另立條目。
  - 同一用法、集中在一種譯法：加 `also`。
  - 一般用語：`check: off`，列為提案。
  - KR 誤命中：`accept`。
  - 同義缺漏：不縮範圍。機械式的可用改寫或人工調整，其餘列入複查。
- [ ] 明確的直接改，邊界的留提案。改前改後的應用率附在報告。
- [ ] `rules lint` 無錯；`go vet ./... && go test ./...` 通過。

## 最後

最終整支審查（最強模型）→ 修正 Critical/Important（Minor 也修，使用者要求）→ 回報（含所有內容變更清單供審閱、rulings、費用與不提交重翻的提醒）→ 使用者同意後才合併。
