package client

import (
	"fmt"
	"path/filepath"
	"testing"
)

// TestClient runs a suite of tests on c. It should be called from
// a test function (see golang.org/pkg/testing/). c should be an
// authenticated client so that all methods behave as normal.
func TestClient(t *testing.T, c Client) {
	testDir(t, c)
	testUpload(t, c)
	testRemove(t, c)
	testPath(t, c)
}

// test mkdir, list
func testDir(t *testing.T, c Client) {
	err := c.Mkdir("/foo")
	if err != nil {
		t.Fatalf("testDir: Mkdir(%q): %v", "/foo", err)
	}

	ents, err := c.List("/")
	if err != nil {
		t.Fatalf("testDir: List(%q): %v", "/", err)
	}
	if len(ents) != 1 || !ents[0].IsDir() || ents[0].Name() != "foo" {
		t.Fatalf("testDir: unexpected entries when listing /: got %v; want [d foo]", dirEntStrings(ents))
	}

	ents, err = c.List("/foo")
	if err != nil {
		t.Fatalf("testDir: List(%q): %v", "/foo", err)
	}
	if len(ents) != 0 {
		t.Fatalf("testDir: unexpected entries when listing /foo: got %v; want []", dirEntStrings(ents))
	}

	err = c.Mkdir("/foo/bar")
	if err != nil {
		t.Fatalf("testDir: Mkdir(%q): %v", "/foo/bar", err)
	}

	ents, err = c.List("/foo")
	if err != nil {
		t.Fatalf("testDir: List(%q): %v", "/foo", err)
	}
	if len(ents) != 1 || !ents[0].IsDir() || ents[0].Name() != "bar" {
		t.Fatalf("testDir: unexpected entries when listing /foo: got %v; want [d bar]", dirEntStrings(ents))
	}

	// clean up
	removeAll(t, c)
}

// test upload, download
func testUpload(t *testing.T, c Client) {
	err := c.Upload("/foo", []byte("foobar"))
	if err != nil {
		t.Fatalf("testUpload: Upload(%q, ...): %v", "/foo", err)
	}

	body, err := c.Download("/foo")
	if err != nil {
		t.Fatalf("testUpload: Download(%q): %v", "/foo", err)
	}
	if string(body) != "foobar" {
		t.Fatalf("testUpload: unexpected file contents; got %q; want %q", string(body), "foobar")
	}

	// the second upload must be smaller than the first
	// so that we test that overwritten files are truncated
	err = c.Upload("/foo", []byte("bar"))
	if err != nil {
		t.Fatalf("testUpload: Upload(%q, ...): %v", "/foo", err)
	}
	body, err = c.Download("/foo")
	if err != nil {
		t.Fatalf("testUpload: Download(%q): %v", "/foo", err)
	}
	if string(body) != "bar" {
		t.Fatalf("testUpload: unexpected file contents; got %q; want %q", string(body), "bar")
	}

	// clean up
	removeAll(t, c)
}

// test remove
func testRemove(t *testing.T, c Client) {
	// test removing directories
	err := c.Mkdir("/foo")
	if err != nil {
		t.Fatalf("testRemove: Mkdir(%q): %v", "/foo", err)
	}

	err = c.Remove("/foo")
	if err != nil {
		t.Fatalf("testRemove: Remove(%q): %v", "/foo", err)
	}

	ents, err := c.List("/")
	if err != nil {
		t.Fatalf("testRemove: List(%q): %v", "/", err)
	}
	if len(ents) != 0 {
		t.Fatalf("testRemove: unexpected entries when listing /: got %v; want []", dirEntStrings(ents))
	}

	// test removing files
	err = c.Upload("/foo", nil)
	if err != nil {
		t.Fatalf("testRemove: Upload(%q, ...): %v", "/foo", err)
	}

	err = c.Remove("/foo")
	if err != nil {
		t.Fatalf("testRemove: Remove(%q): %v", "/foo", err)
	}

	ents, err = c.List("/")
	if err != nil {
		t.Fatalf("testRemove: List(%q): %v", "/", err)
	}
	if len(ents) != 0 {
		t.Fatalf("testRemove: unexpected entries when listing /: got %v; want []", dirEntStrings(ents))
	}
}

