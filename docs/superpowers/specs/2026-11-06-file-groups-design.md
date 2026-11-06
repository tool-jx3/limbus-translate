# 依路徑與檔名分群、「詞彙」頁、條目 `check: warn` 設計

日期：2026-11-06。前一階段（`docs/superpowers/plans/2026-11-06-glossary-context.md`，已合併到 main 5b7b4f6）做了 `family:` 範圍、`also`、長詞條優先、敘事族群只警告與「詞彙範圍」分頁。

## 為什麼

使用者的話：

- 「重新設計檔案分群，以路徑與檔名來分群」
- 「我想要多一個頁面可以看所有詞彙，以及該詞彙使用的條目與未使用的條目」
- 「因為現在還是有 190 多個詞彙未能有效分類」

現在的族群（`guides/routing.yaml` 的 12 個）是人取的名字加萬用字元，看名字看不出包含哪些檔。最大的 `other` 有 562 個檔，技能關鍵字、介面文字、關卡、活動、RPG 都混在裡面。`mirror`、`walpu-refraction` 依「玩法、季節」橫切各種文字類型。結果有兩個：

- 網站側欄與審核篩選的分組幫不上忙。
- 詞彙範圍的統計常落在混在一起的大組，分不出這個詞到底在哪裡不用，所以 192 組 term-missing 很難一次決定。

另外，192 組 term-missing 共 2335 個單位，約七成是第一階段匯入的社群譯文。決定它們要能看到每個詞條「用了」與「沒用」的全部單位。

## 使用者已決定的事

（2026-11-06 對話中以選項回答）

1. **兩層都改。** 網站的顯示與統計改用路徑與檔名推出的分群。族群規則（翻譯指引、詞彙表退回／只警告、待譯順序、`family:` 範圍）也依路徑與檔名重切。
2. **分群自動推出兩層樹。**
   - 第一層：目錄，或根目錄檔名開頭的英文字。
   - 第二層：檔名 `-` 之前的部分，StoryData 依開頭字母。
   - 新檔自動歸組，不用維護。
3. **族群規則採「具名族群，由樹的節點組成」。**
   - 族群仍是文字類型（劇情、語音、技能、關鍵字、介面……）。
   - 每個族群改用樹的節點列出，拆掉 `other` 與 `mirror`。
   - 每個頂層節點都要明確歸到某族群，沒歸的 lint 警告。
   - 新的切法先用實際資料提案，給使用者看過再定。
4. **新的「詞彙」頁是獨立頁面**，網站頂端加連結。列表頁列出全部詞條；點進去看用了譯名與沒用譯名的全部單位。
5. **詞彙表條目加 `check: warn`**：該條目缺譯名時，送出只警告、不退回。

## 目標

1. **分群樹**：每個檔由路徑推出 `top` 與 `sub` 兩層節點（下節）。只看路徑本身，不看其他檔。
2. **族群由節點組成**：`guides/routing.yaml` 的族群改寫成 `nodes`（節點樣式），取代 `match`（檔名萬用字元）。比對規則見下節。每個頂層節點都要有族群；沒歸到的檔落在唯一的後備族群 `unsorted`，lint、`lct plan`、`lct suggest` 照現行「預設族群的新檔」機制提醒。
3. **新的族群切法**：依實際資料提案（另見提案頁），共 16 個族群加 `unsorted`。使用者確認後寫進 `guides/routing.yaml`，並更新 `guides/families.md`。
4. **詞彙表範圍可用節點**：`files`、`not_files` 的項目多一種 `group:<節點樣式>`，與 `family:<id>`、檔名萬用字元並列。
5. **網站側欄改成分群樹**：頂層 → 子群 → 檔，每層顯示檔數，篩選時展開有命中的。族群移到首頁：列出各族群與它的節點、檔數、`glossary` 設定。
6. **詞彙範圍統計的第二種分法改成樹的頂層節點**，取代現在的 5 個目錄加「其餘」。建議的範圍只用 `family:<id>` 或 `group:<頂層節點>`。
7. **「詞彙」頁**（`#/g`）：
   - 列表：全部詞條，可搜尋、排序、篩選。
   - 詳情（`#/g/<條目 ID>`）：該詞條命中的每個單位，分「沒用譯名」與「用了譯名」兩區，可依族群、分群、來源篩選。
