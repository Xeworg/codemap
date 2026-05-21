package indexer

// FileState represents the classification of a file relative to a snapshot.
type FileState int

const (
	FileStateUnchanged FileState = iota
	FileStateChanged
	FileStateNew
	FileStateDeleted
)
