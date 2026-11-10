package main

import (
	"fmt"
	"paperly/locales"
	"strconv"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"paperly/internal/editor"
)

// The icons of the preferences' categories, in the stroke style the rest of
// the interface draws with.
func prefIcon(shapes string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">` + shapes + `</svg>`))
}

var (
	iconFile    = prefIcon(`<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><path d="M14 2v6h6"/>`)
	iconPen     = prefIcon(`<path d="M17 3a2.85 2.83 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5z"/>`)
	iconImage   = prefIcon(`<rect x="3" y="3" width="18" height="18" rx="2"/><circle cx="8.5" cy="8.5" r="1.5"/><path d="m21 15-5-5L5 21"/>`)
	iconMark    = prefIcon(`<rect x="2.5" y="5" width="19" height="14" rx="2"/><path d="M6 15V9l2.5 3L11 9v6M15.5 9v6M13.5 13l2 2 2-2"/>`)
	iconPalette = prefIcon(`<circle cx="12" cy="12" r="9"/><circle cx="8.5" cy="10.5" r="0.8"/><circle cx="12" cy="8" r="0.8"/><circle cx="15.5" cy="10.5" r="0.8"/>`)
	iconSliders = prefIcon(`<path d="M4 21v-7M4 10V3M12 21v-9M12 8V3M20 21v-5M20 12V3M1 14h6M9 8h6M17 16h6"/>`)
)

// prefsTab is one category at the left of the preferences window, as
// Typora's are.
type prefsTab struct {
	id    string
	label string
	icon  *ui.SVG
}

func (a *App) prefsTabs() []prefsTab {
	return []prefsTab{
		{"file", locales.PrefsFile(), iconFile},
		{"editor", locales.PrefsEditor(), iconPen},
		{"image", locales.PrefsImage(), iconImage},
		{"markdown", locales.PrefsMarkdown(), iconMark},
		{"appearance", locales.PrefsAppearance(), iconPalette},
		{"general", locales.PrefsGeneral(), iconSliders},
	}
}

// OpenSettings shows the preferences window from the window a menu choice
// came from, or brings back the one that is already open.
func (a *App) OpenSettings(win *mygo.Window) {
	if win == nil {
		win = mygo.FocusedWindow()
	}
	if a.prefs != nil && !a.prefs.IsDestroyed() {
		// The same window asked again: bring the panel back. Another one
		// did: reopen the panel over it, so that it follows the window the
		// user is in, rather than pulling them to where the panel was.
		if a.prefsParent == win || win == nil {
			a.prefs.Show()
			a.prefs.Focus()
			return
		}
		a.prefs.Close()
	}
	a.prefsFilter = ""
	w := mygo.NewWindow(mygo.WindowOptions{
		Title: locales.Settings(), Width: 800, Height: 600,
		MinWidth: 660, MinHeight: 460,
		// A preferences window never takes the whole screen, and belongs to
		// the window it was asked from: it centers over that window, follows
		// it into a full-screen space, and never pulls the user to another
		// one.
		DisableFullScreen: true,
		Parent:            win,
		Content:           ui.View(a.settingsView),
	})
	w.OnClosed(func() { a.prefs = nil })
	a.prefs = w
	a.prefsParent = win
	w.Show()
}

// settingsView is the preferences window, as Typora's: the categories at the
// left under a search field, what they hold at the right.
func (a *App) settingsView(c *ui.Context) {
	t := a.shared.Look(c.Theme())
	ui.Row(c).Fill().AlignItems(ui.Stretch).Background(t.Background).Children(func() {
		ui.Column(c).Width(176).Background(t.Surface).Children(func() {
			ui.Box(c).Height(16)
			ui.Row(c).Padding(0, 14).Children(func() {
				ui.Themed(c, t).Children(func() {
					in := ui.TextInput(c.Key("prefs.search"), &a.prefsFilter).Grow(1)
					in.FontSize(13)
					in.Background(t.Surface)
					in.Border(1, t.Border)
				})
			})
			ui.Box(c).Height(10)
			shown := false
			for _, tab := range a.prefsTabs() {
				if !a.tabMatches(tab) {
					continue
				}
				shown = true
				tab := tab
				on := tab.id == a.prefsTab
				row := ui.Row(c).Padding(8, 14).Gap(10).AlignItems(ui.Center)
				switch {
				case on:
					row.Background(t.Text.Alpha(0.08))
				case row.Hovered():
					row.Background(t.SurfaceHover)
				}
				row.OnClick(func() { a.prefsTab = tab.id })
				row.Children(func() {
					ui.Icon(c, tab.icon).FontSize(16).TextColor(t.TextMuted)
					el := ui.Text(c, tab.label).FontSize(13).TextColor(t.Text)
					if on {
						el.Bold()
					}
				})
			}
			if !shown {
				ui.Text(c, locales.PrefsSearchNone()).FontSize(12).
					TextColor(t.TextMuted).Padding(6, 14)
			}
			ui.Box(c).Grow(1)
		})
		// The line between the categories and the settings, as the sidebar's
		// border is.
		ui.Box(c).Width(1).Background(t.Border.Alpha(0.45))
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c).Grow(1).Padding(26, 30).Gap(22).Children(func() {
				a.settingsPane(c, t)
			})
		})
	})
}

