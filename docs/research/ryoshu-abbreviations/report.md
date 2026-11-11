# 良秀（료슈）縮略語研究報告

- 分支 `claude/festive-wright-g2k853`。初版提交 `3922a36`；審查第 1 輪的修正提交 `afb4bf6`、`9370303`、`bd38a85`、`bc17f22`（見文末「修正第 1 輪」）。本文的數字已依審查重算。第 2 輪（第 3 節加上 JP／EN 的所有寫法與「類別」、重新核對「刻意不同」）的改動見文末「修正第 2 輪」；第 3 輪（第 3 節改為每行附 KR／JP／EN／繁中的完整說法與來源、依類別分 4 張表）見文末「修正第 3 輪」
- 資料：raw-20261105（`b923efc`）KR／JP／EN；繁中用 tw `008ead4`（build raw-20261105）。以 `lct site` 輸出的對齊資料（KR/JP/EN/TW 同一單位）統計
- 單位寫法：`檔名｜#索引`，`#` 是 `dataList` 中從 0 起算的位置（不是 `id` 欄），RPG 的 `#16/9` 也是位置（StoryData 省略目錄；`RPG:floor-b2｜#16/9` = `RPGSystem/rpg-loc-dialogue-floor-b2.json|/dataList/16/texts/9/text`；`Voice_X` = `PersonalityVoiceDlg/Voice_Ryoshu_X`；`Announcer_Ryoshu` = `BattleAnnouncerDlg/Announcer_Ryoshu_42`）
- 統計腳本在本資料夾的 `scripts/`（用法見 `scripts/README.md`）

## 1. 摘要

| 項目 | 結果 |
|---|---|
| 良秀 KR 台詞 | 2,596 行（本人 model、`file_names: [Ryoshu]` 語音、RPG `speaker`、`Announcer_Ryoshu_42` 共 2,305 行；過去的本人 `과거료슈1–3` 291 行） |
| 有縮略語 | 356 行、389 處；依音節分約 286 種（分開加點／不加點為 291 條）。第 3 節的表把良秀自己不加點的併入加點的、接 이다 的 `다.` 併入本體、修正 3 處邊界後為 284 種，出現 ≥ 2 次的 50 種、155 處（初版依 291 條算 49 種、147 處） |
| 類別（第 3 節） | 一致 258 種（出現 ≥ 2 次的 24 種）、同義異寫 19、異義 5（`어.이.` 未確認）、只有中文不一 2；中文不一共 16 種。出現 ≥ 2 次的 50 種中 JP 換寫法的 17 種、EN 換縮寫的 15 種，大多是同義改寫 |
| 完整說法（第 3 節） | 319 行（KR 相同而 JP／EN／繁中不同的各占一行）。KR 的完整說法有遊戲內展開的 144 行、推測 159、未縮 16；EN 遊戲內 157、不明 8。「遊戲內」逐字核對過出處 |
| 加點 | 379／389 處（不加點 10：`완소`、`감다뒤`、`안심안지`、`유잼`（借阿賴耶的流行語）、`모분조도.`、`내머.`、`검만예극`、`별, 없.`、`로드`、`엔러쉬`） |
| 劇中有人展開 | 88 處、78 種（辛克萊 29、良秀自己 26、莫梭 7、旁白 7、李箱 3…） |
| 繁中用「．」 | 353／356 行；沒有用 `・`、`·`、半形 `.` 的；末字後多半不加「．」（例外 8 處，見 5.1） |
| 阿賴耶 | model `료슈딸`（已加入 araya.md）、`료슈딸2`、`료슈딸연구실`、`아라야공백`、`아라야10대`、`아라야20대`、`아라야정장`、`아라야정장2`，213 行；流行語縮寫 10 處全部不加點 |
| 主要差異 | 良秀：自己造的、逐音節加「.」（379／389）；阿賴耶：學校學來的現成流行語，不加點、當一個詞用，還糾正良秀「그렇게 줄이는 거 아니라고」 |
| 提案 | 37 條（第 2 輪新增 30–37、修改 5），其中第 26 條在第 1 輪撤回，有效 36 條。2026-11-11 採用 20 條（寫成人工調整），其餘 16 條待決定（見第 9 節） |

## 2. 方法

1. 良秀的行：story `model` 在 ryoshu.md `models`（含 `료슈9장*`）或 `models_other_self`；`AbDlg_Ryoshu`、`PersonalityVoiceDlg/Voice_Ryoshu_*`、`EGOVoiceDig/Voice_EGO_Ryoshu_*`；RPG `speaker` = 료슈；另把 `BattleAnnouncerDlg/Announcer_Ryoshu_42`（良秀的播報員語音，34 處縮略語）算進來
2. 候選（審查補上漏掉的 3 處：`E030X｜#94` `알. 못아.`、`E908B｜#45` `흥. 이.`、`Voice_Butler_10408｜#21` `모. 분이다`，已併入下表）：KR 中「單一音節＋`.`」連續兩個以上（容許空格、末字無點、後接助詞）。再過濾：同一行 JP（`・`）、EN（字母縮寫）、繁中（`．`）都沒有縮略標記的候選全部是誤判（99 個，如 `하. 그`、`쯧. 젠`），刪去；兩音節且首字是感嘆詞或句末單字（`뭐. 없`、`하. 시`）的也刪去；再人工修正 16 個邊界（`어. 나.다.`→`어. 나.`、`불. 시. 낭이다`→`불. 시. 낭` 等），刪去 `소. 닭. 용.`（S813B#105，是生肖名，不是縮略語）
3. 不加點的：用 JP/EN/繁中有縮略標記但 KR 沒有點的行找出（P10402#12、S101B#45、S201B#78、S111B#28），加上流行語字串搜尋（`완소`、`감다뒤`、`유잼` 等）
4. 展開：同一行縮略語之後、或同檔前 3 行到後 6 行，找「每個語節首音節依序等於縮略語」的片段；另人工看過所有「~라는 뜻이다」「~라는 의미」行（約 25 行）
5. JP／EN／繁中的對應片段（第 2 輪）：每行的 JP「・」片段、EN 大寫縮寫、繁中「．」片段依位置與 KR 的縮略語配對（`scripts/tokens.py`）；同一行有兩個以上縮略語、片段數對不上或抓到前後文的，讀過整行後人工填（145 處，`scripts/tokens_ov.py`）。再依每個縮略語的 JP、EN、繁中有幾種寫法分類，有變化的都讀過場景判定意思是否相同（`scripts/table2.py`）

## 3. 良秀縮略語完整表

第 3 輪改版（使用者要求，2026-11-09）：每行列出 KR、JP、EN、繁中（TW）的縮略與**完整說法**，並標出完整說法的來源；依類別分成 4 張表；KR 相同而 JP、EN、繁中的寫法不同時，不合併，每種組合各占一行。第 2 輪的表（JP、EN、繁中各列所有寫法與次數）由這一版取代。

- **收錄範圍**：與第 2 輪相同，284 種、389 處。良秀自己不加點的 `모분조도.`、`내머.`、`검만예극` 併入加點的；接 이다 的 `다.`（`거.거.다.`、`그. 그.다.`、`만.단.지.예.다.`）併入本體；借用阿賴耶的流行語 `완소`、`유잼`、`감다뒤`、`안심안지` 也列入。拆成 319 行
- **分行**：同一種 KR 縮略語，JP、EN、繁中三者的寫法組合不同就分成不同行。JP 只差「・」（KR 不加點的那處）、EN 只差點、空格或末點（`SYNC`＝`S.Y.N.C.`）的不算不同，寫在縮略欄的「另」
- **縮略欄**：該處的片段。「（不加點）」＝片段本身沒有點或「・」；「（未縮）」「（無；譯「…」）」＝該語言沒有縮；「（用逗號）」＝用逗號代替點；繁中「（．的）」＝末字後還有一個「．」，接著語尾或名詞（見 5.1）。KR 欄的括號註明另屬的類別、未確認／待確認、第 9 節的提案編號
- **完整欄**：完整說法與來源
  - **遊戲內**：遊戲文字裡**那個語言自己**寫出了完整說法——劇中有人展開、括號註明、更正、旁白解釋，或同一行就寫出（EN 常見 `C.H. (Clock Head)`）；附說話者與位置。JP、EN、繁中看的是各自的譯文：KR 的展開句被譯出，而且譯文的詞對得上該語言縮略的每個字，才算該語言的遊戲內。同一場景較早已完整說過、縮略語明顯就是在縮它的，也算。**遊戲內・他處**＝展開在別的檔案（同一個縮略語在別處被展開，意思相同）
  - **推測**：遊戲裡沒有展開，從那個語言的那一行與上下文推出，逐一對上縮略的每個字；不確定的部分加「？」。KR 的推測以 KR 上下文為主，只靠 JP、EN 的在附註寫明
  - **未縮**：該語言沒有縮，或加點的各段合起來就是完整的詞（`젠. 장.`＝젠장、`無．聊`＝無聊）
  - **不明**：推不出
- **位置**：`檔案路徑#位置`，`#` 是 `dataList` 中從 0 起算的位置；RPG 是 `#項/句`。出現多處的全部列出。附註寫在位置之後
- **類別**（判定見 3.5；與第 2 輪相同）：
  - **一致**：JP、EN、繁中各自只有一種寫法
  - **同義異寫**：JP 或 EN 換了寫法，但每處意思相同。繁中應統一
  - **異義**：同一組 KR 在不同場景是不同意思（劇中展開不同，或 JP、EN 一起改成另一個意思）。繁中可以不同
  - **中文不一**：意思相同而繁中寫法不同
  - 一種縮略語只放在一張表：有中文不一的放「中文不一」（KR 欄註明另屬同義異寫或異義），其次異義、同義異寫，其餘是一致
- **核對**：「遊戲內」的完整說法都由腳本逐字核對過，確實出現在所附位置那個單位的該語言文字裡（`scripts/check_full.py`）；完整說法由人工判讀後再抽查（抽查 103 行：非一致的 61 行全部、有「遊戲內・他處」的 22 行、一致中隨機 40 行；更正 15 處，最多的是漏看同場景已有的展開（6 處），另把 2 處意思對不上的「他處」改回推測。之後再用腳本搜尋其餘的「推測」：上下文裡逐字出現完整說法的，又找到 1 處（`괜. 안. 괜.` 的 EN））
- **統計**：319 行：一致 258 種 258 行、同義異寫 6 種 12 行、異義 4 種 9 行、中文不一 16 種 40 行。完整說法的來源（每行一筆）：KR 遊戲內 144／推測 159／未縮 16；JP 遊戲內 130／推測 168／未縮 21；EN 遊戲內 157／推測 145／未縮 9／不明 8；繁中 遊戲內 130／推測 143／未縮 46。遊戲內之中，展開在別的檔案（遊戲內・他處）的 46 處

### 3.1 一致（258 種、258 行）

