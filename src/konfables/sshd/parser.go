package sshd

import (
	"strings"

	"github.com/eminert/konfi/pkg"
)

const blocksKey = "Blocks"

var canonicalSSHDKeys = map[string]string{
	"acceptenv":                    "AcceptEnv",
	"addressfamily":                "AddressFamily",
	"allowagentforwarding":         "AllowAgentForwarding",
	"allowgroups":                  "AllowGroups",
	"allowstreamlocalforwarding":   "AllowStreamLocalForwarding",
	"allowtcpforwarding":           "AllowTcpForwarding",
	"allowusers":                   "AllowUsers",
	"authenticationmethods":        "AuthenticationMethods",
	"authorizedkeyscommand":        "AuthorizedKeysCommand",
	"authorizedkeyscommanduser":    "AuthorizedKeysCommandUser",
	"authorizedkeysfile":           "AuthorizedKeysFile",
	"banner":                       "Banner",
	"chrootdirectory":              "ChrootDirectory",
	"ciphers":                      "Ciphers",
	"clientalivecountmax":          "ClientAliveCountMax",
	"clientaliveinterval":          "ClientAliveInterval",
	"denygroups":                   "DenyGroups",
	"denyusers":                    "DenyUsers",
	"forcecommand":                 "ForceCommand",
	"gatewayports":                 "GatewayPorts",
	"gssapiauthentication":         "GSSAPIAuthentication",
	"hostbasedauthentication":      "HostbasedAuthentication",
	"hostkey":                      "HostKey",
	"kbdinteractiveauthentication": "KbdInteractiveAuthentication",
	"kexalgorithms":                "KexAlgorithms",
	"listenaddress":                "ListenAddress",
	"logingracetime":               "LoginGraceTime",
	"loglevel":                     "LogLevel",
	"macs":                         "MACs",
	"maxsessions":                  "MaxSessions",
	"maxstartups":                  "MaxStartups",
	"passwordauthentication":       "PasswordAuthentication",
	"permitemptypasswords":         "PermitEmptyPasswords",
	"permitrootlogin":              "PermitRootLogin",
	"permittunnel":                 "PermitTunnel",
	"permittty":                    "PermitTTY",
	"port":                         "Port",
	"printmotd":                    "PrintMotd",
	"pubkeyauthentication":         "PubkeyAuthentication",
	"subsystem":                    "Subsystem",
	"syslogfacility":               "SyslogFacility",
	"usepam":                       "UsePAM",
	"x11forwarding":                "X11Forwarding",
}

var matchAllowedKeys = map[string]bool{
	"acceptenv":                    true,
	"allowagentforwarding":         true,
	"allowgroups":                  true,
	"allowstreamlocalforwarding":   true,
	"allowtcpforwarding":           true,
	"allowusers":                   true,
	"authenticationmethods":        true,
	"authorizedkeyscommand":        true,
	"authorizedkeyscommanduser":    true,
	"authorizedkeysfile":           true,
	"banner":                       true,
	"chrootdirectory":              true,
	"denygroups":                   true,
	"denyusers":                    true,
	"forcecommand":                 true,
	"gatewayports":                 true,
	"gssapiauthentication":         true,
	"hostbasedauthentication":      true,
	"kbdinteractiveauthentication": true,
	"loglevel":                     true,
	"maxsessions":                  true,
	"passwordauthentication":       true,
	"permitemptypasswords":         true,
	"permitrootlogin":              true,
	"permittunnel":                 true,
	"permittty":                    true,
	"pubkeyauthentication":         true,
	"x11forwarding":                true,
}

type parser struct {
	palette []pkg.Field
}

func (p *parser) Palette() []pkg.Field { return p.palette }

func (p *parser) FindValue(data []byte, key string) (string, bool) {
	if isBlocksKey(key) {
		return p.findBlocksValue(data)
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if isMatchLine(line) {
			break
		}
		k, v, ok := parseSSHDLine(line)
		if !ok {
			continue
		}
		if strings.EqualFold(k, key) {
			return v, true
		}
	}
	return "", false
}

