# 詞彙表

`terms.yaml` 是硬性規定：翻譯時注入命中的條目，驗證時檢查禁用譯法。

## 術語相關（原指引）

### 為方便在專案平台上搜尋以及使用，將術語的備註中標註說明，並不排除未來以代號形式做備註。

- 輔助文本描述——**該術語出現在多類文本中**
- 輔助戰鬥相關文本描述——**該術語大多出現在戰鬥文本中**

***以上兩者可以同時存在***

- 異想體——**該術語為異想體**
- BUFF／DEBUFF——**該術語多次或僅出現在 BUFF／DEBUFF 中**
- 僅限 XX——**某一人物專用且僅出現在特定場合（多為觀察報告）**
- 都市術語——**該術語為都市術語，不會在現實世界中出現或是現實世界中的意思有所出入**
- 罪孽／物理屬性／機制——**對應的罪孽屬性／物理屬性以及其機制**
  - 如：憂鬱、打擊、呼吸法……等

## `category` 代號

原指引說「不排除未來以代號形式做備註」，`terms.yaml` 先用以下代號，一個詞可以有多個：

| 代號 | 原指引分類 |
|---|---|
| `general` | 輔助文本描述 |
| `combat` | 輔助戰鬥相關文本描述 |
| `abnormality` | 異想體 |
| `buff` | BUFF／DEBUFF |
| `only:<角色id>` | 僅限 XX（例如 `only:yisang`） |
| `city` | 都市術語 |
| `sin` / `physical` / `mechanic` | 罪孽屬性／物理屬性／機制 |
| `system` | 系統類譯名（「翻譯名稱統一列表」） |
| `ego` | E.G.O 與 E.G.O 飾品的名稱（Paratranz 備註「E.G.O飾品」「E.G.O名稱」等） |

## `files`：只適用於部分檔案

同一個詞在不同的檔案要有不同譯法時（例如「신부」在第十章是牧師、其他章節是神父），用 `files` 限定條目適用的檔：

- `files` 是檔名樣式清單，寫法同 `lct.yaml` 的 `frozen`：`StoryData/S10[0-9][0-9]*`、`*a1c10*`；以 `/**` 結尾的樣式涵蓋該目錄底下所有檔（`RPGSystem/**`）。省略 `files` = 所有檔
- 只有批次的檔名符合時，條目才會注入批次，驗證時才檢查它的 `forbidden`
- 同一個 `kr` 可以有多個條目，其中至多一個 `status: approved` 的條目不設 `files`（`lct rules lint` 檢查；`candidate` 不計，因為它不生效）。某個檔符合有 `files` 的條目時，由那些條目代表這個詞，沒設 `files` 的那一條在這個檔不適用；沒有條目符合就用沒設 `files` 的那一條
- `files` 重疊的條目會同時生效：一個檔符合同一個 `kr` 的多個有 `files` 的條目時，這些條目全部注入批次、全部檢查 `forbidden`。要讓譯法互斥，`files` 的樣式就不要重疊
- 沒設 `files` 的那一條要在 `notes` 指向其他條目，讓人知道哪些檔有例外

## 比對方式：預設整詞，`match: substring` 才比對長詞裡的用法

條目預設只在 KR 單獨成詞時命中（等同 `match: word`，可以省略不寫）：

- 前面不接韓文字，後面沒有韓文字，或接的是助詞、語尾（이/가/을/를/의/에서/로……、하다/되다 的變化）
- 「속도로」「속도가」算，「약속도」「고속도로」不算；「백화가」「백화를」「백화된」算，「백화점」不算；「버스에」算，「림버스」不算；「도시」命中「도시가」，不命中「도시락」
- 這樣短詞不會命中意思不同的長詞，翻譯時不會被注入錯的譯名，也不會因為長詞裡沒有這個譯名而在送出時被 `term-missing` 退回

