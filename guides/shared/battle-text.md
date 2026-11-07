# 戰鬥說明共用指引（ally、enemy、keywords、mirror）

技能、被動、狀態、E.G.O 飾品與鏡牢事件的說明共用的寫法；這四個族群在 `routing.yaml` 以 `include:` 引入。「既有 a／b」是四個族群合計。

## 格式慣例

- 括號用半形「()」，不照 base.md 第 7 條的全形（既有 6,787／6,916：譯文有括號的單位，除了下一條的寫法都只用半形，例 `Skills_Ego.json|/dataList/76/levelList/0/name`「赤瞳(開)」）
  - 例外：KR 在韓文音譯後的括號附原文（`데가제(Dégagé)`、`라 노제(la nausée)`）時，照 base.md「KR 已在括號附上原文」寫「中文意思（原文）」，用全形（既有 122／129，例 `Skills_Abnormality-a1c10p3.json|/dataList/60/levelList/0/name`「脫離（Dégagé）」）
- `(턴당 N회)` 寫「(每回合最多N次)」（既有 618／633，例 `Passives-BossRaid.json|/dataList/14/desc`）
- `최종 위력 ±N` 寫「最終威力±N」，正負號照 KR（既有 755／792，例 `Skills_personality-06.json|/dataList/4/levelList/2/desc`）
- `정신력 N 회복` 寫「恢復N點理智值」；`체력 N 회복` 寫「恢復N點體力」（既有 630／642、141／141，例 `Passives.json|/dataList/15/desc`、`Passives_Enemy.json|/dataList/0/desc`）
- `아군` 寫「友方單位」（既有 1,772／1,857，例 `Passives.json|/dataList/0/desc`）
- `처치 시` 寫「擊殺……時」或「擊殺……後」（既有 115／120，例 `Passives.json|/dataList/106/desc`、`SkillTag.json|/dataList/45/name`）
