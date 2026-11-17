# 我方技能與人格（ally）

## 文本類型

- 人格的基本技能 `Skills.json`（另含防禦、迴避、反擊的通用技能）、人格技能 `Skills_personality-YY`、E.G.O 技能 `Skills_Ego_Personality-YY` 與 `Skills_Ego*`：技能與硬幣說明 `desc`、技能名 `name`、E.G.O 的異想體名 `abName`
- 被動 `Passives*`、`Passive_Ego*`（`Passive-*` 與 `Passives_fools` 是敵方的，歸 enemy）、協助者 `Skills_Assist*`／`Passives_Assist*`／`Assist*`、人格與 E.G.O 的名稱與取得條件 `Personalit*`、`Egos*`、`EGO*`

## 格式慣例

- **`[X] N 부여`**（X 為 Combustion、Laceration、Vibration、Burst、Sinking）：寫「使目標增加N級 [X] 強度」
- **`피해량 +N%`**：寫「造成的傷害+N%」

## 常見錯誤

- **把 `[ID]` 改寫成「中文＋空格」**：KR 寫 `[ID]` 的地方一律保留（base.md「`[ID]` 一律保留」）
- **修正 draft 時刪掉 `<noparse>`**：中文關鍵字名稱上已有的 `<noparse>` 不要刪，規則見 `guides/shared/skill-text.md`「`<noparse>`」