// tabMatches reports whether a category stays in the list while it is
// searched: its name, or any setting in it, holds what was typed.
func (a *App) tabMatches(tab prefsTab) bool {
	if a.prefsFilter == "" {
		return true
	}
	q := strings.ToLower(a.prefsFilter)
	if strings.Contains(strings.ToLower(tab.label), q) {
		return true
	}
	for _, s := range a.paneFields(tab.id) {
		if strings.Contains(strings.ToLower(s), q) {
			return true
		}
	}
	return false
}

// paneFields names the settings a category holds, for the search to match.
func (a *App) paneFields(id string) []string {
	switch id {
	case "file":
		return []string{locales.OpenWhen(), locales.OpenInCurrent(), locales.OpenInNew()}
	case "editor":
		return []string{locales.BodySize(), locales.LineHeight(), locales.BodyWidth()}
	case "image":
		return []string{locales.ImageWhen(), locales.ImageLocation(), locales.ImagePath(),
			locales.RelativePath(), locales.AbsolutePath(), locales.PasteImage()}
	case "markdown":
		return []string{locales.CodeBlock(), locales.WrapCode()}
	case "appearance":
		return []string{locales.Theme(), locales.Light(), locales.Dark(), locales.LangSystem(),
			locales.Custom(), locales.ColorBg(), locales.ColorText(), locales.ColorAccent(),
			locales.ColorSurface(), locales.ColorBorder(), locales.SideWidth(), locales.SideTextIcon(),
			locales.FontSize(), locales.IconSize(), locales.StatusBar(), locales.ShowCount(),
			locales.ReadingSpeedLabel(), locales.FindHighlight(), locales.ColorBg()}
	case "general":
		return []string{locales.Language(), locales.LangZh(), locales.LangEn(),
			locales.LangSystem(), locales.Launch(), locales.RestoreLast()}
	}
	return nil
}

