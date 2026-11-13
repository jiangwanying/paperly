package editor

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/egoist/mygo/ui"

	"paperly/locales"
)

// table is a Markdown table being written cell by cell: the cells as they
// read, the alignment the delimiter row carries, and the cell the caret
// should go to in the next frame.
type table struct {
	on     bool
	rows   [][]string // the header first
	align  []string   // the delimiter row, as written: ---, :--, :-:, --:
	focusR int
	focusC int
}

// reAlignCell is a walk of the delimiter row: a run of dashes, centred or
// pushed to one side by colons.
var reAlignCell = regexp.MustCompile(`^:?-+:?$`)

// parseTable reads a block of source as a table. It gives up on anything
// that is not one — no delimiter row, a delimiter cell that says nothing —
// and the block is then written as source, as it always was.
func parseTable(src string) table {
	lines := strings.Split(strings.TrimRight(src, "\n"), "\n")
	if len(lines) < 2 {
		return table{}
	}
	header := splitRow(lines[0])
	align := splitRow(lines[1])
	if len(header) == 0 || len(align) == 0 {
		return table{}
	}
	for _, a := range align {
		if !reAlignCell.MatchString(strings.TrimSpace(a)) {
			return table{}
		}
	}
	rows := [][]string{header}
	for _, line := range lines[2:] {
		if strings.TrimSpace(line) == "" {
			continue
		}
		rows = append(rows, splitRow(line))
	}
	t := table{on: true, rows: rows, align: align, focusR: -1, focusC: -1}
	t.normalize()
	return t
}

// splitRow cuts a row into its cells: pipes inside cells are escaped, and
// the ones at either end hold nothing.
func splitRow(line string) []string {
	s := strings.TrimSpace(line)
	s = strings.TrimPrefix(s, "|")
	s = strings.TrimSuffix(s, "|")
	var out []string
	var cell strings.Builder
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] == '\\' && i+1 < len(s) && s[i+1] == '|':
			cell.WriteByte('|')
			i++
		case s[i] == '|':
			out = append(out, strings.TrimSpace(cell.String()))
			cell.Reset()
		default:
			cell.WriteByte(s[i])
		}
	}
	out = append(out, strings.TrimSpace(cell.String()))
	return out
}

// markdown writes the table back as the source it came from.
func (t table) markdown() string {
	if len(t.rows) == 0 {
		return ""
	}
	var b strings.Builder
	row := func(cells []string) {
		b.WriteString("|")
		for _, c := range cells {
			b.WriteString(" ")
			b.WriteString(strings.ReplaceAll(c, "|", `\|`))
			b.WriteString(" |")
		}
		b.WriteString("\n")
	}
	row(t.rows[0])
	b.WriteString("|")
	for _, a := range t.align {
		b.WriteString(" " + a + " |")
	}
	b.WriteString("\n")
	for _, r := range t.rows[1:] {
		row(r)
	}
	return strings.TrimRight(b.String(), "\n")
}

// normalize makes every row as wide as the header, and the delimiter row
// too, so that a column added anywhere is a column everywhere.
func (t *table) normalize() {
	if len(t.rows) == 0 {
		return
	}
	cols := len(t.rows[0])
	if cols == 0 {
		cols = 1
		t.rows[0] = []string{""}
	}
	for i, r := range t.rows {
		for len(r) < cols {
			r = append(r, "")
		}
		t.rows[i] = r[:cols]
	}
	for len(t.align) < cols {
		t.align = append(t.align, "---")
	}
	t.align = t.align[:cols]
}

// startTable takes the block being written as a table, or leaves it to the
// text area when it is not one.
func (e *Editor) startTable(src string) {
	e.tbl = parseTable(src)
	if e.tbl.on {
		e.tbl.focusR, e.tbl.focusC = 0, 0
	}
}

// resetTable puts the table away when the block is left.
func (e *Editor) resetTable() { e.tbl = table{focusR: -1, focusC: -1} }

// editTable is the block in writing when it is a table: a grid of cells,
// each one an input, the header above the rows it names.
func (e *Editor) editTable(c *ui.Context) {
	t := c.Theme()
	e.tbl.normalize()
	for r := range e.tbl.rows {
		r := r
		row := ui.Row(c).Gap(-1)
		row.Children(func() {
			for col := range e.tbl.rows[r] {
				col := col
				in := ui.TextInput(c.Key(fmt.Sprintf("cell-%d-%d", r, col)), &e.tbl.rows[r][col])
				in.Grow(1).Padding(5, 8).Border(1, t.Border).Radius(0).FocusRing(false)
				in.FontSize(e.sh.Size).Background(t.Background)
				if r == 0 {
					in.Background(t.Surface).Bold()
				}
				in.OnChange(func() { e.tableChanged() })
				in.OnTextKey(func(k ui.TextKey) bool { return e.tableKey(k, r, col) })
				in.ContextMenu(func(m *ui.Menu) { e.tableMenu(m, r, col) })
				if e.tbl.focusR == r && e.tbl.focusC == col {
					// AutoFocus only takes a cell that has just appeared;
					// a move between cells asks for the focus outright.
					in.Focus()
				}
			}
		})
	}
	e.tbl.focusR, e.tbl.focusC = -1, -1
}

// tableChanged writes the cells back into the block as they are typed.
func (e *Editor) tableChanged() {
	if !e.tbl.on {
		return
	}
	e.record()
	e.draft = e.tbl.markdown()
	e.sync()
}

