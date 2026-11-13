package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"paperly/locales"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"paperly/internal/editor"
)

// App is what every window shares: the documents, the settings, and the
// editor each window shows.
type App struct {
	shared *editor.Shared
	set    *Settings
	saver  *saver

	findWell    ui.Color           // the search color the settings' well shows, whole
	prefs       *mygo.Window       // the preferences window, while it is open
	about       *mygo.Window       // the about panel, while it is open
	trash       func(string) error // the wastebasket, which a test takes in hand
	prefsParent *mygo.Window       // the window it was opened from
	prefsTab    string             // the category it shows
	prefsFilter string             // what its search field holds

	mu   sync.Mutex
	docs *mygo.Documents[string]
	wins map[*mygo.Window]*editor.Editor
	last *mygo.Window
}

// NewApp makes the app's state. Call it before mygo.NewDocuments.
func NewApp(set *Settings, settingsPath string) *App {
	sh := editor.NewShared()
	sh.SideWidth = set.SidebarWidth
	sh.Size = set.FontSize
	sh.SideFont = set.SidebarFont
	sh.SideIcon = set.SidebarIcon
	sh.LineHeight = set.LineHeight
	sh.Measure = set.Measure
	sh.ShowStatusBar = set.ShowStatusBar == nil || *set.ShowStatusBar
	sh.Folder = set.Folder
	sh.WrapCode = set.WrapCode
	sh.ImageDir = set.ImageDir
	sh.ImagePath = set.ImagePath
	sh.PasteImage = set.PasteImage
	sh.ReadingSpeed = set.ReadingSpeed
	sh.Theme = set.Theme
	sh.FindColor = set.FindColor
	sh.FindCustom = editor.ColorOr(ui.RGBA(255, 120, 180, 0.35), set.FindCustom, 0.35)
	sh.Custom = editor.CustomTheme{
		Background: editor.ColorOr(ui.Color{}, set.ThemeColors.Background, 1),
		Text:       editor.ColorOr(ui.Color{}, set.ThemeColors.Text, 1),
		Accent:     editor.ColorOr(ui.Color{}, set.ThemeColors.Accent, 1),
		Surface:    editor.ColorOr(ui.Color{}, set.ThemeColors.Surface, 1),
		Border:     editor.ColorOr(ui.Color{}, set.ThemeColors.Border, 1),
	}
	switch {
	case !set.Sidebar:
		sh.Side = editor.SideNone
	case set.SidebarMode == "tree":
		sh.Side = editor.SideTree
	case set.SidebarMode == "list":
		sh.Side = editor.SideList
	default:
		sh.Side = editor.SideOutline
	}
	a := &App{
		shared:   sh,
		set:      set,
		saver:    newSaver(settingsPath, set),
		wins:     map[*mygo.Window]*editor.Editor{},
		findWell: editor.Opaque(sh.FindCustom),
		trash:    moveToTrash,
	}
	// The folder the file panel shows is remembered too.
	sh.OnFolderChange = func() {
		a.set.Folder = sh.Folder
		a.saver.touch()
	}
	// The panel is a setting: opening or closing it is remembered at once.
	sh.OnSideChange = func() {
		switch sh.Side {
		case editor.SideOutline:
			a.set.Sidebar, a.set.SidebarMode = true, "outline"
		case editor.SideTree:
			a.set.Sidebar, a.set.SidebarMode = true, "tree"
		case editor.SideList:
			a.set.Sidebar, a.set.SidebarMode = true, "list"
		default:
			a.set.Sidebar, a.set.SidebarMode = false, ""
		}
		a.set.SidebarWidth = sh.SideWidth
		a.saver.touch()
	}
	// The menu bar and the settings' title are native, and their words were
	// fixed when they were first made; the windows redraw themselves, for
	// they take their words each frame. A change of language, then, builds
	// them again. The listener runs where the change does, the main thread,
	// as the menu wants.
	mygo.App.OnLocaleChanged(func(string) { a.rebuildNative() })
	return a
}

