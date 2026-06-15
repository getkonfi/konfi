package hypridle

import (
	"strings"
	"testing"
)

var sampleConfig = []byte(`general {
    lock_cmd = pidof hyprlock || hyprlock
    before_sleep_cmd = loginctl lock-session
    after_sleep_cmd = hyprctl dispatch 'hl.dsp.dpms({ action = "enable" })'
    ignore_dbus_inhibit = false
    inhibit_sleep = 2
}

listener {
    timeout = 150
    on-timeout = brightnessctl -s set 10
    on-resume = brightnessctl -r
}

listener {
    timeout = 300
    on-timeout = loginctl lock-session
    ignore_inhibit = true
}
`)

func TestFindValueGeneral(t *testing.T) {
	p := newParser()
	got, ok := p.FindValue(sampleConfig, "general.lock_cmd")
	if !ok {
		t.Fatal("expected general.lock_cmd")
	}
	if got != "pidof hyprlock || hyprlock" {
		t.Fatalf("general.lock_cmd = %q", got)
	}
}

func TestFindValueListeners(t *testing.T) {
	p := newParser()
	got, ok := p.FindValue(sampleConfig, listenersKey)
	if !ok {
		t.Fatal("expected listeners")
	}
	want := strings.Join([]string{
		"150 <-> brightnessctl -s set 10 <-> brightnessctl -r <-> ",
		"300 <-> loginctl lock-session <->  <-> true",
	}, "\n")
	if got != want {
		t.Fatalf("listeners = %q, want %q", got, want)
	}
}

func TestSetValueListenersRewritesListenerBlocks(t *testing.T) {
	p := newParser()
	value := strings.Join([]string{
		"300 <-> loginctl lock-session <->  <-> false",
		"900 <-> hyprctl dispatch dpms off <-> hyprctl dispatch dpms on <-> ",
	}, "\n")

	out, err := p.SetValue(sampleConfig, listenersKey, value)
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)
	for _, want := range []string{
		"lock_cmd = pidof hyprlock || hyprlock",
		"timeout = 300",
		"on-timeout = loginctl lock-session",
		"ignore_inhibit = false",
		"timeout = 900",
		"on-timeout = hyprctl dispatch dpms off",
		"on-resume = hyprctl dispatch dpms on",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("rewritten config missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "brightnessctl") {
		t.Fatalf("old listener remained:\n%s", got)
	}
}

func TestListKeysUsesSyntheticListenersKey(t *testing.T) {
	p := newParser()
	keys := p.ListKeys(sampleConfig)

	seen := make(map[string]bool, len(keys))
	for _, key := range keys {
		seen[key] = true
		if strings.HasPrefix(key, "listener.") {
			t.Fatalf("ListKeys exposed raw listener key %q", key)
		}
	}
	if !seen["general.lock_cmd"] {
		t.Fatalf("ListKeys missing general.lock_cmd: %v", keys)
	}
	if !seen[listenersKey] {
		t.Fatalf("ListKeys missing listeners: %v", keys)
	}
}