// settingsPane is the settings of the category at the left that is chosen.
func (a *App) settingsPane(c *ui.Context, t *ui.Theme) {
	switch a.prefsTab {
	case "file":
		a.paneTitle(c, t, locales.PrefsFile())
		a.paneGroup(c, t, locales.OpenWhen(), func() {
			a.radioRow(c, t, locales.OpenInCurrent(), !a.set.OpenInNewWindow, func() { a.setOpenInNew(false) })
			a.radioRow(c, t, locales.OpenInNew(), a.set.OpenInNewWindow, func() { a.setOpenInNew(true) })
		})
		a.paneNote(c, t, locales.NoteOpen())
	case "editor":
		a.paneTitle(c, t, locales.PrefsEditor())
		a.paneGroup(c, t, locales.BodySize(), func() {
			a.stepper(c, t, "size", "", a.shared.Size, "px", 1, a.setFontSize)
		})
		a.paneGroup(c, t, locales.LineHeight(), func() {
			a.stepper(c, t, "line", "", a.shared.LineHeight, "", 0.1, a.setLineHeight)
		})
		a.paneGroup(c, t, locales.BodyWidth(), func() {
			a.stepper(c, t, "measure", "", a.shared.Measure, "px", 20, a.setMeasure)
		})
	case "image":
		a.paneTitle(c, t, locales.PrefsImage())
		a.paneGroup(c, t, locales.ImageWhen(), func() {
			a.radioRow(c, t, locales.CopyToAssets(), a.set.ImageDir != "same", func() { a.setImageDir("assets") })
			a.radioRow(c, t, locales.CopyToDocFolder(), a.set.ImageDir == "same", func() { a.setImageDir("same") })
		})
		a.paneGroup(c, t, locales.ImagePath(), func() {
			a.radioRow(c, t, locales.RelativePath(), a.set.ImagePath != "absolute", func() { a.setImagePath("relative") })
			a.radioRow(c, t, locales.AbsolutePath(), a.set.ImagePath == "absolute", func() { a.setImagePath("absolute") })
		})
		a.paneGroup(c, t, locales.PasteImage(), func() {
			a.checkRow(c, t, locales.PasteAndSave(), a.set.PasteImage, func(v bool) { a.setPasteImage(v) })
		})
	case "markdown":
		a.paneTitle(c, t, "Markdown")
		a.paneGroup(c, t, locales.CodeBlock(), func() {
			a.checkRow(c, t, locales.WrapCode(), a.set.WrapCode, func(v bool) { a.setWrapCode(v) })
		})
		a.paneNote(c, t, locales.NoteMarkdown())
	case "appearance":
		a.paneTitle(c, t, locales.PrefsAppearance())
		a.paneGroup(c, t, locales.Theme(), func() {
			a.radioRow(c, t, locales.LangSystem(), a.set.Theme == "auto", func() { a.setTheme("auto") })
			a.radioRow(c, t, locales.Light(), a.set.Theme == "light", func() { a.setTheme("light") })
			a.radioRow(c, t, locales.Dark(), a.set.Theme == "dark", func() { a.setTheme("dark") })
			a.radioRow(c, t, locales.Custom(), a.set.Theme == "custom", func() { a.setTheme("custom") })
			if a.set.Theme == "custom" {
				a.themeColorRows(c, t)
			}
		})
		a.paneGroup(c, t, locales.SideWidth(), func() {
			a.stepper(c, t, "sidewidth", "", a.shared.SideWidth, "px", 10, a.setSideWidth)
		})
		a.paneGroup(c, t, locales.SideTextIcon(), func() {
			a.stepper(c, t, "sidefont", locales.FontSize(), a.shared.SideFont, "px", 1, a.setSideFont)
			a.stepper(c, t, "sideicon", locales.IconSize(), a.shared.SideIcon, "px", 1, a.setSideIcon)
		})
		a.paneGroup(c, t, locales.StatusBar(), func() {
			a.checkRow(c, t, locales.ShowCount(), a.set.ShowStatusBar == nil || *a.set.ShowStatusBar,
				func(v bool) { a.setShowStatusBar(v) })
		})
		a.paneGroup(c, t, locales.FindHighlight(), func() {
			a.colorRow(c, t)
		})
		a.paneGroup(c, t, locales.ReadingSpeedLabel(), func() {
			a.stepper(c, t, "reading", "", float32(a.shared.ReadingSpeed), locales.CpmUnit(), 50,
				func(v float32) { a.setReadingSpeed(int(v)) })
		})
		a.paneNote(c, t, locales.NoteReading())
	case "general":
		a.paneTitle(c, t, locales.PrefsGeneral())
		a.paneGroup(c, t, locales.Language(), func() {
			a.radioRow(c, t, locales.LangSystem(), a.set.Language == "", func() { a.setLanguage("") })
			a.radioRow(c, t, locales.LangZh(), a.set.Language == "zh-Hans", func() { a.setLanguage("zh-Hans") })
			a.radioRow(c, t, locales.LangEn(), a.set.Language == "en", func() { a.setLanguage("en") })
		})
		a.paneGroup(c, t, locales.Launch(), func() {
			a.checkRow(c, t, locales.RestoreLast(), a.set.RestoreLast, func(v bool) { a.setRestoreLast(v) })
		})
	default:
		a.prefsTab = "file"
		a.settingsPane(c, t)
		return
	}
	ui.Box(c).Height(6)
	ui.Row(c).Children(func() {
		ui.Box(c).Grow(1)
		b := ui.Row(c).Padding(6, 14).Radius(5).Background(t.Text.Alpha(0.06))
		if b.Hovered() {
			b.Background(t.Text.Alpha(0.1))
		}
		b.OnClick(a.resetSettings)
		b.Children(func() { ui.Text(c, locales.ResetDefaults()).FontSize(13).TextColor(t.Text) })
	})
}

