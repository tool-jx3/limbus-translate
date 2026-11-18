package raw

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/gitx"
	"github.com/tool-jx3/limbus-translate/internal/pack"
)

// OpenKRAt opens the KR pack of the snapshot called tag (such as
// "raw-20261029"), which is how lct plan reads the previous version of the
// original text: OpenAt for "kr".
func OpenKRAt(worktree, tag, dir string) (*pack.Pack, error) {
	return OpenAt(worktree, tag, "kr", dir)
}

// OpenAt opens the pack of lang ("kr", "en" or "jp") of the snapshot called
// tag. The tag is looked up in the repository of worktree, and the worktree
// itself is not touched: <lang>/ of the tag is exported into dir/<lang>, so
// dir is where the pack's files live, and the caller removes it once the pack
// is no longer used. dir should be new or empty.
//
// A tag that does not exist is an error naming it; so is a branch or a commit
// given in its place, and a language that is none of the three.
func OpenAt(worktree, tag, lang, dir string) (*pack.Pack, error) {
	if !isLang(lang) {
		return nil, fmt.Errorf("raw: %q is not a language of the snapshots (kr, en, jp)", lang)
	}
	if err := gitx.Archive(worktree, "refs/tags/"+tag, []string{lang}, dir); err != nil {
		return nil, fmt.Errorf("raw: read the %s pack of tag %s: %w", strings.ToUpper(lang), tag, err)
	}
	return pack.Open(filepath.Join(dir, lang), strings.ToUpper(lang)+"_")
}
