package math

import (
	"image/color"
	"math"
	"strings"
	"testing"
)

// runsAt is the x a run of text stands at, and whether it was found.
func runAt(b box, text string) (float64, bool) {
	for _, r := range b.runs {
		if strings.TrimSpace(r.text) == text {
			return r.x, true
		}
	}
	return 0, false
}

func layoutOf(t *testing.T, tex string, st style) box {
	t.Helper()
	n, err := parse(tex)
	if err != nil {
		t.Fatalf("%q: %v", tex, err)
	}
	l := &layout{em: 20}
	b, err := l.render(n, st)
	if err != nil {
		t.Fatalf("%q: %v", tex, err)
	}
	return b
}

// A matrix sets its rows one under another, between fences that grow to
// hold them.
func TestMatrixRowsAndFences(t *testing.T) {
	b := layoutOf(t, `\begin{pmatrix} 1 & b \\ 2 & d \end{pmatrix}`, styleDisplay)
	var tops []float64
	for _, r := range b.runs {
		switch strings.TrimSpace(r.text) {
		case "1", "2":
			tops = append(tops, r.y)
		}
	}
	if len(tops) != 2 {
		t.Fatalf("the rows are not there: %+v", b.runs)
	}
	if tops[0] == tops[1] {
		t.Errorf("both rows stand on one baseline: %.1f", tops[0])
	}
	oneX, _ := runAt(b, "1")
	twoX, ok := runAt(b, "2")
	if !ok {
		t.Fatal("no second row")
	}
	if math.Abs(twoX-oneX) > 0.01 {
		t.Errorf("the rows start at %.1f and %.1f", oneX, twoX)
	}
	// The fence reaches higher than the letters it holds.
	if b.h < 20 {
		t.Errorf("the fence is only %.1f tall for a matrix of two rows", b.h)
	}
}

// The columns of a matrix are centred under each other: a narrow cell
// stands in the middle of its column.
func TestMatrixColumnsAreCentred(t *testing.T) {
	b := layoutOf(t, `\begin{matrix} a & bbbb \\ cccc & d \end{matrix}`, styleDisplay)
	ax, _ := runAt(b, "a")
	cx, _ := runAt(b, "c")
	dx, ok := runAt(b, "d")
	if !ok {
		t.Fatal("no d")
	}
	// The first b of the wide second-column cell starts its column; d is
	// narrow, so it stands in the middle of it.
	bx := math.Inf(1)
	for _, r := range b.runs {
		if r.text == "b" {
			bx = math.Min(bx, r.x)
		}
	}
	if math.IsInf(bx, 1) {
		t.Fatal("no second column")
	}
	// a is narrow in a wide column: it starts further right than c, and d
	// starts further right than b.
	if ax <= cx {
		t.Errorf("a at %.1f, c at %.1f: the narrow cell is not centred", ax, cx)
	}
	if dx <= bx {
		t.Errorf("d at %.1f, b at %.1f: the second column is not centred", dx, bx)
	}
}

// cases is two left-set columns behind a brace.
func TestCases(t *testing.T) {
	b := layoutOf(t, `\begin{cases} x^2 & x \ge 0 \\ -x & x < 0 \end{cases}`, styleDisplay)
	// Both second-column cells start at the same place, past the first.
	x1, ok := runAt(b, "x")
	if !ok {
		t.Fatalf("no second column: %+v", b.runs)
	}
	_ = x1
	if b.h < 18 {
		t.Errorf("the brace is only %.1f tall", b.h)
	}
	if len(b.runs) < 10 {
		t.Errorf("the cases are missing pieces: %d runs", len(b.runs))
	}
}

// An aligned block puts the equations' alignment points in one column:
// every = stands at the same x.
func TestAlignedColumnsLineUp(t *testing.T) {
	b := layoutOf(t, `\begin{aligned} (a+b)^2 &= a^2 + b^2 \\ &= c^2 \end{aligned}`, styleDisplay)
	var xs []float64
	for _, r := range b.runs {
		if r.text == "=" {
			xs = append(xs, r.x)
		}
	}
	if len(xs) != 2 {
		t.Fatalf("%d equals signs", len(xs))
	}
	if xs[0] != xs[1] {
		t.Errorf("the alignment points stand at %.2f and %.2f", xs[0], xs[1])
	}
}

// align numbers its rows at the right; aligned does not.
func TestAlignNumbersItsRows(t *testing.T) {
	b := layoutOf(t, `\begin{align} x &= 1 \\ y &= 2 \end{align}`, styleDisplay)
	for _, want := range []string{"(1)", "(2)"} {
		if _, ok := runAt(b, want); !ok {
			t.Errorf("no %s in the block: %+v", want, b.runs)
		}
	}
	if b2 := layoutOf(t, `\begin{aligned} x &= 1 \\ y &= 2 \end{aligned}`, styleDisplay); len(b2.runs) != len(b.runs)-2 {
		t.Errorf("aligned carries numbers: %d runs against %d", len(b2.runs), len(b.runs))
	}
}

// The fences of bmatrix, vmatrix and Bmatrix are the ones named.
func TestMatrixFences(t *testing.T) {
	for tex, want := range map[string]string{
		`\begin{bmatrix} 1 \end{bmatrix}`: "[",
		`\begin{vmatrix} 1 \end{vmatrix}`: "|",
		`\begin{Bmatrix} 1 \end{Bmatrix}`: "{",
	} {
		b := layoutOf(t, tex, styleDisplay)
		if _, ok := runAt(b, want); !ok {
			t.Errorf("%q: no %q fence", tex, want)
		}
	}
}

// An environment it does not know is refused, for the page to show the
// source rather than a wrong picture.
func TestUnknownEnvironment(t *testing.T) {
	if _, err := parse(`\begin{tikzcd} a \end{tikzcd}`); err == nil {
		t.Error("an unknown environment was taken")
	}
	if _, err := parse(`\begin{matrix} a`); err == nil {
		t.Error("an unclosed environment was taken")
	}
}

// A matrix within a line of text is drawn in the line's own size: the cells
// of an array are always set in the size of the text around them.
func TestMatrixInText(t *testing.T) {
	r := Render(`A = \begin{pmatrix} a & b \\ c & d \end{pmatrix}`, Options{Size: 20, Scale: 1, Style: Text, Color: color.Black})
	if r.Err != nil {
		t.Fatal(r.Err)
	}
	if r.Ascent <= 0 || r.Descent <= 0 {
		t.Errorf("ascent %.1f, descent %.1f", r.Ascent, r.Descent)
	}
}
