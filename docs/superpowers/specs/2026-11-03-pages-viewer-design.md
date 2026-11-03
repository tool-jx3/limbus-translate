# Limbus 譯文對照網站（GitHub Pages）— 設計

> 狀態：使用者已確認方向（2026-11-03）：Preact＋Vite、全部檔案都放、加 noindex、顯示譯文來源標記與篩選。
> 主設計：`docs/superpowers/specs/2026-10-31-limbus-zh-tw-translation-design.md`（以下稱「主 spec」）。

## 1. 目的

給譯者與校對者一個不用 clone repo 就能用的網頁：找到某個檔案或某句話，同時看 KR／JP／EN／TW 四種語言的對應內容，並能從頭到尾連貫地讀完整個檔案（特別是劇情）。

## 2. 目標、非目標

**目標**

- 全部語言包檔案（約 2,420 個、27 萬個單位）都能瀏覽
- 四語對照：一個單位一列，KR／JP／EN 的對應沿用 `lct` 的對齊（`internal/align`），不以位置硬配；沒有對應的語言留空並標示
- 找檔案：依檔名、族群、劇情地點、說話者篩選
- 全文搜尋：任選一種語言，子字串或正規表示式；結果可跳到該檔案的那一列
- 連貫閱讀：劇情檔以「劇本模式」呈現（說話者、稱號、地點、台詞）；上一檔、下一檔依族群與檔名順序；鍵盤操作
- 標籤呈現：TextMeshPro 標籤（`<color>`、`<size>`、`<b>`、`<i>`、`<ruby>` 等）轉成畫面效果，ruby 顯示在字的上方；可切換成顯示原始標籤
- 譯文來源：每個 TW 譯文標示來源（`paratranz`、`claude`、`human`、`copy`、`fallback`），可依來源篩選
- 深連結：網址指向某個檔案的某個單位，可直接分享
- 不被搜尋引擎收錄（noindex）
- 手機可讀

**非目標**

- 在網頁上編輯譯文、登入、留言（修改仍經由 `main` 的規則檔或人工調整，主 spec §11）
- 保留多個版本：只放最新一次建置
- 後端、資料庫
- 字型檔（Sarasa 很大；用系統字型）

**之後再考慮**：詞彙表詞條高亮與說明、從某一列開 issue 回報譯文、與上一個 Release 的差異比對。

## 3. 限制

- GitHub Pages 是靜態網站：沒有伺服器端路由（用 hash 路由）；單一網站 1 GB 以內、單檔 100 MB 以內
- 網站資料只由 CI 產生並部署，**不 commit 到任何分支**（`raw`、`tw` 只由 pipeline 寫，主 spec §3）
- 網站名稱與頁面不得使用「Claude Code」「Anthropic」或其標誌（主 spec §3）；可以寫「使用 Claude Code 翻譯」
- 授權標示（每頁頁尾）：KR／JP／EN 原文著作權屬 Project Moon，本站非商業、供翻譯校對使用；TW 譯文採用 CC BY-NC-SA 4.0，含 Paratranz 社群譯文（第一階段匯入）
- noindex：每頁 `<meta name="robots" content="noindex, nofollow">`，並提供 `robots.txt`（`Disallow: /`）
- 不讀、不寫、不依賴 tool-jx3/Assets 與 tool-jx3/ParatranzUploader

## 4. 架構

```
raw 分支（KR/JP/EN 原文）┐
tw 分支（建置後的 zh-TW 包、.meta/units 紀錄）┼─ lct site --out site/public/data ─┐
guides/routing.yaml（族群與順序）┘                                          │
site/（Preact＋Vite 原始碼）──────────────── npm run build ──────── site/dist ─┴─ upload-pages-artifact → deploy-pages
```

