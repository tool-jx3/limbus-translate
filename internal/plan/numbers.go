package plan

import (
	"slices"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/batch"
	"github.com/tool-jx3/limbus-translate/internal/existing"
	"github.com/tool-jx3/limbus-translate/internal/rewrite"
	"github.com/tool-jx3/limbus-translate/internal/validate"
)

// patchChanged gives the units the transition queued as changed (p.changed)
// the translation of their new KR text when only numbers changed, and takes
// them off the queue (see Run). A unit is patched when:
//
//   - it is still queued as the transition queued it, and for nothing else:
//     it had no pending item before (something was asked of it already), no
//     step of the plan settled it since (a copy, an adjustment made for the
//     new text: those are off the queue), and no adjustment of it expired
//     (that unit is queued as adjustment-expired, for a translator to read
//     what the person wrote);
//   - its record was a translation of the old KR text that may serve again
//     (existing.Usable: no stand-in, no copy, nothing a person has to look
//     at) and that no person wrote (rewrite.DefaultOrigins: the plan changes
//     a person's wording no more than a rewrite does);
//   - patchNumbers can make the new translation from it;
//   - the new translation passes the checks of a new translation of the new
//     KR text (batch.CheckNewAt), as a rewritten or reused one must.
//
// The record of a patched unit is the one it had, origin, model, batch, raw
// tags, rules and flags, as a rewrite leaves a record it changes: the
// translation is still the one that was made then. Only the text, with the
// new numbers, and the KR text it belongs to are new, and the key when the
// unit moved. Any other unit stays queued as it was.
func (p *planner) patchChanged() {
	for _, c := range p.changed {
		ref, inKR := p.cur[c.Key]
		it, queued := p.pending[c.Key]
		if !inKR || !queued || c.Queued || it.Reason != ReasonKRChanged {
			continue
		}
		if !slices.Contains(rewrite.DefaultOrigins, c.Record.Origin) || !existing.Usable(c.Record, c.KR) {
			continue
		}
		u := ref.unit()
		zh, ok := patchNumbers(c.KR, u.KR, c.Record.ZH)
		if !ok || validate.HasHard(batch.CheckNewAt(p.in.Rules, ref.file.name, u.Pointer, u.KR, zh)) {
			continue
		}
		rec := cloneRecord(c.Record)
		rec.Key, rec.KRSha, rec.ZH = c.Key, ref.sha, zh
		p.records[c.Key] = rec
		delete(p.pending, c.Key)
		p.sum.NumberPatched++
	}
}

// numberMark stands for a number in the frame of a text (numbersOf). No text
// holds it.
const numberMark = "\x00"

// numbersOf splits s into its numbers (validate.Numbers: the runs of ASCII
// digits of its text, none of an [ID], a placeholder or a tag) and its frame,
// which is s with numberMark for every number. spans are where the numbers
// are in s.
func numbersOf(s string) (frame string, numbers []string, spans [][2]int) {
	spans = validate.Numbers(s)
	var b strings.Builder
	at := 0
	for _, n := range spans {
		b.WriteString(s[at:n[0]])
		b.WriteString(numberMark)
		numbers = append(numbers, s[n[0]:n[1]])
		at = n[1]
	}
	b.WriteString(s[at:])
	return b.String(), numbers, spans
}

// patchNumbers makes the translation of newKR from zh, the translation of
// oldKR, when the two KR texts differ in their numbers only and zh shows
// beyond doubt where each of them is. It reports false, and the unit is left
// to a translator, unless all of this holds:
//
//   - oldKR and newKR have the same frame (numbersOf, line ends written
//     alike): the same words, [ID]s, placeholders and tags around as many
//     numbers, of which at least one differs;
//   - zh has the very numbers of oldKR, written the same, in the same order.
//     A translation that writes a number in words or in another form, leaves
//     one out, adds one or moves one is not touched;
//   - every number of oldKR that occurs more than once became the same number
//     wherever it occurs. Then the number that replaces one depends on its
//     value only, and a translation that has two equal numbers the other way
//     round is still right.
//
// The translation is zh with each number replaced by the one newKR has in its
// place; nothing else of it changes.
func patchNumbers(oldKR, newKR, zh string) (string, bool) {
	oldFrame, from, _ := numbersOf(strings.ReplaceAll(oldKR, "\r\n", "\n"))
	newFrame, to, _ := numbersOf(strings.ReplaceAll(newKR, "\r\n", "\n"))
	if oldFrame != newFrame || len(from) != len(to) {
		return "", false
	}
	_, have, spans := numbersOf(zh)
	if len(have) != len(from) {
		return "", false
	}
	became := map[string]string{} // the new number, by the old one
	changed := false
	for i, n := range from {
		if have[i] != n {
			return "", false
		}
		if b, seen := became[n]; seen && b != to[i] {
			return "", false
		}
		became[n] = to[i]
		changed = changed || n != to[i]
	}
	if !changed {
		return "", false
	}
	var b strings.Builder
	at := 0
	for i, n := range spans {
		b.WriteString(zh[at:n[0]])
		b.WriteString(to[i])
		at = n[1]
	}
	b.WriteString(zh[at:])
	return b.String(), true
}
