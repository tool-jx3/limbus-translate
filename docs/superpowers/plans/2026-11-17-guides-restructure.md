# 指引整理 實作計畫

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal：** 譯者讀到的每條規則都是現行版、只寫一次、只帶給用得到的檔；統計與來源移到翻譯時不會讀到的依據檔。

**Architecture：** 先加 lint（擋指向依據檔的連結、警告統計樣式），再依序整理 `base.md`（拆出 `skill-text.md`、`story-labels.md`）、族群與共用指引、角色指引；每份改動的指引都附逐條對照表，依據寫進 `docs/guides-evidence/<同路徑>`。

**Tech Stack：** Go（`internal/rules` 的 lint、`cmd/lct`）、Markdown 指引（臺灣繁體中文）、`guides/routing.yaml`。

**Spec：** `docs/superpowers/specs/2026-11-17-guides-restructure-design.md`

## Global Constraints

- 不改任何規則的意思、不增加規則；只刪已被取代或已不適用的條文（設計 6.2）。兩層說法不一致：明文「取代」「改為」的照後者；沒有明文取代的不自行決定，對照表標「衝突」，兩種說法照原樣留在原處（設計 6.3）。
- 「既有譯文不必為此改」這類指示照寫，不附數字。
- 會送給譯者的指引檔：`guides/base.md`、`guides/families/*.md`、`guides/shared/*.md`、`guides/characters/*.md`。`guides/README.md`、`guides/families.md` 是給人看的。
- 依據檔放在 `docs/guides-evidence/`，路徑與指引對應（`guides/base.md` → `docs/guides-evidence/base.md`）；指引內不寫指向依據檔的連結。
- 指引裡不寫：次數、比例、「既有 a／b」、行數、檔數、「使用者決定／確認 YYYY-MM-DD」、調整檔或改寫檔名、研究報告路徑、出處 key、「自 base.md 移入」、舊譯的經過、不改變寫法的工具行為。
- 族群指引與共用指引每份 ≤ 4096 bytes；任一 KR 檔的指引總量不超過 35,000 bytes（lint 警告線）。
- `guides/shared/ruby.md` 只改指向，≤ 4096 bytes；節名「白名單位置的注音」不變；`base.md` 的「ruby 注音」「ruby 的位置」留在 `base.md`。
- 不動 `glossary/`、`rules/`（含 `rules/requeue/`）、`guides/npcs.yaml`、角色指引開頭的 YAML、任何 `.work/`。
- `prompts/` 與技能不寫模型名稱；翻譯用的 `prompts/translate.md` 與 `translate-pending` 技能不提依據檔。
- 提交前：`go vet ./... && go test ./...`；`go build -o bin/lct ./cmd/lct && bin/lct rules lint --json` 0 錯誤。
- 提交訊息結尾：
  ```
  Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
  Claude-Session: https://claude.ai/code/session_015XrjRvHe6FfxjemSS1js35
  ```

## Review Focus

1. 精簡時規則被默默刪掉或弱化（特別是「既有的不必改」、條件與例外）：每個內容任務的對照表逐句列出去向，審查逐句核對。
2. 拆出去的共用指引沒帶到需要的檔（技能文字出現在四個族群以外、`StageNode*` 的 `place`）：Task 3 量測帶入範圍並列出沒帶到的觸發單位。
3. 舊節名的引用殘留（`prompts/`、族群與角色指引裡的「資料補充」「補充規則」「原指引」「與程式語言有關的翻譯須知」「base.md 第 N 條」）：Task 3 與 Task 7 各跑一次 grep，結果為 0（`site/`、`work/`、`docs/research/` 的舊資料除外）。
4. 統計或依據連結又出現在譯者讀的指引：Task 1 的 lint 擋連結、警告統計；Task 7 要求警告為 0。
5. 角色指引推翻 base 的例外（例如辛克萊對所有人用「您」）在整理後失去優先權或被刪：Task 3 的 base「原則」寫明優先順序；Task 5、6 的審查逐檔確認例外仍在。

---

### Task 1：lint 擋依據連結、警告統計樣式

**Files：**
- Create: `internal/rules/guidetext.go`、`internal/rules/guidetext_test.go`
- Modify: `internal/rules/lint.go`（`Lint` 呼叫新的檢查）、`cmd/lct/rules.go`（加上警告）、`cmd/lct/rules_test.go`

