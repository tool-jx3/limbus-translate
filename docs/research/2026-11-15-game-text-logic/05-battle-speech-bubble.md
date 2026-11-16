# 戰鬥氣泡文字（`BattleSpeechBubbleDlg*.json`）

依據：`TextData_BattleSpeechBubble`、`VoiceGenerator`、`Battle.Voice.VoiceTable`、`BattleUI.BattleUnit.BattleDialogUI`、`BattleUI.BattleCenterSubtitleUI`、`OutterGradiantEffectController`、`ProduceBase_ShowVoice`／`ProduceBase_ShowVoiceMulti`，以及七個直接取用的戰鬥腳本（2026-11-12 版）。「確認」＝對照反組譯讀過；「推定」＝有依據但沒有直接讀到；位址是 RVA。資料：`raw-20261112` 與 `tw` 分支 7fccf1d，12 個檔、每種語言 1,555 筆。

## 先講結論

- **只有 `dlg` 會顯示**。`desc` 是開發者的註記（講者、情境），沒有任何程式讀它；`id` 是**語音事件的名稱**，遊戲用完全相同的字串查，不能改。
- **文字原樣送上畫面**：一筆就是一句，一次整句出現。沒有 `{0}` 代入、沒有取代、沒有切割、沒有去頭尾空白、沒有逐字顯示、沒有關鍵字處理、沒有 ruby 轉換。
- 這份文字會出現在三種畫面元件：**角色頭上的氣泡**（絕大多數）、**畫面中央字幕**（只有一場頭目戰）、**演出時畫面下方／中間的字幕**（技能、E.G.O、頭目演出）。
- **顯示時間與文字長度無關**：有語音的句子是語音長度（最少 3 秒），沒有語音的腳本台詞固定 6 秒。譯文比原文長，也不會多留在畫面上。
- 氣泡的寬度、自動換行、字級、溢出方式**程式裡完全沒有**，都是 prefab 的設定，讀不到。程式不會插入換行，唯一由譯文決定的換行是 `dlg` 裡的 `\n`。
- 整條路徑**沒有語言分支**：不管玩家在遊戲內選 KR、EN 或 JP，中文的處理都一樣。
- 同一個氣泡也顯示 `PersonalityVoiceDlg` 的戰鬥語音台詞（擊殺、混亂、死亡等），所以那些句子適用同一套規則。
- 可以用的標籤：官方在這份文字裡用過 `<i>`、`<color=#rrggbb>`、`<size=NN%>`（最後一個只有 JP 用）。

## 資料與查找

### 一筆條目

`TextData_BattleSpeechBubble`：`id`、`desc`、`dlg` 三個字串欄位。

- `GetDialog()` 與 `GetText(object[] args)` 是同一段程式（0x86A240），直接回傳 `dlg`；**參數被忽略，不做 `string.Format`**（確認）。所以大括號沒有意義，寫了會原樣顯示。
- 讀這份表的程式一共 **12 個**（`method/who.py slot 0x9217E98`，確認），全部只讀 `dlg`。沒有任何一個讀 `desc`。
- 載入方式與其他文字表相同（[01-custom-language.md](01-custom-language.md)）：英文為底，語言包以 id 為單位整筆取代。語言包沒有的 id 顯示英文。

### id 是什麼

id 不是由氣泡的程式組出來的，而是**呼叫者播放的語音事件名稱**：

| 來源 | 說明 |
|---|---|
| FMOD 語音事件的檔名 | `VoiceGenerator.SetMainVoice`（0x17725E0）、`AddMultiVoice`（0x1773580）取事件路徑的最後一段當 id（確認） |
| 時間軸上的字幕 id | `FMODVoiceSubtitleTrack._subtitleId`，有值時取代事件檔名（確認）；實際的值在資源檔裡 |
| 腳本裡寫死的字串 | 1,555 個 KR id 裡有 521 個以完整字串出現在程式裡，另有 `6SV-0{0}`、`battle_react_10613_{0}`、`battle_special_{0}_{1}` 這類格式字串 |

