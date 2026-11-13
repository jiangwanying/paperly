package math

import (
	"fmt"
	"strings"
)

// class is what an atom is to the spacing TeX puts between them.
type class uint8

const (
	classOrd class = iota
	classOp
	classBin
	classRel
	classOpen
	classClose
	classPunct
	classInner
)

type node interface{}

// nAtom is a run of text drawn as it is, one atom of the formula.
type nAtom struct {
	text  string
	style faceStyle
	class class
}

// nList is what a group, or the formula, holds in a row.
type nList struct{ items []node }

// nFrac is \frac, and nSqrt a root with its index, if it has one.
type nFrac struct{ num, den node }

// nBinom is \binom: n above k, between fences, with no bar.
type nBinom struct{ num, den node }
type nSqrt struct{ body, index node }

// nScript is an atom with a subscript, a superscript, or both.
type nScript struct{ base, sub, sup node }

// nBig is a sum, a product or an integral, whose limits stand above and
// below it in display style.
type nBig struct {
	text     string
	sub, sup node
}

// nSpace is room the source asks for, in ems.
type nSpace struct{ em float64 }

// nText is upright text within a formula: \text, \mathrm and \mathbf.
type nText struct {
	text  string
	style faceStyle
}

// nBar is \overline: a rule above what it holds.
type nBar struct{ body node }

// nDelim is \left … \right: fences that grow with what they hold.
type nDelim struct {
	open, close string
	body        node
}

// nMatrix is a row-and-column environment: matrix, cases, aligned. The
// columns are set one of three ways, and an align numbers each of its rows
// at the right.
type nMatrix struct {
	rows   [][]node
	align  []colAlign
	gap    float64 // the room between columns, in ems
	number bool
}

// colAlign is how the cells of a column stand in it.
type colAlign uint8

const (
	colCenter colAlign = iota
	colLeft
	colRight
)

// nStyled is a change of font over a group: \mathbf{x+1}.
type nStyled struct {
	style faceStyle
	body  node
}

// sym is a command that stands for a run of characters.
type sym struct {
	text  string
	style faceStyle
	class class
}

