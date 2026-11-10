// Package doc holds the source of a Markdown document and the blocks it
// parses into. It depends on nothing of the user interface, so tests of it
// run in microseconds.
package doc

import (
	"strings"

	"github.com/egoist/mygo/ui/markdown/parser"
)

// Span is a range of bytes in the source, its start included and its end
// excluded.
type Span struct{ Start, End int }

// Block is one top-level block of the document.
type Block struct {
	Index int
	Span
	Kind  parser.Kind
	Level int // a heading's, from 1 to 6
	Line  int // the 1-based line its first byte is on
}

// Doc is a document: its source, the parse of it, and its top-level blocks.
type Doc struct {
	src    string
	p      *parser.Document
	blocks []Block
	rev    uint64
}

// New parses src into a document.
func New(src string) *Doc {
	d := &Doc{src: src, p: parser.Parse(src)}
	d.refresh()
	return d
}

// Source is the document's bytes.
func (d *Doc) Source() string { return d.src }

// Revision changes whenever the source does.
func (d *Doc) Revision() uint64 { return d.rev }

// Parse is the parsed document, for what needs more than the blocks: the
// lines of a fence, a table's cells, the inlines of a paragraph.
func (d *Doc) Parse() *parser.Document { return d.p }

// Blocks are the top-level blocks, in document order.
func (d *Doc) Blocks() []Block { return d.blocks }

// Len is how many top-level blocks the document has.
func (d *Doc) Len() int { return len(d.blocks) }

// Block returns the block at i.
func (d *Doc) Block(i int) (Block, bool) {
	if i < 0 || i >= len(d.blocks) {
		return Block{}, false
	}
	return d.blocks[i], true
}

// SetSource replaces the whole source, parsing again what changed: the
// parser bounds the change by the common prefix and suffix of the two.
func (d *Doc) SetSource(src string) {
	if src == d.src {
		return
	}
	d.p.Set(src)
	d.src = d.p.Source()
	d.rev++
	d.refresh()
}

// Replace swaps the bytes of s for text, parsing again from the block before
// the change rather than the whole document.
func (d *Doc) Replace(s Span, text string) {
	if s.Start < 0 {
		s.Start = 0
	}
	if s.End > len(d.src) {
		s.End = len(d.src)
	}
	if s.Start > s.End {
		s.Start, s.End = s.End, s.Start
	}
	d.p.Edit(s.Start, s.End, text)
	d.src = d.p.Source()
	d.rev++
	d.refresh()
}

// BlockAt is the index of the block holding offset, or -1 in the blank lines
// between blocks.
func (d *Doc) BlockAt(offset int) int {
	for i, b := range d.blocks {
		// An empty block — a heading whose # has no text yet — holds its
		// one offset, or typing it would drop the writer out of the block.
		if offset >= b.Start && (offset < b.End || (b.Start == b.End && offset == b.Start)) {
			return i
		}
	}
	return -1
}

// LineOf is the 1-based line offset is on.
func (d *Doc) LineOf(offset int) int {
	if offset < 0 {
		offset = 0
	}
	if offset > len(d.src) {
		offset = len(d.src)
	}
	return strings.Count(d.src[:offset], "\n") + 1
}

// OffsetOfLine is the byte its first character is at, or the end of the
// source past the last line.
func (d *Doc) OffsetOfLine(line int) int {
	if line <= 1 {
		return 0
	}
	n, off := 1, 0
	for n < line {
		i := strings.IndexByte(d.src[off:], '\n')
		if i < 0 {
			return len(d.src)
		}
		off += i + 1
		n++
	}
	return off
}

// refresh rebuilds the block table, counting the lines in one pass.
func (d *Doc) refresh() {
	d.blocks = d.blocks[:0]
	line, pos := 1, 0
	for i := range d.p.Children(0) {
		b := d.p.Block(i)
		if b.Start > pos {
			line += strings.Count(d.src[pos:b.Start], "\n")
			pos = b.Start
		}
		d.blocks = append(d.blocks, Block{
			Index: i,
			Span:  Span{Start: b.Start, End: b.End},
			Kind:  b.Kind,
			Level: b.Level,
			Line:  line,
		})
	}
}
