package ui

import (
	"strings"

	"github.com/eminert/konfi/konfables"
	"github.com/eminert/konfi/pkg"
	"github.com/eminert/konfi/theme"
	"github.com/eminert/konfi/ui/editors"
	"github.com/eminert/konfi/ui/widgets"

	"charm.land/lipgloss/v2"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/text"
)

// detail is a sub-model owned by content that renders the preview/detail pane.
type detail struct {
	previewLine  int
	previewFound bool
	previewKey   string
	previewGen   uint64
	docsURL      string
	theme        *theme.Theme

	// editor state
	editor      editors.FieldEditor
	editField   int    // index into fields slice
	editOrigVal string // for cancel restoration

	// scroll state for browse mode
	scrollY int

	// nerd font glyphs or ASCII fallback
	nerdFont bool

	// context synced from content on state changes
	field      *pkg.Field
	config     *pkg.ConfigFile
	konfable   konfables.Konfable
	values     map[string]string
	origValues map[string]string // on-disk baseline for inline old→new diff
	focused    bool

	// cached config lines for renderFileSnippet
	snippetLines []string
	snippetGen   uint64 // config generation that produced snippetLines

	// cached styles
	badgeBase lipgloss.Style
	cachedMD  *widgets.MdRenderer
	cachedMDW int
}

func newDetail(th *theme.Theme) detail {
	return detail{
		previewLine: -1,
		theme:       th,
		badgeBase:   lipgloss.NewStyle().Bold(true).Padding(0, 1),
	}
}

// sync pushes the latest content state into detail and refreshes the preview line.
func (d *detail) sync(f *pkg.Field, config *pkg.ConfigFile, konfable konfables.Konfable, values map[string]string, focused bool) {
	// reset scroll when field changes
	if f != d.field {
		d.scrollY = 0
	}
	d.field = f
	d.config = config
	d.konfable = konfable
	d.values = values
	d.focused = focused
	d.refreshPreviewLine()
}

// reset clears all detail state on app switch.
func (d *detail) reset() {
	d.previewLine = -1
	d.previewFound = false
	d.previewKey = ""
	d.previewGen = 0
	d.docsURL = ""
	d.scrollY = 0
	d.field = nil
	d.config = nil
	d.konfable = nil
	d.values = nil
	d.origValues = nil
	d.focused = false
	d.snippetLines = nil
}

// forceRescan clears the cached key so the next sync re-scans the config.
func (d *detail) forceRescan() {
	d.previewKey = ""
	d.previewGen = 0
	d.snippetLines = nil
}

// refreshPreviewLine updates the preview line from config for the current field.
func (d *detail) refreshPreviewLine() {
	f := d.field
	if f == nil || d.config == nil || d.konfable == nil || d.konfable.Parser() == nil {
		d.previewLine = -1
		d.previewFound = false
		d.previewKey = ""
		d.previewGen = 0
		return
	}
	gen := d.config.Generation()
	if f.Key == d.previewKey && gen == d.previewGen {
		return
	}
	d.previewKey = f.Key
	d.previewGen = gen
	d.previewLine, d.previewFound = d.konfable.Parser().FindLine(d.config.Content(), f.Key)
}

// renderMarkdown renders markdown using the goldmark-based renderer in ui/widgets.
func (d *detail) renderMarkdown(md string, width int) string {
	if d.cachedMD == nil || d.cachedMDW != width {
		d.cachedMD = widgets.NewMDRenderer(d.theme, width)
		d.cachedMDW = width
	}
	if md == "" {
		return ""
	}
	source := []byte(md)
	p := goldmark.DefaultParser()
	doc := p.Parse(text.NewReader(source))
	return strings.TrimRight(d.cachedMD.Render(doc, source), "\n")
}

// View renders the detail pane content — always browse mode.
// editing is handled inline in the field list (content.renderBody).
func (d *detail) View(width, height int) string {
	if d.focused && d.field != nil {
		return d.viewConfigFile(width, height)
	}
	return d.viewBrowse(width, height)
}

func (d *detail) scroll(delta int) {
	d.scrollY += delta
	if d.scrollY < 0 {
		d.scrollY = 0
	}
}

func (d *detail) scrollTop() {
	d.scrollY = 0
}

func (d *detail) scrollBottom() {
	d.scrollY = 1 << 30
}

func (d *detail) centerPreview(viewport int) {
	focusLine := d.configFocusLine()
	if focusLine < 0 {
		d.scrollY = 0
		return
	}
	if viewport < 1 {
		viewport = 1
	}
	d.scrollY = focusLine - viewport/2
	if d.scrollY < 0 {
		d.scrollY = 0
	}
}

func (d *detail) configFocusLine() int {
	if d.previewFound {
		return d.previewLine
	}
	if d.field == nil || d.config == nil || d.konfable == nil || d.konfable.Parser() == nil {
		return -1
	}
	if _, addedLine, ok := d.previewAddedContent(d.config.Content()); ok {
		return addedLine
	}
	if added := d.fallbackAddedLine(); added != "" {
		return len(d.configLines())
	}
	return -1
}
