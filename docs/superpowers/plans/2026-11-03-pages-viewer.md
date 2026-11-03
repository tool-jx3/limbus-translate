# 譯文對照網站（GitHub Pages）Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 做一個部署在 GitHub Pages 的靜態網站：找檔案、全文搜尋、四語（KR／JP／EN／TW）對照、劇本模式連貫閱讀、譯文來源標記與篩選。

**Architecture:** 新指令 `lct site --out DIR`（Go，`internal/site`）讀 raw 與 tw worktree、沿用 `internal/align` 的對齊與 `.meta/units` 的紀錄，輸出 spec §5 的 JSON；`site/`（Preact＋Vite＋TypeScript）是 hash 路由的單頁應用，執行時按需載入這些 JSON，全文搜尋在 Web Worker 比對整份 corpus；`pages.yml` 在 `update`／`rebuild` 成功後產生資料、build 前端並部署，資料不 commit 到任何分支。

**Tech Stack:** Go 1.26.1（沿用）；Node 26（`site/.nvmrc`，`engines.node: ">=22.12"`）；npm；preact 10.29.8、vite 8.3.5、@preact/preset-vite 2.10.6、typescript 7.0.2、vitest 5.1.0、jsdom 30.1.3（以 `package-lock.json` 鎖定，不加其他執行期依賴）；GitHub Actions：`actions/checkout@v5`、`actions/setup-go@v6`、`actions/setup-node@v7`、`actions/configure-pages@v6`、`actions/upload-pages-artifact@v5`、`actions/deploy-pages@v5`。

**Spec:** `docs/superpowers/specs/2026-11-03-pages-viewer-design.md`（以下稱「site spec」）；主 spec `docs/superpowers/specs/2026-10-31-limbus-zh-tw-translation-design.md`（§3 限制、§6.5 對齊、§12 Actions、§15 指令、§16 Repo 結構）

**本計畫範圍外**：site spec §2「之後再考慮」的項目（詞彙高亮、回報 issue、Release 差異）；多版本；後端。

## Global Constraints

- 資料格式照 site spec §5，`format: 1`；JSON 鍵名照抄：meta（`format`、`raw_tag`、`rules_commit`、`tw_commit`、`release`、`langs`、`files`、`units`）、index（`families[].id`、`families[].files`、`files[].path`、`family`、`label`、`speakers`、`units`、`origins`、`missing.en`、`missing.jp`）、檔案（`path`、`family`、`units[].p`、`f`、`i`、`kr`、`jp`、`en`、`zh`、`o`、`fl`）、搜尋（`search/keys.json` 為 `[[檔案編號, "指標"], …]`，`search/<kr|jp|en|zh>.json` 為等長的字串陣列）
- `o` 的值：紀錄的 `origin`（`paratranz`、`claude`、`human`、`copy`、`fallback`），或 `stand-in`（沒有紀錄，或語言包字串與紀錄的 `zh` 不同）
- JSON 以 UTF-8、無 BOM、無縮排輸出；同樣的輸入產生位元組相同的輸出
- 網站資料只由 CI 產生並部署，不 commit 到任何分支；`site/public/data/`、`site/dist/`、`site/node_modules/` 列入 `.gitignore`
- 網站名稱與頁面不得使用「Claude Code」「Anthropic」或其標誌；可以寫「使用 Claude Code 翻譯」
- 頁尾授權文字（照抄）：「KR／JP／EN 原文著作權屬 Project Moon；本站非商業、供翻譯校對使用。TW 譯文採用 CC BY-NC-SA 4.0，含 Paratranz 社群譯文（第一階段匯入）。」
- noindex：`index.html` 有 `<meta name="robots" content="noindex, nofollow">`；`site/public/robots.txt` 內容為 `User-agent: *` 與 `Disallow: /` 兩行
- 前端不使用 `innerHTML`／`dangerouslySetInnerHTML`；不使用外部 CDN；不加分析工具
- 各語言欄的 `lang` 屬性：KR `ko`、JP `ja`、EN `en`、TW `zh-Hant-TW`
- Vite `base: './'`（網站在 `/<repo>/` 底下）；hash 路由：`#/`、`#/f/<路徑>?u=<指標>`、`#/s?q=<字串>&lang=<kr|jp|en|zh>&re=1`
- 每個 CLI 指令支援 `--json`；exit code 0 成功、1 使用者錯誤、2 系統錯誤
- `pages.yml`：觸發 `workflow_run`（`update`、`rebuild`，`completed`）與 `workflow_dispatch`；job `if` 與 `rules-proposal.yml` 相同；權限 `contents: read`、`pages: write`、`id-token: write`；`concurrency: { group: pages, cancel-in-progress: false }`；`environment: github-pages`；`runs-on: ubuntu-24.04`
- workflow 不寫模型名稱；不讀、不寫、不依賴 tool-jx3/Assets 與 tool-jx3/ParatranzUploader
- 提交前 `go vet ./... && go test ./...`；動到 `site/` 的任務另跑 `cd site && npm ci && npm run typecheck && npm test && npm run build`
- commit 訊息結尾：
  ```
  Co-Authored-By: <your session's attribution line>
  Claude-Session: https://claude.ai/code/session_015XrjRvHe6FfxjemSS1js35
  ```
