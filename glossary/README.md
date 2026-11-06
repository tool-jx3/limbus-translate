# 詞彙表

`terms.yaml` 的譯名是專案的預設譯名：翻譯時注入命中的條目，驗證時檢查禁用譯法。缺少 `zh` 預設只警告，不退回（見下一節 `check`）。

`guides/routing.yaml` 標 `glossary: warn` 的族群（列表以 routing.yaml 為準，不在這裡重抄）語境多變：即使條目有 `check: hard`，缺少 `zh` 也只警告，不退回；譯者語境不合可以不用，仍在 `note` 寫原因。`forbidden` 的禁用譯法在這些族群照樣退回。

## `check`：缺少譯名時退回

`check` 的值是 `hard` 或 `warn`，省略等於 `warn`（`lct rules lint` 檢查）：

- 省略或 `warn`：譯文缺少 `zh`（也沒有 `also`）只警告，送出時照樣收下，警告列在審核頁
- `hard`：缺少 `zh` 時，送出會退回，除非該單位的 `note` 寫出條目的 KR 原詞（說明為何在此不照條目譯）；`glossary: warn` 的族群仍只警告。暫定詞沒有 `check`，一律只警告
- `forbidden` 與 `check` 無關，譯文含禁用譯法一律退回
- 改為 `hard` 要有依據：在 `notes` 寫日期、應用率與出現數。網站「詞彙」頁（`#/g?only=hard`）列有「可考慮退回」的條目（應用率高於 95%、至少 20 個單位）；由人決定，經 `review-page` skill 改，`bin/lct glossary scope --json` 是數字的來源。改回只警告就刪掉 `check`

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

## `also`：`zh` 以外的其他正確譯法

同一個 `kr` 依語境有不只一種正確譯法時（例如「대상」多數寫「目標」，指人時寫「對象」），把 `zh` 以外的譯法列在 `also`，並在 `notes` 說明何時用哪一個：

- 譯文（去掉標籤後）含 `zh` 或任一 `also` 就算用了這個條目，不會有 `term-missing`，送出時選 `also` 不需要 note；審核頁的詞彙群組不列這樣的單位，`term-new` 的「有用」也照此計算
- 批次的詞彙表欄位帶著 `also`，譯者依 `notes` 在 `zh` 與 `also` 之間選
- `also` 不得有空字串、不得與 `zh` 相同，也不得被 `forbidden` 擋下（與 `forbidden` 的項目相同、含有它，或符合它的 `re:`），`lct rules lint` 報錯
- 詞彙群組 ID 與逐單位 ID 仍以 `zh` 計，加 `also` 不改 ID
- `also` 只放對這個 `kr` 本來就對的譯法；某些檔才用的譯法用 `files` 另開條目

## `files`：只適用於部分檔案

同一個詞在不同的檔案要有不同譯法時（例如「신부」在第十章是牧師、其他章節是神父），用 `files` 限定條目適用的檔：

- `files` 是檔名樣式清單，寫法同 `lct.yaml` 的 `frozen`：`StoryData/S10[0-9][0-9]*`、`*a1c10*`；以 `/**` 結尾的樣式涵蓋該目錄底下所有檔（`RPGSystem/**`）。`*` 不跨 `/`：`RPGSystem*` 不比對 `RPGSystem/` 底下的檔，要寫 `RPGSystem/**`。省略 `files` = 所有檔
- `files` 的項目也可以寫 `family:<id>`，指 `guides/routing.yaml` 路由判為該族群的檔（`lct` 用 `Family(file)`，路由由上往下第一個符合的族群），例如 `files: ["family:story-main"]` 只適用於主線劇情族群的檔；族群裡的檔隨路由增減，不必逐一列檔名樣式。可以和檔名樣式混用（`files: ["family:mirror", "Enemies-a1c10*"]`，符合任一項就適用）。`<id>` 不在路由的族群裡，`lct rules lint` 報錯；有 raw 時，族群裡沒有任何 KR 包的檔則給警告。批次的詞彙表欄位原樣帶著 `family:<id>`，它指批次 `family` 為 <id> 的單位
- `files`、`not_files` 的項目另有四種寫法，依檔的節點與第三層標記（`FileNode`、`FileTag`）判斷：
  - `group:<樣式>`：檔的節點（子節點或頂層節點）符合 `path.Match` 樣式，`*` 不跨 `/`。例如 `group:Skills_Enemy*`、`group:StoryData/`
  - `chapter:<數字>`：章節標記為該章的檔，例如 `chapter:10`（`StoryData/S1037B.json`）
  - `character:<id>`：角色標記為該角色的檔，`<id>` 是 `guides/characters/` 的角色 id，例如 `character:heathcliff`
  - `event:<樣式>`：活動標記符合樣式的檔，例如 `event:walpu*`（`Bufs-walpu7.json`）
  - 這些寫法可以和檔名樣式、`family:<id>` 混用，在 `files` 與 `not_files` 的效力相同。`chapter:` 的值不是數字、`character:` 不是已有的角色 id、`group:`／`event:` 的樣式不合法或是空的，`lct rules lint` 報錯；有 raw 時，比對不到任何 KR 包的檔則給警告
  - 批次的詞彙表條目原樣帶著這些寫法，另有 `units`：批次裡要用這個條目的單位的 `key`，已經算進範圍與長詞條優先
