package pkg

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/eminert/konfi/pkg/parser"
)

// e2eConfigSetup creates a temp file with initial content and returns a ConfigFile.
func e2eConfigSetup(t *testing.T, content string) (cf *ConfigFile, path string) {
	t.Helper()
	dir := t.TempDir()
	path = filepath.Join(dir, "test.conf")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	fp := NewFilePersister(path)
	cf, err := NewConfigFile(context.Background(), fp)
	if err != nil {
		t.Fatal(err)
	}
	return cf, path
}

// TestE2EFlatConfigWorkflow exercises the full ConfigFile lifecycle
// with a flat key-value parser (like ghostty/kitty).
func TestE2EFlatConfigWorkflow(t *testing.T) {
	initial := "# ghostty config\nfont-size = 14\nbackground = 282828\n\n"
	cf, path := e2eConfigSetup(t, initial)
	p := &parser.FlatParser{Split: parser.SplitEquals, Format: parser.FormatEquals}

	// initial load
	if cf.Dirty() {
		t.Fatal("new configfile should not be dirty")
	}

	// read value
	content := cf.Content()
	val, ok := p.FindValue(content, "font-size")
	if !ok || val != "14" {
		t.Fatalf("FindValue(font-size) = %q, %v", val, ok)
	}

	// modify value
	newContent, err := p.SetValue(content, "font-size", "16")
	if err != nil {
		t.Fatal(err)
	}
	cf.SetContent(newContent)
	if !cf.Dirty() {
		t.Fatal("should be dirty after modification")
	}

	// save to disk
	if err := cf.Save(context.Background()); err != nil {
		t.Fatal(err)
	}
	if cf.Dirty() {
		t.Fatal("should be clean after save")
	}

	// verify on disk
	diskData, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	val, ok = p.FindValue(diskData, "font-size")
	if !ok || val != "16" {
		t.Fatalf("on-disk font-size = %q, %v", val, ok)
	}

	// backup should contain original
	bakData, err := os.ReadFile(BackupPath(path))
	if err != nil {
		t.Fatal(err)
	}
	val, ok = p.FindValue(bakData, "font-size")
	if !ok || val != "14" {
		t.Fatalf("backup font-size = %q, %v; want 14", val, ok)
	}

	// add a new key
	content = cf.Content()
	newContent, err = p.SetValue(content, "foreground", "ebdbb2")
	if err != nil {
		t.Fatal(err)
	}
	cf.SetContent(newContent)

	// delete a key
	newContent, err = p.DeleteKey(cf.Content(), "background")
	if err != nil {
		t.Fatal(err)
	}
	cf.SetContent(newContent)

	// save again
	if err := cf.Save(context.Background()); err != nil {
		t.Fatal(err)
	}

	// verify final state
	diskData, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	_, ok = p.FindValue(diskData, "background")
	if ok {
		t.Error("background should be deleted on disk")
	}
	val, ok = p.FindValue(diskData, "foreground")
	if !ok || val != "ebdbb2" {
		t.Errorf("foreground on disk = %q, %v", val, ok)
	}
	val, ok = p.FindValue(diskData, "font-size")
	if !ok || val != "16" {
		t.Errorf("font-size on disk = %q, %v", val, ok)
	}
}

// TestE2ESectionConfigWorkflow exercises the full ConfigFile lifecycle
// with a section-based parser (like starship/helix).
func TestE2ESectionConfigWorkflow(t *testing.T) {
	initial := "# starship\n[aws]\nformat = \"on [$profile]($style)\"\nstyle = \"bold yellow\"\n\n[git_branch]\nsymbol = \" \"\n"
	cf, _ := e2eConfigSetup(t, initial)
	p := &parser.SectionParser{SplitKey: parser.SplitKeyFirst}

	// read
	content := cf.Content()
	val, ok := p.FindValue(content, "aws.format")
	if !ok || val != "on [$profile]($style)" {
		t.Fatalf("aws.format = %q, %v", val, ok)
	}

	// modify section key
	newContent, err := p.SetValue(content, "aws.style", "bold blue")
	if err != nil {
		t.Fatal(err)
	}
	cf.SetContent(newContent)

	// add new key to existing section
	newContent, err = p.SetValue(cf.Content(), "aws.disabled", "true")
	if err != nil {
		t.Fatal(err)
	}
	cf.SetContent(newContent)

	// add new section
	newContent, err = p.SetValue(cf.Content(), "cmd_duration.min_time", "500")
	if err != nil {
		t.Fatal(err)
	}
	cf.SetContent(newContent)

	// save
	if err := cf.Save(context.Background()); err != nil {
		t.Fatal(err)
	}

	// reload and verify
	if err := cf.Reload(context.Background()); err != nil {
		t.Fatal(err)
	}
	if cf.Dirty() {
		t.Fatal("should be clean after reload")
	}

	content = cf.Content()
	val, ok = p.FindValue(content, "aws.style")
	if !ok || val != "bold blue" {
		t.Errorf("aws.style after reload = %q, %v", val, ok)
	}
	val, ok = p.FindValue(content, "aws.disabled")
	if !ok || val != "true" {
		t.Errorf("aws.disabled after reload = %q, %v", val, ok)
	}
	val, ok = p.FindValue(content, "cmd_duration.min_time")
	if !ok || val != "500" {
		t.Errorf("cmd_duration.min_time after reload = %q, %v", val, ok)
	}

	// original section key untouched
	val, ok = p.FindValue(content, "git_branch.symbol")
	if !ok || val != " " {
		t.Errorf("git_branch.symbol should survive = %q, %v", val, ok)
	}
}

