# 模擬 ruby 的自訂標籤 `<rubycustomize>` 實作計畫

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal：** 譯文可以在白名單位置寫 `<rubycustomize=注音>本文</rubycustomize>`，`lct build` 把它換成遊戲能顯示的 TextMeshPro 標籤；另外讓三語相同的 `ScenarioModelCodes` 的 `name` 照樣送翻。

**Architecture：** 新的純函式套件 `internal/rubyx` 負責語法檢查、去標籤、轉換與還原。白名單由 `internal/rules` 載入；`internal/validate` 依呼叫端給的 `Context.RubyCustomize` 決定接不接受；`internal/build` 用已載入的字型算寬度並轉換。網站與更新頁用「還原」比對來源，網站前端把轉換後的結構顯示成 ruby。

**Tech Stack：** Go（`golang.org/x/image/font/sfnt`、`gopkg.in/yaml.v3`）、TypeScript／Preact（`site/`、Vitest）。

**Spec：** `docs/superpowers/specs/2026-11-16-ruby-customize-design.md`

## Global Constraints

- 標籤語法：`<rubycustomize=注音>本文</rubycustomize>`。名稱只接受小寫；`=` 必須有；注音可以加雙引號；注音、本文都是非空的純文字（不含 `<`、`>`、換行），不巢狀。
- 轉換輸出逐字照這個格式：`<nobr>B<space=S1em><voffset=V><size=Z>R</size></voffset><space=S2em></nobr>`，其中 `S1 = -(wb+wr)/2`、`S2 = (wb-wr)/2`。
  - `wb`：本文每個字元的 advance 加總，單位 em。
  - `wr`：注音的 advance 加總乘上字級比例（`50%` → 0.5）。
  - 數值四捨五入到小數三位，去掉尾端的 0（`-2.25em`、`0.5em`、`0em`）。
  - 字型缺字的字元算 1em。不做 kerning。
- 轉換參數 `size: 50%`、`voffset: 1em`，寫在 `rules/ruby-customize.yaml`。
- 白名單逐字為這 5 項：
  - `ScenarioModelCodes*` 的 `name`
  - `StoryData/**` 的 `teller`
  - `PersonalityVoiceDlg/**` 的 `dlg`
  - `BattleAnnouncerDlg/**` 的 `dlg`
  - `BattleSpeechBubbleDlg*` 的 `dlg`
- 照抄例外：`rules/fields.manual.yaml` 的 `translate_identical`，第一項是 `file_glob: 'ScenarioModelCodes*'`、`field: name`。
- 內容檢查（詞彙表、殘留外文、字數等）只看本文；注音只檢查語法。
- 規則寫在 `guides/`、`glossary/`、`rules/`，不寫進程式或提示詞；`prompts/` 與 skill 不寫模型名稱。
- 提交前：`go vet ./... && go test ./...`。動到 `site/` 時還要跑 `cd site && npm ci && npm test && npm run typecheck && npm run build`。
- 提交訊息結尾：
  ```
  Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
  Claude-Session: https://claude.ai/code/session_015XrjRvHe6FfxjemSS1js35
  ```
- 不在範圍內：設計第 12 節的舊譯文補正（76 筆）。那是機制合併後的另一次變更，9 句台詞要使用者確認譯法。

## Review Focus

1. 白名單位置的譯文被「雙胞胎」複製到不在白名單的同 KR 單位（`internal/batch/twins.go`）時，必須在那個位置檢查失敗，不能帶著標籤進語言包。測試在 Task 4。
2. 不帶 pointer 的檢查路徑：`lct plan` 套用改寫與調整、`lct impact`。這些路徑若把白名單位置的新標籤當成錯誤，改寫會被默默略過、impact 會誤報。測試在 Task 4。
3. 白名單位置的紀錄標籤寫錯時，建置要照現有流程改用代替文字並重新排入待譯，不能讓整個建置失敗。測試在 Task 6。
4. 網站的還原只認第 6 節的完整結構：官方文字裡單獨的 `<nobr>`、`<space>` 不能被改寫。測試在 Task 7（Go 與 TS 兩邊）。
5. 三語相同但不在 `translate_identical` 的欄位（例如 `ScenarioModelCodes` 的 `nickName` 的「LCD」）仍然照抄；而在例外裡、紀錄仍是照抄的單位（UNGEZIEFER KAISER）不能因此從網站搜尋消失。測試在 Task 5、Task 7。

---

### Task 1：`internal/rubyx`：語法、去標籤、轉換、還原

**Files：**
- Create: `internal/rubyx/rubyx.go`
- Test: `internal/rubyx/rubyx_test.go`

