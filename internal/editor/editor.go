// Package editor shows a document and edits it a block at a time: the block
// the caret is in shows its source, and the others show what they become.
package editor

import (
	"paperly/locales"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/egoist/mygo/ui/markdown/parser"

	"paperly/internal/doc"
)

// Editor is one window's view of a document. Its fields are read and
// changed on the main thread, by the view and by the commands.
type Editor struct {
	doc    *doc.Doc
	sh     *Shared
	md     markdown.Options
	mdBase *markdown.Theme // the document's own theme, before the find color
	mdMark *markdown.Theme // and with it, while the find bar is open
	ui     *ui.Theme
	style  int // the Shared.Style the themes were built for
	win    *mygo.Window
	file   *mygo.Document[string] // nil when the editor shows a document of its own

	svc         ui.Services    // the window's clipboard and files, from the last frame
	findOpen    bool           // the find bar shows
	findReplace bool           // and its replacement field
	findText    string         // what it looks for
	replaceText string         // and what it writes in place
	findCase    bool           // matches the case of the text
	findWord    bool           // matches whole words
	hits        []doc.Hit      // where the text is found
	hitAt       int            // the match the bar stands on
	marked      map[int]string // blocks with their matches wrapped, by block
	markedRev   uint64         // the document revision they were written for
	markCol     ui.Color       // the color <mark> was painted in

	source     bool   // the whole document shows as its source
	sourceText string // that source, while it is written
	mdDir      string // the folder the image loader was made for
	active     int    // the block being edited, -1 when none
	pending    bool   // writing a block the document does not have yet
	at         int    // where that block goes, a byte offset in the source
	draft      string
	selA       int // the selection of the text input, in runes, as of the last frame
	selB       int
	wrote      int    // where the text just committed ends, in bytes
	caret      int    // the rune to put the input's caret at once, -1 for none
	itemLine   int    // the item of a list being written, -1 for whole-block mode
	lineDraft  string // that item's source, which its input holds
	itemCaret  int    // the rune to put that input's caret at once, -1 for none

	desc      []histEntry // what the document was, for undo
	redoStack []snapshot  // and what undo took away, for redo
	recAt     time.Time   // when the last step was recorded

	scroll   ui.ScrollState
	pageH    float32         // the room the page has, from the last frame
	pageW    float32         // and its width
	tops     map[int]float32 // where each block starts, in the page, from the last frame
	scrollTo int             // the block to bring into view, -1 for none

	filesDir  string          // the folder the file list shows
	files     []fileItem      // the Markdown files in it
	filesAt   time.Time       // when they were read
	collapsed map[string]bool // the folders of the tree that are closed
	folded    map[int]bool    // the headings of the outline that are closed
	filtering bool            // the panel's search field shows
	filter    string          // what the panel is filtered by

	tedit      TreeEdit    // a name of the tree being written
	searchQ    string      // what the folder was last searched for
	searchDir  string      // the folder that was searched
	searchWhen time.Time   // when, for the search to read it once a second
	searchRes  []searchHit // what it found

	testAsk   func(string) bool // a test's answer to the delete dialog
	testAlert func(error)       // a test's taking of the error dialog
	dbg       string            // a test's window on the merge's stages

	zoomAddr  string // the picture the veil over the page shows
	zoomCache map[string]ui.ImageSource
}

// New shows d in the look sh describes.
func New(d *doc.Doc, sh *Shared) *Editor {
	return &Editor{doc: d, sh: sh, active: -1, scrollTo: -1, pageH: 600, caret: -1,
		itemLine: -1, itemCaret: -1,
		tops: map[int]float32{}, marked: map[int]string{}}
}

// Attach gives the editor the window that shows it and the document it
// saves to, for the status of the file. A document of its own leaves file
// nil.
func (e *Editor) Attach(win *mygo.Window, file *mygo.Document[string]) {
	e.win, e.file = win, file
}

// Reload shows the file's text again, as after a revert or a change made by
// another app. A block being edited is left as it was: the user's own work
// is not thrown away by a background change.
func (e *Editor) Reload(src string) {
	if e.editing() {
		return
	}
	e.doc.SetSource(src)
	e.refind()
}

// Path is the file the editor shows, or empty when it has none.
func (e *Editor) Path() string { return e.path() }

// Doc is the document the editor shows.
func (e *Editor) Doc() *doc.Doc { return e.doc }

// Active is the block being edited, or -1.
func (e *Editor) Active() int { return e.active }

// Editing reports whether a block is being written, a new one included.
func (e *Editor) Editing() bool { return e.editing() }

func (e *Editor) editing() bool { return e.active >= 0 || e.pending }

// Draft is the source of the block being written.
func (e *Editor) Draft() string { return e.draft }

// Pending reports whether the block being written is new.
func (e *Editor) Pending() bool { return e.pending }

// Activate turns a rendered block into its source, in place.
func (e *Editor) Activate(i int) {
	if _, ok := e.doc.Block(i); !ok {
		return
	}
	if e.editing() {
		e.Commit()
	}
	b, ok := e.doc.Block(i)
	if !ok {
		return
	}
	e.active, e.pending = i, false
	e.draft = strings.TrimRight(e.doc.Source()[b.Start:b.End], "\n")
	e.itemLine, e.lineDraft, e.itemCaret = -1, "", -1
	if e.itemsMode() {
		e.itemLine, e.lineDraft, e.itemCaret = 0, linesOf(e.draft)[0], 0
	}
}

