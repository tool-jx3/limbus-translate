package batch

import (
	"slices"

	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/validate"
)

// Verdict is how a translation fares under the submit check.
type Verdict struct {
	// ZH is the translation after normalization; it is the text that is
	// checked and, if accepted, recorded.
	ZH string
	// Issues are the findings. A rejected verdict holds those of the check;
	// an accepted one adds the foreign-name warnings.
	Issues []validate.Issue
	// Rejected is true when an issue is hard.
	Rejected bool
}

// Judge decides whether the translation zh of the KR text kr in file passes
// the submit check, the one decision Submit and the preview share so that
// they cannot drift apart. prov are the provisional terms the translation is
// held to (see provisionalOf); note is the translator's note of the unit.
//
// The text is normalized (validate.Normalize, then the space after the keyword
// names of env.KeywordNames) and checked as a new translation. A missing
// glossary rendering is only a warning in CheckNew; here it is hard where the
// entry behind it is check: hard (a provisional term never is), unless the
// note names the term, the way the entry itself matches a KR text. A family
// with glossary: warn keeps it a warning whatever the note says.
//
// Foreign names are warned about only when the translation is accepted, so a
// rejected verdict never carries them.
func Judge(env Env, prov []rules.Term, file, kr, zh, note string) Verdict {
	zh = validate.NormalizeKeywordSpace(validate.Normalize(kr, zh), env.KeywordNames)
	hits := env.Rules.GlossaryHits(file, kr) // looked up once for the check and the entries
	var scriptOK []string
	if env.Scripts != nil {
		scriptOK = env.Scripts.For(file)
	} else {
		scriptOK = env.Rules.ScriptRenderings(file)
	}
	issues := checkNewTerms(hits, prov, file, kr, zh, scriptOK)
	entries := append(slices.Clip(hits), prov...)
	warnOnly := env.Rules.GlossaryWarnOnly(file)
	for j := range issues {
		if issues[j].Code == validate.CodeTermMissing && !warnOnly && hardEntry(entries, issues[j]) && !noteNames(entries, issues[j], note) {
			issues[j].Severity = validate.SeverityHard
		}
	}
	if validate.HasHard(issues) {
		return Verdict{ZH: zh, Issues: issues, Rejected: true}
	}
	issues = append(issues, foreignNameIssues(zh, env.ForeignNames, env.Rules.ForeignIgnore)...)
	return Verdict{ZH: zh, Issues: issues}
}

// provisionalOf returns the provisional terms of inForce that the glossary of
// a batch carries (provisional: true). The batch glossary only decides which
// KRs are checked; the rendering is the one in inForce, the one the next
// batch would carry.
func provisionalOf(glossary []rules.Term, inForce []rules.Term) []rules.Term {
	var out []rules.Term
	for _, g := range glossary {
		if !g.Provisional {
			continue
		}
		for _, t := range inForce {
			if t.KR == g.KR {
				out = append(out, t)
				break
			}
		}
	}
	return out
}
