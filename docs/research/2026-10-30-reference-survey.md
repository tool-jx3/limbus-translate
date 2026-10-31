# 參考專案調查筆記（2026-10-30）

> 這份是研究紀錄，不是設計文件。目的：保存調查到的事實，讓之後的設計與實作有依據。
> 資料來源：實際下載 2026-10-29 的官方語言包、clone 下列專案閱讀程式與文件。

## 0. 已確認的需求（使用者補充）

- 翻譯以 KR 韓文為主，JP 日文與 EN 英文作為輔助
- 已翻譯過的部分，除非人工介入或原文有改變，不重新翻譯
- 官方 JSON 有些欄位不需翻譯（例如 `id`、`model`），可從三語檔案比對歸納出規則（見 §1）
- 翻譯時要針對不同檔案族群建立對應的指引，例如不同角色、不同檔名類型（見 §1「檔案族群與角色」）
- 分兩階段，因此不需要 Paratranz API，只需在第一階段匯入一次 release 檔案：
  - **第一階段**：以 `user-unknown-0711/ParatranzUploader` 的譯文為主；未翻譯的部分依整理出的規則翻譯，並以 LLC 簡中或 LTM 繁中作為輔助參考
  - **第二階段**：不再使用上游譯文，依規則自己翻譯；規則可以持續改進；支援人工介入調整舊的譯文
- 指引中的分歧已決定（2026-10-31）：第二人稱您／你依 KR 判斷；唐吉訶德稱但丁「經理老爺」、稱浮士德「浮士德小姐」；你／妳、他／她依對象性別區分（但丁一律用你／他）；全大寫替換變數前後不空格；翻譯優先序照列出順序；「특정 추출」統一為「特定提取」；「전용 파편」譯「專用碎片」（候選）
- 既有社群指引已整理進 `guides/`、`glossary/`，並以現有繁中譯文補充（見 §7）

## 1. 官方原文（生肉）

### 下載方式

- 狀態 API：`GET <status_url>` → `{"latest_token":{"created_at":..., "f_token":..., "token":...}}`
  - voidfissure 版本：`https://limbus-api.voidfissure.de/api/status`
  - LCTA 目前改用自己的：`https://limbus.lcta.top/api/status`（同格式，見 `HZBHZB1234/LCTA_auto_update` `src/config.yaml`）
  - LCTA 另提供可自架的狀態服務（`tools/status_server/`，用 Steam 客戶端或 steamcmd 讀 `resources.assets` 取 token）
- 下載樣板：`https://downloadcommon.limbuscompanycdn.org/{token}/Assets/LocalizePatch/localize_{lang}.zip`，`lang ∈ {kr, en, jp}`
  - 已驗證：用 `token`（`l2026...`）下載成功；用 `f_token`（`f2026...`）同路徑回 404，語言包用不到 `f_token`
  - CDN 本身不需認證；只有 token 需要從狀態 API 取得
- LCTA 用 `repository_dispatch`（event `voidfissure_update`）讓 voidfissure 在偵測到更新時主動觸發，不再輪詢

### 檔案格式（以 2026-10-29 版實測）

| 項目 | 實測結果 |
|---|---|
| zip 大小 | en 6.1 MB、kr 6.0 MB；解壓後 en 約 30 MB |
| 根目錄 | `LocalizeTemp_en/`、`LocalizeTemp_kr/` |
| 子目錄 | `BattleAnnouncerDlg/ BgmLyrics/ EGOVoiceDig/ PersonalityVoiceDlg/ RPGSystem/ StoryData/` |
| 檔名 | 帶語言前綴：`EN_Passives-a1c9116.json`、`KR_...`；LLC 與自訂語言輸出**不帶前綴**，且沿用 EN 的檔名 |
| 檔案數 | en 2376、kr 2419；共同 2366，en 獨有 10、kr 獨有 53（兩邊檔名不完全對應，需要對照表或以某語言為骨架） |
| 編碼 | UTF-8；12 個檔有 BOM；2349 個檔是 CRLF |
| 結構 | 多數為 `{"dataList":[{...}]}`；16 個檔沒有 `dataList`；190 個檔有不帶 `id` 的項目；Hakimi 指出有重複 id → **不能把檔案壓成 id→entry map**，要用路徑（含陣列索引）當 key |

### 哪些欄位是「文字」、哪些是「識別碼」

比較 EN 與 KR 同一位置字串是否相同（相同＝很可能是識別碼，不該翻）：

