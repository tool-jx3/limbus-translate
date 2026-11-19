# 資源檔裡的設定：下載網址、檔案清單、樣式表、換行規則、靜態標籤

2026-11-18。前面九章是從遊戲程式讀的；這一章讀的是**遊戲的資料檔**（2026-11-12 版）：

- 遊戲下載的資料：`RemoteLocalizeFileList.json`、`etc/VoiceTable.json`、`LocalizePatchInfo.json`、RPG 的檔案清單、Addressables 目錄（`catalog_S1.json` 與 `catalog_S1.bin`，見下面的補正）。
- 遊戲安裝資料夾 `LimbusCompany_Data` 裡的 `resources.assets`、`globalgamemanagers.assets`、`globalgamemanagers`，以及九個場景檔 `level0`～`level8` 與 `sharedassets0`～`8.assets`。

「確認」＝直接從檔案讀到；「推定」＝有依據但沒有直接讀到。這些檔案都不放進版控，讀法見 [method/](method/README.md) 的 `filelist.py`、`catalog.py`、`unityres.py`。

## 先講結論

1. **官方文字是下載後以檔案讀取的**（設定值確認），每個檔都有固定的網址；`RemoteLocalizeFileList.json`、語音表、RPG 檔案清單都能直接下載。
2. **官方檔案清單**：77 張表、1,259 個檔名；KR 有 17 個檔不在清單裡，遊戲不會載入（細節在 [01](01-custom-language.md)、[07](07-field-display.md) 第 6 節）。RPG 的清單列了 189 個檔，和 KR 的 `RPGSystem/` 完全一致。
3. **啟動用清單**有四張表、五個檔；其中 `ErrorCodeMsgLocal` 不在 `raw` 裡，語言包沒有它，啟動階段的 16 句錯誤訊息推定不會是中文。這 16 句的 id 與文字和 `ErrorCodeMsg` 完全相同；2026-11-18 起建置時會把建好的 `ErrorCodeMsg.json` 複製成 `ErrorCodeMsgLocal.json` 放進語言包（`lct.yaml` 的 `build.copies`）。
4. **樣式表**有 31 個樣式。`highlight` 與 `upgradeHighlight` 是空的，顏色是程式另外加的；`dkr`／`den`／`djp` 系列是**切換到該語言字型專用的材質**，不是單純的顏色。
5. **換行規則**是 TextMeshPro 的避頭尾字元表：開頭括號、`—`、`…` 後面不斷行；結尾括號與 `、。，！？：；` 前面不斷行。
6. **靜態標籤**：全部檔案裡有 5,246 個標籤直接綁定介面文字的 id（1,468 個不同的 id）。估算譯文寬度後，只有 3 個可能超出框。
7. **戰鬥氣泡**的文字框是 350×100，自動縮字 10～25；**劇情台詞**的框寬 1825、字級 45、不縮字，一行約 41 個全形字。譯文估算都放得下，而且比韓文寬鬆。**演出字幕不換行也不縮字**，斷行只能靠 `\n`。

## 1. 下載設定與網址

`resources.assets` 裡的 `TextAssetPatchConfig`（確認）：

| 欄位 | 值 |
|---|---|
| `_patchMode` | 1 |
| `_localizeLoadMode` | 1。[01](01-custom-language.md) 讀到的程式是「0 走 Addressables，其他走檔案」，所以是**檔案模式** |
| `_localizeDataPath` | `Assets/Resources_moved/Localize` |
| `_remoteUrlLocalize` | `https://downloadcommon.limbuscompanycdn.org/<版本代碼>`，版本代碼就是 `raw` 的 `meta/source.json` 記的 `token`（這一版是 `l20261112_…`） |
| `_localizePatchInfoFilePath` | `Assets/LocalizePatch/LocalizePatchInfo.json` |

Addressables 目錄也印證這一點：裡面沒有官方文字檔。

