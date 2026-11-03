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
# 審閱頁上確認「沒問題」的檢查警告記在這裡，` + "`lct check`" + ` 與審閱頁不再把它當作待處理。
# 由 ` + "`lct accept`" + ` 附加，也可以手動新增。
#
# 格式：清單，每筆
#   id      警告的 id：warning/<代碼>/<原文 sha8>/<譯文 sha8>
#   reason  確認的理由（必填）
#   date    YYYY-MM-DD
#
# id 含譯文的雜湊：譯文改了，id 就變，警告會再出現，需要重新確認。
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
