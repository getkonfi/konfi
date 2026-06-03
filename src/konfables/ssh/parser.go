package ssh

import (
	"strings"
)

const hostsKey = "Hosts"

var hostRowDirectives = []string{"HostName", "User", "Port", "IdentityFile", "ProxyJump"}

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
	"identitiesonly":           "IdentitiesOnly",
	"identityagent":            "IdentityAgent",
	"kexalgorithms":            "KexAlgorithms",
	"localforward":             "LocalForward",
	"loglevel":                 "LogLevel",
	"macs":                     "MACs",
	"passwordauthentication":   "PasswordAuthentication",
	"preferredauthentications": "PreferredAuthentications",
	"proxycommand":             "ProxyCommand",
	"pubkeyauthentication":     "PubkeyAuthentication",
	"remoteforward":            "RemoteForward",
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
// scope and Host * defaults; Hosts is a synthetic field for named Host blocks.
type parser struct{}

func (p *parser) FindValue(data []byte, key string) (string, bool) {
	if isHostsKey(key) {
		return findHostsValue(data)
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
	if v, ok := findHostsValue(data); ok {
		m[hostsKey] = v
	}
	return m
}

func (p *parser) FindLine(data []byte, key string) (int, bool) {
	if isHostsKey(key) {
		for _, block := range parseHostBlocks(strings.Split(string(data), "\n")) {
			if !block.defaultHost {
				return block.start, true
			}
		}
		return -1, false
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
	if isHostsKey(key) {
		return setHostsValue(data, value), nil
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

func (p *parser) DeleteKey(data []byte, key string) ([]byte, error) {
	if isHostsKey(key) {
		return deleteHostBlocks(data), nil
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
	if _, ok := findHostsValue(data); ok {
		keys = append(keys, hostsKey)
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
		if ok && strings.EqualFold(k, "Host") {
			if inWild {
				result := make([]string, 0, len(lines)+1)
				result = append(result, lines[:i]...)
				result = append(result, "    "+key+" "+value)
				result = append(result, lines[i:]...)
				return []byte(strings.Join(result, "\n"))
			}
			if isDefaultHostPattern(v) {
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
		result = append(result, "    "+key+" "+value)
		result = append(result, lines[insertAt+1:]...)
		return []byte(strings.Join(result, "\n"))
	}

	result := key + " " + value
	if len(lines) == 0 || len(lines) == 1 && lines[0] == "" {
		return []byte(result + "\n")
	}
	return []byte(result + "\n" + strings.Join(lines, "\n"))
}

type hostBlock struct {
	start       int
	end         int
	patterns    string
	defaultHost bool
	indent      string
	lines       []string
}

type hostRow struct {
	patterns     string
	hostName     string
	user         string
	port         string
	identityFile string
	proxyJump    string
}

func findHostsValue(data []byte) (string, bool) {
	lines := strings.Split(string(data), "\n")
	blocks := parseHostBlocks(lines)
	rows := make([]string, 0, len(blocks))
	for _, block := range blocks {
		if block.defaultHost {
			continue
		}
		row := hostRow{
			patterns:     block.patterns,
			hostName:     firstHostDirective(block.lines, "HostName"),
			user:         firstHostDirective(block.lines, "User"),
			port:         firstHostDirective(block.lines, "Port"),
			identityFile: firstHostDirective(block.lines, "IdentityFile"),
			proxyJump:    firstHostDirective(block.lines, "ProxyJump"),
		}
		rows = append(rows, formatHostRow(row))
	}
	if len(rows) == 0 {
		return "", false
	}
	return strings.Join(rows, "\n"), true
}

func setHostsValue(data []byte, value string) []byte {
	rows := parseHostRows(value)
	lines := strings.Split(string(data), "\n")
	blocks := parseHostBlocks(lines)

	blocksByStart := make(map[int]hostBlock, len(blocks))
	for _, block := range blocks {
		if !block.defaultHost {
			blocksByStart[block.start] = block
		}
	}

	rowsByPattern := make(map[string]hostRow, len(rows))
	rowOrder := make([]string, 0, len(rows))
	for _, row := range rows {
		key := normalizeHostPattern(row.patterns)
		if key == "" {
			continue
		}
		if _, seen := rowsByPattern[key]; !seen {
			rowOrder = append(rowOrder, key)
		}
		rowsByPattern[key] = row
	}

	out := make([]string, 0, len(lines)+len(rows)*6)
	used := make(map[string]bool, len(rowsByPattern))
	insertedNew := false

	for i := 0; i < len(lines); {
		if block, ok := blocksByStart[i]; ok {
			key := normalizeHostPattern(block.patterns)
			if row, keep := rowsByPattern[key]; keep {
				out = append(out, renderHostBlock(row, &block)...)
				used[key] = true
			}
			i = block.end
			continue
		}

		if !insertedNew && isDefaultHostLine(lines[i]) {
			before := len(out)
			out = appendMissingHostRows(out, rowOrder, rowsByPattern, used)
			if len(out) > before && strings.TrimSpace(out[len(out)-1]) != "" {
				out = append(out, "")
			}
			insertedNew = true
		}

		out = append(out, lines[i])
		i++
	}

	if !insertedNew {
		out = appendMissingHostRows(out, rowOrder, rowsByPattern, used)
	}

	return []byte(strings.Join(out, "\n"))
}

func deleteHostBlocks(data []byte) []byte {
	lines := strings.Split(string(data), "\n")
	blocks := parseHostBlocks(lines)
	blocksByStart := make(map[int]hostBlock, len(blocks))
	for _, block := range blocks {
		if !block.defaultHost {
			blocksByStart[block.start] = block
		}
	}

	out := make([]string, 0, len(lines))
	for i := 0; i < len(lines); {
		if block, ok := blocksByStart[i]; ok {
			i = block.end
			continue
		}
		out = append(out, lines[i])
		i++
	}
	return []byte(strings.Join(out, "\n"))
}

func parseHostBlocks(lines []string) []hostBlock {
	var blocks []hostBlock
	for i := 0; i < len(lines); i++ {
		key, value, ok := parseSSHLine(lines[i])
		if !ok || !strings.EqualFold(key, "Host") {
			continue
		}

		end := len(lines)
		for j := i + 1; j < len(lines); j++ {
			nextKey, _, nextOK := parseSSHLine(lines[j])
			if nextOK && (strings.EqualFold(nextKey, "Host") || strings.EqualFold(nextKey, "Match")) {
				end = j
				break
			}
		}

		blockLines := append([]string(nil), lines[i:end]...)
		blocks = append(blocks, hostBlock{
			start:       i,
			end:         end,
			patterns:    value,
			defaultHost: isDefaultHostPattern(value),
			indent:      inferHostIndent(blockLines),
			lines:       blockLines,
		})
		i = end - 1
	}
	return blocks
}

func inferHostIndent(lines []string) string {
	for _, line := range lines[1:] {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || trimmed[0] == '#' {
			continue
		}
		key, _, ok := parseSSHLine(line)
		if ok && !isBlockDirective(key) {
			return line[:len(line)-len(strings.TrimLeft(line, " \t"))]
		}
	}
	return "    "
}

func firstHostDirective(lines []string, want string) string {
	for _, line := range lines[1:] {
		key, value, ok := parseSSHLine(line)
		if ok && strings.EqualFold(key, want) {
			return value
		}
	}
	return ""
}

func parseHostRows(value string) []hostRow {
	var rows []hostRow
	for _, line := range strings.Split(value, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", len(hostRowDirectives)+1)
		for len(parts) < len(hostRowDirectives)+1 {
			parts = append(parts, "")
		}
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		if parts[0] == "" {
			continue
		}
		rows = append(rows, hostRow{
			patterns:     parts[0],
			hostName:     parts[1],
			user:         parts[2],
			port:         parts[3],
			identityFile: parts[4],
			proxyJump:    parts[5],
		})
	}
	return rows
}

func formatHostRow(row hostRow) string {
	return strings.Join([]string{
		row.patterns,
		row.hostName,
		row.user,
		row.port,
		row.identityFile,
		row.proxyJump,
	}, " | ")
}

func renderHostBlock(row hostRow, existing *hostBlock) []string {
	if existing == nil {
		out := []string{"Host " + row.patterns}
		return appendHostRowDirectives(out, row, "    ")
	}

	out := []string{existing.lines[0]}
	body := make([]string, 0, len(existing.lines))
	for _, line := range existing.lines[1:] {
		key, _, ok := parseSSHLine(line)
		if ok && isHostRowDirective(key) {
			continue
		}
		body = append(body, line)
	}

	trailingBlank := make([]string, 0)
	for len(body) > 0 && strings.TrimSpace(body[len(body)-1]) == "" {
		trailingBlank = append(trailingBlank, body[len(body)-1])
		body = body[:len(body)-1]
	}

	out = append(out, body...)
	out = appendHostRowDirectives(out, row, existing.indent)
	for i := len(trailingBlank) - 1; i >= 0; i-- {
		out = append(out, trailingBlank[i])
	}
	return out
}

func appendHostRowDirectives(out []string, row hostRow, indent string) []string {
	values := []string{row.hostName, row.user, row.port, row.identityFile, row.proxyJump}
	for i, directive := range hostRowDirectives {
		if values[i] != "" {
			out = append(out, indent+directive+" "+values[i])
		}
	}
	return out
}

func appendMissingHostRows(out []string, order []string, rows map[string]hostRow, used map[string]bool) []string {
	if len(out) == 1 && out[0] == "" {
		out = out[:0]
	}
	for _, key := range order {
		if used[key] {
			continue
		}
		if len(out) > 0 && strings.TrimSpace(out[len(out)-1]) != "" {
			out = append(out, "")
		}
		out = append(out, renderHostBlock(rows[key], nil)...)
		used[key] = true
	}
	return out
}

func isHostsKey(key string) bool {
	return strings.EqualFold(key, hostsKey)
}

func isBlockDirective(key string) bool {
	return strings.EqualFold(key, "Host") || strings.EqualFold(key, "Match")
}

func isDefaultHostLine(line string) bool {
	key, value, ok := parseSSHLine(line)
	return ok && strings.EqualFold(key, "Host") && isDefaultHostPattern(value)
}

func isDefaultHostPattern(patterns string) bool {
	fields := strings.Fields(patterns)
	return len(fields) == 1 && fields[0] == "*"
}

func isHostRowDirective(key string) bool {
	for _, directive := range hostRowDirectives {
		if strings.EqualFold(key, directive) {
			return true
		}
	}
	return false
}

func normalizeHostPattern(patterns string) string {
	return strings.Join(strings.Fields(patterns), " ")
}

func canonicalSSHKey(key string) string {
	if strings.EqualFold(key, hostsKey) {
		return hostsKey
	}
	for _, directive := range hostRowDirectives {
		if strings.EqualFold(key, directive) {
			return directive
		}
	}
	if v, ok := canonicalSSHKeys[strings.ToLower(key)]; ok {
		return v
	}
	return key
}
