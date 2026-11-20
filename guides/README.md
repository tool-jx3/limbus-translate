# 翻譯指引

翻譯時依「檔案族群」與「說話角色」組合出要套用的指引。CI 自動翻譯與 Claude Code 人工介入都讀這裡的同一份檔案。

## 結構

| 路徑 | 內容 | 何時套用 |
|---|---|---|
| `base.md` | 共通原則：原則、代碼與標籤、標點與書寫、用語、人稱與稱呼 | 每一批都套用 |
| `families/<id>.md` | 各族群的文本類型、語氣與人稱、格式慣例、常見錯誤（每份 ≤ 4096 bytes） | 依檔名判斷族群（`routing.yaml` 的 `guide:`） |
| `shared/<主題>.md` | 跨族群、但不是所有族群都用的規則（每份 ≤ 4096 bytes），例如 `skill-text.md`（技能文字）、`story-labels.md`（劇情標籤欄位）、`phrasing.md`（台詞與敘述的語句） | 用到的族群在 `routing.yaml` 以 `include:` 引入；只給部分檔案的（如 `ruby.md`、`story-labels.md`）以 `guides:` 依節點疊加 |
| `families.md` | 給人看的參考：檔名規律、檔案族群、翻譯優先序、凍結項目 | 不送給譯者 |
| `characters/<id>.md` | 角色語言風格 | 該批出現此角色時（劇情看每行的 `model`，RPG 對話看該行的 `speaker` 欄位，對得上角色 id 才套用；語音／`AbDlg` 看檔名） |
| `npcs.yaml` | 不需要指引的路人：`model`、性別、一句身分（見「新說話者」） | 批次中有這些 `model` 說話時（單位或前後文），批次的 `npcs` 帶上他們的性別與身分；審核 Issue 與規則建議不再把這些 `model` 列為新說話者 |
| `../glossary/terms.yaml` | 譯名統一列表（預設譯名；`check: hard` 的條目缺譯名才退回，`check: off` 的一般用語缺譯名也不警告，禁用譯法一律退回） | 每一批注入命中的條目 |
| `../rules/ruby-customize.yaml` | 可以寫 `<rubycustomize=注音>本文</rubycustomize>` 的位置（檔案＋欄位，每個都在遊戲裡測過）與轉換參數；`lct build` 把它換成遊戲能顯示的標籤，其他位置寫了會退回。什麼時候寫見 `shared/ruby.md`「白名單位置的注音」 | 白名單上的單位在批次帶 `ruby_customize: true` |

疊加順序：`base` → 族群 → 該族群的 `include`（共用指引，依序）→ 依節點或標記疊加的指引（`routing.yaml` 的 `guides:`）→ 角色 → 單檔備註。`base.md`、`shared/skill-text.md` 與 `shared/story-labels.md` 的格式規則是硬性規定；詞彙表的譯名是預設（`check: hard` 的條目與 `forbidden` 除外），角色風格不能推翻它們。

`characters/*.md` 開頭的 YAML 是給程式用的對照資料：

- `models`：劇情 `StoryData` 裡代表此角色本人的 `model` 鍵值（含受傷、剪影、愚人節等狀態差分）；RPG 對話的 `speaker` 欄位也用這份對照
- `models_other_self`：過去、幼年等其他時期的本人，語氣可能不同，翻譯時要看劇情判斷
- `models_not_this`：鍵值裡有角色名但不是本人（例如 `뫼르소엄마` 是莫梭的母親），或是多人同時說話
- `file_names`：`AbDlg_`、`PersonalityVoiceDlg/Voice_`、`EGOVoiceDig/Voice_EGO_` 檔名中的拼法（遊戲檔名拼法不一致，如 `Meursalut`、`Outist`）
- `number`：遊戲內罪人編號；`code`：檔名代碼（`P1YYXX`、人格編號 `1YYXX` 的 `YY`）

### 新說話者

