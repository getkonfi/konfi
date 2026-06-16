package ui

import (
	"strings"
	"testing"
)

func TestHeaderLeftLinesShowsConfigPathWhileLoadPending(t *testing.T) {
	c := newContent(testTheme())
	c.konfable = detailTestKonfable{}

	got := c.headerLeftLines()
	if got[1] != "/tmp/test.conf" {
		t.Fatalf("pending load header path = %q, want config path", got[1])
	}

	c.configLoadFailed = true
	got = c.headerLeftLines()
	if !strings.Contains(got[1], "load failed") {
		t.Fatalf("failed load header path = %q, want load failure label", got[1])
	}
}

func TestLoadAppSplitFlapTargetsConfigPathWhileLoadPending(t *testing.T) {
	c := newContent(testTheme())
	c.konfable = &switchTestKonfable{name: "old"}
	c.insightLines = []string{"ready"}

	_ = c.loadApp(&switchTestKonfable{name: "next"})

	if c.splitFlap == nil {
		t.Fatal("loadApp did not create split-flap animation")
	}
	if got := c.splitFlap.target[1]; got != "/tmp/next.conf" {
		t.Fatalf("split-flap target path = %q, want config path", got)
	}
}