- 變體的選擇（`_1`、`_2`、`-01`、`-02`，隨機或輪流）在**呼叫者**那邊做完，氣泡的程式只做一對一的完全比對。**沒有「從多筆裡隨機抽一筆」，`dlg` 裡也沒有分隔符號**（確認）。
- 資料裡有 218 個 id 是另一個 id 的 `<原 id>-0N` 變體（同一句的另一個錄音版本），其中 140 個的 KR `dlg` 與原 id 完全相同。
- id 的種類（KR）：`<章>SV-…` 737（敵人、NPC）、`battle_s<n>…` 375（技能台詞）、`battle_special…` 332、`battle_react…` 65、`battle_eE…` 22，其餘零星。

### 語音帶出氣泡的流程

`VoiceGenerator.PrintSpeechBubble(unitId, voiceId, length, useCustomLength)`（0x1777B70，確認）依序檢查，任何一項不成立就不顯示：

1. 場景是戰鬥或 RPG（程式比對的是數值 1 與 9；名稱依 enum 宣告順序推定）。
2. 玩家的遊戲設定「戰鬥台詞」（`_battleSpeech`）是開的。
3. 語音表（`etc/VoiceTable.json`，遊戲本體資料，不在語言包裡）存在。表裡有一個前綴比對，但依編譯出來的程式它永遠成立，等於沒有過濾（程式的形狀是確認，實際效果是推定）。
4. 找得到這個單位在場上的顯示物件。
5. 取到的文字不是空的。

取文字的是 `VoiceTable.GetVoiceTextData_SpeachBubble(unitId, voiceKey)`（0x2BEFDD0，確認）：

- `BattleSpeechBubbleDlg` 有這個 id → 用它的 `dlg`（即使是空字串）。
- 沒有 → 到 **`PersonalityVoiceDlg` 裡這個單位自己的檔**找同一個 id（先完全相同，再不分大小寫），用它的 `dlg`。
- 都沒有 → 不顯示。

官方資料裡這兩份表的 id 完全不重疊，所以沒有誰蓋掉誰的問題。

### 不經語音、腳本直接顯示的台詞

七個腳本用寫死的 id 直接取 `dlg` 顯示（確認）。這些句子**沒有語音、固定顯示 6 秒、不受「戰鬥台詞」設定影響**：

| id | 取用的腳本 |
|---|---|
| `battle_speechbubble_cromer_1` | `StageScript_Cromer.OnRoundStart` |
| `battle_speechbubble_10808_1` | `StageScript_8106` |
| `battle_speechbubble_10408_1` | `StageScript_8171.OnRoundStart` |
| `battle_1137_FirstKill` | `UnitScript_1137`、`UnitScript_1141` |
| `10SV-100` | `UnitScript_1498.ShowSpeechBubble` |
| `battle_react_noVoice_10613_1`～`_5` | `UnitScript_10613.PlayReactVoice` |

### 缺 id、空字串、`Unknown`

| 情況 | 語音帶出的氣泡 | 腳本直接顯示的台詞 | 演出字幕 |
|---|---|---|---|
| 合併後的表裡沒有這個 id | 改用 `PersonalityVoiceDlg`；也沒有就不顯示 | **空的氣泡框顯示 6 秒**（`UnitScript_10613` 有檢查，不顯示） | 不顯示 |
| `dlg` 是空字串 | 不顯示（條目存在，所以也不會改用 `PersonalityVoiceDlg`） | 空的氣泡框顯示 6 秒 | 不顯示 |
| `dlg` 寫成 `Unknown` | 原樣顯示 | 原樣顯示 | 原樣顯示 |

- 這份文字**不會由遊戲產生 `Unknown`**：找不到 id 時拿到的預設物件，`dlg` 是 null 而不是 `Unknown`（確認）。這一點與技能、狀態等文字表不同。
- 語言包以 id 整筆取代英文，所以語言包**漏掉**一筆只是顯示英文；語言包**有這一筆但 `dlg` 是空的**，這句話會消失。
- 條目有 `id` 卻完全沒有 `dlg` 這個 key 時的結果是 Unity 原生程式的行為，沒有讀到。

## 三種顯示方式

### A. 角色頭上的氣泡（一般情況）

