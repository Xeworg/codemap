/**
 * CodeMap Pi Extension — registers codemap tools with Pi's ExtensionAPI.
 *
 * This file replaces the legacy integrations/pi/tools/codemap-tool.json artifact.
 * It registers all CLI commands: index, symbol, history, impact, deadcode,
 * query, install, and doctor.
 *
 * Pi v0.35+ loads this from ~/.pi/agent/extensions/codemap-extension.ts
 * (installed by `codemap install`).
 */
// @ts-expect-error — ExtensionAPI resolved by Pi's extension runner via jiti at runtime; not available in local workspace.
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

export default function (pi: ExtensionAPI) {
	pi.registerTool({
		name: "codemap_index",
		description:
			"Scan and index all Go source files in a repository. Creates or updates the symbol DB. Safe to re-run (incremental).",
		arguments: [],
		flags: [
			{
				name: "--db",
				type: "string",
				description:
					"Path to SQLite DB (optional; default: ~/.cache/codemap/<repo-hash>.db)",
				required: false,
			},
			{
				name: "-repo",
				type: "string",
				description:
					"Repository root to index (optional; default: current directory)",
				required: false,
			},
		],
		output: {
			schema: "1.0",
			fields: {
				ok: "bool — true if run completed",
				"data.files_scanned": "int — total .go files discovered",
				"data.files_parsed": "int — files successfully parsed",
				"data.symbols_found": "int — symbols extracted",
				"data.parse_errors": "int — files that failed to parse (fail-soft)",
				"meta.snapshot_id": "int — ID of this snapshot",
				"meta.is_stale": "bool — true if older than 24h",
			},
		},
		exit_codes: {
			0: "Success",
			1: "Runtime error (DB/file)",
			2: "Validation error (missing repo)",
			3: "Data state error",
		},
	});

	pi.registerTool({
		name: "codemap_symbol",
		description:
			"Look up a Go symbol by name. Returns definition, file, line range, signature, and confidence.",
		arguments: [
			{
				name: "name",
				type: "string",
				description: "Symbol name to look up",
				required: true,
			},
		],
		flags: [
			{
				name: "--db",
				type: "string",
				description:
					"Path to SQLite DB (optional; default: ~/.cache/codemap/<repo-hash>.db)",
				required: false,
			},
		],
		output: {
			schema: "1.0",
			fields: {
				ok: "bool — true if symbol found",
				"data.name": "string — symbol name",
				"data.kind": "string — func|type|interface|var|const",
				"data.signature": "string — signature (params only in MVP)",
				"data.start_line": "int — line where symbol starts",
				"data.end_line": "int — line where symbol ends",
				"data.file": "string — relative file path",
				"data.confidence": "string — high|medium|low",
				"data.evidence": "array — evidence entries",
				"meta.snapshot_id": "int",
				"meta.is_stale": "bool",
			},
		},
		exit_codes: {
			0: "Success (symbol found)",
			1: "Runtime error (DB/file)",
			2: "Validation error (missing name)",
			3: "Not found (no index or symbol absent)",
		},
	});

	pi.registerTool({
		name: "codemap_history",
		description:
			"Query commit history for a Go symbol. Returns commits ordered by link strength (strong > medium > weak) and recency.",
		arguments: [
			{
				name: "name",
				type: "string",
				description: "Symbol name to query history for",
				required: true,
			},
		],
		flags: [
			{
				name: "--db",
				type: "string",
				description:
					"Path to SQLite DB (optional; default: ~/.cache/codemap/<repo-hash>.db)",
				required: false,
			},
		],
		output: {
			schema: "1.0",
			fields: {
				ok: "bool — true if query ran",
				"data.symbol_name": "string",
				"data.confidence": "string — medium if history found, low otherwise",
				"data.evidence":
					"array — commit_link entries with commit_hash, description, link_strength",
				"meta.snapshot_id": "int",
				"meta.is_stale": "bool",
			},
		},
		exit_codes: {
			0: "Success",
			1: "Runtime error",
			2: "Validation error",
			3: "Not found (no index or symbol absent)",
		},
	});

	pi.registerTool({
		name: "codemap_impact",
		description:
			"Show symbols impacted by (depend on or relate to) a given symbol, with risk tier and confidence evidence.",
		arguments: [
			{
				name: "symbol",
				type: "string",
				description: "Symbol name to query impact for",
				required: true,
			},
		],
		flags: [
			{
				name: "--db",
				type: "string",
				description:
					"Path to SQLite DB (optional; default: ~/.cache/codemap/<repo-hash>.db)",
				required: false,
			},
		],
		output: {
			schema: "1.0",
			fields: {
				ok: "bool — true if query ran",
				"data.target_symbol": "string — the queried symbol",
				"data.findings": "array — impacted symbols with risk_tier and confidence",
				"meta.snapshot_id": "int",
				"meta.is_stale": "bool",
			},
		},
		exit_codes: {
			0: "Success",
			1: "Runtime error",
			2: "Validation error (missing symbol name)",
			3: "Not found (no index or symbol absent)",
		},
	});

	pi.registerTool({
		name: "codemap_deadcode",
		description:
			"Report symbols classified as unused, likely-unused, or uncertain based on inbound edge counts and heuristic entrypoint detection.",
		arguments: [],
		flags: [
			{
				name: "--db",
				type: "string",
				description:
					"Path to SQLite DB (optional; default: ~/.cache/codemap/<repo-hash>.db)",
				required: false,
			},
			{
				name: "--limit",
				type: "int",
				description: "Maximum number of findings to return (default: 100)",
				required: false,
			},
		],
		output: {
			schema: "1.0",
			fields: {
				ok: "bool — true if query ran",
				"data.findings": "array — dead code findings with classification, suggestion, confidence, evidence",
				"meta.snapshot_id": "int",
				"meta.is_stale": "bool",
			},
		},
		exit_codes: {
			0: "Success",
			1: "Runtime error",
			2: "Validation error",
			3: "No index found (run 'codemap index' first)",
		},
	});

	pi.registerTool({
		name: "codemap_query",
		description:
			"Look up symbols by exact name or prefix match. Returns deterministic JSON with all matches.",
		arguments: [
			{
				name: "term",
				type: "string",
				description: "Symbol name or prefix to search for",
				required: true,
			},
		],
		flags: [
			{
				name: "--db",
				type: "string",
				description:
					"Path to SQLite DB (optional; default: ~/.cache/codemap/<repo-hash>.db)",
				required: false,
			},
		],
		output: {
			schema: "1.0",
			fields: {
				ok: "bool — true if query ran",
				"data.query": "string — the search term",
				"data.matches": "array — matching symbols with name, kind, file, signature",
				"data.count": "int — total matches",
				"meta.snapshot_id": "int",
				"meta.is_stale": "bool",
			},
		},
		exit_codes: {
			0: "Success",
			1: "Runtime error",
			2: "Validation error (missing term)",
			3: "No index found (run 'codemap index' first)",
		},
	});

	pi.registerTool({
		name: "codemap_install",
		description:
			"Install or update the codemap skill and extension into the Pi runtime. Safe to re-run.",
		arguments: [],
		flags: [
			{
				name: "--dry-run",
				type: "bool",
				description: "Check and report actions without applying",
				required: false,
			},
			{
				name: "--json",
				type: "bool",
				description: "Output machine-readable JSON",
				required: false,
			},
			{
				name: "--tui",
				type: "bool",
				description: "Run interactive TUI installer",
				required: false,
			},
		],
		output: {
			schema: "1.0",
			fields: {
				status: "string — applied|up-to-date|dry-run|error",
				checks: "array — installation checks with level and message",
				db_path: "string — default DB path",
				db_exists: "bool",
			},
		},
		exit_codes: {
			0: "Success (applied, up-to-date, or dry-run)",
			1: "Error",
			2: "Flag/validation error",
		},
	});

	pi.registerTool({
		name: "codemap_doctor",
		description:
			"Diagnose codemap environment and Pi integration status. Returns pass/warn with detailed check list.",
		arguments: [],
		flags: [
			{
				name: "--json",
				type: "bool",
				description: "Output machine-readable JSON",
				required: false,
			},
		],
		output: {
			schema: "1.0",
			fields: {
				status: "string — pass|warn",
				checks: "array — diagnostic checks with check, level, and message",
				db_path: "string — default DB path",
				db_exists: "bool",
			},
		},
		exit_codes: {
			0: "Pass or warn",
			1: "Fail or runtime error",
		},
	});
}
