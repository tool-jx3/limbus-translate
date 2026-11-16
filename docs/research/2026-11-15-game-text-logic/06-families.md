# 遊戲自己怎麼分文字檔，與專案族群的比較

2026-11-16。問題：`guides/routing.yaml` 現在的族群（分類）需不需要調整。

依據：`TextDataSet.LoadRemote`（0x13C2C40）、`TextDataSet.LoadLocal`（0x13C27B0）、`TextDataSet.RomoteLocalizeFileList`、各 `TextData_*` 類別的欄位、讀取各張表的函式（2026-11-12 版）；資料是 `raw-20261112` 的 2,449 個 KR 檔與 `tw` 分支 7fccf1d。「確認」＝對照反編譯或反組譯，或是資料的實際數字；「推定」＝有依據但沒有直接讀到。

限制：官方的檔案清單在資源檔裡讀不到，所以「哪個檔在哪張表」是用**條目的欄位對照類別的欄位、檔名對照清單名稱**推出來的；其中約 80 個節點另外用「查這些 id 的程式讀哪張表」確認過。

## 先講結論

- **不需要重劃。** 族群是依「內容與語氣」分的，遊戲是依「資料表」分的，目的不同。大部分不一致的地方不造成問題。
- **有五個檔案（節點）放錯族群**，建議改：`Skills.json`、`Passive-*`、三個 `Passives-*`、`Passives_fools`、`ChoiceEventKeyword`（第 1 節）。影響不大，因為 `ally` 與 `enemy` 的指引幾乎相同，差別只在翻譯順序與下面這一點。
- **比搬檔案更有用的是改「不一致的譯法」的分組方式。** 它現在以「族群＋欄位＋KR」分組；遊戲的一張表被切成兩個族群時，同一句 KR 在兩邊的譯法不會互相比較。現況有 **43 句**同欄位、同 KR、兩個族群各自一致但彼此不同，目前看不到（第 4 節）。
- `identity` 裡的戰鬥氣泡有一半是敵人與 NPC 的台詞，指引裡「從 id 的 `1XXYY` 看是哪位罪人」對它們不適用。建議補指引，不建議搬（第 3 節）。
- 順帶發現：約 2,000 個已翻譯的單位在**遊戲不讀的欄位**裡（第 5 節）。

## 族群在專案裡決定什麼

要判斷「分得對不對」，先看分類實際影響什麼：

| 用途 | 位置 |
|---|---|
| 套用哪一份族群指引與共用指引 | `rules.GuidesFor` |
| 詞彙表是硬性還是只警告（`glossary: warn`） | `rules.GlossaryWarnOnly` |
| 「不一致的譯法」的分組（族群＋欄位＋KR） | `internal/build`、`internal/batch` |
| 待譯項目的發放順序、複查的排序 | `priority` |
| 網站的檔案排序、抽樣檢查（`rules/gold.yaml`）每族群 20 筆 | `internal/site`、`lct gold` |

依欄位做的檢查（關鍵字、分頁、佔位符等）**不看族群**，看的是檔名與欄位，所以族群放錯不會讓檢查漏掉。

## 遊戲怎麼分

- 遊戲有 **77 份清單，一份清單對一張表**（`TextDataSet.LoadRemote`，確認）。沒有任何一張表由兩份清單組成。另有啟動時載入的 4 份清單、依劇情 id 開檔的劇情載入器、RPG 模式自己的載入器。
- 一張表裡的檔案怎麼切（`Skills_Enemy-a1c9p1`、`Skills_personality-03`……）只是檔名，**程式看不到**。
- 所以遊戲眼中：
  - **技能只有一張表**、**被動只有一張表**。罪人、E.G.O、敵人、異想體、協助者的技能都在同一張表，同一個讀取函式，同樣的關鍵字處理（確認）。
  - 罪人以外的單位名稱是一張表：敵人、異想體、部位，**以及協助者**（`Assist*`；`IncontrollableUnitStaticData.GetName` 0x1861CB0，確認）。
  - 狀態（`Bufs*`）與關鍵字說明（`BattleKeywords*`）是兩張平行的表，用同一組 id（1,924 個共用，名稱有 1,910 個相同）。
  - 類別是 `TextData_UI` 的 154 個檔是同一張介面文字表。
  - 所有劇情檔（`S`、`D`、`E`、`ES`、`PC`、`P`、`RPG`）走同一個載入器，程式不看檔名的字母。

