package editor

import (
	"strings"
	"testing"

	mathx "paperly/internal/math"

	"github.com/egoist/mygo/ui"

	"paperly/internal/doc"
)

const inlineBody = `# 相对论

质能方程 $E = mc^2$ 说明质量与能量等价。

价格从 $5 涨到 $10，这不是公式。

行内求和 $\sum_{n=1}^{\infty}\frac{1}{n^2}$ 挤在一行里。
`

// Math between single dollars is an object of the text: the TeX itself is
// not in the page, and the formula is drawn at the size of the text.
func TestInlineMathIsAnObjectOfTheText(t *testing.T) {
	ed := New(doc.New(inlineBody), NewShared())
	tt := ui.NewTester(ed.View, 900, 640)
	tt.Frame()
	tt.Frame()

	var para string
	for _, s := range tt.Texts() {
		if strings.HasPrefix(s, "质能方程") {
			para = s
		}
	}
	if para == "" {
		t.Fatalf("the paragraph is not there: %v", tt.Texts())
	}
	if !strings.Contains(para, "\uFFFC") {
		t.Errorf("no formula stands in the line: %q", para)
	}
	if strings.Contains(para, "mc^2") || strings.Contains(para, "$") {
		t.Errorf("the TeX shows as text: %q", para)
	}

	var found bool
	for k := range ed.mathCache {
		if k.tex == "E = mc^2" {
			found = true
			if k.style != mathx.Text {
				t.Errorf("the formula was set in style %v, want the text's", k.style)
			}
		}
	}
	if !found {
		t.Errorf("the formula was not drawn: %v", ed.mathCache)
	}
}

// Dollars that are not math stay dollars: $5 and $10 are prices.
func TestDollarsThatAreNotMathStayText(t *testing.T) {
	ed := New(doc.New(inlineBody), NewShared())
	tt := ui.NewTester(ed.View, 900, 640)
	tt.Frame()
	tt.Frame()

	var para string
	for _, s := range tt.Texts() {
		if strings.HasPrefix(s, "价格从") {
			para = s
		}
	}
	if !strings.Contains(para, "$5") || !strings.Contains(para, "$10") {
		t.Errorf("the prices lost their dollars: %q", para)
	}
	for k := range ed.mathCache {
		if strings.Contains(k.tex, "5") {
			t.Errorf("%q was taken for math", k.tex)
		}
	}
}

// A formula in a heading is set at the heading's size, the same formula in
// the text at the text's: the size comes from the words around it.
func TestInlineMathFollowsTheSizeOfTheText(t *testing.T) {
	src := "# 标题 $x^2$\n\n正文 $x^2$\n"
	ed := New(doc.New(src), NewShared())
	tt := ui.NewTester(ed.View, 900, 500)
	tt.Frame()
	tt.Frame()

	var sizes []float32
	for k := range ed.mathCache {
		if k.tex == "x^2" {
			sizes = append(sizes, k.size)
		}
	}
	if len(sizes) < 2 {
		t.Fatalf("the formula was drawn at %d sizes, want two: %v", len(sizes), sizes)
	}
	if sizes[0] == sizes[1] {
		t.Errorf("both were set at %.1f, the heading's is not larger", sizes[0])
	}
}

// Clicking the paragraph shows its source, the formula's dollars and TeX
// included, and leaving it draws the formula again.
func TestInlineMathEditsAsSource(t *testing.T) {
	ed := New(doc.New(inlineBody), NewShared())
	tt := ui.NewTester(ed.View, 900, 640)
	tt.Frame()
	tt.Frame()

	// The paragraph just under the heading.
	tt.ClickAt(300, ed.tops[1]+8)
	tt.Frame()
	if ed.Active() != 1 {
		t.Fatalf("the click opened block %d", ed.Active())
	}
	if got := ed.Draft(); !strings.Contains(got, "$E = mc^2$") {
		t.Errorf("the source is %q", got)
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if !strings.Contains(strings.Join(tt.Texts(), " "), "\uFFFC") {
		t.Error("the formula was not drawn again")
	}
}

// A formula the renderer cannot read shows its TeX rather than nothing.
func TestUnknownInlineMathShowsItsSource(t *testing.T) {
	ed := New(doc.New("前 $\\unknownmacro{x}$ 后\n"), NewShared())
	tt := ui.NewTester(ed.View, 900, 400)
	tt.Frame()
	tt.Frame()
	all := strings.Join(tt.Texts(), " ")
	if !strings.Contains(all, "\\unknownmacro") {
		t.Errorf("the formula vanished: %q", all)
	}
}

// A formula that a selection holds is copied as its source, $…$, rather
// than as the placeholder the line keeps for it.
func TestCopyingASelectionKeepsTheTeX(t *testing.T) {
	ed := New(doc.New(inlineBody), NewShared())
	ed.testMod = ui.Alt
	tt := ui.NewTester(ed.View, 900, 640)
	tt.Frame()
	tt.Frame()

	// A click puts a caret in the page, which Cmd+A widens to all of it.
	tt.ClickAt(300, ed.tops[1]+8)
	tt.Frame()
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Frame()
	tt.Command("copy")
	got := tt.Clipboard()
	if !strings.Contains(got, "$E = mc^2$") {
		t.Errorf("the formula did not copy as its source:\n%s", got)
	}
	if strings.ContainsRune(got, '\uFFFC') {
		t.Errorf("the placeholder was copied:\n%q", got)
	}
	if !strings.Contains(got, "$5") {
		t.Errorf("the prices were lost:\n%s", got)
	}
}

// Copy as Markdown finds the source of a paragraph with a formula in it.
func TestMarkdownCopyWithInlineMath(t *testing.T) {
	ed := New(doc.New("前 **粗** 与 $x^2$ 后\n\n第二段\n"), NewShared())
	ed.testMod = ui.Alt
	tt := ui.NewTester(ed.View, 900, 500)
	tt.Frame()
	tt.Frame()

	first := ed.tops[0] + 8
	tt.Press(60, first)
	tt.Move(600, ed.tops[1]+8)
	tt.Release(600, ed.tops[1]+8)
	tt.Frame()
	tt.RightClickAt(300, first)
	tt.Frame()
	if err := tt.ChooseMenuItem("复制为 Markdown"); err != nil {
		t.Fatalf("%v", err)
	}
	tt.Frame()
	got := tt.Clipboard()
	if !strings.Contains(got, "$x^2$") || !strings.Contains(got, "**粗**") {
		t.Errorf("the source is not what was copied:\n%s", got)
	}
}
