# `guides/shared/battle-text.md` 的依據

統計由 `docs/research/scripts/family_guides.py` 算出（raw-20261105）；「既有 a／b」是 ally、enemy、keywords、mirror 四個族群合計（`guides/README.md`「共用指引」）。

## 帶入範圍

- **決定**：ally、enemy、keywords、mirror 在 `guides/routing.yaml` 以 `include:` 引入（2026-11-07 從各族群指引搬進共用指引）

## 格式慣例

### 效果文字的括號

- **統計**（raw-20261105）：既有 6,777／6,883 除了「中文意思（原文）」都只用半形：ally 3,129／3,138、enemy 1,621／1,627、keywords 1,378／1,436、mirror 649／682
- **統計**（raw-20261105）：KR 在韓文音譯後的括號附原文時寫「中文意思（原文）」、用全形，既有 122／126：ally 58／58、enemy 24／24、keywords 40／44
- **統計**（raw-20261105）：mirror 的介面、教學、目錄與事件敘述用全形，既有 23／33
- **決定**：使用者決定 2026-11-07（`guides/base.md`「括號依文本類型」；`docs/guides-evidence/base.md` 同一節）
- **出處**：`Skills_Ego.json|/dataList/76/levelList/0/name`「赤瞳(開)」、`Skills_Abnormality-a1c10p3.json|/dataList/60/levelList/0/name`「脫離（Dégagé）」

### `(턴당 N회)`

- **統計**（raw-20261105）：既有 618／633
- **出處**：`Passives-BossRaid.json|/dataList/14/desc`

### `최종 위력 ±N`

- **統計**（raw-20261105）：既有 755／792
- **出處**：`Skills_personality-06.json|/dataList/4/levelList/2/desc`

### 恢復

- **統計**（raw-20261105）：`정신력 N 회복` 既有 630／642、`체력 N 회복` 既有 141／141
- **出處**：`Passives.json|/dataList/15/desc`、`Passives_Enemy.json|/dataList/0/desc`

### `아군`

- **統計**（raw-20261105）：既有 1,772／1,857
- **出處**：`Passives.json|/dataList/0/desc`

### `처치 시`

- **統計**（raw-20261105）：既有 115／120
- **出處**：`Passives.json|/dataList/106/desc`、`SkillTag.json|/dataList/45/name`
