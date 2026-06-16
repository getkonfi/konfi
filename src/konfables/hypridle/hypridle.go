package hypridle

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

type Hypridle struct {
	*pkg.FilePersister
}

func New(p *pkg.FilePersister) *Hypridle {
	return &Hypridle{FilePersister: p}
}

func DefaultConfigPath() string {
	return pkg.XDGConfigPath("hypr", "hypridle.conf")
}

func (h *Hypridle) Info() konfables.AppInfo {
	return konfables.AppInfo{
		Name:       "hypridle",
		Binary:     "hypridle",
		ConfigPath: h.Path,
		Format:     "hyprland",
		Icon:       "Z",
		NerdIcon:   "\uf186",
	}
}

func (h *Hypridle) Name() string             { return "hypridle" }
func (h *Hypridle) ConfigPath() string       { return h.Path }
func (h *Hypridle) Parser() konfables.Parser { return newParser() }
func (h *Hypridle) Schema() ([]byte, error)  { return schemaData, nil }

func (h *Hypridle) Version(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "hypridle", "--version").Output()
	if err != nil {
		return "", err
	}
	line := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
	if v := pkg.ExtractSemver(line); v != "" {
		return v, nil
	}
	return line, nil
}
