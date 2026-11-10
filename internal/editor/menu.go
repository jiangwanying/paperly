package editor

import (
	"paperly/locales"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown/parser"

	"paperly/internal/doc"
)

// headingLabels names the heading levels the menus offer, in the language
// the app shows.
func headingLabels() []string {
	return []string{locales.Heading1(), locales.Heading2(), locales.Heading3(),
		locales.Heading4(), locales.Heading5(), locales.Heading6()}
}

// blockMenu is the menu of a rendered block, in the order and with the
// words of Typora's: the clipboard, the paragraph's commands, the inline
// formats, insertion, and the way back to the outline.
func (e *Editor) blockMenu(c *ui.Context, m *ui.Menu, i int) {
	// Nothing is selected while the page is read, so these are dim, as
	// Typora's are.
	m.Item(locales.Cut()).Disabled(true)
	m.Item(locales.Copy()).Disabled(true)
	if m.Item(locales.Paste()).Chosen() {
		if text := strings.TrimSpace(c.Services().ReadClipboard()); text != "" {
			e.insertAfter(i, text)
		}
	}
	e.menuBody(c, m, i)
	// The delete of the block is first-level too: a picture right-clicked
	// is asked to go away here, not two menus deep.
	m.Separator()
	if m.Item(locales.DeleteBlock()).Chosen() {
		e.DeleteBlock(i)
	}
}

// editMenu is the menu of the text being written: MyGo's own cut, copy,
// paste and select all, which act on the input, then the rest of Typora's.
func (e *Editor) editMenu(c *ui.Context, m *ui.Menu, i int) {
	m.EditItems()
	m.Separator()
	e.menuBody(c, m, i)
}

// menuBody is what both menus share, from the clipboard's conversions down.
func (e *Editor) menuBody(c *ui.Context, m *ui.Menu, i int) {
	m.Submenu(locales.CopyPasteAs(), func(s *ui.Menu) {
		if s.Item(locales.CopyAsMarkdown()).Chosen() {
			e.copyBlock(c, i)
		}
		s.Item(locales.CopyAsHtml()).Disabled(true)
		s.Item(locales.PasteAsPlain()).Disabled(true)
	})
	m.Separator()

	m.Submenu(locales.MenuParagraph(), func(s *ui.Menu) { e.paragraphItems(s, i) })
	m.Submenu(locales.MenuFormat(), func(s *ui.Menu) { e.markItems(s, i) })
	m.Submenu(locales.Insert(), func(s *ui.Menu) { e.insertItems(s, i) })
	m.Separator()

	if m.Item(locales.RevealInSidebar()).Chosen() {
		e.revealInSidebar(i)
	}
	m.Separator()
	m.Item(locales.AutoFill()).Disabled(true)
}

// canvasMenu is the menu of the empty room around the text: no paragraph
// to change there, so it holds the clipboard and what can be put down.
func (e *Editor) canvasMenu(c *ui.Context, m *ui.Menu) {
	if m.Item(locales.Paste()).Chosen() {
		if text := strings.TrimSpace(c.Services().ReadClipboard()); text != "" {
			e.startAt(len(e.doc.Source()))
			e.draft = text
			e.Commit()
		}
	}
	if m.Item(locales.InsertParagraph()).Chosen() {
		e.StartParagraph()
	}
	m.Separator()
	m.Submenu(locales.Insert(), func(s *ui.Menu) { e.insertItems(s, -1) })
	m.Separator()
	m.Item(locales.AutoFill()).Disabled(true)
}

// paragraphItems are the commands of one paragraph, as Typora's 段落 menu
// lists them.
func (e *Editor) paragraphItems(s *ui.Menu, i int) {
	for level, label := range headingLabels() {
		l := level + 1
		if s.Item(label).Chosen() {
			e.applyTo(i, func(src string) string { return doc.SetHeading(src, l) })
		}
	}
	s.Separator()
	if s.Item(locales.PromoteHeading()).Chosen() {
		e.applyTo(i, doc.PromoteHeading)
	}
	if s.Item(locales.DemoteHeading()).Chosen() {
		e.applyTo(i, doc.DemoteHeading)
	}
	s.Separator()
	s.Item(locales.Table()).Disabled(true)
	if s.Item(locales.CodeBlock()).Chosen() {
		e.applyTo(i, doc.ToggleCode)
	}
	s.Item(locales.MathBlock()).Disabled(true)
	if s.Item(locales.Quote()).Chosen() {
		e.applyTo(i, doc.ToggleQuote)
	}
	if s.Item(locales.OrderedList()).Chosen() {
		e.applyTo(i, func(src string) string { return doc.ToggleList(src, doc.ListOrdered) })
	}
	if s.Item(locales.BulletList()).Chosen() {
		e.applyTo(i, func(src string) string { return doc.ToggleList(src, doc.ListBullet) })
	}
	if s.Item(locales.TaskList()).Chosen() {
		e.applyTo(i, func(src string) string { return doc.ToggleList(src, doc.ListTask) })
	}
	s.Separator()
	if s.Item(locales.Body()).Chosen() {
		e.applyTo(i, func(src string) string { return doc.SetHeading(src, 0) })
	}
	if s.Item(locales.InsertParagraphAbove()).Chosen() {
		e.insertBefore(i)
	}
	if s.Item(locales.InsertParagraphBelow()).Chosen() {
		e.InsertAfter(i)
	}
	if s.Item(locales.DeleteBlock()).Chosen() {
		e.DeleteBlock(i)
	}
}

