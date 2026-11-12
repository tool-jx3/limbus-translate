# I4 同義措辭不同：交給你決定（G570–G572）

[回這一類的目錄](I4.md)。

### G570　交給你決定｜I4 同義措辭不同｜信心低　⚠ 低信心

`inconsistent/enemy/desc/3330fad8`　enemy／desc，2 個單位，JP/EN 全同

- KR：`공격 스킬 종료시, 각 대상에게 (크리티컬로 적중한 참격 코인 개수 x 10)만큼 참격 피해 (해당 피해로는 흐트러지지 않음)`
- JP：`攻撃スキル終了時、各対象へ(クリティカル的中した斬撃コイン個数 × 10)だけ斬撃ダメージ (このダメージでは混乱しない)`　EN：`Attack Skill End: Deal (# of Slash Coins that hit Critically x 10) Slash damage against each target (cannot be Staggered due to this damage)`
- 現有譯法：
  1. `自身在攻擊結束後，對各個攻擊對象造成(造成暴擊的斬擊硬幣數量×10)的斬擊傷害 (此被動造成的傷害無法使目標陷入混亂)`（1 個單位；來源 paratranz 1）
  2. `攻擊技能結束時，對所有目標造成(暴擊命中的硬幣數×10)點的斬擊傷害(本效果造成的傷害不會使目標陷入混亂)`（1 個單位；來源 paratranz 1）
- 判讀：兩種措辭各有優缺：A 加了「自身」「此被動」，B 把「各」改成「所有」並多了「點」，各有偏離 KR
- 建議：交給你決定，選項：
  1. **採 A**：B 改為 A
  2. **採 B**：A 改為 B
  3. **重翻**：依 KR 重翻：攻擊技能結束時，對各個目標造成(暴擊命中的斬擊硬幣數量×10)的斬擊傷害(本效果造成的傷害不會使目標陷入混亂)
- 同意後寫入：依你選的選項；選「保留」寫 `rules/declined.yaml`，選改寫則每個改動寫一個 `rules/rewrite/` 改寫檔（寫法同「統一」）；選項寫「須擬稿」的，先給你看擬好的譯文

### G571　交給你決定｜I4 同義措辭不同｜信心低　⚠ 低信心

`inconsistent/enemy/desc/59f43b70`　enemy／desc，2 個單位，JP/EN 全同

- KR：`본체 체력이 10% 남았을 때, 패턴이 변경됨.`
- JP：`本体体力が残り10%なら、パターンが変わる。`　EN：`At 10% Core HP, gains a new pattern.`
- 現有譯法：
  1. `現存體力不高於10%時，改變自身模式。`（1 個單位；來源 paratranz 1）
  2. `剩餘10%體力時，改變自身模式。`（1 個單位；來源 paratranz 1）
- 判讀：「現存體力不高於10%時」與「剩餘10%體力時」語意可能不同（KR「10% 남았을 때」是否含以下不明）
- 建議：交給你決定，選項：
  1. **採 A（不高於10%）**：B 改為 A
  2. **採 B（剩餘10%）**：A 改為 B
- 同意後寫入：依你選的選項；選「保留」寫 `rules/declined.yaml`，選改寫則每個改動寫一個 `rules/rewrite/` 改寫檔（寫法同「統一」）；選項寫「須擬稿」的，先給你看擬好的譯文

### G572　交給你決定｜I4 同義措辭不同｜信心低　⚠ 低信心

`inconsistent/enemy/desc/dbecfa16`　enemy／desc，2 個單位，JP/EN 全同

- KR：`스테이지 시작시 [HeatingWireOffSpider]를 가지고 시작함⏎⏎턴 시작 시 자신이 보유한 [GotACompliment]가 3 이상이면 해당 턴 동안 [HeatingWireOnSpider]으로 바뀌고, [Laceration]을 부여하는 스킬로 [Combustion]을 추가로 부여함⏎⏎턴 종료시 자신이 보유한 [GotACompliment]가 3 미만이면 [HeatingWireOnSpider]이 [HeatingWireOffSpider]로 전환됨`
- JP：`ステージ開始時、[HeatingWireOffSpider]を持って始める⏎⏎ターン開始時、自分が保有した[GotACompliment]が3以上なら、該当ターンの間[HeatingWireOnSpider]に代わり、[Laceration]を付与するスキルで[Combustion]を追加で付与する⏎⏎ターン終了時、自分が保有した[GotACompliment]が3未満なら[HeatingWireOnSpider]が[HeatingWireOffSpider]に変換される`　EN：`Begins Encounter with [HeatingWireOffSpider]⏎⏎Turn Start: At 3+ [GotACompliment], gain [HeatingWireOnSpider] for the turn and also inflict [Combustion] with Skills that inflict [Laceration]⏎⏎Turn End: At less than 3 [GotACompliment], convert this unit's [HeatingWireOnSpider] to [HeatingWireOffSpider]`
- 現有譯法：
  1. `關卡開始時，使自身帶有 [HeatingWireOffSpider]⏎⏎回合開始時，若自身的 [GotACompliment] 層數不低於3層，則本回合使自身獲得 [HeatingWireOnSpider] 且使自身施加 [Laceration] 的技能額外施加 [Combustion]⏎⏎回合結束時，若自身 [GotACompliment] 的層數低於3層，則使自身的 [HeatingWireOnSpider] 轉變為 [HeatingWireOffSpider]`（1 個單位；來源 claude 1）
  2. `關卡開始時，使自身獲得 [HeatingWireOffSpider]⏎⏎回合開始時，若自身 [GotACompliment] 的層數不低於3層，則本回合內使自身獲得 [HeatingWireOnSpider]；使自身施加 [Laceration] 的攻擊技能額外施加 [Combustion]⏎⏎回合結束時，若自身 [GotACompliment] 的層數少於3，則使自身的 [HeatingWireOnSpider] 效果改為 [HeatingWireOffSpider]`（1 個單位；來源 paratranz 1）
- 判讀：兩種語序、動詞各異（帶有／獲得；層數低於3層／少於3；轉變為／改為），B 的「攻擊技能」比 KR 多「攻擊」
- 建議：交給你決定，選項：
  1. **採 A（claude 譯）**：B 全部改為 A
  2. **採 B（paratranz）**：A 全部改為 B
- 同意後寫入：依你選的選項；選「保留」寫 `rules/declined.yaml`，選改寫則每個改動寫一個 `rules/rewrite/` 改寫檔（寫法同「統一」）；選項寫「須擬稿」的，先給你看擬好的譯文
