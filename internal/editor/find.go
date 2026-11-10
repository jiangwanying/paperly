package editor

import (
	"fmt"
	"paperly/locales"

	"github.com/egoist/mygo/ui"

	"paperly/internal/doc"
)

// OpenFind shows the find bar, with the replace field when replace is set.
func (e *Editor) OpenFind(replace bool) {
	e.findOpen = true
	e.findReplace = replace
	e.refreshHits()
}

// CloseFind hides the bar and its marks.
func (e *Editor) CloseFind() {
	e.findOpen = false
	e.findReplace = false
	e.hits = nil
	e.hitAt = 0
	e.clearMarks()
}

// FindOpen reports whether the find bar shows.
func (e *Editor) FindOpen() bool { return e.findOpen }

// FindNext moves to the match after this one, round to the first.
func (e *Editor) FindNext() {
	if len(e.hits) == 0 {
		return
	}
	e.hitAt = (e.hitAt + 1) % len(e.hits)
	e.revealHit()
}

// FindPrev moves to the match before this one, round to the last.
func (e *Editor) FindPrev() {
	if len(e.hits) == 0 {
		return
	}
	e.hitAt = (e.hitAt - 1 + len(e.hits)) % len(e.hits)
	e.revealHit()
}

// HitCount is how many matches the find bar found, and where it stands
// among them, from 1.
func (e *Editor) HitCount() (at, total int) {
	if len(e.hits) == 0 {
		return 0, 0
	}
	return e.hitAt + 1, len(e.hits)
}

// ReplaceCurrent writes the replacement in place of the match the find bar
// stands on, and moves to the next.
func (e *Editor) ReplaceCurrent() {
	if len(e.hits) == 0 {
		return
	}
	if e.editing() {
		e.Commit()
	}
	h := e.hits[e.hitAt]
	e.record()
	e.doc.Replace(h.Span, e.replaceText)
	e.refreshHits()
	e.hitAt = min(e.hitAt, max(len(e.hits)-1, 0))
	e.revealHit()
}

// ReplaceAll writes the replacement in place of every match, and reports how
// many there were.
func (e *Editor) ReplaceAll() int {
	if e.editing() {
		e.Commit()
	}
	e.record()
	n := e.doc.ReplaceAll(e.query(), e.replaceText)
	e.refreshHits()
	return n
}

// query is what the find bar looks for.
func (e *Editor) query() doc.Query {
	return doc.Query{Text: e.findText, Case: e.findCase, Word: e.findWord}
}

// refreshHits looks for what the bar holds, and brings the first match into
// view.
func (e *Editor) refreshHits() {
	e.hits = e.doc.FindAll(e.query())
	e.clearMarks()
	if e.hitAt >= len(e.hits) {
		e.hitAt = 0
	}
	e.revealHit()
}

// refind looks for the text again after the document changed under the bar:
// the count and the marks follow what is written, without moving the view to
// the match the bar stands on, which typing would pull from under the caret.
func (e *Editor) refind() {
	if !e.findOpen || e.findText == "" {
		return
	}
	e.hits = e.doc.FindAll(e.query())
	e.clearMarks()
	if e.hitAt >= len(e.hits) {
		e.hitAt = max(len(e.hits)-1, 0)
	}
}

// revealHit brings the block of the match into view.
func (e *Editor) revealHit() {
	if len(e.hits) == 0 {
		return
	}
	if b := e.hits[e.hitAt].Block; b >= 0 {
		e.scrollTo = b
	}
}

// blockHit reports whether block i holds a match, and whether it holds the
// one the bar stands on.
func (e *Editor) blockHit(i int) (any, current bool) {
	if !e.findOpen {
		return false, false
	}
	for n, h := range e.hits {
		if h.Block == i {
			if n == e.hitAt {
				return true, true
			}
			any = true
		}
	}
	return any, false
}

