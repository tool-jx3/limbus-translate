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
  - **第一階段（一次性，不自動化）**：在 Claude Code 工作階段中完成。以 `user-unknown-0711/ParatranzUploader` 的譯文為主；未翻的部分依規則翻譯，以 LLC 簡中、LTM 繁中為參考；過程中可以改進規則。完成後把 `raw`、`tw` 的初始狀態上傳到 repo（存檔點除外），第一階段即結束（§8.3）
  - **第二階段（GitHub Actions）**：之後所有自動化都是第二階段。不再使用上游譯文，全部依規則自己翻；規則可以改進；可以人工調整舊譯文
  - **第三階段（將來）**：第二階段穩定後，把 CI 的大量翻譯改用做法 B（§4）

**非目標**

- 不使用 Paratranz API，也不回寫 Paratranz；GitHub Actions 不讀任何上游譯文
- 不做簡轉繁（OpenCC）；LLC 簡中只作為給 Claude 的參考
- 不做網頁介面
- 不做其他語言
- 不做「僅戰鬥版」（之後要加，只是打包時少放幾個檔）

**反目標**（技術上能跑但算失敗的情況）

- 原文沒變的行被重翻，或同一句每週換說法
- 規則未經人審核就改變（CI 只能開 PR 提出修改，合併由人決定）
- 發出驗證失敗的檔案（標籤壞掉會讓遊戲顯示錯亂）
- 人工調整在 KR 原文改變後仍然默默蓋過新譯文
- 把 LTM 等不同翻譯脈絡的名稱混進產出（例如「默爾索」）
- skill 裡寫死模型名稱，或 skill 與 CI 的流程分岔

## 3. 限制

- 語言：Go（最新穩定版）
- Claude：Claude Code CLI 以 headless 模式（`claude -p`）執行；認證用 GitHub secret `ANTHROPIC_API_KEY`（Claude Console 的 key）。官方 Agent SDK 只有 Python／TypeScript，官方文件建議其他語言以子行程執行 CLI，這正是本設計的做法
- 狀態 API 網址放 GitHub secret（第三方服務，可能更換）
- 授權：上游譯文是 CC BY-NC-SA 4.0；只要產出含有匯入的上游譯文，語言包就必須沿用 CC BY-NC-SA 4.0、標示來源、不可商用
- **不依賴 `tool-jx3/Assets`、`tool-jx3/ParatranzUploader`**：本專案不讀取、不寫入這兩個 repo，也不需要它們存在。第一階段下載上游 release 是一次性的，之後不再讀取
- GitHub Actions 單一 job 上限 6 小時
- 原文快照每次約 90 MB（三語解壓後），git 以差異壓縮，週更變動小

### 3.1 Claude 使用規範（依官方文件 code.claude.com/docs/en/legal-and-compliance，2026-10-31 查閱）

- **第二階段（CI）**：用 Claude Console 的 API key，適用 Commercial Terms。官方明文允許把自己的 API key 放在 secrets 給自己的工作流程使用，花費記在 key 的擁有者名下
- **第一階段（本工作階段）**：使用者本人在 Claude Code 中處理自己的專案，屬於一般使用。官方註明 Pro／Max／Ultra 方案的用量上限是以「一般、個人使用」為前提；1.87 萬句、多個子代理並行可能碰到用量上限，必要時分多次進行
- **不做的事**：不提供 Claude 帳號登入給他人、不轉售或代他人付費使用 Claude、不使用他人的帳號憑證
- **名稱**：專案名稱、Release 名稱不使用「Claude Code」「Anthropic」或其標誌；說明文件可以寫「使用 Claude Code 翻譯」
- **Anthropic Usage Policy**：翻譯遊戲文本（含暴力、髒話等虛構內容）沒有問題
- **著作權**：遊戲原文屬於 Project Moon。遊戲官方提供自訂語言功能，本專案非商業、譯文以 CC BY-NC-SA 4.0 發布；但 `raw` 分支會公開保存原文，是否符合 Project Moon 的條款由使用者自行確認（其他社群專案也公開保存原文，例如 LocalizeLimbusCompany）

## 4. 做法（Approaches）

| 做法 | 內容 | 取捨 |
|---|---|---|
| **A（採用）** | Go CLI `lct` 做所有確定性的工作；翻譯流程寫成 Claude Code skill；CI 用 `claude -p` 跑同一份 skill | skill 與自動化天生同步；每輪有 Claude Code 的工具開銷，比直接打 API 貴 |
| B | Go 用 `anthropic-sdk-go` 直接打 Messages API（Batch 半價、prompt caching），skill 只給人用 | 最省、最可控；但有兩個執行器，靠測試防止分岔 |
| C | 另外用 TS／Python Agent SDK 包一層 | 違反 Go 偏好、多一個 runtime，比 `claude -p` 多得到的很少 |

使用者選 A。做法 B 列為**第三階段**，等第二階段穩定後再開發。

為了讓第三階段不必重寫，現在就遵守：

- 「一批」的定義只在 `lct batch next／submit`。批次輸入與輸出的 JSON schema 放在 `schemas/`，是兩種做法共用的契約
- 翻譯指示（輸出格式、參考譯文怎麼用、遇到疑問怎麼回報）寫在 `prompts/translate.md`。skill 讀它；第三階段的 Go 程式以 `go:embed` 嵌入同一份。SKILL.md 只寫流程
- 驗證只有 `lct validate` 一套
- 紀錄裡的 `model`、`batch` 欄位能區分做法 A 與 B 的產出

第三階段的範圍：

- `lct translate --batch-api`：以 `anthropic-sdk-go` 送 Message Batches（半價），使用 prompt caching。重送迴圈（驗證失敗 → 只把失敗的單位再送一次）改由 Go 實作
- 人工介入（`adjust`、`rules-review` 等 skill）仍用 Claude Code（做法 A）

開始第三階段的條件（建議）：

- 第二階段連續數週沒有因流程問題需要人工修復
- 已累積數週的實際花費（`total_cost_usd`）與驗證通過率，可以和 B 的試跑結果比較

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
                              │
                              ▼
                審核 Issue（需要人處理的項目）；規則 PR（CI 提出的規則修改）
