package pacman

import (
	"testing"
)

func FuzzParser(f *testing.F) {
	f.Add([]byte("[options]\nHoldPkg = pacman glibc\nArchitecture = auto\n"), "options.HoldPkg")
	f.Add([]byte("[options]\nColor\nCheckSpace\n"), "options.Color")
	f.Add([]byte("# comment\n[core]\nInclude = /etc/pacman.d/mirrorlist\n"), "core.Include")
	f.Add([]byte(""), "options.Missing")
	f.Add([]byte("[options]\nILoveCandy\n"), "options.ILoveCandy")

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

func TestFindAll(t *testing.T) {
	p := newParser()
	m := p.FindAll([]byte(testConfig))
	if m["options.Color"] != "true" {
		t.Errorf("FindAll[options.Color] = %q", m["options.Color"])
	}
	if m["options.HoldPkg"] != "pacman glibc" {
		t.Errorf("FindAll[options.HoldPkg] = %q", m["options.HoldPkg"])
	}
}

func TestDeleteMissingKey(t *testing.T) {
	p := newParser()
	out, err := p.DeleteKey([]byte(testConfig), "options.Nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != string(testConfig) {
		t.Error("deleting missing key should be no-op")
	}
}
