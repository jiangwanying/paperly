package editor

import (
	"image/png"
	"os"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown/parser"

	"paperly/internal/doc"
)

const envBody = `## 线性代数

矩阵与行列式：

$$
A = \begin{pmatrix} a & b \\ c & d \end{pmatrix}, \qquad
\det A = \begin{vmatrix} a & b \\ c & d \end{vmatrix}
$$

分段函数：

$$
f(x) = \begin{cases}
x^2 & x \ge 0 \\
-x  & x < 0
\end{cases}
$$

推导：

$$
\begin{aligned}
(a+b)^2 &= a^2 + 2ab + b^2 \\
        &= a^2 + 2ab + b^2
\end{aligned}
$$
`

// Matrices, cases and aligned blocks are drawn, not shown as their TeX.
func TestEnvironmentsAreDrawn(t *testing.T) {
	ed := New(doc.New(envBody), NewShared())
	tt := ui.NewTester(ed.View, 900, 720)
	tt.Frame()
	tt.Frame()

	for _, text := range tt.Texts() {
		if strings.Contains(text, "begin{") {
			t.Errorf("the TeX shows on the page: %q", text)
		}
	}
	if len(ed.mathCache) != 3 {
		t.Errorf("drew %d formulas, want 3", len(ed.mathCache))
	}
	f, err := os.Create("/tmp/env-page.png")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, tt.Image()); err != nil {
		t.Fatal(err)
	}
}

// A formula with an environment in it still opens as its source and draws
// again when the caret leaves.
func TestEnvironmentEditsAsSource(t *testing.T) {
	ed := New(doc.New(envBody), NewShared())
	tt := ui.NewTester(ed.View, 900, 720)
	tt.Frame()
	tt.Frame()

	// The first formula block holds the matrix.
	which := -1
	for i := range ed.doc.Blocks() {
		if b, _ := ed.doc.Block(i); b.Kind == parser.MathBlock {
			which = i
			break
		}
	}
	if which < 0 {
		t.Fatal("no formula block")
	}
	tt.ClickAt(300, ed.tops[which]+ed.heights[which]/2)
	tt.Frame()
	if ed.Active() != which {
		t.Fatalf("the click opened block %d, want %d", ed.Active(), which)
	}
	if got := ed.Draft(); !strings.Contains(got, `\begin{pmatrix}`) {
		t.Errorf("the source is %q", got)
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if len(ed.mathCache) != 3 {
		t.Errorf("after leaving, %d formulas are drawn", len(ed.mathCache))
	}
}