// rebuildNative makes again the pieces the system draws once: the menu bar,
// and the settings window's title while it is open.
func (a *App) rebuildNative() {
	mygo.App.SetMenu(a.menu())
	if a.prefs != nil && !a.prefs.IsDestroyed() {
		a.prefs.SetTitle(locales.Settings())
	}
}

// switchTo shows a file in the window it was asked from, as Typora does: the
// window closes as if its close button were clicked — asking to save what
// was written there — and the file takes its place. A close the user cancels
// leaves the window, and opens nothing.
func (a *App) switchTo(win *mygo.Window, path string) {
	a.mu.Lock()
	docs := a.docs
	a.mu.Unlock()
	if docs == nil {
		return
	}
	if win == nil || a.set.OpenInNewWindow {
		a.openFile(path)
		return
	}
	d := docs.ForWindow(win)
	if d == nil || d.State().Path == path {
		return
	}
	// The window reports its closing before the app counts the windows left,
	// so the one that takes its place is up in time.
	win.OnClosed(func() { a.openFile(path) })
	win.Close()
}

// openFile opens a file in a window of its own, as the file list does.
func (a *App) openFile(path string) {
	a.mu.Lock()
	docs := a.docs
	a.mu.Unlock()
	if docs == nil {
		return
	}
	if _, err := docs.Open(path); err != nil {
		log.Printf("open %s: %v", path, err)
	}
}

// Editor is the editor of a window, or nil. Menu commands come with the
// window they act on; a menu item chosen while no window has the focus, as
// on macOS with every window closed, falls back to the last one.
func (a *App) Editor(win *mygo.Window) *editor.Editor {
	if win == nil {
		win = mygo.FocusedWindow()
	}
	if win == nil {
		win = a.last
	}
	if win == nil {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.wins[win]
}

func (a *App) register(win *mygo.Window, ed *editor.Editor) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.wins[win] = ed
	a.last = win
}

func (a *App) unregister(win *mygo.Window) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.wins, win)
	if a.last == win {
		a.last = nil
	}
}

// cmd runs fn on the editor of win, when there is one.
func (a *App) cmd(win *mygo.Window, fn func(ed *editor.Editor)) {
	if ed := a.Editor(win); ed != nil {
		fn(ed)
	}
}

// cmdChecked runs fn and reports what it returns, for a menu item's check
// mark.
func (a *App) cmdChecked(win *mygo.Window, fn func(ed *editor.Editor) bool) bool {
	if ed := a.Editor(win); ed != nil {
		return fn(ed)
	}
	return false
}

// openFolder asks for a folder and shows it in the file panel, as Typora's
// 打开文件夹 does: the panel lists the Markdown files under it.
func (a *App) openFolder(_ *mygo.MenuItem, win *mygo.Window) {
	if win == nil {
		win = mygo.FocusedWindow()
	}
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent:    win,
			Title:     locales.OpenFolderTitle(),
			Directory: true,
			Message:   locales.ChooseFolder(),
		})
		if err != nil || len(paths) == 0 {
			return
		}
		a.shared.SetFolder(paths[0])
		if win != nil {
			win.Update(func() {})
		}
	}()
}

// lastPath is the file the window in front shows, or the last one that had
// a window, for the session to open again at the next start.
func (a *App) lastPath() string {
	if ed := a.Editor(nil); ed != nil {
		if p := ed.Path(); p != "" {
			return p
		}
	}
	return ""
}

// The files of the panel are the app's to change on the disk: renaming
// brings the windows that show the file along, deleting moves it to the
// trash and closes those windows, and the hits of the folder's search open
// at the line they found.