- `lct site`（Go，新指令）：讀 raw 與 tw 兩個 worktree 和 repo 的 `guides/routing.yaml`，輸出網站要的 JSON（§5）
- `site/`（TypeScript、Preact、Vite）：靜態單頁應用，build 時把 `public/data/` 一起放進 `dist/`
- `.github/workflows/pages.yml`：`update`／`rebuild` 在 `main` 上成功後執行，也可以手動執行（§7）

## 5. 資料（`lct site` 的輸出）

所有 JSON 以 UTF-8、無 BOM、無縮排輸出；同樣的輸入產生位元組完全相同的輸出（排序固定）。`format` 是資料格式的版本，前端讀到不認得的版本就顯示「請重新整理」。

### 5.1 `data/meta.json`

```json
{"format": 1, "raw_tag": "raw-20261029", "rules_commit": "…", "tw_commit": "…", "release": "zh-TW-20261029.5",
 "langs": ["kr", "jp", "en", "zh"], "files": 2420, "units": 272783}
```

`release` 是 `tw_commit` 上的 `zh-TW-*` tag，沒有時為空字串。

### 5.2 `data/index.json`

```json
{"families": [{"id": "story-main", "files": [0, 1, …]}, …],
 "files": [{"path": "StoryData/S001A.json", "family": "story-main", "label": "…", "speakers": ["但丁", …],
            "units": 152, "origins": {"paratranz": 140, "claude": 10, "copy": 2}, "missing": {"en": 0, "jp": 3}}, …]}
```

- `families` 依 `guides/routing.yaml` 的 `priority` 排列；族群裡的檔案依檔名的自然順序（數字依數值比較：`S2` 在 `S10` 之前）
- `files` 的順序就是上一檔／下一檔的順序（族群順序，再依檔名），陣列索引就是檔案編號（§5.4 用）
- `label`：劇情檔（`StoryData/`）是第一個非空的 `place` 的 TW 譯文（沒有時用 KR），其他檔為空字串
- `speakers`：檔內 `teller` 的 TW 譯文，去重、依首次出現排序，最多 20 個
- `missing`：KR 有字串而該語言沒有對應的單位數（`FileAligner.Unmatched`）

### 5.3 `data/files/<檔案路徑>.json`

例：`data/files/StoryData/S001A.json`

```json
{"path": "StoryData/S001A.json", "family": "story-main",
 "units": [{"p": "/dataList/0/teller", "f": "teller", "i": 0, "kr": "…", "jp": "…", "en": "…", "zh": "…", "o": "paratranz"}, …]}
```

- `units` 依 KR 檔的順序（`units.Extract`），包含所有單位（`copy` 也包含，前端預設隱藏）
- `p` 指標、`f` 欄位名、`i` 所屬 `dataList` 元素的索引（不在 `dataList` 底下為 -1），前端用 `i` 把同一個元素的單位（說話者、稱號、地點、台詞）合成一段
- `jp`、`en`：對齊到的字串；沒有對應時省略該鍵
- `zh`：建置後語言包裡的字串，也就是玩家看到的（可能是暫用的英文）
- `o`：該單位紀錄的 `origin`（`paratranz`、`claude`、`human`、`copy`、`fallback`）；沒有紀錄，或語言包的字串與紀錄不同（建置時改用英文或 KR）時為 `stand-in`
- `fl`：紀錄的 `flags`（例如 `needs_review`），沒有時省略

### 5.4 `data/search/keys.json` 與 `data/search/<lang>.json`

全文搜尋用，`copy` 單位不收（多是 id、數字）。

- `keys.json`：`[[檔案編號, "指標"], …]`
- `<lang>.json`（`kr`、`jp`、`en`、`zh`）：字串陣列，長度與 `keys.json` 相同、位置一一對應；該語言沒有對應時為空字串

### 5.5 大小

目前的資料：每種語言純文字 13～14 MB、gzip 後約 4 MB。預估 `files/` 合計約 60 MB（gzip 約 16 MB）、`search/` 合計約 60 MB（gzip 約 17 MB），遠低於 Pages 上限。`lct site --json` 回報檔案數與總大小。

