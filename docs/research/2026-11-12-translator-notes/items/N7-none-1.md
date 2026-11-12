# N7 依規則、hint 或草稿的處理說明：確認即可（N444–N658）

[回這一類的目錄](N7.md)。

### N444　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/AbEvents.json|/dataList/52/failureDesc/0/70618c14`　abnormality，批次 20261101T082131-d93117a2，來源 claude

- KR：`수감자가 집중하고 있는 동랑의 빈틈 속으로 깊숙이 무기를 찔러 넣었어.⏎⏎"무능하긴."⏎⏎하지만, 그 빈틈은 동랑이 파놓은 함정에 불과했지.⏎[{0}] 수감자는 공격을 맞고 튕겨나갔어.⏎⏎"하, 뜻은 알겠군."⏎⏎"어쩔 수 없지. 당신들도 이상 째로 사라지게 만들겠어."`
- 現行譯文：`罪人趁東朗專注之際，將武器深深刺入他的破綻。⏎⏎「真是無能。」⏎⏎但那個破綻，不過是東朗設下的陷阱。⏎罪人[{0}]被攻擊彈飛了出去。⏎⏎「哈，你們的意思我明白了。」⏎⏎「沒辦法了。我會讓你們連同李箱一起消失。」`
- 註記：KR 第一句沒有 [{0}]，draft 多一個佔位符，已刪；「집중하고 있는」修飾東朗，依 KR 改譯
- 判讀：KR 無第一個 [{0}]，刪去多餘佔位符並依 KR 修飾關係改譯
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N445　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/AbEventsResultLog-a1c9p1.json|/dataList/2/content/9323e687`　abnormality，批次 20261101T082132-3d370b72，來源 claude

- KR：`<color=#ebcaa2>[{0}] 수감자</color> <color=#e30000>{1} {2}</color><color=#ebcaa2>얻음</color>⏎`
- 現行譯文：`<color=#ebcaa2>罪人[{0}]</color><color=#ebcaa2>獲得</color><color=#e30000>{2} {1}</color>⏎`
- 註記：補回 KR 的第三組 <color> 標籤；{2} {1} 順序沿用 draft／EN
- 判讀：補回 KR 第三組 <color> 標籤；{2} {1} 順序沿用 draft／EN
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N446　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/AbEventsResultLog-a1c9p1.json|/dataList/3/content/9323e687`　abnormality，批次 20261101T082132-3d370b72，來源 claude

- KR：`<color=#ebcaa2>[{0}] 수감자</color> <color=#e30000>{1} {2}</color><color=#ebcaa2>얻음</color>`
- 現行譯文：`<color=#ebcaa2>罪人[{0}]</color><color=#ebcaa2>獲得</color><color=#e30000>{2} {1}</color>`
- 註記：補回 KR 的第三組 <color> 標籤；{2} {1} 順序沿用 draft／EN
- 判讀：同 #5：補回 <color> 標籤，{2} {1} 順序沿用
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N447　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/AbEventsResultLog-pilgrimage.json|/dataList/0/content/e2df6e8d`　abnormality，批次 20261101T112445-a36a3c8a，來源 claude

- KR：`<color=#ebcaa2>{0}가</color> %[{1} {2}]/% 얻음`
- 現行譯文：`<color=#ebcaa2>{0}</color>獲得%[{1} {2}]/%`
- 註記：KR 樣板是 %[{1} {2}]/%（方括號包住兩個佔位符，只有本檔如此）；依提示方括號與佔位符順序照 KR，不比照 EN 改成 [{2} {1}]
- 判讀：KR 樣板方括號包住兩個佔位符，依提示照 KR，不比照 EN
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N448　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/AbEventsResultLog.json|/dataList/65/content/ecb04112`　abnormality，批次 20261101T112451-09611415，來源 claude

- KR：`<color=#ebcaa2>모든 아군이 다음 전투</color> %{0}회 동안 {1} {2}/%<color=#ebcaa2>을 얻음</color>`
- 現行譯文：`<color=#ebcaa2>全體友方單位獲得</color>%{2} {1}，持續{0}場/%<color=#ebcaa2>戰鬥</color>`
- 註記：KR 樣板內是「{0}회 동안 {1} {2}」，樣板外是「다음 전투」「얻음」：「獲得」「戰鬥」放在樣板外，「持續{0}場」留在樣板內；「다음」由「持續」表達
- 判讀：依 KR 樣板內外分界，「獲得」「戰鬥」放樣板外，「持續{0}場」放樣板內
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N449　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/AbEventsResultLog.json|/dataList/67/content/18ab5362`　abnormality，批次 20261101T112451-09611415，來源 claude

- KR：`<color=#ebcaa2>[{0}]이 다음 전투</color> %{1}회 동안 {2} {3}/%<color=#ebcaa2>을 얻음</color>`
- 現行譯文：`<color=#ebcaa2>[{0}]獲得</color>%{3} {2}，持續{1}場/%<color=#ebcaa2>戰鬥</color>`
- 註記：同 #65：「獲得」「戰鬥」在樣板外，「持續{1}場」在樣板內
- 判讀：同 #65 的樣板內外分界
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N450　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/AbEventsResultLog.json|/dataList/90/content/f2425bb4`　abnormality，批次 20261101T112451-09611415，來源 claude

- KR：`<color=#ebcaa2>[{0}]</color> %[{1}] {2}, [{3}] {4}/%<color=#ebcaa2> 얻음</color>`
- 現行譯文：`<color=#ebcaa2>[{0}]</color><color=#ebcaa2>獲得了</color>%{2} [{1}]，{4} [{3}]/%`
- 註記：樣板內依 base.md「%[{1}] {2}/% 寫成 %{2} [{1}]/%」保留數值與 [{1}] 之間的空格（視為 [{n}] 不空格規則的特例），與 #89、#91 一致
- 判讀：樣板內保留數值與 [{1}] 間空格（base.md 特例）
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N451　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/AbEventsResultLog.json|/dataList/91/content/b8bb2ee9`　abnormality，批次 20261101T082138-09611415，來源 claude

- KR：`<color=#ebcaa2>[{0}]</color> %[{1}] {2}/%<color=#ebcaa2> 얻음</color>`
- 現行譯文：`<color=#ebcaa2>[{0}]</color><color=#ebcaa2>獲得了</color>%{2} [{1}]/%`
- 註記：KR 是「얻음」（獲得），草稿寫成「對[{0}]施加」，已改正
- 判讀：KR 是「얻음」，草稿誤寫成「對…施加」，已改正
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N452　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/AbEventsResultLog.json|/dataList/97/content/b8bb2ee9`　abnormality，批次 20261101T082138-09611415，來源 claude

- KR：`<color=#ebcaa2>[{0}]이</color> <color=#e30000>[{1}]</color> {2}<color=#ebcaa2> 얻음</color>`
- 現行譯文：`<color=#ebcaa2>[{0}]</color><color=#ebcaa2>獲得了</color>{2} <color=#e30000>[{1}]</color>`
- 註記：KR 是「얻음」（獲得），草稿寫成「對[{0}]施加」，已改正
- 判讀：同 #91：草稿誤寫「對…施加」，已改為獲得
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N453　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/AbEventsResultLog.json|/dataList/99/content/6ba7d112`　abnormality，批次 20261101T082138-09611415，來源 claude

- KR：`<color=#ebcaa2>[{0}] %[{1}] {2}/%<color=#ebcaa2> 감소</color>`
- 現行譯文：`<color=#ebcaa2>[{0}]<color=#ebcaa2>減少了</color>%{2} [{1}]/%`
- 註記：KR 的第一個 <color> 沒有結束標籤，照 KR 保留兩個開始標籤、一個結束標籤
- 判讀：KR 第一個 <color> 無結束標籤，照 KR 保留標籤結構
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N454　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/AbEvents_Mirror6.json|/dataList/2/failureDesc/0/4d1d5087`　abnormality，批次 20261101T082211-aadd68ce，來源 claude

- KR：`{0} 수감자가 나섰지만, 형태를 구분하기 힘든 그 검은 인영에게 공격은 닿지 않았고, {0} 수감자에게 상처만 늘어날 뿐이었다.⏎⏎검은 것이 다시 환상체의 품 속으로 빨려 들어갔고, 우리는 다시 공격 받기 전에 이 곳을 빠르게 빠져나가기로 했다.⏎⏎…하지만 아직도, 우리의 곁에 그 환상체의 숨결이 감도는 듯한 기분이 든다.`
- 現行譯文：`罪人{0}挺身而出，但攻擊沒能觸及那難以辨清形體的黑色人影，罪人{0}身上的傷口只是越來越多。⏎⏎那黑色的某物再次被吸入了異想體的懷中，我們決定在再次遭受攻擊前，迅速離開這裡。⏎⏎……然而，我們仍覺得那異想體的氣息彷彿還縈繞在身旁。`
- 註記：draft 少一個 {0}，且把「攻擊沒有觸及」誤譯成「遭受攻擊」、「숨결」誤譯成「氣體」，第一段與末段依 KR 重譯
- 判讀：草稿少 {0} 並誤譯攻擊與氣息，首尾段依 KR 重譯
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N455　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/AbEvents_Mirror6.json|/dataList/3/failureDesc/0/37bb9c78`　abnormality，批次 20261101T082211-aadd68ce，來源 claude

- KR：`{0} 수감자는 장미가 눈치채지 못하게 다가가, 한 번에 무기를 휘둘렀다.⏎⏎하지만 예상보다 가시가 단단했는지, 가시는 잘리지 않았고…⏎⏎장미는 그걸 눈치챈 듯, 가시 돋힌 줄기로 수감자의 몸을 감쌌다.⏎⏎…수감자가 어떻게 되었는지는 굳이 말하고 싶지는 않다.`
- 現行譯文：`罪人{0}悄無聲息地接近薔薇，猛然揮下武器。⏎⏎然而也許是荊棘比預想的還要更為堅硬，荊棘並沒有被切斷……⏎⏎薔薇似乎注意到了這點，用長著荊棘的莖纏繞住了罪人的身體。⏎⏎……我不想特地描繪出罪人經歷了什麼。`
- 註記：「什麽」改為臺灣標準字「什麼」
- 判讀：「什麽」改臺灣標準字「什麼」
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N456　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Passive_Ego.json|/dataList/120/desc/5c6b2971`　ally，批次 20261101T075154-40140daf，來源 claude

- KR：`[Breath] 위력 또는 횟수를 얻을 때, 자신의 [Breath] 횟수 1 증가 (턴당 2회)⏎⏎자신이 [PersonalityBreath] 또는 [PersonalityVibration]이면, 기본 공격 가중치가 1이거나 [WideAreaRampage]인 기본 스킬로 가하는 피해량 +10%⏎⏎공격 종료 시 대상이 흐트러짐 상태거나 사망했으면, 다음 턴에 대상이 보유한 [Vibration] 위력의 1/3만큼을 다른 무작위 적들에게 나누어 부여 (소수점 올림, 집중 전투면, 부위로 판정)⏎- 이 효과는 E.G.O 기프트 거울 촉각 공감각 효과와 중첩되어 발동하지 않음`
- 現行譯文：`自身獲得 [Breath] 強度或回數時，使自身增加1回 [Breath] (每回合最多2次)⏎⏎若自身為 [PersonalityBreath] 或 [PersonalityVibration]，則使自身攻擊容量為1的基礎技能或 [WideAreaRampage] 的基礎技能造成的傷害+10%⏎⏎攻擊結束時，若目標陷入混亂或陣亡，則下回合將目標帶有的 [Vibration] 強度的1/3，分散施加給其他隨機敵方單位(無條件進位，若是集中遭遇戰，則改為以部位判定)⏎- 該效果不會與E.G.O飾品鏡觸覺通感的效果疊加`
- 註記：KR 只有一個 [Vibration]，草稿的第二個改寫；草稿「使除該單位以外的全體敵方單位」與 KR「다른 무작위 적들에게 나누어 부여」不符，改為分散施加給其他隨機敵方單位
- 判讀：刪去草稿多餘 [Vibration]，並依 KR 改為分散施加給其他隨機敵人
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N457　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Passives-BossRaid.json|/dataList/20/desc/17a6cc10`　ally，批次 20261101T075401-d4a223e3，來源 claude

- KR：`[Combustion], [Laceration], [Vibration], [Burst], [Sinking]으로 받는 피해량 -30% (소수점 버림[TabExplain], [VibrationExplosion]로 인한 흐트러짐 손상은 제외)⏎⏎[PanicChangeLock] 효과가 적용되며, 턴 종료 시 패닉 상태면 정신력이 45로 변경됨⏎⏎턴 시작 시 아래 효과 적용⏎- [ChoiSwordsmanshipOne]이고, [ChoiSwordsmanship] 수치가 5 미만이면, (5 - [ChoiSwordsmanship] 수치)만큼 <color=#ff6000><mark color=#ff000040><b><u>[Vulnerable]</color></mark></b></u>을 얻음⏎- [ChoiSwordsmanshipMany]이고, [ChoiSwordsmanship] 수치가 5 초과면, 초과하는 값만큼 <color=#ff6000><mark color=#ff000040><b><u>[Vulnerable]</color></mark></b></u>을 얻음`
- 現行譯文：`受到 [Combustion]、[Laceration]、[Vibration]、[Burst] 與 [Sinking] 造成的傷害-30%(無條件捨去[TabExplain]；[VibrationExplosion] 造成的混亂閾值前移除外)⏎⏎使自身獲得 [PanicChangeLock]；回合結束時，若自身處於恐慌狀態，則使自身的理智值變為45點⏎⏎回合開始時，使以下效果生效：⏎- 自身處於 [ChoiSwordsmanshipOne] 時，若自身的 [ChoiSwordsmanship] 層數低於5層，則獲得(5 - [ChoiSwordsmanship] 層數)層<color=#ff6000><mark color=#ff000040><b><u>[Vulnerable]</color></mark></b></u>⏎- 自身處於 [ChoiSwordsmanshipMany] 時，若自身的 [ChoiSwordsmanship] 層數高於5層，則獲得與超出層數等量的<color=#ff6000><mark color=#ff000040><b><u>[Vulnerable]</color></mark></b></u>`
- 註記：補上草稿漏掉的 [TabExplain]；草稿在條目中多加的換行依 KR 合併；「정신력이 45로 변경됨」改「變為45點」
- 判讀：補回 [TabExplain] 並依 KR 合併換行
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N458　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Passives.json|/dataList/151/desc/31144723`　ally，批次 20261101T075528-57d4152f，來源 claude

- KR：`상대의 약점, 취약 속성(내성 1 초과) 으로 공격할 때, 공격 전 대상의 <sprite name="Burst"><color=#e30000><u><link="Burst">파열</color></link></u>만큼 <sprite name="Breath"><color=#f8c200><u><link="Breath">호흡</color></link></u>을 얻음 (턴 당 최대 20)`
- 現行譯文：`使用目標抗性為「脆弱」或「致命」(高於×1)的技能攻擊目標時，目標每帶有1級<sprite name="Burst"><color=#e30000><u><link="Burst">破裂</color></link></u>強度，使自身在攻擊前增加1級<sprite name="Breath"><color=#f8c200><u><link="Breath">呼吸法</color></link></u>強度(每回合最多20級)`
- 註記：草稿拿掉了 sprite 標籤，依 KR 補回並把標籤內的韓文狀態名改為中文
- 判讀：補回 sprite 標籤並把標籤內狀態名改成中文
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N459　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Passives.json|/dataList/169/desc/2bb295a5`　ally，批次 20261101T075528-57d4152f，來源 claude

- KR：`<style="highlight">- 상시 적용 : 탄환 7개 보유</style>⏎- <sprite name="Laceration"><color=#e30000><u><link="Laceration">출혈</color></link></u>이 부여된 대상에게 앞면 적중시 <sprite name="Unjust_Enrichment"><color=#f8c200><u><link="Unjust_Enrichment">부당 수익</color></link></u>을 얻음. (최대 4)⏎<style="highlight">- 탄환 소모하는 코인 적중 시 <sprite name="Unjust_Enrichment"><color=#f8c200><u><link="Unjust_Enrichment">부당 수익</color></link></u>을 전부 소모하여, 피해량의 (25 x 소모한 <sprite name="Unjust_Enrichment"><color=#f8c200><u><link="Unjust_Enrichment">부당 수익</color></link></u>)% 만큼 추가 피해를 줌.</style>⏎- [Bullet]이 없는 경우, 대신 크리티컬 적중 시 <sprite name="Unjust_Enrichment"><color=#f8c200><u><link="Unjust_Enrichment">부당 수익</color></link></u>을 전부 소모하여, (5 x 소모한 <sprite name="Unjust_Enrichment"><color=#f8c200><u><link="Unjust_Enrichment">부당 수익</color></link></u>)%만큼 추가 피해를 줌`
- 現行譯文：`<style="highlight">- 常駐效果：帶有7發彈藥</style>⏎- 正面命中帶有<sprite name="Laceration"><color=#e30000><u><link="Laceration">流血</color></link></u>的目標時，使自身獲得1層<sprite name="Unjust_Enrichment"><color=#f8c200><u><link="Unjust_Enrichment">不義之財</color></link></u>。(最多4層)⏎<style="highlight">- 消耗彈藥的硬幣命中時，消耗所有<sprite name="Unjust_Enrichment"><color=#f8c200><u><link="Unjust_Enrichment">不義之財</color></link></u>，並追加造成傷害量(25×消耗的<sprite name="Unjust_Enrichment"><color=#f8c200><u><link="Unjust_Enrichment">不義之財</color></link></u>層數)%的傷害。</style>⏎- 若未帶有 [Bullet]，則改為暴擊命中時消耗所有<sprite name="Unjust_Enrichment"><color=#f8c200><u><link="Unjust_Enrichment">不義之財</color></link></u>，並追加造成(5×消耗的<sprite name="Unjust_Enrichment"><color=#f8c200><u><link="Unjust_Enrichment">不義之財</color></link></u>層數)%的傷害`
- 註記：草稿依 EN 加的「關卡開始時」KR 沒有，拿掉；「[Bullet]이 없는 경우」草稿誤譯「消耗了所有」，改正
- 判讀：刪去依 EN 加的「關卡開始時」並更正 [Bullet] 條件誤譯
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N460　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Passives.json|/dataList/215/desc/92fc79d3`　ally，批次 20261101T075528-57d4152f，來源 claude

- KR：`이번 전투에서 체력이 0이 되는 피해를 받았을 때, 해당 피해를 받지 않고, 해당 턴 동안 체력이 1로 유지됨. (전투 당 1회)⏎⏎자신의 스킬, 코인 효과로 [Breath] 위력, 횟수를 얻을 때, 자신을 제외한 [Breath]이 없거나, 가장 낮은 검계 인격 2명의 [Breath] 위력, 횟수 1 증가⏎- 전투에 참여한 검계 조직원이 6명 이상이면, 대신 2 증가`
- 現行譯文：`本場戰鬥中，受到使自身體力降至0點的傷害時，無視該傷害，本回合自身的體力不會低於1點(每場戰鬥最多1次)。⏎⏎透過自身的技能或硬幣效果獲得 [Breath] 強度或回數時，使除自身以外未帶有 [Breath] 或其數值最低的2名劍契組人格的 [Breath] 強度與回數各+1⏎- 若參戰的劍契組成員不低於6名，則改為各+2`
- 註記：KR 只有3個 [Breath]，草稿多出的 [Breath] 改寫
- 判讀：刪去多餘 [Breath]，與 KR 數量一致
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N461　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Passives.json|/dataList/313/summary/f596aaff`　ally，批次 20261101T075528-57d4152f，來源 claude

- KR：`탄환을 얻으면, 다음 턴에 신속을 얻음. 대상 처치시, 재장전`
- 現行譯文：`自身獲得彈藥時，下回合獲得迅捷。擊殺目標時，重新裝填`
- 註記：KR 是韓文「재장전」不是 [ID]，拿掉草稿的 [ReloadLament]，譯「重新裝填」
- 判讀：KR 是韓文「재장전」不是 ID，拿掉 [ReloadLament]
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N462　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Passives.json|/dataList/365/desc/3480dd14`　ally，批次 20261101T075528-57d4152f，來源 claude

- KR：`전투 시작 시 자신의 잃은 체력이 80% 이상이고 [Combustion]이 30 미만이면, [Combustion]을 30까지 얻음. (스테이지 당 1회)⏎⏎턴 종료 시 자신의 [Combustion] 6당, 다음 턴에 [AttackUp] 1 얻음 (최대 5)⏎⏎사망 시 아래의 효과 발동⏎- 모든 적에게 [Combustion] 2 부여. 자신의 [Combustion]을 나누어 부여 (1명당 최대 3)⏎- 가장 부족한 속성의 E.G.O 자원 2종 2개씩 획득⏎⏎홍염살 최대 감응 【열화침식】 스킬의 효과로 사망하였으면, 효과가 강화됨⏎- 모든 적에게 [Combustion] 3 부여. 자신의 [Combustion]을 나누어 부여 (1명당 최대 5)⏎- 가장 부족한 속성의 E.G.O 자원 2종 2개씩 획득⏎- 자신의 [Combustion]이 30 이상이었으면, 대기 해제되는 인원 1명에게 [AttackUp] 3 부여`
- 現行譯文：`戰鬥開始時，若自身已失去體力不低於80%，且自身的 [Combustion] 強度低於30級，則使自身增加至30級 [Combustion] 強度(每個關卡最多1次)⏎⏎回合結束時，自身每帶有6級 [Combustion] 強度，下回合使自身獲得1層 [AttackUp] (最多5層)⏎⏎陣亡時，觸發以下效果：⏎- 使全體敵方單位增加2級 [Combustion] 強度，並將自身的 [Combustion] 強度分散施加給敵方單位(每名敵方單位最多3級)⏎- 獲得持有量最少的2種屬性罪孽碎片各2個⏎⏎若自身因技能「紅艷煞 最大感應【劣化侵蝕】」的效果而陣亡，則強化效果：⏎- 使全體敵方單位增加3級 [Combustion] 強度，並將自身的 [Combustion] 強度分散施加給敵方單位(每名敵方單位最多5級)⏎- 獲得持有量最少的2種屬性罪孽碎片各2個⏎- 若自身的 [Combustion] 強度不低於30級，則使解除待命的1名友方單位獲得3層 [AttackUp]`
- 註記：「스테이지 당 1회」改「每個關卡最多1次」（草稿寫每場戰鬥）
- 判讀：스테이지 당 1회 改「每個關卡最多1次」
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N463　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Passives.json|/dataList/407/desc/e904cb1d`　ally，批次 20261101T075528-57d4152f，來源 claude

- KR：`전투에서 퇴각할 때, 아군에게 편성 순서 순으로 다음 턴에 공격 레벨 증가 1 부여⏎- 부여 대상 수 : 자신의 [StartXichun] 수치⏎- 부여 대상이 "가씨 가문"이면, 대신 다음 턴에 공격 레벨 증가 2 부여⏎- 다음 턴에 복귀, 대기 해제되는 인격에게는 부여되지 않음⏎⏎대기 해제 또는 복귀로 등장한 턴에 자신이 [WaitingXichun]를 얻고, 전투 시작시 스킬 순서상 가장 먼저 '적격 - 시【始】'를 무작위 대상에게 사용함`
- 現行譯文：`撤退時，依編隊順序在下回合使友方單位獲得1層攻擊等級提升⏎- 影響的友方單位數：自身的 [StartXichun] 層數⏎- 若該友方單位屬於「賈氏家族」，則改為下回合獲得2層攻擊等級提升⏎- 不對下回合重返戰鬥或解除待命的人格生效⏎⏎自身解除待命或重返戰鬥的回合，使自身獲得 [WaitingXichun]；戰鬥開始時，作為技能順序中最優先的技能，隨機對目標使用「赤擊 - 始」`
- 註記：KR 沒有 [TabExplain]，拿掉；技能名依既有「赤擊 - 始」（KR「시【始】」不重複寫）
- 判讀：刪去 KR 沒有的 [TabExplain]；技能名依既有「赤擊 - 始」
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N464　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Passives.json|/dataList/413/desc/c871b491`　ally，批次 20261101T080902-944f2800，來源 claude

- KR：`상시 적용 : [BulletPropellantAlly] 12개 보유⏎⏎[BulletPropellantAlly], [BulletPropellantSpecialAlly]을 소모하는 코인을 굴릴 때, 해당 특수 탄환이 없는 경우에도 공격을 취소하지 않는 대신 해당 코인의 적중 시 [Combustion] 부여, [Combustion] 횟수 증가 효과가 발동하지 않음⏎⏎이번 전투에서 처음으로 수비 스킬을 장착하였거나 [BulletPropellantAlly]을 전부 사용, 또는 흐트러졌다면, 해당 턴 종료 시 [BulletPropellantSpecialAlly] 8발을 장전하고 [IrritationAlly] 1 얻음 (전투당 1회)⏎- 흐트러짐 상태에서 효과 발동 시, 흐트러짐 상태 해제 (강제 흐트러짐 제외)⏎- [BulletPropellantAlly]이 남아있으면 전부 소멸⏎- 자신이 소모한 [BulletPropellantAlly]과 [BulletPropellantSpecialAlly]의 합이 8 이상이면, [IrritationAlly]이 [HugeIrritationAlly]으로 변경⏎⏎[BulletPropellantSpecialAlly]의 마지막 탄환을 소모하는 기본 공격 스킬을 사용할 경우,⏎- 공격 시작 전 공격 가중치 +2⏎- 다음 턴에 [BeastEyesAlly] 1 얻음`
- 現行譯文：`常駐效果：帶有12發 [BulletPropellantAlly]⏎⏎擲出消耗 [BulletPropellantAlly] 或 [BulletPropellantSpecialAlly] 的硬幣時，即使自身未帶有該特殊彈藥，攻擊也不會取消，但該硬幣的命中時增加 [Combustion] 強度與施加 [Combustion] 回數的效果不會生效⏎⏎若本場戰鬥中自身首次裝備了守備技能，或消耗了自身所有 [BulletPropellantAlly]，或自身陷入混亂，則在該回合結束時，使自身裝填8發 [BulletPropellantSpecialAlly] 並獲得1層 [IrritationAlly] (每場戰鬥最多1次)⏎- 若在混亂狀態下觸發該效果，則解除自身的混亂(不包括強制混亂)⏎- 若自身仍有剩餘的 [BulletPropellantAlly]，則全部消除⏎- 若自身消耗的 [BulletPropellantAlly] 與 [BulletPropellantSpecialAlly] 之和不低於8發，則使自身的 [IrritationAlly] 變更為 [HugeIrritationAlly]⏎⏎使用消耗 [BulletPropellantSpecialAlly] 最後一發彈藥的基礎攻擊技能時，⏎- 攻擊開始前，使攻擊容量+2⏎- 下回合使自身獲得1層 [BeastEyesAlly]`
- 註記：草稿多了 [TabExplain]，且漏了「攻擊容量+2」的數值；另依 KR 刪去草稿自 EN 加入的「關卡開始時」
- 判讀：刪去多餘 [TabExplain] 與 EN 加入的「關卡開始時」，補回數值
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N465　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Passives.json|/dataList/427/desc/49e59d20`　ally，批次 20261101T080902-944f2800，來源 claude

- KR：`[PenetratingSword]을 보유하였다면, 기본 공격 스킬을 사용하여 적에게 적중 전에 [PenetratingSword] 1을 소모하여 [Sinking] 1 부여⏎- 이번 턴에 마지막으로 사용하는 기본 공격 스킬의 마지막 코인 적중 전에 남은 [PenetratingSword]이 2 이상이면, 대신 전부 소모하여 그 수치만큼 [Sinking] 부여⏎⏎[SwordCutwithTear]를 3개 보유하였다면, 기본 공격 스킬을 사용하여 적에게 마지막 코인 적중 전에 [SwordCutwithTear]를 전부 소모하여 [Sinking] 3, [Sinking] 횟수 3을 부여하고 해당 코인의 피해량 +50%⏎- 이 효과가 발동하였다면, 해당 턴 종료 시 정신력이 0 미만이면 정신력이 0으로 초기화 (패닉 또는 E.G.O 침식 상태일 경우, 해제). 그 후, 정신력 30 회복`
- 現行譯文：`若自身帶有 [PenetratingSword]，則自身的基礎攻擊技能命中敵方單位前，消耗1層 [PenetratingSword] 並使目標增加1級 [Sinking] 強度⏎- 本回合自身最後使用的基礎攻擊技能的最後一枚硬幣命中前，若自身剩餘的 [PenetratingSword] 不低於2層，則改為消耗全部層數，並使目標增加等同於消耗量的 [Sinking] 強度⏎⏎若自身的 [SwordCutwithTear] 層數為3層，則自身的基礎攻擊技能的最後一枚硬幣命中敵方單位前，消耗所有 [SwordCutwithTear]，使目標增加3級 [Sinking] 強度與3回 [Sinking]，並使該硬幣造成的傷害+50%⏎- 觸發該效果時，若回合結束時自身的理智值低於0點，則使自身的理智值重置為0點(若處於恐慌或E.G.O侵蝕狀態，則解除)，隨後使自身恢復30點理智值`
- 註記：草稿把「이번 턴에 마지막으로 사용하는 기본 공격 스킬」寫成「最後一個技能」，已補上「基礎攻擊」
- 判讀：補上「基礎攻擊」，修正「最後一個技能」
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N466　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Passives.json|/dataList/473/desc/c4e2a5ea`　ally，批次 20261101T082715-e2a4a0c2，來源 claude

- KR：`대상의 [Burst]과 [Sinking]의 합 2당 피해량 +1% (최대 15%)⏎- 정신력이 없는 대상이면, 이 효과와 최댓값이 2배로 증가⏎⏎자신에게 [GrownHorns]이 있으면, 적을 공격할 때 기본 스킬 마지막 코인의 피해량 +10%⏎- 수비 스킬을 장착한 스킬 슬롯 공격 시, 대신 마지막 코인의 피해량 +20%⏎- 합 가능 수비 스킬과 합 진행 시, 추가로 합 위력 +1`
- 現行譯文：`目標帶有的 [Burst]、[Sinking] 強度之和每有2級，則使自身對目標造成的傷害+1%(最多+15%)⏎- 若目標不帶有理智值，則上述效果及其上限改為2倍⏎⏎若自身帶有 [GrownHorns]，則使自身攻擊敵方單位時，基礎技能的最後一枚硬幣造成的傷害+10%⏎- 若自身攻擊的目標行動槽裝備了守備技能，則改為使最後一枚硬幣造成的傷害+20%⏎- 當自身與可拚點守備技能拚點時，額外使自身拚點威力+1`
- 註記：「수비 스킬」依既有多數（172 處）寫「守備技能」，不寫 prev_zh 的「防備技能」
- 判讀：수비 스킬 依既有多數寫「守備技能」
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N467　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Passives.json|/dataList/480/desc/178acacc`　ally，批次 20261101T080902-944f2800，來源 claude

- KR：`전투 중 누적으로 자신의 [Charge] 횟수 10을 소모할 때마다 [Charge] 1 얻음⏎⏎적에게 침잠 위력을 부여하는 기본 공격 스킬 사용시 메인 타겟에게 [Sinking]이 없으면, [Sinking]을 5 얻고 해당 스킬 효과로 부여하는 [Sinking] 위력이 2배가 됨 (턴당 1회, 집중 전투면 부위로 취급)`
- 現行譯文：`本場戰鬥中，每當自身累計消耗10回 [Charge] 時，使自身增加1級 [Charge] 強度⏎⏎自身使用使敵方單位增加沉淪強度的基礎攻擊技能時，若主要目標不帶有 [Sinking]，則使自身獲得5級 [Sinking] 強度，並使該技能效果增加的 [Sinking] 強度變為2倍(每回合最多1次；集中遭遇戰時，以部位為準)`
- 註記：KR 用韓文「침잠 위력」（無 [ID]），依 base.md 寫「沉淪強度」；草稿「增加2倍」改為「變為2倍」
- 判讀：침잠 위력 依 base.md 寫「沉淪強度」，修正倍數說法
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N468　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Passives.json|/dataList/493/desc/c3df8a62`　ally，批次 20261101T080902-944f2800，來源 claude

- KR：`턴 종료 시 [BlessingOfIndexPrescriptAlly]가 3/6/9이면, [UnlockBuffAlly_1]/[UnlockBuffAlly_2]/[UnlockBuffAlly_3] 얻음⏎⏎[UnlockBuffAlly_1]을 얻으면, 정신력을 15 소모하여 <noparse>개화</noparse> E.G.O::대행 상태가 됨⏎- 정신력 소모 전 정신력이 25까지 증가하고 (증가한 정신력/5)만큼 [KarmaOfIndexAlly] 얻음 (소수점 올림)⏎- 이후 턴 시작 시 [UnlockBuff_Base]이 있고, 정신력이 25 이상이면,⏎정신력을 15 소모하여 해당 상태가 됨⏎⏎<noparse>개화</noparse> E.G.O::대행 상태가 될 때,⏎- 해당 턴의 흐트러짐 해제⏎(해제할 수 없는 흐트러짐이면 해당 상태가 될 수 없음)⏎- 해당 상태일 때, [Blandishment] 얻음⏎⏎턴 시작 시 정신력이 0 이하면, 해당 상태 해제`
- 現行譯文：`回合結束時，若自身的 [BlessingOfIndexPrescriptAlly] 層數為3/6/9層，則使自身獲得 [UnlockBuffAlly_1] / [UnlockBuffAlly_2] / [UnlockBuffAlly_3]⏎⏎自身獲得 [UnlockBuffAlly_1] 時，消耗15點理智值，進入<noparse>綻放</noparse>E.G.O::代行狀態⏎- 消耗理智值前，使自身理智值增加至25點，並獲得(增加的理智值÷5)層 [KarmaOfIndexAlly] (無條件進位)⏎- 此後回合開始時，若自身帶有 [UnlockBuff_Base] 且理智值不低於25點，⏎則消耗15點理智值，進入該狀態⏎⏎進入<noparse>綻放</noparse>E.G.O::代行狀態時，⏎- 解除該回合的混亂⏎(若為無法解除的混亂，則無法進入該狀態)⏎- 處於該狀態時，獲得 [Blandishment]⏎⏎回合開始時，若自身理智值不高於0點，則解除該狀態`
- 註記：草稿的 <noparse> 比 KR 多一組、換行與 KR 不同，且「25 이상」「0 이하」誤為「高於」「低於」，依 KR 重排修正
- 判讀：依 KR 重排 <noparse> 與換行並修正 이상／이하 方向
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N469　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Passives.json|/dataList/553/flavor/63cc471a`　ally，批次 20261101T083831-7c1c6b24，來源 claude

- KR：`“지혜성도. 천살성도. 별이 벼려진 검을 두 개나 손에 넣었다면, 그대는 대체 어떤 별이라 할 수 있겠는가.”⏎”지나친 욕심 끝에 망가지거나 베어져버린 별들이 있음을 그대의 아비 또한 말해주었을 터. 두 자루의 검 중, 오직 하나만을 고르시오.”`
- 現行譯文：`「地慧星刀、天殺星刀。如果連這兩把由星辰鍛造的劍都弄到手了，那你到底算是一顆什麼樣的星呢？」⏎「有些星辰因過度貪求而毀壞，或是遭到斬落，這點你的父輩應該也告訴過你才對。在這兩把劍之中，你只能挑選其中一把。」`
- 註記：除 hint 外，第二句原譯誤解 KR（망가지거나 베어져버린 별들이 있음을…말해주었을 터）已重譯；그대의 아비 依蜘蛛巢既有譯法寫「你的父輩」，不寫「師父」
- 判讀：重譯第二句，그대의 아비 依蜘蛛巢既有譯法寫「你的父輩」
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N470　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心中

`note/Passives.json|/dataList/560/desc/844909a9`　ally，批次 20261101T080902-944f2800，來源 claude

