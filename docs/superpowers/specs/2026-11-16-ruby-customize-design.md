# 模擬 ruby 的自訂標籤 `<rubycustomize>` 與照抄單位的翻譯例外

日期：2026-11-16　狀態：使用者核准 2026-11-16

## 1. 目的

遊戲只在劇情 `StoryData` 的 `content`（與劇情裡的歌詞）處理 `<ruby=注音>本文</ruby>`。其他位置會把標籤原樣顯示（`docs/research/2026-11-15-game-text-logic/02-tags-ruby.md`）。所以這些位置現在只能寫「本文(注音)」，或乾脆不寫注音。例如：
- KR 用本名唸法「요시히데」叫良秀，現行譯文只寫「良秀」，看不出這是本名的唸法；
- 官方三語都保留德文的說話者名「UNGEZIEFER KAISER」，玩家看不出意思。

遊戲自己的 ruby 也只是把 `<ruby>` 轉成一般 TextMeshPro 標籤再顯示（同上報告）：

```text
<nobr>本文<space=-(本文寬+注音寬)/2><voffset=1em><size=50%>注音</size></voffset><space=(本文寬-注音寬)/2></nobr>
```

這次要讓譯文在遊戲不支援 ruby 的位置也能用注音：譯文寫一個好讀的自訂標籤，產生語言包時換成上面這組標籤。

**遊戲內實測（2026-11-16，使用者）**：手寫上面這組標籤（寬度用字型資料算），在下列位置都顯示正常：注音置中在本文上方、沒有露出標籤原文、沒有被切掉。

| 位置 | 測試內容 | 結果 |
|---|---|---|
| 劇情對話框的說話者名牌（`ScenarioModelCodes` 的 `name`） | 害蟲皇帝／UNGEZIEFER KAISER | 正常。名牌傾斜約 6～7 度，注音看起來偏左約 0.25em，可以接受 |
| 劇情紀錄的說話者名稱 | 同上 | 正常 |
| 劇情 `teller` | 沒有另外測 | 視為已測：`teller` 和上兩列是同一塊名牌、同一個紀錄名稱。遊戲的規則是列上的 `teller` 有寫就用它，空的才查 `ScenarioModelCodes` 的 `name`（`07-field-display.md`） |
| 人格語音 `dlg`（大廳卡片／語音分頁） | 良秀／ヨシヒデ，多行台詞的第一行 | 正常，行距正常 |
| 播報員 `dlg`（試聽彈窗） | 同上 | 正常 |
| 播報員 `dlg`（戰鬥中的對話框，會捲動） | 第 4 行有注音、超出 3 行時捲動 | 正常，捲動時沒有露出標籤 |
| 戰鬥氣泡（角色頭上；`BattleSpeechBubbleDlg*` 與人格語音的戰鬥台詞共用） | 在台詞第 2 行尾端加「良秀／ヨシヒデ」 | 正常 |

## 2. 目標、非目標、反目標

**目標**
- 譯文可以寫 `<rubycustomize=注音>本文</rubycustomize>`，只限白名單上的位置。
- `lct build` 產生的語言包（`tw` 分支與 Release）把它換成遊戲能顯示的標籤；譯文紀錄保留原本的寫法。
- 人工調整和自動翻譯都能使用。自動翻譯只在白名單位置被告知可以用；什麼時候用，照劇情 ruby 的同一套規則。
- 欄位規則可以讓「KR、EN、JP 三語相同而照抄」的單位照樣送去翻譯。這份例外清單以後可以再加。
- 網站把轉換後的標籤顯示成 ruby。
- 修正 `guides/base.md` 與檢查裡和遊戲行為衝突的 ruby 規則：說話者名稱、稱號、地點、RPG 對話。
- 補上現有的 76 筆舊譯文（第 12 節）。

**非目標**
- 不改劇情 `content` 的原生 `<ruby>`。
- 不把稱號（`title`、`nickName`）、地點（`place`）、RPG 對話、技能名稱等放進白名單。這些位置等使用者在遊戲裡測過再說。
- 不做字距調整（kerning）、行高補償等細部排版。

**反目標（做了就算失敗）**
- 語言包裡留下沒轉換的 `<rubycustomize>`，遊戲會把它原樣顯示。
- 白名單以外的位置出現轉換後的標籤。
- 因為轉換，譯文紀錄和語言包字串對不上，網站、更新頁把它算成「沒有譯文、用了英文或 KR 代替」（stand-in）。