少數條目的 KR 本來就要在長詞裡命中，才加 `match: substring`：長詞的譯文也含這個譯名時（例如「사무소」命中「정사무소」→ 事務所、「구인회」命中「신구인회」→ 新九人會），用字串包含比對，否則會少了有用的提示。`match` 只有 `word`（預設）與 `substring` 兩個值，其他值 `lct rules lint` 會報錯。批次的詞彙表欄位在條目是 `substring` 時帶 `match: substring`，譯者看得到。

2026-11-04 之前，條目預設以字串包含比對，當時有 19 條加了 `match: word`（백화、버스、제자、세가、지부、오만、아비、심상、마름、보스、키라、가주、소다、카포、복도、아인、슬롯、관통、탐식）。這個寫法照舊有效，現在與省略 `match` 相同。

## `not_files`：排除檔案

條目在某些檔不該適用時（例如敘事檔的用法與其他檔不同），用 `not_files` 把那些檔排除：

- 寫法同 `files`：檔名樣式清單，`StoryData/**` 涵蓋該目錄底下所有檔，也可以是 `*a1c10*` 這類 glob。例如 `not_files: ["StoryData/**"]` 讓條目不用在 `StoryData/` 底下的檔
- 條目適用於一個檔，要同時滿足：沒設 `files`，或 `files` 有一個樣式符合；而且 `not_files` 沒有樣式符合。兩者都設時是這兩個條件的交集
- 上一節 `files` 的優先規則看「適用」：有 `files` 的條目只在它適用的檔（`files` 符合且 `not_files` 不符合）代表這個詞，沒設 `files` 的那一條才在該檔不適用；有 `files` 的條目被 `not_files` 排除的檔，仍用沒設 `files` 的那一條。要讓兩條互補，一條設 `not_files`、另一條設相同樣式的 `files`（例如「신부」：神父那條 `not_files: ["StoryData/S10[0-9][0-9]*"]`，牧師那條 `files: ["StoryData/S10[0-9][0-9]*"]`），每個檔恰好一條適用
- `lct rules lint` 檢查樣式寫法，也會在 `files` 的每個樣式都同時寫在 `not_files`（條目在任何檔都不適用）時報錯；兩個不同的樣式是否重疊不檢查
- 批次是一檔一檔切的，篩選在注入前就做完，批次的詞彙表欄位不帶 `not_files`

## 之後要補的

- 狀態效果、關鍵字（`BattleKeywords*`、`Bufs*`）與角色、NPC 名稱（`ScenarioModelCodes-AutoCreated`）：`lct glossary derive` 已能列出譯文用到、詞彙表還沒有的名稱與現行譯名（見下），由人以 `term-decision` skill 逐一決定。還沒做的是以 id 對齊 EN／JP，把條目的 `en`、`jp` 一併填好

## `lct glossary derive`：找出還沒列進詞彙表的名稱

`lct glossary derive [--source keyword|speaker] [--limit N] [--offset N] [--json]` 列出本專案目前譯文中的關鍵字名稱（`BattleKeywords*`、`Bufs*`）與說話者名稱（`ScenarioModelCodes-AutoCreated`），作為新增詞條的候選。只輸出，不寫檔。

- KR 名稱已經是任何詞條的 `kr`（不論 `status`、`files`）的不列
- `rules/declined.yaml` 已有 `term/<kr>` 的不列
- 譯名仍含韓文（尚未翻譯、KR 原文已改、待審）的不列
- 同一個 KR 名稱有不同譯法時，每種譯法各列一次
- 候選依 KR 排序，有一千多個：`--limit N` 與 `--offset N` 取其中一頁，`--json` 的結果是 `{total, candidates}`，`total` 是分頁前的總數。採用與不採用的詞不再列出，所以下一頁的 `--offset` 是略過的候選數
- 用 `lct glossary derive` 看到的名稱由人決定：採用就寫進 `terms.yaml`（`status: approved`），不採用就記進 `rules/declined.yaml`（`id: term/<kr>`）
