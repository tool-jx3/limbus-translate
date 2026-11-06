# 依路徑與檔名分群、「詞彙」頁、詞彙表預設只警告 實作計畫

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal：** 檔案依路徑與檔名自動分成兩層樹，再加第三層：章節、角色、活動。族群改由樹的節點組成（16 個族群）。詞彙表預設只警告，條目要寫 `check: hard` 才會退回。網站側欄改成分群樹，另新增「詞彙」頁，可以看每個詞條用了與沒用譯名的全部單位。

**Architecture：**
- `internal/rules` 新增兩個推導，都只看路徑，路由、詞彙表範圍、詞彙範圍統計與網站資料共用：
  - 節點推導 `FileNode(file)`；
  - 第三層推導 `(*Rules).FileTag(file)`。
- 路由的 `nodes` 取代 `match`。
- `internal/scope` 的第二種分法改成頂層節點，並交出每個詞條的命中單位，網站匯出成 `data/glossary/`。
- 前端新增分群樹側欄、首頁族群表與「詞彙」頁（`#/g`）。

**Tech Stack：** Go（`internal/rules`、`internal/batch`、`internal/scope`、`internal/site`、`cmd/lct`）、Preact + Vite（`site/`）、`guides/`、`glossary/`、`schemas/`、`prompts/`、`.claude/skills/`。

**Spec：** `docs/superpowers/specs/2026-11-06-file-groups-design.md`（使用者 2026-11-06 以「Ok go」核准）。執行者兩份都要讀；規格是依據，計畫是它的推論。

## Global Constraints

- `CLAUDE.md`：
  - 規則只在 `guides/`、`glossary/`、`rules/`；
  - 狀態只經由 `lct`；
  - `raw`、`tw`、Release 只由 CI 寫入；
  - 所有 `lct` 指令支援 `--json`；
  - skill 與 `prompts/` 不寫模型名稱。
- 節點樣式、`group:`、`event:` 樣式都用 `path.Match`，`*` 不跨 `/`。
- 族群比對：
  - 先 `sub` 後 `top`；
  - 同一層，`nodes` 寫出完整節點名稱的族群優先，其次取路由裡第一個萬用字元樣式符合的族群；
  - 都沒有，就是 `nodes: []` 的後備族群 `unsorted`。
- `guides/routing.yaml` 的族群與 priority 照規格「附錄」逐字照抄。族群順序就是比對順序。
- 第三層：
  - 每個檔最多一個標記，依序取角色 → 章節 → 活動；
  - `a1c<章>` 算章節；
  - 純數字不算活動。
- 「可考慮退回」：`HardRate = 0.95`（應用率要高於它）、`HardMinN = 20`（出現至少 20 個單位）。兩者寫在程式常數，頁面上顯示。
- 詞彙表 `check`：`hard` 或 `warn`，省略等於 `warn`。只有 `check: hard` 且族群 `glossary` 為 `hard`（或省略）時，缺譯名才在送出時退回；note 寫出原詞照舊收下。`forbidden` 一律退回。
- 新的批次檔欄位要同時改 `schemas/batch.schema.json`；skill 與 `prompts/` 的文字改了，`cmd/lct/contract_test.go` 一起改。
- 提交前要跑：
  - `set -o pipefail; go vet ./... && go test ./...`；
  - 動到 `site/`：再跑 `cd site && npm test && npm run typecheck && npm run build`；
  - 動到 `glossary/`、`guides/`：再跑 `go build -o bin/lct ./cmd/lct && bin/lct rules lint`。
- 網站在寬 400px 時頁面不得橫向捲動；寬表格放在自己的 `overflow-x: auto` 容器。
- commit 訊息結尾加 harness 給的署名，另加 `Claude-Session: https://claude.ai/code/session_015XrjRvHe6FfxjemSS1js35`。
- 在 `claude/festive-wright-g2k853` 開發；不 push `main`、`tw`、`raw`。
- 真實資料只在 scratch clone 使用：
  - 先 `git fetch /home/user/limbus-translate refs/remotes/origin/raw:refs/heads/raw-latest refs/remotes/origin/tw:refs/heads/tw-latest`；
  - 再以這兩個分支建 `.work/raw`、`.work/tw`；
  - 不碰 `/home/user/limbus-translate/.work/`。

## Review Focus

1. **新檔的頂層節點沒有族群列到**（遊戲更新多了 `FooBar-tkt.json`）：
   - `Family` 給 `unsorted`，不會是 `""`；
   - `lct suggest` 與 `lct plan` 照現行「預設族群的新檔」提醒；
   - 有 raw 時 lint 警告。
   - → Task 2 `TestUnlistedTopFallsInUnsorted`。
