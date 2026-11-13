package editor

import (
	"image/png"
	"os"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"paperly/internal/doc"
)

const mathBody = `普通一段文字，用来对照公式的大小。

$$
\int_0^1 x^2\,dx = \frac{1}{3}
$$

$$
\sum_{n=1}^{\infty}\frac{1}{n^2} = \frac{\pi^2}{6}
$$

结尾一段。
`

// A formula block is drawn as a picture: its TeX is not in the page's text.
func TestMathBlockIsDrawn(t *testing.T) {
	ed := New(doc.New(mathBody), NewShared())
	tt := ui.NewTester(ed.View, 860, 700)
	tt.Frame()
	tt.Frame()

	for _, text := range tt.Texts() {
		if len(text) > 6 && (text == `\int_0^1 x^2\,dx = \frac{1}{3}` || text == `\frac{\pi^2}{6}`) {
			t.Errorf("the TeX shows as text: %q", text)
		}
	}
	if len(ed.mathCache) != 2 {
		t.Errorf("drew %d formulas, want 2", len(ed.mathCache))
	}
	f, err := os.Create("/tmp/math-page.png")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, tt.Image()); err != nil {
		t.Fatal(err)
	}
}

// The paragraph menu puts a formula below the paragraph and opens it for
// writing, with the caret inside its walls.
func TestInsertFormula(t *testing.T) {
	ed := New(doc.New("第一段。\n\n第二段。\n"), NewShared())
	tt := ui.NewTester(ed.View, 860, 700)
	tt.Frame()
	tt.Frame()

	tt.RightClickAt(300, 40)
	tt.Frame()
	if err := tt.ChooseMenuItem("段落", "公式块"); err != nil {
		t.Fatalf("%v", err)
	}
	tt.Frame()

	got := ed.Doc().Source()
	if !strings.Contains(got, "$$\n\n$$") {
		t.Fatalf("no formula was put in:\n%s", got)
	}
	if i := strings.Index(got, "$$"); i < strings.Index(got, "第一段。") {
		t.Errorf("the formula went above the paragraph:\n%s", got)
	}
	if ed.Active() < 0 {
		t.Errorf("the formula was not opened for writing")
	}
	tt.Type("E = mc^2")
	tt.Frame()
	if got := ed.Doc().Source(); !strings.Contains(got, "$$\nE = mc^2\n$$") {
		t.Errorf("the typing is not in the formula:\n%s", got)
	}
}

// Clicking a formula shows its TeX, which is what is written there; leaving
// it puts the picture back.
func TestMathBlockEdits(t *testing.T) {
	ed := New(doc.New("前面。\n\n$$\nE = mc^2\n$$\n\n后面。\n"), NewShared())
	tt := ui.NewTester(ed.View, 860, 700)
	tt.Frame()
	tt.Frame()

	// The formula is the second block: click in the middle of its room.
	tt.ClickAt(300, ed.tops[1]+ed.heights[1]/2)
	tt.Frame()
	if ed.Active() != 1 {
		t.Fatalf("clicking the formula opened block %d", ed.Active())
	}
	if got, want := ed.Draft(), "$$\nE = mc^2\n$$"; got != want {
		t.Errorf("the formula's source is %q, want %q", got, want)
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if ed.Active() >= 0 {
		t.Error("the formula stayed open after Escape")
	}
	if len(ed.mathCache) == 0 {
		t.Error("the formula was not drawn again")
	}
}

// Return after $$ on a line of its own opens a formula; inside one it adds
// a line to its TeX rather than splitting the block.
func TestEnterInFormula(t *testing.T) {
	ed := New(doc.New(""), NewShared())
	tt := ui.NewTester(ed.View, 860, 700)
	tt.Frame()
	ed.StartParagraph()
	tt.Frame()
	tt.Type("$$")
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if got := ed.Doc().Source(); got != "$$\n\n$$" {
		t.Fatalf("Return after $$ made %q", got)
	}
	tt.Type("x = 1")
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	tt.Type("y = 2")
	tt.Frame()
	if got := ed.Doc().Source(); !strings.Contains(got, "x = 1\ny = 2") {
		t.Errorf("the line was not added inside the formula:\n%s", got)
	}
	if ed.Doc().Len() != 1 {
		t.Errorf("the formula was split into %d blocks", ed.Doc().Len())
	}
}

// The insert menu puts a formula where the empty room was clicked.
func TestInsertFormulaFromThePageMenu(t *testing.T) {
	ed := New(doc.New("第一段。\n"), NewShared())
	tt := ui.NewTester(ed.View, 860, 700)
	tt.Frame()
	tt.Frame()

	tt.RightClickAt(20, 300)
	tt.Frame()
	if err := tt.ChooseMenuItem("插入", "公式块"); err != nil {
		t.Fatalf("%v", err)
	}
	tt.Frame()
	if got := ed.Doc().Source(); !strings.HasSuffix(strings.TrimSpace(got), "$$") {
		t.Errorf("no formula was put in at the end:\n%s", got)
	}
}

// The rendering of formulas can be turned off, for the documents that write
// about prices: what was a formula shows as its source, and the dollars
// stay dollars.
func TestMathCanBeTurnedOff(t *testing.T) {
	sh := NewShared()
	sh.SetMath(false)
	ed := New(doc.New("公式 $x^2$ 与价格 $5 到 $10。\n"), sh)
	tt := ui.NewTester(ed.View, 860, 400)
	tt.Frame()
	tt.Frame()

	all := strings.Join(tt.Texts(), " ")
	if strings.ContainsRune(all, '\uFFFC') {
		t.Errorf("a formula was drawn with the switch off: %q", all)
	}
	if !strings.Contains(all, "$x^2$") {
		t.Errorf("the formula did not fall back to its source: %q", all)
	}
	if len(ed.mathCache) != 0 {
		t.Errorf("%d formulas were drawn", len(ed.mathCache))
	}

	// Back on, the same document draws.
	sh.SetMath(true)
	tt.Frame()
	tt.Frame()
	if len(ed.mathCache) == 0 {
		t.Error("the formula was not drawn after the switch came back")
	}
}
