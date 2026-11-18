# 各類檔案的欄位在什麼時候顯示

2026-11-16。逐張資料表、逐個欄位追「誰讀它、顯示在哪個畫面、什麼條件下才顯示」，並整理遊戲不讀或不顯示的欄位。

依據：各 `TextData_*` 類別的欄位宣告、讀取每張表的函式與它們對欄位位移的讀取、畫面元件的文字指定（2026-11-12 版）；資料是 `raw-20261112` 與 `tw` 分支 7fccf1d。「確認」＝對照反編譯或反組譯，或資料的實際數字；「推定」＝有依據但沒有直接讀到；位址是 RVA。

方法與限制：

- 很多 getter（例如「回傳第一個字串欄位」）被幾百個類別共用同一段機器碼，沒辦法用「誰呼叫它」來追。做法是先找出讀取那張表的所有函式，再看它們拿到條目後讀了哪個位移。
- 「沒有讀取者」的結論因此是「直接讀取的地方都找過了」。經由介面或委派、再用共用 getter 讀的地方看不到，這個風險很小但不是零。
- 畫面元件的版面（寬度、字級、文字物件存不存在）在 prefab 裡，讀不到。
- 「單位數」指 `tw` 分支裡有翻譯紀錄而且不是照抄的單位。

## 先講結論

- **「短說明」只在戰鬥中的技能面板出現。** 硬幣的 `summary`、被動的 `summary`、狀態的 `summary`、飾品的 `simpleDesc` 都是同一個面板裡「這次觸發了什麼」的列。其他地方（人格資訊、提示框、選技面板）一律用完整的 `desc`。
- **狀態有兩份文字**：畫在技能文字裡的名稱、狀態圖示的提示框、人格資訊的狀態清單用 `Bufs*`；點關鍵字連結跳出來的說明、選技面板的關鍵字欄用 `BattleKeywords*`。兩邊要一致。
- **劇情的說話者名稱與稱號各自決定**：列上有寫就用列上的，空的才用 `ScenarioModelCodes*`。
- **約 3,300 個已翻譯的單位在遊戲不顯示的欄位裡**（第 6 節）。最大的是 `Enemies*`／`Assist*` 的 `desc`（1,277）、`EGOVoiceDig` 的 `desc`（335）、`BattleKeywords*` 的 `summary`（236）。
- **會顯示、容易被當成標籤而忽略的**：`PersonalityVoiceDlg` 的 `desc`（語音清單每一格的標題）、`LobbyBGM` 的 `songWriter`、`UnlockCode` 的各欄位。
- 過去報告有 13 處需要修正或補充，已一併改好（第 8 節）。

## 1. 技能、被動、狀態、飾品

### 戰鬥中的技能面板

單位在戰鬥中執行技能時，旁邊會開一個面板（`SkillAndCoinUI`）：技能名稱、威力、硬幣列，以及這次觸發的被動、狀態、飾品。「短說明」全部只出現在這裡。

- **什麼時候開**：開始擲硬幣時（`InitData` 0x2A30550），每擲一枚硬幣更新一次（`StartRollingCoin` 0x2A30160）。只在戰鬥中，大廳不會出現（確認）。
- **遊戲設定關掉「戰鬥能力介面」（`_showBattleAblityUI`）時整塊不顯示**；面板為拼點而開的那一刻也不顯示硬幣文字（確認）。
- **硬幣文字框一次只顯示一枚硬幣**，旁邊是那枚硬幣的羅馬數字。內容是這枚硬幣的各條效果裡**已觸發的那幾條**，用換行接起來。「已觸發」來自戰鬥紀錄裡的效果編號清單（清單的內容是確認，「就是實際觸發的效果」是從名稱推定）。
- **面板有兩種樣式**：詳細與簡易。**只有異想體戰**（以及另一種檢視模式，名稱推定是遭遇戰）用詳細樣式（確認）。簡易樣式下，觸發的被動、狀態、飾品**只列名稱**，短說明不顯示。
- 觸發的列最多 5 列，順序是 E.G.O 被動、被動、支援被動、狀態、飾品；超過的只顯示數量。
- 程式沒有截斷、沒有行數上限，文字框依文字大小調整（確認）。寬度與字級讀不到。同一個元件裡只有**技能名稱**會被截短加「...」。

### 長說明與短說明

| 短的欄位 | 面板裡的位置 | 什麼時候看得到 | 空的時候 | 其他用途 |
|---|---|---|---|---|
| `Skills*` 硬幣的 `summary` | 硬幣文字框 | 設定開啟、不是拼點開面板的那一刻；兩種樣式都有 | 改用同一條的 `desc` | 沒有 |
| `Passive*` 的 `summary` | 觸發的被動那一列 | **只有詳細樣式** | 改用 `desc` | 共鳴階段的說明文字 |
| `Bufs*` 的 `summary` | 觸發的狀態那一列 | **只有詳細樣式**，而且不是空的 | 見下 | 沒有 |
| `EGOgift*` 的 `simpleDesc[]` | 觸發的飾品那一列 | **只有詳細樣式**，而且不是空的 | 只顯示名稱 | 沒有 |

- **硬幣 `summary`**：官方只有 83 條有內容（`Skills_personality` 38、`Skills_Ego_Personality` 27、`Skills.json` 9、`Skills_Enemy` 6、`Skills_Abnormality` 3），中位數 42 字，對應的 `desc` 是 99 字。它是戰鬥當下取代 `desc` 的短版：**KR 是空的就要留空**（否則短文字會蓋掉 `desc`），**KR 有內容就要有內容**（否則會顯示整段 `desc`）。`coindescs[]` 是依位置對應效果編號的，順序不能動。
- **硬幣 `summary` 的關鍵字處理**：`[標籤]`、`[狀態 id]`、純文字名稱三種都做，而且是**每一條各自處理**之後才接起來（確認）。
- **狀態 `summary`：「沒有這個 key」與「空字串」不同**（確認）。沒有 key 的 408 筆會改用完整的 `desc`；寫成 `""` 的 1,201 筆只顯示名稱。機制是：沒有 key 時欄位留在建構時的預設值 `Unknown`，程式看到 `Unknown` 才改用 `desc`。
- **飾品浮在單位頭上的發動文字只有名稱**，沒有 `simpleDesc`。

