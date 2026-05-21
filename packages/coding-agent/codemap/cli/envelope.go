package cli

import (
	"encoding/json"
	"io"
	"time"
)

// Envelope is the stable v1.0 JSON response wrapper for all CLI commands.
type Envelope struct {
	SchemaVersion string      `json:"schema_version"`
	Command       string      `json:"command"`
	OK            bool        `json:"ok"`
	Data          interface{} `json:"data"`
	Errors        []string    `json:"errors"`
	Meta          Meta        `json:"meta"`
}

// Meta contains snapshot and freshness metadata present in every response.
type Meta struct {
	SnapshotID int64  `json:"snapshot_id"`
	HeadRef    string `json:"head_ref"`
	IndexedAt  string `json:"indexed_at"`
	IsStale    bool   `json:"is_stale"`
}

// SymbolData is the data payload for a symbol query.
type SymbolData struct {
	Name       string          `json:"name"`
	Kind       string          `json:"kind"`
	Signature  string          `json:"signature"`
	StartLine  int             `json:"start_line,omitempty"`
	EndLine    int             `json:"end_line,omitempty"`
	File       string          `json:"file,omitempty"`
	Confidence string          `json:"confidence"`
	Evidence   []EvidenceEntry `json:"evidence"`
}

// HistoryData is the data payload for a history query.
type HistoryData struct {
	SymbolName string          `json:"symbol_name"`
	Confidence string          `json:"confidence"`
	Evidence   []EvidenceEntry `json:"evidence"`
}

// EvidenceEntry represents a single evidence item in symbol or history responses.
type EvidenceEntry struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Source      string `json:"source,omitempty"`
}

// IndexData is the data payload for an index run result.
type IndexData struct {
	SnapshotID   int64           `json:"snapshot_id,omitempty"`
	FilesScanned int             `json:"files_scanned"`
	FilesParsed  int             `json:"files_parsed"`
	SymbolsFound int             `json:"symbols_found"`
	ParseErrors  int             `json:"parse_errors"`
	Evidence     []EvidenceEntry `json:"evidence"`
}

// NewEnvelope builds a response envelope with the required v1.0 structure.
func NewEnvelope(cmd string, ok bool, data interface{}, errs []string, meta Meta) *Envelope {
	if errs == nil {
		errs = []string{}
	}
	return &Envelope{
		SchemaVersion: "1.0",
		Command:       cmd,
		OK:            ok,
		Data:          data,
		Errors:        errs,
		Meta:          meta,
	}
}

// WriteErrorEnvelope writes a deterministic JSON error response.
func WriteErrorEnvelope(w io.Writer, cmd string, msg string, meta Meta) {
	env := NewEnvelope(cmd, false, struct{}{}, []string{msg}, meta)
	out, _ := env.Encode()
	_, _ = w.Write(out)
}

// EmptyMeta returns a Meta with zero values.
func EmptyMeta() Meta {
	return Meta{}
}

// Encode writes the envelope as JSON to the encoder, using a deterministic field order.
func (e *Envelope) Encode() ([]byte, error) {
	// deterministic: marshal with sorted keys is not guaranteed,
	// but we use a struct with fixed field order which is stable in Go.
	return json.Marshal(e)
}

// EncodeIndent is like Encode but pretty-prints.
func (e *Envelope) EncodeIndent() ([]byte, error) {
	return json.MarshalIndent(e, "", "  ")
}

// StaleNow marks a snapshot as stale relative to current time.
func StaleNow(indexedAt string) bool {
	if indexedAt == "" {
		return true
	}
	t, err := time.Parse(time.RFC3339, indexedAt)
	if err != nil {
		return true
	}
	// Stale if older than 24 hours.
	return time.Since(t) > 24*time.Hour
}

// DefaultEvidence returns a minimal evidence item for symbol results.
func DefaultEvidence() []EvidenceEntry {
	return []EvidenceEntry{
		{Type: "direct", Description: "symbol extracted from source"},
	}
}

// ConfidenceForSymbol returns a default confidence based on symbol kind.
func ConfidenceForSymbol(kind string) string {
	switch kind {
	case "func", "type", "interface":
		return "high"
	case "var", "const":
		return "medium"
	default:
		return "low"
	}
}
