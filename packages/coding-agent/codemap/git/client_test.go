package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// execCmd runs a command and fails the test on non-zero exit.
func execCmd(t *testing.T, name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		t.Fatalf("%s %v: %v", name, args, err)
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func TestClientIsRepo(t *testing.T) {
	// Real git repo should return true.
	dir := t.TempDir()
	execCmd(t, "git", "init", dir)

	client := NewClient(dir)
	if !client.IsRepo() {
		t.Error("IsRepo() = false, want true for git-initialized dir")
	}

	// Non-git directory should return false.
	nonGit := t.TempDir()
	if NewClient(nonGit).IsRepo() {
		t.Error("IsRepo() = true, want false for non-git dir")
	}
}

func TestFileHistoryReturnsCommits(t *testing.T) {
	dir := t.TempDir()
	execCmd(t, "git", "init", dir)
	execCmd(t, "git", "-C", dir, "config", "user.email", "test@example.com")
	execCmd(t, "git", "-C", dir, "config", "user.name", "Test")

	// Write a file and commit twice.
	writeFile(t, dir, "foo.go", "package foo\n")
	execCmd(t, "git", "-C", dir, "add", "foo.go")
	execCmd(t, "git", "-C", dir, "commit", "-m", "first commit")

	writeFile(t, dir, "foo.go", "package foo\n\nfunc F() {}\n")
	execCmd(t, "git", "-C", dir, "add", "foo.go")
	execCmd(t, "git", "-C", dir, "commit", "-m", "second commit")

	client := NewClient(dir)
	commits, err := client.FileHistory("foo.go", 10)
	if err != nil {
		t.Fatalf("FileHistory: %v", err)
	}
	if len(commits) != 2 {
		t.Fatalf("expected 2 commits for foo.go, got %d", len(commits))
	}

	// Newest first.
	if commits[0].Hash == "" {
		t.Error("newest commit hash is empty")
	}
	if commits[0].Author != "Test" {
		t.Errorf("author = %q, want %q", commits[0].Author, "Test")
	}
	if commits[0].Message != "second commit" {
		t.Errorf("newest message = %q, want %q", commits[0].Message, "second commit")
	}
}

func TestFileHistoryMaxCount(t *testing.T) {
	dir := t.TempDir()
	execCmd(t, "git", "init", dir)
	execCmd(t, "git", "-C", dir, "config", "user.email", "test@example.com")
	execCmd(t, "git", "-C", dir, "config", "user.name", "Test")

	msgs := []string{"a", "b", "c", "d", "e"}
	for _, msg := range msgs {
		writeFile(t, dir, "x.go", "package x\n// "+msg+"\n")
		execCmd(t, "git", "-C", dir, "add", "x.go")
		execCmd(t, "git", "-C", dir, "commit", "-m", msg)
	}

	client := NewClient(dir)
	commits, err := client.FileHistory("x.go", 2)
	if err != nil {
		t.Fatalf("FileHistory: %v", err)
	}
	if len(commits) != 2 {
		t.Errorf("max-count cap: got %d commits, want 2", len(commits))
	}
}

func TestFileHistoryFollowsRenames(t *testing.T) {
	dir := t.TempDir()
	execCmd(t, "git", "init", dir)
	execCmd(t, "git", "-C", dir, "config", "user.email", "test@example.com")
	execCmd(t, "git", "-C", dir, "config", "user.name", "Test")

	writeFile(t, dir, "old.go", "package foo\n")
	execCmd(t, "git", "-C", dir, "add", "old.go")
	execCmd(t, "git", "-C", dir, "commit", "-m", "original")

	execCmd(t, "git", "-C", dir, "mv", "old.go", "new.go")
	execCmd(t, "git", "-C", dir, "add", "-A")
	execCmd(t, "git", "-C", dir, "commit", "-m", "renamed")

	client := NewClient(dir)
	commits, err := client.FileHistory("new.go", 10)
	if err != nil {
		t.Fatalf("FileHistory: %v", err)
	}
	if len(commits) != 2 {
		t.Errorf("--follow rename: got %d commits, want 2 (original + rename)", len(commits))
	}
}

func TestFileHistoryUntracked(t *testing.T) {
	dir := t.TempDir()
	execCmd(t, "git", "init", dir)

	client := NewClient(dir)
	commits, err := client.FileHistory("nonexistent.go", 10)
	if err != nil {
		t.Fatalf("FileHistory for untracked: %v", err)
	}
	// Untracked file with no history: zero commits is acceptable.
	if len(commits) != 0 {
		t.Errorf("untracked file: got %d commits, want 0", len(commits))
	}
}

func TestFileDiffHunksBasic(t *testing.T) {
	dir := t.TempDir()
	execCmd(t, "git", "init", dir)
	execCmd(t, "git", "-C", dir, "config", "user.email", "test@example.com")
	execCmd(t, "git", "-C", dir, "config", "user.name", "Test")

	// Create file and commit.
	writeFile(t, dir, "lines.go", "package lines\n\nfunc A() {}\nfunc B() {}\nfunc C() {}\n")
	execCmd(t, "git", "-C", dir, "add", "lines.go")
	execCmd(t, "git", "-C", dir, "commit", "-m", "base")

	// Change lines.
	writeFile(t, dir, "lines.go", "package lines\n\nfunc X() {}\nfunc Y() {}\nfunc Z() {}\n")
	execCmd(t, "git", "-C", dir, "add", "lines.go")
	execCmd(t, "git", "-C", dir, "commit", "-m", "changed")

	// Get hash of "changed" commit (newest).
	commits, err := NewClient(dir).FileHistory("lines.go", 1)
	if err != nil {
		t.Fatalf("FileHistory: %v", err)
	}
	if len(commits) == 0 {
		t.Fatal("expected at least 1 commit")
	}

	hunks, err := NewClient(dir).FileDiffHunks(commits[0].Hash, "lines.go")
	if err != nil {
		t.Fatalf("FileDiffHunks: %v", err)
	}
	if len(hunks) == 0 {
		t.Error("commit changed lines.go but hunks is empty")
	}
	// EndLine must be > StartLine for every hunk.
	for i, h := range hunks {
		if h.EndLine <= h.StartLine {
			t.Errorf("hunk %d: EndLine(%d) <= StartLine(%d)", i, h.EndLine, h.StartLine)
		}
	}
}

func TestFileDiffHunksMultiHunk(t *testing.T) {
	dir := t.TempDir()
	execCmd(t, "git", "init", dir)
	execCmd(t, "git", "-C", dir, "config", "user.email", "test@example.com")
	execCmd(t, "git", "-C", dir, "config", "user.name", "Test")

	writeFile(t, dir, "m.go", "package m\n\nfunc F1() {}\n\nfunc F2() {}\n\nfunc F3() {}\n")
	execCmd(t, "git", "-C", dir, "add", "m.go")
	execCmd(t, "git", "-C", dir, "commit", "-m", "base")

	// Second commit — must differ from base to produce a real hunk.
	writeFile(t, dir, "m.go", "package m\n\nfunc F1mod() {}\n\nfunc F2() {}\n\nfunc F3() {}\n")
	execCmd(t, "git", "-C", dir, "add", "m.go")
	execCmd(t, "git", "-C", dir, "commit", "-m", "changed")

	commits, err := NewClient(dir).FileHistory("m.go", 1)
	if err != nil {
		t.Fatalf("FileHistory: %v", err)
	}
	if len(commits) == 0 {
		t.Fatal("expected at least 1 commit")
	}

	hunks, err := NewClient(dir).FileDiffHunks(commits[0].Hash, "m.go")
	if err != nil {
		t.Fatalf("FileDiffHunks: %v", err)
	}
	if len(hunks) == 0 {
		t.Error("FileDiffHunks returned no hunks for a committed file")
	}
	for i, h := range hunks {
		if h.EndLine <= h.StartLine {
			t.Errorf("hunk %d: EndLine(%d) <= StartLine(%d)", i, h.EndLine, h.StartLine)
		}
	}
}

func TestFileDiffHunksCreationCommitHasHunks(t *testing.T) {
	dir := t.TempDir()
	execCmd(t, "git", "init", dir)
	execCmd(t, "git", "-C", dir, "config", "user.email", "test@example.com")
	execCmd(t, "git", "-C", dir, "config", "user.name", "Test")

	// Commit a file.
	writeFile(t, dir, "a.go", "package a\n")
	execCmd(t, "git", "-C", dir, "add", "a.go")
	execCmd(t, "git", "-C", dir, "commit", "-m", "a")

	commits, err := NewClient(dir).FileHistory("a.go", 1)
	if err != nil {
		t.Fatalf("FileHistory: %v", err)
	}
	if len(commits) == 0 {
		t.Fatal("expected at least 1 commit")
	}

	// Ask for hunks on the commit that created the file.
	hunks, err := NewClient(dir).FileDiffHunks(commits[0].Hash, "a.go")
	if err != nil {
		t.Fatalf("FileDiffHunks for creation commit: %v", err)
	}
	if len(hunks) == 0 {
		t.Error("expected at least one hunk for creation commit")
	}
}

func TestFileDiffHunksInvalidHash(t *testing.T) {
	client := NewClient(t.TempDir())
	_, err := client.FileDiffHunks("not-a-sha-at-all", "foo.go")
	if err == nil {
		t.Error("expected error for invalid hash, got nil")
	}
}
