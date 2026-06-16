package main

import (
	"reflect"
	"testing"
)

func TestParseHyprlandFields(t *testing.T) {
	got := parseHyprlandFields("", []byte(`
        MS<Int>("general:border_size", "size", 1),
        MS<Bool>("decoration:blur:enabled", "blur", true),
`))
	want := []string{"decoration.blur.enabled", "general.border_size"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("fields = %#v, want %#v", got, want)
	}
}

func TestParseKittyFields(t *testing.T) {
	got := parseKittyFields("", []byte(`
opt('font_family', 'monospace')
    opt('scrollback_lines', 2000)
# opt('commented_out', true)
`))
	want := []string{"font_family", "scrollback_lines"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("fields = %#v, want %#v", got, want)
	}
}

func TestParseRioFields(t *testing.T) {
	got := parseRioFields("rio-backend/src/config/mod.rs", []byte(`
pub struct Config {
    #[serde(default = "default_bool_true", rename = "confirm-before-quit")]
    pub confirm_before_quit: bool,
    #[serde(
        default = "Option::default",
        rename = "adaptive-theme"
    )]
    pub adaptive_theme: Option<AdaptiveTheme>,
    #[serde(default)]
    pub cursor: CursorConfig,
}
`))
	want := []string{
		"config.adaptive-theme",
		"config.confirm-before-quit",
		"config.cursor",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("fields = %#v, want %#v", got, want)
	}
}

func TestSetDiff(t *testing.T) {
	got := setDiff([]string{"b", "c", "a"}, []string{"a", "b"})
	want := []string{"c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("diff = %#v, want %#v", got, want)
	}
}
