package fuzzel

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

type Fuzzel struct {
	*pkg.FilePersister
}

func New(p *pkg.FilePersister) *Fuzzel {
	return &Fuzzel{FilePersister: p}
}

func (f *Fuzzel) Info() konfables.AppInfo {
	return konfables.AppInfo{
		Name:       "fuzzel",
		Binary:     "fuzzel",
		ConfigPath: f.Path,
		Format:     "ini",
		Icon:       "🔎",
		NerdIcon:   "\uf002",
	}
}

func (f *Fuzzel) Name() string             { return "fuzzel" }
func (f *Fuzzel) ConfigPath() string       { return f.Path }
func (f *Fuzzel) Parser() konfables.Parser { return newParser() }
func (f *Fuzzel) Schema() ([]byte, error)  { return schemaData, nil }

func (f *Fuzzel) Version(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "fuzzel", "--version").Output()
	if err != nil {
		return "", err
	}
	line := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
	if v := pkg.ExtractSemver(line); v != "" {
		return v, nil
	}
	return line, nil
}
