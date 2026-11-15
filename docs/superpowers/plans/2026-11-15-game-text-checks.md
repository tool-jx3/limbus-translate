# 依遊戲行為的譯文檢查 實作計畫

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 `validate.Check` 新增 13 個只當警告的檢查代碼，讓依遊戲程式推算會顯示錯的現行譯文出現在審核頁的檢查警告分頁與建置報告。

**Architecture:** 只看單一單位的檢查直接加進 `internal/validate`，靠 `Context` 新增的 `Pointer` 判斷欄位。`keyword-link` 與 `level-diff` 需要整包資料：`validate.Game` 是一份純資料（狀態名稱表、共鳴字串、上一級文字），由新套件 `internal/gametext` 從 KR 檔與翻譯紀錄建出來，建置與網站匯出各建一次、放進 `Context.Game`。網站只在 `names.ts` 補名稱。

**Tech Stack:** Go（`internal/validate`、新的 `internal/gametext`、`internal/build`、`internal/site`、`internal/batch`、`cmd/lct`）；Preact 網站（`site/src/review/names.ts`）。

**Spec:** `docs/superpowers/specs/2026-11-15-game-text-checks-design.md`（使用者 2026-11-15 核可）。執行者要讀 spec 與本計畫；演算法的依據是 `docs/research/2026-11-15-game-text-logic/03-keywords.md`，`method/kwsim.py` 是可對照的 Python 重現。

## Global Constraints

- 新代碼全部是 `validate.ClassStyle`（警告），不擋任何流程
- 既有檢查的代碼、訊息、順序不變；新問題接在既有的後面；既有警告的 ID 不變
- 訊息不可以寫成確定的事實：涉及遊戲行為的句子要帶 `by the game's code`（訊息照既有檢查用英文）
- 只用 `raw`（KR）與翻譯紀錄算得出來的資料，不讀遊戲檔案
- 不新增 TypeScript 的檢查邏輯；`site/src/review/marks.ts` 不動
- 不處理 `excludeAutoLink` 的新增、不檢查「KR 有連結而譯文沒有」、不改 `guides/`
- 沒有 `Pointer` 時，所有要看欄位的新檢查略過；沒有 `Game` 時 `keyword-link`、`level-diff` 略過；都不可以出錯
- 提交前 `go vet ./... && go test ./...`；動到 `site/` 時 `cd site && npm ci && npm test && npm run build && npm run typecheck`
- commit 訊息結尾加 session 規定的兩行；程式碼、文件不寫模型名稱
- 不碰 `.work/`（唯讀使用）；`out/` 不進版控

## Review Focus

1. **名稱表裡的空名稱或 `Unknown`**：狀態的譯名是空字串、`Unknown`，或沒有可用紀錄。預期：空名稱不參加比對（不可以變成「到處都符合」），沒有紀錄的取 KR 名稱。→ Task 2 的 `TestGameNames`。
2. **多位元組字元的位置**：名稱前後是中文、全形空格、全形數字、U+200B。預期：以字元（rune）判斷前後字元，不是位元組。→ Task 2 的 `TestLinkedAfterChar`。
3. **名稱本身含方括號或符號**（`本國劍術[肉]`、`攻擊等級+`）。預期：照字面比對；在有標籤處理的欄位，`[肉]` 先被換掉，所以含它的名稱比對不到。→ Task 2 的 `TestLinkedTagPass`。
4. **子資料夾裡同名族群的檔案**（`RPGSystem/Skills-x.json`）或 `AbEventsResultLog.json`。預期：不當成 `Skills`／`AbEvents`。→ Task 1 的 `TestFamilyAndField`。
5. **`levelList` 形狀異常**：`level` 重複、不是數字、硬幣數量兩級不同、上一級沒有可用紀錄。預期：不檢查、不出錯。→ Task 3 的 `TestLevelPairs`。

---

## 檔案

| 檔案 | 責任 |
|---|---|
| `internal/validate/gamechecks.go`（新） | 族群與欄位的判斷；11 個只看單一單位的檢查；`checkGame` 把新問題接到 `Check` 後面 |
| `internal/validate/keyword.go`（新） | `Game`、`NewGame`、名稱比對、`keyword-link` |
| `internal/validate/leveldiff.go`（新） | `level-diff` |
| `internal/validate/validate.go` | 新代碼常數、`Context.Pointer`、`Context.Game`、`Check` 呼叫 `checkGame` |
| `internal/gametext/gametext.go`（新） | 從 KR 檔與紀錄建 `validate.Game`；`UsedText` |
| `internal/build/build.go`、`internal/site/review.go` | 建 `Game`，`Check` 帶上 `Pointer`、`Game` |
| `internal/batch/*.go`、`cmd/lct/adjust.go`、`cmd/lct/validate.go`、`internal/bootstrap/bootstrap.go` | 知道指標的呼叫處帶上 `Pointer` |
| `site/src/review/names.ts` | 13 個代碼的名稱與說明 |

---

### Task 1: 只看單一單位的 11 個檢查

**Files:**
- Create: `internal/validate/gamechecks.go`、`internal/validate/gamechecks_test.go`
- Modify: `internal/validate/validate.go`（常數、`Context`、`Check`、開頭列舉代碼的註解）