// symbols are the commands a formula may use for a character.
var symbols = map[string]sym{
	// Greek, lower case, which maths writes in italic.
	"alpha": {"α", fsMath, classOrd}, "beta": {"β", fsMath, classOrd},
	"gamma": {"γ", fsMath, classOrd}, "delta": {"δ", fsMath, classOrd},
	"epsilon": {"ϵ", fsMath, classOrd}, "varepsilon": {"ε", fsMath, classOrd},
	"zeta": {"ζ", fsMath, classOrd}, "eta": {"η", fsMath, classOrd},
	"theta": {"θ", fsMath, classOrd}, "vartheta": {"ϑ", fsMath, classOrd},
	"iota": {"ι", fsMath, classOrd}, "kappa": {"κ", fsMath, classOrd},
	"lambda": {"λ", fsMath, classOrd}, "mu": {"μ", fsMath, classOrd},
	"nu": {"ν", fsMath, classOrd}, "xi": {"ξ", fsMath, classOrd},
	"pi": {"π", fsMath, classOrd}, "varpi": {"ϖ", fsMath, classOrd},
	"rho": {"ρ", fsMath, classOrd}, "varrho": {"ϱ", fsMath, classOrd},
	"sigma": {"σ", fsMath, classOrd}, "varsigma": {"ς", fsMath, classOrd},
	"tau": {"τ", fsMath, classOrd}, "upsilon": {"υ", fsMath, classOrd},
	"phi": {"ϕ", fsMath, classOrd}, "varphi": {"φ", fsMath, classOrd},
	"chi": {"χ", fsMath, classOrd}, "psi": {"ψ", fsMath, classOrd},
	"omega": {"ω", fsMath, classOrd},
	// Greek, upper case, which maths writes upright.
	"Gamma": {"Γ", fsRoman, classOrd}, "Delta": {"Δ", fsRoman, classOrd},
	"Theta": {"Θ", fsRoman, classOrd}, "Lambda": {"Λ", fsRoman, classOrd},
	"Xi": {"Ξ", fsRoman, classOrd}, "Pi": {"Π", fsRoman, classOrd},
	"Sigma": {"Σ", fsRoman, classOrd}, "Upsilon": {"Υ", fsRoman, classOrd},
	"Phi": {"Φ", fsRoman, classOrd}, "Psi": {"Ψ", fsRoman, classOrd},
	"Omega": {"Ω", fsRoman, classOrd},
	// Binary operators.
	"pm": {"±", fsMath, classBin}, "mp": {"∓", fsMath, classBin},
	"times": {"×", fsMath, classBin}, "div": {"÷", fsMath, classBin},
	"cdot": {"⋅", fsMath, classBin}, "ast": {"∗", fsMath, classBin},
	"star": {"⋆", fsMath, classBin}, "circ": {"∘", fsMath, classBin},
	"bullet": {"∙", fsMath, classBin}, "cup": {"∪", fsMath, classBin},
	"cap": {"∩", fsMath, classBin}, "setminus": {"∖", fsMath, classBin},
	"oplus": {"⊕", fsMath, classBin}, "otimes": {"⊗", fsMath, classBin},
	// Relations.
	"le": {"≤", fsMath, classRel}, "leq": {"≤", fsMath, classRel},
	"ge": {"≥", fsMath, classRel}, "geq": {"≥", fsMath, classRel},
	"ne": {"≠", fsMath, classRel}, "neq": {"≠", fsMath, classRel},
	"approx": {"≈", fsMath, classRel}, "equiv": {"≡", fsMath, classRel},
	"sim": {"∼", fsMath, classRel}, "cong": {"≅", fsMath, classRel},
	"propto": {"∝", fsMath, classRel}, "in": {"∈", fsMath, classRel},
	"notin": {"∉", fsMath, classRel}, "subset": {"⊂", fsMath, classRel},
	"subseteq": {"⊆", fsMath, classRel}, "supset": {"⊃", fsMath, classRel},
	"to": {"→", fsMath, classRel}, "rightarrow": {"→", fsMath, classRel},
	"leftarrow": {"←", fsMath, classRel}, "Rightarrow": {"⇒", fsMath, classRel},
	"Leftarrow": {"⇐", fsMath, classRel}, "leftrightarrow": {"↔", fsMath, classRel},
	"mapsto": {"↦", fsMath, classRel}, "parallel": {"∥", fsMath, classRel},
	"perp": {"⊥", fsMath, classRel},
	// Ordinary symbols.
	"infty": {"∞", fsMath, classOrd}, "partial": {"∂", fsMath, classOrd},
	"nabla": {"∇", fsMath, classOrd}, "forall": {"∀", fsMath, classOrd},
	"exists": {"∃", fsMath, classOrd}, "emptyset": {"∅", fsMath, classOrd},
	"angle": {"∠", fsMath, classOrd}, "ldots": {"…", fsMath, classOrd},
	"cdots": {"⋯", fsMath, classOrd}, "prime": {"′", fsMath, classOrd},
	"degree": {"°", fsMath, classOrd}, "hbar": {"ℏ", fsMath, classOrd},
	"ell": {"ℓ", fsMath, classOrd}, "Re": {"ℜ", fsMath, classOrd},
	"Im": {"ℑ", fsMath, classOrd}, "aleph": {"ℵ", fsMath, classOrd},
	// Functions, which are upright.
	"sin": {"sin", fsRoman, classOp}, "cos": {"cos", fsRoman, classOp},
	"tan": {"tan", fsRoman, classOp}, "cot": {"cot", fsRoman, classOp},
	"sec": {"sec", fsRoman, classOp}, "csc": {"csc", fsRoman, classOp},
	"log": {"log", fsRoman, classOp}, "ln": {"ln", fsRoman, classOp},
	"exp": {"exp", fsRoman, classOp}, "lim": {"lim", fsRoman, classOp},
	"max": {"max", fsRoman, classOp}, "min": {"min", fsRoman, classOp},
	"sup": {"sup", fsRoman, classOp}, "inf": {"inf", fsRoman, classOp},
	"det": {"det", fsRoman, classOp}, "gcd": {"gcd", fsRoman, classOp},
	"arcsin": {"arcsin", fsRoman, classOp}, "arccos": {"arccos", fsRoman, classOp},
	"arctan": {"arctan", fsRoman, classOp}, "sinh": {"sinh", fsRoman, classOp},
	"cosh": {"cosh", fsRoman, classOp}, "tanh": {"tanh", fsRoman, classOp},
}