> **2026-11-19 補正**：這一句原本寫「21,880 個資源裡沒有官方文字，只有啟動用的 16 個項目」，數字來自 `catalog_S1.json`。那個檔是 2026-05 版留在快取裡的舊目錄（裡面的網址是 `s20260521_…`，共 1,369 個 bundle）；遊戲現在用的是二進位格式的 `catalog_S1.bin`（網址 `s20261112_…`，3,193 個 bundle），所以那兩個數字不是這一版的。
> 對現行目錄只做了字串掃描，沒有完整解析它的格式：裡面沒有 `KR_`／`EN_`／`JP_` 開頭的文字檔名，語言包根目錄與各子資料夾的 1,448 個檔名也只有 2 個同名（`Ch8TextEffectSettings.json`、`Ch8VideoEffectSettings.json`）。有 981 個 `<劇情代號>.json` 與劇情檔同名，推定是 [01](01-custom-language.md) 說的、和台詞逐筆配對的演出資料，不是文字。結論不變。
> 現行目錄另外指出靜態資料包的位置，劇情檔的順序就是從那裡讀的，見 [11-story-order.md](11-story-order.md)。

實際下載確認過的網址（`<base>` 是上表的 `_remoteUrlLocalize`）：

| 檔案 | 網址 |
|---|---|
| 所有文字檔的清冊（雜湊與大小） | `<base>/Assets/LocalizePatch/LocalizePatchInfo.json` |
| 官方檔案清單 | `<base>/Assets/Resources_moved/Localize/RemoteLocalizeFileList.json` |
| 語音表 | `<base>/Assets/Resources_moved/Localize/etc/VoiceTable.json` |
| RPG 檔案清單 | `<base>/Assets/Resources_moved/Localize/rpg-loc-file-list-a1c10p3.json` |
| 各語言的壓縮檔（`lct fetch` 用的） | `<base>/Assets/LocalizePatch/localize_<kr|en|jp>.zip` |

- `LocalizePatchInfo.json` 列了 7,318 個檔：KR 2,449、EN 2,406、JP 2,460，加上上面三個不分語言的檔。KR 的數量和 `raw-20261112` 相同。
- RPG 檔案清單的檔名帶有章節代號（`a1c10p3`），之後的版本可能改名；清冊裡找得到實際的名稱。
- 清冊裡還有一組加密金鑰，是給加密的靜態資料用的，和文字檔無關；這份報告與腳本都不記錄它。

## 2. 檔案清單

**主要清單**（`RemoteLocalizeFileList.json`）的比對結果寫在 [01-custom-language.md](01-custom-language.md)「檔名與路徑」與 [07-field-display.md](07-field-display.md) 第 6 節：17 個 KR 檔不會被載入；[06-families.md](06-families.md) 推出來的「哪個檔在哪張表」全部正確。

**RPG 清單**：十個類別（`ui` 3、`location` 22、`npc` 52、`dialogue` 32、`dialogueChoice` 21、`narration` 14、`item` 3、`quest` 23、`swarmMob` 15、`player` 4），共 189 個檔，和 KR 的 `RPGSystem/` 一個不多、一個不少。

**啟動用清單**（`resources.assets` 的 `Localize/LocalizeFileList`，確認）：

| 表 | 檔案 |
|---|---|
| `loginUI` | `LoginUIText`、`UserAgreements`、`ErrorCodeMsgLocal` |
| `fileDownloadDesc` | `FileDownloadDesc` |
| `battleHint` | `BattleHint` |
| `errorCodeMsg` | `ErrorCodeMsgLocal` |

- 這五個檔的官方文字在遊戲本體裡（每個語言一份）。`LoginUIText`、`UserAgreements`、`FileDownloadDesc` 和 `raw` 的 KR 完全相同；`BattleHint` 的筆數相同、內容不同（本體裡的是舊版）。
- **`ErrorCodeMsgLocal` 不在下載的資料裡，所以 `raw` 沒有、語言包也沒有。** 它有 16 筆（伺服器錯誤、維護中、連線錯誤等），每一筆的 id 與 KR 文字都和 `ErrorCodeMsg.json` 的同一筆相同，其中 14 筆也在 `LoginUIText.json`。
- 影響（推定）：遊戲啟動、還沒下載完資料之前跳出的錯誤訊息，會是遊戲內選的語言。
- 處理：建置語言包時把建好的 `ErrorCodeMsg.json` 整個複製成 `ErrorCodeMsgLocal.json`（`lct.yaml` 的 `build.copies`）。`ErrorCodeMsg.json` 有 20 筆，多出的 4 筆啟動階段用不到，依合併規則無害。**「語言包放了這個檔遊戲就會讀」是從載入程式推定的，還沒有在遊戲裡確認**；確認方式是斷網啟動遊戲，看錯誤訊息是不是中文。

