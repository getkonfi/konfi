package pkg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// JSONParser performs surgical edits on JSON config files.
// keys use dotted paths: "permissions.allow" → {"permissions":{"allow":...}}.
// satisfies Parser and MultiValueParser via structural typing.
type JSONParser struct{}

// Capabilities reports JSON format capabilities.
func (p *JSONParser) Capabilities() ParserCapabilities {
	return ParserCapabilities{
		SupportsNesting:    true,
		LosslessRoundtrip:  false, // marshal normalizes whitespace
		SupportsMultivalue: true,
	}
}

// FindValue returns the string representation of the value at a dotted key path.
// arrays and objects are returned as their JSON representation.
func (p *JSONParser) FindValue(data []byte, key string) (string, bool) {
	path := splitDotPath(key)
	return p.FindValueAtPath(data, path)
}

// FindValueAtPath returns the string representation of the value at the given path segments.
func (p *JSONParser) FindValueAtPath(data []byte, path []string) (string, bool) {
	raw, ok := walkPath(data, path)
	if !ok {
		return "", false
	}
	return rawToString(raw), true
}

// FindLine returns the 0-based line number where the final key in a dotted path appears.
// walks the JSON structure to find the correct nesting depth.
func (p *JSONParser) FindLine(data []byte, key string) (int, bool) {
	path := splitDotPath(key)
	return findLineAtPath(data, path)
}

// SetValue sets a value at a dotted key path, preserving the existing JSON type
// when possible. new keys default to string.
func (p *JSONParser) SetValue(data []byte, key, value string) ([]byte, error) {
	path := splitDotPath(key)
	return p.SetValueAtPath(data, path, value)
}

// SetValueAtPath sets a value at the given path segments with type preservation.
func (p *JSONParser) SetValueAtPath(data []byte, path []string, value string) ([]byte, error) {
	root, err := unmarshalOrdered(data)
	if err != nil {
		if len(bytes.TrimSpace(data)) > 0 {
			return nil, fmt.Errorf("parse json: %w", err)
		}
		root = make(orderedMap, 0)
	}

	existing, found := getPath(root, path)
	var typed any
	if found {
		typed = coerceToExistingType(existing, value)
	} else {
		typed = coerceNewValue(value)
	}

	root = setPath(root, path, typed)
	return marshalIndent(root)
}

// DeleteKey removes the value at a dotted key path.
func (p *JSONParser) DeleteKey(data []byte, key string) ([]byte, error) {
	path := splitDotPath(key)
	root, err := unmarshalOrdered(data)
	if err != nil {
		return data, fmt.Errorf("json: %w", err)
	}

	newRoot, ok := deletePath(root, path)
	if !ok {
		return data, fmt.Errorf("key not found: %s", key)
	}
	return marshalIndent(newRoot)
}

// ListKeys returns all leaf keys as dotted paths. arrays are treated as leaves.
func (p *JSONParser) ListKeys(data []byte) []string {
	root, err := unmarshalOrdered(data)
	if err != nil {
		return nil
	}
	var keys []string
	collectKeys(root, nil, &keys)
	return keys
}

// FindAll returns all leaf key-value pairs as a flat map with dotted keys.
func (p *JSONParser) FindAll(data []byte) map[string]string {
	root, err := unmarshalOrdered(data)
	if err != nil {
		return nil
	}
	m := make(map[string]string)
	collectValues(root, nil, m)
	return m
}

// FindValues returns the elements of a JSON array at the given dotted key path as strings.
func (p *JSONParser) FindValues(data []byte, key string) ([]string, bool) {
	path := splitDotPath(key)
	raw, ok := walkPath(data, path)
	if !ok {
		return nil, false
	}

	var arr []json.RawMessage
	if err := json.Unmarshal(raw, &arr); err != nil {
		// not an array — return single value
		return []string{rawToString(raw)}, true
	}

	vals := make([]string, len(arr))
	for i, elem := range arr {
		vals[i] = rawToString(elem)
	}
	return vals, true
}

// SetValues replaces a value at a dotted key path with a JSON array of strings.
func (p *JSONParser) SetValues(data []byte, key string, values []string) ([]byte, error) {
	path := splitDotPath(key)
	root, err := unmarshalOrdered(data)
	if err != nil {
		if len(bytes.TrimSpace(data)) > 0 {
			return nil, fmt.Errorf("parse json: %w", err)
		}
		root = make(orderedMap, 0)
	}
	root = setPath(root, path, values)
	return marshalIndent(root)
}

// --- ordered map to preserve key order ---

type kvPair struct {
	Key   string
	Value any
}

type orderedMap []kvPair

func (m orderedMap) get(key string) (any, bool) {
	for _, kv := range m {
		if kv.Key == key {
			return kv.Value, true
		}
	}
	return nil, false
}

func (m orderedMap) set(key string, value any) orderedMap {
	for i, kv := range m {
		if kv.Key == key {
			m[i].Value = value
			return m
		}
	}
	return append(m, kvPair{Key: key, Value: value})
}

func (m orderedMap) delete(key string) (orderedMap, bool) {
	for i, kv := range m {
		if kv.Key == key {
			return append(m[:i], m[i+1:]...), true
		}
	}
	return m, false
}