- 開發分支 `claude/festive-wright-g2k853`；每個任務結束 `git push -u origin claude/festive-wright-g2k853`；`main` 只在 Task 8 經使用者同意後以 fast-forward 更新

## Review Focus

1. **看起來像標籤、其實不是的文字**（但丁台詞整句包在 `<…>` 裡、`<B 사무소…>`、`<3`、`a < b`）：必須照原樣顯示成文字，不能被吃掉或當成 HTML — Task 3 `parse keeps non-TMP angle text`
2. **上千單位的大檔**（`Skills.json`、`BattleKeywords*`）與深連結指向檔尾的單位：畫面不能卡住，深連結要捲到正確的列 — Task 5 `virtual window includes deep-link target`
3. **部署更新時頁面還開著**（舊的 `index.json` 對上新的檔案、或 `format` 變了）：要提示重新整理，而不是顯示錯亂的資料 — Task 4 `format mismatch shows reload notice`、`fetch 404 shows reload notice`
4. **病態的正規表示式或命中大量結果**（`.*`、`(a+)+$`、搜「的」）：介面不能凍結；新的搜尋取代舊的 — Task 6 `newer query supersedes older`、`results capped at 1000 with total`
5. **只有 KR 的檔與沒有對應的語言**（KR 比 EN 多 53 個檔；JP 235、EN 351 個單位沒對齊）：該欄顯示「—」並以 `missing` 計數，不能顯示成空白或錯位 — Task 1 `TestBuildFileMissingLanguage`、Task 5 `missing language cell shows dash`

## File Structure

| 檔案 | 責任 | 任務 |
|---|---|---|
| `internal/site/file.go` | 單一檔案的對照資料與摘要 | 1 |
| `internal/site/export.go`、`internal/site/order.go`、`cmd/lct/site.go` | 排序、索引、搜尋 corpus、meta、寫檔；`lct site` | 2 |
| `site/package.json`、`site/package-lock.json`、`site/.nvmrc`、`site/vite.config.ts`、`site/tsconfig.json`、`site/index.html`、`site/public/robots.txt`、`site/src/types.ts`、`site/src/tmp/` | 前端骨架、資料型別、標籤解析與呈現 | 3 |
| `site/src/data.ts`、`site/src/route.ts`、`site/src/settings.ts`、`site/src/app/` | 資料載入、路由、設定、版面、目錄與檔案篩選 | 4 |
| `site/src/view/` | 檔案檢視：表格模式、劇本模式、篩選、上一檔／下一檔、深連結、虛擬捲動 | 5 |
| `site/src/search/` | 全文搜尋（worker、比對、結果頁） | 6 |
| `.github/workflows/pages.yml`、`.github/workflows/ci.yml`、`cmd/lct/contract_test.go`、`README.md`、`CLAUDE.md`、`.gitignore`、主 spec §12／§15／§16 | 部署、CI、合約測試、文件 | 7 |

---

### Task 1: `internal/site` — 單一檔案的對照資料

**Files:**
- Create: `internal/site/file.go`
- Test: `internal/site/file_test.go`（測試資料在測試裡寫入暫存目錄：KR／EN／JP 三個 pack、一個建置後的 zh 包、紀錄 map）

**Interfaces:**
- Consumes: `pack.Pack`（`Files`、`Has`、`Doc`）、`jsonx.Node`（`Strings() []StringField`、`Get(pointer)`）、`units.Extract(file, doc) []units.Unit`（`Pointer`、`Field`、`KR`、`ItemIndex`）、`align.NewFileAligner(file, krDoc, en, jp)`（`Get(pointer) (en, jp *LangAligned, err)`、`Unmatched() (en, jp int, err)`）、`store.Record`、`rules.Rules`（`Family(file) string`）
- Produces:
  - `type Inputs struct { KR, EN, JP, ZH *pack.Pack; Records map[string]store.Record; Rules *rules.Rules }`（`ZH` 是 `<tw>/LimbusCompany_Data/Lang/<lang_dir>`，以空前綴 `pack.Open` 開啟）
  - `const OriginStandIn = "stand-in"`
  - `type Unit struct { P string \`json:"p"\`; F string \`json:"f"\`; I int \`json:"i"\`; KR string \`json:"kr"\`; JP *string \`json:"jp,omitempty"\`; EN *string \`json:"en,omitempty"\`; ZH string \`json:"zh"\`; O string \`json:"o"\`; Fl []string \`json:"fl,omitempty"\` }`
  - `type File struct { Path string \`json:"path"\`; Family string \`json:"family"\`; Units []Unit \`json:"units"\` }`
  - `type Missing struct { EN int \`json:"en"\`; JP int \`json:"jp"\` }`
  - `type FileSummary struct { Path, Family, Label string; Speakers []string; Units int; Origins map[string]int; Missing Missing }`，JSON 鍵 `path`、`family`、`label`、`speakers`、`units`、`origins`、`missing`
  - `func BuildFile(in Inputs, path string) (File, FileSummary, error)`

