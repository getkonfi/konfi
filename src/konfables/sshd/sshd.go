package sshd

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

type SSHD struct {
	*pkg.FilePersister
	palette []pkg.Field
}

func New(p *pkg.FilePersister) *SSHD {
	return &SSHD{FilePersister: p, palette: buildPalette(schemaData)}
}

func buildPalette(data []byte) []pkg.Field {
	s, err := pkg.LoadSchema(data)
	if err != nil {
		return nil
	}
	var fields []pkg.Field
	for si := range s.Sections {
		for fi := range s.Sections[si].Fields {
			f := s.Sections[si].Fields[fi]
			if isBlocksKey(f.Key) || !isMatchAllowed(f.Key) {
				continue
			}
			fields = append(fields, f)
		}
	}
	return fields
}

func (s *SSHD) Info() konfables.AppInfo {
	return konfables.AppInfo{
		Name:       "sshd",
		Binary:     "sshd",
		ConfigPath: s.Path,
		Format:     "sshd",
		Icon:       "\U0001F5A5",
		NerdIcon:   "\uf233", // nf-fa-server
	}
}

func (s *SSHD) Name() string       { return "sshd" }
func (s *SSHD) ConfigPath() string { return s.Path }

func (s *SSHD) Parser() konfables.Parser {
	return &parser{palette: s.palette}
}

func (s *SSHD) Schema() ([]byte, error) { return schemaData, nil }

func (s *SSHD) Version(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, BinaryPath(), "-V")
	out, err := cmd.CombinedOutput()
	if err != nil && len(out) == 0 {
		return "", err
	}
	line := strings.TrimSpace(string(out))
	if idx := strings.IndexByte(line, ','); idx > 0 {
		line = line[:idx]
	}
	return line, nil
}

func DefaultConfigPath() string {
	return filepath.Join(string(os.PathSeparator), "etc", "ssh", "sshd_config")
}

func BinaryPath() string {
	if p, err := exec.LookPath("sshd"); err == nil {
		return p
	}
	for _, p := range []string{"/usr/sbin/sshd", "/usr/local/sbin/sshd", "/opt/homebrew/sbin/sshd"} {
		if info, err := os.Stat(p); err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
			return p
		}
	}
	return "sshd"
}