各族群對應到幾張表：

| 族群 | 檔數 | 遊戲的表 | 判斷 |
|---|---|---|---|
| `story-main` | 591 | 劇情載入器（`S`、`D`、`RPG`）、歌詞表、主線地牢節點表 | 可以維持；見第 3 節 |
| `story-interlude` | 260 | 劇情載入器（`E`、`ES`、`PC`） | 沒問題 |
| `identity` | 393 | 劇情載入器（`P`）、人格語音表、戰鬥氣泡表 | 補指引；見第 3 節 |
| `ego-voice` | 14 | E.G.O 語音表 | 沒問題 |
| `announcer` | 67 | 播報員語音表 | 沒問題 |
| `dante-note` | 26 | 但丁筆記內文表、分類名稱表 | 沒問題 |
| `rpg` | 189 | RPG 載入器 | 沒問題 |
| `ally` | 81 | 技能表與被動表的一部分、人格、E.G.O、協助者名稱（在敵人名稱表）、取得條件 | 有放錯的檔；見第 1 節 |
| `enemy` | 208 | 技能表與被動表的另一部分、敵人名稱表 | 有放錯的檔；見第 1 節 |
| `keywords` | 157 | 9 張表：狀態、關鍵字說明、技能標籤、狀態能力、恐慌、精神狀態、屬性…… | 維持；見第 3 節 |
| `abnormality` | 92 | 異想體圖鑑、異想體事件、事件中的罪人台詞、介面文字表的一部分 | 沒問題 |
| `mirror` | 97 | 飾品、地牢選擇事件、起始增益、主題、介面文字表的一部分 | 可調可不調；見第 3 節 |
| `railway` | 26 | 鐵道的四張表與介面文字表的一部分 | 沒問題 |
| `stage` | 97 | 24 張表：關卡、章節、地牢區域、提示、選擇事件結果、說話者名稱…… | 沒問題 |
| `events` | 32 | 介面文字表的一部分 | 沒問題 |
| `ui` | 119 | 介面文字表的大部分與十多張小表 | 沒問題 |

## 1. 放錯族群的檔案（建議調整）

擁有者的判斷方式：技能、被動的 id 去掉末兩位就是單位 id，拿去對 `Personalities*`、`Enemies*`、`Assist*`。數字都是資料的實際計算（確認）。

| 檔案 | 現在 | 內容 | 建議 |
|---|---|---|---|
| `Skills.json` | `enemy` | 265 筆裡 **258 筆是罪人人格的技能**（例如 `1010101`、`1010203`），另有防禦、迴避、反擊 3 筆通用技能 | 改到 `ally` |
| `Passive-a1c971.json`、`Passive-walpu2.json`（節點 `Passive`） | `ally` | 35 筆裡 34 筆是敵人、異想體的被動，沒有罪人的 | 改到 `enemy` |
| `Passives-a1c9116.json`、`Passives-ycgd.json`、`Passives-BossRaid.json` | `ally` | 14、5 筆是敵人的被動；34 筆是頭目戰的被動 | 改到 `enemy`（要先讓路由能指定個別檔案，見下） |
| `Passives_fools.json` | `ally` | 3 筆都屬於敵方單位 8999；同一個單位的技能 `Skills_fools.json` 在 `enemy` | 改到 `enemy` |
| `ChoiceEventKeyword.json` | `keywords` | 與 `ChoiceEventEffect*`、`ChoiceEventTarget`（都在 `stage`）是同一張表，內容是拼進選擇事件結果句子裡的片段（推定） | 改到 `stage` |

為什麼會放錯：路由是用檔名推出的節點來比對的。

- `Skills.json` 沒有子節點，落到上層節點 `Skills`，而 `Skills` 列在 `enemy`（原意是接住 `Skills_Tutorial`、`Skills_fools`）。
- `ally` 列了節點 `Passive` 與 `Passives`，原意是 `Passive_Ego*` 與 `Passives.json`，但 `Passive-*`、`Passives-*` 的節點也是這兩個。

怎麼改：

