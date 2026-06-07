package pkg

import (
	"strconv"
	"strings"
)

// block engine: models a config file as a sequence of named blocks (e.g. ssh
// Host/Match) with ordered, repeatable, comment-preserving bodies. it can
// reconcile an edited model back to bytes losslessly — untouched lines stay
// byte-identical.
//
// byte-stability strategy: every Raw / RawSpan slice element is the exact
// original source line *including* its line terminator ("\n", "\r\n", or "" for
// a final line with no trailing newline). re-emitting an unchanged element is
// therefore trivially byte-identical, and eol style / trailing-newline are
// preserved with no extra bookkeeping.

// BlockModel is the engine's view of a file: only the named blocks, in order.
// root-level directives and non-named blocks are NOT carried here; they live in
// the bytes and are recovered at reconcile time.
type BlockModel struct {
	Blocks []Block
}

// Block is one named block (a Host/Match block). identity is POSITIONAL via ID,
// never the header string — two "Host foo" blocks are two distinct blocks.
type Block struct {
	ID      string   // positional/stable id assigned at Parse ("b0","b1",…)
	Opener  string   // "Host" or "Match" (canonical-cased)
	Header  string   // raw header text after the opener keyword, verbatim
	RawSpan []string // verbatim lines (with terminators) incl. owned leading run
	Body    []Entry
}

// Entry is one body element of a block. repeatable directives are SEPARATE
// entries (not merged), one per occurrence.
type Entry struct {
	ID     string   // stable id within the block ("e0","e1",…)
	Kind   string   // "directive" | "comment" | "blank" | "other"
	Key    string   // canonical directive key for "directive", else ""
	Values []string // directive value tokens; empty for non-directives
	Raw    []string // verbatim source line(s) (with terminators) for this entry
}

// PlacementRule controls how non-named (flat) blocks are positioned when the
// named region is rewritten during a reorder. it is supplied by the caller; the
// engine bakes in no app-specific assumption (e.g. ssh "Host *").
//
// IsLowPrecedence identifies a flat block that must stay at lowest precedence
// (rendered after the named region, and new named blocks insert before it).
// a nil predicate means "no flat block is low-precedence" (no-op default).
type PlacementRule struct {
	IsLowPrecedence func(opener, header string) bool
}

// rawLine is one physical line with its terminator split out, so we can rejoin
// exactly. eol is "\n", "\r\n", or "" (final line, no trailing newline).
type rawLine struct {
	text string // line content without terminator
	eol  string // terminator as it appeared in source
}

// splitRawLines splits data into physical lines preserving each terminator and
// whether the file ends without a trailing newline.
func splitRawLines(data []byte) []rawLine {
	s := string(data)
	if s == "" {
		return nil
	}
	var lines []rawLine
	i := 0
	for i < len(s) {
		nl := strings.IndexByte(s[i:], '\n')
		if nl < 0 {
			// final line, no trailing newline
			lines = append(lines, rawLine{text: s[i:], eol: ""})
			break
		}
		end := i + nl
		text := s[i:end]
		eol := "\n"
		if text != "" && text[len(text)-1] == '\r' {
			text = text[:len(text)-1]
			eol = "\r\n"
		}
		lines = append(lines, rawLine{text: text, eol: eol})
		i = end + 1
	}
	return lines
}

// full reconstructs the original substring for a physical line (text+eol).
func (l rawLine) full() string { return l.text + l.eol }

// lineKind classifies a physical line by its content (terminator ignored).
func lineKind(text string) (kind, key string, values []string) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return "blank", "", nil
	}
	if trimmed[0] == '#' {
		return "comment", "", nil
	}
	k, v, ok := parseBlockLine(text)
	if !ok {
		return "other", "", nil
	}
	return "directive", k, splitValues(v)
}

// parseBlockLine mirrors the ssh parser's parseSSHLine: "key value" and
// "key = value" forms, indentation-insensitive.
func parseBlockLine(line string) (key, value string, ok bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || trimmed[0] == '#' {
		return "", "", false
	}
	if eqIdx := strings.IndexByte(trimmed, '='); eqIdx >= 0 {
		k := strings.TrimSpace(trimmed[:eqIdx])
		if k != "" && !strings.ContainsAny(k, " \t") {
			return k, strings.TrimSpace(trimmed[eqIdx+1:]), true
		}
	}
	idx := strings.IndexAny(trimmed, " \t")
	if idx < 0 {
		return trimmed, "", true
	}
	return trimmed[:idx], strings.TrimSpace(trimmed[idx+1:]), true
}

