# Limbus Company 台灣繁體中文翻譯系統 — 設計

日期：2026-10-31
相關文件：`docs/research/2026-10-30-reference-survey.md`（調查）、`guides/`、`glossary/`（規則）

## 1. 目的

Limbus Company 每週四更新官方「Custom Language Translation Support」語言包（KR／EN／JP 的 JSON）。本專案每次更新後，自動產出台灣繁體中文的自訂語言包，並保留每次原文與譯文的變化歷史。

- 自動化用 Claude Code（Agent SDK 的 headless 形態）搭配 Claude Console 的 API key，在 GitHub Actions 上跑
- 人可以在 Claude Code 裡用同一套 skill 介入：調整譯文、改進規則
- 翻譯品質靠「規則」累積：指引、詞彙表、欄位規則、人工調整紀錄。規則可以一直改進，改進的結果反映在之後的翻譯

## 2. 目標、非目標、反目標

**目標**

- 每次遊戲更新後，同一天產出可安裝的語言包（GitHub Release）
- 原文只要沒變就不重翻；KR 原文變了就依變化重翻
- 自動翻譯與人工介入走同一套規則、同一個 CLI、同一份 skill
- 每個譯文欄位都能追溯：來自哪裡、根據哪一版原文、用了哪一版規則
- 兩個階段：
  - **第一階段（一次性，不自動化）**：在 Claude Code 工作階段中完成。以 `user-unknown-0711/ParatranzUploader` 的譯文為主；未翻的部分依規則翻譯，以 LLC 簡中、LTM 繁中為參考；過程中可以改進規則。完成後把 `raw`、`tw` 的初始狀態上傳到 repo，第一階段即結束（§8.3）
  - **第二階段（GitHub Actions）**：之後所有自動化都是第二階段。不再使用上游譯文，全部依規則自己翻；規則可以改進；可以人工調整舊譯文

**非目標**

- 不使用 Paratranz API，也不回寫 Paratranz；GitHub Actions 不讀任何上游譯文
- 不做簡轉繁（OpenCC）；LLC 簡中只作為給 Claude 的參考
- 不做網頁介面
- 不做其他語言
- 不做「僅戰鬥版」（之後要加，只是打包時少放幾個檔）

**反目標**（技術上能跑但算失敗的情況）

- 原文沒變的行被重翻，或同一句每週換說法
- CI 自己改了規則（CI 只能提出建議）
- 發出驗證失敗的檔案（標籤壞掉會讓遊戲顯示錯亂）
- 人工調整在 KR 原文改變後仍然默默蓋過新譯文
- 把 LTM 等不同翻譯脈絡的名稱混進產出（例如「默爾索」）
- skill 裡寫死模型名稱，或 skill 與 CI 的流程分岔

## 3. 限制

- 語言：Go（最新穩定版）
- Claude：Claude Code CLI 以 headless 模式（`claude -p`）執行；認證用 GitHub secret `ANTHROPIC_API_KEY`（Claude Console 的 key）。官方 Agent SDK 只有 Python／TypeScript，官方文件建議其他語言以子行程執行 CLI，這正是本設計的做法
- 狀態 API 網址放 GitHub secret（第三方服務，可能更換）
- 授權：上游譯文是 CC BY-NC-SA 4.0；只要產出含有匯入的上游譯文，語言包就必須沿用 CC BY-NC-SA 4.0、標示來源、不可商用
- GitHub Actions 單一 job 上限 6 小時
- 原文快照每次約 90 MB（三語解壓後），git 以差異壓縮，週更變動小

## 4. 做法（Approaches）

