package dconf

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

// managedPath is a dconf absolute path to a managed key.
type managedPath string

// managedKeys lists all dconf paths that this konfable manages.
var managedKeys = []managedPath{
	// org.gnome.desktop.wm.preferences
	"/org/gnome/desktop/wm/preferences/button-layout",
	"/org/gnome/desktop/wm/preferences/focus-mode",
	"/org/gnome/desktop/wm/preferences/auto-raise",
	"/org/gnome/desktop/wm/preferences/titlebar-font",
	"/org/gnome/desktop/wm/preferences/num-workspaces",
	"/org/gnome/desktop/wm/preferences/resize-with-right-button",
	// org.gnome.desktop.input-sources
	"/org/gnome/desktop/input-sources/xkb-options",
	// org.gnome.desktop.peripherals.touchpad
	"/org/gnome/desktop/peripherals/touchpad/tap-to-click",
	"/org/gnome/desktop/peripherals/touchpad/natural-scroll",
	"/org/gnome/desktop/peripherals/touchpad/speed",
	"/org/gnome/desktop/peripherals/touchpad/two-finger-scrolling-enabled",
	"/org/gnome/desktop/peripherals/touchpad/disable-while-typing",
	"/org/gnome/desktop/peripherals/touchpad/edge-scrolling-enabled",
	// org.gnome.desktop.peripherals.mouse
	"/org/gnome/desktop/peripherals/mouse/natural-scroll",
	"/org/gnome/desktop/peripherals/mouse/speed",
	"/org/gnome/desktop/peripherals/mouse/accel-profile",
}

// DconfPersister implements pkg.Persister for the dconf database.
// it does NOT implement pkg.Watchable — there's no cheap way to watch dconf changes.
type DconfPersister struct{}

// Load runs dconf read for each managed key and assembles synthetic bytes.
// format: "/path/to/key = value\n" per line (matches schema.yaml keys).
func (dp *DconfPersister) Load(ctx context.Context) ([]byte, error) {
	var buf bytes.Buffer
	for _, path := range managedKeys {
		val, err := dconfRead(ctx, string(path))
		if err != nil {
			continue
		}
		fmt.Fprintf(&buf, "%s = %s\n", path, val)
	}
	return buf.Bytes(), nil
}

// Save diffs original vs data and calls dconf write only for changed keys.
func (dp *DconfPersister) Save(ctx context.Context, original, data []byte) error {
	origMap := parseFlat(original)
	newMap := parseFlat(data)

	var errs []string
	for key, newVal := range newMap {
		if origVal, ok := origMap[key]; ok && origVal == newVal {
			continue
		}
		if err := dconfWrite(ctx, key, newVal); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", key, err))
		}
	}
	if len(errs) > 0 {
		sort.Strings(errs)
		return fmt.Errorf("dconf write failed: %s", strings.Join(errs, "; "))
	}
	return nil
}

// parseFlat parses "path/key = value" lines into a map.
func parseFlat(data []byte) map[string]string {
	m := make(map[string]string)
	for _, line := range bytes.Split(data, []byte("\n")) {
		s := strings.TrimSpace(string(line))
		if s == "" || s[0] == '#' {
			continue
		}
		k, v, ok := cutKV(s)
		if ok {
			m[k] = v
		}
	}
	return m
}

// dconfRead runs dconf read and strips wrapping quotes.
func dconfRead(ctx context.Context, path string) (string, error) {
	out, err := exec.CommandContext(ctx, "dconf", "read", path).Output()
	if err != nil {
		return "", err
	}
	val := strings.TrimSpace(string(out))
	if val == "" {
		return "", fmt.Errorf("empty value for %s", path)
	}
	val = stripQuotes(val)
	return val, nil
}

// dconfWrite runs dconf write with the given value.
// dconf write requires GVariant format, so string values need single-quote wrapping.
func dconfWrite(ctx context.Context, path, value string) error {
	// dconf write expects a GVariant-formatted value
	gval := toGVariant(value)
	return exec.CommandContext(ctx, "dconf", "write", path, gval).Run()
}

// stripQuotes removes surrounding single quotes from dconf output.
func stripQuotes(s string) string {
	if len(s) >= 2 && s[0] == '\'' && s[len(s)-1] == '\'' {
		return s[1 : len(s)-1]
	}
	return s
}

// toGVariant wraps a plain value in GVariant format for dconf write.
// booleans, numbers, and array-like values pass through; strings get single-quoted.
func toGVariant(s string) string {
	// booleans
	if s == "true" || s == "false" {
		return s
	}
	// integers (e.g. cursor-size, num-workspaces)
	if isNumeric(s) {
		return s
	}
	// floats (e.g. speed: 0.5)
	if isFloat(s) {
		return s
	}
	// already a GVariant array/tuple (starts with [ or @)
	if len(s) > 0 && (s[0] == '[' || s[0] == '@' || s[0] == '(') {
		return s
	}
	// wrap plain strings
	return "'" + s + "'"
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	start := 0
	if s[0] == '-' || s[0] == '+' {
		start = 1
	}
	for _, c := range s[start:] {
		if c < '0' || c > '9' {
			return false
		}
	}
	return start < len(s)
}

func isFloat(s string) bool {
	if s == "" {
		return false
	}
	dotSeen := false
	start := 0
	if s[0] == '-' || s[0] == '+' {
		start = 1
	}
	for _, c := range s[start:] {
		if c == '.' {
			if dotSeen {
				return false
			}
			dotSeen = true
			continue
		}
		if c < '0' || c > '9' {
			return false
		}
	}
	return dotSeen && start < len(s)
}
