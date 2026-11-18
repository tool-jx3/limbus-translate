# 自訂語言包的載入流程

依據：`ProjectMoon.CustomLocalization.*`、`TextDataSet`、`JsonDataList<T>`、`TextLibrary.TextDataConverter`、`TextAssetPatch.*`、`StorySystem.StoryDataParser`／`StoryAssetLoader`、`RPGSystem.RpgLocalizationManager`、`UtilityUI.FontManagerScriptableObject`（2026-11-12 版）。「確認」＝對照反組譯讀過；「推定」＝只看反編譯結果或由一般知識推論；位址是 RVA。

## 先講結論

| 項目 | 行為 |
|---|---|
| 根目錄 | `LimbusCompany_Data/Lang`。`config.json` 放在 `Lang/` 底下，不是語言包資料夾裡 |
| 候選語言包 | `Lang/` 底下的每個子資料夾，不檢查內容 |
| 選哪一個 | `config.json` 的 `lang`；`"-"` 表示不用；值是空的或不在候選裡時，自動選第一個資料夾並改寫 `config.json` |
| 生效條件 | 資料夾存在，**而且**標題字型與內文字型都建立成功。**沒有可用字型，整個語言包（包括文字）都不會生效** |
| 遊戲內語言選項 | 不影響語言包是否生效 |
| 文字的底 | **EN，寫死的**，不管遊戲內選 KR、EN 還是 JP。語言包沒有的檔案或 id，顯示英文 |
| 版面、字級、圖片的分支 | 照遊戲內選的語言（KR／EN／JP）。自訂語言沒有自己的列舉值 |
| 檔名 | 官方檔名去掉 `KR_` 前綴。**要讀哪些檔由官方的檔案清單決定，不掃描語言包資料夾**；清單以外的檔案不會被開啟 |
| 一般文字表（72 個） | 先載入 EN，再用語言包的條目**以 id 為單位整筆取代** |
| JSON 解析 | `UnityEngine.JsonUtility`；只有劇情檔用 SimpleJSON |
| 語言包的 JSON 壞掉 | 沒有 try/catch，那一整張表（連英文的底）都不會載入。視為致命錯誤 |

## 尋找與選擇

- `CustomLocalizeManager.GetLangDataPath`（0x1C1C220）：`Application.dataPath + "/Lang"`，資料夾不存在就回傳 null。
- `GetCandidates`（0x1C1BC40）：列出 `Lang/` 的全部子資料夾名稱，前面加一個 `"-"`。
- `Get`（0x1C1C610，確認）：候選只有 `"-"` → 不用；`config.lang` 是 `"-"` → 不用；`config.lang` 在候選裡 → 用它；否則用第一個資料夾並寫回 `config.json`。**結果在整個執行期間只算一次**，登入畫面的語言包選單改了之後要重開遊戲才生效（推定：選單的確定按鈕對應的函式是空的）。
- `SearchResult.IsDataExist`（0x1C1B6F0）：資料夾存在，而且標題字型、內文字型兩個物件都還在。全程式有 23 處用它判斷「語言包是否生效」。
- `CustomLocalizeManager.IsRunning`（0x867CE0）固定回傳 true；`GetDefaultLangWhenCustom`（0x99E910）固定回傳 1（EN）。

## `config.json`

類別 `CustomLocalizeConfig`，五個欄位：

| 欄位 | 用途 |
|---|---|
| `lang` | 語言包資料夾名稱；`"-"` 表示不用 |
| `samplingPointSize` | 建立字型時的取樣字級，預設 78 |
| `padding` | 字型圖集的留白，預設 5 |
| `titleFont`、`contextFont` | **沒有任何程式讀它們**。用哪個字型完全由資料夾決定 |

- 讀取：`JsonUtility.FromJson`，UTF-8。檔案不存在時用預設值並寫出一份。**格式壞掉時沒有 try/catch**，例外會從遊戲啟動流程丟出去（推定：遊戲無法正常啟動）。
- 遊戲會寫回 `config.json`（縮排格式，五個欄位都寫）：檔案不存在時、`lang` 不在候選裡時、玩家在登入畫面選了語言包時。
- 只寫 `{"lang": "zh-TW"}` 時，其他欄位是預設值 78 與 5（推定：依 `JsonUtility` 會先執行建構子的一般行為；本專案 Release 的 `config.json` 就是這樣寫）。