2. **怪檔名**（`kr.json`、`reward.json`、`Ch1.json`、`StoryData/RPG1.json`、沒有 `-` 的 `Walpu6EventText.json`）都有確定的 `top`／`sub`，而且不會 panic。→ Task 1 `TestFileNodeOddNames`。
3. **`check: hard` 的條目落在 warn 族群的檔，以及暫定詞**：
   - 兩者送出時缺譯名都只警告，照常收下；
   - `forbidden` 仍退回。
   - → Task 4 `TestSubmitGlossaryCheck`。
4. **側欄**：
   - 篩選命中被合併的子群或第三層裡的檔時，所有上層都展開；
   - 換到別的檔時，那個檔的上層自動展開。
   - → Task 7 `Directory.test.tsx`「opens every level above a hit / the current file」。
5. **「詞彙」頁的大詞條與壞連結**：
   - 「用了」區一次顯示 100 個；
   - 換篩選時，顯示數回到 100；
   - 網址的條目 ID 不存在時，顯示「找不到」，不出錯。
   - → Task 8 `GlossaryView.test.tsx`。

---

### Task 1: 節點與第三層的推導

**Files:**
- Create：`internal/rules/groups.go`
- Modify：`internal/rules/characters.go`（`CharacterByFile` 補人格技能檔）
- Test：`internal/rules/groups_test.go`

**Interfaces:**
- Produces：

```go
// Node is the place of a file in the group tree. file is the relative path
// without language prefix, as Family takes it.
type Node struct {
	Top string `json:"top"`           // "StoryData/", "Skills"
	Sub string `json:"sub,omitempty"` // "StoryData/S", "Skills_Enemy"; "" when none
}
func FileNode(file string) Node

const (
	TagCharacter = "character"
	TagChapter   = "chapter"
	TagEvent     = "event"
)
type Tag struct {
	Kind  string `json:"kind"`  // TagCharacter, TagChapter or TagEvent
	Value string `json:"value"` // "heathcliff", "10", "walpu6"
	Label string `json:"label"` // "希斯克里夫", "第 10 章", "walpu6"
}
func (r *Rules) FileTag(file string) (Tag, bool)
```

- [ ] **1.1 測試先行**：`TestFileNode` 表格測試。
  - 輸入與預期的 `{Top, Sub}`：

    | 輸入 | Top | Sub |
    |---|---|---|
    | `StoryData/S1037B.json` | `StoryData/` | `StoryData/S` |
    | `StoryData/1D101A.json` | `StoryData/` | `StoryData/D` |
    | `StoryData/ES050A.json` | `StoryData/` | `StoryData/ES` |
    | `StoryData/RPG1.json` | `StoryData/` | `StoryData/RPG` |
    | `PersonalityVoiceDlg/Voice_Heathcliff_RCorp4_10703.json` | `PersonalityVoiceDlg/` | `PersonalityVoiceDlg/Voice_Heathcliff` |
    | `RPGSystem/rpg-loc-swarm-mob-floor-3-b.json` | `RPGSystem/` | `RPGSystem/rpg_loc` |
    | `Skills_Enemy-mowe.json` | `Skills` | `Skills_Enemy` |
    | `Skills_Abnormality-a1c9p1.json` | `Skills` | `Skills_Abnormality` |
    | `Bufs-walpu7.json` | `Bufs` | `""` |
    | `Bufs_Mirror5.json` | `Bufs` | `Bufs_Mirror` |
    | `StageNode91-a1c5.json` | `StageNode` | `""` |
    | `MirrorDungeonUI_5.json` | `MirrorDungeonUI` | `""` |
    | `Walpu6EventText.json` | `Walpu` | `Walpu6EventText` |

  - 另寫 `TestFileNodeOddNames`：

    | 輸入 | Top | Sub |
    |---|---|---|
    | `kr.json` | `kr` | `""` |
    | `reward.json` | `reward` | `""` |
    | `Ch1.json` | `Ch` | `""` |
    | `9Lives.json` | `9Lives` | `""` |

    開頭不是英文字母時，`Top` 取 `-` 之前的整段。
- [ ] **1.2** 執行 `go test ./internal/rules/ -run 'TestFileNode'`，預期 FAIL（undefined: FileNode）。
- [ ] **1.3 實作 `FileNode`**，規則照規格「分群樹的推導」：
  - 目錄檔：
    - `StoryData` 的子群取 `^\d*([A-Za-z]+)`；
    - 其他目錄取以 `_`、`-` 切開的前兩段，用 `_` 接起來。
  - 根目錄檔：`Top` 是開頭的連續英文字母；`Sub` 是 `-` 之前的部分，去掉結尾的 `[\d_]+`，等於 `Top` 時為 `""`。