| 做法 | 內容 | 取捨 |
|---|---|---|
| **A（採用）** | Go CLI `lct` 做所有確定性的工作；翻譯流程寫成 Claude Code skill；CI 用 `claude -p` 跑同一份 skill | skill 與自動化天生同步；每輪有 Claude Code 的工具開銷，比直接打 API 貴 |
| B | Go 用 `anthropic-sdk-go` 直接打 Messages API（Batch 半價、prompt caching），skill 只給人用 | 最省、最可控；但有兩個執行器，靠測試防止分岔 |
| C | 另外用 TS／Python Agent SDK 包一層 | 違反 Go 偏好、多一個 runtime，比 `claude -p` 多得到的很少 |

使用者選 A。「一批」的定義放在 `lct batch next／submit` 這個接縫上；日後若某週量太大想省錢，可以在同一個接縫後面加上 B 的 Batch API 路徑，prompt 與驗證不必改。

## 5. 整體架構

```text
                 ┌──────────────── main ────────────────┐
                 │ cmd/lct (Go CLI)   .claude/skills/   │
                 │ guides/ glossary/ rules/  lct.yaml   │
                 └──────────────────────────────────────┘
 狀態 API ─┐                 │ 讀規則
 CDN zip ──┴─ lct fetch ─▶ raw（orphan）：每次遊戲更新一個 commit
                              │   （第一階段：lct bootstrap 匯入上游，建立初始 tw）
                              ▼
                    lct plan：逐欄位決定 沿用／匯入／翻譯／重翻
                              │ 待譯清單
                              ▼
               claude -p "/translate-pending"（同一份 skill）
                 └─ 反覆 lct batch next → 翻譯 → lct batch submit
                              │
                              ▼
                    lct build：合併、驗證、打包
                              │
                              ▼
                tw（orphan）：譯文成品＋逐欄位紀錄；GitHub Release
```

## 6. 資料與識別

### 6.1 原文快照（`raw` 分支）

- orphan 分支，版面沿用 `tool-jx3/Assets`：`kr/`、`en/`、`jp/` 下放解壓後原樣的檔案（保留 `KR_` 等前綴、BOM、CRLF），外加 `meta/source.json`（`token`、`created_at`、取得時間、各檔 sha256）
- `.gitattributes` 設 `* -text`，git 不改換行
- 每次遊戲更新一個 commit，打 tag `raw-YYYYMMDD`（同日多次加 `.N`）
- token 與 `meta/source.json` 相同就不建新 commit
- 這個版面讓 `tool-jx3/ParatranzUploader` 之後可以直接改讀這個分支（是否退役 `tool-jx3/Assets` 由使用者決定，見 §20）

### 6.2 翻譯單位與 key

- **檔名**：去掉語言前綴的相對路徑，例如 `StoryData/S1037B.json`、`Skills_personality-08.json`
- **單位**：一個字串欄位。key = `<檔名>|<JSON pointer>`，例如 `StoryData/S1037B.json|/dataList/12/content`
- 不可把檔案壓成 id→條目 的表：有重複 id、無 id 的條目、沒有 `dataList` 的檔
- **骨架**：以 KR 檔的結構為準輸出。檔案集合預設為「KR 與 EN 都有的檔」（2,366 個）；只有 KR 有的 53 個、只有 EN 有的 10 個先不輸出，列在報表（見 §20）

### 6.3 哪些欄位要翻

規則在 `rules/fields.yaml`（自動產生）與 `rules/fields.manual.yaml`（人工）。三層：

1. **欄位層**：整個欄位在三語中永遠相同 → 識別碼，不翻，輸出 KR 值（`key`、`usage`、`imgStr`、`colorCode`、`timeline`…共 15 個）
2. **檔案群×欄位層**：例如 `BattleSpeechBubbleDlg*` 的 `desc`、語音檔的 `desc` 是觸發標籤，不翻
3. **單值層**：這一格三語相同（例如 `......`）→ 原樣輸出

人工規則強制 `id`、`model`、`codeName` 不翻（三語偶有差異，但是識別碼），以及 Hakimi 歸納的特例（`//`、`SE //` 開頭的字串；`Items` 中註明 `name만 번역` 的 desc）。