- [ ] **Step 1: 寫失敗的測試**
  - `TestBuildFileAlignsByIDNotPosition`：KR `dataList` 的 `id` 為 1、2、3，JP 同樣三筆但順序是 2、1、3 → 指標 `/dataList/0/content` 的 `JP` 是 JP 中 `id` 1 那筆的文字
  - `TestBuildFileMissingLanguage`：EN 缺 `id` 3 → 該單位 `EN == nil`、`summary.Missing.EN == 1`；EN pack 整個沒有這個檔 → 所有單位 `EN == nil`、`Missing.EN == 0`
  - `TestBuildFileOrigin`：紀錄 `zh` 與 zh 包字串相同 → `O` 是紀錄的 `origin`；不同 → `O == "stand-in"`；沒有紀錄 → `"stand-in"`；紀錄 `Flags: ["needs_review"]` → `Fl` 相同；`summary.Origins` 計數正確
  - `TestBuildFileItemIndex`：`dataList` 底下的單位 `I` 等於元素索引；`dataList` 以外的字串 `I == -1`
  - `TestBuildFileStoryLabelAndSpeakers`：`StoryData/S9.json`，第一個非空 `place` 的 zh 是「某地」→ `Label == "某地"`；`teller` 的 zh 為「甲、乙、甲、丙」→ `Speakers == ["甲","乙","丙"]`；25 個不同說話者 → 只留前 20 個；非 `StoryData/` 的檔 `Label == ""`
  - `TestBuildFileJSONShape`：一個兩單位的小檔（其中一個沒有 JP）`json.Marshal(file)` 等於寫死的字串（鍵順序 `p,f,i,kr,jp,en,zh,o`，沒有 JP 的單位不出現 `"jp"`）

- [ ] **Step 2: 執行測試確認失敗**

Run: `go test ./internal/site/ -run BuildFile`
Expected: FAIL（package `site` 不存在）

- [ ] **Step 3: 實作 `BuildFile`**（zh 字串取自 `in.ZH.Doc(path)` 的 `Strings()` 依指標建表；對齊每檔只建一次 `FileAligner`）

- [ ] **Step 4: 執行測試確認通過**

Run: `go vet ./... && go test ./internal/site/`
Expected: PASS

- [ ] **Step 5: Commit 並 push**

```bash
git add internal/site
git commit -m "feat(site): the aligned units of one file for the viewer"
git push -u origin claude/festive-wright-g2k853
```

### Task 2: `lct site` — 排序、索引、搜尋 corpus、meta 與寫檔

**Files:**
- Create: `internal/site/order.go`、`internal/site/export.go`、`cmd/lct/site.go`
- Modify: 主 spec §15（指令一覽加 `lct site --out DIR`）
- Test: `internal/site/order_test.go`、`internal/site/export_test.go`、`cmd/lct/site_test.go`

**Interfaces:**
- Consumes: Task 1 的 `Inputs`、`BuildFile`、`FileSummary`；`rules.Rules.Routing.Priority`；`raw.LastSource(rawWorktree)`；`gitx.TagsAt(dir, "zh-TW-")`；tw 的 `.meta/build.json`（讀 rules commit 欄位，沿用 `cmd/lct/rulescommit.go` 或 build 的讀法）
- Produces:
  - `func NaturalLess(a, b string) bool`（數字段依數值比較：`S2` < `S10`、`P10102` < `P10205`；其餘依位元組）
  - `func OrderFiles(paths []string, r *rules.Rules) []string`（族群依 `Routing.Priority`，族群內 `NaturalLess`）
  - `type Meta struct { Format int; RawTag, RulesCommit, TWCommit, Release string; Langs []string; Files, Units int }`，JSON 鍵照 Global Constraints；`Format` 固定 1、`Langs` 固定 `["kr","jp","en","zh"]`
  - `type IndexFamily struct { ID string \`json:"id"\`; Files []int \`json:"files"\` }`；`type Index struct { Families []IndexFamily \`json:"families"\`; Files []FileSummary \`json:"files"\` }`
  - `type Result struct { Out string \`json:"out"\`; Files int \`json:"files"\`; Units int \`json:"units"\`; SearchUnits int \`json:"search_units"\`; Bytes int64 \`json:"bytes"\` }`
  - `func Export(in Inputs, meta Meta, outDir string) (Result, error)`：先清空 `outDir`，再寫 `meta.json`、`index.json`、`files/<路徑>.json`、`search/keys.json`、`search/{kr,jp,en,zh}.json`；檔案集合是 KR pack 的全部檔案
  - `lct site --out DIR [--json]`：`DIR` 相對於 repo 根目錄；必須在 repo 根目錄之內、不是根目錄本身、不含 `.git`，否則使用者錯誤（exit 1）；tw 沒有 `.meta/build.json` 時使用者錯誤

