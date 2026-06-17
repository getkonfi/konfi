package selfupdate

import (
	"bytes"
	"strings"
	"testing"
)

func TestStylerPlainWhenNotInteractive(t *testing.T) {
	// a bytes.Buffer is not a *os.File, so it is never interactive or colored.
	s := newStyler(&bytes.Buffer{})
	if s.color || s.interactive {
		t.Fatalf("buffer styler should be plain: color=%v interactive=%v", s.color, s.interactive)
	}
	if got := s.success("ok"); got != "ok" {
		t.Fatalf("success wrapped plain text: %q", got)
	}
	if got := s.errText("boom"); got != "boom" {
		t.Fatalf("errText wrapped plain text: %q", got)
	}
}

func TestStylerColorWrapsWithAnsi(t *testing.T) {
	s := styler{out: &bytes.Buffer{}, color: true, interactive: true}
	got := s.success("done")
	if !strings.HasPrefix(got, ansiGreen) || !strings.HasSuffix(got, ansiReset) {
		t.Fatalf("success did not wrap with ansi: %q", got)
	}
}

func TestProgressWriterRendersPercent(t *testing.T) {
	var buf bytes.Buffer
	s := styler{out: &buf, interactive: true}
	pw := newProgressWriter(s, "konfi.tar.gz", 100)
	if _, err := pw.Write(make([]byte, 40)); err != nil {
		t.Fatal(err)
	}
	if _, err := pw.Write(make([]byte, 60)); err != nil {
		t.Fatal(err)
	}
	pw.finish()

	out := buf.String()
	if !strings.Contains(out, "100%") {
		t.Fatalf("finish should render 100%%: %q", out)
	}
	if !strings.Contains(out, "konfi.tar.gz") {
		t.Fatalf("progress should include label: %q", out)
	}
	if !strings.HasSuffix(out, "\n") {
		t.Fatalf("finish should end with newline: %q", out)
	}
}

func TestFprintErrorPlain(t *testing.T) {
	var buf bytes.Buffer
	FprintError(&buf, &ManagedInstall{Manager: "homebrew", Command: "brew upgrade konfi", Path: "/opt/homebrew/bin/konfi"})
	if !strings.Contains(buf.String(), "konfi update: ") {
		t.Fatalf("FprintError missing prefix: %q", buf.String())
	}
}
