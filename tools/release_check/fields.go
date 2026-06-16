package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/getkonfi/konfi/pkg"
)

var rawGitHubBase = "https://raw.githubusercontent.com"

type FieldStatus string

const (
	FieldStatusCurrent FieldStatus = "current"
	FieldStatusClean   FieldStatus = "clean"
	FieldStatusAdded   FieldStatus = "added"
	FieldStatusAhead   FieldStatus = "ahead"
	FieldStatusSkipped FieldStatus = "skipped"
	FieldStatusError   FieldStatus = "error"
)

type FieldResult struct {
	App       string      `json:"app"`
	Supported string      `json:"supported,omitempty"`
	Latest    string      `json:"latest,omitempty"`
	Status    FieldStatus `json:"status"`
	Added     []string    `json:"added,omitempty"`
	Detail    string      `json:"detail,omitempty"`
}

type FieldReport struct {
	Results []FieldResult `json:"results"`
}

func (r *FieldReport) HasProblem() bool {
	for _, a := range r.Results {
		if a.Status == FieldStatusError || a.Status == FieldStatusAdded {
			return true
		}
	}
	return false
}

func (r *FieldReport) WriteText(w io.Writer) {
	const format = "%-14s %-12s %-12s %-8s  %s\n"
	fmt.Fprintf(w, format, "APP", "SUPPORTED", "LATEST", "STATUS", "ADDED")
	fmt.Fprintln(w, strings.Repeat("─", 82))

	var added, errors int
	for _, a := range r.Results {
		detail := strings.Join(a.Added, ", ")
		if detail == "" {
			detail = a.Detail
		}
		fmt.Fprintf(w, format, a.App, dash(a.Supported), dash(a.Latest), a.Status, detail)

		switch a.Status {
		case FieldStatusAdded:
			added++
		case FieldStatusError:
			errors++
		}
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "%d total, %d with added fields, %d error\n", len(r.Results), added, errors)
}

func (r *FieldReport) WriteJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

func runFieldChecks(ctx context.Context, client *http.Client, cfg *upstreamConfig, schemaPaths []string, includeSkipped bool) *FieldReport {
	sem := make(chan struct{}, defaultConcurrency)
	var wg sync.WaitGroup
	results := make([]FieldResult, len(schemaPaths))

	for i, path := range schemaPaths {
		wg.Add(1)
		go func(i int, path string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			results[i] = checkFieldsOne(ctx, client, cfg, path)
		}(i, path)
	}
	wg.Wait()

	out := &FieldReport{}
	for _, r := range results {
		if r.Status == FieldStatusSkipped && !includeSkipped {
			continue
		}
		out.Results = append(out.Results, r)
	}
	return out
}

func checkFieldsOne(ctx context.Context, client *http.Client, cfg *upstreamConfig, path string) FieldResult {
	appName := filepath.Base(filepath.Dir(path))
	res := FieldResult{App: appName}

	data, err := os.ReadFile(path)
	if err != nil {
		res.Status = FieldStatusError
		res.Detail = fmt.Sprintf("read schema: %v", err)
		return res
	}
	schema, err := pkg.LoadSchema(data)
	if err != nil {
		res.Status = FieldStatusError
		res.Detail = fmt.Sprintf("parse schema: %v", err)
		return res
	}

	res.Supported = schema.MaxAppVersion
	if schema.Upstream == nil || schema.Upstream.Kind == "none" || schema.Upstream.Kind == "" {
		res.Status = FieldStatusSkipped
		res.Detail = "no upstream configured"
		return res
	}
	if schema.MaxAppVersion == "" {
		res.Status = FieldStatusError
		res.Detail = "schema has no max_app_version"
		return res
	}

	info, err := fetchLatest(ctx, client, cfg, schema.Upstream)
	if err != nil {
		res.Status = FieldStatusError
		res.Detail = err.Error()
		return res
	}
	res.Latest = strings.TrimPrefix(info.Tag, schema.Upstream.TagPrefix)

	version := AppResult{App: appName, Supported: schema.MaxAppVersion}
	classify(&version, info, schema.Upstream.TagPrefix)
	switch version.Status {
	case StatusCurrent:
		res.Status = FieldStatusCurrent
		return res
	case StatusAhead:
		res.Status = FieldStatusAhead
		return res
	case StatusError:
		res.Status = FieldStatusError
		res.Detail = version.Detail
		return res
	}

	if schema.Upstream.Kind != "github" {
		res.Status = FieldStatusError
		res.Detail = "field diff supports github upstreams only"
		return res
	}

	extractor := fieldExtractorFor(appName)
	if extractor == nil {
		res.Status = FieldStatusError
		res.Detail = "no field extractor for app"
		return res
	}

	fromTag := schema.Upstream.TagPrefix + strings.TrimPrefix(schema.MaxAppVersion, schema.Upstream.TagPrefix)
	from, err := extractor.keys(ctx, client, schema.Upstream.Repo, fromTag, githubToken(cfg))
	if err != nil {
		res.Status = FieldStatusError
		res.Detail = fmt.Sprintf("fetch supported fields: %v", err)
		return res
	}
	to, err := extractor.keys(ctx, client, schema.Upstream.Repo, info.Tag, githubToken(cfg))
	if err != nil {
		res.Status = FieldStatusError
		res.Detail = fmt.Sprintf("fetch latest fields: %v", err)
		return res
	}

	res.Added = setDiff(to, from)
	if len(res.Added) > 0 {
		res.Status = FieldStatusAdded
		return res
	}
	res.Status = FieldStatusClean
	return res
}

