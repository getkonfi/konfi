package pacman

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

type Pacman struct {
	*pkg.FilePersister
}

func New(p *pkg.FilePersister) *Pacman {
	return &Pacman{FilePersister: p}
}

func (p *Pacman) Info() konfables.AppInfo {
	return konfables.AppInfo{
		Name:       "pacman",
		Binary:     "pacman",
		ConfigPath: p.Path,
		Format:     "ini",
		Icon:       "",
		NerdIcon:   "\uf303", // nf-linux-archlinux
	}
}

func (p *Pacman) Name() string             { return "pacman" }
func (p *Pacman) ConfigPath() string       { return p.Path }
func (p *Pacman) Parser() konfables.Parser { return newParser() }
func (p *Pacman) Schema() ([]byte, error)  { return schemaData, nil }

// Version runs "pacman --version" and parses "Pacman vX.Y.Z".
func (p *Pacman) Version(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "pacman", "--version").Output()
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if idx := strings.Index(line, "Pacman v"); idx >= 0 {
			ver := line[idx+len("Pacman v"):]
			if sp := strings.IndexByte(ver, ' '); sp > 0 {
				ver = ver[:sp]
			}
			return ver, nil
		}
	}
	return strings.TrimSpace(string(out)), nil
}
