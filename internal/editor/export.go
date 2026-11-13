package editor

import (
	"image"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown/parser"

	"paperly/internal/doc"
)

// PageOptions says how an export's pages are shaped: the room a page gives
// its text, in DIPs, and how many picture pixels one DIP takes.
type PageOptions struct {
	Width, Height float32
	Scale         float32
}

// PageImage is a page drawn, and the blocks it holds.
type PageImage struct {
	Image   image.Image
	From    int
	To      int
	Heading []PageHeading
}

// PageHeading is a heading that stands on a page, for the PDF's bookmarks.
type PageHeading struct {
	Text  string
	Level int
}

// Exporter draws a document as pages, one picture each, without a window:
// the same blocks the page shows, at the width a page gives them, broken
// between blocks so that no line is cut in two.
//
// It lays the whole document out once, then draws a page at a time, so that
// a long document does not have to be drawn in one go. The drawing uses the
// software renderer, and must be asked for on the window's own thread—see
// mygo.RunOnMain; the PDF that holds the pages can be written anywhere.
type Exporter struct {
	e     *Editor
	opt   PageOptions
	spans []span
}

// span is the blocks one page holds.
type span struct{ from, to int }

// NewExporter reads the document and cuts it into pages. src is taken as it
// stands, so that the pages are of what the file holds while the window may
// go on changing it; dir is the folder its pictures are read from.
func NewExporter(src, dir string, sh *Shared, opt PageOptions) (*Exporter, error) {
	if opt.Scale <= 0 {
		opt.Scale = 2
	}
	s := *sh
	// A page is paper: it is drawn light, whatever the window wears, and
	// as wide as the page gives it room to be.
	s.Theme = "light"
	s.Folder = dir
	s.Measure = opt.Width
	e := New(doc.New(src), &s)
	n := e.doc.Len()
	if n == 0 {
		return &Exporter{e: e, opt: opt}, nil
	}

	// The whole document is laid out once in a short frame: every block
	// takes the room it needs, whether or not the frame can show it.
	tt := ui.NewTester(e.pageView(0, n), int(opt.Width), int(opt.Height))
	tt.Frame()
	tt.Frame()

	x := &Exporter{e: e, opt: opt}
	start := 0
	y := float32(0)
	for i := 0; i < n; i++ {
		h := e.heights[i]
		if h <= 0 {
			// A block that did not measure—a rule, say—still takes a line.
			h = e.sh.Size * 1.6
		}
		if y > 0 && y+h > opt.Height {
			x.spans = append(x.spans, span{start, i})
			start, y = i, 0
		}
		y += h
	}
	x.spans = append(x.spans, span{start, n})
	return x, nil
}

// Pages is how many the document makes.
func (x *Exporter) Pages() int { return len(x.spans) }

// Page draws the page at i.
func (x *Exporter) Page(i int) PageImage {
	sp := x.spans[i]
	tt := ui.NewTester(x.e.pageView(sp.from, sp.to), int(x.opt.Width), int(x.opt.Height))
	tt.SetScale(x.opt.Scale)
	tt.Frame()
	tt.Frame()
	p := PageImage{Image: tt.Image(), From: sp.from, To: sp.to}
	p.Heading = x.e.headingsIn(sp.from, sp.to)
	return p
}

// pageView draws the blocks from..to as one page: at the page's width, from
// its top, on paper.
func (e *Editor) pageView(from, to int) func(*ui.Context) {
	return func(c *ui.Context) {
		t := e.ensureTheme(c)
		e.loadImages()
		ui.Column(c).Fill().Background(t.Background).Children(func() {
			for i := from; i < to && i < len(e.doc.Blocks()); i++ {
				e.block(c, i)
			}
		})
	}
}

// headingsIn is the headings of the blocks from..to, as the page shows them:
// the text of the line, without the marks that make it a heading.
func (e *Editor) headingsIn(from, to int) []PageHeading {
	var out []PageHeading
	src := e.doc.Source()
	for i := from; i < to; i++ {
		b, ok := e.doc.Block(i)
		if !ok || b.Kind != parser.Heading {
			continue
		}
		text := strings.TrimSpace(src[b.Start:b.End])
		text = strings.TrimSpace(strings.TrimLeft(text, "#"))
		text = strings.TrimSpace(strings.TrimRight(text, "#"))
		out = append(out, PageHeading{Text: text, Level: b.Level})
	}
	return out
}