// splitValues tokenizes a directive value, honoring double-quoted spans so that
// e.g. Match exec "some command" stays a single token. returns nil for empty.
func splitValues(v string) []string {
	if v == "" {
		return nil
	}
	var out []string
	var cur strings.Builder
	inQuote := false
	has := false
	for i := 0; i < len(v); i++ {
		c := v[i]
		switch {
		case c == '"':
			inQuote = !inQuote
			cur.WriteByte(c)
			has = true
		case (c == ' ' || c == '\t') && !inQuote:
			if has {
				out = append(out, cur.String())
				cur.Reset()
				has = false
			}
		default:
			cur.WriteByte(c)
			has = true
		}
	}
	if has {
		out = append(out, cur.String())
	}
	return out
}

// firstToken returns the lower-cased first whitespace/equals-delimited token.
func firstToken(text string) string {
	k, _, ok := parseBlockLine(text)
	if !ok {
		return ""
	}
	return strings.ToLower(k)
}

// isOpener reports whether text begins a block whose opener is in openers.
// returns the canonical-cased opener token from the source line.
func isOpener(text string, openers []string) (string, bool) {
	tok := firstToken(text)
	if tok == "" {
		return "", false
	}
	for _, o := range openers {
		if strings.EqualFold(o, tok) {
			// recover the verbatim-cased opener from the line
			trimmed := strings.TrimSpace(text)
			idx := strings.IndexAny(trimmed, " \t=")
			if idx < 0 {
				return trimmed, true
			}
			return trimmed[:idx], true
		}
	}
	return "", false
}

// headerOf extracts the raw header text after the opener keyword.
func headerOf(text string) string {
	trimmed := strings.TrimSpace(text)
	// skip opener token
	idx := strings.IndexAny(trimmed, " \t=")
	if idx < 0 {
		return ""
	}
	rest := trimmed[idx:]
	// drop a single leading separator run / optional '='
	rest = strings.TrimLeft(rest, " \t")
	if strings.HasPrefix(rest, "=") {
		rest = strings.TrimLeft(rest[1:], " \t")
	}
	return rest
}

// Parse scans data into a BlockModel. openers names the block keywords (e.g.
// {"Host","Match"}); isNamed filters which blocks the model exposes (a nil
// isNamed exposes every opener block).
//
// comment/blank ownership: a leading run of comment/blank lines immediately
// above a block opener (with no intervening directive) travels WITH the
// following block, so reordering or deleting the block carries that run.
func Parse(data []byte, openers []string, isNamed func(opener, header string) bool) BlockModel {
	lines := splitRawLines(data)
	var model BlockModel
	blockIdx := 0

	i := 0
	for i < len(lines) {
		opener, ok := isOpener(lines[i].text, openers)
		if !ok {
			i++
			continue
		}

		// pull back the leading comment/blank run that owns this opener.
		spanStart := i
		j := i - 1
		for j >= 0 {
			k, _, _ := lineKind(lines[j].text)
			if k == "comment" || k == "blank" {
				j--
				continue
			}
			break
		}
		runStart := j + 1

		// a run is owned only if it isn't already consumed by a prior named
		// block in the model (handled implicitly: parsing is left-to-right and
		// each opener claims the run since its last opener boundary). but a run
		// sitting between a previous block's last directive and this opener
		// belongs to this opener. so the owned run starts at runStart unless
		// runStart falls inside an already-emitted block span — which cannot
		// happen here because we advance i past each block we emit.
		_ = spanStart

		// find the end of this block: next opener line (any opener).
		end := len(lines)
		for n := i + 1; n < len(lines); n++ {
			if _, isOp := isOpener(lines[n].text, openers); isOp {
				end = n
				break
			}
		}

		// but the next block may own a trailing run of comment/blank lines that
		// precede it; those belong to the NEXT block, not this one. trim them
		// off this block's body span.
		bodyEnd := end
		if end < len(lines) {
			t := end - 1
			for t >= i+1 {
				k, _, _ := lineKind(lines[t].text)
				if k == "comment" || k == "blank" {
					t--
					continue
				}
				break
			}
			bodyEnd = t + 1
		}

		named := isNamed == nil || isNamed(opener, headerOf(lines[i].text))
		if named {
			blk := buildBlock(lines, runStart, i, bodyEnd, opener, blockIdx)
			model.Blocks = append(model.Blocks, blk)
			blockIdx++
		}

		i = end
	}

	return model
}

