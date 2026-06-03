package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eminert/konfi/pkg"
)

func TestFetchGitHubLatestFallsBackToTags(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/ghostty-org/ghostty/releases/latest":
			http.NotFound(w, r)
		case "/repos/ghostty-org/ghostty/tags":
			tags := []ghTag{
				{Name: "v1.3.0"},
				{Name: "v1.4.0-beta.1"},
				{Name: "v1.3.1"},
				{Name: "nightly"},
			}
			if err := json.NewEncoder(w).Encode(tags); err != nil {
				t.Errorf("encode tags: %v", err)
			}
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	oldAPI := githubAPI
	githubAPI = srv.URL
	t.Cleanup(func() { githubAPI = oldAPI })

	info, err := fetchGitHubLatest(context.Background(), srv.Client(), &pkg.Upstream{
		Repo:      "ghostty-org/ghostty",
		TagPrefix: "v",
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	if info.Tag != "v1.3.1" {
		t.Fatalf("tag = %q, want v1.3.1", info.Tag)
	}
}
