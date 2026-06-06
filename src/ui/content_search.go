package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/eminert/konfi/pkg"
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

func (c *content) changedFieldKeys() map[string]bool {
	changed := make(map[string]bool)
	for _, change := range c.pendingChanges() {
		changed[change.Key] = true
	}
	return changed
}

// clearTopFilter clears the highest-priority active filter (bookmarks, then
// effective, new, changed, configured) and refilters. cleared reports whether
// any filter was active; clearStatus reports whether the caller should also
// reset the status line (only the filters that set a status message on toggle).
func (c *content) clearTopFilter() (cleared, clearStatus bool) {
	switch {
	case c.bookmarkedOnly:
		c.bookmarkedOnly = false
		clearStatus = true
	case c.showEffective:
		c.showEffective = false
	case c.showNewOnly:
		c.showNewOnly = false
		clearStatus = true
	case c.changedOnly:
		c.changedOnly = false
		clearStatus = true
	case c.configuredOnly:
		c.configuredOnly = false
	default:
		return false, false
	}
	c.refilter()
	c.syncDetail()
	return true, clearStatus
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
	var changedKeys map[string]bool
	if c.changedOnly {
		changedKeys = c.changedFieldKeys()
	}

	for i := range c.fields {
		f := &c.fields[i]
		si := c.fieldSection[i]
		if changedKeys != nil && !changedKeys[f.Key] {
			continue
		}
		if c.configuredOnly && !c.showEffective {
			if _, ok := c.values[f.Key]; !ok {
				continue
			}
		}
		if c.showNewOnly {
			if f.Since == "" {
				continue
			}
			if c.konfable != nil {
				ver := c.versions[c.konfable.Name()]
				if pkg.NormalizeSemver(ver) != "" && pkg.NormalizeSemver(f.Since) != "" && !pkg.FieldIsNewIn(*f, ver) {
					continue
				}
			}
		}
		if c.bookmarkedOnly && c.konfable != nil {
			if !c.bookmarks[c.konfable.Name()+"/"+f.Key] {
				continue
			}
		}
		// insert section header before first field of each section
		if si != lastHeaderSection {
			c.visible = append(c.visible, row{isSection: true, sectionIdx: si, fieldIdx: -1})
			lastHeaderSection = si
		}
		if c.collapsed[si] {
			continue
		}
		c.visible = append(c.visible, row{sectionIdx: si, fieldIdx: i})
	}

	// clear search matches (empty-query path, ranked path handles its own)
	c.searchMatches = c.searchMatches[:0]
	c.searchIdx = 0
	c.searchMatchInfo = nil

	// clamp cursor
	if len(c.visible) == 0 {
		c.cursor = 0
	} else if c.cursor >= len(c.visible) {
		c.cursor = len(c.visible) - 1
	}
	if c.cursor < 0 {
		c.cursor = 0
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

	// precompute section offsets for O(1) flat index lookup
	offsets := make([]int, len(c.schema.Sections))
	offset := 0
	for i, sec := range c.schema.Sections {
		offsets[i] = offset
		offset += len(sec.Fields)
	}

	// apply pre-filters and map to flat field indices
	type rankedField struct {
		flatIdx    int
		sectionIdx int
		score      float64
		matchInfo  string
	}
	var filtered []rankedField
	var changedKeys map[string]bool
	if c.changedOnly {
		changedKeys = c.changedFieldKeys()
	}
	for _, r := range results {
		flatIdx := offsets[r.SectionIdx] + r.FieldIdx

		f := &c.fields[flatIdx]
		if changedKeys != nil && !changedKeys[f.Key] {
			continue
		}
		if c.configuredOnly && !c.showEffective {
			if _, ok := c.values[f.Key]; !ok {
				continue
			}
		}
		if c.showNewOnly {
			if f.Since == "" {
				continue
			}
			if c.konfable != nil {
				ver := c.versions[c.konfable.Name()]
				if pkg.NormalizeSemver(ver) != "" && pkg.NormalizeSemver(f.Since) != "" && !pkg.FieldIsNewIn(*f, ver) {
					continue
				}
			}
		}
		if c.bookmarkedOnly && c.konfable != nil {
			if !c.bookmarks[c.konfable.Name()+"/"+f.Key] {
				continue
			}
		}
		filtered = append(filtered, rankedField{
			flatIdx:    flatIdx,
			sectionIdx: r.SectionIdx,
			score:      r.Score,
			matchInfo:  r.MatchInfo,
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
	c.searchMatchInfo = make(map[int]string)
	for _, si := range sectionOrder {
		sg := sectionMap[si]
		c.visible = append(c.visible, row{isSection: true, sectionIdx: si, fieldIdx: -1})
		if c.collapsed[si] {
			continue
		}
		for _, rf := range sg.fields {
			vi := len(c.visible)
			c.visible = append(c.visible, row{sectionIdx: si, fieldIdx: rf.flatIdx})
			if rf.matchInfo != "" {
				c.searchMatchInfo[vi] = rf.matchInfo
			}
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
}