- [ ] **Step 1: 寫失敗的測試**
  - `TestNaturalLess`：`S2.json` < `S10.json`；`E003I3` < `E003I10`；`a` < `b`；相等時為 false
  - `TestOrderFilesByFamilyThenName`：兩個族群，priority 為 [B, A]；A 有 `S10`、`S2`，B 有 `X1` → `[X1, S2, S10]`
  - `TestExportWritesSpecLayout`：小型 Inputs 匯出後的檔案清單恰好是 `meta.json`、`index.json`、`files/…`（每個 KR 檔一個）、`search/keys.json`、`search/kr.json`、`search/jp.json`、`search/en.json`、`search/zh.json`；`meta.format == 1`；`index.files` 的順序與 `OrderFiles` 相同、`families[].files` 是對應的索引
  - `TestExportSearchCorpus`：`o == "copy"` 的單位不在 corpus；四個語言陣列與 `keys` 等長；沒有 JP 對應的單位在 `jp.json` 是 `""`；`keys[n][0]` 是該檔在 `index.files` 的索引
  - `TestExportDeterministic`：匯出兩次到不同目錄，所有檔案位元組相同
  - `TestExportClearsOutDir`：`outDir` 裡先放 `stale.json` → 匯出後不存在
  - `cmd/lct`：`TestSiteCommandJSON`（以 `build_test.go` 的方式建好 raw 與 tw）→ `lct site --out work/site --json` 輸出含 `files`、`units`、`search_units`、`bytes`，`work/site/meta.json` 的 `raw_tag` 是測試快照的 tag；`TestSiteCommandRefusesOutDir`：`--out .`、`--out ..`、`--out /tmp/x`、`--out .git/x` 都是 exit 1，且沒有刪除任何檔案

- [ ] **Step 2: 執行測試確認失敗**

Run: `go test ./internal/site/ ./cmd/lct/ -run 'NaturalLess|OrderFiles|Export|SiteCommand'`
Expected: FAIL（未定義 `NaturalLess`、`Export`、指令 `site`）

- [ ] **Step 3: 實作**（寫檔用暫存檔再 rename；JSON 用 `encoding/json`，`SetEscapeHTML(false)`）

- [ ] **Step 4: 執行測試確認通過，並以實際資料量測**

Run: `go vet ./... && go test ./...`
Expected: PASS

Run: `go build -o bin/lct ./cmd/lct && time bin/lct site --out work/site --json`
Expected: exit 0，`files` 約 2,420、`units` 約 27 萬；耗時 3 分鐘內、`bytes` 300 MB 以內（實際數字寫進 commit 訊息）

- [ ] **Step 5: Commit 並 push**

```bash
git add internal/site cmd/lct/site.go cmd/lct/site_test.go docs/superpowers/specs/2026-10-31-limbus-zh-tw-translation-design.md
git commit -m "feat(site): lct site writes the index, the files and the search corpus"
git push -u origin claude/festive-wright-g2k853
```

### Task 3: 前端骨架、資料型別與標籤呈現

**Files:**
- Create: `site/package.json`、`site/package-lock.json`、`site/.nvmrc`（內容 `26`）、`site/vite.config.ts`、`site/tsconfig.json`、`site/index.html`、`site/public/robots.txt`、`site/src/main.tsx`、`site/src/types.ts`、`site/src/tmp/parse.ts`、`site/src/tmp/TmpText.tsx`、`site/src/styles.css`、`site/fixtures/data/`（小型的 meta／index／2 個檔案／搜尋 corpus，格式照 site spec §5，供各任務的測試使用）
- Modify: `.gitignore`（`site/node_modules/`、`site/dist/`、`site/public/data/`）
- Test: `site/src/tmp/parse.test.ts`、`site/src/tmp/TmpText.test.tsx`

