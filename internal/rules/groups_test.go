package rules_test

import (
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/rules"
)

func TestFileNode(t *testing.T) {
	tests := []struct{ in, top, sub string }{
		{"StoryData/S1037B.json", "StoryData/", "StoryData/S"},
		{"StoryData/1D101A.json", "StoryData/", "StoryData/D"},
		{"StoryData/ES050A.json", "StoryData/", "StoryData/ES"},
		{"StoryData/RPG1.json", "StoryData/", "StoryData/RPG"},
		{"PersonalityVoiceDlg/Voice_Heathcliff_RCorp4_10703.json", "PersonalityVoiceDlg/", "PersonalityVoiceDlg/Voice_Heathcliff"},
		{"RPGSystem/rpg-loc-swarm-mob-floor-3-b.json", "RPGSystem/", "RPGSystem/rpg_loc"},
		{"Skills_Enemy-mowe.json", "Skills", "Skills_Enemy"},
		{"Skills_Abnormality-a1c9p1.json", "Skills", "Skills_Abnormality"},
		{"Bufs-walpu7.json", "Bufs", ""},
		{"Bufs_Mirror5.json", "Bufs", "Bufs_Mirror"},
		{"StageNode91-a1c5.json", "StageNode", ""},
		{"MirrorDungeonUI_5.json", "MirrorDungeonUI", ""},
		{"Walpu6EventText.json", "Walpu", "Walpu6EventText"},
	}
	for _, tt := range tests {
		got := rules.FileNode(tt.in)
		if got.Top != tt.top || got.Sub != tt.sub {
			t.Errorf("FileNode(%q) = {%q, %q}, want {%q, %q}", tt.in, got.Top, got.Sub, tt.top, tt.sub)
		}
	}
}

func TestFileNodeOddNames(t *testing.T) {
	tests := []struct{ in, top, sub string }{
		{"kr.json", "kr", ""},
		{"reward.json", "reward", ""},
		{"Ch1.json", "Ch", ""},
		{"9Lives.json", "9Lives", ""},
	}
	for _, tt := range tests {
		got := rules.FileNode(tt.in)
		if got.Top != tt.top || got.Sub != tt.sub {
			t.Errorf("FileNode(%q) = {%q, %q}, want {%q, %q}", tt.in, got.Top, got.Sub, tt.top, tt.sub)
		}
	}
}

func TestFileTag(t *testing.T) {
	s := func(v string) *string { return &v }
	r := &rules.Rules{Characters: []rules.Character{
		{ID: "heathcliff", Name: "希斯克里夫", Code: s("07"), FileNames: []string{"Heathcliff"}},
		{ID: "meursault", Name: "默爾索", Code: s("05"), FileNames: []string{"Meursalut"}},
		{ID: "gregor", Name: "格雷戈爾", Code: s("12"), FileNames: []string{"Gregor"}},
	}}
	tests := []struct {
		in   string
		want rules.Tag
	}{
		{"StoryData/P10702.json", rules.Tag{Kind: rules.TagCharacter, Value: "heathcliff", Label: "希斯克里夫"}},
		{"PersonalityVoiceDlg/Voice_Meursalut_X_10501.json", rules.Tag{Kind: rules.TagCharacter, Value: "meursault", Label: "默爾索"}},
		{"Skills_personality-07.json", rules.Tag{Kind: rules.TagCharacter, Value: "heathcliff", Label: "希斯克里夫"}},
		{"Skills_Ego_Personality-12.json", rules.Tag{Kind: rules.TagCharacter, Value: "gregor", Label: "格雷戈爾"}},
		{"StoryData/S1037B.json", rules.Tag{Kind: rules.TagChapter, Value: "10", Label: "第 10 章"}},
		{"StoryData/S001A.json", rules.Tag{Kind: rules.TagChapter, Value: "0", Label: "第 0 章"}},
		{"StoryData/5D302B.json", rules.Tag{Kind: rules.TagChapter, Value: "5", Label: "第 5 章"}},
		{"AbEvents-a1c5p3.json", rules.Tag{Kind: rules.TagChapter, Value: "5", Label: "第 5 章"}},
		{"Skills_Abnormality-tkt-re.json", rules.Tag{Kind: rules.TagEvent, Value: "tkt", Label: "tkt"}},
		{"Bufs-walpu7.json", rules.Tag{Kind: rules.TagEvent, Value: "walpu7", Label: "walpu7"}},
		{"Skills_personality-x1p1c1.json", rules.Tag{Kind: rules.TagEvent, Value: "x1p1c1", Label: "x1p1c1"}},
		{"Skills_personality-a1c9p1.json", rules.Tag{Kind: rules.TagChapter, Value: "9", Label: "第 9 章"}},
	}
	for _, tt := range tests {
		got, ok := r.FileTag(tt.in)
		if !ok || got != tt.want {
			t.Errorf("FileTag(%q) = %+v, %v; want %+v", tt.in, got, ok, tt.want)
		}
	}
	for _, in := range []string{"Bufs.json", "StageNode91-1.json", "StoryData/E302B.json"} {
		if got, ok := r.FileTag(in); ok {
			t.Errorf("FileTag(%q) = %+v, want none", in, got)
		}
	}
}