type fieldExtractor struct {
	paths []string
	parse func(path string, data []byte) []string
}

func (e *fieldExtractor) keys(ctx context.Context, client *http.Client, repo, tag, token string) ([]string, error) {
	seen := make(map[string]struct{})
	for _, path := range e.paths {
		data, err := fetchGitHubRaw(ctx, client, repo, tag, path, token)
		if err != nil {
			return nil, err
		}
		for _, key := range e.parse(path, data) {
			if key == "" {
				continue
			}
			seen[key] = struct{}{}
		}
	}
	return sortedKeys(seen), nil
}

func fieldExtractorFor(app string) *fieldExtractor {
	switch app {
	case "hyprland":
		return &fieldExtractor{
			paths: []string{"src/config/values/ConfigValues.cpp"},
			parse: parseHyprlandFields,
		}
	case "kitty":
		return &fieldExtractor{
			paths: []string{"kitty/options/definition.py"},
			parse: parseKittyFields,
		}
	case "rio":
		return &fieldExtractor{
			paths: []string{
				"rio-backend/src/config/mod.rs",
				"rio-backend/src/config/bell.rs",
				"rio-backend/src/config/bindings.rs",
				"rio-backend/src/config/colors/defaults.rs",
				"rio-backend/src/config/colors/mod.rs",
				"rio-backend/src/config/colors/term.rs",
				"rio-backend/src/config/effects.rs",
				"rio-backend/src/config/hints.rs",
				"rio-backend/src/config/keyboard.rs",
				"rio-backend/src/config/layout.rs",
				"rio-backend/src/config/navigation.rs",
				"rio-backend/src/config/platform.rs",
				"rio-backend/src/config/renderer.rs",
				"rio-backend/src/config/theme.rs",
				"rio-backend/src/config/title.rs",
				"rio-backend/src/config/window.rs",
			},
			parse: parseRioFields,
		}
	default:
		return nil
	}
}

func fetchGitHubRaw(ctx context.Context, client *http.Client, repo, tag, path, token string) ([]byte, error) {
	url := fmt.Sprintf("%s/%s/%s/%s", rawGitHubBase, repo, tag, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: http %d", path, resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%s: read body: %w", path, err)
	}
	return data, nil
}

var (
	hyprlandFieldRE = regexp.MustCompile(`MS<[^>]+>\("([^"]+)"`)
	kittyFieldRE    = regexp.MustCompile(`(?m)^\s*opt\('([^']+)'`)
	rustFieldRE     = regexp.MustCompile(`^\s*pub\s+([A-Za-z_][A-Za-z0-9_]*)\s*:`)
	serdeRenameRE   = regexp.MustCompile(`rename\s*=\s*"([^"]+)"`)
)

func parseHyprlandFields(_ string, data []byte) []string {
	matches := hyprlandFieldRE.FindAllSubmatch(data, -1)
	keys := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		key := strings.ReplaceAll(string(match[1]), ":", ".")
		keys[key] = struct{}{}
	}
	return sortedKeys(keys)
}

func parseKittyFields(_ string, data []byte) []string {
	matches := kittyFieldRE.FindAllSubmatch(data, -1)
	keys := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		keys[string(match[1])] = struct{}{}
	}
	return sortedKeys(keys)
}

func parseRioFields(path string, data []byte) []string {
	prefix := rioFieldPrefix(path)
	keys := make(map[string]struct{})
	scanner := bufio.NewScanner(bytes.NewReader(data))
	var attr strings.Builder
	inAttr := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#[serde") {
			inAttr = true
			attr.Reset()
		}
		if inAttr {
			attr.WriteString(line)
			attr.WriteByte(' ')
			if strings.Contains(line, ")]") {
				inAttr = false
			}
			continue
		}

		match := rustFieldRE.FindStringSubmatch(line)
		if len(match) == 0 {
			if line != "" && !strings.HasPrefix(line, "#[") {
				attr.Reset()
			}
			continue
		}

		key := match[1]
		if rename := serdeRenameRE.FindStringSubmatch(attr.String()); len(rename) > 0 {
			key = rename[1]
		}
		attr.Reset()
		keys[prefix+"."+key] = struct{}{}
	}
	return sortedKeys(keys)
}

func rioFieldPrefix(path string) string {
	rel := strings.TrimPrefix(path, "rio-backend/src/config/")
	rel = strings.TrimSuffix(rel, ".rs")
	rel = strings.TrimSuffix(rel, "/mod")
	rel = strings.ReplaceAll(rel, "/", ".")
	if rel == "" || rel == "mod" {
		return "config"
	}
	return rel
}

func setDiff(a, b []string) []string {
	bSet := make(map[string]struct{}, len(b))
	for _, key := range b {
		bSet[key] = struct{}{}
	}
	var out []string
	for _, key := range a {
		if _, ok := bSet[key]; !ok {
			out = append(out, key)
		}
	}
	sort.Strings(out)
	return out
}

func sortedKeys(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
