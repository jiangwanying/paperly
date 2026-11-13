package editor

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"paperly/internal/doc"
)

// longDoc is a document long enough to be worth culling: the same small
// section, written over and over.
func longDoc(reps int) string {
	block := "## 2.3 manifest.json\n\n一段说明文字，用来撑起文档的体积，包含 **加粗** 与 `代码`。\n\n```json\n{\n  \"Name\": \"Paperly\",\n  \"Version\": \"1.0.0\"\n}\n```\n\n- 列表一\n- 列表二\n- 列表三\n\n"
	return "# 长文档\n\n" + strings.Repeat(block, reps)
}

// A long document lays itself out once, and then only the blocks near the
// viewport: the rest keep their room and wait.
func TestViewportCullsFarBlocks(t *testing.T) {
	ed := New(doc.New(longDoc(60)), NewShared())
	tt := ui.NewTester(ed.View, 1440, 900)
	for i := 0; i < 3; i++ {
		tt.Frame()
	}
	total := ed.Doc().Len()
	if total < 60 {
		t.Fatalf("the document has %d blocks, too few to cull", total)
	}
	if len(ed.heights) != total || ed.cullN != total {
		t.Fatalf("the first frames measured %d heights for %d blocks", len(ed.heights), total)
	}

	ed.laid = 0
	for i := 0; i < 3; i++ {
		tt.Frame()
	}
	perFrame := ed.laid / 3
	if perFrame <= 0 || perFrame > total/3 {
		t.Errorf("%d of %d blocks laid out per frame, want only those near the viewport", perFrame, total)
	}

	// The far end is out of the frame, and the first blocks are not.
	if !ed.culled(total - 1) {
		t.Errorf("the last block was laid out while far below the viewport")
	}
	if ed.culled(0) {
		t.Errorf("the first block was left out while at the top of the viewport")
	}
}

// The page stays as tall with the far blocks left out, so the scrollbar and
// a jump to the end still land at the end.
func TestViewportKeepsThePageTall(t *testing.T) {
	ed := New(doc.New(longDoc(60)), NewShared())
	tt := ui.NewTester(ed.View, 1440, 900)
	for i := 0; i < 3; i++ {
		tt.Frame()
	}
	culled := ed.scroll.MaxY

	// The same document with culling off, for the height it takes whole.
	whole := New(doc.New(longDoc(60)), NewShared())
	tw := ui.NewTester(whole.View, 1440, 900)
	for i := 0; i < 3; i++ {
		tw.Frame()
	}
	whole.cullN = 0 // under the count that culling starts at
	for i := 0; i < 2; i++ {
		tw.Frame()
	}
	if whole.scroll.MaxY <= 0 || culled <= 0 {
		t.Fatalf("the page has no height: culled=%v whole=%v", culled, whole.scroll.MaxY)
	}
	if d := culled - whole.scroll.MaxY; d > 4 || d < -4 {
		t.Errorf("the page is %.1f tall culled and %.1f whole", culled, whole.scroll.MaxY)
	}

	// A jump to the end shows the last block.
	ed.scroll.Y = ed.scroll.MaxY
	for i := 0; i < 3; i++ {
		tt.Frame()
	}
	if ed.culled(ed.Doc().Len() - 1) {
		t.Errorf("the last block is still culled after a jump to the end")
	}
	if ed.tops[ed.Doc().Len()-1] < ed.scroll.Y {
		t.Errorf("the last block sits at %.0f, above the scroll at %.0f", ed.tops[ed.Doc().Len()-1], ed.scroll.Y)
	}
}

// A short document is laid out whole, as it always was: culling is for the
// long ones.
func TestViewportKeepsSmallDocsWhole(t *testing.T) {
	ed := New(doc.New("# 一\n\n一段。\n\n## 二\n\n又一段。"), NewShared())
	tt := ui.NewTester(ed.View, 700, 400)
	for i := 0; i < 3; i++ {
		tt.Frame()
	}
	ed.laid = 0
	tt.Frame()
	if ed.laid != ed.Doc().Len() {
		t.Errorf("%d of %d blocks laid out in a short document", ed.laid, ed.Doc().Len())
	}
}