| KR 縮略 | KR 完整 | JP 縮略 | JP 完整 | EN 縮略 | EN 完整 | TW 縮略 | TW 完整 | 位置 |
|---|---|---|---|---|---|---|---|---|
| 만.단.지.예（另 만. 단. 지. 예.다.） | 만물？을 단축하는 지고？의 예술（推測） | 万・短・至・芸 | 万物？を短縮する至高？の芸術（推測） | S.A.N.G.R.I.A.（另 SANGRIA） | 不明 | 萬．短．至．藝 | 萬物？縮短的至高？藝術（推測） | 5 處：`BattleAnnouncerDlg/Announcer_Ryoshu_42.json#23`、`StoryData/S303B.json#54`、`StoryData/E914B.json#86`、`StoryData/E912B.json#163`、`StoryData/S430B.json#66`（附註：劇中未展開；E912B#155–163 前文講 단축（短縮）與 지고지순（至高至純）；EN S.A.N.G.R.I.A. 只被稱為 acronym（PinkyChibi_41#9）） |
| 거.대. | 거미대가리（遊戲內：료슈，`PersonalityVoiceDlg/Voice_Ryoshu_Spider_10410.json#11`） | く・ヅ | 蜘蛛ヅラ（遊戲內：료슈，`PersonalityVoiceDlg/Voice_Ryoshu_Spider_10410.json#11`） | S.H. | Spiderhead（遊戲內：료슈，`PersonalityVoiceDlg/Voice_Ryoshu_Spider_10410.json#11`） | 蜘．頭 | 蜘蛛頭（遊戲內：료슈，`PersonalityVoiceDlg/Voice_Ryoshu_Spider_10410.json#11`） | 4 處：`PersonalityVoiceDlg/Voice_Ryoshu_Spider_10410.json#13`、`PersonalityVoiceDlg/Voice_Ryoshu_Spider_10410.json#16`、`PersonalityVoiceDlg/Voice_Ryoshu_Spider_10410.json#24`、`StoryData/P10410.json#50`（附註：前一句（Spider_10410#11）說出全名） |
| 해.바. | 해골바가지（遊戲內：료슈，`PersonalityVoiceDlg/Voice_Ryoshu_Spider_10410.json#11`） | しゃ・こ | 髑髏(しゃれこうべ)（遊戲內：료슈，`PersonalityVoiceDlg/Voice_Ryoshu_Spider_10410.json#11`） | S.F. | Skullface（遊戲內：료슈，`PersonalityVoiceDlg/Voice_Ryoshu_Spider_10410.json#11`） | 骷．頭 | 骷髏頭（遊戲內：료슈，`PersonalityVoiceDlg/Voice_Ryoshu_Spider_10410.json#11`） | 4 處：`PersonalityVoiceDlg/Voice_Ryoshu_Spider_10410.json#12`、`PersonalityVoiceDlg/Voice_Ryoshu_Spider_10410.json#19`、`PersonalityVoiceDlg/Voice_Ryoshu_Spider_10410.json#24`、`StoryData/P10410.json#91`（附註：前一句（Spider_10410#11）說出全名） |
| 검.만.예.극（另 검만예극） | 검으로 만들어내는 예술의 극치（遊戲內：旁白，`StoryData/P10402.json#13`） | 剣・創・芸・極（另 剣創芸極（不加點）） | 剣で創り出す芸術の極致（遊戲內：旁白，`StoryData/P10402.json#13`） | PACTS | Peak art created through swordsmanship（遊戲內：旁白，`StoryData/P10402.json#13`） | 劍．創．藝．極 | 用劍創作出的藝術所能達到的極致（遊戲內：旁白，`StoryData/P10402.json#13`） | 3 處：`PersonalityVoiceDlg/Voice_Ryoshu_Seven6_10402.json#17`、`PersonalityVoiceDlg/Voice_Ryoshu_Seven6_10402.json#22`、`StoryData/P10402.json#12`（附註：旁白緊接著解釋（P10402#13）） |
| 성. 업. | 성능 업그레이드（遊戲內：싱클레어，`StoryData/E064X.json#15`） | 性・ア | 性能アップグレード（遊戲內：싱클레어，`StoryData/E064X.json#15`） | F.U. | Functionality Upgrade（遊戲內：싱클레어，`StoryData/E064X.json#15`） | 能．升 | 能力提升（遊戲內：싱클레어，`StoryData/E064X.json#15`） | 3 處：`StoryData/E724B.json#78`、`StoryData/E064X.json#14`、`StoryData/E803B.json#29` |
| 알.우. | 알리우네（推測） | アル・ネ | アルリウネ（推測） | A.U. | Alriune（推測） | 愛．溫 | 愛娜溫（推測） | 3 處：`PersonalityVoiceDlg/Voice_Ryoshu_LEGOEcho_10414.json#4`、`PersonalityVoiceDlg/Voice_Ryoshu_LEGOEcho_10414.json#8`、`PersonalityVoiceDlg/Voice_Ryoshu_LEGOEcho_10414.json#19`（附註：幻想體名；P10414 稱 알리우네／アルリウネ／Alriune／愛娜溫） |
| 히.클. | 히스클리프（推測） | ヒ・ク | ヒースクリフ（推測） | H.C. | Heathcliff（推測） | 希．克 | 希斯克利夫（推測） | 3 處：`BattleAnnouncerDlg/Announcer_Ryoshu_42.json#1`、`StoryData/E913B.json#65`、`StoryData/E903B.json#167`（附註：E913B#65 對希斯克利夫說；E903B#167「히. 클.의 저택」；EN 取 Heath／cliff） |
| 그.태.말.늦. | 그딴 태평한 말을 하기에는 늦었다（遊戲內：싱클레어，`StoryData/E302B.json#145`） | そ・の・い・お | そんなのんきなことを言うには遅すぎる（遊戲內：싱클레어，`StoryData/E302B.json#145`） | C.H.I.C.K.（另 CHICK） | Chirping hesitation isn’t cool, kiddo（遊戲內：싱클레어，`StoryData/E302B.json#145`） | 現．輕．話．晚 | 現在說這些輕鬆話已經晚了（遊戲內：싱클레어，`StoryData/E302B.json#145`） | 2 處：`BattleAnnouncerDlg/Announcer_Ryoshu_42.json#25`、`StoryData/E302B.json#144`（附註：辛克萊猜出展開（E302B#145）） |
| 나.시. | 나갈 시간이다（遊戲內：旁白，`RPGSystem/rpg-loc-dialogue-common-a1c10p1.json#18/2`） | 出・時 | 出る時間（遊戲內：旁白，`RPGSystem/rpg-loc-dialogue-common-a1c10p1.json#18/2`） | G.G., C.H. | Get Going, Clockhead（遊戲內：旁白，`RPGSystem/rpg-loc-dialogue-common-a1c10p1.json#18/2`） | 出．時 | 出去的時間（遊戲內：旁白，`RPGSystem/rpg-loc-dialogue-common-a1c10p1.json#18/2`） | 2 處：`RPGSystem/rpg-loc-dialogue-common-a1c10p1.json#18/0`、`RPGSystem/rpg-loc-dialogue-common-a1c10p1.json#30/0`（附註：劇中給兩解：나갈 시간／나가라 시계（出る時間／出ろ時計；Gotta Go, Chore Hour／Get Going, Clockhead；出去的時間／出去吧時鐘），#30/1 同） |
| 단.테. | 단체？ 테러（推測） | 団・テ | 団体テロ（推測） | D.O.N. and T.E.（另 D.O.N.T.E.） | Deconstruction Of Normalcy, Terrifying Everyone？（推測） | 襲．眾 | 襲擊群眾？（推測） | 2 處：`PersonalityVoiceDlg/Voice_Ryoshu_Yurodivy_10409.json#4`、`PersonalityVoiceDlg/Voice_Ryoshu_Yurodivy_10409.json#23`（附註：諧音 단테；聖愚（유로지비）的活動；EN #23 有 deconstruction of normalcy、no one's terrified，E 推為 Everyone） |
| 대. 박. | 대가리가 박살（遊戲內：旁白，`StoryData/S504B.json#111`） | あ・われ | 哀れ（遊戲內：旁白，`StoryData/S504B.json#111`） | O.O.F. | obliteration of forehead（遊戲內：旁白，`StoryData/S504B.json#111`） | 頭．裂 | 頭顱被打得裂開（遊戲內：旁白，`StoryData/S504B.json#111`） | 2 處：`StoryData/S504B.json#110`、`StoryData/S513B.json#15`（附註：JP あ・われ＝哀れ（旁白「哀れにも」）） |
| 머. 절. | 머리를 절단해（遊戲內：료슈2，`StoryData/E902B.json#71`） | 頭・切 | 頭を切り落として（遊戲內：료슈2，`StoryData/E902B.json#71`） | S.Y.N.C. | snap your neck completely（遊戲內：료슈2，`StoryData/E902B.json#71`） | 分．首 | 身首分家（遊戲內：료슈2，`StoryData/E902B.json#71`） | 2 處：`StoryData/E902B.json#69`、`StoryData/E902B.json#72`（附註：TW 展開「身首分家」字序為首…分） |
| 모. 끝. | 모두 끝（推測） | 臨・終 | 臨終（未縮） | F.U.B.A.R. | Fucked Up Beyond All Recognition（推測） | 臨．終 | 臨終（未縮） | 2 處：`StoryData/S532B.json#21`、`StoryData/S532B.json#56`（附註：EN FUBAR 為慣用縮寫，R 亦作 Repair） |
| 모.분.조.도.（另 모분조도.） | 모가지를 분질러야 조용해지지, 도야지（遊戲內：료슈，`StoryData/S101B.json#46`） | く・へ・し・ぶ（另 くへしぶ（不加點）） | 首を圧し切らないと静かにならないのか、ブタ（遊戲內：료슈，`StoryData/S101B.json#46`） | T.R.A.N.C.H.E.S.（另 TRANCHES） | To Right Against Noisy Crying Hogs Entails Scragging（遊戲內：료슈，`StoryData/S101B.json#46`） | 脖．扭．豬．閉 | 不把脖子扭斷，他們這群豬狗一樣的傢伙就不會閉嘴（遊戲內：료슈，`StoryData/S101B.json#46`） | 2 處：`BattleAnnouncerDlg/Announcer_Ryoshu_42.json#28`、`StoryData/S101B.json#45`（附註：良秀自己展開（S101B#46；Announcer#28 同句）） |
| 벌. 양 | 벌레 양반（推測） | む・だ | 虫の旦那（推測） | B.G.（另 BG） | Bug Guy（推測） | 蟲．男 | 蟲男（未縮） | 2 處：`StoryData/S611B.json#56`、`StoryData/3D309A.json#97`（附註：그레고르 的稱號（IntroductionPreset.json#3：벌레 양반／虫の旦那／Bug Guy／蟲男）） |
| 사.분. | 사지를 분질러（遊戲內：旁白，`StoryData/P10404.json#10`） | し・へ | 四肢をへし切る（遊戲內：旁白，`StoryData/P10404.json#10`） | L.B. | limbs broken（遊戲內：旁白，`StoryData/P10404.json#10`） | 折．肢 | 折斷四肢（遊戲內：旁白，`StoryData/P10404.json#10`） | 2 處：`StoryData/P10404.json#9`、`StoryData/S1005B.json#35` |
| 실. 분. | 실로 분해되는（遊戲內：뫼르소，`RPGSystem/rpg-loc-dialogue-floor-b2.json#51/2`） | い・と・へ・ん・か | 糸へ分解（遊戲內：뫼르소，`RPGSystem/rpg-loc-dialogue-floor-b2.json#51/2`） | S.I.T. | split into threads（遊戲內：뫼르소，`RPGSystem/rpg-loc-dialogue-floor-b2.json#51/2`） | 分．線 | 分解成線（遊戲內：뫼르소，`RPGSystem/rpg-loc-dialogue-floor-b2.json#51/2`） | 2 處：`RPGSystem/rpg-loc-dialogue-floor-b2.json#51/1`、`RPGSystem/rpg-loc-dialogue-floor-b2-b.json#81/1`（附註：floor-b2-b#81/2 同） |
| 알.단.이.예. | 알겠나 단테? 이것이 예술이다（遊戲內：료슈，`PersonalityVoiceDlg/Voice_Ryoshu_Seven6_10402.json#7`） | わ・だ・こ・げ | 分かったかダンテ？これが芸術だ（遊戲內：료슈，`PersonalityVoiceDlg/Voice_Ryoshu_Seven6_10402.json#7`） | GRASPED | Got it, right? Art stands in pure exemplarity, Dante（遊戲內：료슈，`PersonalityVoiceDlg/Voice_Ryoshu_Seven6_10402.json#7`） | 明．了．這．術 | 明白了嗎但丁？這就是藝術（遊戲內：료슈，`PersonalityVoiceDlg/Voice_Ryoshu_Seven6_10402.json#7`） | 2 處：`PersonalityVoiceDlg/Voice_Ryoshu_Seven6_10402.json#7`、`PersonalityVoiceDlg/Voice_Ryoshu_Seven6_10402.json#11`（附註：同句括號展開） |
| 잔.외. | 잔향・외로움（推測） | 寂・残 | 寂しさ・残香（推測） | F.A., S. | Faint Aroma & Solitude（推測） | 餘．孤 | 餘香．孤獨（推測） | 2 處：`PersonalityVoiceDlg/Voice_Ryoshu_LEGOEcho_10414.json#14`、`StoryData/P10414.json#78`（附註：人格名（Personalities.json#169：잔향・외로움／残香・寂しさ／Faint Aroma & Solitude／餘香．孤獨）；JP 兩段次序與人格名相反，寂 標讀音サビ） |
| 좁. 형. | 좁은 형님（遊戲內：싱클레어，`StoryData/S532B.json#10`） | 小・兄 | 小さい兄（遊戲內：싱클레어，`StoryData/S532B.json#10`） | B.B. | B**ch Brother（遊戲內：싱클레어，`StoryData/S532B.json#10`） | 小．兄 | 小心眼的幼兄？（推測） | 2 處：`StoryData/E913B.json#17`、`StoryData/S532B.json#9`（附註：良秀確認 좁＝속 좁은／ケツの穴が小さい／Bitch／小心眼（S532B#74）；로쟈另讀 작은 형님／末兄／Big Brother／幼兄（#11）；雙關中指位階） |
| 초. 깡. | 초록？ 깡통（推測） | み・カ・ン | 緑？のカン（推測） | G.T.C. | green tin can（遊戲內：료슈，`StoryData/P10410.json#68`） | 腦．醃．罐 | 腦啡肽醃罐頭（遊戲內：료슈，`StoryData/P10410.json#68`） | 2 處：`StoryData/P10410.json#69`、`StoryData/P10410.json#76`（附註：KR 깡통 見 #68；초＝초록 KR 無根據，依 EN；JP カ・ン＝カン（缶），拼成みかん雙關） |
| 큰. 형. | 큰형（未縮） | 長・兄 | 長兄（未縮） | G.B. | Great Brother（遊戲內：파우스트，`StoryData/E902B.json#32`） | 長．兄 | 長兄（未縮） | 2 處：`StoryData/E902B.json#62`、`StoryData/E902B.json#62`（附註：EN G.B.＝Great Brother（同場景 #32 파우스트 稱中指的 큰형님 為 a Great Brother of the Middle）；KR、JP、TW 為完整詞逐字拆開） |
| 파.깡. | 파란 깡통（推測） | 青・缶 | 青い缶（推測） | B.T.C. | Blue Tin Can（推測） | 藍．罐 | 藍色罐子（推測） | 2 處：`PersonalityVoiceDlg/Voice_Ryoshu_LEGOEcho_10414.json#1`、`StoryData/P10514.json#60`（附註：指 헤세드（P10514#59–61）） |
| 평.실. | 평범한 실（推測） | 普・糸 | 普通の糸（推測） | O. of a K. | One of a Kind（推測） | 普．線 | 普通的線（推測） | 2 處：`RPGSystem/rpg-loc-dialogue-floor-b2.json#48/1`、`RPGSystem/rpg-loc-dialogue-floor-b2-b.json#78/1` |
| 가. 관. | 가관（未縮） | 見・物 | 見物（未縮） | T.D. | Total？ Disaster（推測） | 亦．見 | 亦可一見？（推測） | `StoryData/E407A.json#35`（附註：TW 推測依據薄弱） |
| 감다뒤 | 감이… 다… 뒤졌다아（遊戲內・他處：료슈딸，`StoryData/S940B.json#34`） | センシ（不加點） | センスが…死んだぁ（遊戲內・他處：료슈딸，`StoryData/S940B.json#34`） | SMH | Shaking... My... Head（遊戲內・他處：료슈딸，`StoryData/S940B.json#34`） | 品死（不加點） | 品味……死光光了（遊戲內・他處：료슈딸，`StoryData/S940B.json#34`） | `PersonalityVoiceDlg/Voice_Ryoshu_KumoPinky_10415.json#20`（附註：借用阿賴耶的流行語；阿賴耶在 S940B#34 展開；EN 原文含 <b> 標籤） |
| 감. 합. | 감쟈… 합니다（遊戲內：료슈만우，`StoryData/E003I3.json#14`） | あ・り | ありがじゅっ…とう…ございましゅ（遊戲內：료슈만우，`StoryData/E003I3.json#14`） | T.Y. | Thank... you（遊戲內：료슈만우，`StoryData/E003I3.json#14`） | 謝．你 | 謝謝你（推測） | `StoryData/E003I3.json#12`（附註：幼兒腔（감쟈＝감사）；TW 劇中作「蟹、蟹……您」，字對不上） |
| 강. 정. | 강렬하고 정열적인（遊戲內：뫼르소，`RPGSystem/rpg-loc-dialogue-floor-b2-b.json#101/5`） | 強・情 | 強烈で情熱的（遊戲內：뫼르소，`RPGSystem/rpg-loc-dialogue-floor-b2-b.json#101/5`） | T.A.R.T. | Torrid and rhapsodic taste（遊戲內：뫼르소，`RPGSystem/rpg-loc-dialogue-floor-b2-b.json#101/5`） | 強．熱 | 強烈而熱情（遊戲內：뫼르소，`RPGSystem/rpg-loc-dialogue-floor-b2-b.json#101/5`） | `RPGSystem/rpg-loc-dialogue-floor-b2-b.json#101/3` |
| 같. 취. | 같은 취급（推測） | 類・友 | 類は…友を（遊戲內：돈키호테，`StoryData/E902B.json#65`） | S.T. | Same treatment（遊戲內：료슈，`StoryData/E902B.json#64`） | 同．待 | 同樣對待（推測） | `StoryData/E902B.json#64`（附註：劇中唐吉訶德誤讀為「같은 취미」「同樣……期待」；JP 類友＝類は友を呼ぶ） |
| 개. 플. | 개인플레이（遊戲內：싱클레어，`StoryData/S833A.json#138`） | 個・プ | 個人プレイ（遊戲內：싱클레어，`StoryData/S833A.json#138`） | P.O.S. | Playing on Solo-mode（遊戲內：싱클레어，`StoryData/S833A.json#138`） | 單．鬥 | 單打獨鬥（遊戲內：싱클레어，`StoryData/S833A.json#138`） | `StoryData/S833A.json#137` |
| 거. 말. 더. 많. | 거참 말 더럽게 많네（推測） | 口・多 | 口数が多い（推測） | T.T.M. | Talk Too Much（推測） | 廢．話．真．多 | 廢話真多（未縮） | `StoryData/E903B.json#247`（附註：TW 逐字加點即完整詞） |
| 건.내.금.포 | 건물 내 금연 포스터（推測） | 建・内・禁・ポ | 建物内禁煙ポスター（推測） | N.S.O.P. | No Smoking On Premises（推測） | 禁．入．建．報 | 禁止進入建築海報？（推測） | `StoryData/E306B.json#14`（附註：TW 字序與 KR 不同，語意不確定） |
| 걸. 작. | 걸작（未縮） | 傑・作 | 傑作（未縮） | A.R.T. | Art（未縮） | 傑．作 | 傑作（未縮） | `StoryData/7D111A.json#37`（附註：KR、JP、TW 逐音節／字加點即完整詞；EN 把 art 逐字母加點？） |
| 검. 결. | 검사 결과（遊戲內：히스클리프，`StoryData/E914B.json#25`） | 検・結 | 検査結果（遊戲內：히스클리프，`StoryData/E914B.json#25`） | T.R. | test results（遊戲內：호엔하임，`StoryData/E914B.json#36`） | 檢．結 | 檢查結果（遊戲內：히스클리프，`StoryData/E914B.json#25`） | `StoryData/E914B.json#37`（附註：前文用語） |
| 검. 완. | 검거 완료（推測） | 検・完 | 検挙完了（推測） | A.O.T.S. | Arrested On The Spot（推測） | 現．逮 | 現場逮捕（推測） | `StoryData/E614B.json#43` |
| 고. 담. | 고급？ 담배（推測） | 高・煙 | 高級煙草（推測） | L.C. | Luxury？ Cigarette（推測） | 高．菸 | 高級香菸（推測） | `PersonalityVoiceDlg/Voice_Ryoshu_Blackwater_10411.json#0`（附註：KR 고＝고급 KR 無直接根據，依 JP 高・、TW 高．；EN L 也可能是 Last（同檔 #2 Last chance today to smoke a cig）） |
| 고.언.옳. | 고기는 언제나 옳다（推測） | 肉・常・正 | 肉は常に正義（推測） | U.M.A.I. | meat's always an unquestionably ideal（遊戲內：료슈，`BattleAnnouncerDlg/Announcer_Ryoshu_42.json#30`） | 肉．永．好 | 肉永遠是最好的（推測） | `BattleAnnouncerDlg/Announcer_Ryoshu_42.json#30`（附註：KR 上下文只有「슬슬 물리는군」，依常見說法；EN 同句解釋字序為 M.A.U.I.，U.M.A.I. 諧日文うまい） |
| 고. 찢. | 고막 찢어지겠다（推測） | 鼓・破 | 鼓膜が破れる（推測） | M.E. | My Ears？（推測） | 耳．裂 | 耳朵要裂了（推測） | `StoryData/E041X.json#50` |
| 공. 절. | 공간 절단（推測） | 空・切 | 空間切断（推測） | D.D.E.D.R.（與 차. 마. 合一） | Dimensional Demon Edge Dimensional Rift？（推測） | 次．魔．空．裂（與 차. 마. 合一） | 次元魔刃空間撕裂（推測） | `StoryData/P11110.json#74`（附註：同劇 P11110#49 有「공간 절단」；技能名 空間切断／Rip Space／空間撕裂；EN、TW 與 차. 마. 合譯） |
| 과.악.만.연. | 과거의 악취가… 만연（遊戲內：료슈，`StoryData/S918B.json#52`） | 過・悪・蔓・延 | 過去の悪臭が…蔓延（遊戲內：료슈，`StoryData/S918B.json#52`） | I.S.O.T.P.I.T.A. | Insidious Stench Of The Past... In The Air（遊戲內：료슈，`StoryData/S918B.json#52`） | 過．惡．蔓．延 | 過去的惡臭……蔓延（遊戲內：료슈，`StoryData/S918B.json#52`） | `StoryData/S918B.json#51`（附註：本人下一句自行展開） |
| 괜. 방. | 괜찮은 방법（推測） | 良・方 | 良い方法（推測） | G.I. | Good Idea（推測） | 妙．招 | 妙招（未縮） | `StoryData/E907B.json#234`（附註：TW 逐字加點即完整詞） |
| 괜. 안. 괜. | 괜찮은지 안 괜찮은지（推測） | だい・ぶ・だい・じょ・ば | 大丈夫かだいじょばないか？（推測） | S.A.T. | sure about this（遊戲內：단테，`StoryData/E903B.json#165`） | 要．不．要．緊 | 要不要緊（未縮） | `StoryData/E903B.json#166`（附註：JP だいじょば＝俗語だいじょばない（大丈夫じゃない）？；EN 呼應但丁「Are you sure about this」） |
| 괜. 예. | 괜찮은 예술（推測） | 悪・無・芸 | 悪くない芸術（推測） | A.W. | Art Work？（推測） | 尚．藝 | 尚可的藝術（推測） | `StoryData/E903B.json#140` |
| 교. 희. | 교체를 희망한다고（遊戲內：싱클레어，`StoryData/E709B.json#4`） | 交・希 | 交代を希望する（遊戲內：싱클레어，`StoryData/E709B.json#4`） | S.T. | switch teams（遊戲內：싱클레어，`StoryData/E709B.json#4`） | 希．換 | 希望換個人（遊戲內：싱클레어，`StoryData/E709B.json#4`） | `StoryData/E709B.json#2` |
| 구.아.구. | 구인회의 아홉 구율（遊戲內：오티스，`StoryData/4D104B.json#7`） | 九・九・規 | 九人会の九つの規律（遊戲內：오티스，`StoryData/4D104B.json#7`） | L.N.L. | League's nine laws（遊戲內：오티스，`StoryData/4D104B.json#7`） | 九．九．律 | 九人會的九戒律（遊戲內：오티스，`StoryData/4D104B.json#7`） | `StoryData/4D104B.json#8` |
| 군.거. | 군것질거리（推測） | 砂・菓 | 砂糖菓子（推測） | J.F. | Junk Food（推測） | 糖．點 | 糖果點心（推測） | `BattleAnnouncerDlg/Announcer_Ryoshu_42.json#30` |
| 그. 가. 낮. | 그럴 가능성은 낮은（遊戲內：싱클레어，`StoryData/E910B.json#50`） | そ・可・低 | その可能性は低い（遊戲內：싱클레어，`StoryData/E910B.json#50`） | T.P.U. | That's Pretty Unlikely（遊戲內：싱클레어，`StoryData/E910B.json#50`） | 那．可．低 | 那種可能性應該很低（遊戲內：싱클레어，`StoryData/E910B.json#50`） | `StoryData/E910B.json#49` |
| 그.로.싱.이. | 그레고르, 로쟈, 싱클레어, 이상（推測） | グ・ロ・シン・イ | グレゴール、ロージャ、シンクレア、イサン（推測） | G., R., S., Y. | Gregor, Rodion, Sinclair, Yi Sang（推測） | 格．羅．辛．李 | 格里高爾、羅佳、辛克萊、李箱（推測） | `BattleAnnouncerDlg/Announcer_Ryoshu_42.json#0`（附註：先前的 Announcer；이／イ 是 이상 不是 이스마엘，依 Announcer 編號（Yisang_6、Ishmael_8）與 EN Y.） |
| 그. 바. 거 | 그럼 바로 거미집（遊戲內：단테，`StoryData/E901B.json#131`） | す・く・い | すぐ蜘蛛の巣へ行く（遊戲內：단테，`StoryData/E901B.json#131`） | H.O.S.R.A. | House of Spiders right away（遊戲內：단테，`StoryData/E901B.json#131`） | 那．直．蜘 | 那是要直接去蜘蛛巢（遊戲內：단테，`StoryData/E901B.json#131`） | `StoryData/E901B.json#132`（附註：良秀轉述단테的話） |
| 그.움.반.해. | 그림자 속에서 움직이는 반푼이 해결사（遊戲內：료슈，`StoryData/S426B.json#33`） | 影・動・阿・フィ | 影の中で動く阿呆なフィクサー（遊戲內：료슈，`StoryData/S426B.json#33`） | F.I.T.S. | Fixers inanely treading shadows（遊戲內：료슈，`StoryData/S426B.json#33`） | 影．移．半．人 | 影子中移動的半吊子收尾人（遊戲內：료슈，`StoryData/S426B.json#33`） | `StoryData/S426B.json#33`（附註：同句寫出） |
| 기. 기. 기. | 기다리고 기다리니… 기어코（遊戲內：료슈，`StoryData/P10406.json#34`） | ま・ま・よ | 待ちに待って…ようやく（遊戲內：료슈，`StoryData/P10406.json#34`） | W.O.W.W.I.E. | Waiting overlong was worth it eventually（遊戲內：료슈，`StoryData/P10406.json#34`） | 等．等．終 | 等啊等啊……終於（遊戲內：료슈，`StoryData/P10406.json#34`） | `StoryData/P10406.json#33` |
| 기. 새. | 기분 나쁜 새끼（推測） | 不・奴 | 不愉快な奴（推測） | A.B. | Annoying bitch？（推測） | 討．混 | 討人厭的混蛋（推測） | `RPGSystem/rpg-loc-dialogue-floor-b2.json#17/0`（附註：뫼르소 只說出前半（floor-b2#17/1：기분 나쁜…／不愉快な…／Annoying b—／討人厭的……）） |
| 기. 잘. 먹. | 기억이라도 잘못 먹은（遊戲內：싱클레어，`StoryData/S1017B.json#61`） | き・ちが | 記憶違い（遊戲內：싱클레어，`StoryData/S1017B.json#61`） | M.P. | memory poisoning（遊戲內：싱클레어，`StoryData/S1017B.json#61`） | 記．吃．錯．了 | 記憶之類的吃錯了（遊戲內：싱클레어，`StoryData/S1017B.json#61`） | `StoryData/S1017B.json#60` |
| 기. 전. | 기전（遊戲內：파우스트만우，`StoryData/E003I2.json#120`） | き・こう | きこう（遊戲內：파우스트만우，`StoryData/E003I2.json#120`） | M.O.D.U.S. | modus（遊戲內：파우스트만우，`StoryData/E003I2.json#120`） | 機．轉（．是啥） | 機轉（遊戲內：파우스트만우，`StoryData/E003I2.json#120`） | `StoryData/E003I2.json#121`（附註：劇中誤讀：本非縮寫（홍루만우 #122）；JP ruby 機構） |
| 길. 말. 필. 없. | 길게 말할 필요 없다（推測） | 長・話・要・無 | 長い話は必要無い（推測） | W.I.H. | Why I'm Here（推測） | 不．要．多．話 | 不要多話（未縮） | `StoryData/P10514.json#51`（附註：EN「You know W.I.H.」後莫梭答又出逃了；TW 逐字加點即完整詞） |
| 꾸. 말. | 꾸물거리지 말아야（遊戲內：뫼르소，`RPGSystem/rpg-loc-dialogue-floor-b2.json#17/9`） | ぐ・ず・す・な | ぐずぐずするべきでない（遊戲內：뫼르소，`RPGSystem/rpg-loc-dialogue-floor-b2.json#17/9`） | D.D. | dilly-dally（遊戲內：뫼르소，`RPGSystem/rpg-loc-dialogue-floor-b2.json#17/9`） | 不．磨 | 不該磨蹭（遊戲內：뫼르소，`RPGSystem/rpg-loc-dialogue-floor-b2.json#17/9`） | `RPGSystem/rpg-loc-dialogue-floor-b2.json#17/8` |
| 꿈. 밤. 모. 분 | 꿈밤, 모가지 분지르기（推測） | 夢・夜・く・へ | 夢見る夜？に首を圧し切る（推測） | S.S.Y.N.C. | Sleepy？ Snap Your Neck Completely（推測） | 安．眠．脖．扭 | 安眠拳、脖子扭斷（推測） | `PersonalityVoiceDlg/Voice_Ryoshu_Butler_10408.json#2`（附註：꿈밤＝넬리的「꿈나라로 보내는 꿀밤」（S612B#3；EN sleepy smack、TW 安眠拳），KR 꿈.밤. 是一詞逐音節加點；JP 把 밤 讀成夜；모.분. 展開見 S101B#46、S210B#5） |
| 날. 치. 기. | 날렵하고 치명적인 기술（遊戲內：싱클레어，`RPGSystem/rpg-loc-dialogue-floor-b2.json#103/4`） | しゅ・び・てき | 俊敏で、致命的な技（遊戲內：싱클레어，`RPGSystem/rpg-loc-dialogue-floor-b2.json#103/4`） | M.U.G. | Mortally Urgent Gift（遊戲內：싱클레어，`RPGSystem/rpg-loc-dialogue-floor-b2.json#103/4`） | 敏．致．技 | 敏捷又致命的技巧（遊戲內：싱클레어，`RPGSystem/rpg-loc-dialogue-floor-b2.json#103/4`） | `RPGSystem/rpg-loc-dialogue-floor-b2.json#103/1`（附註：JP しゅ・び 取自 俊敏（しゅんびん），てき 取自 致命的） |
| 남.손.다.분 | 남은 손가락을 다 분질러（遊戲內：싱클레어，`StoryData/2D102B.json#18`） | の・ゆ・ぜ・へ | 残った指を全部圧し折る（遊戲內：싱클레어，`StoryData/2D102B.json#18`） | B.A.R.F. | break all remaining fingers（遊戲內：싱클레어，`StoryData/2D102B.json#18`） | 餘．指．盡．折 | 剩餘的手指盡數折斷（遊戲內：싱클레어，`StoryData/2D102B.json#18`） | `StoryData/2D102B.json#16` |
| 너. 내. 소. 딸. | 너는 내 소중한 딸（推測） | 私・大・切・娘 | 私の大切な娘（推測） | I.L.Y.T. | I Love You Too（推測） | 我．最．重．兒 | 我最重要的女兒（推測） | `StoryData/S949A.json#124`（附註：回應女兒的완소／ILY；阿賴耶說縮錯了） |
| 녹.마. | 녹음 마이크（推測） | 録・マ | 録音マイク（推測） | R.M. | Recording Mic（推測） | 錄．機 | 錄音機（推測） | `BattleAnnouncerDlg/Announcer_Ryoshu_42.json#32`（附註：交給 Announcer 良秀的麥克風） |
| 눈. 삐. | 눈깔이 삐었군（遊戲內：료슈，`RPGSystem/rpg-loc-dialogue-floor-b2-b.json#54/12`） | 目・節 | 目が節穴（遊戲內：료슈，`RPGSystem/rpg-loc-dialogue-floor-b2-b.json#54/12`） | A.Y.B.D. | are you blind, dumbass（遊戲內：료슈，`RPGSystem/rpg-loc-dialogue-floor-b2-b.json#54/12`） | 眼．瞎 | 眼珠子瞎了（遊戲內：료슈，`RPGSystem/rpg-loc-dialogue-floor-b2-b.json#54/12`） | `RPGSystem/rpg-loc-dialogue-floor-b2-b.json#54/10`（附註：良秀自行解釋） |
| 늙.여. | 늙은 여인（推測） | ろ・ば | 老婆（推測） | O.L. | Old Lady（推測） | 老．婦 | 老婦人（推測） | `PersonalityVoiceDlg/Voice_Ryoshu_LEGOEcho_10414.json#5`（附註：幻想體名；P10414 稱 늙은 여인／老婆（ろうば）／Old Lady／老婦人） |
| 다. 넷. 놈. | 다리가 넷인 놈（推測） | 四・脚・奴 | 四本脚の奴（推測） | F.L.L. | four-legged lowlives（遊戲內：료슈，`PersonalityVoiceDlg/Voice_Ryoshu_Contem_10416.json#3`） | 四．腳．貨 | 四腳貨色（推測） | `PersonalityVoiceDlg/Voice_Ryoshu_Contem_10416.json#3`（附註：同句稱 포레그／フォーレグ／四腳） |
| 다. 많. | 다리 많은（推測） | 多・脚 | 多脚（未縮） | S.M.L. | So Many Legs（推測） | 腳．多（．的） | 腳很多（推測） | `PersonalityVoiceDlg/Voice_Ryoshu_Contem_10416.json#13`（附註：JP、TW 兩字本身即完整詞） |
| 다. 말. | 다 말해（推測） | 全・言 | 全部言え（推測） | F.S. | Full？ Sentence（推測） | 說．全 | 說完全部（推測） | `StoryData/S514B.json#7`（附註：要辛克萊把話說完；EN F 亦可能為 Finish） |
| 다. 분. | 다리를 분질러？（推測） | 細・切 | 細切れ（推測） | S.Y.L.C. | Snap？ Your Legs, Cunt？（推測） | 盡．折 | 盡數折斷？（推測） | `StoryData/E910B.json#56`（附註：KR 比照모.분.（모가지를 분질러）；EN「until you can barely walk」→Legs） |
| 닥. 돌. | 닥치고 돌격（推測） | だま・とつ | 黙って突撃（推測） | C.I. | Charge In？（推測） | 閉．突 | 閉嘴突擊（推測） | `StoryData/S302B.json#9` |
| 닥.돌.죽. | 닥치고 돌격해서 죽기（推測） | 片・突・死 | 片っ端から？突撃して死ぬ（推測） | C.I.D.E. | Charge？ In？ Die？ E？（推測） | 閉．突．死 | 閉嘴突擊去死（推測） | `BattleAnnouncerDlg/Announcer_Ryoshu_42.json#7`（附註：同 S302B#9–11 닥.돌.죽.살.（EN CIDER）的前三字；EN 只有 D＝Die 較可信） |
| 닥.돌.죽.살. | 닥치고 돌격, 죽고 살아나기（推測） | だま・とつ・しん・いき | 黙って突撃・死んで生き返る？（推測） | CIDER | Charge In, Die, E？, R？（推測） | 閉．突．死．生 | 閉嘴突擊、死去又生還？（推測） | `StoryData/S302B.json#11`（附註：닥.돌.＋죽.살.；EN CIDER＝C.I.＋D.E.R.） |
| 닥.시 | 닥쳐, 시계（遊戲內：싱클레어，`StoryData/E303B.json#12`） | だ・と | 黙れ、時計（遊戲內：싱클레어，`StoryData/E303B.json#12`） | S.T.C.U. | shut the clock up（遊戲內：싱클레어，`StoryData/E303B.json#12`） | 閉．鐘 | 閉嘴，時鐘（遊戲內：싱클레어，`StoryData/E303B.json#12`） | `StoryData/E303B.json#11` |
| 닥. 허. | 닥쳐라 허접（遊戲內：료슈，`StoryData/S703B.json#50`） | だ・ざ | 黙れザコ（遊戲內：료슈，`StoryData/S703B.json#50`） | S.T.F.U. | Shut that flaphole, ugly（遊戲內：료슈，`StoryData/S703B.json#50`） | 閉．雜 | 閉嘴吧雜魚（遊戲內：료슈，`StoryData/S703B.json#50`） | `StoryData/S703B.json#47`（附註：辛克萊 #49 先誤讀，良秀 #50 更正） |
| 단. 예. | 단축의 예술（推測） | 短・芸 | 短縮の芸術（推測） | A.O.S. | Art Of Shortening（推測） | 斷．藝 | 斬斷的藝術？（推測） | `StoryData/S918B.json#90`（附註：「단축의 예술／短縮の芸術／art of shortening」見他處語音；TW 斷字所指不確定） |
| 담. 봐. | 담에 봐（推測） | ま・会 | また会おう（推測） | L. | Later（推測） | 再．見 | 再見（未縮） | `StoryData/E914B.json#125`（附註：糾正綺羅的담.또.봐.／ま・会・ね／S.Y.L.／下．次．再．見；KR 담＝다음；TW 逐字加點即完整詞） |
| 답. 하. 모. 분. | 답답하게 하면 모가지를 분질러버린다（推測） | ま・す・く・へ | まだるっこしくするなら首をへし折る（推測） | T.O.I.S.Y.N.C. | Talk, Or I Snap Your Neck Completely（推測） | 如．鬱．脖．扭 | 如果讓我感到鬱悶，就把脖子扭斷（推測） | `StoryData/E042X.json#35`（附註：辛克萊 #36 只展開到「모가지를…／首を…／Snap Your—」就中斷） |
| 도.별. | 도시의 별（遊戲內：호엔하임，`StoryData/E713B.json#67`） | 都・星 | 都市の星（遊戲內：호엔하임，`StoryData/E713B.json#67`） | S.O.T.C. | Star of the City（遊戲內：호엔하임，`StoryData/E713B.json#67`） | 都．星 | 都市之星（遊戲內：호엔하임，`StoryData/E713B.json#67`） | `StoryData/E713B.json#68`（附註：前一句的用語） |
| 도. 언. 안. 진. | 도대체 언제 안으로 진입할（遊戲內：싱클레어，`StoryData/E063X.json#72`） | い・い・な・と | いったいいつになったら中へ突入する（遊戲內：싱클레어，`StoryData/E063X.json#72`） | W.T.H.A.W.G.I. | When the hell are we going in（遊戲內：싱클레어，`StoryData/E063X.json#72`） | 到．時．進．裡 | 到底什麼時候進入裡面（遊戲內：싱클레어，`StoryData/E063X.json#72`） | `StoryData/E063X.json#68`（附註：旁白 #74 確認辛克萊答對） |
| 돈. 없. | 돈키호테 없다（推測） | ド・ない | ドンキホーテがいない（推測） | N.D.Q. | No Don Quixote（推測） | 唐．不 | 唐吉訶德不見了（推測） | `StoryData/E505B.json#29`（附註：以實瑪利接著問唐吉訶德去向；KR 雙關「돈 없다」） |
| 돌. 고. | 돌아다니는 고객들（遊戲內：싱클레어，`RPGSystem/rpg-loc-dialogue-floor-1.json#6/18`） | 徘・客 | 徘徊している顧客（遊戲內：싱클레어，`RPGSystem/rpg-loc-dialogue-floor-1.json#6/18`） | W.C. | wandering customer（遊戲內：싱클레어，`RPGSystem/rpg-loc-dialogue-floor-1.json#6/18`） | 徘．客 | 徘徊於各處的顧客（遊戲內：싱클레어，`RPGSystem/rpg-loc-dialogue-floor-1.json#6/18`） | `RPGSystem/rpg-loc-dialogue-floor-1.json#6/17` |
| 동. 탈. | 동물 탈（推測） | 動・面 | 動物の面（推測） | A.M. | Animal Masks（推測） | 動．面 | 動物面具（推測） | `StoryData/E912B.json#96`（附註：指野獸派） |
| 두. 말. | 두 번 말하게（推測） | 何・同 | 何度も同じこと（推測） | S.I.T. | Say It Twice（推測） | 同．話 | 同樣的話（推測） | `StoryData/P10110.json#36`（附註：KR 或即「두말」一詞加點） |
| 듣. 있. | 듣고 있다（推測） | 通・良 | 通信良好（推測） | I.A.L. | I Am Listening（推測） | 在．聽 | 我在聽（推測） | `PersonalityVoiceDlg/Voice_Ryoshu_LCCB_10406.json#0`（附註：無線電應答） |
| 디. 메. | 디메리트（遊戲內：뉴키라，`StoryData/E910B.json#42`） | デ・メ | デメリット（遊戲內：뉴키라，`StoryData/E910B.json#42`） | T.O. | trade-off（遊戲內：뉴키라，`StoryData/E910B.json#42`） | 副．作 | 副作用（遊戲內：뉴키라，`StoryData/E910B.json#42`） | `StoryData/E910B.json#43`（附註：前一句綺羅的用語） |
| 로드 | 로보토미 드림（遊戲內：이스마엘，`StoryData/S111B.json#27`） | ロド（不加點） | ロボトミードリーム（遊戲內：이스마엘，`StoryData/S111B.json#27`） | LD | Lobotomy Dream（遊戲內：이스마엘，`StoryData/S111B.json#27`） | 腦．夢 | 腦葉夢（遊戲內：이스마엘，`StoryData/S111B.json#27`） | `StoryData/S111B.json#28`（附註：前一句的用語；KR、JP 不加點） |
| 림.컴.버. | 림버스 컴퍼니 버스（推測） | リ・カ・バ | リンバス・カンパニー・バス（推測） | L.C.B.T. | Limbus Company Bus Team？（推測） | 邊．司．巴 | 邊獄公司巴士（推測） | `BattleAnnouncerDlg/Announcer_Ryoshu_42.json#30`（附註：EN 多一個 T；EN 有 bus team 一說（DanteNoteDetail#4）） |
| 림.컴.수. | 림버스 컴퍼니의 수치（遊戲內：오티스，`StoryData/S701B.json#64`） | リン・カン・はじ | リンバス・カンパニーの恥（遊戲內：오티스，`StoryData/S701B.json#64`） | S.O.L.C. | shame of Limbus company（遊戲內：오티스，`StoryData/S701B.json#64`） | 邊．公．恥 | 邊獄公司的恥辱（遊戲內：오티스，`StoryData/S701B.json#64`） | `StoryData/S701B.json#65` |
| 마. 킹. | 마네킹（遊戲內：팔레트，`RPGSystem/rpg-loc-dialogue-floor-b2-b.json#41/4`） | マ・キン | マネキン（遊戲內：팔레트，`RPGSystem/rpg-loc-dialogue-floor-b2-b.json#41/4`） | M.Q. | Mannequin（遊戲內：팔레트，`RPGSystem/rpg-loc-dialogue-floor-b2-b.json#41/4`） | 人．模（．嗎） | 人台模特兒（遊戲內：팔레트，`RPGSystem/rpg-loc-dialogue-floor-b2-b.json#41/4`） | `RPGSystem/rpg-loc-dialogue-floor-b2-b.json#41/3`（附註：팔레트 回話更正（「마네킹 아니야」）） |
| 막. 길. | 막힌 길（推測） | 塞・道 | 塞がった道（推測） | D.E. | Dead End（推測） | 死．路 | 死路（未縮） | `RPGSystem/rpg-loc-dialogue-floor-b2.json#19/0`（附註：TW 兩字本身即完整詞） |
| 만. 족. | 만족（未縮） | よ・し | よし（未縮） | N.B. | Not Bad（推測） | 很．好 | 很好（未縮） | `StoryData/S502B.json#95`（附註：KR、JP、TW 逐音節／字加點即完整詞） |
| 말.넘.심. | 말이 너무 심하다（推測） | 言・酷・過 | 言い方が酷過ぎる（推測） | T.W.H. | That Was Harsh（推測） | 話．太．過 | 話說得太過分（推測） | `StoryData/E306B.json#31` |
| 말. 대. 했. | 말한 대로 했다（推測） | いわ・と・り・やっ | 言われた通りやった（推測） | D.A.Y.S. | Did As You Said（推測） | 照．說．做．了 | 照你說的做了（推測） | `StoryData/E003I2.json#115` |
| 말. 더. 않. | 말을 더 하지 않겠다（推測） | 話・通・無 | 話が通じない（推測） | N.C. | No comment（遊戲內：旁白，`StoryData/S611B.json#62`） | 說．不．通 | 說不通（未縮） | `StoryData/S611B.json#61`（附註：TW 為加點的完整詞） |
| 말. 똑. | 말은 똑바로（推測） | 正・確・言 | 正確に言うと（推測） | B.S. | Bullshit（推測） | 正．確．說 | 正確地說（推測） | `StoryData/PC11A.json#2` |
| 말. 잘. 듣. | 말 잘 듣는（推測） | 話・良・聞・賞 | 話を良く聞く賞（推測） | O.P. | Obedient P？（推測） | 乖．聽．獎 | 乖乖聽話獎（推測） | `StoryData/P10404.json#12`（附註：EN P 可能為 Patient／Pet） |
| 말. 행. | 말보다 행동으로（遊戲內：싱클레어，`StoryData/7D111I3.json#29`） | 言・行 | 言葉よりも行動で（遊戲內：싱클레어，`StoryData/7D111I3.json#29`） | A.C.T.S. | Actions, Cutting Through the Shit（遊戲內：싱클레어，`StoryData/7D111I3.json#29`） | 行．言 | 行動勝於言語（遊戲內：싱클레어，`StoryData/7D111I3.json#29`） | `StoryData/7D111I3.json#28`（附註：EN 略過 the） |
| 맘.들. | 맘에 들어（推測） | 気・入 | 気に入った（推測） | NB | Not Bad（推測） | 我．滿 | 我很滿意（推測） | `StoryData/E402B.json#2` |
| 망. 관. 망. 팀. | 망할 관리자？ 망할 팀（推測） | ク・管・ク・チ | クソ管理人・クソチーム（推測） | P.M.P.C. | Pathetic？ Manager, Pathetic？ Control？（推測） | 爛．管．爛．隊 | 爛主管？爛團隊？（推測） | `StoryData/P10414.json#2` |
| 맞. 말. | 맞는 말（推測） | そ・と | その通り（推測） | I.A. | I Agree（推測） | 說．對 | 說得對（推測） | `StoryData/P11110.json#5` |
| 머. 상. | 머저리 상태（遊戲內：싱클레어，`RPGSystem/rpg-loc-dialogue-floor-1.json#6/18`） | 間・状 | 間抜け状態（遊戲內：싱클레어，`RPGSystem/rpg-loc-dialogue-floor-1.json#6/18`） | C.W. | confused wimps（遊戲內：싱클레어，`RPGSystem/rpg-loc-dialogue-floor-1.json#6/18`） | 不．狀 | 不對勁的狀態（遊戲內：싱클레어，`RPGSystem/rpg-loc-dialogue-floor-1.json#6/18`） | `RPGSystem/rpg-loc-dialogue-floor-1.json#6/17` |
| 머.치. | 머리 좀 치워（遊戲內：싱클레어，`RPGSystem/rpg-loc-dialogue-common-a1c10p1.json#46/4`） | 頭・退 | 頭をどけて（遊戲內：싱클레어，`RPGSystem/rpg-loc-dialogue-common-a1c10p1.json#46/4`） | M.Y.H. | move your head（遊戲內：싱클레어，`RPGSystem/rpg-loc-dialogue-common-a1c10p1.json#46/4`） | 頭．移 | 把頭移開（遊戲內：싱클레어，`RPGSystem/rpg-loc-dialogue-common-a1c10p1.json#46/4`） | `RPGSystem/rpg-loc-dialogue-common-a1c10p1.json#46/3` |
| 머.터. | 머리가 터졌다（推測） | 頭・爆 | 頭が爆発した（推測） | O.O.F. | oof（未縮） | 頭．爆 | 頭爆了（推測） | `StoryData/S926B.json#67`（附註：EN 字母拼成感嘆詞 oof，未見展開） |
| 명.단. | 명줄 단축？（推測） | 命・墓 | 命は墓行き？（推測） | H.I.T. | Hacked In Two（遊戲內：료슈，`PersonalityVoiceDlg/Voice_Ryoshu_Yurodivy_10409.json#12`） | 送．墓 | 送入墳墓（推測） | `PersonalityVoiceDlg/Voice_Ryoshu_Yurodivy_10409.json#12`（附註：KR 雙關 명단（名單），단 也可讀 단절；JP 命（めい）・墓（ぼ）諧 名簿（めいぼ）；EN 同句展開） |
| 모.분.말.도. | 모가지를 분질러야 말을 듣지？ 도야지（推測） | く・へ・い・き・ぶ | 首をへし折らないと言うことを聞かないのか、ブタ（推測） | TRASHES | To Right Against S？ Hogs Entails Scragging（推測） | 脖．扭．豬．閉 | 不把脖子扭斷，這群豬就不會閉嘴（推測） | `StoryData/S430B.json#54`（附註：仿 S101B#45–46「모분조도／くへしぶ／TRANCHES」；TW 與之同形） |
| 모. 시 | 모이나, 시계（遊戲內：싱클레어，`StoryData/E920B.json#74`） | あ・と | 集まるのか、時計（遊戲內：싱클레어，`StoryData/E920B.json#74`） | D.W.R.C. | Do we regroup, Clock（遊戲內：싱클레어，`StoryData/E920B.json#74`） | 集．時 | 要集合嗎，時鐘（遊戲內：싱클레어，`StoryData/E920B.json#74`） | `StoryData/E920B.json#73`（附註：JP あ・と＝あつまる・とけい（讀音）） |
| 모.썰 | 모두 썰어버린다（遊戲內：단테，`StoryData/E302B.json#16`） | ぜん・きり | 全部切り刻んでやる（遊戲內：단테，`StoryData/E302B.json#16`） | STAB | slice them all bare（遊戲內：단테，`StoryData/E302B.json#16`） | 全．切 | 全都切碎（遊戲內：단테，`StoryData/E302B.json#16`） | `StoryData/E302B.json#14` |
| 모. 퇴. | 모조리 퇴출（推測） | 全・退 | 全員退場（推測） | E.W. | 不明 | 全．退 | 全部退場（推測） | `PersonalityVoiceDlg/Voice_Ryoshu_Contem_10416.json#8`（附註：EN E.W. 推不出） |
| 목.에. | 목표 에너지（推測） | 目・エ | 目標エネルギー（推測） | E.Q. | Energy Quota（推測） | 目．能 | 目標能源（推測） | `PersonalityVoiceDlg/Voice_Ryoshu_LEGOEcho_10414.json#22` |
| 목. 죽. | 목소리 죽여（推測） | 息・殺 | 息を殺せ（推測） | S.U. | Shut Up（推測） | 屏．吸 | 屏住呼吸（推測） | `StoryData/E721B.json#50` |
| 뫼. 차. | 뫼르소 차례（推測） | ム・番 | ムルソーの番（推測） | M.'s … T. | Meursault's turn（推測） | 輪．莫 | 輪到莫梭（推測） | `StoryData/S1005B.json#152`（附註：EN 同句有「had my turn」） |
| 무.다.법. | 무기 다루는 법（推測） | 武・扱・方 | 武器の扱い方（推測） | W.A.W. | Wield A Weapon？（推測） | 武．使．方 | 武器的使用方法（推測） | `BattleAnnouncerDlg/Announcer_Ryoshu_42.json#15` |
| 문. 없. | 문제없다（推測） | 問・無 | 問題無し（推測） | N.P. | No Problem（推測） | 沒．問 | 沒問題（推測） | `PersonalityVoiceDlg/Voice_Ryoshu_Wcorp_10405.json#19` |
| 물. 보. 말 | 물어보면 말해（推測） | 訊・無・事 | 訊かれない事は言うな？（推測） | S.Y.M.… Unless asked.（兩個合譯） | Shut Your Mouth（推測） | 沒．問．事 | 沒問的事別說？（推測） | `StoryData/P10413.json#11`（附註：與 안. 물. 입. 닫. 對句；EN 兩個合譯為 S.Y.M.… Unless asked.；JP、TW 末段不確定） |
| 뭐.아 | 뭐가 아니라는（遊戲內：과거료슈3，`StoryData/S947B.json#220`） | な・ち | 何が違うんだ（遊戲內：과거료슈3，`StoryData/S947B.json#220`） | W.I. | What do you mean It's not?（遊戲內：과거료슈3，`StoryData/S947B.json#220`） | 有．差 | 有什麼差別（遊戲內：과거료슈3，`StoryData/S947B.json#220`） | `StoryData/S947B.json#218`（附註：EN W.I. 只取 What、It's） |
| 뭔. 다. 있. | 뭔가 다가오고 있다（推測） | 何・近・寄 | 何かが近寄ってくる（推測） | S.I.C. | Something Is Coming（推測） | 有．什．近 | 有什麼東西在靠近（推測） | `StoryData/S518B.json#73` |
| 뭔. 말. 모. | 뭔 말인지 모르겠다（遊戲內：과거료슈3，`StoryData/E912B.json#55`） | 何・言・不 | 何を言っているのか分からない（遊戲內：과거료슈3，`StoryData/E912B.json#55`） | I.D.U. | I Don't Understand（遊戲內：과거료슈3，`StoryData/E912B.json#55`） | 不．知．說．啥 | 不知道妳在說啥（遊戲內：과거료슈3，`StoryData/E912B.json#55`） | `StoryData/E912B.json#53`（附註：JP 不＝ない） |
| 뭘. 원. 알. | 뭘 원하는지 알아（推測） | 何・知・分 | 何を知りたいのか分かっ（遊戲內：싱클레어，`StoryData/S912B.json#91`） | I.K.W.Y.W. | I know what you wan（遊戲內：싱클레어，`StoryData/S912B.json#91`） | 我．知．妳 | 我知道妳想（遊戲內：싱클레어，`StoryData/S912B.json#91`） | `StoryData/S912B.json#90`（附註：辛克萊解讀被打斷；KR 只說到「뭘 원하시는지…」，알 未出現；EN 完整為 want） |
| 미. 만. | 미관적으로 만족（遊戲內：싱클레어，`StoryData/PC04B.json#29`） | 美・満 | 美観的に満足（遊戲內：싱클레어，`StoryData/PC04B.json#29`） | A.S. | Aesthetically Satisfying（遊戲內：싱클레어，`StoryData/PC04B.json#29`） | 美．滿 | 美觀上感到滿意（遊戲內：싱클레어，`StoryData/PC04B.json#29`） | `StoryData/PC04B.json#28` |
| 바. 머. | 바가지 머리（遊戲內：료슈，`StoryData/E907B.json#55`） | お・頭 | おかっぱ頭（遊戲內：료슈，`StoryData/E907B.json#55`） | B.C. | Bowl Cut（遊戲內：료슈，`StoryData/E907B.json#55`） | 西．頭 | 西瓜頭（遊戲內：료슈，`StoryData/E907B.json#55`） | `StoryData/E907B.json#53` |
| 바. 멍. | 바보 멍청이（推測） | バ・まぬ | バカ、まぬけ（推測） | S.I. | Stupid Idiot（推測） | 笨．蛋 | 笨蛋（未縮） | `StoryData/E003I2.json#130`（附註：TW 逐字加點即完整詞） |
| 반. 반. 위. | 반반보다 위（遊戲內：료슈，`StoryData/E912B.json#108`） | 半・半・上 | 半々より上（遊戲內：료슈，`StoryData/E912B.json#108`） | H.O.H.O.M. | Half Of Half Or More（遊戲內：료슈，`StoryData/E912B.json#108`） | 半．半．上 | 半半以上（遊戲內：료슈，`StoryData/E912B.json#108`） | `StoryData/E912B.json#106`（附註：綺羅 #107 先誤讀為「반의 반보다 위」等，#108 良秀更正） |
| 받.돌. | 받은 만큼 돌려주기（推測） | 受・分・返 | 受けた分だけ返す（推測） | P.B. | Payback（推測） | 原．數．還 | 原數奉還（推測） | `BattleAnnouncerDlg/Announcer_Ryoshu_42.json#25`（附註：EN It's P.B. time＝payback time） |
| 밟. 분. | 밟아 분쇄？（推測） | 踏・粉 | 踏んで粉砕（推測） | S.A.P. | Stomp And Pulverize？（推測） | 踩．粉 | 踩得粉碎（推測） | `PersonalityVoiceDlg/Voice_Ryoshu_Contem_10416.json#7`（附註：KR 분 依 JP 粉／EN P／TW 粉 讀作 분쇄（KR 亦可能是 분노）） |
| 밤. 악. 내. 통. | 밤에 악몽？ 내가？ 통？（推測） | 夜・悪・俺・通 | 夜、悪夢？ 俺、通？（推測） | G.W.N.N & M.T.L. | 不明 | 夜．噩．我．翻 | 夜裡噩夢，我翻來覆去？（推測） | `StoryData/ES290B.json#87`（附註：劇中辛克萊無法解讀；僅「夜／噩夢」較可信） |
| 방. 금. | 방심은 금물（遊戲內：싱클레어，`RPGSystem/rpg-loc-dialogue-floor-b2.json#20/16`） | 油・禁 | 油断は禁物（遊戲內：싱클레어，`RPGSystem/rpg-loc-dialogue-floor-b2.json#20/16`） | N.O.W. | Never off watch（遊戲內：싱클레어，`RPGSystem/rpg-loc-dialogue-floor-b2.json#20/16`） | 切．大 | 切忌大意（遊戲內：싱클레어，`RPGSystem/rpg-loc-dialogue-floor-b2.json#20/16`） | `RPGSystem/rpg-loc-dialogue-floor-b2.json#20/15` |
| 버. 괜. | 버스 기사는 괜찮다（推測） | バ・丈・夫 | バスの運転手は大丈夫（推測） | B.D.O.K. | Bus Driver's OK（推測） | 巴．沒．傷 | 巴士司機沒有受傷（遊戲內：료슈，`StoryData/S902B.json#139`） | `StoryData/S902B.json#139`（附註：同句帶出「버스 기사」；JP バ＋（大）丈夫 雙關） |
| 버.놈. | 버러지？ 같은 놈（推測） | バ・奴 | バカな奴（推測） | B.B. | 不明 | 蠢．傢 | 蠢傢伙（推測） | `BattleAnnouncerDlg/Announcer_Ryoshu_42.json#20`（附註：良秀常用 버러지（Butler_10408#7、P10408#45）；EN B.B.（our team of B.B.）推不出） |
| 버. 못. 들. | 버러지 같은 못들？（推測） | ク・ぎ・共 | クギ共（推測） | W.O.T. | Waste Of Time（推測） | 蟲．釘．們 | 蟲子釘子們（推測） | `StoryData/E923B.json#0`（附註：對手是고정팀（Service des Clous，釘）；KR 或雙關「버러지만도 못한」；JP 把クギ拆兩段（同か・ちく）） |
| 벌.털 | 벌레 털보（遊戲內：그레고르，`StoryData/E302B.json#99`） | け・む | 毛むくじゃらの虫（遊戲內：그레고르，`StoryData/E302B.json#99`） | C.F. | crawling furball（遊戲內：그레고르，`StoryData/E302B.json#99`） | 毛．蟲 | 毛頭蟲子（遊戲內：그레고르，`StoryData/E302B.json#99`） | `StoryData/E302B.json#97` |
| 범. 둘. | 범인 둘（推測） | はん・に | 犯人二人（推測） | T.C. | Two Culprits（推測） | 犯．二 | 犯人有二（推測） | `StoryData/ES290B.json#86` |
| 베.싶.생 | 베고 싶다는 생각（推測） | 斬・望・考 | 斬りたいと望む考え（推測） | S.I.D. | S？ It Down（推測） | 想．考．斬 | 想考慮斬掉它（推測） | `StoryData/S402B.json#83`（附註：EN S 可能為 Slice／Shoot／Strike） |
| 별. 놈. | 별난 놈（推測） | 変・人 | 変人（未縮） | S.O.R. | strange Office Rep（遊戲內：료슈，`StoryData/P10607.json#22`） | 怪．人 | 怪人（未縮） | `StoryData/P10607.json#22`（附註：JP、TW 為加點的完整詞） |
| 별. 다. 금. | 별걸 다 금지하는군（推測） | 変・事・禁 | 変な事まで禁止（推測） | S.P.R. | Stupid？ Pointless？ Rules（推測） | 怪．事．禁 | 怪事都禁止？（推測） | `StoryData/7D101B.json#73`（附註：EN 僅 R=Rules 較可信） |
| 별. 말. | 별말씀을（推測） | 別・問・無 | 別に問題無い（推測） | N.P. | No Problem（推測） | 沒．問 | 沒問題（推測） | `StoryData/S813B.json#123` |
| 별, 없. | 별 이유 없다（推測） | べつ、ない（不加點） | 別に理由はない（推測） | N.M. | Never Mind？（推測） | 沒．因 | 沒什麼原因（推測） | `StoryData/P10402.json#1`（附註：EN 或 No Motive 等；KR 旁白 #3「별 의미는 없었지만」可佐證） |
| 별. 이. 소. 다. | 별 이상한 소리를 다 듣겠군（遊戲內：료슈，`StoryData/P10416.json#58`） | 随・妙・話・聞 | 随分と妙な話を聞かされる（遊戲內：료슈，`StoryData/P10416.json#58`） | S.O.D.A. | Such outlandish drivel, astonishing（遊戲內：료슈，`StoryData/P10416.json#58`） | 盡．聽．怪．話 | 盡是聽到些怪話啊（遊戲內：료슈，`StoryData/P10416.json#58`） | `StoryData/P10416.json#56` |
| 병. 솔. | 병아리？ 인솔？（推測） | ひ・引 | ひよこの引率（推測） | B.S. | Baby Sitting（推測） | 當．保 | 當保姆（推測） | `PersonalityVoiceDlg/Voice_Ryoshu_Wcorp_10405.json#0`（附註：同檔 #6 신입 교육、#10 사람 관리；KR 讀法取自 JP，솔 取 인솔 第二音節，KR 無直接根據；EN #10 有 babysitting） |
| 부. 일. | 부탁할 일（遊戲內：단테，`StoryData/E901B.json#102`） | た・ごと | 頼みごと（推測） | W.K.O.A. | What kind of assistance（遊戲內：단테，`StoryData/E901B.json#102`） | 拜．事 | 拜託的事（遊戲內：단테，`StoryData/E901B.json#102`） | `StoryData/E901B.json#103`（附註：良秀轉述단테的話；JP 劇中作「頼みたいこと」，ごと對不上；파우스트 #125 亦同） |
| 부.재.단.예. | 부의 재분배는 단축의 예술（推測） | 富・再・短・芸 | 富の再分配は短縮の芸術（推測） | R.O.W. is the A.O.S. | Redistribution of Wealth is the Art of Shortening（推測） | 財．再．縮．藝 | 財富再分配是縮短的藝術（推測） | `StoryData/P10409.json#53`（附註：재＝재분배 依旁白 #49 文意與 EN（#49 有 redistribution of wealth）；「단축의 예술／art of shortening」見他處語音） |
| 분.장. | 분홍 장비（推測） | 桃・装 | 桃色の装備（推測） | P.G. | Pink Gear？（推測） | 粉．裝 | 粉色裝備（推測） | `PersonalityVoiceDlg/Voice_Ryoshu_LEGOEcho_10414.json#4`（附註：KR 雙關 분장（扮裝）；指身上的 E.G.O. 裝備） |
| 불. 시. 낭 | 불필요한… 시간 낭비（遊戲內：싱클레어，`StoryData/S913B.json#21`） | 不・時・浪 | 不必要な…時間の浪費（遊戲內：싱클레어，`StoryData/S913B.json#21`） | W.O.T. | Waste... Of... Time（遊戲內：싱클레어，`StoryData/S913B.json#21`） | 無．時．浪 | 無謂的……浪費時間（遊戲內：싱클레어，`StoryData/S913B.json#21`） | `StoryData/S913B.json#19`（附註：TW 展開語序作「浪費時間」，與縮略「時．浪」順序相反） |
| 블.커. | 블랙 커피（推測） | ブラ・コ | ブラックコーヒー（推測） | B.C. | Black Coffee（推測） | 黑．咖 | 黑咖啡（推測） | `PersonalityVoiceDlg/Voice_Ryoshu_LEGOEcho_10414.json#2` |
| 비. 소. | 비명소리（遊戲內：싱클레어，`StoryData/E507B.json#2`） | サ・ビ | 叫び声（遊戲內：싱클레어，`StoryData/E507B.json#2`） | S.O.S. | sounds of screaming（遊戲內：싱클레어，`StoryData/E507B.json#2`） | 慘．聲 | 慘叫聲（遊戲內：싱클레어，`StoryData/E507B.json#2`） | `StoryData/E507B.json#1`（附註：JP サビ 雙關歌曲副歌（呼應「高音域」）） |
| 빠.따. | 빠따（未縮） | バッ・ト | バット（未縮） | Bats（未縮） | Bats（未縮） | 球．棒 | 球棒（未縮） | `StoryData/S1005B.json#40`（附註：指希斯克里夫（球棒）；KR/JP/TW 為整詞加點） |
| 빡. 빡. | 빡빡하다？（推測） | き・つ・い | きつい（未縮） | S.H.I.T. | Shit（未縮） | 火．大 | 火大（未縮） | `RPGSystem/rpg-loc-dialogue-floor-b2-b.json#24/0`（附註：JP、EN、TW 皆為一個詞逐字加點；KR 亦可能讀 빡친다） |
| 빨. 강. | 빨리 강의는 끝내라고（遊戲內：싱클레어，`StoryData/E050X.json#68`） | は・こ | 早く講義は終わらせろ（遊戲內：싱클레어，`StoryData/E050X.json#68`） | E.W.T.L. | Enough with the lecture（遊戲內：싱클레어，`StoryData/E050X.json#68`） | 快．講 | 快點結束講課（遊戲內：싱클레어，`StoryData/E050X.json#68`） | `StoryData/E050X.json#67`（附註：JP は・こ＝はやく・こうぎ（讀音）） |
| 빨. 말. 않. 대. 박. | 빨리 말하지 않으면 대가리 박살 낸다（推測） | 早・言・無・あ・われ | 早く言わないと頭が割れる（推測） | T.O.S.T.O.O.F. | Talk Or Suffer The Obliteration Of Forehead（推測） | 快．說．否．頭．砸 | 快點說，否則就把你的頭砸碎（推測） | `StoryData/S514B.json#4`（附註：辛克萊於 #6、#8 分兩句解釋，非單一單位） |
| 사. 관. | 사전관측（推測） | 事・観 | 事前観測（推測） | P.O. | Preliminary Observation（推測） | 事．觀 | 事前觀測（推測） | `PersonalityVoiceDlg/Voice_Ryoshu_LCCB_10406.json#21`（附註：LCCB 사전관측 3과（S514B 標題、S310B#12）；EN 該課稱 Preliminary Observation Unit 3） |
| 사. 선. | 사람으로 만든 선물（遊戲內：싱클레어，`StoryData/E506A.json#9`） | に・せ・も・の | 人間でせっせと作った贈り物（遊戲內：싱클레어，`StoryData/E506A.json#9`） | G.M.F.H. | Gifts made from humans（遊戲內：싱클레어，`StoryData/E506A.json#9`） | 人．造．物 | 人類製造的禮物（遊戲內：싱클레어，`StoryData/E506A.json#9`） | `StoryData/E506A.json#8`（附註：雙關：JP にせもの（偽物）、TW 人造物） |
| 사.직.놈 | 사무직놈（遊戲內：료슈，`PersonalityVoiceDlg/Voice_Ryoshu_LEGOEcho_10414.json#21`） | じ・しょく | 事務職（遊戲內：료슈，`PersonalityVoiceDlg/Voice_Ryoshu_LEGOEcho_10414.json#21`） | C.E.s | Clerical Employees（推測） | 文．員 | 文職人員（遊戲內：료슈，`PersonalityVoiceDlg/Voice_Ryoshu_LEGOEcho_10414.json#21`） | `PersonalityVoiceDlg/Voice_Ryoshu_LEGOEcho_10414.json#21`（附註：同句前文即寫出；EN 同句作 Clerks，C.E. 未展開） |
| 살. | 살아나기（推測） | 蘇（不加點） | 蘇生（推測） | R. | Repeat（推測） | （無；譯「復活」） | 復活（未縮） | `BattleAnnouncerDlg/Announcer_Ryoshu_42.json#7`（附註：時鐘讓人復活；JP 單字未加點，也可讀 蘇る；EN 套 rinse and repeat，R 兼指復活） |
| 새. 자. | 새 자식（推測） | 新・子 | 新しい子（推測） | N.C. | New Child（推測） | 新．子 | 新的子嗣（推測） | `StoryData/S926B.json#57`（附註：索拉 #63 說「다른 자식들」可佐證） |
| 속. 좁. | 속이 좁다（推測） | ケ・小 | ケツの穴が小さい（推測） | B.B. | 不明 | 氣．小 | 氣量小（推測） | `StoryData/S532B.json#105`（附註：EN 無據可推） |
| 솔. 플. | 솔로 플레이（推測） | ソロ・プ | ソロプレイ（推測） | S.P. | Solo Play（推測） | 單．殺 | 單人擊殺（推測） | `PersonalityVoiceDlg/Voice_Ryoshu_LCCB_10406.json#8` |
| 수. 목. 인. | 수련용 목인？（推測） | 修・木・人 | 修練用の木人（推測） | W.T.D. | wooden training dummy（遊戲內：오티스，`StoryData/S808B.json#142`） | 練．木．人 | 練功木人樁（推測） | `StoryData/S808B.json#141`（附註：KR 劇中稱허수아비；TW 劇中稱木人樁） |
| 수. 시. 다. | 수업 시작（遊戲內：싱클레어，`StoryData/ES130B.json#17`） | じゅ・し | 授業開始（推測） | M.T.L. | much to learn（遊戲內：싱클레어，`StoryData/ES130B.json#17`） | 開．課 | 開始上課（遊戲內：싱클레어，`StoryData/ES130B.json#17`） | `StoryData/ES130B.json#16`（附註：KR 다. 為語尾（수업 시작이다）；JP し取「開始」之「始」音，非字首，故列推測（#17 有「授業開始」）） |
| 순. 돌. | 순번이 돌았군（遊戲內：뫼르소，`RPGSystem/rpg-loc-dialogue-floor-b2.json#16/0`） | 順・回 | 順番が回って（遊戲內：뫼르소，`RPGSystem/rpg-loc-dialogue-floor-b2.json#16/0`） | I.O.T. | It's our turn（遊戲內：뫼르소，`RPGSystem/rpg-loc-dialogue-floor-b2.json#16/0`） | 順．輪 | 順序輪到（遊戲內：뫼르소，`RPGSystem/rpg-loc-dialogue-floor-b2.json#16/0`） | `RPGSystem/rpg-loc-dialogue-floor-b2.json#15/2` |
| 순.차. | 순서대로 차근차근？（推測） | 順・着 | 順番に着実に？（推測） | S.T.E.P. | Step（未縮） | 按．步 | 按照步驟（推測） | `BattleAnnouncerDlg/Announcer_Ryoshu_42.json#14`（附註：EN 字母拼成 step（keep in step，句中 like music or a dance），各字母是否另有所指不明） |
| 순. 협. | 순순히 협조（遊戲內：료슈2，`StoryData/E902B.json#71`） | 大・協 | 大人しく協力（遊戲內：료슈2，`StoryData/E902B.json#71`） | C.Q. | Cooperate quietly（遊戲內：료슈2，`StoryData/E902B.json#71`） | 乖．配 | 乖乖配合（遊戲內：료슈2，`StoryData/E902B.json#71`） | `StoryData/E902B.json#69` |
| 숨. 죽. | 숨겨진 죽간（遊戲內：료슈，`StoryData/P10411.json#3`） | か・ちく | 隠された竹簡（遊戲內：료슈，`StoryData/P10411.json#3`） | S.H.H. | Secret Handover, Here（遊戲內：료슈，`StoryData/P10411.json#3`） | 隱．竹 | 隱藏的竹簡（遊戲內：료슈，`StoryData/P10411.json#3`） | `StoryData/P10411.json#0`（附註：另一隻兔子先誤讀（숨을 죽이라는／家畜／shush／隱入竹林），#3 良秀自己更正） |
| 시. 버. | 시간을 내다 버리는（遊戲內：旁白，`StoryData/P10406.json#10`） | じ・む | 時間を無駄に（遊戲內：旁白，`StoryData/P10406.json#10`） | W.O.T. | waste of time（遊戲內：旁白，`StoryData/P10406.json#10`） | 浪．時 | 浪費時間（遊戲內：旁白，`StoryData/P10406.json#10`） | `StoryData/P10406.json#9` |
| 시. 불. | 시계가 불탄다？（推測） | 時・火 | 時計に火（推測） | C.H. | Clock Head（推測） | 時．火 | 時鐘著火（推測） | `StoryData/S926A.json#80`（附註：시／時＝시계（但丁）；火的說法不確定） |
| 시. 비. | 시계, 비켜（遊戲內：뫼르소，`RPGSystem/rpg-loc-dialogue-floor-b2.json#105/7`） | ど・け | どけ、時計（遊戲內：뫼르소，`RPGSystem/rpg-loc-dialogue-floor-b2.json#105/7`） | T.M. | Timepiece, Move（遊戲內：뫼르소，`RPGSystem/rpg-loc-dialogue-floor-b2.json#105/7`） | 時．挪 | 時鐘，挪開（遊戲內：뫼르소，`RPGSystem/rpg-loc-dialogue-floor-b2.json#105/7`） | `RPGSystem/rpg-loc-dialogue-floor-b2.json#105/6`（附註：JP ど・け 兩字恰為「どけ」） |
| 시. 장. | 시간, 장소（推測） | 時・場 | 時間・場所（推測） | T.A.P. | Time And Place（推測） | 時．地．場 | 時間、地點、場合（推測） | `PersonalityVoiceDlg/Voice_Ryoshu_Contem_10416.json#19`（附註：KR、JP 第三項 상황／状況 照寫；TW 末字「場合」可能照寫） |
| 시. 훈. | 시뮬레이션 훈련（遊戲內：이상，`StoryData/P10412.json#10`） | シ・訓 | シミュレイシェン訓練（遊戲內：이상，`StoryData/P10412.json#10`） | T.S. | Training Simulation（遊戲內：이상，`StoryData/P10412.json#10`） | 模．練 | 模擬訓練（遊戲內：이상，`StoryData/P10412.json#10`） | `StoryData/P10412.json#9` |
| 식. 약. | 식량까지 약탈（遊戲內：싱클레어，`StoryData/PC02B.json#51`） | しょ・り | 食料まで略奪（遊戲內：싱클레어，`StoryData/PC02B.json#51`） | P.T.S.D. | pillage their storage depots（遊戲內：싱클레어，`StoryData/PC02B.json#51`） | 搶．糧 | 連糧食都搶走（遊戲內：싱클레어，`StoryData/PC02B.json#51`） | `StoryData/PC02B.json#50`（附註：TW 字序相反） |
| 신. 고. | 신규 고객（遊戲內：료슈，`StoryData/P10416.json#2`） | 新・客 | 新入りの顧客（遊戲內：료슈，`StoryData/P10416.json#2`） | N.C. | new customer（遊戲內：료슈，`StoryData/P10416.json#2`） | 新．客 | 新客人（遊戲內：료슈，`StoryData/P10416.json#2`） | `StoryData/P10416.json#58`（附註：同場景 #2 良秀先說出全稱） |
| 신. 안 | 신경 안 써（推測） | 気・無 | 気にしない（推測） | D.C. | Don't Care（推測） | 沒．在 | 沒在意（推測） | `StoryData/P10606.json#7`（附註：KR 신 亦可能指紅樓說的「신선」） |
| 신. 정. | 신속하고, 정확하게（遊戲內：료슈，`PersonalityVoiceDlg/Voice_Ryoshu_Butler_10408.json#0`） | 迅・正 | 迅速かつ、正確に（遊戲內：료슈，`PersonalityVoiceDlg/Voice_Ryoshu_Butler_10408.json#0`） | S.P.D. | Speed, Precision, and Decisiveness（遊戲內：료슈，`PersonalityVoiceDlg/Voice_Ryoshu_Butler_10408.json#0`） | 迅．正 | 迅速，而且正確（遊戲內：료슈，`PersonalityVoiceDlg/Voice_Ryoshu_Butler_10408.json#0`） | `PersonalityVoiceDlg/Voice_Ryoshu_Butler_10408.json#0`（附註：同句展開） |
| 실. 크. 다. | 실망이 크다（推測） | 失・デカ | 失望がデカい（推測） | S.A.D. | Such A Disappointment（推測） | 失．望 | 失望（未縮） | `StoryData/P10412.json#18`（附註：TW 兩字本身即完整詞） |
| 싱. 번. | 싱클레어 번역（推測） | シン・訳 | シンクレアの訳（推測） | M.T.L. | Machine Translation？（推測） | 辛．翻 | 辛克萊翻譯（推測） | `StoryData/E914B.json#43`（附註：EN 另造：MTL 慣用縮寫（L 取 translation）） |
| 싱.수. | 싱클레어 수준（推測） | シン・レ | シンクレアレベル（推測） | M.T.L. S.C. | M？ T？ L？ Sinclair（推測） | 辛．萊 | 辛克萊（推測） | `BattleAnnouncerDlg/Announcer_Ryoshu_42.json#27`（附註：TW 把 수준 另寫成「程度」；EN S.C.＝Sinclair（同 Announcer#1），M.T.L. 不明） |
| 싱.어. | 싱클레어（推測） | シン・ア | シンクレア（推測） | S.C. | Sinclair（推測） | 辛．萊 | 辛克萊（推測） | `BattleAnnouncerDlg/Announcer_Ryoshu_42.json#1`（附註：KR 取首、末音節（싱…어）；JP 取 シン…ア；EN 取 Sin／clair） |
| 썩. 표. | 썩은 표정（推測） | 苦・笑 | 苦笑（未縮） | L.F. | Long Face（推測） | 苦．笑 | 苦笑（未縮） | `StoryData/E615A.json#53`（附註：JP、TW 逐字加點即完整詞） |
| 쓸. 소. | 쓸데없는 소리（推測） | 駄・言 | 無駄な言葉（推測） | D.B.S. | Don't Be Silly？（推測） | 別．傻 | 別傻了（推測） | `StoryData/P10809.json#41`（附註：JP 駄言＝無駄な言葉；EN S 或 Stupid） |
| 쓸. 쓸. | 쓸모없고, 쓸데없는（遊戲內：료슈，`StoryData/P10414.json#51`） | か・なし | 蚊の涙ほどの役にも立たん、無駄な話（遊戲內：료슈，`StoryData/P10414.json#51`） | L.O.N.E. | literally otiose, needless, empty（遊戲內：료슈，`StoryData/P10414.json#51`） | 空．荒 | 空洞虛言、滿紙荒唐（遊戲內：료슈，`StoryData/P10414.json#51`） | `StoryData/P10414.json#50`（附註：雙關：쓸쓸／かなし／LONE／空荒；JP 以 ruby カ、ナシ 對應） |
| 아.문.안.연. | 아침엔？ 문 안 연다（推測） | あ・や・て・な | 朝？はやってない（推測） | NOUN | Not Open Until Noon（推測） | 中．前．業 | 中午前不營業（推測） | `PersonalityVoiceDlg/Voice_Ryoshu_23GuCooker_10404.json#1`（附註：KR 아 無根據，依 EN Noon／TW 中午 讀作 아침（亦可能是 아직）；JP あ 讀作 朝 為推測） |
| 안.들. | 안 그래도 들어가 보려고（遊戲內：싱클레어，`StoryData/E724B.json#75`） | ち・入 | ちょうど入ってみよう（遊戲內：싱클레어，`StoryData/E724B.json#75`） | G.N.S. | Good night, snoozefests（遊戲內：싱클레어，`StoryData/E724B.json#75`） | 本．回 | 本來打算回去（遊戲內：싱클레어，`StoryData/E724B.json#75`） | `StoryData/E724B.json#78`（附註：指싱클레어替 재.없. 多補的後半句） |
| 안. 물. 입. 닫. | 안 물어보면 입 닫아（推測） | 訊・無・口・閉 | 訊かれなければ口を閉じろ（推測） | S.Y.M.… Unless asked.（兩個合譯） | Shut Your Mouth（推測） | 沒．問．嘴．閉 | 沒問就把嘴閉上（推測） | `StoryData/P10413.json#11`（附註：EN 兩個合譯為 S.Y.M.… Unless asked.） |
| 안심안지 | 안 심심하고 안 지루（遊戲內：싱클레어，`StoryData/S951B.json#25`） | 退無怠無（不加點） | 退屈でもなく、怠くもない（推測） | SAD | Such A Disappoin（遊戲內：싱클레어，`StoryData/S951B.json#25`） | 不悶不煩（不加點） | 不沉悶也不煩躁（推測） | `StoryData/S951B.json#24`（附註：辛克萊話被打斷；JP 缺末字「無」，TW「無聊」對不上「悶」） |
| 알.단.잘.부 | 알겠나 단테？ 잘 부탁한다（推測） | わ・だ・よ・ろ | 分かったかダンテ？よろしく（推測） | GENNED | 不明 | 知．但．請．關 | 知道了嗎但丁？請多關照（推測） | `PersonalityVoiceDlg/Voice_Ryoshu_Seven6_10402.json#18`（附註：友方陣亡時，請但丁復活；알.단. 依同檔 #7 알.단.이.예.；JP よ・ろ＝よろしく 一詞；EN GENNED 依 GRASPED 句式可能是 Got it…Dante，其餘不明） |
| 알. 둘. 모. | 하나만 알고 둘은 모르는군（遊戲內：료슈，`StoryData/P10414.json#73`） | 一・識・二・不・知 | 一を識って二を知らない（遊戲內：료슈，`StoryData/P10414.json#73`） | N.S.F.P. | not seeing the full picture（遊戲內：료슈，`StoryData/P10414.json#73`） | 知．一．不．知．二 | 只知其一，不知其二（遊戲內：료슈，`StoryData/P10414.json#73`） | `StoryData/P10414.json#71`（附註：JP 不知＝知らない） |
| 알. 듣. | 알아듣게（推測） | わ・る | わかる（推測） | W.Y.M. | What You Mean（推測） | 聽．得．懂．點 | 聽得懂點（未縮） | `StoryData/E003I2.json#124`（附註：TW 逐字加點即完整詞） |
| 알. 듣. 말. | 알아듣게 말해（推測） | 分・よ・言 | 分かるように言え（推測） | W.D.Y.M. | What Do You Mean（推測） | 說．人．話 | 說人話（未縮） | `StoryData/E042X.json#34` |
| 알. 말. | 알고말고（推測） | も・ろん | もちろん（推測） | O.F.C. | Of Course（推測） | 當．然 | 當然（未縮） | `StoryData/E727A.json#12` |
| 알. 먹. | 알아서 먹어（推測） | 勝・食 | 勝手に食え（推測） | I.D.C. | I Don't Care（推測） | 自．吃 | 自己吃（推測） | `PersonalityVoiceDlg/Voice_Ryoshu_Wcorp_10405.json#2` |
| 알. 없. | 알 수 없다（推測） | 分・無 | 分からない（推測） | H.N.I. | Have No Idea（推測） | 不．知．道 | 不知道（未縮） | `StoryData/E903B.json#78`（附註：JP 無＝ない；TW 逐字加點即完整詞） |
| 애. 앞. | 아이 앞에서（遊戲內：싱클레어，`StoryData/E812B.json#111`） | 子・前 | 子供の前で（遊戲內：싱클레어，`StoryData/E812B.json#111`） | N.S.I.F.O.K. | no smoking in front of kids（遊戲內：싱클레어，`StoryData/E812B.json#111`） | 小．前 | 小孩子面前（遊戲內：싱클레어，`StoryData/E812B.json#111`） | `StoryData/E812B.json#110`（附註：KR 애＝아이 的縮約） |
| 애. 애. | 애들에게 애들 다루는 법（遊戲內：싱클레어，`StoryData/ES130B.json#1`） | 子・子 | 子どもに子どもを扱う方法（遊戲內：싱클레어，`StoryData/ES130B.json#1`） | T.K.T.D.W.K. | Teaching kids to deal with kids（遊戲內：싱클레어，`StoryData/ES130B.json#1`） | 小．小 | 教小孩怎麼帶小孩（遊戲內：싱클레어，`StoryData/ES130B.json#1`） | `StoryData/ES130B.json#0` |
| 어. 나.다. | 어린 나다（推測） | 幼・俺 | 幼い俺（推測） | O.C. | Of course（推測） | 幼．我 | 年幼的我（推測） | `StoryData/E906B.json#80`（附註：旁白 #55「어린 시절」佐證） |
| 어. 달. | 어르고 달래야（遊戲內：싱클레어，`StoryData/ES130B.json#11`） | あ・な | あやして宥め（遊戲內：싱클레어，`StoryData/ES130B.json#11`） | S.C.A.N. | soothe, coax, and nurture（遊戲內：싱클레어，`StoryData/ES130B.json#11`） | 撫．安 | 撫摸安慰（遊戲內：싱클레어，`StoryData/ES130B.json#11`） | `StoryData/ES130B.json#8` |
| 어.망.직.식 | 어차피 망하기 직전의 식당（遊戲內：싱클레어，`StoryData/E302B.json#164`） | ど・は・す・れ | どうせ廃業寸前のレストラン（遊戲內：싱클레어，`StoryData/E302B.json#164`） | RAFTS | restaurant already fated to shut（遊戲內：싱클레어，`StoryData/E302B.json#164`） | 遲．要．完．餐 | 遲早要完蛋的餐廳（遊戲內：싱클레어，`StoryData/E302B.json#164`） | `StoryData/E302B.json#163` |
| 어.망.직.전. | 어차피？ 망하기 직전（推測） | ど・破・寸・戦闘 | どうせ破滅？寸前の戦闘（推測） | F.U.B.A.R. | Fucked Up Beyond All Recognition（推測） | 險．滅．前 | 險些？滅亡之前（推測） | `BattleAnnouncerDlg/Announcer_Ryoshu_42.json#8`（附註：KR 직.전. 是 직전 一詞逐音節加點；JP 第四段「戦闘」照寫；EN 為通行縮寫（R 也作 Repair）） |
| 어. 상. | 어이 상실（推測） | あき・た | 呆れた（推測） | G.G. | Good Grief？（推測） | 無．語 | 無語（未縮） | `StoryData/E906B.json#61`（附註：JP あき 標 ruby「呆」；EN 或 Good Game；TW 為加點的完整詞） |
| 어. 없. | 어이가 없다（遊戲內：뉴료슈담배다침，`StoryData/E923I.json#91`） | あき・た | 呆れた（遊戲內：뉴료슈담배다침，`StoryData/E923I.json#91`） | W.T.F. | What a Terrible Front（遊戲內：뉴료슈담배다침，`StoryData/E923I.json#91`） | 無．言 | 真教人無言（遊戲內：뉴료슈담배다침，`StoryData/E923I.json#91`） | `StoryData/E923I.json#89`（附註：本人下一句自行展開） |
| 억. 누. | 억울한 누명（推測） | 濡・衣 | 濡れ衣（推測） | S.T.S. | Stand？ This Slander（推測） | 冤．罪 | 冤罪（未縮） | `StoryData/E910B.json#56`（附註：EN 辛克萊 #57 回「slander」；TW 逐字加點即完整詞） |
| 엄.예. | 엄청？ 예쁘다（推測） | す・ご・き | すごくきれい（推測） | M. thinks it's P. | Mommy thinks it's Pretty（推測） | 很．漂 | 很漂亮（推測） | `StoryData/S947B.json#236`（附註：엄 依 JP／TW 作「很」；EN 讀作 Mommy） |
| 에. 소. | 에고 소드（遊戲內・他處：료슈，`StoryData/E905B.json#4`） | エ・ソ | エゴソード（遊戲內・他處：료슈，`StoryData/E905B.json#4`） | S.S. | sentient sword（遊戲內：단테，`StoryData/E903A.json#47`） | 自．劍 | 有自我的劍（遊戲內・他處：료슈，`StoryData/E905B.json#4`） | `StoryData/E903A.json#44`（附註：後續 E905B 回收（「에고 소드가 아니었나」）；EN 同場景 단테 #47 說出 sentient sword） |
| 엔러쉬 | 엔케팔린 러쉬（遊戲內：이스마엘，`StoryData/S111B.json#27`） | エンラッシュ（不加點） | エンケファリンラッシュ（遊戲內：이스마엘，`StoryData/S111B.json#27`） | E-Rush | Enkephalin Rush（遊戲內：이스마엘，`StoryData/S111B.json#27`） | 腦．潮 | 腦啡肽熱潮（遊戲內：이스마엘，`StoryData/S111B.json#27`） | `StoryData/S111B.json#28`（附註：前一句的用語；KR、JP 不加點） |
| 여. 어. 설. | 여기가 어딘지부터 설명해라（遊戲內：뫼르소，`RPGSystem/rpg-loc-dialogue-floor-b2.json#16/11`） | 場・説 | この場所がどういうところか説明しろ（遊戲內：뫼르소，`RPGSystem/rpg-loc-dialogue-floor-b2.json#16/11`） | W.W.A. | where we're at（遊戲內：뫼르소，`RPGSystem/rpg-loc-dialogue-floor-b2.json#16/11`） | 說．這．哪 | 說明這裡是哪裡（遊戲內：뫼르소，`RPGSystem/rpg-loc-dialogue-floor-b2.json#16/11`） | `RPGSystem/rpg-loc-dialogue-floor-b2.json#16/9` |
| 연. 타. | 연속 타격（推測） | 連・打 | 連続打撃（推測） | C.S. | consecutive strikes（遊戲內：료슈，`PersonalityVoiceDlg/Voice_Ryoshu_LiuAsso4_10407.json#8`） | 連．打 | 連續打擊（推測） | `PersonalityVoiceDlg/Voice_Ryoshu_LiuAsso4_10407.json#8`（附註：KR／JP／TW 也可能就是 연타／連打 一詞逐字加點；EN 同句寫出） |
| 영.준. | 영업 준비（推測） | えい・じゅん | 営業準備（推測） | M.E.P. | Mise En Place（推測） | 餐．備 | 餐前準備（推測） | `PersonalityVoiceDlg/Voice_Ryoshu_23GuCooker_10404.json#1` |
| 예.감.시. | 예술 감상 시간（推測） | 芸・鑑・時 | 芸術鑑賞時間（推測） | A.A.T. | Art Appreciation Time（推測） | 觀．藝．時 | 觀賞藝術的時間（推測） | `PersonalityVoiceDlg/Voice_Ryoshu_LEGOEcho_10414.json#3`（附註：P10514#61 EN 有 art appreciation time 一語（不是展開）） |
| 예.실.인.길. | 예술은 실패했지만 인생은 길다（遊戲內：료슈，`PersonalityVoiceDlg/Voice_Ryoshu_LCB_10401.json#20`） | 芸・失・人・長 | 芸術は失敗したが人生は長い（遊戲內：료슈，`PersonalityVoiceDlg/Voice_Ryoshu_LCB_10401.json#20`） | AFTERWARD | Artistic failure trounces excitement resoundingly, while all roads drag on（遊戲內：료슈，`PersonalityVoiceDlg/Voice_Ryoshu_LCB_10401.json#20`） | 藝．失．人．長 | 藝術雖失敗，人生仍漫長（遊戲內：료슈，`PersonalityVoiceDlg/Voice_Ryoshu_LCB_10401.json#20`） | `PersonalityVoiceDlg/Voice_Ryoshu_LCB_10401.json#20`（附註：同句自行展開） |
| 예. 알. 못. | 예술을 알지 못하는（推測） | 芸・知・無 | 芸術を知らない（推測） | A.L. | artless（遊戲內：단테，`StoryData/ES130B.json#4`） | 藝．無．知 | 藝術無知（推測） | `StoryData/ES130B.json#3`（附註：EN A.L.＝art-less，但丁內心呼應） |
| 오.귀.점.패. | 오늘은 귀찮으니까 점심 따위는 패스（遊戲內：료슈，`PersonalityVoiceDlg/Voice_Ryoshu_Seven6_10402.json#2`） | きょ・め・ひ・ぱ | 今日は面倒だから昼メシなんかパス（遊戲內：료슈，`PersonalityVoiceDlg/Voice_Ryoshu_Seven6_10402.json#2`） | WILT | won’t ingest lunch today（遊戲內：료슈，`PersonalityVoiceDlg/Voice_Ryoshu_Seven6_10402.json#2`） | 今．麻．午．免 | 今天很麻煩，午飯什麼的免了（遊戲內：료슈，`PersonalityVoiceDlg/Voice_Ryoshu_Seven6_10402.json#2`） | `PersonalityVoiceDlg/Voice_Ryoshu_Seven6_10402.json#2`（附註：同句展開） |
| 오. 만. | 오랜만（遊戲內：마왕히스클리프4，`StoryData/P10408.json#52`） | ごう・まん | 郷にまんまと帰ってきやがって（遊戲內：마왕히스클리프4，`StoryData/P10408.json#52`） | R.U.D.E. | rather unfortunate, destructive entrance（遊戲內：마왕히스클리프4，`StoryData/P10408.json#52`） | 無．禮 | 無需多禮（遊戲內：마왕히스클리프4，`StoryData/P10408.json#52`） | `StoryData/P10408.json#51`（附註：雙關：字面為「오만／傲慢／RUDE／無禮」，希斯克里夫回應時展開） |
| 온. 파. | 온통 파랗다（推測） | 全・青 | 全部青い（推測） | B.A.O. | blue all over（遊戲內：싱클레어，`StoryData/E090X.json#93`） | 全．藍 | 全是藍色（推測） | `StoryData/E090X.json#92` |
| 완.소. | 완전 소멸？（推測） | マジ切（不加點） | マジ切り（推測） | I.L.Y. | I Love You（推測） | 真．哀 | 真悲哀（推測） | `BattleAnnouncerDlg/Announcer_Ryoshu_42.json#11`（附註：雙關前句 완소：KR 加點另讀，소 讀法不定；JP マジ切＝マジで切る；EN 反諷套 love you to pieces；TW 真哀 諧音 珍愛） |
| 완소 | 완전 소중（推測） | マジ切（不加點） | マジ大切（推測） | ILY | I Love You（推測） | 珍愛（不加點） | 珍愛（未縮） | `BattleAnnouncerDlg/Announcer_Ryoshu_42.json#11`（附註：阿賴耶的流行語 완소；JP S947B#224 切 標 せつ（大切）） |
| 왜.팔. | 왜 팔을（推測） | 何・腕 | 何で腕を（推測） | W.T.H. | What The Hell（推測） | 何．臂 | 為何手臂（推測） | `BattleAnnouncerDlg/Announcer_Ryoshu_42.json#16`（附註：KR 왜、팔 本身都是單音節詞） |
| 요. 넘. 맛. | 요리가 너무 맛있다（遊戲內：료슈2，`StoryData/E912B.json#31`） | 此・世・味・無 | この世の味ではない（遊戲內：료슈2，`StoryData/E912B.json#31`） | Y.U.M. | Yeah, Unbelievably Mouthwatering（遊戲內：료슈2，`StoryData/E912B.json#31`） | 菜．超．讚 | 菜做得超讚（遊戲內：료슈2，`StoryData/E912B.json#31`） | `StoryData/E912B.json#31`（附註：同句自解；KR 넘＝너무；JP 此＝この、無＝ない） |
| 용. 소. 들. | 용케 소문을 들었군（遊戲內：료슈，`StoryData/P10416.json#12`） | 良・噂・聞 | よくぞ噂を聞きつけた（遊戲內：료슈，`StoryData/P10416.json#12`） | S.W.R.Y. | Surprised word reached you（遊戲內：료슈，`StoryData/P10416.json#12`） | 竟．聽．傳 | 竟然聽到了傳聞啊（遊戲內：료슈，`StoryData/P10416.json#12`） | `StoryData/P10416.json#10` |
| 웃. 소. | 웃기는 소리（推測） | 臍・茶 | 臍で茶を沸かす（推測） | B.S. | Bullshit（推測） | 笑．極 | 可笑至極（推測） | `StoryData/P10408.json#23` |
| 유.잼. | 유잼（未縮） | お・も・ろ | おもろい（推測） | F.A.H. | Fun As Hell（推測） | 有．趣 | 有趣（未縮） | `StoryData/E604B.json#90`（附註：유잼 是俗語本身（유＋재미，S951B#24 亦用「유잼」），只逐字加點） |
| 유잼 | 재미있음（推測） | オモロ（不加點） | おもろい（推測） | NGL | Not Gonna Lie（推測） | 挺有意思的（未縮） | 挺有意思的（未縮） | `StoryData/S951B.json#24`（附註：유잼＝有＋재미 俚語） |
| 이. 계. | 이상한 계략（推測） | 変・計 | 変な計略（推測） | U.S. | Unusual？ Schemes？（推測） | 怪．計 | 怪異計謀（推測） | `PersonalityVoiceDlg/Voice_Ryoshu_Contem_10416.json#11`（附註：KR 無根據，依 JP／EN／TW（계 亦可能是 계획）） |
| 이. 말. | 이상한 말이군（遊戲內：뫼르소，`RPGSystem/rpg-loc-dialogue-floor-b2.json#33/30`） | 妙・言 | 妙なことを言う（遊戲內：뫼르소，`RPGSystem/rpg-loc-dialogue-floor-b2.json#33/30`） | W.T.T.A. | Weird thing to ask（遊戲內：뫼르소，`RPGSystem/rpg-loc-dialogue-floor-b2.json#33/30`） | 怪．話 | 奇怪的話（遊戲內：뫼르소，`RPGSystem/rpg-loc-dialogue-floor-b2.json#33/30`） | `RPGSystem/rpg-loc-dialogue-floor-b2.json#33/29` |
| 이.실.인.길. | 이번？ 실패해도 인생은 길다（推測） | 今・失・人・長 | 今の失敗、人生は長い（推測） | T.O.M.A.R.A.N.A.I. | This outcome may ache, regardless, abide now and improve（遊戲內：료슈，`BattleAnnouncerDlg/Announcer_Ryoshu_42.json#31`） | 現．失．人．長 | 現在失敗，人生還長（推測） | `BattleAnnouncerDlg/Announcer_Ryoshu_42.json#31`（附註：EN 同句展開；TOMARANAI 諧日文止まらない） |
| 이.왜.진. | 이게 왜 진짜（推測） | ま・じ・か | マジか（推測） | N.F.W. | No Fucking Way（推測） | 不．是．吧 | 不是吧（未縮） | `StoryData/P10410.json#71`（附註：JP、TW 為加點的完整詞） |
| 이.히.돈.홍. | 이스마엘, 히스클리프, 돈키호테, 홍루（推測） | イ・ヒ・ドン・ホン | イシュメール、ヒースクリフ、ドンキホーテ、ホンル（推測） | I., H., D.Q., H. | Ishmael, Heathcliff, Don Quixote, Hong Lu（推測） | 以．希．唐．鴻 | 以實瑪利、希斯克利夫、唐吉訶德、鴻璐（推測） | `BattleAnnouncerDlg/Announcer_Ryoshu_42.json#0`（附註：Announcer 編號 Ishmael_8、Heathcliff_14、DonQuixote_21、Honglu_31，都在 Ryoshu_42 之前；이／イ 是 이스마엘，依編號與 EN I.） |
| 일. 말. | 일단 말해봐라（遊戲內：료슈3，`StoryData/ES130A.json#20`） | とり・ま | とりあえずまあ（遊戲內：료슈3，`StoryData/ES130A.json#20`） | T.W.D.T. | Two Words Do the Trick（推測） | 一．完 | 一句話完事？（推測） | `StoryData/ES130A.json#18`（附註：EN 依旁白斜體 two words／do／trick（ES130A#19）；TW 依旁白「一句話」，完字不確定） |
| 일. 수. | 일수？（推測） | 一・手 | 一手（未縮） | P.S. | penetrating strike（遊戲內：료슈，`PersonalityVoiceDlg/Voice_Ryoshu_LiuAsso4_10407.json#8`） | 一．式 | 一式（未縮） | `PersonalityVoiceDlg/Voice_Ryoshu_LiuAsso4_10407.json#8`（附註：KR 推為 일수（一手），也可能是一詞逐音節加點，KR 無直接根據；EN 同句寫出） |
| 임. 동. | 임시로 동행（遊戲內：료슈，`StoryData/E908B.json#112`） | 臨・同 | 臨時で同行（遊戲內：료슈，`StoryData/E908B.json#112`） | T.A. | Temporary Accompaniment（遊戲內：료슈，`StoryData/E908B.json#112`） | 臨．同 | 臨時同行（遊戲內：료슈，`StoryData/E908B.json#112`） | `StoryData/E908B.json#110`（附註：紅樓 #111 先誤讀為「임시 동맹／臨時同盟／Temporary Ally」，良秀 #112 更正） |
| 작. 형. | 작은 형님（遊戲內：다친스미，`StoryData/S510A.json#3`） | ま・け | 末兄（遊戲內：다친스미，`StoryData/S510A.json#3`） | B.B. | Big Brother（遊戲內：다친스미，`StoryData/S510A.json#3`） | 幼．兄 | 幼兄（遊戲內：다친스미，`StoryData/S510A.json#3`） | `StoryData/S510A.json#4`（附註：前一句的用語；JP ま・け＝末兄（まっけい？）讀音） |
| 잔. 썰. | 잔챙이 썰기（推測） | 雑・掃 | 雑魚掃討（推測） | T.A.D. | 不明 | 雜．掃 | 雜魚掃蕩（推測） | `StoryData/P10908.json#39`（附註：EN 無據可推；KR 語尾不確定） |
| 잘. 알. | 잘 알아봤군（推測） | そ・の・と・り | そのとおり（推測） | G.E. | Good Eye（推測） | 沒．錯 | 沒錯（未縮） | `StoryData/S535B.json#38`（附註：KR 語尾不確定） |
| 재.수. | 재수없는 수염（遊戲內：탐정료슈，`StoryData/E605B.json#95`） | で・く | 出しゃばりな口ひげ野郎（遊戲內：탐정료슈，`StoryData/E605B.json#95`） | A.S.S. | Annoying, Stupid Stache（遊戲內：탐정료슈，`StoryData/E605B.json#95`） | 鬼．鬍 | 鬼話連篇的鬍子佬（遊戲內：탐정료슈，`StoryData/E605B.json#95`） | `StoryData/E605B.json#92` |
| 저. 뭐. | 저것들은 뭐냐고（遊戲內：싱클레어，`StoryData/E402A.json#17`） | ア・何 | アレらは何（遊戲內：싱클레어，`StoryData/E402A.json#17`） | R&Ts | Rags and Tags（遊戲內：싱클레어，`StoryData/E402A.json#17`） | 東．說 | 東西在說什麼（遊戲內：싱클레어，`StoryData/E402A.json#17`） | `StoryData/E402A.json#16` |
| 전. 소. | 전부 소탕（推測） | 全・掃 | 全部掃討（推測） | K.T.A. | kill them all（遊戲內：T사대표，`StoryData/E607B.json#33`） | 全．掃 | 完全清掃（遊戲內：T사대표，`StoryData/E607B.json#33`） | `StoryData/E607B.json#32`（附註：EN、TW 由 T사대표 #33 回應時說出（If we do kill them all／如果完全清掃）；KR #33 只說 소탕、JP 只說 掃討，전／全 推為 전부／全部） |
| 전. 연. | 전체연령가（遊戲內：뫼르소，`RPGSystem/rpg-loc-dialogue-floor-b2-b.json#31/0`） | 全・対 | 全年齢対象（遊戲內：뫼르소，`RPGSystem/rpg-loc-dialogue-floor-b2-b.json#31/0`） | A.A.A. | Audience of All Ages（遊戲內：뫼르소，`RPGSystem/rpg-loc-dialogue-floor-b2-b.json#31/0`） | 全．齡（．的） | 全年齡（遊戲內：뫼르소，`RPGSystem/rpg-loc-dialogue-floor-b2-b.json#31/0`） | `RPGSystem/rpg-loc-dialogue-floor-b2-b.json#30/1` |
| 제.행. | 제대로 행동？（推測） | 正・行 | 正しい行動（推測） | P.R.O. | Pro（未縮） | 正．行 | 正確的行動（推測） | `BattleAnnouncerDlg/Announcer_Ryoshu_42.json#17`（附註：KR 與後句 무.상. 合為 제행무상（諸行無常）雙關；EN P.R.O.＋後句 P.E.R.＝PROPER） |
| 젠. 장. | 젠장（未縮） | ク・ソ | クソ（未縮） | S.H.I.T. | Shit（未縮） | 媽．的 | 媽的（未縮） | `PersonalityVoiceDlg/Voice_Ryoshu_Wcorp_10405.json#20`（附註：四種語言都是把一個詞逐字加點） |
| 좋. 말. | 좋다 말았다（遊戲內：旁白，`StoryData/E042X.json#97`） | 糠・喜 | 糠喜び（遊戲內：旁白，`StoryData/E042X.json#97`） | W.A.D. | what a disappointment（遊戲內：旁白，`StoryData/E042X.json#97`） | 真．令．失 | 真令人失望（推測） | `StoryData/E042X.json#101`（附註：旁白 #97 先說出（좋다 말았다／糠喜び／what a disappointment）；TW #97 作「讓人失望」，真、令 對不上，故列推測） |
| 좋.소.다. | 좋은 소식이라（遊戲內：로쟈，`StoryData/P10902.json#10`） | い・ニュ・だ | いいニュースだ（推測） | G.N. | good news（遊戲內：로쟈，`StoryData/P10902.json#10`） | 好．息 | 好消息（遊戲內：로쟈，`StoryData/P10902.json#10`） | `StoryData/P10902.json#9`（附註：KR 다＝이다；JP 羅佳說「いい知らせ」，不含ニュ） |
| 주. 패 | 주먹으로 패서（遊戲內：싱클레어，`RPGSystem/rpg-loc-dialogue-floor-b2-b.json#140/5`） | こ・な | 拳で殴って（遊戲內：싱클레어，`RPGSystem/rpg-loc-dialogue-floor-b2-b.json#140/5`） | P.T. | punching them（遊戲內：싱클레어，`RPGSystem/rpg-loc-dialogue-floor-b2-b.json#140/5`） | 拳．揍 | 拳頭揍（遊戲內：싱클레어，`RPGSystem/rpg-loc-dialogue-floor-b2-b.json#140/5`） | `RPGSystem/rpg-loc-dialogue-floor-b2-b.json#140/3` |
| 죽. 살. | 죽고 살아나기（推測） | しん・いき | 死んで生き返る？（推測） | D.E.R. | Die, E？, R？（推測） | 死．生 | 死去又生還？（推測） | `StoryData/S302B.json#10`（附註：但丁 #15 抱怨要承受的痛苦→反覆死亡復活；EN 只有 D=Die 較可信） |
| 죽. 살. 죽. 살 | 죽다 살아나다 죽다 살아나다（推測） | しん・いき・しん・いき | 死んで生き返り、死んで生き返り（推測） | D.R.D.R. | Die, Revive, Die, Revive（推測） | 死．活．死．活 | 死去又活來，死去又活來（推測） | `StoryData/7D111I.json#9`（附註：取自前句希斯克里夫的說法（#8）） |
| 중.본. | 중앙본부（推測） | 中・本 | 中央本部（推測） | C.C. | Central Command（推測） | 中．本 | 中央本部（推測） | `PersonalityVoiceDlg/Voice_Ryoshu_LEGOEcho_10414.json#1`（附註：腦葉公司的中央本部（UserTicket：로보토미 중앙본부팀／L Corp. Central Command Team）） |
| 즐. 시. | 즐거운 시간（推測） | 愉・時 | 愉しい時間（推測） | F.U.N. | fun（未縮） | 愉．時 | 愉快時光（推測） | `StoryData/E080X.json#111`（附註：EN 字母拼成 fun，未見另有展開） |
| 직. 진. | 직접 나와서 진행（遊戲內：싱클레어，`StoryData/S723B.json#6`） | 直・進 | 直々に出て進行（遊戲內：싱클레어，`StoryData/S723B.json#6`） | G.U.H. | graced us himself（遊戲內：싱클레어，`StoryData/S723B.json#6`） | 直．進 | 直接進場（遊戲內：싱클레어，`StoryData/S723B.json#6`） | `StoryData/S723B.json#5` |
| 질. 지. | 질리고 지루하다（遊戲內：료슈，`StoryData/S810B.json#92`） | あ・き・た | 飽き飽きして退屈（遊戲內：료슈，`StoryData/S810B.json#92`） | B.A.H. | Boring as hell（遊戲內：료슈，`StoryData/S810B.json#92`） | 膩．無 | 看膩了，無聊（遊戲內：료슈，`StoryData/S810B.json#92`） | `StoryData/S810B.json#92`（附註：同句自行展開；JP あ・き・た 亦拼成「飽きた」） |
| 찌.부.다. | 찌부다（未縮） | た・い・ら | 平ら（未縮） | F.L.A.T. | Flat（未縮） | 砸．扁 | 砸扁（未縮） | `PersonalityVoiceDlg/Voice_Ryoshu_Spider_10410.json#8`（附註：四種語言都是把一個詞逐字加點（찌부＋다、たいら、flat、砸扁）） |
| 차. 마. | 차원 마검（推測） | 次・魔 | 次元魔剣（推測） | D.D.E.D.R.（與 공. 절. 合一） | Dimensional Demon Edge Dimensional Rift？（推測） | 次．魔．空．裂（與 공. 절. 合一） | 次元魔刃空間撕裂（推測） | `StoryData/P11110.json#74`（附註：依被動技能名 차원 마검／次元魔剣／Dimensional Demon Edge／次元魔刃；EN、TW 與 공. 절. 合譯，EN 末 R 或為 Rip） |
| 천. 것. | 천한 것（推測） | 卑・者 | 卑しい者（推測） | B.E. | Base Extraction（推測） | 賤．物 | 卑賤之物（推測） | `PersonalityVoiceDlg/Voice_Ryoshu_Contem_10416.json#20` |
| 취. 존. | 취향 존중（推測） | 蓼・虫・好 | 蓼食う虫も好き好き（推測） | Y.D.Y. | You Do You（推測） | 人．各．愛 | 人各有所愛（推測） | `StoryData/P10607.json#4` |
| 타. 둥. | 타래둥지（遊戲內：크로머，`RPGSystem/rpg-loc-dialogue-floor-b2.json#17/3`） | い・巣 | 糸玉の巣（遊戲內：크로머，`RPGSystem/rpg-loc-dialogue-floor-b2.json#17/3`） | S.N. | Skein Nest（遊戲內：크로머，`RPGSystem/rpg-loc-dialogue-floor-b2.json#17/3`） | 線．巢 | 線團之巢（遊戲內：크로머，`RPGSystem/rpg-loc-dialogue-floor-b2.json#17/3`） | `RPGSystem/rpg-loc-dialogue-floor-b2.json#24/1`（附註：良秀引用크로머說的地名（「…이라고 했나?」）） |
| 타. 무. | 타겟 무력화（遊戲內：료슈，`PersonalityVoiceDlg/Voice_Ryoshu_LCCB_10406.json#17`） | タ・む | ターゲット無力化（遊戲內：료슈，`PersonalityVoiceDlg/Voice_Ryoshu_LCCB_10406.json#17`） | T.N. | Target Neutralized（遊戲內：료슈，`PersonalityVoiceDlg/Voice_Ryoshu_LCCB_10406.json#17`） | 目．無 | 目標無效化（遊戲內：료슈，`PersonalityVoiceDlg/Voice_Ryoshu_LCCB_10406.json#17`） | `PersonalityVoiceDlg/Voice_Ryoshu_LCCB_10406.json#17`（附註：同句更正） |
| 태. 만. 잘. | 태엽만 잘 돌아가면（遊戲內：싱클레어，`StoryData/E002X.json#11`） | ぜ・さ・よ | ゼンマイさえよく回るなら（遊戲內：싱클레어，`StoryData/E002X.json#11`） | A.L.A.T.C.T. | As long as the clock turns（遊戲內：싱클레어，`StoryData/E002X.json#11`） | 發．轉．沒 | 發條還在好好轉動就沒什麼問題（遊戲內：싱클레어，`StoryData/E002X.json#11`） | `StoryData/E002X.json#10` |
| 토. 말. 맞. | 토끼의 말이 맞다（遊戲內：싱클레어，`StoryData/PC06B.json#62`） | う・い・た | うさぎの言うことが正しい（遊戲內：싱클레어，`StoryData/PC06B.json#62`） | R.R. | Rabbit’s right（遊戲內：싱클레어，`StoryData/PC06B.json#62`） | 兔．說．對 | 兔子說得對（遊戲內：싱클레어，`StoryData/PC06B.json#62`） | `StoryData/PC06B.json#61` |
| 하. 아. | 하루아침（遊戲內：료슈，`PersonalityVoiceDlg/Voice_Ryoshu_Hongyuan_10413.json#1`） | し・す | 一朝にしてすべて？（推測） | B.O.D.Y. | But One Day a Year（遊戲內：료슈，`PersonalityVoiceDlg/Voice_Ryoshu_Hongyuan_10413.json#1`） | 朝．滅 | 一朝，一切便灰飛煙滅（遊戲內：료슈，`PersonalityVoiceDlg/Voice_Ryoshu_Hongyuan_10413.json#1`） | `PersonalityVoiceDlg/Voice_Ryoshu_Hongyuan_10413.json#1`（附註：同句緊接展開；JP し・す 疑取自同句「一朝にしてすべて」，不確定） |
| 할. 없. | 할 수 없지（推測） | どう・ない | どうしようもない（推測） | N.W.C.D. | nothing we could've done（遊戲內：싱클레어，`StoryData/S529A.json#8`） | 沒．法 | 沒辦法（推測） | `StoryData/S529A.json#7`（附註：辛克萊附和覆述；KR #8「할 수 있는 건 없었겠죠」可佐證） |
| 허. 접. | 허접하다（未縮） | ざ・こ | 雑魚（未縮） | S.A.D. | Such A Disappointment？（推測） | 廢．處 | 廢物，一無是處（推測） | `StoryData/E511A.json#0`（附註：KR、JP 為一個詞逐字加點；EN 同句有 Truly a disappointment（亦可能只是 sad）） |
| 허. 탕. | 허탕（未縮） | 骨・折 | 骨折り損（推測） | A.T.F.N. | All That For Nothing（推測） | 全．費 | 全部白費（遊戲內：료슈，`StoryData/S530B.json#2`） | `StoryData/S530B.json#1`（附註：KR 為加點的完整詞；EN、TW 依本人下一句否定呼應（EN「all that wasn't for nothing」）） |
| 혀. 잘. | 혀가 잘리게（推測） | 舌・切 | 舌切り（推測） | S.Y.T.C. | Sliced？ Your Tongue, Cunt？（推測） | 切．舌 | 切掉舌頭（推測） | `StoryData/S929A.json#12`（附註：EN 比照 S.Y.N.C.，S、C 不確定） |
| 혼.못.죽. | 혼자는 못 죽지（推測） | 死・諸・共 | 死なば諸共（推測） | G.T.F.O. | Get The Fuck Out（推測） | 不．獨．死 | 不會獨自死去（推測） | `StoryData/E724B.json#81` |
| 환. 장. | 환장（遊戲內：료슈，`StoryData/S445B.json#13`） | あ・ほ | アホ（遊戲內：료슈，`StoryData/S445B.json#13`） | BAFFLED | baffled（遊戲內：료슈，`StoryData/S445B.json#13`） | 逼．瘋 | 逼瘋（遊戲內：료슈，`StoryData/S445B.json#13`） | `StoryData/S445B.json#11`（附註：本人更正：未縮略，只是加點） |
| 황. 염. | 황금염료（推測） | 黄・染 | 黄金の染料（推測） | G.D. | Golden Dye（推測） | 黃．染 | 黃金染料（推測） | `RPGSystem/rpg-loc-dialogue-floor-b2.json#33/33`（附註：同場景（floor-b2#30–33）稱 황금염료／黄金の染料／Golden Dye／黃金染料） |
| 흥. 이. | 흥미로운 이야기（推測） | 興・話 | 興味深い話（推測） | I.S. | Interesting Story（推測） | 趣．談 | 有趣的談話（推測） | `StoryData/E908B.json#45` |

