package cli

import (
	"context"
	"database/sql"
	"strings"

	"codrut/packages/coding-agent/codemap/store"
)

// ExplainCause type alias for type-safe string values.
type ExplainCause string

// Explain cause constants.
const (
	CauseStaleIndex          ExplainCause = "stale_index"
	CauseParseError          ExplainCause = "parse_error"
	CauseNameMismatch        ExplainCause = "name_mismatch"
	CauseMissingHistoryLinks ExplainCause = "missing_history_links"
)

// CauseRecommendedActions maps each explain cause to its recommended actions.
// Actions are static per cause to maintain determinism.
var CauseRecommendedActions = map[ExplainCause][]string{
	CauseStaleIndex: {
		"Run 'codemap index' to update the snapshot",
		"Verify the repository path is correct",
	},
	CauseParseError: {
		"Run 'codemap index' to re-parse files",
		"Check for syntax errors in Go source files",
		"Review parse_errors table for affected files",
	},
	CauseNameMismatch: {
		"Verify the symbol name is spelled correctly",
		"Check for renamed or moved symbols",
		"Run 'codemap index' if the code was recently changed",
	},
	CauseMissingHistoryLinks: {
		"Run 'codemap index' to rebuild history links",
		"Ensure the repository has commit history",
		"Symbol exists but has no git history associations",
	},
}

// DeriveSymbolNotFoundCause determines the cause for a not-found symbol.
// It checks, in order: stale snapshot → parse errors → name mismatch.
// The first matching condition determines the cause.
func DeriveSymbolNotFoundCause(ctx context.Context, db *sql.DB, symbolArg string, symExists bool) (string, []string) {
	if symExists {
		// Symbol was found but in a different context - this shouldn't happen
		// since we checked sym == nil before calling. Fallback.
		return string(CauseNameMismatch), CauseRecommendedActions[CauseNameMismatch]
	}

	// Check 1: stale snapshot.
	meta, err := store.GetLatestSnapshotMeta(ctx, db)
	if err == nil && meta.SnapshotID > 0 {
		if StaleNow(meta.IndexedAt) {
			return string(CauseStaleIndex), CauseRecommendedActions[CauseStaleIndex]
		}
	}

	// Check 2: parse errors in this snapshot.
	if meta.SnapshotID > 0 {
		parseErrors, err := store.GetParseErrorsForSnapshot(ctx, db, meta.SnapshotID)
		if err == nil && len(parseErrors) > 0 {
			return string(CauseParseError), CauseRecommendedActions[CauseParseError]
		}
	}

	// Check 3: name mismatch (symbol not in index despite fresh snapshot).
	return string(CauseNameMismatch), CauseRecommendedActions[CauseNameMismatch]
}

// DeriveHistoryNotFoundCause determines the cause for a not-found history query.
// It checks: stale → parse errors → name mismatch → missing_history_links.
func DeriveHistoryNotFoundCause(ctx context.Context, db *sql.DB, symbolArg string, symExists, hasHistory bool) (string, []string) {
	if !symExists {
		return DeriveSymbolNotFoundCause(ctx, db, symbolArg, false)
	}
	if hasHistory {
		// Shouldn't happen - symbol exists but was told has no history.
		// Fallback to missing_history_links.
		return string(CauseMissingHistoryLinks), CauseRecommendedActions[CauseMissingHistoryLinks]
	}

	// Check for stale snapshot first.
	meta, err := store.GetLatestSnapshotMeta(ctx, db)
	if err == nil && meta.SnapshotID > 0 {
		if StaleNow(meta.IndexedAt) {
			return string(CauseStaleIndex), CauseRecommendedActions[CauseStaleIndex]
		}
	}

	// Check for parse errors.
	if meta.SnapshotID > 0 {
		parseErrors, err := store.GetParseErrorsForSnapshot(ctx, db, meta.SnapshotID)
		if err == nil && len(parseErrors) > 0 {
			return string(CauseParseError), CauseRecommendedActions[CauseParseError]
		}
	}

	// Symbol exists in index but has no history links.
	return string(CauseMissingHistoryLinks), CauseRecommendedActions[CauseMissingHistoryLinks]
}

// BuildExplainNotFound creates an ExplainNotFound struct for a given cause string.
func BuildExplainNotFound(cause string) ExplainNotFound {
	// Look up recommended actions from known causes.
	for knownCause, actions := range CauseRecommendedActions {
		if string(knownCause) == cause {
			return ExplainNotFound{
				Cause:              cause,
				RecommendedActions: actions,
			}
		}
	}
	// Unknown cause - return name_mismatch as fallback.
	return ExplainNotFound{
		Cause:              string(CauseNameMismatch),
		RecommendedActions: CauseRecommendedActions[CauseNameMismatch],
	}
}

// TrimSymbolArg normalizes a symbol argument for display/comparison.
// Removes whitespace and common prefixes.
func TrimSymbolArg(arg string) string {
	return strings.TrimSpace(arg)
}
