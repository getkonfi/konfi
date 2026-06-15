package ui

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/eminert/konfi/pkg"
	"github.com/eminert/konfi/theme"
)

const (
	hypridleAppName           = "hypridle"
	hypridleListenersKey      = "listeners"
	hypridleListenerSeparator = " <-> "
)

type hypridleRowKind int

const (
	hypridleRowListener hypridleRowKind = iota
	hypridleRowHook
	hypridleRowInhibitor
)

type hypridleListener struct {
	timeout       string
	onTimeout     string
	onResume      string
	ignoreInhibit string
}

type hypridleRow struct {
	kind        hypridleRowKind
	fieldIdx    int
	label       string
	value       string
	listener    hypridleListener
	listenerIdx int
}

type hypridleCheck struct {
	ok   bool
	text string
}

func (c *content) hypridleDashboardActive() bool {
	return c.konfable != nil && c.konfable.Name() == hypridleAppName && c.schema != nil
}

func (r hypridleRow) sectionLabel() string {
	switch r.kind {
	case hypridleRowListener:
		return "idle listeners"
	case hypridleRowHook:
		return "general hooks"
	case hypridleRowInhibitor:
		return "inhibitor policy"
	default:
		return ""
	}
}

func (c *content) currentFieldIndex() int {
	if c.hypridleDashboardActive() {
		rows := c.hypridleRows()
		if c.cursor < 0 || c.cursor >= len(rows) {
			return -1
		}
		return rows[c.cursor].fieldIdx
	}
	if len(c.visible) == 0 || c.cursor < 0 || c.cursor >= len(c.visible) {
		return -1
	}
	r := c.visible[c.cursor]
	if r.isSection {
		return -1
	}
	return r.fieldIdx
}

func (c *content) fieldIndexByKey(key string) (int, bool) {
	for i := range c.fields {
		if c.fields[i].Key == key {
			return i, true
		}
	}
	return -1, false
}

func (c *content) currentHypridleRow() (hypridleRow, bool) {
	if !c.hypridleDashboardActive() {
		return hypridleRow{}, false
	}
	rows := c.hypridleRows()
	if c.cursor < 0 || c.cursor >= len(rows) {
		return hypridleRow{}, false
	}
	return rows[c.cursor], true
}

func (c *content) clampHypridleCursor() {
	if !c.hypridleDashboardActive() {
		return
	}
	rows := c.hypridleRows()
	if len(rows) == 0 {
		c.cursor = 0
		return
	}
	if c.cursor < 0 {
		c.cursor = 0
	}
	if c.cursor >= len(rows) {
		c.cursor = len(rows) - 1
	}
}

func (c content) updateHypridleSearch(msg tea.Msg) (content, tea.Cmd) {
	km, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return c, nil
	}

	switch km.String() {
	case "esc":
		c.searching = false
		c.search.SetValue("")
		c.search.Blur()
		c.clearHypridleSearchMatches()
		c.clampHypridleCursor()
		c.syncDetail()
		return c, nil
	case "enter":
		c.searching = false
		c.search.Blur()
		c.refreshHypridleSearchMatches()
		c.clampHypridleCursor()
		c.syncDetail()
		return c, nil
	case "down":
		c.moveHypridleCursor(1)
		c.syncDetail()
		return c, nil
	case "up":
		c.moveHypridleCursor(-1)
		c.syncDetail()
		return c, nil
	default:
		var cmd tea.Cmd
		c.search, cmd = c.search.Update(msg)
		c.refreshHypridleSearchMatches()
		c.clampHypridleCursor()
		c.syncDetail()
		return c, cmd
	}
}

