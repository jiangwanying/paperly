package math

import (
	"math"

	"golang.org/x/image/font"
)

// A box is a piece of a formula: how wide it is, how far it reaches above
// the baseline (h) and below it (d), and the runs of text and rules that
// make it, placed on that baseline.
type box struct {
	w, h, d float64
	runs    []run
	rules   []rule
}

// run is a run of text at a place in its box: x from the box's left, y
// above its baseline.
type run struct {
	x, y  float64
	text  string
	style faceStyle
	size  float64
	cjk   bool // set in the Chinese face, which Latin cannot draw
}

// rule is a line of the formula, as a fraction's bar or a root's roof: x
// from the box's left, y from its baseline to the rule's bottom, and its
// thickness.
type rule struct {
	x, y, w, h float64
}

// style is how large a part of a formula stands: the formula itself, or
// what a script, or a script's script, sets smaller.
type style uint8

const (
	styleDisplay style = iota
	styleText
	styleScript
	styleScriptScript
)

// scale is how much of the em a style takes.
func (s style) scale() float64 {
	switch s {
	case styleScript:
		return 0.7
	case styleScriptScript:
		return 0.55
	}
	return 1
}

// fracPart is the style a fraction's parts are set in: a display's
// numerator keeps the size of the text.
func (s style) fracPart() style {
	switch s {
	case styleDisplay:
		return styleText
	case styleText:
		return styleScript
	}
	return styleScriptScript
}

// script is the style a script is set in.
func (s style) script() style {
	if s == styleScript || s == styleScriptScript {
		return styleScriptScript
	}
	return styleScript
}

// layout draws a formula at one em: the size the caller asked for, times
// the picture's scale.
type layout struct {
	em    float64 // pixels
	force *faceStyle
}

// The room a formula keeps, as fractions of the em, as TeX has them.
func (l *layout) axis() float64       { return l.em * 0.25 }
func (l *layout) rule() float64       { return math.Max(l.em*0.045, 0.6) }
func (l *layout) thinSpace() float64  { return l.em * 3 / 18 }
func (l *layout) medSpace() float64   { return l.em * 4 / 18 }
func (l *layout) thickSpace() float64 { return l.em * 5 / 18 }

// render builds the box of a node in a style.
func (l *layout) render(n node, st style) (box, error) {
	switch n := n.(type) {
	case nil:
		return box{}, nil
	case nList:
		return l.list(n.items, st)
	case nAtom:
		return l.atom(n, st), nil
	case nText:
		return l.text(n.text, n.style, l.em*st.scale()), nil
	case nSpace:
		return box{w: n.em * l.em}, nil
	case nFrac:
		return l.frac(n, st)
	case nBinom:
		return l.binom(n, st)
	case nSqrt:
		return l.sqrt(n, st)
	case nScript:
		return l.scripts(n, st)
	case nBig:
		return l.big(n, st)
	case nBar:
		return l.bar(n, st)
	case nDelim:
		return l.delim(n, st)
	case nMatrix:
		return l.matrix(n, st)
	case nStyled:
		old := l.force
		f := n.style
		l.force = &f
		b, err := l.render(n.body, st)
		l.force = old
		return b, err
	}
	return box{}, nil
}

// list sets atoms in a row, with the room TeX puts between their kinds.
func (l *layout) list(items []node, st style) (box, error) {
	var boxes []box
	var classes []class
	for _, it := range items {
		b, err := l.render(it, st)
		if err != nil {
			return box{}, err
		}
		if b.w == 0 && b.h == 0 && b.d == 0 {
			continue
		}
		boxes = append(boxes, b)
		classes = append(classes, classOf(it))
	}
	var out box
	x := 0.0
	for i, b := range boxes {
		// A sign with nothing to its left is not a difference: TeX sets a
		// leading minus— after an opening fence, a relation, another
		// operator, or at the start of the row—with no room around it.
		if classes[i] == classBin && (i == 0 || classes[i-1] != classOrd &&
			classes[i-1] != classClose && classes[i-1] != classInner) {
			classes[i] = classOrd
		}
		if i > 0 {
			x += l.gap(classes[i-1], classes[i], st)
		}
		out = appendAt(out, b, x)
		x += b.w
	}
	out.w = x
	return out, nil
}