### 3.2 同義異寫（6 種、12 行）

| KR 縮略 | KR 完整 | JP 縮略 | JP 完整 | EN 縮略 | EN 完整 | TW 縮略 | TW 完整 | 位置 |
|---|---|---|---|---|---|---|---|---|
| 알. 못. | 알지 못하는 놈（推測） | そ・ん・わ・な | そんなわけない？（推測） | I.F. | ignorant fool（推測） | 懂．自．懂 | 懂的自然懂（推測） | 6 處：`StoryData/S828B.json#31`、`StoryData/E912B.json#39`、`StoryData/E020X.json#44`、`StoryData/S524B.json#66`、`StoryData/S524B.json#70`、`StoryData/E030X.json#94`（附註：KR 俚語「알못」，E030X#94「알. 못아」、S524B#70「알. 못. 들」指人；JP S524B#70「そ・ん・わ・なこと」作修飾語，與 そんなわけない 相合；EN「you I.F.」「surrounded by I.F.」） |
| 알.못. | 알지 못하는 놈（推測） | そ・ん・わ・な | そんなわけない？（推測） | T.N.W.I.M. | That's not what I meant（推測） | 懂．自．懂 | 懂的自然懂（推測） | `StoryData/P10410.json#56`（附註：KR 為俚語「알못」，他處「알. 못아」（E030X#94）、「알. 못. 들」（S524B#70）指人；EN 另造 T.N.W.I.M.（回應李箱更正）） |
| 거. 거.（另 거.거.다.） | 거기서 거기（遊戲內：료슈2，`StoryData/E903A.json#7`） | 似・寄 | 似たり寄ったり（遊戲內：료슈2，`StoryData/E903A.json#7`） | S.D. | Same difference（遊戲內：료슈2，`StoryData/E903A.json#7`） | 此．此 | 此處彼處，此間彼間（遊戲內・他處：료슈，`StoryData/E507A.json#45`） | 3 處：`BattleAnnouncerDlg/Announcer_Ryoshu_42.json#19`、`StoryData/E903A.json#5`、`StoryData/E912B.json#51`（附註：TW 在 E903A#7 作「此處彼處，都一樣」，對不上第二個此，取 E507A#45） |
| 거. 거. | 거기서 거기（遊戲內：료슈，`StoryData/E507A.json#45`） | に・よ | 似たり寄ったり（遊戲內：료슈，`StoryData/E507A.json#45`） | S.S.E. | Same story everywhere（遊戲內：료슈，`StoryData/E507A.json#45`） | 此．此 | 此處彼處，此間彼間（遊戲內：료슈，`StoryData/E507A.json#45`） | `StoryData/E507A.json#43`（附註：JP に・よ取讀音（似、寄）） |
| 노. 잼. | 노잼（未縮） | だ・る | だるい（推測） | B.A.H. | Boring as hell（遊戲內・他處：료슈，`StoryData/S810B.json#92`） | 無．聊 | 無聊（未縮） | 2 處：`StoryData/E054X.json#62`、`StoryData/E054X.json#76`（附註：노잼 是俗語本身；JP 或即俗語「だる」；EN B.A.H. 展開見 S810B#92） |
| 노.잼. | 노잼（未縮） | つ・マン・ネ | つまんね（未縮） | B.A.H. | Boring as hell（遊戲內・他處：료슈，`StoryData/S810B.json#92`） | 無．聊 | 無聊（未縮） | `PersonalityVoiceDlg/Voice_Ryoshu_LCCB_10406.json#6`（附註：노잼 是俗語本身（no＋재미），只逐字加點；EN B.A.H. 展開見 S810B#92（該處 KR 為 질. 지.）） |
| 자.정. | 자색 정오（遊戲內：이상，`StoryData/P10410.json#44`） | む・の・う | 紫の白昼？（推測） | N.O.V. | Noon of Violet（遊戲內：이상，`StoryData/P10410.json#44`） | 紫．午 | 紫羅蘭的正午（遊戲內：이상，`StoryData/P10410.json#44`） | 2 處：`StoryData/P10410.json#41`、`StoryData/P10410.json#43`（附註：劇中誤讀：#42 이상先讀成 자정（JP 無能、EN Night of Violet、TW 午夜），#44 才對；JP #44 作「紫の白昼」但「う」對不上，故列推測） |
| 자.정. | 자색 정오（遊戲內：이상，`StoryData/P10410.json#44`） | む・の・う | 紫の白昼？（推測） | D.O.V. | D？ of Violet（推測） | 紫．午 | 紫羅蘭的正午（遊戲內：이상，`StoryData/P10410.json#44`） | `StoryData/P10410.json#54`（附註：劇中誤讀：#55 이상又讀成 자정（EN Ordeal of Midnight）；EN 改作 D.O.V.，D 不明；JP「う」對不上 白昼） |
| 아.닥. | 아가리를 닥치고（遊戲內・他處：호엔하임，`StoryData/E914B.json#38`） | 口・閉 | 口を閉じろ（推測） | S.T.F.U. | shut the fuck up（遊戲內・他處：호엔하임，`StoryData/E914B.json#38`） | 閉．嘴 | 閉上嘴（遊戲內・他處：호엔하임，`StoryData/E914B.json#38`） | `StoryData/S947B.json#314`（附註：展開在他處（E914B#38）；EN S.T.F.U. 在 S703B#50 另解為 Shut that flaphole, ugly（該處 KR 為 닥. 허.）） |
| 아. 닥. | 아가리를 닥치고（遊戲內：호엔하임，`StoryData/E914B.json#38`） | だ・ま・れ | だまれ（遊戲內：호엔하임，`StoryData/E914B.json#38`） | S.T.F.U. | shut the fuck up（遊戲內：호엔하임，`StoryData/E914B.json#38`） | 閉．嘴 | 閉上嘴（遊戲內：호엔하임，`StoryData/E914B.json#38`） | `StoryData/E914B.json#33`（附註：#41 뉴키라 另作曲解） |
| 알. 바. | 알 바는 아니（遊戲內・他處：탐정로쟈，`StoryData/E614A.json#12`） | し・る・か | 知るか（未縮） | N.O.M.B. | None of my business（推測） | 管．他．的 | 管他的（未縮） | `StoryData/P10409.json#38`（附註：KR 展開在他處（E614A#12）；JP、TW 為完整詞逐字拆開） |
| 알. 바. | 알 바는 아니（遊戲內：탐정로쟈，`StoryData/E614A.json#12`） | し・る・か | 知るか（未縮） | W.C. | Who cares（遊戲內：탐정로쟈，`StoryData/E614A.json#12`） | 管．他．的 | 管他的（未縮） | `StoryData/E614A.json#11`（附註：JP し・る・か、TW 管．他．的 為完整詞逐字拆開；로쟈的 JP「知ったこっちゃない」、TW「誰會在乎呢」對不上縮略的字） |

