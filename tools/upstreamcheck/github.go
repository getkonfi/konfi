package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/eminert/konfi/pkg"
	"golang.org/x/mod/semver"
)

var githubAPI = "https://api.github.com"

type ghRelease struct {
	TagName    string `json:"tag_name"`
	HTMLURL    string `json:"html_url"`
	Prerelease bool   `json:"prerelease"`
	Draft      bool   `json:"draft"`
}

type ghTag struct {
	Name string `json:"name"`
}

// fetchGitHubLatest queries the releases/latest endpoint, which returns
// the newest non-draft, non-prerelease release. if a repo has version tags
// but no GitHub releases, it falls back to the newest stable semver tag.
func fetchGitHubLatest(ctx context.Context, client *http.Client, up *pkg.Upstream, token string) (*ReleaseInfo, error) {
	if up.Repo == "" {
		return nil, fmt.Errorf("github upstream missing repo")
	}

	url := fmt.Sprintf("%s/repos/%s/releases/latest", githubAPI, up.Repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
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

	switch resp.StatusCode {
	case http.StatusNotFound:
		resp.Body.Close()
		return fetchGitHubLatestTag(ctx, client, up, token)
	case http.StatusForbidden:
		resp.Body.Close()
		return nil, fmt.Errorf("rate limited or forbidden (403) — set upstream.github.token")
	case http.StatusOK:
		defer resp.Body.Close()
	default:
		resp.Body.Close()
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

func fetchGitHubLatestTag(ctx context.Context, client *http.Client, up *pkg.Upstream, token string) (*ReleaseInfo, error) {
	url := fmt.Sprintf("%s/repos/%s/tags?per_page=100", githubAPI, up.Repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
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
	case http.StatusForbidden:
		return nil, fmt.Errorf("rate limited or forbidden (403) — set upstream.github.token")
	case http.StatusOK:
	default:
		return nil, fmt.Errorf("github tags: http %d", resp.StatusCode)
	}

	var tags []ghTag
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return nil, fmt.Errorf("decode tags: %w", err)
	}

	tag, ok := newestStableSemverTag(tags, up.TagPrefix)
	if !ok {
		return nil, fmt.Errorf("no stable release (404); no stable semver tags")
	}

	return &ReleaseInfo{
		Tag:         tag,
		ReleaseURL:  fmt.Sprintf("https://github.com/%s/tree/%s", up.Repo, tag),
		CompareTmpl: fmt.Sprintf("https://github.com/%s/compare/%%s...%%s", up.Repo),
	}, nil
}

func newestStableSemverTag(tags []ghTag, tagPrefix string) (string, bool) {
	var bestTag, bestVersion string
	for _, tag := range tags {
		version := strings.TrimPrefix(tag.Name, tagPrefix)
		if tagPrefix != "" && version == tag.Name {
			continue
		}
		version = "v" + strings.TrimPrefix(version, "v")
		if !semver.IsValid(version) || semver.Prerelease(version) != "" {
			continue
		}
		if bestVersion == "" || semver.Compare(version, bestVersion) > 0 {
			bestTag = tag.Name
			bestVersion = version
		}
	}
	return bestTag, bestTag != ""
}
