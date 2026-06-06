package ui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/eminert/konfi/konfables"
	"github.com/eminert/konfi/pkg"
)

// dashboardApp holds summary info for the landing page.
type dashboardApp struct {
	icon            string
	name            string
	installed       bool
	version         string
	configuredCount int    // fields with non-default values
	totalFields     int    // total schema fields
	deprecatedCount int    // deprecated diagnostics
	newCount        int    // fields added in the detected version
	coverage        string // from schema.Coverage
	minAppVersion   string // schema min supported version
	maxAppVersion   string // schema max supported version
}

// buildDashboardApps computes the landing-page tiles and their stats (field
// counts, configured/deprecated counts, coverage). it opens each installed
// app's config file, so it does real I/O — kept out of NewRoot's wiring.
func buildDashboardApps(apps []konfables.Konfable, installed map[string]bool, nerdFont bool, versions map[string]string, schemaCache map[string]*pkg.Schema, newCounts map[string]int) []dashboardApp {
	var out []dashboardApp
	for _, k := range apps {
		info := k.Info()
		nIcon := info.NerdIcon
		if !nerdFont {
			nIcon = info.Icon
		}
		if nIcon == "" {
			nIcon = info.Icon
		}
		da := dashboardApp{
			icon:      nIcon,
			name:      k.Name(),
			installed: installed[k.Name()],
		}
		if v, ok := versions[k.Name()]; ok {
			da.version = v
		}

		// stats from schema
		if s, ok := schemaCache[k.Name()]; ok {
			for si := range s.Sections {
				da.totalFields += len(s.Sections[si].Fields)
			}
			da.coverage = s.Coverage
			da.minAppVersion = s.MinAppVersion
			da.maxAppVersion = s.MaxAppVersion

			// count configured + deprecated for installed apps
			if installed[k.Name()] {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				cf, err := pkg.NewConfigFile(ctx, k)
				cancel()
				if err == nil && cf != nil {
					data := cf.Content()
					p := k.Parser()
					if p != nil {
						configured := 0
						if bp, ok := p.(konfables.BatchParser); ok {
							all := bp.FindAll(data)
							configured = len(all)
						} else {
							// per-field lookup against schema keys
							for si := range s.Sections {
								for fi := range s.Sections[si].Fields {
									if _, found := p.FindValue(data, s.Sections[si].Fields[fi].Key); found {
										configured++
									}
								}
							}
						}
						da.configuredCount = configured

						// deprecated count via diagnostics
						var configKeys []string
						if bp, ok := p.(konfables.BatchParser); ok {
							for key := range bp.FindAll(data) {
								configKeys = append(configKeys, key)
							}
						}
						if len(configKeys) > 0 {
							diags := pkg.Diagnose(configKeys, s, da.version)
							for _, d := range diags {
								if d.Kind == "deprecated" {
									da.deprecatedCount++
								}
							}
						}
					}
				}
			}
		}

		da.newCount = newCounts[k.Name()]
		out = append(out, da)
	}
	return out
}