- [ ] **1.4 測試先行**：`TestFileTag`，用 `testdata` 角色檔，或在測試裡建 `Rules{Characters: …}`：

  | 輸入 | 預期 |
  |---|---|
  | `StoryData/P10702.json` | `{character, heathcliff, 希斯克里夫}` |
  | `PersonalityVoiceDlg/Voice_Meursalut_X_10501.json` | `{character, meursault, …}` |
  | `Skills_personality-07.json` | `{character, heathcliff, …}` |
  | `Skills_Ego_Personality-12.json` | `{character, gregor, …}` |
  | `StoryData/S1037B.json` | `{chapter, "10", "第 10 章"}` |
  | `StoryData/S001A.json` | `{chapter, "0", "第 0 章"}` |
  | `StoryData/5D302B.json` | `{chapter, "5", "第 5 章"}` |
  | `AbEvents-a1c5p3.json` | `{chapter, "5", "第 5 章"}` |
  | `Skills_Abnormality-tkt-re.json` | `{event, "tkt", "tkt"}` |
  | `Bufs-walpu7.json` | `{event, "walpu7", "walpu7"}` |
  | `Skills_personality-x1p1c1.json` | `{event, "x1p1c1", …}` |
  | `Skills_personality-a1c9p1.json` | `{chapter, "9", …}` |

  回傳 `ok == false` 的：`Bufs.json`、`StageNode91-1.json`（純數字）、`StoryData/E302B.json`。
- [ ] **1.5** 先看 1.4 失敗，再實作。
  - `CharacterByFile` 加 `^Skills_(?:personality|Ego_Personality)-(\d\d)$`，比對 `Code`。
  - `FileTag` 依序判斷角色 → 章節 → 活動。
  - 章號去掉開頭的 0（`"00"`→`"0"`）。
- [ ] **1.6** `go test ./internal/rules/` 通過，並跑完整檢查，然後 commit。

### Task 2: 族群改由節點組成、新的路由

**Files:**
- Modify：
  - `internal/rules/routing.go`、`internal/rules/lint.go`、`internal/rules/rules.go`（若有讀 `Match`）
  - `cmd/lct/rules.go`（呼叫新的路由警告）
  - `guides/routing.yaml`（照規格附錄）、`guides/families.md`
  - `glossary/terms.yaml`（兩條 `family:walpu-refraction`）
- Modify：所有測試用的路由（`grep -rn "match:" --include=*.go --include=*.yaml internal cmd testdata`），改成 `nodes`
- Test：`internal/rules/rules_test.go`

**Interfaces:**
- Consumes：Task 1 `FileNode`。
- Produces：
  - `FamilyRule.Nodes []string`（yaml `nodes`），取代 `Match`；
  - `Family(file)`、`GlossaryWarnOnly(file)`、`DefaultFamily()` 的簽名不變；
  - `func LintRoutingWarnings(root string, krFiles []string) []LintError`。

- [ ] **2.1 測試先行**：`TestFamily` 改寫成用 `nodes` 的路由。斷言：
  - 子群優先：`Skills_personality-01.json` 的 `sub` 由 `ally` 的 `"Skills_personality"` 取得；`top` 雖然被 `enemy` 的 `"Skills"` 符合，也不採用。
  - 同層完整名稱優先：`BossRaidUI.json` 歸 `events`（完整名稱 `"BossRaidUI"`），而不是 `ui`（`"*UI"`），即使 `ui` 列在 `events` 前面。
  - 同層都是萬用字元：`MirrorDungeonUI_5.json` 取先列的 `mirror`。
- [ ] **2.2 測試先行**：
  - `TestUnlistedTopFallsInUnsorted`：路由沒有任何樣式符合 `FooBar-tkt.json` → `Family` 回 `"unsorted"`，`DefaultFamily()` 是 `"unsorted"`。
  - `TestRoutingLint`：
    - `match` 欄位：錯誤，訊息含 `nodes`；
    - `nodes: []` 的族群不在最後，或有兩個：錯誤；
    - `nodes` 樣式寫錯：錯誤；
    - `glossary` 值不對：錯誤。
- [ ] **2.3 測試先行**：`TestLintRoutingWarnings`，給 `krFiles`：
  - 樣式比對不到任何節點：警告；
  - 有頂層節點的檔落在 `unsorted`：警告，訊息列出節點；
  - 同層兩族群搶同一節點：警告，訊息寫出取了哪一個。
- [ ] **2.4** 實作。
  - **`match` 的偵測**：用 `yaml.Node` 檢查每個族群有沒有 `match` 鍵，有就由 lint 報錯；載入時把 `match` 當作不存在。
  - **`cmd/lct/rules.go`**：在 `LintScopeWarnings` 旁邊加上 `rules.LintRoutingWarnings(ctx.Root, krPackFiles(ctx.Root))`。