- 前兩項與後兩項改 `guides/routing.yaml` 就可以：`ally` 加上 `Skills`、`Passive_Ego`；`enemy` 加上 `Skills_Tutorial`、`Skills_fools`、`Passive`、`Passives_fools`；`stage` 加上 `ChoiceEventKeyword`，並從原本的族群移掉。
- 第三項改不了：`Passives.json`（罪人）與 `Passives-a1c9116.json`（敵人）的節點都是 `Passives`，節點分不開。要嘛讓路由能寫個別檔名，要嘛接受（53 筆）。

實際影響有限：`ally.md` 與 `enemy.md` 的內容幾乎相同，共用指引與詞彙表的約束也相同。會變的是翻譯順序（`ally` 排在 `enemy` 前面）與不一致比較的分組。

## 2. 同一張表被切到不同族群

| 表 | 族群 | 對遊戲有沒有差別 | 跨族群的相同 KR | 判斷 |
|---|---|---|---|---|
| 技能表、被動表 | `ally`／`enemy` | 完全沒有 | 同欄位 441 句，其中 23 句兩邊譯法不同 | 族群可以維持（我方與敵方分開翻有道理），但不一致比較應該跨過這條線（第 4 節） |
| 敵人名稱表 | `enemy`（`Enemies*`）／`ally`（`Assist*`） | 沒有 | 極少 | 維持。協助者是我方單位，技能與被動也在 `ally` |
| 介面文字表 | `ui`、`events`、`stage`、`mirror`、`abnormality`、`keywords`、`railway` | 載入與格式化都相同 | `mirror`↔`stage` 93 句（人格故事標題） | 維持。這張表本來就是雜項，依內容分比依表分合理 |
| 精神狀態表 | `keywords`（`MentalCondition*`）／`stage`（`reward-popup-text-ProjectGS`） | 規則相同：`{n}` 的個數要完全正確 | 無 | 維持。後者是 Project GS 的成長結果文字，放 `stage` 合內容 |
| 劇情載入器 | `story-main`／`story-interlude`／`identity` | 沒有 | `story-main`↔`story-interlude` 48 句，8 句譯法不同 | 維持 |

## 3. 一個族群裡混了處理方式不同的表

依欄位的檢查不看族群，所以這一類的問題只在**指引寫的內容是否適用**。

**`identity`：三種機制**

| | 人格故事 `StoryData/P` | 人格語音 `PersonalityVoiceDlg/` | 戰鬥氣泡 `BattleSpeechBubbleDlg*` |
|---|---|---|---|
| 取代方式 | 整個檔，列依位置配對 | 整個檔 | 以 id 為單位 |
| ruby | 可以 | 不行 | 不行 |
| 顯示 | 劇情對話框，逐字顯示 | 大廳、結算、抽卡，以及戰鬥氣泡 | 戰鬥氣泡，整句出現，時間跟語音走 |
| 說話者 | 罪人與劇情角色 | 罪人 | **1,555 筆裡 737 筆是敵人、NPC 的台詞**（`<章>SV-…`） |

- 十個分章、分活動的氣泡檔（`-a1c9p1`～`-a1c10p3`、`-exme`、`-twth`、`-walpu10`、`_mowe`）**全部是敵人與 NPC 的台詞**，只有 `-a1c10p2` 有 19 筆罪人的。`_Cultivation` 有 23 筆協助者的台詞。
- 程式不分罪人與敵人，id 只是語音事件的名稱。
- `identity.md` 寫「看 `desc` 與 id 的 `1XXYY` 判斷是誰在說話」。`1XXYY` 對 `<章>SV-…` 不適用，這些只能看 `desc`（例如「베스파 9스킬」「잔느 2페이즈」）；指引也沒有提到這些台詞是敵人與 NPC 的，而且常常就是同一章劇情裡的句子（`9SV-BAT17-26-3` 就是）。
- 建議：**不搬族群**（節點分不開基本檔與分章檔，而且語氣指引對兩者都成立），在 `identity.md` 補一段分章氣泡檔的說明：說話者看 `desc`、台詞要對照同一章的劇情譯文。跨表的比對交給第 4 節。

**`story-main`：劇情之外還有兩張表**

- `BgmLyrics/`：整個檔取代；ruby 只在劇情裡有效，戰鬥中會原樣顯示（[02-tags-ruby.md](02-tags-ruby.md)）。族群不用動，已有對應的檢查。
- `DungeonNode*`：以 id 為單位的節點資訊表，和 `RailwayDungeonNode*`（`railway`）顯示在同一個面板、與 `StageNode*`（`stage`）是同類。內容是主線地牢的節點名稱與敘述。可以搬到 `stage`，也可以不搬；只有 17 個檔、18 筆，影響很小。

