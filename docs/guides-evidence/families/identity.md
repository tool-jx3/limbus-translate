# `guides/families/identity.md` 的依據

標 raw-20261105 的「既有 a／b」與「全在本族群」檢查由 `docs/research/scripts/family_guides.py` 算出（「既有 a／b」的門檻見 `guides/README.md`「新增慣例的門檻」）；標 raw-20261105 的文本類型檔案數來自 `guides/routing.yaml` 與 `guides/families.md`；沒有標資料版本的，是官方 2026-10-29 語言包與當時的既有繁中（依寫入日期 2026-10-31～11-02 推定；`docs/guides-evidence/README.md`「資料依據」）。

## 文本類型

- **統計**（raw-20261105）：人格故事 `StoryData/P1YYXX` 181 檔、人格語音 `PersonalityVoiceDlg/` 194 檔、戰鬥泡泡 `BattleSpeechBubbleDlg*` 11 檔
- **出處**：人格故事出自 `guides/families.md`「原指引」，人格語音出自 `guides/families.md`；戰鬥泡泡（`desc` 不翻）出自 `guides/families.md`「戰鬥泡泡」，罪人代碼出自 `guides/families.md`「檔名代碼與遊戲內編號」，族群指引建立時（2026-11-07）移入本檔
- **出處**：罪人代碼與遊戲內編號見 `guides/characters/*.md` 的 `code`、`number`；遊戲內編號出自 `ScenarioModelCodes` 的 `nickName`，例 `ScenarioModelCodes-AutoCreated.json|/dataList/9/nickName`：辛克萊「11번 수감자」

## 語氣與人稱

### 戰鬥泡泡的說話者

- **統計**（raw-20261112）：1,555 個 id 裡 773 個含一個人格編號（`1XXYY`，罪人代碼 01～12，沒有一個 id 含兩位罪人的編號）；其餘 782 個是 `<章>SV-…` 等敵人、NPC、協助者的台詞。十個分章、分活動的檔幾乎全是後者。218 個 id 是另一個 id 加上 `-0N` 的變體，其中 140 個的 KR 與原 id 相同
- **決定**：使用者決定 2026-11-17：批次帶上條目的 id、`desc` 與依 id 判斷的說話者。在此之前批次沒有這三樣，指引要譯者看的 `desc` 與 id 並不在批次裡
- **經過**：`9SV-BAT17-26-3` 的舊譯是另一句台詞的譯文，同一句 KR 在劇情 `StoryData/S950I2.json` 有正確的譯文
- **出處**：`guides/families.md`「戰鬥泡泡」，族群指引建立時（2026-11-07）移入本檔；id 是語音事件名稱、`desc` 不顯示，見 `docs/research/2026-11-15-game-text-logic/05-battle-speech-bubble.md`；敵人台詞的比例見同目錄 `06-families.md` 第 3 節

## 格式慣例

### 人格語音的分類標籤

- **統計**：`아침 인사` → 晨間問候既有 138 處，55 處寫「早晨問候」；`점심 인사` → 午間問候既有 137 處，56 處寫「午時問候」；`대화 1`、`동기화 후 대화 1` 數字前空一格既有 188、179 處，7 處不空格；`입장` → 進入戰鬥既有 190 處，「入場」「進入」各 1 處；`EX CLEAR 전투 승리` → EX CLEAR戰鬥勝利既有 192 處不空格，1 處空格
- **出處**：`guides/families.md`「人格語音的分類標籤」，族群指引建立時（2026-11-07）移入本檔

### `흐트러질 시 대사`

- **統計**（raw-20261105）：既有 190／190
- **出處**：`PersonalityVoiceDlg/Voice_Outis_LCB_11101.json|/dataList/13/desc`

### `아군 사망`

- **統計**（raw-20261105）：既有 39／39
- **出處**：`PersonalityVoiceDlg/Voice_Faust_LCB_10201.json|/dataList/15/desc`
