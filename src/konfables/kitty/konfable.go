package kitty

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

type Kitty struct {
	*pkg.FilePersister
}

func New(p *pkg.FilePersister) *Kitty {
	return &Kitty{FilePersister: p}
}

func (k *Kitty) Info() konfables.AppInfo {
	return konfables.AppInfo{
		Name:       "kitty",
		Binary:     "kitty",
		ConfigPath: k.Path,
		Format:     "kitty",
		Icon:       "",
		NerdIcon:   "\uf490", // nerd font terminal icon
	}
}

func (k *Kitty) Name() string            { return "kitty" }
func (k *Kitty) ConfigPath() string       { return k.Path }
func (k *Kitty) Parser() konfables.Parser { return newParser() }
func (k *Kitty) Schema() ([]byte, error)  { return schemaData, nil }

// Version runs "kitty --version" and returns the version string.
func (k *Kitty) Version(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "kitty", "--version").Output()
	if err != nil {
		return "", err
	}
	line := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
	return line, nil
}
