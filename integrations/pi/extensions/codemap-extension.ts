/**
 * CodeMap Pi Extension — registers codemap tools with Pi's ExtensionAPI.
 *
 * This file replaces the legacy integrations/pi/tools/codemap-tool.json artifact.
 * It registers three tools: index, symbol, and history.
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
}