- 元件：`BattleUI.BattleUnit.BattleDialogUI` 的 `tmp_dialog`（`TextMeshProUGUI`），每個單位一個。入口是 `BattleUnitUIManager.ShowDialog`（0x29F6270）→ `BattleDialogUI.ShowDialog`（0x29E9320）。呼叫它的就是上面的語音流程與七個腳本（確認）。
- 什麼時候出現：**任何經 `VoiceGenerator` 播放、而且沒有要求關閉氣泡的戰鬥語音**。包括擊殺、混亂、使敵人混亂、死亡、同伴死亡、指令結束、選取、拼點落敗、時間軸上的技能語音，以及大量 `UnitScript_*` 在回合開始、使用技能、階段變化時播的語音（`PlayMultiVoice` 有 185 個呼叫者）。
- 不會出現氣泡的語音：編隊語音、進場語音（呼叫時指定不顯示氣泡）。

### B. 畫面中央字幕

- 元件：`BattleUI.BattleCenterSubtitleUI` 的 `_text`。只由 `StageScript_1522ProduceSystem` 載入（prefab 名稱 `Prefab_1524_JeanneCenterSubtitle`）。
- 條件：`VoiceGenerator.TryPrintCenterSubtitle`（0x1778090，確認）詢問單位的腳本要不要用中央字幕；唯一回答「要」的是 `UnitScript_1522`，而且只在它的某個狀態值是 2 或 3 時（那個值的意義沒有追到，推定是階段）。成立時文字改送到中央字幕，**頭上的氣泡不顯示**。
- 文字來源與 A 完全相同。

### C. 演出時畫面下方／中間的字幕

- 元件：`OutterGradiantEffectController` 的 `_dialogText`（下方）與 `_dialogText_Mid`（中間）。
- 來源：技能、E.G.O、頭目演出上掛的 `ProduceBase_ShowVoice` 或 `ProduceBase_ShowVoiceMulti`。它用語音事件的檔名**先查 `EGOVoiceDig`**（前者查編號 999 的清單，後者查元件上指定的清單），沒有非空的文字才查 **`BattleSpeechBubbleDlg`**（確認）。
- 用下方還是中間，是演出元件上的設定，讀不到。
- 官方資料裡有 6 個 id 兩邊都有（`10SV-067`、`9SV-BAT1-13`、`9SV-BAT14-02-1`～`-4`）；在這種字幕裡會以 `EGOVoiceDig` 的文字為準。

### 不會顯示這份文字的地方

- **技能演出的台詞框**（`BattleSkillViewUIInfo.text_script`）：只讀 `EGOVoiceDig`（確認）。
- 各種浮動文字特效（傷害、狀態、被動、飾品）。
- 程式裡還留著一套舊的「戰鬥對話」系統（`BattleUIDialogManager`，條件有第一回合、擊殺、恐慌、侵蝕等），已經沒有作用：讀資料的函式沒有人呼叫，取台詞的函式固定回傳空值（確認）。

## 文字怎麼上畫面

### 沒有任何加工

三種方式都是把 `dlg` 直接指定給 TextMeshPro 的 `text`（0x29E9E30、0x28F0450、0x2FD9310，確認）。沒有 `string.Format`、`Replace`、`Split`、`Trim`、`Substring`、正規表示式，也沒有關鍵字比對。**頭尾的空白與每一個字元都照寫的樣子送出去。**

唯一的例外：走 `PersonalityVoiceDlg` 那條路的文字會先經過 `Util.GetDlgAfterClearingAllCathy`（清除特定名字的處理），這是那份文字本來就有的。

### 整句一次出現

沒有逐字顯示。氣泡與中央字幕是整句淡入、停留、淡出（淡入淡出各 0.5 秒）；演出字幕是直接出現、時間到直接消失。所以標籤不會被切到一半。

### 顯示時間

氣泡與中央字幕（`BattleDialogUI.ShowDialog`，常數 3.0、6.0、0.5 都對照過，確認）：

```
顯示秒數 = 傳入長度 ≤ 0                    → 6
          傳入長度 > 3 或指定了自訂長度    → 傳入長度
          其他                             → 3
```

| 情況 | 傳入長度 | 實際顯示 |
|---|---|---|
| 有語音，沒有自訂長度 | 語音事件的長度 | 語音長度，最少 3 秒 |
| 有語音，呼叫者或時間軸指定了長度 | 指定的值 | 就是指定的值（短於 1.5 秒時淡入淡出跟著縮短） |
| 語音事件不存在、只給了替代 id | 2 | 3 秒 |
| 腳本直接顯示的台詞 | −1 | 6 秒 |
| 演出字幕 | — | 語音長度加 0.5～1 秒；部分寫法最少 5 秒或固定 6 秒 |

