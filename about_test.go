package main

import (
	"image/png"
	"os"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"paperly/locales"
)

// The about panel shows the icon, the name, the version and the three
// things one asks of it, each a button of its own.
func TestAboutView(t *testing.T) {
	a := NewApp(DefaultSettings(), "")
	tt := ui.NewTester(a.aboutView, 420, 460)
	tt.Frame()
	tt.Frame()

	if appIcon() == nil {
		t.Fatal("the icon did not load")
	}
	all := strings.Join(tt.Texts(), "\n")
	for _, want := range []string{"Paperly " + version, locales.AboutDetail(), "github.com/jiangwanying/paperly", locales.OpenGithub(), locales.CheckUpdates(), locales.Ok()} {
		if !strings.Contains(all, want) {
			t.Errorf("%q is not on the panel:\n%s", want, all)
		}
	}

	f, err := os.Create("/tmp/about.png")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, tt.Image()); err != nil {
		t.Fatal(err)
	}
}

// The buttons stand one under another, as wide as the panel and centred in
// it, the first wearing the accent.
func TestAboutButtonsStack(t *testing.T) {
	a := NewApp(DefaultSettings(), "")
	tt := ui.NewTester(a.aboutView, 420, 460)
	tt.Frame()
	tt.Frame()

	var rects []ui.Rect
	for _, label := range []string{locales.OpenGithub(), locales.CheckUpdates(), locales.Ok()} {
		r, ok := tt.Find(label)
		if !ok {
			t.Fatalf("%q is not there", label)
		}
		rects = append(rects, r)
	}
	if rects[1].Y <= rects[0].Y || rects[2].Y <= rects[1].Y {
		t.Errorf("the buttons are not stacked: %v", rects)
	}
	for _, r := range rects {
		if mid := r.X + r.W/2; abs(mid-210) > 2 {
			t.Errorf("a button stands at %.1f, the panel centres at 210", mid)
		}
	}
	first := rects[1].Y - rects[0].Y
	second := rects[2].Y - rects[1].Y
	if abs(first-second) > 1 {
		t.Errorf("the rows are not evenly set: %.1f and %.1f", first, second)
	}
}

func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
