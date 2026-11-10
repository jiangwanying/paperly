package doc

import "testing"

func TestFindAll(t *testing.T) {
	d := New("Typora 的核心是块级编辑。\n\nblock 与 Block 都要能找到。\n\n最后一行：块级。")
	hits := d.FindAll(Query{Text: "块级"})
	if len(hits) != 2 {
		t.Fatalf("%d hits, want 2", len(hits))
	}
	if hits[0].Block != 0 {
		t.Errorf("the first hit is in block %d", hits[0].Block)
	}
	if hits[1].Block != 2 {
		t.Errorf("the second hit is in block %d", hits[1].Block)
	}
	if got := d.Source()[hits[0].Start:hits[0].End]; got != "块级" {
		t.Errorf("the first hit is %q", got)
	}
}

func TestFindAllCase(t *testing.T) {
	d := New("block Block BLOCK")
	if got := len(d.FindAll(Query{Text: "block"})); got != 3 {
		t.Errorf("without case: %d hits, want 3", got)
	}
	if got := len(d.FindAll(Query{Text: "block", Case: true})); got != 1 {
		t.Errorf("with case: %d hits, want 1", got)
	}
}

func TestFindAllWord(t *testing.T) {
	d := New("cat category cat.\ncat")
	if got := len(d.FindAll(Query{Text: "cat"})); got != 4 {
		t.Errorf("%d hits, want 4", got)
	}
	if got := len(d.FindAll(Query{Text: "cat", Word: true})); got != 3 {
		t.Errorf("whole words: %d hits, want 3", got)
	}
}

// A Chinese character beside a match is no word's edge: 用户 is a whole
// word in 修用户反馈, where no break is written.
func TestFindAllWordCJK(t *testing.T) {
	d := New("修用户反馈的三个问题\n\n用户决定：一文件一窗口。")
	if got := len(d.FindAll(Query{Text: "用户", Word: true})); got != 2 {
		t.Errorf("whole words: %d hits, want 2", got)
	}
	// An ASCII letter still clings to the match.
	d = New("用户A 与 用户B")
	if got := len(d.FindAll(Query{Text: "用户", Word: true})); got != 0 {
		t.Errorf("whole words beside letters: %d hits, want 0", got)
	}
}

func TestReplaceAll(t *testing.T) {
	d := New("一处处处。\n\n处处都有。")
	n := d.ReplaceAll(Query{Text: "处处"}, "块块")
	if n != 2 {
		t.Errorf("replaced %d, want 2", n)
	}
	// The matches do not overlap: 一处处处 has one 处处, at bytes 3 to 9.
	if got := d.Source(); got != "一块块处。\n\n块块都有。" {
		t.Errorf("source %q", got)
	}
	if got := d.ReplaceAll(Query{Text: "没有这个"}, "x"); got != 0 {
		t.Errorf("a missing text replaced %d", got)
	}
}