// renameFile renames a file or a folder of the panel on the disk, and
// brings the windows that show it along: what was written there but not
// saved goes to the new path, and each window takes the file again there.
// A name the folder has already says so, and nothing moves.
func (a *App) renameFile(oldPath, newPath string) error {
	if _, err := os.Stat(newPath); err == nil {
		return fmt.Errorf("%s", locales.NameExists())
	}
	if err := os.Rename(oldPath, newPath); err != nil {
		return err
	}
	a.mu.Lock()
	docs := a.docs
	a.mu.Unlock()
	if docs == nil {
		return nil
	}
	var first error
	for _, d := range docs.Documents() {
		st := d.State()
		np := renamedPath(st.Path, oldPath, newPath)
		if np == "" {
			continue
		}
		if st.Edited {
			// What was written reaches the new path before the window
			// closes, so that nothing is left behind on the old one.
			if err := os.WriteFile(np, []byte(d.Content()), 0o644); err != nil && first == nil {
				first = err
			}
		}
		// The document now reads as its file does: the window closes
		// without asking, and the file opens again at its new name.
		d.SetEdited(false)
		w := d.Window()
		if w != nil {
			w.OnClosed(func() { a.openFile(np) })
			w.Close()
		}
	}
	return first
}

// renamedPath is where a document goes when the file or the folder it was
// read from was renamed, "" when it was not touched.
func renamedPath(path, oldPath, newPath string) string {
	if path == oldPath {
		return newPath
	}
	if strings.HasPrefix(path, oldPath+string(os.PathSeparator)) {
		return newPath + path[len(oldPath):]
	}
	return ""
}

// deleteFile moves a file or a folder of the panel to the trash of the
// system, and closes the windows that show it. A document with something
// written but unsaved keeps the delete from happening, so that what was
// written is not thrown away with the file.
func (a *App) deleteFile(path string) error {
	a.mu.Lock()
	docs := a.docs
	a.mu.Unlock()
	if docs != nil {
		for _, d := range docs.Documents() {
			st := d.State()
			if st.Edited && underPath(st.Path, path) {
				return fmt.Errorf("%s", locales.UnsavedBlock())
			}
		}
	}
	if err := a.trash(path); err != nil {
		return err
	}
	if docs != nil {
		for _, d := range docs.Documents() {
			if underPath(d.State().Path, path) {
				if w := d.Window(); w != nil {
					w.Close()
				}
			}
		}
	}
	return nil
}

// underPath tells whether path is dir itself or a file of it.
func underPath(path, dir string) bool {
	return path == dir || strings.HasPrefix(path, dir+string(os.PathSeparator))
}

// moveToTrash moves a file or a folder to the wastebasket of the system,
// from which it can be brought back.
func moveToTrash(path string) error {
	switch runtime.GOOS {
	case "darwin":
		script := fmt.Sprintf(`tell application "Finder" to delete POSIX file %s`, strconv.Quote(path))
		return exec.Command("osascript", "-e", script).Run()
	case "windows":
		st, err := os.Stat(path)
		if err != nil {
			return err
		}
		p := strings.ReplaceAll(path, "'", "''")
		verb := "DeleteFile"
		if st.IsDir() {
			verb = "DeleteDirectory"
		}
		ps := fmt.Sprintf("Add-Type -AssemblyName Microsoft.VisualBasic; [Microsoft.VisualBasic.FileIO.FileSystem]::%s('%s', 'OnlyErrorDialogs', 'SendToRecycleBin')", verb, p)
		return exec.Command("powershell", "-NoProfile", "-Command", ps).Run()
	}
	return fmt.Errorf("%s", "trash: this platform is not written yet")
}

// openFileAt opens a file and brings a line of it to view, as a hit of the
// folder's search does.
func (a *App) openFileAt(path string, line int) {
	a.mu.Lock()
	docs := a.docs
	a.mu.Unlock()
	if docs == nil {
		return
	}
	d, err := docs.Open(path)
	if err != nil {
		log.Printf("open %s: %v", path, err)
		return
	}
	if w := d.Window(); w != nil {
		a.mu.Lock()
		ed := a.wins[w]
		a.mu.Unlock()
		if ed != nil {
			ed.JumpTo(line)
		}
	}
}
