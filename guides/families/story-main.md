# 主線劇情（story-main）

## 文本類型

- 主線劇情 `StoryData/S*`（409 檔）與主線地牢 `StoryData/<章>D<層>XX`（147 檔）：台詞 `content`、說話者 `teller`、稱號 `title`、地點 `place`（families.md「多數檔案命名規律」；raw-20261105 檔數）
- `DungeonNode*`（17 檔）：主線與間章地牢的節點名稱 `title` 與說明 `desc`（families.md 原指引；raw-20261105 檔數）
- `BgmLyrics/*`（17 檔）：地牢漂浮歌詞及片頭片尾曲歌詞，欄位 `content`（families.md 原指引；raw-20261105 檔數）

## 語氣與人稱

（目前無足夠證據）

## 格式慣例

- **語言樣式標籤**：歌詞等處的 `<style="dkr…">` 是韓文版的語言樣式（EN 為 `den…`、JP 為 `djp…`）。譯文一律寫 `djp…`、保留後綴（如 `djp_CP6_2`、`DJP_CP9_1`），與三個中文社群包相同；驗證把三者視為相同的標籤，正規化會自動改成 `djp`。（自 base.md「補充規則」移入；KR 含 `<style="dkr` 的 157 個單位全在 `BgmLyrics/`）

## 常見錯誤

（目前無足夠證據）
