# I4 同義措辭不同：統一（G518–G566）

[回這一類的目錄](I4.md)。

### G518　統一｜I4 同義措辭不同｜信心中

`inconsistent/rpg/text/d0a959c4`　rpg／text，2 個單位，JP/EN 全同

- KR：`그런데… 다친 곳은 없어, 뫼르?`
- JP：`ところで…怪我はない、ムル？`　EN：`So... you're not hurt anywhere, are you, Meur?`
- 現有譯法：
  1. `話說回來……有受傷嗎，小莫？`（1 個單位；來源 paratranz 1；說話者 로쟈 1）
  2. `話說回來……你有受傷嗎，小莫？`（1 個單位；來源 paratranz 1；說話者 로쟈 1）
- 判讀：KR 無「너」，第二種多出「你」；統一為較貼近 KR 的寫法
- 建議：統一為 `話說回來……有受傷嗎，小莫？`
- 會改：
  - `話說回來……你有受傷嗎，小莫？` → `話說回來……有受傷嗎，小莫？`：1 個單位（RPGSystem/rpg-loc-dialogue-floor-1.json）
    - 差異：`話說回來……〔你→∅〕有受傷嗎，小莫？`
  - 不改：`話說回來……有受傷嗎，小莫？`（1）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G519　統一｜I4 同義措辭不同｜信心中　⚠ 統一後是新寫法（不是現有任何一種）

`inconsistent/stage/content/135875f5`　stage／content，2 個單位，JP/EN 全同

- KR：`'경계심'이 10에 도달할 경우 플로어 매니저가 강력한 스킬을 사용합니다.⏎경계심이 쌓이지 않도록 주의하거나 특정 스킬에 합을 패배하면 경계심을 제거할 수 있습니다.`
- JP：`警戒心が10に達すると、フロアマネージャーが強力なスキルを使用します。⏎警戒心が蓄積しないように注意しましょう。また、特定のスキルとのマッチに敗北すると、警戒心を除去できます。`　EN：`Be too wary and let yourself accumulate 10 stacks of "Guarded Caution", and you will risk inviting the Floor Manager's powerful Skill.⏎Pay attention so that "Guarded Caution" does not reach 10 stacks. You may also choose to lose a Clash against a certain Skill to remove the stacks.`
- 現有譯法：
  1. `「戒心」達到10時，樓層經理會使用強力技能。⏎請注意不要讓戒心累積；與特定技能拚點失敗也能移除戒心。`（1 個單位；來源 claude 1）
  2. `「戒心」層數達到10層時，樓層經理將使用強力技能。⏎請注意別讓戒心累積；此外，與特定技能拚點落敗也能移除戒心。`（1 個單位；來源 claude 1）
- 判讀：兩種都是同一 KR 的 AI 譯文；第二種多出「此外」且與「層數」混用，統一為接近第一種並補上「層」
- 建議：統一為 `「戒心」達到10層時，樓層經理會使用強力技能。⏎請注意別讓戒心累積；與特定技能拚點失敗也能移除戒心。`
- 會改：
  - `「戒心」達到10時，樓層經理會使用強力技能。⏎請注意不要讓戒心累積；與特定技能拚點失敗也能移除戒心。` → `「戒心」達到10層時，樓層經理會使用強力技能。⏎請注意別讓戒心累積；與特定技能拚點失敗也能移除戒心。`：1 個單位（BattleResultHint-a1c10p1.json）
    - 差異：`「戒心」達到10〔∅→層〕時，樓層經理會使…`；`…力技能。⏎請注意〔不要→別〕讓戒心累積；與特…`
  - `「戒心」層數達到10層時，樓層經理將使用強力技能。⏎請注意別讓戒心累積；此外，與特定技能拚點落敗也能移除戒心。` → `「戒心」達到10層時，樓層經理會使用強力技能。⏎請注意別讓戒心累積；與特定技能拚點失敗也能移除戒心。`：1 個單位（BattleResultHint-a1c10p2.json）
    - 差異：`「戒心」〔層數→∅〕達到10層時，樓…`；`…0層時，樓層經理〔將→會〕使用強力技能。⏎…`；`…意別讓戒心累積；〔此外，→∅〕與特定技能拚點落…`；`…，與特定技能拚點〔落→失〕敗也能移除戒心。`
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G520　統一｜I4 同義措辭不同｜信心中

`inconsistent/stage/content/165bc767`　stage／content，2 個單位，JP/EN 全同

- KR：`수셰프는 자신의 가죽 지갑도 재료로 삼기 때문에, 수감자만 지켜서는 요리를 막을 수 없습니다.`
- JP：`スーシェフは自分の革の財布も材料にするため、囚人だけを守っていては料理を阻止できません。`　EN：`Leather Wallets are not an exception when it comes to Sous Chef's ingredient-gathering. You cannot stop her culinary activity by playing defensively.`
- 現有譯法：
  1. `副廚也會把自己的皮夾當作食材，因此光是保護罪人是無法阻止她料理的。`（1 個單位；來源 claude 1）
  2. `副廚連自己的皮夾也會當成食材，因此只保護罪人是無法阻止她料理的。`（1 個單位；來源 claude 1）
- 判讀：同 JP/EN、同義改寫，無錯誤；統一為第一種
- 建議：統一為 `副廚也會把自己的皮夾當作食材，因此光是保護罪人是無法阻止她料理的。`
- 會改：
  - `副廚連自己的皮夾也會當成食材，因此只保護罪人是無法阻止她料理的。` → `副廚也會把自己的皮夾當作食材，因此光是保護罪人是無法阻止她料理的。`：1 個單位（BattleResultHint-a1c10p2.json）
    - 差異：`副廚〔連→也會把〕自己的皮夾也會當…`；`副廚連自己的皮夾〔也會→∅〕當成食材，因此只…`；`…自己的皮夾也會當〔成→作〕食材，因此只保護…`；`…會當成食材，因此〔只→光是〕保護罪人是無法阻…`
  - 不改：`副廚也會把自己的皮夾當作食材，因此光是保護罪人是無法阻止她料理的。`（1）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G521　統一｜I4 同義措辭不同｜信心中

`inconsistent/stage/content/1adb5ec4`　stage／content，2 個單位，JP/EN 全同

- KR：`바느질의 왕이 강력한 스킬을 사용하는 턴에 바늘이들이 생존해있다면 큰 피해를 받을 수 있습니다.`
- JP：`裁縫の王が強力なスキルを使用するターンに針っ子たちが生存していると、大きなダメージを受けることがあります。`　EN：`If there are surviving Needlekins in the same turn as when the King of Needleworks uses a powerful Skill, they will enhance that Skill's damage.`
- 現有譯法：
  1. `在裁縫之王使用強力技能的回合，若針靈們仍然存活，可能會受到大量傷害。`（1 個單位；來源 claude 1）
  2. `若在裁縫之王使用強力技能的回合中仍有針靈存活，可能會受到大量傷害。`（1 個單位；來源 claude 1）
- 判讀：同 JP/EN、同義改寫，無錯誤；統一為第一種
- 建議：統一為 `在裁縫之王使用強力技能的回合，若針靈們仍然存活，可能會受到大量傷害。`
- 會改：
  - `若在裁縫之王使用強力技能的回合中仍有針靈存活，可能會受到大量傷害。` → `在裁縫之王使用強力技能的回合，若針靈們仍然存活，可能會受到大量傷害。`：1 個單位（BattleResultHint-a1c10p2.json）
    - 差異：`〔若→∅〕在裁縫之王使用強…`；`…用強力技能的回合〔中仍有→，若〕針靈存活，可能會…`；`…的回合中仍有針靈〔∅→們仍然〕存活，可能會受到…`
  - 不改：`在裁縫之王使用強力技能的回合，若針靈們仍然存活，可能會受到大量傷害。`（1）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G522　統一｜I4 同義措辭不同｜信心中