// themeColorRows are the colors the custom theme is built from: the page,
// its text, the accent, the panel and the lines, each with a well that opens
// the picker.
func (a *App) themeColorRows(c *ui.Context, t *ui.Theme) {
	rows := []struct {
		label string
		col   *ui.Color
		key   string
	}{
		{locales.ColorBg(), &a.shared.Custom.Background, "background"},
		{locales.ColorText(), &a.shared.Custom.Text, "text"},
		{locales.ColorAccent(), &a.shared.Custom.Accent, "accent"},
		{locales.ColorSurface(), &a.shared.Custom.Surface, "surface"},
		{locales.ColorBorder(), &a.shared.Custom.Border, "border"},
	}
	ui.Column(c).AlignSelf(ui.Stretch).Padding(8, 10).Radius(6).
		Background(t.Text.Alpha(0.035)).Children(func() {
		for _, r := range rows {
			r := r
			row := ui.Row(c).AlignSelf(ui.Stretch).AlignItems(ui.Center).Gap(12).
				Padding(4, 6).Radius(4)
			if row.Hovered() {
				row.Background(t.Text.Alpha(0.05))
			}
			row.Children(func() {
				ui.Text(c, r.label).FontSize(13).TextColor(t.Text)
				ui.Box(c).Grow(1)
				_, changed := a.colorSwatch(c, t, r.col, 64, 22, nil)
				if changed {
					a.setThemeColor(r.key, *r.col)
				}
			})
		}
	})
}

// colorSwatch is a swatch of col, filled whole, that opens the color picker
// below it. It reports whether the picker changed the color this frame.
func (a *App) colorSwatch(c *ui.Context, t *ui.Theme, col *ui.Color, w, h float32, clicked func()) (ui.Element, bool) {
	sw := ui.Box(c).Width(w).Height(h).Radius(5).Background(*col)
	sw.Border(1, t.Text.Alpha(0.14))
	open := ui.Local(sw, "open", func() bool { return false })
	sw.OnClick(func() {
		*open = !*open
		if clicked != nil {
			clicked()
		}
	})
	changed := false
	ui.Popover(c, sw, open, func() {
		ui.Column(c).Padding(10).Children(func() {
			p := ui.ColorPicker(c, col)
			changed = p.Changed()
		})
	})
	if sw.Hovered() {
		sw.Border(1, t.Text.Alpha(0.3))
	}
	return sw, changed
}

// colorRow is the color the search paints: five swatches, a well for a color
// of the user's own, the one in use ringed.
func (a *App) colorRow(c *ui.Context, t *ui.Theme) {
	choices := []struct {
		id  string
		col ui.Color
	}{
		{"pink", ui.RGBA(255, 120, 180, 0.55)},
		{"gray", ui.RGBA(150, 150, 150, 0.55)},
		{"yellow", ui.RGBA(255, 214, 0, 0.6)},
		{"blue", ui.RGBA(64, 156, 255, 0.55)},
		{"green", ui.RGBA(60, 190, 120, 0.55)},
	}
	ui.Row(c).Gap(10).Children(func() {
		for _, opt := range choices {
			opt := opt
			on := a.set.FindColor == opt.id
			sw := ui.Row(c).Width(24).Height(24).Radius(5).Background(opt.col)
			if on {
				sw.Border(2, t.Text)
			} else if sw.Hovered() {
				sw.Border(1, t.TextMuted)
			}
			sw.OnClick(func() { a.setFindColor(opt.id) })
			sw.Children(func() {})
			sw.Label("Search highlight: " + opt.id)
		}
		// A color of the user's own: a button that says so, with a chip of
		// the color in it, which opens the picker.
		ui.Box(c).Width(8)
		a.customColorButton(c, t)
	})
}