// TestE2EPreviewRevert tests the preview/revert-preview cycle.
func TestE2EPreviewRevert(t *testing.T) {
	initial := "key = original\n"
	cf, path := e2eConfigSetup(t, initial)
	p := &parser.FlatParser{Split: parser.SplitEquals, Format: parser.FormatEquals}

	// modify
	newContent, _ := p.SetValue(cf.Content(), "key", "previewed")
	cf.SetContent(newContent)

	// preview writes to disk but keeps dirty
	if err := cf.Preview(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !cf.Dirty() {
		t.Fatal("should be dirty after preview")
	}
	diskData, _ := os.ReadFile(path)
	val, _ := p.FindValue(diskData, "key")
	if val != "previewed" {
		t.Errorf("disk after preview = %q, want previewed", val)
	}

	// revert preview restores original to disk
	if err := cf.RevertPreview(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !cf.Dirty() {
		t.Fatal("should still be dirty after revert preview (working copy differs)")
	}
	diskData, _ = os.ReadFile(path)
	val, _ = p.FindValue(diskData, "key")
	if val != "original" {
		t.Errorf("disk after revert = %q, want original", val)
	}
}

// TestE2EMultipleRoundTrips verifies that repeated set/save/load cycles are stable.
func TestE2EMultipleRoundTrips(t *testing.T) {
	initial := "a = 1\nb = 2\nc = 3\n"
	cf, _ := e2eConfigSetup(t, initial)
	p := &parser.FlatParser{Split: parser.SplitEquals, Format: parser.FormatEquals}

	// cycle 1: modify a
	out, _ := p.SetValue(cf.Content(), "a", "10")
	cf.SetContent(out)
	cf.Save(context.Background())

	// cycle 2: modify b
	out, _ = p.SetValue(cf.Content(), "b", "20")
	cf.SetContent(out)
	cf.Save(context.Background())

	// cycle 3: add d
	out, _ = p.SetValue(cf.Content(), "d", "4")
	cf.SetContent(out)
	cf.Save(context.Background())

	// cycle 4: delete c
	out, _ = p.DeleteKey(cf.Content(), "c")
	cf.SetContent(out)
	cf.Save(context.Background())

	// verify final state
	content := cf.Content()
	expected := map[string]string{"a": "10", "b": "20", "d": "4"}
	for k, v := range expected {
		got, ok := p.FindValue(content, k)
		if !ok || got != v {
			t.Errorf("%s = %q, %v; want %s", k, got, ok, v)
		}
	}
	_, ok := p.FindValue(content, "c")
	if ok {
		t.Error("c should be deleted")
	}
}

// TestE2EGenerationMonotonicallyIncreasing verifies the generation counter.
func TestE2EGenerationMonotonicallyIncreasing(t *testing.T) {
	cf, _ := e2eConfigSetup(t, "key = val\n")
	gen0 := cf.Generation()

	cf.SetContent([]byte("key = new\n"))
	gen1 := cf.Generation()
	if gen1 <= gen0 {
		t.Errorf("generation should increase on SetContent: %d -> %d", gen0, gen1)
	}

	cf.Reload(context.Background())
	gen2 := cf.Generation()
	if gen2 <= gen1 {
		t.Errorf("generation should increase on Reload: %d -> %d", gen1, gen2)
	}
}
