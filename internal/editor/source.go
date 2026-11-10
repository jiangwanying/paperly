package editor

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"paperly/internal/doc"
)

// ToggleSource shows the whole document as its source, as Typora's 源代码模式
// does, or goes back to the blocks. What is written in the source reaches
// the document as it is typed, so that a save keeps it.
func (e *Editor) ToggleSource() {
	if e.source {
		e.doc.SetSource(e.sourceText)
		e.source = false
		return
	}
	// The block being written is part of the document before the whole of it
	// is shown.
	e.Commit()
	e.sourceText = e.doc.Source()
	e.source = true
}

// Source reports whether the whole document shows as its source.
func (e *Editor) Source() bool { return e.source }

// sourceArea is the page in source mode: the document's text, from its first
// character to its last, in the editor's own type and size.
func (e *Editor) sourceArea(c *ui.Context, t *ui.Theme) {
	ut := *e.ui
	ut.FontSize = e.sh.Size
	ui.Column(c).Fill().Padding(30, 0).Children(func() {
		ui.Themed(c, &ut).Children(func() {
			// The area fills the page and scrolls within it, as the text of
			// a document outlasts its window.
			ta := ui.TextAreaBase(c.Key("source"), &e.sourceText).
				Lines(1, 0).Grow(1).Padding(0, 0).AutoFocus()
			ta.Border(0, ui.Color{})
			ta.Background(ui.Color{})
			ta.FocusRing(false)
			ta.ContextMenu(func(m *ui.Menu) { e.canvasMenu(c, m) })
			ranges := e.sourceRanges(e.sourceText, t.TextMuted, t.Text)
			ranges = append(ranges, e.findRanges(e.sourceText, 0, e.highlightColor(), e.highlightColor())...)
			ta.TextRanges(ranges...)
			ta.OnChange(func() { e.doc.SetSource(e.sourceText) })
			ta.OnTextKey(func(k ui.TextKey) bool {
				// Command-/ goes back to the blocks, as it came here.
				if k.Key == ui.KeySlash && k.Mods&ui.Cmd != 0 {
					e.ToggleSource()
					return true
				}
				return false
			})
		})
	})
}

// sourceRanges colors the whole source, line by line: the markers fade, the
// text between them takes its weight, as in a block being written.
func (e *Editor) sourceRanges(src string, muted, plain ui.Color) []ui.TextRange {
	var out []ui.TextRange
	at := 0 // where the line starts, in runes
	for _, line := range strings.SplitAfter(src, "\n") {
		for _, r := range doc.SyntaxRuns(line) {
			c, weight := plain, r.Weight
			if r.Kind == doc.RunMarker {
				c = muted
			}
			out = append(out, ui.TextRange{
				Start:  at + doc.RuneOffset(line, r.Start),
				End:    at + doc.RuneOffset(line, r.End),
				Color:  c,
				Weight: weight,
			})
		}
		at += doc.RuneLen(line)
	}
	return out
}
