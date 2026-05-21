package indexer

import (
	"testing"
)

// TestFileStateConstants ensures enum consistency.
func TestFileStateConstants(t *testing.T) {
	if FileStateUnchanged != 0 {
		t.Errorf("FileStateUnchanged should be 0, got %d", FileStateUnchanged)
	}
	if FileStateChanged != 1 {
		t.Errorf("FileStateChanged should be 1, got %d", FileStateChanged)
	}
	if FileStateNew != 2 {
		t.Errorf("FileStateNew should be 2, got %d", FileStateNew)
	}
	if FileStateDeleted != 3 {
		t.Errorf("FileStateDeleted should be 3, got %d", FileStateDeleted)
	}
}

// TestHashEquality verifies identical content produces identical hashes.
func TestHashEquality(t *testing.T) {
	content := []byte("package main\n\nfunc main() {}\n")
	h1 := HashContent(content)
	h2 := HashContent(content)
	if h1 != h2 {
		t.Errorf("identical content produced different hashes: %q != %q", h1, h2)
	}
}

// TestHashDifference verifies different content produces different hashes.
func TestHashDifference(t *testing.T) {
	c1 := []byte("package main\n\nfunc main() {}\n")
	c2 := []byte("package main\n\nfunc main() { print(1) }\n")
	h1 := HashContent(c1)
	h2 := HashContent(c2)
	if h1 == h2 {
		t.Errorf("different content produced identical hashes: %q == %q", h1, h2)
	}
}

// TestHashEmptyContent verifies empty content has a known hash.
func TestHashEmptyContent(t *testing.T) {
	empty := []byte("")
	h := HashContent(empty)
	if h == "" {
		t.Error("empty content hash should not be empty")
	}
}

// TestDiffClassifyUnchanged verifies unchanged files are classified correctly.
func TestDiffClassifyUnchanged(t *testing.T) {
	path := "pkg/foo.go"
	prevHash := "abc123"
	currHash := "abc123"

	state := ClassifyDiff(path, prevHash, currHash)
	if state != FileStateUnchanged {
		t.Errorf("same hash should be unchanged; got %v", state)
	}
}

// TestDiffClassifyChanged verifies changed files (different hashes) are classified.
func TestDiffClassifyChanged(t *testing.T) {
	path := "pkg/foo.go"
	prevHash := "abc123"
	currHash := "def456"

	state := ClassifyDiff(path, prevHash, currHash)
	if state != FileStateChanged {
		t.Errorf("different hash should be changed; got %v", state)
	}
}

// TestDiffClassifyNew verifies new files (no previous hash) are classified.
func TestDiffClassifyNew(t *testing.T) {
	path := "pkg/newfile.go"
	prevHash := "" // no previous snapshot
	currHash := "def456"

	state := ClassifyDiff(path, prevHash, currHash)
	if state != FileStateNew {
		t.Errorf("no previous hash should be new; got %v", state)
	}
}

// TestDiffClassifyDeleted verifies deleted files (no current hash) are classified.
func TestDiffClassifyDeleted(t *testing.T) {
	path := "pkg/deleted.go"
	prevHash := "abc123"
	currHash := "" // file no longer present

	state := ClassifyDiff(path, prevHash, currHash)
	if state != FileStateDeleted {
		t.Errorf("no current hash should be deleted; got %v", state)
	}
}

// TestDiffClassifyNewWithEmptyPrevHash verifies explicit empty string.
func TestDiffClassifyNewWithEmptyPrevHash(t *testing.T) {
	path := "cmd/main.go"
	state := ClassifyDiff(path, "", "xyz789")
	if state != FileStateNew {
		t.Errorf("empty prevHash should be new; got %v", state)
	}
}

// TestDiffClassifyDeletedWithEmptyCurrHash verifies explicit empty string.
func TestDiffClassifyDeletedWithEmptyCurrHash(t *testing.T) {
	path := "cmd/old.go"
	state := ClassifyDiff(path, "abc", "")
	if state != FileStateDeleted {
		t.Errorf("empty currHash should be deleted; got %v", state)
	}
}
