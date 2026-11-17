# ruby 與注音（共用指引）

本檔只管劇情台詞（`StoryData/` 的 `content`）與帶 `ruby_customize: true` 的單位；這些檔的 `title`、`place` 照下面的「說話者名稱、稱號、地點」，其他沒有標記的欄位照 base.md「其他位置」（規則自 base.md 移入）。

## 格式慣例

- **劇情台詞：ruby 一律保留為標籤，不可改寫成「Y(X)」或「Y（X）」，也不可刪掉**。KR、參考譯文或 JP、EN 用 ruby 表現的讀音、雙關與原意，KR 沒有 ruby 也用 ruby 寫出
  - X 寫該讀音所屬語言的文字。**良秀的兩種讀音**：良秀出自日本文化的作品，讀音照 JP 寫假名。KR 寫 `요시히데`（本名的唸法，家人與蜘蛛巢的人這樣叫她；JP 寫 ヨシヒデ）時寫 `<ruby=ヨシヒデ>良秀</ruby>`；KR 寫 `료슈` 的照常寫「良秀」，只在和 ヨシヒデ 對比時照 JP 寫 `<ruby=りょうしゅう>良秀</ruby>`（「還有，別叫我<ruby=ヨシヒデ>良秀</ruby>，叫我<ruby=りょうしゅう>良秀</ruby>。」）；拉長的唸法照 JP 寫 `<ruby=りょーしゅー>良秀</ruby>`；逐字或結巴地唸出時每個片段各加，假名照 JP（「<ruby=りょう>良</ruby>……<ruby=しゅう>秀</ruby>……？」「<ruby=ヨ>良</ruby>、<ruby=ヨシヒデ>良秀</ruby>姊姊」「<ruby=ヨシ>良</ruby>……<ruby=ヒデ>秀</ruby>……」）。其他外語讀音照 EN 的拼法（`김삿갓` → 「金<ruby=Satgat>笠</ruby>」）
  - 雙關與原意寫中文：「<ruby=李箱>異常</ruby>」「<ruby=研尋>研心</ruby>」「<ruby=請多指教>夜良秀苦</ruby>」；口齒不清或扭曲的台詞標出原意：「<ruby=你不可能>你唔可仍</ruby>」
  - **外語的詞、台詞與感嘆**：KR 用韓文拼寫的外語詞（各章作品的語言、擊劍用語等：`에쥬, 프레?`、`알레!`、卡利斯托的義大利語 `케 벨로`），本文寫中文意思，原文用該語言的文字寫在 ruby：「<ruby=Es-tu prêt>準備好了嗎</ruby>？」「<ruby=Allez>來吧</ruby>！」「<ruby=Che bello>真美</ruby>」；日語的詞寫假名（同良秀）。KR 已在括號附上意思或原文時（`에쥬, 프레? (준비되었나요?)`、`알레!(Allez!)`）也這樣寫，不另加括號
  - 不加：中文的詞（第八章的「怡紅院」）直接寫；韓語的詞照常翻譯，不加韓文讀音（譯文不放韓文字；詞彙表核准的「하나協會」除外）；臺灣也常用的外來語（All In、Email、cosplay）與專有名詞（協會、公司、人名、地名照詞彙表：Zwei 協會、Nagel und Hammer）照常寫；JP 只是標漢字讀音（`<ruby=くつわ>轡</ruby>`、`<ruby=エゴ>E.G.O</ruby>`）或著重點、EN 只是為英文讀者解釋拼音（`<ruby=thatch-roofed house>choga</ruby>` → 茅屋）、X 只是 Y 的外語原文而中文已寫出意思（`<ruby=Nice catch>接得好</ruby>` → 接得好）
- **白名單位置的注音**：帶 `ruby_customize: true` 的單位（`rules/ruby-customize.yaml` 的位置）寫 `<rubycustomize=X>Y</rubycustomize>`，建置時換成遊戲能顯示的標籤，不手寫 `nobr`、`space`；沒帶的單位寫了會退回
  - 時機與 X 同劇情台詞的 ruby，KR 沒有也可以加：KR `요시히데` 寫 `<rubycustomize=ヨシヒデ>良秀</rubycustomize>`
  - X 要簡短（太長會撐寬那一行），只寫讀音、原文或原意；X、Y 不可為空，只放文字，不含 `<`、`>` 或換行，X 可加雙引號；不巢狀，不和 `<ruby>` 寫在同一組
- **說話者名稱、稱號、地點**：說話者名稱（`StoryData/` 的 `teller`、`ScenarioModelCodes` 的 `name`）需要注音時（時機同上）用 `<rubycustomize>`（三語都是 `UNGEZIEFER KAISER` 的寫 `<rubycustomize=UNGEZIEFER KAISER>害蟲皇帝</rubycustomize>`），否則照詞彙表；稱號（`title`、`nickName`）與地點（`place`）不加注音，照詞彙表
