package konfables

import (
	"context"

	"github.com/emin/konfigurator/pkg"
)

// AppInfo describes a konfable application.
type AppInfo struct {
	Name       string
	Binary     string
	ConfigPath string
	Format     string
	Icon       string
	NerdIcon   string // nerd font glyph for sidebar
}

// Parser performs surgical, line-preserving edits on config file bytes.
type Parser interface {
	FindValue(data []byte, key string) (string, bool)
	FindLine(data []byte, key string) (int, bool)
	SetValue(data []byte, key, value string) ([]byte, error)
	DeleteKey(data []byte, key string) ([]byte, error)
	ListKeys(data []byte) []string
}

// Konfable represents a configurable application.
type Konfable interface {
	Info() AppInfo
	Parser() Parser
	Schema() ([]byte, error)
	Name() string
	ConfigPath() string
	pkg.Persister // embeds Load + Save
}

// MultiValueParser is an optional interface for parsers that support
// multi-value keys (e.g., ghostty keybind, palette).
type MultiValueParser interface {
	FindValues(data []byte, key string) ([]string, bool)
	SetValues(data []byte, key string, values []string) ([]byte, error)
}

// BatchParser can return all key-value pairs in a single pass.
type BatchParser interface {
	FindAll(data []byte) map[string]string
}

// BatchMultiParser extends BatchParser with multi-value batch support.
type BatchMultiParser interface {
	FindAllMulti(data []byte) (singles map[string]string, multi map[string][]string)
}

// Versioned is an optional interface for apps that can report their version.
type Versioned interface {
	Version(ctx context.Context) (string, error)
}