func (c content) updateHypridleDashboard(msg tea.KeyPressMsg) (content, tea.Cmd) {
	if !c.focused {
		return c, nil
	}

	switch msg.String() {
	case "space", "enter":
		if f := c.currentField(); f != nil {
			if f.Type == "bool" {
				settingCmd := c.toggleBool(*f)
				errCmd := c.drainErr()
				return c, tea.Batch(settingCmd, errCmd)
			}
			cmd := c.openEditor()
			return c, cmd
		}
	case "a":
		if idx, ok := c.firstHypridleListenerRow(); ok {
			c.cursor = idx
		}
		cmd := c.openEditor()
		return c, cmd
	case "f":
		c.configuredOnly = !c.configuredOnly
		c.refreshHypridleSearchMatches()
		c.clampHypridleCursor()
		c.syncDetail()
	case "g":
		c.showEffective = !c.showEffective
		c.refreshHypridleSearchMatches()
		c.clampHypridleCursor()
		c.syncDetail()
	case "/":
		c.searching = true
		c.search.SetValue("")
		c.clearHypridleSearchMatches()
		return c, c.search.Focus()
	case "n":
		c.nextHypridleSearchMatch(1)
		c.syncDetail()
	case "N":
		c.nextHypridleSearchMatch(-1)
		c.syncDetail()
	case "j", "down":
		c.moveHypridleCursor(1)
		c.syncDetail()
	case "k", "up":
		c.moveHypridleCursor(-1)
		c.syncDetail()
	case "home":
		c.cursor = 0
		c.syncDetail()
	case "end":
		rows := c.hypridleRows()
		if len(rows) > 0 {
			c.cursor = len(rows) - 1
		}
		c.syncDetail()
	case "pgdown":
		c.moveHypridleCursor(c.pageSize())
		c.syncDetail()
	case "pgup":
		c.moveHypridleCursor(-c.pageSize())
		c.syncDetail()
	case "backspace", "delete", "d":
		row, ok := c.currentHypridleRow()
		if ok && row.kind == hypridleRowListener {
			return c, func() tea.Msg {
				return StatusMsg{Text: "edit listeners to delete a listener row"}
			}
		}
		if f := c.currentField(); f != nil && c.konfable != nil && c.config != nil {
			if _, hasCur := c.values[f.Key]; hasCur {
				c.deleteField(*f)
			}
			cmd := c.drainErr()
			return c, cmd
		}
	case "o":
		if url := c.currentDocURL(); url != "" {
			return c, c.openDocs(url)
		}
	default:
		if f := c.currentField(); f != nil && msg.Text != "" {
			if f.Type != "bool" && f.Type != "enum" && f.Type != "list" && f.Type != "multi" && f.Widget == "" {
				r, _ := utf8.DecodeRuneInString(msg.Text)
				cmd := c.openEditorWithSeed(r)
				return c, cmd
			}
		}
	}

	return c, nil
}

func (c *content) moveHypridleCursor(delta int) {
	rows := c.hypridleRows()
	if len(rows) == 0 {
		c.cursor = 0
		return
	}
	c.cursor += delta
	if c.cursor < 0 {
		c.cursor = 0
	}
	if c.cursor >= len(rows) {
		c.cursor = len(rows) - 1
	}
}

func (c *content) firstHypridleListenerRow() (int, bool) {
	rows := c.hypridleRows()
	for i, row := range rows {
		if row.kind == hypridleRowListener {
			return i, true
		}
	}
	return 0, false
}

func (c *content) clearHypridleSearchMatches() {
	c.searchMatches = c.searchMatches[:0]
	c.searchIdx = 0
	c.searchMatchInfo = nil
}

func (c *content) refreshHypridleSearchMatches() {
	query := strings.TrimSpace(c.search.Value())
	if query == "" {
		c.clearHypridleSearchMatches()
		return
	}
	rows := c.hypridleRows()
	c.searchMatches = c.searchMatches[:0]
	for i := range rows {
		c.searchMatches = append(c.searchMatches, i)
	}
	if c.searchIdx >= len(c.searchMatches) {
		c.searchIdx = 0
	}
}

