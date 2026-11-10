package editor

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
)

// Shared is what every window reads and none of them changes.
type Shared struct {
	Measure    float32 // the width of the text column, in DIPs
	Size       float32 // the size of body text, in DIPs
	LineHeight float32 // how tall a line of body text is, against its size
	// Style counts the changes of the look: a window whose themes were
	// built for another number builds them again.
	Style int

	// Side is the panel the windows show at the left, if any, and
	// SideWidth how wide it is.
	Side      SideMode
	SideWidth float32
	// SideFont is the size of the panel's text, and SideIcon the size of the
	// icons in its bar.
	SideFont float32
	SideIcon float32
	// ShowStatusBar tells whether the word count shows under the page.
	ShowStatusBar bool
	last          SideMode // the panel to open again
	lastFile      SideMode // the file panel to open: tree or list

	// Folder is the folder the file panel shows, opened with 打开文件夹…;
	// empty means the folder of the document in the window.
	Folder string

	// WrapCode wraps long lines of code blocks instead of scrolling them.
	WrapCode bool
	// ImageDir is where an inserted picture is copied: "assets" for a folder
	// beside the document, "same" for the document's own.
	ImageDir string
	// ImagePath writes the picture's address as "relative" or "absolute".
	ImagePath string
	// PasteImage saves a picture on the clipboard when it is pasted.
	PasteImage bool
	// ReadingSpeed is how many characters a reader takes a minute.
	ReadingSpeed int
	// Theme is "auto" to follow the system, "light" or "dark" to keep one
	// look whatever the system shows, or "custom" to draw the colors of
	// Custom.
	Theme string
	// Custom is the look the user builds, drawn when Theme is "custom".
	Custom CustomTheme
	// FindColor is the color the search paints: pink by default, or "gray",
	// "yellow", "blue", "green", "pink" and "custom", which paints
	// FindCustom.
	FindColor string
	// FindCustom is the color "custom" paints.
	FindCustom ui.Color
	// OpenFolder shows a folder in the file panel, asking the user to
	// choose one; nil leaves the empty panel inert.
	OpenFolder func(win *mygo.Window)

	// Open shows a file in the window the click came from, asking to save
	// what was written there first; nil leaves the list inert.
	Open func(win *mygo.Window, path string)
	// OpenAt shows a file like Open does, and brings the line to view,
	// as a hit of the folder's search does.
	OpenAt func(win *mygo.Window, path string, line int)
	// Rename renames a file or a folder of the panel on the disk, and
	// brings the windows that show it along; an error explains why not.
	Rename func(oldPath, newPath string) error
	// Delete moves a file or a folder of the panel to the trash, and
	// closes the windows that show it; an error explains why not.
	Delete func(path string) error
	// OnSideChange tells the app that the panel changed, to remember it.
	OnSideChange func()
	// OnFolderChange tells the app that the folder changed, to remember it.
	OnFolderChange func()
}

// SideMode is the panel at the left of a window, as Typora's three: the
// outline of the document, the folder's tree, or its files as a list.
type SideMode int

const (
	// SideNone shows no panel.
	SideNone SideMode = iota
	// SideOutline lists the document's headings.
	SideOutline
	// SideTree shows the document's folder as a tree.
	SideTree
	// SideList lists the Markdown files of the folder.
	SideList
)

// CustomTheme is the look the user builds when Theme is "custom": the page,
// its text, the accent, the panel's face and the lines between things.
type CustomTheme struct {
	Background ui.Color
	Text       ui.Color
	Accent     ui.Color
	Surface    ui.Color
	Border     ui.Color
}

// DefaultCustomTheme is the custom look a first run starts with: Typora's
// light page, near-black text and a blue accent.
func DefaultCustomTheme() CustomTheme {
	return CustomTheme{
		Background: ui.RGBA(251, 251, 251, 1),
		Text:       ui.RGBA(31, 35, 40, 1),
		Accent:     ui.RGBA(59, 130, 246, 1),
		Surface:    ui.RGBA(244, 244, 245, 1),
		Border:     ui.RGBA(227, 227, 230, 1),
	}
}

// NewShared returns the look the editor starts with.
func NewShared() *Shared {
	return &Shared{
		Measure: 860, Size: 16, LineHeight: 1.6,
		SideWidth: 270, Theme: "auto",
		SideFont: 14, SideIcon: 15,
		ShowStatusBar: true, ReadingSpeed: 400,
		Custom: DefaultCustomTheme(), FindColor: "pink",
		FindCustom: ui.RGBA(255, 120, 180, 0.55),
	}
}

