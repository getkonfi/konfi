package ui

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/getkonfi/konfi/konfables"
	"github.com/getkonfi/konfi/pkg"
	cfgparse "github.com/getkonfi/konfi/pkg/parser"
	"github.com/getkonfi/konfi/theme"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

type detailTestPersister struct {
	data []byte
}

func (p *detailTestPersister) Load(context.Context) ([]byte, error) {
	return p.data, nil
}

func (p *detailTestPersister) Save(_ context.Context, _, data []byte) error {
	p.data = data
	return nil
}

type detailTestKonfable struct {
	parser konfables.Parser
	info   konfables.AppInfo
}

func (k detailTestKonfable) Info() konfables.AppInfo              { return k.info }
func (k detailTestKonfable) Parser() konfables.Parser             { return k.parser }
func (k detailTestKonfable) Schema() ([]byte, error)              { return nil, nil }
func (k detailTestKonfable) Name() string                         { return "test" }
func (k detailTestKonfable) ConfigPath() string                   { return "/tmp/test.conf" }
func (k detailTestKonfable) Load(context.Context) ([]byte, error) { return nil, nil }
func (k detailTestKonfable) Save(context.Context, []byte, []byte) error {
	return nil
}

func newDetailTestConfig(t *testing.T, data string) *pkg.ConfigFile {
	t.Helper()
	cf, err := pkg.NewConfigFile(context.Background(), &detailTestPersister{data: []byte(data)})
	if err != nil {
		t.Fatal(err)
	}
	return cf
}

func stripANSI(s string) string {
	return ansiRE.ReplaceAllString(s, "")
}

func testTheme() *theme.Theme {
	return theme.NewTheme(theme.PaletteByName("catppuccin"))
}

func TestDetailMissingFieldShowsTailContextAndAddLine(t *testing.T) {
	th := theme.NewTheme(&theme.Catppuccin)
	p := &cfgparse.FlatParser{Split: cfgparse.SplitEquals, Format: cfgparse.FormatEquals}
	k := detailTestKonfable{parser: p, info: konfables.AppInfo{Format: "ghostty"}}
	cf := newDetailTestConfig(t, "# config\na = 1\nb = 2\nc = 3\n")
	f := &pkg.Field{Key: "d", Type: "string", Default: "four"}

	d := newDetail(th)
	d.sync(f, cf, k, map[string]string{}, true)

	got := stripANSI(d.renderFileSnippet(80, 4))
	if !strings.Contains(got, "  3 b = 2\n  4 c = 3\n+ 5 d = four") {
		t.Fatalf("missing preview did not include tail context plus add line:\n%s", got)
	}
}

func TestDetailBrowseOmitsConfigRuleLabel(t *testing.T) {
	th := theme.NewTheme(&theme.Catppuccin)
	p := &cfgparse.FlatParser{Split: cfgparse.SplitEquals, Format: cfgparse.FormatEquals}
	k := detailTestKonfable{parser: p, info: konfables.AppInfo{Format: "ghostty"}}
	cf := newDetailTestConfig(t, "font = 12\n")
	f := &pkg.Field{Key: "font", Label: "Font", Type: "string", Description: "font size"}

	d := newDetail(th)
	d.sync(f, cf, k, map[string]string{"font": "12"}, true)

	got := stripANSI(d.viewBrowse(80, 20))
	if strings.Contains(got, "── config") {
		t.Fatalf("browse detail should not render config rule label:\n%s", got)
	}
	sep := strings.Repeat("─", 80)
	if !strings.Contains(got, "font size\n\n"+sep+"\n▶ 1 font = 12") {
		t.Fatalf("browse detail should separate description from config snippet:\n%s", got)
	}
	if !strings.Contains(got, "font = 12") {
		t.Fatalf("browse detail lost config snippet:\n%s", got)
	}
}

