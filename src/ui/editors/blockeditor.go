package editors

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/eminert/konfi/pkg"
	"github.com/eminert/konfi/theme"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

// blockEditor edits a BlockModel (ssh Host/Match style blocks) two levels deep:
// a block-list view, and a body view per block that composes real nested
// FieldEditors per directive. the value channel is one opaque encoded string
// (Encode/Decode), written via the raw-widget path. it deliberately does NOT
// implement MultiValueEditor.
type blockEditor struct {
	model   pkg.BlockModel
	palette []pkg.Field
	th      *theme.Theme

	mode blockMode

	// block-list cursor
	blockCursor int

	// body cursor (index into the current block's Body, opener excluded from nav)
	bodyCursor int
	curBlock   int // index of the expanded block in model.Blocks

	// nested directive editor
	editing   bool
	nested    FieldEditor
	editEntry int // index into Body of the entry being edited

	// header editor
	editingHeader bool
	headerInput   textinput.Model
	headerForHost bool

	// add-block flow
	addingBlock   bool
	addStep       int // 0 = opener choice, 1 = header entry
	addOpenerIdx  int
	addOpener     string
	openerOptions []string

	// add-directive menu (palette keys)
	addingDirective bool
	dirCursor       int
}

type blockMode int

const (
	modeBlockList blockMode = iota
	modeBody
)

func (e *blockEditor) Init(field pkg.Field, currentValue string, th *theme.Theme) tea.Cmd {
	e.th = th
	e.palette = field.BlockPalette
	if currentValue == "" {
		e.model = pkg.BlockModel{}
	} else {
		e.model = pkg.Decode(currentValue)
	}
	e.mode = modeBlockList
	e.blockCursor = 0
	e.openerOptions = []string{"Host", "Match"}
	e.headerInput = newFieldInput(th)
	return nil
}

// Value returns the lossless encoding of the current model. with no edits this
// equals the currentValue passed to Init (Decode∘Encode is identity).
func (e *blockEditor) Value() string { return pkg.Encode(e.model) }

func (e *blockEditor) Interaction() InteractionKind { return InteractionList }

func (e *blockEditor) Update(msg tea.Msg) (tea.Cmd, bool, bool) {
	switch {
	case e.editing:
		return e.updateNested(msg)
	case e.editingHeader:
		return e.updateHeader(msg)
	case e.addingBlock:
		return e.updateAddBlock(msg)
	case e.addingDirective:
		return e.updateAddDirective(msg)
	}

	km, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil, false, false
	}
	if e.mode == modeBlockList {
		return e.updateBlockList(km)
	}
	return e.updateBody(km)
}

// ── block-list view ─────────────────────────────────────────────────────────

func (e *blockEditor) updateBlockList(km tea.KeyPressMsg) (tea.Cmd, bool, bool) {
	switch km.String() {
	case "j", "down":
		if e.blockCursor < len(e.model.Blocks)-1 {
			e.blockCursor++
		}
	case "k", "up":
		if e.blockCursor > 0 {
			e.blockCursor--
		}
	case "J":
		// reorder selected block down
		if e.blockCursor < len(e.model.Blocks)-1 {
			bs := e.model.Blocks
			bs[e.blockCursor], bs[e.blockCursor+1] = bs[e.blockCursor+1], bs[e.blockCursor]
			e.blockCursor++
		}
	case "K":
		// reorder selected block up
		if e.blockCursor > 0 {
			bs := e.model.Blocks
			bs[e.blockCursor], bs[e.blockCursor-1] = bs[e.blockCursor-1], bs[e.blockCursor]
			e.blockCursor--
		}
	case "enter":
		if e.blockCursor < len(e.model.Blocks) {
			e.mode = modeBody
			e.curBlock = e.blockCursor
			e.bodyCursor = e.firstNavEntry()
		}
	case "h":
		// edit the selected block's header
		if e.blockCursor < len(e.model.Blocks) {
			return e.startHeaderEdit(e.blockCursor)
		}
	case "a":
		e.addingBlock = true
		e.addStep = 0
		e.addOpenerIdx = 0
	case "d":
		if e.blockCursor < len(e.model.Blocks) {
			e.model.Blocks = append(e.model.Blocks[:e.blockCursor], e.model.Blocks[e.blockCursor+1:]...)
			if e.blockCursor >= len(e.model.Blocks) && e.blockCursor > 0 {
				e.blockCursor--
			}
		}
	case "esc", "left":
		// commit and exit; the model holds every edit (undo can revert).
		return nil, true, false
	}
	return nil, false, false
}

