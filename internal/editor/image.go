package editor

import (
	"errors"
	"fmt"
	neturl "net/url"
	"os"
	"paperly/locales"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"

	"paperly/internal/doc"
)

// imageExts are the files an image insertion, or a paste, takes as pictures.
var imageExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
	".webp": true, ".bmp": true, ".svg": true, ".avif": true,
}

func isImage(path string) bool {
	return imageExts[strings.ToLower(filepath.Ext(path))]
}

// docDir is the folder the document is in, which its pictures are shown
// against: the file's own, or the folder the file panel opened while the
// document has no file yet.
func (e *Editor) docDir() string {
	if e.file != nil {
		if p := e.file.State().Path; p != "" {
			return filepath.Dir(p)
		}
	}
	return e.sh.Folder
}

// loadImages keeps the view's image loader on the document's folder, so that
// ![](assets/a.png) shows the file beside the document.
func (e *Editor) loadImages() {
	dir := e.docDir()
	if dir == e.mdDir {
		return
	}
	e.mdDir = dir
	if dir == "" {
		e.md.LoadImage = nil
		return
	}
	load := markdown.FileImages(dir)
	// An address is written as Markdown takes it, a space as %20: the file
	// it names is the one with the space.
	e.md.LoadImage = func(addr string) (ui.ImageSource, error) {
		if p, err := neturl.PathUnescape(addr); err == nil {
			addr = p
		}
		return load(addr)
	}
}

// ChooseImage asks for a picture and puts it where the caret is: what the
// menu's 图片… does.
func (e *Editor) ChooseImage() { e.chooseImage(-1, -1) }

// chooseImage asks for a picture and puts it where the caret is, or in a
// block of its own below block i when the menu was opened on one that is
// not being written, or at the byte offset at when the menu was the empty
// room's: a picture put in from a paragraph's menu stays under that
// paragraph rather than going to the end of the text. A picture that cannot
// be put in says why, rather than doing nothing.
func (e *Editor) chooseImage(i, at int) {
	win := e.win
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent: win,
			Title:  locales.InsertImage(),
			Filters: []mygo.FileFilter{{
				Name:       locales.FilterImage(),
				Extensions: []string{".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".svg", ".avif"},
				MIMETypes:  []string{"image/*"},
			}},
		})
		if err != nil || len(paths) == 0 {
			return
		}
		put := func() {
			e.startForInsert(i, at)
			if err := e.InsertImage(paths[0]); err != nil {
				e.imageError(err)
			}
		}
		if win != nil {
			win.Update(put)
		} else {
			put()
		}
	}()
}

// startForInsert puts the writing where a block put in from a menu goes: at
// the caret when the menu was opened on the block being written, in a block
// of its own below block i, at the byte offset at when the menu was the
// empty room's, and below the block the pointer last stood on when nothing
// is. With nothing to go by, the text goes at the end, as a new paragraph.
func (e *Editor) startForInsert(i, at int) {
	if i >= 0 {
		if e.active == i {
			return
		}
		if _, ok := e.doc.Block(i); ok {
			e.startAfter(i)
			return
		}
	} else if at >= 0 && !e.editing() {
		e.startAt(at)
		return
	}
	if e.editing() {
		return
	}
	if e.lastBlock >= 0 {
		if _, ok := e.doc.Block(e.lastBlock); ok {
			e.startAfter(e.lastBlock)
			return
		}
	}
	e.StartParagraph()
}