`inconsistent/stage/content/3b0aba2e`　stage／content，2 個單位，JP/EN 全同

- KR：`'경청에 대한 감사'를 보유했다면 잠깐 동안 공격을 하지 않는 것도 방법입니다.`
- JP：`「傾聴への感謝」を保有しているなら、しばらく攻撃を控えるのも一つの方法です。`　EN：`If you have "Gratitude for Listening", perhaps you will want to stay your attacks for a short while.`
- 現有譯法：
  1. `若帶有「感謝傾聽」，暫時不進行攻擊也是個辦法。`（1 個單位；來源 claude 1）
  2. `若帶有「感謝傾聽」，暫時不進行攻擊也是一種方法。`（1 個單位；來源 claude 1）
- 判讀：只差「個辦法」與「一種方法」，同義；統一為第一種
- 建議：統一為 `若帶有「感謝傾聽」，暫時不進行攻擊也是個辦法。`
- 會改：
  - `若帶有「感謝傾聽」，暫時不進行攻擊也是一種方法。` → `若帶有「感謝傾聽」，暫時不進行攻擊也是個辦法。`：1 個單位（BattleResultHint-a1c10p2.json）
    - 差異：`…時不進行攻擊也是〔一種方→個辦〕法。`
  - 不改：`若帶有「感謝傾聽」，暫時不進行攻擊也是個辦法。`（1）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G523　統一｜I4 同義措辭不同｜信心高

`inconsistent/stage/content/edd9ffb4`　stage／content，2 個單位，JP/EN 全同

- KR：`경험치`
- JP：`経験値`　EN：`EXP`
- 現有譯法：
  1. `經驗值`（1 個單位；來源 paratranz 1）
  2. `經驗`（1 個單位；來源 paratranz 1）
- 判讀：「경험치」＝經驗值，「經驗」漏了「值」
- 建議：統一為 `經驗值`
- 會改：
  - `經驗` → `經驗值`：1 個單位（RewardDungeonUI.json）
    - 差異：`經驗〔∅→值〕`
  - 不改：`經驗值`（1）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G524　統一｜I4 同義措辭不同｜信心中

`inconsistent/stage/title/6f0dbd29`　stage／title，2 個單位，JP/EN 全同

- KR：`꿈에 닿으려 했던 위대한 해결사에 대한 이야기`
- JP：`夢へと至らんとした偉大なフィクサーについての物語`　EN：`The Tale of a Great Fixer Who Once Reached for The Dream`
- 現有譯法：
  1. `偉大的收尾人追求夢想的故事`（1 個單位；來源 paratranz 1）
  2. `關於偉大的收尾人追求夢想的故事`（1 個單位；來源 paratranz 1）
- 判讀：KR「~에 대한 이야기」＝「關於…的故事」，第一種漏了「關於」；統一為第二種
- 建議：統一為 `關於偉大的收尾人追求夢想的故事`
- 會改：
  - `偉大的收尾人追求夢想的故事` → `關於偉大的收尾人追求夢想的故事`：1 個單位（StageNode-a1c7p3.json）
    - 差異：`〔∅→關於〕偉大的收尾人追求…`
  - 不改：`關於偉大的收尾人追求夢想的故事`（1）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G525　統一｜I4 同義措辭不同｜信心中

`inconsistent/stage/title/87a19b97`　stage／title，2 個單位，每種 JP/EN 只對到一種譯法

- KR：`환영 인사`
- 現有譯法：
  1. `熱情歡迎`（1 個單位；來源 paratranz 1）
     - JP `歓迎の挨拶`／EN `A Warm Reception`（1）
  2. `熱烈歡迎`（1 個單位；來源 paratranz 1）
     - JP `歓迎の挨拶`／EN `A Warm Welcome`（1）
- 判讀：EN Reception／Welcome，意思同；「熱烈歡迎」較常用
- 建議：統一為 `熱烈歡迎`
- 會改：
  - `熱情歡迎` → `熱烈歡迎`：1 個單位（StageNode-a1c6p1.json）
    - 差異：`熱〔情→烈〕歡迎`
  - 不改：`熱烈歡迎`（1）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G526　統一｜I4 同義措辭不同｜信心中

`inconsistent/story-interlude/content/12bf9f72`　story-interlude／content，5 個單位，每種 JP/EN 只對到一種譯法

- KR：`그렇다.`
- 現有譯法：
  1. `沒錯。`（4 個單位；來源 claude 4；說話者 뫼르소 4）
     - JP `そうだ。`／EN `Confirmed.`（2）
     - JP `そうだ。`／EN `I do.`（1）
     - JP `そうだ。`／EN `I did.`（1）
  2. `對。`（1 個單位；來源 claude 1；說話者 뫼르소 1）
     - JP `そうだ。`／EN `Yes.`（1）
- 判讀：同一說話者（뫼르소），JP 同，「沒錯。」與「對。」同義；取多數
- 建議：統一為 `沒錯。`
- 會改：
  - `對。` → `沒錯。`：1 個單位（StoryData/E917B.json）
    - 差異：`〔對→沒錯〕。`
  - 不改：`沒錯。`（4）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G527　統一｜I4 同義措辭不同｜信心中

`inconsistent/story-interlude/content/0bf6b195`　story-interlude／content，4 個單位，部分譯法對到不同 JP/EN

- KR：`펑!`
- 現有譯法：
  1. `砰！`（3 個單位；來源 claude 3）
     - JP `パァン！`／EN `<i>*POP!*</i>`（2）
     - JP `パァン！`／EN `*POP!*`（1）
  2. `啪！`（1 個單位；來源 claude 1）
     - JP `パァン！`／EN `*POP!*`（1）
- 判讀：JP/EN 全同的音效，「砰」「啪」皆可；統一為多數「砰！」
- 建議：統一為 `砰！`
- 會改：
  - `啪！` → `砰！`：1 個單位（StoryData/E918A.json）
    - 差異：`〔啪→砰〕！`
  - 不改：`砰！`（3）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G528　統一｜I4 同義措辭不同｜信心中

`inconsistent/story-interlude/content/8f9c4b66`　story-interlude／content，2 個單位，JP/EN 全同

- KR：`<…어떻게 생각해, 파우스트?>`
- JP：`<…どう思う、ファウスト？>`　EN：`< ... Your thoughts, Faust?>`
- 現有譯法：
  1. `<……妳怎麼想，浮士德。>`（1 個單位；來源 paratranz 1；說話者 단테 1）
  2. `<……妳怎麼想呢，浮士德？>`（1 個單位；來源 paratranz 1；說話者 단테 1）
- 判讀：KR 是疑問句（?），1 筆以句號收尾；統一為問句
- 建議：統一為 `<……妳怎麼想呢，浮士德？>`
- 會改：
  - `<……妳怎麼想，浮士德。>` → `<……妳怎麼想呢，浮士德？>`：1 個單位（StoryData/PC07B.json）
    - 差異：`<……妳怎麼想〔∅→呢〕，浮士德。>`；`…妳怎麼想，浮士德〔。→？〕>`
  - 不改：`<……妳怎麼想呢，浮士德？>`（1）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G529　統一｜I4 同義措辭不同｜信心中

