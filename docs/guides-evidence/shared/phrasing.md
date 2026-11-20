# `guides/shared/phrasing.md` 的依據

這份共用指引整理自 StoryData 審查（`docs/research/2026-11-13-storydata-accuracy/`）逐項決定譯法時的討論。每條規則都來自使用者對實際句子的取捨，沒有統計。

## 帶入範圍

- **決定**：使用者決定 2026-11-20：寫成共用指引、不放進 `guides/base.md`（技能、關鍵字、介面文字用不到）；以 `include:` 引入 story-main、story-interlude、identity、ego-voice、announcer、dante-note、abnormality、rpg。mirror、stage 也有成句的敘述（E.G.O 飾品說明、選擇事件），但與固定格式的文字混在一起，先不引入
- **經過**：只影響之後新翻或原文變動的單位；既有譯文不因這份指引重翻

## 原則

### 語氣整句對應

- **決定**：使用者決定 2026-11-20（S0455）。逐個助詞對譯的寫法（「就算是現在也好，就由我出面去救他們看看吧」等）使用者認為不通順，改用「不然……好了」
- **出處**：`StoryData/E504B.json|/dataList/46/content`

### 不順時換句型，不加字

- **決定**：使用者決定 2026-11-20（S0326）。先後試過「隱祕……卻又坦誠」（形容詞單獨當謂語，不順）與「低得像在說祕密……卻又十分真誠」（加了音量、比喻、程度詞，不順也離 KR 較遠），最後用「既……又……」
- **出處**：`StoryData/5D107B.json|/dataList/21/content`

### 不替原文補內容

- **決定**：使用者決定 2026-11-20（S0325）。`술통 내기` 的酒桶是賭注，KR 沒寫比什麼；「比賽喝桶裝酒」是 EN 的讀法。定稿整句是「出去以後要不要再來賭一桶酒？就像以前那樣！」，其中「再」是 KR 沒有、使用者決定保留的字，所以指引只取片段當例子
- **出處**：`StoryData/5D106B.json|/dataList/16/content`

### 字面是重點時守字面

- **決定**：使用者決定 2026-11-20（S0330）：`이것의 심장` 寫「這傢伙的心臟」，與同一場 `이것을 죽여야 한다고` 的「這傢伙」一致
- **出處**：`StoryData/5D203B.json|/dataList/78/content`、`StoryData/5D203B.json|/dataList/82/content`

## 句子結構

### 長修飾語

- **決定**：使用者決定 2026-11-20（S0108）。「我都會記得那些從我身上被扯斷的繩索吧。」使用者認為不通順，改成受詞提前
- **出處**：`StoryData/S539B.json|/dataList/24/content`

### 連接語尾

- **決定**：使用者決定 2026-11-20（S0325）：`밖에 나가면` 由「去到外面的話」改為「出去以後」
- **出處**：`StoryData/5D106B.json|/dataList/16/content`

### 形容詞謂語

- **決定**：使用者決定 2026-11-20（S0326），同「不順時換句型，不加字」

## 常見語尾

- **出處**：`-(이)라도`、`-ㄹ까.`：`StoryData/E504B.json|/dataList/46/content`；`-겠지.`：`StoryData/S539B.json|/dataList/24/content`；`-고도`：`StoryData/5D107B.json|/dataList/21/content`；`-ㄹ걸.`：`StoryData/E504B.json|/dataList/47/content` 的既有譯文（`더 적극적으로 배울걸.` → 我就該學得積極點才對。）
