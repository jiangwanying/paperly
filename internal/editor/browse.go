package editor

import (
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown/parser"

	"paperly/locales"
)

// The reading view answers the pointer the way a document on the web does:
// a link opens what it names — a web address in the browser, another file
// of the folder in the app, a heading of the same document in view — and a
// picture of its own shows large, over the page, until a click takes it
// away.

// reSoloImage matches a block that is one picture and nothing else.
var reSoloImage = regexp.MustCompile(`^!\[([^\]]*)\]\(([^)\n]+)\)[ \t]*$`)

// openLink follows a link of the reading view. A web address opens in the
// browser, a heading of the document comes into view, and a file of the
// folder opens as a document of its own, as wiki links do.
func (e *Editor) openLink(c *ui.Context, dest string) {
	if strings.HasPrefix(dest, "#") {
		// The anchor of the view scrolls to the heading; the version
		// makes the same heading asked twice scroll again.
		e.md.Anchor = strings.TrimPrefix(dest, "#")
		e.md.AnchorVersion++
		return
	}
	lower := strings.ToLower(dest)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(lower, "mailto:") {
		c.OpenURL(dest)
		return
	}
	if strings.Contains(lower, "://") {
		return // another scheme of the web's, not ours to follow
	}
	// A path: another document of the folder opens here; a picture or
	// another file opens with what the system opens it with.
	p, err := url.PathUnescape(dest)
	if err != nil {
		p = dest
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(e.docDir(), p)
	}
	st, err := os.Stat(p)
	if err != nil {
		return // a link to nothing stays a link
	}
	if st.IsDir() {
		return
	}
	switch strings.ToLower(filepath.Ext(p)) {
	case ".md", ".markdown":
		if e.sh.OpenAt != nil {
			e.sh.OpenAt(e.win, p, 0)
		}
	default:
		c.OpenURL("file://" + p)
	}
}

// soloImage takes the address and the description of a block that is one
// picture and nothing else.
func soloImage(src string) (addr, alt string, ok bool) {
	m := reSoloImage.FindStringSubmatch(src)
	if m == nil {
		return "", "", false
	}
	return m[2], m[1], true
}

// zoomableBlock builds the reading of a block that is one picture: the
// picture itself, which opens large on a click. false leaves the block to
// the view's own reading.
func (e *Editor) zoomableBlock(c *ui.Context, d *parser.Document, i int) bool {
	b := d.Block(i)
	addr, alt, ok := soloImage(strings.TrimSpace(d.Source()[b.Start:b.End]))
	if !ok {
		return false
	}
	row := ui.Column(c).Padding(2, 0).AlignItems(ui.Center)
	row.OnClick(func() { e.zoomAddr = addr })
	row.Children(func() {
		if src := e.zoomSource(addr); src != nil {
			ui.Image(c, src).MaxWidthPercent(100).Radius(4).Label(alt)
			return
		}
		label := alt
		if label == "" {
			label = addr
		}
		ui.Text(c, label).FontSize(e.ui.FontSize).TextColor(c.Theme().TextMuted)
	})
	return true
}

// zoomSource loads a picture once and keeps it, for the view of the block
// and the view over the page share it.
func (e *Editor) zoomSource(addr string) ui.ImageSource {
	if e.zoomCache == nil {
		e.zoomCache = map[string]ui.ImageSource{}
	}
	if src, ok := e.zoomCache[addr]; ok {
		return src
	}
	var src ui.ImageSource
	if e.md.LoadImage != nil {
		src, _ = e.md.LoadImage(addr)
	}
	e.zoomCache[addr] = src
	return src
}

// zoomOverlay is the picture opened large: a veil over the page with the
// picture in the middle, and a click anywhere takes it away.
func (e *Editor) zoomOverlay(c *ui.Context) {
	if e.zoomAddr == "" {
		return
	}
	addr := e.zoomAddr
	veil := ui.Column(c).Absolute().Fill().
		Background(ui.RGBA(0, 0, 0, 0.72)).
		AlignItems(ui.Center).Justify(ui.Center)
	veil.OnClick(func() { e.zoomAddr = "" })
	veil.Children(func() {
		if src := e.zoomSource(addr); src != nil {
			ui.Image(c, src).MaxWidthPercent(88).MaxHeightPercent(82).Radius(6)
			return
		}
		ui.Text(c, locales.FilesMarkdownOnly()).TextColor(ui.RGBA(1, 1, 1, 0.9))
	})
}

// zoomIsOn tells whether the veil over the page shows.
func (e *Editor) zoomIsOn() bool { return e.zoomAddr != "" }
