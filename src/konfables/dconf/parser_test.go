package dconf

import (
	"bytes"
	"testing"
)

var sampleData = []byte(`/org/gnome/desktop/wm/preferences/button-layout = appmenu:minimize,maximize,close
/org/gnome/desktop/wm/preferences/focus-mode = click
/org/gnome/desktop/wm/preferences/num-workspaces = 4
/org/gnome/desktop/peripherals/touchpad/tap-to-click = true
/org/gnome/desktop/peripherals/touchpad/speed = 0.5
/org/gnome/desktop/peripherals/mouse/accel-profile = adaptive
`)

func TestFindValue(t *testing.T) {
	p := newParser()

	t.Run("existing key", func(t *testing.T) {
		val, ok := p.FindValue(sampleData, "/org/gnome/desktop/wm/preferences/button-layout")
		if !ok {
			t.Fatal("expected to find button-layout")
		}
		if val != "appmenu:minimize,maximize,close" {
			t.Errorf("got %q, want %q", val, "appmenu:minimize,maximize,close")
		}
	})

	t.Run("numeric value", func(t *testing.T) {
		val, ok := p.FindValue(sampleData, "/org/gnome/desktop/wm/preferences/num-workspaces")
		if !ok {
			t.Fatal("expected to find num-workspaces")
		}
		if val != "4" {
			t.Errorf("got %q, want %q", val, "4")
		}
	})

	t.Run("float value", func(t *testing.T) {
		val, ok := p.FindValue(sampleData, "/org/gnome/desktop/peripherals/touchpad/speed")
		if !ok {
			t.Fatal("expected to find speed")
		}
		if val != "0.5" {
			t.Errorf("got %q, want %q", val, "0.5")
		}
	})

	t.Run("missing key", func(t *testing.T) {
		_, ok := p.FindValue(sampleData, "/org/gnome/desktop/wm/preferences/titlebar-font")
		if ok {
			t.Fatal("expected not to find titlebar-font")
		}
	})
}

func TestFindLine(t *testing.T) {
	p := newParser()

	tests := []struct {
		key    string
		want   int
		wantOK bool
	}{
		{"/org/gnome/desktop/wm/preferences/button-layout", 0, true},
		{"/org/gnome/desktop/wm/preferences/focus-mode", 1, true},
		{"/org/gnome/desktop/peripherals/mouse/accel-profile", 5, true},
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
	p := newParser()

	t.Run("replace existing", func(t *testing.T) {
		got, err := p.SetValue(sampleData, "/org/gnome/desktop/wm/preferences/focus-mode", "sloppy")
		if err != nil {
			t.Fatal(err)
		}
		val, ok := p.FindValue(got, "/org/gnome/desktop/wm/preferences/focus-mode")
		if !ok || val != "sloppy" {
			t.Errorf("expected sloppy, got %q (ok=%v)", val, ok)
		}
	})

	t.Run("append new", func(t *testing.T) {
		got, err := p.SetValue(sampleData, "/org/gnome/desktop/wm/preferences/auto-raise", "true")
		if err != nil {
			t.Fatal(err)
		}
		val, ok := p.FindValue(got, "/org/gnome/desktop/wm/preferences/auto-raise")
		if !ok || val != "true" {
			t.Errorf("expected true, got %q (ok=%v)", val, ok)
		}
	})

	t.Run("preserves other keys", func(t *testing.T) {
		got, err := p.SetValue(sampleData, "/org/gnome/desktop/wm/preferences/focus-mode", "mouse")
		if err != nil {
			t.Fatal(err)
		}
		val, ok := p.FindValue(got, "/org/gnome/desktop/wm/preferences/button-layout")
		if !ok || val != "appmenu:minimize,maximize,close" {
			t.Errorf("button-layout should be preserved, got %q", val)
		}
	})
}

func TestDeleteKey(t *testing.T) {
	p := newParser()

	t.Run("delete existing", func(t *testing.T) {
		got, err := p.DeleteKey(sampleData, "/org/gnome/desktop/peripherals/touchpad/tap-to-click")
		if err != nil {
			t.Fatal(err)
		}
		_, ok := p.FindValue(got, "/org/gnome/desktop/peripherals/touchpad/tap-to-click")
		if ok {
			t.Error("expected key to be deleted")
		}
		_, ok = p.FindValue(got, "/org/gnome/desktop/wm/preferences/button-layout")
		if !ok {
			t.Error("other keys should survive delete")
		}
	})

	t.Run("delete missing", func(t *testing.T) {
		got, err := p.DeleteKey(sampleData, "nonexistent")
		if err != nil {
			t.Errorf("DeleteKey(missing): %v", err)
		}
		if !bytes.Equal(got, sampleData) {
			t.Error("DeleteKey(missing) should return data unchanged")
		}
	})
}

func TestListKeys(t *testing.T) {
	p := newParser()
	keys := p.ListKeys(sampleData)

	if len(keys) != 6 {
		t.Fatalf("got %d keys, want 6", len(keys))
	}

	want := map[string]bool{
		"/org/gnome/desktop/wm/preferences/button-layout":      true,
		"/org/gnome/desktop/wm/preferences/focus-mode":         true,
		"/org/gnome/desktop/wm/preferences/num-workspaces":     true,
		"/org/gnome/desktop/peripherals/touchpad/tap-to-click": true,
		"/org/gnome/desktop/peripherals/touchpad/speed":        true,
		"/org/gnome/desktop/peripherals/mouse/accel-profile":   true,
	}
	for _, k := range keys {
		if !want[k] {
			t.Errorf("unexpected key: %q", k)
		}
	}
}

func TestRoundTrip(t *testing.T) {
	p := newParser()

	updated, err := p.SetValue(sampleData, "/org/gnome/desktop/wm/preferences/num-workspaces", "6")
	if err != nil {
		t.Fatal(err)
	}
	val, ok := p.FindValue(updated, "/org/gnome/desktop/wm/preferences/num-workspaces")
	if !ok || val != "6" {
		t.Errorf("round-trip failed: got %q", val)
	}

	val, ok = p.FindValue(updated, "/org/gnome/desktop/peripherals/mouse/accel-profile")
	if !ok || val != "adaptive" {
		t.Errorf("accel-profile should survive: got %q", val)
	}
}

func TestSetValueIdempotent(t *testing.T) {
	p := newParser()
	got, err := p.SetValue(sampleData, "/org/gnome/desktop/wm/preferences/num-workspaces", "4")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, sampleData) {
		t.Error("setting same value should be idempotent")
	}
}

