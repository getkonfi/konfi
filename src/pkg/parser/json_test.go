package parser

import (
	"encoding/json"
	"strings"
	"testing"
)

const testJSON = `{
  "apiKey": "sk-123",
  "permissions": {
    "allow": ["Read", "Write"],
    "deny": ["Bash"]
  },
  "verbose": true,
  "maxTokens": 4096,
  "statusLine": {
    "command": "status",
    "enabled": false
  },
  "hooks": {
    "preCommit": {
      "command": "lint"
    }
  }
}
`

func TestJSONFindValue(t *testing.T) {
	p := &JSONParser{}

	tests := []struct {
		key  string
		want string
		ok   bool
	}{
		{"apiKey", "sk-123", true},
		{"verbose", "true", true},
		{"maxTokens", "4096", true},
		{"permissions.allow", `["Read","Write"]`, true},
		{"permissions.deny", `["Bash"]`, true},
		{"statusLine.command", "status", true},
		{"statusLine.enabled", "false", true},
		{"hooks.preCommit.command", "lint", true},
		{"nonexistent", "", false},
		{"permissions.nonexistent", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got, ok := p.FindValue([]byte(testJSON), tt.key)
			if ok != tt.ok {
				t.Fatalf("FindValue(%q) ok = %v, want %v", tt.key, ok, tt.ok)
			}
			if got != tt.want {
				t.Errorf("FindValue(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestJSONFindValueAtPath(t *testing.T) {
	p := &JSONParser{}
	got, ok := p.FindValueAtPath([]byte(testJSON), []string{"statusLine", "command"})
	if !ok || got != "status" {
		t.Errorf("FindValueAtPath([statusLine,command]) = (%q, %v), want (\"status\", true)", got, ok)
	}
}

func TestJSONFindValueObject(t *testing.T) {
	p := &JSONParser{}
	got, ok := p.FindValue([]byte(testJSON), "statusLine")
	if !ok {
		t.Fatal("expected to find statusLine")
	}
	// should be a compact JSON object
	var m map[string]any
	if err := json.Unmarshal([]byte(got), &m); err != nil {
		t.Fatalf("FindValue(statusLine) not valid JSON: %v", err)
	}
	if m["command"] != "status" {
		t.Errorf("expected command=status in object, got %v", m["command"])
	}
}

func TestJSONFindLine(t *testing.T) {
	p := &JSONParser{}

	tests := []struct {
		key      string
		wantLine int
		ok       bool
	}{
		{"apiKey", 1, true},
		{"verbose", 6, true},
		{"permissions.allow", 3, true},
		{"statusLine.command", 9, true},
		// hooks.preCommit.command should be found at the correct depth, not statusLine.command
		{"hooks.preCommit.command", 14, true},
		{"nonexistent", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got, ok := p.FindLine([]byte(testJSON), tt.key)
			if ok != tt.ok {
				t.Fatalf("FindLine(%q) ok = %v, want %v", tt.key, ok, tt.ok)
			}
			if got != tt.wantLine {
				t.Errorf("FindLine(%q) = %d, want %d", tt.key, got, tt.wantLine)
			}
		})
	}
}

func TestJSONFindLineDistinguishesDepth(t *testing.T) {
	// critical: "command" appears at two different depths.
	// statusLine.command != hooks.preCommit.command
	p := &JSONParser{}

	line1, ok1 := p.FindLine([]byte(testJSON), "statusLine.command")
	line2, ok2 := p.FindLine([]byte(testJSON), "hooks.preCommit.command")

	if !ok1 || !ok2 {
		t.Fatalf("expected both to be found: ok1=%v ok2=%v", ok1, ok2)
	}
	if line1 == line2 {
		t.Errorf("statusLine.command and hooks.preCommit.command should be on different lines, both on %d", line1)
	}
}

func TestJSONSetValuePreservesType(t *testing.T) {
	p := &JSONParser{}

	// set bool — should stay bool, not become "false"
	out, err := p.SetValue([]byte(testJSON), "verbose", "false")
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if v, ok := m["verbose"].(bool); !ok || v != false {
		t.Errorf("verbose should be bool false, got %T %v", m["verbose"], m["verbose"])
	}

	// set number — should stay number
	out, err = p.SetValue(out, "maxTokens", "8192")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if v, ok := m["maxTokens"].(float64); !ok || v != 8192 {
		t.Errorf("maxTokens should be float64 8192, got %T %v", m["maxTokens"], m["maxTokens"])
	}

	// set string — should stay string
	out, err = p.SetValue(out, "apiKey", "sk-456")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if v, ok := m["apiKey"].(string); !ok || v != "sk-456" {
		t.Errorf("apiKey should be string sk-456, got %T %v", m["apiKey"], m["apiKey"])
	}
}

func TestJSONSetValueNewKey(t *testing.T) {
	p := &JSONParser{}

	// new key defaults to string
	out, err := p.SetValue([]byte(testJSON), "newKey", "hello")
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if v, ok := m["newKey"].(string); !ok || v != "hello" {
		t.Errorf("newKey should be string hello, got %T %v", m["newKey"], m["newKey"])
	}
}

func TestJSONSetValueNewBool(t *testing.T) {
	p := &JSONParser{}

	// new key with bool-like value gets inferred as bool
	out, err := p.SetValue([]byte(`{}`), "enabled", "true")
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if v, ok := m["enabled"].(bool); !ok || v != true {
		t.Errorf("enabled should be bool true, got %T %v", m["enabled"], m["enabled"])
	}
}

func TestJSONSetValueNested(t *testing.T) {
	p := &JSONParser{}

	out, err := p.SetValue([]byte(testJSON), "statusLine.command", "newcmd")
	if err != nil {
		t.Fatal(err)
	}

	got, ok := p.FindValue(out, "statusLine.command")
	if !ok || got != "newcmd" {
		t.Errorf("after SetValue, statusLine.command = (%q, %v), want (\"newcmd\", true)", got, ok)
	}
}

func TestJSONSetValueCreatesIntermediateObjects(t *testing.T) {
	p := &JSONParser{}

	out, err := p.SetValue([]byte(`{}`), "a.b.c", "deep")
	if err != nil {
		t.Fatal(err)
	}
	got, ok := p.FindValue(out, "a.b.c")
	if !ok || got != "deep" {
		t.Errorf("a.b.c = (%q, %v), want (\"deep\", true)", got, ok)
	}
}

func TestJSONSetValueAtPath(t *testing.T) {
	p := &JSONParser{}
	out, err := p.SetValueAtPath([]byte(testJSON), []string{"statusLine", "enabled"}, "true")
	if err != nil {
		t.Fatal(err)
	}
	got, ok := p.FindValueAtPath(out, []string{"statusLine", "enabled"})
	if !ok || got != "true" {
		t.Errorf("statusLine.enabled = (%q, %v), want (\"true\", true)", got, ok)
	}
}

func TestJSONDeleteKey(t *testing.T) {
	p := &JSONParser{}

	// delete top-level key
	out, err := p.DeleteKey([]byte(testJSON), "verbose")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := p.FindValue(out, "verbose"); ok {
		t.Error("verbose should be deleted")
	}

	// delete nested key
	out, err = p.DeleteKey([]byte(testJSON), "statusLine.command")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := p.FindValue(out, "statusLine.command"); ok {
		t.Error("statusLine.command should be deleted")
	}
	// parent should still exist
	if _, ok := p.FindValue(out, "statusLine.enabled"); !ok {
		t.Error("statusLine.enabled should still exist")
	}

	// delete nonexistent key returns error
	_, err = p.DeleteKey([]byte(testJSON), "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent key")
	}
}

func TestJSONListKeys(t *testing.T) {
	p := &JSONParser{}

	keys := p.ListKeys([]byte(testJSON))

	expected := []string{
		"apiKey",
		"permissions.allow",
		"permissions.deny",
		"verbose",
		"maxTokens",
		"statusLine.command",
		"statusLine.enabled",
		"hooks.preCommit.command",
	}

	if len(keys) != len(expected) {
		t.Fatalf("ListKeys returned %d keys, want %d: %v", len(keys), len(expected), keys)
	}

	for i, k := range expected {
		if keys[i] != k {
			t.Errorf("keys[%d] = %q, want %q", i, keys[i], k)
		}
	}
}

func TestJSONListKeysArraysAreLeaves(t *testing.T) {
	p := &JSONParser{}
	// arrays should appear as leaves, not expanded into indices
	data := `{"items": ["a", "b", "c"], "name": "test"}`
	keys := p.ListKeys([]byte(data))

	for _, k := range keys {
		if strings.Contains(k, "0") || strings.Contains(k, "1") || strings.Contains(k, "2") {
			t.Errorf("array indices should not appear in keys: %q", k)
		}
	}
	if len(keys) != 2 {
		t.Errorf("expected 2 keys, got %d: %v", len(keys), keys)
	}
}

func TestJSONFindValues(t *testing.T) {
	p := &JSONParser{}

	// array key
	vals, ok := p.FindValues([]byte(testJSON), "permissions.allow")
	if !ok {
		t.Fatal("expected to find permissions.allow")
	}
	if len(vals) != 2 || vals[0] != "Read" || vals[1] != "Write" {
		t.Errorf("FindValues(permissions.allow) = %v, want [Read Write]", vals)
	}

	// non-array key returns single value
	vals, ok = p.FindValues([]byte(testJSON), "apiKey")
	if !ok {
		t.Fatal("expected to find apiKey")
	}
	if len(vals) != 1 || vals[0] != "sk-123" {
		t.Errorf("FindValues(apiKey) = %v, want [sk-123]", vals)
	}
}

func TestJSONSetValues(t *testing.T) {
	p := &JSONParser{}

	out, err := p.SetValues([]byte(testJSON), "permissions.allow", []string{"Read", "Write", "Edit"})
	if err != nil {
		t.Fatal(err)
	}

	vals, ok := p.FindValues(out, "permissions.allow")
	if !ok {
		t.Fatal("expected to find permissions.allow after SetValues")
	}
	if len(vals) != 3 || vals[2] != "Edit" {
		t.Errorf("FindValues after SetValues = %v, want [Read Write Edit]", vals)
	}
}

func TestJSONCapabilities(t *testing.T) {
	p := &JSONParser{}
	caps := p.Capabilities()

	if !caps.SupportsNesting {
		t.Error("expected SupportsNesting = true")
	}
	if !caps.SupportsMultivalue {
		t.Error("expected SupportsMultivalue = true")
	}
	if caps.LosslessRoundtrip {
		t.Error("expected LosslessRoundtrip = false")
	}
	if caps.SupportsComments {
		t.Error("expected SupportsComments = false")
	}
}

func TestJSONSetValueIndentFormat(t *testing.T) {
	p := &JSONParser{}

	out, err := p.SetValue([]byte(`{"a": 1}`), "b", "hello")
	if err != nil {
		t.Fatal(err)
	}

	// should be indented with 2 spaces and end with newline
	s := string(out)
	if !strings.Contains(s, "  ") {
		t.Error("expected 2-space indent")
	}
	if !strings.HasSuffix(s, "\n") {
		t.Error("expected trailing newline")
	}
}

func TestJSONEmptyData(t *testing.T) {
	p := &JSONParser{}

	// FindValue on empty
	if _, ok := p.FindValue(nil, "key"); ok {
		t.Error("expected not found on nil data")
	}

	// ListKeys on empty
	keys := p.ListKeys(nil)
	if len(keys) != 0 {
		t.Errorf("expected 0 keys on nil data, got %d", len(keys))
	}

	// SetValue on empty creates new JSON
	out, err := p.SetValue(nil, "key", "val")
	if err != nil {
		t.Fatal(err)
	}
	got, ok := p.FindValue(out, "key")
	if !ok || got != "val" {
		t.Errorf("after SetValue on nil, key = (%q, %v)", got, ok)
	}
}

func TestJSONMalformedDataReturnsError(t *testing.T) {
	p := &JSONParser{}
	malformed := []byte(`{"key": broken}`)

	_, err := p.SetValue(malformed, "key", "val")
	if err == nil {
		t.Error("SetValue should return error for malformed JSON")
	}

	_, err = p.SetValues(malformed, "key", []string{"a"})
	if err == nil {
		t.Error("SetValues should return error for malformed JSON")
	}

	_, err = p.DeleteKey(malformed, "key")
	if err == nil {
		t.Error("DeleteKey should return error for malformed JSON")
	}
}

func TestJSONKeyOrderPreserved(t *testing.T) {
	input := `{
  "zeta": 1,
  "alpha": 2,
  "mid": 3
}
`
	p := &JSONParser{}
	out, err := p.SetValue([]byte(input), "alpha", "99")
	if err != nil {
		t.Fatal(err)
	}

	keys := p.ListKeys(out)
	expected := []string{"zeta", "alpha", "mid"}
	if len(keys) != len(expected) {
		t.Fatalf("expected %d keys, got %d: %v", len(expected), len(keys), keys)
	}
	for i, k := range expected {
		if keys[i] != k {
			t.Errorf("key[%d] = %q, want %q (order not preserved)", i, keys[i], k)
		}
	}
}