**Interfaces:**
- Produces:
  - `Context.Pointer string`（單位的 RFC 6901 指標；`""` 表示不知道）
  - 常數：`CodeBracketText = "bracket-text"`、`CodePageCount = "page-count"`、`CodeStoryComment = "story-comment"`、`CodeStrayBrace = "stray-brace"`、`CodeRegexChar = "regex-char"`、`CodeLiteralNewline = "literal-newline"`、`CodeReservedValue = "reserved-value"`、`CodeNewlineSpace = "newline-space"`、`CodeQmarkCount = "qmark-count"`、`CodeBulletCount = "bullet-count"`、`CodeRubyForm = "ruby-form"`、`CodeKeywordLink = "keyword-link"`、`CodeLevelDiff = "level-diff"`
  - `func family(file string) string`：根目錄的檔回傳「去掉 `.json` 後到第一個 `-` 或 `_` 之前」；路徑含 `/` 的回傳 `""`
  - `func fieldPath(pointer string) string`：指標去掉開頭的 `dataList` 與所有純數字的節，其餘以 `/` 相連（`/dataList/3/levelList/1/coinlist/0/coindescs/2/desc` → `levelList/coinlist/coindescs/desc`）
  - `func checkGame(kr, zh string, ctx Context) []Issue`：Task 2、3 在裡面各加一個呼叫

- [ ] **Step 1: 寫失敗的測試** `internal/validate/gamechecks_test.go`

```go
func TestFamilyAndField(t *testing.T) {
	for _, c := range []struct{ file, want string }{
		{"Skills_personality-01.json", "Skills"}, {"Passive.json", "Passive"}, {"Passives-a1c9p1.json", "Passives"},
		{"AbEventsResultLog.json", "AbEventsResultLog"}, {"AbEvents-a1c5p1.json", "AbEvents"},
		{"RPGSystem/Skills-x.json", ""}, {"StoryData/S001A.json", ""},
	} {
		if got := family(c.file); got != c.want {
			t.Errorf("family(%q) = %q, want %q", c.file, got, c.want)
		}
	}
	for _, c := range []struct{ pointer, want string }{
		{"/dataList/3/levelList/1/coinlist/0/coindescs/2/desc", "levelList/coinlist/coindescs/desc"},
		{"/dataList/0/simpleDesc/1/simpleDesc", "simpleDesc/simpleDesc"}, {"/dataList/7/name", "name"}, {"", ""},
	} {
		if got := fieldPath(c.pointer); got != c.want {
			t.Errorf("fieldPath(%q) = %q, want %q", c.pointer, got, c.want)
		}
	}
}

// codesOf returns the codes of the game checks Check reports, in order.
func codesOf(kr, zh, file, pointer string) []string

func TestGameChecksPerUnit(t *testing.T) {
	const skill = "/dataList/0/levelList/0/desc"
	for _, c := range []struct{ name, file, pointer, kr, zh string; want []string }{
		{"bracket text", "Passives.json", "/dataList/0/desc", "본국검술 사용", "使用本國劍術[肉]", []string{CodeBracketText}},
		{"an [ID] is no bracket text", "Passives.json", "/dataList/0/desc", "[Laceration] 1", "1層 [Laceration]", nil},
		{"KR has the bracket too", "Skills.json", skill, "사영 [루주]", "映射 [赤紅]", nil},
		{"brackets outside the processed fields", "Bufs.json", "/dataList/0/desc", "설명", "說明[肉]", nil},
		{"fewer pages", "AbEvents.json", "/dataList/0/successDesc/0", "가\n\n나\n\n다", "甲\n\n乙", []string{CodePageCount}},
		{"more pages", "ActionEvents.json", "/dataList/0/options/1/result/0", "가", "甲\n\n乙", []string{CodePageCount}},
		{"same pages", "AbEvents.json", "/dataList/0/eventDesc", "가\n\n나", "甲\n\n乙", nil},
		{"only result/0 is paged", "ActionEvents.json", "/dataList/0/options/1/result/1", "가", "甲\n\n乙", nil},
		{"not AbEventsResultLog", "AbEventsResultLog.json", "/dataList/0/content", "가", "甲\n\n乙", nil},
		{"story comment", "StoryData/S001A.json", "/dataList/0/content", "말", "話 // 註", []string{CodeStoryComment}},
		{"KR comment kept", "StoryData/S001A.json", "/dataList/0/content", "SE // 쾅", "音效 // 砰", nil},
		{"stray brace", "X.json", "/dataList/0/content", "{0} 획득", "獲得{0}}", []string{CodeStrayBrace}},
		{"regex char", "ChoiceEventEffect.json", "/dataList/0/content", "보상: {reward}", "獎勵({reward})", []string{CodeRegexChar}},
		{"full-width brackets are fine", "ChoiceEventEffect.json", "/dataList/0/content", "보상: {reward}", "獎勵（{reward}）", nil},
		{"literal newline", "X.json", "/dataList/0/content", "가", `甲\n乙`, []string{CodeLiteralNewline}},
		{"KR has it too", "X.json", "/dataList/0/content", `가\n나`, `甲\n乙`, nil},
		{"reserved value", "X.json", "/dataList/0/content", "미상", " unknown ", []string{CodeReservedValue}},
		{"Error", "X.json", "/dataList/0/content", "오류", "Error", []string{CodeReservedValue}},
		{"space before newline", "Personalities.json", "/dataList/0/title", "R사\n제4무리", "R公司 \n第四集團軍", []string{CodeNewlineSpace}},
		{"full-width space after newline", "Egos.json", "/dataList/0/name", "가\n나", "甲\n　乙", []string{CodeNewlineSpace}},
		{"no space", "Personalities.json", "/dataList/0/title", "가\n나", "甲\n乙", nil},
		{"other fields may", "Personalities.json", "/dataList/0/desc", "가\n나", "甲 \n乙", nil},
		{"sin type question marks", "Enemies.json", "/dataList/0/name", "탐식 죄종??", "暴食罪孽？？", []string{CodeQmarkCount}},
		{"kept", "Enemies.json", "/dataList/0/name", "탐식 죄종?", "暴食罪孽?", nil},
		{"other names use the full-width mark", "Enemies.json", "/dataList/0/name", "히스클리프?", "希斯克利夫？", nil},
		{"bullets", "StoryTheaterDanteNoteDetail.json", "/dataList/0/desc", "가\n- 나\n- 다", "甲\n- 乙\n丙", []string{CodeBulletCount}},
		{"short ruby", "StoryData/S001A.json", "/dataList/0/content", "말", "<r=讀>話</r>", []string{CodeRubyForm}},
		{"ruby without a reading", "StoryData/S001A.json", "/dataList/0/content", "말", "<ruby>話</ruby>", []string{CodeRubyForm}},
		{"upper case", "StoryData/S001A.json", "/dataList/0/content", "말", "<Ruby=讀>話</Ruby>", []string{CodeRubyForm}},
		{"a proper ruby", "StoryData/S001A.json", "/dataList/0/content", "말", "<ruby=讀>話</ruby>", nil},
	} {
		if got := codesOf(c.kr, c.zh, c.file, c.pointer); !slices.Equal(got, c.want) {
			t.Errorf("%s: codes = %v, want %v", c.name, got, c.want)
		}
	}
}

// Without a pointer the checks that look at the field are skipped; the others still run.
func TestGameChecksWithoutPointer(t *testing.T) {
	if got := codesOf("가\n\n나", "甲", "AbEvents.json", ""); got != nil {
		t.Errorf("codes = %v, want none", got)
	}
	if got := codesOf("가", `甲\n`, "", ""); !slices.Equal(got, []string{CodeLiteralNewline}) {
		t.Errorf("codes = %v, want literal-newline", got)
	}
}

// The new issues are warnings and come after the ones Check had before.
func TestGameChecksAreWarningsAndLast(t *testing.T) // Check("가...", "甲..." + `\n`, …): the last issue is literal-newline, Severity warn, Class style; ascii-ellipsis comes before it
```

