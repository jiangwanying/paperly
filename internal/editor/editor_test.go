package editor

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown/parser"

	"paperly/internal/doc"
)

const sample = `# 让光标所在的那一块，变回 Markdown

Typora 的核心不是「所见即所得」，是**块级编辑**：光标进到哪里，哪里就露出源码。

## 这份骨架要验证的事

- 点击任意段落，它当场变成可编辑的源码
- 按 Esc 提交，重新解析，块重新渲染
`

// A block turns into its source, takes typing, and commits on Escape.
func TestEditInPlace(t *testing.T) {
	ed := New(doc.New(sample), NewShared())
	tt := ui.NewTester(ed.View, 860, 1000)
	tt.Frame()

	// Block 0 is the title, block 1 the paragraph under it.
	ed.Activate(1)
	tt.Frame()
	if ed.Active() != 1 {
		t.Fatalf("active block %d, want 1", ed.Active())
	}

	tt.Type("【插入】")
	tt.Key(0, ui.KeyEscape)
	tt.Frame()

	if ed.Active() != -1 {
		t.Errorf("still editing after Escape")
	}
	if !strings.Contains(ed.Doc().Source(), "【插入】") {
		t.Errorf("the edit is not in the source:\n%s", ed.Doc().Source())
	}
	if ed.Doc().Len() == 0 {
		t.Errorf("the document has no blocks after the edit")
	}
}

// Clicking a rendered block turns it into its source.
func TestClickToEdit(t *testing.T) {
	ed := New(doc.New(sample), NewShared())
	tt := ui.NewTester(ed.View, 860, 1000)
	tt.Frame()
	tt.Frame()

	// The paragraph under the title, at the top of the text column.
	tt.ClickAt(300, 90)
	tt.Frame()

	if ed.Active() != 1 {
		t.Fatalf("clicking the paragraph edited block %d, want 1", ed.Active())
	}
	if !strings.Contains(ed.Draft(), "Typora 的核心") {
		t.Errorf("the block's source is not in the draft: %q", ed.Draft())
	}
}

// TestShot renders the view offscreen. Set MD_UPDATE_SHOTS=1 to write the
// screenshots, which are testdata rather than test output.
func TestShot(t *testing.T) {
	if os.Getenv("MD_UPDATE_SHOTS") == "" {
		t.Skip("set MD_UPDATE_SHOTS=1 to write screenshots")
	}
	ed := New(doc.New(sample), NewShared())
	tt := ui.NewTester(ed.View, 860, 1000)
	tt.Frame()
	tt.Frame()
	shot(t, tt, "shot.png")

	ed.Activate(1)
	tt.Frame()
	shot(t, tt, "shot-edit.png")

	// The outline at the left.
	other := New(doc.New(sample), NewShared())
	other.sh.Side = SideOutline
	tm := ui.NewTester(other.View, 860, 1000)
	tm.Frame()
	tm.Frame()
	shot(t, tm, "shot-outline.png")

	// An empty document, ready for its first line.
	empty := New(doc.New(""), NewShared())
	te := ui.NewTester(empty.View, 860, 700)
	te.Frame()
	te.Frame()
	shot(t, te, "shot-empty.png")

	// The find bar, its matches painted in the text around them.
	find := New(doc.New(findSample), NewShared())
	tf := ui.NewTester(find.View, 860, 900)
	tf.Frame()
	tf.Frame()
	find.OpenFind(true)
	find.findText = "用户"
	find.refreshHits()
	tf.Frame()
	shot(t, tf, "shot-find.png")
}

// findSample has the blocks a search paints: a heading, a paragraph, a
// list, a quote and a table.
const findSample = `# 用户手册

这是用户的段落，讲用户怎么用。

- 用户的一项
- 还有**加粗的用户**

> 引用里的用户

| 列 | 值 |
| --- | --- |
| 用户 | 其他 |
`

func shot(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, tt.Image()); err != nil {
		t.Fatal(err)
	}
	dir := "../../testdata/shots"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dir+"/"+name, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A click in the room below the text starts a paragraph there.
func TestClickBelowText(t *testing.T) {
	ed := New(doc.New(sample), NewShared())
	tt := ui.NewTester(ed.View, 860, 900)
	tt.Frame()
	tt.Frame()

	tt.ClickAt(300, 600)
	tt.Frame()
	if !ed.Pending() {
		t.Fatalf("a click below the text did not start a paragraph")
	}
	tt.Type("结尾加一段")
	tt.Key(0, ui.KeyEscape)
	tt.Frame()

	src := ed.Doc().Source()
	if !strings.HasSuffix(strings.TrimRight(src, "\n"), "结尾加一段") {
		t.Errorf("the paragraph did not join the end:\n%s", src)
	}
	if ed.Doc().Len() != 5 {
		t.Errorf("%d blocks, want 5", ed.Doc().Len())
	}
}

// An empty document takes a click too, and the first paragraph it makes.
func TestEmptyDocument(t *testing.T) {
	ed := New(doc.New(""), NewShared())
	tt := ui.NewTester(ed.View, 860, 700)
	tt.Frame()
	tt.Frame()
	if ed.Doc().Len() != 0 {
		t.Fatalf("%d blocks in an empty document", ed.Doc().Len())
	}

	tt.ClickAt(300, 200)
	tt.Frame()
	if !ed.Pending() {
		t.Fatalf("a click in an empty document did nothing")
	}
	tt.Type("第一句话")
	tt.Key(0, ui.KeyEscape)
	tt.Frame()

	if got := ed.Doc().Source(); got != "第一句话" {
		t.Errorf("source %q", got)
	}
	if ed.Doc().Len() != 1 {
		t.Errorf("%d blocks, want 1", ed.Doc().Len())
	}
}

// A click that writes nothing leaves the document as it was.
func TestEmptyParagraphLeavesNoTrace(t *testing.T) {
	ed := New(doc.New(sample), NewShared())
	tt := ui.NewTester(ed.View, 860, 900)
	tt.Frame()
	tt.Frame()

	tt.ClickAt(300, 600)
	tt.Frame()
	tt.Key(0, ui.KeyEscape)
	tt.Frame()

	if ed.Doc().Len() != 4 || ed.Doc().Source() != sample {
		t.Errorf("the document changed: %d blocks", ed.Doc().Len())
	}
}

// Right-clicking a paragraph opens its menu.
func TestBlockMenu(t *testing.T) {
	ed := New(doc.New(sample), NewShared())
	tt := ui.NewTester(ed.View, 860, 900)
	tt.Frame()
	tt.Frame()

	tt.RightClickAt(300, 90)
	tt.Frame()
	menu := tt.Menu()
	// The order and words of Typora's menu, with the delete of the block
	// at the first level.
	want := []string{"剪切", "拷贝", "粘贴", "复制／粘贴为", "-", "段落", "格式", "插入", "-", "在侧边栏中显示", "-", "自动填充", "-", "删除块"}
	if len(menu) != len(want) {
		t.Fatalf("menu %v, want %v", menu, want)
	}
	for i := range want {
		if menu[i] != want[i] {
			t.Fatalf("menu %v, want %v", menu, want)
		}
	}
}

