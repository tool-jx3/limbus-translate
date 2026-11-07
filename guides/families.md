# 檔案族群

給人看的參考：檔名規律、族群與節點、翻譯優先序、凍結項目。這份檔案**不送給譯者**；送給譯者的族群規則在 `guides/families/<族群 id>.md`（由 `guides/routing.yaml` 各族群的 `guide:` 指定）。

## 多數檔案命名規律

### 原指引

- 主線劇情
  - S**YXX** 為主線劇情，**Y** 表章節序，例：S**0XX** 為序章，依此類推
  - **Z**D**YXX** 為主線地牢，**Z** 表章節序，**Y** 表地牢層數，例：**1**D**2XX** 為第一章地牢第二層，依此類推
    - AbEvents-a1c**Z**p**Y** 為主線地牢事件，**Z** 表章節序，**Y** 表地牢層數
- DungeonNode 為主線地牢和間章地牢節點
- 間章劇情／活動為：E**XXX**
- 人格故事為：P1**YYXX**，**YY** 為罪人代碼
  - 辛克萊為 P1**10XX**，奧提斯為 P1**11XX**，格里高爾為 P1**12XX**
- BgmLyrics 為地牢漂浮歌詞及片頭片尾曲歌詞等
  - ***此項目已施工完畢，暫時禁止更動***
- BattleAnnouncerDlg 為播報員語音
- EGOVoiceDig 為 EGO 語音
- PersonalityVoiceDlg 為人格語音
- 未列出：鏡牢事件、鏡牢飾品、異想體名稱／觀察日誌、人格技能／E.G.O、敵方名稱／技能、但丁筆記

※ 註：多數檔案 01～09 罪人編號與遊戲一致，10 則非但丁而是辛克萊 → 11 奧提斯 → 12 格里高爾

- 戰鬥泡泡＊（全數泡泡在同一個檔案內）
  battle_(s1～s3 及各種狀態) 1**XXYY**，**XX** 表罪人編號，**YY** 表人格
  建議在 BattleSpeechBubbleDlg 檔案內直接搜尋 1**XX**

### 資料補充

> 依據：官方 2026-10-29 KR 語言包

**StoryData 檔名**（共 994 檔）：

| 規則 | 檔數 | 例 | 備註 |
|---|---|---|---|
| `S` + 3～4 碼 | 409 | `S001A`、`S1037B` | 主線；第 10 章起為 4 碼（`S10XX`） |
| `ZDYXX` | 147 | `1D101A`、`5D302B` | 主線地牢 |
| `P1YYXX` | 180 | `P10102`、`P10805` | 人格故事；每位罪人 14～16 檔 |
| `E` + 3 碼 | 202 | `E000X`、`E003I1` | 間章／活動 |
| `PC` + 2 碼 | 19 | `PC01A`、`PC14B` | **原指引未列**。《明日方舟》聯動劇情 |
| `ES` + 3 碼 | 37 | `ES000B`、`ES120A` | **原指引未列**。第八章的特殊劇情模式（場景如「大觀園 特殊教室」） |

**檔名代碼與遊戲內編號**：資料確認了原指引的註記（遊戲內編號見 `ScenarioModelCodes` 的 `nickName`）；對照已移至 `guides/families/identity.md`「文本類型」。

**戰鬥泡泡**：已不是單一檔案，目前有 11 個檔：

- `BattleSpeechBubbleDlg`（817 條）沿用原指引的編號：`battle_<類型>_1XXYY_<序號>`，類型有 `s1`～`s3`、`s3-2`、`react`、`special`、`eE` 等。例：`battle_s3_10617_1_1` = 鴻璐（06）第 17 號人格
- 各賽季另有分檔：`BattleSpeechBubbleDlg-a1c9p1`～`-a1c10p3`、`-exme`、`-twth`、`_Cultivation`、`_mowe`。章節分檔改用 `10SV-051_1` 這種編號（`<章>SV-<序號>`）
- `desc` 不翻、批次沒有說話者時怎麼判斷：已移至 `guides/families/identity.md`

**原指引「未列出」的文本**（檔數；歸哪個族群見下一節，由節點決定，不看這張表）：

