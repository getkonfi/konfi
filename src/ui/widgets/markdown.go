package widgets

import (
	"fmt"
	"strings"

	"github.com/getkonfi/konfi/theme"

	"charm.land/lipgloss/v2"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// MdRenderer holds lipgloss styles derived from the active theme.
type MdRenderer struct {
	width int
	theme *theme.Theme

	bold       lipgloss.Style
	italic     lipgloss.Style
	code       lipgloss.Style
	codeBlock  lipgloss.Style
	heading    lipgloss.Style
	linkText   lipgloss.Style
	linkURL    lipgloss.Style
	body       lipgloss.Style
	listBullet lipgloss.Style
}

func NewMDRenderer(th *theme.Theme, width int) *MdRenderer {
	return &MdRenderer{
		width:  width,
		theme:  th,
		bold:   lipgloss.NewStyle().Foreground(th.Palette.Text).Bold(true),
		italic: lipgloss.NewStyle().Foreground(th.Palette.Subtext).Italic(true),
		code:   lipgloss.NewStyle().Foreground(th.Palette.Accent),
		codeBlock: lipgloss.NewStyle().Foreground(th.Palette.Muted).
			PaddingLeft(2),
		heading:    lipgloss.NewStyle().Foreground(th.Palette.Primary).Bold(true).Underline(true),
		linkText:   lipgloss.NewStyle().Foreground(th.Palette.Primary),
		linkURL:    lipgloss.NewStyle().Foreground(th.Palette.Secondary).Underline(true),
		body:       lipgloss.NewStyle().Foreground(th.Palette.Muted),
		listBullet: lipgloss.NewStyle().Foreground(th.Palette.Muted),
	}
}

// RenderMarkdown parses markdown and returns styled terminal output.
func RenderMarkdown(content string, width int, th *theme.Theme) string {
	if content == "" {
		return ""
	}
	source := []byte(content)
	p := goldmark.DefaultParser()
	doc := p.Parse(text.NewReader(source))
	r := NewMDRenderer(th, width)
	out := r.Render(doc, source)
	return strings.TrimRight(out, "\n")
}

// Render walks the AST and produces styled output.
func (r *MdRenderer) Render(doc ast.Node, source []byte) string {
	var b strings.Builder
	var listIndex int // current ordered-list counter
	var ordered bool  // ordered vs unordered

	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		switch n.Kind() {

		case ast.KindDocument:
			// noop

		case ast.KindHeading:
			if entering {
				// collect all child text, then style it
				txt := r.collectText(n, source)
				b.WriteString(r.heading.Render(txt))
				b.WriteByte('\n')
				return ast.WalkSkipChildren, nil
			}

		case ast.KindParagraph:
			if !entering {
				b.WriteByte('\n')
			}

		case ast.KindList:
			if entering {
				list := n.(*ast.List)
				ordered = list.IsOrdered()
				listIndex = list.Start
				if listIndex == 0 && ordered {
					listIndex = 1
				}
			}

		case ast.KindListItem:
			if entering {
				if ordered {
					b.WriteString(r.listBullet.Render(fmt.Sprintf("%d. ", listIndex)))
					listIndex++
				} else {
					b.WriteString(r.listBullet.Render("  - "))
				}
			} else {
				// ensure newline after list item
				if !strings.HasSuffix(b.String(), "\n") {
					b.WriteByte('\n')
				}
			}

		case ast.KindCodeBlock, ast.KindFencedCodeBlock:
			if entering {
				lines := r.collectLines(n, source)
				styled := r.codeBlock.Render(strings.TrimRight(lines, "\n"))
				b.WriteString(styled)
				b.WriteByte('\n')
				return ast.WalkSkipChildren, nil
			}

		case ast.KindBlockquote:
			if entering {
				txt := r.collectText(n, source)
				bq := r.body.Italic(true).Render("  " + txt)
				b.WriteString(bq)
				b.WriteByte('\n')
				return ast.WalkSkipChildren, nil
			}

		case ast.KindThematicBreak:
			if entering {
				w := r.width
				if w < 3 {
					w = 3
				}
				b.WriteString(r.body.Render(strings.Repeat("─", w)))
				b.WriteByte('\n')
			}

		// inline nodes
		case ast.KindText:
			if entering {
				t := n.(*ast.Text)
				b.WriteString(r.body.Render(string(t.Segment.Value(source))))
				if t.SoftLineBreak() {
					b.WriteByte(' ')
				}
				if t.HardLineBreak() {
					b.WriteByte('\n')
				}
			}

		case ast.KindEmphasis:
			em := n.(*ast.Emphasis)
			if entering {
				txt := r.collectText(n, source)
				if em.Level >= 2 {
					b.WriteString(r.bold.Render(txt))
				} else {
					b.WriteString(r.italic.Render(txt))
				}
				return ast.WalkSkipChildren, nil
			}

		case ast.KindCodeSpan:
			if entering {
				txt := r.collectText(n, source)
				b.WriteString(r.code.Render(txt))
				return ast.WalkSkipChildren, nil
			}

		case ast.KindLink:
			if entering {
				link := n.(*ast.Link)
				txt := r.collectText(n, source)
				dest := string(link.Destination)
				if txt == dest || txt == "" {
					b.WriteString(r.linkURL.Render(dest))
				} else {
					b.WriteString(r.linkText.Render(txt))
					b.WriteString(r.body.Render(" ("))
					b.WriteString(r.linkURL.Render(dest))
					b.WriteString(r.body.Render(")"))
				}
				return ast.WalkSkipChildren, nil
			}

		case ast.KindAutoLink:
			if entering {
				al := n.(*ast.AutoLink)
				url := string(al.URL(source))
				b.WriteString(r.linkURL.Render(url))
			}
		}

		return ast.WalkContinue, nil
	})

	return wordWrap(b.String(), r.width)
}

// collectText recursively extracts raw text from an inline node and its children.
func (r *MdRenderer) collectText(n ast.Node, source []byte) string {
	var b strings.Builder
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		switch c := c.(type) {
		case *ast.Text:
			b.Write(c.Segment.Value(source))
			if c.SoftLineBreak() {
				b.WriteByte(' ')
			}
		case *ast.String:
			b.Write(c.Value)
		default:
			if c.HasChildren() {
				b.WriteString(r.collectText(c, source))
			}
		}
	}
	return b.String()
}

// collectLines extracts raw lines from a code block node.
func (r *MdRenderer) collectLines(n ast.Node, source []byte) string {
	var b strings.Builder
	lines := n.Lines()
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		b.Write(seg.Value(source))
	}
	return b.String()
}

// wordWrap wraps text at word boundaries respecting ANSI escape sequences.
// operates per-line to preserve existing line breaks.
func wordWrap(s string, width int) string {
	if width <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	var out strings.Builder
	for i, line := range lines {
		if i > 0 {
			out.WriteByte('\n')
		}
		if lipgloss.Width(line) <= width {
			out.WriteString(line)
			continue
		}
		// split on spaces, accumulate until width exceeded
		words := strings.Fields(line)
		col := 0
		for j, w := range words {
			ww := lipgloss.Width(w)
			if j > 0 && col+1+ww > width {
				out.WriteByte('\n')
				col = 0
			} else if j > 0 {
				out.WriteByte(' ')
				col++
			}
			out.WriteString(w)
			col += ww
		}
	}
	return out.String()
}