// findRanges marks the matches inside s, whose first byte is at base in the
// source, for a text input that draws s.
func (e *Editor) findRanges(s string, base int, rest ui.Color, current ui.Color) []ui.TextRange {
	if !e.findOpen {
		return nil
	}
	var out []ui.TextRange
	for n, h := range e.hits {
		if h.Start < base || h.End > base+len(s) {
			continue
		}
		c := rest
		if n == e.hitAt {
			c = current
		}
		out = append(out, ui.TextRange{
			Start: doc.RuneOffset(s, h.Start-base),
			End:   doc.RuneOffset(s, h.End-base),
			Color: c,
		})
	}
	return out
}

// findBar is the bar above the status bar: what to look for, where it
// stands among the matches, the two arrows, the replacement and the close.
func (e *Editor) findBar(c *ui.Context, t *ui.Theme) {
	if !e.findOpen {
		return
	}
	ui.Box(c).Height(1).Background(t.Border)
	ui.Row(c).Padding(6, 12).Gap(8).AlignItems(ui.Center).
		Background(t.Surface).Children(func() {
		ui.Themed(c, t).Children(func() {
			in := ui.TextInput(c.Key("find"), &e.findText).Width(190).AutoFocus()
			in.FontSize(13)
			in.Background(t.Background)
			in.Border(1, t.Border)
			in.OnChange(e.refreshHits)
			in.OnTextKey(func(k ui.TextKey) bool {
				switch k.Key {
				case ui.KeyEscape:
					e.CloseFind()
					return true
				case ui.KeyEnter:
					if k.Mods&ui.Shift != 0 {
						e.FindPrev()
					} else {
						e.FindNext()
					}
					return true
				}
				return false
			})
		})

		// The case and whole-word switches, filled while they hold.
		e.findToggle(c, t, "Aa", e.findCase, func() {
			e.findCase = !e.findCase
			e.refreshHits()
		})
		e.findToggle(c, t, locales.FindWord(), e.findWord, func() {
			e.findWord = !e.findWord
			e.refreshHits()
		})

		at, total := e.HitCount()
		count := locales.FindNone()
		if total > 0 {
			count = fmt.Sprintf("%d/%d", at, total)
		}
		ui.Text(c, count).FontSize(12).TextColor(t.TextMuted)

		e.findButton(c, t, "↑", e.FindPrev)
		e.findButton(c, t, "↓", e.FindNext)

		if e.findReplace {
			ui.Themed(c, t).Children(func() {
				in := ui.TextInput(c.Key("replace"), &e.replaceText).Width(190)
				in.FontSize(13)
				in.Background(t.Background)
				in.Border(1, t.Border)
			})
			e.findButton(c, t, locales.ReplaceField(), e.ReplaceCurrent)
			e.findButton(c, t, locales.ReplaceAll(), func() {
				e.ReplaceAll()
			})
		}

		ui.Box(c).Grow(1)
		e.findButton(c, t, "×", e.CloseFind)
	})
}

// findButton is one of the bar's small buttons.
func (e *Editor) findButton(c *ui.Context, t *ui.Theme, label string, fn func()) {
	b := ui.Row(c).Padding(5, 9).Radius(4).Background(t.Text.Alpha(0.06))
	if b.Hovered() {
		b.Background(t.Text.Alpha(0.12))
	}
	b.OnClick(fn)
	b.Children(func() { ui.Text(c, label).FontSize(12).TextColor(t.Text) })
}

// findToggle is a switch of the bar, darker while it holds.
func (e *Editor) findToggle(c *ui.Context, t *ui.Theme, label string, on bool, fn func()) {
	b := ui.Row(c).Padding(5, 8).Radius(4)
	if on {
		b.Background(t.Accent.Alpha(0.18))
	}
	if b.Hovered() {
		b.Background(t.Text.Alpha(0.1))
	}
	b.OnClick(fn)
	b.Children(func() {
		el := ui.Text(c, label).FontSize(12).TextColor(t.TextMuted)
		if on {
			el.TextColor(t.Text).Bold()
		}
	})
}

// blockStart is the source offset the block being written starts at, for the
// search to place its marks in the text being written.
func (e *Editor) blockStart() int {
	if e.pending {
		return e.at
	}
	if b, ok := e.doc.Block(e.active); ok {
		return b.Start
	}
	return 0
}