func (c *content) nextHypridleSearchMatch(delta int) {
	if len(c.searchMatches) == 0 {
		c.refreshHypridleSearchMatches()
	}
	if len(c.searchMatches) == 0 {
		return
	}
	c.searchIdx += delta
	if c.searchIdx < 0 {
		c.searchIdx = len(c.searchMatches) - 1
	}
	if c.searchIdx >= len(c.searchMatches) {
		c.searchIdx = 0
	}
	c.cursor = c.searchMatches[c.searchIdx]
}

func (c *content) hypridleRows() []hypridleRow {
	if !c.hypridleDashboardActive() {
		return nil
	}

	var rows []hypridleRow
	if idx, ok := c.fieldIndexByKey(hypridleListenersKey); ok {
		listeners := parseHypridleListeners(c.values[hypridleListenersKey])
		if len(listeners) == 0 {
			rows = append(rows, hypridleRow{
				kind:        hypridleRowListener,
				fieldIdx:    idx,
				label:       "no listeners configured",
				listenerIdx: -1,
			})
		} else {
			for i, listener := range listeners {
				rows = append(rows, hypridleRow{
					kind:        hypridleRowListener,
					fieldIdx:    idx,
					label:       fmt.Sprintf("listener %d", i+1),
					listener:    listener,
					listenerIdx: i,
				})
			}
		}
	}

	for _, key := range []string{
		"general.lock_cmd",
		"general.unlock_cmd",
		"general.on_lock_cmd",
		"general.on_unlock_cmd",
		"general.before_sleep_cmd",
		"general.after_sleep_cmd",
	} {
		idx, ok := c.fieldIndexByKey(key)
		if !ok {
			continue
		}
		rows = append(rows, hypridleRow{
			kind:     hypridleRowHook,
			fieldIdx: idx,
			label:    hypridleHookLabel(key),
			value:    c.values[key],
		})
	}

	for _, key := range []string{
		"general.ignore_dbus_inhibit",
		"general.ignore_systemd_inhibit",
		"general.ignore_wayland_inhibit",
		"general.inhibit_sleep",
	} {
		idx, ok := c.fieldIndexByKey(key)
		if !ok {
			continue
		}
		rows = append(rows, hypridleRow{
			kind:     hypridleRowInhibitor,
			fieldIdx: idx,
			label:    hypridleInhibitorLabel(key),
			value:    c.hypridleFieldValue(key),
		})
	}

	return c.filterHypridleRows(rows)
}

func (c *content) filterHypridleRows(rows []hypridleRow) []hypridleRow {
	query := strings.ToLower(strings.TrimSpace(c.search.Value()))
	out := rows[:0]
	for _, row := range rows {
		if row.fieldIdx < 0 || row.fieldIdx >= len(c.fields) {
			continue
		}
		f := &c.fields[row.fieldIdx]
		if c.configuredOnly && !c.showEffective {
			if _, ok := c.values[f.Key]; !ok {
				continue
			}
		}
		if c.changedOnly && !c.fieldChanged(f) {
			continue
		}
		if c.bookmarkedOnly && c.konfable != nil {
			if !c.bookmarks[c.konfable.Name()+"/"+f.Key] {
				continue
			}
		}
		if query != "" && !hypridleRowMatches(row, *f, query) {
			continue
		}
		out = append(out, row)
	}
	return out
}

func hypridleRowMatches(row hypridleRow, f pkg.Field, query string) bool {
	haystack := strings.ToLower(strings.Join([]string{
		f.Key,
		f.Label,
		f.Description,
		f.Hint,
		row.label,
		row.value,
		row.listener.timeout,
		row.listener.onTimeout,
		row.listener.onResume,
		row.listener.ignoreInhibit,
	}, " "))
	return strings.Contains(haystack, query)
}

func parseHypridleListeners(value string) []hypridleListener {
	var listeners []hypridleListener
	for _, line := range strings.Split(value, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, hypridleListenerSeparator, 4)
		for len(parts) < 4 {
			parts = append(parts, "")
		}
		listener := hypridleListener{
			timeout:       strings.TrimSpace(parts[0]),
			onTimeout:     strings.TrimSpace(parts[1]),
			onResume:      strings.TrimSpace(parts[2]),
			ignoreInhibit: strings.TrimSpace(parts[3]),
		}
		if listener.timeout == "" && listener.onTimeout == "" && listener.onResume == "" && listener.ignoreInhibit == "" {
			continue
		}
		listeners = append(listeners, listener)
	}
	return listeners
}