## 3. 做法（使用者已選 A）

| | 做法 | 結果 |
|---|---|---|
| **A** | 新的自訂標籤 `<rubycustomize>`，建置時轉換 | **採用**。和遊戲原生 `<ruby>` 分得清楚，檢查規則簡單：`<ruby>` 只在劇情 `content`，新標籤只在白名單。遊戲以後支援更多欄位也不會重複轉換 |
| B | 沿用 `<ruby>`，依位置決定要不要轉換 | 不採用。同一個標籤在不同位置意義不同，誤用時不容易察覺 |
| — | `lct publish` 打包 Release 時才轉換 | 不採用。`tw` 分支的語言包會留著遊戲看不懂的標籤，違反「`tw` 就是成品」 |

## 4. 標籤語法

```text
<rubycustomize=注音>本文</rubycustomize>
```

- 形式照 `<ruby>`：`=` 必須有；名稱只接受小寫 `rubycustomize`；注音可以不加引號，也可以用雙引號包起來（`"…"`）。
- 注音與本文都必須是**純文字**，不能是空的。不能有 `<`、`>`、換行，不能巢狀。
- 同一個欄位可以有多組。
- 不能和原生 `<ruby>` 寫在同一組裡。

## 5. 白名單：`rules/ruby-customize.yaml`

新的規則檔，列出可以使用的「檔案＋欄位」，以及轉換參數：

```yaml
# 可以寫 <rubycustomize=注音>本文</rubycustomize> 的位置。每個位置都在遊戲裡測過才加入
# （見 docs/superpowers/specs/2026-11-16-ruby-customize-design.md 第 1 節）。
size: 50%        # 注音的字級
voffset: 1em     # 注音上移的高度
fields:
  - files: ['ScenarioModelCodes*']      # 劇情名牌、劇情紀錄
    field: name
  - files: ['StoryData/**']             # 同上（列上的說話者名稱優先於 ScenarioModelCodes）
    field: teller
  - files: ['PersonalityVoiceDlg/**']   # 大廳卡片、語音分頁、戰鬥氣泡
    field: dlg
  - files: ['BattleAnnouncerDlg/**']    # 播報員試聽、戰鬥中的播報員對話框
    field: dlg
  - files: ['BattleSpeechBubbleDlg*']   # 戰鬥氣泡（也用於中央字幕、演出字幕）
    field: dlg
```

- `files` 的寫法和其他規則檔相同（`config.MatchAny`），比對的是語言包內的路徑，不含語言前綴。
  - `ScenarioModelCodes*` 包含所有 `ScenarioModelCodes` 開頭的檔，這是刻意的：它們都是說話者名稱表。
- `field` 必須和單位的欄位名（`units.Unit.Field`）完全相同。同一個檔裡的其他欄位不在白名單內，例如人格語音的 `desc`（語音分頁每一格的標題）。
- 每一項旁邊用註解記錄測試的畫面與日期。
- 戰鬥氣泡：
  - 角色頭上的氣泡已實測。
  - 中央字幕、演出字幕沒有另外測。研究報告（`05-battle-speech-bubble.md`）確認這三種元件都把 `dlg` 原樣交給顯示元件，不切割、不逐字顯示，所以一併列入。
- `lct rules lint` 要檢查：
  - 格式正確，沒有未知的鍵；
  - `size`、`voffset` 能解析；
  - 每一項至少對到現有 KR 語言包的一個單位，對不到就給警告。

## 6. 轉換（`lct build`）

在 `internal/build` 寫入每個單位字串之前（`buildFile` 裡 `out.SetString` 之前），把每一組 `<rubycustomize=R>B</rubycustomize>` 換成：

```text
<nobr>B<space=S1em><voffset=V><size=Z>R</size></voffset><space=S2em></nobr>
```

