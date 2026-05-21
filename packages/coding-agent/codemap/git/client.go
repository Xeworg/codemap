// Package git provides a thin client for reading commit history and diff hunks
// from a git repository. It is used by the codemap indexer to wire real git
// commits to symbol history links instead of synthetic placeholders.
package git

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"codrut/packages/coding-agent/codemap/indexer"
)

// CommitInfo holds metadata for a single commit touching a file.
type CommitInfo struct {
	Hash    string
	Author  string
	Date    string // ISO-8601 strict, e.g. "2006-01-02T15:04:05Z07:00"
	Message string
}

// Client wraps a git repository root path and provides history queries.
type Client struct {
	repoRoot string
}

// NewClient returns a client that queries commits under repoRoot.
// The returned client does not check whether repoRoot is a valid git repo;
// use IsRepo() to verify before querying.
func NewClient(repoRoot string) *Client {
	return &Client{repoRoot: repoRoot}
}

// IsRepo reports whether repoRoot is a valid git repository root.
// It runs "git rev-parse --git-dir" which succeeds with exit code 0
// inside a repo and fails outside one.
func (c *Client) IsRepo() bool {
	cmd := exec.Command("git", "-C", c.repoRoot, "rev-parse", "--git-dir")
	return cmd.Run() == nil
}

// FileHistory returns commits that touched the given path, ordered
// reverse-chronologically (newest first). It uses --follow to track renames
// and --first-parent to avoid merge-bubble noise. The maxCommits parameter
// caps the number of commits returned; pass 0 to use git's default.
// Empty slice is returned if the file has no commit history (e.g. untracked).
func (c *Client) FileHistory(path string, maxCommits int) ([]CommitInfo, error) {
	args := []string{"-C", c.repoRoot, "log",
		"--follow", "--first-parent",
		"--format=%H%x00%an%x00%aI%x00%s",
	}
	if maxCommits > 0 {
		args = append(args, "--max-count="+strconv.Itoa(maxCommits))
	}
	args = append(args, "--", path)
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		// git log returns non-zero when there are no commits for the path;
		// treat that as an empty list rather than a hard error.
		if _, ok := err.(*exec.ExitError); ok {
			return nil, nil
		}
		return nil, fmt.Errorf("git log: %w", err)
	}
	return parseCommitLines(out), nil
}

// FileDiffHunks returns the diff hunks for a given commit and path.
// Each hunk covers a contiguous region of added/modified lines in the new
// version of the file. Returns nil if the commit does not touch the path
// (e.g. file existed but was not modified in that commit).
func (c *Client) FileDiffHunks(commitHash, path string) ([]indexer.CommitHunk, error) {
	// Validate that commitHash looks like a SHA (hex, 7–40 chars) to avoid
	// arbitrary command injection. We do not support short names here.
	if !isValidSHA(commitHash) {
		return nil, fmt.Errorf("invalid commit hash: %q", commitHash)
	}
	cmd := exec.Command("git", "-C", c.repoRoot, "show",
		commitHash, "--format=", "--no-color", "-p", "--", path)
	out, err := cmd.Output()
	if err != nil {
		// ExitError means the path was not touched in this commit — normal.
		if _, ok := err.(*exec.ExitError); ok {
			return nil, nil
		}
		return nil, fmt.Errorf("git show: %w", err)
	}
	return parseDiffHunks(string(out)), nil
}

// parseCommitLines splits NUL-delimited lines into CommitInfo records.
func parseCommitLines(output []byte) []CommitInfo {
	var commits []CommitInfo
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\x00", 4)
		if len(parts) < 4 {
			continue
		}
		commits = append(commits, CommitInfo{
			Hash:    parts[0],
			Author:  parts[1],
			Date:    parts[2],
			Message: parts[3],
		})
	}
	if scanner.Err() != nil {
		return commits
	}
	return commits
}

// hunkHeaderRE matches unified-diff @@ headers. The captured groups are:
//   - oldStart, oldCount, newStart, newCount
//
// Format: @@ -<oldStart>[,<oldCount>] +<newStart>[,<newCount>] @@
var hunkHeaderRE = regexp.MustCompile(`@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

// parseDiffHunks extracts line ranges from unified diff output.
// It uses the "+" side (new file) start/count to build CommitHunk ranges.
func parseDiffHunks(output string) []indexer.CommitHunk {
	var hunks []indexer.CommitHunk
	scanner := bufio.NewScanner(strings.NewReader(output))
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		matches := hunkHeaderRE.FindStringSubmatch(line)
		if len(matches) < 4 {
			continue
		}
		newStart, _ := strconv.Atoi(matches[3])
		newCount := 1
		if matches[4] != "" {
			newCount, _ = strconv.Atoi(matches[4])
		}
		hunks = append(hunks, indexer.CommitHunk{
			StartLine: newStart,
			EndLine:   newStart + newCount,
		})
	}
	if scanner.Err() != nil {
		return hunks
	}
	return hunks
}

// isValidSHA checks that s is a hex string of length 7–40 (full SHA or short).
func isValidSHA(s string) bool {
	if len(s) < 7 || len(s) > 40 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}
