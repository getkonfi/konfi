package yazi

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

type Yazi struct {
	*pkg.FilePersister
}

func New(p *pkg.FilePersister) *Yazi {
	return &Yazi{FilePersister: p}
}

func (y *Yazi) Info() konfables.AppInfo {
	return konfables.AppInfo{
		Name:       "yazi",
		Binary:     "yazi",
		ConfigPath: y.Path,
		Format:     "toml",
		Icon:       "📁",
		NerdIcon:   "\uf07b", // nf-fa-folder
	}
}

func (y *Yazi) Name() string             { return "yazi" }
func (y *Yazi) ConfigPath() string       { return y.Path }
func (y *Yazi) Parser() konfables.Parser { return newParser() }
func (y *Yazi) Schema() ([]byte, error)  { return schemaData, nil }

func (y *Yazi) Version(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "yazi", "--version").Output()
	if err != nil {
		return "", err
	}
	line := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
	if v := pkg.ExtractSemver(line); v != "" {
		return v, nil
	}
	return line, nil
}