## 3. 樣式表（`<style="…">`）

`resources.assets` 的 `Default Style Sheet`，31 個樣式（確認）。和譯文有關的：

| 樣式 | 開頭 | 結尾 |
|---|---|---|
| `highlight`、`upgradeHighlight`、`Normal` | （空） | （空） |
| `highlightConverted`、`yellow` | `<color=#f8c200>` | `</color>` |
| `railwayDungeonAlly` | `<color=#009f20>` | `</color>` |
| `railwayDungeonEnemy` | `<color=#990000>` | `</color>` |
| `number_highlight` | `<font="ExcelsiorSans SDF"><material="ExcelsiorSans/ExcelsiorSans SDF UnderLine"><color=#f8c200>` | `</color></material></font>` |
| `bebaskai_underline` | `<font="Common/p)bebas_kai/Bebaskai SDF"><material="…BebasKai SDF UnderLine">` | `</material></font>` |
| `OrangeMark` | `<color=#ff6000><mark color=#ff000040><b><u>` | `</color></mark></b></u>` |
| `DKR`、`DKR_CP6_1`、`DKR_CP6_2`、`DKR_CP9_1` | `<material="KR/p)SCDream(light)/SCDream5 RedDarkGlow…">` | `</font>` |
| `DEN`、`DEN_CP6_1`、`DEN_CP6_2`、`DEN_CP9_1` | `<material="EN/title)mikodacs/Mikodacs SDF RedDarkGlow…">` | `</font>` |
| `DJP`、`DJP_CP6_1`、`DJP_CP6_2`、`DJP_CP9_1` | `<material="JP/title)corporate logo(bold)/Corporate-Logo-Bold-ver2 SDF RedDarkGlow…">` | `</font>` |

其餘是 TextMeshPro 內建的範例樣式（`H1`、`H2`、`H3`、`Quote`、`Link`、`Title`、`C1`～`C3`），官方文字沒有用到。

- **`highlight` 與 `upgradeHighlight` 在樣式表裡是空的**。畫面上的黃字是程式把它們換成 `highlightConverted` 之後才有的（[02-tags-ruby.md](02-tags-ruby.md)：程式用完全相同的字串比對 `<style="highlight">`）。所以這兩個樣式寫法只要差一個字元，就既不會被程式換掉、樣式表也不給顏色，結果是沒有任何效果。
- `number_highlight` 會換成另一個字型（`ExcelsiorSans`）。它是英數字型，推定沒有中文字，包在裡面的應該只有數字與符號。
- **`dkr`／`den`／`djp` 是材質，不是顏色**：開頭是 `<material>`，指向該語言某一個字型的發光材質；結尾寫的是 `</font>`。`_CP6_1`、`_CP6_2`、`_CP9_1` 是不同章節的發光樣式。
  - 遊戲的 TextMeshPro 處理 `<material>` 時，從 `Font/` 加上名稱去載入材質，**不檢查它和目前的字型是否相符**（`TMP_Text.ValidateHtmlTag` 0x6E3BD10–0x6E3BE6D，確認）。
  - 語言包用自己的字型時，這個材質實際畫出來是什麼樣子（有發光、沒有效果、還是字形錯亂），取決於語言包字型是怎麼接到文字元件上的，這一段沒有讀，**沒有確認**。現行指引要求譯文一律寫 `djp…`，建議在遊戲裡找一句有這個樣式的歌詞實際看一次。
- 樣式名稱大小寫都有人用（`dkr` 與 `DKR_CP9_1`），官方資料兩種寫法都有效；依 TextMeshPro 的一般行為，名稱比對不分大小寫（這一點不是從這一版的程式讀的）。

## 4. TextMeshPro 的設定與換行規則

