# 關鍵字、`[標籤]`、`<noparse>`、`[TabExplain]`

依據：`UI.Utility.TMProStringMatcher`、`StringUtility`、`SkillTextDataMediator` 與各 UI 類別（2026-11-12 版）。標「確認」的是對照反組譯讀過的；標「推定」的只看了反編譯結果或由資料推論；位址是 RVA。

## 先講結論

1. 遊戲對文字做兩道處理，**只有少數欄位會經過**：
   - **標籤處理**（`GetSkillTagMatchedString`）：把不是狀態 id 的 `[xxx]` 換成 `SkillTag.json` 的文字。
   - **連結處理**（`GetApplySpriteAndLinkAndColorString`）：把「純文字的狀態名稱」與 `[狀態id]` 換成「圖示＋顏色＋底線連結」。
2. 純文字的狀態名稱要變成連結，條件只有一個：**名稱後面緊接的字元**是「文字結尾、空白、數字、或 `・「」『』.,/&;` 其中之一」。名稱前面是什麼都不影響。全形的「，。、；：（）」都不會觸發。
3. `<noparse>名稱</noparse>` 與 `名稱[TabExplain]` 沒有特別的程式在處理「擋關鍵字」這件事，它們有效只是因為名稱後面變成了 `<` 或 `[`。兩者的差別在**之後怎麼消失**：`<noparse>` 由 TextMeshPro 吃掉，到處都能用；`[TabExplain]` 要靠標籤處理換成空字串，**沒有標籤處理的欄位會把 `[TabExplain]` 原樣顯示出來**。
4. 還有兩個官方機制可以擋：`Bufs*.json` 條目上的 `"excludeAutoLink": true`（這個名稱完全不參加比對），以及名稱前後緊鄰 U+200B（零寬空格）。前者日文官方有在用，後者官方資料沒用過。
5. 會經過處理的欄位很少（下表）。其他地方寫 `[Laceration]` 會原樣顯示成 `[Laceration]`。

## 哪些欄位會經過哪一道處理

「標籤」＝`[WhenUse]` 這類技能標籤會被換掉；「`[ID]`」＝`[Laceration]` 會變成圖示＋名稱連結；「名稱」＝純文字名稱會自動連結。

| 檔案 | 欄位 | 標籤 | `[ID]` | 名稱 | 顯示在 |
|---|---|:-:|:-:|:-:|---|
| `Skills*.json` | `levelList[].desc` | ✔ | ✔ | ✔ | 技能說明（人格資訊、戰鬥選技、E.G.O 選擇、同步比較） |
| `Skills*.json` | `levelList[].coinlist[].coindescs[].desc`、`.summary` | ✔ | ✔ | ✔ | 硬幣效果 |
| `Passives*.json`、`Passive*.json` | `desc`、`summary` | ✔ | ✔ | ✔ | 被動說明、被動摘要、共鳴階段說明 |
| `EGOgift*.json` | `desc` | ✘ | ✔ | ✔ | 所有 E.G.O 飾品說明介面 |
| `EGOgift*.json` | `simpleDesc[].simpleDesc` | ✘ | ✔ | ✔ | 戰鬥中飾品發動的文字 |
| `PanicInfo*.json` | `lowMoraleDescription`、`panicDescription` | ✘ | ✔ | ✔ | 人格資訊的精神分頁 |
| 其他全部 | — | ✘ | ✘ | ✘ | 原樣顯示 |

「其他全部」包括：所有 `name` 欄位（技能、被動、飾品、狀態）、`flavor`、`Bufs*.desc／summary`、`BattleKeywords*` 的全部欄位、`BuffAbilities`、`DungeonStartBuffs`、`MentalCondition`、`AbEvents*`、`ActionEvents*`、`MirrorDungeonUI*`、`BattleHint*`、`BattleResultHint*`、教學、關卡節點、劇情、語音。這些地方的 TextMeshPro 標籤（`<color>`、`<noparse>` 等）仍然有效，但 `[xxx]` 只是普通文字。

