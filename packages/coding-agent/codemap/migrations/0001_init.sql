CREATE TABLE IF NOT EXISTS schema_migrations (
  version TEXT PRIMARY KEY,
  applied_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS snapshots (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  repo_root TEXT NOT NULL,
  head_ref TEXT NOT NULL,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS files (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  path TEXT NOT NULL,
  language TEXT,
  hash TEXT,
  snapshot_id INTEGER NOT NULL,
  FOREIGN KEY(snapshot_id) REFERENCES snapshots(id)
);

CREATE TABLE IF NOT EXISTS symbols (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  file_id INTEGER NOT NULL,
  name TEXT NOT NULL,
  kind TEXT NOT NULL,
  signature TEXT,
  start_line INTEGER NOT NULL,
  end_line INTEGER NOT NULL,
  FOREIGN KEY(file_id) REFERENCES files(id)
);

CREATE TABLE IF NOT EXISTS edges (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  from_symbol_id INTEGER NOT NULL,
  to_symbol_id INTEGER NOT NULL,
  edge_type TEXT NOT NULL,
  FOREIGN KEY(from_symbol_id) REFERENCES symbols(id),
  FOREIGN KEY(to_symbol_id) REFERENCES symbols(id)
);

CREATE TABLE IF NOT EXISTS commits (
  hash TEXT PRIMARY KEY,
  author TEXT,
  date TEXT,
  message TEXT
);

CREATE TABLE IF NOT EXISTS symbol_commits (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  symbol_id INTEGER NOT NULL,
  commit_hash TEXT NOT NULL,
  change_type TEXT,
  FOREIGN KEY(symbol_id) REFERENCES symbols(id),
  FOREIGN KEY(commit_hash) REFERENCES commits(hash)
);

CREATE TABLE IF NOT EXISTS intent_notes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  target_type TEXT NOT NULL,
  target_id TEXT NOT NULL,
  source_type TEXT NOT NULL,
  note TEXT NOT NULL,
  confidence TEXT NOT NULL,
  evidence_ref TEXT,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS parse_errors (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  file TEXT NOT NULL,
  parser TEXT NOT NULL,
  error TEXT NOT NULL,
  snapshot_id INTEGER,
  created_at TEXT NOT NULL,
  FOREIGN KEY(snapshot_id) REFERENCES snapshots(id)
);