| 欄位 | EN≠KR / EN=KR | 判斷 |
|---|---|---|
| `content` | 70931 / 1720 | 文字（相同者多為 `......` 之類） |
| `desc` | 35247 / 4956 | 文字（但 `BattleSpeechBubbleDlg.desc`、語音 `desc` 是事件標籤，見 Hakimi） |
| `name` `title` `teller` `text` `dlg` `speaker` `place` ... | 大多不同 | 文字 |
| `model` | 73 / 46392 | **識別碼**（劇情的說話者模型，值是韓文如 `이스마엘`，不可翻） |
| `id` `key` `songWriter` `codeName` `usage` `imgStr` `colorCode` | 全部相同 | 識別碼 |

三語（KR/EN/JP）同時比對（共同檔案 2366 個；15 個檔三語的字串路徑集合不一致）：

- **三語永遠相同 → 不翻**：`key`(3744)、`songWriter`、`usage`、`imgStr`、`debugNodeId`、`debugNodeID`、`colorCode`、`outlineColorCode`、`timeline`、`chapterNumber`、`keywords`、`iconId`、`iconID`、`mainTextColor`、`mainTextGlowColor`
- **幾乎相同但有例外 → 需人工定規則**：`model`（不同 78 / 相同 46387）、`id`（14 / 23447）、`codeName`（2 / 181）、`subDesc`（5 / 35）
- **依檔案而定**：`BattleSpeechBubbleDlg.desc` 三語相同 827 / 不同 2（是觸發標籤，台詞在 `dlg`）；其他檔的 `desc` 是正文
- **單一值三語相同**（例如 `content` 裡的 `......`）：不需送翻，直接沿用

結論：規則可以分三層自動產生——欄位層（整個欄位三語皆同）、檔案群×欄位層（如上例）、單值層（這一格三語相同）——並把產出的規則檔 commit，新欄位出現時在差異裡看得到。

檔案集合：KR 比 EN 多 53 個檔（例如 `Passives-a1c9p1~p3.json`），LLC 簡中也沒有這些檔。以 KR 為骨架時，要先確認遊戲實際讀取的檔名清單（`tool-jx3/Assets` 裡的 `RemoteLocalizeFileList.json` 可能就是），避免把遊戲不讀的檔也拿去翻。

Hakimi 的規則（`tools/resource_fields.py`）：`id/key/index/level/model` 不翻；`notinclude` 檔不翻；`//`、`SE //` 開頭的來源字串不翻；`Items` 的 desc 寫著 `name만 번역` 時只翻 name。

### 檔案族群與角色（以 KR 字元數計）

| 族群 | 檔數 | 占 KR 字元 | 備註 |
|---|---|---|---|
| `Skills*` | 127 | 20.5% | 技能名稱與效果，術語密度最高 |
| `StoryData/P1xxxx` | 181 | 約 5.6% | 人格故事；`P10805` 的 `108` = 8 號罪人（以實瑪利）、`05` = 人格序號 |
| `StoryData/S*` | 數百 | 約 10% | 主線（`S1037B` 為第 10 章） |
| `StoryData/E*`、`PC*` 等 | 數百 | 約 8% | 其他劇情（活動、間章等，前綴意義待確認） |
| `Passives*` | 91 | 5.1% | |
| `RPGSystem/rpg-loc-*` | 189 | 5.1% | RPG 探索的對話、NPC、選項 |
| `PersonalityVoiceDlg/Voice_<角色>_<人格>_<編號>` | 194 | 4.7% | 人格語音；`desc` 是事件標籤（如 `인격 획득`），台詞在 `dlg` |
| `Bufs*`、`BattleKeywords*` | 101 | 6.5% | 狀態效果與關鍵字（詞彙表來源） |
| `AbnormalityGuides*`、`AbEvents*`、`ActionEvents*`、`ChoiceEventEffect*` | 108 | 約 7.6% | 異想體觀測與事件文本 |
| `EGOgift*` | 29 | 2.7% | |
| `BattleAnnouncerDlg/Announcer_<名稱>` | 65 | 2.2% | 播報員，每個播報員一個檔 |
| `Tutorial*`、`MainUIText*`、`MirrorDungeonUI*`、`UI_*` 等 | — | 約 4% | 系統介面 |
| `BattleSpeechBubbleDlg*` | 11 | 1.0% | 戰鬥對白；`desc` 是觸發標籤 |
| 其他（`StoryTheaterDanteNoteDetail`、`Items`、`BgmLyrics/*`、`Enemies*`…） | — | — | 長尾 |

