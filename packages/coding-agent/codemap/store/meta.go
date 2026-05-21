package store

import (
	"context"
	"database/sql"
	"errors"
)

type SnapshotMeta struct {
	SnapshotID int64
	HeadRef    string
	IndexedAt  string
}

func GetLatestSnapshotMeta(ctx context.Context, db *sql.DB) (SnapshotMeta, error) {
	var m SnapshotMeta
	err := db.QueryRowContext(ctx, `
		SELECT id, head_ref, created_at
		FROM snapshots
		ORDER BY id DESC
		LIMIT 1`).Scan(&m.SnapshotID, &m.HeadRef, &m.IndexedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return SnapshotMeta{}, nil
	}
	return m, err
}
