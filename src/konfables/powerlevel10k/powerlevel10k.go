package powerlevel10k

import (
	_ "embed"
	"os"
	"path/filepath"
	"strings"

	"github.com/getkonfi/konfi/konfables"
	"github.com/getkonfi/konfi/pkg"
)

//go:embed schema.yaml
var schemaData []byte

type Powerlevel10k struct {
	*pkg.FilePersister
}

func New(p *pkg.FilePersister) *Powerlevel10k {
	return &Powerlevel10k{FilePersister: p}
}

func (p *Powerlevel10k) Info() konfables.AppInfo {
	return konfables.AppInfo{
		Name:       "powerlevel10k",
		Binary:     "zsh",
		ConfigPath: p.Path,
		Format:     "zsh",
		Icon:       "⚡",
		NerdIcon:   "\uf0e7", // nf-fa-bolt
	}
}

func (p *Powerlevel10k) Name() string             { return "powerlevel10k" }
func (p *Powerlevel10k) ConfigPath() string       { return p.Path }
func (p *Powerlevel10k) Parser() konfables.Parser { return newParser() }
func (p *Powerlevel10k) Schema() ([]byte, error)  { return schemaData, nil }

// DefaultConfigPath returns the p10k configuration file used by the wizard.
func DefaultConfigPath() string {
	if path := strings.TrimSpace(os.Getenv("POWERLEVEL9K_CONFIG_FILE")); path != "" {
		return path
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".p10k.zsh")
}
