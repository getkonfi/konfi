package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/getkonfi/konfi/pkg"
)

// Status is the high-level outcome for one app.
type Status string

const (
	StatusCurrent Status = "current" // supported == latest
	StatusBehind  Status = "behind"  // supported < latest — schema needs update
	StatusAhead   Status = "ahead"   // supported > latest — unlikely, prereleases skipped
	StatusSkipped Status = "skipped" // no upstream configured
	StatusError   Status = "error"   // network/api failure
)

// ReleaseInfo is what a fetcher returns. CompareTmpl is a printf format
// string with two %s placeholders — from-tag then to-tag. lets the report
// layer decide tag prefixing without teaching fetchers about it.
type ReleaseInfo struct {
	Tag         string
	ReleaseURL  string
	CompareTmpl string
}

type AppResult struct {
	App        string `json:"app"`
	Supported  string `json:"supported,omitempty"`
	Latest     string `json:"latest,omitempty"`
	Status     Status `json:"status"`
	Detail     string `json:"detail,omitempty"`
	ReleaseURL string `json:"release_url,omitempty"`
	CompareURL string `json:"compare_url,omitempty"`
}

type Report struct {
	Results []AppResult `json:"results"`
}

func (r *Report) HasError() bool {
	for _, a := range r.Results {
		if a.Status == StatusError {
			return true
		}
	}
	return false
}

// classify decides Status + CompareURL given the supported version, the
// fetched tag, and the upstream's tag prefix. stripping the prefix yields
// the canonical app version we stored in max_app_version.
func classify(res *AppResult, info *ReleaseInfo, tagPrefix string) {
	latest := strings.TrimPrefix(info.Tag, tagPrefix)
	res.Latest = latest
	res.ReleaseURL = info.ReleaseURL

	if res.Supported == "" {
		res.Status = StatusError
		res.Detail = "schema has no max_app_version"
		return
	}

	cmp, ok := pkg.CompareAppVersions(res.Supported, latest)
	if !ok {
		if res.Supported == latest {
			res.Status = StatusCurrent
		} else {
			res.Status = StatusBehind
			res.CompareURL = fmt.Sprintf(info.CompareTmpl, tagPrefix+res.Supported, info.Tag)
		}
		return
	}

	switch {
	case cmp == 0:
		res.Status = StatusCurrent
	case cmp < 0:
		res.Status = StatusBehind
		res.CompareURL = fmt.Sprintf(info.CompareTmpl, tagPrefix+res.Supported, info.Tag)
	default:
		res.Status = StatusAhead
	}
}

func (r *Report) WriteText(w io.Writer) {
	const format = "%-14s %-12s %-12s %-8s  %s\n"
	fmt.Fprintf(w, format, "APP", "SUPPORTED", "LATEST", "STATUS", "LINK")
	fmt.Fprintln(w, strings.Repeat("─", 70))

	var behind, errors int
	for _, a := range r.Results {
		link := a.CompareURL
		if link == "" {
			link = a.Detail
		}
		fmt.Fprintf(w, format, a.App, dash(a.Supported), dash(a.Latest), a.Status, link)

		switch a.Status {
		case StatusBehind:
			behind++
		case StatusError:
			errors++
		}
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "%d total, %d behind, %d error\n", len(r.Results), behind, errors)
}

func (r *Report) WriteJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

func dash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
