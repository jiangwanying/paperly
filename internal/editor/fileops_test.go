package editor

import (
	"os"
	"path/filepath"
	"testing"

	"paperly/internal/doc"
)

// searchFor looks the folder through: a file matches by its name, or by
// the lines that hold the word; a binary file and a huge one stay out.
func TestSearchFor(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "组"), 0o755)
	os.WriteFile(filepath.Join(dir, "指南.md"), []byte("# 标题\n\n这里是苹果的说明\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "组", "笔记.md"), []byte("banana and apple\nnothing here\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "无关.md"), []byte("nothing at all\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "坏.md"), []byte("md\x00binary apple\n"), 0o644)

	ed := New(doc.New("x"), NewShared())
	ed.sh.Folder = dir
	ed.filter = "苹果"
	hits := ed.searchFor("苹果")
	if len(hits) != 1 || hits[0].name != "指南.md" || len(hits[0].lines) != 1 || hits[0].lines[0].no != 3 {
		t.Fatalf("the content hit is %+v", hits)
	}

	ed.filter = "apple"
	hits = ed.searchFor("apple")
	if len(hits) != 1 || hits[0].name != "笔记.md" {
		t.Fatalf("the deep hit is %+v", hits)
	}
	if len(hits[0].lines) != 1 || hits[0].lines[0].no != 1 {
		t.Errorf("the lines are %+v", hits[0].lines)
	}

	// A name that matches brings the file even when no line does.
	ed.filter = "指南"
	hits = ed.searchFor("指南")
	if len(hits) != 1 || hits[0].name != "指南.md" {
		t.Fatalf("the name hit is %+v", hits)
	}
}

// JumpTo reveals the block that holds the line.
func TestJumpTo(t *testing.T) {
	ed := New(doc.New("# 一\n\npara one\npara one more\n\n## 二\n\nsecond body"), NewShared())
	ed.JumpTo(3) // para one
	if ed.scrollTo != 1 {
		t.Errorf("line 3 revealed block %d, want 1", ed.scrollTo)
	}
	ed.JumpTo(8) // second body
	if ed.scrollTo != 3 {
		t.Errorf("line 8 revealed block %d, want 3", ed.scrollTo)
	}
	ed.JumpTo(0)
	if ed.scrollTo != 3 {
		t.Errorf("line 0 changed the view to %d", ed.scrollTo)
	}
}
