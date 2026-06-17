package ui

import (
	"context"
	"strings"
	"testing"

	"github.com/getkonfi/konfi/pkg"
	cfgparse "github.com/getkonfi/konfi/pkg/parser"
)

func TestHeaderLeftLinesShowsConfigPathWhileLoadPending(t *testing.T) {
	c := newContent(testTheme())
	c.konfable = detailTestKonfable{}

	got := c.headerLeftLines()
	if got[1] != "/tmp/test.conf" {
		t.Fatalf("pending load header path = %q, want config path", got[1])
	}

	c.configLoadFailed = true
	got = c.headerLeftLines()
	if !strings.Contains(got[1], "load failed") {
		t.Fatalf("failed load header path = %q, want load failure label", got[1])
	}
}

func TestLoadAppSplitFlapTargetsConfigPathWhileLoadPending(t *testing.T) {
	c := newContent(testTheme())
	c.konfable = &switchTestKonfable{name: "old"}
	c.insightLines = []string{"ready"}

	_ = c.loadApp(&switchTestKonfable{name: "next"})

	if c.splitFlap == nil {
		t.Fatal("loadApp did not create split-flap animation")
	}
	if got := c.splitFlap.target[1]; got != "/tmp/next.conf" {
		t.Fatalf("split-flap target path = %q, want config path", got)
	}
}

func TestLoadAppSplitFlapAnimatesFirstSelectionFromBlank(t *testing.T) {
	c := newContent(testTheme())

	_ = c.loadApp(&switchTestKonfable{name: "first"})

	if c.splitFlap == nil {
		t.Fatal("first loadApp did not create split-flap animation")
	}
	for i, line := range c.splitFlap.source {
		if line != "" {
			t.Fatalf("split-flap first source line %d = %q, want blank", i, line)
		}
	}
	if got := c.splitFlap.target[0]; got != "first" {
		t.Fatalf("split-flap target title = %q, want first", got)
	}
	if got := c.splitFlap.target[1]; got != "/tmp/first.conf" {
		t.Fatalf("split-flap target path = %q, want config path", got)
	}
}

func TestShowDashboardClearsSplitFlap(t *testing.T) {
	c := newContent(testTheme())
	c.konfable = &switchTestKonfable{name: "old"}
	c.insightLines = []string{"ready"}

	_ = c.loadApp(&switchTestKonfable{name: "next"})
	if c.splitFlap == nil {
		t.Fatal("loadApp did not create split-flap animation")
	}

	c.showDashboard()

	if c.splitFlap != nil {
		t.Fatal("showDashboard kept split-flap animation")
	}
}

func TestLoadAppSplitFlapUsesLoadedConfiguredCount(t *testing.T) {
	schema := &pkg.Schema{
		Sections: []pkg.Section{{
			Name: "general",
			Fields: []pkg.Field{
				{Key: "alpha", Type: "string"},
				{Key: "beta", Type: "string"},
			},
		}},
	}
	k := &switchTestKonfable{
		name:   "first",
		parser: &cfgparse.FlatParser{Split: cfgparse.SplitEquals, Format: cfgparse.FormatEquals},
		data:   []byte("alpha = one\n"),
	}
	c := newContent(testTheme())
	c.schemaCache = map[string]*pkg.Schema{k.Name(): schema}

	_ = c.loadApp(k)

	if c.splitFlap == nil {
		t.Fatal("loadApp did not create split-flap animation")
	}
	if got := c.splitFlap.target[2]; got != "" {
		t.Fatalf("pending split-flap insight = %q, want blank until config load", got)
	}

	cf, err := pkg.NewConfigFile(context.Background(), k)
	if err != nil {
		t.Fatal(err)
	}
	r := &root{
		content:      c,
		dirtyConfigs: make(map[string]dirtyConfigState),
	}
	_, _ = r.Update(appLoadedMsg{appName: k.Name(), config: cf, path: k.ConfigPath()})

	if got := r.content.splitFlap.target[2]; got != "1/2 fields configured across 1 sections" {
		t.Fatalf("loaded split-flap insight = %q, want configured count", got)
	}
}
