package pkg

import (
	"fmt"
	"slices"

	"golang.org/x/mod/semver"
	"gopkg.in/yaml.v3"
)

// Schema describes the configurable fields of an application.
type Schema struct {
	App           string    `yaml:"app"`
	Format        string    `yaml:"format"`
	SchemaVersion string    `yaml:"schema_version,omitempty"`
	MinAppVersion string    `yaml:"min_app_version,omitempty"`
	MaxAppVersion string    `yaml:"max_app_version,omitempty"`
	DocsURL       string    `yaml:"docs_url,omitempty"`
	Hints         []string  `yaml:"hints,omitempty"`
	Sections      []Section `yaml:"sections"`
}

// Section groups related config fields.
type Section struct {
	Name   string  `yaml:"name"`
	Key    string  `yaml:"key"`
	Fields []Field `yaml:"fields"`
}

// Field describes a single config entry.
type Field struct {
	Key         string   `yaml:"key"`
	Label       string   `yaml:"label"`
	Type        string   `yaml:"type"`
	Widget      string   `yaml:"widget,omitempty"` // ui hint: "font", "slider", "path"
	Default     string   `yaml:"default"`
	Description string   `yaml:"description"`
	Options     []string `yaml:"options,omitempty"`
	AltOptions  []string `yaml:"alt_options,omitempty"`
	Min         *float64 `yaml:"min,omitempty"`
	Max         *float64 `yaml:"max,omitempty"`
	Palette     []string `yaml:"palette,omitempty"`
	Example     string   `yaml:"example,omitempty"`
	Hint        string   `yaml:"hint,omitempty"`
	DocURL      string   `yaml:"doc_url,omitempty"`
	Since       string   `yaml:"since,omitempty"`
	Until       string   `yaml:"until,omitempty"`
}

// FilterByVersion returns a new schema containing only fields compatible with v.
// empty v returns an unfiltered copy. invalid semver in since/until is treated as unset.
// sections that become empty after filtering are dropped.
func (s *Schema) FilterByVersion(v string) *Schema {
	out := &Schema{
		App:           s.App,
		Format:        s.Format,
		SchemaVersion: s.SchemaVersion,
		MinAppVersion: s.MinAppVersion,
		MaxAppVersion: s.MaxAppVersion,
		DocsURL:       s.DocsURL,
		Hints:         slices.Clone(s.Hints),
	}

	nv := NormalizeSemver(v)
	if nv == "" {
		// unknown version — copy all sections as-is
		out.Sections = make([]Section, len(s.Sections))
		copy(out.Sections, s.Sections)
		return out
	}

	for _, sec := range s.Sections {
		var filtered []Field
		for fi := range sec.Fields {
			if !fieldVisibleAt(sec.Fields[fi], nv) {
				continue
			}
			filtered = append(filtered, sec.Fields[fi])
		}
		if len(filtered) == 0 {
			continue
		}
		out.Sections = append(out.Sections, Section{
			Name:   sec.Name,
			Key:    sec.Key,
			Fields: filtered,
		})
	}
	return out
}

// CompatibleWith checks if appVersion falls within [MinAppVersion, MaxAppVersion].
// returns ("", true) when compatible or when bounds are unset.
// returns (reason, false) when the version is outside the declared range.
func (s *Schema) CompatibleWith(appVersion string) (string, bool) {
	nv := NormalizeSemver(appVersion)
	if nv == "" {
		return "", true
	}
	if min := NormalizeSemver(s.MinAppVersion); min != "" {
		if semver.Compare(nv, min) < 0 {
			return fmt.Sprintf("schema requires %s %s+, detected %s", s.App, s.MinAppVersion, appVersion), false
		}
	}
	if max := NormalizeSemver(s.MaxAppVersion); max != "" {
		if semver.Compare(nv, max) > 0 {
			return fmt.Sprintf("schema covers %s up to %s, detected %s", s.App, s.MaxAppVersion, appVersion), false
		}
	}
	return "", true
}

// fieldVisibleAt checks if a field is visible at the given normalized semver.
func fieldVisibleAt(f Field, v string) bool {
	if since := NormalizeSemver(f.Since); since != "" {
		if semver.Compare(v, since) < 0 {
			return false
		}
	}
	if until := NormalizeSemver(f.Until); until != "" {
		if semver.Compare(v, until) >= 0 {
			return false
		}
	}
	return true
}

// SchemaKeys returns the set of all field keys across all sections.
func (s *Schema) SchemaKeys() map[string]struct{} {
	keys := make(map[string]struct{})
	for si := range s.Sections {
		for fi := range s.Sections[si].Fields {
			keys[s.Sections[si].Fields[fi].Key] = struct{}{}
		}
	}
	return keys
}

// Diagnostic describes a config issue (unknown key or deprecated field).
type Diagnostic struct {
	Key     string
	Kind    string // "unknown" or "deprecated"
	Message string
}

// Diagnose compares config keys against a schema and returns diagnostics.
// unknown: key not in schema. deprecated: field has Until and version >= Until.
func Diagnose(configKeys []string, schema *Schema, appVersion string) []Diagnostic {
	known := schema.SchemaKeys()
	nv := NormalizeSemver(appVersion)

	// build deprecated set
	deprecated := make(map[string]string) // key → Until
	for si := range schema.Sections {
		for fi := range schema.Sections[si].Fields {
			if schema.Sections[si].Fields[fi].Until != "" {
				deprecated[schema.Sections[si].Fields[fi].Key] = schema.Sections[si].Fields[fi].Until
			}
		}
	}

	seen := make(map[string]bool)
	var diags []Diagnostic
	for _, key := range configKeys {
		if seen[key] {
			continue
		}
		seen[key] = true

		if _, ok := known[key]; !ok {
			diags = append(diags, Diagnostic{
				Key:     key,
				Kind:    "unknown",
				Message: fmt.Sprintf("unknown key %q not in schema", key),
			})
			continue
		}

		if until, ok := deprecated[key]; ok && nv != "" {
			nu := NormalizeSemver(until)
			if nu != "" && semver.Compare(nv, nu) >= 0 {
				diags = append(diags, Diagnostic{
					Key:     key,
					Kind:    "deprecated",
					Message: fmt.Sprintf("key %q deprecated since %s", key, until),
				})
			}
		}
	}
	return diags
}

// LoadSchema parses a YAML schema definition.
func LoadSchema(data []byte) (*Schema, error) {
	var s Schema
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse schema: %w", err)
	}
	return &s, nil
}
