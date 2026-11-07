# RPG 探索（rpg）

## 文本類型

- `RPGSystem/rpg-loc-*`（189 檔）：對話 `rpg-loc-dialogue-*`、選項 `rpg-loc-dialogue-choice-*`、NPC 與怪物名稱 `rpg-loc-npc-*`／`swarm-mob-*`、道具 `rpg-loc-item-*`、任務 `rpg-loc-quest-*`、地點 `rpg-loc-location-*`、旁白 `rpg-loc-narration-*`，以及 `rpg-loc-player-*`、`rpg-loc-ui-*`（families.md 資料補充；raw-20261105 檔名）
- **RPG 探索的對話**：`RPGSystem/rpg-loc-dialogue-*` 是對話，每句的說話者在 `speaker`（KR 名稱，與角色檔的 `models` 相同）；`rpg-loc-dialogue-choice-*` 是選項（自 families.md 移入）

## 語氣與人稱

- **RPG 角色口癖**：副廚（수셰프）幾乎每句以「~(는/다는) 거예요」結尾，譯成句尾「喔」（item-common-a1c10p1／p2 既有），只加在 KR 有 거예요 的陳述句與命令句（包括 `죽을 거예요` 這種未來式、`~라는 거예요` 的命令）；問句不加，包括反問與轉述的問句（`~냐는 거예요`）；不用「的說」（自 families.md 移入）

## 格式慣例

- **RPG 標籤的括號**：`RPGSystem/` 除了對話之外的所有文字（`rpg-loc-item-*` 的 `statText`、`rpg-loc-npc-*`／`swarm-mob-*` 的名稱、`rpg-loc-quest-*` 的 title／goal／description、`rpg-loc-location-*`）括號一律用全形「（）」（既有 statText 29：1、paratranz 任務全是全形，例如「黃金皮革閃光（屍堆）」「（內部）…」）；開發用的代號照原樣放在全形括號裡，括號內的逗號照 KR：「（52,64）」「（S1001B）」。這是 base.md 第 7 條「系統文字用半形」的例外。只有問號的名稱寫半形「???」（既有 226 處）（自 families.md 移入）
- **RPG 的樓層寫法**：任務的 title、goal 與道具名、NPC 名稱是標籤，寫「3F」「B2F」；任務的 description 與對話是敘述，寫「3樓」「地下2樓」（自 families.md 移入）

## 常見錯誤

- **RPG 樓層變體要一致**：`floor-N`、`floor-N-b`、`floor-N-c1`、`floor-N-c2`（以及 `common-*`）有大量 KR 完全相同的句子。KR 相同的句子譯文也要完全相同：已經譯好的那句優先；兩邊都已譯好時以本體 `floor-N` 為準（除非它錯了）。翻譯前先在 `.work/tw/.meta/units/RPGSystem/` 找同一句 KR（自 families.md 移入）
