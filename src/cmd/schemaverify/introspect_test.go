package main

import (
	"testing"
)

func TestParseGhosttyDump(t *testing.T) {
	input := []byte(`# font-family
#
# The font families to use.
#
font-family = JetBrains Mono
# font-size
#
# Font size in points.
#
font-size = 13
# bold-is-bright
bold-is-bright = false
background = #1e1e2e
foreground = #cdd6f4
`)
	keys := parseGhosttyDump(input)

	want := map[string]bool{
		"font-family":     true,
		"font-size":       true,
		"bold-is-bright":  true,
		"background":      true,
		"foreground":      true,
	}

	if len(keys) != len(want) {
		t.Errorf("got %d keys, want %d: %v", len(keys), len(want), keys)
	}
	for _, k := range keys {
		if !want[k] {
			t.Errorf("unexpected key %q", k)
		}
	}
}

func TestParseGhosttyDumpDedup(t *testing.T) {
	input := []byte(`font-family = JetBrains Mono
font-family = Noto Color Emoji
font-size = 13
`)
	keys := parseGhosttyDump(input)
	count := 0
	for _, k := range keys {
		if k == "font-family" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("font-family appeared %d times, want 1", count)
	}
}

func TestParseTmuxDump(t *testing.T) {
	input := []byte(`default-terminal "tmux-256color"
escape-time 10
focus-events on
history-limit 50000
mouse on
set-clipboard on
status-position top
`)
	keys := parseTmuxDump(input)

	want := map[string]bool{
		"default-terminal": true,
		"escape-time":      true,
		"focus-events":     true,
		"history-limit":    true,
		"mouse":            true,
		"set-clipboard":    true,
		"status-position":  true,
	}

	if len(keys) != len(want) {
		t.Errorf("got %d keys, want %d: %v", len(keys), len(want), keys)
	}
	for _, k := range keys {
		if !want[k] {
			t.Errorf("unexpected key %q", k)
		}
	}
}

func TestParseGitDump(t *testing.T) {
	input := []byte(`user.name=Emin
user.email=emin@example.com
core.editor=nvim
core.pager=delta
init.defaultbranch=main
pull.rebase=true
push.autosetupremote=true
`)
	keys := parseGitDump(input)

	want := map[string]bool{
		"user.name":             true,
		"user.email":            true,
		"core.editor":           true,
		"core.pager":            true,
		"init.defaultbranch":    true,
		"pull.rebase":           true,
		"push.autosetupremote":  true,
	}

	if len(keys) != len(want) {
		t.Errorf("got %d keys, want %d: %v", len(keys), len(want), keys)
	}
	for _, k := range keys {
		if !want[k] {
			t.Errorf("unexpected key %q", k)
		}
	}
}

func TestParseGitDumpDedup(t *testing.T) {
	input := []byte(`user.name=Emin
user.name=Override
core.editor=nvim
`)
	keys := parseGitDump(input)
	count := 0
	for _, k := range keys {
		if k == "user.name" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("user.name appeared %d times, want 1", count)
	}
}

func TestParseDconfDump(t *testing.T) {
	input := []byte(`[wm/preferences/]
button-layout='appmenu:minimize,maximize,close'
num-workspaces=4

[interface/]
color-scheme='prefer-dark'
font-name='Cantarell 11'
`)
	keys := parseDconfDump(input)

	want := map[string]bool{
		"/org/gnome/desktop/wm/preferences/button-layout": true,
		"/org/gnome/desktop/wm/preferences/num-workspaces": true,
		"/org/gnome/desktop/interface/color-scheme":         true,
		"/org/gnome/desktop/interface/font-name":            true,
	}

	if len(keys) != len(want) {
		t.Errorf("got %d keys, want %d: %v", len(keys), len(want), keys)
	}
	for _, k := range keys {
		if !want[k] {
			t.Errorf("unexpected key %q", k)
		}
	}
}

func TestParseGnomeDump(t *testing.T) {
	input := []byte(`org.gnome.desktop.interface color-scheme 'prefer-dark'
org.gnome.desktop.interface font-name 'Cantarell 11'
org.gnome.desktop.background picture-uri 'file:///usr/share/backgrounds/default.png'
`)
	keys := parseGnomeDump(input)

	want := map[string]bool{
		"org.gnome.desktop.interface/color-scheme":  true,
		"org.gnome.desktop.interface/font-name":     true,
		"org.gnome.desktop.background/picture-uri":  true,
	}

	if len(keys) != len(want) {
		t.Errorf("got %d keys, want %d: %v", len(keys), len(want), keys)
	}
	for _, k := range keys {
		if !want[k] {
			t.Errorf("unexpected key %q", k)
		}
	}
}