審核 Issue 的「未對應的說話者」與規則建議的 `speaker/…` 是這次更新新出現、指引還不認得的 `model`。依情況處理，補上後下次建置就不再列出：

- **既有角色的變體**（受傷、回憶、劇中劇、愚人節等）：補進該角色檔的 `models`（本人）、`models_other_self`（其他時期的本人）或 `models_not_this`（名字相近但不是本人、多人同時說話）
- **同一群人**（同一組織的士兵、職員、成員等，語氣一致）：寫一份共用的 `characters/<id>.md`，`models` 列出這群人的全部 `model`，內容寫身分、語體與稱呼，幾行即可；之後出現同一群人的新 `model` 就補進去
- **有特色的新角色**：新增 `characters/<id>.md`
- **一次性的路人**：不需要指引，在 `npcs.yaml` 加一筆。`model` 必填；`gender`（`male`／`female`／`unspecified`）看得出才寫；`note` 寫一句身分（出處、第幾章）。名單上的 `model` 不能同時列在角色檔裡（`lct rules lint` 會檢查）

## 資料依據

統計用的原文、繁中與對照資料，以及計算方法，移到 `docs/guides-evidence/README.md`「資料依據」。

## 寫法規範

會送給譯者的指引（`base.md`、`families/*.md`、`shared/*.md`、`characters/*.md`）只寫譯者照著做的規則。統計、來源、經過寫進依據檔（見「依據檔」）；`lct rules lint` 會檢查譯者讀的指引有沒有指向依據檔，並對統計樣式發出警告。

**指引裡寫什麼**

- 每條：粗體標題、規則、例子（KR → 譯文）。一條規則只在一個檔、一個地方寫
- 原因只在它決定「什麼時候適用」時寫一句（例如「遊戲只在劇情台詞處理 `<ruby>`」）
- 「既有譯文不必為此改」這類指示照寫，不附數字
- 例子只寫 KR 與譯文；需要時寫欄位（`place`、`teller`），不寫 key
- 常見錯誤寫錯誤的樣子與正確寫法，不寫次數與發生經過
- 只適用於單一族群的規則寫在該族群的檔，不寫進 `base.md`；`base.md` 只放各族群共通的規則

**指引裡不寫什麼（都進依據檔）**

- 統計：次數、比例、「既有 a／b」、行數、檔數
- 來源：「使用者決定 YYYY-MM-DD」、調整檔或改寫檔名、研究報告路徑、出處 key、「自 base.md 移入」
- 經過：舊譯曾經怎麼錯、哪次匯入怎麼處理
- 工具行為：只在它改變譯者該寫什麼時才寫，否則不寫

**新增慣例的門檻**：「既有 a／b ≥ 80% 且 b ≥ 30」是把慣例寫進指引的條件，數字記在依據檔。a／b 是 raw-20261105 該族群 KR 符合條件、已有譯文紀錄（Paratranz、Claude、人工，不含照抄）的 b 個單位中，a 個照這個寫法。

## 族群指引（`families/<id>.md`）

- 固定四節：文本類型；語氣與人稱；格式慣例（標籤、`[ID]`、數字、標點、換行）；常見錯誤。沒有內容的節不寫，不寫「（目前無足夠證據）」
- 統計數字由 `docs/research/scripts/family_guides.py` 重算（讀 `lct site` 的輸出），重算的數字寫進依據檔；改了寫法時，腳本的樣式一起改

## 共用指引（`shared/<主題>.md`）

