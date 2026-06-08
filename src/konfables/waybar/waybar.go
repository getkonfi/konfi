package waybar

import (
	"context"
	_ "embed"
	"os/exec"
	"strings"

	"github.com/eminert/konfi/konfables"
	"github.com/eminert/konfi/pkg"
)

//go:embed schema.yaml
var schemaData []byte

type Waybar struct {
	*pkg.FilePersister
}

func New(p *pkg.FilePersister) *Waybar {
	return &Waybar{FilePersister: p}
}

func (w *Waybar) Info() konfables.AppInfo {
	return konfables.AppInfo{
		Name:       "waybar",
		Binary:     "waybar",
		ConfigPath: w.Path,
		Format:     "json",
		Icon:       "W",
		NerdIcon:   "\uf0c9",
	}
}

func (w *Waybar) Name() string             { return "waybar" }
func (w *Waybar) ConfigPath() string       { return w.Path }
func (w *Waybar) Parser() konfables.Parser { return newParser() }
func (w *Waybar) Schema() ([]byte, error)  { return schemaData, nil }

func (w *Waybar) Version(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "waybar", "--version").Output()
	if err != nil {
		return "", err
	}
	line := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
	if v := pkg.ExtractSemver(line); v != "" {
		return v, nil
	}
	return line, nil
}
