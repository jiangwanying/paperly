package editor

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"paperly/locales"
)

// The files of the panel answer to the writer: a file or a folder can be
// made, renamed and deleted where the tree shows it, and the folder can be
// searched for the files and the lines that hold a word. The disk work is
// the app's (Shared.Rename and Shared.Delete bring the windows that show a
// file along); what here is the panel's is the asking, the inline editing
// of a name, and the reading of the folder for a search.

// TreeEdit is a name being written in the panel: renaming the file or the
// folder at path, or making one in parent, a folder when folder says so.
// The panel holds one of its own; the exported shape lets a test set one.
type TreeEdit struct {
	Path   string // what is being renamed, "" while making one
	Parent string // where a new one goes
	Folder bool   // a folder is being made
	Name   string // the name as it is being typed
}

// newRows are the "＋" rows at the bottom of the tree: one makes a file,
// one makes a folder, both of the folder the panel shows.
func (e *Editor) newRows(c *ui.Context, t *ui.Theme) {
	for _, r := range []struct {
		label  string
		folder bool
	}{
		{locales.NewFile(), false},
		{locales.NewFolder(), true},
	} {
		r := r
		row := ui.Column(c).Padding(4, 18)
		if row.Hovered() {
			row.Background(t.SurfaceHover)
		}
		row.OnClick(func() { e.tedit = TreeEdit{Parent: e.folder(), Folder: r.folder} })
		row.Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
				ui.Text(c, "＋").FontSize(e.sideSize(-1)).TextColor(t.TextMuted)
				ui.Text(c, r.label).FontSize(e.sideSize(-1)).TextColor(t.TextMuted)
			})
		})
	}
}

// treeMenu is the menu of a row of the tree: what to make inside a folder,
// and the rename and the delete of whatever the row is.
func (e *Editor) treeMenu(m *ui.Menu, path string, dir bool) {
	if dir {
		m.Submenu(locales.Insert(), func(s *ui.Menu) {
			if s.Item(locales.NewFile()).Chosen() {
				e.tedit = TreeEdit{Parent: path}
			}
			if s.Item(locales.NewFolder()).Chosen() {
				e.tedit = TreeEdit{Parent: path, Folder: true}
			}
		})
	}
	if m.Item(locales.Rename()).Chosen() {
		e.tedit = TreeEdit{Path: path, Name: filepath.Base(path)}
	}
	if m.Item(locales.Delete()).Chosen() {
		e.deletePath(path)
	}
}

// commitTreeEdit applies the name the panel is editing: the file or the
// folder is made, or the one being renamed gets its new name. An error
// says why not, as a name the folder has already does.
func (e *Editor) commitTreeEdit() {
	te := e.tedit
	e.tedit = TreeEdit{}
	name := strings.TrimSpace(te.Name)
	if name == "" {
		return
	}
	name = strings.ReplaceAll(name, "/", "")
	if te.Path != "" {
		// A rename keeps the extension the file had, as Finder does when
		// the writer leaves it off.
		if !te.Folder && filepath.Ext(name) == "" {
			name += filepath.Ext(te.Path)
		}
		np := filepath.Join(filepath.Dir(te.Path), name)
		if err := e.sh.Rename(te.Path, np); err != nil {
			e.panelError(err)
		}
		return
	}
	full := filepath.Join(te.Parent, name)
	if !te.Folder {
		ext := filepath.Ext(full)
		if ext == "" {
			full += ".md"
		} else if ext != ".md" && ext != ".markdown" {
			full += ".md"
		}
	}
	if _, err := os.Stat(full); err == nil {
		e.panelError(fmt.Errorf("%s", locales.NameExists()))
		return
	}
	if te.Folder {
		if err := os.MkdirAll(full, 0o755); err != nil {
			e.panelError(err)
		}
		return
	}
	if err := os.WriteFile(full, []byte(""), 0o644); err != nil {
		e.panelError(err)
		return
	}
	if e.sh.Open != nil {
		e.sh.Open(e.win, full)
	}
}

// deletePath asks twice and moves a file or a folder of the panel to the
// trash: the dialog says what goes, and says again what would go with it.
func (e *Editor) deletePath(path string) {
	if !e.askDelete(filepath.Base(path)) {
		return
	}
	if err := e.sh.Delete(path); err != nil {
		e.panelError(err)
	}
}

