# 不一致的譯法：連同 JP/EN 的檢視 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 不一致的譯法多一個「連同 JP/EN」檢視（每種譯法旁顯示 JP/EN、徽章、篩選），審核的 skill 與 `lct accept --dry-run` 也看得到同樣的 JP/EN，決定留給使用者。

**Architecture:** `internal/site/review.go` 分組時記下每個單位的 JP/EN，在 `Variant` 加 `JPEN []JPENGroup`、在 `SuggestionItem` 加 `JPENClass`；`lct accept --dry-run` 原樣印出；前端在規則建議分頁加切換、徽章與篩選；兩個 skill 改用 dry-run 取得 JP/EN。

**Tech Stack:** 沿用 Go 1.26.1、`site/` 的 Preact＋Vite＋TypeScript、Vitest（不加依賴）。

**Spec:** `docs/superpowers/specs/2026-11-04-review-jpen-mode-design.md`；審核頁 spec `docs/superpowers/specs/2026-11-03-review-page-design.md`。

## Global Constraints

- JSON（照抄）：`variants[].jpen` = `[{"jp", "en", "count", "keys"}]`，`jp`／`en` 沒有時省略；`jpen_class` = `"same"` | `"explained"` | `"partial"`；只加鍵，`format` 仍是 2，`variants[].keys` 不變
- 分類：`same` 所有單位的 (JP, EN) 相同；`explained` 不只一種 (JP, EN) 且每一種只對到一種 TW；`partial` 其他；缺少的語言當作一個值
- 子組排序：單位數由多到少，相同時依第一個 key；`keys` 依原本順序（檔案順序、單位順序）
- 網址：`mode=jpen`（沒有就是「只看 KR」）、`jpen=explained|partial|same`
- 文字（照抄）：切換「只看 KR」／「連同 JP/EN」；徽章「JP/EN 可解釋」「部分可解釋」「JP/EN 相同」；篩選標籤「JP/EN」，選項「全部」「可解釋」「部分可解釋」「相同」
- ID、確認清單、`lct suggest` 的輸出、建置都不變
- skill 不因 JP/EN 分類替使用者確認或不採用；skill 只能用 `bin/lct` 指令、`go build`、`Read`、`Edit`；不寫模型名稱
- 不用 `innerHTML`；JP 文字 `lang="ja"`、EN `lang="en"`
- 提交前 `go vet ./... && go test ./...`（以 `set -o pipefail` 確認真正的結束碼）；動到 `site/` 另跑 `cd site && npm run typecheck && npm test && npm run build`
- 開發分支 `claude/festive-wright-g2k853`；`main` 在 Task 4 經使用者同意後以 fast-forward 更新（使用者已說做完後合併）

## Review Focus

1. **缺 JP 或 EN 的單位**：子組省略該鍵、頁面顯示「—」、分類把「沒有」當一個值 — Task 1 `TestReviewJPENMissingLanguage`、Task 2 `jpen row shows a dash`
2. **部分可解釋的組裡，哪一個 JP/EN 是真正不一致的**：同一個 (JP, EN) 出現在兩種以上譯法時，該子組要標「同樣的 JP/EN 有其他譯法」 — Task 2 `marks the JP/EN pair shared by two translations`
3. **深連結 `id` 加上 `mode=jpen`**：目標項目照樣顯示並高亮 — Task 2 `deep link keeps working in the JP/EN view`
4. **篩選數量與「顯示已確認」**：選項的數量算的是目前看得到的項目（隨 `all=1` 變） — Task 2 `jpen filter counts follow show-confirmed`
5. **長的 JP/EN 文字**：換行、不讓頁面左右捲動 — Task 4 截圖檢查

---

### Task 1: Go — JP/EN 子組、分類與 dry-run 輸出

**Files:**
- Modify: `internal/site/review.go`（`group` 記下每個 key 的 JP/EN；`Variant.JPEN`；`SuggestionItem.JPENClass`）、`internal/site/review_test.go`
- Modify: `cmd/lct/accept.go`（`lookupItem` 帶 `JPENClass`；文字輸出列出 JP/EN 與分類）、`cmd/lct/accept_test.go`

