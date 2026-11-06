package rules

import (
	"regexp"
	"strings"
)

// Node is the place of a file in the group tree. file is the relative path
// without language prefix, as Family takes it.
type Node struct {
	Top string `json:"top"`           // "StoryData/", "Skills"
	Sub string `json:"sub,omitempty"` // "StoryData/S", "Skills_Enemy"; "" when none
}

var (
	storySubRe    = regexp.MustCompile(`^\d*([A-Za-z]+)`)
	leadLettersRe = regexp.MustCompile(`^[A-Za-z]+`)
	trailSepRe    = regexp.MustCompile(`[\d_]+$`)
)

// FileNode derives the top and sub node of a file from its path alone.
func FileNode(file string) Node {
	stem := strings.TrimSuffix(file, ".json")
	if dir, name, ok := strings.Cut(stem, "/"); ok {
		n := Node{Top: dir + "/"}
		if dir == "StoryData" {
			if m := storySubRe.FindStringSubmatch(name); m != nil {
				n.Sub = n.Top + m[1]
			}
			return n
		}
		parts := strings.FieldsFunc(name, func(r rune) bool { return r == '_' || r == '-' })
		if len(parts) > 2 {
			parts = parts[:2]
		}
		if len(parts) > 0 {
			n.Sub = n.Top + strings.Join(parts, "_")
		}
		return n
	}
	head, _, _ := strings.Cut(stem, "-")
	top := leadLettersRe.FindString(stem)
	if top == "" {
		return Node{Top: head}
	}
	sub := trailSepRe.ReplaceAllString(head, "")
	if sub == top {
		sub = ""
	}
	return Node{Top: top, Sub: sub}
}

// Kinds of third-level tag.
const (
	TagCharacter = "character"
	TagChapter   = "chapter"
	TagEvent     = "event"
)

// Tag is the third-level mark of a file: a character, a chapter or an event.
type Tag struct {
	Kind  string `json:"kind"`  // TagCharacter, TagChapter or TagEvent
	Value string `json:"value"` // "heathcliff", "10", "walpu6"
	Label string `json:"label"` // "希斯克里夫", "第 10 章", "walpu6"
}

var (
	storyChapterRe = regexp.MustCompile(`^StoryData/S(\d+)\d{2}[A-Za-z]?$`)
	storyDChapter  = regexp.MustCompile(`^StoryData/(\d+)D`)
	rootChapterRe  = regexp.MustCompile(`^a1c(\d+)`)
	eventRe        = regexp.MustCompile(`^[^-_]+`)
)

func chapterTag(digits string) Tag {
	n := strings.TrimLeft(digits, "0")
	if n == "" {
		n = "0"
	}
	return Tag{Kind: TagChapter, Value: n, Label: "第 " + n + " 章"}
}

// FileTag derives the third-level tag of a file from its path alone: the
// character first, then the chapter, then the event. ok is false when the
// file has none.
func (r *Rules) FileTag(file string) (Tag, bool) {
	if c, ok := r.CharacterByFile(file); ok {
		return Tag{Kind: TagCharacter, Value: c.ID, Label: c.Name}, true
	}
	stem := strings.TrimSuffix(file, ".json")
	if strings.Contains(stem, "/") {
		for _, re := range []*regexp.Regexp{storyChapterRe, storyDChapter} {
			if m := re.FindStringSubmatch(stem); m != nil {
				return chapterTag(m[1]), true
			}
		}
		return Tag{}, false
	}
	_, after, ok := strings.Cut(stem, "-")
	if !ok {
		return Tag{}, false
	}
	if m := rootChapterRe.FindStringSubmatch(after); m != nil {
		return chapterTag(m[1]), true
	}
	ev := eventRe.FindString(after)
	if ev == "" || strings.Trim(ev, "0123456789") == "" {
		return Tag{}, false
	}
	return Tag{Kind: TagEvent, Value: ev, Label: ev}, true
}
