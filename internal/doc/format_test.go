package doc

import "testing"

func TestSetHeading(t *testing.T) {
	cases := []struct {
		in    string
		level int
		want  string
	}{
		{"正文", 1, "# 正文"},
		{"## 二级", 1, "# 二级"},
		{"# 一级", 0, "一级"},
		{"### 三级\n第二行", 2, "## 三级\n第二行"},
	}
	for _, c := range cases {
		if got := SetHeading(c.in, c.level); got != c.want {
			t.Errorf("SetHeading(%q, %d) = %q, want %q", c.in, c.level, got, c.want)
		}
	}
}

func TestToggleQuote(t *testing.T) {
	if got := ToggleQuote("一段话"); got != "> 一段话" {
		t.Errorf("quote: %q", got)
	}
	if got := ToggleQuote("> 一段话"); got != "一段话" {
		t.Errorf("unquote: %q", got)
	}
	if got := ToggleQuote("第一行\n第二行"); got != "> 第一行\n> 第二行" {
		t.Errorf("two lines: %q", got)
	}
}

func TestToggleList(t *testing.T) {
	if got := ToggleList("苹果", ListBullet); got != "- 苹果" {
		t.Errorf("bullet: %q", got)
	}
	if got := ToggleList("- 苹果", ListBullet); got != "苹果" {
		t.Errorf("bullet off: %q", got)
	}
	// Switching kinds replaces the marker rather than stacking them.
	if got := ToggleList("- 苹果", ListOrdered); got != "1. 苹果" {
		t.Errorf("bullet to ordered: %q", got)
	}
	if got := ToggleList("1. 苹果", ListTask); got != "- [ ] 苹果" {
		t.Errorf("ordered to task: %q", got)
	}
}

func TestToggleCode(t *testing.T) {
	got := ToggleCode("fmt.Println(1)")
	if got != "```\nfmt.Println(1)\n```" {
		t.Errorf("fence: %q", got)
	}
	if back := ToggleCode(got); back != "fmt.Println(1)" {
		t.Errorf("unfence: %q", back)
	}
}

func TestWrap(t *testing.T) {
	// A selection takes the marks on both sides.
	out, s, e := Wrap("把这句话加粗", 0, 3, "**")
	if out != "**把这句**话加粗" {
		t.Fatalf("wrap: %q", out)
	}
	if s != 2 || e != 5 {
		t.Errorf("selection after wrap: %d..%d", s, e)
	}
	// Wrapping the same run again takes them off.
	back, _, _ := Wrap(out, s, e, "**")
	if back != "把这句话加粗" {
		t.Errorf("unwrap: %q", back)
	}
	// An empty selection gets a pair with the caret between.
	empty, at, at2 := Wrap("ab", 2, 2, "`")
	if empty != "ab``" || at != 3 || at2 != 3 {
		t.Errorf("insert: %q %d %d", empty, at, at2)
	}
}

func TestHeadingLevels(t *testing.T) {
	if got := PromoteHeading("## 二级"); got != "# 二级" {
		t.Errorf("promote: %q", got)
	}
	if got := PromoteHeading("# 一级"); got != "# 一级" {
		t.Errorf("promote at the top: %q", got)
	}
	if got := DemoteHeading("# 一级"); got != "## 一级" {
		t.Errorf("demote: %q", got)
	}
	// A paragraph stays one.
	if got := DemoteHeading("正文"); got != "正文" {
		t.Errorf("demote a paragraph: %q", got)
	}
	if got := HeadingLevel("### 三级"); got != 3 {
		t.Errorf("level: %d", got)
	}
	if got := HeadingLevel("正文"); got != 0 {
		t.Errorf("level of a paragraph: %d", got)
	}
}

func TestWrapPair(t *testing.T) {
	out, s, e := WrapPair("下划线", 0, 3, "<u>", "</u>")
	if out != "<u>下划线</u>" {
		t.Fatalf("wrap: %q", out)
	}
	if s != 3 || e != 6 {
		t.Errorf("selection %d..%d", s, e)
	}
}

func TestStripInline(t *testing.T) {
	cases := map[string]string{
		"**粗** 与 *斜*":         "粗 与 斜",
		"`代码` 和 ~~删除~~":       "代码 和 删除",
		"==高亮== 与 <u>下划线</u>": "高亮 与 下划线",
		"[文字](http://x)":      "文字",
	}
	for in, want := range cases {
		if got := StripInline(in); got != want {
			t.Errorf("StripInline(%q) = %q, want %q", in, got, want)
		}
	}
}
