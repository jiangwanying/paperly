package editor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"

	"paperly/internal/doc"
)

// Clicking a link of the reading view follows it without entering the
// writing of the block, as a document on the web does.
func TestLinkClickFollows(t *testing.T) {
	ed := New(doc.New("[跳转](#标题)\n\n# 标题\n\n正文"), NewShared())
	ed.sh.Side = SideNone
	tt := ui.NewTester(ed.View, 700, 400)
	tt.Frame()
	tt.Frame()

	// The link sits at the top of the page: a few points along its line,
	// one of them is on the words of the link.
	fired := false
	for _, p := range [][2]float32{{50, 30}, {70, 30}, {90, 30}, {110, 30}, {50, 45}, {70, 45}, {90, 45}} {
		e2 := New(doc.New("[跳转](#标题)\n\n# 标题\n\n正文"), NewShared())
		e2.sh.Side = SideNone
		t2 := ui.NewTester(e2.View, 700, 400)
		t2.Frame()
		t2.Frame()
		t2.ClickAt(p[0], p[1])
		t2.Frame()
		if e2.md.Anchor == "标题" {
			fired = true
			if e2.active != -1 {
				t.Fatalf("the click at %v entered the writing: %d", p, e2.active)
			}
			break
		}
	}
	if !fired {
		t.Fatal("no point on the line of the link followed it")
	}
}

// soloImage takes a block that is one picture and nothing else.
func TestSoloImage(t *testing.T) {
	if addr, alt, ok := soloImage("![截图](assets/a%20b.png)"); !ok || addr != "assets/a%20b.png" || alt != "截图" {
		t.Errorf("the picture block is %q %q %v", addr, alt, ok)
	}
	if _, _, ok := soloImage("![图](a.png) 还有文字"); ok {
		t.Errorf("a paragraph with words is not a picture")
	}
	if _, _, ok := soloImage("# 标题"); ok {
		t.Errorf("a heading is not a picture")
	}
}

// A click on a picture of its own opens it large over the page.
func TestImageZoom(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "图画.png")
	if b, err := os.ReadFile("testdata/shots/shot-search.png"); err == nil {
		os.WriteFile(p, b, 0o644)
	}
	ed := New(doc.New("![](图画.png)\n\n正文"), NewShared())
	ed.sh.Folder = dir
	ed.sh.Side = SideNone
	tt := ui.NewTester(ed.View, 700, 400)
	tt.Frame()
	tt.Frame()
	if ed.zoomIsOn() {
		t.Fatal("the veil shows before any click")
	}
	// The picture sits at the top of the page: the click points walk its
	// box until one opens the veil.
	fired := false
	for _, y := range []float32{20, 30, 40, 55, 70, 90, 120} {
		e2 := New(doc.New("![](图画.png)\n\n正文"), NewShared())
		e2.sh.Folder = dir
		e2.sh.Side = SideNone
		t2 := ui.NewTester(e2.View, 700, 400)
		t2.Frame()
		t2.Frame()
		t2.ClickAt(300, y)
		t2.Frame()
		if e2.zoomIsOn() {
			fired = true
			if e2.active != -1 {
				t.Fatalf("the click at y%v entered the writing: %d", y, e2.active)
			}
			break
		}
	}
	if !fired {
		t.Fatal("no click on the picture opened it")
	}
}
