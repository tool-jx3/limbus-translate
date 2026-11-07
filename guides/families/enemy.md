# 敵方技能與名稱（enemy）

## 文本類型

- 敵人名稱與說明 `Enemies*`（46 檔，`name`、`desc`）（routing.yaml；raw-20261105 檔數）
- 敵人與異想體戰鬥的技能 `Skills_Enemy*`（34 檔）、`Skills_Abnormality*`（50 檔）、`Skills`／`Skills_Tutorial`／`Skills_fools`（3 檔）：技能與硬幣說明 `desc`、技能名 `name`（routing.yaml；raw-20261105 檔數）
- 被動 `Passives_Enemy*`（26 檔）、`Passives_Abnormality*`（46 檔）：`name`、`desc`，另有 `flavor`（routing.yaml；raw-20261105 檔數）

## 語氣與人稱

（目前無足夠證據）

## 格式慣例

- `[X] N 부여`（X 為 Combustion、Laceration、Vibration、Burst、Sinking）寫「使目標增加N級 [X] 強度」（既有 1,729／1,810，例 `Skills_Enemy-tkt.json|/dataList/20/levelList/0/coinlist/0/coindescs/1/desc`）
- `피해량 +N%` 寫「造成的傷害+N%」（既有 615／646，例 `Passives_Enemy-a1c6p1.json|/dataList/0/summary`）

## 常見錯誤

- 舊譯曾把 `[ID]` 改寫成「中文＋空格」：共 1,088 處，幾乎都在 `Skills*`（842）、`Passives*`（157）；KR 寫 `[ID]` 的地方一律保留（base.md「資料補充」第 5 條、「補充規則」）
- 修正 draft 時刪掉了中文關鍵字名稱上的 `<noparse>`，後由人工還原；規則見 base.md「`<noparse>`」（rules/adjustments/2026-11-04-noparse.yaml，本族群 7 筆，例 `Passives_Abnormality-mr8.json|/dataList/4/desc`）