// askDelete asks whether the named file or folder goes to the trash, and
// reports the answer. The dialog asks; a test may answer in its place.
func (e *Editor) askDelete(name string) bool {
	if e.testAsk != nil {
		return e.testAsk(name)
	}
	r, err := mygo.Dialog.Message(mygo.MessageOptions{
		Title:              locales.ConfirmDelete(),
		Message:            name,
		Detail:             locales.ConfirmDeleteDetail(name),
		Buttons:            []string{locales.Cancel(), locales.Delete()},
		DefaultButton:      0,
		CancelButton:       0,
		DestructiveButtons: []int{1},
	})
	return err == nil && r.Button == 1
}

// panelError says why a change to the files did not happen. A test may
// take the error instead of the dialog.
func (e *Editor) panelError(err error) {
	if e.testAlert != nil {
		e.testAlert(err)
		return
	}
	go func() {
		_, _ = mygo.Dialog.Message(mygo.MessageOptions{
			Title:   locales.OpFailed(),
			Message: err.Error(),
			Buttons: []string{locales.Ok()},
		})
	}()
}

// editRow is the row that edits a name: the field replaces the label of
// the row being renamed, or stands in for the new file or folder.
func (e *Editor) editRow(c *ui.Context, t *ui.Theme) {
	in := ui.TextInput(c.Key("tree-name"), &e.tedit.Name).AutoFocus()
	in.FontSize(e.sideSize(-1))
	in.Padding(2, 6)
	in.Background(t.Background)
	in.Border(1, t.Border)
	in.FocusRingColor(t.Text)
	in.OnSubmit(func() { e.commitTreeEdit() })
	in.OnTextKey(func(k ui.TextKey) bool {
		if k.Key == ui.KeyEscape {
			e.tedit = TreeEdit{}
			return true
		}
		return false
	})
}

// ---------- the search of the folder ----------

// searchLine is one line of a file that holds what the panel searches for.
type searchLine struct {
	no   int
	text string
}

// searchHit is a file of the folder that holds it, with the lines.
type searchHit struct {
	path  string
	name  string
	lines []searchLine
}

// searchFor looks the folder's Markdown files through for what the panel
// searches: a file matches by its name, or by the lines that hold it. The
// work is kept small: a file beyond a megabyte is skipped, and so is one
// that holds a zero byte, for those are not writings.
func (e *Editor) searchFor(q string) []searchHit {
	dir := e.folder()
	if dir == "" || q == "" {
		return nil
	}
	// The same question within a second reads the folder once.
	if q == e.searchQ && dir == e.searchDir && time.Since(e.searchWhen) < time.Second {
		return e.searchRes
	}
	q = strings.ToLower(q)
	var out []searchHit
	var walk func(dir string, depth int)
	walk = func(dir string, depth int) {
		if depth > 6 || len(out) >= 40 {
			return
		}
		ents, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		sort.Slice(ents, func(i, j int) bool { return ents[i].Name() < ents[j].Name() })
		for _, ent := range ents {
			if len(out) >= 40 {
				return
			}
			name := ent.Name()
			if strings.HasPrefix(name, ".") {
				continue
			}
			path := filepath.Join(dir, name)
			if ent.IsDir() {
				walk(path, depth+1)
				continue
			}
			if ext := strings.ToLower(filepath.Ext(name)); ext != ".md" && ext != ".markdown" {
				continue
			}
			fh, err := os.Open(path)
			if err != nil {
				continue
			}
			st, _ := ent.Info()
			text := ""
			if st != nil && st.Size() < 1<<20 {
				b := make([]byte, st.Size())
				n, _ := fh.Read(b)
				text = string(b[:n])
			}
			fh.Close()
			if strings.ContainsRune(text[:min(len(text), 4096)], 0) {
				continue
			}
			byName := strings.Contains(strings.ToLower(name), q)
			var lines []searchLine
			for i, line := range strings.Split(text, "\n") {
				if len(lines) >= 3 {
					break
				}
				if strings.Contains(strings.ToLower(line), q) {
					lines = append(lines, searchLine{no: i + 1, text: strings.TrimSpace(line)})
				}
			}
			if byName || len(lines) > 0 {
				out = append(out, searchHit{path: path, name: name, lines: lines})
			}
		}
	}
	walk(dir, 0)
	e.searchQ, e.searchDir, e.searchWhen, e.searchRes = q, dir, time.Now(), out
	return out
}