## 字型

`CustomLocalizeManager.GetByLastSelected`（0x1C1CA20）與 `TryLoadFont`（0x1C1D540），確認：

1. 只認路徑裡含有 `.ttf` 或 `.otf` 的檔案，**區分大小寫**（`.TTF`、`.ttc` 不行）。
2. 依序找 `Font/`（共用）、`Font/Title/`（標題）、`Font/Context/`（內文），各取第一個載入成功的檔案。
3. 標題與內文都沒有 → 兩者都用 `Font/` 的共用字型；只有一個 → 另一個也用它。
4. 用 `TMP_FontAsset.CreateFontAsset` 建成動態字型：SDFAA、2048×2048 圖集、可多張圖集。字是用到才畫進圖集，沒有預先建好的字表。
5. 套用方式（`FontManagerScriptableObject.SetFallbackFontsByLanguage` 0x1AD7CD0、`TextMeshProLanguageSetter.UpdateTMP` 0x1ADF810）：語言包字型插在遊戲字型的**備援清單第一位**，再把遊戲主字型裡**所有字母與數字**（`char.IsLetter`、`char.IsDigit`：拉丁字母、數字、韓文、假名、漢字都算）移除，讓它們改由語言包字型顯示。名稱含 `BebasKai`、`ExcelsiorSans` 的字型不動。

結果：

- 文字與數字都用語言包字型。
- **標點與符號**（括號、「…」、「—」、全形標點等）先找遊戲主字型（依遊戲內選的語言），那個字型沒有才用語言包字型。所以同一份譯文，玩家選 KR、EN、JP 時，標點的字形可能不同。
- 用 `BebasKai`、`ExcelsiorSans` 的標籤（多半是數字、英文標題）維持原字型。
- 語言包字型缺字時，繼續往後找其他語言的字型與備援字型；全部都沒有時怎麼顯示，是 TextMeshPro 的行為，沒有查。

本專案的 Release 把同一個字型放在 `Font/Title/` 與 `Font/Context/`，符合上述規則。`tw` 分支本身沒有字型，單獨把 `tw` 分支的內容放進遊戲不會生效。

`UniqueCharacterScanner`（掃描語言包的 JSON、收集用到的字元）與 `EstimateAtlasSize` 在這一版沒有任何呼叫者。

## 文字檔怎麼載入

### 檔名與路徑

`TextDataSet.LoadRemote`（0x13C2C40）向 `TextDataConverter.GetRemoteLocalizeFileList`（0x1ACFB40）拿到官方的檔案清單 `RemoteLocalizeFileList`：77 個欄位，每個欄位是一串檔名（`skill`、`passive`、`buf`、`keyword`、`egoGifts`……）。每張文字表依自己的欄位去讀檔。

- 官方路徑：`<語言>/<LANG>_<名稱>.json`，例如 `en/EN_Skills.json`。
- 語言包路徑：`Lang/<包>/<名稱>.json`；有子資料夾的是 `Lang/<包>/<子資料夾>/<名稱>.json`（`PersonalityVoiceDlg`、`BattleAnnouncerDlg`、`EGOVoiceDig`、`BgmLyrics`、`StoryData`、`RPGSystem`）。

`Skills.json`、`Skills_personality-01.json`、`Bufs-walpu10.json` 這些分開的檔案，都是照官方清單列舉的，三種語言共用同一份清單。**語言包不能新增檔名**；清單以外的檔案（包括 `Info/`、`Font/OFL.txt`）遊戲不會讀。

**清單的內容（2026-11-12 版，2026-11-18 取得）。** `RemoteLocalizeFileList.json` 在遊戲下載的在地化資料夾裡，和 `kr`、`en`、`jp`、`etc` 同一層；它不在 `raw` 分支裡。用 [`method/filelist.py`](method/filelist.py) 和 KR 的 2,449 個檔比對：