// atom is one run of text, in the font its kind asks for.
func (l *layout) atom(n nAtom, st style) box {
	size := l.em * st.scale()
	face := n.style
	if l.force != nil {
		face = *l.force
	}
	return l.text(n.text, face, size)
}

// text is a run measured on the baseline: how far it reaches above and
// below it comes from the glyphs themselves.
func (l *layout) text(s string, st faceStyle, size float64) box {
	if s == "" {
		return box{}
	}
	var b box
	x := 0.0
	rs := []rune(s)
	for i := 0; i < len(rs); {
		// A run of one face: Chinese is not in Latin Modern, and the room
		// it takes is its own.
		cjk := inCJK(st, rs[i])
		j := i + 1
		for j < len(rs) && inCJK(st, rs[j]) == cjk {
			j++
		}
		part := string(rs[i:j])
		f := face(st, size)
		w := width(st, size, part)
		if cjk {
			f = cjkFace(size)
			w = cjkWidth(size, part)
		}
		if f != nil {
			for _, r := range part {
				if r == ' ' {
					continue
				}
				if gb, _, ok := f.GlyphBounds(r); ok {
					b.h = math.Max(b.h, float64(-gb.Min.Y)/64)
					b.d = math.Max(b.d, float64(gb.Max.Y)/64)
				}
			}
		}
		b.runs = append(b.runs, run{x: x, y: 0, text: part, style: st, size: size, cjk: cjk})
		x += w
		i = j
	}
	b.w = x
	if b.h == 0 {
		b.h = size * 0.7
	}
	return b
}

// inCJK reports whether r has to be set in the Chinese face: the Latin one
// has no glyph for it.
func inCJK(st faceStyle, r rune) bool {
	if r == ' ' || hasRune(st, r) {
		return false
	}
	return cjkHas(r)
}

// cjkWidth is how wide s is in the Chinese face, a whole em for a character
// the face does not have.
func cjkWidth(size float64, s string) float64 {
	f := cjkFace(size)
	if f == nil {
		return float64(len([]rune(s))) * size
	}
	return float64(font.MeasureString(f, s)) / 64
}

// frac sets a fraction: the numerator above the bar, the denominator
// below it, the bar on the axis of the formula.
func (l *layout) frac(n nFrac, st style) (box, error) {
	num, err := l.render(n.num, st.fracPart())
	if err != nil {
		return box{}, err
	}
	den, err := l.render(n.den, st.fracPart())
	if err != nil {
		return box{}, err
	}
	axis := l.axis()
	bar := l.rule()
	gap := l.em * 0.12
	var out box
	out.w = math.Max(num.w, den.w) + l.em*0.15
	// The numerator stands on the bar, its descend beside it.
	ny := axis + bar/2 + gap + num.d
	out = appendAt(out, raised(num, ny), (out.w-num.w)/2)
	// The denominator hangs under the bar.
	dy := axis - bar/2 - gap - den.h
	out = appendAt(out, raised(den, dy), (out.w-den.w)/2)
	out.rules = append(out.rules, rule{x: 0, y: axis - bar/2, w: out.w, h: bar})
	return out, nil
}

// binom sets a choice: the two parts stacked on the axis, no bar between
// them, in the fences the source put around them.
func (l *layout) binom(n nBinom, st style) (box, error) {
	num, err := l.render(n.num, st.fracPart())
	if err != nil {
		return box{}, err
	}
	den, err := l.render(n.den, st.fracPart())
	if err != nil {
		return box{}, err
	}
	gap := l.em * 0.1
	var out box
	out.w = math.Max(num.w, den.w)
	out = appendAt(out, raised(num, gap+num.d), (out.w-num.w)/2)
	out = appendAt(out, raised(den, -gap-den.h), (out.w-den.w)/2)
	return out, nil
}

