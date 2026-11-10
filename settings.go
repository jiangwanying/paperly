package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Settings is what the app remembers between runs.
type Settings struct {
	// Language is the tag the interface shows in: "zh-Hans", "en", or ""
	// to follow the system.
	Language string `json:"language"`
	// Sidebar tells whether the panel at the left shows, which one it is
	// ("outline", "tree" or "list"), and how wide it is.
	Sidebar      bool    `json:"sidebar"`
	SidebarMode  string  `json:"sidebarMode"`
	SidebarWidth float32 `json:"sidebarWidth"`
	// Theme is "auto", "light" or "dark".
	Theme string `json:"theme"`
	// FontSize is the size of body text, in DIPs.
	FontSize float32 `json:"fontSize"`
	// LineHeight is how tall a line of body text is, against its size.
	LineHeight float32 `json:"lineHeight"`
	// Measure is how wide the text column is, in DIPs.
	Measure float32 `json:"measure"`
	// SidebarFont is the size of the panel's text, in DIPs, and SidebarIcon
	// the size of the icons in its bar.
	SidebarFont float32 `json:"sidebarFont"`
	SidebarIcon float32 `json:"sidebarIcon"`
	// ShowStatusBar tells whether the word count shows under the page.
	ShowStatusBar *bool `json:"showStatusBar"`
	// Folder is the folder the file panel shows, when one was opened.
	Folder string `json:"folder"`

	// OpenInNewWindow opens a file in a window of its own, as 文件 → 打开
	// does, rather than in the window the click came from.
	OpenInNewWindow bool `json:"openInNewWindow"`
	// WrapCode wraps long lines of code blocks instead of scrolling them.
	WrapCode bool `json:"wrapCode"`
	// ImageDir is where an inserted picture is copied: "assets" for a folder
	// beside the document, or "same" for the document's own folder.
	ImageDir string `json:"imageDir"`
	// ImagePath writes the picture's address as "relative" to the document,
	// or as an "absolute" path.
	ImagePath string `json:"imagePath"`
	// PasteImage saves a picture on the clipboard into the document when it
	// is pasted.
	PasteImage bool `json:"pasteImage"`
	// ReadingSpeed is how many characters a reader takes a minute, for the
	// time the status bar shows.
	ReadingSpeed int `json:"readingSpeed"`
	// RestoreLast opens the file of the last session at start, rather than
	// an empty document.
	// Theme is "auto" to follow the system, or "light" and "dark".
	RestoreLast bool `json:"restoreLast"`
	// LastFile is the file that was open when the app last quit.
	LastFile string `json:"lastFile"`
	// FindColor is the color the search paints its matches: "pink" by
	// default, or "gray", "yellow", "blue", "green" and "custom", which
	// paints FindCustom.
	FindColor string `json:"findColor"`
	// FindCustom is the color "custom" paints, as "#rrggbb".
	FindCustom string `json:"findCustom"`
	// ThemeColors are the colors the "custom" theme draws with, as
	// "#rrggbb" each.
	ThemeColors ThemeColors `json:"themeColors"`
}

// ThemeColors are the colors the user built their theme from: the page, its
// text, the accent, the panel's face and the lines between things.
type ThemeColors struct {
	Background string `json:"background"`
	Text       string `json:"text"`
	Accent     string `json:"accent"`
	Surface    string `json:"surface"`
	Border     string `json:"border"`
}

// DefaultThemeColors are the colors the custom theme starts with.
func DefaultThemeColors() ThemeColors {
	return ThemeColors{
		Background: "#fbfbfb",
		Text:       "#1f2328",
		Accent:     "#3b82f6",
		Surface:    "#f4f4f5",
		Border:     "#e3e3e6",
	}
}

// DefaultSettings is what a first run starts with: Chinese, outline shown,
// following the system's light or dark.
func DefaultSettings() *Settings {
	return &Settings{
		Language:     "zh-Hans",
		Sidebar:      false,
		SidebarMode:  "outline",
		SidebarWidth: 270,
		Theme:        "auto",
		FontSize:     16,
		SidebarFont:  14,
		SidebarIcon:  15,
		LineHeight:   1.6,
		Measure:      860,
		ImageDir:     "assets",
		ImagePath:    "relative",
		PasteImage:   true,
		ReadingSpeed: 400,
		FindColor:    "pink",
		FindCustom:   "#ff78b4",
		ThemeColors:  DefaultThemeColors(),
	}
}

// LoadSettings reads a settings file, and leaves the defaults where the file
// is missing, unreadable or broken: a bad file must not keep the app from
// starting.
func LoadSettings(path string) *Settings {
	s := DefaultSettings()
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, s)
	}
	if s.SidebarWidth <= 0 {
		s.SidebarWidth = 270
	}
	if s.Theme == "" {
		s.Theme = "auto"
	}
	if s.FontSize <= 0 {
		s.FontSize = 16
	}
	if s.SidebarFont <= 0 {
		s.SidebarFont = 14
	}
	if s.SidebarIcon <= 0 {
		s.SidebarIcon = 15
	}
	if s.LineHeight <= 0 {
		s.LineHeight = 1.6
	}
	if s.Measure <= 0 {
		s.Measure = 860
	}
	if s.ImageDir != "same" {
		s.ImageDir = "assets"
	}
	if s.ImagePath != "absolute" {
		s.ImagePath = "relative"
	}
	if s.ReadingSpeed <= 0 {
		s.ReadingSpeed = 400
	}
	switch s.FindColor {
	case "yellow", "blue", "green", "pink", "gray", "custom":
	default:
		s.FindColor = "pink"
	}
	if s.FindCustom == "" {
		s.FindCustom = "#ff78b4"
	}
	d := DefaultThemeColors()
	if s.ThemeColors.Background == "" {
		s.ThemeColors.Background = d.Background
	}
	if s.ThemeColors.Text == "" {
		s.ThemeColors.Text = d.Text
	}
	if s.ThemeColors.Accent == "" {
		s.ThemeColors.Accent = d.Accent
	}
	if s.ThemeColors.Surface == "" {
		s.ThemeColors.Surface = d.Surface
	}
	if s.ThemeColors.Border == "" {
		s.ThemeColors.Border = d.Border
	}
	return s
}

// SaveSettings writes the file whole or not at all: a temporary file renamed
// over it, so a crash mid-write leaves the old one.
func SaveSettings(path string, s *Settings) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// saver writes the settings a moment after the last change, so that dragging
// a splitter does not write the file at every frame.
type saver struct {
	path string
	s    *Settings

	mu    sync.Mutex
	timer *time.Timer
	err   error
}

func newSaver(path string, s *Settings) *saver { return &saver{path: path, s: s} }

// touch schedules a write half a second from now.
func (sv *saver) touch() {
	sv.mu.Lock()
	defer sv.mu.Unlock()
	if sv.timer != nil {
		sv.timer.Stop()
	}
	sv.timer = time.AfterFunc(500*time.Millisecond, func() { _ = sv.flush() })
}

// flush writes at once, for quitting.
func (sv *saver) flush() error {
	sv.mu.Lock()
	if sv.timer != nil {
		sv.timer.Stop()
		sv.timer = nil
	}
	sv.mu.Unlock()
	err := SaveSettings(sv.path, sv.s)
	sv.mu.Lock()
	sv.err = err
	sv.mu.Unlock()
	return err
}