// The paragraph commands rewrite the block they are given.
func TestParagraphCommands(t *testing.T) {
	ed := New(doc.New(sample), NewShared())
	ed.Activate(1)
	ed.ToggleQuote()
	ed.Commit()
	if got := ed.Doc().Source(); !strings.Contains(got, "> Typora 的核心") {
		t.Errorf("the quote did not land:\n%s", got)
	}

	ed.Activate(2)
	ed.SetHeading(0)
	ed.Commit()
	if got := ed.Doc().Source(); strings.Contains(got, "## 这份骨架") {
		t.Errorf("the heading stayed:\n%s", got)
	}
}

// What is typed reaches the document before Escape: a save while writing
// keeps it.
func TestTypingReachesTheDocument(t *testing.T) {
	ed := New(doc.New(sample), NewShared())
	tt := ui.NewTester(ed.View, 860, 900)
	tt.Frame()
	tt.Frame()

	ed.StartParagraph()
	tt.Frame()
	tt.Type("没按 Esc 也要在文档里")
	tt.Frame()

	if !strings.Contains(ed.Doc().Source(), "没按 Esc 也要在文档里") {
		t.Fatalf("the typing is not in the document:\n%s", ed.Doc().Source())
	}
	if !ed.Editing() {
		t.Errorf("the editor left the block while it was being written")
	}
}

// Two Returns at the end open a new paragraph, and writing goes on in it.
func TestReturnOpensAParagraph(t *testing.T) {
	ed := New(doc.New("第一段"), NewShared())
	tt := ui.NewTester(ed.View, 860, 700)
	tt.Frame()

	ed.StartParagraph()
	tt.Frame()
	tt.Type("甲")
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	tt.Type("乙")
	tt.Frame()

	src := ed.Doc().Source()
	if !strings.Contains(src, "第一段") || !strings.Contains(src, "甲") || !strings.Contains(src, "乙") {
		t.Fatalf("the paragraphs are lost:\n%s", src)
	}
	if got := ed.Doc().Len(); got != 3 {
		t.Errorf("%d blocks, want 3:\n%s", got, src)
	}
}

// Return on an empty paragraph leaves a blank line under it, and another for
// each Return after that, as Typora does.
func TestReturnLeavesBlankLines(t *testing.T) {
	ed := New(doc.New("第一段"), NewShared())
	tt := ui.NewTester(ed.View, 860, 700)
	tt.Frame()

	ed.StartParagraph()
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if got := strings.Count(ed.Doc().Source(), "\n"); got != 4 {
		t.Errorf("%d newlines after two Returns, want 4: %q", got, ed.Doc().Source())
	}

	tt.Type("乙")
	tt.Frame()
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if src := ed.Doc().Source(); !strings.Contains(src, "第一段\n\n\n\n乙") {
		t.Errorf("the blank lines did not stay: %q", src)
	}
}

// Clicking a heading in the outline brings it into view.
func TestOutlineScrolls(t *testing.T) {
	ed := New(doc.New(strings.Repeat("段落\n\n", 60)+"## 目标\n\n结尾"), NewShared())
	sh := ed.sh
	sh.Side = SideOutline
	tt := ui.NewTester(ed.View, 860, 600)
	tt.Frame()
	tt.Frame()
	if ed.scroll.Y != 0 {
		t.Fatalf("the page starts scrolled: %v", ed.scroll.Y)
	}

	ed.scrollTo = 60 // the heading, as a click on the outline does
	for i := 0; i < 20; i++ {
		tt.Frame()
	}
	if ed.scroll.Y <= 0 {
		t.Errorf("the page did not scroll to the heading: y=%v", ed.scroll.Y)
	}
}

// A heading shows in the outline without its markers.
func TestHeadingText(t *testing.T) {
	cases := map[string]string{
		"## 这份骨架\n\n正文": "这份骨架",
		"# 一级标题":        "一级标题",
		"###### 六级":     "六级",
	}
	for in, want := range cases {
		if got := headingText(in); got != want {
			t.Errorf("headingText(%q) = %q, want %q", in, got, want)
		}
	}
}

// The panels show at the left, and the buttons under the page toggle them.
func TestSideToggle(t *testing.T) {
	ed := New(doc.New(sample), NewShared())
	tt := ui.NewTester(ed.View, 860, 900)
	tt.Frame()

	ed.sh.Toggle()
	tt.Frame()
	if ed.sh.Side != SideOutline {
		t.Fatalf("the outline did not open")
	}
	// The same button again closes it.
	ed.sh.Toggle()
	if ed.sh.Side != SideNone {
		t.Errorf("the panel stayed open")
	}
	// Another panel opens on its own.
	ed.sh.Show(SideTree)
	if ed.sh.Side != SideTree {
		t.Errorf("the file tree did not open")
	}
}

