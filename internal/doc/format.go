package doc

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// ListKind is the sort of list a paragraph turns into.
type ListKind int

const (
	// ListBullet is a list of items marked with a dash.
	ListBullet ListKind = iota + 1
	// ListOrdered is a numbered list.
	ListOrdered
	// ListTask is a list of check boxes.
	ListTask
)

var (
	reHeadingMark = regexp.MustCompile(`^#{1,6}[ \t]+`)
	reQuoteMark   = regexp.MustCompile(`^>[ \t]?`)
	reTaskMark    = regexp.MustCompile(`^[-*+][ \t]+\[[ xX]\][ \t]+`)
	reBulletMark  = regexp.MustCompile(`^[-*+][ \t]+`)
	reOrderedMark = regexp.MustCompile(`^\d+\.[ \t]+`)
)

// SetHeading makes src a heading of level 1 to 6, or a paragraph for 0: the
// markers it had go, and the ones it should have come.
func SetHeading(src string, level int) string {
	line := FirstLine(src)
	rest := src[len(line):]
	body := reHeadingMark.ReplaceAllString(line, "")
	if level < 1 || level > 6 {
		return body + rest
	}
	return strings.Repeat("#", level) + " " + body + rest
}

// ToggleQuote quotes every line of src, or unquotes them when they all
// already are.
func ToggleQuote(src string) string {
	lines := strings.Split(src, "\n")
	all := true
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		if !reQuoteMark.MatchString(strings.TrimLeft(l, " \t")) {
			all = false
			break
		}
	}
	for i, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		if all {
			lines[i] = reQuoteMark.ReplaceAllString(strings.TrimLeft(l, " \t"), "")
			continue
		}
		if !reQuoteMark.MatchString(strings.TrimLeft(l, " \t")) {
			lines[i] = "> " + l
		}
	}
	return strings.Join(lines, "\n")
}

// ToggleList makes every line of src an item of a list of kind, or takes the
// markers off when they all already are.
func ToggleList(src string, kind ListKind) string {
	lines := strings.Split(src, "\n")
	all := true
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		if !hasListMark(l, kind) {
			all = false
			break
		}
	}
	for i, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		body := stripListMark(l)
		if all {
			lines[i] = body
			continue
		}
		lines[i] = listMark(kind) + body
	}
	return strings.Join(lines, "\n")
}

// ToggleCode puts src in a code fence, or takes the fence off when it has
// one.
func ToggleCode(src string) string {
	trimmed := strings.TrimSpace(src)
	if strings.HasPrefix(trimmed, "```") && strings.HasSuffix(trimmed, "```") && len(trimmed) > 6 {
		lines := strings.Split(strings.Trim(src, "\n"), "\n")
		if len(lines) >= 2 {
			lines = lines[1 : len(lines)-1]
			return strings.Join(lines, "\n")
		}
	}
	return "```\n" + strings.TrimRight(src, "\n") + "\n```"
}

// Wrap puts mark around the runes start to end of src, or takes it off when
// it is already there. It returns the text and where the same text is in it.
// An empty selection gets a pair of marks with the caret between them.
func Wrap(src string, start, end int, mark string) (string, int, int) {
	s, e := ByteOffset(src, start), ByteOffset(src, end)
	if s > e {
		s, e = e, s
	}
	w := utf8.RuneCountInString(mark)
	if e-s >= 2*len(mark) {
		sel := src[s:e]
		if strings.HasPrefix(sel, mark) && strings.HasSuffix(sel, mark) {
			return src[:s] + sel[len(mark):len(sel)-len(mark)] + src[e:], start, end - 2*w
		}
	}
	// The marks may sit just outside the selection, as they do after the
	// run was wrapped and the selection kept: take them off it.
	if s >= len(mark) && e+len(mark) <= len(src) && src[s-len(mark):s] == mark && src[e:e+len(mark)] == mark {
		s -= len(mark)
		e += len(mark)
		return src[:s] + src[s+len(mark):e-len(mark)] + src[e:], start - w, end - w
	}
	if s == e {
		return src[:s] + mark + mark + src[e:], start + w, start + w
	}
	return src[:s] + mark + src[s:e] + mark + src[e:], start + w, end + w
}

