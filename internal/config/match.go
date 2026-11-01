package config

import (
	"path"
	"strings"
)

// MatchAny reports whether file, a slash-separated pack file name such as
// "BgmLyrics/BgmLyrics_Credit7.json", matches any of patterns, the form
// lct.yaml's frozen list takes. A pattern ending in "/**" matches every file
// under that directory, however deeply nested; the directory part is literal.
// Any other pattern is a path.Match glob. A malformed pattern matches nothing.
func MatchAny(patterns []string, file string) bool {
	for _, p := range patterns {
		if dir, ok := strings.CutSuffix(p, "/**"); ok {
			if dir != "" && strings.HasPrefix(file, dir+"/") {
				return true
			}
			continue
		}
		if ok, _ := path.Match(p, file); ok {
			return true
		}
	}
	return false
}