依據（確認）：三個入口函式的呼叫者是用反組譯的 `call` 指令全部列出來的——`GetAppliedAllRichText`（0x2861CA0，兩道都做）15 個、`GetSkillTagMatchedString`（0x2861D10）16 個、`GetApplySpriteAndLinkAndColorString`（0x2863EC0）31 個；E.G.O 飾品的 12 個介面與 `EGOGiftAbility..ctor`（0x2D9AC10）、`UnitInfoMentalTabContent.SetData`（0xFCDC60）只呼叫後者。例如 `EGOGift.EgoGiftInfomationUI.UpdateDesc`（0x1AFC320）的呼叫順序是 `TextData_EgoGift.GetDesc` → `GetApplySpriteAndLinkAndColorString` → `TMP_Text.SetText`。透過委派間接呼叫的情形沒有找到，但無法完全排除。

官方資料的寫法與這張表一致：KR 的 `[TabExplain]` 出現在 `Passives.desc`（93）、`Skills…desc`（44）、`EGOgift.simpleDesc`（28）、`Passives.summary`（13）、硬幣說明（12），`EGOgift.desc` 一次都沒有（那裡官方只用 `<noparse>`，42 處）。

> `EGOgift.simpleDesc` 是例外：官方在那裡寫了 28 處 `[TabExplain]`，但程式上那裡沒有標籤處理。照程式讀，它會原樣顯示；有沒有別的地方把它去掉，沒有在遊戲裡確認。譯文照 KR 保留即可，不要自己新增。

## 第一道：標籤處理

`GetSkillTagMatchedString(text, isGrayColor, isHighlight)`（0x2861D10）。用正規表示式 `\[.*?\]` 找出所有方括號（不跨行，最短比對），由後往前處理每一個：

1. 去掉方括號得到 id。
2. id 是狀態（在遊戲靜態資料的狀態表 `_buffList`／`_doubleBuffList` 找得到）→ **不動**，留給第二道。
3. 否則當成技能標籤：
   - 文字 ＝ `SkillTag.json` 裡該 id 的 `name`；`name` 裡的 `{n}` 換成標籤 id 裡的第一個數字。
   - 顏色 ＝ 靜態資料的介面顏色表裡該 id 的顏色；**找不到就用表裡的第一個顏色**（`UIColorStaticDataList.GetColorData` 0x1853C10：`Find` 失敗後回傳 `list[0]`，確認）。
   - 結果：`<color=#顏色>標籤文字</color>`。

要注意的行為：

- **`SkillTag.json` 找不到這個 id 時不會當機，會顯示 `Unknown`**。`JsonDataList<T>.GetData`（0x43090F0）找不到 id 時會新建一個空條目放進表裡再回傳，而 `TextData_SkillTag` 的建構子把 `name` 設成 `"Unknown"`（確認）。所以在技能、被動文字裡寫 `[注]`、`[1]`、`[肉]` 這種方括號，畫面上會變成有顏色的「Unknown」。
- `[TabExplain]` 是 `SkillTag.json` 裡 `name` 為空字串的條目，所以處理後是 `<color=#…></color>`，畫面上什麼都沒有。它不是特殊語法，只是一個「內容是空的標籤」。
- id 要完全相同（區分大小寫）。`[whenuse]`、`[When Use]` 都會變成 `Unknown`。
- 灰色模式（`isGrayColor`，技能無法使用時）不加個別的顏色標籤，`[TabExplain]` 直接變成空字串；這時它前面的名稱後接字元就是下一個字，擋不住關鍵字（推定：只看了反編譯結果）。

## 第二道：連結處理

`GetApplySpriteAndLinkAndColorString(text)`（0x2863EC0）。同一段文字的結果會快取。分兩步：

### 步驟一：純文字名稱