- [ ] **2.5 換路由**：
  - `guides/routing.yaml` 照規格附錄逐字照抄族群與 priority。檔頭註解改寫成 `nodes` 的說明，內容要涵蓋：節點怎麼推導、比對順序、`unsorted`、`glossary` 的意思。
  - `guides/families.md`：
    - 新增「族群與節點」一節，每個族群一行，寫文字類型與節點；
    - 「對於既有文本翻譯優先序／決定」改成新 priority；
    - 刪掉說「未列出的族群」那張表的族群歸屬欄，保留命名規律。
  - `glossary/terms.yaml`：`흐트러짐 구간`、`키워드` 的 `not_files: ["family:walpu-refraction"]` 改成 `["*walpu*", "*Refraction*", "Railway*"]`，`notes` 加「2026-11-06 族群改由節點組成，改寫成原族群的樣式，範圍不變」。
- [ ] **2.6** 跑完整檢查與 `bin/lct rules lint`（repo 內沒有 raw，只驗錯誤）。
- [ ] **2.7 真實資料**：在 scratch clone 跑 `bin/lct rules lint --json`。
  - 寫入報告：每個族群的檔數（`index.json` 或自寫的小程式皆可，程式不 commit），和規格的表比對。
  - `unsorted` 必須是 0，同層衝突的警告清單全文附上。
- [ ] **2.8** commit。

### Task 3: 詞彙表範圍的新寫法與批次的 `units`

**Files:**
- Modify：`internal/rules/glossary.go`（`matchScope`）、`internal/rules/lint.go`（錯誤與 `LintScopeWarnings`）
- Modify：`internal/batch/batch.go`（`glossaryFor` 給每個條目 `units`）
- Modify：`schemas/batch.schema.json`、`prompts/translate.md`、`.claude/skills/translate-pending/SKILL.md`（若提到範圍規則）、`glossary/README.md`、`cmd/lct/contract_test.go`
- Test：`internal/rules/rules_test.go`、`internal/batch/batch_test.go`

**Interfaces:**
- Consumes：Task 1 `FileNode`、`FileTag`。
- Produces：
  - 常數 `GroupPrefix = "group:"`、`ChapterPrefix = "chapter:"`、`CharacterPrefix = "character:"`、`EventPrefix = "event:"`；
  - `rules.Term.Units []string`（yaml `-`、json `units,omitempty`），只在批次裡填。

- [ ] **3.1 測試先行**：`TestAppliesScopeTokens`。
  - `group:Skills_Enemy*`：命中 `Skills_Enemy-mowe.json`，不命中 `Skills_personality-01.json`。
  - `group:StoryData/`：命中 `StoryData/S1037B.json`。
  - `chapter:10`：命中 `StoryData/S1037B.json`，不命中 `StoryData/S937A.json`。
  - `character:heathcliff`：命中 `PersonalityVoiceDlg/Voice_Heathcliff_RCorp4_10703.json`。
  - `event:walpu*`：命中 `Bufs-walpu7.json`，不命中 `Bufs-tkt.json`。
  - `not_files` 裡的這些寫法照樣排除。
- [ ] **3.2 測試先行**：`TestLintGlossaryScopeTokens`。
  - 錯誤：`chapter:x`、`chapter:-1`、`character:nobody`、`group:[`、`event:[`、`group:`（空）。
  - `LintScopeWarnings` 對比對不到任何檔的 `group:Nope`、`chapter:42`、`event:zzz` 給警告。
- [ ] **3.3** 實作 3.1、3.2。`LintScopeWarnings` 要讀角色檔，做法和 `Lint` 相同。
- [ ] **3.4 測試先行**：`TestBatchGlossaryUnits`。
  - 一個批次有兩個單位：KR 都含 `보스`，一個在 `Skills_Enemy-x.json`，一個在 `Skills_personality-01.json`。
  - 條目是 `보스 → BOSS`，`files: ["group:Skills_Enemy*"]`。
  - 預期批次 `glossary` 裡這個條目的 `units` 只有第一個單位的 key。
  - 被長條目蓋住的單位也不列。