**文字多長都不影響時間。** 最短的顯示時間是 3 秒，所以很短的語音配上很長的譯文會讀不完。

### 同時與先後

- 每個單位一個氣泡。同一個單位來了新的一句，舊的那句先淡出、新的再淡入：**是覆蓋，不是排隊**。不同單位的氣泡可以同時出現。
- 單位死亡與戰鬥開始時會把氣泡收起來。
- 中央字幕與演出字幕各只有一個，新的一句直接取代舊的。

## 版面

### 氣泡：程式裡什麼都沒有

`BattleDialogUI` 整個類別沒有量文字寬高、沒有改字級、沒有開關自動縮字、沒有插入換行、沒有算字數、沒有長度上限，也沒有設定溢出、換行、對齊（確認）。所以**氣泡的寬度與上限、會不會自動換行、溢出時怎麼辦、字級、對齊、外框會不會跟著文字長大，全部是 prefab 的設定，讀不到。**

能推定的只有：

- 外框會跟著內容變高（推定）：程式在執行時讀氣泡的高度，用來把其他浮動文字往上推；而且官方資料同時有只有一個詞的句子與一百多字的句子。
- 程式不插入換行；換行只有 `dlg` 裡的 `\n`，加上元件本身設定的自動換行（設定讀不到）。
- 沒有依角色面向翻轉氣泡的程式，也沒有直排。

### 中央字幕：程式會量大小

`BattleCenterSubtitleUI.SetText`（0x28EFE70，確認）：

- 文字寬度取「實際需要的寬度」與「最大寬度減去左右留白」較小的一個，超過就換行。
- 換行後的高度超過「最大高度減去上下留白」時，打開 TextMeshPro 的自動縮字。
- 外框是文字大小加留白，不小於最小尺寸。

程式裡的預設值：最大寬 1400、最大高 400、留白 80×40、最小外框 600×120（確認；prefab 可以改掉這些值，讀不到）。照預設值，文字在 **1320** 寬換行、超過 **360** 高開始縮字。沒有字數上限，也不會截斷。

### 演出字幕

程式只設定顏色與材質，大小、換行都是 prefab 的設定。

### 沒有語言分支

上面提到的類別都沒有讀 `GlobalGameManager.Lang`（確認）。各語言之間如果有差別，只會來自 prefab 上掛的字型、字級元件（[04-other.md](04-other.md) 第 1 節）。

## 標籤與特殊字元

- 這條路徑沒有程式刪除、計算或跳脫標籤，也沒有關掉 rich text（[02-tags-ruby.md](02-tags-ruby.md)）。標籤原樣交給 TextMeshPro。
- `<ruby>` 在這裡不會被轉換（ruby 只在劇情對話、劇情紀錄、劇情字幕有效），會原樣顯示成文字。
- `[狀態 id]`、`[TabExplain]`、狀態名稱在這裡都不會被處理。
- `//`、`|`、兩個字元的「反斜線＋n」都沒有特別意義，會原樣顯示。

官方資料（每種語言 1,555 筆）：

| | KR | EN | JP | 現行譯文 |
|---|---|---|---|---|
| 有標籤的條目 | 4 | 14 | 5 | 4 |
| 用到的標籤 | `<i>`、`<color=#50acff>` | 同左 | 同左，另有 `<size=50%>`、`<size=75%>` | `<i>`、`<color=#50acff>` |
| 有 `\n` 的條目 | 12（都只有 1 個） | 55（最多 3 個） | 13 | 12 |
| `{`、`}`、`\|`、`//`、`\r`、tab、反斜線＋n | 0 | 0 | 0 | 0 |
| 空的 `dlg`、`Unknown` | 0 | 0 | 0 | 0 |
| 開頭／結尾有空白 | 2／4 | 1／6 | 0／0 | 1／1 |

