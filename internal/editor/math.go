package editor

import (
	"image/color"
	"strings"

	mathx "paperly/internal/math"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
)

// mathKey names a formula drawn: its TeX, the style and size it was set at,
// and the color it was painted.
type mathKey struct {
	tex   string
	style mathx.Style
	size  float32
	color ui.Color
}

// mathPic is a formula drawn, and the room it takes on the page: its width,
// how far it rises above the baseline and how far it sinks below it.
type mathPic struct {
	bmp       *ui.Bitmap
	w, asc    float32
	desc      float32
	totalTall float32
}

// mathBlock draws a display formula where the page stands: centered, at
// the size of the text around it, in the color of the text.
func (e *Editor) mathBlock(c *ui.Context, tex string) {
	t := c.Theme()
	// A display is set a little larger than the text it stands among, as
	// TeX sets it: the formula is the paragraph's subject.
	size := float32(e.sh.Size) * 1.35
	row := ui.Row(c).AlignSelf(ui.Stretch).Justify(ui.Center).Padding(8, 0)
	row.Children(func() {
		if strings.TrimSpace(tex) == "" {
			// A formula with nothing in it yet says where it is.
			ui.Text(c, "$$…$$").FontSize(float32(e.sh.Size) * 0.9).TextColor(t.TextMuted)
			return
		}
		if p, err := e.mathPicture(tex, size, t.Text, mathx.Display); err == nil {
			ui.Image(c, p.bmp).Width(p.w).Height(p.totalTall)
			return
		}
		// A formula the page cannot read shows as its source, dimmed, so
		// that what was written is never lost from sight.
		ui.Text(c, tex).FontSize(float32(e.sh.Size) * 0.9).TextColor(t.TextMuted)
	})
}

// inlineMath sets a formula within a line of text: an object of the text,
// at the size of the text around it, placed by its height above the
// baseline and its depth below it. It is asked again whenever the
// paragraph changes, so what it makes is kept.
func (e *Editor) inlineMath(tex string, size float32) (markdown.InlineObject, bool) {
	if strings.TrimSpace(tex) == "" || size <= 0 {
		return markdown.InlineObject{}, false
	}
	// A shade larger than the words around it: the letters of the math fonts
	// are smaller than the Chinese the page is mostly set in, and a formula
	// set at the text's own size reads as a footnote.
	p, err := e.mathPicture(tex, size*1.1, e.mathText, mathx.Text)
	if err != nil {
		// The TeX itself shows, which is better than nothing.
		return markdown.InlineObject{}, false
	}
	return markdown.InlineObject{
		Width:  p.w,
		Height: p.asc,
		Depth:  p.desc,
		Draw: func(pn *ui.Painter, r ui.Rect) {
			pn.Image(p.bmp, r, ui.FillBox)
		},
	}, true
}

// mathPicture is the picture of a formula, drawn once and kept: the frames
// that follow lay the same formula out again and again.
func (e *Editor) mathPicture(tex string, size float32, col ui.Color, st mathx.Style) (mathPic, error) {
	key := mathKey{tex: tex, style: st, size: size, color: col}
	if p, ok := e.mathCache[key]; ok {
		return p, nil
	}
	r := mathx.Render(tex, mathx.Options{
		Size:  float64(size),
		Style: st,
		Scale: 2,
		Color: color.RGBA{R: col.R, G: col.G, B: col.B, A: 255},
	})
	if r.Err != nil {
		return mathPic{}, r.Err
	}
	p := mathPic{
		bmp:       ui.NewBitmap(r.Image),
		w:         float32(r.Width),
		asc:       float32(r.Ascent),
		desc:      float32(r.Descent),
		totalTall: float32(r.Height),
	}
	if e.mathCache == nil || len(e.mathCache) > 128 {
		e.mathCache = map[mathKey]mathPic{}
	}
	e.mathCache[key] = p
	return p, nil
}
