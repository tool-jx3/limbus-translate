# 鏡牢（mirror）

## 文本類型

- E.G.O 飾品 `EGOgift*`（29 檔）：名稱 `name`、說明 `desc`、簡述 `simpleDesc`；分類 `EgoGiftCategory`（routing.yaml；raw-20261105 檔數）
- 事件 `ActionEvents*`（39 檔）：事件敘述 `desc`、選項 `message`、選項效果 `messageDesc`、結果 `result`（routing.yaml；raw-20261105 檔數與欄位）
- 鏡牢介面 `MirrorDungeon*`（20 檔）、起始增益 `DungeonStartBuffs*`（3 檔）、`TutorialMirrorDungeon`、`StoryTheaterMirrorWorld*`（routing.yaml；raw-20261105 檔數）

## 語氣與人稱

（目前無足夠證據）

## 格式慣例

- 選項效果 `선택 시, …` 寫「選擇後，……」（既有 129／147，例 `ActionEvents-a1c8p2.json|/dataList/7/options/0/messageDesc`）
- 選項效果的 `E.G.O GIFT 획득`（KR 沒寫數量）寫「獲得1件……E.G.O飾品」（既有 310／360，例 `ActionEvents.json|/dataList/37/options/1/messageDesc`）
- 選項效果 `별도 효과 없음` 寫「沒有效果」（既有 44／44，例 `ActionEvents_twth.json|/dataList/0/options/2/messageDesc`）
- `… 판정 성공 시` 寫「……判定 成功後」（既有 122／146，例 `ActionEvents.json|/dataList/37/options/0/messageDesc`）
- 飾品說明的 `피해량 +N%` 寫「造成的傷害+N%」、`최종 위력 ±N` 寫「最終威力±N」（正負號照 KR）（既有 195／227、35／38，例 `EGOgift_MirrorDungeon-EventTheme.json|/dataList/20/simpleDesc/0/simpleDesc`、`EGOgift_MirrorDungeon.json|/dataList/323/simpleDesc/0/simpleDesc`）
- `아군` 寫「友方單位」（既有 357／378，例 `EGOgift_StoryDungeon.json|/dataList/7/desc`）
- KR 方括號裡的韓文名稱譯成中文，方括號照樣保留（`[…]`）（既有 102／102，例 `MirrorDungeonUI_7.json|/dataList/28/content`）

## 常見錯誤

- 漏掉 KR 的標籤：有 205 行 KR 帶某種標籤（如 `<style>`）而舊譯少了，多在 `EGOgift_MirrorDungeon*`（base.md「補充規則」的「標籤」）
- 舊譯曾把 `[ID]` 改寫成「中文＋空格」，`EGOgift*` 有 56 處；KR 寫 `[ID]` 的地方一律保留（base.md「資料補充」第 5 條、「補充規則」）
- 修正 draft 時刪掉了中文關鍵字名稱上的 `<noparse>`，後由人工還原；規則見 base.md「`<noparse>`」（rules/adjustments/2026-11-04-noparse.yaml，本族群 3 筆，例 `EGOgift_MirrorDungeon-EventTheme_2.json|/dataList/25/desc`）