- KR：`스테이지에 첫 등장 시, [MiddleFatherSwordOneOutis] 얻음⏎⏎<style="highlight">각 검의 단계 효과를 얻으면, [CountSkillMidOutis]를 아래 수치에 맞게 얻음⏎- [MiddleFatherSwordOneOutis]: 1⏎- [MiddleFatherSwordTwoOutis]: 2⏎- [MiddleFatherSwordThreeOutis]: 2⏎⏎아래 조건 만족 시 [CountSkillMidOutis] 1 감소 (턴당 1회)⏎- ‘<noparse>원한</noparse> 스탬핑’ 스킬 사용 시 (반격으로 발동 포함)⏎- 질투 완전 공명 수의 합이 6 이상이면, 해당 턴 종료 시⏎⏎[CountSkillMidOutis]가 0이면, 자신의 스킬 종료 시 ‘포장을 뜯어볼까’로 일방 공격함⏎- 턴 종료 시 [CountSkillMidOutis]가 0이고, 해당 턴에 ‘포장을 뜯어볼까’를 사용하지 못했으면, 다음 턴 전투 시작 시 해당 스킬 사용⏎- 자신의 검이 [MiddleFatherSwordOneOutis] 상태일 때, 자신의 체력이 80% 미만이면, [CountSkillMidOutis]와 상<noparse>관</noparse> 없이 해당 스킬 사용⏎⏎자신의 체력이 50% 이하면, [MiddleFatherGutsOutis] 3 얻음 (전투당 1회)⏎⏎전투에서 처음으로 흐트러지면, 턴 종료 시 흐트러짐을 해제하고 [ResentmentSpiderOutis]을 15 얻음 (강제 흐트러짐 제외)⏎⏎턴 시작 시 자신의 [ResentmentSpiderOutis]이 15면, 자신의 조작 슬롯의 기본 스킬 하나를 ‘<noparse>원한</noparse> 스탬핑’ 또는 ‘즉결처형’으로 변경 (전투당 3회)⏎- 가장 왼쪽 슬롯의 아래 스킬만 적용⏎- 해당 슬롯에 이미 해당 스킬이 있는 경우, 발동하지 않음</style>`
- 現行譯文：`戰鬥中首次登場時，使自身獲得 [MiddleFatherSwordOneOutis]⏎⏎<style="highlight">獲得劍的各階段效果時，依以下數值使自身獲得 [CountSkillMidOutis]⏎- [MiddleFatherSwordOneOutis]：1層⏎- [MiddleFatherSwordTwoOutis]：2層⏎- [MiddleFatherSwordThreeOutis]：2層⏎⏎滿足以下條件時，使自身減少1層 [CountSkillMidOutis] (每回合最多1次)⏎- 使用技能「<noparse>仇怨</noparse>踏印」時(包含觸發反擊)⏎- 回合結束時，嫉妒完全共鳴數合計不低於6點⏎⏎若自身的 [CountSkillMidOutis] 層數為0層，則自身的技能結束後，使用「來拆開包裝吧」進行單方面攻擊⏎- 回合結束時，若自身的 [CountSkillMidOutis] 層數為0層，且本回合未能使用「來拆開包裝吧」，則下回合戰鬥開始時使用該技能⏎- 若自身的劍處於 [MiddleFatherSwordOneOutis] 狀態且自身的體力低於80%，則<noparse>無視</noparse> [CountSkillMidOutis] 層數，使用該技能⏎⏎若自身的體力不高於50%，則使自身獲得3層 [MiddleFatherGutsOutis] (每場戰鬥最多1次)⏎⏎本場戰鬥中自身首次陷入混亂時，回合結束時解除自身混亂並使自身獲得15層 [ResentmentSpiderOutis] (不包含強制混亂)⏎⏎回合開始時，若自身的 [ResentmentSpiderOutis] 層數為15層，則使自身行動槽中的1個基礎技能轉變為「<noparse>仇怨</noparse>踏印」或「即刻處刑」(每場戰鬥最多3次)⏎- 僅對位於儀錶盤最左側行動槽下方一行的技能生效⏎- 若該行動槽已有該技能，則不會觸發本效果</style>`
- 註記：KR 的 상<noparse>관</noparse> 沒有對應的中文字，把第三組 <noparse> 放在「無視」
- 判讀：KR 的 <noparse> 無對應中文字，把第三組放在「無視」
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N471　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心中

`note/Passives.json|/dataList/565/desc/844909a9`　ally，批次 20261101T080902-944f2800，來源 claude

- KR：`스테이지에 첫 등장 시, [MiddleFatherSwordOneOutis] 얻음⏎⏎<style="highlight">턴 시작시 [CountSkillMidOutis]가 0이면, [CountSkillMidOutis]를 아래 수치에 맞게 얻음⏎- [MiddleFatherSwordOneOutis]: 1⏎- [MiddleFatherSwordTwoOutis]: 2⏎- [MiddleFatherSwordThreeOutis]: 2⏎⏎아래 조건 만족 시 [CountSkillMidOutis] 1 감소 (턴당 1회)⏎- ‘<noparse>원한</noparse> 스탬핑’ 스킬 사용 시 (반격으로 발동 포함)⏎- 질투 완전 공명 수의 합이 6 이상이면, 해당 턴 종료 시⏎⏎[CountSkillMidOutis]가 0이면, 자신의 스킬 종료 시 ‘포장을 뜯어볼까’로 일방 공격함⏎- 턴 종료 시 [CountSkillMidOutis]가 0이고, 해당 턴에 ‘포장을 뜯어볼까’를 사용하지 못했으면, 다음 턴 전투 시작 시 해당 스킬 사용⏎- 자신의 검이 [MiddleFatherSwordOneOutis] 상태일 때, 자신의 체력이 80% 미만이면, [CountSkillMidOutis]와 상<noparse>관</noparse> 없이 해당 스킬 사용⏎⏎자신의 체력이 50% 이하면, [MiddleFatherGutsOutis] 3 얻음 (전투당 1회)⏎⏎전투에서 처음으로 흐트러지면, 턴 종료 시 흐트러짐을 해제하고 [ResentmentSpiderOutis]을 15 얻음 (강제 흐트러짐 제외)⏎⏎턴 시작 시 자신의 [ResentmentSpiderOutis]이 15면, 자신의 조작 슬롯의 기본 스킬 하나를 ‘<noparse>원한</noparse> 스탬핑’ 또는 ‘즉결처형’으로 변경 (전투당 3회)⏎- 가장 왼쪽 슬롯의 아래 스킬만 적용⏎- 해당 슬롯에 이미 해당 스킬이 있는 경우, 발동하지 않음</style>`
- 現行譯文：`戰鬥中首次登場時，使自身獲得 [MiddleFatherSwordOneOutis]⏎⏎<style="highlight">回合開始時，若自身的 [CountSkillMidOutis] 層數為0層，則依劍的階段，使自身獲得以下數值的 [CountSkillMidOutis]⏎- [MiddleFatherSwordOneOutis]：1層⏎- [MiddleFatherSwordTwoOutis]：2層⏎- [MiddleFatherSwordThreeOutis]：2層⏎⏎滿足以下條件時，使自身減少1層 [CountSkillMidOutis] (每回合最多1次)⏎- 使用技能「<noparse>仇怨</noparse>踏印」時(包含觸發反擊)⏎- 回合結束時，嫉妒完全共鳴數合計不低於6點⏎⏎若自身的 [CountSkillMidOutis] 層數為0層，則自身的技能結束後，使用「來拆開包裝吧」進行單方面攻擊⏎- 回合結束時，若自身的 [CountSkillMidOutis] 層數為0層，且本回合未能使用「來拆開包裝吧」，則下回合戰鬥開始時使用該技能⏎- 若自身的劍處於 [MiddleFatherSwordOneOutis] 狀態且自身的體力低於80%，則<noparse>無視</noparse> [CountSkillMidOutis] 層數，使用該技能⏎⏎若自身的體力不高於50%，則使自身獲得3層 [MiddleFatherGutsOutis] (每場戰鬥最多1次)⏎⏎本場戰鬥中自身首次陷入混亂時，回合結束時解除自身混亂並使自身獲得15層 [ResentmentSpiderOutis] (不包含強制混亂)⏎⏎回合開始時，若自身的 [ResentmentSpiderOutis] 層數為15層，則使自身行動槽中的1個基礎技能轉變為「<noparse>仇怨</noparse>踏印」或「即刻處刑」(每場戰鬥最多3次)⏎- 僅對位於儀錶盤最左側行動槽下方一行的技能生效⏎- 若該行動槽已有該技能，則不會觸發本效果</style>`
- 註記：KR 的 상<noparse>관</noparse> 沒有對應的中文字，把第三組 <noparse> 放在「無視」
- 判讀：同 #560：<noparse> 位置調整
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N472　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Passives.json|/dataList/610/desc/ef8a86e0`　ally，批次 20261101T082715-e2a4a0c2，來源 claude

- KR：`턴 종료시 자신의 [ChargeKhaki] 횟수만큼 다음 턴에 보호막 얻음⏎자신의 [ChargeKhaki] 위력이 2/3/5 이상이면, 기본 스킬 최종 위력 +1/+2/+3`
- 現行譯文：`回合結束時，在下回合使自身獲得等同於自身 [ChargeKhaki] 回數的護盾值⏎若自身的 [ChargeKhaki] 強度不低於2/3/5級，則使基礎技能的最終威力+1/+2/+3`
- 註記：prev_zh 的錯字「機本技能」改為「基礎技能」
- 判讀：更正錯字「機本技能」為「基礎技能」
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N473　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Passives_Assist-a1c9p3.json|/dataList/10/desc/91e8e79f`　ally，批次 20261101T080924-6669d682，來源 claude

- KR：`턴 시작 시 자신을 제외한 속도가 가장 낮은 아군 1명에게 다음 턴에 [Agility] (자신의 [Binding] / 2) 부여 (최대 2. 소수점 버림[TabExplain])⏎⏎모든 아군 중에서 자신이 속도가 가장 낮으면, 기본 공격 스킬로 가하는 피해량 +10%⏎- 자신의 속도가 1이면, 대신 피해량 +20%`
- 現行譯文：`回合開始時，使除自身以外速度值最低的1名友方單位在下回合獲得(自身的 [Binding] 層數÷2)層 [Agility] (最多2層；無條件捨去[TabExplain])⏎⏎若自身為全體友方單位中速度值最低者，則自身的基礎攻擊技能造成的傷害+10%⏎- 若自身的速度值為1，則改為造成的傷害+20%`
- 註記：draft 的 [Binding] 出現兩次、漏了「除自身以外」，第二段多了 KR 沒有的「改為」，已依 KR 修正
- 判讀：補上「除自身以外」並刪去多餘「改為」
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N474　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Ego_Personality-01.json|/dataList/6/levelList/0/desc/5dd29677`　ally，批次 20261101T081006-8d80804b，來源 claude

- KR：`[WideAreaRampage]⏎이 스킬의 효과로 무작위 [SinkingWhite] 부여시 앞면이 나올 확률로 죽은나비를 부여. 부여하지 않은 경우, 산나비를 부여. (나비 1당 개별 판정)⏎⏎로보토미 E.G.O::엄숙한 애도 이상이면,⏎- [BeforeAttack] (우울 공명 수 + 1)만큼 무작위 [BulletLament]를 얻음 (최대 7)⏎- [EndSkill] [ReloadLament]`
- 現行譯文：`[WideAreaRampage]⏎透過本技能的效果隨機施加 [SinkingWhite] 時，以硬幣朝向正面的機率施加亡蝶回數。若自身未施加亡蝶回數，則改為增加生蝶強度(每隻蝶分別判定)⏎⏎若腦葉公司E.G.O::莊嚴哀悼 李箱使用本技能，則⏎- [BeforeAttack] 隨機使自身獲得(憂鬱共鳴數+1)點 [BulletLament] (最多7點)⏎- [EndSkill] 使自身 [ReloadLament]`
- 註記：draft 的 [SinkingWhite] 比 KR 多一個：KR 是「나비 1당 개별 판정」，改寫「每隻蝶分別判定」；補上「隨機」（무작위）與 KR 的空行
- 判讀：刪去多餘 [SinkingWhite]，補上「隨機」與空行
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N475　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Ego_Personality-02.json|/dataList/13/levelList/0/desc/ffbf9585`　ally，批次 20261101T081016-478deefb，來源 claude

- KR：`이 스킬은 외부 효과로 재사용이 불가능함⏎자신의 ([Breath] 위력 + [Charge] 횟수) 4당 합 위력 +1 (최대 5)⏎자신의 [BulletHole] 1당 합 위력 +2⏎[BeforeAttack] [BulletHole]을 전부 소모하고 아래 효과 적용⏎- 소모값 1당 [Breath] 위력 2, [Breath] 횟수 1, [Charge] 횟수 2 얻음⏎- 소모값 1당 피해량 +8%⏎이 스킬에서 [BulletHole]을 5 소모했으면, 정신력 5 회복⏎[EndSkill] [BulletHole] 1 얻음`
- 現行譯文：`本技能不會因外部效果而重複使用⏎自身每帶有4點(自身的 [Breath] 強度+自身的 [Charge] 回數)，則使本技能的拚點威力+1(最多+5)⏎自身每帶有1層 [BulletHole]，則使本技能的拚點威力+2⏎[BeforeAttack] 消耗自身所有 [BulletHole] 並觸發以下效果⏎- 每消耗1層，則使自身獲得2級 [Breath] 強度、1回 [Breath] 與2回 [Charge]⏎- 每消耗1層，則使本技能造成的傷害+8%⏎若本技能消耗了5層 [BulletHole]，則使自身恢復5點理智值⏎[EndSkill] 使自身獲得1層 [BulletHole]`
- 註記：draft 把技能層級的「합 위력」「피해량」寫成「本硬幣」，改為「本技能」；「소모값 1당」多寫的 [BulletHole] 已刪除以符合 KR 的 [ID] 數量
- 判讀：技能層級的數值效果改稱「本技能」，刪去多餘 [BulletHole]
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N476　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Ego_Personality-02.json|/dataList/13/levelList/1/desc/83daff86`　ally，批次 20261101T081016-478deefb，來源 claude

- KR：`이 스킬은 외부 효과로 재사용이 불가능함⏎<style="highlight">이 스킬 공격 가중치보다 낮은 공격 대상 1당 피해량 +5% (집중 전투인 경우, 부위로 판정)</style>⏎자신의 ([Breath] 위력 + [Charge] 횟수) 4당 합 위력 +1 (최대 <style="highlight">7</style>)⏎자신의 [BulletHole] 1당 합 위력 +2⏎<style="highlight">[BeforeAttack] 공격 대상이 1명이면, 피해량 +50% (집중 전투인 경우, 본체로 판정)</style>⏎[BeforeAttack] [BulletHole]을 전부 소모하고 아래 효과 적용⏎- 소모값 1당 [Breath] 위력 2, [Breath] 횟수 1, [Charge] 횟수 2 얻음⏎- 소모값 1당 피해량 +8%⏎이 스킬에서 [BulletHole]을 5 소모했으면, 정신력 5 회복⏎[EndSkill] [BulletHole] 1 얻음`
- 現行譯文：`本技能不會因外部效果而重複使用⏎<style="highlight">本技能的攻擊容量高於攻擊目標數時，每高於1點，使本技能造成的傷害+5%(若為集中遭遇戰，則以部位判定)</style>⏎自身每帶有4點(自身的 [Breath] 強度+自身的 [Charge] 回數)，則使本技能的拚點威力+1(最多+<style="highlight">7</style>)⏎自身每帶有1層 [BulletHole]，則使本技能的拚點威力+2⏎<style="highlight">[BeforeAttack] 若攻擊目標只有1名，則使本技能造成的傷害+50%(若為集中遭遇戰，則以本體判定)</style>⏎[BeforeAttack] 消耗自身所有 [BulletHole] 並觸發以下效果⏎- 每消耗1層，則使自身獲得2級 [Breath] 強度、1回 [Breath] 與2回 [Charge]⏎- 每消耗1層，則使本技能造成的傷害+8%⏎若本技能消耗了5層 [BulletHole]，則使自身恢復5點理智值⏎[EndSkill] 使自身獲得1層 [BulletHole]`
- 註記：draft「每有1名攻擊目標」誤譯 KR「공격 대상이 1명이면」，改為「若攻擊目標只有1名」（同 Skills_Ego_Personality-05 既有寫法）
- 判讀：修正對攻擊目標數的誤譯
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N477　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Ego_Personality-03.json|/dataList/4/levelList/0/coinlist/1/coindescs/1/desc/26a66917`　ally，批次 20261101T081023-2823fe92，來源 claude

- KR：`[BloodDinner_Common_Accumulation] 100이상이면, 1당 피해량 +0.1% (최대 10%)`
- 現行譯文：`若 [BloodDinner_Common_Accumulation] 不低於100，則每1點使本硬幣造成的傷害+0.1%(最多+10%)`
- 註記：草稿重複了 [BloodDinner_Common_Accumulation]（KR 只有一次），改為「每1點」；共用消耗血宴總數是數值，不用「層」
- 判讀：刪去重複 [BloodDinner_Common_Accumulation]，改「每1點」
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N478　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Ego_Personality-04.json|/dataList/9/levelList/2/coinlist/2/coindescs/4/desc/ed36bb13`　ally，批次 20261101T081042-afb7f19a，來源 claude

- KR：`[OnStartCoin] 자신의 [RyoshuEGOPF]을 모두 소모하고, 소모한 수치 5당 피해량 <style="highlight">+1.25% (최대 248%)</style>⏎- 공격 대상이 1명이면, 이 효과가 <style="highlight">2배</style>로 적용됨 (집중 전투인 경우, 본체를 기준으로 판정)`
- 現行譯文：`[OnStartCoin] 消耗自身帶有的所有 [RyoshuEGOPF] 層數，每消耗5層則使本硬幣造成的傷害<style="highlight">+1.25%(最多+248%)</style>⏎- 若目標僅有1名，則上述的傷害增加效果以<style="highlight">2倍</style>計算 (集中遭遇戰時，則以本體為準)`
- 註記：draft 的 highlight 標籤多一組，依 KR 合併為「+1.25%(最多+248%)」
- 判讀：合併多出的 highlight 標籤
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N479　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Ego_Personality-07.json|/dataList/5/levelList/0/desc/f73608bd`　ally，批次 20261101T081103-d897439c，來源 claude

- KR：`[CantIdentify]⏎무작위 대상 공격⏎[BeforeAttack] (최대 공명 수 + 1)만큼 [BoseProjektil]을 얻음 (최대 7)⏎- 해당 공명이 완전 공명이면 [BoseProjektil]을 (최대 공명 수 + 2) 만큼 얻음⏎[BeforeAttack] [BoseProjektil]당 피해량 +2% (최대 14%)⏎[BeforeAttack] [BoseProjektil]당 호흡 1 얻음 (최대 3)⏎[BeforeAttack] 자신의 [Breath] 5당 피해량 +3% (최대 15%)⏎[BeforeAttack] 자신의 [Breath] 5 소모⏎[EndSkill] [BoseProjektil] 전부 소모`
- 現行譯文：`[CantIdentify]⏎隨機指定目標⏎[BeforeAttack] 使自身獲得(最大共鳴數+1)層 [BoseProjektil] (最多7層)⏎- 若該共鳴為完全共鳴，則改為獲得(最大共鳴數+2)層 [BoseProjektil]⏎[BeforeAttack] 自身每帶有1層 [BoseProjektil]，使本技能造成的傷害+2%(最多+14%)⏎[BeforeAttack] 自身每帶有1層 [BoseProjektil]，使自身增加1級呼吸法強度(最多3級)⏎[BeforeAttack] 自身每帶有5級 [Breath] 強度，使本技能造成的傷害+3%(最多+15%)⏎[BeforeAttack] 消耗自身的5級 [Breath] 強度⏎[EndSkill] 消耗所有 [BoseProjektil]`
- 註記：KR 寫韓文「호흡」「진동 폭발」「충전」處，draft 多寫了 [Breath]、[VibrationExplosion]、[Charge]，依 base.md 改成中文名、不加空格
- 判讀：KR 寫韓文處不加 [ID]，改成中文名且不加空格
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N480　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Ego_Personality-09.json|/dataList/4/levelList/0/coinlist/0/coindescs/0/desc/2b0ba9c1`　ally，批次 20261101T081104-b5b99c36，來源 claude

- KR：`[OnSucceedAttack] 대상의 [Laceration] 위력만큼 색욕 속성 피해를 줌.⏎- 색욕 공명이 4 이상이면, 1.2배로 계산⏎(계산 시 반영되는 [Laceration] 위력은 99를 초과할 수 없음.)`
- 現行譯文：`[OnSucceedAttack] 對目標造成等同於其 [Laceration] 強度點色慾傷害⏎- 若色慾共鳴數不低於4，則改為以1.2倍計算⏎(本技能最多計入99級 [Laceration] 強度)`
- 註記：草稿多出 [Laceration]（KR 第二行只寫「1.2배로 계산」），改為「以1.2倍計算」；최대 색욕 완전 공명 수 依 Skills_Ego_Personality-07 寫「最大色慾完全共鳴數」
- 判讀：刪去多餘 [Laceration]，改為「以1.2倍計算」
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N481　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Ego_Personality-09.json|/dataList/5/levelList/0/coinlist/0/coindescs/0/desc/4626035b`　ally，批次 20261101T081104-b5b99c36，來源 claude

- KR：`[OnSucceedAttack] 2 + (색욕 공명 수)만큼 [Laceration] 부여⏎(공명 수 비례 최대 [Laceration] 부여량 : 4)`
- 現行譯文：`[OnSucceedAttack] 使目標增加(2+色慾共鳴數)級 [Laceration] 強度⏎(依共鳴數增加的 [Laceration] 強度最多為4級)`
- 註記：KR 第二行「공명 수 비례 최대 [Laceration] 부여량」草稿漏了 [Laceration]，依 KR 改寫
- 判讀：依 KR 補回第二行 [Laceration]
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N482　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Ego_Personality-10.json|/dataList/2/levelList/0/coinlist/0/coindescs/0/desc/3dc8d9ac`　ally，批次 20261101T081153-e1432077，來源 claude

- KR：`[OnSucceedAttack] 대상에게 [Combustion]이 없으면 화상 4 부여`
- 現行譯文：`[OnSucceedAttack] 若目標未帶有 [Combustion]，則使其增加4級燒傷強度`
- 註記：KR 第二個是韓文「화상」，依 base.md 寫「4級燒傷強度」
- 判讀：KR 是韓文「화상」，寫「4級燒傷強度」
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N483　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Ego_Personality-10.json|/dataList/4/levelList/0/coinlist/0/coindescs/3/desc/b7948175`　ally，批次 20261101T081153-e1432077，來源 claude

- KR：`- 최대 우울 완전 공명 수가 4 이상이면, 이 스킬의 [ShieldManagerCryingToad] 수치가 3으로 부여, 적용됨`
- 現行譯文：`- 若最大憂鬱完全共鳴數不低於4，則本技能賦予與套用的 [ShieldManagerCryingToad] 數值改為3`
- 註記：KR 只有一個 [ShieldManagerCryingToad]（EN 展開成兩個效果），依 KR 改寫為「賦予與套用的數值改為3」
- 判讀：KR 只有一個 [ShieldManagerCryingToad]，依 KR 改寫
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N484　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Ego_Personality-10.json|/dataList/6/levelList/0/desc/d0daaf18`　ally，批次 20261101T081153-e1432077，來源 claude

- KR：`[EndSkill] 대상에게 [Curse]가 있다면 즉시 해당 효과를 2번 발동시키고 [Curse] 2 감소`
- 現行譯文：`[EndSkill] 若目標帶有 [Curse]，則立即觸發該效果2次，並使其 [Curse] 減少2層`
- 註記：KR「해당 효과」不是 [Curse]，草稿多出一個 [Curse]，改為「該效果」
- 判讀：해당 효과 不是 [Curse]，刪去多餘 ID
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N485　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Ego_Personality-12.json|/dataList/11/levelList/0/desc/2988dd1f`　ally，批次 20261101T081159-d9c4fad5，來源 claude

- KR：`아래 조건 만족 시 코인 위력 +1 (최대 3)⏎- 메인 타겟의 [Combustion] 위력 6당⏎- (자신의 [Charge] 위력 - 1)만큼⏎[BeforeAttack] (자신의 [Charge] 위력 - 1)만큼 이 스킬로 부여하는 [Combustion] <noparse>위력 증가</noparse> (최대 2)⏎[BeforeAttack] (최대 공명 수 + 1)만큼 자신의 [Charge] 횟수 증가 (최대 7)⏎[BeforeAttack] 자신의 [Charge] 횟수를 최대 15 소모하여, 소모한 충전 횟수당 피해량 +1% (최대 15%)`
- 現行譯文：`滿足以下條件時，使本技能的硬幣威力增加(最多+3)⏎- 主要目標每帶有6級 [Combustion] 強度，使本技能的硬幣威力+1⏎- 使本技能的硬幣威力+(自身的 [Charge] 強度-1)⏎[BeforeAttack] 使本技能施加的 [Combustion] <noparse>強度增加</noparse>(自身的 [Charge] 強度-1)級(最多2級)⏎[BeforeAttack] 使自身獲得(最大共鳴數+1)回 [Charge] (最多7回)⏎[BeforeAttack] 消耗自身最多15回 [Charge]，每消耗1回，使本技能造成的傷害+1%(最多+15%)`
- 註記：補上 KR 的 <noparse>；刪去多寫的 [Charge]
- 判讀：補上 <noparse>，刪去多寫的 [Charge]
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N486　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Ego_Personality-12.json|/dataList/8/levelList/0/desc/2a9bc3c9`　ally，批次 20261101T081159-d9c4fad5，來源 claude

- KR：`[CantDuel]⏎전투에서 처음 사용시 필요한 E.G.O 자원이 분노 1, 나태 2, 오만 1 감소⏎[MlynarWaiting]이 1 이상이면, 눈부시지 않은 영광 - 광휘(光輝)로 발동됨⏎이 스킬은 적에게 적중하지 않고, 수비 스킬을 발동시키지 않으며, 외부 효과로 재사용이 불가능함⏎[EndSkill] [MlynarWaiting] 1 얻음⏎[EndSkill] 정신력 25 회복⏎[EndSkill] 다음 턴에 [AttackUp] 1 얻음`
- 現行譯文：`[CantDuel]⏎本場戰鬥中首次使用時，消耗的罪孽碎片減少1個暴怒、2個怠惰與1個傲慢⏎若自身的 [MlynarWaiting] 層數不低於1層，則改為發動未照耀的榮光 - 光輝⏎本技能不會命中敵方單位、不會觸發守備技能，且不會因外部效果而重複使用⏎[EndSkill] 使自身獲得1層 [MlynarWaiting]⏎[EndSkill] 使自身恢復25點理智值⏎[EndSkill] 下回合使自身獲得1層 [AttackUp]`
- 註記：刪去 KR 沒有的 [TabExplain]；技能名沿用 dataList/9 既有「未照耀的榮光 - 光輝」
- 判讀：刪去 KR 沒有的 [TabExplain]，技能名沿用既有譯名
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N487　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_personality-01.json|/dataList/21/levelList/1/desc/e2042a0f`　ally，批次 20261101T081203-e4bb157b，來源 claude

- KR：`대상의 [Sinking]과 모든 [SinkingWhite]의 합이 6 이상이면, 코인 위력 +1⏎[WinDuel] [Sinking] 횟수 1 증가⏎[EndSkill] 최대 공명 수만큼 [BulletLament]를 얻음 (최대 6)⏎- 해당 공명이 완전 공명이면, (최대 공명 수 x 2)만큼 얻음 (최대 12)`
- 現行譯文：`若目標的 [Sinking] 強度與兩種 [SinkingWhite] 的點數之和不低於6點，則使本技能的硬幣威力+1⏎[WinDuel] 使目標增加1回 [Sinking]⏎[EndSkill] 使自身獲得(最大共鳴數)點 [BulletLament] (最多6點)⏎- 若該共鳴為完全共鳴，則改為使自身獲得(最大共鳴數×2)點(最多12點)`
- 註記：draft 多寫的 [ID]（KR 用「전부 소모」「추가로 1」「소모값」等省略寫法處）依 KR 刪去；[UnlockBuff_Base] 不在 KR，改回 [UnlockBuffAlly_2]；「7 미만」改「低於7回」
- 判讀：依 KR 刪去多餘 [ID]，改回 [UnlockBuffAlly_2]，「7 미만」改「低於7回」
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N488　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_personality-03.json|/dataList/51/levelList/1/desc/6321e009`　ally，批次 20261101T091739-038e4161，來源 claude

- KR：`[StartBattle] [OathFieldDonQuixote] 3 얻음 (턴당 1회)⏎[StartBattle] 자신과 자신을 제외한 정신력이 가장 낮은 르누아르 소속 아군 인격 1명의 정신력 <style="highlight">5</style> 회복 (턴당 1회)⏎<style="highlight">[StartBattle] 자신의 [NoirShieldUpDonQuixote]가</style>⏎<style="highlight">- 100이면, 대신 ‘거듭 내리쳐, 탄생을 끊으리라!’로 발동됨 (턴당 1회, 이 효과는 일방 공격에도 발동함)</style>⏎<style="highlight">- 100 미만이면, 자신의 최대 체력의 ([NoirShieldUpDonQuixote]/10)%만큼 보호막 얻음 (턴당 1회)</style>⏎[StartBattle] 자신의 [ChargeNoirAlly] 횟수 <style="highlight">3</style> 증가 (턴당 1회)⏎<style="highlight">[WhenUse] 대상의 [Vibration] 위력과 [NoirNailDonQuixote]의 합이 6 이상이면, 코인 위력 +1</style>⏎[WhenUse] 자신의 [ChargeNoirAlly] 위력이 3 이상이면, 코인 위력 +1, <style="highlight">피해량 +10%</style>`
- 現行譯文：`[StartBattle] 使自身獲得3層 [OathFieldDonQuixote] (每回合最多1次)⏎[StartBattle] 使自身與除自身以外理智值最低且屬於漆黑派的1名友方人格恢復<style="highlight">5</style>點理智值(每回合最多1次)⏎<style="highlight">[StartBattle] 若自身帶有的 [NoirShieldUpDonQuixote]</style>⏎<style="highlight">- 為100層，則改為使用「多次鎚下，否決汝的誕生！」(每回合最多1次，本效果在單方面攻擊時也會發動)</style>⏎<style="highlight">- 低於100層，則使自身獲得自身體力上限(自身的 [NoirShieldUpDonQuixote] 層數÷10)%的護盾(每回合最多1次)</style>⏎[StartBattle] 使自身獲得<style="highlight">3</style>回 [ChargeNoirAlly] (每回合最多1次)⏎<style="highlight">[WhenUse] 若目標的 [Vibration] 強度與目標的 [NoirNailDonQuixote] 層數之和不低於6點，則使本技能的硬幣威力+1</style>⏎[WhenUse] 若自身的 [ChargeNoirAlly] 強度不低於3級，則使本技能的硬幣威力+1、<style="highlight">造成的傷害+10%</style>`
- 註記：KR「이 효과는 일방 공격에도 발동함」是「本效果在單方面攻擊時也會發動」，原譯「可作為單方面攻擊使用」意思不同，一併改正
- 判讀：일방 공격에도 발동함 原譯意思不同，一併改正
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N489　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_personality-03.json|/dataList/51/levelList/2/desc/6321e009`　ally，批次 20261101T091739-038e4161，來源 claude

- KR：`[StartBattle] [OathFieldDonQuixote] 3 얻음 (턴당 1회)⏎[StartBattle] 자신과 자신을 제외한 정신력이 가장 낮은 르누아르 소속 아군 인격 1명의 정신력 <style="highlight">6</style> 회복 (턴당 1회)⏎[StartBattle] 자신의 [NoirShieldUpDonQuixote]가⏎- 100이면, 대신 ‘거듭 내리쳐, 탄생을 끊으리라!’로 발동됨 (턴당 1회, 이 효과는 일방 공격에도 발동함)⏎- 100 미만이면, 자신의 최대 체력의 ([NoirShieldUpDonQuixote]/10)%만큼 보호막 얻음 (턴당 1회)⏎[StartBattle] 자신의 [ChargeNoirAlly] 횟수 <style="highlight">4</style> 증가 (턴당 1회)⏎[WhenUse] 대상의 [Vibration] 위력과 [NoirNailDonQuixote]의 합 <style="highlight">6당</style>, 코인 위력 +1 <style="highlight">(최대 2)</style>⏎[WhenUse] 자신의 [ChargeNoirAlly] 위력이 3 이상이면, 코인 위력 +1, 피해량 +<style="highlight">20%</style>`
- 現行譯文：`[StartBattle] 使自身獲得3層 [OathFieldDonQuixote] (每回合最多1次)⏎[StartBattle] 使自身與除自身以外理智值最低且屬於漆黑派的1名友方人格恢復<style="highlight">6</style>點理智值(每回合最多1次)⏎[StartBattle] 若自身帶有的 [NoirShieldUpDonQuixote]⏎- 為100層，則改為使用「多次鎚下，否決汝的誕生！」(每回合最多1次，本效果在單方面攻擊時也會發動)⏎- 低於100層，則使自身獲得自身體力上限(自身的 [NoirShieldUpDonQuixote] 層數÷10)%的護盾(每回合最多1次)⏎[StartBattle] 使自身獲得<style="highlight">4</style>回 [ChargeNoirAlly] (每回合最多1次)⏎[WhenUse] 若目標的 [Vibration] 強度與目標的 [NoirNailDonQuixote] 層數之和<style="highlight">每有6點</style>，則使本技能的硬幣威力+1<style="highlight">(最多+2)</style>⏎[WhenUse] 若自身的 [ChargeNoirAlly] 強度不低於3級，則使本技能的硬幣威力+1、造成的傷害+<style="highlight">20%</style>`
- 註記：KR「이 효과는 일방 공격에도 발동함」是「本效果在單方面攻擊時也會發動」，原譯「可作為單方面攻擊使用」意思不同，一併改正
- 判讀：同 #51 lvl1：修正單方面攻擊的誤譯
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N490　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_personality-04.json|/dataList/17/levelList/2/desc/c93d3c18`　ally，批次 20261101T081234-caea32d9，來源 claude

- KR：`[WhenUse] 자신의 [Vibration] 횟수 2 증가⏎<style="highlight">[WhenUse] 자신의 진동 횟수가 4 이상이면, 합 위력 +1</style>⏎자신의 [Vibration] 횟수 <style="highlight">4</style> 이상이면, 코인 위력 +1`
- 現行譯文：`[WhenUse] 對自身施加2回 [Vibration]⏎<style="highlight">[WhenUse] 若自身的震顫回數不低於4回，則使本技能的拚點威力+1</style>⏎若自身的 [Vibration] 回數不低於<style="highlight">4</style>回，則使本技能的硬幣威力+1`
- 註記：KR 第二行是韓文「진동」，依 base.md 寫「震顫」
- 判讀：KR 第二行是韓文「진동」，依 base.md 寫「震顫」
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N491　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_personality-04.json|/dataList/34/levelList/0/desc/f185c040`　ally，批次 20261101T081234-caea32d9，來源 claude

- KR：`[StartBattle] N사 E.G.O::<noparse>경멸</noparse>, 경외 료슈 우선 공격⏎[StartBattle] 자신의 [BoseProjektil] 3당, 공격 가중치 +1 (최대 2)⏎[WhenUse] 자신의 [Breath]이 5 이상이면, 코인 위력 +1⏎[WhenUse] 자신의 [Breath] 횟수 3 이상이면, 기본 위력 +1⏎[WhenUse] 자신의 [BoseProjektil]당, [Breath] 1 얻음 (최대 7)⏎[EndSkill] 자신의 ([BoseProjektil] X 2) 만큼 정신력 감소 (최대 14 감소)⏎[EndSkill] 전투 시작 시 자신의 [BoseProjektil]이 7이면, 전부 소모하고 자신의 정신력 20 회복`
- 現行譯文：`[StartBattle] 優先指定N公司 E.G.O::<noparse>輕蔑</noparse>，敬畏 良秀⏎[StartBattle] 自身每帶有3層 [BoseProjektil]，則使本技能的攻擊容量+1(最多+2)⏎[WhenUse] 若自身的 [Breath] 強度不低於5級，則使本技能的硬幣威力+1⏎[WhenUse] 若自身的 [Breath] 回數不低於3回，則使本技能的基礎威力+1⏎[WhenUse] 自身每帶有1層 [BoseProjektil]，則使自身增加1級 [Breath] 強度(最多7級)⏎[EndSkill] 使自身失去(自身的 [BoseProjektil] 層數×2)點理智值(最多失去14點)⏎[EndSkill] 若戰鬥開始時自身的 [BoseProjektil] 層數為7層，則將其全部消耗並使自身恢復20點理智值`
- 註記：KR「전부 소모하고」沒有 [BoseProjektil]，改為「將其全部消耗」
- 判讀：KR 無 [BoseProjektil]，改為「將其全部消耗」
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N492　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_personality-04.json|/dataList/56/levelList/2/desc/cc14520d`　ally，批次 20261101T082807-eccb1b7a，來源 claude

- KR：`[DuelCounter]⏎[StartBattle]⏎- 자신의 [NoirSuitAlly]가 60 이상이면,⏎해당 효과가 다음 턴에 [RealisationAlly]로 변경됨⏎- 자신의 [RealisationAlly]가 60 미만이면,⏎해당 효과가 다음 턴에 [NoirSuitAlly]로 변경됨⏎[BeforeUse] 자신의 [RealisationAlly]가 60 이상이면,⏎‘원 킥 - 쿠튀르 뮐티플’로 발동됨 (턴당 1회)⏎- 이 효과가 발동했으면, 공격 종료 시 [RealisationAlly] 전부 소모⏎<style="highlight">[WhenUse] 자신의 [ChargeNoirAlly] 위력이 3/5 이상이면, 코인 위력 +1/+2</style>⏎[WhenUse] 자신의 [ChargeNoirAlly] 횟수 2 증가 (턴당 2회)`
- 現行譯文：`[DuelCounter]⏎[StartBattle]⏎- 若自身的 [NoirSuitAlly] 層數不低於60層，⏎則下回合使該效果轉變為 [RealisationAlly]⏎- 若自身的 [RealisationAlly] 層數低於60層，⏎則下回合使該效果轉變為 [NoirSuitAlly]⏎[BeforeUse] 若自身的 [RealisationAlly] 層數不低於60層，⏎則改為使用「單腳踢 - 多樣訂製」(每回合最多1次)⏎- 觸發該效果後，攻擊結束時消耗所有 [RealisationAlly]⏎<style="highlight">[WhenUse] 若自身的 [ChargeNoirAlly] 強度不低於3/5級，則使本技能的硬幣威力+1/+2</style>⏎[WhenUse] 使自身獲得2回 [ChargeNoirAlly] (每回合最多2次)`
- 註記：KR「60 미만」是「低於」，prev_zh 誤為「不高於」，已改正
- 判讀：60 미만 應為「低於」，已改正
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N493　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_personality-05.json|/dataList/34/levelList/1/desc/1a647773`　ally，批次 20261101T081244-4f341158，來源 claude