8. **條目 `check: warn`**：送出時該條目缺譯名不升為硬性。批次的詞彙表帶出 `check`，讓譯者知道這只是預設譯名。

## 不做的（non-goals）

- 季節／活動代號（檔名 `-` 之後的 `walpu`、`a1c`、`tkt`……）不當成分群或族群。
  - 規則上需要它的兩個詞彙條目（`흐트러짐 구간`、`키워드`，現在寫 `family:walpu-refraction`）改寫成原本族群的檔名萬用字元 `*walpu*`、`*Refraction*`、`Railway*`。條目的範圍不變。
- 第三層分群（例如 StoryData 的章節 `S1`、`S10`）。
- 在網站上直接改規則：網站唯讀，決定照舊經由 `/review-page`。
- 改詞彙表條目的譯名或範圍：這次只改機制與族群，條目內容由使用者之後用新頁面決定。

## 反目標（做了也算失敗）

- **改族群後，已確認或已拒絕的項目重新冒出來。**
  - 例如 `rules/declined.yaml` 的 `inconsistent/<族群>/…` ID 內含族群 id。現有 14 筆都在 `story-main`、`story-interlude`、`identity`，這三個 id 保留。
  - 驗收要確認這 14 筆在新切法下仍對得上。
- **族群變了，詞彙表的退回／只警告悄悄改變。** 驗收列出 `glossary` 模式改變的每個檔。提案切法只有 10 個檔改變：
  - `AbEvents-a1c*` 9 個檔從 warn 變 hard（併入異想體事件）；
  - `StoryData/RPG1.json` 從 hard 變 warn。
- **節點樣式比對不到檔、或同一節點被兩個族群搶。** 處理方式：
  - 比對不到檔：有 raw 時 lint 警告；
  - 同層兩個族群都符合：取路由裡先列的族群，lint 另外列出這種節點，讓人知道誰搶到；
  - 子群一律優先於頂層。
- **「詞彙」頁大到打不開。**
  - 資料依詞條分檔，點開才載入；
  - 「用了譯名」的單位只存片段；
  - 驗收量最大一檔（대상，6151 個單位）的大小與載入時間。

## 分群樹的推導（使用者決定的規則，細節為我的判斷）

先去掉語言前綴（`KR_` 等，和路由現在一樣），再看相對路徑。

- **檔在目錄裡**（`StoryData/S1037B.json`）：
  - `top` 是 `目錄/`（`StoryData/`）。
  - `StoryData/` 的 `sub` 是檔名開頭的英文字母（跳過開頭的數字）：`S`、`D`（`1D101A`）、`E`、`ES`、`P`、`PC`、`RPG`。這與 `guides/families.md` 的 StoryData 命名表一致。
  - 其他目錄的 `sub` 是檔名以 `_`、`-` 切開的前兩段，用 `_` 接起來：
    - `PersonalityVoiceDlg/Voice_Heathcliff`
    - `BattleAnnouncerDlg/Announcer_Charon`
    - `RPGSystem/rpg_loc`
  - 寫成 `目錄/子群`。
- **檔在根目錄**（`Skills_Enemy-mowe.json`）：
  - `top` 是檔名開頭連續的英文字母（`Skills`）。
  - `sub` 是檔名 `-` 之前的部分，去掉結尾的數字與 `_`（`Skills_Enemy`；`StageNode91` 變 `StageNode`；`MirrorDungeonUI_` 變 `MirrorDungeonUI`）。
  - `sub` 等於 `top` 時，這個檔沒有子群。
- **顯示時合併**（只影響網站，不影響規則）：
  - 只有一個檔的子群不顯示，檔直接列在頂層下。
  - 頂層底下只剩一個子群時也不顯示子群。

