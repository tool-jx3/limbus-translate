# I4 同義措辭不同：只修錯的譯法（G567–G569）

[回這一類的目錄](I4.md)。

### G567　只修錯的譯法｜I4 同義措辭不同｜信心中

`inconsistent/mirror/message/0e067c45`　mirror／message，6 個單位，JP/EN 全同

- KR：`자리를 떠난다.`
- JP：`その場から離れる。`　EN：`Leave.`
- 現有譯法：
  1. `離開。`（3 個單位；來源 paratranz 3）
  2. `離開這個地方。`（1 個單位；來源 paratranz 1）
  3. `離開那裡。`（1 個單位；來源 paratranz 1）
  4. `遠離花樹。`（1 個單位；來源 paratranz 1）
- 判讀：「離開這個地方」「離開那裡」多出指示詞，JP その場から離れる、EN Leave. 都沒有；統一成「離開。」，「遠離花樹」是針對花樹事件的情境譯法，保留
- 建議：只改下列譯法，其他保留
- 會改：
  - `離開這個地方。` → `離開。`：1 個單位（ActionEvents-a1c5p3.json）
    - 差異：`離開〔這個地方→∅〕。`
  - `離開那裡。` → `離開。`：1 個單位（ActionEvents-a1c7p3.json）
    - 差異：`離開〔那裡→∅〕。`
  - 不改：`離開。`（3）、`遠離花樹。`（1）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G568　只修錯的譯法｜I4 同義措辭不同｜信心中

`inconsistent/mirror/message/2a37ff8b`　mirror／message，6 個單位，部分譯法對到不同 JP/EN

- KR：`지나친다.`
- 現有譯法：
  1. `越過他們。`（2 個單位；來源 paratranz 2）
     - JP `通り過ぎる。`／EN `Walk away.`（2）
  2. `越過這裡。`（2 個單位；來源 paratranz 2）
     - JP `通り過ぎる。`／EN `Walk away.`（2）
  3. `越過它。`（1 個單位；來源 paratranz 1）
     - JP `通り過ぎる。`／EN `Walk past it.`（1）
  4. `走過它。`（1 個單位；來源 paratranz 1）
     - JP `通り過ぎる。`／EN `Walk past it.`（1）
- 判讀：同檔 ActionEvents_Mirror3 的兩個選項，JP、EN 皆為 Walk past it.，「走過它」與「越過它」用詞不同，統一為「越過它」；「他們」「這裡」是對象不同，保留
- 建議：只改下列譯法，其他保留
- 會改：
  - `走過它。` → `越過它。`：1 個單位（ActionEvents_Mirror3.json）
    - 差異：`〔走→越〕過它。`
  - 不改：`越過他們。`（2）、`越過這裡。`（2）、`越過它。`（1）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出

### G569　只修錯的譯法｜I4 同義措辭不同｜信心中

`inconsistent/mirror/message/e031a869`　mirror／message，4 個單位，部分譯法對到不同 JP/EN

- KR：`가져가지 않는다.`
- 現有譯法：
  1. `不拿走。`（2 個單位；來源 paratranz 2）
     - JP `持っていかない。`／EN `Leave it.`（2）
  2. `不拿刀。`（1 個單位；來源 paratranz 1）
     - JP `持っていかない。`／EN `Don't take the sword.`（1）
  3. `不去拿。`（1 個單位；來源 paratranz 1）
     - JP `持っていかない。`／EN `Leave it.`（1）
- 判讀：JP、EN 相同（持っていかない／Leave it.），「不去拿」與「不拿走」同義，統一為較貼近 KR「가져가지 않는다」的「不拿走。」；「不拿刀」是針對劍的情境譯法，保留
- 建議：只改下列譯法，其他保留
- 會改：
  - `不去拿。` → `不拿走。`：1 個單位（ActionEvents_mowe-re.json）
    - 差異：`不〔去→∅〕拿。`；`不去拿〔∅→走〕。`
  - 不改：`不拿走。`（2）、`不拿刀。`（1）
- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）；這個 id 不再列出