### 狀態（`Bufs*`）與關鍵字說明（`BattleKeywords*`）誰用在哪裡

| 玩家看到的 | 用哪張表 | 欄位 |
|---|---|---|
| 技能、被動、飾品文字裡的狀態名稱（`[id]` 與自動連結） | `Bufs*` | `name` |
| 戰鬥中滑過**狀態圖示**的提示框 | `Bufs*` | `name`、`desc`（代入層數等數值）、`flavor`（有內容才顯示） |
| 人格資訊的**狀態清單**與點開的說明 | `Bufs*` | `name`、`desc`、`flavor` |
| 技能面板裡觸發的狀態那一列 | `Bufs*` | `name`＋層數、`summary` |
| 精神分頁、選擇事件結果、鏡牢起始增益、單位頭上的浮動文字 | `Bufs*` | `name` |
| **點關鍵字連結**跳出來的說明 | `BattleKeywords*` | `name`、`desc`、`flavor` |
| 選技面板的關鍵字欄 | `BattleKeywords*` | `name`、`desc` |
| 關卡增益、編隊增益、鏡牢飾品的增益清單 | `BattleKeywords*` | `name`、`desc` |

- `Bufs*` 的 `summary` **沒有任何提示框使用**。
- `Bufs*` 與 `BattleKeywords*` 的 `flavor` 是各自的一份，顯示在不同的框。
- `BattleKeywords*` 的類別只有 `name`、`desc`、`flavor` 三個欄位；檔案裡的 `summary`、`undefined`、`iconId`、`excludeAutoLink` 都不會被讀。`excludeAutoLink` 寫在 `BattleKeywords*` 裡沒有作用，要寫在 `Bufs*`。

### 各欄位

**`Skills*`**

| 欄位 | KR 有內容的字串 | 顯示在哪裡 | 條件 |
|---|---|---|---|
| `levelList[].name` | 6,217 | 所有顯示技能名稱的地方：人格資訊、技能欄、戰鬥中的技能面板、選技面板、選擇事件紀錄 | 技能面板裡過長會截短加「...」 |
| `levelList[].desc` | 4,985 | 人格資訊的技能頁（大廳與戰鬥）、選技面板、E.G.O 選擇、同步比較、異想體圖鑑 | **不在**戰鬥中的技能面板 |
| `…coindescs[].desc` | 16,816 | 同上那幾處的硬幣效果清單；技能面板在 `summary` 空的時候也用它 | |
| `…coindescs[].summary` | 83 | 只有技能面板的硬幣文字框 | 見上 |
| `levelList[].flavor` | 51 | 人格資訊的技能頁 | 該 prefab 有這個文字物件才顯示 |
| `levelList[].abName` | 672 | 戰鬥中的 E.G.O 技能畫面、人格資訊的 E.G.O 名牌 | 只有 E.G.O 技能 |
| `levelList[].keywords` | 25 | **不讀**（不是類別的欄位） | |
| `levelList[].rawDesc` | 0 | 有值時取代 `desc`；官方沒有資料 | |

**`Passive*`**

| 欄位 | KR | 顯示在哪裡 | 條件 |
|---|---|---|---|
| `name` | 2,611 | 所有被動介面的標題、技能面板的名稱列、選擇事件紀錄 | |
| `desc` | 2,612 | 人格資訊的被動頁、敵人資訊、E.G.O 同步彈窗、戰鬥中被動圖示的提示框、連結被動、鐵道 6 號線 | |
| `summary` | 270 | 技能面板觸發的被動那一列；共鳴階段說明 | 前者只有詳細樣式 |
| `flavor` | 291 | 人格資訊的被動頁 | 該 prefab 有這個文字物件才顯示 |
| `undefined` | 66 | **不讀** | 內容都是 `-` |

**`EGOgift*`**

| 欄位 | KR | 顯示在哪裡 |
|---|---|---|
| `name` | 814 | 所有飾品介面、技能面板的名稱列、單位頭上的發動文字、選擇事件結果 |
| `desc` | 814 | 全部 12 種飾品說明介面 |
| `simpleDesc[].simpleDesc` | 1,370 | 只有技能面板觸發的飾品那一列（詳細樣式） |
| `abnormalityName` | 1 | 沒有找到顯示的地方（推定不顯示） |

從沒取得過的飾品，說明會換成 `MirrorDungeonEgoGiftLockedDesc` 的 `content`。

**人格、E.G.O、敵人**

| 欄位 | KR | 顯示在哪裡 | 條件 |
|---|---|---|---|
| `Personalities*.title`＋`name` | 209 | 顯示名稱是「`title`＋半形空格＋`name`」；單行的地方把 `title` 的 `\n` 換成空格 | |
| `Personalities*.desc` | 208 | **只當 `skinItemDesc` 的後備**；目前的資料用不到（推定不顯示） | 內容是「이상의 첫번째 인격」這類註記 |
| `Personalities*.skinItemTitle`、`skinItemDesc` | 各 1 | 獎勵道具的提示框標題與內文 | 只有造型道具；空的時候改用 `title`／`desc` |
| `Personalities*.nameWithTitle` | 207 | **不讀**（不是類別的欄位；顯示名稱是程式組出來的） | |
| `PersonalityCompanion.title`、`name` | 各 1 | 一起取得的人格組合（見下） | 2026-11-12 新增的表 |
| `Egos*.name` | 122 | 所有顯示 E.G.O 名稱的地方 | |
| `Egos*.desc` | 122 | **不顯示**：唯一的讀取鏈沒有人呼叫（確認） | 內容是「이상 전용 EGO 장비」 |
| `Enemies*.name`、`Assist*.name` | 1,447 | 敵人、異想體、部位、協助者的名稱 | |
| `Enemies*.desc`、`Assist*.desc` | 1,277 | **不讀**：連「取得說明」的函式回傳的都是 `name`（0x1861DD0，確認） | 內容是「본체」「부위」「적 잡몹」 |
| `Characters.name` | 13 | 罪人的名字；很多介面句子的代入值 | |