**Interfaces:**
- Produces:
  - npm scripts：`dev`、`build`（`vite build`）、`preview`、`test`（`vitest run`）、`typecheck`（`tsc --noEmit`）
  - `site/src/types.ts`：`Meta`、`IndexFamily`、`FileSummary`、`Index`、`Unit`、`FileData`、`Origin = 'paratranz'|'claude'|'human'|'copy'|'fallback'|'stand-in'`、`Lang = 'kr'|'jp'|'en'|'zh'`，欄位與 Task 1、2 的 JSON 鍵相同
  - `type TmpNode = {t:'text', s:string} | {t:'token', s:string} | {t:'br'} | {t:'sprite', name:string} | {t:'tag', name:string, value?:string, kids:TmpNode[]}`
  - `parseTMP(s: string): TmpNode[]`：標籤名稱白名單與 `internal/validate` 的 `Tags` 相同（`color`、`size`、`b`、`i`、`u`、`s`、`ruby`、`style`、`link`、`noparse`、`sprite`、`br`、`material`、`font`、`voffset`、`width`、`uppercase` 等，以 `internal/validate/validate_test.go` 的 `TestOnlyKnownTagNamesAreTags` 為準），名稱不分大小寫；白名單以外的 `<…>` 是文字；`[ID]`（`[` 英數 `]`）與 `{n}`、`{Slot}` 等佔位符成為 `token`；`\n` 成為 `br`
  - `<TmpText text={string} raw={boolean} lang={Lang} />`：`raw` 時以等寬字型顯示原字串
  - 呈現對照：`color` → `style.color`（只接受 `#` 加 3／6／8 位十六進位）；`size=N%` → `font-size: (N/100)em`，限制在 0.75～1.5；`size=N`（絕對值）→ `1.25em`；`b`／`i`／`u`／`s` → 對應元素；`ruby=X` → `<ruby>內文<rt>X</rt></ruby>`；`sprite name=X` → 文字 `[sprite:X]`；其他白名單標籤只保留內文；沒有開頭的結束標籤忽略
- `index.html`：`<html lang="zh-Hant-TW">`、noindex meta、`<title>` 為網站名稱（遵守命名限制）
- `styles.css`：以 CSS 變數定義顏色，`@media (prefers-color-scheme: dark)` 時換成深色；字型只用系統字型（`system-ui` 與各語言的系統 CJK 字型），不打包字型檔

- [ ] **Step 1: 建骨架並寫失敗的測試**
  - `parse.test.ts`：`'<color=#e30000>紅</color>字'` → tag(color,'#e30000',[text 紅]) + text 字；巢狀 `<b><i>x</i></b>`；`'<ruby=天退星>插翅虎</ruby>'` → tag ruby；`'甲\n乙'` → text、br、text；`'[Burn] {0}'` → token、text、token；**`parse keeps non-TMP angle text`**：`'<N公司……>'`、`'<B 사무소 해결사>'`、`'<3'`、`'a < b'` 全部只產生 text、內容與原字串相同；`'</size>前'` → 結束標籤被忽略、text 前
  - `TmpText.test.tsx`（jsdom）：`<script>alert(1)</script>` 與 `<img src=x onerror=alert(1)>` 顯示為文字且容器內沒有 `script`、`img` 元素；ruby 產生 `ruby` 與 `rt`；`<size=200%>` 的 font-size 為 `1.5em`；`<color=red>`（非十六進位）不設顏色；`raw` 時 `textContent` 等於原字串

- [ ] **Step 2: 執行測試確認失敗**

Run: `cd site && npm ci && npm test`
Expected: FAIL（`parseTMP`、`TmpText` 未定義）

- [ ] **Step 3: 實作 `parseTMP` 與 `TmpText`**（只用 Preact 建立元素，不用 `innerHTML`）

- [ ] **Step 4: 執行測試確認通過**

Run: `cd site && npm run typecheck && npm test && npm run build`
Expected: PASS；`site/dist/index.html` 含 noindex meta、`site/dist/robots.txt` 存在

- [ ] **Step 5: Commit 並 push**

```bash
git add site .gitignore
git commit -m "feat(site): the viewer's skeleton, its data types and the tag renderer"
git push -u origin claude/festive-wright-g2k853
```

### Task 4: 資料載入、路由、設定、版面與目錄

**Files:**
- Create: `site/src/data.ts`、`site/src/route.ts`、`site/src/settings.ts`、`site/src/app/App.tsx`、`site/src/app/Directory.tsx`、`site/src/app/Footer.tsx`、`site/src/app/Home.tsx`、`site/src/app/Notice.tsx`
- Test: `site/src/route.test.ts`、`site/src/data.test.ts`、`site/src/settings.test.ts`、`site/src/app/Directory.test.tsx`

**Interfaces:**
- Consumes: Task 3 的 `types.ts`、`TmpText`
- Produces:
  - `type Route = {kind:'home'} | {kind:'file', path:string, ptr?:string} | {kind:'search', q:string, lang:Lang, re:boolean}`
  - `parseHash(hash: string): Route`、`fileHref(path: string, ptr?: string): string`、`searchHref(q: string, lang: Lang, re: boolean): string`（路徑每段 `encodeURIComponent`，`/` 保留）
  - `class DataError extends Error { kind: 'format'|'missing'|'network' }`；`loadMeta(): Promise<Meta>`、`loadIndex(): Promise<Index>`、`loadFile(path): Promise<FileData>`（相對於 `./data/`，同一路徑只抓一次；`meta.format !== 1` → `DataError('format')`；HTTP 404 → `DataError('missing')`）
  - `type Settings = { langs: Record<Lang, boolean>; showCopy: boolean; origins: Record<Origin, boolean>; rawTags: boolean; scriptMode: 'auto'|'on'|'off' }`；`loadSettings(): Settings`、`saveSettings(s)`（`localStorage` 鍵 `lct-viewer-settings`；壞掉或不存在就用預設：四語開、`showCopy: false`、所有來源開但 `copy` 關、`rawTags: false`、`scriptMode: 'auto'`）
  - `filterFiles(index: Index, q: string): number[]`：比對 `path`、`label`、`speakers`（不分大小寫、子字串），回傳 `index.files` 的索引
  - 頁尾：授權文字照 Global Constraints；資料版本顯示 `raw_tag` 與 `release`（`release` 非空時連到 `https://github.com/tool-jx3/limbus-translate/releases/tag/<release>`）

