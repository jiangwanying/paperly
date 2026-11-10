package editor

import (
	"os"
	"paperly/locales"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown/parser"
)

// fileItem is one Markdown file of the folder the document is in.
type fileItem struct {
	name string    // the file's name, extension and all
	path string    // where it is
	mod  time.Time // when it last changed
	line string    // the first line of it, for the list to show
}

// sidebar is the panel at the left, as Typora's: a bar at the top with the
// button that switches between the document's files and its outline, the
// name of the panel, and the search that filters it; then the headings or
// the files.
func (e *Editor) sidebar(c *ui.Context, t *ui.Theme) {
	title, other := locales.Outline(), e.sh.Files()
	if e.sh.Side != SideOutline {
		title, other = locales.Documents(), SideOutline
	}
	ui.Column(c).Grow(1).
		Background(t.Surface).
		Children(func() {
			ui.Row(c).Height(40).AlignItems(ui.Center).Children(func() {
				if e.filtering {
					e.filterBar(c, t)
				} else {
					// The left button switches files and outline, as
					// Typora's does; the right one searches what shows.
					e.iconButton(c, t, iconList, func() {
						e.filter = ""
						e.sh.Show(other)
					})
					ui.Row(c).Grow(1).Justify(ui.Center).Children(func() {
						ui.Text(c, title).FontSize(e.sideSize(-1)).TextColor(t.TextMuted)
					})
					e.searchButton(c, t)
				}
			})
			ui.Scroll(c).Grow(1).Children(func() {
				// Typora pads the outline 3px 18px, and its rows bleed to
				// the panel's edges.
				ui.Column(c).Padding(3, 0).Children(func() {
					switch e.sh.Side {
					case SideOutline:
						e.outline(c, t)
					case SideTree:
						if e.filter != "" {
							e.searchResults(c, t)
						} else {
							e.tree(c, t)
						}
					default:
						e.fileList(c, t)
					}
				})
			})
		})
}

// The icons of the panel's bar, drawn as SVG paths so that their strokes
// are as fine and as even as the system's own.
var (
	iconList   = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M8 6h13M8 12h13M8 18h13M3 6h.01M3 12h.01M3 18h.01"/></svg>`))
	iconSearch = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="7"/><path d="m21 21-4.3-4.3"/></svg>`))
	iconBack   = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m15 18-6-6 6-6"/></svg>`))
)

// sideSize is the panel's text, n steps from the size the settings keep: -1
// for what is quieter, -3 for the smallest.
func (e *Editor) sideSize(n float32) float32 {
	return max(e.sh.SideFont+n, 9)
}

// iconButton is one of the two buttons at the ends of the panel's bar.
func (e *Editor) iconButton(c *ui.Context, t *ui.Theme, icon *ui.SVG, fn func()) {
	b := ui.Row(c).Width(40).Height(40).Justify(ui.Center).AlignItems(ui.Center)
	if b.Hovered() {
		b.Background(t.SurfaceHover)
	}
	b.OnClick(fn)
	b.Children(func() { ui.Icon(c, icon).FontSize(e.sh.SideIcon).TextColor(t.TextMuted) })
}

// searchButton is the magnifier at the right of the bar.
func (e *Editor) searchButton(c *ui.Context, t *ui.Theme) {
	b := ui.Row(c).Width(40).Height(40).Justify(ui.Center).AlignItems(ui.Center)
	if b.Hovered() {
		b.Background(t.SurfaceHover)
	}
	b.OnClick(func() { e.filtering = true })
	b.Children(func() { ui.Icon(c, iconSearch).FontSize(e.sh.SideIcon).TextColor(t.TextMuted) })
}

// filterBar replaces the bar while the panel is searched: back, then the
// field that filters what shows.
func (e *Editor) filterBar(c *ui.Context, t *ui.Theme) {
	e.iconButton(c, t, iconBack, func() {
		e.filtering = false
		e.filter = ""
	})
	in := ui.TextInput(c.Key("filter"), &e.filter).AutoFocus()
	in.Background(ui.Color{})
	in.Border(0, ui.Color{})
	in.FocusRing(false)
	in.FontSize(e.sideSize(-2))
	in.Grow(1)
	in.Padding(0, 4)
	_ = t
}

// matches reports whether s holds what the panel is filtered by.
func (e *Editor) matches(s string) bool {
	return e.filter == "" || strings.Contains(strings.ToLower(s), strings.ToLower(e.filter))
}

