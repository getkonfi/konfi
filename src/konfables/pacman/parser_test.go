package pacman

import (
	"testing"
)

const testConfig = `#
# /etc/pacman.conf
#

[options]
HoldPkg = pacman glibc
Architecture = auto
Color
CheckSpace
ParallelDownloads = 5
SigLevel = Required DatabaseOptional
LocalFileSigLevel = Optional

#VerbosePkgLists
ILoveCandy

[core]
Include = /etc/pacman.d/mirrorlist

[extra]
Include = /etc/pacman.d/mirrorlist
`

func TestFindValue(t *testing.T) {
	p := &parser{}
	tests := []struct {
		key  string
		want string
		ok   bool
	}{
		{"options.HoldPkg", "pacman glibc", true},
		{"options.Architecture", "auto", true},
		{"options.Color", "true", true},
		{"options.CheckSpace", "true", true},
		{"options.ILoveCandy", "true", true},
		{"options.ParallelDownloads", "5", true},
		{"options.SigLevel", "Required DatabaseOptional", true},
		{"options.VerbosePkgLists", "", false}, // commented out
		{"options.Missing", "", false},
		{"core.Include", "/etc/pacman.d/mirrorlist", true},
	}
	for _, tt := range tests {
		got, ok := p.FindValue([]byte(testConfig), tt.key)
		if ok != tt.ok || got != tt.want {
			t.Errorf("FindValue(%q) = %q, %v; want %q, %v", tt.key, got, ok, tt.want, tt.ok)
		}
	}
}

func TestFindLine(t *testing.T) {
	p := &parser{}
	tests := []struct {
		key    string
		wantOk bool
	}{
		{"options.HoldPkg", true},
		{"options.Color", true},
		{"options.ILoveCandy", true},
		{"options.Missing", false},
	}
	for _, tt := range tests {
		_, ok := p.FindLine([]byte(testConfig), tt.key)
		if ok != tt.wantOk {
			t.Errorf("FindLine(%q) found=%v; want %v", tt.key, ok, tt.wantOk)
		}
	}
}

func TestSetValueKV(t *testing.T) {
	p := &parser{}

	// replace existing key=value
	data, err := p.SetValue([]byte(testConfig), "options.ParallelDownloads", "10")
	if err != nil {
		t.Fatal(err)
	}
	got, ok := p.FindValue(data, "options.ParallelDownloads")
	if !ok || got != "10" {
		t.Errorf("after SetValue: got %q, %v; want 10, true", got, ok)
	}

	// insert new key in existing section
	data, err = p.SetValue([]byte(testConfig), "options.XferCommand", "/usr/bin/curl -L -C - -f -o %%o %%u")
	if err != nil {
		t.Fatal(err)
	}
	got, ok = p.FindValue(data, "options.XferCommand")
	if !ok || got != "/usr/bin/curl -L -C - -f -o %%o %%u" {
		t.Errorf("after SetValue new key: got %q, %v", got, ok)
	}
}

func TestSetValueBareDirective(t *testing.T) {
	p := &parser{}

	// enable a bare directive that doesn't exist
	data, err := p.SetValue([]byte(testConfig), "options.VerbosePkgLists", "true")
	if err != nil {
		t.Fatal(err)
	}
	got, ok := p.FindValue(data, "options.VerbosePkgLists")
	if !ok || got != "true" {
		t.Errorf("after enable: got %q, %v; want true, true", got, ok)
	}

	// disable an existing bare directive
	data, err = p.SetValue([]byte(testConfig), "options.Color", "false")
	if err != nil {
		t.Fatal(err)
	}
	_, ok = p.FindValue(data, "options.Color")
	if ok {
		t.Error("Color should be removed after setting false")
	}

	// no-op: setting existing bare directive to true
	data, err = p.SetValue([]byte(testConfig), "options.ILoveCandy", "true")
	if err != nil {
		t.Fatal(err)
	}
	got, ok = p.FindValue(data, "options.ILoveCandy")
	if !ok || got != "true" {
		t.Errorf("ILoveCandy should still be present: got %q, %v", got, ok)
	}
}

func TestDeleteKey(t *testing.T) {
	p := &parser{}

	// delete key=value
	data, err := p.DeleteKey([]byte(testConfig), "options.ParallelDownloads")
	if err != nil {
		t.Fatal(err)
	}
	_, ok := p.FindValue(data, "options.ParallelDownloads")
	if ok {
		t.Error("ParallelDownloads should be deleted")
	}

	// delete bare directive
	data, err = p.DeleteKey([]byte(testConfig), "options.Color")
	if err != nil {
		t.Fatal(err)
	}
	_, ok = p.FindValue(data, "options.Color")
	if ok {
		t.Error("Color should be deleted")
	}

	// delete non-existent key is no-op
	data, err = p.DeleteKey([]byte(testConfig), "options.Nonexistent")
	if err != nil {
		t.Fatal(err)
	}
}

func TestListKeys(t *testing.T) {
	p := &parser{}
	keys := p.ListKeys([]byte(testConfig))
	expected := map[string]bool{
		"options.HoldPkg":           true,
		"options.Architecture":      true,
		"options.Color":             true,
		"options.CheckSpace":        true,
		"options.ParallelDownloads": true,
		"options.SigLevel":          true,
		"options.LocalFileSigLevel": true,
		"options.ILoveCandy":        true,
		"core.Include":              true,
		"extra.Include":             true,
	}
	if len(keys) != len(expected) {
		t.Errorf("ListKeys: got %d keys %v, want %d", len(keys), keys, len(expected))
	}
	for _, k := range keys {
		if !expected[k] {
			t.Errorf("unexpected key: %q", k)
		}
	}
}

func TestRoundTrip(t *testing.T) {
	p := &parser{}

	// set then find
	data, err := p.SetValue([]byte(testConfig), "options.ParallelDownloads", "3")
	if err != nil {
		t.Fatal(err)
	}
	got, ok := p.FindValue(data, "options.ParallelDownloads")
	if !ok || got != "3" {
		t.Fatalf("round-trip failed: got %q, %v", got, ok)
	}

	// other keys survive
	got, ok = p.FindValue(data, "options.HoldPkg")
	if !ok || got != "pacman glibc" {
		t.Errorf("HoldPkg should survive: got %q, %v", got, ok)
	}
	got, ok = p.FindValue(data, "options.Color")
	if !ok || got != "true" {
		t.Errorf("Color should survive: got %q, %v", got, ok)
	}
}
