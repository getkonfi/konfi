package ui

import (
	"strings"
	"testing"

	"github.com/eminert/konfi/pkg"
)

func TestRenderFieldValueBoolUsesTextOnly(t *testing.T) {
	c := &content{theme: testTheme()}
	f := pkg.Field{Type: "bool"}

	for _, tc := range []struct {
		name      string
		value     string
		isDefault bool
	}{
		{name: "default false", value: "false", isDefault: true},
		{name: "configured true", value: "true", isDefault: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := stripANSI(c.renderFieldValue(f, tc.value, tc.isDefault))
			if got != tc.value {
				t.Fatalf("renderFieldValue() = %q, want %q", got, tc.value)
			}
			if strings.ContainsAny(got, "●○") {
				t.Fatalf("bool field value should not render a status dot: %q", got)
			}
		})
	}
}

func TestRenderFieldValueColorUsesHashMarkerAndHex(t *testing.T) {
	c := &content{theme: testTheme()}
	f := pkg.Field{Type: "color"}

	got := stripANSI(c.renderFieldValue(f, "aabbcc", false))
	if got != "## #aabbcc" {
		t.Fatalf("renderFieldValue() = %q, want %q", got, "## #aabbcc")
	}
	if strings.Contains(got, "██") {
		t.Fatalf("color field value should not render block swatches: %q", got)
	}
}

func TestColorEditorSelectionDoesNotUseSquareBrackets(t *testing.T) {
	var e colorEditor
	_ = e.Init(pkg.Field{Palette: []string{"#112233"}}, "#112233", testTheme())

	got := stripANSI(e.View(80))
	firstLine := strings.Split(got, "\n")[0]
	if strings.ContainsAny(firstLine, "[]") {
		t.Fatalf("selected color cell should not use square brackets: %q", firstLine)
	}
	if !strings.Contains(firstLine, "#112233") {
		t.Fatalf("selected color cell should show the hex code: %q", firstLine)
	}
}
