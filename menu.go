package main

import (
	"paperly/locales"
	"runtime"

	"github.com/egoist/mygo"

	"paperly/internal/doc"
	"paperly/internal/editor"
)

// menu is the menu bar: 文件 编辑 段落 格式 视图 帮助, in Chinese.
//
// File and Edit are MyGo's roles: they act on the window's document and its
// text input, and MyGo labels them in the interface's language. The rest are
// the editor's own commands; the ones whose feature is not written yet are
// disabled and keep their IDs, so that the shape of the menu does not change
// as they come.
func (a *App) menu() *mygo.Menu {
	return mygo.NewMenu(a.menuItems())
}

// menuItems builds the items of the menu bar, in order.
func (a *App) menuItems() []*mygo.MenuItem {
	file := []*mygo.MenuItem{
		{Role: mygo.RoleNew},
		{Role: mygo.RoleOpen},
		{ID: "file.folder", Label: locales.OpenFolder(), Accelerator: "Shift+CmdOrCtrl+O", Click: a.openFolder},
		{Role: mygo.RoleOpenRecent},
		mygo.Separator(),
		{Role: mygo.RoleSave},
		{Role: mygo.RoleSaveAs, Accelerator: "Shift+CmdOrCtrl+S"},
		{Role: mygo.RoleRevert},
		mygo.Separator(),
		{Role: mygo.RoleRename},
		{Role: mygo.RoleMoveTo},
		mygo.Separator(),
		{Role: mygo.RoleClose},
	}
	file = append(file, a.fileTail(runtime.GOOS == "darwin")...)
	return []*mygo.MenuItem{
		// The application menu, with the preferences where a Mac app puts
		// them: under its name, on ⌘,.
		{Role: mygo.RoleAppMenu, Submenu: []*mygo.MenuItem{
			{Role: mygo.RoleAbout},
			mygo.Separator(),
			{ID: "app.prefs", Label: locales.Settings(), Accelerator: "CmdOrCtrl+,", Click: a.openSettings},
			mygo.Separator(),
			{Role: mygo.RoleServices},
			mygo.Separator(),
			{Role: mygo.RoleHide},
			{Role: mygo.RoleHideOthers},
			{Role: mygo.RoleUnhide},
			mygo.Separator(),
			{Role: mygo.RoleQuit},
		}},
		{Label: locales.MenuFile(), Submenu: file},
		{Label: locales.MenuEdit(), Submenu: []*mygo.MenuItem{
			// The window's own undo, not the text input's: a block is
			// written into the document as it is typed, so the input's
			// stack knows nothing of what the document was.
			{ID: "edit.undo", Label: locales.Undo(), Accelerator: "CmdOrCtrl+Z", Click: a.do(func(ed *editor.Editor) { ed.Undo() })},
			{ID: "edit.redo", Label: locales.Redo(), Accelerator: "Shift+CmdOrCtrl+Z", Click: a.do(func(ed *editor.Editor) { ed.Redo() })},
			mygo.Separator(),
			{Role: mygo.RoleCut},
			{Role: mygo.RoleCopy},
			{Role: mygo.RolePaste},
			{ID: "edit.pasteImage", Label: locales.PasteImage(), Accelerator: "Shift+CmdOrCtrl+V", Click: a.do(func(ed *editor.Editor) { ed.PasteImage() })},
			{Role: mygo.RoleDelete},
			{Role: mygo.RoleSelectAll},
			mygo.Separator(),
			{ID: "find", Label: locales.Find(), Accelerator: "CmdOrCtrl+F", Click: a.do(func(ed *editor.Editor) { ed.OpenFind(false) })},
			{ID: "find.next", Label: locales.FindNext(), Accelerator: "CmdOrCtrl+G", Click: a.do(func(ed *editor.Editor) { ed.FindNext() })},
			{ID: "find.prev", Label: locales.FindPrev(), Accelerator: "Shift+CmdOrCtrl+G", Click: a.do(func(ed *editor.Editor) { ed.FindPrev() })},
			{ID: "find.replace", Label: locales.Replace(), Accelerator: "Alt+CmdOrCtrl+F", Click: a.do(func(ed *editor.Editor) { ed.OpenFind(true) })},
		}},
		{Label: locales.MenuParagraph(), Submenu: []*mygo.MenuItem{
			{ID: "para.h1", Label: locales.Heading1(), Accelerator: "CmdOrCtrl+1", Click: a.do(func(ed *editor.Editor) { ed.SetHeading(1) })},
			{ID: "para.h2", Label: locales.Heading2(), Accelerator: "CmdOrCtrl+2", Click: a.do(func(ed *editor.Editor) { ed.SetHeading(2) })},
			{ID: "para.h3", Label: locales.Heading3(), Accelerator: "CmdOrCtrl+3", Click: a.do(func(ed *editor.Editor) { ed.SetHeading(3) })},
			{ID: "para.h4", Label: locales.Heading4(), Accelerator: "CmdOrCtrl+4", Click: a.do(func(ed *editor.Editor) { ed.SetHeading(4) })},
			{ID: "para.h5", Label: locales.Heading5(), Accelerator: "CmdOrCtrl+5", Click: a.do(func(ed *editor.Editor) { ed.SetHeading(5) })},
			{ID: "para.h6", Label: locales.Heading6(), Accelerator: "CmdOrCtrl+6", Click: a.do(func(ed *editor.Editor) { ed.SetHeading(6) })},
			{ID: "para.p", Label: locales.Body(), Accelerator: "CmdOrCtrl+0", Click: a.do(func(ed *editor.Editor) { ed.SetHeading(0) })},
			mygo.Separator(),
			{ID: "para.up", Label: locales.PromoteHeading(), Accelerator: "CmdOrCtrl+Plus", Click: a.do(func(ed *editor.Editor) { ed.PromoteHeading() })},
			{ID: "para.down", Label: locales.DemoteHeading(), Accelerator: "CmdOrCtrl+-", Click: a.do(func(ed *editor.Editor) { ed.DemoteHeading() })},
			mygo.Separator(),
			{ID: "para.quote", Label: locales.Quote(), Accelerator: "Alt+CmdOrCtrl+Q", Click: a.do(func(ed *editor.Editor) { ed.ToggleQuote() })},
			{ID: "para.ul", Label: locales.BulletList(), Accelerator: "Alt+CmdOrCtrl+U", Click: a.do(func(ed *editor.Editor) { ed.ToggleList(doc.ListBullet) })},
			{ID: "para.ol", Label: locales.OrderedList(), Accelerator: "Alt+CmdOrCtrl+O", Click: a.do(func(ed *editor.Editor) { ed.ToggleList(doc.ListOrdered) })},
			{ID: "para.task", Label: locales.TaskList(), Accelerator: "Alt+CmdOrCtrl+X", Click: a.do(func(ed *editor.Editor) { ed.ToggleList(doc.ListTask) })},
			mygo.Separator(),
			{ID: "para.code", Label: locales.CodeBlock(), Accelerator: "Alt+CmdOrCtrl+C", Click: a.do(func(ed *editor.Editor) { ed.ToggleCode() })},
			{ID: "para.table", Label: locales.Table(), Accelerator: "Alt+CmdOrCtrl+T", Disabled: true},
		}},
		{Label: locales.MenuFormat(), Submenu: []*mygo.MenuItem{
			{ID: "fmt.bold", Label: locales.Bold(), Accelerator: "CmdOrCtrl+B", Click: a.do(func(ed *editor.Editor) { ed.Bold() })},
			{ID: "fmt.italic", Label: locales.Italic(), Accelerator: "CmdOrCtrl+I", Click: a.do(func(ed *editor.Editor) { ed.Italic() })},
			{ID: "fmt.code", Label: locales.InlineCode(), Accelerator: "CmdOrCtrl+E", Click: a.do(func(ed *editor.Editor) { ed.InlineCode() })},
			{ID: "fmt.strike", Label: locales.Strike(), Accelerator: "Shift+CmdOrCtrl+X", Click: a.do(func(ed *editor.Editor) { ed.Strike() })},
			{ID: "fmt.underline", Label: locales.Underline(), Accelerator: "CmdOrCtrl+U", Click: a.do(func(ed *editor.Editor) { ed.Underline() })},
			{ID: "fmt.highlight", Label: locales.Highlight(), Accelerator: "Shift+CmdOrCtrl+H", Click: a.do(func(ed *editor.Editor) { ed.Highlight() })},
			mygo.Separator(),
			{ID: "fmt.link", Label: locales.Link(), Accelerator: "CmdOrCtrl+K", Click: a.do(func(ed *editor.Editor) { ed.Link() })},
			mygo.Separator(),
			{ID: "fmt.clear", Label: locales.ClearStyle(), Accelerator: "CmdOrCtrl+\\", Click: a.do(func(ed *editor.Editor) { ed.ClearStyle() })},
			mygo.Separator(),
			{ID: "fmt.image", Label: locales.Image(), Accelerator: "Shift+CmdOrCtrl+I", Click: a.do(func(ed *editor.Editor) { ed.ChooseImage() })},
		}},
		{Label: locales.MenuView(), Submenu: []*mygo.MenuItem{
			{ID: "view.sidebar", Label: locales.Sidebar(), Accelerator: "Shift+CmdOrCtrl+L", Click: a.do(func(ed *editor.Editor) { ed.ToggleSidebar() })},
			mygo.Separator(),
			// Typora's three panels, on its shortcuts.
			{ID: "view.outline", Label: locales.Outline(), Accelerator: "Shift+CmdOrCtrl+1", Click: a.do(func(ed *editor.Editor) { ed.ShowSide(editor.SideOutline) })},
			{ID: "view.tree", Label: locales.FileTree(), Accelerator: "Shift+CmdOrCtrl+2", Click: a.do(func(ed *editor.Editor) { ed.ShowSide(editor.SideTree) })},
			{ID: "view.list", Label: locales.FileList(), Accelerator: "Shift+CmdOrCtrl+3", Click: a.do(func(ed *editor.Editor) { ed.ShowSide(editor.SideList) })},
			mygo.Separator(),
			{ID: "view.source", Label: locales.SourceMode(), Type: mygo.MenuItemCheckbox, Accelerator: "CmdOrCtrl+/",
				Click: func(it *mygo.MenuItem, win *mygo.Window) {
					a.cmd(win, func(ed *editor.Editor) {
						ed.ToggleSource()
						it.SetChecked(ed.Source())
					})
				}},
			mygo.Separator(),
			{ID: "view.focus", Label: locales.FocusMode(), Accelerator: "F8", Disabled: true},
			{ID: "view.typewriter", Label: locales.TypewriterMode(), Accelerator: "F9", Disabled: true},
			mygo.Separator(),
			{Role: mygo.RoleToggleFullScreen},
			mygo.Separator(),
			{Label: locales.Language(), Submenu: a.languageItems()},
		}},
		{Label: locales.MenuHelp(), Submenu: []*mygo.MenuItem{
			{ID: "help.about", Label: locales.About(), Click: about},
			{ID: "help.update", Label: locales.CheckUpdates(), Click: func(*mygo.MenuItem, *mygo.Window) { checkForUpdates() }},
			mygo.Separator(),
			{ID: "help.keys", Label: locales.Shortcuts(), Disabled: true},
		}},
	}
}