func TestDetailPreviewLineRescansWhenConfigChanges(t *testing.T) {
	th := theme.NewTheme(&theme.Catppuccin)
	p := &cfgparse.FlatParser{Split: cfgparse.SplitEquals, Format: cfgparse.FormatEquals}
	k := detailTestKonfable{parser: p, info: konfables.AppInfo{Format: "ghostty"}}
	cf := newDetailTestConfig(t, "a = 1\n")
	f := &pkg.Field{Key: "d", Type: "string", Default: "four"}

	d := newDetail(th)
	d.sync(f, cf, k, map[string]string{}, true)
	if !strings.Contains(stripANSI(d.renderFileSnippet(80, 3)), "+ 2 d = four") {
		t.Fatal("initial missing preview did not show add line")
	}

	cf.SetContent([]byte("a = 1\nd = four\n"))
	d.sync(f, cf, k, map[string]string{"d": "four"}, true)

	got := stripANSI(d.renderFileSnippet(80, 3))
	if !strings.Contains(got, "▶ 2 d = four") {
		t.Fatalf("preview did not rescan to existing line after config change:\n%s", got)
	}
	if strings.Contains(got, "+ 2 d = four") {
		t.Fatalf("preview still showed add line after config change:\n%s", got)
	}
}

func TestDetailConfigSnippetShowsChangeAsHunk(t *testing.T) {
	th := theme.NewTheme(&theme.Catppuccin)
	p := &cfgparse.FlatParser{Split: cfgparse.SplitEquals, Format: cfgparse.FormatEquals}
	k := detailTestKonfable{parser: p, info: konfables.AppInfo{Format: "ghostty"}}
	// config already holds the new value (edits update content live)
	cf := newDetailTestConfig(t, "a = 1\nfont = 14\n")
	f := &pkg.Field{Key: "font", Type: "string", Default: "12"}

	d := newDetail(th)
	d.sync(f, cf, k, map[string]string{"font": "14"}, true)
	d.origValues = map[string]string{"font": "12"} // baseline differs from current

	got := stripANSI(d.renderFileSnippet(80, 5))
	if !strings.Contains(got, "- 2 font = 12") {
		t.Fatalf("config snippet did not show removed old line:\n%s", got)
	}
	if !strings.Contains(got, "+ 2 font = 14") {
		t.Fatalf("config snippet did not show added new line:\n%s", got)
	}

	// unchanged field: no hunk, just the focused line
	d.origValues = map[string]string{"font": "14"}
	got = stripANSI(d.renderFileSnippet(80, 5))
	if strings.Contains(got, "- 2 font") || strings.Contains(got, "+ 2 font") {
		t.Fatalf("unchanged field should not render a hunk:\n%s", got)
	}
	if !strings.Contains(got, "▶ 2 font = 14") {
		t.Fatalf("unchanged field should render the focused line:\n%s", got)
	}
}

func TestDetailFocusedViewShowsScrollableConfigFile(t *testing.T) {
	th := theme.NewTheme(&theme.Catppuccin)
	p := &cfgparse.FlatParser{Split: cfgparse.SplitEquals, Format: cfgparse.FormatEquals}
	k := detailTestKonfable{parser: p, info: konfables.AppInfo{Format: "ghostty"}}
	cf := newDetailTestConfig(t, "a = 1\nb = 2\nc = 3\nd = 4\ne = 5\n")
	f := &pkg.Field{Key: "d", Type: "string", Default: "4"}

	d := newDetail(th)
	d.sync(f, cf, k, map[string]string{"d": "4"}, true)
	d.centerPreview(3)

	got := stripANSI(d.View(40, 5))
	if !strings.Contains(got, "▶ 4 d = 4") {
		t.Fatalf("focused config view did not highlight selected config line:\n%s", got)
	}
	if !strings.Contains(got, "↕") {
		t.Fatalf("focused config view did not include scroll indicator:\n%s", got)
	}

	d.scroll(1)
	scrolled := stripANSI(d.View(40, 5))
	if scrolled == got {
		t.Fatalf("scrolling focused config view did not change output:\n%s", scrolled)
	}
}

func TestDetailFocusedViewShowsMissingFieldAsAddedLine(t *testing.T) {
	th := theme.NewTheme(&theme.Catppuccin)
	p := &cfgparse.FlatParser{Split: cfgparse.SplitEquals, Format: cfgparse.FormatEquals}
	k := detailTestKonfable{parser: p, info: konfables.AppInfo{Format: "ghostty"}}
	cf := newDetailTestConfig(t, "a = 1\nb = 2\n")
	f := &pkg.Field{Key: "c", Type: "string", Default: "3"}

	d := newDetail(th)
	d.sync(f, cf, k, map[string]string{}, true)
	d.centerPreview(3)

	got := stripANSI(d.View(40, 5))
	if !strings.Contains(got, "+ 3 c = 3") {
		t.Fatalf("focused config view did not show missing field as added line:\n%s", got)
	}
}