**一起取得的人格組合（`PersonalityCompanion`）**

2026-11-12 新增的表（`TextDataSet` 位移 0x268，類別 `TextData_PersonalityCompanion`：`title` 0x18、`name` 0x20）。目前只有一筆：`id` 1 是楔事務所的唐吉訶德與辛克萊，`title` 是「쐐기 사무소\n해결사」，`name` 是「돈키호테·싱클레어」。整個程式讀這張表的只有四個函式（確認）：

| 顯示在哪裡 | 讀什麼 | 處理 |
|---|---|---|
| 名牌：`NameTagDisplayData.FromPersonality`（0x1471560），由 `NameTag.SetData`、抽卡取得新人格的演出（`GetNewCardTimelineScriptBase.SetData`）、抽卡畫面的當期人格展示（`GachaUIPanel` 的 `StartDisplayPickup`）呼叫 | `title`、`name`，各一個文字框 | `title` 的 `\n` 換成半形空格；`name` 原樣（`\n` 會換行） |
| 抽卡機率表：`RateInfoItem.SetData`（0x207D180）→ `BuildName`（0x207EA40） | `title`、`name` | 組成 `[`＋`title`＋`]`＋半形空格＋`name`，再把整串的 `\n` 換成半形空格 |
| 抽卡保底兌換：`PityExchangeSlot.SetData`（0x2078930） | `title`、`name` | 兩個都把 `\n` 換成半形空格 |
| 販賣機的人格商品格：`PersonalitySlotCoordinator.MakeUpCompanion`（0x14B9D80），由 `VendingMachineUIPersonalityGoodsSlot.SetData` 呼叫 | 只有 `title` | 原樣設進文字框（`ApplySlotView` 0x14BA320）；程式會先檢查 `title` 裡有沒有 `\n`，用途沒有追，推定是選版面 |

- 規則和 `Personalities*` 的稱號相同：`title` 的 `\n` 兩旁不要有空格；`title` 不要自己帶方括號。
- `name` 在名牌上是原樣顯示，在另外兩處 `\n` 會變成空格。官方 EN 的 `name` 是「Don Quixote\n& Sinclair」，KR、JP 沒有 `\n`；譯文沒有 `\n`，三處顯示一致。
- 兩個人格各自的稱號與名字仍在 `Personalities*`（`10316`、`11017`），這張表只是組合的顯示名稱。

**其他小表**

| 欄位 | 顯示在哪裡 |
|---|---|
| `SkillTag.name` | 取代技能、被動文字裡的 `[標籤 id]` |
| `BuffAbilities` 的 `desc`、`variation`、`variation2` | 產生狀態說明用的範本，代入數值與狀態名稱後才顯示；`name`、`summary` 不是欄位 |
| `PanicInfo*` 的 `panicName`、`lowMoraleDescription`、`panicDescription`、`flavor` | 人格資訊的精神分頁 |
| `MentalCondition*` 的 `add`、`min` | 精神分頁的條件；Project GS 的成長結果。`{n}` 個數不對就不顯示 |
| `DanteAbility*` 的 `rawDesc` | 但丁能力卡片上的一行效果 |
| `DanteAbility*` 的 `desc` | 選擇目標的彈窗裡的說明 |
| `AttributeText.name` | 屬性名稱；`desc` 沒有資料也沒有讀取者 |

## 2. 劇情

### 劇情檔的每一列

| key | KR | 顯示在哪裡 | 條件 |
|---|---|---|---|
| `content` | 63,031 | 對話框（逐字）、黑幕台詞、演出字幕、戰鬥中插入的劇情、劇情紀錄 | |
| `teller` | 9,543 | 對話框的名牌；劇情紀錄的名字 | 見下 |
| `title` | 13,295 | 名牌下的稱號 | 見下；**劇情紀錄不顯示稱號** |
| `place` | 1,501 | 地點橫幅 | 見下 |
| `model` | 46,893 | 不顯示；是查名稱、立繪、名牌顏色的 key | |
| `id` | — | key；**`-1` 的列整列跳過** | 25 列（21 個 `content`、3 個 `teller`、3 個 `title`）永遠不顯示 |
| `d` | 61 列 | **不讀**（是 `id` 的筆誤） | |

**名稱與稱號**（`StoryPlayData` 的建構式 0x2139280，確認）：

- `model`、`teller`、`title` 全空，**或**演出檔把這一列標成獨白 → 旁白樣式，名稱與稱號都不顯示（即使列上有寫）。
- `model` 有值：
  - 名稱＝列上的 `teller`；空的話用 `model` 去查 `ScenarioModelCodes*` 的 `name`。
  - 稱號＝列上的 `title`；空的話查 `ScenarioModelCodes*` 的 `nickName`。
  - **兩者各自決定**。KR 有 4,131 列是稱號用列上的、名稱用查的；379 列相反。
- `model` 是空的：`teller` 與 `title` **都有值**才顯示，否則兩個都不顯示。
- 列上有寫的一定優先。語言包生效與否只影響「用 `model` 查」時查的是語言包的 `ScenarioModelCodes*` 還是內建名稱。
- KR 的實際分布：名稱來自列上 9,540 列、來自 `ScenarioModelCodes` 39,376 列；稱號來自列上 13,292 列、來自 `nickName` 35,624 列。**大部分的名稱與稱號其實是 `ScenarioModelCodes*` 的譯文。**

**地點**：有內容的 `place` 每一列都會套用，並且**一直留著，直到遇到 `place` 是 `-1` 的列**才淡出。沒有這個 key 的列不改變橫幅。橫幅原本是隱藏的才會逐字打出來。

