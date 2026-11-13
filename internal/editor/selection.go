package editor

import (
	"strings"
	"unicode/utf8"

	"paperly/locales"

	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown/parser"
)

// selecting reports whether the page takes a selection rather than a click
// on the block it lands on: it does while the selection modifier, ⌥, is
// held, as a page of rendered text does everywhere else, and for as long as
// a selection made then holds — so that Copy still has it once the key is
// let go, as a page in a browser keeps what was selected.
//
// A block that can be clicked, which is how writing starts, keeps its text
// out of the page's selection — a control owns its own text — so the two
// cannot stand together: the modifier takes the click away and gives the
// text back for the while it is held. The click that drops the selection
// ends the mode, and the block under it opens on the next one.
func (e *Editor) selecting(c *ui.Context) bool {
	return c.Modifiers()&ui.Alt != 0 || e.testMod&ui.Alt != 0 || e.pageSel > 0
}

// selectable lets the text of the page be selected across its blocks: the
// pointer drags from one paragraph into another and both are held as one
// run, as they read, rather than a block at a time. Shift-click and Shift
// with the arrows extend the run, and Copy puts it on the clipboard.
func (e *Editor) selectable(col ui.Element) ui.Element {
	return col.Selectable().CopyData(e.copySelection)
}

// copySelection is what Copy puts on the clipboard for a selection made
// across blocks: each paragraph's text, set apart from the next by a blank
// line, as the page reads. What was selected is what is copied — the marks
// of the source are not in it, as they are not on the page. A formula is
// the exception: it is copied as its source, $…$, since the picture it
// stands as is not text at all.
func (e *Editor) copySelection(parts []ui.SelectedText) transfer.Data {
	src := e.doc.Source()
	var segs []mdSeg
	at := 0
	var b strings.Builder
	for i, p := range parts {
		if i > 0 {
			// A paragraph of a list, or a cell of a table, is a part too:
			// the blank line is what sets the blocks of the page apart.
			b.WriteString("\n\n")
		}
		text := partText(p)
		if strings.ContainsRune(text, objectChar) {
			if segs == nil {
				segs = e.sourceSegments()
			}
			var k int
			text, k = withSources(src, segs, p, at)
			at = k + 1
		}
		b.WriteString(text)
	}
	return transfer.TextData(b.String())
}

// objectChar is the rune a formula stands as in a paragraph's text.
const objectChar = '\uFFFC'

// withSources is a part's text with each formula it holds written as its
// source, $…$, and the paragraph it was found in, for the parts that follow
// to be looked for after it. A part no paragraph shows keeps its text.
func withSources(src string, segs []mdSeg, p ui.SelectedText, from int) (string, int) {
	i := -1
	for k := from; k < len(segs); k++ {
		if segs[k].text == p.Text {
			i = k
			break
		}
	}
	if i < 0 {
		return partText(p), from
	}
	r := []rune(p.Text)
	a := min(max(p.Start, 0), len(r))
	z := min(max(p.End, 0), len(r))
	var b strings.Builder
	for k := a; k < z; k++ {
		if r[k] != objectChar || k+1 >= len(segs[i].at) {
			b.WriteRune(r[k])
			continue
		}
		start := segs[i].at[k]
		end := min(segs[i].end[k], len(src))
		if start < 0 || start >= end {
			continue
		}
		b.WriteString(src[start:end])
	}
	return b.String(), i
}

// partText is the text a part of a selection holds: its offsets count
// runes, and a part cut at either end holds only what is between them.
func partText(p ui.SelectedText) string {
	r := []rune(p.Text)
	a := min(max(p.Start, 0), len(r))
	z := min(max(p.End, 0), len(r))
	if a > z {
		a, z = z, a
	}
	return string(r[a:z])
}

// markSelection is the source of what the selection holds, as Markdown:
// from the byte where its first part starts to the byte where its last one
// ends, with the marks the page keeps out of sight between them — the
// asterisks, the links' addresses, the list's dashes.
func (e *Editor) markSelection(parts []ui.SelectedText) (string, bool) {
	segs := e.sourceSegments()
	if len(segs) == 0 || len(parts) == 0 {
		return "", false
	}
	src := e.doc.Source()
	first, last := parts[0], parts[len(parts)-1]
	i, from := findSegment(segs, 0, first.Text, first.Start)
	if i < 0 {
		return "", false
	}
	_, to := findSegment(segs, i, last.Text, last.End)
	if to < from {
		return "", false
	}
	return strings.TrimRight(src[from:to], " \t\n"), true
}