**`mirror`：`ActionEvents*` 不只是鏡牢的**

- `ActionEvents*`（40 個檔）是**所有地牢的選擇事件**，包含主線地牢（`ActionEvents-a1c…`）與各活動，分頁規則與 `abnormality` 的 `AbEvents*` 相同。
- `mirror` 套用戰鬥說明共用指引、詞彙表是硬性；`battle-text.md` 已經為 `ActionEvents` 的敘述欄位寫了例外（「不是效果文字」）。這表示它其實不太屬於這個族群。
- 可以搬到 `abnormality`（同樣是選擇事件、詞彙表只警告），也可以維持現狀靠那條例外。搬的話 `rules/declined.yaml` 裡 6 筆 `inconsistent/mirror/…` 要檢查。

**`keywords`：9 張表、5 種規則**

- `Bufs` 的 `desc`／`summary` 是格式字串；`BattleKeywords` 是原樣顯示的提示文字；`PanicInfo` 有關鍵字連結但沒有標籤處理；`MentalCondition` 的 `{n}` 個數要完全正確；`SkillTag` 的名稱會被插進技能文字。
- 這些差別都是欄位層級的，由檢查處理，不是族群能解決的。維持。
- `UnitKeyword*`、`ResistText` 在遊戲裡只是介面文字表的普通字串，但內容是關鍵字名稱，放這裡合理。

**`ally`：兩個不是戰鬥文字的檔**

- `Personality_Get_Condition`、`EGO_Get_Condition` 是大廳提示框的句子（「如何取得」），不是技能文字，卻套用戰鬥說明指引。可以搬到 `ui`；影響小。

**`ego-voice`**

- `Voice_EGO_Enemy_999`（50 筆）是敵人、頭目的台詞，由演出字幕讀取，優先於氣泡文字（[05](05-battle-speech-bubble.md)）。`ego-voice.md` 已經把它列在內，維持。

## 4. 不一致比較的分組（比搬檔案更有效）

現在的分組是「族群＋欄位＋KR」。問題不在族群分錯，而在**相同的句子本來就會跨族群出現**：

- 同一句 KR（8 字以上）出現在兩個以上族群的有 1,044 句。
- 同欄位、同 KR、出現在兩個族群的有 801 組（族群兩兩計算）。其中 **43 組是兩邊各自只有一種譯法、但彼此不同**，所以兩邊的比較都不成立：

| 族群 | 共用的句子 | 譯法不同而看不到 | 例 |
|---|---|---|---|
| `ally`↔`enemy` | 441 | 23 | 技能名「전입 신고 처리」：處理小鎮移居申請／處理小鎮新移居申請 |
| `story-main`↔`story-interlude` | 48 | 8 | 多半是語氣不同的短句 |
| `enemy`↔`keywords` | 32 | 2 | 被動名與狀態名「잊혀진 기사의 검」：被忘卻的騎士之劍／被遺忘的騎士之劍 |
| `identity`↔`story-interlude` | 8 | 2 | |
| 其他八組 | | 各 1 | 例：人格稱號與人格故事標題 |

- 另外還有**跨欄位**的：劇情 `content` 與氣泡 `dlg`（9 句相同）、與人格語音 `dlg`（8 句），其中 5 句譯法不同。`9SV-BAT17-26-3` 屬於這一種。

建議（第 1、2 項已在同日做成警告 `cross-group`：KR 8 字以上，現況 67 句、234 個單位）：

1. 加一層**跨族群**的比較：同欄位、同 KR、在不同族群的譯法不同就列出來。現況會多 43 組，量很小。
2. 台詞類再加**跨欄位**的比較：劇情 `content`、氣泡 `dlg`、人格語音 `dlg`、E.G.O 語音 `dlg` 視為同一類。現況多 5 組左右。
3. 不要直接改現有的分組鍵：`rules/declined.yaml` 有 302 筆 id 是 `inconsistent/<族群>/<欄位>/…`，鍵一改就全部失效。新的比較用新的種類名稱另外列。

## 5. 順帶發現（不是分類問題）

**遊戲不讀的欄位**

官方檔案裡有些 key 不是對應類別的欄位，所以不會被載入、不會顯示（類別的欄位是確認；Unity 的 `JsonUtility` 會忽略多出來的 key 是推定）：