### `ScenarioModelCodes*`

- `name`（745）、`nickName`（572）只在「列上沒寫、而且有 `model`」時使用。
- `nickName` 找不到 id 時用內建的**英文**稱號；`name` 沒有後備，會是空白。
- KR 劇情裡有 306 個 `name`、144 個 `nickName` 所屬的 id 從來沒有被查過（不是任何列的 `model`，或用到的列都自己寫了名稱）。這些**很可能不顯示**（從資料推定）。

### 劇情相關的小表

| 表 | 欄位 | 顯示在哪裡 |
|---|---|---|
| `Story-a1c8p3` | `content`（1） | 劇情裡的文字特效 |
| `Ch8TextEffectSettings`、`Ch8VideoEffectSettings` | `…texts[].texts[].text`（46、21） | 第 8 章的文字與影片特效；其餘 key 都是版面數值與色碼。`mainText`、`subText` 官方都是空的 |
| `IntroduceCharacter` | `name`、`description`（各 12） | 劇情裡的罪人介紹卡；`sentence` 官方都是空的 |
| `StoryTheaterMain`、`StoryTheaterOther` | `title`（63） | 劇情劇場的清單與彈窗 |
| | `desc`（11） | 只有主線頁的章節簡介；章節未解鎖時顯示的是解鎖條件 |
| `StoryTheaterDanteNoteDetail*` | `desc`（177） | 但丁筆記內文 |
| | `title` | 查找的 key 對不上，標題永遠是介面文字「紀錄 #n」；官方也沒有資料 |
| `StoryTheaterDanteNote*` | `content`（100） | 分類名稱；也是筆記內文裡 `<id>N</id>` 連結的文字 |

## 3. 語音與歌詞

### 人格語音（`PersonalityVoiceDlg/*`）

- **`desc` 會顯示**：人格故事畫面的語音分頁裡，每一格的標題就是 `desc`（0x1FBF5D0 → 0x1FC31B0，確認）。檔案裡每一筆各一格，依檔案順序。翻譯它是必要的。
- `dlg` 顯示在哪裡由 id 決定：

| id | 顯示在哪裡 |
|---|---|
| 全部 | 語音分頁：播放時顯示那一句 |
| `lobby_morning/noon/night_…` | 大廳卡片的問候 |
| `smalltalk_…` | 大廳卡片，點擊時 |
| `neglect_…` | 大廳卡片，閒置時 |
| `get_…` | 抽卡取得新人格 |
| `gacksung_…` | 同步升級的演出面板 |
| `battle_clear_…`、`battle_clear_ex_…`、`battle_defeat_…` | 戰鬥結算畫面 |
| `choice_success_p_…`、`choice_fail_n_…` | 判定事件，判定之後罪人的對話框 |
| `battle_select`、`battle_endcommand`、`battle_kill`、`battle_break`、`battle_enemy_break`、`battle_dead`、`battle_allydead` | 戰鬥氣泡（[05](05-battle-speech-bubble.md)） |
| `formation_…`、`battleentry_…` | **只在語音分頁**（播放時不顯示文字） |

- 文字清單也影響播哪一句語音：閒聊語音選到的編號在清單裡沒有時，改播第 1 句。

### 其他語音與歌詞

| 檔案 | 欄位 | KR | 顯示在哪裡 |
|---|---|---|---|
| `EGOVoiceDig/*` | `dlg` | 377 | 技能演出的台詞框、演出字幕、抽卡取得新 E.G.O |
| | `desc` | 335 | **不讀**：讀這張表的 23 個函式都沒有讀它，也沒有 E.G.O 語音清單的介面（確認） |
| `BattleAnnouncerDlg/*` | `dlg` | 1,950 | 戰鬥中的播報員對話框、播報員試聽彈窗、抽卡取得新播報員 |
| | `imgStr` | 105 | 不顯示；那一句用的頭像 key |
| | `isHide`、`playWeight` | — | 資料：試聽按鈕要不要跳過、同類台詞的隨機權重 |
| `Announcer*` | `name` | 75 | 播報員名稱（編隊、試聽、戰鬥、抽卡紀錄） |
| `AnnouncerVoiceType` | `content` | 32 | 試聽彈窗的語音分類標籤 |
| `BgmLyrics/*` | `content` | 662 | 戰鬥歌詞、戰鬥與劇情的上方歌詞、製作名單、PV |
| | `id`、`vocalid` | — | 行號、歌手編號，不顯示 |
| `BattleSpeechBubbleDlg*` | `dlg` | 1,555 | 見 [05](05-battle-speech-bubble.md) |
| | `desc` | 886 | 不讀 |

## 4. 選擇事件、異想體、關卡

### 判定事件（`AbEvents*`）

畫面上**沒有事件標題**，只有介面文字的標籤。流程（確認）：

1. `eventDesc`：上方的框，事件一開就顯示，分頁。330 筆裡 214 筆沒有，框就是空的。
2. `behaveDesc`：下方的框，`eventDesc` 讀完後出現，玩家選罪人時一直留著。分頁。
3. 判定。
4. `successDesc[i]` 或 `failureDesc[i]`：結果文字，分頁。
5. 效果紀錄（`AbEventsResultLog*`、`ChoiceEventEffect*`）。

- 沒有「第一次來／再來」的分支；後續的文字是另一個事件 id。
- `i` 由靜態資料的結果條件（地牢）或每個異想體寫死的程式（戰鬥）決定，與文字無關。陣列的位置要與 KR 相同。
- **`prevDesc` 不在這個流程裡**：它是戰鬥中異想體事件開啟的**前一回合**出現的「預兆」橫幅，多條警告用換行接起來，並且**轉成大寫**。它不分頁。
- `cancelDesc`：產生它的分支在程式裡走不到，官方也沒有資料。
- `title`（197）：不是類別的欄位，不讀。內容是「던전 103003 선택지」這類開發標籤。