**Interfaces：**
- Produces：
  - `const Tag = "rubycustomize"`
  - `func Has(s string) bool`：含有 `<rubycustomize` 或 `</rubycustomize`，大小寫不拘。
  - `func Problems(s string) []string`：語法問題，英文短句，沒有問題時為 nil。
  - `func Strip(s string) string`：每組換成本文。只在 `Problems` 為 nil 時呼叫。
  - `type Style struct { Size string; Scale float64; VOffset string }`，例如 `{"50%", 0.5, "1em"}`。
  - `func ParseStyle(size, voffset string) (Style, error)`：`size` 必須是 `N%`（1～100），`voffset` 必須是數字加 `em`。
  - `func Convert(s string, st Style, advance func(rune) float64) string`：`advance` 回傳 em，缺字由呼叫端回傳 1。
  - `func Restore(s string) string`：把轉換後的完整結構還原成 `<rubycustomize=R>B</rubycustomize>`，其他文字不動。

- [ ] **Step 1：寫會失敗的測試**

```go
func TestConvertMatchesInGameTests(t *testing.T) {
	st := Style{Size: "50%", Scale: 0.5, VOffset: "1em"}
	// Sarasa Gothic TC Bold 的 advance（/1000）：漢字、假名 1em；英文字母照字型。
	latin := map[rune]float64{' ': .237, 'A': .747, 'E': .607, 'F': .587, 'G': .760, 'I': .281, 'K': .719, 'N': .762, 'R': .657, 'S': .655, 'U': .732, 'Z': .664}
	adv := func(r rune) float64 { if w, ok := latin[r]; ok { return w }; return 1 }
	cases := map[string]string{
		"<rubycustomize=ヨシヒデ>良秀</rubycustomize>":                 "<nobr>良秀<space=-2em><voffset=1em><size=50%>ヨシヒデ</size></voffset><space=0em></nobr>",
		"<rubycustomize=UNGEZIEFER KAISER>害蟲皇帝</rubycustomize>":    "<nobr>害蟲皇帝<space=-4.542em><voffset=1em><size=50%>UNGEZIEFER KAISER</size></voffset><space=-0.542em></nobr>",
		"就算<rubycustomize=\"ヨシヒデ\">良秀</rubycustomize>姊姊":        "就算<nobr>良秀<space=-2em><voffset=1em><size=50%>ヨシヒデ</size></voffset><space=0em></nobr>姊姊",
		"<rubycustomize=李箱>異常</rubycustomize>與<rubycustomize=ヨシヒデ>良秀</rubycustomize>": "<nobr>異常<space=-1.5em><voffset=1em><size=50%>李箱</size></voffset><space=0.5em></nobr>與<nobr>良秀<space=-2em><voffset=1em><size=50%>ヨシヒデ</size></voffset><space=0em></nobr>",
	}
	for in, want := range cases {
		if got := Convert(in, st, adv); got != want { t.Errorf("Convert(%q) = %q, want %q", in, got, want) }
	}
}

func TestProblems(t *testing.T) {
	ok := []string{"<rubycustomize=ヨシヒデ>良秀</rubycustomize>", "<rubycustomize=\"A B\">甲</rubycustomize>乙", "沒有標籤"}
	bad := []string{
		"<rubycustomize>良秀</rubycustomize>",                 // 沒有 =
		"<rubycustomize=>良秀</rubycustomize>",                // 空的注音
		"<rubycustomize=ヨシヒデ></rubycustomize>",             // 空的本文
		"<rubycustomize=ヨシヒデ>良<b>秀</b></rubycustomize>",   // 本文有標籤
		"<rubycustomize=ヨシ\nヒデ>良秀</rubycustomize>",         // 換行
		"<rubycustomize=ヨシヒデ>良秀",                          // 沒有閉合
		"良秀</rubycustomize>",                                // 多出的閉合
		"<RubyCustomize=ヨシヒデ>良秀</RubyCustomize>",          // 大寫
		"<rubycustomize=甲><rubycustomize=乙>丙</rubycustomize></rubycustomize>", // 巢狀
	}
	for _, s := range ok { if p := Problems(s); p != nil { t.Errorf("Problems(%q) = %v, want none", s, p) } }
	for _, s := range bad { if Problems(s) == nil { t.Errorf("Problems(%q) = none, want a problem", s) } }
}

func TestStripAndRestore(t *testing.T) {
	src := "就算<rubycustomize=ヨシヒデ>良秀</rubycustomize>姊姊"
	if got := Strip(src); got != "就算良秀姊姊" { t.Errorf("Strip = %q", got) }
	conv := Convert(src, Style{"50%", 0.5, "1em"}, func(rune) float64 { return 1 })
	if got := Restore(conv); got != src { t.Errorf("Restore(Convert(x)) = %q, want %q", got, src) }
	// 不是完整結構的 nobr、space 不動
	for _, s := range []string{"<nobr>甲乙</nobr>", "甲<space=1em>乙", "<nobr>甲<space=-1em><voffset=1em>乙</voffset></nobr>"} {
		if got := Restore(s); got != s { t.Errorf("Restore(%q) = %q, want unchanged", s, got) }
	}
}

func TestParseStyle(t *testing.T) {
	st, err := ParseStyle("50%", "1em")
	if err != nil || st != (Style{"50%", 0.5, "1em"}) { t.Fatalf("ParseStyle = %v, %v", st, err) }
	for _, bad := range [][2]string{{"50", "1em"}, {"0%", "1em"}, {"150%", "1em"}, {"50%", "1px"}, {"50%", "em"}} {
		if _, err := ParseStyle(bad[0], bad[1]); err == nil { t.Errorf("ParseStyle(%q, %q) accepted", bad[0], bad[1]) }
	}
}
```