// The file list holds the Markdown files of the folder, by name.
func TestReadMarkdown(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"b.md", "a.markdown", "c.txt", ".hidden.md"} {
		if err := os.WriteFile(dir+"/"+name, []byte("# x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	files := readMarkdown(dir)
	if len(files) != 2 {
		t.Fatalf("%d files, want 2: %+v", len(files), files)
	}
	if files[0].name != "a.markdown" || files[1].name != "b.md" {
		t.Errorf("files %q, %q", files[0].name, files[1].name)
	}
	if readMarkdown("") != nil {
		t.Errorf("an empty folder lists files")
	}
}

// Return in the middle of a paragraph splits it there, rather than adding a
// line at the end of the document.
func TestReturnSplitsAtCaret(t *testing.T) {
	ed := New(doc.New("第一段第二段"), NewShared())
	tt := ui.NewTester(ed.View, 860, 700)
	tt.Frame()
	ed.Activate(0)
	tt.Frame()

	// Put the caret before the last three characters.
	for i := 0; i < 3; i++ {
		tt.Key(0, ui.KeyLeft)
	}
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()

	src := ed.Doc().Source()
	if src != "第一段\n\n第二段" {
		t.Fatalf("the split is %q, want the paragraph cut at the caret", src)
	}
	if ed.Doc().Len() != 2 {
		t.Errorf("%d blocks, want 2: %q", ed.Doc().Len(), src)
	}
}

// The file tree lists folders before files, and closes a folder that was.
func TestTreeRows(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a.md", "zzz.md", "sub/b.md", "sub/c.txt", ".hidden.md"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("# x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	ed := New(doc.New(""), NewShared())
	rows := ed.treeRows(dir)
	var got []string
	for _, r := range rows {
		got = append(got, strings.Repeat(" ", r.depth)+r.item.name)
	}
	want := []string{"sub", " b.md", "a.md", "zzz.md"}
	if len(got) != len(want) {
		t.Fatalf("tree %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("tree %q, want %q", got, want)
		}
	}

	ed.collapsed = map[string]bool{filepath.Join(dir, "sub"): true}
	rows = ed.treeRows(dir)
	if len(rows) != 3 || !rows[0].closed {
		t.Errorf("a closed folder still shows its files: %+v", rows)
	}
}

// The file list shows the first line of a file, without its markers.
func TestFirstLine(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.md")
	if err := os.WriteFile(p, []byte("## 今天做的事\n\n正文\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := firstLine(p); got != "今天做的事" {
		t.Errorf("first line %q", got)
	}
	if got := firstLine(filepath.Join(dir, "none.md")); got != "" {
		t.Errorf("a missing file gives %q", got)
	}
}

// Right-clicking the text being written offers the input's own commands, and
// then the rest of Typora's menu.
func TestEditMenu(t *testing.T) {
	ed := New(doc.New(sample), NewShared())
	tt := ui.NewTester(ed.View, 860, 900)
	tt.Frame()
	tt.Frame()
	tt.ClickAt(300, 90)
	tt.Frame()

	tt.RightClickAt(300, 80)
	tt.Frame()
	menu := tt.Menu()
	if len(menu) == 0 {
		t.Fatal("no menu on the text being written")
	}
	if menu[len(menu)-1] != "自动填充" {
		t.Errorf("menu %v does not end as Typora's does", menu)
	}
	var hasParagraph bool
	for _, l := range menu {
		if l == "段落" {
			hasParagraph = true
		}
	}
	if !hasParagraph {
		t.Errorf("menu %v has no 段落", menu)
	}
	// The input's own items come first: four of them, then a separator.
	if menu[3] != "-" && menu[4] != "-" {
		t.Errorf("the input's commands are not first: %v", menu)
	}
}

// The divider between the panel and the page moves with the pointer, and the
// panel keeps the width it is dragged to.
func TestSidebarDrag(t *testing.T) {
	ed := New(doc.New(sample), NewShared())
	ed.sh.Side = SideOutline
	tt := ui.NewTester(ed.View, 900, 700)
	for i := 0; i < 3; i++ {
		tt.Frame()
	}
	start := ed.sh.SideWidth
	if start < 200 {
		t.Fatalf("the panel starts at %v", start)
	}

	// The divider sits at the panel's right edge.
	tt.Press(start, 300)
	tt.Move(start+60, 300)
	tt.Frame()
	tt.Move(start+60, 300)
	tt.Frame()
	tt.Release(start+60, 300)
	tt.Frame()

	if ed.sh.SideWidth <= start+20 {
		t.Fatalf("the panel is %v wide after a drag from %v", ed.sh.SideWidth, start)
	}
}

// Inserting a picture copies it beside the document and puts the Markdown
// that shows it where the caret is.
func TestInsertImage(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "pic.png")
	// A PNG's signature is enough: the file is copied, not decoded.
	if err := os.WriteFile(src, []byte("\x89PNG\r\n\x1a\n\x00\x00"), 0o644); err != nil {
		t.Fatal(err)
	}

	ed := New(doc.New("第一段"), NewShared())
	ed.sh.Folder = dir // the folder the document would be saved in
	tt := ui.NewTester(ed.View, 860, 700)
	tt.Frame()
	ed.Activate(0)
	tt.Frame()

	if err := ed.InsertImage(src); err != nil {
		t.Fatalf("insert: %v", err)
	}
	tt.Frame()

	copied := filepath.Join(dir, "assets", "pic.png")
	if _, err := os.Stat(copied); err != nil {
		t.Fatalf("the picture was not copied: %v", err)
	}
	if got := ed.Doc().Source(); !strings.Contains(got, "![pic](assets/pic.png)") {
		t.Errorf("the Markdown is missing:\n%s", got)
	}
	if ed.Active() < 0 && !ed.Pending() {
		t.Errorf("the editor left the block the picture went into")
	}
}

// A second picture of the same name does not overwrite the first.
func TestUniqueName(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.png"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := uniqueName(dir, "a.png"); got != "a-1.png" {
		t.Errorf("uniqueName = %q, want a-1.png", got)
	}
	if got := uniqueName(dir, "b.png"); got != "b.png" {
		t.Errorf("uniqueName of a new name = %q", got)
	}
}

// The source mode shows the whole document, and what is typed there is in
// the document when it is left.
func TestSourceMode(t *testing.T) {
	ed := New(doc.New(sample), NewShared())
	tt := ui.NewTester(ed.View, 860, 700)
	tt.Frame()
	tt.Frame()
	if ed.Source() {
		t.Fatal("the editor starts in source mode")
	}

	ed.ToggleSource()
	tt.Frame()
	if !ed.Source() {
		t.Fatal("the source mode did not open")
	}
	if ed.Doc().Source() != sample {
		t.Fatalf("the source mode changed the document")
	}

	// What is typed reaches the document at once, so a save keeps it.
	tt.Type("\n\n写在源码里的字")
	tt.Frame()
	if !strings.Contains(ed.Doc().Source(), "写在源码里的字") {
		t.Errorf("the typing is not in the document:\n%s", ed.Doc().Source())
	}

	ed.ToggleSource()
	tt.Frame()
	if ed.Source() {
		t.Errorf("the source mode stayed open")
	}
	if !strings.Contains(ed.Doc().Source(), "写在源码里的字") {
		t.Errorf("the blocks lost the typing")
	}
}

// A long document shows whole in the source mode, and its page scrolls.
func TestSourceModeLong(t *testing.T) {
	long := strings.Repeat("这是一段用来撑高源码页的文字。\n\n", 120)
	ed := New(doc.New(long), NewShared())
	tt := ui.NewTester(ed.View, 900, 700)
	tt.Frame()
	ed.ToggleSource()
	tt.Frame()
	tt.Frame()
	if got := ed.Doc().Source(); got != long {
		t.Errorf("the source mode changed a long document: %d bytes, want %d", len(got), len(long))
	}
	if !strings.Contains(ed.sourceText, "这是一段用来撑高源码页的文字。") {
		t.Errorf("the source does not hold the document")
	}
}

// The find bar counts the matches, walks them, and replaces them.
func TestFind(t *testing.T) {
	ed := New(doc.New("块级编辑。\n\n块级块级，还是块级。"), NewShared())
	tt := ui.NewTester(ed.View, 860, 700)
	tt.Frame()
	tt.Frame()

	ed.OpenFind(false)
	ed.findText = "块级"
	ed.refreshHits()
	tt.Frame()

	if _, total := ed.HitCount(); total != 4 {
		t.Fatalf("%d matches, want 4", total)
	}
	if at, _ := ed.HitCount(); at != 1 {
		t.Errorf("starts at %d, want 1", at)
	}

	ed.FindNext()
	if at, _ := ed.HitCount(); at != 2 {
		t.Errorf("after next: %d, want 2", at)
	}
	ed.FindPrev()
	ed.FindPrev()
	if at, _ := ed.HitCount(); at != 4 {
		t.Errorf("before the first, it wraps to %d, want 4", at)
	}

	// A bar with no matches says so, and stays put.
	ed.findText = "没有的词"
	ed.refreshHits()
	if _, total := ed.HitCount(); total != 0 {
		t.Errorf("%d matches for a missing word", total)
	}

	// Replacing one match writes the replacement and looks again.
	ed.findText = "块级"
	ed.refreshHits()
	ed.replaceText = "段落"
	ed.ReplaceCurrent()
	tt.Frame()
	if got := ed.Doc().Source(); !strings.Contains(got, "段落编辑") {
		t.Errorf("the first match was not replaced:\n%s", got)
	}
	if _, total := ed.HitCount(); total != 3 {
		t.Errorf("%d matches after one replacement, want 3", total)
	}

	// Replacing all writes it everywhere.
	ed.ReplaceAll()
	tt.Frame()
	if got := ed.Doc().Source(); strings.Contains(got, "块级") {
		t.Errorf("matches are left:\n%s", got)
	}
	if _, total := ed.HitCount(); total != 0 {
		t.Errorf("%d matches after replacing all", total)
	}

	ed.CloseFind()
	if ed.FindOpen() {
		t.Errorf("the bar stayed open")
	}
}

// The case and whole-word switches narrow the search.
func TestFindSwitches(t *testing.T) {
	ed := New(doc.New("Block block BLOCK"), NewShared())
	ed.findText = "block"
	ed.OpenFind(false)
	if _, total := ed.HitCount(); total != 3 {
		t.Fatalf("%d matches, want 3", total)
	}
	ed.findCase = true
	ed.refreshHits()
	if _, total := ed.HitCount(); total != 1 {
		t.Errorf("case matters: %d matches, want 1", total)
	}
}

// The count follows what is written while the bar is open, so that a match
// typed after the search is counted, and marked, at once.
func TestFindCountFollowsEdits(t *testing.T) {
	ed := New(doc.New("用户一\n\n用户二"), NewShared())
	tt := ui.NewTester(ed.View, 860, 600)
	tt.Frame()
	ed.OpenFind(false)
	ed.findText = "用户"
	ed.refreshHits()
	tt.Frame()
	if _, total := ed.HitCount(); total != 2 {
		t.Fatalf("%d matches, want 2", total)
	}

	ed.StartParagraph()
	tt.Frame()
	tt.Type("用户三")
	tt.Frame()
	if _, total := ed.HitCount(); total != 3 {
		t.Errorf("%d matches after typing a third, want 3", total)
	}
}

// A search marks the words themselves wherever text shows: in a heading, a
// paragraph, a list item, a quote and a table's cells, in the middle of
// the markup — but never in a link's address or a code block.
func TestMarksWords(t *testing.T) {
	src := "# 用户手册\n\n这是用户的段落。\n\n" +
		"- 用户一项\n- **加粗的用户**\n\n" +
		"> 引用里的用户\n\n" +
		"| 列 | 值 |\n| --- | --- |\n| 用户 | 其他 |\n\n" +
		"[用户](https://example.com/用户) 结尾\n\n" +
		"```\n用户\n```\n"
	ed := New(doc.New(src), NewShared())
	ed.findText = "用户"
	ed.OpenFind(false)

	cases := []struct {
		block     int
		want, not string
	}{
		{0, "# <mark>用户</mark>手册", ""},
		{1, "这是<mark>用户</mark>的段落。", ""},
		{2, "- <mark>用户</mark>一项", ""},
		{2, "**加粗的<mark>用户</mark>**", ""},
		{3, "> 引用里的<mark>用户</mark>", ""},
		{4, "| <mark>用户</mark> | 其他", ""},
		{5, "[<mark>用户</mark>](", ""},
		{5, "", "example.com/<mark>用户</mark>"},
	}
	for _, c := range cases {
		got, ok := ed.markedSource(c.block)
		if !ok {
			t.Fatalf("block %d has no marked source", c.block)
		}
		if c.want != "" && !strings.Contains(got, c.want) {
			t.Errorf("block %d: %q does not hold %q", c.block, got, c.want)
		}
		if c.not != "" && strings.Contains(got, c.not) {
			t.Errorf("block %d marks a link's address: %q", c.block, got)
		}
	}
	if _, ok := ed.markedSource(6); ok {
		t.Errorf("a code block got a marked source")
	}
}

// The tags go only where the bytes are the text itself: an escape, an
// entity and an address written as a link keep their markup.
func TestMarksLeaveMarkupAlone(t *testing.T) {
	ed := New(doc.New("a\\*b 和 &amp; 与 <https://example.com>\n"), NewShared())
	ed.OpenFind(false)
	for _, q := range []string{"*", "amp", "example"} {
		ed.findText = q
		ed.refreshHits()
		if got, ok := ed.markedSource(0); ok {
			t.Errorf("searching %q marked the markup: %q", q, got)
		}
	}
}

// Undo puts back what the document was before the typing, and redo takes it
// away again.
func TestUndoRedo(t *testing.T) {
	ed := New(doc.New("开头"), NewShared())
	tt := ui.NewTester(ed.View, 860, 600)
	tt.Frame()
	ed.Activate(0)
	tt.Frame()
	tt.Type("甲")
	tt.Frame()
	if src := ed.Doc().Source(); !strings.Contains(src, "甲") {
		t.Fatalf("the typing did not reach the document: %q", src)
	}

	ed.Undo()
	tt.Frame()
	if src := ed.Doc().Source(); strings.Contains(src, "甲") {
		t.Errorf("undo left the typing: %q", src)
	}

	ed.Redo()
	tt.Frame()
	if src := ed.Doc().Source(); !strings.Contains(src, "甲") {
		t.Errorf("redo did not put it back: %q", src)
	}
}

// Backspace on an empty paragraph takes the blank line Return made, one at a
// time.
func TestBackspaceTakesBlankLine(t *testing.T) {
	ed := New(doc.New("第一段"), NewShared())
	tt := ui.NewTester(ed.View, 860, 600)
	tt.Frame()
	ed.StartParagraph()
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if got := strings.Count(ed.Doc().Source(), "\n"); got != 4 {
		t.Fatalf("%d newlines, want 4", got)
	}

	tt.Key(0, ui.KeyBackspace)
	tt.Frame()
	if got := strings.Count(ed.Doc().Source(), "\n"); got != 3 {
		t.Errorf("%d newlines after Backspace, want 3", got)
	}
	tt.Key(0, ui.KeyBackspace)
	tt.Frame()
	if got := strings.Count(ed.Doc().Source(), "\n"); got != 2 {
		t.Errorf("%d newlines after the second Backspace, want 2", got)
	}
}

// A picture whose file name has a space in it still shows as a picture: the
// address is written as Markdown takes it.
func TestImageAddressEscaped(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "a b.png")
	if err := os.WriteFile(src, []byte{0x89, 'P', 'N', 'G'}, 0o644); err != nil {
		t.Fatal(err)
	}
	sh := NewShared()
	sh.Folder = dir
	ed := New(doc.New("x"), sh)
	tt := ui.NewTester(ed.View, 800, 600)
	tt.Frame()
	if err := ed.InsertImage(src); err != nil {
		t.Fatalf("insert: %v", err)
	}
	got := ed.Doc().Source()
	if !strings.Contains(got, "![a b](assets/a%20b.png)") {
		t.Fatalf("the address has a space in it: %q", got)
	}
	d := ed.Doc()
	for i := 0; i < d.Len(); i++ {
		for _, in := range d.Parse().Inlines(d.Blocks()[i].Index) {
			if in.Kind == parser.Image {
				return
			}
		}
	}
	t.Errorf("the view does not read it as a picture: %q", got)
}

