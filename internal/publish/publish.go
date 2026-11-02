// Package publish prepares a Release of the language pack: it commits the
// build in the tw worktree, numbers the Release, writes its notes and says
// whether a Release is due, which it is only when the pack differs from that
// of the latest Release. Pushing the commit and creating the Release are left
// to the workflow.
package publish

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/tool-jx3/limbus-translate/internal/build"
	"github.com/tool-jx3/limbus-translate/internal/gitx"
	"github.com/tool-jx3/limbus-translate/internal/plan"
	"github.com/tool-jx3/limbus-translate/internal/raw"
	"github.com/tool-jx3/limbus-translate/internal/store"
)

const (
	// TagPrefix starts the name of every Release tag. It does not follow the
	// language directory of the pack (lang_dir in lct.yaml): Releases are
	// named zh-TW-YYYYMMDD.N.
	TagPrefix = "zh-TW-"
	// notesName is the file Run writes the release notes to, in outDir.
	notesName = "release-notes.md"

	dayLayout = "20060102"

	// licenseURL is the text of the licence of the translations.
	licenseURL = "https://creativecommons.org/licenses/by-nc-sa/4.0/"
)

// importedSource is the upstream source whose translations phase 1 imported
// into the records (origin paratranz). Every other source was only read, as a
// reference.
const importedSource = "paratranz"

// referenceOrder is the order the reference sources are listed in the notes;
// any other name follows, sorted.
var referenceOrder = []string{"ltm", "llc"}

// ErrNotBuilt marks a refusal because there is no usable build to publish: no
// .meta/build.json, one with no usable raw tag, or a zip that is missing or is
// not the one the build made. The fix is lct build. It is build.ErrNotBuilt, so
// that a worktree that was never built is the same error from build.ReadInfo.
var ErrNotBuilt = build.ErrNotBuilt

// ErrDirtyRules marks a refusal because the build was made from rules with
// changes that were not committed (its rules_commit ends in gitx.DirtySuffix): the
// Release could not be traced to the rules that made it. The fix is to commit
// the rules and run lct build again.
var ErrDirtyRules = errors.New("publish: the build was made from rules that were not committed")

var (
	// rawTagRe matches the tag of a raw snapshot, "raw-YYYYMMDD" or
	// "raw-YYYYMMDD.N" (see raw.NextTag), and captures the day.
	rawTagRe = regexp.MustCompile(`^` + regexp.QuoteMeta(raw.TagPrefix) + `(\d{8})(?:\.\d+)?$`)
	// releaseTagRe matches a Release tag and captures the day and N.
	releaseTagRe = regexp.MustCompile(`^` + regexp.QuoteMeta(TagPrefix) + `(\d{8})\.(\d+)$`)
)

// ReleaseTag returns the tag of the next Release of the pack built from the
// snapshot rawTag: "zh-TW-YYYYMMDD.N", with the day of the snapshot and N one
// more than the highest N of that day among existing (the tags that exist
// already; names that are not Release tags are ignored), or 1.
func ReleaseTag(rawTag string, existing []string) (string, error) {
	m := rawTagRe.FindStringSubmatch(rawTag)
	if m == nil {
		return "", fmt.Errorf("publish: raw tag %q is not %sYYYYMMDD or %sYYYYMMDD.N", rawTag, raw.TagPrefix, raw.TagPrefix)
	}
	day := m[1]
	if _, err := time.Parse(dayLayout, day); err != nil {
		return "", fmt.Errorf("publish: raw tag %q does not hold a date", rawTag)
	}
	highest := 0
	for _, name := range existing {
		m := releaseTagRe.FindStringSubmatch(name)
		if m == nil || m[1] != day {
			continue
		}
		// A number too large for an int cannot be reached by counting.
		if n, err := strconv.Atoi(m[2]); err == nil {
			highest = max(highest, n)
		}
	}
	return fmt.Sprintf("%s%s.%d", TagPrefix, day, highest+1), nil
}

// LastRelease returns the latest Release of the pack: of the Release tags
// (zh-TW-YYYYMMDD.N) in the repo of twRoot, the one with the latest day and,
// within the day, the highest N, compared as numbers (.10 is after .9). It
// returns "" when there is none. Names that are not Release tags are ignored.
func LastRelease(twRoot string) (string, error) {
	tags, err := gitx.Tags(twRoot, TagPrefix)
	if err != nil {
		return "", err
	}
	return latestRelease(tags), nil
}

