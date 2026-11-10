package editor

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"

	"paperly/internal/doc"
	"paperly/locales"
)

// The item being written. A list of two items and more edits by the item,
// as Typora's blocks do and Obsidian's Live Preview draws: the item the
// caret is in shows as its source, and the others show as they will read,
// laid out and styled as the reading view lays them out. The draft stays
// the source of the whole block; only what the input holds is one line of
// it, which goes back into the draft, and with it to the document, as each
// character lands.

// itemsMode tells whether the block being written edits by the item: a
// list whose every line opens with a marker of its own. A quote keeps
// whole-block writing, and so does any line without a marker, as a wrapped
// one is.
func (e *Editor) itemsMode() bool {
	if !strings.Contains(e.draft, "\n") {
		return false
	}
	for _, line := range linesOf(e.draft) {
		t := strings.TrimSpace(line)
		if t == "" {
			continue
		}
		_, quote, _, _, ok := listMark(line)
		if !ok || quote {
			// A line whose content was deleted back to the marker itself,
			// as "- " loses its room: it is an item being emptied, not the
			// end of item writing.
			if !reBareMark.MatchString(t) {
				return false
			}
		}
	}
	return true
}

// reBareMark is a marker with nothing after it, the space gone with the
// text that was deleted.
var reBareMark = regexp.MustCompile(`^([-*+]|\d+[.)])$`)

// mergeJoin computes the line that joins the item cur to the one above it
// (prev), and the caret at the seam. The content of an emptied item — the
// marker alone, its room gone with the deleted text — joins as nothing.
func mergeJoin(prev, cur string) (string, int) {
	_, _, mark, _, marked := listMark(cur)
	body := cur
	if marked {
		body = strings.TrimPrefix(cur, mark)
	} else if reBareMark.MatchString(strings.TrimSpace(cur)) {
		body = "" // the marker itself: an empty item joins as nothing
	}
	head := strings.TrimRight(prev, " \t")
	merged, caret := head, doc.RuneLen(head)
	if strings.TrimSpace(body) != "" {
		merged += " " + body
		caret++
	}
	return merged, caret
}

// linesOf splits a run of source into its lines.
func linesOf(s string) []string { return strings.Split(s, "\n") }

// lineStartRune is the rune the line n of the draft starts at.
func (e *Editor) lineStartRune(n int) int {
	start := 0
	for _, line := range linesOf(e.draft) {
		if n <= 0 {
			return start
		}
		n--
		start += doc.RuneLen(line) + 1
	}
	return start
}

// resetItem leaves item writing, as leaving the block does.
func (e *Editor) resetItem() {
	e.itemLine, e.lineDraft, e.itemCaret = -1, "", -1
}

// refocusItem moves the writing caret, which counts runes of the whole
// draft, into the item that now holds it, and makes that the item being
// written. It follows a Return, which worked on the draft as a whole.
func (e *Editor) refocusItem() {
	if !e.itemsMode() || e.caret < 0 {
		return
	}
	pos, start := e.caret, 0
	lines := linesOf(e.draft)
	for n, line := range lines {
		end := start + doc.RuneLen(line)
		if pos <= end {
			e.itemLine, e.lineDraft, e.caret = n, line, -1
			e.itemCaret = pos - start
			return
		}
		start = end + 1
	}
	last := len(lines) - 1
	e.itemLine, e.lineDraft, e.caret = last, lines[last], -1
	e.itemCaret = doc.RuneLen(lines[last])
}

// resyncItem follows the draft that the sync may have rewritten: the item
// being written keeps its place in it, or the block writes whole again
// when its lines stopped being items.
func (e *Editor) resyncItem() {
	if e.itemLine < 0 {
		return
	}
	if !e.itemsMode() {
		e.resetItem()
		return
	}
	lines := linesOf(e.draft)
	if e.itemLine >= len(lines) {
		e.itemLine = len(lines) - 1
	}
	if line := lines[e.itemLine]; line != e.lineDraft {
		e.lineDraft = line
	}
}

