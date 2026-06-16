package ssh

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

type SSH struct {
	*pkg.FilePersister
	palette []pkg.Field
}

func New(p *pkg.FilePersister) *SSH {
	return &SSH{FilePersister: p, palette: buildPalette(schemaData)}
}

// buildPalette flattens the schema's flat directive fields into a block-editor
// palette, excluding the synthetic Blocks field. a parse failure yields an empty
// palette.
func buildPalette(data []byte) []pkg.Field {
	s, err := pkg.LoadSchema(data)
	if err != nil {
		return nil
	}
	var fields []pkg.Field
	for si := range s.Sections {
		for fi := range s.Sections[si].Fields {
			f := s.Sections[si].Fields[fi]
			if isBlocksKey(f.Key) {
				continue
			}
			fields = append(fields, f)
		}
	}
	return fields
}

func (s *SSH) Info() konfables.AppInfo {
	return konfables.AppInfo{
		Name:       "ssh",
		Binary:     "ssh",
		ConfigPath: s.Path,
		Format:     "ssh",
		Icon:       "\U0001F511",
		NerdIcon:   "\uf084", // nf-fa-key
	}
}

func (s *SSH) Name() string       { return "ssh" }
func (s *SSH) ConfigPath() string { return s.Path }

func (s *SSH) Parser() konfables.Parser {
	return &parser{
		palette: s.palette,
		openers: []string{"Host", "Match"},
		isNamed: isNamedBlock,
		place:   pkg.PlacementRule{IsLowPrecedence: isLowPrecedenceBlock},
	}
}

func (s *SSH) Schema() ([]byte, error) { return schemaData, nil }

// Version runs "ssh -V" and extracts the version string from stderr.
func (s *SSH) Version(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "ssh", "-V")
	// ssh -V writes to stderr
	out, err := cmd.CombinedOutput()
	if err != nil && len(out) == 0 {
		return "", err
	}
	line := strings.TrimSpace(string(out))
	// "OpenSSH_9.9p1, OpenSSL ..." → "OpenSSH_9.9p1"
	if idx := strings.IndexByte(line, ','); idx > 0 {
		line = line[:idx]
	}
	return line, nil
}

// DefaultConfigPath returns ~/.ssh/config.
func DefaultConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ssh", "config")
}