func (p *parser) FindAll(data []byte) map[string]string {
	lines := strings.Split(string(data), "\n")
	m := make(map[string]string)
	for _, line := range lines {
		if isMatchLine(line) {
			break
		}
		k, v, ok := parseSSHDLine(line)
		if !ok {
			continue
		}
		m[canonicalSSHDKey(k)] = v
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
	for i, line := range lines {
		if isMatchLine(line) {
			break
		}
		k, _, ok := parseSSHDLine(line)
		if !ok {
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
	for i, line := range lines {
		if isMatchLine(line) {
			break
		}
		k, _, ok := parseSSHDLine(line)
		if !ok {
			continue
		}
		if strings.EqualFold(k, key) {
			lines[i] = replaceSSHDValue(line, value)
			return []byte(strings.Join(lines, "\n")), nil
		}
	}
	return insertGlobalKey(lines, key, value), nil
}

func (p *parser) DeleteKey(data []byte, key string) ([]byte, error) {
	if isBlocksKey(key) {
		return p.deleteBlocks(data), nil
	}
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if isMatchLine(line) {
			break
		}
		k, _, ok := parseSSHDLine(line)
		if !ok {
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
	for _, line := range lines {
		if isMatchLine(line) {
			break
		}
		k, _, ok := parseSSHDLine(line)
		if ok {
			keys = append(keys, canonicalSSHDKey(k))
		}
	}
	if _, ok := p.findBlocksValue(data); ok {
		keys = append(keys, blocksKey)
	}
	return keys
}

func parseSSHDLine(line string) (key, value string, ok bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || trimmed[0] == '#' {
		return "", "", false
	}
	if eqIdx := strings.IndexByte(trimmed, '='); eqIdx >= 0 {
		k := strings.TrimSpace(trimmed[:eqIdx])
		if k != "" && !strings.ContainsAny(k, " \t") {
			return k, strings.TrimSpace(trimmed[eqIdx+1:]), true
		}
	}
	idx := strings.IndexAny(trimmed, " \t")
	if idx < 0 {
		return trimmed, "", true
	}
	return trimmed[:idx], strings.TrimSpace(trimmed[idx+1:]), true
}

func replaceSSHDValue(line, newValue string) string {
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

func insertGlobalKey(lines []string, key, value string) []byte {
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		return []byte(key + " " + value + "\n")
	}

	matchAt := -1
	for i, line := range lines {
		if isMatchLine(line) {
			matchAt = i
			break
		}
	}

	hadTrailingNewline := lines[len(lines)-1] == ""
	body := lines
	if hadTrailingNewline {
		body = body[:len(body)-1]
	}

	out := append([]string(nil), body...)
	insert := key + " " + value
	if matchAt >= 0 {
		insertAt := matchAt
		for insertAt > 0 {
			trimmed := strings.TrimSpace(body[insertAt-1])
			if trimmed == "" || strings.HasPrefix(trimmed, "#") {
				insertAt--
				continue
			}
			break
		}
		out = append([]string(nil), body[:insertAt]...)
		if len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
			out = append(out, insert)
		} else {
			out = append(out, insert, "")
		}
		out = append(out, body[insertAt:]...)
	} else {
		if len(out) > 0 && strings.TrimSpace(out[len(out)-1]) != "" {
			out = append(out, insert)
		} else {
			out = append(out, insert)
		}
	}
	if hadTrailingNewline {
		out = append(out, "")
	}
	return []byte(strings.Join(out, "\n"))
}

func (p *parser) findBlocksValue(data []byte) (string, bool) {
	model := pkg.Parse(data, []string{"Match"}, isMatchBlock)
	if len(model.Blocks) == 0 {
		return "", false
	}
	return pkg.Encode(model), true
}

func (p *parser) setBlocksValue(data []byte, value string) []byte {
	return pkg.Reconcile(data, pkg.Decode(value), []string{"Match"}, isMatchBlock, pkg.PlacementRule{})
}

func (p *parser) deleteBlocks(data []byte) []byte {
	return pkg.Reconcile(data, pkg.BlockModel{}, []string{"Match"}, isMatchBlock, pkg.PlacementRule{})
}

func (p *parser) findBlocksLine(data []byte) (int, bool) {
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if isMatchLine(line) {
			return i, true
		}
	}
	return -1, false
}

func isBlocksKey(key string) bool {
	return strings.EqualFold(key, blocksKey)
}

func isMatchBlock(opener, _ string) bool {
	return strings.EqualFold(opener, "Match")
}

func isMatchLine(line string) bool {
	key, _, ok := parseSSHDLine(line)
	return ok && strings.EqualFold(key, "Match")
}

func isMatchAllowed(key string) bool {
	return matchAllowedKeys[strings.ToLower(key)]
}

func canonicalSSHDKey(key string) string {
	if v, ok := canonicalSSHDKeys[strings.ToLower(key)]; ok {
		return v
	}
	return key
}
