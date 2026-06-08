package waybar

import "github.com/eminert/konfi/pkg/parser"

type jsoncParser struct {
	base parser.JSONParser
}

func newParser() *jsoncParser {
	return &jsoncParser{}
}

func (p *jsoncParser) FindValue(data []byte, key string) (string, bool) {
	return p.base.FindValue(stripJSONC(data), key)
}

func (p *jsoncParser) FindLine(data []byte, key string) (int, bool) {
	return p.base.FindLine(stripJSONC(data), key)
}

func (p *jsoncParser) SetValue(data []byte, key, value string) ([]byte, error) {
	return p.base.SetValue(stripJSONC(data), key, value)
}

func (p *jsoncParser) DeleteKey(data []byte, key string) ([]byte, error) {
	return p.base.DeleteKey(stripJSONC(data), key)
}

func (p *jsoncParser) ListKeys(data []byte) []string {
	return p.base.ListKeys(stripJSONC(data))
}

func (p *jsoncParser) FindAll(data []byte) map[string]string {
	return p.base.FindAll(stripJSONC(data))
}

func (p *jsoncParser) FindValues(data []byte, key string) ([]string, bool) {
	return p.base.FindValues(stripJSONC(data), key)
}

func (p *jsoncParser) SetValues(data []byte, key string, values []string) ([]byte, error) {
	return p.base.SetValues(stripJSONC(data), key, values)
}

func stripJSONC(data []byte) []byte {
	return stripTrailingCommas(stripJSONCComments(data))
}

func stripJSONCComments(data []byte) []byte {
	out := make([]byte, 0, len(data))
	inString := false
	escaped := false
	inLineComment := false
	inBlockComment := false

	for i := 0; i < len(data); i++ {
		ch := data[i]

		if inLineComment {
			if ch == '\n' {
				inLineComment = false
				out = append(out, ch)
			} else {
				out = append(out, ' ')
			}
			continue
		}

		if inBlockComment {
			if ch == '*' && i+1 < len(data) && data[i+1] == '/' {
				inBlockComment = false
				out = append(out, ' ', ' ')
				i++
				continue
			}
			if ch == '\n' {
				out = append(out, ch)
			} else {
				out = append(out, ' ')
			}
			continue
		}

		if inString {
			out = append(out, ch)
			if escaped {
				escaped = false
				continue
			}
			switch ch {
			case '\\':
				escaped = true
			case '"':
				inString = false
			}
			continue
		}

		if ch == '"' {
			inString = true
			out = append(out, ch)
			continue
		}
		if ch == '/' && i+1 < len(data) {
			switch data[i+1] {
			case '/':
				inLineComment = true
				out = append(out, ' ', ' ')
				i++
				continue
			case '*':
				inBlockComment = true
				out = append(out, ' ', ' ')
				i++
				continue
			}
		}
		out = append(out, ch)
	}
	return out
}

func stripTrailingCommas(data []byte) []byte {
	out := append([]byte(nil), data...)
	inString := false
	escaped := false

	for i := 0; i < len(out); i++ {
		ch := out[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			switch ch {
			case '\\':
				escaped = true
			case '"':
				inString = false
			}
			continue
		}

		if ch == '"' {
			inString = true
			continue
		}
		if ch != ',' {
			continue
		}

		j := i + 1
		for j < len(out) && isJSONWhitespace(out[j]) {
			j++
		}
		if j < len(out) && (out[j] == '}' || out[j] == ']') {
			out[i] = ' '
		}
	}
	return out
}

func isJSONWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n'
}
