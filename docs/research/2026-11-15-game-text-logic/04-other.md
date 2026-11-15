# 其他會讓譯文寫法影響顯示結果的地方

依據：各 `TextData_*` 類別、`ChoiceEvent*`、`StorySystem.Dialog`、`SkillTextDataMediator`、`Util` 等（2026-11-12 版）。「確認」＝對照反組譯讀過；沒有標的是只看反編譯結果；位址是 RVA。關鍵字、標籤、ruby、載入流程各有專章，這裡不重複。

## 1. 自訂語言跑在哪一種官方語言的分支上

- `LOCALIZE_LANGUAGE`：KR＝0、EN＝1、JP＝2。沒有自訂語言的值。
- `GlobalGameManager.Lang`（0x13922B0，確認）直接回傳遊戲內選項的語言，語言包不會改它。所以**文字的底是 EN，程式的語言分支卻照玩家選的語言**。第一次啟動時，系統語言是韓文選 KR、日文選 JP，其餘（包括中文）選 EN。

找到的語言分支：

| 位置 | 依語言的差別 | 對中文的影響 |
|---|---|---|
| `TMProStringMatcher.CheckKRText`（選擇事件紀錄、玩家名片） | 只在 KR 時把 `(을/를)` 這類助詞選項換成正確的一個，而且前一個字必須是韓文字（確認） | 沒有。`(消耗的層數/10)` 這種半形括號加斜線不會被改 |
| `PersonalityTextDataManager.GetTitleWithName`（鐵道增益文字） | KR、EN：「稱號＋半形空格＋名字」；JP：不加空格 | 選 KR、EN 時稱號與名字之間有一個半形空格 |
| `UIVerticalWriteLocalize` | KR 用直排的文字物件，其他用橫排 | 選 KR 時部分標籤是直排 |
| `TextSizeByLocalize` 等 | 字級、邊界、物件顯示依語言取 prefab 裡的值 | 中文套用所選語言的字級；數值讀不到 |
| `FormationNameDataManager` | 自動產生的編隊名稱長度上限：KR 20、EN 40、JP 20 | 依所選語言 |
| 做成圖片的文字 | 只有三種語言的圖 | 照所選語言 |
| 劇情 `P10704` 的名字遮蔽 | 把 KR／EN／JP 的特定名字換成 `■■■■■` | 中文名字不會被遮蔽 |

沒有找到只針對英文的處理（複數、序數、大小寫轉換套在譯文上）。

## 2. 佔位符與 `string.Format`

譯文在這些地方是 .NET 的格式字串：`{0}`、`{1}` 會被代入，**多寫的編號或單獨的 `{`、`}` 會丟出 `FormatException`**。

| 文字 | 什麼時候格式化 | 出錯時 |
|---|---|---|
| `AbEvents*` 的 `eventDesc`、`prevDesc`、`behaveDesc` | **一律**，而且是以 0 個參數（`AppendFormat(text, 空陣列)`，確認） | 沒有 try/catch。這三個欄位裡不能有任何 `{`、`}` |
| `AbEvents*` 的 `successDesc[]`、`failureDesc[]`、`cancelDesc` | 一律 | 有 try/catch |
| `Bufs*` 的 `desc`、`summary` | 顯示狀態說明時，以層數等數值 | 沒有 try/catch |
| `EGOgift*.desc` | 該飾品有參數時 | 沒有 try/catch |
| `BuffAbilities`、`DungeonStartBuffs.description`、`RailwayDungeonBuff`、`MirrorDungeonTheme.name`、`ActionEvents.desc`、`Quest`、`BattlePassMission`、`Item` 的說明、`UnlockCode` 等 | 呼叫者有給參數時 | 沒有 try/catch |
| 介面文字（`TextData_UI.content`） | 呼叫者有給參數時；沒給參數就原樣回傳 | 沒有 try/catch |
| `AbEventsResultLog*.content` | `{n}` 的**相異個數**等於參數個數時才格式化，否則 `{n}` 原樣顯示（確認） | 沒有 try/catch |
| `MentalCondition*` 的 `add`、`min` | `{n}` 的相異個數等於條件 id 裡的數字個數時才格式化；否則這段說明不顯示，並且每次都記錯誤 | — |
| `Skills*` 的 `desc` | 介面上都不給參數；不給參數時**單一位數的 `{0}`～`{9}` 會被刪掉** | — |

譯者看不到參數個數，所以規則是：**`{n}` 的集合與 KR 完全相同，除此之外不寫任何 `{`、`}`**。現行譯文 2,087 個含 `{n}` 的字串都符合。

### 具名的 `{…}`

