# 關鍵字與狀態（keywords）

## 文本類型

- 狀態 `Bufs*` 與戰鬥關鍵字 `BattleKeywords*`：名稱 `name`、說明 `desc`、摘要 `summary`、`flavor`
- 恐慌 `PanicInfo*`：`panicName`、`panicDescription`、`lowMoraleDescription`；另有 `UnitKeyword*`、`MentalCondition*` 等

## 格式慣例

- **恐慌**：說明裡的標題行 `- 패닉` 寫「- 陷入恐慌」
- **條列的「- 」**：說明各條開頭的「- 」照 KR 保留，條數相同
- **`- 최댓값: N`**：寫「- 最大值：N」
- **`수치 N당`**（本效果的層數）：寫「每帶有N層本效果」或「每有N層」
- **`공격 레벨 ±N`**：寫「攻擊等級±N」，正負號照 KR
- **`적중 시`**：寫「命中時」

## 常見錯誤

- **狀態名稱的〜寫成～**：照 KR 寫，不換成外觀相近的字元，例如 `ConfessionThirdPhase` 的名稱是「〜」（U+301C），不是數值範圍用的「～」（U+FF5E）。遊戲會把技能、被動、E.G.O飾品、恐慌說明裡與狀態名稱相同、後面接數字或空白的字串變成關鍵字連結；名稱寫成「～」，「1～2層」這類範圍就會多出圖示
