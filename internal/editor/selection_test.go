package editor

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"paperly/internal/doc"
)

const selDoc = `第一段，开头在这里，结尾在那里。

第二段，中间隔了一段。

第三段，选区的另一端在这里。
`

// dragSelect drags from the start of the first paragraph into the last one,
// with the selection modifier held.
func dragSelect(t *testing.T, ed *Editor, tt *ui.Tester) {
	t.Helper()
	first, ok := tt.Find("第一段，开头在这里，结尾在那里。")
	if !ok {
		t.Fatal("no first paragraph")
	}
	last, ok := tt.Find("第三段，选区的另一端在这里。")
	if !ok {
		t.Fatal("no last paragraph")
	}
	tt.Press(first.X+first.W/4, first.Y+first.H/2)
	tt.Move(last.X+last.W/4, last.Y+last.H/2)
	tt.Release(last.X+last.W/4, last.Y+last.H/2)
	tt.Frame()
}

// While the selection modifier is held, a drag runs from a paragraph into
// another and Copy takes their text.
func TestSelectAcrossBlocks(t *testing.T) {
	ed := New(doc.New(selDoc), NewShared())
	ed.testMod = ui.Alt
	tt := ui.NewTester(ed.View, 860, 1000)
	tt.Frame()
	tt.Frame()

	dragSelect(t, ed, tt)
	if ed.Active() >= 0 || ed.Pending() {
		t.Errorf("the drag opened block %d for writing", ed.Active())
	}

	tt.Command("copy")
	got := tt.Clipboard()
	want := "在那里。\n\n第二段，中间隔了一段。\n\n第三段，选区的另一端在这"
	if got != want {
		t.Errorf("copied %q, want %q", got, want)
	}
}

// The run is character for character: it starts and ends where the pointer
// did, not at the ends of the paragraphs it crosses.
func TestSelectStartsAndEndsAtThePointer(t *testing.T) {
	ed := New(doc.New(selDoc), NewShared())
	ed.testMod = ui.Alt
	tt := ui.NewTester(ed.View, 860, 1000)
	tt.Frame()
	tt.Frame()

	dragSelect(t, ed, tt)
	tt.Command("copy")
	got := tt.Clipboard()
	head := got
	if i := strings.Index(got, "\n"); i >= 0 {
		head = got[:i]
	}
	if !strings.Contains("第一段，开头在这里，结尾在那里。", head) || strings.HasPrefix(head, "第一段") {
		t.Errorf("the run does not start mid-paragraph: %q", head)
	}
	if strings.HasSuffix(got, "。") {
		t.Errorf("the run ends at the paragraph's end, not at the pointer: %q", got)
	}
}

// The selection outlasts the modifier, so that Copy still has it once the
// key is let go.
func TestSelectionOutlastsTheModifier(t *testing.T) {
	ed := New(doc.New(selDoc), NewShared())
	ed.testMod = ui.Alt
	tt := ui.NewTester(ed.View, 860, 1000)
	tt.Frame()
	tt.Frame()

	dragSelect(t, ed, tt)
	ed.testMod = 0
	tt.Frame()
	tt.Frame()

	tt.Command("copy")
	if got := tt.Clipboard(); !strings.Contains(got, "第二段") {
		t.Errorf("the selection did not outlast the modifier: %q", got)
	}
}

// A click drops the selection without opening a block; the next click opens
// the one it lands on.
func TestClickDropsTheSelectionThenEdits(t *testing.T) {
	ed := New(doc.New(selDoc), NewShared())
	ed.testMod = ui.Alt
	tt := ui.NewTester(ed.View, 860, 1000)
	tt.Frame()
	tt.Frame()

	dragSelect(t, ed, tt)
	ed.testMod = 0
	tt.Frame()

	tt.ClickAt(300, 65)
	tt.Frame()
	if ed.Active() >= 0 || ed.Pending() {
		t.Errorf("the click that dropped the selection also opened block %d", ed.Active())
	}

	tt.ClickAt(300, 65)
	tt.Frame()
	if ed.Active() != 1 {
		t.Errorf("the second click opened block %d, want 1", ed.Active())
	}
}