func (c *content) viewHypridleDashboard(outerStyle lipgloss.Style, innerW, bodyH int) string {
	c.clampHypridleCursor()

	headerStr := c.renderHeader(innerW)

	c.breadcrumb.SetWidth(innerW)
	crumbStr := c.breadcrumb.View()
	if crumbStr != "" {
		crumbStr += "\n"
		bodyH--
		if bodyH < 3 {
			bodyH = 3
		}
	}

	body := c.renderHypridleDashboardBody(innerW)
	if line := c.hypridleCursorLine(innerW); line >= 0 {
		if line < c.scrollY {
			c.scrollY = line
		}
		if line >= c.scrollY+bodyH {
			c.scrollY = line - bodyH + 1
		}
	}
	lines := strings.Split(body, "\n")
	if c.scrollY >= len(lines) {
		c.scrollY = max(0, len(lines)-1)
	}
	if c.scrollY > 0 && c.scrollY < len(lines) {
		lines = lines[c.scrollY:]
	}
	if len(lines) > bodyH {
		lines = lines[:bodyH]
	}

	return outerStyle.Render(headerStr + crumbStr + strings.Join(lines, "\n"))
}

func (c *content) hypridleCursorLine(width int) int {
	rows := c.hypridleRows()
	if len(rows) == 0 || c.cursor < 0 || c.cursor >= len(rows) {
		return -1
	}

	line := 0
	if c.searching || strings.TrimSpace(c.search.Value()) != "" {
		line++
	}
	if c.filterIndicatorVisible() {
		line++
	}

	lastKind := hypridleRowKind(-1)
	for idx, row := range rows {
		if row.kind != lastKind {
			if lastKind != hypridleRowKind(-1) {
				line++
			}
			line++
			if row.kind == hypridleRowListener && width >= 72 {
				line++
			}
			lastKind = row.kind
		}
		if idx == c.cursor {
			return line
		}
		line += hypridleRowHeight(row, width)
	}
	return -1
}

func hypridleRowHeight(row hypridleRow, width int) int {
	if row.kind == hypridleRowListener && row.listenerIdx >= 0 && width < 72 {
		return 4
	}
	return 1
}

func (c *content) renderHypridleDashboardBody(width int) string {
	rows := c.hypridleRows()
	var b strings.Builder

	if c.searching || strings.TrimSpace(c.search.Value()) != "" {
		prompt := c.theme.Primary.Render("/ ")
		var countStr string
		if strings.TrimSpace(c.search.Value()) != "" {
			countStr = c.theme.Muted.Render(fmt.Sprintf("  %d/%d matches", selectedPosition(c.cursor, rows), len(rows)))
		} else {
			countStr = c.theme.Muted.Render(fmt.Sprintf("  %d rows", len(rows)))
		}
		if c.searching {
			b.WriteString(prompt + c.search.View() + countStr)
		} else {
			b.WriteString(prompt + c.theme.Subtext.Render(c.search.Value()) + countStr)
		}
		b.WriteByte('\n')
	}

	if c.filterIndicatorVisible() {
		var labels []string
		if c.bookmarkedOnly {
			labels = append(labels, "bookmarks")
		}
		if c.showEffective {
			labels = append(labels, "effective")
		}
		if c.changedOnly {
			labels = append(labels, "changed")
		}
		if c.configuredOnly {
			labels = append(labels, "configured")
		}
		b.WriteString(c.theme.Warning.Render("▸ " + strings.Join(labels, " + ")))
		b.WriteByte('\n')
	}

	if c.changedOnly && len(rows) == 0 && len(c.pendingChanges()) == 0 {
		b.WriteString(c.theme.Muted.Render("no unsaved changes"))
		b.WriteByte('\n')
		return b.String()
	}

	c.renderHypridleListeners(&b, width, rows)
	c.renderHypridleFields(&b, width, rows, hypridleRowHook, "general hooks")
	c.renderHypridleFields(&b, width, rows, hypridleRowInhibitor, "inhibitor policy")
	c.renderHypridleChecks(&b, width)

	return strings.TrimRight(b.String(), "\n")
}

