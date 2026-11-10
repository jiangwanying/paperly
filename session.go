package main

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"paperly/internal/doc"
	"paperly/internal/editor"
)

// windowFor creates the window that shows a document, and the editor in it.
// MyGo sets the window's title, its represented file and its edited state,
// asks to save before it closes, and closes the document with it.
func (a *App) windowFor(d *mygo.Document[string]) *mygo.Window {
	ed := editor.New(doc.New(d.Content()), a.shared)
	win := mygo.NewWindow(mygo.WindowOptions{
		Width:     900,
		Height:    900,
		MinWidth:  520,
		MinHeight: 420,
		// Every document's window opens where the last one was.
		StateKey: "document",
		Content:  ui.View(ed.View),
	})
	ed.Attach(win, d)
	a.register(win, ed)

	win.OnFocus(func() {
		a.mu.Lock()
		a.last = win
		a.mu.Unlock()
	})
	win.OnClosed(func() { a.unregister(win) })

	// Reverted, or changed by another app: show the file's text.
	d.OnReload(func() {
		ed.Reload(d.Content())
		win.Invalidate()
	})
	d.OnStateChange(func(mygo.DocumentState) { win.Invalidate() })
	return win
}