### 3.3 異義（4 種、9 行）

| KR 縮略 | KR 完整 | JP 縮略 | JP 完整 | EN 縮略 | EN 完整 | TW 縮略 | TW 完整 | 位置 |
|---|---|---|---|---|---|---|---|---|
| 황. 가.（提案 37） | 황금가지（推測） | 黄・枝 | 黄金の枝（推測） | G.B. | Golden Bough（推測） | 金．枝 | 金枝（未縮） | `StoryData/E914B.json#88`（附註：KR 無直接展開，依 KR 劇情詞 황금가지（尋回失物）與 S1013B#11 뫼르소 的讀法，JP 黄・枝 佐證；TW 為完整詞逐字拆開） |
| 황. 가.（提案 37） | 황금가죽（遊戲內：료슈，`StoryData/S1013B.json#12`） | お・の | 黄金の皮（遊戲內：료슈，`StoryData/S1013B.json#12`） | G.B. | Golden Buildingskin（遊戲內：료슈，`StoryData/S1013B.json#12`） | 黃．金（．才行嗎） | 黃金皮革（遊戲內：료슈，`StoryData/S1013B.json#12`） | `StoryData/S1013B.json#10`（附註：劇中誤讀：#11 뫼르소讀成 황금가지（JP 黄金の枝、EN Golden Bough、TW 金枝），#12 良秀更正；JP お＝黄金（おうごん）） |
| 황. 가.（提案 37） | 황금가죽（遊戲內：료슈，`StoryData/S1013B.json#12`） | お・の | 黄金の皮（遊戲內：료슈，`StoryData/S1013B.json#12`） | G.B. | Golden Buildingskin（遊戲內：료슈，`StoryData/S1013B.json#12`） | 黃．金（．指的） | 黃金皮革（遊戲內：료슈，`StoryData/S1013B.json#12`） | `StoryData/S1013B.json#12`（附註：同句更正（否定 황금가지）；JP お＝黄金（おうごん）） |
| 말. 안. 어. | 말 안 듣는 어른（遊戲內：싱클레어，`StoryData/ES130B.json#10`） | 言・聞・無・大 | 言うことを聞かない大人（遊戲內：싱클레어，`StoryData/ES130B.json#10`） | U.A. | unruly adult（遊戲內：싱클레어，`StoryData/ES130B.json#10`） | 不．聽．大 | 不聽話的大人（遊戲內：싱클레어，`StoryData/ES130B.json#10`） | `StoryData/ES130B.json#6`（附註：JP 無＝ない；同寫法 #8 改指 어린아이（雙關）） |
| 말. 안. 어. | 말 안듣는 어린아이（遊戲內：싱클레어，`StoryData/ES130B.json#11`） | 言・聞・無・子 | 言うことを聞かない子ども（遊戲內：싱클레어，`StoryData/ES130B.json#11`） | U.C. | unruly child（遊戲內：싱클레어，`StoryData/ES130B.json#11`） | 不．聽．小 | 不聽話的小孩（遊戲內：싱클레어，`StoryData/ES130B.json#11`） | `StoryData/ES130B.json#8`（附註：JP 無＝ない；KR 어 同字 #6 指 어른、#8 指 어린아이（雙關）） |
| 모.불.아.위. | 모든 것을 불살라서라도, 아름다움을 위해（遊戲內：료슈，`EGOVoiceDig/Voice_EGO_Ryoshu_4.json#0`） | 総・燃・美・為 | 総てを燃やし尽くせども、美しさの為（遊戲內：료슈，`EGOVoiceDig/Voice_EGO_Ryoshu_4.json#0`） | O.S.H.A.R.E. | Obligate style hearkens aflame this robust earth（遊戲內：료슈，`EGOVoiceDig/Voice_EGO_Ryoshu_4.json#0`） | 縱．焚．僅．美 | 縱使焚燒一切，僅為這片美景（遊戲內：료슈，`EGOVoiceDig/Voice_EGO_Ryoshu_4.json#0`） | `EGOVoiceDig/Voice_EGO_Ryoshu_4.json#0`（附註：同句括號展開；EN 的 this 不在縮寫內） |
| 모.불.아.위. | 모든 것을 불살라서라도, 아라야를 위해（遊戲內：료슈，`EGOVoiceDig/Voice_EGO_Ryoshu_4.json#1`） | 総・燃・阿・為 | 総てを燃やし尽くせども、アラヤの為（遊戲內：료슈，`EGOVoiceDig/Voice_EGO_Ryoshu_4.json#1`） | O.S.H.A.R.E. | Overcome the Searing Hellfire for Araya, to Reality's Extinction（遊戲內：료슈，`EGOVoiceDig/Voice_EGO_Ryoshu_4.json#1`） | 縱．焚．僅．阿 | 縱使焚燒一切，僅為了阿賴耶（遊戲內：료슈，`EGOVoiceDig/Voice_EGO_Ryoshu_4.json#1`） | `EGOVoiceDig/Voice_EGO_Ryoshu_4.json#1`（附註：同句括號展開；JP 阿＝阿頼耶（文中寫アラヤ）） |
| 어.이.（未確認；提案 21） | 어이（未縮） | お・い | おい（未縮） | H.W. | Hey, W？（推測） | 喂．你 | 喂，你（未縮） | `StoryData/5D205B.json#3`（附註：對魁魁格的招呼；KR 無展開；EN W 不明） |
| 어.이.（未確認；提案 21） | 어이（未縮） | お、イ（用逗號） | おい、イサン（推測） | E.Y. | Eh, Yi？（推測） | 喂，李（用逗號） | 喂，李箱（推測） | `StoryData/S445B.json#0`（附註：李箱答「부른 거요?」；KR 無展開，未確認是否含 이상；JP／TW 用李箱的「イ／李」，EN Y 疑為 Yi（或 you）；現有資料此句 JP 為空，JP 依先前紀錄的 お、イ） |

