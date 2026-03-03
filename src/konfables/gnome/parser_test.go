package gnome

import (
	"bytes"
	"testing"
)

var sampleData = []byte(`org.gnome.desktop.interface/color-scheme = prefer-dark
org.gnome.desktop.interface/gtk-theme = Adwaita
org.gnome.desktop.interface/cursor-size = 24
org.gnome.desktop.interface/enable-animations = true
org.gnome.desktop.background/primary-color = #023c88
`)

func TestFindValue(t *testing.T) {
	p := &parser{}

	t.Run("existing key", func(t *testing.T) {
		val, ok := p.FindValue(sampleData, "org.gnome.desktop.interface/color-scheme")
		if !ok {
			t.Fatal("expected to find color-scheme")
		}
		if val != "prefer-dark" {
			t.Errorf("got %q, want %q", val, "prefer-dark")
		}
	})

	t.Run("numeric value", func(t *testing.T) {
		val, ok := p.FindValue(sampleData, "org.gnome.desktop.interface/cursor-size")
		if !ok {
			t.Fatal("expected to find cursor-size")
		}
		if val != "24" {
			t.Errorf("got %q, want %q", val, "24")
		}
	})

	t.Run("missing key", func(t *testing.T) {
		_, ok := p.FindValue(sampleData, "org.gnome.desktop.interface/font-name")
		if ok {
			t.Fatal("expected not to find font-name")
		}
	})
}