// focusItem puts the writing caret in the item n, at its end when atEnd
// says so, at its start otherwise. A click on a rendered item asks for it.
func (e *Editor) focusItem(n int, atEnd bool) {
	lines := linesOf(e.draft)
	if n < 0 || n >= len(lines) {
		return
	}
	e.itemLine, e.lineDraft = n, lines[n]
	e.itemCaret = 0
	if atEnd {
		e.itemCaret = doc.RuneLen(lines[n])
	}
}

// setLine writes the text of the item n back into the draft, which goes on
// to the document as each character lands. An item emptied back to its
// marker is written as the marker with its room — the parser holds the
// whole line, and the next Backspace, at the line's first letter, joins it
// to the one above.
func (e *Editor) setLine(n int, text string) {
	lines := linesOf(e.draft)
	if n < 0 || n >= len(lines) || lines[n] == text {
		return
	}
	if t := strings.TrimSpace(text); reBareMark.MatchString(t) && !strings.HasSuffix(text, " ") {
		// The text lost its room to the Backspace: the room comes back,
		// and the caret, which the deleted character held, goes to the
		// line's first letter, where the next Backspace joins the item.
		text = t + " "
		if n == e.itemLine {
			e.itemCaret = 0
		}
	}
	lines[n] = text
	e.draft = strings.Join(lines, "\n")
	e.sync()
	e.resyncItem()
}

// itemToParagraph takes an emptied item out of the list, as the Backspace
// at its first letter asks: the marker goes with the line, and a paragraph
// of its own opens where the item was, set off from the list on both
// sides, for the next characters to fill.
func (e *Editor) itemToParagraph() {
	e.record()
	b, ok := e.doc.Block(e.active)
	if !ok {
		return
	}
	lines := linesOf(e.draft)
	i := e.itemLine
	if i >= len(lines) {
		return
	}
	before := strings.Join(lines[:i], "\n")
	after := strings.Join(lines[i+1:], "\n")
	// The line goes, and the breaks on both sides of it keep whatever
	// stays apart: the writing that follows starts a paragraph there.
	// The region is the whole room the block had — the parser may hold
	// less of an emptied line than the draft does, and what it set apart
	// goes with the line.
	src := e.doc.Source()
	region := b.Span
	if nb, ok := e.doc.Block(e.active + 1); ok {
		region.End = nb.Start
	} else {
		region.End = len(src)
	}
	var body string
	at := b.Start
	if before != "" {
		body += before + "\n\n"
		at += len(before) + 2
	}
	if after != "" {
		body += "\n\n" + after
	}
	e.doc.SetSource(src[:b.Start] + body + src[region.End:])
	e.active, e.pending, e.at, e.draft = -1, true, at, ""
	e.wrote = at
	e.resetItem()
	e.refind()
}

// mergeItemUp joins the item being written to the one above it, as
// Backspace at its first letter does: the break and the lower item's
// marker go, and the caret waits at the seam. It is called from the key,
// so the input is told its new text through k: writing the bound value
// alone leaves the editor to write what it holds over it.
func (e *Editor) mergeItemUp(k *ui.TextKey) {
	if e.itemLine <= 0 {
		return
	}
	lines := linesOf(e.draft)
	prev, cur := lines[e.itemLine-1], lines[e.itemLine]
	merged, caret := mergeJoin(prev, cur)
	lines[e.itemLine-1] = merged
	lines = append(lines[:e.itemLine], lines[e.itemLine+1:]...)
	e.itemLine--
	e.draft = strings.Join(lines, "\n")

	e.lineDraft = merged
	e.itemCaret = caret
	if k != nil {
		k.SetText(merged)
	}
	e.sync()
	e.resyncItem()
}

// lineBase is where the item being written sits in the document, so that
// the hits of a search know what part of it this input holds.
func (e *Editor) lineBase() int {
	base, at := e.blockStart(), 0
	for n, line := range linesOf(e.draft) {
		if n == e.itemLine {
			return base + at
		}
		at += len(line) + 1
	}
	return base
}