// buildBlock assembles a Block from physical line indices. runStart..openLine is
// the owned leading comment/blank run; openLine is the opener; openLine..bodyEnd
// is opener+body.
func buildBlock(lines []rawLine, runStart, openLine, bodyEnd int, opener string, idx int) Block {
	blk := Block{
		ID:     "b" + strconv.Itoa(idx),
		Opener: opener,
		Header: headerOf(lines[openLine].text),
	}

	span := make([]string, 0, bodyEnd-runStart)
	for n := runStart; n < bodyEnd; n++ {
		span = append(span, lines[n].full())
	}
	blk.RawSpan = span

	// body entries: the owned leading run (runStart..openLine) becomes leading
	// comment/blank entries; the opener line itself is an "other"-kind entry so
	// the whole span is reconstructable from Body alone; then body directives.
	entryIdx := 0
	for n := runStart; n < bodyEnd; n++ {
		if n == openLine {
			blk.Body = append(blk.Body, Entry{
				ID:   "e" + strconv.Itoa(entryIdx),
				Kind: "opener",
				Raw:  []string{lines[n].full()},
			})
			entryIdx++
			continue
		}
		kind, key, values := lineKind(lines[n].text)
		blk.Body = append(blk.Body, Entry{
			ID:     "e" + strconv.Itoa(entryIdx),
			Kind:   kind,
			Key:    key,
			Values: values,
			Raw:    []string{lines[n].full()},
		})
		entryIdx++
	}

	return blk
}

// inferIndent returns the leading whitespace to use for directives added to
// this block, taken from the first existing directive.
func (b Block) inferIndent() string {
	for _, e := range b.Body {
		if e.Kind != "directive" || len(e.Raw) == 0 {
			continue
		}
		raw := e.Raw[0]
		return raw[:len(raw)-len(strings.TrimLeft(raw, " \t"))]
	}
	return "    "
}

// inferEol returns the dominant terminator used in the block (for new lines).
func (b Block) inferEol() string {
	for _, e := range b.Body {
		if len(e.Raw) == 0 {
			continue
		}
		raw := e.Raw[0]
		if strings.HasSuffix(raw, "\r\n") {
			return "\r\n"
		}
		if strings.HasSuffix(raw, "\n") {
			return "\n"
		}
	}
	return "\n"
}

// inferUsesEquals reports whether the block writes directives as "key = value".
func (b Block) inferUsesEquals() bool {
	for _, e := range b.Body {
		if e.Kind != "directive" || len(e.Raw) == 0 {
			continue
		}
		trimmed := strings.TrimSpace(stripEol(e.Raw[0]))
		if eqIdx := strings.IndexByte(trimmed, '='); eqIdx >= 0 {
			k := strings.TrimSpace(trimmed[:eqIdx])
			if k != "" && !strings.ContainsAny(k, " \t") {
				return true
			}
		}
		return false
	}
	return false
}

func stripEol(s string) string {
	s = strings.TrimSuffix(s, "\n")
	return strings.TrimSuffix(s, "\r")
}

// renderEntry produces the physical line(s) for an entry. unchanged entries
// re-emit their verbatim Raw; directive value edits do a surgical single-line
// replace preserving indentation and key/value style.
func renderEntry(e Entry, indent, eol string, useEquals bool) []string {
	switch e.Kind {
	case "directive":
		return []string{renderDirective(e, indent, eol, useEquals)}
	default:
		if len(e.Raw) > 0 {
			return e.Raw
		}
		return nil
	}
}

// renderDirective emits a directive line. if Raw is present and still matches
// the model's key/values, it is returned verbatim (byte-stable); otherwise the
// line is rebuilt from Key+Values using the block's indent/style.
func renderDirective(e Entry, indent, eol string, useEquals bool) string {
	if len(e.Raw) == 1 && rawMatchesDirective(e.Raw[0], e.Key, e.Values) {
		return e.Raw[0]
	}
	// rebuild: preserve original indent if we have a Raw to learn from.
	ind := indent
	if len(e.Raw) == 1 {
		r := e.Raw[0]
		ind = r[:len(r)-len(strings.TrimLeft(r, " \t"))]
	}
	val := strings.Join(e.Values, " ")
	sep := " "
	if useEquals {
		sep = " = "
	}
	if len(e.Raw) == 1 {
		// honor the original line's own separator style
		trimmed := strings.TrimSpace(stripEol(e.Raw[0]))
		if eqIdx := strings.IndexByte(trimmed, '='); eqIdx >= 0 {
			k := strings.TrimSpace(trimmed[:eqIdx])
			if k != "" && !strings.ContainsAny(k, " \t") {
				sep = " = "
			} else {
				sep = " "
			}
		} else {
			sep = " "
		}
	}
	line := ind + e.Key + sep + val
	return line + eol
}

