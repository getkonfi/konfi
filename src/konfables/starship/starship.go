package starship

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

type Starship struct {
	*pkg.FilePersister
}

func New(p *pkg.FilePersister) *Starship {
	return &Starship{FilePersister: p}
}

func (s *Starship) Info() konfables.AppInfo {
	return konfables.AppInfo{
		Name:       "starship",
		Binary:     "starship",
		ConfigPath: s.Path,
		Format:     "toml",
		Icon:       "🚀",
		NerdIcon:   "\uf197", //  rocket
	}
}

func (s *Starship) Name() string             { return "starship" }
func (s *Starship) ConfigPath() string       { return s.Path }
func (s *Starship) Parser() konfables.Parser { return newParser() }
func (s *Starship) Schema() ([]byte, error)  { return schemaData, nil }

// Version runs "starship --version" and parses "starship X.Y.Z".
func (s *Starship) Version(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "starship", "--version").Output()
	if err != nil {
		return "", err
	}
	line := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
	if _, ver, ok := strings.Cut(line, " "); ok {
		return strings.TrimSpace(ver), nil
	}
	return line, nil
}

// DefaultConfigPath returns the standard starship.toml location.
func DefaultConfigPath() string {
	if path := strings.TrimSpace(os.Getenv("STARSHIP_CONFIG")); path != "" {
		return path
	}
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "starship.toml")
}