// tableKey moves the caret through the grid: Tab and Shift-Tab along a row,
// Return down a column, Esc out of the block.
func (e *Editor) tableKey(k ui.TextKey, r, col int) bool {
	switch k.Key {
	case ui.KeyEscape:
		e.Commit()
		return true
	case ui.KeyTab:
		step := 1
		if k.Mods&ui.Shift != 0 {
			step = -1
		}
		e.tableStep(r, col, step)
		return true
	case ui.KeyEnter:
		if k.Mods&ui.Shift != 0 {
			return false
		}
		e.tableDown(r, col)
		return true
	}
	return false
}

// tableStep moves the caret a cell forward or back, wrapping at the ends and
// opening a row when it goes past the last cell.
func (e *Editor) tableStep(r, col, step int) {
	cols := len(e.tbl.rows[0])
	at := r*cols + col + step
	if at < 0 || at >= len(e.tbl.rows)*cols {
		return
	}
	e.tbl.focusR, e.tbl.focusC = at/cols, at%cols
}

// tableDown moves the caret to the cell below, opening a row when it is on
// the last one.
func (e *Editor) tableDown(r, col int) {
	if r+1 < len(e.tbl.rows) {
		e.tbl.focusR, e.tbl.focusC = r+1, col
		return
	}
	e.tableAddRow(len(e.tbl.rows)-1, true)
}

// tableAddRow puts an empty row above or below the one at r.
func (e *Editor) tableAddRow(r int, below bool) {
	if !e.tbl.on {
		return
	}
	e.record()
	at := r
	if below {
		at = r + 1
	}
	cols := len(e.tbl.rows[0])
	row := make([]string, cols)
	rows := make([][]string, 0, len(e.tbl.rows)+1)
	rows = append(rows, e.tbl.rows[:at]...)
	rows = append(rows, row)
	rows = append(rows, e.tbl.rows[at:]...)
	e.tbl.rows = rows
	e.tbl.focusR, e.tbl.focusC = at, 0
	e.tableRebuild()
}

// tableDeleteRow takes the row at r away. The header stays: a table without
// one is not a table.
func (e *Editor) tableDeleteRow(r int) {
	if !e.tbl.on || r == 0 || len(e.tbl.rows) <= 2 {
		return
	}
	e.record()
	e.tbl.rows = append(e.tbl.rows[:r], e.tbl.rows[r+1:]...)
	if e.tbl.rows[0] == nil {
		e.tbl.rows[0] = []string{""}
	}
	e.tbl.focusR, e.tbl.focusC = r-1, 0
	e.tableRebuild()
}

// tableAddCol puts an empty column left or right of the one at col.
func (e *Editor) tableAddCol(col int, right bool) {
	if !e.tbl.on {
		return
	}
	e.record()
	at := col
	if right {
		at = col + 1
	}
	for i, r := range e.tbl.rows {
		cells := make([]string, 0, len(r)+1)
		cells = append(cells, r[:at]...)
		cells = append(cells, "")
		cells = append(cells, r[at:]...)
		e.tbl.rows[i] = cells
	}
	align := make([]string, 0, len(e.tbl.align)+1)
	align = append(align, e.tbl.align[:at]...)
	align = append(align, "---")
	align = append(align, e.tbl.align[at:]...)
	e.tbl.align = align
	e.tbl.focusR, e.tbl.focusC = 0, at
	e.tableRebuild()
}

// tableDeleteCol takes the column at col away, and the alignment with it.
func (e *Editor) tableDeleteCol(col int) {
	if !e.tbl.on || len(e.tbl.rows[0]) <= 1 {
		return
	}
	e.record()
	for i, r := range e.tbl.rows {
		e.tbl.rows[i] = append(append([]string{}, r[:col]...), r[col+1:]...)
	}
	if col < len(e.tbl.align) {
		e.tbl.align = append(append([]string{}, e.tbl.align[:col]...), e.tbl.align[col+1:]...)
	}
	at := col
	if at >= len(e.tbl.rows[0]) {
		at = len(e.tbl.rows[0]) - 1
	}
	e.tbl.focusR, e.tbl.focusC = 0, at
	e.tableRebuild()
}

// tableRebuild writes the grid back into the block, and the block into the
// document, as one step.
func (e *Editor) tableRebuild() {
	e.tbl.normalize()
	e.draft = e.tbl.markdown()
	e.sync()
}

// tableMenu is the menu of a cell: the rows and columns around it.
func (e *Editor) tableMenu(m *ui.Menu, r, col int) {
	if m.Item(locales.TableRowAbove()).Chosen() {
		e.tableAddRow(r, false)
	}
	if m.Item(locales.TableRowBelow()).Chosen() {
		e.tableAddRow(r, true)
	}
	if m.Item(locales.TableColLeft()).Chosen() {
		e.tableAddCol(col, false)
	}
	if m.Item(locales.TableColRight()).Chosen() {
		e.tableAddCol(col, true)
	}
	if r > 0 && len(e.tbl.rows) > 2 {
		m.Separator()
		if m.Item(locales.TableDeleteRow()).Chosen() {
			e.tableDeleteRow(r)
		}
	}
	if len(e.tbl.rows[0]) > 1 {
		if r == 0 {
			m.Separator()
		}
		if m.Item(locales.TableDeleteCol()).Chosen() {
			e.tableDeleteCol(col)
		}
	}
}