- [ ] **Step 1: 寫失敗的測試**
  - `route.test.ts`：`parseHash('')`、`'#/'` → home；`fileHref('StoryData/S001A.json','/dataList/14/content')` 再 `parseHash` 得到相同的 path 與 ptr；含空白與 `%` 的路徑來回相同；`'#/s?q=%E5%BE%9E&lang=zh&re=1'` → search、`re: true`；未知路徑 → home
  - `data.test.ts`（以假的 `fetch`）：讀 `site/fixtures/data/meta.json` 成功；**`format mismatch shows reload notice`**：`format: 2` → `DataError` 的 `kind == 'format'`；**`fetch 404 shows reload notice`**：檔案 404 → `kind == 'missing'`；同一路徑呼叫兩次只 fetch 一次
  - `settings.test.ts`：沒有存檔 → 預設值；存檔是壞掉的 JSON → 預設值；存後再讀相同
  - `Directory.test.tsx`：fixture 的 index 依族群列出；`filterFiles(index,'s001')` 命中 `StoryData/S001A.json`；以說話者名稱也能命中

- [ ] **Step 2: 執行測試確認失敗**

Run: `cd site && npm test`
Expected: FAIL

- [ ] **Step 3: 實作**（頁首：網站名稱、檔案篩選框、全文搜尋框（送出時導向 `searchHref`，語言預設 `zh`）、四個語言開關（寫入 `Settings.langs`）；`DataError` 由 `Notice` 顯示「資料已更新，請重新整理」與重新整理按鈕；左側目錄在寬度 900px 以下收成抽屜）

- [ ] **Step 4: 執行測試確認通過**

Run: `cd site && npm run typecheck && npm test && npm run build`
Expected: PASS

- [ ] **Step 5: Commit 並 push**

```bash
git add site
git commit -m "feat(site): data loading, routes, settings, layout and the file directory"
git push -u origin claude/festive-wright-g2k853
```

### Task 5: 檔案檢視（表格模式、劇本模式、篩選、連貫閱讀）

**Files:**
- Create: `site/src/view/FileView.tsx`、`site/src/view/TableView.tsx`、`site/src/view/ScriptView.tsx`、`site/src/view/group.ts`、`site/src/view/nav.ts`、`site/src/view/virtual.ts`、`site/src/view/OriginBadge.tsx`
- Test: `site/src/view/group.test.ts`、`site/src/view/nav.test.ts`、`site/src/view/virtual.test.ts`、`site/src/view/FileView.test.tsx`

**Interfaces:**
- Consumes: Task 3 `TmpText`、`types.ts`；Task 4 `loadFile`、`loadIndex`、`Settings`、`fileHref`
- Produces:
  - `type Entry = { i: number; units: Unit[]; place?: Unit; teller?: Unit; title?: Unit; placeChanged: boolean }`
  - `groupEntries(units: Unit[]): Entry[]`：相鄰且 `i` 相同（且 `i >= 0`）的單位合成一段；`i == -1` 的單位各自一段；`f` 為 `place`、`teller`、`title` 的單位放進對應欄位，其餘留在 `units`；`placeChanged` 在 `place` 的 KR 與上一個有 `place` 的段不同時為 true
  - `visibleUnits(units: Unit[], s: Settings): Unit[]`：依 `s.origins` 與 `s.showCopy` 篩選
  - `isScriptFile(path: string): boolean`：`StoryData/` 開頭，或 `RPGSystem/` 底下檔名含 `dialogue`
  - `neighbors(index: Index, path: string): { prev?: string; next?: string }`：依 `index.files` 的順序
  - `windowRange(count: number, rowHeight: number, viewportTop: number, viewportHeight: number, overscan: number, target?: number): [start: number, end: number]`：`count <= 1000` 時回傳全部；`target` 有值時範圍必含 `target`
  - 鍵盤：`[`、`]` 上一檔／下一檔；`j`、`k` 下一段／上一段（輸入框聚焦時不作用）