**名稱表**（`SetSkillKeywords` 0x2865E60，第一次用到時建立）：對 enum `BUFF_UNIQUE_KEYWORD` 的每個成員（這一版 2,178 個），到 `Bufs*.json` 找同 id 的條目：

- `name` 是空的或等於 `Unknown`（不分大小寫）→ 不收。
- 收進「名稱 → id」的表；同名的多個 id，先到的優先。
- 條目有 `"excludeAutoLink": true` → 名稱不放進比對用的 Trie（`StringUtility.BuildBuffTrie` 0x19F7440，確認）。

所以名稱表**來自語言包自己的 `Bufs*.json`**：譯文把狀態名稱譯成什麼，什麼就是關鍵字；`excludeAutoLink` 也是讀語言包的（`TextData_Buf` 位移 0x38，全程式只有這裡讀它）。`BattleKeywords*.json` 不參與比對。這一版 `Bufs*.json` 有 54 個 id 不是 enum 成員（例如 `RailBuffForLine2_*`、`Hardening`、`Taunt`），它們的名稱不會自動連結。

**比對**（`StringUtility.ParseBuffKeywords` 0x19F77F0，確認）：

1. 用 Aho-Corasick 找出文字中所有名稱的所有出現位置（完全相同才算，區分大小寫，可以重疊）。
2. 依名稱長度由長到短排序，逐一採用；與已採用的範圍重疊的丟掉。**長的名稱先佔位**：「震顫引爆」佔住之後，裡面的「震顫」不會再算。
3. 名稱的前一個字元或後一個字元是 U+200B → 丟掉。

**決定要不要連結**（回到 `GetApplySpriteAndLinkAndColorString`，確認）。對每個留下來的名稱，看它後面的文字：

- 排除：後面（相隔至多 1 個字元）是介面文字 `resonance`（「共鳴」）、或去掉空格後是 `perfect_resonance`（「完全共鳴」）、或是 `속성`、`Abs.` → 不連結。這是為了「憤怒 共鳴」這類寫法。
- 名稱在文字結尾 → 連結。
- 下一個字元是空白（`char.IsWhiteSpace`：半形空格、全形空格 U+3000、換行、Tab、不換行空格 U+00A0 都算）→ 連結。
- 下一個字元是數字（`char.IsDigit`：全形數字也算）→ 連結。
- 下一個字元是 `・「」『』.,/&;` 這 10 個字元之一 → 連結。
- 其他 → 不連結。

連結後的樣子：`<sprite name="ID"><color=#顏色><u><link="ID">名稱</color></link></u>`，顏色依狀態是正面、負面或中性。

注意「長的先佔位」發生在「看後接字元」之前：「流血強度 」如果有個狀態叫「流血強度」就由它決定；長名稱佔了位卻因後接字元不合而沒連結時，裡面的短名稱也不會連結。

### 步驟二：`[ID]`

`GetBuffMatchedString`（0x2862650）。再用 `\[.*?\]` 找一次方括號，id 在狀態表找得到的，換成：

`<sprite name="圖示"><color=#顏色><u><link="ID">` ＋ `Bufs*.json` 該 id 的 `name` ＋ `</color></link></u>`

- 語言包的 `Bufs*.json` 沒有這個 id → 名稱顯示成 `Unknown`（同樣是 `GetData` 新建空條目的結果）。
- id 不是狀態 → 不動，原樣顯示。所以在沒有第一道處理的欄位（飾品 `desc` 等），`[WhenUse]`、`[TabExplain]` 會原樣出現在畫面上。
- `[ID]` 是不是狀態，看的是遊戲靜態資料，不是語言包；譯文只能照 KR 用同一個 id。

### 點連結之後

`SkillDescriptionInteraction.TooltipInterectionFunction`（0xFAFBD0）把連結 id 轉成 `BUFF_UNIQUE_KEYWORD`，交給 `BuffKeywordToolTipUI.SetData`（0xFAE130），顯示 **`BattleKeywords*.json`** 的 `name`、`desc`、`flavor`，**原樣顯示、不再做任何關鍵字處理**。

