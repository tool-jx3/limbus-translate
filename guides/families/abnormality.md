# 異想體圖鑑與事件（abnormality）

## 文本類型

- 異想體圖鑑 `AbnormalityGuides*`（26 檔）：名稱 `name`、線索 `clue`、觀察日誌 `story`（families.md 資料補充；raw-20261105 檔數）
- 異想體事件 `AbEvents*`（37 檔）：標題 `title`、事件敘述 `eventDesc`／`prevDesc`、選擇罪人的提問 `behaveDesc`、結果 `successDesc`／`failureDesc`；結果紀錄 `AbEventsResultLog*`（16 檔，`content`）（families.md 資料補充；raw-20261105 檔數與欄位）
- `AbDlg_<罪人>`（12 檔）：罪人對異想體的台詞（`teller`、`dialog`），套用該罪人的角色指引（families.md 資料補充；README.md）

## 語氣與人稱

（目前無足夠證據）

## 格式慣例

- **`%…/%` 樣板**：`%[{1}] {2}/%` 寫成 `%{2} [{1}]/%`（既有 36 處中 35 處）；KR 樣板外的字（例如動詞 `얻음`、`다음 전투 N회 동안`）留在樣板外，不移進 `%…/%`（自 base.md「補充規則」移入；KR 含 `%…/%` 的 92 個單位全在本族群）
- `[{n}] 수감자` 寫「罪人[{n}]」，「罪人」在前（既有 172／177，例 `AbEvents-a1c5p3.json|/dataList/3/successDesc/0`）
- `behaveDesc` 的 `어떤／어느 수감자…?` 寫「（讓）哪位罪人……？」（既有 130／139，例 `AbEvents.json|/dataList/40/behaveDesc`）
- 觀察日誌、結果紀錄裡各條開頭的「- 」照 KR 保留，條數相同（既有 32／32，例 `AbnormalityGuides.json|/dataList/90/storyList/0/story`）

## 常見錯誤

（目前無足夠證據）
