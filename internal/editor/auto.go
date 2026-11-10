package editor

import (
	"fmt"
	"regexp"
	"strconv"

	"paperly/internal/doc"
)

// The markers a list item, a quote or a fence opens with, as the writer
// typed them.
var (
	reTaskMark  = regexp.MustCompile(`^[-*+] \[[ xX]\][ \t]+`)
	reNumMark   = regexp.MustCompile(`^(\d+)([.)])[ \t]+`)
	reBullet    = regexp.MustCompile(`^([-*+])[ \t]+`)
	reQuoteMark = regexp.MustCompile(`^>[ \t]`)
	reFence     = regexp.MustCompile("^```([\\w+-]*)[ \\t]*$")
)

// listMark is the marker the draft opens with: the kind of list it makes,
// whether it is a quote, the mark as written, and the number of an ordered
// one. ok reports whether there was one.
func listMark(draft string) (kind doc.ListKind, quote bool, mark string, num int, ok bool) {
	if m := reTaskMark.FindString(draft); m != "" {
		return doc.ListTask, false, m, 0, true
	}
	if m := reNumMark.FindStringSubmatch(draft); m != nil {
		n, _ := strconv.Atoi(m[1])
		return doc.ListOrdered, false, m[0], n, true
	}
	if m := reBullet.FindString(draft); m != "" {
		return doc.ListBullet, false, m, 0, true
	}
	if m := reQuoteMark.FindString(draft); m != "" {
		return 0, true, "> ", 0, true
	}
	return 0, false, "", 0, false
}

// nextMark is the marker the item after this one opens with: the same
// bullet, the number counted up, an empty box, or the quote's mark.
func nextMark(kind doc.ListKind, quote bool, mark string, num int) string {
	switch {
	case quote:
		return "> "
	case kind == doc.ListOrdered:
		return fmt.Sprintf("%d. ", num+1)
	case kind == doc.ListTask:
		return "- [ ] "
	}
	return mark
}
