package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"paperly/internal/doc"
	"paperly/internal/editor"
)

// renameFile moves a file on the disk; a name the folder has already says
// so, and nothing moves.
func TestRenameFile(t *testing.T) {
	dir := t.TempDir()
	a := NewApp(DefaultSettings(), "")
	old := filepath.Join(dir, "笔记.md")
	os.WriteFile(old, []byte("内容"), 0o644)

	if err := a.renameFile(old, filepath.Join(dir, "新名.md")); err != nil {
		t.Fatalf("rename: %v", err)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Errorf("the old file is still there")
	}
	if b, err := os.ReadFile(filepath.Join(dir, "新名.md")); err != nil || string(b) != "内容" {
		t.Errorf("the new file is %q %v", b, err)
	}

	if err := a.renameFile(filepath.Join(dir, "新名.md"), filepath.Join(dir, "新名.md")); err == nil {
		t.Errorf("renaming onto itself went through")
	}
	err := a.renameFile(filepath.Join(dir, "新名.md"), filepath.Join(dir, "other.md"))
	os.WriteFile(filepath.Join(dir, "other.md"), []byte("x"), 0o644)
	if err := a.renameFile(filepath.Join(dir, "新名.md"), filepath.Join(dir, "other.md")); err == nil {
		t.Errorf("a rename onto an existing file went through")
	}
	_ = err
}

// deleteFile moves a file to the trash — the app's own trash, which a test
// takes in its hand — and a folder with what is in it.
func TestDeleteFileTrashes(t *testing.T) {
	dir := t.TempDir()
	a := NewApp(DefaultSettings(), "")
	var trashed []string
	basket := t.TempDir()
	a.trash = func(path string) error {
		trashed = append(trashed, path)
		// The wastebasket takes the file away from where it was.
		return os.Rename(path, filepath.Join(basket, fmt.Sprint(len(trashed))+"-"+filepath.Base(path)))
	}

	f := filepath.Join(dir, "a.md")
	os.WriteFile(f, []byte("x"), 0o644)
	if err := a.deleteFile(f); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if len(trashed) != 1 || trashed[0] != f {
		t.Errorf("the trash took %v", trashed)
	}
	if _, err := os.Stat(f); !os.IsNotExist(err) {
		t.Errorf("the file is still on the disk")
	}

	sub := filepath.Join(dir, "组")
	os.MkdirAll(sub, 0o755)
	os.WriteFile(filepath.Join(sub, "b.md"), []byte("y"), 0o644)
	if err := a.deleteFile(sub); err != nil {
		t.Fatalf("delete folder: %v", err)
	}
	if len(trashed) != 2 || trashed[1] != sub {
		t.Errorf("the folder went to the trash as %v", trashed)
	}

	// A failure of the trash explains itself.
	a.trash = func(string) error { return errors.New("no wastebasket") }
	if err := a.deleteFile(f); err == nil {
		t.Errorf("a failed trash went through")
	}
}

// The panel's rename and delete go to the app, and the dialogs a test
// answers in the place of the system's.
func TestPanelFileOps(t *testing.T) {
	dir := t.TempDir()
	sh := editor.NewShared()
	sh.Folder = dir
	ed := editor.New(mustDoc("x"), sh)
	var renamed [2]string
	sh.Rename = func(oldPath, newPath string) error {
		renamed[0], renamed[1] = oldPath, newPath
		// The rename really moves the file, as the app's does.
		return os.Rename(oldPath, newPath)
	}
	var deleted string
	sh.Delete = func(path string) error { deleted = path; return nil }

	// Making a file: the name takes .md, and the file opens.
	f := filepath.Join(dir, "笔记.md")
	ed.SetTreeEdit(editor.TreeEdit{Parent: dir})
	ed.SetTreeName("笔记")
	ed.CommitTreeEdit()
	if _, err := os.Stat(f); err != nil {
		t.Errorf("the file was not made: %v", err)
	}

	// Renaming a file: the app is asked, the extension stays.
	old := filepath.Join(dir, "旧.md")
	os.WriteFile(old, []byte("y"), 0o644)
	ed.SetTreeEdit(editor.TreeEdit{Path: old})
	ed.SetTreeName("新")
	ed.CommitTreeEdit()
	if renamed[0] != old || renamed[1] != filepath.Join(dir, "新.md") {
		t.Errorf("the rename asked for %q -> %q", renamed[0], renamed[1])
	}

	// A name that exists says so, and the app is not asked.
	ed.SetTreeEdit(editor.TreeEdit{Path: old})
	ed.SetTreeName(strings.TrimSuffix(filepath.Base(renamed[1]), ".md"))
	var alert error
	ed.TakeAlert(func(err error) { alert = err })
	ed.CommitTreeEdit()
	if alert == nil {
		t.Errorf("the existing name went through")
	}
	if renamed[1] != filepath.Join(dir, "新.md") {
		t.Errorf("the rename happened after all")
	}

	// Deleting asks, and the answer no leaves the file.
	ed.TestAsk(func(string) bool { return false })
	ed.DeletePath(renamed[1])
	if deleted != "" {
		t.Errorf("the delete happened without the yes")
	}
	ed.TestAsk(func(string) bool { return true })
	ed.DeletePath(renamed[1])
	if deleted != renamed[1] {
		t.Errorf("the delete asked for %q", deleted)
	}
}

// renamedPath tells the new path of a file under a renamed one.
func TestRenamedPath(t *testing.T) {
	if got := renamedPath("/d/笔记.md", "/d/笔记.md", "/d/新.md"); got != "/d/新.md" {
		t.Errorf("the file itself: %q", got)
	}
	if got := renamedPath("/d/组/a.md", "/d/组", "/d/新组"); got != "/d/新组/a.md" {
		t.Errorf("a file of the folder: %q", got)
	}
	if got := renamedPath("/d/其他.md", "/d/组", "/d/新组"); got != "" {
		t.Errorf("an untouched file: %q", got)
	}
	if underPath("/d/组", "/d/组") != true || underPath("/d/组/a.md", "/d/组") != true ||
		underPath("/d/组员.md", "/d/组") != false {
		t.Errorf("underPath is wrong")
	}
}

// mustDoc makes the document the panel tests edit.
func mustDoc(src string) *doc.Doc { return doc.New(src) }