`codesOf` 只回傳新代碼（上面 `Produces` 列的 13 個），忽略既有檢查的代碼。

- [ ] **Step 2: 跑測試確認失敗**

Run: `go test ./internal/validate/ -run 'TestFamilyAndField|TestGameChecks' -v`
Expected: 編譯失敗（`family` 等未定義）

- [ ] **Step 3: 實作**

`validate.go`：加常數與 `Context.Pointer`、`Context.Game *Game`（`Game` 型別在 Task 2 定義；這個 Task 先放 `type Game struct{}` 在 `keyword.go` 讓它能編譯），`Check` 最後 `issues = append(issues, checkGame(kr, zh, ctx)...)`，並更新開頭列舉 style 代碼的註解。

`gamechecks.go`：`family`、`fieldPath`、`checkGame` 與各檢查。規則照 spec §5 的表；spec 沒有寫死而這裡決定的：

- 範圍表（族群 → 欄位路徑）：
  - `bracket-text`：`Skills` 的 `levelList/desc`、`levelList/coinlist/coindescs/desc`、`levelList/coinlist/coindescs/summary`；`Passive`、`Passives` 的 `desc`、`summary`
  - `page-count`：`AbEvents` 的 `eventDesc`、`prevDesc`、`successDesc`、`failureDesc`；`ActionEvents` 的 `desc`，以及指標以 `/result/0` 結尾且欄位路徑是 `options/result` 的
  - `story-comment`：`file` 以 `StoryData/` 開頭且欄位路徑是 `content`
  - `newline-space`：`Personalities`、`PersonalityCompanion` 的 `title`；`Egos` 的 `name`
  - `qmark-count`：`Enemies` 的 `name`，且 KR 符合 `죄종\?+$`
  - `bullet-count`：`StoryTheaterDanteNote`、`StoryTheaterDanteNoteDetail` 的 `desc`
  - `regex-char`：`ChoiceEventEffect` 的 `content`
- `bracket-text`：`\[[^\]\n]*\]` 的全部比對中，不是 `idRe` 整段符合的個數，譯文比 KR 多才報；訊息列出譯文多出來的那幾個
- `page-count`：段數 ＝ `strings.Split(s, "\n\n")` 的長度；`s` 是 `""` 或 `"Error"` 時為 0。訊息 `%d pages where KR has %d`；譯文較少時加 `; by the game's code a custom language is padded with "..." pages`
- `stray-brace`：`placeholderRe.ReplaceAllString(s, "")` 後數 `{`、`}`
- `regex-char`：`strings.FieldsFunc(s, func(r rune) bool { return r == '{' || r == '}' })` 的每一段，以 `syntax.Parse("{"+p+"}", syntax.Perl)` 編譯；KR 有任何一段失敗就不檢查
- `reserved-value`：`strings.EqualFold(strings.TrimSpace(zh), "unknown")` 或 `strings.TrimSpace(zh) == "Error"`，且 KR 以同樣方式判斷不是
- `newline-space`：`\n` 前或後一個字元是 `' '` 或 `'　'`
- `bullet-count`：`strings.Count(s, "\n-")`
- `ruby-form`：譯文符合 `<r=|</r>|<ruby>|` 或「不分大小寫符合 `</?ruby` 但不是小寫的 `<ruby`／`</ruby`」
- 發出順序依 `Produces` 列的常數順序（`keyword-link`、`level-diff` 最後）

- [ ] **Step 4: 跑測試確認通過**

Run: `go test ./internal/validate/`
Expected: PASS（既有測試一併通過）

- [ ] **Step 5: Commit**

```bash
git add internal/validate && git commit -m "validate：依遊戲行為的單位檢查（11 個警告代碼）"
```

---

### Task 2: `keyword-link`

**Files:**
- Create: `internal/validate/keyword.go`、`internal/validate/keyword_test.go`
- Modify: `internal/validate/gamechecks.go`（`checkGame` 呼叫）

