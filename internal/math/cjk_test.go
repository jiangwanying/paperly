package math

import (
	"image/color"
	"strings"
	"testing"
)

// Chinese in \text{} is drawn in the Chinese face, sized like the words
// around it: the formula needs no switch of language to hold a word.
func TestChineseInText(t *testing.T) {
	b := layoutOf(t, `\text{当 } x > 0`, styleDisplay)
	var cjk int
	for _, r := range b.runs {
		if r.text == "当" {
			if !r.cjk {
				t.Errorf("the character was set in the Latin face")
			}
			cjk++
		}
	}
	if cjk != 1 {
		t.Fatalf("the character is not there: %+v", b.runs)
	}
	if b.h < 14 {
		t.Errorf("the line is only %.1f tall with a character in it", b.h)
	}
	// The character takes a whole em, and Latin follows it.
	if b.w < 20 {
		t.Errorf("the formula is %.1f wide", b.w)
	}
}

// A piecewise function written in Chinese renders whole.
func TestChineseCases(t *testing.T) {
	r := Render(`f(x) = \begin{cases} x^2 & \text{当 } x \ge 0 \\ -x & \text{当 } x < 0 \end{cases}`,
		Options{Size: 20, Scale: 2, Style: Display, Color: color.Black})
	if r.Err != nil {
		t.Fatal(r.Err)
	}
	if r.Width <= 0 || r.Height <= 0 {
		t.Fatalf("nothing was drawn: %v x %v", r.Width, r.Height)
	}
}

// The letters of the number sets are the blackboard bold ones the math font
// carries.
func TestBlackboardSets(t *testing.T) {
	b := layoutOf(t, `\mathbb{R}`, styleDisplay)
	if len(b.runs) == 0 || !strings.Contains(b.runs[0].text, "ℝ") {
		t.Errorf("R is not the blackboard R: %+v", b.runs)
	}
}

// Chinese that the subset does not carry still leaves a box of its own, so
// the formula around it keeps its shape.
func TestMissingChineseKeepsItsRoom(t *testing.T) {
	if cjkHas('𠀋') {
		t.Skip("the face carries the character after all")
	}
	b := layoutOf(t, `a\text{𠀋}b`, styleDisplay)
	if b.w <= 0 {
		t.Errorf("the formula is %.1f wide", b.w)
	}
}