### 選項事件（`ActionEvents*`）

1. `desc`：上方的框，分頁；讀完才出現選項。
2. `options[i].message`：按鈕文字。`options[i].messageDesc`：按鈕下方的效果提示。
3. 點選後：`options[i].message` 再出現一次當結果標題，接著 `options[i].result[j]`，分頁。

- **`messageDesc` 在這個選項被選過一次之前顯示成 `???`**（劇情地牢與鏡牢，有寫死的例外）。空的 `messageDesc` 不會被換掉。
- 按鈕的數量由靜態資料決定，不是 `options` 的長度。**語言包少了一個選項時，按鈕上會出現一段韓文除錯字串**。
- `result[j]` 超出範圍是空字串，結果框會被跳過。
- **`name`（202）、`subDesc`（40）沒有讀取者**（確認；風險見開頭）。`subDesc` 的內容本身就是「사용 안하는 텍스트」。
- `title`（11）、`prevDesc`（1）不是類別的欄位。

### 事件裡罪人的對話框（`AbDlg_*`）

- **判定之前**（滑過或選取罪人的頭像）：依預期成功率的級距（非常高、高、普通、低、非常低，門檻 0.9／0.6／0.4／0.1）選一類，再隨機挑一句。這個人格沒有自己的台詞時，用該罪人第一個人格的台詞。
- **判定之後**：先找 `PersonalityVoiceDlg` 的 `choice_success_p_…`／`choice_fail_n_…`；找不到才用 `AbDlg` 的 `SUCCESS`／`FAILURE`。197 個人格語音檔裡 196 個有這兩句，所以 `AbDlg` 的這兩類幾乎用不到。
- `usage` 的格式是「(事件 id, 類型)」的清單。**只有類型有作用**；事件 id 會被解析（必須是數字，否則載入時出錯），但從來不用來挑台詞。
- `teller`（169）：沒有讀取者。`voicefile`／`voiceFile`：不讀。
- 這個對話框在所有判定事件都有（地牢事件也是），不只異想體事件。

### 選擇事件的結果句（`ChoiceEventEffect*`、`ChoiceEventTarget`、`ChoiceEventKeyword`）

三個檔是同一張表（確認）。`ChoiceEventEffect*` 是結果紀錄的範本，另外兩個是代入 `{targets}`、`{conditions}` 等位置的片段。

### 異想體圖鑑（`AbnormalityGuides*`）

| 欄位 | KR | 顯示在哪裡 | 條件 |
|---|---|---|---|
| `storyList[].story` | 398 | 人格資訊的觀察紀錄彈窗，每一筆一段 | `level` 不大於研究等級才解鎖（擊殺 1／3／6／10／15 次是等級 1～5；`level` 0 是初次遭遇就有） |
| `codeName` | 187 | 人格資訊的異想體名牌、戰鬥中的狀態列 | 未解鎖時顯示 `???` |
| `name` | 193 | **只有**戰鬥中選定目標的詳細面板 | 名牌與狀態列用的是 `Enemies*` 的 `name`，兩邊要翻成一樣 |
| `clue` | 183 | **沒有讀取者** | 183 筆裡 180 筆是佔位的「단서」 |
| `desc` | 9 | 不是類別的欄位，不讀 | |

`storyList[].level` 重複不會出錯：把它放進字典的函式沒有人呼叫，介面是直接逐筆列出。

### 關卡、章節、地牢

| 檔案 | 欄位 | KR | 顯示在哪裡 |
|---|---|---|---|
| `StageNode*` | `title` | 516 | 章節地圖的關卡標籤、關卡資訊面板的標題 |
| | `desc` | 190 | 關卡資訊面板的說明 |
| | `place` | 208 | **沒有讀取者**（確認；劇情裡的地點來自劇情檔的 `place`） |
| `DungeonNode*` | `stageList[].title`、`.desc` | 501、267 | 劇情地牢點節點開的資訊面板 |
| | `stageList[].debugNodeId ` | 126 | 不讀（key 的結尾有一個空格，不是欄位） |
| `RailwayDungeonNode*` | `title`、`desc` | 73、37 | 同一個資訊面板；「要移動到這個節點嗎」的對話框 |
| `HellsChickenDungeonNode`、`MirrorDungeonNode1-1` | 全部 | 30 | **整張表沒有人讀**（確認） |
| `StageChapterText*` | `company`、`area` | 各 32 | 章節選擇地圖的區域標題 |
| | `chaptertitle`、`chapter` | 31、26 | 章節橫幅、主線頁標題、賽季開始面板等 |
| | `chapterNumber`、`timeline` | 26、31 | 主線頁的章節編號、各種橫幅的時間 |
| `StagePartText` | `parttitle` | 3 | 篇章選擇 |
| `DungeonArea*` | `content` | 113 | 劇情地牢地圖上的標籤 |
| `Quest` | `content` | 8 | 關卡資訊的通關條件、戰鬥中的任務計數 |
| `RailwayDungeon` | `name`、`longName` | 8、2 | 路線名稱；`longName` 只在 4 號線的結算面板 |
| `RailwayDungeonStationName*` | `nameList[].content`、`.shortName` | 78、59 | 上方狀態列與進度清單；獎勵說明裡的區段標籤 |

- **`StageChapterText` 的 `area` 還有第二個用途**：地圖上每個區域的標籤是「DISTRICT 」加上 `area` 裡的**半形數字**。寫成中文數字或全形數字時，標籤只剩「DISTRICT 」。現行譯文用的是半形數字，沒有問題。

### 鏡牢、提示、教學