**Interfaces:**
- Consumes: Task 1 的 `family`、`fieldPath`、`Context.Pointer`、`Context.Game`、`CodeKeywordLink`
- Produces:

```go
// GameBuf is one entry of the Bufs* files.
type GameBuf struct {
	ID string
	KR string // the KR name
	ZH string // the name the pack ships: the record's text, or KR when the unit has no usable record
	ExcludeAutoLink bool
}

// LevelText is the text of the same field one uptie level lower.
type LevelText struct {
	Level  int // the lower level
	KR, ZH string
}

// GameInput is what NewGame builds a Game from.
type GameInput struct {
	Bufs []GameBuf // in file-name order, then document order
	// Resonance and PerfectResonance are the texts the pack ships for the ids
	// resonance and perfect_resonance of BattleUIText.json; "" when there is none.
	Resonance, PerfectResonance string
	// PrevLevel is, by unit key (units.Key), the text one level lower of a
	// Skills desc or coin desc; a unit without a lower level, or whose lower
	// level has no usable record, has no entry.
	PrevLevel map[string]LevelText
}

func NewGame(in GameInput) *Game

// Link is a name of the text that the game turns into a keyword link.
type Link struct {
	Start int    // byte offset of the name in the text
	Name  string
	IDs   []string // every id whose name it is, in the order of GameInput.Bufs
}

// Linked returns the names of text the game links, by its code: text is a
// field as written; tagPass says the field goes through the skill-tag pass first.
func (g *Game) Linked(text string, tagPass bool) []Link
```

- [ ] **Step 1: 寫失敗的測試** `internal/validate/keyword_test.go`

共用的測試資料：

```go
func testGame() *Game {
	return NewGame(GameInput{
		Bufs: []GameBuf{
			{ID: "Laceration", KR: "출혈", ZH: "流血"},
			{ID: "Vibration", KR: "진동", ZH: "震顫"},
			{ID: "VibrationExplosion", KR: "진동 폭발", ZH: "震顫引爆"},
			{ID: "AttackLevelAdder", KR: "공격 레벨+", ZH: "攻擊等級+"},
			{ID: "CheerUpXichun", KR: "원[援]", ZH: "援"},
			{ID: "ConfessionThirdPhase", KR: "〜", ZH: "～"},
			{ID: "ComboSkill", KR: "연", ZH: "連", ExcludeAutoLink: true},
			{ID: "ChoiSwordsmanship", KR: "본국검술[살]", ZH: "本國劍術[肉]"},
			{ID: "HardA", KR: "고난", ZH: "苦難"}, {ID: "HardB", KR: "고난", ZH: "苦難"},
			{ID: "Wrath", KR: "분노", ZH: "憤怒"},
			{ID: "Empty", KR: "빈", ZH: ""}, {ID: "Unknown", KR: "모름", ZH: "Unknown"},
		},
		Resonance: "共鳴", PerfectResonance: "完全共鳴",
	})
}

// names returns the Name of each link.
func names(links []Link) []string
```

測試與斷言：