**Interfaces：**
- Produces：
  - `func TranslatorGuides(root string) ([]string, error)`：`guides/base.md`、`guides/families/*.md`、`guides/shared/*.md`、`guides/characters/*.md` 存在的檔，repo 相對路徑、排序。
  - `func (l *linter) guideEvidenceLinks()`：上面的檔任一行含 `guides-evidence` 就報錯：`LintError{File: rel, Message: "line N mentions guides-evidence: a guide the translator reads must not point to the evidence files"}`。
  - `func LintGuideTextWarnings(root string) []LintError`：上面的檔每一行符合下列任一樣式就一個警告，`Message: "line N looks like evidence (statistics or a source); move it to docs/guides-evidence/<same path>"`。

樣式（逐字，Go regexp）：
```
既有 ?[0-9]
[0-9][0-9,]*／[0-9]
[0-9]+(?:\.[0-9]+)?%[（）]
使用者(?:決定|確認)
raw-20[0-9]{6}
docs/research/
```

- [ ] **Step 1：寫失敗的測試** `TestGuideEvidenceLinks`、`TestLintGuideTextWarnings`（`guidetext_test.go`，暫存目錄當 root）：
  - `guides/families/x.md` 有一行 `見 docs/guides-evidence/families/x.md` → `Lint` 回報一個錯誤，`File` 是 `guides/families/x.md`，訊息含 `line 1`。
  - 同一行放在 `guides/README.md`、`guides/families.md` → 沒有錯誤、沒有警告。
  - 警告各一行命中：`（既有 2,073 處）`、`（2,255／2,303）`、`97.6%（`、`（使用者決定 2026-11-07）`、`raw-20261105`、`docs/research/x.md`。
  - 不命中：`不低於3/5級`、`使自身獲得N級 [X] 強度`、`+N%`、`<size=50%>`、`1回 [Laceration]`。
- [ ] **Step 2：** `go test -count=1 -run 'TestGuideEvidenceLinks|TestLintGuideTextWarnings' ./internal/rules/` → FAIL（未定義）。
- [ ] **Step 3：** 實作；`Lint` 在 `l.baseGuide()` 之後呼叫 `l.guideEvidenceLinks()`；`cmd/lct/rules.go` 在既有警告之後加 `rules.LintGuideTextWarnings(ctx.Root)`。`cmd/lct/rules_test.go` 加一個案例：fixture 的族群指引含統計時 `rules lint --json` 的 `warnings` 有它、`ok` 仍為 true。
- [ ] **Step 4：** `go test -count=1 ./internal/rules/ ./cmd/lct/` → PASS；`bin/lct rules lint --json` 在真實 repo：0 錯誤，新警告的數量寫進報告（整理前的基準）。
- [ ] **Step 5：** 提交「規則檢查：譯者讀的指引不可指向依據檔，統計樣式警告」。

### Task 2：寫法規範、依據目錄、維護技能

**Files：**
- Modify: `guides/README.md`（寫法規範與依據檔說明）、`.claude/skills/{adjust,review,review-page,rules-review}/SKILL.md`、`cmd/lct/contract_test.go`、`docs/research/scripts/README.md`、`docs/research/scripts/family_guides.py`（只改 docstring 與輸出說明的指向）
- Create: `docs/guides-evidence/README.md`

**Interfaces：**
- Consumes：Task 1 的譯者指引範圍與樣式。
- Produces：依據檔的路徑規則（`guides/<p>` → `docs/guides-evidence/<p>`）與段落格式（用與指引相同的規則標題分段；每段可有「統計（資料版本）」「決定」「經過」「出處」），後續任務照用。

- [ ] **Step 1：寫失敗的測試**（contract test）：四個維護技能的 `allowed-tools` 都有 `Edit(./docs/guides-evidence/**)`；`prompts/translate.md` 與 `.claude/skills/translate-pending/SKILL.md` 不含 `guides-evidence`。
- [ ] **Step 2：** `go test -count=1 -run Contract ./cmd/lct/` → FAIL。
- [ ] **Step 3：** 改四個技能：`allowed-tools` 加 `Edit(./docs/guides-evidence/**)`；在它們提到改 `guides/` 的地方加一句「新增或修改規則時，統計與來源寫進 `docs/guides-evidence/` 的對應檔，不寫進指引」（技能行數上限照 contract test）。
- [ ] **Step 4：** 改寫 `guides/README.md` 的寫法規範，內容照設計第 5 節（寫什麼、不寫什麼、門檻 a／b ≥ 80% 且 b ≥ 30 記在依據檔、沒有內容的節不寫）與第 9 節（依據檔在哪、翻譯不讀）。寫 `docs/guides-evidence/README.md`（路徑對應、段落格式、不送譯者、改依據不觸發 `rebuild.yml`）。`docs/research/scripts/README.md` 與 `family_guides.py` 的說明改成數字寫進依據檔。
- [ ] **Step 5：** `go test -count=1 ./cmd/lct/` → PASS；提交「指引：寫法規範與依據檔目錄；維護技能可寫依據檔」。