- 只有批次的檔名符合時，條目才會注入批次，驗證時才檢查它的 `forbidden`
- 同一個 `kr` 可以有多個條目，其中至多一個 `status: approved` 的條目不設 `files`（`lct rules lint` 檢查；`candidate` 不計，因為它不生效）。某個檔有適用的、設了 `files` 的條目（`files` 符合，且 `not_files` 不符合，見下面的 `not_files`）時，由那些條目代表這個詞，沒設 `files` 的那一條在這個檔不適用；沒有這樣的條目就用沒設 `files` 的那一條
- `files` 重疊的條目會同時生效：一個檔符合同一個 `kr` 的多個有 `files` 的條目時，這些條目全部注入批次、全部檢查 `forbidden`。要讓譯法互斥，`files` 的樣式就不要重疊
- 沒設 `files` 的那一條要在 `notes` 指向其他條目，讓人知道哪些檔有例外

## 比對方式：預設整詞，`match: substring` 才比對長詞裡的用法

條目預設只在 KR 單獨成詞時命中（等同 `match: word`，可以省略不寫）：

- 前面不接韓文字，後面沒有韓文字，或接的是助詞、語尾（이/가/을/를/의/에서/로……、하다/되다 的變化）
- 「속도로」「속도가」算（「도시잖아요」「복도니까요」「관리자답게」也算），「약속도」「고속도로」不算；「백화가」「백화를」「백화된」算，「백화점」不算；「버스에」算，「림버스」不算；「도시」命中「도시가」，不命中「도시락」
- 這樣短詞不會命中意思不同的長詞，翻譯時不會被注入錯的譯名，也不會因為長詞裡沒有這個譯名而在送出時被 `term-missing` 退回（只有 `check: hard` 的條目在 hard 族群缺譯名才會退回，見 `check` 一節）

少數條目的 KR 本來就要在長詞裡命中，才加 `match: substring`：長詞的譯文也含這個譯名時（例如「사무소」命中「정사무소」→ 事務所、「구인회」命中「신구인회」→ 新九人會），用字串包含比對，否則會少了有用的提示。`match` 只有 `word`（預設）與 `substring` 兩個值，其他值 `lct rules lint` 會報錯。批次的詞彙表欄位在條目設了 `match` 時就帶著（`word` 也帶，`substring` 也帶），沒設就不帶，譯者看得到。

2026-11-06 之前，條目預設以字串包含比對；2026-11-04 起有 19 條加了 `match: word`（백화、버스、제자、세가、지부、오만、아비、심상、마름、보스、키라、가주、소다、카포、복도、아인、슬롯、관통、탐식）。這個寫法照舊有效，現在與省略 `match` 相同。

## `not_files`：排除檔案

條目在某些檔不該適用時（例如敘事檔的用法與其他檔不同），用 `not_files` 把那些檔排除：