func TestFindLine(t *testing.T) {
	p := &parser{}

	tests := []struct {
		key    string
		want   int
		wantOK bool
	}{
		{"org.gnome.desktop.interface/color-scheme", 0, true},
		{"org.gnome.desktop.interface/gtk-theme", 1, true},
		{"org.gnome.desktop.background/primary-color", 4, true},
		{"nonexistent", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got, ok := p.FindLine(sampleData, tt.key)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestSetValue(t *testing.T) {
	p := &parser{}

	t.Run("replace existing", func(t *testing.T) {
		got, err := p.SetValue(sampleData, "org.gnome.desktop.interface/color-scheme", "default")
		if err != nil {
			t.Fatal(err)
		}
		val, ok := p.FindValue(got, "org.gnome.desktop.interface/color-scheme")
		if !ok || val != "default" {
			t.Errorf("expected default, got %q (ok=%v)", val, ok)
		}
	})

	t.Run("append new", func(t *testing.T) {
		got, err := p.SetValue(sampleData, "org.gnome.desktop.interface/font-name", "Inter 11")
		if err != nil {
			t.Fatal(err)
		}
		val, ok := p.FindValue(got, "org.gnome.desktop.interface/font-name")
		if !ok || val != "Inter 11" {
			t.Errorf("expected Inter 11, got %q (ok=%v)", val, ok)
		}
	})

	t.Run("preserves other keys", func(t *testing.T) {
		got, err := p.SetValue(sampleData, "org.gnome.desktop.interface/color-scheme", "prefer-light")
		if err != nil {
			t.Fatal(err)
		}
		// other keys untouched
		val, ok := p.FindValue(got, "org.gnome.desktop.interface/gtk-theme")
		if !ok || val != "Adwaita" {
			t.Errorf("gtk-theme should be preserved, got %q", val)
		}
	})
}

func TestDeleteKey(t *testing.T) {
	p := &parser{}

	t.Run("delete existing", func(t *testing.T) {
		got, err := p.DeleteKey(sampleData, "org.gnome.desktop.interface/enable-animations")
		if err != nil {
			t.Fatal(err)
		}
		_, ok := p.FindValue(got, "org.gnome.desktop.interface/enable-animations")
		if ok {
			t.Error("expected key to be deleted")
		}
		// other keys still present
		_, ok = p.FindValue(got, "org.gnome.desktop.interface/color-scheme")
		if !ok {
			t.Error("other keys should survive delete")
		}
	})

	t.Run("delete missing", func(t *testing.T) {
		_, err := p.DeleteKey(sampleData, "nonexistent")
		if err == nil {
			t.Error("expected error when deleting missing key")
		}
	})
}

func TestListKeys(t *testing.T) {
	p := &parser{}
	keys := p.ListKeys(sampleData)

	if len(keys) != 5 {
		t.Fatalf("got %d keys, want 5", len(keys))
	}

	want := map[string]bool{
		"org.gnome.desktop.interface/color-scheme":      true,
		"org.gnome.desktop.interface/gtk-theme":         true,
		"org.gnome.desktop.interface/cursor-size":       true,
		"org.gnome.desktop.interface/enable-animations": true,
		"org.gnome.desktop.background/primary-color":    true,
	}
	for _, k := range keys {
		if !want[k] {
			t.Errorf("unexpected key: %q", k)
		}
	}
}

func TestRoundTrip(t *testing.T) {
	p := &parser{}

	// set a new value, read it back
	updated, err := p.SetValue(sampleData, "org.gnome.desktop.interface/cursor-size", "48")
	if err != nil {
		t.Fatal(err)
	}
	val, ok := p.FindValue(updated, "org.gnome.desktop.interface/cursor-size")
	if !ok || val != "48" {
		t.Errorf("round-trip failed: got %q", val)
	}

	// original value still intact for other keys
	val, ok = p.FindValue(updated, "org.gnome.desktop.background/primary-color")
	if !ok || val != "#023c88" {
		t.Errorf("primary-color should survive: got %q", val)
	}
}

func TestSetValueIdempotent(t *testing.T) {
	p := &parser{}
	got, err := p.SetValue(sampleData, "org.gnome.desktop.interface/cursor-size", "24")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, sampleData) {
		t.Error("setting same value should be idempotent")
	}
}

func TestRoundTripGolden(t *testing.T) {
	p := &parser{}

	src := []byte(`org.gnome.desktop.interface/color-scheme = prefer-dark
org.gnome.desktop.interface/gtk-theme = Adwaita
org.gnome.desktop.interface/cursor-size = 24
org.gnome.desktop.interface/enable-animations = true
org.gnome.desktop.interface/font-name = Cantarell 11
org.gnome.desktop.background/primary-color = #023c88
org.gnome.desktop.background/picture-options = zoom
`)

	// step 1: modify an existing value
	out, err := p.SetValue(src, "org.gnome.desktop.interface/color-scheme", "default")
	if err != nil {
		t.Fatal(err)
	}
	v, ok := p.FindValue(out, "org.gnome.desktop.interface/color-scheme")
	if !ok || v != "default" {
		t.Fatalf("SetValue color-scheme: got %q ok=%v", v, ok)
	}

	// step 2: modify a numeric value
	out, err = p.SetValue(out, "org.gnome.desktop.interface/cursor-size", "48")
	if err != nil {
		t.Fatal(err)
	}
	v, ok = p.FindValue(out, "org.gnome.desktop.interface/cursor-size")
	if !ok || v != "48" {
		t.Fatalf("SetValue cursor-size: got %q ok=%v", v, ok)
	}

	// step 3: add a new key
	out, err = p.SetValue(out, "org.gnome.desktop.interface/icon-theme", "Papirus")
	if err != nil {
		t.Fatal(err)
	}
	v, ok = p.FindValue(out, "org.gnome.desktop.interface/icon-theme")
	if !ok || v != "Papirus" {
		t.Fatalf("SetValue icon-theme: got %q ok=%v", v, ok)
	}

	// step 4: verify untouched keys preserved
	for _, key := range []string{
		"org.gnome.desktop.interface/gtk-theme",
		"org.gnome.desktop.interface/enable-animations",
		"org.gnome.desktop.interface/font-name",
		"org.gnome.desktop.background/primary-color",
		"org.gnome.desktop.background/picture-options",
	} {
		if _, ok := p.FindValue(out, key); !ok {
			t.Errorf("key %q lost during round-trip", key)
		}
	}

	// step 5: ListKeys covers everything
	keys := p.ListKeys(out)
	keySet := make(map[string]bool)
	for _, k := range keys {
		keySet[k] = true
	}
	if !keySet["org.gnome.desktop.interface/icon-theme"] {
		t.Error("ListKeys missing newly added icon-theme")
	}
	if !keySet["org.gnome.desktop.interface/cursor-size"] {
		t.Error("ListKeys missing modified cursor-size")
	}
}

func FuzzParser(f *testing.F) {
	f.Add([]byte("org.gnome.desktop.interface/color-scheme = prefer-dark\n"), "org.gnome.desktop.interface/color-scheme")
	f.Add([]byte("org.gnome.desktop.interface/cursor-size = 24\norg.gnome.desktop.background/primary-color = #023c88\n"), "org.gnome.desktop.interface/cursor-size")
	f.Add([]byte(""), "missing")
	f.Add([]byte("schema/key = value with spaces\n"), "schema/key")
	f.Add([]byte("a/b = c\nd/e = f\n"), "a/b")

	p := &parser{}
	f.Fuzz(func(t *testing.T, data []byte, key string) {
		p.FindValue(data, key)
		p.FindLine(data, key)
		p.ListKeys(data)
		if out, err := p.SetValue(data, key, "fuzzval"); err == nil {
			p.FindValue(out, key)
			p.ListKeys(out)
		}
		// DeleteKey returns error for missing keys in gnome, but should not panic
		p.DeleteKey(data, key)
	})
}