### 3.4 中文不一（16 種、40 行）

| KR 縮略 | KR 完整 | JP 縮略 | JP 完整 | EN 縮略 | EN 完整 | TW 縮略 | TW 完整 | 位置 |
|---|---|---|---|---|---|---|---|---|
| 모. 분.（另屬同義異寫；提案 10） | 모가지를 분질러버린다（遊戲內：료슈，`StoryData/S210B.json#5`） | く・へ | 首をへし斬れば（遊戲內：싱클레어，`StoryData/ES130B.json#10`） | S.Y.N.C.（另 S.Y.N.—、SYNC） | Snap Your Neck Completely（遊戲內：료슈，`StoryData/S210B.json#5`） | 脖．扭 | 脖子扭斷（推測） | 16 處：`BattleAnnouncerDlg/Announcer_Ryoshu_42.json#1`、`BattleAnnouncerDlg/Announcer_Ryoshu_42.json#5`、`RPGSystem/rpg-loc-dialogue-floor-b2-b.json#36/25`、`StoryData/ES130B.json#6`、`StoryData/P10409.json#14`、`StoryData/P10413.json#18`、`StoryData/4D305A.json#25`、`StoryData/S504B.json#53`、`StoryData/S422B.json#3`、`StoryData/S210B.json#3`、`StoryData/S502B.json#41`、`StoryData/S813B.json#12`、`StoryData/P10408.json#8`、`StoryData/E605B.json#52`、`StoryData/S1005B.json#35`、`PersonalityVoiceDlg/Voice_Ryoshu_Butler_10408.json#21`（附註：TW 劇中解釋作「折斷你脖子」（S210B#5），無「扭」） |
| 모. 분.（另屬同義異寫；提案 10） | 모가지를 분질러버린다（遊戲內・他處：료슈，`StoryData/S210B.json#5`） | く・へ | 首をへし斬れば（遊戲內・他處：싱클레어，`StoryData/ES130B.json#10`） | S.H.N.C. | Snap Her Neck Completely（推測） | 脖．扭 | 脖子扭斷（推測） | `RPGSystem/rpg-loc-dialogue-floor-b2.json#22/9`（附註：KR／JP 展開見 S210B#5、ES130B#10；EN H＝Her（對象克羅默，女）） |
| 모. 분.（另屬同義異寫；提案 10） | 모가지를 분질러버린다（遊戲內・他處：료슈，`StoryData/S210B.json#5`） | く・へ | 首をへし斬れば（遊戲內・他處：싱클레어，`StoryData/ES130B.json#10`） | C.I.H. | Cut In Half（推測） | 脖．扭 | 脖子扭斷（推測） | `StoryData/E725B.json#35`（附註：KR／JP 展開見 S210B#5、ES130B#10；EN 依句法（have the clock C.I.H）與同場景 #45「cut this beast in half」） |
| 모. 분.（另屬同義異寫；提案 10） | 모가지를 분질러버린다（遊戲內・他處：료슈，`StoryData/S210B.json#5`） | く・へ | 首をへし斬れば（遊戲內・他處：싱클레어，`StoryData/ES130B.json#10`） | SYNC | Snap Your Neck Completely（遊戲內・他處：료슈，`StoryData/S210B.json#5`） | 脖．扭．豬．閉 | 不扭斷脖子就不會閉嘴啊。那些跟豬沒兩樣的傢伙們（遊戲內・他處：료슈，`BattleAnnouncerDlg/Announcer_Ryoshu_42.json#28`） | `StoryData/E406B.json#16`（附註：EN SYNC＝S.Y.N.C.；TW 沿用 모.분.조.도. 的四段譯法（Announcer#28 同句解釋，語序不同；S101B#46 作「脖子扭斷…豬狗…閉嘴」），莫梭改作蟹．閉） |
| 모. 분.（另屬同義異寫；提案 10） | 모가지를 분질러버린다（遊戲內・他處：료슈，`StoryData/S210B.json#5`） | く・へ | 首をへし斬れば（遊戲內・他處：싱클레어，`StoryData/ES130B.json#10`） | O.O.F. | obliteration of forehead？（推測） | 脖．扭 | 脖子扭斷（推測） | `StoryData/E512B.json#14`（附註：KR／JP 展開見 S210B#5、ES130B#10；EN O.O.F. 本處未展開，推為沿用 S504B#111 的 obliteration of forehead（該處 KR 為 대. 박.，描述頭被打爛），本句只說要殺她，存疑） |
| 모. 분.（另屬同義異寫；提案 10） | 모가지를 분질러버린다（遊戲內・他處：료슈，`StoryData/S210B.json#5`） | く・へ | 首をへし斬れば（遊戲內・他處：싱클레어，`StoryData/ES130B.json#10`） | S.T.N.C. | Snap Their Necks Completely（推測） | 脖．扭 | 脖子扭斷（推測） | `StoryData/PC02B.json#50`（附註：KR／JP 展開見 S210B#5、ES130B#10；EN 比照 S.Y.N.C.（Snap Your Neck Completely）） |
| 시. 대.（提案 1） | 시계 대가리（遊戲內：히스클리프，`RPGSystem/rpg-loc-dialogue-common-a1c10p1.json#50/0`） | 時・ヅ | 時計ヅラ（遊戲內：히스클리프，`RPGSystem/rpg-loc-dialogue-common-a1c10p1.json#50/0`） | C.H. | Clockhead（遊戲內：히스클리프，`RPGSystem/rpg-loc-dialogue-common-a1c10p1.json#50/0`） | 時．頭 | 時鐘頭（遊戲內：히스클리프，`RPGSystem/rpg-loc-dialogue-common-a1c10p1.json#50/0`） | 11 處：`BattleAnnouncerDlg/Announcer_Ryoshu_42.json#13`、`BattleAnnouncerDlg/Announcer_Ryoshu_42.json#14`、`RPGSystem/rpg-loc-dialogue-common-a1c10p1.json#50/2`、`RPGSystem/rpg-loc-dialogue-floor-b2.json#20/3`、`RPGSystem/rpg-loc-dialogue-floor-b2.json#99/5`、`RPGSystem/rpg-loc-dialogue-floor-1-b.json#3/6`、`StoryData/E901B.json#103`、`StoryData/S1017B.json#57`、`StoryData/E001X.json#23`、`StoryData/S929A.json#4`、`StoryData/S1004B.json#16`（附註：同場景 #50/0 히스클리프喊出；EN 合寫 Clockhead） |
| 시. 대（提案 1） | 시계 대가리（遊戲內・他處：히스클리프，`RPGSystem/rpg-loc-dialogue-common-a1c10p1.json#50/0`） | 時・ヅ | 時計ヅラ（遊戲內・他處：히스클리프，`RPGSystem/rpg-loc-dialogue-common-a1c10p1.json#50/0`） | C.H. | Clockhead（遊戲內・他處：히스클리프，`RPGSystem/rpg-loc-dialogue-common-a1c10p1.json#50/0`） | 鐘．頭 | 時鐘頭（推測） | `StoryData/S1027B.json#18`（附註：展開見 RPG:common-a1c10p1#50/0；TW 鐘取「時鐘」之鐘，劇中「時鐘頭」對應的是時．頭） |
| 관. 없.（另屬同義異寫；提案 2、3） | 관심 없다（推測） | 興・無 | 興味無い（推測） | I.D.C. | I Don't Care（推測） | 無．趣 | 無趣（未縮） | `PersonalityVoiceDlg/Voice_Ryoshu_LEGO_10412.json#4`（附註：KR／JP 同句有「관심 있다」「興味ない」；TW 無．趣即「無趣」（P10514#63 재. 없. 亦作無．趣）） |
| 관. 없.（另屬同義異寫；提案 2、3） | 관심 없다（推測） | 関・無 | 関心無い（推測） | I.D.C. | I Don't Care（推測） | 沒．興 | 沒興趣（推測） | `StoryData/P10412.json#35`（附註：指이상 #30「이제는 관심 밖의 일이오」（JP 関心の外、TW 不感興趣）） |
| 관. 없.（另屬同義異寫；提案 2、3） | 관심 없다（推測） | 興・無 | 興味無い（推測） | I.D.C. | I Don't Care（推測） | 無．謂 | 無所謂（推測） | `StoryData/E605B.json#31` |
| 그.그.（另屬異義；提案 34） | 그럼 그렇지（推測） | 案・定 | 案の定（推測） | W.A. | was anticipated（遊戲內・他處：뉴키라，`StoryData/E903A.json#8`） | 意．中 | 意料之中（推測） | `BattleAnnouncerDlg/Announcer_Ryoshu_42.json#19`（附註：KR 依語氣（쳇…하지만…거.거.）讀，非 그게 그거，JP／EN 佐證；EN W.A. 展開見 E903A#8） |
| 그. 그.다.（另屬異義；提案 34） | 그게 그거（遊戲內：뉴키라，`StoryData/E903A.json#8`） | に・もの | 似たもの（遊戲內：뉴키라，`StoryData/E903A.json#8`） | W.A. | was anticipated（遊戲內：뉴키라，`StoryData/E903A.json#8`） | 都．樣 | 都一樣（遊戲內：뉴키라，`StoryData/E903A.json#8`） | `StoryData/E903A.json#9`（附註：키라叫료슈縮成「그게 그거」，료슈再縮成 그. 그.） |
| 그. 그.（另屬異義；提案 34） | 그게 그거（遊戲內：싱클레어，`StoryData/E302B.json#33`） | ど・ど | どっちもどっち（遊戲內：싱클레어，`StoryData/E302B.json#33`） | S.D. | Same difference（遊戲內：싱클레어，`StoryData/E302B.json#33`） | 那．那 | 那就那樣（遊戲內：싱클레어，`StoryData/E302B.json#33`） | `StoryData/E302B.json#32`（附註：싱클레어反問時展開） |
| 쓸.없.（另屬同義異寫；近義；提案 35） | 쓸모없는（推測） | 無・用 | 無用な（推測） | B.S. | bullshit？（推測） | 無．用（．噗哈哈） | 無用的（推測） | 2 處：`RPGSystem/rpg-loc-dialogue-floor-b2.json#45/1`、`RPGSystem/rpg-loc-dialogue-floor-b2-b.json#74/1`（附註：後接道具名 파하하（JP フハハ、EN "Bwahahaha!"、TW 噗哈哈）；KR 亦可讀 쓸데없는；EN 以 Worthless B.S. 代整個詞，B.S. 或兼 Bwahahaha 雙關） |
| 쓸. 없.（另屬同義異寫；近義；提案 35） | 쓸모없다（推測） | 不・要 | 不要（未縮） | W.C. | Who cares（遊戲內・他處：탐정로쟈，`StoryData/E614A.json#12`） | 沒．用 | 沒用（未縮） | `StoryData/P11110.json#38`（附註：KR 無展開，亦可讀 쓸데없다；EN W.C. 展開在他處（E614A#12，該處 KR 為 알. 바.）） |
| 어. 떠.（另屬同義異寫；提案 8） | 어중이떠중이（遊戲內：뫼르소，`RPGSystem/rpg-loc-dialogue-floor-b2.json#16/11`） | 烏・衆 | 烏合の衆（遊戲內：뫼르소，`RPGSystem/rpg-loc-dialogue-floor-b2.json#16/11`） | M.B. | mediocre bunch（遊戲內：뫼르소，`RPGSystem/rpg-loc-dialogue-floor-b2.json#16/11`） | 烏．眾 | 烏合之眾（遊戲內：뫼르소，`RPGSystem/rpg-loc-dialogue-floor-b2.json#16/11`） | `RPGSystem/rpg-loc-dialogue-floor-b2.json#16/9`（附註：뫼르소 逐句解讀） |
| 어. 떠.（另屬同義異寫；提案 8） | 어중이떠중이（遊戲內：싱클레어，`StoryData/S812A.json#27`） | う・ど・め | 烏合の衆どもめ（推測） | M.B. | mediocre bunch（遊戲內：싱클레어，`StoryData/S812A.json#27`） | 烏．眾 | 烏合之眾（遊戲內：싱클레어，`StoryData/S812A.json#27`） | `StoryData/S812A.json#26`（附註：JP 싱클레어 作「烏合の衆ども」只對上 う・ど，め 推測為罵語接尾，故列推測） |
| 어. 떠.（另屬同義異寫；提案 8） | 어중이떠중이（遊戲內・他處：싱클레어，`StoryData/S812A.json#27`） | 有・無 | 有象無象（推測） | M.B. | mediocre bunch（遊戲內・他處：싱클레어，`StoryData/S812A.json#27`） | 有．無 | 有象無象？（推測） | `StoryData/P10414.json#75`（附註：KR／EN 展開在他處（S812A#27）；JP 他處以「有象無象」譯 어중이떠중이（S111B#26、P10916#65）；TW 照 JP，中文亦可讀「有眼無珠」） |
| 재. 없.（另屬同義異寫；提案 5、30） | 재미없는（遊戲內：싱클레어，`StoryData/E724B.json#75`） | 退・話 | 退屈な話（遊戲內：싱클레어，`StoryData/E724B.json#75`） | B.W.A.H.A. | Bad, weak, and highly annoying（遊戲內：싱클레어，`StoryData/E724B.json#75`） | 話．沒 | 對話沒意思（遊戲內：싱클레어，`StoryData/E724B.json#75`） | `StoryData/E724B.json#74`（附註：#78 良秀「안.들.은 아직 말하지 않았는데」認可 재.없. 的解讀） |
| 재. 없.（另屬同義異寫；提案 5、30） | 재미없는（遊戲內・他處：싱클레어，`StoryData/E724B.json#75`） | 面・無 | 面白く無い（推測） | B.A.H. | Boring as hell（遊戲內・他處：료슈，`StoryData/S810B.json#92`） | 無．趣 | 無趣（未縮） | `StoryData/P10514.json#63`（附註：KR 展開在他處（E724B#75）；EN B.A.H. 展開在他處（S810B#92 良秀自釋，該處 KR 為 질. 지.）） |
| 재. 없.（另屬同義異寫；提案 5、30） | 재미없는（遊戲內・他處：싱클레어，`StoryData/E724B.json#75`） | 面・無 | 面白く無い（推測） | B.A.H. | Boring as hell（遊戲內・他處：료슈，`StoryData/S810B.json#92`） | 沒．致 | 沒興致（推測） | `StoryData/S951B.json#28`（附註：KR 展開在他處（E724B#75）；EN B.A.H. 展開在他處（S810B#92）） |
| 내.머.（另屬同義異寫；提案 33） | 내려라, 머저리들（遊戲內：료슈，`BattleAnnouncerDlg/Announcer_Ryoshu_42.json#0`） | 降・阿 | 降りろ、阿呆ども（遊戲內：료슈，`BattleAnnouncerDlg/Announcer_Ryoshu_42.json#0`） | U.I.T.G.A.A.N. | Useless idiotic travelers, go and abscond, now（遊戲內：료슈，`BattleAnnouncerDlg/Announcer_Ryoshu_42.json#0`） | 退．蠢 | 退下，蠢貨們（遊戲內：료슈，`BattleAnnouncerDlg/Announcer_Ryoshu_42.json#0`） | `BattleAnnouncerDlg/Announcer_Ryoshu_42.json#0`（附註：同句自行解釋） |
| 내머.（另屬同義異寫；提案 33） | 내리자, 머저리들（遊戲內：료슈，`StoryData/S201B.json#80`） | おあ（不加點） | 下りるぞ、阿呆ども（遊戲內：료슈，`StoryData/S201B.json#80`） | U.I.T.G.A.A.N. | Useless idiotic travelers go and abscond, now（遊戲內：료슈，`StoryData/S201B.json#80`） | 下．蠢 | 該下車了，蠢貨們（遊戲內：료슈，`StoryData/S201B.json#80`） | `StoryData/S201B.json#78`（附註：JP おあ取讀音（下りる、阿呆）） |
| 동. 감.（另屬同義異寫；提案 6、7） | 동감（未縮） | 同・感 | 同感（未縮） | I.A. | I agree（推測） | 同．想 | 同樣這麼想？（推測） | `StoryData/P10607.json#26`（附註：KR／JP 為完整詞逐字拆開；TW 中間字不定） |
| 동. 감.（另屬同義異寫；提案 6、7） | 동감（未縮） | どう・かん | 同感（未縮） | I.A. | I agree（推測） | 同．意 | 同意（未縮） | `StoryData/S514B.json#97`（附註：KR、JP（どう・かん＝同感的讀音）、TW 皆為完整詞逐字拆開） |
| 맘.들.않.（另屬同義異寫；提案 28） | 마음에 들지 않으신대요（遊戲內・他處：旁白，`StoryData/S415B.json#62`） | 気・入・無 | 気に入らない（推測） | D.L.I. | doesn't like it（遊戲內・他處：旁白，`StoryData/S415B.json#62`） | 真．不．爽 | 真不爽（未縮） | `BattleAnnouncerDlg/Announcer_Ryoshu_42.json#21`（附註：KR／EN 展開見 S415B#62（旁白引싱클레어轉述）；JP 無＝ない） |
| 맘.들.않.（另屬同義異寫；提案 28） | 마음에 들지 않으신대요（遊戲內：旁白，`StoryData/S415B.json#62`） | 気・入・な | 気に入らない（遊戲內：旁白，`StoryData/S415B.json#62`） | D.L.I. | doesn't like it（遊戲內：旁白，`StoryData/S415B.json#62`） | 不．怎．滿 | 不怎麼滿意（遊戲內：旁白，`StoryData/S415B.json#62`） | `StoryData/S415B.json#61`（附註：旁白引싱클레어的轉述） |
| 무.상.（另屬同義異寫；待確認；提案 32） | 무엇이든？ 상관없다（推測） | 関・無 | 関係無い（推測） | P.E.R. | 不明 | 沒．問 | 沒問題（推測） | `BattleAnnouncerDlg/Announcer_Ryoshu_42.json#17`（附註：與前句 제.행. 合成 제행무상（諸行無常）雙關；KR 상＝상관 依 E920B#86 用法，무 不確定） |
| 무. 상.（另屬同義異寫；待確認；提案 32） | 무엇이든？ 상관없다（推測） | どう・い | どうでもいい（推測） | T.F. | The Floor（推測） | 無．妨 | 無妨（未縮） | `StoryData/E920B.json#86`（附註：무 也可能是 무슨（상관）；EN 改縮 바닥（the floor），不對應 무. 상.） |
| 믿. 없.（另屬同義異寫；提案 27） | 믿을 수 없다（推測） | 信・無 | 信じられない（推測） | N.W. | No way（遊戲內：료슈，`PersonalityVoiceDlg/Voice_Ryoshu_LiuAsso4_10407.json#23`） | 無．信 | 無法相信（推測） | `PersonalityVoiceDlg/Voice_Ryoshu_LiuAsso4_10407.json#23`（附註：EN 同句隨即寫出 No way；JP 無＝ない） |
| 믿. 없.（另屬同義異寫；提案 27） | 믿을 수 없다（推測） | 信・無 | 信じられない（推測） | N.F.W. | No Fucking Way（推測） | 難．信 | 難以置信（推測） | `StoryData/E725B.json#49`（附註：JP 無＝ない） |
| 버.안（另屬同義異寫；提案 31） | 버스 안（遊戲內・他處：홍루，`StoryData/E512B.json#2`） | バ・中 | バスの中（推測） | O.T.B. | On The Bus（推測） | 蠢．中 | 蠢貨？之中（推測） | `BattleAnnouncerDlg/Announcer_Ryoshu_42.json#26`（附註：KR 展開見 E512B#2；JP バ・中 推為 バスの中（E512B#2「バスの中」展開的是該處的 バ・な）；TW 似把 버 解作蠢（同 Announcer#20 버.놈.→蠢．傢）） |
| 버. 안.（另屬同義異寫；提案 31） | 버스 안（遊戲內：홍루，`StoryData/E512B.json#2`） | バ・な | バスの中（遊戲內：홍루，`StoryData/E512B.json#2`） | T.B. | the bus（遊戲內：홍루，`StoryData/E512B.json#2`） | 巴．內 | 巴士內（遊戲內：홍루，`StoryData/E512B.json#2`） | `StoryData/E512B.json#1`（附註：JP な取「中（なか）」讀音） |
| 삼.대.（提案 36） | 삼조씨… 대답（遊戲內：싱클레어，`StoryData/S430B.json#59`） | サム・こ | サムジョさん…答えろ（遊戲內：싱클레어，`StoryData/S430B.json#59`） | S.A. | Samjo... She wants answers（遊戲內：싱클레어，`StoryData/S430B.json#59`） | 三．答 | 三朝先生……她讓您回答（遊戲內：싱클레어，`StoryData/S430B.json#59`） | `StoryData/S430B.json#58`（附註：JP こ＝答え（こたえ）） |
| 삼.대（提案 36） | 삼조 씨… 이번엔 대가리（遊戲內：싱클레어，`StoryData/S430B.json#62`） | サム・こ | サムジョさん…今度はこうべをカチ割る（遊戲內：싱클레어，`StoryData/S430B.json#62`） | S.A. | Samjo... She says she'll assail you（遊戲內：싱클레어，`StoryData/S430B.json#62`） | 三．打 | 三朝先生……她這次說要打爆您的頭（遊戲內：싱클레어，`StoryData/S430B.json#62`） | `StoryData/S430B.json#61`（附註：雙關：同一 삼.대 第二次改指 대가리；#63 良秀只更正時間（지금 대가리를 깬다）） |
| 상. 없.（另屬同義異寫；提案 4） | 상관없다（推測） | 関・無 | 関係無い（推測） | I.D.C. | I Don't Care（推測） | 無．謂 | 無所謂（推測） | `PersonalityVoiceDlg/Voice_Ryoshu_LiuAsso4_10407.json#2` |
| 상. 없.（另屬同義異寫；提案 4） | 상관없다（推測） | 関・無 | 関係無い（推測） | Y.D.Y. | You Do You（推測） | 不．關 | 不關我的事？（推測） | `StoryData/P10606.json#1`（附註：TW 關之後的部分不確定） |
| 엔.팔.（另屬同義異寫；提案 9） | 엔케팔린（遊戲內・他處：推播通知，`LoginUIText.json#61`） | エン・ファ | エンケファリン（遊戲內・他處：推播通知，`LoginUIText.json#61`） | E.P. | Enkephalin？（推測） | 腦．肽 | 腦啡肽（遊戲內・他處：推播通知，`LoginUIText.json#61`） | `PersonalityVoiceDlg/Voice_Ryoshu_LEGOEcho_10414.json#7`（附註：KR／JP／TW 展開在他處：LoginUIText#61「엔.팔.충.완.(엔케팔린 충전 완료.)」的括號；EN E.P. 疑取 En-／-Phalin 兩段，無展開） |
| 엔. 팔.（另屬同義異寫；提案 9） | 엔케팔린（遊戲內・他處：推播通知，`LoginUIText.json#61`） | エン・ファ | エンケファリン（遊戲內・他處：推播通知，`LoginUIText.json#61`） | E. | Enkephalin（遊戲內・他處：推播通知，`LoginUIText.json#61`） | （無；譯「腦啡肽」） | 腦啡肽（未縮） | `StoryData/P10410.json#76`（附註：展開在他處（LoginUIText#61；EN「E.C.(Enkephalin Charged.)」以 E 代 Enkephalin）；EN「on E」兼 ecstasy 俚語雙關） |