- [ ] **Step 2：確認失敗**　`go test ./internal/rubyx/`　預期：編譯失敗（套件不存在）
- [ ] **Step 3：實作 `internal/rubyx/rubyx.go`**
  - 用一個正規表示式找每一組。比對名稱要區分大小寫，讓大寫寫法落到 `Problems`。
  - `Restore` 的正規表示式只認完整結構：`<nobr>([^<>]+)<space=-?[0-9.]+em><voffset=[^<>]+><size=[^<>]+>([^<>]+)</size></voffset><space=-?[0-9.]+em></nobr>`。
  - 數值格式：`strconv.FormatFloat(math.Round(x*1000)/1000, 'f', -1, 64)`，並把 `-0` 寫成 `0`。
- [ ] **Step 4：確認通過**　`go test ./internal/rubyx/`　預期：PASS
- [ ] **Step 5：提交**　`git add internal/rubyx && git commit`，訊息「rubyx：<rubycustomize> 的語法檢查、轉換與還原」

### Task 2：白名單 `rules/ruby-customize.yaml`

**Files：**
- Create: `rules/ruby-customize.yaml`、`internal/rules/rubycustomize.go`、`internal/rules/rubycustomize_test.go`
- Modify: `internal/rules/rules.go`（`Rules` 加欄位、`Load` 讀檔）、`internal/rules/lint.go`（`Lint` 呼叫）、`cmd/lct/rules.go`（加入警告）

**Interfaces：**
- Consumes：`rubyx.ParseStyle`
- Produces：
  - `const RubyCustomizeFile = "rules/ruby-customize.yaml"`
  - `type RubyCustomize struct { Size string \`yaml:"size"\`; VOffset string \`yaml:"voffset"\`; Fields []RubyField \`yaml:"fields"\` }`
  - `type RubyField struct { Files []string \`yaml:"files"\`; Field string \`yaml:"field"\` }`
  - `Rules.RubyCustomize RubyCustomize`
  - `func (r *Rules) RubyCustomizeAt(file, pointer string) bool`：`pointer` 為 "" 時回傳 false。欄位用 `units.FieldName(pointer)`，檔案用 `config.MatchAny(f.Files, file)`。
  - `func (r *Rules) RubyStyle() (rubyx.Style, error)`
  - `func LintRubyCustomizeWarnings(root string, krFiles []string) []LintError`

- [ ] **Step 1：寫會失敗的測試**

```go
func TestRubyCustomizeAt(t *testing.T) {
	r, err := Load("../..")
	if err != nil { t.Fatal(err) }
	yes := [][2]string{
		{"ScenarioModelCodes-AutoCreated.json", "/dataList/630/name"},
		{"StoryData/P10916.json", "/dataList/3/teller"},
		{"PersonalityVoiceDlg/Voice_Heathcliff_KumoThumb_10716.json", "/dataList/6/dlg"},
		{"BattleAnnouncerDlg/Announcer_RienSora_54.json", "/dataList/1/dlg"},
		{"BattleSpeechBubbleDlg-a1c9p3.json", "/dataList/275/dlg"},
	}
	no := [][2]string{
		{"ScenarioModelCodes-AutoCreated.json", "/dataList/630/nickName"},
		{"StoryData/S945B.json", "/dataList/47/title"},
		{"StoryData/S945B.json", "/dataList/47/place"},
		{"StoryData/S945B.json", "/dataList/47/content"},
		{"PersonalityVoiceDlg/Voice_Heathcliff_KumoThumb_10716.json", "/dataList/6/desc"},
		{"RPGSystem/rpg-loc-dialogue-common-a1c10p2.json", "/dataList/9/texts/6/text"},
		{"Skills_personality-x1p1c1.json", "/dataList/3/levelList/0/name"},
		{"ScenarioModelCodes-AutoCreated.json", ""},
	}
	for _, c := range yes { if !r.RubyCustomizeAt(c[0], c[1]) { t.Errorf("%v not allowed", c) } }
	for _, c := range no { if r.RubyCustomizeAt(c[0], c[1]) { t.Errorf("%v allowed", c) } }
	st, err := r.RubyStyle()
	if err != nil || st.Scale != 0.5 || st.VOffset != "1em" { t.Errorf("RubyStyle = %v, %v", st, err) }
}
```

  另外寫 lint 的測試（照 `npcs` 的測試寫法，用暫存目錄）：
  - 未知的鍵、缺 `files` 或 `field`、`size: 50`、`files` 樣式不合法，各報一個 `LintError`，`File` 是 `rules/ruby-customize.yaml`。
  - `LintRubyCustomizeWarnings(root, []string{"Skills.json"})`：每一項的 `files` 都對不到，回傳 5 個警告。
  - 檔案不存在時，`Load` 成功，`RubyCustomizeAt` 一律 false。

