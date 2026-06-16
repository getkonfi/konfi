package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/getkonfi/konfi/konfables"
	"github.com/getkonfi/konfi/theme"
)

// labelColumnWidth computes the max label width for the active section.
// headerLeftLines returns the left column lines for the header.
func (c *content) headerLeftLines() []string {
	title := ""
	if c.konfable != nil {
		title = c.konfable.Name()
		if v, ok := c.versions[c.konfable.Name()]; ok && v != "" {
			title += " " + v
		}
	}

	path := ""
	if c.config != nil {
		path = c.config.Path
		if path == "" && c.konfable != nil {
			path = c.konfable.Info().Name
		}
	} else if c.konfable != nil {
		if c.configLoadFailed {
			path = "load failed — browse only"
		} else {
			path = c.konfable.ConfigPath()
			if path == "" {
				path = c.konfable.Info().Name
			}
		}
	}
	if c.fileState != "" {
		path += " [" + c.fileState + "]"
	}

	insight := ""
	if len(c.insightLines) > 0 {
		insight = c.insightLines[c.insightIdx%len(c.insightLines)]
	}

	return []string{title, path, insight}
}

// renderHeader produces the two-column header or narrow fallback.
// always renders exactly logoBlockH lines + trailing newline.
func (c *content) renderHeader(width int) string {
	hh := logoBlockH

	if c.konfable == nil {
		// no app selected — empty header padded to height
		lines := make([]string, hh)
		for i := range lines {
			lines[i] = ""
		}
		return strings.Join(lines, "\n") + "\n"
	}

	// build right column: logo (animated if running, static otherwise)
	var rightLines []string
	if c.logoAnim != nil && !c.logoAnim.Done {
		art := c.logoAnim.CurrentFrame().Render()
		rightLines = strings.Split(art, "\n")
	} else if logo, ok := konfables.Logos[c.konfable.Name()]; ok {
		art := logo.Render()
		rightLines = strings.Split(art, "\n")
	}
	rightW := 0
	for _, l := range rightLines {
		if w := lipgloss.Width(l); w > rightW {
			rightW = w
		}
	}
	rightBlock := strings.Join(rightLines, "\n")

	leftW := width - rightW - 2 // 2 chars gap
	if leftW < 20 {
		// narrow fallback: centered logo
		var lines []string
		if c.logoAnim != nil && !c.logoAnim.Done {
			art := c.logoAnim.CurrentFrame().Render()
			lines = append(lines, strings.Split(centerBlock(art, width), "\n")...)
		} else if logo, ok := konfables.Logos[c.konfable.Name()]; ok {
			art := logo.Render()
			lines = append(lines, strings.Split(centerBlock(art, width), "\n")...)
		}
		lines = append(lines, "")
		for len(lines) < hh {
			lines = append(lines, "")
		}
		if len(lines) > hh {
			lines = lines[:hh]
		}
		return strings.Join(lines, "\n") + "\n"
	}

	// two-column: build left lines
	leftData := c.headerLeftLines()
	if c.splitFlap != nil && !c.splitFlap.done {
		// replace with split-flap animation frames
		leftData = make([]string, len(c.splitFlap.current))
		copy(leftData, c.splitFlap.current)
	}

	// style + truncate left lines
	styledLeft := make([]string, len(leftData))
	styles := []lipgloss.Style{c.theme.Primary, c.theme.Muted, c.theme.InsightText}
	for i, line := range leftData {
		// truncate to leftW (plain text before styling)
		if len(line) > leftW {
			line = theme.Truncate(line, leftW)
		}
		s := c.theme.Text
		if i < len(styles) {
			s = styles[i]
		}
		// line 1 (path): color fileState suffix
		if i == 1 && c.fileState != "" {
			switch c.fileState {
			case "unsaved":
				s = c.theme.Warning
			case "reloaded":
				s = c.theme.Accent
			case "new":
				s = c.theme.Muted
			}
		}
		// line 2 (insight): use warning style for linter diagnostics
		if i == 2 && c.insightWarningCount > 0 && len(c.insightLines) > 0 {
			idx := c.insightIdx % len(c.insightLines)
			if idx < c.insightWarningCount {
				s = c.theme.Warning
			}
		}
		styledLeft[i] = s.Render(line)
	}

	// pad left lines to headerHeight
	for len(styledLeft) < hh {
		styledLeft = append(styledLeft, "")
	}

	// build left block with fixed width for alignment
	leftBlock := lipgloss.NewStyle().Width(leftW).Render(strings.Join(styledLeft[:hh], "\n"))

	// right-align the right column
	rightStyle := lipgloss.NewStyle().Width(rightW + 2).Align(lipgloss.Right)
	// note: these two styles depend on dynamic widths, computed once per renderHeader call
	rightAligned := rightStyle.Render(rightBlock)

	joined := lipgloss.JoinHorizontal(lipgloss.Top, leftBlock, rightAligned)

	// pad output to exactly headerHeight rows
	outLines := strings.Split(joined, "\n")
	for len(outLines) < hh {
		outLines = append(outLines, "")
	}
	if len(outLines) > hh {
		outLines = outLines[:hh]
	}

	return strings.Join(outLines, "\n") + "\n"
}