### 3.5 異義與中文不一的核對

JP、EN 一起換寫法的有 8 種（`무.상.`、`그.그.`、`거. 거.`、`버.안`、`쓸.없.`、`재. 없.`、`말. 안. 어.`、`어.이.`），讀過場景後只有 `말. 안. 어.` 有 KR 展開證明意思不同；`그.그.`（Announcer#19）的異義只靠 JP、EN 與 KR 語氣，`어.이.` 只靠 JP、EN、繁中（未確認）；其餘 5 種意思相同或近義（`쓸.없.`；`무.상.` 待確認），是 JP、EN 各自改寫。另外 `모.불.아.위.`、`황. 가.` 只有 JP 換字（EN 字母相同），但劇中的括號展開或更正顯示意思不同，也是異義。

**異義（5 種，繁中不同是對的；`어.이.` 未確認）**

1. `말. 안. 어.`（ES130B#6、#8）：同一場景說兩次，辛克萊分別展開成「말 안 듣는 **어**른 놈들은 모가지를 분질러버리면…」（#10）、「말 안듣는 **어**린아이는 어르고 달래야…」（#11）。JP 言・聞・無・大／言・聞・無・子、EN U.A.／U.C. 都跟著變；繁中 不．聽．大／不．聽．小 正確
2. `모.불.아.위.`（EGO_4#0、#1）：兩條語音各自在括號展開：「모든 것을 불살라서라도, **아**름다움을 위해」／「…**아**라야를 위해」。JP 総・燃・美・為／総・燃・阿・為；EN 都是 O.S.H.A.R.E.，但括號裡的展開不同。繁中 縱．焚．僅．美／縱．焚．僅．阿 要對得上各自的括號，正確（KR 用同一組字做雙關，三種譯文都沒保留）
3. `황. 가.`：E914B#88「어차피, 황. 가.를 찾으면 되찾을 수 있을 거다」讀成 황금가지（繁中 金．枝）：這個讀法來自 JP 黄・枝、EN G.B.，KR 沒有展開。S1013B#10「황. 가. 라도 있어야 하나?」莫梭讀成「황금가지라도」（#11），良秀更正「황. 가. 는 황금가지가 아니라, 황금가죽이다」（#12）：這裡是 황금가죽，而且是誤讀場景。JP お・の（黄金の）、EN G.B.（Golden Bough／Golden Buildingskin）、繁中 黃．金 都是一組字兩讀，正確。初版把繁中記成 黃．金．才、黃．金．指 是抓錯：後面的「才行嗎」「指的」是正文，只是末字後多了一個「．」（提案 37）
4. `그.그.`：Announcer#19「쳇. 그.그. 하지만 그래봤자, 거.거.다.」讀成 그럼 그렇지（繁中 意．中）：這個讀法來自 JP 案・定、EN "knew it. W.A."，加上「쳇.」咂嘴、「하지만 그래봤자」轉折的語氣；KR 沒有展開。E302B#32＝그게 그거（辛克萊「그게 그거라니요?!」#33；JP ど・ど、EN S.D.）；E903A#9 也是 그게 그거（키라 建議「그게 그거, 정도로 줄이면 되잖아」#8；JP に・もの；EN 把키라 的話改成 "say it was anticipated"，所以也寫 W.A.）。Announcer#19 與後兩處異義；後兩處同義，繁中 那．那／都．樣 不一（提案 34）
5. `어.이.`（未確認）：5D205B#3 對魁魁格（JP お・い、EN H.W.、繁中 喂．你），S445B#0 叫李箱（李箱答「…부른 거요?」；JP お、イ、EN E.Y.、繁中 喂，李）。KR 兩處完全相同，S445B#0 也沒有展開；把第二字讀成李箱（이）只靠 JP、EN、繁中三者一起換，KR 沒有根據，所以暫列異義、未確認。若只是逐字加點的「어이」（喂），就是同義，應統一成 喂．你 或 喂．李（提案 21 只改逗號）。能決定的是 KR 的根據：同一場景或別處有人把這個 이 說成「이상」，或良秀用同樣的 `어.X.` 叫別人時第二字都是對象的名字；目前沒有找到（網站資料這行的 JP 是空的，raw 有「お、イ。」，見「修正第 2 輪」）

**中文不一（16 種）**

- 劇中誤讀、應只用一組字：`삼.대.`（S430B#58–63）。辛克萊先讀成「삼조씨… **대**답하시래요…」（#59），良秀再說一次 `삼.대…!`，辛克萊改讀「이번엔 **대**가리를 깨버린다고…!」（#62），良秀：「틀렸어. 지금 대가리를 깬다고 했다.」（#63）。同一組字兩讀：JP サム・こ（こたえ／こうべ）、EN S.A.（answers／assail）都只用一組。繁中 三．答（#58）／三．打（#61）分成兩組，#61 的 三．打 讀不出 #59 的「回答」，是不一。初版寫「刻意不同」有誤（提案 36）
- 同義、劇中有展開：
  - `재. 없.` E724B#74：辛克萊「…재미없는 대화라 안 그래도 들어가 보려고 하셨대요.」（#75）讀對了，連還沒說的 `안.들.` 也猜到（良秀 #78「안.들.은 아직 말하지 않았는데… 성.업.된 건가?」）。所以這裡就是 재미없다，與 P10514#63（無．趣）、S951B#28（沒．致）同義。話．沒 只是取了辛克萊多說的「대화」，無．趣 一樣讀得出（「這場對話很無趣」）。JP 退・話、EN B.W.A.H.A. 也是跟著辛克萊那句改寫。初版「配合辛克萊的解讀，保留」改為不一（提案 30）
  - `내.머.`：Announcer#0 同句展開「내려라, 머저리들」、S201B#80「내리자, 머저리들」，同義；繁中 退．蠢／下．蠢（提案 33）
  - `그.그.` 的 그게 그거：見上「異義」4（提案 34）
- 同義或近義、JP 與 EN 都換了寫法（`버.안` 有 KR 展開確認；`쓸.없.` 近義；`무.상.` 待確認）：
  - `버.안`：E512B#1 下一句鴻璐「여기는 버스 안이에요~」（JP バ・な、EN T.B.，繁中 巴．內）；Announcer#26「그러니 시시비비는 버.안에서 배틀로얄로 가리는 게 좋겠군」，JP バ・中（バスの中）、EN "once we're back O.T.B."（on the bus）。兩處都是 버스 안；Announcer#26 的 蠢．中 是誤譯。初版說「蠢．中 = 버러지 안，刻意不同」有誤（提案 31）
  - `무.상.`（待確認）：Announcer#17「호오. 제.행.이군. 그렇다면 무.상.이다.」（JP 関・無、EN P.E.R.，繁中 沒．問）、E920B#86「무. 상. 그냥 바닥에 던져둬라.」（JP どう・い、EN "throw them on T.F."，T.F. 縮的其實是 the floor；繁中 無．妨）。E920B#86 的上下文是「無所謂，丟地上」；Announcer#17 接在 `제.행.` 後面，可能是「제행무상」（諸行無常）的文字遊戲，說它也是「無所謂」只靠 JP 関・無（EN P.R.O.／P.E.R. 合成 PROPER，看不出意思），KR 沒有展開，所以待確認（提案 32）
  - `쓸.없.`：P11110#38「쓸. 없. 아직도 이해를 포기하지 못했을 줄이야」是 쓸데없다（「多餘、白費」，JP 不・要，繁中 沒．用），RPG 兩處「바닥에 쓸.없.파하하가」是 쓸모없는（「沒用處」，JP 無・用，繁中 無．用）。近義而不完全相同，列為輕微的不一（提案 35）
- 同義、只有 JP 或 EN 換寫法（初版已有提案）：`모. 분.`（E406B#16 脖．扭．豬．閉，提案 10）、`관. 없.`（2、3）、`어. 떠.`（8）、`맘.들.않.`（28）、`상. 없.`（4）、`믿. 없.`（27）、`엔.팔.`（9）、`동. 감.`（6、7）
- JP、EN 都一致、只有繁中不同：`시. 대.`（S1027B#18 鐘．頭，提案 1）、`삼.대.`（上述）

### 3.6 良秀人格的技能、被動名稱（不在 ryoshu.md 掛載範圍，供對照）

| KR | JP | EN | 既有繁中 | 出處 |
|---|---|---|---|---|
| 에.순 | エ・循 | E.C. | 能．循 | `Skills_personality-04｜#0` |
| 차.마.공.절 | 次・魔による空・切（P11110） | D.D.E.D.R. | 次．魔．空．裂 | `Skills_personality-04｜#2`（劇情 P11110#74 拆成 `차. 마.`、`공. 절.`） |
| 타.무 | タ・む | T.N. | 目．無 | `Skills_personality-04｜#5`（語音 LCCB#17 有展開「타겟 무력화」） |
| 대.박 | あ・われ | O.O.F. | 頭．裂 | `Skills_personality-04｜#6` |
| 모.분 | く・へ | S.Y.N.C. | 迎客之道第2式 脖．扭 | `Skills_personality-04｜#14` |
| 거.대 / 해.바 | く・ヅ／しゃ・こ | S.H./S.F. | 蜘．頭 / 骷．頭 | `Skills_personality-04｜#21` |
| 예.감.시. | 芸・観・時 | A.A.T. | 觀．藝．時 | `Skills_personality-04｜#42` |
| 반.갈 | はん・われ | C. I. H. | 刀．斷 | `Passives｜#10`（1040201） |
| 담.탐 | ヤ・切 | A S.B. | 菸．休×2、煙．休×1 | `Passives｜#171`、`#172`、`#173`（1040601/02/11） |
| 제.지 / 조.감 | 制・支／助・感 | N.O./The A.I. | 鎮．支／助．感 | `Passives｜#283`、`#284` |
| 군.가.곁.좋（附括號展開） | 君・可・傍・良 | I.L.T.L.T. | 君．可．保．好 | `Passives｜#478`（1041302） |
| 요.준、재.사、신.재 발견 | りょ・し、しょ・が、しん・ざい | P.C.、I.H.、F.I. | 烹．備、獵．材、新．材發現 | `Skills｜#184`、`#185`、`#187`（104040x） |
| 료.고.파（料理店名） | りょ・ミ・パ | R.B. | 良．派 | `Passives｜#105`、`UnitKeyword｜#34`、`Personality_Get_Condition｜#64–67`、`StoryTheaterPersonality｜#109/#112` 等 |
| 넌.버.실.（附下一行展開「넌 치프 버틀러 실격이다」） | お・バ・し | S.A.D. | 你．首．失 | `BattleSpeechBubbleDlg｜#2`（battle_speechbubble_10408_1） |
| 모. 뿐. | く・ひぇ | S.Y.N.C. | 脖．扭 | `BattleSpeechBubbleDlg｜#531` |
| 닭. 모. 분. | とり・く・へ | S.C.N.C. | 總．脖．扭 | `Skills_personality-x1p1c1｜#14` |
| 모.분.조.도 | く・へ・し・ぶ | TRANCHES | 脖．扭．豬．閉 | `IntroductionPreset｜#4` |
| 우.미.다（活動名） | う・み・ダ | S.E.A. | 去．海．邊（10 個 UI 單位一致） | `TutorialMainUIText｜#370` 等 |

## 4. KR 怎麼構成縮略語

### 4.1 規則：每個語節（詞）取第一個音節

- 劇中有展開的 78 種（88 處）都能對上「每個語節取首音節」；另人工看過約 25 行「~라는 뜻이다／의미」，只有 `반. 반. 위.` 把一個詞 `반반` 兩個音節都留下（良秀自己展開「반반보다 위」，E912B#108）
- 展開例：
  - `모.분.조.도.` = **모**가지를 **분**질러야 **조**용해지지, **도**야지（S101B#46；Announcer_Ryoshu#28 同句展開）
  - `남.손.다.분` = **남**은 **손**가락을 **다** **분**질러（辛克萊，2D102B#18）
  - `알.단.이.예.` = **알**겠나 **단**테? **이**것이 **예**술이다（同句括號，Voice_Seven6#7）
  - `예.실.인.길.` = **예**술은 **실**패했지만 **인**생은 **길**다（Voice_LCB#20）
  - `질. 지.` = **질**리고 **지**루하다（S810B#92）
  - `뭐.아` = **뭐**가 **아**니라는 거지?（S947B#220）
  - `뭔. 말. 모.` = **뭔** **말**인지 **모**르겠다（E912B#55）
  - `별. 이. 소. 다.` = **별** **이**상한 **소**리를 **다** 듣겠군（P10416#58）
- 助詞、語尾、`하다`、`이다` 都不取；依存名詞、輔助動詞、句末述語也常整個略去（`모.불.아.위.` 略 것을、`남.손.다.분` 略 버리겠다、`별. 이. 소. 다.` 略 듣겠군）；音節數（第 2 輪修正 3 處邊界後重算）：2 音節 291 處、3 音節 52、4 音節 42、5 音節 3（`빨. 말. 않. 대. 박.` S514B#4，以及 `만.단.지.예.` 後面接 `다.` 的 2 處）、1 音節 1（`살.`，Announcer#7，呼應 `닥.돌.죽.살.`）

### 4.2 加點的位置

- 每個音節後加「.」：379／389 處
- 空格（全部加點的縮略語合計，不是 `모.분.` 一個）：音節間有空格的 255 處、沒有空格的 124 處；同一縮略語兩種都有（`모.분.`／`모. 분.`，`시.대.`／`시. 대.`，`알.못.`／`알. 못.`），看不出意思差別
- 末字的點：344／379 有；省略時多半是後面直接接助詞或語尾（`버.안에서` Announcer#26、`대.박의` S513B#15、`모.썰이다` E302B#14、`알. 못아` E030X#94）、問號（`시. 대?` S1004B#16、S1027B#18、`모. 시?` E920B#73）或刪節號（`모. 분…` S813B#12）
- 縮略語當名詞用：後面接助詞（`히.클.과`、`싱.어.에`、`자.정.이`、`평.실.은`），或接 `하다`（`모.분.해주지`、`모. 분. 할 시간`、`순.차.하면`）、`이다`（`모. 분. 이다`、`만.단.지.예.다.`）
- 早期不加點：第一章 S101B#45 `모분조도.`、第二章 S201B#78 `내머.`（兩者下一句都由良秀自己展開）；第二章地牢 2D102B#16 `남.손.다.분` 起改為加點

### 4.3 其他構成方式

- 截取一個詞或人名的音節（不是每個詞首；以下展開都是推定，劇中沒有說出）：`히.클.`（히스클리프，3 處）、`싱.어.`（싱클레어，首尾音節，Announcer#1）、`엔.팔.`（엔케팔린）、`알.우.`（알라우네）、`어.떠.`（어중이떠중이，3 處）、`사.직.`（사무직）、`마. 킹.`（마네킹）、`하. 아.`（하루아침，Voice_Hongyuan#1 後面自己說出「하루아침」）、`감. 합.`（감사합니다，E003I3#12）
- 人名列舉：`그.로.싱.이.`（그레고르、로쟈、싱클레어、이상）、`이.히.돈.홍.`（이스마엘、히스클리프、돈키호테、홍루），Announcer#0
- 把短詞逐字加點（沒有縮短）：`젠. 장.`、`걸. 작.`、`동. 감.`、`만. 족.`、`오. 만.`、`환. 장.`、`허. 탕.`、`허. 접.`、`직. 진.`、`어.이.`；也會刻意撞上現成的詞做雙關：`대. 박.`＝대가리 박살（S504B#111 旁白展開）、`재.수.`＝재수없는 수염（E605B#95）、`날. 치. 기.`＝날렵하고 치명적인 기술（辛克萊，RPG floor-b2#103/4）、`쓸. 쓸.`＝쓸모없고 쓸데없는（P10414#51）、`단.테.`（Voice_Yurodivy#4，後句「시계… 너를 부른 게 아니다」）、`명.단.`（「명단에 있는 놈들 모두 명.단.이다」）
- 把別人剛說的話縮掉：`시. 대.` ← 히스클리프「시계 대가리!」（RPG common-a1c10p1#50/0）；`부. 일.` ← 호엔하임「부탁할 일」（E901B#101）；`구.아.구.` ← 오티스「구인회의 아홉 구율」（4D104B#7）；`도.별.` ← 「도시의 별」（E713B#67）；`작. 형.` ← 「작은 형님」（S510A#3）；`림.컴.수.` ← 「림버스 컴퍼니의 수치」（S701B#64）；`로드, 엔러쉬` ← 이스마엘（S111B#27）
- 整句只有縮略語：86／356 行（容許一個感嘆詞時 94 行；初版寫的 110 有誤）

## 5. 繁中怎麼譯（既有）

### 5.1 原則：縮中文，不照 KR 音節

