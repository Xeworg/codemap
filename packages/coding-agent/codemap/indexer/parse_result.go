package indexer

// Symbol represents a Go top-level declaration extracted from a source file.
type Symbol struct {
	Name      string
	Kind      string // "func", "type", "var", "const"
	Signature string
	StartLine int
	EndLine   int
}

// ParseResult holds the outcome of parsing a single Go source file.
type ParseResult struct {
	Symbols []Symbol
}
