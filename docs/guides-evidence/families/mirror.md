# `guides/families/mirror.md` 的依據

標 raw-20261105 的「既有 a／b」與「全在本族群」檢查由 `docs/research/scripts/family_guides.py` 算出（「既有 a／b」的門檻見 `guides/README.md`「新增慣例的門檻」）；標 raw-20261105 的文本類型檔案數來自 `guides/routing.yaml` 與 `guides/families.md`；沒有標資料版本的，是官方 2026-10-29 語言包與當時的既有繁中（依寫入日期 2026-10-31～11-02 推定；`docs/guides-evidence/README.md`「資料依據」）。

## 文本類型

- **統計**（raw-20261105）：`EGOgift*` 29 檔、`ActionEvents*` 39 檔、`MirrorDungeon*` 20 檔、`DungeonStartBuffs*` 3 檔；`ActionEvents*` 的欄位也依 raw-20261105
- **出處**：`guides/routing.yaml`

## 格式慣例

### `선택 시, …`

- **統計**（raw-20261105）：既有 129／147
- **出處**：`ActionEvents-a1c8p2.json|/dataList/7/options/0/messageDesc`

### `E.G.O GIFT 획득`

- **統計**（raw-20261105）：既有 310／360
- **出處**：`ActionEvents.json|/dataList/37/options/1/messageDesc`

### `별도 효과 없음`

- **統計**（raw-20261105）：既有 44／44
- **出處**：`ActionEvents_twth.json|/dataList/0/options/2/messageDesc`

### `… 판정 성공 시`

- **統計**（raw-20261105）：既有 122／146
- **出處**：`ActionEvents.json|/dataList/37/options/0/messageDesc`

### `피해량 +N%`

- **統計**（raw-20261105）：既有 195／227
- **出處**：`EGOgift_MirrorDungeon-EventTheme.json|/dataList/20/simpleDesc/0/simpleDesc`

## 常見錯誤

### 漏掉 KR 的標籤

- **統計**：有 205 行 KR 帶某種標籤（如 `<style>`）而舊譯少了，多在 `EGOgift_MirrorDungeon*`（`docs/guides-evidence/base.md`「種類與數量同 KR」）

### 把 `[ID]` 改寫成「中文＋空格」

- **統計**：舊譯在 `EGOgift*` 有 56 處；全部統計與還原經過見 `docs/guides-evidence/base.md`「`[ID]` 一律保留」

### 修正 draft 時刪掉 `<noparse>`

- **經過**：修正 draft 時刪掉了中文關鍵字名稱上的 `<noparse>`，後由人工還原
- **決定**：人工調整 `rules/adjustments/2026-11-04-noparse.yaml`
- **統計**：該調整檔本族群 3 筆
- **出處**：`EGOgift_MirrorDungeon-EventTheme_2.json|/dataList/25/desc`