- 寫法同 `files`：檔名樣式清單，`StoryData/**` 涵蓋該目錄底下所有檔，也可以是 `*a1c10*` 這類 glob。`*` 不跨 `/`：`RPGSystem*` 不比對 `RPGSystem/` 底下的檔，要寫 `RPGSystem/**`。例如 `not_files: ["StoryData/**"]` 讓條目不用在 `StoryData/` 底下的檔
- 條目適用於一個檔，要同時滿足：沒設 `files`，或 `files` 有一個樣式符合；而且 `not_files` 沒有樣式符合。兩者都設時是這兩個條件的交集
- 上一節 `files` 的優先規則看「適用」：有 `files` 的條目只在它適用的檔（`files` 符合且 `not_files` 不符合）代表這個詞，沒設 `files` 的那一條才在該檔不適用；有 `files` 的條目被 `not_files` 排除的檔，仍用沒設 `files` 的那一條。要讓兩條互補，一條設 `not_files`、另一條設相同樣式的 `files`（例如「신부」：神父那條 `not_files: ["StoryData/S10[0-9][0-9]*"]`，牧師那條 `files: ["StoryData/S10[0-9][0-9]*"]`），每個檔恰好一條適用
- `not_files` 的項目同樣可以寫 `family:<id>`，把整個族群的檔排除，也可以和樣式混用，例如 `not_files: ["family:story-main"]` 讓條目不用在主線劇情族群的檔；比起把族群的檔名樣式抄過來，優先用 `family:<id>`（族群由節點組成，檔名樣式容易多排除到、或漏掉路由判給別的族群的檔）。`family:<id>` 的檢查與 `files` 相同
- `lct rules lint` 檢查樣式寫法；`files`、`not_files` 的樣式若沒有比對到 KR 包的任何檔（寫錯檔名、檔案改名），在 raw 工作樹存在時給警告（`warning:`，不算錯誤、不影響結束碼）；也會在 `files` 的每個樣式都同時寫在 `not_files`（條目在任何檔都不適用）時報錯；兩個不同的樣式是否重疊不檢查
- 劇情以外的小檔會併成一個批次，批次裡的單位來自不同的檔，所以批次的詞彙表欄位和 `files` 一樣帶著條目的 `not_files`（設了才帶）：`file` 符合其中一個樣式的單位不適用這個條目

## 長詞條優先：一處 KR 由蓋住它的長條目代表

同一處 KR 被兩個適用的條目命中，其中一個的命中範圍完全蓋住另一個時（例如「조직의 보스」蓋住其中的「보스」），那一處只套用長的那個：

- 每個適用於該檔的條目（`files`、`not_files` 與優先規則照上面各節）依自己的 `match` 在 KR（去掉標籤後，同比對）找出所有命中位置；某一處命中落在另一個條目某一處命中的範圍內、而且那個範圍嚴格較大，這一處不算
- 條目只要還有一處沒被蓋住，就算命中這個單位：「조직의 보스가 보스 전투에 나섰다」兩個條目都命中，「조직의 보스가 왔다」只命中「조직의 보스」
- 只有完全蓋住才讓位：只是部分重疊（「가나다」與「나다라」在「가나다라」）或範圍相同（同一個 `kr` 的兩個條目）時，兩個都套用。`match: substring` 條目在長詞裡的命中一樣參與比較；整詞條目只在它單獨成詞的地方蓋住別的條目
- 被蓋住的條目在該單位不算命中：批次不因這個單位帶它，送出時不檢查它的 `zh` 與 `forbidden`，審核頁的詞彙群組不列這個單位
- 譯者登記、生效中的暫定詞與詞彙表條目一起比較：暫定詞「와일드헌트」在「와일드헌트 히스클리프」裡被詞彙表條目蓋住，那一處不算；反過來，詞彙表條目被暫定詞蓋住的那一處也不算。批次、送出、建置與審核頁的警告、審核頁 `term-new` 的命中都這樣比較；已被人不採用的暫定詞不生效，什麼也不蓋住。既有譯文不因暫定詞本身出現警告（暫定詞在那裡只用來蓋住詞彙表條目）。`lct validate` 與 `lct adjust add` 只看詞彙表，不看暫定詞
- 不必在長條目寫它取代哪些短條目：固定搭配的譯法與短條目不同時，新增長的條目（`kr` 寫完整的搭配）就好
- 2026-11-06 起如此；之前兩個條目在這種地方都套用

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
