package claude

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

type Claude struct {
	*TieredPersister
}

func New(p *TieredPersister) *Claude {
	return &Claude{TieredPersister: p}
}

func (c *Claude) Info() konfables.AppInfo {
	return konfables.AppInfo{
		Name:       "claude",
		Binary:     "claude",
		ConfigPath: c.tiers[0].Path, // global tier
		Format:     "json",
		Icon:       "◈",
		NerdIcon:   "\uf2c0",
	}
}

func (c *Claude) Name() string             { return "claude" }
func (c *Claude) ConfigPath() string        { return c.tiers[0].Path }
func (c *Claude) Parser() konfables.Parser  { return &pkg.JSONParser{} }
func (c *Claude) Schema() ([]byte, error)   { return schemaData, nil }

// Version runs "claude --version" and returns the version string.
func (c *Claude) Version(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "claude", "--version").Output()
	if err != nil {
		return "", err
	}
	line := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
	return line, nil
}
