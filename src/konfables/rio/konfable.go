package rio

import (
	"context"
	_ "embed"
	"os/exec"
	"strings"

	"github.com/eminert/konfi/konfables"
	"github.com/eminert/konfi/pkg"
)

//go:embed schema.yaml
var schemaData []byte

type Rio struct {
	*pkg.FilePersister
}

func New(p *pkg.FilePersister) *Rio {
	return &Rio{FilePersister: p}
}

func (r *Rio) Info() konfables.AppInfo {
	return konfables.AppInfo{
		Name:       "rio",
		Binary:     "rio",
		ConfigPath: r.Path,
		Format:     "toml",
		Icon:       "R",
		NerdIcon:   "\ue795", // terminal
		AutoReload: true,
	}
}

func (r *Rio) Name() string             { return "rio" }
func (r *Rio) ConfigPath() string       { return r.Path }
func (r *Rio) Parser() konfables.Parser { return newParser() }
func (r *Rio) Schema() ([]byte, error)  { return schemaData, nil }

// Version runs "rio --version" and parses "rio X.Y.Z".
func (r *Rio) Version(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "rio", "--version").Output()
	if err != nil {
		return "", err
	}
	line := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
	fields := strings.Fields(line)
	if len(fields) >= 2 {
		return fields[1], nil
	}
	return line, nil
}
