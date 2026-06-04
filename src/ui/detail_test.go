package ui

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/eminert/konfi/konfables"
	"github.com/eminert/konfi/pkg"
	cfgparse "github.com/eminert/konfi/pkg/parser"
	"github.com/eminert/konfi/theme"
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

func TestDetailMissingFieldShowsTailContextAndAddLine(t *testing.T) {
	th := theme.NewTheme(&theme.Catppuccin)
	p := &cfgparse.FlatParser{Split: cfgparse.SplitEquals, Format: cfgparse.FormatEquals}
	k := detailTestKonfable{parser: p, info: konfables.AppInfo{Format: "ghostty"}}
	cf := newDetailTestConfig(t, "# config\na = 1\nb = 2\nc = 3\n")
	f := &pkg.Field{Key: "d", Type: "string", Default: "four"}

	d := newDetail(th)
	d.sync(f, cf, k, map[string]string{}, true)

	got := stripANSI(d.renderFileSnippet(80, 4))
	if !strings.Contains(got, "  b = 2\n  c = 3\n+ d = four") {
		t.Fatalf("missing preview did not include tail context plus add line:\n%s", got)
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
	if !strings.Contains(stripANSI(d.renderFileSnippet(80, 3)), "+ d = four") {
		t.Fatal("initial missing preview did not show add line")
	}

	cf.SetContent([]byte("a = 1\nd = four\n"))
	d.sync(f, cf, k, map[string]string{"d": "four"}, true)

	got := stripANSI(d.renderFileSnippet(80, 3))
	if !strings.Contains(got, "▶ d = four") {
		t.Fatalf("preview did not rescan to existing line after config change:\n%s", got)
	}
	if strings.Contains(got, "+ d = four") {
		t.Fatalf("preview still showed add line after config change:\n%s", got)
	}
}