- [ ] **3.5 實作**：`glossaryFor` 記下每個條目被哪些單位的 `TermsIn` 回傳，寫進 `Units`。暫定詞相同。
- [ ] **3.6 文件與 schema**：
  - **`prompts/translate.md`**：「`glossary` 條目的 `units` 列出要用這個條目的單位，已經算進 `files`、`not_files`（含 `family:`、`group:`、`chapter:`、`character:`、`event:`）與長詞條優先；不在 `units` 的單位不必用，即使 KR 含這個詞」。原本逐項解釋範圍與長詞條優先的三段改成這一段。
  - **`schemas/batch.schema.json`**：glossary item 加 `units`（字串陣列）。
  - **`glossary/README.md`**：範圍一節加四種寫法與例子。
  - **`cmd/lct/contract_test.go`**：釘住 `units` 那句；移除對舊段落的釘。
- [ ] **3.7** 跑完整檢查與 `TestSchemasMatchTheTypes`，然後 commit。

### Task 4: 詞彙表預設只警告、`check: hard`

**Files:**
- Modify：
  - `internal/rules/glossary.go`（`Term.Check`、`termFields`）、`internal/rules/lint.go`
  - `internal/batch/submit.go`
  - `schemas/batch.schema.json`
  - `prompts/translate.md`、`.claude/skills/translate-pending/SKILL.md`、`.claude/skills/review-page/SKILL.md`
  - `glossary/README.md`、`README.md`、`cmd/lct/contract_test.go`
- Test：`internal/batch/batch_test.go`、`internal/rules/rules_test.go`

**Interfaces:**
- Produces：
  - `rules.Term.Check string`（yaml/json `check`，`omitempty`）；
  - 常數 `CheckHard = "hard"`、`CheckWarn = "warn"`；
  - `func (t Term) Hard() bool`。

- [ ] **4.1 測試先行**：`TestSubmitGlossaryCheck`，四種情況：
  1. 條目沒寫 `check`，hard 族群，缺譯名，note 空：收下，結果的 `Warnings` 含 `term-missing`。
  2. `check: hard`，hard 族群，缺譯名，note 沒寫原詞：退回（`term-missing` 為 hard）。note 寫了原詞：收下。
  3. `check: hard`，warn 族群：收下並警告。
  4. 譯文含 `forbidden`：照舊退回。

  另外，暫定詞（`Provisional`）缺譯名：收下並警告。
- [ ] **4.2** 現有預期「缺譯名就退回」的送出測試，改成在條目寫 `check: hard`。新測試要先看到失敗。
- [ ] **4.3 實作**：
  - `submit.go` 只在「issue 背後的條目 `Hard()`」且 `!warnOnly` 且 `!noteNames` 時升為 hard。找條目的方式同 `noteNames`：比對 `KR`、`ZH`。
  - lint：`check` 只能是 `""`、`hard`、`warn`。
- [ ] **4.4 文件與 skill**：
  - **`prompts/translate.md`**：「譯文缺少 `zh` 會被退回」那段改成：
    - 詞彙表的譯名是預設；
    - 語境不合可以不用，在 `note` 寫 KR 原詞與理由；
    - 只有 `check: "hard"` 的條目缺譯名會退回（`glossary_check: "warn"` 的批次除外），`note` 寫出原詞就收下。
  - **`.claude/skills/translate-pending/SKILL.md`**：同步，若有提到退回條件。
  - **`.claude/skills/review-page/SKILL.md`**：`glossary/…` ID 的決定多兩種：
    - 「改為退回」：在條目加 `check: hard`，`notes` 寫日期、應用率與出現數（取自 `bin/lct glossary scope --json` 的條目）；
    - 「改回只警告」：刪掉 `check`。
  - **`glossary/README.md`**：加 `check` 一節；**`README.md`**：同步相關句子。
  - **`cmd/lct/contract_test.go`**：釘住以上新句子，並加 mutation 案例。
- [ ] **4.5** 跑完整檢查與 `bin/lct rules lint`，然後 commit。

### Task 5: 詞彙範圍統計改用頂層節點、詞條命中單位、可考慮退回

**Files:**
- Modify：`internal/scope/scope.go`、`cmd/lct/glossary.go`（用法說明）、`site/src/types.ts`、`site/src/review/ScopeView.tsx`（種類名稱與說明）、`.claude/skills/review-page/SKILL.md`（建議的寫法）、`cmd/lct/contract_test.go`
- Test：`internal/scope/scope_test.go`、`site/src/review/ScopeView.test.tsx`

**Interfaces:**
- Consumes：Task 1 `FileNode`；Task 3 `GroupPrefix`；Task 4 `Term.Hard()`。
- Produces：
  - `GroupDir` 改名 `GroupNode = "group"`，群組名是 `group:<Top>`；
  - `Dirs`、`RestGroup` 移除；
  - `Thresholds` 加 `HardRate float64`（json `hard_rate`，0.95）、`HardMinN int`（json `hard_min_n`，20）；
  - `Entry` 加 `Check string`（json `check,omitempty`）、`N int`、`Used int`、`Rate float64`、`HardCandidate bool`（json `n`、`used`、`rate`、`hard_candidate`）。這幾個數就是 `Report.Units` 裡該條目的單位：實際命中，substring 條目的詞中詞也算（2026-11-06 執行時的判斷，取代原本「族群群組加總、不含詞中詞」）。
  - `Report.Units map[string][]EntryUnit`（json `-`），鍵是 `Entry.ID`：