| 檔案 | 欄位 | 顯示在哪裡 |
|---|---|---|
| `DungeonStartBuffs*` | `description`（116） | 鏡牢起始增益的選擇介面；代入數值與狀態名稱 |
| `MirrorDungeonTheme` | `name`（122） | 主題卡片、樓層標籤、節點資訊標題 |
| | `specialName`（1） | 呼叫者要求時取代 `name`；哪個呼叫者會要求沒有追到 |
| `MirrorDungeonEgoGiftLockedDesc` | `content`（65） | 還沒取得過的飾品，取代飾品說明 |
| `MirrorDungeonRentalName` | `content`（38） | 鏡牢入口的預設編隊名稱 |
| `BattleHint` | `content`（35） | 載入畫面的提示 |
| `BattleResultHint*` | `content`（106） | 戰鬥結算畫面的關卡專屬提示 |
| `BattleHint_NormalBattle`、`_AbnorBattle` | `content` | 關卡沒有專屬提示時的通用提示（推定） |
| `TutorialDesc*` 等 | `content`（428） | 教學的文字框與彈窗 |
| `ProjectGSLessonName` | `content`、`teacher` | 課程名稱；組合需求清單裡的名字 |

## 5. 大廳、商店、道具、RPG

| 檔案 | 欄位 | KR | 顯示在哪裡 |
|---|---|---|---|
| `Items*` | `name` | 199 | 背包、獎勵清單、提示框標題；也用來排序 |
| | `desc` | 199 | 提示框內文、背包說明 |
| | `flavor` | 161 | **只在背包的道具詳細面板** |
| `UserBanner` | `name`、`desc` | 各 97 | 獎勵名稱；個人檔案橫幅的提示框 |
| `UserTicket-*` | `name`、`desc` | 各 90 | 個人檔案的票券裝飾清單；長按的提示框 |
| `IAPProduct*` | `name`、`desc` | 各 195 | 商店、購買彈窗、信件 |
| `GachaTitle`、`GachaNotice` | `content` | 218、4 | 抽卡的提醒、紀錄、保底到期彈窗 |
| `BattlePass_Mission` | `content` | 14 | 通行證任務 |
| `UnlockCode` | `openCondition` | 46 | 關卡、主題樓層、鐵道路線的解鎖條件 |
| | `openConditionNumber` | 6 | **只在頭目討伐**的文字裡取代 `openCondition` |
| | `relatedChapterText` | 38 | 載入圖選擇的章節標題；大廳音樂的取得說明 |
| | `askLevelUp` | 3 | 升級確認 |
| `LobbyBGM*` | `name`、`songWriter` | 各 356 | 音樂清單；`songWriter` KR 與 EN 的值相同 |
| `Personality_Get_Condition`、`EGO_Get_Condition` | `content` | 395 | 大廳橫幅的提示框 |
| `LoginUIText`、`UserAgreements` | `content` | 96 | 登入、版本、下載大小、使用者條款 |
| `FileDownloadDesc` | `title`、`desc` | 各 9 | 下載遊戲資料時輪播的頁面 |
| `ErrorCodeMsg` | `content` | 20 | 伺服器錯誤訊息 |
| 介面文字（154 個檔） | `content` | 5,721 | 依 id 取用，遍布各處 |

**RPG（`RPGSystem/*`）**：`key`、`texts[].index`、`steps[].index` 是 key，其餘都是顯示的文字。

- 對話的每一句是用 `texts[].index` 找的，不是依位置。
- 對話的 `title` 只在 `speaker` 有內容時顯示；官方資料沒有 `title`。
- 任務的 `steps[].goalDescription1`～`8` 依編號放進對應的目標欄位。

## 6. 遊戲不讀、不顯示的欄位

翻譯這些欄位沒有壞處，但不會出現在遊戲裡。可以在 `rules/` 的欄位規則裡排到最後或不送翻譯。

### 不是類別欄位的 key（不會被載入）

類別的欄位是確認；「Unity 讀 JSON 時會忽略多出來的 key」是推定。

| 欄位 | KR 有內容的字串 | 已翻譯的單位 |
|---|---|---|
| `BattleKeywords*` 的 `summary` | 243 | 236 |
| `Personalities*` 的 `nameWithTitle` | 207 | 207 |
| `AbEvents*` 的 `title` | 197 | 170 |
| `ActionEvents*` 的 `title`、`prevDesc` | 12 | 11 |
| `AbnormalityGuides*` 的 `desc` | 9 | 9 |
| `BuffAbilities` 的 `name` | 2 | 2 |
| `Bufs*`、`BattleKeywords*`、`Passive*` 的 `undefined`；`iconId`、`iconID` | 約 1,970 | 0 |
| `BattleKeywords*` 的 `excludeAutoLink` | 1 | — |
| `Skills*` 的 `levelList[].keywords` | 25 | 0 |
| `DungeonNode*` 的 `stageList[].debugNodeId ` | 126 | 0 |
| `AbDlg_*` 的 `voiceFile`（欄位是小寫的 `voicefile`） | 0 | 0 |
| 劇情檔的 `d` | 61 列 | 0 |

### 是類別的欄位，但沒有顯示它的地方

| 欄位 | KR 有內容的字串 | 已翻譯的單位 | 依據 |
|---|---|---|---|
| `Enemies*`、`Assist*` 的 `desc` | 1,277 | 1,277 | getter 沒有呼叫者；用這張表的 4 個函式都讀 `name`（確認） |
| `EGOVoiceDig/*` 的 `desc` | 335 | 335 | 23 個讀取函式都不讀它（確認） |
| `Personalities*` 的 `desc` | 208 | 208 | 只當造型道具說明的後備，目前資料用不到（推定） |
| `StageNode*` 的 `place` | 208 | 208 | 14 個讀取函式都不讀它；介面資料也沒有對應的成員（確認） |
| `AbnormalityGuides*` 的 `clue` | 183 | 183 | 6 個讀取函式都不讀它（確認） |
| `ActionEvents*` 的 `name` | 202 | 171 | 11 個讀取函式與 2 個使用快取條目的函式都不讀它（確認） |
| `AbDlg_*` 的 `teller` | 169 | 169 | 只讀 `personalityid`、`usage`、`dialog`（確認） |
| `Egos*` 的 `desc` | 122 | 122 | 讀取鏈的終點沒有呼叫者（確認） |
| `HellsChickenDungeonNode`、`MirrorDungeonNode1-1` 整個檔 | 30 | 30 | 整張表沒有人讀（確認） |
| `ActionEvents*` 的 `subDesc` | 40 | 5 | 同 `name` |
| `EGOgift*` 的 `abnormalityName` | 1 | 1 | 推定 |
| `BattleSpeechBubbleDlg*` 的 `desc` | 886 | 0 | [05](05-battle-speech-bubble.md) |

