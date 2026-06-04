package gnome

import (
	"context"
	"os/exec"
	"strings"

	"github.com/eminert/konfi/pkg"
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

// NewPersister builds the gsettings-backed persister. it reads each key with
// gsettings get and writes changed keys with gsettings set. Load emits
// "schema/key = value\n" lines; Save addresses each key by splitting that form.
func NewPersister() pkg.Persister {
	return &pkg.CommandPersister[managedKey]{
		Keys:    managedKeys,
		LineKey: func(mk managedKey) string { return mk.Schema + "/" + mk.Key },
		Read:    func(ctx context.Context, mk managedKey) (string, error) { return gsettingsGet(ctx, mk.Schema, mk.Key) },
		Write: func(ctx context.Context, lineKey, value string) error {
			schema, key, ok := splitFlatKey(lineKey)
			if !ok {
				return nil
			}
			return gsettingsSet(ctx, schema, key, value)
		},
		ErrPrefix: "gsettings set",
	}
}

// splitFlatKey splits "org.gnome.desktop.interface/color-scheme" into (schema, key, true).
func splitFlatKey(flat string) (schema, key string, ok bool) {
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
