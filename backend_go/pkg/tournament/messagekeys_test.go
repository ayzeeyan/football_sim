package tournament

import (
	"os"
	"strings"
	"testing"
)

// TestInboxCategoriesAreTheWireContract keeps the message-key layer honest:
// every PushInbox call site uses a MsgCategory constant (no raw string
// literals), and the category constants match the frontend's
// InboxItem.category union.
func TestInboxCategoriesAreTheWireContract(t *testing.T) {
	// 1. No raw category literals at call sites in production code.
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		raw, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if strings.Contains(string(raw), `PushInbox("`) {
			t.Fatalf("%s calls PushInbox with a raw category literal; use the MsgCategory constants", name)
		}
	}

	// 2. The catalogue covers every category the frontend union declares.
	want := map[string]bool{
		"match": false, "transfer": false, "wonderkid": false, "honour": false,
		"race": false, "cup": false, "system": false, "injury": false,
		"dugout": false, "youth": false, "nxgn": false, "milestone": false,
		"manager": false, "watch": false, "club": false,
	}
	if len(InboxCategories) != len(want) {
		t.Fatalf("catalogue has %d categories, want %d", len(InboxCategories), len(want))
	}
	seen := map[string]bool{}
	for _, cat := range InboxCategories {
		if seen[cat] {
			t.Fatalf("duplicate category %q in InboxCategories", cat)
		}
		seen[cat] = true
		if _, ok := want[cat]; !ok {
			t.Fatalf("category %q is not part of the frontend wire contract", cat)
		}
		want[cat] = true
	}
	for cat, covered := range want {
		if !covered {
			t.Fatalf("frontend category %q missing from InboxCategories", cat)
		}
	}

	// 3. Constants keep their persisted wire values.
	if MsgCategoryMatch != "match" || MsgCategoryMilestone != "milestone" || MsgCategoryNXGN != "nxgn" {
		t.Fatal("category constants must keep their persisted wire values")
	}
}