func selectedPosition(cursor int, rows []hypridleRow) int {
	if len(rows) == 0 {
		return 0
	}
	if cursor < 0 {
		return 1
	}
	if cursor >= len(rows) {
		return len(rows)
	}
	return cursor + 1
}

func (c *content) renderHypridleListeners(b *strings.Builder, width int, rows []hypridleRow) {
	listenerRows := filterHypridleRowsByKind(rows, hypridleRowListener)
	if len(listenerRows) == 0 {
		return
	}

	b.WriteString(c.hypridleSectionHeader("idle listeners", width))
	b.WriteByte('\n')
	if width >= 72 {
		b.WriteString(c.theme.Muted.Render("   timeout  on-timeout command"))
		if width >= 96 {
			b.WriteString(c.theme.Muted.Render(strings.Repeat(" ", 19) + "on-resume command"))
		}
		b.WriteString(c.theme.Muted.Render("  inhibitors"))
		b.WriteByte('\n')
	}

	for idx, row := range rows {
		if row.kind != hypridleRowListener {
			continue
		}
		b.WriteString(c.renderHypridleListenerRow(row, idx, width))
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
}

func (c *content) renderHypridleFields(b *strings.Builder, width int, rows []hypridleRow, kind hypridleRowKind, title string) {
	sectionRows := filterHypridleRowsByKind(rows, kind)
	if len(sectionRows) == 0 {
		return
	}

	b.WriteString(c.hypridleSectionHeader(title, width))
	b.WriteByte('\n')
	for idx, row := range rows {
		if row.kind != kind {
			continue
		}
		b.WriteString(c.renderHypridleFieldRow(row, idx, width))
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
}

func filterHypridleRowsByKind(rows []hypridleRow, kind hypridleRowKind) []hypridleRow {
	var out []hypridleRow
	for _, row := range rows {
		if row.kind == kind {
			out = append(out, row)
		}
	}
	return out
}

func (c *content) hypridleSectionHeader(title string, width int) string {
	header := c.theme.Primary.Bold(true).Render("── " + title + " ")
	if pad := width - lipgloss.Width(header); pad > 0 {
		header += c.theme.Muted.Faint(true).Render(strings.Repeat("─", pad))
	}
	return header
}

func (c *content) renderHypridleListenerRow(row hypridleRow, rowIdx, width int) string {
	marker := "  "
	if c.fieldListFocused() && rowIdx == c.cursor {
		marker = c.theme.Primary.Render("▎ ")
	}

	if row.listenerIdx < 0 {
		text := c.theme.Muted.Render("no listeners configured")
		return marker + text + c.theme.Muted.Render("  press a to edit listener rules")
	}

	timeout := row.listener.timeout
	if timeout == "" {
		timeout = "-"
	} else if _, err := strconv.Atoi(timeout); err == nil {
		timeout += "s"
	}
	onTimeout := hypridleDisplayCommand(row.listener.onTimeout)
	onResume := hypridleDisplayCommand(row.listener.onResume)
	inhibit := hypridleListenerInhibitLabel(row.listener.ignoreInhibit)

	if width < 72 {
		lines := []string{
			marker + c.theme.FieldLabel.Render(row.label) + " " + c.theme.FieldValue.Render(timeout),
			"    on-timeout " + c.theme.FieldValue.Render(onTimeout),
			"    on-resume  " + c.theme.FieldValue.Render(onResume),
			"    inhibitors " + c.theme.FieldValue.Render(inhibit),
		}
		return strings.Join(lines, "\n")
	}

	timeoutW := 8
	inhibitW := 10
	resumeW := 0
	if width >= 96 {
		resumeW = 28
	}
	actionW := width - lipgloss.Width(marker) - timeoutW - inhibitW - resumeW - 6
	if actionW < 18 {
		actionW = 18
	}

	parts := []string{
		marker,
		c.theme.FieldValue.Render(fitCell(timeout, timeoutW)),
		c.theme.FieldValue.Render(fitCell(onTimeout, actionW)),
	}
	if resumeW > 0 {
		parts = append(parts, c.theme.FieldValue.Render(fitCell(onResume, resumeW)))
	}
	parts = append(parts, c.theme.FieldValue.Render(fitCell(inhibit, inhibitW)))
	return strings.Join(parts, " ")
}

func (c *content) renderHypridleFieldRow(row hypridleRow, rowIdx, width int) string {
	marker := "  "
	if c.fieldListFocused() && rowIdx == c.cursor {
		marker = c.theme.Primary.Render("▎ ")
	}

	f := &c.fields[row.fieldIdx]
	configured := false
	if _, ok := c.values[f.Key]; ok {
		configured = true
	}
	dot := c.fieldStateDot(configured, c.fieldChanged(f), f.Until != "")

	labelW := 22
	if width < 70 {
		labelW = 18
	}
	value := row.value
	if row.kind == hypridleRowHook {
		value = hypridleDisplayCommand(value)
	}
	maxValueW := width - lipgloss.Width(marker) - labelW - lipgloss.Width(dot) - 3
	if maxValueW < 8 {
		maxValueW = 8
	}

	label := c.fieldLabelStyle(c.fieldListFocused() && rowIdx == c.cursor, c.fieldChanged(f), f.Until != "").Render(fitCell(row.label, labelW))
	valueStyle := c.theme.FieldValue
	if value == "-" {
		valueStyle = c.theme.FieldDefault
	}
	return marker + label + " " + dot + " " + valueStyle.Render(theme.Truncate(value, maxValueW))
}

func fitCell(s string, width int) string {
	s = theme.Truncate(s, width)
	if pad := width - lipgloss.Width(s); pad > 0 {
		s += strings.Repeat(" ", pad)
	}
	return s
}

func hypridleDisplayCommand(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	return value
}

func hypridleListenerInhibitLabel(value string) string {
	if strings.EqualFold(strings.TrimSpace(value), "true") {
		return "ignore"
	}
	return "respect"
}

func (c *content) hypridleFieldValue(key string) string {
	idx, ok := c.fieldIndexByKey(key)
	if !ok {
		return "-"
	}
	f := &c.fields[idx]
	value, configured := c.values[key]
	if !configured || strings.TrimSpace(value) == "" {
		value = f.Default
	}
	switch key {
	case "general.ignore_dbus_inhibit", "general.ignore_systemd_inhibit", "general.ignore_wayland_inhibit":
		if strings.EqualFold(strings.TrimSpace(value), "true") {
			return "ignored"
		}
		return "respected"
	case "general.inhibit_sleep":
		return hypridleSleepInhibitLabel(value)
	default:
		if strings.TrimSpace(value) == "" {
			return "-"
		}
		return value
	}
}

func hypridleSleepInhibitLabel(value string) string {
	switch strings.TrimSpace(value) {
	case "0":
		return "disabled"
	case "1":
		return "wait before_sleep"
	case "2", "":
		return "auto"
	case "3":
		return "wait for lock"
	default:
		return value
	}
}

func hypridleHookLabel(key string) string {
	switch key {
	case "general.lock_cmd":
		return "lock_cmd"
	case "general.unlock_cmd":
		return "unlock_cmd"
	case "general.on_lock_cmd":
		return "on_lock_cmd"
	case "general.on_unlock_cmd":
		return "on_unlock_cmd"
	case "general.before_sleep_cmd":
		return "before_sleep_cmd"
	case "general.after_sleep_cmd":
		return "after_sleep_cmd"
	default:
		return key
	}
}

func hypridleInhibitorLabel(key string) string {
	switch key {
	case "general.ignore_dbus_inhibit":
		return "dbus inhibitors"
	case "general.ignore_systemd_inhibit":
		return "systemd inhibitors"
	case "general.ignore_wayland_inhibit":
		return "wayland inhibitors"
	case "general.inhibit_sleep":
		return "sleep protection"
	default:
		return key
	}
}

func (c *content) renderHypridleChecks(b *strings.Builder, width int) {
	checks := c.hypridleChecks()
	if len(checks) == 0 {
		return
	}
	b.WriteString(c.hypridleSectionHeader("checks", width))
	b.WriteByte('\n')
	for _, check := range checks {
		prefix := c.theme.Success.Render("✓ ")
		style := c.theme.Muted
		if !check.ok {
			prefix = c.theme.Warning.Render("! ")
			style = c.theme.Warning
		}
		line := prefix + style.Render(theme.Truncate(check.text, max(8, width-2)))
		b.WriteString(line)
		b.WriteByte('\n')
	}
}

func (c *content) hypridleChecks() []hypridleCheck {
	var checks []hypridleCheck
	listeners := parseHypridleListeners(c.values[hypridleListenersKey])
	if len(listeners) == 0 {
		checks = append(checks, hypridleCheck{text: "no idle listeners configured"})
	}

	hasDisplayOff := false
	hasDisplayResume := false
	for _, listener := range listeners {
		if strings.TrimSpace(listener.timeout) == "" {
			checks = append(checks, hypridleCheck{text: "listener is missing timeout"})
		}
		if strings.TrimSpace(listener.onTimeout) == "" {
			checks = append(checks, hypridleCheck{text: "listener is missing on-timeout command"})
		}
		if isHypridleDisplayOff(listener.onTimeout) {
			hasDisplayOff = true
			if strings.TrimSpace(listener.onResume) == "" {
				checks = append(checks, hypridleCheck{text: "display-off listener has no on-resume command"})
			}
		}
		if isHypridleDisplayOn(listener.onResume) {
			hasDisplayResume = true
		}
	}

	if hasDisplayOff && !hasDisplayResume && strings.TrimSpace(c.values["general.after_sleep_cmd"]) == "" {
		checks = append(checks, hypridleCheck{text: "display-off setup has no obvious display-on recovery hook"})
	}
	for _, key := range []string{
		"general.ignore_dbus_inhibit",
		"general.ignore_systemd_inhibit",
		"general.ignore_wayland_inhibit",
	} {
		if strings.EqualFold(strings.TrimSpace(c.values[key]), "true") {
			checks = append(checks, hypridleCheck{text: hypridleInhibitorLabel(key) + " are ignored"})
		}
	}
	if strings.TrimSpace(c.values["general.inhibit_sleep"]) == "1" && strings.TrimSpace(c.values["general.before_sleep_cmd"]) == "" {
		checks = append(checks, hypridleCheck{text: "sleep protection waits for before_sleep_cmd, but it is not set"})
	}

	if len(checks) == 0 {
		checks = append(checks, hypridleCheck{ok: true, text: "no local issues found"})
	}
	return checks
}

func isHypridleDisplayOff(command string) bool {
	command = strings.ToLower(command)
	return strings.Contains(command, "dpms off") ||
		strings.Contains(command, "action = \"off\"") ||
		strings.Contains(command, "action=\"off\"") ||
		strings.Contains(command, "action = \"disable\"") ||
		strings.Contains(command, "action=\"disable\"")
}

func isHypridleDisplayOn(command string) bool {
	command = strings.ToLower(command)
	return strings.Contains(command, "dpms on") ||
		strings.Contains(command, "action = \"on\"") ||
		strings.Contains(command, "action=\"on\"") ||
		strings.Contains(command, "action = \"enable\"") ||
		strings.Contains(command, "action=\"enable\"")
}
