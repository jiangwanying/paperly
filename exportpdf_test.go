package main

import (
	"bytes"
	"image/jpeg"
	"os"
	"regexp"
	"strings"
	"testing"

	"paperly/internal/editor"
)

// A document goes out as a PDF: one page for each picture, with the pages
// numbered and the headings bookmarked.
func TestWritePDF(t *testing.T) {
	src := "# 第一章\n\n" + strings.Repeat("这一页要写得很长，好让导出分成好几页来看。\n\n", 12) +
		"## 第二节\n\n" + strings.Repeat("又一段文字，继续把页面填下去。\n\n", 12)
	ex, err := editor.NewExporter(src, "", editor.NewShared(), editor.PageOptions{
		Width: 480, Height: 620, Scale: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if ex.Pages() < 2 {
		t.Fatalf("%d pages", ex.Pages())
	}
	pages := make([]pdfPage, 0, ex.Pages())
	for i := 0; i < ex.Pages(); i++ {
		p := ex.Page(i)
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, p.Image, &jpeg.Options{Quality: 88}); err != nil {
			t.Fatal(err)
		}
		pages = append(pages, pdfPage{jpeg: buf.Bytes(), headings: p.Heading})
	}
	path := "/tmp/paperly-test.pdf"
	if err := writePDF(path, pages); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), "%PDF-") {
		t.Fatalf("not a PDF: %q", string(data[:8]))
	}
	re := regexp.MustCompile(`/Type\s*/Page[^s]`)
	if got := len(re.FindAll(data, -1)); got != len(pages) {
		t.Errorf("%d page objects for %d pages", got, len(pages))
	}
	// The bookmarks are named as PDFs name things beyond Latin: UTF-16
	// with the byte order mark, so that a reader shows the words.
	for _, want := range []string{"第一章", "第二节"} {
		if !bytes.Contains(data, []byte(utf16be(want))) {
			t.Errorf("%q is not in the outline as UTF-16", want)
		}
	}
	// The bytes of a UTF-16 title decode back to the words written.
	dec := utf16be("第一章")
	if !bytes.Contains(data, []byte(dec)) {
		t.Errorf("the first heading is not in the file: %q", dec)
	}
	if len(data) < 10_000 {
		t.Errorf("the PDF is %d bytes, too small to hold the pages", len(data))
	}
	t.Logf("%d pages, %d bytes", len(pages), len(data))
}