角色辨識的三個層次：

1. **檔名**：`PersonalityVoiceDlg/Voice_<角色>_*`、`EGOVoiceDig/Voice_EGO_<角色>_<n>`、`AbDlg_<角色>`、`Announcer_<名稱>`。拼法不一致（`Donquixote`/`DonQuixote`、`Outist`、`Meursalut`），需要別名表
2. **編號**：人格編號 `1SSII`（`SS` = 罪人序號 01 李箱…12 格雷戈爾，`II` = 人格序號），用於 `StoryData/P1SSII`、語音檔尾碼、`Skills_*` 等
3. **逐行**：`StoryData` 每行的 `model`（韓文說話者鍵值）。出現次數最多：단테 2987、파우스트 2516、이스마엘 2365、히스클리프 2024、오티스 1919、홍루 1852、싱클레어 1850、로쟈 1832、돈키호테 1695、그레고르 1679、이상 1562、료슈 1425、뫼르소 1202、베르길리우스 761。也有變體鍵值（`단테2`、`과거료슈3`），需要對到同一角色。`ScenarioModelCodes-AutoCreated` 提供 `{id, name, nickName}`（例：`이상` → `1번 수감자`）

同一角色的不同人格語氣可能差很多：例如 `Voice_DonQuixote_Bloodfiend_10310` 用正式的 `~겁니다`、稱對方 `당신`，與唐吉訶德平常的語氣不同。指引可能需要「角色 → 人格」兩層。

StoryData 每行的欄位組合：`content+model` 33972、只有 `content`（旁白）13669、`content+model+teller+title` 6958、`content+model+title` 3937；`place` 只在場景開頭。

### 自訂語言包的安裝結構（ParatranzUploader / LTM 的發佈包）

```text
LimbusCompany_Data/
  Lang/
    config.json            # {"lang": "<資料夾名>"}
    <資料夾名>/
      *.json, StoryData/ ...
      Font/Context/*.ttf   # 內文字型
      Font/Title/*.ttf     # 標題字型
```

ParatranzUploader 與 LTM 都用 Sarasa Gothic TC Bold（SIL OFL）。

## 2. 上游譯文來源

| 來源 | 語言 | 取得方式 | 更新節奏（觀察） | 授權 |
|---|---|---|---|---|
| `user-unknown-0711/ParatranzUploader` releases | 繁中（Paratranz 校對） | 資產 `localize_files.zip`（`Lang/TW` 內容）、`CLT_only_text_*.zip`、`CLT_complete_*.zip` | 週四當天或之後數天，release notes 會列「尚未翻譯」的部分 | 內容 CC BY-NC-SA 4.0；程式 GPL-3.0 |
| `LimbusTraditionalMandarin/storyline` releases | 繁中（由 LLC 轉繁再校對，主要是劇情） | 資產 `Hant.zip`；另有 `allinone` pre-release | 約每週，最近一次 10/22 | CC BY-NC-SA 4.0 |
| `LocalizeLimbusCompany/LocalizeLimbusCompany` releases | 簡中 | 資產 `LimbusLocalize_*.zip`（main 分支已停在 2026-08-06，要用 release） | 週四之後陸續 | CC BY-NC-SA 4.0 |

**ParatranzUploader 匯出邏輯的關鍵細節**（`main.go` `exportAssetsWithArtifact`）：
以 KR 檔為骨架 → 先整個覆蓋 EN 文字 → 再套第二個 Paratranz 專案（去掉 `[LLC]` 前綴）→ 再套主專案。
所以它 release 裡「未翻譯」的條目是**英文原文**，不是空字串。合併時必須把「值 == EN 原文」視為未翻譯，否則備援鏈永遠不會觸發。
如果能拿到 Paratranz API（artifact 裡有每條的 stage），判斷會比字串比對可靠。

簡轉繁：ParatranzUploader 的 `opencc/custom.json` 以 `STPhrases + STCharacters` 為底，加上 `dictionary/CustomPhrases.txt`（195 條，例如 `使用本技能后→使用本技能後`）與 `CustomCharacters.txt`（6 條）。此字典源自 LTM 的 ptsd 專案（GPL-3.0）。

## 3. 現有管線（tool-jx3 帳號下）

