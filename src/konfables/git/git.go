package git

import (
	"context"
	_ "embed"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/emin/konfigurator/konfables"
	"github.com/emin/konfigurator/pkg"
)

//go:embed schema.yaml
var schemaData []byte

type Git struct {
	*pkg.FilePersister
}

func New(p *pkg.FilePersister) *Git {
	return &Git{FilePersister: p}
}

func (g *Git) Info() konfables.AppInfo {
	return konfables.AppInfo{
		Name:       "git",
		Binary:     "git",
		ConfigPath: g.Path,
		Format:     "ini",
		Icon:       "",
		NerdIcon:   "\ue702", // nf-dev-git
	}
}

func (g *Git) Name() string             { return "git" }
func (g *Git) ConfigPath() string       { return g.Path }
func (g *Git) Parser() konfables.Parser { return &parser{} }
func (g *Git) Schema() ([]byte, error)  { return schemaData, nil }

// Version runs "git --version" and parses "git version X.Y.Z".
func (g *Git) Version(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "git", "--version").Output()
	if err != nil {
		return "", err
	}
	line := strings.TrimSpace(string(out))
	fields := strings.Fields(line)
	if len(fields) >= 3 {
		return fields[2], nil
	}
	return line, nil
}

// DefaultConfigPath returns the user-level git config path.
func DefaultConfigPath() string {
	// XDG first
	xdg := pkg.XDGConfigPath("git", "config")
	if pkg.FileExists(xdg) {
		return xdg
	}
	// fall back to ~/.gitconfig
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".gitconfig")
}