// StartParagraph writes a new block after the last one, as a click in the
// empty room below the text does. The block joins the document when what was
// written is kept: an empty one leaves no trace.
func (e *Editor) StartParagraph() {
	e.startAt(len(e.doc.Source()))
}

// InsertAfter writes a new block below block i.
func (e *Editor) InsertAfter(i int) {
	b, ok := e.doc.Block(i)
	if !ok {
		return
	}
	e.startAt(b.End)
}

func (e *Editor) startAt(at int) {
	if e.pending && e.at == at {
		return
	}
	if e.editing() {
		e.Commit()
	}
	if at < 0 {
		at = 0
	}
	if n := len(e.doc.Source()); at > n {
		at = n
	}
	e.active, e.pending, e.at, e.draft = -1, true, at, ""
	e.resetItem()
}

// Cancel drops the block being written.
func (e *Editor) Cancel() {
	e.active, e.pending, e.draft = -1, false, ""
	e.resetItem()
}

// sync writes the draft into the document as it is typed, without leaving
// the block being written, so that the file is never behind the screen: a
// save while writing keeps what was written.
func (e *Editor) sync() {
	e.record()
	if e.pending {
		if strings.TrimSpace(e.draft) == "" {
			return
		}
		out, start := insertBlock(e.doc.Source(), e.at, e.draft)
		e.doc.SetSource(out)
		e.wrote = start + len(e.draft)
		e.pending = false
		if idx := e.doc.BlockAt(start); idx >= 0 {
			e.active = idx
		} else {
			e.active = -1
		}
		e.refind()
		return
	}
	if e.active < 0 {
		return
	}
	b, ok := e.doc.Block(e.active)
	if !ok {
		return
	}
	// An empty block is one the parser holds as nothing — a heading whose #
	// was deleted again. Its room in the source goes with it, and writing
	// goes on in a fresh paragraph here.
	if e.draft == "" && b.Start == b.End {
		src := e.doc.Source()
		prevEnd, nextStart := 0, len(src)
		if e.active > 0 {
			if pb, ok := e.doc.Block(e.active - 1); ok {
				prevEnd = pb.End
			}
		}
		if nb2, ok := e.doc.Block(e.active + 1); ok {
			nextStart = nb2.Start
		}
		sep := ""
		if prevEnd > 0 && nextStart < len(src) {
			sep = "\n\n"
		}
		e.doc.SetSource(src[:prevEnd] + sep + src[nextStart:])
		e.active, e.pending, e.at = -1, true, prevEnd
		e.resetItem()
		e.refind()
		return
	}
	// The parser holds less than the draft while a line it has no room for
	// is being written — an empty "2. " or "> " under a list or a quote:
	// the room it dropped goes with the draft, so that writing replaces it
	// whole instead of leaving it behind.
	span := b.Span
	// The writer's block is the last one: whatever trails it — the room a
	// deleted line left, a stray mark the parser set apart — is theirs to
	// replace with what the draft holds.
	if e.active == e.doc.Len()-1 {
		span.End = len(e.doc.Source())
	}
	if n := len(e.draft); n > span.End-span.Start {
		end := b.Start + n
		if end > len(e.doc.Source()) {
			end = len(e.doc.Source())
		}
		if end >= span.End && e.doc.Source()[span.End:end] == e.draft[span.End-b.Start:span.End-b.Start+end-span.End] {
			span.End = end
		}
	}
	e.doc.Replace(span, e.draft)
	e.refind()
	nb, ok := e.doc.Block(e.active)
	if !ok {
		// The block went away with its text: keep writing where it was, as
		// a block the document will take again with the next character.
		e.active, e.pending, e.at = -1, true, b.Start
		e.resetItem()
		return
	}
	// The parser cuts the trailing spaces of a last line off its block, so
	// a "- " just typed ends the span before the draft ends: what counts is
	// the text without that trailing room.
	if strings.TrimRight(e.doc.Source()[nb.Start:nb.End], " \t\n") != strings.TrimRight(e.draft, " \t\n") {
		// The parser holds less than the draft when a line it has no room
		// for is still being written — an empty "2. " or "> " under a list
		// or a quote: keep writing, and the text that follows makes the
		// line a real one.
		if end := b.Start + len(e.draft); end >= nb.End && end <= len(e.doc.Source()) &&
			e.doc.Source()[nb.End:end] == e.draft[nb.End-b.Start:] {
			return
		}
		// The text became more than one block, or joined its neighbour: the
		// block the caret is in now is the one holding the end of it, and
		// that is where writing goes on.
		end := b.Start + len(e.draft)
		if idx := e.doc.BlockAt(end - 1); idx >= 0 && end > b.Start {
			last, _ := e.doc.Block(idx)
			if idx != e.active {
				e.active = idx
				e.draft = strings.TrimRight(e.doc.Source()[last.Start:last.End], "\n")
				e.resyncItem()
				return
			}
			// The block holds more than what was written — the room a
			// deleted line left, or a mark the parser set apart, came
			// back with it. What the writer holds is the truth: the rest
			// goes, or it would be adopted as the text they typed.
			e.doc.Replace(last.Span, strings.TrimRight(e.draft, "\n"))
			e.refind()
			return
		}
		// The text ended in blank lines, as two Returns at the end do: the
		// next character starts a paragraph right there.
		e.active, e.pending, e.at, e.draft = -1, true, b.Start+len(e.draft), ""
		e.wrote = e.at
		e.resetItem()
	}
}