// customColorButton is the button that opens the picker for the search
// color of the user's own: the words say what it is, the chip in it shows
// the color, and the ring says it is the one in use.
func (a *App) customColorButton(c *ui.Context, t *ui.Theme) {
	on := a.set.FindColor == "custom"
	b := ui.Row(c).Height(24).Padding(3, 9).Gap(6).Radius(5).AlignItems(ui.Center).
		Background(t.Text.Alpha(0.06)).Border(1, t.Border)
	if on {
		b.Border(2, t.Text)
	} else if b.Hovered() {
		b.Background(t.Text.Alpha(0.12))
	}
	open := ui.Local(b, "open", func() bool { return false })
	b.OnClick(func() {
		*open = !*open
		a.setFindColor("custom")
	})
	b.Children(func() {
		ui.Box(c).Width(12).Height(12).Radius(3).Background(a.findWell)
		ui.Text(c, locales.Custom()).FontSize(12).TextColor(t.Text)
	})
	ui.Popover(c, b, open, func() {
		ui.Column(c).Padding(10).Children(func() {
			p := ui.ColorPicker(c, &a.findWell)
			if p.Changed() {
				a.setFindCustom(a.findWell)
			}
		})
	})
}

// paneTitle is the heading of a category, as Typora's sections start.
func (a *App) paneTitle(c *ui.Context, t *ui.Theme, label string) {
	ui.Text(c, label).FontSize(19).Bold().TextColor(t.Text)
}

// paneGroup is one setting: what it is called, then the control that sets
// it, with a hairline between the settings.
func (a *App) paneGroup(c *ui.Context, t *ui.Theme, label string, body func()) {
	ui.Column(c).Gap(10).Children(func() {
		ui.Text(c, label).FontSize(14).Bold().TextColor(t.Text)
		body()
	})
	ui.Box(c).Height(1).Background(t.Border.Alpha(0.5))
}

// paneNote is a line of explanation under a setting, smaller and quieter.
func (a *App) paneNote(c *ui.Context, t *ui.Theme, text string) {
	ui.Text(c, text).FontSize(12).TextColor(t.TextMuted)
}

// itoa is a small integer without a decimal point.
func itoa(n int) string {
	if n < 0 {
		return "-" + itoa(-n)
	}
	var b [20]byte
	i := len(b)
	for {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
		if n == 0 {
			break
		}
	}
	return string(b[i:])
}

// radioRow is one of the values a setting can take, marked when it is the
// one in use.
func (a *App) radioRow(c *ui.Context, t *ui.Theme, label string, on bool, fn func()) {
	row := ui.Row(c).Padding(7, 12).Gap(8).Radius(5)
	switch {
	case on:
		row.Background(t.Text.Alpha(0.08))
	case row.Hovered():
		row.Background(t.SurfaceHover)
	}
	row.OnClick(fn)
	row.Children(func() {
		mark := "○"
		if on {
			mark = "●"
		}
		ui.Text(c, mark).FontSize(12).TextColor(t.TextMuted)
		el := ui.Text(c, label).FontSize(13).TextColor(t.Text)
		if on {
			el.Bold()
		}
	})
}

// stepper is a value with a button either side of it, as a size is chosen.
func (a *App) stepper(c *ui.Context, t *ui.Theme, key, label string, value float32, unit string, step float32, set func(float32)) {
	format := func(v float32) string {
		if step >= 1 {
			return fmt.Sprintf("%.0f", v)
		}
		return fmt.Sprintf("%.1f", v)
	}
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		if label != "" {
			ui.Text(c, label).FontSize(13).TextColor(t.Text).Width(64)
		}
		a.stepButton(c, t, "−", func() { set(value - step) })
		row := ui.Row(c)
		row.Children(func() {
			// The value is a field: the arrows move it, and typing a number
			// in it sets it as well.
			field := ui.Local(row, key, func() string { return format(value) })
			ui.Themed(c, t).Children(func() {
				in := ui.TextInput(c.Key("step-"+key), field).Width(62).Height(28)
				in.FontSize(14)
				in.Padding(0, 0)
				in.TextAlign(ui.Center)
				in.Background(t.Background)
				in.Border(1, t.Border)
				in.FocusRingColor(t.Text)
				if !in.Focused() {
					*field = format(value)
				}
				in.OnChange(func() {
					if v, err := strconv.ParseFloat(strings.TrimSpace(*field), 64); err == nil {
						set(float32(v))
					}
				})
				in.OnSubmit(func() { *field = format(value) })
			})
		})
		if unit != "" {
			ui.Text(c, unit).FontSize(13).TextColor(t.TextMuted)
		}
		a.stepButton(c, t, "+", func() { set(value + step) })
	})
}