`inconsistent/story-main/content/61207259`　story-main／content，2 個單位，JP/EN 全同

- KR：`그렇지만 크로머는 아직도 의기양양한 채로, 사형선고라도 내리듯 손을 뻗어 싱클레어의 코트 안주머니를 뒤진다.`
- JP：`しかしクローマーはまだ意気揚々として、⏎死刑宣告でも下すように手を伸ばしてシンクレアのコートの内ポケットをまさぐる。`　EN：`As opposed to us, Kromer still stands triumphantly. She reaches for the inside of Sinclair’s coat in what seems like a declaration of his fate.`
- 現有譯法：
  1. `但是克羅默依舊得意洋洋，伸出手在辛克萊的外套口袋中摸索著，彷彿在宣佈死刑一般。`（1 個單位；來源 paratranz 1）
  2. `但是克羅默依舊得意洋洋，伸出手在辛克萊的外套內袋中摸索著，彷彿在宣布死刑一般。`（1 個單位；來源 paratranz 1）
- 判讀：「宣佈」應寫「宣布」；안주머니 是內側口袋，「內袋」較貼近 KR，統一成第二種
- 建議：統一為 `但是克羅默依舊得意洋洋，伸出手在辛克萊的外套內袋中摸索著，彷彿在宣布死刑一般。`
- 會改：
  - `但是克羅默依舊得意洋洋，伸出手在辛克萊的外套口袋中摸索著，彷彿在宣佈死刑一般。` → `但是克羅默依舊得意洋洋，伸出手在辛克萊的外套內袋中摸索著，彷彿在宣布死刑一般。`：1 個單位（StoryData/3D308B.json）
    - 差異：`…手在辛克萊的外套〔口→內〕袋中摸索著，彷彿…`；`…摸索著，彷彿在宣〔佈→布〕死刑一般。`
  - 不改：`但是克羅默依舊得意洋洋，伸出手在辛克萊的外套內袋中摸索著，彷彿在宣布死刑一般。`（1）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G530　統一｜I4 同義措辭不同｜信心中

`inconsistent/story-main/desc/b3f9ca21`　story-main／desc，8 個單位，JP/EN 全同

- KR：`컨테이너들이 있는 층을 지나 내려오니, 서늘한 공기가 우리를 맞이한다. 곳곳에 보이는 얼음들은 끝이 머지않았음을 암시한다.`
- JP：`コンテナのある階を通り過ぎて下りていくと、冷え冷えとした空気が私たちを出迎えた。あちこちに見える氷が、終わりはそう遠くないことを暗示する。`　EN：`Moving past the containers, we’re met with a chill air. The chunks of ice hint that we’re almost to the end.`
- 現有譯法：
  1. `在經過貨櫃時，周圍空氣變得更加寒涼。大塊的冰塊提示著我們已經即將抵達旅途的終點。`（6 個單位；來源 paratranz 6）
  2. `在經過貨櫃時，周圍空氣變得愈發寒涼。大塊的冰塊提示著我們已經即將抵達旅途的終點。`（2 個單位；來源 paratranz 2）
- 判讀：「愈發」與「更加」同義，單位數 6 對 2，統一成多數
- 建議：統一為 `在經過貨櫃時，周圍空氣變得更加寒涼。大塊的冰塊提示著我們已經即將抵達旅途的終點。`
- 會改：
  - `在經過貨櫃時，周圍空氣變得愈發寒涼。大塊的冰塊提示著我們已經即將抵達旅途的終點。` → `在經過貨櫃時，周圍空氣變得更加寒涼。大塊的冰塊提示著我們已經即將抵達旅途的終點。`：2 個單位（DungeonNode1-2.json×2）
    - 差異：`…時，周圍空氣變得〔愈發→更加〕寒涼。大塊的冰塊…`
  - 不改：`在經過貨櫃時，周圍空氣變得更加寒涼。大塊的冰塊提示著我們已經即將抵達旅途的終點。`（6）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G531　統一｜I4 同義措辭不同｜信心低　⚠ 低信心

`inconsistent/story-main/teller/2bc16c94`　story-main／teller，5 個單位，JP/EN 全同

- KR：`N사 직원`
- JP：`N社職員`　EN：`N Corp. Staff`
- 現有譯法：
  1. `N公司員工`（3 個單位；來源 claude 3）
  2. `N公司職員`（2 個單位；來源 claude 2）
- 判讀：「員工」與「職員」同義，同為 claude 譯，單位數 3 對 2，統一成多數
- 建議：統一為 `N公司員工`
- 會改：
  - `N公司職員` → `N公司員工`：2 個單位（StoryData/S1044B.json×2）
    - 差異：`N公司〔職→∅〕員`；`N公司職員〔∅→工〕`
  - 不改：`N公司員工`（3）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G532　統一｜I4 同義措辭不同｜信心中

`inconsistent/ui/content/316cf60f`　ui／content，5 個單位，JP/EN 全同

- KR：`[보상 교환] 페이지에서 보상을 수령할 수 있는 기간이 표시됩니다.`
- JP：`[報酬交換]ページに報酬を受け取れる期間が表示されます。`　EN：`The [Reward Exchange]'s open duration is shown here.`
- 現有譯法：
  1. `此處顯示了[獎勵兌換]的獎勵領取時間。`（4 個單位；來源 paratranz 4）
  2. `此處展示了[獎勵兌換]的開放時間。`（1 個單位；來源 paratranz 1）
- 判讀：兩者只差「顯示了…領取時間」與「展示了…開放時間」，KR 為「기간 … 표시」，取多數寫法
- 建議：統一為 `此處顯示了[獎勵兌換]的獎勵領取時間。`
- 會改：
  - `此處展示了[獎勵兌換]的開放時間。` → `此處顯示了[獎勵兌換]的獎勵領取時間。`：1 個單位（TutorialMainUIText.json）
    - 差異：`此處〔展→顯〕示了[獎勵兌換]…`；`…了[獎勵兌換]的〔開放→獎勵領取〕時間。`
  - 不改：`此處顯示了[獎勵兌換]的獎勵領取時間。`（4）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G533　統一｜I4 同義措辭不同｜信心中

`inconsistent/ui/content/3b612757`　ui／content，4 個單位，JP/EN 全同

- KR：`종착역에 입장하면 더이상 이전 역으로 되돌아갈 수 없기 때문에 신중한 선택이 필요합니다.`
- JP：`終着駅に入場すると以前の駅へは戻れなくなるので、慎重に選択する必要があります。`　EN：`You cannot go back to the previous Station once you enter the Terminus. Choose carefully.`
- 現有譯法：
  1. `進入終點站後將再也無法回到以往的站點，請您慎重考慮。`（2 個單位；來源 paratranz 2）
  2. `進入終點站後將無法回到之前的站點，請慎重選擇。`（2 個單位；來源 paratranz 2）
- 判讀：兩者意思相同，取較貼近 KR「이전 역」「신중한 선택」的「之前的站點，請慎重選擇」
- 建議：統一為 `進入終點站後將無法回到之前的站點，請慎重選擇。`
- 會改：
  - `進入終點站後將再也無法回到以往的站點，請您慎重考慮。` → `進入終點站後將無法回到之前的站點，請慎重選擇。`：2 個單位（TutorialMainUIText.json×2）
    - 差異：`進入終點站後將〔再也→∅〕無法回到以往的站…`；`…後將再也無法回到〔以往→之前〕的站點，請您慎重…`；`…到以往的站點，請〔您→∅〕慎重考慮。`；`…的站點，請您慎重〔考慮→選擇〕。`
  - 不改：`進入終點站後將無法回到之前的站點，請慎重選擇。`（2）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G534　統一｜I4 同義措辭不同｜信心中　⚠ 統一後是新寫法（不是現有任何一種）