### Task 3：`base.md` 重寫，拆出 `skill-text.md`、`story-labels.md`

**Files：**
- Modify: `guides/base.md`、`guides/characters/dante.md`、`guides/routing.yaml`、`guides/families/{ally,enemy,keywords,mirror}.md`（只改指向 base 的文字）、`guides/shared/battle-text.md`（只改指向）、`guides/shared/ruby.md`（只改指向）、`guides/families/rpg.md`（只改指向）、`guides/README.md`（結構表）、`prompts/translate.md`、`cmd/lct/contract_test.go`、`internal/rules/*_test.go`（若有測試釘住 base 的節名或 routing 項）
- Create: `guides/shared/skill-text.md`、`guides/shared/story-labels.md`、`docs/guides-evidence/base.md`

**Interfaces：**
- Consumes：Task 1 的 `LintGuideTextWarnings`、Task 2 的依據檔格式。
- Produces：`base.md` 的新節名（後續任務改指向時用）：`## 原則`、`## 代碼與標籤`、`## 標點與書寫`、`## 用語`、`## 人稱與稱呼`；`skill-text.md`、`story-labels.md` 的條目標題。

- [ ] **Step 1：量測帶入範圍**（`.superpowers/tmp/` 的小程式，唯讀用主 checkout 的 `.work/raw`；`rules.Load`、`pack.OpenRaw`、`GuidesFor`）：技能文字的觸發單位（`\[[A-Za-z_0-9]+\]\s*(강도|횟수)`、`[0-9]+\s*(강도|횟수)`、`소모`、`<noparse>`）依 top 節點與族群計數；`place`、`teller`、`title`、`name`、`nickName` 依檔計數。決定 `skill-text.md` 的帶入：ally、enemy、keywords、mirror 以 `include:` 接在 `battle-text.md` 之後；四族群以外觸發單位 ≥ 30 的 top 節點以 `guides:` 帶入。`story-labels.md` 以 `guides:` 帶給 `StoryData/`、`ScenarioModelCodes*`、`StageNode*`。數字寫進報告。
- [ ] **Step 2：對照表**：照設計 6.2 的表，把現行 `base.md` 每一句的去向列進報告（新節／`skill-text.md`／`story-labels.md`／`dante.md`／依據檔／刪（原因））；衝突照 6.3 標「衝突」並原樣保留。
- [ ] **Step 3：改寫**：照設計 6.1 的五節寫 `base.md`；寫 `skill-text.md`、`story-labels.md`（各 ≤ 4096 bytes；`skill-text.md` 超過時照設計 7.1 依主題分成兩份、由同樣的族群引入）；`dante.md` 移入「但丁台詞開頭是英文詞時」；`routing.yaml` 加帶入；統計、來源、經過寫進 `docs/guides-evidence/base.md`。
- [ ] **Step 4：改指向**：`prompts/translate.md`（「標點與書寫」的補充規則、「第二人稱：您／你」「性別：你／妳、他／她」「與程式語言有關的翻譯須知」改成新節名）、族群與共用指引裡指向 base 舊節名的文字、`guides/README.md` 結構表加兩份新共用指引；contract test 若釘住舊節名一併改。然後跑：
  `grep -rn "資料補充\|補充規則\|原指引\|與程式語言有關\|base.md 第\|base.md」第" guides/base.md guides/shared prompts .claude/skills` → 只剩角色與族群指引（Task 4–6 處理），base 與 shared 為 0。
- [ ] **Step 5：檢查**：`bin/lct rules lint --json` 0 錯誤；`LintGuideTextWarnings` 對 `guides/base.md`、`guides/shared/skill-text.md`、`guides/shared/story-labels.md` 為 0；量測各組指引總量前 5 名（前後對照）與兩份新共用指引帶到的檔數；`go vet ./... && go test -count=1 ./...` PASS。
- [ ] **Step 6：** 提交「指引：base.md 依主題重寫，技能文字與劇情標籤欄位拆成共用指引；依據移到 docs/guides-evidence」。

