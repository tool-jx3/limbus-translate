# 戰鬥說明共用指引（ally、enemy、keywords、mirror）

技能、被動、狀態、E.G.O 飾品與鏡牢事件的說明共用的寫法。

## 格式慣例

- **效果文字的括號**：括號的一般規則見 base.md「括號依文本類型」。效果文字用半形：`적안(開)` → 赤瞳(開)；韓文音譯後附原文的（`데가제(Dégagé)`、`라 노제(la nausée)`）寫「中文意思（原文）」用全形：`데가제(Dégagé)` → 脫離（Dégagé）
  - mirror 的介面、教學、目錄與事件敘述（`MirrorDungeonUI*`、`TutorialMirrorDungeon`、`StoryTheaterMirrorWorld`、`ActionEvents` 的 `desc`／`message`／`result`／`name`）不是效果文字，用全形
- **`(턴당 N회)`**：寫「(每回合最多N次)」
- **`최종 위력 ±N`**：寫「最終威力±N」，正負號照 KR
- **恢復**：`정신력 N 회복` 寫「恢復N點理智值」；`체력 N 회복` 寫「恢復N點體力」
- **`아군`**：寫「友方單位」
- **`처치 시`**：寫「擊殺……時」或「擊殺……後」
