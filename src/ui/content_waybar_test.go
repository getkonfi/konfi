package ui

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/getkonfi/konfi/konfables/waybar"
	"github.com/getkonfi/konfi/pkg"
)

const waybarListRefreshConfig = `{
  "position": "top",
  "height": 30,
  "modules-left": ["hyprland/workspaces", "hyprland/window"],
  "modules-center": ["clock"],
  "modules-right": ["network", "pulseaudio", "battery", "tray"]
}
`

func newWaybarContentTestModel(t *testing.T) content {
	t.Helper()

	k := waybar.New(pkg.NewFilePersister(filepath.Join(t.TempDir(), "config.jsonc")))
	schemaData, err := k.Schema()
	if err != nil {
		t.Fatal(err)
	}
	schema, err := pkg.LoadSchema(schemaData)
	if err != nil {
		t.Fatal(err)
	}

	c := newContent(testTheme())
	c.width = 120
	c.height = 20
	c.konfable = k
	c.config = newDetailTestConfig(t, waybarListRefreshConfig)
	c.schema = schema
	c.buildFieldList()
	c.refreshValues()
	c.snapshotOrigValues()
	return c
}

func TestWaybarListRefreshUsesMultiValueDisplayShape(t *testing.T) {
	c := newWaybarContentTestModel(t)

	if got, want := c.values["position"], "top"; got != want {
		t.Fatalf("position = %q, want %q", got, want)
	}
	if got, want := c.values["height"], "30"; got != want {
		t.Fatalf("height = %q, want %q", got, want)
	}
	if got, want := c.values["modules-right"], "network, pulseaudio, battery, tray"; got != want {
		t.Fatalf("modules-right = %q, want %q", got, want)
	}
	if got, want := c.values["modules-center"], "clock"; got != want {
		t.Fatalf("modules-center = %q, want %q", got, want)
	}
}

func TestWaybarModulesRightEditUndoKeepsArrayItems(t *testing.T) {
	c := newWaybarContentTestModel(t)

	commitTestField(t, &c, "modules-right", "network\nbattery\ntray")
	assertWaybarModulesRight(t, c.config.Content(), []string{"network", "battery", "tray"})
	if got, want := c.values["modules-right"], "network, battery, tray"; got != want {
		t.Fatalf("after edit modules-right = %q, want %q", got, want)
	}

	var cmdErr string
	c, cmd := c.Update(UndoMsg{})
	if cmd != nil {
		if msg := cmd(); msg != nil {
			if status, ok := msg.(StatusMsg); ok {
				cmdErr = status.Text
			}
		}
	}
	if cmdErr != "" {
		t.Fatalf("undo returned status error: %s", cmdErr)
	}

	assertWaybarModulesRight(t, c.config.Content(), []string{"network", "pulseaudio", "battery", "tray"})
	if got, want := c.values["modules-right"], "network, pulseaudio, battery, tray"; got != want {
		t.Fatalf("after undo modules-right = %q, want %q", got, want)
	}
}

func assertWaybarModulesRight(t *testing.T, data []byte, want []string) {
	t.Helper()

	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		t.Fatalf("waybar config is not json: %v\n%s", err, data)
	}
	raw, ok := root["modules-right"].([]any)
	if !ok {
		t.Fatalf("modules-right = %#v, want array", root["modules-right"])
	}
	if len(raw) != len(want) {
		t.Fatalf("len(modules-right) = %d, want %d: %#v", len(raw), len(want), raw)
	}
	for i, wantItem := range want {
		got, ok := raw[i].(string)
		if !ok {
			t.Fatalf("modules-right[%d] = %#v, want string", i, raw[i])
		}
		if got != wantItem {
			t.Fatalf("modules-right[%d] = %q, want %q", i, got, wantItem)
		}
	}
}
