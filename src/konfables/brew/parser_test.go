package brew

import (
	"slices"
	"strings"
	"testing"

	"github.com/eminert/konfi/konfables"
	"github.com/eminert/konfi/pkg"
)

const sample = `# my brewfile
tap "homebrew/bundle"
tap "homebrew/cask-fonts"

brew "git"
brew "wget", args: ["with-iri"]
brew "zsh"

cask "firefox"
cask "1password"

mas "Xcode", id: 497799835
mas "Things", id: 904280696

vscode "ms-python.python"

# unmanaged entry types are preserved
cask_args appdir: "/Applications"
whalebrew "whalebrew/wget"
if OS.mac?
  cask "rectangle"
end
`

func TestFindValues(t *testing.T) {
	p := &parser{}
	data := []byte(sample)
	tests := []struct {
		key  string
		want []string
	}{
		{"tap", []string{"homebrew/bundle", "homebrew/cask-fonts"}},
		{"brew", []string{"git", "wget", "zsh"}},
		{"cask", []string{"firefox", "1password", "rectangle"}},
		{"vscode", []string{"ms-python.python"}},
	}
	for _, tt := range tests {
		got, ok := p.FindValues(data, tt.key)
		if !ok {
			t.Errorf("FindValues(%q) not found", tt.key)
			continue
		}
		if !slices.Equal(got, tt.want) {
			t.Errorf("FindValues(%q) = %v, want %v", tt.key, got, tt.want)
		}
	}
}

func TestFindMasValue(t *testing.T) {
	p := &parser{}
	got, ok := p.FindValue([]byte(sample), "mas")
	if !ok {
		t.Fatal("mas not found")
	}
	want := "Xcode | 497799835\nThings | 904280696"
	if got != want {
		t.Errorf("FindValue(mas) = %q, want %q", got, want)
	}
}

func TestSetValuesPreservesInlineArgsAndComments(t *testing.T) {
	p := &parser{}
	// keep git + wget (with args), drop zsh, add ripgrep
	out, err := p.SetValues([]byte(sample), "brew", []string{"git", "wget", "ripgrep"})
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)

	if !strings.Contains(s, `brew "wget", args: ["with-iri"]`) {
		t.Error("inline args on wget were not preserved")
	}
	if strings.Contains(s, `brew "zsh"`) {
		t.Error("zsh should have been removed")
	}
	if !strings.Contains(s, `brew "ripgrep"`) {
		t.Error("ripgrep should have been added")
	}
	if !strings.Contains(s, "# my brewfile") {
		t.Error("comment was not preserved")
	}
	// unmanaged lines must survive untouched
	for _, keep := range []string{`cask_args appdir: "/Applications"`, `whalebrew "whalebrew/wget"`, "if OS.mac?", "end"} {
		if !strings.Contains(s, keep) {
			t.Errorf("unmanaged line %q was not preserved", keep)
		}
	}
	// new formula inserted within the brew group, not after unrelated entries
	gitIdx := strings.Index(s, `brew "git"`)
	rgIdx := strings.Index(s, `brew "ripgrep"`)
	caskIdx := strings.Index(s, `cask "firefox"`)
	if gitIdx >= rgIdx || rgIdx >= caskIdx {
		t.Errorf("ripgrep not grouped with formulae (git=%d rg=%d cask=%d)", gitIdx, rgIdx, caskIdx)
	}
}

func TestMasRoundTripAndIDEdit(t *testing.T) {
	p := &parser{}
	// edit Xcode's id, drop Things, add Tailscale
	val := "Xcode | 497799999\nTailscale | 1475387142"
	out, err := p.SetValue([]byte(sample), "mas", val)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, `mas "Xcode", id: 497799999`) {
		t.Error("Xcode id edit did not take effect")
	}
	if strings.Contains(s, "Things") {
		t.Error("Things should have been removed")
	}
	if !strings.Contains(s, `mas "Tailscale", id: 1475387142`) {
		t.Error("Tailscale should have been added")
	}
	// round-trip back through FindValue
	got, _ := p.FindValue(out, "mas")
	want := "Xcode | 497799999\nTailscale | 1475387142"
	if got != want {
		t.Errorf("round-trip mas = %q, want %q", got, want)
	}
}

func TestWriteFieldMasStructListUsesSetValue(t *testing.T) {
	p := &parser{}
	field := pkg.Field{Key: "mas", Type: "list", Widget: "structlist"}
	value := "Xcode | 497799999\nTailscale | 1475387142"

	out, err := konfables.WriteField(p, []byte(sample), field, value, "brew")
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, `mas "Xcode", id: 497799999`) || !strings.Contains(s, `mas "Tailscale", id: 1475387142`) {
		t.Fatalf("mas structlist did not write through raw SetValue:\n%s", out)
	}
}

func TestSetValuesOnEmptyFile(t *testing.T) {
	p := &parser{}
	out, err := p.SetValues([]byte(""), "brew", []string{"git"})
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "brew \"git\"\n" {
		t.Errorf("empty-file add = %q, want %q", string(out), "brew \"git\"\n")
	}
}

func TestDeleteKey(t *testing.T) {
	p := &parser{}
	out, err := p.DeleteKey([]byte(sample), "tap")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := p.FindValues(out, "tap"); ok {
		t.Error("taps should be gone after DeleteKey")
	}
	// sibling keys untouched
	if _, ok := p.FindValues(out, "brew"); !ok {
		t.Error("DeleteKey(tap) should not affect formulae")
	}
}

func TestListKeys(t *testing.T) {
	p := &parser{}
	got := p.ListKeys([]byte(sample))
	for _, want := range []string{"tap", "brew", "cask", "mas", "vscode"} {
		if !slices.Contains(got, want) {
			t.Errorf("ListKeys missing %q (got %v)", want, got)
		}
	}
}
