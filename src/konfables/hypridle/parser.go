package hypridle

import (
	"strings"

	hyprparser "github.com/getkonfi/konfi/pkg/parser"
)

const (
	listenersKey      = "listeners"
	listenerSeparator = " <-> "
)

type parser struct {
	base *hyprparser.HyprParser
}

type listenerSpec struct {
	timeout       string
	onTimeout     string
	onResume      string
	ignoreInhibit string
}

type lineSpan struct {
	start int
	end   int
}

func newParser() *parser {
	return &parser{base: hyprparser.NewHyprParser()}
}

func (p *parser) FindValue(data []byte, key string) (string, bool) {
	if isListenersKey(key) {
		return findListenersValue(data)
	}
	return p.base.FindValue(data, key)
}

func (p *parser) FindLine(data []byte, key string) (int, bool) {
	if isListenersKey(key) {
		spans := listenerSpans(strings.Split(string(data), "\n"))
		if len(spans) == 0 {
			return -1, false
		}
		return spans[0].start, true
	}
	return p.base.FindLine(data, key)
}

func (p *parser) SetValue(data []byte, key, value string) ([]byte, error) {
	if isListenersKey(key) {
		return setListenersValue(data, value), nil
	}
	return p.base.SetValue(data, key, value)
}

func (p *parser) DeleteKey(data []byte, key string) ([]byte, error) {
	if isListenersKey(key) {
		return setListenersValue(data, ""), nil
	}
	return p.base.DeleteKey(data, key)
}

func (p *parser) ListKeys(data []byte) []string {
	keys := p.base.ListKeys(data)
	out := make([]string, 0, len(keys)+1)
	seen := make(map[string]bool, len(keys)+1)
	for _, key := range keys {
		if strings.HasPrefix(key, "listener.") || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, key)
	}
	if _, ok := findListenersValue(data); ok && !seen[listenersKey] {
		out = append(out, listenersKey)
	}
	return out
}

func (p *parser) FindAll(data []byte) map[string]string {
	all := p.base.FindAll(data)
	for key := range all {
		if strings.HasPrefix(key, "listener.") {
			delete(all, key)
		}
	}
	if value, ok := findListenersValue(data); ok {
		all[listenersKey] = value
	}
	return all
}

func isListenersKey(key string) bool {
	return strings.EqualFold(key, listenersKey)
}

func findListenersValue(data []byte) (string, bool) {
	lines := strings.Split(string(data), "\n")
	spans := listenerSpans(lines)
	if len(spans) == 0 {
		return "", false
	}

	rows := make([]string, 0, len(spans))
	for _, span := range spans {
		spec := parseListenerBlock(lines[span.start+1 : span.end])
		rows = append(rows, joinListenerRow(spec))
	}
	return strings.Join(rows, "\n"), true
}

func setListenersValue(data []byte, value string) []byte {
	lines := strings.Split(string(data), "\n")
	spans := listenerSpans(lines)
	specs := parseListenerRows(value)
	rendered := renderListeners(specs)

	if len(spans) == 0 {
		if len(rendered) == 0 {
			return data
		}
		return []byte(strings.Join(appendAtEnd(lines, rendered), "\n"))
	}

	out := make([]string, 0, len(lines)+len(rendered))
	inserted := false
	spanIdx := 0
	for i := 0; i < len(lines); {
		if spanIdx < len(spans) && i == spans[spanIdx].start {
			if !inserted {
				out = append(out, rendered...)
				inserted = true
			}
			i = spans[spanIdx].end + 1
			spanIdx++
			continue
		}
		out = append(out, lines[i])
		i++
	}
	return []byte(strings.Join(out, "\n"))
}

