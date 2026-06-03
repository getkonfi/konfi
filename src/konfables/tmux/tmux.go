package tmux

import (
	"context"
	_ "embed"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/eminert/konfi/konfables"
	"github.com/eminert/konfi/pkg"
)

//go:embed schema.yaml
var schemaData []byte

type Tmux struct {
	*pkg.FilePersister
}

func New(p *pkg.FilePersister) *Tmux {
	return &Tmux{FilePersister: p}
}

func (t *Tmux) Info() konfables.AppInfo {
	return konfables.AppInfo{
		Name:       "tmux",
		Binary:     "tmux",
		ConfigPath: t.Path,
		Format:     "tmux",
		Icon:       "",
		NerdIcon:   "\uebc8", // nf-md-dock_window
		ReloadCmd:  []string{"tmux", "source-file", t.Path},
	}
}

func (t *Tmux) Name() string             { return "tmux" }
func (t *Tmux) ConfigPath() string       { return t.Path }
func (t *Tmux) Parser() konfables.Parser { return newParser() }
func (t *Tmux) Schema() ([]byte, error)  { return schemaData, nil }

// Version runs "tmux -V" and parses "tmux X.Y".
func (t *Tmux) Version(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "tmux", "-V").Output()
	if err != nil {
		return "", err
	}
	line := strings.TrimSpace(string(out))
	fields := strings.Fields(line)
	if len(fields) >= 2 {
		return fields[1], nil
	}
	return line, nil
}

// DefaultConfigPath returns the tmux config path, preferring XDG.
func DefaultConfigPath() string {
	xdg := pkg.XDGConfigPath("tmux", "tmux.conf")
	if pkg.FileExists(xdg) {
		return xdg
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".tmux.conf")
}
