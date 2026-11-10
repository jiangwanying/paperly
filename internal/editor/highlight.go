package editor

import (
	"sort"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown/parser"
)

// A block shows its matches by wrapping them in <mark> in the source the
// view draws. The view paints that tag in the find color, word for word,
// wherever text shows: in a paragraph, a heading, a list item, a quote, a
// table cell — the markup around the words stays as it is.

// textSpan is a stretch of a block's source that shows as text.
type textSpan struct{ start, end int }

// textSpans collects the stretches of block idx's source that show as
// text, the blocks within it — an item's paragraphs, a quote's, a table's
// cells — included: the places a tag can go without touching markup.
func textSpans(d *parser.Document, idx int, out []textSpan) []textSpan {
	next := d.Block(idx).Next
	for j := idx; j >= 0 && j < next; j++ {
		b := d.Block(j)
		if b == nil {
			break
		}
		switch b.Kind {
		case parser.Paragraph, parser.Heading, parser.TableCell:
			out = inlineSpans(d.Inlines(j), 0, out)
		}
	}
	return out
}

// inlineSpans collects the text of inlines: a code span, math, a picture
// or raw HTML shows as something other than its source's characters, and
// an address written as a link as <https://…> is no place for a tag.
func inlineSpans(in []parser.Inline, depth int, out []textSpan) []textSpan {
	if depth > 8 {
		return out
	}
	for i := 0; i < len(in); {
		n := in[i]
		switch n.Kind {
		case parser.Text:
			if n.End > n.Start {
				out = append(out, textSpan{n.Start, n.End})
			}
		case parser.Strong, parser.Emphasis, parser.Strikethrough:
			out = inlineSpans(in[i+1:n.Next], depth+1, out)
			i = n.Next
			continue
		case parser.Link:
			if !n.Auto {
				out = inlineSpans(in[i+1:n.Next], depth+1, out)
			}
			i = n.Next
			continue
		}
		i++
	}
	return out
}

// markedSource is block i's source with its matches wrapped in <mark>; the
// second answer reports whether the block has a match a tag can wrap. A
// block with no such text — a code block, the rules of a table — has none,
// and its caller tints the whole of it instead.
func (e *Editor) markedSource(i int) (string, bool) {
	if !e.findOpen {
		return "", false
	}
	if e.markedRev != e.doc.Revision() {
		clear(e.marked)
		e.markedRev = e.doc.Revision()
	}
	if s, ok := e.marked[i]; ok {
		return s, s != ""
	}
	s, ok := e.markBlock(i)
	if ok {
		e.marked[i] = s
	}
	return s, ok
}

// markBlock writes the tags into block i's source.
func (e *Editor) markBlock(i int) (string, bool) {
	b, ok := e.doc.Block(i)
	if !ok {
		return "", false
	}
	blocks := e.doc.Blocks()
	if i < 0 || i >= len(blocks) {
		return "", false
	}
	d := e.doc.Parse()
	idx := blocks[i].Index
	if d == nil || idx < 0 || idx >= d.Len() {
		return "", false
	}
	spans := textSpans(d, idx, nil)
	if len(spans) == 0 {
		return "", false
	}
	src := e.doc.Source()
	if b.End > len(src) || b.Start > b.End {
		return "", false
	}
	body := src[b.Start:b.End]
	// The matches that fall in those stretches, in the block's bytes.
	var cuts []textSpan
	for _, sp := range spans {
		for _, h := range e.hits {
			s, end := max(h.Start, sp.start), min(h.End, sp.end)
			if s >= end || !markSafe(src, s, end) {
				continue
			}
			cuts = append(cuts, textSpan{s - b.Start, end - b.Start})
		}
	}
	if len(cuts) == 0 {
		return "", false
	}
	sort.Slice(cuts, func(a, c int) bool { return cuts[a].start < cuts[c].start })
	var sb strings.Builder
	sb.Grow(len(body) + 13*len(cuts))
	last := 0
	for _, c := range cuts {
		if c.start < last || c.end > len(body) {
			continue
		}
		sb.WriteString(body[last:c.start])
		sb.WriteString("<mark>")
		sb.WriteString(body[c.start:c.end])
		sb.WriteString("</mark>")
		last = c.end
	}
	sb.WriteString(body[last:])
	return sb.String(), true
}

// markSafe reports whether the bytes s to e of src are the text itself: a
// tag written into an escape (\*) or an entity (&amp;) would break it.
func markSafe(src string, s, e int) bool {
	if s > 0 {
		switch src[s-1] {
		case '\\':
			return false
		case '&':
			if entityEnd(src, s-1) > 0 {
				return false
			}
		}
	}
	for i := s; i < e; i++ {
		switch src[i] {
		case '\\':
			return false
		case '&':
			if entityEnd(src, i) > 0 {
				return false
			}
		}
	}
	return true
}

// entityEnd is where the entity starting at i ends, or -1 when what starts
// there is a plain ampersand.
func entityEnd(src string, i int) int {
	j := i + 1
	if j < len(src) && src[j] == '#' {
		j++
		if j < len(src) && (src[j] == 'x' || src[j] == 'X') {
			j++
		}
	}
	start := j
	for j < len(src) && j-i <= 32 && isEntityByte(src[j]) {
		j++
	}
	if j == start || j >= len(src) || src[j] != ';' {
		return -1
	}
	return j + 1
}

func isEntityByte(c byte) bool {
	return '0' <= c && c <= '9' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z'
}

// markTheme paints <mark> in the find color while the bar is open, and in
// the document's own color when it is not: the view builds its text anew
// as the color changes.
func (e *Editor) markTheme() {
	if e.mdBase == nil {
		return
	}
	if !e.findOpen || len(e.hits) == 0 {
		e.md.Theme = e.mdBase
		return
	}
	c := e.highlightColor()
	if e.mdMark == nil || e.markCol != c {
		md := *e.mdBase
		md.Mark = c
		e.mdMark, e.markCol = &md, c
	}
	e.md.Theme = e.mdMark
}

// clearMarks drops what the last search painted.
func (e *Editor) clearMarks() {
	clear(e.marked)
	e.markedRev = 0
}

// highlightColor is the color the search paints, from the settings: the pink
// it starts with, unless another is chosen, or the one the user mixed.
func (e *Editor) highlightColor() ui.Color {
	switch e.sh.FindColor {
	case "gray":
		return ui.RGBA(128, 128, 128, 0.28)
	case "yellow":
		return ui.RGBA(255, 214, 0, 0.35)
	case "blue":
		return ui.RGBA(64, 156, 255, 0.30)
	case "green":
		return ui.RGBA(60, 190, 120, 0.30)
	case "custom":
		if e.sh.FindCustom.A > 0 {
			return e.sh.FindCustom
		}
	}
	return ui.RGBA(255, 120, 180, 0.30)
}
