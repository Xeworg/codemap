-- Migration 0002: add link_strength to symbol_commits
-- This column records how directly a commit touches a symbol's line range.

-- Add link_strength column with CHECK constraint enforcing allowed values.
-- SQLite does not enforce CHECK on existing rows during ALTER TABLE,
-- so we use a trigger to validate on INSERT/UPDATE.
ALTER TABLE symbol_commits ADD COLUMN link_strength TEXT NOT NULL DEFAULT 'weak';

-- Trigger to reject non-enum values at INSERT time.
CREATE TRIGGER IF NOT EXISTS enforce_link_strength_enum
BEFORE INSERT ON symbol_commits
FOR EACH ROW
WHEN NEW.link_strength NOT IN ('strong', 'medium', 'weak')
BEGIN
  SELECT RAISE(ABORT, 'link_strength must be one of: strong, medium, weak');
END;

-- Same enforcement for UPDATE.
CREATE TRIGGER IF NOT EXISTS enforce_link_strength_enum_update
BEFORE UPDATE ON symbol_commits
FOR EACH ROW
WHEN NEW.link_strength NOT IN ('strong', 'medium', 'weak')
BEGIN
  SELECT RAISE(ABORT, 'link_strength must be one of: strong, medium, weak');
END;
