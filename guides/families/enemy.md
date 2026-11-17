# 敵方技能與名稱（enemy）

## 文本類型

- 敵人名稱與說明 `Enemies*`（`name`、`desc`）
- 敵人與異想體戰鬥的技能 `Skills_Enemy*`、`Skills_Abnormality*`、`Skills_Tutorial`／`Skills_fools`：技能與硬幣說明 `desc`、技能名 `name`
- 被動 `Passives_Enemy*`、`Passives_Abnormality*`、`Passive-*`、`Passives_fools`：`name`、`desc`，另有 `flavor`

## 格式慣例

- **`[X] N 부여`**（X 為 Combustion、Laceration、Vibration、Burst、Sinking）：寫「使目標增加N級 [X] 強度」
- **`피해량 +N%`**：寫「造成的傷害+N%」

## 常見錯誤

- **把 `[ID]` 改寫成「中文＋空格」**：KR 寫 `[ID]` 的地方一律保留（base.md「`[ID]` 一律保留」）
- **修正 draft 時刪掉 `<noparse>`**：中文關鍵字名稱上已有的 `<noparse>` 不要刪，規則見 `guides/shared/skill-text.md`「`<noparse>`」
