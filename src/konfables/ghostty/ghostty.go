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

type Ghostty struct{}

func New() *Ghostty { return &Ghostty{} }

func (g *Ghostty) Info() konfables.AppInfo {
	return konfables.AppInfo{
		Name:       "ghostty",
		Binary:     "ghostty",
		ConfigPath: pkg.XDGConfigPath("ghostty", "config"),
		Format:     "ghostty",
		Icon:       "👻",
		NerdIcon:   "\uf47d", // 󰑽 ghost
	}
}

func (g *Ghostty) Name() string            { return "ghostty" }
func (g *Ghostty) ConfigPath() string       { return pkg.XDGConfigPath("ghostty", "config") }
func (g *Ghostty) Parser() konfables.Parser { return &parser{} }
func (g *Ghostty) Schema() ([]byte, error)  { return schemaData, nil }

// Version runs "ghostty --version" and returns the version string.
func (g *Ghostty) Version(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "ghostty", "--version").Output()
	if err != nil {
		return "", err
	}
	// output is typically a single line like "1.2.3" or multi-line; take first line
	line := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
	return line, nil
}
