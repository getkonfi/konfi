package pacman

import (
	"testing"

	"github.com/eminert/konfi/pkg"
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
	p := newParser()
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
	p := newParser()
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
	p := newParser()

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
	p := newParser()

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
	p := newParser()

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
	unchanged, err := p.DeleteKey([]byte(testConfig), "options.Nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if string(unchanged) != testConfig {
		t.Error("deleting non-existent key should leave config unchanged")
	}
}

func TestListKeys(t *testing.T) {
	p := newParser()
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
	p := newParser()

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

func TestBareDirectiveFalseLikeValuesDoNotInsert(t *testing.T) {
	p := newParser()
	base := []byte("[options]\n")

	for _, value := range []string{"", "false", "0", "no", "off"} {
		out, err := p.SetValue(base, "options.CheckSpace", value)
		if err != nil {
			t.Fatal(err)
		}
		if got, ok := p.FindValue(out, "options.CheckSpace"); ok {
			t.Fatalf("SetValue(CheckSpace, %q) inserted bare directive with value %q:\n%s", value, got, out)
		}
	}
}

func TestBareDirectiveFalseLikeValuesDeleteExisting(t *testing.T) {
	p := newParser()
	base := []byte("[options]\nCheckSpace\n")

	out, err := p.SetValue(base, "options.CheckSpace", "off")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := p.FindValue(out, "options.CheckSpace"); ok {
		t.Fatalf("CheckSpace should be removed after setting off:\n%s", out)
	}
}

func TestCleanMethodAllowsMultiTokenValue(t *testing.T) {
	p := newParser()
	data := []byte("[options]\nCleanMethod = KeepInstalled KeepCurrent\n")

	got, ok := p.FindValue(data, "options.CleanMethod")
	if !ok || got != "KeepInstalled KeepCurrent" {
		t.Fatalf("FindValue(CleanMethod) = %q, %v", got, ok)
	}

	out, err := p.SetValue(data, "options.CleanMethod", "KeepCurrent KeepInstalled")
	if err != nil {
		t.Fatal(err)
	}
	got, ok = p.FindValue(out, "options.CleanMethod")
	if !ok || got != "KeepCurrent KeepInstalled" {
		t.Fatalf("after SetValue(CleanMethod) = %q, %v", got, ok)
	}
}

func TestPacmanSchemaBareDirectiveDefaults(t *testing.T) {
	schema, err := pkg.LoadSchema(schemaData)
	if err != nil {
		t.Fatal(err)
	}

	checkSpace := findField(t, schema, "options.CheckSpace")
	if checkSpace.Default != "false" {
		t.Fatalf("CheckSpace default = %q, want false", checkSpace.Default)
	}

	cleanMethod := findField(t, schema, "options.CleanMethod")
	if cleanMethod.Type != "string" {
		t.Fatalf("CleanMethod type = %q, want string", cleanMethod.Type)
	}
}

func findField(t *testing.T, schema *pkg.Schema, key string) pkg.Field {
	t.Helper()
	for _, section := range schema.Sections {
		for i := range section.Fields {
			field := section.Fields[i]
			if field.Key == key {
				return field
			}
		}
	}
	t.Fatalf("schema field %q not found", key)
	return pkg.Field{}
}