`lct fields learn` 每次建置重算第 1、2 層，結果有變就列入報表（例如遊戲新增欄位），由人決定是否更新規則檔。

### 6.4 逐欄位紀錄（`tw` 分支的 `.meta/units/`）

每個輸出檔對應一個 `.meta/units/<檔名>.jsonl`，每行一個單位：

```json
{"key": "StoryData/S1037B.json|/dataList/12/content",
 "kr_sha": "…", "zh": "…",
 "origin": "paratranz | claude | human | copy | fallback",
 "since": "raw-20261029", "rules": "<main commit>", "model": "…", "batch": "…",
 "flags": ["needs_review"]}
```

- `kr_sha`：KR 原文（去 BOM、CRLF→LF）的 sha256
- `origin`：`copy` = 識別碼或三語相同；`fallback` = 翻譯失敗暫用 EN 原文
- 上一版 `tw` 的紀錄就是這一版的「既有譯文」

## 7. 規則

規則都在 `main`，由人審核後才變更。CI 只讀，不寫。

| 規則 | 位置 | 性質 |
|---|---|---|
| 共通指引 | `guides/base.md` | 硬性格式規則＋共通風格 |
| 族群指引 | `guides/families.md`，之後可拆成 `guides/families/<族群>.md` | 依檔名套用 |
| 角色指引 | `guides/characters/<id>.md`（YAML 開頭含 `models`、`file_names`、`gender`…） | 依說話者套用 |
| 人格指引 | `guides/identities/<人格編號>.md`（需要時才建） | 依檔名中的人格編號套用 |
| 詞彙表 | `glossary/terms.yaml`（`status: approved／candidate`） | 硬性；只注入命中的條目 |
| 欄位規則 | `rules/fields*.yaml` | 硬性 |
| 人工調整 | `rules/adjustments/*.yaml` | 見 §11 |
| 路由 | `guides/routing.yaml`（檔名樣式 → 族群指引；人格編號規則） | 程式用 |

**疊加順序**：base → 族群 → 角色 → 人格 → 單檔備註。詞彙表、欄位規則與 `base.md` 的格式規則是硬性規定，風格指引不能推翻。

`lct guide resolve <批次>` 決定一批要帶哪些指引檔與詞彙條目；CI 與人工都經過這個指令，所以看到的內容一定相同。

**名稱表**（自動產生，具約束力）：

- 關鍵字表：`[ID]` → 本專案目前 `BattleKeywords*`／`Bufs*` 譯文中的名稱
- 說話者名稱表：`model` → 本專案目前 `ScenarioModelCodes` 譯文中的名稱
- 這兩張表反映語言包自己的現行譯名，所以不需要逐條審核就能約束翻譯；要改名走 `term-decision`。第一階段時它們來自 Paratranz 的譯名
- Claude 在 `note` 提出的新詞只是候選，不具約束力，審核通過才寫進 `glossary/terms.yaml`
- LTM／LLC 的名稱只作參考。從兩者的 `ScenarioModelCodes` 與本專案不同的名稱（例如「默爾索」）產生「他源名稱」清單；譯文出現時列為警告

## 8. 每次建置的決策邏輯

### 8.1 新舊原文對齊

`lct plan` 比對上一版 `raw` 與這一版 `raw`，把每個新單位對到舊單位：

1. 條目有檔內唯一的 `id`：以（`id`, 條目內欄位路徑）對齊
2. 其他（`StoryData`、重複 id、無 id）：把檔內單位依序列成（欄位名, KR 文字）序列，做序列差異比對（LCS／Myers）。對上的沿用；插入的是新單位；替換的配對成「舊→新」

結果：每個單位標為 **不變**（KR 相同）、**變更**（對到舊單位但 KR 不同）、**新增**、**刪除**。插行造成的位置平移不算變更。

**相同文字重用**：新增單位的 KR 若與同族群、同欄位、（對話則）同說話者的既有單位完全相同，直接沿用其譯文。短字串只限同檔重用（門檻由實作者定）。