- 幾個族群都用、但不是所有族群都用的規則寫在共用指引，只寫一份；用到的族群在 `routing.yaml` 以 `include:` 引入（依位置給的例外以 `guides:` 引入，例如 `ruby.md`）。不必另立四節，只寫用得到的節（節名與族群指引相同；`phrasing.md` 不是格式規則，節名依內容另取）
- 門檻（≥ 80% 且 b ≥ 30）以引入它的族群合計計算，每個引入的族群自己有單位時，比例也不低於 80%（腳本逐族群列出）；數字記在依據檔。規則只在部分族群成立時，不引入其他族群
- 現有：`battle-text.md`（ally、enemy、keywords、mirror：效果文字的括號用半形、`(턴당 N회)`、`최종 위력`、`정신력／체력 N 회복`、`아군`、`처치 시`）；`korean-brackets.md`（ui、events、stage、rpg、ally、enemy、keywords、mirror：方括號裡的韓文照譯）；`skill-text.md`（ally、enemy、keywords、mirror 接在 `battle-text.md` 之後引入：強度／回數／層數的格式、韓文狀態名加空格、技能文字的動詞、消耗數值、斜線與除法、`<noparse>`）；`story-labels.md`（以 `guides:` 給 `StoryData/`、`ScenarioModelCodes`、`StageNode`、`RPGSystem/`、`AbDlg`：地點欄位、稱號與說話者名稱的空格、標籤欄位不留結尾空格）；`ruby.md`（以 `guides:` 給 `StoryData/`、`ScenarioModelCodes`、`PersonalityVoiceDlg/`、`BattleAnnouncerDlg/`、`BattleSpeechBubbleDlg`：劇情台詞的 ruby 與白名單位置的 `<rubycustomize>`；其他檔照 `base.md`「ruby 的位置」）；`phrasing.md`（story-main、story-interlude、identity、ego-voice、announcer、dante-note、abnormality、rpg：台詞與敘述怎麼寫成通順的中文——內容逐項保留、語氣整句對應、句子結構、常見語尾的句型；規則來自使用者對實際句子的取捨，不適用統計門檻）
- 只在兩三個族群成立、不另立共用檔的，仍寫在各族群的檔：`[X] N 부여`（ally、enemy）、`피해량 +N%`（ally、enemy、mirror）。改其中一份時，其他族群的同一條要一起改

## 角色指引（`characters/<id>.md`）

- 開頭的 YAML 是給程式用的對照資料（見「結構」）；內文只寫這個角色的寫法規則。社群原指引不保留原文：仍有效的說法改寫成規則，語體比例、行數等統計寫進依據檔
- 節名依內容取用：**身分**、**語氣與人稱**（說話風格、語體、自稱、第二人稱）、**稱呼**（怎麼稱呼但丁與其他人、別人怎麼叫他）、**用語**（角色特有的譯名與用詞）、**例句**；需要時另立一節（例如良秀的「縮略語」、但丁的「格式」）。沒有內容的節不寫
- **例句**：表格只有 KR 與既有譯文兩欄，出處寫進依據檔。例句的「既有繁中」是現有譯文，用來示範語氣，不代表每句都是定稿
- **待確認**：還沒決定的寫法另立一節；有暫行寫法時寫出決定前譯者照什麼寫（例如「決定前照既有最多的，同票用第一個」，並寫出照這條得出的目前寫法），有提案的寫「提案」；次數與出處寫進依據檔
- 角色指引優先於族群指引與 `base.md`（見 `base.md`「原則」），推翻 base 的角色例外照寫在角色檔（例如辛克萊對他人全部尊稱、多用「您」；唐吉訶德用「汝」）；格式規則與詞彙表不能被角色風格推翻
- 資料和既有說法對不上時，依 KR 原文判斷；決定的來源與日期寫進依據檔，指引只寫決定後的規則

## 依據檔

統計、決定來源、經過與出處放在 `docs/guides-evidence/`，路徑與指引對應（`guides/base.md` → `docs/guides-evidence/base.md`）。格式與維護見 `docs/guides-evidence/README.md`。

- 翻譯時不讀：批次的 `guides` 只列 `guides/` 底下的檔；譯者讀的指引內不寫指向依據檔的連結（只有這份 README 寫它在哪裡）
- 新增或修改規則時，依據寫進對應的依據檔，不寫進指引