// Without the modifier a click still opens the block it lands on.
func TestClickStillEdits(t *testing.T) {
	ed := New(doc.New(selDoc), NewShared())
	tt := ui.NewTester(ed.View, 860, 1000)
	tt.Frame()
	tt.Frame()

	tt.ClickAt(300, 65)
	tt.Frame()
	if ed.Active() != 1 {
		t.Errorf("clicking the second paragraph opened block %d, want 1", ed.Active())
	}
}

// A part cut at either end holds only the runes between its offsets.
func TestPartText(t *testing.T) {
	if got := partText(ui.SelectedText{Text: "选区的另一端", Start: 3, End: 5}); got != "另一" {
		t.Errorf("partText = %q, want 另一", got)
	}
	if got := partText(ui.SelectedText{Text: "短", Start: 0, End: 9}); got != "短" {
		t.Errorf("partText past the end = %q, want 短", got)
	}
}

const mdDoc = "第一段有 **粗体** 与 `代码` 和 [链接](https://a.b)。\n\n第二段，中间一句话。\n\n第三段结尾。\n"

// The menu of a selection offers its source as Markdown, with the marks the
// page keeps out of sight.
func TestCopySelectionAsMarkdown(t *testing.T) {
	ed := New(doc.New(mdDoc), NewShared())
	ed.testMod = ui.Alt
	tt := ui.NewTester(ed.View, 860, 700)
	tt.Frame()
	tt.Frame()

	first, ok := tt.Find("第一段有 粗体 与 代码 和 链接。")
	if !ok {
		t.Fatal("no first paragraph")
	}
	last, ok := tt.Find("第三段结尾。")
	if !ok {
		t.Fatal("no last paragraph")
	}
	tt.Press(first.X+100, first.Y+first.H/2)
	tt.Move(last.X+last.W/2, last.Y+last.H/2)
	tt.Release(last.X+last.W/2, last.Y+last.H/2)
	tt.Frame()

	// Right-click inside the selection, as a reader would on the text.
	tt.RightClickAt(last.X+50, last.Y+last.H/2)
	tt.Frame()
	if err := tt.ChooseMenuItem("复制为 Markdown"); err != nil {
		t.Fatalf("%v", err)
	}
	tt.Frame()
	md := tt.Clipboard()
	if !strings.Contains(md, "`代码`") || !strings.Contains(md, "[链接](https://a.b)") {
		t.Errorf("the marks are lost: %q", md)
	}
	if !strings.Contains(md, "第二段，中间一句话。") {
		t.Errorf("the middle paragraph is missing: %q", md)
	}

	// The same run as the page reads it, for the difference to show.
	tt.Command("copy")
	if text := tt.Clipboard(); strings.Contains(text, "`") {
		t.Errorf("the plain copy carries the source's marks: %q", text)
	}
}

// A whole document selected with Cmd+A copies whole, as the page reads it.
func TestSelectAllCopiesThePage(t *testing.T) {
	ed := New(doc.New(mdDoc), NewShared())
	ed.testMod = ui.Alt
	tt := ui.NewTester(ed.View, 860, 700)
	tt.Frame()
	tt.Frame()

	// A click puts a caret in the text, which Cmd+A then widens to the page.
	tt.ClickAt(300, 40)
	tt.Frame()
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Frame()
	tt.Command("copy")
	got := tt.Clipboard()
	for _, want := range []string{"第一段有 粗体 与 代码 和 链接。", "第二段，中间一句话。", "第三段结尾。"} {
		if !strings.Contains(got, want) {
			t.Errorf("the select-all copy is missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "`") || strings.Contains(got, "[链接](") {
		t.Errorf("the select-all copy carries the source's marks:\n%s", got)
	}
}
