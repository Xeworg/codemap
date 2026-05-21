-- Migration 0003: add indexing summary fields to snapshots table.
-- These columns are needed by the index command's JSON envelope.
ALTER TABLE snapshots ADD COLUMN files_scanned INTEGER DEFAULT 0;
ALTER TABLE snapshots ADD COLUMN files_parsed INTEGER DEFAULT 0;
ALTER TABLE snapshots ADD COLUMN symbols_found INTEGER DEFAULT 0;
ALTER TABLE snapshots ADD COLUMN parse_errors INTEGER DEFAULT 0;