```go
func TestLinkedAfterChar(t *testing.T) {
	g := testGame()
	for _, c := range []struct{ text string; want []string }{
		{"施加1級流血 強度", []string{"流血"}}, {"施加流血", []string{"流血"}}, {"流血3", []string{"流血"}},
		{"流血　強度", []string{"流血"}}, {"流血３", []string{"流血"}}, {"流血\n", []string{"流血"}},
		{"「流血」", []string{"流血"}}, {"流血・震顫 ", []string{"流血", "震顫"}}, {"流血,", []string{"流血"}},
		{"流血，", nil}, {"流血。", nil}, {"流血強度", nil}, {"流血)", nil}, {"流血</color>", nil},
		{"​流血 ", nil}, {"流血​ ", nil},
		{"攻擊等級+1", []string{"攻擊等級+"}}, {"使用「加斥我援」對", []string{"援"}}, {"增加1～2回", []string{"～"}},
		{"連 ", nil}, // excludeAutoLink
		{"震顫引爆 ", []string{"震顫引爆"}}, // the longest name takes the place
		{"震顫引爆，", nil},                 // and keeps it even when it is not linked
		{"憤怒 共鳴時", nil}, {"憤怒 完全 共鳴時", nil}, {"憤怒 攻擊", []string{"憤怒"}},
		// the game only looks for the word when the text goes on after it
		{"憤怒 共鳴", []string{"憤怒"}},
	} {
		if got := names(g.Linked(c.text, false)); !slices.Equal(got, c.want) {
			t.Errorf("Linked(%q) = %v, want %v", c.text, got, c.want)
		}
	}
}

func TestLinkedTagPass(t *testing.T) {
	g := testGame()
	// [TabExplain] is no buff id: with the tag pass it becomes a tag, without it stays "[": neither links.
	for _, tagPass := range []bool{true, false} {
		if got := g.Linked("流血[TabExplain] 強度", tagPass); got != nil {
			t.Errorf("tagPass %v: %v, want none", tagPass, got)
		}
	}
	// A name with brackets is cut by the tag pass, and matched as written without it.
	if got := g.Linked("本國劍術[肉] ", true); got != nil {
		t.Errorf("with the tag pass: %v", got)
	}
	if got := names(g.Linked("本國劍術[肉] ", false)); !slices.Equal(got, []string{"本國劍術[肉]"}) {
		t.Errorf("without the tag pass: %v", got)
	}
	// A buff id in brackets is left alone by the tag pass: the name before it is followed by "[".
	if got := g.Linked("流血[Vibration]", true); got != nil {
		t.Errorf("before an [ID]: %v", got)
	}
}

func TestGameNames(t *testing.T) {
	g := testGame()
	if got := g.Linked("Unknown 與 的 ", false); got != nil { // "" and Unknown are no names
		t.Errorf("%v, want none", got)
	}
	l := g.Linked("苦難 ", false)
	if len(l) != 1 || !slices.Equal(l[0].IDs, []string{"HardA", "HardB"}) || l[0].Start != 0 {
		t.Errorf("%+v", l)
	}
	if got := g.Linked("好流血 ", false); len(got) != 1 || got[0].Start != len("好") {
		t.Errorf("%+v, want Start at the byte offset", got)
	}
}

func TestKeywordLinkCheck(t *testing.T) {
	g := testGame()
	check := func(file, pointer, kr, zh string) []string // the Message of each keyword-link issue of Check with Game g
	// hasCode(issues, code) reports whether an issue of the code is among them (a helper of this file)
	const desc = "/dataList/0/levelList/0/desc"
	for _, c := range []struct{ name, file, pointer, kr, zh string; issues int }{
		{"KR never names the state", "Skills.json", desc, "1~2 증가", "增加1～2回", 1},
		{"KR names it by id", "Skills.json", desc, "[Laceration] 부여", "施加流血 ", 0},
		{"KR names it by its KR name", "Skills.json", desc, "출혈을 부여", "施加流血 ", 0},
		{"one of the ids of the name", "Skills.json", desc, "[HardB] 부여", "施加苦難 ", 0},
		{"not a processed field", "Skills.json", "/dataList/0/levelList/0/name", "이름", "流血 ", 0},
		{"not a processed file", "Bufs.json", "/dataList/0/desc", "설명", "流血 ", 0},
		{"gift desc", "EGOgift_MirrorDungeon.json", "/dataList/0/desc", "설명", "流血 ", 1},
		{"gift simpleDesc", "EGOgift.json", "/dataList/0/simpleDesc/1/simpleDesc", "설명", "流血 ", 1},
		{"panic", "PanicInfo.json", "/dataList/0/panicDescription", "설명", "流血 ", 1},
		{"passive summary", "Passives.json", "/dataList/0/summary", "설명", "流血 ", 1},
		{"coin summary", "Skills.json", "/dataList/0/levelList/0/coinlist/0/coindescs/0/summary", "설명", "流血 ", 1},
	} {
		if got := check(c.file, c.pointer, c.kr, c.zh); len(got) != c.issues {
			t.Errorf("%s: %v, want %d issue", c.name, got, c.issues)
		}
	}
	// One issue per unit names every link; the message is no statement of fact.
	msg := check("Skills.json", desc, "설명", "流血 與增加1～2回")
	if len(msg) != 1 || !strings.Contains(msg[0], "流血") || !strings.Contains(msg[0], "ConfessionThirdPhase") || !strings.Contains(msg[0], "by the game's code") {
		t.Errorf("%v", msg)
	}
	// No Game, or no Pointer: nothing.
	if is := Check("설명", "流血 ", Context{File: "Skills.json", Pointer: desc}); hasCode(is, CodeKeywordLink) {
		t.Error("issue without a Game")
	}
	if is := Check("설명", "流血 ", Context{File: "Skills.json", Game: g}); hasCode(is, CodeKeywordLink) {
		t.Error("issue without a Pointer")
	}
}
```

- [ ] **Step 2: 跑測試確認失敗**

Run: `go test ./internal/validate/ -run 'TestLinked|TestGameNames|TestKeywordLinkCheck' -v`
Expected: 編譯失敗（`NewGame` 未定義）

- [ ] **Step 3: 實作 `keyword.go`**

`Game` 內部保存：名稱 → ids（依輸入順序）、參加比對的名稱（非空、不分大小寫不等於 `unknown`、沒有 `ExcludeAutoLink`）、全部的 id 集合（標籤處理用）、id → KR 名稱。

`Linked` 的演算法（spec §5.1 步驟 2–4；簽名與測試沒有決定的部分）：

1. `tagPass` 時，把每個 `\[.*?\]`（不跨行）中括號內不是 `Bufs` id 的整段換成等長度無關的單一 `<`；要保留原文位置的對應，讓 `Link.Start` 是原文的位元組位置。
2. 逐字元位置找所有以該位置開頭的名稱（以名稱的第一個字元建索引即可），全部收集。
3. 依名稱的字元數由長到短排序（同長度依位置），逐一採用；與已採用的範圍重疊就丟掉；前一個或後一個字元是 U+200B 就丟掉。**被採用但之後不連結的名稱仍然佔住範圍。**
4. 對採用的名稱看後面的文字 `rest`（以字元計）：
   - `Resonance` 非空、`len(rest) > len(Resonance)+1` 且 `rest` 的前 `len(Resonance)+1` 個字元含有它 → 不連結
   - `PerfectResonance` 非空、`len(rest) > len(PerfectResonance)+2` 且 `rest` 的前 `len(PerfectResonance)+2` 個字元去掉半形空格後含有它 → 不連結
   - `rest` 為空、或第一個字元 `unicode.IsSpace`、`unicode.IsDigit`、或在 `・「」『』.,/&;` 裡 → 連結；否則不連結
5. 依位置排序回傳。

`keyword-link` 檢查：範圍表（族群、欄位路徑、是否 `tagPass`）照 spec §5.1；對 `Linked(zh, tagPass)` 的每個連結，`IDs` 中任何一個的 `"["+id+"]"` 或非空的 KR 名稱出現在 `kr` 裡就略過。剩下的合成一個問題，訊息：

```
by the game's code these words are linked to states KR does not name: 「流血」 (Laceration), 「～」 (ConfessionThirdPhase)
```

同名多個 id 時括號內只寫第一個 id。

- [ ] **Step 4: 跑測試確認通過**

