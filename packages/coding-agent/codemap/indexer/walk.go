package indexer

import (
	"os"
	"path/filepath"
	"strings"
)

// Default exclusions for repositories.
var defaultExclusions = []string{
	".git/",
	"node_modules/",
	"vendor/",
	"dist/",
	"build/",
	".codemap/",
}

// Indexer handles file discovery, filtering, and diff detection.
type Indexer struct {
	repoRoot    string
	customRules []string
	loadedRules []string
}

// NewIndexer creates an Indexer for the given repo root.
// It optionally loads .codemapignore from the root directory.
func NewIndexer(repoRoot string) *Indexer {
	i := &Indexer{repoRoot: repoRoot}
	_ = i.loadIgnoreFile() // ignore errors; defaults apply even if file missing
	return i
}

// loadIgnoreFile reads .codemapignore and parses rules into loadedRules.
func (i *Indexer) loadIgnoreFile() error {
	path := filepath.Join(i.repoRoot, ".codemapignore")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimRight(line, "\r")
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		i.loadedRules = append(i.loadedRules, line)
		i.customRules = append(i.customRules, line)
	}
	return nil
}

// shouldExclude returns true if the path matches any exclusion rule.
// Rules are evaluated in order; later negation rules (!pattern) override
// earlier positive matches, matching gitignore semantics.
func (i *Indexer) shouldExclude(path string) bool {
	// Default exclusions apply first and cannot be un-excluded.
	for _, pattern := range defaultExclusions {
		if strings.HasPrefix(path, pattern) || strings.Contains(path, "/"+pattern) {
			return true
		}
	}

	// Merge custom + loaded rules, evaluate in order.
	allRules := append([]string{}, i.customRules...)
	allRules = append(allRules, i.loadedRules...)

	excluded := false
	for _, rule := range allRules {
		if matchRule(rule, path) {
			if strings.HasPrefix(rule, "!") {
				excluded = false // negation overrides earlier matches
			} else {
				excluded = true
			}
		}
	}
	return excluded
}

// matchRule performs simple glob matching for a rule against a path.
// Supports: *.ext, dir/**, literal patterns. Handles negation prefix ! internally.
func matchRule(rule, path string) bool {
	// Strip negation prefix — caller may check ! separately, but we need
	// the bare pattern for matching.
	negated := strings.HasPrefix(rule, "!")
	if negated {
		rule = rule[1:]
	}

	matched := false

	// Glob: *.ext
	if strings.HasPrefix(rule, "*.") {
		ext := rule[1:] // keep the dot: ".go"
		matched = strings.HasSuffix(path, ext) ||
			strings.HasSuffix(filepath.Base(path), ext)
	} else if strings.HasSuffix(rule, "/**") {
		// Glob: dir/**
		prefix := strings.TrimSuffix(rule, "/**")
		matched = strings.HasPrefix(path, prefix+"/") || path == prefix
	} else if strings.HasPrefix(rule, "**/") {
		// Glob: **/foo — suffix match
		suffix := strings.TrimPrefix(rule, "**/")
		matched = strings.HasSuffix(path, suffix)
	} else {
		// Literal or directory prefix
		matched = path == rule || strings.HasPrefix(path, rule+"/") || strings.HasSuffix(path, "/"+rule)
	}

	return matched
}

// isParseCandidate returns true only for .go files.
func (i *Indexer) isParseCandidate(path string) bool {
	return strings.HasSuffix(path, ".go")
}

// DiscoverFiles walks the repo tree and returns all parse-candidate files.
// It excludes paths matching default and custom rules.
func (i *Indexer) DiscoverFiles() ([]FileEntry, error) {
	var candidates []FileEntry

	err := filepath.Walk(i.repoRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			rel, _ := filepath.Rel(i.repoRoot, path)
			// Exclude the repo root itself
			if rel == "." {
				return nil
			}
			// Check if directory should be excluded
			if i.shouldExclude(rel) {
				return filepath.SkipDir
			}
			return nil
		}

		rel, err := filepath.Rel(i.repoRoot, path)
		if err != nil {
			return err
		}
		if i.shouldExclude(rel) {
			return nil
		}
		if !i.isParseCandidate(rel) {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		hash := HashContent(data)
		candidates = append(candidates, FileEntry{Path: rel, Hash: hash})
		return nil
	})

	if err != nil {
		return nil, err
	}
	return candidates, nil
}
