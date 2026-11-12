# N7 依規則、hint 或草稿的處理說明：交給你決定（N443–N443）

[回這一類的目錄](N7.md)。

### N443　交給你決定｜N7 依規則、hint 或草稿的處理說明｜信心中

`note/Passive_Ego.json|/dataList/123/desc/561ddf13`　ally，批次 20261112T035558-e72870f7，來源 claude

- KR：`자신의 1.5를 초과하는 물리 내성이 1.5로 조정됨 (흐트러짐 상태 제외)⏎자신이 흐트러짐 상태면, 받는 피해량 -30% (강제 흐트러짐 포함)⏎⏎자신의 [DespairKnightEGOMind]가 999면, '눈물로 벼려낸 검' E.G.O 스킬(각성, 침식) 사용 시 E.G.O 자원을 소모하지 않음⏎⏎자신을 포함한 아군이 받은 피해만큼 [DespairKnightEGOMind] 얻음 (보호막 피해 포함)⏎아군 사망시 [DespairKnightEGOSword] 1 얻음 (턴당 1회)⏎[DespairKnightEGOProtect]를 보유한 아군 사망 시 [DespairKnightEGOSword] 1 얻음 (턴당 1회)⏎⏎사망 상태인 아군 2명당 자신의 공격 레벨 +1, 방어 레벨 -1 (최대 5)`
- 現行譯文：`自身高於1.5的物理抗性調整為1.5 (混亂狀態除外)⏎若自身處於混亂狀態，則使自身受到的傷害-30% (包括強制混亂)⏎⏎若自身的 [DespairKnightEGOMind] 為999，則使用「淚鋒之劍」E.G.O技能(覺醒、侵蝕)時不消耗罪孽碎片⏎⏎使自身獲得等同於包括自身在內的友方單位所受傷害的 [DespairKnightEGOMind] (包括護盾受到的傷害)⏎友方單位陣亡時，使自身獲得1層 [DespairKnightEGOSword] (每回合最多1次)⏎帶有 [DespairKnightEGOProtect] 的友方單位陣亡時，使自身獲得1層 [DespairKnightEGOSword] (每回合最多1次)⏎⏎每有2名處於陣亡狀態的友方單位，使自身攻擊等級+1、防禦等級-1 (最多5)`
- 註記：JP 把「涙で研ぎ澄まされた剣」包在 <noparse> 裡（名稱同關鍵字 [DespairKnightEGOSword]），KR 沒有，依規則未新增 <noparse>；若遊戲中會被讀成關鍵字需補上（125 的 desc、summary 同）
- 判讀：JP 有 <noparse> 包住「淚鋒之劍」而 KR 無，依規則未新增，詢問是否需補
- 理由：是否補 <noparse> 需人確認遊戲行為；base.md 規定未確認不新增。
- 查證：Passive_Ego.json#123 現行「淚鋒之劍」未包 <noparse>；EGOgift_walpu10#4 的 KR 與譯文有包 <noparse>（但 123／125 的 KR 沒有）。base.md 規定新增 <noparse> 只限已在遊戲中確認會被讀成關鍵字處，目前無確認。
- 建議：交給你決定，選項：
  1. 維持現狀：不新增 <noparse>，確認註記即可（不改任何單位）
  2. 補 <noparse>：在 Passive_Ego.json 的 #123 desc 與 #125 desc（含註記所說的 summary）把「淚鋒之劍」改為「<noparse>淚鋒之劍</noparse>」，須先在遊戲中確認會被讀成關鍵字
- 同意後寫入：依你選的選項，並`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出
