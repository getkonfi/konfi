package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type Severity int

const (
	Pass Severity = iota
	Info
	Warn
	Fail
)

func (s Severity) String() string {
	switch s {
	case Pass:
		return "PASS"
	case Info:
		return "INFO"
	case Warn:
		return "WARN"
	case Fail:
		return "FAIL"
	}
	return "?"
}

func (s Severity) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

type Finding struct {
	Severity Severity `json:"severity"`
	Category string   `json:"category"`
	Message  string   `json:"message"`
}

type AppReport struct {
	App      string    `json:"app"`
	Findings []Finding `json:"findings"`
}

func (r *AppReport) MaxSeverity() Severity {
	highest := Pass
	for _, f := range r.Findings {
		if f.Severity > highest {
			highest = f.Severity
		}
	}
	return highest
}

type Report struct {
	Apps []AppReport `json:"apps"`
}

func (r *Report) HasFail() bool {
	for i := range r.Apps {
		if r.Apps[i].MaxSeverity() >= Fail {
			return true
		}
	}
	return false
}

func (r *Report) HasWarn() bool {
	for i := range r.Apps {
		if r.Apps[i].MaxSeverity() >= Warn {
			return true
		}
	}
	return false
}

func (r *Report) WriteText(w io.Writer, verbose bool) {
	for _, app := range r.Apps {
		sev := app.MaxSeverity()
		fmt.Fprintf(w, "[%s] %s\n", sev, app.App)
		for _, f := range app.Findings {
			if !verbose && f.Severity <= Pass {
				continue
			}
			fmt.Fprintf(w, "  %s  %s\n", f.Severity, f.Message)
		}
	}

	// summary
	var fails, warns, passes int
	for _, app := range r.Apps {
		switch app.MaxSeverity() {
		case Fail:
			fails++
		case Warn:
			warns++
		default:
			passes++
		}
	}
	parts := []string{fmt.Sprintf("%d passed", passes)}
	if warns > 0 {
		parts = append(parts, fmt.Sprintf("%d warned", warns))
	}
	if fails > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", fails))
	}
	fmt.Fprintf(w, "\n%s\n", strings.Join(parts, ", "))
}

func (r *Report) WriteJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}