// renderDashboard builds the welcome/landing page shown before any app is selected.
func (c *content) renderDashboard(width int) string {
	var b strings.Builder

	// logo
	if logo, ok := konfables.Logos["konfi"]; ok {
		art := logo.Render()
		b.WriteString(centerBlock(art, width))
		b.WriteByte('\n')
	}

	// title + version
	title := c.theme.Primary.Bold(true).Render("konfi")
	ver := c.theme.Muted.Render(" v" + c.appVersion)
	b.WriteString(centerLine(title+ver, width))
	b.WriteByte('\n')
	b.WriteByte('\n')

	// app list
	var installed, notInstalled []dashboardApp
	var totalDeprecated, totalNew int
	for i := range c.dashboardApps {
		if c.dashboardApps[i].installed {
			installed = append(installed, c.dashboardApps[i])
			totalDeprecated += c.dashboardApps[i].deprecatedCount
			totalNew += c.dashboardApps[i].newCount
		} else {
			notInstalled = append(notInstalled, c.dashboardApps[i])
		}
	}

	// sort installed: most configured first, then alphabetical
	sort.Slice(installed, func(i, j int) bool {
		if installed[i].configuredCount != installed[j].configuredCount {
			return installed[i].configuredCount > installed[j].configuredCount
		}
		return installed[i].name < installed[j].name
	})
	// sort not-detected alphabetically
	sort.Slice(notInstalled, func(i, j int) bool {
		return notInstalled[i].name < notInstalled[j].name
	})

	// aggregate summary — actionable signals only
	if len(installed) > 0 {
		var parts []string
		if totalNew > 0 {
			parts = append(parts, fmt.Sprintf("%d new", totalNew))
		}
		if totalDeprecated > 0 {
			parts = append(parts, fmt.Sprintf("%d deprecated", totalDeprecated))
		}
		if bm := len(c.bookmarks); bm > 0 {
			parts = append(parts, fmt.Sprintf("%d bookmarked", bm))
		}
		if len(parts) > 0 {
			summary := strings.Join(parts, " · ")
			b.WriteString(centerLine(c.theme.Muted.Render(summary), width))
			b.WriteByte('\n')
			b.WriteByte('\n')
		}
	}

	ruleW := width / 2
	if ruleW < 20 {
		ruleW = 20
	}
	if ruleW > width {
		ruleW = width
	}

	// compute column widths across both groups for alignment
	nameW, verW := 0, 0
	for i := range installed {
		if len(installed[i].name) > nameW {
			nameW = len(installed[i].name)
		}
		if len(installed[i].version) > verW {
			verW = len(installed[i].version)
		}
	}
	for i := range notInstalled {
		if len(notInstalled[i].name) > nameW {
			nameW = len(notInstalled[i].name)
		}
	}

	// build all lines first, then left-align the block at a single offset
	var lines []string
	maxW := 0

	if len(installed) > 0 {
		label := "── installed "
		pad := ruleW - len(label)
		if pad < 0 {
			pad = 0
		}
		hdr := c.theme.Muted.Render(label + strings.Repeat("─", pad))
		lines = append(lines, hdr)
		for i := range installed {
			a := &installed[i]
			icon := c.theme.Primary.Render(a.icon)
			name := c.theme.Text.Render(" " + padRight(a.name, nameW))
			ver := strings.Repeat(" ", verW+2)
			if a.version != "" {
				ver = "  " + padRight(a.version, verW)
			}
			ver = c.theme.Muted.Render(ver)
			stats := c.dashboardStats(a)
			lines = append(lines, icon+name+ver+stats)
		}
	}

	if len(notInstalled) > 0 {
		lines = append(lines, "") // blank separator
		label := "── not detected "
		pad := ruleW - len(label)
		if pad < 0 {
			pad = 0
		}
		hdr := c.theme.Muted.Render(label + strings.Repeat("─", pad))
		lines = append(lines, hdr)
		for i := range notInstalled {
			a := &notInstalled[i]
			icon := c.theme.Muted.Faint(true).Render(a.icon)
			name := c.theme.Muted.Faint(true).Render(" " + padRight(a.name, nameW))
			ver := ""
			switch {
			case a.minAppVersion != "" && a.maxAppVersion != "":
				ver = fmt.Sprintf("  %s – %s", a.minAppVersion, a.maxAppVersion)
			case a.minAppVersion != "":
				ver = fmt.Sprintf("  %s+", a.minAppVersion)
			case a.maxAppVersion != "":
				ver = fmt.Sprintf("  up to %s", a.maxAppVersion)
			}
			if ver != "" {
				ver = c.theme.Muted.Faint(true).Render(ver)
			}
			lines = append(lines, icon+name+ver)
		}
	}

	// find widest line, then left-align all lines at the same offset
	for _, l := range lines {
		if w := lipgloss.Width(l); w > maxW {
			maxW = w
		}
	}
	leftPad := (width - maxW) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	prefix := strings.Repeat(" ", leftPad)
	for _, l := range lines {
		b.WriteString(prefix + l)
		b.WriteByte('\n')
	}

	b.WriteByte('\n')
	hints := []struct{ key, desc string }{
		{"↑↓", "navigate"},
		{"⏎", "select"},
		{"/", "search"},
		{"?", "help"},
	}
	var parts []string
	for _, h := range hints {
		k := c.theme.Primary.Render(h.key)
		d := c.theme.Muted.Render(" " + h.desc)
		parts = append(parts, k+d)
	}
	hintLine := strings.Join(parts, c.theme.Muted.Render("   "))
	b.WriteString(centerLine(hintLine, width))

	return b.String()
}

// dashboardStats formats the actionable stats suffix for a dashboard app.
// configured count is omitted — sort order communicates engagement.
func (c *content) dashboardStats(a *dashboardApp) string {
	var parts []string
	if a.newCount > 0 {
		parts = append(parts, fmt.Sprintf("%d new", a.newCount))
	}
	if a.deprecatedCount > 0 {
		parts = append(parts, fmt.Sprintf("%d deprecated", a.deprecatedCount))
	}
	if a.coverage != "" && a.coverage != "full" {
		parts = append(parts, a.coverage)
	}
	if len(parts) == 0 {
		return ""
	}
	return c.theme.Muted.Render("  " + strings.Join(parts, " · "))
}