**寬度怎麼算**
- 本文寬 `wb`：B 每個字元的前進寬度（advance）加總，以 em 為單位。
- 注音寬 `wr`：R 的前進寬度加總，再乘上字級比例 Z（50% 時乘 0.5）。
- `S1 = -(wb + wr) / 2`，`S2 = (wb - wr) / 2`。
- 前進寬度用建置已經載入的字型（`lct.yaml` 的 `font`，`internal/build` 的 `in.face`，`sfnt`）：字形的 advance ÷ unitsPerEm。
  - Sarasa Gothic TC Bold 的漢字、假名、注音符號、全形標點都剛好是 1em；英文字母不等寬，例如 `a` 0.581em。
  - 不做 kerning。實測的誤差在 1～3px 以內。
  - 字型沒有某個字元時，那個字元算 1em。缺字本來就會列在建置報告的缺字清單，不另外讓建置失敗。（設計者決定）

**輸出格式**
- 數值四捨五入到小數三位，去掉尾端的 0（`-2.25em`、`0.5em`、`0em`）。
- 同樣的輸入每次輸出位元組都相同。
- 已經驗證過的實例（和實測用的字串逐字相同）：
  - `<rubycustomize=ヨシヒデ>良秀</rubycustomize>` → `<nobr>良秀<space=-2em><voffset=1em><size=50%>ヨシヒデ</size></voffset><space=0em></nobr>`
  - `<rubycustomize=UNGEZIEFER KAISER>害蟲皇帝</rubycustomize>` → `<nobr>害蟲皇帝<space=-4.542em><voffset=1em><size=50%>UNGEZIEFER KAISER</size></voffset><space=-0.542em></nobr>`

**保護措施**
- 只轉換白名單位置的單位。白名單以外的位置出現新標籤時，這筆紀錄無法通過檢查（第 7 節），建置照現有流程改用 EN 或 KR 代替並重新排入待譯，不會轉換。
- 寫完語言包後，任何字串裡還有 `<rubycustomize` 就讓建置失敗（系統錯誤）。
- 字形檢查（缺字）用轉換前的文字或轉換後的文字都可以，結果相同。
- 建置報告記錄轉換了幾組。

**轉換與還原的函式**
- 轉換寫成一個獨立的套件，例如 `internal/rubyx`。
- 這個套件提供兩個函式：
  - **轉換**：輸入是文字和字型的寬度資料，只有建置會用到字型。
  - **還原**：把第 6 節輸出的固定結構還原成 `<rubycustomize=R>B</rubycustomize>`，不需要字型。
- 第 8 節的比對用還原，所以網站、更新頁不需要載入字型。（設計者決定）

## 7. 檢查（`internal/validate`）

**白名單位置**
- 允許成對的 `<rubycustomize=…>…</rubycustomize>`，而且 KR 不用有。標籤比對（`tagDiff`）要把它排除，就像劇情 `content` 排除 `<ruby>` 一樣。
- 語法不符合第 4 節時，是硬性錯誤（structural）：缺 `=`、空的注音或本文、裡面有標籤或換行、巢狀、沒有閉合。

**白名單以外**
- 出現 `<rubycustomize` 就是硬性錯誤，訊息要寫明「這個位置不在 rules/ruby-customize.yaml」。

**呼叫端**
- 檢查需要知道這個單位在不在白名單。做法交給實作者：可以在 `validate.Context` 加一個由呼叫端填的欄位，或讓 Context 帶白名單。
- 所有呼叫 `validate.Check` 的地方都要正確填寫：建置、`batch submit`／判定、`batch preview`、`adjust add`、`lct validate`、網站審核資料。

**修正 RPG 對話**
- 現在 `isStory` 把 RPG 對話（`RPGSystem/*dialogue*`）當成劇情，接受 KR 沒有的 `<ruby>`。但遊戲在 RPG 不處理 `<ruby>`。
- 改成 RPG 對話不接受 KR 沒有的 `<ruby>`。實作者要先檢查 `isStory` 的其他用途，只改 ruby 這一部分。
- 現在 RPG 語言包裡沒有任何 `<ruby>`，這個修正不會讓現有譯文失敗。

**其他（設計者決定）**
- 譯文內容的檢查只看本文，不看注音。這包括詞彙表（含 `check: hard`、禁用譯法）、殘留韓文或外文、字數等。注音只檢查第 4 節的語法。
  - 原因：注音常常是假名或外文原文（`ヨシヒデ`、`UNGEZIEFER KAISER`），不是譯文本身。
- KR 本來就不會有 `<rubycustomize>`，不必為此另設規則。

## 8. 紀錄與語言包字串的比對