- KR：`[StartBattle] [MeursaultSporeBulletReloading] (턴당 1회)⏎<style="highlight">[WhenUseEvade] 공격자의 [Combustion]과 [Laceration]의 합 6당 코인 위력 +1 (최대 3)</style>⏎[OnSucceedEvade] 자신의 체력 10 회복 (턴당 1회)⏎<style="highlight">- [MeursaultWorkerBee] 보유 중이면, 수치당 5씩 회복량 추가</style>`
- 現行譯文：`[StartBattle] 使自身 [MeursaultSporeBulletReloading] (每回合最多1次)⏎<style="highlight">[WhenUseEvade] 攻擊者的 [Combustion] 強度與 [Laceration] 強度之和每有6級，使本技能的硬幣威力+1(最多+3)</style>⏎[OnSucceedEvade] 使自身恢復10點體力(每回合最多1次)⏎<style="highlight">- 若自身帶有 [MeursaultWorkerBee]，則每帶有1層，使恢復量額外增加5點</style>`
- 註記：草稿最後一行把回復量寫成「理智值」；KR「회복량」指上一行的體力回復（EN: heal 5 HP），已改正
- 判讀：最後一行回復量指體力回復，不是理智值
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N494　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_personality-07.json|/dataList/15/levelList/0/desc/05a79114`　ally，批次 20261101T081318-e4766b67，來源 claude

- KR：`[DuelCounter]⏎[StartBattle] [NightPathfinding]이 없으면, 다음 턴에 [NightPathfinding]을 1 얻음 (턴 당 1회 발동)⏎[EndBattle] [NightPathfinding]이 있으면 제거함`
- 現行譯文：`[DuelCounter]⏎[StartBattle] 若自身未帶有 [NightPathfinding]，則下回合使自身獲得1層 [NightPathfinding] (每回合最多1次)⏎[EndBattle] 若自身帶有 [NightPathfinding]，則將其移除`
- 註記：draft「(每回合最多1層)」與 KR「턴 당 1회 발동」不符，改為「每回合最多1次」
- 判讀：턴 당 1회 發動 改為「每回合最多1次」
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N495　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_personality-07.json|/dataList/45/levelList/1/desc/2ca35895`　ally，批次 20261101T081318-e4766b67，來源 claude

- KR：`[BeforeUse] 대상의 [DuelEdgeAlly]가 5면, ‘세치오나투라 디 체르보’로 발동됨 (턴당 1회)⏎[WhenUse] 대상의 [Combustion]과 [Vibration]의 합 <style="highlight">6당, 코인 위력 +1 (최대 2)</style>⏎[WhenUse] 대상의 [DuelEdgeAlly] 수치에 따라 다음 효과 전부 얻음⏎<style="highlight">- 수치 1당 피해량 +6% (최대 30%)</style>⏎- 1 : [Vibration] 위력 부여값 +1⏎- 2 : [Vibration] 횟수 부여값 +1⏎- 3 : [Combustion] 위력 부여값 +1⏎- 4 : [Combustion] 횟수 부여값 +1⏎[WhenUse] [EvolvingManual]이 있거나 대상의 [DuelEdgeAlly]가 3 이상이면, 모든 코인이 [SuperCoin]으로 변경됨`
- 現行譯文：`[BeforeUse] 若目標帶有5層 [DuelEdgeAlly]，則改為使用「肢解鹿軀」(每回合最多1次)⏎[WhenUse] 目標的 [Combustion] 強度與 [Vibration] 強度之和<style="highlight">每有6級，則使本技能的硬幣威力+1(最多+2)</style>⏎[WhenUse] 根據目標的 [DuelEdgeAlly] 層數，相應使本技能獲得以下全部效果⏎<style="highlight">- 每有1層，使本技能造成的傷害+6%(最多+30%)</style>⏎- 不低於1層：增加的 [Vibration] 強度額外+1級⏎- 不低於2層：增加的 [Vibration] 回數額外+1回⏎- 不低於3層：增加的 [Combustion] 強度額外+1級⏎- 不低於4層：增加的 [Combustion] 回數額外+1回⏎[WhenUse] 若自身帶有 [EvolvingManual] 或目標帶有的 [DuelEdgeAlly] 層數不低於3層，則使本技能的所有硬幣轉變為 [SuperCoin]`
- 註記：draft 漏了 KR 的「- 」清單符號與「전부」，已補上
- 判讀：補上 KR 的「- 」清單符號與「전부」
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N496　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_personality-08.json|/dataList/45/levelList/1/desc/bb53b5a4`　ally，批次 20261101T081324-7b7e9481，來源 claude

- KR：`[RougeIshmaelCocoon] 전용 스킬⏎[CanDuelGuard]⏎[StartBattle] 자신의 [ChargeRougeAlly] 횟수가 10 이상이면, 10 소모하여 다음 턴에 편성 순서가 가장 빠른 아군 2명에게 [ResultEnhancement] 1 부여 (턴당 1회)⏎<style="highlight">- 대상이 르루주 소속이면, 다음 턴에 자신의 [ChargeRougeAlly] 횟수가 5 증가하고, 대상에게 [AttackDmgUp] 1 추가 부여</style>⏎[StartBattle] 자신의 최대 체력의 <style="highlight">100%</style>만큼 체력 회복 (턴당 1회)⏎<style="highlight">[WinDuel] 자신에게 부여된 해제 가능한 부정적인 효과 중 무작위 1개 제거 (턴당 1회)</style>⏎[WinDuel] 자신의 [ChargeRougeAlly] 횟수 5 증가 (턴당 4회)`
- 現行譯文：`[RougeIshmaelCocoon] 專用技能⏎[CanDuelGuard]⏎[StartBattle] 若自身的 [ChargeRougeAlly] 回數不低於10回，則消耗10回並在下回合使最先編入的2名友方單位獲得1層 [ResultEnhancement] (每回合最多1次)⏎<style="highlight">- 若對象屬於丹紅派，則下回合使自身的 [ChargeRougeAlly] 回數增加5回，並額外對該對象施加1層 [AttackDmgUp]</style>⏎[StartBattle] 使自身恢復體力上限<style="highlight">100%</style>的體力(每回合最多1次)⏎<style="highlight">[WinDuel] 隨機解除1種自身帶有且可解除的負面狀態(每回合最多1次)</style>⏎[WinDuel] 使自身獲得5回 [ChargeRougeAlly] (每回合最多4次)`
- 註記：補上 KR 的「- 」；르루주 依詞彙表寫「丹紅派」；刪去多出的 [ChargeRougeAlly]
- 判讀：補上「- 」，르루주 依詞彙表寫「丹紅派」，刪去多餘 ID
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N497　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_personality-09.json|/dataList/19/levelList/0/desc/e4e4244b`　ally，批次 20261101T081330-d976f4f0，來源 claude

- KR：`[BeforeUse] 자신의 [BloomingThornsRodionFirst]가 25 이상이면, '돈키호테류 경혈[TabExplain] 오의 피날레'로 발동됨⏎[WhenUse] 자신과 아군 (최대 공명 수)명에게 이번 턴과 다음 턴에 [FestivalFeverRodion] 3 + (자신의 [BloodDinner_Accumulation] 50 당 1) 부여 (최대 10)⏎[WhenUse] 자신과 아군 (최대 공명 수)명에게 [BloomingThornsRodionFirst] 3 부여`
- 現行譯文：`[BeforeUse] 若自身的 [BloomingThornsRodionFirst] 層數不低於25層，則改為使用「唐吉訶德派硬血[TabExplain] 奧義 落幕」⏎[WhenUse] 本回合與下回合使自身與(最大共鳴數)名友方單位獲得3層 [FestivalFeverRodion]，自身每帶有50層 [BloodDinner_Accumulation]，額外獲得1層(最多10層)⏎[WhenUse] 使自身與(最大共鳴數)名友方單位獲得3層 [BloomingThornsRodionFirst]`
- 註記：KR 的 [FestivalFeverRodion] 只出現一次，刪去草稿重複的 ID
- 判讀：刪去重複的 [FestivalFeverRodion]
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N498　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_personality-09.json|/dataList/35/levelList/2/coinlist/1/coindescs/1/desc/4149bcf5`　ally，批次 20261101T081330-d976f4f0，來源 claude

- KR：`<style="highlight">[OnSucceedAttack] 자신의 [Charge] 횟수가 8 이상이면, 4 소모하여 다음 턴에 [HitTakeDamageUp] 1 부여 (턴당 1회)</style>`
- 現行譯文：`<style="highlight">[OnSucceedAttack] 若自身帶有的 [Charge] 回數不低於8回，則消耗4回並使目標於下回合獲得1層 [HitTakeDamageUp] (每回合最多1次)</style>`
- 註記：KR「4 소모하여」沒有 ID，刪去重複的 [Charge]
- 判讀：KR 的 4 소모하여 無 ID，刪去重複 [Charge]
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N499　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_personality-09.json|/dataList/41/levelList/1/coinlist/0/coindescs/2/desc/f826685a`　ally，批次 20261101T081330-d976f4f0，來源 claude

- KR：`<style="highlight">- 대상에게 이 효과가 없으면, 1 추가 부여</style>`
- 現行譯文：`<style="highlight">- 若目標未帶有本效果，則額外施加1層</style>`
- 註記：KR「이 효과」沒有 ID，改為「本效果」
- 判讀：이 효과 無 ID，改為「本效果」
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N500　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_personality-09.json|/dataList/53/levelList/0/desc/506faeee`　ally，批次 20261101T081330-d976f4f0，來源 claude

- KR：`[StartBattle] 자신을 제외한 편성 순서가 가장 빠른 아군 1명 (충전 인격 우선)에게 [AttackUp] 2, 자신의 최대 체력의 5%만큼 보호막 부여 (최소 1. 턴당 1회)⏎- [PersonalityCharge]이면 [AttackUp] 1, 자신의 최대 체력의 5%만큼 보호막 추가 부여⏎⏎[WhenUse] 대상의 [Vibration] 위력이 6 이상이면, 코인 위력 +1⏎[WhenUse] 자신의 [ChargeKhakiAlly] 횟수를 5 소모하여, 최종 위력 +2`
- 現行譯文：`[StartBattle] 使除自身以外編入順序最早的1名友方單位(充能人格優先)獲得2層 [AttackUp] 與自身體力上限5%的護盾(最少1點，每回合最多1次)⏎- 若該人格為 [PersonalityCharge]，則使其額外獲得1層 [AttackUp] 與自身體力上限5%的護盾⏎⏎[WhenUse] 若目標的 [Vibration] 強度不低於6級，則使本技能的硬幣威力+1⏎[WhenUse] 消耗自身5回 [ChargeKhakiAlly] 並使本技能的最終威力+2`
- 註記：KR「충전 인격 우선」是韓文，寫「充能人格優先」，不是 [Charge]
- 判讀：충전 인격 우선 是韓文，寫「充能人格優先」
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N501　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_personality-10.json|/dataList/38/levelList/1/desc/cf79eebf`　ally，批次 20261101T081335-01176b52，來源 claude

- KR：`[CanDuelGuard]⏎턴 시작시 자신의 [Breath] 위력이 25 이상이면 ‘회’로 변경됨⏎⏎[StartBattle] [Breath] 8 얻음 (턴당 1회)⏎[StartBattle] 자신의 [Breath] 횟수 <style="highlight">2</style> 증가 (턴당 1회)`
- 現行譯文：`[CanDuelGuard]⏎回合開始時，若自身的 [Breath] 強度不低於25級，則使本技能變更為「回」⏎⏎[StartBattle] 使自身增加8級 [Breath] 強度 (每回合最多1次)⏎[StartBattle] 使自身獲得<style="highlight">2</style>回 [Breath] (每回合最多1次)`
- 註記：草稿把「25 이상」寫成「高於25級」，已改為「不低於25級」
- 判讀：「25 이상」改為「不低於25級」
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N502　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_personality-10.json|/dataList/52/levelList/1/desc/d8b29dae`　ally，批次 20261101T082850-7ca1553d，來源 claude

- KR：`[DuelCounter]⏎[TransformSinclair] 전용 스킬⏎[StartBattle] 자신에게 [BuffetSinclair]가 없고 자신의 [ChargeRougeAlly] 횟수가 15 미만이면, 최대 체력의 (15 - 현재 [ChargeRougeAlly] 횟수)%만큼 체력을 소모하여 자신의 [ChargeRougeAlly] 횟수를 15로 증가 (턴당 1회)⏎⏎[WhenUse] 대상의 [Laceration] 4당, 최종 위력 +1 (최대 2)⏎<style="highlight">[WhenUse] 자신의 [ChargeRougeAlly] 위력이 3/5 이상이면, 코인 위력 +1/+2</style>`
- 現行譯文：`[DuelCounter]⏎[TransformSinclair] 專用技能⏎[StartBattle] 若自身未帶有 [BuffetSinclair] 且自身的 [ChargeRougeAlly] 回數低於15回，則消耗自身體力上限(15-自身目前的 [ChargeRougeAlly] 回數)%的體力，使自身的 [ChargeRougeAlly] 回數增加至15回(每回合最多1次)⏎⏎[WhenUse] 目標每帶有4級 [Laceration] 強度，則使本技能的最終威力+1(最多+2)⏎<style="highlight">[WhenUse] 若自身的 [ChargeRougeAlly] 強度不低於3/5級，則使本技能的硬幣威力+1/+2</style>`
- 註記：KR「15 미만」「15로 증가」：改為「低於15回」「增加至15回」（prev_zh 為「不高於」「獲得15回」）
- 判讀：미만、15로 증가 改為「低於15回」「增加至15回」
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N503　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_personality-11.json|/dataList/34/levelList/0/coinlist/0/coindescs/1/desc/cb4b126a`　ally，批次 20261101T081347-ffc79d9b，來源 claude

- KR：`[OnSucceedAttack] 대상의 [Vibration]을 최대 3 소모⏎- (소모한 [Vibration] x 2)만큼 자신의 진동 횟수 증가⏎- 다음 턴에 (소모한 [Vibration] - 1)만큼 [Binding] 부여`
- 現行譯文：`[OnSucceedAttack] 消耗目標最多3級 [Vibration] 強度⏎- 使自身的震顫回數增加(消耗的 [Vibration] 強度×2)回⏎- 下回合對目標施加(消耗的 [Vibration] 強度-1)層 [Binding]`
- 註記：KR 第二行寫韓文「진동」、沒有 [ID]，依 base.md 改寫成官方名「震顫」（draft 多了一個 [Vibration]）
- 判讀：韓文「진동」改官方名「震顫」，刪去多餘 [Vibration]
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N504　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_personality-11.json|/dataList/34/levelList/1/coinlist/0/coindescs/1/desc/cb4b126a`　ally，批次 20261101T081347-ffc79d9b，來源 claude

- KR：`[OnSucceedAttack] 대상의 [Vibration]을 최대 3 소모<style="highlight"></style>⏎- (소모한 [Vibration] x 2)만큼 자신의 진동 횟수 증가<style="highlight"></style>⏎- 다음 턴에 (소모한 [Vibration] - 1)만큼 [Binding] 부여<style="highlight"></style>`
- 現行譯文：`[OnSucceedAttack] 消耗目標最多3級 [Vibration] 強度<style="highlight"></style>⏎- 使自身的震顫回數增加(消耗的 [Vibration] 強度×2)回<style="highlight"></style>⏎- 下回合對目標施加(消耗的 [Vibration] 強度-1)層 [Binding]<style="highlight"></style>`
- 註記：KR 第二行寫韓文「진동」、沒有 [ID]，依 base.md 改寫成官方名「震顫」（draft 多了一個 [Vibration]）
- 判讀：同 #34 lvl0：韓文「진동」改「震顫」
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N505　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_personality-12.json|/dataList/32/levelList/1/desc/ec4bc2e6`　ally，批次 20261101T081403-9dccb05b，來源 claude

- KR：`[CantDuel]⏎이 스킬은 수비 스킬을 발동시키지 않고, E.G.O 자원을 획득할 수 없음⏎적(본체)의 잃은 체력 20%당, 코인 위력 +1 (최대 <style="highlight">4</style>)⏎<style="highlight">적(본체)의 잃은 체력 비율만큼 <noparse>피해량 증가</noparse></style>⏎- 적(본체)의 체력이 20% 이하면, <style="highlight">대신</style> 피해량 +<style="highlight">120</style>%⏎<style="highlight">[WhenUse] 다음 턴에 [Agility] 1 얻음 (턴당 1회)</style>`
- 現行譯文：`[CantDuel]⏎本技能不會觸發目標的守備技能；使用本技能不會獲得罪孽碎片⏎目標(若為異想體，則以本體為準)每失去最大體力值的20%，則使本技能硬幣威力+1(最多+<style="highlight">4</style>)⏎<style="highlight">目標(若為異想體，則以本體為準)每失去最大體力值的1%，則使本技能<noparse>造成的傷害+1%</noparse></style>⏎- 若目標(若為異想體，則以本體為準)帶有的體力不高於最大值的20%，則<style="highlight">改為</style>使本技能造成的傷害+<style="highlight">120</style>%⏎<style="highlight">[WhenUse] 下回合使自身獲得1層 [Agility] (每回合最多1次)</style>`
- 註記：草稿漏了 <noparse>，已補；「20% 이하」草稿寫「少於」，改為「不高於」
- 判讀：補上 <noparse>，「이하」改「不高於」
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N506　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_personality-12.json|/dataList/33/levelList/1/desc/f4308365`　ally，批次 20261101T081403-9dccb05b，來源 claude

- KR：`자신의 [Agility]과 대상의 [Binding]의 합 1당, 피해량 +<style="highlight">15</style>% (최대 <style="highlight">60</style>%)⏎<style="highlight">대상에게 보호막이 있으면, 피해량 +30% (집중 전투일 경우, 본체 또는 부위로 판정)</style>⏎<style="highlight">적(본체)의 잃은 체력 비율만큼 <noparse>피해량 증가</noparse></style>⏎- 적(본체)의 체력이 20% 이하면, <style="highlight">대신</style> 피해량 +<style="highlight">100</style>%⏎<style="highlight">[WhenUse] [DefenseDown] 5 부여</style>⏎[WhenUse] 대상의 [Laceration]과 [Vibration]의 합 4당, 코인 위력 +1 (최대 <style="highlight">3</style>)⏎[WinDuel] <style="highlight">[Vibration] 4 부여.</style> [VibrationExplosion]`
- 現行譯文：`自身每帶有1層 [Agility] 以及目標每帶有1層 [Binding]，則使本技能造成的傷害+<style="highlight">15</style>%(最多+<style="highlight">60</style>%)⏎<style="highlight">若目標帶有護盾，則使本技能造成的傷害+30%(若為集中遭遇戰，則以本體或部位判定)</style>⏎<style="highlight">目標(若為異想體，則以本體為準)每失去最大體力值的1%，則使本技能<noparse>造成的傷害+1%</noparse></style>⏎- 若目標(若為異想體，則以本體為準)體力值不高於最大值的20%，則<style="highlight">改為</style>造成的傷害+<style="highlight">100</style>%⏎<style="highlight">[WhenUse] 對目標施加5層 [DefenseDown]</style>⏎[WhenUse] 目標帶有的 [Laceration] 強度與 [Vibration] 強度之和每有4級，則使本技能硬幣威力+1(最多+<style="highlight">3</style>)⏎[WinDuel] <style="highlight">使目標增加4級 [Vibration] 強度</style>，然後使其 [VibrationExplosion]`
- 註記：草稿漏了 <noparse>，已補；「20% 이하」改為「不高於」；集中戰鬥括號依 KR 改為「以本體或部位判定」
- 判讀：補上 <noparse>，「이하」改「不高於」，集中戰鬥括號依 KR
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N507　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/BattleAnnouncerDlg/Announcer_RienSora_54.json|/dataList/1/dlg/6f3f7b80`　announcer，批次 20261101T073214-8d4ed150，來源 claude

- KR：`요시히데 언니가 도, 돌아와도… 가끔 이렇게 뤼엔 씨랑 외, 외출할 수 있을까요?⏎지금 생각하기엔 너무 이른 의문 같구나. 하지만… 안 될 이유는 없지 않겠니.⏎그건 기, 기쁘네요. 헤헤.`
- 現行譯文：`就算良秀姊姊回、回來了……我偶爾還可以像這樣跟禮恩先生一起外、外出嗎？⏎現在思考這個問題似乎還太早了。不過……也沒有不行的理由吧。⏎那真、真令人高興呢。嘿嘿。`
- 註記：draft 的 <ruby=YoshiHide>良秀</ruby> 只是讀音，依 base.md 只留「良秀」；「姐姐」依詞彙表改「姊姊」；第二句「안 될 이유는 없지」原譯「沒有不這麼做的理由」改為「沒有不行的理由」
- 判讀：ruby 只留「良秀」；姊姊、「沒有不行的理由」依規則調整
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N508　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Enemies-a1c10p2.json|/dataList/0/desc/e7693311`　enemy，批次 20261101T081419-de23c58f，來源 claude

- KR：`B3 보스`
- 現行譯文：`B3F BOSS`
- 註記：樓層依 base.md 介面標籤寫法：B3／4층／B2층 보스 → B3F BOSS／4F BOSS／B2F BOSS
- 判讀：B3 보스 依 base.md 介面標籤寫法作「B3F BOSS」
- 理由：註記只是說明處理依據，無待改動，確認即可。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N509　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Passives_Abnormality-a1c10p3.json|/dataList/28/desc/cff4177e`　enemy，批次 20261101T081511-223d5cc6，來源 claude

- KR：`전투 중 <color=#ff6000><mark color=#ff000040><b><u>처음으로 피격</color></mark></b></u>받기 전, 자신의 타겟팅과 상관없이 <color=#ff6000><mark color=#ff000040><b><u>장착한 ‘백화점 설계도’를 사용함</color></mark></b></u> (턴당 1회)⏎⏎기본 스킬로 받는 피해량 -50%⏎⏎<color=#ff6000><mark color=#ff000040><b><u>E.G.O 스킬로 받는 피해량 +100%</color></mark></b></u>⏎⏎재단사 잔느 사르트르의 ‘포착’ 스킬 피격 시 사망함`
- 現行譯文：`戰鬥中<color=#ff6000><mark color=#ff000040><b><u>首次受到攻擊</color></mark></b></u>前，無論自身指定的目標為何，<color=#ff6000><mark color=#ff000040><b><u>使用裝備的「百貨公司的藍圖」</color></mark></b></u>(每回合最多1次)⏎⏎受到基礎技能造成的傷害-50%⏎⏎<color=#ff6000><mark color=#ff000040><b><u>受到E.G.O技能造成的傷害+100%</color></mark></b></u>⏎⏎受到裁縫師簡．沙特的「捕捉」技能攻擊時死亡`
- 註記：KR 的標籤結束順序本身不成巢狀（</color></mark></b></u>），照 KR 原樣保留；百貨公司的藍圖、捕捉、裁縫師簡．沙特、存有之匱乏、以此鐮刃回應！等沿用既有技能／敵人名
- 判讀：KR 的標籤結束順序本身不成巢狀（</color></mark></b></u>），照 KR 原樣保留；百貨公司的藍圖、
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N510　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Passives_Abnormality-a1c7p3.json|/dataList/24/desc/9d7dc415`　enemy，批次 20261101T081552-9b683e8e，來源 claude

- KR：`턴 종료 시 패닉이면 패닉을 해제하고, [BloodDinner]을 100 소모해서 [Enhancement] 3을 얻음⏎- [BloodDinner]을 소모하지 못했으면 [Laceration]을 30 얻고 [Enhancement] 3, 피해량 증가 2를 얻음`
- 現行譯文：`回合結束時，若自身陷入恐慌，則解除自身的恐慌，消耗100層 [BloodDinner] 並使自身獲得3層 [Enhancement]⏎- 若未能消耗 [BloodDinner]，則使自身增加30級 [Laceration] 強度，並獲得3層 [Enhancement] 與2層傷害強化`
- 註記：draft「對自身施加30級…強度」改為「使自身增加」；KR「피해량 증가」無 [ID]，寫官方名「傷害強化」
- 判讀：draft「對自身施加30級…強度」改為「使自身增加」；KR「피해량 증가」無 [ID]，寫官方名「傷害強化」
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N511　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Passives_Abnormality-a1c7p3.json|/dataList/44/desc/5cef77b0`　enemy，批次 20261101T081552-9b683e8e，來源 claude

- KR：`생성된 다음 턴까지 처치되지 않았으면, 해당 턴 종료 시 이 유닛이 제거되며, 제거 시 효과가 발동됨.⏎이후 이 유닛에게 꿰뚫렸던 아군이 최대 체력의 30%만큼 고정 피해를 받고, [Laceration] 위력 10, 횟수 5를 얻음.`
- 現行譯文：`若直到生成後的下回合都未被擊殺，則在該回合結束時去除本單位，並觸發去除時的效果。⏎之後，被本單位刺穿的友方單位受到其體力上限30%的固定傷害，並增加10級 [Laceration] 強度與5回流血。`
- 註記：draft「未在生成的下回合被去除」與 KR「처치되지 않았으면」（未被擊殺）不符，已改正
- 判讀：draft「未在生成的下回合被去除」與 KR「처치되지 않았으면」（未被擊殺）不符，已改正
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N512　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Passives_Abnormality-a1c9p1.json|/dataList/1/desc/bfe6a7b8`　enemy，批次 20261101T081617-a19c6a5e，來源 claude

- KR：`[DuelEdge] 수치에 따라 다음 효과를 얻음 (중복 적용)⏎- 수치 1당 부여된 대상에게 가하는 피해량 +5%, 받는 피해량 -5% ⏎- 1 이상이면, 부여된 대상과 합 진행 시 합 위력 +1⏎- 2 이상이면, 부여된 대상과 합 진행 시 기본 위력 +1⏎- 3 이상이면, 부여된 대상과 합 진행 시 코인 위력 +1, 대상에게 공격 적중 시 [SeveredTendon] 3 부여⏎- 4 이상이면, 부여된 대상과 합 진행 시 최종 위력 +1`
- 現行譯文：`根據目標帶有的 [DuelEdge] 層數，使自身獲得以下效果(可重複適用)：⏎- 目標每帶有1層，則使目標對自身造成的傷害-5%，且自身對其造成的傷害+5%⏎- 與帶有不低於1層的目標拚點時，使自身拚點威力+1⏎- 與帶有不低於2層的目標拚點時，使自身基礎威力+1⏎- 與帶有不低於3層的目標拚點時，使自身硬幣威力+1，且攻擊命中目標時，對其施加3層 [SeveredTendon]⏎- 與帶有不低於4層的目標拚點時，使自身最終威力+1`
- 註記：草稿漏了「(중복 적용)」，已補「(可重複適用)」
- 判讀：草稿漏了「(중복 적용)」，已補「(可重複適用)」
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N513　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Passives_Abnormality-a1c9p2.json|/dataList/8/desc/3e25bdaa`　enemy，批次 20261101T091916-31c65cb3，來源 claude

- KR：`체력이 30% 이하가 되었을 때, 얻는 [MRcAiBa] +1`
- 現行譯文：`自身殘存體力不高於30%時，獲得的 [MRcAiBa] 層數額外+1`
- 註記：原譯「透過技能效果獲得的」KR 沒有「技能效果」，依 KR 刪去
- 判讀：原譯「透過技能效果獲得的」KR 沒有「技能效果」，依 KR 刪去
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N514　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Passives_Abnormality-a1c9p2.json|/dataList/9/desc/5b4a9d6a`　enemy，批次 20261101T081633-4865b873，來源 claude

- KR：`날개 부위 파괴 시, 마지막 공격자에게 자신의 [MRcAiBa]만큼⏎[MRcAiBb]를 부여하고, 자신의 [MRcAiBa]를 소멸함⏎⏎‘홍염살’을 사용할 때, [MRcAiBa]가 0개면 ‘불씨[TabExplain] 지피기’를 추가로 사용함`
- 現行譯文：`當自身的翅膀部位遭到破壞時，對最後攻擊部位的敵方單位施加與自身帶有的 [MRcAiBa] 層數等量的⏎[MRcAiBb]；隨後解除自身帶有的所有 [MRcAiBa]⏎⏎自身使用「紅艷煞」時，若自身帶有的 [MRcAiBa] 層數為0，則額外使用「點燃火種[TabExplain]」`
- 註記：KR 第一行在「만큼」後換行，已依 KR 補上換行；[TabExplain] 只保留 KR 的一個
- 判讀：KR 第一行在「만큼」後換行，已依 KR 補上換行；[TabExplain] 只保留 KR 的一個
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N515　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Passives_Abnormality-a1c9p3.json|/dataList/79/desc/c13e31d5`　enemy，批次 20261101T081639-6dfc99e4，來源 claude

- KR：`턴 시작 시,⏎- 료슈를 제외한 무작위 수감자에게 [IndexPrescriptTargetToPersonality]을 부여하고, 우선 공격함⏎<color=#ff6000><mark=#ff000040><b><u>- [IndexPrescriptTargetToPersonality]에게 사용하는 스킬의 코인이 전부 [SuperCoin]으로 변경됨</u></b></mark></color>⏎· [IndexPrescriptTargetToPersonality]이 사망하거나 대상을 공격하지 못하게 될 경우,<color=#ff6000><mark=#ff000040><b><u>사용하지 않은 나머지 스킬을 료슈 제외 무작위 수감자 한 명에게 사용 (현재 체력이 가장 낮은 대상 우선, 체력 1인 경우 제외)</u></b></mark></color>⏎· 다른 모든 수감자 사망 시, 료슈에게 부여⏎- [BlessingOfIndexPrescriptEnemy]만큼 [Protection] 얻음 (최대 9)⏎<color=#ff6000><mark=#ff000040><b><u>· [KarmaOfIndexRien_2Phase] 20당, 1 감소</u></b></mark></color>⏎⏎[BlessingOfIndexPrescriptEnemy]당 [IndexPrescriptTargetToPersonality]에게 가하는 피해량 +5% (최대 40%)⏎- [BlessingOfIndexPrescriptEnemy]가 9면, 대신 모든 대상에게 가하는 피해량 +50%⏎⏎페이즈가 변경된 턴부터 효과 발동⏎- 두 턴마다 [BlackNightmare] 얻음.⏎- [BlackNightmare]을 얻을 때 [StackRienSpecialSkill]가 9 미만이면, 특수 패턴 발동`
- 現行譯文：`回合開始時：⏎- 對良秀除外的隨機1名罪人施加 [IndexPrescriptTargetToPersonality] 並且優先攻擊該罪人⏎<color=#ff6000><mark=#ff000040><b><u>- 自身對 [IndexPrescriptTargetToPersonality] 使用的技能硬幣將全數轉變為 [SuperCoin]</u></b></mark></color>⏎．若 [IndexPrescriptTargetToPersonality] 陣亡或自身無法攻擊目標時，<color=#ff6000><mark=#ff000040><b><u>隨機對除良秀以外的1名罪人使用剩餘的技能(優先指定現存體力最低的目標，不包括體力為1點的單位)</u></b></mark></color>⏎．其他罪人全部陣亡時，則對良秀施加⏎- 根據自身帶有的 [BlessingOfIndexPrescriptEnemy] 層數，使自身獲得等量的 [Protection] 層數 (最多9層)⏎<color=#ff6000><mark=#ff000040><b><u>．自身每帶有20層 [KarmaOfIndexRien_2Phase]，則使其減少1層</u></b></mark></color>⏎⏎自身每帶有1層 [BlessingOfIndexPrescriptEnemy]，則使自身對 [IndexPrescriptTargetToPersonality] 造成的傷害+5%(最多+40%)⏎- 若自身的 [BlessingOfIndexPrescriptEnemy] 層數為9層，則改為對所有目標造成的傷害+50%⏎⏎從轉換階段的回合起觸發以下效果⏎- 每經過2回合，使自身獲得 [BlackNightmare]⏎- 獲得 [BlackNightmare] 時，若自身帶有的 [StackRienSpecialSkill] 層數低於9層，則觸發特殊模式`
- 註記：KR「페이즈」寫「階段」、「패턴」寫「模式」；「9 미만」改為「低於9層」
- 判讀：KR「페이즈」寫「階段」、「패턴」寫「模式」；「9 미만」改為「低於9層」
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N516　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Passives_Abnormality-mr7.json|/dataList/14/desc/8a192b8b`　enemy，批次 20261101T081724-a2edc918，來源 claude

- KR：`턴 시작 시 흐트러짐 상태면 흐트러짐을 해제⏎⏎무작위 수감자 1명을 지정하여, [TeachersPrey]을 부여⏎- 이후 4턴마다 1번씩 발동⏎- 스킬 ‘처분’으로 대상을 처치할 시 다음 턴에 다시 발동`
- 現行譯文：`回合開始時，若自身陷入混亂，則解除自身混亂⏎⏎隨機指定1名罪人並對其施加 [TeachersPrey]⏎- 此後，每經過4回合觸發1次該效果⏎- 自身使用技能「處分」擊殺目標時，則下回合再次觸發該效果`
- 註記：draft 的 [TeachersPreyMirror] 與 KR 不符，改為 [TeachersPrey]
- 判讀：draft 的 [TeachersPreyMirror] 與 KR 不符，改為 [TeachersPrey]
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N517　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Passives_Abnormality-mr7.json|/dataList/31/desc/79230ed0`　enemy，批次 20261101T081724-a2edc918，來源 claude

