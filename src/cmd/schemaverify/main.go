package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/emin/konfigurator/pkg"
)

func main() {
	var (
		app     string
		offline bool
		noExec  bool
		asJSON  bool
		strict  bool
		verbose bool
		color   string
	)

	flag.StringVar(&app, "app", "", "verify only this app (repeatable via multiple --app flags)")
	flag.BoolVar(&offline, "offline", false, "skip network checks")
	flag.BoolVar(&noExec, "no-exec", false, "skip exec-based checks")
	flag.BoolVar(&asJSON, "json", false, "json output")
	flag.BoolVar(&strict, "strict", false, "treat warnings as failures")
	flag.BoolVar(&verbose, "v", false, "verbose output")
	flag.StringVar(&color, "color", "auto", "color output: auto, always, never")
	flag.Parse()

	schemas, err := discoverSchemas(app)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}

	if len(schemas) == 0 {
		fmt.Fprintf(os.Stderr, "no schemas found\n")
		os.Exit(2)
	}

	ctx := context.Background()
	var report Report
	var text *textReportWriter

	if !asJSON {
		useColor, err := resolveColor(color, os.Stdout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(2)
		}
		text = newTextReportWriter(os.Stdout, verbose, useColor)
		text.WriteHeader()
	}

	for _, path := range schemas {
		ar := checkStructural(path)
		schema := loadSchemaQuiet(path)

		// phase 2: url verification
		if !offline && schema != nil {
			ar.Findings = append(ar.Findings, checkURLs(ctx, schema, defaultURLConcurrency)...)
		}

		// phase 3: observed config introspection
		if !noExec && schema != nil {
			ar.Findings = append(ar.Findings, checkDump(ctx, schema)...)
		}

		report.Apps = append(report.Apps, ar)
		if text != nil {
			text.WriteApp(ar)
		}
	}

	if asJSON {
		if err := report.WriteJSON(os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "json error: %v\n", err)
			os.Exit(2)
		}
	} else {
		text.WriteSummary(report)
	}

	if report.HasFail() {
		os.Exit(1)
	}
	if strict && report.HasWarn() {
		os.Exit(1)
	}
}

func resolveColor(mode string, out *os.File) (bool, error) {
	switch strings.ToLower(mode) {
	case "always":
		return true, nil
	case "never":
		return false, nil
	case "auto":
		return shouldColor(out), nil
	default:
		return false, fmt.Errorf("invalid --color value %q, want auto, always, or never", mode)
	}
}

func shouldColor(out *os.File) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if os.Getenv("CLICOLOR_FORCE") != "" && os.Getenv("CLICOLOR_FORCE") != "0" {
		return true
	}
	if os.Getenv("CLICOLOR") == "0" || os.Getenv("TERM") == "dumb" {
		return false
	}
	info, err := out.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// loadSchemaQuiet loads and parses a schema file, returning nil on error.
func loadSchemaQuiet(path string) *pkg.Schema {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	s, err := pkg.LoadSchema(data)
	if err != nil {
		return nil
	}
	return s
}

func discoverSchemas(appFilter string) ([]string, error) {
	pattern := filepath.Join("konfables", "*", "schema.yaml")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob: %w", err)
	}

	if appFilter != "" {
		specific := filepath.Join("konfables", appFilter, "schema.yaml")
		if slices.Contains(matches, specific) {
			return []string{specific}, nil
		}
		return nil, fmt.Errorf("no schema found for app %q", appFilter)
	}

	sort.Strings(matches)
	return matches, nil
}
