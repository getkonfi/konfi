package pkg

import (
	"testing"
)

// helper to build sections concisely
func mkField(key, label, desc string) Field {
	return Field{Key: key, Label: label, Description: desc}
}

func TestSearch_KeyMatchRanksAboveDescription(t *testing.T) {
	sections := []Section{{
		Name: "general",
		Fields: []Field{
			{Key: "opacity", Label: "Opacity", Description: "window transparency level"},
			{Key: "window-mode", Label: "Window Mode", Description: "set the opacity of the window"},
		},
	}}

	idx := NewSearchIndex(sections)
	results := idx.Search("opacity")

	if len(results) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(results))
	}

	// field with "opacity" in key should rank first
	first := results[0]
	if sections[first.SectionIdx].Fields[first.FieldIdx].Key != "opacity" {
		t.Errorf("expected key-match field first, got key=%q",
			sections[first.SectionIdx].Fields[first.FieldIdx].Key)
	}

	if results[0].Score <= results[1].Score {
		t.Errorf("key match score (%.4f) should exceed description-only match (%.4f)",
			results[0].Score, results[1].Score)
	}
}

func TestSearch_SynonymExpansion(t *testing.T) {
	sections := []Section{{
		Name: "window",
		Fields: []Field{
			{Key: "background-opacity", Label: "Background Opacity", Description: "controls window opacity"},
		},
	}}

	idx := NewSearchIndex(sections)
	results := idx.Search("transparency")

	if len(results) == 0 {
		t.Fatal("synonym search for 'transparency' should find fields with 'opacity'")
	}

	got := sections[results[0].SectionIdx].Fields[results[0].FieldIdx]
	if got.Key != "background-opacity" {
		t.Errorf("expected background-opacity, got %q", got.Key)
	}
}

func TestSearch_MultiWordQuery(t *testing.T) {
	sections := []Section{{
		Name: "appearance",
		Fields: []Field{
			{Key: "font-size", Label: "Font Size", Description: "size of the font in points"},
			{Key: "font-family", Label: "Font Family", Description: "primary font to use"},
			{Key: "window-size", Label: "Window Size", Description: "initial window dimensions"},
		},
	}}

	idx := NewSearchIndex(sections)
	results := idx.Search("font size")

	if len(results) == 0 {
		t.Fatal("expected results for 'font size'")
	}

	// font-size should rank highest — both terms match the key
	top := sections[results[0].SectionIdx].Fields[results[0].FieldIdx]
	if top.Key != "font-size" {
		t.Errorf("expected font-size to rank first, got %q", top.Key)
	}
}

func TestSearch_IDFRareTermScoresHigher(t *testing.T) {
	// "color" appears in many fields, "ligature" is rare — a match on
	// the rare term should score higher due to inverse document frequency
	fields := []Field{
		{Key: "color-primary", Label: "Primary Color", Description: "main color"},
		{Key: "color-secondary", Label: "Secondary Color", Description: "accent color"},
		{Key: "color-background", Label: "Background Color", Description: "bg color"},
		{Key: "color-foreground", Label: "Foreground Color", Description: "fg color"},
		{Key: "color-cursor", Label: "Cursor Color", Description: "cursor color"},
		{Key: "ligature", Label: "Ligatures", Description: "enable font ligature rendering"},
	}
	sections := []Section{{Name: "visual", Fields: fields}}

	idx := NewSearchIndex(sections)

	colorResults := idx.Search("color")
	ligatureResults := idx.Search("ligature")

	if len(colorResults) == 0 || len(ligatureResults) == 0 {
		t.Fatal("expected results for both queries")
	}

	// the top ligature hit should outscore the top color hit
	if ligatureResults[0].Score <= colorResults[0].Score {
		t.Errorf("rare term 'ligature' top score (%.4f) should exceed common term 'color' top score (%.4f)",
			ligatureResults[0].Score, colorResults[0].Score)
	}
}

func TestSearch_EmptyQuery(t *testing.T) {
	sections := []Section{{
		Name: "general",
		Fields: []Field{
			{Key: "foo", Label: "Foo", Description: "bar"},
		},
	}}

	idx := NewSearchIndex(sections)

	if results := idx.Search(""); results != nil {
		t.Errorf("empty query should return nil, got %d results", len(results))
	}
	if results := idx.Search("   "); results != nil {
		t.Errorf("whitespace-only query should return nil, got %d results", len(results))
	}
}

func TestSearch_SectionNameContribution(t *testing.T) {
	sections := []Section{
		{
			Name: "keybindings",
			Fields: []Field{
				{Key: "copy", Label: "Copy", Description: "copy selected text"},
				{Key: "paste", Label: "Paste", Description: "paste from clipboard"},
			},
		},
		{
			Name: "appearance",
			Fields: []Field{
				{Key: "theme", Label: "Theme", Description: "color scheme to use"},
			},
		},
	}

	idx := NewSearchIndex(sections)
	results := idx.Search("keybindings")

	if len(results) == 0 {
		t.Fatal("searching section name should return results")
	}

	// all top results should come from the keybindings section
	for _, r := range results {
		if sections[r.SectionIdx].Name != "keybindings" {
			continue
		}
		// found at least one from the right section
		return
	}
	t.Error("expected at least one result from the 'keybindings' section")
}

func TestExpandSynonyms(t *testing.T) {
	syns := ExpandSynonyms("transparency")
	if len(syns) == 0 {
		t.Fatal("expected synonyms for 'transparency'")
	}

	found := false
	for _, s := range syns {
		if s == "opacity" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'opacity' in synonyms for 'transparency', got %v", syns)
	}

	// bidirectional
	syns2 := ExpandSynonyms("opacity")
	found = false
	for _, s := range syns2 {
		if s == "transparency" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'transparency' in synonyms for 'opacity', got %v", syns2)
	}

	// unknown term returns nil
	if got := ExpandSynonyms("xyznonexistent"); got != nil {
		t.Errorf("expected nil for unknown term, got %v", got)
	}
}