// ByteOffset is the byte a rune of s starts at, for runeAt counted in runes
// from its start.
func ByteOffset(s string, runeAt int) int {
	if runeAt <= 0 {
		return 0
	}
	i, n := 0, 0
	for i < len(s) && n < runeAt {
		_, size := utf8.DecodeRuneInString(s[i:])
		i += size
		n++
	}
	return i
}

// hasListMark reports whether line is an item of a list of kind.
func hasListMark(line string, kind ListKind) bool {
	line = strings.TrimLeft(line, " \t")
	switch kind {
	case ListTask:
		return reTaskMark.MatchString(line)
	case ListOrdered:
		return reOrderedMark.MatchString(line)
	default:
		return reBulletMark.MatchString(line) && !reTaskMark.MatchString(line)
	}
}

// stripListMark takes any list marker off a line, keeping its indentation.
func stripListMark(line string) string {
	indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
	body := strings.TrimLeft(line, " \t")
	body = reTaskMark.ReplaceAllString(body, "")
	body = reBulletMark.ReplaceAllString(body, "")
	body = reOrderedMark.ReplaceAllString(body, "")
	return indent + body
}

// listMark is the marker of a list of kind.
func listMark(kind ListKind) string {
	switch kind {
	case ListTask:
		return "- [ ] "
	case ListOrdered:
		return "1. "
	default:
		return "- "
	}
}

// WrapPair puts open before the runes start to end of src and close after
// them, as a link's brackets and address are, or nothing when the run
// already wears them.
func WrapPair(src string, start, end int, open, close string) (string, int, int) {
	s, e := ByteOffset(src, start), ByteOffset(src, end)
	if s > e {
		s, e = e, s
	}
	if s == e {
		return src[:s] + open + close + src[e:],
			start + utf8.RuneCountInString(open), start + utf8.RuneCountInString(open)
	}
	return src[:s] + open + src[s:e] + close + src[e:],
		start + utf8.RuneCountInString(open), end + utf8.RuneCountInString(open)
}

var (
	reInlineCode  = regexp.MustCompile("`([^`\n]+)`")
	reStrongText  = regexp.MustCompile(`\*\*([^*\n]+)\*\*`)
	reStrikeText  = regexp.MustCompile(`~~([^~\n]+)~~`)
	reHiliteText  = regexp.MustCompile(`==([^=\n]+)==`)
	reUnderlined  = regexp.MustCompile(`<u>([^<\n]+)</u>`)
	reEmphText    = regexp.MustCompile(`\*([^*\n]+)\*`)
	reLinkText    = regexp.MustCompile(`\[([^\]\n]+)\]\([^)\n]*\)`)
	reEmphasisLow = regexp.MustCompile(`_([^_\n]+)_`)
)

// StripInline takes the inline marks — strong, emphasis, code, strike,
// highlight, underline and links — off a run of text, as Clear Format does.
func StripInline(s string) string {
	s = reInlineCode.ReplaceAllString(s, "$1")
	s = reStrongText.ReplaceAllString(s, "$1")
	s = reStrikeText.ReplaceAllString(s, "$1")
	s = reHiliteText.ReplaceAllString(s, "$1")
	s = reUnderlined.ReplaceAllString(s, "$1")
	s = reEmphText.ReplaceAllString(s, "$1")
	s = reEmphasisLow.ReplaceAllString(s, "$1")
	s = reLinkText.ReplaceAllString(s, "$1")
	return s
}

// PromoteHeading raises a heading a level, as Typora's 提升标题级别 does; a
// paragraph stays one.
func PromoteHeading(src string) string {
	if level := HeadingLevel(src); level > 1 {
		return SetHeading(src, level-1)
	}
	return src
}

// DemoteHeading lowers a heading a level, as Typora's 降低标题级别 does; a
// paragraph stays one.
func DemoteHeading(src string) string {
	if level := HeadingLevel(src); level >= 1 && level < 6 {
		return SetHeading(src, level+1)
	}
	return src
}

// HeadingLevel is the level of the heading a block starts with, 0 for one
// that is no heading.
func HeadingLevel(src string) int {
	n := 0
	for n < len(src) && n < 6 && src[n] == '#' {
		n++
	}
	if n == 0 || n >= len(src) || (src[n] != ' ' && src[n] != '\t') {
		return 0
	}
	return n
}