兩張表合計，已翻譯的單位約 3,340 個。

### 不在官方檔案清單裡的檔（整個檔不會被載入）

2026-11-18 取得 2026-11-12 版的檔案清單後核對（[`method/filelist.py`](method/filelist.py)）。遊戲只讀清單列出的檔名，程式的字串常值裡也沒有這些檔名，所以下面 17 個檔的內容不會出現在遊戲裡（清單的內容是確認；「沒有別的載入路徑」是依字串常值與載入函式推定）。

| 檔案 | KR 有內容的字串 | 譯文已改動的字串 | 說明 |
|---|---|---|---|
| `StoryTheaterPersonality` | 292 | 292 | 人格故事的標題（`P10102` 等 93 筆）、舊版的人格簡介與分析、幾個按鈕字 |
| `StoryTheaterMirrorWorldStoryTitle` | 183 | 183 | 「남부 세븐 협회 6과, 이상 이야기」這類標題。遊戲顯示的標題是程式拼出來的（`StoryTheaterPersonalityInfo.GetStoryTitle` 0x18229F0，[09](09-assembled-text.md)），不讀這個檔 |
| `StoryTheaterMirrorWorld` | 26 | 26 | `MirrorWorld_Name_1` 等世界名稱 |
| `DungeonStartBuffs_MD6` | 44 | 44 | 清單只列 `DungeonStartBuffs_MD7`；44 個 id 在 `_MD7` 都有 |
| `DungeonStartBuffs` | 25 | 25 | 同上；25 個 id 有 23 個在 `_MD7` |
| `UserTicket-{L,R,EGOBg}-a1c10p1` | 各 6 | 各 3 | 名稱是「???」的暫定版；清單列的是 `-a1c10p3` |
| `Announcer-a1c10p1`、`Announcer-a1c9p1` | 3、2 | 1、0 | 名稱是「??」的暫定版 |
| `SynergyKeyword` | 1 | 1 | 見下 |
| `ChapterBannerConfig`、`Items-BossRaid`、`Skills_Enemy-a1c9p2`、`UserTicket-{L,R,EGOBg}-a1c9p3` | 0 | 0 | 空檔 |

合計 594 個字串，譯文已翻 581 個。這些檔翻不翻都不影響畫面；[06-families.md](06-families.md) 提到的 `mirror`↔`stage` 93 句跨族群相同（人格故事標題）就是前兩個檔之間的。

`HellsChickenDungeonNode`、`MirrorDungeonNode1-1`、`LobbyBGMCategory` 在清單裡，會被載入；前兩個是載入之後沒有人讀。

`SynergyKeyword.json`（2026-11-12 新增，一筆：`MAGICAL_GIRL`／「마법 소녀」）的依據：

- 文字表與檔案清單的 77 個欄位裡沒有共鳴關鍵字的表（確認）；程式的字串常值裡也沒有這個檔名。2026-11-12 版的官方檔案清單沒有把它列進任何一張表（確認），所以它不會被載入。
- 程式裡的共鳴關鍵字是列舉 `SYNERGY_KEYWORD`（`NONE`、`MAGICAL_GIRL`），由單位與 E.G.O 的靜態資料帶著。讀它的函式都在 `SynergyAbilityManager`、`SynergyPassiveAbilityBase.CheckActiveCondition`（0x3222C50）、`SynergyUIDataBuilder`（`GetGroupList` 0xF4FF00、`GetUnsatisfiedConditionValue` 0xF4F970、`CollectActiveMember` 0xF4F630），用途是數人數、判斷條件；沒有一個把它轉成字串去查文字表（確認這些函式的呼叫；經由虛擬函式的讀取不能完全排除）。
- 共鳴介面顯示的文字來自別處：標題是被動的 `name`（`SynergyUIDataBuilder.GetName` 0xF4FD60 → `PassiveStaticData.GetName`），說明是被動的 `summary`，成員是「`[`稱號`]` 名字」，人數條件是 `BattleUIText` 的 `PassiveSynergyConditionCount`。
- 「마법 소녀」（中間有空格）在整包原文裡只出現在這個檔；被動名稱寫的是「마법소녀」。

所以這一筆照常翻譯即可，目前不影響畫面；之後的版本如果多了讀它的表，`fingerprint.py` 會在 `TextDataSet` 的欄位差異裡列出來，檔案清單也會多出它的檔名。

### 有條件才不顯示的

- 劇情檔 `id` 是 `-1` 的 25 列（21 個 `content`）。
- 劇情裡演出檔標成獨白的列，上面的 `teller`、`title`。哪些列有這個標記在演出檔裡，讀不到。
- `ScenarioModelCodes*` 裡從來沒被查過的 306 個 `name`、144 個 `nickName`（推定）。
- `StoryTheaterDanteNoteDetail*` 的 `title`、`AbEvents*` 的 `cancelDesc`、`Skills*` 的 `rawDesc`：程式會讀但條件不成立，官方也沒有資料。
- 官方檔案裡有、但永遠是空的：`IntroduceCharacter` 的 `sentence`、第 8 章特效的 `mainText`／`subText`。

## 7. 同一樣東西有兩份文字時，誰優先

