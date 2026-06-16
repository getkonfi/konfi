package hyprland

import (
	"context"
	_ "embed"
	"os/exec"
	"strings"

	"github.com/getkonfi/konfi/konfables"
	"github.com/getkonfi/konfi/pkg"
)

//go:embed schema.yaml
var schemaData []byte

type Hyprland struct {
	*pkg.FilePersister
}

func New(p *pkg.FilePersister) *Hyprland {
	return &Hyprland{FilePersister: p}
}

func (h *Hyprland) Info() konfables.AppInfo {
	return konfables.AppInfo{
		Name:       "hyprland",
		Binary:     "Hyprland",
		ConfigPath: h.Path,
		Format:     "hyprland",
		Icon:       "💧",
		NerdIcon:   "\U000f058c", // nf-md-water
		ReloadCmd:  []string{"hyprctl", "reload"},
	}
}

func (h *Hyprland) Name() string             { return "hyprland" }
func (h *Hyprland) ConfigPath() string       { return h.Path }
func (h *Hyprland) Parser() konfables.Parser { return newParser() }
func (h *Hyprland) Schema() ([]byte, error)  { return schemaData, nil }

// Version runs "hyprctl version" and extracts the version tag.
func (h *Hyprland) Version(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "hyprctl", "version").Output()
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(line, "Tag:"); ok {
			tag := strings.TrimSpace(after)
			if idx := strings.Index(tag, ","); idx >= 0 {
				tag = tag[:idx]
			}
			return strings.TrimSpace(tag), nil
		}
	}
	return strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0]), nil
}
