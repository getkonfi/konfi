package kitty

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
		Icon:       "🐱",
		NerdIcon:   "\U000f011b", // nf-md-cat
		ReloadCmd:  []string{"kitty", "@", "set-colors", "--all", "--configured"},
	}
}

func (k *Kitty) Name() string             { return "kitty" }
func (k *Kitty) ConfigPath() string       { return k.Path }
func (k *Kitty) Parser() konfables.Parser { return newParser() }
func (k *Kitty) Schema() ([]byte, error)  { return schemaData, nil }

// Version runs "kitty --version" and returns the bare semver string.
// "kitty --version" prints e.g. "kitty 0.39.1 created by Kovid Goyal" — we
// strip the banner so schema-level since/until gating can compare against
// a valid semver.
func (k *Kitty) Version(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "kitty", "--version").Output()
	if err != nil {
		return "", err
	}
	line := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
	if v := pkg.ExtractSemver(line); v != "" {
		return v, nil
	}
	return line, nil
}