// ── body view (inside one block) ────────────────────────────────────────────

func (e *blockEditor) updateBody(km tea.KeyPressMsg) (tea.Cmd, bool, bool) {
	body := e.model.Blocks[e.curBlock].Body
	switch km.String() {
	case "j", "down":
		if n := e.nextNavEntry(e.bodyCursor); n >= 0 {
			e.bodyCursor = n
		}
	case "k", "up":
		if p := e.prevNavEntry(e.bodyCursor); p >= 0 {
			e.bodyCursor = p
		}
	case "enter":
		if e.bodyCursor >= 0 && e.bodyCursor < len(body) && body[e.bodyCursor].Kind == "directive" {
			return e.startDirectiveEdit(e.bodyCursor)
		}
	case "a":
		if len(e.palette) > 0 {
			e.addingDirective = true
			e.dirCursor = 0
		}
	case "d":
		if e.bodyCursor >= 0 && e.bodyCursor < len(body) && body[e.bodyCursor].Kind == "directive" {
			blk := &e.model.Blocks[e.curBlock]
			blk.Body = append(blk.Body[:e.bodyCursor], blk.Body[e.bodyCursor+1:]...)
			if n := e.prevNavEntry(e.bodyCursor); n >= 0 && e.bodyCursor >= len(blk.Body) {
				e.bodyCursor = n
			} else if e.bodyCursor >= len(blk.Body) {
				e.bodyCursor = e.firstNavEntry()
			}
		}
	case "esc", "left":
		e.mode = modeBlockList
	}
	return nil, false, false
}

// firstNavEntry returns the index of the first navigable (directive) entry, or
// -1 if none.
func (e *blockEditor) firstNavEntry() int {
	body := e.model.Blocks[e.curBlock].Body
	for i := range body {
		if body[i].Kind == "directive" {
			return i
		}
	}
	return -1
}

func (e *blockEditor) nextNavEntry(from int) int {
	body := e.model.Blocks[e.curBlock].Body
	for i := from + 1; i < len(body); i++ {
		if body[i].Kind == "directive" {
			return i
		}
	}
	return -1
}

func (e *blockEditor) prevNavEntry(from int) int {
	body := e.model.Blocks[e.curBlock].Body
	for i := from - 1; i >= 0; i-- {
		if body[i].Kind == "directive" {
			return i
		}
	}
	return -1
}

// ── nested directive editing ────────────────────────────────────────────────

func (e *blockEditor) startDirectiveEdit(entryIdx int) (tea.Cmd, bool, bool) {
	entry := e.model.Blocks[e.curBlock].Body[entryIdx]
	f := e.fieldFor(entry.Key)
	e.nested = ForField(f)
	e.editing = true
	e.editEntry = entryIdx
	cur := strings.Join(entry.Values, " ")
	return e.nested.Init(f, cur, e.th), false, false
}

func (e *blockEditor) updateNested(msg tea.Msg) (tea.Cmd, bool, bool) {
	cmd, done, canceled := e.nested.Update(msg)
	if !done {
		return cmd, false, false
	}
	if !canceled {
		// write the nested editor's value back into the entry
		blk := &e.model.Blocks[e.curBlock]
		entry := &blk.Body[e.editEntry]
		entry.Values = splitDirectiveValue(e.nested.Value())
	}
	e.editing = false
	e.nested = nil
	return cmd, false, false
}

