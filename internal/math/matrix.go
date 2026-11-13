package math

import (
	"math"
	"strconv"
)

// matrix sets rows of cells in columns, with the room between them that TeX
// gives an array: a column as wide as its widest cell, a row as tall as its
// tallest, and the whole centred on the axis of the line it stands on.
func (l *layout) matrix(n nMatrix, st style) (box, error) {
	if len(n.rows) == 0 {
		return box{}, nil
	}
	cells := make([][]box, len(n.rows))
	cols := 0
	for i, row := range n.rows {
		cells[i] = make([]box, len(row))
		for j, c := range row {
			b, err := l.render(c, st)
			if err != nil {
				return box{}, err
			}
			cells[i][j] = b
		}
		cols = max(cols, len(row))
	}
	if cols == 0 {
		return box{}, nil
	}

	// A column is as wide as its widest cell.
	widths := make([]float64, cols)
	for i := range cells {
		for j, b := range cells[i] {
			widths[j] = math.Max(widths[j], b.w)
		}
	}
	gap := n.gap * l.em
	colsX := make([]float64, cols)
	total := 0.0
	for j, w := range widths {
		colsX[j] = total
		total += w
		if j < cols-1 {
			total += gap
		}
	}

	// A row is as tall as its tallest cell above and below the baseline.
	asc := make([]float64, len(cells))
	desc := make([]float64, len(cells))
	for i := range cells {
		for _, b := range cells[i] {
			asc[i] = math.Max(asc[i], b.h)
			desc[i] = math.Max(desc[i], b.d)
		}
	}
	rowGap := l.em * 0.45
	ys := make([]float64, len(cells))
	for i := 1; i < len(cells); i++ {
		ys[i] = ys[i-1] - (desc[i-1] + rowGap + asc[i])
	}
	bottom := ys[len(ys)-1] - desc[len(ys)-1]

	// An array hangs on the axis of the line, not on its own bottom: what
	// it holds is read as one piece of the formula.
	dy := l.axis() - (asc[0]+bottom)/2

	var out box
	for i := range cells {
		for j, b := range cells[i] {
			x := colsX[j]
			switch alignAt(n.align, j) {
			case colCenter:
				x += (widths[j] - b.w) / 2
			case colRight:
				x += widths[j] - b.w
			}
			out = appendAt(out, raised(b, ys[i]+dy), x)
		}
	}
	out.w = total

	if n.number {
		// The number of the row stands at the right, as TeX puts a tag:
		// clear of the longest line, at the size of the text.
		for i := range cells {
			tag := l.text("("+strconv.Itoa(i+1)+")", fsRoman, l.em*0.9)
			x := total + l.em*1.4
			out = appendAt(out, raised(tag, ys[i]+dy), x)
			out.w = math.Max(out.w, x+tag.w)
		}
	}
	return out, nil
}

// alignAt is how column j stands: the list names them in turn, and a list of
// right and left—an aligned block—goes on alternating for the columns past
// its end, as TeX aligns equations.
func alignAt(a []colAlign, j int) colAlign {
	if len(a) == 0 {
		return colCenter
	}
	if len(a) == 2 && a[0] == colRight && a[1] == colLeft {
		if j%2 == 0 {
			return colRight
		}
		return colLeft
	}
	if j < len(a) {
		return a[j]
	}
	return a[len(a)-1]
}
