package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/emin/konfigurator/pkg"

	"gopkg.in/yaml.v3"
)

// Generator produces a schema for a single app.
type Generator interface {
	Generate(ctx context.Context) (*pkg.Schema, error)
}

var registry = map[string]Generator{
	"ghostty": &ghosttyGenerator{},
}

func main() {
	app := flag.String("app", "", "generate schema for a specific app")
	check := flag.Bool("check", false, "compare generated schemas against committed files")
	flag.Parse()

	ctx := context.Background()

	targets := registry
	if *app != "" {
		g, ok := registry[*app]
		if !ok {
			fmt.Fprintf(os.Stderr, "unknown app: %s\n", *app)
			os.Exit(1)
		}
		targets = map[string]Generator{*app: g}
	}

	var failed bool
	for name, gen := range targets {
		schema, err := gen.Generate(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: generate failed: %v\n", name, err)
			failed = true
			continue
		}

		data, err := marshalSchema(schema)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: marshal failed: %v\n", name, err)
			failed = true
			continue
		}

		dest := schemaPath(name)

		if *check {
			existing, err := os.ReadFile(dest)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: cannot read committed schema: %v\n", name, err)
				failed = true
				continue
			}
			if !bytes.Equal(data, existing) {
				fmt.Fprintf(os.Stderr, "%s: schema drift detected (run 'make schema-gen' to update)\n", name)
				failed = true
				continue
			}
			fmt.Printf("%s: ok\n", name)
			continue
		}

		if err := os.WriteFile(dest, data, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "%s: write failed: %v\n", name, err)
			failed = true
			continue
		}
		fmt.Printf("%s: wrote %s\n", name, dest)
	}

	if failed {
		os.Exit(1)
	}
}

// schemaPath returns the path to the schema.yaml for a given app.
func schemaPath(name string) string {
	return filepath.Join("konfables", name, "schema.yaml")
}

func marshalSchema(s *pkg.Schema) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(s); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
