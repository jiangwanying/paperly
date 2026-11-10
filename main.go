package main

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"
	// The interface's own strings, menus included, in 简体中文.
	_ "github.com/egoist/mygo/locales/zh-Hans"
)

const sample = `# 让光标所在的那一块，变回 Markdown

Typora 的核心不是「所见即所得」，是**块级编辑**：光标进到哪里，哪里就露出源码。

## 这份骨架要验证的事

- 点击任意段落，它当场变成可编辑的源码
- 按 Esc 提交，重新解析，块重新渲染
- 中文输入法、撤销、选区都交给 MyGo 的 TextArea

> 先跑通手感，再谈表格、公式、导出。

` + "```go" + `
func main() {
	// 光标在这一块里，它就是源码
}
` + "```" + `

| 能力 | 谁负责 |
| --- | --- |
| 解析 | ui/markdown/parser |
| 渲染 | markdown.View |
| 编辑 | ui.TextArea |
`

// adoptOldSettings copies the settings the app wrote under its older name
// into the folder it reads now, the first time it starts: the theme, the
// panel and the folder are the user's, not the name's.
func adoptOldSettings(path string) {
	if _, err := os.Stat(path); err == nil {
		return
	}
	dir := filepath.Dir(path)
	base := filepath.Dir(dir)
	for _, old := range []string{"mdpad Dev", "mdpad"} {
		from := filepath.Join(base, old)
		b, err := os.ReadFile(filepath.Join(from, "settings.json"))
		if err != nil {
			continue
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return
		}
		if err := os.WriteFile(path, b, 0o644); err != nil {
			return
		}
		// The files opened lately go with them.
		if r, err := os.ReadFile(filepath.Join(from, "recent-documents.json")); err == nil {
			_ = os.WriteFile(filepath.Join(dir, "recent-documents.json"), r, 0o644)
		}
		return
	}
}

func main() {
	settingsPath := ""
	if dir, err := mygo.App.Path(mygo.PathUserData); err == nil {
		settingsPath = filepath.Join(dir, "settings.json")
		adoptOldSettings(settingsPath)
	}
	set := LoadSettings(settingsPath)
	// The app is written in Chinese first, so that every string MyGo shows
	// of its own — the menu roles, the dialogs, the text menu — is Chinese
	// too, and a reader of another language still finds it in English.
	mygo.App.SetLanguages("zh-Hans", "en")
	if set.Language != "" {
		mygo.App.SetLocale(set.Language)
	} else {
		// Follow the system, which reads Chinese here and falls back to it
		// where it matches neither language.
		mygo.App.SetLocale("")
	}

	a := NewApp(set, settingsPath)
	// A file the launch carries opens first of all: the double click of a
	// .md in Finder, when the system says Paperly opens such files, and
	// the path a command line or another program hands over.
	var opened []string
	for _, arg := range os.Args[1:] {
		st, err := os.Stat(arg)
		if err != nil || st.IsDir() {
			continue
		}
		switch strings.ToLower(filepath.Ext(arg)) {
		case ".md", ".markdown":
			opened = append(opened, arg)
		}
	}
	// The file of the last session opens first, when the settings say so and
	// it is still there, and the launch carries no file of its own: an
	// untitled document would be left beside it otherwise.
	restore := len(opened) == 0 && set.RestoreLast && set.LastFile != ""
	if restore {
		if _, err := os.Stat(set.LastFile); err != nil {
			restore = false
		}
	}
	a.docs = mygo.NewDocuments(mygo.DocumentsOptions[string]{
		Types: []mygo.DocumentType{
			{Name: "Markdown Document", Extensions: []string{"md", "markdown"}},
		},
		New: func() string { return "" },
		Read: func(r io.Reader) (string, error) {
			b, err := io.ReadAll(r)
			return string(b), err
		},
		Write: func(w io.Writer, text string) error {
			_, err := io.WriteString(w, text)
			return err
		},
		// Typing a character and deleting it leaves the document unedited.
		Equal:  func(a, b string) bool { return a == b },
		Window: a.windowFor,
		// Typora saves when told to; the recovery copies still work.
		Autosave:      mygo.AutosaveOff,
		AutosaveDelay: 2 * time.Second,
		AskOnQuit:     true,
		NoUntitled:    restore,
	})
	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) {
		// The file in front is the one to open again at the next start.
		if p := a.lastPath(); p != "" {
			a.set.LastFile = p
		}
		_ = a.saver.flush()
	})
	mygo.App.WhenReady(func() {
		mygo.App.SetMenu(a.menu())
		for _, p := range opened {
			a.openFile(p)
		}
		if restore {
			a.openFile(set.LastFile)
		}
	})
	// A .md the system asks the app to open — a double click in the Finder
	// once Paperly is the default of Markdown, or an Open With — takes a
	// window of its own.
	mygo.App.OnOpenFile(func(path string) {
		a.openFile(path)
	})
	// The file list of the panel shows a file in the window it is in.
	a.shared.Open = a.switchTo
	a.shared.OpenAt = func(_ *mygo.Window, path string, line int) { a.openFileAt(path, line) }
	a.shared.Rename = a.renameFile
	a.shared.Delete = a.deleteFile

	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