## 6. 前端

### 6.1 路由（hash）

- `#/`：首頁（目錄與搜尋）
- `#/f/<檔案路徑>?u=<指標>`：檔案檢視；`u` 有值時捲到該單位並高亮
- `#/s?q=<字串>&lang=<kr|jp|en|zh>&re=1`：全文搜尋結果

顯示哪些語言、是否顯示 copy、來源篩選、標籤顯示方式存在 `localStorage`（不放進網址，避免連結過長）。

### 6.2 版面

- 頁首：網站名稱、檔案篩選輸入框、全文搜尋框、語言開關（KR、JP、EN、TW，各自可關）
- 左側目錄：族群（可收合）→ 檔案；檔案篩選比對路徑、`label`、`speakers`；手機上收成抽屜
- 主區域：檔案檢視或搜尋結果
- 頁尾：授權標示（§3）、資料版本（`raw_tag`、`release`，連到 Release 頁）

### 6.3 檔案檢視

- **表格模式**（預設，非劇情檔）：一個單位一列，第一欄是欄位名（滑過顯示完整指標，點擊複製深連結），接著是開啟的語言欄；TW 欄附來源標記（顏色＋文字，例如 `claude`）
- **劇本模式**（`StoryData/`、`RPGSystem/*dialogue*` 的預設；其他檔可切換）：同一個 `dataList` 元素合成一段：`place` 改變時插入地點分隔線；說話者行顯示 `teller`（與 `title`）；台詞各語言並排（寬螢幕）或上下堆疊（窄螢幕）
- **來源篩選**：勾選要看的來源；未勾選的單位隱藏（劇本模式中，整段的台詞被篩掉時整段隱藏）；預設不顯示 `copy`
- **連貫閱讀**：上一檔／下一檔按鈕（依 `index.files` 的順序）；鍵盤 `[`、`]` 切換檔案，`j`、`k` 移到下一段／上一段
- **大檔**：超過 1,000 列時只渲染可見範圍附近的列（虛擬捲動），深連結仍能捲到正確位置
- 每個語言欄設定對應的 `lang` 屬性（`ko`、`ja`、`en`、`zh-Hant-TW`），讓瀏覽器選對字形

### 6.4 標籤呈現

- 解析 TextMeshPro 標籤成節點樹，再產生 DOM；**文字一律當文字插入，不使用 innerHTML**，任何字串都不能注入 HTML
- `<color=#rrggbb>` → 文字顏色；`<size=N>`、`<size=N%>` → 相對字級（限制在 0.75～1.5 倍）；`<b>`、`<i>`、`<u>`、`<s>` → 對應樣式；`<ruby=X>Y</ruby>` → `<ruby>Y<rt>X</rt></ruby>`；`<br>` 與字串中的換行 → 換行；`<sprite …>` → 以代號顯示（例如 `[sprite:Burn]`）；`<style="…">`、`<link>`、`<noparse>` 等其他標籤 → 只保留內文
- `[ID]`、`{0}` 等保留原樣，加上淡色背景以便辨識
- 開關「顯示原始標籤」：改為顯示原字串（等寬字型）

### 6.5 搜尋

- **找檔案**：首頁與左側目錄的篩選框，即時篩選 `index.files`
- **全文搜尋**：在 Web Worker 中執行。第一次搜尋某種語言時下載 `search/keys.json` 與該語言的 corpus，之後留在記憶體；比對前去掉標籤；子字串比對不分大小寫（拉丁字母）；正規表示式用 JavaScript `RegExp`（語法錯誤時提示）
- **新的搜尋取代舊的**：worker 開始比對後無法中途停止（例如 `(a+)+$` 會跑不完），所以舊搜尋若已開始比對就被取代時，頁面終止該 worker、另開一個（它會重新下載 corpus）再送新查詢；舊搜尋還在下載時不終止。比對超過 10 秒也由頁面終止 worker，並提示搜尋太久已停止、請改寫正規表示式
- 結果依檔案分組，每筆顯示命中語言的字串（命中處高亮）、同一單位的 KR 與 TW；最多顯示前 1,000 筆並告知總數；點擊跳到 `#/f/<路徑>?u=<指標>`

