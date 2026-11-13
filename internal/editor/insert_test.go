package editor

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"paperly/internal/doc"
)

const insertDoc = `第一段，光标所在的段落。

第二段，后面还有内容。

第三段，文章在这里结束。
`

// Insert from the page's own menu, opened in the margin beside a paragraph
// in the middle, lands under that paragraph rather than at the end of the
// text.
func TestInsertInTheMarginLandsThere(t *testing.T) {
	ed := New(doc.New(insertDoc), NewShared())
	tt := ui.NewTester(ed.View, 860, 1000)
	tt.Frame()
	tt.Frame()

	// The left margin, beside the second paragraph.
	tt.RightClickAt(20, 65)
	tt.Frame()
	if err := tt.ChooseMenuItem("插入", "插入表格"); err != nil {
		t.Fatalf("choose: %v", err)
	}
	tt.Frame()

	got := ed.Doc().Source()
	table := strings.Index(got, "| 列 1 |")
	if table < strings.Index(got, "第二段") || table > strings.Index(got, "第三段") {
		t.Errorf("the table went somewhere else:\n%s", got)
	}
}

// A block put in below the text still goes to the end.
func TestInsertBelowTheTextStaysAtTheEnd(t *testing.T) {
	ed := New(doc.New(insertDoc), NewShared())
	tt := ui.NewTester(ed.View, 860, 1000)
	tt.Frame()
	tt.Frame()

	tt.RightClickAt(20, 500)
	tt.Frame()
	if err := tt.ChooseMenuItem("插入", "水平分割线"); err != nil {
		t.Fatalf("choose: %v", err)
	}
	tt.Frame()
	got := ed.Doc().Source()
	if !strings.HasSuffix(strings.TrimSpace(got), "---") {
		t.Errorf("the rule did not land at the end:\n%s", got)
	}
}

// Above the first paragraph, a block put in goes to the top.
func TestInsertAboveTheTextGoesToTheTop(t *testing.T) {
	ed := New(doc.New(insertDoc), NewShared())
	tt := ui.NewTester(ed.View, 860, 1000)
	tt.Frame()
	tt.Frame()

	tt.RightClickAt(20, 4)
	tt.Frame()
	if err := tt.ChooseMenuItem("插入", "代码块"); err != nil {
		t.Fatalf("choose: %v", err)
	}
	tt.Frame()
	got := ed.Doc().Source()
	if !strings.HasPrefix(strings.TrimSpace(got), "```") {
		t.Errorf("the code block did not land at the top:\n%s", got)
	}
}

// A click in the margin beside a paragraph puts the caret under it.
func TestClickInTheMarginWritesThere(t *testing.T) {
	ed := New(doc.New(insertDoc), NewShared())
	tt := ui.NewTester(ed.View, 860, 1000)
	tt.Frame()
	tt.Frame()

	tt.ClickAt(20, 65)
	tt.Frame()
	if !ed.Pending() {
		t.Fatalf("no block is being written")
	}
	if b, _ := ed.doc.Block(1); ed.at != b.End {
		t.Errorf("the new block goes at %d, want %d", ed.at, b.End)
	}
	tt.Type("插在中间")
	tt.Frame()
	got := ed.Doc().Source()
	if strings.Index(got, "插在中间") > strings.Index(got, "第三段") {
		t.Errorf("the typing went past the last paragraph:\n%s", got)
	}
}

// A picture put in from a paragraph's menu stays under that paragraph.
func TestImageFromBlockMenuStaysThere(t *testing.T) {
	ed := New(doc.New(insertDoc), NewShared())
	tt := ui.NewTester(ed.View, 860, 1000)
	tt.Frame()
	tt.Frame()

	// The menu names the block; the picture's own dialog is native, so what
	// follows the choice is what is tested here.
	tt.RightClickAt(300, 65)
	tt.Frame()
	if ed.lastBlock != 1 {
		t.Fatalf("the menu marked block %d, want 1", ed.lastBlock)
	}
	ed.startForInsert(1, -1)
	ed.draft = "![red](assets/red.png)"
	ed.Commit()
	tt.Frame()

	got := ed.Doc().Source()
	if strings.Index(got, "![red]") > strings.Index(got, "第三段") {
		t.Errorf("the picture went to the end:\n%s", got)
	}
}

