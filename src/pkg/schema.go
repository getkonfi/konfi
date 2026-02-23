package pkg

import (
	"fmt"
	"slices"

	"golang.org/x/mod/semver"
	"gopkg.in/yaml.v3"
)

// Schema describes the configurable fields of an application.
type Schema struct {
	App      string    `yaml:"app"`
	Format   string    `yaml:"format"`
	DocsURL  string    `yaml:"docs_url,omitempty"`
	Hints    []string  `yaml:"hints,omitempty"`
	Sections []Section `yaml:"sections"`
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
	Default     string   `yaml:"default"`
	Description string   `yaml:"description"`
	Options     []string `yaml:"options,omitempty"`
	Min         *float64 `yaml:"min,omitempty"`
	Max         *float64 `yaml:"max,omitempty"`
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
		App:     s.App,
		Format:  s.Format,
		DocsURL: s.DocsURL,
		Hints:   slices.Clone(s.Hints),
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

// LoadSchema parses a YAML schema definition.
func LoadSchema(data []byte) (*Schema, error) {
	var s Schema
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse schema: %w", err)
	}
	return &s, nil
}