// sqrt sets a root: the radical, grown to hold what is under it, and the
// roof that runs over it.
func (l *layout) sqrt(n nSqrt, st style) (box, error) {
	body, err := l.render(n.body, st)
	if err != nil {
		return box{}, err
	}
	bar := l.rule()
	// The radical grows to hold the body, and is never set smaller: a
	// radical drawn at a size below its own would come out with strokes
	// too thin for the formula around it.
	want := body.h + body.d + l.em*0.12
	rad := l.sized("√", fsMath, want)
	// The radical's foot stands where the body's lowest ink does.
	rad = raised(rad, rad.d-body.d)
	// The bar runs from the top of the hook, so that the two meet: the
	// glyph's roof reaches no higher than the box it is measured in.
	roofY := rad.h - bar
	var out box
	out = appendAt(out, rad, 0)
	out = appendAt(out, body, rad.w-bar*2)
	out.rules = append(out.rules, rule{x: rad.w - bar*2, y: roofY, w: body.w + bar*2.5, h: bar})
	out.h = math.Max(out.h, roofY+bar)
	out.d = math.Max(out.d, body.d)
	if n.index != nil {
		idx, err := l.render(n.index, st.script())
		if err != nil {
			return box{}, err
		}
		ix := math.Max(0, rad.w*0.35)
		out = appendAt(out, idx, ix)
		out.h = math.Max(out.h, roofY+idx.h+l.em*0.05)
		out.w = math.Max(out.w, ix+idx.w)
	}
	return out, nil
}

// fitted draws one character at the size that makes its ink as tall as
// want pixels, at most.
// sized sets a glyph at the height asked for, larger or smaller: an
// operator within a line of text is smaller than the same operator with a
// line to itself, and its glyph has to be drawn so.
func (l *layout) sized(s string, st faceStyle, want float64) box {
	size := l.em
	b := l.text(s, st, size)
	for i := 0; i < 4; i++ {
		h := b.h + b.d
		if h <= 0 {
			break
		}
		k := want / h
		if math.Abs(k-1) < 0.02 {
			break
		}
		size *= k
		b = l.text(s, st, size)
	}
	return b
}

func (l *layout) fitted(s string, st faceStyle, want float64) box {
	size := l.em
	b := l.text(s, st, size)
	for i := 0; i < 12 && b.h+b.d < want; i++ {
		size *= 1.1
		b = l.text(s, st, size)
	}
	return b
}

// scripts sets an atom with what follows it, above and below.
func (l *layout) scripts(n nScript, st style) (box, error) {
	// A sum's limits stand above and below it, an integral's beside it:
	// the operator sets them itself.
	if op, ok := n.base.(nBig); ok {
		if n.sub != nil {
			op.sub = n.sub
		}
		if n.sup != nil {
			op.sup = n.sup
		}
		return l.big(op, st)
	}
	base, err := l.render(n.base, st)
	if err != nil {
		return box{}, err
	}
	out := base
	supY := math.Max(l.em*0.42, base.h-l.em*0.15)
	subY := -math.Max(l.em*0.2, 0)
	var sub, sup box
	if n.sup != nil {
		if sup, err = l.render(n.sup, st.script()); err != nil {
			return box{}, err
		}
	}
	if n.sub != nil {
		if sub, err = l.render(n.sub, st.script()); err != nil {
			return box{}, err
		}
	}
	if n.sup != nil {
		x := base.w + l.em*0.04
		out = appendAt(out, raised(sup, supY), x)
	}
	if n.sub != nil {
		y := subY
		if n.sup != nil {
			y = math.Min(y, supY-sup.d-l.em*0.2-sub.h)
		}
		x := base.w + l.em*0.04
		out = appendAt(out, raised(sub, y), x)
	}
	return out, nil
}

// big sets a sum or an integral: the operator grown, and its limits above
// and below it, or beside it for an integral.
func (l *layout) big(n nBig, st style) (box, error) {
	// An integral's limits stand beside it in either style; a sum's stand
	// above and below it only when it has a line to itself, as TeX sets
	// them, and beside it within a line of text.
	integral := n.text == "∫" || n.text == "∬" || n.text == "∭" || n.text == "∮"
	side := integral || st != styleDisplay
	var op box
	if st == styleDisplay {
		height := l.em * 1.5
		if integral {
			height = l.em * 2.1
		}
		op = l.fitted(n.text, fsMath, height)
	} else {
		// Within a line of text the operator is set smaller, as TeX sets
		// it, so that the line it stands in grows as little as may be.
		want := l.em * 1.35
		if integral {
			want = l.em * 1.6
		}
		op = l.sized(n.text, fsMath, want)
	}
	// The operator stands on the axis, not on the baseline.
	axis := l.axis()
	op = raised(op, axis-(op.h-op.d)/2)

	var sub, sup box
	var err error
	if n.sup != nil {
		if sup, err = l.render(n.sup, st.script()); err != nil {
			return box{}, err
		}
	}
	if n.sub != nil {
		if sub, err = l.render(n.sub, st.script()); err != nil {
			return box{}, err
		}
	}
	var out box
	out = appendAt(out, op, 0)
	if side {
		x := op.w + l.em*0.05
		if n.sup != nil {
			out = appendAt(out, raised(sup, axis+l.em*0.5), x)
		}
		if n.sub != nil {
			out = appendAt(out, raised(sub, axis-l.em*0.5-sub.h), x)
		}
		return out, nil
	}
	if n.sup != nil {
		y := op.h + l.em*0.12 + sup.d
		out = appendAt(out, raised(sup, y), (op.w-sup.w)/2)
	}
	if n.sub != nil {
		y := -(op.d + l.em*0.18 + sub.h)
		out = appendAt(out, raised(sub, y), (op.w-sub.w)/2)
	}
	out.w = math.Max(out.w, math.Max(sup.w, sub.w))
	return out, nil
}

