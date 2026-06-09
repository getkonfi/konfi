package ssh

import (
	"strings"

	"github.com/eminert/konfi/pkg"
)

const blocksKey = "Blocks"

var repeatableSSHKeys = map[string]bool{
	"identityfile": true,
}

var canonicalSSHKeys = map[string]string{
	"addkeystoagent":           "AddKeysToAgent",
	"batchmode":                "BatchMode",
	"certificatefile":          "CertificateFile",
	"ciphers":                  "Ciphers",
	"compression":              "Compression",
	"connectionattempts":       "ConnectionAttempts",
	"connecttimeout":           "ConnectTimeout",
	"dynamicforward":           "DynamicForward",
	"forwardagent":             "ForwardAgent",
	"forwardx11":               "ForwardX11",
	"forwardx11trusted":        "ForwardX11Trusted",
	"hashknownhosts":           "HashKnownHosts",
	"hostname":                 "HostName",
	"identitiesonly":           "IdentitiesOnly",
	"identityagent":            "IdentityAgent",
	"identityfile":             "IdentityFile",
	"kexalgorithms":            "KexAlgorithms",
	"localforward":             "LocalForward",
	"loglevel":                 "LogLevel",
	"macs":                     "MACs",
	"passwordauthentication":   "PasswordAuthentication",
	"port":                     "Port",
	"preferredauthentications": "PreferredAuthentications",
	"proxycommand":             "ProxyCommand",
	"proxyjump":                "ProxyJump",
	"pubkeyauthentication":     "PubkeyAuthentication",
	"remoteforward":            "RemoteForward",
	"user":                     "User",
	"requesttty":               "RequestTTY",
	"serveralivecountmax":      "ServerAliveCountMax",
	"serveraliveinterval":      "ServerAliveInterval",
	"stricthostkeychecking":    "StrictHostKeyChecking",
	"tcpkeepalive":             "TCPKeepAlive",
	"updatehostkeys":           "UpdateHostKeys",
	"verifyhostkeydns":         "VerifyHostKeyDNS",
	"visualhostkey":            "VisualHostKey",
}

// parser handles OpenSSH client config. ordinary fields are read from global
// scope and Host * defaults; Blocks is a synthetic field backed by the block
// engine over named Host/Match blocks.
type parser struct {
	palette []pkg.Field
	openers []string
	isNamed func(opener, header string) bool
	place   pkg.PlacementRule
}

// Palette exposes the schema-derived directive palette for block editing.
func (p *parser) Palette() []pkg.Field { return p.palette }

func (p *parser) FindValue(data []byte, key string) (string, bool) {
	if isBlocksKey(key) {
		return p.findBlocksValue(data)
	}

	lines := strings.Split(string(data), "\n")
	scope := scopeGlobal
	for _, line := range lines {
		scope = updateScope(line, scope)
		if scope == scopeOther {
			continue
		}
		k, v, ok := parseSSHLine(line)
		if !ok || isBlockDirective(k) {
			continue
		}
		if strings.EqualFold(k, key) {
			return v, true
		}
	}
	return "", false
}

func (p *parser) FindValues(data []byte, key string) ([]string, bool) {
	if !isRepeatableSSHKey(key) {
		return nil, false
	}

	lines := strings.Split(string(data), "\n")
	scope := scopeGlobal
	var values []string
	for _, line := range lines {
		scope = updateScope(line, scope)
		if scope == scopeOther {
			continue
		}
		k, v, ok := parseSSHLine(line)
		if !ok || isBlockDirective(k) {
			continue
		}
		if strings.EqualFold(k, key) {
			values = append(values, v)
		}
	}
	if len(values) == 0 {
		return nil, false
	}
	return values, true
}

// FindAll returns key-value pairs from global and Host * scopes in a single pass.
func (p *parser) FindAll(data []byte) map[string]string {
	lines := strings.Split(string(data), "\n")
	m := make(map[string]string)
	scope := scopeGlobal
	for _, line := range lines {
		scope = updateScope(line, scope)
		if scope == scopeOther {
			continue
		}
		k, v, ok := parseSSHLine(line)
		if !ok || isBlockDirective(k) {
			continue
		}
		m[canonicalSSHKey(k)] = v
	}
	if v, ok := p.findBlocksValue(data); ok {
		m[blocksKey] = v
	}
	return m
}

func (p *parser) FindLine(data []byte, key string) (int, bool) {
	if isBlocksKey(key) {
		return p.findBlocksLine(data)
	}

	lines := strings.Split(string(data), "\n")
	scope := scopeGlobal
	for i, line := range lines {
		scope = updateScope(line, scope)
		if scope == scopeOther {
			continue
		}
		k, _, ok := parseSSHLine(line)
		if !ok || isBlockDirective(k) {
			continue
		}
		if strings.EqualFold(k, key) {
			return i, true
		}
	}
	return -1, false
}