func (a *App) stepButton(c *ui.Context, t *ui.Theme, label string, fn func()) {
	b := ui.Row(c).Width(30).Height(28).Justify(ui.Center).AlignItems(ui.Center).
		Radius(5).Background(t.Text.Alpha(0.06))
	if b.Hovered() {
		b.Background(t.Text.Alpha(0.12))
	}
	b.OnClick(fn)
	b.Children(func() { ui.Text(c, label).FontSize(15).TextColor(t.Text) })
}

// checkRow is a setting that holds or not: a row that says what it does,
// with a box that is ticked while it holds.
func (a *App) checkRow(c *ui.Context, t *ui.Theme, label string, on bool, fn func(bool)) {
	row := ui.Row(c).Padding(7, 12).Gap(10).AlignItems(ui.Center).Radius(5)
	if row.Hovered() {
		row.Background(t.SurfaceHover)
	}
	row.OnClick(func() { fn(!on) })
	row.Children(func() {
		// In grays: ticked it fills with the muted ink and shows a check the
		// color of the surface, unticked it is just its outline.
		box := ui.Row(c).Width(16).Height(16).Radius(4).
			Justify(ui.Center).AlignItems(ui.Center)
		if on {
			box.Background(t.TextMuted)
			box.Children(func() { ui.Text(c, "✓").FontSize(10).TextColor(t.Surface) })
		} else {
			box.Border(1.4, t.TextMuted)
		}
		el := ui.Text(c, label).FontSize(13).TextColor(t.Text)
		if on {
			el.Bold()
		}
	})
}

// ---------- the settings, each remembered at once ----------

// setLanguage shows the interface in a language; "" follows the system. The
// windows redraw themselves, for they take their words each frame; the
// native menu bar and the settings' title are rebuilt by the listener main
// put in when the app started.
func (a *App) setLanguage(tag string) {
	a.set.Language = tag
	mygo.App.SetLocale(tag)
	a.saver.touch()
}

// setOpenInNew tells whether a click in the file panel opens a window of its
// own.
func (a *App) setOpenInNew(v bool) {
	a.set.OpenInNewWindow = v
	a.saver.touch()
}

// setImageDir is where an inserted picture is copied.
func (a *App) setImageDir(dir string) {
	a.set.ImageDir = dir
	a.shared.ImageDir = dir
	a.saver.touch()
}

// setImagePath is how an inserted picture is addressed.
func (a *App) setImagePath(kind string) {
	a.set.ImagePath = kind
	a.shared.ImagePath = kind
	a.saver.touch()
}

// setPasteImage tells whether a picture on the clipboard is saved when it is
// pasted.
func (a *App) setPasteImage(v bool) {
	a.set.PasteImage = v
	a.shared.PasteImage = v
	a.saver.touch()
}

// setFindColor is the color the search paints its matches.
func (a *App) setFindColor(id string) {
	a.set.FindColor = id
	a.shared.FindColor = id
	a.saver.touch()
}

// setFindCustom is the color the search paints after the user mixed one.
func (a *App) setFindCustom(c ui.Color) {
	// The highlight is a wash over the words, not a block of paint: the
	// color keeps the opacity every choice of the search has.
	a.findWell = editor.Opaque(c)
	c = c.Alpha(0.35)
	a.set.FindColor = "custom"
	a.set.FindCustom = editor.ColorHex(c)
	a.shared.FindColor = "custom"
	a.shared.FindCustom = c
	a.saver.touch()
}

// setThemeColor takes one color of the custom theme, and restyles the
// windows with it.
func (a *App) setThemeColor(key string, c ui.Color) {
	hex := editor.ColorHex(c)
	switch key {
	case "background":
		a.set.ThemeColors.Background = hex
	case "text":
		a.set.ThemeColors.Text = hex
	case "accent":
		a.set.ThemeColors.Accent = hex
	case "surface":
		a.set.ThemeColors.Surface = hex
	case "border":
		a.set.ThemeColors.Border = hex
	}
	a.shared.SetCustom(a.shared.Custom)
	a.saver.touch()
}

