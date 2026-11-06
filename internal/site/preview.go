package site

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/align"
	"github.com/tool-jx3/limbus-translate/internal/batch"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
	"github.com/tool-jx3/limbus-translate/internal/validate"
)

// PreviewInputs are what Export needs to write the data of the translation
// preview (data/preview.json and data/preview/<path>): the viewer rebuilds, from
// it and the files, the batch a unit would get if it were sent for translation
// alone (see batch.Previewer). Without it (Inputs.Preview nil) Export writes
// no preview.
type PreviewInputs struct {
	Previewer *batch.Previewer
	// RulesCommit names the rules the preview was made under: the page shows
	// it as the rules the batches are simulated with.
	RulesCommit string
	// Prompt is the text of prompts/translate.md.
	Prompt string
	// Guides maps the repo-relative, slash-separated path of each guide file
	// (guides/base.md) to its text. Every guide a batch names must be in it.
	Guides map[string]string
}

// PreviewGuide is a guide file in data/preview.json.
type PreviewGuide struct {
	Path string `json:"path"`
	Text string `json:"text"`
}

// PreviewShared is data/preview.json: what every batch shares. Guides lists
// only the guides some batch names, numbered in the order they are named first
// (files in the order of the index, units in document order, guides in the
// order of Batch.Guides); the Unit.G of a file are numbers in this list.
type PreviewShared struct {
	RulesCommit string         `json:"rules_commit"`
	Prompt      string         `json:"prompt"`
	Guides      []PreviewGuide `json:"guides"`
}

// PreviewFile is data/preview/<path>: what the batches of the units of a file
// have in common, and the part of each that is not in files/<path>. Units has
// the length and the order of the units of files/<path>; the entry of a copy
// is null.
//
// The batch of a unit is rebuilt as batch.Batch{ID: "preview", Family,
// GlossaryCheck, File: path, Files: [path], Guides, Glossary, Keywords, Units:
// [one unit], Context}: Guides are the paths of the numbers in Unit.G;
// Glossary is Terms[t] for each t of Unit.T, with Units set to [the key of
// the unit] (an empty list is []); Keywords is the names of Unit.K, taken from
// Keywords ({} when none); Units and Context are never null.
type PreviewFile struct {
	Family        string `json:"family"`
	GlossaryCheck string `json:"glossary_check"`
	// Terms are the glossary entries of the batches of the file, each once,
	// without their Units.
	Terms []rules.Term `json:"terms"`
	// Keywords are the names of the keywords the batches of the file name.
	Keywords map[string]string `json:"keywords"`
	// RD are the values most records of the file have; a record leaves out
	// the fields that equal them. Absent when the file has no record.
	RD    *PreviewRecDefaults `json:"rd,omitempty"`
	Units []*PreviewUnit      `json:"units"`
}

// PreviewRecDefaults are the most common origin, rules commit and since of the
// records of a file (the least of them where two are as common).
type PreviewRecDefaults struct {
	Origin string `json:"origin"`
	Rules  string `json:"rules"`
	Since  string `json:"since"`
}

// PreviewItem is the part of the batch unit that comes from the pending item.
type PreviewItem struct {
	Reason string            `json:"reason"`
	PrevKR string            `json:"prev_kr,omitempty"`
	PrevZH string            `json:"prev_zh,omitempty"`
	Draft  string            `json:"draft,omitempty"`
	Hint   string            `json:"hint,omitempty"`
	Refs   map[string]string `json:"refs,omitempty"`
}

// PreviewState is batch.PreviewState without the rules commit, which
// PreviewShared has. Every field is left out at its zero value, and a unit
// whose fields are all zero has no st.
type PreviewState struct {
	Pending     bool   `json:"pending,omitempty"`
	NeedsReview bool   `json:"needs_review,omitempty"`
	LeasedTo    string `json:"leased_to,omitempty"`
	Frozen      bool   `json:"frozen,omitempty"`
	Attempts    int    `json:"attempts,omitempty"`
	LastError   string `json:"last_error,omitempty"`
}

// PreviewRecord is the stored translation of a unit. Stale is true when it was
// made from another KR text than the one there is now. Origin, Rules and Since
// are nil where the value is that of PreviewFile.RD (a pointer, so that an
// empty value of its own can be told from the default).
type PreviewRecord struct {
	Origin *string  `json:"origin,omitempty"`
	Model  string   `json:"model,omitempty"`
	Batch  string   `json:"batch,omitempty"`
	Since  *string  `json:"since,omitempty"`
	Rules  *string  `json:"rules,omitempty"`
	Flags  []string `json:"flags,omitempty"`
	Stale  bool     `json:"stale,omitempty"`
}

// PreviewCheck is how the record fares under the submit check. ZH is nil where
// the text checked is the ZH of the unit in files/<path>. A record whose check
// is an accept, with no issue, of that text has no PreviewCheck at all.
type PreviewCheck struct {
	Verdict string           `json:"verdict"`
	ZH      *string          `json:"zh,omitempty"`
	Issues  []validate.Issue `json:"issues,omitempty"`
}

