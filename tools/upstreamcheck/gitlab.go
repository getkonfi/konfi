package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/eminert/konfi/pkg"
)

type glRelease struct {
	TagName string `json:"tag_name"`
	Links   struct {
		Self string `json:"self"`
	} `json:"_links"`
}

// fetchGitLabLatest queries the releases list endpoint for the first
// (most recent) release. gitlab's api has no /latest shortcut — sorting
// is newest-first by default.
//
// host is required because self-hosted instances vary (gitlab.com,
// gitlab.archlinux.org, gitlab.gnome.org, etc). repo is the full project
// path ("group/subgroup/name") which we url-encode for the api path.
func fetchGitLabLatest(ctx context.Context, client *http.Client, up *pkg.Upstream, token string) (*ReleaseInfo, error) {
	if up.Host == "" {
		return nil, fmt.Errorf("gitlab upstream missing host")
	}
	if up.Repo == "" {
		return nil, fmt.Errorf("gitlab upstream missing repo")
	}

	projectPath := url.PathEscape(up.Repo)
	apiURL := fmt.Sprintf("https://%s/api/v4/projects/%s/releases?per_page=1", up.Host, projectPath)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, http.NoBody)
	if err != nil {
		return nil, err
	}
	if token != "" {
		// gitlab accepts PRIVATE-TOKEN for PATs and Authorization: Bearer for OAuth;
		// use PRIVATE-TOKEN since our config stores long-lived PATs.
		req.Header.Set("PRIVATE-TOKEN", token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("auth failed (%d) — set upstream.gitlab.tokens[%q]", resp.StatusCode, up.Host)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gitlab: http %d", resp.StatusCode)
	}

	var releases []glRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	if len(releases) == 0 {
		return nil, fmt.Errorf("no releases")
	}

	rel := releases[0]
	return &ReleaseInfo{
		Tag:         rel.TagName,
		ReleaseURL:  fmt.Sprintf("https://%s/%s/-/releases/%s", up.Host, up.Repo, rel.TagName),
		CompareTmpl: fmt.Sprintf("https://%s/%s/-/compare/%%s...%%s", up.Host, up.Repo),
	}, nil
}
