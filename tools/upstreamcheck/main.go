package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/eminert/konfi/pkg"
)

const (
	defaultTimeout     = 15 * time.Second
	defaultConcurrency = 6
)

func main() {
	var (
		app     string
		asJSON  bool
		verbose bool
		timeout time.Duration
		quiet   bool
	)

	flag.StringVar(&app, "app", "", "check only this app")
	flag.BoolVar(&asJSON, "json", false, "json output")
	flag.BoolVar(&verbose, "v", false, "verbose: include skipped apps in text output")
	flag.BoolVar(&quiet, "quiet", false, "suppress config-source log")
	flag.DurationVar(&timeout, "timeout", defaultTimeout, "per-request timeout")
	flag.Parse()

	cfg, loaded, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(2)
	}
	if !quiet && !asJSON {
		if len(loaded) == 0 {
			fmt.Fprintln(os.Stderr, "no config found — proceeding unauthenticated (github 60 req/hr cap applies)")
		} else {
			fmt.Fprintf(os.Stderr, "config: %v\n", loaded)
		}
	}

	schemas, err := discoverSchemas(app)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}
	if len(schemas) == 0 {
		fmt.Fprintln(os.Stderr, "no schemas found")
		os.Exit(2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout*time.Duration(len(schemas)))

	client := &http.Client{Timeout: timeout}
	report := runChecks(ctx, client, cfg, schemas, verbose)
	cancel()

	if asJSON {
		if err := report.WriteJSON(os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "json: %v\n", err)
			os.Exit(2)
		}
	} else {
		report.WriteText(os.Stdout)
	}

	if report.HasError() {
		os.Exit(1)
	}
}

func discoverSchemas(appFilter string) ([]string, error) {
	root := schemaRoot()
	pattern := filepath.Join(root, "*", "schema.yaml")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob: %w", err)
	}
	if appFilter != "" {
		specific := filepath.Join(root, appFilter, "schema.yaml")
		if slices.Contains(matches, specific) {
			return []string{specific}, nil
		}
		return nil, fmt.Errorf("no schema for %q", appFilter)
	}
	sort.Strings(matches)
	return matches, nil
}

func schemaRoot() string {
	for _, dir := range []string{
		"konfables",
		filepath.Join("src", "konfables"),
		filepath.Join("..", "src", "konfables"),
		filepath.Join("..", "..", "src", "konfables"),
	} {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}
	return "konfables"
}

// runChecks loads each schema, dispatches fetchers concurrently, and
// assembles a report. schemas without an upstream block are marked skipped.
func runChecks(ctx context.Context, client *http.Client, cfg *upstreamConfig, schemaPaths []string, includeSkipped bool) *Report {
	sem := make(chan struct{}, defaultConcurrency)
	var wg sync.WaitGroup
	results := make([]AppResult, len(schemaPaths))

	for i, path := range schemaPaths {
		wg.Add(1)
		go func(i int, path string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			results[i] = checkOne(ctx, client, cfg, path)
		}(i, path)
	}
	wg.Wait()

	out := &Report{}
	for _, r := range results {
		if r.Status == StatusSkipped && !includeSkipped {
			continue
		}
		out.Results = append(out.Results, r)
	}
	return out
}

func checkOne(ctx context.Context, client *http.Client, cfg *upstreamConfig, path string) AppResult {
	appName := filepath.Base(filepath.Dir(path))
	res := AppResult{App: appName}

	data, err := os.ReadFile(path)
	if err != nil {
		res.Status = StatusError
		res.Detail = fmt.Sprintf("read schema: %v", err)
		return res
	}
	schema, err := pkg.LoadSchema(data)
	if err != nil {
		res.Status = StatusError
		res.Detail = fmt.Sprintf("parse schema: %v", err)
		return res
	}

	res.Supported = schema.MaxAppVersion
	if schema.Upstream == nil || schema.Upstream.Kind == "none" || schema.Upstream.Kind == "" {
		res.Status = StatusSkipped
		res.Detail = "no upstream configured"
		return res
	}

	info, err := fetchLatest(ctx, client, cfg, schema.Upstream)
	if err != nil {
		res.Status = StatusError
		res.Detail = err.Error()
		return res
	}

	classify(&res, info, schema.Upstream.TagPrefix)
	return res
}

func fetchLatest(ctx context.Context, client *http.Client, cfg *upstreamConfig, up *pkg.Upstream) (*ReleaseInfo, error) {
	switch up.Kind {
	case "github":
		return fetchGitHubLatest(ctx, client, up, githubToken(cfg))
	case "gitlab":
		return fetchGitLabLatest(ctx, client, up, gitlabTokenFor(cfg, up.Host))
	default:
		return nil, fmt.Errorf("unknown upstream kind %q", up.Kind)
	}
}
