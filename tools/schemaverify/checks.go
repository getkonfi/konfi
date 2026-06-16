package main

import (
	"fmt"
	"os"

	"github.com/getkonfi/konfi/pkg"
)

var validTypes = map[string]bool{
	"string": true,
	"number": true,
	"bool":   true,
	"enum":   true,
	"color":  true,
	"list":   true,
	"multi":  true,
	"path":   true,
}

var validWidgets = map[string]bool{
	"font":        true,
	"slider":      true,
	"path":        true,
	"stylestring": true,
	"hook":        true,
	"structlist":  true,
	"blocklist":   true,
	"patternlist": true,
	"togglemap":   true,
	"rawtoml":     true,
}

// checkStructural runs offline structural validation on a single schema file.
func checkStructural(path string) AppReport {
	data, err := os.ReadFile(path)
	if err != nil {
		return AppReport{
			App:      path,
			Findings: []Finding{{Fail, "structural", fmt.Sprintf("read error: %v", err)}},
		}
	}

	schema, err := pkg.LoadSchema(data)
	if err != nil {
		return AppReport{
			App:      path,
			Findings: []Finding{{Fail, "structural", fmt.Sprintf("parse error: %v", err)}},
		}
	}

	report := AppReport{App: schema.App}
	if report.App == "" {
		report.App = path
	}

	// top-level required fields
	if schema.App == "" {
		report.Findings = append(report.Findings, Finding{Fail, "structural", "missing top-level 'app'"})
	}
	if schema.Format == "" {
		report.Findings = append(report.Findings, Finding{Fail, "structural", "missing top-level 'format'"})
	}
	if len(schema.Sections) == 0 {
		report.Findings = append(report.Findings, Finding{Fail, "structural", "no sections defined"})
	}
	// goal-1 quality bar: every schema should point at upstream docs.
	// warn-level (passes by default; --strict turns it into a failure).
	if schema.DocsURL == "" {
		report.Findings = append(report.Findings, Finding{
			Warn, "structural", "missing top-level 'docs_url'",
		})
	}

	// validate semver fields
	for _, pair := range []struct{ name, val string }{
		{"schema_version", schema.SchemaVersion},
		{"min_app_version", schema.MinAppVersion},
		{"max_app_version", schema.MaxAppVersion},
		{"format_since", schema.FormatSince},
	} {
		if pair.val != "" && pkg.NormalizeSemver(pair.val) == "" {
			report.Findings = append(report.Findings, Finding{
				Fail, "structural", fmt.Sprintf("invalid semver in %s: %q", pair.name, pair.val),
			})
		}
	}

	seenKeys := make(map[string]bool)

	for si, sec := range schema.Sections {
		if sec.Name == "" {
			report.Findings = append(report.Findings, Finding{
				Fail, "structural", fmt.Sprintf("section[%d]: empty name", si),
			})
		}

		for fi := range sec.Fields {
			f := &sec.Fields[fi]
			loc := fmt.Sprintf("section %q field[%d]", sec.Name, fi)

			// required field attributes
			if f.Key == "" {
				report.Findings = append(report.Findings, Finding{
					Fail, "structural", fmt.Sprintf("%s: missing key", loc),
				})
			}
			if f.Label == "" {
				report.Findings = append(report.Findings, Finding{
					Fail, "structural", fmt.Sprintf("%s (%s): missing label", loc, f.Key),
				})
			}
			if f.Type == "" {
				report.Findings = append(report.Findings, Finding{
					Fail, "structural", fmt.Sprintf("%s (%s): missing type", loc, f.Key),
				})
			} else if !validTypes[f.Type] {
				report.Findings = append(report.Findings, Finding{
					Fail, "structural", fmt.Sprintf("%s (%s): invalid type %q", loc, f.Key, f.Type),
				})
			}

			// widget validation
			if f.Widget != "" && !validWidgets[f.Widget] {
				report.Findings = append(report.Findings, Finding{
					Fail, "structural", fmt.Sprintf("%s (%s): invalid widget %q", loc, f.Key, f.Widget),
				})
			}

			// enum requires options
			if f.Type == "enum" && len(f.Options) == 0 {
				report.Findings = append(report.Findings, Finding{
					Fail, "structural", fmt.Sprintf("%s (%s): enum type without options", loc, f.Key),
				})
			}

			// goal-1 quality bar: every field must explain itself.
			// warn-level (passes by default; --strict turns it into a failure).
			if f.Description == "" {
				report.Findings = append(report.Findings, Finding{
					Warn, "structural", fmt.Sprintf("%s (%s): missing description", loc, f.Key),
				})
			}

			// number without min/max is a warn
			if f.Type == "number" {
				if f.Min == nil || f.Max == nil {
					report.Findings = append(report.Findings, Finding{
						Warn, "structural", fmt.Sprintf("%s (%s): number missing min/max", loc, f.Key),
					})
				}
			}

			// duplicate key check
			if f.Key != "" {
				if seenKeys[f.Key] {
					report.Findings = append(report.Findings, Finding{
						Fail, "structural", fmt.Sprintf("%s: duplicate key %q", loc, f.Key),
					})
				}
				seenKeys[f.Key] = true
			}

			// field-level semver
			for _, pair := range []struct{ name, val string }{
				{"since", f.Since},
				{"until", f.Until},
			} {
				if pair.val != "" && pkg.NormalizeSemver(pair.val) == "" {
					report.Findings = append(report.Findings, Finding{
						Fail, "structural", fmt.Sprintf("%s (%s): invalid semver in %s: %q", loc, f.Key, pair.name, pair.val),
					})
				}
			}
		}
	}

	if len(report.Findings) == 0 {
		report.Findings = append(report.Findings, Finding{Pass, "structural", "all structural checks passed"})
	}

	return report
}
