package site

import (
	"slices"
	"sort"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/rules"
)

func TestNaturalLess(t *testing.T) {
	for _, tt := range []struct{ a, b string }{
		{"S2.json", "S10.json"},
		{"E003I3", "E003I10"},
		{"P10102", "P10205"},
		{"a", "b"},
		{"S1", "S1.json"}, // a prefix comes first
		{"", "a"},
		{"S9", "S10"},
		{"S9a", "S10"}, // the number decides before what follows it
		{"S01", "S1"},  // equal numbers: the bytes decide, so that the order is total
		{"A.json", "S1.json"},
		{"S1.json", "StoryData"}, // a digit against a letter: the bytes
		{"9", "a"},
	} {
		if !NaturalLess(tt.a, tt.b) {
			t.Errorf("NaturalLess(%q, %q) = false, want true", tt.a, tt.b)
		}
		if NaturalLess(tt.b, tt.a) {
			t.Errorf("NaturalLess(%q, %q) = true, want false", tt.b, tt.a)
		}
	}
	for _, s := range []string{"", "a", "S2.json", "E003I10", "007"} {
		if NaturalLess(s, s) {
			t.Errorf("NaturalLess(%q, %q) = true, want false", s, s)
		}
	}
}

// Sorting with NaturalLess gives one order whatever the order of the input.
func TestNaturalLessIsATotalOrder(t *testing.T) {
	want := []string{"S01", "S1", "S2", "S2a", "S10", "S10.json", "T1", "S100"}
	sort.Slice(want, func(i, j int) bool { return NaturalLess(want[i], want[j]) })
	for _, perm := range [][]int{{0, 1, 2, 3, 4, 5, 6, 7}, {7, 6, 5, 4, 3, 2, 1, 0}, {3, 1, 7, 0, 5, 2, 6, 4}} {
		got := make([]string, len(want))
		for i, p := range perm {
			got[i] = want[p]
		}
		sort.Slice(got, func(i, j int) bool { return NaturalLess(got[i], got[j]) })
		if !slices.Equal(got, want) {
			t.Errorf("sorted %v = %v, want %v", perm, got, want)
		}
	}
	if want[0] != "S01" || want[len(want)-1] != "T1" {
		t.Errorf("order = %v", want)
	}
}

func TestOrderFilesByFamilyThenName(t *testing.T) {
	r := &rules.Rules{Routing: rules.Routing{
		Families: []rules.FamilyRule{
			{ID: "B", Match: []string{"X*"}},
			{ID: "A"},
		},
		Priority: []string{"B", "A"},
	}}
	in := []string{"S10.json", "X1.json", "S2.json"}
	got := OrderFiles(in, r)
	if want := []string{"X1.json", "S2.json", "S10.json"}; !slices.Equal(got, want) {
		t.Errorf("OrderFiles = %v, want %v", got, want)
	}
	if want := []string{"S10.json", "X1.json", "S2.json"}; !slices.Equal(in, want) {
		t.Errorf("OrderFiles changed its input: %v", in)
	}

	// The priority decides, not the order of the families.
	r.Routing.Priority = []string{"A", "B"}
	if got, want := OrderFiles(in, r), []string{"S2.json", "S10.json", "X1.json"}; !slices.Equal(got, want) {
		t.Errorf("OrderFiles with priority A, B = %v, want %v", got, want)
	}
}

// A family the priority does not list comes after the listed ones, by id.
func TestOrderFilesUnlistedFamilies(t *testing.T) {
	r := &rules.Rules{Routing: rules.Routing{
		Families: []rules.FamilyRule{
			{ID: "Z", Match: []string{"Z*"}},
			{ID: "Y", Match: []string{"Y*"}},
			{ID: "A"},
		},
		Priority: []string{"A"},
	}}
	got := OrderFiles([]string{"Z1.json", "Y10.json", "A1.json", "Y2.json"}, r)
	if want := []string{"A1.json", "Y2.json", "Y10.json", "Z1.json"}; !slices.Equal(got, want) {
		t.Errorf("OrderFiles = %v, want %v", got, want)
	}
}

func TestOrderFilesEmpty(t *testing.T) {
	if got := OrderFiles(nil, testRules()); got == nil || len(got) != 0 {
		t.Errorf("OrderFiles(nil) = %#v, want an empty, non-nil slice", got)
	}
}
