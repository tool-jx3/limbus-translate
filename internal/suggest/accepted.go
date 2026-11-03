package suggest

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/tool-jx3/limbus-translate/internal/rulefile"
)

// AcceptedFile is the list of the check warnings that people confirmed as
// fine, relative to the repo root. It is optional, and has the format of
// DeclinedFile; the id of an entry is itemid.Warning of the KR text and the
// translation, so a change of the translation brings the warning back.
const AcceptedFile = "rules/accepted.yaml"

// acceptedIDPattern is the id of a check warning: warning/<code>/<sha8 of KR>/<sha8 of ZH>.
var acceptedIDPattern = regexp.MustCompile(`^warning/[a-z0-9-]+/[0-9a-f]{8}/[0-9a-f]{8}$`)

// acceptedHeader is the comment that a new AcceptedFile starts with.
const acceptedHeader = `# 已確認的檢查警告
#
# 審核頁上確認「沒問題」的檢查警告記在這裡，審核頁與建置報告的警告統計
# （warnings、warning_samples）都會略過它。由 ` + "`lct accept`" + ` 附加，也可以手動新增。
#
# 格式：清單，每筆
#   id      警告的 id：warning/<代碼>/<sha8(KR)>/<sha8(TW)>
#   reason  確認的理由（必填）
#   date    YYYY-MM-DD
#
# id 不含檔案或位置：KR 與 TW 都相同的單位共用同一筆確認，新快照讓單位換了指標
# 也仍然有效；TW 譯文一改，id 就變，確認隨即失效，警告會再出現，需要重新確認。
#
#   - id: "warning/kana/0123abcd/89ef4567"
#     reason: 日文假名是刻意保留的
#     date: 2026-11-05
`

// LoadAccepted reads the accepted list of root, with the rules of LoadDeclined:
// the same entries, and a root without the file has none.
func LoadAccepted(root string) ([]Declined, error) { return loadEntries(root, AcceptedFile) }

// LintAccepted checks the accepted list of root and returns every problem it
// finds: what LoadAccepted refuses, an id that is not a warning id, an empty
// reason, a date that is not YYYY-MM-DD, and an id that is listed twice. A root
// without the file has no issues.
func LintAccepted(root string) []rulefile.Issue {
	return lintEntries(root, AcceptedFile, func(list []Declined) []string {
		return entryProblems(list, func(id string) string {
			if acceptedIDPattern.MatchString(id) {
				return ""
			}
			return fmt.Sprintf("id %q is not a warning id; it is warning/<code>/<8 hex digits>/<8 hex digits>", id)
		})
	})
}

// AppendEntries adds entries at the end of file (relative to root), one entry
// as DeclinedFile has them: the id in double quotes, then reason and date. The
// text of the file stays as it is, comments included; a file that does not
// exist is first written with the comment of AcceptedFile. No entries leave the
// file untouched.
func AppendEntries(root, file string, entries []Declined) error {
	if len(entries) == 0 {
		return nil
	}
	path := filepath.Join(root, filepath.FromSlash(file))
	old, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		old = []byte(acceptedHeader)
	} else if err != nil {
		return err
	}
	var seq yaml.Node
	seq.Kind = yaml.SequenceNode
	for _, e := range entries {
		seq.Content = append(seq.Content, entryNode(e))
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&seq); err != nil {
		return err
	}
	if err := enc.Close(); err != nil {
		return err
	}
	out := old
	if len(out) > 0 && out[len(out)-1] != '\n' {
		out = append(out, '\n')
	}
	out = append(out, buf.Bytes()...)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}

// entryNode is the YAML mapping of an entry. The id is always in double quotes;
// the reason is quoted by the encoder where its text needs it; a date that is a
// day is written bare, as the entries of DeclinedFile have it.
func entryNode(e Declined) *yaml.Node {
	str := func(value string, style yaml.Style) *yaml.Node {
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value, Style: style}
	}
	date := str(e.Date, 0)
	if t, err := time.Parse("2006-01-02", e.Date); err != nil || t.Format("2006-01-02") != e.Date {
		date.Style = yaml.DoubleQuotedStyle
	} else {
		date.Tag = "" // implicit: written without quotes
	}
	reason := str(strings.TrimRight(e.Reason, "\n"), 0)
	if strings.Contains(reason.Value, "\n") {
		reason.Style = yaml.DoubleQuotedStyle // never a block scalar inside the entry
	}
	return &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{
		str(declinedFields[0], 0), str(e.ID, yaml.DoubleQuotedStyle),
		str(declinedFields[1], 0), reason,
		str(declinedFields[2], 0), date,
	}}
}

// RemoveEntries deletes the entries of file (relative to root) whose id is in
// ids and returns the ids it removed, in the order of the file. Only the lines
// of the removed entries go: comments, blank lines and the text of the other
// entries stay as they are. A file that does not exist, or lists none of ids,
// is left untouched.
func RemoveEntries(root, file string, ids map[string]bool) (removed []string, err error) {
	path := filepath.Join(root, filepath.FromSlash(file))
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) != 1 || doc.Content[0].Kind != yaml.SequenceNode {
		return nil, nil
	}
	items := doc.Content[0].Content
	lines := strings.SplitAfter(string(data), "\n")
	drop := make([]bool, len(lines))
	for i, item := range items {
		id := ""
		if item.Kind == yaml.MappingNode {
			for j := 0; j+1 < len(item.Content); j += 2 {
				if item.Content[j].Value == "id" {
					id = item.Content[j+1].Value
				}
			}
		}
		if id == "" || !ids[id] {
			continue
		}
		first := item.Line - 1
		end := len(lines)
		if i+1 < len(items) {
			end = items[i+1].Line - 1
		}
		// Blank and comment lines after the entry belong to what follows.
		for end > first+1 {
			if t := strings.TrimSpace(lines[end-1]); t != "" && !strings.HasPrefix(t, "#") {
				break
			}
			end--
		}
		for l := first; l < end; l++ {
			drop[l] = true
		}
		removed = append(removed, id)
	}
	if len(removed) == 0 {
		return nil, nil
	}
	var out strings.Builder
	for l, text := range lines {
		if !drop[l] {
			out.WriteString(text)
		}
	}
	if err := os.WriteFile(path, []byte(out.String()), 0o644); err != nil {
		return nil, err
	}
	return removed, nil
}