Run: `go test ./internal/validate/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/validate && git commit -m "validate：keyword-link，依遊戲的關鍵字比對找出 KR 沒提到的連結"
```

---

### Task 3: `level-diff`

**Files:**
- Create: `internal/validate/leveldiff.go`、`internal/validate/leveldiff_test.go`
- Modify: `internal/validate/gamechecks.go`（`checkGame` 呼叫）

**Interfaces:**
- Consumes: `GameInput.PrevLevel`、`LevelText`、`Context.File`、`Context.Pointer`、`Context.Game`、`CodeLevelDiff`
- Produces: `func (g *Game) prevLevel(file, pointer string) (LevelText, bool)`（以 `file + "|" + pointer` 查 `PrevLevel`；`validate` 不 import `units`，key 的格式與 `units.Key` 相同，在註解寫明）

- [ ] **Step 1: 寫失敗的測試**

```go
func TestLevelDiff(t *testing.T) {
	const key = "Skills.json|/dataList/0/levelList/1/desc"
	check := func(prev LevelText, kr, zh string) []string // messages of the level-diff issues, Game with PrevLevel{key: prev}
	same := LevelText{Level: 1, KR: "가\n나 3", ZH: "甲\n乙3 "}
	for _, c := range []struct{ name string; prev LevelText; kr, zh string; issues int }{
		{"the unchanged line differs", same, "가\n나 4", "甲 \n乙4 ", 1},
		{"only the changed line differs", same, "가\n나 4", "甲\n乙 4", 0},
		{"identical", same, "가\n나 3", "甲\n乙3 ", 0},
		{"the lower level lacks the line in zh", LevelText{Level: 1, KR: "가\n나", ZH: "甲"}, "가\n나", "甲\n乙", 1},
		{"KR has more lines now", same, "가\n나 3\n다", "甲\n乙3 \n丙", 0},
		{"a trailing space counts", same, "가\n나 3", "甲\n乙3", 1},
	} {
		if got := check(c.prev, c.kr, c.zh); len(got) != c.issues {
			t.Errorf("%s: %v, want %d issue", c.name, got, c.issues)
		}
	}
	msg := check(same, "가\n나 4", "甲 \n乙4 ")
	if len(msg) != 1 || !strings.Contains(msg[0], "line 1") || !strings.Contains(msg[0], "level 1") {
		t.Errorf("%v", msg)
	}
}

// A unit with no entry of PrevLevel, or checked without a Game, has no issue.
func TestLevelDiffWithoutLowerLevel(t *testing.T)
```

- [ ] **Step 2: 跑測試確認失敗**

Run: `go test ./internal/validate/ -run TestLevelDiff -v`
Expected: FAIL

- [ ] **Step 3: 實作 `leveldiff.go`**

兩邊各以 `\n` 切行。對行號 i：KR 這一級與上一級都有第 i 行且相等，而譯文兩級的第 i 行不相等（任一邊沒有第 i 行算不相等）→ 記下。有記下的行就報一個問題，訊息：

```
line %d is the same as level %d in KR but differs in the translation (%q / %q); by the game's code the uptie comparison marks it as changed
```

`%d` 是第一個不同行的行號（從 1 起算），兩個字串是上一級與這一級的該行，各截到 40 個字元。不同的行不只一行時在結尾加 `, and %d more`。

- [ ] **Step 4: 跑測試確認通過**

Run: `go test ./internal/validate/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/validate && git commit -m "validate：level-diff，KR 兩個同步等級相同而譯文不同的行"
```

---

### Task 4: `internal/gametext`：從 KR 檔與紀錄建 `validate.Game`

**Files:**
- Create: `internal/gametext/gametext.go`、`internal/gametext/gametext_test.go`

**Interfaces:**
- Consumes: `validate.GameInput`、`validate.GameBuf`、`validate.LevelText`、`validate.NewGame`；`pack.Pack`（`Files()`、`Doc(file)`）、`jsonx.Node`、`units.Key`、`units.KRSha`、`store.Record`（`ZH`、`KRSha`、`Origin`）、`store.OriginCopy`、`config.MatchAny`
- Produces:

```go
// Text returns the translation the pack ships for the unit of file and
// pointer whose KR text is kr; ok is false when the unit has no usable record.
type Text func(file, pointer, kr string) (zh string, ok bool)

// UsedText is the Text of a build: the text of the record of the unit when
// the record was made from kr (or the file is frozen) and is not a copy.
// The field rules are not asked: the units Load reads are all translated.
func UsedText(records map[string]store.Record, frozen []string) Text

// Load builds the game information of the pack kr with the translations text gives.
func Load(kr *pack.Pack, text Text) (*validate.Game, error)

// Input is Load before NewGame, for tests.
func Input(kr *pack.Pack, text Text) (validate.GameInput, error)
```

- [ ] **Step 1: 寫失敗的測試**（用 `t.TempDir()` 寫小的 KR 檔再 `pack.Open(dir, "KR_")`；紀錄用 `store.Record{ZH:…, KRSha: units.KRSha(kr), Origin: store.OriginClaude}`）