// A picture whose address has a space in it is drawn as a picture: the view
// reads the address as Markdown takes it, the source staying as it is.
func TestReadableImageSrc(t *testing.T) {
	cases := []struct{ in, want string }{
		{"![a b](assets/a b.png)", "![a b](<assets/a b.png>)"},
		{"![a](assets/a.png)", "![a](assets/a.png)"},
		{"![a](<assets/a b.png>)", "![a](<assets/a b.png>)"},
		{"前面 ![a b](x y.png) 后面", "前面 ![a b](<x y.png>) 后面"},
	}
	for _, c := range cases {
		if got := readableSrc(c.in); got != c.want {
			t.Errorf("readableSrc(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// Return at the end of a list item or a quote continues it with its marker,
// and an ordered one counts up.
func TestReturnContinuesLists(t *testing.T) {
	cases := []struct{ start, next string }{
		{"- 待办一", "- "},
		{"* 星号", "* "},
		{"1. 第一", "2. "},
		{"- [ ] 买牛奶", "- [ ] "},
		{"> 引用", "> "},
	}
	for _, c := range cases {
		ed := New(doc.New(c.start), NewShared())
		tt := ui.NewTester(ed.View, 860, 600)
		tt.Frame()
		ed.Activate(0)
		tt.Frame()
		tt.Key(0, ui.KeyEnd)
		tt.Frame()
		tt.Key(0, ui.KeyEnter)
		tt.Frame()
		if !strings.HasSuffix(ed.draft, "\n"+c.next) {
			t.Errorf("%q: the next line ends in %q, draft %q", c.start, c.next, ed.draft)
		}
		tt.Type("下一条")
		tt.Frame()
		tt.Key(0, ui.KeyEscape)
		tt.Frame()
		want := c.start + "\n" + c.next + "下一条"
		if src := ed.Doc().Source(); src != want {
			t.Errorf("%q: source %q, want %q", c.start, src, want)
		}
	}
}

// Return on a marker with nothing after it leaves the list: the marker goes,
// and an empty paragraph stays.
func TestReturnExitsList(t *testing.T) {
	ed := New(doc.New("- 待办一"), NewShared())
	tt := ui.NewTester(ed.View, 860, 600)
	tt.Frame()
	ed.Activate(0)
	tt.Frame()
	tt.Key(0, ui.KeyEnd)
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if ed.draft != "" {
		t.Fatalf("the marker did not go: %q", ed.draft)
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if src := ed.Doc().Source(); src != "- 待办一" {
		t.Errorf("source %q, want the list as it was", src)
	}
}

// Return in the middle of an item splits it in two, both of them items.
func TestReturnSplitsItem(t *testing.T) {
	ed := New(doc.New("- 两半"), NewShared())
	tt := ui.NewTester(ed.View, 860, 600)
	tt.Frame()
	ed.Activate(0)
	tt.Frame()
	tt.Key(0, ui.KeyHome)
	tt.Frame()
	tt.Key(0, ui.KeyRight)
	tt.Key(0, ui.KeyRight)
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if src := ed.Doc().Source(); src != "- \n- 两半" {
		t.Errorf("source %q, want %q", src, "- \n- 两半")
	}
}

// Return after ``` opens the body of a code block with the caret inside it,
// and Return inside adds a line to the code.
func TestFenceOpensAndTypes(t *testing.T) {
	ed := New(doc.New("一段"), NewShared())
	tt := ui.NewTester(ed.View, 860, 600)
	tt.Frame()
	ed.StartParagraph()
	tt.Frame()
	tt.Type("```")
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if ed.draft != "```\n\n```" {
		t.Fatalf("draft %q, want a closed fence", ed.draft)
	}
	tt.Type("fmt.Println()")
	tt.Frame()
	if ed.draft != "```\nfmt.Println()\n```" {
		t.Fatalf("draft %q — the caret did not go inside the fence", ed.draft)
	}
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if ed.draft != "```\nfmt.Println()\n\n```" {
		t.Errorf("draft %q, want a line added inside", ed.draft)
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if src := ed.Doc().Source(); !strings.Contains(src, "```\nfmt.Println()\n\n```") {
		t.Errorf("source %q", src)
	}
}

// Typing # opens an empty heading and keeps writing in it, and the heading
// can be deleted again before it has a title.
func TestHashOpensEditing(t *testing.T) {
	ed := New(doc.New("上面一段"), NewShared())
	tt := ui.NewTester(ed.View, 760, 420)
	tt.Frame()
	ed.StartParagraph()
	tt.Frame()
	tt.Type("#")
	tt.Frame()
	if !ed.Editing() {
		t.Fatalf("typing # left the editor: draft %q, active %d, pending %v",
			ed.draft, ed.active, ed.pending)
	}
	tt.Type(" 标题")
	tt.Frame()
	if ed.draft != "# 标题" {
		t.Fatalf("draft %q, want %q", ed.draft, "# 标题")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if src := ed.Doc().Source(); !strings.Contains(src, "# 标题") {
		t.Fatalf("source %q", src)
	}

	// And it can be deleted again.
	ed.Activate(1)
	tt.Frame()
	tt.Key(0, ui.KeyEnd)
	tt.Frame()
	for i := 0; i < 8; i++ {
		tt.Key(0, ui.KeyBackspace)
		tt.Frame()
	}
	if ed.Doc().Len() != 1 {
		t.Errorf("the heading is still there: %q", ed.Doc().Source())
	}
}

// The markers of the line the caret is on show faint, and those of the
// other lines hide, so the rest of the block reads as it will once done.
func TestRangesHideOffCaretLine(t *testing.T) {
	ed := New(doc.New("x"), NewShared())
	draft := "- one **bold** line\n- two **bold** line"
	muted := ui.RGBA(120, 120, 120, 0.6)
	plain := ui.RGBA(20, 20, 20, 1)
	hidden := ui.Color{}
	// at is the run that starts at byte start, faint as wantMuted says.
	at := func(rs []ui.TextRange, start int, wantMuted bool) ui.TextRange {
		for _, r := range rs {
			if r.Start == start && (r.Color == muted) == wantMuted {
				return r
			}
		}
		return ui.TextRange{}
	}

	// The caret on the first line: its markers faint, the second line's
	// hidden, the dashes of both lines staying, for each opens its line.
	rs := ed.ranges(draft, 0, muted, plain, hidden)
	if got := at(rs, 0, true); got.Color != muted || got.End != 2 {
		t.Errorf("the first dash is %+v", got)
	}
	if got := at(rs, 20, true); got.Color != muted {
		t.Errorf("the second line's dash hid: %+v", got)
	}
	if got := at(rs, 6, true); got.Color != muted {
		t.Errorf("the caret line's bold opener hid: %+v", got)
	}
	if got := at(rs, 12, true); got.Color != muted {
		t.Errorf("the caret line's bold closer hid: %+v", got)
	}
	if got := at(rs, 26, true); got.Color.A != 0 {
		t.Errorf("the other line's bold opener shows: %+v", got)
	}
	if got := at(rs, 32, true); got.Color.A != 0 {
		t.Errorf("the other line's bold closer shows: %+v", got)
	}
	if got := at(rs, 8, false); got.Color != plain || got.Weight != 700 {
		t.Errorf("the bold text lost its weight: %+v", got)
	}
	if got := at(rs, 28, false); got.Color != plain || got.Weight != 700 {
		t.Errorf("the other bold text lost its weight: %+v", got)
	}

	// The caret on the second line: the other way about.
	rs = ed.ranges(draft, 1, muted, plain, hidden)
	if got := at(rs, 6, true); got.Color.A != 0 {
		t.Errorf("the first line's bold opener shows: %+v", got)
	}
	if got := at(rs, 26, true); got.Color != muted {
		t.Errorf("the caret line's bold opener hid: %+v", got)
	}
	if got := at(rs, 0, true); got.Color != muted {
		t.Errorf("the first dash hid: %+v", got)
	}
	if got := at(rs, 20, true); got.Color != muted {
		t.Errorf("the second dash hid: %+v", got)
	}
}

// LineOf is the line a byte stands on.
func TestLineOf(t *testing.T) {
	s := "ab\ncd\ne"
	if doc.LineOf(s, 0) != 0 || doc.LineOf(s, 2) != 0 {
		t.Errorf("the first line is wrong: %d %d", doc.LineOf(s, 0), doc.LineOf(s, 2))
	}
	if doc.LineOf(s, 3) != 1 || doc.LineOf(s, 5) != 1 {
		t.Errorf("the second line is wrong: %d %d", doc.LineOf(s, 3), doc.LineOf(s, 5))
	}
	if doc.LineOf(s, 6) != 2 || doc.LineOf(s, 99) != 2 {
		t.Errorf("the third line is wrong: %d %d", doc.LineOf(s, 6), doc.LineOf(s, 99))
	}
}

// A list of two items and more edits by the item; a quote, a lone item, a
// fence and a wrapped line edit whole.
func TestItemsMode(t *testing.T) {
	ed := New(doc.New("x"), NewShared())
	yes := "1. 甲\n2. 乙"
	ed.draft = yes
	if !ed.itemsMode() {
		t.Errorf("a two-item list does not edit by the item")
	}
	for _, no := range []string{
		"- 甲\nwrapped line",
		"> 甲\n> 乙",
		"1. 甲",
		"```go\ncode\n```",
		"- 甲\n\n段落",
	} {
		ed.draft = no
		if ed.itemsMode() {
			t.Errorf("%q edits by the item, want whole-block", no)
		}
	}
	ed.draft = yes
}

// Opening a list for writing puts the caret in its first item.
func TestActivateListFocusesFirstItem(t *testing.T) {
	ed := New(doc.New("1. 甲\n2. 乙"), NewShared())
	ed.Activate(0)
	if ed.itemLine != 0 || ed.lineDraft != "1. 甲" {
		t.Errorf("the first item is not the one written: %d %q", ed.itemLine, ed.lineDraft)
	}
}

// Typing in the item reaches the document, as typing in the block does.
func TestItemTypingReachesDocument(t *testing.T) {
	ed := New(doc.New("1. 甲\n2. 乙"), NewShared())
	tt := ui.NewTester(ed.View, 700, 400)
	tt.Frame()
	tt.Frame()
	ed.Activate(0)
	tt.Frame()
	tt.Type("更")
	tt.Frame()
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if src := ed.Doc().Source(); !strings.Contains(src, "更1. 甲") {
		t.Errorf("the typing did not reach the document: %q", src)
	}
}

// The caret that draft runes hold moves into the item that now holds it.
func TestRefocusItem(t *testing.T) {
	ed := New(doc.New("x"), NewShared())
	ed.draft = "1. 甲\n2. 乙"
	ed.caret = 4 // the end of the first item
	ed.refocusItem()
	if ed.itemLine != 0 || ed.itemCaret != 4 || ed.caret != -1 {
		t.Errorf("the end of the first item: %d %d %d", ed.itemLine, ed.itemCaret, ed.caret)
	}
	ed.caret = 5 // the start of the second
	ed.refocusItem()
	if ed.itemLine != 1 || ed.lineDraft != "2. 乙" || ed.itemCaret != 0 {
		t.Errorf("the start of the second item: %d %q %d", ed.itemLine, ed.lineDraft, ed.itemCaret)
	}
}

// Return at the end of an item continues the list, and the item that opens
// is the one being written.
func TestItemEnterContinues(t *testing.T) {
	ed := New(doc.New("1. 甲\n2. 乙"), NewShared())
	tt := ui.NewTester(ed.View, 700, 400)
	tt.Frame()
	tt.Frame()
	ed.Activate(0)
	tt.Frame()
	ed.itemCaret = doc.RuneLen("1. 甲")
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if ed.draft != "1. 甲\n2. \n2. 乙" {
		t.Fatalf("the draft is %q", ed.draft)
	}
	if ed.itemLine != 1 || ed.lineDraft != "2. " {
		t.Errorf("the new item is not being written: %d %q", ed.itemLine, ed.lineDraft)
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if src := ed.Doc().Source(); src != "1. 甲\n2. \n2. 乙" {
		t.Errorf("the source is %q", src)
	}
}

// Return in the middle of an item splits it, and the second half is the
// one being written.
func TestItemEnterSplits(t *testing.T) {
	ed := New(doc.New("1. 甲乙\n2. 丙"), NewShared())
	ed.Activate(0)
	if !ed.itemsMode() || ed.itemLine != 0 || ed.lineDraft != "1. 甲乙" {
		t.Fatalf("the list does not edit by the item: %v %d %q", ed.itemsMode(), ed.itemLine, ed.lineDraft)
	}
	tt := ui.NewTester(ed.View, 700, 400)
	tt.Frame()
	tt.Frame()
	ed.itemCaret = doc.RuneLen("1. 甲")
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if ed.draft != "1. 甲\n2. 乙\n2. 丙" {
		t.Fatalf("the draft is %q", ed.draft)
	}
	if ed.itemLine != 1 || ed.lineDraft != "2. 乙" {
		t.Errorf("the second half is not being written: %d %q", ed.itemLine, ed.lineDraft)
	}
}

// Backspace at the first letter of an item joins it to the one above, its
// marker going, and the caret waits at the seam.
func TestItemBackspaceMerges(t *testing.T) {
	ed := New(doc.New("1. 甲\n2. 乙"), NewShared())
	tt := ui.NewTester(ed.View, 700, 400)
	tt.Frame()
	tt.Frame()
	ed.Activate(0)
	tt.Frame()
	ed.focusItem(1, false)
	tt.Frame()
	tt.Key(0, ui.KeyBackspace)
	tt.Frame()
	if ed.draft != "1. 甲 乙" {
		t.Errorf("the merge is %q", ed.draft)
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if src := ed.Doc().Source(); src != "1. 甲 乙" {
		t.Errorf("the source is %q", src)
	}
}

// Undo in item writing puts the document and the items back.
func TestItemUndo(t *testing.T) {
	ed := New(doc.New("1. 甲\n2. 乙"), NewShared())
	tt := ui.NewTester(ed.View, 700, 400)
	tt.Frame()
	tt.Frame()
	ed.Activate(0)
	tt.Frame()
	tt.Type("更")
	tt.Frame()
	ed.Undo()
	tt.Frame()
	if src := ed.Doc().Source(); strings.Contains(src, "更") {
		t.Errorf("undo left the typing: %q", src)
	}
	if ed.itemLine < 0 || ed.lineDraft != linesOf(ed.draft)[ed.itemLine] {
		t.Errorf("the item being writing did not follow: %d %q %q", ed.itemLine, ed.lineDraft, ed.draft)
	}
}

// The chain a writer walks when emptying an item: the content goes, the
// marker alone stays and the item writing holds, the backspace at the
// marker joins the item above, and typing goes on somewhere real at the
// end of it.
func TestItemEmptyingChain(t *testing.T) {
	ed := New(doc.New("1. 甲\n2. 乙\n3. 丙"), NewShared())
	ed.Activate(0)
	tt := ui.NewTester(ed.View, 700, 400)
	tt.Frame()
	tt.Frame()

	// The content of the last item deleted back to its marker, a
	// Backspace at a time: item writing holds over the emptied item, whose
	// line reads as the marker with its room.
	ed.focusItem(2, true) // the caret at the end of 3. 丙
	tt.Frame()
	for i := 0; i < 2; i++ {
		tt.Key(0, ui.KeyBackspace)
		tt.Frame()
	}
	if !ed.itemsMode() || ed.itemLine != 2 || ed.lineDraft != "3. " {
		t.Fatalf("item writing did not hold: %v %d %q", ed.itemsMode(), ed.itemLine, ed.lineDraft)
	}

	// The next Backspace, at the marker's first letter, takes the marker
	// out the way Typora's does: the item leaves the list, and a paragraph
	// of its own opens where it was.
	tt.Key(0, ui.KeyBackspace)
	tt.Frame()
	if !ed.Pending() || ed.draft != "" {
		t.Errorf("the marker did not leave: pending=%v draft=%q", ed.Pending(), ed.draft)
	}
	if src := ed.Doc().Source(); src != "1. 甲\n2. 乙\n\n" {
		t.Errorf("the list did not keep its two items: %q", src)
	}

	// And the typing that follows lands in the paragraph that opened.
	tt.Type("新内容")
	tt.Frame()
	if src := ed.Doc().Source(); src != "1. 甲\n2. 乙\n\n新内容" {
		t.Errorf("the typing did not land in the new paragraph: %q", src)
	}
}

// mergeJoin is the arithmetic of a join: the emptied item joins as
// nothing, the written one keeps its text after the one above.
func TestMergeJoin(t *testing.T) {
	if m, c := mergeJoin("2. 乙", "3."); m != "2. 乙" || c != doc.RuneLen("2. 乙") {
		t.Errorf("the emptied item joined as %q at %d", m, c)
	}
	if m, c := mergeJoin("2. 乙", "3. 丙"); m != "2. 乙 丙" || c != doc.RuneLen("2. 乙")+1 {
		t.Errorf("the written item joined as %q at %d", m, c)
	}
	if m, c := mergeJoin("- 甲", "- 乙"); m != "- 甲 乙" || c != doc.RuneLen("- 甲")+1 {
		t.Errorf("the bullet join is %q at %d", m, c)
	}
}

// A block that holds more than what was written — the room a deleted line
// left, or a mark the parser set apart, come back with it — is healed to
// what the writer holds, and the stray tail is not adopted as their text.
func TestSyncHealsStrayTail(t *testing.T) {
	ed := New(doc.New("### 测试###"), NewShared())
	ed.Activate(0)
	ed.draft = "### 测试" // what the writer holds; the doc grew a tail
	ed.selA = doc.RuneOffset(ed.draft, doc.RuneLen(ed.draft))
	ed.selB = ed.selA
	ed.sync()
	if src := ed.Doc().Source(); src != "### 测试" {
		t.Errorf("the stray tail stayed: %q", src)
	}
	if ed.draft != "### 测试" {
		t.Errorf("the tail was adopted into the draft: %q", ed.draft)
	}
}

// An emptied item in the middle of a list leaves a paragraph of its own
// between the items that stay, and the first one leaves it above them.
func TestItemToParagraphMiddleAndFirst(t *testing.T) {
	ed := New(doc.New("1. 甲\n2. 乙\n3. 丙"), NewShared())
	tt := ui.NewTester(ed.View, 700, 400)
	tt.Frame()
	tt.Frame()
	ed.Activate(0)
	tt.Frame()

	// The middle item emptied, and its marker taken: the list splits
	// around the paragraph that opens. Three Backspaces: the content, the
	// marker's room, the marker itself.
	ed.focusItem(1, true)
	tt.Frame()
	for i := 0; i < 3; i++ {
		tt.Key(0, ui.KeyBackspace)
		tt.Frame()
	}
	if !ed.Pending() {
		t.Fatalf("the middle item did not leave: %q", ed.draft)
	}
	tt.Type("新")
	tt.Frame()
	if src := ed.Doc().Source(); src != "1. 甲\n\n新\n\n3. 丙" {
		t.Errorf("the middle paragraph is %q", src)
	}

	// The first item of another list, emptied and unmarked: the paragraph
	// opens above the items that stay.
	ed2 := New(doc.New("1. 甲\n2. 乙"), NewShared())
	t2 := ui.NewTester(ed2.View, 700, 400)
	t2.Frame()
	t2.Frame()
	ed2.Activate(0)
	t2.Frame()
	ed2.focusItem(0, true)
	t2.Frame()
	for i := 0; i < 3; i++ {
		t2.Key(0, ui.KeyBackspace)
		t2.Frame()
	}
	if !ed2.Pending() {
		t.Fatalf("the first item did not leave: %q", ed2.draft)
	}
	t2.Type("头")
	t2.Frame()
	if src := ed2.Doc().Source(); src != "头\n\n2. 乙" {
		t.Errorf("the first paragraph is %q", src)
	}
}

// The heading takes its characters in the ways a writer and an input
// method produce them — one at a time, several in a frame, a word
// committed whole — and the document holds exactly what was typed.
func TestHeadTypingWays(t *testing.T) {
	// One character at a time, from a fresh paragraph.
	ed := New(doc.New(""), NewShared())
	tt := ui.NewTester(ed.View, 700, 400)
	tt.Frame()
	tt.Frame()
	tt.ClickAt(300, 100)
	tt.Frame()
	for _, ch := range []string{"#", "#", "#", " ", "测", "试"} {
		tt.Type(ch)
		tt.Frame()
	}
	if src := ed.Doc().Source(); src != "### 测试" {
		t.Errorf("the typed heading is %q", src)
	}

	// Several events inside one frame, as a fast typist makes.
	ed2 := New(doc.New(""), NewShared())
	t2 := ui.NewTester(ed2.View, 700, 400)
	t2.Frame()
	t2.Frame()
	t2.ClickAt(300, 100)
	t2.Frame()
	t2.Type("#")
	t2.Type("#")
	t2.Type("#")
	t2.Type(" ")
	t2.Type("测试")
	t2.Frame()
	t2.Frame()
	if src := ed2.Doc().Source(); src != "### 测试" {
		t.Errorf("the burst heading is %q", src)
	}

	// Return at the end of a heading opens the paragraph below, and the
	// typing that follows lands there, not in the heading.
	ed3 := New(doc.New("### 测试"), NewShared())
	t3 := ui.NewTester(ed3.View, 700, 500)
	t3.Frame()
	t3.Frame()
	ed3.Activate(0)
	t3.Frame()
	end := doc.RuneOffset(ed3.draft, doc.RuneLen(ed3.draft))
	ed3.selA, ed3.selB = end, end
	t3.Frame()
	t3.Key(0, ui.KeyEnter)
	t3.Frame()
	t3.Type("###")
	t3.Frame()
	if src := ed3.Doc().Source(); src != "### 测试\n\n###" {
		t.Errorf("the return and the marks made %q", src)
	}

	// The writer deletes the heading back and retypes it: the paragraph
	// and the heading trade places, and the document stays whole.
	ed4 := New(doc.New("### 测试"), NewShared())
	t4 := ui.NewTester(ed4.View, 700, 400)
	t4.Frame()
	t4.Frame()
	ed4.Activate(0)
	t4.Frame()
	for i := 0; i < 6; i++ {
		t4.Key(0, ui.KeyBackspace)
		t4.Frame()
	}
	for _, ch := range []string{"#", "#", "#", " ", "测", "试"} {
		t4.Type(ch)
		t4.Frame()
	}
	if src := ed4.Doc().Source(); src != "### 测试" {
		t.Errorf("the retyped heading is %q", src)
	}
}

// The blank lines Return made between two blocks go, one at a time, and
// the last of them joins the block above rather than staying forever: the
// caret waits at the end of it, in its last item when it is a list.
func TestBackspaceBlankLastLineJoins(t *testing.T) {
	ed := New(doc.New("- 甲\n- 乙\n\n\n\n\n## 二"), NewShared())
	tt := ui.NewTester(ed.View, 760, 520)
	tt.Frame()
	tt.Frame()
	// The caret on the last blank line of the gap, as a click puts it:
	// five breaks are between the list and the heading, and the caret
	// sits after the fifth.
	ed.startAt(15)
	tt.Frame()
	if !ed.Pending() {
		t.Fatal("the gap did not take the caret")
	}

	// Each Backspace takes a blank line out, until the separation between
	// the neighbours is all that is left.
	tt.Key(0, ui.KeyBackspace)
	tt.Frame()
	if got := ed.Doc().Source(); got != "- 甲\n- 乙\n\n\n\n## 二" {
		t.Fatalf("the first blank line is %q", got)
	}
	tt.Key(0, ui.KeyBackspace)
	tt.Frame()
	if got := ed.Doc().Source(); got != "- 甲\n- 乙\n\n\n## 二" {
		t.Fatalf("the second blank line is %q", got)
	}

	// The last one: the empty line joins the list, the caret waits at the
	// end of its last item, and the writing goes on there.
	tt.Key(0, ui.KeyBackspace)
	tt.Frame()
	if ed.Pending() {
		t.Errorf("the empty line stayed under the caret")
	}
	if ed.active != 0 || ed.itemLine != 1 || ed.lineDraft != "- 乙" {
		t.Errorf("the caret is not at the end of the last item: %d %d %q",
			ed.active, ed.itemLine, ed.lineDraft)
	}
	if got := ed.Doc().Source(); got != "- 甲\n- 乙\n\n\n## 二" {
		t.Errorf("the blocks are not what they were: %q", got)
	}

	// And the typing goes on in the item the caret waits in.
	tt.Type("!")
	tt.Frame()
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if got := ed.Doc().Source(); !strings.Contains(got, "- 乙!") {
		t.Errorf("the typing did not land: %q", got)
	}
}

// The empty line at the very start of the document goes too, and nothing
// is there to join when the last of them is taken.
func TestBackspaceBlankAtStart(t *testing.T) {
	ed := New(doc.New("\n\n\n第一段"), NewShared())
	tt := ui.NewTester(ed.View, 760, 400)
	tt.Frame()
	tt.Frame()
	ed.startAt(3)
	tt.Frame()
	for i := 0; i < 3; i++ {
		tt.Key(0, ui.KeyBackspace)
		tt.Frame()
	}
	if got := ed.Doc().Source(); got != "第一段" {
		t.Fatalf("the leading lines are %q", got)
	}
	tt.Key(0, ui.KeyBackspace)
	tt.Frame()
	if got := ed.Doc().Source(); got != "第一段" {
		t.Errorf("the text itself lost a letter: %q", got)
	}
}
