package editor

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"paperly/internal/doc"
)

const tableSrc = "| 功能 | 快捷键 |\n| --- | :-: |\n| 加粗 | ⌘B |\n| 查找 | ⌘F |"

// A table reads into cells, and writes back as it came: the alignment row
// survives, and a pipe inside a cell stays escaped.
func TestTableRoundTrip(t *testing.T) {
	tb := parseTable(tableSrc)
	if !tb.on {
		t.Fatal("the table did not parse")
	}
	if len(tb.rows) != 3 || len(tb.rows[0]) != 2 {
		t.Fatalf("the table is %v", tb.rows)
	}
	if tb.rows[1][0] != "加粗" || tb.rows[1][1] != "⌘B" {
		t.Errorf("the first row is %v", tb.rows[1])
	}
	if got := tb.markdown(); got != tableSrc {
		t.Errorf("the table wrote %q", got)
	}

	// A pipe inside a cell is escaped on the way out and read back whole.
	tb.rows[1][0] = "a | b"
	if md := tb.markdown(); !strings.Contains(md, `a \| b`) {
		t.Errorf("the pipe was not escaped: %q", md)
	}
	if again := parseTable(tb.markdown()); again.rows[1][0] != "a | b" {
		t.Errorf("the pipe did not read back: %q", again.rows[1][0])
	}
}

// What is not a table is left to the text area.
func TestTableFallback(t *testing.T) {
	for _, src := range []string{"just words", "| a | b |\n| nope | nope |", "| only |"} {
		if tb := parseTable(src); tb.on {
			t.Errorf("%q parsed as a table", src)
		}
	}
}

// Opening a table puts a grid of cells in its place, and what is typed lands
// in the block, as every keystroke does.
func TestTableTyping(t *testing.T) {
	ed := New(doc.New(tableSrc), NewShared())
	tt := ui.NewTester(ed.View, 760, 500)
	tt.Frame()
	tt.Frame()
	ed.Activate(0)
	if !ed.tbl.on {
		t.Fatal("the block did not open as a table")
	}
	tt.Frame()
	tt.Frame()
	tt.Type("名称")
	tt.Frame()
	if src := ed.Doc().Source(); !strings.HasPrefix(src, "| 功能名称 |") {
		t.Errorf("the typing did not reach the table: %q", src)
	}
}

// Tab moves to the next cell, and Shift-Tab back; Return goes down a row.
func TestTableKeysMove(t *testing.T) {
	ed := New(doc.New(tableSrc), NewShared())
	tt := ui.NewTester(ed.View, 760, 500)
	tt.Frame()
	tt.Frame()
	ed.Activate(0)
	tt.Frame()
	tt.Frame()

	tt.Key(0, ui.KeyTab)
	tt.Frame()
	tt.Frame()
	tt.Type("键")
	tt.Frame()
	if src := ed.Doc().Source(); !strings.HasPrefix(src, "| 功能 | 快捷键键 |") {
		t.Errorf("Tab did not move to the next cell: %q", src)
	}

	tt.Key(ui.Shift, ui.KeyTab)
	tt.Frame()
	tt.Frame()
	tt.Type("X")
	tt.Frame()
	if src := ed.Doc().Source(); !strings.HasPrefix(src, "| 功能X | 快捷键键 |") {
		t.Errorf("Shift-Tab did not move back: %q", src)
	}

	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	tt.Frame()
	tt.Type("Y")
	tt.Frame()
	if src := ed.Doc().Source(); !strings.Contains(src, "| 加粗Y |") {
		t.Errorf("Return did not go down a row: %q", src)
	}
}

// The menu adds and takes away rows and columns, and the block follows.
func TestTableRowsAndColumns(t *testing.T) {
	ed := New(doc.New(tableSrc), NewShared())
	tt := ui.NewTester(ed.View, 760, 500)
	tt.Frame()
	tt.Frame()
	ed.Activate(0)
	tt.Frame()

	ed.tableAddRow(0, true)
	if len(ed.tbl.rows) != 4 {
		t.Fatalf("a row was not added: %d rows", len(ed.tbl.rows))
	}
	if src := ed.Doc().Source(); !strings.Contains(src, "|  |  |") {
		t.Errorf("the empty row is not in the block: %q", src)
	}

	ed.tableAddCol(1, true)
	if len(ed.tbl.rows[0]) != 3 {
		t.Fatalf("a column was not added: %v", ed.tbl.rows[0])
	}
	if src := ed.Doc().Source(); !strings.Contains(src, "| --- | :-: | --- |") {
		t.Errorf("the alignment row did not grow: %q", src)
	}

	ed.tableDeleteCol(2)
	if src := ed.Doc().Source(); !strings.Contains(src, "| --- | :-: |") {
		t.Errorf("the column did not go: %q", src)
	}
	ed.tableDeleteRow(1)
	if len(ed.tbl.rows) != 3 {
		t.Errorf("the row did not go: %d rows", len(ed.tbl.rows))
	}

	// Esc leaves the table, with what was written kept.
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if ed.active != -1 || ed.tbl.on {
		t.Errorf("Esc did not leave the table: active=%d on=%v", ed.active, ed.tbl.on)
	}
}
