package main

import (
	"bytes"
	"image/png"
	"os"
	"testing"

	"github.com/egoist/mygo/ui"
)

// The preferences window shows the language, the size and the panel's width.
func TestSettingsView(t *testing.T) {
	a := NewApp(DefaultSettings(), "")
	tt := ui.NewTester(a.settingsView, 780, 540)
	tt.Frame()
	tt.Frame()

	if a.shared.Size != 16 {
		t.Fatalf("the text starts at %v, want 16", a.shared.Size)
	}
	// One click on + makes the text bigger, and remembers it.
	a.setFontSize(17)
	if a.shared.Size != 17 || a.set.FontSize != 17 {
		t.Errorf("size %v, setting %v, want 17", a.shared.Size, a.set.FontSize)
	}
	tt.Frame()

	// The other settings change through their setters too.
	a.setLineHeight(1.8)
	if a.shared.LineHeight != 1.8 || a.set.LineHeight != 1.8 {
		t.Errorf("line height %v / %v, want 1.8", a.shared.LineHeight, a.set.LineHeight)
	}
	a.setMeasure(720)
	if a.shared.Measure != 720 || a.set.Measure != 720 {
		t.Errorf("measure %v / %v, want 720", a.shared.Measure, a.set.Measure)
	}
	if a.shared.ShowStatusBar != (a.set.ShowStatusBar == nil || *a.set.ShowStatusBar) {
		t.Errorf("the status bar and its setting disagree")
	}

	// Back to what a first run starts with.
	a.resetSettings()
	if a.shared.Size != 16 || a.shared.SideWidth != 270 || a.shared.Measure != 860 {
		t.Errorf("after a reset: size %v, panel %v, column %v",
			a.shared.Size, a.shared.SideWidth, a.shared.Measure)
	}
	if a.shared.LineHeight != 1.6 {
		t.Errorf("line height %v", a.shared.LineHeight)
	}
	if a.set.Language != "zh-Hans" || !a.shared.ShowStatusBar {
		t.Errorf("language %q, status bar %v", a.set.Language, a.shared.ShowStatusBar)
	}

	// The categories switch, and the search narrows them.
	a.prefsTab = "editor"
	tt.Frame()
	if !a.tabMatches(prefsTab{id: "editor"}) {
		t.Errorf("编辑器 does not match its own fields")
	}
	a.prefsFilter = "字号"
	tt.Frame()
	if a.tabMatches(prefsTab{id: "file"}) {
		t.Errorf("文件 matches 字号")
	}
	if !a.tabMatches(prefsTab{id: "editor"}) {
		t.Errorf("编辑器 does not match 字号")
	}
	a.prefsFilter = ""

	// The appearance pane renders, and its toggle flips the status bar.
	a.prefsTab = "appearance"
	tt.Frame()
	on := true
	a.set.ShowStatusBar = &on
	a.shared.ShowStatusBar = true
	off := false
	a.set.ShowStatusBar = &off
	a.shared.ShowStatusBar = false
	tt.Frame()
	a.set.ShowStatusBar = &on
	a.shared.ShowStatusBar = true
	tt.Frame()

	if os.Getenv("MD_UPDATE_SHOTS") != "" {
		var buf bytes.Buffer
		if err := png.Encode(&buf, tt.Image()); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile("testdata/settings.png", buf.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
		// The custom theme, with its colors.
		a.setTheme("custom")
		tt.Frame()
		tt.Frame()
		buf.Reset()
		if err := png.Encode(&buf, tt.Image()); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile("testdata/settings-custom.png", buf.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
		a.setTheme("auto")
	}
}
