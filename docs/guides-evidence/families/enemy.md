# `guides/families/enemy.md` 的依據

標 raw-20261105 的「既有 a／b」與「全在本族群」檢查由 `docs/research/scripts/family_guides.py` 算出（「既有 a／b」的門檻見 `guides/README.md`「新增慣例的門檻」）；標 raw-20261105 的文本類型檔案數來自 `guides/routing.yaml` 與 `guides/families.md`；沒有標資料版本的，是官方 2026-10-29 語言包與當時的既有繁中（依寫入日期 2026-10-31～11-02 推定；`docs/guides-evidence/README.md`「資料依據」）。

## 文本類型

- **統計**（raw-20261105）：`Enemies*` 46 檔；`Skills_Enemy*` 34 檔、`Skills_Abnormality*` 50 檔、`Skills_Tutorial`／`Skills_fools` 2 檔；`Passives_Enemy*` 26 檔、`Passives_Abnormality*` 46 檔、`Passive-*` 2 檔、`Passives_fools` 1 檔
- **出處**：`guides/routing.yaml`

## 格式慣例

### `[X] N 부여`

- **統計**（raw-20261105）：既有 1,729／1,810
- **出處**：`Skills_Enemy-tkt.json|/dataList/20/levelList/0/coinlist/0/coindescs/1/desc`

### `피해량 +N%`

- **統計**（raw-20261105）：既有 615／646
- **出處**：`Passives_Enemy-a1c6p1.json|/dataList/0/summary`

## 常見錯誤

### 把 `[ID]` 改寫成「中文＋空格」

- **統計**：舊譯共 1,088 處，幾乎都在 `Skills*`（842）、`Passives*`（157）；全部統計與還原經過見 `docs/guides-evidence/base.md`「`[ID]` 一律保留」

### 修正 draft 時刪掉 `<noparse>`

- **經過**：修正 draft 時刪掉了中文關鍵字名稱上的 `<noparse>`，後由人工還原
- **決定**：人工調整 `rules/adjustments/2026-11-04-noparse.yaml`
- **統計**：該調整檔本族群 7 筆
- **出處**：`Passives_Abnormality-mr8.json|/dataList/4/desc`
