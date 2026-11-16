# `<ruby>` 與其他標籤

依據：`TMPro.CustomRubyTextMeshProUGUI`、`TMPro.CustomRubyTextMeshProDefinitions`（遊戲自己的版本）、第三方的 `RubyTextMeshPro.dll`、`StorySystem.DialogController`、`StoryMarkerSubtitlePlayer`、`LogContentPrefab`、`TextCenterPlayHandler`、`SkillTextDataMediator`（2026-11-12 版）。「確認」＝對照反組譯讀過；「推定」＝只看反編譯結果或由一般知識推論；位址是 RVA。

## 先講結論

| 位置 | `<ruby>` | 一般 TextMeshPro 標籤 |
|---|:-:|:-:|
| `StoryData/*.json` 的 `content`（一般對話框、黑畫面對話、演出字幕） | ✔ 只能寫 `<ruby=注音>本文</ruby>` | ✔ |
| 劇情紀錄（同一個 `content`） | ✔ | ✔（第一個 `<color=#…>` 起始標籤會被刪掉） |
| 戰鬥中或演出中的劇情字幕（同樣讀 `StoryData` 的 `content`） | ✔ | ✔（文字裡只要有 `<`，就不會自動拆成兩段顯示） |
| `BgmLyrics/*.json`，在劇情裡顯示時 | ✔ | ✔ |
| `BgmLyrics/*.json`，在戰鬥裡顯示時 | ✘ | ✔ |
| `StoryData` 的 `teller`、`title`、`place` | ✘ | ✔ |
| 其他全部（戰鬥台詞、人格語音、播報員、大廳、技能、被動、狀態、飾品、介面、但丁筆記、RPG） | ✘ | ✔ |

- ✘ 的地方沒有任何程式轉換或移除 `<ruby>`，文字會原樣交給 TextMeshPro。TextMeshPro 遇到不認識的標籤會把它當一般文字畫出來（這一點是 TextMeshPro 的一般行為，不是從這一版程式讀到的），所以玩家會看到 `<ruby=…>本文</ruby>`。
- 現行譯文的 ruby 共 155 處，全部在 `StoryData` 的 `content`，格式都正確。
- 後續（2026-11-16）：✘ 的位置裡，在遊戲裡測過能顯示注音的（說話者名稱、人格語音、播報員、戰鬥氣泡），譯文改寫 `<rubycustomize=注音>本文</rubycustomize>`，`lct build` 換成下面「ruby 怎麼變成畫面」那組 TextMeshPro 標籤。位置見 `rules/ruby-customize.yaml`，設計見 `docs/superpowers/specs/2026-11-16-ruby-customize-design.md`。

## ruby 的語法

正規表示式（`RubyTextMeshProDefinitions` 靜態建構子 0x5EE3CA0，沒有任何選項）：

```text
<r(uby)?="?(?<ruby>[\s\S]*?)"?>(?<val>[\s\S]*?)<\/r(uby)?>
```

- 起始 `<ruby=` 或 `<r=`，結束 `</ruby>` 或 `</r>`。**但對話框與字幕只在文字含有 `<ruby` 這五個字元時才轉換**（`DialogController.ShowDialog` 0x20C4B30 用 `Contains("<ruby")` 判斷，`StoryMarkerSubtitlePlayer.Show` 0x20CDEC0 相同；確認）。所以只寫短寫法 `<r=…>…</r>` 的句子在對話框不會轉換，在劇情紀錄卻會。**一律寫 `<ruby=…>…</ruby>`**。
- `=` 必須有。`<ruby>本文</ruby>` 不會被比對到。
- 區分大小寫，`<Ruby=…>` 不算。
- 注音可以加雙引號也可以不加；官方 1,026 處都沒加。
- **注音裡不能有 `>`**，所以注音裡不能放標籤。本文裡可以有標籤與換行（JP 官方有 `<ruby=肉>サ<size=50%>ル</size></ruby>`）。
- 不能巢狀：裡面的 `<ruby=…>` 會被當成外層本文的一部分，第一個 `</ruby>` 就結束外層。

## ruby 怎麼變成畫面