- 繁中是把**完整意思譯成中文，再每個詞取一個代表字**（多半首字），可照中文語序重排：`꾸. 말.` → 不．磨、`사.분.`（사지를 분질러）→ 折．肢；固定寫法 `모.분.` → 脖．扭（20／21）則保留 KR 的順序；`남.손.다.분`（剩餘的手指盡數折斷）→ 餘．指．盡．折；`요. 넘. 맛.`（菜做得超讚）→ 菜．超．讚；`꾸. 말.`（꾸물거리지 말아야）→ 不．磨（「不該磨蹭」，否定詞移到前面）；`실. 분.` → 分．線（「分解成線」）
- 人名取譯名中可辨認的字（不一定是首字：`히.클.` → 希．克、`싱.어.` → 辛．萊）：`그.로.싱.이.` → 格．羅．辛．李、`이.히.돈.홍.` → 以．希．唐．鴻、`히.클.` → 希．克
- 字數：與 KR 音節數相同 341／380 處（可配對的；第 2 輪以第 3 節逐處核對的片段重算，初版 334／385），多 26、少 13（例：`알. 바.` → 管．他．的、`취. 존.` → 人．各．愛、`실. 크. 다.` → 失．望）
- 分隔：全形「．」353／356 行；沒有 `・`、`·`、半形 `.`；末字後多半不加「．」（KR 的末點變成句號或省略）；例外 8 處是後面直接接語尾或名詞時留了點：S1013B#10 黃．金．才行嗎、#12 黃．金．指的、Voice_Contem#13 腳．多．的、RPG floor-b2-b#30/1 全．齡．的、#41/3 人．模．嗎、E003I2#121 機．轉．是啥、RPG floor-b2#45/1 與 b2-b#74/1 無．用．噗哈哈（初版寫「一律」有誤；前兩處會讀成三字，見提案 37）。沒有「．」的 3 行：S445B#0「喂，李。」（點掉了，見提案 21）、S951B#24（KR 不加點）、Voice_KumoPinky#20（KR 不加點）
- KR 不加點時：良秀自己造的 6 處繁中**全部加了點**（S101B#45 脖．扭．豬．閉、S201B#78 下．蠢、P10402#1 沒．因、P10402#12 劍．創．藝．極、S111B#28 腦．夢、腦．潮）；只有借用阿賴耶的流行語不加（品死、不悶不煩／挺有意思的、珍愛）。初版「KR 沒加點的不加點」的寫法有誤，已改

### 5.2 JP、EN

- JP：縮的是 JP 譯文，每個詞取一個假名或漢字、用「・」隔開：く・へ（首をへし折る）、時・ヅ（時計ヅラ）、何・言・不、総・燃・美・為；有「・」的 342／353 行。偶爾保留 KR 的音做雙關（団・テ = 단테、サム・こ = 삼조）
- 異寫（第 2 輪，見 3.2–3.5）：出現 ≥ 2 次的 50 種中，JP 換寫法的 17 種、EN 換縮寫的 15 種（只差點的不算），兩者都換的 8 種。多半是同義改寫：EN 另造縮寫（`모. 분.` S.Y.N.C.×17 之外還有 S.T.N.C.、O.O.F.、C.I.H.、S.H.N.C.；`알. 못.` I.F.／T.N.W.I.M.；`알. 바.` N.O.M.B.／W.C.），JP 換同義詞（`노. 잼.` つ・マン・ネ／だ・る；`아.닥.` 口・閉／だ・ま・れ）。兩者都換的 8 種裡，只有 `말. 안. 어.` 有 KR 展開證明意思不同；`그.그.` 靠 JP、EN 加上 KR 語氣，`어.이.` 未確認；`거. 거.` E507A#43（に・よ、S.S.E.，展開仍是「거기서 거기」）、`재. 없.` E724B#74、`버.안` 意思相同，`쓸.없.` 近義，`무.상.` 待確認。所以 JP、EN 換寫法不能單獨當成繁中換字的依據，換字要有 KR 的根據
- EN：另造英文句子，縮寫常拼成一個英文字（S.Y.N.C.、SANGRIA、TRANCHES、PACTS、GRASPED、STAB、O.O.F.、F.U.B.A.R.）；有縮寫 351／353 行，其中帶點 324、不帶點 27。EN 也會在 KR 沒有縮寫的地方自己加（P10606#5「B.S.」、E923I#104「G.P.A.」、S940B#39「Nodding M.H.」），所以只能當「這裡是縮略語」的輔助，不能當意思的依據
- 三種語言都把良秀與阿賴耶分開：JP 良秀用「・」，阿賴耶用片假名流行語（センシ、マジ切、全略）；EN 良秀多半帶點，阿賴耶用不帶點的網路縮寫（GOAT、SMH、ILY、IDK）

### 5.3 同場景展開時是否一致

88 處有展開。逐一比對，繁中的縮略語與展開句的譯文大多對得上，而且雙關處理得很細：
- `시. 비.`：莫梭「時鐘，挪開」、良秀「時間被挪移了嗎」→ 時．挪 兩種都讀得出（RPG floor-b2#105/6–9）
- `임. 동.`：鴻璐「臨時同盟」、良秀「臨時同行」→ 臨．同（E908B#110–112）
- `같. 취.`：良秀「同等對待」、唐吉訶德誤讀成「同樣……期待」→ 同．待（E902B#64–65）
- `숨. 죽.`：「隱入竹林」／「隱藏的竹簡」→ 隱．竹（P10411#0–3；第一種讀法為了配合改譯）
- `삼.대.`：辛克萊先讀成「대답」（#59）、再讀成「대가리를 깨버린다」（#62），良秀說「틀렸어. 지금 대가리를 깬다고 했다」（#63）。這是同一組字兩讀，JP サム・こ、EN S.A. 都只用一組；繁中 三．答／三．打 分成兩組，#61 的 三．打 讀不出「回答」。初版寫「刻意不同」有誤，應照 `시. 비.` 用一組字（提案 36）
- `안.들.` → 本．回 配合辛克萊「本來打算回去」（E724B#75–78），對得上。`재. 없.` → 話．沒 則不必要：辛克萊讀對了（#78 良秀認可），這裡就是 재미없다，與 P10514#63 無．趣 同義，無．趣 也讀得出「這場對話很無趣」。初版當成配合解讀而保留，改為不一（提案 30）
- `말. 안. 어.`：同一場景兩次，辛克萊分別展開成「말 안 듣는 어른」「말 안듣는 어린아이」（ES130B#10–11）→ 不．聽．大／不．聽．小，JP 大／子、EN U.A.／U.C. 也跟著變。這才是意思不同、該換字的例子

對不上的見第 9 節提案 11–17、30、36。

## 6. 第九章：解釋縮略語的場景

第九章主線（S9xx）裡良秀（含過去的本人）的縮略語只有 17 處（S902B#139 `버. 괜.`、S912B#90 `뭘. 원. 알.`、S913B#19 `불. 시. 낭`、S918B#51 `과.악.만.연.`、S918B#90 `단. 예.`、S926A#80 `시. 불.`、S926B#57 `새. 자.`、S926B#67 `머.터.`、S929A#4 `시.대.`、S929A#12 `혀. 잘.`、S947B#218/#236/#314、S949A#124、S951B#24×2（不加點）、S951B#28）；章後的間章 E901B–E914B（阿賴耶已成為刀鞘，E912B#58）有 46 處。第九章的重點是**縮略語的來歷與良秀、阿賴耶兩種縮法的對照**：

1. **S940B#10–42（回憶，阿賴耶教流行語）**
   - #15–16 阿賴耶逐字強調 `란.도.셀!`、`초.등.학.생!`（不是縮略語；JP ラ・ン・ド・セ・ル、EN ran-do-seru；繁中 雙．肩．背．包！、小．學．生！）
   - #19 `세젤맛` → #23 阿賴耶自己展開「세상에서 제일 맛있다」（繁中 世一好吃 / 「世界第一好吃」）
   - #24 良秀（과거료슈3）「…별걸 다 줄이는군.」→ #25 阿賴耶「그건 별다줄이라고 말하면 돼.」（繁中「什麼都要縮寫啊」→「全縮」）
   - #29 良秀：流行語總有一天會風化；#31 阿賴耶「그리구 단축의 미학이 중요하다면서.」——說明良秀當時已經講究「縮短的美學」
   - #32 `감다뒤` → #33 良秀反問 → #34 阿賴耶「감이… 다… 뒤졌다아… 자! 그래서 감다뒤!」（逐字拉長地展開）；#38 再用；#36 `오키`
   - #41–42 現在的良秀「모르겠으니까, 다시 한번…」
2. **S947B#195–237（回憶，良秀用加點縮法、阿賴耶糾正）**
   - #216–217 阿賴耶「그게 아니라니까. 그렇게 줄이는 거 아니라고.」→ #218 良秀 `뭐.아?` → #220 良秀自己展開「뭐가 아니라는 거지? 라는 뜻이다.」（繁中 有．差？／有什麼差別？的意思）
   - #224 阿賴耶示範 `완소`（「엄마는 나에게 있어서 완소야」，繁中「對我來說，媽最愛！」）；#228 良秀「수박 먹어, 아라야.」→ #229 阿賴耶「이럴 땐 수먹아! 라고 하는 거야. 흐음, 요건 아닌가?」（她自己也覺得造得不好）；#230 `암툰`
   - #236 良秀 `엄.예.`（JP す・ご・き、EN M. thinks it's P.；繁中 很．漂）→ #237 阿賴耶「아니이…!」
3. **S919B#0–5**：同一段「그렇게 줄이는 거 아니라고」的閃回
4. **S949A#119–128（高潮）**：阿賴耶「엄마는 나에게 있어서 완소야.」(#121) → 良秀（료슈9장다침）「아라야.」「`너. 내. 소. 딸.`」(#123–124，推定 너는 내 소중한 딸，劇中未展開；`소` 接 `완소` 的 소중) → 阿賴耶（회상）「아이참… 그렇게 줄이는 거 아니라니까… 마지막으로 한 번 더 알려줄게.」(#125–128)。良秀用自己的加點縮法回答女兒的流行語
5. **S951B#20–30（章末）**：良秀「안심안지… 아니, 그럭저럭 유잼이었다」(#24，不加點，阿賴耶式)；辛克萊「안 심심하고 안 지루…」(#25)、「이제는 자연스럽게 말 줄이시네요」(#26)；良秀 `재. 없.` (#28) →「이게 좋군. 그냥 원래대로 간다.」(#30)。同一場景兩種縮法並列，是區分兩者最直接的證據
6. **E9xx 間章**
   - E912B#28–56（便當回憶）：`요. 넘. 맛.` 同句展開「요리가 너무 맛있다는 뜻」(#31)；`알.못.` (#39)；過去的良秀 `거. 거.` (#51)、`뭔. 말. 모.` (#53) → 阿賴耶「또 이상하게 줄였지?」(#54) → 展開 (#55) → 阿賴耶「진짜 엄마, 별다줄이야」(#56)
   - E903A#3–10：키라問 `거. 거.가 뭔데?` → 良秀「거기서 거기, 라는 뜻이지」→ 키라「그게 그거, 정도로 줄이면 되잖아」→ 良秀「그건 그. 그.다.」（連別人的建議也要加點）
   - E914B#122–125：키라學著說 `담. 또. 봐.` → 良秀「그건 담. 봐.라고 줄이는 거다」（糾正別人的縮法；繁中 下．次．再．見 → 再．見）
   - E912B#106–108：`반. 반. 위.` → 키라「반의 반보다 위?」→ 良秀「아니. 반반보다 위… 라는 뜻이다」
   - E908B#110–112（`임. 동.`）、E902B#62–72（`큰. 형.`、`같. 취.` 唐吉訶德誤讀、`순. 협.`／`머. 절.` 良秀自己展開）、E907B#53–55（`바. 머.` = 바가지 머리，키라問）、E914B#43「싱. 번. 보다 나은 구석이 있군」（說別人解讀得比「辛克萊翻譯」好）

## 7. 阿賴耶（아라야）

### 7.1 model 鍵值

| model | ScenarioModelCodes name／nickName | 行數 | 檔案 |
|---|---|---|---|
| 료슈딸 | ???／??? | 43 | S901B 3、S919B 6、S929A 5、S931B 8（teller 아라야）、S940B 21 |
| 료슈딸2 | 아라야／거미집 | 104 | S941B 1、S947B 33、S948A 38、S949A 25、E906B 2、E912B 5 |
| 료슈딸연구실 | 아라야／??? | 24 | S941B |
| 아라야공백 | ■■■ | 5 | E906B 2、E912B 3 |
| 아라야10대、아라야20대 | 아라야／거미집 | 6、5 | S949A |
| 아라야정장、아라야정장2 | 지혜성／거미집 | 2、24 | S948A、S948B、S949A |

- 另：`소지아비`（小指父輩，name 지혜성）在 E910A#0–8、S948A#100–118 有 12 行 teller 是 아라야，E910A#2 검지아비 說「다른 제자들에게는 지혜성의 말투로 대하는 게 좋을 것 같구나」；看來成年的阿賴耶就是地慧星。這些行沒有縮略語，也沒逐一確認所有 `소지아비*`（273 行）都是她，所以沒有放進 araya.md，留待人工決定是否列為 `models_other_self`
- 沒有阿賴耶的語音檔（檔名找不到 Araya）

### 7.2 阿賴耶的縮略語

| KR | 劇中的完整說法 | JP | EN | 既有繁中 | 出處 |
|---|---|---|---|---|---|
| 세젤맛 | 세상에서 제일 맛있다（阿賴耶，S940B#23） | せかいちうま | GOAT（Greatest Of All Time） | 世一好吃×2 | S940B#19、#23 |
| 별다줄 | 별걸 다 줄이는군（良秀，S940B#24） | 全略×2 | IMO?／IDK…SMH | 全縮；什麼都要縮寫（未縮） | S940B#25、E912B#56 |
| 감다뒤 | 감이… 다… 뒤졌다아（阿賴耶，S940B#34） | センシ | SMH | 品死×3（另良秀 S940B#33、Voice_KumoPinky#20 也是 品死） | S940B#32、#34、#38 |
| 완소 | 未展開（완전 소중） | マジ<ruby=せつ>切</ruby> | ILY | 媽最愛×2 | S947B#224、S949A#121 |
| 수먹아 | 수박 먹어, 아라야（良秀，S947B#228） | スくア | HSWA | 吃西耶 | S947B#229 |
| 오키 | — | オケ | Okie | OK | S940B#36 |
| 암툰 | （아무튼） | ま、とにかく | Anyway | 總之 | S947B#230 |
| 당근 | （당연하지） | — | — | 當然 | S949A#98 |
| 란.도.셀! / 초.등.학.생!（逐字強調，不是縮略語） | — | ラ・ン・ド・セ・ル／しょう・がく・せい | ran-do-seru／Ele-men-tary | 雙．肩．背．包！／小．學．生！ | S940B#15、#16 |

- 說縮法的台詞：S940B#31「단축의 미학이 중요하다면서」、#35「다른 것들은 이렇게 줄여서 읽으면 돼」；糾正良秀：S919B#1、S947B#217、S949A#127「그렇게 줄이는 거 아니라고／아니라니까」、E912B#54「또 이상하게 줄였지?」
- 名字：她不縮名字，反而把良秀拉長成 `료-슈`（S901B#9、S931B#31、S949A#91、#100）

### 7.3 與良秀的差異（依資料）

| 面向 | 良秀 | 阿賴耶 |
|---|---|---|
| 分隔 | 每個音節後加「.」376／386 | 一個詞、不加點 10／10（세젤맛、별다줄、감다뒤、완소、수먹아） |
| 來源 | 自己造的，約 285 種、多數只用一次 | 學校學來的現成流行語（S940B#27「학교에서 배운 거」）；自造的只有 `수먹아`，她自己也懷疑 |
| 取哪個音節 | 每個語節首音節，也截人名、長詞 | 也是每個語節首音節（세상에서／젤／맛），但用流行語的慣用形（제일 → 젤） |
| 用法 | 常整句只有縮略語，句末加點 | 當一個詞放在句子裡（「세젤맛 수박이야」「완소야」），接一般語尾 |
| 人名 | 縮人名（`히.클.`、`그.로.싱.이.`），叫人用縮略的綽號（`시.대.`、`벌.양.`） | 不縮；把「료슈」拉長成 `료-슈` |
| 句末 | 縮略語後常是句號或接 `이다` | 兒童口語、撒嬌拼字（갠찮어、그리구、이쁘자나） |
| JP／EN | ・／帶點的縮寫 | 片假名流行語／不帶點的網路縮寫 |

### 7.4 araya.md

（第 1 輪：`료슈딸` 已加入 `models`，`TestCharacterByModel` 改為 `{"료슈딸", "araya", false}` 並以 `{"그렉료슈", "", false}` 保留不屬於任何角色的例子；以下為初版說明。）

建立了。理由：model 鍵值與良秀不同；有 10 處流行語縮寫加 6 行談縮法，集中在第九章關鍵場景，而且既有繁中已有該照用的寫法（品死、世一好吃、媽最愛）與一處不一致（별다줄）。
限制：`료슈딸`（幼年，S940B 的教學場景就在這個鍵值）沒有列入 models，因為 `internal/rules/rules_test.go` 的 `TestCharacterByModel` 以 `{"료슈딸", "", false}`（"Ryoshu's daughter"）為「不屬於任何角色」的例子，列入會讓 `go test` 失敗，而本任務只提交兩個指引檔。目前 `료슈딸` 出現的 5 個檔都有良秀說話，batch 會掛 ryoshu.md，ryoshu.md 的「與阿賴耶的縮略語不同」仍會送到；但單獨預覽 `S940B#25` 時沒有角色指引。建議另案把該測試換成別的例子後，把 `료슈딸` 加進 araya.md 的 `models`。

## 8. 其他角色模仿或評論

- **辛克萊**：最常替良秀翻譯（29 處展開，如 2D102B#18、E302B#164、S913B#21、E063X#72）；也學著用：RPG floor-3#63/4「료슈 씨라면… 이럴 때 진. 지. 라고 하셨을 것 같아요」（繁中 平．淡）、S1004B#332 `기.완.`、S532B#97 `대. 박.`；S951B#26 評論她「자연스럽게 말 줄이시네요」
- **莫梭**：7 處展開（RPG floor-b2#17/9、#33/30、#51/2、#105/7、b2-b#101/5），還引述良秀在巴士上說過的 `신. 실. 분.`（RPG floor-b2#51/4）；E406B#17 糾正良秀「아니, 여기서는 다. 분.이 맞다」
- **키라（뉴키라）**：E903A#6、#8 問意思並建議縮成「그게 그거」；E907B#54「바. 머.는 또 뭔데…」；E914B#122 自己說 `담. 또. 봐.`（被良秀糾正）
- **鴻璐**：PC02B#52「모. 분. 도 하면 안 된답니다」；E001X（換身體事件）中 model `홍루` 的 `다. 뒤.`、`시. 대.`、`일. 격. 필.`、`모. 해. 모. 돌.` 是良秀的意識（E001X#203 旁白：抽第 11 根菸、找大刀）
- **格里高爾**：P11204#26、#30 店名 `료.고.파`（「료슈의 고기파이가 아니라 료.고.파」）；E302B#99 解讀 `벌.털`（벌레 털보）
- **但丁**：S302B#15「닥.돌.죽.살 중에 내가 감당해야 할 고통」；RPG floor-b2-b#140/6「없. 어. 를 아직 말하지도 않은 것 같지만…」
- **希斯克利夫**：S1004B#15 也叫但丁 `시. 대?`；**唐吉訶德** E901B#122 引述 `부. 일.`、E902B#65 誤讀 `같. 취.`；**羅佳** E509B#132 `히. 스.`；**維吉里烏斯** E001X#351 `하. 차.`；**李箱** S111B#30 批評「그러한 잉글리쉬는 본래의 뜻을 가물하게 만들지 않겠소?」；**賈惜春** ES130B#2「너희 관리자 말고도 통역이 필요한 사람이 더 있었구나」
- 旁白：E908B#46「료슈의 줄임말에 질색하는 표정을 짓는 키라와 엉터리 풀이를 내어주는 돈키호테」；P10404#22「말을 줄이는 버릇은 아이의 확고한 취향인가봐」；P10416#14「용소들로 줄이는 난데 없는 단축이 너무나 당혹스럽다」
- 良秀自己的說法：「단축의 예술」（Announcer#32）、「단축의 묘미」（ES130A#18）、「단축과 생략은 뉘앙스가 전혀 다르다」（S513B#18）、「아무나 할 수 없는 건, 덜어내는 것. 단축하는 것이지」（E912B#155）

## 9. 不一致與提案（不改譯文，供人工決定）

**採用狀況（2026-11-11 使用者決定）**：採用 1、6–15、17–21、24、25、31、37，共 20 條，照建議寫成人工調整 `rules/adjustments/2026-11-11-ryoshu-proposals.yaml`（24 個單位；提案 24 連同 S940B#24 一起改）。其餘 2–5、16、22、23、27–30、32–36 維持不動，待決定（26 已撤回）。

| # | 單位 | KR | 既有繁中 | 建議 | 理由 |
|---|---|---|---|---|---|
| 1 | `StoryData/S1027B.json｜#18`（claude） | 불만 있나? 시. 대? | 有意見嗎？鐘．頭？ | 有意見嗎？時．頭？ | `시.대.` 11／12 寫 時．頭；詞彙表禁用 鐘．頭，但條目 `시.대.`／`시. 대.` 對不上 `시. 대?`，沒被擋下（`lct validate`：`시. 대?` 無警告，`시. 대.` 報 forbidden-term） |
| 2 | `Voice_Ryoshu_LEGO_10412｜#4` | 관. 없. 너희들끼리… | 無．趣。… | 沒．興。… | `관.없.`（관심 없다）既有 無．趣／沒．興／無．謂 三種；無．趣 又與 `재.없.` 重複 |
| 3 | `E605B｜#31` | 관. 없. 본론을 말해라. | 無．謂。說重點。 | 沒．興。說重點。 | 同上 |
| 4 | `P10606｜#1` | 상. 없. 오늘은 변덕으로… | 不．關。… | 無．謂。… | `상.없.`（상관없다）Voice_LiuAsso4#2 寫 無．謂 |
| 5 | `S951B｜#28`（第 2 輪修改理由） | 재. 없. | 沒．致。 | 無．趣。 | `재.없.` P10514#63 寫 無．趣；E724B#74 的 話．沒 也是同義，見提案 30（初版說它配合辛克萊的解讀、應保留，有誤） |
| 6 | `P10607｜#26` | 동. 감. 이런 건… | 同．想。… | 同．感。… | `동감` 本身就是「同感」；另一處寫 同．意；JP 同・感 |
| 7 | `S514B｜#97` | …동. 감. | ……同．意。 | ……同．感。 | 同上 |
| 8 | `P10414｜#75` | 잘 봐둬라. 어. 떠. | 給我看好了。有．無。 | 給我看好了。烏．眾。 | `어.떠.`（어중이떠중이）另 2 處寫 烏．眾；有．無 是照 JP「有象無象」 |
| 9 | `P10410｜#76` | 초. 깡.이 엔. 팔.에 절여져 한 말은 무시한다. | …泡在腦啡肽裡的腦．醃．罐說的話。 | …泡在腦．肽裡的腦．醃．罐說的話。 | 縮略語遺失；Voice_LEGOEcho#7 `엔.팔.` → 腦．肽 |
| 10 | `E406B｜#16` | 모. 분. | 脖．扭．豬．閉。 | 脖．扭。 | KR 只有 `모.분.`，譯文多了 豬．閉 |
| 11 | `E406B｜#17`（莫梭） | 아니, 여기서는 다. 분.이 맞다. | 不，這裡說「脖．扭．蟹．閉」更好。 | 不，這裡該說盡．折。 | `다. 분.`（다 분질러）＝ 2D102B 餘．指．盡．折、E910B#56 盡．折 |
| 12 | `S210B｜#5` | 모가지를 분질러버린다는 뜻 이다. | 折斷你脖子的意思。 | 扭斷你脖子的意思。 | 展開要對上 脖．扭（S101B#46、Announcer#28 都寫「扭斷」） |
| 13 | `ES130B｜#10`（辛克萊） | 말 안 듣는 어른 놈들은 모가지를 분질러버리면 그만이지만… | 不聽話的大人，折斷脖子便可…… | 不聽話的大人，扭斷脖子便可…… | 同上（#6 是 不．聽．大，脖．扭） |
| 14 | `P10413｜#19` | 모… 그, 그거지? 모가지를 분질러 버린다는… | 折……是、是那個嗎？將脖子折斷…… | 脖……是、是那個嗎？將脖子扭斷…… | 結巴的第一字要是縮略語的第一字（#18 脖．扭） |
| 15 | `Voice_Ryoshu_Seven6_10402｜#7`、`#11` | 알.단.이.예.(알겠나 단테? 이것이 예술이다.) | 明．了．這．術。（明白了嗎但丁？這就是藝術。） | 明．但．這．藝。（括號不動） | 漏了「但」；同檔 #18 `알.단.잘.부` 寫 知．但．請．關 |
| 16 | `ES130A｜#18` | 일. 말. 로 끝날 것을… | 要一．完…… | 用總．說就能了結的事…… | #20 展開「일단 말해봐라」譯「總之先說吧」，一．完 對不上 |
| 17 | `E402A｜#16`、`#17` | 저. 뭐. ／저것들은 뭐냐고 하시는 것 같네요… | 東．說。／她應該是「那些東西在說什麼」的意思…… | 那．啥。／她應該是在問「那些東西是啥」…… | KR 是「那些東西是什麼」，「在說什麼」是誤譯 |
| 18 | `Passives.json｜#173`（1040611） | 담.탐 | 煙．休 | 菸．休 | 同名被動 #171、#172 寫 菸．休 |
| 19 | `Announcer_Ryoshu_42｜#7` | 닥.돌.죽. 다음은 시계의 고통과 함께 살.이겠군. | 閉．突．死，下一步就是伴隨著時鐘的痛苦復活了。 | 閉．突．死。接下來就是伴著時鐘之痛的生．了吧。 | `살.` 遺失；S302B#11 `닥.돌.죽.살.` → 閉．突．死．生 |
| 20 | `P11110｜#74` | 차. 마.를 통한 공. 절.은 나의 피를 머금었을 때야 예술적인… | 次．魔．空．裂只有在浸透我的血液時才是藝術…… | 透過次．魔的空．裂，只有在浸透我的血液時才是藝術…… | 兩個縮略語被併成一個（技能名本身是 次．魔．空．裂） |
| 21 | `S445B｜#0` | 어.이. | 喂，李。 | 喂．李。 | KR 加點，譯文改成逗號 |
| 22 | `Announcer_Ryoshu_42｜#11`（現已掛 ryoshu.md） | 아라야시키는 내게 있어 완소인 검이다. … 네놈들도 완.소. 확정이다. | 阿賴耶識對我來說是珍愛的刀。…你們也確定真．哀了。 | 阿賴耶識對我來說是最愛的刀。…你們也確定最．哀了。 | 「내게 있어 완소」照搬阿賴耶的「나에게 있어서 완소」（既有「媽最愛」）；保留 愛／哀 雙關 |
| 23 | `S949A｜#124` | 너. 내. 소. 딸. | 我．最．重．兒。 | 妳．我．愛．女。（妳是我的愛女） | KR 第一字是 너（妳）；要接得上 #121「媽最愛」；第九章高潮，需人工決定 |
| 24 | `E912B｜#56`（claude） | 하… 진짜 엄마, 별다줄이야. | 唉……媽媽真的是什麼都要縮寫耶。 | 唉……媽媽真的是「全縮」耶。 | 流行語遺失；S940B#25 寫「全縮」。可一併把 S940B#24「……什麼都要縮寫啊。」改成「……全都要縮寫啊。」讓「全縮」能還原 |
| 25 | `S514B｜#4` | 빨. 말. 않. 대. 박. | 快．說．否．頭．砸。 | 快．說．否．頭．裂。 | `대.박.` 其他 2 處與技能名都是 頭．裂 |
| 26 | `S111B｜#28` | 로드, 엔러쉬… 멋있는 줄임이군… | 腦．夢，腦．潮……真是不錯的略語…… | （撤回）保留 腦．夢，腦．潮 | 初版理由「是別人的說法」有誤：以實瑪利說的是全稱（#27），`로드, 엔러쉬` 是良秀自己縮的；依修正後的規則，良秀自己的縮略語一律加「．」，既有譯文正確 |
| 27 | `Voice_Ryoshu_LiuAsso4_10407｜#23` | 믿. 없. | 無．信。 | 難．信。 | E725B#49 `믿. 없.` 寫 難．信（輕微） |
| 28 | `Announcer_Ryoshu_42｜#21` | 맘.들.않. | 真．不．爽。 | 不．怎．滿。 | S415B#61 同一縮略語寫 不．怎．滿（輕微） |
| 29 | `StoryData/S210B.json｜#4`（第 1 輪新增） | 모분이 뭔데?! | 那是什麼意思？！ | 脖．扭是什麼意思？！ | 追問時沒照抄 #3 的 脖．扭，#5 的解釋就接不上（對照 E903A#6 此．此是啥？） |
| 30 | `StoryData/E724B.json｜#74`、`#75`（辛克萊）（第 2 輪新增） | 재. 없.／…재미없는 대화라 안 그래도 들어가 보려고 하셨대요. | 話．沒。／……她說這場對話沒意思，本來打算回去睡覺了。 | 無．趣。／……她說這場對話很無趣，本來就打算回去睡覺了。 | 辛克萊讀對了（#78 良秀「안.들.은 아직 말하지 않았는데…」），這裡就是 재미없다，與 P10514#63 同義；#78 本．回 仍對得上「本來…回去」 |
| 31 | `BattleAnnouncerDlg/Announcer_Ryoshu_42.json｜#26`（第 2 輪新增） | 그러니 시시비비는 버.안에서 배틀로얄로 가리는 게 좋겠군. | 既然如此，是非對錯就在蠢．中大逃殺決定吧。 | 既然如此，是非對錯就在巴．內用大逃殺決定吧。 | `버.안`＝버스 안（JP バ・中、EN "once we're back O.T.B."），與 E512B#1 巴．內（#2 鴻璐「여기는 버스 안이에요」）同義；蠢．中 是誤譯 |
| 32 | `Announcer_Ryoshu_42｜#17`（第 2 輪新增；待確認） | 호오. 제.행.이군. 그렇다면 무.상.이다. | 嚯。是正．行啊。那就沒．問了。 | 嚯。是正．行啊。那就無．妨了。 | 待確認：E920B#86 是「無所謂」（無．妨）；Announcer#17 接在 `제.행.` 後，可能是「제행무상」的文字遊戲，說它同義只靠 JP 関・無（KR 沒有展開）。若同義，選 無．妨：首字對得上 무，也不與 `문. 없.`、`별. 말.` 的 沒．問 重複 |
| 33 | `Announcer_Ryoshu_42｜#0`（第 2 輪新增） | 내.머. 훗. 내려라, 머저리들이라는 뜻이다. | 退．蠢。哼。退下，蠢貨們的意思。 | 下．蠢。哼。下去，蠢貨們的意思。 | 與 S201B#78 `내머.` 下．蠢（#80「내리자, 머저리들」→「該下車了，蠢貨們」）同義；展開句一起改才對得上 |
| 34 | `StoryData/E302B.json｜#32`、`#33`（辛克萊）（第 2 輪新增） | 그. 그.／그게 그거라니요?! | 那．那。／那就那樣嗎？！ | 都．樣。／什麼叫都一樣？！ | 그게 그거；E903A#9 同義寫 都．樣（#8 키라「縮成『都一樣』不就好了」）。Announcer#19 意．中 是 그럼 그렇지，意思不同，保留 |
| 35 | `StoryData/P11110.json｜#38`（第 2 輪新增） | 쓸. 없. 아직도 이해를 포기하지 못했을 줄이야. | 沒．用。竟然還沒放棄理解啊。 | 無．用。竟然還沒放棄理解啊。 | RPG floor-b2#45/1、b2-b#74/1 的 `쓸.없.` 寫 無．用。쓸데없다（多餘，JP 不・要）與 쓸모없다（沒用處，JP 無・用）是近義、不完全相同；輕微，保留也可以 |
| 36 | `StoryData/S430B.json｜#58`、`#59`（辛克萊）（第 2 輪新增） | 삼.대.／삼조씨… 대답하시래요… | 三．答。／三朝先生……她讓您回答…… | 二擇一：（甲）三．打。／三朝先生……她要您打開天窗說亮話……（#61–63 不動）；（乙）三．開。／三朝先生……她要您開口……（#61 三．開……！，#62「她這次說要讓您腦袋開花……！」，#63「我說現在就讓他腦袋開花。」） | 劇中同一組字兩讀（#59 대답、#62 대가리），JP サム・こ、EN S.A. 都只用一組；照 `시. 비.` 時．挪 用一組字。（甲）只改兩行，但「打開天窗說亮話」是「說實話」，少了單純「回答」的意思；（乙）「開口」較接近「回答」，要改 #58、#59、#61、#62、#63 |
| 37 | `StoryData/S1013B.json｜#10`、`#12`（第 2 輪新增） | …황. 가. 라도 있어야 하나?／쯧… 황. 가. 는 황금가지가 아니라, 황금가죽이다. | ……得要有黃．金．才行嗎？／嘖……黃．金．指的不是金枝，是黃金皮革。 | ……得要有黃．金才行嗎？／嘖……黃．金指的不是金枝，是黃金皮革。 | 末字後多了「．」，會讀成三字的 黃．金．才、黃．金．指（初版的表就這樣抓錯）。黃．金 本身兩讀（金枝／皮革）、與 E914B#88 金．枝 異義，都對；同類的 腳．多．的 等見 5.1，不另列 |

