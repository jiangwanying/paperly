package math

import (
	_ "embed"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
)

// cjkTTF is Noto Serif SC, cut down to the characters Chinese writing
// needs—GB 2312's six thousand odd, the punctuation, and the Latin the
// formulas already have—so that \text{当 } can be set in the formula
// without carrying a whole typeface. Noto is under the SIL Open Font
// License, which lets it travel with the app; the license itself is beside
// it, in fonts/OFL.txt.
//
//go:embed fonts/NotoSerifSC-Regular.ttf
var cjkTTF []byte

var (
	cjkOnce sync.Once
	cjkFont *sfnt.Font

	cjkMu    sync.Mutex
	cjkFaces = map[float64]font.Face{}
)

// loadCJK parses the Chinese face once, and says whether it can be used.
func loadCJK() *sfnt.Font {
	cjkOnce.Do(func() {
		if len(cjkTTF) == 0 {
			return
		}
		f, err := opentype.Parse(cjkTTF)
		if err == nil {
			cjkFont = f
		}
	})
	return cjkFont
}

// cjkFace is the Chinese face at size pixels, made once and kept.
func cjkFace(size float64) font.Face {
	f := loadCJK()
	if f == nil {
		return nil
	}
	cjkMu.Lock()
	defer cjkMu.Unlock()
	if fc, ok := cjkFaces[size]; ok {
		return fc
	}
	fc, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size: size, DPI: 72, Hinting: font.HintingNone,
	})
	if err != nil {
		fc = nil
	}
	cjkFaces[size] = fc
	return fc
}

// cjkHas reports whether the Chinese face can draw r.
func cjkHas(r rune) bool {
	f := loadCJK()
	if f == nil {
		return false
	}
	idx, err := f.GlyphIndex(nil, r)
	return err == nil && idx != 0
}
