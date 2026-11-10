package doc

import "strings"

// Query is what to look for in the document.
type Query struct {
	Text string
	Case bool // match the case of the text
	Word bool // whole words only
}

// Hit is one match: its bytes in the source, and the block they fall in.
// Block is -1 for a match that falls between blocks, in the blank lines of
// the source.
type Hit struct {
	Span
	Block int
}

// FindAll returns every match of q in the document, in the order they read.
func (d *Doc) FindAll(q Query) []Hit {
	if q.Text == "" {
		return nil
	}
	var hits []Hit
	for at := 0; at <= len(d.src); {
		i := indexOf(d.src[at:], q.Text, q.Case)
		if i < 0 {
			break
		}
		s := at + i
		e := s + len(q.Text)
		if q.Word && !wholeWord(d.src, s, e) {
			at = s + 1
			continue
		}
		hits = append(hits, Hit{Span: Span{s, e}, Block: d.BlockAt(s)})
		at = e
	}
	return hits
}

// ReplaceAll writes text in place of every match of q, and returns how many
// there were.
func (d *Doc) ReplaceAll(q Query, text string) int {
	hits := d.FindAll(q)
	if len(hits) == 0 {
		return 0
	}
	var b strings.Builder
	b.Grow(len(d.src))
	last := 0
	for _, h := range hits {
		b.WriteString(d.src[last:h.Start])
		b.WriteString(text)
		last = h.End
	}
	b.WriteString(d.src[last:])
	d.SetSource(b.String())
	return len(hits)
}

// indexOf finds needle in hay, the case its Case says, by bytes: a match of
// a case-folded needle is the same length as the needle, as ASCII letters
// fold within a byte, and the scripts that fold outside one are not folded
// here at all.
func indexOf(hay, needle string, matchCase bool) int {
	if matchCase {
		return strings.Index(hay, needle)
	}
	n := len(needle)
	for i := 0; i+n <= len(hay); i++ {
		if equalFoldAt(hay[i:i+n], needle) {
			return i
		}
	}
	return -1
}

// equalFoldAt reports whether a and b are equal but for the case of ASCII
// letters.
func equalFoldAt(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		x, y := a[i], b[i]
		if 'A' <= x && x <= 'Z' {
			x += 'a' - 'A'
		}
		if 'A' <= y && y <= 'Z' {
			y += 'a' - 'A'
		}
		if x != y {
			return false
		}
	}
	return true
}

// wholeWord reports whether the bytes s to e stand alone: what touches them
// is not the stuff of a word. Only the letters and digits of ASCII, and the
// underscore, are that stuff: a Chinese character beside a match does not
// make it part of a longer word, as few, if any, words are written without a
// break around them, and as \b in the scripts the other editors match with
// reads it.
func wholeWord(src string, s, e int) bool {
	word := func(b byte) bool {
		return b == '_' || ('0' <= b && b <= '9') ||
			('a' <= b && b <= 'z') || ('A' <= b && b <= 'Z')
	}
	if s > 0 && word(src[s-1]) {
		return false
	}
	if e < len(src) && word(src[e]) {
		return false
	}
	return true
}