func TestRoundTripGolden(t *testing.T) {
	p := newParser()

	src := []byte(`/org/gnome/desktop/wm/preferences/button-layout = appmenu:minimize,maximize,close
/org/gnome/desktop/wm/preferences/focus-mode = click
/org/gnome/desktop/wm/preferences/num-workspaces = 4
/org/gnome/desktop/peripherals/touchpad/tap-to-click = true
/org/gnome/desktop/peripherals/touchpad/speed = 0.5
/org/gnome/desktop/peripherals/mouse/accel-profile = adaptive
`)

	// step 1: modify existing
	out, err := p.SetValue(src, "/org/gnome/desktop/wm/preferences/focus-mode", "sloppy")
	if err != nil {
		t.Fatal(err)
	}
	v, ok := p.FindValue(out, "/org/gnome/desktop/wm/preferences/focus-mode")
	if !ok || v != "sloppy" {
		t.Fatalf("SetValue focus-mode: got %q ok=%v", v, ok)
	}

	// step 2: modify numeric
	out, err = p.SetValue(out, "/org/gnome/desktop/wm/preferences/num-workspaces", "6")
	if err != nil {
		t.Fatal(err)
	}
	v, ok = p.FindValue(out, "/org/gnome/desktop/wm/preferences/num-workspaces")
	if !ok || v != "6" {
		t.Fatalf("SetValue num-workspaces: got %q ok=%v", v, ok)
	}

	// step 3: add new key
	out, err = p.SetValue(out, "/org/gnome/desktop/wm/preferences/auto-raise", "true")
	if err != nil {
		t.Fatal(err)
	}
	v, ok = p.FindValue(out, "/org/gnome/desktop/wm/preferences/auto-raise")
	if !ok || v != "true" {
		t.Fatalf("SetValue auto-raise: got %q ok=%v", v, ok)
	}

	// step 4: untouched keys preserved
	for _, key := range []string{
		"/org/gnome/desktop/wm/preferences/button-layout",
		"/org/gnome/desktop/peripherals/touchpad/tap-to-click",
		"/org/gnome/desktop/peripherals/touchpad/speed",
		"/org/gnome/desktop/peripherals/mouse/accel-profile",
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
	if !keySet["/org/gnome/desktop/wm/preferences/auto-raise"] {
		t.Error("ListKeys missing newly added auto-raise")
	}
	if !keySet["/org/gnome/desktop/wm/preferences/num-workspaces"] {
		t.Error("ListKeys missing modified num-workspaces")
	}
}

func TestPersisterHelpers(t *testing.T) {
	t.Run("stripQuotes", func(t *testing.T) {
		tests := []struct {
			input string
			want  string
		}{
			{"'prefer-dark'", "prefer-dark"},
			{"'Adwaita'", "Adwaita"},
			{"24", "24"},
			{"'value with spaces'", "value with spaces"},
			{"''", ""},
			{"'a'", "a"},
		}
		for _, tt := range tests {
			got := stripQuotes(tt.input)
			if got != tt.want {
				t.Errorf("stripQuotes(%q) = %q, want %q", tt.input, got, tt.want)
			}
		}
	})

	t.Run("toGVariant", func(t *testing.T) {
		tests := []struct {
			input string
			want  string
		}{
			{"true", "true"},
			{"false", "false"},
			{"4", "4"},
			{"0.5", "0.5"},
			{"-0.75", "-0.75"},
			{"click", "'click'"},
			{"appmenu:close", "'appmenu:close'"},
			{"[('xkb', 'us')]", "[('xkb', 'us')]"},
			{"@as []", "@as []"},
		}
		for _, tt := range tests {
			got := toGVariant(tt.input)
			if got != tt.want {
				t.Errorf("toGVariant(%q) = %q, want %q", tt.input, got, tt.want)
			}
		}
	})

	t.Run("isNumeric", func(t *testing.T) {
		if !isNumeric("42") {
			t.Error("42 should be numeric")
		}
		if !isNumeric("-3") {
			t.Error("-3 should be numeric")
		}
		if isNumeric("3.14") {
			t.Error("3.14 should not be integer-numeric")
		}
		if isNumeric("abc") {
			t.Error("abc should not be numeric")
		}
		if isNumeric("") {
			t.Error("empty should not be numeric")
		}
	})

	t.Run("isFloat", func(t *testing.T) {
		if !isFloat("3.14") {
			t.Error("3.14 should be float")
		}
		if !isFloat("-0.5") {
			t.Error("-0.5 should be float")
		}
		if isFloat("42") {
			t.Error("42 should not be float")
		}
		if isFloat("") {
			t.Error("empty should not be float")
		}
	})

	t.Run("xkb read normalization", func(t *testing.T) {
		tests := []struct {
			input string
			want  string
		}{
			{"@as []", ""},
			{"[]", ""},
			{"['caps:ctrl_modifier']", "caps:ctrl_modifier"},
			{"['caps:ctrl_modifier', 'compose:ralt']", "caps:ctrl_modifier,compose:ralt"},
			{"'caps:ctrl_modifier,compose:ralt'", "caps:ctrl_modifier,compose:ralt"},
		}
		for _, tt := range tests {
			got := normalizeDconfValue(xkbOptionsPath, tt.input)
			if got != tt.want {
				t.Errorf("normalizeDconfValue(%q) = %q, want %q", tt.input, got, tt.want)
			}
		}
	})

	t.Run("xkb write serialization", func(t *testing.T) {
		tests := []struct {
			input string
			want  string
		}{
			{"", "@as []"},
			{"caps:ctrl_modifier", "['caps:ctrl_modifier']"},
			{"caps:ctrl_modifier,compose:ralt", "['caps:ctrl_modifier', 'compose:ralt']"},
			{" caps:ctrl_modifier, compose:ralt ", "['caps:ctrl_modifier', 'compose:ralt']"},
			{"['caps:ctrl_modifier']", "['caps:ctrl_modifier']"},
		}
		for _, tt := range tests {
			got := toDconfGVariant(xkbOptionsPath, tt.input)
			if got != tt.want {
				t.Errorf("toDconfGVariant(%q) = %q, want %q", tt.input, got, tt.want)
			}
		}
	})

	t.Run("xkb string escaping", func(t *testing.T) {
		got := quoteGVariantString(`grp:foo\bar's`)
		want := `'grp:foo\\bar\'s'`
		if got != want {
			t.Errorf("quoteGVariantString = %q, want %q", got, want)
		}

		opts, ok := parseGVariantStringArray("['grp:foo\\\\bar\\'s']")
		if !ok || len(opts) != 1 || opts[0] != `grp:foo\bar's` {
			t.Errorf("parseGVariantStringArray = %v ok=%v", opts, ok)
		}
	})
}

func FuzzParser(f *testing.F) {
	f.Add([]byte("/org/gnome/desktop/wm/preferences/focus-mode = click\n"), "/org/gnome/desktop/wm/preferences/focus-mode")
	f.Add([]byte("/org/gnome/desktop/peripherals/touchpad/speed = 0.5\n/org/gnome/desktop/peripherals/mouse/speed = 0.0\n"), "/org/gnome/desktop/peripherals/touchpad/speed")
	f.Add([]byte(""), "missing")
	f.Add([]byte("path/key = value with spaces\n"), "path/key")
	f.Add([]byte("a/b = c\nd/e = f\n"), "a/b")

	p := newParser()
	f.Fuzz(func(t *testing.T, data []byte, key string) {
		p.FindValue(data, key)
		p.FindLine(data, key)
		p.ListKeys(data)
		if out, err := p.SetValue(data, key, "fuzzval"); err == nil {
			p.FindValue(out, key)
			p.ListKeys(out)
		}
		p.DeleteKey(data, key)
	})
}
