-- 0004_graph_cache.sql
-- Materialized impact cache for multi-hop blast-radius queries.
-- Provides fast lookups for repeated impact analysis without re-running CTE.

CREATE TABLE IF NOT EXISTS symbol_impact_cache (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_symbol_id INTEGER NOT NULL,
    target_symbol_id INTEGER NOT NULL,
    depth INTEGER NOT NULL CHECK(depth BETWEEN 1 AND 10),
    edge_path TEXT,
    risk_score REAL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY(source_symbol_id) REFERENCES symbols(id),
    FOREIGN KEY(target_symbol_id) REFERENCES symbols(id)
);

CREATE INDEX IF NOT EXISTS idx_sic_source_depth ON symbol_impact_cache(source_symbol_id, depth);
CREATE INDEX IF NOT EXISTS idx_sic_target ON symbol_impact_cache(target_symbol_id);
CREATE INDEX IF NOT EXISTS idx_sic_updated ON symbol_impact_cache(updated_at);

CREATE TABLE IF NOT EXISTS file_impact_cache (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    from_file_id INTEGER NOT NULL,
    to_file_id INTEGER NOT NULL,
    depth INTEGER NOT NULL CHECK(depth BETWEEN 1 AND 10),
    edge_count INTEGER NOT NULL,
    risk_score REAL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY(from_file_id) REFERENCES files(id),
    FOREIGN KEY(to_file_id) REFERENCES files(id)
);

CREATE INDEX IF NOT EXISTS idx_fic_from_depth ON file_impact_cache(from_file_id, depth);
CREATE INDEX IF NOT EXISTS idx_fic_updated ON file_impact_cache(updated_at);