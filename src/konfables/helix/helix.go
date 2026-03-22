package helix

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

type Helix struct {
	*pkg.FilePersister
}

func New(p *pkg.FilePersister) *Helix {
	return &Helix{FilePersister: p}
}

func (h *Helix) Info() konfables.AppInfo {
	return konfables.AppInfo{
		Name:       "helix",
		Binary:     "hx",
		ConfigPath: h.Path,
		Format:     "toml",
		Icon:       "🧬",
		NerdIcon:   "\uf121", // nf-fa-code
	}
}

func (h *Helix) Name() string             { return "helix" }
func (h *Helix) ConfigPath() string       { return h.Path }
func (h *Helix) Parser() konfables.Parser { return &parser{} }
func (h *Helix) Schema() ([]byte, error)  { return schemaData, nil }

// Version runs "hx --version" and parses "helix X.Y.Z (hash)".
func (h *Helix) Version(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "hx", "--version").Output()
	if err != nil {
		return "", err
	}
	line := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
	fields := strings.Fields(line)
	if len(fields) >= 2 {
		return fields[1], nil
	}
	return line, nil
}

// DefaultConfigPath returns the standard helix config.toml location.
func DefaultConfigPath() string {
	return pkg.XDGConfigPath("helix", "config.toml")
}