- [ ] **Step 2：確認失敗**　`go test ./internal/rules/ -run 'RubyCustomize'`　預期：編譯失敗
- [ ] **Step 3：實作**
  - 建立 `rules/ruby-customize.yaml`：內容照設計第 5 節，包含第 5 項 `BattleSpeechBubbleDlg*`；每項加上測試畫面與「2026-11-16 實測」的註解。
  - `Load` 讀這個可選檔案，照 `NPCsFile` 的寫法。
  - `Lint` 加 `l.rubyCustomize()`。
  - `cmd/lct/rules.go` 把 `LintRubyCustomizeWarnings` 加進 `report.Warnings`。
- [ ] **Step 4：確認通過**　`go test ./internal/rules/ ./cmd/lct/`；`go build -o bin/lct ./cmd/lct && bin/lct rules lint --json`　預期：沒有錯誤
- [ ] **Step 5：提交**　訊息「規則：rubycustomize 白名單 rules/ruby-customize.yaml」

### Task 3：檢查 `internal/validate`

**Files：**
- Modify: `internal/validate/validate.go`
- Test: `internal/validate/rubycustomize_test.go`

**Interfaces：**
- Consumes：`rubyx.Has`、`rubyx.Problems`、`rubyx.Strip`
- Produces：
  - `Context.RubyCustomize bool`：這個單位的位置在白名單上，由呼叫端填寫。
  - `const CodeRubyCustomize = "rubycustomize"`，類別是 `ClassStructural`。

- [ ] **Step 1：寫會失敗的測試**

```go
func TestCheckRubyCustomize(t *testing.T) {
	kr, ctx := "요시히데", Context{File: "StoryData/P10916.json", Pointer: "/dataList/3/teller", RubyCustomize: true}
	if is := Check(kr, "<rubycustomize=ヨシヒデ>良秀</rubycustomize>", ctx); len(is) != 0 { t.Errorf("allowed: %v", is) }
	// 本文仍受詞彙表約束：注音裡有「良秀」不算
	ctx.Terms = []Term{{KR: "요시히데", ZH: "良秀"}}
	if is := Check(kr, "<rubycustomize=良秀>本名</rubycustomize>", ctx); !hasCode(is, CodeTermMissing) { t.Errorf("term in the reading only must fail: %v", is) }
	ctx.Terms = nil
	if is := Check(kr, "<rubycustomize=>良秀</rubycustomize>", ctx); !hasHard(is, CodeRubyCustomize) { t.Errorf("syntax: %v", is) }
	ctx.RubyCustomize = false
	if is := Check(kr, "<rubycustomize=ヨシヒデ>良秀</rubycustomize>", ctx); !hasHard(is, CodeRubyCustomize) { t.Errorf("place: %v", is) }
	// 注音是假名，不觸發殘留假名的檢查
	ctx.RubyCustomize = true
	if is := Check("그레고르", "<rubycustomize=UNGEZIEFER KAISER>害蟲皇帝</rubycustomize>", ctx); len(is) != 0 { t.Errorf("reading checked as text: %v", is) }
}

func TestRPGDialogueTakesNoRubyKRLacks(t *testing.T) {
	is := Check("나무", "<ruby=Sephirot>生命之樹</ruby>", Context{File: "RPGSystem/rpg-loc-dialogue-common-a1c10p2.json"})
	if !hasCode(is, CodeTagMismatch) { t.Errorf("RPG dialogue accepted a ruby KR lacks: %v", is) }
	if is := Check("나무", "<ruby=Sephirot>生命之樹</ruby>", Context{File: "StoryData/S945B.json"}); hasCode(is, CodeTagMismatch) { t.Errorf("story content: %v", is) }
}
```

`hasCode`、`hasHard` 若不存在，就寫成這個測試檔裡的小函式。`CodeTermMissing` 用這個套件裡詞彙缺漏的實際代碼名稱。

