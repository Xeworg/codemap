package indexer

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashContent computes a SHA-256 hash of the given content.
// Returns a hex-encoded string of the hash.
func HashContent(content []byte) string {
	h := sha256.Sum256(content)
	return hex.EncodeToString(h[:])
}