// rawMatchesDirective reports whether raw still represents key with values, so
// an unedited directive re-emits verbatim.
func rawMatchesDirective(raw, key string, values []string) bool {
	k, v, ok := parseBlockLine(stripEol(raw))
	if !ok || !strings.EqualFold(k, key) {
		return false
	}
	got := splitValues(v)
	if len(got) != len(values) {
		return false
	}
	for i := range got {
		if got[i] != values[i] {
			return false
		}
	}
	return true
}

// renderBlock produces a block's physical lines from its Body. unchanged blocks
// (whose Body still reconstructs RawSpan verbatim) re-emit RawSpan exactly.
func renderBlock(b Block) []string {
	if blockUnchanged(b) {
		return b.RawSpan
	}
	indent := b.inferIndent()
	eol := b.inferEol()
	useEquals := b.inferUsesEquals()
	var out []string
	for _, e := range b.Body {
		out = append(out, renderEntry(e, indent, eol, useEquals)...)
	}
	return out
}

// blockUnchanged reports whether the block can re-emit RawSpan verbatim, i.e.
// nothing was edited: Body's Raw lines still concatenate to RawSpan AND every
// directive's Raw still matches its current Key/Values (a value edit leaves Raw
// stale, which this catches).
func blockUnchanged(b Block) bool {
	var got []string
	for _, e := range b.Body {
		if e.Kind == "directive" && (len(e.Raw) != 1 || !rawMatchesDirective(e.Raw[0], e.Key, e.Values)) {
			return false
		}
		got = append(got, e.Raw...)
	}
	if len(got) != len(b.RawSpan) {
		return false
	}
	for i := range got {
		if got[i] != b.RawSpan[i] {
			return false
		}
	}
	return true
}

// Reconcile re-parses currentData (all openers) to recover flat-owned regions
// (root directives + non-named blocks) and the current named-block spans/order,
// then merges in edited to produce final bytes. untouched regions stay
// byte-identical.
func Reconcile(currentData []byte, edited BlockModel, openers []string, isNamed func(opener, header string) bool, place PlacementRule) []byte {
	lines := splitRawLines(currentData)
	regions := segment(lines, openers, isNamed)

	// map current named blocks by id (positional) to detect order change.
	var currentNamedIDs []string
	for _, r := range regions {
		if r.kind == regionNamed {
			currentNamedIDs = append(currentNamedIDs, r.id)
		}
	}
	var editedIDs []string
	editedByID := make(map[string]Block, len(edited.Blocks))
	for _, b := range edited.Blocks {
		editedIDs = append(editedIDs, b.ID)
		editedByID[b.ID] = b
	}

	if sameOrder(currentNamedIDs, editedIDs) {
		return reconcileInPlace(regions, editedByID)
	}
	return reconcileReorder(regions, edited, place)
}

type regionKind int

const (
	regionFlat  regionKind = iota // root directives or non-named blocks
	regionNamed                   // a named block span
)

type region struct {
	kind   regionKind
	id     string   // positional id for named regions ("b0",…)
	opener string   // for named/flat-block regions
	header string   // header text for the opener (flat or named)
	lines  []string // verbatim physical lines (with terminators)
}

