package align

import (
	"github.com/tool-jx3/limbus-translate/internal/jsonx"
	"github.com/tool-jx3/limbus-translate/internal/pack"
)

// FileAligner finds the EN and JP strings of the units of one KR file. The
// alignment is slow, so it runs once per file, when the first unit asks.
type FileAligner struct {
	file   string
	kr     *jsonx.Node
	en, jp *pack.Pack
	loaded bool
	enAt   map[string]LangAligned
	jpAt   map[string]LangAligned
}

// NewFileAligner aligns file of the EN and JP packs to kr, its KR document. A
// pack that is nil, or that lacks the file, has no strings for it.
func NewFileAligner(file string, kr *jsonx.Node, en, jp *pack.Pack) *FileAligner {
	return &FileAligner{file: file, kr: kr, en: en, jp: jp}
}

// Get returns the EN and JP strings aligned to the KR string at pointer; nil
// means there is none (see AlignLang). The error is that of reading a file.
func (a *FileAligner) Get(pointer string) (en, jp *LangAligned, err error) {
	if !a.loaded {
		if a.enAt, err = a.align(a.en); err != nil {
			return nil, nil, err
		}
		if a.jpAt, err = a.align(a.jp); err != nil {
			return nil, nil, err
		}
		a.loaded = true
	}
	pick := func(m map[string]LangAligned) *LangAligned {
		if v, ok := m[pointer]; ok {
			return &v
		}
		return nil
	}
	return pick(a.enAt), pick(a.jpAt), nil
}

func (a *FileAligner) align(other *pack.Pack) (map[string]LangAligned, error) {
	if other == nil || !other.Has(a.file) {
		return nil, nil
	}
	doc, err := other.Doc(a.file)
	if err != nil {
		return nil, err
	}
	return AlignLang(a.file, a.kr, doc), nil
}