// SetFolder shows a folder in the file panel, and tells the app so.
func (sh *Shared) SetFolder(dir string) {
	if sh.Folder == dir {
		return
	}
	sh.Folder = dir
	if sh.OnFolderChange != nil {
		sh.OnFolderChange()
	}
}

// Show opens a panel at the left.
func (sh *Shared) Show(m SideMode) {
	if m == SideNone || sh.Side == m {
		return
	}
	sh.Side = m
	sh.remember(m)
	sh.changed()
}

// Files is the file panel the button at the top of the sidebar opens: the
// tree or the list, whichever was shown last.
func (sh *Shared) Files() SideMode {
	if sh.lastFile == SideTree || sh.lastFile == SideList {
		return sh.lastFile
	}
	return SideList
}

// remember keeps the file panel the user chose, to open it again.
func (sh *Shared) remember(m SideMode) {
	if m == SideTree || m == SideList {
		sh.lastFile = m
	}
}

// Toggle closes the panel when one shows, and opens the last one, or the
// outline, when none does.
func (sh *Shared) Toggle() {
	if sh.Side != SideNone {
		sh.last = sh.Side
		sh.Side = SideNone
	} else {
		sh.Side = sh.last
		if sh.Side == SideNone {
			sh.Side = SideOutline
		}
	}
	sh.remember(sh.Side)
	sh.changed()
}

// SetSize sets the size of body text, between 12 and 24 DIPs.
func (sh *Shared) SetSize(size float32) {
	if size < 12 {
		size = 12
	}
	if size > 24 {
		size = 24
	}
	if sh.Size == size {
		return
	}
	sh.Size = size
	sh.Style++
}

// SetTheme keeps the app light, dark, as the user builds it, or following the
// system, and restyles the windows.
func (sh *Shared) SetTheme(kind string) {
	if kind != "light" && kind != "dark" && kind != "custom" {
		kind = "auto"
	}
	if sh.Theme == kind {
		return
	}
	sh.Theme = kind
	sh.Style++
}

// SetCustom takes the look the user built, and restyles the windows.
func (sh *Shared) SetCustom(c CustomTheme) {
	sh.Custom = c
	sh.Style++
}

// Look is the theme a window draws with: the system's, unless the settings
// keep one.
func (sh *Shared) Look(system *ui.Theme) *ui.Theme {
	switch sh.Theme {
	case "light":
		return ui.LightTheme()
	case "dark":
		return ui.DarkTheme()
	case "custom":
		return customLook(sh.Custom)
	}
	return system
}

// customLook builds the theme the user's colors make: a light or a dark one
// as the page's own brightness says, with their colors put over it and the
// rest worked out from them.
func customLook(c CustomTheme) *ui.Theme {
	if c.Background.A == 0 {
		c = DefaultCustomTheme()
	}
	dark := luma(c.Background) < 0.5
	var t *ui.Theme
	if dark {
		t = ui.DarkTheme()
	} else {
		t = ui.LightTheme()
	}
	t.Dark = dark
	t.Background = c.Background
	t.Surface = c.Surface
	t.SurfaceHover = c.Surface.Mix(c.Text, 0.08)
	t.SurfacePressed = c.Surface.Mix(c.Text, 0.14)
	t.Border = c.Border
	t.Text = c.Text
	t.TextMuted = c.Text.Alpha(0.55)
	t.Accent = c.Accent
	t.AccentHover = c.Accent.Mix(c.Text, 0.12)
	t.AccentPressed = c.Accent.Mix(c.Text, 0.2)
	t.AccentText = ui.RGBA(255, 255, 255, 1)
	if luma(c.Accent) > 0.65 {
		t.AccentText = ui.RGBA(17, 17, 17, 1)
	}
	t.Focus = c.Accent
	t.Selection = c.Accent.Alpha(0.28)
	t.Scrollbar = c.Text.Alpha(0.25)
	t.Inverse = c.Text
	t.InverseText = c.Background
	return t
}

// luma is how bright a color is, from 0 to 1.
func luma(c ui.Color) float32 {
	return (0.2126*float32(c.R) + 0.7152*float32(c.G) + 0.0722*float32(c.B)) / 255
}

// ColorOr reads a color the settings kept as "#rrggbb", or the one to fall
// back to, with the alpha given.
func ColorOr(fallback ui.Color, hex string, alpha float32) ui.Color {
	if c, ok := ParseColor(hex); ok {
		return c.Alpha(alpha)
	}
	return fallback
}

// Opaque is c without its transparency, as a swatch of it shows.
func Opaque(c ui.Color) ui.Color {
	c.A = 255
	return c
}