### 8.2 每個單位的譯文來源

依序取第一個成立的：

| 順序 | 條件 | 結果 |
|---|---|---|
| 1 | 欄位規則判定不翻 | `copy` |
| 2 | 檔案在凍結清單（`lct.yaml` 的 `frozen`，目前為 `BgmLyrics/**`）且已有譯文 | 沿用既有譯文，不被取代；KR 變了也不動，只列入報表 |
| 3 | 有人工調整，且其 `kr_sha` = 目前 KR | 用人工調整，`human` |
| 4 | 既有紀錄的 `kr_sha` = 目前 KR | 沿用 |
| 5 | 相同文字重用成立 | 沿用 |
| 6 | 其他（新增、KR 變更、人工調整已過期） | 進待譯清單 |

- 這是第二階段（GitHub Actions）的邏輯。第一階段匯入的 `paratranz` 紀錄照第 4 條沿用，直到 KR 變更或人工介入
- 人工調整的 `kr_sha` 與目前 KR 不同 → 調整「過期」，該單位重翻；舊調整當作參考給 Claude，報表列出

### 8.3 第一階段（一次性，在 Claude Code 工作階段中完成）

步驟：

1. 建好 `lct` 的核心功能與 skill
2. 以 2026-10-29 的官方語言包（token `l20261029_n2rX1Jku6_9sBiu9Np24`）建立 `raw` 的第一個 commit
3. `lct bootstrap` 匯入固定版本的上游：ParatranzUploader release `1793285016` 的 `localize_files.zip`、LTM release `1792667238` 的 `Hant.zip`、LLC release `2026103002`（2026-10-30）的 `LimbusLocalize_2026103002.zip`。依下面的條件決定每個單位是直接採用 Paratranz，還是進待譯清單
4. 在這個工作階段中，以 `translate-pending` skill 翻完待譯清單。走的是與 CI 相同的 `lct batch next／submit` 與驗證；可以多個子代理並行。這一步的花費算在 Claude Code 工作階段，不用 Console API key
5. 過程中發現的規則問題，直接改進 `guides/`、`glossary/`
6. `lct build` 產生第一版 `tw`，與 `raw` 一起 push 到 repo；第一階段結束
7. 手動觸發一次 `rebuild.yml`，由 CI 打包並發布第一個 Release（這個工作階段無法呼叫建立 Release 的 API）

第一階段結束後，上游譯文不再被讀取；`lct bootstrap` 只保留給重建初始狀態用。

「有效譯文」的條件：

- 不等於 EN 原文（EN ≠ KR 時）——ParatranzUploader 用英文填未翻條目，約 8.6%
- 不等於 KR 原文、不含諺文、含漢字（純數字或符號的值走 `copy`）
- **檔案骨架相符**：ParatranzUploader 以當時的 KR 檔為骨架輸出，識別欄位（`id`、`model` 等）原樣保留。匯入前先比對 Paratranz 檔與 10/29 KR 檔的識別欄位序列與結構：
  - 完全相同 → 依 JSON pointer 直接對應
  - 不同 → 以識別欄位序列做序列比對，只匯入對得上的單位；對不上的進待譯清單

匯入前的正規化：

- 把「官方中文關鍵字名＋空格」還原為 `[ID]`（依 `guides/base.md`，約 1,079 處可機械還原）
- 依 `base.md` 的規則修正 `[ID]` 前後空格

正規化後跑 §10 的驗證：

- 通過 → `paratranz`
- 不通過（例如少了標籤，約 205 行）→ 進待譯清單，標為「修正」，把 Paratranz 譯文當作草稿交給 Claude，要求最小幅度修正

### 8.4 待譯清單

存在 `tw` 分支的 `.meta/pending.jsonl`，跨次建置保留。每筆記錄：key、原因（新增／KR 變更／調整過期／Paratranz 修正（只在第一階段）／人工要求重翻）、上一版 KR 與譯文、人工調整內容（若有）。

