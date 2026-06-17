package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempSchema(t *testing.T, yaml string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "schema.yaml")
	if err := os.WriteFile(p, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func findSeverity(findings []Finding, sev Severity) []Finding {
	var out []Finding
	for _, f := range findings {
		if f.Severity == sev {
			out = append(out, f)
		}
	}
	return out
}

func TestValidSchema(t *testing.T) {
	path := writeTempSchema(t, `
app: testapp
format: toml
docs_url: https://example.com/docs
sections:
  - name: General
    key: ""
    fields:
      - key: color
        label: Color
        type: color
        default: "#000"
        description: a color
      - key: mode
        label: Mode
        type: enum
        default: dark
        description: theme mode
        options: [dark, light]
`)
	r := checkStructural(path)
	if r.MaxSeverity() > Pass {
		t.Errorf("expected pass, got findings: %+v", r.Findings)
	}
}

func TestMissingDocsURL(t *testing.T) {
	path := writeTempSchema(t, `
app: testapp
format: toml
sections:
  - name: General
    fields:
      - key: x
        label: X
        type: string
        default: ""
        description: test
`)
	r := checkStructural(path)
	warns := findSeverity(r.Findings, Warn)
	found := false
	for _, f := range warns {
		if contains(f.Message, "docs_url") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warn for missing top-level docs_url, got: %+v", r.Findings)
	}
}

func TestMissingFieldDescription(t *testing.T) {
	path := writeTempSchema(t, `
app: testapp
format: toml
docs_url: https://example.com/docs
sections:
  - name: General
    fields:
      - key: x
        label: X
        type: string
        default: ""
        description: ""
`)
	r := checkStructural(path)
	warns := findSeverity(r.Findings, Warn)
	found := false
	for _, f := range warns {
		if contains(f.Message, "missing description") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warn for missing field description, got: %+v", r.Findings)
	}
}

func TestMissingRequiredFields(t *testing.T) {
	path := writeTempSchema(t, `
app: testapp
format: toml
sections:
  - name: General
    fields:
      - key: ""
        label: ""
        type: ""
        default: ""
        description: ""
`)
	r := checkStructural(path)
	fails := findSeverity(r.Findings, Fail)
	if len(fails) < 3 {
		t.Errorf("expected at least 3 fails (key, label, type), got %d: %+v", len(fails), fails)
	}
}

func TestDuplicateKeys(t *testing.T) {
	path := writeTempSchema(t, `
app: testapp
format: toml
sections:
  - name: Section A
    fields:
      - key: foo
        label: Foo
        type: string
        default: ""
        description: first
  - name: Section B
    fields:
      - key: foo
        label: Foo Again
        type: string
        default: ""
        description: duplicate
`)
	r := checkStructural(path)
	fails := findSeverity(r.Findings, Fail)
	found := false
	for _, f := range fails {
		if f.Category == "structural" && contains(f.Message, "duplicate key") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected duplicate key finding, got: %+v", r.Findings)
	}
}

func TestEnumWithoutOptions(t *testing.T) {
	path := writeTempSchema(t, `
app: testapp
format: toml
sections:
  - name: General
    fields:
      - key: mode
        label: Mode
        type: enum
        default: dark
        description: theme
`)
	r := checkStructural(path)
	fails := findSeverity(r.Findings, Fail)
	found := false
	for _, f := range fails {
		if contains(f.Message, "enum type without options") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected enum-without-options finding, got: %+v", r.Findings)
	}
}

func TestNumberWithoutMinMax(t *testing.T) {
	path := writeTempSchema(t, `
app: testapp
format: toml
sections:
  - name: General
    fields:
      - key: size
        label: Size
        type: number
        default: "10"
        description: font size
`)
	r := checkStructural(path)
	warns := findSeverity(r.Findings, Warn)
	if len(warns) == 0 {
		t.Errorf("expected warn for number without min/max, got: %+v", r.Findings)
	}
	fails := findSeverity(r.Findings, Fail)
	if len(fails) > 0 {
		t.Errorf("number without min/max should warn, not fail: %+v", fails)
	}
}

func TestInvalidSemver(t *testing.T) {
	path := writeTempSchema(t, `
app: testapp
format: toml
schema_version: not-a-version
sections:
  - name: General
    fields:
      - key: x
        label: X
        type: string
        default: ""
        description: test
        since: bad
`)
	r := checkStructural(path)
	fails := findSeverity(r.Findings, Fail)
	if len(fails) < 2 {
		t.Errorf("expected at least 2 semver fails, got %d: %+v", len(fails), fails)
	}
}

func TestLooseTopLevelAppVersion(t *testing.T) {
	path := writeTempSchema(t, `
app: tmux
format: tmux
max_app_version: "3.6b"
docs_url: https://man7.org/linux/man-pages/man1/tmux.1.html
sections:
  - name: General
    fields:
      - key: mouse
        label: Mouse
        type: enum
        default: "off"
        description: mouse support
        options: ["on", "off"]
`)
	r := checkStructural(path)
	if fails := findSeverity(r.Findings, Fail); len(fails) > 0 {
		t.Errorf("loose top-level app version should pass: %+v", fails)
	}
}

func TestEmptySectionName(t *testing.T) {
	path := writeTempSchema(t, `
app: testapp
format: toml
sections:
  - name: ""
    fields:
      - key: x
        label: X
        type: string
        default: ""
        description: test
`)
	r := checkStructural(path)
	fails := findSeverity(r.Findings, Fail)
	found := false
	for _, f := range fails {
		if contains(f.Message, "empty name") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected empty section name finding, got: %+v", r.Findings)
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
