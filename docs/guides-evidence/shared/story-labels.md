# `guides/shared/story-labels.md` 的依據

這些規則原本在 `guides/base.md`「與程式語言有關的翻譯須知」的資料補充與補充規則，指引整理（設計 2026-11-17，6.2、7.2 節）時移到這份共用指引。「帶入範圍」以外的統計，資料是官方 2026-10-29 語言包與當時的既有繁中（`docs/guides-evidence/README.md`「資料依據」）。

## 帶入範圍

- **統計**（raw-20261112 的 KR）：`place` 在 `StoryData/`（story-main 988、story-interlude 514 個單位）與 `StageNode`（223 個單位，14 檔）；`teller` 在 `StoryData/`（story-main 4,239、identity 3,811、story-interlude 1,493）與 `AbDlg`（169 個單位，12 檔，罪人名或「組織＋罪人名」，例如 `리우 협회 그레고르` → 六協會 格里高爾）；`nickName` 只在 `ScenarioModelCodes`（758）；`RPGSystem/` 的 `speaker` 6,000 個單位（29 檔）、`displayName`（NPC 與道具名稱）889 個（70 檔）、`title` 232 個（20 檔），其中含 `르루주` 的 145 個（`displayName` 96、`speaker` 40、`title` 9）
- **統計**（raw-20261112）：帶到的檔：`StoryData/` 1,001 檔、`RPGSystem/` 189 檔、`StageNode` 43 檔、`AbDlg` 12 檔、`ScenarioModelCodes` 1 檔，共 1,246 檔；帶 `place`、`teller`、`nickName` 的檔都帶到
- **決定**：以 `guides:` 依節點帶給 `StoryData/`、`ScenarioModelCodes*`、`StageNode*`（設計 7.2 節）。Task 3 審查發現 `RPGSystem/`（`speaker`、`displayName`，例如「丹紅派 手提包展廳 設計師」）與 `AbDlg`（`teller`）原本從 base 讀到這些規則，拆出後會讀不到；控制者裁定（2026-11-17）也帶給 `RPGSystem/` 與 `AbDlg`，開頭的欄位說明一併列出

## 格式慣例

### 地點欄位

- **統計**：KR 地點本來就寫 `7F` 時照寫「7F」，既有 48 處
- **統計**（raw-20261112）：既有地點欄位「內部」前有空格 126 處、沒有 18 處；「外部」3 處都有空格；「甲板」有空格 18 處、沒有 13 處
- **決定**：使用者決定 2026-11-17：「內部」統一成前面空一格，沒有空格的 18 處以 `rules/rewrite/2026-11-17-place-naibu-space*.yaml` 改寫；其中 7 處的 KR 沒有 `심사장 내부`（`심사장 1층 실험실 폐허`、`1층 실험실 폐허`、`지하 1층 실험실 폐허`、`14구, 폐건물 안, …`），一併照 KR 改

### 稱號與說話者名稱的空格

- **統計**：既有中指、環指、拇指 583 處都空格
- **統計**（raw-20261112）：幫派老大的說話者名稱：死兔幫 老大 41、鐵工會 老大 21 處有空格；豆豆幫老大 9 處（teller 8、ScenarioModelCodes 1）、ScenarioModelCodes 的鐵工會老大、死兔幫老大各 1 處沒有
- **決定**：使用者決定 2026-11-17 照現行規則，沒有空格的 11 處以 `rules/rewrite/2026-11-17-gang-boss-space*.yaml` 改成有空格（另考慮過幫派老大一律不空格、只改鐵工會，沒有採用）