`resources.assets` 的 `TMP Settings`（確認的部分）：材質與字型的預設路徑是 `Font/`，圖示集是 `Sprite Assets/`；預設字級 36，自動縮字的預設比例是 0.5 到 2 倍。

換行規則是兩份字元表（確認內容；作用依 TextMeshPro 的一般行為）：

- **後面不能斷行的字元**（`LineBreaking Leading Characters`）：`([｛〔〈《「『【〘〖〝‘“｟«$—…‥〳〴〵\［（{£¥"々〇＄￥￦ #`
- **前面不能斷行的字元**（`LineBreaking Following Characters`）：`)]｝〕〉》」』】〙〗〟’”｠»`、日文的小假名與長音、`々〻‐゠–〜?!‼⁇⁈⁉・、%,.:;。！？］）：；＝}¢°"†‡℃〆％，．`

對譯文的意義：

- 中文字之間可以在任何位置自動換行，但不會把 `「『（《` 留在行尾，也不會把 `」』）》、。，！？：；` 放到行首。這是遊戲自動處理的，譯文不用為了避頭尾手動加 `\n`。
- `—` 與 `…` 的後面不斷行：`……` 會和後面的第一個字連在一起換行。
- 全形的 `～`（U+FF5E）與間隔號 `·`（U+00B7）不在表裡，可能出現在行首；`〜`（U+301C）與 `・`（U+30FB）在表裡。
- 譯文用的 `「」『』（）《》【】` 都有涵蓋。

## 5. 靜態標籤：哪個介面物件顯示哪個文字 id

介面上固定不變的字（按鈕、標題、欄位名稱）是由 `UITextDataLoader` 元件依 `key` 從介面文字表取字（確認元件的欄位）。本體的資源檔與九個場景檔合計：

- 27,428 個文字元件（`TextMeshProUGUI`），其中 5,246 個掛了 `UITextDataLoader`，共 1,468 個不同的 `key`。最多的是 `resources.assets`（1,862 個）、`sharedassets2.assets`（1,258 個）、`level7`（841 個）、`level8`（594 個）、`level5`（488 個）。
- 5,246 個標籤裡有 5,205 個的 `key` 在介面文字表（清單 `ui` 的檔案）或啟動用的表找得到。
- 文字元件的設定（5,246 個標籤）：自動縮字 3,575 個；會自動換行 3,929 個、不換行 1,316 個；超出時直接溢出 5,065 個、截斷 176 個、加刪節號 4 個。
- 完整的對照表用 `python3 -I method/unityres.py <LimbusCompany_Data> labels <檔案…>` 產生：檔案、`key`、物件路徑、字級、縮字範圍、換行、溢出方式、固定寬度。

**譯文會不會超出框**：取「不換行、寬度固定」的 555 個標籤，以「全形字 1 個字寬、半形字 0.55 個字寬」乘上字級（有自動縮字的用最小字級）估算。譯文估計超出框、而且比韓文寬的只有 3 個 `key`：

| `key` | 位置 | 框寬／字級 | KR | 譯文 |
|---|---|---|---|---|
| `equal_or_more` | 獎勵地牢的建議等級（`sharedassets2.assets`） | 71／35 | `이상` | `或以上`（估 1.48 倍） |
| `connect_to_google_account` | 設定的帳號連結按鈕 | 289／50 | `Google 연동` | `綁定Google帳號`（估 1.26 倍） |
| `connect_to_apple_account` | 同上 | 311／50 | `Apple 연동` | `綁定Apple帳號`（估 1.08 倍） |

這個估算的限制：

- 字寬是估的，語言包字型的實際寬度沒有量。
- 2,402 個標籤的寬度跟著上層物件伸縮，2,247 個會自動換行，這兩類沒有估（換行的要看高度）。
- 只涵蓋靜態標籤。程式動態設定的文字（技能說明、提示框）不在內；氣泡與劇情對話框見下一節。

## 6. 戰鬥與劇情的文字框（場景檔）

戰鬥場景是 `level3`，劇情場景是 `level4`。下面的數字用 `python3 -I method/unityres.py <LimbusCompany_Data> tmp level3 <路徑的正規表示式>` 讀出（確認）；「一行幾個字」是用框寬除以字級算的，全形字當 1 個字寬。