## 9. 翻譯流程（Claude）

### 9.1 批次

- `lct batch next [--family F] [--max-units N]` 產生 `work/batches/<id>.json`：
  - 要讀的指引檔（依疊加順序）、命中的詞彙條目、關鍵字表（`[ID]` → 本專案的中文名）
  - 每個單位：key、說話者（角色 id）、KR／EN／JP、參考譯文（只在第一階段：LLC 簡中、LTM 繁中、未通過驗證的 Paratranz 草稿；明確標示為參考，不是答案）、上一版 KR 與譯文、人工調整內容、需保留的 `[ID]`／佔位符／標籤
  - 劇情：一個檔（一個場景）一批；未變的前後行附上既有譯文作為上下文
  - 其他：同檔分批，每批上限 N 個單位
  - 批次依 `guides/families.md` 的翻譯優先序發放
- Claude 寫 `work/batches/<id>.out.json`（`[{key, zh, note?}]`），再執行 `lct batch submit <id>`
- `submit` 逐單位驗證，回傳 JSON：`{accepted: [...], rejected: [{key, errors}]}`。Claude 修正後重送；同一單位失敗 3 次（可設定）就標 `needs_review`，暫用 EN 原文（`fallback`），留在待譯清單
- `note` 用來回報疑問或新詞，進報表，不自動變成規則

### 9.2 CI 執行

- 安裝固定版本的 Claude Code CLI，build `lct`
- 迴圈執行：`claude -p "/translate-pending" --output-format json --model "$LCT_MODEL" --max-turns <N>`
  - 直到 `lct pending --count` 為 0、累計 `total_cost_usd` 達到 `LCT_MAX_USD`，或時間快到上限
  - 沒翻完的留在待譯清單，下次建置繼續；已翻完的照常打包，未翻部分用 `fallback`
- `.claude/settings.json` 限制工具：只允許執行 `lct`、讀取 repo、寫入 `work/`
- 模型由 variable `LCT_MODEL` 指定（預設 `claude-opus-5-5`），skill 不寫死模型
- 每次的花費、模型、批次數寫入 `.meta/build.json` 與 job summary

### 9.3 Skills（`.claude/skills/`）

| skill | 用途 | 誰用 |
|---|---|---|
| `translate-pending` | 依 §9.1 迴圈翻完待譯清單 | CI；第一階段在本工作階段中使用；人在本機執行時只產生預覽 |
| `weekly-update` | 在本機依序跑 fetch → plan → translate-pending → build，等同 CI 流程，結果只寫到本機的 `out/` 供進遊戲測試 | 人 |
| `adjust` | 人工調整譯文，並把調整整理進規則（§11） | 人 |
| `rules-review` | 整理報表中的建議與尚未歸納的調整，提出指引／詞彙修改 | 人 |
| `term-decision` | 決定詞彙表候選條目 | 人 |

SKILL.md 只寫流程；規則內容都引用 `guides/`、`glossary/` 的檔案。所有狀態變更都經由 `lct`，不手改 JSON／YAML。

**寫入權**：第一階段的初始 commit 之後，`raw`、`tw` 分支與 Release 只由 CI 寫入。人的所有介入都經由 `main`：人工調整寫 `rules/adjustments/`，重翻要求寫 `rules/requeue/`（CI 處理後在 `tw` 的紀錄中標記為已處理）。也可以在手動觸發 workflow 時帶參數。這樣不需要處理多人同時寫 `tw` 的衝突。

## 10. 驗證（`lct validate`，CI 與 skill 共用）

**硬性（不通過就退回）**

結構類檢查（JSON、結構、`[ID]`、佔位符、標籤、但丁的 `<…>`、非空）對所有單位一律強制，不通過的既有單位改用 `fallback` 並進待譯清單。內容類檢查（禁用譯法、殘留諺文／簡體）對新翻的單位強制；規則改變後才不符合的既有單位只列為警告，等人決定是否重翻（不違反「規則改了不自動重翻」）。