| 欄位 | 已有翻譯紀錄的單位 |
|---|---|
| `BattleKeywords*` 的 `summary` | 1,456 |
| `AbEvents*` 的 `title` | 321 |
| `Personalities*` 的 `nameWithTitle` | 209 |
| `ActionEvents*` 的 `title`、`prevDesc` | 12 |
| `AbnormalityGuides*` 的 `desc` | 9 |
| `BuffAbilities` 的 `name`、`summary` | 4 |

翻譯它們沒有壞處，但不會出現在遊戲裡。可以考慮在 `rules/` 的欄位規則裡把這些欄位排到最後或不送翻譯。`AbEvents.title` 與 `ActionEvents.name` 有 30 句相同，是 `abnormality`↔`mirror` 之間相同句子的主要來源，而前者根本不顯示。

**同一張表裡 id 相同、文字不同的檔**

- `DungeonStartBuffs.json`、`_MD6`、`_MD7`：44 個 id 重複，40 個的文字不同（例如「(최대 100)」與「(최대 {1})」）。
- `UserTicket-{L,R,EGOBg}-a1c10p1` 與 `-a1c10p3`：各 3 個 id（「???」與真名）。
- `Announcer-a1c10p1` 與 `-a1c10p3`、`Announcer.json`：5 個 id。

官方清單實際列了哪一個讀不到。語言包是後列的檔蓋掉先列的（[01-custom-language.md](01-custom-language.md)），所以這些 id 在每個檔裡的譯文最好一致，或至少知道它們互相會蓋。

## 已做的調整（2026-11-16）

- `guides/routing.yaml`：`Skills.json` 改歸 `ally`；`Passive-a1c971`、`Passive-walpu2`、`Passives_fools` 改歸 `enemy`；`ChoiceEventKeyword` 改歸 `stage`。只有這五個檔換族群。三個 `Passives-*` 沒有動。
- 換族群後「不一致的譯法」的組會跟著變：`enemy` 少 12 組、`ally` 多 17 組。少掉的 12 組裡有 5 組是先前確認保留的（`견제`、`방어`、`연속 베기`、`타격`、`후비기`），它們在 `ally` 以新的 id 出現，已在 `rules/declined.yaml` 補上同樣的決定；舊的 5 筆留著。其餘是新出現、要看的建議。
- 過渡期：網站的建議清單來自上一次建置。下一次建置之前，網站匯出會回報 15 個「建議與單位對不上」的組（`review_mismatches`），建置後消失。
- 既有譯文、其他警告、`rules/accepted.yaml`、`rules/gold.yaml` 都不受影響；`ally.md`、`enemy.md`、`stage.md` 的檔案清單已更新。

## 調整要付的代價

- 檔案換族群後，`rules/declined.yaml` 裡以舊族群命名的 `inconsistent/…` id 不再對得上（`ally` 2 筆、`enemy` 14 筆、`mirror` 6 筆、`stage` 2 筆；`keywords` 沒有）。搬之前要先看這些 id 指的是不是被搬的檔。
- `rules/gold.yaml` 的抽樣是每族群 20 筆，紀錄裡寫了族群。
- 族群指引（`ally.md`、`enemy.md`、`identity.md`）裡寫的檔案清單與統計數字要跟著改。
- 節點是從檔名推的，同節點的檔分不開（`Passives-*`、`BattleSpeechBubbleDlg-*`）。要分就得讓 `routing.yaml` 能寫個別檔名或樣式，這是程式的修改。

## 讀不到、沒有確認的部分

- 官方三份檔案清單的內容：是不是每個 KR 檔都有被列進去、同一張表裡檔案的先後、id 重複的檔哪一個有效。
- `SynergyKeyword`、`ChapterBannerConfig`（空的）與四個 `Tutorial*` 檔屬於哪張表；鏡牢節點表（`MirrorDungeonNode1-1`、`HellsChickenDungeonNode`）與 `LobbyBGMCategory` 由誰讀。
- `ChoiceEventKeyword`、`ChoiceEventTarget` 屬於選擇事件結果表，是從「讀這張表的類別名稱就是這兩個檔的 id」推的，沒有逐行確認。
- 介面文字表 98 個節點裡有 32 個是只憑類別與檔名歸進去的。
