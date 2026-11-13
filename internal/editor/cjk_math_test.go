package editor

import (
	"image/png"
	"os"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"paperly/internal/doc"
)

const cjkMathBody = `## 分段函数

$$
f(x) = \begin{cases}
x^2 & \text{当 } x \ge 0 \\
-x  & \text{当 } x < 0
\end{cases}
$$

速度 $v = 3\ \text{米/秒}$，其中 $n \in \mathbb{N}$。
`

// The words of a formula are set in Chinese when they are written so.
func TestChineseInFormulaIsDrawn(t *testing.T) {
	ed := New(doc.New(cjkMathBody), NewShared())
	tt := ui.NewTester(ed.View, 900, 560)
	tt.Frame()
	tt.Frame()

	for _, text := range tt.Texts() {
		if strings.Contains(text, "begin{cases}") || strings.Contains(text, "米/秒") {
			t.Errorf("the source shows on the page: %q", text)
		}
	}
	// The piecewise function, the speed, and n in N.
	if len(ed.mathCache) != 3 {
		t.Errorf("drew %d formulas, want 3", len(ed.mathCache))
	}
	f, err := os.Create("/tmp/cjk-page.png")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, tt.Image()); err != nil {
		t.Fatal(err)
	}
}