- [ ] **Step 2：確認失敗**　`go test ./internal/validate/ -run 'RubyCustomize|RPGDialogue'`
- [ ] **Step 3：實作**
  - 在 `Check` 一開始（判斷空字串之後）加上：
    - `rubyx.Has(zh)` 而且 `!ctx.RubyCustomize`：加一個硬性的 `CodeRubyCustomize`，訊息寫明「這個位置不在 rules/ruby-customize.yaml」，其餘檢查照原文進行。
    - 允許，但 `Problems` 不是空的：加硬性問題（列出問題），其餘檢查照原文進行。
    - 允許，而且語法正確：`zh = rubyx.Strip(zh)`，之後的所有檢查都只看本文。
  - `isStory` 的其他用途保留。只有 `tagDiff` 的 `addRuby` 改用新的 `isStoryContent(file)`：`StoryData/` 開頭，不含 RPG 對話。用 `grep -n isStory internal/validate` 確認每個用途。
- [ ] **Step 4：確認通過**　`go test ./internal/validate/`
- [ ] **Step 5：提交**　訊息「檢查：rubycustomize 只在白名單位置；RPG 對話不再接受 KR 沒有的 ruby」

### Task 4：所有檢查的呼叫端填寫 `RubyCustomize`

**Files：**
- Modify:
  - `internal/build/build.go:628`
  - `internal/site/review.go:433`
  - `internal/batch/batch.go`（`checkNewTerms`、新增 `CheckNewAt`、`CheckNewTermsAt`）
  - `internal/batch/twins.go:174`
  - `internal/plan/plan.go:1245,1288,1339`
  - `internal/impact/impact.go:109,113`
  - `internal/bootstrap/bootstrap.go:357`
  - `cmd/lct/adjust.go:150`
  - `cmd/lct/validate.go:75`
- Test：在 `internal/batch`、`internal/plan`、`internal/impact` 既有的測試檔各加一個測試

**Interfaces：**
- Consumes：`rules.Rules.RubyCustomizeAt`、`validate.Context.RubyCustomize`
- Produces：
  - `func CheckNewAt(r *rules.Rules, file, pointer, kr, zh string) []validate.Issue`：同 `CheckNew`；`pointer` 只用來判斷白名單，`Context.Pointer` 仍是 ""，所以看欄位的檢查照舊略過。
  - `func CheckNewTermsAt(r *rules.Rules, provisional []rules.Term, file, pointer, kr, zh string) []validate.Issue`：同上。

- [ ] **Step 1：寫會失敗的測試**
  - `TestCheckNewAtRubyCustomize`（batch）：用 `rules.Load("../..")`。
    - `CheckNewAt(r, "StoryData/P10916.json", "/dataList/3/teller", "요시히데", "<rubycustomize=ヨシヒデ>良秀</rubycustomize>")` 沒有硬性問題；
    - 同一句放在 `/dataList/3/content` 有硬性的 `rubycustomize`。
  - `TestTwinsDoNotCarryRubyCustomizeOutOfTheWhitelist`（batch，對應 Review Focus 1）：兩個同 KR 的單位，一個是 `teller`（有新標籤的紀錄），一個是 `content`。`content` 不得被填入這個譯文。
  - `TestApplyRewritesKeepsRubyCustomizeInTheWhitelist`（plan，對應 Review Focus 2）：改寫把 `teller` 的「良秀」改成 `<rubycustomize=ヨシヒデ>良秀</rubycustomize>`，結果套用到紀錄。
  - `TestImpactAcceptsWhitelistedRubyCustomize`（impact，對應 Review Focus 2）：白名單位置的新標籤紀錄不計入新增的硬性問題。
- [ ] **Step 2：確認失敗**　`go test ./internal/batch/ ./internal/plan/ ./internal/impact/ -run 'RubyCustomize'`
- [ ] **Step 3：實作**
  - 每個呼叫端都設 `RubyCustomize: r.RubyCustomizeAt(file, pointer)`：
    - 建置用 `u.File`、`u.Pointer`；
    - plan 從 key 拆出 pointer；
    - impact、twins 用 `u.Pointer`。
  - `checkNewTerms` 用 judge 傳進來的 pointer。
- [ ] **Step 4：確認通過**　`go vet ./... && go test ./...`
- [ ] **Step 5：提交**　訊息「檢查的呼叫端依白名單填寫 RubyCustomize」

### Task 5：照抄例外 `translate_identical`

**Files：**
- Modify: `internal/fields/fields.go`（`manualRules`、`LoadManual`、`Classify`）、`rules/fields.manual.yaml`
- Test: `internal/fields/fields_test.go`、`internal/build/build_test.go`

**Interfaces：**
- Produces：`manualRules.TranslateIdentical []FieldGlob \`yaml:"translate_identical"\``，`type FieldGlob struct { FileGlob string \`yaml:"file_glob"\`; Field string \`yaml:"field"\` }`。比對方式和 `name_only` 相同：`path.Match(FileGlob, path.Base(u.File))`，欄位完全相同。