- [ ] **Step 1: 寫失敗的測試**
  - `group.test.ts`：同一 `i` 的 teller、title、content 合成一段且 `teller`、`title` 欄位正確；`i == -1` 各自一段；place 由「A」變「B」時 `placeChanged` 為 true，再出現「B」為 false；`visibleUnits` 在 `origins.claude == false` 時去掉 claude 單位、`showCopy == false` 時去掉 copy
  - `nav.test.ts`：第一個檔沒有 `prev`、最後一個沒有 `next`；中間的檔兩者皆有且依 `index.files` 順序
  - `virtual.test.ts`：`windowRange(500, …)` → `[0, 500]`；`windowRange(5000, 30, 0, 900, 20)` 只含前約 50 列；**`virtual window includes deep-link target`**：`windowRange(5000, 30, 0, 900, 20, 4800)` 含 4800
  - `FileView.test.tsx`（jsdom，fixture 檔）：`StoryData/` 的檔預設劇本模式、其他檔表格模式；`ptr` 指定的單位所在列有 `is-target` class；語言欄的 `lang` 屬性正確；**`missing language cell shows dash`**：沒有 `jp` 的單位，JP 欄文字為「—」；TW 欄有來源標記文字（例如 `claude`）；關掉 EN 後沒有 EN 欄

- [ ] **Step 2: 執行測試確認失敗**

Run: `cd site && npm test`
Expected: FAIL

- [ ] **Step 3: 實作**（劇本模式：寬度 900px 以上各語言並排，以下上下堆疊；整段的台詞都被篩掉時整段隱藏；捲到深連結目標後加上 `is-target` 並聚焦）

- [ ] **Step 4: 執行測試確認通過**

Run: `cd site && npm run typecheck && npm test && npm run build`
Expected: PASS

- [ ] **Step 5: Commit 並 push**

```bash
git add site
git commit -m "feat(site): the file view, in table and script mode, with filters and reading order"
git push -u origin claude/festive-wright-g2k853
```

### Task 6: 全文搜尋

**Files:**
- Create: `site/src/search/match.ts`、`site/src/search/engine.ts`、`site/src/search/worker.ts`、`site/src/search/client.ts`、`site/src/search/SearchView.tsx`
- Test: `site/src/search/match.test.ts`、`site/src/search/engine.test.ts`、`site/src/search/client.test.ts`

**Interfaces:**
- Consumes: Task 3 `parseTMP`（去標籤）；Task 4 `Route`、`searchHref`、`fileHref`、`loadIndex`
- Produces:
  - `stripTags(s: string): string`：去掉白名單標籤、保留文字與 `[ID]`、`{n}`
  - `makeMatcher(q: string, regex: boolean): ((s: string) => [number, number][]) | {error: string}`：子字串比對對拉丁字母不分大小寫；regex 用 `new RegExp(q, 'giu')`，語法錯誤回傳 `{error}`；空字串回傳 `{error}`
  - `type Hit = { file: number; ptr: string; ranges: [number, number][] }`
  - `searchCorpus(keys: [number, string][], texts: string[], match, limit = 1000): { hits: Hit[]; total: number }`（比對 `stripTags` 後的字串；`total` 是全部命中數）
  - worker 訊息：`{id, q, lang, re}` → `{id, hits, total} | {id, error}`；client 端 `search(q, lang, re): Promise<…>`，新的呼叫讓舊的 promise 以 `{superseded: true}` 結束、並忽略舊 `id` 的回覆
  - 結果頁：依檔案分組；每筆顯示命中語言的字串（命中處 `<mark>`）、同一單位的 KR 與 TW（以 `loadFile` 取得，只載入顯示中的檔）；點擊連到 `fileHref(path, ptr)`

- [ ] **Step 1: 寫失敗的測試**
  - `match.test.ts`：`stripTags('<color=#fff>甲</color>[Burn]')` → `'甲[Burn]'`；子字串 `'cinq'` 命中 `'Cinq協會'`；`'協會'` 命中兩處時回傳兩個範圍；regex `'對話 \\d+'` 命中 `'對話 3'`；`'('` → `{error}`；空字串 → `{error}`
  - `engine.test.ts`：fixture corpus 中 copy 以外的單位都能被搜到；**`results capped at 1000 with total`**：2,500 筆都命中 → `hits.length == 1000`、`total == 2500`
  - `client.test.ts`（以假 worker）：**`newer query supersedes older`**：先後呼叫兩次，第一個結果為 `{superseded: true}`、第二個為實際結果，即使舊回覆較晚到也不覆蓋

- [ ] **Step 2: 執行測試確認失敗**

Run: `cd site && npm test`
Expected: FAIL

- [ ] **Step 3: 實作**（worker 第一次搜某語言時抓 `data/search/keys.json` 與 `data/search/<lang>.json` 並快取；介面顯示「下載中」與命中總數）

- [ ] **Step 4: 執行測試確認通過**

Run: `cd site && npm run typecheck && npm test && npm run build`
Expected: PASS

- [ ] **Step 5: Commit 並 push**