`inconsistent/ui/content/4d0e55cc`　ui／content，4 個單位，JP/EN 全同

- KR：`편성한 인격의 체력, 정신력, 사망 상태가 표시됩니다.`
- JP：`編成した人格の体力・精神力、死亡状態が表示されます。`　EN：`The HP, SP, and availability of your Identities are shown here.`
- 現有譯法：
  1. `能夠表示編隊中人格的體力，理智值以及陣亡狀態。`（3 個單位；來源 paratranz 3）
  2. `編隊的人格、體力、理智值、死亡狀態將展示於此。`（1 個單位；來源 paratranz 1）
- 判讀：兩種結構不同，且第 1 種用逗號分隔體力與理智值；KR 為並列「체력, 정신력, 사망 상태」，統一用頓號並列、「死亡」與 KR 對應
- 建議：統一為 `能夠顯示編隊中人格的體力、理智值以及死亡狀態。`
- 會改：
  - `能夠表示編隊中人格的體力，理智值以及陣亡狀態。` → `能夠顯示編隊中人格的體力、理智值以及死亡狀態。`：3 個單位（TutorialMainUIText.json×3）
    - 差異：`能夠〔表→顯〕示編隊中人格的體…`；`…編隊中人格的體力〔，→、〕理智值以及陣亡狀…`；`…體力，理智值以及〔陣→死〕亡狀態。`
  - `編隊的人格、體力、理智值、死亡狀態將展示於此。` → `能夠顯示編隊中人格的體力、理智值以及死亡狀態。`：1 個單位（TutorialMainUIText.json）
    - 差異：`〔∅→能夠顯示〕編隊的人格、體力…`；`編隊〔的→中〕人格、體力、理智…`；`編隊的人格〔、→的〕體力、理智值、死…`；`…格、體力、理智值〔、→以及〕死亡狀態將展示於…`；`…理智值、死亡狀態〔將展示於此→∅〕。`
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G535　統一｜I4 同義措辭不同｜信心中

`inconsistent/ui/content/f467ccda`　ui／content，4 個單位，JP/EN 全同

- KR：`입장 방법`
- JP：`入場方法`　EN：`How to Enter`
- 現有譯法：
  1. `如何進入戰鬥`（2 個單位；來源 paratranz 2）
  2. `入場方式`（2 個單位；來源 paratranz 2）
- 判讀：「如何進入戰鬥」多了「戰鬥」，KR「입장 방법」只有入場方式，統一為「入場方式」
- 建議：統一為 `入場方式`
- 會改：
  - `如何進入戰鬥` → `入場方式`：2 個單位（TutorialMainUIText.json×2）
    - 差異：`〔如何進→∅〕入戰鬥`；`如何進入〔戰鬥→場方式〕`
  - 不改：`入場方式`（2）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G536　統一｜I4 同義措辭不同｜信心中

`inconsistent/ui/content/fa77a6c3`　ui／content，4 個單位，JP/EN 全同

- KR：`특정 스테이지부터 다시 시작할 경우 해당 스테이지 입장 시점의 인격 상태와 E.G.O 자원 보유량이 반영된 상태로 전투를 시작하며, 그 이후 스테이지의 클리어 기록이 초기화되어 사라집니다.`
- JP：`特定のステージから再挑戦する場合、該当するステージに入場した時点での人格状態とE.G.O資源保有量が反映された状態で戦闘が開始されます。それ以降のステージクリア記録は初期化されます。`　EN：`When you do so, the battle will start with the Sinners’ statuses and E.G.O resources reverted to that point, and any progress you've made since that Stage will be reset.`
- 現有譯法：
  1. `從特定關卡重新開始戰鬥時，罪人的狀態和罪孽碎片持有量將恢復到首次進入該關卡時的狀態。該關卡之後通過的所有關卡的完成進度也都將被重置。`（3 個單位；來源 paratranz 3）
  2. `若從特定關卡重新開始戰鬥，罪人的狀態和罪孽碎片持有量將恢復到首次進入該關卡時的狀態，並且在該關卡之後通過的所有關卡的完成進度都將被重置。`（1 個單位；來源 paratranz 1）
- 判讀：兩者內容相同，只差分成兩句或以「並且」連成一句，取多數的兩句寫法
- 建議：統一為 `從特定關卡重新開始戰鬥時，罪人的狀態和罪孽碎片持有量將恢復到首次進入該關卡時的狀態。該關卡之後通過的所有關卡的完成進度也都將被重置。`
- 會改：
  - `若從特定關卡重新開始戰鬥，罪人的狀態和罪孽碎片持有量將恢復到首次進入該關卡時的狀態，並且在該關卡之後通過的所有關卡的完成進度都將被重置。` → `從特定關卡重新開始戰鬥時，罪人的狀態和罪孽碎片持有量將恢復到首次進入該關卡時的狀態。該關卡之後通過的所有關卡的完成進度也都將被重置。`：1 個單位（TutorialMainUIText.json）
    - 差異：`〔若→∅〕從特定關卡重新開…`；`…關卡重新開始戰鬥〔∅→時〕，罪人的狀態和罪…`；`…入該關卡時的狀態〔，並且在→。〕該關卡之後通過的…`；`…有關卡的完成進度〔∅→也〕都將被重置。`
  - 不改：`從特定關卡重新開始戰鬥時，罪人的狀態和罪孽碎片持有量將恢復到首次進入該關卡時的狀態。該關卡之後通過的所有關卡的完成進度也都將被重置。`（3）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G537　統一｜I4 同義措辭不同｜信心中

`inconsistent/ui/content/129499b2`　ui／content，3 個單位，每種 JP/EN 只對到一種譯法

- KR：`미획득`
- 現有譯法：
  1. `未獲得`（2 個單位；來源 paratranz 2）
     - JP `未獲得`／EN `Not Owned`（1）
     - JP `未獲得`／EN `Unacquired`（1）
  2. `未擁有`（1 個單位；來源 paratranz 1）
     - JP `未獲得`／EN `Unowned`（1）
- 判讀：KR「미획득」，統一為「未獲得」（多數，也貼近 JP 未獲得）
- 建議：統一為 `未獲得`
- 會改：
  - `未擁有` → `未獲得`：1 個單位（MainUIText.json）
    - 差異：`未〔擁有→獲得〕`
  - 不改：`未獲得`（2）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G538　統一｜I4 同義措辭不同｜信心低　⚠ 低信心

`inconsistent/ui/content/16f64fe4`　ui／content，3 個單位，JP/EN 全同

- KR：`변경`
- JP：`変更`　EN：`Change`
- 現有譯法：
  1. `更換`（2 個單位；來源 paratranz 2）
  2. `變更`（1 個單位；來源 paratranz 1）
- 判讀：「更換」「變更」同義，取多數「更換」
- 建議：統一為 `更換`
- 會改：
  - `變更` → `更換`：1 個單位（SelectLoadingCgUi.json）
    - 差異：`〔變→∅〕更`；`變更〔∅→換〕`
  - 不改：`更換`（2）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G539　統一｜I4 同義措辭不同｜信心低　⚠ 低信心

`inconsistent/ui/content/1eac6595`　ui／content，3 個單位，每種 JP/EN 只對到一種譯法