- `tool-jx3/Assets`：main 分支直接存 `kr/ en/ jp/` 原文、`RemoteLocalizeFileList.json`、`etc/VoiceTable.json`；commit 訊息 `Update: <時間>`；push 後用 `DISPATCH_PAT` 對 `tool-jx3/ParatranzUploader` 發 `release-from-assets`
- `tool-jx3/ParatranzUploader`：與 upstream 不同 HEAD，`sync-assets` 把差異上傳 Paratranz，`release` 從 Paratranz artifact 匯出打包

新專案的「原文 orphan branch」在功能上與 `tool-jx3/Assets` 重疊。

## 4. 參考專案可借鏡處

### wayne930242/game-doc-template（skill 與自動化同步的做法）

- 只有 skills，沒有 commands/agents；子代理 prompt 是 skill 目錄裡的 `*-prompt.md`
- 跨工具共用：`AGENTS.md -> CLAUDE.md`、`.codex/skills` `.gemini/skills` `.agents/skills` 都 symlink 到 `.claude/skills`
- 確定性的工作都放進腳本（`--json` 輸出、exit code 0/1/2、`--fail-on-*`），狀態只能透過 CLI 改（禁止手改 JSON）；LLM 只做語意工作
- 詞彙表 schema：`zh`、`status`（approved/candidate/deprecated）、`keep_english`、`original_term`、`forbidden[]`；未知欄位拒絕
- `translation_context.py` 用來源、詞彙表、風格檔的 sha256 判斷需不需要重建 context
- `style-decisions.json`：`{decision, reason, alternatives[]}`，整份注入翻譯與審校 prompt
- 契約測試：檢查 SKILL.md 的步驟順序與必要字句、禁止寫死模型名稱
- 臺灣繁中規範：全形 `，。、；：「」『』（）`、`……`、不用支語

### buxinzi2233/LimbusCompany-CN-Hakimi（遊戲格式規則與 references）

- `docs/references/` 七類：風格與格式規範、角色口吻（自稱、稱呼他人、口頭禪，引用資源 ID）、賽季文學考據、名詞檔、各章劇情摘要、世界觀、哲學脈絡
- 這些筆記**不自動注入 prompt**，由 `TRANSLATION_GUIDE.md` 的「哪個問題查哪裡」表格按需查閱；早期 AI 寫的筆記有捏造，後來每條都要附檔案＋ID 佐證
- 格式規則：`{0}` 等佔位符數量須與 KR 一致；TextMeshPro 標籤成對且反序關閉（`sprite`、`br` 自閉合），屬性值不翻；`[關鍵字]` 後接半形空格（他們自己承認未在遊戲內驗證）；禁全形波浪號（字型缺字）
- 工具：linter L01–L08、per-field diff（missing_field、placeholder_mismatch、korean_text…）、reconcile（既有譯文 → 官方 LLC → 原文）、release gate 區分「舊問題」與「新回歸」
- 「僅戰鬥」版：拿掉新章節 StoryData 等檔，讓遊戲退回原文，避免劇透
- 不要學：約 30 個把翻譯字典寫死在程式裡的腳本、季節特定的 E2E 測試、份量重但易捏造的哲學長文

### HZBHZB1234/LCTA_auto_update

**v1（現行）**：Actions 取 token → 下載 jp/kr/en → 取 LLC 最新 release 當「熟肉」→ 先處理 `BattleKeywords*`（整併成狀態效果表 `{id,kr,jp,en,cn}`）與 `ScenarioModelCodes-AutoCreated`（說話者表）→ 其他檔並行。每檔：找出 LLC 尚未覆蓋的（條目, 欄位路徑）→ Aho-Corasick 比對術語／效果／角色 → 消歧 → LLM 以 KR 為主、JP/EN 輔助翻成簡中（回傳含 `confidence`、`new_terms`）→ 確定性驗證與自動修正 → 以 KR 結構重建。低信心退回 KR 原文；同 raw token 會跳過。

**v1 的缺口**：只看「有沒有譯文」，不看「原文有沒有變」，KR 改了舊譯文就默默過期。

**v2 規劃（全部未實作）**中值得帶走的觀念：

