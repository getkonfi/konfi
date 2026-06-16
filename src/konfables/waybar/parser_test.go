package waybar

import (
	"encoding/json"
	"testing"

	"github.com/getkonfi/konfi/konfables"
)

const sampleConfig = `{
  "position": "top",
  "layer": "top",
  "height": 30,
  "modules-left": ["hyprland/workspaces", "hyprland/window"],
  "modules-center": ["clock"],
  "modules-right": ["network", "pulseaudio", "battery", "tray"],
  "clock": {
    "format": "{:%H:%M}"
  },
  "battery": {
    "format": "{capacity}% {icon}"
  },
  "network": {
    "format-wifi": "{essid} ({signalStrength}%)"
  },
  "pulseaudio": {
    "format": "{volume}% {icon}"
  },
  "tray": {
    "spacing": 10
  }
}
`

func TestParserFindsWaybarKeys(t *testing.T) {
	p := newParser()

	tests := []struct {
		key  string
		want string
	}{
		{"position", "top"},
		{"layer", "top"},
		{"height", "30"},
		{"clock.format", "{:%H:%M}"},
		{"battery.format", "{capacity}% {icon}"},
		{"network.format-wifi", "{essid} ({signalStrength}%)"},
		{"pulseaudio.format", "{volume}% {icon}"},
		{"tray.spacing", "10"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got, ok := p.FindValue([]byte(sampleConfig), tt.key)
			if !ok {
				t.Fatalf("FindValue(%q) not found", tt.key)
			}
			if got != tt.want {
				t.Fatalf("FindValue(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestParserReadsModuleLists(t *testing.T) {
	p := newParser()

	vals, ok := p.FindValues([]byte(sampleConfig), "modules-right")
	if !ok {
		t.Fatal("modules-right not found")
	}
	want := []string{"network", "pulseaudio", "battery", "tray"}
	if len(vals) != len(want) {
		t.Fatalf("len(modules-right) = %d, want %d", len(vals), len(want))
	}
	for i := range want {
		if vals[i] != want[i] {
			t.Fatalf("modules-right[%d] = %q, want %q", i, vals[i], want[i])
		}
	}
}

func TestParserSetsWaybarValues(t *testing.T) {
	p := newParser()

	out, err := p.SetValue([]byte(sampleConfig), "tray.spacing", "12")
	if err != nil {
		t.Fatal(err)
	}
	out, err = p.SetValue(out, "clock.format", "{:%a %H:%M}")
	if err != nil {
		t.Fatal(err)
	}

	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	tray := got["tray"].(map[string]any)
	if tray["spacing"] != float64(12) {
		t.Fatalf("tray.spacing = %#v, want 12", tray["spacing"])
	}
	clock := got["clock"].(map[string]any)
	if clock["format"] != "{:%a %H:%M}" {
		t.Fatalf("clock.format = %#v", clock["format"])
	}
}

func TestParserAcceptsJSONC(t *testing.T) {
	p := newParser()
	jsonc := []byte(`{
  // waybar examples often use comments
  "position": "top",
  "clock": {
    "format": "{:%H:%M}", // trailing comma is common in jsonc
  },
}`)

	got, ok := p.FindValue(jsonc, "clock.format")
	if !ok || got != "{:%H:%M}" {
		t.Fatalf("FindValue(clock.format) = %q, %v", got, ok)
	}

	out, err := p.SetValue(jsonc, "position", "bottom")
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("SetValue output is not json: %v\n%s", err, out)
	}
	if parsed["position"] != "bottom" {
		t.Fatalf("position = %#v, want bottom", parsed["position"])
	}
}

func TestParserImplementsKonfableParser(t *testing.T) {
	var _ konfables.Parser = newParser()
}
