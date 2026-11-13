# StoryData 角色研究報告

2026-11-13。以 StoryData（劇情檔）的 **KR 原文為準**，逐一整理角色的說話方式，並與現行的角色翻譯指引（`guides/characters/`）比較。指引與 KR 不同、指引沒寫到、現行譯文沒照指引的地方都逐條列出，每一點附 KR 例句（`檔名#編號`）。**這份報告不改任何指引或譯文**；每份角色報告的第 6 節是建議，寫成可以直接貼進指引的文字，你同意哪幾條，我再改指引。

## 摘要

- 涵蓋 StoryData 的全部說話者中，台詞 50 行以上的角色：
  - **有指引的 16 位**：但丁、維吉里烏斯、卡戎、12 位罪人、阿賴耶，各一份完整報告。
  - **沒有指引的 65 位**，分 7 組報告；其中 35 位建議新增指引，並附草稿。
  - **另外 6 位有指引、但 StoryData 只有 1～14 行**（菲利普、奧斯卡、薩爾瓦多、尤娜、帕梅拉、帕梅莉），只列出台詞（[minor.md](minor.md)）。
- 有指引的 16 位共提出 **285 點**：與指引不同 13 點、部分一致 64 點、指引沒寫 103 點、現行譯文偏離指引 43 點、model 歸屬 62 點。
- 幾乎每位角色都有同樣的五個問題，見[跨角色的共同問題](#跨角色的共同問題)，建議先決定這五件事。

## 資料與方法

| 項目 | 版本 |
|---|---|
| 程式與規則 | `main` 71168fe |
| 譯文 | `tw` 5f25d25（tw-latest） |
| 原文 | `raw-20261112`（KR、JP、EN） |

1. **整理資料**：StoryData 1,001 個檔、63,035 行台詞。說話者依每行的 `model` 對到角色：有指引的角色用指引的 `models`／`models_other_self`，沒有指引的依顯示名稱歸併。每位角色整理出每一行台詞（含前一行的說話者，大致可看出對誰說），以及粗略統計（句尾語體、自稱、第二人稱的 KR／TW 對照、稱呼）。統計是用正規表示式算的，只當線索；各報告的數字都由子代理重新以腳本實算（例如 `~죠` 原本被誤算為반말）。
2. **逐角色分析**：每位有指引的角色一個子代理，依 [說明](method/CHAR.md) 讀全部台詞，對照現行指引、`guides/base.md`，必要時看 JP／EN 與前後文。沒有指引的角色每組一個子代理。
3. **核對**：報告中引用的每個 `檔名#編號`（共 4,000 多處）都以腳本確認存在於 StoryData；例句文字由子代理從資料檔逐字取出。
4. **網路資料**：可以查 Limbus Company wiki（wiki.gg），不用 namu.wiki、wikiwiki.jp。這個環境中 WebFetch 讀不到 wiki，curl 可以讀 wiki.gg、讀不到 fandom；部分報告引用了 wiki.gg 的頁面，其餘只依 KR 原文。劇情原文與網路資料衝突時以 KR 為準。

限制：分析由模型完成；比例、行數是實算的，「該怎麼譯」的判斷是建議，需要你決定。各報告的「待決定」都列了選項。

## 怎麼讀

每份角色報告（`characters/<id>.md`）的章節固定：

1. **角色概要**
2. **KR 說話方式**：語體、自稱、第二人稱與對各角色的稱呼（表）、口頭禪與用詞、變體 model。例句寫成「`檔名#編號`：KR → 現行 TW」
3. **與現行指引比較**：指引的每一條判定「一致／部分一致／不同／KR 看不出」，另列「KR 有、指引沒寫」
4. **現行 TW 是否照指引**：系統性的偏差與比例
5. **指引沒涵蓋的 model**
6. **建議**：編號條列，每條標明「新增」或「修改第幾條」，附理由與證據；沒把握的寫「待決定」與選項

`data/<id>.json` 是同樣的比較結果（每點的類型、指引原文、KR 所見、證據、建議）。

沒有指引的角色在 `others/group-N.md`，每位一節：概要、說話方式、現行譯文是否前後一致、是否建議新增指引，建議的話附指引草稿。

## 怎麼同意

告訴我角色與建議編號，例如「浮士德 1–4 同意，5 選保留」「辛克萊全部同意」「新增霍恩海姆、賈惜春的指引」。同意後：

- 指引規則：照報告第 6 節的文字改 `guides/characters/<id>.md`，經由 `main` 提交
- model 歸屬：改指引的 frontmatter（`models`、`models_other_self`、`models_not_this`）
- 新指引：照草稿新增 `guides/characters/<id>.md`，並確認 `models` 沒有和既有指引重複
- 現行譯文的偏差：指引改好後，另外列出要改的單位清單給你同意，再以人工調整或重翻處理（這份報告只指出偏差與比例，沒有逐句列出全部要改的單位）

## 總覽

### 有指引的角色

| 角色 | 行數 | 不同 | 部分一致 | 指引未寫 | 譯文偏差 | model 歸屬 | 合計 | 報告 |
|---|---:|---:|---:|---:|---:|---:|---:|---|
| 但丁（`dante`） | 3438 | 1 | 3 | 10 | 1 | 2 | 17 | [characters/dante.md](characters/dante.md) |
| 維吉里烏斯（`vergilius`） | 954 | 0 | 5 | 6 | 5 | 5 | 21 | [characters/vergilius.md](characters/vergilius.md) |
| 卡戎（`charon`） | 163 | 0 | 3 | 5 | 2 | 2 | 12 | [characters/charon.md](characters/charon.md) |
| 李箱（`yisang`） | 1619 | 1 | 5 | 5 | 3 | 4 | 18 | [characters/yisang.md](characters/yisang.md) |
| 浮士德（`faust`） | 2690 | 1 | 5 | 5 | 3 | 5 | 19 | [characters/faust.md](characters/faust.md) |
| 唐吉訶德（`donquixote`） | 2299 | 0 | 2 | 7 | 3 | 7 | 19 | [characters/donquixote.md](characters/donquixote.md) |
| 良秀（`ryoshu`） | 2051 | 1 | 3 | 7 | 1 | 3 | 15 | [characters/ryoshu.md](characters/ryoshu.md) |
| 莫梭（`meursault`） | 1323 | 1 | 4 | 7 | 3 | 3 | 18 | [characters/meursault.md](characters/meursault.md) |
| 鴻璐（`honglu`） | 2226 | 0 | 4 | 8 | 4 | 3 | 19 | [characters/honglu.md](characters/honglu.md) |
| 希斯克利夫（`heathcliff`） | 2550 | 1 | 7 | 8 | 1 | 7 | 24 | [characters/heathcliff.md](characters/heathcliff.md) |
| 以實瑪利（`ishmael`） | 2578 | 3 | 5 | 3 | 2 | 2 | 15 | [characters/ishmael.md](characters/ishmael.md) |
| 羅佳（`rodion`） | 2062 | 1 | 4 | 9 | 2 | 3 | 19 | [characters/rodion.md](characters/rodion.md) |
| 辛克萊（`sinclair`） | 2047 | 1 | 3 | 8 | 3 | 6 | 21 | [characters/sinclair.md](characters/sinclair.md) |
| 奧提斯（`outis`） | 1971 | 0 | 4 | 8 | 2 | 7 | 21 | [characters/outis.md](characters/outis.md) |
| 格里高爾（`gregor`） | 1745 | 1 | 3 | 5 | 3 | 2 | 14 | [characters/gregor.md](characters/gregor.md) |
| 阿賴耶（`araya`） | 213 | 1 | 4 | 2 | 5 | 1 | 13 | [characters/araya.md](characters/araya.md) |

欄位說明：「不同」是指引寫的與 KR 相反或不符；「部分一致」是方向對但不完整或有例外；「指引未寫」是 KR 有明顯特徵、指引沒提；「譯文偏差」是現行 TW 沒照指引；「model 歸屬」是指引的 `models` 漏列、歸錯或需要另立指引的 model。

### 沒有指引的角色

| 角色 | 行數 | 建議新增指引 | 報告 |
|---|---:|:---:|---|
| 索尼亞（`sonya`） | 110 | 是 | [others/group-1.md](others/group-1.md) |
| 冬柏（過去）（`dongbaek-past`） | 72 |  | [others/group-1.md](others/group-1.md) |
| 箱李（`sangi`） | 51 |  | [others/group-1.md](others/group-1.md) |
| 德米安（`demian`） | 68 |  | [others/group-1.md](others/group-1.md) |
| 亞哈（`ahab`） | 216 | 是 | [others/group-1.md](others/group-1.md) |
| 魁魁格（`queequeg`） | 84 | 是 | [others/group-1.md](others/group-1.md) |
| 巴里（`bari`） | 80 |  | [others/group-1.md](others/group-1.md) |
| 桑丘（`sancho`） | 402 | 是 | [others/group-1.md](others/group-1.md) |
| 王照（`wang-zhao`） | 85 |  | [others/group-1.md](others/group-1.md) |
| 主持人（家主審查）（`mc-grandview`） | 84 |  | [others/group-1.md](others/group-1.md) |
| 林黛玉（`lin-daiyu`） | 186 | 是 | [others/group-1.md](others/group-1.md) |
| 霍恩海姆（호엔하임）（`hohenheim`） | 697 | 是 | [others/group-2.md](others/group-2.md) |
| 賈母（가모）（`jiamu`） | 221 | 是 | [others/group-2.md](others/group-2.md) |
| T公司代表／助手（T사대표，哈伯特）（`t-corp-rep`） | 130 | 是 | [others/group-2.md](others/group-2.md) |
| 賈環（가환／에고가환）（`jia-huan`） | 123 |  | [others/group-2.md](others/group-2.md) |
| 尤莉（유리）（`yuri`） | 152 |  | [others/group-2.md](others/group-2.md) |
| 輕逃（경도）（`kyungdo`） | 76 |  | [others/group-2.md](others/group-2.md) |
| 時間殺人魔（시간살인마）（`time-killer`） | 58 |  | [others/group-2.md](others/group-2.md) |
| 史易華（`si-yihua`） | 56 |  | [others/group-3.md](others/group-3.md) |
| 子路（`zilu`） | 85 |  | [others/group-3.md](others/group-3.md) |
| 子貢（`zigong`） | 161 | 是 | [others/group-3.md](others/group-3.md) |
| 賈元春（`jia-yuanchun`） | 145 | 是 | [others/group-3.md](others/group-3.md) |
| 亞倫（`allen`） | 65 |  | [others/group-3.md](others/group-3.md) |
| 綺羅（`kira`） | 275 | 是 | [others/group-3.md](others/group-3.md) |
| 櫻桃（`aengdu`） | 256 | 是 | [others/group-3.md](others/group-3.md) |
| 以斯拉（`ezra`） | 219 | 是 | [others/group-3.md](others/group-3.md) |
| 金笠（`kimsatgat`） | 238 | 是 | [others/group-3.md](others/group-3.md) |
| 賈惜春（`gasichun`） | 663 | 是 | [others/group-4.md](others/group-4.md) |
| 衛（`wei`） | 124 |  | [others/group-4.md](others/group-4.md) |
| 醉掌之夜客（`chwisu`） | 129 | 是 | [others/group-4.md](others/group-4.md) |
| 薛蟠（`xue-pan`） | 67 |  | [others/group-4.md](others/group-4.md) |
| 阿米婭（`amiya`） | 185 |  | [others/group-4.md](others/group-4.md) |
| 艾麗妮（`irene`） | 118 |  | [others/group-4.md](others/group-4.md) |
| 凱爾希（`kaltsit`） | 73 |  | [others/group-4.md](others/group-4.md) |
| 邁爾斯（`meyers`） | 217 | 是 | [others/group-5.md](others/group-5.md) |
| 博士（`doctor`） | 64 |  | [others/group-5.md](others/group-5.md) |
| 克羅默的姊姊（`kromer-sister`） | 62 |  | [others/group-5.md](others/group-5.md) |
| 簡（`jeanne`） | 232 | 是 | [others/group-5.md](others/group-5.md) |
| Carmen（`carmen`） | 256 | 是 | [others/group-5.md](others/group-5.md) |
| 赫爾曼（`herman`） | 88 |  | [others/group-5.md](others/group-5.md) |
| 埃菲（`effie`） | 50 |  | [others/group-5.md](others/group-5.md) |
| 索德（`sord`） | 218 | 是 | [others/group-5.md](others/group-5.md) |
| 三朝（`samjo`） | 142 | 是 | [others/group-5.md](others/group-5.md) |
| 東朗（`donglang`） | 408 | 是 | [others/group-6.md](others/group-6.md) |
| 皮利托（`pilot`） | 84 | 是 | [others/group-6.md](others/group-6.md) |
| 辛德利（`hindley`） | 104 | 是 | [others/group-6.md](others/group-6.md) |
| 奈莉（`nelly`） | 223 | 是 | [others/group-6.md](others/group-6.md) |
| 亞細亞（`asia`） | 63 |  | [others/group-6.md](others/group-6.md) |
| 死兔幫老大（`deadrabbits-boss`） | 56 |  | [others/group-6.md](others/group-6.md) |
| 林頓（`linton`） | 124 | 是 | [others/group-6.md](others/group-6.md) |
| 凱瑟琳（`catherine`） | 166 | 是 | [others/group-6.md](others/group-6.md) |
| 切薩拉（`cesara`） | 72 |  | [others/group-6.md](others/group-6.md) |
| 參孫（`samson`） | 216 | 是 | [others/group-6.md](others/group-6.md) |
| 薛寶釵（`xue-baochai`） | 168 | 是 | [others/group-7.md](others/group-7.md) |
| 賈丘（孔丘）（`gachiu`） | 266 | 是 | [others/group-7.md](others/group-7.md) |
| 雷橫（`noehoeng`） | 106 | 是 | [others/group-7.md](others/group-7.md) |
| 阿莉莎（`alisa`） | 99 |  | [others/group-7.md](others/group-7.md) |
| 馬頓（`marten`） | 61 |  | [others/group-7.md](others/group-7.md) |
| 摩西（`moses`） | 117 |  | [others/group-7.md](others/group-7.md) |
| 韋斯帕（`vespa`） | 56 |  | [others/group-7.md](others/group-7.md) |
| 馬蒂亞斯（`middle-matthias`） | 76 |  | [others/group-7.md](others/group-7.md) |
| 地慧星（`pinky-jihyeseong`） | 228 | 是 | [others/group-7.md](others/group-7.md) |
| 瓦倫奇娜（`thumb-valencina`） | 229 | 是 | [others/group-7.md](others/group-7.md) |
| 卡利斯托（`ring-callisto`） | 136 | 是 | [others/group-7.md](others/group-7.md) |
| 禮恩（`index-yuen`） | 285 | 是 | [others/group-7.md](others/group-7.md) |

## 跨角色的共同問題

### 1. 對女性的第二人稱「妳」

`guides/base.md` 規定對女性用「妳」，但現行譯文大量寫成「你」，幾乎每位角色都有，而且多是社群舊譯（本專案新譯大多已寫「妳」）。各報告實算的例子：

| 角色 | 情形 |
|---|---|
| 維吉里烏斯 | 對女性罪人與卡戎說話的 65 行中 51 行寫「你」 |
| 希斯克利夫 | 第六章對女性的 36 行中 35 行寫「你」 |
| 以實瑪利 | 對女性約 75% 寫「你」（對亞哈 23 行全是「你」、魁魁格 12 行中 11 行） |
| 良秀 | 對女性罪人 15 行中 11 行 |
| 李箱 | 對女性約 59% 寫「你」 |
| 奧提斯 | 主線對女性罪人 32 行中 16 行 |
| 羅佳 | 可確定對象是女性的 28 行中 15 行（全是社群舊譯；本專案新譯的 6 行都是「妳」） |
| 鴻璐 | 對惜春 26 行中 12 行 |
| 其他 | 但丁、浮士德、格里高爾、辛克萊、唐吉訶德、阿賴耶的報告也列了這一項；配角中東朗、林頓、辛德利、賈元春等也有 |

這不是指引的問題，是舊譯沒照規則。**建議**：另做一次專門的修正——以腳本找出「說話對象是女性」的台詞（依前後行的說話者與 KR 稱呼），由模型逐句判斷，產生要改的清單給你同意後，以人工調整寫入。需要你決定要不要做、範圍（全部 StoryData，或先做主線）。

### 2. 對但丁用「您」還是「你」

各指引的規定不同，現行譯文也不一致：

- 浮士德：對但丁的 87 行中 26 行寫「你」，指引規定「您」
- 鴻璐：22 行中 5 行寫「你」
- 以實瑪利：指引寫「全部使用您」，但 KR 對但丁的 70 行中 34 行沒有敬語標記（直呼「단테」、用「당신」），集中在第五章以後
- 李箱：沒有敬語標記卻寫「您」的 11 行中有 9 行；指引「有 -시- 就用您」照字面套用會和 KR 語感衝突
- 奧提斯：大致一致（「您」46、「你」2）

**需要你決定**：對但丁的「您」要（A）每個角色固定（照指引），還是（B）依 KR 當句有沒有敬語標記決定。各報告的建議多偏向 A 為主、B 為例外，細節在各報告第 6 節。

### 3. 指引的「資料補充」數字不準

多份指引的「資料補充」有 KR 語體比例等統計（例如李箱「반말 40%」），是以前用正規表示式算的，把 `~죠`、`~군`、`~니까` 等誤判，數字不對：李箱實際是하오체 72%、하게체 17%、真正的반말不到 1%；以實瑪利重算後是해요체 51%、반말 44%。但丁、浮士德、希斯克利夫、莫梭、辛克萊、羅佳、維吉里烏斯的報告也指出了同樣的問題。**建議**：照各報告重算的數字改寫，或乾脆刪掉比例只留描述。

### 4. model 歸屬

指引的 `models` 漏列或歸錯的 model（細節見各報告第 5 節）：

- 希斯克利夫：`후디히스`、`뿅망히스`、`곰스클리프` 是本人，建議加入 `models`
- 辛克萊：`퀴즈싱클` 加入 `models`；`기사싱클`、`기사싱클2`（劇中劇的白月騎士）放 `models_other_self`
- 格里高爾：`말레고르` 加入 `models`
- 唐吉訶德：`기사돈키` 是本人，加入 `models`；`키호테`、`가면키호테`、`키호테빛` 是她的父親「拉·曼查領的唐吉訶德」，放 `models_not_this` 並建議另立指引；桑丘（`산초`、`옛산초`、`옛돈키`）就是罪人本人（第 1 組配角報告同樣判斷），`돈키호테산초／2` 可移進 `models`，指引的「待確認」可以結案
- 維吉里烏斯：`산손베르길리우스` 的說話者是參孫，移到 `models_not_this`；`베르길파우` 漏列（浮士德報告也提到）
- 其餘角色的人格故事、愚人節、受傷、回憶等變體，各報告第 5 節都有判斷

### 5. E001X 的換身體劇情

E001X 中罪人的意識互換，model 寫的是身體的主人，說話的卻是另一個人（例如 `#102` 起 model `로쟈` 說話的是莫梭）。卡戎、但丁、唐吉訶德、浮士德、希斯克利夫、莫梭、奧提斯、羅佳、辛克萊、維吉里烏斯、李箱的報告都碰到這一段。**建議**：在 E001X 的單檔備註寫一張對照表（從第幾行起哪個 model 是誰的意識），讓翻譯時套用正確的角色指引，而不是在每份角色指引各寫一次。

### 其他值得注意的

- **但丁**：任務說明原本寫「維吉里烏斯聽得懂但丁的時鐘聲」，KR 顯示他聽不懂，要靠罪人轉達（`S002B#155`、`S708B#16`），報告已更正
- **「經理」與「執行經理」**：KR `관리자님` 詞彙表寫「經理」，現行譯文有不少「執行經理」（奧提斯 210 行中 39 行、莫梭也有），各報告建議統一
- **新指引候選**：沒有指引的配角中，台詞多且特色明顯的有霍恩海姆（697 行）、賈惜春（663 行）、桑丘（應併入唐吉訶德）、東朗、禮恩等，見上表
- **《明日方舟》合作角色**（阿米婭、凱爾希、艾麗妮等）：原作繁中官方譯法這次查不到，報告標「待人工對照」，並建議把羅德島、礦石病、恐魚、海嗣等專有名詞補進詞彙表

## 檔案

- `characters/<id>.md`：16 位有指引角色的報告；`data/<id>.json`：比較結果
- `others/group-1.md` … `group-7.md`：沒有指引的角色；`data/others-group-N.json`
- `minor.md`：StoryData 台詞很少的 6 位
- `method/CHAR.md`：給子代理的說明；`method/prep.py`、`method/charprep.py`：整理資料的腳本