- KR：`턴 시작 시,⏎- 무작위 수감자에게 [IndexPrescriptTargetToPersonality]을 부여하고, 우선 공격함⏎<color=#ff6000><mark=#ff000040><b><u>- [IndexPrescriptTargetToPersonality]에게 사용하는 스킬의 코인이 전부 [SuperCoin]으로 변경됨</u></b></mark></color>⏎- [BlessingOfIndexPrescriptEnemy]만큼 [Protection] 얻음 (최대 9)⏎<color=#ff6000><mark=#ff000040><b><u>· [KarmaOfIndexRien_2Phase] 20당, 1 감소</u></b></mark></color>⏎⏎공격 시 [IndexPrescriptTargetToPersonality]이 사망하거나 대상을 공격하지 못하게 될 경우,⏎<color=#ff6000><mark=#ff000040><b><u>사용하지 않은 나머지 스킬을 다른 무작위 수감자에게 사용⏎- 현재 체력이 가장 낮은 대상 우선 (체력 1인 경우 제외)⏎- 무작위로 선택된 수감자가 사망하거나 체력이 1이 될 경우, 다른 수감자에게 반복</u></b></mark></color>⏎⏎[BlessingOfIndexPrescriptEnemy]당 [IndexPrescriptTargetToPersonality]에게 가하는 피해량 +5% (최대 40%)⏎- [BlessingOfIndexPrescriptEnemy]가 9면, 대신 모든 대상에게 가하는 피해량 +50%⏎⏎페이즈가 변경된 턴부터 효과 발동⏎- 턴 시작 시 [BlackNightmare] 얻음⏎· 직전 턴에 강력한 스킬을 사용하였으면, 해당 턴에는 [BlackNightmare] 사라짐⏎- [BlackNightmare]을 얻을 때 [StackRienSpecialSkill]가 9 미만이면, 특수 패턴 발동`
- 現行譯文：`回合開始時，⏎- 隨機對1名罪人施加 [IndexPrescriptTargetToPersonality] 並優先攻擊該罪人⏎<color=#ff6000><mark=#ff000040><b><u>- 自身對 [IndexPrescriptTargetToPersonality] 使用的技能硬幣將全數轉變為 [SuperCoin]</u></b></mark></color>⏎- 根據自身帶有的 [BlessingOfIndexPrescriptEnemy] 層數，使自身獲得等量的 [Protection] 層數 (最多9層)⏎<color=#ff6000><mark=#ff000040><b><u>．自身每帶有20層 [KarmaOfIndexRien_2Phase] 時，則使獲得的層數減少1層</u></b></mark></color>⏎⏎攻擊時 [IndexPrescriptTargetToPersonality] 陣亡或自身無法攻擊目標時，⏎<color=#ff6000><mark=#ff000040><b><u>隨機對另1名罪人使用未使用的剩餘技能⏎- 優先指定現存體力最低的目標(不包括體力為1點的單位)⏎- 隨機指定的罪人陣亡或體力降至1點時，對其他罪人重複該效果</u></b></mark></color>⏎⏎自身每帶有1層 [BlessingOfIndexPrescriptEnemy]，則使自身對 [IndexPrescriptTargetToPersonality] 造成的傷害+5%(最多+40%)⏎- 若自身的 [BlessingOfIndexPrescriptEnemy] 層數為9層時，則改為對所有目標造成的傷害+50%⏎⏎從改變模式的回合起觸發效果⏎- 回合開始時，使自身獲得 [BlackNightmare]⏎．若上回合自身使用了強力技能，則本回合不會獲得 [BlackNightmare]⏎- 獲得 [BlackNightmare] 時，若自身帶有的 [StackRienSpecialSkill] 層數低於9層，則觸發特殊模式`
- 註記：draft 多一個 [Protection] 已拿掉；「9 미만」改為「低於9層」
- 判讀：draft 多一個 [Protection] 已拿掉；「9 미만」改為「低於9層」
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N518　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Passives_Abnormality-twth.json|/dataList/2/desc/5255ff35`　enemy，批次 20261101T081748-6eed33ea，來源 claude

- KR：`전투 종료 시 [BearClawWound], [EagleClawWound], [BoldBrushstrokes]를⏎전부 보유한 수감자에게 <color=#ff6000><mark=#ff000040><b><u>다음 턴에 [IntenseColors] 1 부여</u></b></mark></color>⏎- 대상이 보유한 [BearClawWound], [EagleClawWound]가⏎각각 3/6/9 이상이면, 대신 2/3/4 부여`
- 現行譯文：`戰鬥結束時，對同時帶有 [BearClawWound]、[EagleClawWound] 與 [BoldBrushstrokes] 的⏎罪人<color=#ff6000><mark=#ff000040><b><u>在下回合施加1層 [IntenseColors]</u></b></mark></color>⏎- 若目標帶有的 [BearClawWound] 與 [EagleClawWound]⏎各不低於3/6/9層，則改為施加2/3/4層`
- 註記：刪去草稿自 EN 加入、KR 沒有的「(以層數更低的效果為基準)」
- 判讀：刪去草稿自 EN 加入、KR 沒有的「(以層數更低的效果為基準)」
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N519　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Passives_Abnormality-twth.json|/dataList/5/desc/572c652e`　enemy，批次 20261101T081748-6eed33ea，來源 claude

- KR：`턴 시작 시, <color=#ff6000><mark=#ff000040><b><u>[Protection] 8 얻음</u></b></mark></color>⏎- 보유한 <color=#ff6000><mark=#ff000040><b><u>부정적인 효과 수만큼</u></b></mark></color> 턴 시작 시 획득하는 수치가 2 감소⏎- ‘당신의 피부 아래 맥동하는 붉은 색감을 거칠게.’ <color=#ff6000><mark=#ff000040><b><u>합 패배 시,</u></b></mark></color> 이 효과가 다음 턴에 한하여 <color=#ff6000><mark=#ff000040><b><u>[Vulnerable] 2 얻는 효과</u></b></mark></color>로 변경됨⏎⏎스킬 코인으로 <color=#ff6000><mark=#ff000040><b><u>피격받을 때마다 방어 레벨이 1씩 증가</u></b></mark></color> (턴당 최대 20)⏎- 턴 종료 시, 이 효과로 얻은 방어 레벨이 소멸됨⏎- ‘당신의 피부 아래 맥동하는 붉은 색감을 거칠게.’ <color=#ff6000><mark=#ff000040><b><u>합 패배 시,</u></b></mark></color> 이 효과가 다음 턴에 한하여 스킬 코인으로 피격받을 때마다 <color=#ff6000><mark=#ff000040><b><u>방어 레벨이 1씩 감소</u></b></mark></color>하는 효과로 변경됨 (최대 20, 해당 턴 종료 시 효과 소멸됨)`
- 現行譯文：`回合開始時，<color=#ff6000><mark=#ff000040><b><u>使自身獲得8層 [Protection]</u></b></mark></color>⏎- 自身每帶有<color=#ff6000><mark=#ff000040><b><u>1種負面效果</u></b></mark></color>，回合開始時獲得的層數減少2層⏎- 使用「將你膚下那脈動的紅色顏料揮灑開來吧。」<color=#ff6000><mark=#ff000040><b><u>拚點失敗時，</u></b></mark></color>該效果在下回合<color=#ff6000><mark=#ff000040><b><u>改為使自身獲得2層 [Vulnerable]</u></b></mark></color>⏎⏎<color=#ff6000><mark=#ff000040><b><u>每次受到技能硬幣攻擊時，使自身增加1級防禦等級</u></b></mark></color>(每回合最多20級)⏎- 回合結束時，解除該效果增加的防禦等級⏎- 使用「將你膚下那脈動的紅色顏料揮灑開來吧。」<color=#ff6000><mark=#ff000040><b><u>拚點失敗時，</u></b></mark></color>該效果在下回合改為每次受到技能硬幣攻擊時，<color=#ff6000><mark=#ff000040><b><u>使自身減少1級防禦等級</u></b></mark></color>(最多20級，回合結束時解除)`
- 註記：KR「[Vulnerable] 2 얻는 효과」是自身獲得，草稿寫成「施加」，已改正
- 判讀：KR「[Vulnerable] 2 얻는 효과」是自身獲得，草稿寫成「施加」，已改正
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N520　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Passives_Abnormality.json|/dataList/116/desc/80a6c1c2`　enemy，批次 20261101T091948-08250be2，來源 claude

- KR：`턴 종료 시 준비된 고기가 부여된 아군을 처치하지 못했으면 [Hunger] 1 증가. 체력이 75% 미만이면, 2 증가, 체력이 50% 미만이면, 3 증가.⏎턴 시작 시 [PreparedMeat]가 부여된 아군을 처치했으면, [Hunger] 1 감소.`
- 現行譯文：`回合結束時，若自身未能擊殺帶有儲備糧的友方單位，則對自身施加1層 [Hunger]。若現存體力低於75%，則施加2層。若現存體力低於50%，則施加3層。⏎回合開始時，若自身擊殺了帶有 [PreparedMeat] 的友方單位，則減少1層 [Hunger]。`
- 註記：KR 第1行寫韓文「준비된 고기」、第2行才是 [PreparedMeat]，原譯位置相反，依 KR 對調
- 判讀：KR 第1行寫韓文「준비된 고기」、第2行才是 [PreparedMeat]，原譯位置相反，依 KR 對調
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N521　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Passives_Abnormality.json|/dataList/30/desc/00fa89f5`　enemy，批次 20261101T091948-08250be2，來源 claude

- KR：`체력이 50% 미만이면 매 턴마다 광란 상태가 된다.`
- 現行譯文：`若體力低於50%，則每回合都進入狂暴狀態。`
- 註記：KR 寫韓文狀態名「광란」，依 base.md 寫官方名「狂暴」不加空格
- 判讀：KR 寫韓文狀態名「광란」，依 base.md 寫官方名「狂暴」不加空格
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N522　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Passives_Abnormality_Refraction2BokGak.json|/dataList/1/desc/7d574a1c`　enemy，批次 20261101T084201-db03f7d9，來源 claude

- KR：`[Sinking]을 얻을 때 위력과 횟수를 1 추가로 얻음 (순환[TabExplain] 버프는 해당하지 않음)`
- 現行譯文：`自身被施加 [Sinking] 強度或回數時，強度與回數各額外增加1 (不適用於巡迴[TabExplain]效果)`
- 註記：draft 的 [ID] 出現 3 次，KR 只有 1 次；改寫為「強度與回數各額外增加1」
- 判讀：draft 的 [ID] 出現 3 次，KR 只有 1 次；改寫為「強度與回數各額外增加1」
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N523　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Passives_Abnormality_Refraction5.json|/dataList/2/desc/1ca4affa`　enemy，批次 20261101T084206-0391cb67，來源 claude

- KR：`남은 체력이 전체 체력의 75% 이하일 때, 패턴 변경⏎패턴 변경 시 옛 G사 부장과 짙어진 패잔병을 소환하고, 현재 체력이 가장 낮은 대상에게 징표 부여.⏎- 자신의 속도 최솟값, 최댓값이 1이 됨.⏎- 보호막 체력 300을 얻음.`
- 現行譯文：`現存體力不高於75%時，改變攻擊模式⏎攻擊模式改變時，召喚舊G公司部長與變本加厲的殘兵，並對現存體力最低的目標施加標記。⏎- 自身的最小與最大速度值變為1。⏎- 使自身獲得300點護盾體力。`
- 註記：KR 寫韓文狀態名「징표」、沒有 [ID]：拿掉草稿的 [FellBulletMark]，改寫官方名「標記」
- 判讀：KR 寫韓文狀態名「징표」、沒有 [ID]：拿掉草稿的 [FellBulletMark]，改寫官方名「標記」
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N524　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Passives_Abnormality_Refraction5.json|/dataList/23/desc/a7cfa72c`　enemy，批次 20261101T084206-0391cb67，來源 claude

- KR：`전투 시작 시 자신에게 징표가 없으면, 보호막 300을 얻음.␣`
- 現行譯文：`戰鬥開始時，若自身未帶有標記，則使自身獲得300點護盾`
- 註記：KR 寫「징표」、沒有 [ID]：拿掉草稿的 [FellBulletMark]，改寫官方名「標記」
- 判讀：KR 寫「징표」、沒有 [ID]：拿掉草稿的 [FellBulletMark]，改寫官方名「標記」
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N525　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Passives_Abnormality_Refraction6.json|/dataList/38/desc/fce428ff`　enemy，批次 20261101T092612-f86dc9ae，來源 claude

- KR：`<color=#ff9000><mark=#ff000040><b><u>코인마다 무기가 무작위로 정해지며 특수 효과 적용</u></b></mark></color>⏎- [RienWeapon01Hatchet]⏎- [RienWeapon02Stiletto]⏎- [RienWeapon03Greatsword]⏎- [RienWeapon04Rapier]⏎- [RienWeapon05Sledgehammer]⏎- [RienWeapon06Ultragreatsword]⏎- [RienWeapon07Lance]⏎- [RienWeapon08Chain]⏎- [RienWeapon09Scythe]⏎⏎공격 스킬 사용 시, [StackRienSpecialSkill] 1 얻음 (최대 9)⏎- 스킬별 1회 획득 가능 (동일한 스킬로 2를 초과하여 얻을 수 없음)⏎⏎<color=#ff9000><mark=#ff000040><b><u>턴 시작 시 [StackRienSpecialSkill]가 9면, 강력한 스킬을 사용</u></b></mark></color>⏎- 첫 사용 후 공격 종료 시, [Shin_Rien] 얻음⏎- 스킬 종류별 [StackRienSpecialSkill] 획득 여부 초기화⏎- 속도 최솟값, 최댓값 +15⏎- 턴 종료 시 다음 턴에, [Vulnerable] 1, [ResultReduction] 2 얻음⏎⏎등장 이후 네 번째 턴 시작 시, [StackRienSpecialSkill]가 9 미만이면, 9까지 증가하고 (증가한 수치 x 3)만큼 [KarmaOfIndexRien] 얻음⏎- 이후 네 턴마다 이 효과 발동`
- 現行譯文：`<color=#ff9000><mark=#ff000040><b><u>自身每枚硬幣的武器均會隨機決定且帶有特殊效果</u></b></mark></color>⏎- [RienWeapon01Hatchet]⏎- [RienWeapon02Stiletto]⏎- [RienWeapon03Greatsword]⏎- [RienWeapon04Rapier]⏎- [RienWeapon05Sledgehammer]⏎- [RienWeapon06Ultragreatsword]⏎- [RienWeapon07Lance]⏎- [RienWeapon08Chain]⏎- [RienWeapon09Scythe]⏎⏎使用攻擊技能時，使自身獲得1層 [StackRienSpecialSkill] (最多9層)⏎- 每個技能僅可獲得1次 (無法以相同的技能獲得高於2層)⏎⏎<color=#ff9000><mark=#ff000040><b><u>回合開始時，若自身帶有的 [StackRienSpecialSkill] 層數為9層，則使用強力技能</u></b></mark></color>⏎- 首次使用強力技能後，攻擊結束時使自身獲得 [Shin_Rien]⏎- 重置各技能是否已獲得 [StackRienSpecialSkill] 的紀錄⏎- 自身最小與最大速度值+15⏎- 回合結束時，下回合使自身獲得1層 [Vulnerable] 與2層 [ResultReduction]⏎⏎自身登場後的第4回合開始時，若自身的 [StackRienSpecialSkill] 層數低於9層，則增加至9層並使自身獲得(增加的層數×3)層 [KarmaOfIndexRien]⏎- 此後，每4回合觸發一次該效果`
- 註記：prev_zh 的 <b><u> 是空標籤、文字在外，已改成包住整句；KR「동일한 스킬로 2를 초과하여 얻을 수 없음」依 KR 寫「無法以相同的技能獲得高於2層」（EN 是 twice）
- 判讀：prev_zh 的 <b><u> 是空標籤、文字在外，已改成包住整句；KR「동일한 스킬로 2를 초과하여 얻을 수 없
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N526　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Passives_Abnormality_Refraction6.json|/dataList/44/desc/fce428ff`　enemy，批次 20261101T092612-f86dc9ae，來源 claude

- KR：`<color=#ff9000><mark=#ff000040><b><u>코인마다 무기가 무작위로 정해지며 특수 효과 적용</u></b></mark></color>⏎- [RienWeapon01Hatchet2phase]⏎- [RienWeapon02Stiletto2phase]⏎- [RienWeapon03Greatsword2phase]⏎- [RienWeapon04Rapier2phase]⏎- [RienWeapon05Sledgehammer2phase]⏎- [RienWeapon06Ultragreatsword2phase]⏎- [RienWeapon07Lance2phase]⏎- [RienWeapon08Chain2phase]⏎- [RienWeapon09Scythe2phase]⏎⏎공격 스킬 사용 시, [StackRienSpecialSkill] 1 얻음 (최대 9)⏎- 스킬별 1회 획득 가능 (동일한 스킬로 2를 초과하여 얻을 수 없음)⏎- 특수 패턴에서만 사용하는 스킬로는 얻지 않음⏎⏎턴 시작 시 [StackRienSpecialSkill]가 9면, 강력한 스킬을 사용⏎- 스킬 종류별 [StackRienSpecialSkill] 획득 여부 초기화⏎- 속도 최솟값, 최댓값 +15⏎- 턴 종료 시 다음 턴에, [Vulnerable] 2, [ResultReduction] 3 얻음⏎⏎페이즈가 변경된 턴부터 효과 발동⏎- 세 턴마다 [StackRienSpecialSkill]가 9 미만이면,⏎9까지 증가하고 (증가한 수치 x 5)만큼 [KarmaOfIndexRien_2Phase] 얻음`
- 現行譯文：`<color=#ff9000><mark=#ff000040><b><u>自身每枚硬幣的武器均會隨機決定且帶有特殊效果</u></b></mark></color>⏎- [RienWeapon01Hatchet2phase]⏎- [RienWeapon02Stiletto2phase]⏎- [RienWeapon03Greatsword2phase]⏎- [RienWeapon04Rapier2phase]⏎- [RienWeapon05Sledgehammer2phase]⏎- [RienWeapon06Ultragreatsword2phase]⏎- [RienWeapon07Lance2phase]⏎- [RienWeapon08Chain2phase]⏎- [RienWeapon09Scythe2phase]⏎⏎使用攻擊技能時，使自身獲得1層 [StackRienSpecialSkill] (最多9層)⏎- 每個技能僅可獲得1次 (無法以相同的技能獲得高於2層)⏎- 僅在特殊模式使用的技能不會獲得⏎⏎回合開始時，若自身的 [StackRienSpecialSkill] 層數為9層，則使用強力技能⏎- 重置各技能是否已獲得 [StackRienSpecialSkill] 的紀錄⏎- 自身最小與最大速度值+15⏎- 回合結束時，下回合使自身獲得2層 [Vulnerable] 與3層 [ResultReduction]⏎⏎從階段變更的回合起觸發以下效果⏎- 每3回合，若自身的 [StackRienSpecialSkill] 層數低於9層，⏎則增加至9層並使自身獲得(增加的層數×5)層 [KarmaOfIndexRien_2Phase]`
- 註記：prev_zh 的 <b><u> 是空標籤、文字在外，已改成包住整句；KR「동일한 스킬로 2를 초과하여 얻을 수 없음」依 KR 寫「無法以相同的技能獲得高於2層」（EN 是 twice）
- 判讀：prev_zh 的 <b><u> 是空標籤、文字在外，已改成包住整句；KR「동일한 스킬로 2를 초과하여 얻을 수 없
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N527　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Passives_Enemy-a1c5p1.json|/dataList/11/desc/499533a3`　enemy，批次 20261101T081823-40d4eac8，來源 claude

- KR：`적중시 대상에게 <sprite name="Laceration"><color=#e30000><u><link="Laceration">출혈</color></link></u>이 있다면, 대상의 <sprite name="Laceration"><color=#e30000><u><link="Laceration">출혈</color></link></u>만큼 정신력 회복`
- 現行譯文：`攻擊命中時，若目標帶有<sprite name="Laceration"><color=#e30000><u><link="Laceration">流血</color></link></u>，則根據目標帶有的<sprite name="Laceration"><color=#e30000><u><link="Laceration">流血</color></link></u>強度，使自身恢復相應的理智值`
- 註記：補回 KR 的 sprite／link 標籤
- 判讀：補回 KR 的 sprite／link 標籤
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N528　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Passives_Enemy-a1c5p1.json|/dataList/2/desc/499533a3`　enemy，批次 20261101T081823-40d4eac8，來源 claude

- KR：`<sprite name="Breath"><color=#f8c200><u><link="Breath">호흡</color></link></u>이 있을 때, 합 승리 시 <sprite name="Breath"><color=#f8c200><u><link="Breath">호흡</color></link></u>을 5 얻음.`
- 現行譯文：`若自身帶有<sprite name="Breath"><color=#f8c200><u><link="Breath">呼吸法</color></link></u>，則拚點勝利時使自身增加5級<sprite name="Breath"><color=#f8c200><u><link="Breath">呼吸法</color></link></u>強度。`
- 註記：補回 KR 的 sprite／link 標籤
- 判讀：補回 KR 的 sprite／link 標籤
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N529　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Passives_Enemy-a1c5p1.json|/dataList/4/desc/499533a3`　enemy，批次 20261101T081823-40d4eac8，來源 claude

- KR：`적중시 대상에게 <sprite name="Laceration"><color=#e30000><u><link="Laceration">출혈</color></link></u>이 있다면, 대상의 <sprite name="Laceration"><color=#e30000><u><link="Laceration">출혈</color></link></u>만큼 정신력 회복`
- 現行譯文：`攻擊命中時，若目標帶有<sprite name="Laceration"><color=#e30000><u><link="Laceration">流血</color></link></u>，則根據目標帶有的<sprite name="Laceration"><color=#e30000><u><link="Laceration">流血</color></link></u>強度，使自身恢復相應的理智值`
- 註記：補回 KR 的 sprite／link 標籤
- 判讀：補回 KR 的 sprite／link 標籤
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N530　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Passives_Enemy-a1c5p1.json|/dataList/5/desc/178930b5`　enemy，批次 20261101T081823-40d4eac8，來源 claude

- KR：`속도가 1일 때, <sprite name="Protection"><color=#f8c200><u><link="Protection">보호</color></link></u>를 얻으면 1 추가로 얻음.`
- 現行譯文：`若本單位速度值為1，則獲得<sprite name="Protection"><color=#f8c200><u><link="Protection">守護</color></link></u>時，額外獲得1層。`
- 註記：補回 KR 的 sprite／link 標籤；draft 漏了「獲得守護時」的條件，已補
- 判讀：補回 KR 的 sprite／link 標籤；draft 漏了「獲得守護時」的條件，已補
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N531　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Passives_Enemy-a1c5p1.json|/dataList/6/desc/499533a3`　enemy，批次 20261101T081823-40d4eac8，來源 claude

- KR：`적중시 대상에게 <sprite name="Laceration"><color=#e30000><u><link="Laceration">출혈</color></link></u>이 있다면, 대상의 <sprite name="Laceration"><color=#e30000><u><link="Laceration">출혈</color></link></u>만큼 정신력 회복`
- 現行譯文：`攻擊命中時，若目標帶有<sprite name="Laceration"><color=#e30000><u><link="Laceration">流血</color></link></u>，則根據目標帶有的<sprite name="Laceration"><color=#e30000><u><link="Laceration">流血</color></link></u>強度，使自身恢復相應的理智值`
- 註記：補回 KR 的 sprite／link 標籤
- 判讀：補回 KR 的 sprite／link 標籤
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N532　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Passives_Enemy.json|/dataList/20/desc/3a22043f`　enemy，批次 20261101T081824-b0e9076a，來源 claude

- KR：`<sprite name="Laceration"><color=#f8c200><u><link="Laceration">출혈</color></link></u>과 <sprite name="Combustion"><color=#f8c200><u><link="Combustion">화상</color></link></u>에 입는 피해 반감.`
- 現行譯文：`受到的<sprite name="Laceration"><color=#f8c200><u><link="Laceration">流血</color></link></u>及<sprite name="Combustion"><color=#f8c200><u><link="Combustion">燒傷</color></link></u>傷害減半。`
- 註記：補回 draft 漏掉的標籤；KR 的標籤本身交錯（</color></link></u>），照 KR 原樣保留
- 判讀：補回 draft 漏掉的標籤；KR 的標籤本身交錯（</color></link></u>），照 KR 原樣保留
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N533　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills.json|/dataList/113/levelList/0/coinlist/1/coindescs/0/desc/f080d7d1`　enemy，批次 20261101T104945-9daaee2d，來源 claude

- KR：`[OnSucceedAttack] 수비 위력 감소 2 부여`
- 現行譯文：`[OnSucceedAttack] 對目標施加2層破綻`
- 註記：草稿誤寫成 [OnSucceedAttackHead]，依 KR 改為 [OnSucceedAttack]
- 判讀：草稿誤寫成 [OnSucceedAttackHead]，依 KR 改為 [OnSucceedAttack]
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N534　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Abnormality-BossRaid.json|/dataList/33/levelList/0/desc/71b68a06`　enemy，批次 20261101T081828-3b1b6bba，來源 claude

- KR：`[Prey_leiheng]을 보유한 대상을 우선으로 지정⏎<color=#ff6000><mark=#ff000040><b><u>속도에 상관없이 이 스킬과 합을 할 수 있으며, 이 스킬을 사용하기 전까지 흐트러짐 상태가 되지 않음</u></b></mark></color>⏎[WhenUse] [BattleSense]을 전부 소모. 소모한 수치가⏎- 10 이상이면, 10당 공격 가중치 +1 (최대 2)⏎- 25일 경우, 메인 타겟의 [Combustion]과 [Vibration]의 합 4당 최종 위력 +1 (최대 5)⏎[DefeatDuel] [Paralysis] 1 얻음⏎[DefeatDuel] 피해량이 80%만큼 감소하고 이 스킬의 피해로 대상이 흐트러지지 않으며, <color=#ff6000><mark=#ff000040><b><u>[Combustion] 위력과 횟수, [Vibration] 위력과 횟수 부여량이 절반으로 감소</u></b></mark></color> (다른 효과로 증가되는 수치 제외)⏎[EndSkill] [FullReload] (턴당 1회)⏎- [BulletPropellantSpecial]을 보유하였다면, 대신 [ReloadKeepAmmo] (턴당 1회)⏎[EndSkill] 이번 턴 종료`
- 現行譯文：`優先指定帶有 [Prey_leiheng] 的目標⏎<color=#ff6000><mark=#ff000040><b><u>可無視速度值與本技能拚點。直到使用本技能前，自身不會陷入混亂</u></b></mark></color>⏎[WhenUse] 消耗所有 [BattleSense]，若消耗的層數：⏎- 不低於10層，則每消耗10層，使本技能的攻擊容量+1(最多+2)⏎- 為25層，則主要目標的 [Combustion] 與 [Vibration] 強度之和每有4級，使本技能的最終威力+1(最多+5)⏎[DefeatDuel] 對自身施加1層 [Paralysis]⏎[DefeatDuel] 使本技能造成的傷害-80%，且本技能造成的傷害不會使目標陷入混亂，<color=#ff6000><mark=#ff000040><b><u>令本技能施加的 [Combustion] 強度與回數、[Vibration] 強度與回數減半</u></b></mark></color>(不包括其他效果增加的施加量)⏎[EndSkill] 使自身 [FullReload] (每回合最多1次)⏎- 若自身帶有 [BulletPropellantSpecial]，則改為使自身 [ReloadKeepAmmo] (每回合最多1次)⏎[EndSkill] 結束本回合`
- 註記：draft 是另一個等級（34）的內容：最多+4／+6、多了 KR 沒有的「50層」一行、「25일 경우」寫成不低於；已依 KR 改為最多+2／+5、刪掉 50 層一行
- 判讀：draft 是另一個等級（34）的內容：最多+4／+6、多了 KR 沒有的「50層」一行、「25일 경우」寫成不低於；已
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N535　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Abnormality-a1c10p1.json|/dataList/2/levelList/0/coinlist/0/coindescs/1/desc/481efa5d`　enemy，批次 20261101T081836-aae3f51f，來源 claude

- KR：`[OnSucceedAttack] [VibrationExplosion]. 진동 횟수 1 감소`
- 現行譯文：`[OnSucceedAttack] 使目標 [VibrationExplosion]，並使其震顫回數減少1回`
- 註記：KR 寫韓文「진동」處 draft 多寫了 [Vibration]，改中文名；其他單位同理刪去 KR 用「수치」省略的 [ID]，「미만」改「低於」
- 判讀：KR 寫韓文「진동」處 draft 多寫了 [Vibration]，改中文名；其他單位同理刪去 KR 用「수치」省略的 
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N536　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Abnormality-a1c10p2.json|/dataList/15/levelList/0/coinlist/0/coindescs/1/desc/28f3655d`　enemy，批次 20261101T081843-841e37ee，來源 claude

- KR：`[UnBrokenCoinOnSucceedAttack] 침잠 10 부여`
- 現行譯文：`[UnBrokenCoinOnSucceedAttack] 使目標增加10級沉淪強度`
- 註記：KR 用韓文「침잠」、沒有 [ID]，依 base.md 寫官方名「沉淪」
- 判讀：KR 用韓文「침잠」、沒有 [ID]，依 base.md 寫官方名「沉淪」
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N537　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Abnormality-a1c10p2.json|/dataList/44/levelList/0/desc/ffd16dc6`　enemy，批次 20261101T081843-841e37ee，來源 claude

- KR：`아군에게 사용. 아군에게 피해량 0으로 고정⏎[WhenUse] 대상의 [Vibration] 4당, 최종 위력 +2 (최대 6)⏎[WhenUse] 자신의 [ChargeNoir] 위력이 3 이상이면, 모든 코인이 [SuperCoin]으로 변경됨⏎[WhenUse] 자신의 [ChargeNoir] 횟수가 10 미만이면, [ChargeNoir] 횟수를 10이 되도록 얻고, 얻은 수치 2당 [Vulnerable] 1 얻음⏎[DefeatDuel] 자신 [ChargeNoir] 횟수 절반 소모⏎[EndSkill] 자신의 [ChargeNoir] 횟수 모두 소모`
- 現行譯文：`本技能只會對友方單位使用，且對友方單位造成的傷害固定為0點⏎[WhenUse] 目標每帶有4級 [Vibration] 強度，則使本技能的最終威力+2(最多+6)⏎[WhenUse] 若自身的 [ChargeNoir] 強度不低於3級，則使本技能所有的硬幣轉變為 [SuperCoin]⏎[WhenUse] 若自身的 [ChargeNoir] 回數低於10回，則使自身的 [ChargeNoir] 增加至10回，且每增加2回，對自身施加1層 [Vulnerable]⏎[DefeatDuel] 消耗自身一半 [ChargeNoir] 回數⏎[EndSkill] 消耗自身所有 [ChargeNoir] 回數`
- 註記：KR「10 미만」改為「低於10回」（草稿誤為「不高於」）
- 判讀：KR「10 미만」改為「低於10回」（草稿誤為「不高於」）
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N538　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Abnormality-a1c10p3.json|/dataList/1/levelList/0/desc/0ad7d705`　enemy，批次 20261101T081851-1233f1a9，來源 claude

- KR：`[WhenUse]⏎- 전장의 <color=#ff6000><mark=#ff000040><b><u>형벌 촛대 1당, 최종 위력 +2 (최대 8)</u></b></mark></color>⏎- 자신의 [ChargeNoir] 횟수가 15 이상이면, 5 소모하여 최종 위력 +2⏎[WhenUse] 자신의 [ChargeNoir] 횟수 5 증가⏎[DefeatDuel] 이 스킬의 피해로 대상이 흐트러지지 않음`
- 現行譯文：`[WhenUse]⏎- 戰場上<color=#ff6000><mark=#ff000040><b><u>每有1支刑罰的燭台存活，則使本技能的最終威力+2(最多+8)</u></b></mark></color>⏎- 若自身的 [ChargeNoir] 回數不低於15回，則消耗5回並使本技能的最終威力+2⏎[WhenUse] 使自身獲得5回 [ChargeNoir]⏎[DefeatDuel] 本技能造成的傷害不會使目標陷入混亂`
- 註記：刪去多寫的 [ChargeNoir]（本批多處同樣處理）
- 判讀：刪去多寫的 [ChargeNoir]（本批多處同樣處理）
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N539　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Abnormality-a1c10p3.json|/dataList/40/levelList/0/desc/d9ad4810`　enemy，批次 20261101T081851-1233f1a9，來源 claude

- KR：`[WhenUse] 자신의 특수 [Charge] (위력 + 횟수) 합 4당 최종 위력 +1 (최대 3)⏎[WhenUse] 모든 대상의 부정적인 효과 종류 합 2개당 피해량 +10% (최대 50%, 대상마다 중복되는 효과는 별개로 취급)⏎<color=#ff6000><mark color=#ff000040><b><u>[WinDuel] 다음 턴에 [JeanneCheat] 1 부여</color></mark></b></u>⏎[DefeatDuel] 피해량 -80%, 이 스킬의 피해로 대상이 흐트러지지 않음`
- 現行譯文：`[WhenUse] 自身的特殊 [Charge] (強度+回數)之和每有4點，則使本技能的最終威力+1(最多+3)⏎[WhenUse] 所有目標帶有的負面效果種類之和每有2種，則使本技能造成的傷害+10%(最多+50%，不同目標上重複的效果分別計算)⏎<color=#ff6000><mark color=#ff000040><b><u>[WinDuel] 下回合對目標施加1層 [JeanneCheat]</color></mark></b></u>⏎[DefeatDuel] 使本技能造成的傷害-80%，且本技能造成的傷害不會使目標陷入混亂`
- 註記：標籤順序照 KR 原樣（KR 本身交錯）
- 判讀：標籤順序照 KR 原樣（KR 本身交錯）
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N540　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Abnormality-a1c10p3.json|/dataList/47/levelList/0/desc/21f0b898`　enemy，批次 20261101T081851-1233f1a9，來源 claude

- KR：`[WhenUse] 자신의 [ChargeNoir] 횟수 5 증가⏎- 대상에게 [Vibration]이 있으면, 추가로 5 증가⏎[WhenUse] 자신의 [ChargeNoir] 횟수 2당 최종 위력 +1 (최대 2)⏎[WhenUse] 자신의 [ChargeNoir] 횟수가 10 이상이면, 코인 적중시마다 [Vibration] 위력 5 부여`
- 現行譯文：`[WhenUse] 使自身獲得5回 [ChargeNoir]⏎- 若目標帶有 [Vibration]，則額外增加5回⏎[WhenUse] 自身每帶有2回 [ChargeNoir]，則使本技能的最終威力+1(最多+2)⏎[WhenUse] 若自身的 [ChargeNoir] 回數不低於10回，則每枚硬幣命中時，使目標增加5級 [Vibration] 強度`
- 註記：KR「코인 적중시마다」是每枚硬幣命中時，draft 誤為「本硬幣」
- 判讀：KR「코인 적중시마다」是每枚硬幣命中時，draft 誤為「本硬幣」
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N541　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Abnormality-a1c10p3.json|/dataList/8/levelList/0/desc/f9477398`　enemy，批次 20261101T081851-1233f1a9，來源 claude

- KR：`<color=#ff6000><mark=#ff000040><b><u>[GoldenFabricWarderSecondPhase]를 보유한 수감자</u></b></mark></color>가 명령 페이즈 시점에 이 스킬이 장착된 슬롯을 메인 타겟하여 공격 시, 해당 스킬의 피해량 +100% (E.G.O 스킬이면, 대신 피해량 +200%)⏎[EndBattle] [EncompassingVoiceSecondPhase]가 100 미만이면, 100까지 증가⏎- 증가한 수치만큼 간수의 정신력 감소 (최대 45)`
- 現行譯文：`<color=#ff6000><mark=#ff000040><b><u>若帶有 [GoldenFabricWarderSecondPhase] 的罪人</u></b></mark></color>在指揮階段指定裝備本技能的行動槽為主要目標進行攻擊，則使該罪人技能造成的傷害+100%(若為E.G.O技能，則改為+200%)⏎[EndBattle] 若自身的 [EncompassingVoiceSecondPhase] 低於100層，則增加至100層⏎- 使獄卒失去等同於增加層數的理智值(最多45點)`
- 註記：KR「해당 스킬의 피해량」是罪人那個技能的傷害，draft 誤為「本技能」；「100 미만」改為「低於100層」
- 判讀：KR「해당 스킬의 피해량」是罪人那個技能的傷害，draft 誤為「本技能」；「100 미만」改為「低於100層」
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N542　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Abnormality-a1c5p3.json|/dataList/1/levelList/0/coinlist/0/coindescs/1/desc/f2056c48`　enemy，批次 20261101T081854-c56c4b51，來源 claude

- KR：`[OnSucceedAttack] 대상에게 [Combustion]이 있다면, 화상 4 부여`
- 現行譯文：`[OnSucceedAttack] 若目標帶有 [Combustion]，則使其增加4級燒傷強度`
- 註記：KR 第二處寫韓文「화상」、無 [ID]，依 base.md 寫官方名「燒傷」
- 判讀：KR 第二處寫韓文「화상」、無 [ID]，依 base.md 寫官方名「燒傷」
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N543　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Abnormality-a1c7p2.json|/dataList/3/levelList/0/coinlist/0/coindescs/0/desc/26f022e4`　enemy，批次 20261101T081903-0d5304e9，來源 claude