實際資料（raw-20261105，2422 個 KR 檔）：165 個頂層、384 個節點。提案頁列出全部。

**節點樣式**：用 `path.Match`，比對節點名稱，`*` 不跨 `/`。

| 寫法 | 指的是 |
|---|---|
| `Skills` | 頂層 |
| `Skills_Enemy*` | 根目錄的子群 |
| `StoryData/` | 目錄頂層 |
| `StoryData/E*` | 目錄子群 |
| `*UIText` | 名稱以 UIText 結尾的頂層 |

## 族群由節點組成（我的判斷）

`guides/routing.yaml`：

```yaml
families:
  - id: story-main
    nodes: ["StoryData/S", "StoryData/D", "StoryData/RPG", "DungeonNode", "BgmLyrics/"]
    glossary: warn
    guide: guides/families.md
  ...
  - id: unsorted
    nodes: []          # 後備：沒有節點歸到的檔；只能有一個，放最後
    guide: guides/families.md
```

**比對**：一個檔的族群照下面的順序決定。

1. 先用它的 `sub`，依路由順序找第一個 `nodes` 有樣式符合的族群。
2. 沒有，再用 `top` 做同樣的事。
3. 都沒有，就是 `unsorted`。

`Family(file)` 仍只看路徑。

**lint**：

- `nodes` 樣式寫錯：錯誤。
- 有 raw 時：
  - 樣式比對不到任何節點：警告；
  - 有頂層節點的檔落在 `unsorted`：警告，列出節點；
  - 同一節點在同一層被兩個族群的樣式符合：警告，說明取了哪一個。
- `match` 欄位不再接受：錯誤，訊息指向 `nodes`。

**沿用的機制**：

- `priority`
- `glossary: hard|warn`
- `family:<id>`
- 「預設族群的新檔」提醒（`DefaultFamily` 改指 `unsorted`）

## 提案的族群切法（待使用者確認）

| 族群 | glossary | 節點 | 檔 | 單位 | 主要來自 |
|---|---|---|---|---|---|
| story-main | warn | `StoryData/S`、`StoryData/D`、`StoryData/RPG`、`DungeonNode`、`BgmLyrics/` | 591 | 69863 | story-main |
| story-interlude | warn | `StoryData/E`、`StoryData/ES`、`StoryData/PC` | 256 | 39153 | story-interlude |
| identity | warn | `StoryData/P`、`PersonalityVoiceDlg/`、`BattleSpeechBubbleDlg` | 386 | 44459 | identity |
| ego-voice | warn | `EGOVoiceDig/` | 14 | 1124 | ego-voice |
| announcer | warn | `BattleAnnouncerDlg/` | 65 | 3931 | announcer |
| dante-note | warn | `StoryTheaterDanteNote*` | 26 | 731 | dante-note |
| ally | hard | 我方技能、被動、人格、E.G.O、協助者 | 80 | 17544 | ally、other 9、walpu 4 |
| enemy | hard | `Enemies`、`Skills`、`Skills_Enemy*`、`Skills_Abnormality*`、`Passives_Enemy*`、`Passives_Abnormality*` | 205 | 22875 | enemy、walpu 36、mirror 5 |
| keywords（新） | hard | `Bufs*`、`BattleKeywords*`、`UnitKeyword`、`SkillTag`、`BuffAbilities`、`PanicInfo`、`MentalCondition`…… | 153 | 18398 | other 118、walpu 23、mirror 12 |
| abnormality | hard | `AbnormalityGuides*`、`AbDlg`、`AbEvents*`、`AbEventsResultLog*` | 91 | 4063 | abnormality、walpu 21、story-main 9、mirror 7 |
| mirror | hard | `MirrorDungeon*`、`EGOgift*`、`ActionEvents*`、`DungeonStartBuffs`、`TutorialMirrorDungeon`、`StoryTheaterMirrorWorld*`、`EgoGiftCategory` | 95 | 10491 | mirror、walpu 12 |
| railway（新） | hard | `Railway*`、`Walpu` | 26 | 852 | walpu-refraction |
| rpg（新） | hard | `RPGSystem/` | 189 | 20258 | other |
| stage（新） | hard | 關卡、地牢、選擇事件、劇場…… | 95 | 6086 | other |
| events（新） | hard | `Event*`、`*Event`、出席、活動文字…… | 25 | 1057 | other |
| ui（新） | hard | `*UI`、`*UIText`、`Tutorial*`、商店、通行證、說明…… | 125 | 11990 | other 121、mirror 4 |
| unsorted | hard | （後備） | 0 | 0 | |