- KR：`세이브 포인트`
- 現有譯法：
  1. `存檔點`（2 個單位；來源 paratranz 2）
     - JP `セーブポイント`／EN `Checkpoints`（2）
  2. `檢查點`（1 個單位；來源 paratranz 1）
     - JP `セーブポイント`／EN `Checkpoint`（1）
- 判讀：「存檔點」「檢查點」同義，教學文本多用「存檔點」，取多數
- 建議：統一為 `存檔點`
- 會改：
  - `檢查點` → `存檔點`：1 個單位（BattleUIText.json）
    - 差異：`〔檢查→存檔〕點`
  - 不改：`存檔點`（2）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G540　統一｜I4 同義措辭不同｜信心中

`inconsistent/ui/content/29563cff`　ui／content，3 個單位，每種 JP/EN 只對到一種譯法

- KR：`최초 E.G.O 기프트 선택`
- 現有譯法：
  1. `初始E.G.O飾品選擇`（2 個單位；來源 paratranz 2）
     - JP `初期E.G.Oギフト選択`／EN `Starting E.G.O Gift`（2）
  2. `選擇初始E.G.O飾品`（1 個單位；來源 paratranz 1）
     - JP `初期E.G.Oギフト選択`／EN `Starting E.G.O Gift Choice`（1）
- 判讀：兩者只差語序，取多數的「初始E.G.O飾品選擇」，貼近 KR 名詞語序
- 建議：統一為 `初始E.G.O飾品選擇`
- 會改：
  - `選擇初始E.G.O飾品` → `初始E.G.O飾品選擇`：1 個單位（TutorialMainUIText.json）
    - 差異：`〔選擇→∅〕初始E.G.O飾…`；`…始E.G.O飾品〔∅→選擇〕`
  - 不改：`初始E.G.O飾品選擇`（2）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G541　統一｜I4 同義措辭不同｜信心低　⚠ 低信心

`inconsistent/ui/content/2991f61c`　ui／content，3 個單位，部分譯法對到不同 JP/EN

- KR：`기타`
- 現有譯法：
  1. `雜項`（2 個單位；來源 paratranz 2）
     - JP `その他`／EN `Other`（1）
     - JP `その他`／EN `Misc.`（1）
  2. `其他`（1 個單位；來源 paratranz 1）
     - JP `その他`／EN `Other`（1）
- 判讀：「雜項」「其他」同義，取多數「雜項」
- 建議：統一為 `雜項`
- 會改：
  - `其他` → `雜項`：1 個單位（FormationNameEtcFilter.json）
    - 差異：`〔其他→雜項〕`
  - 不改：`雜項`（2）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G542　統一｜I4 同義措辭不同｜信心中

`inconsistent/ui/content/67d915c3`　ui／content，3 個單位，每種 JP/EN 只對到一種譯法

- KR：`편성 편집`
- 現有譯法：
  1. `編輯編隊`（2 個單位；來源 paratranz 2）
     - JP `編成を編集`／EN `Edit Team`（2）
  2. `編輯隊伍`（1 個單位；來源 paratranz 1）
     - JP `編成の編集`／EN `Edit Team`（1）
- 判讀：「編隊」在其他介面（전투 편성 → 編隊）已使用，統一為「編輯編隊」
- 建議：統一為 `編輯編隊`
- 會改：
  - `編輯隊伍` → `編輯編隊`：1 個單位（TutorialMainUIText.json）
    - 差異：`編輯〔∅→編〕隊伍`；`編輯隊〔伍→∅〕`
  - 不改：`編輯編隊`（2）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G543　統一｜I4 同義措辭不同｜信心低　⚠ 低信心

`inconsistent/ui/content/b01e7803`　ui／content，3 個單位，JP/EN 全同

- KR：`적 정보`
- JP：`敵情報`　EN：`Enemy Info`
- 現有譯法：
  1. `敵方資訊`（2 個單位；來源 paratranz 2）
  2. `敵方情報`（1 個單位；來源 paratranz 1）
- 判讀：「敵方資訊」「敵方情報」同義，取多數「敵方資訊」
- 建議：統一為 `敵方資訊`
- 會改：
  - `敵方情報` → `敵方資訊`：1 個單位（TutorialMainUIText.json）
    - 差異：`敵方〔情報→資訊〕`
  - 不改：`敵方資訊`（2）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G544　統一｜I4 同義措辭不同｜信心低　⚠ 低信心

`inconsistent/ui/content/cc58ffa2`　ui／content，3 個單位，JP/EN 全同

- KR：`전투 편성`
- JP：`戦闘編成`　EN：`Team Setup`
- 現有譯法：
  1. `編隊`（2 個單位；來源 paratranz 2）
  2. `戰鬥編隊`（1 個單位；來源 paratranz 1）
- 判讀：「編隊」「戰鬥編隊」同義，取多數「編隊」
- 建議：統一為 `編隊`
- 會改：
  - `戰鬥編隊` → `編隊`：1 個單位（TutorialMainUIText.json）
    - 差異：`〔戰鬥→∅〕編隊`
  - 不改：`編隊`（2）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G545　統一｜I4 同義措辭不同｜信心低　⚠ 低信心

`inconsistent/ui/content/239fe144`　ui／content，2 個單位，JP/EN 全同

- KR：`정신 소모량`
- JP：`精神消耗量`　EN：`Sanity Cost`
- 現有譯法：
  1. `理智值消耗`（1 個單位；來源 paratranz 1）
  2. `理智消耗量`（1 個單位；來源 paratranz 1）
- 判讀：「理智值消耗」「理智消耗量」同義，詞彙表 정신력 為「理智值」，取「理智值消耗」
- 建議：統一為 `理智值消耗`
- 會改：
  - `理智消耗量` → `理智值消耗`：1 個單位（TutorialMainUIText.json）
    - 差異：`理智〔∅→值〕消耗量`；`理智消耗〔量→∅〕`
  - 不改：`理智值消耗`（1）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G546　統一｜I4 同義措辭不同｜信心低　⚠ 低信心

`inconsistent/ui/content/253d1510`　ui／content，2 個單位，JP/EN 全同

- KR：`이름`
- JP：`名前`　EN：`Name`
- 現有譯法：
  1. `名稱`（1 個單位；來源 paratranz 1）
  2. `名字`（1 個單位；來源 paratranz 1）
- 判讀：「名稱」「名字」同義，介面標籤用「名稱」
- 建議：統一為 `名稱`
- 會改：
  - `名字` → `名稱`：1 個單位（UserInfo_Friends.json）
    - 差異：`名〔字→稱〕`
  - 不改：`名稱`（1）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G547　統一｜I4 同義措辭不同｜信心低　⚠ 低信心

`inconsistent/ui/content/3ceb8b2c`　ui／content，2 個單位，JP/EN 全同

- KR：`필요`
- JP：`必要`　EN：`Req.`
- 現有譯法：
  1. `需求`（1 個單位；來源 paratranz 1）
  2. `需要`（1 個單位；來源 paratranz 1）
- 判讀：「需求」「需要」同義，EN 為 Req.，取「需求」
- 建議：統一為 `需求`
- 會改：
  - `需要` → `需求`：1 個單位（BattleUIText.json）
    - 差異：`需〔要→求〕`
  - 不改：`需求`（1）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G548　統一｜I4 同義措辭不同｜信心中

`inconsistent/ui/content/3d7b681b`　ui／content，2 個單位，每種 JP/EN 只對到一種譯法