// outlineNode is one heading of the outline, with the headings under it.
type outlineNode struct {
	block    int
	level    int
	text     string
	children []*outlineNode
}

// outline shows the headings of the document, nested by their level, the
// section being read in bold: a click brings one into view, and the arrow
// opens or closes the headings under it.
func (e *Editor) outline(c *ui.Context, t *ui.Theme) {
	nodes := e.outlineTree()
	if e.filter != "" {
		nodes = e.filterOutline(nodes)
	}
	if len(nodes) == 0 {
		msg := locales.NoHeadings()
		if e.filter != "" {
			msg = locales.NoMatchingHeadings()
		}
		ui.Text(c, msg).FontSize(e.sideSize(-3)).TextColor(t.TextMuted).Padding(6, 18)
		return
	}
	current := e.currentHeading()
	e.outlineNodes(c, t, nodes, 0, current)
}

// filterOutline keeps the headings that match the filter, and those above
// them that lead to one, as Typora keeps them while the outline is
// searched.
func (e *Editor) filterOutline(nodes []*outlineNode) []*outlineNode {
	var kept []*outlineNode
	for _, n := range nodes {
		children := e.filterOutline(n.children)
		if e.matches(n.text) || len(children) > 0 {
			c := *n
			c.children = children
			kept = append(kept, &c)
		}
	}
	return kept
}

func (e *Editor) outlineNodes(c *ui.Context, t *ui.Theme, nodes []*outlineNode, depth, current int) {
	for _, n := range nodes {
		n := n
		open := e.filter != "" || !e.folded[n.block]
		row := ui.Column(c).Padding(3, 18)
		if row.Hovered() {
			row.Background(t.SurfaceHover)
		}
		row.OnClick(func() { e.scrollTo = n.block })
		row.Children(func() {
			ui.Row(c).AlignItems(ui.Start).Children(func() {
				ui.Box(c).Width(float32(depth) * e.sh.SideFont)
				if len(n.children) > 0 && e.filter == "" {
					arrow := "▸"
					if open {
						arrow = "▾"
					}
					ui.Box(c).Width(4)
					el := ui.Text(c, arrow).FontSize(e.sideSize(-3)).TextColor(t.TextMuted).Width(e.sh.SideFont)
					el.OnClick(func() {
						if e.folded == nil {
							e.folded = map[int]bool{}
						}
						e.folded[n.block] = open
					})
				} else {
					ui.Box(c).Width(18)
				}
				el := ui.Text(c, n.text).FontSize(e.sideSize(0))
				switch {
				case e.filter != "" && e.matches(n.text):
					el.TextColor(t.Text).Bold()
				case n.block == current:
					el.TextColor(t.Text).Bold()
				default:
					el.TextColor(t.TextMuted)
				}
			})
		})
		if open {
			e.outlineNodes(c, t, n.children, depth+1, current)
		}
	}
}

// outlineTree nests the headings of the document by their level, as the
// outline shows them.
func (e *Editor) outlineTree() []*outlineNode {
	var roots, stack []*outlineNode
	for i, b := range e.doc.Blocks() {
		if b.Kind != parser.Heading {
			continue
		}
		n := &outlineNode{block: i, level: b.Level, text: headingText(e.doc.Source()[b.Start:b.End])}
		for len(stack) > 0 && stack[len(stack)-1].level >= n.level {
			stack = stack[:len(stack)-1]
		}
		if len(stack) == 0 {
			roots = append(roots, n)
		} else {
			p := stack[len(stack)-1]
			p.children = append(p.children, n)
		}
		stack = append(stack, n)
	}
	return roots
}

// currentHeading is the heading of the section at the top of the page, the
// one the outline marks while the document is read.
func (e *Editor) currentHeading() int {
	cur := -1
	for i, b := range e.doc.Blocks() {
		if b.Kind != parser.Heading {
			continue
		}
		top, ok := e.tops[i]
		if !ok {
			continue
		}
		if top <= e.scroll.Y+40 {
			cur = i
			continue
		}
		break
	}
	return cur
}

