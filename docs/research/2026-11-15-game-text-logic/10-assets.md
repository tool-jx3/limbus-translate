# 資源檔裡的設定：下載網址、檔案清單、樣式表、換行規則、靜態標籤

2026-11-18。前面九章是從遊戲程式讀的；這一章讀的是**遊戲的資料檔**（2026-11-12 版）：

- 遊戲下載的資料：`RemoteLocalizeFileList.json`、`etc/VoiceTable.json`、`LocalizePatchInfo.json`、RPG 的檔案清單、Addressables 目錄 `catalog_S1.json`。
- 遊戲安裝資料夾 `LimbusCompany_Data` 裡的 `resources.assets`、`globalgamemanagers.assets`、`globalgamemanagers`。

「確認」＝直接從檔案讀到；「推定」＝有依據但沒有直接讀到。這些檔案都不放進版控，讀法見 [method/](method/README.md) 的 `filelist.py`、`catalog.py`、`unityres.py`。場景檔（`level*`、`sharedassets*`）還沒有讀，戰鬥氣泡的寬度與字級仍然未知。

## 先講結論

1. **官方文字是下載後以檔案讀取的**（設定值確認），每個檔都有固定的網址；`RemoteLocalizeFileList.json`、語音表、RPG 檔案清單都能直接下載。
2. **官方檔案清單**：77 張表、1,259 個檔名；KR 有 17 個檔不在清單裡，遊戲不會載入（細節在 [01](01-custom-language.md)、[07](07-field-display.md) 第 6 節）。RPG 的清單列了 189 個檔，和 KR 的 `RPGSystem/` 完全一致。
3. **啟動用清單**有四張表、五個檔；其中 `ErrorCodeMsgLocal` 不在 `raw` 裡，語言包沒有它，啟動階段的 16 句錯誤訊息推定不會是中文。這 16 句的 id 與文字和 `ErrorCodeMsg` 完全相同，語言包可以用既有的譯文補一個 `ErrorCodeMsgLocal.json`。
4. **樣式表**有 31 個樣式。`highlight` 與 `upgradeHighlight` 是空的，顏色是程式另外加的；`dkr`／`den`／`djp` 系列是**切換到該語言字型專用的材質**，不是單純的顏色。
5. **換行規則**是 TextMeshPro 的避頭尾字元表：開頭括號、`—`、`…` 後面不斷行；結尾括號與 `、。，！？：；` 前面不斷行。
6. **靜態標籤**：`resources.assets` 裡有 1,862 個標籤直接綁定介面文字的 id（952 個不同的 id）。估算譯文寬度後，只有 2 個可能超出框。

## 1. 下載設定與網址

`resources.assets` 裡的 `TextAssetPatchConfig`（確認）：

| 欄位 | 值 |
|---|---|
| `_patchMode` | 1 |
| `_localizeLoadMode` | 1。[01](01-custom-language.md) 讀到的程式是「0 走 Addressables，其他走檔案」，所以是**檔案模式** |
| `_localizeDataPath` | `Assets/Resources_moved/Localize` |
| `_remoteUrlLocalize` | `https://downloadcommon.limbuscompanycdn.org/<版本代碼>`，版本代碼就是 `raw` 的 `meta/source.json` 記的 `token`（這一版是 `l20261112_…`） |
| `_localizePatchInfoFilePath` | `Assets/LocalizePatch/LocalizePatchInfo.json` |

Addressables 目錄也印證這一點：21,880 個資源裡沒有官方文字，只有啟動用的 16 個項目。

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
- 影響（推定）：遊戲啟動、還沒下載完資料之前跳出的錯誤訊息，會是遊戲內選的語言。語言包只要多一個 `ErrorCodeMsgLocal.json`，內容取自已經翻好的 `ErrorCodeMsg.json` 那 16 筆，就能補上。

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

介面上固定不變的字（按鈕、標題、欄位名稱）是由 `UITextDataLoader` 元件依 `key` 從介面文字表取字（確認元件的欄位）。`resources.assets` 裡：

- 8,145 個文字元件（`TextMeshProUGUI`），其中 1,862 個掛了 `UITextDataLoader`，共 952 個不同的 `key`，分布在 116 個介面的 prefab。5 個標籤讀的是啟動用的表。
- 這 1,862 個標籤裡有 1,851 個的 `key` 在介面文字表（清單 `ui` 的檔案）找得到。
- 文字元件的設定（1,862 個標籤）：自動縮字 1,295 個；會自動換行 1,372 個、不換行 489 個；超出時直接溢出 1,754 個、截斷 106 個、加刪節號 1 個。
- 完整的對照表用 `python3 -I method/unityres.py <LimbusCompany_Data> labels` 產生：`key`、物件路徑、字級、縮字範圍、換行、溢出方式、固定寬度。

**譯文會不會超出框**：取「不換行、寬度固定」的 237 個標籤，以「全形字 1 個字寬、半形字 0.55 個字寬」乘上字級（有自動縮字的用最小字級）估算。譯文估計超出框、而且比韓文寬的只有 2 個：

| `key` | 位置 | 框寬／字級 | KR | 譯文 |
|---|---|---|---|---|
| `connect_to_google_account` | 設定的帳號連結按鈕 | 289／50 | `Google 연동` | `綁定Google帳號`（估 1.26 倍） |
| `connect_to_apple_account` | 同上 | 311／50 | `Apple 연동` | `綁定Apple帳號`（估 1.08 倍） |

這個估算的限制：

- 字寬是估的，語言包字型的實際寬度沒有量。
- 699 個標籤的寬度跟著上層物件伸縮，914 個會自動換行，這兩類沒有估（換行的要看高度）。
- 只涵蓋 `resources.assets` 裡的靜態標籤。程式動態設定的文字（技能說明、對話、氣泡）與場景檔裡的介面都不在內。

## 讀不到、沒有確認的部分

- 場景檔（`level0`～`level8`、`sharedassets*.assets`）：戰鬥氣泡、演出字幕、劇情對話框的寬度、字級、換行與溢出設定。
- 語言包的字型怎麼接到文字元件上，以及 `<material>`（`dkr`／`den`／`djp`）在語言包字型上的實際效果。
- `TMP Settings` 的其餘欄位（缺字時顯示的字元、後備字型清單）只讀了能確定位置的部分。
- 啟動階段的錯誤訊息在語言包生效時實際顯示哪個語言（依載入規則推定是遊戲內選的語言）。