### 戰鬥氣泡（`BattleDialogUI` 的 `tmp_dialog`）

| 項目 | 值 |
|---|---|
| 文字框 | 350×100；氣泡本身 400×150 |
| 字級 | 自動縮字，10～25（元件上的 30 不生效） |
| 換行 | 自動換行，置中 |
| 超出框 | 直接溢出，不截斷、不加刪節號 |
| 各語言的設定 | 文字物件上有 `TextMeshProLanguageSetter`，但它每個語言只記字型與材質，**沒有字級** |

- 25 級時一行 14 個全形字、約 3 行（行高以 1.2 倍字級估）；字多就縮小，縮到 10 級最多約 280 字。
- 以這個算法估 `BattleSpeechBubbleDlg*` 的 1,555 句：

| 估計字級 | 譯文 | KR |
|---|---|---|
| 25（最大） | 1,533 | 1,521 |
| 20～25 | 21 | 32 |
| 15～20 | 1 | 2 |
| 15 以下、放不下 | 0 | 0 |

- 譯文最小的是 `9SV-BAT17-01`（75 字，19 級）；KR 同一句 111 字，16.5 級。譯文沒有比 KR 更擠的句子。
- 人格語音（`PersonalityVoiceDlg`）的戰鬥台詞也顯示在同一個氣泡，沒有另外估。

### 演出字幕（`OutterGradiantEffectController`）

| 物件 | 字級 | 換行 | 縮字 |
|---|---|---|---|
| 下方 `_dialogText` | 35 | 不換行 | 不縮 |
| 中間 `_dialogText_Mid` | 35 | 不換行 | 不縮 |
| 上方 `_dialogText_Upper` | 25 | 不換行 | 不縮 |
| 技能演出的台詞框 | 35 | 不換行 | 不縮 |

- 三個欄位各指向哪個物件是從元件的資料確認的。**這裡的文字只在 `dlg` 的 `\n` 換行**，一行太長就往兩邊溢出。
- 畫布依螢幕縮放，參考解析度 1600×720。依 Unity 的一般行為換算，16:9 的螢幕上一行約 **36 個全形字**就到螢幕寬（上方的 25 級約 51 個）。這個換算是推定，沒有在遊戲裡量過。
- 官方 JP、EN 在長句裡會自己加 `\n`（例如 `EGOVoiceDig` 的 `battle_awaken_20209_1`），KR 沒有加。
- 單行寬度超過 36 個全形字的句子：

| 檔案 | 譯文 | KR | 只有譯文超過 |
|---|---|---|---|
| `BattleSpeechBubbleDlg*`（1,555 句） | 31 | 47 | 0 |
| `EGOVoiceDig/*`（377 句） | 10 | 21 | 2 |

- 只有譯文超過的 2 句：`battle_erosion_20908_1`（譯文 42 字，KR 27 字，譯文明顯比原文長）、`battle_awaken_20209_1`（譯文 37 字，KR 36 字）。
- **哪些句子真的走演出字幕**由演出資料決定，讀不到，所以上表是候選，不是確定會超出。KR 自己也有幾十句超過，表示多數長句其實是顯示在會換行的氣泡裡。

### 劇情對話框（`StorySystem.DialogController`）

畫布的參考解析度是 3000×1350。

| 物件 | 框 | 字級 | 換行 | 一行約 |
|---|---|---|---|---|
| 台詞 `tmp_serif` | 1825×149 | 45，元件本身不縮字，字距 −3 | 自動換行 | 41 個全形字 |
| 名字 `tmp_name` | 316×100 | 自動縮字 35～50 | 自動換行 | 50 級 6 字、35 級 9 字 |
| 稱號 `tmp_nickName` | 274×63 | 自動縮字 20～35 | 自動換行 | 35 級 7 字、20 級 13 字 |
| 黑幕台詞 | 寬 3000 | 55，不縮字 | 自動換行 | 54 個全形字 |
| 劇情紀錄的內文 | 寬 1110 | 50，不縮字 | 自動換行 | 22 個全形字 |
| 劇情紀錄的名字 | 300×58 | 自動縮字 18～35 | 自動換行 | |