// Commit writes the edited source back into the document, which parses
// again from the block before the change. A new block with nothing in it is
// dropped rather than added.
func (e *Editor) Commit() {
	e.record()
	if e.pending {
		text := strings.TrimSpace(e.draft)
		at := e.at
		e.pending, e.draft = false, ""
		e.resetItem()
		if text == "" {
			e.refind()
			return
		}
		out, start := insertBlock(e.doc.Source(), at, text)
		e.doc.SetSource(out)
		e.wrote = start + len(text)
		e.refind()
		return
	}
	if e.active < 0 {
		return
	}
	b, ok := e.doc.Block(e.active)
	text := strings.TrimRight(e.draft, "\n")
	e.active, e.draft = -1, ""
	e.resetItem()
	if !ok {
		return
	}
	e.doc.Replace(b.Span, text)
	e.wrote = b.Start + len(text)
	e.refind()
}

// insertBlock puts text into src as a paragraph of its own at the byte at,
// set apart from its neighbours by a blank line, and keeping the blank
// lines the writer left around it: Return on an empty paragraph makes
// another, and what is written under them stays under them. It returns the
// source and where the text starts.
func insertBlock(src string, at int, text string) (string, int) {
	at = min(max(at, 0), len(src))
	before, after := src[:at], src[at:]
	if t := strings.TrimRight(before, "\n"); before != "" && len(before)-len(t) < 2 {
		before = t + "\n\n"
	}
	if t := strings.TrimLeft(after, "\n"); after != "" && len(after)-len(t) < 2 {
		after = "\n\n" + t
	}
	return before + text + after, len(before)
}

// splitAtCaret ends the block where the caret is and starts the next one
// there, as Return does in Typora: the paragraph appears under the caret,
// not at the end of the document. Return on an empty paragraph leaves a
// blank line under it, and another for each Return after that. Return at
// the end of a list item or a quote continues it with its marker, and an
// ordered one counts up; on a marker with nothing after it, it leaves the
// list; in the middle of an item, it splits it in two, both of them items.
// Return after ``` or ```lang opens the body of a code block, and inside
// one it adds a line to the code.
func (e *Editor) splitAtCaret() {
	if e.pending && strings.TrimSpace(e.draft) == "" {
		e.blankLine()
		return
	}
	// The selection of the input counts runes of what it holds: the whole
	// draft in whole-block writing, one item of it in item writing.
	sel := e.selInDraftA()
	at := doc.ByteOffset(e.draft, sel)
	// Return inside a code block adds a line to it: the fence is part of
	// the draft, so the line stays in the code.
	if strings.Contains(e.draft, "\n") && strings.HasPrefix(strings.TrimLeft(e.draft, " \t"), "```") {
		e.draft = e.draft[:at] + "\n" + e.draft[at:]
		e.sync()
		return
	}
	// Return after ``` or ```lang on a line of its own opens the body of a
	// code block, with the caret inside it.
	if m := reFence.FindStringSubmatch(strings.TrimSpace(e.draft)); m != nil {
		e.record()
		e.draft = "```" + m[1] + "\n\n```"
		e.sync()
		e.caret = doc.RuneOffset(e.draft, len("```"+m[1])+1)
		return
	}
	// The line the caret is on, and the marker it opens with, if any.
	ls := strings.LastIndexByte(e.draft[:at], '\n') + 1
	le := len(e.draft)
	if i := strings.IndexByte(e.draft[ls:], '\n'); i >= 0 {
		le = ls + i
	}
	line := e.draft[ls:le]
	kind, quote, mark, num, marked := listMark(line)
	if marked {
		// Return on a line that is nothing but its marker leaves the list
		// or the quote: the line goes, and a plain paragraph stays. The
		// parser may hold less of it than the draft does — an empty "2. "
		// or "> " — so the line comes out of the source whole.
		if strings.TrimSpace(line[len(mark):]) == "" && at >= ls+len(mark) {
			e.record()
			b, _ := e.doc.Block(e.active)
			end := le
			if b.End > end {
				end = b.End
			}
			cut := ls
			if ls > 0 {
				cut = ls - 1 // the break before the line goes with it
			}
			e.doc.SetSource(e.doc.Source()[:cut] + e.doc.Source()[end:])
			e.active, e.pending, e.at, e.draft = -1, true, cut, ""
			e.resetItem()
			e.refind()
			return
		}
		next := nextMark(kind, quote, mark, num)
		// Return at the end of an item continues it: the marker opens the
		// next one, where the caret waits.
		if at == le {
			e.record()
			e.draft = e.draft[:le] + "\n" + next + e.draft[le:]
			e.sync()
			e.caret = doc.RuneOffset(e.draft, le) + 1 + doc.RuneLen(next)
			e.refocusItem()
			return
		}
		// Return in the middle of an item splits it in two, both of them
		// items.
		if at >= ls+len(mark) {
			e.record()
			e.draft = e.draft[:at] + "\n" + next + e.draft[at:]
			e.sync()
			e.caret = doc.RuneOffset(e.draft, at) + 1 + doc.RuneLen(next)
			e.refocusItem()
			return
		}
	}
	start := sel
	mid := doc.ByteOffset(e.draft, start)
	before, after := e.draft[:mid], e.draft[mid:]
	e.draft = before
	e.Commit()
	e.startAt(e.wrote)
	e.draft = strings.TrimLeft(after, "\n")
	if strings.TrimSpace(e.draft) != "" {
		e.sync()
	}
}