// sourceSegments are the paragraphs of the page as its source reads: the
// text each shows, and where each of its runes stands in the document's
// source, for a selection's ends to be found there.
func (e *Editor) sourceSegments() []mdSeg {
	src := e.doc.Source()
	var out []mdSeg
	for i := range e.doc.Blocks() {
		b, ok := e.doc.Block(i)
		if !ok {
			continue
		}
		text := strings.TrimRight(src[b.Start:b.End], "\n")
		// The same syntax the page reads: a formula between single dollars
		// is one object there, and has to be one here for its source to be
		// found.
		d := parser.New(parser.Default | parser.ExtDollarMath)
		d.Set(text)
		for k := 0; k < d.Len(); k++ {
			pb := d.Block(k)
			switch pb.Kind {
			case parser.Paragraph, parser.Heading, parser.TableCell:
			default:
				continue
			}
			if seg, ok := segmentOf(text, b.Start, d.Inlines(k)); ok {
				out = append(out, seg)
			}
		}
	}
	return out
}

// mdSeg is one paragraph of the page: the text it shows, the source byte
// each rune of it starts at, and the byte just past each of them — a
// formula stands as one rune and takes a whole span of the source.
type mdSeg struct {
	text string
	at   []int
	end  []int
}

// segmentOf reads the runes of a paragraph's inlines, and where each of
// them stands in the source: a code span's backticks and a link's address
// are not in the text it shows, so the runes are found in the inline's own
// source rather than counted from its start.
func segmentOf(src string, base int, inls []parser.Inline) (mdSeg, bool) {
	var text strings.Builder
	var at, ends []int
	for _, in := range inls {
		var shown string
		switch in.Kind {
		case parser.Math:
			// A formula stands as one object in the text, and its source is
			// the fence, the TeX and the fence again: the rune starts where
			// the first fence does, and ends past the last.
			text.WriteRune(objectChar)
			at = append(at, base+in.Start)
			ends = append(ends, base+in.End)
			continue
		case parser.Text, parser.Code, parser.HTML,
			parser.FootnoteRef, parser.Mention, parser.Reference:
			shown = in.Text
		case parser.SoftBreak:
			shown = " "
		case parser.LineBreak:
			shown = "\n"
		default:
			continue // a container: its children follow it
		}
		if shown == "" {
			continue
		}
		off := base + in.Start
		if in.End > in.Start && in.End <= len(src) {
			if k := strings.Index(src[in.Start:in.End], shown); k >= 0 {
				off = base + in.Start + k
			}
		}
		for _, r := range shown {
			text.WriteRune(r)
			at = append(at, off)
			off += utf8.RuneLen(r)
			ends = append(ends, off)
		}
	}
	if text.Len() == 0 || len(at) == 0 {
		return mdSeg{}, false
	}
	at = append(at, ends[len(ends)-1])
	return mdSeg{text: text.String(), at: at, end: ends}, true
}

// lastRune is the last rune of s, which is not empty.
func lastRune(s string) rune {
	r := []rune(s)
	return r[len(r)-1]
}

// findSegment is the paragraph at or after from that shows text, and the
// source byte the rune at stands at — the selection's start in the first
// part, its end in the last. It is -1 for a text no paragraph shows, as the
// source of a code block or a rule.
func findSegment(segs []mdSeg, from int, text string, at int) (int, int) {
	for i := from; i < len(segs); i++ {
		if segs[i].text != text {
			continue
		}
		at = min(max(at, 0), len(segs[i].at)-1)
		return i, segs[i].at[at]
	}
	return -1, 0
}

// selectMenu is the menu of the page while it selects: the text under the
// pointer as it reads, the source of it as Markdown, or the whole page.
func (e *Editor) selectMenu(c *ui.Context, m *ui.Menu, page ui.Element) {
	m.EditItems()
	m.Separator()
	it := m.Item(locales.CopyAsMarkdown())
	// The source is had only where the paragraphs of the page can be found
	// again: a selection that ends inside a code block or a rule has no
	// paragraph to stand in, and copies as its text alone.
	if md, ok := e.markSelection(page.SelectedTexts()); ok {
		if it.Chosen() {
			c.Services().WriteClipboard(md)
		}
	} else {
		it.Disabled(true)
	}
}