// bigOps are the operators whose limits stand above and below them.
var bigOps = map[string]string{
	"sum": "∑", "prod": "∏", "coprod": "∐", "int": "∫", "iint": "∬",
	"iiint": "∭", "oint": "∮", "bigcup": "⋃", "bigcap": "⋂",
}

// parser turns a formula's source into nodes.
type parser struct {
	src     []rune
	pos     int
	right   string // the delimiter \right named last
	lastEnd string // the environment the last \end closed
}

// parse reads a formula.
func parse(tex string) (node, error) {
	p := &parser{src: []rune(textOf(tex))}
	items, term, err := p.items(false)
	if err != nil {
		return nil, err
	}
	if term != "" {
		return nil, fmt.Errorf("math: unexpected %s", term)
	}
	return nList{items: items}, nil
}

// items reads atoms until the source ends, a group closes, or \right
// closes a \left: term says which.
func (p *parser) items(inGroup bool) ([]node, string, error) {
	var out []node
	for {
		p.space()
		if p.pos >= len(p.src) {
			if inGroup {
				return nil, "", fmt.Errorf("math: missing }")
			}
			return out, "", nil
		}
		switch p.src[p.pos] {
		case '}':
			if !inGroup {
				return nil, "", fmt.Errorf("math: unexpected }")
			}
			p.pos++
			return out, "}", nil
		case '\\':
			if p.peekName() == "right" {
				if inGroup {
					return nil, "", fmt.Errorf(`math: \right in a group`)
				}
				p.pos += len([]rune(`\right`))
				p.right = p.delim()
				return out, `\right`, nil
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

// scripts takes the sub- and superscripts that follow an atom, if any.
func (p *parser) scripts(base node) (node, error) {
	var sub, sup node
	for {
		p.space()
		if p.pos >= len(p.src) {
			break
		}
		switch p.src[p.pos] {
		case '^':
			p.pos++
			n, err := p.arg()
			if err != nil {
				return nil, err
			}
			sup = n
		case '_':
			p.pos++
			n, err := p.arg()
			if err != nil {
				return nil, err
			}
			sub = n
		default:
			if sub != nil || sup != nil {
				return nScript{base: base, sub: sub, sup: sup}, nil
			}
			return base, nil
		}
	}
	if sub != nil || sup != nil {
		return nScript{base: base, sub: sub, sup: sup}, nil
	}
	return base, nil
}

// arg reads one atom as the argument of a command or a script.
func (p *parser) arg() (node, error) {
	p.space()
	if p.pos >= len(p.src) {
		return nList{}, nil
	}
	if p.src[p.pos] == '{' {
		p.pos++
		items, _, err := p.items(true)
		if err != nil {
			return nil, err
		}
		return nList{items: items}, nil
	}
	n, err := p.atom()
	if err != nil {
		return nil, err
	}
	if n == nil {
		return nList{}, nil
	}
	return n, nil
}

// atom reads one atom: a character, a group, or a command.
func (p *parser) atom() (node, error) {
	if p.pos >= len(p.src) {
		return nil, nil
	}
	c := p.src[p.pos]
	switch c {
	case '{':
		p.pos++
		items, term, err := p.items(true)
		if err != nil {
			return nil, err
		}
		if term != "}" {
			return nil, fmt.Errorf("math: missing }")
		}
		if len(items) == 1 {
			return items[0], nil
		}
		return nList{items: items}, nil
	case '\\':
		return p.command()
	case '&':
		p.pos++
		return nSpace{em: 1}, nil
	case '\'':
		p.pos++
		return nScript{base: nList{}, sup: nAtom{text: "′", style: fsMath, class: classOrd}}, nil
	}
	p.pos++
	return charNode(c), nil
}

// charNode is the atom a character is in maths: letters in italic,
// digits and punctuation as they are.
func charNode(c rune) node {
	switch {
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z':
		return nAtom{text: string(c), style: fsItalic, class: classOrd}
	case c >= '0' && c <= '9':
		return nAtom{text: string(c), style: fsRoman, class: classOrd}
	case c == ' ', c == '\t', c == '\n', c == '\r':
		return nil
	}
	sym := map[rune]sym{
		'+': {"+", fsMath, classBin}, '-': {"−", fsMath, classBin},
		'=': {"=", fsMath, classRel}, '<': {"<", fsMath, classRel},
		'>': {">", fsMath, classRel}, '*': {"∗", fsMath, classBin},
		'/': {"/", fsRoman, classOrd}, ':': {":", fsMath, classRel},
		';': {";", fsMath, classPunct}, ',': {",", fsMath, classPunct},
		'.': {".", fsMath, classOrd}, '!': {"!", fsMath, classClose},
		'?': {"?", fsMath, classOrd}, '(': {"(", fsMath, classOpen},
		')': {")", fsMath, classClose}, '[': {"[", fsMath, classOpen},
		']': {"]", fsMath, classClose}, '|': {"|", fsMath, classOrd},
	}[c]
	if sym.text == "" {
		return nAtom{text: string(c), style: fsMath, class: classOrd}
	}
	return nAtom{text: sym.text, style: sym.style, class: sym.class}
}

// command reads the command at the backslash, and what it builds.
func (p *parser) command() (node, error) {
	p.pos++ // the backslash
	name := p.name()
	switch name {
	case "begin":
		return p.environment()
	case "binom", "dbinom", "tbinom":
		num, err := p.arg()
		if err != nil {
			return nil, err
		}
		den, err := p.arg()
		if err != nil {
			return nil, err
		}
		return nDelim{open: "(", close: ")", body: nBinom{num: num, den: den}}, nil
	case "frac", "dfrac", "tfrac":
		num, err := p.arg()
		if err != nil {
			return nil, err
		}
		den, err := p.arg()
		if err != nil {
			return nil, err
		}
		return nFrac{num: num, den: den}, nil
	case "sqrt":
		var index node
		p.space()
		if p.pos < len(p.src) && p.src[p.pos] == '[' {
			p.pos++
			var items []node
			for {
				p.space()
				if p.pos >= len(p.src) {
					return nil, fmt.Errorf("math: missing ]")
				}
				if p.src[p.pos] == ']' {
					p.pos++
					break
				}
				n, err := p.atom()
				if err != nil {
					return nil, err
				}
				if n != nil {
					items = append(items, n)
				}
			}
			index = nList{items: items}
		}
		body, err := p.arg()
		if err != nil {
			return nil, err
		}
		return nSqrt{body: body, index: index}, nil
	case "text", "textrm", "mbox":
		return nText{text: p.group(), style: fsRoman}, nil
	case "mathrm":
		return nStyled{style: fsRoman, body: p.mustArg()}, nil
	case "mathbb":
		return nAtom{text: blackboard(p.group()), style: fsMath, class: classOrd}, nil
	case "mathbf", "textbf":
		return nStyled{style: fsBold, body: p.mustArg()}, nil
	case "mathit":
		return nStyled{style: fsItalic, body: p.mustArg()}, nil
	case "overline", "bar":
		body, err := p.arg()
		if err != nil {
			return nil, err
		}
		return nBar{body: body}, nil
	case "left":
		open := p.delim()
		body, _, err := p.items(false)
		if err != nil {
			return nil, err
		}
		close := p.right
		p.right = ""
		if close == "" {
			return nil, fmt.Errorf(`math: \left without \right`)
		}
		return nDelim{open: open, close: close, body: nList{items: body}}, nil
	case "right":
		return nil, fmt.Errorf(`math: \right without \left`)
	case "quad":
		return nSpace{em: 1}, nil
	case "qquad":
		return nSpace{em: 2}, nil
	case ",":
		return nSpace{em: 3.0 / 18}, nil
	case ":", ">":
		return nSpace{em: 4.0 / 18}, nil
	case ";":
		return nSpace{em: 5.0 / 18}, nil
	case "!":
		return nSpace{em: -3.0 / 18}, nil
	case " ":
		return nSpace{em: 0.25}, nil
	case "{", "}", "%", "$", "&", "#", "_":
		return nAtom{text: name, style: fsMath, class: classOrd}, nil
	case "\\":
		return nList{}, nil
	}
	if text, ok := bigOps[name]; ok {
		b := nBig{text: text}
		// The limits are read by scripts(), which follows.
		return b, nil
	}
	if s, ok := symbols[name]; ok {
		return nAtom{text: s.text, style: s.style, class: s.class}, nil
	}
	return nil, fmt.Errorf("math: unknown command \\%s", name)
}

// blackboard is a letter as the blackboard bold of the number sets: the
// math font has the sets' own letters, and the rest stand upright.
func blackboard(s string) string {
	sets := map[rune]string{
		'C': "ℂ", 'H': "ℍ", 'N': "ℕ", 'P': "ℙ", 'Q': "ℚ", 'R': "ℝ", 'Z': "ℤ",
	}
	var b strings.Builder
	for _, r := range s {
		if g, ok := sets[r]; ok {
			b.WriteString(g)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// mustArg is an argument that a command needs: a missing one is empty.
func (p *parser) mustArg() node {
	n, err := p.arg()
	if err != nil {
		return nList{}
	}
	return n
}

// name reads a command's name: its letters, or the one character after
// the backslash when it is not a letter.
func (p *parser) name() string {
	start := p.pos
	for p.pos < len(p.src) {
		c := p.src[p.pos]
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' {
			p.pos++
			continue
		}
		break
	}
	if p.pos == start && p.pos < len(p.src) {
		p.pos++
	}
	return string(p.src[start:p.pos])
}

// peekName is the name of the command at the backslash, unread.
func (p *parser) peekName() string {
	save := p.pos
	p.pos++
	name := p.name()
	p.pos = save
	return name
}

// group reads the braces' text as it is, spaces and all: \text{…}.
func (p *parser) group() string {
	p.space()
	if p.pos >= len(p.src) || p.src[p.pos] != '{' {
		return ""
	}
	p.pos++
	depth := 1
	var b strings.Builder
	for p.pos < len(p.src) {
		c := p.src[p.pos]
		p.pos++
		switch c {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return b.String()
			}
		}
		b.WriteRune(c)
	}
	return b.String()
}

// delim reads the delimiter a \left or \right names.
func (p *parser) delim() string {
	p.space()
	if p.pos >= len(p.src) {
		return ""
	}
	if p.src[p.pos] == '\\' {
		p.pos++
		name := p.name()
		switch name {
		case "langle":
			return "⟨"
		case "rangle":
			return "⟩"
		case "lfloor":
			return "⌊"
		case "rfloor":
			return "⌋"
		case "lceil":
			return "⌈"
		case "rceil":
			return "⌉"
		case "{":
			return "{"
		case "}":
			return "}"
		case "|":
			return "‖"
		}
		return ""
	}
	c := p.src[p.pos]
	p.pos++
	if c == '.' {
		return ""
	}
	return string(c)
}

// space skips the room TeX ignores between atoms.
func (p *parser) space() {
	for p.pos < len(p.src) {
		switch p.src[p.pos] {
		case ' ', '\t', '\n', '\r':
			p.pos++
		default:
			return
		}
	}
}