// tree shows the folder of the document as a tree of folders and Markdown
// files, as Typora's file tree does.
func (e *Editor) tree(c *ui.Context, t *ui.Theme) {
	root := e.folder()
	if root == "" {
		ui.Text(c, locales.SideFolderHint()).FontSize(e.sideSize(-3)).
			TextColor(t.TextMuted).Padding(6, 4)
		return
	}
	current := e.path()
	if e.tedit.on() && e.tedit.Path == "" {
		// A name is being written for a new file or folder of the root.
		ui.Row(c).Padding(4, 18).AlignItems(ui.Center).Children(func() {
			ui.Box(c).Width(12)
			e.editRow(c, t)
		})
	}
	for _, r := range e.treeRows(root) {
		r := r
		on := !r.dir && r.item.path == current
		renaming := e.tedit.on() && e.tedit.Path == r.item.path
		row := ui.Column(c).Padding(4, 18)
		if !renaming {
			row.ContextMenu(func(m *ui.Menu) { e.treeMenu(m, r.item.path, r.dir) })
		}
		switch {
		case on:
			row.Background(t.Text.Alpha(0.06))
		case row.Hovered():
			row.Background(t.SurfaceHover)
		}
		switch {
		case renaming:
			row.Children(func() {
				ui.Row(c).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Width(float32(r.depth) * 12)
					if r.dir {
						ui.Box(c).Width(18)
					} else {
						ui.Box(c).Width(12)
					}
					e.editRow(c, t)
				})
			})
			continue
		case r.dir:
			row.OnClick(func() {
				if e.collapsed == nil {
					e.collapsed = map[string]bool{}
				}
				e.collapsed[r.item.path] = !r.closed
			})
		default:
			row.OnClick(func() {
				if !on && e.sh.Open != nil {
					e.sh.Open(e.win, r.item.path)
				}
			})
		}
		row.Children(func() {
			ui.Row(c).AlignItems(ui.Start).Children(func() {
				ui.Box(c).Width(float32(r.depth) * 12)
				label := strings.TrimSuffix(r.item.name, filepath.Ext(r.item.name))
				if r.dir {
					arrow := "▸"
					if !r.closed {
						arrow = "▾"
					}
					ui.Text(c, arrow).FontSize(e.sideSize(-4)).TextColor(t.TextMuted)
					ui.Box(c).Width(4)
					label = r.item.name
				} else {
					ui.Box(c).Width(12)
				}
				el := ui.Text(c, label).FontSize(e.sideSize(-1))
				if on {
					el.TextColor(t.Text).Bold()
				} else {
					el.TextColor(t.TextMuted)
				}
			})
		})
	}
	e.newRows(c, t)
}

// treeRow is one line of the file tree: a folder, closed or open, or a file.
type treeRow struct {
	item   fileItem
	dir    bool
	closed bool
	depth  int
}

// treeRows walks a folder into the lines the tree shows, the folders it
// opened first and then its files, each by name.
func (e *Editor) treeRows(dir string) []treeRow {
	return e.walk(dir, 0)
}

func (e *Editor) walk(dir string, depth int) []treeRow {
	if depth > 6 {
		return nil
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var dirs, files []fileItem
	for _, ent := range ents {
		name := ent.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		path := filepath.Join(dir, name)
		if ent.IsDir() {
			dirs = append(dirs, fileItem{name: name, path: path})
			continue
		}
		switch strings.ToLower(filepath.Ext(name)) {
		case ".md", ".markdown":
			files = append(files, fileItem{name: name, path: path})
		}
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].name < dirs[j].name })
	sort.Slice(files, func(i, j int) bool { return files[i].name < files[j].name })

	var rows []treeRow
	for _, d := range dirs {
		closed := e.collapsed[d.path]
		rows = append(rows, treeRow{item: d, dir: true, closed: closed, depth: depth})
		if !closed {
			rows = append(rows, e.walk(d.path, depth+1)...)
		}
	}
	for _, f := range files {
		rows = append(rows, treeRow{item: f, depth: depth})
	}
	return rows
}