- **`\n` 的數量在官方各語言之間並不一致**：EN 有 45 筆與 KR 不同，JP 有 1 筆。所以行數不是必須與 KR 相同的東西，譯文可以依長度自行決定要不要斷行。
- KR 有 `\n` 的 12 筆包含全部有換行的腳本台詞（`battle_speechbubble_cromer_1`、`battle_speechbubble_10808_1`、`battle_speechbubble_10408_1`、`battle_1137_FirstKill`）。

長度（看得到的字數，不含標籤與換行）：

| | 中位數 | 90% | 99% | 最長 |
|---|---|---|---|---|
| KR | 11 | 32 | 64 | 111 |
| EN | 29 | 80 | 159 | 241 |
| JP | 11 | 31 | 63 | 105 |
| 現行譯文 | 9 | 23 | 44 | 75 |

官方自己就有不換行的一百多字（KR、JP）與兩百多字（EN）的句子，所以氣泡容得下這個長度；它實際怎麼折行讀不到。**官方 CJK 的最大值 111 字可以當作安全範圍。**

## 相鄰的戰鬥文字

| 文字 | 顯示的地方 | 與氣泡的關係 |
|---|---|---|
| `PersonalityVoiceDlg/*` 的戰鬥語音（`battle_kill_…`、`battle_break_…`、`battle_enemy_break_…`、`battle_dead_…`、`battle_allydead_…`、`battle_endcommand_…`、`battle_select_…`） | **同一個氣泡**（以及中央字幕） | `BattleSpeechBubbleDlg` 沒有這個 id 時的後備。顯示規則完全相同。進場語音（`battleentry`）不顯示氣泡 |
| `BattleAnnouncerDlg/*` | `BattleAnnouncerUI`（播報員的對話框，有自己的捲動） | 不同元件。播報員的語音也會走到氣泡的流程，但播報員不是場上的單位，找不到顯示物件，所以沒有氣泡（推定） |
| `AbDlg_*`（判定事件裡罪人的對話框，地牢事件也有） | `AbnormalityChoiceDialogController` | 不同元件。判定之後先用 `PersonalityVoiceDlg` 的 `choice_…`，沒有才用 `AbDlg` |
| `EGOVoiceDig/*` | 技能演出的台詞框；演出字幕（優先於氣泡文字） | 不走氣泡 |
| `BgmLyrics/*` 在戰鬥中 | 主歌詞是另一個元件；「上方歌詞」用的是演出字幕同一個控制器的另一個文字物件 | 不同文字物件 |

## 讀不到、沒有確認的部分

- 氣泡與演出字幕在 prefab 裡的全部設定：寬度、自動換行、溢出方式、字級與自動縮字範圍、對齊、留白。中央字幕在 prefab 裡實際的數值（只知道程式的預設值）。
- `etc/VoiceTable.json` 的內容。
- FMOD 事件名稱與時間軸字幕 id 都在資源檔裡，所以**哪些 id 真的會被播到、哪些已經是用不到的資料**，無法從程式判斷。
- 誰呼叫 `ProduceBase_ShowVoice.SetDialog`（虛擬呼叫），以及每句語音走下方還是中間。
- `UnitScript_1522` 用來判斷中央字幕的狀態值代表什麼。
- 氣泡會不會隨角色面向左右翻轉。
- 以上都是靜態分析，沒有在遊戲裡看過實際的折行位置。

## 現行譯文的比對結果

12 個檔、1,555 筆，全部與 KR 比對。

沒有問題的：檔名、id 集合與順序、沒有重複 id、`dlg` 沒有空的、沒有 `Unknown`、沒有殘留的韓文或假名、`desc` 與 KR 相同（沒有 `desc` 的 12 筆也維持沒有）、沒有大括號、標籤與 KR 逐一相同（4 筆）、`\n` 數量與 KR 相同（12 筆）、十個腳本直接取用的 id 都在且不是空的、140 組「變體與原 id 的 KR 相同」的譯文也都相同、最長 75 字（低於官方 CJK 的 111）。

需要看的：

