package doc

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// RunKind is what a run of the source is, while a block is written.
type RunKind uint8

const (
	// RunText is the text itself.
	RunText RunKind = iota
	// RunMarker is syntax that shows faint while the caret is on its line,
	// and hides on the others: the "**" around strong text.
	RunMarker
	// RunPrefix is the marker a line opens with — the "- " of an item, the
	// "> " of a quote, the "#" of a heading, a fence. It stays faint on
	// every line: hidden, a list would lose its bullets, a quote its mark.
	RunPrefix
	// RunStrong is strong text, RunEmph emphasized text, RunCode code.
	RunStrong
	RunEmph
	RunCode
)

// Run is a range of bytes and the weight its text shows at.
type Run struct {
	Span
	Kind   RunKind
	Weight int
}

var (
	// A line's own marker, as each line of a multi-line block has one: an
	// item, a quote, a heading, and the checkbox of a task before the
	// bullet it rides on.
	reLinePrefix = regexp.MustCompile(`(?m)^([ \t]*(?:#{1,6}[ \t]|>[ \t]|[-*+][ \t]+\[[xX ]\][ \t]|[-*+][ \t]|\d+[.)][ \t]))`)
	// A fence opens or closes a code block on a line of its own.
	reFenceLine = regexp.MustCompile("(?m)^([ \\t]*(?:```+|~~~+)[ \\t]*)")
	reStrong    = regexp.MustCompile(`\*\*([^*\n]+)\*\*`)
	reEmph      = regexp.MustCompile(`\*([^*\n]+)\*`)
	reCode      = regexp.MustCompile("`([^`\\n]+)`")
	reRule      = regexp.MustCompile(`^(?:---+|\*\*\*+)\s*$`)
)

// FirstLine is the line a block starts with.
func FirstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// BlockSize is the size the text shows at while it is written: a heading is
// as large as it will be once the block is done, so typing "# " turns the
// line into one at once.
func BlockSize(s string) float32 {
	line := FirstLine(s)
	if reRule.MatchString(strings.TrimSpace(line)) {
		return 16
	}
	trimmed := strings.TrimLeft(line, " \t")
	switch {
	case strings.HasPrefix(trimmed, "##### "), strings.HasPrefix(trimmed, "###### "):
		return 16
	case strings.HasPrefix(trimmed, "#### "):
		return 17
	case strings.HasPrefix(trimmed, "### "):
		return 20
	case strings.HasPrefix(trimmed, "## "):
		return 24
	case strings.HasPrefix(trimmed, "# "):
		return 32
	}
	return 16
}

// SyntaxRuns finds the markers and the inline spans of a block being
// written: typing "**粗**" shows 粗 in bold with the asterisks faint, before
// the block is even submitted.
func SyntaxRuns(s string) []Run {
	var out []Run
	add := func(from, to int, kind RunKind, weight int) {
		if from < to {
			out = append(out, Run{Span: Span{Start: from, End: to}, Kind: kind, Weight: weight})
		}
	}

	if m := reRule.FindStringIndex(s); m != nil {
		add(m[0], m[1], RunPrefix, 0)
	}
	// Every line of a multi-line block has a marker of its own, as a list
	// of three items has three dashes.
	for _, m := range reLinePrefix.FindAllStringSubmatchIndex(s, -1) {
		add(m[2], m[3], RunPrefix, 0)
	}
	for _, m := range reFenceLine.FindAllStringSubmatchIndex(s, -1) {
		add(m[2], m[3], RunPrefix, 0)
	}

	for _, m := range reStrong.FindAllStringSubmatchIndex(s, -1) {
		add(m[0], m[2], RunMarker, 0)
		add(m[2], m[3], RunStrong, 700)
		add(m[3], m[1], RunMarker, 0)
	}
	// The asterisks inside "**bold**" read as one strong span, not as an
	// emphasis between them.
	inStrong := func(at int) bool {
		for _, m := range reStrong.FindAllStringSubmatchIndex(s, -1) {
			if at >= m[0] && at < m[1] {
				return true
			}
		}
		return false
	}
	for _, m := range reEmph.FindAllStringSubmatchIndex(s, -1) {
		if inStrong(m[0]) {
			continue
		}
		add(m[0], m[2], RunMarker, 0)
		add(m[2], m[3], RunEmph, 550)
		add(m[3], m[1], RunMarker, 0)
	}
	for _, m := range reCode.FindAllStringSubmatchIndex(s, -1) {
		add(m[0], m[2], RunMarker, 0)
		add(m[2], m[3], RunCode, 500)
		add(m[3], m[1], RunMarker, 0)
	}
	return out
}

// LineOf is the line an offset stands on, as the breaks before it count.
func LineOf(s string, at int) int {
	if at <= 0 {
		return 0
	}
	if at > len(s) {
		at = len(s)
	}
	return strings.Count(s[:at], "\n")
}

// RuneOffset is the offset in runes of a byte offset in s.
func RuneOffset(s string, at int) int {
	if at <= 0 {
		return 0
	}
	if at > len(s) {
		at = len(s)
	}
	return utf8.RuneCountInString(s[:at])
}

// RuneLen is how many runes s holds.
func RuneLen(s string) int { return utf8.RuneCountInString(s) }
