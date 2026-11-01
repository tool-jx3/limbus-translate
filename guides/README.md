# 翻譯指引

翻譯時依「檔案族群」與「說話角色」組合出要套用的指引。CI 自動翻譯與 Claude Code 人工介入都讀這裡的同一份檔案。

## 結構

| 路徑 | 內容 | 何時套用 |
|---|---|---|
| `base.md` | 共通原則、程式格式須知、標點 | 每一批都套用 |
| `families.md` | 檔名規律、檔案族群、翻譯優先序、凍結項目 | 依檔名判斷族群 |
| `characters/<id>.md` | 角色語言風格 | 該批出現此角色時（劇情看每行的 `model`，RPG 對話看該行的 `speaker` 欄位，對得上角色 id 才套用；語音／`AbDlg` 看檔名） |
| `../glossary/terms.yaml` | 譯名統一列表（硬性規定） | 每一批注入命中的條目 |

疊加順序：`base` → 族群 → 角色 → 單檔備註。詞彙表與 `base.md` 的格式規則是硬性規定，角色風格不能推翻它們。

`characters/*.md` 開頭的 YAML 是給程式用的對照資料：

- `models`：劇情 `StoryData` 裡代表此角色本人的 `model` 鍵值（含受傷、剪影、愚人節等狀態差分）；RPG 對話的 `speaker` 欄位也用這份對照
- `models_other_self`：過去、幼年等其他時期的本人，語氣可能不同，翻譯時要看劇情判斷
- `models_not_this`：鍵值裡有角色名但不是本人（例如 `뫼르소엄마` 是莫梭的母親），或是多人同時說話
- `file_names`：`AbDlg_`、`PersonalityVoiceDlg/Voice_`、`EGOVoiceDig/Voice_EGO_` 檔名中的拼法（遊戲檔名拼法不一致，如 `Meursalut`、`Outist`）
- `number`：遊戲內罪人編號；`code`：檔名代碼（`P1YYXX`、人格編號 `1YYXX` 的 `YY`）

## 標記

每份指引依需要分成以下段落：

- **原指引**：既有的社群指引，保留原文
- **資料補充**：依現有繁體譯文統計得出，附數字與出處
- **決定**：資料和原指引對不上時，經人工決定的結果與理由（原則：依 KR 原文判斷）
- **待確認**：還沒決定、需要人工確認的地方

## 資料依據

- 原文：官方 2026-10-29 語言包（token `l20261029_…`），KR／EN／JP
- 繁中：`user-unknown-0711/ParatranzUploader` release `1793285016`（2026-10-29）的 `localize_files.zip`
- 對照：`LimbusTraditionalMandarin/storyline` release `1792667238`（2026-10-22）的 `Hant.zip`
- 只統計「確實是中文譯文」的行：排除等於 EN 原文（ParatranzUploader 用英文填未翻條目）、等於 KR、仍含諺文的行
- 角色統計只用本人的基本鍵值（例如 `model=이상`），加上該角色的語音、`AbDlg` 檔
- KR 語體比例只計「以動詞語尾結束的句子」：합쇼체（~니다）、해요체（~요）、하오체（~오／~소）、반말（其他常體語尾）。這是粗略分類，用來看傾向
- 統計腳本在 `docs/research/scripts/`

例句的「既有繁中」是現有譯文，用來示範語氣，不代表每句都是定稿。