func (p *parser) SetValue(data []byte, key, value string) ([]byte, error) {
	if isBlocksKey(key) {
		return p.setBlocksValue(data, value), nil
	}

	lines := strings.Split(string(data), "\n")
	scope := scopeGlobal
	for i, line := range lines {
		scope = updateScope(line, scope)
		if scope == scopeOther {
			continue
		}
		k, _, ok := parseSSHLine(line)
		if !ok || isBlockDirective(k) {
			continue
		}
		if strings.EqualFold(k, key) {
			lines[i] = replaceSSHValue(line, value)
			return []byte(strings.Join(lines, "\n")), nil
		}
	}

	return insertSSHKey(lines, key, value), nil
}

func (p *parser) SetValues(data []byte, key string, values []string) ([]byte, error) {
	if !isRepeatableSSHKey(key) {
		return data, nil
	}

	lines := strings.Split(string(data), "\n")
	out := make([]string, 0, len(lines)+len(values))
	scope := scopeGlobal
	inserted := false
	for _, line := range lines {
		scope = updateScope(line, scope)
		if scope == scopeOther {
			out = append(out, line)
			continue
		}
		k, _, ok := parseSSHLine(line)
		if !ok || isBlockDirective(k) || !strings.EqualFold(k, key) {
			out = append(out, line)
			continue
		}
		if !inserted {
			for _, value := range values {
				out = append(out, replaceSSHValue(line, value))
			}
			inserted = true
		}
	}
	if inserted {
		return []byte(strings.Join(out, "\n")), nil
	}
	if len(values) == 0 {
		return data, nil
	}
	return insertSSHKeys(lines, canonicalSSHKey(key), values), nil
}

func (p *parser) DeleteKey(data []byte, key string) ([]byte, error) {
	if isBlocksKey(key) {
		return p.deleteBlocks(data), nil
	}

	lines := strings.Split(string(data), "\n")
	scope := scopeGlobal
	for i, line := range lines {
		scope = updateScope(line, scope)
		if scope == scopeOther {
			continue
		}
		k, _, ok := parseSSHLine(line)
		if !ok || isBlockDirective(k) {
			continue
		}
		if strings.EqualFold(k, key) {
			lines = append(lines[:i], lines[i+1:]...)
			return []byte(strings.Join(lines, "\n")), nil
		}
	}
	return data, nil
}

func (p *parser) ListKeys(data []byte) []string {
	lines := strings.Split(string(data), "\n")
	var keys []string
	scope := scopeGlobal
	for _, line := range lines {
		scope = updateScope(line, scope)
		if scope == scopeOther {
			continue
		}
		k, _, ok := parseSSHLine(line)
		if ok && !isBlockDirective(k) {
			keys = append(keys, canonicalSSHKey(k))
		}
	}
	if _, ok := p.findBlocksValue(data); ok {
		keys = append(keys, blocksKey)
	}
	return keys
}

type sshScope int

const (
	scopeGlobal sshScope = iota
	scopeWild
	scopeOther
)

func updateScope(line string, current sshScope) sshScope {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || trimmed[0] == '#' {
		return current
	}
	key, value, ok := parseSSHLine(trimmed)
	if !ok {
		return current
	}
	if strings.EqualFold(key, "Host") {
		if isDefaultHostPattern(value) {
			return scopeWild
		}
		return scopeOther
	}
	if strings.EqualFold(key, "Match") {
		return scopeOther
	}
	return current
}

func parseSSHLine(line string) (key, value string, ok bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || trimmed[0] == '#' {
		return "", "", false
	}

	if eqIdx := strings.IndexByte(trimmed, '='); eqIdx >= 0 {
		k := strings.TrimSpace(trimmed[:eqIdx])
		if k != "" && !strings.ContainsAny(k, " \t") {
			v := strings.TrimSpace(trimmed[eqIdx+1:])
			return k, v, true
		}
	}

	idx := strings.IndexAny(trimmed, " \t")
	if idx < 0 {
		return trimmed, "", true
	}

	key = trimmed[:idx]
	value = strings.TrimSpace(trimmed[idx+1:])
	return key, value, true
}

func replaceSSHValue(line, newValue string) string {
	trimmed := strings.TrimSpace(line)
	indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]

	if eqIdx := strings.IndexByte(trimmed, '='); eqIdx >= 0 {
		k := strings.TrimSpace(trimmed[:eqIdx])
		if k != "" && !strings.ContainsAny(k, " \t") {
			return indent + k + " = " + newValue
		}
	}

	idx := strings.IndexAny(trimmed, " \t")
	if idx < 0 {
		return line
	}
	keyword := trimmed[:idx]
	return indent + keyword + " " + newValue
}

func insertSSHKey(lines []string, key, value string) []byte {
	return insertSSHKeys(lines, key, []string{value})
}