- KR：`[OnSucceedAttack] 대상의 [FestivalFever]에 비례하여 다음 턴까지 [AttackUp] 1, [DefenseUp] 1 부여 ([FestivalFever] 3당 1 부여, 각각 최대 3)`
- 現行譯文：`[OnSucceedAttack] 依目標的 [FestivalFever] 層數，對其施加持續至下回合的 [AttackUp] 與 [DefenseUp] 各1層 (每3層 [FestivalFever] 施加1層，各自最多3層)`
- 註記：draft 漏了「다음 턴까지」並把「각각 최대 3」寫成「每個目標最多3次」，已依 KR 改正
- 判讀：draft 漏了「다음 턴까지」並把「각각 최대 3」寫成「每個目標最多3次」，已依 KR 改正
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N544　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Abnormality-a1c7p3.json|/dataList/30/levelList/0/desc/55cd5c26`　enemy，批次 20261101T081914-ccbeac12，來源 claude

- KR：`[Laceration]이 가장 많은 대상 우선 공격⏎[BeforeAttack] [Laceration]이 가장 많은 대상이면, 공격 가중치 +2, 합 위력 +3⏎[WhenUse] [BloodDinner]을 최대 30 소모하여, 소모한 [BloodDinner] 15당 [BloodArmor] 1 얻음. 코인 위력 +1 (최대 2)⏎- [BloodDinner]을 소모하지 못했으면, [Laceration] 10 얻고 발동시킴. [Laceration] 횟수 1 감소⏎[EndSkill] 이 스킬로 입히는 피해량만큼 [BloodDinner] 증가`
- 現行譯文：`優先指定 [Laceration] 最多的目標⏎[BeforeAttack] 若目標為 [Laceration] 最多的目標，則使本技能的攻擊容量+2，拚點威力+3⏎[WhenUse] 消耗最多30層 [BloodDinner]，每消耗15層 [BloodDinner]，使自身獲得1層 [BloodArmor] 並使本技能的硬幣威力+1(最多+2)⏎- 若未消耗 [BloodDinner]，則使自身增加10級 [Laceration] 強度並立即觸發，使自身的 [Laceration] 回數減少1回⏎[EndSkill] 獲得層數等同於本技能傷害量的 [BloodDinner]`
- 註記：草稿最後一行寫「減少2回」，KR「횟수 1 감소」，已改為1回
- 判讀：草稿最後一行寫「減少2回」，KR「횟수 1 감소」，已改為1回
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N545　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Abnormality-a1c8p3.json|/dataList/41/levelList/0/coinlist/0/coindescs/1/desc/d1d165d4`　enemy，批次 20261101T081914-92a653f6，來源 claude

- KR：`[OnSucceedAttack] 대상이 아군이면 남은 체력을 흡수하고, 대상의 [ImperfectEternalLife]를 흡수하여 다음 턴에 모든 부위가 흡수한 수치만큼 해당 효과를 얻음. 이후 흡수한 대상을 제거함`
- 現行譯文：`[OnSucceedAttack] 若目標為友方單位，則吸收其剩餘體力與其帶有的 [ImperfectEternalLife]，下回合使自身所有部位獲得等同於吸收層數的該效果；隨後移除被吸收的目標`
- 註記：draft 多一個 [ImperfectEternalLife]（KR 只有一個）已改寫；補上「所有部位」
- 判讀：draft 多一個 [ImperfectEternalLife]（KR 只有一個）已改寫；補上「所有部位」
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N546　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Abnormality-a1c8p3.json|/dataList/5/levelList/0/desc/72db597c`　enemy，批次 20261101T081914-92a653f6，來源 claude

- KR：`[Prey]을 보유한 대상을 우선으로 지정⏎속도에 상관없이 이 스킬과 합을 할 수 있으며, 이 스킬을 사용하기 전까지 흐트러짐 상태가 되지 않음⏎[WhenUse] [BattleSense]을 전부 소모. 소모한 수치가⏎- 10 이상이면, 10당 공격 가중치 +1 (최대 2)⏎- 25일 경우, 메인 타겟의 [Combustion]과 [Vibration]의 합 4당 최종 위력 +1 (최대 5)⏎[DefeatDuel] [ResultReduction] 4 얻음⏎[DefeatDuel] [Paralysis] 1 얻음⏎[DefeatDuel] 피해량이 80%만큼 감소하고 이 스킬의 피해로 대상이 흐트러지지 않으며, [Combustion] 위력과 횟수, [Vibration] 위력과 횟수 부여량이 절반으로 감소 (다른 효과로 증가되는 수치 제외)⏎[EndSkill] [FullReload] (턴당 1회)⏎- [BulletPropellantSpecial]을 보유하였다면, 대신 [ReloadKeepAmmo] (턴당 1회)⏎[EndSkill] 이번 턴 종료`
- 現行譯文：`優先指定帶有 [Prey] 的目標⏎可無視速度值與本技能拚點。直到使用本技能前，自身不會陷入混亂⏎[WhenUse] 消耗所有 [BattleSense]，若消耗的層數：⏎- 不低於10層，則每消耗10層，使本技能的攻擊容量+1(最多+2)⏎- 為25層，則主要目標的 [Combustion] 與 [Vibration] 強度之和每有4級，使本技能的最終威力+1(最多+5)⏎[DefeatDuel] 對自身施加4層 [ResultReduction]⏎[DefeatDuel] 對自身施加1層 [Paralysis]⏎[DefeatDuel] 使本技能造成的傷害-80%，且本技能造成的傷害不會使目標陷入混亂，令本技能施加的 [Combustion] 強度與回數、[Vibration] 強度與回數減半(不包括其他效果增加的施加量)⏎[EndSkill] 使自身 [FullReload] (每回合最多1次)⏎- 若自身帶有 [BulletPropellantSpecial]，則改為使自身 [ReloadKeepAmmo] (每回合最多1次)⏎[EndSkill] 結束本回合`
- 註記：draft 多出的 [BattleSense]、[Combustion]、[Vibration] 已拿掉，使數量與 KR 相同
- 判讀：draft 多出的 [BattleSense]、[Combustion]、[Vibration] 已拿掉，使數量與 KR
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N547　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Abnormality-a1c8p3.json|/dataList/6/levelList/0/desc/72db597c`　enemy，批次 20261101T081914-92a653f6，來源 claude

- KR：`[Prey]을 보유한 대상을 우선으로 지정⏎속도에 상관없이 이 스킬과 합을 할 수 있으며, 이 스킬을 사용하기 전까지 흐트러짐 상태가 되지 않음⏎[WhenUse] [BattleSense]을 전부 소모. 소모한 수치가⏎- 10 이상이면, 10당 공격 가중치 +1 (최대 4)⏎- 25 이상이면, 메인 타겟의 [Combustion]과 [Vibration]의 합 4당 최종 위력 +1 (최대 6)⏎- 50일 경우, 코인 위력 +1, 피해량 +50%⏎[DefeatDuel] [ResultReduction] 2 얻음⏎[DefeatDuel] [Paralysis] 6 얻음⏎[DefeatDuel] 피해량이 80%만큼 감소하고 이 스킬의 피해로 대상이 흐트러지지 않으며, [Combustion] 위력과 횟수, [Vibration] 위력과 횟수 부여량이 절반으로 감소 (다른 효과로 증가되는 수치 제외)⏎[EndSkill] [FullReload] (턴당 1회)⏎- [BulletPropellantSpecial]을 보유하였다면, 대신 [ReloadKeepAmmo] (턴당 1회)⏎[EndSkill] 이번 턴 종료`
- 現行譯文：`優先指定帶有 [Prey] 的目標⏎可無視速度值與本技能拚點。直到使用本技能前，自身不會陷入混亂⏎[WhenUse] 消耗所有 [BattleSense]，若消耗的層數：⏎- 不低於10層，則每消耗10層，使本技能的攻擊容量+1(最多+4)⏎- 不低於25層，則主要目標的 [Combustion] 與 [Vibration] 強度之和每有4級，使本技能的最終威力+1(最多+6)⏎- 為50層，則使本技能的硬幣威力+1、造成的傷害+50%⏎[DefeatDuel] 對自身施加2層 [ResultReduction]⏎[DefeatDuel] 對自身施加6層 [Paralysis]⏎[DefeatDuel] 使本技能造成的傷害-80%，且本技能造成的傷害不會使目標陷入混亂，令本技能施加的 [Combustion] 強度與回數、[Vibration] 強度與回數減半(不包括其他效果增加的施加量)⏎[EndSkill] 使自身 [FullReload] (每回合最多1次)⏎- 若自身帶有 [BulletPropellantSpecial]，則改為使自身 [ReloadKeepAmmo] (每回合最多1次)⏎[EndSkill] 結束本回合`
- 註記：draft 多出的 [BattleSense]、[Combustion]、[Vibration] 已拿掉，使數量與 KR 相同
- 判讀：draft 多出的 [BattleSense]、[Combustion]、[Vibration] 已拿掉，使數量與 KR
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N548　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Abnormality-a1c971.json|/dataList/12/levelList/0/desc/4167a64b`　enemy，批次 20261101T081919-eea50ec3，來源 claude

- KR：`[Laceration]이 가장 많은 대상 우선 공격⏎[BeforeAttack] 자신의 [BloodArmor] 10 당 공격 가중치 +1 (최대 3)⏎[WhenUse] 대상의 [Laceration] 5 당, 합 위력 +1 (최대 3)⏎[WhenUse] [BloodDinner]을 최대 30 소모하여, 소모한 [BloodDinner] 15당 [BloodArmor] 1 얻음. 코인 위력 +1 (최대 2)⏎- [BloodDinner]을 소모하지 못했으면, [Laceration] 10 얻고 발동시킴. [Laceration] 횟수 1 감소⏎[EndSkill] 이 스킬로 입히는 피해량만큼 [BloodDinner] 증가`
- 現行譯文：`優先指定 [Laceration] 最多的目標⏎[BeforeAttack] 自身每帶有10層 [BloodArmor]，使本技能的攻擊容量+1(最多+3)⏎[WhenUse] 目標每帶有5級 [Laceration] 強度，則使本技能的拚點威力+1(最多+3)⏎[WhenUse] 消耗最多30層 [BloodDinner]，每消耗15層 [BloodDinner]，使自身獲得1層 [BloodArmor] 並使本技能的硬幣威力+1(最多+2)⏎- 若未消耗 [BloodDinner]，則使自身增加10級 [Laceration] 強度並立即觸發，並使自身的 [Laceration] 回數減少1回⏎[EndSkill] 獲得層數等同於本技能傷害量的 [BloodDinner]`
- 註記：草稿把「[Laceration] 횟수 1 감소」寫成「減少2回」，已依 KR 改為1回
- 判讀：草稿把「[Laceration] 횟수 1 감소」寫成「減少2回」，已依 KR 改為1回
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N549　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Abnormality-a1c9p1.json|/dataList/21/levelList/0/desc/5bee4d85`　enemy，批次 20261101T081938-a74142f2，來源 claude

- KR：`이 스킬을 장착한 턴에 아래의 효과가 적용됨⏎- 이 스킬 사용 전까지 <color=#ff6000><mark=#ff000040><b><u>공격 스킬로 피격 시</u></b></mark></color>, [UnsteadyTheRings] 1 얻음 (적 스킬당 1회, E.G.O 스킬 포함)⏎- ‘아이언메이든[TabExplain]- 가시’ 효과가 발동되지 않음⏎- 이 스킬 사용 전까지 흐트러짐 상태가 되지 않음⏎⏎이 스킬로 대상 처치 시, 정신력 15 회복⏎[WhenUse] 대상의 [Laceration] 4당 코인 위력 +1 (최대 4)⏎[WhenUse] 대상의 [Laceration] 2당 피해량 +10% (최대 100%)⏎[DefeatDuel] 이 스킬의 피해로 대상이 흐트러지지 않음⏎[EndSkill] [BoneBladeTheRings] 소멸`
- 現行譯文：`裝備本技能的回合，觸發以下效果⏎- 使用本技能前，<color=#ff6000><mark=#ff000040><b><u>受到攻擊技能命中時</u></b></mark></color>，使自身獲得1層 [UnsteadyTheRings] (敵方每個技能最多1次；包含E.G.O技能)⏎- 「鐵處女[TabExplain] - 棘」效果不會觸發⏎- 使用本技能前，自身不會陷入混亂⏎⏎本技能擊殺目標時，使自身恢復15點理智值⏎[WhenUse] 目標每帶有4級 [Laceration] 強度，則使本技能的硬幣威力+1(最多+4)⏎[WhenUse] 目標每帶有2級 [Laceration] 強度，則使本技能造成的傷害+10%(最多+100%)⏎[DefeatDuel] 本技能造成的傷害無法使目標陷入混亂⏎[EndSkill] 消除自身帶有的 [BoneBladeTheRings]`
- 註記：draft 多出的 [TabExplain] 刪去（KR 只在「아이언메이든[TabExplain]- 가시」用一次）；KR 沒有「被動技能」「並準備使用」，依 KR 修正；補上缺的右括號
- 判讀：draft 多出的 [TabExplain] 刪去（KR 只在「아이언메이든[TabExplain]- 가시」用一次）；
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N550　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Abnormality-a1c9p1.json|/dataList/23/levelList/0/desc/f4b77772`　enemy，批次 20261101T081938-a74142f2，來源 claude

- KR：`이번 턴에 ‘아이언메이든[TabExplain]- 가시’ 효과로 <color=#ff6000><mark=#ff000040><b><u>적에게 주는 피해량 20배 증가</u></b></mark></color> (이 효과는 상쇄로 인하여 해제되지 않음)⏎[StartBattle] 공격 스킬로 피격시, 공격자의 [Laceration] 횟수 1 증가 (E.G.O 스킬 포함)`
- 現行譯文：`本回合中，「鐵處女[TabExplain] - 棘」效果<color=#ff6000><mark=#ff000040><b><u>對敵方造成的傷害增加至20倍</u></b></mark></color>(本效果不會因偏轉而解除)⏎[StartBattle] 受到攻擊技能命中時，使攻擊者增加1回 [Laceration] (包含E.G.O技能)`
- 註記：補上 KR 的「이번 턴에」；상쇄 依 BattleUIText 寫「偏轉」
- 判讀：補上 KR 的「이번 턴에」；상쇄 依 BattleUIText 寫「偏轉」
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N551　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Abnormality-a1c9p3.json|/dataList/33/levelList/0/desc/d6aeb1dd`　enemy，批次 20261101T092200-58f5979e，來源 claude

- KR：`[WhenUse] 자신에게 [ReinforcedTattooSpider]이 있으면, 코인 적중시 추가 타격 피해 10⏎[BeforeAttack] 자신의 체력이 50% 이하일 때, 피해량 +40%`
- 現行譯文：`[WhenUse] 若自身帶有 [ReinforcedTattooSpider]，則使本技能的硬幣命中時，額外造成10點打擊傷害⏎[BeforeAttack] 若自身的現存體力不高於上限的50%，則使本技能造成的傷害+40%`
- 註記：prev_zh 把 KR「추가 타격 피해 10」寫成5點，已改為10點
- 判讀：prev_zh 把 KR「추가 타격 피해 10」寫成5點，已改為10點
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N552　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Abnormality-a1c9p3.json|/dataList/34/levelList/0/desc/8f476c85`　enemy，批次 20261101T092200-58f5979e，來源 claude

- KR：`피격 시 <color=#ff6000><mark=#ff000040><b><u>자신의 [ReinforcedTattooSpider] 1당 이 스킬 1회 재사용</u></b></mark></color> (최대 1회)⏎[WhenUse] 자신에게 [ReinforcedTattooSpider]이 있으면, 코인 적중시 대상의 [Laceration] 횟수 1 증가⏎[BeforeAttack] 자신의 체력이 50% 이하일 때, 피해량 +40%`
- 現行譯文：`受到攻擊時，<color=#ff6000><mark=#ff000040><b><u>自身每帶有1層 [ReinforcedTattooSpider] 將重複使用本技能1次</u></b></mark></color>(最多1次)⏎[WhenUse] 若自身帶有 [ReinforcedTattooSpider]，則本技能的硬幣命中時，使目標增加1回 [Laceration]⏎[BeforeAttack] 若自身的現存體力不高於上限的50%，則使本技能造成的傷害+40%`
- 註記：prev_zh 漏了 KR「(최대 1회)」，已補上(最多1次)
- 判讀：prev_zh 漏了 KR「(최대 1회)」，已補上(最多1次)
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N553　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Abnormality-a1c9p3.json|/dataList/46/levelList/0/desc/8f476c85`　enemy，批次 20261101T092200-58f5979e，來源 claude

- KR：`피격 시 <color=#ff6000><mark=#ff000040><b><u>자신의 [ReinforcedTattooSpider] 1당 이 스킬 1회 재사용</u></b></mark></color> (최대 1회)⏎[WhenUse] 자신에게 [ReinforcedTattooSpider]이 있으면, 코인 적중시 대상의 [Laceration] 횟수 1 증가⏎[BeforeAttack] 자신의 체력이 50% 이하일 때, 피해량 +40%`
- 現行譯文：`受到攻擊時，<color=#ff6000><mark=#ff000040><b><u>自身每帶有1層 [ReinforcedTattooSpider] 將重複使用本技能1次</u></b></mark></color>(最多1次)⏎[WhenUse] 若自身帶有 [ReinforcedTattooSpider]，則本技能的硬幣命中時，使目標增加1回 [Laceration]⏎[BeforeAttack] 若自身的現存體力不高於上限的50%，則使本技能造成的傷害+40%`
- 註記：prev_zh 漏了 KR「(최대 1회)」，已補上(最多1次)
- 判讀：prev_zh 漏了 KR「(최대 1회)」，已補上(最多1次)
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N554　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Abnormality-pilgrimage.json|/dataList/20/levelList/0/desc/e90b971b`　enemy，批次 20261101T081954-e2d46ba8，來源 claude

- KR：`[WhenUse] [AbyssalVitality]이 50 이상이면, 수치를 5 소모하여 이 스킬에서 부여하는 신경 손상 부여값 +2⏎[WhenUse] 대상의 [NervousImpairment] 5당, [AbyssalVitality]을 1 소모하여 합 위력 +1 (최대 5)`
- 現行譯文：`[WhenUse] 若自身帶有不低於50層的 [AbyssalVitality]，則消耗5層並使本技能施加的神經損傷層數額外+2層⏎[WhenUse] 目標每帶有5層 [NervousImpairment]，則消耗1層 [AbyssalVitality] 並使本技能的拚點威力+1(最多+5)`
- 註記：draft 多出的 [AbyssalVitality] 已拿掉（KR 寫「수치를…소모」）；KR 寫韓文「신경 손상」、無 [ID]，改寫官方名「神經損傷」
- 判讀：draft 多出的 [AbyssalVitality] 已拿掉（KR 寫「수치를…소모」）；KR 寫韓文「신경 손상」、
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N555　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Abnormality-pilgrimage.json|/dataList/21/levelList/0/desc/e2d15175`　enemy，批次 20261101T081954-e2d46ba8，來源 claude

- KR：`[WhenUse] [AbyssalVitality]이 50 이상이면, 수치를 4 소모하여 코인 위력 +1⏎[WhenUse] 대상의 [NervousImpairment] 10당, [AbyssalVitality]을 3 소모하여 코인 위력 +1 (최대 2)`
- 現行譯文：`[WhenUse] 若自身帶有不低於50層的 [AbyssalVitality]，則消耗4層並使本技能的硬幣威力+1⏎[WhenUse] 目標每帶有10層 [NervousImpairment]，則消耗3層 [AbyssalVitality] 並使本技能的硬幣威力+1(最多+2)`
- 註記：draft 多出的 [AbyssalVitality] 已拿掉（KR 寫「수치를…소모」）
- 判讀：draft 多出的 [AbyssalVitality] 已拿掉（KR 寫「수치를…소모」）
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N556　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Abnormality-pilgrimage.json|/dataList/29/levelList/0/desc/e90b971b`　enemy，批次 20261101T081954-e2d46ba8，來源 claude

- KR：`[WhenUse] [AbyssalVitality]이 50 이상이면, 수치를 5 소모하여 이 스킬에서 부여하는 신경 손상 부여값 +2⏎[WhenUse] 대상의 [NervousImpairment] 10당, [AbyssalVitality]을 1 소모하여 합 위력 +1 (최대 2)`
- 現行譯文：`[WhenUse] 若自身帶有不低於50層的 [AbyssalVitality]，則消耗5層並使本技能施加的神經損傷層數額外+2層⏎[WhenUse] 目標每帶有10層 [NervousImpairment]，則消耗1層 [AbyssalVitality] 並使本技能的拚點威力+1(最多+2)`
- 註記：draft 多出的 [AbyssalVitality] 已拿掉（KR 寫「수치를…소모」）；KR 寫韓文「신경 손상」、無 [ID]，改寫官方名「神經損傷」
- 判讀：draft 多出的 [AbyssalVitality] 已拿掉（KR 寫「수치를…소모」）；KR 寫韓文「신경 손상」、
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N557　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Abnormality-pilgrimage.json|/dataList/30/levelList/0/desc/e2d15175`　enemy，批次 20261101T081954-e2d46ba8，來源 claude

- KR：`[WhenUse] [AbyssalVitality]이 50 이상이면, 수치를 4 소모하여 코인 위력 +1⏎[WhenUse] 대상의 [NervousImpairment] 10당, [AbyssalVitality]을 3 소모하여 코인 위력 +1 (최대 2)`
- 現行譯文：`[WhenUse] 若自身帶有不低於50層的 [AbyssalVitality]，則消耗4層並使本技能的硬幣威力+1⏎[WhenUse] 目標每帶有10層 [NervousImpairment]，則消耗3層 [AbyssalVitality] 並使本技能的硬幣威力+1(最多+2)`
- 註記：draft 多出的 [AbyssalVitality] 已拿掉（KR 寫「수치를…소모」）
- 判讀：draft 多出的 [AbyssalVitality] 已拿掉（KR 寫「수치를…소모」）
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N558　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Abnormality-tkt-re.json|/dataList/10/levelList/0/desc/13380ca0`　enemy，批次 20261101T092219-28df2d07，來源 claude

- KR：`[TimeAcceleration] 상태면, 코인 위력 +1⏎[WhenUse] 자신의 체력이 75% 미만이면, [OwnTime]을 40 소모하여, 합 위력 +2⏎[DefeatDuel] 피해량이 80%만큼 감소하고 이 스킬의 피해로 대상이 흐트러지지 않음`
- 現行譯文：`若自身帶有 [TimeAcceleration]，則使本技能的硬幣威力+1⏎[WhenUse] 若現存體力低於75%，則消耗40層 [OwnTime] 並使本技能的拚點威力+2⏎[DefeatDuel] 使本技能造成的傷害-80%，且本技能造成的傷害不會使目標陷入混亂`
- 註記：原譯多了 KR 沒有的「(每回合最多1次)」，且「-80」漏了 %（KR 80%만큼 감소），一併改正
- 判讀：原譯多了 KR 沒有的「(每回合最多1次)」，且「-80」漏了 %（KR 80%만큼 감소），一併改正
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N559　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Abnormality-tkt-re.json|/dataList/2/levelList/0/desc/13380ca0`　enemy，批次 20261101T092219-28df2d07，來源 claude

- KR：`[TimeAcceleration] 상태면, 코인 위력 +1⏎[WhenUse] 자신의 체력이 75% 미만이면, [LimitedTime]을 40 소모하여, 합 위력 +2⏎[DefeatDuel] 피해량이 80%만큼 감소하고 이 스킬의 피해로 대상이 흐트러지지 않음`
- 現行譯文：`若自身帶有 [TimeAcceleration]，則使本技能的硬幣威力+1⏎[WhenUse] 若現存體力低於75%，則消耗40層 [LimitedTime] 並使本技能的拚點威力+2⏎[DefeatDuel] 使本技能造成的傷害-80%，且本技能造成的傷害不會使目標陷入混亂`
- 註記：原譯多了 KR 沒有的「(每回合最多1次)」，且「-80」漏了 %（KR 80%만큼 감소），一併改正
- 判讀：原譯多了 KR 沒有的「(每回合最多1次)」，且「-80」漏了 %（KR 80%만큼 감소），一併改正
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N560　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Abnormality-tkt-re.json|/dataList/9/levelList/0/desc/319f57fd`　enemy，批次 20261101T081958-684225b0，來源 claude

- KR：`[TimeAcceleration] 상태면, 합 위력 +2⏎[WhenUse] 모든 부위가 보유한 [OwnTime]의 합이 600 이상이면, 대상의 [Laceration] 횟수 3 증가⏎[WhenUse] 모든 부위가 보유한 [OwnTime]의 합이 500 이상이면, 대상의 [Laceration] 횟수 1 증가⏎[WhenUse] 모든 부위가 보유한 [OwnTime]의 합이 400 이상이면, [OwnTime]을 20 소모하여, 이 스킬에서 부여하는 [Laceration] 위력과 횟수 +1`
- 現行譯文：`若自身帶有 [TimeAcceleration]，則使本技能的拚點威力+2⏎[WhenUse] 若所有部位的 [OwnTime] 層數之和不低於600層，則使目標增加3回 [Laceration]⏎[WhenUse] 若所有部位的 [OwnTime] 層數之和不低於500層，則使目標增加1回 [Laceration]⏎[WhenUse] 若所有部位的 [OwnTime] 層數之和不低於400層，則消耗20層 [OwnTime] 並使本技能增加的 [Laceration] 強度與回數額外+1`
- 註記：草稿第二行寫「增加2回」，KR 是 3，已改正
- 判讀：草稿第二行寫「增加2回」，KR 是 3，已改正
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N561　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Abnormality-walpu10.json|/dataList/22/levelList/0/desc/4e9a335c`　enemy，批次 20261112T040755-8f3b61ed，來源 claude

- KR：`[DuelCounter]⏎원호방어 전용 스킬⏎[WhenUse] 자신의 잃은 체력 20%당 합 위력 +1 (최대 4)⏎[WhenUse]⏎- [SupportProtect]한 대상 아군과 자신의 정신력 5 회복⏎- 최대 체력의 15%만큼 보호막 얻음 (턴당 1회)⏎- [Protection] 2 얻음 (턴당 1회)`
- 現行譯文：`[DuelCounter]⏎掩護防禦 專用技能⏎[WhenUse] 自身每失去20%體力，拚點威力+1 (最多4)⏎[WhenUse]⏎- 自身與受到 [SupportProtect] 的目標友方單位恢復5點理智值⏎- 獲得相當於最大體力15%的護盾 (每回合最多1次)⏎- 使自身獲得2層 [Protection] (每回合最多1次)`
- 註記：원호방어：KR 未用 [ID]，寫關鍵字名稱「掩護防禦 」
- 判讀：원호방어：KR 未用 [ID]，寫關鍵字名稱「掩護防禦 」
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N562　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Abnormality_Mirror6.json|/dataList/23/levelList/0/desc/4df86f5d`　enemy，批次 20261101T092442-347fce81，來源 claude

- KR：`[WhenUse] [BloodDinner]을 최대 100 소모하여 아래 효과 적용⏎- 소모한 [BloodDinner]만큼 본체와 몸통 부위 체력 회복⏎- 소모한 [BloodDinner] 20 당 이 스킬의 최종 위력 +1⏎[EndSkill] 이 스킬로 소모한 [BloodDinner]이 30 미만이면, 다음 턴에 [Vulnerable] 2 얻음`
- 現行譯文：`[WhenUse] 消耗最多100層 [BloodDinner] 並觸發以下效果⏎- 使自身本體與軀幹部位恢復等同於消耗的 [BloodDinner] 層數的體力⏎- 每消耗20層 [BloodDinner]，使本技能的最終威力+1⏎[EndSkill] 若本技能消耗的 [BloodDinner] 層數低於30層，則下回合對自身施加2層 [Vulnerable]`
- 註記：prev_zh 第三行誤譯為「下回合使自身獲得1層攻擊等級提升」，依 KR「이 스킬의 최종 위력 +1」改為「使本技能的最終威力+1」
- 判讀：prev_zh 第三行誤譯為「下回合使自身獲得1層攻擊等級提升」，依 KR「이 스킬의 최종 위력 +1」改為「使本技能
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N563　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Abnormality_Refraction2_bokgak.json|/dataList/1/levelList/0/desc/d5d7e085`　enemy，批次 20261101T084232-d72b623a，來源 claude

- KR：`[WhenUse] 대상이 가지고 있는 [WrappedCurseTagRe]만큼 [WrappedCurseTagRe] 부여⏎- 대상에게 [WrappedCurseTagRe]이 없으면 3 부여`
- 現行譯文：`[WhenUse] 對目標施加層數等同於其 [WrappedCurseTagRe] 層數的 [WrappedCurseTagRe]⏎- 若目標未帶有 [WrappedCurseTagRe]，則施加3層`
- 註記：draft 多一個 [WrappedCurseTagRe]，依 KR 刪除
- 判讀：draft 多一個 [WrappedCurseTagRe]，依 KR 刪除
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N564　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Abnormality_Refraction2_bokgak.json|/dataList/3/levelList/0/desc/3597b6b8`　enemy，批次 20261101T084232-d72b623a，來源 claude

- KR：`[StartBattle] 모든 부위에 [Protection] 3 얻음⏎[WhenUse] 대상의 [WrappedCurseTagRe]을 전부 소멸⏎- 소멸시킨 만큼 [ReattachedCurseTag_Re] 얻음`
- 現行譯文：`[StartBattle] 使所有部位獲得3層 [Protection]⏎[WhenUse] 消除目標所有 [WrappedCurseTagRe]⏎- 獲得與消除層數相同的 [ReattachedCurseTag_Re]`
- 註記：draft 多一個 [WrappedCurseTagRe]，依 KR 改寫末行
- 判讀：draft 多一個 [WrappedCurseTagRe]，依 KR 改寫末行
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N565　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Abnormality_Refraction5.json|/dataList/23/levelList/0/desc/d89aa13f`　enemy，批次 20261101T092631-9014f6b6，來源 claude

- KR：`[WhenUse] [BloodDinner]을 최대 100 소모하여 아래 효과 적용⏎- 소모한 [BloodDinner]만큼 본체와 몸통 부위 체력 회복⏎- 소모한 [BloodDinner] 20 당 이 스킬의 최종 위력 +1⏎[DefeatDuel] 최종 위력 -5⏎[EndSkill] 이 스킬로 소모한 [BloodDinner]이 30 미만이면, 다음 턴에 [Vulnerable] 2 얻음`
- 現行譯文：`[WhenUse] 消耗最多100層 [BloodDinner] 並觸發以下效果⏎- 使自身本體與軀幹部位恢復等同於消耗的 [BloodDinner] 層數的體力⏎- 每消耗20層 [BloodDinner]，使本技能的最終威力+1⏎[DefeatDuel] 使本技能的最終威力-5⏎[EndSkill] 若本技能消耗的 [BloodDinner] 層數低於30層，則下回合對自身施加2層 [Vulnerable]`
- 註記：prev_zh 第3行誤譯為「下回合獲得1層攻擊等級提升」，依 KR 改為「使本技能的最終威力+1」
- 判讀：prev_zh 第3行誤譯為「下回合獲得1層攻擊等級提升」，依 KR 改為「使本技能的最終威力+1」
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N566　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Abnormality_Refraction6.json|/dataList/12/levelList/0/desc/e2d17218`　enemy，批次 20261101T084251-a7c3fc83，來源 claude

- KR：`속도와 상관없이 이 스킬과 합을 할 수 있음⏎[WhenUse]⏎- 대상의 [Combustion] 6당 합 위력 +1 (최대 6)⏎- 체력이 66%/33% 미만이면, 두 번째/모든 코인이 [SuperCoin]으로 변경됨⏎[DefeatDuel] 이 스킬의 공격 종료시까지 대상이 흐트러지지 않음`
- 現行譯文：`可無視速度值與本技能拚點⏎[WhenUse]⏎- 目標每帶有6級 [Combustion] 強度，則使本技能的拚點威力 +1 (最多+6)⏎- 若自身的體力值低於66%/33%，則使本技能的第2枚/所有硬幣轉變為 [SuperCoin]⏎[DefeatDuel] 直到本技能的攻擊結束為止，目標不會陷入混亂`
- 註記：prev 把「두 번째」（第2枚）誤譯為「最後一枚」，13、15、16 同；「미만」改為「低於」
- 判讀：prev 把「두 번째」（第2枚）誤譯為「最後一枚」，13、15、16 同；「미만」改為「低於」
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N567　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Abnormality_Refraction6.json|/dataList/21/levelList/0/coinlist/1/coindescs/2/desc/12f0e563`　enemy，批次 20261101T084251-a7c3fc83，來源 claude