// searchResults shows what the folder holds for the word: each file with
// its lines, the word loud in them; a line click opens the file there.
func (e *Editor) searchResults(c *ui.Context, t *ui.Theme) {
	hits := e.searchFor(e.filter)
	if len(hits) == 0 {
		msg := locales.SearchNone()
		ui.Text(c, msg).FontSize(e.sideSize(-3)).TextColor(t.TextMuted).Padding(6, 18)
		return
	}
	ui.Text(c, locales.SearchFilesN(len(hits))).FontSize(e.sideSize(-3)).
		TextColor(t.TextMuted).Padding(4, 18)
	for _, h := range hits {
		h := h
		head := ui.Column(c).Padding(4, 18)
		if head.Hovered() {
			head.Background(t.SurfaceHover)
		}
		head.OnClick(func() {
			if e.sh.Open != nil {
				e.sh.Open(e.win, h.path)
			}
		})
		head.Children(func() {
			el := ui.Text(c, h.name).FontSize(e.sideSize(0)).Bold()
			el.TextColor(t.Text)
		})
		for _, l := range h.lines {
			l := l
			row := ui.Column(c).Padding(3, 18)
			if row.Hovered() {
				row.Background(t.SurfaceHover)
			}
			row.OnClick(func() {
				if e.sh.OpenAt != nil {
					e.sh.OpenAt(e.win, h.path, l.no)
				}
			})
			row.Children(func() {
				ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
					ui.Text(c, fmt.Sprint(l.no)).FontSize(e.sideSize(-2)).TextColor(t.TextMuted)
					e.searchLineText(c, t, l.text)
				})
			})
		}
	}
}

// searchLineText draws a line of a hit with the word the panel searches
// for in the text's own color and the rest of it quieter.
func (e *Editor) searchLineText(c *ui.Context, t *ui.Theme, line string) {
	q := strings.ToLower(e.filter)
	ui.Row(c).AlignItems(ui.Center).Children(func() {
		for {
			i := strings.Index(strings.ToLower(line), q)
			if i < 0 {
				if line != "" {
					ui.Text(c, line).FontSize(e.sideSize(-2)).TextColor(t.TextMuted)
				}
				return
			}
			if i > 0 {
				ui.Text(c, line[:i]).FontSize(e.sideSize(-2)).TextColor(t.TextMuted)
			}
			ui.Text(c, line[i:i+len(e.filter)]).FontSize(e.sideSize(-2)).TextColor(t.Text).Bold()
			line = line[i+len(e.filter):]
		}
	})
}

// on tells whether the panel is editing a name of the tree.
func (t TreeEdit) on() bool { return t.Path != "" || t.Parent != "" }

// The rest of the panel's file operations, for the tests of the app
// package to drive without the dialogs of the system.

// SetTreeEdit puts a name being written in the panel.
func (e *Editor) SetTreeEdit(te TreeEdit) { e.tedit = te }

// SetTreeName is the name as it is being typed.
func (e *Editor) SetTreeName(name string) { e.tedit.Name = name }

// CommitTreeEdit applies the name being written.
func (e *Editor) CommitTreeEdit() { e.commitTreeEdit() }

// DeletePath asks about the delete, and asks the app for it when the
// answer is yes.
func (e *Editor) DeletePath(path string) { e.deletePath(path) }

// TestAsk answers the delete dialog the way a test decides.
func (e *Editor) TestAsk(fn func(string) bool) { e.testAsk = fn }

// TakeAlert takes the error dialog the panel would show.
func (e *Editor) TakeAlert(fn func(error)) { e.testAlert = fn }

// JumpTo brings a line of the document into view, as a hit of the folder's
// search does: the block that holds the line is the one revealed.
func (e *Editor) JumpTo(line int) {
	if line <= 0 {
		return
	}
	src, at := e.doc.Source(), 0
	for n := 1; n < line && at <= len(src); n++ {
		if i := strings.IndexByte(src[at:], '\n'); i >= 0 {
			at += i + 1
		} else {
			break
		}
	}
	if idx := e.doc.BlockAt(at); idx >= 0 {
		e.scrollTo = idx
	}
}
