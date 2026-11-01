package validate

import (
	_ "embed"
	"strings"
	"sync"
	"unicode/utf8"
)

// simplified.txt is generated from OpenCC's dictionaries by
// tools/gen-simplified, which documents the rules and how to run it.
//
//go:embed simplified.txt
var simplifiedData string

// simplifiedChars returns the set of characters only Simplified Chinese uses.
var simplifiedChars = sync.OnceValue(func() map[rune]bool {
	set := map[rune]bool{}
	for _, line := range strings.Split(simplifiedData, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || utf8.RuneCountInString(line) != 1 {
			continue
		}
		c, _ := utf8.DecodeRuneInString(line)
		set[c] = true
	}
	return set
})