| 位置 | 內容 | 說明 |
|---|---|---|
| `BattleSpeechBubbleDlg-a1c9p1.json` 的 `9SV-BAT4-04` | 「小生的責任就此結束，該退場了。 」 | 結尾多一個半形空格（KR 沒有）。程式不會去掉，會照樣排進氣泡 |
| `BattleSpeechBubbleDlg.json` 的 `battle_s3_10514_1_1` | 「 粉塵灼燒。臨床實驗成功。」 | 開頭多一個半形空格（KR 沒有） |
| `BattleSpeechBubbleDlg-a1c9p3.json` 的 `9SV-BAT17-26-3` | 「沒有理由離開吧，女兒。因為妳遺忘了一切，也失去了一切。」 | **內容與現行原文對不上**。KR 是「미리 익숙해지는 게 낫지 않겠니.」，EN 是「May as well get used to it, right?」，JP 是「なら、慣れておいた方がいいだろう。」。是用長度比例找到的（譯文 27 字，KR 18、JP 17），前後幾句的譯文都對得上，只有這一句不同 |
| `PersonalityVoiceDlg/Voice_Rodion_Ncorp_10904.json` 的 `battle_endcommand_10904_1`、`battle_clear_10904_1` | 「 淨化！」、「 信仰的勝利。」 | 開頭多一個半形空格（KR 沒有）。前者會顯示在氣泡裡 |

`9SV-BAT17-26-3` 為什麼沒有出現在規則建議的「不一致的譯法」：那個比較以「類別＋欄位＋KR」分組（`internal/build`），只在同一組內比。這句 KR 在整個語言包只出現兩次，一次在劇情 `StoryData/S950I2.json` 的 `content`（類別 `story-main`，譯文「那麼早點習慣比較好吧。」），一次在這裡的 `dlg`（類別 `identity`），兩邊各自只有一種譯法，所以不成立。另一個方向（同一句譯文對應不同的 KR：這句譯文同時是 `9SV-BAT17-24`、`9SV-BAT17-24-01` 的譯文）原本沒有檢查，已加上警告 `shared-line`（範圍是氣泡與人格語音的 `dlg`，譯文 10 字以上）。粗估：氣泡、人格語音與劇情之間 KR 相同而譯文不同的有 5 句；氣泡檔內同一句譯文（10 字以上）對應不同 KR 的有 4 組，其中 3 組是 KR 只差敬語的正常情況。

`PersonalityVoiceDlg` 裡 id 以 `battle` 開頭的 1,987 筆用同樣的項目比過，除了上面兩處開頭空格以外，`\n` 數量、標籤、大括號、空字串都沒有差異。

## 建議的檢查

對象：`BattleSpeechBubbleDlg*.json` 的 `dlg`，以及 `PersonalityVoiceDlg/*.json` 裡 id 以 `battle` 開頭的 `dlg`。

**會讓句子消失或出現空框**

1. id 集合與 KR 相同，沒有重複、沒有空的 id。
2. `dlg` 不是空字串（KR 是空的除外）。腳本直接取用的十個 id 尤其重要：空的會出現 6 秒的空氣泡框。

**會顯示出不該有的字**

3. 沒有 `{`、`}`；沒有 `<ruby`；沒有「反斜線＋n」、`\r`、tab。
4. 標籤只用官方在這份文字用過的 `<i>`、`<color=#rrggbb>`、`<size=NN%>`，成對，而且種類與數量與 KR 相同。
5. 值不是 `Unknown`；沒有殘留的韓文、假名。

**只影響呈現**

6. 開頭、結尾沒有空白（KR 同一筆有的除外）；沒有空行（連續的 `\n`、以 `\n` 開頭或結尾）。
7. `\n` 的數量與 KR 不同時提醒，不當成錯誤（官方 EN 就有 45 筆不同）。
8. 長度：看得到的字數超過 KR 與 JP 較長者的 1.3 倍、而且至少多 5 個字時提醒；超過 111 字（官方 CJK 的最大值）時提醒。這一項同時能找到譯文與原文對不上的句子。
9. `<原 id>-0N` 的變體，如果 KR 與原 id 的文字相同，譯文也應該相同。
10. `desc` 與 KR 相同。翻譯它沒有壞處，但也沒有任何效果。

與現有檢查的關係：第 2～5 項已經由對所有欄位都做的既有檢查涵蓋（`empty`、`stray-brace`、`ruby-form`、`literal-newline`、`tag-mismatch`、`reserved-value`、`hangul`、`kana`）。**還沒有檢查的是第 6 項（頭尾空白）、第 8 項（現有的 `length-ratio` 要超過 3 倍才提醒，抓不到這裡的情況）與第 9 項。**
