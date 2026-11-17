# `guides/shared/story-labels.md` 的依據

這些規則原本在 `guides/base.md`「與程式語言有關的翻譯須知」的資料補充與補充規則，指引整理（設計 2026-11-17，6.2、7.2 節）時移到這份共用指引。統計的資料是官方 2026-10-29 語言包與當時的既有繁中（`guides/README.md`「資料依據」）。

## 帶入範圍

- **統計**（`.work/raw` 2026-11-17 的 KR）：`place` 在 `StoryData/`（story-main 988、story-interlude 514 個單位）與 `StageNode`（223 個單位，14 檔）；`teller` 在 `StoryData/`（story-main 4,239、identity 3,811、story-interlude 1,493）與 `AbDlg`（169 個單位，12 檔）；`nickName` 只在 `ScenarioModelCodes`（758）。帶到的檔：`StoryData/` 1,001 檔、`StageNode` 43 檔、`ScenarioModelCodes` 1 檔，共 1,045 檔
- **決定**：以 `guides:` 依節點帶給 `StoryData/`、`ScenarioModelCodes*`、`StageNode*`（設計 7.2 節）。`AbDlg` 的 `teller`（罪人或人格名，例如「六協會 格里高爾」）沒有帶到

## 格式慣例

### 地點欄位

- **統計**：KR 地點本來就寫 `7F` 時照寫「7F」，既有 48 處

### 稱號與說話者名稱的空格

- **統計**：既有中指、環指、拇指 583 處都空格

### 百貨公司的店名

- **統計**：句子裡不加空格的「丹紅派精品專櫃」既有 11 處
