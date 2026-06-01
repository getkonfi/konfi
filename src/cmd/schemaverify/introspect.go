package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/emin/konfigurator/pkg"
)

const dumpTimeout = 30 * time.Second
const observedConfigCategory = "observed_config"

// Introspector extracts known config keys from an app.
type Introspector interface {
	Name() string
	Available() bool
	DumpKeys(ctx context.Context) ([]string, error)
}

// introspectorFor returns the introspector for a given app, or nil.
func introspectorFor(app string) Introspector {
	switch app {
	case "ghostty":
		return &execIntrospector{
			app:    "ghostty",
			binary: "ghostty",
			args:   []string{"+show-config", "--default", "--docs"},
			parser: parseGhosttyDump,
		}
	case "kitty":
		return &execIntrospector{
			app:    "kitty",
			binary: "kitty",
			args:   []string{"--debug-config"},
			parser: parseKittyDump,
		}
	case "tmux":
		return &execIntrospector{
			app:    "tmux",
			binary: "tmux",
			args:   []string{"show-options", "-g"},
			parser: parseTmuxDump,
		}
	case "git":
		return &execIntrospector{
			app:    "git",
			binary: "git",
			args:   []string{"config", "--list"},
			parser: parseGitDump,
		}
	case "gnome":
		return &multiExecIntrospector{
			app:    "gnome",
			binary: "gsettings",
			commands: [][]string{
				{"list-recursively", "org.gnome.desktop.interface"},
				{"list-recursively", "org.gnome.desktop.background"},
			},
			parser: parseGnomeDump,
		}
	case "dconf":
		return &execIntrospector{
			app:    "dconf",
			binary: "dconf",
			args:   []string{"dump", "/org/gnome/desktop/"},
			parser: parseDconfDump,
		}
	case "pacman":
		return &execIntrospector{
			app:    "pacman",
			binary: "pacman-conf",
			args:   nil,
			parser: parsePacmanDump,
		}
	case "ssh":
		return &staticIntrospector{
			app: "ssh",
			keys: []string{
				"Hostname", "Port", "User",
				"ServerAliveInterval", "ServerAliveCountMax",
				"ConnectTimeout", "ConnectionAttempts", "TCPKeepAlive",
				"ProxyJump", "ProxyCommand", "RequestTTY", "BatchMode",
				"PubkeyAuthentication", "PasswordAuthentication",
				"PreferredAuthentications", "IdentitiesOnly",
				"AddKeysToAgent", "IdentityFile", "IdentityAgent", "CertificateFile",
				"StrictHostKeyChecking", "VerifyHostKeyDNS", "HashKnownHosts",
				"UpdateHostKeys", "Ciphers", "KexAlgorithms", "MACs",
				"ForwardAgent", "ForwardX11", "ForwardX11Trusted",
				"LocalForward", "RemoteForward", "DynamicForward",
				"Compression", "LogLevel", "VisualHostKey",
			},
		}
	default:
		return nil
	}
}

// checkDump runs introspection for a single app and compares against its schema.
func checkDump(ctx context.Context, schema *pkg.Schema) []Finding {
	intr := introspectorFor(schema.App)
	if intr == nil {
		return []Finding{{Info, observedConfigCategory, fmt.Sprintf("no introspector for %s", schema.App)}}
	}

	if !intr.Available() {
		return []Finding{{Info, observedConfigCategory, fmt.Sprintf("%s not available in PATH", schema.App)}}
	}

	// probe version for version-filtered comparison
	version := probeVersion(schema.App)
	filtered := schema
	if version != "" {
		filtered = schema.FilterByVersion(version)
	}

	dctx, cancel := context.WithTimeout(ctx, dumpTimeout)
	defer cancel()

	dumpKeys, err := intr.DumpKeys(dctx)
	if err != nil {
		return []Finding{{Warn, observedConfigCategory, fmt.Sprintf("%s config introspection failed: %v", schema.App, err)}}
	}

	schemaKeys := filtered.SchemaKeys()
	dumpSet := make(map[string]struct{}, len(dumpKeys))
	for _, k := range dumpKeys {
		dumpSet[k] = struct{}{}
	}

	isPartial := schema.Coverage != "full" && schema.Coverage != ""

	var findings []Finding

	// missing: observed in local config but not in schema
	for _, k := range dumpKeys {
		if _, ok := schemaKeys[k]; !ok {
			sev := Warn
			if isPartial {
				sev = Info
			}
			findings = append(findings, Finding{sev, observedConfigCategory,
				fmt.Sprintf("%s: key %q in observed config but not schema", schema.App, k)})
		}
	}

	// extra: in schema but not observed locally
	for k := range schemaKeys {
		if _, ok := dumpSet[k]; !ok {
			sev := Warn
			if isPartial {
				sev = Info
			}
			findings = append(findings, Finding{sev, observedConfigCategory,
				fmt.Sprintf("%s: key %q in schema but not observed config", schema.App, k)})
		}
	}

	if len(findings) == 0 {
		findings = append(findings, Finding{Pass, observedConfigCategory,
			fmt.Sprintf("%s: %d keys match", schema.App, len(schemaKeys))})
	}

	return findings
}