// PreviewUnit is what the batch of one unit has that files/<path> and
// PreviewFile do not.
type PreviewUnit struct {
	// SP is the speaker of the batch unit, and of the unit as a line of
	// context.
	SP string `json:"sp"`
	// G, T and K are the guides (numbers in PreviewShared.Guides), the
	// glossary entries (numbers in PreviewFile.Terms) and the keyword ids of
	// the batch, in its order (K sorted).
	G  []int       `json:"g"`
	T  []int       `json:"t"`
	K  []string    `json:"k"`
	It PreviewItem `json:"it"`
	// PZ is 1 where It.PrevZH is the ZH of the unit in files/<path>, and then
	// It.PrevZH is left out.
	PZ int `json:"pz,omitempty"`
	// M are the methods of the aligned EN and JP strings, whose values are
	// those of files/<path>: two characters, EN and then JP, each p (pointer),
	// i (id), s (sequence) or - (no aligned string). Absent when neither
	// language has one.
	M string `json:"m,omitempty"`
	// Keep is absent when nothing is to be kept (no ids, placeholders or tags,
	// and no Dante wrapper).
	Keep *batch.Constraints `json:"keep,omitempty"`
	// C are the lines of context: the number of the unit in files/<path> and,
	// where the line shows another translation than the unit's ZH there, that
	// translation. The speaker and the KR text are those of the unit.
	C [][]any `json:"c"`
	// St is absent when all its fields are zero.
	St  *PreviewState  `json:"st,omitempty"`
	Rec *PreviewRecord `json:"rec,omitempty"`
	Chk *PreviewCheck  `json:"chk,omitempty"`
	// NT is the notes on the unit, joined by a line feed.
	NT string `json:"nt,omitempty"`
}

// guideTable numbers the guides in the order they are named first.
type guideTable struct {
	texts map[string]string
	list  []PreviewGuide
	at    map[string]int
}

func (g *guideTable) number(path string) (int, error) {
	if n, ok := g.at[path]; ok {
		return n, nil
	}
	text, ok := g.texts[path]
	if !ok {
		return 0, fmt.Errorf("site: preview: the guide %s is named by a batch but is not in the guides", path)
	}
	n := len(g.list)
	g.at[path] = n
	g.list = append(g.list, PreviewGuide{Path: path, Text: text})
	return n, nil
}