// blankLine writes a blank line where the new paragraph is, so that Return
// on an empty one makes another, as Typora does: the caret moves down a
// line and the line stays in the file.
func (e *Editor) blankLine() {
	src := e.doc.Source()
	at := min(max(e.at, 0), len(src))
	from := 0
	if i := e.pendingAfter(); i >= 0 {
		if b, ok := e.doc.Block(i); ok {
			from = b.End
		}
	}
	// A paragraph is set apart by a blank line already: the first Return
	// writes both its breaks, and each Return after that writes one more.
	add := 3 - strings.Count(src[from:at], "\n")
	if add < 1 {
		add = 1
	}
	e.record()
	e.doc.SetSource(src[:at] + strings.Repeat("\n", add) + src[at:])
	e.at = at + add
	e.refind()
}

// snapshot is the editor as it was before a change: the document, and where
// writing was.
type snapshot struct {
	src     string
	active  int
	pending bool
	at      int
}

// histEntry is a snapshot with what the step after it is joined to: the steps
// of one block, in a row, undo together as a word typed does.
type histEntry struct {
	snapshot
	block int
	when  time.Time
}

// snapshot is the editor as it stands.
func (e *Editor) snapshot() snapshot {
	return snapshot{src: e.doc.Source(), active: e.active, pending: e.pending, at: e.at}
}

// record keeps the state before a change, so that undo can put it back. Steps
// of the same block in a row are one step, so that a word typed is undone at
// once rather than a letter at a time.
func (e *Editor) record() {
	n := e.snapshot()
	if k := len(e.desc); k > 0 {
		last := &e.desc[k-1]
		if last.block == n.active && time.Since(last.when) < 900*time.Millisecond {
			last.when = time.Now()
			e.redoStack = e.redoStack[:0]
			return
		}
	}
	e.desc = append(e.desc, histEntry{snapshot: n, block: n.active, when: time.Now()})
	if len(e.desc) > 200 {
		e.desc = e.desc[1:]
	}
	e.redoStack = e.redoStack[:0]
}

// Undo puts the document back as it was before the last change.
func (e *Editor) Undo() {
	for len(e.desc) > 0 {
		last := e.desc[len(e.desc)-1]
		e.desc = e.desc[:len(e.desc)-1]
		if last.src == e.doc.Source() {
			continue
		}
		e.redoStack = append(e.redoStack, e.snapshot())
		e.restore(last.snapshot)
		return
	}
}

// Redo puts back what Undo took away.
func (e *Editor) Redo() {
	for len(e.redoStack) > 0 {
		n := len(e.redoStack) - 1
		last := e.redoStack[n]
		e.redoStack = e.redoStack[:n]
		if last.src == e.doc.Source() {
			continue
		}
		e.desc = append(e.desc, histEntry{snapshot: e.snapshot(), block: e.active, when: time.Now()})
		e.restore(last)
		return
	}
}

// restore puts the document and the writing back as s has them: the block
// being written shows its text again, from the restored source.
func (e *Editor) restore(s snapshot) {
	if s.src != e.doc.Source() {
		e.doc.SetSource(s.src)
	}
	e.active, e.pending, e.at = s.active, s.pending, s.at
	e.draft = ""
	if e.active >= 0 {
		if b, ok := e.doc.Block(e.active); ok {
			e.draft = strings.TrimRight(e.doc.Source()[b.Start:b.End], "\n")
		} else {
			e.active = -1
		}
	}
	e.selA = doc.RuneOffset(e.draft, len(e.draft))
	e.selB = e.selA
	e.resyncItem()
	if e.itemLine >= 0 {
		e.selA = doc.RuneLen(e.lineDraft)
		e.selB = e.selA
		e.itemCaret = e.selA
	}
	e.refind()
}

// DeleteBlock takes a block out of the document, and the blank lines around
// it, so that it leaves no empty paragraph behind.
func (e *Editor) DeleteBlock(i int) {
	b, ok := e.doc.Block(i)
	if !ok {
		return
	}
	if e.active == i || e.pending {
		e.Cancel()
	}
	src := e.doc.Source()
	s := b.Span
	if s.End < len(src) {
		for s.End < len(src) && src[s.End] == '\n' {
			s.End++
		}
	} else {
		for s.Start > 0 && src[s.Start-1] == '\n' {
			s.Start--
		}
	}
	e.record()
	e.doc.Replace(s, "")
	e.refind()
}

// The commands of the menu bar and the context menu. Those that rewrite a
// block need one being written; the ones the context menu runs take the
// block they were opened on.

// SetHeading makes the block being written a heading of level 1 to 6, or a
// paragraph for 0.
func (e *Editor) SetHeading(level int) {
	e.apply(func(src string) string { return doc.SetHeading(src, level) })
}

// ToggleQuote quotes the block being written, or unquotes it.
func (e *Editor) ToggleQuote() { e.apply(doc.ToggleQuote) }