// MarshalJSON produces JSON with keys in insertion order.
func (m orderedMap) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, kv := range m {
		if i > 0 {
			buf.WriteByte(',')
		}
		keyBytes, err := json.Marshal(kv.Key)
		if err != nil {
			return nil, err
		}
		buf.Write(keyBytes)
		buf.WriteByte(':')
		valBytes, err := json.Marshal(kv.Value)
		if err != nil {
			return nil, err
		}
		buf.Write(valBytes)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// UnmarshalJSON decodes JSON objects preserving key order.
func (m *orderedMap) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '{' {
		return fmt.Errorf("expected '{', got %v", tok)
	}

	*m = make(orderedMap, 0)
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return err
		}
		key, ok := keyTok.(string)
		if !ok {
			return fmt.Errorf("expected string key, got %T", keyTok)
		}

		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return err
		}

		// try to decode nested object as orderedMap
		val := decodeRawValue(raw)
		*m = append(*m, kvPair{Key: key, Value: val})
	}

	// consume closing '}'
	if _, err := dec.Token(); err != nil {
		return err
	}
	return nil
}

// --- internal helpers ---

func splitDotPath(key string) []string {
	if key == "" {
		return nil
	}
	return strings.Split(key, ".")
}

// decodeRawValue converts raw JSON to an ordered Go value.
// objects become orderedMap, arrays become []any, scalars become native types.
func decodeRawValue(raw json.RawMessage) any {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil
	}

	switch trimmed[0] {
	case '{':
		var om orderedMap
		if err := json.Unmarshal(raw, &om); err == nil {
			return om
		}
	case '[':
		var arr []json.RawMessage
		if err := json.Unmarshal(raw, &arr); err == nil {
			result := make([]any, len(arr))
			for i, elem := range arr {
				result[i] = decodeRawValue(elem)
			}
			return result
		}
	}

	// scalar
	var v any
	if err := json.Unmarshal(raw, &v); err == nil {
		return v
	}
	return string(raw)
}

// unmarshalOrdered parses JSON bytes into an orderedMap.
func unmarshalOrdered(data []byte) (orderedMap, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return make(orderedMap, 0), nil
	}
	var om orderedMap
	if err := json.Unmarshal(data, &om); err != nil {
		return nil, err
	}
	return om, nil
}

// walkPath navigates raw JSON bytes to a dotted path, returning the raw value.
func walkPath(data []byte, path []string) (json.RawMessage, bool) {
	if len(path) == 0 {
		return nil, false
	}

	current := json.RawMessage(bytes.TrimSpace(data))
	for _, seg := range path {
		// decode current level as object
		dec := json.NewDecoder(bytes.NewReader(current))
		tok, err := dec.Token()
		if err != nil {
			return nil, false
		}
		if delim, ok := tok.(json.Delim); !ok || delim != '{' {
			return nil, false
		}

		found := false
		for dec.More() {
			keyTok, err := dec.Token()
			if err != nil {
				return nil, false
			}
			key, ok := keyTok.(string)
			if !ok {
				return nil, false
			}
			var raw json.RawMessage
			if err := dec.Decode(&raw); err != nil {
				return nil, false
			}
			if key == seg {
				current = raw
				found = true
				break
			}
		}
		if !found {
			return nil, false
		}
	}
	return current, true
}

// rawToString converts a raw JSON value to a display string.
func rawToString(raw json.RawMessage) string {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return ""
	}

	// string: unquote
	if trimmed[0] == '"' {
		var s string
		if err := json.Unmarshal(trimmed, &s); err == nil {
			return s
		}
	}

	// object or array: return compact JSON
	if trimmed[0] == '{' || trimmed[0] == '[' {
		var buf bytes.Buffer
		if err := json.Compact(&buf, trimmed); err == nil {
			return buf.String()
		}
	}

	// bool/number: literal text
	return string(trimmed)
}

// findLineAtPath locates the line number of the final key in the path.
// walks nesting depth to avoid matching wrong keys at the wrong level.
func findLineAtPath(data []byte, path []string) (int, bool) {
	if len(path) == 0 {
		return -1, false
	}

	lines := bytes.Split(data, []byte("\n"))
	// depth tracks brace nesting: 0 = outside root, 1 = inside root object, etc.
	// we compute depth BEFORE processing each line's content for key matching,
	// then update depth based on braces found on the line.
	depth := 0
	segIdx := 0 // which segment of path we're looking for

	for i, line := range lines {
		trimmed := bytes.TrimSpace(line)

		// count opening/closing braces on this line, but also check key
		// at the depth before any closing braces on this line affect us.
		// strategy: compute depth at point of key occurrence.
		//
		// for a line like `"permissions": {` at depth=1:
		//   - key "permissions" appears at depth 1 (segIdx=0 needs depth=1)
		//   - then { pushes depth to 2
		//
		// for a line like `"allow": [...]` at depth=2:
		//   - key "allow" appears at depth 2 (segIdx=1 needs depth=2)

		// target depth for current path segment: segIdx + 1
		targetDepth := segIdx + 1

		if segIdx < len(path) && depth == targetDepth {
			target := path[segIdx]
			if keyOnLine(trimmed, target) {
				if segIdx == len(path)-1 {
					return i, true
				}
				segIdx++
			}
		}

		// update depth based on object braces on this line (ignore array brackets)
		inString := false
		escaped := false
		for _, ch := range trimmed {
			if escaped {
				escaped = false
				continue
			}
			switch ch {
			case '\\':
				if inString {
					escaped = true
				}
			case '"':
				inString = !inString
			case '{':
				if !inString {
					depth++
				}
			case '}':
				if !inString {
					depth--
					// if we've exited the scope we were searching in, step back
					if depth < segIdx+1 && segIdx > 0 {
						segIdx = depth - 1
						if segIdx < 0 {
							segIdx = 0
						}
					}
				}
			}
		}
	}
	return -1, false
}

