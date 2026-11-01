package raw

import (
	"fmt"
	"path/filepath"

	"github.com/tool-jx3/limbus-translate/internal/gitx"
	"github.com/tool-jx3/limbus-translate/internal/pack"
)

// OpenKRAt opens the KR pack of the snapshot called tag (such as
// "raw-20261029"), which is how lct plan reads the previous version of the
// original text. The tag is looked up in the repository of worktree, and the
// worktree itself is not touched: kr/ of the tag is exported into dir/kr, so
// dir is where the pack's files live, and the caller removes it once the pack
// is no longer used. dir should be new or empty.
//
// A tag that does not exist is an error naming it; so is a branch or a commit
// given in its place.
func OpenKRAt(worktree, tag, dir string) (*pack.Pack, error) {
	if err := gitx.Archive(worktree, "refs/tags/"+tag, []string{"kr"}, dir); err != nil {
		return nil, fmt.Errorf("raw: read the KR pack of tag %s: %w", tag, err)
	}
	return pack.Open(filepath.Join(dir, "kr"), "KR_")
}