- [ ] **Step 1：寫會失敗的測試**
  - `TestClassifyTranslateIdentical`：用 `LoadManual("../..")` 或一份暫存的 `fields.manual.yaml`。
    - `ScenarioModelCodes-AutoCreated.json` 的 `name`，KR、EN、JP 都是「UNGEZIEFER KAISER」→ `Translate == true`；
    - 同檔的 `nickName`，三語都是「LCD」→ `Reason == ReasonIdenticalValue`（對應 Review Focus 5）；
    - 同檔的 `id` → `ReasonIdentifierField`。
  - `TestLoadManualRejectsBadTranslateIdentical`：`file_glob` 是空的、`field` 是空的、`file_glob: "["`，各回傳 `*RuleError`。
  - `TestRulesLintReportsBadFieldsManual`（`cmd/lct/rules_test.go`）：暫存 repo 的 `rules/fields.manual.yaml` 寫 `translate_identical: [{file_glob: '', field: name}]` → `lct rules lint` 的錯誤裡有一筆，`File` 是 `rules/fields.manual.yaml`。現在 `rules lint` 不讀這個檔，要加上。
  - `TestBuildKeepsCopyRecordOfTranslateIdenticalUnit`（build）：
    - 單位原本有 `origin: copy` 的紀錄，文字等於 KR，KR 雜湊相同；
    - 加了例外後建置 → 語言包字串仍是 KR，這筆不排入待譯；
    - 紀錄換成人工翻譯「`<rubycustomize=UNGEZIEFER KAISER>害蟲皇帝</rubycustomize>`」後建置 → 輸出是轉換後的字串（這部分可以等 Task 6 完成後再補）。
- [ ] **Step 2：確認失敗**　`go test ./internal/fields/`
- [ ] **Step 3：實作**
  - `Classify` 的第 7 條加上「不在 `translate_identical`」的條件，並更新文件註解的條列。
  - `cmd/lct/rules.go` 的 `runRulesLint` 呼叫 `fields.LoadManual(ctx.Root)`。是 `*fields.RuleError` 就加進 `report.Errors`，`File` 填 `rules/fields.manual.yaml`；其他錯誤照現有方式當成系統錯誤。
  - `rules/fields.manual.yaml` 加入：
    ```yaml
    translate_identical:   # KR、EN、JP 三語相同也照樣翻譯的欄位；理由見 docs/superpowers/specs/2026-11-16-ruby-customize-design.md 第 9 節
      - { file_glob: 'ScenarioModelCodes*', field: name }   # 說話者名稱：官方保留原文的名字也要能加注音（UNGEZIEFER KAISER）
    ```
- [ ] **Step 4：確認通過**　`go vet ./... && go test ./...`
- [ ] **Step 5：提交**　訊息「欄位規則：translate_identical，ScenarioModelCodes 的 name 三語相同也送翻」

### Task 6：建置時轉換

**Files：**
- Modify:
  - `internal/build/build.go`：`buildFile`，以及報告的 `Report`；
  - `internal/build/pack.go`：寬度；
  - `internal/build/report.go`：報告多一行。
- Test: `internal/build/build_test.go`、`internal/build/rubycustomize_test.go`

**Interfaces：**
- Consumes：`rubyx.Convert`、`rubyx.Has`、`rules.Rules.RubyCustomizeAt`、`rules.Rules.RubyStyle`
- Produces：
  - `func (in *inputs) advance(r rune) float64`：`in.face` 的 GlyphAdvance，在 ppem 等於 unitsPerEm 時的值，除以 unitsPerEm；缺字（glyph 0）回傳 1。
  - `Report.RubyConverted int \`json:"ruby_converted"\``

- [ ] **Step 1：寫會失敗的測試**
  - `TestConvertWithShippedFontMatchesInGameTests`：
    - 載入 `../../assets/fonts/SarasaGothicTC-Bold.ttf`，用 `advance` 轉換設計第 6 節的兩個實例；
    - 結果和 Task 1 的期望字串逐字相同。這是對實測結果的守門測試。
  - `TestBuildConvertsWhitelistedRubyCustomize`：用既有的測試語言包工具，加一個 `ScenarioModelCodes-AutoCreated.json`。
    - `name` 的紀錄是新標籤 → 語言包裡是轉換後的字串；
    - `Report.RubyConverted == 1`；
    - 語言包裡沒有 `<rubycustomize`。
  - `TestBuildFallsBackOnMalformedRubyCustomize`（對應 Review Focus 3）：白名單位置的紀錄是 `<rubycustomize=>良秀</rubycustomize>` → 建置成功，這個單位顯示代替文字並排入待譯（`ReasonStructuralFail`）。
  - `TestEnsureNoRubyCustomize`：直接測試防護函式 `func ensureNoRubyCustomize(file, pointer, text string) error`。
    - 含 `<rubycustomize` 或 `</rubycustomize` 時回傳錯誤，錯誤訊息包含檔名與 pointer；
    - 轉換後的字串與一般文字回傳 nil。
    - `buildFile` 在每個單位寫入語言包前呼叫它。