```go
func TestInputBufs(t *testing.T)
// KR_Bufs.json: [{id:"A",name:"가"},{id:"B",name:"나",excludeAutoLink:true},{id:"C",name:"다"},{id:3,name:"라"}]
// KR_Bufs-x.json: [{id:"D",name:"마"}]; KR_BattleKeywords.json: [{id:"Z",name:"자"}] (not read)
// records: A -> "甲"; C -> a copy; D -> made from another KR text
// want Bufs == [{A 가 甲 false} {B 나 나 true} {C 다 다 false} {3 라 라 false} {D 마 마 false}]  (file-name order: Bufs.json before Bufs-x.json, as names.KeywordTable sorts)

func TestInputResonance(t *testing.T)
// KR_BattleUIText.json: [{id:"resonance",content:"공명"},{id:"perfect_resonance",content:"완전공명"}]
// record of resonance -> "共鳴", none of perfect_resonance: Resonance == "共鳴", PerfectResonance == "완전공명"
// no BattleUIText.json at all: both ""

func TestLevelPairs(t *testing.T)
// KR_Skills.json, one skill with levelList levels [1,3,4]: desc "가","가2","가3"; coinlist[0].coindescs[0].desc at each level; level 4 has a second coin level 3 lacks
// all with records: PrevLevel has exactly the keys of the level 3 and level 4 desc and coin descs that have a counterpart:
//   /dataList/0/levelList/1/desc -> {Level 1, KR "가", ZH …}; /dataList/0/levelList/2/desc -> {Level 3, …};
//   /dataList/0/levelList/2/coinlist/1/coindescs/0/desc has no entry (level 3 has no coin 1)
// the level 1 desc without a usable record: the level 3 desc has no entry
// a skill whose levels repeat (1,1) or whose level is not a number: no entry, no error
// a file under a folder (RPGSystem/Skills-x.json) and Skills-like families (SkillTag.json) are not read

func TestUsedText(t *testing.T)
// a record made from the KR text: its ZH, true; made from another text: false; the same in a frozen file: true; a copy: false; no record: false
```

- [ ] **Step 2: 跑測試確認失敗**

Run: `go test ./internal/gametext/ -v`
Expected: 編譯失敗

- [ ] **Step 3: 實作**

- 讀的檔：根目錄（檔名沒有 `/`）中族群（與 `validate` 相同的規則：去掉 `.json` 後到第一個 `-` 或 `_` 之前）是 `Bufs`、`Skills` 的檔，以及 `BattleUIText.json`。族群規則在這個套件自己寫一份小函式並在註解指向 `validate.family`（`validate` 的是未匯出的）。
- `Bufs` 的檔案順序：去掉 `.json` 後以字串排序（與 `names.KeywordTable` 相同）。每一筆取 `id`（字串用值、數字用原文字面）、`name`（字串才收；單位的指標是 `/dataList/<i>/name`）、`excludeAutoLink`（`Bool` 且字面是 `true`）。沒有 `id` 或 `name` 不是字串的略過。
- 「上一級」：同一筆技能的 `levelList` 中 `level` 是整數的各筆，依 `level` 由小到大；`level` 有重複或有不是整數的，整筆技能不產生配對。相鄰的兩筆是一組；每組產生 `desc` 的配對，以及兩邊都有的 `coinlist[c].coindescs[d].desc` 的配對（依位置）。兩邊的 `desc` 都要是字串、都要有可用譯文，才寫進 `PrevLevel`。

- [ ] **Step 4: 跑測試確認通過**

Run: `go test ./internal/gametext/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/gametext && git commit -m "gametext：從 KR 檔與翻譯紀錄建出檢查用的遊戲文字資訊"
```

---

### Task 5: 接進建置、網站匯出與其他呼叫處

**Files:**
- Modify: `internal/build/build.go`（`builder` 加欄位、`buildLocked` 建 `Game`、`useRecord` 的 `Check`）
- Modify: `internal/site/review.go`（`BuildReview` 建 `Game`、`Check`）
- Modify: `internal/batch/batch.go`、`internal/batch/judge.go` 或 `submit.go`、`cmd/lct/adjust.go`、`cmd/lct/validate.go`、`internal/bootstrap/bootstrap.go`（帶上 `Pointer`）
- Test: `internal/build/build_test.go`、`internal/site/review_test.go`

**Interfaces:**
- Consumes: `gametext.Load`、`gametext.UsedText`、`validate.Context.Pointer`、`validate.Context.Game`
- Produces: 無新的匯出介面

- [ ] **Step 1: 寫失敗的測試**

`internal/site/review_test.go` 加 `TestReviewGameWarnings`：以 `reviewInputs` 的方式另建一組輸入——

```
KR_Bufs.json    [{"id":"Laceration","name":"출혈"}]
KR_Skills.json  [{"id":1,"levelList":[{"level":1,"desc":"가\n나 3"},{"level":2,"desc":"가\n나 4"}]},
                 {"id":2,"levelList":[{"level":1,"desc":"설명"}]}]
records         Bufs name -> "流血"; skill 1 level 1 desc -> "甲\n乙3"; level 2 desc -> "甲 \n乙4"; skill 2 desc -> "流血 "
```

斷言：`r.Warnings` 中 `keyword-link` 恰好一項，`Keys` 是技能 2 的 `desc`；`level-diff` 恰好一項，`Keys` 是技能 1 等級 2 的 `desc`；兩項的 `Message` 非空、`ID` 以 `warning/keyword-link/`、`warning/level-diff/` 開頭。

`internal/build/build_test.go` 加 `TestBuildGameWarnings`：同一份資料跑建置（照該檔既有的 fixture 輔助函式），斷言 `rep.Warnings["keyword-link"] == 1`、`rep.Warnings["level-diff"] == 1`，且 `rep.WarningSamples` 兩個代碼的 key 與上面相同。

- [ ] **Step 2: 跑測試確認失敗**

Run: `go test ./internal/site/ ./internal/build/ -run 'GameWarnings' -v`
Expected: FAIL（沒有這兩個代碼的警告）

- [ ] **Step 3: 實作**

