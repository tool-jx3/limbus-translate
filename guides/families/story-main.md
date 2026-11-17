# 主線劇情（story-main）

## 文本類型

- 主線劇情 `StoryData/S*` 與主線地牢 `StoryData/<章>D<層>XX`：台詞 `content`、說話者 `teller`、稱號 `title`、地點 `place`
- `DungeonNode*`：主線與間章地牢的節點名稱 `title` 與說明 `desc`
- `BgmLyrics/*`：地牢漂浮歌詞及片頭片尾曲歌詞，欄位 `content`

## 格式慣例

- **語言樣式標籤**：歌詞等處的 `<style="dkr…">` 是韓文版的語言樣式（EN 為 `den…`、JP 為 `djp…`）。譯文一律寫 `djp…`、保留後綴（如 `djp_CP6_2`、`DJP_CP9_1`）
