# 鏡牢（mirror）

## 文本類型

- E.G.O 飾品 `EGOgift*`：名稱 `name`、說明 `desc`、簡述 `simpleDesc`；分類 `EgoGiftCategory`
- 事件 `ActionEvents*`：事件敘述 `desc`、選項 `message`、選項效果 `messageDesc`、結果 `result`
- 鏡牢介面 `MirrorDungeon*`、起始增益 `DungeonStartBuffs*`、`TutorialMirrorDungeon`、`StoryTheaterMirrorWorld*`

## 格式慣例

- **`선택 시, …`**：選項效果寫「選擇後，……」
- **`E.G.O GIFT 획득`**（KR 沒寫數量）：選項效果寫「獲得1件……E.G.O飾品」
- **`별도 효과 없음`**：選項效果寫「沒有效果」
- **`… 판정 성공 시`**：寫「……判定 成功後」
- **`피해량 +N%`**：飾品說明寫「造成的傷害+N%」

## 常見錯誤

- **漏掉 KR 的標籤**：KR 帶某種標籤（如 `<style>`）而譯文少了，多在 `EGOgift_MirrorDungeon*`（base.md「代碼與標籤」的「種類與數量同 KR」）
- **把 `[ID]` 改寫成「中文＋空格」**：KR 寫 `[ID]` 的地方一律保留（base.md「`[ID]` 一律保留」）
- **修正 draft 時刪掉 `<noparse>`**：中文關鍵字名稱上已有的 `<noparse>` 不要刪，規則見 `guides/shared/skill-text.md`「`<noparse>`」