```go
type EntryUnit struct {
	Key, File, KR, ZH, Origin, Note string
	Used bool
}
```

  `Report.Units` 只列真正命中的單位，命中的判斷和審核頁相同。

- [ ] **5.1 測試先行**：
  - 現有的分組測試改用頂層節點：`StoryData/` 的檔歸 `group:StoryData/`，`Skills_Enemy-x.json` 歸 `group:Skills`。
  - `TestProposalPatternsMatchFiles` 改成只接受 `family:` 與 `group:`。
- [ ] **5.2 測試先行**：`TestEntryRateAndHardCandidate`。
  - 出現 20、用了 20：`Rate == 1`、`HardCandidate`。
  - 出現 20、用了 19（0.95，不高於 0.95）：不是候選。
  - 出現 19、全用：不是候選。
  - 條目已是 `check: hard`：不是候選。
- [ ] **5.3 測試先行**：`TestReportUnits`。
  - 被長條目蓋住的單位不在 `Units`。
  - `Used` 和 `Term.Used` 一致。
  - word 條目的詞中詞不列。
- [ ] **5.4** 實作。也更新：
  - `lct glossary scope` 的用法說明；
  - ScopeView 的群組種類文字：「目錄」改「分群」；
  - review-page skill 建議範例裡的 `StoryData/**` 寫法改成 `group:`。
- [ ] **5.5** 跑完整檢查與 site 檢查，然後 commit。

### Task 6: 網站資料：分群樹、族群表、`data/glossary/`

**Files:**
- Modify：`internal/site/file.go`（`FileSummary`）、`internal/site/export.go`（`Index`、`IndexFamily`、新檔）
- Create：`internal/site/glossary.go`
- Test：`internal/site/export_test.go`、`internal/site/glossary_test.go`

**Interfaces:**
- Consumes：Task 1 `FileNode`、`FileTag`、`Tag`；Task 5 `scope.Report`（`Entries`、`Thresholds`、`Units`）；`itemid` 的 term-missing 單位 ID 函式（和審核頁用的同一個）；`rules/accepted.yaml` 的確認。
- Produces：

```go
// FileSummary gains:
Top string    `json:"top"`
Sub string    `json:"sub,omitempty"`
Tag *rules.Tag `json:"tag,omitempty"`

// IndexFamily gains:
Glossary string   `json:"glossary"` // "hard" or "warn"
Nodes    []string `json:"nodes"`

// Index gains:
Groups []IndexGroup `json:"groups"`

type IndexGroup struct {
	Kind     string       `json:"kind"`  // "top", "sub", "chapter", "character", "event"
	Node     string       `json:"node"`  // node name or tag value
	Label    string       `json:"label"` // node name or Tag.Label
	Files    []int        `json:"files"` // files directly in this group
	Children []IndexGroup `json:"children"`
}
```

  - **`data/glossary/index.json`**：`{"thresholds": scope.Thresholds, "entries": []scope.Entry}`。
  - **`data/glossary/<a>-<b>.json`**：條目 ID `glossary/<a>/<b>` 對應的檔，內容是 `{"id": string, "missing": []GlossaryUnit, "used": []GlossaryUnit}`。

```go
type GlossaryUnit struct {
	Key      string          `json:"key"`
	Family   string          `json:"family"`
	Top      string          `json:"top"`
	Sub      string          `json:"sub,omitempty"`
	Tag      *rules.Tag      `json:"tag,omitempty"`
	Origin   string          `json:"origin"`
	KR       string          `json:"kr"`
	ZH       string          `json:"zh"`
	Note     string          `json:"note,omitempty"`
	ID       string          `json:"id,omitempty"`       // missing only: warning/term-missing/…
	Accepted *suggest.Confirmed `json:"accepted,omitempty"` // missing only
}
```

- [ ] **6.1 測試先行**：`TestIndexGroups`，用 6 個假檔：
  - `StoryData/S1037B`、`StoryData/S937A`、`StoryData/P10702`、`Skills_Enemy-a`、`Skills_Enemy-b`、`Bufs-walpu7`。
  - 斷言：
    - 頂層依檔數多到少排；
    - `Skills` 只有一個子群 `Skills_Enemy`，所以不顯示子群，檔直接列在 `Skills` 下；
    - `StoryData` 底下 `P` 只有一個檔、收進頂層後只剩 `S` 一個子群，所以不顯示子群，`chapter` 子層「第 9 章」「第 10 章」直接掛在頂層下，依章號由小到大（執行時依規格更正）；
    - 只有一個檔的子群不顯示；
    - 第三層即使只有一個檔也顯示。
