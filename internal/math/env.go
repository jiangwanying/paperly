package math

import (
	"fmt"
	"strings"
)

// environment reads the body of \begin{name} up to its \end, and builds
// what the name stands for: a grid of cells for a matrix, one column for a
// case of a function, an aligned block for equations.
func (p *parser) environment() (node, error) {
	name := strings.TrimSpace(p.group())
	rows, err := p.rows(name)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nList{}, nil
	}
	switch name {
	case "matrix", "smallmatrix":
		return nMatrix{rows: rows, gap: 0.9}, nil
	case "pmatrix":
		return nDelim{open: "(", close: ")", body: nMatrix{rows: rows, gap: 0.9}}, nil
	case "bmatrix":
		return nDelim{open: "[", close: "]", body: nMatrix{rows: rows, gap: 0.9}}, nil
	case "Bmatrix":
		return nDelim{open: "{", close: "}", body: nMatrix{rows: rows, gap: 0.9}}, nil
	case "vmatrix":
		return nDelim{open: "|", close: "|", body: nMatrix{rows: rows, gap: 0.9}}, nil
	case "Vmatrix":
		return nDelim{open: "‖", close: "‖", body: nMatrix{rows: rows, gap: 0.9}}, nil
	case "cases", "dcases":
		// A left brace and two left-set columns: the value, and when it
		// holds.
		m := nMatrix{rows: rows, gap: 0.8, align: []colAlign{colLeft, colLeft}}
		return nDelim{open: "{", body: m}, nil
	case "aligned", "align*", "gathered", "gather*":
		return nMatrix{rows: rows, gap: 0.6, align: []colAlign{colRight, colLeft}}, nil
	case "align", "gather":
		return nMatrix{rows: rows, gap: 0.6, align: []colAlign{colRight, colLeft}, number: true}, nil
	}
	return nil, fmt.Errorf("math: unknown environment %s", name)
}

// rows reads an environment's cells: & between the cells of a row, \\\\ between
// the rows, and \end{name} at the close.
func (p *parser) rows(env string) ([][]node, error) {
	var rows [][]node
	var row []node
	for {
		items, term, err := p.cell()
		if err != nil {
			return nil, err
		}
		row = append(row, nList{items: items})
		switch term {
		case "&":
			continue
		case "\\\\":
			rows = append(rows, row)
			row = nil
		case "end":
			// A \\ that ends the last row leaves an empty one behind: TeX
			// sets nothing for it, and neither do we.
			if len(row) > 1 || len(items) > 0 || len(rows) == 0 {
				rows = append(rows, row)
			}
			if got := strings.TrimSpace(p.lastEnd); got != "" && got != env {
				return nil, fmt.Errorf("math: \\begin{%s} closed by \\end{%s}", env, got)
			}
			return rows, nil
		}
	}
}

// cell reads the atoms of one cell, up to the &, \\\\ or \end that closes it.
func (p *parser) cell() ([]node, string, error) {
	var out []node
	for {
		p.space()
		if p.pos >= len(p.src) {
			return nil, "", fmt.Errorf("math: missing \\end")
		}
		switch p.src[p.pos] {
		case '&':
			p.pos++
			return out, "&", nil
		case '}':
			return nil, "", fmt.Errorf("math: unexpected }")
		case '\\':
			switch p.peekName() {
			case "end":
				if err := p.end(); err != nil {
					return nil, "", err
				}
				return out, "end", nil
			case "\\":
				p.pos += 2
				p.breakRoom()
				return out, "\\\\", nil
			}
		}
		n, err := p.atom()
		if err != nil {
			return nil, "", err
		}
		if n == nil {
			continue
		}
		n, err = p.scripts(n)
		if err != nil {
			return nil, "", err
		}
		out = append(out, n)
	}
}

// breakRoom skips the room a \\ asks for, as in \\[4pt]: the pages that
// write it mean the rows to stand apart, which ours do already.
func (p *parser) breakRoom() {
	p.space()
	if p.pos >= len(p.src) || p.src[p.pos] != '[' {
		return
	}
	for p.pos < len(p.src) && p.src[p.pos] != ']' {
		p.pos++
	}
	if p.pos < len(p.src) {
		p.pos++
	}
}

// end reads \end{name} and keeps the name.
func (p *parser) end() error {
	p.pos++ // the backslash
	p.name()
	p.space()
	p.lastEnd = strings.TrimSpace(p.group())
	return nil
}