```

## 6. 資料與識別

### 6.1 原文快照（`raw` 分支）

- orphan 分支：`kr/`、`en/`、`jp/` 下放解壓後原樣的檔案（保留 `KR_` 等前綴、BOM、CRLF），外加 `meta/source.json`（`token`、`created_at`、取得時間、各檔 sha256）
- 從 2026-10-29 的官方語言包開始，不匯入 `tool-jx3/Assets` 的歷史
- `.gitattributes` 設 `* -text`，git 不改換行
- 每次遊戲更新一個 commit，打 tag `raw-YYYYMMDD`（同日多次加 `.N`）
- token 與 `meta/source.json` 相同就不建新 commit

### 6.2 翻譯單位與 key

- **檔名**：去掉語言前綴的相對路徑，例如 `StoryData/S1037B.json`、`Skills_personality-08.json`
- **單位**：一個字串欄位。key = `<檔名>|<JSON pointer>`，例如 `StoryData/S1037B.json|/dataList/12/content`
- 不可把檔案壓成 id→條目 的表：有重複 id、無 id 的條目、沒有 `dataList` 的檔
- **骨架**：以 KR 檔的結構為準輸出。**檔案集合以 KR 為準**：輸出全部 KR 檔（10/29 為 2,419 個）。其中只有 KR 有的檔（53 個）沒有 EN／JP 可參考，只依 KR 翻譯；只有 EN 有的檔（10 個）不輸出，列在報表

### 6.3 哪些欄位要翻

規則在 `rules/fields.yaml`（自動產生）與 `rules/fields.manual.yaml`（人工）。三層：

1. **欄位層**：整個欄位在三語中永遠相同 → 識別碼，不翻，輸出 KR 值（`key`、`usage`、`imgStr`、`colorCode`、`timeline`…共 15 個）
2. **檔案群×欄位層**：例如 `BattleSpeechBubbleDlg*` 的 `desc` 是觸發標籤，不翻。`PersonalityVoiceDlg` 的 `desc` 是語音標題（`인격 획득` → EN「Identity Acquisition」），要翻
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

### 6.5 跨語言對齊（KR 與 EN／JP、參考譯文）

輸出一律用 KR 的結構與順序，所以其他語言檔的順序不影響產出；但交給 Claude 的 EN／JP 參考、§6.3 的三語比對、第一階段的 LLC／LTM 參考，都必須先「對到同一句」。不能只靠 JSON 路徑（陣列位置）對齊。

2026-10-29 實測（KR、EN、JP 三語都有的 2,366 個檔）：

| 情況 | JP | EN | 例 |
|---|---|---|---|
| 同一批 id、順序不同 | 2 檔 | 0 | `EventVerificationUIText.json`（18 條中 14 條位置不同）、`Passives.json`（第 627～633 條） |
| 劇情 id 打錯（位置、說話者、內容都對得上，只有 id 不同） | 5 檔 | 0 | `StoryData/E001X.json` 第 201 行：KR `id=201`、JP `id=211` |
| 多一行 | 1 檔 | 1 檔 | `StoryData/P10511.json`：EN、JP 比 KR 多一行無 id 的內容，之後全部錯開 |
| 其他 id 集合不同 | — | 1 檔 | `EGOVoiceDig/Voice_EGO_Enemy_999.json`：EN 少一條 |

對齊規則，依檔逐一判斷：

1. **id 對齊**：`dataList` 條目在兩邊都有 id、且 id 在檔內唯一 → 依 id 對齊（處理換順序）
2. **序列對齊**：`StoryData`、沒有 id、id 重複，或 id 對齊後有大量對不上的 → 以每條的「指紋」做序列比對（LCS）。劇情的 id 只是行號，不可靠，所以指紋用 `model`、`place`／`teller`／`title` 等欄位是否存在，以及欄位名集合，不用 id（處理 id 打錯與插行）
3. **巢狀陣列**（例如技能的等級、硬幣）：在已對齊的條目內，有 id 就依 id，否則依位置
4. **驗證**：對齊後檢查識別欄位（`model` 等）是否一致；不一致或對不上的單位標為「未對齊」

使用方式：

- 交給 Claude 的批次中，每個單位的 EN／JP 都附上對齊方式（`pointer`／`id`／`sequence`）；「未對齊」的就不提供那個語言，寧可少給，也不給錯的句子
- §6.3 的三語比對只用已對齊的單位
- 新舊 KR 的比對（§8.1）、第一階段的 Paratranz／LLC／LTM 匯入（§8.3）用同一套對齊程式
- 報表列出有跨語言順序或 id 不一致的檔，以及「未對齊」的單位數

## 7. 規則

規則都在 `main`，由人審核後才變更。CI 不能直接寫入 `main`，只能開 PR 提出修改（§11.2）。

| 規則 | 位置 | 性質 |
|---|---|---|
| 共通指引 | `guides/base.md` | 硬性格式規則＋共通風格 |
| 族群指引 | `guides/families.md`，之後可拆成 `guides/families/<族群>.md` | 依檔名套用 |
| 角色指引 | `guides/characters/<id>.md`（YAML 開頭含 `models`、`file_names`、`gender`…） | 依說話者套用 |
| 人格指引 | `guides/identities/<人格編號>.md`（需要時才建） | 依檔名中的人格編號套用 |
| 詞彙表 | `glossary/terms.yaml`（`status: approved／candidate`） | 硬性；只注入命中的條目；條目可用 `files` 限定適用的檔（見下） |
| 欄位規則 | `rules/fields*.yaml` | 硬性 |
| 人工調整 | `rules/adjustments/*.yaml` | 見 §11 |
| 重翻要求 | `rules/requeue/*.yaml` | 見 §11.1；每個檔由 CI 套用一次 |
| 改寫檔 | `rules/rewrite/*.yaml` | 見 §11.1；以正規表示式機械改寫現有譯文，每個檔由 CI 套用一次 |
| 路由 | `guides/routing.yaml`（檔名樣式 → 族群指引；人格編號規則） | 程式用 |
| 外來名稱忽略清單 | `rules/foreign-names.ignore.yaml`（YAML 字串清單，可省略） | 「他源名稱」警告略過清單中的詞 |

**詞彙條目的適用範圍**：

- 條目可加 `files`：檔名樣式清單，寫法同 `lct.yaml` 的 `frozen`（`StoryData/S10[0-9][0-9]*`、`RPGSystem/**`）；省略 = 所有檔。只有批次的檔名符合時，條目才注入批次，也才檢查它的禁用譯法；`lct validate` 與 `lct adjust add` 同樣依單位所在的檔判斷（`lct validate` 以 `--file` 指定檔名）
- 同一個 `kr` 可以有多個條目，其中至多一個 `status: approved` 的條目不設 `files`（`rules lint` 檢查，`candidate` 不生效、不計；`files` 的樣式也在這裡檢查）。某個檔符合有 `files` 的條目時，由這些條目代表這個詞，沒設 `files` 的那一條不適用；沒有任何條目符合時，用沒設 `files` 的那一條
- 例：「신부」在第十章的檔（`*a1c10*`、`StoryData/S10[0-9][0-9]*`、`RPGSystem/**`、`Enemies-a1c10*`）譯「牧師」並禁用「神父」，其他檔譯「神父」；第十章的劇情檔是 S10xx（四位數），不含第一章的 S101B 等

**疊加順序**：base → 族群 → 角色 → 人格 → 單檔備註。詞彙表、欄位規則與 `base.md` 的格式規則是硬性規定，風格指引不能推翻。

`lct guide resolve <批次>` 決定一批要帶哪些指引檔與詞彙條目；CI 與人工都經過這個指令，所以看到的內容一定相同。

**名稱表**（自動產生，具約束力）：

- 關鍵字表：`[ID]` → 本專案目前 `BattleKeywords*`／`Bufs*` 譯文中的名稱
- 說話者名稱表：`model` → 本專案目前 `ScenarioModelCodes` 譯文中的名稱
- 這兩張表反映語言包自己的現行譯名，所以不需要逐條審核就能約束翻譯；要改名走 `term-decision`。第一階段時它們來自 Paratranz 的譯名
- Claude 在 `note` 提出的新詞只是候選，不具約束力，審核通過才寫進 `glossary/terms.yaml`
- LTM／LLC 的名稱只作參考。從兩者的 `ScenarioModelCodes` 與本專案不同的名稱（例如「默爾索」）產生「他源名稱」清單；譯文出現時列為警告
- 他源名稱裡有些只是常用詞（LTM 把某個說話者譯成「媽媽」），譯文用到並不是用錯名稱。這類詞列在 `rules/foreign-names.ignore.yaml`（YAML 字串清單，初始為「媽媽」「獨白」），譯文含有清單中的詞時不警告。比對的是整個他源名稱

## 8. 每次建置的決策邏輯

### 8.1 新舊原文對齊

`lct plan` 比對上一版 `raw` 與這一版 `raw`，把每個新單位對到舊單位：

1. 條目有檔內唯一的 `id`：以（`id`, 條目內欄位路徑）對齊
2. 其他（`StoryData`、重複 id、無 id）：把檔內單位依序列成（欄位名, KR 文字）序列，做序列差異比對（LCS／Myers）。對上的沿用；插入的是新單位；替換的配對成「舊→新」

這與跨語言對齊（§6.5）用同一套對齊程式，只是比對的內容不同：新舊 KR 之間比 KR 文字；跨語言之間文字不同，所以比識別欄位。

KR 在檔案中間插入資料很常見。`tool-jx3/Assets` 最近 12 次更新中，約 30 個檔有中間插入，例如：

- `StoryData/S918A.json`：在第 2、4、6 行之間各插入一行，後面 47 句不變的內容全部錯開
- `EGOgift_MirrorDungeon.json`：插入 1 條，720 句不變的內容換了位置
- `Bufs*`、`BattleKeywords*`：在既有條目裡新增欄位

處理方式：

- 劇情的 id 是行號，插行後會全部重新編號，所以劇情一律用序列對齊，不用 id
- 有唯一 id 的檔用 id 對齊；條目內新增的欄位是新單位
- 對齊後，不變的單位帶著原譯文移到新的 key（JSON pointer）；`.meta/units/` 的紀錄每次建置都以新 key 重寫

結果：每個單位標為 **不變**（KR 相同）、**變更**（對到舊單位但 KR 不同）、**新增**、**刪除**。插行造成的位置平移不算變更。

**相同文字重用**：新增單位的 KR 若與同族群、同欄位、（對話則）同說話者的既有單位完全相同，直接沿用其譯文。「既有單位」包含上一版同檔中這次被刪除的單位，所以 id 被重新編號的條目不會重翻。短字串只限同檔重用（門檻由實作者定）。

### 8.2 每個單位的譯文來源

依序取第一個成立的：

| 順序 | 條件 | 結果 |
|---|---|---|
| 1 | 欄位規則判定不翻 | `copy` |
| 2 | 檔案在凍結清單（`lct.yaml` 的 `frozen`，原為 `BgmLyrics/**`，2026-11-02 解除後為空）且已有譯文 | 沿用既有譯文，不被取代；KR 變了也不動，只列入報表 |
| 3 | 有人工調整對得上這個單位（查找方式見 §11.1），且其 `kr_sha` = 目前 KR | 用人工調整，`human` |
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
   - 第一階段的存檔點：
     - `raw` 在 snapshot 後即 push，供中斷後接續
     - 每一波翻譯結束、沒有批次在跑時，commit `tw` 的 `.meta` 並 push `tw`
     - 第一階段紀錄的 `model` 記 `phase1-session`
5. 過程中發現的規則問題，直接改進 `guides/`、`glossary/`
6. `lct build` 產生第一版 `tw`，與 `raw` 一起最終 push 到 repo；第一階段結束
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
- 相同的待譯行只翻一次：同族群、同欄位、（對話則）同說話者、KR 完全相同的待譯單位互為兄弟句；短字串（門檻同 §8.1）只限同檔
  - `batch next`：某單位的兄弟句正被另一個未過期的批次租用時，這次不發放它；同一批也不會放進兩個兄弟句
  - `submit`：收下一個單位後，把同一份譯文寫給它的兄弟句，寫入的紀錄與一般收下的單位相同（`Batch` 為這一批的 id，`Model` 為送出者）。兄弟句有租約且未過期、標了 `needs_review`、帶有人為這個單位寫的內容（人工要求重翻的說明 `hint`，或過期人工調整的舊譯文參考），或是這一批的單位，就不寫；這類單位在 `batch next` 也不因兄弟句而延後；譯文先以兄弟句所在檔的詞彙條目檢查，沒通過的不沿用，仍留在待譯清單。沿用的 key 列在結果的 `reused`
- `note` 用來回報疑問或新詞，進報表，不自動變成規則

### 9.2 CI 執行

- 用官方支援的兩種形式之一，兩者都讀 repo 的 `.claude/skills/` 與 `CLAUDE.md`：
  - **翻譯迴圈**：安裝固定版本的 Claude Code CLI，以 headless 模式執行（官方對 Python／TypeScript 以外語言的建議做法）
  - **規則 PR**（`rules-proposal.yml`）：用官方的 `anthropics/claude-code-action@v1`，`prompt: "/rules-review"`。它以 Claude GitHub App 的身分推送與開 PR，所以 `ci.yml` 會在這個 PR 上正常執行（用 `GITHUB_TOKEN` 開的 PR 不會觸發其他 workflow）。需要在 repo 安裝 Claude GitHub App。Claude 的認證同樣用 Console 的 API key（`anthropic_api_key: ${{ secrets.ANTHROPIC_API_KEY }}`），費用記在 Console；GitHub App 只負責 GitHub 上的操作，與計費無關
- build `lct`
- 迴圈執行：`claude -p "/translate-pending" --output-format json --model "$LCT_MODEL" --max-turns <N>`
  - 直到 `lct pending --count` 為 0、累計 `total_cost_usd` 達到 `LCT_MAX_USD`，或時間快到上限
  - 沒翻完的留在待譯清單，下次建置繼續；已翻完的照常打包，未翻部分用 `fallback`
- `.claude/settings.json` 限制工具：只允許執行 `lct`、讀取 repo、寫入 `work/`
- 模型由 repo variable `LCT_MODEL` 指定，預設值設在 variable，不寫在檔案；skill 不寫死模型
- 每次的花費、模型、批次數寫入 `.meta/build.json` 與 job summary

### 9.3 Skills（`.claude/skills/`）

| skill | 用途 | 誰用 |
|---|---|---|
| `translate-pending` | 依 §9.1 迴圈翻完待譯清單 | CI；第一階段在本工作階段中使用；人在本機執行時只產生預覽 |
| `weekly-update` | 在本機依序跑 fetch → plan → translate-pending → build，等同 CI 流程，結果只寫到本機的 `out/` 供進遊戲測試 | 人 |
| `adjust` | 人工調整譯文，並把調整整理進規則（§11） | 人 |
| `rules-review` | 把 `lct suggest` 的建議（§11.2）整理成指引、詞彙、規則的修改與 PR 說明；CI 以 `lct proposal publish` 開或更新規則 PR，人在本機只留下修改，自己 commit | CI（開 PR）與人 |
| `term-decision` | 決定詞彙表候選條目：候選來自 `lct suggest` 的 `term/…` 與 `lct glossary derive`（分頁）；每個詞由人決定採用（`status: approved`）、改譯法、不採用（`rules/declined.yaml`）或略過，譯法與現行譯文不同時提議重翻要求檔 | 人 |
| `review` | 依 `lct review`（與審核 Issue 相同的項目，讀本機 `tw` 的狀態）帶人逐項處理（調整譯文、改規則、確認過期的調整、決定是否解除凍結），最後提醒 commit 到 `main` 並勾選 Issue | 人 |

SKILL.md 只寫流程；翻譯指示在 `prompts/translate.md`，規則內容都引用 `guides/`、`glossary/` 的檔案。所有狀態變更都經由 `lct`，不手改 JSON／YAML。

依官方 skill 規格：

- 每個 skill 是 `.claude/skills/<名稱>/SKILL.md`，開頭 YAML 寫 `name`、`description`
- `allowed-tools` 預先允許該 skill 需要的工具（例如 `Bash(lct *)`），CI 與人工執行都不必逐次確認
- 只該由人或 CI 明確呼叫的 skill（`translate-pending`、`adjust`、`rules-review`、`term-decision`、`review`）設 `disable-model-invocation: true`，避免 Claude 在對話中自行啟動
- SKILL.md 保持在 500 行以內，其他內容放在引用的檔案；`CLAUDE.md` 保持精簡（每次執行都會讀）
- headless 模式（`claude -p "/skill-name"`）一次只能呼叫一個 skill

**寫入權**：第一階段的初始 commit 之後，`raw`、`tw` 分支與 Release 只由 CI 寫入。CI 對 `main` 只能開 PR，不能直接 push。人的所有介入都經由 `main`：人工調整寫 `rules/adjustments/`，重翻要求寫 `rules/requeue/`（CI 處理後在 `tw` 的紀錄中標記為已處理）。也可以在手動觸發 workflow 時帶參數。這樣不需要處理多人同時寫 `tw` 的衝突。

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
     raw: raw-20261029       # 調整時根據的原文版本
     kr_sha: …
     zh: 啊……但丁閣下，時鐘的時針好像在不停顫抖呢？您還好嗎？
     reason: 鴻璐對但丁說話用敬語，第二人稱用「您」
     rule: guides/characters/honglu.md   # 歸納到哪條規則；一次性的錯字修正寫 one-off
   ```

4. **把調整歸納進規則**：Claude 提出對應的指引或詞彙修改，人同意後寫入
5. 依新規則找出其他違反的 Claude 譯文，人同意後：
   - 要重翻：`lct requeue --to-file <主題>` 寫入 `rules/requeue/<日期>-<主題>.yaml`（`reason`、`keys`、`files`、`origins`；範圍與理由，至少一個 key 或 file）
   - 只是機械式的字串修正（例如關鍵字名稱後補半形空格）：寫 `rules/rewrite/<日期>-<主題>.yaml`（`reason`、`files`、`origins`、`pattern`、`replace`，選填 `kr_pattern`），由 CI 以正規表示式改寫現有譯文，不經過翻譯
   - 兩種檔都是一次性的：`plan` 套用後把檔名（去掉 `.yaml`）記在 `tw` 的 `.meta/applied.json`，之後不再套用；改檔名等於新的要求
   - 第一階段沒有 CI 的 `plan`，`lct requeue` 不加 `--to-file` 時直接寫入本機待譯清單
6. commit 到 `main` → 觸發重建（§12），CI 依調整、重翻要求與改寫檔更新 `tw`

人工調整只在它所根據的 KR 版本有效。KR 變了就依目前規則重翻；調整背後的道理已經寫進規則，會反映在重翻結果。

**查找**：因為插行會讓 key 改變，人工調整不能只靠 key 對應：

1. `key` 所指的單位 `kr_sha` 相同 → 套用
2. 否則在同一檔、同欄位中找 `kr_sha` 相同的單位（多個時取最接近原位置的）→ 套用，並在報表列為「key 已移動」；`rules-proposal.yml` 會在規則 PR 中順便更新這些 key
3. 都找不到 → 這筆調整已過期（KR 改了），該單位重翻

### 11.2 規則改進（`rules-review` skill）

- 來源：週報中的建議（不一致譯法、新說話者、新欄位、新檔案族群、Claude 的 `note`、詞彙候選）、人工調整的 key 移動與過期。調整背後的道理在 §11.1 第 4 步就歸納進規則（調整的 `rule` 欄位），這裡不再建議
- **CI 開 PR**：每次 `update.yml`／`rebuild.yml` 成功結束後，若有新的建議（`lct suggest` 略過已在規則 PR、已不採用、已處理的 id 之後不是空的），`rules-proposal.yml` 以 `anthropics/claude-code-action@v1`、`prompt: "/rules-review"`（§9.2）跑同一份 skill：
  - 在分支 `rules-proposal/<日期>` 修改 `guides/`、`glossary/`、`rules/`，對 `main` 開 PR
  - PR 說明逐項列出：建議內容、理由、證據（檔名#id、KR 與現行譯文、出現次數）
  - PR 說明附上影響評估：`lct impact --base origin/main --markdown …` 以目前 `tw` 的譯文，數出新規則比 `main` 的規則多讓多少既有單位出現硬性問題（沒有重翻的單位照舊）
  - 若建議要套用到舊譯文，PR 裡另附 `rules/requeue/*.yaml`；合併即代表人同意重翻
- **人審核**：可以直接改 PR 內容、刪掉不要的項目，再決定合併或關閉。合併後由 `rebuild.yml` 依新規則重建
- **不重複提出**：每個建議有穩定的 id（同一件事每次產生相同的 id）。人不採用的項目記在 `rules/declined.yaml`（在 PR 中移到這裡再合併）；關閉而未合併的 CI PR 中的項目也視為不採用。CI 會略過這兩者
  - `lct suggest [--exclude-file F]...` 從 `tw` 的 `.meta/build.json` 與 `.meta/plan.json`（只用 `raw_tag` 與目前 raw tag 相同的那份）、詞彙表與調整檔產生建議，結果依種類排序：`note`、`speaker`、`family`、`field`、`adjustment-moved`、`adjustment-expired`、`term`，最後是 `inconsistent`（單位數多的在前），同類同大小時依 id，同樣的輸入每次順序相同；`--kind`、`--limit`、`--offset` 在這個順序上選種類、取一頁；有建議也是 exit 0（建議不是「發現」）。`rules/declined.yaml` 與每個 `--exclude-file`（可重複，一行一個 id，空行與 `#` 開頭的行略過）中的 id 不再提出，輸出 `{suggestions, excluded}`。每個建議附至多 10 列證據，PR 說明以單獨一行的 `<!-- suggestion:<id> -->` 標示
  - id 是種類、`/` 與下表的內容；`sha8` 是 KR 或文字的 SHA-256 前 8 個小寫十六進位字元。`plan.json` 的新說話者與新檔案在同一個 snapshot 內不會隨規則重算，所以規則已經回答的項目每次執行都再檢查，已回答就不提：

    | 種類 | id | 來源 | 規則已回答，不提 |
    |---|---|---|---|
    | 不一致譯法 | `inconsistent/<族群>/<欄位>/<sha8(KR)>` | 報表的不一致譯法，每一組 | |
    | 新說話者 | `speaker/<model>` | `new_speakers` | `guides/characters/` 已認得這個 model：本人、另一時期的本人（`models_other_self`），或列為不是這個角色（`models_not_this`） |
    | 新欄位 | `field/<add\|remove>/<群組或 ->/<欄位>` | `field_changes`；識別欄位沒有群組，寫 `-` | |
    | 新檔案族群 | `family/<檔案>` | `new_files_default_family` | `guides/routing.yaml` 已把該檔歸入預設族群以外的族群 |
    | 譯者備註 | `note/<key>/<sha8(備註文字)>` | 報表的 `notes` | |
    | 詞彙候選 | `term/<kr>` | `glossary/terms.yaml` 中 `status: candidate` 的條目 | |
    | key 已移動的調整 | `adjustment-moved/<舊 key>` | `adjustments.moved` | 調整檔已改成新 key（沒有舊 key） |
    | 已過期的調整 | `adjustment-expired/<key>` | `adjustments.expired` | 調整檔已沒有這個 key |

  - `rules/declined.yaml` 是清單，每筆 `id`、`reason`（不採用的理由，必填）、`date`（`YYYY-MM-DD`）。`lct rules lint` 檢查 id 的種類是上表之一、`reason` 非空、`date` 的格式與 id 不重複
- 同一時間最多一個 CI 開的規則 PR；已有未處理的 PR 時，新建議加進該 PR：`lct proposal start --date …` 選分支（有開著的規則 PR 就用它的，更新到 origin 的樣子並把 `origin/main` 合併進去，讓程式、skill 與規則是 `main` 目前的樣子，合併有衝突就拒絕並列出衝突的檔案；沒有就是 `rules-proposal/<日期>`，origin 已有這個名稱且曾開過 PR 時用 `-2`、`-3`…），寫出 `work/proposal/exclude.txt`（PR 已有的 id、關閉未合併的 PR 的 id、已合併的不一致譯法；列在「需要人決定」一節的不算已合併，會再提出）與 `work/proposal/existing-body.md`（開著的 PR 現在的說明，人改過的也在內）。skill 以後者為底寫 `work/proposal/body.md`，`lct proposal publish --body-file …` 只 commit `guides/`、`glossary/`、`rules/` 的變更、推 `rules-proposal/*` 分支、開或更新 PR，並拒絕少了開著的 PR 已有的任一個 id 的說明，以及含有 token 或金鑰一類字串（`ghp_`、`gho_`、`ghs_`、`ghu_`、`github_pat_`、`x-access-token`、`AUTHORIZATION: basic`、`sk-ant-`）的說明與規則檔修改。skill 沒有把握的項目不改檔、不記進 `rules/declined.yaml`，列在說明的「需要人決定」一節（單獨一行的標記與一行理由），讓它進規則 PR、不再每次都排在前面；說明只新增這類項目也照樣發布（沒有開著的 PR 時 `lct proposal publish` 以一個沒有變更的 commit 開 PR），沒改規則檔、說明也沒有新項目時才拒絕（nothing to propose）。skill 結束前把結果寫在 `work/proposal/outcome.txt`（`published`、`nothing: <理由>`、`failed: <原因>`），workflow 的最後一步據此檢查：`published` 時 PR 的說明要有一個開始前沒有的 id，`nothing` 只留下記錄，其餘算錯誤——`lct proposal publish` 被拒絕時 action 不會失敗，已開著的 PR 不能當作這次有成果的證據
- 人也可以在本機用 `rules-review` skill 提出修改，流程相同
- **確認過的項目（審核頁，`review-page` skill）**：網站的審核頁（`docs/superpowers/specs/2026-11-03-review-page-design.md`）列出規則建議、檢查警告、譯者註記與待辦，人複製項目 ID 貼進對話，由 `review-page` skill 處理。「沒問題」用 `lct accept --reason R ID...`：規則建議與 `note/…` 記進 `rules/declined.yaml`，檢查警告 `warning/<代碼>/<sha8(KR)>/<sha8(TW)>` 記進新的 `rules/accepted.yaml`（格式與 `declined.yaml` 相同：`id`、`reason` 必填、`date`；`lct rules lint` 檢查；id 不含 key，所以單位換指標後確認仍有效，譯文一改 id 就變、確認失效）。ID 不是目前項目（依本機 `.work/raw`、`.work/tw` 判斷）時整批拒絕、不寫任何檔，exit 1；已在清單的 ID 略過；`--tidy` 刪除不再對應警告的 `accepted.yaml` 條目。「要改」必須讓譯文真的改變：寫調整，或改規則並附重翻要求檔。建置略過確認過的項目：報表的 `warnings` 計數與 `warning_samples` 不含 `accepted.yaml` 的 id，挑選前 500 組不一致譯法時先略過 `declined.yaml` 的 `inconsistent/…`
- 規則改了不會自動重翻舊譯文。要套用到舊譯文，必須由人指定範圍：用 `lct requeue`，或合併附有 `rules/requeue/` 的 PR（這都算人工介入）

## 12. GitHub Actions

| workflow | 觸發 | 內容 |
|---|---|---|
| `update.yml` | 每週四 11:35 UTC+8（cron `35 3 * * 4`）；`repository_dispatch`（`limbus_update`）；手動（可選 `force`、`requeue`、`max_usd`） | fetch → 有新 token 才 snapshot → plan → 翻譯迴圈 → build → commit `tw` → 語言包與最新的 Release 不同才建立 Release（§13）→ 有需要處理的項目時開審核 Issue（§14）。待譯清單不是空的也會繼續翻。沒有新 token、也沒有待譯，而且沒有還欠著的 Release（`lct publish --check`）、`tw` 最後一個 commit 不是只保存翻譯進度的（`translate: keep the progress of …`，保存後 build 失敗的情況），才直接結束 |
| `rebuild.yml` | push 到 `main` 且動到 `rules/**`、`glossary/**`、`guides/**`、`lct.yaml`；手動 | 不 fetch；plan → 有待譯才翻 → build → commit `tw` → 語言包與最新的 Release 不同才建立 Release → 有需要處理的項目時開審核 Issue |
| `rules-proposal.yml` | `update.yml`／`rebuild.yml` 完成且成功時（`workflow_run`，`conclusion == 'success'`）；手動 | §11.2：checkout（完整歷史）→ 取得 `raw`、`tw` 並建 worktree → build `lct` → 設定 git 身分 → `lct proposal start` 選規則 PR 的分支 → `lct suggest --exclude-file work/proposal/exclude.txt`，沒有新建議就結束 → `anthropics/claude-code-action@v1` 跑 `rules-review` skill（`--model` 用 `vars.LCT_MODEL`，`--allowedTools` 與 skill 的 `allowed-tools` 相同，`--disallowedTools "Read(./.git/**)"`；checkout 設 `persist-credentials: false`），開或更新規則 PR → 讀 `work/proposal/outcome.txt`，用 `gh pr list --state open --head <分支>` 確認 PR 有這次新增的建議並把網址寫進 job summary。逾時 60 分鐘 |
| `pages.yml` | `update.yml`／`rebuild.yml` 完成且成功、在 `main` 上、不是 PR 時（`workflow_run`，條件與 `rules-proposal.yml` 相同）；手動（沒有輸入） | 譯文對照網站（`site/`，設計見 `docs/superpowers/specs/2026-11-03-pages-viewer-design.md`）：checkout（完整歷史與 tag，`persist-credentials: false`）→ 取得 `raw`、`tw` 並建 worktree → build `lct` → `lct site --out site/public/data`（輸出含審核頁用的 `review/*.json`；非 0 就整個執行失敗，不部署）→ setup-node（版本取自 `site/.nvmrc`）→ `npm ci`、`npm run build`（於 `site/`）→ `configure-pages` → `upload-pages-artifact`（`site/dist`）→ `deploy-pages`。`environment: github-pages`，逾時 30 分鐘。第一次部署前，人要到 Settings → Pages 把 Source 選為「GitHub Actions」（README） |
| `ci.yml` | PR 與 push | `go test`；`lct rules lint`（指引開頭 YAML、詞彙表、調整檔的 schema）；skill 與 workflow 契約測試；改到規則的 PR 另外跑 `lct impact --base origin/<base>`（以目前 `tw` 的譯文與 `raw` 的原文，比較 PR 的規則和 base 分支的規則），把這次規則變更多讓多少既有單位出現硬性問題寫進 job summary；找到也不算失敗。另有 `site` job：於 `site/` 跑 `npm ci`、`npm run typecheck`、`npm test`、`npm run build`（只有 `npm test` 讀 `site/fixtures` 的小型測試資料；這裡的建置沒有網站資料；都不需要 `raw`／`tw`） |

- 所有寫入 `raw`／`tw` 的 workflow 共用一個 `concurrency` 群組，依序執行；`rules-proposal.yml` 只寫規則 PR，用自己的群組 `rules-proposal`（`cancel-in-progress: false`）；`pages.yml` 只讀 `raw`／`tw`，用自己的群組 `pages`（`cancel-in-progress: false`）
- secrets：`ANTHROPIC_API_KEY`、`LIMBUS_STATUS_URL`
- variables：`LIMBUS_CDN_TEMPLATE`（預設 `https://downloadcommon.limbuscompanycdn.org/{token}/Assets/LocalizePatch/localize_{lang}.zip`）、`LCT_MODEL`、`LCT_MAX_USD`
- `GITHUB_TOKEN` 需要 `contents: write`；`rules-proposal.yml` 另需 `pull-requests: write`、`issues: write`（建立 label）與 `id-token: write`（action 向 Claude GitHub App 換 token）；`update.yml`／`rebuild.yml` 開審核 Issue 需要 `issues: write`；`pages.yml` 只需要 `contents: read`、`pages: write`、`id-token: write`，不使用 secrets
- `rules-proposal.yml` 以 Claude GitHub App 的身分開 PR（§9.2），所以 `ci.yml` 會在規則 PR 上執行，影響評估也會出現在 PR 的 Checks（job summary）。`proposal start` 與確認 PR 的步驟用 `github.token` 讀 PR；action 內 `lct proposal publish` 需要的 token 由 action 提供，取不到時的處理見 README
- GitHub 只從預設分支執行排程；公開 repo 連續 60 天沒有活動時會停用排程。每週寫入 `raw`／`tw` 應能維持活動；若被停用，到 Actions 頁面重新啟用即可
- 排程在整點附近可能延遲數分鐘到數十分鐘

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

- 兩個 zip：只有文字、含字型
- 字型：Sarasa Gothic TC Bold（SIL OFL），固定版本並驗證 checksum，連同 OFL 授權文字一起打包
- Release tag `zh-TW-YYYYMMDD.N`；release notes 由報表摘要產生
- Release 只在語言包與最新一個 `zh-TW-*` Release 不同時才建立：比較兩個 `tw` commit 的 `LimbusCompany_Data/` 檔案樹（git blob），不含 `Info/version.json`（它記錄規則 commit、建置時間、模型與花費，每次建置都不同）。相同就不建立 Release，`tw` 照常 commit 與 push。`lct publish --check` 只讀 `tw` 的 git 狀態（HEAD、tag、檔案樹），回報目前的 `tw` 是否需要 Release、與哪個 tag 相同
- `version.json` 與 `.meta/build.json` 記錄：raw tag 與 token、`main` commit、模型、花費；第一階段的初始建置另記上游 release tag

## 14. 報表

每次建置產生一份報表，分成「需要你處理」與「資訊」兩部分，每一項都附連結。

### 14.1 你會在哪裡看到

| 地方 | 內容 | 怎麼知道有新的 |
|---|---|---|
| **審核 Issue**（每次遊戲更新一則，例如「審核：2026-11-05 更新」） | 只有「需要你處理」的項目，做成勾選清單 | GitHub 對新 Issue 的通知（網頁、Email、手機 App） |
| **規則 PR**（§11.2） | CI 提出的規則修改 | GitHub 對新 PR 的通知 |
| **GitHub Release 說明** | 精簡版：這次更新了什麼、各來源比例、仍顯示英文的數量 | 下載語言包時看到 |
| **Actions 執行頁的 summary** | 完整報表，另附 `tw` 的比較頁連結（建置前的 `tw` commit 到 `tw`），可直接看這次譯文變了什麼 | 進到該次執行查看；失敗時 GitHub 會寄信 |
| **`tw` 分支 `.meta/reports/<raw tag>.md`** | 完整報表的存檔，可回頭比較歷次；附 raw 的比較頁連結，不附 `tw` 的 | 需要時查閱 |

沒有需要處理的項目時不開 Issue。`lct review` 從本機 `tw` 的狀態印出同樣的項目（`{title, sections: [{kind, title, hint, items}]}`，與 CI 開 Issue 用的是同一份清單），`review` skill（§9.3）據此在 Claude Code 裡帶你逐項處理（接受、改譯文、改規則）。

### 14.2 需要你處理

| 項目 | 例 | 處理方式 |
|---|---|---|
| 翻譯失敗、暫用英文的單位（`needs_review`） | 3 次都沒通過驗證 | 看原因；用 `adjust` 修正，或改規則後要求重翻 |
| 人工調整已過期 | KR 改了，已依規則重翻 | 確認新譯文；沒問題就把舊調整刪掉 |
| 凍結檔的 KR 有變（`BgmLyrics/**`） | 歌詞原文被修改 | 決定是否解除凍結 |
| 翻譯沒有執行（`not_translated`） | 有待譯的單位，但沒有設定 `ANTHROPIC_API_KEY` 或 `LCT_MODEL` | 補上設定後手動執行 `rebuild`，待譯清單還在 |
| 未對應的說話者 | 新角色、新 NPC 的 `model` | 規則 PR 會附上建議的角色資料，審核 PR 即可 |
| 新的欄位、檔案族群 | 遊戲新增欄位 | 規則 PR 會附上建議的欄位規則 |

### 14.3 資訊

- 這次的輸入：raw tag 與 token、`main` commit、模型、花費
- 原文變化：新增／修改／刪除的檔；中間插入的檔；附 GitHub 比較頁連結（`raw-上次...raw-這次`），可直接看 KR 的差異
- 譯文變化：各來源（`paratranz`／`claude`／`human`／`copy`／`fallback`）的單位數；這次新翻、因 KR 變更重翻的數量；`tw` 的比較頁連結（建置前的 `tw` commit 到 `tw`）只放在 Actions 執行頁的 summary，不放進存檔的報表：它隨每次發佈而變，放進存檔會讓每次建置都為了報表而改動 `tw`（Release 是否建立只比較 `LimbusCompany_Data/` 的語言包，報表在 `.meta/` 底下，不影響，§13）
- 自動處理的事：人工調整「key 已移動」、id 重新編號後沿用的譯文
- 跨語言順序或 id 不一致的檔，「未對齊」的單位數
- 驗證警告：數量與前幾筆，完整清單在存檔
- 同一 KR 不同譯法的清單；Claude 的 `note`

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
| `lct publish [--check]` | commit `tw`、備妥 Release（tag、說明、zip；語言包與最新的 Release 不同才需要建立）；`--check` 只讀 `tw` 的 git 狀態，回報目前的 `tw` 是否需要 Release、與哪個 tag 相同，不 commit |
| `lct search` | 依 key、文字、說話者、檔名找單位 |
| `lct adjust add` | 新增人工調整 |
| `lct requeue` | 指定範圍送回待譯清單；`--to-file` 只寫 `rules/requeue/` 的要求檔，不動狀態（§11.1） |
| `lct adjust tidy` | 調整檔跟上目前 snapshot 的 plan：key 已移動的改成新 key，單位已重翻的過期調整刪除（§11.1） |
| `lct suggest [--exclude-file F]...` | 規則還能學到什麼，附穩定的 id；略過 `rules/declined.yaml` 與排除檔的 id；有建議也是 exit 0（§11.2） |
| `lct accept --reason R [--date D] [--file F] [ID...]`、`lct accept --tidy`、`lct accept --dry-run [--file F] ID...` | 把審核頁上確認過的項目記進 `rules/declined.yaml`（規則建議、`note/…`）或 `rules/accepted.yaml`（`warning/…`）；不是目前項目的 ID 整批拒絕（exit 1）；`--tidy` 刪除已不對應警告的 `accepted.yaml` 條目；`--dry-run` 唯讀，印出每個 ID 在審核頁上的那一項（種類、記進哪個清單、是否已確認、KR／TW、註記或檢查說明、單位、各種譯法），不是目前項目的 ID 列在 `unknown`（exit 1）；讀 `.work/raw`、`.work/tw`（§11.2） |
| `lct impact --base REV [--markdown F]` | 以目前工作樹的規則和 REV 的規則檢查 `tw` 的既有譯文，數出新規則多讓多少單位出現硬性問題；規則 PR 的影響評估（§11.2、§12） |
| `lct proposal start／publish` | `start` 選規則 PR 的分支（重用的分支會合併 `origin/main`，衝突就拒絕）、寫出 `work/proposal/exclude.txt` 與 `existing-body.md`；`publish` 只 commit 規則檔、推 `rules-proposal/*` 分支，開或更新規則 PR，拒絕含有 token 或金鑰一類字串的說明與修改；需要 `GH_TOKEN` 或 `GITHUB_TOKEN` 與 git 身分（§11.2） |
| `lct translate` | 用 claude CLI 執行 `translate-pending` skill，直到沒有待譯、花費或時間用完，或某次執行失敗（§9.2）；`--not-run REASON` 不執行 claude，只在 `.meta/translate.json` 記下翻譯沒有執行與原因（CI 有待譯、但缺 API key 或模型，或 claude CLI 安裝失敗時用，審核 Issue 因此列出「翻譯沒有執行」與原因，§14.2；dry run 不記） |
| `lct fields learn` | 重算欄位規則 |
| `lct glossary derive [--source keyword\|speaker] [--limit N] [--offset N]` | 從 `BattleKeywords*`、`Bufs*`（`keyword`）與 `ScenarioModelCodes`（`speaker`）列出詞彙表還沒有、也沒被不採用的名稱，作為候選詞條；只輸出，不寫檔；依 KR 排序，`--limit`／`--offset` 取其中一頁，JSON 為 `{total, candidates}`（`total` 是頁數限制前的總數） |
| `lct review` | 目前 `tw` 狀態中需要人處理的項目（與審核 Issue 相同）：標題與各節的 `kind`、`title`、`hint`、`items`；沒有 `tw` 的建置紀錄時 exit 1；§14 |
| `lct rules lint` | 規則檔 schema 檢查 |
| `lct site --out DIR` | 輸出譯文對照網站要的資料（索引、各檔的 KR／JP／EN／TW 對照、全文搜尋用 corpus，以及審核頁的 `review/*.json`，格式見 `docs/superpowers/specs/2026-11-03-pages-viewer-design.md` §5）；讀 `raw`、`tw` 兩個 worktree，`DIR` 相對於 repo 根目錄，會先清空；`DIR` 在 repo 之外、是 repo 根目錄、含 `.git`、含被 git 追蹤的檔案（會先清空，不能清掉原始碼）、或包含／位於 `raw`、`tw` worktree 時拒絕；`tw` 沒有建置紀錄、或建置的 snapshot 不是 `raw` 目前的 snapshot 時 exit 1 |

所有指令支援 `--json` 輸出；exit code：0 成功、1 有發現或使用者錯誤、2 系統錯誤。`raw`／`tw` 分支以 `git worktree` 操作。

## 16. Repo 結構（`main`）

```text
cmd/lct/                 CLI 入口
internal/…               fetch、snapshot、align、units、rules、guide、validate、pack、report…
guides/  glossary/  rules/   （`rules/accepted.yaml`：審核頁確認過的檢查警告，§11.2）
testdata/                測試用的官方原檔片段
prompts/                 翻譯指示（skill 與第三階段共用）
schemas/                 批次輸入／輸出、規則檔的 JSON schema
lct.yaml                 輸出資料夾名、字型、凍結清單、批次大小、重試次數、第一階段用的上游版本
.claude/skills/          §9.3（含 `review-page`：處理審核頁貼上的項目 ID，§11.2）
.claude/settings.json    工具權限
CLAUDE.md                專案說明（AGENTS.md 指向它）
LICENSE                  程式碼授權（GPL-3.0）
LICENSE-CONTENT          內容授權（CC BY-NC-SA 4.0），說明適用範圍
site/                    譯文對照網站（Preact＋Vite；`lct site` 輸出的資料在 `site/public/data/`，與 `site/dist/`、`site/node_modules/` 都不進版控）
.github/workflows/       §12
docs/
```

## 17. 測試

- Go 單元測試：對齊（含插行）、來源決策、Paratranz 有效判定與 `[ID]` 還原、驗證規則
- 真實資料 fixture：從 10/29 語言包挑小量檔案
- 跨語言對齊 fixture：`EventVerificationUIText.json`、`Passives.json`（換順序）、`StoryData/E001X.json`、`StoryData/S706B.json`（id 打錯）、`StoryData/P10511.json`（多一行）
- 重播測試：驗證「不變不重翻、變了才重翻」，包含中間插入的情況與人工調整 key 移動。第一階段時，把幾組有中間插入的官方原檔前後版本（例如 `StoryData/S918A.json`、`EGOgift_MirrorDungeon.json`、`Bufs*`，取自調查時看過的歷史）複製進 `testdata/`；之後再以本 repo 的 `raw` 歷史補充。測試不在執行時讀取 `tool-jx3/Assets`
- 不呼叫 Claude 的乾跑模式（待譯全部走 `fallback`），讓 CI 能完整跑過流程
- skill 契約測試：skill 引用的檔案與 `lct` 指令都存在；不含模型名稱

## 18. 已決定的事項

**使用者決定**

- Go；Claude Code（Agent SDK）＋ Console API key；skill 與自動化同步（做法 A）
- GitHub Actions；原文與譯文各一個 orphan 分支；狀態 API 網址放 secret
- 兩個階段的範圍（§2）；兩階段都能改進規則
- 第一階段不自動化，在 Claude Code 工作階段中一次完成並上傳（存檔點除外）；之後的 GitHub Actions 都是第二階段
- CI 可以開 PR 改進規則，由人審核決定是否調整、合併
- 做法 B 列為第三階段，第二階段穩定後再開發
- 需要人處理的項目以 GitHub Issue 通知（§14）
- `update.yml` 每週四 11:35 UTC+8 執行一次。其他日子的臨時更新與未翻完的待譯清單，靠手動觸發、`repository_dispatch` 或下週四處理
- 輸出檔案以 KR 為準（§6.2）
- `StoryData` 的 `PC*` 是明日方舟聯動劇情，`ES*` 是第八章的特殊劇情模式
- 不動 `tool-jx3/Assets`、`tool-jx3/ParatranzUploader`，本專案也不依賴這兩個 repo（§3）
- `raw` 從 2026-10-29 開始，不匯入 `tool-jx3/Assets` 的歷史
- 程式碼授權 GPL-3.0
- KR 為主、JP／EN 輔助
- 原文沒變不重翻；KR 變了依變化重翻
- Release（文字版＋含字型版）只在語言包內容與最新一個 `zh-TW-*` Release 不同時才發（不含 `Info/version.json`，§13）；`tw` 每次建置照常 commit 與 push
- 人工調整不是最高優先的永久覆寫，而是歸納進規則
- 依檔案族群與角色建立指引
- 一律保留 `[英文ID]`；您／你依 KR；你／妳依對象性別（但丁用你／他）；唐吉訶德稱「經理老爺」「浮士德小姐」；全大寫替換變數不加空格；翻譯優先序照列出順序

**設計者的判斷（可推翻）**

- 輸出資料夾名 `zh-TW`；字型 Sarasa Gothic TC
- 人工調整存在 `main` 的 `rules/adjustments/`，與其他規則一起審核
- 翻譯失敗暫用 EN 原文
- 內容授權：`guides/`、`glossary/`、`rules/` 與 `tw` 分支的譯文用 CC BY-NC-SA 4.0；`raw` 分支是 Project Moon 的原文，著作權屬於原作者，只為比對差異而保存
- `raw`、`tw` 與 Release 只由 CI 寫入；人工介入一律經由 `main`（§9.3）
- Paratranz 匯入先比對識別欄位骨架，對不上的單位不匯入（§8.3）
- 名稱表（關鍵字、說話者）取自本專案現行譯文，直接具約束力；Claude 提出的新詞要審核（§7）
- 結構類驗證一律強制；內容類驗證對規則改變後的既有單位只警告（§10）
- 凍結（`BgmLyrics/**`）＝既有譯文不被取代，KR 變了也不動、只報告；新加入的歌詞照一般流程翻譯
- 規則 PR 用官方的 `anthropics/claude-code-action`；翻譯迴圈用 Claude Code CLI 的 headless 模式（§9.2）
- 第二階段的模型由 repo variable `LCT_MODEL` 指定，預設值設在 variable，不寫在檔案；每次建置花費上限 `LCT_MAX_USD` 預設 30 美元，都可隨時改 variable

## 19. 交給實作者決定

- Go 套件拆分、YAML／JSON 函式庫、序列差異演算法
- 相同文字重用的長度門檻
- 批次大小、重試次數、`--max-turns` 的預設值
- 簡體專用字的判定方式
- 字型取得方式（建置時下載固定版本，或放在 repo）
- release notes 的格式

## 20. 待確認

目前沒有。