// ToggleList makes the block being written a list, or takes the list off it.
func (e *Editor) ToggleList(kind doc.ListKind) {
	e.apply(func(src string) string { return doc.ToggleList(src, kind) })
}

// ToggleCode puts the block being written in a code fence, or takes it out.
func (e *Editor) ToggleCode() { e.apply(doc.ToggleCode) }

// Bold, Italic, InlineCode, Strike and Highlight put their marks around the
// selection, or take them off when it already has them.
func (e *Editor) Bold()       { e.mark("**") }
func (e *Editor) Italic()     { e.mark("*") }
func (e *Editor) InlineCode() { e.mark("`") }
func (e *Editor) Strike()     { e.mark("~~") }
func (e *Editor) Highlight()  { e.mark("==") }

// Underline and Link wrap the selection in marks of two halves, as HTML's
// and a link's are.
func (e *Editor) Underline() { e.markPair("<u>", "</u>") }
func (e *Editor) Link()      { e.markPair("[", "](url)") }

func (e *Editor) markPair(open, close string) {
	if !e.editing() {
		return
	}
	nt, a, b := doc.WrapPair(e.draft, e.selInDraftA(), e.selInDraftB(), open, close)
	e.draft = nt
	e.afterItemEdit(b)
	_ = a
}

// ClearStyle takes the inline marks off the block being written.
func (e *Editor) ClearStyle() { e.apply(doc.StripInline) }

// PromoteHeading raises the heading being written a level, DemoteHeading
// lowers it.
func (e *Editor) PromoteHeading() { e.apply(doc.PromoteHeading) }
func (e *Editor) DemoteHeading()  { e.apply(doc.DemoteHeading) }

func (e *Editor) mark(m string) {
	if !e.editing() {
		return
	}
	nt, _, b := doc.Wrap(e.draft, e.selInDraftA(), e.selInDraftB(), m)
	e.draft = nt
	e.afterItemEdit(b)
}

// apply rewrites the block being written.
func (e *Editor) apply(fn func(string) string) {
	if e.editing() {
		e.draft = fn(e.draft)
		if e.itemLine >= 0 {
			e.sync()
			e.resyncItem()
		}
	}
}

// selInDraftA and selInDraftB are the selection the input holds, counted
// from the start of the draft: item writing counts them from its own item.
func (e *Editor) selInDraftA() int {
	if e.itemLine >= 0 {
		return e.selA + e.lineStartRune(e.itemLine)
	}
	return e.selA
}

func (e *Editor) selInDraftB() int {
	if e.itemLine >= 0 {
		return e.selB + e.lineStartRune(e.itemLine)
	}
	return e.selB
}

// afterItemEdit follows a change made to the draft from outside the input:
// the caret it names moves into the item that now holds it, and the
// document takes the draft at once.
func (e *Editor) afterItemEdit(endRune int) {
	if e.itemLine < 0 {
		return
	}
	e.caret = endRune
	e.refocusItem()
	e.sync()
	e.resyncItem()
}

// applyTo rewrites block i, whether it is being written or not.
func (e *Editor) applyTo(i int, fn func(string) string) {
	if i < 0 || e.active == i {
		if e.editing() {
			e.draft = fn(e.draft)
		}
		return
	}
	b, ok := e.doc.Block(i)
	if !ok {
		return
	}
	e.record()
	src := strings.TrimRight(e.doc.Source()[b.Start:b.End], "\n")
	e.doc.Replace(b.Span, fn(src))
	e.refind()
}

// ToggleSidebar shows the panel at the left, or hides it.
func (e *Editor) ToggleSidebar() { e.sh.Toggle() }

// ShowSide opens one of the panels at the left.
func (e *Editor) ShowSide(m SideMode) { e.sh.Show(m) }

// View builds the window: the panel at the left if one shows, the page, and
// the bar under it.
func (e *Editor) View(c *ui.Context) {
	t := e.sh.Look(c.Theme())
	if e.ui == nil || e.style != e.sh.Style {
		md, ut := documentTheme(t, e.sh)
		e.style = e.sh.Style
		// The view leaves the pointer alone: a click on a paragraph enters
		// editing rather than starting a selection of its own.
		e.md = markdown.Options{Theme: md, ShareSelection: true, WrapCode: e.sh.WrapCode}
		e.mdBase, e.mdMark = md, nil
		e.ui = ut
	}
	e.markTheme()
	e.svc = c.Services()
	e.loadImages()
	// The reading view answers the pointer: links open what they name,
	// a picture of its own opens large over the page.
	e.md.OnLink = func(dest string) { e.openLink(c, dest) }
	e.md.Block = func(c *ui.Context, d *parser.Document, i int) bool {
		return e.zoomableBlock(c, d, i)
	}
	ui.Row(c).Fill().AlignItems(ui.Stretch).Background(t.Background).Children(func() {
		if e.sh.Side == SideNone {
			e.page(c, t)
			e.zoomOverlay(c)
			return
		}
		// The panel and the page share the window, with a divider the
		// pointer drags: the split takes SideWidth as the panel's width and
		// writes it back as the divider moves. A double click puts the
		// panel back at the width it opens with. The line is the theme's
		// own until the pointer takes it; the split draws the line it is
		// dragged by in the accent, which is black here rather than blue.
		// It is drawn with the theme around the split, and the panes take
		// the window's own inside.
		line := *t
		line.Accent = ui.RGBA(0, 0, 0, 1)
		if t.Dark {
			line.Accent = t.Text
		}
		ui.Themed(c, &line).Fill().Children(func() {
			sp := ui.Split(c, &e.sh.SideWidth,
				func() {
					ui.Themed(c, t).Fill().Children(func() { e.sidebar(c, t) })
				},
				func() {
					ui.Themed(c, t).Fill().Children(func() { e.page(c, t) })
				},
			).Fill().SplitReset(270)
			if sp.Changed() {
				e.sh.SideMoved()
			}
			e.zoomOverlay(c)
		})
	})
}