// useThemeColors puts the colors the settings kept into the look the windows
// draw with.
func (a *App) useThemeColors(c ThemeColors) {
	a.set.ThemeColors = c
	a.shared.Custom = editor.CustomTheme{
		Background: editor.ColorOr(ui.Color{}, c.Background, 1),
		Text:       editor.ColorOr(ui.Color{}, c.Text, 1),
		Accent:     editor.ColorOr(ui.Color{}, c.Accent, 1),
		Surface:    editor.ColorOr(ui.Color{}, c.Surface, 1),
		Border:     editor.ColorOr(ui.Color{}, c.Border, 1),
	}
}

// setTheme keeps the app light or dark, or following the system.
func (a *App) setTheme(kind string) {
	a.set.Theme = kind
	a.shared.SetTheme(kind)
	a.saver.touch()
}

// setSideFont is the size of the panel's text.
func (a *App) setSideFont(v float32) {
	a.shared.SetSideFont(v)
	a.set.SidebarFont = a.shared.SideFont
	a.saver.touch()
}

// setSideIcon is the size of the icons in the panel's bar.
func (a *App) setSideIcon(v float32) {
	a.shared.SetSideIcon(v)
	a.set.SidebarIcon = a.shared.SideIcon
	a.saver.touch()
}

// setWrapCode wraps long lines of code blocks, or lets them scroll.
func (a *App) setWrapCode(v bool) {
	a.set.WrapCode = v
	a.shared.SetWrapCode(v)
	a.saver.touch()
}

// setShowStatusBar shows or hides the word count under the page.
func (a *App) setShowStatusBar(v bool) {
	a.set.ShowStatusBar = &v
	a.shared.ShowStatusBar = v
	a.saver.touch()
}

// setReadingSpeed is how many characters a reader takes a minute.
func (a *App) setReadingSpeed(n int) {
	a.shared.SetReadingSpeed(n)
	a.set.ReadingSpeed = a.shared.ReadingSpeed
	a.saver.touch()
}

// setRestoreLast tells whether the file of the last session opens at start.
func (a *App) setRestoreLast(v bool) {
	a.set.RestoreLast = v
	a.saver.touch()
}

// setFontSize changes the size of the text of every window, and remembers it.
func (a *App) setFontSize(size float32) {
	a.shared.SetSize(size)
	a.set.FontSize = a.shared.Size
	a.saver.touch()
}

// setLineHeight changes how tall a line of body text is, and remembers it.
func (a *App) setLineHeight(h float32) {
	a.shared.SetLineHeight(h)
	a.set.LineHeight = a.shared.LineHeight
	a.saver.touch()
}

// setMeasure changes how wide the text column is, and remembers it.
func (a *App) setMeasure(w float32) {
	a.shared.SetMeasure(w)
	a.set.Measure = a.shared.Measure
	a.saver.touch()
}

// setSideWidth changes how wide the panel at the left is, and remembers it.
func (a *App) setSideWidth(w float32) {
	a.shared.SetSideWidth(w)
	a.set.SidebarWidth = a.shared.SideWidth
	a.saver.touch()
}

// resetSettings puts back what a first run starts with.
func (a *App) resetSettings() {
	d := DefaultSettings()
	a.setLanguage(d.Language)
	a.setFontSize(d.FontSize)
	a.setLineHeight(d.LineHeight)
	a.setMeasure(d.Measure)
	a.setSideWidth(d.SidebarWidth)
	a.setSideFont(d.SidebarFont)
	a.setSideIcon(d.SidebarIcon)
	a.setShowStatusBar(true)
	a.setReadingSpeed(d.ReadingSpeed)
	a.setWrapCode(d.WrapCode)
	a.setImageDir(d.ImageDir)
	a.setImagePath(d.ImagePath)
	a.setPasteImage(d.PasteImage)
	a.setOpenInNew(d.OpenInNewWindow)
	a.setRestoreLast(d.RestoreLast)
	a.setTheme(d.Theme)
	a.useThemeColors(d.ThemeColors)
	a.shared.FindCustom = editor.ColorOr(ui.RGBA(255, 120, 180, 0.35), d.FindCustom, 0.35)
	a.findWell = editor.Opaque(a.shared.FindCustom)
	a.set.FindCustom = d.FindCustom
	a.setFindColor(d.FindColor)
	a.saver.touch()
}
