package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/emin/konfigurator/pkg"
)

const (
	defaultURLConcurrency = 10
	urlRequestTimeout     = 10 * time.Second
)

type urlResult struct {
	url     string
	status  int
	err     error
	location string // redirect target if 3xx
}

// checkURLs validates all doc_url and docs_url references in a loaded schema.
func checkURLs(ctx context.Context, schema *pkg.Schema, concurrency int) []Finding {
	if concurrency <= 0 {
		concurrency = defaultURLConcurrency
	}

	urls := collectURLs(schema)
	if len(urls) == 0 {
		return nil
	}

	// deduplicate by base url (fragment stripped)
	type entry struct {
		base     string
		original string
	}
	seen := make(map[string]bool)
	var deduped []entry
	for _, u := range urls {
		base := stripFragment(u)
		if seen[base] {
			continue
		}
		seen[base] = true
		deduped = append(deduped, entry{base: base, original: u})
	}

	sem := make(chan struct{}, concurrency)
	var mu sync.Mutex
	var findings []Finding

	client := &http.Client{
		Timeout: urlRequestTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// don't follow redirects — we want to capture the 3xx
			return http.ErrUseLastResponse
		},
	}

	var wg sync.WaitGroup
	for _, e := range deduped {
		wg.Add(1)
		go func(e entry) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			r := headURL(ctx, client, e.base)

			mu.Lock()
			defer mu.Unlock()

			switch {
			case r.err != nil:
				findings = append(findings, Finding{Warn, "url", fmt.Sprintf("%s — %v", e.base, r.err)})
			case r.status >= 200 && r.status < 300:
				// pass — no finding needed
			case r.status >= 300 && r.status < 400:
				findings = append(findings, Finding{Info, "url",
					fmt.Sprintf("%s → %d redirect to %s", e.base, r.status, r.location)})
			default:
				findings = append(findings, Finding{Warn, "url",
					fmt.Sprintf("%s → %d", e.base, r.status)})
			}
		}(e)
	}
	wg.Wait()

	if len(findings) == 0 {
		findings = append(findings, Finding{Pass, "url",
			fmt.Sprintf("all %d urls reachable", len(deduped))})
	}
	return findings
}

// headURL performs a HEAD request with one retry on timeout.
func headURL(ctx context.Context, client *http.Client, rawURL string) urlResult {
	r := doHead(ctx, client, rawURL)
	if r.err != nil && isTimeout(r.err) {
		r = doHead(ctx, client, rawURL)
	}
	return r
}

func doHead(ctx context.Context, client *http.Client, rawURL string) urlResult {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, rawURL, http.NoBody)
	if err != nil {
		return urlResult{url: rawURL, err: err}
	}
	req.Header.Set("User-Agent", "konfi-schemaverify/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return urlResult{url: rawURL, err: err}
	}
	resp.Body.Close()

	loc := ""
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		loc = resp.Header.Get("Location")
	}
	return urlResult{url: rawURL, status: resp.StatusCode, location: loc}
}

func isTimeout(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	return errors.Is(err, context.DeadlineExceeded)
}

// collectURLs gathers all doc_url and docs_url values from a schema.
func collectURLs(schema *pkg.Schema) []string {
	seen := make(map[string]bool)
	var urls []string
	add := func(u string) {
		if u == "" || seen[u] {
			return
		}
		seen[u] = true
		urls = append(urls, u)
	}
	add(schema.DocsURL)
	for si := range schema.Sections {
		for fi := range schema.Sections[si].Fields {
			add(schema.Sections[si].Fields[fi].DocURL)
		}
	}
	return urls
}

func stripFragment(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	u.Fragment = ""
	return u.String()
}
