# 關鍵字與狀態（keywords）

## 文本類型

- 狀態 `Bufs*`（52 檔）與戰鬥關鍵字 `BattleKeywords*`（51 檔）：名稱 `name`、說明 `desc`、摘要 `summary`、`flavor`（routing.yaml；raw-20261105 檔數）
- 恐慌 `PanicInfo*`（28 檔）：`panicName`、`panicDescription`、`lowMoraleDescription`；另有 `UnitKeyword*`（14 檔）、`MentalCondition*` 等（routing.yaml；raw-20261105 檔數）

## 語氣與人稱

（目前無足夠證據）

## 格式慣例

- **恐慌**：說明裡的標題行 `- 패닉` 寫「- 陷入恐慌」（既有 156：6）（自 base.md「補充規則」移入；KR 以 `- 패닉` 開頭的 165 個單位全在本族群）
- 說明各條開頭的「- 」照 KR 保留，條數相同（既有 2,255／2,303，例 `Bufs.json|/dataList/9/desc`）
- `- 최댓값: N` 寫「- 最大值：N」（既有 676／718，例 `Bufs-cultivation.json|/dataList/3/desc`）
- `수치 N당`（本效果的層數）寫「每帶有N層本效果」或「每有N層」（既有 238／266，例 `BattleKeywords-a1c9p1.json|/dataList/21/desc`）
- `(턴당 N회)` 寫「(每回合最多N次)」（既有 126／138，例 `Bufs-a1c9p1.json|/dataList/16/desc`）
- `공격 레벨 +N`、`최종 위력 +N` 寫「攻擊等級+N」「最終威力+N」（既有 304／324、182／198，例 `Bufs.json|/dataList/593/desc`、`Bufs.json|/dataList/108/summary`）
- `정신력 N 회복` 寫「恢復N點理智值」（既有 150／152，例 `Bufs.json|/dataList/306/desc`）
- `적중 시` 寫「命中時」；`처치 시` 寫「擊殺……時」（既有 193／197、31／31，例 `Bufs.json|/dataList/47/desc`、`SkillTag.json|/dataList/45/name`）
- `아군` 寫「友方單位」（既有 297／326，例 `Bufs.json|/dataList/12/desc`）
- KR 方括號裡的韓文名稱譯成中文，方括號照樣保留（`[…]`）（既有 308／308，例 `SkillTag.json|/dataList/24/name`）

## 常見錯誤

（目前無足夠證據）
