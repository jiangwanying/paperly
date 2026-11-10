package doc

import "testing"

// A heading grows as its marker is typed.
func TestBlockSize(t *testing.T) {
	cases := map[string]float32{
		"# 一":    32,
		"## 二":   24,
		"### 三":  20,
		"#### 四": 17,
		"正文":     16,
		"> 引用":   16,
		"- 列表":   16,
	}
	for src, want := range cases {
		if got := BlockSize(src); got != want {
			t.Errorf("BlockSize(%q) = %v, want %v", src, got, want)
		}
	}
}

// The delimiters fade and the text between them takes its weight.
func TestSyntaxRuns(t *testing.T) {
	// "**粗体**": two bytes of asterisks, six of Chinese, two more.
	runs := SyntaxRuns("**粗体**")
	if len(runs) != 3 {
		t.Fatalf("%d runs for strong text, want 3", len(runs))
	}
	if runs[1].Kind != RunStrong || runs[1].Weight != 700 || runs[1].Start != 2 || runs[1].End != 8 {
		t.Errorf("the strong text is %+v", runs[1])
	}
	if runs[0].Kind != RunMarker || runs[0].End != 2 || runs[2].Start != 8 {
		t.Errorf("the delimiters are %+v and %+v", runs[0], runs[2])
	}

	// A line's own marker dims, and takes no weight.
	rp := SyntaxRuns("# 标题")
	if len(rp) != 1 || rp[0].Start != 0 || rp[0].End != 2 || rp[0].Kind != RunPrefix {
		t.Errorf("the heading marker is %+v", rp)
	}

	// Every line of a multi-line block has a marker of its own.
	items := SyntaxRuns("- 一\n- 二")
	if len(items) != 2 || items[0].Kind != RunPrefix || items[1].Kind != RunPrefix {
		t.Errorf("the item markers are %+v", items)
	}
	if items[1].Start != 6 || items[1].End != 8 {
		t.Errorf("the second dash is %+v", items[1])
	}

	// The checkbox of a task rides on its bullet, all of it one marker.
	task := SyntaxRuns("- [ ] 买牛奶")
	if len(task) != 1 || task[0].Kind != RunPrefix || task[0].End != 6 {
		t.Errorf("the task marker is %+v", task)
	}

	// A fence line is a marker too.
	fence := SyntaxRuns("```go\ncode\n```")
	if len(fence) != 2 || fence[0].Kind != RunPrefix || fence[1].Kind != RunPrefix {
		t.Errorf("the fences are %+v", fence)
	}

	// Inline markers stay RunMarker, wherever they stand.
	inline := SyntaxRuns("**粗** 后行 **细**")
	kinds := [4]RunKind{inline[0].Kind, inline[2].Kind, inline[3].Kind, inline[5].Kind}
	if kinds != [4]RunKind{RunMarker, RunMarker, RunMarker, RunMarker} {
		t.Errorf("the inline markers are %+v", inline)
	}
}

// The blocks of a document, with where they sit and what they are.
func TestBlocks(t *testing.T) {
	d := New("# 标题\n\n正文一\n\n## 小标题\n\n- 列表\n")
	if d.Len() != 4 {
		t.Fatalf("%d blocks, want 4: %+v", d.Len(), d.Blocks())
	}
	h, _ := d.Block(0)
	if h.Kind.String() != "heading" || h.Level != 1 || h.Line != 1 {
		t.Errorf("first block %+v", h)
	}
	h2, _ := d.Block(2)
	if h2.Level != 2 || h2.Line != 5 {
		t.Errorf("third block %+v", h2)
	}
	if d.Source()[h.Start:h.End] != "# 标题" {
		t.Errorf("the first block's source is %q", d.Source()[h.Start:h.End])
	}
}

// Replacing a block keeps the rest of the document, and the blocks after it
// move.
func TestReplace(t *testing.T) {
	d := New("# 标题\n\n正文\n\n结尾\n")
	if d.BlockAt(0) != 0 {
		t.Fatalf("offset 0 is in block %d", d.BlockAt(0))
	}
	b, _ := d.Block(1)
	d.Replace(b.Span, "新的正文，长一些。")
	if got := d.Source(); got != "# 标题\n\n新的正文，长一些。\n\n结尾\n" {
		t.Errorf("source after the replace:\n%q", got)
	}
	if d.Revision() == 0 {
		t.Errorf("the revision did not change")
	}
}

// Line lookups agree with each other.
func TestLines(t *testing.T) {
	d := New("一\n二\n三\n")
	for line := 1; line <= 3; line++ {
		off := d.OffsetOfLine(line)
		if got := d.LineOf(off); got != line {
			t.Errorf("line %d at offset %d reads as line %d", line, off, got)
		}
	}
}