有些程式用「紀錄的 `zh` 等於語言包字串」來判斷這個單位顯示的是不是紀錄的譯文。轉換後兩者不同，所以這些地方要改成比較「**紀錄**」和「**語言包字串還原後**」（第 6 節的還原函式，不需要字型）：

| 位置 | 用途 |
|---|---|
| `internal/site/file.go` | 單位的 `o`（來源） |
| `internal/site/export.go` | 抽樣（gold）的 `Shipped` |
| `internal/history/compare.go` 的 `origin` | 更新頁的來源 |

實作者要用搜尋確認沒有其他這種比對，例如 `history/causes.go` 和調整的比對。注意：這類比對有些拿調整的 `zh` 和紀錄比，有些拿紀錄和語言包比，只有後者需要轉換。

## 9. 欄位規則例外：照抄單位照樣翻譯

在 `rules/fields.manual.yaml` 加一個鍵：

```yaml
translate_identical:            # 即使 KR、EN、JP 三語相同也照樣翻譯的欄位
  - file_glob: 'ScenarioModelCodes*'
    field: name                 # 說話者名稱：官方保留原文的名字也要能加注音，例如 UNGEZIEFER KAISER
    pattern: '\p{L}'            # 選填：KR 符合這個正規表示式才適用；沒有字母的名稱（???）維持照抄
```

- `fields.Classify` 的第 7 條（`identical-value`）不適用於清單上的欄位。其他條（空白、識別欄位等）照舊。
- `pattern` 是選填的 Go 正規表示式，比對 KR；有寫時，KR 符合才適用例外。寫錯是規則錯誤，`lct rules lint` 會報出。
- 清單上的照抄單位會變成「要翻譯」，影響如下：
  - 已有的照抄紀錄（`origin: copy`，KR 雜湊相同）在建置時會被當成可用的紀錄，文字等於 KR，不會自動重排。實作者要確認這個行為，並把它寫成測試。
  - 人工調整可以覆蓋它，不會再被第 7 條擋下。
  - 以後新出現的三語相同名稱會照一般流程送翻。
- 網站搜尋：這些單位不再是照抄，照一般單位收入。
- `lct rules lint` 要檢查新鍵的格式。

## 10. 自動翻譯與指引

**批次**
- 單位在白名單上時，批次裡的這個單位帶一個標記，例如 `BatchUnit` 加 `ruby_customize: true`（`omitempty`）。
- `schemas/batch.schema.json` 和 `prompts/translate.md` 一併更新：有標記的單位可以用 `<rubycustomize>`，沒有標記的不能用。

**`guides/base.md` 的 ruby 段落改寫**
- 新增「白名單位置的注音」：
  - 位置見 `rules/ruby-customize.yaml`；
  - 寫法是 `<rubycustomize=注音>本文</rubycustomize>`，不手寫 `nobr`、`space` 等標籤；
  - 什麼時候用照劇情 ruby 的規則：良秀的兩種讀音、外語原文、雙關、扭曲的台詞。良秀的例子：KR `요시히데` 寫 `<rubycustomize=ヨシヒデ>良秀</rubycustomize>`。
- 說話者名稱（`teller`、`ScenarioModelCodes` 的 `name`）：
  - 改用 `<rubycustomize>`；
  - 刪掉「KR 有 ruby 才加」。研究報告確認遊戲在這些位置不處理 `<ruby>`。
- 稱號（`title`、`nickName`）、地點（`place`）：不加注音，照詞彙表。
- RPG 對話：
  - 從「劇情台詞：ruby 一律保留為標籤」中移出，改比照「其他檔案」寫「本文(注音)」；
  - 一併註明遊戲在 RPG 不處理 `<ruby>`；
  - 等在遊戲裡測過，可以再加入白名單。
- 「其他檔案」的規則維持「本文(注音)」，另外註明白名單位置改用新標籤。

**其他文件**
- `guides/README.md`、`docs/research/2026-11-15-game-text-logic/02-tags-ruby.md` 補一句，指向新機制。

## 11. 網站

- 網站顯示的是語言包字串，也就是轉換後的標籤。
- `site/src/tmp` 的解析器要認得第 6 節輸出的這個固定結構：`<nobr>B<space=…><voffset=…><size=…>R</size></voffset><space=…></nobr>`，把它顯示成 `<ruby>B<rt>R</rt></ruby>`，和原生 `<ruby>` 的顯示方式相同。
- 不符合這個結構的 `nobr`、`space` 等照現狀處理。
- 檔案頁、搜尋結果、更新頁都會經過同一個解析器。
- 搜尋要能用本文或注音找到。實作者要確認搜尋語料去掉標籤之後兩段文字都還在，而且沒有黏在一起；必要時在兩段之間保留分隔。