- 句子裡看到的名稱來自 `Bufs*.json` 的 `name`，點開後的標題來自 `BattleKeywords*.json` 的 `name`。兩個檔的同一個 id 名稱應該一致（KR 有 1,924 個共同 id，其中 14 個名稱不同）。
- `BattleKeywords*.json` 沒有這個 id → 標題與說明都顯示 `Unknown`。
- 戰鬥中滑過狀態圖示看到的是 `Bufs*.json` 的 `desc`／`summary`，會以層數等數值做 `string.Format`（見 [04-other.md](04-other.md)）。

## 擋住自動連結的方法比較

| 方法 | 原理 | 哪裡有效 | 注意 |
|---|---|---|---|
| `<noparse>名稱</noparse>` | 名稱後面變成 `<`；標籤本身由 TextMeshPro 吃掉 | 有連結處理的所有欄位 | 裡面**不可以放 `[ID]`**：遊戲會先把 `[ID]` 換成一串標籤，TextMeshPro 在 `<noparse>` 裡不解析標籤，那串標籤會整段顯示出來（依 TextMeshPro 的一般行為推定；官方資料與現行譯文都沒有這種寫法）。裡面放 `<color>` 等標籤也會被原樣顯示 |
| `名稱[TabExplain]` | 名稱後面變成 `[`；標籤處理把它換成空的顏色標籤 | **只有**技能、硬幣、被動的 `desc`／`summary` | 在飾品 `desc`、`PanicInfo` 會原樣顯示 `[TabExplain]`；`[` 前不能有空格（有空格就等於名稱後面接空白，會連結）；可以插在名稱中間把名稱切斷（JP 官方的寫法：`夜明け事務[TabExplain]所`） |
| `"excludeAutoLink": true` | 這個名稱完全不參加比對 | 整個語言包 | 寫在 `Bufs*.json` 的條目上。這個名稱從此只能用 `[ID]` 連結。KR／EN 有 14 個條目帶這個欄位，JP 多了 3 個（`～`、`種`、`始`） |
| 名稱前或後放 U+200B | 比對時直接跳過 | 有連結處理的所有欄位 | 官方資料沒用過；字型若沒有這個字元，是否顯示成缺字方塊沒有確認 |
| 名稱後面接其他字元 | 不在允許清單就不連結 | 同上 | 例如全形標點、中文字、`)`、`<`。「流血。」「流血，」「流血)」「流血強度」都不會連結 |

反過來，**要讓純文字名稱連結**：名稱後面留一個半形空格（或讓它在句尾、後面接數字）。這就是原指引「中文字＋空格」的由來；「2層強壯 ，」的空格是必要的，因為「，」不在清單裡。

## 用官方資料與遊戲內結果驗證

- 專案裡兩個在遊戲中確認過的案例都符合：「攻擊等級+1」（名稱「攻擊等級+」後面接數字）、「加斥我援」對…（名稱「援」後面接「」」）。
- KR 官方用 `<noparse>` 或 `[TabExplain]` 擋掉的名稱共 294 處，名稱後一個字元是空格 206、文字結尾 35、`,` 28、換行 6——都是會觸發的情形。另有 9 處是 `)`（例如「소수점 <noparse>버림</noparse>)」），依規則本來就不會連結；官方有多包的情形，**不能反過來從「官方有包」推論「不包就會連結」**。
- JP 官方的寫法與 KR 不同（在名稱中間插 `[TabExplain]`、在假名前也包 `<noparse>`），其中很多依現行規則是多餘的。可能是舊版規則留下來的，沒有辦法從這一版程式確認。

## zh-TW 現況（用 `method/kwsim.py` 推算，未在遊戲中逐一確認）

資料：`raw-20261112` 的 KR 與 `tw` 分支 7fccf1d。範圍：上表會處理的欄位（`Skills*`、`Passive*`、`EGOgift*`、`PanicInfo*`），共 28,876 個欄位。

