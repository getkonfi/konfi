package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHeadURLFallsBackToGetWhenHeadRejected(t *testing.T) {
	var gotGet bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead:
			http.Error(w, "bad head", http.StatusBadRequest)
		case http.MethodGet:
			gotGet = true
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	}))
	defer srv.Close()

	client := srv.Client()
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	got := headURL(context.Background(), client, srv.URL)
	if got.status != http.StatusOK {
		t.Fatalf("expected GET fallback status %d, got %d (%v)", http.StatusOK, got.status, got.err)
	}
	if !gotGet {
		t.Fatal("expected GET fallback")
	}
}

func TestHeadURLKeepsRedirectResult(t *testing.T) {
	var gotGet bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead:
			w.Header().Set("Location", "/next")
			w.WriteHeader(http.StatusMovedPermanently)
		case http.MethodGet:
			gotGet = true
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	}))
	defer srv.Close()

	client := srv.Client()
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	got := headURL(context.Background(), client, srv.URL)
	if got.status != http.StatusMovedPermanently {
		t.Fatalf("expected redirect status %d, got %d (%v)", http.StatusMovedPermanently, got.status, got.err)
	}
	if got.location != "/next" {
		t.Fatalf("expected redirect location /next, got %q", got.location)
	}
	if gotGet {
		t.Fatal("did not expect GET fallback for redirect")
	}
}