- 台詞的字級另外由程式依實際排出來的行數調整：3 行以內 45、4 行 40、5 行以上 38（[02-tags-ruby.md](02-tags-ruby.md)）。prefab 的 45 與「3 行以內」一致。文字超出框的部分會溢出，不會被截掉。
- 以每行 41 字估 `StoryData` 的 63,031 句台詞：

| 行數 | 譯文 | KR |
|---|---|---|
| 1 行 | 59,624 | 55,168 |
| 2 行 | 3,366 | 7,724 |
| 3 行 | 41 | 135 |
| 4 行以上 | 0 | 4 |

- 譯文普遍比韓文短，沒有需要四行以上的句子。
- 名字欄：譯文有 21 個不同的說話者名稱超過 9 個全形字（98 列，例如「被矇住雙眼的裴廓德鎮居民」），縮到 35 級仍會折成兩行。稱號沒有超過 13 個全形字的。
- 台詞的 ruby 版本（`tmp_serifRuby`）與一般版本的字級、字距相同。

### 從這一節得到的建議（2026-11-18 提出，決定先不採行）

下面兩項都只是記錄。指引（`guides/`）與譯文都沒有因此修改；估算沒有在遊戲裡驗證過，等實際看到問題再處理。

**建議一：`EGOVoiceDig` 的長句比照官方日文、英文加 `\n`。**

- 依據：演出字幕與技能演出的台詞框都不換行、不縮字，一行太長只會往兩邊溢出；官方 JP、EN 在同一批句子裡會自己加 `\n`，KR 不加。
- 可以寫成的規則：`EGOVoiceDig/*` 的 `dlg` 單行超過約 36 個全形字時，在語意停頓處加一個 `\n`；JP 或 EN 有加的句子，比照它們的位置。
- 沒有採行的原因：一行 36 字是用畫布設定推算的，沒有在遊戲裡量過；哪些句子真的顯示在不換行的字幕裡也讀不到。

**建議二：三處可能超出的譯文。**

| 位置 | `id`／`key` | 現行譯文 | 估算 | 可以怎麼改 |
|---|---|---|---|---|
| `EGOVoiceDig/Voice_EGO_Rodion_9.json` | `battle_erosion_20908_1` | 到底會是誰呢？是你嗎？……還是你？究竟是哪邊能燃得更華麗、更炙熱呢……挺讓人在意的對吧？ | 42 個全形字；KR 27、JP 33 | 譯文比原文多了不少字，可以精簡，或在「還是你？」之後加 `\n` |
| `EGOVoiceDig/Voice_EGO_Faust_2.json` | `battle_awaken_20209_1` | 漫長的旅途。在這浪潮不止的時代中，尋找希望與未來。去找尋屬於自己的答案吧。 | 37 個全形字；KR 36 | JP 在「時代の中で、」之後、EN 在 `restlessness,` 之後加了 `\n`，可以比照 |
| `RewardDungeonUI.json`（獎勵地牢的建議等級） | `equal_or_more` | 或以上 | 框寬 71、字級 35，只夠兩個全形字；估 1.48 倍 | KR「이상」、JP「以上」都是兩個字，可以改成「以上」 |

- 設定頁的 `connect_to_google_account`、`connect_to_apple_account`（「綁定Google帳號」「綁定Apple帳號」）估 1.26 倍與 1.08 倍，同樣沒有修改。

## 讀不到、沒有確認的部分

- 中央字幕（`BattleCenterSubtitleUI`）不在場景檔與本體的資源檔裡，推定在下載的 bundle 裡，它在 prefab 上的實際數值仍未知。
- 哪些句子顯示在不換行的演出字幕裡（演出資料決定）。
- 一行幾個字、估計字級都是用字寬的估算值算的，沒有量語言包字型，也沒有在遊戲裡比對。
- 語言包的字型怎麼接到文字元件上，以及 `<material>`（`dkr`／`den`／`djp`）在語言包字型上的實際效果。
- `TMP Settings` 的其餘欄位（缺字時顯示的字元、後備字型清單）只讀了能確定位置的部分。
- 啟動階段的錯誤訊息在語言包生效時實際顯示哪個語言（依載入規則推定是遊戲內選的語言）。
