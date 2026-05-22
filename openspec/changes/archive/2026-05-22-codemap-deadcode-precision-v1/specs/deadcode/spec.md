# Delta for deadcode

## ADDED Requirements

### Requirement: Indexer edge population for deadcode evidence

The system MUST populate symbol reference/call edges during indexing so deadcode analysis can use non-zero inbound relationship evidence when such relationships exist.

#### Scenario: Indexing persists inbound-capable edges

- GIVEN a Go repository where one symbol references or calls another symbol
- WHEN indexing completes
- THEN the index contains persisted edges representing that relationship
- AND deadcode analysis can read those edges as inbound evidence for the referenced symbol.

### Requirement: Method and init symbol coverage for deadcode

The system MUST include method symbols and `init` functions in index-time symbol extraction and edge analysis coverage used by deadcode.

#### Scenario: Methods are included in deadcode evidence graph

- GIVEN a type method that is invoked from another symbol
- WHEN indexing and deadcode analysis run
- THEN the method is present as a symbol candidate
- AND inbound edges to that method are available to deadcode classification.

#### Scenario: init participation reduces false positives

- GIVEN package-level `init` functions and symbols used by initialization flow
- WHEN indexing and deadcode analysis run
- THEN `init` functions are represented in symbol coverage
- AND deadcode confidence for initialization-linked symbols reflects that implicit runtime usage.

### Requirement: Deadcode confidence reflects explicit and implicit usage evidence

The system MUST raise confidence for symbols with explicit inbound edges and MUST downgrade or classify as `uncertain` when only implicit runtime/public-entry usage heuristics apply.

#### Scenario: Explicit inbound evidence improves confidence

- GIVEN a symbol with one or more inbound edges from indexed references/calls
- WHEN deadcode findings are produced
- THEN the symbol is NOT classified as `unused`
- AND confidence reflects explicit usage evidence.

#### Scenario: Runtime/public entry heuristics reduce false positives

- GIVEN a symbol that matches runtime or public-entry heuristics (for example `main`, `init`, exported API, or conventional `cmd/` entrypoints) without explicit inbound edges
- WHEN deadcode findings are produced
- THEN the symbol is classified as `uncertain` or non-removal guidance equivalent
- AND confidence indicates implicit-usage uncertainty instead of high-confidence unused status.