- KR：`[OnSucceedAttack] 파열 횟수 2 증가`
- 現行譯文：`[OnSucceedAttack] 使目標增加2回破裂`
- 註記：KR 用韓文狀態名「파열」，沒有 [Burst]，依 base.md 寫官方中文名「破裂」
- 判讀：KR 用韓文狀態名「파열」，沒有 [Burst]，依 base.md 寫官方中文名「破裂」
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N568　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Abnormality_Refraction6.json|/dataList/5/levelList/0/desc/aa484b26`　enemy，批次 20261101T084251-a7c3fc83，來源 claude

- KR：`이 스킬과 합을 진행한 스킬의 <color=#ff6000><mark=#ff000040><b><u>공격 유형, 속성 유형</u></b></mark></color>이 다가오는 미래에 영향을 끼침⏎- 4스테이지의 적들이 [RustedWingScalesExplain] 얻음`
- 現行譯文：`與本技能進行拚點的技能的<color=#ff6000><mark=#ff000040><b><u>攻擊類型和罪孽屬性</u></b></mark></color>將影響即將到來的未來⏎- 關卡4的敵方單位將獲得 [RustedWingScalesExplain]`
- 註記：draft 在「未來」中間插入 [TabExplain]，KR 沒有，已刪除
- 判讀：draft 在「未來」中間插入 [TabExplain]，KR 沒有，已刪除
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N569　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Enemy-a1c5p2.json|/dataList/10/levelList/0/desc/499533a3`　enemy，批次 20261101T082027-1807e017，來源 claude

- KR：`대상에게 <sprite name="Sinking"><color=#e30000><u><link="Sinking">침잠</link></u></color>이 있다면, 체력 회복 효과가 체력이 가장 낮은 백화 상태의 아군 1명에게 같이 적용됨`
- 現行譯文：`若目標帶有<sprite name="Sinking"><color=#e30000><u><link="Sinking">沉淪</link></u></color>，則本技能的體力恢復效果同時對現存體力最低的1名白化的友方單位生效`
- 註記：補回 KR 的 sprite／link 標籤
- 判讀：補回 KR 的 sprite／link 標籤
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N570　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Enemy-a1c5p2.json|/dataList/11/levelList/0/desc/60df2fc8`　enemy，批次 20261101T082027-1807e017，來源 claude

- KR：`대상에게 <sprite name="Sinking"><color=#e30000><u><link="Sinking">침잠</link></u></color>이 있다면, 체력 회복 효과가 체력이 가장 낮은 백화 상태의 아군 1명에게 같이 적용됨`
- 現行譯文：`若目標帶有<sprite name="Sinking"><color=#e30000><u><link="Sinking">沉淪</link></u></color>，則本技能的體力恢復效果同時對現存體力最低的1名白化的友方單位生效`
- 註記：補回 KR 的 sprite／link 標籤；KR 與 /dataList/10 相同、沒有「자신을 제외한」，拿掉 draft 的「除自身以外」以與 KR 一致（EN 有 other）
- 判讀：補回 KR 的 sprite／link 標籤；KR 與 /dataList/10 相同、沒有「자신을 제외한」，拿掉 d
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N571　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Enemy-a1c5p2.json|/dataList/12/levelList/0/coinlist/0/coindescs/0/desc/499533a3`　enemy，批次 20261101T082027-1807e017，來源 claude

- KR：`대상이 가지고있는 버프 하나를 다음 턴에 자신이 얻음.⏎(공격 위력 증가, 신속, 피해량 증가, <sprite name="Protection"><color=#f8c200><u><link="Protection">보호</link></u></color>에 한함)`
- 現行譯文：`下回合使自身獲得1種目標帶有的增益狀態。⏎(僅限強壯、迅捷、傷害強化與<sprite name="Protection"><color=#f8c200><u><link="Protection">守護</link></u></color>)`
- 註記：補回 KR 的 sprite／link 標籤
- 判讀：補回 KR 的 sprite／link 標籤
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N572　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Enemy-a1c5p2.json|/dataList/12/levelList/0/desc/499533a3`　enemy，批次 20261101T082027-1807e017，來源 claude

- KR：`[WhenUse] 대상에게 <sprite name="AaCePbBa"><color=#e30000><u><link="AaCePbBa">창백한 소음</link></u></color>이 있을 때, 자신을 제외한 체력이 가장 낮은 백화 상태의 아군 체력 30 회복`
- 現行譯文：`[WhenUse] 若目標帶有<sprite name="AaCePbBa"><color=#e30000><u><link="AaCePbBa">蒼白的噪音</link></u></color>，則使除自身以外現存體力最低的1名白化的友方單位恢復30點體力`
- 註記：補回 KR 的 sprite／link 標籤
- 判讀：補回 KR 的 sprite／link 標籤
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N573　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Enemy-a1c5p2.json|/dataList/20/levelList/0/desc/499533a3`　enemy，批次 20261101T082027-1807e017，來源 claude

- KR：`대상에게 <sprite name="AaCePbBa"><color=#e30000><u><link="AaCePbBa">창백한 소음</link></u></color>이 있을 때, 자신을 제외한 살아있는 아군당 대상의 정신력 피해 1`
- 現行譯文：`若目標帶有<sprite name="AaCePbBa"><color=#e30000><u><link="AaCePbBa">蒼白的噪音</link></u></color>，則除自身以外每有1名友方單位存活，對目標造成1點理智傷害`
- 註記：補回 KR 的 sprite／link 標籤
- 判讀：補回 KR 的 sprite／link 標籤
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N574　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Enemy-a1c5p2.json|/dataList/6/levelList/0/coinlist/0/coindescs/1/desc/499533a3`　enemy，批次 20261101T082027-1807e017，來源 claude

- KR：`[OnSucceedAttack] 대상이 가지고있는 버프 하나를 다음 턴에 자신이 얻음.⏎(공격 위력 증가, 신속, 피해량 증가, <sprite name="Protection"><color=#f8c200><u><link="Protection">보호</link></u></color>에 한함)`
- 現行譯文：`[OnSucceedAttack] 下回合使自身獲得1種目標帶有的增益狀態。⏎(僅限強壯、迅捷、傷害強化與<sprite name="Protection"><color=#f8c200><u><link="Protection">守護</link></u></color>)`
- 註記：補回 KR 的 sprite／link 標籤
- 判讀：補回 KR 的 sprite／link 標籤
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N575　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Enemy-a1c5p2.json|/dataList/9/levelList/0/coinlist/0/coindescs/1/desc/499533a3`　enemy，批次 20261101T082027-1807e017，來源 claude

- KR：`대상이 가지고있는 버프 하나를 다음 턴에 자신이 얻음.⏎(공격 위력 증가, 신속, 피해량 증가, <sprite name="Protection"><color=#f8c200><u><link="Protection">보호</link></u></color>에 한함)`
- 現行譯文：`下回合使自身獲得1種目標帶有的增益狀態。⏎(僅限強壯、迅捷、傷害強化與<sprite name="Protection"><color=#f8c200><u><link="Protection">守護</link></u></color>)`
- 註記：補回 KR 的 sprite／link 標籤
- 判讀：補回 KR 的 sprite／link 標籤
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N576　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Enemy-a1c6p2.json|/dataList/23/levelList/0/coinlist/2/coindescs/2/desc/642c809b`　enemy，批次 20261101T082037-1eb155da，來源 claude

- KR：`[OnSucceedAttackHead] [DefenseDown] 2 부여`
- 現行譯文：`[OnSucceedAttackHead] 對目標施加2層 [DefenseDown]`
- 註記：KR 寫 [DefenseDown]，draft 改成中文名，還原為 [ID]
- 判讀：KR 寫 [DefenseDown]，draft 改成中文名，還原為 [ID]
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N577　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Enemy-a1c6p2.json|/dataList/36/levelList/0/desc/14955389`　enemy，批次 20261101T082037-1eb155da，來源 claude

- KR：`[WinDuel] 다음 턴에 [Enhancement], 피해량 증가 1 얻음`
- 現行譯文：`[WinDuel] 下回合使自身獲得1層 [Enhancement] 與1層傷害強化`
- 註記：KR 寫韓文「피해량 증가」（無 [ID]），依 base.md 寫官方名「傷害強化」（Bufs.json）；draft 多加了 KR 沒有的 [AttackDmgUp]
- 判讀：KR 寫韓文「피해량 증가」（無 [ID]），依 base.md 寫官方名「傷害強化」（Bufs.json）；draft
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N578　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Enemy-a1c7p1.json|/dataList/15/levelList/0/desc/a3ce8f4c`　enemy，批次 20261101T082038-e8ee0ca2，來源 claude

- KR：`[WhenUse] 혈찬을 최대 50 소모하여 소모한 혈찬 10당 [BloodScissor] 1 얻음.⏎[WhenUse] 이 스킬에서 [BloodDinner]을 소모하지 못했으면, [Laceration] 5, [HardenedBlood]을 1 얻음.⏎[WhenUse] 대상의 [Laceration]이 5 이상이면, 코인 위력 +1`
- 現行譯文：`[WhenUse] 消耗最多50層血宴，每消耗10層血宴，使自身獲得1層 [BloodScissor]⏎[WhenUse] 若本技能未消耗 [BloodDinner]，則使自身增加5級 [Laceration] 強度並使自身獲得1層 [HardenedBlood]⏎[WhenUse] 若目標的 [Laceration] 強度不低於5級，則使本技能的硬幣威力+1`
- 註記：KR 第一行用韓文「혈찬」，依 base.md 寫「血宴」，不寫 [BloodDinner]
- 判讀：KR 第一行用韓文「혈찬」，依 base.md 寫「血宴」，不寫 [BloodDinner]
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N579　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Skills_Enemy-a1c7p2.json|/dataList/27/levelList/0/desc/bf858559`　enemy，批次 20261101T082046-68dfcca2，來源 claude

- KR：`[WhenUse] 자신의 [FestivalFever] 5 또는 대상의 [Laceration] 6당 최종 위력 +1 (최대 2)⏎[WhenUse] [Laceration] 15를 얻은 다음 즉시 1회 발동시키고 [Laceration] 횟수 1 감소`
- 現行譯文：`[WhenUse] 自身每帶有5層 [FestivalFever] 或目標每帶有6級 [Laceration] 強度，使本技能的最終威力+1(最多+2)⏎[WhenUse] 使自身增加15級 [Laceration] 強度並立即觸發1次，隨後使自身的 [Laceration] 回數減少1回`
- 註記：刪去多寫的 [Laceration]（本批同樣處理）
- 判讀：刪去多寫的 [Laceration]（本批同樣處理）
- 理由：譯者說明依 KR／規則修正草稿，現行譯文已照改，不需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N580　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/PersonalityVoiceDlg/Voice_DonQuixote_Wedge_10316.json|/dataList/2/dlg/19e89bc4`　identity，批次 20261112T034808-4c191ca9，來源 claude

- KR：`<color=#8b9c15>탕수육.</color> 육개장! <color=#8b9c15>장봉뵈르.</color> 르… 블로숑을 얹은 감자 그롸탕! <color=#8b9c15>하… 탕, 탕… 바오?</color> 오… 오므롸이스는 어떤가! <color=#8b9c15>…훗, 스뫼레브뢰.</color> 으에엥?! 뢰는 또 뭔가, 뢰는!!! <color=#8b9c15>샌드위치예요. 자, 그럼 오늘 점심은 사시는 거죠?</color> 우… 알았네. 오므롸이스… 포장해오리다.`
- 現行譯文：`<color=#8b9c15>糖醋肉。</color> 肉醬麵！ <color=#8b9c15>麵包佐火腿奶油。</color> 油……油封鴨佐起司焗烤馬鈴薯濃湯！ <color=#8b9c15>唉……湯、湯……湯包？</color> 包……包著蛋的歐姆萊斯如何！ <color=#8b9c15>……哼，斯莫勒布洛。</color> 嗚欸欸？！洛又是什麼啊，洛！！！ <color=#8b9c15>是三明治啦。好了，那今天的午餐就由您請客了吧？</color> 唔……知道了。吾去外帶……歐姆萊斯回來。`
- 註記：KR 是食物接龍（탕수육→육개장→장봉뵈르→르…그롸탕→탕바오→오므롸이스→스뫼레브뢰），JP 也改寫成日文的接龍；中文依字尾接龍重寫：唐吉訶德的怪腔「오므롸이스」寫成音譯「歐姆萊斯」，辛克萊最後用丹麥開放式三明治（smørrebrød）的音譯讓唐吉訶德接不下去
- 判讀：KR 食物接龍，中文依字尾重寫接龍
- 理由：文字遊戲無法直譯，已重寫，與 Sinclair 側同段一致
- 查證：Voice_Sinclair_Wedge_11017 #2 使用相同接龍譯文
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N581　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/PersonalityVoiceDlg/Voice_Heathcliff_Contem_10717.json|/dataList/7/dlg/8913fcfe`　identity，批次 20261101T074258-92ace468，來源 claude

- KR：`원하는 대로 손바느질해 줬잖아!!! 뭐?! 균일해서 기계 티가 난다고?! 이 개#%@!가! 이리 안 와?! 네놈 피부 위에서 직접 18땀을 재현해 주마! 이 바늘로!!!`
- 現行譯文：`不是照你要的手縫了嗎！！！什麼？！太均勻了，看起來像機器縫的？！你這狗#%@!！還不給我過來？！我就直接在你這傢伙的皮膚上重現18針！用這根針！！！`
- 註記：KR「18땀」的18有韓文髒話（씨발）的諧音，中文無對應諧音，保留數字18
- 判讀：「18땀」的 18 在韓文是髒話諧音，中文無對應，保留數字
- 理由：諧音無法保留，現行處理合理
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N582　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/PersonalityVoiceDlg/Voice_Heathcliff_KumoThumb_10716.json|/dataList/5/dlg/877ec7a4`　identity，批次 20261101T074300-e858d5ee，來源 claude

- KR：`매번 티켓은 훨씬 잘 했다고 갈구더니… 쯧. 뭐가 뭔지 모르겠네. 싱클레어 그 자식 가르쳐주는 거 보니까 아예 다른 검술이잖아?⏎…뭐, 됐나. 어차피 집 나가신 요시히데, 그자의 검술만 배끼면 되니까. 시킨 대로만 하자. 시킨 대로만.`
- 現行譯文：`明明每次都挑我毛病，說門票做得比我好多了……嘖。還真是搞不懂。看她教辛克萊那傢伙時，用的根本就是完全不同的劍術啊？⏎……算了，隨便。反正只要複製出離家出走的良秀用的劍術就好了。就照著被吩咐的去做吧。只照著被吩咐的。`
- 註記：draft 的「戲謔」與 KR 갈구다（找碴、數落）不符，改為「挑我毛病」；ruby 依 base.md 只留「良秀」
- 判讀：draft 的「戲謔」與 KR 갈구다（找碴）不符，改為「挑我毛病」；ruby 只留「良秀」
- 理由：依 KR 修正並依 base.md 處理 ruby，已落實
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N583　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/PersonalityVoiceDlg/Voice_Ishmael_Xichun_10812.json|/dataList/4/dlg/6dde4974`　identity，批次 20261101T074319-76d3e723，來源 claude

- KR：`홍원의 가주 후보자, 가씨 가의 이스마엘. 단서를 찾아 홍원으로 돌아왔어요. 어… 뭔가 이상하지 않냐고요…?⏎…이상할 게 뭐가 있어요. 저는 태어날 때부터 가씨 가문에서 살아왔으니, 가씨인게 당연하잖아요.`
- 現行譯文：`我是鴻園的家主候選人，賈家的以實瑪利。為了尋找線索而回到了鴻園。呃……問我是不是有哪裡怪怪的……？⏎……有什麼好奇怪的呢。我從出生起便在賈氏家族生活，姓賈不是理所當然的嘛。`
- 註記：draft「找到線索後便回到了」依 KR「단서를 찾아 돌아왔어요」（JP 手掛かりを求めて）改為「為了尋找線索而回到了」；補上「~냐고요」的轉述語氣
- 判讀：draft 誤譯「找到線索後」，依 KR 改為「為了尋找線索而回到」並補轉述語氣
- 理由：已依 KR 修正
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N584　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/PersonalityVoiceDlg/Voice_Ryoshu_Contem_10416.json|/dataList/11/dlg/477d2006`　identity，批次 20261101T074413-12f6d408，來源 claude

- KR：`르루주놈들의 이. 계. 를 무너뜨릴 시간이군.`
- 現行譯文：`是時候摧毀丹紅派那群傢伙的怪．計了。`
- 註記：縮略語「이. 계.」依 JP「変・計」（이상한 계략）譯「怪．計」
- 判讀：縮略語「이. 계.」依 JP 譯「怪．計」
- 理由：字謎處理說明，已落實
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N585　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/PersonalityVoiceDlg/Voice_Ryoshu_Contem_10416.json|/dataList/3/dlg/ac296229`　identity，批次 20261101T074413-12f6d408，來源 claude

- KR：`솔드 아웃. 이번에 올라온 고객들은 안목이 있군. 다. 넷. 놈. 매장 문을 닫을 시간이다.⏎햇빛에 바래진 신발은 없는지, 지면에 남은 역겨운 자취가 없는지 다 확인 한 뒤에 들어가도록. 비천한 포레그인 네놈들의 유일한 쓸모니까.`
- 現行譯文：`售罄。這次上樓的顧客還算有眼光。四．腳．貨。該關店門了。⏎確認過有沒有被陽光曬褪色的鞋、地面上有沒有留下噁心的痕跡之後再進去。畢竟這是你們這群卑賤的四腳唯一的用處。`
- 註記：縮略語「다. 넷. 놈.」（다리 넷 달린 놈）譯「四．腳．貨」
- 判讀：縮略語「다. 넷. 놈.」譯「四．腳．貨」
- 理由：字謎處理說明，已落實
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N586　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/PersonalityVoiceDlg/Voice_Ryoshu_Contem_10416.json|/dataList/7/dlg/2aa451b4`　identity，批次 20261101T074413-12f6d408，來源 claude

- KR：`르누아르의 미학은 절제에서 시작된다. 그저 자신을 보여주지 못해 안달난 르루주놈들은 결코 이해하지 못한 패션이지.⏎그놈들의 패션엔 하이라이트가 너무 많아. 어디 눈을 둬야할지 모르니, 자꾸만 밟. 분.을 참을 수가 없고.␣`
- 現行譯文：`漆黑派的美學始於節制。那是只急著炫耀自己的丹紅派傢伙們永遠無法理解的時尚。⏎那些傢伙的時尚亮點太多了。讓人不知道該把視線放在哪裡，所以我老是忍不住想踩．粉。`
- 註記：縮略語「밟. 분.」（밟아 분쇄）譯「踩．粉」
- 判讀：縮略語「밟. 분.」譯「踩．粉」
- 理由：字謎處理說明，已落實
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N587　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/PersonalityVoiceDlg/Voice_Ryoshu_Contem_10416.json|/dataList/8/dlg/df582709`　identity，批次 20261101T074413-12f6d408，來源 claude

- KR：`신발은 땅을 딛기 위해 만들어진 예술… 바닥에 닿는 면이 적을 수록, 작품은 더 고상해지고, 패션의 가치 또한 드높아진다.⏎그 상식을 거부하고, 서스럼 없이 지면을 밟아대는 르루주의 디자이너놈들은… 모. 퇴. 해도 모자람이 없을 테지.`
- 現行譯文：`鞋子是為了踏地而生的藝術……接觸地面的面積越小，作品就越高雅，時尚的價值也越高。⏎拒絕這種常識、毫無顧忌地踩踏地面的丹紅派設計師們……就算讓他們全．退也不為過吧。`
- 註記：縮略語「모. 퇴.」依 JP「全・退」解為모두 퇴출，譯「全．退」
- 判讀：縮略語「모. 퇴.」依 JP 譯「全．退」
- 理由：字謎處理說明，已落實
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N588　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/StoryData/P10114.json|/dataList/50/content/42c55ad9`　identity，批次 20261101T074440-77728845，來源 claude

- KR：`그래서 내가 말하기를… 그럼 홍원에 즐거운 것이 있게끔 만들려면, 가주가 되는 것이 낫지 않겠나… 라고 했지.`
- 現行譯文：`於是我說……那麼若想要為鴻園帶來快樂，成為家主豈不是更好……我是這麼說的。`
- 註記：draft「應以成為家主而努力」偏向 EN，依 KR「가주가 되는 것이 낫지 않겠나」改為「成為家主豈不是更好」；麽→麼
- 判讀：draft 偏向 EN，依 KR 改為「成為家主豈不是更好」
- 理由：已依 KR 修正
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N589　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/StoryData/P10315.json|/dataList/16/content/96294f5c`　identity，批次 20261101T074531-a7bc4ca9，來源 claude

- KR：`그 정의가 짐작되지 않으니, 이 층을 떠날 때까지 곁에 두고 지켜볼 것이라네.`
- 現行譯文：`吾揣度不出汝的正義為何，故在汝離開此樓層之前，吾都將把汝留在身旁觀察。`
- 註記：KR 정의 是定義／正義的雙關，下一句杜布瓦反問；中文取「正義」（後句以天秤呼應）
- 判讀：정의（定義／正義）雙關，取「正義」
- 理由：雙關取捨說明，無需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N590　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/StoryData/P10416.json|/dataList/2/content/bbf3a655`　identity，批次 20261101T074623-0b2f9902，來源 claude

- KR：`이제 막 티를 벗었군. 신규 고객놈인가.`
- 現行譯文：`才剛褪去青澀啊。是新客人嗎。`
- 註記：良秀的縮略語：용.소.들. → 竟．聽．傳（竟然聽到了傳聞啊）；별.이.소.다. → 盡．聽．怪．話（盡是聽到些怪話啊）；신.고.（신규 고객）→ 新．客（新客人）
- 判讀：良秀的縮略語（用소들、별이소다、신고）的重寫處理
- 理由：字謎處理說明，已落實
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N591　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/StoryData/P10508.json|/dataList/71/content/709eddb5`　identity，批次 20261101T080812-f4ee48cd，來源 claude

- KR：`전통과 위계가 엄격한 S사에서는 고용인이 도망치거나 금기를 범했을 때, 추노꾼을 풀어 쫓게 만들지.`
- 現行譯文：`在傳統與位階森嚴的S公司，受僱之人若是逃跑或觸犯禁忌，就會派出推奴手去追捕。`
- 註記：prev_zh 把「고용인」（受僱者）誤譯成「僱主」，已改正
- 判讀：prev_zh 把 고용인 誤譯成「僱主」，已改正
- 理由：已修正；「推奴手」與詞彙表 추노꾼 一致
- 查證：glossary 추노꾼 → 推奴手，現行相符
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N592　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/StoryData/P10512.json|/dataList/51/content/7a87147c`　identity，批次 20261101T074632-b2627976，來源 claude

- KR：`거기에 ‘나’를, 날뛰게 할 만한 값까지 얹어 치를 수 있는 준비가 되어있는지를 묻는 겁니다.`
- 現行譯文：`我是在問您，是否做好了準備，連足以讓「我」大鬧一場的代價也一併付上。`
- 註記：draft 的 <ruby=俺>我</ruby> 只是 JP 原文，依 base.md 只留「我」；draft 漏譯「값까지 얹어 치를」（連代價一併付上），已補
- 判讀：去掉 JP ruby「俺」並補譯漏譯的「값까지 얹어 치를」
- 理由：已依 base.md 與 KR 修正
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N593　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/StoryData/P10613.json|/dataList/40/content/fd42afb8`　identity，批次 20261101T074642-9a915499，來源 claude

- KR：`대관원에 사는 이라면 한 번쯤은 들어보았을 선인에 대한 노랫말이다.`
- 現行譯文：`住在大觀園內的人們應該都曾聽過，關於仙人們的歌謠。`
- 註記：草稿的 <ruby=．> 只是著重號（KR 無標籤、無雙關），不改寫成「Y(X)」，直接拿掉
- 判讀：草稿的著重號 ruby 直接拿掉
- 理由：依規則處理，已落實
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N594　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/StoryData/P10616.json|/dataList/107/content/6b943de8`　identity，批次 20261101T074710-8618fd48，來源 claude

- KR：`도시의 해결사가… 어떻게 의뢰자와 결투자, 두 사람을 ‘해결’하는지를 말이야.`
- 現行譯文：`記錄都市的收尾人……是如何替委託人與決鬥者這兩個人「收尾」的。`
- 註記：KR 以「해결」呼應해결사；譯「收尾」呼應「收尾人」
- 判讀：해결 呼應 해결사，譯「收尾」
- 理由：詞彙表 해결사＝收尾人，譯法一致
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N595　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/StoryData/P10714.json|/dataList/16/content/53ccb70d`　identity，批次 20261101T074725-711d4e70，來源 claude

- KR：`늬들만 재미 보면 다냐? 우리가 쪼아먹을 건 남겨뒀어야지!`
- 現行譯文：`難道只讓你們爽就好了嗎？也該留點讓我們啄食的獵物啊！`
- 註記：KR 無 <ruby>；draft 的 <ruby=殺>啄</ruby> 只是解釋字義（啄＝獵殺），依規則把意思融入句子（「啄食的獵物」），不加括號
- 判讀：draft 的 ruby 解釋字義，改把意思融入句子
- 理由：依規則處理，已落實
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N596　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/StoryData/P10807.json|/dataList/17/content/8c47ce23`　identity，批次 20261101T074823-42abfd76，來源 claude

- KR：`당연히 아이에게 남은 건 아무것도 없었고… 상황을 전부 파악하기도 전에 크<size=110%>랲</size>게가 아이를 향해 성큼성큼 기어 오고 있었지.`
- 現行譯文：`當然，孩子身上什麼也沒剩下……甚至在她完全掌握情況之前，就有一隻<size=110%>廢料蟹</size>正大步大步地向她爬來。`
- 註記：draft「咄咄逼人地」取自 EN，依 KR 성큼성큼 改為「大步大步地」
- 判讀：성큼성큼 依 KR 改為「大步大步地」
- 理由：已依 KR 修正
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N597　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/StoryData/P10807.json|/dataList/4/content/88bc6c6a`　identity，批次 20261101T074823-42abfd76，來源 claude

- KR：`그렇게 쌓인 폐품 더미들은, 벌써 크<size=110%>랲</size>게 두 마리의 크기를 넘어설 만큼 쌓여있었어.`
- 現行譯文：`就這樣堆起來的廢品堆，已經比兩隻<size=110%>廢料蟹</size>還要高了。`
- 註記：draft 前半「很快這隻螃蟹就被放進了廢料堆」是 EN 的內容，KR 沒有，已刪；KR 只說廢品堆已高過兩隻廢料蟹
- 判讀：刪去 draft 中 EN 才有的內容，依 KR 只說廢品堆高過兩隻蟹
- 理由：已依 KR 修正
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N598　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/StoryData/P10812.json|/dataList/93/content/a37c5578`　identity，批次 20261101T074835-8aa25374，來源 claude

- KR：`입에서 나오는 모든 인의로운 말은 텅 비어있었으니까.`
- 現行譯文：`嘴裡吐出的所有充滿仁義的話語，也都不過是虛無縹緲之詞。`
- 註記：草稿 <ruby=伽藍堂>虛無縹緲</ruby> 的注音只是日文「がらんどう」的字義，無雙關，依規則併入句意、不加括號；補上 KR 的「모든」
- 判讀：ruby 無雙關，併入句意並補「모든」
- 理由：依規則處理，已落實
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N599　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/StoryData/P10815.json|/dataList/54/content/6dc40bcf`　identity，批次 20261101T080826-5668f9fb，來源 claude

- KR：`그럼 이번엔 후배만 믿는다? 변곡점은 팀장님이 찾아주셨으니까, 저 부분만 베면 돼.`
- 現行譯文：`那這次就全靠後輩囉？曲折點已經由隊長幫我們找出來了，只要砍那個部分就行了。`
- 註記：prev_zh 把「찾아주셨으니까」（已經找到）譯成未來式，已改正
- 判讀：prev_zh 把「찾아주셨으니까」誤成未來式，已改正
- 理由：已修正
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N600　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/StoryData/P10916.json|/dataList/133/teller/0c8c7a40`　identity，批次 20261101T074931-79c6c2d3，來源 claude

- KR：`요시히데`
- 現行譯文：`良秀`
- 註記：草稿 <ruby=YoshiHide>良秀</ruby> 的注音只是讀音，依 base.md 只留「良秀」
- 判讀：teller 去掉 ruby，只留「良秀」
- 理由：依 base.md 處理，已落實
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N601　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/StoryData/P10916.json|/dataList/24/teller/2b4c8d32`　identity，批次 20261101T074900-5926b9be，來源 claude

- KR：`요시히데`
- 現行譯文：`良秀`
- 註記：草稿 <ruby=YoshiHide>良秀</ruby> 的注音只是讀音，依規則只留「良秀」（既有譯文 요시히데 皆作良秀）；티켓 沿用 S9 的「門票」，로지온 用「羅狄翁」
- 判讀：teller 去掉 ruby，只留「良秀」
- 理由：依 base.md 處理，已落實
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N602　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/StoryData/P10916.json|/dataList/272/teller/f842070d`　identity，批次 20261101T074939-1ee3caef，來源 claude

- KR：`요시히데`
- 現行譯文：`良秀`
- 註記：teller 依 base.md 把草稿的 <ruby=YoshiHide>良秀</ruby> 改成只留「良秀」
- 判讀：teller 去掉 ruby，只留「良秀」
- 理由：依 base.md 處理，已落實
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N603　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/StoryData/P11007.json|/dataList/41/content/3b9c36d2`　identity，批次 20261101T075059-623ad9b5，來源 claude

- KR：`영 개운하지 않은 표정으로, 아이는 힘겹게 크<size=110%>랲</size>게를 썰어나갔지.`
- 現行譯文：`孩子帶著滿臉的不痛快，費力地將<size=110%>廢料蟹</size>一隻隻砍倒。`
- 註記：KR 썰어나갔지 是一隻接一隻地砍，draft「拚盡全力才把……打倒」略改
- 判讀：썰어나갔지 是一隻接一隻地砍，已修正 draft
- 理由：已依 KR 修正
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N604　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/StoryData/P11013.json|/dataList/8/content/90ee6b4c`　identity，批次 20261101T075105-5a8615e1，來源 claude

- KR：`아! 저번에는 그 유명하신 카포 콰르토… 아, 함부로 그분의 존함을 적을 순 없겠죠.`
- 現行譯文：`啊！前幾天那位有名的四階指揮官……啊，可不能如此輕率地記載那位的尊名。`
- 註記：KR 沒有 ruby；draft 的 <ruby=御名>名字</ruby> 只是解釋字義，依 base.md 融入句子寫「尊名」（KR 존함）
- 判讀：draft 的 ruby 解釋字義，融入句子寫「尊名」
- 理由：依 base.md 處理，已落實
- 查證：詞彙表 카포→指揮官，本句「四階指揮官」一致
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N605　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/StoryData/P11016.json|/dataList/68/content/a836ba4c`　identity，批次 20261101T083824-d76991d6，來源 claude

- KR：`예상치 못한 시즌에 들이닥쳤다고 생각하겠지만…`
- 現行譯文：`他們想必以為自己是在意想不到的一季突襲而來……`
- 註記：prev_zh 寫「你們以為」，但這是旁白描述漆黑派（前一行 #67），KR 생각하겠지만 的主詞是他們，改為「他們」
- 判讀：prev_zh 主詞「你們」誤，改為「他們」
- 理由：已修正
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N606　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/StoryData/P11215.json|/dataList/56/content/126e4cd4`　identity，批次 20261101T075126-6201a7a1，來源 claude

- KR：`그렇게 곧 죽을 것 같은 복선만 깔지 않아도 한 사람분은 되겠군.`
- 現行譯文：`只要不立下那種隨時都會死去的旗子，你應該就能獨當一面了吧。`
- 註記：草稿 <ruby=Flag>旗子</ruby> 的注音只是外語原文、無雙關，依規則只留「旗子」
- 判讀：ruby 為外語原文無雙關，只留「旗子」
- 理由：依規則處理，已落實
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N607　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/BattleKeywords-a1c10p2.json|/dataList/28/desc/324fc851`　keywords，批次 20261101T084326-cf67bd4b，來源 claude

- KR：`- 전투에 등장한 후, 2턴마다 턴 시작시 르 보미스망의 영향 범위가 확산됨⏎· 1단계 : 뿌이⏎· 2단계 : 모든 아군 수감자에게 적용⏎· 3단계 : 모든 대상에게 적용`
- 現行譯文：`- 於戰鬥中登場後，每2回合於回合開始時擴大嘔吐物的影響範圍⏎．第1階段：普依⏎．第2階段：適用於所有友方罪人⏎．第3階段：適用於所有目標`
- 註記：KR 只寫「르 보미스망」，依狀態名寫「嘔吐物」，不加原文
- 判讀：KR 只寫 르 보미스망，依狀態名寫「嘔吐物」不加原文
- 理由：依規則處理，無需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N608　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/BattleKeywords-walpu10.json|/dataList/30/desc/abe77479`　keywords，批次 20261112T034008-535e2fbd，來源 claude

- KR：`- 사기 저하⏎- 화상이 있는 적에게 가하는 피해량 -5%⏎- 진동이 있는 적에게 가하는 흐트러짐 피해량 +5%`
- 現行譯文：`- 士氣低落⏎- 對帶有燒傷 的敵方單位造成的傷害-5%⏎- 對帶有震顫 的敵方單位造成的混亂閾值前移量+5%`
- 註記：흐트러짐 피해량 依 EN「Stagger Threshold raised」、JP「混乱区間前進量」譯「混亂閾值前移量」
- 判讀：흐트러짐 피해량 依 EN/JP 譯「混亂閾值前移量」
- 理由：依輔助語言處理，無需動作
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N609　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心中

`note/BattleKeywords_Mirror8.json|/dataList/0/desc/75292a55`　keywords，批次 20261105T043422-68ebdc7c，來源 claude

- KR：`최댓값: 3⏎- 수치 1당 코인 위력 +1, 적에게 가하는 기본 스킬 피해량 +15%⏎- 와일드헌트 히스클리프가 퇴각 및 사망했으면, 소멸`
- 現行譯文：`最大值：3⏎- 每1層使自身的硬幣威力+1、對敵方造成的基礎技能傷害+15%⏎- 若狂獵希斯克利夫已撤退或死亡，則本效果消失`
- 註記：와일드헌트 히스클리프 寫「狂獵希斯克利夫」（人格名「狂獵」，Personalities）。「狂獵」也是 Bufs 的效果名，EGOgift_MirrorDungeon_8 的 KR 用 <noparse>와일드헌트</noparse> 包住；本句 KR 沒有 noparse，依規定未自行新增，遊戲中可能被讀成關鍵字，需確認
- 判讀：와일드헌트（狂獵）在本句 KR 沒有 noparse，依規定未自行新增，需遊戲內確認
- 理由：base.md 的 noparse 規則：新增只限已在遊戲中確認會被讀成關鍵字的地方，譯者未新增是對的。這需要實機確認，無法由譯文判斷；確認為已讀即可，若日後遊戲內確認會被讀成關鍵字再另行處理。
- 查證：狀態名 와일드헌트→狂獵在 BattleKeywords.json#284/name、Bufs.json#361/name；Bufs_Mirror8.json#0/desc 與本單位同文同譯。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N610　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Bufs-BossRaid.json|/dataList/9/desc/c1b95199`　keywords，批次 20261101T104454-3c2a77ef，來源 claude

- KR：`- 스테이지 시작 시 <color=#f8c200>중지 소속</color> 인격이 4명 이상 출전한 상태면, 리카르도의 첫 번째, 두 번째 <color=#f8c200>패턴이 변경</color>되며, <color=#f8c200>모든 E.G.O 자원을 2개씩 얻음</color>⏎- <color=#f8c200>수비 스킬을 장착</color>한 인격에게는 리카르도의 스킬 효과로 <color=#f8c200>마비가 부여되지 않음</color>⏎- 인격의 스킬 중 <color=#f8c200>출혈 또는 특수 출혈</color>을 부여하거나 <color=#f8c200>질투</color> 속성에 해당하는 공격 스킬의 피해량 +30% (E.G.O 스킬일 경우, 대신 피해량 +100%)⏎- 전투 시작 시 <color=#f8c200>완전 공명</color>이 발동했으면, 가장 수량이 적은 속성의 <color=#f8c200>E.G.O 자원 1개를 4회 획득</color>`
- 現行譯文：`- 關卡開始時，若登場的<color=#f8c200>屬於中指</color>的人格不低於4名，則<color=#f8c200>改變</color>里卡多第1、2階段的模式，並<color=#f8c200>獲得所有屬性的罪孽碎片各2個</color>⏎- <color=#f8c200>裝備了守備技能</color>的人格不會被里卡多的技能效果<color=#f8c200>施加麻痺</color>⏎- 人格施加<color=#f8c200>流血或特殊流血</color>的攻擊技能或<color=#f8c200>嫉妒</color>的攻擊技能造成的傷害+30%(若是E.G.O技能，則改為造成的傷害+100%)⏎- 戰鬥開始時，若觸發<color=#f8c200>完全共鳴</color>，則重複4次使我方獲得<color=#f8c200>持有數最少的罪孽碎片1個</color>`
- 註記：KR「첫 번째, 두 번째 패턴」字面是「第1、第2個模式」，沿用同檔 #13 既有的「第1、2階段的模式」以保持一致
- 判讀：「첫 번째, 두 번째 패턴」沿用同檔 #13 的「第1、2階段的模式」
- 理由：為與同檔既有譯文一致，不需動作。
- 查證：Bufs-BossRaid.json#13/desc（paratranz）寫「里卡多第1、2階段的模式」，本單位用語相同；KR 相同的 BattleKeywords 單位 BattleKeywords-BossRaid.json\|/dataList/9/desc 現行譯文與本單位完全相同。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N611　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Bufs-a1c10p2.json|/dataList/13/flavor/dae0f993`　keywords，批次 20261101T092737-1dff83ff，來源 claude

- KR：`표류하는 자들은 흐름을 바꿀 수는 없어도 흐름을 공고히 하고 흐름 속에서 자신의 일을 한다.`
- 現行譯文：`漂流者即使無法改變流向，也會鞏固流向，並在流向之中完成自己的工作。`
- 註記：未照 hint：hint 引的 BattleKeywords-a1c10p2 舊譯文已在批次 20261101T092653-1dbe095a 依 hint 改成與 BattleKeywords.json／Bufs.json（KR 相同）一致的譯文，這裡也用該譯文，讓四個檔案一致
- 判讀：未照 hint：舊譯已在批次 20261101T092653-1dbe095a 改成與 BattleKeywords.json／Bufs.json 一致
- 理由：譯者選擇與同 KR 的其他檔一致，不需動作。
- 查證：KR 相同的 BattleKeywords 單位 BattleKeywords-a1c10p2.json\|/dataList/13/flavor、BattleKeywords.json\|/dataList/581/flavor 現行譯文與本單位完全相同
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N612　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Bufs-a1c10p2.json|/dataList/15/flavor/dae0f993`　keywords，批次 20261101T092737-1dff83ff，來源 claude

- KR：`지하에서 만들어진 원초의 재료를 끊어 내기 위한 가위는 더욱 특별하고 거대하다. 처음으로 직조된 천을 자르고 잘라 왔던 이 가위에서 꽃이 피어나는 건 근원을 잘라 왔기 때문이며, 자연스러운 현상이다.`
- 現行譯文：`用來剪斷在地下製成之原初材料的剪刀，更加特別而巨大。這把一直裁剪著最初織成之布料的剪刀之所以會綻放花朵，是因為它剪斷的一直是根源，這是自然的現象。`
- 註記：未照 hint：hint 引的 BattleKeywords-a1c10p2 舊譯文已在批次 20261101T092653-1dbe095a 依 hint 改成與 BattleKeywords.json／Bufs.json（KR 相同）一致的譯文，這裡也用該譯文，讓四個檔案一致
- 判讀：未照 hint：舊譯已在批次 20261101T092653-1dbe095a 改成與 BattleKeywords.json／Bufs.json 一致
- 理由：譯者選擇與同 KR 的其他檔一致，不需動作。
- 查證：KR 相同的 BattleKeywords 單位 BattleKeywords-a1c10p2.json\|/dataList/15/flavor、BattleKeywords.json\|/dataList/583/flavor 現行譯文與本單位完全相同
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N613　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Bufs-a1c10p2.json|/dataList/17/flavor/dae0f993`　keywords，批次 20261101T092737-1dff83ff，來源 claude

- KR：`옷감이나 재목 따위를 치수에 맞게 베어 내기 위한 표식`
- 現行譯文：`為了將布料或木材等依尺寸裁切而做的標記`
- 註記：未照 hint：hint 引的 BattleKeywords-a1c10p2 舊譯文已在批次 20261101T092653-1dbe095a 依 hint 改成與 BattleKeywords.json／Bufs.json（KR 相同）一致的譯文，這裡也用該譯文，讓四個檔案一致
- 判讀：未照 hint：舊譯已在批次 20261101T092653-1dbe095a 改成與 BattleKeywords.json／Bufs.json 一致
- 理由：譯者選擇與同 KR 的其他檔一致，不需動作。
- 查證：KR 相同的 BattleKeywords 單位 BattleKeywords-a1c10p2.json\|/dataList/17/flavor、BattleKeywords.json\|/dataList/582/flavor 現行譯文與本單位完全相同
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N614　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Bufs-a1c10p2.json|/dataList/23/desc/143aebef`　keywords，批次 20261101T084625-34b4d879，來源 claude

- KR：`- 최댓값 : 2⏎- 받는 피해량 -10%⏎- 수치가 2면, 최종 위력 -2`
- 現行譯文：`- 最大值：2⏎- 受到的傷害-10%⏎- 本效果為2層時，最終威力-2`
- 註記：KR「수치가 2면」：寫「本效果為2層時」（BattleKeywords 既有「每有2層」）
- 判讀：「수치가 2면」寫「本效果為2層時」
- 理由：依 KR 語意的說明，不需動作。
- 查證：KR 相同的 BattleKeywords 單位 BattleKeywords-a1c10p2.json\|/dataList/23/desc 現行譯文與本單位完全相同
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N615　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Bufs-a1c9116.json|/dataList/7/desc/8123124e`　keywords，批次 20261101T092832-2fbe2a97，來源 claude

- KR：`최댓값 : 4⏎체력이 0이 되는 피해를 받으면, 80% 확률로 해당 피해를 받지 않고, 체력을 60 회복, 흐트러짐 해제하고 이 버프 소멸⏎⏎아래 상황에서는 사망 방지 및 회복 효과가 발동하지 않음⏎- 피격 시 코인 피해량이 자신의 최대 체력을 초과⏎- 체력 회복 감소 효과를 보유⏎턴 종료시 수치 1 감소`
- 現行譯文：`最大值：4⏎受到使體力降至0點的傷害時，有80%的機率不會受到傷害並使自身恢復60點體力，解除自身的混亂並解除本效果⏎⏎以下情況不會觸發避免陣亡與體力恢復效果⏎- 受到攻擊時硬幣造成的傷害高於自身體力上限⏎- 自身帶有體力恢復降低效果⏎回合結束時，本效果層數減少1層`
- 註記：照 BattleKeywords-a1c9116 的譯文，只拿掉「體力恢復降低 效果」中多餘的空格（base.md：KR 寫韓文狀態名時中文名前後不加空格）；BattleKeywords-a1c9116.json\|/dataList/7/desc 也該拿掉
- 判讀：照 BattleKeywords-a1c9116 譯文，拿掉「體力恢復降低 效果」的空格
- 理由：不需要動作。
- 查證：KR 相同的 BattleKeywords 單位 BattleKeywords-a1c9116.json\|/dataList/7/desc 現行譯文與本單位完全相同；譯文內沒有「數量＋狀態名」需要補空格之處。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N616　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Bufs-twth.json|/dataList/3/desc/565c10ab`　keywords，批次 20261101T092912-71578915，來源 claude

- KR：`- 약지 소속에게 받는 피해량 +(수치x5)%⏎· 대상이 약지 야수파 소속이면 효과가 강화되어, 대신 피해량 +(수치x10)%⏎- 전투 시작 시 (수치+1)만큼 출혈, 침잠 얻고 출혈 횟수, 침잠 횟수 증가⏎- 전투 종료 시 이 효과가 소멸되고 찢긴 색채 [루주], 찢긴 색채 [블뢰] 수치가 절반으로 감소 (소수점 버림)⏎- 최댓값 : 4`
- 現行譯文：`- 受到屬於環指單位的傷害+(本效果層數×5)%⏎．若攻擊者屬於環指野獸派，則改為受到傷害+(本效果層數×10)%⏎- 戰鬥開始時，使自身增加(本效果層數+1)級流血強度、(本效果層數+1)級沉淪強度、(本效果層數+1)回流血、(本效果層數+1)回沉淪⏎- 戰鬥結束時，解除本效果並使自身的被撕裂的色彩[赤紅]與被撕裂的色彩[青藍]層數減半(無條件捨去)⏎- 最大值：4`
- 註記：照 BattleKeywords-twth 的同條譯文，但依 base.md 拿掉中文狀態名後的空格（KR 無 [ID] 時中文名前後不加空格）；BattleKeywords-twth 該條也應一併修正
- 判讀：照 BattleKeywords-twth 同條譯文並去掉狀態名後的空格
- 理由：不需要動作。
- 查證：KR 相同的 BattleKeywords 單位 BattleKeywords-twth.json\|/dataList/3/desc 現行譯文與本單位完全相同；譯文內沒有「數量＋狀態名」需要補空格之處。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N617　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Bufs.json|/dataList/192/desc/4907b944`　keywords，批次 20261101T092919-ce37de65，來源 claude

- KR：`턴 시작 시 공격 위력 감소, 마비, 속박 중 무작위로 수치만큼 얻고 소멸`
- 現行譯文：`回合開始時，隨機施加虛弱、麻痺、束縛中的一種效果，其層數與本效果相同，隨後解除本效果`
- 註記：依 BattleKeywords.json 的譯文，但 KR 寫韓文狀態名（無 [ID]），依 base.md 官方名前後不加空格，刪去名稱後的空格；BattleKeywords.json 同一條宜一併改
- 判讀：依 BattleKeywords.json 同條譯文，刪去狀態名後的空格
- 理由：不需要動作。
- 查證：KR 相同的 BattleKeywords 單位 BattleKeywords.json\|/dataList/169/desc 現行譯文與本單位完全相同；譯文內沒有「數量＋狀態名」需要補空格之處。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N618　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Bufs.json|/dataList/192/summary/4907b944`　keywords，批次 20261101T092919-ce37de65，來源 claude

- KR：`턴 시작 시 공격 위력 감소, 마비, 속박 중 무작위로 수치만큼 얻고 소멸`
- 現行譯文：`回合開始時，隨機施加虛弱、麻痺、束縛中的一種效果，其層數與本效果相同，隨後解除本效果`
- 註記：依 BattleKeywords.json 的譯文，但 KR 寫韓文狀態名（無 [ID]），依 base.md 官方名前後不加空格，刪去名稱後的空格；BattleKeywords.json 同一條宜一併改
- 判讀：依 BattleKeywords.json 同條譯文，刪去狀態名後的空格
- 理由：不需要動作。
- 查證：KR 相同的 BattleKeywords 單位 BattleKeywords.json\|/dataList/169/summary 現行譯文與本單位完全相同；譯文內沒有「數量＋狀態名」需要補空格之處。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N619　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Bufs.json|/dataList/208/desc/4907b944`　keywords，批次 20261101T092919-ce37de65，來源 claude

- KR：`행동 불가⏎턴 종료 시 정신력 -20. 정신력이 -45가 된 경우 정신력을 최대로 회복하고, 다음 턴에 아찔한 정신 상태가 됨⏎2턴 동안 해제되지 않으면, 턴 종료 시 상태 해제`
- 現行譯文：`無法行動⏎回合結束時失去20點理智值。若理智值降為-45，則完全恢復理智值，並在下回合施加恍惚的意識⏎若本效果在2回合內沒有解除，則在回合結束時解除`
- 註記：依 BattleKeywords.json 的譯文，但 KR 寫韓文狀態名（無 [ID]），依 base.md 官方名前後不加空格，刪去名稱後的空格；BattleKeywords.json 同一條宜一併改
- 判讀：依 BattleKeywords.json 同條譯文，刪去狀態名後的空格
- 理由：不需要動作。
- 查證：KR 相同的 BattleKeywords 單位 BattleKeywords.json\|/dataList/189/desc、BattleKeywords.json\|/dataList/214/desc 現行譯文與本單位完全相同；譯文內沒有「數量＋狀態名」需要補空格之處。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N620　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Bufs.json|/dataList/229/desc/4907b944`　keywords，批次 20261101T092919-ce37de65，來源 claude

- KR：`행동 불가⏎턴 종료 시 정신력 -20. 정신력이 -45가 된 경우 정신력을 최대로 회복하고, 다음 턴에 아찔한 정신 상태가 됨⏎2턴 동안 해제되지 않으면, 턴 종료 시 상태 해제`
- 現行譯文：`無法行動⏎回合結束時失去20點理智值。若理智值降為-45，則完全恢復理智值，並在下回合施加恍惚的意識⏎若本效果在2回合內沒有解除，則在回合結束時解除`
- 註記：依 BattleKeywords.json 的譯文，但 KR 寫韓文狀態名（無 [ID]），依 base.md 官方名前後不加空格，刪去名稱後的空格；BattleKeywords.json 同一條宜一併改
- 判讀：依 BattleKeywords.json 同條譯文，刪去狀態名後的空格
- 理由：不需要動作。
- 查證：KR 相同的 BattleKeywords 單位 BattleKeywords.json\|/dataList/189/desc、BattleKeywords.json\|/dataList/214/desc 現行譯文與本單位完全相同；譯文內沒有「數量＋狀態名」需要補空格之處。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N621　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Bufs.json|/dataList/412/summary/f66049ca`　keywords，批次 20261101T092920-6ebafc37，來源 claude

- KR：`호흡 위력 얻음, 호흡 횟수 증가`
- 現行譯文：`獲得呼吸法強度與回數`
- 註記：依 base.md「KR 寫韓文狀態名、沒有 [ID] 時，官方中文名前後不加空格」去掉 BattleKeywords.json 譯文中狀態名旁多餘的空格，其餘照該譯文；BattleKeywords.json 那一條也應去掉
- 判讀：去掉 BattleKeywords.json 譯文中狀態名旁多餘的空格
- 理由：不需要動作。
- 查證：KR 相同的 BattleKeywords 單位 BattleKeywords.json\|/dataList/335/summary 現行譯文與本單位完全相同；譯文內沒有「數量＋狀態名」需要補空格之處。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N622　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Bufs.json|/dataList/580/summary/f66049ca`　keywords，批次 20261101T092920-6ebafc37，來源 claude

- KR：`조건 만족 시 보호 얻음`
- 現行譯文：`滿足條件時獲得守護`
- 註記：依 base.md「KR 寫韓文狀態名、沒有 [ID] 時，官方中文名前後不加空格」去掉 BattleKeywords.json 譯文中狀態名旁多餘的空格，其餘照該譯文；BattleKeywords.json 那一條也應去掉
- 判讀：去掉 BattleKeywords.json 譯文中狀態名旁多餘的空格
- 理由：不需要動作。
- 查證：KR 相同的 BattleKeywords 單位 BattleKeywords.json\|/dataList/496/summary 現行譯文與本單位完全相同；譯文內沒有「數量＋狀態名」需要補空格之處。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N623　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Bufs.json|/dataList/590/desc/f66049ca`　keywords，批次 20261101T092920-6ebafc37，來源 claude

- KR：`- 최댓값: 2⏎- 이 효과가 있는 턴 동안 ‘크리틱’을 사용할 수 없음⏎- 턴 시작시 정신력 5 회복⏎- 턴 종료시 1 감소⏎- 이 버프 소멸 시 (전장에 있는 스튜던트 인격 수)만큼 다음 턴에 경의를 얻음`
- 現行譯文：`- 最大值：2⏎- 帶有本效果的回合內，自身不能使用「批評」⏎- 回合開始時，使自身恢復5點理智值⏎- 回合結束時，本效果層數減少1層⏎- 本效果解除時，下回合使自身獲得層數等同於場上屬於環指 學徒的人格數的敬意`
- 註記：依 base.md「KR 寫韓文狀態名、沒有 [ID] 時，官方中文名前後不加空格」去掉 BattleKeywords.json 譯文中狀態名旁多餘的空格，其餘照該譯文；BattleKeywords.json 那一條也應去掉
- 判讀：去掉 BattleKeywords.json 譯文中狀態名旁多餘的空格
- 理由：不需要動作。
- 查證：KR 相同的 BattleKeywords 單位 BattleKeywords.json\|/dataList/506/desc 現行譯文與本單位完全相同；譯文內沒有「數量＋狀態名」需要補空格之處。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N624　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/Bufs.json|/dataList/652/desc/f34b7925`　keywords，批次 20261101T092920-6ebafc37，來源 claude

- KR：`- 최댓값 : 4⏎- 수치당 아래 효과 얻음⏎· 1 ~ : 공격 레벨 +1⏎· 2 ~ : 방어 레벨 +2⏎· 3 ~ : 자신의 스킬로 얻는 충전 횟수 +1⏎· 4 ~ : 자신의 스킬 피해량 +(충전 위력 x 4)% (최대 20%)⏎⏎- 이 효과의 최대치를 초과하여 얻으면, 초과값만큼 다음 턴에 얻음⏎- 턴 종료시 1 감소`
- 現行譯文：`- 最大值：4⏎- 每有1層本效果，則獲得以下效果⏎．不低於1層：攻擊等級+1⏎．不低於2層：防禦等級+2⏎．不低於3層：使自身技能獲得的充能回數額外+1⏎．不低於4層：使自身技能造成的傷害+(自身的充能強度×4)%(最多+20%)⏎⏎- 若獲得本效果後的層數高於最大值，則高於最大值的層數將在下回合獲得⏎- 回合結束時，使本效果層數減少1層`
- 註記：KR「최대치를 초과하여」依比較用語寫「高於最大值」；「1 ~」「2 ~」是「1層以上」，保留「不低於1層」
- 判讀：比較用語「초과」寫「高於最大值」；「1 ~」保留「不低於1層」
- 理由：依 base.md 比較用語規則（초과＝高於、이상＝不低於），不需動作。
- 查證：KR 相同的 BattleKeywords 單位 BattleKeywords.json\|/dataList/568/desc 現行譯文與本單位完全相同
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N625　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/ActionEvents-a1c8p2.json|/dataList/11/desc/aa118e86`　mirror，批次 20261101T082234-58a217c9，來源 claude

- KR：`<i>과거와 미래는 지혜롭고 어질어도 깨뜨릴 수 없고,⏎인연과 운명은 가까운 이도 만날 수 없게 한다.</i>⏎⏎’지하까지 도달한 후보자 여러분 환영합니다!’⏎⏎‘이곳에선 동전을 모을 기회가 지상보다 더 많이 주어진다는 사실. 알고 계신가요?’⏎⏎‘예를 들면… 지금, 이 장소 아래에도 동전이 숨겨져 있죠.’ ⏎⏎‘위험이 도사리는 아래로 내려가, 동전을 쟁취할 후보자는 과연 누구일까요!’⏎⏎언뜻 보기에도 위험해 보이는 폐허의 깊고 어두운 아래.⏎⏎동전이 필요한 건 맞지만, 자칫하면 큰 부상을 당하거나 죽을 수도 있을 것 같다.⏎⏎어떻게 하면 좋을까?`
- 現行譯文：`<i>過去未來，莫謂智賢能打破；⏎前因後果，須知親近不相逢。</i>⏎⏎「歡迎來到地下的各位候選人！」⏎⏎「你們知道嗎？在這裡，收集銅錢的機會比在地面上更多。」⏎⏎「比方說……現在，諸位的腳下也藏著一文銅錢。」⏎⏎「向下進入險象環生之地，最終奪得銅錢的候選人究竟會是誰呢！」⏎⏎一看便知是危機四伏的廢墟，又深又暗的下層區域。⏎⏎雖然確實需要銅錢，但稍有不慎便有可能受重傷乃至死亡。⏎⏎該怎麼做呢？`
- 註記：麽→麼
- 判讀：修正 draft／hint 的簡體或異體字（麽→麼 等）
- 理由：現行譯文已是正體用字，不需動作。
- 查證：現行譯文已搜尋「麽、里面、乾凈、痴、閑愁、哢」，皆無。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N626　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/ActionEvents-a1c8p2.json|/dataList/11/options/1/result/0/aa118e86`　mirror，批次 20261101T082234-58a217c9，來源 claude

- KR：`동전도 중요하지만, 수감자의 목숨을 벌써부터 마구잡이로 쓰기엔 불확실한 요소가 너무 많다.⏎이번에는 자리를 피하기로 했다.`
- 現行譯文：`銅錢固然重要，但我不想這麼早就開始揮霍罪人的性命，不確定因素實在太多了。⏎這次我決定避免這麼做。`
- 註記：麽→麼
- 判讀：修正 draft／hint 的簡體或異體字（麽→麼）
- 理由：現行譯文已是正體用字，不需動作。
- 查證：現行譯文已搜尋「麽」，無。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N627　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/ActionEvents-a1c8p2.json|/dataList/13/desc/4c6bc67a`　mirror，批次 20261101T082234-58a217c9，來源 claude

- KR：`방 한가운데에 항아리들이 모여있다.⏎설반이 미처 쓰지 못한 항아리일까?⏎⏎그중 뚜껑이 열려 있는 항아리 하나가 유독 눈에 띈다.⏎그 안에서 무언가 소리가 들리는 것 같다.⏎⏎항아리 안에 누가 갇혀 있는 걸지도 모른다.⏎하지만 쉽사리 건드릴 만한 느낌은 아니다.⏎⏎어떻게 할까?`
- 現行譯文：`房間中央有許多甕放在一起。⏎這是薛蟠還沒來得及使用的甕嗎？⏎⏎其中一個開著口的甕尤其引人注目。⏎似乎從裡面傳來了什麼聲音。⏎⏎也許有人被困在了甕中。⏎但感覺這不是可以輕易觸碰的東西。⏎⏎該如何抉擇？`
- 註記：麽→麼；里面→裡面
- 判讀：修正 draft／hint 的簡體或異體字（麽→麼、里面→裡面）
- 理由：現行譯文已是正體用字，不需動作。
- 查證：現行譯文已搜尋「麽、里面」，皆無。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N628　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/ActionEvents-a1c8p2.json|/dataList/14/desc/f7d02bdb`　mirror，批次 20261101T082234-58a217c9，來源 claude

- KR：`<i>원한은 서로 되갚으면 끝이 없어 무겁고,⏎이별과 재회는 모두 정해진 운명이니,⏎목숨의 길고 짧음은 전생의 업이다.</i>⏎⏎<i>노년의 부귀는 드문 요행이라,⏎깨달은 자는 출가의 길에 오르고,⏎미혹된 자는 부질없이 생을 허비하는구나.</i>⏎⏎<i>모이 다 쪼아 먹은 새들은 숲으로 날아갔으니,⏎흰 눈 덮인 들판은 발자국 하나 없이 맑고 깨끗하리라.</i>⏎⏎‘동전을 잃어 아쉬운 후보자가 보이는군요?’⏎‘관중 분들이 기회를 주고 싶다고 하십니다!’⏎‘그렇다면… 동전을 얻을 수 있는 목인을 여러분에게 보내드려야겠군요?’`
- 現行譯文：`<i>冤冤相報自非輕，⏎分離聚合皆前定。⏎欲知命短問前生，</i>⏎⏎<i>老來富貴也真僥倖。⏎看破的，遁入空門；⏎癡迷的，枉送了性命。</i>⏎⏎<i>——好一似食盡鳥投林，⏎落了片白茫茫大地真乾淨！</i>⏎⏎「看到有候選人因丟了銅錢而遺憾？」⏎「觀眾們說想給他們一個機會！」⏎「那麼……是不是得給諸位送來一些能從中獲得銅錢的木人呢？」`
- 註記：麽→麼；乾凈→乾淨；痴迷→癡迷；依 KR 的分段，把「老來富貴也真僥倖」移到第二個 <i> 段
- 判讀：修正異體字（麽→麼、乾凈→乾淨、痴→癡），並依 KR 分段調整詩句位置
- 理由：現行譯文已是正體用字，不需動作。
- 查證：現行譯文已搜尋「麽、乾凈、痴」，皆無。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N629　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/ActionEvents-a1c8p2.json|/dataList/16/desc/bfab4ea8`　mirror，批次 20261101T082234-58a217c9，來源 claude

- KR：`<i>봄날의 꿈은 구름 따라 흩어지고,⏎흩날린 꽃잎은 흐르는 물에 떠내려갔으니,⏎모든 아들과 딸들에게 물음을 남기노라.⏎부질없는 근심, 어찌 찾아 헤매는가?</i>⏎⏎방에 들어가자 다른 후보자의 모습이 보였다.⏎그 순간, 방이 닫히면서 사회자의 목소리가 들려왔다.⏎⏎‘상대 후보자를 처치하여 승리한 후보자에게 좋은 기회가 주어지는 방으로 갈 기회를 드리겠습니다!’⏎⏎‘하지만… 그냥 싸우기만 해서는 바깥에서 싸우는 것과 크게 다르지 않겠죠?’⏎⏎덜컹하는 소리와 함께 수상한 연기가 바닥에서 새어나오기 시작했다.⏎연기를 들이마신 수감자들의 표정이 굳어지며, 여기저기서 기침 소리가 터져 나왔다.⏎…위험한 상황일지도.`
- 現行譯文：`<i>春夢隨雲散，⏎飛花逐水流；⏎寄言眾兒女：⏎何必覓閒愁。</i>⏎⏎進入房間後，我們看見了另一位候選人。⏎就在下一瞬，房門砰然關閉，主持人的聲音隨之響起：⏎⏎「擊敗對手並獲得勝利的候選人，將有機會前往蘊含更多機遇的房間！」⏎⏎「不過……如果只是普通地戰鬥，那就和在外面戰鬥沒什麼區別了，對吧？」⏎⏎伴隨著喀噹聲，奇怪的煙霧從地板縫隙間悄然冒出。⏎吸入煙霧的罪人們表情逐漸僵硬，房間四處傳來劇烈的咳嗽聲。⏎……情況似乎不妙。`
- 註記：麽→麼；閑愁→閒愁；哢哢聲→喀噹聲
- 判讀：修正異體字（麽→麼、閑愁→閒愁、哢哢聲→喀噹聲）
- 理由：現行譯文已是正體用字，不需動作。
- 查證：現行譯文已搜尋「麽、閑愁、哢」，皆無。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N630　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/ActionEvents-a1c8p2.json|/dataList/19/desc/aa118e86`　mirror，批次 20261101T082234-58a217c9，來源 claude

- KR：`꼭 동전을 줄 이유가 있을까?⏎어차피 다시 싸워서 이기면 동전을 되찾을 수 있다.⏎아니, 그뿐만 아니라 상대 후보자의 동전까지 얻을 수 있겠지.⏎⏎하지만… 그래도 되는 걸까?`
- 現行譯文：`真的有拿出銅錢的必要嗎？⏎反正只要再將其戰勝，便能拿回銅錢。⏎不僅如此，甚至還能得到對方候選人的銅錢。⏎⏎但是……真的可以這麼做嗎？`
- 註記：麽→麼
- 判讀：修正 draft／hint 的簡體或異體字（麽→麼）
- 理由：現行譯文已是正體用字，不需動作。
- 查證：現行譯文已搜尋「麽」，無。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N631　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/ActionEvents-pilgrimage.json|/dataList/8/title/f7d7bd99`　mirror，批次 20261101T082238-8186bdf8，來源 claude

- KR：`등대지기, 고요함(대침묵)에 대한 이야기`
- 現行譯文：`燈塔管理員與大靜謐的故事`
- 註記：「고요함(대침묵)」兩者中文皆為「大靜謐」，不重複（同 AbEvents-pilgrimage）
- 判讀：「고요함(대침묵)」兩者中文皆為「大靜謐」，不重複（同 AbEvents-pilgrimage）
- 理由：與同系列既有寫法一致，不需動作。
- 查證：AbEvents-pilgrimage#6、ActionEvents-pilgrimage#9 等的「大靜謐」寫法相同，ActionEvents-pilgrimage#5、#6 內文亦為「大靜謐」。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N632　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/ActionEvents-walpu6.json|/dataList/2/desc/ca405895`　mirror，批次 20261101T092451-cde2257d，來源 claude

- KR：`중앙본부 사무직 책상 위에 보고서가 놓여 있다.⏎⏎<DAY ■■, ■■ ■■ 연구 용역 결과 보고서>⏎⏎몇 군데 검열되어 있었지만 내용은, 새로운 보호장 탄환에 대한 것인 것 같았다.⏎⏎'트라우마 방지장은 외부에서 오는 정신 피해를 ■■ ■■, 또는 ■■■ 방지함.'⏎⏎'보호장 탄환 ■■이 ■■할 시, 관리자 또는 ■■■에게 문의할 것.'⏎⏎무언가 잔뜩 지워진 보고서 옆에는 '트라우마 방지장 샘플 및 설명서'이라고 적힌 박스가 있다.`
- 現行譯文：`中央本部的文職辦公桌上放著一份報告書。⏎⏎<DAY ■■，■■ ■■ 研究業務結果報告書>⏎⏎雖然有幾個地方經過審查，但內容似乎是有關新型的保護力場子彈。⏎⏎「創傷防止力場可■■ ■■或■■■防止來自外部的精神傷害。」⏎⏎「保護力場子彈■■■■時，請諮詢主管或是■■■。」⏎⏎內容被大量塗改的報告書旁，有一個寫著「創傷防止力場樣品及說明書」的箱子。`
- 註記：報告標題改譯為「<DAY ■■，■■ ■■ 研究業務結果報告書>」，保留 DAY 與遮蔽字；補上 KR 的「似乎」（것 같았다），依 KR 語序調整遮蔽字位置
- 判讀：報告標題保留 DAY 與遮蔽字，補上「似乎」，依 KR 語序調整遮蔽字
- 理由：格式與語氣處理的說明，不需動作。
- 查證：現行譯文含「<DAY ■■，■■ ■■ 研究業務結果報告書>」與「似乎」。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N633　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/ActionEvents_Mirror6.json|/dataList/6/options/1/result/0/87de348a`　mirror，批次 20261101T095641-f2ddff9d，來源 claude

- KR：`"요새 T사에도 골치 아픈 사건들이 많아서요. 외부인에게도 의뢰를 부탁하고 있습니다."⏎⏎"마침 유로지비가 이 곳에서 난동을 부리겠다는 성명을 냈죠. 그들을 격퇴하면, 허가원 없이 배지를 드리겠습니다."그 말이 끝마쳐지자 마자, 인상적인 스카프를 두른 자들이 하나 둘 나타나기 시작했다.`
- 現行譯文：`「最近T公司也有不少棘手的事件。所以也在委託外部人士處理。」⏎⏎「正好聖愚發表了聲明，說要在這裡鬧事。能擊退他們的話，即使沒有申請書，也能授予各位徽章。」話音剛落，圍著令人印象深刻圍巾的人們便一個接一個地出現了。`
- 註記：prev_zh 把「유로지비가…성명을 냈죠」（聖愚發表聲明要在此作亂）誤譯為「發布了…委託」，已依 KR 改正
- 判讀：prev_zh 把「유로지비가…성명을 냈죠」誤譯成「發布了…委託」，已依 KR 改正
- 理由：譯文已更正，不需動作。
- 查證：現行譯文為「聖愚發表了聲明，說要在這裡鬧事」。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N634　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/EGOgift_MirrorDungeon-EventTheme_2.json|/dataList/27/desc/f15a6123`　mirror，批次 20261101T082317-8eec3e2b，來源 claude

- KR：`아군이 턴 종료 시 피해를 입힌 공격 스킬 수에 비례하여 다음 턴에 [Breath] 위력 2 획득.⏎- 집중 전투 방식의 스테이지에서는 효과가 강화되어 [Breath] 위력 4 획득.⏎⏎턴 시작 시, 체력이 가장 낮은 아군에게 호흡 2, [Breath] 횟수 2, [Protection] 2 부여.⏎⏎아군이 본인의 정신력에 비례하여 적에게 입히는 피해량 +(정신력 절댓값. 최소 0)%⏎⏎더하기 코인 스킬을 보유한 아군이 흐트러짐 상태에서 풀려날 때 정신력 최대로 회복 (침식 상태일 경우에는 발동하지 않음).⏎⏎모든 스킬이 빼기 코인인 아군의 경우에는 턴 시작 시 [Paralysis] 5 얻음`
- 現行譯文：`回合結束時，友方單位每有1個攻擊技能造成了傷害，下回合使其獲得2級 [Breath] 強度。⏎- 若是集中遭遇戰，則改為獲得4級 [Breath] 強度。⏎⏎回合開始時，使現存體力最低的1名友方單位獲得2級呼吸法強度、2回 [Breath] 與2層 [Protection]。⏎⏎友方單位造成的傷害+(該單位理智值的絕對值)%(最少+0%)⏎⏎當擁有加算硬幣技能的友方單位解除混亂時，使其恢復全部理智值(不對侵蝕的單位生效)。⏎⏎回合開始時，對擁有的技能全部為減算硬幣技能的友方單位施加5層 [Paralysis]`
- 註記：KR 第一個「호흡」是韓文、無 [ID]，改寫官方名「呼吸法」
- 判讀：譯者說明依 KR／規則修正草稿（標籤、數量、條件或官方名），已處理完畢
- 理由：譯者說明依 KR 修正草稿，譯文已照 KR 處理，不需動作。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N635　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/EGOgift_MirrorDungeon-EventTheme_2.json|/dataList/33/desc/54162278`　mirror，批次 20261101T082317-8eec3e2b，來源 claude

- KR：`[Bullet]을 소모할 수 있는 스킬에서 소모하지 않을 경우, 이번 턴과 다음 턴에 [Enhancement] 1 얻음 (턴 당 1회)⏎⏎[Bullet]을 얻으면, 이번 턴과 다음 턴에 사용하는 스킬 3의 피해량 +50%⏎⏎[Bullet]을 소모하는 스킬 사용 시, [Breath] 3 [Breath] 횟수 1 얻음`
- 現行譯文：`若可消耗 [Bullet] 的技能未進行消耗，則本回合與下回合使自身獲得1層 [Enhancement] (每回合最多1次)⏎⏎獲得 [Bullet] 時，本回合與下回合使3技能造成的傷害+50%⏎⏎使用消耗 [Bullet] 的技能時，使自身獲得3級 [Breath] 強度與1回 [Breath]`
- 註記：draft 多一個 [Bullet] 已拿掉
- 判讀：譯者說明依 KR／規則修正草稿（標籤、數量、條件或官方名），已處理完畢
- 理由：譯者說明依 KR 修正草稿，譯文已照 KR 處理，不需動作。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N636　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/EGOgift_MirrorDungeon-EventTheme_2.json|/dataList/35/desc/6922d9b1`　mirror，批次 20261101T082317-8eec3e2b，來源 claude

- KR：`[Burst] 횟수를 부여하는 기본 공격 스킬로 적과 합 승리 시, 대상의 [Burst] 횟수 3 증가. (적 별로 턴 당 2회)⏎⏎적중 시 대상 적에게 [Burst]이 없으면, [Burst] 위력을 부여하는 코인 효과가 [Burst] 위력을 부여하는 대신 [Burst] 횟수를 부여 (최대 4, 적 별로 턴 당 2회, 환상체일 경우 부위 별로 계산)⏎⏎[Burst] 위력 또는 [Burst] 횟수를 부여하는 기본 공격 스킬이 각 스킬의 보유량만큼 공격 레벨이 증가 (최대 3).⏎- 해당 스킬의 보유량이 3개 이상인 경우, 공격 레벨이 2만큼 추가로 증가.`
- 現行譯文：`增加 [Burst] 回數的基礎攻擊技能與敵方單位拚點勝利時，使其增加3回 [Burst] (每回合每名敵方單位最多2次)⏎⏎命中時，若目標敵方單位未帶有 [Burst]，則增加 [Burst] 強度的硬幣效果不增加 [Burst] 強度，改為增加 [Burst] 回數(最多4回，每回合每名敵方單位最多2次，若是異想體則改為每個部位)⏎⏎增加 [Burst] 強度或 [Burst] 回數的基礎攻擊技能增加等同於其擁有量的攻擊等級(最多3級)⏎- 若該技能擁有量不低於3個，則額外增加2級攻擊等級`
- 註記：補足 KR 的 [Burst] 數量（「[Burst] 위력을 부여하는 대신」）
- 判讀：譯者說明依 KR／規則修正草稿（標籤、數量、條件或官方名），已處理完畢
- 理由：譯者說明依 KR 修正草稿，譯文已照 KR 處理，不需動作。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N637　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/EGOgift_MirrorDungeon-EventTheme_2.json|/dataList/54/desc/52b30603`　mirror，批次 20261101T082317-8eec3e2b，來源 claude

- KR：`턴 시작 시, 중지 소속 인격이 2인 이상일 때 발동 (출격 인원을 기준으로 함)⏎⏎같은 소속의 인격이 피격 시, 같은 소속의 인원들 타격 피해량 +5% (턴 당 최대 15%)⏎중지 소속 인격일 경우, 효과를 대신하여 공격 레벨 증가 1 얻음, 타격 피해량 +10% (턴 당 최대 3, 턴 당 최대 30%)⏎⏎타격 스킬의 기본 위력 +1⏎- 중지 소속 인격일 경우 효과를 대신하여, 기본 위력 +1, 코인 위력 +1⏎- 중지 소속 인격일 경우 대상이 [RetaliationBook]이면 효과를 대신하여, 기본 위력 +1, 코인 위력 +1, 피해량 +30%⏎⏎중지 인격의 경우, 전투 동안 속도 최솟값 +1, 속도 최댓값 +1`
- 現行譯文：`回合開始時，若屬於中指的友方單位不低於2名，則觸發該效果(以參戰人員為基準)⏎⏎友方單位受到攻擊時，使與該友方單位屬於同一陣營的全體友方單位造成的打擊傷害+5%(每回合最多+15%)⏎屬於中指的友方單位改為獲得1層攻擊等級提升並使造成的打擊傷害+10%(每回合最多3次，每回合最多+30%)⏎⏎打擊技能的基礎威力+1⏎- 屬於中指的友方單位改為基礎威力+1、硬幣威力+1⏎- 屬於中指的友方單位的主要目標帶有 [RetaliationBook] 時，改為基礎威力+1、硬幣威力+1、造成的傷害+30%⏎⏎屬於中指的友方單位在本場戰鬥中的最小速度值+1、最大速度值+1`
- 註記：KR「공격 레벨 증가」無 [ID]，寫官方名「攻擊等級提升」；draft 的「+10%」與 KR「+30%」不符，已改正
- 判讀：譯者說明依 KR／規則修正草稿（標籤、數量、條件或官方名），已處理完畢
- 理由：譯者說明依 KR 修正草稿，譯文已照 KR 處理，不需動作。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N638　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/EGOgift_MirrorDungeon-EventTheme_2.json|/dataList/55/desc/d20e93a2`　mirror，批次 20261101T082317-8eec3e2b，來源 claude

- KR：`턴 시작 시, [Burst] 위력 또는 [Burst] 횟수를 부여하는 공격 스킬을 보유한 인격이 5인 이상이면, 이번 전투 동안 발동 (E.G.O 스킬 제외. 대기 인원 제외)⏎⏎자신의 공격 스킬로 [Burst]을 가진 대상을 처치할 시 (집중 전투의 경우 부위 파괴), 대상의 [Burst] 위력을 나누어 모든 적 (집중 전투일 경우 부위)에게 부여함 (최대 3씩)⏎⏎자신의 공격 스킬로 [BurstAgility]을 가진 대상을 처치할 시 (집중 전투의 경우 부위 파괴), 위 효과를 대신하여 대상의 [Burst] 위력과 횟수를 나누어 모든 적 (집중 전투일 경우 부위)에게 부여함 (최대 6씩)⏎⏎[Burst]을 가진 대상과 합을 할 때, (대상의 [Burst] / 20)만큼 합 위력이 증가 (최대 2)⏎- [BurstAgility]을 가진 대상과 합을 할 때, 위 효과를 대신하여 (대상의 [Burst] / 10)만큼 코인 위력이 증가 (최대 2)⏎⏎[BurstAgility] 효과로 피해를 줄 때, ([BurstAgility] 피해 / 2)만큼 고정 피해 입힘`
- 現行譯文：`回合開始時，若擁有增加 [Burst] 強度或 [Burst] 回數的攻擊技能的友方單位不低於5名，則在本場戰鬥期間觸發該效果。(不包括E.G.O技能、不包括待命人員)⏎⏎自身的攻擊技能擊殺帶有 [Burst] 的目標(若是集中遭遇戰，則改為破壞部位)時，將目標的 [Burst] 強度平均施加給全體敵方單位(若是集中遭遇戰，則改為所有部位)(每名敵方單位最多3級)⏎⏎自身的攻擊技能擊殺帶有 [BurstAgility] 的目標(若是集中遭遇戰，則改為破壞部位)時，將目標的 [Burst] 強度與回數平均施加給全體敵方單位(若是集中遭遇戰，則改為所有部位)(每名敵方單位最多6點)⏎⏎與帶有 [Burst] 的目標拚點時，使自身的拚點威力+(目標的 [Burst] 強度÷20)(最多+2)⏎- 與帶有 [BurstAgility] 的目標拚點時，改為使自身的硬幣威力+(目標的 [Burst] 強度÷10)(最多+2)⏎⏎透過 [BurstAgility] 效果造成傷害時，對目標造成( [BurstAgility] 傷害量÷2)點固定傷害`
- 註記：draft 多一個 [Burst] 已拿掉
- 判讀：譯者說明依 KR／規則修正草稿（標籤、數量、條件或官方名），已處理完畢
- 理由：譯者說明依 KR 修正草稿，譯文已照 KR 處理，不需動作。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N639　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/EGOgift_MirrorDungeon-StoryTheme.json|/dataList/17/simpleDesc/2/simpleDesc/dfe5805b`　mirror，批次 20261101T082336-cebebfb8，來源 claude

- KR：`흐트러진 대상은 보유한 진동 위력이 2배로 증가`
- 現行譯文：`使陷入混亂的目標所帶有的震顫強度變為2倍`
- 註記：KR 用韓文「진동」、沒有 [ID]，依 base.md 寫官方名「震顫」
- 判讀：譯者說明依 KR／規則修正草稿（標籤、數量、條件或官方名），已處理完畢
- 理由：譯者說明依 KR 修正草稿，譯文已照 KR 處理，不需動作。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N640　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/EGOgift_MirrorDungeon-StoryTheme.json|/dataList/9/desc/dbe43b3e`　mirror，批次 20261101T082336-cebebfb8，來源 claude

- KR：`아군이 [Laceration] 위력, [Laceration] 횟수 또는 특수 출혈을 부여하는 스킬로 적과 합하여 승리 시, [NailPersonality] 2 ~ 3 부여.⏎⏎N사 광신도 소속 인격이 적을 공격할 때 피해량 +25%, 턴 시작 시 광신 1 얻음. 해당 소속 인격의 정신력이 최대이거나 이전 턴에 색욕 공명을 3 이상 했으면, 해당 인격이 [AttackDmgUp] 3을 추가로 얻음.⏎⏎N사 광신도 소속 인격의 최종 위력 +2, 더하기 코인 위력 +1, 빼기 코인 위력 -1, 공격 스킬의 [NailPersonality] 부여량 +1.⏎⏎N사 광신도 소속 인격이 <기계 융화 생명체>인 적에게 가하는 기본 스킬 피해량 +50%`
- 現行譯文：`友方單位使用增加 [Laceration] 強度、[Laceration] 回數或施加特殊流血的技能與敵方單位拚點勝利時，對目標施加2～3層 [NailPersonality]。⏎⏎屬於N公司狂信徒的人格攻擊敵方單位時造成的傷害+25%，並在回合開始時獲得1層狂信。若該人格的理智值為最大值，或上回合達成3個以上的色慾共鳴，則額外使其獲得3層 [AttackDmgUp]。⏎⏎屬於N公司狂信徒的人格獲得最終威力+2、加算硬幣威力+1、減算硬幣威力-1，攻擊技能施加的 [NailPersonality] 層數額外+1層。⏎⏎屬於N公司狂信徒的人格使用基礎技能對<機械融合生命體>敵方單位造成的傷害+50%`
- 註記：草稿多出 KR 沒有的「對帶有 [Laceration] 的目標」，已刪除；KR 韓文狀態名「광신」寫官方名「狂信」不加空格
- 判讀：譯者說明依 KR／規則修正草稿（標籤、數量、條件或官方名），已處理完畢
- 理由：譯者說明依 KR 修正草稿，譯文已照 KR 處理，不需動作。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N641　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/EGOgift_MirrorDungeon-mowe.json|/dataList/10/desc/2a2502dd`　mirror，批次 20261101T082400-712cf886，來源 claude

- KR：`턴 시작 시, [Charge] 횟수 또는 특수 충전을 획득하는 공격 스킬을 보유한 인격이 5인 이상이면, 이번 전투 동안 발동 (E.G.O 스킬 제외. 대기 인원 제외)⏎⏎[Charge] 횟수 또는특수 충전(위력 고정)을 소모하는 더하기 코인 기본 공격 스킬의 코인 위력 +1, 피해량 +(해당 스킬로 소모한 [Charge] 횟수와 특수 충전(위력 고정)의 합 x 4)% (최대 40%).⏎- 빼기 코인 기본 공격 스킬일 경우, 코인 위력 대신 기본 위력 +(4/코인 수) (최소 1, 소수점 버림)⏎⏎[Charge] 횟수 또는 특수 충전을 보유한 아군이 기본 공격 스킬로 합 진행 시 합 위력 +1, 턴 시작 시 이번 전투 동안 해당 합에서 승리한 횟수만큼 [AttackUp] 2 얻음 (최대 6).⏎⏎전투 중 사망에 이르는 피해를 받을 때, 보유한 [Charge] 횟수와 특수 충전(위력 고정)을 전부 소모하고 최대 체력의 (10 + 소모한 충전 횟수와 특수 충전(위력 고정)의 합)%만큼 즉시 회복 (전투당 1회), 사망 전까지 [Protection] 3 얻음.⏎⏎※ 위력과 횟수를 함께 보유한 특수 충전은 충전 횟수로 계산됨`
- 現行譯文：`回合開始時，若帶有獲得 [Charge] 回數或特殊充能攻擊技能的友方單位不低於5名，則在本場戰鬥期間觸發該效果。(不包括E.G.O技能、不包括待命人員)⏎⏎消耗 [Charge] 回數或特殊充能(固定強度)的加算硬幣基礎攻擊技能獲得硬幣威力+1、造成的傷害+(該技能消耗的 [Charge] 回數與特殊充能(固定強度)之和×4)%(最多+40%)⏎- 若為減算硬幣的基礎攻擊技能，則使硬幣威力改為基礎威力+(4÷技能硬幣數)(最少+1，無條件捨去)⏎⏎帶有 [Charge] 回數或特殊充能的友方單位基礎攻擊技能拚點時，使其拚點威力+1。回合開始時，本場戰鬥中每拚點勝利1次，使其獲得2層 [AttackUp]。(最多6層)⏎⏎本場戰鬥中，友方單位受到致死傷害時，消耗其所有 [Charge] 回數與特殊充能(固定強度)並使其立即恢復體力上限(10+消耗的充能回數與特殊充能(固定強度)之和)%的體力(每場戰鬥最多1次)。使該友方單位獲得3層 [Protection] 直到其陣亡。⏎⏎※ 擁有強度與回數的特殊充能以其回數計算`
- 註記：draft 多一個 [Charge] 已拿掉（KR 該處寫韓文「충전」、無 [ID]，寫官方名「充能」）
- 判讀：譯者說明依 KR／規則修正草稿（標籤、數量、條件或官方名），已處理完畢
- 理由：譯者說明依 KR 修正草稿，譯文已照 KR 處理，不需動作。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N642　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/EGOgift_MirrorDungeon-mowe.json|/dataList/11/desc/be5960e8`　mirror，批次 20261101T082400-712cf886，來源 claude

- KR：`턴 시작 시, [Charge] 횟수 또는 특수 충전을 획득하는 공격 스킬을 보유한 인격이 5인 이상이면, 이번 전투 동안 발동 (E.G.O 스킬 제외. 대기 인원 제외)⏎⏎[Charge] 횟수 또는 특수 충전을 소모하는 더하기 코인 기본 공격 스킬의 코인 위력 +1, 피해량 +(해당 스킬로 소모한 [Charge] 횟수와 특수 충전의 합 x 4)% (최대 40%).⏎- 빼기 코인 기본 공격 스킬일 경우, 코인 위력 대신 기본 위력 +(4/코인 수) (최소 1, 소수점 버림)⏎⏎보유한 [Charge] 횟수와 특수 충전의 합이 15 이상인 상태에서 [Charge] 횟수 또는 특수 충전을 소모하는 스킬 3을 사용할 경우, 공격 종료 시에 [Charge] 횟수 또는 특수 충전이 남아있다면 이를 전부 소모하여 입힌 피해의 (소모한 [Charge] 횟수와 특수 충전의 합 x 7.5)%만큼 추가 고정 피해를 입힘 (최대 75%).⏎⏎[Charge] 횟수 최대치 +5.`
- 現行譯文：`回合開始時，若帶有獲得 [Charge] 回數或特殊充能的攻擊技能的友方單位不低於5名，則在本場戰鬥期間觸發該效果。(不包括E.G.O技能、不包括待命人員)⏎⏎消耗 [Charge] 回數或特殊充能的加算硬幣基礎攻擊技能硬幣威力+1、造成的傷害+(該技能消耗的 [Charge] 回數與特殊充能之和×4)% (最多+40%)⏎- 若為減算硬幣的基礎攻擊技能，則使硬幣威力改為基礎威力+(4÷該技能硬幣數)(最少+1，無條件捨去)⏎⏎[Charge] 回數與特殊充能之和不低於15點的友方單位使用消耗 [Charge] 回數或特殊充能的3技能時，若攻擊後有剩餘的 [Charge] 回數或特殊充能，則消耗所有剩餘的回數與特殊充能並對目標額外造成(消耗的 [Charge] 回數與特殊充能之和×7.5)%的固定傷害。(最多75%)⏎⏎[Charge] 回數的最大值+5。`
- 註記：draft 多一個 [Charge] 已拿掉
- 判讀：譯者說明依 KR／規則修正草稿（標籤、數量、條件或官方名），已處理完畢
- 理由：譯者說明依 KR 修正草稿，譯文已照 KR 處理，不需動作。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N643　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/EGOgift_MirrorDungeon-mowe.json|/dataList/5/desc/2a2502dd`　mirror，批次 20261101T082400-712cf886，來源 claude

- KR：`[Charge] 횟수 또는 특수 충전을 보유한 아군이 합 패배 시, [Charge] 횟수와 특수 충전의 합만큼 보호막을 얻음 (턴 당 1회).⏎⏎체력이 절반 이하인 경우 효과가 강화되어, (충전 횟수와 특수 충전의 합 x 2)만큼 보호막을 얻음 (턴 당 1회)⏎⏎멀티크랙 사무소 소속 인격이 이번 전투 동안 체력 피해를 입었다면, 다음 턴에 이전 턴 동안 잃은 체력의 33%만큼 회복 (버림하여 처리)`
- 現行譯文：`帶有 [Charge] 回數或特殊充能的友方單位拚點失敗時，使其獲得等同於其 [Charge] 回數與特殊充能之和的護盾。(每回合最多1次)⏎⏎若該友方單位現存體力不高於50%，則改為獲得(其充能回數與特殊充能之和×2)點護盾。(每回合最多1次)⏎⏎本場戰鬥中，屬於多裂紋事務所的友方單位受到體力傷害時，下回合使其恢復上回合失去體力33%的體力(無條件捨去)`
- 註記：draft 多一個 [Charge] 已拿掉（KR 該處寫韓文「충전」、無 [ID]，寫官方名「充能」）
- 判讀：譯者說明依 KR／規則修正草稿（標籤、數量、條件或官方名），已處理完畢
- 理由：譯者說明依 KR 修正草稿，譯文已照 KR 處理，不需動作。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N644　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/EGOgift_MirrorDungeon.json|/dataList/115/desc/6630c90c`　mirror，批次 20261101T082401-24537171，來源 claude

- KR：`턴 시작 시, 적 전체(환상체일 경우, 모든 부위)에게 [Burst] 위력 3, [Burst] 횟수 3 부여.⏎⏎턴 종료 시, 모든 적(환상체일 경우, 모든 부위)이 (파열 위력 x 파열 횟수)만큼 탐식 피해를 입음. (파열 횟수 최대 5까지 적용)`
- 現行譯文：`回合開始時，使全體敵方單位(若是異想體，則改為所有部位)增加3級 [Burst] 強度與3回 [Burst]。⏎⏎回合結束時，對全體敵方單位(若是異想體，則改為所有部位)造成(其破裂強度×破裂回數)點暴食傷害。(最多計入5回破裂)`
- 註記：KR 第三行用韓文「파열」，寫「破裂」
- 判讀：譯者說明依 KR／規則修正草稿（標籤、數量、條件或官方名），已處理完畢
- 理由：譯者說明依 KR 修正草稿，譯文已照 KR 處理，不需動作。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N645　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/EGOgift_MirrorDungeon.json|/dataList/174/desc/48642f40`　mirror，批次 20261101T082401-24537171，來源 claude

- KR：`[VibrationExplosion]이 적용된 적에게 이번 턴에 [Vulnerable] 1 부여  (공격 스킬을 사용하여 진동 폭발 부여 시, 스킬 당 최대 1회).`
- 現行譯文：`使敵方單位 [VibrationExplosion] 時，本回合對其施加1層 [Vulnerable] (使用攻擊技能觸發震顫引爆時，每個技能最多1次)。`
- 註記：KR 括號內是韓文「진동 폭발」，寫「震顫引爆」
- 判讀：譯者說明依 KR／規則修正草稿（標籤、數量、條件或官方名），已處理完畢
- 理由：譯者說明依 KR 修正草稿，譯文已照 KR 處理，不需動作。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N646　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/EGOgift_MirrorDungeon.json|/dataList/178/desc/5440b5ff`　mirror，批次 20261101T082401-24537171，來源 claude

- KR：`턴 종료 시 [Burst] 상태인 적의 수(환상체일 경우, 모든 부위)+1만큼, 다음 턴에 무작위 아군 하나에게 [AttackDmgUp] 1, [Agility] 1 효과 중 하나를 부여.`
- 現行譯文：`回合結束時，於下回合隨機使1名友方單位獲得1層 [AttackDmgUp] 或1層 [Agility] 其中一種效果，次數等同於(處於 [Burst] 狀態的敵方單位數(若是異想體，則改為所有部位)+1)。`
- 註記：KR 只有一個 [Burst]；草稿「若有…帶有」改變了條件（KR 為敵方數+1 次，0 名時仍有 1 次），依 KR 重寫
- 判讀：譯者說明依 KR／規則修正草稿（標籤、數量、條件或官方名），已處理完畢
- 理由：譯者說明依 KR 修正草稿，譯文已照 KR 處理，不需動作。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N647　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/EGOgift_MirrorDungeon.json|/dataList/225/desc/cfbf4cb9`　mirror，批次 20261101T082401-24537171，來源 claude

- KR：`아군이 [Vibration] 위력 또는 [Vibration] 횟수를 부여하는 공격 스킬(본인 또는 아군을 대상으로 하는 경우 포함)로 적을 흐트러뜨릴 경우 (E.G.O 스킬 제외), 다음 턴에 대상이 보유한 [Vibration] 위력의 1/3만큼을 다른 무작위 적들(환상체일 경우, 무작위 부위)에게 [Vibration] 위력을 나누어 부여함⏎⏎ 해당 효과를 부여할 다른 적들이 없는 경우, 효과가 변경되어 다음 턴에 대상이 보유한 [Vibration] 위력이 1.5배로 증가`
- 現行譯文：`友方單位透過增加 [Vibration] 強度或 [Vibration] 回數的攻擊技能(包括以自身或友方單位為對象的情況)使敵方單位陷入混亂時(不包括E.G.O技能)，下回合隨機使其他敵方單位(若是異想體，則改為隨機部位)增加共計該單位 [Vibration] 強度1/3的 [Vibration] 強度⏎⏎若沒有可施加本效果的其他敵方單位，則效果改為下回合使目標的 [Vibration] 強度變為1.5倍`
- 註記：草稿漏了「다음 턴에」（下回合），且「본인 또는 아군을 대상으로 하는 경우 포함」誤為「對自身施加」，依 KR 修正
- 判讀：譯者說明依 KR／規則修正草稿（標籤、數量、條件或官方名），已處理完畢
- 理由：譯者說明依 KR 修正草稿，譯文已照 KR 處理，不需動作。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N648　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/EGOgift_MirrorDungeon.json|/dataList/278/desc/50586692`　mirror，批次 20261105T043556-dc1810ed，來源 claude

- KR：`참격 스킬을 2개 이상 보유한 모든 아군이 이번 전투 동안 첫 턴에는 합 위력 +2, 두 번째 턴부터는 [Agility] <style="upgradeHighlight">2</style> 얻고 합 위력 +2.⏎⏎시 협회 소속 인격이 유지 또는 회복 가능한 최대 체력의 상한선이 <style="upgradeHighlight">50%</style>로 고정되는 대신 받는 피해량과 자신이 받는 회복량이 <style="upgradeHighlight">75%</style> 감소 (전투 진입 시 발동, E.G.O 기프트로 인한 회복 제외).⏎⏎시 협회 소속 인격이라면 매 턴 시작시마다 참격 위력 증가 1, 참격 피해량 증가 <style="upgradeHighlight">2</style> 얻음.␣`
- 現行譯文：`本場戰鬥中，擁有不低於2個斬擊技能的友方單位首個回合的拚點威力+2，從第2回合起使其獲得<style="upgradeHighlight">2</style>層 [Agility] 並使其拚點威力+2。⏎⏎屬於し協會的友方單位恢復體力最多恢復至體力上限的<style="upgradeHighlight">50</style>%，但受到的傷害與使自身恢復的體力-<style="upgradeHighlight">75</style>%(關卡開始時立即觸發；不包含E.G.O飾品恢復的體力)。⏎⏎每回合開始時，使屬於し協會的友方單位獲得1層斬擊威力提升 與<style="upgradeHighlight">2</style>層斬擊傷害強化␣`
- 註記：prev_zh 的強調標籤包住「2層 [Agility] 」，依「強調標籤只包 KR 包住的部分」改成只包「2」
- 判讀：譯者說明依 KR／規則修正草稿（標籤、數量、條件或官方名），已處理完畢
- 理由：譯者說明依 KR 修正草稿，譯文已照 KR 處理，不需動作。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N649　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/EGOgift_MirrorDungeon.json|/dataList/82/desc/618888e2`　mirror，批次 20261101T082401-24537171，來源 claude

- KR：`턴 종료 시, [Combustion] 상태인 적에게 대상이 보유한 [Combustion] 횟수의 절반 만큼을 소모하여 ([Combustion] 위력 x 소모한 [Combustion] 횟수)만큼 추가 분노 피해를 입힘.(화상 횟수 최대 5 소모)⏎⏎불꽃의 편린 효과로 소모한 [Combustion] 횟수만큼, 다음 턴 동안 대상의 방어 레벨이 감소 (최대 3).`
- 現行譯文：`回合結束時，使帶有 [Combustion] 的敵方單位消耗一半 [Combustion] 回數並對其額外造成(其 [Combustion] 強度×消耗的 [Combustion] 回數)點暴怒傷害。(最多消耗5回燒傷)⏎⏎下回合使其防禦等級減少(炎鱗消耗的 [Combustion] 回數)級(最多減少3級)`
- 註記：KR「화상 횟수 최대 N 소모」是韓文，寫「燒傷」，刪去多出的 [Combustion]
- 判讀：譯者說明依 KR／規則修正草稿（標籤、數量、條件或官方名），已處理完畢
- 理由：譯者說明依 KR 修正草稿，譯文已照 KR 處理，不需動作。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N650　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/EGOgift_MirrorDungeon.json|/dataList/84/desc/7960df90`　mirror，批次 20261101T082401-24537171，來源 claude

- KR：`턴 종료 시, [Combustion] 상태인 적에게 대상이 보유한 [Combustion] 횟수의 절반 만큼을 소모하여 ([Combustion] 위력 x 소모한 [Combustion] 횟수)만큼 추가 분노 피해를 입힘. <style="upgradeHighlight">(화상 횟수 최대 10 소모)</style>⏎⏎불꽃의 편린 효과로 소모한 [Combustion] 횟수만큼, 다음 턴 동안 대상의 방어 레벨이 감소 <style="upgradeHighlight">(최대 8).</style>`
- 現行譯文：`回合結束時，使帶有 [Combustion] 的敵方單位消耗一半 [Combustion] 回數並對其額外造成(其 [Combustion] 強度×消耗的 [Combustion] 回數)點暴怒傷害。<style="upgradeHighlight">(最多消耗10回燒傷)</style>⏎⏎下回合使其防禦等級減少(炎鱗消耗的 [Combustion] 回數)級<style="upgradeHighlight">(最多減少8級)</style>`
- 註記：草稿多了「×2」，KR／EN 都沒有，已刪去
- 判讀：譯者說明依 KR／規則修正草稿（標籤、數量、條件或官方名），已處理完畢
- 理由：譯者說明依 KR 修正草稿，譯文已照 KR 處理，不需動作。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N651　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/EGOgift_MirrorDungeon_2.json|/dataList/1/desc/d443c690`　mirror，批次 20261101T082410-dc19ddf0，來源 claude

- KR：`턴 시작 시, [Combustion] 및 특수 화상을 부여하는 공격 스킬을 보유한 인격이 5인 이상이면, 이번 전투 동안 발동 (E.G.O 스킬 제외. 대기 인원 제외)⏎⏎스킬(E.G.O 스킬 포함) 적중 시 대상의 [Combustion] 횟수가 3 이상이면, 대상의 [Combustion]을 1회 발동함. ([Combustion] 횟수 1 감소, 스킬당 1회 발동)⏎⏎턴 종료시, [Combustion] 횟수를 3 이상 보유한 적이 [Combustion]을 1회 더 발동함 ([Combustion] 횟수 1 감소)⏎⏎[Combustion] 상태인 적이 사망하였을 경우(환상체일 경우, 본체 사망 시), 남은 [Combustion] 위력 수치만큼을 다음 턴 시작 시 [Combustion] 위력 수치가 제일 적은 적 중 하나에게 부여.⏎⏎턴 시작 시 [Combustion] 상태인 적에게 이번 턴 동안 [AttackDown] 4 부여⏎⏎[Combustion]을 보유한 적이 사망하였을 경우, 다음 턴 시작시, [Combustion] 부여 스킬을 보유한 아군 (사망한 적의 [Combustion] 위력/15)명이 [AttackDmgUp] 2 얻음. (턴 당 1회 발동)`
- 現行譯文：`回合開始時，若擁有施加 [Combustion] 或特殊燒傷的攻擊技能的人格不低於5名，則在本場戰鬥期間觸發該效果。(不包括E.G.O技能、不包括待命人員)⏎⏎技能命中時(包括E.G.O技能)，若目標的 [Combustion] 回數不低於3回，則觸發目標的 [Combustion] 1次。(使其 [Combustion] 回數減少1回，每個技能最多1次)⏎⏎回合結束時，[Combustion] 回數不低於3回的敵方單位額外觸發1次 [Combustion] (使其 [Combustion] 回數減少1回)⏎⏎帶有 [Combustion] 的敵方單位陣亡時(若是異想體，則在本體陣亡時)，下回合開始時將陣亡單位剩餘的 [Combustion] 強度施加給 [Combustion] 強度最低的1名敵方單位。⏎⏎回合開始時，本回合對帶有 [Combustion] 的敵方單位施加4層 [AttackDown]⏎⏎帶有 [Combustion] 的敵方單位陣亡時，在下回合開始時使擁有施加 [Combustion] 的技能的(陣亡單位的 [Combustion] 強度/15)名友方單位獲得2層 [AttackDmgUp]。(每回合最多1次)`
- 註記：草稿第一段多寫了 KR 沒有的「[Combustion] 回數」，依 KR 改為「施加 [Combustion] 或特殊燒傷」
- 判讀：譯者說明依 KR／規則修正草稿（標籤、數量、條件或官方名），已處理完畢
- 理由：譯者說明依 KR 修正草稿，譯文已照 KR 處理，不需動作。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N652　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/EGOgift_MirrorDungeon_2.json|/dataList/10/desc/fe2e2076`　mirror，批次 20261101T082410-dc19ddf0，來源 claude

- KR：`턴 시작 시, [Burst] 위력, [Burst] 횟수를 부여하는 공격 스킬을 보유한 인격이 5인 이상이면, 이번 전투 동안 발동 (E.G.O 스킬 제외. 대기 인원 제외)⏎⏎[Burst] 효과가 적이 보유한 보호막에 입히는 피해량 +150%⏎⏎턴 시작 시, [Burst] 상태인 적 또는 부위의 공격 레벨과 방어 레벨이 (3 + [Burst] 위력/2) 만큼 감소. (최대 6)⏎⏎스킬 효과로 [Burst] 위력, [Burst] 횟수를 부여할 때마다 대상에게 [Burst] 위력 2, [Burst] 횟수 1 부여. 대상의 [Burst] 위력이 15 미만이면, [Burst] 위력 부여 시 추가 2 부여.⏎⏎[Burst] 위력, 횟수 또는 특수 파열을 부여하는 더하기 코인 스킬의 코인 위력 +1⏎[Burst] 위력, 횟수 또는 특수 파열을 부여하는 빼기 코인 스킬의 기본 위력 + (4/코인 수) (최소 1. 소수점 버림)`
- 現行譯文：`回合開始時，若擁有增加 [Burst] 強度或 [Burst] 回數的攻擊技能的人格不低於5名，則在本場戰鬥期間觸發該效果(不包括E.G.O技能、不包括待命人員)⏎⏎[Burst] 效果對敵方單位帶有的護盾造成的傷害+150%⏎⏎回合開始時，帶有 [Burst] 的敵方單位或部位減少(3+(其 [Burst] 強度÷2))級攻擊等級與防禦等級。(最多6級)⏎⏎透過技能效果增加 [Burst] 強度、[Burst] 回數時，使目標增加2級 [Burst] 強度與1回 [Burst]。若目標的 [Burst] 強度低於15級，則增加 [Burst] 強度時額外增加2級。⏎⏎增加 [Burst] 強度、回數或施加特殊破裂的加算硬幣技能的硬幣威力+1⏎增加 [Burst] 強度、回數或施加特殊破裂的減算硬幣技能的基礎威力+(4÷硬幣數)(最少+1，無條件捨去)`
- 註記：KR「턴 시작 시」是回合開始時，草稿誤為「關卡開始時」
- 判讀：譯者說明依 KR／規則修正草稿（標籤、數量、條件或官方名），已處理完畢
- 理由：譯者說明依 KR 修正草稿，譯文已照 KR 處理，不需動作。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N653　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/EGOgift_MirrorDungeon_2.json|/dataList/18/desc/c9ba162e`　mirror，批次 20261101T082410-dc19ddf0，來源 claude

- KR：`턴 시작 시, [Charge] 횟수 또는 특수 충전을 획득하는 공격 스킬을 보유한 인격이 5인 이상이면, 이번 전투 동안 발동 (E.G.O 스킬 제외. 대기 인원 제외)⏎⏎자신의 [Charge] 횟수를 소모한 스킬 공격 종료 시 대상이 사망한 경우, (해당 스킬로 소모한 [Charge] 횟수/3)만큼 자신의 [Charge] 횟수 증가⏎⏎턴 시작 시 [Charge] 횟수가 3 미만인 경우, [Charge] 횟수 3 증가. 턴 시작 시 충전 횟수가 3 이상인 경우, [AttackDmgUp] 1 얻음.`
- 現行譯文：`回合開始時，若擁有獲得 [Charge] 回數或特殊充能的攻擊技能的人格不低於5名，則在本場戰鬥期間觸發該效果。(不包括E.G.O技能、不包括待命人員)⏎⏎消耗自身的 [Charge] 回數的技能攻擊後，若擊殺目標，則使自身獲得(該技能消耗的 [Charge] 回數/3)回 [Charge]⏎⏎回合開始時，若自身的 [Charge] 回數低於3回，則使自身獲得3回 [Charge]。回合開始時，若自身的充能回數不低於3回，則使自身獲得1層 [AttackDmgUp]。`
- 註記：KR 第二個「충전 횟수」是韓文（無 [ID]），寫「充能回數」
- 判讀：譯者說明依 KR／規則修正草稿（標籤、數量、條件或官方名），已處理完畢
- 理由：譯者說明依 KR 修正草稿，譯文已照 KR 處理，不需動作。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N654　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/EGOgift_MirrorDungeon_2.json|/dataList/7/desc/e3e8c2c3`　mirror，批次 20261101T082410-dc19ddf0，來源 claude

- KR：`턴 시작 시, [Vibration] 위력, [Vibration] 횟수 또는 특수 진동을 부여하는 공격 스킬을 보유한 인격이 5인 이상이면, 이번 전투 동안 발동 (E.G.O 스킬 제외. 대기 인원 제외. 인격이 스스로 [Vibration] 횟수를 얻는 경우에는 발동 x)⏎⏎[Vibration] 위력 또는 [Vibration] 횟수를 부여하거나 자신의 [Vibration] 횟수를 소모하는 스킬로 합 승리 시 남은 코인만큼 대상에게 [Vibration] 위력 부여. 자신의 진동 횟수 증가 (스킬당 자신의 진동 횟수 최대 3 증가). 대상에게 [Vibration]이 없으면, 대상의 [Vibration] 횟수 3 증가.⏎⏎턴 시작 시 [ParryingResultUp]를 1 + (자신의 [Vibration] 횟수/5)만큼 얻음. (최대 3)⏎⏎자신의 [Vibration] 횟수를 소모하는 스킬을 사용했을 때, 적 처치 혹은 흐트러짐 상태로 만들경우 (소모한 [Vibration] 횟수 / 2)만큼 자신의 [Vibration] 횟수 증가 (스킬당 1회)⏎⏎스킬의 피해량 +(자신이 소모한 [Vibration] 횟수 + 적이 보유한 [Vibration] 위력 + 5)% (E.G.O 스킬 제외, 최대 50%)⏎대상에게 [VibrationNesting]이 있을 때, 그 대상에게 중첩되지 않은 유형의 [Vibration]으로 [Switch_Vibration], [FusionVibration]을 사용하면, 턴 종료 시 대상이 보유한 [VibrationNesting]을 다음 턴까지 연장함`
- 現行譯文：`回合開始時，若擁有增加 [Vibration] 強度、[Vibration] 回數或施加特殊震顫的攻擊技能的人格不低於5名，則在本場戰鬥期間觸發該效果(不包括E.G.O技能、不包括待命人員、不包括僅擁有對自身施加 [Vibration] 回數的攻擊技能的人格)⏎⏎增加 [Vibration] 強度、[Vibration] 回數或消耗自身的 [Vibration] 回數的技能拚點勝利時，使目標增加(剩餘硬幣數)級 [Vibration] 強度，並使自身的震顫回數增加(剩餘硬幣數)回(每個技能最多增加3回)。若目標未帶有 [Vibration]，則對其施加3回 [Vibration]。⏎⏎回合開始時，使自身獲得(1+(自身的 [Vibration] 回數/5))層 [ParryingResultUp]。(最多3層)⏎⏎若消耗自身的 [Vibration] 回數的技能擊殺目標或使目標陷入混亂，則對自身施加(消耗的 [Vibration] 回數/2)回 [Vibration] (每個技能最多1次)⏎⏎技能造成的傷害+(自身消耗的 [Vibration] 回數+目標的 [Vibration] 強度+5)%(不包括E.G.O技能，最多+50%)⏎若目標帶有 [VibrationNesting]，則以該目標未疊加類型的 [Vibration] 對其使用 [Switch_Vibration]、[FusionVibration] 時，回合結束時使其 [VibrationNesting] 延長至下回合`
- 註記：KR「자신의 진동 횟수」是韓文（無 [ID]），寫「震顫回數」；末段依 KR 改為一個 [Vibration]
- 判讀：譯者說明依 KR／規則修正草稿（標籤、數量、條件或官方名），已處理完畢
- 理由：譯者說明依 KR 修正草稿，譯文已照 KR 處理，不需動作。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N655　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/EGOgift_MirrorDungeon_2.json|/dataList/7/simpleDesc/0/simpleDesc/76c94802`　mirror，批次 20261101T082410-dc19ddf0，來源 claude

- KR：`합 승리 시, 남은 코인만큼 진동 부여. 대상에게 [Vibration]이 없으면, 진동 횟수 증가`
- 現行譯文：`拚點勝利時，使目標增加(剩餘硬幣數)級震顫強度，若目標未帶有 [Vibration]，則使其震顫回數增加`
- 註記：KR 的「진동」是韓文（無 [ID]），寫官方名「震顫」
- 判讀：譯者說明依 KR／規則修正草稿（標籤、數量、條件或官方名），已處理完畢
- 理由：譯者說明依 KR 修正草稿，譯文已照 KR 處理，不需動作。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N656　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/EGOgift_MirrorDungeon_6.json|/dataList/8/desc/3ba74747`　mirror，批次 20261101T082438-51655e35，來源 claude

- KR：`턴 시작 시, 매 턴마다 [GiftCannon] 2, [GiftGlass] 2 얻음⏎2번째 턴부터, 매 턴마다 획득하는 값이 각각 1씩 증가함 (최대 8회 증가)`
- 現行譯文：`回合開始時，每回合使自身獲得2層 [GiftCannon] 並對自身施加2層 [GiftGlass]⏎從第2回合起，每回合獲得的層數各自增加1層(最多增加8次)`
- 註記：draft 第二行多出的 [GiftCannon]、[GiftGlass] 已拿掉（KR 寫「획득하는 값이 각각」），並依 KR 改為「最多增加8次」
- 判讀：譯者說明依 KR／規則修正草稿（標籤、數量、條件或官方名），已處理完畢
- 理由：譯者說明依 KR 修正草稿，譯文已照 KR 處理，不需動作。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N657　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/EGOgift_MirrorDungeon_8.json|/dataList/2/desc/2215c7f0`　mirror，批次 20261101T083920-0e2cb3bf，來源 claude

- KR：`[Charge] 위력이 2 이상인 아군이 [VibrationExplosion] 시 아래 효과 적용⏎- (자신의 [Charge] 위력 x 횟수)만큼 질투 피해를 줌 (스킬당 2회, 1회당 최대 100)⏎- 위 효과로 질투 피해를 2번 주었으면, 다음 턴에 대상에게 [Binding] 3 부여⏎⏎아군이 스킬로 [VibrationExplosion] 시 [VibrationExplosion]마다 (자신의 [Charge] 위력 / 3)만큼 다음 턴에 [AttackUp] 얻음 (턴당 최대 5, 소수점 <noparse>버림</noparse>)`
- 現行譯文：`若友方單位的 [Charge] 強度不低於2級且使目標 [VibrationExplosion] 時，適用以下效果⏎- 造成(自身的 [Charge] 強度×回數)點嫉妒傷害(每個技能2次，每次最多100點)⏎- 若透過上述效果造成了2次嫉妒傷害，則下回合對目標施加3層 [Binding]⏎⏎友方單位以技能使目標 [VibrationExplosion] 時，每次 [VibrationExplosion] 使自身於下回合獲得(自身的 [Charge] 強度÷3)層 [AttackUp] (每回合最多5層，無條件<noparse>捨去</noparse>)`
- 註記：draft 多一個 [Charge]、少一個 [VibrationExplosion]，「1회당 최대 100」誤為每回合，末段漏「下回合」，已依 KR 修正
- 判讀：譯者說明依 KR／規則修正草稿（標籤、數量、條件或官方名），已處理完畢
- 理由：譯者說明依 KR 修正草稿，譯文已照 KR 處理，不需動作。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文

### N658　確認即可｜N7 依規則、hint 或草稿的處理說明｜信心高

`note/EGOgift_StoryDungeon-a1c7p3.json|/dataList/7/desc/58d92aa6`　mirror，批次 20261101T083925-9671105f，來源 claude

- KR：`스테이지 1턴 시작 시 전투에 배치된 인격의 정신력이 5 감소하고, 해당 인격이 3턴동안 [Enhancement] 1 얻음.⏎혈귀 인격인 경우, 정신력 10 감소. 3턴동안 [Enhancement] 2를 얻고, 스킬, 코인의 효과로 [Laceration] 위력, 횟수를 부여할 때 부여량 +1`
- 現行譯文：`關卡的首個回合開始時，使全體參戰的人格失去5點理智值，3回合內使其獲得1層 [Enhancement]。⏎若該人格屬於血魔，則使其失去10點理智值，3回合內使其獲得2層 [Enhancement]，且以技能或硬幣效果施加 [Laceration] 強度與回數時，施加量+1`
- 註記：draft 的「友方單位」依 KR 인격 改為「人格」；KR 只有一個 [Laceration]，改寫為「強度與回數」
- 判讀：譯者說明依 KR／規則修正草稿（標籤、數量、條件或官方名），已處理完畢
- 理由：譯者說明依 KR 修正草稿，譯文已照 KR 處理，不需動作。
- 同意後寫入：`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出；不改譯文
