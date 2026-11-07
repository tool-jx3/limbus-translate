# 我方技能與人格（ally）

## 文本類型

- 人格技能 `Skills_personality-YY`（15 檔）、E.G.O 技能 `Skills_Ego_Personality-YY`（14 檔）與 `Skills_Ego*`（2 檔）：技能與硬幣說明 `desc`、技能名 `name`、E.G.O 的異想體名 `abName`（routing.yaml；raw-20261105 檔數）
- 被動 `Passive*`（16 檔）、協助者 `Skills_Assist*`／`Passives_Assist*`／`Assist*`（26 檔）、人格與 E.G.O 的名稱與取得條件 `Personalit*`、`Egos*`、`EGO*`（7 檔）（routing.yaml；raw-20261105 檔數）

## 語氣與人稱

（目前無足夠證據）

## 格式慣例

- `[X] N 부여`（X 為 Combustion、Laceration、Vibration、Burst、Sinking）寫「使目標增加N級 [X] 強度」（既有 1,691／1,776，例 `Skills_personality-12.json|/dataList/8/levelList/0/desc`）
- `(턴당 N회)` 寫「(每回合最多N次)」（既有 412／414，例 `Passives-BossRaid.json|/dataList/14/desc`）
- `피해량 +N%` 寫「造成的傷害+N%」（既有 756／807，例 `Passives.json|/dataList/3/desc`）
- `최종 위력 +N` 寫「最終威力+N」（既有 227／239，例 `Skills_personality-06.json|/dataList/4/levelList/2/desc`）
- `정신력 N 회복` 寫「恢復N點理智值」（既有 201／209，例 `Passives.json|/dataList/15/desc`）；`체력 N 회복` 寫「恢復N點體力」（既有 47／47，例 `Passives.json|/dataList/35/desc`）
- `아군` 寫「友方單位」（既有 810／844，例 `Passives.json|/dataList/0/desc`）
- `처치 시` 寫「擊殺……時」或「擊殺……後」（既有 47／50，例 `Passives.json|/dataList/106/desc`）

## 常見錯誤

- 舊譯曾把 `[ID]` 改寫成「中文＋空格」：共 1,088 處，幾乎都在 `Skills*`（842）、`Passives*`（157）；KR 寫 `[ID]` 的地方一律保留（base.md「資料補充」第 5 條、「補充規則」）
- 修正 draft 時刪掉了中文關鍵字名稱上的 `<noparse>`，後由人工還原；規則見 base.md「`<noparse>`」（rules/adjustments/2026-11-04-noparse.yaml，本族群 8 筆，例 `Passives.json|/dataList/259/desc`）