- 純文字名稱的自動連結共 5,753 處，前幾名是流血 643、破裂 496、震顫 464、充能 362、呼吸法 347、沉淪 287、燒傷 259、束縛 202。
- **同一欄位 KR 沒有、譯文多出來的連結：382 個**。最多的是：
  - **`ConfessionThirdPhase`「～」68 個欄位**。這個狀態 KR／EN 的名稱是「〜」（U+301C），譯文是「～」（U+FF5E），而譯文的數值範圍也寫「1～2層」。「～」後面接數字，會被當成這個狀態的關鍵字，在「～」前面多出圖示並變成連結。JP 官方同樣把名稱寫成「～」，做法是把條目標成 `excludeAutoLink`。**建議**：把譯名改回「〜」，或在 `Bufs-a1c10p3.json` 的這個條目加 `"excludeAutoLink": true`。
  - 流血 35、攻擊等級+ 21、沉淪 19、彈藥 14、丟棄 10、破裂 10、傷害強化 10、震顫 10、本能 10……多半是 KR 寫成一般詞語（後面不是空格）而譯文在名稱後留了空格，或譯文的寫法剛好是「名稱＋數字」（「攻擊等級+1」）。
- **KR 有、譯文沒有的連結：429 個**。最多的是「피해량 증가」（`AttackDmgUp`）91 個：KR 把「傷害增加」當一般敘述寫，後面剛好是空格而被連結；譯文寫成「造成的傷害+10%」就沒有。這類不一定是錯。
- 譯文裡單一個字的狀態名稱有 11 個：`掌`、`棺`、`蝶`、`喜`、`待`、`始`、`援`、`針`、`鬼`、`～`、`連`（`連` 已有 `excludeAutoLink`）。單字名稱最容易在句尾、空格前、數字前、「」前誤中；JP 官方把 `始`、`種`、`～` 標成 `excludeAutoLink`。
- 譯文的 `[TabExplain]` 位置與數量和 KR 完全相同（192 處），沒有出現在不處理的欄位。
- 沒有把 `[ID]` 包在 `<noparse>` 裡的情形；沒有使用 U+200B。

## 建議的檢查

1. **連結差異**：對上表會處理的欄位，分別用 KR 與譯文算出「會出現的連結 id 集合」（純文字名稱＋`[ID]`），列出譯文多出來的與少掉的。多出來的幾乎都該處理（改寫、加 `<noparse>` 或 `excludeAutoLink`）；少掉的要人看。`method/kwsim.py diff` 就是這個檢查。
2. **不處理的欄位裡的方括號**：`[TabExplain]` 與技能標籤不該出現在 `EGOgift.desc`、`PanicInfo`、`Bufs`、`BattleKeywords` 等欄位（除非 KR 同一欄位也有）。
3. **處理的欄位裡的方括號**：技能、硬幣、被動文字裡每一個 `[xxx]`，xxx 必須是 KR 同一欄位出現過的 id（或至少是 `SkillTag.json`／`Bufs*.json` 的 id），否則畫面上是 `Unknown`。
4. **`<noparse>` 的內容**：裡面不可以有 `[`、`<`。
5. **`[TabExplain]` 前面**：不可以是空白。
6. **狀態名稱**：`Bufs*.json` 與 `BattleKeywords*.json` 同 id 的 `name` 是否一致（KR 不一致的 14 個除外）；`Bufs*.json` 的名稱是否含 `[`、`]`、`<`、前後空白；單字名稱與常用詞（如「～」）是否需要 `excludeAutoLink`；KR 有 `excludeAutoLink` 的條目，譯文是否保留。
7. **`BattleUIText.json` 的 `cantIdentify`** 必須等於 `<color=#ff0000>` ＋ `SkillTag.json` 的 `CantIdentify` 名稱 ＋ `</color>`（`GetOverClockSkillString` 0x28658F0 用字串比對決定要不要加上或拿掉這段文字；現行譯文符合）。
