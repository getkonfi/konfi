package ghostty

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

type Ghostty struct {
	*pkg.FilePersister
}

func New(p *pkg.FilePersister) *Ghostty {
	return &Ghostty{FilePersister: p}
}

func (g *Ghostty) Info() konfables.AppInfo {
	return konfables.AppInfo{
		Name:       "ghostty",
		Binary:     "ghostty",
		ConfigPath: g.Path,
		Format:     "ghostty",
		Icon:       "👻",
		NerdIcon:   "\uf47d", // 󰑽 ghost
		AutoReload: true,
	}
}

func (g *Ghostty) Name() string            { return "ghostty" }
func (g *Ghostty) ConfigPath() string       { return g.Path }
func (g *Ghostty) Parser() konfables.Parser { return newParser() }
func (g *Ghostty) Schema() ([]byte, error)  { return schemaData, nil }

// Version runs "ghostty --version" and returns the bare semver string.
// "ghostty --version" prints e.g. "Ghostty 1.1.3" — we strip the banner so
// schema-level since/until gating can compare against a valid semver.
func (g *Ghostty) Version(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "ghostty", "--version").Output()
	if err != nil {
		return "", err
	}
	line := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
	if v := pkg.ExtractSemver(line); v != "" {
		return v, nil
	}
	return line, nil
}