// page is the document and the bar under it, with the button that opens the
// panel at the top right.
func (e *Editor) page(c *ui.Context, t *ui.Theme) {
	var page ui.Element
	ui.Column(c).Grow(1).Children(func() {
		ui.Column(c).Grow(1).Children(func() {
			if e.source {
				page = ui.Column(c).Grow(1)
				page.Children(func() { e.sourceArea(c, t) })
			} else {
				page = ui.Scroll(c).TrackScroll(&e.scroll).Grow(1)
				page.Children(func() {
					e.canvas(c)
				})
			}
			// Typora puts the sidebar's button at the top right of the
			// window, over the page.
			e.sideToggle(c, t)
		})
		e.findBar(c, t)
		e.status(c, t)
	})
	if b := page.Bounds(); b.H > 0 {
		e.pageH, e.pageW = b.H, b.W
	}
}

// sideToggle is the small button at the top right of the page that shows
// and hides the panel, as Typora's is.
func (e *Editor) sideToggle(c *ui.Context, t *ui.Theme) {
	row := ui.Row(c).Absolute().Top(8).Right(12).Padding(6, 8).Radius(4)
	if row.Hovered() {
		row.Background(t.Text.Alpha(0.06))
	}
	col := t.TextMuted
	if e.sh.Side != SideNone {
		col = t.Text
	}
	row.OnClick(e.sh.Toggle)
	row.Children(func() {
		ui.Text(c, "☰").FontSize(16).TextColor(col)
	})
}

// canvas is the page: the blocks, the one being written, and the room below
// them, all of it ready to take a click.
func (e *Editor) canvas(c *ui.Context) {
	blocks := e.doc.Blocks()
	at := e.pendingAfter()
	// The page takes the click that no block took: the room between and
	// below them, and the margins either side of the text. A click on a
	// block never reaches it.
	canvasRow := ui.Row(c).MinHeight(e.pageH).
		AlignItems(ui.Stretch).
		OnClick(e.StartParagraph).
		ContextMenu(func(m *ui.Menu) { e.canvasMenu(c, m) })
	canvasRow.Children(func() {
		ui.Box(c).Grow(1).OnClick(e.StartParagraph)
		ui.Column(c).Width(e.measure()).Padding(28, 0).Children(func() {
			if e.pending && at < 0 {
				e.blankRoom(c, 0, e.at, e.at)
				e.newBlock(c)
			}
			for i := range blocks {
				if i > 0 {
					prev, _ := e.doc.Block(i - 1)
					cur, _ := e.doc.Block(i)
					e.blankRoom(c, prev.End, cur.Start, cur.Start)
				}
				e.block(c, i)
				if e.pending && at == i {
					if b, ok := e.doc.Block(i); ok {
						e.blankRoom(c, b.End, e.at, e.at)
					}
					e.newBlock(c)
				}
			}
			if !e.editing() && len(blocks) == 0 {
				// An empty document is just an empty page: the click
				// below puts the caret in its first paragraph.
				ui.Box(c).Height(24)
			}
			// The room under the last block takes a click too, and puts the
			// caret in a new paragraph at the end.
			ui.Box(c).Grow(1).MinHeight(60).OnClick(e.StartParagraph)
		})
		ui.Box(c).Grow(1).OnClick(e.StartParagraph)
	})
}

// blankRoom is the room the blank lines between two places in the source
// ask for: one line of them is the blank line every paragraph has after it,
// the ones beyond it are the empty lines the writer made with Return. A
// click on them puts the caret at at, the end of them, so that Backspace
// takes them away again.
func (e *Editor) blankRoom(c *ui.Context, from, to, at int) {
	src := e.doc.Source()
	if from < 0 || to <= from || to > len(src) {
		return
	}
	n := strings.Count(src[from:to], "\n") - 2
	if n <= 0 {
		return
	}
	if n > 20 {
		n = 20
	}
	b := ui.Box(c).Height(float32(n) * e.sh.Size * e.sh.LineHeight).AlignSelf(ui.Stretch)
	if at >= 0 {
		b.OnClick(func() { e.startAt(at) })
	}
}