- [ ] **Step 2：確認失敗**　`go test ./internal/build/ -run 'RubyCustomize|ShippedFont'`
- [ ] **Step 3：實作**
  - `buildFile` 在 `b.fill` 之後、`b.glyphs.addString` 之前處理：
    - `rubyx.Has(text) && b.rules.RubyCustomizeAt(file, u.Pointer)` 時，`text = rubyx.Convert(text, style, in.advance)`，計數加一；
    - 接著呼叫 `ensureNoRubyCustomize(file, u.Pointer, text)`，訊息寫成 `build: %s %s: <rubycustomize> left in the pack`。
  - `style` 在建置開始時由 `RubyStyle()` 取得，錯誤即失敗。
  - `report.go` 在摘要加一行轉換數。
- [ ] **Step 4：確認通過**　`go vet ./... && go test ./...`
- [ ] **Step 5：提交**　訊息「建置：白名單位置的 rubycustomize 依字型轉換成 TextMeshPro 標籤」

### Task 7：網站與更新頁

**Files：**
- Modify: `internal/site/file.go:168`、`internal/site/export.go:185`、`internal/history/compare.go:98`、`site/src/tmp/parse.ts`
- Test: `internal/site/file_test.go`、`internal/history/compare_test.go`、`site/src/tmp/parse.test.ts`、`site/src/tmp/TmpText.test.tsx`

**Interfaces：**
- Consumes：`rubyx.Restore`
- Produces（TS）：`export function restoreRubyCustomize(s: string): string`，在 `parseTMP` 一開始呼叫。正規表示式和 Go 的 `Restore` 相同，換成 `<ruby=R>B</ruby>`。

- [ ] **Step 1：寫會失敗的測試**
  - Go：
    - `TestUnitOriginSeesThroughRubyCustomize`（site）：紀錄是 `<rubycustomize=ヨシヒデ>良秀</rubycustomize>`（origin human），語言包是轉換後的字串 → `u.O == "human"`，不是 `stand-in`；
    - history 的 `origin` 寫同樣的測試；
    - gold：同樣情況 `Shipped` 是空的。
  - TS：
    - `restoreRubyCustomize('<nobr>良秀<space=-2em><voffset=1em><size=50%>ヨシヒデ</size></voffset><space=0em></nobr>')` → `'<ruby=ヨシヒデ>良秀</ruby>'`；
    - `'<nobr>甲乙</nobr>'` 不變（對應 Review Focus 4）；
    - `TmpText` 把轉換後的字串顯示成含 `<rt>ヨシヒデ</rt>` 的 `<ruby>`；
    - `stripTags`（`site/src/search/match.ts`）處理轉換後的字串後，同時包含「良秀」與「ヨシヒデ」。
  - Go：`TestExportSearchCorpusKeepsCopiesOfTranslateIdenticalUnits`（site）。
    - 情境：`ScenarioModelCodes` 的 `name` 三語相同，在 `translate_identical` 裡，紀錄仍是 `origin: copy`（舊譯文補正前就是這樣）。
    - 預期：這個單位留在搜尋語料裡。
    - 原因：`export.go` 現在只收 `Reason == fields.ReasonIdenticalValue` 的照抄單位。加了例外後，`Classify` 改判為要翻譯，UNGEZIEFER KAISER 會從搜尋消失。
- [ ] **Step 2：確認失敗**　`go test ./internal/site/ ./internal/history/ -run RubyCustomize`；`cd site && npx vitest run src/tmp`
- [ ] **Step 3：實作**
  - 三處 Go 比對改成 `rec.ZH == pack || rec.ZH == rubyx.Restore(pack)`。
  - `export.go` 收照抄單位的條件改成：`Classify` 的結果是 `Translate` 或 `Reason == ReasonIdenticalValue`。
  - 用 `grep -rn "rec.ZH ==\|== rec.ZH" internal` 確認沒有其他「紀錄對語言包」的比對。
- [ ] **Step 4：確認通過**　`go vet ./... && go test ./...`；`cd site && npm ci && npm test && npm run typecheck && npm run build`
- [ ] **Step 5：提交**　訊息「網站、更新頁：認得轉換後的 rubycustomize」

### Task 8：批次標記、指引與文件

**Files：**
- Modify:
  - `internal/batch/batch.go`（`BatchUnit`）、`internal/batch/next.go`（建立單位處）
  - `schemas/batch.schema.json`、`prompts/translate.md`
  - `guides/base.md`（ruby 段落，約第 114～133 行）、`guides/README.md`
  - `docs/research/2026-11-15-game-text-logic/02-tags-ruby.md`
- Test: `internal/batch/batch_test.go`；contract 測試（`cmd/lct`）依現狀調整

