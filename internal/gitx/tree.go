package gitx

import (
	"fmt"
	"strings"
)

// TreeBlobs lists the files below path in the tree of rev, as a map from the
// path of each file, relative to the root of the repo, to the id of its blob
// (git ls-tree -r). Two revisions hold the same bytes in a file exactly when
// the blob ids are equal, so comparing the maps compares the trees without
// reading a file. rev is a tag, a branch or a commit; it is an error if it
// does not exist. A path with no file below it gives an empty map.
func TreeBlobs(dir, rev, path string) (map[string]string, error) {
	if strings.HasPrefix(rev, "-") {
		return nil, fmt.Errorf("gitx: tree: revision %q looks like an option", rev)
	}
	// -z: names are not quoted, whatever they hold.
	out, err := git(dir, "ls-tree", "-r", "-z", "--full-tree", rev, "--", path)
	if err != nil {
		return nil, err
	}
	blobs := map[string]string{}
	for _, entry := range strings.Split(out, "\x00") {
		if entry == "" {
			continue
		}
		// "<mode> <type> <id>\t<path>"
		meta, name, ok := strings.Cut(entry, "\t")
		fields := strings.Fields(meta)
		if !ok || len(fields) != 3 {
			return nil, fmt.Errorf("gitx: tree: unexpected line of git ls-tree: %q", entry)
		}
		blobs[name] = fields[2]
	}
	return blobs, nil
}