### Task 4：族群指引與共用指引

**Files：**
- Modify: `guides/families/*.md`（16 份）、`guides/shared/battle-text.md`、`guides/shared/korean-brackets.md`
- Create: `docs/guides-evidence/families/<id>.md`、`docs/guides-evidence/shared/<name>.md`（有依據的才建）

**Interfaces：**
- Consumes：Task 3 的 base 新節名、`skill-text.md` 條目標題。

- [ ] **Step 1：對照表**：每份檔每一句的去向（留／改寫／依據檔／刪（原因）／衝突）。
- [ ] **Step 2：改寫**：照設計第 8 節：統計與來源移到依據檔；「文本類型」去掉檔數；沒有內容的節不寫；常見錯誤保留錯誤樣子、去掉次數與經過；只重述 base 的條目改成指向 base 的一句或刪；指向 base 舊節名的改成新節名。
- [ ] **Step 3：檢查**：`LintGuideTextWarnings` 對這些檔為 0；每份 ≤ 4096 bytes；`bin/lct rules lint --json` 0 錯誤；`go test -count=1 ./cmd/lct/ ./internal/rules/` PASS。
- [ ] **Step 4：** 提交「指引：族群與共用指引只留規則，依據移到 docs/guides-evidence」。

### Task 5：角色指引（罪人與主要角色）

**Files：**
- Modify: `guides/characters/{yisang,faust,donquixote,ryoshu,meursault,honglu,heathcliff,ishmael,rodion,sinclair,outis,gregor,dante,vergilius,charon}.md`
- Create: 對應的 `docs/guides-evidence/characters/<id>.md`

- [ ] **Step 1：對照表**（每份檔每一句）。
- [ ] **Step 2：改寫**：照設計第 8 節：去掉「原指引／資料補充」兩層，仍有效的內容寫成規則；統計移到依據檔；「例句」表留 KR 與既有譯文兩欄，「出處」欄移到依據檔；開頭 YAML 不動；推翻 base 的角色例外照寫（例如辛克萊對所有人用「您」）。節名照設計第 14 節的建議取用。
- [ ] **Step 3：檢查**：`LintGuideTextWarnings` 對這些檔為 0；`bin/lct rules lint --json` 0 錯誤（角色 YAML 照常解析）；`go test -count=1 ./internal/rules/ ./cmd/lct/` PASS。
- [ ] **Step 4：** 提交「指引：罪人與主要角色指引只留規則，依據移到 docs/guides-evidence」。

### Task 6：角色指引（其餘）

**Files：**
- Modify: `guides/characters/*.md` 中 Task 5 以外的全部
- Create: 對應的 `docs/guides-evidence/characters/<id>.md`

- [ ] 步驟同 Task 5（對照表、改寫、檢查、提交「指引：其餘角色指引只留規則，依據移到 docs/guides-evidence」）。

### Task 7：驗收

**Files：** 不改程式；結果記在 SDD 帳本。若檢查發現問題，回到負責的任務修。

- [ ] **Step 1：** `bin/lct rules lint --json`：0 錯誤；`LintGuideTextWarnings` 的警告 0；其他警告與 Task 1 前的基準相同。
- [ ] **Step 2：** 舊節名 grep（同 Task 3 Step 4，但範圍改成 `guides/ prompts/ .claude/skills/`，排除 `guides/README.md` 的歷史說明）→ 0。
- [ ] **Step 3：** 字數：各組指引總量前 5 名（與整理前對照）；兩份新共用指引帶到的節點與檔數；技能文字觸發單位沒帶到 `skill-text.md` 的數量。
- [ ] **Step 4：依據不外流**：待譯是 0，產生不出批次，所以改用 `bin/lct guide resolve` 對劇情（`StoryData/S101B.json`）、人格語音、技能（`Skills_*`）、介面（`UI*`）各一個檔列出指引，確認沒有 `docs/guides-evidence`，且列出的檔 `LintGuideTextWarnings` 為 0。
- [ ] **Step 5：** 收集 Task 3–6 對照表裡的「衝突」，整理成一份清單交給使用者決定（不在這次自行解決）。
- [ ] **Step 6：** `go vet ./... && go test -count=1 ./...` PASS。
