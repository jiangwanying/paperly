package main

import (
	"strings"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/i18n"
)

// The menu bar has the six menus, in order, and every item of ours has an
// ID and a label.
func TestMenuShape(t *testing.T) {
	a := NewApp(DefaultSettings(), "")
	items := a.menuItems()

	var labels []string
	for _, it := range items {
		if it.Label != "" {
			labels = append(labels, it.Label)
		}
	}
	want := []string{"文件", "编辑", "段落", "格式", "视图", "帮助"}
	if len(labels) != len(want) {
		t.Fatalf("menus %q, want %q", labels, want)
	}
	for i := range want {
		if labels[i] != want[i] {
			t.Fatalf("menus %q, want %q", labels, want)
		}
	}

	ids := map[string]bool{}
	var walk func(items []*mygo.MenuItem)
	walk = func(items []*mygo.MenuItem) {
		for _, it := range items {
			if it.Role == "" && it.Label != "" && it.Submenu == nil {
				if it.ID == "" {
					t.Errorf("%q has no ID", it.Label)
				}
				if ids[it.ID] {
					t.Errorf("ID %q is used twice", it.ID)
				}
				ids[it.ID] = true
			}
			walk(it.Submenu)
		}
	}
	walk(items)

	if !ids["view.source"] || !ids["fmt.bold"] || !ids["help.about"] {
		t.Errorf("some IDs are missing: %v", ids)
	}
}

// Two items never share a shortcut: the menu would answer for one of them.
func TestMenuShortcutsUnique(t *testing.T) {
	a := NewApp(DefaultSettings(), "")
	seen := map[string]string{}
	var walk func(items []*mygo.MenuItem)
	walk = func(items []*mygo.MenuItem) {
		for _, it := range items {
			if it.Accelerator != "" {
				if other, ok := seen[it.Accelerator]; ok {
					t.Errorf("%q and %q share the shortcut %s", other, it.Label, it.Accelerator)
				}
				seen[it.Accelerator] = it.Label
			}
			walk(it.Submenu)
		}
	}
	walk(a.menuItems())
}

// The interface starts in Chinese, and the catalog that translates MyGo's
// own strings is linked in (see main.go).
func TestChineseByDefault(t *testing.T) {
	if got := DefaultSettings().Language; got != "zh-Hans" {
		t.Errorf("the default language is %q, want zh-Hans", got)
	}
}

// Windows and Linux find the preferences and quitting at the end of the file
// menu: they have no application menu for MyGo to hide the roles into.
func TestFileTail(t *testing.T) {
	a := NewApp(DefaultSettings(), "")
	if got := a.fileTail(true); got != nil {
		t.Errorf("macOS gets a tail: %v", got)
	}
	tail := a.fileTail(false)
	if len(tail) != 3 {
		t.Fatalf("%d items, want 3", len(tail))
	}
	if tail[1].Label != "设置" || tail[1].Accelerator != "CmdOrCtrl+," {
		t.Errorf("preferences item %+v", tail[1])
	}
	if tail[2].Role != mygo.RoleQuit {
		t.Errorf("quit role %q", tail[2].Role)
	}
}

// A change of language builds the native pieces again: the menu bar's words
// were fixed when it was first made.
func TestRebuildNativeFollowsLanguage(t *testing.T) {
	a := NewApp(DefaultSettings(), "")
	relabel := func() string {
		a.rebuildNative()
		var out []string
		for _, it := range mygo.App.Menu().Items() {
			if it != nil && it.Label != "" {
				out = append(out, it.Label)
			}
		}
		return strings.Join(out, "|")
	}

	i18n.SetLocaleFunc(func() string { return "en" })
	defer i18n.SetLocaleFunc(func() string { return "zh-Hans" })
	if got := relabel(); !strings.Contains(got, "File") || !strings.Contains(got, "Format") {
		t.Errorf("the menu bar stayed in Chinese: %q", got)
	}
	i18n.SetLocaleFunc(func() string { return "zh-Hans" })
	if got := relabel(); !strings.Contains(got, "文件") || !strings.Contains(got, "格式") {
		t.Errorf("the menu bar did not come back: %q", got)
	}
}
