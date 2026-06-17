package ui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getkonfi/konfi/pkg"
	cfgparse "github.com/getkonfi/konfi/pkg/parser"
	"github.com/getkonfi/konfi/setup"
	"github.com/getkonfi/konfi/setup/cst"
	"github.com/getkonfi/konfi/theme"

	tea "charm.land/bubbletea/v2"
)

func TestCycleThemePersistsKonfiConfigWithoutBackup(t *testing.T) {
	r, path := newKonfiThemeTestRoot(t)

	cmd := r.cycleTheme()
	if cmd == nil {
		t.Fatal("cycleTheme did not return a config save command")
	}
	runRootCmd(t, r, cmd)

	assertKonfiThemeSavedWithoutBackup(t, r, path)
}

func TestThemeSettingChangeReloadsActiveKonfiConfigWithoutBackup(t *testing.T) {
	r, path := newKonfiThemeTestRoot(t)

	newData, err := r.content.konfable.Parser().SetValue(r.content.config.Content(), "theme", "solarized")
	if err != nil {
		t.Fatal(err)
	}
	r.content.config.SetContent(newData)
	r.content.refreshValues()
	if !r.content.config.Dirty() {
		t.Fatal("test setup expected active config to be dirty")
	}

	_, cmd := r.Update(KonfSettingChangedMsg{Key: "theme", Value: "solarized"})
	runRootCmd(t, r, cmd)

	assertKonfiThemeSavedWithoutBackup(t, r, path)
}

func newKonfiThemeTestRoot(t *testing.T) (r *root, path string) {
	t.Helper()

	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	path = cst.ConfigFilePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("theme: rose pine\nbackup_limit: 5\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cf, err := pkg.NewConfigFile(context.Background(), pkg.NewFilePersister(path))
	if err != nil {
		t.Fatal(err)
	}
	th := theme.NewTheme(theme.PaletteByName("rose pine"))
	c := newContent(th)
	c.konfable = &switchTestKonfable{
		name:   "konfi",
		parser: &cfgparse.FlatParser{Split: cfgparse.SplitColon, Format: cfgparse.FormatColon},
	}
	c.config = cf
	c.schema = &pkg.Schema{
		Sections: []pkg.Section{{
			Fields: []pkg.Field{{Key: "theme", Label: "Theme", Type: "enum"}},
		}},
	}
	c.buildFieldList()
	c.refreshValues()
	c.snapshotOrigValues()

	r = &root{
		app: &setup.App{
			Config: &setup.KonfConfig{Theme: "rose pine", BackupLimit: 5},
			Theme:  th,
		},
		content:      c,
		status:       newStatusbar(th),
		dirtyConfigs: make(map[string]dirtyConfigState),
	}
	return r, path
}

func runRootCmd(t *testing.T, r *root, cmd tea.Cmd) {
	t.Helper()

	if cmd == nil {
		return
	}
	switch msg := cmd().(type) {
	case nil:
		return
	case tea.BatchMsg:
		for _, batchCmd := range msg {
			runRootCmd(t, r, batchCmd)
		}
	default:
		_, next := r.Update(msg)
		runRootCmd(t, r, next)
	}
}

func assertKonfiThemeSavedWithoutBackup(t *testing.T, r *root, path string) {
	t.Helper()

	wantTheme := r.app.Config.Theme
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "theme: "+wantTheme) {
		t.Fatalf("config did not persist theme %q:\n%s", wantTheme, data)
	}
	if _, err := os.Stat(pkg.BackupPath(path)); !os.IsNotExist(err) {
		t.Fatalf("backup should not exist after instant theme save: %v", err)
	}
	if r.content.config.Dirty() {
		t.Fatal("active konfi config stayed dirty after instant theme save")
	}
	if got := r.content.values["theme"]; got != wantTheme {
		t.Fatalf("active konfi theme value = %q, want %q", got, wantTheme)
	}
}
