package align

import (
	"github.com/tool-jx3/limbus-translate/internal/jsonx"
	"github.com/tool-jx3/limbus-translate/internal/pack"
)

// FileAligner finds the EN and JP strings of the units of one KR file. The
// alignment is slow, so it runs once per file, when the first unit asks.
type FileAligner struct {
	file         string
	kr           *jsonx.Node
	en, jp       *pack.Pack
	loaded       bool
	enAt         map[string]LangAligned
	jpAt         map[string]LangAligned
	hasEN, hasJP bool // the pack has the file
}

// NewFileAligner aligns file of the EN and JP packs to kr, its KR document. A
// pack that is nil, or that lacks the file, has no strings for it.
func NewFileAligner(file string, kr *jsonx.Node, en, jp *pack.Pack) *FileAligner {
	return &FileAligner{file: file, kr: kr, en: en, jp: jp}
}

// Get returns the EN and JP strings aligned to the KR string at pointer; nil
// means there is none (see AlignLang). The error is that of reading a file.
func (a *FileAligner) Get(pointer string) (en, jp *LangAligned, err error) {
	if err := a.load(); err != nil {
		return nil, nil, err
	}
	pick := func(m map[string]LangAligned) *LangAligned {
		if v, ok := m[pointer]; ok {
			return &v
		}
		return nil
	}
	return pick(a.enAt), pick(a.jpAt), nil
}

// Unmatched counts the strings of the KR file that have no EN string aligned
// to them, and those that have no JP string. A language whose pack lacks the
// file (or is nil) counts none: the file is missing there, which is not a
// misalignment of its units.
func (a *FileAligner) Unmatched() (en, jp int, err error) {
	if err := a.load(); err != nil {
		return 0, 0, err
	}
	for _, f := range a.kr.Strings() {
		if _, ok := a.enAt[f.Pointer]; a.hasEN && !ok {
			en++
		}
		if _, ok := a.jpAt[f.Pointer]; a.hasJP && !ok {
			jp++
		}
	}
	return en, jp, nil
}

// load aligns both languages, once.
func (a *FileAligner) load() (err error) {
	if a.loaded {
		return nil
	}
	if a.enAt, err = a.align(a.en); err != nil {
		return err
	}
	if a.jpAt, err = a.align(a.jp); err != nil {
		return err
	}
	a.hasEN, a.hasJP = a.has(a.en), a.has(a.jp)
	a.loaded = true
	return nil
}

func (a *FileAligner) has(p *pack.Pack) bool { return p != nil && p.Has(a.file) }

func (a *FileAligner) align(other *pack.Pack) (map[string]LangAligned, error) {
	if !a.has(other) {
		return nil, nil
	}
	doc, err := other.Doc(a.file)
	if err != nil {
		return nil, err
	}
	return AlignLang(a.file, a.kr, doc), nil
}