`CustomRubyTextMeshProUGUI.set_uneditedText`（0x1ACFD70）→ `ReplaceRubyTags`（0x1ACFF30）→ `CustomReplaceRubyTags`（0x1AD02A0）。每一組 ruby 換成：

```text
<nobr>本文<space=-(本文寬+注音寬)/2><voffset=高度><size=比例%>注音</size></voffset><space=(本文寬-注音寬)/2></nobr>
```

- 本文寬、注音寬用 TextMeshPro 的 `GetPreferredValues` 量（以那個元件的字型與字級）。
- `<nobr>` 讓本文與注音不會被斷行拆開；本文很長時整段不能斷行。
- 程式裡的預設值：注音大小 50%、上移 `1em`、不固定行高、對齊方式為「注音置中於本文」（建構子 0x1AD00E0）。字幕用的元件是執行時建立的，用的就是這組預設值；對話框與劇情紀錄的元件在 prefab 裡，實際設定值在資源檔中，**讀不到**。
- 對話框裡，ruby 元件只用來把字串轉成上面的標籤，真正顯示的是一般的文字元件。打字效果是先把整段文字交給 TextMeshPro，再逐字改變頂點顏色（`TextCenterPlayHandler.PlayTextReveal` 0x187DED0），不是用字串切割，所以**標籤不會被打字效果切壞**；有 ruby 的句子裡，字級與本文不同的字（注音，以及同一句裡其他 `<size>` 的字）會跟著相鄰的字一起出現。

## 劇情文字的其他處理

- **字級依行數改變**（非黑畫面）：3 行以內 45、4 行 40、5 行以上 38，5 行以上對話框位置也會改。行數是 TextMeshPro 實際排出來的行數。沒有截斷。
- 版面沒有依 KR／EN／JP 分支。自動斷行是 TextMeshPro 的。
- `content` 送到對話框前只做一件事：`GetColorMappedLinkedString`（0x2865B30）把每個 `<link…</link>` 包上 `<color=#d0855d>`。官方劇情檔沒有 `<link`。沒有玩家名稱之類的佔位符。
- 接續上一句的台詞（`isContinue`）以「上一句＋一個半形空格＋這一句」組成。
- 劇情紀錄：`LogContentPrefab.GetRemoveRichText`（0x20EB230）只刪掉**第一個** `<color=#…>` 起始標籤，`</color>` 與之後的顏色標籤都留著。
- 劇情字幕的自動拆段（`StoryManager.TrySplitMarkerContent` 0x2121370，確認）：文字裡有任何 `<` 就不拆；否則在第一個 `!`、`,`、`?`、`！`、`，`、`？`、`、` 之後拆成兩段（後半要有字母或數字）。「。」與「…」不是拆點。
- 唯一依語言分支的劇情文字處理：劇情 `P10704` 第 3 句在特定條件下把「凱西／凱瑟琳」的原文字串（KR `캐시`／`캐서린`、EN `Cathy`／`Catherine`、JP `キャサリン`／`キャシー`）換成 `■■■■■`；語言不是這三種時記一筆錯誤、文字不變。譯文在這一句不會被遮蔽。

## 其他標籤

- **沒有任何遊戲程式把 rich text 關掉**（掃過所有寫入 `TMP_Text.m_isRichText` 的指令）。個別 prefab 上的設定在資源檔裡，讀不到。
- **`<noparse>`**：遊戲程式沒有賦予它任何意義，只有 TextMeshPro 自己處理。它能擋關鍵字的原因見 [03-keywords.md](03-keywords.md)。
- **`<style="…">`**：樣式定義在 TextMeshPro 的樣式表資源裡，讀不到；名稱必須是官方用過的。官方用過：`highlight`、`upgradeHighlight`、`highlightConverted`、`yellow`、`number_highlight`、`railwayDungeonEnemy`、`railwayDungeonAlly`，以及歌詞用的 `dkr…`（KR）、`djp…`（JP）、`den…`（EN）。歌詞樣式名稱依語言不同，譯文沿用 KR 的名稱；哪一組搭配譯文字型最合適沒有辦法從程式判斷。
- **`<style="highlight">` 與 `<style="upgradeHighlight">` 是用完全相同的字串比對的**（雙引號、沒有空格、小寫）。同步等級比較畫面（`SkillTextDataMediator`）在新等級文字含有 `<style="highlight">` 且與舊等級不同時，把它換成 `<style="highlightConverted">`；飾品強化預覽把 `<style="upgradeHighlight">` 換成同一個。寫法不同（單引號、多空格）就不會被換。少了 `</style>` 不會當機，但後面的標籤顏色會全部變黃（推定：只看了反編譯結果）。
- **同步等級比較的另一條路**（文字沒有 `<style="highlight">` 時；`SkillTextDataMediator.ColorChangedText` 0x1830BC0，推定）：把新舊兩級的文字各自以換行切開，逐行比較；整行相同的不標；只有數字不同的，只把變了的數字上色；其他的整行上色。所以：
  - 原文相鄰兩級完全相同的那一行，譯文也要完全相同（差一個字就整行變色）。
  - 原文只有數字不同的那一行，譯文也只能數字不同。
  - 行數（換行）要與原文一致，數字用半形。
  - 違反不會當機，只是強調顯示不對。
