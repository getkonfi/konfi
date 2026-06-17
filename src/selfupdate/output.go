package selfupdate

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// minimal ansi palette for the pre-tui update command. the tui's lipgloss
// theme depends on a color profile that is only set up once the program runs,
// so this path keeps its own small, predictable styling.
const (
	ansiReset = "\x1b[0m"
	ansiDim   = "\x1b[2m"
	ansiRed   = "\x1b[31m"
	ansiGreen = "\x1b[32m"
	ansiCyan  = "\x1b[36m"
)

// styler decides whether to colorize output and tracks where to write it.
type styler struct {
	out         io.Writer
	color       bool
	interactive bool
}

func newStyler(w io.Writer) styler {
	interactive := isInteractive(w)
	_, noColor := os.LookupEnv("NO_COLOR")
	color := interactive && !noColor && os.Getenv("TERM") != "dumb"
	return styler{out: w, color: color, interactive: interactive}
}

// isInteractive reports whether w is a character device (a terminal).
func isInteractive(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func (s styler) wrap(code, text string) string {
	if !s.color {
		return text
	}
	return code + text + ansiReset
}

func (s styler) success(text string) string { return s.wrap(ansiGreen, text) }
func (s styler) info(text string) string    { return s.wrap(ansiCyan, text) }
func (s styler) dim(text string) string     { return s.wrap(ansiDim, text) }
func (s styler) errText(text string) string { return s.wrap(ansiRed, text) }

// FprintError writes a user-facing update error to w, in red when interactive.
func FprintError(w io.Writer, err error) {
	s := newStyler(w)
	fmt.Fprintln(w, s.errText("konfi update: "+err.Error()))
}

// progressWriter renders a single-line download bar that redraws in place.
// it is only used on interactive terminals with a known content length.
type progressWriter struct {
	s       styler
	label   string
	total   int64
	written int64
	last    time.Time
	started bool
}

func newProgressWriter(s styler, label string, total int64) *progressWriter {
	return &progressWriter{s: s, label: label, total: total}
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	pw.written += int64(len(p))
	pw.render(false)
	return len(p), nil
}

func (pw *progressWriter) render(final bool) {
	now := time.Now()
	if !final && pw.started && now.Sub(pw.last) < 80*time.Millisecond {
		return
	}
	pw.last = now
	pw.started = true

	pct := 0.0
	if pw.total > 0 {
		pct = float64(pw.written) / float64(pw.total)
	}
	if pct > 1 {
		pct = 1
	}

	const width = 24
	filled := int(pct * width)
	bar := pw.s.success(strings.Repeat("█", filled)) + pw.s.dim(strings.Repeat("░", width-filled))
	fmt.Fprintf(pw.s.out, "\r  %s %s %3.0f%% %s", pw.s.info("↓"), bar, pct*100, pw.label)
}

func (pw *progressWriter) finish() {
	pw.render(true)
	fmt.Fprintln(pw.s.out)
}