- `{affectedTarget}`（`ActionEvents`、`AbEvents` 的結果文字）：換成受影響角色的名字，以 `, `（半形逗號＋空格）相連。逐字比對，區分大小寫。
- `ChoiceEventEffect*.content`（`ChoiceEventResultSet.SetFormatString` 0x27BBEC0，確認）：以 `{`、`}` 切開文字，大括號裡的字是參數名（`{reward}`、`{targets}`、`{conditions}`、`{amount}`、`{stack}`、`{hpAmount}`、`{mpAmount}`、`{target}`、`{waveNum}` 等）。兩個後果：
  - 參數名拼錯或被翻譯 → **整個 `{…}` 靜靜消失**。
  - 大括號以外的每一段文字都會被拿去**當正規表示式編譯**。括號不成對、`++`、`**`、結尾是 `\` 會丟出例外。例如 `({reward})` 會出錯（切出來的兩段是 `(` 與 `)`），`（{reward}）` 沒問題。這個欄位的半形 `(`、`)`、`[`、`]`、`\`、`+`、`*`、`|` 都要小心。
- `AbEventsResultLog*.content` 的 `%…/%`：依參數裡的狀態，把這一段上色。`%` 與 `/%` 要成對，數量、順序與 KR 相同；`/%` 後面緊接的半形 `,` 或 `.` 會被一起處理，全形標點不受影響。
- `SkillTag.json` 的 `{n}`：換成標籤 id 裡的數字。
- 全大寫的 `[ATTRIBUTES]` 這類變數怎麼代入，這次沒有追。

## 3. 選擇事件的分頁

`ChoiceScrollViewBox`（0x27C5520）把文字以 **`\n\n`** 切成頁，一頁一頁按著看；**反斜線加 n 這兩個字元**（JSON 裡寫成 `\\n`）也是分頁符號。

語言包生效時（只有自訂語言會做，見 [01-custom-language.md](01-custom-language.md)），`AbEvents*` 的 `eventDesc`、`prevDesc`、`successDesc[i]`、`failureDesc[i]` 與 `ActionEvents*` 的 `desc`、`options[i].result[0]` 會跟靜態資料裡的應有頁數比較（`ChoiceEventLocalizeCheckStaticData.ReplaceText` 0x17A58A0）：

- 頁數不夠 → 後面補 `...` 當作一頁，補到夠為止；陣列太短也補。
- 頁數太多 → 不處理（推定：演出與文字會對不上）。
- 計算頁數時，空字串與剛好是 `Error` 的文字算 0 頁。

所以這些欄位的 `\n\n` 數量要與 KR 相同。現行譯文有 5 處比 KR 少：

| 檔案 | id | 欄位 | KR 頁數 → 譯文頁數 |
|---|---|---|---|
| `AbEvents_Refraction2.json` | 853901 | `failureDesc[0]` | 7 → 5 |
| `AbEvents_Mirror.json` | 90200501 | `successDesc[0]` | 6 → 5 |
| `AbEvents_Mirror.json` | 90200601 | `successDesc[0]` | 6 → 5 |
| `ActionEvents_Mirror.json` | 901024 | `options[2].result[0]` | 5 → 4 |
| `ActionEvents.json` | 10308 | `options[0].result[0]` | 4 → 3 |

## 4. 名稱裡的換行、問號與長度

- **稱號裡的 `\n` 在單行顯示的地方會變成半形空格**（`TextData_Personality.GetTitle_ExceptionNewLine` 0x13D1D80、`NameTagDisplayData`、共鳴說明等）：人格的 `title`、E.G.O 的 `name`、同伴的 `title`。所以 `\n` 旁邊不要再有空格（現行譯文有一處 `R公司 \n第四集團軍\n馴鹿隊`，會顯示成兩個空格），也要留意 `\n` 斷在詞中間時會多出一個空格。
- **罪孽型敵人的名字**（`Util.CountMatchingWords_Classification` 0x1297AC0，確認）：名字以半形 `?` 切開，去掉 `?` 後加上「N型」（N＝`?` 的個數＋1，文字來自介面的 `septempeccata_classification`）。這類名字要保留與 KR 相同個數的**半形** `?`；全形 `？` 不會被計算，會留在名字裡。現行譯文 14 個這類名字都正確。
- **技能名稱太長會被截斷**（`SkillAndCoinUI.UpdateSkillNameSize` 0x2A33390）：戰鬥中技能名稱寬度超過 455 像素時，用 `Substring` 截短再加 `...`。名稱裡有標籤的話可能被切壞，技能名稱不要放標籤。
- 狀態與關鍵字的名稱不要有 `[`、`]`：遊戲自己的檢查函式（`TextData_Buf.CheckText`）會把它列為問題，而且在技能文字裡會被當成標籤處理。

## 5. 結構欄位：照抄，不能增減

語言包的條目是整筆取代，下面這些不是文字的內容要與 KR 相同：

| 檔案 | 欄位 | 程式怎麼用 |
|---|---|---|
| `Skills*` | `levelList[].level`、`coinlist[]`、`coindescs[]` | `level` 以數值查找（找不到用較低的等級）；硬幣與硬幣說明**依位置**對應，多一個少一個都會讓說明錯到別的硬幣上 |
| `Bufs*` | `excludeAutoLink` | 見 [03-keywords.md](03-keywords.md)；可以依譯名需要自行增加 |
| `EGOgift*` | `simpleDesc[].abilityID` | 以數值查找；找不到時畫面顯示一段韓文除錯訊息 |
| `AbEvents*` | `successDesc[]`、`failureDesc[]` 的長度 | 依位置 |
| `ActionEvents*` | `options[]`、`result[]` | 依位置；超出範圍回傳空字串，選項會沒有文字 |
| `AbDlg_*` | `personalityid`、`voicefile`、`usage` | `usage` 會被解析成事件編號與成功率 |
| `AbnormalityGuides*` | `storyList[].level` | 放進字典，**同一筆裡 `level` 重複會丟出例外** |
| `BgmLyrics/*` | `id`、`vocalid` | `id` 是行號 |
| `BattleAnnouncerDlg/*` | `imgStr`、`isHide`、`playWeight` | 資料 |
| 劇情檔 | `id`、`model` | 見 [01-custom-language.md](01-custom-language.md) |
| `ScenarioModelCodes*` | `id` | 劇情的 `model` |

另外，「KR 是空字串的欄位，譯文也要是空字串；KR 有內容的，譯文也要有」：被動的 `summary`、硬幣的 `summary` 是空的時會改用 `desc`；狀態的 `summary` 是 null 時改用 `desc`。「沒有這個欄位」與「空字串」不同，沒有欄位時多半是 `Unknown`。

## 6. 換行與特殊字元

- JSON 的 `\n` 就是換行，照樣顯示。寫成兩個字元的「反斜線＋n」只有在選擇事件（分頁）與 RPG 最後一段過場會被處理，其他地方會原樣顯示出 `\n`（KR 官方有 15 處這種筆誤）。沒有 `<br>` 的處理。
- `\n` 的其他意義：技能、被動文字的同步等級比較以換行為單位（見 [02-tags-ruby.md](02-tags-ruby.md)）；但丁筆記裡 `\n-`（換行後緊接半形 `-`）是條列符號，程式會在那裡調整行距（`DantesNoteDetailInfoData.GetDescText` 0x17AD8F0）。
- 劇情 `content` 的 `//` 之後會被丟掉。
- 除了關鍵字比對之外，全形／半形空格、U+200B、`・`、`「」` 沒有被特別處理。沒有找到以譯名當字典 key 或以譯名排序的程式。
- 鐵道的增益文字與「稱號＋名字」會 `Trim()`，頭尾的空白會不見；其他地方的結尾空格都保留。

## 7. 現行譯文的比對結果

資料：`raw-20261112` 的 KR 與 `tw` 分支 7fccf1d，121,796 組字串、1,001 個劇情檔。

沒有問題的：佔位符（`{n}` 的集合）、具名 `{…}`、劇情檔結構、非字串的值、陣列長度、欄位集合、欄位是否為空、id 重複（只有 KR 自己重複的那些）。

有差異的：

- 選擇事件頁數比 KR 少：5 處（上表）。
- 同步等級比較：KR 相鄰兩級完全相同、譯文卻不同的行，技能說明 58 行、硬幣說明 7 個。很多只是空格位置不同，例如 `Skills.json` 的 1010203：一級寫「施加3層麻痺 」，另一級寫「施加3層 麻痺」。後果只是比較畫面會把沒變的行標成有變。
- 但丁筆記的條列：`StoryTheaterDanteNoteDetail.json` 的 `DanteNoteDetail_6`、`DanteNoteDetail_11`，`\n-` 的數量 KR 是 2、譯文是 3。
- 稱號的 `\n` 旁有空格：1 處（上述 `R公司 \n…`）。

## 建議的檢查

1. `{n}` 的集合與 KR 相同；沒有其他的 `{`、`}`（KR 同一欄位有的具名 `{…}` 逐字相同）。
2. `ChoiceEventEffect*.content`：大括號以外的每一段是合法的正規表示式片段（最簡單的做法：禁止 KR 沒有的半形 `(`、`)`、`[`、`]`、`\`、`+`、`*`、`|`）。
3. `AbEventsResultLog*`：`%`、`/%` 的數量與順序和 KR 相同。
4. 選擇事件的六種欄位：`\n\n` 切出的頁數與 KR 相同；沒有連續三個以上的換行；沒有「反斜線＋n」。
5. 人格 `title`、E.G.O `name`：`\n` 前後沒有空白。
6. 名字含半形 `?` 的敵人：`?` 個數與 KR 相同。
7. 技能、被動相鄰等級：KR 相同的行譯文也相同（含結尾空格）；KR 只差數字的行譯文也只差數字；行數相同。
8. `AbnormalityGuides*` 的 `storyList[].level` 不重複；所有陣列長度、非字串值與 KR 相同；KR 為空的欄位譯文為空。
9. 但丁筆記：`\n-` 的數量與 KR 相同。
10. 技能名稱、狀態名稱不含標籤與方括號。