// A block put in from the menu of the block being written goes below it,
// once what was written is committed.
func TestInsertAfterTheBlockBeingWritten(t *testing.T) {
	ed := New(doc.New(insertDoc), NewShared())
	tt := ui.NewTester(ed.View, 860, 1000)
	tt.Frame()
	tt.Frame()

	ed.Activate(1)
	tt.Frame()
	tt.Type("，又写了一些")
	tt.Frame()

	tt.RightClickAt(300, 65)
	tt.Frame()
	if err := tt.ChooseMenuItem("插入", "插入表格"); err != nil {
		t.Fatalf("choose: %v", err)
	}
	tt.Frame()

	got := ed.Doc().Source()
	if !strings.Contains(got, "第二段，后面还有内容。，又写了一些") {
		t.Errorf("what was written was lost:\n%s", got)
	}
	table := strings.Index(got, "| 列 1 |")
	if table < strings.Index(got, "又写了一些") || table > strings.Index(got, "第三段") {
		t.Errorf("the table went somewhere else:\n%s", got)
	}
}

// With no caret, a picture put in from the app's menu goes below the block
// the pointer last stood on, not to the end of the text.
func TestImageWithNoCaretGoesBelowTheBlockClicked(t *testing.T) {
	ed := New(doc.New(insertDoc), NewShared())
	tt := ui.NewTester(ed.View, 860, 1000)
	tt.Frame()
	tt.Frame()

	tt.ClickAt(300, 65)
	tt.Frame()
	tt.Key(0, ui.KeyEscape)
	tt.Frame()

	ed.startForInsert(-1, -1)
	ed.draft = "![red](assets/red.png)"
	ed.Commit()
	tt.Frame()

	got := ed.Doc().Source()
	pic := strings.Index(got, "![red]")
	if pic < strings.Index(got, "第二段") || pic > strings.Index(got, "第三段") {
		t.Errorf("the picture went somewhere else:\n%s", got)
	}
}

// The menu of the paragraph being written puts a block where that paragraph
// waits, not at the end of the document.
func TestMenuOfTheNewParagraphInsertsThere(t *testing.T) {
	ed := New(doc.New(insertDoc), NewShared())
	tt := ui.NewTester(ed.View, 860, 1000)
	tt.Frame()
	tt.Frame()

	tt.ClickAt(20, 65)
	tt.Frame()
	if !ed.Pending() {
		t.Fatalf("no paragraph is being written")
	}
	at := ed.at
	if got := ed.menuAt(-1); got != at {
		t.Fatalf("the menu puts blocks at %d, want %d", got, at)
	}

	ed.insertAt(ed.menuAt(-1), "| 列 1 |\n| --- | --- |\n|  |  |")
	got := ed.Doc().Source()
	table := strings.Index(got, "| 列 1 |")
	if table < strings.Index(got, "第二段") || table > strings.Index(got, "第三段") {
		t.Errorf("the table went somewhere else:\n%s", got)
	}
}

// A table put in from the menu of the empty paragraph a click opened goes
// into that paragraph's room, not to the end of the text.
func TestInsertInTheParagraphBeingWritten(t *testing.T) {
	ed := New(doc.New(insertDoc), NewShared())
	tt := ui.NewTester(ed.View, 860, 1000)
	tt.Frame()
	tt.Frame()

	// A click in the margin opens an empty paragraph beside the second one.
	tt.ClickAt(20, 65)
	tt.Frame()
	if !ed.Pending() {
		t.Fatalf("no paragraph is being written")
	}
	at := ed.at

	// Its own menu, over the empty paragraph.
	tt.RightClickAt(300, 85)
	tt.Frame()
	if err := tt.ChooseMenuItem("插入", "插入表格"); err != nil {
		t.Fatalf("choose: %v", err)
	}
	tt.Frame()

	got := ed.Doc().Source()
	table := strings.Index(got, "| 列 1 |")
	if table < strings.Index(got, "第二段") || table > strings.Index(got, "第三段") {
		t.Errorf("the table went somewhere else (at %d, the paragraph was %d):\n%s", table, at, got)
	}
}