// editItems is the active block of a list, item by item: the item the
// caret is in shows as its source, and the others show as they will read.
func (e *Editor) editItems(c *ui.Context, t *ui.Theme) {
	lines := linesOf(e.draft)
	if e.itemLine < 0 || e.itemLine >= len(lines) {
		e.itemLine, e.lineDraft = 0, lines[0]
	}
	ui.Column(c).Gap(0).Children(func() {
		for n, line := range lines {
			n := n
			if n == e.itemLine {
				e.editItem(c, t)
				continue
			}
			ui.Column(c.Key("it-" + fmt.Sprint(n))).OnClick(func() { e.focusItem(n, true) }).Children(func() {
				markdown.View(c, readableSrc(line), &e.md)
			})
		}
	})
}

// editItem is the input of the item being written: its source, with the
// marks faint, and the keys that move along the list around it.
func (e *Editor) editItem(c *ui.Context, t *ui.Theme) {
	ta := ui.TextAreaBase(c.Key("item"), &e.lineDraft).Lines(1, 40).Padding(0, 0).AutoFocus()
	ta.Label(locales.EditArea())
	ta.Border(0, ui.Color{})
	ta.Background(ui.Color{})
	ta.FocusRing(false)
	ta.ContextMenu(func(m *ui.Menu) { e.editMenu(c, m, e.active) })
	ta.OnPaste(func(text string) bool {
		if s := strings.TrimSpace(text); s != "" && !isImage(s) {
			return false
		}
		return e.PasteImage()
	})
	// The one line is the line the caret is on: every mark of it shows.
	ranges := e.ranges(e.lineDraft, 0, t.TextMuted, t.Text, t.Background)
	ranges = append(ranges, e.findRanges(e.lineDraft, e.lineBase(), e.highlightColor(), e.highlightColor())...)
	ta.TextRanges(ranges...)
	if e.itemCaret >= 0 {
		// The input applies this caret in its own time; the mirror takes
		// the value now, so that a key in the next frames reads the caret
		// that was asked for, not the one it had.
		ta.SetTextSelection(e.itemCaret, e.itemCaret)
		e.selA, e.selB = e.itemCaret, e.itemCaret
		e.itemCaret = -1
	}
	ta.OnChange(func() { e.setLine(e.itemLine, e.lineDraft) })
	e.selA, e.selB = ta.TextSelection()
	ta.OnTextKey(func(k ui.TextKey) bool {
		switch k.Key {
		case ui.KeyEscape:
			e.Commit()
			return true
		case ui.KeyV:
			// A picture on the clipboard goes in as one; text goes on to
			// the input, which pastes it as it always did.
			if k.Mods&(ui.Cmd|ui.Ctrl) != 0 && k.Mods&(ui.Shift|ui.Alt) == 0 {
				if e.PasteImage() {
					return true
				}
			}
			return false
		case ui.KeyBackspace:
			if e.selA == 0 && e.selB == 0 {
				// The item emptied — only its marker, or nothing, is left:
				// the Backspace takes the marker out, as Typora's does,
				// and a paragraph of its own opens where the item was.
				if t := strings.TrimSpace(e.lineDraft); t == "" || reBareMark.MatchString(t) {
					e.itemToParagraph()
					return true
				}
				if e.itemLine > 0 {
					e.mergeItemUp(&k)
					return true
				}
			}
			return false
		case ui.KeyUp:
			if e.selA == 0 && e.selB == 0 && e.itemLine > 0 {
				e.focusItem(e.itemLine-1, true)
				return true
			}
			return false
		case ui.KeyDown:
			if e.selA == e.selB && e.selA == doc.RuneLen(e.lineDraft) &&
				e.itemLine < len(linesOf(e.draft))-1 {
				e.focusItem(e.itemLine+1, false)
				return true
			}
			return false
		case ui.KeyEnter:
			// Return ends the item where the caret is, as Typora does.
			if k.Mods&(ui.Shift|ui.Alt|ui.Ctrl|ui.Cmd) != 0 {
				return false
			}
			e.splitAtCaret()
			return true
		}
		return false
	})
}