## 12. 舊譯文補正（76 筆）

**時機與方式**
- 等機制合併進 `main`、CI 用新版建置之後，另開一次變更，用現有的規則檔補上，不手改狀態。
- 改寫檔由計畫（`lct plan`）套用到紀錄。之後建置在檢查紀錄時會帶白名單資訊，所以改寫後的文字照第 7 節檢查。實作者要確認套用改寫的路徑沒有不帶白名單的檢查；如果有，要一併帶上。
- 9 句台詞（4 句人格語音、2 句播報員、3 句戰鬥氣泡）的譯法由設計者先擬好，使用者確認後才寫成調整。這一步不在實作計畫裡自動完成。

| 內容 | 筆數 | 做法 |
|---|---|---|
| 劇情 `teller`：KR `요시히데`，現行「良秀」 | 62 | 改寫檔：`files: [StoryData/**]`，`pattern: '^良秀$'`，`kr_pattern: '^요시히데$'` → `<rubycustomize=ヨシヒデ>良秀</rubycustomize>` |
| `ScenarioModelCodes` 的 `name`：KR `요시히데`（過去的良秀 1～3） | 3 | 同一種改寫，`files: [ScenarioModelCodes*]` |
| 人格語音 `dlg`：KR 有 `요시히데` | 4 | 人工調整；每句的譯法先給使用者確認（句子裡可能同時有 `료슈`） |
| 播報員 `dlg`：KR 有 `요시히데` | 2 | 同上 |
| 戰鬥氣泡 `dlg`：KR 有 `요시히데` | 3 | 同上 |
| `UNGEZIEFER KAISER`（`ScenarioModelCodes` #628、#630） | 2 | 人工調整：`<rubycustomize=UNGEZIEFER KAISER>害蟲皇帝</rubycustomize>` |

## 13. 測試

**轉換**
- 第 6 節的兩個實例要輸出逐字相同的結果。
- 全形字與英文字母混合。
- 注音比本文寬（`S2` 是負數）。
- 一個欄位有多組。
- 小數格式（去掉尾端 0、`0em`）。

**檢查**
- 白名單位置接受正確的寫法；第 4 節的每一種錯誤寫法都報錯。
- 白名單以外出現新標籤會報錯。
- RPG 對話不再接受 KR 沒有的 `<ruby>`。

**建置**
- 白名單位置的紀錄被轉換。
- 語言包殘留 `<rubycustomize` 時建置失敗。
- 網站和更新頁的來源判斷：轉換後的字串仍是紀錄的來源，不是 stand-in。

**欄位規則**
- `translate_identical` 讓三語相同的 `name` 變成要翻譯，其他欄位不受影響。
- 已有的照抄紀錄在加了例外之後的行為。

**網站**
- 解析器把轉換後的結構顯示成 ruby；不完整的結構照原樣處理。

**驗收**
- 用實際資料建置。
- 對 `ScenarioModelCodes` #630 和希斯克利夫語音 `smalltalk_10716_3` 的輸出，要和實測字串逐字相同。

## 14. 留給實作者

- 轉換函式的套件名稱與介面。
- `validate.Context` 怎麼帶白名單資訊。
- 批次標記的名稱。
- 網站解析器怎麼辨識結構，以及顯示方式的細節。
- `rules/ruby-customize.yaml` 的載入位置：`rules` 套件或獨立載入。

## 15. 風險

- **其他字型：** 遊戲某些介面若用別的字型或另外設定字距，注音會略偏；名牌實測偏差約 0.25em，主要來自名牌傾斜。
- **行高：** 有注音的那一行會變高，接近版面上限的文字可能提早換頁或捲動。
- **太長的注音：** `<nobr>` 讓本文和注音不能斷行。注音很長或落在行尾時，可能把那一行撐寬。
  - 不設長度上限，也不加檢查（設計者決定）。
  - 指引會要求注音簡短，只寫讀音、原文或原意。
- **以後新增的白名單位置：** 每一項都要先實測。研究報告指出技能名稱會被程式截斷，不能加入。