- `walpu-refraction` 拆進各文字類型。
- `mirror` 只留鏡牢專屬的類型；`Bufs_Mirror` 這類子群依類型歸到 keywords 等。
- **priority**：story-main → story-interlude → ego-voice → announcer → identity → ally → enemy → keywords → abnormality → dante-note → mirror → railway → rpg → stage → events → ui → unsorted。
- 完整的節點對照（每個頂層、子群歸哪個族群、現在在哪個族群）放在提案頁，確認後由實作照抄。

## 詞彙表 `group:` 範圍

- `files`、`not_files` 的項目可以是 `group:<節點樣式>`。檔的 `top` 或 `sub` 有一個符合，就算命中。
- lint 檢查樣式語法；有 raw 時，比對不到節點就警告，同 `family:`。
- `schemas/batch.schema.json`、`glossary/README.md`、`prompts/translate.md` 一併說明。
- **既有條目的遷移**：
  - `family:walpu-refraction` 兩條改成 `*walpu*`、`*Refraction*`、`Railway*`；
  - `family:story-main`、`story-interlude`、`identity`、`ally` 照舊，但族群的檔變了，驗收列出命中改變的單位數。

## 網站

- **側欄**：分群樹。頂層依檔數多到少排，子群同樣排序，子群下列檔。目前檔所在的頂層與子群展開。篩選時，有命中的全部展開。
- **首頁**：族群表，欄位是族群、glossary、節點、檔數。
- **審核頁的「族群」篩選**：照舊用族群（族群變細了，篩選就有用）。
- **`index.json`**：每個檔加 `top`、`sub`；另有 `groups`（顯示用的樹，已合併單檔子群）。

## 詞彙範圍統計

- `internal/scope` 的第二種分法從 `Dirs`（5 個目錄＋其餘）改成樹的頂層節點。
- 群組名是 `group:<頂層>`，建議的樣式也用這個寫法。
- 門檻不變。
- 頁面與 `/review-page` 的說明改用新寫法。

## 「詞彙」頁

**路由**

- `#/g`：列表。query：`q` 搜尋、`only=missing` 只看有沒用的、`family` 篩族群。
- `#/g/<條目 ID>`：詳情。條目 ID 同詞彙範圍的 `glossary/<sha8 KR>/<sha8 ZH>`。query：`family`、`group`、`origin`、`q`。

頂端加「詞彙」連結。

**列表**：全部 595 條，欄位：

- 原詞
- 譯名（含 `also`）
- `check`
- 範圍：`files`、`not_files`、`match` 的摘要
- 出現、用了、沒用、沒用比例

預設依「沒用」由多到少排序，也可以依原詞排序。

**詳情**

- **上方**：
  - 條目內容：`notes`、範圍、`check`；
  - 各族群與各頂層的出現／用了；
  - 連到審核頁的詞彙群組（`termgroup/…`）與詞彙範圍的條目。
- **「沒用譯名」區**：全部單位。
  - 每個單位附：KR（標出原詞）、TW 全文、來源、note、已確認標記、到檔案那一列的連結、「複製 ID」（`warning/term-missing/…`，照現行 `/review-page` 流程決定）。
  - 另有「複製這些單位」：把目前篩選下的單位 ID 一次複製，供整批重翻、確認或保留。
- **「用了譯名」區**：每次顯示 100 個，有「再顯示 100 個」。
  - 每個單位附：KR、TW 片段（原詞與譯名前後各約 30 字）、來源、連結。
- **篩選**：族群、頂層分群、來源（paratranz、claude、human）、文字。

