package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/emin/konfigurator/pkg"

	"golang.org/x/mod/semver"
)

func (c *content) buildFieldList() {
	c.fields = nil
	c.fieldSection = nil
	if c.schema == nil {
		return
	}
	for si, sec := range c.schema.Sections {
		for range sec.Fields {
			c.fieldSection = append(c.fieldSection, si)
		}
		c.fields = append(c.fields, sec.Fields...)
	}
	c.searchIndex = pkg.NewSearchIndex(c.schema.Sections)
	// cache label column width and pre-padded labels
	c.labelW = 0
	for i := range c.fields {
		if len(c.fields[i].Label) > c.labelW {
			c.labelW = len(c.fields[i].Label)
		}
	}
	c.paddedLabels = make([]string, len(c.fields))
	for i := range c.fields {
		c.paddedLabels[i] = fmt.Sprintf("%-*s", c.labelW, c.fields[i].Label)
	}
	c.refilter()
}

// refilter rebuilds the visible row slice with interleaved section headers.
func (c *content) refilter() {
	c.visible = c.visible[:0]
	if c.schema == nil {
		return
	}

	query := strings.ToLower(strings.TrimSpace(c.search.Value()))

	// delegate to bm25 ranking when a query is active
	if query != "" && c.searchIndex != nil {
		c.refilterRanked(query)
		return
	}

	// track which section we last emitted a header for
	lastHeaderSection := -1

	for i := range c.fields {
		f := &c.fields[i]
		si := c.fieldSection[i]
		if c.configuredOnly {
			if _, ok := c.values[f.Key]; !ok {
				continue
			}
		}
		if c.showNewOnly {
			if f.Since == "" {
				continue
			}
			if c.konfable != nil {
				nv := pkg.NormalizeSemver(c.versions[c.konfable.Name()])
				ns := pkg.NormalizeSemver(f.Since)
				if nv != "" && ns != "" && semver.MajorMinor(ns) != semver.MajorMinor(nv) {
					continue
				}
			}
		}
		// insert section header before first field of each section
		if si != lastHeaderSection {
			c.visible = append(c.visible, row{isSection: true, sectionIdx: si, fieldIdx: -1})
			lastHeaderSection = si
		}
		c.visible = append(c.visible, row{sectionIdx: si, fieldIdx: i})
	}

	// clear search matches (empty-query path, ranked path handles its own)
	c.searchMatches = c.searchMatches[:0]
	c.searchIdx = 0

	// clamp cursor
	if len(c.visible) == 0 {
		c.cursor = 0
	} else if c.cursor >= len(c.visible) {
		c.cursor = len(c.visible) - 1
	}
	if c.cursor < 0 {
		c.cursor = 0
	}

	// ensure cursor is not stuck on a section header after clamping
	if len(c.visible) > 0 && c.cursor >= 0 && c.cursor < len(c.visible) && c.visible[c.cursor].isSection {
		c.skipSectionHeaders(1)
	}
}

// refilterRanked builds the visible row slice using bm25-ranked results.
func (c *content) refilterRanked(query string) {
	c.visible = c.visible[:0]
	c.searchMatches = c.searchMatches[:0]

	results := c.searchIndex.Search(query)
	if len(results) == 0 {
		c.searchIdx = 0
		c.cursor = 0
		return
	}

	// apply pre-filters and map to flat field indices
	type rankedField struct {
		flatIdx    int
		sectionIdx int
		score      float64
	}
	var filtered []rankedField
	for _, r := range results {
		// compute flat field index from section/field position
		flatIdx := 0
		for si := 0; si < r.SectionIdx; si++ {
			flatIdx += len(c.schema.Sections[si].Fields)
		}
		flatIdx += r.FieldIdx

		f := &c.fields[flatIdx]
		if c.configuredOnly {
			if _, ok := c.values[f.Key]; !ok {
				continue
			}
		}
		if c.showNewOnly {
			if f.Since == "" {
				continue
			}
			if c.konfable != nil {
				nv := pkg.NormalizeSemver(c.versions[c.konfable.Name()])
				ns := pkg.NormalizeSemver(f.Since)
				if nv != "" && ns != "" && semver.MajorMinor(ns) != semver.MajorMinor(nv) {
					continue
				}
			}
		}
		filtered = append(filtered, rankedField{
			flatIdx:    flatIdx,
			sectionIdx: r.SectionIdx,
			score:      r.Score,
		})
	}

	if len(filtered) == 0 {
		c.searchIdx = 0
		c.cursor = 0
		return
	}

	// group by section, track best score per section
	type sectionGroup struct {
		sectionIdx int
		bestScore  float64
		fields     []rankedField
	}
	sectionMap := make(map[int]*sectionGroup)
	var sectionOrder []int
	for _, rf := range filtered {
		sg, ok := sectionMap[rf.sectionIdx]
		if !ok {
			sg = &sectionGroup{sectionIdx: rf.sectionIdx}
			sectionMap[rf.sectionIdx] = sg
			sectionOrder = append(sectionOrder, rf.sectionIdx)
		}
		sg.fields = append(sg.fields, rf)
		if rf.score > sg.bestScore {
			sg.bestScore = rf.score
		}
	}

	// sort sections by best score descending
	sort.Slice(sectionOrder, func(i, j int) bool {
		return sectionMap[sectionOrder[i]].bestScore > sectionMap[sectionOrder[j]].bestScore
	})

	// build visible rows: section header + ranked fields
	for _, si := range sectionOrder {
		sg := sectionMap[si]
		c.visible = append(c.visible, row{isSection: true, sectionIdx: si, fieldIdx: -1})
		for _, rf := range sg.fields {
			c.visible = append(c.visible, row{sectionIdx: si, fieldIdx: rf.flatIdx})
		}
	}

	// build search match indices (all field rows are matches)
	for vi, r := range c.visible {
		if !r.isSection {
			c.searchMatches = append(c.searchMatches, vi)
		}
	}
	if c.searchIdx >= len(c.searchMatches) {
		c.searchIdx = 0
	}

	// clamp cursor
	if c.cursor >= len(c.visible) {
		c.cursor = len(c.visible) - 1
	}
	if c.cursor < 0 {
		c.cursor = 0
	}
	if c.visible[c.cursor].isSection {
		c.skipSectionHeaders(1)
	}
}

// skipSectionHeaders advances the cursor past section header rows in the given direction.
func (c *content) skipSectionHeaders(dir int) {
	for c.cursor >= 0 && c.cursor < len(c.visible) && c.visible[c.cursor].isSection {
		c.cursor += dir
	}
	if c.cursor < 0 {
		c.cursor = 0
	}
	if c.cursor >= len(c.visible) {
		c.cursor = len(c.visible) - 1
	}
}