// keyOnLine checks if a line contains "key": (JSON key at expected position).
func keyOnLine(line []byte, key string) bool {
	quoted := `"` + key + `"`
	idx := bytes.Index(line, []byte(quoted))
	if idx < 0 {
		return false
	}
	// check that a colon follows (possibly with whitespace)
	rest := bytes.TrimSpace(line[idx+len(quoted):])
	return len(rest) > 0 && rest[0] == ':'
}

// getPath retrieves a value from the orderedMap tree.
func getPath(root orderedMap, path []string) (any, bool) {
	current := any(root)
	for _, seg := range path {
		m, ok := current.(orderedMap)
		if !ok {
			return nil, false
		}
		val, found := m.get(seg)
		if !found {
			return nil, false
		}
		current = val
	}
	return current, true
}

// setPath sets a value deep in the orderedMap tree, creating intermediate maps.
func setPath(root orderedMap, path []string, value any) orderedMap {
	if len(path) == 0 {
		return root
	}
	if len(path) == 1 {
		return root.set(path[0], value)
	}

	// get or create nested map
	existing, found := root.get(path[0])
	var nested orderedMap
	if found {
		if m, ok := existing.(orderedMap); ok {
			nested = m
		} else {
			nested = make(orderedMap, 0)
		}
	} else {
		nested = make(orderedMap, 0)
	}

	nested = setPath(nested, path[1:], value)
	return root.set(path[0], nested)
}

// deletePath removes a value from the orderedMap tree, returning the new root.
func deletePath(root orderedMap, path []string) (orderedMap, bool) {
	if len(path) == 0 {
		return root, false
	}
	if len(path) == 1 {
		return root.delete(path[0])
	}

	existing, found := root.get(path[0])
	if !found {
		return root, false
	}
	nested, ok := existing.(orderedMap)
	if !ok {
		return root, false
	}
	newNested, deleted := deletePath(nested, path[1:])
	if !deleted {
		return root, false
	}
	return root.set(path[0], newNested), true
}

// coerceToExistingType converts a string value to match the type of existing.
func coerceToExistingType(existing any, value string) any {
	switch existing.(type) {
	case bool:
		if v, err := strconv.ParseBool(value); err == nil {
			return v
		}
	case float64:
		if v, err := strconv.ParseFloat(value, 64); err == nil {
			return v
		}
	case []any:
		// try to parse as JSON array
		var arr []any
		if err := json.Unmarshal([]byte(value), &arr); err == nil {
			return arr
		}
	}
	return value
}

// coerceNewValue infers type for a new key. defaults to string.
func coerceNewValue(value string) any {
	if v, err := strconv.ParseBool(value); err == nil {
		return v
	}
	if v, err := strconv.ParseFloat(value, 64); err == nil {
		return v
	}
	return value
}

// marshalIndent produces pretty-printed JSON with 2-space indent and trailing newline.
func marshalIndent(root orderedMap) ([]byte, error) {
	compact, err := json.Marshal(root)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, compact, "", "  "); err != nil {
		return nil, err
	}
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

// collectKeys gathers all leaf paths as dotted strings.
// arrays are treated as atomic leaves — we don't descend into indices.
func collectKeys(m orderedMap, prefix []string, keys *[]string) {
	for _, kv := range m {
		fullPath := append(append([]string{}, prefix...), kv.Key)
		switch child := kv.Value.(type) {
		case orderedMap:
			collectKeys(child, fullPath, keys)
		default:
			*keys = append(*keys, strings.Join(fullPath, "."))
		}
	}
}

// collectValues gathers all leaf paths and their string values into a map.
func collectValues(m orderedMap, prefix []string, out map[string]string) {
	for _, kv := range m {
		fullPath := append(append([]string{}, prefix...), kv.Key)
		switch child := kv.Value.(type) {
		case orderedMap:
			collectValues(child, fullPath, out)
		default:
			key := strings.Join(fullPath, ".")
			out[key] = anyToString(kv.Value)
		}
	}
}

// anyToString converts a decoded JSON value to its display string.
func anyToString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case bool:
		if val {
			return "true"
		}
		return "false"
	case float64:
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'f', -1, 64)
	case []any:
		b, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprintf("%v", val)
		}
		return string(b)
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", val)
	}
}
