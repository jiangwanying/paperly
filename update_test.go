package main

import "testing"

// newerThan compares the build's version with the newest release's: the
// three numbers count up, unreadable ones never count as newer.
func TestNewerThan(t *testing.T) {
	same := []string{"0.1.0", "v0.1.0", "0.1"}
	for _, cur := range same {
		if newerThan(cur, cur) {
			t.Errorf("v%s counts as newer than itself", cur)
		}
	}
	for _, c := range []struct{ cur, latest string }{
		{"0.1.0", "0.1.1"}, {"0.1.0", "0.2.0"}, {"0.1.0", "1.0.0"},
		{"0.1.0", "v1.0.0"}, {"0.9.9", "1.0.0"},
	} {
		if !newerThan(c.cur, c.latest) {
			t.Errorf("v%s missed that v%s is newer", c.cur, c.latest)
		}
	}
	// Lower or unreadable is not newer.
	for _, c := range []struct{ cur, latest string }{
		{"0.2.0", "0.1.0"}, {"1.0.0", "0.9.9"}, {"0.1.0", "abc"},
		{"0.1.0", "v"}, {"0.1.0", ""}, {"0.1.0", "0.1.0-beta"},
	} {
		if newerThan(c.cur, c.latest) {
			t.Errorf("v%s took v%s as newer", c.cur, c.latest)
		}
	}
}

// parseSemver reads the shapes a release tag arrives in.
func TestParseSemver(t *testing.T) {
	if v, ok := parseSemver("v0.1.1"); !ok || v != [3]int{0, 1, 1} {
		t.Errorf("v0.1.1 is %v %v", v, ok)
	}
	if v, ok := parseSemver("1.2"); !ok || v != [3]int{1, 2, 0} {
		t.Errorf("1.2 is %v %v", v, ok)
	}
	for _, bad := range []string{"", "v", "abc", "1.2.x", "-1.0.0", "1.2.3.4"} {
		if _, ok := parseSemver(bad); ok {
			t.Errorf("%q parsed as a version", bad)
		}
	}
}
