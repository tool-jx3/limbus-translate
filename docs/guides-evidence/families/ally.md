# `guides/families/ally.md` 的依據

標 raw-20261105 的統計由 `docs/research/scripts/family_guides.py` 算出（「既有 a／b」見 `guides/README.md`「新增慣例的門檻」）；沒有標資料版本的，是官方 2026-10-29 語言包與當時的既有繁中（`guides/README.md`「資料依據」）。

## 文本類型

- **統計**（raw-20261105）：`Skills.json` 1 檔（另含防禦、迴避、反擊 3 筆通用技能）、`Skills_personality-YY` 15 檔、`Skills_Ego_Personality-YY` 14 檔、`Skills_Ego*` 2 檔；`Passives*`、`Passive_Ego*` 13 檔；`Skills_Assist*`／`Passives_Assist*`／`Assist*` 26 檔；`Personalit*`、`Egos*`、`EGO*` 7 檔
- **出處**：`guides/routing.yaml`

## 格式慣例

### `[X] N 부여`

- **統計**（raw-20261105）：既有 1,691／1,776
- **出處**：`Skills_personality-12.json|/dataList/8/levelList/0/desc`

### `피해량 +N%`

- **統計**（raw-20261105）：既有 756／807
- **出處**：`Passives.json|/dataList/3/desc`

## 常見錯誤

### 把 `[ID]` 改寫成「中文＋空格」

- **統計**：舊譯共 1,088 處，幾乎都在 `Skills*`（842）、`Passives*`（157）；全部統計與還原經過見 `docs/guides-evidence/base.md`「`[ID]` 一律保留」

### 修正 draft 時刪掉 `<noparse>`

- **經過**：修正 draft 時刪掉了中文關鍵字名稱上的 `<noparse>`，後由人工還原
- **決定**：人工調整 `rules/adjustments/2026-11-04-noparse.yaml`
- **統計**：該調整檔本族群 8 筆
- **出處**：`Passives.json|/dataList/259/desc`
