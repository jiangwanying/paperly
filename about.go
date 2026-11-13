package main

import (
	"bytes"
	_ "embed"
	"image"
	_ "image/png"
	"strings"
	"sync"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"paperly/locales"
)

// iconPNG is the app's icon, the one the window and the dock wear: the
// about panel shows it, so that the name has a face beside it.
//
//go:embed assets/icon.png
var iconPNG []byte

var (
	iconOnce sync.Once
	iconBmp  *ui.Bitmap
)

// appIcon is the icon as a bitmap, decoded once.
func appIcon() *ui.Bitmap {
	iconOnce.Do(func() {
		img, _, err := image.Decode(bytes.NewReader(iconPNG))
		if err != nil {
			return
		}
		iconBmp = ui.NewBitmap(img)
	})
	return iconBmp
}

// showAbout opens the panel that says what this app is, the way the
// platform's own about box would: the icon, the name, the version, and the
// three things one asks about it, in one row under it all.
func (a *App) showAbout(_ *mygo.MenuItem, win *mygo.Window) {
	if win == nil {
		win = mygo.FocusedWindow()
	}
	if a.about != nil && !a.about.IsDestroyed() {
		a.about.Show()
		a.about.Focus()
		return
	}
	w := mygo.NewWindow(mygo.WindowOptions{
		Title:             "Paperly",
		Width:             420,
		Height:            460,
		MinWidth:          380,
		MinHeight:         440,
		DisableResize:     true,
		DisableFullScreen: true,
		// It belongs to the window it was asked from: it opens over that
		// window, and follows it to another screen.
		Parent:  win,
		Content: ui.View(a.aboutView),
	})
	// A panel that says what the app is has no menu of its own: an empty
	// menu, since nil would mean the app's, as on Windows it does.
	w.SetMenu(mygo.NewMenu(nil))
	w.OnClosed(func() { a.about = nil })
	a.about = w
	w.Show()
}

// aboutView is the panel, as the platform's own about box has it: the icon
// over the name and its version, what the app is, where it lives, and a
// button for each thing one can do about it, full width under the rest.
func (a *App) aboutView(c *ui.Context) {
	t := a.shared.Look(c.Theme())
	ui.Column(c).Fill().AlignItems(ui.Center).Padding(28, 24).Gap(13).Children(func() {
		if bmp := appIcon(); bmp != nil {
			ui.Image(c, bmp).Width(68).Height(68)
		}
		ui.Text(c, "Paperly "+version).FontSize(20).Bold().TextColor(t.Text).
			FontFeatures("tnum").Padding(4, 0)
		ui.Text(c, locales.AboutDetail()).FontSize(14).TextColor(t.Text).
			Width(348).TextAlign(ui.Center)
		ui.Text(c, locales.GithubRepoLabel()+": "+strings.TrimPrefix(repoURL, "https://")).
			FontSize(13).TextColor(t.Text).Width(348).TextAlign(ui.Center)
		// The room the words leave goes under them, so that the buttons
		// stand at the foot of the panel, as a dialog's do.
		ui.Box(c).Grow(1)
		a.aboutButton(c, t, locales.OpenGithub(), true, func() { openURL(repoURL) })
		a.aboutButton(c, t, locales.CheckUpdates(), false, func() { checkForUpdates() })
		a.aboutButton(c, t, locales.Ok(), false, func() {
			if a.about != nil {
				a.about.Close()
			}
		})
	})
}

// aboutButton is one of the panel's buttons, as wide as the panel and set
// under the one before it, the first wearing the accent as a dialog's
// default button does.
func (a *App) aboutButton(c *ui.Context, t *ui.Theme, label string, primary bool, click func()) {
	b := ui.Row(c).AlignSelf(ui.Stretch).Height(38).Radius(8).
		Justify(ui.Center).OnClick(click)
	labelColor := t.Text
	if primary {
		b.Background(t.Accent)
		if b.Hovered() {
			b.Background(t.AccentHover)
		}
		labelColor = t.AccentText
	} else {
		b.Background(t.Surface)
		b.Border(1, t.Border)
		if b.Hovered() {
			b.Background(t.SurfaceHover)
		}
	}
	b.Children(func() { ui.Text(c, label).FontSize(14).TextColor(labelColor) })
}