- 先查再譯：P1 依路徑查權威譯文、P2 完全相同的 KR 文字直接重用、P3 從權威譯文挖術語實際譯法（`docs/converge-to-authority.md`）
- KR 為基準；以穩定 ID 對齊而非劇情陣列索引（插入一行會讓後面全部錯位）（`04`）
- 術語來源分層：觀察到／驗證過／官方發佈；模型自創的名字永遠不升格為權威（`05`）
- 權威譯文晚到是常態：分為採用／重譯／重驗／等待（`12`）
- 每次發佈都綁定所有輸入版本（遊戲 token、上游資產 hash、詞彙表版本、prompt 版本）（`14`）

**可直接照抄的機制**：

- 單位 key＝（去前綴檔名, 條目, JSON 欄位路徑）；跳過 `id`/`usage`/`model`、數字、null、`''`、`'-'`
- 「已覆蓋」判定：權威值為空而 KR 有字、或權威值等於 KR 且含諺文 → 未覆蓋。繁中版要再加：等於 EN 原文、含假名、含簡體字
- 逐欄位填入 KR 骨架，**不整條替換**（整條替換會吞掉舊 id 下新增的技能等級或硬幣）
- 驗證：`[ID]` 空白、KR/JP/EN 出現的英文效果 ID 必須在譯文出現（ID 或其中文名）、跳脫的 rich text 還原、殘留諺文＝未翻、回傳數量與 id 對齊；v1 沒做但該補的：`<0>` 類佔位符完全一致、標籤平衡、殘留簡體字
- 修復迴圈：確定性修正 → 只把失敗的格子送回 LLM → 通過同一套檢查才收
- 報表：同一 KR 文字或同一 id 出現不同譯法時列出

**對單人維護的週更專案過重**：「完全無人」原則（與你要的人工介入相衝）、八種 AI 角色與仲裁、持久工作圖、逐 commit 追 LLC、實體圖、向量檢索、觀測台。全量重建一次只要幾秒（約 23 萬個欄位），冪等重建就夠。

**資料事實補充**（LCTA 的統計）：每次 LLC release 約 98.98% 欄位不變、0.65% 修訂、0.37% 新增；`StoryData/` 以陣列位置對齊；英文引擎 ID 如 `[Bleed]`、`[OnSucceedAttack]` 必須原樣保留；佔位符 `<0>`、rich text `<color=#…>` `<mark>` `<size>` `<noparse>`；遊戲字型缺 `～`。

## 5. Claude 整合

- 官方 Claude Agent SDK 只有 Python 與 TypeScript。官方文件對其他語言的建議是：以子行程執行 `claude -p ... --output-format json`（headless），這就是 Agent SDK 底層在做的事（SDK 會啟動 Claude Code binary）
- Agent SDK / headless 模式會自動載入專案 `.claude/skills`、`CLAUDE.md`，因此 CI 跑的 skill 與人在 Claude Code 裡用的 skill 是**同一份檔案**
- 認證：Claude Console 的 API key → GitHub secret `ANTHROPIC_API_KEY`
- Go 直接呼叫 API：`github.com/anthropics/anthropic-sdk-go/v2`，可用 Message Batches（半價、非即時）、prompt caching、structured outputs（`output_config.format`）

## 7. 既有繁中語料（2026-10-29）

- ParatranzUploader release `1793285016` 的 `localize_files.zip`：2380 檔
- LTM release `1792667238` 的 `Hant.zip`：2348 檔（不只劇情，是完整語言包）
- 需要翻譯的單位（排除識別欄位與三語相同的值）共 181,855 個：

| 狀態 | Paratranz | LTM |
|---|---|---|
| 確實是中文譯文 | 89.7% | 97.9% |
| 等於 EN 原文（未翻） | 8.6% | 0.2% |
| 不含漢字 | 1.4% | 1.4% |
| 缺檔或缺條目 | 0.1% | 0.4% |

- Paratranz 覆蓋最差的族群：`RPGSystem` 62.2%、`BattleSpeechBubbleDlg` 85.4%、`StoryData` 86.9%
- 兩邊都有中文的 162,639 個單位中，有 64,269 個（約 40%）譯文不同，是兩條不同的翻譯脈絡（LTM 由 LLC 轉繁再校對）。角色名也不同，例如莫梭／默爾索
- 只有 GitHub release 的直接下載網址可用；GitHub API 對未掛載的 repo 會被這個環境的代理擋下

## 6. 實測過程備註

- 本次雲端環境的網路政策擋掉 `limbus-api.voidfissure.de` 與 `store.steampowered.com`，所以 Steam 公告原文與狀態 API 回應沒有直接讀到；CDN（`downloadcommon.limbuscompanycdn.org`）可連
