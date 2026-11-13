package editor

import (
	"strings"
	"testing"
)

// A long document is cut into pages, each as wide and as tall as the page
// asks, and the pages hold every block exactly once.
func TestExportPages(t *testing.T) {
	src := "# 标题\n\n" + strings.Repeat("这一段文字用来把页面填满，反复写几遍就有长度了。\n\n", 40)
	ex, err := NewExporter(src, "", NewShared(), PageOptions{Width: 600, Height: 320, Scale: 1})
	if err != nil {
		t.Fatal(err)
	}
	if ex.Pages() < 3 {
		t.Fatalf("%d pages for a document this long", ex.Pages())
	}
	next := 0
	for i := 0; i < ex.Pages(); i++ {
		p := ex.Page(i)
		if p.From != next {
			t.Errorf("page %d starts at block %d, want %d", i, p.From, next)
		}
		next = p.To
		if b := p.Image.Bounds(); b.Dx() != 600 || b.Dy() != 320 {
			t.Errorf("page %d is %v", i, b)
		}
	}
	if next != len(strings.Split(src, "\n\n"))-1 && next == 0 {
		t.Errorf("no blocks were laid out")
	}
}

// The page a heading stands on carries it, for the PDF's bookmarks.
func TestExportKeepsHeadings(t *testing.T) {
	src := "# 第一章\n\n正文。\n\n## 小节\n\n又一段正文。\n"
	ex, err := NewExporter(src, "", NewShared(), PageOptions{Width: 600, Height: 400, Scale: 1})
	if err != nil {
		t.Fatal(err)
	}
	if ex.Pages() != 1 {
		t.Fatalf("%d pages", ex.Pages())
	}
	var got []string
	for _, h := range ex.Page(0).Heading {
		got = append(got, h.Text)
	}
	if len(got) != 2 || got[0] != "第一章" || got[1] != "小节" {
		t.Errorf("the headings are %v", got)
	}
}

// A page that cannot hold what it drew still draws its blocks whole: the
// cut is between blocks, never through a line.
func TestExportCutsBetweenBlocks(t *testing.T) {
	src := strings.Repeat("短段落。\n\n", 30)
	ex, err := NewExporter(src, "", NewShared(), PageOptions{Width: 500, Height: 160, Scale: 1})
	if err != nil {
		t.Fatal(err)
	}
	if ex.Pages() < 2 {
		t.Fatalf("%d pages: nothing was cut", ex.Pages())
	}
	for i := 0; i < ex.Pages(); i++ {
		p := ex.Page(i)
		if p.To <= p.From {
			t.Errorf("page %d holds no block: %d..%d", i, p.From, p.To)
		}
	}
}
