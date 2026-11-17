# `guides/families/keywords.md` 的依據

標 raw-20261105 的「既有 a／b」與「全在本族群」檢查由 `docs/research/scripts/family_guides.py` 算出（「既有 a／b」的門檻見 `guides/README.md`「新增慣例的門檻」）；標 raw-20261105 的文本類型檔案數來自 `guides/routing.yaml` 與 `guides/families.md`；沒有標資料版本的，是官方 2026-10-29 語言包與當時的既有繁中（依寫入日期 2026-10-31～11-02 推定；`docs/guides-evidence/README.md`「資料依據」）。

## 文本類型

- **統計**（raw-20261105）：`Bufs*` 52 檔、`BattleKeywords*` 51 檔、`PanicInfo*` 28 檔、`UnitKeyword*` 14 檔
- **出處**：`guides/routing.yaml`

## 格式慣例

### 恐慌

- **統計**：`- 패닉` 寫「- 陷入恐慌」既有 156：6
- **統計**（raw-20261105）：KR 以 `- 패닉` 開頭的 165 個單位全在本族群
- **經過**：原在 `guides/base.md`「補充規則」，族群指引建立時（2026-11-07）移入本檔

### 條列的「- 」

- **統計**（raw-20261105）：既有 2,255／2,303
- **出處**：`Bufs.json|/dataList/9/desc`

### `- 최댓값: N`

- **統計**（raw-20261105）：既有 676／718
- **出處**：`Bufs-cultivation.json|/dataList/3/desc`

### `수치 N당`

- **統計**（raw-20261105）：既有 238／266
- **出處**：`BattleKeywords-a1c9p1.json|/dataList/21/desc`

### `공격 레벨 ±N`

- **統計**（raw-20261105）：既有 304／324
- **出處**：`Bufs.json|/dataList/593/desc`

### `적중 시`

- **統計**（raw-20261105）：既有 193／197
- **出處**：`Bufs.json|/dataList/47/desc`

## 常見錯誤

### 狀態名稱的符號

- **出處**：研究報告 `docs/research/2026-11-15-game-text-logic/03-keywords.md`；`Bufs-a1c10p3.json|/dataList/20/name`
- **決定**：人工調整 `rules/adjustments/2026-11-16-keyword-link.yaml` 把 `ConfessionThirdPhase` 的名稱與說明照 KR、EN 寫「〜」，同時補上這條常見錯誤
