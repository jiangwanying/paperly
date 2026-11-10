package main

import (
	"os"
	"path/filepath"
	"testing"
)

// Settings survive a round trip, and a broken file leaves the defaults.
func TestSettingsRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	s := DefaultSettings()
	s.Language = "en"
	s.Sidebar = false
	s.SidebarWidth = 260
	s.Theme = "dark"
	s.SidebarFont = 16
	s.SidebarIcon = 18
	if err := SaveSettings(path, s); err != nil {
		t.Fatal(err)
	}

	got := LoadSettings(path)
	if got.Language != "en" || got.Sidebar || got.SidebarWidth != 260 || got.Theme != "dark" {
		t.Errorf("read back %+v", got)
	}
	if got.SidebarFont != 16 || got.SidebarIcon != 18 {
		t.Errorf("the panel's sizes came back as %v / %v", got.SidebarFont, got.SidebarIcon)
	}

	// No temporary file is left behind.
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Errorf("a temporary file is left: %v", err)
	}
}

func TestSettingsDefaults(t *testing.T) {
	dir := t.TempDir()

	missing := LoadSettings(filepath.Join(dir, "none.json"))
	// A first run opens on the page alone: no panel until it is asked for.
	if missing.Language != "zh-Hans" || missing.Sidebar || missing.Theme != "auto" {
		t.Errorf("missing file gives %+v", missing)
	}

	broken := filepath.Join(dir, "broken.json")
	if err := os.WriteFile(broken, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := LoadSettings(broken)
	if got.Language != "zh-Hans" || got.SidebarWidth != 270 {
		t.Errorf("broken file gives %+v", got)
	}
}

// The search paints pink until another color is chosen, and both the color
// of the user's own and the colors of their theme survive a round trip.
func TestSettingsColors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	s := DefaultSettings()
	if s.FindColor != "pink" || s.FindCustom == "" {
		t.Fatalf("a first run starts with %q / %q", s.FindColor, s.FindCustom)
	}
	s.FindColor = "custom"
	s.FindCustom = "#abcdef"
	s.Theme = "custom"
	s.ThemeColors = ThemeColors{
		Background: "#101010",
		Text:       "#eeeeee",
		Accent:     "#123456",
		Surface:    "#181818",
		Border:     "#303030",
	}
	if err := SaveSettings(path, s); err != nil {
		t.Fatal(err)
	}
	got := LoadSettings(path)
	if got.FindColor != "custom" || got.FindCustom != "#abcdef" {
		t.Errorf("the search color came back as %q / %q", got.FindColor, got.FindCustom)
	}
	if got.Theme != "custom" || got.ThemeColors.Accent != "#123456" || got.ThemeColors.Text != "#eeeeee" {
		t.Errorf("the theme colors came back as %+v", got.ThemeColors)
	}
}
