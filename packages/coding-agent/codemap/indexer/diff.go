package indexer

import "strings"

// ClassifyDiff classifies a file's state relative to a previous snapshot.
//
// Classification rules:
//   - prevHash == "" && currHash != "": New
//   - prevHash != "" && currHash == "": Deleted
//   - prevHash != "" && currHash != "" && equal: Unchanged
//   - prevHash != "" && currHash != "" && different: Changed
func ClassifyDiff(path, prevHash, currHash string) FileState {
	prevEmpty := prevHash == ""
	currEmpty := currHash == ""

	if prevEmpty && !currEmpty {
		return FileStateNew
	}
	if !prevEmpty && currEmpty {
		return FileStateDeleted
	}
	if prevHash == currHash {
		return FileStateUnchanged
	}
	return FileStateChanged
}

// DiffSet holds the four categorised file sets from a diff operation.
type DiffSet struct {
	Changed   []FileEntry
	New       []FileEntry
	Deleted   []FileEntry
	Unchanged []FileEntry
}

// FileEntry pairs a path with its content hash and parsed symbols.
type FileEntry struct {
	Path    string
	Hash    string
	Symbols []Symbol // populated by processParse
}

// ClassifyFiles classifies a map of path->hash against known previous hashes.
// prevSnapshot maps path -> hash from the previous snapshot (may be nil for new files).
func ClassifyFiles(currFiles map[string]string, prevSnapshot map[string]string) DiffSet {
	var ds DiffSet
	for path, currHash := range currFiles {
		prevHash, exists := prevSnapshot[path]
		if !exists || prevHash == "" {
			ds.New = append(ds.New, FileEntry{Path: path, Hash: currHash})
		} else if prevHash == currHash {
			ds.Unchanged = append(ds.Unchanged, FileEntry{Path: path, Hash: currHash})
		} else {
			ds.Changed = append(ds.Changed, FileEntry{Path: path, Hash: currHash})
		}
	}
	// Remaining entries in prevSnapshot not in currFiles are deleted.
	for path, prevHash := range prevSnapshot {
		if _, exists := currFiles[path]; !exists {
			ds.Deleted = append(ds.Deleted, FileEntry{Path: path, Hash: prevHash})
		}
	}
	return ds
}

// NormalizeEmptyHash returns "" for any value that looks like an empty hash
// (all zeros, common placeholder for "no hash").
func NormalizeEmptyHash(h string) string {
	h = strings.TrimSpace(h)
	if h == "" || h == "0000000000000000000000000000000000000000000000000000000000000000" {
		return ""
	}
	return h
}