func insertSSHKeys(lines []string, key string, values []string) []byte {
	if len(values) == 0 {
		return []byte(strings.Join(lines, "\n"))
	}

	directives := make([]string, 0, len(values))
	for _, value := range values {
		directives = append(directives, "    "+key+" "+value)
	}

	inWild := false
	insertAt := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || trimmed[0] == '#' {
			if inWild {
				insertAt = i
			}
			continue
		}
		k, v, ok := parseSSHLine(trimmed)
		if ok && isBlockDirective(k) {
			if inWild {
				result := make([]string, 0, len(lines)+1)
				result = append(result, lines[:i]...)
				result = append(result, directives...)
				result = append(result, lines[i:]...)
				return []byte(strings.Join(result, "\n"))
			}
			if strings.EqualFold(k, "Host") && isDefaultHostPattern(v) {
				inWild = true
				insertAt = i
			}
		} else if inWild {
			insertAt = i
		}
	}

	if inWild && insertAt >= 0 {
		result := make([]string, 0, len(lines)+1)
		result = append(result, lines[:insertAt+1]...)
		result = append(result, directives...)
		result = append(result, lines[insertAt+1:]...)
		return []byte(strings.Join(result, "\n"))
	}

	// no Host * block exists: create one at lowest precedence (after all
	// host-specific blocks). ssh is first-match-wins, so prepending at root
	// would create a HIGH-precedence global, not a default — never do that.
	return appendDefaultHostBlock(lines, key, values)
}

// appendDefaultHostBlock appends a new "Host *" block (lowest precedence).
// it preserves a trailing-newline-or-not input shape.
func appendDefaultHostBlock(lines []string, key string, values []string) []byte {
	blockLines := []string{"Host *"}
	for _, value := range values {
		blockLines = append(blockLines, "    "+key+" "+value)
	}
	block := strings.Join(blockLines, "\n")

	// empty / blank-only input: emit just the block (no leading newline).
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		return []byte(block + "\n")
	}

	// honor an original trailing newline: a final empty element means the
	// source ended in "\n". drop it, append the block, restore the newline.
	hadTrailingNewline := lines[len(lines)-1] == ""
	body := lines
	if hadTrailingNewline {
		body = body[:len(body)-1]
	}

	out := append([]string(nil), body...)
	if len(out) > 0 && strings.TrimSpace(out[len(out)-1]) != "" {
		out = append(out, "")
	}
	out = append(out, blockLines...)
	if hadTrailingNewline {
		out = append(out, "")
	}
	return []byte(strings.Join(out, "\n"))
}

func isBlocksKey(key string) bool {
	return strings.EqualFold(key, blocksKey)
}

func isRepeatableSSHKey(key string) bool {
	return repeatableSSHKeys[strings.ToLower(key)]
}

// isNamedBlock reports whether a block is "named" for the engine: any Match
// block, or a Host block whose header is NOT the default "*" pattern.
func isNamedBlock(opener, header string) bool {
	if strings.EqualFold(opener, "Match") {
		return true
	}
	if strings.EqualFold(opener, "Host") {
		return !isDefaultHostPattern(header)
	}
	return false
}

// isLowPrecedenceBlock identifies the flat block kept at lowest precedence: the
// default "Host *" stanza.
func isLowPrecedenceBlock(opener, header string) bool {
	return strings.EqualFold(opener, "Host") && isDefaultHostPattern(header)
}

// findBlocksValue encodes the named-block model. it reports ok=false when there
// are no named blocks, preserving "not configured when empty" semantics.
func (p *parser) findBlocksValue(data []byte) (string, bool) {
	model := pkg.Parse(data, p.openers, p.isNamed)
	if len(model.Blocks) == 0 {
		return "", false
	}
	return pkg.Encode(model), true
}

// setBlocksValue reconciles a decoded named-block model back into data, leaving
// root directives and the default Host * block byte-stable.
func (p *parser) setBlocksValue(data []byte, value string) []byte {
	return pkg.Reconcile(data, pkg.Decode(value), p.openers, p.isNamed, p.place)
}

// deleteBlocks drops every named block, keeping flat regions byte-stable.
func (p *parser) deleteBlocks(data []byte) []byte {
	return pkg.Reconcile(data, pkg.BlockModel{}, p.openers, p.isNamed, p.place)
}

// findBlocksLine returns the source line index of the first named block.
func (p *parser) findBlocksLine(data []byte) (int, bool) {
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		opener, header, ok := openerHeader(line)
		if ok && p.isNamed(opener, header) {
			return i, true
		}
	}
	return -1, false
}

// openerHeader reports whether line opens a Host/Match block, returning the
// canonical opener and its raw header text.
func openerHeader(line string) (opener, header string, ok bool) {
	key, value, parsed := parseSSHLine(line)
	if !parsed {
		return "", "", false
	}
	if strings.EqualFold(key, "Host") || strings.EqualFold(key, "Match") {
		return key, value, true
	}
	return "", "", false
}

func isBlockDirective(key string) bool {
	return strings.EqualFold(key, "Host") || strings.EqualFold(key, "Match")
}

func isDefaultHostPattern(patterns string) bool {
	fields := strings.Fields(patterns)
	return len(fields) == 1 && fields[0] == "*"
}

func canonicalSSHKey(key string) string {
	if v, ok := canonicalSSHKeys[strings.ToLower(key)]; ok {
		return v
	}
	return key
}