// markItems are the inline formats, run on the selection of the block being
// written, or on the whole of block i when it is not.
func (e *Editor) markItems(s *ui.Menu, i int) {
	type format struct {
		label string
		mods  ui.Modifiers
		key   ui.Key
		mark  string
		open  string
		close string
	}
	formats := []format{
		{label: locales.Bold(), mods: ui.Cmd, key: ui.KeyB, mark: "**"},
		{label: locales.Italic(), mods: ui.Cmd, key: ui.KeyI, mark: "*"},
		{label: locales.Underline(), mods: ui.Cmd, key: ui.KeyU, open: "<u>", close: "</u>"},
		{label: locales.InlineCode(), mods: ui.Ctrl, key: ui.KeyBackquote, mark: "`"},
		{label: locales.Strike(), mods: ui.Ctrl | ui.Shift, key: ui.KeyBackquote, mark: "~~"},
		{label: locales.Highlight(), mods: ui.Cmd | ui.Shift, key: ui.KeyH, mark: "=="},
	}
	for _, f := range formats {
		if s.Item(f.label).Shortcut(f.mods, f.key).Chosen() {
			if f.open != "" {
				e.markAtPair(i, f.open, f.close)
			} else {
				e.markAt(i, f.mark)
			}
		}
	}
	s.Separator()
	if s.Item(locales.Link()).Shortcut(ui.Cmd, ui.KeyK).Chosen() {
		e.markAtPair(i, "[", "](url)")
	}
	s.Separator()
	if s.Item(locales.ClearStyle()).Shortcut(ui.Cmd, ui.KeyBackslash).Chosen() {
		e.applyTo(i, doc.StripInline)
	}
}

// insertItems are the blocks the menu can put below the one it opened on.
func (e *Editor) insertItems(s *ui.Menu, i int) {
	if s.Item(locales.Image()).Chosen() {
		e.ChooseImage()
	}
	if s.Item(locales.InsertTable()).Chosen() {
		e.insertAfter(i, locales.InsertTableTemplate())
	}
	s.Item(locales.MathBlock()).Disabled(true)
	if s.Item(locales.HorizontalRule()).Chosen() {
		e.insertAfter(i, "---")
	}
	if s.Item(locales.CodeBlock()).Chosen() {
		e.insertAfter(i, "```\n\n```")
	}
}

// copyBlock puts a block's source on the clipboard.
func (e *Editor) copyBlock(c *ui.Context, i int) {
	if b, ok := e.doc.Block(i); ok {
		c.Services().WriteClipboard(strings.TrimRight(e.doc.Source()[b.Start:b.End], "\n"))
	}
}

// revealInSidebar opens the outline and marks the section block i is in.
func (e *Editor) revealInSidebar(i int) {
	head := -1
	for j, b := range e.doc.Blocks() {
		if j > i {
			break
		}
		if b.Kind == parser.Heading {
			head = j
		}
	}
	e.sh.Show(SideOutline)
	if head >= 0 {
		e.scrollTo = head
	}
}

// markAt puts mark around the selection of block i, or around the block.
func (e *Editor) markAt(i int, mark string) {
	if e.active == i {
		e.mark(mark)
		return
	}
	e.applyTo(i, func(src string) string {
		out, _, _ := doc.Wrap(src, 0, doc.RuneLen(src), mark)
		return out
	})
}

// markAtPair puts a pair of marks around the selection of block i, or
// around the block.
func (e *Editor) markAtPair(i int, open, close string) {
	if e.active == i {
		e.markPair(open, close)
		return
	}
	e.applyTo(i, func(src string) string {
		out, _, _ := doc.WrapPair(src, 0, doc.RuneLen(src), open, close)
		return out
	})
}

// insertAfter puts text in a block of its own below block i, or at the end
// of the document when there is no block, as while one is being written.
func (e *Editor) insertAfter(i int, text string) {
	if i < 0 {
		e.startAt(len(e.doc.Source()))
	} else {
		e.InsertAfter(i)
	}
	e.draft = text
	e.Commit()
}

// insertBefore puts an empty block above block i, ready to write in.
func (e *Editor) insertBefore(i int) {
	b, ok := e.doc.Block(i)
	if !ok {
		return
	}
	e.startAt(b.Start)
}