- [ ] **6.2 測試先行**：`TestIndexFamiliesCarryNodes`：每個族群帶 `glossary` 與 `nodes`。`glossary` 省略的族群輸出 `"hard"`。
- [ ] **6.3 測試先行**：`TestGlossaryFiles`。
  - `missing` 的 KR、TW 保留原文；超過 400 字時，改用 `validate.MatchText` 後的前 400 字加「…」。
  - `used` 的 KR、TW 是 `validate.MatchText` 後的片段：
    - KR 以第一處原詞為中心，前後各 30 字；
    - TW 以第一處譯名（`zh` 或 `also`）為中心，前後各 30 字；
    - 截斷的一邊加「…」。
  - `missing` 單位有 `id`；在 `rules/accepted.yaml` 的帶 `accepted`。
  - 檔名由條目 ID 換算。
- [ ] **6.4** 實作，並接進 `Export`。`Result` 加 `GlossaryBytes int64`（json `glossary_bytes`）。
- [ ] **6.5** 跑完整檢查，然後 commit。

### Task 7: 前端：分群樹側欄、首頁族群表

**Files:**
- Modify：`site/src/types.ts`、`site/src/data.ts`（若需要）、`site/src/app/Directory.tsx`、`site/src/app/Home.tsx`、`site/src/styles.css`
- Test：`site/src/app/Directory.test.tsx`、`site/src/app/App.test.tsx`（首頁），以及 fixtures（`site/src/testdata.ts` 或 fixtures 目錄裡的 `index.json`）

**Interfaces:**
- Consumes：Task 6 的 `index.json`（`groups`、`families[].glossary`、`families[].nodes`、`files[].top/sub/tag`）。

- [ ] **7.1 測試先行**：`Directory.test.tsx`。
  - 依 `groups` 畫三層，每層顯示檔數（含子層）。
  - 「opens every level above a hit」：檔案篩選命中的檔所在的每一層都展開；沒命中的層不顯示。
  - 「opens the levels of the current file」：`current` 換到另一個檔時，它的上層展開，原本展開的不收起。
  - 「章節」「角色」「活動」三個下拉選單：
    - 選了之後只顯示有該標記的檔，並展開到檔；
    - 選單選項取自 `files[].tag`：章節依數字，角色依 `label`，活動依字母；
    - 三個選單同時只用一個，選一個會清掉另兩個。
- [ ] **7.2 測試先行**：首頁的族群表。欄位是族群、glossary（hard／warn 標記）、節點（code 列表）、檔數。順序依 `index.families`。
- [ ] **7.3** 實作。版面沿用現有 `dir-*` 樣式：
  - 每層用 `button[aria-expanded]` 切換；
  - 第三層與子群縮排不同；
  - 寬 400px 不橫向捲動。
- [ ] **7.4** 跑 site 檢查與完整檢查。用 Playwright 截寬 1280 與 400 的圖存在 scratchpad，報告附 `scrollWidth`／`clientWidth`。然後 commit。

### Task 8: 前端：「詞彙」頁

**Files:**
- Create：`site/src/glossary/GlossaryView.tsx`（列表）、`site/src/glossary/TermView.tsx`（詳情）、`site/src/glossary/GlossaryView.test.tsx`、`site/src/glossary/TermView.test.tsx`
- Modify：`site/src/route.ts`（＋測試）、`site/src/data.ts`（載入並快取 `glossary/index.json` 與條目檔）、`site/src/types.ts`、`site/src/app/App.tsx`、`site/src/app/Header.tsx`、`site/src/styles.css`

**Interfaces:**
- Consumes：Task 6 的 `data/glossary/*`；現有 `IdCopy`、`UnitLink`、`OriginBadge`、`Row`（`mark` 參數）、`ItemList` 的分頁做法。
- Produces：兩種路由。

```ts
| { kind: 'glossary'; q: string; sort: 'missing' | 'kr' | 'rate'; only: '' | 'missing' | 'hard'; family: string }
| { kind: 'term'; id: string; family: string; group: string; chapter: string; character: string; event: string; origin: string; q: string }
```

  - 網址：
    - `#/g?q=&sort=&only=&family=`；
    - `#/g/<a>/<b>?family=&group=&chapter=&character=&event=&origin=&q=`，`id` 為 `glossary/<a>/<b>`；
    - `glossaryHref(…)`、`termHref(id, …)`。