**Interfaces:**
- Produces（`internal/site`）：
  - `type JPENGroup struct { JP *string \`json:"jp,omitempty"\`; EN *string \`json:"en,omitempty"\`; Count int \`json:"count"\`; Keys []string \`json:"keys"\` }`
  - `Variant.JPEN []JPENGroup \`json:"jpen"\``；`SuggestionItem.JPENClass string \`json:"jpen_class,omitempty"\``（只有 `inconsistent` 有值）
  - `func jpenClass(variants []Variant) string`
- Produces（`cmd/lct`）：`lookupItem.JPENClass string \`json:"jpen_class,omitempty"\``；`--dry-run` 文字輸出在每個 `variant (n): …` 下一行一個 `    JP …／EN …（n）`（缺少時「—」），並在項目下印 `  JP/EN: <分類>`

- [ ] **Step 1: 寫失敗的測試**
  - `TestReviewJPENGroups`：同 KR 的三個單位，A 譯法兩個單位 JP/EN 不同、B 譯法一個 → A 的 `JPEN` 兩組、各 `Count` 1，順序依第一個 key；B 一組
  - `TestReviewJPENClass`：三種情形各得 `same`、`explained`、`partial`
  - `TestReviewJPENMissingLanguage`：一個單位沒有 EN → 子組沒有 `en` 鍵（JSON），且與有 EN 的單位分成兩組
  - `TestAcceptDryRunShowsJPEN`：`lct accept --dry-run --json` 的 inconsistent 項目有 `jpen_class` 與 `variants[].jpen`；文字輸出含 `JP/EN:`
- [ ] **Step 2: 執行確認失敗** — Run: `go test ./internal/site/ ./cmd/lct/ -run 'JPEN'` — Expected: FAIL（欄位不存在）
- [ ] **Step 3: 實作**（`group` 加 `jpen map[string][2]*string`（key → JP、EN），在建 `g.keys` 的同一處填；`variantsOf` 依每個譯法的 keys 分子組）
- [ ] **Step 4: 執行確認通過** — Run: `set -o pipefail; go vet ./... && go test ./...` — Expected: exit 0
- [ ] **Step 5: Commit 並 push**：`feat(site): the JP/EN of each variant of an inconsistent group, and its class`

### Task 2: 前端 — 切換、JP/EN 子組、徽章、篩選

**Files:**
- Modify: `site/src/types.ts`（`JPENGroup`、`Variant.jpen`、`SuggestionItem.jpen_class`）、`site/src/route.ts`（review 路由加 `mode: 'kr' | 'jpen'`、`jpen: '' | 'explained' | 'partial' | 'same'`；`reviewHref` 的 opts 同名）、`site/src/review/filter.ts`、`site/src/review/ReviewView.tsx`、`site/src/review/SuggestionCard.tsx`、`site/src/styles.css`
- Modify: `site/fixtures/data/review/suggestions.json`（現有 inconsistent 項目加 `jpen` 與 `jpen_class`；再加一個 `partial` 的組，含一個缺 EN 的單位——對應的檔案資料與 `rv`、`meta.review`、`fixtures.test.ts` 一併更新）
- Test: `site/src/route.test.ts`、`site/src/review/filter.test.ts`、`site/src/review/ReviewView.test.tsx`、`site/src/fixtures.test.ts`

**Interfaces:**
- Consumes: Task 1 的 JSON
- Produces: `filterSuggestions(items, { …既有, jpen })` 只篩 `kind === 'inconsistent'` 的項目，其他種類照常保留

- [ ] **Step 1: 寫失敗的測試**
  - `review route keeps mode and jpen`：`parseHash(reviewHref('suggestions', { mode: 'jpen', jpen: 'partial', id: 'x' }))` 來回不變；沒有 `mode` → `'kr'`
  - `JP/EN view shows each variant's JP and EN`：`mode=jpen` 時卡片的每個譯法下有它的 JP、EN 與數量；`mode` 沒有時與現在相同（沒有 JP/EN 子組、沒有徽章）
  - `jpen row shows a dash`：缺 EN 的子組 EN 欄顯示「—」
  - `marks the JP/EN pair shared by two translations`：`partial` 組裡出現在兩種譯法下的 (JP, EN) 子組都有標記「同樣的 JP/EN 有其他譯法」
  - `badges and filter`：三種徽章文字照抄；`jpen=same` 只留 `same` 的 inconsistent 組，非 inconsistent 的建議照常顯示；選項數量正確
  - `jpen filter counts follow show-confirmed`：`all=1` 時數量包含確認過的組
  - `deep link keeps working in the JP/EN view`：`mode=jpen&id=<inconsistent id>` → 該卡片已渲染且高亮