### 6.6 其他

- 深色模式依系統設定（`prefers-color-scheme`）
- 不使用外部 CDN；所有資源同源
- 不收集任何使用者資料（沒有分析工具）

## 7. 建置與部署

### 7.1 `pages.yml`

- 觸發：`workflow_run`（`update`、`rebuild`，`completed`），job 的 `if` 與 `rules-proposal.yml` 相同（成功、`main`、不是 pull request）；另可手動執行
- 權限：`contents: read`、`pages: write`、`id-token: write`；`concurrency: { group: pages, cancel-in-progress: false }`；`environment: github-pages`
- 步驟：checkout `main`（`persist-credentials: false`）→ 取 `raw`、`tw` 分支並建 worktree → setup-go → `go build -o bin/lct ./cmd/lct` → `bin/lct site --out site/public/data --json` → setup-node（版本與 `site/.nvmrc` 相同）→ `npm ci`、`npm run build`（於 `site/`）→ `actions/upload-pages-artifact`（`site/dist`）→ `actions/deploy-pages`
- action 版本與其他 workflow 一致（`actions/checkout@v5`、`actions/setup-go@v6`、`runs-on: ubuntu-24.04`），Pages 相關 action 用實作時的最新主版本，合約測試釘住

### 7.2 `ci.yml`

- 新增一個 job：`site/` 的 `npm ci`、`npm test`、`npm run build`（用一份小型的測試資料，不需要 raw／tw）
- Go 的 `lct site` 測試照常在 `go test ./...` 裡

### 7.3 一次性設定（人做）

Settings → Pages → Source 選「GitHub Actions」。README 的設定說明列出這一步與網站網址。

## 8. 測試

- **Go（`internal/site`、`cmd/lct`）**：以小型的 raw／tw 測試資料
  - 輸出結構與 §5 一致；同樣的輸入兩次輸出位元組相同
  - JP／EN 依對齊而非位置（測試資料讓 JP 的順序與 KR 不同）；沒有對應時省略鍵、`missing` 計數正確
  - `o` 的判斷：有紀錄且語言包字串相同時是紀錄的 `origin`；語言包字串不同時是 `stand-in`
  - 族群與檔名順序（自然排序）；`label`、`speakers`
  - 搜尋 corpus 不含 `copy` 單位、各語言長度與 `keys.json` 相同
  - `lct site --json` 的輸出欄位；`--out` 已存在時先清空再寫
- **前端（Vitest）**：標籤解析與渲染（含惡意字串不產生 HTML 元素）、路由解析、單位分組成段、檔案篩選、搜尋 worker 的比對（子字串、regex、去標籤）
- **workflow 合約測試**：`pages.yml` 的觸發、權限、concurrency、步驟順序、action 版本；README 列出網站設定
- **人工驗收**：本機以實際資料執行 `lct site` 與 `npm run preview`，用 Playwright 截圖檢查表格模式、劇本模式、搜尋、手機寬度

## 9. 已決定的事項

- 前端用 Preact＋Vite（TypeScript）
- 全部檔案都放
- 加 noindex（meta 與 robots.txt）
- 顯示譯文來源標記並可篩選
- 資料只放最新一次建置，不 commit 到任何分支
- 全文搜尋在瀏覽器端以 Web Worker 比對整份 corpus（不建倒排索引），支援正規表示式

## 10. 交給實作者決定

- 虛擬捲動用小型函式庫或自己寫（不超過約 150 行就自己寫）
- CSS 的組織方式（不引入大型 UI 框架）
- 網站的顯示名稱（遵守 §3 的命名限制）