**Interfaces：**
- Produces：`BatchUnit.RubyCustomize bool \`json:"ruby_customize,omitempty"\``

- [ ] **Step 1：寫會失敗的測試**　`TestBatchMarksRubyCustomizeUnits`：
  - 批次裡 `teller` 單位的 `ruby_customize` 是 true；
  - `content` 單位沒有這個鍵；
  - 批次 JSON 通過 `schemas/batch.schema.json` 的驗證（用既有的 schema 測試工具）。
- [ ] **Step 2：確認失敗**　`go test ./internal/batch/ -run RubyCustomize`
- [ ] **Step 3：實作**
  - **批次：**
    - 依 `rules.RubyCustomizeAt(file, pointer)` 設定標記；
    - schema 加上 `ruby_customize`（boolean）。
  - **`prompts/translate.md`：** 加一條：`ruby_customize` 為 true 的單位可以寫 `<rubycustomize=注音>本文</rubycustomize>`，時機照 `guides/base.md`「白名單位置的注音」；其他單位不可以寫。
  - **`guides/base.md` 的 ruby 段落**（依設計第 10 節，規則內容以設計為準）：
    1. 新增小節「白名單位置的注音」：
       - 位置見 `rules/ruby-customize.yaml`（說話者名稱、人格語音、播報員、戰鬥氣泡）；
       - 寫法 `<rubycustomize=注音>本文</rubycustomize>`，不手寫 `nobr`、`space`；
       - 時機和劇情 ruby 相同；例子：KR `요시히데` 寫 `<rubycustomize=ヨシヒデ>良秀</rubycustomize>`；
       - 注音要簡短，只寫讀音、原文或原意。
    2. 「說話者名稱（`teller`）、標題、地點：KR 有 ruby 才加」改成：
       - 說話者名稱（`teller`、`ScenarioModelCodes` 的 `name`）用新標籤；
       - 稱號（`title`、`nickName`）與地點（`place`）不加注音、照詞彙表；
       - 註明遊戲在這三個欄位不處理 `<ruby>`。
    3. 「劇情（`StoryData/` 與 RPG 對話 `RPGSystem/*dialogue*`）」改為只有 `StoryData/`。另外說明 RPG 對話：遊戲不處理 `<ruby>`，比照「其他檔案」寫「本文(注音)」，等實測後再考慮白名單。
       - 第 114、115 行提到「劇情與 RPG 對話改寫中文意思、原文寫在 ruby」「劇情與 RPG 對話改用 ruby」，也要一併改。
    4. 「其他檔案」一條補一句：白名單位置改用新標籤。
  - **`guides/README.md`：** 結構表加一列，說明 `rules/ruby-customize.yaml` 的用途。
  - **`02-tags-ruby.md`：** 在「結論」後補一句，指向新機制與設計文件。
- [ ] **Step 4：確認通過**　`go vet ./... && go test ./...`；`bin/lct rules lint --json` 沒有錯誤（`base.md` 有長度或格式檢查時會在這裡出現）
- [ ] **Step 5：提交**　訊息「指引與批次：白名單位置使用 rubycustomize；修正 teller／title／place 與 RPG 對話的 ruby 規則」

### Task 9：實際資料驗收

**Files：** 不改程式。結果記在 SDD 帳本裡。

- [ ] **Step 1：建置前後比較。**
  - 在 scratchpad 建一份暫存 clone：用 `git worktree add` 取出這個分支，並在它的 `.work/raw`、`.work/tw` 分別放 `origin/raw`、`origin/tw` 的 worktree，不動主 repo 的 `.work/`。
  - 分別用 `origin/main` 與這個分支編出的 `lct` 執行 `lct build --json`，每次建置前把 `.work/tw` 還原成 `origin/tw`。
  - 預期：兩份報告除了新增的 `ruby_converted`（應為 0）以外，`shown`、待譯數、`structural-fail` 數量都相同。有差異時逐筆查明原因，記在帳本裡。
  - 完成後刪除暫存 clone 與 worktree。
- [ ] **Step 2：網站確認。**
  - `bin/lct site --out site/public/data`；
  - `ScenarioModelCodes-AutoCreated.json` 的 #628、#630 的 `name`：
    - 預期 `o` 仍是 `copy`：照抄紀錄被當成可用的紀錄，見 Task 5；
    - 字串仍是「UNGEZIEFER KAISER」；
    - 搜尋 `UNGEZIEFER` 找得到這兩筆。
- [ ] **Step 3：檢查確認。**
  - `bin/lct validate --kr '그레고르' --zh '<rubycustomize=UNGEZIEFER KAISER>害蟲皇帝</rubycustomize>' --file ScenarioModelCodes-AutoCreated.json --pointer /dataList/630/name` → 沒有問題；
  - 同一句換成 `--pointer /dataList/630/nickName` → 硬性的 `rubycustomize`。
