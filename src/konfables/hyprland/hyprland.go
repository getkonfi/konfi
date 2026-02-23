package hyprland

import (
	"context"
	_ "embed"
	"os/exec"
	"strings"

	"github.com/emin/konfigurator/konfables"
	"github.com/emin/konfigurator/pkg"
)

//go:embed schema.yaml
var schemaData []byte

type Hyprland struct{}

func New() *Hyprland { return &Hyprland{} }

func (h *Hyprland) Info() konfables.AppInfo {
	return konfables.AppInfo{
		Name:       "hyprland",
		Binary:     "Hyprland",
		ConfigPath: pkg.XDGConfigPath("hypr", "hyprland.conf"),
		Format:     "hyprland",
		Icon:       "🪟",
		NerdIcon:   "\uf219", //  window
	}
}

func (h *Hyprland) Name() string             { return "hyprland" }
func (h *Hyprland) ConfigPath() string       { return pkg.XDGConfigPath("hypr", "hyprland.conf") }
func (h *Hyprland) Parser() konfables.Parser { return newParser() }
func (h *Hyprland) Schema() ([]byte, error)  { return schemaData, nil }

// Version runs "hyprctl version" and extracts the version tag.
func (h *Hyprland) Version(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "hyprctl", "version").Output()
	if err != nil {
		return "", err
	}
	// look for "Tag: v0.45.0" line
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(line, "Tag:"); ok {
			tag := strings.TrimSpace(after)
			// tag may contain extra info after comma: "v0.45.0, ..."
			if idx := strings.Index(tag, ","); idx >= 0 {
				tag = tag[:idx]
			}
			return strings.TrimSpace(tag), nil
		}
	}
	// fallback: first line
	return strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0]), nil
}