// bar sets a rule over what it holds.
func (l *layout) bar(n nBar, st style) (box, error) {
	body, err := l.render(n.body, st)
	if err != nil {
		return box{}, err
	}
	bar := l.rule()
	out := body
	out.rules = append(out.rules, rule{x: 0, y: body.h + l.em*0.1, w: body.w, h: bar})
	out.h = body.h + l.em*0.1 + bar
	return out, nil
}

// delim sets fences that grow with what they hold.
func (l *layout) delim(n nDelim, st style) (box, error) {
	body, err := l.render(n.body, st)
	if err != nil {
		return box{}, err
	}
	want := body.h + body.d + l.em*0.2
	var out box
	x := 0.0
	if n.open != "" {
		b := l.fitted(n.open, fsMath, want)
		b = raised(b, (body.h-body.d)/2-(b.h-b.d)/2)
		out = appendAt(out, b, 0)
		x = b.w
	}
	out = appendAt(out, body, x)
	out.w = math.Max(out.w, x+body.w)
	if n.close != "" {
		b := l.fitted(n.close, fsMath, want)
		b = raised(b, (body.h-body.d)/2-(b.h-b.d)/2)
		out = appendAt(out, b, out.w)
	}
	out.h = math.Max(out.h, body.h)
	out.d = math.Max(out.d, body.d)
	return out, nil
}

// appendAt adds a box to another, moved right by x.
func appendAt(dst, src box, x float64) box {
	for _, r := range src.runs {
		r.x += x
		dst.runs = append(dst.runs, r)
	}
	for _, r := range src.rules {
		r.x += x
		dst.rules = append(dst.rules, r)
	}
	dst.w = math.Max(dst.w, x+src.w)
	dst.h = math.Max(dst.h, src.h)
	dst.d = math.Max(dst.d, src.d)
	return dst
}

// raised moves a box up by y, its baseline with it.
func raised(b box, y float64) box {
	for i := range b.runs {
		b.runs[i].y += y
	}
	for i := range b.rules {
		b.rules[i].y += y
	}
	b.h += y
	b.d -= y
	return b
}

// classOf is what an atom counts as when the room between atoms is set.
func classOf(n node) class {
	switch n := n.(type) {
	case nAtom:
		return n.class
	case nBig:
		return classOp
	case nDelim:
		if n.open != "" {
			return classOpen
		}
		return classClose
	case nMatrix:
		return classInner
	case nList:
		if len(n.items) == 0 {
			return classOrd
		}
		return classOf(n.items[0])
	case nScript:
		return classOf(n.base)
	case nStyled:
		return classOf(n.body)
	case nBar:
		return classOf(n.body)
	}
	return classOrd
}

// gap is the room TeX puts between two atoms of these kinds: none beside
// an opening or a closing fence, a thin space beside a function, more
// around a relation.
func (l *layout) gap(a, b class, st style) float64 {
	if a == classOpen || b == classClose || b == classOpen || a == classClose {
		return 0
	}
	if a == classPunct || b == classPunct {
		return 0
	}
	// A script keeps only the thin spaces: in a superscript there is no
	// room for TeX's medium and thick ones, and its spacing table is not
	// used at all.
	script := st == styleScript || st == styleScriptScript
	if a == classRel || b == classRel {
		if script {
			return 0
		}
		return l.thickSpace()
	}
	if a == classBin || b == classBin {
		if script {
			return 0
		}
		return l.medSpace()
	}
	if a == classOp || b == classOp || a == classInner || b == classInner {
		return l.thinSpace()
	}
	return 0
}