- KR：`전투 화면 내 정보량 조절 설정`
- 現有譯法：
  1. `戰鬥畫面資訊量設定`（1 個單位；來源 paratranz 1）
     - JP `戦闘UI内の情報量設定`／EN `Adjusting Density of Information on Screen During Combat`（1）
  2. `戰鬥介面資訊量設定`（1 個單位；來源 paratranz 1）
     - JP `戦闘UI内の情報量調整設定`／EN `Adjusting Density of Information on Screen During Combat`（1）
- 判讀：「畫面」「介面」同義，KR 為「화면」，統一為「戰鬥畫面資訊量設定」
- 建議：統一為 `戰鬥畫面資訊量設定`
- 會改：
  - `戰鬥介面資訊量設定` → `戰鬥畫面資訊量設定`：1 個單位（TutorialRenewalDesc.json）
    - 差異：`戰鬥〔介→畫〕面資訊量設定`
  - 不改：`戰鬥畫面資訊量設定`（1）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G549　統一｜I4 同義措辭不同｜信心低　⚠ 低信心

`inconsistent/ui/content/467adc61`　ui／content，2 個單位，JP/EN 全同

- KR：`추출 이력`
- JP：`抽出履歴`　EN：`Extraction History`
- 現有譯法：
  1. `提取紀錄`（1 個單位；來源 paratranz 1）
  2. `提取歷史`（1 個單位；來源 paratranz 1）
- 判讀：「紀錄」「歷史」同義，介面用「紀錄」
- 建議：統一為 `提取紀錄`
- 會改：
  - `提取歷史` → `提取紀錄`：1 個單位（TutorialMainUIText.json）
    - 差異：`提取〔歷史→紀錄〕`
  - 不改：`提取紀錄`（1）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G550　統一｜I4 同義措辭不同｜信心中

`inconsistent/ui/content/52e5063d`　ui／content，2 個單位，JP/EN 全同

- KR：`전투 이후 남은 E.G.O 자원은 다음 역에서 사용할 수 있습니다.`
- JP：`戦闘後に残ったE.G.O資源は、次の駅にて使用することができます。`　EN：`The E.G.O resources held after clearing a Station can be carried over to the next Station.`
- 現有譯法：
  1. `通關本場戰鬥後剩餘的罪孽碎片在下一站依舊能夠使用。`（1 個單位；來源 paratranz 1）
  2. `戰鬥後剩餘的罪孽碎片可在下一站中使用。`（1 個單位；來源 paratranz 1）
- 判讀：兩者意思相同，第 2 種更簡潔貼近 KR
- 建議：統一為 `戰鬥後剩餘的罪孽碎片可在下一站中使用。`
- 會改：
  - `通關本場戰鬥後剩餘的罪孽碎片在下一站依舊能夠使用。` → `戰鬥後剩餘的罪孽碎片可在下一站中使用。`：1 個單位（TutorialMainUIText.json）
    - 差異：`〔通關本場→∅〕戰鬥後剩餘的罪孽…`；`…後剩餘的罪孽碎片〔∅→可〕在下一站依舊能夠…`；`…罪孽碎片在下一站〔依舊能夠→中〕使用。`
  - 不改：`戰鬥後剩餘的罪孽碎片可在下一站中使用。`（1）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G551　統一｜I4 同義措辭不同｜信心低　⚠ 低信心

`inconsistent/ui/content/5e7daa10`　ui／content，2 個單位，JP/EN 全同

- KR：`이벤트 배너 획득 조건을 확인할 수 있습니다.`
- JP：`イベントバナーの獲得条件を確認することができます。`　EN：`You can check the requirements for acquiring banners here.`
- 現有譯法：
  1. `能夠確認活動橫幅的獲取條件。`（1 個單位；來源 paratranz 1）
  2. `可以在此確認活動橫幅的獲取條件。`（1 個單位；來源 paratranz 1）
- 判讀：兩者意思相同，第 2 種含「此處」，與 EN「here」對應
- 建議：統一為 `可以在此確認活動橫幅的獲取條件。`
- 會改：
  - `能夠確認活動橫幅的獲取條件。` → `可以在此確認活動橫幅的獲取條件。`：1 個單位（TutorialMainUIText.json）
    - 差異：`〔能夠→可以在此〕確認活動橫幅的獲…`
  - 不改：`可以在此確認活動橫幅的獲取條件。`（1）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G552　統一｜I4 同義措辭不同｜信心低　⚠ 低信心

`inconsistent/ui/content/60149824`　ui／content，2 個單位，JP/EN 全同

- KR：`공격 유형과 속성`
- JP：`攻撃タイプと属性`　EN：`Attack Types and Affinities`
- 現有譯法：
  1. `攻擊類型及屬性`（1 個單位；來源 paratranz 1）
  2. `攻擊類型和屬性`（1 個單位；來源 paratranz 1）
- 判讀：「及」「和」同義，取「和」較口語
- 建議：統一為 `攻擊類型和屬性`
- 會改：
  - `攻擊類型及屬性` → `攻擊類型和屬性`：1 個單位（TutorialDesc.json）
    - 差異：`攻擊類型〔及→和〕屬性`
  - 不改：`攻擊類型和屬性`（1）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G553　統一｜I4 同義措辭不同｜信心低　⚠ 低信心

`inconsistent/ui/content/70802961`　ui／content，2 個單位，JP/EN 全同

- KR：`보유 중`
- JP：`保有中`　EN：`Owned`
- 現有譯法：
  1. `已擁有`（1 個單位；來源 paratranz 1）
  2. `已持有`（1 個單位；來源 paratranz 1）
- 判讀：「已擁有」「已持有」同義；詞彙表「보유 패시브」為「持有被動」，取「已持有」
- 建議：統一為 `已持有`
- 會改：
  - `已擁有` → `已持有`：1 個單位（LobbyBGMUI.json）
    - 差異：`已〔擁→持〕有`
  - 不改：`已持有`（1）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G554　統一｜I4 同義措辭不同｜信心中

`inconsistent/ui/content/7531281d`　ui／content，2 個單位，每種 JP/EN 只對到一種譯法

- KR：`상대방의 친구 상한이 가득찼습니다.`
- 現有譯法：
  1. `該玩家的好友列表已滿。`（1 個單位；來源 paratranz 1）
     - JP `相手のフレンドリストが上限に達しました。`／EN `This user’s friend list is full.`（1）
  2. `對方的好友列表已滿。`（1 個單位；來源 paratranz 1）
     - JP `相手のフレンドリストが上限に達しました。`／EN `This user’s friend list became full.`（1）
- 判讀：KR「상대방」為對方，第 1 種「該玩家」不如「對方」貼近
- 建議：統一為 `對方的好友列表已滿。`
- 會改：
  - `該玩家的好友列表已滿。` → `對方的好友列表已滿。`：1 個單位（UserInfo_Friends.json）
    - 差異：`〔該玩家→對方〕的好友列表已滿。`
  - 不改：`對方的好友列表已滿。`（1）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G555　統一｜I4 同義措辭不同｜信心中　⚠ 統一後是新寫法（不是現有任何一種）　⚠ 整理時補上一項變更（見報告「整理時的更正」）

`inconsistent/ui/content/9cd88aa5`　ui／content，2 個單位，JP/EN 全同

- KR：`입장 시, 보유한 인격과 E.G.O를 선택하여 편성합니다.`
- JP：`入場時、保有している人格とE.G.Oを選択して編成します。`　EN：`You will choose a number of Identities and E.G.O to begin the expedition with.`
- 現有譯法：
  1. `進入關卡時，您可以選擇所擁有的人格與E.G.O來組建隊伍。`（1 個單位；來源 paratranz 1）
  2. `入場時，可以選擇所擁有的人格與E.G.O來進行組建隊伍。`（1 個單位；來源 paratranz 1）