// buildPreviewFile makes the preview data of a file from its units and their
// previews, which have the same length and order.
func buildPreviewFile(file File, pvs []*batch.UnitPreview, guides *guideTable) (PreviewFile, error) {
	if len(pvs) != len(file.Units) {
		return PreviewFile{}, fmt.Errorf("site: preview %s: %d units previewed, %d in the file", file.Path, len(pvs), len(file.Units))
	}
	out := PreviewFile{
		Family:   file.Family,
		Terms:    []rules.Term{},
		Keywords: map[string]string{},
		Units:    make([]*PreviewUnit, len(file.Units)),
	}
	glossaryCheckSet := false // out.GlossaryCheck is the one of the batches
	byPointer := make(map[string]int, len(file.Units))
	for i, u := range file.Units {
		byPointer[u.P] = i
	}
	termAt := map[string]int{}
	var origins, rulesOf, sinces []string // of the records of the file
	for i, pv := range pvs {
		if pv == nil {
			continue
		}
		b := pv.Batch
		if len(b.Units) != 1 || b.Family != file.Family || b.File != file.Path {
			return PreviewFile{}, fmt.Errorf("site: preview %s: the batch of unit %d is not that of one unit of the file", file.Path, i)
		}
		if !glossaryCheckSet {
			out.GlossaryCheck, glossaryCheckSet = b.GlossaryCheck, true
		} else if b.GlossaryCheck != out.GlossaryCheck {
			return PreviewFile{}, fmt.Errorf("site: preview %s: the batch of unit %d has glossary_check %q, the others %q", file.Path, i, b.GlossaryCheck, out.GlossaryCheck)
		}
		bu := b.Units[0]
		fu := file.Units[i]
		if wantKey := units.Key(file.Path, fu.P); bu.Key != wantKey {
			return PreviewFile{}, fmt.Errorf("site: preview %s: unit %d is %s, want %s", file.Path, i, bu.Key, wantKey)
		}
		u := &PreviewUnit{
			SP: bu.Speaker,
			G:  []int{},
			T:  []int{},
			K:  []string{},
			It: PreviewItem{Reason: bu.Reason, PrevKR: bu.PrevKR, PrevZH: bu.PrevZH, Draft: bu.Draft, Hint: bu.Hint, Refs: bu.Refs},
			C:  [][]any{},
		}
		if k := bu.Keep; len(k.IDs) > 0 || len(k.Placeholders) > 0 || len(k.Tags) > 0 || k.DanteWrapper {
			u.Keep = &k
		}
		if st := (PreviewState{
			Pending: pv.State.Pending, NeedsReview: pv.State.NeedsReview, LeasedTo: pv.State.LeasedTo,
			Frozen: pv.State.Frozen, Attempts: pv.State.Attempts, LastError: pv.State.LastError,
		}); st != (PreviewState{}) {
			u.St = &st
		}
		if bu.PrevZH != "" && bu.PrevZH == fu.ZH {
			u.PZ = 1
			u.It.PrevZH = ""
		}
		for _, g := range b.Guides {
			n, err := guides.number(g)
			if err != nil {
				return PreviewFile{}, err
			}
			u.G = append(u.G, n)
		}
		for _, t := range b.Glossary {
			t.Units = nil
			enc, err := json.Marshal(t)
			if err != nil {
				return PreviewFile{}, fmt.Errorf("site: preview %s: %w", file.Path, err)
			}
			n, ok := termAt[string(enc)]
			if !ok {
				n = len(out.Terms)
				termAt[string(enc)] = n
				out.Terms = append(out.Terms, t)
			}
			u.T = append(u.T, n)
		}
		for id, name := range b.Keywords {
			out.Keywords[id] = name
			u.K = append(u.K, id)
		}
		sort.Strings(u.K)
		m, err := methodChars(bu.EN, bu.JP)
		if err != nil {
			return PreviewFile{}, fmt.Errorf("site: preview %s: unit %d: %w", file.Path, i, err)
		}
		u.M = m
		for _, line := range b.Context {
			_, pointer, err := units.SplitKey(line.Key)
			if err != nil {
				return PreviewFile{}, fmt.Errorf("site: preview %s: %w", file.Path, err)
			}
			at, ok := byPointer[pointer]
			if !ok || line.Key != units.Key(file.Path, pointer) {
				return PreviewFile{}, fmt.Errorf("site: preview %s: the context line %s is no unit of the file", file.Path, line.Key)
			}
			if line.ZH == file.Units[at].ZH {
				u.C = append(u.C, []any{at})
			} else {
				u.C = append(u.C, []any{at, line.ZH})
			}
		}
		if rec := pv.Record; rec != nil {
			origins = append(origins, rec.Origin)
			rulesOf = append(rulesOf, rec.Rules)
			sinces = append(sinces, rec.Since)
			u.Rec = &PreviewRecord{
				Origin: &rec.Origin, Model: rec.Model, Batch: rec.Batch, Since: &rec.Since, Rules: &rec.Rules,
				Flags: slices.Clone(rec.Flags), Stale: rec.KRSha != units.KRSha(fu.KR),
			}
		}
		if c := pv.Check; c != nil {
			if c.Verdict != "accept" || len(c.Issues) > 0 || c.ZH != fu.ZH {
				u.Chk = &PreviewCheck{Verdict: c.Verdict, Issues: c.Issues}
				if c.ZH != fu.ZH {
					u.Chk.ZH = &c.ZH
				}
			}
		}
		u.NT = joinNotes(pv.Notes)
		out.Units[i] = u
	}
	if len(origins) > 0 {
		rd := PreviewRecDefaults{Origin: mostCommon(origins), Rules: mostCommon(rulesOf), Since: mostCommon(sinces)}
		out.RD = &rd
		for _, u := range out.Units {
			if u == nil || u.Rec == nil {
				continue
			}
			if *u.Rec.Origin == rd.Origin {
				u.Rec.Origin = nil
			}
			if *u.Rec.Rules == rd.Rules {
				u.Rec.Rules = nil
			}
			if *u.Rec.Since == rd.Since {
				u.Rec.Since = nil
			}
		}
	}
	return out, nil
}

// mostCommon is the value that occurs most often in vs, the least of those
// that occur as often, "" for none.
func mostCommon(vs []string) string {
	count := map[string]int{}
	best, n := "", 0
	for _, v := range vs {
		count[v]++
	}
	for v, c := range count {
		if c > n || c == n && v < best {
			best, n = v, c
		}
	}
	return best
}

// methodChars is PreviewUnit.M for the aligned EN and JP strings of a unit.
func methodChars(en, jp *align.LangAligned) (string, error) {
	if en == nil && jp == nil {
		return "", nil
	}
	one := func(a *align.LangAligned) (string, error) {
		if a == nil {
			return "-", nil
		}
		switch a.Method {
		case align.MethodPointer:
			return "p", nil
		case align.MethodID:
			return "i", nil
		case align.MethodSequence:
			return "s", nil
		}
		return "", fmt.Errorf("the alignment method %q has no character", a.Method)
	}
	e, err := one(en)
	if err != nil {
		return "", err
	}
	j, err := one(jp)
	if err != nil {
		return "", err
	}
	return e + j, nil
}

func joinNotes(notes []store.Note) string {
	texts := make([]string, len(notes))
	for i, n := range notes {
		texts[i] = n.Note
	}
	return strings.Join(texts, "\n")
}
