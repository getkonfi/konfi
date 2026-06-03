package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

const maxDefaultFindings = 8

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

func (r *Report) WriteText(w io.Writer, verbose, color bool) {
	text := newTextReportWriter(w, verbose, color)
	text.WriteHeader()
	for _, app := range r.Apps {
		text.WriteApp(app)
	}
	text.WriteSummary(*r)
}

type textReportWriter struct {
	w        io.Writer
	verbose  bool
	style    textStyle
	wroteApp bool
}

func newTextReportWriter(w io.Writer, verbose, color bool) *textReportWriter {
	return &textReportWriter{
		w:       w,
		verbose: verbose,
		style:   textStyle{color: color},
	}
}

func (w *textReportWriter) WriteHeader() {
	fmt.Fprintf(w.w, "%s\n", w.style.title("schema verify"))
}

func (w *textReportWriter) WriteApp(app AppReport) {
	if w.wroteApp {
		fmt.Fprintln(w.w)
	}
	w.wroteApp = true

	sev := app.MaxSeverity()
	findings := visibleFindings(app.Findings, w.verbose)
	details := displayFindings(app.Findings, w.verbose)
	summary := findingSummary(findings)
	if summary != "" {
		fmt.Fprintf(w.w, "%s  %s  %s\n", w.style.severity(sev), w.style.app(app.App), w.style.muted(summary))
	} else {
		fmt.Fprintf(w.w, "%s  %s\n", w.style.severity(sev), w.style.app(app.App))
	}

	for _, f := range details {
		fmt.Fprintf(w.w, "  %s  %s  %s\n", w.style.findingSeverity(f.Severity), w.style.category(f.Category), f.Message)
	}
}

func (w *textReportWriter) WriteSummary(report Report) {
	var fails, warns, passes int
	for _, app := range report.Apps {
		switch app.MaxSeverity() {
		case Fail:
			fails++
		case Warn:
			warns++
		default:
			passes++
		}
	}
	parts := []string{w.style.summary(Pass, fmt.Sprintf("%d passed", passes))}
	if warns > 0 {
		parts = append(parts, w.style.summary(Warn, fmt.Sprintf("%d warned", warns)))
	}
	if fails > 0 {
		parts = append(parts, w.style.summary(Fail, fmt.Sprintf("%d failed", fails)))
	}
	fmt.Fprintf(w.w, "\n%s  %s\n", w.style.muted("summary"), strings.Join(parts, w.style.muted(", ")))
}

func (r *Report) WriteJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

type textStyle struct {
	color bool
}

func (s textStyle) title(v string) string {
	return s.paint("38;5;153;1", v)
}

func (s textStyle) app(v string) string {
	return s.paint("38;5;252;1", v)
}

func (s textStyle) muted(v string) string {
	return s.paint("38;5;245", v)
}

func (s textStyle) category(v string) string {
	return s.paint("38;5;109", padRight(categoryLabel(v), 15))
}

func (s textStyle) severity(sev Severity) string {
	return s.paint(severityColor(sev), strings.ToLower(sev.String()))
}

func (s textStyle) findingSeverity(sev Severity) string {
	return s.paint(severityColor(sev), strings.ToLower(sev.String()))
}

func (s textStyle) summary(sev Severity, v string) string {
	return s.paint(severityColor(sev), v)
}

func (s textStyle) paint(code, v string) string {
	if !s.color {
		return v
	}
	return "\x1b[" + code + "m" + v + "\x1b[0m"
}

func severityColor(sev Severity) string {
	switch sev {
	case Pass:
		return "38;5;108"
	case Info:
		return "38;5;110"
	case Warn:
		return "38;5;180"
	case Fail:
		return "38;5;167"
	default:
		return "38;5;245"
	}
}

func categoryLabel(category string) string {
	switch category {
	case "structural":
		return "schema"
	case "url":
		return "docs"
	case observedConfigCategory:
		return "observed config"
	default:
		return strings.ReplaceAll(category, "_", " ")
	}
}

func visibleFindings(findings []Finding, verbose bool) []Finding {
	out := make([]Finding, 0, len(findings))
	for _, f := range findings {
		if !verbose && f.Severity <= Pass {
			continue
		}
		out = append(out, f)
	}
	return out
}

func displayFindings(findings []Finding, verbose bool) []Finding {
	visible := visibleFindings(findings, verbose)
	if verbose || len(visible) <= maxDefaultFindings {
		return visible
	}

	out := make([]Finding, 0, len(visible))
	infoByCategory := make(map[string][]Finding)
	var categoryOrder []string

	for _, f := range visible {
		if f.Severity != Info {
			out = append(out, f)
			continue
		}
		if _, ok := infoByCategory[f.Category]; !ok {
			categoryOrder = append(categoryOrder, f.Category)
		}
		infoByCategory[f.Category] = append(infoByCategory[f.Category], f)
	}

	for _, category := range categoryOrder {
		group := infoByCategory[category]
		if len(group) == 1 {
			out = append(out, group[0])
			continue
		}
		out = append(out, Finding{
			Severity: Info,
			Category: category,
			Message:  fmt.Sprintf("%d info findings hidden; run with -v to expand", len(group)),
		})
	}

	return out
}

func findingSummary(findings []Finding) string {
	if len(findings) == 0 {
		return ""
	}
	counts := make(map[Severity]int)
	for _, f := range findings {
		counts[f.Severity]++
	}

	var parts []string
	for _, sev := range []Severity{Fail, Warn, Info, Pass} {
		if counts[sev] == 0 {
			continue
		}
		parts = append(parts, fmt.Sprintf("%d %s", counts[sev], strings.ToLower(sev.String())))
	}
	return strings.Join(parts, ", ")
}

func padRight(v string, width int) string {
	if len(v) >= width {
		return v
	}
	return v + strings.Repeat(" ", width-len(v))
}