// backspaceBlank takes one of the blank lines under the paragraph being
// written, so that the lines Return made can be taken away again: the caret
// stays where it is and the line above it goes.
func (e *Editor) backspaceBlank() bool {
	if !e.pending || strings.TrimSpace(e.draft) != "" {
		return false
	}
	src := e.doc.Source()
	at := min(max(e.at, 0), len(src))
	from := 0
	above := -1
	if i := e.pendingAfter(); i >= 0 {
		above = i
		if b, ok := e.doc.Block(i); ok {
			from = b.End
		}
	}
	if at-from > 2 && at > 0 && src[at-1] == '\n' {
		// A blank line above the caret goes, and the caret takes its place.
		e.record()
		e.doc.SetSource(src[:at-1] + src[at:])
		e.at = at - 1
		e.refind()
		return true
	}
	// The separation between the neighbours is all that is left above the
	// caret, or the caret sits right at it: the empty line joins the block
	// above, and the caret waits at the end of it, as Typora's does. In a
	// list the writing goes on in its last item, the rest rendered as it
	// reads.
	if above >= 0 {
		b, ok := e.doc.Block(above)
		if !ok {
			return false
		}
		e.pending = false
		e.active = above
		e.draft = strings.TrimRight(e.doc.Source()[b.Start:b.End], "\n")
		e.caret = doc.RuneOffset(e.draft, len(e.draft))
		e.refocusItem()
		e.refind()
		return true
	}
	if at > 0 && src[at-1] == '\n' {
		// The document starts here: the leading room goes, line by line.
		e.record()
		e.doc.SetSource(src[:at-1] + src[at:])
		e.at = at - 1
		e.refind()
		return true
	}
	return false
}

// measure is how wide the text column is: as wide as the look asks for,
// unless the window or the panel leaves less room than that.
func (e *Editor) measure() float32 {
	w := e.sh.Measure
	if e.pageW > 0 {
		if max := e.pageW - 96; w > max {
			w = max
		}
	}
	if w < 240 {
		w = 240
	}
	return w
}

// pendingAfter is the block the new block goes after, -1 before the first.
// A block the parser holds as nothing takes no room of its own and is not
// the one the writing goes after.
func (e *Editor) pendingAfter() int {
	if !e.pending {
		return -1
	}
	idx := -1
	for i, b := range e.doc.Blocks() {
		if b.End > e.at {
			break
		}
		if b.End > b.Start {
			idx = i
		}
	}
	return idx
}

func (e *Editor) block(c *ui.Context, i int) {
	b, ok := e.doc.Block(i)
	if !ok {
		return
	}
	if i == e.active {
		// The block being written shows at the size its syntax says: a
		// heading grows as the "# " is typed, and the markers fade. It sits
		// on the page as the text it becomes does, with no frame of its own.
		// A list writes by the item: the item the caret is in is the input,
		// and the others show as they will read.
		ut := *e.ui
		ut.FontSize = doc.BlockSize(e.draft) * e.sh.Size / 16
		col := ui.Column(c.Key(i)).Padding(e.above(i), 0).
			ContextMenu(func(m *ui.Menu) { e.blockMenu(c, m, i) })
		col.Children(func() {
			ui.Themed(c, &ut).Children(func() {
				if e.itemsMode() {
					e.editItems(c, c.Theme())
				} else {
					e.editArea(c)
				}
			})
		})
		e.reveal(col, i)
		return
	}
	// The view takes no selection of its own (ShareSelection), so the click
	// and the menu of the block around it get the pointer.
	src := e.doc.Source()[b.Start:b.End]
	col := ui.Column(c.Key(i)).Padding(e.above(i), 0).
		OnClick(func() { e.Activate(i) }).
		ContextMenu(func(m *ui.Menu) { e.blockMenu(c, m, i) })
	// A block a search lands in shows its words marked: the matches are
	// wrapped in <mark> in the source the view draws, in a paragraph, a
	// heading, an item, a quote or a cell alike. A block with no text to
	// wrap — a code block, a table's rules — is tinted as a whole instead.
	// A block left as it is shows its pictures, whose addresses are made as
	// Markdown takes them.
	if marked, ok := e.markedSource(i); ok {
		src = marked
	} else {
		// Code is shown as it is written: no address in it is a picture.
		if b.Kind != parser.CodeBlock && b.Kind != parser.HTMLBlock && b.Kind != parser.MathBlock {
			src = readableSrc(src)
		}
		if any, _ := e.blockHit(i); any {
			col.Background(e.highlightColor().Alpha(0.12))
		}
	}
	col.Children(func() {
		markdown.View(c, src, &e.md)
	})
	e.reveal(col, i)
	if b := col.Bounds(); b.H > 0 {
		e.tops[i] = b.Y + e.scroll.Y
	}
}

// reveal brings a block into view when an outline click asked for it: the
// scroll has to be asked in the frame that lays the block out.
func (e *Editor) reveal(el ui.Element, i int) {
	if e.scrollTo != i {
		return
	}
	el.ScrollIntoView(ui.ScrollIntoViewOptions{Block: ui.ScrollStart})
	e.scrollTo = -1
}

// newBlock shows the block being written that the document does not have
// yet.
func (e *Editor) newBlock(c *ui.Context) {
	ut := *e.ui
	ut.FontSize = doc.BlockSize(e.draft) * e.sh.Size / 16
	col := ui.Column(c.Key("new")).Padding(e.above(len(e.doc.Blocks())), 0).
		ContextMenu(func(m *ui.Menu) { e.blockMenu(c, m, -1) })
	col.Children(func() {
		ui.Themed(c, &ut).Children(func() { e.editArea(c) })
	})
	e.reveal(col, e.doc.Len())
}