// fileList shows the Markdown files of the document's folder as a flat
// list, as Typora's file list does: the name, the day it changed, and the
// first line of it.
func (e *Editor) fileList(c *ui.Context, t *ui.Theme) {
	files := e.folderFiles()
	shown := 0
	current := e.path()
	for _, f := range files {
		if e.filter != "" && !e.matches(f.name) {
			continue
		}
		shown++
		f := f
		on := f.path == current
		row := ui.Column(c).Padding(6, 18)
		switch {
		case on:
			row.Background(t.Text.Alpha(0.06))
		case row.Hovered():
			row.Background(t.SurfaceHover)
		}
		row.OnClick(func() {
			if !on && e.sh.Open != nil {
				e.sh.Open(e.win, f.path)
			}
		})
		row.ContextMenu(func(m *ui.Menu) { e.treeMenu(m, f.path, false) })
		row.Children(func() {
			ui.Row(c).Children(func() {
				name := strings.TrimSuffix(f.name, filepath.Ext(f.name))
				el := ui.Text(c, name).FontSize(e.sideSize(0))
				if on {
					el.TextColor(t.Text).Bold()
				} else {
					el.TextColor(t.Text).Bold()
				}
				ui.Text(c, filepath.Ext(f.name)).FontSize(e.sideSize(-2)).TextColor(t.TextMuted)
			})
			if !f.mod.IsZero() {
				ui.Text(c, f.mod.Format("2006-01-02")).FontSize(e.sideSize(-2)).
					TextColor(t.TextMuted).Padding(2, 0)
			}
			if f.line != "" {
				ui.Text(c, f.line).FontSize(e.sideSize(-1)).TextColor(t.TextMuted).Padding(2, 0)
			}
		})
	}
	if shown == 0 {
		if e.folder() == "" && e.filter == "" {
			// No folder and no file: the panel offers to open one, as
			// Typora's empty panel does.
			row := ui.Row(c).Padding(8, 18).Gap(8).AlignItems(ui.Center).Radius(5)
			if row.Hovered() {
				row.Background(t.SurfaceHover)
			}
			if e.sh.OpenFolder != nil {
				row.OnClick(func() { e.sh.OpenFolder(e.win) })
			}
			row.Children(func() {
				ui.Text(c, "＋").FontSize(e.sideSize(-1)).TextColor(t.TextMuted)
				ui.Text(c, locales.OpenFolder()).FontSize(e.sideSize(-1))
			})
			ui.Text(c, locales.FilesMarkdownOnly()).
				FontSize(e.sideSize(-3)).TextColor(t.TextMuted).Padding(4, 30)
			return
		}
		msg := locales.NoMarkdownFiles()
		if e.filter != "" {
			msg = locales.NoMatchingFiles()
		}
		ui.Text(c, msg).FontSize(e.sideSize(-3)).TextColor(t.TextMuted).Padding(6, 18)
	}
}

// folderFiles reads the Markdown files of the document's folder, again when
// the folder changes or a second has passed, so that a file made in another
// app shows up without reopening the window.
func (e *Editor) folderFiles() []fileItem {
	dir := e.folder()
	if dir == e.filesDir && time.Since(e.filesAt) < time.Second {
		return e.files
	}
	e.filesDir, e.filesAt = dir, time.Now()
	e.files = readMarkdown(dir)
	return e.files
}

// folder is the folder of the document, or "" for one without a file yet.
func (e *Editor) folder() string {
	if e.sh.Folder != "" {
		return e.sh.Folder
	}
	if p := e.path(); p != "" {
		return filepath.Dir(p)
	}
	return ""
}

// path is where the document is saved, or "" for one not saved yet.
func (e *Editor) path() string {
	if e.file == nil {
		return ""
	}
	return e.file.State().Path
}

// readMarkdown lists the Markdown files of a folder, by name, the hidden
// ones left out.
func readMarkdown(dir string) []fileItem {
	var files []fileItem
	if dir == "" {
		return files
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		return files
	}
	for _, ent := range ents {
		if ent.IsDir() || strings.HasPrefix(ent.Name(), ".") {
			continue
		}
		switch strings.ToLower(filepath.Ext(ent.Name())) {
		case ".md", ".markdown":
			it := fileItem{name: ent.Name(), path: filepath.Join(dir, ent.Name())}
			if info, err := ent.Info(); err == nil {
				it.mod = info.ModTime()
			}
			it.line = firstLine(it.path)
			files = append(files, it)
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].name < files[j].name })
	return files
}

// headingText is what a heading says, without its markers.
func headingText(src string) string {
	line := src
	if i := strings.IndexByte(line, '\n'); i >= 0 {
		line = line[:i]
	}
	line = strings.TrimLeft(line, " \t")
	i := 0
	for i < len(line) && line[i] == '#' {
		i++
	}
	return strings.TrimSpace(line[i:])
}

// firstLine is the first line of a file, its markers cut away and its tail
// shortened, for the file list to show under the file's name.
func firstLine(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	buf := make([]byte, 512)
	n, _ := f.Read(buf)
	text := strings.ToValidUTF8(string(buf[:n]), "")
	line, _, _ := strings.Cut(text, "\n")
	line = strings.TrimSpace(line)
	for len(line) > 0 && (line[0] == '#' || line[0] == '>' || line[0] == '-' || line[0] == '*') {
		line = strings.TrimSpace(line[1:])
	}
	if r := []rune(line); len(r) > 34 {
		line = string(r[:34]) + "…"
	}
	return line
}