- [ ] **8.1 測試先行**：`route.test.ts` 解析與產生上面兩種網址，並來回一致。不合格的 `#/g/x` 回首頁。
- [ ] **8.2 測試先行**：`GlossaryView.test.tsx`。
  - 列出全部條目，欄位：原詞、譯名（含 `also`）、`check`、範圍摘要、出現、用了、沒用、應用率（百分比，一位小數）。
  - 預設依沒用數由多到少排；可以切換依原詞或應用率排序。
  - `only=hard` 只列 `hard_candidate`，並顯示門檻句「應用率高於 95%、出現至少 20 個單位」。數字取自 `thresholds`，改 fixture 的值，文字跟著變。
  - 搜尋比對原詞、譯名、also、notes。
  - 每列連到 `termHref`。
- [ ] **8.3 測試先行**：`TermView.test.tsx`。
  - 上方：條目內容、各族群與各分群的出現／用了表、到 `#/r/warnings?q=<termgroup id>` 與 `#/r/scope?id=<entry id>` 的連結、條目的「複製 ID」（`glossary/…`）。
  - 「沒用譯名」區：
    - 全部單位，每個單位附 KR（標出原詞）、TW、來源、note、已確認標記、連結、「複製 ID」；
    - 「複製這些單位」把目前篩選下沒確認的單位，以 `ID「TW 片段」→ ` 一行一個複製。
  - 「用了譯名」區：
    - 先顯示 100 個，「再顯示 100 個」；
    - 換任何篩選都回到 100 個。
  - 篩選：族群、分群（頂層）、章節、角色、活動、來源、文字。兩區同時套用。
  - 網址的 ID 不在 `index.json`：顯示「找不到詞條 <id>」。
  - 條目檔載入失敗：顯示錯誤訊息，不出錯。
- [ ] **8.4 實作**：頂端加「詞彙」連結，位置在「審核」後。
- [ ] **8.5** 跑 site 檢查與完整檢查，用真實資料截圖（寬 1280、400，列表與 대상 詳情）。報告附 `scrollWidth`／`clientWidth`，以及 대상 條目檔的大小與載入時間。然後 commit。

### Task 9: 真實資料驗收（只量測，不改 repo）

在 scratch clone，以 main（改前，`git merge-base origin/main HEAD`）與分支 HEAD 各跑 `lct site`，寫報告 `task-9-report.md`。

- [ ] **9.1 族群**：
  - 每個族群的檔數、單位數與規格的表比對；
  - `unsorted` 必須是 0；
  - `lct rules lint` 0 錯誤；
  - 列出全部警告。
- [ ] **9.2 glossary 模式**：族群 `glossary` 從 hard 變 warn 的檔，清單與規格的 83 個比對。
- [ ] **9.3 已存的決定**：
  - `rules/declined.yaml` 的 14 筆 `inconsistent/…` 在分支的建議裡仍是「已拒絕」，沒有重新出現；
  - 審核頁五個分頁的數字改前改後對照，差異逐項說明。
- [ ] **9.4 詞彙**：
  - term-missing 群組數與單位數改前改後對照（預期不變，只有族群歸屬變）；
  - 詞彙範圍建議數與種類對照；
  - 「可考慮退回」的條目數。
- [ ] **9.5 「詞彙」頁**：
  - `data/glossary/` 總大小與最大一檔的大小；
  - 대상 詳情頁的載入時間；
  - 寬 400px 的 `scrollWidth`。
- [ ] **9.6 時間**：`lct site` 改前改後各跑三次。

---

## 我在計畫中的判斷

1. **批次條目加 `units`。** 新的範圍寫法（`group:`、`chapter:`……）譯者無法自己判斷。所以批次直接列出每個條目要用在哪些單位，提示詞改成看 `units`。這也取代了原本要譯者自己判斷 `files`、`family:` 與長詞條優先的三段說明。代價：批次檔變大一點，contract test 要改。
2. **暫定詞也只警告。** 暫定詞沒有 `check`，跟著新預設。
3. **第三層的顯示。** 第三層即使只有一個檔也顯示（它有意義）；子群只有一個檔時合併。
4. **「詞彙」頁的連結。**
   - 詳情連到審核頁時，用 `#/r/warnings?q=<termgroup id>`；
   - 連到詞彙範圍時，用 `#/r/scope?id=<entry id>`（現有的深連結）。
5. **`Entry.N`、`Entry.Used`、`Entry.Rate`** 就是 `Report.Units` 裡該條目的單位數（實際命中；substring 條目的詞中詞也算），「詞彙」頁的列表數字、詳情與「可考慮退回」用同一組數字（執行時改定，原為族群群組加總）。