```bash
git add site
git commit -m "feat(site): full-text search in a worker, by substring or regular expression"
git push -u origin claude/festive-wright-g2k853
```

### Task 7: 部署、CI、合約測試與文件

**Files:**
- Create: `.github/workflows/pages.yml`
- Modify: `.github/workflows/ci.yml`（新增 `site` job）、`cmd/lct/contract_test.go`（`workflowNames` 加 `pages.yml`、新增 `checkPagesWorkflow` 與 mutation、ci.yml 的 site job 檢查）、`README.md`（網站一節：網址 `https://tool-jx3.github.io/limbus-translate/`、一次性設定 Settings → Pages → Source 選「GitHub Actions」、`pages.yml` 列進 workflow 表）、`CLAUDE.md`（指令加 `cd site && npm ci && npm test && npm run build`）、主 spec §12（workflow 表）與 §16（repo 結構加 `site/`）

**Interfaces:**
- Consumes: Task 2 的 `lct site --out site/public/data --json`；Task 3 的 npm scripts 與 `site/.nvmrc`
- Produces: `pages.yml` 的步驟依序為：checkout `main`（`fetch-depth: 0`、`persist-credentials: false`）→ fetch `raw`、`tw` 並 `git worktree add .work/raw raw`、`.work/tw tw` → setup-go → `go build -o bin/lct ./cmd/lct` → `bin/lct site --out site/public/data --json` → setup-node（`node-version-file: site/.nvmrc`、`cache: npm`、`cache-dependency-path: site/package-lock.json`）→ `npm ci`、`npm run build`（`working-directory: site`）→ `actions/configure-pages@v6` → `actions/upload-pages-artifact@v5`（`path: site/dist`）→ `actions/deploy-pages@v5`；`ci.yml` 的 `site` job：setup-node 同上 → `npm ci`、`npm run typecheck`、`npm test`、`npm run build`

- [ ] **Step 1: 寫失敗的合約測試**
  - `checkPagesWorkflow`：觸發（`workflow_run` 的 `workflows: [update, rebuild]`、`types: [completed]`，以及 `workflow_dispatch`）、job `if` 等於 `rulesProposalIf`、權限恰好 `contents: read`、`pages: write`、`id-token: write`、concurrency、`environment: github-pages`、上述步驟順序（`bin/lct site` 在 `npm run build` 之前、`upload-pages-artifact` 在 `deploy-pages` 之前）、action 版本、不含 `secrets.`
  - mutation：拿掉 `id-token: write`、`cancel-in-progress: true`、`upload-pages-artifact` 的 `path` 改成 `site`、`bin/lct site` 移到 `npm run build` 之後、加上 `secrets.ANTHROPIC_API_KEY` → 各自被指出
  - `ci.yml` 有 `site` job，且跑 `npm test` 與 `npm run build`
  - README 的網站一節有網址與「GitHub Actions」設定（`checkRefsDocumented` 照常通過）

- [ ] **Step 2: 執行測試確認失敗**

Run: `go test ./cmd/lct/ -run 'Contract|Workflow'`
Expected: FAIL（沒有 `pages.yml`）

- [ ] **Step 3: 寫 `pages.yml`、`ci.yml` 的 site job 與文件**

- [ ] **Step 4: 執行測試確認通過**

Run: `go vet ./... && go test ./... && (cd site && npm ci && npm run typecheck && npm test && npm run build)`
Expected: PASS

- [ ] **Step 5: Commit 並 push**

```bash
git add .github/workflows cmd/lct/contract_test.go README.md CLAUDE.md docs/superpowers/specs/2026-10-31-limbus-zh-tw-translation-design.md
git commit -m "ci(pages): build and deploy the viewer after update and rebuild"
git push -u origin claude/festive-wright-g2k853
```

### Task 8（控制者）：實際資料驗收與上線

- [ ] **Step 1: 以實際資料建置並預覽**

Run: `go build -o bin/lct ./cmd/lct && bin/lct site --out site/public/data --json && cd site && npm run build && npm run preview`
Expected: 本機網站可開；`site/dist` 總大小記下來（預期 200 MB 以內）

- [ ] **Step 2: 以 Playwright 截圖檢查**（桌面 1280 寬與手機 390 寬）：首頁目錄；`StoryData/E914B.json` 劇本模式（ruby 顯示在字上方，例如「<ruby=ヨシヒデ>良秀</ruby>」）；`Skills.json` 表格模式與只看 `claude` 的篩選；深連結 `#/f/BgmLyrics/BgmLyrics_Cromer2.json?u=/dataList/45/content`；搜尋 TW「專用碎片」與 KR regex `요시히데`；頁尾授權文字與 noindex meta

- [ ] **Step 3: 回報使用者並取得同意後**：`main` 以 fast-forward 更新；請使用者在 Settings → Pages 把 Source 設為「GitHub Actions」；手動執行一次 `pages.yml`，確認部署成功、網址可開、`robots.txt` 存在