func listenerSpans(lines []string) []lineSpan {
	var spans []lineSpan
	depth := 0

	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if depth == 0 && isListenerOpen(trimmed) {
			blockDepth := braceBalance(trimmed)
			end := i
			for end+1 < len(lines) && blockDepth > 0 {
				end++
				blockDepth += braceBalance(strings.TrimSpace(lines[end]))
			}
			spans = append(spans, lineSpan{start: i, end: end})
			i = end
			continue
		}
		depth += braceBalance(trimmed)
		if depth < 0 {
			depth = 0
		}
	}

	return spans
}

func isListenerOpen(trimmed string) bool {
	return trimmed == "listener {" || trimmed == "listener{"
}

func parseListenerBlock(lines []string) listenerSpec {
	var spec listenerSpec
	for _, line := range lines {
		key, value, ok := parseAssignment(line)
		if !ok {
			continue
		}
		switch key {
		case "timeout":
			spec.timeout = value
		case "on-timeout":
			spec.onTimeout = value
		case "on-resume":
			spec.onResume = value
		case "ignore_inhibit":
			spec.ignoreInhibit = value
		}
	}
	return spec
}

func parseListenerRows(value string) []listenerSpec {
	var specs []listenerSpec
	for _, line := range strings.Split(value, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, listenerSeparator, 4)
		for len(parts) < 4 {
			parts = append(parts, "")
		}
		spec := listenerSpec{
			timeout:       strings.TrimSpace(parts[0]),
			onTimeout:     strings.TrimSpace(parts[1]),
			onResume:      strings.TrimSpace(parts[2]),
			ignoreInhibit: strings.TrimSpace(parts[3]),
		}
		if spec.timeout == "" && spec.onTimeout == "" && spec.onResume == "" && spec.ignoreInhibit == "" {
			continue
		}
		specs = append(specs, spec)
	}
	return specs
}

func joinListenerRow(spec listenerSpec) string {
	return strings.Join([]string{
		spec.timeout,
		spec.onTimeout,
		spec.onResume,
		spec.ignoreInhibit,
	}, listenerSeparator)
}

func renderListeners(specs []listenerSpec) []string {
	if len(specs) == 0 {
		return nil
	}
	out := make([]string, 0, len(specs)*7)
	for i, spec := range specs {
		if i > 0 {
			out = append(out, "")
		}
		out = append(out, "listener {")
		if spec.timeout != "" {
			out = append(out, "    timeout = "+spec.timeout)
		}
		if spec.onTimeout != "" {
			out = append(out, "    on-timeout = "+spec.onTimeout)
		}
		if spec.onResume != "" {
			out = append(out, "    on-resume = "+spec.onResume)
		}
		if spec.ignoreInhibit != "" {
			out = append(out, "    ignore_inhibit = "+spec.ignoreInhibit)
		}
		out = append(out, "}")
	}
	return out
}

func appendAtEnd(lines, rendered []string) []string {
	if len(lines) == 0 || len(lines) == 1 && lines[0] == "" {
		return append(append([]string(nil), rendered...), "")
	}

	hadTrailingNewline := lines[len(lines)-1] == ""
	body := append([]string(nil), lines...)
	if hadTrailingNewline {
		body = body[:len(body)-1]
	}
	if len(body) > 0 && strings.TrimSpace(body[len(body)-1]) != "" {
		body = append(body, "")
	}
	body = append(body, rendered...)
	if hadTrailingNewline {
		body = append(body, "")
	}
	return body
}

func parseAssignment(line string) (key, value string, ok bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || trimmed[0] == '#' || trimmed == "}" || strings.HasSuffix(trimmed, "{") {
		return "", "", false
	}
	k, v, found := strings.Cut(trimmed, " = ")
	if found {
		return k, v, true
	}
	k, v, found = strings.Cut(trimmed, "=")
	if !found {
		return "", "", false
	}
	return strings.TrimSpace(k), strings.TrimSpace(v), true
}

func braceBalance(s string) int {
	bal := 0
	for _, r := range s {
		switch r {
		case '{':
			bal++
		case '}':
			bal--
		}
	}
	return bal
}