- JSON 有效；結構（鍵、型別、陣列長度）與 KR 骨架相同；識別欄位等於 KR
- `[ID]` 的種類與數量與 KR 相同（全大寫替換變數同樣要保留）
- 佔位符（`{0}`、`<0>`、`{Slot}`…）的種類與數量與 KR 相同
- TextMeshPro 標籤種類、數量相同，正確巢狀，屬性值未改
- 不含諺文、不含簡體專用字
- KR 整句包在 `<…>`（但丁台詞）時，譯文也必須包住
- 不含詞彙表的禁用譯法（中文以子字串比對，`re:` 開頭為正規表示式）
- KR 有內容時譯文不可為空

**自動修正（在驗證前）**

- `[ID]` 前後空格依 `base.md` 第 6 條與補充規則
- `&lt;`、`&gt;` 還原

**警告（進報表）**

- 中文旁的半形 `,`、`.`、`:`，`...`，`“”`
- 含假名
- 譯文長度與 KR 比例異常
- 同一 KR 文字在同族群出現不同譯法
- 輸出字元超出字型涵蓋範圍（建置時以字型的字元表檢查）

## 11. 人工介入與規則改進

兩個階段都適用。

### 11.1 人工調整（`adjust` skill）

1. 人描述要改的內容（某一句、某角色的某種說法、某個詞）
2. `lct search` 找出受影響的單位（依 key、文字、說話者、檔名樣式）
3. Claude 擬新譯文；`lct adjust add` 驗證後寫入 `rules/adjustments/<日期>-<主題>.yaml`：

   ```yaml
   - key: StoryData/E002X.json|/dataList/14/content
     kr_sha: …
     zh: 啊……但丁閣下，時鐘的時針好像在不停顫抖呢？您還好嗎？
     reason: 鴻璐對但丁說話用敬語，第二人稱用「您」
     rule: guides/characters/honglu.md   # 歸納到哪條規則；一次性的錯字修正寫 one-off
   ```

4. **把調整歸納進規則**：Claude 提出對應的指引或詞彙修改，人同意後寫入
5. 依新規則找出其他違反的 Claude 譯文，人同意後 `lct requeue` 寫入 `rules/requeue/<日期>-<主題>.yaml`（範圍與理由）
6. commit 到 `main` → 觸發重建（§12），CI 依調整與重翻要求更新 `tw`

人工調整只在它所根據的 KR 版本有效。KR 變了就依目前規則重翻；調整背後的道理已經寫進規則，會反映在重翻結果。

### 11.2 規則改進（`rules-review` skill）

- 來源：週報中的建議（不一致譯法、新說話者、新欄位、新檔案族群、Claude 的 `note`）、尚未歸納進規則的人工調整
- Claude 提出具體的檔案修改；人同意後 commit 到 `main`
- 規則改了不會自動重翻舊譯文。要套用到舊譯文，由人以 `lct requeue` 指定範圍（這算人工介入）

## 12. GitHub Actions

| workflow | 觸發 | 內容 |
|---|---|---|
| `update.yml` | 週四更新時段每 20 分鐘（約 UTC 01:00–05:00）；每日一次；`repository_dispatch`（`limbus_update`）；手動（可選 `force`、`requeue`、`max_usd`） | fetch → 有新 token 才 snapshot → plan → 翻譯迴圈 → build → commit `tw` → Release。每日那次另外：待譯清單不是空的就繼續翻。沒有新 token、也沒有待譯就直接結束 |
| `rebuild.yml` | push 到 `main` 且動到 `rules/**`、`glossary/**`、`guides/**`、`lct.yaml`；手動 | 不 fetch；plan → 有待譯才翻 → build → commit `tw` → Release |
| `ci.yml` | PR 與 push | `go test`；`lct rules lint`（指引開頭 YAML、詞彙表、調整檔的 schema）；skill 契約測試；改到規則的 PR 另外以目前 `tw` 跑 `lct validate`，列出這次規則變更會讓多少既有單位不符合 |

