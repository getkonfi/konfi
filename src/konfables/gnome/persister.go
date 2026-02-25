package gnome

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

// managedKey describes a gsettings schema + key pair.
type managedKey struct {
	Schema string
	Key    string
}

// managedKeys lists all gsettings keys that the GNOME konfable manages.
var managedKeys = []managedKey{
	// org.gnome.desktop.interface
	{"org.gnome.desktop.interface", "color-scheme"},
	{"org.gnome.desktop.interface", "gtk-theme"},
	{"org.gnome.desktop.interface", "icon-theme"},
	{"org.gnome.desktop.interface", "cursor-theme"},
	{"org.gnome.desktop.interface", "cursor-size"},
	{"org.gnome.desktop.interface", "font-name"},
	{"org.gnome.desktop.interface", "document-font-name"},
	{"org.gnome.desktop.interface", "monospace-font-name"},
	{"org.gnome.desktop.interface", "text-scaling-factor"},
	{"org.gnome.desktop.interface", "clock-format"},
	{"org.gnome.desktop.interface", "clock-show-seconds"},
	{"org.gnome.desktop.interface", "clock-show-weekday"},
	{"org.gnome.desktop.interface", "show-battery-percentage"},
	{"org.gnome.desktop.interface", "enable-animations"},
	// org.gnome.desktop.background
	{"org.gnome.desktop.background", "picture-uri"},
	{"org.gnome.desktop.background", "picture-options"},
	{"org.gnome.desktop.background", "primary-color"},
	{"org.gnome.desktop.background", "secondary-color"},
}

// GsettingsPersister implements pkg.Persister for GNOME gsettings.
// it does NOT implement pkg.Watchable — there's no cheap way to watch dconf changes.
type GsettingsPersister struct{}

// Load runs gsettings get for each managed key and assembles synthetic bytes.
// format: "schema/key = value\n" per line.
func (gp *GsettingsPersister) Load(ctx context.Context) ([]byte, error) {
	var buf bytes.Buffer
	for _, mk := range managedKeys {
		val, err := gsettingsGet(ctx, mk.Schema, mk.Key)
		if err != nil {
			// skip keys that don't exist on this system
			continue
		}
		fmt.Fprintf(&buf, "%s/%s = %s\n", mk.Schema, mk.Key, val)
	}
	return buf.Bytes(), nil
}

// Save diffs original vs data and calls gsettings set only for changed keys.
func (gp *GsettingsPersister) Save(ctx context.Context, original, data []byte) error {
	origMap := parseFlat(original)
	newMap := parseFlat(data)

	// collect changed keys
	var errs []string
	for key, newVal := range newMap {
		if origVal, ok := origMap[key]; ok && origVal == newVal {
			continue
		}
		schema, gsKey, ok := splitFlatKey(key)
		if !ok {
			continue
		}
		if err := gsettingsSet(ctx, schema, gsKey, newVal); err != nil {
			errs = append(errs, fmt.Sprintf("%s/%s: %v", schema, gsKey, err))
		}
	}
	if len(errs) > 0 {
		sort.Strings(errs)
		return fmt.Errorf("gsettings set failed: %s", strings.Join(errs, "; "))
	}
	return nil
}

// parseFlat parses "schema/key = value" lines into a map.
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

// cutKV splits "key = value" into (key, value, true).
func cutKV(s string) (string, string, bool) {
	idx := strings.Index(s, " = ")
	if idx < 0 {
		return "", "", false
	}
	return s[:idx], s[idx+3:], true
}

// splitFlatKey splits "org.gnome.desktop.interface/color-scheme" into (schema, key, true).
func splitFlatKey(flat string) (string, string, bool) {
	idx := strings.LastIndex(flat, "/")
	if idx < 0 {
		return "", "", false
	}
	return flat[:idx], flat[idx+1:], true
}

// gsettingsGet runs gsettings get and strips the single-quote wrapping.
func gsettingsGet(ctx context.Context, schema, key string) (string, error) {
	out, err := exec.CommandContext(ctx, "gsettings", "get", schema, key).Output()
	if err != nil {
		return "", err
	}
	val := strings.TrimSpace(string(out))
	// gsettings wraps string values in single quotes: 'value'
	val = stripQuotes(val)
	return val, nil
}

// gsettingsSet runs gsettings set with the given value.
func gsettingsSet(ctx context.Context, schema, key, value string) error {
	return exec.CommandContext(ctx, "gsettings", "set", schema, key, value).Run()
}

// stripQuotes removes surrounding single quotes from gsettings output.
func stripQuotes(s string) string {
	if len(s) >= 2 && s[0] == '\'' && s[len(s)-1] == '\'' {
		return s[1 : len(s)-1]
	}
	return s
}