- 建置：`buildLocked` 在迴圈前 `game, err := gametext.Load(src.KR, gametext.UsedText(records, opt.Frozen))`，放進 `builder`；`useRecord` 的 `validate.Context` 加 `Pointer: u.Pointer, Game: b.game`。
- 網站：`BuildReview` 在檔案迴圈前 `gametext.Load(in.KR, gametext.UsedText(in.Records, rv.Frozen))`；`Check` 的 `Context` 加 `Pointer: ku.Pointer, Game: game`。
- 其他呼叫處：`validate.Check` 的 `Context` 在知道單位指標的地方加 `Pointer`（`grep -n 'validate.Context{' -r internal cmd` 逐一看；拿不到指標的維持原樣）。`lct validate` 增加可省略的 `--pointer` 旗標，照該指令既有旗標的寫法。這些地方不提供 `Game`。

- [ ] **Step 4: 跑全部測試**

Run: `go vet ./... && go test ./...`
Expected: PASS。若有既有測試因為 fixture 文字剛好觸發新警告而失敗，逐一判斷：是新檢查正確報出的就更新該測試的預期並在 commit 訊息列出；不是就修檢查。

- [ ] **Step 5: 用真實資料核對**（需要 `.work/raw`、`.work/tw`；沒有就跳過並在回報中說明）

```bash
go build -o bin/lct ./cmd/lct && bin/lct site --out out/site-new --json > /dev/null
python3 -I - <<'EOF'
import json, collections
def count(p):
    c = collections.Counter()
    for w in json.load(open(p))["items"]:
        c[w["code"]] += len(w.get("keys", []))
    return c
old, new = count("out/site-base/review/warnings.json"), count("out/site-new/review/warnings.json")
for k in sorted(set(old) | set(new)):
    print("%-18s %5d -> %5d" % (k, old[k], new[k]))
EOF
```

Expected：既有代碼的數字完全不變；新代碼的數量級符合 spec §5 的「現況」（`keyword-link` 約 130、`level-diff` 約 61、`newline-space` 9、`page-count` 5、`bracket-text` 4、`bullet-count` 2，其餘 0）。差距明顯時先查原因再往下。`out/site-base` 是改動前以 `bin/lct site --out out/site-base` 產生的基準；沒有的話先在 `main` 上產生。

再確認既有的確認清單沒有失效：`bin/lct accept --tidy --dry-run`（或該指令實際提供的唯讀方式）列出的失效條目，與改動前相同。

- [ ] **Step 6: Commit**

```bash
git add internal cmd && git commit -m "建置與網站匯出帶上遊戲文字資訊：新檢查出現在建置報告與審核頁"
```

---

### Task 6: 審核頁的名稱與說明

**Files:**
- Modify: `site/src/review/names.ts`
- Test: `site/src/review/names.test.ts`

- [ ] **Step 1: 寫失敗的測試**（加在 `names.test.ts` 既有的 `describe` 旁）

```ts
it('names every check of the game text', () => {
  const codes = ['keyword-link', 'bracket-text', 'page-count', 'level-diff', 'story-comment', 'stray-brace', 'regex-char',
    'literal-newline', 'reserved-value', 'newline-space', 'qmark-count', 'bullet-count', 'ruby-form']
  for (const code of codes) {
    const info = warningInfo(code, 'message of the check')
    expect(info.name).not.toBe(code)
    expect(info.desc).not.toBe('message of the check')
  }
})
```

- [ ] **Step 2: 跑測試確認失敗**

Run: `cd site && npm ci && npx vitest run src/review/names.test.ts`
Expected: FAIL

- [ ] **Step 3: 在 `WARNING_NAMES` 加 13 筆**（文字固定如下）

| 代碼 | name | desc |
|---|---|---|
| `keyword-link` | 關鍵字誤連結 | 依遊戲程式推算，TW 的這些字會被連成 KR 沒提到的狀態 |
| `bracket-text` | 方括號文字 | 技能、被動文字裡不是 ID 的 [文字]，依遊戲程式推算會顯示成 Unknown |
| `page-count` | 頁數不同 | 選擇事件以空行分頁；頁數比 KR 少時遊戲會補「...」頁 |
| `level-diff` | 同步等級比較 | KR 兩個同步等級相同的行，TW 不同：比較畫面會標成有變 |
| `story-comment` | // 截斷 | 劇情文字裡 // 之後的字不會顯示 |
| `stray-brace` | 多餘的大括號 | KR 沒有的 { 或 }，代入數值時可能出錯 |
| `regex-char` | 正規表示式字元 | 這個欄位的文字會被遊戲當成正規表示式，半形括號等符號不成對會出錯 |
| `literal-newline` | 字面的 \n | 反斜線加 n 不是換行，會原樣顯示（選擇事件裡會被當成分頁） |
| `reserved-value` | 保留字 | 值是 Unknown 或 Error 時，遊戲當成沒有資料 |
| `newline-space` | 換行旁的空格 | 稱號的換行在單行顯示時變成空格，旁邊再有空格就是兩個 |
| `qmark-count` | 問號個數 | 罪孽型敵人名字的半形 ? 個數決定「N型」，要和 KR 相同 |
| `bullet-count` | 條列符號 | 但丁筆記裡「換行後接 -」的數量和 KR 不同 |
| `ruby-form` | ruby 寫法 | 對話框只轉換 <ruby=注音>本文</ruby> 這一種寫法 |

- [ ] **Step 4: 跑網站的檢查**

Run: `cd site && npm test && npm run build && npm run typecheck`
Expected: 全部通過

- [ ] **Step 5: 更新文件並 Commit**

- spec 開頭的狀態改為「使用者已核可（2026-11-15）」。
- `docs/research/2026-11-15-game-text-logic/README.md` 的「給網站檢查的建議」一節開頭加一行，指向 spec 並說明哪些已做成檢查。

```bash
git add site docs && git commit -m "審核頁：依遊戲行為檢查的名稱與說明"
```