// languageItems lists the languages the interface shows in, the one in use
// checked.
func (a *App) languageItems() []*mygo.MenuItem {
	choose := func(id, label, tag string) *mygo.MenuItem {
		return &mygo.MenuItem{
			ID:      id,
			Label:   label,
			Type:    mygo.MenuItemRadio,
			Checked: a.set.Language == tag,
			Click: func(*mygo.MenuItem, *mygo.Window) {
				a.set.Language = tag
				mygo.App.SetLocale(tag)
				a.saver.touch()
			},
		}
	}
	return []*mygo.MenuItem{
		choose("lang.zh", locales.LangZh(), "zh-Hans"),
		choose("lang.en", locales.LangEn(), "en"),
		mygo.Separator(),
		choose("lang.system", locales.LangSystem(), ""),
	}
}

// about shows the app's name, its version and where it lives on GitHub;
// the buttons open the repository and run the update check.
func about(*mygo.MenuItem, *mygo.Window) {
	go func() {
		r, err := mygo.Dialog.Message(mygo.MessageOptions{
			Title:   "Paperly",
			Message: "Paperly " + version,
			Detail:  locales.AboutDetail() + "\n\n" + locales.GithubRepoLabel() + ": " + repoURL,
			Buttons: []string{locales.OpenGithub(), locales.CheckUpdates(), locales.Ok()},
		})
		if err != nil {
			return
		}
		switch r.Button {
		case 0:
			openURL(repoURL)
		case 1:
			checkForUpdates()
		}
	}()
}

