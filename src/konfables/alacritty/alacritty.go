package alacritty

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

type Alacritty struct{}

func New() *Alacritty { return &Alacritty{} }

func (a *Alacritty) Info() konfables.AppInfo {
	return konfables.AppInfo{
		Name:       "alacritty",
		Binary:     "alacritty",
		ConfigPath: pkg.XDGConfigPath("alacritty", "alacritty.toml"),
		Format:     "toml",
		Icon:       "🖥",
		NerdIcon:   "\ue795", //  terminal
	}
}

func (a *Alacritty) Name() string             { return "alacritty" }
func (a *Alacritty) ConfigPath() string       { return pkg.XDGConfigPath("alacritty", "alacritty.toml") }
func (a *Alacritty) Parser() konfables.Parser { return &parser{} }
func (a *Alacritty) Schema() ([]byte, error)  { return schemaData, nil }

// Version runs "alacritty --version" and parses "alacritty X.Y.Z".
func (a *Alacritty) Version(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "alacritty", "--version").Output()
	if err != nil {
		return "", err
	}
	// output: "alacritty 0.15.1 (abcdef12)" — extract version token
	line := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
	fields := strings.Fields(line)
	if len(fields) >= 2 {
		return fields[1], nil
	}
	return line, nil
}