// fieldFor returns the palette field for a directive key, restricted to
// single-value kinds. list/multi palette fields fall back to a plain string
// editor (repetition is the block model's job, not a nested list editor).
func (e *blockEditor) fieldFor(key string) pkg.Field {
	for i := range e.palette {
		if strings.EqualFold(e.palette[i].Key, key) {
			return restrictToSingleValue(e.palette[i])
		}
	}
	// unknown directive: edit as a plain string under its key
	return pkg.Field{Key: key, Type: "string"}
}

// restrictToSingleValue maps a palette field to a single-value editable field.
// allowed nested kinds: string, number, enum, color, path, slider. list/multi
// degrade to a plain string editor (repetition is the block model's job).
func restrictToSingleValue(f pkg.Field) pkg.Field {
	switch f.Widget {
	case "slider", "path":
		return f
	}
	switch f.Type {
	case "number", "enum", "color":
		out := f
		out.Widget = "" // drop any multi-value widget hint
		return out
	case "list", "multi":
		return pkg.Field{Key: f.Key, Label: f.Label, Type: "string"}
	default:
		out := f
		out.Widget = ""
		return out
	}
}

// splitDirectiveValue tokenizes a nested editor's value back into the entry's
// value tokens, honoring double-quoted spans so e.g. exec "some cmd" stays one
// token. mirrors the block engine's tokenizer so re-encoded values round-trip.
func splitDirectiveValue(v string) []string {
	if strings.TrimSpace(v) == "" {
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

// ── add-directive menu ──────────────────────────────────────────────────────

func (e *blockEditor) updateAddDirective(msg tea.Msg) (tea.Cmd, bool, bool) {
	km, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil, false, false
	}
	switch km.String() {
	case "j", "down":
		if e.dirCursor < len(e.palette)-1 {
			e.dirCursor++
		}
	case "k", "up":
		if e.dirCursor > 0 {
			e.dirCursor--
		}
	case "enter":
		f := e.palette[e.dirCursor]
		blk := &e.model.Blocks[e.curBlock]
		entry := pkg.Entry{
			ID:     e.nextEntryID(blk),
			Kind:   "directive",
			Key:    f.Key,
			Values: nil,
		}
		blk.Body = append(blk.Body, entry)
		e.bodyCursor = len(blk.Body) - 1
		e.addingDirective = false
		// open the nested editor immediately so the new directive gets a value
		return e.startDirectiveEdit(e.bodyCursor)
	case "esc":
		e.addingDirective = false
	}
	return nil, false, false
}

// nextEntryID returns a fresh stable entry id for a block.
func (e *blockEditor) nextEntryID(b *pkg.Block) string {
	maxN := -1
	for _, en := range b.Body {
		if strings.HasPrefix(en.ID, "e") {
			if n, err := strconv.Atoi(en.ID[1:]); err == nil && n > maxN {
				maxN = n
			}
		}
	}
	return "e" + strconv.Itoa(maxN+1)
}

// ── header editing ──────────────────────────────────────────────────────────

func (e *blockEditor) startHeaderEdit(blockIdx int) (tea.Cmd, bool, bool) {
	blk := e.model.Blocks[blockIdx]
	e.editingHeader = true
	e.blockCursor = blockIdx
	e.headerForHost = strings.EqualFold(blk.Opener, "Host")
	e.headerInput = newFieldInput(e.th)
	e.headerInput.SetValue(blk.Header)
	e.headerInput.CursorEnd()
	return e.headerInput.Focus(), false, false
}

func (e *blockEditor) updateHeader(msg tea.Msg) (tea.Cmd, bool, bool) {
	if km, ok := msg.(tea.KeyPressMsg); ok {
		switch km.String() {
		case "enter":
			e.commitHeader()
			e.editingHeader = false
			e.headerInput.Blur()
			return nil, false, false
		case "esc":
			e.editingHeader = false
			e.headerInput.Blur()
			return nil, false, false
		}
	}
	var cmd tea.Cmd
	e.headerInput, cmd = e.headerInput.Update(msg)
	return cmd, false, false
}

// commitHeader writes the edited header back. Host headers are normalized as a
// space-separated pattern set; Match headers are stored verbatim (quoting and
// exec args preserved — never field-split).
func (e *blockEditor) commitHeader() {
	raw := strings.TrimSpace(e.headerInput.Value())
	header := raw
	if e.headerForHost {
		header = strings.Join(strings.Fields(raw), " ")
	}
	blk := &e.model.Blocks[e.blockCursor]
	blk.Header = header
	// the opener entry's Raw is now stale; clear it so renderBlock rebuilds the
	// opener line from Opener+Header. block engine keeps the opener as an entry.
	e.refreshOpenerEntry(blk)
}

// refreshOpenerEntry rewrites the opener entry's Raw to reflect Opener+Header so
// the encoded model and any downstream reconcile emit the edited header.
func (e *blockEditor) refreshOpenerEntry(b *pkg.Block) {
	for i := range b.Body {
		if b.Body[i].Kind != "opener" {
			continue
		}
		eol := "\n"
		if len(b.Body[i].Raw) == 1 {
			switch {
			case strings.HasSuffix(b.Body[i].Raw[0], "\r\n"):
				eol = "\r\n"
			case strings.HasSuffix(b.Body[i].Raw[0], "\n"):
				eol = "\n"
			default:
				eol = ""
			}
		}
		line := b.Opener
		if b.Header != "" {
			line += " " + b.Header
		}
		b.Body[i].Raw = []string{line + eol}
		// keep RawSpan consistent with the body so encode/reconcile agree
		e.rebuildRawSpan(b)
		return
	}
}

// rebuildRawSpan recomputes a block's RawSpan from its body entry Raw lines.
func (e *blockEditor) rebuildRawSpan(b *pkg.Block) {
	var span []string
	for _, en := range b.Body {
		span = append(span, en.Raw...)
	}
	b.RawSpan = span
}

// ── add-block flow ──────────────────────────────────────────────────────────

func (e *blockEditor) updateAddBlock(msg tea.Msg) (tea.Cmd, bool, bool) {
	if e.addStep == 0 {
		km, ok := msg.(tea.KeyPressMsg)
		if !ok {
			return nil, false, false
		}
		switch km.String() {
		case "j", "down":
			if e.addOpenerIdx < len(e.openerOptions)-1 {
				e.addOpenerIdx++
			}
		case "k", "up":
			if e.addOpenerIdx > 0 {
				e.addOpenerIdx--
			}
		case "enter":
			e.addOpener = e.openerOptions[e.addOpenerIdx]
			e.addStep = 1
			e.headerForHost = strings.EqualFold(e.addOpener, "Host")
			e.headerInput = newFieldInput(e.th)
			return e.headerInput.Focus(), false, false
		case "esc":
			e.addingBlock = false
		}
		return nil, false, false
	}

	// step 1: header entry
	if km, ok := msg.(tea.KeyPressMsg); ok {
		switch km.String() {
		case "enter":
			e.commitNewBlock()
			e.addingBlock = false
			e.headerInput.Blur()
			return nil, false, false
		case "esc":
			e.addingBlock = false
			e.headerInput.Blur()
			return nil, false, false
		}
	}
	var cmd tea.Cmd
	e.headerInput, cmd = e.headerInput.Update(msg)
	return cmd, false, false
}

// commitNewBlock builds a new block from the chosen opener and header and
// appends it to the model.
func (e *blockEditor) commitNewBlock() {
	raw := strings.TrimSpace(e.headerInput.Value())
	header := raw
	if e.headerForHost {
		header = strings.Join(strings.Fields(raw), " ")
	}
	line := e.addOpener
	if header != "" {
		line += " " + header
	}
	line += "\n"
	blk := pkg.Block{
		ID:      e.nextBlockID(),
		Opener:  e.addOpener,
		Header:  header,
		RawSpan: []string{line},
		Body: []pkg.Entry{
			{ID: "e0", Kind: "opener", Raw: []string{line}},
		},
	}
	e.model.Blocks = append(e.model.Blocks, blk)
	e.blockCursor = len(e.model.Blocks) - 1
}

// nextBlockID returns a fresh positional block id.
func (e *blockEditor) nextBlockID() string {
	maxN := -1
	for _, b := range e.model.Blocks {
		if strings.HasPrefix(b.ID, "b") {
			if n, err := strconv.Atoi(b.ID[1:]); err == nil && n > maxN {
				maxN = n
			}
		}
	}
	return "b" + strconv.Itoa(maxN+1)
}

// ── view ────────────────────────────────────────────────────────────────────

func (e *blockEditor) View(width int) string {
	if e.mode == modeBody {
		return e.viewBody(width)
	}
	return e.viewBlockList(width)
}

func (e *blockEditor) viewBlockList(width int) string {
	var b strings.Builder

	if e.addingBlock {
		return e.viewAddBlock(width)
	}

	if len(e.model.Blocks) == 0 {
		b.WriteString("    " + e.th.Muted.Render("(no blocks) press a to add"))
		b.WriteByte('\n')
		b.WriteString("    " + e.th.Muted.Render("a:add  esc/←:save"))
		return b.String()
	}

	for i, blk := range e.model.Blocks {
		label := blk.Opener
		if blk.Header != "" {
			label += " " + blk.Header
		}
		switch {
		case i == e.blockCursor && e.editingHeader:
			fmt.Fprintf(&b, "    %s %s ", e.th.Primary.Render(">"), e.th.Muted.Render(blk.Opener))
			e.headerInput.SetWidth(width - 8 - len(blk.Opener))
			b.WriteString(e.headerInput.View())
		case i == e.blockCursor:
			fmt.Fprintf(&b, "    %s %s", e.th.Primary.Render(">"), e.th.Text.Bold(true).Render(label))
		default:
			fmt.Fprintf(&b, "      %s", e.th.Subtext.Render(label))
		}
		b.WriteByte('\n')
	}

	if e.editingHeader {
		b.WriteString("    " + e.th.Muted.Render("⏎:save  esc:cancel"))
	} else {
		b.WriteString("    " + e.th.Muted.Render("⏎:open  a:add  d:delete  h:header  J/K:move  esc/←:save"))
	}
	return b.String()
}

func (e *blockEditor) viewAddBlock(width int) string {
	var b strings.Builder
	if e.addStep == 0 {
		b.WriteString("    " + e.th.Muted.Render("opener:"))
		b.WriteByte('\n')
		for i, o := range e.openerOptions {
			if i == e.addOpenerIdx {
				fmt.Fprintf(&b, "      %s %s", e.th.Primary.Render(">"), e.th.Text.Bold(true).Render(o))
			} else {
				fmt.Fprintf(&b, "        %s", e.th.Subtext.Render(o))
			}
			b.WriteByte('\n')
		}
		b.WriteString("    " + e.th.Muted.Render("⏎:next  esc:cancel"))
		return b.String()
	}
	b.WriteString("    " + e.th.Muted.Render(e.addOpener+" "))
	e.headerInput.SetWidth(width - 8 - len(e.addOpener))
	b.WriteString(e.headerInput.View())
	b.WriteByte('\n')
	b.WriteString("    " + e.th.Muted.Render("⏎:add  esc:cancel"))
	return b.String()
}

func (e *blockEditor) viewBody(width int) string {
	var b strings.Builder
	blk := e.model.Blocks[e.curBlock]

	label := blk.Opener
	if blk.Header != "" {
		label += " " + blk.Header
	}
	b.WriteString("    " + e.th.Accent.Render(label))
	b.WriteByte('\n')

	if e.addingDirective {
		return b.String() + e.viewAddDirective()
	}

	hasDir := false
	for i, en := range blk.Body {
		if en.Kind != "directive" {
			continue
		}
		hasDir = true
		display := en.Key
		if len(en.Values) > 0 {
			display += " " + strings.Join(en.Values, " ")
		}
		switch {
		case e.editing && i == e.editEntry:
			fmt.Fprintf(&b, "    %s %s", e.th.Primary.Render(">"), e.th.Text.Bold(true).Render(en.Key))
			b.WriteByte('\n')
			b.WriteString(e.nested.View(width))
		case i == e.bodyCursor:
			fmt.Fprintf(&b, "    %s %s", e.th.Primary.Render(">"), e.th.Text.Bold(true).Render(display))
		default:
			fmt.Fprintf(&b, "      %s", e.th.Subtext.Render(display))
		}
		b.WriteByte('\n')
	}

	if !hasDir && !e.editing {
		b.WriteString("    " + e.th.Muted.Render("(no directives) press a to add"))
		b.WriteByte('\n')
	}

	if e.editing {
		b.WriteString("    " + e.th.Muted.Render("⏎:save  esc:cancel"))
	} else {
		b.WriteString("    " + e.th.Muted.Render("⏎:edit  a:add  d:delete  esc/←:back"))
	}
	return b.String()
}

func (e *blockEditor) viewAddDirective() string {
	var b strings.Builder
	b.WriteString("    " + e.th.Muted.Render("directive:"))
	b.WriteByte('\n')
	for i := range e.palette {
		f := &e.palette[i]
		label := f.Key
		if f.Label != "" {
			label = f.Label
		}
		if i == e.dirCursor {
			fmt.Fprintf(&b, "      %s %s", e.th.Primary.Render(">"), e.th.Text.Bold(true).Render(label))
		} else {
			fmt.Fprintf(&b, "        %s", e.th.Subtext.Render(label))
		}
		b.WriteByte('\n')
	}
	b.WriteString("    " + e.th.Muted.Render("⏎:add  esc:cancel"))
	return b.String()
}

// ── height / offset ─────────────────────────────────────────────────────────

func (e *blockEditor) Height() int {
	if e.mode == modeBody {
		return e.heightBody()
	}
	return e.heightBlockList()
}

func (e *blockEditor) heightBlockList() int {
	if e.addingBlock {
		if e.addStep == 0 {
			return len(e.openerOptions) + 2 // label + options + help
		}
		return 2 // header input + help
	}
	h := len(e.model.Blocks)
	if h == 0 {
		h = 1 // empty hint
	}
	h++ // help line
	return h
}

func (e *blockEditor) heightBody() int {
	if e.addingDirective {
		return 1 + len(e.palette) + 2 // block header + label + options + help
	}
	blk := e.model.Blocks[e.curBlock]
	dirs := 0
	for _, en := range blk.Body {
		if en.Kind == "directive" {
			dirs++
		}
	}
	h := 1 + dirs // block header line + directive rows
	if dirs == 0 {
		h++ // empty hint
	}
	if e.editing && e.nested != nil {
		h += e.nested.Height() // nested editor body
	}
	h++ // help line
	return h
}

func (e *blockEditor) CursorOffset() int {
	if e.mode == modeBlockList {
		return e.blockCursor
	}
	// body view: one row per directive above the cursor, plus the block header.
	blk := e.model.Blocks[e.curBlock]
	row := 1 // block header line
	for i, en := range blk.Body {
		if en.Kind != "directive" {
			continue
		}
		if i == e.bodyCursor {
			break
		}
		row++
	}
	if e.editing && e.nested != nil {
		if oe, ok := e.nested.(OffsetEditor); ok {
			row += oe.CursorOffset()
		}
	}
	return row
}