// probeVersion tries to detect the app version via common flags.
func probeVersion(app string) string {
	var cmd *exec.Cmd
	switch app {
	case "ghostty":
		cmd = exec.Command("ghostty", "--version")
	case "kitty":
		cmd = exec.Command("kitty", "--version")
	case "tmux":
		cmd = exec.Command("tmux", "-V")
	case "git":
		cmd = exec.Command("git", "--version")
	case "pacman":
		cmd = exec.Command("pacman", "--version")
	default:
		return ""
	}

	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return extractVersion(string(out))
}

// extractVersion pulls a semver-like version from command output.
func extractVersion(output string) string {
	for word := range strings.FieldsSeq(output) {
		word = strings.TrimRight(word, ",;)")
		if v := pkg.NormalizeSemver(word); v != "" {
			return word
		}
	}
	return ""
}

// execIntrospector runs a single command and parses its output.
type execIntrospector struct {
	app    string
	binary string
	args   []string
	parser func([]byte) []string
}

func (e *execIntrospector) Name() string { return e.app }

func (e *execIntrospector) Available() bool {
	_, err := exec.LookPath(e.binary)
	return err == nil
}

func (e *execIntrospector) DumpKeys(ctx context.Context) ([]string, error) {
	cmd := exec.CommandContext(ctx, e.binary, e.args...)
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("exec %s: %w", e.binary, err)
	}
	return e.parser(out), nil
}

// multiExecIntrospector runs multiple commands and merges results.
type multiExecIntrospector struct {
	app      string
	binary   string
	commands [][]string
	parser   func([]byte) []string
}

func (m *multiExecIntrospector) Name() string { return m.app }

func (m *multiExecIntrospector) Available() bool {
	_, err := exec.LookPath(m.binary)
	return err == nil
}

func (m *multiExecIntrospector) DumpKeys(ctx context.Context) ([]string, error) {
	var all []byte
	for _, args := range m.commands {
		cmd := exec.CommandContext(ctx, m.binary, args...)
		cmd.Env = append(os.Environ(), "LC_ALL=C")
		out, err := cmd.Output()
		if err != nil {
			continue // best effort — some schemas may not exist on all systems
		}
		all = append(all, out...)
	}
	return m.parser(all), nil
}

// staticIntrospector returns a hardcoded key list.
type staticIntrospector struct {
	app  string
	keys []string
}

func (s *staticIntrospector) Name() string      { return s.app }
func (s *staticIntrospector) Available() bool    { return true }
func (s *staticIntrospector) DumpKeys(_ context.Context) ([]string, error) {
	return s.keys, nil
}

// --- parsers ---

// parseGhosttyDump parses `ghostty +show-config --default --docs`.
// format: `key = value` lines, `#` lines are comments/docs.
func parseGhosttyDump(data []byte) []string {
	var keys []string
	seen := make(map[string]bool)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if idx := strings.Index(line, "="); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			if key != "" && !seen[key] {
				seen[key] = true
				keys = append(keys, key)
			}
		}
	}
	return keys
}

// parseKittyDump parses `kitty --debug-config`.
// format: `key value` lines (space-separated), `#` lines are comments.
func parseKittyDump(data []byte) []string {
	var keys []string
	seen := make(map[string]bool)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			key := parts[0]
			if !seen[key] {
				seen[key] = true
				keys = append(keys, key)
			}
		}
	}
	return keys
}

// parseTmuxDump parses `tmux show-options -g`.
// format: `option value` lines.
func parseTmuxDump(data []byte) []string {
	var keys []string
	seen := make(map[string]bool)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			key := parts[0]
			if !seen[key] {
				seen[key] = true
				keys = append(keys, key)
			}
		}
	}
	return keys
}

// parseGitDump parses `git config --list`.
// format: `section.key=value` lines.
func parseGitDump(data []byte) []string {
	var keys []string
	seen := make(map[string]bool)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if idx := strings.Index(line, "="); idx > 0 {
			key := line[:idx]
			if !seen[key] {
				seen[key] = true
				keys = append(keys, key)
			}
		}
	}
	return keys
}

// parseGnomeDump parses `gsettings list-recursively <schema>`.
// format: `org.gnome.desktop.interface key value` → key becomes `schema/key`.
func parseGnomeDump(data []byte) []string {
	var keys []string
	seen := make(map[string]bool)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			key := parts[0] + "/" + parts[1]
			if !seen[key] {
				seen[key] = true
				keys = append(keys, key)
			}
		}
	}
	return keys
}

// parseDconfDump parses `dconf dump /path/`.
// INI format: [section] headers + key=value lines → full dconf paths.
func parseDconfDump(data []byte) []string {
	var keys []string
	seen := make(map[string]bool)
	section := ""
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = "/org/gnome/desktop/" + line[1:len(line)-1]
			if !strings.HasSuffix(section, "/") {
				section += "/"
			}
			continue
		}
		if idx := strings.Index(line, "="); idx > 0 {
			key := section + strings.TrimSpace(line[:idx])
			if !seen[key] {
				seen[key] = true
				keys = append(keys, key)
			}
		}
	}
	return keys
}

// parsePacmanDump parses `pacman-conf` output.
// format: mixed — section headers in [brackets], key = value or bare key lines.
func parsePacmanDump(data []byte) []string {
	var keys []string
	seen := make(map[string]bool)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			continue
		}
		key := line
		if idx := strings.Index(line, "="); idx > 0 {
			key = strings.TrimSpace(line[:idx])
		}
		if !seen[key] {
			seen[key] = true
			keys = append(keys, key)
		}
	}
	return keys
}