| 文本 | 檔名樣式 | 檔數 |
|---|---|---|
| 鏡牢事件 | `AbEvents_Mirror*`、`ActionEvents_Mirror*`、`MirrorDungeon*`、`DungeonStartBuffs*` | 約 70 |
| 鏡牢飾品 | `EGOgift_*`（如 `EGOgift_MirrorDungeon-StoryTheme_2`） | 29 |
| 異想體名稱／觀察日誌 | `AbnormalityGuides*`、`AbEvents*`、`AbEventsResultLog*`；`AbDlg_<罪人>` 是罪人對異想體的台詞 | 約 91 |
| 人格技能／E.G.O | `Skills_personality-YY`、`Skills_Ego_Personality-YY`、`Passive_Ego*`、`Egos*`、`Personality*` | 各 14 份技能檔＋其他 |
| 敵方名稱／技能 | `Enemies*`、`Skills_Enemy*`、`Passives_Enemy*`、`Skills_Abnormality*`、`Passives_Abnormality*` | 約 200 |
| 但丁筆記 | `StoryTheaterDanteNote`、`StoryTheaterDanteNoteDetail_<章>` | 26 |
| 瓦夜活動檔 | 檔名含 `walpu`（如 `AbEvents-walpu6`） | 47 |
| 折射軌道活動檔 | 檔名含 `Refraction`、`Railway`（如 `RailwayDungeonUI_5`） | 81 |
| RPG 探索 | `RPGSystem/rpg-loc-*`（對話、NPC、選項、旁白） | 189 |
| 協助者 | `Skills_Assist*`、`Passives_Assist*` | 14 |

**RPG 探索的對話**、**RPG 標籤的括號**、**RPG 的樓層寫法**、**RPG 角色口癖**、**RPG 樓層變體要一致**：已移至 `guides/families/rpg.md`

**人格語音的分類標籤**：已移至 `guides/families/identity.md`「格式慣例」

**檔案集合**：輸出以 KR 為準（已決定）。KR 比 EN 多 53 個檔（例如 `Passives-a1c9p1～p3`），這些檔沒有 EN／JP 可參考，只依 KR 翻譯。只有 EN 有的 10 個檔不輸出。

### 待確認

- 原指引寫「全數泡泡在同一個檔案內」，現在已分成 11 個檔，章節檔改用 `<章>SV-<序號>` 編號；原指引需更新

## 族群與節點

族群由分群樹的節點組成（`guides/routing.yaml` 的 `nodes`）。檔名推導出 `top`、`sub` 兩層節點，先看 `sub` 再看 `top`；同一層寫出完整名稱的族群優先，否則取路由裡第一個樣式符合的族群；都沒有的檔歸 `unsorted`。瓦夜與折射軌道活動檔不再自成一個族群，依文字類型歸進下面的族群。各族群的節點以 `guides/routing.yaml` 為準；各族群送給譯者的指引是 `guides/families/<族群 id>.md`。

| 族群 | 文字類型 | 詞彙表 |
|---|---|---|
| `story-main` | 主線劇情、主線地牢、歌詞 | warn |
| `story-interlude` | 間章與活動劇情、聯動劇情 | warn |
| `identity` | 人格故事、人格語音、戰鬥泡泡 | warn |
| `ego-voice` | E.G.O 台詞 | warn |
| `announcer` | 播報員台詞 | warn |
| `dante-note` | 但丁的筆記 | warn |
| `rpg` | RPG 探索 | hard |
| `ally` | 我方：人格與 E.G.O 的技能、被動、名稱，協助者 | hard |
| `enemy` | 敵方：敵人與異想體戰鬥的名稱、技能、被動 | hard |
| `keywords` | 關鍵字、狀態、戰鬥術語 | hard |
| `abnormality` | 異想體圖鑑、觀察日誌、異想體事件 | warn |
| `mirror` | 鏡牢專屬：飾品、事件、起始增益 | hard |
| `railway` | 折射軌道 | hard |
| `stage` | 關卡、地牢區域、選擇事件、角色與場景名稱 | hard |
| `events` | 活動 | hard |
| `ui` | 介面、教學、商店、公告 | hard |
| `unsorted` | 後備：沒有族群列到的新檔 | hard |

## 凍結項目

- 目前沒有凍結的檔案（`lct.yaml` 的 `frozen` 為空）
- `BgmLyrics/*`：原指引註明「此項目已施工完畢，暫時禁止更動」，第一階段因此凍結；2026-11-02 由人工明確解除，改走一般流程（既有譯文的 KR 沒變就沿用，修改經由人工調整）

## 對於既有文本翻譯優先序

### 原指引

1. 主線劇情
   - 主線
   - 歌詞
   - 人物事件
   - 地牢故事＋事件
2. 間章
3. E.G.O 台詞（含我方／敵方）
4. 播報員台詞
5. 人格卡
   - 人格語音台詞
   - 人格故事
7. 我方
   - 人格技能名稱
   - 人格 E.G.O 名稱
8. 敵方
   - 敵人名稱
   - 敵人技能名稱
6. 異想體圖鑑
9. 但丁的筆記
10. 鏡牢
    - 飾品名稱
    - 觸發事件
11. 瓦夜／折射軌道

### 決定

- **照列出的順序**，不照編號。實際順序（`guides/routing.yaml` 的 `priority`）：`story-main` → `story-interlude` → `ego-voice` → `announcer` → `identity` → `ally` → `enemy` → `keywords` → `abnormality` → `dante-note` → `mirror` → `railway` → `rpg` → `stage` → `events` → `ui` → `unsorted`
- 自動化中的用途：當週待翻量很大、需要分批時，決定先翻哪些族群；第二階段複查或改進舊譯文時，決定先處理哪些族群