- 77 張表、1,259 個檔名（不重複 1,255 個）。劇情（`StoryData/` 1,001 個檔）與 RPG（`RPGSystem/` 189 個檔）由各自的載入器讀，不在清單裡。
- **KR 有 17 個檔不在任何一張表裡，遊戲不會載入**：`StoryTheaterPersonality`（292 筆）、`StoryTheaterMirrorWorldStoryTitle`（183 筆）、`StoryTheaterMirrorWorld`（26 筆）、`DungeonStartBuffs`（25 筆）、`DungeonStartBuffs_MD6`（44 筆）、`SynergyKeyword`（1 筆）、`Announcer-a1c9p1`、`Announcer-a1c10p1`、六個 `UserTicket-{L,R,EGOBg}-{a1c9p3,a1c10p1}`，以及三個空檔（`ChapterBannerConfig`、`Items-BossRaid`、`Skills_Enemy-a1c9p2`）。合計 594 個有內容的字串。細節見 [07-field-display.md](07-field-display.md) 第 6 節。
- 清單裡有 17 個檔名 KR 沒有對應的檔（例如 `Egos-a1c9p2`、`UserBanner-a1c7p3`）；沒有檔就跳過，沒有影響。EN 少 18 個、JP 少 7 個。
- `BattlePass_Banner`、`EventLCBCheckupText` 在介面表裡各列了兩次；`DungeonArea1-1`、`DungeonArea1-2` 同時列在介面表與地牢區域表，兩張表各讀一次。
- 同一張表裡不同檔案的重複 id 只有 3 個（`Announcer` 與 `Announcer-a1c10p3`，內容相同）。語言包以清單裡排在後面的檔為準。

現行譯文各資料夾的檔名集合與 KR 完全相同（根目錄 964、`StoryData` 1,001、`PersonalityVoiceDlg` 197、`RPGSystem` 189、`BattleAnnouncerDlg` 67、`BgmLyrics` 17、`EGOVoiceDig` 14）。

### 底是 EN

`TextDataConverter.InitRemote<T>`（共用實作 0x3BC1EE0，確認）：語言包生效時，把要載入的官方語言**直接改成 1（EN）**，再另外準備一個「讀語言包」的函式。劇情（`StoryDataParser.GetScriptText` 0x20B2B40）與 RPG（`RpgLocalizationManager`）也一樣。

例外是啟動時就要用的四張表（登入介面、檔案下載說明、戰鬥提示、錯誤代碼訊息；`TextDataSet.LoadLocal` 0x13C27B0）：底是遊戲內選的語言，語言包同樣以 id 覆蓋。這四張表的官方文字放在遊戲本體的 Resources 裡，每個語言五個檔：`BattleHint`、`FileDownloadDesc`、`LoginUIText`、`UserAgreements`、`ErrorCodeMsgLocal`（資源目錄 `catalog_S1.json` 的 `Localize/<語言>/…`，確認）。前四個在 `raw` 的 KR 裡也有同名的檔，所以語言包有翻；`ErrorCodeMsgLocal` 只在遊戲本體裡，`raw` 沒有；沒有這個檔時啟動階段的錯誤訊息會是遊戲內選的語言（推定），所以建置時用 `ErrorCodeMsg.json` 複製一份放進語言包（[10-assets.md](10-assets.md) 第 2 節）。哪個檔對到哪張表寫在 `Localize/LocalizeFileList`（同樣在本體的 Resources）：登入介面讀 `LoginUIText`、`UserAgreements`、`ErrorCodeMsgLocal`，錯誤代碼訊息只讀 `ErrorCodeMsgLocal`（[10-assets.md](10-assets.md) 第 2 節）。

### 合併規則：一般文字表

`JsonDataList<T>.SetDictionary`（共用實作 0x43088D0，確認）：先把 EN 的條目以 id 放進字典（id 重複時先到的留下，並記一筆錯誤），再把語言包的條目逐一 `字典[id] = 條目`（直接覆蓋）。

| 情形 | 結果 |
|---|---|
| 語言包沒有這個檔 | 跳過，整個檔顯示英文 |
| 語言包有這個檔，但少了某個 id | 那個 id 顯示英文 |
| 語言包的條目少了某個欄位 | **整筆條目被語言包的取代，少掉的欄位是型別的預設值**，不會去拿英文的。字串欄位的預設值多半是 `Unknown`（見下） |
| 語言包多出 id | 加進字典，遊戲用得到才有作用 |
| 語言包裡同一個 id 出現兩次（同檔或同一張表的不同檔） | 後面的蓋掉前面的，不會記錯誤 |
| EN 沒有這個檔，語言包有（只有 KR 有的內容） | 語言包的條目照樣載入 |
| 語言包的檔是 0 bytes | 跳過 |
| 語言包的 JSON 格式錯誤 | `JsonUtility.FromJson` 丟出例外，沒有人接。這張表的載入流程中斷，**整張表是空的（英文也沒有）**，載入計數也不會歸零。玩家看到什麼沒有追 |
| 讀檔編碼 | UTF-8，會偵測並去掉 BOM |

