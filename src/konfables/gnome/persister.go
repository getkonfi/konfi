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
	{"org.gnome.desktop.interface", "enable-animations"},
	{"org.gnome.desktop.interface", "accent-color"},
	{"org.gnome.desktop.interface", "enable-hot-corners"},
	{"org.gnome.desktop.interface", "overlay-scrolling"},
	{"org.gnome.desktop.interface", "scaling-factor"},
	{"org.gnome.desktop.interface", "cursor-blink"},
	{"org.gnome.desktop.interface", "cursor-blink-time"},
	{"org.gnome.desktop.interface", "cursor-blink-timeout"},
	{"org.gnome.desktop.interface", "gtk-enable-primary-paste"},
	{"org.gnome.desktop.interface", "locate-pointer"},
	{"org.gnome.desktop.interface", "toolkit-accessibility"},
	{"org.gnome.desktop.interface", "font-name"},
	{"org.gnome.desktop.interface", "document-font-name"},
	{"org.gnome.desktop.interface", "monospace-font-name"},
	{"org.gnome.desktop.interface", "text-scaling-factor"},
	{"org.gnome.desktop.interface", "font-antialiasing"},
	{"org.gnome.desktop.interface", "font-hinting"},
	{"org.gnome.desktop.interface", "font-rendering"},
	{"org.gnome.desktop.interface", "font-rgba-order"},
	{"org.gnome.desktop.interface", "clock-format"},
	{"org.gnome.desktop.interface", "clock-show-date"},
	{"org.gnome.desktop.interface", "clock-show-seconds"},
	{"org.gnome.desktop.interface", "clock-show-weekday"},
	{"org.gnome.desktop.interface", "show-battery-percentage"},
	// org.gnome.desktop.background
	{"org.gnome.desktop.background", "picture-uri"},
	{"org.gnome.desktop.background", "picture-uri-dark"},
	{"org.gnome.desktop.background", "picture-options"},
	{"org.gnome.desktop.background", "primary-color"},
	{"org.gnome.desktop.background", "secondary-color"},
	{"org.gnome.desktop.background", "color-shading-type"},
	{"org.gnome.desktop.background", "picture-opacity"},
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
		Delete: func(ctx context.Context, lineKey string) error {
			schema, key, ok := splitFlatKey(lineKey)
			if !ok {
				return nil
			}
			return gsettingsReset(ctx, schema, key)
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
	val = normalizeGSettingsValue(schema, key, val)
	return val, nil
}

// gsettingsSet runs gsettings set with the given value.
func gsettingsSet(ctx context.Context, schema, key, value string) error {
	value = serializeGSettingsValue(schema, key, value)
	return exec.CommandContext(ctx, "gsettings", "set", schema, key, value).Run()
}

func gsettingsReset(ctx context.Context, schema, key string) error {
	return exec.CommandContext(ctx, "gsettings", "reset", schema, key).Run()
}

// stripQuotes removes surrounding single quotes from gsettings output.
func stripQuotes(s string) string {
	if len(s) >= 2 && s[0] == '\'' && s[len(s)-1] == '\'' {
		return s[1 : len(s)-1]
	}
	return s
}

func normalizeGSettingsValue(schema, key, value string) string {
	if schema == "org.gnome.desktop.interface" && key == "scaling-factor" {
		return normalizeUnsignedGVariant(value)
	}
	return value
}

func serializeGSettingsValue(schema, key, value string) string {
	if schema == "org.gnome.desktop.interface" && key == "scaling-factor" {
		if n := normalizeUnsignedGVariant(value); isUnsignedInteger(n) {
			return "uint32 " + n
		}
	}
	return value
}

func normalizeUnsignedGVariant(value string) string {
	parts := strings.Fields(value)
	if len(parts) == 2 && strings.HasPrefix(parts[0], "uint") && isUnsignedInteger(parts[1]) {
		return parts[1]
	}
	return value
}

func isUnsignedInteger(value string) bool {
	if value == "" {
		return false
	}
	for _, c := range value {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