// test PWD, CD, and other path-related functionality
func testPath(t *testing.T, c Client) {
	err := c.Mkdir("/foo")
	if err != nil {
		t.Fatalf("testPath: Mkdir(%q): %v", "/foo", err)
	}

	// cd to /foo
	err = c.CD("/foo")
	if err != nil {
		t.Fatalf("testPath: CD(%q): %v", "/foo", err)
	}

	pwd, err := c.PWD()
	if err != nil {
		t.Fatalf("testPath: PWD(): %v", err)
	}
	if pwd != "/foo" && pwd != "/foo/" {
		t.Fatalf("testPath: unexpected pwd: got %v; want /foo or /foo/", pwd)
	}

	// cd back to /
	err = c.CD("..")
	if err != nil {
		t.Fatalf("testPath: CD(%q): %v", "..", err)
	}

	pwd, err = c.PWD()
	if err != nil {
		t.Fatalf("testPath: PWD(): %v", err)
	}
	if pwd != "/" {
		t.Fatalf("testPath: unexpected pwd: got %v; want /", pwd)
	}

	// cd back to /foo for more testing
	err = c.CD("/foo")
	if err != nil {
		t.Fatalf("testPath: CD(%q): %v", "/foo", err)
	}

	pwd, err = c.PWD()
	if err != nil {
		t.Fatalf("testPath: PWD(): %v", err)
	}
	if pwd != "/foo" && pwd != "/foo/" {
		t.Fatalf("testPath: unexpected pwd: got %v; want /foo or /foo/", pwd)
	}

	// note that this is a relative path
	err = c.Mkdir("bar")
	if err != nil {
		t.Fatalf("testPath: Mkdir(%q): %v", "bar", err)
	}

	// list using relative path
	ents, err := c.List(".")
	if err != nil {
		t.Fatalf("testPath: List(%q): %v", ".", err)
	}
	if len(ents) != 1 || !ents[0].IsDir() || ents[0].Name() != "bar" {
		t.Fatalf("testPath: unexpected entries when listing /foo: got %v; want [d bar]", dirEntStrings(ents))
	}

	// list using absolute path
	ents, err = c.List("/foo")
	if err != nil {
		t.Fatalf("testPath: List(%q): %v", ".", err)
	}
	if len(ents) != 1 || !ents[0].IsDir() || ents[0].Name() != "bar" {
		t.Fatalf("testPath: unexpected entries when listing /foo: got %v; want [d bar]", dirEntStrings(ents))
	}

	// upload using relative path, but with ./ syntax
	err = c.Upload("./bar/baz", []byte("baz"))
	if err != nil {
		t.Fatalf("testPath: Upload(%q, ...): %v", "./bar/baz", err)
	}

	body, err := c.Download("./bar/baz")
	if err != nil {
		t.Fatalf("testPath: Download(%q): %v", "./bar/baz", err)
	}
	if string(body) != "baz" {
		t.Fatalf("testPath: unexpected file contents for ./bar/baz: got %q; want %q", string(body), "baz")
	}

	// make sure that /.. is /
	ents, err = c.List("../../../../../../../../")
	if err != nil {
		t.Fatalf("testPath: List(%q): %v", "../../../../../../../../", err)
	}
	if len(ents) != 1 || !ents[0].IsDir() || ents[0].Name() != "foo" {
		t.Fatalf("testPath: unexpected entries when listing ../../../../../../../../: got %v; want [d foo]", dirEntStrings(ents))
	}

	// clean up
	removeAll(t, c)
}

func removeAll(t *testing.T, c Client) {
	removeAllHelper(t, c, "/")
}

func removeAllHelper(t *testing.T, c Client, dir string) {
	ents, err := c.List(dir)
	if err != nil {
		t.Fatalf("removeAll: List(%q): %v", dir, err)
	}
	for _, ent := range ents {
		if ent.IsDir() {
			removeAllHelper(t, c, filepath.Join(dir, ent.Name()))
		}
		err = c.Remove(filepath.Join(dir, ent.Name()))
		if err != nil {
			t.Fatalf("removeAll: Remove(%q): %v", filepath.Join(dir, ent.Name()), err)
		}

	}
}

type dirEntStringer struct {
	d DirEnt
}

func (d dirEntStringer) String() string { return DirEntString(d.d) }

func dirEntStrings(dirents []DirEnt) string {
	var stringers []dirEntStringer
	for _, d := range dirents {
		stringers = append(stringers, dirEntStringer{d})
	}
	return fmt.Sprint(stringers)
}