- 判讀：兩者同義，統一為較簡潔、對應「입장 시」的寫法
- 建議：統一為 `入場時，可以選擇所擁有的人格與E.G.O來組建隊伍。`
- 會改：
  - `入場時，可以選擇所擁有的人格與E.G.O來進行組建隊伍。` → `入場時，可以選擇所擁有的人格與E.G.O來組建隊伍。`：1 個單位（TutorialMainUIText.json）
    - 差異：`…格與E.G.O來〔進行→∅〕組建隊伍。`
  - `進入關卡時，您可以選擇所擁有的人格與E.G.O來組建隊伍。` → `入場時，可以選擇所擁有的人格與E.G.O來組建隊伍。`：1 個單位（TutorialMainUIText.json）
    - 差異：`〔進→∅〕入關卡時，您可以…`；`進入〔關卡→場〕時，您可以選擇所…`；`進入關卡時，〔您→∅〕可以選擇所擁有的…`
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G556　統一｜I4 同義措辭不同｜信心中

`inconsistent/ui/content/9cdc6f1e`　ui／content，2 個單位，JP/EN 全同

- KR：`전투 참가 가능 수`
- JP：`参加可能人数`　EN：`Participant Limit`
- 現有譯法：
  1. `可參戰人數`（1 個單位；來源 paratranz 1）
  2. `戰鬥參與人數`（1 個單位；來源 paratranz 1）
- 判讀：「可參戰人數」貼近 KR「참가 가능 수」，統一
- 建議：統一為 `可參戰人數`
- 會改：
  - `戰鬥參與人數` → `可參戰人數`：1 個單位（TutorialMainUIText.json）
    - 差異：`〔∅→可參〕戰鬥參與人數`；`戰〔鬥參與→∅〕人數`
  - 不改：`可參戰人數`（1）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G557　統一｜I4 同義措辭不同｜信心中　⚠ 統一後是新寫法（不是現有任何一種）　⚠ 整理時補上一項變更（見報告「整理時的更正」）

`inconsistent/ui/content/a4bdca8d`　ui／content，2 個單位，JP/EN 全同

- KR：`역을 선택하여 입장합니다.`
- JP：`駅を選択して入場します。`　EN：`Select the Station to enter.`
- 現有譯法：
  1. `選擇站點並進入戰鬥。`（1 個單位；來源 paratranz 1）
  2. `選擇站點進入關卡。`（1 個單位；來源 paratranz 1）
- 判讀：KR 只說「역을 선택하여 입장」，第 1 種多了「戰鬥」，第 2 種多了「關卡」，皆為補譯；統一為不補的寫法
- 建議：統一為 `選擇站點並進入。`
- 會改：
  - `選擇站點進入關卡。` → `選擇站點並進入。`：1 個單位（TutorialMainUIText.json）
    - 差異：`選擇站點〔∅→並〕進入關卡。`；`選擇站點進入〔關卡→∅〕。`
  - `選擇站點並進入戰鬥。` → `選擇站點並進入。`：1 個單位（TutorialMainUIText.json）
    - 差異：`選擇站點並進入〔戰鬥→∅〕。`
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G558　統一｜I4 同義措辭不同｜信心中

`inconsistent/ui/content/a99b644e`　ui／content，2 個單位，每種 JP/EN 只對到一種譯法

- KR：`이용 약관`
- 現有譯法：
  1. `服務條款`（1 個單位；來源 paratranz 1）
     - JP `利用約款`／EN `Terms of Service`（1）
  2. `用戶協議`（1 個單位；來源 paratranz 1）
     - JP `利用約款`／EN `User Agreement`（1）
- 判讀：「服務條款」為臺灣慣用，「用戶協議」是大陸用語
- 建議：統一為 `服務條款`
- 會改：
  - `用戶協議` → `服務條款`：1 個單位（UserAgreements.json）
    - 差異：`〔用戶協議→服務條款〕`
  - 不改：`服務條款`（1）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G559　統一｜I4 同義措辭不同｜信心中　⚠ 統一後是新寫法（不是現有任何一種）

`inconsistent/ui/content/c53dec51`　ui／content，2 個單位，JP/EN 全同

- KR：`이어서 진행할 경우 등장하는 적의 체력이 유지되며, 남은 체력 비율이 역 위쪽에 표시됩니다.`
- JP：`引き続き進行する場合は登場する敵の体力が維持され、残存体力の比率が駅の上側に表示されます。`　EN：`Should you choose to continue, enemies' current HP will be saved. Their remaining HP percentages will be shown above the Station.`
- 現有譯法：
  1. `繼續進行戰鬥時，不會刷新敵方單位的現存體力，並將在站點上方顯示現存體力的比例條。`（1 個單位；來源 paratranz 1）
  2. `如果選擇繼續，敵人當前的生命值將被保留。其剩餘生命值將以百分比形式顯示在站點上方。`（1 個單位；來源 paratranz 1）
- 判讀：兩者譯法差異大：第 2 種用「生命值」「百分比」，第 1 種多譯「比例條」「不會刷新」；本專案 체력 為「體力」，統一為貼近 KR 的新寫法
- 建議：統一為 `繼續進行時，出現的敵人體力將會維持，剩餘體力比例會顯示在站點上方。`
- 會改：
  - `繼續進行戰鬥時，不會刷新敵方單位的現存體力，並將在站點上方顯示現存體力的比例條。` → `繼續進行時，出現的敵人體力將會維持，剩餘體力比例會顯示在站點上方。`：1 個單位（TutorialMainUIText.json）
    - 差異：`繼續進行〔戰鬥→∅〕時，不會刷新敵方…`；`繼續進行戰鬥時，〔不會刷新→出現的〕敵方單位的現存體…`；`…鬥時，不會刷新敵〔方單位的現存→人〕體力，並將在站點…`；`…方單位的現存體力〔∅→將會維持〕，並將在站點上方…`；`…單位的現存體力，〔並將→剩餘體力比例會顯示〕在站點上方顯示現…`；`…，並將在站點上方〔顯示現存體力的比例條→∅〕。`
  - `如果選擇繼續，敵人當前的生命值將被保留。其剩餘生命值將以百分比形式顯示在站點上方。` → `繼續進行時，出現的敵人體力將會維持，剩餘體力比例會顯示在站點上方。`：1 個單位（TutorialMainUIText.json）
    - 差異：`〔如果選擇→∅〕繼續，敵人當前的…`；`如果選擇繼續〔∅→進行時〕，敵人當前的生命…`；`如果選擇繼續，〔∅→出現的〕敵人當前的生命值…`；`…果選擇繼續，敵人〔當前的生命值→體力〕將被保留。其剩餘…`；`…人當前的生命值將〔被保留。其→會維持，〕剩餘生命值將以百…`；`…將被保留。其剩餘〔生命值將以百分→體力〕比形式顯示在站點…`；`…生命值將以百分比〔形式→例會〕顯示在站點上方。`
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G560　統一｜I4 同義措辭不同｜信心中

`inconsistent/ui/content/d6a8c525`　ui／content，2 個單位，JP/EN 全同

- KR：`상세 정보`
- JP：`詳細情報`　EN：`Details`
- 現有譯法：
  1. `詳情`（1 個單位；來源 paratranz 1）
  2. `更多情報`（1 個單位；來源 paratranz 1）