- [ ] **Step 2: 執行確認失敗** — Run: `cd site && npm test` — Expected: FAIL
- [ ] **Step 3: 實作**（切換放在規則建議分頁的篩選列；「JP/EN」篩選只在 `mode=jpen` 時出現；子組列 JP、EN 用 `TmpText` 與 `lang`）
- [ ] **Step 4: 執行確認通過** — Run: `cd site && npm run typecheck && npm test && npm run build` — Expected: PASS
- [ ] **Step 5: Commit 並 push**：`feat(site): a JP/EN view of the inconsistent translations, with badges and a filter`

### Task 3: skill 與合約

**Files:**
- Modify: `.claude/skills/review-page/SKILL.md`、`.claude/skills/rules-review/SKILL.md`（`allowed-tools` 加 `Bash(bin/lct accept --dry-run *)`）、`.github/workflows/rules-proposal.yml`（給 action 的工具清單加同一項）、`cmd/lct/contract_test.go`（`rulesReviewTools` 加這一項；`reviewPageSays`、`rulesReviewSays` 加「不因 JP/EN 分類替人決定」的要求）
- Modify: 審核頁 spec §9 與 JP/EN 設計的「審核時」一節不用改（已寫好）；README 的審核頁段落加一句「連同 JP/EN」

**Interfaces:** Consumes Task 1 的 dry-run 輸出

- [ ] **Step 1: 寫失敗的合約測試**：`rulesReviewTools` 含 `Bash(bin/lct accept --dry-run *)`，且與 `rules-proposal.yml` 的工具清單一致；兩個 skill 的本文須含「不因 JP/EN 分類」相關文字（`…Says`）
- [ ] **Step 2: 執行確認失敗** — Run: `go test ./cmd/lct/ -run 'Contract|RulesReview|ReviewPage'` — Expected: FAIL
- [ ] **Step 3: 改 skill、workflow 與 README**：
  - `review-page`：查不一致的譯法時，把每種譯法的 JP/EN 子組與分類給使用者看；不因分類自動確認
  - `rules-review`：處理不一致的譯法時先 `bin/lct accept --dry-run --json '<id>'` 取得 JP/EN，證據中列出各譯法的 JP/EN 與分類；不因分類寫進 `rules/declined.yaml`、不因分類統一譯法；沒有其他把握的理由時列進「需要人決定」
- [ ] **Step 4: 執行確認通過** — Run: `set -o pipefail; go vet ./... && go test ./...` — Expected: exit 0
- [ ] **Step 5: Commit 並 push**：`feat: the review skills see each variant's JP/EN and leave the decision to the person`

### Task 4: 實際資料驗收、最終審查、合併

- [ ] **Step 1:** `bin/lct site --out site/public/data --json`；`review/suggestions.json` 的 `jpen_class` 數量（以目前資料預期 169／86／245，可能因 R8 與確認而略有差異，記下實際值）；`bin/lct accept --dry-run '<一個 explained 組的 id>'` 的文字輸出
- [ ] **Step 2:** `cd site && npm run build`，掛在 `/limbus-translate/` 子路徑，Playwright（1280 淺色、390 深色）：切換、JP/EN 子組、徽章、篩選與數量、深連結、無左右捲動、無 console 錯誤
- [ ] **Step 3:** 一位新的審查者（最強的模型）看整個變更（`400c41b..HEAD`）；修正、再驗證
- [ ] **Step 4:** 回報使用者；`main` 以 fast-forward 更新（包含先前的 し協會 詞彙與標點確認），確認 `ci`、`rebuild`、`pages` 成功
