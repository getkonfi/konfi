package brew

import (
	"context"
	_ "embed"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/getkonfi/konfi/konfables"
	"github.com/getkonfi/konfi/pkg"
)

//go:embed schema.yaml
var schemaData []byte

type Brew struct {
	*pkg.FilePersister
}

func New(p *pkg.FilePersister) *Brew {
	return &Brew{FilePersister: p}
}

func (b *Brew) Info() konfables.AppInfo {
	return konfables.AppInfo{
		Name:       "brew",
		Binary:     "brew",
		ConfigPath: b.Path,
		Format:     "brewfile",
		Icon:       "🍺",
		NerdIcon:   "", // nf-fa-beer
	}
}

func (b *Brew) Name() string             { return "brew" }
func (b *Brew) ConfigPath() string       { return b.Path }
func (b *Brew) Parser() konfables.Parser { return &parser{} }
func (b *Brew) Schema() ([]byte, error)  { return schemaData, nil }

// Version runs "brew --version" and returns the first line, e.g. "Homebrew 4.2.0".
func (b *Brew) Version(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "brew", "--version").Output()
	if err != nil {
		return "", err
	}
	line := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
	if v := pkg.ExtractSemver(line); v != "" {
		return v, nil
	}
	return line, nil
}

// DefaultConfigPath resolves the global Brewfile used by `brew bundle --global`.
func DefaultConfigPath() string {
	if env := strings.TrimSpace(os.Getenv("HOMEBREW_BUNDLE_FILE_GLOBAL")); env != "" {
		return env
	}
	home, _ := os.UserHomeDir()
	if xdg := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); xdg != "" {
		return filepath.Join(xdg, "homebrew", "Brewfile")
	}
	homebrew := filepath.Join(home, ".homebrew", "Brewfile")
	if pkg.FileExists(homebrew) {
		return homebrew
	}
	return filepath.Join(home, ".Brewfile")
}
