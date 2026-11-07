# 人格故事、人格語音與戰鬥泡泡（identity）

## 文本類型

- 人格故事 `StoryData/P1YYXX`（181 檔）：欄位同劇情（`content`、`teller`、`title`）（families.md 原指引；raw-20261105 檔數）
- 人格語音 `PersonalityVoiceDlg/Voice_<罪人>_<人格>_<編號>`（194 檔）：`desc` 是分類標籤、`dlg` 是台詞（families.md；raw-20261105 檔數）
- 戰鬥泡泡 `BattleSpeechBubbleDlg*`（11 檔）：台詞在 `dlg`；`desc` 是觸發標籤，三語幾乎相同，不翻（自 families.md「戰鬥泡泡」移入）
- 罪人代碼：檔名與人格編號 `1YYXX` 的 YY，例如 `10310` = 唐吉訶德（03）第 10 號人格；代碼 10、11、12 是辛克萊、奧提斯、格里高爾，遊戲內是 11、12、13 號（自 families.md「檔名代碼與遊戲內編號」移入）；各罪人的代碼與遊戲內編號見 `guides/characters/<id>.md` 的 `code`、`number`，01～09 兩者相同（characters/*.md；遊戲內編號出自 `ScenarioModelCodes` 的 `nickName`，例 `ScenarioModelCodes-AutoCreated.json|/dataList/9/nickName`：辛克萊「11번 수감자」）

## 語氣與人稱

- 戰鬥泡泡的批次沒有說話者：看 KR 的 `desc`（觸發標籤常含角色名）與 `id`（`1XXYY` = 罪人代碼 XX、人格 YY）判斷是誰在說話（自 families.md「戰鬥泡泡」移入）

## 格式慣例

- **人格語音的分類標籤**：`PersonalityVoiceDlg` 的 `desc` 是語音的分類標籤，所有人格寫法相同，照既有多數：`아침 인사` → 晨間問候（既有 138 處；55 處寫「早晨問候」）、`점심 인사` → 午間問候（既有 137 處；56 處寫「午時問候」）、`저녁 인사` → 晚間問候、`대화 1` → 對話 1、`동기화 후 대화 1` → 同步後對話 1（數字前空一格，同 KR；既有 188、179 處，7 處不空格）、`입장` → 進入戰鬥（既有 190 處；「入場」「進入」各 1 處）、`EX CLEAR 전투 승리` → EX CLEAR戰鬥勝利（既有 192 處不空格，1 處空格）；標籤結尾不留空格（自 families.md「人格語音的分類標籤」移入）
- 標籤 `흐트러질 시 대사` 寫「自身混亂」（既有 190／190，例 `PersonalityVoiceDlg/Voice_Outis_LCB_11101.json|/dataList/13/desc`）
- 標籤 `아군 사망` 寫「友方陣亡」（既有 39／39，例 `PersonalityVoiceDlg/Voice_Faust_LCB_10201.json|/dataList/15/desc`）

## 常見錯誤

（目前無足夠證據）
