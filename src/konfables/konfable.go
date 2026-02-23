package konfables

import "context"

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
}

// Konfable represents a configurable application.
// implicitly satisfies setup.Konfable (Name + ConfigPath).
type Konfable interface {
	Info() AppInfo
	Parser() Parser
	Schema() ([]byte, error)
	Name() string
	ConfigPath() string
}

// Versioned is an optional interface for apps that can report their version.
type Versioned interface {
	Version(ctx context.Context) (string, error)
}