// InsertImage puts a picture where the caret is: the file is copied into an
// assets folder beside the document, and the Markdown that shows it takes
// the caret's place.
func (e *Editor) InsertImage(src string) error {
	if !isImage(src) {
		return errors.New(locales.NotAnImage())
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return e.insertImageData(data, filepath.Base(src))
}

// InsertImageData is InsertImage for a picture the clipboard had in memory.
func (e *Editor) InsertImageData(data []byte) error {
	name := locales.PastedName() + time.Now().Format("2006-01-02-15.04.05") + ".png"
	return e.insertImageData(data, name)
}

// insertImageData writes the picture into the document's assets folder and
// inserts the Markdown for it.
func (e *Editor) insertImageData(data []byte, name string) error {
	dir := e.docDir()
	if dir == "" {
		return errors.New(locales.NeedSavedDoc())
	}
	if !isImage(name) {
		name += ".png"
	}
	// The folder the picture is copied into, and how the Markdown addresses
	// it, are settings: an assets folder beside the document or the
	// document's own, reached by a relative or an absolute path.
	folder := dir
	rel := ""
	if e.sh.ImageDir != "same" {
		folder = filepath.Join(dir, "assets")
		rel = "assets"
	}
	if err := os.MkdirAll(folder, 0o755); err != nil {
		return err
	}
	name = uniqueName(folder, name)
	path := filepath.Join(folder, name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}
	addr := filepath.ToSlash(filepath.Join(rel, name))
	if e.sh.ImagePath == "absolute" {
		addr = filepath.ToSlash(path)
	}
	// A space or a bracket ends a link's destination in Markdown, and the
	// picture then shows as its source rather than as itself.
	addr = mdImagePath(addr)
	alt := strings.TrimSuffix(name, filepath.Ext(name))
	e.insertText(fmt.Sprintf("![%s](%s)", alt, addr))
	return nil
}

// mdImagePath is a path as Markdown takes it in an image's address: the
// characters that would end it, escaped.
func mdImagePath(p string) string {
	return strings.NewReplacer(
		" ", "%20", "(", "%28", ")", "%29", "<", "%3C", ">", "%3E",
	).Replace(p)
}

// readableSrc is a block's source with the address of each picture written
// as Markdown takes it, when the address holds a space: a picture put in
// before this was written, or one whose file name has a space in it, shows
// as a picture rather than as its own source. The source itself is left as
// it is; only what the view draws is fixed.
func readableSrc(src string) string {
	if !strings.Contains(src, "![") || !strings.Contains(src, " ") {
		return src
	}
	var b strings.Builder
	last := 0
	for i := 0; i+1 < len(src); i++ {
		if src[i] != '!' || src[i+1] != '[' {
			continue
		}
		j := strings.Index(src[i:], "](")
		if j < 0 {
			break
		}
		j += i + 2
		k := j
		for k < len(src) && src[k] != ')' {
			k++
		}
		if k >= len(src) {
			break
		}
		dest := src[j:k]
		if strings.Contains(dest, " ") && !strings.HasPrefix(dest, "<") &&
			!strings.ContainsAny(dest, "\"'") {
			b.WriteString(src[last:j])
			b.WriteString("<")
			b.WriteString(dest)
			b.WriteString(">")
			last = k
		}
		i = k
	}
	if last == 0 {
		return src
	}
	b.WriteString(src[last:])
	return b.String()
}

// uniqueName is name, or name with -1, -2… before its extension, when the
// folder already holds it.
func uniqueName(dir, name string) string {
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	out := name
	for i := 1; ; i++ {
		if _, err := os.Stat(filepath.Join(dir, out)); os.IsNotExist(err) {
			return out
		}
		out = fmt.Sprintf("%s-%d%s", stem, i, ext)
	}
}

// PasteImage puts a picture the clipboard holds where the caret is: a file
// copied in Finder, or the data of a screenshot. It reports whether the
// clipboard had one, so that a paste of text goes on as usual.
func (e *Editor) PasteImage() bool {
	if !e.sh.PasteImage {
		return false
	}
	data := e.svc.ReadClipboardData(transfer.PNG, transfer.FileList)
	if files, err := data.Files(); err == nil {
		for _, f := range files {
			if isImage(f) {
				if err := e.InsertImage(f); err == nil {
					return true
				}
			}
		}
	}
	if b, err := data.Read(transfer.PNG); err == nil && len(b) > 0 {
		if err := e.InsertImageData(b); err == nil {
			return true
		}
	}
	// A screenshot on a Mac sits in the clipboard as TIFF alone: the
	// window's clipboard turns it into a PNG for us.
	if b := mygo.Clipboard.ReadImage(); len(b) > 0 {
		if err := e.InsertImageData(b); err == nil {
			return true
		} else {
			e.imageError(err)
		}
	}
	return false
}

// imageError says why a picture did not go in, rather than leaving the paste
// or the menu with nothing to show for itself.
func (e *Editor) imageError(err error) {
	go func() {
		_, _ = mygo.Dialog.Message(mygo.MessageOptions{
			Title:   locales.InsertImage(),
			Message: locales.InsertFailed(),
			Detail:  err.Error(),
			Buttons: []string{locales.Ok()},
		})
	}()
}

// insertText puts s where the caret is in the block being written; with no
// block being written, it starts a paragraph for it first, below the block
// the pointer last stood on rather than at the end of the text.
func (e *Editor) insertText(s string) {
	e.record()
	if !e.editing() {
		e.startForInsert(-1, -1)
	}
	if !e.editing() {
		return
	}
	at := doc.ByteOffset(e.draft, e.selInDraftA())
	if at > len(e.draft) {
		at = len(e.draft)
	}
	e.draft = e.draft[:at] + s + e.draft[at:]
	e.selA += utf8.RuneCountInString(s)
	e.selB = e.selA
	if e.itemLine >= 0 {
		e.resyncItem()
		e.itemCaret = e.selA
	}
	e.sync()
}
