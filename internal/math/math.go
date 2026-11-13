// Package math lays out the TeX of a formula and draws it as a picture:
// the subset of TeX that writing about code, physics or maths needs —
// letters and Greek, scripts, fractions, roots, sums and integrals, and
// fences that grow with what they hold — with the spacing TeX gives them.
//
// It draws into an image rather than into the page's text, so a formula
// looks the same in every window and at every scale, and the page's own
// fonts and layout stay out of it.
package math

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// Style is how a formula is set: a display of its own, as a paragraph of
// mathematics, or within a line of text, where it must not stand taller
// than the words around it.
type Style int

const (
	// Display sets a formula that has a line to itself: larger operators,
	// and their limits above and below them.
	Display Style = iota
	// Text sets a formula within a line of text: smaller operators, their
	// limits beside them, and the whole kept to the size of the text.
	Text
)

// Options says how a formula is drawn.
type Options struct {
	// Size is the em of the formula, in DIPs.
	Size float64
	// Style is how it is set: Display when zero.
	Style Style
	// Scale is how many pixels a DIP takes in the picture: 2 for a
	// display that a retina screen shows.
	Scale float64
	// Color paints the glyphs and the rules.
	Color color.Color
}

// Result is a formula drawn: its picture, and the room it takes on the
// page, in DIPs.
type Result struct {
	Image *image.RGBA
	// Width and Height are the room the picture takes, in DIPs.
	Width, Height float64
	// Ascent is how far it rises above the baseline, Descent how far it
	// sinks below it: a formula within a line of text is placed by these.
	Ascent, Descent float64
	Err             error
}

// Pad is the room left around the glyphs, in DIPs: italic letters and
// radicals reach past the boxes they are measured in.
const Pad = 1.0

// Render lays out tex and draws it. A formula it cannot read comes back
// with Err set, for the caller to show its source instead.
func Render(tex string, opt Options) Result {
	if opt.Size <= 0 {
		opt.Size = 16
	}
	if opt.Scale <= 0 {
		opt.Scale = 1
	}
	if opt.Color == nil {
		opt.Color = color.Black
	}
	n, err := parse(tex)
	if err != nil {
		return Result{Err: err}
	}
	// The layout works in DIPs, as the page does; paint scales them to the
	// pixels of the picture, once.
	l := &layout{em: opt.Size}
	b, err := l.render(n, opt.Style.teX())
	if err != nil {
		return Result{Err: err}
	}
	return paint(b, opt)
}

// paint draws a box's runs and rules into a picture, pad around it.
func paint(b box, opt Options) Result {
	const pad = Pad
	w := b.w + 2*pad
	h := b.h + b.d + 2*pad
	px := max(int(math.Ceil(w*opt.Scale)), 1)
	py := max(int(math.Ceil(h*opt.Scale)), 1)
	img := image.NewRGBA(image.Rect(0, 0, px, py))
	src := image.NewUniform(opt.Color)
	baseline := b.h + pad // from the top of the picture, in DIPs

	for _, r := range b.runs {
		f := face(r.style, r.size*opt.Scale)
		if r.cjk {
			f = cjkFace(r.size * opt.Scale)
		}
		if f == nil {
			continue
		}
		d := &font.Drawer{
			Dst:  img,
			Src:  src,
			Face: f,
			Dot: fixed.Point26_6{
				X: fixed.Int26_6(math.Round((pad + r.x) * opt.Scale * 64)),
				Y: fixed.Int26_6(math.Round((baseline - r.y) * opt.Scale * 64)),
			},
		}
		d.DrawString(r.text)
	}
	for _, r := range b.rules {
		x0 := int(math.Round((pad + r.x) * opt.Scale))
		x1 := int(math.Round((pad + r.x + r.w) * opt.Scale))
		y0 := int(math.Round((baseline - r.y - r.h) * opt.Scale))
		y1 := int(math.Round((baseline - r.y) * opt.Scale))
		rect := image.Rect(x0, y0, max(x1, x0+1), max(y1, y0+1))
		draw.Draw(img, rect, src, image.Point{}, draw.Over)
	}
	return Result{
		Image: img, Width: w, Height: h,
		Ascent: b.h + pad, Descent: b.d + pad,
	}
}

// teX is the style the layout sets: the two differ only in how large an
// operator is and where its limits go.
func (s Style) teX() style {
	if s == Text {
		return styleText
	}
	return styleDisplay
}