- 所有寫入 `raw`／`tw` 的 workflow 共用一個 `concurrency` 群組，依序執行
- secrets：`ANTHROPIC_API_KEY`、`LIMBUS_STATUS_URL`
- variables：`LIMBUS_CDN_TEMPLATE`（預設 `https://downloadcommon.limbuscompanycdn.org/{token}/Assets/LocalizePatch/localize_{lang}.zip`）、`LCT_MODEL`、`LCT_MAX_USD`
- `GITHUB_TOKEN` 需要 `contents: write`

## 13. 打包與發佈

```text
LimbusCompany_Data/
  Lang/
    config.json          # {"lang": "zh-TW"}
    zh-TW/
      *.json、StoryData/ …（UTF-8 無 BOM，輸出要可重現）
      Font/Context/*.ttf、Font/Title/*.ttf
      Info/              # 授權（CC BY-NC-SA 4.0）、來源標示、version.json（建置輸入）
```

- 兩個 zip：只有文字、含字型（同 ParatranzUploader 的做法）
- 字型：Sarasa Gothic TC（SIL OFL），固定版本並驗證 checksum
- Release tag `zh-TW-YYYYMMDD.N`；release notes 由報表摘要產生
- `version.json` 與 `.meta/build.json` 記錄：raw tag 與 token、`main` commit、模型、花費；第一階段的初始建置另記上游 release tag

## 14. 報表

每次建置寫 `tw` 分支的 `.meta/report.md`，並放進 job summary：

- 各來源的單位數（`paratranz`／`claude`／`human`／`copy`／`fallback`）
- 這次新增、因 KR 變更重翻、人工調整過期、仍待譯、`needs_review` 的單位
- 驗證警告
- 新欄位、新檔案族群、未對應的說話者（`model` 不在任何角色的清單中）
- 同一 KR 不同譯法的清單
- Claude 的 `note`
- 花費與模型

## 15. `lct` 指令一覽

| 指令 | 用途 |
|---|---|
| `lct fetch` | 讀狀態 API、比對 token、下載 zip |
| `lct bootstrap` | 第一階段：匯入固定版本的上游譯文，建立初始紀錄與待譯清單（§8.3） |
| `lct snapshot` | 寫入 `raw` 分支並打 tag |
| `lct plan` | §8：對齊、決定來源、更新待譯清單 |
| `lct pending [--count]` | 查待譯清單 |
| `lct batch next／submit` | §9.1 |
| `lct guide resolve` | 一批要帶的指引與詞彙 |
| `lct validate` | §10 |
| `lct build` | 合併、驗證、打包、寫報表 |
| `lct publish` | commit `tw`、建立 Release |
| `lct search` | 依 key、文字、說話者、檔名找單位 |
| `lct adjust add` | 新增人工調整 |
| `lct requeue` | 指定範圍送回待譯清單 |
| `lct fields learn` | 重算欄位規則 |
| `lct glossary derive` | 從 `BattleKeywords*`、`Bufs*`、`ScenarioModelCodes` 產生候選詞條 |
| `lct rules lint` | 規則檔 schema 檢查 |

所有指令支援 `--json` 輸出；exit code：0 成功、1 有發現或使用者錯誤、2 系統錯誤。`raw`／`tw` 分支以 `git worktree` 操作。

## 16. Repo 結構（`main`）

```text
cmd/lct/                 CLI 入口
internal/…               fetch、snapshot、align、units、rules、guide、validate、pack、report…
guides/  glossary/  rules/
lct.yaml                 輸出資料夾名、字型、凍結清單、批次大小、重試次數、第一階段用的上游版本
.claude/skills/          §9.3
.claude/settings.json    工具權限
CLAUDE.md                專案說明（AGENTS.md 指向它）
.github/workflows/       §12
docs/
```