// segment splits the file into ordered regions: named-block spans (with their
// owned leading comment/blank run) and flat regions (everything else). it
// mirrors Parse's ownership rule so ids line up.
func segment(lines []rawLine, openers []string, isNamed func(opener, header string) bool) []region {
	var regions []region
	blockIdx := 0
	flatStart := 0

	flushFlat := func(upto int) {
		if upto <= flatStart {
			return
		}
		var ls []string
		for n := flatStart; n < upto; n++ {
			ls = append(ls, lines[n].full())
		}
		regions = append(regions, region{kind: regionFlat, lines: ls})
	}

	i := 0
	for i < len(lines) {
		opener, ok := isOpener(lines[i].text, openers)
		if !ok {
			i++
			continue
		}

		// owned leading run start
		j := i - 1
		for j >= flatStart {
			k, _, _ := lineKind(lines[j].text)
			if k == "comment" || k == "blank" {
				j--
				continue
			}
			break
		}
		runStart := j + 1

		// block end = next opener
		end := len(lines)
		for n := i + 1; n < len(lines); n++ {
			if _, isOp := isOpener(lines[n].text, openers); isOp {
				end = n
				break
			}
		}
		// trim trailing comment/blank run (owned by next block)
		bodyEnd := end
		if end < len(lines) {
			t := end - 1
			for t >= i+1 {
				k, _, _ := lineKind(lines[t].text)
				if k == "comment" || k == "blank" {
					t--
					continue
				}
				break
			}
			bodyEnd = t + 1
		}

		header := headerOf(lines[i].text)
		named := isNamed == nil || isNamed(opener, header)

		// flush flat region up to the owned run start
		flushFlat(runStart)

		var ls []string
		for n := runStart; n < bodyEnd; n++ {
			ls = append(ls, lines[n].full())
		}
		if named {
			regions = append(regions, region{
				kind:   regionNamed,
				id:     "b" + strconv.Itoa(blockIdx),
				opener: opener,
				header: header,
				lines:  ls,
			})
			blockIdx++
		} else {
			regions = append(regions, region{
				kind:   regionFlat,
				opener: opener,
				header: header,
				lines:  ls,
			})
		}

		flatStart = bodyEnd
		i = end
		// any comment/blank between bodyEnd and end stays in flat/next-block;
		// reset flatStart to bodyEnd so it is captured before the next opener.
	}
	flushFlat(len(lines))
	return regions
}

func sameOrder(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// reconcileInPlace edits each named region's bytes surgically, drops named
// regions absent from edited, and leaves flat regions byte-identical.
func reconcileInPlace(regions []region, editedByID map[string]Block) []byte {
	var sb strings.Builder
	for _, r := range regions {
		if r.kind == regionFlat {
			for _, l := range r.lines {
				sb.WriteString(l)
			}
			continue
		}
		blk, keep := editedByID[r.id]
		if !keep {
			continue // deleted block: drop its span (incl. owned run)
		}
		for _, l := range renderBlock(blk) {
			sb.WriteString(l)
		}
	}
	return []byte(sb.String())
}

// reconcileReorder rewrites the named region in edited order. flat low-precedence
// blocks (per place) are pushed to the end; new named blocks insert before them.
func reconcileReorder(regions []region, edited BlockModel, place PlacementRule) []byte {
	// partition flat regions: those before the named region, low-precedence
	// flats (kept last), and the rest.
	var leading []region
	var lowPrec []region
	firstNamed := -1
	lastNamed := -1
	for i, r := range regions {
		if r.kind == regionNamed {
			if firstNamed < 0 {
				firstNamed = i
			}
			lastNamed = i
		}
	}

	isLow := func(r region) bool {
		if place.IsLowPrecedence == nil || r.kind != regionFlat {
			return false
		}
		return r.opener != "" && place.IsLowPrecedence(r.opener, r.header)
	}

	var sb strings.Builder

	// emit everything before the named region verbatim, except low-precedence
	// flats which are deferred to the end.
	emitOutside := func(lo, hi int) {
		for i := lo; i < hi; i++ {
			r := regions[i]
			if isLow(r) {
				lowPrec = append(lowPrec, r)
				continue
			}
			for _, l := range r.lines {
				sb.WriteString(l)
			}
		}
	}

	if firstNamed < 0 {
		// no current named region: emit all flats, then the edited blocks
		// before low-precedence flats.
		emitOutside(0, len(regions))
		writeEditedBlocks(&sb, edited)
		for _, r := range lowPrec {
			for _, l := range r.lines {
				sb.WriteString(l)
			}
		}
		return []byte(sb.String())
	}

	emitOutside(0, firstNamed)
	_ = leading

	// emit edited named blocks in order
	writeEditedBlocks(&sb, edited)

	// emit flats interleaved within the old named region (low-precedence
	// deferred) then everything after the named region.
	for i := firstNamed; i <= lastNamed; i++ {
		r := regions[i]
		if r.kind == regionNamed {
			continue
		}
		if isLow(r) {
			lowPrec = append(lowPrec, r)
			continue
		}
		for _, l := range r.lines {
			sb.WriteString(l)
		}
	}
	emitOutside(lastNamed+1, len(regions))

	for _, r := range lowPrec {
		for _, l := range r.lines {
			sb.WriteString(l)
		}
	}
	return []byte(sb.String())
}

func writeEditedBlocks(sb *strings.Builder, edited BlockModel) {
	for _, b := range edited.Blocks {
		for _, l := range renderBlock(b) {
			sb.WriteString(l)
		}
	}
}