// ColorHex is c as "#rrggbb", the way the settings keep a color.
func ColorHex(c ui.Color) string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

// ParseColor reads a color written as "#rrggbb"; the second answer reports
// whether it could.
func ParseColor(s string) (ui.Color, bool) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 {
		return ui.Color{}, false
	}
	n, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return ui.Color{}, false
	}
	return ui.RGBA(uint8(n>>16), uint8(n>>8), uint8(n), 1), true
}

// SetSideFont sets the size of the panel's text, between 11 and 18 DIPs.
func (sh *Shared) SetSideFont(v float32) {
	v = min(max(v, 11), 18)
	if sh.SideFont == v {
		return
	}
	sh.SideFont = v
}

// SetSideIcon sets the size of the icons in the panel's bar, between 12 and
// 24 DIPs.
func (sh *Shared) SetSideIcon(v float32) {
	v = min(max(v, 12), 24)
	if sh.SideIcon == v {
		return
	}
	sh.SideIcon = v
}

// SetWrapCode wraps long lines of code blocks, or lets them scroll, and
// restyles the windows.
func (sh *Shared) SetWrapCode(wrap bool) {
	if sh.WrapCode == wrap {
		return
	}
	sh.WrapCode = wrap
	sh.Style++
}

// SetReadingSpeed sets how many characters a reader takes a minute.
func (sh *Shared) SetReadingSpeed(n int) {
	if n < 100 {
		n = 100
	}
	if n > 2000 {
		n = 2000
	}
	sh.ReadingSpeed = n
}

// SetLineHeight sets how tall a line of body text is, between 1.2 and 2.2
// against its size.
func (sh *Shared) SetLineHeight(h float32) {
	if h < 1.2 {
		h = 1.2
	}
	if h > 2.2 {
		h = 2.2
	}
	if sh.LineHeight == h {
		return
	}
	sh.LineHeight = h
	sh.Style++
}

// SetMeasure sets how wide the text column is, between 560 and 1200 DIPs.
func (sh *Shared) SetMeasure(w float32) {
	if w < 560 {
		w = 560
	}
	if w > 1200 {
		w = 1200
	}
	if sh.Measure == w {
		return
	}
	sh.Measure = w
	sh.Style++
}

// SetSideWidth sets how wide the panel at the left is.
func (sh *Shared) SetSideWidth(w float32) {
	if w < 160 {
		w = 160
	}
	if w > 640 {
		w = 640
	}
	if sh.SideWidth == w {
		return
	}
	sh.SideWidth = w
	sh.changed()
}

// SideMoved tells the app that the divider moved the panel, to remember
// the width it gave it.
func (sh *Shared) SideMoved() { sh.changed() }

func (sh *Shared) changed() {
	if sh.OnSideChange != nil {
		sh.OnSideChange()
	}
}

// documentTheme builds the two themes a document shows with: the colors,
// sizes and spacing of its Markdown, and those of the text inputs, whose
// caret is black rather than the theme's accent blue.
func documentTheme(t *ui.Theme, sh *Shared) (*markdown.Theme, *ui.Theme) {
	md := markdown.DefaultTheme(t)
	// The sizes and spacing of Typora's github theme: 16px text on a 1.6
	// line, headings 2.25, 1.75, 1.5 and 1.25 as large, each with a line of
	// its own above and below.
	md.FontSize = sh.Size
	md.LineHeight = sh.LineHeight
	md.BlockSpacing = 1.0
	md.ItemSpacing = 0.35
	md.ListIndent = 1.5
	md.Headings = [6]markdown.HeadingStyle{
		{Size: 2.25, Weight: 700, Top: 1.0, Bottom: 0.6},
		{Size: 1.75, Weight: 700, Top: 1.0, Bottom: 0.6},
		{Size: 1.5, Weight: 700, Top: 1.0, Bottom: 0.5},
		{Size: 1.25, Weight: 700, Top: 1.0, Bottom: 0.5},
		{Size: 1.0, Weight: 700, Top: 1.0, Bottom: 0.5},
		{Size: 1.0, Weight: 700, Top: 1.0, Bottom: 0.5},
	}
	md.TableCellPadding = [2]float32{0.75, 0.45}

	ut := *t
	ut.FontSize = sh.Size
	// The caret is black on a light page, and the color of the text on a
	// dark one, where black would be invisible.
	if ut.Dark {
		ut.Accent = ut.Text
		ut.AccentText = ut.Background
	} else {
		ut.Accent = ui.RGBA(26, 26, 26, 1)
		ut.AccentText = ui.RGBA(255, 255, 255, 1)
	}
	return md, &ut
}
