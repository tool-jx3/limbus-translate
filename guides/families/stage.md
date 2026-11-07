# 關卡與場景（stage）

## 文本類型

- 關卡節點 `StageNode*`（42 檔）：關卡名 `title`、地點 `place`、說明 `desc`（routing.yaml；raw-20261105 檔數）
- 劇情說話者名稱 `ScenarioModelCodes`（`name`、`nickName`）；選擇事件 `ChoiceEvent*`（8 檔）；戰鬥提示 `BattleResultHint*`、`BattleHint*`（11 檔）；地牢區域 `DungeonArea*`、`DungeonName`（7 檔）；劇情目錄 `StoryTheater*`（4 檔）與其他（routing.yaml；raw-20261105 檔數）

## 語氣與人稱

（目前無足夠證據）

## 格式慣例

- `클리어` 寫「通關」（既有 40／47，例 `StageStatisticUIText.json|/dataList/0/content`）
- KR 方括號裡的韓文名稱譯成中文，方括號照樣保留（`[…]`）（既有 39／39，例 `HellsChicken.json|/dataList/7/content`）

## 常見錯誤

- draft 或舊譯可能是錯位的別列內容（Paratranz 錯位），先逐句對照 KR：`StoryTheaterPersonality` 有 2 列放了上一列的角色資料，已由人工修正（rules/adjustments/2026-11-04-review-page-fixes.yaml，例 `StoryTheaterPersonality.json|/dataList/237/content`）