## 17. 測試

- Go 單元測試：對齊（含插行）、來源決策、Paratranz 有效判定與 `[ID]` 還原、驗證規則
- 真實資料 fixture：從 10/29 語言包挑小量檔案
- 重播測試：用 `tool-jx3/Assets` 的歷次 commit 當作連續幾週的原文，驗證「不變不重翻、變了才重翻」
- 不呼叫 Claude 的乾跑模式（待譯全部走 `fallback`），讓 CI 能完整跑過流程
- skill 契約測試：skill 引用的檔案與 `lct` 指令都存在；不含模型名稱

## 18. 已決定的事項

**使用者決定**

- Go；Claude Code（Agent SDK）＋ Console API key；skill 與自動化同步（做法 A）
- GitHub Actions；原文與譯文各一個 orphan 分支；狀態 API 網址放 secret
- 兩個階段的範圍（§2）；兩階段都能改進規則
- 第一階段不自動化，在 Claude Code 工作階段中一次完成並上傳；之後的 GitHub Actions 都是第二階段
- KR 為主、JP／EN 輔助
- 原文沒變不重翻；KR 變了依變化重翻
- 人工調整不是最高優先的永久覆寫，而是歸納進規則
- 依檔案族群與角色建立指引
- 一律保留 `[英文ID]`；您／你依 KR；你／妳依對象性別（但丁用你／他）；唐吉訶德稱「經理老爺」「浮士德小姐」；全大寫替換變數不加空格；翻譯優先序照列出順序

**設計者的判斷（可推翻）**

- 輸出資料夾名 `zh-TW`；字型 Sarasa Gothic TC
- 人工調整存在 `main` 的 `rules/adjustments/`，與其他規則一起審核
- 翻譯失敗暫用 EN 原文
- `raw` 版面沿用 `tool-jx3/Assets`
- 每次建置都發 Release（文字版＋含字型版）
- 只輸出 KR 與 EN 都有的檔
- `raw`、`tw` 與 Release 只由 CI 寫入；人工介入一律經由 `main`（§9.3）
- Paratranz 匯入先比對識別欄位骨架，對不上的單位不匯入（§8.3）
- 名稱表（關鍵字、說話者）取自本專案現行譯文，直接具約束力；Claude 提出的新詞要審核（§7）
- 結構類驗證一律強制；內容類驗證對規則改變後的既有單位只警告（§10）
- 凍結（`BgmLyrics/**`）＝既有譯文不被取代，KR 變了也不動、只報告；新加入的歌詞照一般流程翻譯
- 每日一次繼續翻待譯清單（§12）
- 第二階段預設模型 `claude-opus-5-5`、每次建置花費上限 `LCT_MAX_USD` 預設 30 美元，都可隨時改 variable

## 19. 交給實作者決定

- Go 套件拆分、YAML／JSON 函式庫、序列差異演算法
- 相同文字重用的長度門檻
- 批次大小、重試次數、`--max-turns` 的預設值
- 簡體專用字的判定方式
- 字型取得方式（下載固定版本或放在 repo）
- release notes 的格式

## 20. 待確認

- 遊戲在自訂語言下實際讀哪些檔：只有 KR 有的 53 個檔、只有 EN 有的 10 個檔要不要輸出（Steam 公告在本環境無法讀取）
- `StoryData` 中 `PC*`、`ES*` 前綴的正式名稱
- `tool-jx3/Assets` 與 `tool-jx3/ParatranzUploader` 之後是否改用本專案的 `raw` 分支或退役
- `raw` 分支要從 10/29 開始，還是先匯入 `tool-jx3/Assets` 的歷次 commit 當作過去的更新歷史
- 程式碼的授權（譯文內容已確定為 CC BY-NC-SA 4.0）