// latestRelease returns the latest of the Release tags among names, or "".
func latestRelease(names []string) string {
	var last string
	var lastDay string
	lastN := -1
	for _, name := range names {
		m := releaseTagRe.FindStringSubmatch(name)
		if m == nil {
			continue
		}
		// A number too large for an int cannot be told from another: skipped,
		// as ReleaseTag does.
		n, err := strconv.Atoi(m[2])
		if err != nil {
			continue
		}
		// The days have eight digits each, so they compare as text.
		if last == "" || m[1] > lastDay || (m[1] == lastDay && n > lastN) {
			last, lastDay, lastN = name, m[1], n
		}
	}
	return last
}

// packDir is the directory of the pack in tw, the one the zips hold.
const packDir = "LimbusCompany_Data"

// checkLangDir refuses a language directory that is not a plain name.
func checkLangDir(langDir string) error {
	if langDir == "" || langDir == "." || strings.ContainsAny(langDir, `/\`) || !filepath.IsLocal(langDir) {
		return fmt.Errorf("publish: language directory %q is not a plain name", langDir)
	}
	return nil
}

// SamePack reports whether the revisions a and b of the tw repo hold the same
// pack: the same files, with the same content, below LimbusCompany_Data/.
// Info/version.json of the language directory langDir is left out, as it
// records the rules commit, the time and the cost of the build, which differ
// between two builds of the same translations; everything else, such as the
// licence beside it, is part of the pack. Files outside LimbusCompany_Data/,
// such as .meta/, are not. It is an error if a revision does not exist.
func SamePack(twRoot, a, b, langDir string) (bool, error) {
	if err := checkLangDir(langDir); err != nil {
		return false, err
	}
	left, err := packBlobs(twRoot, a, langDir)
	if err != nil {
		return false, err
	}
	right, err := packBlobs(twRoot, b, langDir)
	if err != nil {
		return false, err
	}
	return maps.Equal(left, right), nil
}

// packBlobs lists the files of the pack in rev, without the version file.
func packBlobs(twRoot, rev, langDir string) (map[string]string, error) {
	blobs, err := gitx.TreeBlobs(twRoot, rev, packDir)
	if err != nil {
		return nil, err
	}
	delete(blobs, packDir+"/Lang/"+langDir+"/Info/version.json")
	return blobs, nil
}

// verdict is what the Release rule says of the HEAD of tw.
type verdict struct {
	// head is the HEAD of tw, "" when it has no commit.
	head string
	// releaseNeeded: head has no Release tag, and its pack is not that of the
	// latest Release (or there is no Release).
	releaseNeeded bool
	// sameAs is the latest Release when the pack of head is its pack.
	sameAs string
	// tag is the latest Release tag on head, if it has one.
	tag string
}

// decide applies the rule to the HEAD of the tw worktree. It reads commits,
// tags and trees; no file of the worktree.
func decide(twRoot, langDir string) (verdict, error) {
	head, err := gitx.Head(twRoot)
	if err != nil || head == "" {
		return verdict{}, err
	}
	d := verdict{head: head}
	onHead, err := gitx.TagsAt(twRoot, TagPrefix)
	if err != nil {
		return verdict{}, err
	}
	d.tag = latestRelease(onHead)
	last, err := LastRelease(twRoot)
	if err != nil {
		return verdict{}, err
	}
	if last != "" {
		same, err := SamePack(twRoot, head, last, langDir)
		if err != nil {
			return verdict{}, err
		}
		if same {
			d.sameAs = last
		}
	}
	// A commit with a Release tag has its Release; one without it, that an
	// earlier run made and whose Release did not follow, is still due it if
	// the pack is not the one the last Release has. A tag of the prefix that
	// is not a Release tag (zh-TW-latest) is no Release.
	d.releaseNeeded = d.tag == "" && d.sameAs == ""
	return d, nil
}

// Check is what CheckTW says of the HEAD of tw. The JSON keys are read by the
// workflow.
type Check struct {
	// TWCommit is the HEAD of tw, "" when it has no commit.
	TWCommit string `json:"tw_commit"`
	// ReleaseNeeded is whether a Release must be made from TWCommit, as for
	// Result.
	ReleaseNeeded bool `json:"release_needed"`
	// SameAs is the latest Release when the pack of TWCommit is its pack, "" if
	// not (always printed).
	SameAs string `json:"same_as"`
	// Tag is the latest Release tag on TWCommit: the Release it has; "" if it
	// has none. It is not the tag of the next Release, which only a build
	// (Result.Tag) can number.
	Tag string `json:"tag"`
}

func (c Check) String() string {
	switch {
	case c.TWCommit == "":
		return "tw has no commit: no Release needed"
	case c.ReleaseNeeded:
		return fmt.Sprintf("tw at %s: release needed", c.TWCommit)
	case c.Tag != "":
		return fmt.Sprintf("tw at %s: released as %s", c.TWCommit, c.Tag)
	default:
		return fmt.Sprintf("tw at %s: release not needed, the pack is the one of %s", c.TWCommit, c.SameAs)
	}
}

// CheckTW says whether a Release would be due for the commit tw is at, as Run
// says it after a build, but reading only what git holds: the HEAD, the tags
// and the trees of tw. It commits nothing and writes nothing, and needs no
// build: neither .meta/build.json nor the zips. The files of the worktree,
// committed or not, are not read. langDir is as for Run.
func CheckTW(twRoot, langDir string) (*Check, error) {
	if err := checkLangDir(langDir); err != nil {
		return nil, err
	}
	d, err := decide(twRoot, langDir)
	if err != nil {
		return nil, err
	}
	return &Check{TWCommit: d.head, ReleaseNeeded: d.releaseNeeded, SameAs: d.sameAs, Tag: d.tag}, nil
}

// Result is what Run did. The JSON keys are read by the workflow.
type Result struct {
	// TWCommit is the HEAD of the tw worktree: the commit the Release is made
	// from. Changed is whether this run made it; if not, it was there already.
	TWCommit string `json:"tw_commit"`
	Changed  bool   `json:"changed"`
	// Tag is the tag the Release should get.
	Tag string `json:"tag"`
	// NotesPath is the file holding the release notes.
	NotesPath string `json:"notes_path"`
	// Assets are the files to attach to the Release: the zip with the text
	// only, then the zip that also holds the font.
	Assets []string `json:"assets"`
	// ReleaseNeeded is whether a Release must be made from TWCommit: it has no
	// Release tag, and the pack in it is not the pack of the latest Release
	// (see LastRelease), or there is none yet. A commit that has no Release
	// tag because an earlier run committed it and the Release did not follow
	// is still due its Release.
	ReleaseNeeded bool `json:"release_needed"`
	// SameAs is the latest Release, when the pack of TWCommit is the pack of
	// that Release (see SamePack); it is empty when the pack differs or there
	// is no Release yet, and the key is left out then.
	SameAs string `json:"same_as,omitempty"`
}

func (r Result) String() string {
	state := "unchanged"
	if r.Changed {
		state = "committed"
	}
	release := "not needed"
	if r.ReleaseNeeded {
		release = "needed"
	} else if r.SameAs != "" {
		release += ", the pack is the one of " + r.SameAs
	}
	s := fmt.Sprintf("%s: tw %s at %s, release %s\nnotes: %s\nassets:", r.Tag, state, r.TWCommit, release, r.NotesPath)
	for _, a := range r.Assets {
		s += "\n  " + a
	}
	return s
}

// Run commits the build that lct build left in the tw worktree and prepares
// the Release of it: the tag, the notes (outDir/release-notes.md) and the list
// of the zips to attach. root is the repo, whose tags number the Releases;
// twRoot is the tw worktree; outDir is the directory lct build wrote the zips
// to and langDir the language directory it named them after (lang_dir in
// lct.yaml), which must be a plain name, as for build.ZipPaths.
//
// Everything that can be refused is checked before tw is touched: a worktree
// that was not built, a raw tag that does not give a Release tag, or a zip that
// is missing or is not the one the build made (its sha256 is not the one in
// build.json) is ErrNotBuilt; a build made from rules that were not committed
// is ErrDirtyRules. If the build is what tw already holds, nothing is
// committed and TWCommit is the HEAD of tw. The tag is the next free one of
// the day of the snapshot, so until a Release with it exists (its tag is in
// the repo), running again gives the same tag. Result.ReleaseNeeded says
// whether a Release is due for TWCommit: only when its pack differs from that
// of the latest Release. tw is committed and the notes are written whatever it
// says.
func Run(root, twRoot, outDir, langDir string) (*Result, error) {
	if err := checkLangDir(langDir); err != nil {
		return nil, err
	}
	info, err := build.ReadInfo(twRoot)
	if err != nil {
		return nil, err
	}
	if strings.HasSuffix(info.RulesCommit, gitx.DirtySuffix) {
		return nil, fmt.Errorf("%w (rules_commit %q); commit the rules and run lct build again", ErrDirtyRules, info.RulesCommit)
	}
	// What the snapshot changed, for the notes; a plan of another snapshot
	// says nothing about this build.
	sum, err := build.ReadPlan(twRoot, info.RawTag)
	if err != nil {
		return nil, err
	}
	existing, err := gitx.Tags(root, TagPrefix)
	if err != nil {
		return nil, err
	}
	tag, err := ReleaseTag(info.RawTag, existing)
	if err != nil {
		return nil, fmt.Errorf("publish: %w: .meta/build.json: %v", ErrNotBuilt, err)
	}
	// The raw tag is a plain name now, safe to put in a path.
	text, complete := build.ZipPaths(build.Options{OutDir: outDir, LangDir: langDir, RawTag: info.RawTag})
	assets := []string{text, complete}
	for _, name := range assets {
		if err := checkZip(name, info.Zips); err != nil {
			return nil, err
		}
	}

	// No writer of the store, such as a batch, may change tw between the files
	// being staged and the commit being made.
	var sha string
	var changed bool
	err = store.Open(twRoot).WithLock(func() error {
		var err error
		sha, changed, err = gitx.CommitAll(twRoot, fmt.Sprintf("build: %s (rules %s)", info.RawTag, info.RulesCommit))
		return err
	})
	if err != nil {
		return nil, err
	}

	// Whether the commit, made now or by an earlier run whose Release did not
	// follow, needs a Release: whether its pack differs from the last one.
	d, err := decide(twRoot, langDir)
	if err != nil {
		return nil, err
	}

	notesPath := filepath.Join(outDir, notesName)
	if err := os.WriteFile(notesPath, Notes(info, sum, tag, langDir), 0o644); err != nil {
		return nil, err
	}
	return &Result{TWCommit: sha, Changed: changed, Tag: tag, NotesPath: notesPath, Assets: assets, ReleaseNeeded: d.releaseNeeded, SameAs: d.sameAs}, nil
}

// checkZip refuses the zip at name unless it is a file with the sha256 that
// zips, the checksums of the build, give for its name.
func checkZip(name string, zips map[string]string) error {
	fi, err := os.Stat(name)
	if err == nil && !fi.Mode().IsRegular() || errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("publish: %w: %s is missing; run lct build first", ErrNotBuilt, name)
	}
	if err != nil {
		return err
	}
	want, ok := zips[filepath.Base(name)]
	if !ok {
		return fmt.Errorf("publish: %w: .meta/build.json has no checksum for %s; run lct build first", ErrNotBuilt, filepath.Base(name))
	}
	got, err := build.FileSHA256(name)
	if err != nil {
		return err
	}
	if !strings.EqualFold(got, want) {
		return fmt.Errorf("publish: %w: %s has sha256 %s, but the build made one with %s; run lct build first", ErrNotBuilt, name, got, want)
	}
	return nil
}

// originLabels say in a few words what each source of the shown text is: the
// origins of the translations, and the stand-in text (build.Report.Shown).
var originLabels = map[string]string{
	store.OriginParatranz: "第一階段匯入的社群譯文",
	store.OriginClaude:    "Claude 翻譯",
	store.OriginHuman:     "人工調整",
	store.OriginCopy:      "不需翻譯，保留原文",
	build.ShownStandIn:    "尚未翻譯、翻譯失敗或破壞原文結構，暫用英文",
}

// Notes renders the release notes of the Release called tag, built from info
// and from sum, the plan of the snapshot (nil when there is none), in
// Markdown. langDir is the language directory of the pack (lang_dir in
// lct.yaml), the language the install text tells to choose. The tag is the
// title: the Release name must not hold the name of the product that
// translated the pack, which the text below it may.
//
// The notes are short (spec 14.1): the parts come in this order: the title;
// what the pack is; how to install it; the raw version and what changed in it;
// the rules version; the share of each source of the translations and how many
// units still show English; the licence and the sources. The report of the
// build, with its tables and lists, is not repeated.
func Notes(info build.Info, sum *plan.Summary, tag, langDir string) []byte {
	var b strings.Builder
	rep := info.Report

	fmt.Fprintf(&b, "# %s\n\n", tag)
	b.WriteString("Limbus Company 臺灣繁體中文語言包，使用 Claude Code 翻譯。\n\n")

	b.WriteString("## 安裝方式\n\n")
	fmt.Fprintf(&b, "把 zip 內的 `LimbusCompany_Data/` 解壓到遊戲資料夾，在遊戲設定選擇 %s。\n\n", langDir)
	b.WriteString("zip 有兩種：text 只有文字；complete 另含字型。\n\n")

	b.WriteString("## 原文版本\n\n")
	fmt.Fprintf(&b, "- raw tag：`%s`\n- token：`%s`\n", info.RawTag, info.RawToken)
	if sum != nil && sum.RawTag == info.RawTag && sum.PrevRawTag != sum.RawTag {
		c := sum.Changes
		fmt.Fprintf(&b, "- 與上一版 `%s` 相比：新增 %d、修改 %d、刪除 %d 個單位\n", sum.PrevRawTag, c.New, c.Changed, c.Deleted)
	}
	b.WriteString("\n")

	b.WriteString("## 規則版本\n\n")
	fmt.Fprintf(&b, "- rules commit：`%s`\n\n", info.RulesCommit)

	b.WriteString("## 譯文來源\n\n")
	// Each unit counts once, in what it shows (build.Report.Shown), so the
	// shares add up to the whole.
	total := 0
	for _, n := range rep.Shown {
		total += n
	}
	if total > 0 {
		fmt.Fprintf(&b, "各來源占全部 %d 個單位的比例：\n\n", total)
		for _, origin := range slices.Sorted(maps.Keys(rep.Shown)) {
			n := rep.Shown[origin]
			fmt.Fprintf(&b, "- %s %.1f%%", origin, 100*float64(n)/float64(total))
			if label, ok := originLabels[origin]; ok {
				fmt.Fprintf(&b, "：%s（%d 個單位）", label, n)
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	// An older build record did not count the stand-ins; the pending units are
	// the least that show English.
	english := max(rep.StandIn, rep.Pending)
	fmt.Fprintf(&b, "仍顯示英文：%d 個單位", english)
	if english > 0 {
		fmt.Fprintf(&b, "（尚未翻譯 %d；翻譯失敗或破壞原文結構而暫用英文 %d）", rep.Pending, english-rep.Pending)
	}
	b.WriteString("\n\n")

	b.WriteString("## 授權與來源\n\n")
	// Sources are credited only when records of them went into the pack.
	imported := rep.Origins[store.OriginParatranz] > 0
	if imported {
		fmt.Fprintf(&b, "- 譯文採用 [CC BY-NC-SA 4.0](%s) 授權，含 Paratranz 社群譯文（第一階段匯入）。\n", licenseURL)
		if ref, ok := info.Upstream[importedSource]; ok {
			fmt.Fprintf(&b, "- 匯入：%s: %s\n", importedSource, ref)
		}
		if names := referenceNames(info.Upstream); len(names) > 0 {
			b.WriteString("- 第一階段的參考譯文（未匯入）：\n")
			for _, name := range names {
				fmt.Fprintf(&b, "  - %s: %s\n", name, info.Upstream[name])
			}
		}
	} else {
		fmt.Fprintf(&b, "- 譯文採用 [CC BY-NC-SA 4.0](%s) 授權。\n", licenseURL)
	}
	b.WriteString("- 遊戲原文的著作權屬 Project Moon。\n")
	b.WriteString("- 字型 Sarasa Gothic（SIL OFL），在 complete 版內。\n")
	return []byte(b.String())
}

// referenceNames returns the names of the reference sources in the order the
// notes list them: referenceOrder first, then any others sorted. The imported
// source is not one of them.
func referenceNames(upstream map[string]string) []string {
	var names []string
	for _, name := range referenceOrder {
		if _, ok := upstream[name]; ok {
			names = append(names, name)
		}
	}
	for _, name := range slices.Sorted(maps.Keys(upstream)) {
		if name != importedSource && !slices.Contains(referenceOrder, name) {
			names = append(names, name)
		}
	}
	return names
}
