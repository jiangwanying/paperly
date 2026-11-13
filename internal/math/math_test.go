package math

import (
	"image/color"
	"math"
	"testing"
)

// Formulas that the renderer must lay out: the shapes a document about
// code, physics or maths asks for.
var samples = []string{
	`\int_0^1 x^2\,dx = \frac{1}{3}`,
	`E = mc^2`,
	`\sum_{n=1}^{\infty}\frac{1}{n^2} = \frac{\pi^2}{6}`,
	`\sqrt{\frac{a+b}{2\pi}}\cos\omega`,
	`\left(\frac{a}{b}\right)^{n+1}`,
	`\alpha + \beta \leq \gamma \times \delta`,
	`\overline{AB} \perp \overline{CD}`,
	`\text{if } x > 0 \text{ then } y = \sqrt{x}`,
}

func TestRenderSamples(t *testing.T) {
	for _, tex := range samples {
		r := Render(tex, Options{Size: 20, Scale: 2, Color: color.Black})
		if r.Err != nil {
			t.Errorf("%q: %v", tex, r.Err)
			continue
		}
		if r.Width <= 0 || r.Height <= 0 {
			t.Errorf("%q: %vx%v DIPs", tex, r.Width, r.Height)
		}
		if r.Image == nil || r.Image.Bounds().Dx() == 0 {
			t.Errorf("%q: no picture", tex)
		}
	}
}

// A formula it cannot read says so, for the page to show its source.
func TestUnknownCommand(t *testing.T) {
	if r := Render(`\begin{tikzcd} a \end{tikzcd}`, Options{Size: 16}); r.Err == nil {
		t.Error("an environment it does not know was taken")
	}
	if r := Render(`x +`, Options{Size: 16}); r.Err != nil {
		t.Errorf("a formula that ends early: %v", r.Err)
	}
}

// The fences a formula may be written with are not part of it.
func TestFences(t *testing.T) {
	plain := Render(`x^2`, Options{Size: 16, Scale: 1})
	for _, tex := range []string{`$x^2$`, `$$x^2$$`, `\(x^2\)`, `\[x^2\]`} {
		r := Render(tex, Options{Size: 16, Scale: 1})
		if r.Err != nil {
			t.Fatalf("%q: %v", tex, r.Err)
		}
		if math.Abs(r.Width-plain.Width) > 0.01 {
			t.Errorf("%q is %.2f wide, %q is %.2f", tex, r.Width, "x^2", plain.Width)
		}
	}
}

// A superscript stands above the baseline, a subscript below it.
func TestScriptsStandAboveAndBelow(t *testing.T) {
	n, err := parse(`x_i^2`)
	if err != nil {
		t.Fatal(err)
	}
	l := &layout{em: 20}
	b, err := l.render(n, styleDisplay)
	if err != nil {
		t.Fatal(err)
	}
	var sup, sub bool
	for _, r := range b.runs {
		switch r.text {
		case "2":
			sup = r.y > 0
		case "i":
			sub = r.y < 0
		}
	}
	if !sup || !sub {
		t.Errorf("the scripts are not above and below: %+v", b.runs)
	}
	if b.h <= 0 || b.d <= 0 {
		t.Errorf("the box has no height or no depth: %.1f %.1f", b.h, b.d)
	}
}

// The picture is drawn in the color it is asked for.
func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func TestColor(t *testing.T) {
	r := Render(`x`, Options{Size: 20, Scale: 1, Color: color.RGBA{R: 200, G: 30, B: 40, A: 255}})
	if r.Err != nil {
		t.Fatal(r.Err)
	}
	found := false
	for y := 0; y < r.Image.Bounds().Dy() && !found; y++ {
		for x := 0; x < r.Image.Bounds().Dx(); x++ {
			n := color.NRGBAModel.Convert(r.Image.At(x, y)).(color.NRGBA)
			if n.A > 200 && abs(int(n.R)-200) < 3 && abs(int(n.G)-30) < 3 && abs(int(n.B)-40) < 3 {
				found = true
				break
			}
		}
	}
	if !found {
		t.Error("the glyphs are not the color asked for")
	}
}
