package pkg

import "golang.org/x/mod/semver"

// NormalizeSemver prepends "v" if missing and validates via semver.
// returns empty string for non-semver input.
func NormalizeSemver(v string) string {
	if v == "" {
		return ""
	}
	if v[0] != 'v' {
		v = "v" + v
	}
	if !semver.IsValid(v) {
		return ""
	}
	return v
}
