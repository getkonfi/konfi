package main

import "testing"

func TestClassifyLooseAppVersionAhead(t *testing.T) {
	res := &AppResult{App: "tmux", Supported: "3.6b"}
	classify(res, &ReleaseInfo{
		Tag:         "3.6a",
		ReleaseURL:  "https://github.com/tmux/tmux/releases/tag/3.6a",
		CompareTmpl: "https://github.com/tmux/tmux/compare/%s...%s",
	}, "")
	if res.Status != StatusAhead {
		t.Fatalf("status = %s, want %s", res.Status, StatusAhead)
	}
}
