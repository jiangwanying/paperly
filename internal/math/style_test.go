package math

import (
	"image/color"
	"testing"
)

// A formula within a line of text keeps its limits beside the operator and
// stands no taller than the line; on a line of its own it puts them above
// and below, and is larger.
func TestStylesSetTheLimitsDifferently(t *testing.T) {
	const tex = `\sum_{n=1}^{\infty} x`
	display := Render(tex, Options{Size: 16, Scale: 1, Style: Display, Color: color.Black})
	text := Render(tex, Options{Size: 16, Scale: 1, Style: Text, Color: color.Black})
	if display.Err != nil || text.Err != nil {
		t.Fatalf("%v %v", display.Err, text.Err)
	}
	if text.Width <= display.Width {
		t.Errorf("the text style is %.1f wide, the display %.1f: the limits are not beside it", text.Width, display.Width)
	}
	if text.Ascent >= display.Ascent {
		t.Errorf("the text style rises %.1f, the display %.1f: the limits are still above it", text.Ascent, display.Ascent)
	}
	if text.Height >= display.Height {
		t.Errorf("the text style is %.1f tall, the display %.1f", text.Height, display.Height)
	}
}

// The picture says how far it rises above the baseline and how far it sinks
// below it, so that a line of text can be laid out around it.
func TestAscentAndDescent(t *testing.T) {
	r := Render(`\frac{a}{b}`, Options{Size: 20, Scale: 1, Style: Text, Color: color.Black})
	if r.Err != nil {
		t.Fatal(r.Err)
	}
	if r.Ascent <= 0 || r.Descent <= 0 {
		t.Fatalf("ascent %.1f descent %.1f", r.Ascent, r.Descent)
	}
	if got := r.Ascent + r.Descent; got < r.Height-0.01 || got > r.Height+0.01 {
		t.Errorf("ascent+descent = %.2f, the picture is %.2f tall", got, r.Height)
	}
}
