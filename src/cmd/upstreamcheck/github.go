package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/emin/konfigurator/pkg"
)

const githubAPI = "https://api.github.com"

type ghRelease struct {
	TagName    string `json:"tag_name"`
	HTMLURL    string `json:"html_url"`
	Prerelease bool   `json:"prerelease"`
	Draft      bool   `json:"draft"`
}

// fetchGitHubLatest queries the releases/latest endpoint, which returns
// the newest non-draft, non-prerelease release. repos that only cut
// prereleases return 404 — the caller treats that as "no stable release yet".
func fetchGitHubLatest(ctx context.Context, client *http.Client, up *pkg.Upstream, token string) (*ReleaseInfo, error) {
	if up.Repo == "" {
		return nil, fmt.Errorf("github upstream missing repo")
	}

	url := fmt.Sprintf("%s/repos/%s/releases/latest", githubAPI, up.Repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotFound:
		return nil, fmt.Errorf("no stable release (404)")
	case http.StatusForbidden:
		return nil, fmt.Errorf("rate limited or forbidden (403) — set upstream.github.token")
	case http.StatusOK:
	default:
		return nil, fmt.Errorf("github: http %d", resp.StatusCode)
	}

	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	return &ReleaseInfo{
		Tag:         rel.TagName,
		ReleaseURL:  rel.HTMLURL,
		CompareTmpl: fmt.Sprintf("https://github.com/%s/compare/%%s...%%s", up.Repo),
	}, nil
}