**命中的定義**

- 和審核頁、詞彙範圍完全相同，經由 `Rules.Applies`、`Term.OccursIn`、長詞條優先，有效暫定詞參與覆蓋。
- 是否「用了」以 `Term.Used` 判斷。

**資料**

- `data/glossary/index.json`：列表所需。可以和 `review/scope.json` 的 Entry 共用計算。
- `data/glossary/<sha8 KR>-<sha8 ZH>.json`：該條目的單位。
  - 「沒用」存全文，各上限 400 字。
  - 「用了」存片段。

**估計**

- 全部 77,544 個命中，其中 2,326 個沒用。
- 最大的是 대상，6151 個單位，估 3 MB 以內，gzip 後約 1 MB。
- 驗收實測。

## 條目 `check: warn`

- `glossary/terms.yaml` 的條目可以寫 `check: warn`。省略或 `hard` 是現在的行為。
- **送出時**：
  - 條目是 `warn`，或檔的族群是 `glossary: warn`，缺譯名都只警告。
  - `forbidden` 照舊退回。
- **建置、審核頁、詞彙範圍**：照舊列出缺譯名的單位，不因 `warn` 消失；頁面標出條目是 `warn`。
- **批次**：詞彙表帶 `check`。同時更新 `schemas/batch.schema.json`、`prompts/translate.md`、`translate-pending` skill 的說明與 `cmd/lct/contract_test.go`。
- **lint**：值只能是 `hard`、`warn`。
- **`/review-page`**：多一種決定，「暫時只警告」（加 `check: warn`，`notes` 寫日期與理由）。

## 驗收（真實資料，scratch clone）

- **族群**：
  - 每個族群的檔數、單位數與上表一致；
  - `unsorted` 為 0；
  - lint 0 錯誤。
  - 列出警告：同層兩族群搶同一節點的清單。
- **glossary 模式**：改變的檔就是上面那 10 個。
- **已存的決定**：
  - `rules/declined.yaml` 的 14 筆 `inconsistent/…` 仍對得上（不重新出現）；
  - 審核頁各分頁數字和改前比較，差異逐項說明。
- **詞彙表命中**：term-missing 群組數與單位數和改前比較，差異逐項說明。
- **詞彙範圍**：建議數與種類改前改後對照。
- **「詞彙」頁**：
  - `data/glossary/` 總大小、最大一檔大小；
  - 대상 詳情頁的載入時間與 400px 版面。
- **時間**：`lct site` 改前改後各三次。

## 我的判斷（請確認）

1. 推導細節：
   - StoryData 子群用開頭字母（跳過數字）；
   - 其他目錄用前兩段；
   - 根目錄子群去掉結尾數字與 `_`。
2. 比對順序：子群優先於頂層；同層取路由先列的族群；沒歸到的落在 `unsorted`。
3. 季節代號不分群；兩個 `family:walpu-refraction` 條目改寫成原萬用字元。
4. 提案的 16 個族群與 priority。特別是：
   - `AbEvents-a1c*` 併入 abnormality（9 檔 warn → hard）；
   - `RPGSystem/` 自成 `rpg` 並維持 hard；
   - `StoryData/RPG1` 歸 story-main。
5. 詞彙範圍的第二種分法改成頂層節點，建議用 `group:<頂層>`。
6. 首頁顯示族群表，側欄改成分群樹，審核篩選照舊用族群。
7. 「詞彙」頁：
   - 「沒用」存全文、「用了」存片段；
   - 「用了」每次顯示 100 個；
   - 列表預設依沒用數排序。
8. `check: warn` 只影響送出，建置與審核照舊列出。

## 留給實作的

- `rules` 套件裡節點推導與比對的函式名稱與資料結構。
- `index.json`、`data/glossary/*.json` 的確切欄位，寫進計畫的 Interfaces。
- 側欄、列表、詳情的版面細節，沿用現有元件（`ItemList`、`IdCopy`、`UnitLink`、`OriginBadge`、`Row` 的標記）。