所以語言包的每一筆條目都要**欄位齊全**：不翻譯的欄位（`id`、數值、布林、`excludeAutoLink` 等）也要照抄。現行譯文與 KR 逐筆比對，沒有少 id、少欄位或多欄位。

### 找不到 id 時

`JsonDataList<T>.GetData(id)`（共用實作 0x43090F0，確認）：字典裡沒有這個 id 時，**新建一筆空的條目放進字典再回傳**，並記一筆 `:: cannot find id`。它從不回傳 null。空條目的欄位是建構子給的預設值，多數類別是字串 `Unknown`（`TextData_UI.content`、`TextData_Buf` 的 `name`／`desc`／`summary`、`TextData_SkillTag.name`、`TextData_EgoGift`、`TextData_BattleKeywords` 等；`BattlePassMission` 是 `UNKNOWN`；`ActionEvents` 是空字串）。

所以畫面上出現 `Unknown`，通常代表「這個 id 在 EN 與語言包都找不到」或「語言包的條目少了這個欄位」。

### 合併規則不同的表

| 檔案 | 規則 | 依據 |
|---|---|---|
| `PersonalityVoiceDlg/*`、`BattleAnnouncerDlg/*`、`BgmLyrics/*` | **整個檔取代**：語言包的檔有至少一筆條目就整個用它，缺的 id 不會補英文；語言包的檔是空的才用 EN | `PersonalityVoiceJsonDataList.<Init>b__0` 0x13C7EB0（確認），另兩個相同（推定） |
| `AbDlg_*.json` | 整個檔取代。非文字欄位（如 `usage`）會被程式解析，要照抄 | `AbEventCharDlgRoot.SetData` 0x13CB220 |
| `EGOVoiceDig/*` | 逐筆取代，但**只取代 EN 檔裡有的 id**；EN 沒有這個檔時語言包的內容被忽略 | `EGOVoiceJsonDataList.SetData` 0x13C8500 |
| `StoryData/*` | 整個檔取代；語言包沒有這個檔就用 EN 的 | `StoryDataParser.GetScriptText` 0x20B2B40（確認） |
| `RPGSystem/*` | 以 key 為單位覆蓋在 EN 上；語言包沒有的檔靜靜跳過 | `RpgLocDomain<T>.RegisterFile` 0x4B59410（確認） |

語音與歌詞的字典以**檔名最後一段 `_` 之後的字**當 key，檔名必須與官方完全相同。

## 劇情檔

`StoryDataParser.GetScenario`（0x20B26C0，確認）：

- 語言包的劇情檔（`dataList`）與遊戲內的演出檔**依陣列位置一一配對**：第 i 筆台詞配第 i 筆演出（背景、CG、音效、立繪、語音）。
- `id` 是 -1 的列會跳過不顯示，但**仍然佔一個位置**。
- 所以劇情檔的列數與順序必須與官方完全相同，包含 `id: -1` 的註解列。增、刪、對調任何一列，之後的演出都會錯位。
- 從語言包讀的欄位只有 `id`、`place`、`model`、`teller`、`title`、`content`；其他都來自演出檔。
- **`content` 裡 `//` 之後的文字會被丟掉**（`Dialog` 建構子 0x209B990 取 `Split("//")` 的第一段；官方用它寫註解，例如 `SE // 와장창!`）。
- 說話者名稱與稱號（`StoryPlayData` 的建構式 0x2139280，確認；2026-11-16 修正：原本引用的 `Dialog.GetTellerName` 沒有人呼叫）：
  - `model` 有值：名稱是列上的 `teller`，空的話用 `model` 去查名稱；稱號是列上的 `title`，空的話用 `model` 去查稱號。**兩者各自決定。**
  - `model` 是空的：`teller` **與 `title` 都有值**才顯示，否則名稱與稱號都不顯示。
  - `model`、`teller`、`title` 全空的列當成旁白樣式；演出檔把這一列標成獨白時也是，列上寫的名稱與稱號不顯示。把原本空的欄位填上、或把有值的清空，會改變呈現方式。
  - 各欄位顯示的時機見 [07-field-display.md](07-field-display.md) 第 2 節。