- 判讀：「詳情」「更多情報」同義，KR 為「상세 정보」，取簡短「詳情」
- 建議：統一為 `詳情`
- 會改：
  - `更多情報` → `詳情`：1 個單位（BattleUIText.json）
    - 差異：`〔更多→詳〕情報`；`更多情〔報→∅〕`
  - 不改：`詳情`（1）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G561　統一｜I4 同義措辭不同｜信心中

`inconsistent/ui/content/d7ccf7be`　ui／content，2 個單位，JP/EN 全同

- KR：`현재 계정`
- JP：`現在のアカウント`　EN：`Current Account`
- 現有譯法：
  1. `當前帳戶`（1 個單位；來源 paratranz 1）
  2. `當前帳號`（1 個單位；來源 paratranz 1）
- 判讀：「帳戶」「帳號」同義，遊戲慣用「帳號」，與 77 一致
- 建議：統一為 `當前帳號`
- 會改：
  - `當前帳戶` → `當前帳號`：1 個單位（IntegrateAccountUI.json）
    - 差異：`當前帳〔戶→號〕`
  - 不改：`當前帳號`（1）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G562　統一｜I4 同義措辭不同｜信心中

`inconsistent/ui/content/d81cc0e0`　ui／content，2 個單位，JP/EN 全同

- KR：`연동된 계정`
- JP：`連携済みアカウント`　EN：`Linked Accounts`
- 現有譯法：
  1. `已綁定帳號`（1 個單位；來源 paratranz 1）
  2. `關聯帳號`（1 個單位；來源 paratranz 1）
- 判讀：「已綁定帳號」「關聯帳號」同義，JP「連携済み」＝已綁定，取「已綁定帳號」
- 建議：統一為 `已綁定帳號`
- 會改：
  - `關聯帳號` → `已綁定帳號`：1 個單位（kr_settings-ui-donttranslate.json）
    - 差異：`〔關聯→已綁定〕帳號`
  - 不改：`已綁定帳號`（1）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G563　統一｜I4 同義措辭不同｜信心中

`inconsistent/ui/content/e47da70c`　ui／content，2 個單位，JP/EN 全同

- KR：`획득 조건이 어려울수록 더욱 특별한 배너를 얻게 됩니다.`
- JP：`獲得条件が難しいほど、より特別なバナーを獲得できます。`　EN：`You can acquire more lavishly ornamented banners as you meet more difficult conditions.`
- 現有譯法：
  1. `獲取條件越為苛刻，所能獲得的裝飾橫幅越為特別。`（1 個單位；來源 paratranz 1）
  2. `獲取條件越苛刻，獲得的橫幅越特別。`（1 個單位；來源 paratranz 1）
- 判讀：第 1 種多了「裝飾」「所能」，第 2 種更貼近 KR
- 建議：統一為 `獲取條件越苛刻，獲得的橫幅越特別。`
- 會改：
  - `獲取條件越為苛刻，所能獲得的裝飾橫幅越為特別。` → `獲取條件越苛刻，獲得的橫幅越特別。`：1 個單位（TutorialMainUIText.json）
    - 差異：`獲取條件越〔為→∅〕苛刻，所能獲得的…`；`…取條件越為苛刻，〔所能→∅〕獲得的裝飾橫幅越…`；`…苛刻，所能獲得的〔裝飾→∅〕橫幅越為特別。`；`…獲得的裝飾橫幅越〔為→∅〕特別。`
  - 不改：`獲取條件越苛刻，獲得的橫幅越特別。`（1）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G564　統一｜I4 同義措辭不同｜信心中

`inconsistent/ui/content/f1bc419c`　ui／content，2 個單位，JP/EN 全同

- KR：`추출 종류`
- JP：`抽出種類`　EN：`Category`
- 現有譯法：
  1. `類型`（1 個單位；來源 paratranz 1）
  2. `提取類型`（1 個單位；來源 paratranz 1）
- 判讀：KR「추출 종류」，第 2 種「提取類型」完整對應；第 1 種漏「提取」
- 建議：統一為 `提取類型`
- 會改：
  - `類型` → `提取類型`：1 個單位（MainUIText.json）
    - 差異：`〔∅→提取〕類型`
  - 不改：`提取類型`（1）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G565　統一｜I4 同義措辭不同｜信心中

`inconsistent/ui/desc/50a83b2f`　ui／desc，4 個單位，JP/EN 全同

- KR：`시즌 2 레미니센스 기간 내에 순환 횟수 5 이상, 누적 턴 수 200턴 이하로 2호선 클리어 시 획득`
- JP：`シーズン2 レミニセンス期間中に循環回数5以上、累計ターン数200以下で2号線クリア時に獲得`　EN：`Obtained by clearing Line 2 with 5 or more Cycle Counts and 200 or less Total Turns before Season 2: Reminiscence ends`
- 現有譯法：
  1. `第2賽季 追憶期間內，以巡迴數5以上，總計回合數200以下通關2號線時獲得`（3 個單位；來源 paratranz 3）
  2. `在第2賽季 追憶 期間以不少於5的巡迴數，不多於200的累計回合數通關2號線後獲得`（1 個單位；來源 paratranz 1）
- 判讀：只差語序與用字（「總計回合數200以下」「不少於…不多於」），取多數的簡潔寫法
- 建議：統一為 `第2賽季 追憶期間內，以巡迴數5以上，總計回合數200以下通關2號線時獲得`
- 會改：
  - `在第2賽季 追憶 期間以不少於5的巡迴數，不多於200的累計回合數通關2號線後獲得` → `第2賽季 追憶期間內，以巡迴數5以上，總計回合數200以下通關2號線時獲得`：1 個單位（UserBanner.json）
    - 差異：`〔在→∅〕第2賽季␣追憶␣…`；`在第2賽季␣追憶〔␣→∅〕期間以不少於5的…`；`…賽季␣追憶␣期間〔∅→內，〕以不少於5的巡迴…`；`…季␣追憶␣期間以〔不少於5的→∅〕巡迴數，不多於2…`；`…不少於5的巡迴數〔∅→5以上〕，不多於200的…`；`…少於5的巡迴數，〔不多於200的累→總〕計回合數通關2號…`；`…00的累計回合數〔∅→200以下〕通關2號線後獲得`；`…回合數通關2號線〔後→時〕獲得`
  - 不改：`第2賽季 追憶期間內，以巡迴數5以上，總計回合數200以下通關2號線時獲得`（3）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G566　統一｜I4 同義措辭不同｜信心中

`inconsistent/ui/desc/6d5f3ca7`　ui／desc，4 個單位，JP/EN 全同

- KR：`순환 횟수 5 이상으로 2호선 클리어 시 획득`
- JP：`循環回数5以上で、2号線クリア時に獲得`　EN：`Obtained by clearing Line 2 with 5 or more Cycle Counts`
- 現有譯法：
  1. `以巡迴數5以上通關2號線時獲得`（3 個單位；來源 paratranz 3）
  2. `以不少於5的巡迴數通關2號線後獲得`（1 個單位；來源 paratranz 1）
- 判讀：只差語序與用字，取多數寫法
- 建議：統一為 `以巡迴數5以上通關2號線時獲得`
- 會改：
  - `以不少於5的巡迴數通關2號線後獲得` → `以巡迴數5以上通關2號線時獲得`：1 個單位（UserBanner.json）
    - 差異：`以〔不少於5的→∅〕巡迴數通關2號線…`；`…不少於5的巡迴數〔∅→5以上〕通關2號線後獲得`；`…巡迴數通關2號線〔後→時〕獲得`
  - 不改：`以巡迴數5以上通關2號線時獲得`（3）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出
