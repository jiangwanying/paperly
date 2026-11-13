package main

import (
	"bytes"
	"fmt"
	"image/jpeg"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"github.com/go-pdf/fpdf"

	"paperly/internal/editor"
	"paperly/locales"
)

// The page a PDF exports on: A4 in points, with the margins a document
// wants—room for the text, and a line under it for the number.
const (
	pdfPageW = 595.276
	pdfPageH = 841.89
	pdfSide  = 56.0
	pdfTop   = 56.0
	pdfFoot  = 64.0
)

// pointsPerDIP is how many points one DIP of the page takes: 96 to the inch
// against 72.
const pointsPerDIP = 72.0 / 96.0

// exportPDF asks where the PDF goes, and writes the document there, a page
// at a time: the pages as the window draws them, without the window.
func (a *App) exportPDF(win *mygo.Window) {
	ed := a.Editor(win)
	if ed == nil {
		return
	}
	def := "Paperly.pdf"
	if p := ed.Path(); p != "" {
		base := strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))
		def = filepath.Join(filepath.Dir(p), base+".pdf")
	}
	out, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
		Parent:      win,
		Title:       locales.ExportPdf(),
		DefaultPath: def,
		ButtonLabel: locales.ExportPdf(),
		Filters:     []mygo.FileFilter{{Name: "PDF", Extensions: []string{".pdf"}}},
	})
	if err != nil || strings.TrimSpace(out) == "" {
		return
	}
	if !strings.EqualFold(filepath.Ext(out), ".pdf") {
		out += ".pdf"
	}
	// The pages are drawn from the source as it stands: the window may go
	// on being written in while they are.
	src := ed.Doc().Source()
	dir := ""
	if p := ed.Path(); p != "" {
		dir = filepath.Dir(p)
	}
	sh := ed.Shared()

	fail := func(err error) {
		_, _ = mygo.Dialog.Message(mygo.MessageOptions{
			Parent:  win,
			Type:    mygo.MessageError,
			Title:   locales.ExportPdf(),
			Message: locales.ExportFailed(),
			Detail:  err.Error(),
			Buttons: []string{locales.Ok()},
		})
	}

	width := float32(pdfPageW-2*pdfSide) / pointsPerDIP
	height := float32(pdfPageH-pdfTop-pdfFoot) / pointsPerDIP

	go func() {
		var ex *editor.Exporter
		mygo.RunOnMain(func() {
			ex, err = editor.NewExporter(src, dir, sh, editor.PageOptions{
				Width: width, Height: height, Scale: 2,
			})
		})
		if err != nil {
			fail(err)
			return
		}
		// Each page is drawn on the window's thread, one at a time, and
		// pressed into a JPEG at once: the pages themselves need not be
		// kept.
		pages := make([]pdfPage, 0, ex.Pages())
		for i := 0; i < ex.Pages(); i++ {
			var p editor.PageImage
			mygo.RunOnMain(func() { p = ex.Page(i) })
			var buf bytes.Buffer
			if err := jpeg.Encode(&buf, p.Image, &jpeg.Options{Quality: 88}); err != nil {
				fail(err)
				return
			}
			pages = append(pages, pdfPage{jpeg: buf.Bytes(), headings: p.Heading})
		}
		if err := writePDF(out, pages); err != nil {
			fail(err)
			return
		}
		mygo.Shell.ShowItemInFolder(out)
	}()
}

// utf16be is s as a PDF text string holds it when it is not Latin: UTF-16
// from the high byte down, with the mark that says so in front.
func utf16be(s string) string {
	rs := []rune(s)
	out := make([]byte, 2, 2+2*len(rs))
	out[0], out[1] = 0xFE, 0xFF
	for _, r := range rs {
		switch {
		case r > 0xFFFF:
			// Outside the common ground a character takes two pieces.
			r -= 0x10000
			hi := 0xD800 + r>>10
			lo := 0xDC00 + r&0x3FF
			out = append(out, byte(hi>>8), byte(hi), byte(lo>>8), byte(lo))
		default:
			out = append(out, byte(r>>8), byte(r))
		}
	}
	return string(out)
}

// pdfPage is one page ready for the PDF: its picture as a JPEG, and the
// headings that stand on it.
type pdfPage struct {
	jpeg     []byte
	headings []editor.PageHeading
}

// writePDF puts the pages in a PDF, one picture a page, with the number of
// each at the foot and the headings as bookmarks.
func writePDF(path string, pages []pdfPage) error {
	if len(pages) == 0 {
		return fmt.Errorf("the document is empty")
	}
	pdf := fpdf.NewCustom(&fpdf.InitType{
		UnitStr:        "pt",
		Size:           fpdf.SizeType{Wd: pdfPageW, Ht: pdfPageH},
		OrientationStr: "P",
	})
	pdf.SetMargins(pdfSide, pdfTop, pdfSide)
	pdf.SetAutoPageBreak(false, 0)
	pdf.SetTitle(filepath.Base(path), true)
	pdf.SetCreator("Paperly", true)

	contentW := pdfPageW - 2*pdfSide
	contentH := pdfPageH - pdfTop - pdfFoot

	// The headings, as bookmarks: a level that jumps more than one step
	// would have no parent, so each is held to one below the last.
	level := 0
	for i, p := range pages {
		pdf.AddPage()
		name := fmt.Sprintf("page%d", i+1)
		pdf.RegisterImageOptionsReader(name, fpdf.ImageOptions{ImageType: "JPG"}, bytes.NewReader(p.jpeg))
		pdf.ImageOptions(name, pdfSide, pdfTop, contentW, contentH, false,
			fpdf.ImageOptions{ImageType: "JPG"}, 0, "")

		// The number of the page, under it.
		pdf.SetFont("Helvetica", "", 9)
		pdf.SetTextColor(130, 130, 130)
		pdf.SetXY(pdfSide, pdfPageH-pdfFoot+20)
		pdf.CellFormat(contentW, 10, fmt.Sprintf("%d / %d", i+1, len(pages)), "", 0, "C", false, 0, "")

		// A bookmark stands on the page it names, so it is set while that
		// page is the one being written. The title is written as UTF-16
		// with the byte order mark in front, which is how a PDF names
		// things beyond Latin: the library would take the page's own font
		// for it, and the page has none to give.
		for _, h := range p.headings {
			l := h.Level
			if l > level+1 {
				l = level + 1
			}
			if l < 1 {
				l = 1
			}
			pdf.Bookmark(utf16be(h.Text), l, pdfTop)
			level = l
		}
	}
	return pdf.OutputFileAndClose(path)
}