不列為提案、只記錄：
- 雙關遺失但沒有好的修法：`단.테.`（Voice_Yurodivy#4、#23 → 襲．眾，後句「時鐘……沒叫你」失去笑點；JP 団・テ、EN D.O.N.T.E. 都保留）、`명.단.`（Voice_Yurodivy#12 → 送．墓，失去「名單」的雙關；EN H.I.T. List 保留）
- 第 2 輪重新核對初版的「刻意不同、應保留」（詳見 3.5、5.3）：`모.불.아.위.`（美／阿，括號展開不同）、`황. 가.`（E914B#88 황금가지 金．枝，讀法來自 JP、EN／S1013B 황금가죽、誤讀場景 黃．金）、`그.그.` 的 意．中（Announcer#19 그럼 그렇지，讀法來自 JP、EN 與 KR 語氣）是異義，保留；`삼.대.`（劇中誤讀，應用一組字，提案 36）、`버.안`（蠢．中 是誤譯，兩處都是 버스 안，提案 31）、`그.그.` 的 那．那／都．樣（同義，提案 34）改列提案；初版另在 5.3 與提案 5 說 `재. 없.` E724B#74 話．沒「配合辛克萊的解讀」，實為同義（提案 30）。另外 `말. 안. 어.`（不．聽．大／不．聽．小）是異義，保留；`어.이.`（喂．你／喂，李）暫列異義、未確認（見 3.2）
- `버.놈.`（Announcer#20「그딴 컨디션으로 버.놈.에게 덤빈다라」，繁中 蠢．傢）：EN "Charging at our team of B.B." 讀成巴士上的我們，JP バ・奴 兩可（バカ／バス），劇中沒有展開。若是 버스 놈들，蠢．傢 也是誤譯；無法判斷，只記錄
- `료.고.파` 在 12 個以上單位都寫 良．派（少了「肉」），格里高爾 P11204#30 的展開「良秀的大肉派」還讀得出，不建議改
- `모.분.말.도…`（S430B#54）KR 第三字是 말 不是 조，繁中沿用 脖．扭．豬．閉；劇中沒有展開，無法判斷

## 10. 其他發現（不在本次提交範圍）

1. （第 1 輪已以 routing `guides:` 處理）`BattleAnnouncerDlg/Announcer_Ryoshu_42`（34 處縮略語，含 `내.머.` 的展開）**不會掛 ryoshu.md**：`lct guide resolve` 只給 base＋announcer。`CharacterByFile` 的檔名前綴只有 `AbDlg_`、`PersonalityVoiceDlg/Voice_`、`EGOVoiceDig/Voice_EGO_`。建議加上 `BattleAnnouncerDlg/Announcer_`，或在 routing 加 guides 條目
2. 良秀人格的戰鬥氣泡（`BattleSpeechBubbleDlg`，如 battle_speechbubble_10408_1）、技能與被動名（`Skills_personality-04`、`Passives` 1040xxx）也不掛 ryoshu.md
3. （第 1 輪已處理，連同 `그레고르F`、`이스마엘F`）ryoshu.md 的 `models` 有 `료슈F`：18 行全在 E001X（換身體事件），說話的是鴻璐的意識（「죄송해요, 단테 님」「사탕 갖고 오셨나요?」）；反過來 model `홍루` 在 E001X 的縮略語才是良秀。建議把 `료슈F` 移到 `models_not_this`（本次依指示不動 header）
4. （第 1 輪已新增條目 `시. 대`）詞彙表 `시.대.`／`시. 대.` 沒涵蓋 `시. 대?`、`시. 대`（S1004B#16、S1027B#18），所以 S1027B#18 的 鐘．頭 沒被擋（已用 `lct validate` 確認）；可補一個 `re:` 形式
5. `소지아비*` 是否列為阿賴耶的 `models_other_self`，見 7.1
6. （第 1 輪已處理）araya.md 加 `료슈딸` 前要先改 `TestCharacterByModel`，見 7.4

## 11. 寫進指引的內容與驗證（初版；第 1 輪的版本見下節）

- `guides/characters/ryoshu.md`（6,111 bytes）：header 與原有段落不動；「資料補充」下新增「縮略語：KR 怎麼縮」「縮略語：譯法」「縮略語：劇中展開或猜測時」「反覆出現的縮略語照既有寫法」（表擴充為 25 個縮略語，並更正 `거.거.` 的提問者是 키라、`우.미.` 實為 `우.미.다`），列出 4 個待確認的統一提案；新增「與阿賴耶的縮略語不同」；例句不動
- `guides/characters/araya.md`（新檔）：NPC header（`number: null`、`code: null`、`models: [료슈딸2, 료슈딸연구실, 아라야공백]`、`models_other_self: [아라야10대, 아라야20대, 아라야정장, 아라야정장2]`、`file_names: []`）；身分、語氣、`료-슈`、流行語縮寫的寫法與既有譯法表、逐字強調的點、例句
- 驗證：`bin/lct rules lint` → `ok: the rule files are valid`（只有原本就有的 routing 警告）；`go vet ./... && go test ./...` 全部通過；`bin/lct batch preview`：
  - `StoryData/S949A.json|/dataList/124/content`（료슈9장다침）→ base、story-main、`guides/characters/ryoshu.md`
  - `StoryData/S951B.json|/dataList/28/content`（료슈9장이후반담배）→ 同上
  - `StoryData/S947B.json|/dataList/224/content`（료슈딸2）→ base、story-main、`guides/characters/araya.md`
  - `StoryData/E912B.json|/dataList/56/content`（료슈딸2）→ base、story-interlude、`guides/characters/araya.md`
  - `StoryData/S940B.json|/dataList/25/content`（료슈딸）→ 沒有角色指引（見 7.4）

## 修正第 1 輪

審查：第 1 輪 changes needed（16 項，以下是修正），第 2 輪複審通過。四個提交：

| 提交 | 檔案 | 內容 |
|---|---|---|
| `afb4bf6` | `guides/characters/ryoshu.md`、`araya.md`、`gregor.md`、`ishmael.md`、`honglu.md`、`internal/rules/rules_test.go` | 指引修正、幼年阿賴耶、E001X 換身體 |
| `9370303` | `guides/routing.yaml`、`internal/rules/rules_test.go` | 播報員語音掛 ryoshu.md；測試輔助函式 |
| `bd38a85` | `glossary/terms.yaml` | `시. 대` 條目 |
| `bc17f22` | `guides/characters/ryoshu.md` | 借用的 `완소` 說明用字 |

### 重要

1. **譯法規則改正**：初版「KR 沒加點的不加點（S101B#45、S111B#28）」是錯的。良秀自己造的 6 處不加點縮略語，繁中全部加了「．」（脖．扭．豬．閉、下．蠢、沒．因、劍．創．藝．極、腦．夢／腦．潮）。現在的規則是：良秀自己的縮略語一律加「．」，只有借用阿賴耶的流行語不加點（品死、不悶不煩）。提案 26 隨之撤回
2. **`료슈딸` 加入 araya.md `models`**：
   - `rules_test.go` 改為 `{"료슈딸", "araya", false}`，新增 `{"그렉료슈", "", false}`
   - ryoshu.md 的 `models_not_this` 保留 `료슈딸`，araya.md header 的測試註解刪除
   - `lct batch preview "StoryData/S940B.json|/dataList/25/content"` 的指引：base、story-main、**`guides/characters/araya.md`**（修正前沒有角色指引）
3. **播報員語音**：在 `guides/routing.yaml` 新增 `guides:`：`- when: "BattleAnnouncerDlg/Announcer_Ryoshu"`、`guide: guides/characters/ryoshu.md`，並加註解說明為何不用通用前綴。
   - 驗證：`lct guide resolve --file BattleAnnouncerDlg/Announcer_Ryoshu_42.json` 與 `lct batch preview "BattleAnnouncerDlg/Announcer_Ryoshu_42.json|/dataList/11/dlg"` 都給出 base、announcer、**ryoshu**
   - routing 原本沒有 `guides:` 鍵，而 `internal/rules/rules_test.go` 的 `addGuideRules` 會在檔尾再寫一次 `guides:`。結果 YAML 鍵重複，`TestGuidesForLayers`、`TestLintGuideRules` 等 10 個測試失敗
   - 因此把這個輔助函式改成「插進既有的 `guides:` 清單前端；沒有清單才新增」。測試內容不變

### 次要

4. **數字**：已依審查重算，與審查相同。
   - 總數 389 處、356 行；加點 379／389；繁中用「．」353／356 行；`모.분.` 脖．扭 20／21；`알.못.` 懂．自．懂 7／7
   - 漏掉的 3 處補進完整表：`E030X｜#94` `알. 못아.` 懂．自．懂；`E908B｜#45` `흥. 이.` 趣．談（初版把 흥 當感嘆詞刪掉了）；`Voice_Butler_10408｜#21` `모. 분이다` 脖．扭
   - 「整句只有縮略語」改為 86／356（容許一個感嘆詞時 94）
   - 「252／124」是全部加點縮略語的合計，現在是 255／124，報告已改寫法；指引裡刪掉這個數字
5. **規則用語**：
   - 語序：改用 `꾸. 말.` → 不．磨 舉例（報告另舉 折．肢）
   - 人名：改為「取譯名中可辨認的字」（希．克、辛．萊）
   - KR 規則：補上依存名詞、輔助動詞、句末述語也常略去（`모.불.아.위.`、`남.손.다.분`）
6. **劇中沒有說出的展開**：`너. 내. 소. 딸.`（標「劇中未展開」）、`히.클.` 히스클리프（標「推定」）、araya.md 的 `완소`／`오키`／`암툰`／`당근`（標「未展開；推定」）。`어.떠.` 的展開從指引刪掉，報告標「推定」
7. **완소**：既有譯法是「最愛」（「媽」來自 엄마는），兩份指引都改寫為「완소＝最愛」。ryoshu.md 註明 Announcer_Ryoshu_42#11 應作「最愛的刀」，既有是「珍愛」，即提案 22
8. **對照順序**：ryoshu.md 改成成對的寫法：감다뒤＝品死、완소＝最愛、세젤맛＝世一好吃、별다줄＝全縮
9. **`모.분.` 的展開**：不再寫成規則，改列在待確認：既有 折斷 3（S210B#5、ES130B#10、P10413#19）、扭斷 2（S101B#46、Announcer_Ryoshu_42#28），提案用扭斷
10. **待確認**：獨立成 `## 待確認` 段，開頭寫「決定前照既有最多的，同票用第一個」。各項的第一個寫法：
    - `관.없.` 沒．興；`상.없.` 無．謂；`재.없.` 無．趣；`동.감.` 同．意
    - `모.분.` 展開：折斷
    - S949A#124：照既有 我．最．重．兒
    - araya.md：`별다줄` 全縮
11. **重複或追問時照抄**：新增「別人重複或追問時照抄同一組字（E903A#6 此．此是啥？；S210B#4 `모분이 뭔데?!` 漏了）」，並新增提案 29
12. **篇幅**：ryoshu.md 由 6,111 B 減為 4,633 B。
    - 刪掉不影響選擇的統計（252／124、343／376、110、332／382、各人展開次數）
    - 與阿賴耶重複的內容改為指向 araya.md
    - 「既有寫法」由表格改為一行清單，並刪去只出現一次的 `디.메.`、`다.넷.놈.` 與只在 UI 的 `우.미.다`（這些寫法仍記錄在報告）
13. **소지아비**：araya.md「身分」補一句：它是小指父輩稱號的 key，成年的阿賴耶以 teller 아라야 用它說話（E910A#0–8、S948A#100–118），同一組 key 也是前任（S940B 等回憶），所以要看劇情判斷；不列入 models
14. **詞彙表**：新增 `kr: 시. 대`（zh 時．頭、forbidden 鐘．頭、整詞比對）。
    - 全部 KR 中末字沒點的只有 `시. 대?` 3 處：S1004B#15、#16、S1027B#18
    - `lct glossary scope`：`시. 대` 有 3 個單位，2 個用 時．頭（`시.대.` 4／4、`시. 대.` 7／7 不變）
    - `lct batch preview "StoryData/S1027B.json|/dataList/18/content"`：glossary 列出 `시. 대 → 時．頭`，check 由 accept 變為 **reject**（`forbidden-term [hard] "鐘．頭" is a forbidden rendering of 시. 대`）
15. **E001X 換身體**：逐行核對後移動。
    - 核對：
      - `료슈F` 18 行，是鴻璐的意識（「죄송해요, 단테 님」「사탕 갖고 오셨나요?」）
      - `그레고르F` 13 行，是以實瑪利的意識（#154「저는… 천성이 뱃사람인데… 곤충 팔이 있으니까요」、#195「저기요, 그레고르 씨! 제 몸으로 담배 피우지 마시라고요」）
      - `이스마엘F` 9 行，是格里高爾的意識（#151「여어, 관리자 양반」、#152「…울적해하는 내 얼굴」、#196「안 태워… 그냥 물고만 있는 거야」）
      - model `홍루` 是良秀的意識（#202 `다. 뒤.`、#205「어이. 검, 내놓도록」、#242 `시. 대.`、#321「…번역.」）
    - 處理：
      - 三個 F key 從各自的 `models` 移到 `models_not_this`，附註「E001X 換身體，X 的意識」
      - honglu.md「資料補充」加一條：E001X 的 `홍루` 是良秀的意識，照良秀的語氣與縮略語譯；`료슈F` 才是鴻璐
    - 另外看到 E001X 的 `싱클레어`（「야, 돌았냐?」）與 `히스클리프2`（「근육… 늘었는데…」）也像是互換。這兩個 key 在其他檔也用，沒有移動，留待決定
    - 也可以考慮把 F key 列進意識所屬角色的 `models`（例如 `료슈F` → honglu.md），本輪照指示只移到 `models_not_this`
16. **提案 26**：理由改正（`로드, 엔러쉬` 是良秀自己縮的），並依新規則撤回

### 驗證

- `bin/lct rules lint`：`ok: the rule files are valid`（只有原有的 routing 節點警告）
- `go vet ./... && go test ./...`：37 個套件全部 ok。另在 `afb4bf6`、`9370303` 各自的工作樹跑過 `internal/rules` 與 `cmd/lct` 的測試，也都 ok
- `lct batch preview`：
  - item 2：`StoryData/S940B.json|/dataList/25/content`（료슈딸）→ base、story-main、araya.md
  - item 3：`BattleAnnouncerDlg/Announcer_Ryoshu_42.json|/dataList/11/dlg` → base、announcer、ryoshu.md
  - item 14：`StoryData/S1027B.json|/dataList/18/content` → glossary `시. 대 → 時．頭`，check: reject（forbidden-term 鐘．頭）
- 檔案大小：ryoshu.md 4,633 B、araya.md 3,804 B、honglu.md 2,700 B

## 修正第 2 輪

使用者指出三個問題：第 3 節的表 JP、EN 各只列一種寫法；沒有區分「同義異寫」與「異義」；`버.안` 的「刻意不同」是錯的。以下是本輪的改動（只改 `docs/research/ryoshu-abbreviations/**` 與 `guides/characters/ryoshu.md`，不改譯文）。

### 第 3 節重做

- 逐處取出與縮略語對應的 JP、EN、繁中片段，列出全部寫法與次數，加「類別」欄（說明見第 3 節開頭）。出現兩次以上的 50 種完整列出（3.1）；只出現一次的 234 種另成一表（3.3）
- 初版表的配對錯誤（同一行有兩個以上縮略語，或行內有別的「・」詞時抓到別的片段）：
  - JP 10 列：`사. 분.`（S1005B#35，記成 `모. 분.` 的 く・へと，應為 し・へ）、`살.`（記成 片・突・死，應為 蘇，不加點）、`싱.어.`（ヒ・ク／く・へ → シン・ア）、`모. 퇴.`、`밟. 분.`、`이. 계.`（抓到 ル・ルージュ、ル・ノワール 這類名字）、`머. 상.` 與 `돌. 고.`（兩個片段黏成一個）、`차. 마.` 與 `공. 절.`（同上）；其餘約 110 列只是多抓了前後文（「ぱ頭も類・友っていう」→ 類・友）
  - EN 7 列：`말. 안. 어.`（S.Y.N.C → ES130B#6 U.A.、#8 U.C.）、`어. 달.`（U.C. → S.C.A.N.）、`머. 상.` 與 `돌. 고.`（W.C.／C.W. 對調）、`완.소.` 與 `완소`（I.L.Y.／ILY 對調）、`엔러쉬`（LD → E-Rush）
  - 繁中 3 列：`황. 가.`（黃．金．才、黃．金．指 → 黃．金，見 3.2）、`엔.팔.`（P10410#76 記成 `초. 깡.` 的 腦．醃．罐；實際沒有縮，譯「腦啡肽」）、`살.`（記成 閉．突．死；實際沒有縮，譯「復活」）
- 邊界修正 3 處：`알.단.잘.부`、`죽. 살. 죽. 살`、`쓸.없.`＋파하하（見第 3 節開頭）；4.1 的音節數、5.1 的字數統計隨之重算
- 分組：良秀自己不加點的併入加點的、接 이다 的 `다.` 併入本體，284 種（摘要已更新）
- 繁中末字後的點：初版寫「一律不加」，實際有 8 處留了點（5.1、提案 37）

### 「刻意不同」重新核對

| 縮略語 | 初版 | 場景 | 判定 | 改動 |
|---|---|---|---|---|
| `삼.대.`（S430B#58–63） | 三．答／三．打，刻意不同 | 辛克萊先讀「삼조씨… 대답하시래요」（#59），再讀「이번엔 대가리를 깨버린다고」（#62），良秀「틀렸어. 지금 대가리를 깬다고 했다.」（#63）；JP サム・こ、EN S.A. 兩次都一樣 | 劇中誤讀：同一組字兩讀，照 `시. 비.` 時．挪 的規則應只用一組字 | 5.3 改寫；新增提案 36；指引的「猜錯或雙關」加上此例 |
| `버.안`（Announcer#26、E512B#1） | 蠢．中＝버러지 안／巴．內＝버스 안，刻意不同 | E512B#2 鴻璐「여기는 버스 안이에요~」（JP バ・な、EN T.B.）；Announcer#26「시시비비는 버.안에서 배틀로얄로」（JP バ・中、EN "once we're back O.T.B."） | 同義（兩處都是 버스 안），蠢．中 是誤譯 | 新增提案 31 |
| `모.불.아.위.`（EGO_4#0、#1） | 美／阿，刻意不同 | 兩條各自在括號展開：「…아름다움을 위해」／「…아라야를 위해」；JP 美／阿 跟著變 | 真正異義 | 保留 |
| `황. 가.`（E914B#88、S1013B#10–12） | 金．枝／黃．金，配合雙關 | E914B#88 讀成 황금가지來自 JP 黄・枝、EN G.B.（KR 沒有展開）；S1013B 莫梭讀成 황금가지（#11），良秀更正為 황금가죽（#12），JP お・の、EN G.B.、繁中 黃．金 都一組字兩讀 | 真正異義；S1013B 內部也照規則只用一組字 | 保留；初版記成 黃．金．才／黃．金．指 是抓錯，新增提案 37（末字後的點） |
| `그.그.`（Announcer#19、E302B#32、E903A#9） | 意．中／那．那／都．樣，各自配合場景 | Announcer#19 讀成 그럼 그렇지來自 JP 案・定、EN "knew it. W.A." 與「쳇…하지만」的語氣（KR 沒有展開）；E302B#32（辛克萊「그게 그거라니요?!」）與 E903A#9（키라「그게 그거, 정도로 줄이면 되잖아」）都是 그게 그거 | 意．中 與另兩處異義，保留；那．那／都．樣 同義、不一 | 新增提案 34 |
| `재. 없.`（E724B#74） | 話．沒，配合辛克萊的解讀，保留（5.3、提案 5） | 辛克萊「재미없는 대화라 안 그래도 들어가 보려고 하셨대요」讀對了，良秀 #78 認可 | 同義（재미없다），話．沒 與 P10514#63 無．趣、S951B#28 沒．致 不一 | 5.3 改寫；提案 5 改理由；新增提案 30 |

另外找到的：`말. 안. 어.`（ES130B#6／#8，어른／어린아이）是異義，繁中不同是對的；`어.이.`（5D205B#3 對魁魁格／S445B#0 叫李箱，JP お・い／お、イ、EN H.W.／E.Y.）KR 沒有根據，暫列異義、未確認。`내.머.`（退．蠢／下．蠢）是新找到的中文不一（提案 33），`쓸.없.`（沒．用／無．用，近義）、`무.상.`（Announcer#17 沒．問／E920B#86 無．妨，待確認）也列入提案 35、32。`버.놈.`（Announcer#20 蠢．傢）也可能是 버스 놈들，無法判斷，只記錄在第 9 節。

### 提案

- 新增 8 條：30（`재. 없.` E724B#74）、31（`버.안` Announcer#26）、32（`무.상.` Announcer#17）、33（`내.머.` Announcer#0）、34（`그.그.` E302B#32）、35（`쓸.없.` P11110#38）、36（`삼.대.` S430B#58）、37（S1013B#10、#12 末字後的點）
- 修改 1 條：5（理由改正：E724B#74 也是同義）
- 第 9 節「不列為提案」的「刻意不同、應保留」改寫為上表的判定

### 指引（`guides/characters/ryoshu.md`）

- 「縮略語：劇中展開或猜測時」：
  - 「猜錯或雙關」改為「也只用一組字」，加上 S430B `삼.대.` 的反例
  - 新增一條：JP、EN 只供確認是縮略語，異寫不要跟（JP 換詞、EN 另造縮寫，如 `모.분.` S.Y.N.C.／S.T.N.C.），同一組 KR 用同一組字；換字要有 KR 的根據：劇中展開不同，或 KR 上下文明顯不同義（ES130B `말. 안. 어.` 不．聽．大／小），JP、EN 只能佐證（複審後改寫，見下）
- 「譯法」的「JP、EN 只供確認是縮略語」併入上面那條
- 「待確認」的 `재.없.` 加上 話．沒
- 「依據」一行縮短。大小 4,633 B → 4,916 B

### 腳本

- 新增 `scripts/tokens.py`、`scripts/tokens_ov.py`、`scripts/table2.py`（用法見 `scripts/README.md`）
- 網站資料（rows.pkl）與 raw 對照：356 行的 KR、JP、EN 與 tw `008ead4` 的繁中逐行比對，只有 S445B#0 的 JP 不同（網站資料是空的，raw 是「お、イ。」），表已照 raw 改

### 複審後的修正

`4eb5c77` 的獨立複審通過（5 種異義、13 種同義異寫、22 種一致與 `삼.대.`、`버.안`、`재. 없.` 的判定都確認），另提出以下各點，已修正：
- 指引的新規則原寫「劇中展開不同，或 JP、EN 一起改成別的意思才換」：後半讓 JP、EN 一致就能決定意思，初版 `버.안` 正是這樣錯的。改為換字要有 KR 的根據（劇中展開不同，或 KR 上下文明顯不同義），JP、EN 只能佐證
- `어.이.`：KR 兩處相同、S445B#0 沒有展開，讀成李箱只靠 JP、EN、繁中，改標「未確認」，並寫出能決定的根據（3.2）
- `그.그.` Announcer#19（그럼 그렇지）、`황. 가.` E914B#88（황금가지）：註明讀法來自 JP、EN（前者另有「쳇…하지만」的語氣），KR 沒有展開
- 提案 36：三．打 與 三．開 並列，並註明「打開天窗說亮話」少了單純「回答」的意思
- `쓸.없.`：쓸데없다（P11110，JP 不・要）與 쓸모없는（RPG，JP 無・用）是近義、不完全相同，改寫說明與提案 35
- `무.상.` Announcer#17：可能是「제행무상」的文字遊戲，與 E920B#86 同義只靠 JP 関・無 推定，提案 32 標「待確認」
- 單位寫法註明 `#` 是 `dataList` 中從 0 起算的位置
- 腳本：`table2.py` 新增 `NOTE`（類別的「未確認」「待確認」「近義」附註）；類別的計數不變

### 驗證

- `bin/lct rules lint`：`ok: the rule files are valid`（只有原有的 routing 節點警告）
- `go vet ./... && go test ./...`：37 個套件全部 ok
- 表的自我核對：直接讀 raw-20261105 的 `kr/`、`jp/`、`en/` 與 `git show origin/tw:…`（`008ead4`），核對 3.1 的 `모. 분.`（其中 PC02B#50、E725B#35、E406B#16）、`노. 잼.`（LCCB#6）、`버.안`、`무.상.`、`그.그.`、`재. 없.`、`황. 가.`、`내.머.`、`말. 안. 어.`、`삼.대.`、`모.불.아.위.`（#1）、`어.이.`、`쓸.없.`（P11110#38、floor-b2#45/1）、`사. 분.`（S1005B#35）、`알. 못.`（P10410#56）、`거. 거.`（E507A#43）與 3.3 的 `머. 상.`、`돌. 고.`、`살.`、`완.소.`、`완소`、`로드`、`엔러쉬` 共 23 列，所列單位的 JP、EN、繁中片段都與 raw、tw 相同；連同上一項的逐行比對，表的資料來源與 raw 一致

## 修正第 3 輪

使用者要求第 3 節的表列出完整資訊：KR 縮略、KR 完整（標示遊戲內說明或推測）、JP 縮略、JP 完整、EN 縮略、EN 完整、現在的繁中縮略、繁中完整、對應的檔案位置；依類別分成一致、同義異寫、異義、中文不一 4 張表；KR 相同而 JP、EN、繁中不同時不合併，各占一行。

### 第 3 節重做

- 依（KR 縮略語、JP、EN、繁中寫法）分行：284 種拆成 319 行（`scripts/rows4.py`，沿用第 2 輪逐處取出的片段 `tok.pkl` 與類別 `CLS`）
- 完整說法：每行附前 3 處的上下文（前後各 3 個單位），分 7 份人工判讀四種語言的完整說法與來源（判讀說明見 `scripts/full_forms.md`），結果在 `scripts/full_forms.json`。來源分遊戲內、推測、未縮、不明，定義見第 3 節開頭；「遊戲內」由 `scripts/check_full.py` 逐字核對出處
- 抽查與更正：見第 3 節開頭的「核對」。各份對「加點的各段合起來就是完整的詞」（`젠. 장.`、`無．聊`）標法不一，統一為「未縮」；不確定的標記統一為全形「？」
- 節號變動：第 2 輪的 3.1（出現兩次以上）、3.3（只出現一次）合併後依類別改為 3.1–3.4；原 3.2「異義與中文不一的核對」改為 3.5，原 3.4「良秀人格的技能、被動名稱」改為 3.6。「修正第 1、2 輪」裡的節號是當時的，不改

### 判讀中改正的劇中展開

第 2 輪的「劇中展開」欄有幾處取的是劇中的**誤讀**，良秀隨後自己更正；本輪改用更正後的說法：

- `숨. 죽.`（P10411）：另一隻兔子先讀成「숨을 죽이라는」（#1），良秀 #3 更正為 숨겨진 죽간
- `반. 반. 위.`（E912B）：綺羅 #107 讀成「반의 반보다 위」，良秀 #108 更正為 반반보다 위
- `임. 동.`（E908B）：紅樓 #111 讀成「임시 동맹」，良秀 #112 更正為 임시로 동행
- `같. 취.`（E902B#64）：唐吉訶德讀成「같은 취미」與上下文不合，改為推測 같은 취급（JP 類・友＝類は友を呼ぶ）
- `황. 가.`（S1013B#10、#12）：莫梭讀成 황금가지（#11），良秀 #12 更正為 황금가죽；E914B#88 仍是 황금가지（推測，依 JP、EN）

### 驗證

- `scripts/check_full.py`：319 行齊全，「遊戲內」的出處逐字核對 0 錯
- 表由 `scripts/table3.py` 產生，與報告第 3.1–3.4 節相同
