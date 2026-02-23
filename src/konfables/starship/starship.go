package starship

import (
	"context"
	_ "embed"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/emin/konfigurator/konfables"
)

//go:embed schema.yaml
var schemaData []byte

type Starship struct{}

func New() *Starship { return &Starship{} }

func (s *Starship) Info() konfables.AppInfo {
	return konfables.AppInfo{
		Name:       "starship",
		Binary:     "starship",
		ConfigPath: configPath(),
		Format:     "toml",
		Icon:       "🚀",
		NerdIcon:   "\uf197", //  rocket
	}
}

func (s *Starship) Name() string             { return "starship" }
func (s *Starship) ConfigPath() string       { return configPath() }
func (s *Starship) Parser() konfables.Parser { return &parser{} }
func (s *Starship) Schema() ([]byte, error)  { return schemaData, nil }

// Version runs "starship --version" and parses "starship X.Y.Z".
func (s *Starship) Version(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "starship", "--version").Output()
	if err != nil {
		return "", err
	}
	// output: "starship 1.21.1\n..." — extract version token
	line := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
	if _, ver, ok := strings.Cut(line, " "); ok {
		return strings.TrimSpace(ver), nil
	}
	return line, nil
}

// starship.toml lives at ~/.config/starship.toml (not in a subdirectory)
func configPath() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "starship.toml")
}
