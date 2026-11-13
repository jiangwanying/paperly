package math

import (
	"fmt"
	"strings"
	"sync"

	"github.com/go-fonts/latin-modern/lmmath"
	"github.com/go-fonts/latin-modern/lmroman12bold"
	"github.com/go-fonts/latin-modern/lmroman12italic"
	"github.com/go-fonts/latin-modern/lmroman12regular"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
)

// faceStyle picks the font a run of text is drawn in.
type faceStyle uint8

const (
	fsRoman  faceStyle = iota // digits, operators, \mathrm and \text
	fsItalic                  // variables
	fsBold
	fsMath // symbols, Greek, big operators, radicals
)

type faceKey struct {
	style faceStyle
	size  float64 // pixels
}

var (
	fontsOnce sync.Once
	fontsErr  error
	fontSet   [4]*sfnt.Font

	facesMu sync.Mutex
	faces   = map[faceKey]font.Face{}
	metrics = map[faceKey]font.Metrics{}
)

func loadFonts() error {
	fontsOnce.Do(func() {
		parse := func(b []byte) *sfnt.Font {
			f, err := opentype.Parse(b)
			if err != nil && fontsErr == nil {
				fontsErr = err
			}
			return f
		}
		fontSet[fsRoman] = parse(lmroman12regular.TTF)
		fontSet[fsItalic] = parse(lmroman12italic.TTF)
		fontSet[fsBold] = parse(lmroman12bold.TTF)
		fontSet[fsMath] = parse(lmmath.TTF)
	})
	return fontsErr
}

// face is the font at size pixels, made once and kept: a formula asks for
// the same few sizes over and over.
func face(st faceStyle, size float64) font.Face {
	if err := loadFonts(); err != nil {
		return nil
	}
	k := faceKey{st, size}
	facesMu.Lock()
	defer facesMu.Unlock()
	if f, ok := faces[k]; ok {
		return f
	}
	f, err := opentype.NewFace(fontSet[st], &opentype.FaceOptions{
		Size: size, DPI: 72, Hinting: font.HintingNone,
	})
	if err != nil {
		f = nil
	}
	faces[k] = f
	if f != nil {
		metrics[k] = f.Metrics()
	}
	return f
}

// width is how wide s is at size pixels, in pixels.
func width(st faceStyle, size float64, s string) float64 {
	f := face(st, size)
	if f == nil {
		return 0
	}
	return float64(font.MeasureString(f, s)) / 64
}

// xheight is the height of a small x at size pixels, and axis the height
// of the bar of a fraction: the line a formula stands on.
func xheight(size float64) float64 {
	f := face(fsRoman, size)
	if f == nil {
		return size * 0.45
	}
	facesMu.Lock()
	m := metrics[faceKey{fsRoman, size}]
	facesMu.Unlock()
	if m.XHeight <= 0 {
		return size * 0.45
	}
	return float64(m.XHeight) / 64
}

// hasRune reports whether the math font can draw r.
func hasRune(st faceStyle, r rune) bool {
	if err := loadFonts(); err != nil {
		return false
	}
	f := fontSet[st]
	if f == nil {
		return false
	}
	idx, err := f.GlyphIndex(nil, r)
	return err == nil && idx != 0
}

// textOf trims the fences a formula is written with.
func textOf(tex string) string {
	t := strings.TrimSpace(tex)
	switch {
	case strings.HasPrefix(t, "$$") && strings.HasSuffix(t, "$$") && len(t) >= 4:
		return strings.TrimSpace(t[2 : len(t)-2])
	case strings.HasPrefix(t, `\[`) && strings.HasSuffix(t, `\]`):
		return strings.TrimSpace(t[2 : len(t)-2])
	case strings.HasPrefix(t, `\(`) && strings.HasSuffix(t, `\)`):
		return strings.TrimSpace(t[2 : len(t)-2])
	case strings.HasPrefix(t, "$") && strings.HasSuffix(t, "$") && len(t) >= 2:
		return strings.TrimSpace(t[1 : len(t)-1])
	}
	return t
}

var errUnknown = fmt.Errorf("math: unknown command")