// editArea is the source of the block being written, at the size and weight
// its markers say. It draws no frame: the text is on the page as the render
// of it is.
func (e *Editor) editArea(c *ui.Context) {
	t := c.Theme()
	ta := ui.TextAreaBase(c.Key("edit"), &e.draft).Lines(1, 40).Padding(0, 0).AutoFocus()
	ta.Label(locales.EditArea())
	ta.Border(0, ui.Color{})
	ta.Background(ui.Color{})
	ta.FocusRing(false)
	// Typora's menu, with the input's own cut, copy and paste in front.
	ta.ContextMenu(func(m *ui.Menu) { e.editMenu(c, m, e.active) })
	// A paste of a picture puts the picture in, as Typora does; text goes on
	// to the input, which pastes it as it always did.
	ta.OnPaste(func(text string) bool {
		if t := strings.TrimSpace(text); t != "" && !isImage(t) {
			return false
		}
		return e.PasteImage()
	})
	// The caret decides which line's markers show: they hide on the other
	// lines, so the rest of the block reads as it will once done. The
	// selection is the input's own, as of the frame before, or the keys
	// this frame routed.
	_, head := ta.TextAnchorCaret()
	e.selA, e.selB = ta.TextSelection()
	caretLine := doc.LineOf(e.draft, doc.ByteOffset(e.draft, head))
	// The hidden markers take the page's own color: painted over, they are
	// gone, and the range stays a color the input accepts.
	ranges := e.ranges(e.draft, caretLine, t.TextMuted, t.Text, t.Background)
	ranges = append(ranges, e.findRanges(e.draft, e.blockStart(), e.highlightColor(), e.highlightColor())...)
	ta.TextRanges(ranges...)
	// A change that moved the caret for the writer, as Return continuing a
	// list does, puts it where the writing goes on.
	if e.caret >= 0 {
		// As the item's input does: the mirror takes the caret at once.
		ta.SetTextSelection(e.caret, e.caret)
		e.selA, e.selB = e.caret, e.caret
		e.caret = -1
	}
	// Every character reaches the document, so that the file and its edited
	// dot follow what is on the screen.
	ta.OnChange(e.sync)
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
			// On an empty paragraph, Backspace takes the blank line Return
			// made, rather than leaving it there for good.
			if e.backspaceBlank() {
				return true
			}
			return false
		case ui.KeyEnter:
			// Return ends the paragraph where the caret is, as Typora does,
			// rather than adding a line only to jump to the document's end.
			if k.Mods&(ui.Shift|ui.Alt|ui.Ctrl|ui.Cmd) != 0 {
				return false
			}
			e.splitAtCaret()
			return true
		}
		return false
	})
}

// status is the bar under the page: how much is written, and the buttons
// that show the panels at the left.
func (e *Editor) status(c *ui.Context, t *ui.Theme) {
	if !e.sh.ShowStatusBar {
		return
	}
	chars := len([]rune(e.doc.Source()))
	ui.Row(c).Padding(6, 18).AlignItems(ui.Center).Gap(14).Children(func() {
		ui.Text(c, locales.StatusChars(chars)).FontSize(13).TextColor(t.TextMuted)
		if m := readingMinutes(chars, e.sh.ReadingSpeed); m != "" {
			ui.Text(c, m).FontSize(13).TextColor(t.TextMuted)
		}
	})
}

// above is the room a block asks for, as a heading opens a section.
func (e *Editor) above(i int) float32 {
	if i == 0 {
		return 0
	}
	b, ok := e.doc.Block(i)
	if !ok {
		return 0
	}
	prev, _ := e.doc.Block(i - 1)
	switch {
	case b.Kind == parser.Heading && b.Level == 1:
		return 36
	case b.Kind == parser.Heading && b.Level == 2:
		return 30
	case b.Kind == parser.Heading:
		return 24
	case prev.Kind == parser.Heading:
		return 4
	}
	return 0
}

// ranges turns the syntax runs of a block into the styles of its text input,
// in runes as the input counts them: the text its markers wrap at its
// weight, the markers faint. The markers of the line the caret is on show,
// and those of the other lines hide, so the rest of the block reads as it
// will once it is done; a line's own marker — the dash of an item, the
// angle of a quote — stays faint on every line, for hidden, the shape of
// the block goes with it.
func (e *Editor) ranges(s string, caretLine int, muted, plain, hidden ui.Color) []ui.TextRange {
	runs := doc.SyntaxRuns(s)
	out := make([]ui.TextRange, 0, len(runs))
	for _, r := range runs {
		c, weight := plain, r.Weight
		switch r.Kind {
		case doc.RunPrefix:
			c = muted
		case doc.RunMarker:
			c = hidden
			if doc.LineOf(s, r.Start) == caretLine {
				c = muted
			}
		}
		out = append(out, ui.TextRange{
			Start:  doc.RuneOffset(s, r.Start),
			End:    doc.RuneOffset(s, r.End),
			Color:  c,
			Weight: weight,
		})
	}
	return out
}

// readingMinutes is how long the text takes to read at the speed the reader
// set, as the status bar shows it: empty for text too short to say.
func readingMinutes(chars, speed int) string {
	if speed <= 0 || chars < 100 {
		return ""
	}
	m := (chars + speed - 1) / speed
	if m < 1 {
		m = 1
	}
	if m < 60 {
		return locales.ReadingMinutes(m)
	}
	return locales.ReadingHours(m/60, m%60)
}
