package ui

import (
	"strings"
	"testing"

	"github.com/getkonfi/konfi/pkg"
	cfgparse "github.com/getkonfi/konfi/pkg/parser"

	tea "charm.land/bubbletea/v2"
)

func newHypridleContentTestModel(t *testing.T) content {
	t.Helper()

	parser := cfgparse.NewHyprParser()
	k := &switchTestKonfable{name: hypridleAppName, parser: parser}
	c := newContent(testTheme())
	c.focused = true
	c.width = 120
	c.height = 24
	c.konfable = k
	c.config = newDetailTestConfig(t, "")
	c.schema = &pkg.Schema{
		DocsURL: "https://wiki.hypr.land/Hypr-Ecosystem/hypridle/",
		Sections: []pkg.Section{
			{
				Name: "General",
				Fields: []pkg.Field{
					{Key: "general.lock_cmd", Label: "Lock Command", Type: "string"},
					{Key: "general.unlock_cmd", Label: "Unlock Command", Type: "string"},
					{Key: "general.on_lock_cmd", Label: "On Lock Command", Type: "string"},
					{Key: "general.on_unlock_cmd", Label: "On Unlock Command", Type: "string"},
					{Key: "general.before_sleep_cmd", Label: "Before Sleep Command", Type: "string"},
					{Key: "general.after_sleep_cmd", Label: "After Sleep Command", Type: "string"},
					{Key: "general.ignore_dbus_inhibit", Label: "Ignore DBus Inhibit", Type: "bool", Default: "false"},
					{Key: "general.ignore_systemd_inhibit", Label: "Ignore Systemd Inhibit", Type: "bool", Default: "false"},
					{Key: "general.ignore_wayland_inhibit", Label: "Ignore Wayland Inhibit", Type: "bool", Default: "false"},
					{Key: "general.inhibit_sleep", Label: "Inhibit Sleep", Type: "enum", Default: "2", Options: []string{"0", "1", "2", "3"}},
				},
			},
			{
				Name: "Listeners",
				Fields: []pkg.Field{
					{Key: hypridleListenersKey, Label: "Listeners", Type: "list", Widget: "structlist"},
				},
			},
		},
	}
	c.buildFieldList()
	c.values = map[string]string{
		hypridleListenersKey:             "300 <-> loginctl lock-session <->  <-> false\n600 <-> hyprctl dispatch dpms off <-> hyprctl dispatch dpms on <-> false",
		"general.lock_cmd":               "pidof hyprlock || hyprlock",
		"general.before_sleep_cmd":       "loginctl lock-session",
		"general.after_sleep_cmd":        "hyprctl dispatch dpms on",
		"general.ignore_dbus_inhibit":    "false",
		"general.ignore_systemd_inhibit": "false",
		"general.ignore_wayland_inhibit": "false",
		"general.inhibit_sleep":          "2",
	}
	c.origValues = cloneValues(c.values)
	c.syncDetail()
	return c
}

func TestHypridleDashboardRendersDomainLayout(t *testing.T) {
	c := newHypridleContentTestModel(t)

	got := stripANSI(c.renderHypridleDashboardBody(120))

	for _, want := range []string{
		"idle listeners",
		"timeout",
		"on-timeout command",
		"on-resume command",
		"general hooks",
		"inhibitor policy",
		"dbus inhibitors",
		"respected",
		"checks",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("dashboard missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "idle sequence") {
		t.Fatalf("dashboard should use hypridle terminology, got:\n%s", got)
	}
}

func TestHypridleCurrentFieldMapsListenerRowToListenersField(t *testing.T) {
	c := newHypridleContentTestModel(t)
	c.cursor = 0

	f := c.currentField()
	if f == nil || f.Key != hypridleListenersKey {
		t.Fatalf("current field = %#v, want listeners", f)
	}

	c.cursor = 2
	f = c.currentField()
	if f == nil || f.Key != "general.lock_cmd" {
		t.Fatalf("current field = %#v, want general.lock_cmd", f)
	}
}

func TestHypridleBackspaceListenerRowDoesNotDeleteAllListeners(t *testing.T) {
	c := newHypridleContentTestModel(t)
	before := c.values[hypridleListenersKey]
	c.cursor = 0

	var cmd tea.Cmd
	c, cmd = c.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	if cmd == nil {
		t.Fatal("backspace listener row should return a status command")
	}
	if got := c.values[hypridleListenersKey]; got != before {
		t.Fatalf("listeners changed on dashboard revert: got %q want %q", got, before)
	}
	msg, ok := cmd().(StatusMsg)
	if !ok {
		t.Fatalf("revert command emitted %T, want StatusMsg", cmd())
	}
	if !strings.Contains(msg.Text, "no field changes") {
		t.Fatalf("revert status = %q, want no-change feedback", msg.Text)
	}
}

func TestHypridleDListenerRowDoesNotDeleteAllListeners(t *testing.T) {
	c := newHypridleContentTestModel(t)
	before := c.values[hypridleListenersKey]
	c.cursor = 0

	var cmd tea.Cmd
	c, cmd = c.Update(tea.KeyPressMsg{Text: "d"})
	if cmd == nil {
		t.Fatal("d on listener row should return a status command")
	}
	if got := c.values[hypridleListenersKey]; got != before {
		t.Fatalf("listeners changed on dashboard delete: got %q want %q", got, before)
	}
	msg, ok := cmd().(StatusMsg)
	if !ok {
		t.Fatalf("delete command emitted %T, want StatusMsg", cmd())
	}
	if !strings.Contains(msg.Text, "edit listeners") {
		t.Fatalf("delete status = %q, want guidance to edit listeners", msg.Text)
	}
}