- **`<link>`**：技能、被動、飾品說明裡的 `<link="ID">` 只有在 ID 是狀態 id 時點了才有反應。官方資料裡有手寫的 `<sprite name="Burst"><color=#e30000><u><link="Burst">파열</color></link></u>`（約 250 處），除了顯示的名稱以外要原樣保留。
- **但丁筆記的 `<id>數字</id>`**：會變成黃色底線連結；內容以 `int.Parse` 轉成數字，不是數字會出錯。官方三種語言目前都沒有用到。
- **文字動畫外掛**（Febucci Text Animator，`<shake>`、`<wave>`、`<bounce>` 等）：遊戲裡有這個外掛，但只用在寫死的「NOW LOADING...」，沒有任何譯文會經過它。不要使用這些標籤。
- **會刪掉標籤的地方**：RPG 最後一段過場的字幕會刪掉全部 `<…>`；戰鬥歌詞只在計算長度時去掉標籤（8 字以內、15 字以內各用不同版面），顯示時保留。

## 讀不到、沒有確認的部分

- 對話框與劇情紀錄的 ruby 設定值（大小、高度、行高、對齊方式）在 prefab 裡。
- 對話框依行數把字級改成 40 或 38 時，沒有找到同步 ruby 轉換元件字級的程式；4 行以上的句子裡，注音的位置可能略偏（推定，沒有在遊戲中確認）。
- JP 官方 `ActionEvents_exme.json` 有一個 `name` 帶 `<ruby>`：`ActionEvents*` 的 `name` 沒有任何讀取者，根本不顯示（2026-11-16 補充，見 [07-field-display.md](07-field-display.md)）。
- 劇情對話裡的 `<link>` 點了有沒有反應，沒有追。

## 建議的檢查

1. `<ruby`、`<r=`、`</ruby>`、`</r>` 只能出現在 `StoryData/*.json` 的 `content` 與 `BgmLyrics/*.json`；其他位置（包含劇情的 `teller`、`title`、`place`）一律報錯。`BgmLyrics` 裡有 ruby 時提醒：戰鬥中會原樣顯示。
2. 每一組 ruby 都要符合 `<ruby=[^>"]+>[\s\S]*?</ruby>`：不用 `<r=`、不省略 `=`、不用大寫、注音裡沒有 `>`、沒有巢狀、本文裡的標籤成對。含有 `<ruby` 卻沒有完整比對到的字串會原樣顯示。
3. 標籤種類：譯文用到的標籤名稱必須在 KR 同一欄位出現過；`<style>` 的名稱必須與 KR 相同，且 `<style="highlight">`、`<style="upgradeHighlight">` 的寫法要逐字相同、每個都有 `</style>`。
4. 有多個同步等級的技能與被動：各等級的行數與 KR 相同；KR 相鄰兩級相同的行，譯文也相同；KR 只差數字的行，譯文也只差數字。
5. 劇情字幕：提醒有 `<` 的句子不會自動拆段。
6. 劇情 `content` 的長度：超過 3 行字級會縮小，5 行以上版面會變。行數要用字型實際排版才知道，只能用字數粗估。