// version is the build's, replaced when packaging.
var version = "0.1.2"

// canFormat reports whether a command that rewrites text has anywhere to
// write: the block being edited.
func canFormat(ed *editor.Editor) bool { return ed.Active() >= 0 }

// do turns a command on an editor into a menu item's click, run on the
// editor of the window the menu was chosen in.
func (a *App) do(fn func(ed *editor.Editor)) func(*mygo.MenuItem, *mygo.Window) {
	return func(_ *mygo.MenuItem, win *mygo.Window) { a.cmd(win, fn) }
}

// openSettings is the menu item that shows the preferences window.
func (a *App) openSettings(_ *mygo.MenuItem, win *mygo.Window) { a.OpenSettings(win) }

// fileTail is what the file menu holds after the document commands. On
// macOS the preferences and quitting live in the application menu, which is
// the platform's way; Windows and Linux have no such menu — MyGo hides the
// role there — so both go at the end of Edit's neighbour, as their apps
// put them.
func (a *App) fileTail(mac bool) []*mygo.MenuItem {
	if mac {
		return nil
	}
	return []*mygo.MenuItem{
		mygo.Separator(),
		{ID: "file.prefs", Label: locales.Settings(), Accelerator: "CmdOrCtrl+,", Click: a.openSettings},
		{Role: mygo.RoleQuit},
	}
}