1. 硬幣效果：戰鬥中的技能面板用 `summary`（空的用 `desc`）；其他地方一律 `desc`。
2. 被動：技能面板與共鳴階段說明用 `summary`（空的用 `desc`）；其他地方一律 `desc`。
3. 狀態：提示框與人格資訊一律 `desc`；技能面板用 `summary`（沒有 key 用 `desc`、空字串只顯示名稱）。
4. 同一個關鍵字：畫在文字裡的名稱與戰鬥、人格資訊的狀態介面用 `Bufs*`；點連結的說明、選技面板、增益清單用 `BattleKeywords*`。
5. 人格的顯示名稱＝`title`＋空格＋`name`。造型道具：`skinItemTitle`／`skinItemDesc` 優先於 `title`／`desc`。
6. 異想體的名稱：名牌與狀態列用 `Enemies*` 的 `name`；選定目標的詳細面板用 `AbnormalityGuides*` 的 `name`。
7. 劇情的名稱：列上的 `teller` 優先於 `ScenarioModelCodes*` 的 `name`。稱號：列上的 `title` 優先於 `nickName`，再來是內建的英文稱號。
8. 戰鬥氣泡：`BattleSpeechBubbleDlg` 優先於 `PersonalityVoiceDlg`。技能演出的台詞框：這個單位自己的 `EGOVoiceDig` 優先於共用的那一份。演出字幕：`EGOVoiceDig` 優先於 `BattleSpeechBubbleDlg`。
9. 判定之後罪人的對話框：`PersonalityVoiceDlg` 的 `choice_…` 優先於 `AbDlg` 的 `SUCCESS`／`FAILURE`。
10. 飾品說明：沒取得過的飾品顯示 `MirrorDungeonEgoGiftLockedDesc`，取得過的顯示飾品自己的 `desc`。
11. `UnlockCode`：頭目討伐的文字裡 `openConditionNumber` 優先於 `openCondition`。
12. `MirrorDungeonTheme`：有要求而且 `specialName` 有內容時優先於 `name`。
13. 但丁能力：卡片上顯示 `rawDesc`，選擇目標的彈窗顯示 `desc`。
14. 道具：提示框是 `name`＋`desc`；`flavor` 只在背包的詳細面板。

## 8. 對過去報告的修正

這次的結果與先前各章對照後，有下列出入，已直接改在各章：

| 位置 | 原本的寫法 | 修正 |
|---|---|---|
| [01](01-custom-language.md) 劇情檔的說話者名稱 | 引用 `Dialog.GetTellerName` | 那個函式沒有人呼叫；實際的程式是 `StoryPlayData` 的建構式。三條名稱規則是對的，補上稱號的規則與獨白標記 |
| [02](02-tags-ruby.md) JP 的 `ActionEvents` `name` 帶 `<ruby>` | 沒有追到顯示在哪裡 | `ActionEvents*` 的 `name` 沒有讀取者，根本不顯示 |
| [03](03-keywords.md) 欄位表 | 硬幣 `summary`、被動 `summary`、`simpleDesc` 的顯示位置寫得籠統 | 改成只在戰鬥中的技能面板；`simpleDesc` 不是單位頭上的發動文字 |
| [03](03-keywords.md) 兩份文字 | 滑過狀態圖示看到 `desc`／`summary` | 提示框只用 `desc`；`summary` 沒有任何提示框使用 |
| [04](04-other.md) 第 3 節分頁 | 六種欄位會分頁並補頁 | 分頁的與補頁的不是同一組：`behaveDesc` 會分頁但不補頁；`prevDesc` 會補頁但顯示時不分頁 |
| [04](04-other.md) 第 5 節 `ActionEvents*` | 超出範圍選項會沒有文字 | `result[j]` 超出範圍是空的；少了一個選項時按鈕上是韓文除錯字串 |
| [04](04-other.md) 第 5 節 `AbDlg_*` | `usage` 解析成事件編號與成功率 | 是「事件 id、類型」；事件 id 從不用來挑台詞 |
| [04](04-other.md) 第 5 節 `AbnormalityGuides*` | `level` 重複會丟出例外 | 不會：放進字典的函式沒有人呼叫 |
| [04](04-other.md) 第 5 節 狀態的 `summary` | 是 null 時改用 `desc` | 沒有這個 key 時改用 `desc`；空字串只顯示名稱 |
| [05](05-battle-speech-bubble.md) 相鄰的戰鬥文字 | `AbDlg_*` 是異想體事件裡的台詞 | 所有判定事件都有；判定之後先用人格語音 |
| [06](06-families.md) 第 3 節人格語音的顯示 | 大廳、結算、抽卡、戰鬥氣泡 | 還有語音清單（`desc` 是標題）、同步演出、判定事件 |
| [06](06-families.md) 第 5 節不讀的欄位、README | 只列了不是類別欄位的 key，數字含照抄的紀錄 | 改指向本章第 6 節 |
| [06](06-families.md) 讀不到的部分 | 鏡牢節點表由誰讀不明；`ChoiceEventKeyword` 的歸屬是推定 | 那張表沒有人讀；歸屬已確認 |

沒有發現出入的：[01](01-custom-language.md) 的載入流程、[02](02-tags-ruby.md) 的 ruby、[03](03-keywords.md) 的觸發規則、[05](05-battle-speech-bubble.md) 的氣泡顯示方式。

## 讀不到、沒有確認的部分

- prefab 裡的設定：技能面板文字框的寬度與字級、`flavor` 的文字物件是否存在、面板裡的關鍵字連結能不能點。
- 「戰鬥能力介面」這個設定在選項畫面上的名稱；詳細樣式的第二個條件（檢視模式 3）的確切名稱。
- 經由虛擬函式的讀取：狀態 `summary`、單位 `desc` 的虛擬呼叫掃過，沒有找到顯示它們的地方，但不能完全排除。
- 劇情的獨白標記在哪些列、哪些 `model` 會被轉換（都在演出檔與靜態資料）。
- `MirrorDungeonTheme` 的 `specialName` 由誰要求；戰鬥結算提示的完整條件。
- RPG 裡內容像開發註記的任務說明會不會被列出來。
