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
// of the source are not in it, as they are not on the page.
func (e *Editor) copySelection(parts []ui.SelectedText) transfer.Data {
	var b strings.Builder
	for i, p := range parts {
		if i > 0 {
			// A paragraph of a list, or a cell of a table, is a part too:
			// the blank line is what sets the blocks of the page apart.
			b.WriteString("\n\n")
		}
		b.WriteString(partText(p))
	}
	return transfer.TextData(b.String())
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
		d := parser.Parse(text)
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

// mdSeg is one paragraph of the page: the text it shows, and the source
// byte each rune of it starts at, with where the text ends after them.
type mdSeg struct {
	text string
	at   []int
}

// segmentOf reads the runes of a paragraph's inlines, and where each of
// them stands in the source: a code span's backticks and a link's address
// are not in the text it shows, so the runes are found in the inline's own
// source rather than counted from its start.
func segmentOf(src string, base int, inls []parser.Inline) (mdSeg, bool) {
	var text strings.Builder
	var at []int
	for _, in := range inls {
		var shown string
		switch in.Kind {
		case parser.Text, parser.Code, parser.HTML, parser.Math,
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
		}
	}
	if text.Len() == 0 || len(at) == 0 {
		return mdSeg{}, false
	}
	at = append(at, at[len(at)-1]+utf8.RuneLen(lastRune(text.String())))
	return mdSeg{text: text.String(), at: at}, true
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