- 語言包生效時，用 `model` 查名稱查的是**語言包的 `ScenarioModelCodes*.json`**（`name` 是名稱、`nickName` 是稱號；`StoryAssetLoader.GetTellerName` 0x20AFCD0），不是遊戲內建的 KR／EN／JP 名稱。`model` 是韓文代碼，同時是立繪、名牌顏色的 key，**不可以翻譯**。
- 劇情檔用 SimpleJSON 解析，格式錯誤時的行為沒有查。

現行譯文 1,001 個劇情檔的列數、`id` 順序、`model` 都與 KR 相同。

## 只對自訂語言做的事

1. **選擇事件的頁數補齊**（`TextData_AbnormalityEvents.CheckData` 0x13C9DB0、`TextData_ActionEvents.CheckData` 0x13CDCB0）。事件文字以 `\n\n` 分頁，一頁一頁按著看。語言包生效時，遊戲拿靜態資料裡的「應有頁數」來比：頁數不夠就在後面補上內容為 `...` 的頁（陣列長度不夠也補），太多則不處理。官方語言不做這個檢查。詳見 [04-other.md](04-other.md)。
2. 劇情的說話者名稱與稱號改從語言包的 `ScenarioModelCodes` 讀（見上）。
3. 畫面上顯示「自訂語言」的標示物件；劇情與戰鬥的設定按鈕會換顏色。
4. 部分把文字做成圖片的按鈕（朝聖活動的確定、取消）改成用文字標籤顯示；戰鬥通行證的標題用「非 KR」那一組物件。
5. 字型的處理（見上）。

## 語言包改不了的東西

- 做成圖片的文字（橫幅、標誌、部分按鈕）：只有 KR／EN／JP 三種，照遊戲內選的語言顯示。
- 依語言調整的字級、邊界、物件顯示（`TextSizeByLocalize`、`TextMarginByLocalize`、`ShowObjectByLocalize`）：照遊戲內選的語言。數值在 prefab 裡，讀不到。
- 伺服器傳來的訊息只有 KR／EN／JP 的字串（推定）。
- 語言包沒有的內容一律是英文。

## 讀不到、沒有確認的部分

- 2026-11-18 已解決：遊戲用**檔案模式**載入官方文字（`TextAssetPatchConfig` 的設定值是 1，確認），啟動用檔案清單的內容也讀到了。兩者與各檔的下載網址見 [10-assets.md](10-assets.md)。
- `JsonUtility` 對不嚴格的 JSON（註解、結尾多的逗號、型別不符）容忍到什麼程度：那是 Unity 的原生程式。官方檔有把數字寫進字串欄位（`"id": 0`）的情形，表示這一種可以；其他不要依賴。
- 某張表載入失敗後玩家實際看到什麼。

## 建議的檢查

1. **JSON 可解析**，根是物件、有 `dataList`；每個檔案的檔名（去掉前綴後）必須在 KR 的檔名集合裡，不多不少。
2. **每筆條目的欄位集合與 KR 相同**；非字串的值（數字、布林、陣列長度）與 KR 相同；`id` 的集合與 KR 相同、檔內不重複。
3. **KR 有、EN 沒有的 id 或檔案**：語言包一定要有，否則會顯示 `Unknown` 或整段消失（EN 不能當底）。反過來，語言包缺的內容會顯示英文，不是韓文。
4. 整個檔取代的資料夾（語音、播報員、歌詞、`AbDlg_*`、劇情）：每個檔的條目數與 id 要與 KR 完全相同。
5. 劇情檔：列數與 `id` 順序與 KR 相同；`model` 與 KR 逐字相同；`teller`、`title` 是否為空與 KR 相同；`content` 不含 `//`（KR 同一列也有的除外）；每個用到的 `model` 在語言包的 `ScenarioModelCodes*.json` 有條目。
6. 值不可以剛好是 `Unknown`（不分大小寫）或 `Error`：程式把它們當成「沒有資料」。
7. Release 的字型檔副檔名是小寫 `.ttf` 或 `.otf`；`config.json` 是合法 JSON。
